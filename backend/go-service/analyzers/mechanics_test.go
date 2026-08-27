package analyzers

import "testing"

func TestHorizontalVelocityHistoryRequiresExactAvailableSample(t *testing.T) {
	history := make(horizontalVelocityHistory)
	playerID := uint64(7)
	history.record(playerID, horizontalVelocitySample{round: 2, tick: 100, speed: 180, available: true})
	history.record(playerID, horizontalVelocitySample{round: 2, tick: 102, speed: 90, available: true})

	if _, available := history.at(playerID, 2, 101); available {
		t.Fatal("a missing tick must not reuse an older velocity")
	}
	if speed, available := history.at(playerID, 2, 100); !available || speed != 180 {
		t.Fatalf("expected exact causal sample, got speed=%f available=%t", speed, available)
	}
	if _, available := history.at(playerID, 3, 100); available {
		t.Fatal("a velocity from another round must not be reused")
	}
}

func TestHorizontalVelocityHistoryPreservesUnavailableSample(t *testing.T) {
	history := make(horizontalVelocityHistory)
	playerID := uint64(8)
	history.record(playerID, horizontalVelocitySample{round: 1, tick: 50, speed: 140, available: true})
	history.record(playerID, horizontalVelocitySample{round: 1, tick: 50, available: false})

	if _, available := history.at(playerID, 1, 50); available {
		t.Fatal("an unavailable observation must not be classified as stationary")
	}
	if len(history[playerID]) != 1 {
		t.Fatalf("same-tick observation should be replaced, got %d samples", len(history[playerID]))
	}
}

func TestCalculateCounterStrafeRating(t *testing.T) {
	tests := []struct {
		name  string
		speed float64
		want  float64
	}{
		{name: "accurate", speed: 80, want: 100},
		{name: "partial", speed: 160, want: 50},
		{name: "running", speed: 240, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := calculateCounterStrafeRating(test.speed, 80, 240)
			if got != test.want {
				t.Fatalf("got %f, want %f", got, test.want)
			}
		})
	}
}
