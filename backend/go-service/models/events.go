package models

// KillEvent representa un kill ocurrido en la partida
type KillEvent struct {
	Tick              int     `json:"tick"`
	Round             int     `json:"round"`
	Killer            string  `json:"killer"`
	KillerSteamID     uint64  `json:"killer_steam_id"`
	Victim            string  `json:"victim"`
	VictimSteamID     uint64  `json:"victim_steam_id"`
	Assister          string  `json:"assister,omitempty"`
	AssisterSteamID   uint64  `json:"assister_steam_id,omitempty"`
	Weapon            string  `json:"weapon"`
	IsHeadshot        bool    `json:"is_headshot"`
	IsWallbang        bool    `json:"is_wallbang"`
	PenetratedObjects int     `json:"penetrated_objects"` // Número de objetos/paredes penetrados
	AttackerBlind     bool    `json:"attacker_blind"`
	NoScope           bool    `json:"no_scope"`
	ThroughSmoke      bool    `json:"through_smoke"`
	Distance          float32 `json:"distance"`
	KillerX           float64 `json:"killer_x"`
	KillerY           float64 `json:"killer_y"`
	KillerZ           float64 `json:"killer_z"`
	VictimX           float64 `json:"victim_x"`
	VictimY           float64 `json:"victim_y"`
	VictimZ           float64 `json:"victim_z"`

	// Mechanics on Kill
	CounterStrafeRating float64 `json:"counter_strafe_rating,omitempty"` // 0-100
	TimeToDamage        float64 `json:"time_to_damage,omitempty"`        // ms
}

// DamageEvent representa daño infligido a un jugador
type DamageEvent struct {
	Tick         int    `json:"tick"`
	Round        int    `json:"round"`
	Attacker     string `json:"attacker"`
	Victim       string `json:"victim"`
	Weapon       string `json:"weapon"`
	HealthDamage int    `json:"health_damage"`
	ArmorDamage  int    `json:"armor_damage"`
	HitGroup     string `json:"hit_group"`
	VictimHealth int    `json:"victim_health"`
}

// FlashEvent representa una flash detonada
type FlashEvent struct {
	Tick    int           `json:"tick"`
	Round   int           `json:"round"`
	Thrower string        `json:"thrower"`
	X       float64       `json:"x"`
	Y       float64       `json:"y"`
	Z       float64       `json:"z"`
	Victims []FlashVictim `json:"victims"`
}

// FlashVictim representa un jugador flasheado
type FlashVictim struct {
	Name     string  `json:"name"`
	Duration float32 `json:"duration"`
}

// HEEvent representa una HE detonada
type HEEvent struct {
	Tick    int     `json:"tick"`
	Round   int     `json:"round"`
	Thrower string  `json:"thrower"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
}

// SmokeEvent representa un smoke lanzado
type SmokeEvent struct {
	Tick    int     `json:"tick"`
	Round   int     `json:"round"`
	Thrower string  `json:"thrower"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
}

// MolotovEvent representa un molotov/incendiary lanzado
type MolotovEvent struct {
	Tick    int     `json:"tick"`
	Round   int     `json:"round"`
	Thrower string  `json:"thrower"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
}

// BombEvent representa eventos relacionados con la bomba
type BombEvent struct {
	EventType string  `json:"event_type"` // "plant", "defuse", "explode"
	Tick      int     `json:"tick"`
	Round     int     `json:"round"`
	Player    string  `json:"player,omitempty"`
	Site      string  `json:"site,omitempty"`
	X         float64 `json:"x,omitempty"`
	Y         float64 `json:"y,omitempty"`
	Z         float64 `json:"z,omitempty"`
}

// MovementLog guarda la posición, velocidad y estado del jugador en un tick
type MovementLog struct {
	Round     int     `json:"round"`
	Tick      int     `json:"tick"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Z         float64 `json:"z"`
	Speed     float64 `json:"speed"`
	IsDucking bool    `json:"is_ducking"`
	Pitch     float32 `json:"pitch"`
	Yaw       float32 `json:"yaw"`
}

// RoundEconomyStats representa la economía de un equipo en una ronda
type RoundEconomyStats struct {
	Round          int    `json:"round"`
	Team           string `json:"team"`
	StartMoney     int    `json:"start_money"`
	EquipmentValue int    `json:"equipment_value"`
	MoneySpent     int    `json:"money_spent"`
	FullBuys       int    `json:"full_buys"`
	PartialBuys    int    `json:"partial_buys"`
	Saves          int    `json:"saves"`
}

// PlayerStats representa las estadísticas finales de un jugador
type PlayerStats struct {
	SteamID uint64  `json:"steam_id"`
	Name    string  `json:"name"`
	Team    string  `json:"team"`
	Kills   int     `json:"kills"`
	Deaths  int     `json:"deaths"`
	Assists int     `json:"assists"`
	Damage  int     `json:"damage"`
	MVPs    int     `json:"mvps"`
	ADR     float64 `json:"adr"`
	KDRatio float64 `json:"kd_ratio"`
}

