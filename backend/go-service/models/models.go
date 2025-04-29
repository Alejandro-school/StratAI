package models

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
)

type RoundItem struct {
	Name      string `json:"name"`
	Price     int    `json:"price"`
	Armor     int    `json:"armor,omitempty"`
	HasDefuse bool   `json:"has_defuse,omitempty"`
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
}

// MovementLog guarda la posición, velocidad y estado del jugador en un tick concreto.
type MovementLog struct {
	Tick      int     `json:"tick"`       // Tick del juego
	X         float64 `json:"x_position"` // Posición X
	Y         float64 `json:"y_position"` // Posición Y
	Z         float64 `json:"z_position"` // Posición Z
	Speed     float64 `json:"speed"`      // Velocidad en este tick (aprox. unidades/seg)
	IsDucking bool    `json:"is_ducking"` // Verdadero si el jugador está agachado
}

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

	// Disparos "raros"
	AccidentalShots int `json:"accidental_shots"`

	// Estadísticas de multi-kill
	DoubleKills int `json:"double_kills"`
	TripleKills int `json:"triple_kills"`
	QuadKills   int `json:"quad_kills"`
	Ace         int `json:"ace"`

	// Clutch: ronda ganada siendo el único vivo
	ClutchWins int `json:"clutch_wins"`

	// Contadores de ticks (agachado, caminando, corriendo)
	CrouchTicks int `json:"crouch_ticks"`
	WalkTicks   int `json:"walk_ticks"`
	RunTicks    int `json:"run_ticks"`

	// Registro detallado de movimiento (tick a tick)
	Movement []MovementLog `json:"movement"`

	// Nuevas métricas adicionales:
	BlindKills        int `json:"blind_kills"`         // Kills realizadas mientras el atacante estaba cegado
	NoScopeKills      int `json:"noscope_kills"`       // Kills sin usar mira telescópica
	ThroughSmokeKills int `json:"through_smoke_kills"` // Kills realizadas disparando a través del humo
	Footsteps         int `json:"footsteps"`           // Número de pasos (eventos Footstep)
	Jumps             int `json:"jumps"`               // Número de saltos (eventos PlayerJump)
	Sounds            int `json:"sounds"`              // Número de sonidos emitidos (eventos PlayerSound)
	SmokesThrown      int `json:"smokes_thrown"`       // Número de granadas de humo lanzadas
	DecoysThrown      int `json:"decoys_thrown"`       // Número de decoys lanzados
	MolotovsThrown    int `json:"molotovs_thrown"`     // Número de molotovs lanzadas
	HeThrown          int `json:"he_thrown"`           // Número de HE lanzadas
	FlashesThrown     int `json:"flashes_thrown"`      // Número de granadas flash lanzadas
	UtilityDamage     int `json:"utility_damage"`      // Daño infligido con utilidades (molotov, HE, incendiarias)
	BombPlants        int `json:"bomb_plants"`         // Número de veces que el jugador plantó la bomba
	BombDefuses       int `json:"bomb_defuses"`        // Número de veces que el jugador desactivó la bomba
	EntryKills        int `json:"entry_kills"`         // Primer kill de la ronda
	TradeKills        int `json:"trade_kills"`         // Kills realizadas en trade
}

// EventLog registra un evento relevante ocurrido durante la partida.
type EventLog struct {
	Round     int    `json:"round"`
	Timestamp int    `json:"timestamp"` // Tick del juego
	EventType string `json:"event_type"`
	Details   string `json:"details"`

	WeaponUsed string `json:"weapon_used,omitempty"` // Para kills, arma usada (opcional)
	Team       string `json:"team,omitempty"`        // Campo para indicar equipo ("T"/"CT" o vacío)
}

// MolotovData almacena información del lanzamiento de un molotov
type MolotovData struct {
	Thrower  *common.Player
	Position r3.Vector
}

// ProjectileTrajectoryEntry representa un punto de la trayectoria de una granada
type ProjectileTrajectoryEntry struct {
	FrameID int     `json:"frame_id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
	TimeMS  int64   `json:"time_ms"`
}

// -------------------------------------
// NUEVO: estructuras para granadas
// -------------------------------------

// Position es una pequeña ayuda para exportar coordenadas de explosión
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// GrenadeMetadata guarda info interna de cada granada durante el parseo
type GrenadeMetadata struct {
	ProjectileID      int                         `json:"projectile_id"`
	Round             int                         `json:"round"`
	ThrowerSteamID    uint64                      `json:"thrower_steamid"`
	ThrowerName       string                      `json:"thrower_name"`
	ThrowerTeam       string                      `json:"thrower_team"`
	NadeType          string                      `json:"nade_type"`
	Exploded          bool                        `json:"exploded"`
	ExplosionPosition *Position                   `json:"explosion_position,omitempty"`
	Trajectory        []ProjectileTrajectoryEntry `json:"trajectory"`
	ThrowTick         int                         `json:"-"` // ★ tick en el que se lanzó (sólo para ordenar)
}

// DemoParseResult almacena la información completa de la partida.
type DemoParseResult struct {
	MatchID       string        `json:"match_id"`
	MapName       string        `json:"map_name"`
	Duration      string        `json:"match_duration"`
	Result        string        `json:"result"`
	TeamScore     int           `json:"team_score"`
	OpponentScore int           `json:"opponent_score"`
	Players       []PlayerStats `json:"players"`
	EventLogs     []EventLog    `json:"event_logs"`
	Filename      string        `json:"filename"`
	Date          string        `json:"date"`

	EconomyHistory []*RoundEconomyStats `json:"economy_history"`

	// Trayectorias de proyectiles (utilidad interna, no se exporta)
	Trajectories map[int][]ProjectileTrajectoryEntry `json:"-"`

	// ★ NUEVO: lista final y completa de todas las granadas con su trayectoria
	Grenades []GrenadeMetadata `json:"grenades,omitempty"`

	// Mapas internos usados durante el parseo (no exportar)
	GrenadeMetas        map[int]*GrenadeMetadata            `json:"-"`
	GrenadeTrajectories map[int][]ProjectileTrajectoryEntry `json:"-"`
}
