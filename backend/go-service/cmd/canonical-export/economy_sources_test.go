package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cs2-demo-service/models"
)

func TestNativeSourceWriterBindsV2AndRequiresCompletedProducer(t *testing.T) {
	root := t.TempDir()
	ctx := &models.DemoContext{EconomyNativeSources: &models.EconomyNativeSources{Snapshots: []models.EconomySourceSnapshot{}, VersionObservations: []models.EconomyVersionObservation{}, GenericEventCounts: map[string]int{}}}
	opts := options{sourceAlias: "demo-aaaaaaaaaaaaaaaaaaaa", checksum: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", buildIDValue: "test-build"}
	if writeEconomyNativeSources(ctx, opts, root) == nil {
		t.Fatal("incomplete parse accepted")
	}
	ctx.ParseCompleted = true
	if writeEconomyNativeSources(ctx, opts, root) == nil {
		t.Fatal("missing v2 source accepted")
	}
	input := []byte(`{"fixture":"source"}`)
	if err := os.WriteFile(filepath.Join(root, "economy_candidate_manifest.json"), input, 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeEconomyNativeSources(ctx, opts, root); err != nil {
		t.Fatal(err)
	}
	var manifest map[string]json.RawMessage
	payload, err := os.ReadFile(filepath.Join(root, "economy_native_sources_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	var source, artifact economyArtifactReference
	if err = json.Unmarshal(manifest["source_candidate"], &source); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(manifest["artifact"], &artifact); err != nil {
		t.Fatal(err)
	}
	actual, _ := economyFileReference(root, source.Path)
	if source != actual || source.Bytes != int64(len(input)) {
		t.Fatal("source hash binding differs")
	}
	actual, _ = economyFileReference(root, artifact.Path)
	if artifact != actual {
		t.Fatal("data hash binding differs")
	}
	if string(manifest["comparison_allowed"]) != "false" {
		t.Fatal("comparison unexpectedly enabled")
	}
}
