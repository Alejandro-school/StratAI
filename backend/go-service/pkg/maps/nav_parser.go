package maps

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/golang/geo/r3"
)

// NavArea represents a navigation area
type NavArea struct {
	ID          uint32
	NW          r3.Vector // Used for v16 (AABB)
	SE          r3.Vector // Used for v16 (AABB)
	PlaceID     uint16
	Connections []uint32 // Directed adjacent NAV-area IDs (ladder IDs are kept separate in the file format)

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
	// SubVersion
	var subVersion uint32
	if err := binary.Read(f, binary.LittleEndian, &subVersion); err != nil {
		return nil, err
	}

	// Source 1 stores the BSP size here. Source 2 uses this word for flags,
	// including the analyzed bit. Neither value is needed by this package.
	var headerWord uint32
	if err := binary.Read(f, binary.LittleEndian, &headerWord); err != nil {
		return nil, err
	}

	// --- VERSION SPECIFIC PARSING ---

	if version >= 31 && version <= 35 {
		// CS2 (Source 2) Logic
		return parseSource2(f, mesh, version)
	}
	if version != 16 {
		return nil, fmt.Errorf("unsupported nav version: %d", version)
	}

	// IsAnalyzed (Only for v16 / Source 1)
	var isAnalyzed uint8
	if err := binary.Read(f, binary.LittleEndian, &isAnalyzed); err != nil {
		return nil, err
	}

	// PlaceCount
	var placeCount uint16
	if err := binary.Read(f, binary.LittleEndian, &placeCount); err != nil {
		return nil, err
	}

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
		area.Center = centerOfNavCorners(area.Corners)

		// Connections (4 directions: N, E, S, W)
		for dir := 0; dir < 4; dir++ {
			var count uint32
			if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
				return nil, fmt.Errorf("failed to read connection count for area %d: %v", area.ID, err)
			}
			for k := 0; k < int(count); k++ {
				var connectID uint32
				if err := binary.Read(r, binary.LittleEndian, &connectID); err != nil {
					return nil, fmt.Errorf("failed to read connection for area %d: %v", area.ID, err)
				}
				area.Connections = append(area.Connections, connectID)
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
			if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
				return nil, fmt.Errorf("failed to read ladder connection count for area %d: %v", area.ID, err)
			}
			for k := 0; k < int(count); k++ {
				var connectID uint32
				if err := binary.Read(r, binary.LittleEndian, &connectID); err != nil {
					return nil, fmt.Errorf("failed to read ladder connection for area %d: %v", area.ID, err)
				}
				area.Connections = append(area.Connections, connectID)
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

const (
	maxSource2CollectionEntries = uint32(1 << 24)
	maxSource2StringBytes       = 1 << 20
)

type source2Reader struct {
	reader       io.Reader
	remaining    uint64
	hasRemaining bool
}

func newSource2Reader(r io.Reader) *source2Reader {
	remaining, ok := source2RemainingBytes(r)
	return &source2Reader{reader: r, remaining: remaining, hasRemaining: ok}
}

func (r *source2Reader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if r.hasRemaining {
		if uint64(n) > r.remaining {
			r.remaining = 0
		} else {
			r.remaining -= uint64(n)
		}
	}
	return n, err
}

func (r *source2Reader) read(value any, label string) error {
	if err := binary.Read(r, binary.LittleEndian, value); err != nil {
		return fmt.Errorf("failed to read %s: %w", label, err)
	}
	return nil
}

func (r *source2Reader) readCount(label string, minimumElementBytes uint64) (uint32, error) {
	var count uint32
	if err := r.read(&count, label); err != nil {
		return 0, err
	}
	if count > maxSource2CollectionEntries {
		return 0, fmt.Errorf("invalid %s %d: exceeds safety limit %d", label, count, maxSource2CollectionEntries)
	}
	if r.hasRemaining && minimumElementBytes > 0 && uint64(count) > r.remaining/minimumElementBytes {
		return 0, fmt.Errorf("invalid %s %d: requires at least %d bytes but only %d remain", label, count, uint64(count)*minimumElementBytes, r.remaining)
	}
	return count, nil
}

func source2RemainingBytes(r io.Reader) (uint64, bool) {
	if reader, ok := r.(interface{ Len() int }); ok {
		remaining := reader.Len()
		if remaining >= 0 {
			return uint64(remaining), true
		}
	}

	seeker, ok := r.(io.Seeker)
	if !ok {
		return 0, false
	}
	position, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, false
	}
	end, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, false
	}
	if _, err := seeker.Seek(position, io.SeekStart); err != nil || end < position {
		return 0, false
	}
	return uint64(end - position), true
}

