// Package visualcontact contains the deterministic geometry primitives shared
// by replay visibility analysis and the causal observed-contact ledger.
package visualcontact

import (
	"math"

	"github.com/golang/geo/r3"
)

const (
	CanonicalAspectRatio  = 16.0 / 9.0
	SourceReferenceAspect = 4.0 / 3.0

	FOVMethodVersion     = "rectangular_frustum_16_9_from_source_4_3@1"
	SmokeMethodVersion   = "segment_to_smoke_cylinder@1"
	RaycastMethodVersion = "map_mesh_segment_raycast@1"
)

// Frustum is a deterministic rectangular view frustum. Circular can be set
// for scoped sniper optics, matching the existing reaction analyzer behavior.
type Frustum struct {
	forward           r3.Vector
	right             r3.Vector
	up                r3.Vector
	tanHalfHorizontal float64
	tanHalfVertical   float64
	circular          bool
}

// NewFrustum converts the Source reference horizontal FOV to the canonical
// 16:9 viewport used by the existing visibility analyzer.
func NewFrustum(pitch, yaw, referenceHorizontalFOV float64, circular bool) Frustum {
	verticalFOV := 2 * math.Atan(math.Tan(degreesToRadians(referenceHorizontalFOV)/2)/SourceReferenceAspect)
	horizontalFOV := 2 * math.Atan(math.Tan(verticalFOV/2)*CanonicalAspectRatio)
	forward := AnglesToVector(pitch, yaw)
	yawRadians := degreesToRadians(yaw)
	right := r3.Vector{X: -math.Sin(yawRadians), Y: math.Cos(yawRadians)}

	return Frustum{
		forward:           forward,
		right:             right,
		up:                forward.Cross(right),
		tanHalfHorizontal: math.Tan(horizontalFOV / 2),
		tanHalfVertical:   math.Tan(verticalFOV / 2),
		circular:          circular,
	}
}

// Contains reports whether target is inside the view frustum from origin.
func (frustum Frustum) Contains(origin, target r3.Vector) bool {
	toTarget := target.Sub(origin)
	depth := frustum.forward.Dot(toTarget)
	if depth <= 0 {
		return false
	}

	horizontal := math.Abs(frustum.right.Dot(toTarget) / depth)
	vertical := math.Abs(frustum.up.Dot(toTarget) / depth)
	if horizontal > frustum.tanHalfHorizontal || vertical > frustum.tanHalfVertical {
		return false
	}
	if !frustum.circular {
		return true
	}
	horizontal /= frustum.tanHalfHorizontal
	vertical /= frustum.tanHalfVertical
	return horizontal*horizontal+vertical*vertical <= 1
}

// AnglesToVector converts Source pitch/yaw degrees to a unit direction.
func AnglesToVector(pitch, yaw float64) r3.Vector {
	pitchRadians := degreesToRadians(pitch)
	yawRadians := degreesToRadians(yaw)
	return r3.Vector{
		X: math.Cos(pitchRadians) * math.Cos(yawRadians),
		Y: math.Cos(pitchRadians) * math.Sin(yawRadians),
		Z: -math.Sin(pitchRadians),
	}
}

// DistancePointToSegment returns the shortest 3D distance to a finite segment.
func DistancePointToSegment(point, start, end r3.Vector) float64 {
	segment := end.Sub(start)
	denominator := segment.Dot(segment)
	if denominator == 0 {
		return point.Sub(start).Norm()
	}
	projection := point.Sub(start).Dot(segment) / denominator
	projection = math.Max(0, math.Min(1, projection))
	nearest := start.Add(segment.Mul(projection))
	return point.Sub(nearest).Norm()
}

// DistancePointToSegment2D treats smoke as a vertical cylinder because replay
// active-effect snapshots currently publish X/Y and radius, but not Z.
func DistancePointToSegment2D(point, start, end r3.Vector) float64 {
	point.Z = 0
	start.Z = 0
	end.Z = 0
	return DistancePointToSegment(point, start, end)
}

func IsFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func IsFiniteVector(value r3.Vector) bool {
	return IsFinite(value.X) && IsFinite(value.Y) && IsFinite(value.Z)
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
}
