package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cs2-demo-service/models"
	combatledger "cs2-demo-service/pkg/combat"
	"cs2-demo-service/pkg/objective"
	utilityledger "cs2-demo-service/pkg/utility"
)

func TestBuildQualityReportDetectsRoundAndEquipmentMismatches(t *testing.T) {
	ctx := &models.DemoContext{
		CurrentRound:   2,
		ParseCompleted: true,
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{
			{Round: 1, StartTick: 10, EndTick: 20},
		}},
		AI_EconomyRounds: []models.AI_EconomyRound{{
			Round: 1,
			Players: []models.AI_EconomyPlayer{{
				SteamID: 1, FinalEquipmentValue: 1000, FinalEquipmentValueCalculated: 900,
			}},
		}},
	}

	prepareObjectiveQualityBaseline(ctx)
	report := buildQualityReport(ctx)
	if report.Status != "fail" || report.UsableForTraining {
		t.Fatalf("unexpected quality result: %+v", report)
	}
	if report.SchemaVersion != block6QualitySchema || len(report.Checks) == 0 {
		t.Fatalf("quality checks were not exported: %+v", report)
	}
	if qualityCheckStatus(report, "visibility_model") != "pass" || report.VisibilityDistanceLimit != "none" {
		t.Fatalf("visibility contract was not exported: %+v", report)
	}
	for _, name := range []string{"replay_round_count", "economy_round_count", "replay_round_sequence", "economy_round_sequence", "tracking_round_sequence"} {
		if qualityCheckStatus(report, name) != "fail" {
			t.Fatalf("expected %s to fail: %+v", name, report.Checks)
		}
	}
}

func TestBuildQualityReportRejectsRoundZeroAndDuplicateReplayRound(t *testing.T) {
	ctx := &models.DemoContext{
		CurrentRound:   2,
		ParseCompleted: true,
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{
			{Round: 1},
			{Round: 1},
		}},
		AI_EconomyRounds: []models.AI_EconomyRound{{Round: 1}, {Round: 2}},
		AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{
			validQualityTrackingEvent(1, 10, 0),
			validQualityTrackingEvent(2, 20, 0),
		},
		AI_Duels:         []models.AI_Duel{{Round: 0}},
		AI_GrenadeEvents: []models.AI_GrenadeEvent{{Round: 0}},
	}

	prepareObjectiveQualityBaseline(ctx)
	report := buildQualityReport(ctx)
	if report.ReplayRoundSequenceErrors != 2 || report.CombatRoundSequenceErrors != 1 || report.GrenadeRoundSequenceErrors != 1 {
		t.Fatalf("round sequence errors were not detected: %+v", report)
	}
	for _, name := range []string{"replay_round_sequence", "combat_round_sequence", "grenade_round_sequence"} {
		if qualityCheckStatus(report, name) != "fail" {
			t.Fatalf("expected %s to fail: %+v", name, report.Checks)
		}
	}
}

func TestBuildQualityReportRejectsInvalidDerivedValues(t *testing.T) {
	ctx := &models.DemoContext{
		CurrentRound:   2,
		ParseCompleted: true,
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{
			{Round: 1},
			{Round: 2},
		}},
		AI_EconomyRounds: []models.AI_EconomyRound{
			{
				Round: 1,
				Players: []models.AI_EconomyPlayer{{
					EquipmentValueEnd: 600,
					EndEquipment:      []models.AI_WeaponItem{{Weapon: "AK-47", Price: 500}},
				}},
			},
			{Round: 2},
		},
		AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{
			{Round: 1},
			{Round: 2},
		},
		AI_Duels: []models.AI_Duel{{
			Round: 1,
			Attacker: models.AI_DuelParticipant{
				InitialCrosshairError: 10,
				PitchError:            1,
				YawError:              1,
			},
			Victims: []models.AI_DuelParticipant{{
				InitialCrosshairError: 20,
				PitchError:            2,
				YawError:              2,
			}},
		}},
		AI_PlayersSummary: []models.AI_PlayerStats{{
			AccuracyOverall: 101,
			WeaponStats: map[string]models.AI_WeaponStat{
				"AK-47": {Accuracy: -1},
			},
		}},
	}

	prepareObjectiveQualityBaseline(ctx)
	report := buildQualityReport(ctx)
	if report.AccuracyRangeMismatches != 2 || report.CrosshairVectorMismatches != 2 || report.EconomyEndMismatches != 1 {
		t.Fatalf("derived-value mismatches were not detected: %+v", report)
	}
	for _, name := range []string{"accuracy_ranges", "crosshair_vector_consistency", "economy_round_end_reconciliation"} {
		if qualityCheckStatus(report, name) != "fail" {
			t.Fatalf("expected %s to fail: %+v", name, report.Checks)
		}
	}
}

