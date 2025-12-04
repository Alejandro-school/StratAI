package maps

import (
	"fmt"
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
}

// MapManager handles loading maps and performing visibility checks
type MapManager struct {
	mapsDir     string
	currentMesh *geometry.Mesh // Only support GLTF meshes for CS2
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
	viewX := shooter.ViewDirectionX()
	viewY := shooter.ViewDirectionY()
	viewDir := r3.Vector{X: float64(viewX), Y: float64(viewY), Z: 0} // Simplified 2D check for FOV is usually enough

	// Normalize
	toEnemy2D := r3.Vector{X: toEnemy.X, Y: toEnemy.Y, Z: 0}.Normalize()
	viewDir = viewDir.Normalize()

	// Dot product
	dot := viewDir.Dot(toEnemy2D)
	// acos(dot) gives angle.
	// 90 degree FOV means +/- 45 degrees from center.
	// cos(45°) = 0.707 - usar valor exacto para CS2
	if dot < 0.707 { // 90° FOV total (real CS2 FOV)
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
