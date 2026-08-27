package parser

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"cs2-demo-service/models"
	combatledger "cs2-demo-service/pkg/combat"
	"cs2-demo-service/pkg/objective"
	utilityledger "cs2-demo-service/pkg/utility"
)

func TestBuildCanonicalPlayerStatesSkipsAnonymousPlayer(t *testing.T) {
	ctx := &models.DemoContext{
		CurrentRound: 1,
		AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{
			{Round: 1, Event: models.AI_TrackingEvent{Tick: 10, PlayerSteamID: 0}},
			{Round: 1, Event: models.AI_TrackingEvent{Tick: 10, PlayerSteamID: 42}},
		},
	}
	rosters := canonicalRosterInfo{playerTeams: map[uint64]string{42: "team_a"}}

	states := buildCanonicalPlayerStates(ctx, "match-test", rosters)

	if len(states[1]) != 1 {
		t.Fatalf("expected one identified player state, got %d", len(states[1]))
	}
	if states[1][0].PlayerID != "steam:42" {
		t.Fatalf("unexpected player id %q", states[1][0].PlayerID)
	}
}

func TestCanonicalRosterIdentityQualityRejectsProlongedAnonymousRun(t *testing.T) {
	ctx := &models.DemoContext{}
	for tick := 0; tick <= 12*64; tick += 32 {
		ctx.AI_TrackingEventsWithRound = append(
			ctx.AI_TrackingEventsWithRound,
			models.AI_TrackingEventWithRound{
				Round: 3,
				Event: models.AI_TrackingEvent{Tick: tick, PlayerSteamID: 0},
			},
		)
	}

	if err := validateCanonicalRosterIdentityQuality(ctx, 64); err == nil {
		t.Fatal("expected prolonged anonymous participant to fail roster identity quality")
	}
}

func TestCanonicalRosterIdentityQualityAllowsBriefDisconnectedObservations(t *testing.T) {
	ctx := &models.DemoContext{AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{
		{Round: 1, Event: models.AI_TrackingEvent{Tick: 0, PlayerSteamID: 0}},
		{Round: 1, Event: models.AI_TrackingEvent{Tick: 4 * 64, PlayerSteamID: 0}},
		{Round: 1, Event: models.AI_TrackingEvent{Tick: 8 * 64, PlayerSteamID: 0}},
	}}

	if err := validateCanonicalRosterIdentityQuality(ctx, 64); err != nil {
		t.Fatalf("brief disconnected observations should not fail: %v", err)
	}
}

func TestCanonicalRoundBoundsReplacePreliminaryEpochData(t *testing.T) {
	start, end := 100, 200
	rounds := []models.CanonicalRound{{RoundNumber: 1, StartTick: &start, EndTick: &end}}
	combatEvents := []combatledger.Event{
		{LocalID: "preliminary", Round: 1, Tick: 20, SequenceInRound: 1, SequenceInTick: 1},
		{LocalID: "competitive", Round: 1, Tick: 120, SequenceInRound: 2, SequenceInTick: 1},
	}
	filteredCombat := filterCanonicalCombatLedgerToRounds(combatEvents, rounds)
	if len(filteredCombat) != 1 || filteredCombat[0].LocalID != "competitive" ||
		filteredCombat[0].SequenceInRound != 1 || filteredCombat[0].SequenceInTick != 1 {
		t.Fatalf("preliminary combat epoch was not replaced: %+v", filteredCombat)
	}

	throws := []utilityledger.Throw{
		{ID: "r1-u0001", Round: 1, Sequence: 1, Launch: utilityledger.ThrowSnapshot{Tick: utilityledger.TickObservation{Tick: 30, Status: utilityledger.AvailabilityObserved}}},
		{ID: "r1-u0002", Round: 1, Sequence: 2, Launch: utilityledger.ThrowSnapshot{Tick: utilityledger.TickObservation{Tick: 130, Status: utilityledger.AvailabilityObserved}}},
	}
	filteredThrows := filterCanonicalUtilityThrowsToRounds(throws, rounds)
	if len(filteredThrows) != 1 || filteredThrows[0].ID != "r1-u0001" || filteredThrows[0].Sequence != 1 || filteredThrows[0].Launch.Tick.Tick != 130 {
		t.Fatalf("preliminary utility epoch was not replaced and resequenced: %+v", filteredThrows)
	}

	states := map[int][]models.CanonicalPlayerState{
		1: {{RoundNumber: 1, Tick: 40}, {RoundNumber: 1, Tick: 140}},
	}
	filteredStates := filterCanonicalPlayerStatesToRounds(states, rounds)
	if len(filteredStates[1]) != 1 || filteredStates[1][0].Tick != 140 {
		t.Fatalf("preliminary player-state epoch was not replaced: %+v", filteredStates)
	}
}

func TestApplyCanonicalCombatProjectionUsesFilteredSummaries(t *testing.T) {
	stats := []models.AI_PlayerStats{{
		SteamID: "7",
		NativeScoreboard: models.AI_NativePlayerStats{
			Kills: 2, Deaths: 1, TotalDamage: 100,
		},
		KillsObserved: 9,
	}}
	summaries := map[uint64]combatledger.PlayerSummary{
		7: {Kills: 2, Deaths: 1, EnemyDamage: 100, ShotsFired: 3, ShotsHit: 2, ShotsMissed: 1},
	}

	applyCanonicalCombatProjection(stats, summaries)

	got := stats[0]
	if got.KillsObserved != 2 || got.DeathsObserved != 1 || got.CombatDamageObserved != 100 ||
		got.KillsNativeMinusObserved != 0 || got.CombatDamageUnattributedDelta != 0 ||
		got.ShotsFired != 3 || got.ShotsHit != 2 || got.ShotsMissed != 1 {
		t.Fatalf("filtered combat projection did not reconcile: %+v", got)
	}
}

