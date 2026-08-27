package parser

import (
	"testing"

	"cs2-demo-service/models"
)

func TestBlock6QualityRejectsIdentityMoneyPriceAndPurchaseProvenance(t *testing.T) {
	metadata := block6QualityMetadata()
	zero := 0
	invalidMoney := 16001
	economyPlayers := models.CanonicalEconomyPlayerExport{
		PriceTable: metadata.PriceTable,
		Players: []models.CanonicalEconomyPlayer{{
			RoundNumber: 1,
			PlayerID:    "steam:1",
			TeamID:      "CT",
			Side:        "ct",
			Money: models.CanonicalEconomyMoney{
				RoundStartObserved: models.CanonicalAvailabilityValue{Amount: &invalidMoney, Status: "observed"},
				FreezeEndObserved:  models.CanonicalAvailabilityValue{Amount: &zero, Status: "observed"},
				AfterBuyObserved:   models.CanonicalAvailabilityValue{Amount: &zero, Status: "observed"},
				AfterBuyCalculated: models.CanonicalAvailabilityValue{Amount: &zero, Status: "calculated"},
				RoundEndObserved:   models.CanonicalAvailabilityValue{Amount: &zero, Status: "observed"},
			},
			InventoryStart:  block6ObservedInventory(&zero),
			InventoryFreeze: block6ObservedInventory(&zero),
			InventoryEnd:    block6ObservedInventory(&zero),
			SpentInBuy:      models.CanonicalAvailabilityValue{Amount: &zero, Status: "observed"},
			Transactions: []models.CanonicalEconomyTransaction{{
				TransactionID: "purchase:1",
				Type:          "purchase",
				ActorPlayerID: "",
				Item: models.CanonicalEconomyItem{
					PurchasedItem: stringPointer("future"),
					Price: models.CanonicalEconomyPrice{
						Amount: &zero, Status: "unknown", TableVersion: block6PriceTableVersion,
					},
				},
			}},
		}},
	}
	assessment := assessBlock6Quality(
		&models.DemoContext{CurrentRound: 1},
		[]models.CanonicalRound{{RoundNumber: 1, SideAssignments: []models.CanonicalSideAssignment{{TeamID: "team_a", Side: "ct"}, {TeamID: "team_b", Side: "t"}}}},
		[]models.CanonicalParticipant{{PlayerID: "steam:1", TeamID: "team_a"}},
		nil,
		models.CanonicalEconomyRoundExport{EconomyRules: metadata.EconomyRules},
		economyPlayers,
		models.CanonicalPlayerStatsExport{},
		models.CanonicalClutchEventExport{},
		metadata,
	)
	if assessment.teamIdentity == 0 || assessment.moneyTransitions == 0 || assessment.priceTableVersion == 0 || assessment.purchaseProvenance == 0 {
		t.Fatalf("hard Block 6 violations were not detected: %+v", assessment)
	}
}

func TestBlock6QualityDetectsMissingSimultaneousTeamClutch(t *testing.T) {
	metadata := block6QualityMetadata()
	participants := []models.CanonicalParticipant{
		{PlayerID: "steam:1", TeamID: "team_a"}, {PlayerID: "steam:2", TeamID: "team_a"},
		{PlayerID: "steam:3", TeamID: "team_b"}, {PlayerID: "steam:4", TeamID: "team_b"},
	}
	victimTwo := "steam:2"
	victimThree := "steam:3"
	combat := []models.CanonicalCombatEvent{
		{EventID: "kill:1", RoundNumber: 1, Tick: 10, IsKill: true, TargetPlayerID: &victimTwo},
		{EventID: "kill:2", RoundNumber: 1, Tick: 11, IsKill: true, TargetPlayerID: &victimThree},
	}
	rounds := []models.CanonicalRound{{
		RoundNumber: 1,
		SideAssignments: []models.CanonicalSideAssignment{
			{TeamID: "team_a", Side: "ct"}, {TeamID: "team_b", Side: "t"},
		},
	}}
	assessment := assessBlock6Quality(
		&models.DemoContext{CurrentRound: 1}, rounds, participants, combat,
		models.CanonicalEconomyRoundExport{EconomyRules: metadata.EconomyRules},
		models.CanonicalEconomyPlayerExport{PriceTable: metadata.PriceTable},
		models.CanonicalPlayerStatsExport{},
		models.CanonicalClutchEventExport{},
		metadata,
	)
	if assessment.clutchReconciliation == 0 {
		t.Fatal("missing CT/T simultaneous clutch attempts were not detected")
	}
}

func TestBlock6UnavailableInventoryIsCoverageWarningNotHardFailure(t *testing.T) {
	metadata := block6QualityMetadata()
	assessment := assessBlock6Quality(
		&models.DemoContext{CurrentRound: 1},
		[]models.CanonicalRound{{RoundNumber: 1, SideAssignments: []models.CanonicalSideAssignment{{TeamID: "team_a", Side: "ct"}}}},
		[]models.CanonicalParticipant{{PlayerID: "steam:1", TeamID: "team_a"}},
		nil,
		models.CanonicalEconomyRoundExport{EconomyRules: metadata.EconomyRules},
		models.CanonicalEconomyPlayerExport{
			PriceTable: metadata.PriceTable,
			Players: []models.CanonicalEconomyPlayer{{
				RoundNumber: 1, PlayerID: "steam:1", TeamID: "team_a", Side: "ct",
				InventoryStart:  models.CanonicalEconomyInventory{Status: "not_observed", Items: []models.CanonicalEconomyItem{}},
				InventoryFreeze: models.CanonicalEconomyInventory{Status: "not_observed", Items: []models.CanonicalEconomyItem{}},
				InventoryEnd:    models.CanonicalEconomyInventory{Status: "not_observed", Items: []models.CanonicalEconomyItem{}},
			}},
		},
		models.CanonicalPlayerStatsExport{}, models.CanonicalClutchEventExport{}, metadata,
	)
	if assessment.nativeCalculatedViolations != 0 || assessment.observationWarnings < 3 {
		t.Fatalf("unavailable inventory was not classified as coverage: %+v", assessment)
	}
}

func block6QualityMetadata() models.CanonicalMatchMetadata {
	return buildCanonicalMatchMetadata(
		"match",
		"",
		models.CanonicalExportProvenance{
			Source: "demo", DemoChecksum: block6Hash("demo"),
		},
	)
}

func block6ObservedInventory(value *int) models.CanonicalEconomyInventory {
	return models.CanonicalEconomyInventory{
		Status: "observed_with_calculated_valuation", NativeValue: value,
		CalculatedValue: value, Items: []models.CanonicalEconomyItem{},
	}
}
