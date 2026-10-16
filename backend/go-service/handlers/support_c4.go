package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type supportC4Life struct {
	life, class string
	alive       bool
}
type supportC4Tracker struct {
	objects              map[st.Entity]*supportC4Life
	fingerprint          string
	inventoryFingerprint string
}

func (t *supportC4Tracker) created(e st.Entity, class string) {
	life := &supportC4Life{life: fmt.Sprintf("support-item-%06d", len(t.objects)+1), class: class, alive: true}
	t.objects[e] = life
	e.OnDestroy(func() { life.alive = false })
}

func (h *ReplayHandler) registerSupportC4() {
	if h.ctx.SupportC4Observations == nil {
		return
	}
	t := &supportC4Tracker{objects: map[st.Entity]*supportC4Life{}}
	h.supportC4State = t
	h.ctx.Parser.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, class := range h.ctx.Parser.ServerClasses().All() {
			index, clip := false, false
			for _, name := range class.PropertyEntries() {
				index = index || name == "m_iItemDefinitionIndex"
				clip = clip || name == "m_iClip1"
			}
			if class.Name() == "CC4" || (index && clip) {
				class.OnEntityCreated(func(e st.Entity) {
					t.created(e, class.Name())
					if class.Name() == "CC4" {
						h.bindSupportAttempt(e, t.objects[e])
						h.bindSupportPosition(e, t.objects[e])
					}
				})
			}
		}
	})
}

func c4Reference(raw any, current map[int]st.Entity) (models.SupportC4Reference, st.Entity) {
	r := models.SupportC4Reference{Status: "absent"}
	if raw == nil {
		return r, nil
	}
	r.Handle = supportHandle(raw)
	if r.Handle == nil {
		r.Status = "unsupported_type"
		return r, nil
	}
	if *r.Handle&16383 == 16383 {
		r.Status = "invalid_sentinel"
		return r, nil
	}
	e := current[int(*r.Handle&16383)]
	if e == nil {
		r.Status = "entity_unavailable"
		return r, nil
	}
	if e.ID() != int(*r.Handle&16383) || e.SerialNum() != int(*r.Handle>>14) {
		r.Status = "serial_mismatch"
		return r, nil
	}
	r.Status = "resolved"
	return r, e
}

func (t *supportC4Tracker) itemReference(raw any, current map[int]st.Entity) models.SupportC4Reference {
	r, e := c4Reference(raw, current)
	if e == nil {
		return r
	}
	life := t.objects[e]
	if life == nil {
		r.Status = "kind_or_life_unavailable"
		return r
	}
	if !life.alive {
		r.Status = "destroyed"
		return r
	}
	r.TargetLife = &life.life
	return r
}

func (h *ReplayHandler) captureSupportC4(base models.SupportFrame, players map[uint64]*common.Player) {
	data := h.ctx.SupportC4Observations
	if data == nil {
		return
	}
	t := h.supportC4State
	current := h.ctx.Parser.GameState().Entities()
	state := models.SupportC4State{Players: []models.SupportC4Player{}, Entities: []models.SupportC4Entity{}}
	needed := map[st.Entity]bool{}
	// Only native CC4 objects are inspected outside participant inventories. No
	// bomb position, cached Carrier, equipment.Owner or SDK Inventory is used.
	for e, life := range t.objects {
		if life.alive && life.class == "CC4" && current[e.ID()] == e {
			needed[e] = true
		}
	}
	for _, actor := range base.Players {
		r := models.SupportC4Player{NativeActor: actor.NativeActor, InventoryStatus: "pawn_unavailable", Slots: []models.SupportC4Reference{}}
		p := players[actor.NativeActor]
		proof := objectivePlayer(p, actor)
		if proof.PawnLife != nil {
			pawn := p.PlayerPawnEntity()
			if current[pawn.ID()] == pawn && h.support.lives[pawn] == *proof.PawnLife {
				r.PawnLife, r.PawnHandle = proof.PawnLife, actor.PawnHandle
				r.InventoryStatus = "absent"
				if raw := supportProperty(pawn, identityInventory); raw != nil {
					r.InventoryStatus = "unsupported_type"
					if array, ok := raw.([]any); ok {
						length := len(array)
						r.InventoryLength = &length
						r.InventoryStatus = "outside_probe"
						if length <= 64 {
							r.InventoryStatus = "observed"
							for slot := 0; slot < length; slot++ {
								ref := t.itemReference(supportProperty(pawn, fmt.Sprintf("%s.%04d", identityInventory, slot)), current)
								r.Slots = append(r.Slots, ref)
								if ref.Status == "resolved" {
									needed[current[int(*ref.Handle&16383)]] = true
								}
							}
						}
					}
				}
			}
		}
		state.Players = append(state.Players, r)
	}
	for e := range needed {
		life := t.objects[e]
		owner, pawn := c4Reference(supportProperty(e, "m_hOwnerEntity"), current)
		if pawn != nil {
			// Pawn aliases come from this exact support identity tracker. An
			// unobserved/destroyed pawn cannot satisfy the ownership join.
			if alias := h.support.lives[pawn]; alias != "" {
				owner.TargetLife = &alias
			} else {
				owner.Status = "kind_or_life_unavailable"
			}
		}
		state.Entities = append(state.Entities, models.SupportC4Entity{Life: life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), ClassName: life.class, Definition: entryScalar(e, "m_iItemDefinitionIndex"), Owner: owner})
	}
	sort.Slice(state.Entities, func(i, j int) bool { return state.Entities[i].Life < state.Entities[j].Life })
	// NativeActor is not serialized, so include it in the in-memory fingerprint
	// until the exporter assigns opaque aliases. Every frame is freshly read.
	actors := make([]uint64, len(state.Players))
	for i, p := range state.Players {
		actors[i] = p.NativeActor
	}
	raw, _ := json.Marshal(struct {
		State  models.SupportC4State
		Actors []uint64
	}{state, actors})
	cut := models.SupportC4Cut{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch, ObjectiveOrder: h.supportObjectiveOrder}
	changed := len(data.Cuts) == 0 || string(raw) != t.fingerprint
	if len(data.Cuts) > 0 {
		prev := data.Cuts[len(data.Cuts)-1]
		changed = changed || prev.Instance != cut.Instance || prev.Epoch != cut.Epoch
	}
	if changed {
		state.Sequence, state.FirstCut = len(data.States)+1, cut.SourceSequence
		data.States = append(data.States, state)
		t.fingerprint = string(raw)
	}
	cut.StateSequence = len(data.States)
	data.Cuts = append(data.Cuts, cut)
	h.captureSupportInventory(base, state, current, cut.StateSequence)
}
