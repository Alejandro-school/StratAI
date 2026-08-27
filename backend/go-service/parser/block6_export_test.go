package parser

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"cs2-demo-service/models"
)

func TestClutchLedgerCovers1vXOutcomesAndSimultaneousTeams(t *testing.T) {
	for _, enemies := range []int{1, 2, 3, 5} {
		t.Run(clutchState(enemies), func(t *testing.T) {
			participants := clutchParticipants(enemies)
			winner := "team_a"
			rounds := []models.CanonicalRound{{
				RoundID: "match:round:001", RoundNumber: 1, WinnerTeamID: &winner,
				SideAssignments: []models.CanonicalSideAssignment{{TeamID: "team_a", Side: "ct"}, {TeamID: "team_b", Side: "t"}},
			}}
			events := []models.CanonicalCombatEvent{{
				EventID: "kill:1", RoundNumber: 1, Tick: 100, SequenceInTick: 1, IsKill: true,
				ActorPlayerID: stringPointer("steam:101"), TargetPlayerID: stringPointer("steam:2"), Relation: "enemy",
			}}
			export := buildCanonicalClutchEvents("match", rounds, participants, events)
			attempt := clutchForTeam(t, export.ClutchEvents, "team_a")
			if attempt.State != clutchState(enemies) || attempt.EnemiesAtStart != enemies || !attempt.Attempt || attempt.Result != "won" {
				t.Fatalf("unexpected clutch attempt: %+v", attempt)
			}
			if enemies == 1 && len(export.ClutchEvents) != 2 {
				t.Fatalf("1v1 must preserve simultaneous attempts for both teams: %+v", export.ClutchEvents)
			}
		})
	}
}

func TestClutchLedgerLostAndNotEvaluable(t *testing.T) {
	participants := clutchParticipants(2)
	events := []models.CanonicalCombatEvent{{
		EventID: "kill:1", RoundNumber: 1, Tick: 100, SequenceInTick: 1, IsKill: true,
		ActorPlayerID: stringPointer("steam:101"), TargetPlayerID: stringPointer("steam:2"), Relation: "enemy",
	}}
	winner := "team_b"
	lost := buildCanonicalClutchEvents("match", []models.CanonicalRound{{RoundID: "match:round:001", RoundNumber: 1, WinnerTeamID: &winner}}, participants, events)
	if attempt := clutchForTeam(t, lost.ClutchEvents, "team_a"); attempt.Result != "lost" || attempt.EvaluationStatus != "evaluated" {
		t.Fatalf("lost clutch was not reconciled: %+v", attempt)
	}
	unavailable := buildCanonicalClutchEvents("match", []models.CanonicalRound{{RoundID: "match:round:001", RoundNumber: 1}}, participants, events)
	if attempt := clutchForTeam(t, unavailable.ClutchEvents, "team_a"); attempt.Result != "not_evaluable" || attempt.EvaluationStatus != "not_evaluable" {
		t.Fatalf("missing round outcome was fabricated: %+v", attempt)
	}
}

func TestClutchLedgerIsDeterministicForReverseEventsAndParticipantOrder(t *testing.T) {
	winner := "team_a"
	rounds := []models.CanonicalRound{{RoundID: "match:round:001", RoundNumber: 1, WinnerTeamID: &winner}}
	participants := clutchParticipants(3)
	events := []models.CanonicalCombatEvent{
		{EventID: "kill:2", RoundNumber: 1, Tick: 110, SequenceInTick: 1, IsKill: true, ActorPlayerID: stringPointer("steam:1"), TargetPlayerID: stringPointer("steam:101"), Relation: "enemy"},
		{EventID: "kill:1", RoundNumber: 1, Tick: 100, SequenceInTick: 1, IsKill: true, ActorPlayerID: stringPointer("steam:101"), TargetPlayerID: stringPointer("steam:2"), Relation: "enemy"},
	}
	first := buildCanonicalClutchEvents("match", rounds, participants, events)
	reverseParticipants := append([]models.CanonicalParticipant(nil), participants...)
	for left, right := 0, len(reverseParticipants)-1; left < right; left, right = left+1, right-1 {
		reverseParticipants[left], reverseParticipants[right] = reverseParticipants[right], reverseParticipants[left]
	}
	reverseEvents := []models.CanonicalCombatEvent{events[1], events[0]}
	second := buildCanonicalClutchEvents("match", rounds, reverseParticipants, reverseEvents)
	leftJSON, _ := json.Marshal(first)
	rightJSON, _ := json.Marshal(second)
	if !reflect.DeepEqual(leftJSON, rightJSON) {
		t.Fatalf("clutch JSON depends on insertion order:\n%s\n%s", leftJSON, rightJSON)
	}
}

