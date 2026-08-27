package analyzers

import (
	"cs2-demo-service/pkg/playerstate"
	"testing"

	"github.com/golang/geo/r3"
)

func TestClassifySprayMovementKeepsUnknownSeparateFromStationary(t *testing.T) {
	wasMoving, available := classifySprayMovement(playerstate.MotionEstimate{})
	if wasMoving || available {
		t.Fatal("an unavailable motion estimate must remain unknown")
	}

	wasMoving, available = classifySprayMovement(playerstate.MotionEstimate{
		Vector:    r3.Vector{},
		Available: true,
	})
	if wasMoving || !available {
		t.Fatal("an observed zero velocity must be classified as stationary")
	}

	wasMoving, available = classifySprayMovement(playerstate.MotionEstimate{
		Vector:    r3.Vector{Z: 900},
		Available: true,
	})
	if wasMoving || !available {
		t.Fatal("spray movement must use horizontal velocity and ignore vertical speed")
	}

	wasMoving, available = classifySprayMovement(playerstate.MotionEstimate{
		Vector:    r3.Vector{X: 40, Y: 40},
		Available: true,
	})
	if !wasMoving || !available {
		t.Fatal("horizontal speed above the threshold must be classified as moving")
	}
}

func TestClassifySprayQuality(t *testing.T) {
	tests := []struct {
		name             string
		hits, shots      int
		avgPitch, avgYaw float32
		want             string
	}{
		{name: "excellent", hits: 7, shots: 10, avgPitch: 4, avgYaw: 4, want: "excellent"},
		{name: "good", hits: 5, shots: 10, avgPitch: 8, avgYaw: 8, want: "good"},
		{name: "fair", hits: 3, shots: 10, avgPitch: 12, avgYaw: 12, want: "fair"},
		{name: "poor", hits: 2, shots: 10, avgPitch: 3, avgYaw: 3, want: "poor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySprayQuality(tt.hits, tt.shots, tt.avgPitch, tt.avgYaw)
			if got != tt.want {
				t.Fatalf("classifySprayQuality() = %q, want %q", got, tt.want)
			}
		})
	}
}
