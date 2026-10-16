package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"strings"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type motionTestClass struct {
	st.ServerClass
	name string
}

func (c motionTestClass) Name() string { return c.name }

type motionTestEntity struct {
	*attemptTestEntity
	class string
}

func (e *motionTestEntity) ServerClass() st.ServerClass { return motionTestClass{name: e.class} }
func (e *motionTestEntity) PropertyValueMust(name string) st.PropertyValue {
	return st.PropertyValue{Any: e.values[name]}
}
func (e *motionTestEntity) Property(name string) st.Property {
	if strings.HasPrefix(name, "m_vecAbsVelocity.") || strings.HasPrefix(name, "m_vecVelocity.") {
		panic("SDK cannot descend through native vector scalar")
	}
	return e.attemptTestEntity.Property(name)
}

func motionFixture() (*ReplayHandler, *common.Player, *motionTestEntity, *attemptTestState, models.SupportFrame) {
	h, old, pawn, _, base, current := c4Fixture()
	e := &motionTestEntity{attemptTestEntity: &attemptTestEntity{ownershipEntity: pawn.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}, class: "CCSPlayerPawn"}
	e.values["m_vecAbsVelocity"] = [3]float32{0, 0, 0}
	e.values["m_fFlags"] = uint64(1)
	e.values["m_hGroundEntity"] = uint64(0)
	e.values["m_MoveType"] = uint64(2)
	e.values["CBodyComponent.m_hParent"] = uint64(16777215)
	world := &motionTestEntity{attemptTestEntity: &attemptTestEntity{ownershipEntity: ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{}}, index: 0, serial: 0}, updates: map[string][]st.PropertyUpdateHandler{}}, class: "CWorld"}
	current[5], current[0] = e, world
	p := common.NewPlayer(&supportProvider{pawn: e})
	p.Entity = old.Entity
	p.SteamID64 = old.SteamID64
	p.IsConnected = true
	h.support.lives = map[st.Entity]string{e: *base.Players[0].PawnLife}
	h.support.frame = 99
	gs := &attemptTestState{c4TestState: c4TestState{current: current}, tick: 200}
	h.ctx.Parser = attemptTestParser{gs: gs}
	h.supportObjectiveClock = &supportClockTracker{types: map[string]map[string]supportClockField{"CCSPlayerPawn": {"m_vecAbsVelocity": {"VectorWS", ""}, "m_fFlags": {"uint32", ""}, "m_hGroundEntity": {"CHandle< CBaseEntity >", ""}, "m_MoveType": {"MoveType_t", ""}, "CBodyComponent": {"CBodyComponent", "Body"}}, "Body": {"m_hParent": {"CGameSceneNodeHandle", ""}}}, order: 7}
	h.ctx.SupportNativeMotionObservations = &models.SupportNativeMotionObservations{}
	h.supportMotion = &supportMotionTracker{objects: map[st.Entity]*supportMotionLife{}, references: map[st.Entity]*supportMotionReferenceLife{}}
	h.bindSupportMotion(e)
	return h, p, e, gs, base
}

func TestSupportNativeMotionCreationReplayFreshCutsAndNativeZero(t *testing.T) {
	h, p, e, _, base := motionFixture()
	data := h.ctx.SupportNativeMotionObservations
	e.update("m_vecAbsVelocity", [3]float32{0, 3, 4})
	if data.Deliveries[len(data.Deliveries)-1].Kind != "creation_replay" {
		t.Fatal("creation callback misclassified")
	}
	h.support.frame = 100
	capture := func() {
		h.captureSupportNativeMotion(base, map[uint64]*common.Player{p.SteamID64: p})
		base.Sequence++
		base.Frame++
	}
	capture()
	capture()
	if len(data.States) != 1 || data.States[0].Ground.Status != "resolved" || *data.States[0].Ground.ClassName != "CWorld" || data.States[0].Fields["m_vecVelocity"].Status != "absent" || data.States[0].Fields["m_vecAbsVelocity"].Value.([]float64)[0] != 0 {
		t.Fatal(data.States)
	}
	before, _ := json.Marshal(data.States[0])
	e.update("m_fFlags", uint64(0))
	capture()
	if len(data.States) != 2 || data.Deliveries[len(data.Deliveries)-1].Kind != "update" || data.Cuts[2].ClockOrder != 7 {
		t.Fatal("cut/update not native")
	}
	base.Epoch++
	capture()
	base.Instance++
	capture()
	if len(data.States) != 4 {
		t.Fatal("state crossed temporal domain")
	}
	after, _ := json.Marshal(data.States[0])
	if string(before) != string(after) {
		t.Fatal("retained state mutated")
	}
}

