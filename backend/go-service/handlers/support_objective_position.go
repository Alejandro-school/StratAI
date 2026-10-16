package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"math"
	"sort"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var supportObjectivePositionFields = []string{
	"CBodyComponent.m_cellX", "CBodyComponent.m_cellY", "CBodyComponent.m_cellZ",
	"CBodyComponent.m_vecX", "CBodyComponent.m_vecY", "CBodyComponent.m_vecZ",
	"CBodyComponent.m_hParent", "CBodyComponent.m_angRotation", "CBodyComponent.m_flScale",
	"CBodyComponent.m_name", "CBodyComponent.m_hierarchyAttachName",
	"CBodyComponent.m_hParent.m_hOwner", "CBodyComponent.m_hParent.m_name",
	"CBodyComponent.m_vecAbsOrigin", "CBodyComponent.m_nodeToWorld",
	"m_hMoveParent", "m_iParentAttachment", "m_vecAbsOrigin", "m_vecOrigin",
	"m_hOwnerEntity", "m_hBombDefuser", "m_bBombTicking", "m_bBombDefused",
	"m_iItemDefinitionIndex", "m_fFlags", "m_bStartedArming",
	"m_entitySpottedState.m_bSpotted", "m_entitySpottedState.m_bSpottedByMask.0000", "m_entitySpottedState.m_bSpottedByMask.0001",
}

type supportPositionLife struct {
	life         *supportC4Life
	createdFrame int
	names        []string
	previous     map[string]models.SupportClockScalar
	sequences    map[string]int
	sealed       bool
}

type supportPositionTracker struct {
	objects     map[st.Entity]*supportPositionLife
	fingerprint string
}

func positionScalar(raw any, declared string) models.SupportClockScalar {
	r := clockScalar(raw, declared)
	var values []float64
	switch v := raw.(type) {
	case []float32:
		values = make([]float64, len(v))
		for i, x := range v {
			values[i] = float64(x)
		}
	case []float64:
		values = append([]float64{}, v...)
	case [3]float32:
		values = []float64{float64(v[0]), float64(v[1]), float64(v[2])}
	case [3]float64:
		values = []float64{v[0], v[1], v[2]}
	default:
		return r
	}
	if len(values) != 3 {
		return r
	}
	for _, x := range values {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			r.Status = "nonfinite"
			return r
		}
	}
	r.Status, r.Value = "observed", values
	return r
}

func (h *ReplayHandler) registerSupportObjectivePosition() {
	if h.ctx.SupportObjectivePositionObservations != nil {
		h.supportPosition = &supportPositionTracker{objects: map[st.Entity]*supportPositionLife{}}
	}
}

func (h *ReplayHandler) positionDelivery(life *supportPositionLife, kind, name string, raw any) {
	data := h.ctx.SupportObjectivePositionObservations
	declared := h.supportObjectiveClock.declared(life.life.class, name)
	value := positionScalar(raw, declared)
	previous, ok := life.previous[name]
	if !ok {
		previous = clockScalar(nil, declared)
	}
	d := models.SupportObjectivePositionDelivery{Sequence: len(data.Deliveries) + 1, Frame: h.attemptNextFrame(), Tick: h.ctx.Parser.GameState().IngameTick(), Life: life.life.life, Kind: kind, Property: name, Previous: previous, Value: value}
	data.Deliveries = append(data.Deliveries, d)
	if name != "" {
		life.previous[name], life.sequences[name] = value, d.Sequence
	}
}

func (h *ReplayHandler) positionProperty(e st.Entity, class, name string) st.Property {
	// The SDK panics when a probe descends through a scalar field. Resolve
	// the declaration before asking it to traverse a possibly absent path.
	if h.supportObjectiveClock.declared(class, name) == "" {
		return nil
	}
	return e.Property(name)
}

func (h *ReplayHandler) positionValue(e st.Entity, class, name string) any {
	if property := h.positionProperty(e, class, name); property != nil {
		return property.Value().Any
	}
	return nil
}

func (h *ReplayHandler) bindSupportPosition(e st.Entity, identity *supportC4Life) {
	if h.supportPosition == nil {
		return
	}
	names := map[string]bool{}
	for _, name := range supportObjectivePositionFields {
		names[name] = true
	}
	// PropertyEntries uses serializer names for nested prefixes. Retain each
	// direct native body member instead. A collection remains an explicitly
	// unsupported native value; guessing child paths without array indices is
	// invalid in the SDK and cannot establish an attachment transform.
	body := h.supportObjectiveClock.types[identity.class]["CBodyComponent"]
	for name := range h.supportObjectiveClock.types[body.nested] {
		names["CBodyComponent."+name] = true
	}
	life := &supportPositionLife{life: identity, createdFrame: h.attemptNextFrame(), previous: map[string]models.SupportClockScalar{}, sequences: map[string]int{}}
	for name := range names {
		life.names = append(life.names, name)
	}
	sort.Strings(life.names)
	h.supportPosition.objects[e] = life
	for _, name := range life.names {
		h.positionDelivery(life, "created", name, h.positionValue(e, identity.class, name))
		if prop := h.positionProperty(e, identity.class, name); prop != nil {
			prop.OnUpdate(func(v st.PropertyValue) {
				kind := "update"
				if h.attemptNextFrame() == life.createdFrame {
					kind = "creation_replay"
				}
				h.positionDelivery(life, kind, name, v.Any)
			})
		}
	}
	e.OnDestroy(func() { h.positionDelivery(life, "destroyed", "", nil) })
}

