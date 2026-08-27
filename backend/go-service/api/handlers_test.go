package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func writeTestArtifact(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o640); err != nil {
		t.Fatal(err)
	}
}

func TestManifestIdentityAndAtomicCommit(t *testing.T) {
	root := t.TempDir()
	finalDir := filepath.Join(root, "match_123")
	stagedDir := filepath.Join(root, "staged", "match_123")
	if err := os.MkdirAll(stagedDir, 0o750); err != nil {
		t.Fatal(err)
	}
	artifactData := []byte("{}")
	artifactPath := filepath.Join(stagedDir, "canonical", "core", "match.json")
	writeTestArtifact(t, artifactPath, artifactData)
	artifactHash, err := hashFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest := exportManifest{
		MatchID:             "123",
		Checksum:            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ParserSchemaVersion: defaultParserSchemaVersion,
		CommittedAt:         time.Now().UTC(),
		ExportFormatVersion: exportFormatVersion,
		ValidatorVersion:    canonicalValidatorVersion,
		ValidationStatus:    "passed",
		Artifacts: []exportArtifact{{
			Path: "canonical/core/match.json", SHA256: artifactHash, Bytes: int64(len(artifactData)),
		}},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagedDir, "manifest.json"), data, 0o640); err != nil {
		t.Fatal(err)
	}

	if err := commitExport(stagedDir, finalDir); err != nil {
		t.Fatal(err)
	}
	if !hasCommittedManifest(finalDir, manifest.Checksum, manifest.ParserSchemaVersion) {
		t.Fatal("committed manifest was not recognized")
	}
	if hasCommittedManifest(finalDir, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", defaultParserSchemaVersion) {
		t.Fatal("different checksum must not be treated as idempotent")
	}
}

