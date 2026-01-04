package models

// TimelineEvent es el contenedor universal para todos los eventos de la timeline
// Cada evento tiene Type + campos específicos según el tipo
type TimelineEvent struct {
	Type  string `json:"type"` // "game_state", "kill", "damage", "grenade", "buy", "bomb", etc.
	Tick  int    `json:"tick"`
	Round int    `json:"round"`

	// GameState sampling (cada 1 segundo = 128 ticks)
	GameState *GameStateSnapshot `json:"game_state,omitempty"`

	// Combat events
	Kill   *KillEvent   `json:"kill,omitempty"`
	Damage *DamageEvent `json:"damage,omitempty"`

	// Grenade events
	Flash   *FlashEvent   `json:"flash,omitempty"`
	HE      *HEEvent      `json:"he,omitempty"`
	Smoke   *SmokeEvent   `json:"smoke,omitempty"`
	Molotov *MolotovEvent `json:"molotov,omitempty"`

	// Economy
	Buy *BuyEvent `json:"buy,omitempty"`

	// Bomb
	Bomb *BombEvent `json:"bomb,omitempty"`

	// Round lifecycle
	RoundStart *RoundStartEvent `json:"round_start,omitempty"`
	RoundEnd   *RoundEndEvent   `json:"round_end,omitempty"`

	// Tactical situations (detectados)
	Tactical *TacticalEvent `json:"tactical,omitempty"`

	// Chat
	Chat *ChatEvent `json:"chat,omitempty"`

	// Grenade Trajectories
	GrenadeTrajectory *GrenadeTrajectoryEvent `json:"grenade_trajectory,omitempty"`
}

// GameStateSnapshot captura el estado completo del juego en un tick
// Se genera cada 128 ticks (1 segundo) para dar contexto completo
type GameStateSnapshot struct {
	CTScore       int                   `json:"ct_score"`
	TScore        int                   `json:"t_score"`
	TimeRemaining float64               `json:"time_remaining"` // Segundos restantes en ronda
	BombPlanted   bool                  `json:"bomb_planted"`
	BombSite      string                `json:"bomb_site,omitempty"`
	Players       []PlayerStateSnapshot `json:"players"`
	CTAlive       int                   `json:"ct_alive"`
	TAlive        int                   `json:"t_alive"`
	Phase         string                `json:"phase"` // "freezetime", "live", "over"
}

// PlayerStateSnapshot es el estado de un jugador en un momento dado
type PlayerStateSnapshot struct {
	Name            string   `json:"name"`
	SteamID         uint64   `json:"steam_id"`
	Team            string   `json:"team"` // "CT", "T"
	IsAlive         bool     `json:"is_alive"`
	HP              int      `json:"hp"`
	Armor           int      `json:"armor"`
	HasHelmet       bool     `json:"has_helmet"`
	Money           int      `json:"money"`
	EquipValue      int      `json:"equip_value"`
	X               float64  `json:"x"`
	Y               float64  `json:"y"`
	Z               float64  `json:"z"`
	VelocityX       float64  `json:"velocity_x"`
	VelocityY       float64  `json:"velocity_y"`
	VelocityZ       float64  `json:"velocity_z"`
	ViewX           float64  `json:"view_x"` // Pitch
	ViewY           float64  `json:"view_y"` // Yaw
	ActiveWeapon    string   `json:"active_weapon"`
	PrimaryWeapon   string   `json:"primary_weapon,omitempty"`
	SecondaryWeapon string   `json:"secondary_weapon,omitempty"`
	Grenades        []string `json:"grenades,omitempty"`
	HasDefuser      bool     `json:"has_defuser"`
	FlashDuration   float64  `json:"flash_duration"` // Duración flash actual
}

// RoundStartEvent marca inicio de ronda
type RoundStartEvent struct {
	RoundNumber  int                  `json:"round_number"`
	CTStartMoney int                  `json:"ct_start_money"`
	TStartMoney  int                  `json:"t_start_money"`
	CTScore      int                  `json:"ct_score"`
	TScore       int                  `json:"t_score"`
	CTLossBonus  int                  `json:"ct_loss_bonus"`
	TLossBonus   int                  `json:"t_loss_bonus"`
	Players      []PlayerEconomyState `json:"players"` // Estado económico inicial de los jugadores
}

type PlayerEconomyState struct {
	SteamID   uint64   `json:"steam_id"`
	Name      string   `json:"name"`
	Team      string   `json:"team"`
	AreaName  string   `json:"area_name,omitempty"` // Spawn area at round start
	Money     int      `json:"money"`
	Inventory []string `json:"inventory"` // Items al inicio de la ronda
}

// RoundEndEvent marca fin de ronda
type RoundEndEvent struct {
	RoundNumber int                      `json:"round_number"`
	Winner      string                   `json:"winner"` // "CT", "T"
	Reason      string                   `json:"reason"` // "BombDefused", "TargetBombed", "TerroristsEliminated", etc.
	CTScore     int                      `json:"ct_score"`
	TScore      int                      `json:"t_score"`
	Survivors   []PlayerSurvivalSnapshot `json:"survivors,omitempty"` // Estado de supervivientes al final de ronda
}

