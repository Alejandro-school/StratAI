package parser

import (
	"bufio"
	"compress/gzip"
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
	engagementpkg "cs2-demo-service/pkg/engagement"
	"cs2-demo-service/pkg/objective"
	"cs2-demo-service/pkg/playerstate"
	"cs2-demo-service/pkg/utility"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const (
	canonicalExportFormatVersion          = "3.8.0"
	canonicalAnonymousPlayerRunSeconds    = 10.0
	canonicalAnonymousPlayerMaxGapSeconds = 2.0
)

type canonicalArtifactWriter struct {
	root      string
	artifacts []models.CanonicalArtifact
}

func ExportCanonicalBundle(
	ctx *models.DemoContext,
	matchID string,
	matchDir string,
	playedAt string,
	durationSeconds float64,
	tickRate float64,
	quality interface{},
	provenanceValues ...models.CanonicalExportProvenance,
) error {
	provenance := models.CanonicalExportProvenance{Source: "demo", DemoChecksum: strings.Repeat("0", 64)}
	if len(provenanceValues) > 0 {
		provenance = provenanceValues[0]
	}
	if err := validateCanonicalRosterIdentityQuality(ctx, tickRate); err != nil {
		return fmt.Errorf("canonical roster identity quality gates failed: %w", err)
	}
	root := filepath.Join(matchDir, "canonical")
	writer := canonicalArtifactWriter{root: root, artifacts: make([]models.CanonicalArtifact, 0, 16+ctx.CurrentRound)}
	for _, directory := range []string{"core", "events", filepath.Join("states", "player_states"), filepath.Join("states", "tactical"), "derived", "causal", "diagnostics"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o750); err != nil {
			return fmt.Errorf("create canonical directory %s: %w", directory, err)
		}
	}

	rosters := buildCanonicalRosters(ctx)
	matchExport := buildCanonicalMatch(ctx, matchID, playedAt, durationSeconds, tickRate, rosters)
	metadata := buildCanonicalMatchMetadata(matchID, playedAt, provenance)
	objectiveEvents, err := buildCanonicalObjectiveEvents(ctx, matchID, tickRate)
	if err != nil {
		return fmt.Errorf("build canonical objective events: %w", err)
	}
	rounds := buildCanonicalRounds(ctx, matchID, rosters, objectiveEvents)
	if err := validateCanonicalOutcomeConsistency(matchExport, rounds, ctx.ReplayData); err != nil {
		return fmt.Errorf("validate canonical outcomes: %w", err)
	}
	if err := validateCanonicalObjectives(rounds, objectiveEvents); err != nil {
		return fmt.Errorf("validate canonical objectives: %w", err)
	}
	if err := writer.writeJSON("match", filepath.Join("core", "match.json"), "stratai.match@1", 1, nil, matchExport); err != nil {
		return err
	}
	if err := writer.writeJSON("match_metadata", filepath.Join("core", "match_metadata.json"), "stratai.match_metadata@1", 1, nil, metadata); err != nil {
		return err
	}
	participants := buildCanonicalParticipants(ctx, matchID, rosters)
	if err := writer.writeJSON("participants", filepath.Join("core", "participants.json"), "stratai.participants@1", len(participants.Players), []string{"player_id"}, participants); err != nil {
		return err
	}
	if err := writer.writeJSON("rounds", filepath.Join("core", "rounds.json"), "stratai.rounds@2", len(rounds.Rounds), []string{"round_number"}, rounds); err != nil {
		return err
	}

	combatEvents, err := buildCanonicalCombatEvents(ctx, matchID, rounds.Rounds)
	if err != nil {
		return fmt.Errorf("build canonical combat events: %w", err)
	}
	if err := writer.writeJSONLines("combat_events", filepath.Join("events", "combat_events.jsonl"), "stratai.combat_event@2", []string{"round_number", "tick", "sequence_in_tick", "event_id"}, combatEvents); err != nil {
		return err
	}
	utilityEvents, err := buildCanonicalUtilityEvents(ctx, matchID, rounds.Rounds)
	if err != nil {
		return fmt.Errorf("build canonical utility events: %w", err)
	}
	if err := writer.writeJSONLines("utility_events", filepath.Join("events", "utility_events.jsonl"), "stratai.utility_event@2", []string{"round_number", "sequence_in_round", "event_id"}, utilityEvents); err != nil {
		return err
	}
	if err := writer.writeJSONLines("objective_events", filepath.Join("events", "objective_events.jsonl"), "stratai.objective_event@2", []string{"round_number", "tick", "sequence_in_tick", "event_id"}, objectiveEvents); err != nil {
		return err
	}

	statesByRound := buildCanonicalPlayerStates(ctx, matchID, rosters)
	statesByRound = filterCanonicalPlayerStatesToRounds(statesByRound, rounds.Rounds)
	if err := validateCanonicalPlayerStates(statesByRound); err != nil {
		return fmt.Errorf("validate canonical player states: %w", err)
	}
	for roundNumber := 1; roundNumber <= ctx.CurrentRound; roundNumber++ {
		path := filepath.Join("states", "player_states", fmt.Sprintf("round_%03d.jsonl", roundNumber))
		if err := writer.writeJSONLines("player_states", path, "stratai.player_state@3", []string{"tick", "player_id"}, statesByRound[roundNumber]); err != nil {
			return err
		}
	}
	if ctx.ReplayData == nil {
		return fmt.Errorf("build tactical export: replay data is unavailable")
	}
	tacticalSource := filterCanonicalReplayToRounds(ctx.ReplayData, rounds.Rounds)
	if tacticalSource == nil {
		return fmt.Errorf("build tactical export: filtered replay data is unavailable")
	}
	tacticalReplay := *tacticalSource
	tacticalReplay.Metadata = tacticalSource.Metadata
	tacticalReplay.Metadata.MatchID = matchID
	tacticalReplay.Metadata.TickRate = tickRate
	tactical, err := BuildTacticalExport(&tacticalReplay, ctx.MapManager)
	if err != nil {
		return err
	}
	tacticalSampling := models.TacticalSamplingManifest{
		SchemaID: models.TacticalSamplingSchemaID, MatchID: matchID,
		IdentitySemantics: tactical.IdentitySemantics, JoinKeys: tactical.JoinKeys,
		Sampling: tactical.Sampling, PhysicalRowCount: len(tactical.PhysicalRows),
		OracleRowCount: len(tactical.OracleRows), GapCount: len(tactical.Gaps),
	}
	if err := writer.writeJSON("tactical_sampling", filepath.Join("states", "tactical", "sampling.json"), models.TacticalSamplingSchemaID, 1, nil, tacticalSampling); err != nil {
		return err
	}
	if err := writer.writeGzipJSONLines("tactical_observations", filepath.Join("states", "tactical", "observed.jsonl.gz"), models.TacticalPhysicalSchemaID, []string{"round_number", "tick", "join_keys.observer_id", "join_keys.subject_id"}, tactical.PhysicalRows); err != nil {
		return err
	}
	if err := writer.writeJSONLines("tactical_oracle", filepath.Join("states", "tactical", "oracle.jsonl"), models.TacticalOracleSchemaID, []string{"round_number", "tick", "join_keys.subject_id"}, tactical.OracleRows); err != nil {
		return err
	}
	if err := writer.writeJSONLines("tactical_gaps", filepath.Join("states", "tactical", "gaps.jsonl"), models.TacticalSamplingGapSchemaID, []string{"round_number", "tick", "reason"}, tactical.Gaps); err != nil {
		return err
	}

	engagements, trades, err := engagementpkg.Derive(
		matchID,
		tickRate,
		rounds.Rounds,
		participants.Players,
		combatEvents,
		statesByRound,
		ctx.MapManager,
	)
	if err != nil {
		return fmt.Errorf("derive canonical engagements and trades: %w", err)
	}
	causalPartitions, err := buildCanonicalCausalPartitions(matchID, engagements.Engagements, trades)
	if err != nil {
		return fmt.Errorf("build canonical causal partitions: %w", err)
	}
	legacyStats := buildCanonicalPlayerMatchStats(ctx, matchID, trades, combatEvents, rounds.Rounds)
	clutches := buildCanonicalClutchEvents(matchID, rounds.Rounds, participants.Players, combatEvents)
	stats := buildCanonicalBlock6PlayerStats(matchID, rosters, legacyStats, clutches)
	economyRounds, economyPlayers := buildCanonicalBlock6Economy(ctx, matchID, rosters, rounds.Rounds, objectiveEvents, combatEvents, playedAt)
	qualityReportValue, err := canonicalQualityReport(quality)
	if err != nil {
		return err
	}
	// The combat tracker can contain observations from a preliminary epoch that
	// canonical round bounds intentionally exclude. The published quality report
	// must describe the rows that were actually exported, not the larger raw
	// snapshot inspected by the pre-export gate.
	qualityReportValue.CombatLedgerEvents = len(combatEvents)
	assessment := assessEngagementQuality(
		matchID,
		tickRate,
		rounds.Rounds,
		participants.Players,
		combatEvents,
		statesByRound,
		engagements,
		trades,
		legacyStats,
		ctx.MapManager,
	)
	qualityReportValue.applyEngagementQuality(assessment)
	if qualityReportValue.hasHardEngagementFailure() {
		return fmt.Errorf("canonical engagement quality gates failed: %s", strings.Join(qualityReportValue.engagementFailureDetails, "; "))
	}
	block6Assessment := assessBlock6Quality(ctx, rounds.Rounds, participants.Players, combatEvents, economyRounds, economyPlayers, stats, clutches, metadata)
	qualityReportValue.applyBlock6Quality(block6Assessment)
	if qualityReportValue.hasHardBlock6Failure() {
		return fmt.Errorf("canonical block 6 quality gates failed: %s", strings.Join(qualityReportValue.block6FailureDetails, "; "))
	}
	if err := writer.writeJSON("engagements", filepath.Join("derived", "engagements.json"), "stratai.engagements@2", len(engagements.Engagements), []string{"round_number", "start_tick", "start_sequence_in_tick", "engagement_id"}, engagements); err != nil {
		return err
	}
	if err := writer.writeJSON("trades", filepath.Join("derived", "trades.json"), "stratai.trades@1", len(trades.Candidates), []string{"round_number", "death_tick", "death_sequence_in_tick", "trade_candidate_id"}, trades); err != nil {
		return err
	}
	if err := writer.writeJSONLines("decisions", filepath.Join("causal", "decisions.jsonl"), block7DecisionSchema, []string{"round_number", "t0_tick", "decision_id"}, causalPartitions.Decisions); err != nil {
		return err
	}
	if err := writer.writeJSONLines("decision_features", filepath.Join("causal", "decision_features.jsonl"), block7DecisionFeatureSchema, []string{"round_number", "t0_tick", "decision_id"}, causalPartitions.DecisionFeatures); err != nil {
		return err
	}
	if err := writer.writeJSONLines("oracle_context", filepath.Join("causal", "oracle_context.jsonl"), block7OracleContextSchema, []string{"round_number", "t0_tick", "decision_id"}, causalPartitions.OracleContext); err != nil {
		return err
	}
	if err := writer.writeJSONLines("decision_outcomes", filepath.Join("causal", "outcomes.jsonl"), block7OutcomeSchema, []string{"round_number", "t0_tick", "decision_id"}, causalPartitions.Outcomes); err != nil {
		return err
	}
	if err := writer.writeJSONLines("quality_masks", filepath.Join("causal", "quality_masks.jsonl"), block7QualityMaskSchema, []string{"round_number", "t0_tick", "decision_id"}, causalPartitions.QualityMasks); err != nil {
		return err
	}
	if err := writer.writeJSON("economy_rounds", filepath.Join("derived", "economy_rounds.json"), "stratai.economy_round@1", len(economyRounds.Rounds), []string{"round_number", "team_id"}, economyRounds); err != nil {
		return err
	}
	if err := writer.writeJSON("economy_players", filepath.Join("derived", "economy_players.json"), "stratai.economy_player@1", len(economyPlayers.Players), []string{"round_number", "player_id"}, economyPlayers); err != nil {
		return err
	}
	if err := writer.writeJSON("player_stats", filepath.Join("derived", "player_stats.json"), "stratai.player_stats@1", len(stats.Players), []string{"player_id"}, stats); err != nil {
		return err
	}
	if err := writer.writeJSON("clutch_events", filepath.Join("derived", "clutch_events.json"), "stratai.clutch_event@1", len(clutches.ClutchEvents), []string{"round_number", "team_id", "clutch_id"}, clutches); err != nil {
		return err
	}
	qualityExport := models.CanonicalQualityReportExport{
		SchemaID: "stratai.quality_report@1", MatchID: matchID, Report: qualityReportValue,
	}
	if err := writer.writeJSON("quality_report", filepath.Join("diagnostics", "quality_report.json"), "stratai.quality_report@1", 1, nil, qualityExport); err != nil {
		return err
	}

	sort.Slice(writer.artifacts, func(i, j int) bool { return writer.artifacts[i].Path < writer.artifacts[j].Path })
	manifest := models.CanonicalManifest{
		SchemaID: "stratai.canonical_manifest@3", ExportFormatVersion: canonicalExportFormatVersion,
		MatchID: matchID, DemoChecksumSHA256: provenance.DemoChecksum,
		ParserVersion: block6ParserVersion, ConfigurationHashes: metadata.ConfigurationHashes,
		TransformationVersions: metadata.TransformationVersions,
		Lineage:                buildCanonicalLineage(ctx, tickRate, provenance, metadata, qualityReportValue),
		Artifacts:              writer.artifacts,
	}
	if err := writeJSON(filepath.Join(root, "manifest.json"), manifest); err != nil {
		return fmt.Errorf("write canonical manifest: %w", err)
	}
	return nil
}

func validateCanonicalRosterIdentityQuality(ctx *models.DemoContext, tickRate float64) error {
	if ctx == nil || tickRate <= 0 || math.IsNaN(tickRate) || math.IsInf(tickRate, 0) {
		return nil
	}
	ticksByRound := make(map[int][]int)
	for _, state := range ctx.AI_TrackingEventsWithRound {
		if state.Event.PlayerSteamID == 0 && state.Round > 0 && state.Event.Tick >= 0 {
			ticksByRound[state.Round] = append(ticksByRound[state.Round], state.Event.Tick)
		}
	}
	thresholdTicks := int(math.Ceil(canonicalAnonymousPlayerRunSeconds * tickRate))
	maxGapTicks := int(math.Ceil(canonicalAnonymousPlayerMaxGapSeconds * tickRate))
	rounds := make([]int, 0, len(ticksByRound))
	for roundNumber := range ticksByRound {
		rounds = append(rounds, roundNumber)
	}
	sort.Ints(rounds)
	for _, roundNumber := range rounds {
		ticks := ticksByRound[roundNumber]
		sort.Ints(ticks)
		if len(ticks) < 2 {
			continue
		}
		runStart := ticks[0]
		previous := ticks[0]
		for _, tick := range ticks[1:] {
			if tick == previous {
				continue
			}
			if tick-previous > maxGapTicks {
				if previous-runStart >= thresholdTicks {
					return fmt.Errorf("round %d has a prolonged unreconciled playing participant", roundNumber)
				}
				runStart = tick
			}
			previous = tick
		}
		if previous-runStart >= thresholdTicks {
			return fmt.Errorf("round %d has a prolonged unreconciled playing participant", roundNumber)
		}
	}
	return nil
}

func canonicalQualityReport(value interface{}) (qualityReport, error) {
	if report, ok := value.(qualityReport); ok {
		return report, nil
	}
	if value == nil {
		return qualityReport{}, fmt.Errorf("canonical quality report is required")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return qualityReport{}, fmt.Errorf("marshal canonical quality report: %w", err)
	}
	var report qualityReport
	if err := json.Unmarshal(encoded, &report); err != nil {
		return qualityReport{}, fmt.Errorf("decode canonical quality report: %w", err)
	}
	return report, nil
}

func (writer *canonicalArtifactWriter) writeJSON(
	artifactType, relativePath, schemaID string,
	recordCount int,
	sortOrder []string,
	payload interface{},
) error {
	path := filepath.Join(writer.root, relativePath)
	if err := writeJSON(path, payload); err != nil {
		return fmt.Errorf("write canonical artifact %s: %w", relativePath, err)
	}
	return writer.register(artifactType, relativePath, schemaID, "json", recordCount, sortOrder)
}

func (writer *canonicalArtifactWriter) writeJSONLines(
	artifactType, relativePath, schemaID string,
	sortOrder []string,
	records interface{},
) error {
	path := filepath.Join(writer.root, relativePath)
	recordCount, err := writeJSONLines(path, records)
	if err != nil {
		return fmt.Errorf("write canonical artifact %s: %w", relativePath, err)
	}
	return writer.register(artifactType, relativePath, schemaID, "jsonl", recordCount, sortOrder)
}

func (writer *canonicalArtifactWriter) writeGzipJSONLines(
	artifactType, relativePath, schemaID string,
	sortOrder []string,
	records interface{},
) error {
	path := filepath.Join(writer.root, relativePath)
	recordCount, err := writeGzipJSONLines(path, records)
	if err != nil {
		return fmt.Errorf("write canonical artifact %s: %w", relativePath, err)
	}
	return writer.registerWithCompression(
		artifactType,
		relativePath,
		schemaID,
		"jsonl",
		"gzip",
		recordCount,
		sortOrder,
	)
}

func (writer *canonicalArtifactWriter) register(
	artifactType, relativePath, schemaID, format string,
	recordCount int,
	sortOrder []string,
) error {
	return writer.registerWithCompression(
		artifactType,
		relativePath,
		schemaID,
		format,
		"",
		recordCount,
		sortOrder,
	)
}

