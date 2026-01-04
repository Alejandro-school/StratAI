package maps

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cs2-demo-service/pkg/geometry"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
)

// VisibilityChecker defines the interface for checking visibility between two points
type VisibilityChecker interface {
	// IsVisible checks if there is a clear line of sight between start and end
	IsVisible(start, end r3.Vector) bool
	// IsLoaded returns true if a map is currently loaded
	IsLoaded() bool
	// GetCallout returns the callout name for a given position
	GetCallout(pos r3.Vector) string
	// RayCast returns the distance to the first intersection and the surface normal, or -1 if none
	RayCast(origin, dir r3.Vector, maxDist float64) (float64, r3.Vector)
}

// MapManager handles loading maps and performing visibility checks
type MapManager struct {
	mapsDir     string
	currentMesh *geometry.Mesh // Only support GLTF meshes for CS2
	currentNav  *NavMesh       // Navigation Mesh for callouts
	callouts    []Callout      // List of named callouts (from places.json)
	mapName     string
	mutex       sync.RWMutex
	useFallback bool // If true, use heuristic (FOV/Smoke) only
}

// NewMapManager creates a new map manager
func NewMapManager(mapsDir string) *MapManager {
	return &MapManager{
		mapsDir: mapsDir,
	}
}

// GetCallout returns the callout name for a given position
func (m *MapManager) GetCallout(pos r3.Vector) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Priority 1: Check JSON callouts (CS2 specific)
	if len(m.callouts) > 0 {
		if name := FindCallout(pos, m.callouts); name != "" {
			return name
		}
	}

	// Priority 2: Check Nav Mesh (Legacy/Fallback)
	if m.currentNav != nil {
		return m.currentNav.GetPlaceName(pos)
	}
	return ""
}

// RayCast returns the distance to the first intersection and the surface normal, or -1 if none
func (m *MapManager) RayCast(origin, dir r3.Vector, maxDist float64) (float64, r3.Vector) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.currentMesh != nil {
		return m.currentMesh.RayCast(origin, dir, maxDist)
	}
	return -1, r3.Vector{}
}

// IsLoaded returns true if a map is currently loaded
func (m *MapManager) IsLoaded() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.currentMesh != nil && !m.useFallback
}

