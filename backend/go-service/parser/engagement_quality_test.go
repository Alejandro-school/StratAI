package parser

import (
	"reflect"
	"testing"

	"cs2-demo-service/models"
	engagementpkg "cs2-demo-service/pkg/engagement"

	"github.com/golang/geo/r3"
)

func TestEngagementQualityPassesCoherentArtifactsAndExportsAllGates(t *testing.T) {
	fixture := newEngagementQualityFixture(t, engagementQualityStates())
	assessment := fixture.assess()
	if assessment.eventContract != 0 || assessment.atomicProvenance != 0 ||
		assessment.participantReconciliation != 0 || assessment.roleConsistency != 0 ||
		assessment.temporalConsistency != 0 || assessment.causalAvailability != 0 ||
		assessment.tradeReconciliation != 0 || assessment.statsReconciliation != 0 ||
		assessment.determinism != 0 {
		t.Fatalf("coherent artifacts failed quality: %+v", assessment)
	}
	wantNames := []string{
		"engagement_event_contract",
		"engagement_atomic_provenance",
		"engagement_participant_reconciliation",
		"engagement_role_consistency",
		"engagement_temporal_consistency",
		"engagement_causal_availability",
		"engagement_trade_reconciliation",
		"engagement_stats_reconciliation",
		"engagement_determinism",
		"engagement_observation_coverage",
	}
	checks := assessment.checks()
	gotNames := make([]string, 0, len(checks))
	for _, check := range checks {
		gotNames = append(gotNames, check.Name)
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("quality gate names = %v, want %v", gotNames, wantNames)
	}
	report := qualityReport{Status: "pass", UsableForTraining: true}
	report.applyEngagementQuality(assessment)
	if report.SchemaVersion != 10 || report.hasHardEngagementFailure() || report.Status != "warning" || !report.UsableForTraining {
		t.Fatalf("coherent engagement report is not publishable: %+v", report)
	}
}

func TestEngagementQualityRejectsContractContradictions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*engagementQualityFixture)
		failed func(engagementQualityAssessment) bool
	}{
		{
			name: "exchange no longer projects the atomic hurt",
			mutate: func(fixture *engagementQualityFixture) {
				fixture.engagements.Engagements[0].Exchanges[0].ActorPlayerID = "steam:4"
			},
			failed: func(got engagementQualityAssessment) bool { return got.eventContract > 0 },
		},
		{
			name: "atomic source closure is incomplete",
			mutate: func(fixture *engagementQualityFixture) {
				fixture.engagements.Engagements[0].Exchanges[0].SourceEventIDs = []string{fixture.engagements.Engagements[0].Exchanges[0].ExchangeID}
			},
			failed: func(got engagementQualityAssessment) bool { return got.atomicProvenance > 0 },
		},
		{
			name: "participant is omitted",
			mutate: func(fixture *engagementQualityFixture) {
				fixture.engagements.Engagements[0].Participants = fixture.engagements.Engagements[0].Participants[:1]
			},
			failed: func(got engagementQualityAssessment) bool { return got.participantReconciliation > 0 },
		},
		{
			name: "first damage is rewritten from the winner",
			mutate: func(fixture *engagementQualityFixture) {
				fixture.engagements.Engagements[0].FirstDamageDealer.PlayerID = engagementQualityString("steam:2")
			},
			failed: func(got engagementQualityAssessment) bool { return got.roleConsistency > 0 },
		},
		{
			name: "duration exceeds atomic bounds",
			mutate: func(fixture *engagementQualityFixture) {
				fixture.engagements.Engagements[0].EndTick += 1
			},
			failed: func(got engagementQualityAssessment) bool { return got.temporalConsistency > 0 },
		},
		{
			name: "future participant state leaks into t0",
			mutate: func(fixture *engagementQualityFixture) {
				fixture.engagements.Engagements[0].CausalContext.ParticipantStates[0].AvailabilityTick = engagementQualityInt(999)
			},
			failed: func(got engagementQualityAssessment) bool { return got.causalAvailability > 0 },
		},
		{
			name: "one response kill completes two rows",
			mutate: func(fixture *engagementQualityFixture) {
				duplicate := fixture.trades.Completions[0]
				duplicate.TradeCompletionID += "-duplicate"
				fixture.trades.Completions = append(fixture.trades.Completions, duplicate)
			},
			failed: func(got engagementQualityAssessment) bool { return got.tradeReconciliation > 0 },
		},
		{
			name: "trade stats drift",
			mutate: func(fixture *engagementQualityFixture) {
				fixture.stats.Players[3].Metrics.TradeKills++
			},
			failed: func(got engagementQualityAssessment) bool { return got.statsReconciliation > 0 },
		},
		{
			name: "artifact differs from deterministic derivation",
			mutate: func(fixture *engagementQualityFixture) {
				fixture.engagements.Engagements[0], fixture.engagements.Engagements[1] = fixture.engagements.Engagements[1], fixture.engagements.Engagements[0]
			},
			failed: func(got engagementQualityAssessment) bool { return got.determinism > 0 },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newEngagementQualityFixture(t, engagementQualityStates())
			test.mutate(&fixture)
			assessment := fixture.assess()
			if !test.failed(assessment) || len(assessment.failureDetails) == 0 {
				t.Fatalf("contradiction was accepted: %+v", assessment)
			}
			report := qualityReport{Status: "pass", UsableForTraining: true}
			report.applyEngagementQuality(assessment)
			if !report.hasHardEngagementFailure() || report.UsableForTraining || report.Status != "fail" {
				t.Fatalf("hard failure was not propagated: %+v", report)
			}
		})
	}
}

