package models

// AdvancedPlayerStats extiende las estadísticas básicas con información adicional.
type AdvancedPlayerStats struct {
	Basic              BasicPlayerStats `json:"basic_stats"`
	Team               string           `json:"team"`
	DistanceTraveled   float64          `json:"distance_traveled"`
	ShotsFiredNoReason int              `json:"shots_fired_no_reason"`
	Purchases          int              `json:"purchases"`
	InitialMoney       int              `json:"initial_money"`
	BadFlashCount      int              `json:"bad_flash_count"`
	BadMolotovCount    int              `json:"bad_molotov_count"`
	TeamKillCount      int              `json:"team_kill_count"`
	TeamDamage         int              `json:"team_damage"`
	AccidentalShots    int              `json:"accidental_shots"`
	CrouchTicks        int              `json:"crouch_ticks"`
	WalkTicks          int              `json:"walk_ticks"`
	RunTicks           int              `json:"run_ticks"`
	Movement           []MovementLog    `json:"movement"`
}

// AdvancedDemoParseResult extiende el resultado básico con detalles adicionales.
type AdvancedDemoParseResult struct {
	Basic          BasicDemoParseResult `json:"basic_result"`
	EconomyHistory []RoundEconomyStats  `json:"economy_history"`
	EventLogs      []EventLog           `json:"event_logs"`
}
