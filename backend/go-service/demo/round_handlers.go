package demo

import (
	"cs2-demo-service/models"
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// -----------------------------------------------------------------------------
// registerRoundHandlers – registra todos los eventos del ciclo de ronda.
// -----------------------------------------------------------------------------
func registerRoundHandlers(ctx *DemoContext) {

	// ──────────────────────────────────────────────────────────────────────────
	// ROUND START
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.RoundStart) {

		// ╭───────────────────────────────────────────────────────────────────╮
		// │ •• PERSISTIR GRANADAS DE LA RONDA ANTERIOR ANTES DEL RESET ••    │
		// ╰───────────────────────────────────────────────────────────────────╯
		for key, meta := range ctx.GrenadeMetas {
			traj := ctx.GrenadeTrajectories[key]

			ctx.AllGrenadeTrajectories = append(ctx.AllGrenadeTrajectories,
				models.GrenadeMetadata{
					ProjectileID:      meta.ProjectileID,
					Round:             meta.Round,
					ThrowerSteamID:    meta.ThrowerSteamID,
					ThrowerName:       meta.ThrowerName,
					ThrowerTeam:       meta.ThrowerTeam,
					NadeType:          meta.NadeType,
					Exploded:          meta.Exploded,
					ExplosionPosition: meta.ExplosionPosition,
					Trajectory:        traj,
				})
		}

		//---------------------------------------------------------------------
		// 🚩 RESET de estructuras que usan entity-ID (granadas)
		//---------------------------------------------------------------------
		ctx.GrenadeMetas = make(map[int]*models.GrenadeMetadata)
		ctx.GrenadeTrajectories = make(map[int][]models.ProjectileTrajectoryEntry)
		ctx.lastGrenadeRecordTick = make(map[int]int)
		ctx.PendingMolotovs = make(map[int]models.MolotovData)

		//---------------------------------------------------------------------
		// Estado de la ronda
		//---------------------------------------------------------------------
		ctx.RoundNumber++
		ctx.RoundPlayerKills = make(map[uint64]int)
		ctx.ClutchCandidateT = nil
		ctx.ClutchCandidateCT = nil
		ctx.InRound = false
		ctx.RoundFirstKill = true
		ctx.LastTeamKillTick = make(map[int]int)

		//---------------------------------------------------------------------
		// Economía inicial
		//---------------------------------------------------------------------
		ctx.RoundEconomyMap[ctx.RoundNumber] = make(map[uint64]*models.RoundEconomyStats)

		for _, pl := range ctx.parser.GameState().Participants().All() {
			if pl.SteamID64 == 0 || pl.Team == common.TeamUnassigned {
				continue
			}

			bonus := ctx.LossBonusT
			if pl.Team == common.TeamCounterTerrorists {
				bonus = ctx.LossBonusCT
			}

			econ := &models.RoundEconomyStats{
				RoundNumber:  ctx.RoundNumber,
				Name:         pl.Name,
				Team:         teamString(pl),
				InitialMoney: pl.Money(),
				FinalMoney:   pl.Money(),
				LossBonus:    bonus,
			}
			econ.StartRoundItems = getPlayerInventorySnapshot(pl)

			ctx.EconomyHistory = append(ctx.EconomyHistory, econ)
			ctx.RoundEconomyMap[ctx.RoundNumber][pl.SteamID64] = econ
			ctx.LastKnownMoney[pl.SteamID64] = pl.Money()
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"RoundStart",
			fmt.Sprintf("Round %d started (freeze time)", ctx.RoundNumber),
			"",
		))
	})

	// ──────────────────────────────────────────────────────────────────────────
	// FREEZETIME END
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		ctx.InRound = true

		freezeTimeEndTick := ctx.parser.GameState().IngameTick()
		buyTimeSeconds := 20 // ejemplo
		buyWindowTicks := int(ctx.parser.TickRate() * float64(buyTimeSeconds))
		ctx.BuyWindowEndTickForRound[ctx.RoundNumber] = freezeTimeEndTick + buyWindowTicks

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"FreezeTimeEnded",
			fmt.Sprintf("Round %d: freeze ended (buy window ~%ds)", ctx.RoundNumber, buyTimeSeconds),
			"",
		))
	})

	// ──────────────────────────────────────────────────────────────────────────
	// ROUND END
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.RoundEnd) {
		ctx.InRound = false
		ctx.InRoundEndPeriod = true
		ctx.LastRoundWinner = e.Winner
		ctx.LastRoundReason = e.Reason

		bombPlantedWin := (e.Reason == events.RoundEndReasonTargetBombed)

		// Actualizamos loss-bonus
		if e.Winner == common.TeamTerrorists {
			ctx.LossBonusCT = UpdateLossBonus(ctx.LossBonusCT, false, bombPlantedWin)
			ctx.LossBonusT = UpdateLossBonus(ctx.LossBonusT, true, bombPlantedWin)
		} else if e.Winner == common.TeamCounterTerrorists {
			ctx.LossBonusT = UpdateLossBonus(ctx.LossBonusT, false, false)
			ctx.LossBonusCT = UpdateLossBonus(ctx.LossBonusCT, true, false)
		}

		// Flash / multi-kills / clutch
		processFlashEvents(ctx)
		processMultiKills(ctx)

		if e.Winner == common.TeamTerrorists && ctx.ClutchCandidateT != nil {
			ctx.PlayerStatsMap[ctx.ClutchCandidateT.SteamID].ClutchWins++
		} else if e.Winner == common.TeamCounterTerrorists && ctx.ClutchCandidateCT != nil {
			ctx.PlayerStatsMap[ctx.ClutchCandidateCT.SteamID].ClutchWins++
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"RoundEnd",
			fmt.Sprintf("Round %d ended. Reason=%v (Winner=%v)", ctx.RoundNumber, e.Reason, e.Winner),
			"",
		))
	})

	// ──────────────────────────────────────────────────────────────────────────
	// ROUND END OFFICIAL
	// ──────────────────────────────────────────────────────────────────────────
	ctx.parser.RegisterEventHandler(func(e events.RoundEndOfficial) {
		var survivors []string
		for _, pl := range ctx.parser.GameState().Participants().Playing() {
			if pl.IsAlive() {
				survivors = append(survivors, pl.Name)
			}
		}

		winType := "kill"
		switch ctx.LastRoundReason {
		case events.RoundEndReasonTargetBombed:
			winType = "bomb"
		case events.RoundEndReasonBombDefused:
			winType = "defuse"
		}

		winAmount := GetWinAmount(ctx.LastRoundWinner, winType)

		// Calculamos dinero final jugador a jugador
		for _, pl := range ctx.parser.GameState().Participants().All() {
			econ, ok := ctx.RoundEconomyMap[ctx.RoundNumber][pl.SteamID64]
			if !ok {
				continue
			}

			finalCalc := econ.InitialMoney
			finalCalc -= econ.SpentInBuy
			finalCalc += econ.KillReward
			finalCalc += econ.RewardforPlant

			if pl.Team == ctx.LastRoundWinner {
				finalCalc += winAmount
			} else {
				finalCalc += econ.LossBonus
			}
			econ.FinalMoney = finalCalc
		}

		// Inventario final
		for _, pl := range ctx.parser.GameState().Participants().All() {
			econ, ok := ctx.RoundEconomyMap[ctx.RoundNumber][pl.SteamID64]
			if !ok {
				continue
			}
			econ.EndRoundItems = buildEndRoundItems(pl)
		}

		ctx.BombPlanted = false
		ctx.InRoundEndPeriod = false

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"RoundEndOfficial",
			fmt.Sprintf("Survivors equip: %v", survivors),
			"",
		))
	})
}