func TestExportCanonicalBundleWritesDeterministicReferentialBundle(t *testing.T) {
	matchID := "match-test"
	ctx := canonicalTestContext()
	matchDir := t.TempDir()
	quality := map[string]interface{}{"status": "pass", "usable_for_training": true}

	if err := ExportCanonicalBundle(ctx, matchID, matchDir, "2026-08-13T10:00:00Z", 42.5, 64, quality); err != nil {
		t.Fatalf("ExportCanonicalBundle() error = %v", err)
	}

	canonicalDir := filepath.Join(matchDir, "canonical")
	manifest := readCanonicalJSON[models.CanonicalManifest](t, filepath.Join(canonicalDir, "manifest.json"))
	if manifest.SchemaID != "stratai.canonical_manifest@3" || manifest.ExportFormatVersion != "3.8.0" || manifest.ParserVersion != "v16" || len(manifest.ConfigurationHashes) == 0 {
		t.Fatalf("unexpected manifest version: %+v", manifest)
	}
	if len(manifest.Artifacts) != 25 {
		t.Fatalf("artifact count = %d, want 25", len(manifest.Artifacts))
	}
	paths := make([]string, 0, len(manifest.Artifacts))
	playerStateSchema := ""
	tacticalObservationsFound := false
	for _, artifact := range manifest.Artifacts {
		paths = append(paths, artifact.Path)
		if artifact.ArtifactType == "player_states" {
			playerStateSchema = artifact.SchemaID
		}
		if artifact.SortOrder == nil {
			t.Errorf("sort_order must be an array for %s", artifact.Path)
		}
		if artifact.ArtifactType == "tactical_observations" {
			tacticalObservationsFound = true
			if artifact.Path != "states/tactical/observed.jsonl.gz" || artifact.Format != "jsonl" || artifact.Compression != "gzip" {
				t.Errorf("unexpected tactical observation encoding: %+v", artifact)
			}
		}
		path := filepath.Join(canonicalDir, filepath.FromSlash(artifact.Path))
		checksum, err := checksumFile(path)
		if err != nil {
			t.Fatalf("checksum %s: %v", artifact.Path, err)
		}
		if checksum != artifact.SHA256 {
			t.Errorf("checksum mismatch for %s", artifact.Path)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", artifact.Path, err)
		}
		if info.Size() != artifact.Bytes {
			t.Errorf("size mismatch for %s", artifact.Path)
		}
	}
	if !sort.StringsAreSorted(paths) {
		t.Fatalf("manifest paths are not deterministic: %v", paths)
	}
	if playerStateSchema != "stratai.player_state@3" {
		t.Fatalf("player state artifact schema = %q", playerStateSchema)
	}
	if !tacticalObservationsFound {
		t.Fatal("tactical_observations artifact is missing")
	}

	match := readCanonicalJSON[map[string]interface{}](t, filepath.Join(canonicalDir, "core", "match.json"))
	if _, nested := match["match"]; nested {
		t.Fatal("match fields must be top-level, not nested under match")
	}
	if match["round_count"] != float64(2) {
		t.Fatalf("round_count = %v, want 2", match["round_count"])
	}
	teams := match["teams"].([]interface{})
	if teams[0].(map[string]interface{})["team_id"] != "team_a" || teams[0].(map[string]interface{})["starting_side"] != "ct" {
		t.Fatalf("unexpected stable roster: %v", teams[0])
	}

	participants := readCanonicalJSON[models.CanonicalParticipantsExport](t, filepath.Join(canonicalDir, "core", "participants.json"))
	gotPlayerIDs := make([]string, 0, len(participants.Players))
	for _, player := range participants.Players {
		gotPlayerIDs = append(gotPlayerIDs, player.PlayerID)
		if player.TeamID == "" || player.TeamID == "unknown" {
			t.Errorf("player %s has no stable team", player.PlayerID)
		}
	}
	wantPlayerIDs := []string{"steam:11", "steam:22", "steam:33", "steam:44"}
	if !reflect.DeepEqual(gotPlayerIDs, wantPlayerIDs) {
		t.Fatalf("player IDs = %v, want %v", gotPlayerIDs, wantPlayerIDs)
	}

	rounds := readCanonicalJSON[models.CanonicalRoundsExport](t, filepath.Join(canonicalDir, "core", "rounds.json"))
	if len(rounds.Rounds) != 2 || rounds.Rounds[1].RoundID != matchID+":round:002" {
		t.Fatalf("unexpected canonical rounds: %+v", rounds.Rounds)
	}
	wantAssignments := []models.CanonicalSideAssignment{{TeamID: "team_a", Side: "t"}, {TeamID: "team_b", Side: "ct"}}
	if !reflect.DeepEqual(rounds.Rounds[1].SideAssignments, wantAssignments) {
		t.Fatalf("round 2 assignments = %+v, want %+v", rounds.Rounds[1].SideAssignments, wantAssignments)
	}
	if rounds.Rounds[0].WinnerSide == nil || *rounds.Rounds[0].WinnerSide != "ct" {
		t.Fatalf("replay winner must override a conflicting legacy round: %+v", rounds.Rounds[0])
	}
	if teams[0].(map[string]interface{})["score"] != float64(1) || teams[1].(map[string]interface{})["score"] != float64(1) {
		t.Fatalf("stable scores must map the final side score: %v", teams)
	}
	if !reflect.DeepEqual(
		rounds.Rounds[1].TeamScoresAfter,
		[]models.CanonicalTeamScore{{TeamID: "team_a", Score: 1}, {TeamID: "team_b", Score: 1}},
	) {
		t.Fatalf("round stable score = %+v", rounds.Rounds[1].TeamScoresAfter)
	}

	combat := readCanonicalJSONLine[models.CanonicalCombatEvent](t, filepath.Join(canonicalDir, "events", "combat_events.jsonl"))
	if combat.EventID != matchID+":combat:001:000000100:001" || combat.ActorPlayerID == nil || *combat.ActorPlayerID != "steam:11" {
		t.Fatalf("unexpected combat event: %+v", combat)
	}
	utility := readCanonicalJSONLine[models.CanonicalUtilityEvent](t, filepath.Join(canonicalDir, "events", "utility_events.jsonl"))
	if utility.UtilityType != "flashbang" || utility.Lifecycle.Detonation.Tick == nil || *utility.Lifecycle.Detonation.Tick != 95 {
		t.Fatalf("unexpected utility event: %+v", utility)
	}
	if utility.Details.Extinguished || utility.Details.Thrower != "A" {
		t.Fatalf("utility detail projection was not preserved: %+v", utility.Details)
	}
	state := readCanonicalJSONLine[models.CanonicalPlayerState](t, filepath.Join(canonicalDir, "states", "player_states", "round_001.jsonl"))
	if state.StateID != matchID+":state:001:000000090:steam:11" || state.TeamID != "team_a" {
		t.Fatalf("unexpected player state: %+v", state)
	}
	if state.SchemaID != "stratai.player_state@3" || state.HorizontalVelocityWorldUPS == nil || *state.HorizontalVelocityWorldUPS != 64 {
		t.Fatalf("player state motion contract was not preserved: %+v", state)
	}
	if state.ActiveWeapon == nil || *state.ActiveWeapon != "AK-47" || state.ActiveWeaponStatus != "observed" {
		t.Fatalf("player state weapon contract was not preserved: %+v", state)
	}
	rawState := readCanonicalJSONLine[map[string]interface{}](t, filepath.Join(canonicalDir, "states", "player_states", "round_001.jsonl"))
	if _, exists := rawState["last_observed_active_weapon"]; !exists || rawState["last_observed_active_weapon"] != nil {
		t.Fatalf("missing weapon history must be explicit null: %v", rawState)
	}
	engagements := readCanonicalJSON[models.CanonicalEngagementsExport](t, filepath.Join(canonicalDir, "derived", "engagements.json"))
	if engagements.SchemaID != "stratai.engagements@2" || len(engagements.Engagements) != 1 || len(engagements.Engagements[0].SourceEventIDs) != 2 {
		t.Fatalf("engagement provenance = %+v", engagements.Engagements)
	}
	engagement := engagements.Engagements[0]
	if engagement.Initiator.PlayerID == nil || *engagement.Initiator.PlayerID != "steam:11" ||
		engagement.FirstDamageDealer.PlayerID == nil || *engagement.FirstDamageDealer.PlayerID != "steam:11" ||
		engagement.OutcomeContext.WinnerPlayerID == nil || *engagement.OutcomeContext.WinnerPlayerID != "steam:11" ||
		len(engagement.Exchanges) != 1 || engagement.Exchanges[0].ActorPlayerID != "steam:11" {
		t.Fatalf("causal engagement roles were not preserved: %+v", engagement)
	}
	trades := readCanonicalJSON[models.CanonicalTradesExport](t, filepath.Join(canonicalDir, "derived", "trades.json"))
	if trades.SchemaID != "stratai.trades@1" || len(trades.Candidates) != 1 {
		t.Fatalf("trade candidates were not exported: %+v", trades)
	}
	economy := readCanonicalJSON[models.CanonicalEconomyPlayerExport](t, filepath.Join(canonicalDir, "derived", "economy_players.json"))
	if len(economy.Players) != 8 || economy.Players[0].TeamID != "team_a" || len(economy.Players[0].Transactions) != 1 {
		t.Fatalf("rich economy projection was not preserved: %+v", economy)
	}
	metadata := readCanonicalJSON[models.CanonicalMatchMetadata](t, filepath.Join(canonicalDir, "core", "match_metadata.json"))
	if metadata.PlayedAt == nil || metadata.PlayedAtStatus != "observed" || metadata.PriceTable.Version != block6PriceTableVersion {
		t.Fatalf("metadata provenance was not preserved: %+v", metadata)
	}
	stats := readCanonicalJSON[models.CanonicalPlayerStatsExport](t, filepath.Join(canonicalDir, "derived", "player_stats.json"))
	if stats.SchemaID != "stratai.player_stats@1" || len(stats.Players) != 4 || !stats.Players[0].Rating.Approximate {
		t.Fatalf("player stats contract was not preserved: %+v", stats)
	}
	clutches := readCanonicalJSON[models.CanonicalClutchEventExport](t, filepath.Join(canonicalDir, "derived", "clutch_events.json"))
	if clutches.SchemaID != "stratai.clutch_event@1" || clutches.Algorithm.Version != block6ClutchVersion {
		t.Fatalf("clutch contract was not preserved: %+v", clutches)
	}
	qualityDocument := readCanonicalJSON[map[string]interface{}](t, filepath.Join(canonicalDir, "diagnostics", "quality_report.json"))
	report := qualityDocument["report"].(map[string]interface{})
	combatRecordCount := -1
	for _, artifact := range manifest.Artifacts {
		if artifact.ArtifactType == "combat_events" {
			combatRecordCount = artifact.RecordCount
			break
		}
	}
	if combatRecordCount < 0 {
		t.Fatal("combat_events artifact is missing from the canonical manifest")
	}
	if report["combat_ledger_events"] != float64(combatRecordCount) {
		t.Fatalf(
			"quality combat_ledger_events = %v, want exported record count %d",
			report["combat_ledger_events"],
			combatRecordCount,
		)
	}
	checks := make(map[string]string)
	for _, rawCheck := range report["checks"].([]interface{}) {
		check := rawCheck.(map[string]interface{})
		checks[check["name"].(string)] = check["status"].(string)
	}
	for _, name := range []string{
		"economy_team_identity", "economy_native_calculated_reconciliation", "economy_money_transition",
		"economy_purchase_provenance", "economy_price_table_version", "stats_scoreboard_reconciliation",
		"stats_utility_reconciliation", "clutch_attempt_reconciliation", "warmup_contamination",
		"metadata_provenance", "metadata_checksum_lineage", "economy_determinism", "stats_determinism",
		"economy_observation_coverage",
	} {
		if _, ok := checks[name]; !ok {
			t.Errorf("missing Block 6 quality gate %s", name)
		}
	}
}

