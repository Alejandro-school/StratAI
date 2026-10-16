package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"fmt"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"math"
	"strings"
)

type supportPhysicalTracker struct {
	lives   map[st.Entity]string
	dead    map[st.Entity]bool
	handles map[uint64]st.Entity
	reused  map[uint64]bool
}

func supportFact(value any) models.SupportPhysicalFact {
	return models.SupportPhysicalFact{Status: "observed", Value: value}
}

func supportMissing(reason string) models.SupportPhysicalFact {
	return models.SupportPhysicalFact{Status: "unavailable", Reason: &reason}
}

func supportProperty(entity st.Entity, name string) any {
	if entity == nil {
		return nil
	}
	if p := entity.Property(name); p != nil {
		return p.Value().Any
	}
	return nil
}

func supportView(value any) models.SupportPhysicalFact {
	if value == nil {
		return supportMissing("native_fields_absent")
	}
	var v []float32
	switch raw := value.(type) {
	case [3]float32:
		v = raw[:]
	case []float32:
		if len(raw) == 3 {
			v = raw
		}
	}
	if len(v) != 3 {
		return supportMissing("invalid_native_type")
	}
	for _, n := range v {
		if math.IsNaN(float64(n)) || math.IsInf(float64(n), 0) {
			return supportMissing("nonfinite")
		}
	}
	return supportFact([2]float64{float64(v[1]), float64(v[0])}) // yaw, pitch, matching SDK
}

func supportBoolean(value any) models.SupportPhysicalFact {
	if value == nil {
		return supportMissing("native_fields_absent")
	}
	if b, ok := value.(bool); ok {
		return supportFact(b)
	}
	return supportMissing("invalid_native_type")
}

