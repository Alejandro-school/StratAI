package demo

import (
	"cs2-demo-service/models"
	"fmt"
	"math"
	"strings"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// registerMovementHandlers se encarga de los eventos de movimiento, footstep, etc.
func registerMovementHandlers(ctx *DemoContext) {

	// --------------------------------------------------
	// Footstep -> Contador sin EventLog individual (reduce ruido)
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
			ps.Footsteps++ // Solo contador, sin EventLog
		}
	})

	// --------------------------------------------------
	// Jump -> Solo contador, sin EventLog
	// --------------------------------------------------
	ctx.parser.RegisterEventHandler(func(e events.PlayerJump) {
		if e.Player == nil || ctx.parser.GameState().IsWarmupPeriod() {
			return
		}
		ps := getOrCreatePlayerStats(ctx, e.Player.SteamID64, e.Player.Name)
		ps.Jumps++
		// EventLog eliminado: solo agregamos contador
	})

	// --------------------------------------------------
	// PlayerSound -> ELIMINADO (poco valor para IA)
	// --------------------------------------------------
	// ctx.parser.RegisterEventHandler(func(e events.PlayerSound) {
	// 	...
	// })

	// --------------------------------------------------
	// FrameDone => calcular movimiento cada X ticks
	// --------------------------------------------------
	ctx.parser.RegisterEventHandler(func(e events.FrameDone) {
		if !ctx.InRound {
			return
		}
		currentTick := ctx.parser.GameState().IngameTick()

		// Guardamos info de movimiento cada 10 ticks (~cada 78ms en 128tick)
		// Aumentado de 50 a 10 para mejor tracking de reaction_analysis
		if currentTick%10 != 0 {
			return
		}
		deltaTicks := currentTick - ctx.LastTick
		if deltaTicks <= 0 {
			return
		}
		ctx.LastTick = currentTick
		deltaTicksFloat := float64(deltaTicks)

		// Obtenemos el nombre del mapa para las zonas
		mapName := ctx.parser.Header().MapName

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
				ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
					eventType,
					fmt.Sprintf("Player %s toggled scope %s", pl.Name, strings.ToLower(strings.TrimPrefix(eventType, "Scoped"))),
					teamString(pl),
				))
				ctx.LastScopedState[sid] = isScopedNow
			}

			// Calculamos distancia recorrida y velocidad
			currPos := pl.Position()

			// === NUEVO: Determinar zona del mapa ===
			zone := GetZoneForPosition(mapName, currPos)

			// === NUEVO: Capturar ViewAngles ===
			viewDirectionX := pl.ViewDirectionX() // Pitch (vertical)

			// Calcular Yaw (ángulo horizontal) desde el vector de dirección
			// demoinfocs no da yaw directo, lo calculamos del ViewDirection
			yaw := calculateYawFromViewDirection(pl)

			// === NUEVO: Sistema de visibilidad - USANDO DATOS DEL MOTOR ===
			visibleEnemies := []uint64{}
			for _, enemy := range ctx.parser.GameState().Participants().Playing() {
				if enemy.Team == pl.Team || !enemy.IsAlive() {
					continue
				}
				// Usar HasSpotted del motor del juego - MÁS PRECISO que FOV manual
				if pl.HasSpotted(enemy) {
					visibleEnemies = append(visibleEnemies, enemy.SteamID64)
				}
			}

			// === NUEVO: Team Proximity (distancia a compañeros) ===
			nearestDist := -1.0
			teammatesNear := 0
			for _, teammate := range ctx.parser.GameState().Participants().All() {
				if teammate.Team != pl.Team || !teammate.IsAlive() || teammate.SteamID64 == sid {
					continue
				}
				dist := vectorDistance(currPos, teammate.Position())

				// Contar compañeros dentro de 500u
				if dist <= 500.0 {
					teammatesNear++
				}

				// Trackear el más cercano
				if nearestDist < 0 || dist < nearestDist {
					nearestDist = dist
				}
			}

			isIsolated := (nearestDist > 1000.0 || nearestDist < 0)

			if lastPos, ok := ctx.LastPositions[sid]; ok {
				deltaDist := vectorDistance(currPos, lastPos)
				speed := deltaDist * 128.0 / deltaTicksFloat

				ctx.LastSpeed[sid] = speed

				// Actualizamos PlayerStats
				ps := getOrCreatePlayerStats(ctx, sid, pl.Name)
				ps.DistanceTraveled += deltaDist

				// === NUEVO: Crosshair Placement Analysis ===
				// Pitch negativo = mira hacia abajo, positivo = hacia arriba
				// Rango típico: -89 a +89 grados
				// Head level típico: -10 a +10 grados (aproximadamente horizontal)
				// Ground level: < -30 grados
				pitch := viewDirectionX
				if pitch >= -15 && pitch <= 15 {
					ps.TimeAtHeadLevel++
				} else if pitch >= -30 && pitch < -15 {
					ps.TimeAtBodyLevel++
				} else if pitch < -30 {
					ps.TimeAtGroundLevel++
				}

				// === NUEVO: Rotation Tracking - Detectar cambios de zona ===
				lastZone, hasLastZone := ctx.LastZone[sid]
				if hasLastZone && lastZone != zone && zone != "" && lastZone != "" {
					// El jugador cambió de zona
					startTick := ctx.ZoneEnteredTick[sid]
					endTick := currentTick
					durationMs := int(float64(endTick-startTick) / ctx.parser.TickRate() * 1000)

					// Determinar razón del cambio
					triggerReason := "unknown"
					if ctx.BombPlantedTick > 0 && startTick < ctx.BombPlantedTick && endTick > ctx.BombPlantedTick {
						triggerReason = "bomb_planted"
					}

					// Determinar si fue correcto (si es CT y bomba plantada, debe rotar al sitio)
					wasCorrect := false
					wasTooLate := false
					if pl.Team == common.TeamCounterTerrorists && ctx.BombPlantedSite != "" {
						// Si rotó hacia el sitio de la bomba
						if (ctx.BombPlantedSite == "A" && (zone == "A Site" || zone == "A Long" || zone == "A Short")) ||
							(ctx.BombPlantedSite == "B" && (zone == "B Site" || zone == "B Apartments" || zone == "B Tunnels")) {
							wasCorrect = true

							// Verificar si llegó tarde (más de 10 segundos después del plant)
							if (endTick - ctx.BombPlantedTick) > int(ctx.parser.TickRate()*10) {
								wasTooLate = true
							}
						}
					}

					rotationEvent := models.RotationEvent{
						Round:           ctx.RoundNumber,
						FromZone:        lastZone,
						ToZone:          zone,
						StartTick:       startTick,
						EndTick:         endTick,
						DurationMs:      durationMs,
						TriggerReason:   triggerReason,
						WasCorrect:      wasCorrect,
						WasTooLate:      wasTooLate,
						BombPlantedSite: ctx.BombPlantedSite,
					}

					ps.Rotations = append(ps.Rotations, rotationEvent)
				}

				// Actualizar zona actual
				if zone != lastZone {
					ctx.LastZone[sid] = zone
					ctx.ZoneEnteredTick[sid] = currentTick
				}

				// Guardamos en Movement (sampleo optimizado para IA)
				ps.Movement = append(ps.Movement, models.MovementLog{
					Round:               ctx.RoundNumber,
					Tick:                currentTick,
					X:                   currPos.X,
					Y:                   currPos.Y,
					Z:                   currPos.Z,
					Speed:               speed,
					IsDucking:           pl.IsDucking(),
					Zone:                zone,
					Pitch:               viewDirectionX,
					Yaw:                 yaw,
					NearestTeammateDist: nearestDist,
					TeammatesWithin500:  teammatesNear,
					IsIsolated:          isIsolated,
				})

				// === NUEVO: Guardar snapshot de visibilidad ===
				if len(visibleEnemies) > 0 {
					ps.Visibility = append(ps.Visibility, models.VisibilitySnapshot{
						Tick:           currentTick,
						VisibleEnemies: visibleEnemies,
						LookingAt:      determineLookingAt(pl, visibleEnemies, ctx),
					})
				}

				// === NUEVO: Reaction Time - Tracking mejorado ===
				// Inicializar mapas si no existen
				if _, exists := ctx.EnemyFirstSeenTick[sid]; !exists {
					ctx.EnemyFirstSeenTick[sid] = make(map[uint64]int)
				}
				if _, exists := ctx.ReactionRegistered[sid]; !exists {
					ctx.ReactionRegistered[sid] = make(map[uint64]bool)
				}
				if _, exists := ctx.LastVisibleEnemies[sid]; !exists {
					ctx.LastVisibleEnemies[sid] = make(map[uint64]bool)
				}

				// Convertir slice a map para búsqueda rápida
				currentlyVisible := make(map[uint64]bool)
				for _, enemyID := range visibleEnemies {
					currentlyVisible[enemyID] = true
				}

				// Detectar enemigos que ACABAN DE APARECER (no estaban visibles en frame anterior)
				for _, enemyID := range visibleEnemies {
					wasVisibleBefore := ctx.LastVisibleEnemies[sid][enemyID]

					if !wasVisibleBefore {
						// ENEMIGO ACABA DE APARECER - Resetear tracking
						ctx.EnemyFirstSeenTick[sid][enemyID] = currentTick
						ctx.ReactionRegistered[sid][enemyID] = false
					}
				}

				// Detectar enemigos que DESAPARECIERON (estaban visibles pero ya no)
				for enemyID := range ctx.LastVisibleEnemies[sid] {
					if !currentlyVisible[enemyID] {
						// Enemigo desapareció - limpiar del mapa de visibles
						delete(ctx.LastVisibleEnemies[sid], enemyID)
					}
				}

				// Actualizar estado de enemigos visibles para el próximo frame
				for enemyID := range currentlyVisible {
					ctx.LastVisibleEnemies[sid][enemyID] = true
				}
			}
			// Actualizamos la última posición
			ctx.LastPositions[sid] = currPos
		}
	})
}

