package maps

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNavigationCandidatePinnedAssetAndDirectedGraph(t *testing.T) {
	var data bytes.Buffer
	for _, value := range []uint32{0xfeedface, 35, 1, 1} {
		writeLE(t, &data, value)
	}
	data.Write(buildSource2V35Fixture(t))
	path := filepath.Join(t.TempDir(), "fixture.nav")
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(data.Bytes()))
	graph, err := LoadNavigationCandidate(path, "fixture.nav", checksum)
	if err != nil {
		t.Fatal(err)
	}
	if graph.AssetSHA256 != checksum || len(graph.Nodes) != 2 || len(graph.Edges) != 2 {
		t.Fatalf("unexpected snapshot: %+v", graph)
	}
	if _, err := LoadNavigationCandidate(path, "fixture.nav", "wrong"); err == nil {
		t.Fatal("accepted wrong hash")
	}
	for _, index := range []int{0, 4, 8} {
		bad := append([]byte(nil), data.Bytes()...)
		bad[index]++
		if err := os.WriteFile(path, bad, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadNavigationCandidate(path, "fixture.nav", fmt.Sprintf("%x", sha256.Sum256(bad))); err == nil {
			t.Fatal("accepted incompatible NAV")
		}
	}
}
