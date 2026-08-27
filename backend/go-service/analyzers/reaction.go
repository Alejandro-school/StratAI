package analyzers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"math"
	"sort"
	"sync"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

const maxWorkers = 6

// Estructura para jobs de visibility check
type visibilityJob struct {
	shooterEyes r3.Vector
	enemyCenter r3.Vector
	enemyHead   r3.Vector
	shooterID   uint64
	enemyID     uint64
}

type visibilityResult struct {
	shooterID uint64
	enemyID   uint64
	isVisible bool
	raycasts  int
}

func sortVisibilityResults(results []visibilityResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].shooterID != results[j].shooterID {
			return results[i].shooterID < results[j].shooterID
		}
		return results[i].enemyID < results[j].enemyID
	})
}

func selectReactionTarget(
	firstSeen map[uint64]models.FirstSeenData,
	visible map[uint64]bool,
) (uint64, bool) {
	var targetEnemyID uint64
	bestCrosshairError := math.MaxFloat64
	found := false
	for enemyID, data := range firstSeen {
		if data.FirstShotTick != 0 || !visible[enemyID] {
			continue
		}
		error := data.CrosshairPlacementError
		if error < bestCrosshairError ||
			(found && error == bestCrosshairError && enemyID < targetEnemyID) {
			bestCrosshairError = error
			targetEnemyID = enemyID
			found = true
		}
	}
	return targetEnemyID, found
}