func (h *ReplayHandler) positionPawnReference(raw any, current map[int]st.Entity) models.SupportC4Reference {
	ref, pawn := c4Reference(raw, current)
	if pawn != nil {
		if h.support != nil && h.support.lives[pawn] != "" {
			alias := h.support.lives[pawn]
			ref.TargetLife = &alias
		} else {
			ref.Status = "kind_or_life_unavailable"
		}
	}
	return ref
}

func (h *ReplayHandler) positionObject(e st.Entity, life *supportPositionLife, current map[int]st.Entity) models.SupportObjectivePositionObject {
	r := models.SupportObjectivePositionObject{Life: life.life.life, Handle: uint64(e.SerialNum())<<14 | uint64(e.ID()), ClassName: life.life.class, CreatedFrame: life.createdFrame,
		Fields: map[string]models.SupportClockScalar{}, FieldSequences: map[string]int{},
		Owner: h.positionPawnReference(h.positionValue(e, life.life.class, "m_hOwnerEntity"), current), Defuser: h.positionPawnReference(h.positionValue(e, life.life.class, "m_hBombDefuser"), current)}
	parent, target := c4Reference(h.positionValue(e, life.life.class, "CBodyComponent.m_hParent"), current)
	if target != nil {
		identity := ""
		if h.support != nil {
			identity = h.support.lives[target]
		}
		if tracked := h.supportC4State.objects[target]; tracked != nil && tracked.alive {
			identity = tracked.life
		}
		if tracked := h.supportObjectiveClock.objects[target]; tracked != nil && tracked.alive {
			identity = tracked.life
		}
		if identity != "" {
			parent.TargetLife = &identity
		} else {
			parent.Status = "kind_or_life_unavailable"
		}
	}
	r.Parent = parent
	for _, name := range life.names {
		r.Fields[name] = positionScalar(h.positionValue(e, life.life.class, name), h.supportObjectiveClock.declared(life.life.class, name))
		r.FieldSequences[name] = life.sequences[name]
	}
	return r
}

func (h *ReplayHandler) captureSupportPositionBirths() {
	if h.supportPosition == nil || h.support == nil {
		return
	}
	data := h.ctx.SupportObjectivePositionObservations
	current := h.ctx.Parser.GameState().Entities()
	objects := []st.Entity{}
	for e, life := range h.supportPosition.objects {
		if !life.sealed && life.life.alive && current[e.ID()] == e {
			objects = append(objects, e)
		}
	}
	sort.Slice(objects, func(i, j int) bool {
		return h.supportPosition.objects[objects[i]].life.life < h.supportPosition.objects[objects[j]].life.life
	})
	for _, e := range objects {
		life := h.supportPosition.objects[e]
		life.sealed = true
		data.Births = append(data.Births, models.SupportObjectivePositionBirth{Sequence: len(data.Births) + 1, Frame: h.support.frame, Tick: h.ctx.Parser.GameState().IngameTick(), Instance: h.support.instance, Epoch: h.support.epoch,
			DeliverySequence: len(data.Deliveries), Object: h.positionObject(e, life, current)})
	}
}

func (h *ReplayHandler) captureSupportObjectivePosition(base models.SupportFrame) {
	if h.supportPosition == nil {
		return
	}
	data := h.ctx.SupportObjectivePositionObservations
	current := h.ctx.Parser.GameState().Entities()
	state := models.SupportObjectivePositionState{Objects: []models.SupportObjectivePositionObject{}}
	for e, life := range h.supportPosition.objects {
		if life.life.alive && current[e.ID()] == e {
			state.Objects = append(state.Objects, h.positionObject(e, life, current))
		}
	}
	sort.Slice(state.Objects, func(i, j int) bool { return state.Objects[i].Life < state.Objects[j].Life })
	raw, _ := json.Marshal(state)
	cut := models.SupportObjectivePositionCut{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch, ClockOrder: h.supportObjectiveClock.order, DeliverySequence: len(data.Deliveries), BirthSequence: len(data.Births)}
	changed := len(data.Cuts) == 0 || string(raw) != h.supportPosition.fingerprint
	if len(data.Cuts) > 0 {
		prior := data.Cuts[len(data.Cuts)-1]
		changed = changed || prior.Instance != cut.Instance || prior.Epoch != cut.Epoch
	}
	if changed {
		state.Sequence, state.FirstCut = len(data.States)+1, base.Sequence
		data.States = append(data.States, state)
		h.supportPosition.fingerprint = string(raw)
	}
	cut.StateSequence = len(data.States)
	data.Cuts = append(data.Cuts, cut)
}
