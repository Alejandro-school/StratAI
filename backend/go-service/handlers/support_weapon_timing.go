package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var supportTimingPawnFields = []string{"m_pWeaponServices.m_hActiveWeapon", "m_pWeaponServices.m_flNextAttack", "m_flSimulationTime", "m_nTickBase"}
var supportTimingWeaponFields = []string{"m_hOwnerEntity", "m_nNextPrimaryAttackTick", "m_flNextPrimaryAttackTickRatio", "m_nNextSecondaryAttackTick", "m_flNextSecondaryAttackTickRatio", "m_nPostponeFireReadyTicks", "m_flPostponeFireReadyFrac"}

type supportTimingLife struct {
	life, class, kind string
	createdFrame      int
	alive             bool
	names             []string
	previous          map[string]models.SupportClockScalar
	sequences         map[string]int
	fingerprint       string
	stateSequence     int
}
type supportTimingTracker struct {
	objects map[st.Entity]*supportTimingLife
}

func (h *ReplayHandler) registerSupportWeaponTiming() {
	if h.ctx.SupportWeaponTimingObservations == nil {
		return
	}
	h.supportTiming = &supportTimingTracker{objects: map[st.Entity]*supportTimingLife{}}
	h.ctx.Parser.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, class := range h.ctx.Parser.ServerClasses().All() {
			name := class.Name()
			if name == "CCSPlayerPawn" || h.supportObjectiveClock.declared(name, "m_nNextPrimaryAttackTick") != "" {
				class.OnEntityCreated(func(e st.Entity) { h.bindSupportTiming(e) })
			}
		}
	})
}

func (h *ReplayHandler) timingDelivery(life *supportTimingLife, kind, name string, raw any) {
	data := h.ctx.SupportWeaponTimingObservations
	declared := h.supportObjectiveClock.declared(life.class, name)
	value := clockScalar(raw, declared)
	previous, ok := life.previous[name]
	if !ok {
		previous = clockScalar(nil, declared)
	}
	d := models.SupportObjectivePositionDelivery{Sequence: len(data.Deliveries) + 1, Frame: h.attemptNextFrame(), Tick: h.ctx.Parser.GameState().IngameTick(), Life: life.life, Kind: kind, Property: name, Previous: previous, Value: value}
	data.Deliveries = append(data.Deliveries, d)
	if name != "" {
		life.previous[name], life.sequences[name] = value, d.Sequence
	}
}

func (h *ReplayHandler) bindSupportTiming(e st.Entity) {
	if h.supportTiming == nil {
		return
	}
	name, kind, fields := e.ServerClass().Name(), "weapon", supportTimingWeaponFields
	if name == "CCSPlayerPawn" {
		kind, fields = "pawn", supportTimingPawnFields
	}
	life := &supportTimingLife{life: fmt.Sprintf("support-timing-entity-%06d", len(h.supportTiming.objects)+1), class: name, kind: kind, names: fields, createdFrame: h.attemptNextFrame(), alive: true, previous: map[string]models.SupportClockScalar{}, sequences: map[string]int{}}
	h.supportTiming.objects[e] = life
	for _, field := range fields {
		h.timingDelivery(life, "created", field, h.positionValue(e, name, field))
		if prop := h.positionProperty(e, name, field); prop != nil {
			prop.OnUpdate(func(v st.PropertyValue) {
				kind := "update"
				if h.attemptNextFrame() == life.createdFrame {
					kind = "creation_replay"
				}
				h.timingDelivery(life, kind, field, v.Any)
			})
		}
	}
	e.OnDestroy(func() { h.timingDelivery(life, "destroyed", "", nil); life.alive = false })
}

func (h *ReplayHandler) timingReference(raw any, current map[int]st.Entity) models.SupportC4Reference {
	ref, e := c4Reference(raw, current)
	if e != nil {
		life := h.supportTiming.objects[e]
		if life != nil && life.alive {
			ref.TargetLife = &life.life
		} else {
			ref.Status = "kind_or_life_unavailable"
		}
	}
	return ref
}

