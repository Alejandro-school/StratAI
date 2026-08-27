package geometry

import (
	"testing"

	"github.com/golang/geo/r3"
)

func TestBuildBVHStaysBalancedForClusteredTriangles(t *testing.T) {
	triangles := make([]Triangle, 257)
	for i := range triangles {
		x := float64(i) * 0.001
		triangles[i] = Triangle{
			V0: r3.Vector{X: x, Y: 0, Z: 0},
			V1: r3.Vector{X: x, Y: 1, Z: 0},
			V2: r3.Vector{X: x, Y: 0, Z: 1},
		}
	}

	root := BuildBVH(triangles, 0)
	maxLeafSize := 0
	var visit func(*BVHNode)
	visit = func(node *BVHNode) {
		if node.Left == nil && node.Right == nil {
			maxLeafSize = max(maxLeafSize, len(node.Triangles))
			return
		}
		visit(node.Left)
		visit(node.Right)
	}
	visit(root)

	if maxLeafSize > 8 {
		t.Fatalf("BVH leaf contains %d triangles; expected at most 8", maxLeafSize)
	}
}

func TestBalancedBVHPreservesRayIntersection(t *testing.T) {
	triangles := []Triangle{
		{V0: r3.Vector{X: 5, Y: -1, Z: -1}, V1: r3.Vector{X: 5, Y: 1, Z: -1}, V2: r3.Vector{X: 5, Y: 0, Z: 1}},
		{V0: r3.Vector{X: 20, Y: -1, Z: -1}, V1: r3.Vector{X: 20, Y: 1, Z: -1}, V2: r3.Vector{X: 20, Y: 0, Z: 1}},
	}
	mesh := Mesh{Triangles: triangles, BVH: BuildBVH(triangles, 0)}

	if !mesh.RayIntersects(r3.Vector{}, r3.Vector{X: 10}) {
		t.Fatal("expected the ray to hit the near triangle")
	}
	if mesh.RayIntersects(r3.Vector{}, r3.Vector{Y: 10}) {
		t.Fatal("unexpected intersection outside the triangles")
	}
}
