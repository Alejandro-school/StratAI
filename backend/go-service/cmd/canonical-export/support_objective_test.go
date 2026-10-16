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

func TestSupportObjectiveModeProtectsDestinationBeforeParse(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	for _, mode := range []string{"valid", "nonempty", "non-temporary", "support", "physical", "economy", "spatial"} {
		t.Run(mode, func(t *testing.T) {
			root := emptySmokeOutputRoot(t)
			if mode == "nonempty" {
				if err := os.WriteFile(filepath.Join(root, "sentinel"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "non-temporary" {
				root = "."
			}
			args := []string{"--demo", demo, "--output", root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--ai-coach-support-objective-candidate"}
			switch mode {
			case "physical":
				args = append(args, "--ai-coach-support-physical-candidate")
			case "support":
				args = append(args, "--ai-coach-support-candidate")
			case "economy":
				args = append(args, "--ai-coach-economy-candidate")
			case "spatial":
				args = append(args, "--ai-coach-spatial-candidate")
			}
			calls := 0
			var out, diagnostics bytes.Buffer
			code := run(args, &out, &diagnostics, dependencies{parseSupportObjective: func(string) (*parserpkg.ParseDemoResult, error) { calls++; return nil, errors.New("stop at parser") }})
			if mode == "valid" {
				if calls != 1 || code != 4 {
					t.Fatal("wrong mode", code, calls, diagnostics.String())
				}
			} else if calls != 0 || code == 0 {
				t.Fatal("precheck bypassed")
			}
		})
	}
}

func TestSupportObjectiveWriterBindsV1AndOmitsNativeIdentity(t *testing.T) {
	root := t.TempDir()
	opts := options{sourceAlias: "demo-" + strings.Repeat("a", 20), checksum: strings.Repeat("a", 64), buildIDValue: "fixture"}
	ctx := &models.DemoContext{ParseCompleted: true, SupportObservations: &models.SupportObservations{Frames: []models.SupportFrame{{Sequence: 1, Players: []models.SupportPlayer{{NativeActor: 76561198012345678, ActorAlias: "support-actor-" + strings.Repeat("b", 20)}}}}}, SupportObjectiveObservations: &models.SupportObjectiveObservations{Frames: []models.SupportObjectiveFrame{{SourceSequence: 1, Players: []models.SupportObjectivePlayer{{NativeActor: 76561198012345678}}}}}}
	if writeSupportObjective(ctx, opts, root) == nil {
		t.Fatal("missing v1 source accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "support_observations.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeSupportObjective(ctx, opts, root); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "support_objective_observations.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("76561198012345678")) {
		t.Fatal("native identity exposed")
	}
	var data struct {
		Source   economyArtifactReference `json:"support_source"`
		Training bool                     `json:"training_allowed"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	expected, _ := economyFileReference(root, "support_observations.json")
	if data.Source != expected || data.Training {
		t.Fatal("wrong reference/gate")
	}
}
