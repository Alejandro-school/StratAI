package geometry

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/golang/geo/r3"
	"github.com/qmuntal/gltf"
)

// Triangle represents a single triangle in 3D space
type Triangle struct {
	V0, V1, V2 r3.Vector
	Normal     r3.Vector
}

// AABB represents an Axis-Aligned Bounding Box
type AABB struct {
	Min, Max r3.Vector
}

// Intersects checks if a ray intersects the AABB
// Returns true if intersection occurs, and the distance t
func (box AABB) Intersects(origin, dir r3.Vector, maxDist float64) bool {
	tMin := 0.0
	tMax := maxDist

	for i := 0; i < 3; i++ {
		var invD, t0, t1 float64
		var minVal, maxVal, originVal float64

		if i == 0 {
			invD = 1.0 / dir.X
			minVal = box.Min.X
			maxVal = box.Max.X
			originVal = origin.X
		} else if i == 1 {
			invD = 1.0 / dir.Y
			minVal = box.Min.Y
			maxVal = box.Max.Y
			originVal = origin.Y
		} else {
			invD = 1.0 / dir.Z
			minVal = box.Min.Z
			maxVal = box.Max.Z
			originVal = origin.Z
		}

		t0 = (minVal - originVal) * invD
		t1 = (maxVal - originVal) * invD

		if invD < 0.0 {
			t0, t1 = t1, t0
		}

		if t0 > tMin {
			tMin = t0
		}
		if t1 < tMax {
			tMax = t1
		}

		if tMax <= tMin {
			return false
		}
	}
	return true
}

// BVHNode represents a node in the Bounding Volume Hierarchy
type BVHNode struct {
	AABB      AABB
	Left      *BVHNode
	Right     *BVHNode
	Triangles []Triangle // Only leaf nodes have triangles
}

// Mesh represents a collection of triangles (the map geometry)
type Mesh struct {
	Triangles []Triangle
	BVH       *BVHNode // Optimization: BVH Root
}

// BuildBVH constructs a BVH from a list of triangles
func BuildBVH(triangles []Triangle, depth int) *BVHNode {
	node := &BVHNode{}
	node.AABB = CalculateBounds(triangles)

	// Leaf node criteria: few triangles or max depth
	if len(triangles) <= 8 || depth > 20 {
		node.Triangles = triangles
		return node
	}

	// Split logic: Longest axis midpoint
	axis := 0 // 0:X, 1:Y, 2:Z
	extent := node.AABB.Max.Sub(node.AABB.Min)
	if extent.Y > extent.X && extent.Y > extent.Z {
		axis = 1
	} else if extent.Z > extent.X && extent.Z > extent.Y {
		axis = 2
	}

	mid := (node.AABB.Min.Add(node.AABB.Max)).Mul(0.5)
	var midVal float64
	if axis == 0 {
		midVal = mid.X
	} else if axis == 1 {
		midVal = mid.Y
	} else {
		midVal = mid.Z
	}

	var leftTris, rightTris []Triangle
	for _, tri := range triangles {
		// Use centroid to decide side
		centroid := tri.V0.Add(tri.V1).Add(tri.V2).Mul(1.0 / 3.0)
		var val float64
		if axis == 0 {
			val = centroid.X
		} else if axis == 1 {
			val = centroid.Y
		} else {
			val = centroid.Z
		}

		if val < midVal {
			leftTris = append(leftTris, tri)
		} else {
			rightTris = append(rightTris, tri)
		}
	}

	// Handle edge case where all triangles end up on one side
	if len(leftTris) == 0 || len(rightTris) == 0 {
		node.Triangles = triangles
		return node
	}

	node.Left = BuildBVH(leftTris, depth+1)
	node.Right = BuildBVH(rightTris, depth+1)

	return node
}

// CalculateBounds computes the AABB for a set of triangles
func CalculateBounds(triangles []Triangle) AABB {
	min := r3.Vector{X: 1e9, Y: 1e9, Z: 1e9}
	max := r3.Vector{X: -1e9, Y: -1e9, Z: -1e9}

	for _, tri := range triangles {
		for _, v := range []r3.Vector{tri.V0, tri.V1, tri.V2} {
			if v.X < min.X {
				min.X = v.X
			}
			if v.Y < min.Y {
				min.Y = v.Y
			}
			if v.Z < min.Z {
				min.Z = v.Z
			}

			if v.X > max.X {
				max.X = v.X
			}
			if v.Y > max.Y {
				max.Y = v.Y
			}
			if v.Z > max.Z {
				max.Z = v.Z
			}
		}
	}
	return AABB{Min: min, Max: max}
}

