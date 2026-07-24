package analyzers

import "testing"

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
