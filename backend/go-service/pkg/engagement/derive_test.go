package engagement

import (
	"reflect"
	"testing"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
)

func TestDeriveKeepsCausalRolesIndependentFromOutcome(t *testing.T) {
	tests := []struct {
		name               string
		events             []models.CanonicalCombatEvent
		wantInitiator      string
		wantFirstAggressor string
		wantFirstDamage    string
		wantWinner         string
		wantOutcome        string
		wantExchangeActors []string
	}{
		{
			name: "initiator wins terminal exchange",
			events: []models.CanonicalCombatEvent{
				hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-a", nil),
				kill("kill-ab", 100, 2, "a", "ct", "b", "t", "hurt-ab"),
			},
			wantInitiator: "a", wantFirstDamage: "a", wantWinner: "a",
			wantOutcome: "kill", wantExchangeActors: []string{"a"},
		},
		{
			name: "initiator loses after bilateral damage",
			events: []models.CanonicalCombatEvent{
				fire("fire-a", 90, 1, "a", "ct", "shot-a", "miss"),
				hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-b", nil),
				hurt("hurt-ba", 110, 1, "b", "t", "a", "ct", "shot-c", nil),
				kill("kill-ba", 110, 2, "b", "t", "a", "ct", "hurt-ba"),
			},
			wantInitiator: "a", wantFirstAggressor: "a", wantFirstDamage: "a", wantWinner: "b",
			wantOutcome: "kill", wantExchangeActors: []string{"a", "b"},
		},
		{
			name: "defender fires first and loses",
			events: []models.CanonicalCombatEvent{
				fire("fire-b", 90, 1, "b", "t", "shot-a", "miss"),
				hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-b", nil),
				kill("kill-ab", 100, 2, "a", "ct", "b", "t", "hurt-ab"),
			},
			wantInitiator: "b", wantFirstAggressor: "b", wantFirstDamage: "a", wantWinner: "a",
			wantOutcome: "kill", wantExchangeActors: []string{"a"},
		},
		{
			name: "bilateral non-terminal exchange",
			events: []models.CanonicalCombatEvent{
				hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-a", nil),
				hurt("hurt-ba", 110, 1, "b", "t", "a", "ct", "shot-b", nil),
			},
			wantInitiator: "a", wantFirstDamage: "a", wantOutcome: "disengaged",
			wantExchangeActors: []string{"a", "b"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engagements, _, err := deriveFixture(test.events, completeStates(), 500)
			if err != nil {
				t.Fatalf("Derive() error = %v", err)
			}
			if len(engagements.Engagements) != 1 {
				t.Fatalf("engagement count = %d, want 1", len(engagements.Engagements))
			}
			got := engagements.Engagements[0]
			assertRolePlayer(t, "initiator", got.Initiator, test.wantInitiator)
			assertRolePlayer(t, "first aggressor", got.FirstAggressor, test.wantFirstAggressor)
			assertRolePlayer(t, "first damage", got.FirstDamageDealer, test.wantFirstDamage)
			assertOptionalPlayer(t, "winner", got.OutcomeContext.WinnerPlayerID, test.wantWinner)
			if got.OutcomeContext.Outcome != test.wantOutcome {
				t.Fatalf("outcome = %q, want %q", got.OutcomeContext.Outcome, test.wantOutcome)
			}
			actors := make([]string, 0, len(got.Exchanges))
			for _, exchange := range got.Exchanges {
				actors = append(actors, exchange.ActorPlayerID)
			}
			if !reflect.DeepEqual(actors, test.wantExchangeActors) {
				t.Fatalf("exchange actors = %v, want %v", actors, test.wantExchangeActors)
			}
		})
	}
}

