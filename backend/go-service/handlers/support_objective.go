package handlers

import (
	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var supportObjectiveRules = []string{"m_bBombPlanted", "m_bFreezePeriod", "m_bWarmupPeriod", "m_iRoundTime"}
var supportObjectivePawnFields = []string{"m_bInBombZone", "m_bIsDefusing", "m_pItemServices.m_bHasDefuser"}

func (h *ReplayHandler) objectiveEvent(kind string) {
	if h.ctx.SupportObjectiveObservations == nil || h.support == nil || h.currentRound == nil {
		return
	}
	t := h.support
	tick, epoch := h.ctx.Parser.GameState().IngameTick(), t.epoch
	if t.hasTick && tick < t.lastTick {
		epoch++
	}
	h.supportObjectiveOrder++
	data := h.ctx.SupportObjectiveObservations
	data.Events = append(data.Events, models.SupportObjectiveEvent{
		Sequence: len(data.Events) + 1, Order: h.supportObjectiveOrder, Frame: t.frame + 1,
		Tick: tick, Instance: t.instance, Epoch: epoch, Kind: kind,
	})
}

func (h *ReplayHandler) registerSupportObjective() {
	if h.ctx.SupportObjectiveObservations == nil {
		return
	}
	// The fixed SDK may synthesize these from Source 2 entity callbacks. We
	// retain their delivery clock; never claim an original network event tick.
	h.ctx.Parser.RegisterEventHandler(func(events.BombPlanted) { h.objectiveEvent("bomb_planted") })
	h.ctx.Parser.RegisterEventHandler(func(events.BombDefused) { h.objectiveEvent("bomb_defused") })
	h.ctx.Parser.RegisterEventHandler(func(events.BombExplode) { h.objectiveEvent("bomb_exploded") })
}

func objectivePlayer(p *common.Player, base models.SupportPlayer) models.SupportObjectivePlayer {
	r := models.SupportObjectivePlayer{NativeActor: base.NativeActor, Fields: map[string]models.EconomyEntryScalar{}}
	var pawn st.Entity
	if p != nil && p.SteamID64 == base.NativeActor && base.Connected && base.IdentityStatus == "observed" && base.PawnHandle != nil && base.PawnLife != nil && base.Alive != nil && *base.Alive && base.AliveStatus == "observed" && base.PositionStatus == "observed" && base.Position != nil {
		pawn = p.PlayerPawnEntity()
		if pawn != nil && (uint64(pawn.ID()) != *base.PawnHandle&16383 || uint64(pawn.SerialNum()) != *base.PawnHandle>>14) {
			pawn = nil
		}
	}
	if pawn != nil {
		r.PawnLife = base.PawnLife
	}
	for _, name := range supportObjectivePawnFields {
		r.Fields[name] = entryScalar(pawn, name)
	}
	return r
}

func (h *ReplayHandler) captureSupportObjective(base models.SupportFrame, players map[uint64]*common.Player) {
	data := h.ctx.SupportObjectiveObservations
	if data == nil {
		return
	}
	h.supportObjectiveOrder++
	f := models.SupportObjectiveFrame{SourceSequence: base.Sequence, Frame: base.Frame, Tick: base.Tick, Instance: base.Instance, Epoch: base.Epoch,
		Order: h.supportObjectiveOrder, EventSequence: len(data.Events), Rules: map[string]models.EconomyEntryScalar{}, Players: []models.SupportObjectivePlayer{}}
	var entity st.Entity
	if rules := h.ctx.Parser.GameState().Rules(); rules != nil {
		entity = rules.Entity()
	}
	for _, name := range supportObjectiveRules {
		// Source 2 bindGameRules in the pinned SDK uses m_pGameRules; the
		// older GameRules.RoundTime accessor still names the Source 1 path.
		f.Rules[name] = entryScalar(entity, "m_pGameRules."+name)
	}
	for _, actor := range base.Players {
		f.Players = append(f.Players, objectivePlayer(players[actor.NativeActor], actor))
	}
	data.Frames = append(data.Frames, f)
}