func (h *ReplayHandler) timingState(e st.Entity, base models.SupportFrame, current map[int]st.Entity) int {
	life := h.supportTiming.objects[e]
	s := models.SupportWeaponTimingState{Instance: base.Instance, Epoch: base.Epoch, Life: life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), ClassName: life.class, Kind: life.kind, CreatedFrame: life.createdFrame, Fields: map[string]models.SupportClockScalar{}, FieldSequences: map[string]int{}}
	for _, name := range life.names {
		s.Fields[name] = clockScalar(h.positionValue(e, life.class, name), h.supportObjectiveClock.declared(life.class, name))
		s.FieldSequences[name] = life.sequences[name]
	}
	refName := "m_hOwnerEntity"
	if life.kind == "pawn" {
		refName = "m_pWeaponServices.m_hActiveWeapon"
	}
	s.Reference = h.timingReference(h.positionValue(e, life.class, refName), current)
	raw, _ := json.Marshal(s)
	if string(raw) != life.fingerprint {
		data := h.ctx.SupportWeaponTimingObservations
		s.Sequence, s.FirstCut = len(data.States)+1, base.Sequence
		data.States = append(data.States, s)
		life.fingerprint, life.stateSequence = string(raw), s.Sequence
	}
	return life.stateSequence
}

func (h *ReplayHandler) captureSupportWeaponTiming(base models.SupportFrame, players map[uint64]*common.Player, physical models.SupportPhysicalFrame) {
	if h.supportTiming == nil {
		return
	}
	data, current := h.ctx.SupportWeaponTimingObservations, h.ctx.Parser.GameState().Entities()
	cut := models.SupportWeaponTimingCut{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch, ClockOrder: h.supportObjectiveClock.order, DeliverySequence: len(data.Deliveries), Players: []models.SupportWeaponTimingPlayer{}}
	for i, actor := range base.Players {
		row := models.SupportWeaponTimingPlayer{NativeActor: actor.NativeActor, WeaponStatus: "current_pawn_unavailable"}
		p := players[actor.NativeActor]
		proof := objectivePlayer(p, actor)
		if proof.PawnLife != nil {
			e := p.PlayerPawnEntity()
			life := h.supportTiming.objects[e]
			if life != nil && life.alive && current[e.ID()] == e && h.support.lives[e] == *proof.PawnLife {
				seq := h.timingState(e, base, current)
				row.PawnLife, row.PawnHandle, row.PawnStateSequence = proof.PawnLife, actor.PawnHandle, &seq
				row.WeaponStatus = "current_weapon_unavailable"
				state := data.States[seq-1]
				ref := state.Reference
				if ref.Status == "resolved" && ref.Handle != nil {
					weapon := current[int(*ref.Handle&16383)]
					wl := h.supportTiming.objects[weapon]
					if wl != nil && wl.alive && wl.kind == "weapon" {
						ws := h.timingState(weapon, base, current)
						row.WeaponStateSequence = &ws
						owner := data.States[ws-1].Reference
						physicalState := physical.Players[i].Context
						if owner.Status == "resolved" && owner.TargetLife != nil && *owner.TargetLife == life.life && owner.Handle != nil && *owner.Handle == *actor.PawnHandle {
							row.WeaponStatus = "physical_weapon_identity_unavailable"
							if physical.Players[i].NativeActor == actor.NativeActor && physicalState != nil && physicalState.WeaponIdentityStatus == "observed" && physicalState.WeaponHandle != nil && *physicalState.WeaponHandle == *ref.Handle && physicalState.WeaponLife != nil && h.supportPhysicalState != nil && h.supportPhysicalState.lives[weapon] == *physicalState.WeaponLife && !h.supportPhysicalState.dead[weapon] && physicalState.Facts["active_weapon"].Status == "observed" {
								row.WeaponStatus, row.WeaponLife = "observed", physicalState.WeaponLife
							}
						} else {
							row.WeaponStatus = "current_weapon_owner_unaccredited"
						}
					}
				}
			}
		}
		cut.Players = append(cut.Players, row)
	}
	cut.StateSequence = len(data.States)
	data.Cuts = append(data.Cuts, cut)
}
