package analyzers

// Regressions exercise the real FrameDone callback. SDK entities and wall
// outcomes are synthetic; they do not certify visual perception in a demo.
import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/maps"
	"github.com/golang/geo/r3"
	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	dp "github.com/markus-wa/godispatch"
	"testing"
)

type contactProbeEntity struct {
	st.Entity
	id         int
	position   r3.Vector
	properties map[string]st.PropertyValue
}

func (e *contactProbeEntity) PropertyValue(n string) (st.PropertyValue, bool) {
	v, ok := e.properties[n]
	return v, ok
}
func (e *contactProbeEntity) PropertyValueMust(n string) st.PropertyValue {
	v, ok := e.PropertyValue(n)
	if !ok {
		panic("missing fixture property: " + n)
	}
	return v
}
func (e *contactProbeEntity) Position() r3.Vector { return e.position }
func (e *contactProbeEntity) ID() int             { return e.id }
func (e *contactProbeEntity) SerialNum() int      { return 1 }

type contactProbeProvider struct{ pawn st.Entity }

func (p contactProbeProvider) IngameTick() int                              { return 128 }
func (p contactProbeProvider) TickRate() float64                            { return 64 }
func (p contactProbeProvider) FindPlayerByHandle(uint64) *common.Player     { return nil }
func (p contactProbeProvider) FindPlayerByPawnHandle(uint64) *common.Player { return nil }
func (p contactProbeProvider) FindWeaponByEntityID(int) *common.Equipment   { return nil }
func (p contactProbeProvider) FindEntityByHandle(uint64) st.Entity          { return p.pawn }

type contactProbeParticipants struct {
	dem.Participants
	players []*common.Player
}

func (p contactProbeParticipants) Playing() []*common.Player { return p.players }

type contactProbeGameState struct {
	dem.GameState
	participants dem.Participants
}

func (g contactProbeGameState) IngameTick() int                { return 128 }
func (g contactProbeGameState) Participants() dem.Participants { return g.participants }

type contactProbeParser struct {
	dem.Parser
	state   dem.GameState
	onFrame func(events.FrameDone)
}

func (p *contactProbeParser) GameState() dem.GameState { return p.state }
func (p *contactProbeParser) TickRate() float64        { return 64 }
func (p *contactProbeParser) RegisterEventHandler(h any) dp.HandlerIdentifier {
	if f, ok := h.(func(events.FrameDone)); ok {
		p.onFrame = f
	}
	return nil
}

type contactProbeGeometry struct {
	maps.VisibilityChecker
	head, chest, loaded bool
}

func (g contactProbeGeometry) IsLoaded() bool { return g.loaded }
func (g contactProbeGeometry) IsVisible(_, end r3.Vector) bool {
	if end.X != 1000 {
		return false
	}
	if end.Z == 64 {
		return g.head
	}
	if end.Z == 40 {
		return g.chest
	}
	return false
}

func contactProbePlayer(id uint64, team common.Team, x float64, yaw float32, flash float32) *common.Player {
	pawn := &contactProbeEntity{id: int(id), position: r3.Vector{X: x}, properties: map[string]st.PropertyValue{
		"m_iHealth": {Any: int32(100)}, "m_vecX": {Any: float32(0)}, "m_vecY": {Any: float32(0)},
		"m_vecZ": {Any: float32(64)}, "m_angEyeAngles": {Any: [3]float32{0, yaw, 0}},
		"m_bIsScoped": {Any: false}, "m_pWeaponServices.m_hActiveWeapon": {Any: uint64(0)},
	}}
	p := common.NewPlayer(contactProbeProvider{pawn: pawn})
	p.Entity = &contactProbeEntity{properties: map[string]st.PropertyValue{
		"m_hPawn": {Any: uint64(id)}, "m_hPlayerPawn": {Any: uint64(id)},
	}}
	p.SteamID64, p.Team, p.FlashTick, p.FlashDuration = id, team, 128, flash
	return p
}

func TestContactValidationActualFrameCallback(t *testing.T) {
	cases := []struct {
		name                string
		head, chest, loaded bool
		yaw, flash          float32
		smokes              map[int64]r3.Vector
		want                bool
	}{
		{name: "clear_contact", head: true, chest: true, loaded: true, want: true},
		{name: "both_wall_rays_blocked", loaded: true},
		{name: "outside_fov", head: true, chest: true, loaded: true, yaw: 180},
		{name: "flashed_observer", head: true, chest: true, loaded: true, flash: 2},
		{name: "missing_geometry", head: true, chest: true},
		{name: "smoke_blocks_both_rays", head: true, chest: true, loaded: true, smokes: map[int64]r3.Vector{1: {X: 500, Z: 64}}},
		{name: "smoke_outside_both_rays", head: true, chest: true, loaded: true, smokes: map[int64]r3.Vector{1: {X: 500, Y: 200, Z: 64}}, want: true},
		// Neither ray supplies a visible part: the head has smoke and the
		// chest has a wall. A contact here would be a false positive.
		{name: "head_in_smoke_chest_behind_wall", head: true, chest: false, loaded: true, smokes: map[int64]r3.Vector{1: {X: 500, Y: 143.9, Z: 64}}},
		{name: "head_behind_wall_chest_in_smoke", head: false, chest: true, loaded: true, smokes: map[int64]r3.Vector{1: {X: 500, Y: 143.9, Z: 52}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			observer := contactProbePlayer(11, common.TeamTerrorists, 0, c.yaw, c.flash)
			enemy := contactProbePlayer(22, common.TeamCounterTerrorists, 1000, 0, 0)
			p := &contactProbeParser{state: contactProbeGameState{participants: contactProbeParticipants{players: []*common.Player{observer, enemy}}}}
			ctx := models.NewDemoContext(p)
			ctx.InRound, ctx.ActualRoundNumber = true, 1
			ctx.CaptureCalculatedVision = true
			ctx.MapManager = contactProbeGeometry{head: c.head, chest: c.chest, loaded: c.loaded}
			if c.smokes != nil {
				ctx.ActiveSmokes = c.smokes
			}
			RegisterReactionAnalyzer(ctx)
			p.onFrame(events.FrameDone{})
			got := ctx.LastVisibleEnemies[11][22]
			t.Logf("actual_callback_contact=%v expected_contact=%v raycasts=%d", got, c.want, ctx.VisibilityRaycasts)
			if got != c.want {
				t.Errorf("contact=%v, want %v: no single target ray satisfies all visibility conditions", got, c.want)
			}
			samples := ctx.CalculatedVisionSamples[1]
			if len(samples) != 1 || samples[0].Tick != 128 || samples[0].AvailabilityTick != 128 {
				t.Fatalf("missing causal contact sample: %+v", samples)
			}
			found := false
			for _, contact := range samples[0].Contacts {
				if contact.ObserverID == 11 && contact.SubjectID == 22 {
					found = true
					if contact.ObserverSide != "T" || contact.SubjectSide != "CT" || contact.X != 1000 || contact.Z != 0 {
						t.Fatal("wrong captured contact")
					}
				}
			}
			if found != c.want {
				t.Fatalf("exported contact=%v, want %v", found, c.want)
			}
			if !c.loaded && samples[0].Status != "unavailable" {
				t.Fatal("missing geometry labelled calculated")
			}
		})
	}
}
