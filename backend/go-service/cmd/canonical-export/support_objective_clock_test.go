package main

import (
	"bytes"
	parserpkg "cs2-demo-service/parser"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSupportObjectiveClockModeProtectsDestinationBeforeParse(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	for _, mode := range []string{"valid", "nonempty", "non-temporary", "support", "physical", "economy", "spatial", "objective", "c4"} {
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
			args := []string{"--demo", demo, "--output", root, "--alias", "demo-" + checksum[:20], "--checksum", checksum, "--ai-coach-support-objective-clock-candidate"}
			switch mode {
			case "c4":
				args = append(args, "--ai-coach-support-c4-candidate")
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
			code := run(args, &out, &diagnostics, dependencies{parseSupportObjectiveClock: func(string) (*parserpkg.ParseDemoResult, error) { calls++; return nil, errors.New("stop at parser") }})
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
