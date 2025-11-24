package models

type RoundItem struct {
	Name      string `json:"name"`
	Price     int    `json:"price"`
	Armor     int    `json:"armor,omitempty"`
	HasDefuse bool   `json:"has_defuse,omitempty"`
}

// RoundContext proporciona contexto de la ronda para IA
type RoundContext struct {
	RoundNumber         int  `json:"round_number"`
	IsFirstHalf         bool `json:"is_first_half"`         // Rondas 1-12
	IsSecondHalf        bool `json:"is_second_half"`        // Rondas 13-24
	IsOvertime          bool `json:"is_overtime"`           // Rondas 25+
	OvertimeNumber      int  `json:"overtime_number"`       // Qué OT es (1, 2, 3...)
	RoundsUntilHalftime int  `json:"rounds_until_halftime"` // Cuántas faltan para cambio
	IsLastBeforeSwitch  bool `json:"is_last_before_switch"` // Última antes de cambio
	WasEconomyReset     bool `json:"was_economy_reset"`     // Si hubo reset en esta ronda
}

// models.go
type RoundEconomyStats struct {
	// 1) Identificación de la ronda y del jugador
	RoundNumber int    `json:"round_number"`
	Name        string `json:"name"`
	Team        string `json:"team,omitempty"`

	// 2) Dinero inicial y bonus
	InitialMoney int `json:"initial_money"`
	LossBonus    int `json:"loss_bonus"`
	//Equipamiento con el que inicia la ronda
	StartRoundItems []RoundItem `json:"start_round_items,omitempty"`

	// 3) Gasto total y compras detalladas
	SpentInBuy int         `json:"spent_in_buy"`
	Purchases  []RoundItem `json:"purchases,omitempty"`

	// 5) Recompensas
	KillReward     int `json:"kill_reward"`
	RewardforPlant int `json:"Reward_for_Plant"`
	// 6) Equipamiento y dinero con el que termina la ronda
	EndRoundItems []RoundItem `json:"end_round_items,omitempty"`
	FinalMoney    int         `json:"final_money"`

	// NUEVO: Contexto de la ronda
	Context RoundContext `json:"context,omitempty"`
}

// MovementLog guarda la posición, velocidad y estado del jugador en un tick concreto.
type MovementLog struct {
	Round     int     `json:"round"`      // Número de ronda
	Tick      int     `json:"tick"`       // Tick del juego
	X         float64 `json:"x_position"` // Posición X
	Y         float64 `json:"y_position"` // Posición Y
	Z         float64 `json:"z_position"` // Posición Z
	Speed     float64 `json:"speed"`      // Velocidad en este tick (aprox. unidades/seg)
	IsDucking bool    `json:"is_ducking"` // Verdadero si el jugador está agachado
	Zone      string  `json:"zone"`       // Zona del mapa donde está el jugador

	// ViewAngles - crítico para análisis de crosshair placement
	Pitch float32 `json:"pitch"` // Ángulo vertical (-90 a 90 grados)
	Yaw   float32 `json:"yaw"`   // Ángulo horizontal (0-360 grados)

	// Team Proximity - para análisis de posicionamiento
	NearestTeammateDist float64 `json:"nearest_teammate_dist"` // Distancia al compañero más cercano
	TeammatesWithin500  int     `json:"teammates_within_500"`  // Cuántos compañeros a <500u
	IsIsolated          bool    `json:"is_isolated"`           // >1000u del compañero más cercano
}

// VisibilitySnapshot guarda qué enemigos ve el jugador en un momento dado
type VisibilitySnapshot struct {
	Tick           int      `json:"tick"`
	VisibleEnemies []uint64 `json:"visible_enemies"` // SteamIDs de enemigos visibles
	LookingAt      *uint64  `json:"looking_at"`      // SteamID del enemigo al que apunta (si hay)
}

// PlayerStateSnapshot - ELIMINADO: Datos redundantes, ya capturados en MovementLog y otros eventos