// RegisterReactionAnalyzer registra el analizador de reaction time
func RegisterReactionAnalyzer(ctx *models.DemoContext) {
	history := newVisibilityHistory(visibilitySampleStride)

	// Detectar cuando un enemigo se vuelve visible
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		currentTick := ctx.Parser.GameState().IngameTick()

		if !ctx.InRound || ctx.ActualRoundNumber == 0 {
			history.clear()
			clear(ctx.EnemyFirstSeenTick)
			clear(ctx.LastVisibleEnemies)
			return
		}

		if ctx.MapManager == nil || !ctx.MapManager.IsLoaded() {
			history.clear()
			if currentTick%visibilitySampleStride == 0 {
				ctx.VisibilitySkippedTicks++
			}
			clear(ctx.EnemyFirstSeenTick)
			clear(ctx.LastVisibleEnemies)
			return
		}
		history.add(captureVisibilityFrame(ctx, currentTick))
		if currentTick%visibilitySampleStride != 0 {
			return
		}
		ctx.VisibilitySampledTicks++
		jiggleGraceWindow := jiggleGraceTicks(ctx)

		// PARALELIZACIÓN: Preparar jobs de visibility checking
		var jobs []visibilityJob

		for _, shooter := range ctx.Parser.GameState().Participants().Playing() {
			if shooter.SteamID64 == 0 || !shooter.IsAlive() {
				continue
			}

			shooterID := shooter.SteamID64
			shooterTeam := shooter.Team
			shooterEyes, nativeEyes := playerstate.EyePosition(shooter)
			if nativeEyes {
				ctx.NativeEyePositions++
			} else {
				ctx.FallbackEyePositions++
			}

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
			if shooter.IsBlinded() {
				clear(ctx.EnemyFirstSeenTick[shooterID])
				clear(ctx.LastVisibleEnemies[shooterID])
				continue
			}
			shooterState := visibilityStateFromPlayer(ctx, shooter, currentTick)
			shooterState.eyes = shooterEyes
			shooterFrustum := playerViewFrustum(shooterState)

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

				enemyHead, _ := playerstate.EyePosition(enemy)
				if !shooterFrustum.contains(shooterEyes, enemyHead) &&
					!shooterFrustum.contains(shooterEyes, enemyCenter) {
					ctx.LastVisibleEnemies[shooterID][enemyID] = false
					continue
				}

				if smokeBlocksSight(ctx.ActiveSmokes, shooterEyes, enemyHead) &&
					smokeBlocksSight(ctx.ActiveSmokes, shooterEyes, enemyCenter) {
					ctx.LastVisibleEnemies[shooterID][enemyID] = false
					continue
				}

				// Añadir job a la cola
				jobs = append(jobs, visibilityJob{
					shooterEyes: shooterEyes,
					enemyCenter: enemyCenter,
					enemyHead:   enemyHead,
					shooterID:   shooterID,
					enemyID:     enemyID,
				})
			}
		}

		// Procesar jobs en paralelo con worker pool limitado
		var wg sync.WaitGroup
		jobsChan := make(chan visibilityJob, len(jobs))
		resultsChan := make(chan visibilityResult, len(jobs))

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
					raycasts := 1
					isVisibleChest := false
					if !isVisibleHead {
						isVisibleChest = ctx.MapManager.IsVisible(job.shooterEyes, job.enemyCenter)
						raycasts++
					}
					isVisible := isVisibleHead || isVisibleChest

					resultsChan <- visibilityResult{
						shooterID: job.shooterID,
						enemyID:   job.enemyID,
						isVisible: isVisible,
						raycasts:  raycasts,
					}
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

		results := make([]visibilityResult, 0, len(jobs))
		for result := range resultsChan {
			results = append(results, result)
		}
		sortVisibilityResults(results)

		// Procesar resultados y actualizar estado
		for _, result := range results {
			ctx.VisibilityRaycasts += result.raycasts
			wasVisible := ctx.LastVisibleEnemies[result.shooterID][result.enemyID]

			if result.isVisible {
				if wasVisible {
					if data, ok := ctx.EnemyFirstSeenTick[result.shooterID][result.enemyID]; ok {
						data.LastSeenTick = currentTick
						ctx.EnemyFirstSeenTick[result.shooterID][result.enemyID] = data
					}
				} else {
					currentFrame := history.frames[len(history.frames)-1]
					recordVisiblePair(ctx, history, currentFrame, result.shooterID, result.enemyID)
				}
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
					delete(ctx.LastVisibleEnemies[shooterID], enemyID)
					continue
				}

				// 2. Si ha pasado mucho tiempo desde el último avistamiento (Grace Period Exceeded)
				// 64 ticks = 500ms. Si lleva > 500ms oculto, asumimos que el encuentro terminó.
				if currentTick-data.LastSeenTick > jiggleGraceWindow {
					delete(enemiesMap, enemyID)
					delete(ctx.LastVisibleEnemies[shooterID], enemyID)
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
		refineShotVisibility(ctx, history, shooterID, currentTick)

		// Verificar si hay un enemigo visible recién detectado
		if ctx.EnemyFirstSeenTick[shooterID] != nil {
			targetEnemyID, found := selectReactionTarget(
				ctx.EnemyFirstSeenTick[shooterID],
				ctx.LastVisibleEnemies[shooterID],
			)
			if !found {
				return
			}

			if firstSeenData, ok := ctx.EnemyFirstSeenTick[shooterID][targetEnemyID]; ok {
				ticksSinceVisible := currentTick - firstSeenData.Tick

				tickRate := ctx.Parser.TickRate()
				if tickRate <= 0 {
					tickRate = 64
				}
				if ticksSinceVisible >= 0 && ticksSinceVisible <= int(2.5*tickRate) {
					tickInterval := 1000.0 / tickRate
					reactionTimeMs := int(float64(ticksSinceVisible) * tickInterval)

					// Obtener enemy player para calcular metadata
					var enemy *common.Player
					for _, p := range ctx.Parser.GameState().Participants().Playing() {
						if p.SteamID64 == targetEnemyID {
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
						wasFlashed = e.Shooter.IsBlinded()
						flashDuration = float32(e.Shooter.FlashDurationTimeRemaining().Seconds())

						// Posiciones
						shooterPos, _ := playerstate.EyePosition(e.Shooter)
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
						Round:                    ctx.CurrentRound,
						EnemyID:                  targetEnemyID,
						FirstSeenTick:            firstSeenData.Tick,
						FirstShotTick:            currentTick,
						ReactionTimeMs:           reactionTimeMs,
						WasFlashed:               wasFlashed,
						FlashDuration:            flashDuration,
						SmokeInPath:              smokeInPath,
						Distance:                 distance,
						CrosshairPlacementError:  firstSeenData.CrosshairPlacementError,
						PitchError:               firstSeenData.PitchError,
						YawError:                 firstSeenData.YawError,
						ShooterVelocity:          firstSeenData.ShooterVelocity,
						ShooterVelocityAvailable: firstSeenData.ShooterVelocityAvailable,
					}
					if firstSeenData.FirstDamageTick >= firstSeenData.Tick {
						reactionEvent.TimeToDamage = float64(firstSeenData.FirstDamageTick-firstSeenData.Tick) * tickInterval
					}
					playerData.ReactionTimes = append(playerData.ReactionTimes, reactionEvent)
					firstSeenData.FirstShotTick = currentTick
					ctx.EnemyFirstSeenTick[shooterID][targetEnemyID] = firstSeenData

					// Limpiar tracking para este enemigo específico
					// REMOVED: Do NOT delete here. We need this data for PlayerHurt (FirstDamageTick) calculation.
					// It will be cleaned up by the FrameDone "grace period" check.
					// delete(ctx.EnemyFirstSeenTick[shooterID], targetEnemyID)
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
		refineDamageVisibility(ctx, history, attackerID, victimID, currentTick)
		recordFirstDamageReaction(ctx, attackerID, victimID, currentTick)
		EnrichCapturedCombatReaction(ctx, CombatReactionEventKey{
			Round:             ctx.ActualRoundNumber,
			Tick:              currentTick,
			AttackerSteamID:   attackerID,
			VictimSteamID:     victimID,
			Weapon:            e.Weapon.String(),
			Hitgroup:          CombatHitgroup(e.HitGroup),
			Damage:            e.HealthDamageTaken,
			VictimHealthAfter: e.Health,
			VictimArmorAfter:  e.Armor,
		})
	})

	// Handler de Kill events para validar reaction times con datos oficiales
	ctx.Parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer == nil || e.Victim == nil {
			return
		}

		killerID := e.Killer.SteamID64
		victimID := e.Victim.SteamID64
		currentTick := ctx.Parser.GameState().IngameTick()
		refineDamageVisibility(ctx, history, killerID, victimID, currentTick)
		tickRate := ctx.Parser.TickRate()
		if tickRate <= 0 {
			tickRate = 64
		}

		// Buscar si hay un reaction time reciente para esta kill
		if playerData, exists := ctx.MatchData.Players[killerID]; exists {
			for i := len(playerData.ReactionTimes) - 1; i >= 0; i-- {
				rt := &playerData.ReactionTimes[i]

				if rt.EnemyID == victimID && float64(currentTick-rt.FirstShotTick) <= 2.5*tickRate {
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
		hitgroup := "chest"
		if e.IsHeadshot {
			hitgroup = "head"
		}
		EnrichCapturedCombatReaction(ctx, CombatReactionEventKey{
			Round:           ctx.ActualRoundNumber,
			Tick:            currentTick,
			AttackerSteamID: killerID,
			VictimSteamID:   victimID,
			IsKill:          true,
			Weapon:          e.Weapon.String(),
			Hitgroup:        hitgroup,
		})
	})
}

// Helper function
func distancePointToSegment(p, a, b r3.Vector) float64 {
	ab := b.Sub(a)
	if ab.Dot(ab) == 0 {
		return p.Sub(a).Norm()
	}
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

func smokeBlocksSight(smokes map[int64]r3.Vector, from, to r3.Vector) bool {
	for _, smoke := range smokes {
		if distancePointToSegment(smoke, from, to) < 144 {
			return true
		}
	}
	return false
}

func anglesToR3Vector(pitch, yaw float32) r3.Vector {
	// Normalize pitch from demoinfocs range (270 to 90, where 270 = -90) to standard -90 to 90
	// demoinfocs: 270° = looking up (-90°), 90° = looking down (+90°), 0° = horizontal
	p := float64(pitch)
	if p > 180 {
		p = p - 360 // Convert 270 → -90, 315 → -45, etc.
	}
	p = p * math.Pi / 180.0

	y := float64(yaw) * math.Pi / 180.0
	sinP := math.Sin(p)
	cosP := math.Cos(p)
	sinY := math.Sin(y)
	cosY := math.Cos(y)
	// Source uses positive pitch for looking down while world Z points up.
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
