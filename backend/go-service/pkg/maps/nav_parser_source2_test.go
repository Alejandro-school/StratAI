package maps

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/geo/r3"
)

type source2FixtureConnection struct {
	targetID uint32
	edgeID   uint32
}

func TestParseSource2V35ReadsAreaLayoutAndEightByteConnections(t *testing.T) {
	payload := buildSource2V35Fixture(t)
	mesh, err := parseSource2(bytes.NewReader(payload), &NavMesh{}, 35)
	if err != nil {
		t.Fatalf("parseSource2() error = %v", err)
	}

	if got, want := len(mesh.corners), 6; got != want {
		t.Fatalf("corner count = %d, want %d", got, want)
	}
	if got, want := len(mesh.polygons), 2; got != want {
		t.Fatalf("polygon count = %d, want %d", got, want)
	}
	if got, want := len(mesh.Areas), 2; got != want {
		t.Fatalf("area count = %d, want %d", got, want)
	}

	first := mesh.Areas[0]
	if first.ID != 1 {
		t.Errorf("first area ID = %d, want 1", first.ID)
	}
	if got, want := first.Connections, []uint32{2, 2}; !equalUint32s(got, want) {
		t.Errorf("first area connections = %v, want %v; edge and ladder IDs must not become area edges", got, want)
	}
	if got, want := first.NW, (r3.Vector{X: 0, Y: 0, Z: 0}); got != want {
		t.Errorf("first area NW = %v, want %v", got, want)
	}
	if got, want := first.SE, (r3.Vector{X: 10, Y: 10, Z: 0}); got != want {
		t.Errorf("first area SE = %v, want %v", got, want)
	}
	if math.Abs(first.Center.X-10.0/3.0) > 1e-9 || math.Abs(first.Center.Y-10.0/3.0) > 1e-9 || first.Center.Z != 0 {
		t.Errorf("first area center = %v, want (10/3, 10/3, 0)", first.Center)
	}

	second := mesh.Areas[1]
	if second.ID != 2 {
		t.Errorf("second area ID = %d, want 2", second.ID)
	}
	if got, want := second.Connections, []uint32{1}; !equalUint32s(got, want) {
		t.Errorf("second area connections = %v, want %v", got, want)
	}
}