// -----------------------------------------------------------------------------
// HELPER FUNCTIONS
// -----------------------------------------------------------------------------

// getPlayerInventorySnapshot devuelve el inventario inicial (sin cuchillo).
func getPlayerInventorySnapshot(pl *common.Player) []models.RoundItem {
	var items []models.RoundItem
	if pl == nil {
		return items
	}
	for _, w := range pl.Inventory {
		if w == nil || w.Type == common.EqKnife {
			continue
		}
		items = append(items, models.RoundItem{
			Name:  w.Type.String(),
			Price: getWeaponPrice(w.Type),
		})
	}
	return items
}

// buildEndRoundItems construye la lista de equipamiento al final de la ronda.
func buildEndRoundItems(pl *common.Player) []models.RoundItem {
	var items []models.RoundItem
	if pl == nil {
		return items
	}

	// Armor / casco
	if pl.Armor() > 0 {
		n := "Kevlar"
		if pl.HasHelmet() {
			n = "Kevlar + Helmet"
		}
		items = append(items, models.RoundItem{
			Name:  n,
			Armor: pl.Armor(),
		})
	}

	// Kit de desactivación
	if pl.HasDefuseKit() {
		items = append(items, models.RoundItem{Name: "Defuse Kit"})
	}

	// Armas y utilidades (sin cuchillo ni repetición de armadura)
	for _, w := range pl.Inventory {
		if w == nil {
			continue
		}
		if w.Type == common.EqKnife ||
			w.Type == common.EqKevlar ||
			w.Type == common.EqHelmet {
			continue
		}
		items = append(items, models.RoundItem{
			Name:  w.Type.String(),
			Price: getWeaponPrice(w.Type),
		})
	}
	return items
}

// processFlashEvents compila los enemigos/aliados cegados por cada flash.
func processFlashEvents(ctx *DemoContext) {
	for sid, list := range ctx.FlashEvents {
		ps := getOrCreatePlayerStats(ctx, sid, "")
		for _, fd := range list {
			ps.EnemiesFlashed += fd.enemyCount
			if fd.friendlyCount > fd.enemyCount && fd.friendlyCount > 0 {
				ps.BadFlashCount++
				ctx.EventLogs = append(ctx.EventLogs, newEventLog(
					ctx.RoundNumber,
					"BadFlash",
					fmt.Sprintf("%s's flash blinded %d teammates vs %d enemies",
						ps.Name, fd.friendlyCount, fd.enemyCount),
					"",
				))
			}
		}
	}
	ctx.FlashEvents = make(map[uint64][]flashData)
}

// processMultiKills asigna double-kills, triple-kills, ace, etc.
func processMultiKills(ctx *DemoContext) {
	for sid, kills := range ctx.RoundPlayerKills {
		ps := getOrCreatePlayerStats(ctx, sid, "")
		switch kills {
		case 2:
			ps.DoubleKills++
		case 3:
			ps.TripleKills++
		case 4:
			ps.QuadKills++
		case 5:
			ps.Ace++
		}
	}
}