func TestWriteGzipJSONLinesIsByteDeterministicAndLossless(t *testing.T) {
	records := []map[string]interface{}{
		{"schema_id": "test@1", "tick": 4, "value": "repeated repeated repeated"},
		{"schema_id": "test@1", "tick": 8, "value": "repeated repeated repeated"},
	}
	firstPath := filepath.Join(t.TempDir(), "first.jsonl.gz")
	secondPath := filepath.Join(t.TempDir(), "second.jsonl.gz")

	firstCount, err := writeGzipJSONLines(firstPath, records)
	if err != nil {
		t.Fatalf("write first gzip JSONL: %v", err)
	}
	secondCount, err := writeGzipJSONLines(secondPath, records)
	if err != nil {
		t.Fatalf("write second gzip JSONL: %v", err)
	}
	if firstCount != len(records) || secondCount != len(records) {
		t.Fatalf("record counts = %d, %d; want %d", firstCount, secondCount, len(records))
	}

	firstBytes, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read first gzip JSONL: %v", err)
	}
	secondBytes, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("read second gzip JSONL: %v", err)
	}
	if !reflect.DeepEqual(firstBytes, secondBytes) {
		t.Fatal("gzip JSONL output is not byte deterministic")
	}

	file, err := os.Open(firstPath)
	if err != nil {
		t.Fatalf("open gzip JSONL: %v", err)
	}
	decompressed, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		t.Fatalf("open gzip stream: %v", err)
	}
	scanner := bufio.NewScanner(decompressed)
	decoded := make([]map[string]interface{}, 0, len(records))
	for scanner.Scan() {
		var record map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("decode gzip JSONL: %v", err)
		}
		decoded = append(decoded, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan gzip JSONL: %v", err)
	}
	if err := decompressed.Close(); err != nil {
		t.Fatalf("close gzip stream: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close gzip JSONL: %v", err)
	}
	if len(decoded) != len(records) || decoded[0]["tick"] != float64(4) || decoded[1]["tick"] != float64(8) {
		t.Fatalf("decoded records = %+v", decoded)
	}
}

func TestCanonicalOutcomeValidationRejectsShiftedRound(t *testing.T) {
	ctx := canonicalTestContext()
	rosters := buildCanonicalRosters(ctx)
	match := buildCanonicalMatch(ctx, "test", "", 0, 64, rosters)
	rounds := buildCanonicalRounds(ctx, "test", rosters, nil)
	rounds.Rounds[0].WinnerTeamID = stringPointer("team_b")

	if err := validateCanonicalOutcomeConsistency(match, rounds, ctx.ReplayData); err == nil {
		t.Fatal("shifted or inconsistent winner must fail before export commit")
	}
}