func (h *ReplayHandler) supportPhysical(p *common.Player, base models.SupportPlayer, sample models.ReplayPlayerState, tick int) models.SupportPhysicalPlayer {
	r := models.SupportPhysicalPlayer{NativeActor: base.NativeActor}
	if p == nil || base.IdentityStatus != "observed" || !base.Connected || base.Alive == nil || !*base.Alive || base.Position == nil || (base.Side != "CT" && base.Side != "T") {
		return r
	}
	pawn := p.PlayerPawnEntity()
	facts := map[string]models.SupportPhysicalFact{}
	for _, k := range []string{"view_angles", "walking", "ducking", "active_weapon", "ammo_in_magazine", "ammo_reserve", "reloading"} {
		facts[k] = supportMissing("weapon_identity_unavailable")
	}
	state := &models.SupportPhysicalState{WeaponIdentityStatus: "unavailable", Facts: facts}
	r.Context = state
	facts["view_angles"] = supportView(supportProperty(pawn, "m_angEyeAngles"))
	facts["walking"] = supportBoolean(supportProperty(pawn, "m_bIsWalking"))
	flags := supportProperty(pawn, "m_fFlags")
	facts["ducking"] = supportMissing("native_fields_absent")
	if flags != nil {
		if raw, ok := flags.(uint64); ok {
			facts["ducking"] = supportFact(common.PlayerFlags(raw).Ducking())
		} else {
			facts["ducking"] = supportMissing("invalid_native_type")
		}
	}
	state.WeaponHandle = supportHandle(supportProperty(pawn, "m_pWeaponServices.m_hActiveWeapon"))
	if state.WeaponHandle == nil {
		return r
	}
	handle := *state.WeaponHandle
	if handle&16383 == 16383 {
		state.WeaponIdentityStatus = "unresolved"
		return r
	}
	weapon := p.ActiveWeapon()
	if weapon == nil || weapon.Entity == nil || weapon.Entity.ID() != int(handle&16383) || weapon.Entity.SerialNum() != int(handle>>14) {
		state.WeaponIdentityStatus = "unresolved"
		return r
	}
	if h.supportPhysicalState == nil {
		h.supportPhysicalState = &supportPhysicalTracker{lives: map[st.Entity]string{}, dead: map[st.Entity]bool{}, handles: map[uint64]st.Entity{}, reused: map[uint64]bool{}}
	}
	t := h.supportPhysicalState
	entity := weapon.Entity
	if t.dead[entity] {
		state.WeaponIdentityStatus = "destroyed"
		return r
	}
	life, exists := t.lives[entity]
	if !exists {
		life = fmt.Sprintf("support-weapon-%06d", len(t.lives)+1)
		t.lives[entity] = life
		entity.OnDestroy(func() { t.dead[entity] = true })
	}
	if prev, ok := t.handles[handle]; ok && prev != entity {
		t.reused[handle] = true
	}
	t.handles[handle] = entity
	state.WeaponLife = &life
	state.WeaponIdentityStatus = "observed"
	obs := sample.ObservedWeaponState
	if obs.ActiveWeapon == nil || obs.ActiveWeaponStatus != models.ObservedWeaponFactObserved || obs.ActiveWeaponAvailabilityTick == nil || *obs.ActiveWeaponAvailabilityTick != tick || *obs.ActiveWeapon != strings.TrimSpace(weapon.String()) {
		facts["active_weapon"] = supportMissing("current_weapon_observation_unavailable")
		return r
	}
	facts["active_weapon"] = supportFact(*obs.ActiveWeapon)
	if weapon.Class() == common.EqClassUnknown {
		for _, k := range []string{"ammo_in_magazine", "ammo_reserve", "reloading"} {
			facts[k] = supportMissing("unknown_weapon_class")
		}
		return r
	}
	if !weaponClassUsesAmmunition(weapon.Class()) {
		for _, k := range []string{"ammo_in_magazine", "ammo_reserve", "reloading"} {
			reason := "non_ammunition_weapon"
			facts[k] = models.SupportPhysicalFact{Status: "not_applicable", Reason: &reason}
		}
		return r
	}
	// The pinned SDK exposes encoded uint32 clip+1; retain it for recomputation.
	facts["ammo_in_magazine"] = supportMissing("native_fields_absent_or_invalid")
	if raw, ok := supportProperty(entity, "m_iClip1").(uint32); ok {
		state.RawClip = &raw
		if raw > 0 && obs.AmmoInMagazineStatus == models.ObservedWeaponFactObserved && obs.AmmoInMagazine != nil && obs.AmmoInMagazineAvailabilityTick != nil && *obs.AmmoInMagazineAvailabilityTick == tick && *obs.AmmoInMagazine == int(raw-1) {
			facts["ammo_in_magazine"] = supportFact(*obs.AmmoInMagazine)
		}
	}
	facts["ammo_reserve"] = supportMissing("native_fields_absent_or_invalid")
	for _, name := range []string{"m_pReserveAmmo.0000", "m_iPrimaryReserveAmmoCount"} {
		value := supportProperty(entity, name)
		if value == nil {
			continue
		}
		state.ReserveSource = &name
		if raw, ok := value.(int32); ok {
			state.RawReserve = &raw
			if raw >= 0 && obs.AmmoReserveStatus == models.ObservedWeaponFactObserved && obs.AmmoReserve != nil && obs.AmmoReserveAvailabilityTick != nil && *obs.AmmoReserveAvailabilityTick == tick && *obs.AmmoReserve == int(raw) {
				facts["ammo_reserve"] = supportFact(*obs.AmmoReserve)
			}
		}
		break
	}
	facts["reloading"] = supportMissing("reload_signal_unavailable")
	reload, ok := h.reloadState.observations[p.SteamID64]
	if t.reused[handle] {
		facts["reloading"] = supportMissing("weapon_handle_reused")
		return r
	}
	if ok && reload.available && reload.signalTick <= tick && reload.weaponID == playerstate.EquipmentID(weapon) && reload.availabilityTick == tick && obs.IsReloadingStatus == models.ObservedWeaponFactObserved && obs.IsReloading != nil && obs.IsReloadingAvailabilityTick != nil && *obs.IsReloadingAvailabilityTick == tick && *obs.IsReloading == reload.value {
		facts["reloading"] = supportFact(*obs.IsReloading)
		signal := string(reload.signal)
		state.ReloadSignal = &signal
		state.ReloadSignalTick = &reload.signalTick
	}
	return r
}