func TestEngagementQualityTreatsMissingObservationsAsWarning(t *testing.T) {
	fixture := newEngagementQualityFixture(t, nil)
	assessment := fixture.assess()
	if assessment.observationWarnings == 0 {
		t.Fatal("missing causal/trade observations did not produce a coverage warning")
	}
	if assessment.eventContract != 0 || assessment.atomicProvenance != 0 ||
		assessment.participantReconciliation != 0 || assessment.roleConsistency != 0 ||
		assessment.temporalConsistency != 0 || assessment.causalAvailability != 0 ||
		assessment.tradeReconciliation != 0 || assessment.statsReconciliation != 0 || assessment.determinism != 0 {
		t.Fatalf("explicit unavailable observations became hard failures: %+v", assessment)
	}
	report := qualityReport{Status: "pass", UsableForTraining: true}
	report.applyEngagementQuality(assessment)
	if report.Status != "warning" || !report.UsableForTraining || report.hasHardEngagementFailure() {
		t.Fatalf("coverage-only warning made the export unusable: %+v", report)
	}
}

type engagementQualityFixture struct {
	matchID      string
	tickRate     float64
	rounds       []models.CanonicalRound
	participants []models.CanonicalParticipant
	events       []models.CanonicalCombatEvent
	states       map[int][]models.CanonicalPlayerState
	engagements  models.CanonicalEngagementsExport
	trades       models.CanonicalTradesExport
	stats        models.CanonicalPlayerMatchStatsExport
}

func newEngagementQualityFixture(t *testing.T, states []models.CanonicalPlayerState) engagementQualityFixture {
	t.Helper()
	matchID := "quality-match"
	tickRate := 64.0
	start, end := 0, 500
	rounds := []models.CanonicalRound{{RoundID: matchID + ":round:001", RoundNumber: 1, StartTick: &start, EndTick: &end}}
	participants := []models.CanonicalParticipant{
		{PlayerID: "steam:1", SteamID: "1", TeamID: "ct-team"},
		{PlayerID: "steam:2", SteamID: "2", TeamID: "t-team"},
		{PlayerID: "steam:3", SteamID: "3", TeamID: "ct-team"},
		{PlayerID: "steam:4", SteamID: "4", TeamID: "t-team"},
	}
	events := engagementQualityEvents(matchID)
	statesByRound := map[int][]models.CanonicalPlayerState{1: states}
	engagements, trades, err := engagementpkg.Derive(matchID, tickRate, rounds, participants, events, statesByRound, engagementQualityVisibility{})
	if err != nil {
		t.Fatalf("Derive() error = %v", err)
	}
	ctx := &models.DemoContext{AI_PlayersSummary: []models.AI_PlayerStats{
		{SteamID: "1"}, {SteamID: "2"}, {SteamID: "3"}, {SteamID: "4"},
	}}
	stats := buildCanonicalPlayerMatchStats(ctx, matchID, trades, events)
	return engagementQualityFixture{
		matchID: matchID, tickRate: tickRate, rounds: rounds, participants: participants,
		events: events, states: statesByRound, engagements: engagements, trades: trades, stats: stats,
	}
}