func TestBuildQualityReportAllowsSparseCombatAndGrenadeRounds(t *testing.T) {
	ctx := &models.DemoContext{
		CurrentRound:   2,
		ParseCompleted: true,
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{
			{Round: 1},
			{Round: 2},
		}},
		AI_EconomyRounds: []models.AI_EconomyRound{{Round: 1}, {Round: 2}},
		AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{
			validQualityTrackingEvent(1, 10, 0),
			validQualityTrackingEvent(2, 20, 0),
		},
		AI_Duels:         []models.AI_Duel{{Round: 2}},
		AI_GrenadeEvents: nil,
	}

	prepareObjectiveQualityBaseline(ctx)
	report := buildQualityReport(ctx)
	if report.Status != "pass" || !report.UsableForTraining {
		t.Fatalf("sparse event rounds should be valid: %+v", report)
	}
}

func TestBuildQualityReportWarnsOnNativeEndValueMismatch(t *testing.T) {
	ctx := &models.DemoContext{
		CurrentRound:   1,
		ParseCompleted: true,
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{
			{Round: 1},
		}},
		AI_EconomyRounds: []models.AI_EconomyRound{{
			Round: 1,
			Players: []models.AI_EconomyPlayer{{
				EquipmentValueEnd:           500,
				EquipmentValueEndNative:     600,
				EquipmentValueEndCalculated: 500,
				EndEquipment:                []models.AI_WeaponItem{{Weapon: "AK-47", Price: 500}},
			}},
		}},
		AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{validQualityTrackingEvent(1, 10, 0)},
	}

	prepareObjectiveQualityBaseline(ctx)
	report := buildQualityReport(ctx)
	if report.Status != "warning" || !report.UsableForTraining || report.EconomyEndNativeMismatches != 1 {
		t.Fatalf("native end-value drift should be a usable warning: %+v", report)
	}
	if qualityCheckStatus(report, "economy_round_end_reconciliation") != "pass" ||
		qualityCheckStatus(report, "economy_round_end_native_reconciliation") != "warning" {
		t.Fatalf("unexpected round-end checks: %+v", report.Checks)
	}
}

func TestBuildQualityReportReconcilesAuthoritativeUtilityDamageWithNativeScoreboard(t *testing.T) {
	ctx := &models.DemoContext{
		CurrentRound:     1,
		ParseCompleted:   true,
		ReplayData:       &models.ReplayData{Rounds: []models.ReplayRound{{Round: 1}}},
		AI_EconomyRounds: []models.AI_EconomyRound{{Round: 1}},
		AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{
			validQualityTrackingEvent(1, 10, 0),
		},
		AI_PlayersSummary: []models.AI_PlayerStats{{
			SteamID: "1", UtilityDamage: 42,
			NativeScoreboard: models.AI_NativePlayerStats{UtilityDamage: 41},
		}},
	}

	prepareObjectiveQualityBaseline(ctx)
	report := buildQualityReport(ctx)
	if report.NativeScoreboardMismatches != 1 || qualityCheckStatus(report, "native_scoreboard_reconciliation") != "warning" {
		t.Fatalf("native utility damage drift was not reported: %+v", report)
	}
}