func TestCanonicalObjectiveLifecycleReconcilesRoundFacts(t *testing.T) {
	tracker := objective.NewTracker()
	terrorist := objective.Actor{SteamID: 11, Name: "T", Side: "T"}
	defuser := objective.Actor{SteamID: 22, Name: "CT", Side: "CT"}
	position := objective.ObservedPosition(100, 200, 10, objective.SourceDemoinfocsEvent)
	input := func(tick int, actor objective.Actor, site string) objective.EventInput {
		return objective.EventInput{
			Round: 1, Tick: tick, Actor: actor, Site: site,
			Position: position, Source: objective.SourceDemoinfocsEvent,
		}
	}

	tracker.BeginRound(1, 10)
	tracker.NativeSnapshot(objective.NativeObservation{Round: 1, Tick: 11, Carrier: terrorist, Position: position})
	tracker.Drop(input(20, terrorist, ""))
	tracker.Pickup(input(25, terrorist, ""))
	tracker.PlantStart(input(30, terrorist, "A"))
	tracker.PlantAbort(input(35, terrorist, "A"))
	tracker.PlantStart(input(40, terrorist, "A"))
	tracker.Plant(input(45, terrorist, "A"))
	firstDefuse := input(50, defuser, "A")
	firstDefuse.HasKit = boolPointer(true)
	tracker.DefuseStart(firstDefuse)
	tracker.DefuseAbort(input(52, defuser, "A"))
	secondDefuse := input(55, defuser, "A")
	secondDefuse.HasKit = boolPointer(false)
	tracker.DefuseStart(secondDefuse)
	tracker.Defuse(input(60, defuser, "A"))
	tracker.EndRound(1, 65)

	ctx := &models.DemoContext{
		Objectives:   tracker,
		CurrentRound: 1,
		MatchData: &models.MatchData{Rounds: []models.RoundData{{
			Round: 1, Winner: "CT", Reason: "7", CTScore: 1,
			BombPlanted: true, BombSite: "A", BombTick: 45,
		}}},
	}
	events, err := buildCanonicalObjectiveEvents(ctx, "objective-test", 64)
	if err != nil {
		t.Fatalf("buildCanonicalObjectiveEvents() error = %v", err)
	}
	rounds := buildCanonicalRounds(ctx, "objective-test", canonicalRosterInfo{}, events)
	if err := validateCanonicalObjectives(rounds, events); err != nil {
		t.Fatalf("valid objective lifecycle rejected: %v", err)
	}

	if len(events) != 11 {
		t.Fatalf("objective event count = %d, want 11", len(events))
	}
	round := rounds.Rounds[0]
	if round.StartTick == nil || *round.StartTick != 10 {
		t.Fatalf("canonical round did not retain the native RoundStart boundary: %+v", round)
	}
	if !round.Objective.WasBombPlanted || round.Objective.Outcome != "defused" ||
		round.Objective.PlantAttempts != 2 || round.Objective.PlantAborts != 1 ||
		round.Objective.DefuseAttempts != 2 || round.Objective.DefuseAborts != 1 ||
		round.Objective.BombDrops != 1 || round.Objective.BombPickups != 1 {
		t.Fatalf("unexpected objective summary: %+v", round.Objective)
	}
	if round.BombPlanted == nil || !*round.BombPlanted || round.BombSite == nil || *round.BombSite != "A" ||
		round.BombTick == nil || *round.BombTick != 45 {
		t.Fatalf("legacy objective projection differs from lifecycle: %+v", round)
	}

	for _, event := range events {
		if event.SchemaID != "stratai.objective_event@2" || event.PhaseAfter == "" {
			t.Fatalf("event contract is incomplete: %+v", event)
		}
		if event.AttemptID != nil && (event.AttemptOutcome == nil || event.AttemptStartObserved == nil) {
			t.Fatalf("attempt metadata is incomplete: %+v", event)
		}
	}
}

func TestCanonicalObjectiveAllowsObservedTerminalWithoutFabricatedStart(t *testing.T) {
	tracker := objective.NewTracker()
	tracker.BeginRound(1, 10)
	tracker.Plant(objective.EventInput{
		Round: 1, Tick: 20, Actor: objective.Actor{SteamID: 11, Side: "T"}, Site: "A",
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsEvent),
		Source:   objective.SourceDemoinfocsEvent,
	})
	tracker.Defuse(objective.EventInput{
		Round: 1, Tick: 25, Actor: objective.Actor{SteamID: 22, Side: "CT"}, Site: "A",
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsEvent),
		Source:   objective.SourceDemoinfocsEvent,
	})
	tracker.EndRound(1, 30)
	ctx := &models.DemoContext{
		Objectives:   tracker,
		CurrentRound: 1,
		MatchData: &models.MatchData{Rounds: []models.RoundData{{
			Round: 1, Winner: "CT", Reason: "7", CTScore: 1,
			BombPlanted: true, BombSite: "A", BombTick: 20,
		}}},
	}
	events, err := buildCanonicalObjectiveEvents(ctx, "missing-start", 64)
	if err != nil {
		t.Fatalf("buildCanonicalObjectiveEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("objective event count = %d, want 2", len(events))
	}
	for _, event := range events {
		if event.AttemptStartObserved == nil || *event.AttemptStartObserved || event.ActionDurationMS != nil {
			t.Fatalf("missing start was not represented explicitly: %+v", events)
		}
	}
	if events[1].EventType != "bomb_defuse" || events[1].HasDefuseKit != nil {
		t.Fatalf("unobserved defuse start fabricated a kit state: %+v", events[1])
	}
	rounds := buildCanonicalRounds(ctx, "missing-start", canonicalRosterInfo{}, events)
	if err := validateCanonicalObjectives(rounds, events); err != nil {
		t.Fatalf("causal terminal without a fabricated start was rejected: %v", err)
	}
}

func TestCanonicalObjectiveDerivesMissingActorSideFromObjectiveRole(t *testing.T) {
	tracker := objective.NewTracker()
	tracker.BeginRound(1, 10)
	tracker.Plant(objective.EventInput{
		Round: 1, Tick: 20, Actor: objective.Actor{SteamID: 11}, Site: "A",
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsEvent),
		Source:   objective.SourceDemoinfocsEvent,
	})
	tracker.Defuse(objective.EventInput{
		Round: 1, Tick: 25, Actor: objective.Actor{SteamID: 22}, Site: "A",
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsEvent),
		Source:   objective.SourceDemoinfocsEvent,
	})
	tracker.EndRound(1, 30)
	ctx := &models.DemoContext{Objectives: tracker, CurrentRound: 1}

	events, err := buildCanonicalObjectiveEvents(ctx, "missing-side", 64)
	if err != nil {
		t.Fatalf("buildCanonicalObjectiveEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("objective event count = %d, want 2", len(events))
	}
	wantSides := []string{"t", "ct"}
	for index, event := range events {
		if event.ActorPlayerID == nil || event.ActorSide == nil || *event.ActorSide != wantSides[index] {
			t.Fatalf("objective actor pair was not completed from its role: %+v", event)
		}
		if err := validateCanonicalObjectiveObservation(event); err != nil {
			t.Fatalf("derived objective actor side was rejected: %v", err)
		}
	}
}

func TestCanonicalObjectiveValidationRejectsSemanticContradictions(t *testing.T) {
	ctx := &models.DemoContext{
		Objectives:   objective.NewTracker(),
		CurrentRound: 1,
		MatchData:    &models.MatchData{Rounds: []models.RoundData{{Round: 1, Winner: "CT", Reason: "8", CTScore: 1}}},
	}
	ctx.Objectives.BeginRound(1, 10)
	ctx.Objectives.EndRound(1, 20)
	events, err := buildCanonicalObjectiveEvents(ctx, "invalid-objective", 64)
	if err != nil {
		t.Fatalf("buildCanonicalObjectiveEvents() error = %v", err)
	}
	rounds := buildCanonicalRounds(ctx, "invalid-objective", canonicalRosterInfo{}, events)
	rounds.Rounds[0].Objective.WasBombPlanted = true
	if err := validateCanonicalObjectives(rounds, events); err == nil {
		t.Fatal("round summary contradiction was accepted")
	}
}

