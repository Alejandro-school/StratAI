package handlers

import (
	"math"
	"testing"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type supportProvider struct{ pawn st.Entity }

func (p *supportProvider) IngameTick() int                              { return 100 }
func (p *supportProvider) TickRate() float64                            { return 64 }
func (p *supportProvider) FindPlayerByHandle(uint64) *common.Player     { return nil }
func (p *supportProvider) FindPlayerByPawnHandle(uint64) *common.Player { return nil }
func (p *supportProvider) FindWeaponByEntityID(int) *common.Equipment   { return nil }
func (p *supportProvider) FindEntityByHandle(uint64) st.Entity          { return p.pawn }

func TestSupportHandlePreservesZeroAndFullSeries(t *testing.T) {
	for _, value := range []any{uint64(0), uint32(0), int(0), int32(0), int64(0), uint64(20000<<14 | 5)} {
		if supportHandle(value) == nil {
			t.Fatal("observed handle lost", value)
		}
	}
	for _, value := range []any{nil, true, float64(1), "1", int(-1), int32(-1), int64(-1), uint64(math.MaxUint32) + 1} {
		if supportHandle(value) != nil {
			t.Fatal("invalid handle accepted", value)
		}
	}
}

func TestSupportClockRunsAcrossUnsampledFramesAndIsOptIn(t *testing.T) {
	h := NewReplayHandler(&models.DemoContext{})
	h.supportClock(100)
	h.supportRoundStart()
	if h.support != nil {
		t.Fatal("normal replay enabled support")
	}
	h.ctx.SupportObservations = &models.SupportObservations{}
	h.supportRoundStart()
	for _, tick := range []int{100, 100, 99, 101} {
		h.supportClock(tick)
	}
	if h.support.frame != 4 || h.support.epoch != 1 || h.support.instance != 1 {
		t.Fatal("clock delivery/epoch lost")
	}
	h.supportRoundStart()
	if h.support.instance != 2 {
		t.Fatal("round instance reused")
	}
}

func TestSupportPawnLifeAndUnavailableIdentityNeverCarryForward(t *testing.T) {
	pawn := &cashflowEntity{ownershipEntity: ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{}}, index: 5, serial: 20000}}
	controller := &ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{"m_hPawn": uint64(1), "m_hPlayerPawn": uint64(20000<<14 | 5)}}}
	provider := &supportProvider{pawn: pawn}
	p := common.NewPlayer(provider)
	p.SteamID64, p.Entity, p.IsConnected = 7, controller, true
	tk := &supportTracker{lives: map[st.Entity]string{}}
	sample := models.ReplayPlayerState{SteamID: 7, Team: "CT", Alive: true, X: 0, Y: 0, Z: 0}
	absent := tk.player(p, sample)
	if absent.Position != nil || absent.Alive != nil || absent.PositionStatus != "native_fields_absent" {
		t.Fatal("missing native components became zero/false")
	}
	for _, name := range []string{"CBodyComponent.m_cellX", "CBodyComponent.m_cellY", "CBodyComponent.m_cellZ", "CBodyComponent.m_vecX", "CBodyComponent.m_vecY", "CBodyComponent.m_vecZ", "m_iHealth", "m_lifeState"} {
		pawn.values[name] = uint64(0)
	}
	first := tk.player(p, sample)
	if first.IdentityStatus != "observed" || first.Position == nil || first.Position.X != 0 {
		t.Fatal("zero position or identity lost")
	}
	again := tk.player(p, sample)
	if *first.PawnLife != *again.PawnLife {
		t.Fatal("same life changed")
	}
	pawn.values["CBodyComponent.m_vecX"] = float32(math.Inf(1))
	if invalid := tk.player(p, sample); invalid.Position != nil || invalid.PositionStatus != "nonfinite" {
		t.Fatal("nonfinite coordinate normalized")
	}
	pawn.values["CBodyComponent.m_vecX"] = float32(0)
	controller.values["m_hPlayerPawn"] = uint64(21024<<14 | 5)
	if tk.player(p, sample).PawnLife != nil {
		t.Fatal("full series truncated")
	}
	controller.values["m_hPlayerPawn"] = uint64(20000<<14 | 5)
	pawn.destroy()
	after := tk.player(p, sample)
	if *after.PawnLife == *first.PawnLife {
		t.Fatal("destroyed life reused")
	}
	delete(controller.values, "m_hPlayerPawn")
	missing := tk.player(p, sample)
	if missing.Position != nil || missing.Alive != nil || missing.PawnLife != nil {
		t.Fatal("absent identity inherited state")
	}
	if first.Position == nil || first.PawnLife == nil {
		t.Fatal("past snapshot mutated")
	}
}