func TestBuildQualityReportRejectsMovingSeriesWithZeroVelocity(t *testing.T) {
	events := []models.AI_TrackingEventWithRound{
		validQualityTrackingEvent(1, 10, 0),
		validQualityTrackingEvent(1, 20, 10),
		validQualityTrackingEvent(1, 30, 20),
	}
	for index := range events {
		events[index].Event.VelocityVector = models.AI_Vector{}
	}
	ctx := &models.DemoContext{
		CurrentRound:               1,
		ParseCompleted:             true,
		ReplayData:                 &models.ReplayData{Rounds: []models.ReplayRound{{Round: 1}}},
		AI_EconomyRounds:           []models.AI_EconomyRound{{Round: 1}},
		AI_TrackingEventsWithRound: events,
	}

	prepareObjectiveQualityBaseline(ctx)
	report := buildQualityReport(ctx)
	if report.UsableForTraining || qualityCheckStatus(report, "player_state_motion_signal") != "fail" {
		t.Fatalf("zero-velocity moving series was accepted: %+v", report)
	}
}

func validQualityTrackingEvent(round, tick int, x float64) models.AI_TrackingEventWithRound {
	weapon := "AK-47"
	phaseTime := 100.0
	roundTime := 100.0
	return models.AI_TrackingEventWithRound{
		Round: round,
		Event: models.AI_TrackingEvent{
			Tick: tick, PlayerSteamID: 1, Position: models.AI_Vector{X: x}, IsAlive: true,
			VelocityVector: models.AI_Vector{X: 10}, VelocityAvailable: true,
			VelocitySource: "position_delta", VelocityMeasurementTicks: 1,
			ActiveWeapon: &weapon, ActiveWeaponStatus: models.ActiveWeaponStatusObserved,
			RoundTimeRemaining: 100, ObjectivePhase: "preplant",
			PhaseTimeRemaining: &phaseTime, RoundClockRemaining: &roundTime,
		},
	}
}

func prepareObjectiveQualityBaseline(ctx *models.DemoContext) {
	if ctx.Combat == nil {
		ctx.Combat = combatledger.NewTracker()
	}
	if ctx.Utilities == nil {
		ctx.Utilities = utilityledger.NewTracker()
	}
	if ctx.Objectives == nil {
		ctx.Objectives = objective.NewTracker()
		for round := 1; round <= ctx.CurrentRound; round++ {
			ctx.Objectives.BeginRound(round, 0)
			ctx.Objectives.EndRound(round, 0)
		}
	}
	if ctx.MatchData == nil {
		ctx.MatchData = &models.MatchData{}
		for round := 1; round <= ctx.CurrentRound; round++ {
			ctx.MatchData.Rounds = append(ctx.MatchData.Rounds, models.RoundData{Round: round, Reason: "9"})
		}
	}
}

func TestObjectiveQualityPassesCoherentLifecycle(t *testing.T) {
	tracker := objective.NewTracker()
	terrorist := objective.Actor{SteamID: 1, Name: "T", Side: "T"}
	counterTerrorist := objective.Actor{SteamID: 2, Name: "CT", Side: "CT"}
	tracker.BeginRound(1, 10)
	tracker.NativeSnapshot(objective.NativeObservation{
		Round: 1, Tick: 10, Carrier: terrorist,
		Position: objective.ObservedPosition(10, 20, 0, objective.SourceDemoinfocsNativeSnapshot),
	})
	tracker.PlantStart(objectiveInput(1, 20, terrorist, "A"))
	tracker.Plant(objectiveInput(1, 30, terrorist, "A"))
	hasKit := true
	defuseInput := objectiveInput(1, 40, counterTerrorist, "A")
	defuseInput.HasKit = &hasKit
	tracker.DefuseStart(defuseInput)
	defuseInput.Tick = 50
	tracker.Defuse(defuseInput)
	tracker.EndRound(1, 60)

	carriedState := validQualityTrackingEvent(1, 12, 0)
	carriedState.Event.PlayerSteamID = terrorist.SteamID
	carriedState.Event.Team = "T"
	carriedState.Event.HasC4 = true
	plantedState := validQualityTrackingEvent(1, 35, 0)
	plantedState.Event.PlayerSteamID = terrorist.SteamID
	plantedState.Event.Team = "T"
	setQualityTrackingPhase(&plantedState.Event, "planted", 35)

	ctx := &models.DemoContext{
		CurrentRound: 1, ParseCompleted: true, Objectives: tracker,
		MatchData: &models.MatchData{MapName: "de_test", Rounds: []models.RoundData{{
			Round: 1, Reason: "7", BombPlanted: true, BombSite: "A", BombTick: 30,
		}}},
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{{
			Round: 1, StartTick: 10, EndTick: 60,
			Events: []models.ReplayEvent{
				{Tick: 30, Type: "bomb_plant", PlayerID: 1, Site: "A", X: 30, Y: 2},
				{Tick: 50, Type: "bomb_defuse", PlayerID: 2, Site: "A", X: 40, Y: 2},
			},
			Frames: []models.ReplayFrame{
				qualityReplayFrame(12, objective.StateCarried, objective.PhasePreplant, uint64Pointer(1), true),
				qualityReplayFrame(35, objective.StatePlanted, objective.PhasePlanted, nil, false),
				qualityReplayFrame(50, objective.StateDefused, objective.PhaseResolved, nil, false),
			},
		}}},
		AI_EconomyRounds:           []models.AI_EconomyRound{{Round: 1}},
		AI_TrackingEventsWithRound: []models.AI_TrackingEventWithRound{carriedState, plantedState},
	}

	report := buildQualityReport(ctx)
	for _, name := range []string{
		"objective_event_contract", "objective_round_reconciliation", "objective_terminal_reconciliation",
		"objective_lifecycle", "objective_replay_projection", "objective_carrier_consistency",
	} {
		if qualityCheckStatus(report, name) != "pass" {
			t.Fatalf("expected %s to pass: %+v", name, report.Checks)
		}
	}
	if report.ObjectiveEvents != 5 || report.ObjectivePlantStarts != 1 || report.ObjectivePlants != 1 ||
		report.ObjectiveDefuseStarts != 1 || report.ObjectiveDefuses != 1 || report.ObjectiveAttempts != 2 {
		t.Fatalf("objective counters are incomplete: %+v", report)
	}
}

