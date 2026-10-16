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

func TestSupportC4ModeProtectsDestinationBeforeParse(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	for _, mode := range []string{"valid", "nonempty", "non-temporary", "support", "physical", "economy", "spatial", "objective"} {
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
			args := []string{"--demo", demo, "--output", root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--ai-coach-support-c4-candidate"}
			switch mode {
			case "objective":
				args = append(args, "--ai-coach-support-objective-candidate")
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
			code := run(args, &out, &diagnostics, dependencies{parseSupportC4: func(string) (*parserpkg.ParseDemoResult, error) { calls++; return nil, errors.New("stop at parser") }})
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

func TestSupportC4WriterBindsBothSourcesAndOpaqueActors(t *testing.T) {
	root := t.TempDir()
	opts := options{sourceAlias: "demo-" + strings.Repeat("a", 20), checksum: strings.Repeat("a", 64), buildIDValue: "fixture"}
	alias := "support-actor-" + strings.Repeat("b", 20)
	ctx := &models.DemoContext{ParseCompleted: true,
		SupportObservations:          &models.SupportObservations{Frames: []models.SupportFrame{{Sequence: 1, Frame: 100, Players: []models.SupportPlayer{{NativeActor: 76561198012345678, ActorAlias: alias}}}}},
		SupportObjectiveObservations: &models.SupportObjectiveObservations{Frames: []models.SupportObjectiveFrame{{SourceSequence: 1, Order: 7}}},
		SupportC4Observations:        &models.SupportC4Observations{Cuts: []models.SupportC4Cut{{SourceSequence: 1, Frame: 100, ObjectiveOrder: 7, StateSequence: 1}}, States: []models.SupportC4State{{Sequence: 1, FirstCut: 1, Players: []models.SupportC4Player{{NativeActor: 76561198012345678}}}}},
	}
	if writeSupportC4(ctx, opts, root) == nil {
		t.Fatal("missing previous sources accepted")
	}
	for _, name := range []string{"support_observations.json", "support_objective_observations.json"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeSupportC4(ctx, opts, root); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "support_c4_observations.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("76561198012345678")) || !bytes.Contains(raw, []byte(alias)) {
		t.Fatal("actor identity not opaque")
	}
	var data struct {
		Support   economyArtifactReference `json:"support_source"`
		Objective economyArtifactReference `json:"objective_source"`
		Training  bool                     `json:"training_allowed"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	support, _ := economyFileReference(root, "support_observations.json")
	objective, _ := economyFileReference(root, "support_objective_observations.json")
	if data.Support != support || data.Objective != objective || data.Training {
		t.Fatal("source or gate mismatch")
	}
	ctx.SupportC4Observations.Cuts[0].ObjectiveOrder++
	if writeSupportC4(ctx, opts, root) == nil {
		t.Fatal("cut mismatch accepted")
	}
}