// ==================== HELPERS PARA FASE 2 ====================

// calculateYawFromViewDirection calcula el ángulo horizontal (yaw) desde el ViewDirection
func calculateYawFromViewDirection(pl *common.Player) float32 {
	// demoinfocs no expone Yaw directo, pero podemos aproximarlo
	// desde la posición del jugador y su dirección de visión
	// Por simplicidad, usamos ViewDirectionY como aproximación del yaw
	return pl.ViewDirectionY()
}

// isPlayerVisible verifica si un enemigo es visible para el jugador
// Implementación simplificada: verifica distancia y aproximación al campo de visión
func isPlayerVisible(ctx *DemoContext, player *common.Player, enemy *common.Player) bool {
	const maxVisibilityDistance = 3500.0 // Distancia máxima de visibilidad
	const fovAngle = 135.0               // Campo de visión en grados (270° total, 135° a cada lado) - más realista

	playerPos := player.Position()
	enemyPos := enemy.Position()

	// Verificar distancia
	dist := vectorDistance(playerPos, enemyPos)
	if dist > maxVisibilityDistance {
		return false
	}

	// Verificar que el enemigo esté en el FOV del jugador
	yaw := player.ViewDirectionY() // Yaw (horizontal)

	// Calcular vector hacia el enemigo
	dx := enemyPos.X - playerPos.X
	dy := enemyPos.Y - playerPos.Y

	// Calcular ángulo hacia el enemigo en el plano horizontal
	angleToEnemy := float32(math.Atan2(float64(dy), float64(dx)) * 180.0 / math.Pi)

	// Normalizar ángulos a rango [-180, 180]
	yawNormalized := normalizeAngle(yaw)
	angleToEnemyNormalized := normalizeAngle(angleToEnemy)

	// Calcular diferencia angular
	angleDiff := math.Abs(yawNormalized - angleToEnemyNormalized)
	if angleDiff > 180 {
		angleDiff = 360 - angleDiff
	}

	// El enemigo debe estar dentro del FOV
	if angleDiff > fovAngle {
		return false
	}

	// TODO: Implementar raycast para line-of-sight real
	// Por ahora, con distancia + FOV ya es más realista

	return true
}

