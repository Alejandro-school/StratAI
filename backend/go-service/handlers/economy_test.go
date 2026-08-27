package handlers

import (
	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"testing"
)

func TestUniqueEconomyPlayersPrefersConnectedNewestEntityAndSorts(t *testing.T) {
	players := []*common.Player{
		{SteamID64: 20, EntityID: 2, IsConnected: true},
		{SteamID64: 10, EntityID: 1, IsConnected: false},
		{SteamID64: 10, EntityID: 3, IsConnected: true},
		{SteamID64: 20, EntityID: 4, IsConnected: true},
		{SteamID64: 0, EntityID: 9, IsConnected: true},
		nil,
	}
	unique := uniqueEconomyPlayers(players)
	if len(unique) != 2 || unique[0].SteamID64 != 10 || unique[0].EntityID != 3 ||
		unique[1].SteamID64 != 20 || unique[1].EntityID != 4 {
		t.Fatalf("economy participants were not normalized deterministically: %+v", unique)
	}
}

func TestWithoutLegacyEconomyRoundReplacesOnlyMatchingRound(t *testing.T) {
	rounds := []models.RoundEconomyStats{
		{Round: 1, Team: "CT"},
		{Round: 1, Team: "T"},
		{Round: 2, Team: "CT"},
	}

	got := withoutLegacyEconomyRound(rounds, 1)
	if len(got) != 1 || got[0].Round != 2 || got[0].Team != "CT" {
		t.Fatalf("matching legacy snapshots were not replaced: %+v", got)
	}
}

func TestWithoutDetailedEconomyRoundReplacesOnlyMatchingRound(t *testing.T) {
	rounds := []models.AI_EconomyRound{
		{Round: 1},
		{Round: 2},
		{Round: 1},
	}

	got := withoutDetailedEconomyRound(rounds, 1)
	if len(got) != 1 || got[0].Round != 2 {
		t.Fatalf("matching detailed snapshots were not replaced: %+v", got)
	}
}