// LoadGLTF loads a .gltf or .glb file and returns a Mesh
func LoadGLTF(path string) (*Mesh, error) {
	doc, err := gltf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open gltf: %w", err)
	}

	var triangles []Triangle
	min := r3.Vector{X: 1e9, Y: 1e9, Z: 1e9}
	max := r3.Vector{X: -1e9, Y: -1e9, Z: -1e9}

	// Physics groups to INCLUDE (whitelist approach)
	// Only load geometry from these specific physics groups for raycasting
	includeGroups := []string{
		"clip",        // playerclip - blocks players and bullets
		"grenadeclip", // blocks grenades (included for completeness)
		"passbullets", // explicitly blocks bullets
		"window",      // glass/windows
	}

	for _, mesh := range doc.Meshes {
		// Check if mesh name matches any of our included groups
		meshName := strings.ToLower(mesh.Name)
		meshIncluded := false
		for _, group := range includeGroups {
			if strings.Contains(meshName, group) {
				meshIncluded = true
				break
			}
		}

		for _, primitive := range mesh.Primitives {
			// We only support TRIANGLES (mode 4)
			if primitive.Mode != gltf.PrimitiveTriangles {
				continue
			}

			// Check material name as fallback
			shouldInclude := meshIncluded
			if !shouldInclude && primitive.Material != nil {
				matIdx := int(*primitive.Material)
				if matIdx < len(doc.Materials) {
					matName := strings.ToLower(doc.Materials[matIdx].Name)
					for _, group := range includeGroups {
						if strings.Contains(matName, group) {
							shouldInclude = true
							break
						}
					}
				}
			}

			if !shouldInclude {
				continue
			}

			// Get Position Accessor
			posIdx, ok := primitive.Attributes[gltf.POSITION]
			if !ok {
				continue
			}
			posAccessor := doc.Accessors[posIdx]

			// Read positions
			// This is a simplified reader assuming float32 vec3
			// In a robust implementation we should handle sparse accessors, etc.
			// qmuntal/gltf doesn't provide high-level geometry extraction out of the box easily
			// without some boilerplate.

			// Let's try to read the buffer view
			if posAccessor.BufferView == nil {
				continue
			}
			bufferView := doc.BufferViews[*posAccessor.BufferView]
			buffer := doc.Buffers[bufferView.Buffer]

			data := buffer.Data[bufferView.ByteOffset : bufferView.ByteOffset+bufferView.ByteLength]

			// Parse vertices (assuming VEC3 FLOAT)
			// Stride
			stride := 12 // 3 * 4 bytes
			if bufferView.ByteStride != 0 {
				stride = int(bufferView.ByteStride)
			}

			count := int(posAccessor.Count)
			verts := make([]r3.Vector, count)

			for i := 0; i < count; i++ {
				offset := i * stride
				if offset+12 > len(data) {
					break
				}

				// Read float32 x, y, z
				// Little endian
				x := float32FromBytes(data[offset : offset+4])
				y := float32FromBytes(data[offset+4 : offset+8])
				z := float32FromBytes(data[offset+8 : offset+12])

				v := r3.Vector{X: float64(x), Y: float64(y), Z: float64(z)}
				verts[i] = v

				// Update bounds
				if v.X < min.X {
					min.X = v.X
				}
				if v.Y < min.Y {
					min.Y = v.Y
				}
				if v.Z < min.Z {
					min.Z = v.Z
				}
				if v.X > max.X {
					max.X = v.X
				}
				if v.Y > max.Y {
					max.Y = v.Y
				}
				if v.Z > max.Z {
					max.Z = v.Z
				}
			}

			// Indices
			if primitive.Indices != nil {
				indicesAccessor := doc.Accessors[*primitive.Indices]
				if indicesAccessor.BufferView == nil {
					continue
				}
				ibv := doc.BufferViews[*indicesAccessor.BufferView]
				ib := doc.Buffers[ibv.Buffer]
				idata := ib.Data[ibv.ByteOffset : ibv.ByteOffset+ibv.ByteLength]

				icount := int(indicesAccessor.Count)

				// Component type: 5121 (UBYTE), 5123 (USHORT), 5125 (UINT)
				compType := indicesAccessor.ComponentType

				for i := 0; i < icount; i += 3 {
					var i0, i1, i2 int

					if compType == gltf.ComponentUshort {
						i0 = int(uint16FromBytes(idata[i*2 : i*2+2]))
						i1 = int(uint16FromBytes(idata[(i+1)*2 : (i+1)*2+2]))
						i2 = int(uint16FromBytes(idata[(i+2)*2 : (i+2)*2+2]))
					} else if compType == gltf.ComponentUint {
						i0 = int(uint32FromBytes(idata[i*4 : i*4+4]))
						i1 = int(uint32FromBytes(idata[(i+1)*4 : (i+1)*4+4]))
						i2 = int(uint32FromBytes(idata[(i+2)*4 : (i+2)*4+4]))
					} else {
						// Byte
						i0 = int(idata[i])
						i1 = int(idata[i+1])
						i2 = int(idata[i+2])
					}

					if i0 >= len(verts) || i1 >= len(verts) || i2 >= len(verts) {
						continue
					}

					v0 := verts[i0]
					v1 := verts[i1]
					v2 := verts[i2]

					// Compute Normal
					edge1 := v1.Sub(v0)
					edge2 := v2.Sub(v0)
					normal := edge1.Cross(edge2).Normalize()

					triangles = append(triangles, Triangle{
						V0:     v0,
						V1:     v1,
						V2:     v2,
						Normal: normal,
					})
				}
			}
		}
	}

	// Build BVH
	fmt.Println("Building BVH...")
	bvhRoot := BuildBVH(triangles, 0)
	fmt.Println("BVH Built.")

	return &Mesh{
		Triangles: triangles,
		BVH:       bvhRoot,
	}, nil
}

