package demo

import (
	"cs2-demo-service/models"
	"fmt"
)

// ProcessDemoFileBasic llama al parser core y transforma el resultado en BasicDemoParseResult.
func ProcessDemoFileBasic(filePath, userSteamID, jsonData string) (*models.BasicDemoParseResult, error) {
	// Llamada al parser core que retorna el resultado avanzado.
	advancedResult, err := ProcessDemoFile(filePath, userSteamID, jsonData)
	if err != nil {
		return nil, err
	}

	// Transformación de las estadísticas de jugadores avanzadas a básicas.
	var basicPlayers []models.BasicPlayerStats
	for _, p := range advancedResult.Players {
		basicPlayers = append(basicPlayers, models.BasicPlayerStats{
			SteamID:        p.SteamID,
			Name:           p.Name,
			Team:           p.Team,
			Avatar:         p.Avatar,
			Kills:          p.Kills,
			Assists:        p.Assists,
			Deaths:         p.Deaths,
			KDRatio:        p.KDRatio,
			HSPercentage:   p.HSPercentage,
			ADR:            p.ADR,
			EnemiesFlashed: p.EnemiesFlashed,
			Position:       p.Position,
			ShotsFired:     p.ShotsFired,
			DoubleKills:    p.DoubleKills,
			TripleKills:    p.TripleKills,
			QuadKills:      p.QuadKills,
			Ace:            p.Ace,
			ClutchWins:     p.ClutchWins,
			EntryKills:     p.EntryKills,
			UtilityDamage:  p.UtilityDamage,
			FlashAssists:   p.FlashAssists,
		})
	}

	basicResult := models.BasicDemoParseResult{
		MatchID:       advancedResult.MatchID,
		MapName:       advancedResult.MapName,
		Duration:      advancedResult.Duration,
		Result:        advancedResult.Result,
		TeamScore:     advancedResult.TeamScore,
		OpponentScore: advancedResult.OpponentScore,
		Players:       basicPlayers,
		Filename:      advancedResult.Filename,
		Date:          advancedResult.Date,
		EventLogs:     advancedResult.EventLogs, // Incluir event logs para estadísticas de armas
	}

	// DEBUG: Verificar cuántos eventos se están incluyendo
	fmt.Printf("DEBUG: BasicDemoParseResult para %s - EventLogs count: %d\n", advancedResult.MatchID, len(advancedResult.EventLogs))
	if len(advancedResult.EventLogs) > 0 {
		fmt.Printf("DEBUG: Primer evento: %+v\n", advancedResult.EventLogs[0])
	}

	return &basicResult, nil
}
