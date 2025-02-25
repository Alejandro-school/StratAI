package models

// BasicPlayerStats contiene únicamente las estadísticas que se mostrarán en el frontend.
type BasicPlayerStats struct {
	SteamID        string  `json:"steamID"`
	Name           string  `json:"name"`
	Team           string  `json:"team"` // <-- Agregado
	Avatar         string  `json:"avatar"`
	Kills          int     `json:"kills"`
	Assists        int     `json:"assists"`
	Deaths         int     `json:"deaths"`
	KDRatio        float64 `json:"kd_ratio"`
	HSPercentage   float64 `json:"hs_percentage"`
	ADR            float64 `json:"adr"`
	EnemiesFlashed int     `json:"EnemiesFlashed"`
	Position       int     `json:"position"`
	ShotsFired     int     `json:"shots_fired"`
	AimPlacement   float64 `json:"aim_placement"`
	DoubleKills    int     `json:"double_kills"`
	TripleKills    int     `json:"triple_kills"`
	QuadKills      int     `json:"quad_kills"`
	Ace            int     `json:"ace"`
	ClutchWins     int     `json:"clutch_wins"`
}

// BasicDemoParseResult solo contiene la información necesaria para el frontend.
type BasicDemoParseResult struct {
	MatchID       string             `json:"match_id"`
	MapName       string             `json:"map_name"`
	Duration      string             `json:"match_duration"`
	Result        string             `json:"result"`
	TeamScore     int                `json:"team_score"`
	OpponentScore int                `json:"opponent_score"`
	Players       []BasicPlayerStats `json:"players"`
	Filename      string             `json:"filename"`
	Date          string             `json:"date"`
}