func TestBuildCanonicalCombatEventsIsDeterministicWithinTick(t *testing.T) {
	firstTracker := canonicalDeterminismCombatTracker(false)
	secondTracker := canonicalDeterminismCombatTracker(true)
	firstContext := &models.DemoContext{Combat: firstTracker}
	secondContext := &models.DemoContext{Combat: secondTracker}

	first, err := buildCanonicalCombatEvents(firstContext, "match-combat-order")
	if err != nil {
		t.Fatalf("first buildCanonicalCombatEvents() error = %v", err)
	}
	second, err := buildCanonicalCombatEvents(secondContext, "match-combat-order")
	if err != nil {
		t.Fatalf("second buildCanonicalCombatEvents() error = %v", err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first combat events: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second combat events: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("permuted same-tick events produced different JSON\nfirst:  %s\nsecond: %s", firstJSON, secondJSON)
	}
	if sha256.Sum256(firstJSON) != sha256.Sum256(secondJSON) {
		t.Fatal("permuted same-tick events produced different hashes")
	}
	for index, event := range first {
		wantID := canonicalEventID("match-combat-order", "combat", event.RoundNumber, event.Tick, index+1)
		if event.EventID != wantID || second[index].EventID != wantID {
			t.Errorf("event %d ID is not stable: first=%q second=%q want=%q", index, event.EventID, second[index].EventID, wantID)
		}
	}
}

func TestExportCanonicalBundleRejectsNonSerializableCombatSortKey(t *testing.T) {
	ctx := canonicalTestContext()
	ctx.Combat.RecordBulletDamage(combatledger.BulletDamageInput{
		Round: 1, Tick: 101, Actor: canonicalCombatPlayerRef(11, "A", "CT"),
		Target: canonicalCombatPlayerRef(33, "C", "T"), Distance: math.NaN(),
	})
	matchDir := t.TempDir()

	err := ExportCanonicalBundle(ctx, "match-invalid-combat", matchDir, "", 0, 64, nil)
	if err == nil {
		t.Fatal("ExportCanonicalBundle accepted a non-serializable combat event")
	}
	combatPath := filepath.Join(matchDir, "canonical", "events", "combat_events.jsonl")
	if _, statErr := os.Stat(combatPath); !os.IsNotExist(statErr) {
		t.Fatalf("combat artifact was written before sort-key validation: %v", statErr)
	}
}

func TestBuildCanonicalEngagementsIsDeterministicAcrossPermutedInputs(t *testing.T) {
	duels := canonicalDeterminismDuels()
	firstContext := &models.DemoContext{AI_Duels: []models.AI_Duel{
		permutedCanonicalDuel(duels["far"], "run-1-counter-9", true),
		permutedCanonicalDuel(duels["single"], "run-1-counter-3", false),
		permutedCanonicalDuel(duels["near"], "run-1-counter-7", true),
	}}
	secondContext := &models.DemoContext{AI_Duels: []models.AI_Duel{
		permutedCanonicalDuel(duels["near"], "run-2-counter-1", false),
		permutedCanonicalDuel(duels["far"], "run-2-counter-2", false),
		permutedCanonicalDuel(duels["single"], "run-2-counter-3", true),
	}}
	events := canonicalDeterminismCombatEvents()
	firstInputJSON, err := json.Marshal(firstContext.AI_Duels)
	if err != nil {
		t.Fatalf("marshal first source duels: %v", err)
	}
	secondInputJSON, err := json.Marshal(secondContext.AI_Duels)
	if err != nil {
		t.Fatalf("marshal second source duels: %v", err)
	}

	first, err := buildCanonicalEngagements(firstContext, "match-deterministic", events)
	if err != nil {
		t.Fatalf("first buildCanonicalEngagements() error = %v", err)
	}
	second, err := buildCanonicalEngagements(secondContext, "match-deterministic", reverseCanonicalCombatEvents(events))
	if err != nil {
		t.Fatalf("second buildCanonicalEngagements() error = %v", err)
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first engagements: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second engagements: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("permuted inputs produced different JSON\nfirst:  %s\nsecond: %s", firstJSON, secondJSON)
	}
	if sha256.Sum256(firstJSON) != sha256.Sum256(secondJSON) {
		t.Fatal("permuted inputs produced different engagement hashes")
	}
	wantVictimOrder := [][]string{
		{"steam:22", "steam:33"},
		{"steam:22", "steam:33"},
		{"steam:44"},
	}
	for index, want := range wantVictimOrder {
		if !reflect.DeepEqual(first.Engagements[index].VictimPlayerIDs, want) {
			t.Fatalf("engagement %d victims = %v, want %v", index, first.Engagements[index].VictimPlayerIDs, want)
		}
	}
	if first.Engagements[0].Details.Context.Distance != 100 || first.Engagements[1].Details.Context.Distance != 200 {
		t.Fatalf("full-detail tie-break order is unstable: %+v", first.Engagements)
	}

	for _, engagement := range first.Engagements {
		if engagement.Details.DuelID != engagement.EngagementID {
			t.Errorf("details duel ID = %q, want %q", engagement.Details.DuelID, engagement.EngagementID)
		}
		if !sort.StringsAreSorted(engagement.VictimPlayerIDs) {
			t.Errorf("victim IDs are not sorted: %v", engagement.VictimPlayerIDs)
		}
		if !sort.StringsAreSorted(engagement.SourceEventIDs) {
			t.Errorf("source event IDs are not sorted: %v", engagement.SourceEventIDs)
		}
		for index := 1; index < len(engagement.Details.Victims); index++ {
			previous := canonicalPlayerID(engagement.Details.Victims[index-1].SteamID)
			current := canonicalPlayerID(engagement.Details.Victims[index].SteamID)
			if previous > current {
				t.Errorf("detail victims are not sorted: %s before %s", previous, current)
			}
		}
	}
	firstInputAfterJSON, err := json.Marshal(firstContext.AI_Duels)
	if err != nil {
		t.Fatalf("marshal first source duels after export: %v", err)
	}
	secondInputAfterJSON, err := json.Marshal(secondContext.AI_Duels)
	if err != nil {
		t.Fatalf("marshal second source duels after export: %v", err)
	}
	if string(firstInputJSON) != string(firstInputAfterJSON) || string(secondInputJSON) != string(secondInputAfterJSON) {
		t.Fatal("canonical normalization mutated its source duels")
	}
}

func TestCanonicalPlayerStateValidationRejectsSemanticContradictions(t *testing.T) {
	horizontal := 10.0
	window := 1
	weapon := "AK-47"
	phaseClockMS := int64(90_000)
	valid := models.CanonicalPlayerState{
		SchemaID: "stratai.player_state@3", StateID: "state-1", RoundNumber: 1, Tick: 10,
		Position: models.CanonicalVector{}, IsAlive: true,
		HorizontalVelocityWorldUPS: &horizontal,
		VelocityVectorWorldUPS:     &models.CanonicalVector{X: 10},
		VelocitySource:             "position_delta",
		VelocityMeasurementTicks:   &window,
		ActiveWeapon:               &weapon,
		ActiveWeaponStatus:         models.ActiveWeaponStatusObserved,
		ObjectivePhase:             "preplant",
		PhaseTimeRemainingMS:       &phaseClockMS,
		RoundClockRemainingMS:      &phaseClockMS,
		RoundTimeMS:                phaseClockMS,
	}
	if err := validateCanonicalPlayerStates(map[int][]models.CanonicalPlayerState{1: {valid}}); err != nil {
		t.Fatalf("valid state rejected: %v", err)
	}

	deadWithWeapon := valid
	deadWithWeapon.IsAlive = false
	deadWithWeapon.VelocitySource = "not_applicable"
	deadWithWeapon.HorizontalVelocityWorldUPS = nil
	deadWithWeapon.VelocityVectorWorldUPS = nil
	deadWithWeapon.VelocityMeasurementTicks = nil
	if err := validateCanonicalPlayerStates(map[int][]models.CanonicalPlayerState{1: {deadWithWeapon}}); err == nil {
		t.Fatal("dead player with active weapon was accepted")
	}

	incoherentVector := valid
	incoherentVector.VelocityVectorWorldUPS = &models.CanonicalVector{X: 20}
	if err := validateCanonicalPlayerStates(map[int][]models.CanonicalPlayerState{1: {incoherentVector}}); err == nil {
		t.Fatal("incoherent scalar and vector velocity was accepted")
	}
}

func TestCanonicalEngagementObservationValidationAcceptsCausalObservations(t *testing.T) {
	attacker := canonicalValidDuelParticipant()
	historicalVictim := canonicalValidDuelParticipant()
	historicalVelocity := 100.0
	historicalWindow := 0
	historicalTick := 99
	historicalWeapon := "M4A1-S"
	historicalVictim.Velocity = &historicalVelocity
	historicalVictim.VelocitySource = "native"
	historicalVictim.VelocityObservation = models.VelocityObservationLastAlive
	historicalVictim.VelocityMeasurementTicks = &historicalWindow
	historicalVictim.VelocityObservedTick = &historicalTick
	historicalVictim.EngagementType = "hold"
	historicalVictim.ActiveWeapon = &historicalWeapon
	historicalVictim.ActiveWeaponObservation = models.ActiveWeaponObservationLastObserved
	historicalVictim.ActiveWeaponObservedTick = &historicalTick

	unavailableVictim := canonicalUnavailableDuelParticipant()
	unavailableVictim.Weapon = "m4a1"
	export := canonicalEngagementObservationFixture(attacker, historicalVictim, unavailableVictim)

	if err := validateCanonicalEngagementObservations(export); err != nil {
		t.Fatalf("valid engagement observations rejected: %v", err)
	}
}

func TestCanonicalEngagementObservationValidationRejectsSemanticContradictions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.AI_DuelParticipant)
	}{
		{name: "available velocity missing value", mutate: func(player *models.AI_DuelParticipant) { player.Velocity = nil }},
		{name: "available velocity non finite", mutate: func(player *models.AI_DuelParticipant) { value := math.NaN(); player.Velocity = &value }},
		{name: "available velocity implausible", mutate: func(player *models.AI_DuelParticipant) { value := 2001.0; player.Velocity = &value }},
		{name: "available velocity unavailable source", mutate: func(player *models.AI_DuelParticipant) { player.VelocitySource = "stale_gap" }},
		{name: "available velocity unavailable observation", mutate: func(player *models.AI_DuelParticipant) {
			player.VelocityObservation = models.VelocityObservationUnavailable
		}},
		{name: "available velocity future observation", mutate: func(player *models.AI_DuelParticipant) { tick := 101; player.VelocityObservedTick = &tick }},
		{name: "position delta zero window", mutate: func(player *models.AI_DuelParticipant) { window := 0; player.VelocityMeasurementTicks = &window }},
		{name: "native nonzero window", mutate: func(player *models.AI_DuelParticipant) { player.VelocitySource = "native" }},
		{name: "engagement type conflicts with velocity", mutate: func(player *models.AI_DuelParticipant) { player.EngagementType = "hold" }},
		{name: "observed active weapon missing value", mutate: func(player *models.AI_DuelParticipant) { player.ActiveWeapon = nil }},
		{name: "observed active weapon future tick", mutate: func(player *models.AI_DuelParticipant) { tick := 101; player.ActiveWeaponObservedTick = &tick }},
		{name: "invalid active weapon observation", mutate: func(player *models.AI_DuelParticipant) { player.ActiveWeaponObservation = "guessed" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attacker := canonicalValidDuelParticipant()
			test.mutate(&attacker)
			export := canonicalEngagementObservationFixture(attacker, canonicalUnavailableDuelParticipant())
			if err := validateCanonicalEngagementObservations(export); err == nil {
				t.Fatal("semantic contradiction was accepted")
			}
		})
	}

	t.Run("invalid victim observation", func(t *testing.T) {
		victim := canonicalUnavailableDuelParticipant()
		velocity := 0.0
		victim.Velocity = &velocity
		export := canonicalEngagementObservationFixture(canonicalValidDuelParticipant(), victim)
		if err := validateCanonicalEngagementObservations(export); err == nil {
			t.Fatal("invalid victim observation was accepted")
		}
	})

	unavailableTests := []struct {
		name   string
		mutate func(*models.AI_DuelParticipant)
	}{
		{name: "unavailable velocity with current observation", mutate: func(player *models.AI_DuelParticipant) {
			player.VelocityObservation = models.VelocityObservationCurrentTick
		}},
		{name: "unavailable velocity with available source", mutate: func(player *models.AI_DuelParticipant) { player.VelocitySource = "native" }},
		{name: "unavailable velocity classified as hold", mutate: func(player *models.AI_DuelParticipant) { player.EngagementType = "hold" }},
		{name: "unavailable weapon with value", mutate: func(player *models.AI_DuelParticipant) {
			value := "Knife"
			tick := 100
			player.ActiveWeapon = &value
			player.ActiveWeaponObservedTick = &tick
		}},
	}
	for _, test := range unavailableTests {
		t.Run(test.name, func(t *testing.T) {
			victim := canonicalUnavailableDuelParticipant()
			test.mutate(&victim)
			export := canonicalEngagementObservationFixture(canonicalValidDuelParticipant(), victim)
			if err := validateCanonicalEngagementObservations(export); err == nil {
				t.Fatal("unavailable observation contradiction was accepted")
			}
		})
	}
}

