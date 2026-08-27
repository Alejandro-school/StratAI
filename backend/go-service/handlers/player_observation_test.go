package handlers

import (
	"testing"

	"cs2-demo-service/models"
)

func TestResolveActiveWeaponStateKeepsCurrentAndHistoricalFactsSeparate(t *testing.T) {
	last := &models.ActiveWeaponObservation{Weapon: "USP-S", Tick: 100, RoundNumber: 1}

	observed := resolveActiveWeaponState(true, "AK-47", last)
	if observed.Status != models.ActiveWeaponStatusObserved || observed.CurrentWeapon == nil || *observed.CurrentWeapon != "AK-47" {
		t.Fatalf("observed state = %+v", observed)
	}

	unavailable := resolveActiveWeaponState(true, "", last)
	if unavailable.Status != models.ActiveWeaponStatusUnavailable || unavailable.CurrentWeapon != nil || unavailable.LastObservation != last {
		t.Fatalf("unavailable state = %+v", unavailable)
	}

	dead := resolveActiveWeaponState(false, "", last)
	if dead.Status != models.ActiveWeaponStatusNotApplicable || dead.CurrentWeapon != nil || dead.LastObservation != last {
		t.Fatalf("dead state = %+v", dead)
	}
}

func TestResolveActiveWeaponStateNeverFabricatesKnife(t *testing.T) {
	state := resolveActiveWeaponState(true, "", nil)
	if state.CurrentWeapon != nil || state.LastObservation != nil {
		t.Fatalf("missing observation produced a weapon: %+v", state)
	}
}
