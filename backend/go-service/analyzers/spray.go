package analyzers

import (
	"cs2-demo-service/models"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterSprayAnalyzer registra el analizador de sprays
func RegisterSprayAnalyzer(ctx *models.DemoContext) {
	parser := ctx.Parser

	// Detectar inicio de spray en WeaponFire
	parser.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter == nil || e.Weapon == nil {
			return
		}

		sid := e.Shooter.SteamID64
		currentTick := ctx.Parser.GameState().IngameTick()

		// Verificar si es continuación de spray existente
		lastFireTick, exists := ctx.LastWeaponFireTick[sid]

		// Spray = disparos consecutivos con menos de 20 ticks de diferencia (~156ms en 128tick)
		isSprayContinuation := exists && (currentTick-lastFireTick) < 20

		if isSprayContinuation {
			// Continuar spray existente
			spray, sprayExists := ctx.CurrentSpray[sid]
			if !sprayExists {
				return
			}
			spray.ShotCount++

			// Actualizar pitch/yaw acumulado
			spray.PitchSum += float32(math.Abs(float64(e.Shooter.ViewDirectionX())))
			spray.YawSum += float32(math.Abs(float64(e.Shooter.ViewDirectionY())))

		} else {
			// Iniciar nuevo spray (aunque sea de 1 disparo)
			vel := e.Shooter.Velocity()
			speed := math.Sqrt(vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z)

			ctx.CurrentSpray[sid] = &models.SprayData{
				StartTick:   currentTick,
				ShotCount:   1,
				Hits:        0,
				Headshots:   0,
				StartPitch:  e.Shooter.ViewDirectionX(),
				StartYaw:    e.Shooter.ViewDirectionY(),
				PitchSum:    float32(math.Abs(float64(e.Shooter.ViewDirectionX()))),
				YawSum:      float32(math.Abs(float64(e.Shooter.ViewDirectionY()))),
				WasMoving:   speed > 50,
				WasCrouched: e.Shooter.IsDucking(),
				Weapon:      e.Weapon.String(),
			}
		}

		ctx.LastWeaponFireTick[sid] = currentTick
	})

	// Detectar hits dentro del spray
	ctx.Parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker == nil {
			return
		}

		sid := e.Attacker.SteamID64
		spray, exists := ctx.CurrentSpray[sid]

		if exists && spray != nil {
			spray.Hits++
			if e.HitGroup == 1 { // Head
				spray.Headshots++
			}
		}
	})

	// Finalizar sprays al final de ronda
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) {
		// Finalizar todos los sprays activos
		for sid, spray := range ctx.CurrentSpray {
			if spray != nil && spray.ShotCount >= 3 {
				finalizeSpray(ctx, sid, spray)
			}
		}
		// Limpiar todos los sprays
		ctx.CurrentSpray = make(map[uint64]*models.SprayData)
	})

	// Finalizar sprays al cambiar de arma
	ctx.Parser.RegisterEventHandler(func(e events.ItemEquip) {
		if e.Player == nil {
			return
		}

		sid := e.Player.SteamID64
		spray, exists := ctx.CurrentSpray[sid]

		if exists && spray != nil && spray.ShotCount >= 3 {
			finalizeSpray(ctx, sid, spray)
			delete(ctx.CurrentSpray, sid)
		}
	})
}

// finalizeSpray guarda el análisis del spray completado
func finalizeSpray(ctx *models.DemoContext, sid uint64, spray *models.SprayData) {
	// Spray finalizado - calcular métricas
	avgPitch := spray.PitchSum / float32(spray.ShotCount)
	avgYaw := spray.YawSum / float32(spray.ShotCount)

	// Clasificar calidad del spray
	quality := classifySprayQuality(spray.Hits, spray.ShotCount, avgPitch, avgYaw)
	hitRate := float64(spray.Hits) / float64(spray.ShotCount)

	// Obtener/crear PlayerData
	playerData, exists := ctx.MatchData.Players[sid]
	if !exists {
		// No podemos crear PlayerData sin nombre, skip
		return
	}

	// Guardar análisis de spray
	sprayAnalysis := models.SprayAnalysis{
		Round:       ctx.CurrentRound,
		StartTick:   spray.StartTick,
		EndTick:     ctx.Parser.GameState().IngameTick(),
		ShotCount:   spray.ShotCount,
		Hits:        spray.Hits,
		Headshots:   spray.Headshots,
		Weapon:      spray.Weapon,
		HitRate:     hitRate,
		Quality:     quality,
		WasMoving:   spray.WasMoving,
		WasCrouched: spray.WasCrouched,
	}

	playerData.Sprays = append(playerData.Sprays, sprayAnalysis)
}

// classifySprayQuality clasifica la calidad del control de spray
func classifySprayQuality(hits, shots int, avgPitch, avgYaw float32) string {
	hitRate := float64(hits) / float64(shots)
	recoilControl := (avgPitch + avgYaw) / 2

	if hitRate >= 0.7 && recoilControl < 5 {
		return "excellent"
	} else if hitRate >= 0.5 && recoilControl < 10 {
		return "good"
	} else if hitRate >= 0.3 {
		return "fair"
	}
	return "poor"
}
