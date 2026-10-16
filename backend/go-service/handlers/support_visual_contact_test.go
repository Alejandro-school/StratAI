package handlers

import (
	"cs2-demo-service/models"
	"encoding/json"
	"math"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func visualFixture() (*ReplayHandler, *common.Player, *motionTestEntity, *attemptTestState, models.SupportFrame) {
	h, p, pawn, gs, base := motionFixture()
	for _, name := range supportVisualFields {
		if name == "m_vecX" || name == "m_vecY" || name == "m_vecZ" {
			h.supportObjectiveClock.types["CCSPlayerPawn"][name] = supportClockField{"CNetworkedQuantizedFloat", ""}
			pawn.values[name] = float32(0)
		}
	}
	h.supportObjectiveClock.types["CCSPlayerPawn"]["m_flFlashDuration"] = supportClockField{"float32", ""}
	pawn.values["m_flFlashDuration"] = float32(0)
	for _, axis := range []string{"X", "Y", "Z"} {
		h.supportObjectiveClock.types["Body"]["m_cell"+axis] = supportClockField{"uint16", ""}
		h.supportObjectiveClock.types["Body"]["m_vec"+axis] = supportClockField{"CNetworkedQuantizedFloat", ""}
		pawn.values["CBodyComponent.m_cell"+axis] = uint64(32)
		pawn.values["CBodyComponent.m_vec"+axis] = float32(0)
	}
	h.ctx.SupportVisualObservations = &models.SupportVisualObservations{Deliveries: []models.SupportObjectivePositionDelivery{}, States: []models.SupportVisualState{}, Cuts: []models.SupportVisualCut{}}
	h.supportVisual = &supportVisualTracker{objects: map[st.Entity]*supportMotionLife{}}
	h.bindSupportVisual(pawn)
	return h, p, pawn, gs, base
}

func TestSupportVisualNativeZeroMissingAndNonfinite(t *testing.T) {
	h, p, pawn, _, base := visualFixture()
	h.support.frame = 100
	capture := func() {
		h.captureSupportVisualContact(base, map[uint64]*common.Player{p.SteamID64: p})
		base.Sequence++
		base.Frame++
	}
	capture()
	data := h.ctx.SupportVisualObservations
	state := data.States[0]
	eye := supportVisualEye(data.Cuts[0].Players[0].Coordinates, state.Fields)
	if eye == nil || *eye != (r3.Vector{}) || state.Fields["m_flFlashDuration"].Value != float64(0) || state.Fields["m_pCameraServices.m_iFOV"].Status != "absent" {
		t.Fatal("zero or absence lost")
	}
	pawn.update("m_vecZ", float32(math.NaN()))
	capture()
	state = data.States[len(data.States)-1]
	if state.Fields["m_vecZ"].Status != "nonfinite" || supportVisualEye(data.Cuts[1].Players[0].Coordinates, state.Fields) != nil {
		t.Fatal("nonfinite eye accepted")
	}
	pawn.update("m_vecZ", nil)
	capture()
	if data.States[len(data.States)-1].Fields["m_vecZ"].Status != "absent" {
		t.Fatal("missing eye normalized")
	}
}

func TestSupportVisualCreationDeliverySameTickOrderAndDomains(t *testing.T) {
	h, p, pawn, _, base := visualFixture()
	data := h.ctx.SupportVisualObservations
	pawn.update("m_flFlashDuration", float32(2))
	if data.Deliveries[len(data.Deliveries)-1].Kind != "creation_replay" {
		t.Fatal("creation frontier")
	}
	h.support.frame = 100
	capture := func() {
		h.captureSupportVisualContact(base, map[uint64]*common.Player{p.SteamID64: p})
		base.Sequence++
		base.Frame++
		h.support.frame++
	}
	capture()
	capture()
	if len(data.States) != 1 {
		t.Fatal("unchanged delivered state not shared")
	}
	before, _ := json.Marshal(data.States)
	pawn.update("m_flFlashDuration", float32(0))
	h.supportObjectiveClock.order++
	capture()
	if len(data.States) != 2 || data.Cuts[2].Tick != data.Cuts[0].Tick || data.Cuts[2].ClockOrder != data.Cuts[0].ClockOrder+1 || data.Deliveries[len(data.Deliveries)-1].Kind != "update" {
		t.Fatal("same tick changed delivery collapsed")
	}
	base.Instance++
	capture()
	base.Epoch++
	capture()
	if len(data.States) != 4 {
		t.Fatal("state crossed domain")
	}
	after, _ := json.Marshal(data.States[:1])
	if string(before) != string(after) {
		t.Fatal("future mutated prior state")
	}
}

func TestSupportVisualCurrentLifeSeriesAndDestruction(t *testing.T) {
	for _, mode := range []string{"dead", "disconnected", "wrong-series", "cached", "destroyed", "reused-life"} {
		t.Run(mode, func(t *testing.T) {
			h, p, pawn, gs, base := visualFixture()
			h.support.frame = 100
			h.captureSupportVisualContact(base, map[uint64]*common.Player{p.SteamID64: p})
			switch mode {
			case "dead":
				dead := false
				base.Players[0].Alive = &dead
			case "disconnected":
				base.Players[0].Connected = false
			case "wrong-series":
				pawn.serial++
			case "cached":
				delete(gs.current, pawn.ID())
			case "destroyed":
				for _, fn := range pawn.destroyers {
					fn()
				}
			case "reused-life":
				h.support.lives[pawn] = "support-pawn-999999"
			}
			base.Sequence++
			base.Frame++
			h.captureSupportVisualContact(base, map[uint64]*common.Player{p.SteamID64: p})
			row := h.ctx.SupportVisualObservations.Cuts[1].Players[0]
			if row.StateSequence != nil || len(row.Coordinates) != 0 {
				t.Fatal("unavailable native pawn leaked", mode)
			}
		})
	}
}

type visualGeometryFixture struct {
	loaded, clear bool
	calls         int
}

func (g *visualGeometryFixture) IsLoaded() bool                { return g.loaded }
func (g *visualGeometryFixture) IsVisible(_, _ r3.Vector) bool { g.calls++; return g.clear }
func (g *visualGeometryFixture) GetCallout(r3.Vector) string   { return "" }
func (g *visualGeometryFixture) RayCast(_, _ r3.Vector, _ float64) (float64, r3.Vector) {
	return -1, r3.Vector{}
}

func TestSupportVisualRayIsCurrentOpponentsOnlyAndDoesNotAssumeMissingGeometry(t *testing.T) {
	h, p, pawn, gs, base := visualFixture()
	values := map[string]any{}
	for k, v := range pawn.values {
		values[k] = v
	}
	other := &motionTestEntity{attemptTestEntity: &attemptTestEntity{ownershipEntity: ownershipEntity{sourceTestEntity: sourceTestEntity{values: values}, index: 6, serial: 4}, updates: map[string][]st.PropertyUpdateHandler{}}, class: "CCSPlayerPawn"}
	handle := uint64(4<<14 | 6)
	life := "support-pawn-000002"
	p2 := common.NewPlayer(&supportProvider{pawn: other})
	p2.SteamID64 = 8
	p2.IsConnected = true
	p2.Entity = &ownershipEntity{sourceTestEntity: sourceTestEntity{values: map[string]any{"m_hPawn": uint64(1), "m_hPlayerPawn": handle}}}
	gs.current[6] = other
	h.support.lives[other] = life
	h.bindSupportVisual(other)
	actor := base.Players[0]
	actor.NativeActor = 8
	actor.PawnHandle = &handle
	actor.PawnLife = &life
	actor.Side = "T"
	base.Players = append(base.Players, actor)
	h.support.frame = 100
	geometry := &visualGeometryFixture{loaded: true, clear: true}
	h.ctx.MapManager = geometry
	capture := func() models.SupportVisualCut {
		h.captureSupportVisualContact(base, map[uint64]*common.Player{7: p, 8: p2})
		base.Sequence++
		base.Frame++
		return h.ctx.SupportVisualObservations.Cuts[len(h.ctx.SupportVisualObservations.Cuts)-1]
	}
	cut := capture()
	if len(cut.Rays) != 2 || cut.Rays[0].Clear == nil || !*cut.Rays[0].Clear {
		t.Fatal("current ray not observed", cut)
	}
	geometry.clear = false
	cut = capture()
	if *cut.Rays[0].Clear {
		t.Fatal("blocked ray carried as clear")
	}
	geometry.loaded = false
	before := geometry.calls
	cut = capture()
	if cut.Rays[0].Clear != nil || geometry.calls != before {
		t.Fatal("missing geometry treated as physical obstruction")
	}
	base.Players[1].Side = "CT"
	if len(capture().Rays) != 0 {
		t.Fatal("same-team subject evaluated as enemy")
	}
}
