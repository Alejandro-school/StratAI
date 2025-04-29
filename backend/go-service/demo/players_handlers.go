package demo

import (
	"fmt"

	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

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
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(
				ctx.RoundNumber,
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
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(
				ctx.RoundNumber,
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
				ctx.EventLogs = append(ctx.EventLogs, newEventLog(
					ctx.RoundNumber,
					"TeamKill",
					fmt.Sprintf("Player %s killed teammate %s", killer.Name, victim.Name),
					teamString(killer),
				))
			} else {
				// Kill normal
				if victim != nil {
					psK.Kills++
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
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(
				ctx.RoundNumber,
				"DeathPosition",
				fmt.Sprintf("Player %s died at (%.1f, %.1f, %.1f)", victim.Name, deathPos.X, deathPos.Y, deathPos.Z),
				teamString(victim),
			))
		}

	FINISH_KILL:
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"Kill",
			fmt.Sprintf("Killer=%s, Victim=%s, Weapon=%s, HS=%v, NoScope=%v, ThroughSmoke=%v, AttackerBlind=%v, Dist=%.2f",
				nameOrNil(killer), nameOrNil(victim), weaponUsed, e.IsHeadshot, e.NoScope, e.ThroughSmoke, e.AttackerBlind, e.Distance),
			teamString(killer),
		))
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

		if e.Attacker.Team == e.Player.Team && e.Attacker.Team != common.TeamUnassigned {
			psA.TeamDamage += int(e.HealthDamage)
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(
				ctx.RoundNumber,
				"TeamDamage",
				fmt.Sprintf("%s dealt %d dmg to teammate %s", e.Attacker.Name, e.HealthDamage, e.Player.Name),
				teamString(e.Attacker),
			))
		} else {
			ctx.DamageDone[sidA] += float64(e.HealthDamage)
		}

		psA.ShotsConnected++

		if e.Weapon != nil {
			wt := e.Weapon.Type
			if wt == common.EqMolotov || wt == common.EqIncendiary || wt == common.EqHE {
				psA.UtilityDamage += int(e.HealthDamage)
			}
		}
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"PlayerHurt",
			fmt.Sprintf("Player %s took %d dmg from %s (newHP=%d, newArmor=%d)",
				e.Player.Name, e.HealthDamage, e.Attacker.Name, e.Player.Health(), e.Player.Armor()),
			teamString(e.Attacker),
		))
	})

	// --- BulletDamage ---
	ctx.parser.RegisterEventHandler(func(e events.BulletDamage) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"BulletDamage",
			fmt.Sprintf("Attacker=%s, Victim=%s, Dist=%.1f, Penetrations=%d, NoScope=%v, AttackerInAir=%v",
				nameOrNil(e.Attacker), nameOrNil(e.Victim), e.Distance, e.NumPenetrations, e.IsNoScope, e.IsAttackerInAir),
			teamString(e.Attacker),
		))
	})

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
		if isGrenade(e.Weapon) {
			// No registramos log para granadas en WeaponFire
			return
		}

		sid := e.Shooter.SteamID64
		ps := getOrCreatePlayerStats(ctx, sid, e.Shooter.Name)
		ps.ShotsFired++

		speed := ctx.LastSpeed[sid]
		if speed > 200.0 {
			ps.AccidentalShots++
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"WeaponFire",
			fmt.Sprintf("Player %s fired %s (speed=%.2f)", e.Shooter.Name, e.Weapon.String(), speed),
			teamString(e.Shooter),
		))
	})

	// --- WeaponReload ---
	ctx.parser.RegisterEventHandler(func(e events.WeaponReload) {
		if ctx.parser.GameState().IsWarmupPeriod() {
			return
		}

		if e.Player == nil {
			return
		}

		currentTick := ctx.parser.GameState().IngameTick()
		lastTick, exists := ctx.LastReloadTick[e.Player.SteamID64]
		// Si ya se registró un reload recientemente, ignoramos este evento.
		if exists && (currentTick-lastTick < 100) {
			return
		}
		// Actualizamos el último tick registrado para este jugador.
		ctx.LastReloadTick[e.Player.SteamID64] = currentTick

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"WeaponReload",
			fmt.Sprintf("Player %s recargó su arma.", e.Player.Name),
			teamString(e.Player),
		))
	})

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
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(
				ctx.RoundNumber,
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

			ctx.EventLogs = append(ctx.EventLogs, newEventLog(
				ctx.RoundNumber,
				"ItemBought",
				fmt.Sprintf("Player %s bought %s [cost=%d]", e.Player.Name, e.Weapon.String(), cost),
				teamString(e.Player),
			))
		} else {
			// Caso: pick-up gratis (arma en el suelo, etc.)
			ctx.LastKnownMoney[sid] = currentMoney
			ctx.EventLogs = append(ctx.EventLogs, newEventLog(
				ctx.RoundNumber,
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
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"ItemRefund",
			fmt.Sprintf("Player %s refunded %s (+%d)", e.Player.Name, e.Weapon.String(), cost),
			teamString(e.Player),
		))
	})
}

// getWeaponPrice simplifica la obtención de precio para un EquipmentType.
func getWeaponPrice(eqType common.EquipmentType) int {
	if p, ok := weaponPrices[eqType]; ok {
		return p
	}
	return 0
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

func isGrenade(w *common.Equipment) bool {
	if w == nil {
		return false
	}
	switch w.Type {
	case common.EqFlash, common.EqSmoke, common.EqHE, common.EqDecoy, common.EqMolotov, common.EqIncendiary:
		return true
	}
	return false
}