// PlayerStats representa las estadísticas finales de un jugador.
type PlayerStats struct {
	SteamID          string  `json:"steamID"`
	Name             string  `json:"name"`
	Team             string  `json:"team"`
	Kills            int     `json:"kills"`
	Assists          int     `json:"assists"`
	Deaths           int     `json:"deaths"`
	ADR              float64 `json:"adr"`           // Average Damage per Round
	HSPercentage     float64 `json:"hs_percentage"` // Porcentaje de kills que fueron headshot
	KDRatio          float64 `json:"kd_ratio"`      // Kills/Deaths
	FlashAssists     int     `json:"flash_assists"`
	Avatar           string  `json:"avatar"`
	Rank             string  `json:"rank"`
	Position         int     `json:"position"`          // Orden final (1º, 2º, etc.)
	DistanceTraveled float64 `json:"distance_traveled"` // Suma de distancias recorridas
	EnemiesFlashed   int     `json:"enemiesFlashed"`
	ShotsFired       int     `json:"shots_fired"`
	ShotsConnected   int     `json:"shots_connected"`
	Purchases        int     `json:"purchases"`
	InitialMoney     int     `json:"initial_money"`

	// Nueva métrica: precisión de disparos en porcentaje
	Accuracy float64 `json:"accuracy"`

	// Errores o acciones negativas
	BadFlashCount   int `json:"bad_flash_count"`
	BadMolotovCount int `json:"bad_molotov_count"`

	// Fuego amigo
	TeamKillCount int `json:"team_kill_count"`
	TeamDamage    int `json:"team_damage"`

	// Estadísticas de multi-kill
	DoubleKills int `json:"double_kills"`
	TripleKills int `json:"triple_kills"`
	QuadKills   int `json:"quad_kills"`
	Ace         int `json:"ace"`

	// Clutch: ronda ganada siendo el único vivo
	ClutchWins int `json:"clutch_wins"`

	// Registro detallado de movimiento (sampleo optimizado)
	Movement []MovementLog `json:"movement"`

	// Visibilidad - qué enemigos ve el jugador (sampleo cada 50 ticks)
	Visibility []VisibilitySnapshot `json:"visibility,omitempty"`

	// Contexto de disparos (análisis de aim y movimiento)
	Shots []ShotContext `json:"shots,omitempty"`

	// Reaction Times (tiempos de reacción medidos)
	ReactionTimes []ReactionTimeEvent `json:"reaction_times,omitempty"`

	// Rotaciones (cambios de zona durante la partida)
	Rotations []RotationEvent `json:"rotations,omitempty"`

	// Recoil Analysis (sprays completos)
	RecoilSprays []RecoilSpray `json:"recoil_sprays,omitempty"`

	// Nuevas métricas adicionales:
	BlindKills        int     `json:"blind_kills"`         // Kills realizadas mientras el atacante estaba cegado
	NoScopeKills      int     `json:"noscope_kills"`       // Kills sin usar mira telescópica
	ThroughSmokeKills int     `json:"through_smoke_kills"` // Kills realizadas disparando a través del humo
	Footsteps         int     `json:"footsteps"`           // Número de pasos audibles por enemigos (<1250u)
	Jumps             int     `json:"jumps"`               // Número de saltos (eventos PlayerJump)
	FirstShotHits     int     `json:"first_shot_hits"`     // Primer disparo del engagement que impacta
	FirstShotMisses   int     `json:"first_shot_misses"`   // Primer disparo del engagement que falla
	FirstShotAccuracy float64 `json:"first_shot_accuracy"` // % de acierto del primer disparo

	// Crosshair Placement Metrics
	TimeAtHeadLevel   int     `json:"time_at_head_level"`   // Ticks con crosshair a altura de cabeza
	TimeAtBodyLevel   int     `json:"time_at_body_level"`   // Ticks con crosshair a altura de cuerpo
	TimeAtGroundLevel int     `json:"time_at_ground_level"` // Ticks con crosshair apuntando al suelo
	CrosshairScore    float64 `json:"crosshair_score"`      // Score general de placement (0-100)

	// Reaction Time Metrics
	TotalReactionTimeMs int     `json:"total_reaction_time_ms"` // Suma de todos los reaction times
	ReactionTimeCount   int     `json:"reaction_time_count"`    // Número de reaction times medidos
	AvgReactionTimeMs   float64 `json:"avg_reaction_time_ms"`   // Promedio de reaction time

	// Recoil Control Metrics
	TotalSprays     int     `json:"total_sprays"`      // Número de sprays identificados
	ExcellentSprays int     `json:"excellent_sprays"`  // Sprays con control excelente
	PoorSprays      int     `json:"poor_sprays"`       // Sprays con mal control
	AvgSprayControl float64 `json:"avg_spray_control"` // Score promedio (0-100)

	BombPlants  int `json:"bomb_plants"`  // Número de veces que el jugador plantó la bomba
	BombDefuses int `json:"bomb_defuses"` // Número de veces que el jugador desactivó la bomba
	EntryKills  int `json:"entry_kills"`  // Primer kill de la ronda
	TradeKills  int `json:"trade_kills"`  // Kills realizadas en trade

	// Weapon Stats - NUEVO: estadísticas por arma
	WeaponStats map[string]*WeaponStat `json:"weapon_stats,omitempty"`
}

