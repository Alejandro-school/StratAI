package parser

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"cs2-demo-service/models"
)

type canonicalReplayIndex struct {
	SchemaID          string                      `json:"schema_id"`
	MatchID           string                      `json:"match_id"`
	Metadata          models.ReplayMetadata       `json:"metadata"`
	SampleStrideTicks int                         `json:"sample_stride_ticks"`
	SampleIntervalMS  float64                     `json:"sample_interval_ms"`
	Rounds            []canonicalReplayRoundIndex `json:"rounds"`
}

type canonicalReplayRoundIndex struct {
	RoundNumber     int                  `json:"round_number"`
	StartTick       int                  `json:"start_tick"`
	EndTick         int                  `json:"end_tick"`
	WinnerSide      string               `json:"winner_side"`
	FrameCount      int                  `json:"frame_count"`
	EventCount      int                  `json:"event_count"`
	CombatShotCount int                  `json:"combat_shot_count"`
	Events          []models.ReplayEvent `json:"events"`
	Path            string               `json:"path"`
	SHA256          string               `json:"sha256"`
	Bytes           int64                `json:"bytes"`
}

type canonicalReplayRound struct {
	SchemaID string             `json:"schema_id"`
	MatchID  string             `json:"match_id"`
	Round    models.ReplayRound `json:"round"`
}

func exportCanonicalReplayPresentation(
	matchDir, matchID string,
	replay *models.ReplayData,
	combatEventIDs, combatShotIDs map[string]string,
) error {
	if replay == nil {
		return nil
	}

	replayDir := filepath.Join(matchDir, "canonical", "presentation", "replay")
	if err := os.MkdirAll(replayDir, 0o755); err != nil {
		return fmt.Errorf("create canonical replay directory: %w", err)
	}

	rounds := append([]models.ReplayRound(nil), replay.Rounds...)
	sort.Slice(rounds, func(i, j int) bool { return rounds[i].Round < rounds[j].Round })
	artifacts := make([]map[string]interface{}, 0, len(rounds)+1)
	indexRounds := make([]canonicalReplayRoundIndex, 0, len(rounds))
	for _, sourceRound := range rounds {
		round, err := canonicalReplayCombatReferences(sourceRound, combatEventIDs, combatShotIDs)
		if err != nil {
			return err
		}
		relativePath := fmt.Sprintf("presentation/replay/round_%03d.json.gz", round.Round)
		absolutePath := filepath.Join(matchDir, "canonical", filepath.FromSlash(relativePath))
		payload := canonicalReplayRound{
			SchemaID: "stratai.replay_round@5",
			MatchID:  matchID,
			Round:    round,
		}
		if err := writeGzipJSON(absolutePath, payload); err != nil {
			return err
		}
		checksum, size, err := artifactIntegrity(absolutePath)
		if err != nil {
			return err
		}
		indexRounds = append(indexRounds, canonicalReplayRoundIndex{
			RoundNumber:     round.Round,
			StartTick:       round.StartTick,
			EndTick:         round.EndTick,
			WinnerSide:      normalizeSide(round.Winner),
			FrameCount:      len(round.Frames),
			EventCount:      len(round.Events),
			CombatShotCount: len(round.CombatShots),
			Events:          round.Events,
			Path:            relativePath,
			SHA256:          checksum,
			Bytes:           size,
		})
		artifacts = append(artifacts, canonicalArtifactDescriptor(
			"replay_round", relativePath, "stratai.replay_round@5", "json", "gzip", 1, checksum, size,
		))
	}

	stride := replaySampleStride(rounds)
	intervalMS := 0.0
	if replay.Metadata.TickRate > 0 {
		intervalMS = float64(stride) * 1000 / replay.Metadata.TickRate
	}
	index := canonicalReplayIndex{
		SchemaID:          "stratai.replay_index@5",
		MatchID:           matchID,
		Metadata:          replay.Metadata,
		SampleStrideTicks: stride,
		SampleIntervalMS:  intervalMS,
		Rounds:            indexRounds,
	}
	indexRelativePath := "presentation/replay/index.json"
	indexPath := filepath.Join(matchDir, "canonical", filepath.FromSlash(indexRelativePath))
	if err := writeJSON(indexPath, index); err != nil {
		return err
	}
	checksum, size, err := artifactIntegrity(indexPath)
	if err != nil {
		return err
	}
	artifacts = append(artifacts, canonicalArtifactDescriptor(
		"replay_index", indexRelativePath, "stratai.replay_index@5", "json", "", 1, checksum, size,
	))

	return appendCanonicalArtifacts(filepath.Join(matchDir, "canonical", "manifest.json"), artifacts)
}

func canonicalReplayCombatReferences(
	source models.ReplayRound,
	eventIDs, shotIDs map[string]string,
) (models.ReplayRound, error) {
	round := source
	round.Events = append([]models.ReplayEvent(nil), source.Events...)
	for index := range round.Events {
		event := &round.Events[index]
		if event.Type != "player_hurt" && event.Type != "kill" {
			continue
		}
		mapped := make([]string, 0, len(event.SourceEventIDs))
		for _, sourceID := range event.SourceEventIDs {
			canonicalID, ok := eventIDs[sourceID]
			if !ok {
				return models.ReplayRound{}, fmt.Errorf("replay combat event %s references unknown source %s", event.ID, sourceID)
			}
			mapped = append(mapped, canonicalID)
		}
		event.SourceEventIDs = mapped
	}
	round.CombatShots = append([]models.ReplayShot(nil), source.CombatShots...)
	if err := canonicalizeReplayShots(round.CombatShots, eventIDs, shotIDs); err != nil {
		return models.ReplayRound{}, err
	}
	round.Frames = append([]models.ReplayFrame(nil), source.Frames...)
	for frameIndex := range round.Frames {
		round.Frames[frameIndex].Players = canonicalReplayPlayers(source.Frames[frameIndex].Players)
		round.Frames[frameIndex].Shots = append([]models.ReplayShot(nil), source.Frames[frameIndex].Shots...)
		if err := canonicalizeReplayShots(round.Frames[frameIndex].Shots, eventIDs, shotIDs); err != nil {
			return models.ReplayRound{}, err
		}
	}
	return round, nil
}

