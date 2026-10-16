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

func TestTransactionTraceWriterBindsContext(t *testing.T) {
	root := t.TempDir()
	ctx := &models.DemoContext{EconomyTransactionTrace: &models.EconomyTransactionTrace{Rows: []models.EconomyTraceRow{}}}
	opts := options{sourceAlias: "demo-aaaaaaaaaaaaaaaaaaaa", checksum: strings.Repeat("a", 64), buildIDValue: "fixture"}
	if writeEconomyTrace(ctx, opts, root) == nil {
		t.Fatal("incomplete parse accepted")
	}
	ctx.ParseCompleted = true
	if writeEconomyTrace(ctx, opts, root) == nil {
		t.Fatal("missing provenance accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "economy_candidate_manifest.json"), []byte(`{"fixture":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeEconomyTrace(ctx, opts, root); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "economy_transaction_trace.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Source   economyArtifactReference `json:"source_candidate"`
		ID       string                   `json:"schema_id"`
		Training bool                     `json:"training_allowed"`
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	source, _ := economyFileReference(root, "economy_candidate_manifest.json")
	if data.Source != source || data.ID != "stratai.economy_transaction_trace@1" || data.Training {
		t.Fatal("provenance differs")
	}
}

func TestTransactionTraceSelectorAndPrechecks(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	for _, mode := range []string{"valid", "nonempty", "not-opted-in", "non-temporary"} {
		t.Run(mode, func(t *testing.T) {
			root := emptySmokeOutputRoot(t)
			calls := 0
			var output, diagnostics bytes.Buffer
			args := []string{"--demo", demo, "--output", root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--economy-candidate-version", "7"}
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
			code := run(args, &output, &diagnostics, dependencies{parseEconomyTrace: func(string) (*parserpkg.ParseDemoResult, error) {
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