// WeaponStat almacena estadísticas de una arma específica
type WeaponStat struct {
	Kills           int     `json:"kills"`
	Headshots       int     `json:"headshots"`
	ShotsFired      int     `json:"shots_fired"`
	ShotsHit        int     `json:"shots_hit"`
	Accuracy        float64 `json:"accuracy"`
	DamageDealt     int     `json:"damage_dealt"`
	HeadshotPercent float64 `json:"headshot_percent"`
}

// ShotContext almacena contexto del disparo para análisis de aim
type ShotContext struct {
	Tick       int     `json:"tick"`
	Weapon     string  `json:"weapon"`
	Velocity   float64 `json:"velocity"`   // Velocidad al disparar
	WasMoving  bool    `json:"was_moving"` // Velocity > 50u/s
	WasDucking bool    `json:"was_ducking"`
	WasScoped  bool    `json:"was_scoped"`
	Zone       string  `json:"zone"`
	ViewPitch  float32 `json:"view_pitch"` // Para análisis de recoil
	ViewYaw    float32 `json:"view_yaw"`   // Para análisis de recoil
}

// RecoilSpray almacena información sobre un spray completo
type RecoilSpray struct {
	Round           int     `json:"round"`
	StartTick       int     `json:"start_tick"`
	EndTick         int     `json:"end_tick"`
	ShotCount       int     `json:"shot_count"`
	Weapon          string  `json:"weapon"`
	TotalPitchDelta float32 `json:"total_pitch_delta"` // Cambio total vertical (recoil)
	TotalYawDelta   float32 `json:"total_yaw_delta"`   // Cambio total horizontal
	AvgPitchPerShot float32 `json:"avg_pitch_per_shot"`
	AvgYawPerShot   float32 `json:"avg_yaw_per_shot"`
	ControlQuality  string  `json:"control_quality"` // "excellent", "good", "fair", "poor"
}

// ReactionTimeEvent almacena un evento de reaction time
type ReactionTimeEvent struct {
	Round           int     `json:"round"`
	EnemyID         uint64  `json:"enemy_id"`
	FirstSeenTick   int     `json:"first_seen_tick"`
	FirstShotTick   int     `json:"first_shot_tick"`
	ReactionTimeMs  int     `json:"reaction_time_ms"`
	DistanceToEnemy float64 `json:"distance_to_enemy"`
	KilledEnemy     bool    `json:"killed_enemy"`
}

// RotationEvent almacena información sobre rotaciones del jugador
type RotationEvent struct {
	Round           int    `json:"round"`
	FromZone        string `json:"from_zone"`
	ToZone          string `json:"to_zone"`
	StartTick       int    `json:"start_tick"`
	EndTick         int    `json:"end_tick"`
	DurationMs      int    `json:"duration_ms"`
	TriggerReason   string `json:"trigger_reason"`    // "bomb_planted", "spotted_enemy", "team_call", "unknown"
	WasCorrect      bool   `json:"was_correct"`       // Si rotó al sitio correcto
	WasTooLate      bool   `json:"was_too_late"`      // Llegó después del objetivo
	BombPlantedSite string `json:"bomb_planted_site"` // A qué sitio se plantó
}