func (writer *canonicalArtifactWriter) registerWithCompression(
	artifactType, relativePath, schemaID, format, compression string,
	recordCount int,
	sortOrder []string,
) error {
	if sortOrder == nil {
		sortOrder = []string{}
	}
	path := filepath.Join(writer.root, relativePath)
	checksum, err := checksumFile(path)
	if err != nil {
		return fmt.Errorf("checksum canonical artifact %s: %w", relativePath, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat canonical artifact %s: %w", relativePath, err)
	}
	writer.artifacts = append(writer.artifacts, models.CanonicalArtifact{
		ArtifactType: artifactType, Path: filepath.ToSlash(relativePath), SchemaID: schemaID,
		Format: format, Compression: compression, RecordCount: recordCount,
		SHA256: checksum, Bytes: info.Size(), SortOrder: sortOrder,
	})
	return nil
}

func writeJSONLines(path string, records interface{}) (int, error) {
	rows := reflect.ValueOf(records)
	if rows.Kind() != reflect.Slice {
		return 0, fmt.Errorf("records must be a slice")
	}
	file, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	buffer := bufio.NewWriter(file)
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	for index := 0; index < rows.Len(); index++ {
		if err := encoder.Encode(rows.Index(index).Interface()); err != nil {
			_ = file.Close()
			return 0, err
		}
	}
	if err := buffer.Flush(); err != nil {
		_ = file.Close()
		return 0, err
	}
	if err := file.Close(); err != nil {
		return 0, err
	}
	return rows.Len(), nil
}

func writeGzipJSONLines(path string, records interface{}) (int, error) {
	rows := reflect.ValueOf(records)
	if rows.Kind() != reflect.Slice {
		return 0, fmt.Errorf("records must be a slice")
	}
	file, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	compressed := gzip.NewWriter(file)
	encoder := json.NewEncoder(compressed)
	encoder.SetEscapeHTML(false)
	for index := 0; index < rows.Len(); index++ {
		if err := encoder.Encode(rows.Index(index).Interface()); err != nil {
			_ = compressed.Close()
			_ = file.Close()
			return 0, err
		}
	}
	if err := compressed.Close(); err != nil {
		_ = file.Close()
		return 0, err
	}
	if err := file.Close(); err != nil {
		return 0, err
	}
	return rows.Len(), nil
}

type canonicalRosterInfo struct {
	playerTeams map[uint64]string
	teams       []models.CanonicalTeam
	sides       map[int]map[string]string
}

func buildCanonicalRosters(ctx *models.DemoContext) canonicalRosterInfo {
	info := canonicalRosterInfo{
		playerTeams: make(map[uint64]string),
		teams:       make([]models.CanonicalTeam, 0, 2),
		sides:       make(map[int]map[string]string, ctx.CurrentRound),
	}
	initialSides := make(map[string][]uint64)
	if len(ctx.AI_EconomyRounds) > 0 {
		rounds := append([]models.AI_EconomyRound(nil), ctx.AI_EconomyRounds...)
		sort.Slice(rounds, func(i, j int) bool { return rounds[i].Round < rounds[j].Round })
		for _, player := range rounds[0].Players {
			if player.SteamID != 0 {
				side := normalizeSide(player.Team)
				initialSides[side] = append(initialSides[side], player.SteamID)
			}
		}
	}
	if len(initialSides) < 2 {
		initialSides = make(map[string][]uint64)
		earliestRound := 0
		for _, state := range ctx.AI_TrackingEventsWithRound {
			if earliestRound == 0 || state.Round < earliestRound {
				earliestRound = state.Round
			}
		}
		seen := make(map[uint64]struct{})
		for _, state := range ctx.AI_TrackingEventsWithRound {
			if state.Round != earliestRound || state.Event.PlayerSteamID == 0 {
				continue
			}
			if _, exists := seen[state.Event.PlayerSteamID]; exists {
				continue
			}
			seen[state.Event.PlayerSteamID] = struct{}{}
			side := normalizeSide(state.Event.Team)
			initialSides[side] = append(initialSides[side], state.Event.PlayerSteamID)
		}
	}

	type rosterGroup struct {
		side      string
		steamIDs  []uint64
		signature string
	}
	groups := make([]rosterGroup, 0, 2)
	for _, side := range []string{"ct", "t"} {
		steamIDs := append([]uint64(nil), initialSides[side]...)
		if len(steamIDs) == 0 {
			continue
		}
		sort.Slice(steamIDs, func(i, j int) bool { return steamIDs[i] < steamIDs[j] })
		parts := make([]string, 0, len(steamIDs))
		for _, steamID := range steamIDs {
			parts = append(parts, strconv.FormatUint(steamID, 10))
		}
		groups = append(groups, rosterGroup{side: side, steamIDs: steamIDs, signature: strings.Join(parts, ",")})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].signature < groups[j].signature })
	for index, group := range groups {
		teamID := fmt.Sprintf("team_%c", 'a'+rune(index))
		info.teams = append(info.teams, models.CanonicalTeam{TeamID: teamID, StartingSide: group.side})
		for _, steamID := range group.steamIDs {
			info.playerTeams[steamID] = teamID
		}
	}

	for _, round := range ctx.AI_EconomyRounds {
		counts := make(map[string]map[string]int)
		for _, player := range round.Players {
			teamID := info.playerTeams[player.SteamID]
			if teamID == "" {
				continue
			}
			if counts[teamID] == nil {
				counts[teamID] = make(map[string]int)
			}
			counts[teamID][normalizeSide(player.Team)]++
		}
		info.sides[round.Round] = selectCanonicalTeamSides(info.teams, counts)
		for _, player := range round.Players {
			if info.playerTeams[player.SteamID] != "" {
				continue
			}
			side := normalizeSide(player.Team)
			if teamID := teamForSideValue(info.sides[round.Round], side); teamID != "" {
				info.playerTeams[player.SteamID] = teamID
			}
		}
	}
	return info
}

func selectCanonicalTeamSides(teams []models.CanonicalTeam, counts map[string]map[string]int) map[string]string {
	result := make(map[string]string, len(teams))
	for _, team := range teams {
		if counts[team.TeamID]["ct"] > counts[team.TeamID]["t"] {
			result[team.TeamID] = "ct"
		} else if counts[team.TeamID]["t"] > 0 {
			result[team.TeamID] = "t"
		}
	}
	if len(teams) == 2 {
		if result[teams[0].TeamID] == "" && result[teams[1].TeamID] != "" {
			result[teams[0].TeamID] = oppositeSide(result[teams[1].TeamID])
		}
		if result[teams[1].TeamID] == "" && result[teams[0].TeamID] != "" {
			result[teams[1].TeamID] = oppositeSide(result[teams[0].TeamID])
		}
	}
	return result
}

func canonicalTeamScore(ctx *models.DemoContext, teamID string, rosters canonicalRosterInfo) int {
	winners := make(map[int]string, ctx.CurrentRound)
	if ctx.ReplayData != nil {
		for _, round := range ctx.ReplayData.Rounds {
			winners[round.Round] = normalizeSide(round.Winner)
		}
	}
	for _, round := range ctx.MatchData.Rounds {
		if winners[round.Round] == "" || winners[round.Round] == "unknown" {
			winners[round.Round] = normalizeSide(round.Winner)
		}
	}
	score := 0
	for roundNumber, winnerSide := range winners {
		winnerTeam := canonicalTeamForSide(rosters, roundNumber, winnerSide)
		if winnerTeam != nil && *winnerTeam == teamID {
			score++
		}
	}
	return score
}

func canonicalTeamForSide(rosters canonicalRosterInfo, roundNumber int, side string) *string {
	teamID := teamForSideValue(rosters.sides[roundNumber], side)
	if teamID == "" {
		return nil
	}
	return &teamID
}

func teamForSideValue(assignments map[string]string, side string) string {
	for teamID, assignedSide := range assignments {
		if assignedSide == side {
			return teamID
		}
	}
	return ""
}

func canonicalStableScoresFromSideScores(
	ctScore, tScore int,
	assignments map[string]string,
) map[string]int {
	scores := make(map[string]int, len(assignments))
	for teamID, side := range assignments {
		switch side {
		case "ct":
			scores[teamID] = ctScore
		case "t":
			scores[teamID] = tScore
		}
	}
	return scores
}

func oppositeSide(side string) string {
	if side == "ct" {
		return "t"
	}
	if side == "t" {
		return "ct"
	}
	return "unknown"
}

func buildCanonicalMatch(ctx *models.DemoContext, matchID, playedAt string, durationSeconds, tickRate float64, rosters canonicalRosterInfo) models.CanonicalMatchExport {
	playedAtValue, _ := canonicalPlayedAt(playedAt)
	teams := append([]models.CanonicalTeam(nil), rosters.teams...)
	stableFinalScores := canonicalStableScoresFromSideScores(
		ctx.MatchData.CTScore,
		ctx.MatchData.TScore,
		rosters.sides[ctx.CurrentRound],
	)
	for index := range teams {
		if score, ok := stableFinalScores[teams[index].TeamID]; ok {
			teams[index].Score = score
		} else {
			teams[index].Score = canonicalTeamScore(ctx, teams[index].TeamID, rosters)
		}
	}
	var winnerTeamID *string
	if len(teams) == 2 && teams[0].Score != teams[1].Score {
		winner := teams[0].TeamID
		if teams[1].Score > teams[0].Score {
			winner = teams[1].TeamID
		}
		winnerTeamID = &winner
	}
	return models.CanonicalMatchExport{
		SchemaID: "stratai.match@1", MatchID: matchID,
		CanonicalMatch: models.CanonicalMatch{
			MapName: ctx.MatchData.MapName, PlayedAt: playedAtValue, TickRateHz: tickRate,
			DurationMS: int64(durationSeconds * 1000), RoundCount: ctx.CurrentRound,
			CTScore: ctx.MatchData.CTScore, TScore: ctx.MatchData.TScore,
			WinnerSide: normalizeSide(ctx.MatchData.Winner), WinnerTeamID: winnerTeamID, Teams: teams,
			CoordinateUnits: "source_world_units",
		},
	}
}

func buildCanonicalParticipants(ctx *models.DemoContext, matchID string, rosters canonicalRosterInfo) models.CanonicalParticipantsExport {
	names := make(map[uint64]string)
	add := func(steamID uint64, name string) {
		if steamID == 0 {
			return
		}
		if names[steamID] == "" || name != "" {
			names[steamID] = name
		}
	}
	for steamID, player := range ctx.MatchData.Players {
		add(steamID, player.Name)
	}
	for _, player := range ctx.AI_PlayersSummary {
		steamID, _ := strconv.ParseUint(player.SteamID, 10, 64)
		add(steamID, player.Name)
	}
	for _, event := range ctx.AI_CombatEvents {
		add(event.AttackerSteamID, event.AttackerName)
		add(event.VictimSteamID, event.VictimName)
	}
	for _, event := range ctx.AI_GrenadeEvents {
		add(event.ThrowerSteamID, event.Thrower)
		for _, player := range event.BlindedPlayers {
			add(player.SteamID, player.Name)
		}
		for _, player := range event.DamagedPlayers {
			add(player.SteamID, player.Name)
		}
	}
	if ctx.Objectives != nil {
		for _, event := range ctx.Objectives.Events() {
			add(event.Actor.SteamID, event.Actor.Name)
		}
	}
	for _, state := range ctx.AI_TrackingEventsWithRound {
		add(state.Event.PlayerSteamID, "")
	}
	if ctx.ReplayData != nil {
		for _, round := range ctx.ReplayData.Rounds {
			for _, frame := range round.Frames {
				for _, player := range frame.Players {
					add(player.SteamID, player.Name)
				}
			}
		}
	}

	steamIDs := make([]uint64, 0, len(names))
	for steamID := range names {
		steamIDs = append(steamIDs, steamID)
	}
	sort.Slice(steamIDs, func(i, j int) bool { return steamIDs[i] < steamIDs[j] })
	players := make([]models.CanonicalParticipant, 0, len(steamIDs))
	for _, steamID := range steamIDs {
		steamIDValue := strconv.FormatUint(steamID, 10)
		teamID := rosters.playerTeams[steamID]
		if teamID == "" {
			teamID = "unknown"
		}
		players = append(players, models.CanonicalParticipant{
			PlayerID: "steam:" + steamIDValue, SteamID: steamIDValue, DisplayName: names[steamID],
			TeamID: teamID,
		})
	}
	sort.Slice(players, func(i, j int) bool { return players[i].PlayerID < players[j].PlayerID })
	return models.CanonicalParticipantsExport{SchemaID: "stratai.participants@1", MatchID: matchID, Players: players}
}

func buildCanonicalRounds(
	ctx *models.DemoContext,
	matchID string,
	rosters canonicalRosterInfo,
	objectiveEvents []models.CanonicalObjectiveEvent,
) models.CanonicalRoundsExport {
	byRound := make(map[int]models.RoundData, len(ctx.MatchData.Rounds))
	for _, round := range ctx.MatchData.Rounds {
		byRound[round.Round] = round
	}
	bounds := make(map[int]models.ReplayRound)
	if ctx.ReplayData != nil {
		for _, round := range ctx.ReplayData.Rounds {
			bounds[round.Round] = round
		}
	}
	rounds := make([]models.CanonicalRound, 0, ctx.CurrentRound)
	for number := 1; number <= ctx.CurrentRound; number++ {
		round := models.CanonicalRound{
			RoundID:     canonicalRoundID(matchID, number),
			RoundNumber: number,
		}
		if sides := rosters.sides[number]; len(sides) > 0 {
			for _, team := range rosters.teams {
				if side := sides[team.TeamID]; side != "" {
					round.SideAssignments = append(round.SideAssignments, models.CanonicalSideAssignment{TeamID: team.TeamID, Side: side})
				}
			}
		}
		replayRound, hasReplayRound := bounds[number]
		if hasReplayRound {
			round.StartTick = intPointer(replayRound.StartTick)
			round.EndTick = intPointer(replayRound.EndTick)
			if replayRound.Winner != "" {
				round.WinnerSide = stringPointer(normalizeSide(replayRound.Winner))
			}
		}
		if ctx.Objectives != nil {
			if summary, exists := ctx.Objectives.RoundSummary(number); exists && summary.RoundStartTick >= 0 {
				round.StartTick = intPointer(summary.RoundStartTick)
			}
		}
		if result, ok := byRound[number]; ok {
			winnerSide := normalizeSide(result.Winner)
			if !hasReplayRound || replayRound.Winner == "" {
				round.WinnerSide = stringPointer(winnerSide)
			}
			round.WinReason, round.RawWinReasonCode = canonicalRoundEndReason(result.Reason)
			round.CTScoreAfter = intPointer(result.CTScore)
			round.TScoreAfter = intPointer(result.TScore)
			stableScores := canonicalStableScoresFromSideScores(result.CTScore, result.TScore, rosters.sides[number])
			for _, team := range rosters.teams {
				if score, exists := stableScores[team.TeamID]; exists {
					round.TeamScoresAfter = append(round.TeamScoresAfter, models.CanonicalTeamScore{
						TeamID: team.TeamID,
						Score:  score,
					})
				}
			}
		}
		round.Objective = summarizeCanonicalRoundObjective(number, round.RawWinReasonCode, objectiveEvents)
		round.BombPlanted = boolPointer(round.Objective.WasBombPlanted)
		round.BombSite = round.Objective.Site
		round.BombTick = round.Objective.PlantTick
		if round.WinnerTeamID == nil && round.WinnerSide != nil {
			round.WinnerTeamID = canonicalTeamForSide(rosters, number, *round.WinnerSide)
		}
		rounds = append(rounds, round)
	}
	return models.CanonicalRoundsExport{SchemaID: "stratai.rounds@2", MatchID: matchID, Rounds: rounds}
}

func summarizeCanonicalRoundObjective(
	roundNumber int,
	winReasonCode *int,
	events []models.CanonicalObjectiveEvent,
) models.CanonicalRoundObjective {
	summary := models.CanonicalRoundObjective{Outcome: "not_planted"}
	for index := range events {
		event := &events[index]
		if event.RoundNumber != roundNumber {
			continue
		}
		switch event.EventType {
		case "bomb_drop":
			summary.BombDrops++
		case "bomb_pickup":
			summary.BombPickups++
		case "bomb_plant_start":
			summary.PlantAttempts++
		case "bomb_plant_abort":
			summary.PlantAborts++
		case "bomb_plant":
			summary.WasBombPlanted = true
			summary.PlantEventID = stringPointer(event.EventID)
			summary.Site = event.Site
			summary.PlantTick = intPointer(event.Tick)
			summary.PlanterPlayerID = event.ActorPlayerID
		case "bomb_defuse_start":
			summary.DefuseAttempts++
		case "bomb_defuse_abort":
			summary.DefuseAborts++
		case "bomb_defuse":
			summary.Outcome = "defused"
			summary.ResolutionEventID = stringPointer(event.EventID)
			summary.ResolutionTick = intPointer(event.Tick)
			summary.ResolverPlayerID = event.ActorPlayerID
		case "bomb_explode":
			summary.Outcome = "exploded"
			summary.ResolutionEventID = stringPointer(event.EventID)
			summary.ResolutionTick = intPointer(event.Tick)
		}
	}
	if summary.WasBombPlanted && summary.ResolutionEventID == nil {
		summary.Outcome = "elimination_after_plant"
	}
	if !summary.WasBombPlanted && winReasonCode != nil && *winReasonCode == 12 {
		summary.Outcome = "time_expired"
	}
	return summary
}

