package parser

import (
	"encoding/json"
	"strings"
	"testing"

	"cs2-demo-service/models"
)

func economyCandidateFixture() *models.DemoContext {
	tick, freeze, end := 100, 200, 300
	fact := models.EconomyFact{Value: 800, Status: "observed", Source: "m_pInGameMoneyServices.m_iAccount", ObservationTick: &tick, AvailabilityTick: &tick}
	return &models.DemoContext{MapName: "de_mirage", ReplayData: &models.ReplayData{Metadata: models.ReplayMetadata{TickRate: 64}},
		EconomyObservations: &models.EconomyObservationLedger{ActorAliases: map[uint64]string{123: "p0001"}, Rounds: []models.EconomyObservedRound{{
			Instance: 1, RoundNumber: 1, StartTick: tick, FreezeEndTick: &freeze, EndTick: &end,
			Snapshots: []models.EconomySnapshot{{Phase: "initial_available", Tick: tick, CallbackSequence: 2,
				Actors: []models.EconomyActorSnapshot{{ActorAlias: "p0001", Side: "CT", Connected: true, Facts: map[string]models.EconomyFact{"money": fact}}}}},
			Events: []models.EconomyCallbackEvent{{Tick: tick}, {Tick: 101}, {Tick: freeze}, {Tick: 201}},
		}}},
	}
}

func TestEconomyCandidatePartitionsBoundariesWithoutRewritingInitial(t *testing.T) {
	ctx := economyCandidateFixture()
	checksum := strings.Repeat("a", 64)
	first, err := BuildEconomyCandidate(ctx, "demo-"+checksum[:20], checksum, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	round := first.Rounds[0]
	if len(round.Execution.Events) != 1 || len(round.Reconciliation.OtherEvents) != 3 {
		t.Fatal("boundary/later callbacks entered freeze execution")
	}
	before, _ := json.Marshal(round.InitialState)
	ctx.EconomyObservations.Rounds[0].Events = append(ctx.EconomyObservations.Rounds[0].Events, models.EconomyCallbackEvent{Kind: "refund", Tick: 299})
	ctx.AI_EconomyRounds = []models.AI_EconomyRound{{Round: 1, Players: []models.AI_EconomyPlayer{{SteamID: 123, InitialMoney: 999, MoneyAtRoundEnd: 9999}}}}
	second, err := BuildEconomyCandidate(ctx, "demo-"+checksum[:20], checksum, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(second.Rounds[0].InitialState)
	if string(before) != string(after) {
		t.Fatal("future aggregate/activity changed initial state")
	}
	encoded, _ := json.Marshal(second)
	if strings.Contains(string(encoded), "steam:123") || len(second.Participants[0].ActorKey) != 64 {
		t.Fatal("participant bridge exposed identity")
	}
	if len(second.Rounds[0].Reconciliation.Comparisons) != 2 {
		t.Fatal("missing aggregate diagnostics")
	}
}

func TestEconomyCandidateRejectsFutureFactAndAmbiguousRestartJoins(t *testing.T) {
	ctx := economyCandidateFixture()
	checksum := strings.Repeat("a", 64)
	future := 101
	fact := ctx.EconomyObservations.Rounds[0].Snapshots[0].Actors[0].Facts["money"]
	fact.AvailabilityTick = &future
	ctx.EconomyObservations.Rounds[0].Snapshots[0].Actors[0].Facts["money"] = fact
	if _, err := BuildEconomyCandidate(ctx, "demo-"+checksum[:20], checksum, "fixture"); err == nil {
		t.Fatal("future initial value accepted")
	}
	ctx = economyCandidateFixture()
	ctx.AI_EconomyRounds = []models.AI_EconomyRound{{Round: 1, Players: []models.AI_EconomyPlayer{{SteamID: 123}}}}
	restart := ctx.EconomyObservations.Rounds[0]
	restart.Instance = 2
	ctx.EconomyObservations.Rounds = append(ctx.EconomyObservations.Rounds, restart)
	candidate, err := BuildEconomyCandidate(ctx, "demo-"+checksum[:20], checksum, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	for _, round := range candidate.Rounds {
		if len(round.Reconciliation.Comparisons) != 0 {
			t.Fatal("restart joined by ambiguous round number")
		}
	}
}
