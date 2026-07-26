package handlers

import (
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
