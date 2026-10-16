package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	parserpkg "cs2-demo-service/parser"
)

func TestSpatialCandidateModeEnablesThreeProducersAndRejectsOutputsBeforeParse(t *testing.T) {
	demo, checksum := writeDemoFixture(t)
	base := []string{"--demo", demo, "--alias", "demo-" + checksum[:20], "--checksum", checksum,
		"--maps-root", t.TempDir(), "--exposure-packages-root", "exposure", "--cover-packages-root", "cover",
		"--exit-route-packages-root", "exit", "--ai-coach-spatial-candidate"}
	empty := emptySmokeOutputRoot(t)
	opts, err := parseOptions(append(base, "--output", empty), io.Discard)
	if err != nil || !opts.aiCoachExposureValidation || !opts.aiCoachCoverValidation || !opts.aiCoachExitRouteValidation {
		t.Fatalf("combined mode did not enable spatial producers: %v", err)
	}
	if _, err := parseOptions(append(append([]string{}, base...), "--output", empty, "--ai-coach-smoke"), io.Discard); err == nil {
		t.Fatal("conflicting smoke mode accepted")
	}
	nonempty := emptySmokeOutputRoot(t)
	if err := os.WriteFile(filepath.Join(nonempty, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{nonempty, os.TempDir(), filepath.VolumeName(os.TempDir()) + string(filepath.Separator)} {
		var stdout, stderr bytes.Buffer
		deps := dependencies{parse: func(string) (*parserpkg.ParseDemoResult, error) {
			t.Fatal("unsafe output reached the parser")
			return nil, nil
		}}
		if code := run(append(append([]string{}, base...), "--output", output), &stdout, &stderr, deps); code != 6 {
			t.Fatalf("unsafe output returned %d: %s", code, stderr.String())
		}
	}
	data, _ := os.ReadFile(filepath.Join(nonempty, "keep.txt"))
	if string(data) != "keep" {
		t.Fatal("preflight overwrote existing data")
	}
}
