package handlers

import (
	"cs2-demo-service/models"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterPlayerHandlers registra handlers para snapshots de jugadores
func RegisterPlayerHandlers(ctx *models.DemoContext) {
	// Sample movement cada 10 ticks (~78ms en 128tick)
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		currentTick := ctx.Parser.GameState().IngameTick()

		// Sample cada 10 ticks
		if currentTick%10 != 0 {
			return
		}

		// Actualizar LastTick
		if currentTick <= ctx.LastTick {
			return
		}
		ctx.LastTick = currentTick

		// Procesar cada jugador activo
		for _, player := range ctx.Parser.GameState().Participants().Playing() {
			sid := player.SteamID64
			if sid == 0 || !player.IsAlive() {
				continue
			}

			// Asegurar que existe PlayerData
			playerData, exists := ctx.MatchData.Players[sid]
			if !exists {
				playerData = &models.PlayerData{
					SteamID:  sid,
					Name:     player.Name,
					Movement: []models.MovementLog{},
				}
				ctx.MatchData.Players[sid] = playerData
			}

			// Calcular velocidad desde Velocity() nativo
			vel := player.Velocity()
			speed := math.Sqrt(vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z)

			// Calcular distancia recorrida
			currPos := player.Position()
			if lastPos, ok := ctx.LastPositions[sid]; ok {
				deltaDist := math.Sqrt(
					math.Pow(currPos.X-lastPos.X, 2) +
						math.Pow(currPos.Y-lastPos.Y, 2) +
						math.Pow(currPos.Z-lastPos.Z, 2),
				)
				playerData.Distance += deltaDist
			}
			ctx.LastPositions[sid] = currPos

			// Crosshair placement analysis
			pitch := player.ViewDirectionX()

			// Inicializar crosshair stats si no existe
			if _, ok := ctx.CrosshairStats[sid]; !ok {
				ctx.CrosshairStats[sid] = &models.CrosshairStats{}
			}

			// Clasificar por altura de mira
			if pitch >= -15 && pitch <= 15 {
				ctx.CrosshairStats[sid].TimeAtHeadLevel++
			} else if pitch >= -30 && pitch < -15 {
				ctx.CrosshairStats[sid].TimeAtBodyLevel++
			} else if pitch < -30 {
				ctx.CrosshairStats[sid].TimeAtGroundLevel++
			}

			// Crear snapshot de movimiento
			snapshot := models.MovementLog{
				Round:     ctx.CurrentRound,
				Tick:      currentTick,
				X:         currPos.X,
				Y:         currPos.Y,
				Z:         currPos.Z,
				Speed:     speed,
				IsDucking: player.IsDucking(),
				Pitch:     pitch,
				Yaw:       player.ViewDirectionY(),
			}

			playerData.Movement = append(playerData.Movement, snapshot)
		}
	})

}