// EventLog registra un evento relevante ocurrido durante la partida.
type EventLog struct {
	Round     int    `json:"round"`
	Timestamp int    `json:"timestamp"` // Tick del juego
	EventType string `json:"event_type"`
	Details   string `json:"details"`

	WeaponUsed string `json:"weapon_used,omitempty"` // Para kills, arma usada (opcional)
	Team       string `json:"team,omitempty"`        // Campo para indicar equipo ("T"/"CT" o vacío)

	// Contexto de disparo (para WeaponFire events)
	ShotContext *ShotContext `json:"shot_context,omitempty"`
}

// GrenadeTrajectoryPoint representa un punto en la trayectoria de una granada
type GrenadeTrajectoryPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// GrenadeEvent representa una granada lanzada durante la partida
type GrenadeEvent struct {
	Round             int                    `json:"round"`
	Tick              int                    `json:"tick"` // Tick cuando se lanza
	ThrowerSteamID    string                 `json:"thrower_steam_id"`
	ThrowerName       string                 `json:"thrower_name"`
	ThrowerTeam       string                 `json:"thrower_team"`
	GrenadeType       string                 `json:"grenade_type"` // "Flash", "Smoke", "HE", "Molotov", "Incendiary", "Decoy"
	ThrowPosition     GrenadeTrajectoryPoint `json:"throw_position"`
	ExplosionPosition GrenadeTrajectoryPoint `json:"explosion_position"`

	// Estado del lanzador al tirar
	ThrowerDucking  bool `json:"thrower_ducking"`  // Si estaba agachado
	ThrowerAirborne bool `json:"thrower_airborne"` // Si estaba en el aire (jumpthrow)

	// Eventos de efectividad (específicos por tipo de granada)
	// Flash - usar punteros para que solo aparezcan cuando tienen valor
	EnemiesFlashed   *int     `json:"enemies_flashed,omitempty"`    // Cuántos enemigos cegó
	TeammatesFlashed *int     `json:"teammates_flashed,omitempty"`  // Cuántos aliados cegó (teamflash)
	AvgFlashDuration *float64 `json:"avg_flash_duration,omitempty"` // Duración promedio del cegado (segundos)

	// HE - usar punteros para que solo aparezcan cuando tienen valor
	TotalDamage      *int `json:"total_damage,omitempty"`      // Daño total causado
	EnemiesDamaged   *int `json:"enemies_damaged,omitempty"`   // Cuántos enemigos dañó
	TeammatesDamaged *int `json:"teammates_damaged,omitempty"` // Cuántos aliados dañó

	// Smoke - usar punteros para que solo aparezcan cuando tienen valor
	SmokeStartTick   *int `json:"smoke_start_tick,omitempty"`   // Tick cuando empieza a desplegarse
	SmokeExpiredTick *int `json:"smoke_expired_tick,omitempty"` // Tick cuando se disipa
	SmokeDurationMs  *int `json:"smoke_duration_ms,omitempty"`  // Duración del smoke en ms

	// Molotov/Incendiary - usar punteros para que solo aparezcan cuando tienen valor
	FireStartTick   *int `json:"fire_start_tick,omitempty"`   // Tick cuando empieza a arder
	FireExpiredTick *int `json:"fire_expired_tick,omitempty"` // Tick cuando se apaga
	FireDurationMs  *int `json:"fire_duration_ms,omitempty"`  // Duración del fuego en ms
	FireDamage      *int `json:"fire_damage,omitempty"`       // Daño total del fuego
}

// DemoParseResult almacena la información completa de la partida.
type DemoParseResult struct {
	MatchID       string        `json:"match_id"`
	MapName       string        `json:"map_name"`
	Duration      string        `json:"duration"` // Cambiado de match_duration a duration para consistencia
	Result        string        `json:"result"`
	TeamScore     int           `json:"team_score"`
	OpponentScore int           `json:"opponent_score"`
	Players       []PlayerStats `json:"players"`
	EventLogs     []EventLog    `json:"event_logs"`
	Filename      string        `json:"filename"`
	Date          string        `json:"date"`

	EconomyHistory []*RoundEconomyStats `json:"economy_history"`
	Grenades       []GrenadeEvent       `json:"grenades"`
}
