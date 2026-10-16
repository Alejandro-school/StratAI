package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"math"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func timingFixture() (*ReplayHandler, *common.Player, *motionTestEntity, *motionTestEntity, *attemptTestState, models.SupportFrame, models.SupportPhysicalFrame) {
	h, p, pawn, gs, base := motionFixture()
	weapon := &motionTestEntity{attemptTestEntity: &attemptTestEntity{ownershipEntity: ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{}}, index: 9, serial: 4}, updates: map[string][]st.PropertyUpdateHandler{}}, class: "CAK47"}
	wh := uint64(4<<14 | 9)
	ph := *base.Players[0].PawnHandle
	pawn.values["m_pWeaponServices.m_hActiveWeapon"] = wh
	pawn.values["m_pWeaponServices.m_flNextAttack"] = float32(0)
	weapon.values["m_hOwnerEntity"] = ph
	weapon.values["m_nNextPrimaryAttackTick"] = uint64(0)
	weapon.values["m_flNextPrimaryAttackTickRatio"] = float32(0)
	gs.current[9] = weapon
	h.supportObjectiveClock.types["CCSPlayerPawn"]["m_pWeaponServices"] = supportClockField{"CCSPlayer_WeaponServices", "Services"}
	h.supportObjectiveClock.types["Services"] = map[string]supportClockField{"m_hActiveWeapon": {"CHandle< CBasePlayerWeapon >", ""}, "m_flNextAttack": {"GameTime_t", ""}}
	h.supportObjectiveClock.types["CAK47"] = map[string]supportClockField{}
	for _, name := range supportTimingWeaponFields {
		typ := "GameTick_t"
		if name == "m_hOwnerEntity" {
			typ = "CHandle< CBaseEntity >"
		}
		if name == "m_flNextPrimaryAttackTickRatio" || name == "m_flNextSecondaryAttackTickRatio" || name == "m_flPostponeFireReadyFrac" {
			typ = "float32"
		}
		h.supportObjectiveClock.types["CAK47"][name] = supportClockField{typ, ""}
	}
	h.ctx.SupportWeaponTimingObservations = &models.SupportWeaponTimingObservations{Deliveries: []models.SupportObjectivePositionDelivery{}, States: []models.SupportWeaponTimingState{}, Cuts: []models.SupportWeaponTimingCut{}}
	h.supportTiming = &supportTimingTracker{objects: map[st.Entity]*supportTimingLife{}}
	h.bindSupportTiming(pawn)
	h.bindSupportTiming(weapon)
	physicalLife := "support-weapon-000001"
	h.supportPhysicalState = &supportPhysicalTracker{lives: map[st.Entity]string{weapon: physicalLife}, dead: map[st.Entity]bool{}}
	physical := models.SupportPhysicalFrame{SourceSequence: base.Sequence, Players: []models.SupportPhysicalPlayer{{NativeActor: p.SteamID64, Context: &models.SupportPhysicalState{WeaponHandle: &wh, WeaponLife: &physicalLife, WeaponIdentityStatus: "observed", Facts: map[string]models.SupportPhysicalFact{"active_weapon": supportFact("AK-47")}}}}}
	return h, p, pawn, weapon, gs, base, physical
}

func TestSupportTimingNativeZeroReplayAndDistinctSameTickCuts(t *testing.T) {
	h, p, pawn, weapon, _, base, physical := timingFixture()
	data := h.ctx.SupportWeaponTimingObservations
	weapon.update("m_nNextPrimaryAttackTick", uint64(1234))
	if data.Deliveries[len(data.Deliveries)-1].Kind != "creation_replay" {
		t.Fatal("baseline replay lost")
	}
	h.support.frame = 100
	capture := func() {
		h.captureSupportWeaponTiming(base, map[uint64]*common.Player{p.SteamID64: p}, physical)
		base.Sequence++
		base.Frame++
	}
	capture()
	capture()
	if len(data.States) != 2 || data.Cuts[0].Players[0].WeaponStatus != "observed" || data.States[0].Fields["m_pWeaponServices.m_flNextAttack"].Value != float64(0) || data.States[1].Fields["m_nNextSecondaryAttackTick"].Status != "absent" {
		t.Fatal(data.States, data.Cuts)
	}
	before, _ := json.Marshal(data.States)
	pawn.update("m_pWeaponServices.m_flNextAttack", float32(12.5))
	h.supportObjectiveClock.order++
	capture()
	if len(data.States) != 3 || data.Deliveries[len(data.Deliveries)-1].Kind != "update" || data.Cuts[2].ClockOrder != 8 || data.Cuts[2].Tick != data.Cuts[0].Tick {
		t.Fatal("same tick update collapsed")
	}
	after, _ := json.Marshal(data.States[:2])
	if string(before) != string(after) {
		t.Fatal("prior state mutated")
	}
	base.Epoch++
	capture()
	base.Instance++
	capture()
	if len(data.States) != 7 {
		t.Fatal("state crossed domain", len(data.States))
	}
}