func TestEconomyTransactionsPreserveUnknownPricesOwnersRefundsAndExchange(t *testing.T) {
	priceVersion := block6PriceTableVersion
	round := models.AI_EconomyRound{Round: 1, Events: &models.AI_EconomyRoundEvents{
		Drops:   []models.AI_EconomyDrop{{Tick: 10, DropperID: 11, ReceiverID: 22, Weapon: "AK-47", WeaponValue: 2700, PriceStatus: "known", PriceTableVersion: priceVersion, EntityID: 99, PickedUp: true}},
		Pickups: []models.AI_EconomyPickup{{Tick: 11, PlayerID: 22, FromPlayerID: 11, Weapon: "AK-47", WeaponValue: 2700, PriceStatus: "known", PriceTableVersion: priceVersion, EntityID: 99, FromDrop: true}},
		Refunds: []models.AI_EconomyRefund{{Tick: 12, PlayerID: 22, Weapon: "Unknown", RefundValue: 0, PriceStatus: "unknown", PriceTableVersion: priceVersion}},
	}}
	player := models.AI_EconomyPlayer{SteamID: 22, Purchases: []models.AI_WeaponItem{{Weapon: "Future Item", PriceStatus: "unknown", PriceTableVersion: priceVersion, ObservationStatus: "observed", OriginalOwnerID: 22, OriginalOwnerStatus: "observed"}}}
	transactions := canonicalPlayerTransactions("match", round, player)
	types := make(map[string]models.CanonicalEconomyTransaction)
	for _, transaction := range transactions {
		types[transaction.Type] = transaction
	}
	for _, kind := range []string{"purchase", "pickup", "exchange", "refund"} {
		if _, ok := types[kind]; !ok {
			t.Fatalf("missing %s transaction: %+v", kind, transactions)
		}
	}
	if types["purchase"].Item.Price.Amount != nil || types["purchase"].Item.Price.Status != "unknown" {
		t.Fatalf("unknown purchase price became zero: %+v", types["purchase"])
	}
	if types["purchase"].Item.OriginalOwnerID == nil || *types["purchase"].Item.OriginalOwnerID != "steam:22" {
		t.Fatalf("purchase owner was not preserved by player ID: %+v", types["purchase"])
	}
	if types["pickup"].Item.OriginalOwnerID == nil || *types["pickup"].Item.OriginalOwnerID != "steam:11" {
		t.Fatalf("pickup owner was not preserved by player ID: %+v", types["pickup"])
	}
}

func TestEconomyRewardsExposePlantExplodeDefuseAndKillsWithoutInventingAmounts(t *testing.T) {
	winner := "team_a"
	round := models.CanonicalRound{RoundID: "match:round:001", RoundNumber: 1, WinnerTeamID: &winner}
	actor := "steam:11"
	objectives := []models.CanonicalObjectiveEvent{
		{EventID: "plant", RoundNumber: 1, EventType: "bomb_plant", ActorPlayerID: &actor},
		{EventID: "explode", RoundNumber: 1, EventType: "bomb_explode", ActorPlayerID: &actor},
		{EventID: "defuse", RoundNumber: 1, EventType: "bomb_defuse", ActorPlayerID: &actor},
	}
	combat := []models.CanonicalCombatEvent{{EventID: "kill", RoundNumber: 1, IsKill: true, Relation: "enemy", ActorPlayerID: &actor}}
	rewards := canonicalTeamRewards("match", round, "team_a", "t", "win", models.CanonicalLossBonus{}, map[string]string{actor: "team_a"}, objectives, combat)
	types := make(map[string]bool)
	for _, reward := range rewards {
		types[reward.Type] = true
		if reward.Type != "loss_bonus" && (reward.ObservedAmount != nil || reward.CalculatedAmount != nil) {
			t.Fatalf("unobserved reward amount was fabricated: %+v", reward)
		}
	}
	for _, kind := range []string{"round_win", "plant", "explode", "defuse", "kill"} {
		if !types[kind] {
			t.Fatalf("missing %s reward: %+v", kind, rewards)
		}
	}
}