func TestLoadNavMeshV35Fixture(t *testing.T) {
	var file bytes.Buffer
	writeLE(t, &file, uint32(0xFEEDFACE))
	writeLE(t, &file, uint32(35))
	writeLE(t, &file, uint32(1))
	writeLE(t, &file, uint32(1))
	file.Write(buildSource2V35Fixture(t))

	path := filepath.Join(t.TempDir(), "fixture-v35.nav")
	if err := os.WriteFile(path, file.Bytes(), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	mesh, err := LoadNavMesh(path)
	if err != nil {
		t.Fatalf("LoadNavMesh() error = %v", err)
	}
	if got, want := len(mesh.Areas), 2; got != want {
		t.Fatalf("area count = %d, want %d", got, want)
	}
}

func TestParseSource2V35RejectsConnectionCountBeyondRemainingBytes(t *testing.T) {
	var payload bytes.Buffer
	writeSource2FixturePrefix(t, &payload, []uint32{0, 1, 2})
	writeLE(t, &payload, uint32(1)) // area count
	writeSource2AreaHeader(t, &payload, 1, 0)
	writeLE(t, &payload, uint32(95_289)) // first edge count, with no connection records
	payload.Write(make([]byte, 9))       // enough for the conservative per-area lower bound

	_, err := parseSource2(bytes.NewReader(payload.Bytes()), &NavMesh{}, 35)
	if err == nil {
		t.Fatal("parseSource2() error = nil, want a bounded-count error")
	}
	if !strings.Contains(err.Error(), "connection count") || !strings.Contains(err.Error(), "only 9 remain") {
		t.Fatalf("parseSource2() error = %q, want connection count/remaining-bytes context", err)
	}
}

func TestParseSource2V35RejectsInvalidPolygonCornerReference(t *testing.T) {
	var payload bytes.Buffer
	writeLE(t, &payload, uint32(3))
	for _, corner := range []r3.Vector{{X: 0}, {Y: 1}, {Z: 2}} {
		writeLE(t, &payload, float32(corner.X))
		writeLE(t, &payload, float32(corner.Y))
		writeLE(t, &payload, float32(corner.Z))
	}
	writeLE(t, &payload, uint32(1))
	writeLE(t, &payload, uint8(3))
	for _, index := range []uint32{0, 1, 99} {
		writeLE(t, &payload, index)
	}
	writeLE(t, &payload, uint32(0xFFFFFFFF))

	_, err := parseSource2(bytes.NewReader(payload.Bytes()), &NavMesh{}, 35)
	if err == nil || !strings.Contains(err.Error(), "invalid corner index 99") {
		t.Fatalf("parseSource2() error = %v, want invalid corner index", err)
	}
}

func TestLoadNavMeshRejectsUnsupportedSource2VersionBeforeParsing(t *testing.T) {
	var file bytes.Buffer
	writeLE(t, &file, uint32(0xFEEDFACE))
	writeLE(t, &file, uint32(36))
	writeLE(t, &file, uint32(1))
	writeLE(t, &file, uint32(0))

	path := filepath.Join(t.TempDir(), "v36.nav")
	if err := os.WriteFile(path, file.Bytes(), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := LoadNavMesh(path)
	if err == nil || !strings.Contains(err.Error(), "unsupported nav version: 36") {
		t.Fatalf("LoadNavMesh() error = %v, want unsupported v36", err)
	}
}

func TestLoadNavMeshV35MirageAssetWhenAvailable(t *testing.T) {
	path := filepath.Clean(filepath.Join("..", "..", "..", "data", "maps", "de_mirage", "de_mirage.nav"))
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("local Mirage NAV asset is not available")
	} else if err != nil {
		t.Fatalf("stat Mirage NAV asset: %v", err)
	}

	mesh, err := LoadNavMesh(path)
	if err != nil {
		t.Fatalf("LoadNavMesh(%q) error = %v", path, err)
	}
	if got, want := len(mesh.corners), 3613; got != want {
		t.Fatalf("Mirage corner count = %d, want %d", got, want)
	}
	if got, want := len(mesh.polygons), 2544; got != want {
		t.Fatalf("Mirage polygon count = %d, want %d", got, want)
	}
	if got, want := len(mesh.Areas), 2544; got != want {
		t.Fatalf("Mirage area count = %d, want %d", got, want)
	}

	areaIDs := make(map[uint32]struct{}, len(mesh.Areas))
	connectionCount := 0
	for _, area := range mesh.Areas {
		if len(area.Corners) < 3 {
			t.Fatalf("Mirage area %d has %d corners", area.ID, len(area.Corners))
		}
		areaIDs[area.ID] = struct{}{}
		connectionCount += len(area.Connections)
	}
	if connectionCount == 0 {
		t.Fatal("Mirage parsed with zero area connections")
	}
	for _, area := range mesh.Areas {
		for _, targetID := range area.Connections {
			if _, exists := areaIDs[targetID]; !exists {
				t.Fatalf("Mirage area %d references missing area %d", area.ID, targetID)
			}
		}
	}

	manager := &MapManager{currentNav: mesh}
	graph, err := manager.NavigationGraphSnapshot()
	if err != nil {
		t.Fatalf("NavigationGraphSnapshot() error = %v", err)
	}
	if graph == nil || len(graph.Nodes) != len(mesh.Areas) || len(graph.Edges) == 0 {
		t.Fatalf("navigation graph = %#v, want %d nodes and non-empty edges", graph, len(mesh.Areas))
	}
	t.Logf("Mirage NAV: corners=%d polygons=%d areas=%d raw_connections=%d graph_edges=%d", len(mesh.corners), len(mesh.polygons), len(mesh.Areas), connectionCount, len(graph.Edges))
}

func buildSource2V35Fixture(t *testing.T) []byte {
	t.Helper()
	var payload bytes.Buffer

	writeLE(t, &payload, uint32(6))
	for _, corner := range []r3.Vector{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 10, Z: 0},
		{X: 10, Y: 0, Z: 0},
		{X: 10, Y: 10, Z: 0},
		{X: 0, Y: 10, Z: 0},
	} {
		writeLE(t, &payload, float32(corner.X))
		writeLE(t, &payload, float32(corner.Y))
		writeLE(t, &payload, float32(corner.Z))
	}

	writeLE(t, &payload, uint32(2))
	for _, indices := range [][]uint32{{0, 1, 2}, {3, 4, 5}} {
		writeLE(t, &payload, uint8(len(indices)))
		for _, index := range indices {
			writeLE(t, &payload, index)
		}
		writeLE(t, &payload, uint32(0xFFFFFFFF))
	}
	writeLE(t, &payload, uint32(0)) // v32+ global unknown

	writeLE(t, &payload, uint32(1))
	payload.WriteString("moving_platform")
	payload.WriteByte(0)
	payload.Write(make([]byte, 48))

	writeLE(t, &payload, uint32(2))
	writeSource2Area(t, &payload, 1, 0, [][]source2FixtureConnection{
		{{targetID: 2, edgeID: 900}},
		nil,
		{{targetID: 2, edgeID: 901}},
	}, []uint32{700}, []uint32{701})
	writeSource2Area(t, &payload, 2, 1, [][]source2FixtureConnection{
		{{targetID: 1, edgeID: 902}},
		nil,
		nil,
	}, nil, nil)

	return payload.Bytes()
}

