package demo

import (
	"cs2-demo-service/models"
	"fmt"
	"strings"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// registerMovementHandlers se encarga de los eventos de movimiento, footstep, etc.
func registerMovementHandlers(ctx *DemoContext) {

	// --------------------------------------------------
	// Footstep -> ahora se registra sólo si hay enemigo cerca
	// --------------------------------------------------
	ctx.parser.RegisterEventHandler(func(e events.Footstep) {
		if e.Player == nil || ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		// Definimos un radio de escucha
		const footstepHearRadius = 1250.0

		// Posición del jugador que emite el footstep
		footstepPos := e.Player.Position()

		// Verificamos si hay al menos un enemigo vivo dentro de ese radio
		relevant := false
		for _, pl := range ctx.parser.GameState().Participants().Playing() {
			if pl == nil || pl.Team == e.Player.Team || !pl.IsAlive() {
				// Mismo equipo o muerto, no es un "enemigo" que pueda oírlo
				continue
			}
			dist := vectorDistance(footstepPos, pl.Position())
			if dist <= footstepHearRadius {
				relevant = true
				break
			}
		}

		if relevant {
			ps := getOrCreatePlayerStats(ctx, e.Player.SteamID64, e.Player.Name)
			ps.Footsteps++ // incrementamos el contador de pasos
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(
				ctx.RoundNumber,
				"Footstep",
				fmt.Sprintf("Player %s made a footstep (enemy within %.0f units)", e.Player.Name, footstepHearRadius),
				teamString(e.Player),
			))
		}
	})

	// --------------------------------------------------
	// Jump
	// --------------------------------------------------
	ctx.parser.RegisterEventHandler(func(e events.PlayerJump) {
		if e.Player == nil || ctx.parser.GameState().IsWarmupPeriod() {
			return
		}
		ps := getOrCreatePlayerStats(ctx, e.Player.SteamID64, e.Player.Name)
		ps.Jumps++
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"Jump",
			fmt.Sprintf("Player %s jumped", e.Player.Name),
			teamString(e.Player),
		))
	})

	// --------------------------------------------------
	// PlayerSound
	// --------------------------------------------------
	ctx.parser.RegisterEventHandler(func(e events.PlayerSound) {
		if e.Player == nil || ctx.parser.GameState().IsWarmupPeriod() {
			return
		}
		ps := getOrCreatePlayerStats(ctx, e.Player.SteamID64, e.Player.Name)
		ps.Sounds++
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"Sound",
			fmt.Sprintf("Player %s emitted a sound", e.Player.Name),
			teamString(e.Player),
		))
	})

	// --------------------------------------------------
	// FrameDone => calcular movimiento cada X ticks
	// --------------------------------------------------
	ctx.parser.RegisterEventHandler(func(e events.FrameDone) {
		if !ctx.InRound {
			return
		}
		currentTick := ctx.parser.GameState().IngameTick()

		// Guardamos info de movimiento cada 10 ticks (~cada 78ms en 128tick)
		if currentTick%10 != 0 {
			return
		}
		deltaTicks := currentTick - ctx.LastTick
		if deltaTicks <= 0 {
			return
		}
		ctx.LastTick = currentTick
		deltaTicksFloat := float64(deltaTicks)

		// Recorremos los jugadores para actualizar su posición y velocidad
		for _, pl := range ctx.parser.GameState().Participants().All() {
			sid := pl.SteamID64
			if sid == 0 || !pl.IsAlive() {
				continue
			}

			// Detectar si está usando la mira telescópica
			isScopedNow := false
			if wpn := pl.ActiveWeapon(); wpn != nil && wpn.ZoomLevel() > 0 {
				isScopedNow = true
			}

			// Si cambia de scoping, registramos un evento
			if isScopedNow != ctx.LastScopedState[sid] {
				eventType := "ScopedOut"
				if isScopedNow {
					eventType = "ScopedIn"
				}
				ctx.EventLogs = append(ctx.EventLogs, newEventLog(
					ctx.RoundNumber,
					eventType,
					fmt.Sprintf("Player %s toggled scope %s", pl.Name, strings.ToLower(strings.TrimPrefix(eventType, "Scoped"))),
					teamString(pl),
				))
				ctx.LastScopedState[sid] = isScopedNow
			}

			// Calculamos distancia recorrida y velocidad
			currPos := pl.Position()
			if lastPos, ok := ctx.LastPositions[sid]; ok {
				deltaDist := vectorDistance(currPos, lastPos)
				speed := deltaDist * 128.0 / deltaTicksFloat

				ctx.LastSpeed[sid] = speed
				if speed > ctx.MaxSpeed[sid] {
					ctx.MaxSpeed[sid] = speed
				}

				// Actualizamos PlayerStats
				ps := getOrCreatePlayerStats(ctx, sid, pl.Name)
				ps.DistanceTraveled += deltaDist

				// Registramos tick a tick si está agachado, andando o corriendo
				if pl.IsDucking() {
					ps.CrouchTicks++
				} else if speed < 100 {
					ps.WalkTicks++
				} else {
					ps.RunTicks++
				}
				// Guardamos en Movement (para tus análisis de trayectorias)
				ps.Movement = append(ps.Movement, models.MovementLog{
					Tick:      currentTick,
					X:         currPos.X,
					Y:         currPos.Y,
					Z:         currPos.Z,
					Speed:     speed,
					IsDucking: pl.IsDucking(),
				})
			}
			// Actualizamos la última posición
			ctx.LastPositions[sid] = currPos
		}
	})
}
