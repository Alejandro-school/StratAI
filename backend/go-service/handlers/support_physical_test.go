package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"math"
	"testing"
)

type physicalProvider struct {
	*supportProvider
	weapon *common.Equipment
}

func (p *physicalProvider) FindWeaponByEntityID(int) *common.Equipment { return p.weapon }

type physicalPawn struct{ *cashflowEntity }

func (p *physicalPawn) PropertyValueMust(name string) st.PropertyValue {
	return st.PropertyValue{Any: p.values[name]}
}

func physicalFixture() (*ReplayHandler, *common.Player, *physicalPawn, *cashflowEntity, models.SupportPlayer, models.ReplayPlayerState) {
	pawn := &physicalPawn{&cashflowEntity{ownershipEntity: ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{"m_angEyeAngles": [3]float32{10, 20, 0}, "m_bIsWalking": false, "m_fFlags": uint64(0), "m_pWeaponServices.m_hActiveWeapon": uint64(2<<14 | 10)}}, index: 5, serial: 3}}}
	w := &cashflowEntity{ownershipEntity: ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{"m_iClip1": uint32(1), "m_pReserveAmmo.0000": int32(0)}}, index: 10, serial: 2}}
	weapon := common.NewEquipment(common.EqAK47)
	weapon.Entity = w
	provider := &physicalProvider{supportProvider: &supportProvider{pawn: pawn}, weapon: weapon}
	player := common.NewPlayer(provider)
	player.SteamID64 = 7
	player.IsConnected = true
	player.Entity = &ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{"m_hPawn": uint64(1), "m_hPlayerPawn": uint64(3<<14 | 5)}}}
	alive := true
	base := models.SupportPlayer{NativeActor: 7, Side: "CT", Connected: true, IdentityStatus: "observed", Alive: &alive, Position: &models.SupportPosition{}}
	zero := 0
	tick := 100
	name := weapon.String()
	no := false
	sample := models.ReplayPlayerState{SteamID: 7, Alive: true, ObservedWeaponState: models.ReplayWeaponStateObservation{ActiveWeapon: &name, ActiveWeaponStatus: "observed", ActiveWeaponAvailabilityTick: &tick, AmmoInMagazine: &zero, AmmoInMagazineStatus: "observed", AmmoInMagazineAvailabilityTick: &tick, AmmoReserve: &zero, AmmoReserveStatus: "observed", AmmoReserveAvailabilityTick: &tick, IsReloading: &no, IsReloadingStatus: "observed", IsReloadingAvailabilityTick: &tick}}
	h := NewReplayHandler(&models.DemoContext{})
	h.reloadState.recordActiveSignal(7, true, playerstate.EquipmentID(weapon), playerstate.EquipmentID(weapon), false, tick, replayReloadSignalWeaponFire)
	return h, player, pawn, w, base, sample
}

