package parser

import (
	"testing"

	"cs2-demo-service/models"
	combatledger "cs2-demo-service/pkg/combat"
)

func TestCombatContractGateRejectsMissingProvenanceAndInvalidSourceLink(t *testing.T) {
	tracker := combatledger.NewTracker()
	tracker.RecordWeaponFire(combatledger.FireInput{
		Round: 1, Tick: 10, Actor: combatQualityPlayer(1, "T"), Weapon: combatQualityWeapon("AK-47"),
	})
	tracker.EndRound(1, 20)
	events := tracker.Snapshot()
	ctx := &models.DemoContext{MatchData: &models.MatchData{Players: map[uint64]*models.PlayerData{1: nil}}}
	if violations := combatContractViolations(ctx, events, tracker.Diagnostics()); violations != 0 {
		t.Fatalf("coherent fire was rejected with %d violations: %+v", violations, events)
	}

	broken := append([]combatledger.Event(nil), events...)
	broken[0].Source = ""
	broken[0].Weapon.IsUtility = nil
	broken[0].SourceEventIDs = []string{broken[0].LocalID}
	if violations := combatContractViolations(ctx, broken, tracker.Diagnostics()); violations < 3 {
		t.Fatalf("expected source, weapon provenance and self-link failures, got %d", violations)
	}
}

func TestCombatCallbackAccountingGateAcceptsExplicitDiscard(t *testing.T) {
	diagnostics := combatledger.Diagnostics{
		ObservedByType:    map[combatledger.EventType]int{combatledger.EventPlayerHurt: 1},
		RecordedByType:    map[combatledger.EventType]int{combatledger.EventPlayerHurt: 0},
		DiscardedByType:   map[combatledger.EventType]int{combatledger.EventPlayerHurt: 1},
		DiscardedByReason: map[string]int{combatledger.DiscardOutsideRound: 1},
	}
	if violations := combatCallbackAccountingViolations(diagnostics); violations != 0 {
		t.Fatalf("explicit discard should conserve callback accounting, got %d", violations)
	}
	diagnostics.DiscardedByReason = map[string]int{"unexpected": 1}
	if violations := combatCallbackAccountingViolations(diagnostics); violations == 0 {
		t.Fatal("unknown discard reason should fail accounting")
	}
}

func TestCombatContractAllowsExactAtomicLinkWhenFireCallbackIsUnavailable(t *testing.T) {
	tracker := combatledger.NewTracker()
	tracker.RecordBulletDamage(combatledger.BulletDamageInput{
		Round: 1, Tick: 10, Actor: combatQualityPlayer(1, "T"), Target: combatQualityPlayer(2, "CT"),
		Distance: 100, Direction: combatledger.Vector{X: 1},
	})
	tracker.RecordPlayerHurt(combatledger.HurtInput{
		Round: 1, Tick: 10, Actor: combatQualityPlayer(1, "T"), Target: combatQualityPlayer(2, "CT"),
		Weapon: combatQualityWeapon("AK-47"), HealthDamage: 20, HealthDamageTaken: 20,
		HealthAfter: 80, ArmorAfter: 50, Hitgroup: "chest",
	})
	events := tracker.Snapshot()
	ctx := &models.DemoContext{MatchData: &models.MatchData{Players: map[uint64]*models.PlayerData{1: nil, 2: nil}}}
	if events[1].CorrelationStatus != combatledger.CorrelationExact || events[1].ShotID != "" {
		t.Fatalf("expected exact bullet-to-hurt link without fabricated shot: %+v", events[1])
	}
	if violations := combatContractViolations(ctx, events, tracker.Diagnostics()); violations != 0 {
		t.Fatalf("valid exact atomic link without fire was rejected: %d", violations)
	}
}

func TestCombatReplayGateRejectsShotProvenanceDrift(t *testing.T) {
	tracker := combatledger.NewTracker()
	tracker.RecordWeaponFire(combatledger.FireInput{
		Round: 1, Tick: 10, Actor: combatQualityPlayer(1, "T"), Weapon: combatQualityWeapon("AK-47"),
	})
	tracker.EndRound(1, 20)
	fire := tracker.Snapshot()[0]
	shot := models.ReplayShot{
		Tick: fire.Tick, SourceEventID: fire.LocalID, ShotID: fire.ShotID,
		Result: string(fire.ShotResult), ResultStatus: string(fire.ShotResultStatus),
	}
	shot.ShotID = "wrong-shot"
	replay := &models.ReplayData{Rounds: []models.ReplayRound{{
		Round:  1,
		Frames: []models.ReplayFrame{{Tick: 10, Shots: []models.ReplayShot{shot}}},
	}}}
	if mismatches := combatReplayMismatches(replay, []combatledger.Event{fire}); mismatches == 0 {
		t.Fatal("replay shot provenance drift was accepted")
	}
}