func TestObjectiveQualityTreatsMissingAttemptCallbacksAsCoverage(t *testing.T) {
	tracker := objective.NewTracker()
	terrorist := objective.Actor{SteamID: 1, Side: "T"}
	tracker.BeginRound(1, 10)
	tracker.NativeSnapshot(objective.NativeObservation{
		Round: 1, Tick: 10, Carrier: terrorist,
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsNativeSnapshot),
	})
	tracker.Plant(objectiveInput(1, 20, terrorist, "A"))
	tracker.EndRound(1, 30)
	tracker.BeginRound(2, 40)
	tracker.NativeSnapshot(objective.NativeObservation{
		Round: 2, Tick: 40, Carrier: terrorist,
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsNativeSnapshot),
	})
	tracker.PlantStart(objectiveInput(2, 50, terrorist, "B"))
	tracker.EndRound(2, 60)
	ctx := &models.DemoContext{
		CurrentRound: 2, ParseCompleted: true, Objectives: tracker,
		MatchData: &models.MatchData{Rounds: []models.RoundData{
			{Round: 1, Reason: "9", BombPlanted: true, BombSite: "A", BombTick: 20},
			{Round: 2, Reason: "9"},
		}},
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{
			{Round: 1, Events: []models.ReplayEvent{{Tick: 20, Type: "bomb_plant", PlayerID: 1, Site: "A", X: 20, Y: 2}}},
			{Round: 2},
		}},
	}

	stats := assessObjectiveQuality(ctx)
	if stats.lifecycleViolations != 0 || stats.attemptsMissingStart != 1 || stats.attemptsUnclosed != 1 {
		t.Fatalf("callback coverage was classified incorrectly: %+v", stats)
	}
	if stats.lifecycleCheck().Status != "warning" || stats.eventContractCheck().Status != "pass" || stats.hasHardFailure() {
		t.Fatalf("sparse callbacks must remain usable warnings: %+v", stats.checks())
	}
}

