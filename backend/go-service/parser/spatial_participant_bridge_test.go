package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSpatialBridgeUsesSameAliasesAndRejectsUnreconciledIdentity(t *testing.T) {
	replay := exposureReplayFixture()
	checksum := strings.Repeat("a", 64)
	first, err := BuildSpatialParticipantBridge(replay, "demo-"+checksum[:20], checksum)
	if err != nil {
		t.Fatal(err)
	}
	aliases, _ := exposureParticipantAliases(replay.Rounds)
	if len(first.Participants) != len(aliases) || first.TrainingAllowed {
		t.Fatal("invalid bridge coverage or authorization")
	}
	encoded, _ := json.Marshal(first)
	if strings.Contains(string(encoded), "steam:") || strings.Contains(string(encoded), "SteamID") {
		t.Fatal("bridge retained raw identity")
	}
	for _, entry := range first.Participants {
		if len(entry.ActorKey) != 64 || !strings.HasPrefix(entry.Alias, "participant-") {
			t.Fatal("bridge keys must be opaque")
		}
	}
	second, _ := BuildSpatialParticipantBridge(replay, "demo-"+checksum[:20], checksum)
	encoded2, _ := json.Marshal(second)
	if string(encoded) != string(encoded2) {
		t.Fatal("bridge must be deterministic")
	}
	replay.Rounds[0].Frames[0].Players[0].SteamID = 0
	if _, err := BuildSpatialParticipantBridge(replay, "demo-"+checksum[:20], checksum); err == nil {
		t.Fatal("unreconciled participant accepted")
	}
}
