package demo

import (
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

func registerBombHandlers(ctx *DemoContext) {
	ctx.parser.RegisterEventHandler(func(e events.BombPlanted) {
		ctx.BombPlanted = true
		planter := e.Player
		var tAlive, ctAlive int
		for _, p := range ctx.parser.GameState().Participants().Playing() {
			if p.Team == common.TeamTerrorists && p.IsAlive() {
				tAlive++
			} else if p.Team == common.TeamCounterTerrorists && p.IsAlive() {
				ctAlive++
			}
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"BombPlanted",
			fmt.Sprintf("Bomb planted by %s at (T_alive=%d,CT_alive=%d)", nameOrNil(planter), tAlive, ctAlive),
			teamString(planter),
		))

		// Sumamos los 300 a MoneyDuringRound (no final)
		if planter != nil {
			ps := getOrCreatePlayerStats(ctx, planter.SteamID64, planter.Name)
			ps.BombPlants++

			if econ, ok := ctx.RoundEconomyMap[ctx.RoundNumber][planter.SteamID64]; ok {
				econ.RewardforPlant += 300
			}
		}
	})

	ctx.parser.RegisterEventHandler(func(e events.BombDefused) {
		defuser := e.Player
		var tAlive, ctAlive int
		for _, p := range ctx.parser.GameState().Participants().Playing() {
			if p.Team == common.TeamTerrorists && p.IsAlive() {
				tAlive++
			} else if p.Team == common.TeamCounterTerrorists && p.IsAlive() {
				ctAlive++
			}
		}
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"BombDefused",
			fmt.Sprintf("Bomb defused by %s at (T_alive=%d,CT_alive=%d)", nameOrNil(defuser), tAlive, ctAlive),
			teamString(defuser),
		))
		if defuser != nil {
			ps := getOrCreatePlayerStats(ctx, defuser.SteamID64, defuser.Name)
			ps.BombDefuses++
		}
	})

	ctx.parser.RegisterEventHandler(func(e events.BombExplode) {
		var tAlive, ctAlive int
		for _, p := range ctx.parser.GameState().Participants().Playing() {
			if p.Team == common.TeamTerrorists && p.IsAlive() {
				tAlive++
			} else if p.Team == common.TeamCounterTerrorists && p.IsAlive() {
				ctAlive++
			}
		}
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"BombExploded",
			fmt.Sprintf("Bomb exploded (T_alive=%d,CT_alive=%d)", tAlive, ctAlive),
			"",
		))
	})

	ctx.parser.RegisterEventHandler(func(e events.BombPlantBegin) {
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"BombBeginPlant",
			fmt.Sprintf("Player %s comenzó a plantar la bomba.", nameOrNil(e.Player)),
			teamString(e.Player),
		))
	})

	ctx.parser.RegisterEventHandler(func(e events.BombDefuseStart) {
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx.RoundNumber,
			"BombBeginDefuse",
			fmt.Sprintf("Player %s comenzó a desactivar la bomba.", nameOrNil(e.Player)),
			teamString(e.Player),
		))
	})
}
