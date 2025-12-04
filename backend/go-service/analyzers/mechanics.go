package analyzers

import (
	"cs2-demo-service/models"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterMechanicsAnalyzer registra el analizador de mecánicas avanzadas
func RegisterMechanicsAnalyzer(ctx *models.DemoContext) {
	// Tracking de velocidad para Counter-Strafe (últimos 10 ticks)
	velocityHistory := make(map[uint64][]float64)

	// Tracking de Recoil - Deshabilitado por ahora
	// Los datos de m_aimPunchAngle no son confiables en la posición del evento WeaponFire
	lastViewPitch := make(map[uint64]float32)
	lastViewYaw := make(map[uint64]float32)

	// 1. Tracking de Velocidad (FrameDone)
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		for _, player := range ctx.Parser.GameState().Participants().Playing() {
			if !player.IsAlive() {
				continue
			}
			sid := player.SteamID64
			vel := player.Velocity()
			speed := math.Sqrt(vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z)

			// Mantener historial de 10 ticks
			history := velocityHistory[sid]
			if len(history) >= 10 {
				history = history[1:]
			}
			history = append(history, speed)
			velocityHistory[sid] = history

			// Actualizar tracking de View para futuras features
			lastViewPitch[sid] = player.ViewDirectionX()
			lastViewYaw[sid] = player.ViewDirectionY()
		}
	})

	// 2. Análisis al Disparar (WeaponFire)
	ctx.Parser.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter == nil {
			return
		}
		sid := e.Shooter.SteamID64

		// --- Counter-Strafe Analysis ---
		history := velocityHistory[sid]
		if len(history) > 5 {
			// Velocidad actual
			currentSpeed := history[len(history)-1]
			// Velocidad hace 5 ticks (~40ms)
			prevSpeed := history[len(history)-5]

			// Solo analizamos si venía corriendo (necesidad de counter-strafe)
			if prevSpeed > 150 {
				rating := 0.0

				if currentSpeed < 50 {
					// Buen counter-strafe (frenado total)
					rating = 100.0
				} else {
					// Mal counter-strafe (disparó moviéndose)
					rating = 100.0 - (currentSpeed / 250.0 * 100.0)
					if rating < 0 {
						rating = 0
					}
				}

				// Guardamos siempre si hubo oportunidad de counter-strafe
				recordMechanicStat(ctx, sid, "counter_strafe", rating)
				ctx.LastShotMechanics[sid] = &models.ShotMechanics{
					CounterStrafeRating: rating,
					Tick:                ctx.Parser.GameState().IngameTick(),
				}
			}
		}
	})
}

// Helper para registrar stats
func recordMechanicStat(ctx *models.DemoContext, steamID uint64, statType string, value float64) {
	// Asegurar que existe el jugador en MatchData
	if ctx.MatchData.Players[steamID] == nil {
		return // O inicializarlo
	}

	player := ctx.MatchData.Players[steamID]
	if player.Mechanics == nil {
		player.Mechanics = &models.MechanicsStats{}
	}

	// Acumulación simple (promedio móvil)
	switch statType {
	case "counter_strafe":
		if player.Mechanics.AvgCounterStrafeRating == 0 {
			player.Mechanics.AvgCounterStrafeRating = value
		} else {
			player.Mechanics.AvgCounterStrafeRating = (player.Mechanics.AvgCounterStrafeRating + value) / 2
		}
	}
}