func validateCanonicalOutcomeConsistency(
	match models.CanonicalMatchExport,
	rounds models.CanonicalRoundsExport,
	replay *models.ReplayData,
) error {
	if len(rounds.Rounds) != match.RoundCount {
		return fmt.Errorf("round count %d does not match %d", len(rounds.Rounds), match.RoundCount)
	}

	declaredScores := make(map[string]int, len(match.Teams))
	wins := make(map[string]int, len(match.Teams))
	previousScores := make(map[string]int, len(match.Teams))
	for _, team := range match.Teams {
		declaredScores[team.TeamID] = team.Score
		wins[team.TeamID] = 0
		previousScores[team.TeamID] = 0
	}
	replayWinners := make(map[int]string)
	if replay != nil {
		for _, round := range replay.Rounds {
			replayWinners[round.Round] = normalizeSide(round.Winner)
		}
	}

	for index, round := range rounds.Rounds {
		expectedNumber := index + 1
		if round.RoundNumber != expectedNumber || round.WinnerSide == nil || round.WinnerTeamID == nil {
			return fmt.Errorf("round %d has incomplete outcome", expectedNumber)
		}
		if replayWinner, exists := replayWinners[round.RoundNumber]; exists && replayWinner != *round.WinnerSide {
			return fmt.Errorf("round %d winner differs from replay", round.RoundNumber)
		}
		if _, exists := declaredScores[*round.WinnerTeamID]; !exists {
			return fmt.Errorf("round %d references unknown winning team", round.RoundNumber)
		}
		assignedSides := make(map[string]string, len(round.SideAssignments))
		for _, assignment := range round.SideAssignments {
			assignedSides[assignment.TeamID] = assignment.Side
		}
		if len(assignedSides) != len(declaredScores) || assignedSides[*round.WinnerTeamID] != *round.WinnerSide {
			return fmt.Errorf("round %d winner conflicts with side assignments", round.RoundNumber)
		}

		currentScores := make(map[string]int, len(round.TeamScoresAfter))
		for _, teamScore := range round.TeamScoresAfter {
			if _, duplicate := currentScores[teamScore.TeamID]; duplicate {
				return fmt.Errorf("round %d repeats team score", round.RoundNumber)
			}
			currentScores[teamScore.TeamID] = teamScore.Score
		}
		if len(currentScores) != len(declaredScores) {
			return fmt.Errorf("round %d has incomplete stable scores", round.RoundNumber)
		}
		for teamID := range declaredScores {
			expectedScore := previousScores[teamID]
			if teamID == *round.WinnerTeamID {
				expectedScore++
			}
			if currentScores[teamID] != expectedScore {
				return fmt.Errorf("round %d score transition is inconsistent for %s", round.RoundNumber, teamID)
			}
		}
		wins[*round.WinnerTeamID]++
		previousScores = currentScores
	}

	for teamID, score := range declaredScores {
		if wins[teamID] != score || previousScores[teamID] != score {
			return fmt.Errorf("final score is inconsistent for %s", teamID)
		}
	}
	return nil
}

func filterCanonicalCombatLedgerToRounds(events []combat.Event, rounds []models.CanonicalRound) []combat.Event {
	filtered := make([]combat.Event, 0, len(events))
	sequenceByRound := make(map[int]int)
	sequenceByTick := make(map[[2]int]int)
	for _, event := range events {
		if !canonicalTickWithinRound(rounds, event.Round, event.Tick) {
			continue
		}
		sequenceByRound[event.Round]++
		tickKey := [2]int{event.Round, event.Tick}
		sequenceByTick[tickKey]++
		event.SequenceInRound = sequenceByRound[event.Round]
		event.SequenceInTick = sequenceByTick[tickKey]
		filtered = append(filtered, event)
	}
	return filtered
}

func buildCanonicalCombatEvents(ctx *models.DemoContext, matchID string, rounds ...[]models.CanonicalRound) ([]models.CanonicalCombatEvent, error) {
	if ctx == nil || ctx.Combat == nil {
		return nil, fmt.Errorf("combat tracker unavailable")
	}
	ledger := ctx.Combat.Snapshot()
	if len(rounds) > 0 {
		ledger = filterCanonicalCombatLedgerToRounds(ledger, rounds[0])
	}
	eventIDs, shotIDs := canonicalCombatReferenceMaps(ledger, matchID)

	tickRate := 0.0
	if ctx.Parser != nil {
		tickRate = ctx.Parser.TickRate()
	}
	records := make([]models.CanonicalCombatEvent, 0, len(ledger))
	reactions := newCombatReactionProjection(ctx.AI_CombatEvents)
	for _, event := range ledger {
		record, err := canonicalCombatRecord(event, matchID, tickRate, eventIDs, shotIDs)
		if err != nil {
			return nil, err
		}
		reactions.apply(&record, event)
		records = append(records, record)
	}
	return records, nil
}

type combatReactionKey struct {
	round, tick   int
	actor, target uint64
	eventType     combat.EventType
	weapon        string
	damage        int
}

type combatReactionValue struct {
	reactionTime float64
	timeToDamage float64
}

type combatReactionProjection struct {
	values map[combatReactionKey][]combatReactionValue
	next   map[combatReactionKey]int
}

func newCombatReactionProjection(source []models.RawCombatEvent) *combatReactionProjection {
	projection := &combatReactionProjection{
		values: make(map[combatReactionKey][]combatReactionValue),
		next:   make(map[combatReactionKey]int),
	}
	for _, event := range source {
		eventType := combat.EventPlayerHurt
		damage := event.Damage
		if event.IsKill {
			eventType = combat.EventKill
			damage = 0
		}
		key := combatReactionKey{
			round: event.Round, tick: event.Tick, actor: event.AttackerSteamID, target: event.VictimSteamID,
			eventType: eventType, weapon: strings.ToLower(strings.TrimSpace(event.Weapon)), damage: damage,
		}
		projection.values[key] = append(projection.values[key], combatReactionValue{
			reactionTime: event.TimeToReaction,
			timeToDamage: event.TimeToDamage,
		})
	}
	for key := range projection.values {
		sort.Slice(projection.values[key], func(left, right int) bool {
			if projection.values[key][left].reactionTime != projection.values[key][right].reactionTime {
				return projection.values[key][left].reactionTime < projection.values[key][right].reactionTime
			}
			return projection.values[key][left].timeToDamage < projection.values[key][right].timeToDamage
		})
	}
	return projection
}

func (projection *combatReactionProjection) apply(record *models.CanonicalCombatEvent, event combat.Event) {
	if projection == nil || record == nil || event.Actor.Status != combat.AvailabilityObserved ||
		event.Target.Status != combat.AvailabilityObserved ||
		(event.Type != combat.EventPlayerHurt && event.Type != combat.EventKill) {
		return
	}
	damage := 0
	if event.Type == combat.EventPlayerHurt && event.HealthDamageTaken != nil {
		damage = *event.HealthDamageTaken
	}
	key := combatReactionKey{
		round: event.Round, tick: event.Tick, actor: event.Actor.ID, target: event.Target.ID,
		eventType: event.Type, weapon: strings.ToLower(strings.TrimSpace(event.Weapon.Name)), damage: damage,
	}
	index := projection.next[key]
	values := projection.values[key]
	if index >= len(values) {
		return
	}
	projection.next[key] = index + 1
	record.ReactionTimeMS = positiveFloatPointer(values[index].reactionTime)
	record.TimeToDamageMS = positiveFloatPointer(values[index].timeToDamage)
}

func canonicalCombatReferenceMaps(ledger []combat.Event, matchID string) (map[string]string, map[string]string) {
	eventIDs := make(map[string]string, len(ledger))
	shotIDs := make(map[string]string)
	shotSequence := make(map[int]int)
	for _, event := range ledger {
		eventIDs[event.LocalID] = canonicalEventID(matchID, "combat", event.Round, event.Tick, event.SequenceInTick)
		if event.Type == combat.EventWeaponFire {
			shotSequence[event.Round]++
			shotIDs[event.ShotID] = fmt.Sprintf("%s:shot:%03d:%06d", matchID, event.Round, shotSequence[event.Round])
		}
	}
	return eventIDs, shotIDs
}

func canonicalCombatRecord(
	event combat.Event,
	matchID string,
	tickRate float64,
	eventIDs map[string]string,
	shotIDs map[string]string,
) (models.CanonicalCombatEvent, error) {
	actorID, actorSide := canonicalCombatPlayer(event.Actor)
	targetID, targetSide := canonicalCombatPlayer(event.Target)
	assisterID, assisterSide := canonicalCombatPlayer(event.Assister)
	record := models.CanonicalCombatEvent{
		SchemaID: "stratai.combat_event@2", MatchID: matchID, EventID: eventIDs[event.LocalID],
		RoundID: canonicalRoundID(matchID, event.Round), RoundNumber: event.Round, Tick: event.Tick,
		SequenceInTick: event.SequenceInTick, SequenceInRound: event.SequenceInRound,
		EventType: string(event.Type), Source: event.Source, SourceEventIDs: make([]string, 0, len(event.SourceEventIDs)),
		TickStatus: string(combat.AvailabilityObserved), SubtickStatus: string(combat.AvailabilityUnavailable),
		TimeSecondsStatus: string(combat.AvailabilityUnavailable), TimeSecondsSource: combat.SourceUnavailable,
		ActorPlayerID: actorID, ActorSide: actorSide, ActorStatus: combatAvailability(event.Actor.Status), ActorSource: availabilitySource(event.Actor.Source),
		TargetPlayerID: targetID, TargetSide: targetSide, TargetStatus: combatAvailability(event.Target.Status), TargetSource: availabilitySource(event.Target.Source),
		AssisterPlayerID: assisterID, AssisterSide: assisterSide, AssisterStatus: combatAvailability(event.Assister.Status), AssisterSource: availabilitySource(event.Assister.Source),
		Relation: string(event.Relation), WeaponStatus: combatAvailability(event.Weapon.Status), WeaponSource: availabilitySource(event.Weapon.Source),
		WeaponIsUtility: cloneCanonicalBool(event.Weapon.IsUtility),
		ActorPosition:   canonicalCombatVector(event.Actor.Position), ActorPositionStatus: combatAvailability(event.Actor.PositionStatus), ActorPositionSource: availabilitySource(event.Actor.PositionSource),
		TargetPosition: canonicalCombatVector(event.Target.Position), TargetPositionStatus: combatAvailability(event.Target.PositionStatus), TargetPositionSource: availabilitySource(event.Target.PositionSource),
		CorrelationStatus: string(event.CorrelationStatus), CorrelationSource: availabilitySource(event.CorrelationSource),
		ShotResultStatus: combatAvailability(event.ShotResultStatus), ShotResultSource: availabilitySource(event.ShotResultSource),
		ShotResultAvailabilityTick: cloneCanonicalInt(event.ShotResultAvailabilityTick),
		ViewYaw:                    cloneCanonicalFloat(event.ViewYaw), ViewPitch: cloneCanonicalFloat(event.ViewPitch),
		ImpactPosition: canonicalCombatVector(event.ImpactPosition), ImpactPositionStatus: combatAvailability(event.ImpactPositionStatus), ImpactPositionSource: availabilitySource(event.ImpactPositionSource),
		BulletDistanceWorldUnits: cloneCanonicalFloat(event.BulletDistance), DamageDirection: canonicalCombatVector(event.DamageDirection),
		PenetratedObjects: cloneCanonicalInt(event.PenetratedObjects), NoScope: cloneCanonicalBool(event.NoScope), AttackerInAir: cloneCanonicalBool(event.AttackerInAir),
		ThroughSmoke: cloneCanonicalBool(event.ThroughSmoke), AttackerBlind: cloneCanonicalBool(event.AttackerBlind), KillDistanceWorldUnits: cloneCanonicalFloat(event.KillDistance),
		HealthDamage: cloneCanonicalInt(event.HealthDamage), HealthDamageTaken: cloneCanonicalInt(event.HealthDamageTaken),
		ArmorDamage: cloneCanonicalInt(event.ArmorDamage), ArmorDamageTaken: cloneCanonicalInt(event.ArmorDamageTaken),
		HealthBefore: cloneCanonicalInt(event.HealthBefore), HealthAfter: cloneCanonicalInt(event.HealthAfter),
		ArmorBefore: cloneCanonicalInt(event.ArmorBefore), ArmorAfter: cloneCanonicalInt(event.ArmorAfter),
		DamageStatus: combatAvailability(event.DamageStatus), DamageSource: availabilitySource(event.DamageSource),
		Hitgroup: cloneCanonicalString(event.Hitgroup), HitgroupStatus: combatAvailability(event.HitgroupStatus), HitgroupSource: availabilitySource(event.HitgroupSource),
		IsHeadshot: cloneCanonicalBool(event.IsHeadshot), IsKill: event.IsKill, AssistedFlash: cloneCanonicalBool(event.AssistedFlash),
		ReloadPhase: cloneCanonicalString(event.ReloadPhase), ReloadEndTick: cloneCanonicalInt(event.ReloadEndTick), ReloadEndStatus: combatAvailability(event.ReloadEndStatus),
		PreviousWeapon: cloneCanonicalString(event.PreviousWeapon), PreviousWeaponStatus: combatAvailability(event.PreviousWeaponStatus), IsWeaponSwitch: cloneCanonicalBool(event.IsWeaponSwitch),
		AmmoInMagazine: cloneCanonicalInt(event.Ammo.InMagazine), AmmoReserve: cloneCanonicalInt(event.Ammo.Reserve),
		AmmoStatus: combatAvailability(event.Ammo.Status), AmmoSource: availabilitySource(event.Ammo.Source),
	}
	if tickRate > 0 && !math.IsNaN(tickRate) && !math.IsInf(tickRate, 0) {
		seconds := float64(event.Tick) / tickRate
		record.TimeSeconds = &seconds
		record.TimeSecondsStatus = string(combat.AvailabilityDerived)
		record.TimeSecondsSource = "tick_divided_by_tick_rate"
	}
	if event.Weapon.Status == combat.AvailabilityObserved {
		record.Weapon = cloneCanonicalString(&event.Weapon.Name)
	}
	if event.ShotID != "" {
		shotID, ok := shotIDs[event.ShotID]
		if !ok {
			return models.CanonicalCombatEvent{}, fmt.Errorf("combat event %s references unknown shot %s", event.LocalID, event.ShotID)
		}
		record.ShotID = &shotID
	}
	if event.Type == combat.EventWeaponFire && event.ShotResult != combat.ShotUnavailable {
		result := string(event.ShotResult)
		record.ShotResult = &result
	}
	for _, sourceID := range event.SourceEventIDs {
		canonicalID, ok := eventIDs[sourceID]
		if !ok {
			return models.CanonicalCombatEvent{}, fmt.Errorf("combat event %s references unknown source %s", event.LocalID, sourceID)
		}
		record.SourceEventIDs = append(record.SourceEventIDs, canonicalID)
	}
	if _, err := json.Marshal(record); err != nil {
		return models.CanonicalCombatEvent{}, fmt.Errorf("encode combat event %s: %w", event.LocalID, err)
	}
	return record, nil
}

func canonicalCombatPlayer(player combat.PlayerRef) (*string, *string) {
	if player.Status != combat.AvailabilityObserved || player.ID == 0 {
		return nil, nil
	}
	playerID := canonicalPlayerID(player.ID)
	var side *string
	if normalized := normalizeSide(player.Side); normalized != "" {
		side = &normalized
	}
	return &playerID, side
}

func canonicalCombatVector(vector *combat.Vector) *models.CanonicalVector {
	if vector == nil {
		return nil
	}
	return &models.CanonicalVector{X: vector.X, Y: vector.Y, Z: vector.Z}
}

func combatAvailability(value combat.Availability) string {
	if value == "" {
		return string(combat.AvailabilityUnavailable)
	}
	return string(value)
}

func availabilitySource(source string) string {
	if source == "" {
		return combat.SourceUnavailable
	}
	return source
}

func cloneCanonicalInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneCanonicalFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneCanonicalBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneCanonicalString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func filterCanonicalUtilityThrowsToRounds(throws []utility.Throw, rounds []models.CanonicalRound) []utility.Throw {
	filtered := make([]utility.Throw, 0, len(throws))
	sequenceByRound := make(map[int]int)
	for _, throw := range throws {
		if throw.Launch.Tick.Status == utility.AvailabilityObserved &&
			!canonicalTickWithinRound(rounds, throw.Round, throw.Launch.Tick.Tick) {
			continue
		}
		sequenceByRound[throw.Round]++
		throw.Sequence = sequenceByRound[throw.Round]
		throw.ID = fmt.Sprintf("r%d-u%04d", throw.Round, throw.Sequence)
		filtered = append(filtered, throw)
	}
	return filtered
}

func canonicalTickWithinRound(rounds []models.CanonicalRound, roundNumber, tick int) bool {
	for _, round := range rounds {
		if round.RoundNumber != roundNumber {
			continue
		}
		if round.StartTick != nil && tick < *round.StartTick {
			return false
		}
		if round.EndTick != nil && tick > *round.EndTick {
			return false
		}
		return true
	}
	return true
}

func buildCanonicalUtilityEvents(ctx *models.DemoContext, matchID string, rounds ...[]models.CanonicalRound) ([]models.CanonicalUtilityEvent, error) {
	if ctx.Utilities == nil {
		return nil, fmt.Errorf("utility tracker is unavailable")
	}
	throws := ctx.Utilities.Snapshot()
	if len(rounds) > 0 {
		throws = filterCanonicalUtilityThrowsToRounds(throws, rounds[0])
	}
	return projectCanonicalUtilityThrows(matchID, throws), nil
}

