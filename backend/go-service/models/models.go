package models

// PlayerStats representa las estadísticas de UN jugador.
type PlayerStats struct {
	SteamID      string  `json:"steamID"`
	Name         string  `json:"name"`
	Team         string  `json:"team"` // "T", "CT" o "UNK"
	Kills        int     `json:"kills"`
	Assists      int     `json:"assists"`
	Deaths       int     `json:"deaths"`
	ADR          float64 `json:"adr"`
	HSPercentage float64 `json:"hs_percentage"`
	KDRatio      float64 `json:"kd_ratio"`
	FlashAssists int     `json:"flash_assists"`
	RankScore    int     `json:"rank_score"` // 🎯 Puntuación de Premiere
	Avatar       string  `json:"avatar"`     // 🖼️ URL del avatar
	Position     int     `json:"position"`   // 🏆 Posición basada en rendimiento

}

// DemoParseResult almacena la info de toda la partida.
type DemoParseResult struct {
	MatchID       string        `json:"match_id"` // un ID único para identificar la partida
	MapName       string        `json:"map_name"`
	Duration      string        `json:"duration"` // Ej: "28:44"
	Result        string        `json:"result"`   // "victory" o "defeat"
	TeamScore     int           `json:"team_score"`
	OpponentScore int           `json:"opponent_score"`
	Players       []PlayerStats `json:"players"`
	Filename      string        `json:"filename"`
	Date          string        `json:"date"` // Extraída de la fecha del archivo
}
