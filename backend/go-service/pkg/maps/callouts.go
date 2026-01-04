package maps

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/golang/geo/r3"
)

// Callout represents a named area in the map defined by a bounding box or a point
type Callout struct {
	Name string `json:"name"`
	// Bounding Box (Optional)
	Min *Vector3 `json:"min,omitempty"`
	Max *Vector3 `json:"max,omitempty"`
	// Point Seed (Optional, for .vmap extraction)
	X float64 `json:"x,omitempty"`
	Y float64 `json:"y,omitempty"`
	Z float64 `json:"z,omitempty"`
}

// Vector3 is a helper struct for JSON unmarshalling
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Contains checks if a point is inside the callout's bounding box
func (c *Callout) Contains(pos r3.Vector) bool {
	if c.Min == nil || c.Max == nil {
		return false
	}
	return pos.X >= c.Min.X && pos.X <= c.Max.X &&
		pos.Y >= c.Min.Y && pos.Y <= c.Max.Y &&
		pos.Z >= c.Min.Z && pos.Z <= c.Max.Z
}

// LoadCallouts loads callouts from a JSON file
func LoadCallouts(filePath string) ([]Callout, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var callouts []Callout
	if err := json.NewDecoder(file).Decode(&callouts); err != nil {
		return nil, fmt.Errorf("failed to decode callouts JSON: %w", err)
	}

	return callouts, nil
}

// FindCallout returns the name of the callout containing the position, or empty string
func FindCallout(pos r3.Vector, callouts []Callout) string {
	// 1. Check Bounding Boxes (Priority)
	// TODO: Optimize this with a spatial index (Octree/Grid) if we have many callouts
	for _, c := range callouts {
		if c.Contains(pos) {
			return c.Name
		}
	}

	// 2. Check Nearest Point Seed (Fallback for VMap extraction)
	// Only if we didn't find a bounding box match
	var bestName string
	minDist := 500.0 // Threshold units (approx 12 meters)

	for _, c := range callouts {
		// Skip if it's a bounding box (already checked)
		if c.Min != nil {
			continue
		}

		// Calculate distance
		dx := c.X - pos.X
		dy := c.Y - pos.Y
		dz := c.Z - pos.Z
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

		if dist < minDist {
			minDist = dist
			bestName = c.Name
		}
	}

	return bestName
}
