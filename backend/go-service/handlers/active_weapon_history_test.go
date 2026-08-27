package handlers

import (
	"testing"

	"cs2-demo-service/models"
)

func TestCombatWeaponObservationSeparatesCurrentAndHistorical(t *testing.T) {
	currentWeapon := "AK-47"
	current := combatActiveWeaponObservationFromState(activeWeaponState{
		CurrentWeapon: &currentWeapon,
	}, 512)
	if current.Weapon == nil || *current.Weapon != currentWeapon ||
		current.Observation != models.ActiveWeaponObservationObservedCurrent ||
		current.ObservedTick == nil || *current.ObservedTick != 512 {
		t.Fatalf("current observation = %+v", current)
	}

	last := combatActiveWeaponObservationFromState(activeWeaponState{
		LastObservation: &models.ActiveWeaponObservation{Weapon: "AWP", Tick: 128, RoundNumber: 4},
	}, 512)
	if last.Weapon == nil || *last.Weapon != "AWP" ||
		last.Observation != models.ActiveWeaponObservationLastObserved ||
		last.ObservedTick == nil || *last.ObservedTick != 128 {
		t.Fatalf("historical observation = %+v", last)
	}

	unavailable := combatActiveWeaponObservationFromState(activeWeaponState{}, 512)
	if unavailable.Weapon != nil || unavailable.ObservedTick != nil || unavailable.Observation != models.ActiveWeaponObservationUnavailable {
		t.Fatalf("unavailable observation = %+v", unavailable)
	}
}

func TestVictimWeaponProvenanceSurvivesDuelAggregation(t *testing.T) {
	weapon := "AWP"
	observedTick := 128
	stats := aggregateVictimStats([]models.RawCombatEvent{{
		VictimSteamID:            7,
		VictimWeapon:             &weapon,
		VictimWeaponObservation:  models.ActiveWeaponObservationLastObserved,
		VictimWeaponObservedTick: &observedTick,
	}}, 7)
	if stats.ActiveWeapon == nil || *stats.ActiveWeapon != weapon ||
		stats.ActiveWeaponObservation != models.ActiveWeaponObservationLastObserved ||
		stats.ActiveWeaponObservedTick == nil || *stats.ActiveWeaponObservedTick != observedTick {
		t.Fatalf("victim weapon provenance was lost: %+v", stats)
	}

	encoded := mustDecodeJSON(t, models.AI_DuelParticipant{
		ActiveWeapon:             stats.ActiveWeapon,
		ActiveWeaponObservation:  stats.ActiveWeaponObservation,
		ActiveWeaponObservedTick: stats.ActiveWeaponObservedTick,
	})
	if encoded["weapon"] != "" || encoded["active_weapon"] != weapon ||
		encoded["active_weapon_observation"] != models.ActiveWeaponObservationLastObserved {
		t.Fatalf("victim weapon fields were conflated: %+v", encoded)
	}
}
