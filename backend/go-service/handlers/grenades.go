package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type utilityHandlerState struct {
	round              int
	phase              utilityRoundPhase
	nextFallbackEffect int64
	equipmentEffects   map[*common.Equipment]int64
}

type utilityRoundPhase uint8

const (
	utilityRoundInactive utilityRoundPhase = iota
	utilityRoundLive
	utilityRoundPost
)

type UtilityHandler struct {
	ctx   *models.DemoContext
	state utilityHandlerState
}

func RegisterGrenadeHandlers(ctx *models.DemoContext) *UtilityHandler {
	if ctx.Utilities == nil {
		ctx.Utilities = utility.NewTracker()
	}
	handler := &UtilityHandler{
		ctx: ctx,
		state: utilityHandlerState{
			equipmentEffects: make(map[*common.Equipment]int64),
		},
	}

	ctx.Parser.RegisterEventHandler(func(events.RoundStart) {
		gameState := ctx.Parser.GameState()
		if gameState == nil {
			return
		}
		round := 0
		warmup := gameState.IsWarmupPeriod()
		if !warmup {
			round = replayRoundNumber(gameState.TotalRoundsPlayed())
		}
		handler.beginRound(round, warmup)
	})
	ctx.Parser.RegisterEventHandler(func(events.RoundEnd) {
		handler.state.markRoundEnd()
	})
	// demoinfocs keeps dispatching post-round events and emits this boundary
	// before the next RoundStart, including its CS2 compatibility path.
	ctx.Parser.RegisterEventHandler(func(events.RoundEndOfficial) {
		handler.finishCurrentRound()
	})

	registerUtilityThrowHandlers(ctx, handler)
	registerUtilityLifecycleHandlers(ctx, handler)
	registerUtilityImpactHandlers(ctx, handler)
	return handler
}

func (handler *UtilityHandler) Finalize() {
	handler.finishCurrentRound()
}

func (handler *UtilityHandler) beginRound(round int, warmup bool) {
	previousRound, closePrevious, startRound := handler.state.beginRound(round, warmup)
	if closePrevious {
		handler.finishRound(previousRound)
	}
	if !startRound {
		if warmup || round <= 0 {
			handler.ctx.ActiveSmokes = make(map[int64]r3.Vector, 8)
			handler.state.equipmentEffects = make(map[*common.Equipment]int64)
		}
		return
	}
	handler.ctx.Utilities.BeginRound(round)
	handler.ctx.ActiveSmokes = make(map[int64]r3.Vector, 8)
	handler.state.equipmentEffects = make(map[*common.Equipment]int64)
}

func (handler *UtilityHandler) finishCurrentRound() bool {
	round, active := handler.state.finishRound()
	if !active {
		return false
	}
	handler.finishRound(round)
	return true
}

func (handler *UtilityHandler) finishRound(round int) {
	handler.ctx.Utilities.EndRound(round)
	refreshLegacyUtilityProjection(handler.ctx)
	handler.ctx.ActiveSmokes = make(map[int64]r3.Vector, 8)
	handler.state.equipmentEffects = make(map[*common.Equipment]int64)
}

func (handler *UtilityHandler) acceptsCallback(group utility.CallbackGroup) bool {
	warmup, available := handler.warmupStatus()
	if available && handler.state.records(warmup) {
		return true
	}
	reason := utility.CallbackDiscardOutsideRound
	if available && warmup {
		reason = utility.CallbackDiscardWarmup
	}
	handler.ctx.Utilities.RecordDiscardedCallback(group, reason)
	return false
}

func (handler *UtilityHandler) invalidCallback(group utility.CallbackGroup) {
	handler.ctx.Utilities.RecordDiscardedCallback(group, utility.CallbackDiscardInvalid)
}

func (handler *UtilityHandler) warmupStatus() (bool, bool) {
	if handler == nil || handler.ctx == nil || handler.ctx.Parser == nil {
		return false, false
	}
	gameState := handler.ctx.Parser.GameState()
	if gameState == nil {
		return false, false
	}
	return gameState.IsWarmupPeriod(), true
}

func (state *utilityHandlerState) beginRound(round int, warmup bool) (int, bool, bool) {
	previousRound, closePrevious := 0, false
	if state.phase != utilityRoundInactive && (state.round != round || state.phase == utilityRoundPost || warmup) {
		previousRound, closePrevious = state.round, true
	}
	if warmup || round <= 0 {
		state.round = 0
		state.phase = utilityRoundInactive
		return previousRound, closePrevious, false
	}
	if state.phase == utilityRoundLive && state.round == round {
		return 0, false, false
	}
	state.round = round
	state.phase = utilityRoundLive
	return previousRound, closePrevious, true
}

func (state *utilityHandlerState) markRoundEnd() bool {
	if state.phase == utilityRoundInactive {
		return false
	}
	state.phase = utilityRoundPost
	return true
}

func (state *utilityHandlerState) finishRound() (int, bool) {
	if state.phase == utilityRoundInactive || state.round <= 0 {
		return 0, false
	}
	round := state.round
	state.round = 0
	state.phase = utilityRoundInactive
	return round, true
}

func (state *utilityHandlerState) records(warmup bool) bool {
	return !warmup && state.round > 0 &&
		(state.phase == utilityRoundLive || state.phase == utilityRoundPost)
}

func shouldRecordGrenadeRound(roundNumber int, warmup bool) bool {
	return roundNumber > 0 && !warmup
}

func getTeamSide(player *common.Player) string {
	if player == nil {
		return ""
	}
	switch player.Team {
	case common.TeamCounterTerrorists:
		return "CT"
	case common.TeamTerrorists:
		return "T"
	default:
		return ""
	}
}
