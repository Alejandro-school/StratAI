package handlers

import (
	"fmt"
	"math"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type supportTracker struct {
	frame, instance, epoch, lastTick, nextLife int
	hasTick                                    bool
	lives                                      map[st.Entity]string
}

func (h *ReplayHandler) supportClock(tick int) {
	if h.ctx.SupportObservations == nil {
		return
	}
	if h.support == nil {
		h.support = &supportTracker{lives: map[st.Entity]string{}}
	}
	t := h.support
	if t.hasTick && tick < t.lastTick {
		t.epoch++
	}
	t.lastTick, t.hasTick = tick, true
	t.frame++
}

func (h *ReplayHandler) supportRoundStart() {
	if h.ctx.SupportObservations == nil {
		return
	}
	if h.support == nil {
		h.support = &supportTracker{lives: map[st.Entity]string{}}
	}
	h.support.instance++
}

func supportHandle(value any) *uint64 {
	var n uint64
	switch v := value.(type) {
	case uint64:
		n = v
	case uint32:
		n = uint64(v)
	case int:
		if v < 0 {
			return nil
		}
		n = uint64(v)
	case int32:
		if v < 0 {
			return nil
		}
		n = uint64(v)
	case int64:
		if v < 0 {
			return nil
		}
		n = uint64(v)
	default:
		return nil
	}
	if n > math.MaxUint32 {
		return nil
	}
	return &n
}

func (t *supportTracker) player(p *common.Player, sample models.ReplayPlayerState) models.SupportPlayer {
	r := models.SupportPlayer{NativeActor: sample.SteamID, Side: sample.Team, IdentityStatus: "unavailable", PositionStatus: "identity_unavailable", AliveStatus: "identity_unavailable"}
	if p == nil || p.SteamID64 == 0 || p.SteamID64 != sample.SteamID {
		return r
	}
	r.Connected = p.IsConnected
	if p.Entity == nil {
		return r
	}
	f := entryScalar(p.Entity, "m_hPlayerPawn")
	r.PawnHandle = supportHandle(f.Value)
	if r.PawnHandle == nil {
		return r
	}
	pawn := p.PlayerPawnEntity()
	if pawn == nil || int(*r.PawnHandle&16383) != pawn.ID() || int(*r.PawnHandle>>14) != pawn.SerialNum() {
		r.IdentityStatus = "pawn_reference_unresolved"
		return r
	}
	life, exists := t.lives[pawn]
	if !exists {
		t.nextLife++
		life = fmt.Sprintf("support-pawn-%06d", t.nextLife)
		t.lives[pawn] = life
		pawn.OnDestroy(func() { delete(t.lives, pawn) })
	}
	r.PawnLife = &life
	r.IdentityStatus = "observed"
	r.PositionStatus, r.AliveStatus = "observed", "native_fields_absent"
	for _, name := range []string{"CBodyComponent.m_cellX", "CBodyComponent.m_cellY", "CBodyComponent.m_cellZ", "CBodyComponent.m_vecX", "CBodyComponent.m_vecY", "CBodyComponent.m_vecZ"} {
		prop := pawn.Property(name)
		if prop == nil || prop.Value().Any == nil {
			r.PositionStatus = "native_fields_absent"
			break
		}
		switch v := prop.Value().Any.(type) {
		case float32:
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				r.PositionStatus = "nonfinite"
			}
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				r.PositionStatus = "nonfinite"
			}
		}
	}
	if math.IsNaN(sample.Z) || math.IsInf(sample.Z, 0) {
		r.PositionStatus = "nonfinite"
	}
	if r.PositionStatus == "observed" {
		r.Position = &models.SupportPosition{X: sample.X, Y: sample.Y, Z: sample.Z}
	}
	if hp, state := pawn.Property("m_iHealth"), pawn.Property("m_lifeState"); hp != nil && state != nil && hp.Value().Any != nil && state.Value().Any != nil {
		alive := sample.Alive
		r.Alive, r.AliveStatus = &alive, "observed"
	}
	return r
}

func (h *ReplayHandler) captureSupport(sample models.ReplayFrame, step int) {
	if h.ctx.SupportObservations == nil {
		return
	}
	t := h.support
	frame := models.SupportFrame{Sequence: len(h.ctx.SupportObservations.Frames) + 1, Frame: t.frame, Instance: t.instance,
		RoundNumber: h.currentRound.Round, Tick: sample.Tick, Epoch: t.epoch, SampleStep: step, Phase: h.roundPhase, Players: []models.SupportPlayer{}}
	rate := h.ctx.Parser.TickRate()
	if !math.IsNaN(rate) && !math.IsInf(rate, 0) && rate >= 16 && rate <= 256 {
		frame.TickRate = &rate
	}
	players := map[uint64]*common.Player{}
	for _, p := range h.ctx.Parser.GameState().Participants().Playing() {
		if p != nil {
			players[p.SteamID64] = p
		}
	}
	physical := models.SupportPhysicalFrame{SourceSequence: frame.Sequence, Players: []models.SupportPhysicalPlayer{}}
	for _, p := range sample.Players {
		base := t.player(players[p.SteamID], p)
		frame.Players = append(frame.Players, base)
		if h.ctx.SupportPhysicalObservations != nil {
			physical.Players = append(physical.Players, h.supportPhysical(players[p.SteamID], base, p, sample.Tick))
		}
	}
	h.ctx.SupportObservations.Frames = append(h.ctx.SupportObservations.Frames, frame)
	h.captureSupportObjective(frame, players)
	h.captureSupportC4(frame, players)
	h.captureSupportObjectiveClock(frame, players)
	h.captureSupportAttemptCut(frame)
	h.captureSupportObjectivePosition(frame)
	h.captureSupportNativeMotion(frame, players)
	h.captureSupportWeaponTiming(frame, players, physical)
	h.captureSupportVisualContact(frame, players)
	if h.ctx.SupportPhysicalObservations != nil {
		h.ctx.SupportPhysicalObservations.Frames = append(h.ctx.SupportPhysicalObservations.Frames, physical)
	}
}