func TestExportCanonicalBundleRejectsInvalidEngagementBeforeWritingArtifact(t *testing.T) {
	ctx := canonicalTestContext()
	ctx.AI_Duels[0].Attacker.EngagementType = "hold"
	matchDir := t.TempDir()

	err := ExportCanonicalBundle(ctx, "match-invalid-engagement", matchDir, "", 0, 64, nil)
	if err == nil {
		t.Fatal("ExportCanonicalBundle accepted an invalid engagement observation")
	}
	engagementPath := filepath.Join(matchDir, "canonical", "derived", "engagements.json")
	if _, statErr := os.Stat(engagementPath); !os.IsNotExist(statErr) {
		t.Fatalf("engagement artifact was written before validation: %v", statErr)
	}
}

func canonicalEngagementObservationFixture(attacker models.AI_DuelParticipant, victims ...models.AI_DuelParticipant) models.CanonicalEngagementsExport {
	return models.CanonicalEngagementsExport{Engagements: []models.CanonicalEngagement{{
		EngagementID: "engagement-1",
		EndTick:      100,
		Details: models.AI_Duel{
			Attacker: attacker,
			Victims:  victims,
		},
	}}}
}

func canonicalValidDuelParticipant() models.AI_DuelParticipant {
	velocity := 125.0
	measurementTicks := 1
	observedTick := 100
	activeWeapon := "AK-47"
	return models.AI_DuelParticipant{
		Weapon:                   "ak47",
		ActiveWeapon:             &activeWeapon,
		ActiveWeaponObservation:  models.ActiveWeaponObservationObservedCurrent,
		ActiveWeaponObservedTick: &observedTick,
		Velocity:                 &velocity,
		VelocityAvailable:        true,
		VelocitySource:           "position_delta",
		VelocityObservation:      models.VelocityObservationCurrentTick,
		VelocityMeasurementTicks: &measurementTicks,
		VelocityObservedTick:     &observedTick,
		EngagementType:           "peek",
	}
}

