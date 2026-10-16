package main

import (
	"bytes"
	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleContextSelectorUsesChangedProducerOnce(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	root := emptySmokeOutputRoot(t)
	calls := 0
	var output, diagnostics bytes.Buffer
	code := run([]string{"--demo", demo, "--output", root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--ai-coach-economy-candidate", "--economy-candidate-version", "4"}, &output, &diagnostics, dependencies{
		parseEconomyRules: func(string) (*parserpkg.ParseDemoResult, error) {
			calls++
			return nil, errors.New("fixture stops after selector")
		},
	})
	if calls != 1 || code != 4 {
		t.Fatalf("wrong producer path: calls=%d code=%d %s", calls, code, diagnostics.String())
	}
}

func TestRuleContextWriterBindsNativeProducerAndProbe(t *testing.T) {
	root := t.TempDir()
	ctx := &models.DemoContext{EconomyRuleContext: &models.EconomyRuleContext{Messages: []models.EconomyConvarMessage{}, Boundaries: []models.EconomyRuleBoundary{}, Sessions: []models.EconomySessionProbe{}}}
	opts := options{sourceAlias: "demo-aaaaaaaaaaaaaaaaaaaa", checksum: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", buildIDValue: "test-build"}
	if writeEconomyRuleContext(ctx, opts, root) == nil {
		t.Fatal("incomplete parse accepted")
	}
	ctx.ParseCompleted = true
	if writeEconomyRuleContext(ctx, opts, root) == nil {
		t.Fatal("missing native source accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "economy_native_sources_manifest.json"), []byte(`{"fixture":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeEconomyRuleContext(ctx, opts, root); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "economy_rule_context_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Source     economyArtifactReference `json:"source_native"`
		Artifact   economyArtifactReference `json:"artifact"`
		Probe      string                   `json:"probe_sha256"`
		Comparison bool                     `json:"comparison_allowed"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	source, _ := economyFileReference(root, manifest.Source.Path)
	artifact, _ := economyFileReference(root, manifest.Artifact.Path)
	_, probe := models.LoadEconomyRuleProbe()
	if manifest.Source != source || manifest.Artifact != artifact || manifest.Probe != probe || manifest.Comparison {
		t.Fatal("native rule context binding differs")
	}
}