func TestObjectiveQualityRejectsContradictoryReconciliation(t *testing.T) {
	tracker := objective.NewTracker()
	terrorist := objective.Actor{SteamID: 1, Side: "T"}
	tracker.BeginRound(1, 10)
	tracker.NativeSnapshot(objective.NativeObservation{
		Round: 1, Tick: 10, Carrier: terrorist,
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsNativeSnapshot),
	})
	tracker.PlantStart(objectiveInput(1, 20, terrorist, "A"))
	tracker.Plant(objectiveInput(1, 30, terrorist, "A"))
	tracker.Drop(objectiveInput(1, 35, terrorist, "A"))
	tracker.Explode(objectiveInput(1, 40, objective.Actor{}, "A"))
	tracker.EndRound(1, 50)
	ctx := &models.DemoContext{
		CurrentRound: 1, ParseCompleted: true, Objectives: tracker,
		MatchData: &models.MatchData{Rounds: []models.RoundData{{Round: 1, Reason: "7"}}},
	}

	stats := assessObjectiveQuality(ctx)
	if stats.contractViolations != 0 || stats.roundMismatches == 0 || stats.terminalMismatches == 0 || !stats.hasHardFailure() {
		t.Fatalf("objective contradictions were not rejected: %+v", stats)
	}
	if stats.eventContractCheck().Status != "pass" || stats.roundReconciliationCheck().Status != "fail" ||
		stats.terminalReconciliationCheck().Status != "fail" {
		t.Fatalf("objective gates did not fail: %+v", stats.checks())
	}
}

func TestObjectiveQualityCountsMissingActorAndPositionAsCoverage(t *testing.T) {
	tracker := objective.NewTracker()
	tracker.BeginRound(1, 10)
	tracker.Drop(objective.EventInput{
		Round: 1, Tick: 20, Source: objective.SourceDemoinfocsEvent,
		Position: objective.UnavailablePosition(objective.SourceDemoinfocsEvent),
	})
	tracker.EndRound(1, 30)
	ctx := &models.DemoContext{
		CurrentRound: 1, ParseCompleted: true, Objectives: tracker,
		MatchData: &models.MatchData{Rounds: []models.RoundData{{Round: 1, Reason: "9"}}},
	}

	stats := assessObjectiveQuality(ctx)
	if stats.missingActors != 1 || stats.missingPositions != 1 || stats.contractViolations != 0 {
		t.Fatalf("missing observations were not measured as coverage: %+v", stats)
	}
	if stats.eventContractCheck().Status != "warning" || stats.hasHardFailure() {
		t.Fatalf("missing observations must be a usable warning: %+v", stats.checks())
	}
}

func TestObjectiveQualityRejectsTrackingAndReplayCarrierContradictions(t *testing.T) {
	tracker := objective.NewTracker()
	terrorist := objective.Actor{SteamID: 1, Side: "T"}
	tracker.BeginRound(1, 10)
	tracker.NativeSnapshot(objective.NativeObservation{
		Round: 1, Tick: 10, Carrier: terrorist,
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsNativeSnapshot),
	})
	tracker.PlantStart(objectiveInput(1, 20, terrorist, "A"))
	tracker.Plant(objectiveInput(1, 30, terrorist, "A"))
	tracker.EndRound(1, 40)
	tracking := []models.AI_TrackingEventWithRound{
		{Round: 1, Event: models.AI_TrackingEvent{Tick: 10, PlayerSteamID: 1, IsAlive: false, HasC4: true}},
		{Round: 1, Event: models.AI_TrackingEvent{Tick: 10, PlayerSteamID: 2, IsAlive: true, HasC4: true}},
		{Round: 1, Event: models.AI_TrackingEvent{Tick: 35, PlayerSteamID: 2, IsAlive: true, HasC4: true}},
	}
	ctx := &models.DemoContext{
		CurrentRound: 1, ParseCompleted: true, Objectives: tracker,
		MatchData: &models.MatchData{Rounds: []models.RoundData{{
			Round: 1, Reason: "9", BombPlanted: true, BombSite: "A", BombTick: 30,
		}}},
		AI_TrackingEventsWithRound: tracking,
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{{Round: 1, Frames: []models.ReplayFrame{
			{
				Tick: 10, Bomb: &models.ReplayBombState{State: objective.StateCarried, CarrierID: uint64Pointer(1)},
				Players: []models.ReplayPlayerState{{SteamID: 1, Alive: false, HasC4: true}, {SteamID: 2, Alive: true, HasC4: true}},
			},
			{
				Tick: 35, Bomb: &models.ReplayBombState{State: objective.StatePlanted, CarrierID: uint64Pointer(1)},
				Players: []models.ReplayPlayerState{{SteamID: 2, Alive: true, HasC4: true}},
			},
		}}}},
	}

	stats := assessObjectiveQuality(ctx)
	if stats.trackingCarrierMismatches != 2 || stats.replayCarrierMismatches != 2 || !stats.hasHardFailure() {
		t.Fatalf("carrier contradictions were not detected: %+v", stats)
	}
	if stats.carrierConsistencyCheck().Status != "fail" {
		t.Fatalf("carrier quality gate did not fail: %+v", stats.carrierConsistencyCheck())
	}
}

