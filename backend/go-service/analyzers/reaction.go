package analyzers

import (
	"cs2-demo-service/models"
	"math"
	"sync"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// Worker pool para paralelizar raycasts - límite de 6 workers (75% de 8 cores)
const maxWorkers = 6

// Estructura para jobs de visibility check
type visibilityJob struct {
	shooter     *common.Player
	enemy       *common.Player
	shooterEyes r3.Vector
	enemyCenter r3.Vector
	enemyHead   r3.Vector
	shooterID   uint64
	enemyID     uint64
	currentTick int
}

// RegisterReactionAnalyzer registra el analizador de reaction time
func RegisterReactionAnalyzer(ctx *models.DemoContext) {
	// Detectar cuando un enemigo se vuelve visible
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		currentTick := ctx.Parser.GameState().IngameTick()

		// OPTIMIZATION: Sampling enabled (4 ticks = ~31ms at 128tick)
		// This reduces raycasting load by 75%
		if currentTick%4 != 0 {
			return
		}

		// Actualizar smokes activos en el contexto
		ctx.ActiveSmokes = []r3.Vector{}

		// Detectar smokes/molotovs activos mediante Infernos
		for _, inferno := range ctx.Parser.GameState().Infernos() {
			// Obtener fires activos como lista
			fires := inferno.Fires().Active().List()
			if len(fires) > 0 {
				// Calcular centroid de los fires activos
				var sumX, sumY, sumZ float64
				for _, fire := range fires {
					sumX += fire.X
					sumY += fire.Y
					sumZ += fire.Z
				}
				centroid := r3.Vector{
					X: sumX / float64(len(fires)),
					Y: sumY / float64(len(fires)),
					Z: sumZ / float64(len(fires)),
				}
				ctx.ActiveSmokes = append(ctx.ActiveSmokes, centroid)
			}
		}

		// También revisar proyectiles de smoke que aún no detonaron
		for _, proj := range ctx.Parser.GameState().GrenadeProjectiles() {
			if proj.WeaponInstance != nil {
				weapon := proj.WeaponInstance.Type
				// Smoke grenades: 45 = weapon_smokegrenade
				if weapon == 45 {
					ctx.ActiveSmokes = append(ctx.ActiveSmokes, proj.Position())
				}
			}
		}

		// PARALELIZACIÓN: Preparar jobs de visibility checking
		var jobs []visibilityJob

		for _, shooter := range ctx.Parser.GameState().Participants().Playing() {
			if shooter.SteamID64 == 0 || !shooter.IsAlive() {
				continue
			}

			shooterID := shooter.SteamID64
			shooterTeam := shooter.Team
			shooterEyes := shooter.Position()
			shooterEyes.Z += 64

			// Inicializar mapas si no existen
			if ctx.EnemyFirstSeenTick[shooterID] == nil {
				ctx.EnemyFirstSeenTick[shooterID] = make(map[uint64]models.FirstSeenData)
			}
			if ctx.FirstDamageTick[shooterID] == nil {
				ctx.FirstDamageTick[shooterID] = make(map[uint64]int)
			}
			if ctx.LastVisibleEnemies[shooterID] == nil {
				ctx.LastVisibleEnemies[shooterID] = make(map[uint64]bool)
			}

			// Crear jobs para cada enemigo
			for _, enemy := range ctx.Parser.GameState().Participants().Playing() {
				if enemy.SteamID64 == 0 || !enemy.IsAlive() {
					continue
				}
				if enemy.Team == shooterTeam {
					continue
				}

				enemyID := enemy.SteamID64
				enemyCenter := enemy.Position()
				enemyCenter.Z += 40

				// Spatial hashing - skip si está muy lejos
				dx := shooterEyes.X - enemyCenter.X
				dy := shooterEyes.Y - enemyCenter.Y
				dist2DSquared := dx*dx + dy*dy

				if dist2DSquared > 12250000 { // 3500² units
					continue
				}

				// FOV CHECK OPTIMIZATION:
				// Skip enemies that are behind the shooter (Dot product < 0)
				// This compensates for the removal of the cache.
				toEnemy := r3.Vector{
					X: enemyCenter.X - shooterEyes.X,
					Y: enemyCenter.Y - shooterEyes.Y,
					Z: enemyCenter.Z - shooterEyes.Z,
				}.Normalize()

				viewDir := anglesToR3Vector(shooter.ViewDirectionY(), shooter.ViewDirectionX())
				dotProduct := viewDir.Dot(toEnemy)
				if dotProduct < -0.2 { // Allow slightly > 90 deg for peripheral vision (~100 deg fov)
					continue
				}

				// Calculate Heat Position for Dual Raycast
				enemyHead := enemy.Position()
				enemyHead.Z += 62

				// Añadir job a la cola
				jobs = append(jobs, visibilityJob{
					shooter:     shooter,
					enemy:       enemy,
					shooterEyes: shooterEyes,
					enemyCenter: enemyCenter,
					enemyHead:   enemyHead,
					shooterID:   shooterID,
					enemyID:     enemyID,
					currentTick: currentTick,
				})
			}
		}

		// Procesar jobs en paralelo con worker pool limitado
		var wg sync.WaitGroup
		jobsChan := make(chan visibilityJob, len(jobs))
		resultsChan := make(chan struct {
			shooterID uint64
			enemyID   uint64
			isVisible bool
			shooter   *common.Player
			enemy     *common.Player
		}, len(jobs))

		// Lanzar workers (máximo 6 para no saturar CPU)
		numWorkers := maxWorkers
		if len(jobs) < numWorkers {
			numWorkers = len(jobs)
		}

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range jobsChan {
					// DUAL RAYCAST STRATEGY
					// 1. Check HEAD first (Z+62). This catches head-peeks (common in CS).
					// 2. If Head is blocked, check CHEST (Z+40).
					// This ensures we catch all visibility cases.

					isVisibleHead := ctx.MapManager.IsVisible(job.shooterEyes, job.enemyHead)
					isVisibleChest := false
					if !isVisibleHead {
						isVisibleChest = ctx.MapManager.IsVisible(job.shooterEyes, job.enemyCenter)
					}
					isVisible := isVisibleHead || isVisibleChest

					// REMOVED HeuristicIsVisible (Smoke Check)
					// We want purely geometric visibility for FirstSeen.
					// SmokeInPath is calculated later in the event metadata.

					resultsChan <- struct {
						shooterID uint64
						enemyID   uint64
						isVisible bool
						shooter   *common.Player
						enemy     *common.Player
					}{job.shooterID, job.enemyID, isVisible, job.shooter, job.enemy}
				}
			}()
		}

		// Enviar jobs a workers
		for _, job := range jobs {
			jobsChan <- job
		}
		close(jobsChan)

		// Esperar resultados
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Procesar resultados y actualizar estado
		for result := range resultsChan {
			wasVisible := ctx.LastVisibleEnemies[result.shooterID][result.enemyID]

			if result.isVisible {
				// Si ya era visible, solo actualizamos LastSeenTick
				if wasVisible {
					if data, ok := ctx.EnemyFirstSeenTick[result.shooterID][result.enemyID]; ok {
						data.LastSeenTick = currentTick
						ctx.EnemyFirstSeenTick[result.shooterID][result.enemyID] = data
					}
				} else {
					// Enemigo ACABA DE APARECER (o reaparecer)

					// JIGGLE PEEK CHECK:
					// Si existe un registro previo reciente (< 500ms = ~64 ticks), lo mantenemos.
					isJigglePeek := false
					if data, ok := ctx.EnemyFirstSeenTick[result.shooterID][result.enemyID]; ok {
						if currentTick-data.LastSeenTick < 64 {
							// Es un jiggle peek, mantenemos el FirstSeenTick original
							data.LastSeenTick = currentTick
							ctx.EnemyFirstSeenTick[result.shooterID][result.enemyID] = data
							isJigglePeek = true
						}
					}

					if !isJigglePeek {
						// Nuevo avistamiento real
						// Calcular Crosshair Placement Error
						// Determinar objetivo (Cabeza vs Cuerpo) según arma
						targetPos := result.enemy.Position()
						activeWeapon := result.shooter.ActiveWeapon()
						isSniper := false
						if activeWeapon != nil {
							wType := activeWeapon.Type
							if wType == common.EqAWP || wType == common.EqSSG08 || wType == common.EqG3SG1 || wType == common.EqScar20 {
								isSniper = true
							}
						}

						if isSniper {
							targetPos.Z += 40.0 // Altura aproximada pecho/estómago
						} else {
							targetPos.Z += 62.0 // Altura aproximada cabeza
						}

						playerEyePos := result.shooter.Position()
						playerEyePos.Z += 64.0 // Altura ojos

						// Vector Ideal (Desde ojos a objetivo)
						vecIdeal := r3.Vector{
							X: targetPos.X - playerEyePos.X,
							Y: targetPos.Y - playerEyePos.Y,
							Z: targetPos.Z - playerEyePos.Z,
						}.Normalize()

						// Vector Real (Hacia donde mira el jugador)
						// NOTE: ViewDirectionX is Yaw, ViewDirectionY is Pitch
						vecReal := anglesToR3Vector(result.shooter.ViewDirectionY(), result.shooter.ViewDirectionX())

						// Calcular ángulo total
						angleError := calculateAngle(vecIdeal, vecReal)

						// Calcular Pitch y Yaw Error por separado
						// Pitch: Ángulo vertical. Pitch ideal vs Pitch real.
						// Pitch es -90 (arriba) a 90 (abajo) en CS.
						// Vector Ideal Pitch: asin(-z)
						idealPitch := float64(-math.Asin(vecIdeal.Z) * (180.0 / math.Pi))
						realPitch := float64(result.shooter.ViewDirectionY())
						pitchError := math.Abs(idealPitch - realPitch)

						// Yaw: Ángulo horizontal. Atan2(y, x)
						idealYaw := float64(math.Atan2(vecIdeal.Y, vecIdeal.X) * (180.0 / math.Pi))
						realYaw := float64(result.shooter.ViewDirectionX())

						// Normalizar Yaw diff a [-180, 180]
						yawDiff := idealYaw - realYaw
						for yawDiff > 180 {
							yawDiff -= 360
						}
						for yawDiff < -180 {
							yawDiff += 360
						}
						yawError := math.Abs(yawDiff)

						ctx.EnemyFirstSeenTick[result.shooterID][result.enemyID] = models.FirstSeenData{
							Tick:                    currentTick,
							LastSeenTick:            currentTick,
							CrosshairPlacementError: angleError,
							PitchError:              pitchError,
							YawError:                yawError,
						}
					}
				}
			} else if !result.isVisible && wasVisible {
				// Enemigo DESAPARECIÓ
				// NO borramos inmediatamente para permitir Jiggle Peeks.
				// Solo actualizamos LastSeenTick (que ya debería ser el anterior frame visible)
				// El Garbage Collector se encargará de limpiar si pasa mucho tiempo.
			}

			// Actualizar estado
			ctx.LastVisibleEnemies[result.shooterID][result.enemyID] = result.isVisible
		}

		// CLEANUP: Remove enemies that are no longer playing OR haven't been seen in a while
		// We need to do this because they won't appear in the 'jobs' list anymore
		playingSet := make(map[uint64]bool)
		for _, p := range ctx.Parser.GameState().Participants().Playing() {
			playingSet[p.SteamID64] = true
		}

		for shooterID, enemiesMap := range ctx.EnemyFirstSeenTick {
			for enemyID, data := range enemiesMap {
				// 1. Si el enemigo ya no juega (muerto/desconectado), borrar.
				if !playingSet[enemyID] {
					delete(enemiesMap, enemyID)
					continue
				}

				// 2. Si ha pasado mucho tiempo desde el último avistamiento (Grace Period Exceeded)
				// 64 ticks = 500ms. Si lleva > 500ms oculto, asumimos que el encuentro terminó.
				if currentTick-data.LastSeenTick > 64 {
					delete(enemiesMap, enemyID)
				}
			}
			// Limpiar mapa vacío
			if len(enemiesMap) == 0 {
				delete(ctx.EnemyFirstSeenTick, shooterID)
			}
		}

	})

	// Detectar disparos y calcular reaction time
	ctx.Parser.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter == nil {
			return
		}

		shooterID := e.Shooter.SteamID64
		currentTick := ctx.Parser.GameState().IngameTick()

		// Verificar si hay un enemigo visible recién detectado
		if ctx.EnemyFirstSeenTick[shooterID] != nil {
			for enemyID, firstSeenData := range ctx.EnemyFirstSeenTick[shooterID] {
				ticksSinceVisible := currentTick - firstSeenData.Tick

				// Reaction time válido: entre 0 y 320 ticks (0ms - ~2500ms at 128tick)
				// Rango amplio para capturar desde reacciones instantáneas hasta más lentas
				if ticksSinceVisible >= 0 && ticksSinceVisible <= 320 {
					tickRate := ctx.Parser.TickRate()
					if tickRate == 0 {
						tickRate = 128 // Fallback
					}
					tickInterval := 1000.0 / tickRate
					reactionTimeMs := int(float64(ticksSinceVisible) * tickInterval)

					// Obtener enemy player para calcular metadata
					var enemy *common.Player
					for _, p := range ctx.Parser.GameState().Participants().Playing() {
						if p.SteamID64 == enemyID {
							enemy = p
							break
						}
					}

					// Calcular metadata
					wasFlashed := false
					flashDuration := float32(0.0)
					smokeInPath := false
					distance := 0.0

					if enemy != nil {
						// Flash check
						if e.Shooter.FlashDuration > 0 {
							wasFlashed = true
							flashDuration = e.Shooter.FlashDuration
						}

						// Posiciones
						shooterPos := e.Shooter.Position()
						shooterPos.Z += 64 // Eye level
						enemyPos := enemy.Position()
						enemyPos.Z += 40 // Chest level

						// Distancia
						distance = shooterPos.Sub(enemyPos).Norm()

						// Smoke check
						for _, smokePos := range ctx.ActiveSmokes {
							if distancePointToSegment(smokePos, shooterPos, enemyPos) < 140.0 {
								smokeInPath = true
								break
							}
						}
					}

					// Obtener/crear PlayerData
					playerData, exists := ctx.MatchData.Players[shooterID]
					if !exists {
						playerData = &models.PlayerData{
							SteamID: shooterID,
							Name:    e.Shooter.Name,
						}
						ctx.MatchData.Players[shooterID] = playerData
					}

					// Guardar reaction time event con metadata
					reactionEvent := models.ReactionTimeEvent{
						Round:                   ctx.CurrentRound,
						EnemyID:                 enemyID,
						FirstSeenTick:           firstSeenData.Tick,
						FirstShotTick:           currentTick,
						ReactionTimeMs:          reactionTimeMs,
						WasFlashed:              wasFlashed,
						FlashDuration:           flashDuration,
						SmokeInPath:             smokeInPath,
						Distance:                distance,
						CrosshairPlacementError: firstSeenData.CrosshairPlacementError,
						PitchError:              firstSeenData.PitchError,
						YawError:                firstSeenData.YawError,
					}
					playerData.ReactionTimes = append(playerData.ReactionTimes, reactionEvent)

					// Limpiar tracking para este enemigo específico
					// REMOVED: Do NOT delete here. We need this data for PlayerHurt (FirstDamageTick) calculation.
					// It will be cleaned up by the FrameDone "grace period" check.
					// delete(ctx.EnemyFirstSeenTick[shooterID], enemyID)
				}
			}
		}
	})

	// Track FirstDamageTick (when damage is dealt, not just when shooting)
	ctx.Parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker == nil || e.Player == nil {
			return
		}

		attackerID := e.Attacker.SteamID64
		victimID := e.Player.SteamID64
		currentTick := ctx.Parser.GameState().IngameTick()

		// Initialize maps if needed
		if ctx.FirstDamageTick[attackerID] == nil {
			ctx.FirstDamageTick[attackerID] = make(map[uint64]int)
		}

		// Record FirstDamageTick if not set (for other analytics)
		if _, exists := ctx.FirstDamageTick[attackerID][victimID]; !exists {
			ctx.FirstDamageTick[attackerID][victimID] = currentTick
		}

		// ALWAYS calculate TimeToDamage for every hit
		// This ensures we capture the TTD for the kill shot even if there was prior damage.
		if firstSeenMap, ok := ctx.EnemyFirstSeenTick[attackerID]; ok {
			if firstSeenData, ok := firstSeenMap[victimID]; ok {
				deltaTicks := currentTick - firstSeenData.Tick
				if deltaTicks >= 0 {
					tickRate := ctx.Parser.TickRate()
					if tickRate == 0 {
						tickRate = 64
					}
					timeToDamageMs := float64(deltaTicks) * (1000.0 / tickRate)

					// Find the most recent ReactionTimeEvent for this attacker/victim pair
					if playerData, exists := ctx.MatchData.Players[attackerID]; exists {
						// Update the LAST reaction time event with TimeToDamage
						// We look for a recent event (last 128 ticks) to associate this damage with.
						for i := len(playerData.ReactionTimes) - 1; i >= 0; i-- {
							rt := &playerData.ReactionTimes[i]
							if rt.EnemyID == victimID && rt.TimeToDamage == 0 {
								// Check timeframe: Allow longer window since TTFD can be high
								if currentTick-rt.FirstShotTick <= 128 {
									rt.TimeToDamage = timeToDamageMs
									break
								}
							}
						}
					}

					// NOTE: We no longer delete FirstSeenTick here because:
					// 1. PlayerHurt fires BEFORE WeaponFire in the same tick
					// 2. This was causing WeaponFire to miss the reaction time data
					// 3. The cleanup in FrameDone (grace period 64 ticks) handles stale entries
					//
					// OLD CODE (removed):
					// delete(firstSeenMap, victimID)
				}
			}
		}
	})

	// Handler de Kill events para validar reaction times con datos oficiales
	ctx.Parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer == nil || e.Victim == nil {
			return
		}

		killerID := e.Killer.SteamID64
		victimID := e.Victim.SteamID64
		currentTick := ctx.Parser.GameState().IngameTick()

		// Buscar si hay un reaction time reciente para esta kill
		if playerData, exists := ctx.MatchData.Players[killerID]; exists {
			// Buscar reaction time en los últimos 10 ticks (~156ms)
			for i := len(playerData.ReactionTimes) - 1; i >= 0; i-- {
				rt := &playerData.ReactionTimes[i]

				// Si es el reaction time correcto (mismo enemigo, tick cercano)
				if rt.EnemyID == victimID && (currentTick-rt.FirstShotTick) <= 10 {
					// Enriquecer con datos oficiales del juego
					rt.PenetratedObjects = e.PenetratedObjects

					// Validar y corregir discrepancias
					if e.ThroughSmoke && !rt.SmokeInPath {
						rt.SmokeInPath = true
					}

					if e.AttackerBlind && !rt.WasFlashed {
						rt.WasFlashed = true
					}

					break
				}
			}
		}
	})
}

// Helper function
func distancePointToSegment(p, a, b r3.Vector) float64 {
	ab := b.Sub(a)
	ap := p.Sub(a)
	t := ap.Dot(ab) / ab.Dot(ab)
	if t < 0.0 {
		t = 0.0
	} else if t > 1.0 {
		t = 1.0
	}
	nearest := a.Add(ab.Mul(t))
	return p.Sub(nearest).Norm()
}

func anglesToR3Vector(pitch, yaw float32) r3.Vector {
	p := float64(pitch) * math.Pi / 180.0
	y := float64(yaw) * math.Pi / 180.0
	sinP := math.Sin(p)
	cosP := math.Cos(p)
	sinY := math.Sin(y)
	cosY := math.Cos(y)
	return r3.Vector{X: cosP * cosY, Y: cosP * sinY, Z: -sinP}
}

func calculateAngle(v1, v2 r3.Vector) float64 {
	dot := v1.Dot(v2)
	// Clamp dot product to [-1, 1] to avoid NaN with Acos
	if dot > 1.0 {
		dot = 1.0
	} else if dot < -1.0 {
		dot = -1.0
	}
	angleRad := math.Acos(dot)
	return angleRad * (180.0 / math.Pi)
}