func projectCanonicalUtilityThrows(matchID string, source []utility.Throw) []models.CanonicalUtilityEvent {
	throws := append([]utility.Throw(nil), source...)
	sort.SliceStable(throws, func(i, j int) bool {
		if throws[i].Round != throws[j].Round {
			return throws[i].Round < throws[j].Round
		}
		if throws[i].Sequence != throws[j].Sequence {
			return throws[i].Sequence < throws[j].Sequence
		}
		return throws[i].ID < throws[j].ID
	})
	records := make([]models.CanonicalUtilityEvent, 0, len(throws))
	for _, throw := range throws {
		affected, damageSummary := projectCanonicalUtilityEffects(throw)
		flashSummary := canonicalUtilityFlashSummary(affected)
		record := models.CanonicalUtilityEvent{
			SchemaID:           "stratai.utility_event@2",
			MatchID:            matchID,
			EventID:            matchID + ":utility:" + throw.ID,
			SourceThrowID:      throw.ID,
			SourceEntityStatus: string(throw.EntityStatus),
			SourceEntitySource: throw.EntitySource,
			RoundID:            canonicalRoundID(matchID, throw.Round),
			RoundNumber:        throw.Round,
			SequenceInRound:    throw.Sequence,
			EventType:          "utility_throw",
			UtilityType:        string(throw.Type),
			UtilityTypeStatus:  canonicalUtilityTypeStatus(throw),
			UtilityTypeSource:  throw.TypeSource,
			ThrowerStatus:      string(throw.Actor.Status),
			ThrowerSource:      throw.Actor.Source,
			Correlation:        canonicalUtilityCorrelation(throw.Lifecycle.Correlation),
			Launch:             projectCanonicalUtilityLaunch(throw.Launch),
			Trajectory:         projectCanonicalUtilityTrajectory(throw.Trajectory),
			Lifecycle:          projectCanonicalUtilityLifecycle(throw),
			AffectedPlayers:    affected,
			FlashSummary:       flashSummary,
			DamageSummary:      damageSummary,
			Details:            projectLegacyUtilityDetails(throw),
		}
		if throw.EntityStatus == utility.AvailabilityObserved {
			record.SourceEntity = &models.CanonicalUtilityEntityReference{
				RoundNumber: throw.Round,
				EntityID:    throw.SourceEntityID,
				Generation:  throw.SourceEntityGeneration,
			}
		}
		if throw.Actor.Status == utility.AvailabilityObserved && throw.Actor.ID != 0 {
			playerID := canonicalPlayerID(throw.Actor.ID)
			record.ThrowerPlayerID = &playerID
		}
		if side := normalizeSide(throw.Actor.Side); throw.Actor.Status == utility.AvailabilityObserved && side != "unknown" {
			record.ThrowerSide = &side
		}
		records = append(records, record)
	}
	return records
}

type canonicalUtilityDurationAggregate struct {
	count    int
	totalMS  float64
	complete bool
}

func canonicalUtilityFlashSummary(affected []models.CanonicalAffectedPlayer) models.CanonicalUtilityFlashSummary {
	total := canonicalUtilityDurationAggregate{complete: true}
	buckets := map[string]*canonicalUtilityDurationAggregate{
		string(utility.RelationEnemy):    {complete: true},
		string(utility.RelationTeammate): {complete: true},
		string(utility.RelationSelf):     {complete: true},
		string(utility.RelationUnknown):  {complete: true},
	}
	summary := models.CanonicalUtilityFlashSummary{}
	for _, player := range affected {
		if player.BlindDuration.Status == string(utility.AvailabilityNotApplicable) {
			continue
		}
		summary.PlayersTotal++
		bucket := buckets[player.Relation]
		if bucket == nil {
			bucket = buckets[string(utility.RelationUnknown)]
		}
		bucket.count++
		total.count++
		if player.BlindDuration.Status != string(utility.AvailabilityObserved) || player.BlindDuration.Milliseconds == nil {
			bucket.complete = false
			total.complete = false
		} else {
			bucket.totalMS += *player.BlindDuration.Milliseconds
			total.totalMS += *player.BlindDuration.Milliseconds
		}
		switch player.Relation {
		case string(utility.RelationEnemy):
			summary.EnemiesFlashed++
		case string(utility.RelationTeammate):
			summary.TeammatesFlashed++
		case string(utility.RelationSelf):
			summary.SelfFlashed++
		default:
			summary.UnknownFlashed++
		}
	}
	summary.TotalDurationMS = canonicalUtilityAggregateDuration(total)
	summary.EnemyDurationMS = canonicalUtilityAggregateDuration(*buckets[string(utility.RelationEnemy)])
	summary.TeammateDurationMS = canonicalUtilityAggregateDuration(*buckets[string(utility.RelationTeammate)])
	summary.SelfDurationMS = canonicalUtilityAggregateDuration(*buckets[string(utility.RelationSelf)])
	summary.UnknownDurationMS = canonicalUtilityAggregateDuration(*buckets[string(utility.RelationUnknown)])
	return summary
}

func canonicalUtilityAggregateDuration(aggregate canonicalUtilityDurationAggregate) *float64 {
	if aggregate.count > 0 && !aggregate.complete {
		return nil
	}
	value := aggregate.totalMS
	return &value
}

func canonicalUtilityTypeStatus(source utility.Throw) string {
	if source.Type == utility.TypeUnknown || source.TypeSource == utility.SourceUnavailable {
		return string(utility.AvailabilityUnavailable)
	}
	return string(utility.AvailabilityObserved)
}

func projectCanonicalUtilityLaunch(source utility.ThrowSnapshot) models.CanonicalUtilityLaunch {
	result := models.CanonicalUtilityLaunch{
		TickStatus:                string(source.Tick.Status),
		TickSource:                source.Tick.Source,
		Position:                  canonicalUtilityVectorObservation(source.Position),
		View:                      canonicalUtilityViewObservation(source.View),
		ThrowerVelocity:           canonicalUtilityVelocityObservation(source.ThrowerVelocity),
		ProjectileInitialVelocity: canonicalUtilityVelocityObservation(source.ProjectileInitialVelocity),
		Stance:                    canonicalUtilityStanceObservation(source.Stance),
		Area:                      canonicalUtilityStringObservation(source.Area),
	}
	if source.Tick.Status == utility.AvailabilityObserved {
		tick := source.Tick.Tick
		result.Tick = &tick
	}
	return result
}

func projectCanonicalUtilityTrajectory(source utility.Trajectory) models.CanonicalUtilityTrajectory {
	sourceSamples := append([]utility.TrajectorySample(nil), source.Samples...)
	sort.Slice(sourceSamples, func(i, j int) bool {
		left, right := sourceSamples[i], sourceSamples[j]
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		if left.Position != right.Position {
			if left.Position.X != right.Position.X {
				return left.Position.X < right.Position.X
			}
			if left.Position.Y != right.Position.Y {
				return left.Position.Y < right.Position.Y
			}
			return left.Position.Z < right.Position.Z
		}
		return left.Source < right.Source
	})
	samples := make([]models.CanonicalUtilityTrajectorySample, 0, len(sourceSamples))
	for _, sample := range sourceSamples {
		samples = append(samples, models.CanonicalUtilityTrajectorySample{
			Tick: sample.Tick, Position: canonicalUtilityVector(sample.Position), Source: sample.Source,
		})
	}
	sourceBounces := append([]utility.BounceObservation(nil), source.Bounces...)
	sort.Slice(sourceBounces, func(i, j int) bool {
		left, right := sourceBounces[i], sourceBounces[j]
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		if left.Number != right.Number {
			return left.Number < right.Number
		}
		if left.PositionStatus != right.PositionStatus {
			return left.PositionStatus < right.PositionStatus
		}
		if left.Position != right.Position {
			if left.Position.X != right.Position.X {
				return left.Position.X < right.Position.X
			}
			if left.Position.Y != right.Position.Y {
				return left.Position.Y < right.Position.Y
			}
			return left.Position.Z < right.Position.Z
		}
		return left.Source < right.Source
	})
	bounces := make([]models.CanonicalUtilityBounceObservation, 0, len(sourceBounces))
	for _, bounce := range sourceBounces {
		observation := models.CanonicalUtilityBounceObservation{
			Tick: bounce.Tick, PositionStatus: string(bounce.PositionStatus), Number: bounce.Number, Source: bounce.Source,
		}
		if bounce.PositionStatus == utility.AvailabilityObserved {
			position := canonicalUtilityVector(bounce.Position)
			observation.Position = &position
		}
		bounces = append(bounces, observation)
	}
	result := models.CanonicalUtilityTrajectory{
		BounceStatus: string(source.BounceStatus),
		BounceSource: source.BounceSource,
		Samples:      samples,
		Bounces:      bounces,
		Status:       string(source.Status),
		Source:       source.Source,
	}
	if source.BounceStatus == utility.AvailabilityObserved {
		bounceCount := source.BounceCount
		result.BounceCount = &bounceCount
	}
	return result
}

func projectCanonicalUtilityLifecycle(source utility.Throw) models.CanonicalUtilityLifecycle {
	lifecycle := source.Lifecycle
	endReason := models.CanonicalUtilityStringObservation{
		Status: string(utility.AvailabilityUnavailable), Source: utility.SourceUnavailable,
	}
	if lifecycle.EndReason != utility.EndReasonUnavailable {
		value := string(lifecycle.EndReason)
		endReason = models.CanonicalUtilityStringObservation{
			Value: &value, Status: string(utility.AvailabilityObserved), Source: lifecycle.EndReasonSource,
		}
	}
	return models.CanonicalUtilityLifecycle{
		Status:                string(lifecycle.Status),
		Detonation:            canonicalUtilityMomentObservation(lifecycle.Detonation),
		EffectStart:           canonicalUtilityMomentObservation(lifecycle.EffectStart),
		Expiration:            canonicalUtilityMomentObservation(lifecycle.Expiration),
		Destroy:               canonicalUtilityMomentObservation(lifecycle.Destroy),
		Extinguish:            canonicalUtilityMomentObservation(lifecycle.Extinguish),
		Area:                  canonicalUtilityStringObservation(lifecycle.Area),
		ExtinguishedByThrowID: canonicalUtilityStringObservation(lifecycle.ExtinguishedByThrowID),
		ExtinguishCorrelation: canonicalUtilityCorrelation(lifecycle.ExtinguishAttribution),
		Duration:              canonicalUtilityDurationObservation(lifecycle.Duration),
		EndReason:             endReason,
	}
}

type canonicalUtilityAffectedBuilder struct {
	key      string
	victim   utility.PlayerRef
	relation utility.Relation
	flash    *utility.FlashEffect
	damage   []utility.DamageEffect
}

func projectCanonicalUtilityEffects(source utility.Throw) ([]models.CanonicalAffectedPlayer, models.CanonicalUtilityDamageSummary) {
	builders := make(map[string]*canonicalUtilityAffectedBuilder)
	flashes := append([]utility.FlashEffect(nil), source.Flashes...)
	sort.Slice(flashes, func(i, j int) bool { return canonicalUtilityFlashLess(flashes[i], flashes[j]) })
	for index := range flashes {
		effect := flashes[index]
		key := canonicalUtilityVictimKey(effect.Victim, "flash", index)
		builder := canonicalUtilityEffectBuilder(builders, key, effect.Victim, effect.Relation)
		copy := effect
		builder.flash = &copy
	}
	damage := append([]utility.DamageEffect(nil), source.Damage...)
	sort.Slice(damage, func(i, j int) bool { return canonicalUtilityDamageLess(damage[i], damage[j]) })
	for index, effect := range damage {
		key := canonicalUtilityVictimKey(effect.Victim, "damage", index)
		builder := canonicalUtilityEffectBuilder(builders, key, effect.Victim, effect.Relation)
		builder.damage = append(builder.damage, effect)
	}
	ordered := make([]*canonicalUtilityAffectedBuilder, 0, len(builders))
	for _, builder := range builders {
		sort.SliceStable(builder.damage, func(i, j int) bool {
			left, right := builder.damage[i], builder.damage[j]
			if left.Tick != right.Tick {
				return left.Tick < right.Tick
			}
			if left.HealthDamage != right.HealthDamage {
				return left.HealthDamage < right.HealthDamage
			}
			if left.ArmorDamage != right.ArmorDamage {
				return left.ArmorDamage < right.ArmorDamage
			}
			return left.Source < right.Source
		})
		ordered = append(ordered, builder)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].key < ordered[j].key })
	affected := make([]models.CanonicalAffectedPlayer, 0, len(ordered))
	summary := models.CanonicalUtilityDamageSummary{}
	for _, builder := range ordered {
		player := projectCanonicalAffectedPlayer(*builder)
		affected = append(affected, player)
		addCanonicalUtilityDamageSummary(&summary, player)
	}
	return affected, summary
}

func canonicalUtilityFlashLess(left, right utility.FlashEffect) bool {
	if left.Tick != right.Tick {
		return left.Tick < right.Tick
	}
	if left.Victim.ID != right.Victim.ID {
		return left.Victim.ID < right.Victim.ID
	}
	if left.Relation != right.Relation {
		return left.Relation < right.Relation
	}
	if left.Duration.Value != right.Duration.Value {
		return left.Duration.Value < right.Duration.Value
	}
	return left.Source < right.Source
}

func canonicalUtilityDamageLess(left, right utility.DamageEffect) bool {
	if left.Tick != right.Tick {
		return left.Tick < right.Tick
	}
	if left.Victim.ID != right.Victim.ID {
		return left.Victim.ID < right.Victim.ID
	}
	if left.HealthDamage != right.HealthDamage {
		return left.HealthDamage < right.HealthDamage
	}
	if left.ArmorDamage != right.ArmorDamage {
		return left.ArmorDamage < right.ArmorDamage
	}
	if left.Kill != right.Kill {
		return !left.Kill
	}
	if left.Relation != right.Relation {
		return left.Relation < right.Relation
	}
	return left.Source < right.Source
}

func canonicalUtilityEffectBuilder(
	builders map[string]*canonicalUtilityAffectedBuilder,
	key string,
	victim utility.PlayerRef,
	relation utility.Relation,
) *canonicalUtilityAffectedBuilder {
	if existing := builders[key]; existing != nil {
		if existing.victim.Status != utility.AvailabilityObserved && victim.Status == utility.AvailabilityObserved {
			existing.victim = victim
		}
		if existing.relation == utility.RelationUnknown && relation != utility.RelationUnknown {
			existing.relation = relation
		}
		return existing
	}
	builder := &canonicalUtilityAffectedBuilder{key: key, victim: victim, relation: relation}
	builders[key] = builder
	return builder
}

func projectCanonicalAffectedPlayer(source canonicalUtilityAffectedBuilder) models.CanonicalAffectedPlayer {
	playerSource := source.victim.Source
	if len(source.damage) > 0 {
		playerSource = utility.SourcePlayerHurt
	} else if source.flash != nil {
		playerSource = utility.SourcePlayerFlashed
	}
	player := models.CanonicalAffectedPlayer{
		PlayerStatus: string(source.victim.Status),
		PlayerSource: playerSource,
		Relation:     string(source.relation),
		IsEnemy:      source.relation == utility.RelationEnemy,
		IsSelf:       source.relation == utility.RelationSelf,
		BlindDuration: models.CanonicalUtilityDurationObservation{
			Status: string(utility.AvailabilityNotApplicable), Source: utility.SourceUnavailable,
		},
		BlindCorrelation: models.CanonicalUtilityCorrelation{
			Status: string(utility.CorrelationUnavailable), Source: utility.SourceUnavailable,
		},
		DamageEvents: make([]models.CanonicalUtilityDamageEffect, 0, len(source.damage)),
	}
	if source.victim.Status == utility.AvailabilityObserved && source.victim.ID != 0 {
		playerID := canonicalPlayerID(source.victim.ID)
		player.PlayerID = &playerID
	}
	if side := normalizeSide(source.victim.Side); source.victim.Status == utility.AvailabilityObserved && side != "unknown" {
		player.Side = &side
	}
	if source.flash != nil {
		player.BlindDuration = canonicalUtilityDurationObservation(source.flash.Duration)
		player.BlindCorrelation = canonicalUtilityCorrelation(source.flash.Correlation)
	}
	if len(source.damage) == 0 {
		return player
	}
	healthDamage, armorDamage, isKill := 0, 0, false
	for _, effect := range source.damage {
		healthDamage += effect.HealthDamage
		armorDamage += effect.ArmorDamage
		isKill = isKill || effect.Kill
		player.DamageEvents = append(player.DamageEvents, models.CanonicalUtilityDamageEffect{
			Tick: effect.Tick, HealthDamage: effect.HealthDamage, ArmorDamage: effect.ArmorDamage,
			IsKill: effect.Kill, Source: effect.Source, Correlation: canonicalUtilityCorrelation(effect.Correlation),
		})
	}
	player.Damage = &healthDamage
	player.ArmorDamage = &armorDamage
	player.IsKill = &isKill
	return player
}

func addCanonicalUtilityDamageSummary(summary *models.CanonicalUtilityDamageSummary, player models.CanonicalAffectedPlayer) {
	if player.Damage == nil || player.ArmorDamage == nil {
		return
	}
	summary.TotalDamage += *player.Damage
	summary.TotalArmorDamage += *player.ArmorDamage
	hasDamage := *player.Damage > 0 || *player.ArmorDamage > 0
	switch player.Relation {
	case string(utility.RelationEnemy):
		summary.EnemyDamage += *player.Damage
		summary.EnemyArmorDamage += *player.ArmorDamage
		if hasDamage {
			summary.EnemiesDamaged++
		}
		if player.IsKill != nil && *player.IsKill {
			summary.EnemyKills++
		}
	case string(utility.RelationTeammate):
		summary.TeammateDamage += *player.Damage
		summary.TeammateArmorDamage += *player.ArmorDamage
		if hasDamage {
			summary.TeammatesDamaged++
		}
		if player.IsKill != nil && *player.IsKill {
			summary.TeammateKills++
		}
	case string(utility.RelationSelf):
		summary.SelfDamage += *player.Damage
		summary.SelfArmorDamage += *player.ArmorDamage
		summary.SelfDamaged = hasDamage
		if player.IsKill != nil && *player.IsKill {
			summary.SelfKills++
		}
	default:
		summary.UnknownDamage += *player.Damage
		summary.UnknownArmorDamage += *player.ArmorDamage
		if hasDamage {
			summary.UnknownPlayersDamaged++
		}
		if player.IsKill != nil && *player.IsKill {
			summary.UnknownKills++
		}
	}
}