func TestObjectiveReplayProjectionReconcilesEveryMarkerField(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.ReplayData)
	}{
		{name: "round", mutate: func(replay *models.ReplayData) { replay.Rounds[0].Round = 2 }},
		{name: "tick", mutate: func(replay *models.ReplayData) { replay.Rounds[0].Events[0].Tick++ }},
		{name: "type", mutate: func(replay *models.ReplayData) { replay.Rounds[0].Events[0].Type = "bomb_defuse" }},
		{name: "actor", mutate: func(replay *models.ReplayData) { replay.Rounds[0].Events[0].PlayerID = 2 }},
		{name: "site", mutate: func(replay *models.ReplayData) { replay.Rounds[0].Events[0].Site = "B" }},
		{name: "position", mutate: func(replay *models.ReplayData) { replay.Rounds[0].Events[0].X += 2 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := objectiveReplayProjectionFixture()
			test.mutate(ctx.ReplayData)
			stats := assessObjectiveQuality(ctx)
			if stats.replayEventMismatches == 0 || stats.replayProjectionCheck().Status != "fail" || !stats.hasHardFailure() {
				t.Fatalf("%s mismatch was accepted: %+v", test.name, stats)
			}
		})
	}
}

func TestObjectiveReplayProjectionAllowsPositionTolerance(t *testing.T) {
	ctx := objectiveReplayProjectionFixture()
	ctx.ReplayData.Rounds[0].Events[0].X += 0.5
	ctx.ReplayData.Rounds[0].Events[0].Y += 0.5
	stats := assessObjectiveQuality(ctx)
	if stats.replayEventMismatches != 0 || stats.replayProjectionCheck().Status != "pass" || stats.hasHardFailure() {
		t.Fatalf("position tolerance rejected a coherent replay marker: %+v", stats)
	}
}

func TestObjectiveReplayProjectionAcceptsExplosion(t *testing.T) {
	tracker := objective.NewTracker()
	terrorist := objective.Actor{SteamID: 1, Side: "T"}
	tracker.BeginRound(1, 10)
	tracker.PlantStart(objectiveInput(1, 20, terrorist, "A"))
	tracker.Plant(objectiveInput(1, 30, terrorist, "A"))
	explode := objectiveInput(1, 50, terrorist, "A")
	explode.Position = objective.ObservedPosition(30, 2, 3, objective.SourceDemoinfocsEvent)
	tracker.Explode(explode)
	tracker.EndRound(1, 60)
	ctx := &models.DemoContext{
		CurrentRound: 1, ParseCompleted: true, Objectives: tracker,
		MatchData: &models.MatchData{Rounds: []models.RoundData{{
			Round: 1, Reason: "1", BombPlanted: true, BombSite: "A", BombTick: 30,
		}}},
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{{
			Round: 1,
			Events: []models.ReplayEvent{
				{Tick: 30, Type: "bomb_plant", PlayerID: 1, Site: "A", X: 30, Y: 2},
				{Tick: 50, Type: "bomb_explode", PlayerID: 1, Site: "A", X: 30, Y: 2},
			},
		}}},
	}
	stats := assessObjectiveQuality(ctx)
	if stats.replayEventMismatches != 0 || stats.replayProjectionCheck().Status != "pass" || stats.hasHardFailure() {
		t.Fatalf("coherent explosion projection was rejected: %+v", stats)
	}
}

