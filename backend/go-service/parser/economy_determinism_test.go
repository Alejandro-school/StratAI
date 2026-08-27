package parser

import (
	"reflect"
	"testing"

	"cs2-demo-service/models"
)

func TestCanonicalEconomyProjectionSortsNestedCollections(t *testing.T) {
	player := models.AI_EconomyPlayer{
		StartRoundItems: []models.AI_WeaponItem{
			{Weapon: "AK-47", EntityID: 20},
			{Weapon: "Flashbang", EntityID: 10},
		},
		Refunds: []string{"Smoke Grenade", "HE Grenade"},
	}
	details := canonicalEconomyPlayerDetails(player)
	if details.StartRoundItems[0].EntityID != 10 || !reflect.DeepEqual(details.Refunds, []string{"HE Grenade", "Smoke Grenade"}) {
		t.Fatalf("nested player economy is not deterministic: %+v", details)
	}
	if player.StartRoundItems[0].EntityID != 20 {
		t.Fatal("canonical projection mutated the parser-owned economy slice")
	}

	events := &models.AI_EconomyRoundEvents{Pickups: []models.AI_EconomyPickup{
		{Tick: 20, PlayerID: 2, EntityID: 20, Weapon: "AK-47"},
		{Tick: 10, PlayerID: 1, EntityID: 10, Weapon: "M4A1"},
	}}
	projected := canonicalEconomyEvents(events)
	if projected.Pickups[0].Tick != 10 || events.Pickups[0].Tick != 20 {
		t.Fatalf("round economy event sorting or cloning failed: projected=%+v source=%+v", projected.Pickups, events.Pickups)
	}
}
