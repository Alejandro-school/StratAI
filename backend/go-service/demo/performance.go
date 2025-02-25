package demo

import (
	"cs2-demo-service/models"
)

// AggregatePersonalPerformance itera sobre las partidas procesadas y acumula las estadísticas del jugador identificado por playerSteamID.
func AggregatePersonalPerformance(demos []models.DemoParseResult, playerSteamID string) models.PersonalPerformance {
	var totalKills, totalDeaths, totalAssists int
	var totalDoubleKills, totalTripleKills, totalQuadKills, totalAces, totalClutchWins int
	var sumADR, sumAimPlacement float64
	var matchesPlayed int

	for _, demo := range demos {
		for _, player := range demo.Players {
			if player.SteamID == playerSteamID {
				matchesPlayed++
				totalKills += player.Kills
				totalDeaths += player.Deaths
				totalAssists += player.Assists
				totalDoubleKills += player.DoubleKills
				totalTripleKills += player.TripleKills
				totalQuadKills += player.QuadKills
				totalAces += player.Ace
				totalClutchWins += player.ClutchWins
				sumADR += player.ADR
				sumAimPlacement += player.AimPlacement
			}
		}
	}

	kdRatio := safeDivide(totalKills, totalDeaths)
	avgADR := safeDivideFloat(sumADR, float64(matchesPlayed))
	avgAimPlacement := safeDivideFloat(sumAimPlacement, float64(matchesPlayed))

	performance := models.PersonalPerformance{
		MatchesPlayed:       matchesPlayed,
		TotalKills:          totalKills,
		TotalDeaths:         totalDeaths,
		TotalAssists:        totalAssists,
		KDRatio:             kdRatio,
		AverageADR:          avgADR,
		AverageAimPlacement: avgAimPlacement,
		TotalDoubleKills:    totalDoubleKills,
		TotalTripleKills:    totalTripleKills,
		TotalQuadKills:      totalQuadKills,
		TotalAces:           totalAces,
		TotalClutchWins:     totalClutchWins,
	}

	return performance
}

func safeDivide(a, b int) float64 {
	if b == 0 {
		return float64(a)
	}
	return float64(a) / float64(b)
}

func safeDivideFloat(a, b float64) float64 {
	if b == 0.0 {
		return a
	}
	return a / b
}
