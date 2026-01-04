package maps

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/golang/geo/r3"
)

// NavArea represents a navigation area
type NavArea struct {
	ID      uint32
	NW      r3.Vector // Used for v16 (AABB)
	SE      r3.Vector // Used for v16 (AABB)
	PlaceID uint16

	// v35 Geometry
	Corners []r3.Vector
	Center  r3.Vector
}

// NavMesh represents the navigation mesh
type NavMesh struct {
	Places []string
	Areas  []NavArea

	// Internal data for v35 reconstruction
	corners  []r3.Vector
	polygons []navPolygon
}

type navPolygon struct {
	indices []uint32
}

// LoadNavMesh loads a .nav file (Supports CS:GO v16 and CS2 v35)
func LoadNavMesh(path string) (*NavMesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mesh := &NavMesh{}

	// Debug: Print first 32 bytes
	headerBuf := make([]byte, 32)
	f.Read(headerBuf)
	fmt.Printf("Header Hex: %x\n", headerBuf)
	f.Seek(0, 0)

	// Read Magic
	var magic uint32
	if err := binary.Read(f, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if magic != 0xFEEDFACE {
		return nil, fmt.Errorf("invalid magic number: %x", magic)
	}

	// Read Version
	var version uint32
	if err := binary.Read(f, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	fmt.Printf("Version: %d\n", version)

	// SubVersion
	var subVersion uint32
	if err := binary.Read(f, binary.LittleEndian, &subVersion); err != nil {
		return nil, err
	}
	fmt.Printf("SubVersion: %d\n", subVersion)

	// BspSize
	var bspSize uint32
	if err := binary.Read(f, binary.LittleEndian, &bspSize); err != nil {
		return nil, err
	}
	fmt.Printf("BspSize: %d\n", bspSize)

	// --- VERSION SPECIFIC PARSING ---

	if version >= 31 {
		// CS2 (Source 2) Logic
		// In v35, it seems IsAnalyzed and PlaceCount are NOT present immediately after BspSize.
		// Or at least, the data following BspSize looks like CornerCount (0x10A9 = 4265).
		// So we jump straight to version specific parsing.
		return parseSource2(f, mesh, version)
	}

	// IsAnalyzed (Only for v16 / Source 1)
	var isAnalyzed uint8
	if err := binary.Read(f, binary.LittleEndian, &isAnalyzed); err != nil {
		return nil, err
	}
	fmt.Printf("IsAnalyzed: %d\n", isAnalyzed)

	// PlaceCount
	var placeCount uint16
	if err := binary.Read(f, binary.LittleEndian, &placeCount); err != nil {
		return nil, err
	}
	fmt.Printf("Place Count: %d\n", placeCount)

	// Places
	mesh.Places = make([]string, placeCount)
	for i := 0; i < int(placeCount); i++ {
		var len uint16
		if err := binary.Read(f, binary.LittleEndian, &len); err != nil {
			return nil, err
		}

		buf := make([]byte, len)
		if err := binary.Read(f, binary.LittleEndian, &buf); err != nil {
			return nil, err
		}
		// Remove null terminator if present
		if len > 0 && buf[len-1] == 0 {
			mesh.Places[i] = string(buf[:len-1])
		} else {
			mesh.Places[i] = string(buf)
		}
	}

	if version == 16 {
		// CS:GO (Source 1) Logic
		return parseSource1(f, mesh)
	}

	return nil, fmt.Errorf("unsupported nav version: %d", version)
}

func parseSource1(r io.Reader, mesh *NavMesh) (*NavMesh, error) {
	// AreaCount
	var areaCount uint32
	if err := binary.Read(r, binary.LittleEndian, &areaCount); err != nil {
		return nil, err
	}

	mesh.Areas = make([]NavArea, 0, areaCount)

	for i := 0; i < int(areaCount); i++ {
		area := NavArea{}

		// ID
		if err := binary.Read(r, binary.LittleEndian, &area.ID); err != nil {
			return nil, err
		}

		// Flags
		var flags uint32
		if err := binary.Read(r, binary.LittleEndian, &flags); err != nil {
			return nil, err
		}

		// NW, SE, NE Z, SW Z
		var nwX, nwY, nwZ float32
		var seX, seY, seZ float32
		var neZ, swZ float32

		binary.Read(r, binary.LittleEndian, &nwX)
		binary.Read(r, binary.LittleEndian, &nwY)
		binary.Read(r, binary.LittleEndian, &nwZ)
		binary.Read(r, binary.LittleEndian, &seX)
		binary.Read(r, binary.LittleEndian, &seY)
		binary.Read(r, binary.LittleEndian, &seZ)
		binary.Read(r, binary.LittleEndian, &neZ)
		binary.Read(r, binary.LittleEndian, &swZ)

		area.NW = r3.Vector{X: float64(nwX), Y: float64(nwY), Z: float64(nwZ)}
		area.SE = r3.Vector{X: float64(seX), Y: float64(seY), Z: float64(seZ)}

		// Calculate corners for unified interface
		area.Corners = []r3.Vector{
			{X: float64(nwX), Y: float64(nwY), Z: float64(nwZ)}, // NW
			{X: float64(seX), Y: float64(nwY), Z: float64(neZ)}, // NE
			{X: float64(seX), Y: float64(seY), Z: float64(seZ)}, // SE
			{X: float64(nwX), Y: float64(seY), Z: float64(swZ)}, // SW
		}

		// Connections (4 directions: N, E, S, W)
		for dir := 0; dir < 4; dir++ {
			var count uint32
			binary.Read(r, binary.LittleEndian, &count)
			for k := 0; k < int(count); k++ {
				var connectID uint32
				binary.Read(r, binary.LittleEndian, &connectID)
			}
		}

		// Hiding Spots
		var hidingCount uint8
		binary.Read(r, binary.LittleEndian, &hidingCount)
		for k := 0; k < int(hidingCount); k++ {
			var hidID uint32
			binary.Read(r, binary.LittleEndian, &hidID)
			var hidFlags uint8
			binary.Read(r, binary.LittleEndian, &hidFlags)
			var hx, hy, hz float32
			binary.Read(r, binary.LittleEndian, &hx)
			binary.Read(r, binary.LittleEndian, &hy)
			binary.Read(r, binary.LittleEndian, &hz)
		}

		// Encounter Paths
		var encounterCount uint32
		binary.Read(r, binary.LittleEndian, &encounterCount)
		for k := 0; k < int(encounterCount); k++ {
			var fromID, fromDir, toID, toDir uint32
			var spotCount uint8
			binary.Read(r, binary.LittleEndian, &fromID)
			binary.Read(r, binary.LittleEndian, &fromDir)
			binary.Read(r, binary.LittleEndian, &toID)
			binary.Read(r, binary.LittleEndian, &toDir)
			binary.Read(r, binary.LittleEndian, &spotCount)
			for s := 0; s < int(spotCount); s++ {
				var orderID uint32
				var t float32
				binary.Read(r, binary.LittleEndian, &orderID)
				binary.Read(r, binary.LittleEndian, &t)
			}
		}

		// PlaceID
		if err := binary.Read(r, binary.LittleEndian, &area.PlaceID); err != nil {
			return nil, err
		}

		// Skip Ladder Connections (2 directions)
		for dir := 0; dir < 2; dir++ {
			var count uint32
			binary.Read(r, binary.LittleEndian, &count)
			for k := 0; k < int(count); k++ {
				var connectID uint32
				binary.Read(r, binary.LittleEndian, &connectID)
			}
		}

		// Earliest Occupier (2 teams)
		for team := 0; team < 2; team++ {
			var time float32
			binary.Read(r, binary.LittleEndian, &time)
		}

		mesh.Areas = append(mesh.Areas, area)
	}

	return mesh, nil
}

func parseSource2(r io.Reader, mesh *NavMesh, version uint32) (*NavMesh, error) {
	fmt.Printf("Parsing Source 2 Nav Mesh (Version %d)\n", version)
	// 1. Read Corners
	var cornerCount uint32
	if err := binary.Read(r, binary.LittleEndian, &cornerCount); err != nil {
		return nil, fmt.Errorf("failed to read corner count: %v", err)
	}
	fmt.Printf("Corner Count: %d\n", cornerCount)

	mesh.corners = make([]r3.Vector, cornerCount)
	for i := 0; i < int(cornerCount); i++ {
		var x, y, z float32
		if err := binary.Read(r, binary.LittleEndian, &x); err != nil {
			return nil, fmt.Errorf("failed to read corner X at %d: %v", i, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &y); err != nil {
			return nil, fmt.Errorf("failed to read corner Y at %d: %v", i, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &z); err != nil {
			return nil, fmt.Errorf("failed to read corner Z at %d: %v", i, err)
		}
		mesh.corners[i] = r3.Vector{X: float64(x), Y: float64(y), Z: float64(z)}
	}

	if f, ok := r.(*os.File); ok {
		offset, _ := f.Seek(0, io.SeekCurrent)
		fmt.Printf("Offset after corners: %d\n", offset)
	}

	// 2. Read Polygons
	var polygonCount uint32
	if err := binary.Read(r, binary.LittleEndian, &polygonCount); err != nil {
		return nil, fmt.Errorf("failed to read polygon count: %v", err)
	}
	fmt.Printf("Polygon Count: %d\n", polygonCount)

	mesh.polygons = make([]navPolygon, polygonCount)
	for i := 0; i < int(polygonCount); i++ {
		var polyCornerCount uint8
		if err := binary.Read(r, binary.LittleEndian, &polyCornerCount); err != nil {
			return nil, fmt.Errorf("failed to read poly corner count at %d: %v", i, err)
		}

		indices := make([]uint32, polyCornerCount)
		if err := binary.Read(r, binary.LittleEndian, &indices); err != nil {
			return nil, fmt.Errorf("failed to read poly indices at %d: %v", i, err)
		}
		mesh.polygons[i] = navPolygon{indices: indices}

		// Skip UnkPoly (Version >= 35)
		if version >= 35 {
			var unkPoly uint32
			if err := binary.Read(r, binary.LittleEndian, &unkPoly); err != nil {
				return nil, fmt.Errorf("failed to read unkPoly at %d: %v", i, err)
			}
		}
	}

	// 3. Global Unknowns
	if version >= 32 {
		var unk2 uint32
		if err := binary.Read(r, binary.LittleEndian, &unk2); err != nil {
			return nil, fmt.Errorf("failed to read unk2: %v", err)
		}
	}
	if version >= 35 {
		var unk3 uint32
		if err := binary.Read(r, binary.LittleEndian, &unk3); err != nil {
			return nil, fmt.Errorf("failed to read unk3: %v", err)
		}
	}

	// 4. Areas
	var areaCount uint32
	if err := binary.Read(r, binary.LittleEndian, &areaCount); err != nil {
		return nil, fmt.Errorf("failed to read area count: %v", err)
	}
	fmt.Printf("Area Count: %d\n", areaCount)

	mesh.Areas = make([]NavArea, areaCount)
	for i := 0; i < int(areaCount); i++ {
		area := NavArea{}

		// Determine Byte Order
		// Area 0 is Little Endian. Area 1+ seems to be Big Endian in v35?
		var order binary.ByteOrder = binary.LittleEndian
		if i > 0 {
			order = binary.BigEndian
		}

		// ID
		if err := binary.Read(r, order, &area.ID); err != nil {
			return nil, fmt.Errorf("failed to read area ID at %d: %v", i, err)
		}

		var flags uint64
		var hullIndex uint32
		var polygonIndex uint32
		var unkArea uint32
		var padding uint8

		var flags32 uint32
		if err := binary.Read(r, order, &flags32); err != nil {
			return nil, fmt.Errorf("failed to read flags at %d: %v", i, err)
		}
		flags = uint64(flags32)

		if err := binary.Read(r, order, &hullIndex); err != nil {
			return nil, fmt.Errorf("failed to read hullIndex at %d: %v", i, err)
		}

		if err := binary.Read(r, order, &polygonIndex); err != nil {
			return nil, fmt.Errorf("failed to read polygonIndex at %d: %v", i, err)
		}

		if err := binary.Read(r, order, &unkArea); err != nil {
			return nil, fmt.Errorf("failed to read unkArea at %d: %v", i, err)
		}

		// Padding (1 byte)
		if err := binary.Read(r, binary.LittleEndian, &padding); err != nil {
			return nil, fmt.Errorf("failed to read padding at %d: %v", i, err)
		}

		// Reconstruct corners
		if int(polygonIndex) < len(mesh.polygons) {
			poly := mesh.polygons[polygonIndex]
			area.Corners = make([]r3.Vector, len(poly.indices))
			for k, idx := range poly.indices {
				if int(idx) < len(mesh.corners) {
					area.Corners[k] = mesh.corners[idx]
				}
			}
			// Calculate AABB for compatibility
			minX, minY, minZ := 1e9, 1e9, 1e9
			maxX, maxY, maxZ := -1e9, -1e9, -1e9
			for _, c := range area.Corners {
				if c.X < minX {
					minX = c.X
				}
				if c.Y < minY {
					minY = c.Y
				}
				if c.Z < minZ {
					minZ = c.Z
				}
				if c.X > maxX {
					maxX = c.X
				}
				if c.Y > maxY {
					maxY = c.Y
				}
				if c.Z > maxZ {
					maxZ = c.Z
				}
			}
			area.NW = r3.Vector{X: minX, Y: minY, Z: minZ}
			area.SE = r3.Vector{X: maxX, Y: maxY, Z: maxZ}
		}

		// Skip UnkArea (Already read)
		// var unkArea uint32
		// if err := binary.Read(r, binary.LittleEndian, &unkArea); err != nil {
		// 	return nil, fmt.Errorf("failed to read unkArea at %d: %v", i, err)
		// }

		if i < 5 {
			fmt.Printf("Area %d: ID=%d Flags=%x Hull=%d Poly=%d Unk=%d\n", i, area.ID, flags, hullIndex, polygonIndex, unkArea)
		}

		// Connections (Per Edge)
		edgeCount := len(area.Corners)
		for k := 0; k < edgeCount; k++ {
			var connCount uint32
			if err := binary.Read(r, order, &connCount); err != nil {
				return nil, fmt.Errorf("failed to read connCount at %d edge %d: %v", i, k, err)
			}
			for j := 0; j < int(connCount); j++ {
				var targetID uint32
				if err := binary.Read(r, order, &targetID); err != nil {
					return nil, fmt.Errorf("failed to read targetID at %d edge %d conn %d: %v", i, k, j, err)
				}
			}
		}

		// Skip Legacy Data (2 bytes in v35?)
		// Analysis of dump shows 2 bytes of zeros before LadderAboveCount
		legacySkip := make([]byte, 2)
		if err := binary.Read(r, binary.LittleEndian, &legacySkip); err != nil {
			return nil, fmt.Errorf("failed to read legacySkip at %d: %v", i, err)
		}

		// Ladders Above
		var ladderAboveCount uint32
		if err := binary.Read(r, order, &ladderAboveCount); err != nil {
			return nil, fmt.Errorf("failed to read ladderAboveCount at %d: %v", i, err)
		}
		for k := 0; k < int(ladderAboveCount); k++ {
			var id uint32
			if err := binary.Read(r, order, &id); err != nil {
				return nil, fmt.Errorf("failed to read ladderAboveID at %d: %v", i, err)
			}
		}

		// Ladders Below
		var ladderBelowCount uint32
		if err := binary.Read(r, order, &ladderBelowCount); err != nil {
			return nil, fmt.Errorf("failed to read ladderBelowCount at %d: %v", i, err)
		}
		for k := 0; k < int(ladderBelowCount); k++ {
			var id uint32
			if err := binary.Read(r, order, &id); err != nil {
				return nil, fmt.Errorf("failed to read ladderBelowID at %d: %v", i, err)
			}
		}

		// PlaceID (Finally!)
		if err := binary.Read(r, order, &area.PlaceID); err != nil {
			return nil, fmt.Errorf("failed to read PlaceID at %d: %v", i, err)
		}

		// Earliest Occupier (2 teams)
		for team := 0; team < 2; team++ {
			var time float32
			if err := binary.Read(r, order, &time); err != nil {
				return nil, fmt.Errorf("failed to read time at %d team %d: %v", i, team, err)
			}
		}

		// DEBUG: Read 4 extra bytes to fix alignment?
		// It seems we are 4 bytes short in Area 0 reading.
		if i == 0 {
			dummy := make([]byte, 4)
			r.Read(dummy)
		}

		mesh.Areas[i] = area
	}

	return mesh, nil
}

// GetNearestArea returns the navigation area closest to the given position
func (nm *NavMesh) GetNearestArea(pos r3.Vector) *NavArea {
	var bestArea *NavArea
	minZDist := 10000.0

	for i := range nm.Areas {
		area := &nm.Areas[i]
		// Check if point is inside the polygon (2D)
		if isPointInPolygon(pos, area.Corners) {
			// Check Z distance
			zDist := pos.Z - area.Center.Z
			if len(area.Corners) > 0 {
				// Use average Z of corners if center is not reliable
				avgZ := 0.0
				for _, c := range area.Corners {
					avgZ += c.Z
				}
				avgZ /= float64(len(area.Corners))
				zDist = pos.Z - avgZ
			}

			if zDist < 0 {
				zDist = -zDist
			}

			if zDist < minZDist {
				minZDist = zDist
				bestArea = area
			}
		}
	}
	return bestArea
}

// GetPlaceName returns the place name for a given position
func (nm *NavMesh) GetPlaceName(pos r3.Vector) string {
	area := nm.GetNearestArea(pos)
	if area != nil {
		if area.PlaceID > 0 && int(area.PlaceID-1) < len(nm.Places) {
			return nm.Places[area.PlaceID-1]
		}
	}
	return ""
}

func isPointInPolygon(p r3.Vector, poly []r3.Vector) bool {
	inside := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		if (poly[i].Y > p.Y) != (poly[j].Y > p.Y) &&
			p.X < (poly[j].X-poly[i].X)*(p.Y-poly[i].Y)/(poly[j].Y-poly[i].Y)+poly[i].X {
			inside = !inside
		}
		j = i
	}
	return inside
}
