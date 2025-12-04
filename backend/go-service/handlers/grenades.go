package handlers

import (
	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterGrenadeHandlers registra handlers de granadas
func RegisterGrenadeHandlers(ctx *models.DemoContext) {
	// Flashbang explode
	ctx.Parser.RegisterEventHandler(func(e events.FlashExplode) {
		if e.Thrower == nil {
			return
		}

		flashEvent := models.FlashEvent{
			Tick:    ctx.Parser.GameState().IngameTick(),
			Round:   ctx.CurrentRound,
			Thrower: e.Thrower.Name,
			X:       float64(e.Position.X),
			Y:       float64(e.Position.Y),
			Z:       float64(e.Position.Z),
			Victims: []models.FlashVictim{},
		}

		// Recorrer jugadores flasheados usando FlashDuration nativo
		for _, player := range ctx.Parser.GameState().Participants().Playing() {
			if player.FlashDuration > 0 {
				flashEvent.Victims = append(flashEvent.Victims, models.FlashVictim{
					Name:     player.Name,
					Duration: player.FlashDuration,
				})
			}
		}

		ctx.MatchData.Flashes = append(ctx.MatchData.Flashes, flashEvent)

		// ADD TO TIMELINE
		AddGrenadeToTimeline(ctx, ctx.Parser.GameState().IngameTick(), "flash", &flashEvent)
	})

	// HE grenade explode
	ctx.Parser.RegisterEventHandler(func(e events.HeExplode) {
		if e.Thrower == nil {
			return
		}

		heEvent := models.HEEvent{
			Tick:    ctx.Parser.GameState().IngameTick(),
			Round:   ctx.CurrentRound,
			Thrower: e.Thrower.Name,
			X:       float64(e.Position.X),
			Y:       float64(e.Position.Y),
			Z:       float64(e.Position.Z),
		}

		ctx.MatchData.HEGrenades = append(ctx.MatchData.HEGrenades, heEvent)

		// ADD TO TIMELINE
		AddGrenadeToTimeline(ctx, ctx.Parser.GameState().IngameTick(), "he", &heEvent)
	})

	// Smoke started
	ctx.Parser.RegisterEventHandler(func(e events.SmokeStart) {
		if e.Thrower == nil {
			return
		}

		smokeEvent := models.SmokeEvent{
			Tick:    ctx.Parser.GameState().IngameTick(),
			Round:   ctx.CurrentRound,
			Thrower: e.Thrower.Name,
			X:       float64(e.Position.X),
			Y:       float64(e.Position.Y),
			Z:       float64(e.Position.Z),
		}

		ctx.MatchData.Smokes = append(ctx.MatchData.Smokes, smokeEvent)

		// ADD TO TIMELINE
		AddGrenadeToTimeline(ctx, ctx.Parser.GameState().IngameTick(), "smoke", &smokeEvent)
	})

	// Molotov/Incendiary started (no tiene thrower info, se registra en grenadeProjectileThrow)
	// Por ahora solo registramos posición sin thrower
	// ctx.Parser.RegisterEventHandler(func(e events.InfernoStart) {
	// 	molotovEvent := models.MolotovEvent{
	// 		Tick:    ctx.Parser.GameState().IngameTick(),
	// 		Round:   ctx.CurrentRound,
	// 		X:       float64(e.Position.X),
	// 		Y:       float64(e.Position.Y),
	// 		Z:       float64(e.Position.Z),
	// 	}
	// 	ctx.MatchData.Molotovs = append(ctx.MatchData.Molotovs, molotovEvent)
	// })

	// --- TRAJECTORY TRACKING ---

	// 1. Start tracking on throw
	ctx.Parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		if e.Projectile == nil || e.Projectile.Thrower == nil {
			return
		}

		// Iniciar tracking
		ctx.ActiveGrenadeTrajectories[e.Projectile.Entity.ID()] = &models.GrenadeTrajectoryEvent{
			GrenadeType: e.Projectile.WeaponInstance.Type.String(),
			Thrower:     e.Projectile.Thrower.Name,
			ThrowerID:   e.Projectile.Thrower.SteamID64,
			Positions:   []models.XYZ{}, // Se llenará frame a frame
		}
	})

	// 2. Update positions every tick (or every N ticks)
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		// Solo si hay granadas activas
		if len(ctx.ActiveGrenadeTrajectories) == 0 {
			return
		}

		// Iterar sobre proyectiles activos en el juego
		for _, proj := range ctx.Parser.GameState().GrenadeProjectiles() {
			// Si lo estamos trackeando
			if traj, exists := ctx.ActiveGrenadeTrajectories[proj.Entity.ID()]; exists {
				pos := proj.Position()
				traj.Positions = append(traj.Positions, models.XYZ{
					X: float64(pos.X),
					Y: float64(pos.Y),
					Z: float64(pos.Z),
				})
			}
		}
	})

	// 3. Finish tracking on destroy
	ctx.Parser.RegisterEventHandler(func(e events.GrenadeProjectileDestroy) {
		if e.Projectile == nil {
			return
		}

		traj, exists := ctx.ActiveGrenadeTrajectories[e.Projectile.Entity.ID()]
		if !exists {
			return
		}

		// Guardar posición final
		pos := e.Projectile.Position()
		traj.LandPosition = models.XYZ{
			X: float64(pos.X),
			Y: float64(pos.Y),
			Z: float64(pos.Z),
		}

		// Agregar evento a timeline
		event := models.TimelineEvent{
			Type:              "grenade_trajectory",
			Tick:              ctx.Parser.GameState().IngameTick(),
			Round:             ctx.CurrentRound,
			GrenadeTrajectory: traj,
		}
		AddTimelineEvent(ctx, event)

		// Limpiar tracking
		delete(ctx.ActiveGrenadeTrajectories, e.Projectile.Entity.ID())
	})
}
