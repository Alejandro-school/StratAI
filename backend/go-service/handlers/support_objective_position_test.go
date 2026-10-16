package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"math"
	"strings"
	"testing"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type positionTestClass struct{ st.ServerClass }

func (positionTestClass) PropertyEntries() []string {
	return []string{"CBodyComponent.m_extraSpatialField"}
}

type positionTestEntity struct{ *attemptTestEntity }

func (*positionTestEntity) ServerClass() st.ServerClass { return positionTestClass{} }

func (e *positionTestEntity) Property(name string) st.Property {
	if name == "CBodyComponent.m_hParent.m_hOwner" || name == "CBodyComponent.m_hParent.m_name" {
		panic("SDK cannot descend through a scalar parent handle")
	}
	if strings.HasPrefix(name, "CBodyComponent.m_animationCollection.") {
		panic("SDK collection child requires a native array index")
	}
	return e.attemptTestEntity.Property(name)
}

func positionFixture() (*ReplayHandler, *positionTestEntity, *attemptTestState) {
	h, _, _, old, _, current := c4Fixture()
	e := &positionTestEntity{&attemptTestEntity{ownershipEntity: old.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}}
	for _, axis := range []string{"X", "Y", "Z"} {
		e.values["CBodyComponent.m_cell"+axis] = uint64(32)
		e.values["CBodyComponent.m_vec"+axis] = float32(0)
	}
	e.values["CBodyComponent.m_hParent"] = uint64(16777215)
	e.values["CBodyComponent.m_angRotation"] = []float32{0, 0, 0}
	e.values["CBodyComponent.m_extraSpatialField"] = float32(7)
	e.values["CBodyComponent.m_animationCollection"] = []any{uint64(1)}
	current[10] = e
	h.supportC4State.created(e, "CC4")
	gs := &attemptTestState{c4TestState: c4TestState{current: current}, tick: 200}
	h.ctx.Parser = attemptTestParser{gs: gs}
	h.ctx.SupportObjectivePositionObservations = &models.SupportObjectivePositionObservations{}
	h.ctx.SupportObservations = &models.SupportObservations{}
	h.support.frame, h.support.instance = 99, 1
	h.supportObjectiveClock = &supportClockTracker{types: map[string]map[string]supportClockField{"CC4": {"CBodyComponent": {"", "Body"}}, "Body": {"m_hParent": {"CGameSceneNodeHandle", ""}}}, objects: map[st.Entity]*supportC4Life{}}
	for name := range e.values {
		if strings.HasPrefix(name, "CBodyComponent.") {
			if name != "CBodyComponent.m_hParent" {
				h.supportObjectiveClock.types["Body"][strings.TrimPrefix(name, "CBodyComponent.")] = supportClockField{"float32", ""}
			}
		} else {
			h.supportObjectiveClock.types["CC4"][name] = supportClockField{"uint32", ""}
		}
	}
	h.supportObjectiveClock.types["Body"]["m_animationCollection"] = supportClockField{"CUtlVector<Animation>", "Animation"}
	h.supportObjectiveClock.types["Animation"] = map[string]supportClockField{"m_token": {"uint32", ""}}
	h.registerSupportObjectivePosition()
	h.bindSupportPosition(e, h.supportC4State.objects[e])
	return h, e, gs
}

func TestObjectivePositionCreationReplayUpdatesAndBirthAreSeparate(t *testing.T) {
	h, e, gs := positionFixture()
	d := h.ctx.SupportObjectivePositionObservations
	initial := len(d.Deliveries)
	e.update("CBodyComponent.m_vecX", float32(10))
	if d.Deliveries[initial].Kind != "creation_replay" {
		t.Fatal(d.Deliveries[initial])
	}
	h.supportClock(gs.tick)
	h.captureSupportPositionBirths()
	if len(d.Births) != 1 || d.Births[0].Object.Fields["CBodyComponent.m_vecX"].Value != float64(10) || d.Births[0].Frame != 100 {
		t.Fatal(d.Births)
	}
	before, _ := json.Marshal(d.Births)
	e.update("CBodyComponent.m_vecX", float32(20))
	last := d.Deliveries[len(d.Deliveries)-1]
	if last.Kind != "update" || last.Previous.Value != float64(10) || last.Frame != 101 || last.Tick != 200 {
		t.Fatal(last)
	}
	h.supportClock(gs.tick)
	h.captureSupportPositionBirths()
	after, _ := json.Marshal(d.Births)
	if string(before) != string(after) {
		t.Fatal("sealed birth mutated")
	}
}