// PlayerSurvivalSnapshot captura el equipo que un jugador llevaba al final de la ronda
type PlayerSurvivalSnapshot struct {
	SteamID                uint64   `json:"steam_id"`
	Name                   string   `json:"name"`
	Team                   string   `json:"team"`
	Survived               bool     `json:"survived"`
	EndRoundItems          []string `json:"end_round_items,omitempty"` // Inventario al final de ronda
	EquipmentValueSurvived int      `json:"equipment_value_survived"`  // Valor total del equipo guardado
}

// BuyEvent representa una compra de un jugador
type BuyEvent struct {
	Player    string `json:"player"`
	SteamID   uint64 `json:"steam_id"`
	Item      string `json:"item"`
	Cost      int    `json:"cost"`
	MoneyLeft int    `json:"money_left"`
	Refund    bool   `json:"refund,omitempty"`
}

// TacticalEvent representa situaciones tácticas detectadas
type TacticalEvent struct {
	SituationType string                 `json:"situation_type"` // "clutch", "first_kill", "trade", "save", "execute"
	Players       []string               `json:"players"`        // Jugadores involucrados
	Details       map[string]interface{} `json:"details,omitempty"`
}

// RoundTimeline contiene todos los eventos de una ronda ordenados por tick
type RoundTimeline struct {
	RoundNumber int             `json:"round_number"`
	Events      []TimelineEvent `json:"events"`
	StartTick   int             `json:"start_tick"`
	EndTick     int             `json:"end_tick"`
}

// MatchMetadata contiene información general de la partida
type MatchMetadata struct {
	MatchID      string              `json:"match_id"`
	Map          string              `json:"map"`
	Date         string              `json:"date"`
	TickRate     float64             `json:"tick_rate"`
	Duration     float64             `json:"duration"` // Segundos
	CTScore      int                 `json:"ct_score"`
	TScore       int                 `json:"t_score"`
	Winner       string              `json:"winner"`
	Players      []PlayerMetadata    `json:"players"`
	RoundSummary []RoundSummary      `json:"round_summary"`
	EconomyStats []RoundEconomyStats `json:"economy_stats"`
	TotalRounds  int                 `json:"total_rounds"`
}

// PlayerMetadata contiene información básica y estadísticas de un jugador
type PlayerMetadata struct {
	Name    string  `json:"name"`
	SteamID uint64  `json:"steam_id"`
	Team    string  `json:"team"` // "CT" o "T" (team inicial)
	Kills   int     `json:"kills"`
	Deaths  int     `json:"deaths"`
	Assists int     `json:"assists"`
	Damage  int     `json:"damage"`
	MVPs    int     `json:"mvps"`
	KDRatio float64 `json:"kd_ratio"`
	ADR     float64 `json:"adr"`
}

// RoundSummary resumen de una ronda para metadata
type RoundSummary struct {
	Round   int    `json:"round"`
	Winner  string `json:"winner"`
	Reason  string `json:"reason"`
	CTScore int    `json:"ct_score"`
	TScore  int    `json:"t_score"`
}

// ChatEvent representa un mensaje de chat
type ChatEvent struct {
	SenderName    string `json:"sender_name"`
	SenderSteamID uint64 `json:"sender_steam_id"`
	SenderTeam    string `json:"sender_team"`
	Text          string `json:"text"`
	IsTeamChat    bool   `json:"is_team_chat"`
}

// GrenadeTrajectoryEvent representa la trayectoria de una granada
type GrenadeTrajectoryEvent struct {
	GrenadeType     string `json:"grenade_type"`
	Thrower         string `json:"thrower"`
	ThrowerID       uint64 `json:"thrower_id"`
	TickThrow       int    `json:"tick_throw"` // Tick when the throw started
	ThrowerAreaName string `json:"thrower_area_name,omitempty"`
	LandAreaName    string `json:"land_area_name,omitempty"`
	Positions       []XYZ  `json:"positions"` // Lista de coordenadas de la trayectoria
	LandPosition    XYZ    `json:"land_position"`
}

type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// --- Analysis Structures (For AI) ---

// CombatAnalysis agrupa eventos de combate por ronda para la IA
type CombatAnalysis struct {
	Rounds []CombatRound `json:"rounds"`
}

type CombatRound struct {
	RoundNumber int            `json:"round_number"`
	Kills       []CombatKill   `json:"kills"`
	Damages     []CombatDamage `json:"damages"`
}