func projectLegacyUtilityDetails(source utility.Throw) models.AI_GrenadeEvent {
	event := models.AI_GrenadeEvent{
		Round:          source.Round,
		Type:           legacyUtilityType(source.Type),
		Thrower:        source.Actor.Name,
		ThrowerSteamID: source.Actor.ID,
		ThrowerSide:    legacyUtilitySide(source.Actor.Side),
		TickThrow:      source.Launch.Tick.Tick,
		DidBounce:      source.Trajectory.BounceStatus == utility.AvailabilityObserved && source.Trajectory.BounceCount > 0,
		Extinguished:   source.Lifecycle.EndReason == utility.EndReasonSmokeExtinguished,
	}
	if source.Launch.Position.Status == utility.AvailabilityObserved {
		event.StartPosition = canonicalLegacyUtilityVector(source.Launch.Position.Value)
	}
	if source.Launch.View.Status == utility.AvailabilityObserved {
		event.ThrowViewVector = canonicalLegacyUtilityVector(source.Launch.View.Vector)
	}
	if source.Launch.Area.Status == utility.AvailabilityObserved {
		event.ThrowerAreaName = source.Launch.Area.Value
	}
	if source.Lifecycle.Area.Status == utility.AvailabilityObserved {
		event.LandArea = source.Lifecycle.Area.Value
	}
	if moment, ok := canonicalUtilityEffectMoment(source.Lifecycle); ok {
		event.TickExplode = moment.Tick
		event.EndPosition = canonicalLegacyUtilityVector(moment.Position)
	}
	if source.Lifecycle.Duration.Status == utility.AvailabilityObserved {
		event.Duration = source.Lifecycle.Duration.Value
	}
	projectLegacyFlashEffects(source, &event)
	projectLegacyDamageEffects(source, &event)
	return event
}

func projectLegacyFlashEffects(source utility.Throw, target *models.AI_GrenadeEvent) {
	effects := append([]utility.FlashEffect(nil), source.Flashes...)
	sort.SliceStable(effects, func(i, j int) bool {
		if effects[i].Victim.ID != effects[j].Victim.ID {
			return effects[i].Victim.ID < effects[j].Victim.ID
		}
		return effects[i].Tick < effects[j].Tick
	})
	for _, effect := range effects {
		if effect.Victim.Status == utility.AvailabilityObserved {
			duration := float32(0)
			if effect.Duration.Status == utility.AvailabilityObserved {
				duration = float32(effect.Duration.Value)
			}
			target.BlindedPlayers = append(target.BlindedPlayers, models.AI_BlindedPlayer{
				SteamID: effect.Victim.ID, Name: effect.Victim.Name, Duration: duration,
				Team: legacyUtilitySide(effect.Victim.Side), IsEnemy: effect.Relation == utility.RelationEnemy,
				IsSelf: effect.Relation == utility.RelationSelf, Relation: string(effect.Relation),
			})
		}
		switch effect.Relation {
		case utility.RelationEnemy:
			target.EnemiesBlinded++
		case utility.RelationTeammate, utility.RelationSelf:
			target.AlliesBlinded++
		}
	}
}

