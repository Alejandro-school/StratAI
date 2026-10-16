package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var supportNativeMotionFields = []string{
	"m_vecVelocity", "m_vecAbsVelocity", "m_vecBaseVelocity",
	"m_vecVelocity.m_vecX", "m_vecVelocity.m_vecY", "m_vecVelocity.m_vecZ",
	"m_vecAbsVelocity.m_vecX", "m_vecAbsVelocity.m_vecY", "m_vecAbsVelocity.m_vecZ",
	"m_vecVelocity[0]", "m_vecVelocity[1]", "m_vecVelocity[2]",
	"m_fFlags", "m_hGroundEntity", "m_nGroundBodyIndex", "m_MoveType", "m_nActualMoveType", "m_MoveCollide",
	"CBodyComponent.m_hParent", "m_hMoveParent", "m_iParentAttachment",
	"m_flSimulationTime", "m_nTickBase",
	"m_pMovementServices.m_flMaxspeed", "m_pMovementServices.m_flFallVelocity",
	"m_pMovementServices.m_nLastLandedTick", "m_pMovementServices.m_flLastLandedFrac",
	"m_pMovementServices.m_flLastLandedVelocityX", "m_pMovementServices.m_flLastLandedVelocityY", "m_pMovementServices.m_flLastLandedVelocityZ",
	"m_pMovementServices.m_nLastJumpTick", "m_pMovementServices.m_flLastJumpFrac", "m_pMovementServices.m_flLastJumpVelocityZ",
}

type supportMotionLife struct {
	life          string
	createdFrame  int
	alive         bool
	previous      map[string]models.SupportClockScalar
	sequences     map[string]int
	fingerprint   string
	stateSequence int
}

type supportMotionReferenceLife struct {
	life  string
	alive bool
}
type supportMotionTracker struct {
	objects    map[st.Entity]*supportMotionLife
	references map[st.Entity]*supportMotionReferenceLife
}

func (h *ReplayHandler) registerSupportNativeMotion() {
	if h.ctx.SupportNativeMotionObservations == nil {
		return
	}
	h.supportMotion = &supportMotionTracker{objects: map[st.Entity]*supportMotionLife{}, references: map[st.Entity]*supportMotionReferenceLife{}}
	h.ctx.Parser.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, class := range h.ctx.Parser.ServerClasses().All() {
			if class.Name() == "CCSPlayerPawn" {
				class.OnEntityCreated(func(e st.Entity) { h.bindSupportMotion(e) })
			}
		}
	})
}

func (h *ReplayHandler) motionDelivery(life *supportMotionLife, kind, name string, raw any) {
	data := h.ctx.SupportNativeMotionObservations
	declared := h.supportObjectiveClock.declared("CCSPlayerPawn", name)
	value := positionScalar(raw, declared)
	previous, ok := life.previous[name]
	if !ok {
		previous = positionScalar(nil, declared)
	}
	d := models.SupportObjectivePositionDelivery{Sequence: len(data.Deliveries) + 1, Frame: h.attemptNextFrame(), Tick: h.ctx.Parser.GameState().IngameTick(), Life: life.life, Kind: kind, Property: name, Previous: previous, Value: value}
	data.Deliveries = append(data.Deliveries, d)
	if name != "" {
		life.previous[name], life.sequences[name] = value, d.Sequence
	}
}

func (h *ReplayHandler) bindSupportMotion(e st.Entity) {
	if h.supportMotion == nil {
		return
	}
	life := &supportMotionLife{life: fmt.Sprintf("support-motion-pawn-%06d", len(h.supportMotion.objects)+1), createdFrame: h.attemptNextFrame(), alive: true, previous: map[string]models.SupportClockScalar{}, sequences: map[string]int{}}
	h.supportMotion.objects[e] = life
	for _, name := range supportNativeMotionFields {
		h.motionDelivery(life, "created", name, h.positionValue(e, "CCSPlayerPawn", name))
		if p := h.positionProperty(e, "CCSPlayerPawn", name); p != nil {
			p.OnUpdate(func(v st.PropertyValue) {
				kind := "update"
				if h.attemptNextFrame() == life.createdFrame {
					kind = "creation_replay"
				}
				h.motionDelivery(life, kind, name, v.Any)
			})
		}
	}
	e.OnDestroy(func() { h.motionDelivery(life, "destroyed", "", nil); life.alive = false })
}