func canonicalUnavailableDuelParticipant() models.AI_DuelParticipant {
	return models.AI_DuelParticipant{
		VelocitySource:          "insufficient_history",
		VelocityObservation:     models.VelocityObservationUnavailable,
		ActiveWeaponObservation: models.ActiveWeaponObservationUnavailable,
	}
}

func canonicalDeterminismDuels() map[string]models.AI_Duel {
	attacker := canonicalUnavailableDuelParticipant()
	attacker.SteamID = 11
	attacker.Name = "attacker"
	victim22 := canonicalUnavailableDuelParticipant()
	victim22.SteamID = 22
	victim22.Name = "victim-22"
	victim33 := canonicalUnavailableDuelParticipant()
	victim33.SteamID = 33
	victim33.Name = "victim-33"
	victim44 := canonicalUnavailableDuelParticipant()
	victim44.SteamID = 44
	victim44.Name = "victim-44"

	base := models.AI_Duel{
		Round: 4, TickStart: 100, TickEnd: 102, Type: "duel", Outcome: "damage",
		Attacker: attacker, VictimCount: 2,
		Victims: []models.AI_DuelParticipant{victim22, victim33},
		Exchanges: []models.AI_DuelExchange{
			{Tick: 100, Attacker: "attacker", Weapon: "ak47", Damage: 20, Hitgroup: "chest"},
			{Tick: 102, Attacker: "attacker", Weapon: "ak47", Damage: 30, Hitgroup: "head"},
		},
		Context: models.AI_DuelContext{Distance: 100},
	}
	far := base
	far.Context.Distance = 200
	single := base
	single.VictimCount = 1
	single.Victims = []models.AI_DuelParticipant{victim44}
	single.Context.Distance = 50
	return map[string]models.AI_Duel{"near": base, "far": far, "single": single}
}

func permutedCanonicalDuel(source models.AI_Duel, duelID string, reverseNested bool) models.AI_Duel {
	duel := source
	duel.DuelID = duelID
	duel.Victims = append([]models.AI_DuelParticipant(nil), source.Victims...)
	duel.Exchanges = append([]models.AI_DuelExchange(nil), source.Exchanges...)
	if reverseNested {
		for left, right := 0, len(duel.Victims)-1; left < right; left, right = left+1, right-1 {
			duel.Victims[left], duel.Victims[right] = duel.Victims[right], duel.Victims[left]
		}
		for left, right := 0, len(duel.Exchanges)-1; left < right; left, right = left+1, right-1 {
			duel.Exchanges[left], duel.Exchanges[right] = duel.Exchanges[right], duel.Exchanges[left]
		}
	}
	return duel
}

func canonicalDeterminismCombatEvents() []models.CanonicalCombatEvent {
	return []models.CanonicalCombatEvent{
		{EventID: "match-deterministic:combat:004:000000102:003", EventType: string(combatledger.EventPlayerHurt), RoundNumber: 4, Tick: 102, ActorPlayerID: stringPointer("steam:11"), TargetPlayerID: stringPointer("steam:33")},
		{EventID: "match-deterministic:combat:004:000000100:001", EventType: string(combatledger.EventPlayerHurt), RoundNumber: 4, Tick: 100, ActorPlayerID: stringPointer("steam:11"), TargetPlayerID: stringPointer("steam:22")},
		{EventID: "match-deterministic:combat:004:000000101:002", EventType: string(combatledger.EventPlayerHurt), RoundNumber: 4, Tick: 101, ActorPlayerID: stringPointer("steam:11"), TargetPlayerID: stringPointer("steam:44")},
	}
}

func canonicalDeterminismCombatTracker(reverse bool) *combatledger.Tracker {
	tracker := combatledger.NewTracker()
	inputs := []combatledger.HurtInput{
		{Round: 2, Tick: 500, Actor: canonicalCombatPlayerRef(33, "C", "CT"), Target: canonicalCombatPlayerRef(44, "D", "T"), Weapon: canonicalCombatWeapon("ak47"), HealthDamage: 18, HealthDamageTaken: 18, HealthAfter: 82, Hitgroup: "chest"},
		{Round: 2, Tick: 500, Actor: canonicalCombatPlayerRef(11, "A", "CT"), Target: canonicalCombatPlayerRef(22, "B", "T"), Weapon: canonicalCombatWeapon("m4a1"), HealthDamage: 100, HealthDamageTaken: 100, HealthAfter: 0, Hitgroup: "head"},
		{Round: 2, Tick: 500, Actor: canonicalCombatPlayerRef(11, "A", "CT"), Target: canonicalCombatPlayerRef(33, "C", "T"), Weapon: canonicalCombatWeapon("hegrenade"), HealthDamage: 35, HealthDamageTaken: 35, HealthAfter: 65, Hitgroup: "generic"},
	}
	if reverse {
		for left, right := 0, len(inputs)-1; left < right; left, right = left+1, right-1 {
			inputs[left], inputs[right] = inputs[right], inputs[left]
		}
	}
	for _, input := range inputs {
		tracker.RecordPlayerHurt(input)
	}
	return tracker
}

func canonicalCombatPlayerRef(id uint64, name, side string) combatledger.PlayerRef {
	return combatledger.PlayerRef{ID: id, Name: name, Side: side, Status: combatledger.AvailabilityObserved, Source: combatledger.SourceCallbackPlayer}
}

func canonicalCombatWeapon(name string) combatledger.WeaponRef {
	isUtility := name == "hegrenade"
	return combatledger.WeaponRef{Name: name, Status: combatledger.AvailabilityObserved, Source: combatledger.SourceCallbackWeapon, IsUtility: &isUtility}
}

