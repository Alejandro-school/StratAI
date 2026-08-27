package maps

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/geo/r3"
)

func TestVisibilityIsUnknownWhenNoPhysicsMeshIsLoaded(t *testing.T) {
	manager := NewMapManager(t.TempDir())
	if manager.IsVisible(r3.Vector{}, r3.Vector{X: 100}) {
		t.Fatal("missing geometry must not be reported as visible")
	}
	if manager.TraceRay(r3.Vector{}, r3.Vector{X: 100}) {
		t.Fatal("missing geometry must not be reported as a clear ray")
	}
}

func TestInputProvenanceHashesRelativeAssetAndReturnsCopy(t *testing.T) {
	mapsDir := t.TempDir()
	assetPath := filepath.Join(mapsDir, "de_test", "de_test.nav")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	contents := []byte("map-asset")
	if err := os.WriteFile(assetPath, contents, 0o644); err != nil {
		t.Fatal(err)
	}

	manager := NewMapManager(mapsDir)
	loadErr := fmt.Errorf("unsupported nav version")
	manager.recordInput("nav_mesh", assetPath, false, loadErr)
	inputs := manager.InputProvenance()
	input := inputs["nav_mesh"]
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(contents))
	if input.RelativePath != "de_test/de_test.nav" || input.SHA256 != expectedHash {
		t.Fatalf("unexpected input provenance: %+v", input)
	}
	if input.Used || input.LoadError != loadErr.Error() {
		t.Fatalf("expected inspected-but-unused input: %+v", input)
	}

	delete(inputs, "nav_mesh")
	if _, ok := manager.InputProvenance()["nav_mesh"]; !ok {
		t.Fatal("InputProvenance must return a defensive copy")
	}
}
