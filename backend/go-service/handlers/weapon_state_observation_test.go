package handlers

import (
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestNormalizeRawWeaponAmmoPreservesObservedZero(t *testing.T) {
	magazine, reserve, applicable := normalizeRawWeaponAmmo(rawWeaponAmmoObservation{
		class: common.EqClassRifle, entityAvailable: true,
		magazine: 0, reserve: 0, reserveAvailable: true,
	})
	if !applicable || magazine == nil || *magazine != 0 || reserve == nil || *reserve != 0 {
		t.Fatalf("observed zero was not preserved: applicable=%v magazine=%v reserve=%v", applicable, magazine, reserve)
	}
}

func TestNormalizeRawWeaponAmmoPreservesPositiveValues(t *testing.T) {
	magazine, reserve, applicable := normalizeRawWeaponAmmo(rawWeaponAmmoObservation{
		class: common.EqClassPistols, entityAvailable: true,
		magazine: 7, reserve: 28, reserveAvailable: true,
	})
	if !applicable || magazine == nil || *magazine != 7 || reserve == nil || *reserve != 28 {
		t.Fatalf("positive ammo missing: applicable=%v magazine=%v reserve=%v", applicable, magazine, reserve)
	}
}

func TestNormalizeRawWeaponAmmoRejectsDemoinfocsSentinels(t *testing.T) {
	tests := []struct {
		name        string
		raw         rawWeaponAmmoObservation
		wantMag     bool
		wantReserve bool
	}{
		{
			name: "missing m_iClip1 is negative one",
			raw: rawWeaponAmmoObservation{
				class: common.EqClassRifle, entityAvailable: true,
				magazine: -1, reserve: 30, reserveAvailable: true,
			},
			wantReserve: true,
		},
		{
			name: "nil entity zero is unavailable",
			raw: rawWeaponAmmoObservation{
				class: common.EqClassRifle, magazine: 0, reserve: 0,
			},
		},
		{
			name: "missing reserve property zero is unavailable",
			raw: rawWeaponAmmoObservation{
				class: common.EqClassRifle, entityAvailable: true,
				magazine: 12, reserve: 0, reserveAvailable: false,
			},
			wantMag: true,
		},
		{
			name: "negative reserve is unavailable",
			raw: rawWeaponAmmoObservation{
				class: common.EqClassSMG, entityAvailable: true,
				magazine: 20, reserve: -1, reserveAvailable: true,
			},
			wantMag: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			magazine, reserve, applicable := normalizeRawWeaponAmmo(test.raw)
			if !applicable || (magazine != nil) != test.wantMag || (reserve != nil) != test.wantReserve {
				t.Fatalf("normalized sentinels incorrectly: applicable=%v magazine=%v reserve=%v", applicable, magazine, reserve)
			}
		})
	}
}

func TestNormalizeRawWeaponAmmoRejectsKnifeC4AndGrenades(t *testing.T) {
	for _, class := range []common.EquipmentClass{common.EqClassEquipment, common.EqClassGrenade, common.EqClassUnknown} {
		magazine, reserve, applicable := normalizeRawWeaponAmmo(rawWeaponAmmoObservation{
			class: class, entityAvailable: true,
			magazine: 0, reserve: 0, reserveAvailable: true,
		})
		if applicable || magazine != nil || reserve != nil {
			t.Fatalf("class %v fabricated ammunition: applicable=%v magazine=%v reserve=%v", class, applicable, magazine, reserve)
		}
	}
}