func TestMetadataKeepsMissingAndOutOfRangeDatesExplicit(t *testing.T) {
	metadata := buildCanonicalMatchMetadata("match", "", models.CanonicalExportProvenance{Source: "demo", DemoChecksum: block6Hash("demo")})
	if metadata.PlayedAt != nil || metadata.PlayedAtStatus != "unavailable" || metadata.PlayedAtSource != nil {
		t.Fatalf("missing date was fabricated: %+v", metadata)
	}
	if metadata.PriceTable.ApplicabilityStatus != "unverified_match_date" {
		t.Fatalf("missing date applicability = %s", metadata.PriceTable.ApplicabilityStatus)
	}

	metadata = buildCanonicalMatchMetadata("match", "2025-01-02T03:04:05Z", models.CanonicalExportProvenance{Source: "demo", DemoChecksum: block6Hash("demo")})
	if metadata.PlayedAt == nil || metadata.PlayedAtSource == nil || *metadata.PlayedAtSource != "demo" {
		t.Fatalf("reliable origin date was not preserved: %+v", metadata)
	}
	if metadata.PriceTable.ApplicabilityStatus != "unverified_outside_effective_range" {
		t.Fatalf("historical table applicability = %s", metadata.PriceTable.ApplicabilityStatus)
	}
}

func TestLedgerOnlyPlayerKeepsNativeScoreboardUnavailableAndRatingApproximate(t *testing.T) {
	legacy := models.CanonicalPlayerMatchStatsExport{
		Players: []models.CanonicalPlayerMatchStats{{
			PlayerID: "steam:11",
			Metrics: models.AI_PlayerStats{
				SteamID: "11", RoundsPlayed: 1, HLTVRating: 1.05,
				KillsObserved: 1, DeathsObserved: 0,
			},
		}},
	}
	stats := buildCanonicalBlock6PlayerStats(
		"match",
		canonicalRosterInfo{playerTeams: map[uint64]string{11: "team_a"}},
		legacy,
		models.CanonicalClutchEventExport{},
	)
	if len(stats.Players) != 1 {
		t.Fatalf("ledger-only player missing: %+v", stats)
	}
	player := stats.Players[0]
	if player.NativeScoreboardStatus != "unavailable" || player.NativeScoreboard != nil {
		t.Fatalf("native scoreboard was fabricated: %+v", player)
	}
	if !player.Rating.Approximate || player.Rating.AlgorithmVersion != block6RatingVersion || player.Rating.Value == nil {
		t.Fatalf("approximate rating provenance missing: %+v", player.Rating)
	}
}

func clutchParticipants(enemies int) []models.CanonicalParticipant {
	participants := []models.CanonicalParticipant{
		{PlayerID: "steam:1", TeamID: "team_a"},
		{PlayerID: "steam:2", TeamID: "team_a"},
	}
	for index := 1; index <= enemies; index++ {
		participants = append(participants, models.CanonicalParticipant{PlayerID: "steam:" + strconv.Itoa(100+index), TeamID: "team_b"})
	}
	return participants
}

func clutchState(enemies int) string {
	return "1v" + strconv.Itoa(enemies)
}

func clutchForTeam(t *testing.T, events []models.CanonicalClutchEvent, teamID string) models.CanonicalClutchEvent {
	t.Helper()
	for _, event := range events {
		if event.TeamID == teamID {
			return event
		}
	}
	t.Fatalf("no clutch for %s in %+v", teamID, events)
	return models.CanonicalClutchEvent{}
}