func TestDeriveClassifiesCollateralMultiTargetAndInterruptedCombat(t *testing.T) {
	t.Run("one shot with two victims is collateral", func(t *testing.T) {
		shot := "shot-collateral"
		events := []models.CanonicalCombatEvent{
			hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", shot, nil),
			hurt("hurt-ac", 100, 2, "a", "ct", "c", "t", shot, nil),
		}
		engagements, _, err := deriveFixture(events, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if len(engagements.Engagements) != 1 || engagements.Engagements[0].EngagementType != "collateral" || len(engagements.Engagements[0].Participants) != 3 {
			t.Fatalf("unexpected collateral projection: %+v", engagements.Engagements)
		}
	})

	t.Run("separate shots against two victims are multi target", func(t *testing.T) {
		events := []models.CanonicalCombatEvent{
			hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-a", nil),
			hurt("hurt-ac", 120, 1, "a", "ct", "c", "t", "shot-b", nil),
		}
		engagements, _, err := deriveFixture(events, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if len(engagements.Engagements) != 1 || engagements.Engagements[0].EngagementType != "multi_target" {
			t.Fatalf("unexpected multi-target projection: %+v", engagements.Engagements)
		}
	})

	t.Run("long interruption closes a non-kill engagement", func(t *testing.T) {
		events := []models.CanonicalCombatEvent{
			hurt("hurt-ab-1", 100, 1, "a", "ct", "b", "t", "shot-a", nil),
			hurt("hurt-ab-2", 197, 1, "a", "ct", "b", "t", "shot-b", nil),
		}
		engagements, _, err := deriveFixture(events, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if len(engagements.Engagements) != 2 {
			t.Fatalf("engagement count = %d, want 2", len(engagements.Engagements))
		}
		for _, engagement := range engagements.Engagements {
			if engagement.OutcomeContext.Outcome != "disengaged" {
				t.Fatalf("interrupted engagement outcome = %q", engagement.OutcomeContext.Outcome)
			}
		}
	})

	t.Run("nearby independent pairs stay separate", func(t *testing.T) {
		events := []models.CanonicalCombatEvent{
			hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-a", nil),
			hurt("hurt-cd", 110, 1, "c", "ct", "d", "t", "shot-c", nil),
		}
		engagements, _, err := deriveFixture(events, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if len(engagements.Engagements) != 2 {
			t.Fatalf("engagement count = %d, want 2", len(engagements.Engagements))
		}
	})
}

func TestDeriveAggressorPreludeRequiresExclusiveCausalAttribution(t *testing.T) {
	t.Run("one miss cannot initiate two independent engagements", func(t *testing.T) {
		events := []models.CanonicalCombatEvent{
			fire("fire-a", 90, 1, "a", "ct", "shot-miss", "miss"),
			hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-a", nil),
			hurt("hurt-da", 110, 1, "d", "t", "a", "ct", "shot-d", nil),
		}
		engagements, _, err := deriveFixture(events, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if len(engagements.Engagements) != 2 {
			t.Fatalf("engagement count = %d, want 2", len(engagements.Engagements))
		}
		for _, engagement := range engagements.Engagements {
			if engagement.FirstAggressor.Status != "unavailable" {
				t.Fatalf("ambiguous miss assigned to %s: %+v", engagement.EngagementID, engagement.FirstAggressor)
			}
			for _, sourceID := range engagement.SourceEventIDs {
				if sourceID == "fire-a" {
					t.Fatalf("ambiguous miss leaked into %s provenance", engagement.EngagementID)
				}
			}
		}
	})

	t.Run("post-damage shot ancestor cannot become first aggressor", func(t *testing.T) {
		events := []models.CanonicalCombatEvent{
			hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-a", []string{"fire-after"}),
			fire("fire-after", 100, 2, "a", "ct", "shot-a", "hit"),
		}
		engagements, _, err := deriveFixture(events, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if engagements.Engagements[0].FirstAggressor.Status != "unavailable" {
			t.Fatalf("post-damage shot became causal: %+v", engagements.Engagements[0].FirstAggressor)
		}
	})

	t.Run("observed causal ancestor suppresses inferred miss prelude", func(t *testing.T) {
		events := []models.CanonicalCombatEvent{
			fire("fire-miss", 90, 1, "a", "ct", "shot-miss", "miss"),
			fire("fire-hit", 95, 1, "a", "ct", "shot-hit", "hit"),
			hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-hit", []string{"fire-hit"}),
		}
		engagements, _, err := deriveFixture(events, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		engagement := engagements.Engagements[0]
		if engagement.FirstAggressor.Status != "observed" || engagement.FirstAggressor.Source != "causal_shot_ancestor" {
			t.Fatalf("causal ancestor role = %+v", engagement.FirstAggressor)
		}
		for _, sourceID := range engagement.SourceEventIDs {
			if sourceID == "fire-miss" {
				t.Fatal("redundant inferred prelude was retained")
			}
		}
	})
}

func TestDeriveTradesUseExplicitEligibilityAndOneToOneMatching(t *testing.T) {
	t.Run("isolated death is not tradeable", func(t *testing.T) {
		events := fatalExchange("ab", 100, "b", "t", "a", "ct")
		states := []models.CanonicalPlayerState{
			state("a", "ct", 80, true), state("b", "t", 80, true),
			state("c", "ct", 80, false), state("d", "t", 80, true),
		}
		_, trades, err := deriveFixture(events, states, 500)
		if err != nil {
			t.Fatal(err)
		}
		if len(trades.Candidates) != 1 || trades.Candidates[0].Evaluation != "not_tradeable" {
			t.Fatalf("unexpected isolated-death evaluation: %+v", trades.Candidates)
		}
	})

	t.Run("no attempt and failed attempt remain distinct", func(t *testing.T) {
		base := fatalExchange("ab", 100, "b", "t", "a", "ct")
		_, noAttempt, err := deriveFixture(base, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if noAttempt.Candidates[0].Evaluation != "not_attempted" {
			t.Fatalf("evaluation = %q, want not_attempted", noAttempt.Candidates[0].Evaluation)
		}
		attempt := append(base, hurt("hurt-cb", 120, 1, "c", "ct", "b", "t", "shot-c", nil))
		_, failed, err := deriveFixture(attempt, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if failed.Candidates[0].Evaluation != "failed" || !reflect.DeepEqual(failed.Candidates[0].AttemptEventIDs, []string{"hurt-cb"}) {
			t.Fatalf("unexpected failed attempt: %+v", failed.Candidates[0])
		}
	})

	t.Run("response outside the millisecond-derived window is not a trade", func(t *testing.T) {
		events := append(fatalExchange("ab", 100, "b", "t", "a", "ct"), fatalExchange("cb", 421, "c", "ct", "b", "t")...)
		_, trades, err := deriveFixture(events, completeStates(), 500)
		if err != nil {
			t.Fatal(err)
		}
		for _, candidate := range trades.Candidates {
			if candidate.OriginalKillEventID == "kill-ab" {
				if candidate.Evaluation != "not_attempted" || candidate.TradeCompletionID != nil {
					t.Fatalf("out-of-window response completed a trade: %+v", candidate)
				}
				return
			}
		}
		t.Fatal("original trade candidate not found")
	})

	t.Run("eligibility keeps two possible traders and excludes a historically observed dead teammate", func(t *testing.T) {
		events := fatalExchange("ab", 100, "b", "t", "a", "ct")
		states := append(completeStates(), state("e", "ct", 80, true))
		states = append(states,
			models.CanonicalPlayerState{
				SchemaID: "stratai.player_state@3", StateID: "state-c-dead", RoundID: "match:round:001",
				RoundNumber: 1, Tick: 95, PlayerID: "c", Side: "ct", IsAlive: false,
			},
		)
		_, trades, err := deriveFixtureWithParticipants(events, states, fixtureParticipantsWithE(), 500)
		if err != nil {
			t.Fatal(err)
		}
		candidate := trades.Candidates[0]
		if !reflect.DeepEqual(candidate.EligibleTeammatePlayerIDs, []string{"e"}) {
			t.Fatalf("eligible teammates = %v, want only living e", candidate.EligibleTeammatePlayerIDs)
		}
		states = completeStates()
		states = append(states, state("e", "ct", 80, true))
		_, trades, err = deriveFixtureWithParticipants(events, states, fixtureParticipantsWithE(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(trades.Candidates[0].EligibleTeammatePlayerIDs, []string{"c", "e"}) {
			t.Fatalf("two eligible traders were not retained: %+v", trades.Candidates[0])
		}
	})

	t.Run("boundary completion and counter trade are retained", func(t *testing.T) {
		events := append(fatalExchange("ab", 100, "b", "t", "a", "ct"), fatalExchange("cb", 420, "c", "ct", "b", "t")...)
		events = append(events, fatalExchange("dc", 430, "d", "t", "c", "ct")...)
		_, trades, err := deriveFixture(events, completeStates(), 800)
		if err != nil {
			t.Fatal(err)
		}
		if len(trades.Completions) != 2 || trades.Completions[0].ElapsedTicks != 320 || trades.Completions[0].TraderPlayerID != "c" {
			t.Fatalf("unexpected boundary/counter completions: %+v", trades.Completions)
		}
		candidateByKill := make(map[string]models.CanonicalTradeCandidate)
		for _, candidate := range trades.Candidates {
			candidateByKill[candidate.OriginalKillEventID] = candidate
		}
		counter := candidateByKill["kill-cb"]
		if counter.CounterTradeOfCompletionID == nil || *counter.CounterTradeOfCompletionID != trades.Completions[0].TradeCompletionID {
			t.Fatalf("counter-trade link missing: %+v", counter)
		}
	})

	t.Run("one response chooses the most recent overlapping candidate", func(t *testing.T) {
		events := append(fatalExchange("ab", 100, "b", "t", "a", "ct"), fatalExchange("eb", 110, "b", "t", "e", "ct")...)
		events = append(events, fatalExchange("cb", 120, "c", "ct", "b", "t")...)
		states := append(completeStates(), state("e", "ct", 80, true))
		_, trades, err := deriveFixtureWithParticipants(events, states, fixtureParticipantsWithE(), 500)
		if err != nil {
			t.Fatal(err)
		}
		if len(trades.Completions) != 1 || trades.Completions[0].OriginalVictimPlayerID != "e" {
			t.Fatalf("overlapping candidates were not matched most-recent-first: %+v", trades.Completions)
		}
	})

	t.Run("missing state is explicit and cannot complete a trade", func(t *testing.T) {
		events := append(fatalExchange("ab", 100, "b", "t", "a", "ct"), fatalExchange("cb", 120, "c", "ct", "b", "t")...)
		_, trades, err := deriveFixture(events, nil, 500)
		if err != nil {
			t.Fatal(err)
		}
		if trades.Candidates[0].Evaluation != "not_evaluable" || len(trades.Completions) != 0 {
			t.Fatalf("missing eligibility observations were treated as facts: %+v", trades)
		}
	})
}

func TestTradeabilityRequiresCompletePhysicalEvidence(t *testing.T) {
	events := fatalExchange("ab", 100, "b", "t", "a", "ct")

	derive := func(t *testing.T, states []models.CanonicalPlayerState, visibility fixtureVisibility) models.CanonicalTradeCandidate {
		t.Helper()
		start, end := 0, 500
		rounds := []models.CanonicalRound{{RoundID: "match:round:001", RoundNumber: 1, StartTick: &start, EndTick: &end}}
		_, trades, err := Derive(
			"match", 64, rounds, fixtureParticipants(), events,
			map[int][]models.CanonicalPlayerState{1: states}, visibility,
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(trades.Candidates) != 1 || len(trades.Candidates[0].Connections) != 1 {
			t.Fatalf("unexpected physical trade rows: %+v", trades)
		}
		return trades.Candidates[0]
	}

	positionedStates := func(teammateX, killerX float64, teammateYaw float32, teammateAlive bool) []models.CanonicalPlayerState {
		states := completeStates()
		for index := range states {
			switch states[index].PlayerID {
			case "b":
				states[index].Position.X = killerX
			case "c":
				states[index].Position.X = teammateX
				states[index].ViewYawDeg = teammateYaw
				states[index].IsAlive = teammateAlive
			}
		}
		return states
	}

	t.Run("all required evidence produces a positive tri-state", func(t *testing.T) {
		candidate := derive(t, positionedStates(0, 100, 0, true), fixtureVisibility{})
		if candidate.TradePossible == nil || !*candidate.TradePossible || candidate.TradePossibleStatus != "derived" || candidate.Evaluation != "not_attempted" {
			t.Fatalf("complete evidence was not accepted: %+v", candidate)
		}
		connection := candidate.Connections[0]
		if connection.StateAvailabilityTick == nil || *connection.StateAvailabilityTick > candidate.DeathTick ||
			connection.LineOfSight == nil || !*connection.LineOfSight || connection.Eligible == nil || !*connection.Eligible {
			t.Fatalf("positive connection lacks causal physical evidence: %+v", connection)
		}
	})

	t.Run("missing map geometry abstains instead of inventing false", func(t *testing.T) {
		candidate := derive(t, positionedStates(0, 100, 0, true), fixtureVisibility{unavailable: true})
		connection := candidate.Connections[0]
		if candidate.TradePossible != nil || candidate.TradePossibleStatus != "unavailable" || candidate.Evaluation != "not_evaluable" ||
			connection.LineOfSight != nil || connection.LineOfSightStatus != "unavailable" || connection.Eligible != nil {
			t.Fatalf("missing geometry was collapsed into a value: %+v", candidate)
		}
	})

	t.Run("blocked line of sight is a measured negative", func(t *testing.T) {
		candidate := derive(t, positionedStates(0, 100, 0, true), fixtureVisibility{blocked: true})
		if candidate.TradePossible == nil || *candidate.TradePossible || candidate.Evaluation != "not_tradeable" ||
			!testContainsString(candidate.Connections[0].IneligibilityReasons, "no_line_of_sight") {
			t.Fatalf("blocked geometry was not rejected: %+v", candidate)
		}
	})

	t.Run("distance connection time orientation and alive are mandatory", func(t *testing.T) {
		cases := []struct {
			name   string
			states []models.CanonicalPlayerState
			reason string
		}{
			{name: "far", states: positionedStates(0, 2_000, 0, true), reason: "distance_exceeds_threshold"},
			{name: "back-facing", states: positionedStates(100, 0, 0, true), reason: "facing_away"},
			{name: "dead", states: positionedStates(0, 100, 0, false), reason: "teammate_dead"},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				candidate := derive(t, test.states, fixtureVisibility{})
				if candidate.TradePossible == nil || *candidate.TradePossible || candidate.Evaluation != "not_tradeable" ||
					!testContainsString(candidate.Connections[0].IneligibilityReasons, test.reason) {
					t.Fatalf("required physical condition %q was ignored: %+v", test.reason, candidate)
				}
			})
		}
	})
}

func TestDeriveIsInvariantToInputPermutation(t *testing.T) {
	events := append(fatalExchange("ab", 100, "b", "t", "a", "ct"), fatalExchange("cb", 120, "c", "ct", "b", "t")...)
	states := completeStates()
	participants := fixtureParticipants()
	wantEngagements, wantTrades, err := deriveFixtureWithParticipants(events, states, participants, 500)
	if err != nil {
		t.Fatal(err)
	}
	reverse(events)
	reverse(states)
	reverse(participants)
	gotEngagements, gotTrades, err := deriveFixtureWithParticipants(events, states, participants, 500)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotEngagements, wantEngagements) || !reflect.DeepEqual(gotTrades, wantTrades) {
		t.Fatalf("permutation changed derivation\nwant engagements=%+v\ngot engagements=%+v\nwant trades=%+v\ngot trades=%+v", wantEngagements, gotEngagements, wantTrades, gotTrades)
	}
}

func TestDeriveCausalContextIgnoresParticipantsFirstSeenAfterT0(t *testing.T) {
	base := []models.CanonicalCombatEvent{
		hurt("hurt-ab", 100, 1, "a", "ct", "b", "t", "shot-a", nil),
		hurt("hurt-ac", 120, 1, "a", "ct", "c", "t", "shot-b", nil),
	}
	mutated := []models.CanonicalCombatEvent{
		base[0],
		hurt("hurt-ad", 120, 1, "a", "ct", "d", "t", "shot-b", nil),
	}

	want, _, err := deriveFixture(base, completeStates(), 500)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := deriveFixture(mutated, completeStates(), 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(want.Engagements) != 1 || len(got.Engagements) != 1 {
		t.Fatalf("expected one engagement, got %d and %d", len(want.Engagements), len(got.Engagements))
	}
	if !reflect.DeepEqual(want.Engagements[0].CausalContext, got.Engagements[0].CausalContext) {
		t.Fatalf("post-t0 participant changed causal context\nwant=%+v\ngot=%+v", want.Engagements[0].CausalContext, got.Engagements[0].CausalContext)
	}
	ids := make([]string, 0, len(got.Engagements[0].CausalContext.ParticipantStates))
	for _, state := range got.Engagements[0].CausalContext.ParticipantStates {
		ids = append(ids, state.PlayerID)
	}
	if !reflect.DeepEqual(ids, []string{"a", "b"}) {
		t.Fatalf("causal participants = %v, want only actors visible at t0", ids)
	}
}

func TestBuildCausalContextDoesNotReadLaterSameTickExchangeDistance(t *testing.T) {
	distance := 640.0
	context := buildCausalContext(
		models.CanonicalCombatEvent{Tick: 100, SequenceInTick: 1},
		[]models.CanonicalEngagementParticipant{{PlayerID: "actor"}},
		[]models.CanonicalEngagementExchange{{
			Tick: 100, SequenceInTick: 3, DistanceWorldUnits: &distance,
		}},
		nil,
	)
	if context.InitialDistanceWorldUnits != nil || context.InitialDistanceStatus != "unavailable" {
		t.Fatalf("later same-tick distance leaked into T0 context: %+v", context)
	}
}

func deriveFixture(
	events []models.CanonicalCombatEvent,
	states []models.CanonicalPlayerState,
	roundEnd int,
) (models.CanonicalEngagementsExport, models.CanonicalTradesExport, error) {
	return deriveFixtureWithParticipants(events, states, fixtureParticipants(), roundEnd)
}

func deriveFixtureWithParticipants(
	events []models.CanonicalCombatEvent,
	states []models.CanonicalPlayerState,
	participants []models.CanonicalParticipant,
	roundEnd int,
) (models.CanonicalEngagementsExport, models.CanonicalTradesExport, error) {
	start := 0
	rounds := []models.CanonicalRound{{RoundID: "match:round:001", RoundNumber: 1, StartTick: &start, EndTick: &roundEnd}}
	return Derive("match", 64, rounds, participants, events, map[int][]models.CanonicalPlayerState{1: states}, fixtureVisibility{})
}

func fixtureParticipants() []models.CanonicalParticipant {
	return []models.CanonicalParticipant{
		{PlayerID: "a", TeamID: "ct-team"},
		{PlayerID: "b", TeamID: "t-team"},
		{PlayerID: "c", TeamID: "ct-team"},
		{PlayerID: "d", TeamID: "t-team"},
	}
}

func fixtureParticipantsWithE() []models.CanonicalParticipant {
	return append(fixtureParticipants(), models.CanonicalParticipant{PlayerID: "e", TeamID: "ct-team"})
}

func completeStates() []models.CanonicalPlayerState {
	return []models.CanonicalPlayerState{
		state("a", "ct", 80, true),
		state("b", "t", 80, true),
		state("c", "ct", 80, true),
		state("d", "t", 80, true),
	}
}

func state(playerID, side string, tick int, alive bool) models.CanonicalPlayerState {
	velocity := 0.0
	weapon := "rifle"
	return models.CanonicalPlayerState{
		SchemaID: "stratai.player_state@3", StateID: "state-" + playerID, RoundID: "match:round:001",
		RoundNumber: 1, Tick: tick, PlayerID: playerID, Side: side, IsAlive: alive,
		HorizontalVelocityWorldUPS: &velocity, VelocitySource: "tracking", ActiveWeapon: &weapon, ActiveWeaponStatus: "observed",
	}
}

func fatalExchange(id string, tick int, actor, actorSide, target, targetSide string) []models.CanonicalCombatEvent {
	hurtID := "hurt-" + id
	return []models.CanonicalCombatEvent{
		hurt(hurtID, tick, 1, actor, actorSide, target, targetSide, "shot-"+id, nil),
		kill("kill-"+id, tick, 2, actor, actorSide, target, targetSide, hurtID),
	}
}

func fire(id string, tick, sequence int, actor, side, shotID, result string) models.CanonicalCombatEvent {
	return models.CanonicalCombatEvent{
		SchemaID: "stratai.combat_event@2", EventID: id, RoundID: "match:round:001", RoundNumber: 1,
		Tick: tick, SequenceInTick: sequence, SequenceInRound: sequence, EventType: "weapon_fire",
		ActorPlayerID: pointer(actor), ActorSide: pointer(side), Relation: "none", ShotID: pointer(shotID), ShotResult: pointer(result),
	}
}

func hurt(
	id string,
	tick, sequence int,
	actor, actorSide, target, targetSide, shotID string,
	sources []string,
) models.CanonicalCombatEvent {
	damage := 25
	return models.CanonicalCombatEvent{
		SchemaID: "stratai.combat_event@2", EventID: id, RoundID: "match:round:001", RoundNumber: 1,
		Tick: tick, SequenceInTick: sequence, SequenceInRound: sequence, EventType: "player_hurt", Relation: "enemy",
		ActorPlayerID: pointer(actor), ActorSide: pointer(actorSide), TargetPlayerID: pointer(target), TargetSide: pointer(targetSide),
		ShotID: pointer(shotID), HealthDamage: &damage, SourceEventIDs: sources,
	}
}

func kill(id string, tick, sequence int, actor, actorSide, target, targetSide, hurtID string) models.CanonicalCombatEvent {
	return models.CanonicalCombatEvent{
		SchemaID: "stratai.combat_event@2", EventID: id, RoundID: "match:round:001", RoundNumber: 1,
		Tick: tick, SequenceInTick: sequence, SequenceInRound: sequence, EventType: "kill", Relation: "enemy",
		ActorPlayerID: pointer(actor), ActorSide: pointer(actorSide), TargetPlayerID: pointer(target), TargetSide: pointer(targetSide),
		SourceEventIDs: []string{hurtID},
	}
}

func assertRolePlayer(t *testing.T, label string, role models.CanonicalRoleAssignment, want string) {
	t.Helper()
	assertOptionalPlayer(t, label, role.PlayerID, want)
}

func assertOptionalPlayer(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if want == "" {
		if got != nil {
			t.Fatalf("%s = %q, want unavailable", label, *got)
		}
		return
	}
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", label, got, want)
	}
}

func pointer(value string) *string {
	return &value
}

func reverse[T any](values []T) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func testContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type fixtureVisibility struct {
	unavailable bool
	blocked     bool
}

func (fixture fixtureVisibility) IsLoaded() bool {
	return !fixture.unavailable
}

func (fixture fixtureVisibility) IsVisible(_, _ r3.Vector) bool {
	return !fixture.blocked
}

func (fixtureVisibility) GetCallout(r3.Vector) string {
	return ""
}

func (fixtureVisibility) RayCast(r3.Vector, r3.Vector, float64) (float64, r3.Vector) {
	return -1, r3.Vector{}
}
