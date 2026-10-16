package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type replayOrderEntity struct{ ownershipEntity }

func (e *replayOrderEntity) Position() r3.Vector { return r3.Vector{X: 10, Y: 20, Z: 30} }
func (e *replayOrderEntity) PropertyValueMust(name string) st.PropertyValue {
	value, ok := e.PropertyValue(name)
	if !ok {
		panic("missing fixture property: " + name)
	}
	return value
}

func replayOrderFixture() (*ReplayHandler, *replayRoundRegistrationParser, *objectiveTestState) {
	state := &objectiveTestState{tick: 110}
	parser := &replayRoundRegistrationParser{state: state}
	h := RegisterReplayHandlers(&models.DemoContext{Parser: parser, CaptureCalculatedVision: true})
	h.currentRound = &models.ReplayRoundData{Round: 1, StartTick: 100}
	h.replayEventThrows = make(map[int64][]models.ReplayContextEvent)
	h.replayEventProjectileEntities = make(map[int][]models.ReplayContextEvent)
	h.replayEventInfernos = make(map[int64]models.ReplayContextEvent)
	return h, parser, state
}

// The real dispatcher iterates a map. Exercise both orders explicitly instead
// of hoping its random iteration exposes a missing dependency during a test.
func replayDispatchOrder[E any](parser *replayRoundRegistrationParser, event E, reverse bool) {
	for offset := range parser.eventHandlers {
		index := offset
		if reverse {
			index = len(parser.eventHandlers) - 1 - offset
		}
		if callback, ok := parser.eventHandlers[index].(func(E)); ok {
			callback(event)
		}
	}
}

func TestReplayUtilityThrowAndSmokeAreIndependentOfCallbackOrder(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		h, parser, state := replayOrderFixture()
		entity := &replayOrderEntity{ownershipEntity: ownershipEntity{index: 17}}
		weapon := &common.Equipment{Type: common.EqSmoke}
		projectile := &common.GrenadeProjectile{Entity: entity, WeaponInstance: weapon,
			Thrower: &common.Player{SteamID64: 7, Team: common.TeamTerrorists}}
		replayDispatchOrder(parser, events.GrenadeProjectileThrow{Projectile: projectile}, reverse)
		if h.currentRound.EventContext == nil || len(h.currentRound.EventContext.Events) != 1 {
			t.Fatalf("launch lost when reverse=%v: %+v", reverse, h.currentRound.EventContext)
		}
		launch := h.currentRound.EventContext.Events[0]
		if launch.ProjectileID != "1" || launch.ActorID != 7 || launch.Tick != 110 || launch.AvailabilityTick != 110 {
			t.Fatalf("launch identity or availability changed: %+v", launch)
		}
		state.tick = 120
		replayDispatchOrder(parser, events.SmokeStart{GrenadeEvent: events.GrenadeEvent{Grenade: weapon,
			GrenadeEntityID: 17, Position: r3.Vector{X: 100, Y: 200}}}, reverse)
		smoke := h.activeSmokes[playerstate.EquipmentID(weapon)]
		if smoke == nil || smoke.ProjectileID != launch.ProjectileID || smoke.ThrowerID != launch.ActorID {
			t.Fatalf("positive smoke link lost when reverse=%v: %+v", reverse, smoke)
		}
		state.tick = 130
		replayDispatchOrder(parser, events.SmokeExpired{GrenadeEvent: events.GrenadeEvent{Grenade: weapon,
			GrenadeEntityID: 17, Position: r3.Vector{X: 100, Y: 200}}}, reverse)
		rows := h.currentRound.EventContext.Events
		if len(rows) != 3 || rows[1].Type != "utility_start" || rows[2].Type != "utility_end" || len(h.activeSmokes) != 0 {
			t.Fatalf("smoke lifecycle lost or duplicated: %+v", rows)
		}
		if rows[1].Tick != 120 || rows[2].Tick != 130 || rows[1].ProjectileID != launch.ProjectileID || rows[2].ProjectileID != launch.ProjectileID {
			t.Fatalf("smoke provenance differs: %+v", rows)
		}
	}
}

func TestReplayInfernoIsIndependentOfCallbackOrderAndExpiryHasNoNewPosition(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		h, parser, state := replayOrderFixture()
		entity := &replayOrderEntity{ownershipEntity: ownershipEntity{index: 19,
			sourceTestEntity: sourceTestEntity{values: map[string]any{"m_fireCount": int32(0)}}}}
		inferno := common.NewInferno(nil, entity, &common.Player{SteamID64: 7, Team: common.TeamTerrorists})
		replayDispatchOrder(parser, events.InfernoStart{Inferno: inferno}, reverse)
		key, _ := replayInfernoKey(inferno)
		active := h.activeInfernos[key]
		if active == nil || active.EffectID == "" || active.ThrowerID != 7 {
			t.Fatalf("inferno link lost when reverse=%v: %+v", reverse, active)
		}
		state.tick = 130
		replayDispatchOrder(parser, events.InfernoExpired{Inferno: inferno}, reverse)
		rows := h.currentRound.EventContext.Events
		if len(rows) != 2 || rows[1].EffectID != rows[0].EffectID || rows[1].ActorID != 7 || len(h.activeInfernos) != 0 {
			t.Fatalf("inferno lifecycle differs: %+v", rows)
		}
		if rows[1].X != nil || rows[1].Y != nil || rows[1].Z != nil {
			t.Fatalf("expiry incorrectly reobserved a historical center: %+v", rows[1])
		}
	}
}