// RoundData representa el resultado de una ronda
type RoundData struct {
	Round       int    `json:"round"`
	Winner      string `json:"winner"`
	Reason      string `json:"reason"`
	CTScore     int    `json:"ct_score"`
	TScore      int    `json:"t_score"`
	BombPlanted bool   `json:"bomb_planted"`
	BombSite    string `json:"bomb_site,omitempty"`
	BombTick    int    `json:"bomb_tick,omitempty"`
}

// PlayerData agrupa todo lo relativo a un jugador durante el parsing
type PlayerData struct {
	SteamID  uint64        `json:"steam_id"`
	Name     string        `json:"name"`
	Team     string        `json:"team"`
	Distance float64       `json:"distance"` // Distancia acumulada
	Movement []MovementLog `json:"movement"`

	// Stats del scoreboard (populated by BuildMatchData)
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
	Damage  int `json:"damage"`
	MVPs    int `json:"mvps"`

	// Advanced analytics
	Crosshair     *CrosshairStats     `json:"crosshair,omitempty"`
	Mechanics     *MechanicsStats     `json:"mechanics,omitempty"`
	ReactionTimes []ReactionTimeEvent `json:"reaction_times,omitempty"`
	Sprays        []SprayAnalysis     `json:"sprays,omitempty"`
}

// MechanicsStats agrupa métricas mecánicas avanzadas
type MechanicsStats struct {
	AvgCounterStrafeRating float64 `json:"avg_counter_strafe_rating"` // 0-100
	AvgRecoilControl       float64 `json:"avg_recoil_control"`        // 0-100
	AvgTimeToDamage        float64 `json:"avg_time_to_damage"`        // ms
}

// ReactionTimeEvent almacena un evento de reaction time
type ReactionTimeEvent struct {
	Round          int    `json:"round"`
	EnemyID        uint64 `json:"enemy_id"`
	FirstSeenTick  int    `json:"first_seen_tick"`
	FirstShotTick  int    `json:"first_shot_tick"`
	ReactionTimeMs int    `json:"reaction_time_ms"`

	// Metadata para analítica avanzada
	WasFlashed        bool    `json:"was_flashed"`        // Si tenía flash residual
	FlashDuration     float32 `json:"flash_duration"`     // Duración del flash en segundos
	SmokeInPath       bool    `json:"smoke_in_path"`      // Si había humo en la línea de visión
	Distance          float64 `json:"distance"`           // Distancia al enemigo en unidades
	PenetratedObjects int     `json:"penetrated_objects"` // Objetos penetrados en la kill (0 = visión clara)
}

// CrosshairStats estadísticas de crosshair placement
type CrosshairStats struct {
	TimeAtHeadLevel   int `json:"time_at_head_level"`
	TimeAtBodyLevel   int `json:"time_at_body_level"`
	TimeAtGroundLevel int `json:"time_at_ground_level"`
}

// SprayData tracking de spray en progreso (usado internamente, no exportado)
type SprayData struct {
	StartTick   int
	ShotCount   int
	Hits        int
	Headshots   int
	StartPitch  float32
	StartYaw    float32
	PitchSum    float32
	YawSum      float32
	WasMoving   bool
	WasCrouched bool
	Weapon      string
}

// SprayAnalysis almacena análisis de un spray
type SprayAnalysis struct {
	Round       int     `json:"round"`
	StartTick   int     `json:"start_tick"`
	EndTick     int     `json:"end_tick"`
	ShotCount   int     `json:"shot_count"`
	Hits        int     `json:"hits"`
	Headshots   int     `json:"headshots"`
	Weapon      string  `json:"weapon"`
	HitRate     float64 `json:"hit_rate"`
	Quality     string  `json:"quality"` // excellent, good, fair, poor
	WasMoving   bool    `json:"was_moving"`
	WasCrouched bool    `json:"was_crouched"`
}

// MatchData es el output final completo
type MatchData struct {
	// Metadata
	MatchID  string `json:"match_id"`
	MapName  string `json:"map_name"`
	CTScore  int    `json:"ct_score"`
	TScore   int    `json:"t_score"`
	Winner   string `json:"winner"`
	Duration int    `json:"duration"` // En ticks

	// Round data
	Rounds  []RoundData         `json:"rounds"`
	Economy []RoundEconomyStats `json:"economy"`

	// Player data (indexado por SteamID64)
	Players map[uint64]*PlayerData `json:"players"`

	// Combat events
	Kills  []KillEvent   `json:"kills"`
	Damage []DamageEvent `json:"damage"`

	// Grenade events
	Flashes    []FlashEvent   `json:"flashes"`
	HEGrenades []HEEvent      `json:"he_grenades"`
	Smokes     []SmokeEvent   `json:"smokes"`
	Molotovs   []MolotovEvent `json:"molotovs"`

	// Bomb events
	BombEvents []BombEvent `json:"bomb_events"`

	// Player stats (para scoreboard final)
	PlayerStats []PlayerStats `json:"player_stats"`
}
