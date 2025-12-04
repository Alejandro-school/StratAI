package handlers

import (
	"cs2-demo-service/models"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterEconomyHandlers registra handlers de economía
func RegisterEconomyHandlers(ctx *models.DemoContext) {
	// Round start - capturar economía inicial
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		gs := ctx.Parser.GameState()

		// Economía CT
		ctEconomy := models.RoundEconomyStats{
			Round:          ctx.CurrentRound,
			Team:           "CT",
			StartMoney:     0,
			EquipmentValue: 0,
			MoneySpent:     0,
			FullBuys:       0,
			PartialBuys:    0,
			Saves:          0,
		}

		ctPlayers := 0
		for _, player := range gs.TeamCounterTerrorists().Members() {
			if player.IsConnected {
				ctPlayers++
				ctEconomy.StartMoney += player.Money()
				ctEconomy.EquipmentValue += player.EquipmentValueCurrent()

				// Clasificar tipo de compra
				equipValue := player.EquipmentValueCurrent()
				if equipValue >= 4000 {
					ctEconomy.FullBuys++
				} else if equipValue >= 2000 {
					ctEconomy.PartialBuys++
				} else {
					ctEconomy.Saves++
				}
			}
		}

		if ctPlayers > 0 {
			ctEconomy.StartMoney /= ctPlayers
			ctEconomy.EquipmentValue /= ctPlayers
		}

		ctx.MatchData.Economy = append(ctx.MatchData.Economy, ctEconomy)

		// Economía T
		tEconomy := models.RoundEconomyStats{
			Round:          ctx.CurrentRound,
			Team:           "T",
			StartMoney:     0,
			EquipmentValue: 0,
			MoneySpent:     0,
			FullBuys:       0,
			PartialBuys:    0,
			Saves:          0,
		}

		tPlayers := 0
		for _, player := range gs.TeamTerrorists().Members() {
			if player.IsConnected {
				tPlayers++
				tEconomy.StartMoney += player.Money()
				tEconomy.EquipmentValue += player.EquipmentValueCurrent()

				equipValue := player.EquipmentValueCurrent()
				if equipValue >= 4000 {
					tEconomy.FullBuys++
				} else if equipValue >= 2000 {
					tEconomy.PartialBuys++
				} else {
					tEconomy.Saves++
				}
			}
		}

		if tPlayers > 0 {
			tEconomy.StartMoney /= tPlayers
			tEconomy.EquipmentValue /= tPlayers
		}

		ctx.MatchData.Economy = append(ctx.MatchData.Economy, tEconomy)
	})

	// Round freeze time end - calcular dinero gastado
	ctx.Parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		gs := ctx.Parser.GameState()

		// Actualizar MoneySpent para economía CT
		for i := range ctx.MatchData.Economy {
			econ := &ctx.MatchData.Economy[i]
			if econ.Round == ctx.CurrentRound && econ.Team == "CT" {
				totalSpent := 0
				ctPlayers := 0
				for _, player := range gs.TeamCounterTerrorists().Members() {
					if player.IsConnected {
						totalSpent += player.MoneySpentThisRound()
						ctPlayers++
					}
				}
				if ctPlayers > 0 {
					econ.MoneySpent = totalSpent / ctPlayers
				}
			}
		}

		// Actualizar MoneySpent para economía T
		for i := range ctx.MatchData.Economy {
			econ := &ctx.MatchData.Economy[i]
			if econ.Round == ctx.CurrentRound && econ.Team == "T" {
				totalSpent := 0
				tPlayers := 0
				for _, player := range gs.TeamTerrorists().Members() {
					if player.IsConnected {
						totalSpent += player.MoneySpentThisRound()
						tPlayers++
					}
				}
				if tPlayers > 0 {
					econ.MoneySpent = totalSpent / tPlayers
				}
			}
		}
	})
}
