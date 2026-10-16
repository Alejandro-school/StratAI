package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSoundAuditRejectsDestinationBeforeReadingDemo(t *testing.T) {
	for _, output := range []string{".", filepath.Join(t.TempDir(), "missing")} {
		if err := run("demo-must-not-be-read", output, strings.Repeat("0", 64)); err == nil || strings.Contains(err.Error(), "demo-must-not-be-read") {
			t.Fatalf("destination guard did not precede source read: %v", err)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "retained.txt"), []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run("demo-must-not-be-read", dir, strings.Repeat("0", 64)); err == nil || strings.Contains(err.Error(), "demo-must-not-be-read") {
		t.Fatalf("nonempty output reached the source: %v", err)
	}
}

func TestSoundAuditRejectsWrongChecksumWithoutParsing(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(t.TempDir(), "fake.dem")
	if err := os.WriteFile(input, []byte("not a demo"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(input, dir, strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum rejection, got %v", err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatal("failed preflight wrote an artifact")
	}
}