func projectLegacyDamageEffects(source utility.Throw, target *models.AI_GrenadeEvent) {
	builders := make(map[uint64]*models.AI_DamagedPlayer)
	for _, effect := range source.Damage {
		damage := max(0, effect.HealthDamage)
		armorDamage := max(0, effect.ArmorDamage)
		target.DamageDealt += damage
		target.ArmorDamageDealt += armorDamage
		switch effect.Relation {
		case utility.RelationEnemy:
			target.EnemyDamage += damage
			target.EnemyArmorDamage += armorDamage
		case utility.RelationTeammate:
			target.FriendlyDamage += damage
			target.FriendlyArmorDamage += armorDamage
		case utility.RelationSelf:
			target.SelfDamage += damage
			target.SelfArmorDamage += armorDamage
		}
		if effect.Victim.Status != utility.AvailabilityObserved {
			continue
		}
		player := builders[effect.Victim.ID]
		if player == nil {
			player = &models.AI_DamagedPlayer{
				SteamID: effect.Victim.ID, Name: effect.Victim.Name,
				Team:    legacyUtilitySide(effect.Victim.Side),
				IsEnemy: effect.Relation == utility.RelationEnemy, IsSelf: effect.Relation == utility.RelationSelf,
				Relation: string(effect.Relation),
			}
			builders[effect.Victim.ID] = player
		}
		player.Damage += damage
		player.ArmorDamage += armorDamage
		player.IsKill = player.IsKill || effect.Kill
	}
	ids := make([]uint64, 0, len(builders))
	for steamID := range builders {
		ids = append(ids, steamID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, steamID := range ids {
		player := builders[steamID]
		target.DamagedPlayers = append(target.DamagedPlayers, *player)
		hasDamage := player.Damage > 0 || player.ArmorDamage > 0
		switch utility.Relation(player.Relation) {
		case utility.RelationSelf:
			target.SelfDamaged = hasDamage
		case utility.RelationEnemy:
			if hasDamage {
				target.EnemiesDamaged++
			}
			if player.IsKill {
				target.Kills++
			}
		case utility.RelationTeammate:
			if hasDamage {
				target.AlliesDamaged++
			}
		}
	}
}

func canonicalUtilityVictimKey(victim utility.PlayerRef, domain string, index int) string {
	if victim.ID != 0 {
		return fmt.Sprintf("player:%020d", victim.ID)
	}
	return fmt.Sprintf("unavailable:%s:%09d", domain, index)
}

func canonicalUtilityVector(source utility.Vector) models.CanonicalVector {
	return models.CanonicalVector{X: source.X, Y: source.Y, Z: source.Z}
}

func canonicalLegacyUtilityVector(source utility.Vector) models.AI_Vector {
	return models.AI_Vector{X: source.X, Y: source.Y, Z: source.Z}
}

func canonicalUtilityVectorObservation(source utility.VectorObservation) models.CanonicalUtilityVectorObservation {
	result := models.CanonicalUtilityVectorObservation{Status: string(source.Status), Source: source.Source}
	if source.Status == utility.AvailabilityObserved {
		value := canonicalUtilityVector(source.Value)
		result.Value = &value
	}
	return result
}

func canonicalUtilityViewObservation(source utility.ViewObservation) models.CanonicalUtilityViewObservation {
	result := models.CanonicalUtilityViewObservation{Status: string(source.Status), Source: source.Source}
	if source.Status == utility.AvailabilityObserved {
		yaw, pitch := source.Yaw, source.Pitch
		vector := canonicalUtilityVector(source.Vector)
		result.YawDeg, result.PitchDeg, result.Vector = &yaw, &pitch, &vector
	}
	return result
}

func canonicalUtilityVelocityObservation(source utility.VelocityObservation) models.CanonicalUtilityVelocityObservation {
	result := models.CanonicalUtilityVelocityObservation{Status: string(source.Status), Source: source.Source}
	if source.Status == utility.AvailabilityObserved {
		vector := canonicalUtilityVector(source.Vector)
		horizontal, observedTick, window := source.HorizontalSpeed, source.ObservedTick, source.MeasurementWindowTicks
		result.Vector = &vector
		result.HorizontalWorldUnitsPerS = &horizontal
		result.ObservedTick = &observedTick
		result.MeasurementWindowTicks = &window
	}
	return result
}

func canonicalUtilityStringObservation(source utility.StringObservation) models.CanonicalUtilityStringObservation {
	result := models.CanonicalUtilityStringObservation{Status: string(source.Status), Source: source.Source}
	if source.Status == utility.AvailabilityObserved {
		value := source.Value
		result.Value = &value
	}
	return result
}

func canonicalUtilityStanceObservation(source utility.StanceObservation) models.CanonicalUtilityStringObservation {
	result := models.CanonicalUtilityStringObservation{Status: string(source.Status), Source: source.Source}
	if source.Status == utility.AvailabilityObserved {
		value := string(source.Value)
		result.Value = &value
	}
	return result
}

func canonicalUtilityMomentObservation(source utility.TickPositionObservation) models.CanonicalUtilityMomentObservation {
	result := models.CanonicalUtilityMomentObservation{
		Status: string(source.Status), PositionStatus: string(source.PositionStatus), Source: source.Source,
	}
	if source.Status == utility.AvailabilityObserved {
		tick := source.Tick
		result.Tick = &tick
	}
	if source.PositionStatus == utility.AvailabilityObserved {
		position := canonicalUtilityVector(source.Position)
		result.Position = &position
	}
	return result
}

func canonicalUtilityDurationObservation(source utility.ScalarObservation) models.CanonicalUtilityDurationObservation {
	result := models.CanonicalUtilityDurationObservation{Status: string(source.Status), Source: source.Source}
	if source.Status == utility.AvailabilityObserved {
		milliseconds := source.Value * 1000
		result.Milliseconds = &milliseconds
	}
	return result
}

func canonicalUtilityCorrelation(source utility.Correlation) models.CanonicalUtilityCorrelation {
	return models.CanonicalUtilityCorrelation{Status: string(source.Status), Source: source.Source}
}

func canonicalUtilityEffectMoment(source utility.Lifecycle) (utility.TickPositionObservation, bool) {
	if source.EffectStart.Status == utility.AvailabilityObserved {
		return source.EffectStart, true
	}
	if source.Detonation.Status == utility.AvailabilityObserved {
		return source.Detonation, true
	}
	return utility.TickPositionObservation{}, false
}

func legacyUtilityType(value utility.Type) string {
	switch value {
	case utility.TypeFlashbang:
		return "Flashbang"
	case utility.TypeSmoke:
		return "Smoke"
	case utility.TypeHE:
		return "HE"
	case utility.TypeMolotov:
		return "Molotov"
	case utility.TypeIncendiary:
		return "Incendiary"
	case utility.TypeDecoy:
		return "Decoy"
	default:
		return "Unknown"
	}
}

func legacyUtilitySide(value string) string {
	side := normalizeSide(value)
	if side == "unknown" {
		return ""
	}
	return strings.ToUpper(side)
}

func buildCanonicalObjectiveEvents(
	ctx *models.DemoContext,
	matchID string,
	tickRate float64,
) ([]models.CanonicalObjectiveEvent, error) {
	if ctx.Objectives == nil {
		return nil, fmt.Errorf("objective tracker is unavailable")
	}
	sources := ctx.Objectives.Events()
	sort.SliceStable(sources, func(i, j int) bool {
		left, right := sources[i], sources[j]
		if left.Round != right.Round {
			return left.Round < right.Round
		}
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		if objectiveEventPriority(left.Type) != objectiveEventPriority(right.Type) {
			return objectiveEventPriority(left.Type) < objectiveEventPriority(right.Type)
		}
		if left.Actor.SteamID != right.Actor.SteamID {
			return left.Actor.SteamID < right.Actor.SteamID
		}
		if objectiveEntityID(left.EntityID) != objectiveEntityID(right.EntityID) {
			return objectiveEntityID(left.EntityID) < objectiveEntityID(right.EntityID)
		}
		return left.Sequence < right.Sequence
	})

	records := make([]models.CanonicalObjectiveEvent, 0, len(sources))
	canonicalAttemptIDs := make(map[string]string)
	attemptOrdinals := make(map[string]int)
	lastRound, lastTick, sequenceInTick := -1, -1, 0
	for index, event := range sources {
		if event.Round != lastRound || event.Tick != lastTick {
			lastRound, lastTick, sequenceInTick = event.Round, event.Tick, 0
		}
		sequenceInTick++
		record := models.CanonicalObjectiveEvent{
			SchemaID:       "stratai.objective_event@2",
			MatchID:        matchID,
			EventID:        canonicalEventID(matchID, "objective", event.Round, event.Tick, index+1),
			RoundID:        canonicalRoundID(matchID, event.Round),
			RoundNumber:    event.Round,
			Tick:           event.Tick,
			SequenceInTick: sequenceInTick,
			EventType:      canonicalObjectiveEventType(event.Type),
			Site:           nullableString(event.Site),
			PositionStatus: event.Position.Status,
			Source:         canonicalObjectiveSource(event.Source),
			StateAfter:     string(event.StateAfter),
			PhaseAfter:     string(event.PhaseAfter),
			HasDefuseKit:   event.HasKit,
			BombEntityID:   event.EntityID,
		}
		if event.Actor.SteamID != 0 {
			playerID := canonicalPlayerID(event.Actor.SteamID)
			record.ActorPlayerID = &playerID
			if side := canonicalObjectiveActorSide(event.Type, event.Actor.Side); side != "unknown" {
				record.ActorSide = stringPointer(side)
			}
		}
		if event.Position.Available() {
			if !finiteObjectivePosition(event.Position) {
				return nil, fmt.Errorf("%s has a non-finite position", event.ID)
			}
			record.Position = &models.CanonicalVector{
				X: event.Position.X,
				Y: event.Position.Y,
				Z: event.Position.Z,
			}
		}
		if event.AttemptID != "" {
			attemptID, exists := canonicalAttemptIDs[event.AttemptID]
			if !exists {
				kind := canonicalObjectiveAttemptKind(event.Type)
				ordinalKey := fmt.Sprintf("%03d:%s", event.Round, kind)
				attemptOrdinals[ordinalKey]++
				attemptID = fmt.Sprintf(
					"%s:objective-attempt:%03d:%s:%03d",
					matchID,
					event.Round,
					kind,
					attemptOrdinals[ordinalKey],
				)
				canonicalAttemptIDs[event.AttemptID] = attemptID
			}
			record.AttemptID = &attemptID
			outcome := string(event.AttemptOutcome)
			record.AttemptOutcome = &outcome
			startObserved := event.AttemptStartObserved
			record.AttemptStartObserved = &startObserved
		}
		if event.DurationTicks != nil {
			durationMS := ticksToMilliseconds(*event.DurationTicks, tickRate)
			record.ActionDurationMS = &durationMS
		}
		records = append(records, record)
	}
	return records, nil
}

func canonicalObjectiveActorSide(eventType objective.EventType, observedSide string) string {
	if side := normalizeSide(observedSide); side != "unknown" {
		return side
	}
	switch canonicalObjectiveEventType(eventType) {
	case "bomb_carrier_snapshot", "bomb_drop", "bomb_pickup", "bomb_plant_start", "bomb_plant_abort", "bomb_plant":
		return "t"
	case "bomb_defuse_start", "bomb_defuse_abort", "bomb_defuse":
		return "ct"
	default:
		return "unknown"
	}
}

func canonicalObjectiveAttemptKind(eventType objective.EventType) string {
	switch eventType {
	case objective.EventPlantStart, objective.EventPlantAbort, objective.EventPlant:
		return "plant"
	case objective.EventDefuseStart, objective.EventDefuseAbort, objective.EventDefuse:
		return "defuse"
	default:
		return "unknown"
	}
}

func canonicalObjectiveAttemptKindFromEventType(eventType string) string {
	switch eventType {
	case "bomb_plant_start", "bomb_plant_abort", "bomb_plant":
		return "plant"
	case "bomb_defuse_start", "bomb_defuse_abort", "bomb_defuse":
		return "defuse"
	default:
		return ""
	}
}

func objectiveEventPriority(eventType objective.EventType) int {
	switch eventType {
	case objective.EventCarrierSnapshot:
		return 0
	case objective.EventDrop:
		return 1
	case objective.EventPickup:
		return 2
	case objective.EventPlantStart:
		return 3
	case objective.EventPlantAbort:
		return 4
	case objective.EventPlant:
		return 5
	case objective.EventDefuseStart:
		return 6
	case objective.EventDefuseAbort:
		return 7
	case objective.EventDefuse:
		return 8
	case objective.EventExplode:
		return 9
	default:
		return 100
	}
}

func canonicalObjectiveEventType(eventType objective.EventType) string {
	if eventType == objective.EventCarrierSnapshot {
		return "bomb_carrier_snapshot"
	}
	return "bomb_" + string(eventType)
}

func canonicalObjectiveSource(source string) string {
	switch source {
	case objective.SourceDemoinfocsNativeSnapshot:
		return "game_state_snapshot"
	case objective.SourceDemoinfocsEvent:
		return "demoinfocs_event"
	default:
		return source
	}
}

func objectiveEntityID(entityID *int) int {
	if entityID == nil {
		return -1
	}
	return *entityID
}

func finiteObjectivePosition(position objective.Position) bool {
	return !math.IsNaN(position.X) && !math.IsInf(position.X, 0) &&
		!math.IsNaN(position.Y) && !math.IsInf(position.Y, 0) &&
		!math.IsNaN(position.Z) && !math.IsInf(position.Z, 0)
}

func ticksToMilliseconds(ticks int, tickRate float64) int64 {
	if ticks <= 0 || tickRate <= 0 || math.IsNaN(tickRate) || math.IsInf(tickRate, 0) {
		return 0
	}
	return int64(math.Round(float64(ticks) * 1000 / tickRate))
}

func secondsToMillisecondsPointer(seconds *float64) *int64 {
	if seconds == nil {
		return nil
	}
	milliseconds := int64(math.Round(*seconds * 1000))
	return &milliseconds
}

func canonicalOptionalInt(value *int) string {
	if value == nil {
		return "null"
	}
	return strconv.Itoa(*value)
}

var canonicalObjectiveEventTypes = map[string]struct{}{
	"bomb_carrier_snapshot": {},
	"bomb_drop":             {},
	"bomb_pickup":           {},
	"bomb_plant_start":      {},
	"bomb_plant_abort":      {},
	"bomb_plant":            {},
	"bomb_defuse_start":     {},
	"bomb_defuse_abort":     {},
	"bomb_defuse":           {},
	"bomb_explode":          {},
}

func validateCanonicalObjectives(
	rounds models.CanonicalRoundsExport,
	events []models.CanonicalObjectiveEvent,
) error {
	roundByNumber := make(map[int]models.CanonicalRound, len(rounds.Rounds))
	for _, round := range rounds.Rounds {
		roundByNumber[round.RoundNumber] = round
	}
	stateByRound := make(map[int]string, len(rounds.Rounds))
	type attemptValidation struct {
		kind                  string
		startCount            int
		terminalCount         int
		terminalStartObserved *bool
		startActor            *string
		startSite             *string
		startKit              *bool
		terminalActor         *string
		terminalSite          *string
		terminalKit           *bool
	}
	attempts := make(map[string]attemptValidation)
	lastRound, lastTick, lastSequence := -1, -1, 0
	for _, event := range events {
		round, exists := roundByNumber[event.RoundNumber]
		if !exists || event.RoundID != round.RoundID {
			return fmt.Errorf("%s references an unknown round", event.EventID)
		}
		if _, exists := canonicalObjectiveEventTypes[event.EventType]; !exists {
			return fmt.Errorf("%s has unknown type %q", event.EventID, event.EventType)
		}
		if round.StartTick != nil && event.Tick < *round.StartTick ||
			round.EndTick != nil && event.Tick > *round.EndTick {
			return fmt.Errorf(
				"%s at tick %d is outside round bounds start=%s end=%s",
				event.EventID,
				event.Tick,
				canonicalOptionalInt(round.StartTick),
				canonicalOptionalInt(round.EndTick),
			)
		}
		if event.RoundNumber != lastRound || event.Tick != lastTick {
			lastRound, lastTick, lastSequence = event.RoundNumber, event.Tick, 0
		}
		lastSequence++
		if event.SequenceInTick != lastSequence {
			return fmt.Errorf("%s has a non-contiguous sequence_in_tick", event.EventID)
		}
		if err := validateCanonicalObjectiveObservation(event); err != nil {
			return fmt.Errorf("%s: %w", event.EventID, err)
		}
		previousState := stateByRound[event.RoundNumber]
		if err := validateCanonicalObjectiveTransition(previousState, event); err != nil {
			return fmt.Errorf("%s: %w", event.EventID, err)
		}
		stateByRound[event.RoundNumber] = event.StateAfter
		if event.AttemptID != nil {
			attempt := attempts[*event.AttemptID]
			kind := canonicalObjectiveAttemptKindFromEventType(event.EventType)
			if attempt.kind != "" && attempt.kind != kind {
				return fmt.Errorf("attempt %s mixes %s and %s events", *event.AttemptID, attempt.kind, kind)
			}
			attempt.kind = kind
			if strings.HasSuffix(event.EventType, "_start") {
				attempt.startCount++
				attempt.startActor = event.ActorPlayerID
				attempt.startSite = event.Site
				attempt.startKit = event.HasDefuseKit
			} else {
				attempt.terminalCount++
				attempt.terminalStartObserved = event.AttemptStartObserved
				attempt.terminalActor = event.ActorPlayerID
				attempt.terminalSite = event.Site
				attempt.terminalKit = event.HasDefuseKit
			}
			attempts[*event.AttemptID] = attempt
		}
	}
	for attemptID, attempt := range attempts {
		if attempt.startCount > 1 || attempt.terminalCount > 1 {
			return fmt.Errorf(
				"attempt %s has %d starts and %d terminal events",
				attemptID,
				attempt.startCount,
				attempt.terminalCount,
			)
		}
		if attempt.terminalCount == 1 && attempt.terminalStartObserved != nil {
			if *attempt.terminalStartObserved && attempt.startCount != 1 {
				return fmt.Errorf("attempt %s claims an observed start but has none", attemptID)
			}
			if !*attempt.terminalStartObserved && attempt.startCount != 0 {
				return fmt.Errorf("attempt %s contains a start marked unobserved", attemptID)
			}
			if *attempt.terminalStartObserved &&
				(!reflect.DeepEqual(attempt.startActor, attempt.terminalActor) ||
					!reflect.DeepEqual(attempt.startSite, attempt.terminalSite) ||
					!reflect.DeepEqual(attempt.startKit, attempt.terminalKit)) {
				return fmt.Errorf("attempt %s changes actor, site or kit state", attemptID)
			}
		}
	}
	for _, round := range rounds.Rounds {
		if err := validateCanonicalRoundObjective(round, events); err != nil {
			return fmt.Errorf("round %d: %w", round.RoundNumber, err)
		}
	}
	return nil
}

func validateCanonicalObjectiveObservation(event models.CanonicalObjectiveEvent) error {
	if event.SchemaID != "stratai.objective_event@2" {
		return fmt.Errorf("schema is %q", event.SchemaID)
	}
	if event.Source != "demoinfocs_event" && event.Source != "game_state_snapshot" {
		return fmt.Errorf("source is %q", event.Source)
	}
	switch event.PositionStatus {
	case "observed":
		if event.Position == nil || math.IsNaN(event.Position.X) || math.IsInf(event.Position.X, 0) ||
			math.IsNaN(event.Position.Y) || math.IsInf(event.Position.Y, 0) ||
			math.IsNaN(event.Position.Z) || math.IsInf(event.Position.Z, 0) {
			return fmt.Errorf("observed position is missing or non-finite")
		}
	case "unavailable":
		if event.Position != nil {
			return fmt.Errorf("unavailable position contains coordinates")
		}
	default:
		return fmt.Errorf("position_status is %q", event.PositionStatus)
	}
	if event.ActionDurationMS != nil && *event.ActionDurationMS < 0 {
		return fmt.Errorf("action_duration_ms is negative")
	}
	if (event.ActorPlayerID == nil) != (event.ActorSide == nil) {
		return fmt.Errorf("actor_player_id and actor_side must be available together")
	}
	attemptKind := canonicalObjectiveAttemptKindFromEventType(event.EventType)
	if attemptKind == "" {
		if event.AttemptID != nil || event.AttemptOutcome != nil || event.AttemptStartObserved != nil || event.ActionDurationMS != nil {
			return fmt.Errorf("non-attempt event contains attempt metadata")
		}
	} else {
		if event.AttemptID == nil || strings.TrimSpace(*event.AttemptID) == "" ||
			event.AttemptOutcome == nil || event.AttemptStartObserved == nil {
			return fmt.Errorf("attempt event is missing identity or observation metadata")
		}
		isStart := strings.HasSuffix(event.EventType, "_start")
		if isStart {
			if *event.AttemptOutcome != "in_progress" || !*event.AttemptStartObserved || event.ActionDurationMS != nil {
				return fmt.Errorf("attempt start contains invalid outcome or duration")
			}
		} else {
			expectedOutcome := "completed"
			if strings.HasSuffix(event.EventType, "_abort") {
				expectedOutcome = "aborted"
			}
			if *event.AttemptOutcome != expectedOutcome {
				return fmt.Errorf("attempt terminal outcome is %q, want %q", *event.AttemptOutcome, expectedOutcome)
			}
			if *event.AttemptStartObserved && event.ActionDurationMS == nil {
				return fmt.Errorf("terminal with an observed start has no duration")
			}
			if !*event.AttemptStartObserved && event.ActionDurationMS != nil {
				return fmt.Errorf("terminal without an observed start contains a duration")
			}
		}
		if attemptKind == "defuse" {
			if *event.AttemptStartObserved && event.HasDefuseKit == nil {
				return fmt.Errorf("defuse attempt with an observed start has unknown kit state")
			}
			if !*event.AttemptStartObserved && event.HasDefuseKit != nil {
				return fmt.Errorf("defuse attempt without an observed start contains kit state")
			}
		}
	}

	if event.Site == nil {
		switch event.EventType {
		case "bomb_plant", "bomb_defuse", "bomb_explode":
			return fmt.Errorf("terminal objective event has no bomb site")
		}
	}
	if event.ActorSide != nil {
		expectedSide := ""
		switch event.EventType {
		case "bomb_carrier_snapshot", "bomb_drop", "bomb_pickup", "bomb_plant_start", "bomb_plant_abort", "bomb_plant":
			expectedSide = "t"
		case "bomb_defuse_start", "bomb_defuse_abort", "bomb_defuse":
			expectedSide = "ct"
		}
		if expectedSide != "" && *event.ActorSide != expectedSide {
			return fmt.Errorf("actor side is %q, want %q", *event.ActorSide, expectedSide)
		}
	}
	return nil
}

func validateCanonicalObjectiveTransition(
	previousState string,
	event models.CanonicalObjectiveEvent,
) error {
	allowedPrevious := map[string]map[string]struct{}{
		"bomb_carrier_snapshot": {"": {}, "carried": {}, "dropped": {}},
		"bomb_drop":             {"": {}, "carried": {}, "planting": {}},
		"bomb_pickup":           {"": {}, "carried": {}, "dropped": {}},
		"bomb_plant_start":      {"": {}, "carried": {}},
		"bomb_plant_abort":      {"planting": {}, "dropped": {}},
		"bomb_plant":            {"planting": {}},
		"bomb_defuse_start":     {"planted": {}},
		"bomb_defuse_abort":     {"defusing": {}},
		"bomb_defuse":           {"defusing": {}},
		"bomb_explode":          {"planted": {}, "defusing": {}},
	}
	expectedState := map[string]string{
		"bomb_carrier_snapshot": "carried",
		"bomb_drop":             "dropped",
		"bomb_pickup":           "carried",
		"bomb_plant_start":      "planting",
		"bomb_plant_abort":      "carried",
		"bomb_plant":            "planted",
		"bomb_defuse_start":     "defusing",
		"bomb_defuse_abort":     "planted",
		"bomb_defuse":           "defused",
		"bomb_explode":          "exploded",
	}
	if event.EventType == "bomb_plant_abort" && previousState == "dropped" {
		expectedState[event.EventType] = "dropped"
	}
	expectedPhase := map[string]string{
		"bomb_carrier_snapshot": "preplant",
		"bomb_drop":             "preplant",
		"bomb_pickup":           "preplant",
		"bomb_plant_start":      "planting",
		"bomb_plant_abort":      "preplant",
		"bomb_plant":            "planted",
		"bomb_defuse_start":     "defusing",
		"bomb_defuse_abort":     "planted",
		"bomb_defuse":           "resolved",
		"bomb_explode":          "resolved",
	}
	if _, allowed := allowedPrevious[event.EventType][previousState]; !allowed {
		observedStartMissing := event.AttemptStartObserved != nil && !*event.AttemptStartObserved
		if !observedStartMissing || !canonicalObjectiveTransitionWithoutStart(previousState, event.EventType) {
			return fmt.Errorf("invalid transition from %q via %q", previousState, event.EventType)
		}
	}
	if event.StateAfter != expectedState[event.EventType] {
		return fmt.Errorf("state_after %q does not match %q", event.StateAfter, event.EventType)
	}
	if event.PhaseAfter != expectedPhase[event.EventType] {
		return fmt.Errorf("phase_after %q does not match %q", event.PhaseAfter, event.EventType)
	}
	return nil
}

func canonicalObjectiveTransitionWithoutStart(previousState, eventType string) bool {
	switch eventType {
	case "bomb_plant_abort", "bomb_plant":
		return previousState == "" || previousState == "carried" || previousState == "dropped"
	case "bomb_defuse_abort", "bomb_defuse":
		return previousState == "planted"
	default:
		return false
	}
}

func validateCanonicalRoundObjective(
	round models.CanonicalRound,
	events []models.CanonicalObjectiveEvent,
) error {
	expected := summarizeCanonicalRoundObjective(round.RoundNumber, round.RawWinReasonCode, events)
	var plant, terminal *models.CanonicalObjectiveEvent
	plantCount, terminalCount := 0, 0
	for index := range events {
		event := &events[index]
		if event.RoundNumber != round.RoundNumber {
			continue
		}
		switch event.EventType {
		case "bomb_plant":
			plant, plantCount = event, plantCount+1
		case "bomb_defuse", "bomb_explode":
			terminal, terminalCount = event, terminalCount+1
		}
	}
	if plantCount > 1 || terminalCount > 1 {
		return fmt.Errorf("contains %d plants and %d terminal events", plantCount, terminalCount)
	}
	if terminal != nil {
		if plant == nil || terminal.Tick < plant.Tick {
			return fmt.Errorf("terminal event is not preceded by a plant")
		}
		if plant.Site == nil || terminal.Site == nil || *plant.Site != *terminal.Site {
			return fmt.Errorf("plant and terminal sites differ")
		}
	}
	if !reflect.DeepEqual(round.Objective, expected) {
		return fmt.Errorf("objective summary differs from lifecycle events")
	}
	if round.BombPlanted == nil || *round.BombPlanted != expected.WasBombPlanted ||
		!reflect.DeepEqual(round.BombSite, expected.Site) ||
		!reflect.DeepEqual(round.BombTick, expected.PlantTick) {
		return fmt.Errorf("legacy bomb fields differ from objective summary")
	}
	if round.RawWinReasonCode == nil || round.WinReason == nil {
		return fmt.Errorf("win reason is unavailable")
	}
	expectedReason := canonicalRoundEndReasons[*round.RawWinReasonCode]
	if expectedReason == "" {
		expectedReason = "unknown"
	}
	if *round.WinReason != expectedReason {
		return fmt.Errorf("win reason label does not match raw code")
	}
	switch *round.RawWinReasonCode {
	case 1:
		if expected.Outcome != "exploded" {
			return fmt.Errorf("target_bombed round has outcome %q", expected.Outcome)
		}
	case 7:
		if expected.Outcome != "defused" {
			return fmt.Errorf("bomb_defused round has outcome %q", expected.Outcome)
		}
	case 8:
		if expected.WasBombPlanted && expected.ResolutionEventID == nil {
			return fmt.Errorf("CT win after plant has no defuse event")
		}
	case 9:
		if expected.Outcome == "defused" {
			return fmt.Errorf("terrorist win cannot end in defuse")
		}
	case 12:
		if expected.WasBombPlanted || expected.Outcome != "time_expired" {
			return fmt.Errorf("target_saved round has objective outcome %q", expected.Outcome)
		}
	}
	return nil
}

func buildCanonicalPlayerStates(ctx *models.DemoContext, matchID string, rosters canonicalRosterInfo) map[int][]models.CanonicalPlayerState {
	states := append([]models.AI_TrackingEventWithRound(nil), ctx.AI_TrackingEventsWithRound...)
	sort.SliceStable(states, func(i, j int) bool {
		if states[i].Round != states[j].Round {
			return states[i].Round < states[j].Round
		}
		if states[i].Event.Tick != states[j].Event.Tick {
			return states[i].Event.Tick < states[j].Event.Tick
		}
		return states[i].Event.PlayerSteamID < states[j].Event.PlayerSteamID
	})
	byRound := make(map[int][]models.CanonicalPlayerState, ctx.CurrentRound)
	for _, state := range states {
		event := state.Event
		if event.PlayerSteamID == 0 {
			continue
		}
		var horizontalVelocity *float64
		var velocityVector *models.CanonicalVector
		var velocityMeasurementTicks *int
		if event.VelocityAvailable {
			horizontal := math.Hypot(event.VelocityVector.X, event.VelocityVector.Y)
			horizontalVelocity = &horizontal
			vector := canonicalVector(event.VelocityVector)
			velocityVector = &vector
			measurementTicks := event.VelocityMeasurementTicks
			velocityMeasurementTicks = &measurementTicks
		}
		playerID := canonicalPlayerID(event.PlayerSteamID)
		teamID := "unknown"
		if assigned := rosters.playerTeams[event.PlayerSteamID]; assigned != "" {
			teamID = assigned
		}
		byRound[state.Round] = append(byRound[state.Round], models.CanonicalPlayerState{
			SchemaID: "stratai.player_state@3", MatchID: matchID,
			StateID: fmt.Sprintf("%s:state:%03d:%09d:%s", matchID, state.Round, event.Tick, playerID),
			RoundID: canonicalRoundID(matchID, state.Round), RoundNumber: state.Round, Tick: event.Tick,
			PlayerID: playerID, TeamID: teamID, Side: normalizeSide(event.Team), Position: canonicalVector(event.Position),
			Area: event.AreaName, ViewYawDeg: event.ViewAngleYaw, ViewPitchDeg: event.ViewAnglePitch,
			HorizontalVelocityWorldUPS: horizontalVelocity, VelocityVectorWorldUPS: velocityVector,
			VelocitySource: string(event.VelocitySource), VelocityMeasurementTicks: velocityMeasurementTicks,
			Health: event.Health, Armor: event.Armor, IsAlive: event.IsAlive,
			ActiveWeapon: event.ActiveWeapon, ActiveWeaponStatus: event.ActiveWeaponStatus,
			LastObservedActiveWeapon:     event.LastObservedActiveWeapon,
			LastObservedActiveWeaponTick: event.LastObservedActiveWeaponTick, HasC4: event.HasC4,
			HasDefuseKit: event.HasDefuseKit, IsPlanting: event.IsPlanting, IsDefusing: event.IsDefusing,
			IsWalking: event.IsWalking, IsDucking: event.IsDucking, NearbyTeammates: event.NearbyTeammates,
			RoundTimeMS: int64(math.Round(event.RoundTimeRemaining * 1000)), ObjectivePhase: event.ObjectivePhase,
			PhaseTimeRemainingMS:  secondsToMillisecondsPointer(event.PhaseTimeRemaining),
			RoundClockRemainingMS: secondsToMillisecondsPointer(event.RoundClockRemaining),
			BombTimeRemainingMS:   secondsToMillisecondsPointer(event.BombTimeRemaining),
		})
	}
	return byRound
}

func filterCanonicalPlayerStatesToRounds(
	statesByRound map[int][]models.CanonicalPlayerState,
	rounds []models.CanonicalRound,
) map[int][]models.CanonicalPlayerState {
	filtered := make(map[int][]models.CanonicalPlayerState, len(statesByRound))
	for roundNumber, states := range statesByRound {
		for _, state := range states {
			if canonicalTickWithinRound(rounds, roundNumber, state.Tick) {
				filtered[roundNumber] = append(filtered[roundNumber], state)
			}
		}
	}
	return filtered
}

func validateCanonicalPlayerStates(statesByRound map[int][]models.CanonicalPlayerState) error {
	for roundNumber, states := range statesByRound {
		for _, state := range states {
			if state.SchemaID != "stratai.player_state@3" {
				return fmt.Errorf("%s uses schema %q", state.StateID, state.SchemaID)
			}
			if state.RoundNumber != roundNumber || state.RoundNumber <= 0 || state.Tick < 0 {
				return fmt.Errorf("%s has an invalid round or tick", state.StateID)
			}
			if !canonicalVectorIsFinite(state.Position) {
				return fmt.Errorf("%s has a non-finite position", state.StateID)
			}
			if err := validateCanonicalMotionState(state); err != nil {
				return fmt.Errorf("%s: %w", state.StateID, err)
			}
			if err := validateCanonicalWeaponState(state); err != nil {
				return fmt.Errorf("%s: %w", state.StateID, err)
			}
			if err := validateCanonicalObjectiveState(state); err != nil {
				return fmt.Errorf("%s: %w", state.StateID, err)
			}
		}
	}
	return nil
}

func validateCanonicalMotionState(state models.CanonicalPlayerState) error {
	source := playerstate.VelocitySource(state.VelocitySource)
	available := source == playerstate.VelocitySourceNative || source == playerstate.VelocitySourcePositionDelta
	if !state.IsAlive {
		if source != playerstate.VelocitySourceNotApplicable {
			return fmt.Errorf("dead player velocity source is %q", source)
		}
		available = false
	} else if source == playerstate.VelocitySourceNotApplicable {
		return fmt.Errorf("alive player velocity cannot be not_applicable")
	}

	if !available {
		switch source {
		case playerstate.VelocitySourceInsufficientHistory,
			playerstate.VelocitySourceNotApplicable,
			playerstate.VelocitySourceRejected,
			playerstate.VelocitySourceStaleGap,
			playerstate.VelocitySourceEntityChanged,
			playerstate.VelocitySourceNonMonotonicTick:
		default:
			return fmt.Errorf("unknown velocity source %q", source)
		}
		if state.HorizontalVelocityWorldUPS != nil || state.VelocityVectorWorldUPS != nil || state.VelocityMeasurementTicks != nil {
			return fmt.Errorf("unavailable velocity contains measured values")
		}
		return nil
	}

	if state.HorizontalVelocityWorldUPS == nil || state.VelocityVectorWorldUPS == nil || state.VelocityMeasurementTicks == nil {
		return fmt.Errorf("available velocity is missing measured values")
	}
	vector := *state.VelocityVectorWorldUPS
	horizontal := *state.HorizontalVelocityWorldUPS
	if !canonicalVectorIsFinite(vector) || math.IsNaN(horizontal) || math.IsInf(horizontal, 0) || horizontal < 0 {
		return fmt.Errorf("velocity contains a non-finite or negative value")
	}
	expectedHorizontal := math.Hypot(vector.X, vector.Y)
	if math.Abs(horizontal-expectedHorizontal) > 1e-6 {
		return fmt.Errorf("horizontal velocity does not match its vector")
	}
	if horizontal > playerstate.MaxPlausibleHorizontalSpeedUPS || math.Abs(vector.Z) > playerstate.MaxPlausibleVerticalSpeedUPS {
		return fmt.Errorf("velocity exceeds plausible bounds")
	}
	if source == playerstate.VelocitySourcePositionDelta {
		if *state.VelocityMeasurementTicks < 1 || *state.VelocityMeasurementTicks > playerstate.MaxPositionDeltaIntervalTicks {
			return fmt.Errorf("position delta has invalid measurement window")
		}
	} else if *state.VelocityMeasurementTicks != 0 {
		return fmt.Errorf("native velocity must use a zero-tick measurement window")
	}
	return nil
}

func validateCanonicalWeaponState(state models.CanonicalPlayerState) error {
	if (state.LastObservedActiveWeapon == nil) != (state.LastObservedActiveWeaponTick == nil) {
		return fmt.Errorf("last observed weapon and tick must be present together")
	}
	if state.LastObservedActiveWeapon != nil {
		if strings.TrimSpace(*state.LastObservedActiveWeapon) == "" || *state.LastObservedActiveWeaponTick > state.Tick {
			return fmt.Errorf("last observed weapon is invalid or comes from the future")
		}
	}

	if !state.IsAlive {
		if state.ActiveWeaponStatus != models.ActiveWeaponStatusNotApplicable || state.ActiveWeapon != nil {
			return fmt.Errorf("dead player has a current active weapon")
		}
		return nil
	}

	switch state.ActiveWeaponStatus {
	case models.ActiveWeaponStatusObserved:
		if state.ActiveWeapon == nil || strings.TrimSpace(*state.ActiveWeapon) == "" {
			return fmt.Errorf("observed active weapon is empty")
		}
	case models.ActiveWeaponStatusUnavailable:
		if state.ActiveWeapon != nil {
			return fmt.Errorf("unavailable active weapon contains a value")
		}
	default:
		return fmt.Errorf("alive player has invalid active weapon status %q", state.ActiveWeaponStatus)
	}
	return nil
}

func validateCanonicalObjectiveState(state models.CanonicalPlayerState) error {
	if state.IsPlanting && state.IsDefusing {
		return fmt.Errorf("player cannot plant and defuse at the same time")
	}
	if !state.IsAlive && (state.HasC4 || state.IsPlanting || state.IsDefusing) {
		return fmt.Errorf("dead player has an active objective role")
	}
	if state.IsPlanting && (!state.HasC4 || state.Side != "t" || state.ObjectivePhase != "planting") {
		return fmt.Errorf(
			"planting state conflicts with player role or objective phase (has_c4=%t side=%q phase=%q)",
			state.HasC4,
			state.Side,
			state.ObjectivePhase,
		)
	}
	if state.IsDefusing && (state.Side != "ct" || state.ObjectivePhase != "defusing") {
		return fmt.Errorf("defusing state conflicts with player role or objective phase")
	}
	if state.PhaseTimeRemainingMS == nil || *state.PhaseTimeRemainingMS < 0 {
		return fmt.Errorf("phase clock is unavailable or negative")
	}
	if state.RoundTimeMS != *state.PhaseTimeRemainingMS {
		return fmt.Errorf("legacy round clock differs from the phase clock")
	}

	sameClock := func(clock *int64) bool {
		return clock != nil && *clock >= 0 && *clock == *state.PhaseTimeRemainingMS
	}
	switch state.ObjectivePhase {
	case "preplant", "planting":
		if !sameClock(state.RoundClockRemainingMS) || state.BombTimeRemainingMS != nil {
			return fmt.Errorf("preplant phase contains invalid clock availability")
		}
	case "planted", "defusing":
		if state.RoundClockRemainingMS != nil || !sameClock(state.BombTimeRemainingMS) {
			return fmt.Errorf("postplant phase contains invalid clock availability")
		}
	case "resolved":
		if *state.PhaseTimeRemainingMS != 0 || state.RoundClockRemainingMS != nil || state.BombTimeRemainingMS != nil {
			return fmt.Errorf("resolved phase contains an active clock")
		}
	default:
		return fmt.Errorf("unknown objective phase %q", state.ObjectivePhase)
	}
	return nil
}

func canonicalVectorIsFinite(vector models.CanonicalVector) bool {
	return !math.IsNaN(vector.X) && !math.IsInf(vector.X, 0) &&
		!math.IsNaN(vector.Y) && !math.IsInf(vector.Y, 0) &&
		!math.IsNaN(vector.Z) && !math.IsInf(vector.Z, 0)
}

type canonicalDuelCandidate struct {
	detail        models.AI_Duel
	attackerID    string
	victimIDs     []string
	victimIDKey   string
	detailSortKey string
}

func buildCanonicalEngagements(
	ctx *models.DemoContext,
	matchID string,
	combatEvents []models.CanonicalCombatEvent,
) (models.CanonicalEngagementsExport, error) {
	duels := make([]canonicalDuelCandidate, 0, len(ctx.AI_Duels))
	for _, duel := range ctx.AI_Duels {
		candidate, err := buildCanonicalDuelCandidate(duel)
		if err != nil {
			return models.CanonicalEngagementsExport{}, err
		}
		duels = append(duels, candidate)
	}
	sort.Slice(duels, func(i, j int) bool { return canonicalDuelLess(duels[i], duels[j]) })

	engagements := make([]models.CanonicalEngagement, 0, len(duels))
	for index, candidate := range duels {
		duel := candidate.detail
		victimSet := make(map[string]struct{}, len(candidate.victimIDs))
		for _, playerID := range candidate.victimIDs {
			victimSet[playerID] = struct{}{}
		}
		sourceIDs := make([]string, 0, len(duel.Exchanges))
		for _, event := range combatEvents {
			if event.EventType != string(combat.EventPlayerHurt) && event.EventType != string(combat.EventKill) {
				continue
			}
			if event.RoundNumber != duel.Round || event.Tick < duel.TickStart || event.Tick > duel.TickEnd ||
				event.ActorPlayerID == nil || *event.ActorPlayerID != candidate.attackerID || event.TargetPlayerID == nil {
				continue
			}
			if _, ok := victimSet[*event.TargetPlayerID]; ok {
				sourceIDs = append(sourceIDs, event.EventID)
			}
		}
		sort.Strings(sourceIDs)
		engagementID := fmt.Sprintf("%s:engagement:%06d", matchID, index+1)
		duel.DuelID = engagementID
		engagements = append(engagements, models.CanonicalEngagement{
			EngagementID: engagementID,
			RoundID:      canonicalRoundID(matchID, duel.Round), RoundNumber: duel.Round,
			StartTick: duel.TickStart, EndTick: duel.TickEnd, EngagementType: strings.ToLower(duel.Type),
			Outcome: strings.ToLower(duel.Outcome), AttackerPlayerID: candidate.attackerID,
			VictimPlayerIDs: candidate.victimIDs, SourceEventIDs: sourceIDs, AlgorithmVersion: "duel_consolidation@1",
			Details: duel,
		})
	}
	return models.CanonicalEngagementsExport{SchemaID: "stratai.engagements@1", MatchID: matchID, Engagements: engagements}, nil
}

func buildCanonicalDuelCandidate(source models.AI_Duel) (canonicalDuelCandidate, error) {
	detail := source
	detail.DuelID = ""

	var err error
	detail.Victims, err = sortCanonicalDuelVictims(source.Victims)
	if err != nil {
		return canonicalDuelCandidate{}, fmt.Errorf("sort duel victims: %w", err)
	}
	detail.Exchanges, err = sortCanonicalDuelExchanges(source.Exchanges)
	if err != nil {
		return canonicalDuelCandidate{}, fmt.Errorf("sort duel exchanges: %w", err)
	}

	victimIDs := make([]string, 0, len(detail.Victims))
	for _, victim := range detail.Victims {
		victimIDs = append(victimIDs, canonicalPlayerID(victim.SteamID))
	}
	sort.Strings(victimIDs)

	sortPayload := struct {
		Round  int            `json:"round"`
		Detail models.AI_Duel `json:"detail"`
	}{Round: detail.Round, Detail: detail}
	sortKey, err := json.Marshal(sortPayload)
	if err != nil {
		return canonicalDuelCandidate{}, fmt.Errorf("encode duel sort key: %w", err)
	}
	return canonicalDuelCandidate{
		detail:        detail,
		attackerID:    canonicalPlayerID(detail.Attacker.SteamID),
		victimIDs:     victimIDs,
		victimIDKey:   strings.Join(victimIDs, "\x00"),
		detailSortKey: string(sortKey),
	}, nil
}

func sortCanonicalDuelVictims(source []models.AI_DuelParticipant) ([]models.AI_DuelParticipant, error) {
	type keyedVictim struct {
		participant models.AI_DuelParticipant
		playerID    string
		detailKey   string
	}
	victims := make([]keyedVictim, 0, len(source))
	for _, participant := range source {
		encoded, err := json.Marshal(participant)
		if err != nil {
			return nil, fmt.Errorf("encode victim %d sort key: %w", participant.SteamID, err)
		}
		victims = append(victims, keyedVictim{
			participant: participant,
			playerID:    canonicalPlayerID(participant.SteamID),
			detailKey:   string(encoded),
		})
	}
	sort.Slice(victims, func(i, j int) bool {
		if victims[i].playerID != victims[j].playerID {
			return victims[i].playerID < victims[j].playerID
		}
		return victims[i].detailKey < victims[j].detailKey
	})

	result := make([]models.AI_DuelParticipant, len(victims))
	for index, victim := range victims {
		result[index] = victim.participant
	}
	return result, nil
}

func sortCanonicalDuelExchanges(source []models.AI_DuelExchange) ([]models.AI_DuelExchange, error) {
	type keyedExchange struct {
		exchange  models.AI_DuelExchange
		detailKey string
	}
	exchanges := make([]keyedExchange, 0, len(source))
	for _, exchange := range source {
		encoded, err := json.Marshal(exchange)
		if err != nil {
			return nil, fmt.Errorf("encode exchange at tick %d sort key: %w", exchange.Tick, err)
		}
		exchanges = append(exchanges, keyedExchange{exchange: exchange, detailKey: string(encoded)})
	}
	sort.Slice(exchanges, func(i, j int) bool {
		if exchanges[i].exchange.Tick != exchanges[j].exchange.Tick {
			return exchanges[i].exchange.Tick < exchanges[j].exchange.Tick
		}
		return exchanges[i].detailKey < exchanges[j].detailKey
	})

	result := make([]models.AI_DuelExchange, len(exchanges))
	for index, exchange := range exchanges {
		result[index] = exchange.exchange
	}
	return result, nil
}

func canonicalDuelLess(left, right canonicalDuelCandidate) bool {
	if left.detail.Round != right.detail.Round {
		return left.detail.Round < right.detail.Round
	}
	if left.detail.TickStart != right.detail.TickStart {
		return left.detail.TickStart < right.detail.TickStart
	}
	if left.detail.TickEnd != right.detail.TickEnd {
		return left.detail.TickEnd < right.detail.TickEnd
	}
	leftType, rightType := strings.ToLower(left.detail.Type), strings.ToLower(right.detail.Type)
	if leftType != rightType {
		return leftType < rightType
	}
	leftOutcome, rightOutcome := strings.ToLower(left.detail.Outcome), strings.ToLower(right.detail.Outcome)
	if leftOutcome != rightOutcome {
		return leftOutcome < rightOutcome
	}
	if left.detail.Attacker.SteamID != right.detail.Attacker.SteamID {
		return left.detail.Attacker.SteamID < right.detail.Attacker.SteamID
	}
	if left.victimIDKey != right.victimIDKey {
		return left.victimIDKey < right.victimIDKey
	}
	return left.detailSortKey < right.detailSortKey
}

func validateCanonicalEngagementObservations(export models.CanonicalEngagementsExport) error {
	for _, engagement := range export.Engagements {
		if err := validateCanonicalDuelParticipant(engagement.Details.Attacker, engagement.EndTick); err != nil {
			return fmt.Errorf("%s attacker: %w", engagement.EngagementID, err)
		}
		for index, victim := range engagement.Details.Victims {
			if err := validateCanonicalDuelParticipant(victim, engagement.EndTick); err != nil {
				return fmt.Errorf("%s victim[%d]: %w", engagement.EngagementID, index, err)
			}
		}
	}
	return nil
}

func validateCanonicalDuelParticipant(participant models.AI_DuelParticipant, engagementEndTick int) error {
	if err := validateCanonicalDuelParticipantMotion(participant, engagementEndTick); err != nil {
		return fmt.Errorf("movement observation: %w", err)
	}
	if err := validateCanonicalDuelParticipantWeapon(participant, engagementEndTick); err != nil {
		return fmt.Errorf("active weapon observation: %w", err)
	}
	return nil
}

func validateCanonicalDuelParticipantMotion(participant models.AI_DuelParticipant, engagementEndTick int) error {
	if !participant.VelocityAvailable {
		return validateCanonicalUnavailableDuelMotion(participant)
	}
	return validateCanonicalAvailableDuelMotion(participant, engagementEndTick)
}

func validateCanonicalAvailableDuelMotion(participant models.AI_DuelParticipant, engagementEndTick int) error {
	if participant.Velocity == nil || participant.VelocityMeasurementTicks == nil || participant.VelocityObservedTick == nil {
		return fmt.Errorf("available velocity is missing measured values")
	}
	velocity := *participant.Velocity
	if math.IsNaN(velocity) || math.IsInf(velocity, 0) || velocity < 0 || velocity > playerstate.MaxPlausibleHorizontalSpeedUPS {
		return fmt.Errorf("velocity is non-finite, negative, or implausible")
	}
	if *participant.VelocityObservedTick < 0 || *participant.VelocityObservedTick > engagementEndTick {
		return fmt.Errorf("velocity observation tick is invalid or comes from the future")
	}
	if participant.VelocityObservation != models.VelocityObservationCurrentTick &&
		participant.VelocityObservation != models.VelocityObservationLastAlive {
		return fmt.Errorf("available velocity has observation %q", participant.VelocityObservation)
	}

	switch playerstate.VelocitySource(participant.VelocitySource) {
	case playerstate.VelocitySourceNative:
		if *participant.VelocityMeasurementTicks != 0 {
			return fmt.Errorf("native velocity must use a zero-tick measurement window")
		}
	case playerstate.VelocitySourcePositionDelta:
		if *participant.VelocityMeasurementTicks < 1 || *participant.VelocityMeasurementTicks > playerstate.MaxPositionDeltaIntervalTicks {
			return fmt.Errorf("position delta has invalid measurement window")
		}
	default:
		return fmt.Errorf("available velocity has source %q", participant.VelocitySource)
	}

	expectedEngagementType := "hold"
	if velocity > models.EngagementPeekVelocityThresholdUPS {
		expectedEngagementType = "peek"
	}
	if participant.EngagementType != expectedEngagementType {
		return fmt.Errorf("engagement type %q does not match velocity %.3f", participant.EngagementType, velocity)
	}
	return nil
}

func validateCanonicalUnavailableDuelMotion(participant models.AI_DuelParticipant) error {
	if participant.Velocity != nil || participant.VelocityMeasurementTicks != nil || participant.VelocityObservedTick != nil {
		return fmt.Errorf("unavailable velocity contains measured values")
	}
	if participant.VelocityObservation != models.VelocityObservationUnavailable {
		return fmt.Errorf("unavailable velocity has observation %q", participant.VelocityObservation)
	}
	if !canonicalVelocitySourceIsUnavailable(playerstate.VelocitySource(participant.VelocitySource)) {
		return fmt.Errorf("unavailable velocity has source %q", participant.VelocitySource)
	}
	if participant.EngagementType != "" {
		return fmt.Errorf("unavailable velocity has engagement type %q", participant.EngagementType)
	}
	return nil
}

func canonicalVelocitySourceIsUnavailable(source playerstate.VelocitySource) bool {
	switch source {
	case playerstate.VelocitySourceInsufficientHistory,
		playerstate.VelocitySourceNotApplicable,
		playerstate.VelocitySourceRejected,
		playerstate.VelocitySourceStaleGap,
		playerstate.VelocitySourceEntityChanged,
		playerstate.VelocitySourceNonMonotonicTick:
		return true
	default:
		return false
	}
}

func validateCanonicalDuelParticipantWeapon(participant models.AI_DuelParticipant, engagementEndTick int) error {
	switch participant.ActiveWeaponObservation {
	case models.ActiveWeaponObservationObservedCurrent, models.ActiveWeaponObservationLastObserved:
		if participant.ActiveWeapon == nil || strings.TrimSpace(*participant.ActiveWeapon) == "" || participant.ActiveWeaponObservedTick == nil {
			return fmt.Errorf("observed active weapon is missing its value or tick")
		}
		if *participant.ActiveWeaponObservedTick < 0 || *participant.ActiveWeaponObservedTick > engagementEndTick {
			return fmt.Errorf("active weapon observation tick is invalid or comes from the future")
		}
	case models.ActiveWeaponObservationUnavailable:
		if participant.ActiveWeapon != nil || participant.ActiveWeaponObservedTick != nil {
			return fmt.Errorf("unavailable active weapon contains an observed value")
		}
	default:
		return fmt.Errorf("invalid active weapon observation %q", participant.ActiveWeaponObservation)
	}
	return nil
}

func buildCanonicalPlayerRoundEconomy(ctx *models.DemoContext, matchID string, rosters canonicalRosterInfo) models.CanonicalPlayerRoundEconomyExport {
	records := make([]models.CanonicalPlayerRoundEconomy, 0, len(ctx.AI_EconomyRounds)*10)
	rounds := make([]models.CanonicalEconomyRoundContext, 0, len(ctx.AI_EconomyRounds))
	for _, round := range ctx.AI_EconomyRounds {
		rounds = append(rounds, models.CanonicalEconomyRoundContext{
			RoundID: canonicalRoundID(matchID, round.Round), RoundNumber: round.Round,
			Teams: round.Teams, Events: canonicalEconomyEvents(round.Events),
		})
		players := append([]models.AI_EconomyPlayer(nil), round.Players...)
		sort.Slice(players, func(i, j int) bool { return players[i].SteamID < players[j].SteamID })
		for _, player := range players {
			teamID := rosters.playerTeams[player.SteamID]
			if teamID == "" {
				teamID = "unknown"
			}
			records = append(records, models.CanonicalPlayerRoundEconomy{
				RoundID: canonicalRoundID(matchID, round.Round), RoundNumber: round.Round,
				PlayerID: canonicalPlayerID(player.SteamID), TeamID: teamID, Side: normalizeSide(player.Team),
				InitialMoney: player.InitialMoney, MoneyAfterBuy: player.MoneyAfterBuy, SpentInBuy: player.SpentInBuy,
				EquipmentValueStart:     player.EquipmentValueStartCalculated,
				EquipmentValueFreezeEnd: player.FinalEquipmentValueCalculated,
				MoneyAtRoundEnd:         player.MoneyAtRoundEnd,
				EquipmentValueRoundEnd:  player.EquipmentValueEndCalculated,
				Survived:                player.Survived, Outcome: strings.ToLower(player.Outcome), Source: "calculated",
				Details: canonicalEconomyPlayerDetails(player),
			})
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].RoundNumber != records[j].RoundNumber {
			return records[i].RoundNumber < records[j].RoundNumber
		}
		return records[i].PlayerID < records[j].PlayerID
	})
	sort.SliceStable(rounds, func(i, j int) bool { return rounds[i].RoundNumber < rounds[j].RoundNumber })
	return models.CanonicalPlayerRoundEconomyExport{
		SchemaID: "stratai.player_round_economy@1", MatchID: matchID, Rounds: rounds, Records: records,
	}
}

func canonicalEconomyPlayerDetails(player models.AI_EconomyPlayer) models.AI_EconomyPlayer {
	player.StartRoundItems = sortedEconomyItems(player.StartRoundItems)
	player.Purchases = sortedEconomyItems(player.Purchases)
	player.FinalEquipment = sortedEconomyItems(player.FinalEquipment)
	player.EndEquipment = sortedEconomyItems(player.EndEquipment)
	player.Refunds = append([]string(nil), player.Refunds...)
	sort.Strings(player.Refunds)
	return player
}

func sortedEconomyItems(items []models.AI_WeaponItem) []models.AI_WeaponItem {
	result := append([]models.AI_WeaponItem(nil), items...)
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].EntityID != result[right].EntityID {
			return result[left].EntityID < result[right].EntityID
		}
		if result[left].Weapon != result[right].Weapon {
			return result[left].Weapon < result[right].Weapon
		}
		if result[left].OriginalOwner != result[right].OriginalOwner {
			return result[left].OriginalOwner < result[right].OriginalOwner
		}
		return result[left].Price < result[right].Price
	})
	return result
}

