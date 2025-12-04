package handlers

import (
	"cs2-demo-service/models"
	"fmt"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterRoundHandlers registra handlers para eventos de ronda
func RegisterRoundHandlers(ctx *models.DemoContext) {
	// Round start
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		// ctx.CurrentRound is managed by timeline handlers now (based on GameState)
		// ctx.CurrentRound++ 
		ctx.InRound = true
		ctx.BombPlanted = false
		ctx.BombSite = ""
		ctx.BombTick = 0

		roundData := models.RoundData{
			Round:   ctx.CurrentRound,
			Winner:  "",
			Reason:  "",
			CTScore: 0,
			TScore:  0,
		}

		ctx.MatchData.Rounds = append(ctx.MatchData.Rounds, roundData)
	})

	// Freeze time end
	ctx.Parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		// Marcar inicio de ronda activa
		ctx.InRound = true
	})

	// Round end
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) {
		ctx.InRound = false

		if len(ctx.MatchData.Rounds) > 0 {
			lastRound := &ctx.MatchData.Rounds[len(ctx.MatchData.Rounds)-1]

			// Winner usando enum nativo
			switch e.Winner {
			case common.TeamTerrorists:
				lastRound.Winner = "T"
			case common.TeamCounterTerrorists:
				lastRound.Winner = "CT"
			}

			// Reason usando enum nativo
			lastRound.Reason = fmt.Sprintf("%v", e.Reason)

			// Scores actuales
			gs := ctx.Parser.GameState()
			lastRound.CTScore = gs.TeamCounterTerrorists().Score()
			lastRound.TScore = gs.TeamTerrorists().Score()

			// Bomba plantada en esta ronda
			if ctx.BombPlanted {
				lastRound.BombPlanted = true
				lastRound.BombSite = ctx.BombSite
				lastRound.BombTick = ctx.BombTick
			}
		}
	})
}
