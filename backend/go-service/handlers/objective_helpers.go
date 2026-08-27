package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/objective"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func ensureObjectiveTracker(ctx *models.DemoContext) *objective.Tracker {
	if ctx.Objectives == nil {
		ctx.Objectives = objective.NewTracker()
	}
	return ctx.Objectives
}

func objectiveRoundActive(ctx *models.DemoContext) bool {
	if ctx == nil || ctx.Parser == nil || ctx.Parser.GameState() == nil || ctx.Parser.GameState().IsWarmupPeriod() {
		return false
	}
	snapshot := ensureObjectiveTracker(ctx).Snapshot()
	return snapshot.Round > 0 && snapshot.Phase != objective.PhaseResolved
}

func objectiveActor(player *common.Player) objective.Actor {
	if player == nil {
		return objective.Actor{}
	}
	return objective.Actor{
		SteamID: player.SteamID64,
		Name:    player.Name,
		Side:    getTeamString(player.Team),
	}
}

func objectivePlayerPosition(player *common.Player, source string) objective.Position {
	if player == nil {
		return objective.UnavailablePosition(source)
	}
	position := player.Position()
	return objective.ObservedPosition(position.X, position.Y, position.Z, source)
}

func objectiveBombPosition(
	snapshot objective.Snapshot,
	fallback objective.Position,
) objective.Position {
	if snapshot.Position.Available() {
		return snapshot.Position
	}
	return fallback
}

func nativeObjectiveObservation(ctx *models.DemoContext, round, tick int) objective.NativeObservation {
	observation := objective.NativeObservation{
		Round:    round,
		Tick:     tick,
		Position: objective.UnavailablePosition(objective.SourceDemoinfocsNativeSnapshot),
	}
	if ctx == nil || ctx.Parser == nil || ctx.Parser.GameState() == nil {
		return observation
	}
	bomb := ctx.Parser.GameState().Bomb()
	if bomb == nil {
		return observation
	}
	position := bomb.Position()
	observation.Position = objective.ObservedPosition(
		position.X,
		position.Y,
		position.Z,
		objective.SourceDemoinfocsNativeSnapshot,
	)
	observation.Carrier = objectiveActor(bomb.Carrier)
	return observation
}

func currentObjectiveRound(ctx *models.DemoContext) int {
	if ctx.CurrentRound > 0 {
		return ctx.CurrentRound
	}
	if ctx.Parser != nil && ctx.Parser.GameState() != nil {
		return ctx.Parser.GameState().TotalRoundsPlayed() + 1
	}
	return 0
}

func objectiveEventInput(ctx *models.DemoContext, player *common.Player, site string) objective.EventInput {
	tick := ctx.Parser.GameState().IngameTick()
	round := currentObjectiveRound(ctx)
	input := objective.EventInput{
		Round:    round,
		Tick:     tick,
		Actor:    objectiveActor(player),
		Site:     site,
		Position: objectivePlayerPosition(player, objective.SourceDemoinfocsEvent),
		Source:   objective.SourceDemoinfocsEvent,
	}
	if player == nil {
		observation := nativeObjectiveObservation(ctx, round, tick)
		input.Position = observation.Position
	}
	return input
}
