package parser

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cs2-demo-service/models"
)

func TestCanonicalReplayCombatReferencesDropsAnonymousPlayers(t *testing.T) {
	source := models.ReplayRound{Frames: []models.ReplayFrame{{Players: []models.ReplayPlayerState{
		{SteamID: 0, Name: "anonymous"},
		{SteamID: 42, Name: "identified"},
	}}}}

	round, err := canonicalReplayCombatReferences(source, nil, nil)
	if err != nil {
		t.Fatalf("canonicalReplayCombatReferences() error = %v", err)
	}
	if len(round.Frames[0].Players) != 1 || round.Frames[0].Players[0].SteamID != 42 {
		t.Fatalf("unexpected canonical replay players: %#v", round.Frames[0].Players)
	}
	if len(source.Frames[0].Players) != 2 {
		t.Fatal("canonical replay conversion mutated the source frame")
	}
}

func TestWriteGzipJSONIsByteDeterministic(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.json.gz")
	second := filepath.Join(root, "second.json.gz")
	payload := map[string]interface{}{
		"schema_id": "stratai.test@1",
		"values":    []int{3, 2, 1},
	}
	if err := writeGzipJSON(first, payload); err != nil {
		t.Fatal(err)
	}
	if err := writeGzipJSON(second, payload); err != nil {
		t.Fatal(err)
	}
	firstBytes, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("identical replay payloads produced different gzip bytes")
	}
}

func TestCanonicalReplayUsesVersionedSchemas(t *testing.T) {
	matchDir := t.TempDir()
	canonicalDir := filepath.Join(matchDir, "canonical")
	if err := os.MkdirAll(canonicalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonicalDir, "manifest.json"), []byte(`{"artifacts":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	replay := &models.ReplayData{
		Metadata: models.ReplayMetadata{SchemaVersion: 5, MatchID: "replay-version"},
		Rounds:   []models.ReplayRound{{Round: 1, StartTick: 10, EndTick: 20}},
	}
	if err := exportCanonicalReplayPresentation(matchDir, "replay-version", replay, nil, nil); err != nil {
		t.Fatal(err)
	}

	index := readCanonicalJSON[canonicalReplayIndex](t, filepath.Join(canonicalDir, "presentation", "replay", "index.json"))
	if index.SchemaID != "stratai.replay_index@5" || index.Metadata.SchemaVersion != 5 {
		t.Fatalf("unexpected replay index contract: %+v", index)
	}
	roundPath := filepath.Join(canonicalDir, filepath.FromSlash(index.Rounds[0].Path))
	file, err := os.Open(roundPath)
	if err != nil {
		t.Fatal(err)
	}
	decompressor, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	var round canonicalReplayRound
	decodeErr := json.NewDecoder(decompressor).Decode(&round)
	closeErr := decompressor.Close()
	fileCloseErr := file.Close()
	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if fileCloseErr != nil {
		t.Fatal(fileCloseErr)
	}
	if round.SchemaID != "stratai.replay_round@5" {
		t.Fatalf("unexpected replay round contract: %+v", round)
	}
}

func TestFilterCanonicalReplayReplacesPreliminaryRoundEpoch(t *testing.T) {
	start, end := 100, 200
	replay := &models.ReplayData{Rounds: []models.ReplayRound{{
		Round:       1,
		StartTick:   1,
		EndTick:     200,
		Frames:      []models.ReplayFrame{{Tick: 20}, {Tick: 120}},
		Events:      []models.ReplayEvent{{Tick: 30}, {Tick: 130}},
		CombatShots: []models.ReplayShot{{Tick: 40}, {Tick: 140}},
	}}}

	filtered := filterCanonicalReplayToRounds(replay, []models.CanonicalRound{{
		RoundNumber: 1,
		StartTick:   &start,
		EndTick:     &end,
	}})

	if filtered == nil || len(filtered.Rounds) != 1 {
		t.Fatalf("filtered replay is incomplete: %+v", filtered)
	}
	round := filtered.Rounds[0]
	if round.StartTick != start || round.EndTick != end || len(round.Frames) != 1 || round.Frames[0].Tick != 120 ||
		len(round.Events) != 1 || round.Events[0].Tick != 130 || len(round.CombatShots) != 1 || round.CombatShots[0].Tick != 140 {
		t.Fatalf("preliminary replay epoch survived filtering: %+v", round)
	}
	if replay.Rounds[0].StartTick != 1 || len(replay.Rounds[0].Frames) != 2 {
		t.Fatal("replay filter mutated parser-owned data")
	}
}