func reverseRawCombatEvents(source []models.RawCombatEvent) []models.RawCombatEvent {
	reversed := append([]models.RawCombatEvent(nil), source...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed
}

func reverseCanonicalCombatEvents(source []models.CanonicalCombatEvent) []models.CanonicalCombatEvent {
	reversed := append([]models.CanonicalCombatEvent(nil), source...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed
}

func canonicalTestContext() *models.DemoContext {
	objectives := objective.NewTracker()
	objectives.BeginRound(1, 80)
	objectives.EndRound(1, 120)
	objectives.BeginRound(2, 130)
	objectives.EndRound(2, 180)
	utilities := utilityledger.NewTracker()
	utilities.BeginRound(1)
	utilities.RecordThrow(utilityledger.ThrowInput{
		Round: 1, RuntimeEntityID: 90, EntitySource: utilityledger.SourceProjectileEntity,
		Type: utilityledger.TypeFlashbang, TypeSource: utilityledger.SourceWeaponInstance,
		Actor: utilityledger.PlayerRef{
			ID: 11, Name: "A", Side: "CT", Status: utilityledger.AvailabilityObserved,
			Source: utilityledger.SourceProjectileThrower,
		},
		Launch: canonicalTestUtilityLaunch(90),
	})
	utilities.RecordDetonation(utilityledger.CallbackHint{
		Round: 1, RuntimeEntityID: 90, EntitySource: utilityledger.SourceProjectileEntity,
		Type: utilityledger.TypeFlashbang, ActorID: 11, Tick: 95, TickRate: 64,
		Position: utilityledger.Vector{X: 10, Y: 20, Z: 30}, PositionStatus: utilityledger.AvailabilityObserved,
	}, utilityledger.SourceFlashExplode)
	utilities.EndRound(1)
	utilities.BeginRound(2)
	utilities.EndRound(2)
	combatTracker := combatledger.NewTracker()
	combatTracker.RecordPlayerHurt(combatledger.HurtInput{
		Round: 1, Tick: 100, Actor: canonicalCombatPlayerRef(11, "A", "CT"),
		Target: canonicalCombatPlayerRef(33, "C", "T"), Weapon: canonicalCombatWeapon("ak47"),
		HealthDamage: 100, HealthDamageTaken: 100, HealthAfter: 0, Hitgroup: "head",
	})
	combatTracker.RecordKill(combatledger.KillInput{
		Round: 1, Tick: 100, Actor: canonicalCombatPlayerRef(11, "A", "CT"),
		Target: canonicalCombatPlayerRef(33, "C", "T"), Weapon: canonicalCombatWeapon("ak47"),
	})
	combatTracker.EndRound(1, 120)
	phaseTimeRemaining := 90.0
	return &models.DemoContext{
		MatchData: &models.MatchData{
			MapName: "de_nuke", CTScore: 1, TScore: 1, Winner: "CT",
			Players: map[uint64]*models.PlayerData{
				11: {SteamID: 11, Name: "A"}, 22: {SteamID: 22, Name: "B"},
				33: {SteamID: 33, Name: "C"}, 44: {SteamID: 44, Name: "D"},
			},
			Rounds: []models.RoundData{
				{Round: 1, Winner: "T", Reason: "8", CTScore: 1, TScore: 0},
				{Round: 2, Winner: "T", Reason: "8", CTScore: 1, TScore: 1},
			},
		},
		CurrentRound: 2,
		Objectives:   objectives,
		Utilities:    utilities,
		Combat:       combatTracker,
		AI_EconomyRounds: []models.AI_EconomyRound{
			{Round: 1, Players: []models.AI_EconomyPlayer{
				{SteamID: 11, Team: "CT", Purchases: []models.AI_WeaponItem{{Weapon: "AK-47", Price: 2700}}}, {SteamID: 22, Team: "CT"},
				{SteamID: 33, Team: "T"}, {SteamID: 44, Team: "T"},
			}},
			{Round: 2, Players: []models.AI_EconomyPlayer{
				{SteamID: 11, Team: "T"}, {SteamID: 22, Team: "T"},
				{SteamID: 33, Team: "CT"}, {SteamID: 44, Team: "CT"},
			}},
		},
		AI_CombatEvents: []models.RawCombatEvent{{
			Round: 1, Tick: 100, IsKill: true, AttackerSteamID: 11, AttackerName: "A", AttackerTeam: "CT",
			VictimSteamID: 33, VictimName: "C", VictimTeam: "T", Weapon: "ak47", Damage: 100,
		}},
		AI_Duels: []models.AI_Duel{{
			Round: 1, TickStart: 100, TickEnd: 100, Type: "duel", Outcome: "kill",
			Attacker: func() models.AI_DuelParticipant {
				participant := canonicalValidDuelParticipant()
				participant.SteamID = 11
				participant.MapArea = "Lobby"
				return participant
			}(),
			Victims: func() []models.AI_DuelParticipant {
				participant := canonicalUnavailableDuelParticipant()
				participant.SteamID = 33
				return []models.AI_DuelParticipant{participant}
			}(),
			Context: models.AI_DuelContext{IsTrade: true},
		}},
		AI_GrenadeEvents: []models.AI_GrenadeEvent{{
			Round: 1, Type: "Flashbang", ThrowerSteamID: 11, Thrower: "A", ThrowerSide: "CT",
			TickThrow: 90, TickExplode: 95, Extinguished: true,
		}},
		AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{{
			Round: 1, Event: models.AI_TrackingEvent{
				Tick: 90, PlayerSteamID: 11, Team: "CT", IsAlive: true,
				VelocityVector: models.AI_Vector{X: 64}, VelocityAvailable: true,
				VelocitySource: "position_delta", VelocityMeasurementTicks: 1,
				ActiveWeapon: stringPointer("AK-47"), ActiveWeaponStatus: "observed",
				ObjectivePhase: "preplant", PhaseTimeRemaining: &phaseTimeRemaining,
				RoundClockRemaining: &phaseTimeRemaining, RoundTimeRemaining: phaseTimeRemaining,
			},
		}},
		AI_PlayersSummary: []models.AI_PlayerStats{
			{SteamID: "11", Name: "A"}, {SteamID: "22", Name: "B"},
			{SteamID: "33", Name: "C"}, {SteamID: "44", Name: "D"},
		},
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{
			{Round: 1, StartTick: 80, EndTick: 120, Winner: "CT", Events: []models.ReplayEvent{{Tick: 110, Type: "bomb_plant", PlayerID: 11, Site: "A"}}},
			{Round: 2, StartTick: 130, EndTick: 180, Winner: "CT"},
		}},
	}
}

func canonicalTestUtilityLaunch(tick int) utilityledger.ThrowSnapshot {
	return utilityledger.ThrowSnapshot{
		Tick: utilityledger.TickObservation{Tick: tick, Status: utilityledger.AvailabilityObserved, Source: utilityledger.SourceProjectileThrow},
		Position: utilityledger.VectorObservation{
			Value: utilityledger.Vector{X: 1, Y: 2, Z: 3}, Status: utilityledger.AvailabilityObserved,
			Source: utilityledger.SourceProjectilePosition,
		},
		View: utilityledger.ViewObservation{
			Vector: utilityledger.Vector{X: 1}, Status: utilityledger.AvailabilityObserved,
			Source: utilityledger.SourcePlayerView,
		},
		ThrowerVelocity: utilityledger.VelocityObservation{
			Vector: utilityledger.Vector{X: 10}, HorizontalSpeed: 10, ObservedTick: tick,
			Status: utilityledger.AvailabilityObserved, Source: utilityledger.SourceVelocityNative,
		},
		ProjectileInitialVelocity: utilityledger.VelocityObservation{
			Vector: utilityledger.Vector{X: 100}, HorizontalSpeed: 100, ObservedTick: tick,
			Status: utilityledger.AvailabilityObserved, Source: utilityledger.SourceProjectileVelocity,
		},
		Stance: utilityledger.StanceObservation{
			Value: utilityledger.StanceStanding, Status: utilityledger.AvailabilityObserved, Source: utilityledger.SourcePlayerState,
		},
		Area: utilityledger.StringObservation{
			Value: "Lobby", Status: utilityledger.AvailabilityObserved, Source: utilityledger.SourcePlayerLastPlace,
		},
	}
}

func readCanonicalJSON[T any](t *testing.T, path string) T {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}

func readCanonicalJSONLine[T any](t *testing.T, path string) T {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("read first JSONL row from %s: %v", path, scanner.Err())
	}
	var value T
	if err := json.Unmarshal(scanner.Bytes(), &value); err != nil {
		t.Fatalf("decode first row from %s: %v", path, err)
	}
	return value
}
