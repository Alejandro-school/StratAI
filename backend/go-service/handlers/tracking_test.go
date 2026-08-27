package handlers

import (
	"cs2-demo-service/pkg/objective"
	"reflect"
	"testing"
)

func TestFlattenPlayerWeaponsPreservesCountsAndOrder(t *testing.T) {
	entries := []playerWeaponEntry{
		{name: "Knife", priority: 4, count: 1},
		{name: "Flashbang", priority: 2, count: 2},
		{name: "Five-SeveN", priority: 1, count: 1},
		{name: "MP7", priority: 0, count: 1},
		{name: "Smoke Grenade", priority: 2, count: 1},
	}

	actual := flattenPlayerWeapons(entries)
	expected := []string{"MP7", "Five-SeveN", "Flashbang", "Flashbang", "Smoke Grenade", "Knife"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected inventory: got %v, want %v", actual, expected)
	}
}

func TestReconcileObjectiveRoleRejectsStickyNativePlantingFlag(t *testing.T) {
	hasC4, isPlanting, isDefusing, disagreement := reconcileObjectiveRole(
		42,
		true,
		false,
		true,
		false,
		objective.Snapshot{Phase: objective.PhasePreplant, Carrier: objective.Actor{SteamID: 7}},
	)
	if hasC4 || isPlanting || isDefusing || !disagreement {
		t.Fatalf("sticky native planting flag was not reconciled: c4=%t planting=%t defusing=%t disagreement=%t", hasC4, isPlanting, isDefusing, disagreement)
	}
}

func TestReconcileObjectiveRoleUsesLedgerPlantingActor(t *testing.T) {
	hasC4, isPlanting, isDefusing, disagreement := reconcileObjectiveRole(
		42,
		true,
		false,
		false,
		false,
		objective.Snapshot{
			Phase:          objective.PhasePlanting,
			Carrier:        objective.Actor{SteamID: 42},
			PlantingPlayer: objective.Actor{SteamID: 42},
		},
	)
	if !hasC4 || !isPlanting || isDefusing || !disagreement {
		t.Fatalf("ledger planting actor was not projected: c4=%t planting=%t defusing=%t disagreement=%t", hasC4, isPlanting, isDefusing, disagreement)
	}
}

func TestReconcileObjectiveRoleRejectsAnonymousAndDeadCarriers(t *testing.T) {
	for _, test := range []struct {
		name      string
		playerID  uint64
		alive     bool
		carrierID uint64
	}{
		{name: "anonymous", playerID: 0, alive: true, carrierID: 0},
		{name: "dead", playerID: 42, alive: false, carrierID: 42},
	} {
		t.Run(test.name, func(t *testing.T) {
			hasC4, isPlanting, isDefusing, disagreement := reconcileObjectiveRole(
				test.playerID,
				test.alive,
				true,
				false,
				false,
				objective.Snapshot{
					Phase:   objective.PhasePreplant,
					Carrier: objective.Actor{SteamID: test.carrierID},
				},
			)
			if hasC4 || isPlanting || isDefusing || !disagreement {
				t.Fatalf("invalid player retained an objective role: c4=%t planting=%t defusing=%t disagreement=%t", hasC4, isPlanting, isDefusing, disagreement)
			}
		})
	}
}

func TestReconcileObjectiveRoleRejectsStickyInventoryAfterDrop(t *testing.T) {
	hasC4, isPlanting, isDefusing, disagreement := reconcileObjectiveRole(
		42,
		true,
		true,
		false,
		false,
		objective.Snapshot{
			State: objective.StateDropped,
			Phase: objective.PhasePreplant,
		},
	)
	if hasC4 || isPlanting || isDefusing || !disagreement {
		t.Fatalf("dropped C4 remained in a player inventory: c4=%t planting=%t defusing=%t disagreement=%t", hasC4, isPlanting, isDefusing, disagreement)
	}
}

func TestFlattenPlayerWeaponsUsesLargestReportedGrenadeCount(t *testing.T) {
	entries := []playerWeaponEntry{
		{name: "Flashbang", priority: 2, count: 2},
		{name: "Flashbang", priority: 2, count: 1},
	}

	actual := flattenPlayerWeapons(entries)
	expected := []string{"Flashbang", "Flashbang"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected duplicate handling: got %v, want %v", actual, expected)
	}
}

func TestNormalizeGrenadeTypeKeepsIncendiaryDistinct(t *testing.T) {
	if actual := normalizeGrenadeType("Incendiary Grenade"); actual != "incendiary" {
		t.Fatalf("unexpected incendiary type: got %q", actual)
	}
	if actual := normalizeGrenadeType("Molotov"); actual != "molotov" {
		t.Fatalf("unexpected molotov type: got %q", actual)
	}
}

func TestMergePlayerEquipmentTracksLoadoutChangesAndTransientFrames(t *testing.T) {
	equipment := mergePlayerEquipment(
		replayPlayerEquipment{},
		true,
		[]string{"MP7", "Five-SeveN", "Flashbang", "Flashbang", "Knife"},
		"MP7",
		true,
		true,
		false,
	)
	equipment = mergePlayerEquipment(equipment, true, nil, "", true, true, false)
	expected := []string{"MP7", "Five-SeveN", "Flashbang", "Flashbang", "Knife"}
	if !reflect.DeepEqual(equipment.weapons, expected) || equipment.activeWeapon != "MP7" {
		t.Fatalf("transient frame lost equipment: %+v", equipment)
	}

	equipment = mergePlayerEquipment(
		equipment,
		true,
		[]string{"AK-47", "Five-SeveN", "Smoke Grenade", "Knife"},
		"AK-47",
		false,
		true,
		false,
	)
	if equipment.activeWeapon != "AK-47" || equipment.hasDefuseKit {
		t.Fatalf("valid loadout change was not applied: %+v", equipment)
	}
}

func TestMergePlayerEquipmentKeepsLastLoadoutAfterDeath(t *testing.T) {
	cached := replayPlayerEquipment{
		weapons:      []string{"AWP", "P250", "Knife"},
		activeWeapon: "AWP",
		hasHelmet:    true,
	}
	actual := mergePlayerEquipment(cached, false, nil, "", false, false, false)
	if !reflect.DeepEqual(actual, cached) {
		t.Fatalf("death changed cached equipment: got %+v, want %+v", actual, cached)
	}
}

func TestMergePlayerEquipmentDoesNotInventActiveWeapon(t *testing.T) {
	aliveWithoutObservation := mergePlayerEquipment(
		replayPlayerEquipment{},
		true,
		[]string{"Knife"},
		"",
		false,
		false,
		false,
	)
	if aliveWithoutObservation.activeWeapon != "" {
		t.Fatalf("invented active weapon without an observation: %+v", aliveWithoutObservation)
	}

	deadWithParserFallback := mergePlayerEquipment(
		replayPlayerEquipment{},
		false,
		nil,
		"Knife",
		false,
		false,
		false,
	)
	if deadWithParserFallback.activeWeapon != "" {
		t.Fatalf("accepted dead-player parser fallback as an observation: %+v", deadWithParserFallback)
	}
}

func TestResetPlayerEquipmentClearsRoundCache(t *testing.T) {
	handler := &ReplayHandler{
		playerEquipment: map[uint64]replayPlayerEquipment{
			42: {weapons: []string{"AK-47"}},
		},
	}
	handler.resetPlayerEquipment()
	if len(handler.playerEquipment) != 0 {
		t.Fatalf("round cache was not reset: %+v", handler.playerEquipment)
	}
}