func (fixture engagementQualityFixture) assess() engagementQualityAssessment {
	return assessEngagementQuality(
		fixture.matchID, fixture.tickRate, fixture.rounds, fixture.participants,
		fixture.events, fixture.states, fixture.engagements, fixture.trades, fixture.stats,
		engagementQualityVisibility{},
	)
}

func engagementQualityEvents(matchID string) []models.CanonicalCombatEvent {
	return []models.CanonicalCombatEvent{
		engagementQualityEvent(matchID, "fire-12", "weapon_fire", 90, 1, "steam:1", "ct", "", "", nil),
		engagementQualityEvent(matchID, "hurt-12", "player_hurt", 100, 1, "steam:1", "ct", "steam:2", "t", []string{"fire-12"}),
		engagementQualityEvent(matchID, "kill-12", "kill", 100, 2, "steam:1", "ct", "steam:2", "t", []string{"hurt-12"}),
		engagementQualityEvent(matchID, "fire-41", "weapon_fire", 110, 1, "steam:4", "t", "", "", nil),
		engagementQualityEvent(matchID, "hurt-41", "player_hurt", 120, 1, "steam:4", "t", "steam:1", "ct", []string{"fire-41"}),
		engagementQualityEvent(matchID, "kill-41", "kill", 120, 2, "steam:4", "t", "steam:1", "ct", []string{"hurt-41"}),
	}
}

func engagementQualityEvent(
	matchID, eventID, eventType string,
	tick, sequence int,
	actor, actorSide, target, targetSide string,
	sources []string,
) models.CanonicalCombatEvent {
	event := models.CanonicalCombatEvent{
		SchemaID: "stratai.combat_event@2", MatchID: matchID, EventID: eventID,
		RoundID: matchID + ":round:001", RoundNumber: 1, Tick: tick,
		SequenceInTick: sequence, SequenceInRound: tick*10 + sequence,
		EventType: eventType, SourceEventIDs: sources, Relation: "none",
	}
	if actor != "" {
		event.ActorPlayerID = engagementQualityString(actor)
		event.ActorSide = engagementQualityString(actorSide)
	}
	if target != "" {
		event.TargetPlayerID = engagementQualityString(target)
		event.TargetSide = engagementQualityString(targetSide)
		event.Relation = "enemy"
	}
	if eventType == "weapon_fire" {
		event.ShotID = engagementQualityString("shot-" + eventID)
		result := "hit"
		event.ShotResult = &result
	}
	if eventType == "player_hurt" {
		damage := 100
		event.HealthDamage = &damage
		event.HealthDamageTaken = &damage
	}
	return event
}

func engagementQualityStates() []models.CanonicalPlayerState {
	return []models.CanonicalPlayerState{
		engagementQualityState("steam:1", "ct"),
		engagementQualityState("steam:2", "t"),
		engagementQualityState("steam:3", "ct"),
		engagementQualityState("steam:4", "t"),
	}
}

func engagementQualityState(playerID, side string) models.CanonicalPlayerState {
	velocity := 0.0
	weapon := "rifle"
	return models.CanonicalPlayerState{
		SchemaID: "stratai.player_state@3", MatchID: "quality-match", StateID: "state-" + playerID,
		RoundID: "quality-match:round:001", RoundNumber: 1, Tick: 80, PlayerID: playerID, Side: side,
		IsAlive: true, HorizontalVelocityWorldUPS: &velocity, VelocitySource: "position_delta",
		ActiveWeapon: &weapon, ActiveWeaponStatus: "observed",
	}
}

func engagementQualityString(value string) *string {
	return &value
}

func engagementQualityInt(value int) *int {
	return &value
}

type engagementQualityVisibility struct{}

func (engagementQualityVisibility) IsLoaded() bool { return true }

func (engagementQualityVisibility) IsVisible(r3.Vector, r3.Vector) bool { return true }

func (engagementQualityVisibility) GetCallout(r3.Vector) string { return "" }

func (engagementQualityVisibility) RayCast(r3.Vector, r3.Vector, float64) (float64, r3.Vector) {
	return -1, r3.Vector{}
}
