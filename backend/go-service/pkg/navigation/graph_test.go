package navigation

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/golang/geo/r3"
)

func TestEvaluateAssociatesPointInsideLargeNAVPolygonBeyondCenterTolerance(t *testing.T) {
	graph := &Graph{
		Version: GraphContractVersion,
		Nodes: []Node{
			{
				ID:       1,
				Position: r3.Vector{X: 500, Y: 500},
				Corners: []r3.Vector{
					{}, {X: 1000}, {X: 1000, Y: 1000}, {Y: 1000},
				},
			},
			{
				ID:       2,
				Position: r3.Vector{X: 1500, Y: 500},
				Corners: []r3.Vector{
					{X: 1000}, {X: 2000}, {X: 2000, Y: 1000}, {X: 1000, Y: 1000},
				},
			},
		},
		Edges: []Edge{{FromID: 1, ToID: 2, Weight: 1000}},
	}
	measurement, err := Evaluate(
		graph,
		r3.Vector{X: 10, Y: 10},
		r3.Vector{X: 1990, Y: 10},
		DefaultConfig(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Reachable == nil || !*measurement.Reachable {
		t.Fatalf("large containing polygons were not associated: %#v", measurement)
	}
	if measurement.OriginAssociationDistance == nil || *measurement.OriginAssociationDistance <= DefaultEndpointTolerance {
		t.Fatalf("fixture did not prove association beyond center tolerance: %#v", measurement)
	}
}

func TestEvaluateContainingPolygonChoosesClosestVerticalLayer(t *testing.T) {
	polygon := []r3.Vector{{}, {X: 100}, {X: 100, Y: 100}, {Y: 100}}
	graph := &Graph{
		Version: GraphContractVersion,
		Nodes: []Node{
			{ID: 1, Position: r3.Vector{X: 50, Y: 50, Z: 0}, Corners: polygon},
			{ID: 2, Position: r3.Vector{X: 50, Y: 50, Z: 200}, Corners: polygon},
		},
		Edges: []Edge{{FromID: 2, ToID: 1, Weight: 200}},
	}
	measurement, err := Evaluate(
		graph,
		r3.Vector{X: 10, Y: 10, Z: 210},
		r3.Vector{X: 10, Y: 10, Z: 5},
		DefaultConfig(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if measurement.Reachable == nil || !*measurement.Reachable {
		t.Fatalf("stacked NAV layers were not associated by Z: %#v", measurement)
	}
}

func TestEvaluateRejectsInvalidVerticalTolerance(t *testing.T) {
	config := DefaultConfig()
	config.EndpointVerticalTolerance = 0
	_, err := Evaluate(nil, r3.Vector{}, r3.Vector{}, config)
	if err == nil || !strings.Contains(err.Error(), "vertical tolerance") {
		t.Fatalf("invalid vertical tolerance error = %v", err)
	}
}

func TestPreparedEvaluatorMatchesEvaluate(t *testing.T) {
	graph := &Graph{
		Version: GraphContractVersion,
		Nodes: []Node{
			{ID: 1, Position: r3.Vector{}},
			{ID: 2, Position: r3.Vector{X: 100}},
			{ID: 3, Position: r3.Vector{X: 300}},
		},
		Edges: []Edge{{FromID: 1, ToID: 2, Weight: 100}},
	}
	prepared, err := NewPreparedEvaluator(graph, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	queries := []struct {
		origin r3.Vector
		target r3.Vector
	}{
		{origin: r3.Vector{}, target: r3.Vector{X: 100}},
		{origin: r3.Vector{X: 100}, target: r3.Vector{}},
		{origin: r3.Vector{}, target: r3.Vector{X: 1000}},
	}
	for _, query := range queries {
		want, err := Evaluate(graph, query.origin, query.target, DefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		got, err := prepared.Evaluate(query.origin, query.target)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("prepared measurement differs\ngot:  %#v\nwant: %#v", got, want)
		}
	}

	empty, err := NewPreparedEvaluator(nil, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	got, err := empty.Evaluate(r3.Vector{}, r3.Vector{X: 1})
	if err != nil {
		t.Fatal(err)
	}
	want, err := Evaluate(nil, r3.Vector{}, r3.Vector{X: 1}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("empty prepared measurement differs: got %#v want %#v", got, want)
	}
}

func TestEvaluatePathReturnsStableDirectedPolyline(t *testing.T) {
	graph := &Graph{
		Version: GraphContractVersion,
		Nodes: []Node{
			{ID: 1, Position: r3.Vector{X: 0, Y: 0}},
			{ID: 2, Position: r3.Vector{X: 100, Y: 0}},
			{ID: 3, Position: r3.Vector{X: 100, Y: 100}},
			{ID: 4, Position: r3.Vector{X: 200, Y: 100}},
		},
		Edges: []Edge{
			{FromID: 1, ToID: 2, Weight: 100},
			{FromID: 2, ToID: 3, Weight: 100},
			{FromID: 3, ToID: 4, Weight: 100},
		},
	}
	prepared, err := NewPreparedEvaluator(graph, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	origin := r3.Vector{X: 1, Y: 1}
	target := r3.Vector{X: 199, Y: 99}
	trace, err := prepared.EvaluatePath(origin, target)
	if err != nil {
		t.Fatal(err)
	}
	if trace.Measurement.Reachable == nil || !*trace.Measurement.Reachable {
		t.Fatalf("trace unexpectedly unreachable: %#v", trace.Measurement)
	}
	want := []r3.Vector{
		origin,
		{X: 0, Y: 0},
		{X: 100, Y: 0},
		{X: 100, Y: 100},
		{X: 200, Y: 100},
		target,
	}
	if !reflect.DeepEqual(trace.Points, want) {
		t.Fatalf("trace points = %#v, want %#v", trace.Points, want)
	}
	reverse, err := prepared.EvaluatePath(target, origin)
	if err != nil {
		t.Fatal(err)
	}
	if reverse.Measurement.Reachable == nil || *reverse.Measurement.Reachable || len(reverse.Points) != 0 {
		t.Fatalf("reverse directed trace = %#v", reverse)
	}
}

func TestPreparedEvaluatorCachesDirectedPathsAndDefensivelyCopiesGraph(t *testing.T) {
	graph := &Graph{
		Version: GraphContractVersion,
		Nodes: []Node{
			{ID: 1, Position: r3.Vector{}, Corners: []r3.Vector{{X: -1, Y: -1}, {X: 1, Y: -1}, {X: 1, Y: 1}}},
			{ID: 2, Position: r3.Vector{X: 100}},
		},
		Edges: []Edge{{FromID: 1, ToID: 2, Weight: 100}},
	}
	prepared, err := NewPreparedEvaluator(graph, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	graph.Nodes[0].Position.X = 10_000
	graph.Nodes[0].Corners[0].X = 10_000
	graph.Edges[0].Weight = 9_999

	forward, err := prepared.Evaluate(r3.Vector{}, r3.Vector{X: 100})
	if err != nil {
		t.Fatal(err)
	}
	if forward.Reachable == nil || !*forward.Reachable || forward.PathDistance == nil || *forward.PathDistance != 100 {
		t.Fatalf("prepared graph changed after source mutation: %#v", forward)
	}
	reverse, err := prepared.Evaluate(r3.Vector{X: 100}, r3.Vector{})
	if err != nil {
		t.Fatal(err)
	}
	if reverse.Reachable == nil || *reverse.Reachable || reverse.PathDistance != nil {
		t.Fatalf("directed reverse path unexpectedly reachable: %#v", reverse)
	}
	if len(prepared.shortestPathCache) != 2 {
		t.Fatalf("directed cache entries = %d, want 2", len(prepared.shortestPathCache))
	}
	if _, err := prepared.Evaluate(r3.Vector{}, r3.Vector{X: 100}); err != nil {
		t.Fatal(err)
	}
	if len(prepared.shortestPathCache) != 2 {
		t.Fatalf("cache grew after repeated pair: %d", len(prepared.shortestPathCache))
	}
}

func TestNewPreparedEvaluatorRejectsInvalidGraph(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{{ID: 1, Position: r3.Vector{}}},
		Edges: []Edge{{FromID: 1, ToID: 2, Weight: 1}},
	}
	_, err := NewPreparedEvaluator(graph, DefaultConfig())
	if err == nil || !strings.Contains(err.Error(), "missing to node") {
		t.Fatalf("invalid graph error = %v", err)
	}
}

func TestPreparedEvaluatorSupportsConcurrentCacheUse(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: 1, Position: r3.Vector{}},
			{ID: 2, Position: r3.Vector{X: 100}},
		},
		Edges: []Edge{
			{FromID: 1, ToID: 2, Weight: 100},
			{FromID: 2, ToID: 1, Weight: 100},
		},
	}
	prepared, err := NewPreparedEvaluator(graph, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}

	const workers = 16
	resultErrors := make(chan error, workers)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(reverse bool) {
			defer wait.Done()
			origin, target := r3.Vector{}, r3.Vector{X: 100}
			if reverse {
				origin, target = target, origin
			}
			measurement, err := prepared.Evaluate(origin, target)
			if err != nil {
				resultErrors <- err
				return
			}
			if measurement.Reachable == nil || !*measurement.Reachable {
				resultErrors <- errors.New("unreachable concurrent result")
			}
		}(worker%2 == 1)
	}
	wait.Wait()
	close(resultErrors)
	for err := range resultErrors {
		if err != nil {
			t.Fatal(err)
		}
	}
}
