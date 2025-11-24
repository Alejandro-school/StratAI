package demo

import (
	"fmt"

	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// ========== HELPER: Finalizar spray y calcular calidad ==========
func finalizeSpray(ctx *DemoContext, sid uint64, ps *models.PlayerStats, spray *models.RecoilSpray) {
	if spray.ShotCount < 3 {
		// No consideramos sprays de menos de 3 disparos
		return
	}

	// Calcular promedios
	spray.AvgPitchPerShot = spray.TotalPitchDelta / float32(spray.ShotCount-1)
	spray.AvgYawPerShot = spray.TotalYawDelta / float32(spray.ShotCount-1)

	// Determinar calidad basada en control del recoil
	// Menor delta = mejor control
	avgTotal := spray.AvgPitchPerShot + spray.AvgYawPerShot

	if avgTotal < 2.0 {
		spray.ControlQuality = "excellent"
		ps.ExcellentSprays++
	} else if avgTotal < 5.0 {
		spray.ControlQuality = "good"
	} else if avgTotal < 10.0 {
		spray.ControlQuality = "fair"
	} else {
		spray.ControlQuality = "poor"
		ps.PoorSprays++
	}

	ps.TotalSprays++
	ps.RecoilSprays = append(ps.RecoilSprays, *spray)
}

// ========== HELPER: Valor absoluto de float32 ==========
func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func registerPlayerHandlers(ctx *DemoContext) {
	// --- Kills ---
	ctx.parser.RegisterEventHandler(func(e events.Kill) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		currentTick := ctx.parser.GameState().IngameTick()
		killer := e.Killer
		victim := e.Victim

		weaponUsed := "unknown"
		if e.Weapon != nil {
			weaponUsed = e.Weapon.String()
		}

		// Manejo normal...
		if ctx.InRoundEndPeriod && victim != nil {
			ctx.DiedPostRound[victim.SteamID64] = true
		}
		if killer == nil && victim != nil {
			// WorldKill => muertes ambientales
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
				"WorldKill",
				fmt.Sprintf("Player %s died due to environmental causes (%s)", victim.Name, weaponUsed),
				"",
			))
			// Stats
			sidV := victim.SteamID64
			psV := getOrCreatePlayerStats(ctx, sidV, victim.Name)
			psV.Deaths++
			goto FINISH_KILL
		}
		if killer != nil && victim != nil && killer.SteamID64 == victim.SteamID64 {
			// Suicidio
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
				"Suicide",
				fmt.Sprintf("Player %s committed suicide using %s", victim.Name, weaponUsed),
				teamString(victim),
			))
		}

		// Lógica con killer
		if killer != nil {
			sidK := killer.SteamID64
			psK := getOrCreatePlayerStats(ctx, sidK, killer.Name)

			// TeamKill
			if victim != nil && killer.Team == victim.Team && killer.Team != common.TeamUnassigned {
				psK.TeamKillCount++
				ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
					"TeamKill",
					fmt.Sprintf("Player %s killed teammate %s", killer.Name, victim.Name),
					teamString(killer),
				))
			} else {
				// Kill normal
				if victim != nil {
					psK.Kills++

					// === NUEVO: Trackear weapon stats ===
					if psK.WeaponStats == nil {
						psK.WeaponStats = make(map[string]*models.WeaponStat)
					}
					if _, exists := psK.WeaponStats[weaponUsed]; !exists {
						psK.WeaponStats[weaponUsed] = &models.WeaponStat{}
					}
					psK.WeaponStats[weaponUsed].Kills++

					if e.IsHeadshot {
						psK.WeaponStats[weaponUsed].Headshots++
					}
				}
				if e.IsHeadshot {
					ctx.HeadshotsCount[sidK]++
				}
				if e.AssistedFlash && e.Assister != nil && e.Assister != killer {
					sidA := e.Assister.SteamID64
					psA := getOrCreatePlayerStats(ctx, sidA, e.Assister.Name)
					psA.FlashAssists++
				}
				if e.AttackerBlind {
					psK.BlindKills++
				}
				if e.NoScope {
					psK.NoScopeKills++
				}
				if e.ThroughSmoke {
					psK.ThroughSmokeKills++
				}
				if ctx.RoundFirstKill {
					psK.EntryKills++
					ctx.RoundFirstKill = false
				}
				lastKillTick, exists := ctx.LastTeamKillTick[int(killer.Team)]
				const tradeThresholdTicks = 2 * 128
				if exists && (currentTick-lastKillTick) <= tradeThresholdTicks {
					psK.TradeKills++
				}
				ctx.LastTeamKillTick[int(killer.Team)] = currentTick
				ctx.RoundPlayerKills[sidK]++

				// -- AÑADIMOS killReward:
				if e.Weapon != nil {
					reward := getKillRewardType(e.Weapon.Type)
					if econ, ok := ctx.RoundEconomyMap[ctx.RoundNumber][killer.SteamID64]; ok {
						econ.KillReward += reward
					}
				}
			}

			// Lógica de clutch
			gameState := ctx.parser.GameState()
			for _, team := range []common.Team{common.TeamTerrorists, common.TeamCounterTerrorists} {
				var alive []uint64
				for _, pl := range gameState.Participants().All() {
					if pl.Team == team && pl.IsAlive() {
						alive = append(alive, pl.SteamID64)
					}
				}
				if len(alive) == 1 {
					var opposingTeam common.Team
					if team == common.TeamTerrorists {
						opposingTeam = common.TeamCounterTerrorists
					} else {
						opposingTeam = common.TeamTerrorists
					}
					var enemyAlive []uint64
					for _, pl := range gameState.Participants().All() {
						if pl.Team == opposingTeam && pl.IsAlive() {
							enemyAlive = append(enemyAlive, pl.SteamID64)
						}
					}
					if len(enemyAlive) >= 2 {
						if team == common.TeamTerrorists && ctx.ClutchCandidateT == nil {
							ctx.ClutchCandidateT = &ClutchCandidate{
								SteamID:    alive[0],
								EnemyCount: len(enemyAlive),
							}
						} else if team == common.TeamCounterTerrorists && ctx.ClutchCandidateCT == nil {
							ctx.ClutchCandidateCT = &ClutchCandidate{
								SteamID:    alive[0],
								EnemyCount: len(enemyAlive),
							}
						}
					}
				}
			}
		}

		// Stats de la víctima
		if victim != nil {
			sidV := victim.SteamID64
			psV := getOrCreatePlayerStats(ctx, sidV, victim.Name)
			psV.Deaths++
			deathPos := victim.Position()

			// EventLog de muerte con estado capturado
			deathLog := newEventLog(ctx,
				"DeathPosition",
				fmt.Sprintf("Player %s died at (%.1f, %.1f, %.1f)", victim.Name, deathPos.X, deathPos.Y, deathPos.Z),
				teamString(victim),
			)

			ctx.EventLogs = append(ctx.EventLogs, deathLog)
		}

	FINISH_KILL:
		// Capturar estado completo para IA
		eventLog := newEventLog(ctx,
			"Kill",
			fmt.Sprintf("Killer=%s, Victim=%s, Weapon=%s, HS=%v, NoScope=%v, ThroughSmoke=%v, AttackerBlind=%v, Dist=%.2f",
				nameOrNil(killer), nameOrNil(victim), weaponUsed, e.IsHeadshot, e.NoScope, e.ThroughSmoke, e.AttackerBlind, e.Distance),
			teamString(killer),
		)
		eventLog.WeaponUsed = weaponUsed

		ctx.EventLogs = append(ctx.EventLogs, eventLog)
	})

	// --- PlayerHurt ---
	ctx.parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		if e.Attacker == nil || e.Attacker == e.Player {
			return
		}
		sidA := e.Attacker.SteamID64
		psA := getOrCreatePlayerStats(ctx, sidA, e.Attacker.Name)

		// Trackear daño de granadas HE y fuego
		weapon := e.Weapon
		if weapon != nil {
			weaponType := weapon.Type

			// Si el daño fue causado por HE
			if weaponType == common.EqHE {
				// Buscar la HE más reciente de este atacante
				for i := len(ctx.Grenades) - 1; i >= 0; i-- {
					g := &ctx.Grenades[i]
					if g.GrenadeType == "HE" && g.ThrowerSteamID == fmt.Sprintf("%d", sidA) && g.Round == ctx.RoundNumber {
						// Inicializar punteros si es necesario
						if g.TotalDamage == nil {
							zero := 0
							g.TotalDamage = &zero
							g.EnemiesDamaged = &zero
							g.TeammatesDamaged = &zero
						}

						*g.TotalDamage += int(e.HealthDamage)

						// Contar enemigos dañados vs aliados dañados (solo contar unique)
						if e.Player.Team == e.Attacker.Team {
							*g.TeammatesDamaged++
						} else {
							*g.EnemiesDamaged++
						}
						break
					}
				}
			}

			// Si el daño fue causado por fuego (molotov/incendiary)
			if weaponType == common.EqMolotov || weaponType == common.EqIncendiary {
				// Buscar el molotov/incendiary más reciente de este atacante
				for i := len(ctx.Grenades) - 1; i >= 0; i-- {
					g := &ctx.Grenades[i]
					if (g.GrenadeType == "Molotov" || g.GrenadeType == "Incendiary") &&
						g.ThrowerSteamID == fmt.Sprintf("%d", sidA) && g.Round == ctx.RoundNumber {
						// Inicializar puntero si es necesario
						if g.FireDamage == nil {
							zero := 0
							g.FireDamage = &zero
						}
						*g.FireDamage += int(e.HealthDamage)
						break
					}
				}
			}
		}

		if e.Attacker.Team == e.Player.Team && e.Attacker.Team != common.TeamUnassigned {
			psA.TeamDamage += int(e.HealthDamage)
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
				"TeamDamage",
				fmt.Sprintf("%s dealt %d dmg to teammate %s", e.Attacker.Name, e.HealthDamage, e.Player.Name),
				teamString(e.Attacker),
			))
		} else {
			ctx.DamageDone[sidA] += float64(e.HealthDamage)

			// Utility damage tracking eliminado - simplificado para que la IA lo aprenda

			psA.ShotsConnected++

			// === NUEVO: Trackear damage por arma ===
			if weapon != nil {
				weaponName := weapon.String()
				if psA.WeaponStats == nil {
					psA.WeaponStats = make(map[string]*models.WeaponStat)
				}
				if _, exists := psA.WeaponStats[weaponName]; !exists {
					psA.WeaponStats[weaponName] = &models.WeaponStat{}
				}
				psA.WeaponStats[weaponName].DamageDealt += int(e.HealthDamage)
				psA.WeaponStats[weaponName].ShotsHit++
			}

			// === NUEVO: First Shot Accuracy tracking ===
			// Si el disparo que causó el daño fue un "first shot", contar como hit
			if wasFirst, exists := ctx.LastShotWasFirst[sidA]; exists && wasFirst {
				psA.FirstShotHits++
				// Decrementar el miss que se contó en WeaponFire
				if psA.FirstShotMisses > 0 {
					psA.FirstShotMisses--
				}
				ctx.LastShotWasFirst[sidA] = false // Marcar como ya contado
			}
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
			"PlayerHurt",
			fmt.Sprintf("Player %s took %d dmg from %s (newHP=%d, newArmor=%d)",
				e.Player.Name, e.HealthDamage, e.Attacker.Name, e.Player.Health(), e.Player.Armor()),
			teamString(e.Attacker),
		))
	})

	// --- BulletDamage ---
	// ELIMINADO: Evento redundante con PlayerHurt, genera demasiado ruido para IA
	// ctx.parser.RegisterEventHandler(func(e events.BulletDamage) {
	// 	...
	// })

	// --- WeaponFire ---
	ctx.parser.RegisterEventHandler(func(e events.WeaponFire) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		if e.Shooter == nil {
			return
		}

		if e.Weapon != nil && e.Weapon.Type == common.EqKnife {
			// Omitimos el log de WeaponFire para el cuchillo
			return
		}

		sid := e.Shooter.SteamID64
		ps := getOrCreatePlayerStats(ctx, sid, e.Shooter.Name)
		ps.ShotsFired++

		// === NUEVO: Trackear shots fired por arma ===
		weaponName := e.Weapon.String()
		if ps.WeaponStats == nil {
			ps.WeaponStats = make(map[string]*models.WeaponStat)
		}
		if _, exists := ps.WeaponStats[weaponName]; !exists {
			ps.WeaponStats[weaponName] = &models.WeaponStat{}
		}
		ps.WeaponStats[weaponName].ShotsFired++

		currentTick := ctx.parser.GameState().IngameTick()

		// === NUEVO: First Shot Tracking ===
		// Considerar "first shot" si han pasado >128 ticks (~1s) desde último disparo
		const firstShotThreshold = 128
		isFirstShot := false
		if lastTick, exists := ctx.LastWeaponFireTick[sid]; !exists || (currentTick-lastTick) > firstShotThreshold {
			isFirstShot = true
			// Incrementar miss por defecto, se decrementará si hay hit en PlayerHurt
			ps.FirstShotMisses++
		}
		ctx.LastWeaponFireTick[sid] = currentTick
		ctx.LastShotWasFirst[sid] = isFirstShot

		// === NUEVO: Capturar contexto del disparo para análisis de aim ===
		velocity := ctx.LastSpeed[sid]
		wasMoving := velocity > 50.0 // Threshold para considerar "en movimiento"
		wasScoped := false
		if wpn := e.Shooter.ActiveWeapon(); wpn != nil {
			wasScoped = wpn.ZoomLevel() > 0
		}

		mapName := ctx.parser.Header().MapName
		zone := GetZoneForPosition(mapName, e.Shooter.Position())

		shotContext := models.ShotContext{
			Tick:       ctx.parser.GameState().IngameTick(),
			Weapon:     e.Weapon.String(),
			Velocity:   velocity,
			WasMoving:  wasMoving,
			WasDucking: e.Shooter.IsDucking(),
			WasScoped:  wasScoped,
			Zone:       zone,
		}

		// === NUEVO: Recoil Analysis - Trackear view angles ===
		pitch := e.Shooter.ViewDirectionX()
		yaw := calculateYawFromViewDirection(e.Shooter)
		shotContext.ViewPitch = pitch
		shotContext.ViewYaw = yaw

		// Detectar spray: disparos consecutivos con el mismo arma en menos de 500ms
		weaponFireTick := ctx.parser.GameState().IngameTick()
		const sprayWindow = 64 // ~500ms en 128 tick/s

		if spray, exists := ctx.CurrentSpray[sid]; exists {
			// Continúa spray existente si:
			// 1. Misma arma
			// 2. Menos de 500ms desde último disparo
			if spray.Weapon == e.Weapon.String() && (weaponFireTick-spray.EndTick) < sprayWindow {
				// Calcular deltas
				lastPitch := ctx.LastShotPitch[sid]
				lastYaw := ctx.LastShotYaw[sid]

				pitchDelta := abs32(pitch - lastPitch)
				yawDelta := abs32(yaw - lastYaw)

				spray.ShotCount++
				spray.EndTick = weaponFireTick
				spray.TotalPitchDelta += pitchDelta
				spray.TotalYawDelta += yawDelta
			} else {
				// Finalizar spray anterior y empezar uno nuevo
				finalizeSpray(ctx, sid, ps, spray)

				// Iniciar nuevo spray
				ctx.CurrentSpray[sid] = &models.RecoilSpray{
					Round:     ctx.RoundNumber,
					StartTick: weaponFireTick,
					EndTick:   weaponFireTick,
					ShotCount: 1,
					Weapon:    e.Weapon.String(),
				}
			}
		} else {
			// Iniciar nuevo spray
			ctx.CurrentSpray[sid] = &models.RecoilSpray{
				Round:     ctx.RoundNumber,
				StartTick: weaponFireTick,
				EndTick:   weaponFireTick,
				ShotCount: 1,
				Weapon:    e.Weapon.String(),
			}
		}

		// Actualizar último pitch/yaw
		ctx.LastShotPitch[sid] = pitch
		ctx.LastShotYaw[sid] = yaw

		// Guardar en PlayerStats para análisis agregado
		ps.Shots = append(ps.Shots, shotContext)

		// === NUEVO: Reaction Time - Tiempo desde que vio enemigo hasta que disparó ===
		if seenEnemies, exists := ctx.EnemyFirstSeenTick[sid]; exists {
			currentTick := ctx.parser.GameState().IngameTick()

			// Inicializar mapa de registrados si no existe
			if _, exists := ctx.ReactionRegistered[sid]; !exists {
				ctx.ReactionRegistered[sid] = make(map[uint64]bool)
			}

			// Buscar si algún enemigo visible fue visto recientemente
			for enemyID, firstSeenTick := range seenEnemies {
				// Saltar si ya registramos este evento de reacción
				if ctx.ReactionRegistered[sid][enemyID] {
					continue
				}

				// Ventana de reacción: 3 segundos (384 ticks a 128 tick/s)
				const reactionWindow = 384
				timeSinceSeen := currentTick - firstSeenTick

				if timeSinceSeen <= reactionWindow && timeSinceSeen > 0 {
					// Calcular tiempo de reacción en ms
					reactionTimeMs := int(float64(timeSinceSeen) / ctx.parser.TickRate() * 1000)

					// Buscar el enemigo para calcular distancia
					var enemy *common.Player
					for _, pl := range ctx.parser.GameState().Participants().All() {
						if pl.SteamID64 == enemyID {
							enemy = pl
							break
						}
					}

					distance := 0.0
					if enemy != nil {
						distance = vectorDistance(e.Shooter.Position(), enemy.Position())
					}

					// Guardar evento de reaction time
					reactionEvent := models.ReactionTimeEvent{
						Round:           ctx.RoundNumber,
						EnemyID:         enemyID,
						FirstSeenTick:   firstSeenTick,
						FirstShotTick:   currentTick,
						ReactionTimeMs:  reactionTimeMs,
						DistanceToEnemy: distance,
						KilledEnemy:     false, // Se puede determinar después si fue kill
					}

					ps.ReactionTimes = append(ps.ReactionTimes, reactionEvent)
					ps.TotalReactionTimeMs += reactionTimeMs
					ps.ReactionTimeCount++

					// Marcar como registrado para no contar múltiples veces
					ctx.ReactionRegistered[sid][enemyID] = true

					break // Solo un enemigo por disparo
				}
			}
		}

		// EventLog con contexto
		eventLog := newEventLog(ctx,
			"WeaponFire",
			fmt.Sprintf("Player %s fired %s (vel=%.1f, moving=%v)", e.Shooter.Name, e.Weapon.String(), velocity, wasMoving),
			teamString(e.Shooter),
		)
		eventLog.ShotContext = &shotContext

		ctx.EventLogs = append(ctx.EventLogs, eventLog)
	})

	// --- WeaponReload ---
	// ELIMINADO: Genera ruido masivo sin valor para IA
	// Podemos derivar reloads críticos del análisis de ticks entre disparos
	// ctx.parser.RegisterEventHandler(func(e events.WeaponReload) {
	// 	...
	// })

	// --------------------------------------------------
	// ITEM PICKUP => actualizamos SpentInBuy (si aplica)
	// --------------------------------------------------
	// players_handlers.go
	ctx.parser.RegisterEventHandler(func(e events.ItemPickup) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		if e.Player == nil || e.Weapon == nil {
			return
		}
		sid := e.Player.SteamID64
		eqType := e.Weapon.Type

		econMapForRound, ok := ctx.RoundEconomyMap[ctx.RoundNumber]
		if !ok {
			return
		}
		econ, ok := econMapForRound[sid]
		if !ok {
			return
		}

		// Ignoramos cuchillo, armas "world", etc.
		if eqType == common.EqKnife ||
			eqType == common.EqWorld ||
			eqType == common.EqUnknown {
			return
		}

		cost := getWeaponPrice(e.Weapon.Type)

		// Si no tiene precio => simple "pick_up" del suelo.
		if cost <= 0 {
			ctx.LastKnownMoney[sid] = e.Player.Money()
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
				"ItemPickup",
				fmt.Sprintf("Player %s picked_up %s", e.Player.Name, e.Weapon.String()),
				teamString(e.Player),
			))
			return
		}

		// Obtenemos el dinero antes de la acción (o del freeze time)
		lastMoney, exists := ctx.LastKnownMoney[sid]
		if !exists {
			lastMoney = econ.InitialMoney
		}

		currentMoney := e.Player.Money()
		delta := lastMoney - currentMoney

		// Si el delta >= cost => se considera compra real
		if delta >= cost {
			econ.SpentInBuy += cost

			// ¡Añadimos a la lista de compras!
			econ.Purchases = append(econ.Purchases, models.RoundItem{
				Name:  e.Weapon.String(),
				Price: cost,
			})

			// Incrementamos conteo de compras en PlayerStats (si te interesa llevar track aquí).
			ps := getOrCreatePlayerStats(ctx, sid, e.Player.Name)
			ps.Purchases++

			// Ajustamos el "último dinero conocido" tras la compra
			ctx.LastKnownMoney[sid] = lastMoney - cost

			ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
				"ItemBought",
				fmt.Sprintf("Player %s bought %s [cost=%d]", e.Player.Name, e.Weapon.String(), cost),
				teamString(e.Player),
			))
		} else {
			// Caso: pick-up gratis (arma en el suelo, etc.)
			ctx.LastKnownMoney[sid] = currentMoney
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
				"ItemPickup",
				fmt.Sprintf("Player %s picked_up %s [cost=%d]", e.Player.Name, e.Weapon.String(), cost),
				teamString(e.Player),
			))
		}
	})

	// ----------------------------------------------
	// ITEM REFUND => restamos a SpentInBuy si aplica
	// ----------------------------------------------
	ctx.parser.RegisterEventHandler(func(e events.ItemRefund) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		if e.Player == nil || e.Weapon == nil {
			return
		}
		sid := e.Player.SteamID64

		econMapForRound, ok := ctx.RoundEconomyMap[ctx.RoundNumber]
		if !ok {
			return
		}
		econ, ok := econMapForRound[sid]
		if !ok {
			return
		}

		cost := getWeaponPrice(e.Weapon.Type)
		if cost <= 0 {
			return
		}

		currentTick := ctx.parser.GameState().IngameTick()
		buyWindowEnd := ctx.BuyWindowEndTickForRound[ctx.RoundNumber]

		// Si el reembolso ocurre dentro de la ventana de compra,
		// le restamos a SpentInBuy
		if currentTick <= buyWindowEnd {
			econ.SpentInBuy -= cost
			if econ.SpentInBuy < 0 {
				econ.SpentInBuy = 0 // Evitar valores negativos por seguridad
			}
		} else {
			// Si tuviéramos un SpentOutsideBuy, le restaríamos allí
		}

		// Registramos en logs
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(ctx,
			"ItemRefund",
			fmt.Sprintf("Player %s refunded %s (+%d)", e.Player.Name, e.Weapon.String(), cost),
			teamString(e.Player),
		))
	})
}

// getWeaponPrice obtiene el precio de un arma directamente del map
func getWeaponPrice(eqType common.EquipmentType) int {
	return weaponPrices[eqType]
}

// getOrCreatePlayerStats busca PlayerStats en el mapa; si no existe, lo crea.
func getOrCreatePlayerStats(ctx *DemoContext, sid uint64, name string) *models.PlayerStats {
	ps, exists := ctx.PlayerStatsMap[sid]
	if !exists {
		ps = &models.PlayerStats{Name: name}
		ctx.PlayerStatsMap[sid] = ps
	} else if ps.Name == "" && name != "" {
		ps.Name = name
	}
	return ps
}
