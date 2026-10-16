package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func createSecurityDirectoryLink(t *testing.T, fixtureRoot, linkPath, targetPath string) {
	t.Helper()
	for _, candidate := range []string{linkPath, targetPath} {
		relative, err := filepath.Rel(fixtureRoot, candidate)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			t.Fatalf("link fixture path must remain inside its temporary root: %q", candidate)
		}
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("link fixture destination must not already exist: %q (%v)", linkPath, err)
	}
	if err := os.Symlink(targetPath, linkPath); err != nil {
		if runtime.GOOS != "windows" {
			t.Fatalf("cannot exercise the required symlink boundary: %v", err)
		}
		// Windows may require elevation for symlinks. Directory junctions exercise
		// the same real-path escape without granting symlink privileges.
		command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
			"New-Item -ItemType Junction -Path $env:STRATAI_PATH_TEST_LINK -Target $env:STRATAI_PATH_TEST_TARGET -ErrorAction Stop | Out-Null")
		command.Env = append(os.Environ(),
			"STRATAI_PATH_TEST_LINK="+linkPath,
			"STRATAI_PATH_TEST_TARGET="+targetPath,
		)
		if output, junctionErr := command.CombinedOutput(); junctionErr != nil {
			t.Fatalf("cannot exercise the required link boundary: symlink failed (%v), junction failed (%v): %s", err, junctionErr, output)
		}
	}
	linkInfo, err := os.Stat(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !linkInfo.IsDir() || !os.SameFile(linkInfo, targetInfo) {
		t.Fatal("the link fixture must resolve to the intended directory")
	}
}

func TestIsPathSafeAcceptsRegularDemoFilesInsideTheTree(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "demos")
	direct := filepath.Join(base, "synthetic.dem")
	nested := filepath.Join(base, "nested", "synthetic.dem")
	writeTestArtifact(t, direct, []byte("synthetic fixture, not a playable demo"))
	writeTestArtifact(t, nested, []byte("synthetic nested fixture"))

	for _, target := range []string{direct, nested} {
		if !isPathSafe(base, target) {
			t.Errorf("regular demo inside the allowed tree was rejected: %q", target)
		}
	}
}

func TestIsPathSafeRejectsExternalSiblingMissingAndDirectoryPaths(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "demos")
	external := filepath.Join(root, "private", "synthetic.dem")
	sibling := filepath.Join(root, "demos-other", "synthetic.dem")
	directory := filepath.Join(base, "directory.dem")
	writeTestArtifact(t, external, []byte("synthetic external fixture"))
	writeTestArtifact(t, sibling, []byte("synthetic sibling fixture"))
	if err := os.MkdirAll(directory, 0o750); err != nil {
		t.Fatal(err)
	}

	for _, scenario := range []struct {
		name   string
		target string
	}{
		{"outside tree", external},
		{"sibling sharing prefix", sibling},
		{"missing file", filepath.Join(base, "missing.dem")},
		{"base directory itself", base},
		{"directory with demo extension", directory},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if isPathSafe(base, scenario.target) {
				t.Fatalf("path is not a regular file confined to the demo tree: %q", scenario.target)
			}
		})
	}
}

func TestIsPathSafeRejectsExistingLinkEscapingTheDemoTree(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "demos")
	outside := filepath.Join(root, "private")
	writeTestArtifact(t, filepath.Join(base, "allowed.dem"), []byte("synthetic allowed fixture"))
	writeTestArtifact(t, filepath.Join(outside, "private.dem"), []byte("synthetic private fixture"))
	link := filepath.Join(base, "external-link")
	createSecurityDirectoryLink(t, root, link, outside)
	escaped := filepath.Join(link, "private.dem")
	if _, err := os.Stat(escaped); err != nil {
		t.Fatalf("the escape fixture must point to an existing file: %v", err)
	}

	if isPathSafe(base, escaped) {
		t.Fatal("a symlink or junction inside demos must not authorize a file outside demos")
	}
}

func TestIsPathSafeAcceptsExistingLinkRemainingInsideTheDemoTree(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "demos")
	inside := filepath.Join(base, "nested")
	writeTestArtifact(t, filepath.Join(inside, "synthetic.dem"), []byte("synthetic internal fixture"))
	link := filepath.Join(base, "internal-link")
	createSecurityDirectoryLink(t, root, link, inside)

	if !isPathSafe(base, filepath.Join(link, "synthetic.dem")) {
		baseResolved, baseErr := filepath.EvalSymlinks(base)
		targetResolved, targetErr := filepath.EvalSymlinks(filepath.Join(link, "synthetic.dem"))
		linkTarget, linkErr := os.Readlink(link)
		_, statErr := os.Stat(filepath.Join(link, "synthetic.dem"))
		t.Logf("link=%q readlink=%q (%v), stat=%v", link, linkTarget, linkErr, statErr)
		t.Fatalf("an existing link inside the tree must remain usable: base=%q (%v), target=%q (%v)", baseResolved, baseErr, targetResolved, targetErr)
	}
}
