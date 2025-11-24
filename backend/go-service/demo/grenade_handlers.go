package demo

import (
	"cs2-demo-service/models"
	"fmt"
	"math"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// registerGrenadeHandlers registra los event handlers para capturar granadas
// Según la documentación oficial de demoinfocs-golang:
// - GrenadeProjectileThrow: Se dispara cuando se LANZA una granada (crea la entidad)
// - GrenadeProjectileDestroy: Se dispara cuando se DESTRUYE/DETONA (útil para trayectoria completa)
// - FlashExplode: Cuando explota un flash
// - PlayerFlashed: Cuando un jugador es cegado
// - HeExplode: Cuando explota una HE
// - SmokeStart: Cuando un smoke empieza a desplegarse
// - SmokeExpired: Cuando un smoke se disipa
// - FireGrenadeStart: Cuando un molotov/incendiary empieza a arder
// - FireGrenadeExpired: Cuando se apaga
func registerGrenadeHandlers(ctx *DemoContext) {
	// Mapa temporal para almacenar índices de granadas por EntityID
	grenadeIndexByEntity := make(map[int]int)

	// Mapas para relacionar eventos de efectividad con granadas
	flashEventsByThrower := make(map[uint64]int)   // throwerSteamID -> grenadeIndex (último flash lanzado)
	heEventsByPosition := make(map[[3]float64]int) // posición aproximada -> grenadeIndex
	smokeEventsByPosition := make(map[[3]float64]int)
	// Para infernos (molotov/incendiary)
	// - Mapeo por UniqueID del inferno -> índice de granada
	// - También guardamos metadatos del inferno al inicio para resolver condiciones de carrera
	fireIndexByInfernoID := make(map[int64]int)
	infernoStartTickByID := make(map[int64]int)
	infernoCenterByID := make(map[int64][3]float64)

	throwCount := 0
	destroyCount := 0

	// Evento 1: Capturar cuando se lanza la granada (TODAS las granadas)
	ctx.parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		projectile := e.Projectile
		if projectile == nil {
			return
		}

		// Ignorar granadas de la ronda 0 (antes de que empiece la partida real)
		if ctx.RoundNumber < 1 {
			return
		}

		throwCount++

		var throwerSteamID string
		var throwerName string
		var throwerTeam string
		var throwerDucking bool
		var throwerAirborne bool

		if projectile.Thrower != nil {
			throwerSteamID = fmt.Sprintf("%d", projectile.Thrower.SteamID64)
			throwerName = projectile.Thrower.Name
			throwerTeam = teamToString(projectile.Thrower.Team)

			// Capturar estado del lanzador
			throwerDucking = projectile.Thrower.IsDucking()
			throwerAirborne = projectile.Thrower.IsAirborne()
		}

		grenadeType := grenadeTypeToString(projectile.WeaponInstance.Type)

		// Posición inicial del lanzamiento
		throwPos := projectile.Position()

		// Tick actual
		currentTick := ctx.parser.GameState().IngameTick()

		// Crear el evento de granada
		grenadeEvent := models.GrenadeEvent{
			Round:          ctx.RoundNumber,
			Tick:           currentTick,
			ThrowerSteamID: throwerSteamID,
			ThrowerName:    throwerName,
			ThrowerTeam:    throwerTeam,
			GrenadeType:    grenadeType,
			ThrowPosition: models.GrenadeTrajectoryPoint{
				X: throwPos.X,
				Y: throwPos.Y,
				Z: throwPos.Z,
			},
			ExplosionPosition: models.GrenadeTrajectoryPoint{
				X: 0,
				Y: 0,
				Z: 0,
			},
			ThrowerDucking:  throwerDucking,
			ThrowerAirborne: throwerAirborne,
		}

		// Añadir a la lista y guardar el índice
		grenadeIndex := len(ctx.Grenades)
		ctx.Grenades = append(ctx.Grenades, grenadeEvent)

		// Almacenar el índice para actualizar después
		if projectile.Entity != nil {
			grenadeIndexByEntity[projectile.Entity.ID()] = grenadeIndex
		}

		// Para flashes, guardar referencia por thrower para relacionar con PlayerFlashed
		if projectile.Thrower != nil && grenadeType == "Flash" {
			flashEventsByThrower[projectile.Thrower.SteamID64] = grenadeIndex
		}
	})

	// Evento 2: Actualizar con la trayectoria completa cuando se destruye
	ctx.parser.RegisterEventHandler(func(e events.GrenadeProjectileDestroy) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		projectile := e.Projectile
		if projectile == nil {
			return
		}

		// Ignorar granadas de la ronda 0
		if ctx.RoundNumber < 1 {
			return
		}

		destroyCount++

		// Buscar el índice de la granada en el slice
		grenadeIndex := -1
		if projectile.Entity != nil {
			idx, exists := grenadeIndexByEntity[projectile.Entity.ID()]
			if exists {
				grenadeIndex = idx
			}
		}

		// Si no encontramos el evento previo, puede ser que el Throw no se haya capturado
		// En ese caso, lo creamos aquí como fallback
		if grenadeIndex == -1 {
			var throwerSteamID string
			var throwerName string
			var throwerTeam string

			if projectile.Thrower != nil {
				throwerSteamID = fmt.Sprintf("%d", projectile.Thrower.SteamID64)
				throwerName = projectile.Thrower.Name
				throwerTeam = teamToString(projectile.Thrower.Team)
			}

			grenadeType := grenadeTypeToString(projectile.WeaponInstance.Type)

			// Use Trajectory2 (v4 library) instead of deprecated Trajectory
			trajectory := projectile.Trajectory2
			if len(trajectory) == 0 {
				return
			}

			throwPos := trajectory[0].Position
			explosionPos := trajectory[len(trajectory)-1].Position

			grenadeEvent := models.GrenadeEvent{
				Round:          ctx.RoundNumber,
				ThrowerSteamID: throwerSteamID,
				ThrowerName:    throwerName,
				ThrowerTeam:    throwerTeam,
				GrenadeType:    grenadeType,
				ThrowPosition: models.GrenadeTrajectoryPoint{
					X: throwPos.X,
					Y: throwPos.Y,
					Z: throwPos.Z,
				},
				ExplosionPosition: models.GrenadeTrajectoryPoint{
					X: explosionPos.X,
					Y: explosionPos.Y,
					Z: explosionPos.Z,
				},
			}

			ctx.Grenades = append(ctx.Grenades, grenadeEvent)
			// Asignar índice creado al entity si está disponible
			if projectile.Entity != nil {
				grenadeIndexByEntity[projectile.Entity.ID()] = len(ctx.Grenades) - 1
			}
		} else {
			// Actualizar el evento existente con la trayectoria completa
			trajectory := projectile.Trajectory2
			if len(trajectory) > 0 {
				// Actualizar posición de LANZAMIENTO (primer punto de la trayectoria)
				throwPos := trajectory[0].Position
				ctx.Grenades[grenadeIndex].ThrowPosition = models.GrenadeTrajectoryPoint{
					X: throwPos.X,
					Y: throwPos.Y,
					Z: throwPos.Z,
				}

				// Actualizar posición de EXPLOSIÓN (último punto de la trayectoria)
				explosionPos := trajectory[len(trajectory)-1].Position
				ctx.Grenades[grenadeIndex].ExplosionPosition = models.GrenadeTrajectoryPoint{
					X: explosionPos.X,
					Y: explosionPos.Y,
					Z: explosionPos.Z,
				}

				// Para Molotov/Incendiary, resolver posible condición de carrera:
				// El InfernoStart puede llegar ANTES de conocer la ExplosionPosition (destroy),
				// lo que impide el mapeo por proximidad en ese momento. Aquí, cuando ya
				// conocemos la posición de explosión, intentamos asociar esta granada
				// con el inferno más cercano que aún no esté mapeado.
				grenadeType := ctx.Grenades[grenadeIndex].GrenadeType
				if grenadeType == "Molotov" || grenadeType == "Incendiary" {
					// 1) Inicializar campos si no existen (para garantizar presencia en JSON)
					if ctx.Grenades[grenadeIndex].FireStartTick == nil {
						zeroTick := 0
						ctx.Grenades[grenadeIndex].FireStartTick = &zeroTick
					}
					if ctx.Grenades[grenadeIndex].FireExpiredTick == nil {
						zeroTick := 0
						ctx.Grenades[grenadeIndex].FireExpiredTick = &zeroTick
					}
					if ctx.Grenades[grenadeIndex].FireDurationMs == nil {
						zeroDuration := 0
						ctx.Grenades[grenadeIndex].FireDurationMs = &zeroDuration
					}

					// 2) Si aún no está mapeado a un inferno, buscar el inferno activo más cercano
					alreadyMapped := false
					for _, idx := range fireIndexByInfernoID {
						if idx == grenadeIndex {
							alreadyMapped = true
							break
						}
					}

					if !alreadyMapped {
						// Buscar el inferno con centro más cercano a la ExplosionPosition
						gx := ctx.Grenades[grenadeIndex].ExplosionPosition.X
						gy := ctx.Grenades[grenadeIndex].ExplosionPosition.Y
						gz := ctx.Grenades[grenadeIndex].ExplosionPosition.Z

						minDist := 1e12
						var bestInfernoID int64 = 0
						for infID, center := range infernoCenterByID {
							// Saltar infernos ya mapeados
							if _, exists := fireIndexByInfernoID[infID]; exists {
								continue
							}
							dx := center[0] - gx
							dy := center[1] - gy
							dz := center[2] - gz
							dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
							if dist < minDist {
								minDist = dist
								bestInfernoID = infID
							}
						}

						// Umbral de 300u como en el fallback del InfernoStart
						if bestInfernoID != 0 && minDist <= 300 {
							// Asociar y fijar el FireStartTick usando el tick de inicio del inferno
							if startTick, ok := infernoStartTickByID[bestInfernoID]; ok {
								ctx.Grenades[grenadeIndex].FireStartTick = &startTick
							}
							fireIndexByInfernoID[bestInfernoID] = grenadeIndex
						}
					}
				}
			}
		}
	})

	// Evento 3: Capturar cuando un jugador es cegado por un flash
	ctx.parser.RegisterEventHandler(func(e events.PlayerFlashed) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber < 1 {
			return
		}

		if e.Attacker == nil || e.Player == nil {
			return
		}

		// Buscar el último flash lanzado por este atacante
		grenadeIndex, exists := flashEventsByThrower[e.Attacker.SteamID64]
		if !exists {
			return
		}

		g := &ctx.Grenades[grenadeIndex]

		// Inicializar punteros si es la primera vez
		if g.EnemiesFlashed == nil {
			zero := 0
			g.EnemiesFlashed = &zero
			g.TeammatesFlashed = &zero
			avgZero := 0.0
			g.AvgFlashDuration = &avgZero
		}

		// Calcular duración del flash en segundos
		flashDuration := e.FlashDuration().Seconds()

		// Determinar si es enemigo o compañero
		if e.Player.Team == e.Attacker.Team {
			// Teamflash
			*g.TeammatesFlashed++
		} else {
			// Flash a enemigo
			*g.EnemiesFlashed++
		}

		// Actualizar duración promedio
		totalFlashed := *g.EnemiesFlashed + *g.TeammatesFlashed
		if totalFlashed > 0 {
			currentAvg := *g.AvgFlashDuration
			*g.AvgFlashDuration = (currentAvg*float64(totalFlashed-1) + flashDuration) / float64(totalFlashed)
		}
	})

	// Evento 4: Capturar cuando explota una HE
	ctx.parser.RegisterEventHandler(func(e events.HeExplode) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber < 1 {
			return
		}

		// Guardar posición de explosión para relacionar con el daño
		pos := e.Position
		posKey := [3]float64{
			math.Round(pos.X/10) * 10, // Redondear a 10 unidades para agrupar
			math.Round(pos.Y/10) * 10,
			math.Round(pos.Z/10) * 10,
		}

		// Buscar la granada HE más cercana a esta posición
		if e.Thrower != nil {
			for i := len(ctx.Grenades) - 1; i >= 0; i-- {
				g := &ctx.Grenades[i]
				if g.GrenadeType == "HE" && g.ThrowerSteamID == fmt.Sprintf("%d", e.Thrower.SteamID64) {
					// Verificar que sea de esta ronda
					if g.Round == ctx.RoundNumber {
						heEventsByPosition[posKey] = i
						break
					}
				}
			}
		}
	})

	// Evento 5: Capturar cuando se despliega un smoke
	ctx.parser.RegisterEventHandler(func(e events.SmokeStart) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber < 1 {
			return
		}

		pos := e.Position
		posKey := [3]float64{
			math.Round(pos.X/10) * 10,
			math.Round(pos.Y/10) * 10,
			math.Round(pos.Z/10) * 10,
		}

		currentTick := ctx.parser.GameState().IngameTick()

		// Buscar el smoke más cercano a esta posición
		if e.Thrower != nil {
			for i := len(ctx.Grenades) - 1; i >= 0; i-- {
				g := &ctx.Grenades[i]
				if g.GrenadeType == "Smoke" && g.ThrowerSteamID == fmt.Sprintf("%d", e.Thrower.SteamID64) {
					if g.Round == ctx.RoundNumber {
						g.SmokeStartTick = &currentTick
						smokeEventsByPosition[posKey] = i
						break
					}
				}
			}
		}
	})

	// Evento 6: Capturar cuando expira un smoke
	ctx.parser.RegisterEventHandler(func(e events.SmokeExpired) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber < 1 {
			return
		}

		pos := e.Position
		posKey := [3]float64{
			math.Round(pos.X/10) * 10,
			math.Round(pos.Y/10) * 10,
			math.Round(pos.Z/10) * 10,
		}

		currentTick := ctx.parser.GameState().IngameTick()

		// Buscar el smoke correspondiente
		if grenadeIndex, exists := smokeEventsByPosition[posKey]; exists {
			g := &ctx.Grenades[grenadeIndex]
			g.SmokeExpiredTick = &currentTick

			// Calcular duración en ms (asumiendo 128 tickrate)
			tickRate := ctx.parser.TickRate()
			if tickRate > 0 && g.SmokeStartTick != nil && *g.SmokeStartTick > 0 {
				durationTicks := currentTick - *g.SmokeStartTick
				durationMs := int(float64(durationTicks) / float64(tickRate) * 1000)
				g.SmokeDurationMs = &durationMs
			}
		}
	})

	// Evento 7: Capturar cuando empieza un fuego (molotov/incendiary)
	// Usamos InfernoStart en lugar de FireGrenadeStart porque es más confiable según la documentación
	ctx.parser.RegisterEventHandler(func(e events.InfernoStart) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber < 1 {
			return
		}

		if e.Inferno == nil {
			return
		}

		currentTick := ctx.parser.GameState().IngameTick()
		infernoID := e.Inferno.UniqueID()

		// Guardar metadatos del inferno para resolver carreras más tarde
		infernoStartTickByID[infernoID] = currentTick
		// Calcular y almacenar el centro del inferno desde sus fuegos
		cx, cy, cz := 0.0, 0.0, 0.0
		count := 0.0
		for _, f := range e.Inferno.Fires().List() {
			cx += f.Vector.X
			cy += f.Vector.Y
			cz += f.Vector.Z
			count += 1.0
		}
		if count > 0 {
			cx /= count
			cy /= count
			cz /= count
		}
		infernoCenterByID[infernoID] = [3]float64{cx, cy, cz}

		matchedIndex := -1

		// 1) Intentar por lanzador
		if thrower := e.Inferno.Thrower(); thrower != nil {
			for i := len(ctx.Grenades) - 1; i >= 0; i-- {
				g := &ctx.Grenades[i]
				if (g.GrenadeType == "Molotov" || g.GrenadeType == "Incendiary") && g.ThrowerSteamID == fmt.Sprintf("%d", thrower.SteamID64) {
					// Preferimos misma ronda pero si no coincide, igual vinculamos al más reciente
					matchedIndex = i
					if g.Round == ctx.RoundNumber {
						break
					}
				}
			}
		}

		// 2) Fallback por proximidad espacial al centro del inferno (si no hubo match por lanzador)
		if matchedIndex == -1 {
			center := infernoCenterByID[infernoID]
			cx, cy, cz := center[0], center[1], center[2]
			// Buscar la granada (Molotov/Incendiary) más cercana por posición de explosión
			minDist := 1e12
			for i := len(ctx.Grenades) - 1; i >= 0; i-- {
				g := &ctx.Grenades[i]
				if g.GrenadeType != "Molotov" && g.GrenadeType != "Incendiary" {
					continue
				}
				dx := g.ExplosionPosition.X - cx
				dy := g.ExplosionPosition.Y - cy
				dz := g.ExplosionPosition.Z - cz
				dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if dist < minDist {
					minDist = dist
					matchedIndex = i
				}
				// Si está muy cerca, paramos pronto
				if dist < 50 {
					break
				}
			}
			// Umbral de distancia razonable para asociar (en unidades del mundo)
			if minDist > 300 {
				matchedIndex = -1
			}
		}

		if matchedIndex != -1 {
			g := &ctx.Grenades[matchedIndex]
			g.FireStartTick = &currentTick
			fireIndexByInfernoID[infernoID] = matchedIndex
		}
	})

	// Evento 8: Capturar cuando expira un fuego
	// Usamos InfernoExpired en lugar de FireGrenadeExpired
	ctx.parser.RegisterEventHandler(func(e events.InfernoExpired) {
		if ctx.parser.GameState().IsWarmupPeriod() || ctx.RoundNumber < 1 {
			return
		}

		if e.Inferno == nil {
			return
		}

		currentTick := ctx.parser.GameState().IngameTick()
		infernoID := e.Inferno.UniqueID()

		// Buscar el fuego correspondiente por su UniqueID
		if grenadeIndex, exists := fireIndexByInfernoID[infernoID]; exists {
			g := &ctx.Grenades[grenadeIndex]
			g.FireExpiredTick = &currentTick

			// Calcular duración en ms
			tickRate := ctx.parser.TickRate()
			if tickRate > 0 && g.FireStartTick != nil && *g.FireStartTick > 0 {
				durationTicks := currentTick - *g.FireStartTick
				durationMs := int(float64(durationTicks) / float64(tickRate) * 1000)
				g.FireDurationMs = &durationMs
			}

			// Limpiar metadatos del inferno para evitar fugas
			delete(infernoStartTickByID, infernoID)
			delete(infernoCenterByID, infernoID)
			delete(fireIndexByInfernoID, infernoID)
		}
	})

	// Log final al terminar el parsing
	ctx.parser.RegisterEventHandler(func(e events.RoundEnd) {
		fmt.Printf("📊 Estadísticas de granadas - Lanzamientos: %d, Destrucciones: %d\n", throwCount, destroyCount)
	})
}

func grenadeTypeToString(eqType common.EquipmentType) string {
	switch eqType {
	case common.EqFlash:
		return "Flash"
	case common.EqSmoke:
		return "Smoke"
	case common.EqHE:
		return "HE"
	case common.EqMolotov:
		return "Molotov"
	case common.EqIncendiary:
		return "Incendiary"
	case common.EqDecoy:
		return "Decoy"
	default:
		return "Unknown"
	}
}

func teamToString(team common.Team) string {
	switch team {
	case common.TeamTerrorists:
		return "T"
	case common.TeamCounterTerrorists:
		return "CT"
	case common.TeamSpectators:
		return "Spectators"
	case common.TeamUnassigned:
		return "Unassigned"
	default:
		return "Unknown"
	}
}
