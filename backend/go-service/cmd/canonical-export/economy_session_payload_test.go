package main

import (
	"bytes"
	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionPayloadWriterBindsContextAndVersion(t *testing.T) {
	root := t.TempDir()
	opts := options{sourceAlias: "demo-aaaaaaaaaaaaaaaaaaaa", checksum: strings.Repeat("a", 64), buildIDValue: "fixture"}
	ctx := &models.DemoContext{EconomySessionPayloads: &models.EconomySessionPayloads{Sessions: []models.EconomySessionPayload{}}}
	if writeEconomySessionPayloads(ctx, opts, root) == nil {
		t.Fatal("incomplete parse accepted")
	}
	ctx.ParseCompleted = true
	if writeEconomySessionPayloads(ctx, opts, root) == nil {
		t.Fatal("missing context accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "economy_rule_context_manifest.json"), []byte(`{"fixture":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeEconomySessionPayloads(ctx, opts, root); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "economy_session_payload.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Source  economyArtifactReference `json:"source_context"`
		Probe   string                   `json:"probe_sha256"`
		ID      string                   `json:"schema_id"`
		Binding string                   `json:"exact_build_asset_binding"`
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	source, _ := economyFileReference(root, "economy_rule_context_manifest.json")
	_, probe := models.LoadEconomySessionPayloadProbe()
	if data.Source != source || data.Probe != probe || data.ID != "stratai.economy_session_payload@1" || data.Binding != "not_accredited" {
		t.Fatal("provenance differs")
	}
}

func TestSessionPayloadSelectorAndPrechecks(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	for _, mode := range []string{"valid", "nonempty", "not-opted-in", "non-temporary"} {
		t.Run(mode, func(t *testing.T) {
			root := emptySmokeOutputRoot(t)
			calls := 0
			var output, diagnostics bytes.Buffer
			args := []string{"--demo", demo, "--output", root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--economy-candidate-version", "5"}
			if mode != "not-opted-in" {
				args = append(args, "--ai-coach-economy-candidate")
			}
			if mode == "nonempty" {
				if err := os.WriteFile(filepath.Join(root, "sentinel"), []byte("preserve"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "non-temporary" {
				args[3] = "."
			}
			code := run(args, &output, &diagnostics, dependencies{parseEconomySessions: func(string) (*parserpkg.ParseDemoResult, error) {
				calls++
				return nil, errors.New("fixture stops at parser")
			}})
			if mode == "valid" {
				if code != 4 || calls != 1 {
					t.Fatalf("selector: %d %d %s", code, calls, diagnostics.String())
				}
			} else if calls != 0 || code == 0 {
				t.Fatal("precheck bypassed")
			}
		})
	}
}
