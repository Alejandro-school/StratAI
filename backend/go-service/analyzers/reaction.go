package analyzers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/maps"
	"fmt"
	"sync"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// Worker pool para paralelizar raycasts - límite de 6 workers (75% de 8 cores)
const maxWorkers = 6

// Cache de raycasts para evitar cálculos repetidos
// Key: hash de posiciones (shooter_x_y_z + enemy_x_y_z)
// Value: resultado del raycast + tick de expiración
type raycastCacheEntry struct {
	isVisible  bool
	expireTick int
}

var (
	raycastCache      = make(map[string]raycastCacheEntry)
	raycastCacheMutex sync.RWMutex
	cacheTTL          = 8 // Cachear por 8 ticks (~62ms en 128tick)
)

// Estructura para jobs de visibility check
type visibilityJob struct {
	shooter     *common.Player
	enemy       *common.Player
	shooterEyes r3.Vector
	enemyCenter r3.Vector
	shooterID   uint64
	enemyID     uint64
	currentTick int
}

// Genera key única para posiciones (redondeo a 10 units para tolerar micro-movimientos)
func getCacheKey(shooterPos, enemyPos r3.Vector) string {
	sx := int(shooterPos.X / 10)
	sy := int(shooterPos.Y / 10)
	sz := int(shooterPos.Z / 10)
	ex := int(enemyPos.X / 10)
	ey := int(enemyPos.Y / 10)
	ez := int(enemyPos.Z / 10)
	return fmt.Sprintf("%d_%d_%d_%d_%d_%d", sx, sy, sz, ex, ey, ez)
}

// RegisterReactionAnalyzer registra el analizador de reaction time
func RegisterReactionAnalyzer(ctx *models.DemoContext) {
	// Detectar cuando un enemigo se vuelve visible
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		currentTick := ctx.Parser.GameState().IngameTick()

		// Samplear cada 4 ticks (~31ms en 128tick) - balance entre precisión y rendimiento
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
				ctx.EnemyFirstSeenTick[shooterID] = make(map[uint64]int)
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

				// Añadir job a la cola
				jobs = append(jobs, visibilityJob{
					shooter:     shooter,
					enemy:       enemy,
					shooterEyes: shooterEyes,
					enemyCenter: enemyCenter,
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
					// Intentar obtener del cache primero
					cacheKey := getCacheKey(job.shooterEyes, job.enemyCenter)

					raycastCacheMutex.RLock()
					cached, exists := raycastCache[cacheKey]
					raycastCacheMutex.RUnlock()

					var isVisible bool

					if exists && cached.expireTick > job.currentTick {
						// Cache hit - reutilizar resultado
						isVisible = cached.isVisible
					} else {
						// Cache miss - hacer raycast real
						isVisible = ctx.MapManager.IsVisible(job.shooterEyes, job.enemyCenter)

						// Guardar en cache
						raycastCacheMutex.Lock()
						raycastCache[cacheKey] = raycastCacheEntry{
							isVisible:  isVisible,
							expireTick: job.currentTick + cacheTTL,
						}
						raycastCacheMutex.Unlock()
					}

					// Heuristic checks (siempre se ejecutan - dependen de smokes dinámicos)
					if isVisible {
						isVisible = maps.HeuristicIsVisible(job.shooter, job.enemy, ctx.ActiveSmokes)
					}

					resultsChan <- struct {
						shooterID uint64
						enemyID   uint64
						isVisible bool
					}{job.shooterID, job.enemyID, isVisible}
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

			if result.isVisible && !wasVisible {
				// Enemigo ACABA DE APARECER
				ctx.EnemyFirstSeenTick[result.shooterID][result.enemyID] = currentTick
			} else if !result.isVisible && wasVisible {
				// Enemigo DESAPARECIÓ - INVALIDAR tracking
				delete(ctx.EnemyFirstSeenTick[result.shooterID], result.enemyID)
			}

			// Actualizar estado
			ctx.LastVisibleEnemies[result.shooterID][result.enemyID] = result.isVisible
		}

		// Limpiar cache expirado cada 100 ticks (~780ms) para evitar memory leak
		if currentTick%100 == 0 {
			raycastCacheMutex.Lock()
			for key, entry := range raycastCache {
				if entry.expireTick <= currentTick {
					delete(raycastCache, key)
				}
			}
			raycastCacheMutex.Unlock()
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
			for enemyID, firstSeenTick := range ctx.EnemyFirstSeenTick[shooterID] {
				ticksSinceVisible := currentTick - firstSeenTick

				// Reaction time válido: entre 1 y 200 ticks (~8ms - 1562ms)
				// Rango amplio para capturar desde reacciones instantáneas hasta más lentas
				if ticksSinceVisible >= 1 && ticksSinceVisible <= 200 {
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
						Round:          ctx.CurrentRound,
						EnemyID:        enemyID,
						FirstSeenTick:  firstSeenTick,
						FirstShotTick:  currentTick,
						ReactionTimeMs: reactionTimeMs,
						WasFlashed:     wasFlashed,
						FlashDuration:  flashDuration,
						SmokeInPath:    smokeInPath,
						Distance:       distance,
					}
					playerData.ReactionTimes = append(playerData.ReactionTimes, reactionEvent)

					fmt.Printf("✅ Reaction Time: %s -> %dms (Dist:%.0f Flash:%v Smoke:%v)\n",
						e.Shooter.Name, reactionTimeMs, distance, wasFlashed, smokeInPath)

					// Limpiar tracking para este enemigo específico
					delete(ctx.EnemyFirstSeenTick[shooterID], enemyID)
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
						fmt.Printf("⚠️  Discrepancia Smoke: Kill dice ThroughSmoke pero RT no detectó - %s vs %s\n",
							e.Killer.Name, e.Victim.Name)
						rt.SmokeInPath = true
					}

					if e.AttackerBlind && !rt.WasFlashed {
						fmt.Printf("⚠️  Discrepancia Flash: Kill dice AttackerBlind pero RT no detectó - %s vs %s\n",
							e.Killer.Name, e.Victim.Name)
						rt.WasFlashed = true
					}

					// Log de wallbangs
					if e.PenetratedObjects > 0 {
						fmt.Printf("🧱 Wallbang: %s penetró %d objeto(s) para matar a %s (RT: %dms)\n",
							e.Killer.Name, e.PenetratedObjects, e.Victim.Name, rt.ReactionTimeMs)
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