func canonicalEconomyEvents(events *models.AI_EconomyRoundEvents) *models.AI_EconomyRoundEvents {
	if events == nil {
		return nil
	}
	result := &models.AI_EconomyRoundEvents{
		Drops:   append([]models.AI_EconomyDrop(nil), events.Drops...),
		Pickups: append([]models.AI_EconomyPickup(nil), events.Pickups...),
		Refunds: append([]models.AI_EconomyRefund(nil), events.Refunds...),
	}
	sort.SliceStable(result.Drops, func(left, right int) bool {
		if result.Drops[left].Tick != result.Drops[right].Tick {
			return result.Drops[left].Tick < result.Drops[right].Tick
		}
		if result.Drops[left].EntityID != result.Drops[right].EntityID {
			return result.Drops[left].EntityID < result.Drops[right].EntityID
		}
		if result.Drops[left].DropperID != result.Drops[right].DropperID {
			return result.Drops[left].DropperID < result.Drops[right].DropperID
		}
		return result.Drops[left].Weapon < result.Drops[right].Weapon
	})
	sort.SliceStable(result.Pickups, func(left, right int) bool {
		if result.Pickups[left].Tick != result.Pickups[right].Tick {
			return result.Pickups[left].Tick < result.Pickups[right].Tick
		}
		if result.Pickups[left].EntityID != result.Pickups[right].EntityID {
			return result.Pickups[left].EntityID < result.Pickups[right].EntityID
		}
		if result.Pickups[left].PlayerID != result.Pickups[right].PlayerID {
			return result.Pickups[left].PlayerID < result.Pickups[right].PlayerID
		}
		return result.Pickups[left].Weapon < result.Pickups[right].Weapon
	})
	sort.SliceStable(result.Refunds, func(left, right int) bool {
		if result.Refunds[left].Tick != result.Refunds[right].Tick {
			return result.Refunds[left].Tick < result.Refunds[right].Tick
		}
		if result.Refunds[left].PlayerID != result.Refunds[right].PlayerID {
			return result.Refunds[left].PlayerID < result.Refunds[right].PlayerID
		}
		return result.Refunds[left].Weapon < result.Refunds[right].Weapon
	})
	return result
}