func TestSupportPhysicalNativeZerosAndMissingTypes(t *testing.T) {
	for _, mode := range []string{"observed", "missing_angles", "bad_angles", "nonfinite", "walking_integer", "flags_missing", "clip_zero", "clip_negative", "reserve_negative", "reload_uninitialized", "stale_ammo"} {
		t.Run(mode, func(t *testing.T) {
			h, p, pawn, w, base, sample := physicalFixture()
			field := "view_angles"
			switch mode {
			case "missing_angles":
				delete(pawn.values, "m_angEyeAngles")
			case "bad_angles":
				pawn.values["m_angEyeAngles"] = []float32{1}
			case "nonfinite":
				pawn.values["m_angEyeAngles"] = [3]float32{0, float32(math.Inf(1)), 0}
			case "walking_integer":
				pawn.values["m_bIsWalking"] = uint64(0)
				field = "walking"
			case "flags_missing":
				delete(pawn.values, "m_fFlags")
				field = "ducking"
			case "clip_zero":
				w.values["m_iClip1"] = uint32(0)
				field = "ammo_in_magazine"
			case "clip_negative":
				w.values["m_iClip1"] = int32(-1)
				field = "ammo_in_magazine"
			case "reserve_negative":
				w.values["m_pReserveAmmo.0000"] = int32(-1)
				field = "ammo_reserve"
			case "reload_uninitialized":
				h.reloadState.invalidatePlayer(7)
				field = "reloading"
			case "stale_ammo":
				past := 99
				sample.ObservedWeaponState.AmmoInMagazineAvailabilityTick = &past
				field = "ammo_in_magazine"
			}
			r := h.supportPhysical(p, base, sample, 100).Context
			if mode != "observed" {
				if r.Facts[field].Status != "unavailable" || r.Facts[field].Value != nil {
					t.Fatal("missing or invalid signal became observed", field)
				}
				return
			}
			if r.Facts["view_angles"].Value != ([2]float64{20, 10}) || r.Facts["walking"].Value != false || r.Facts["ducking"].Value != false || r.Facts["ammo_in_magazine"].Value != 0 || r.Facts["ammo_reserve"].Value != 0 || r.Facts["reloading"].Value != false {
				t.Fatal("zero/false/native angle lost")
			}
		})
	}
}

func TestSupportPhysicalFullIdentityDestructionAndReuse(t *testing.T) {
	h, p, pawn, w, base, sample := physicalFixture()
	first := h.supportPhysical(p, base, sample, 100).Context
	pawn.values["m_pWeaponServices.m_hActiveWeapon"] = uint64(1026<<14 | 10)
	if r := h.supportPhysical(p, base, sample, 100).Context; r.WeaponLife != nil || r.Facts["active_weapon"].Value != nil {
		t.Fatal("full serial mismatch accepted")
	}
	pawn.values["m_pWeaponServices.m_hActiveWeapon"] = uint64(2<<14 | 10)
	w.destroy()
	if r := h.supportPhysical(p, base, sample, 100).Context; r.WeaponIdentityStatus != "destroyed" {
		t.Fatal("destroyed entity reused")
	}
	replacement := &cashflowEntity{ownershipEntity: w.ownershipEntity}
	p.ActiveWeapon().Entity = replacement
	r := h.supportPhysical(p, base, sample, 100).Context
	if r.WeaponLife == nil || *r.WeaponLife == *first.WeaponLife || r.Facts["reloading"].Status != "unavailable" || *r.Facts["reloading"].Reason != "weapon_handle_reused" {
		t.Fatal("new object inherited reload identity")
	}
	if first.Facts["reloading"].Status != "observed" {
		t.Fatal("past snapshot changed")
	}
}

func TestSupportPhysicalNoAmmoAndUnavailableActors(t *testing.T) {
	for _, weaponType := range []common.EquipmentType{common.EqKnife, common.EqHE, common.EqBomb} {
		h, p, _, _, base, sample := physicalFixture()
		p.ActiveWeapon().Type = weaponType
		name := p.ActiveWeapon().String()
		sample.ObservedWeaponState.ActiveWeapon = &name
		r := h.supportPhysical(p, base, sample, 100).Context
		for _, field := range []string{"ammo_in_magazine", "ammo_reserve", "reloading"} {
			if r.Facts[field].Status != "not_applicable" || r.Facts[field].Value != nil {
				t.Fatal("nonammo fabricated", field)
			}
		}
	}
	for _, mode := range []string{"dead", "disconnected", "identity", "position", "side"} {
		h, p, _, _, base, sample := physicalFixture()
		switch mode {
		case "dead":
			no := false
			base.Alive = &no
		case "disconnected":
			base.Connected = false
		case "identity":
			base.IdentityStatus = "unavailable"
		case "position":
			base.Position = nil
		case "side":
			base.Side = "unknown"
		}
		if h.supportPhysical(p, base, sample, 100).Context != nil {
			t.Fatal("unavailable actor inherited context", mode)
		}
	}
}
