package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cs2-demo-service/models"
	parserpkg "cs2-demo-service/parser"
)

func TestRunRejectsMissingArgumentsWithoutLeakingPaths(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, &stdout, &stderr, dependencies{})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stdout.String(), `"error_code":"invalid_arguments"`) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunRejectsChecksumMismatchBeforeParsing(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	called := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			called = true
			return nil, errors.New("must not be called")
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", filepath.Join(t.TempDir(), "out"),
		"--alias", "demo-" + checksum[:20],
		"--checksum", strings.Repeat("0", 64),
	}, &stdout, &stderr, deps)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3", code)
	}
	if called {
		t.Fatal("parser was called after checksum mismatch")
	}
	if strings.Contains(stdout.String(), demoPath) {
		t.Fatal("structured output leaked source path")
	}
}

func TestRunPublishesOnlyAfterSuccessfulExport(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := filepath.Join(t.TempDir(), "out")
	context := &models.DemoContext{}
	deps := dependencies{
		parse: func(path string) (*parserpkg.ParseDemoResult, error) {
			if path != demoPath {
				t.Fatalf("parse path = %q, want fixture", path)
			}
			return &parserpkg.ParseDemoResult{Context: context}, nil
		},
		export: func(
			actualContext *models.DemoContext,
			alias, stagingRoot, playedAt string,
			provenance models.CanonicalExportProvenance,
		) error {
			if actualContext != context || alias != "demo-"+checksum[:20] {
				t.Fatal("export received unexpected context or alias")
			}
			if playedAt != "" || provenance.DemoChecksum != checksum ||
				provenance.Source != "demo" || provenance.BuildIdentifier != buildID {
				t.Fatalf("unexpected provenance: %+v playedAt=%q", provenance, playedAt)
			}
			bundle := filepath.Join(stagingRoot, "match_"+alias)
			if err := os.MkdirAll(bundle, 0o750); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bundle, "marker"), []byte("ok"), 0o600)
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
	}, &stdout, &stderr, deps)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	finalBundle := filepath.Join(outputRoot, "match_demo-"+checksum[:20])
	if _, err := os.Stat(filepath.Join(finalBundle, "marker")); err != nil {
		t.Fatalf("published marker: %v", err)
	}
	if strings.Contains(stdout.String(), demoPath) || strings.Contains(stdout.String(), outputRoot) {
		t.Fatal("structured output leaked a private path")
	}
}

func TestRunDoesNotPublishFailedExport(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := filepath.Join(t.TempDir(), "out")
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			return &parserpkg.ParseDemoResult{Context: &models.DemoContext{}}, nil
		},
		export: func(
			*models.DemoContext, string, string, string, models.CanonicalExportProvenance,
		) error {
			return errors.New("quality gate failed")
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
	}, &stdout, &stderr, deps)
	if code != 5 {
		t.Fatalf("exit code = %d, want 5", code)
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "match_demo-"+checksum[:20])); !os.IsNotExist(err) {
		t.Fatalf("failed export was published: %v", err)
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("staging was not cleaned: %v", entries)
	}
}

func TestRunRefusesToOverwriteBundle(t *testing.T) {
	demoPath, checksum := writeDemoFixture(t)
	outputRoot := t.TempDir()
	finalBundle := filepath.Join(outputRoot, "match_demo-"+checksum[:20])
	if err := os.Mkdir(finalBundle, 0o750); err != nil {
		t.Fatal(err)
	}
	called := false
	deps := dependencies{
		parse: func(string) (*parserpkg.ParseDemoResult, error) {
			called = true
			return nil, nil
		},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{
		"--demo", demoPath,
		"--output", outputRoot,
		"--alias", "demo-" + checksum[:20],
		"--checksum", checksum,
	}, &stdout, &stderr, deps)
	if code != 6 {
		t.Fatalf("exit code = %d, want 6", code)
	}
	if called {
		t.Fatal("parser was called before overwrite protection")
	}
}

func writeDemoFixture(t *testing.T) (string, string) {
	t.Helper()
	payload := []byte("PBDEMS2\x00fixture")
	digest := sha256.Sum256(payload)
	checksum := hex.EncodeToString(digest[:])
	path := filepath.Join(t.TempDir(), "fixture.dem")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, checksum
}
