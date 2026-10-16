package handlers

import (
	"testing"

	"cs2-demo-service/models"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type ownershipParticipants struct {
	dem.Participants
	players []*common.Player
}

func (p ownershipParticipants) All() []*common.Player { return p.players }

type ownershipGameState struct {
	dem.GameState
	tick           int
	warmup, freeze bool
	players        []*common.Player
}

func (g *ownershipGameState) IngameTick() int          { return g.tick }
func (g *ownershipGameState) IsWarmupPeriod() bool     { return g.warmup }
func (g *ownershipGameState) IsFreezetimePeriod() bool { return g.freeze }
func (g *ownershipGameState) Participants() dem.Participants {
	return ownershipParticipants{players: g.players}
}

type ownershipParser struct {
	dem.Parser
	state *ownershipGameState
}

func (p ownershipParser) GameState() dem.GameState { return p.state }

type ownershipEntity struct {
	sourceTestEntity
	index, serial int
}

func (e *ownershipEntity) ID() int        { return e.index }
func (e *ownershipEntity) SerialNum() int { return e.serial }
func (e *ownershipEntity) PropertyValue(name string) (st.PropertyValue, bool) {
	v, ok := e.values[name]
	return st.PropertyValue{Any: v}, ok
}

func ownershipTrackerFixture() (*economyIdentityTracker, *ownershipGameState, *ownershipEntity, *ownershipEntity) {
	pawn := &ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{identityInventory: []any{uint64(5)}, identityInventory + ".0000": uint64(20<<14 | 5), identityInventory + ".0001": uint64(20<<14 | 5)}}, index: 1, serial: 2}
	weapon := &ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{"m_hOwnerEntity": uint64(2<<14 | 1)}}, index: 5, serial: 20}
	controller := &ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{"m_hPlayerPawn": uint64(2<<14 | 1)}}}
	gs := &ownershipGameState{tick: 1, freeze: true, players: []*common.Player{{SteamID64: 7, Entity: controller, Team: common.TeamTerrorists}}}
	end := 10
	ctx := &models.DemoContext{Parser: ownershipParser{state: gs}, EconomyObservations: &models.EconomyObservationLedger{ActorAliases: map[uint64]string{}, Rounds: []models.EconomyObservedRound{{Instance: 1, StartTick: 0, FreezeEndTick: &end}}}, EconomyOwnershipObservations: &models.EconomyOwnershipObservations{}}
	t := &economyIdentityTracker{ctx: ctx, ledger: &models.EconomyNativeIdentity{}, live: map[int]*identityEntity{1: {alias: "native-entity-000001", kind: "pawn", entity: pawn, alive: true}, 5: {alias: "native-entity-000002", kind: "equipment", entity: weapon, alive: true}}}
	return t, gs, pawn, weapon
}

func TestOwnershipCutsReadFreshStateAndOnlyActiveSlots(t *testing.T) {
	tracker, gs, pawn, weapon := ownershipTrackerFixture()
	tracker.captureOwnership()
	ledger := tracker.ctx.EconomyOwnershipObservations
	if len(ledger.States[0].Pawns[0].Slots) != 1 || ledger.States[0].Equipment[0].Owner.Status != "resolved" {
		t.Fatal("active inventory or owner missing")
	}
	gs.tick, tracker.frame = 2, 1
	tracker.captureOwnership()
	if len(ledger.Cuts) != 2 || len(ledger.States) != 1 {
		t.Fatal("identical fresh state lost its cut or was duplicated")
	}
	pawn.values[identityInventory] = []any{}
	weapon.values["m_hOwnerEntity"] = uint64(4294967295)
	gs.tick, tracker.frame = 3, 2
	tracker.captureOwnership()
	if len(ledger.States) != 2 || len(ledger.States[1].Pawns[0].Slots) != 0 || ledger.States[1].Equipment[0].Owner.Status != "invalid_sentinel" {
		t.Fatal("fresh changes concealed")
	}
	if len(ledger.States[0].Pawns[0].Slots) != 1 {
		t.Fatal("past snapshot mutated")
	}
}

func TestOwnershipLateAliasDoesNotBackfillAndWindowIsStrict(t *testing.T) {
	tracker, gs, _, _ := ownershipTrackerFixture()
	tracker.captureOwnership()
	tracker.ctx.EconomyObservations.ActorAliases[7] = "p0001"
	gs.tick, tracker.frame = 2, 1
	tracker.captureOwnership()
	ledger := tracker.ctx.EconomyOwnershipObservations
	if ledger.States[0].Pawns[0].ActorAlias != nil || ledger.States[1].Pawns[0].ActorAlias == nil {
		t.Fatal("late alias backfilled or ignored")
	}
	for _, tick := range []int{0, 10, 11} {
		gs.tick = tick
		tracker.captureOwnership()
	}
	gs.tick, gs.warmup = 3, true
	tracker.captureOwnership()
	gs.warmup, gs.freeze = false, false
	tracker.captureOwnership()
	if len(ledger.Cuts) != 2 {
		t.Fatal("cut outside strict native freeze")
	}
}
