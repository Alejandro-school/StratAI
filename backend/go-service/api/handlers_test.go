package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManifestIdentityAndAtomicCommit(t *testing.T) {
	root := t.TempDir()
	finalDir := filepath.Join(root, "match_123")
	stagedDir := filepath.Join(root, "staged", "match_123")
	if err := os.MkdirAll(stagedDir, 0o750); err != nil {
		t.Fatal(err)
	}
	manifest := exportManifest{
		MatchID:             "123",
		Checksum:            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ParserSchemaVersion: "v2",
		CommittedAt:         time.Now().UTC(),
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
	if hasCommittedManifest(finalDir, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "v2") {
		t.Fatal("different checksum must not be treated as idempotent")
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
