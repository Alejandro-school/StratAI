package handlers

import (
	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterBombHandlers registra handlers de bomba
func RegisterBombHandlers(ctx *models.DemoContext) {
	// Bomb planted
	ctx.Parser.RegisterEventHandler(func(e events.BombPlanted) {
		ctx.BombPlanted = true
		ctx.BombTick = ctx.Parser.GameState().IngameTick()

		// Determinar sitio
		if e.Site == 65 { // ASCII 'A'
			ctx.BombSite = "A"
		} else if e.Site == 66 { // ASCII 'B'
			ctx.BombSite = "B"
		}

		// Crear evento
		if e.Player != nil {
			plantEvent := models.BombEvent{
				EventType: "plant",
				Tick:      ctx.BombTick,
				Round:     ctx.CurrentRound,
				Player:    e.Player.Name,
				Site:      ctx.BombSite,
				X:         float64(e.Player.Position().X),
				Y:         float64(e.Player.Position().Y),
				Z:         float64(e.Player.Position().Z),
			}
			ctx.MatchData.BombEvents = append(ctx.MatchData.BombEvents, plantEvent)

			// ADD TO TIMELINE
			AddBombToTimeline(ctx, ctx.BombTick, &plantEvent)
		}
	})

	// Bomb defused
	ctx.Parser.RegisterEventHandler(func(e events.BombDefused) {
		if e.Player == nil {
			return
		}

		defuseEvent := models.BombEvent{
			EventType: "defuse",
			Tick:      ctx.Parser.GameState().IngameTick(),
			Round:     ctx.CurrentRound,
			Player:    e.Player.Name,
			Site:      ctx.BombSite,
			X:         float64(e.Player.Position().X),
			Y:         float64(e.Player.Position().Y),
			Z:         float64(e.Player.Position().Z),
		}

		ctx.MatchData.BombEvents = append(ctx.MatchData.BombEvents, defuseEvent)

		// ADD TO TIMELINE
		AddBombToTimeline(ctx, ctx.Parser.GameState().IngameTick(), &defuseEvent)
	})

	// Bomb exploded
	ctx.Parser.RegisterEventHandler(func(e events.BombExplode) {
		explodeEvent := models.BombEvent{
			EventType: "explode",
			Tick:      ctx.Parser.GameState().IngameTick(),
			Round:     ctx.CurrentRound,
			Site:      ctx.BombSite,
		}

		ctx.MatchData.BombEvents = append(ctx.MatchData.BombEvents, explodeEvent)

		// ADD TO TIMELINE
		AddBombToTimeline(ctx, ctx.Parser.GameState().IngameTick(), &explodeEvent)
	})
}