// normalizeAngle normaliza un ángulo al rango [-180, 180]
func normalizeAngle(angle float32) float64 {
	a := float64(angle)
	for a > 180 {
		a -= 360
	}
	for a < -180 {
		a += 360
	}
	return a
}

// determineLookingAt determina a qué enemigo está mirando el jugador
// Retorna el SteamID del enemigo más cercano al centro del crosshair
func determineLookingAt(player *common.Player, visibleEnemies []uint64, ctx *DemoContext) *uint64 {
	if len(visibleEnemies) == 0 {
		return nil
	}

	const aimThreshold = 15.0 // Grados de tolerancia para considerar que "mira" al enemigo

	playerPos := player.Position()
	// Vector de dirección de la vista del jugador
	// En demoinfocs, ViewDirectionX es el pitch, ViewDirectionY es aproximado al yaw

	var closestEnemy *uint64
	smallestAngle := aimThreshold

	for _, enemyID := range visibleEnemies {
		// Buscar el enemigo en GameState
		var enemy *common.Player
		for _, pl := range ctx.parser.GameState().Participants().All() {
			if pl.SteamID64 == enemyID {
				enemy = pl
				break
			}
		}

		if enemy == nil {
			continue
		}

		enemyPos := enemy.Position()

		// Calcular ángulo entre dirección de vista y dirección al enemigo
		// Simplificación: usamos distancia euclidiana como proxy
		// En producción, calcularías el ángulo real usando vectores

		dx := enemyPos.X - playerPos.X
		dy := enemyPos.Y - playerPos.Y

		// Distancia 2D (horizontal)
		dist2D := math.Sqrt(dx*dx + dy*dy)

		// Si está muy cerca al centro de la vista (simplificación)
		// Necesitaríamos el ángulo real, pero esto es una aproximación
		if dist2D < smallestAngle*100 { // Factor de escala arbitrario
			smallestAngle = float64(dist2D / 100)
			closestEnemy = &enemyID
		}
	}

	return closestEnemy
}
