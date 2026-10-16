package handlers

import (
	"fmt"
	"sort"

	"cs2-demo-service/models"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var supportAttemptProperties = []string{"m_bStartedArming", "m_fArmedTime", "m_hOwnerEntity"}

type supportAttemptLife struct {
	life         *supportC4Life
	createdFrame int
	last         map[string]models.SupportClockScalar
}

type supportAttemptTracker struct {
	objects      map[st.Entity]*supportAttemptLife
	network      map[string][]*models.SupportClockMessage
	lastDelivery int
	wasActive    bool
}

func (h *ReplayHandler) attemptNextFrame() int {
	if h.support == nil {
		return 1
	}
	return h.support.frame + 1
}

func (h *ReplayHandler) registerSupportAttemptOrigin() {
	if h.ctx.SupportAttemptOriginObservations != nil {
		h.supportAttempt = &supportAttemptTracker{objects: map[st.Entity]*supportAttemptLife{}, network: map[string][]*models.SupportClockMessage{"tick": {}, "packet_tick": {}, "interval": {}}}
	}
}

func (h *ReplayHandler) attemptMessage(kind string, message *models.SupportClockMessage) {
	if h.supportAttempt != nil && (kind == "tick" || kind == "packet_tick" || kind == "interval") {
		h.supportAttempt.network[kind] = append(h.supportAttempt.network[kind], message)
	}
}

func (h *ReplayHandler) attemptDelivery(e st.Entity, life *supportAttemptLife, kind, property string, raw any) {
	data := h.ctx.SupportAttemptOriginObservations
	value := clockScalar(raw, h.supportObjectiveClock.declared("CC4", property))
	previous, exists := life.last[property]
	if !exists {
		previous = clockScalar(nil, h.supportObjectiveClock.declared("CC4", property))
	}
	data.Deliveries = append(data.Deliveries, models.SupportAttemptDelivery{Sequence: len(data.Deliveries) + 1, Frame: h.attemptNextFrame(), Tick: h.ctx.Parser.GameState().IngameTick(), Life: life.life.life, Kind: kind, Property: property, Previous: previous, Value: value})
	life.last[property] = value
}

func (h *ReplayHandler) bindSupportAttempt(e st.Entity, itemLife *supportC4Life) {
	if h.supportAttempt == nil {
		return
	}
	life := &supportAttemptLife{life: itemLife, createdFrame: h.attemptNextFrame(), last: map[string]models.SupportClockScalar{}}
	h.supportAttempt.objects[e] = life
	for _, name := range supportAttemptProperties {
		h.attemptDelivery(e, life, "created", name, supportProperty(e, name))
		if property := e.Property(name); property != nil {
			property.OnUpdate(func(value st.PropertyValue) {
				kind := "update"
				// v5.2.0 replays initial property handlers AFTER OnCreateFinished.
				// Conservatively exclude every delivery in that creation frame.
				if h.attemptNextFrame() == life.createdFrame {
					kind = "creation_replay"
				}
				h.attemptDelivery(e, life, kind, name, value.Any)
			})
		}
	}
	e.OnDestroy(func() { h.attemptDelivery(e, life, "destroyed", "", nil) })
}

// Runs on EVERY FrameDone, before the 16 Hz sampling decision. It seals all
// messages in the delivery frame, without assuming an ordering between project
// callbacks and SDK packet decoding. Multiple native ticks remain ambiguous.
func (h *ReplayHandler) captureSupportAttemptFrame() {
	data := h.ctx.SupportAttemptOriginObservations
	if data == nil {
		return
	}
	t, clock := h.supportAttempt, h.supportObjectiveClock
	current := h.ctx.Parser.GameState().Entities()
	active := false
	for e, life := range t.objects {
		if life.life.alive && current[e.ID()] == e && supportProperty(e, "m_bStartedArming") == true {
			active = true
		}
	}
	defer func() {
		t.lastDelivery, t.wasActive = len(data.Deliveries), active
		t.network = map[string][]*models.SupportClockMessage{"tick": {}, "packet_tick": {}, "interval": {}}
	}()
	if !active && !t.wasActive && t.lastDelivery == len(data.Deliveries) {
		return
	}
	r := models.SupportAttemptFrame{Sequence: len(data.Frames) + 1, Frame: h.support.frame, Tick: h.support.lastTick, Instance: h.support.instance, Epoch: h.support.epoch, Phase: h.roundPhase, DeliverySequence: len(data.Deliveries), Network: t.network, Interval: clock.network["interval"], Items: []models.SupportAttemptItem{}}
	rules := h.ctx.Parser.GameState().Rules()
	if rules != nil && rules.Entity() != nil {
		e := rules.Entity()
		if life := clock.objects[e]; life != nil && life.alive && current[e.ID()] == e {
			r.Rules = &models.SupportClockObject{Life: life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), ClassName: life.class, Fields: clock.fields(e, life.class, "m_pGameRules.", supportClockRules)}
		}
	}
	changed := map[string]bool{}
	for _, delivery := range data.Deliveries[t.lastDelivery:] {
		changed[delivery.Life] = true
	}
	for e, life := range t.objects {
		isCurrent := life.life.alive && current[e.ID()] == e
		if !isCurrent && !changed[life.life.life] {
			continue
		}
		item := models.SupportAttemptItem{Life: life.life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), CreatedFrame: life.createdFrame, Current: isCurrent, Fields: clock.fields(e, "CC4", "", supportAttemptProperties), Definition: entryScalar(e, "m_iItemDefinitionIndex"), InventoryStatus: "owner_unavailable", InventorySlots: []models.SupportC4Reference{}}
		owner, pawn := c4Reference(supportProperty(e, "m_hOwnerEntity"), current)
		if pawn != nil {
			if alias := h.support.lives[pawn]; alias != "" {
				owner.TargetLife = &alias
			} else {
				owner.Status = "kind_or_life_unavailable"
			}
			item.InventoryStatus = "absent"
			if raw := supportProperty(pawn, identityInventory); raw != nil {
				item.InventoryStatus = "unsupported_type"
				if slots, ok := raw.([]any); ok {
					item.InventoryStatus = "outside_probe"
					if len(slots) <= 64 {
						item.InventoryStatus = "observed"
						for slot := range slots {
							item.InventorySlots = append(item.InventorySlots, h.supportC4State.itemReference(supportProperty(pawn, fmt.Sprintf("%s.%04d", identityInventory, slot)), current))
						}
					}
				}
			}
		}
		item.Owner = owner
		r.Items = append(r.Items, item)
	}
	sort.Slice(r.Items, func(i, j int) bool { return r.Items[i].Life < r.Items[j].Life })
	data.Frames = append(data.Frames, r)
}

func (h *ReplayHandler) captureSupportAttemptCut(base models.SupportFrame) {
	data := h.ctx.SupportAttemptOriginObservations
	if data == nil {
		return
	}
	clock := h.ctx.SupportObjectiveClockObservations.Frames[len(h.ctx.SupportObjectiveClockObservations.Frames)-1]
	data.Cuts = append(data.Cuts, models.SupportAttemptCut{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch, ClockOrder: clock.Order, FrameSequence: len(data.Frames), DeliverySequence: len(data.Deliveries)})
}
