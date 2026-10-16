package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	"testing"
)

type attemptTestEntity struct {
	ownershipEntity
	updates    map[string][]st.PropertyUpdateHandler
	destroyers []func()
}
type attemptTestProperty struct {
	st.Property
	entity *attemptTestEntity
	name   string
}

func (p attemptTestProperty) Value() st.PropertyValue {
	return st.PropertyValue{Any: p.entity.values[p.name]}
}
func (p attemptTestProperty) OnUpdate(fn st.PropertyUpdateHandler) {
	p.entity.updates[p.name] = append(p.entity.updates[p.name], fn)
}
func (e *attemptTestEntity) Property(name string) st.Property {
	if _, ok := e.values[name]; !ok {
		return nil
	}
	return attemptTestProperty{entity: e, name: name}
}
func (e *attemptTestEntity) OnDestroy(fn func()) { e.destroyers = append(e.destroyers, fn) }
func (e *attemptTestEntity) update(name string, value any) {
	e.values[name] = value
	for _, fn := range e.updates[name] {
		fn(st.PropertyValue{Any: value})
	}
}

type attemptTestState struct {
	c4TestState
	tick int
}

func (s *attemptTestState) IngameTick() int      { return s.tick }
func (s *attemptTestState) Rules() dem.GameRules { return nil }

type attemptTestParser struct {
	dem.Parser
	gs *attemptTestState
}

func (p attemptTestParser) GameState() dem.GameState { return p.gs }

func attemptFixture() (*ReplayHandler, *attemptTestEntity, *attemptTestState) {
	h, _, _, old, _, current := c4Fixture()
	e := &attemptTestEntity{ownershipEntity: old.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}
	e.values["m_bStartedArming"] = false
	e.values["m_fArmedTime"] = float32(0)
	current[10] = e
	h.supportC4State.created(e, "CC4")
	gs := &attemptTestState{c4TestState: c4TestState{current: current}, tick: 200}
	h.ctx.Parser = attemptTestParser{gs: gs}
	h.ctx.SupportAttemptOriginObservations = &models.SupportAttemptOriginObservations{}
	h.ctx.SupportObservations = &models.SupportObservations{}
	h.supportObjectiveClock = &supportClockTracker{network: map[string]*models.SupportClockMessage{}, types: map[string]map[string]supportClockField{}}
	h.registerSupportAttemptOrigin()
	h.bindSupportAttempt(e, h.supportC4State.objects[e])
	return h, e, gs
}

func TestAttemptCreationReplayIsNotAStartAndLaterCallbacksKeepOrder(t *testing.T) {
	h, e, gs := attemptFixture()
	e.update("m_bStartedArming", true)
	e.update("m_fArmedTime", float32(20))
	data := h.ctx.SupportAttemptOriginObservations
	if len(data.Deliveries) != 5 || data.Deliveries[3].Kind != "creation_replay" {
		t.Fatal(data.Deliveries)
	}
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	first, _ := json.Marshal(data.Frames[0])
	e.update("m_bStartedArming", false)
	e.update("m_bStartedArming", true)
	e.update("m_fArmedTime", float32(21))
	if data.Deliveries[6].Kind != "update" || data.Deliveries[6].Previous.Value != false || data.Deliveries[7].Property != "m_fArmedTime" {
		t.Fatal(data.Deliveries)
	}
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame() // Same tick, distinct FrameDone.
	if data.Frames[1].Frame != data.Frames[0].Frame+1 || data.Frames[1].DeliverySequence != 8 {
		t.Fatal(data.Frames)
	}
	after, _ := json.Marshal(data.Frames[0])
	if string(first) != string(after) {
		t.Fatal("earlier frame mutated")
	}
}

func TestAttemptEveryActiveFrameSealsAllNativeTickMessagesAndResetsLists(t *testing.T) {
	h, e, gs := attemptFixture()
	e.update("m_bStartedArming", true)
	h.clockMessage("tick", uint32(10))
	h.clockMessage("packet_tick", uint32(10))
	h.clockMessage("packet_tick", uint32(11))
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	rows := h.ctx.SupportAttemptOriginObservations.Frames
	if len(rows) != 2 || len(rows[0].Network["packet_tick"]) != 2 || len(rows[1].Network["packet_tick"]) != 0 {
		t.Fatal(rows)
	}
	if rows[0].Items[0].Owner.Status != "resolved" || rows[0].Items[0].InventoryStatus != "observed" || len(rows[0].Items[0].InventorySlots) != 1 {
		t.Fatal(rows[0].Items)
	}
	e.update("m_bStartedArming", false)
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	if len(h.ctx.SupportAttemptOriginObservations.Frames) != 3 {
		t.Fatal("inactive unchanged frames should not expand ledger")
	}
}

func TestAttemptDestructionAndSameHandleNewLifeDoNotInheritStart(t *testing.T) {
	h, e, gs := attemptFixture()
	e.update("m_bStartedArming", true)
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	for _, fn := range e.destroyers {
		fn()
	}
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	data := h.ctx.SupportAttemptOriginObservations
	if data.Frames[1].Items[0].Current || data.Deliveries[len(data.Deliveries)-1].Kind != "destroyed" {
		t.Fatal("life destruction lost")
	}
	replacement := &attemptTestEntity{ownershipEntity: e.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}
	gs.current[10] = replacement
	h.supportC4State.created(replacement, "CC4")
	h.bindSupportAttempt(replacement, h.supportC4State.objects[replacement])
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	if data.Frames[2].Items[0].Life == data.Frames[0].Items[0].Life {
		t.Fatal("reused full handle inherited life")
	}
}

func TestAttemptDisabledAndAbsentPropertiesNeverInventUpdates(t *testing.T) {
	h := NewReplayHandler(&models.DemoContext{})
	h.registerSupportAttemptOrigin()
	h.captureSupportAttemptFrame()
	h.captureSupportAttemptCut(models.SupportFrame{})
	if h.supportAttempt != nil {
		t.Fatal("observer enabled without selector")
	}
	h, e, gs := attemptFixture()
	delete(e.values, "m_fArmedTime")
	h.supportClock(gs.tick)
	h.captureSupportAttemptFrame()
	if h.ctx.SupportAttemptOriginObservations.Frames[0].Items[0].Fields["m_fArmedTime"].Status != "absent" {
		t.Fatal("missing fresh property filled from prior callback")
	}
}