func buildCanonicalPlayerMatchStats(
	ctx *models.DemoContext,
	matchID string,
	trades models.CanonicalTradesExport,
	combatEvents []models.CanonicalCombatEvent,
	canonicalRounds ...[]models.CanonicalRound,
) models.CanonicalPlayerMatchStatsExport {
	stats := make([]models.AI_PlayerStats, 0, len(ctx.AI_PlayersSummary))
	for _, player := range ctx.AI_PlayersSummary {
		steamID, err := strconv.ParseUint(player.SteamID, 10, 64)
		if (err != nil || steamID == 0) && !canonicalPlayerStatsContainData(player) {
			continue
		}
		stats = append(stats, player)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].SteamID < stats[j].SteamID })
	if len(canonicalRounds) > 0 && ctx != nil && ctx.Combat != nil {
		applyCanonicalCombatProjection(stats, combat.Summaries(
			filterCanonicalCombatLedgerToRounds(ctx.Combat.Snapshot(), canonicalRounds[0]),
		))
	}
	statsByPlayerID := make(map[string]*models.AI_PlayerStats, len(stats))
	for index := range stats {
		stats[index].TradeKills = 0
		stats[index].TradedDeaths = 0
		stats[index].TradeAttempts = 0
		stats[index].FailedTradeAttempts = 0
		stats[index].UntradeableDeaths = 0
		stats[index].NonEvaluableTradeDeaths = 0
		statsByPlayerID["steam:"+stats[index].SteamID] = &stats[index]
	}
	eventByID := make(map[string]models.CanonicalCombatEvent, len(combatEvents))
	for _, event := range combatEvents {
		eventByID[event.EventID] = event
	}
	completionByCandidate := make(map[string]models.CanonicalTradeCompletion, len(trades.Completions))
	for _, completion := range trades.Completions {
		completionByCandidate[completion.TradeCandidateID] = completion
	}
	for _, candidate := range trades.Candidates {
		victim := statsByPlayerID[candidate.OriginalVictimPlayerID]
		if victim != nil {
			switch candidate.Evaluation {
			case "not_tradeable":
				victim.UntradeableDeaths++
			case "not_evaluable":
				victim.NonEvaluableTradeDeaths++
			}
		}
		attemptActors := make(map[string]struct{})
		for _, eventID := range candidate.AttemptEventIDs {
			event, exists := eventByID[eventID]
			if !exists || event.ActorPlayerID == nil {
				continue
			}
			attemptActors[*event.ActorPlayerID] = struct{}{}
		}
		if completion, exists := completionByCandidate[candidate.TradeCandidateID]; exists {
			attemptActors[completion.TraderPlayerID] = struct{}{}
		}
		for playerID := range attemptActors {
			if player := statsByPlayerID[playerID]; player != nil {
				player.TradeAttempts++
				if candidate.Evaluation == "failed" {
					player.FailedTradeAttempts++
				}
			}
		}
	}
	for _, completion := range trades.Completions {
		if trader := statsByPlayerID[completion.TraderPlayerID]; trader != nil {
			trader.TradeKills++
		}
		if victim := statsByPlayerID[completion.OriginalVictimPlayerID]; victim != nil {
			victim.TradedDeaths++
		}
	}
	players := make([]models.CanonicalPlayerMatchStats, 0, len(stats))
	for _, player := range stats {
		players = append(players, models.CanonicalPlayerMatchStats{
			PlayerID: "steam:" + player.SteamID, Source: "post_match_derived_engagement_trade", Metrics: player,
		})
	}
	return models.CanonicalPlayerMatchStatsExport{SchemaID: "stratai.player_match_stats@2", MatchID: matchID, Players: players}
}

func applyCanonicalCombatProjection(stats []models.AI_PlayerStats, summaries map[uint64]combat.PlayerSummary) {
	for index := range stats {
		player := &stats[index]
		player.KillsObserved = 0
		player.DeathsObserved = 0
		player.AssistsObserved = 0
		player.KillsNativeMinusObserved = player.NativeScoreboard.Kills
		player.DeathsNativeMinusObserved = player.NativeScoreboard.Deaths
		player.AssistsNativeMinusObserved = player.NativeScoreboard.Assists
		player.Headshots = 0
		player.FlashAssists = 0
		player.CombatDamageObserved = 0
		player.CombatDamageUnattributedDelta = player.NativeScoreboard.TotalDamage
		player.FriendlyDamage = 0
		player.SelfDamage = 0
		player.ShotsFired = 0
		player.ShotsHit = 0
		player.ShotsMissed = 0
		player.BodyPartHits = make(map[string]int)
		player.WeaponStats = make(map[string]models.AI_WeaponStat)

		steamID, err := strconv.ParseUint(player.SteamID, 10, 64)
		if err != nil {
			continue
		}
		summary, exists := summaries[steamID]
		if !exists {
			continue
		}
		player.KillsObserved = summary.Kills
		player.DeathsObserved = summary.Deaths
		player.AssistsObserved = summary.Assists
		player.KillsNativeMinusObserved = player.NativeScoreboard.Kills - summary.Kills
		player.DeathsNativeMinusObserved = player.NativeScoreboard.Deaths - summary.Deaths
		player.AssistsNativeMinusObserved = player.NativeScoreboard.Assists - summary.Assists
		player.Headshots = summary.Headshots
		player.FlashAssists = summary.FlashAssists
		player.CombatDamageObserved = summary.EnemyDamage
		player.CombatDamageUnattributedDelta = player.NativeScoreboard.TotalDamage - summary.EnemyDamage
		player.FriendlyDamage = summary.FriendlyDamage
		player.SelfDamage = summary.SelfDamage
		player.ShotsFired = summary.ShotsFired
		player.ShotsHit = summary.ShotsHit
		player.ShotsMissed = summary.ShotsMissed
		for hitgroup, count := range summary.BodyPartHits {
			player.BodyPartHits[hitgroup] = count
		}
		for weapon, weaponSummary := range summary.WeaponStats {
			player.WeaponStats[weapon] = models.AI_WeaponStat{
				Kills: weaponSummary.Kills, Headshots: weaponSummary.Headshots, Damage: weaponSummary.Damage,
				ShotsFired: weaponSummary.ShotsFired, ShotsHit: weaponSummary.ShotsHit, ShotsMissed: weaponSummary.ShotsMissed,
			}
		}
	}
}

func canonicalPlayerStatsContainData(player models.AI_PlayerStats) bool {
	player.SteamID = ""
	player.Name = ""
	player.Team = ""
	player.NativeScoreboardStatus = ""
	if len(player.MultiKills) == 0 {
		player.MultiKills = nil
	}
	if len(player.GrenadeDamage) == 0 {
		player.GrenadeDamage = nil
	}
	if len(player.BodyPartHits) == 0 {
		player.BodyPartHits = nil
	}
	if len(player.WeaponStats) == 0 {
		player.WeaponStats = nil
	}
	return !reflect.DeepEqual(player, models.AI_PlayerStats{})
}

func canonicalRoundID(matchID string, round int) string {
	return fmt.Sprintf("%s:round:%03d", matchID, round)
}

func canonicalPlayerID(steamID uint64) string {
	return "steam:" + strconv.FormatUint(steamID, 10)
}

func canonicalEventID(matchID, domain string, round, tick, ordinal int) string {
	return fmt.Sprintf("%s:%s:%03d:%09d:%03d", matchID, domain, round, tick, ordinal)
}

func canonicalVector(vector models.AI_Vector) models.CanonicalVector {
	return models.CanonicalVector{X: vector.X, Y: vector.Y, Z: vector.Z}
}

func normalizeSide(side string) string {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "ct", "counterterrorists", "counter-terrorists":
		return "ct"
	case "t", "terrorists":
		return "t"
	default:
		return "unknown"
	}
}

func normalizeUtilityType(utilityType string) string {
	normalized := strings.ToLower(strings.TrimSpace(utilityType))
	switch normalized {
	case "flash", "flashbang":
		return "flashbang"
	case "smoke", "smokegrenade":
		return "smoke"
	case "he", "hegrenade", "high explosive grenade":
		return "he_grenade"
	case "molotov", "incendiary", "incendiarygrenade":
		return "molotov"
	case "decoy", "decoygrenade":
		return "decoy"
	default:
		return strings.ReplaceAll(normalized, " ", "_")
	}
}

func normalizeObjectiveType(eventType string) string {
	normalized := strings.ToLower(strings.TrimSpace(eventType))
	if strings.HasPrefix(normalized, "bomb_") {
		return normalized
	}
	switch normalized {
	case "plant", "planted":
		return "bomb_plant"
	case "defuse", "defused":
		return "bomb_defuse"
	case "explode", "exploded":
		return "bomb_explode"
	default:
		return "bomb_" + strings.ReplaceAll(normalized, " ", "_")
	}
}

var canonicalRoundEndReasons = map[int]string{
	0:  "still_in_progress",
	1:  "target_bombed",
	2:  "vip_escaped",
	3:  "vip_killed",
	4:  "terrorists_escaped",
	5:  "ct_stopped_escape",
	6:  "terrorists_stopped",
	7:  "bomb_defused",
	8:  "ct_win",
	9:  "terrorists_win",
	10: "draw",
	11: "hostages_rescued",
	12: "target_saved",
	13: "hostages_not_rescued",
	14: "terrorists_not_escaped",
	15: "vip_not_escaped",
	16: "game_start",
	17: "terrorists_surrender",
	18: "ct_surrender",
	19: "terrorists_planted",
	20: "cts_reached_hostage",
}

func canonicalRoundEndReason(raw string) (*string, *int) {
	trimmed := strings.TrimSpace(raw)
	code, err := strconv.Atoi(trimmed)
	if err != nil {
		return nullableString(trimmed), nil
	}
	label, exists := canonicalRoundEndReasons[code]
	if !exists {
		label = "unknown"
	}
	return stringPointer(label), intPointer(code)
}

func intPointer(value int) *int          { return &value }
func boolPointer(value bool) *bool       { return &value }
func stringPointer(value string) *string { return &value }

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return stringPointer(value)
}

func positiveFloatPointer(value float64) *float64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	return &value
}