func (h *ReplayHandler) motionReference(raw any, current map[int]st.Entity) models.SupportNativeMotionReference {
	ref, target := c4Reference(raw, current)
	r := models.SupportNativeMotionReference{SupportC4Reference: ref}
	if target == nil {
		return r
	}
	life := h.supportMotion.references[target]
	if life == nil {
		life = &supportMotionReferenceLife{life: fmt.Sprintf("support-motion-reference-%06d", len(h.supportMotion.references)+1), alive: true}
		h.supportMotion.references[target] = life
		target.OnDestroy(func() { life.alive = false })
	}
	if !life.alive {
		r.Status = "entity_unavailable"
		return r
	}
	class := target.ServerClass().Name()
	r.TargetLife, r.ClassName = &life.life, &class
	return r
}

func (h *ReplayHandler) motionState(e st.Entity, life *supportMotionLife, base models.SupportFrame, current map[int]st.Entity) models.SupportNativeMotionState {
	r := models.SupportNativeMotionState{Instance: base.Instance, Epoch: base.Epoch, Life: life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), ClassName: "CCSPlayerPawn", CreatedFrame: life.createdFrame, Fields: map[string]models.SupportClockScalar{}, FieldSequences: map[string]int{}}
	for _, name := range supportNativeMotionFields {
		r.Fields[name] = positionScalar(h.positionValue(e, "CCSPlayerPawn", name), h.supportObjectiveClock.declared("CCSPlayerPawn", name))
		r.FieldSequences[name] = life.sequences[name]
	}
	r.Ground = h.motionReference(h.positionValue(e, "CCSPlayerPawn", "m_hGroundEntity"), current)
	r.Parent = h.motionReference(h.positionValue(e, "CCSPlayerPawn", "CBodyComponent.m_hParent"), current)
	return r
}

func (h *ReplayHandler) captureSupportNativeMotion(base models.SupportFrame, players map[uint64]*common.Player) {
	if h.supportMotion == nil {
		return
	}
	data := h.ctx.SupportNativeMotionObservations
	current := h.ctx.Parser.GameState().Entities()
	cut := models.SupportNativeMotionCut{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch, ClockOrder: h.supportObjectiveClock.order, DeliverySequence: len(data.Deliveries), Players: []models.SupportNativeMotionPlayer{}}
	for _, actor := range base.Players {
		player := models.SupportNativeMotionPlayer{NativeActor: actor.NativeActor}
		p := players[actor.NativeActor]
		proof := objectivePlayer(p, actor)
		if proof.PawnLife != nil {
			e := p.PlayerPawnEntity()
			life := h.supportMotion.objects[e]
			if life != nil && life.alive && current[e.ID()] == e && h.support.lives[e] == *proof.PawnLife {
				state := h.motionState(e, life, base, current)
				raw, _ := json.Marshal(state)
				if string(raw) != life.fingerprint {
					state.Sequence, state.FirstCut = len(data.States)+1, base.Sequence
					data.States = append(data.States, state)
					life.fingerprint = string(raw)
					life.stateSequence = state.Sequence
				}
				seq := life.stateSequence
				player.PawnLife, player.PawnHandle, player.StateSequence = proof.PawnLife, actor.PawnHandle, &seq
			}
		}
		cut.Players = append(cut.Players, player)
	}
	cut.StateSequence = len(data.States)
	data.Cuts = append(data.Cuts, cut)
}
