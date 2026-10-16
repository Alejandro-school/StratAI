package maps

import (
	"testing"

	"cs2-demo-service/pkg/navigation"

	"github.com/golang/geo/r3"
)

func TestNavigationGraphSnapshotUsesNAVConnectionsAndIsDefensive(t *testing.T) {
	manager := NewMapManager("maps")
	manager.currentNav = &NavMesh{Areas: []NavArea{
		{
			ID:          20,
			Corners:     []r3.Vector{{X: 10}, {X: 10}},
			Connections: []uint32{},
		},
		{
			ID:          10,
			Corners:     []r3.Vector{{}, {}},
			Connections: []uint32{20, 20},
		},
	}}
	manager.inputs["nav_mesh"] = InputProvenance{
		RelativePath: "de_test/de_test.nav",
		SHA256:       "abc123",
		Used:         true,
	}

	graph, err := manager.NavigationGraphSnapshot()
	if err != nil {
		t.Fatalf("NavigationGraphSnapshot() error = %v", err)
	}
	if graph == nil || graph.Version != navigation.GraphContractVersion {
		t.Fatalf("graph = %#v", graph)
	}
	if graph.AssetReference != "de_test/de_test.nav" || graph.AssetSHA256 != "abc123" {
		t.Fatalf("graph provenance = %#v", graph)
	}
	if len(graph.Nodes) != 2 || graph.Nodes[0].ID != 10 || graph.Nodes[1].ID != 20 {
		t.Fatalf("nodes are not sorted deterministically: %#v", graph.Nodes)
	}
	if len(graph.Edges) != 1 || graph.Edges[0].FromID != 10 || graph.Edges[0].ToID != 20 || graph.Edges[0].Weight != 10 {
		t.Fatalf("NAV connections were not preserved: %#v", graph.Edges)
	}

	graph.Nodes[0].Position.X = 999
	graph.Nodes[0].Corners[0].X = 999
	graphAgain, err := manager.NavigationGraphSnapshot()
	if err != nil {
		t.Fatalf("second NavigationGraphSnapshot() error = %v", err)
	}
	if graphAgain.Nodes[0].Position.X != 0 {
		t.Fatal("snapshot mutation leaked back into MapManager")
	}
	if graphAgain.Nodes[0].Corners[0].X != 0 {
		t.Fatal("snapshot corner mutation leaked back into MapManager")
	}
}

func TestNavigationGraphSnapshotMissingNAVFailsClosed(t *testing.T) {
	manager := NewMapManager("maps")
	graph, err := manager.NavigationGraphSnapshot()
	if err != nil {
		t.Fatalf("NavigationGraphSnapshot() error = %v", err)
	}
	if graph != nil {
		t.Fatalf("graph = %#v, want nil", graph)
	}
}