// LoadMap attempts to load a map file for the given map name
// Only supports .gltf (CS2 Physics Hull)
func (m *MapManager) LoadMap(mapName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.mapName == mapName && m.currentMesh != nil {
		return nil // Already loaded
	}

	// Sanitize map name (remove path, extension)
	baseName := filepath.Base(mapName)
	// Remove extension if present (e.g. de_mirage.bsp -> de_mirage)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// 1. Try loading .gltf (CS2)
	// Priority 1: maps/mapName/mapName_physics.gltf (Source 2 Viewer export)
	gltfPath := filepath.Join(m.mapsDir, baseName, baseName+"_physics.gltf")

	if _, err := os.Stat(gltfPath); err != nil {
		// Priority 2: maps/mapName/mapName.gltf
		gltfPath = filepath.Join(m.mapsDir, baseName, baseName+".gltf")
	}

	if _, err := os.Stat(gltfPath); err != nil {
		// Priority 3: maps/mapName.gltf (Flat structure)
		gltfPath = filepath.Join(m.mapsDir, baseName+".gltf")
	}

	if _, err := os.Stat(gltfPath); err == nil {
		fmt.Printf("Loading CS2 Mesh (GLTF): %s\n", gltfPath)
		mesh, err := geometry.LoadGLTF(gltfPath)
		if err == nil {
			m.currentMesh = mesh
			m.mapName = mapName
			m.useFallback = false
			fmt.Printf("CS2 Mesh loaded successfully: %s (%d triangles)\n", mapName, len(mesh.Triangles))

			// Try loading .nav file
			navPath := filepath.Join(m.mapsDir, baseName, baseName+".nav")
			if _, err := os.Stat(navPath); err == nil {
				fmt.Printf("Loading Nav Mesh: %s\n", navPath)
				nav, err := LoadNavMesh(navPath)
				if err == nil {
					m.currentNav = nav
					fmt.Printf("Nav Mesh loaded successfully: %d areas, %d places\n", len(nav.Areas), len(nav.Places))
				} else {
					// v36 parsing not fully supported yet - fallback to demo's LastPlaceName() will be used
					fmt.Printf("Nav Mesh v36 not supported (using demo fallback): %v\n", err)
				}
			} else {
				fmt.Printf("Nav file not found: %s\n", navPath)
			}

			// Try loading places.json (CS2 Callouts)
			// Priority 1: maps/mapName/mapName_places.json
			placesPath := filepath.Join(m.mapsDir, baseName, baseName+"_places.json")
			if _, err := os.Stat(placesPath); err != nil {
				// Priority 2: maps/mapName_places.json
				placesPath = filepath.Join(m.mapsDir, baseName+"_places.json")
			}

			if _, err := os.Stat(placesPath); err == nil {
				fmt.Printf("Loading Callouts JSON: %s\n", placesPath)
				callouts, err := LoadCallouts(placesPath)
				if err == nil {
					m.callouts = callouts
					fmt.Printf("Callouts loaded successfully: %d places\n", len(callouts))

					// Map seeds to NavMesh if available
					if m.currentNav != nil {
						m.MapCalloutsToNavMesh(callouts)
					}
				} else {
					fmt.Printf("Failed to load Callouts JSON: %v\n", err)
				}
			} else {
				// No places.json - will use demo's LastPlaceName() as fallback
				// fmt.Printf("Callouts file not found: %s (using demo fallback)\n", placesPath)
			}

			return nil
		}
		fmt.Printf("Failed to load GLTF: %v\n", err)
	}

	// Fallback
	m.useFallback = true
	m.currentMesh = nil
	m.mapName = mapName
	fmt.Printf("Map file not found. Using Heuristic Mode.\n")
	return fmt.Errorf("map file not found")
}

// IsVisible performs the visibility check
func (m *MapManager) IsVisible(start, end r3.Vector) bool {
	// If we have a map loaded (Mesh), use Raycasting
	if m.IsLoaded() {
		return m.TraceRay(start, end)
	}

	// Fallback: Always return true (let the heuristic handle it)
	return true
}

// TraceRay checks if a ray from start to end is obstructed by the map geometry
func (m *MapManager) TraceRay(start, end r3.Vector) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.currentMesh == nil {
		return true // Should not happen if IsLoaded() is checked, but safe default
	}

	// RayIntersects returns true if BLOCKED
	// So IsVisible = !RayIntersects
	return !m.currentMesh.RayIntersects(start, end)
}

