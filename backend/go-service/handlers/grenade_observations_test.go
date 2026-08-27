package handlers

import (
	"math"
	"testing"

	"cs2-demo-service/pkg/utility"
)

func TestNormalizedThrowViewUsesSignedYaw(t *testing.T) {
	observation := normalizedThrowView(270, 15)
	if observation.Status != utility.AvailabilityObserved || observation.Yaw != -90 || observation.Pitch != 15 {
		t.Fatalf("unexpected normalized view: %+v", observation)
	}
}

func TestNormalizedThrowViewRejectsInvalidAngles(t *testing.T) {
	for _, input := range []struct {
		yaw   float64
		pitch float64
	}{{math.NaN(), 0}, {0, math.Inf(1)}, {0, 90.1}, {0, -90.1}} {
		observation := normalizedThrowView(input.yaw, input.pitch)
		if observation.Status != utility.AvailabilityUnavailable || observation.Source != utility.SourceUnavailable {
			t.Fatalf("invalid angles were exported as observed: %+v", observation)
		}
	}
}
