package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
)

func TestEconomyModeRejectsUnsafeDestinationsAndMixedModesBeforeParsing(t *testing.T) {
	for _, version := range []string{"1", "2", "3", "4"} {
		t.Run("version_"+version, func(t *testing.T) {
			demo, checksum := writeDemoFixture(t)
			dirty := emptySmokeOutputRoot(t)
			if err := os.WriteFile(filepath.Join(dirty, "keep"), []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
			outside, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range []struct{ root, extra string }{{dirty, ""}, {outside, ""}, {os.TempDir(), ""}, {emptySmokeOutputRoot(t), "--ai-coach-smoke"}, {emptySmokeOutputRoot(t), "--ai-coach-spatial-candidate"}} {
				called := false
				parse := func(string) (*parserpkg.ParseDemoResult, error) {
					called = true
					return nil, errors.New("must not parse")
				}
				args := []string{"--demo", demo, "--output", item.root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--ai-coach-economy-candidate"}
				args = append(args, "--economy-candidate-version", version)
				if item.extra != "" {
					args = append(args, item.extra)
				}
				var out, diagnostic bytes.Buffer
				code := run(args, &out, &diagnostic, dependencies{parse: parse, parseEconomy: parse, parseEconomyV2: parse, parseEconomySources: parse, parseEconomyRules: parse})
				if code == 0 || called {
					t.Fatalf("unsafe mode invoked parser: %d %s", code, out.String())
				}
			}
			content, _ := os.ReadFile(filepath.Join(dirty, "keep"))
			if string(content) != "preserve" {
				t.Fatal("existing output changed")
			}
		})
	}
}

func TestEconomyModeParsesOnceAndKeepsCanonicalExport(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	root := emptySmokeOutputRoot(t)
	ctx := &models.DemoContext{MapName: "de_mirage", ReplayData: &models.ReplayData{Metadata: models.ReplayMetadata{TickRate: 64}}, EconomyObservations: &models.EconomyObservationLedger{
		ActorAliases: map[uint64]string{123: "p0001"}, Rounds: []models.EconomyObservedRound{{Instance: 1, RoundNumber: 1, StartTick: 100, Snapshots: []models.EconomySnapshot{{Phase: "initial_available", Tick: 100, Actors: []models.EconomyActorSnapshot{{ActorAlias: "p0001"}}}}}},
	}}
	calls := 0
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) { t.Fatal("normal parser selected"); return nil, nil },
		parseEconomy: func(string) (*parserpkg.ParseDemoResult, error) {
			calls++
			return &parserpkg.ParseDemoResult{Context: ctx}, nil
		},
		export: func(actual *models.DemoContext, alias, staging, playedAt string, provenance models.CanonicalExportProvenance) error {
			if actual != ctx || provenance.DemoChecksum != checksum {
				t.Fatal("canonical context/provenance changed")
			}
			path := filepath.Join(staging, "match_"+alias, "canonical")
			if err := os.MkdirAll(path, 0750); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(path, "manifest.json"), []byte(`{"fixture":true}`), 0600)
		},
	}
	var out, diagnostic bytes.Buffer
	code := run([]string{"--demo", demo, "--output", root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--ai-coach-economy-candidate"}, &out, &diagnostic, deps)
	if code != 0 || calls != 1 {
		t.Fatalf("code=%d calls=%d %s %s", code, calls, out.String(), diagnostic.String())
	}
	if !strings.Contains(out.String(), `"economy_candidate_manifest"`) {
		t.Fatal("candidate descriptor missing")
	}
	files := relativeRegularFiles(t, root)
	if len(files) != 3 {
		t.Fatalf("unexpected extra producers/files: %v", files)
	}
}