func TestCombatReplayGateAcceptsUniqueShotWithoutCarrierFrame(t *testing.T) {
	tracker := combatledger.NewTracker()
	tracker.RecordWeaponFire(combatledger.FireInput{
		Round: 1, Tick: 10, Actor: combatQualityPlayer(1, "T"), Weapon: combatQualityWeapon("AK-47"),
	})
	tracker.EndRound(1, 20)
	fire := tracker.Snapshot()[0]
	replay := &models.ReplayData{Rounds: []models.ReplayRound{{
		Round: 1, EndTick: 20, CombatShots: []models.ReplayShot{replayShotFromCombatEvent(fire)},
	}}}
	if mismatches := combatReplayMismatches(replay, []combatledger.Event{fire}); mismatches != 0 {
		t.Fatalf("frame-independent combat shot was rejected: %d", mismatches)
	}
}

func replayShotFromCombatEvent(event combatledger.Event) models.ReplayShot {
	weapon := ""
	if event.Weapon.Status == combatledger.AvailabilityObserved {
		weapon = event.Weapon.Name
	}
	return models.ReplayShot{
		Tick: event.Tick, SourceEventID: event.LocalID, ShotID: event.ShotID,
		ShooterID: event.Actor.ID, Weapon: weapon,
		Result: string(event.ShotResult), ResultStatus: string(event.ShotResultStatus),
		PositionStatus: string(event.Actor.PositionStatus), PositionSource: event.Actor.PositionSource,
		EndpointStatus: string(combatledger.AvailabilityUnavailable), EndpointSource: combatledger.SourceUnavailable,
		Hit: event.ShotResult == combatledger.ShotHit,
	}
}

func TestCombatStatsGateRejectsMissingLedgerParticipant(t *testing.T) {
	tracker := combatledger.NewTracker()
	tracker.RecordWeaponFire(combatledger.FireInput{
		Round: 1, Tick: 10, Actor: combatQualityPlayer(1, "T"), Weapon: combatQualityWeapon("AK-47"),
	})
	tracker.EndRound(1, 20)
	ctx := &models.DemoContext{}
	statsMismatches, _ := combatStatsMismatches(ctx, tracker.Snapshot())
	if statsMismatches != 1 {
		t.Fatalf("missing projected participant should produce one mismatch, got %d", statsMismatches)
	}
}

func TestQualityReportHardCombatFailure(t *testing.T) {
	if (qualityReport{}).hasHardCombatFailure() {
		t.Fatal("empty combat quality report must be publishable")
	}
	for _, report := range []qualityReport{
		{CombatContractViolations: 1},
		{CombatCallbackAccountingViolations: 1},
		{CombatPlayerStatsMismatches: 1},
		{CombatReplayProjectionMismatches: 1},
		{CombatNativeDeltaMismatches: 1},
		{CombatDeterminismViolations: 1},
	} {
		if !report.hasHardCombatFailure() {
			t.Fatalf("combat contradiction was not treated as a hard failure: %+v", report)
		}
	}
}

func TestCombatStatsGateIgnoresOnlyEmptyAnonymousPlaceholder(t *testing.T) {
	ctx := &models.DemoContext{AI_PlayersSummary: []models.AI_PlayerStats{{}}}
	statsMismatches, nativeMismatches := combatStatsMismatches(ctx, nil)
	if statsMismatches != 0 || nativeMismatches != 0 {
		t.Fatalf("empty anonymous placeholder became a hard mismatch: stats=%d native=%d", statsMismatches, nativeMismatches)
	}
	ctx.AI_PlayersSummary[0].KillsObserved = 1
	statsMismatches, _ = combatStatsMismatches(ctx, nil)
	if statsMismatches != 1 {
		t.Fatalf("anonymous combat data was not rejected: stats=%d", statsMismatches)
	}
}

func combatQualityPlayer(id uint64, side string) combatledger.PlayerRef {
	position := combatledger.Vector{X: float64(id)}
	return combatledger.PlayerRef{
		ID: id, Side: side, Status: combatledger.AvailabilityObserved, Source: combatledger.SourceCallbackPlayer,
		Position: &position, PositionStatus: combatledger.AvailabilityObserved, PositionSource: combatledger.SourceCallbackPosition,
	}
}

func combatQualityWeapon(name string) combatledger.WeaponRef {
	isUtility := false
	return combatledger.WeaponRef{
		Name: name, Status: combatledger.AvailabilityObserved, Source: combatledger.SourceCallbackWeapon, IsUtility: &isUtility,
	}
}