func TestSupportTimingOwnerSwitchDropAndCachedWeaponRejected(t *testing.T) {
	h, p, pawn, weapon, gs, base, physical := timingFixture()
	h.support.frame = 100
	capture := func() models.SupportWeaponTimingPlayer {
		h.captureSupportWeaponTiming(base, map[uint64]*common.Player{p.SteamID64: p}, physical)
		base.Sequence++
		base.Frame++
		return h.ctx.SupportWeaponTimingObservations.Cuts[len(h.ctx.SupportWeaponTimingObservations.Cuts)-1].Players[0]
	}
	weapon.update("m_hOwnerEntity", uint64(16777215))
	if capture().WeaponStatus != "current_weapon_owner_unaccredited" {
		t.Fatal("dropped owner accepted")
	}
	weapon.update("m_hOwnerEntity", *base.Players[0].PawnHandle)
	physical.Players[0].Context.WeaponHandle = new(uint64)
	if capture().WeaponStatus != "physical_weapon_identity_unavailable" {
		t.Fatal("prior weapon physical proof accepted")
	}
	delete(gs.current, weapon.ID())
	if capture().WeaponStateSequence != nil {
		t.Fatal("cached weapon accepted")
	}
	pawn.update("m_pWeaponServices.m_hActiveWeapon", uint64(16777215))
	if capture().WeaponStateSequence != nil {
		t.Fatal("sentinel active weapon accepted")
	}
}

func TestSupportTimingDestructionFullSerialAndReusedNativeHandle(t *testing.T) {
	h, p, pawn, weapon, gs, base, physical := timingFixture()
	h.support.frame = 100
	capture := func() models.SupportWeaponTimingPlayer {
		h.captureSupportWeaponTiming(base, map[uint64]*common.Player{p.SteamID64: p}, physical)
		base.Sequence++
		base.Frame++
		return h.ctx.SupportWeaponTimingObservations.Cuts[len(h.ctx.SupportWeaponTimingObservations.Cuts)-1].Players[0]
	}
	first := capture()
	oldLife := h.ctx.SupportWeaponTimingObservations.States[*first.WeaponStateSequence-1].Life
	for _, fn := range weapon.destroyers {
		fn()
	}
	if capture().WeaponStateSequence != nil {
		t.Fatal("destroyed weapon accepted")
	}
	other := &motionTestEntity{attemptTestEntity: &attemptTestEntity{ownershipEntity: weapon.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}, class: "CAK47"}
	other.serial++
	gs.current[9] = other
	h.bindSupportTiming(other)
	if capture().WeaponStateSequence != nil {
		t.Fatal("wrong serial accepted")
	}
	// A new native entity can reuse exactly the full handle; it still gets a new life.
	other.serial--
	other.values = map[string]any{"m_hOwnerEntity": *base.Players[0].PawnHandle}
	fresh := &motionTestEntity{attemptTestEntity: &attemptTestEntity{ownershipEntity: other.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}, class: "CAK47"}
	gs.current[9] = fresh
	h.bindSupportTiming(fresh)
	newPhysical := "support-weapon-000002"
	if capture().WeaponStatus != "physical_weapon_identity_unavailable" {
		t.Fatal("prior physical life accepted for a reused full handle")
	}
	physical.Players[0].Context.WeaponLife = &newPhysical
	h.supportPhysicalState.lives[fresh] = newPhysical
	last := capture()
	state := h.ctx.SupportWeaponTimingObservations.States[*last.WeaponStateSequence-1]
	if state.Life == oldLife || state.Fields["m_nNextPrimaryAttackTick"].Status != "absent" {
		t.Fatal("deadline inherited by new life")
	}
	for _, fn := range pawn.destroyers {
		fn()
	}
	if capture().PawnStateSequence != nil {
		t.Fatal("destroyed pawn accepted")
	}
}

func TestSupportTimingUnsupportedNonfiniteAndMissingAreDistinct(t *testing.T) {
	h, p, pawn, weapon, _, base, physical := timingFixture()
	h.support.frame = 100
	pawn.update("m_pWeaponServices.m_flNextAttack", float32(math.Inf(1)))
	weapon.update("m_flNextPrimaryAttackTickRatio", "invalid")
	h.captureSupportWeaponTiming(base, map[uint64]*common.Player{p.SteamID64: p}, physical)
	states := h.ctx.SupportWeaponTimingObservations.States
	if states[0].Fields["m_pWeaponServices.m_flNextAttack"].Status != "nonfinite" || states[1].Fields["m_flNextPrimaryAttackTickRatio"].Status != "unsupported_type" || states[1].Fields["m_nNextSecondaryAttackTick"].Status != "absent" {
		t.Fatal(states)
	}
	h = NewReplayHandler(&models.DemoContext{})
	h.registerSupportWeaponTiming()
	h.bindSupportTiming(nil)
	h.captureSupportWeaponTiming(models.SupportFrame{}, nil, models.SupportPhysicalFrame{})
	if h.supportTiming != nil {
		t.Fatal("enabled in default mode")
	}
}
