package models

// PersonalPerformance representa el resumen del desempeño personal del jugador a lo largo de varias partidas.
type PersonalPerformance struct {
	MatchesPlayed       int     `json:"matches_played"`
	TotalKills          int     `json:"total_kills"`
	TotalDeaths         int     `json:"total_deaths"`
	TotalAssists        int     `json:"total_assists"`
	KDRatio             float64 `json:"kd_ratio"`
	AverageADR          float64 `json:"average_adr"`
	AverageAimPlacement float64 `json:"average_aim_placement"`
	TotalDoubleKills    int     `json:"total_double_kills"`
	TotalTripleKills    int     `json:"total_triple_kills"`
	TotalQuadKills      int     `json:"total_quad_kills"`
	TotalAces           int     `json:"total_aces"`
	TotalClutchWins     int     `json:"total_clutch_wins"`
}
