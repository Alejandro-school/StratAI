package demo

import (
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

func registerBombHandlers(ctx *DemoContext) {
	ctx.parser.RegisterEventHandler(func(e events.BombPlanted) {
		ctx.BombPlanted = true
		planter := e.Player
		tAlive, ctAlive := getAlivePlayersCount(ctx)

		// === NUEVO: Guardar información del plant para rotation tracking ===
		ctx.BombPlantedTick = ctx.parser.GameState().IngameTick()
		// Determinar sitio de la bomba (A o B) desde el sitio del plant
		// Bombsite es un tipo especial (rune), convertirlo a string
		siteChar := rune(e.Site)
		if siteChar == 'A' {
			ctx.BombPlantedSite = "A"
		} else if siteChar == 'B' {
			ctx.BombPlantedSite = "B"
		} else {
			ctx.BombPlantedSite = "Unknown"
		}

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx,
			"BombPlanted",
			fmt.Sprintf("Bomb planted by %s at %c (T_alive=%d,CT_alive=%d)", nameOrNil(planter), e.Site, tAlive, ctAlive),
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
		tAlive, ctAlive := getAlivePlayersCount(ctx)

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx,
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
		tAlive, ctAlive := getAlivePlayersCount(ctx)

		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx,
			"BombExploded",
			fmt.Sprintf("Bomb exploded (T_alive=%d,CT_alive=%d)", tAlive, ctAlive),
			"",
		))
	})

	ctx.parser.RegisterEventHandler(func(e events.BombPlantBegin) {
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx,
			"BombBeginPlant",
			fmt.Sprintf("Player %s started planting the bomb.", nameOrNil(e.Player)),
			teamString(e.Player),
		))
	})

	ctx.parser.RegisterEventHandler(func(e events.BombDefuseStart) {
		ctx.EventLogs = append(ctx.EventLogs, newEventLog(
			ctx,
			"BombBeginDefuse",
			fmt.Sprintf("Player %s started defusing the bomb.", nameOrNil(e.Player)),
			teamString(e.Player),
		))
	})
}