// HeuristicIsVisible implements the "Option 2" logic: FOV + Smoke + Flash
// This is static and doesn't need the map file
func HeuristicIsVisible(shooter, enemy *common.Player, activeSmokes []r3.Vector) bool {
	// 1. Basic Radar Check (IsSpottedBy) - REMOVED
	// We now use Raycasting for wall checks, so we don't rely on IsSpottedBy which can be flaky in CS2.
	// if !enemy.IsSpottedBy(shooter) {
	// 	return false
	// }

	// 2. Flashbang Check
	// Flash duration > 0.3s afecta significativamente la visión
	// Completamente ciego si > 1.5s (full flash)
	if shooter.FlashDuration > 1.5 {
		return false // Completamente flasheado
	}
	// Si tiene flash residual (0.3-1.5s), permitir pero será marcado en metadata

	// 3. FOV Check
	// Calculate angle between shooter's view vector and vector to enemy
	// PositionEyes() is not supported for CS2 yet, so we approximate:
	// Eye position = Position + (0, 0, 64) (Standing eye height)
	shooterPos := shooter.Position()
	shooterPos.Z += 64

	// enemy.Position() returns feet. Add Z for center/chest approx.
	enemyPos := enemy.Position()
	enemyPos.Z += 40 // Approx chest height

	// Vector to enemy
	toEnemy := enemyPos.Sub(shooterPos)

	// Shooter view vector
	// ViewDirectionX is Pitch, ViewDirectionY is Yaw
	// We only care about Yaw for 2D FOV check
	yaw := float64(shooter.ViewDirectionY())
	yawRad := yaw * math.Pi / 180.0

	// Calculate 2D view direction vector from Yaw
	viewDir := r3.Vector{
		X: math.Cos(yawRad),
		Y: math.Sin(yawRad),
		Z: 0,
	}

	// Normalize
	toEnemy2D := r3.Vector{X: toEnemy.X, Y: toEnemy.Y, Z: 0}.Normalize()
	// viewDir is already normalized by definition (cos^2 + sin^2 = 1)

	// Dot product
	dot := viewDir.Dot(toEnemy2D)
	// acos(dot) gives angle.
	// 90 degree FOV means +/- 45 degrees from center. cos(45°) = 0.707
	// 16:9 FOV means +/- 53 degrees from center. cos(53°) = 0.601
	// Usamos 0.6 para dar soporte a 16:9. Es mejor un falso positivo (decir que lo ve cuando no)
	// que un falso negativo (decir que no lo ve, y que luego tenga 0ms de reaction time).
	if dot < 0.6 {
		return false
	}

	// 4. Smoke Check
	// Simple sphere check: Is there a smoke grenade center close to the line of sight?
	// Line segment: shooterPos to enemyPos
	for _, smokePos := range activeSmokes {
		if distancePointToSegment(smokePos, shooterPos, enemyPos) < 140.0 { // 140 units is approx smoke radius
			return false
		}
	}

	return true
}

func distancePointToSegment(p, a, b r3.Vector) float64 {
	ab := b.Sub(a)
	ap := p.Sub(a)

	// Project p onto ab, computing parameterized position t
	t := ap.Dot(ab) / ab.Dot(ab)

	// Clamp t to segment [0, 1]
	if t < 0.0 {
		t = 0.0
	} else if t > 1.0 {
		t = 1.0
	}

	// Compute nearest point on segment
	nearest := a.Add(ab.Mul(t))

	// Distance
	return p.Sub(nearest).Norm()
}

// MapCalloutsToNavMesh maps point-based callouts to NavMesh PlaceIDs
func (m *MapManager) MapCalloutsToNavMesh(callouts []Callout) {
	if m.currentNav == nil {
		return
	}

	// 1. Find Max PlaceID
	maxPlaceID := uint16(0)
	for i := range m.currentNav.Areas {
		if m.currentNav.Areas[i].PlaceID > maxPlaceID {
			maxPlaceID = m.currentNav.Areas[i].PlaceID
		}
	}

	// 2. Initialize Places array
	if len(m.currentNav.Places) < int(maxPlaceID) {
		newPlaces := make([]string, maxPlaceID)
		copy(newPlaces, m.currentNav.Places)
		m.currentNav.Places = newPlaces
	}

	// 3. Map Seeds to PlaceIDs
	count := 0
	for _, c := range callouts {
		// Only process point seeds (where Min/Max are nil)
		if c.Min == nil && c.Max == nil {
			pos := r3.Vector{X: c.X, Y: c.Y, Z: c.Z}
			area := m.currentNav.GetNearestArea(pos)
			if area != nil && area.PlaceID > 0 {
				// PlaceID is 1-based
				idx := int(area.PlaceID) - 1
				if idx < len(m.currentNav.Places) {
					m.currentNav.Places[idx] = c.Name
					count++
				}
			}
		}
	}
	fmt.Printf("Mapped %d callouts to NavMesh PlaceIDs\n", count)
}