func TestObjectiveQualityRejectsExplosionPositionDrift(t *testing.T) {
	tracker := objective.NewTracker()
	terrorist := objective.Actor{SteamID: 1, Side: "T"}
	tracker.BeginRound(1, 10)
	tracker.PlantStart(objectiveInput(1, 20, terrorist, "A"))
	tracker.Plant(objectiveInput(1, 30, terrorist, "A"))
	explode := objectiveInput(1, 50, terrorist, "A")
	explode.Position = objective.ObservedPosition(100, 2, 3, objective.SourceDemoinfocsEvent)
	tracker.Explode(explode)
	tracker.EndRound(1, 60)
	ctx := &models.DemoContext{
		CurrentRound: 1, ParseCompleted: true, Objectives: tracker,
		MatchData: &models.MatchData{Rounds: []models.RoundData{{
			Round: 1, Reason: "1", BombPlanted: true, BombSite: "A", BombTick: 30,
		}}},
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{{
			Round: 1,
			Events: []models.ReplayEvent{
				{Tick: 30, Type: "bomb_plant", PlayerID: 1, Site: "A", X: 30, Y: 2},
				{Tick: 50, Type: "bomb_explode", PlayerID: 1, Site: "A", X: 100, Y: 2},
			},
		}}},
	}
	stats := assessObjectiveQuality(ctx)
	if stats.contractViolations != 1 || stats.eventContractCheck().Status != "fail" || !stats.hasHardFailure() {
		t.Fatalf("explosion position drift was accepted: %+v", stats)
	}
	if !strings.Contains(strings.Join(stats.failureDetails, "\n"), "explosion position drifted") {
		t.Fatalf("explosion drift detail was not recorded: %+v", stats.failureDetails)
	}
}

func TestObjectiveReplayProjectionRejectsPostResolutionMarker(t *testing.T) {
	ctx := objectiveReplayProjectionFixture()
	ctx.ReplayData.Rounds[0].Events = append(ctx.ReplayData.Rounds[0].Events, models.ReplayEvent{
		Tick: 51, Type: "bomb_plant", PlayerID: 1, Site: "A", X: 30, Y: 2,
	})
	stats := assessObjectiveQuality(ctx)
	if stats.replayEventMismatches != 1 || stats.replayProjectionCheck().Status != "fail" || !stats.hasHardFailure() {
		t.Fatalf("post-resolution replay marker was accepted: %+v", stats)
	}
	if !strings.Contains(strings.Join(stats.failureDetails, "\n"), "after objective resolution") {
		t.Fatalf("post-resolution failure detail was not recorded: %+v", stats.failureDetails)
	}
	report := buildQualityReport(ctx)
	if report.ObjectiveReplayEventMismatches != 1 || qualityCheckStatus(report, "objective_replay_projection") != "fail" ||
		!report.hasHardObjectiveFailure() {
		t.Fatalf("replay projection mismatch was not exported as a hard failure: %+v", report)
	}
}

func objectiveReplayProjectionFixture() *models.DemoContext {
	tracker := objective.NewTracker()
	terrorist := objective.Actor{SteamID: 1, Side: "T"}
	counterTerrorist := objective.Actor{SteamID: 2, Side: "CT"}
	tracker.BeginRound(1, 10)
	tracker.NativeSnapshot(objective.NativeObservation{
		Round: 1, Tick: 10, Carrier: terrorist,
		Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsNativeSnapshot),
	})
	tracker.PlantStart(objectiveInput(1, 20, terrorist, "A"))
	tracker.Plant(objectiveInput(1, 30, terrorist, "A"))
	hasKit := true
	defuseStart := objectiveInput(1, 40, counterTerrorist, "A")
	defuseStart.HasKit = &hasKit
	tracker.DefuseStart(defuseStart)
	defuse := objectiveInput(1, 50, counterTerrorist, "A")
	tracker.Defuse(defuse)
	tracker.EndRound(1, 60)
	return &models.DemoContext{
		CurrentRound: 1, ParseCompleted: true, Objectives: tracker,
		MatchData: &models.MatchData{Rounds: []models.RoundData{{
			Round: 1, Reason: "7", BombPlanted: true, BombSite: "A", BombTick: 30,
		}}},
		ReplayData: &models.ReplayData{Rounds: []models.ReplayRound{{
			Round: 1,
			Events: []models.ReplayEvent{
				{Tick: 30, Type: "bomb_plant", PlayerID: 1, Site: "A", X: 30, Y: 2},
				{Tick: 50, Type: "bomb_defuse", PlayerID: 2, Site: "A", X: 50, Y: 2},
			},
		}}},
	}
}

