package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/objective"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

// RegisterBombHandlers records the complete causal C4 lifecycle in one ledger.
// The legacy BombEvents projection remains limited to the three visual markers
// consumed by the current frontend.
func RegisterBombHandlers(ctx *models.DemoContext) {
	tracker := ensureObjectiveTracker(ctx)

	ctx.Parser.RegisterEventHandler(func(e events.BombDropped) {
		if !objectiveRoundActive(ctx) {
			return
		}
		input := objectiveEventInput(ctx, e.Player, "")
		entityID := e.EntityID
		input.EntityID = &entityID
		tracker.Drop(input)
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombPickup) {
		if !objectiveRoundActive(ctx) {
			return
		}
		tracker.Pickup(objectiveEventInput(ctx, e.Player, ""))
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombPlantBegin) {
		if !objectiveRoundActive(ctx) {
			return
		}
		tracker.PlantStart(objectiveEventInput(ctx, e.Player, bombSite(e.Site)))
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombPlantAborted) {
		if !objectiveRoundActive(ctx) {
			return
		}
		tracker.PlantAbort(objectiveEventInput(ctx, e.Player, tracker.Snapshot().Site))
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombPlanted) {
		if !objectiveRoundActive(ctx) {
			return
		}
		event := tracker.Plant(objectiveEventInput(ctx, e.Player, bombSite(e.Site)))
		appendLegacyBombMarker(ctx, "plant", event)
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombDefuseStart) {
		if !objectiveRoundActive(ctx) {
			return
		}
		input := objectiveEventInput(ctx, e.Player, tracker.Snapshot().Site)
		hasKit := e.HasKit
		input.HasKit = &hasKit
		tracker.DefuseStart(input)
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombDefuseAborted) {
		if !objectiveRoundActive(ctx) {
			return
		}
		tracker.DefuseAbort(objectiveEventInput(ctx, e.Player, tracker.Snapshot().Site))
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombDefused) {
		if !objectiveRoundActive(ctx) {
			return
		}
		event := tracker.Defuse(objectiveEventInput(ctx, e.Player, bombSite(e.Site)))
		appendLegacyBombMarker(ctx, "defuse", event)
	})

	ctx.Parser.RegisterEventHandler(func(e events.BombExplode) {
		if !objectiveRoundActive(ctx) {
			return
		}
		input := objectiveEventInput(ctx, e.Player, bombSite(e.Site))
		input.Position = objectiveBombPosition(tracker.Snapshot(), input.Position)
		event := tracker.Explode(input)
		appendLegacyBombMarker(ctx, "explode", event)
	})
}

func appendLegacyBombMarker(ctx *models.DemoContext, eventType string, event objective.Event) {
	marker := models.BombEvent{
		EventType: eventType,
		Tick:      event.Tick,
		Round:     event.Round,
		Player:    event.Actor.Name,
		Site:      event.Site,
	}
	if event.Position.Available() {
		marker.X = event.Position.X
		marker.Y = event.Position.Y
		marker.Z = event.Position.Z
	}
	ctx.MatchData.BombEvents = append(ctx.MatchData.BombEvents, marker)
	AddBombToTimeline(ctx, event.Tick, &marker)
}

func bombSite(site events.Bombsite) string {
	switch site {
	case events.BombsiteA:
		return "A"
	case events.BombsiteB:
		return "B"
	default:
		return ""
	}
}