func filterCanonicalReplayToRounds(replay *models.ReplayData, rounds []models.CanonicalRound) *models.ReplayData {
	if replay == nil {
		return nil
	}
	filtered := &models.ReplayData{Metadata: replay.Metadata, Rounds: make([]models.ReplayRound, 0, len(rounds))}
	byRound := make(map[int]models.ReplayRound, len(replay.Rounds))
	for _, round := range replay.Rounds {
		byRound[round.Round] = round
	}
	for _, canonicalRound := range rounds {
		source, exists := byRound[canonicalRound.RoundNumber]
		if !exists {
			continue
		}
		round := source
		if canonicalRound.StartTick != nil {
			round.StartTick = *canonicalRound.StartTick
		}
		if canonicalRound.EndTick != nil {
			round.EndTick = *canonicalRound.EndTick
		}
		round.Frames = make([]models.ReplayFrame, 0, len(source.Frames))
		for _, frame := range source.Frames {
			if canonicalTickWithinRound(rounds, source.Round, frame.Tick) {
				round.Frames = append(round.Frames, frame)
			}
		}
		round.Events = make([]models.ReplayEvent, 0, len(source.Events))
		for _, event := range source.Events {
			if canonicalTickWithinRound(rounds, source.Round, event.Tick) {
				round.Events = append(round.Events, event)
			}
		}
		round.CombatShots = make([]models.ReplayShot, 0, len(source.CombatShots))
		for _, shot := range source.CombatShots {
			if canonicalTickWithinRound(rounds, source.Round, shot.Tick) {
				round.CombatShots = append(round.CombatShots, shot)
			}
		}
		filtered.Rounds = append(filtered.Rounds, round)
	}
	return filtered
}

func canonicalReplayPlayers(players []models.ReplayPlayerState) []models.ReplayPlayerState {
	canonical := make([]models.ReplayPlayerState, 0, len(players))
	for _, player := range players {
		if player.SteamID != 0 {
			canonical = append(canonical, player)
		}
	}
	return canonical
}

func canonicalizeReplayShots(shots []models.ReplayShot, eventIDs, shotIDs map[string]string) error {
	for index := range shots {
		shot := &shots[index]
		eventID, eventOK := eventIDs[shot.SourceEventID]
		shotID, shotOK := shotIDs[shot.ShotID]
		if !eventOK || !shotOK {
			return fmt.Errorf("replay shot %s has invalid combat provenance", shot.SourceEventID)
		}
		shot.SourceEventID = eventID
		shot.ShotID = shotID
	}
	return nil
}

func replaySampleStride(rounds []models.ReplayRound) int {
	for _, round := range rounds {
		for index := 1; index < len(round.Frames); index++ {
			stride := round.Frames[index].Tick - round.Frames[index-1].Tick
			if stride > 0 {
				return stride
			}
		}
	}
	return 0
}

func writeGzipJSON(path string, value interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	writer := gzip.NewWriter(file)
	encodeErr := json.NewEncoder(writer).Encode(value)
	gzipErr := writer.Close()
	closeErr := file.Close()
	if encodeErr != nil {
		return fmt.Errorf("encode %s: %w", path, encodeErr)
	}
	if gzipErr != nil {
		return fmt.Errorf("compress %s: %w", path, gzipErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", path, closeErr)
	}
	return nil
}

func artifactIntegrity(path string) (string, int64, error) {
	checksum, err := checksumFile(path)
	if err != nil {
		return "", 0, fmt.Errorf("checksum %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, fmt.Errorf("stat %s: %w", path, err)
	}
	return checksum, info.Size(), nil
}

func canonicalArtifactDescriptor(
	artifactType, path, schemaID, format, compression string,
	recordCount int,
	checksum string,
	size int64,
) map[string]interface{} {
	descriptor := map[string]interface{}{
		"artifact_type": artifactType,
		"path":          path,
		"schema_id":     schemaID,
		"format":        format,
		"record_count":  recordCount,
		"sha256":        checksum,
		"bytes":         size,
		"sort_order":    []string{},
	}
	if compression != "" {
		descriptor["compression"] = compression
	}
	return descriptor
}

func appendCanonicalArtifacts(manifestPath string, additions []map[string]interface{}) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read canonical manifest: %w", err)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("decode canonical manifest: %w", err)
	}
	rawArtifacts, ok := manifest["artifacts"].([]interface{})
	if !ok {
		return fmt.Errorf("canonical manifest artifacts is not an array")
	}
	for _, addition := range additions {
		rawArtifacts = append(rawArtifacts, addition)
	}
	sort.Slice(rawArtifacts, func(i, j int) bool {
		left, _ := rawArtifacts[i].(map[string]interface{})["path"].(string)
		right, _ := rawArtifacts[j].(map[string]interface{})["path"].(string)
		return left < right
	})
	manifest["artifacts"] = rawArtifacts
	return writeJSON(manifestPath, manifest)
}