func TestTrackingStateContractValidatesObjectivePhaseClocksAndRoles(t *testing.T) {
	valid := validQualityTrackingEvent(1, 10, 0).Event
	if trackingStateContractInvalid(valid) {
		t.Fatalf("valid preplant state was rejected: %+v", valid)
	}
	postplant := valid
	setQualityTrackingPhase(&postplant, "planted", 35)
	if trackingStateContractInvalid(postplant) {
		t.Fatalf("valid planted state was rejected: %+v", postplant)
	}

	invalid := []models.AI_TrackingEvent{
		validQualityTrackingEvent(1, 10, 0).Event,
		validQualityTrackingEvent(1, 10, 0).Event,
		validQualityTrackingEvent(1, 10, 0).Event,
		validQualityTrackingEvent(1, 10, 0).Event,
	}
	invalid[0].PhaseTimeRemaining = nil
	invalid[1].BombTimeRemaining = float64Pointer(10)
	invalid[2].ObjectivePhase = "planting"
	invalid[2].IsPlanting = true
	invalid[3].ObjectivePhase = "resolved"
	invalid[3].RoundClockRemaining = nil
	*invalid[3].PhaseTimeRemaining = 1
	invalid[3].RoundTimeRemaining = 1
	for index, event := range invalid {
		if !trackingStateContractInvalid(event) {
			t.Fatalf("invalid objective state %d was accepted: %+v", index, event)
		}
	}
}

func objectiveInput(round, tick int, actor objective.Actor, site string) objective.EventInput {
	return objective.EventInput{
		Round: round, Tick: tick, Actor: actor, Site: site, Source: objective.SourceDemoinfocsEvent,
		Position: objective.ObservedPosition(float64(tick), 2, 3, objective.SourceDemoinfocsEvent),
	}
}

func setQualityTrackingPhase(event *models.AI_TrackingEvent, phase string, remaining float64) {
	event.ObjectivePhase = phase
	event.RoundTimeRemaining = remaining
	event.PhaseTimeRemaining = float64Pointer(remaining)
	event.RoundClockRemaining = nil
	event.BombTimeRemaining = nil
	if phase == "preplant" || phase == "planting" {
		event.RoundClockRemaining = float64Pointer(remaining)
	} else if phase == "planted" || phase == "defusing" {
		event.BombTimeRemaining = float64Pointer(remaining)
	}
}

func qualityReplayFrame(tick int, state objective.State, phase objective.Phase, carrierID *uint64, hasC4 bool) models.ReplayFrame {
	return models.ReplayFrame{
		Tick: tick, Bomb: &models.ReplayBombState{State: state, ObjectivePhase: phase, CarrierID: carrierID},
		Players: []models.ReplayPlayerState{{SteamID: 1, Alive: true, HasC4: hasC4}},
	}
}

func float64Pointer(value float64) *float64 { return &value }
func uint64Pointer(value uint64) *uint64    { return &value }

func qualityCheckStatus(report qualityReport, name string) string {
	for _, check := range report.Checks {
		if check.Name == name {
			return check.Status
		}
	}
	return ""
}

func TestQualityReportHardObjectiveFailure(t *testing.T) {
	if (qualityReport{}).hasHardObjectiveFailure() {
		t.Fatal("an empty objective quality report must be publishable")
	}
	for _, report := range []qualityReport{
		{ObjectiveContractViolations: 1},
		{ObjectiveRoundMismatches: 1},
		{ObjectiveTerminalMismatches: 1},
		{ObjectiveLifecycleViolations: 1},
		{ObjectiveCarrierMismatches: 1},
	} {
		if !report.hasHardObjectiveFailure() {
			t.Fatalf("objective contradiction was not treated as a hard failure: %+v", report)
		}
	}
}

func TestWriteJSONAndStreamingChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.json")
	payload := map[string][]int{"values": {1, 2, 3}}
	if err := writeJSON(path, payload); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("export is not valid JSON: %q", data)
	}
	firstChecksum, err := checksumFile(path)
	if err != nil {
		t.Fatal(err)
	}
	secondChecksum, err := checksumFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if firstChecksum == "" || firstChecksum != secondChecksum {
		t.Fatalf("checksum is not stable: %q != %q", firstChecksum, secondChecksum)
	}
}