func TestPreCanonicalManifestDoesNotHideCanonicalMigration(t *testing.T) {
	root := t.TempDir()
	legacy := exportManifest{
		MatchID:             "123",
		Checksum:            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ParserSchemaVersion: "v8",
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	writeTestArtifact(t, filepath.Join(root, "manifest.json"), data)

	if hasCommittedManifest(root, legacy.Checksum, legacy.ParserSchemaVersion) {
		t.Fatal("legacy manifest without canonical catalog must be reprocessed")
	}
}

func TestConfiguredConcurrencyDefaultsAndRejectsInvalidValues(t *testing.T) {
	t.Setenv("GO_MAX_CONCURRENT_DEMOS", "")
	if configuredConcurrency() != 2 {
		t.Fatal("expected default concurrency of 2")
	}
	t.Setenv("GO_MAX_CONCURRENT_DEMOS", "0")
	if configuredConcurrency() != 2 {
		t.Fatal("invalid concurrency must fall back to 2")
	}
	t.Setenv("GO_MAX_CONCURRENT_DEMOS", "4")
	if configuredConcurrency() != 4 {
		t.Fatal("valid concurrency was not applied")
	}
}

func TestBuildArtifactCatalogIsDeterministicAndUsesCanonicalMetadata(t *testing.T) {
	stagedDir := filepath.Join(t.TempDir(), "match_123")
	writeTestArtifact(t, filepath.Join(stagedDir, "manifest.json"), []byte(`{"excluded":true}`))
	writeTestArtifact(t, filepath.Join(stagedDir, "canonical", "core", "match.json"), []byte(`{"schema_id":"stratai.match@1"}`))
	writeTestArtifact(t, filepath.Join(stagedDir, "canonical", "events", "combat_events.jsonl"), []byte("{\"event_id\":\"event_1\"}\n"))
	writeTestArtifact(t, filepath.Join(stagedDir, "canonical", "manifest.json"), []byte(`{
		"schema_id":"stratai.canonical_manifest@1",
		"export_format_version":"3.4.0",
		"match_id":"123",
		"artifacts":[
			{"artifact_type":"combat_events","path":"events/combat_events.jsonl","schema_id":"stratai.combat_event@1","format":"jsonl"},
			{"artifact_type":"match","path":"core/match.json","schema_id":"stratai.match@1","format":"json"}
		]
	}`))

	first, err := buildArtifactCatalog(stagedDir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildArtifactCatalog(stagedDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("artifact catalog must be deterministic")
	}

	expectedPaths := []string{
		"canonical/core/match.json",
		"canonical/events/combat_events.jsonl",
		"canonical/manifest.json",
	}
	if len(first) != len(expectedPaths) {
		t.Fatalf("expected %d artifacts, got %d", len(expectedPaths), len(first))
	}
	for index, expectedPath := range expectedPaths {
		if first[index].Path != expectedPath {
			t.Fatalf("artifact %d: expected path %q, got %q", index, expectedPath, first[index].Path)
		}
		fullPath := filepath.Join(stagedDir, filepath.FromSlash(expectedPath))
		expectedHash, err := hashFile(fullPath)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Fatal(err)
		}
		if first[index].SHA256 != expectedHash || first[index].Bytes != info.Size() {
			t.Fatalf("artifact %q has incorrect integrity metadata", expectedPath)
		}
	}

	if first[0].ArtifactType != "match" || first[0].SchemaID != "stratai.match@1" || first[0].Format != "json" || first[0].Compression != "none" {
		t.Fatalf("canonical match metadata was not propagated: %+v", first[0])
	}
	if first[1].ArtifactType != "combat_events" || first[1].SchemaID != "stratai.combat_event@1" || first[1].Format != "jsonl" || first[1].Compression != "none" {
		t.Fatalf("canonical event metadata was not propagated: %+v", first[1])
	}
	if first[2].ArtifactType != "canonical_manifest" || first[2].SchemaID != "stratai.canonical_manifest@1" {
		t.Fatalf("canonical manifest must have a known schema: %+v", first[2])
	}
}

func TestInferArtifactEncoding(t *testing.T) {
	tests := []struct {
		path        string
		format      string
		compression string
	}{
		{path: "artifact.json", format: "json", compression: "none"},
		{path: "events.jsonl.zst", format: "jsonl", compression: "zstd"},
		{path: "table.parquet.gz", format: "parquet", compression: "gzip"},
	}
	for _, test := range tests {
		format, compression := inferArtifactEncoding(test.path)
		if format != test.format || compression != test.compression {
			t.Fatalf("%s: expected %s/%s, got %s/%s", test.path, test.format, test.compression, format, compression)
		}
	}
}

func TestCanonicalManifestKeepsIdempotentIdentity(t *testing.T) {
	root := t.TempDir()
	artifactData := []byte("{}")
	artifactPath := filepath.Join(root, "canonical", "core", "match.json")
	writeTestArtifact(t, artifactPath, artifactData)
	artifactHash, err := hashFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest := exportManifest{
		MatchID:             "123",
		Checksum:            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ParserSchemaVersion: defaultParserSchemaVersion,
		CommittedAt:         time.Now().UTC(),
		ExportFormatVersion: exportFormatVersion,
		ValidatorVersion:    canonicalValidatorVersion,
		ValidationStatus:    "passed",
		Artifacts: []exportArtifact{{
			ArtifactType: "match",
			Path:         "canonical/core/match.json",
			Format:       "json",
			Compression:  "none",
			SHA256:       artifactHash,
			Bytes:        int64(len(artifactData)),
		}},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeTestArtifact(t, filepath.Join(root, "manifest.json"), data)

	if !hasCommittedManifest(root, manifest.Checksum, manifest.ParserSchemaVersion) {
		t.Fatal("canonical catalog must preserve idempotent identity checks")
	}
	var decoded exportManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.MatchID != manifest.MatchID || decoded.Checksum != manifest.Checksum || decoded.ParserSchemaVersion != manifest.ParserSchemaVersion {
		t.Fatal("manifest identity fields changed during round trip")
	}
	if decoded.ExportFormatVersion != exportFormatVersion || len(decoded.Artifacts) != 1 {
		t.Fatal("new manifest catalog fields were not preserved")
	}
}
