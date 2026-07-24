package handlers

import "testing"

func TestCalculateLossBonus(t *testing.T) {
	tests := []struct {
		name   string
		losses int
		want   int
	}{
		{name: "base", losses: 0, want: 1400},
		{name: "one loss", losses: 1, want: 1900},
		{name: "maximum", losses: 4, want: 3400},
		{name: "capped", losses: 8, want: 3400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateLossBonus(tt.losses); got != tt.want {
				t.Fatalf("calculateLossBonus(%d) = %d, want %d", tt.losses, got, tt.want)
			}
		})
	}
}