type CombatKill struct {
	Tick         int     `json:"tick"`
	Time         float64 `json:"time"` // Segundos desde inicio ronda
	AttackerName string  `json:"attacker_name"`
	AttackerTeam string  `json:"attacker_team"`
	VictimName   string  `json:"victim_name"`
	VictimTeam   string  `json:"victim_team"`
	Weapon       string  `json:"weapon"`
	Headshot     bool    `json:"headshot"`
	Wallbang     bool    `json:"wallbang"`
	ThroughSmoke bool    `json:"through_smoke"`
	Blind        bool    `json:"blind"`
	Distance     float64 `json:"distance"`
	IsEntry      bool    `json:"is_entry"` // Primera kill de la ronda

	// Nuevas métricas tácticas calculadas
	IsTraded            bool    `json:"is_traded"` // ¿Fue vengado rápidamente?
	TradeKillerName     string  `json:"trade_killer_name,omitempty"`
	TimeUntilTrade      float64 `json:"time_until_trade,omitempty"`
	NearestTeammateDist float64 `json:"nearest_teammate_dist"` // Distancia al aliado más cercano (soporte)
	NearestTeammateName string  `json:"nearest_teammate_name"`

	// Mechanics
	CounterStrafeRating float64 `json:"counter_strafe_rating,omitempty"`
	TimeToDamage        float64 `json:"time_to_damage,omitempty"`
}

type CombatDamage struct {
	Tick         int    `json:"tick"`
	AttackerName string `json:"attacker_name"`
	VictimName   string `json:"victim_name"`
	Weapon       string `json:"weapon"`
	Damage       int    `json:"damage"`
	ArmorDamage  int    `json:"armor_damage"`
	HitGroup     string `json:"hit_group"`
}

// PositioningAnalysis agrupa datos de posicionamiento clave por ronda
type PositioningAnalysis struct {
	Rounds []PositioningRound `json:"rounds"`
}

type PositioningRound struct {
	RoundNumber int `json:"round_number"`
	// Posiciones iniciales (post-freezetime) para detectar setups
	InitialPositions []PlayerPosition `json:"initial_positions"`
	// Dónde ocurrió la primera kill (punto de contacto)
	FirstKillPosition *PositionEvent `json:"first_kill_position,omitempty"`
	// Dónde se plantó la bomba
	BombPlantPosition *PositionEvent `json:"bomb_plant_position,omitempty"`
}

type PlayerPosition struct {
	Name     string  `json:"name"`
	Team     string  `json:"team"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Z        float64 `json:"z"`
	AreaName string  `json:"area_name"` // "Long A", "Mid", etc.
}

type PositionEvent struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Z        float64 `json:"z"`
	AreaName string  `json:"area_name"`
}

// UtilityAnalysis agrupa uso de granadas por ronda
type UtilityAnalysis struct {
	Rounds []UtilityRound `json:"rounds"`
}

type UtilityRound struct {
	RoundNumber int            `json:"round_number"`
	Grenades    []UtilityEvent `json:"grenades"`
}

type UtilityEvent struct {
	Tick        int     `json:"tick"`
	Type        string  `json:"type"` // "flash", "he", "smoke", "molotov"
	ThrowerName string  `json:"thrower_name"`
	ThrowerTeam string  `json:"thrower_team"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Z           float64 `json:"z"`

	// Métricas de efectividad
	EnemiesBlinded   int     `json:"enemies_blinded,omitempty"`   // Solo para flash
	TeammatesBlinded int     `json:"teammates_blinded,omitempty"` // Solo para flash
	TotalBlindTime   float64 `json:"total_blind_time,omitempty"`  // Suma de duración
}

// --- Objective & Clutch Analysis ---

type ObjectiveAnalysis struct {
	Rounds []ObjectiveRound `json:"rounds"`
}

type ObjectiveRound struct {
	RoundNumber   int        `json:"round_number"`
	BombPlant     *BombPoint `json:"bomb_plant,omitempty"`
	BombDefuse    *BombPoint `json:"bomb_defuse,omitempty"`
	BombExplosion *BombPoint `json:"bomb_explosion,omitempty"`
	Winner        string     `json:"winner"` // "CT" or "T"
	WinReason     string     `json:"win_reason"`
}

type BombPoint struct {
	Tick           int     `json:"tick"`
	Time           float64 `json:"time"` // Segundos desde inicio ronda
	PlayerName     string  `json:"player_name,omitempty"`
	Site           string  `json:"site,omitempty"` // "A" or "B"
	PlayersAliveCT int     `json:"players_alive_ct"`
	PlayersAliveT  int     `json:"players_alive_t"`
}

type ClutchAnalysis struct {
	Rounds []ClutchRound `json:"rounds"`
}

type ClutchRound struct {
	RoundNumber int           `json:"round_number"`
	Clutches    []ClutchEvent `json:"clutches"`
}

type ClutchEvent struct {
	PlayerName    string `json:"player_name"`
	Team          string `json:"team"`           // "CT" or "T"
	OpponentCount int    `json:"opponent_count"` // El 'X' en 1vX (1, 2, 3, 4, 5)
	Won           bool   `json:"won"`
	HasDied       bool   `json:"has_died"`   // Si murió durante el clutch
	TickStart     int    `json:"tick_start"` // Cuando se quedó solo
}
