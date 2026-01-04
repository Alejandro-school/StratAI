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
		if e.Shooter == nil || e.Weapon == nil {
			return
		}
		sid := e.Shooter.SteamID64

		// --- Counter-Strafe Analysis (100% Precise using CS2 official formula) ---
		history := velocityHistory[sid]
		if len(history) > 5 {
			// Current velocity at shot time
			currentSpeed := history[len(history)-1]
			// Velocity 5 ticks ago (~40ms at 128 tick)
			prevSpeed := history[len(history)-5]

			// Get weapon-specific accuracy threshold (34% of MaxPlayerSpeed)
			weaponName := e.Weapon.String()
			accuracyThreshold := models.GetAccuracyThreshold(weaponName)
			weaponMaxSpeed := models.GetWeaponMaxSpeed(weaponName)

			// Only analyze if player was moving significantly (needed to counter-strafe)
			// Use weapon-specific threshold instead of fixed 150
			if prevSpeed > accuracyThreshold {
				rating := 0.0

				// Official CS2 Formula:
				// Speed <= 34% MaxSpeed → Perfect accuracy (100% rating)
				// Speed > 34% MaxSpeed → Linear penalty up to 100% MaxSpeed (0% rating)
				if currentSpeed <= accuracyThreshold {
					// Perfect counter-strafe (stopped below accuracy threshold)
					rating = 100.0
				} else if currentSpeed >= weaponMaxSpeed {
					// Still running at max speed (no counter-strafe)
					rating = 0.0
				} else {
					// Partial counter-strafe: linear interpolation
					// Formula: rating = 100 * (1 - (speed - threshold) / (maxSpeed - threshold))
					inaccuracyRange := weaponMaxSpeed - accuracyThreshold
					speedAboveThreshold := currentSpeed - accuracyThreshold
					rating = 100.0 * (1.0 - (speedAboveThreshold / inaccuracyRange))
					if rating < 0 {
						rating = 0
					}
				}

				// Record mechanics stat and save to context
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
