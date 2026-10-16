package geometry

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"github.com/golang/geo/r3"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// A wall exported in ordinary Source 2 world geometry, rather than a player clip.
func replayVisionFixture(t *testing.T, meshName, materialName string) string {
	t.Helper()
	data := make([]byte, 39)
	for i, v := range []float32{5, -1, -1, 5, 1, -1, 5, 0, 1} {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(v))
	}
	copy(data[36:], []byte{0, 1, 2})
	doc := map[string]any{
		"asset":       map[string]any{"version": "2.0"},
		"buffers":     []any{map[string]any{"byteLength": 39, "uri": "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(data)}},
		"bufferViews": []any{map[string]any{"buffer": 0, "byteLength": 36}, map[string]any{"buffer": 0, "byteOffset": 36, "byteLength": 3}},
		"accessors":   []any{map[string]any{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3"}, map[string]any{"bufferView": 1, "componentType": 5121, "count": 3, "type": "SCALAR"}},
		"materials":   []any{map[string]any{"name": materialName}},
		"meshes":      []any{map[string]any{"name": meshName, "primitives": []any{map[string]any{"attributes": map[string]any{"POSITION": 0}, "indices": 1, "material": 0, "mode": 4}}}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "wall.gltf")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReplayVisionBlocksOrdinaryOpaqueWorldInBothDirections(t *testing.T) {
	for _, surface := range []string{"concrete", "brick", "metal", "Wood", "plaster"} {
		t.Run(surface, func(t *testing.T) {
			path := replayVisionFixture(t, "physics_group_"+surface, "")
			legacy, err := LoadGLTF(path)
			if err != nil {
				t.Fatal(err)
			}
			mesh, err := LoadReplayVisionGLTF(path)
			if err != nil {
				t.Fatal(err)
			}
			start, end := r3.Vector{}, r3.Vector{X: 10}
			if legacy.RayIntersects(start, end) {
				t.Fatal("legacy profile changed")
			}
			if !mesh.RayIntersects(start, end) || !mesh.RayIntersects(end, start) {
				t.Fatalf("opaque wall did not obstruct vision: %#v", mesh.Triangles)
			}
			if mesh.RayIntersects(start, r3.Vector{Y: 10}) {
				t.Fatal("open ray incorrectly obstructed")
			}
		})
	}
}

func TestReplayVisionIncludesOpaqueMaterialFallback(t *testing.T) {
	mesh, err := LoadReplayVisionGLTF(replayVisionFixture(t, "unnamed", "physics_group_concrete_material"))
	if err != nil {
		t.Fatal(err)
	}
	if !mesh.RayIntersects(r3.Vector{}, r3.Vector{X: 10}) {
		t.Fatal("material wall omitted")
	}
}

func TestReplayVisionDoesNotTreatTransparentOrMovementClipsAsOpaque(t *testing.T) {
	for _, name := range []string{"physics_group_glass", "physics_group_chainlink", "physics_npcclip_playerclip", "physics_csgo_grenadeclip", "physics_passbullets_window_chainlink"} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadReplayVisionGLTF(replayVisionFixture(t, name, "")); err == nil {
				t.Fatal("nonopaque-only mesh must not claim usable vision geometry")
			}
		})
	}
}

func TestFlatWallBVHMatchesIndependentTriangleIntersection(t *testing.T) {
	mesh, err := LoadReplayVisionGLTF(replayVisionFixture(t, "physics_group_concrete", ""))
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]r3.Vector{
		{{}, {X: 10}}, {{X: 10}, {}}, {{Y: -1, Z: -1}, {X: 10, Y: -1, Z: -1}},
		{{Y: 2}, {X: 10, Y: 2}}, {{X: 5, Y: -2}, {X: 5, Y: 2}},
	} {
		start, end := pair[0], pair[1]
		direction := end.Sub(start)
		expected := false
		for _, tri := range mesh.Triangles {
			if RayCastTriangle(start, direction.Normalize(), tri, direction.Norm()) > 0 {
				expected = true
			}
		}
		if mesh.RayIntersects(start, end) != expected {
			t.Fatalf("BVH mismatch for %v -> %v", start, end)
		}
	}
}
