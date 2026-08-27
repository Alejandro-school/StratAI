package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSteamIDsMarshalAsStrings(t *testing.T) {
	data, err := json.Marshal(ReplayEvent{KillerID: 76561198000000001, PlayerID: 76561198000000002})
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(data)
	if !strings.Contains(jsonText, `"killer_id":"76561198000000001"`) || !strings.Contains(jsonText, `"player_id":"76561198000000002"`) {
		t.Fatalf("unsafe numeric ID in JSON: %s", jsonText)
	}
}

func TestTrackingSteamIDMarshalsAsString(t *testing.T) {
	data, err := json.Marshal(AI_TrackingEvent{PlayerSteamID: 76561198000000001})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"player_steam_id":"76561198000000001"`) {
		t.Fatalf("unsafe tracking ID in JSON: %s", data)
	}
}