func float32FromBytes(b []byte) float32 {
	bits := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return math.Float32frombits(bits)
}

func uint16FromBytes(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func uint32FromBytes(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// LoadOBJ loads a .obj file and returns a Mesh
// NOTE: This is a simplified loader. It expects vertices (v) and faces (f).
// It ignores normals/textures in the file and computes face normals.
func LoadOBJ(path string) (*Mesh, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var vertices []r3.Vector
	var triangles []Triangle

	// Initialize bounds inverted
	min := r3.Vector{X: 1e9, Y: 1e9, Z: 1e9}
	max := r3.Vector{X: -1e9, Y: -1e9, Z: -1e9}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "v": // Vertex
			if len(parts) < 4 {
				continue
			}
			x, _ := strconv.ParseFloat(parts[1], 64)
			y, _ := strconv.ParseFloat(parts[2], 64)
			z, _ := strconv.ParseFloat(parts[3], 64)
			v := r3.Vector{X: x, Y: y, Z: z}
			vertices = append(vertices, v)

			// Update bounds
			if x < min.X {
				min.X = x
			}
			if y < min.Y {
				min.Y = y
			}
			if z < min.Z {
				min.Z = z
			}
			if x > max.X {
				max.X = x
			}
			if y > max.Y {
				max.Y = y
			}
			if z > max.Z {
				max.Z = z
			}

		case "f": // Face
			// Supports "f v1 v2 v3" or "f v1/vt1/vn1 ..."
			if len(parts) < 4 {
				continue
			}
			var indices []int
			for _, p := range parts[1:] {
				// Handle v/vt/vn format
				idxStr := strings.Split(p, "/")[0]
				idx, _ := strconv.Atoi(idxStr)
				// OBJ indices are 1-based
				indices = append(indices, idx-1)
			}

			// Triangulate polygon (fan)
			for i := 1; i < len(indices)-1; i++ {
				v0 := vertices[indices[0]]
				v1 := vertices[indices[i]]
				v2 := vertices[indices[i+1]]

				// Compute Normal
				edge1 := v1.Sub(v0)
				edge2 := v2.Sub(v0)
				normal := edge1.Cross(edge2).Normalize()

				triangles = append(triangles, Triangle{
					V0:     v0,
					V1:     v1,
					V2:     v2,
					Normal: normal,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Build BVH
	bvhRoot := BuildBVH(triangles, 0)

	return &Mesh{
		Triangles: triangles,
		BVH:       bvhRoot,
	}, nil
}

// RayCast returns the distance to the first intersection and the surface normal, or -1 if none
func (m *Mesh) RayCast(origin, dir r3.Vector, maxDist float64) (float64, r3.Vector) {
	if m.BVH == nil {
		return -1, r3.Vector{}
	}
	return rayCastBVH(m.BVH, origin, dir, maxDist)
}

func rayCastBVH(node *BVHNode, origin, dir r3.Vector, maxDist float64) (float64, r3.Vector) {
	// Check AABB
	if !node.AABB.Intersects(origin, dir, maxDist) {
		return -1, r3.Vector{}
	}

	// Leaf node
	if node.Left == nil && node.Right == nil {
		minDist := -1.0
		var bestNormal r3.Vector

		for _, tri := range node.Triangles {
			dist := RayCastTriangle(origin, dir, tri, maxDist)
			if dist > 0 {
				if minDist == -1 || dist < minDist {
					minDist = dist
					bestNormal = tri.Normal
				}
			}
		}
		return minDist, bestNormal
	}

	// Internal node - check children
	distLeft := -1.0
	var normLeft r3.Vector
	distRight := -1.0
	var normRight r3.Vector

	if node.Left != nil {
		distLeft, normLeft = rayCastBVH(node.Left, origin, dir, maxDist)
	}
	if node.Right != nil {
		distRight, normRight = rayCastBVH(node.Right, origin, dir, maxDist)
	}

	if distLeft != -1 && distRight != -1 {
		if distLeft < distRight {
			return distLeft, normLeft
		}
		return distRight, normRight
	}
	if distLeft != -1 {
		return distLeft, normLeft
	}
	return distRight, normRight
}

// RayCastTriangle returns distance t if intersection occurs, or -1
func RayCastTriangle(origin, dir r3.Vector, tri Triangle, maxDist float64) float64 {
	const epsilon = 1e-6

	edge1 := tri.V1.Sub(tri.V0)
	edge2 := tri.V2.Sub(tri.V0)

	h := dir.Cross(edge2)
	a := edge1.Dot(h)

	if a > -epsilon && a < epsilon {
		return -1 // Ray is parallel to triangle
	}

	f := 1.0 / a
	s := origin.Sub(tri.V0)
	u := f * s.Dot(h)

	if u < 0.0 || u > 1.0 {
		return -1
	}

	q := s.Cross(edge1)
	v := f * dir.Dot(q)

	if v < 0.0 || u+v > 1.0 {
		return -1
	}

	t := f * edge2.Dot(q)

	if t > epsilon && t < maxDist {
		return t
	}

	return -1
}

// RayIntersectsMesh checks if a ray hits any triangle in the mesh
// Returns true if BLOCKED (hit something), false if CLEAR
// limitDist: max distance to check (0 for infinite)
func (m *Mesh) RayIntersects(start, end r3.Vector) bool {
	if m.BVH == nil {
		return false
	}

	dir := end.Sub(start)
	dist := dir.Norm()
	dir = dir.Normalize()

	return intersectBVH(m.BVH, start, dir, dist)
}

func intersectBVH(node *BVHNode, origin, dir r3.Vector, maxDist float64) bool {
	// Check AABB
	if !node.AABB.Intersects(origin, dir, maxDist) {
		return false
	}

	// Leaf node
	if node.Left == nil && node.Right == nil {
		for _, tri := range node.Triangles {
			if RayIntersectsTriangle(origin, dir, tri, maxDist) {
				return true
			}
		}
		return false
	}

	// Internal node - check children
	if node.Left != nil {
		if intersectBVH(node.Left, origin, dir, maxDist) {
			return true
		}
	}
	if node.Right != nil {
		if intersectBVH(node.Right, origin, dir, maxDist) {
			return true
		}
	}

	return false
}

// RayIntersectsTriangle implements Möller–Trumbore intersection algorithm
func RayIntersectsTriangle(origin, dir r3.Vector, tri Triangle, maxDist float64) bool {
	const epsilon = 1e-6

	edge1 := tri.V1.Sub(tri.V0)
	edge2 := tri.V2.Sub(tri.V0)

	h := dir.Cross(edge2)
	a := edge1.Dot(h)

	if a > -epsilon && a < epsilon {
		return false // Ray is parallel to triangle
	}

	f := 1.0 / a
	s := origin.Sub(tri.V0)
	u := f * s.Dot(h)

	if u < 0.0 || u > 1.0 {
		return false
	}

	q := s.Cross(edge1)
	v := f * dir.Dot(q)

	if v < 0.0 || u+v > 1.0 {
		return false
	}

	// At this stage we can compute t to find out where the intersection point is on the line.
	t := f * edge2.Dot(q)

	if t > epsilon && t < maxDist {
		return true // Intersection detected
	}

	return false
}
