package parser

import (
	"cs2-demo-service/models"
)

// BuildMatchData construye la estructura final de datos desde el contexto
func BuildMatchData(ctx *models.DemoContext) *models.MatchData {
	matchData := ctx.MatchData

	// Agregar metadata del header
	header := ctx.Parser.Header()
	matchData.MapName = header.MapName

	// Agregar scores finales
	gameState := ctx.Parser.GameState()
	matchData.TScore = gameState.TeamTerrorists().Score()
	matchData.CTScore = gameState.TeamCounterTerrorists().Score()

	// Determinar ganador
	if matchData.TScore > matchData.CTScore {
		matchData.Winner = "T"
	} else if matchData.CTScore > matchData.TScore {
		matchData.Winner = "CT"
	} else {
		matchData.Winner = "Draw"
	}

	// Agregar stats finales de scoreboard a cada jugador
	for _, player := range gameState.Participants().All() {
		if player.SteamID64 == 0 {
			continue
		}

		playerData, exists := matchData.Players[player.SteamID64]
		if !exists {
			playerData = &models.PlayerData{
				SteamID: player.SteamID64,
				Name:    player.Name,
			}
			matchData.Players[player.SteamID64] = playerData
		}

		// Stats del scoreboard nativo
		playerData.Kills = player.Kills()
		playerData.Deaths = player.Deaths()
		playerData.Assists = player.Assists()
		playerData.Damage = player.TotalDamage()
		playerData.MVPs = player.MVPs()

		// Team
		switch player.Team {
		case 2:
			playerData.Team = "T"
		case 3:
			playerData.Team = "CT"
		default:
			playerData.Team = "Spectator"
		}

		// Crosshair stats si existen
		if ch, ok := ctx.CrosshairStats[player.SteamID64]; ok {
			playerData.Crosshair = &models.CrosshairStats{
				TimeAtHeadLevel:   ch.TimeAtHeadLevel,
				TimeAtBodyLevel:   ch.TimeAtBodyLevel,
				TimeAtGroundLevel: ch.TimeAtGroundLevel,
			}
		}

		// Sprays y ReactionTimes ya están guardados directamente en playerData por los analyzers
	}

	return matchData
}