func writeSource2FixturePrefix(t *testing.T, payload *bytes.Buffer, indices []uint32) {
	t.Helper()
	writeLE(t, payload, uint32(3))
	for _, corner := range []r3.Vector{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}} {
		writeLE(t, payload, float32(corner.X))
		writeLE(t, payload, float32(corner.Y))
		writeLE(t, payload, float32(corner.Z))
	}
	writeLE(t, payload, uint32(1))
	writeLE(t, payload, uint8(len(indices)))
	for _, index := range indices {
		writeLE(t, payload, index)
	}
	writeLE(t, payload, uint32(0xFFFFFFFF))
	writeLE(t, payload, uint32(0)) // v32+ global unknown
	writeLE(t, payload, uint32(0)) // v35 movable mesh count
}

func writeSource2Area(
	t *testing.T,
	payload *bytes.Buffer,
	areaID uint32,
	polygonIndex uint32,
	edges [][]source2FixtureConnection,
	laddersAbove []uint32,
	laddersBelow []uint32,
) {
	t.Helper()
	writeSource2AreaHeader(t, payload, areaID, polygonIndex)
	for _, connections := range edges {
		writeLE(t, payload, uint32(len(connections)))
		for _, connection := range connections {
			writeLE(t, payload, connection.targetID)
			writeLE(t, payload, connection.edgeID)
		}
	}
	writeLE(t, payload, uint8(0))
	writeLE(t, payload, uint32(0))
	writeLE(t, payload, uint32(len(laddersAbove)))
	for _, ladderID := range laddersAbove {
		writeLE(t, payload, ladderID)
	}
	writeLE(t, payload, uint32(len(laddersBelow)))
	for _, ladderID := range laddersBelow {
		writeLE(t, payload, ladderID)
	}
}

func writeSource2AreaHeader(t *testing.T, payload *bytes.Buffer, areaID, polygonIndex uint32) {
	t.Helper()
	writeLE(t, payload, areaID)
	writeLE(t, payload, int64(0x0102030405060708))
	writeLE(t, payload, uint8(3))
	writeLE(t, payload, polygonIndex)
	writeLE(t, payload, float32(0))
}

func writeLE(t *testing.T, payload *bytes.Buffer, value any) {
	t.Helper()
	if err := binary.Write(payload, binary.LittleEndian, value); err != nil {
		t.Fatalf("binary.Write(%T): %v", value, err)
	}
}

func equalUint32s(left, right []uint32) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