func TestSupportNativeMotionGroundReferenceUsesCurrentSerialAndLife(t *testing.T) {
	h, p, _, gs, base := motionFixture()
	h.support.frame = 100
	capture := func() {
		h.captureSupportNativeMotion(base, map[uint64]*common.Player{p.SteamID64: p})
		base.Sequence++
		base.Frame++
	}
	capture()
	old := gs.current[0].(*motionTestEntity)
	first := *h.ctx.SupportNativeMotionObservations.States[0].Ground.TargetLife
	for _, fn := range old.destroyers {
		fn()
	}
	other := &motionTestEntity{attemptTestEntity: &attemptTestEntity{ownershipEntity: old.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}, class: "CWorld"}
	other.serial = 1
	gs.current[0] = other
	capture()
	if h.ctx.SupportNativeMotionObservations.States[1].Ground.Status != "serial_mismatch" {
		t.Fatal("stale ground series accepted")
	}
	other.serial = 0
	capture()
	ref := h.ctx.SupportNativeMotionObservations.States[2].Ground
	if ref.Status != "resolved" || *ref.TargetLife == first {
		t.Fatal("reference identity carried across entity destruction")
	}
}

func TestSupportNativeMotionDestructionAndReusedPawnHandle(t *testing.T) {
	h, p, e, gs, base := motionFixture()
	h.support.frame = 100
	h.captureSupportNativeMotion(base, map[uint64]*common.Player{p.SteamID64: p})
	first := h.ctx.SupportNativeMotionObservations.States[0].Life
	for _, fn := range e.destroyers {
		fn()
	}
	base.Sequence++
	base.Frame++
	h.captureSupportNativeMotion(base, map[uint64]*common.Player{p.SteamID64: p})
	if h.ctx.SupportNativeMotionObservations.Cuts[1].Players[0].StateSequence != nil {
		t.Fatal("destroyed pawn remained available")
	}
	other := &motionTestEntity{attemptTestEntity: &attemptTestEntity{ownershipEntity: e.ownershipEntity, updates: map[string][]st.PropertyUpdateHandler{}}, class: "CCSPlayerPawn"}
	gs.current[5] = other
	q := common.NewPlayer(&supportProvider{pawn: other})
	q.Entity = p.Entity
	q.SteamID64 = p.SteamID64
	q.IsConnected = true
	alias := "support-pawn-000002"
	h.support.lives[other] = alias
	base.Players[0].PawnLife = &alias
	h.bindSupportMotion(other)
	h.support.frame = 101
	base.Sequence++
	base.Frame++
	h.captureSupportNativeMotion(base, map[uint64]*common.Player{q.SteamID64: q})
	last := h.ctx.SupportNativeMotionObservations.States[1]
	if last.Life == first || last.Handle != h.ctx.SupportNativeMotionObservations.States[0].Handle {
		t.Fatal("native pawn life reused")
	}
}

func TestSupportNativeMotionDoesNotUseUnavailableActorOrCachedPawn(t *testing.T) {
	h, p, _, gs, base := motionFixture()
	h.support.frame = 100
	delete(gs.current, 5)
	h.captureSupportNativeMotion(base, map[uint64]*common.Player{p.SteamID64: p})
	if h.ctx.SupportNativeMotionObservations.Cuts[0].Players[0].StateSequence != nil {
		t.Fatal("cached pawn accepted")
	}
	h = NewReplayHandler(&models.DemoContext{})
	h.registerSupportNativeMotion()
	h.bindSupportMotion(nil)
	h.captureSupportNativeMotion(models.SupportFrame{}, nil)
	if h.supportMotion != nil {
		t.Fatal("normal mode enabled motion capture")
	}
}