func TestObjectivePositionFreshCutsParentSerialAndEpoch(t *testing.T) {
	h, e, gs := positionFixture()
	h.supportClock(gs.tick)
	h.captureSupportPositionBirths()
	base := models.SupportFrame{Sequence: 1, Frame: 100, Tick: 200, Instance: 1}
	capture := func() { h.captureSupportObjectivePosition(base); base.Sequence++; base.Frame++ }
	capture()
	capture()
	d := h.ctx.SupportObjectivePositionObservations
	if len(d.States) != 1 || d.States[0].Objects[0].Parent.Status != "invalid_sentinel" {
		t.Fatal(d.States)
	}
	first, _ := json.Marshal(d.States[0])
	e.update("CBodyComponent.m_hParent", uint64(3<<14|5))
	capture()
	parent := d.States[1].Objects[0].Parent
	if parent.Status != "resolved" || parent.TargetLife == nil || *parent.TargetLife != "support-pawn-000001" {
		t.Fatal(parent)
	}
	e.update("CBodyComponent.m_hParent", uint64(4<<14|5))
	capture()
	if d.States[2].Objects[0].Parent.Status != "serial_mismatch" {
		t.Fatal(d.States[2])
	}
	base.Epoch++
	capture()
	if len(d.States) != 4 {
		t.Fatal("state crossed epoch")
	}
	after, _ := json.Marshal(d.States[0])
	if string(first) != string(after) {
		t.Fatal("previous state mutated")
	}
	if d.States[0].Objects[0].Fields["CBodyComponent.m_extraSpatialField"].Value != float64(7) {
		t.Fatal("native body field omitted")
	}
}

func TestObjectivePositionDestructionAndReusedHandleHaveDistinctLives(t *testing.T) {
	h, e, gs := positionFixture()
	h.supportClock(gs.tick)
	h.captureSupportPositionBirths()
	for _, fn := range e.destroyers {
		fn()
	}
	h.captureSupportObjectivePosition(models.SupportFrame{Sequence: 1, Frame: 100, Tick: 200, Instance: 1})
	d := h.ctx.SupportObjectivePositionObservations
	if len(d.States[0].Objects) != 0 || d.Deliveries[len(d.Deliveries)-1].Kind != "destroyed" {
		t.Fatal(d)
	}
	other := &positionTestEntity{&attemptTestEntity{ownershipEntity: e.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}}
	gs.current[10] = other
	h.supportC4State.created(other, "CC4")
	h.bindSupportPosition(other, h.supportC4State.objects[other])
	h.supportClock(gs.tick)
	h.captureSupportPositionBirths()
	if len(d.Births) != 2 || d.Births[0].Object.Life == d.Births[1].Object.Life || d.Births[0].Object.Handle != d.Births[1].Object.Handle {
		t.Fatal(d.Births)
	}
}

func TestObjectivePositionVectorsPresenceAndDisabledCapture(t *testing.T) {
	values := []float32{0, 1, 2}
	observed := positionScalar(values, "QAngle")
	values[0] = 99
	if observed.Status != "observed" || observed.Value.([]float64)[0] != 0 {
		t.Fatal(observed)
	}
	if positionScalar([]float32{0, float32(math.NaN()), 1}, "QAngle").Status != "nonfinite" {
		t.Fatal("NaN accepted")
	}
	if positionScalar([]float32{0, 1}, "QAngle").Status != "unsupported_type" {
		t.Fatal("partial vector accepted")
	}
	for _, raw := range []any{[3]float32{0, 1, 2}, [3]float64{0, 1, 2}} {
		value := positionScalar(raw, "QAngle")
		if value.Status != "observed" || value.Value.([]float64)[2] != 2 {
			t.Fatal("fixed native vector not retained", value)
		}
	}
	if positionScalar([3]float32{0, float32(math.Inf(1)), 0}, "QAngle").Status != "nonfinite" {
		t.Fatal("fixed nonfinite vector accepted")
	}
	if positionScalar(nil, "Vector").Status != "absent" || positionScalar(false, "bool").Value != false {
		t.Fatal("absence collapsed into zero")
	}
	h := NewReplayHandler(&models.DemoContext{})
	h.registerSupportObjectivePosition()
	h.bindSupportPosition(nil, nil)
	h.captureSupportPositionBirths()
	h.captureSupportObjectivePosition(models.SupportFrame{})
	if h.supportPosition != nil {
		t.Fatal("non-candidate capture enabled")
	}
}

func TestObjectivePositionUndeclaredNestedParentProbeDoesNotCallSDK(t *testing.T) {
	h, _, gs := positionFixture()
	h.supportClock(gs.tick)
	h.captureSupportPositionBirths()
	for _, name := range []string{"CBodyComponent.m_hParent.m_hOwner", "CBodyComponent.m_hParent.m_name"} {
		value := h.ctx.SupportObjectivePositionObservations.Births[0].Object.Fields[name]
		if value.Status != "absent" || value.DeclaredType != nil {
			t.Fatal(value)
		}
	}
	collection := h.ctx.SupportObjectivePositionObservations.Births[0].Object.Fields["CBodyComponent.m_animationCollection"]
	if collection.Status != "unsupported_type" || *collection.NativeType != "[]interface {}" || *collection.DeclaredType != "CUtlVector<Animation>" {
		t.Fatal(collection)
	}
}
