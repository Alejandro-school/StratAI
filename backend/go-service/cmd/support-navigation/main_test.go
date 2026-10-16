package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRequestRejectsTraversalUnknownFutureAndTrailingData(t *testing.T) {
	for _, input := range []string{
		`{"map_name":"../de_mirage","asset_sha256":"` + strings.Repeat("a", 64) + `"}`,
		`{"map_name":"de_mirage","asset_sha256":"` + strings.Repeat("a", 64) + `","future":true}`,
		`{"map_name":"de_mirage","asset_sha256":"` + strings.Repeat("a", 64) + `"} {}`,
		`{"map_name":"de_mirage","asset_sha256":"wrong"}`,
	} {
		var out bytes.Buffer
		if err := run(strings.NewReader(input), &out); err == nil || out.Len() != 0 {
			t.Fatal("invalid request produced data")
		}
	}
}

func TestMissingEndpointComponentCannotBecomeObservedZero(t *testing.T) {
	input := `{"map_name":"de_mirage","asset_sha256":"` + strings.Repeat("a", 64) + `","queries":[{"origin":{"X":0,"Y":0},"target":{"X":0,"Y":0,"Z":0}}]}`
	var out bytes.Buffer
	err := run(strings.NewReader(input), &out)
	if err == nil || !strings.Contains(err.Error(), "endpoint component") || out.Len() != 0 {
		t.Fatalf("missing endpoint not rejected first: %v", err)
	}
}
