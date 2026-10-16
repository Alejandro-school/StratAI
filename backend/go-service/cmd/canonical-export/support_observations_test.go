package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
)

func TestSupportWriterRequiresCompleteParseAndBindsOpaqueSource(t *testing.T) {
	root := t.TempDir()
	opts := options{sourceAlias: "demo-" + strings.Repeat("a", 20), checksum: strings.Repeat("a", 64), buildIDValue: "fixture"}
	bundle := "match_" + opts.sourceAlias
	ctx := &models.DemoContext{SupportObservations: &models.SupportObservations{Frames: []models.SupportFrame{{Players: []models.SupportPlayer{{NativeActor: 76561198012345678}}}}}}
	if writeSupportObservations(ctx, opts, root, bundle) == nil {
		t.Fatal("incomplete parse accepted")
	}
	ctx.ParseCompleted = true
	if writeSupportObservations(ctx, opts, root, bundle) == nil {
		t.Fatal("missing base accepted")
	}
	if err := os.MkdirAll(filepath.Join(root, bundle, "canonical"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, bundle, "canonical/manifest.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeSupportObservations(ctx, opts, root, bundle); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "support_observations.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("76561198012345678")) || bytes.Contains(raw, []byte("NativeActor")) {
		t.Fatal("native identity leaked")
	}
	var data struct {
		Base     economyArtifactReference `json:"base_manifest"`
		Training bool                     `json:"training_allowed"`
		ID       string                   `json:"schema_id"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	expected, _ := economyFileReference(root, filepath.Join(bundle, "canonical/manifest.json"))
	if data.Base != expected || data.Training || data.ID != "stratai.support_observations@1" {
		t.Fatal("provenance/gates differ")
	}
}

func TestSupportModeRejectsUnsafeDestinationsAndCombinedModesBeforeParse(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	for _, mode := range []string{"valid", "nonempty", "non-temporary", "economy", "spatial"} {
		t.Run(mode, func(t *testing.T) {
			root := emptySmokeOutputRoot(t)
			if mode == "nonempty" {
				if err := os.WriteFile(filepath.Join(root, "sentinel"), []byte("preserve"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "non-temporary" {
				root = "."
			}
			args := []string{"--demo", demo, "--output", root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--ai-coach-support-candidate"}
			if mode == "economy" {
				args = append(args, "--ai-coach-economy-candidate")
			}
			if mode == "spatial" {
				args = append(args, "--ai-coach-spatial-candidate")
			}
			calls := 0
			var output, diagnostics bytes.Buffer
			code := run(args, &output, &diagnostics, dependencies{parseSupport: func(string) (*parserpkg.ParseDemoResult, error) { calls++; return nil, errors.New("stop at parser") }})
			if mode == "valid" {
				if calls != 1 || code != 4 {
					t.Fatalf("wrong selector: %d %d %s", calls, code, diagnostics.String())
				}
			} else if calls != 0 || code == 0 {
				t.Fatal("precheck bypassed")
			}
		})
	}
}