func (r *source2Reader) skipCString(label string) error {
	var octet [1]byte
	for i := 0; i < maxSource2StringBytes; i++ {
		if _, err := io.ReadFull(r, octet[:]); err != nil {
			return fmt.Errorf("failed to read %s: %w", label, err)
		}
		if octet[0] == 0 {
			return nil
		}
	}
	return fmt.Errorf("invalid %s: exceeds safety limit %d", label, maxSource2StringBytes)
}

func parseSource2(input io.Reader, mesh *NavMesh, version uint32) (*NavMesh, error) {
	if version < 31 || version > 35 {
		return nil, fmt.Errorf("unsupported Source 2 nav version: %d", version)
	}
	r := newSource2Reader(input)

	cornerCount, err := r.readCount("corner count", 12)
	if err != nil {
		return nil, err
	}
	mesh.corners = make([]r3.Vector, int(cornerCount))
	for i := range mesh.corners {
		var x, y, z float32
		if err := r.read(&x, fmt.Sprintf("corner X at %d", i)); err != nil {
			return nil, err
		}
		if err := r.read(&y, fmt.Sprintf("corner Y at %d", i)); err != nil {
			return nil, err
		}
		if err := r.read(&z, fmt.Sprintf("corner Z at %d", i)); err != nil {
			return nil, err
		}
		if !source2Finite(float64(x)) || !source2Finite(float64(y)) || !source2Finite(float64(z)) {
			return nil, fmt.Errorf("invalid non-finite corner at %d", i)
		}
		mesh.corners[i] = r3.Vector{X: float64(x), Y: float64(y), Z: float64(z)}
	}

	minimumPolygonBytes := uint64(1)
	if version >= 35 {
		minimumPolygonBytes += 4
	}
	polygonCount, err := r.readCount("polygon count", minimumPolygonBytes)
	if err != nil {
		return nil, err
	}
	mesh.polygons = make([]navPolygon, int(polygonCount))
	for i := range mesh.polygons {
		var polygonCornerCount uint8
		if err := r.read(&polygonCornerCount, fmt.Sprintf("polygon corner count at %d", i)); err != nil {
			return nil, err
		}
		if polygonCornerCount < 3 {
			return nil, fmt.Errorf("invalid polygon corner count %d at %d", polygonCornerCount, i)
		}
		if r.hasRemaining && uint64(polygonCornerCount)*4 > r.remaining {
			return nil, fmt.Errorf("invalid polygon corner count %d at %d: requires %d bytes but only %d remain", polygonCornerCount, i, uint64(polygonCornerCount)*4, r.remaining)
		}

		indices := make([]uint32, int(polygonCornerCount))
		if err := r.read(&indices, fmt.Sprintf("polygon indices at %d", i)); err != nil {
			return nil, err
		}
		for corner, index := range indices {
			if index >= cornerCount {
				return nil, fmt.Errorf("invalid corner index %d at polygon %d corner %d (corner count %d)", index, i, corner, cornerCount)
			}
		}
		mesh.polygons[i] = navPolygon{indices: indices}

		if version >= 35 {
			var movableMeshID uint32
			if err := r.read(&movableMeshID, fmt.Sprintf("movable mesh id at polygon %d", i)); err != nil {
				return nil, err
			}
		}
	}

	if version >= 32 {
		var unknown uint32
		if err := r.read(&unknown, "Source 2 global unknown"); err != nil {
			return nil, err
		}
	}
	if version >= 35 {
		movableMeshCount, err := r.readCount("movable mesh count", 49)
		if err != nil {
			return nil, err
		}
		for i := uint32(0); i < movableMeshCount; i++ {
			if err := r.skipCString(fmt.Sprintf("movable mesh id at %d", i)); err != nil {
				return nil, err
			}
			var referenceTransform [48]byte
			if _, err := io.ReadFull(r, referenceTransform[:]); err != nil {
				return nil, fmt.Errorf("failed to read movable mesh transform at %d: %w", i, err)
			}
		}
	}

	areaCount, err := r.readCount("area count", 34)
	if err != nil {
		return nil, err
	}
	mesh.Areas = make([]NavArea, int(areaCount))
	areaIDs := make(map[uint32]struct{}, int(areaCount))
	for i := range mesh.Areas {
		area := NavArea{}
		if err := r.read(&area.ID, fmt.Sprintf("area ID at %d", i)); err != nil {
			return nil, err
		}
		if area.ID == 0 {
			return nil, fmt.Errorf("invalid zero area ID at %d", i)
		}
		if _, duplicate := areaIDs[area.ID]; duplicate {
			return nil, fmt.Errorf("duplicate area ID %d at %d", area.ID, i)
		}
		areaIDs[area.ID] = struct{}{}

		var attributeFlags int64
		if err := r.read(&attributeFlags, fmt.Sprintf("attribute flags at area %d", i)); err != nil {
			return nil, err
		}
		var hullIndex uint8
		if err := r.read(&hullIndex, fmt.Sprintf("hull index at area %d", i)); err != nil {
			return nil, err
		}
		var polygonIndex uint32
		if err := r.read(&polygonIndex, fmt.Sprintf("polygon index at area %d", i)); err != nil {
			return nil, err
		}
		if polygonIndex >= polygonCount {
			return nil, fmt.Errorf("invalid polygon index %d at area %d (polygon count %d)", polygonIndex, i, polygonCount)
		}
		var unknown float32
		if err := r.read(&unknown, fmt.Sprintf("unknown float at area %d", i)); err != nil {
			return nil, err
		}

		polygon := mesh.polygons[polygonIndex]
		area.Corners = make([]r3.Vector, len(polygon.indices))
		for corner, index := range polygon.indices {
			area.Corners[corner] = mesh.corners[index]
		}
		setNavAreaGeometry(&area)

		for edge := range area.Corners {
			connectionCount, err := r.readCount(fmt.Sprintf("connection count at area %d edge %d", i, edge), 8)
			if err != nil {
				return nil, err
			}
			for connection := uint32(0); connection < connectionCount; connection++ {
				var targetID uint32
				if err := r.read(&targetID, fmt.Sprintf("target area ID at area %d edge %d connection %d", i, edge, connection)); err != nil {
					return nil, err
				}
				var edgeID uint32
				if err := r.read(&edgeID, fmt.Sprintf("target edge ID at area %d edge %d connection %d", i, edge, connection)); err != nil {
					return nil, err
				}
				area.Connections = append(area.Connections, targetID)
			}
		}

		var legacyHidingSpotCount uint8
		if err := r.read(&legacyHidingSpotCount, fmt.Sprintf("legacy hiding spot count at area %d", i)); err != nil {
			return nil, err
		}
		var legacyEncounterCount uint32
		if err := r.read(&legacyEncounterCount, fmt.Sprintf("legacy encounter count at area %d", i)); err != nil {
			return nil, err
		}
		if legacyHidingSpotCount != 0 || legacyEncounterCount != 0 {
			return nil, fmt.Errorf("unsupported legacy data at area %d (hiding spots %d, encounters %d)", i, legacyHidingSpotCount, legacyEncounterCount)
		}

		for _, direction := range []string{"above", "below"} {
			ladderCount, err := r.readCount(fmt.Sprintf("ladder %s count at area %d", direction, i), 4)
			if err != nil {
				return nil, err
			}
			for ladder := uint32(0); ladder < ladderCount; ladder++ {
				var ladderID uint32
				if err := r.read(&ladderID, fmt.Sprintf("ladder %s ID at area %d index %d", direction, i, ladder)); err != nil {
					return nil, err
				}
			}
		}

		mesh.Areas[i] = area
	}

	for _, area := range mesh.Areas {
		for _, targetID := range area.Connections {
			if _, exists := areaIDs[targetID]; !exists {
				return nil, fmt.Errorf("area %d references missing area %d", area.ID, targetID)
			}
		}
	}

	return mesh, nil
}

func source2Finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func setNavAreaGeometry(area *NavArea) {
	if len(area.Corners) == 0 {
		return
	}
	min := area.Corners[0]
	max := area.Corners[0]
	for _, corner := range area.Corners[1:] {
		min.X = math.Min(min.X, corner.X)
		min.Y = math.Min(min.Y, corner.Y)
		min.Z = math.Min(min.Z, corner.Z)
		max.X = math.Max(max.X, corner.X)
		max.Y = math.Max(max.Y, corner.Y)
		max.Z = math.Max(max.Z, corner.Z)
	}
	area.NW = min
	area.SE = max
	area.Center = centerOfNavCorners(area.Corners)
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

func centerOfNavCorners(corners []r3.Vector) r3.Vector {
	if len(corners) == 0 {
		return r3.Vector{}
	}
	var center r3.Vector
	for _, corner := range corners {
		center = center.Add(corner)
	}
	return center.Mul(1 / float64(len(corners)))
}
