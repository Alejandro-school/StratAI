package models

// AI_Metadata represents the global context for the match
type AI_Metadata struct {
	MatchID               string  `json:"match_id"`
	MapName               string  `json:"map_name"`
	AnalyzedPlayerSteamID string  `json:"analyzed_player_steam_id,omitempty"` // Optional: if focusing on one player
	FinalScore            string  `json:"final_score"`                        // e.g. "13-11"
	Winner                string  `json:"winner"`                             // "T" or "CT"
	Date                  string  `json:"date"`                               // Date when match was played (ISO 8601 format)
	DurationSeconds       float64 `json:"duration_seconds"`                   // Total match duration in seconds
	TickRate              float64 `json:"tick_rate"`                          // Server tick rate (64 or 128)
	TotalRounds           int     `json:"total_rounds"`                       // Total rounds played
	AverageRank           string  `json:"average_rank,omitempty"`             // e.g. "Faceit Lvl 8"
}

// AI_EconomyMatch represents the economy data for a match
type AI_EconomyMatch struct {
	MatchID string            `json:"match_id"`
	Rounds  []AI_EconomyRound `json:"rounds"`
}

// AI_EconomyRound represents one round of economy
type AI_EconomyRound struct {
	Round   int                       `json:"round"`
	Teams   map[string]AI_EconomyTeam `json:"teams"`
	Players []AI_EconomyPlayer        `json:"players"`
	Events  *AI_EconomyRoundEvents    `json:"events,omitempty"`
}

type AI_EconomyTeam struct {
	TeamID          int     `json:"team_id"` // Demo team/side entity ID (normally 2=T, 3=CT), not a stable roster ID
	ClanName        string  `json:"clan_name,omitempty"`
	TotalMoney      int     `json:"total_money"`
	LossBonus       int     `json:"loss_bonus"`
	AverageMoney    int     `json:"average_money"`
	MoneySpread     int     `json:"money_spread"`     // Difference between richest and poorest
	GiniCoefficient float64 `json:"gini_coefficient"` // Inequality measure (0=equal, 1=unequal)
	RoundsWon       int     `json:"rounds_won"`       // Cumulative rounds won by this team
}

type AI_EconomyPlayer struct {
	SteamID                       uint64          `json:"steam_id,string"`
	Name                          string          `json:"name"`
	Team                          string          `json:"team"`
	SpawnArea                     string          `json:"spawn_area"`
	InitialMoney                  int             `json:"initial_money"`
	MoneyAfterBuy                 int             `json:"money_after_buy"`
	MoneyAfterBuyCalculated       int             `json:"money_after_buy_calculated"`
	MoneyAfterBuyObserved         bool            `json:"money_after_buy_observed"`
	NextRoundMinMoney             int             `json:"next_round_min_money"` // Deprecated legacy field; never used by the canonical projection.
	StartRoundItems               []AI_WeaponItem `json:"start_round_items"`    // Items at round start with EntityID
	EquipmentValueStart           int             `json:"equipment_value_start"`
	EquipmentValueStartCalculated int             `json:"equipment_value_start_calculated"`
	SpentInBuy                    int             `json:"spent_in_buy"`
	Purchases                     []AI_WeaponItem `json:"purchases"` // Purchase events observed with EntityID; may not cover armor or refunds
	PurchasesObservedValue        int             `json:"purchases_observed_value"`
	PurchasesVsSpentDelta         int             `json:"purchases_vs_spent_delta"` // spent_in_buy - purchases_observed_value
	FinalEquipment                []AI_WeaponItem `json:"final_equipment"`          // Items at end of freeze time with EntityID
	FinalEquipmentValue           int             `json:"final_equipment_value"`    // Total value at freeze time end
	FinalEquipmentValueCalculated int             `json:"final_equipment_value_calculated"`
	FinalMoney                    int             `json:"final_money"` // Legacy alias of money_at_round_end
	MoneyAtRoundEnd               int             `json:"money_at_round_end"`
	EquipmentValueEnd             int             `json:"equipment_value_end"` // Legacy alias of equipment_value_end_calculated
	EquipmentValueEndNative       int             `json:"equipment_value_end_native"`
	EquipmentValueEndCalculated   int             `json:"equipment_value_end_calculated"`
	EndEquipment                  []AI_WeaponItem `json:"end_equipment"` // Physical inventory observed at RoundEnd
	Outcome                       string          `json:"outcome"`
	WinReason                     string          `json:"win_reason"`
	Survived                      bool            `json:"survived"`
	Refunds                       []string        `json:"refunds,omitempty"` // Items refunded
}

// AI_WeaponItem represents a weapon/item with tracking info
type AI_WeaponItem struct {
	Weapon              string `json:"weapon"`
	Price               int    `json:"price"`
	PriceStatus         string `json:"price_status"`
	PriceTableVersion   string `json:"price_table_version"`
	EntityID            int64  `json:"entity_id,string"`
	OriginalOwner       string `json:"original_owner,omitempty"`
	OriginalOwnerID     uint64 `json:"original_owner_steam_id,omitempty,string"`
	OriginalOwnerStatus string `json:"original_owner_status"`
	ObservationStatus   string `json:"observation_status"`
}

// AI_Item is kept for backward compatibility but prefer AI_WeaponItem
type AI_Item struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// AI_EconomyRoundEvents contains all economy-related events for a round
type AI_EconomyRoundEvents struct {
	Drops   []AI_EconomyDrop   `json:"drops,omitempty"`
	Pickups []AI_EconomyPickup `json:"pickups,omitempty"` // Non-purchase pickups (ground weapons)
	Refunds []AI_EconomyRefund `json:"refunds,omitempty"`
}

// AI_EconomyDrop represents a weapon drop event
type AI_EconomyDrop struct {
	Tick              int    `json:"tick"`
	Dropper           string `json:"dropper"`
	DropperID         uint64 `json:"dropper_steam_id,string"`
	DropperMoney      int    `json:"dropper_money"`
	Weapon            string `json:"weapon"`
	WeaponValue       int    `json:"weapon_value"`
	PriceStatus       string `json:"price_status"`
	PriceTableVersion string `json:"price_table_version"`
	EntityID          int64  `json:"entity_id,omitempty,string"`
	Receiver          string `json:"receiver,omitempty"` // If picked up by teammate
	ReceiverID        uint64 `json:"receiver_steam_id,omitempty,string"`
	ReceiverMoney     int    `json:"receiver_money,omitempty"`
	PickedUp          bool   `json:"picked_up"` // Was the weapon picked up
}

// AI_EconomyPickup represents picking up a weapon (not from purchase)
type AI_EconomyPickup struct {
	Tick              int    `json:"tick"`
	Player            string `json:"player"`
	PlayerID          uint64 `json:"player_steam_id,string"`
	Weapon            string `json:"weapon"`
	WeaponValue       int    `json:"weapon_value"`
	PriceStatus       string `json:"price_status"`
	PriceTableVersion string `json:"price_table_version"`
	EntityID          int64  `json:"entity_id,omitempty,string"`
	FromDrop          bool   `json:"from_drop"`             // Was from a teammate drop
	FromPlayer        string `json:"from_player,omitempty"` // If from drop, who dropped it
	FromPlayerID      uint64 `json:"from_player_steam_id,omitempty,string"`
}

// AI_EconomyRefund represents an item refund (CS2 only)
type AI_EconomyRefund struct {
	Tick              int    `json:"tick"`
	Player            string `json:"player"`
	PlayerID          uint64 `json:"player_steam_id,string"`
	Weapon            string `json:"weapon"`
	RefundValue       int    `json:"refund_value"`
	PriceStatus       string `json:"price_status"`
	PriceTableVersion string `json:"price_table_version"`
}

// AI_CombatDuel represents a combat engagement (kill or damage exchange)
type AI_CombatDuel struct {
	DuelID     string               `json:"duel_id"`
	Type       string               `json:"type"` // "kill" or "damage"
	Round      int                  `json:"-"`    // Round number (not exported in JSON, used for grouping)
	Tick       int                  `json:"tick"`
	WinnerSide string               `json:"winner_side"` // "CT" or "T" (Attacker side for damage)
	Killer     AI_CombatParticipant `json:"killer"`      // Attacker/Shooter
	Victim     AI_CombatParticipant `json:"victim"`      // Target
	Context    AI_CombatDuelContext `json:"context"`
}

type AI_CombatParticipant struct {
	SteamID        uint64 `json:"steam_id,string"`
	Name           string `json:"name"`
	Team           string `json:"team"`
	MapArea        string `json:"map_area,omitempty"` // [NEW] Area where player is located
	Weapon         string `json:"weapon"`
	HealthBefore   int    `json:"health_before"`   // HP before the duel
	HealthAfter    int    `json:"health_after"`    // HP after the duel
	ArmorBefore    int    `json:"armor_before"`    // Armor before the duel
	ArmorAfter     int    `json:"armor_after"`     // Armor after the duel
	EquipmentValue int    `json:"equipment_value"` // Total inventory value

	// [NEW] Resource state
	AmmoInMagazine int  `json:"ammo_in_magazine,omitempty"` // Bullets left in clip
	AmmoReserve    int  `json:"ammo_reserve,omitempty"`     // Total bullets remaining
	HasHelmet      bool `json:"has_helmet,omitempty"`       // Head protection
	HasDefuser     bool `json:"has_defuser,omitempty"`      // CT defuse kit

	// [NEW] Weapon state (for snipers/scoped weapons)
	IsScoped  bool `json:"is_scoped,omitempty"`  // Is player scoped in
	ZoomLevel int  `json:"zoom_level,omitempty"` // Zoom level (0=none, 1=first, 2=second)

	Metrics AI_CombatParticipantMetrics `json:"metrics"`
}

type AI_CombatParticipantMetrics struct {
	CrosshairPlacementError float64 `json:"crosshair_placement_error,omitempty"` // degrees
	PitchError              float64 `json:"pitch_error,omitempty"`               // degrees (vertical)
	YawError                float64 `json:"yaw_error,omitempty"`                 // degrees (horizontal)
	TimeToDamage            float64 `json:"time_to_damage,omitempty"`            // ms
	Velocity                float64 `json:"velocity,omitempty"`                  // units/s (magnitude)
	IsBlind                 bool    `json:"is_blind"`
	DamageDealt             int     `json:"damage_dealt"` // Damage dealt in the duel
	IsHeadshot              bool    `json:"is_headshot"`  // If the kill was a headshot

	// [NEW] Physics Context - Movement state at moment of engagement
	VelocityX  float64 `json:"velocity_x,omitempty"`  // X component of velocity
	VelocityY  float64 `json:"velocity_y,omitempty"`  // Y component of velocity
	VelocityZ  float64 `json:"velocity_z,omitempty"`  // Z component of velocity
	IsAirborne bool    `json:"is_airborne,omitempty"` // Was jumping/falling
	IsDucking  bool    `json:"is_ducking,omitempty"`  // Was crouching
	IsWalking  bool    `json:"is_walking,omitempty"`  // Was shift-walking

	// [NEW] Mechanics - Shooting technique quality
	CounterStrafeRating float64 `json:"counter_strafe_rating,omitempty"` // 0-1 quality of stop
	ShotsFired          int     `json:"shots_fired,omitempty"`           // Spray bullet index

	// [NEW] Awareness - Sensory state
	FlashDuration float64 `json:"flash_duration,omitempty"` // Seconds remaining of blind
}

type AI_CombatDuelContext struct {
	Distance          float64 `json:"distance"` // units
	IsTrade           bool    `json:"is_trade"`
	ThroughSmoke      bool    `json:"through_smoke"`      // Kill through smoke
	IsWallbang        bool    `json:"is_wallbang"`        // Kill through wall
	PenetratedObjects int     `json:"penetrated_objects"` // Number of penetrated surfaces
	NoScope           *bool   `json:"no_scope,omitempty"` // No scope kill (AWP/Scout only)
	BombPlanted       bool    `json:"bomb_planted"`       // Was bomb planted when kill occurred
	AliveCT           int     `json:"alive_ct"`           // Alive CTs at time of kill
	AliveT            int     `json:"alive_t"`            // Alive Ts at time of kill
	IsOpeningKill     bool    `json:"is_opening_kill"`    // First kill of the round
	Assister          string  `json:"assister,omitempty"` // Player who assisted
	AssisterSteamID   uint64  `json:"assister_steam_id,omitempty,string"`

	// [NEW] Temporal context
	RoundTimeRemaining float64 `json:"round_time_remaining,omitempty"` // Seconds left in round

	// [NEW] Positional context
	HeightDiff float64 `json:"height_diff,omitempty"` // Vertical distance (killer.Z - victim.Z)

	// [NEW] Visibility context - for exposure error detection
	EnemiesVisibleToVictim int `json:"enemies_visible_to_victim,omitempty"` // Enemies visible to victim at death
	EnemiesVisibleToKiller int `json:"enemies_visible_to_killer,omitempty"` // Enemies visible to killer at kill
}

// AI_CombatRound groups combat events by round
type AI_CombatRound struct {
	Round  int             `json:"round"`
	Events []AI_CombatDuel `json:"events"`
}

// AI_CombatExport is the root structure for combat.json
type AI_CombatExport struct {
	Rounds []AI_CombatRound `json:"rounds"`
}

// ============================================================================
// NEW CONSOLIDATED DUEL MODELS
// ============================================================================

// AI_Duel represents a consolidated combat engagement
// Supports single duel (1v1), grenade (1vN), and collateral (1vN with one bullet)
type AI_Duel struct {
	DuelID      string               `json:"duel_id"`
	Type        string               `json:"type"`         // "duel", "grenade", "collateral"
	Outcome     string               `json:"outcome"`      // "kill", "damage", "multi_kill"
	VictimCount int                  `json:"victim_count"` // Number of victims (1 for normal duel)
	Round       int                  `json:"-"`            // For grouping (not exported)
	TickStart   int                  `json:"tick_start"`
	TickEnd     int                  `json:"tick_end"`
	DurationMs  float64              `json:"duration_ms,omitempty"`
	Attacker    AI_DuelParticipant   `json:"attacker"` // The player who dealt damage
	Victims     []AI_DuelParticipant `json:"victims"`  // Array of victims (can be 1 or more)
	Exchanges   []AI_DuelExchange    `json:"exchanges,omitempty"`
	Context     AI_DuelContext       `json:"context"`
}

// AI_DuelParticipant represents aggregated data for a duel participant
type AI_DuelParticipant struct {
	// Identity
	SteamID  uint64     `json:"steam_id,string"`
	Name     string     `json:"name"`
	Team     string     `json:"team"`
	MapArea  string     `json:"map_area,omitempty"`
	Position *AI_Vector `json:"position,omitempty"` // [NEW] Exact coordinate for level filtering

	// Weapon & Combat Stats
	Weapon                   string  `json:"weapon"`
	ActiveWeapon             *string `json:"active_weapon"`
	ActiveWeaponObservation  string  `json:"active_weapon_observation"`
	ActiveWeaponObservedTick *int    `json:"active_weapon_observed_tick"`
	TotalDamageDealt         int     `json:"total_damage_dealt"`
	DamageReceived           int     `json:"damage_received,omitempty"` // For grenade victims
	Hits                     int     `json:"hits"`
	Headshots                int     `json:"headshots,omitempty"`
	ShotsFired               int     `json:"shots_fired,omitempty"`

	// Health & Armor State
	HealthBefore int `json:"health_before"`
	HealthAfter  int `json:"health_after"`
	ArmorBefore  int `json:"armor_before"`
	ArmorAfter   int `json:"armor_after"`

	// Equipment & Resources
	EquipmentValue int `json:"equipment_value,omitempty"`
	AmmoInMagazine int `json:"ammo_in_magazine,omitempty"`
	AmmoReserve    int `json:"ammo_reserve,omitempty"`

	// Aim Metrics (for attacker)
	InitialCrosshairError float64 `json:"initial_crosshair_error,omitempty"`
	PitchError            float64 `json:"pitch_error,omitempty"`
	YawError              float64 `json:"yaw_error,omitempty"`
	TimeToReaction        float64 `json:"time_to_reaction,omitempty"`         // First shot reaction time (ms)
	TimeToFirstDamage     float64 `json:"time_to_first_damage,omitempty"`     // First damage time (ms)
	AvgTimeToReaction     float64 `json:"avg_time_to_reaction,omitempty"`     // Average across exchanges
	AvgTimeToFirstDamage  float64 `json:"avg_time_to_first_damage,omitempty"` // Average across exchanges

	// Player Movement & State
	Velocity                 *float64 `json:"velocity"`
	VelocityAvailable        bool     `json:"velocity_available"`
	VelocitySource           string   `json:"velocity_source"`
	VelocityObservation      string   `json:"velocity_observation"`
	VelocityMeasurementTicks *int     `json:"velocity_measurement_window_ticks"`
	VelocityObservedTick     *int     `json:"velocity_observed_tick"`
	EngagementType           string   `json:"engagement_type,omitempty"` // "peek" (>100 u/s) or "hold" (≤100 u/s)
	IsBlind                  bool     `json:"is_blind,omitempty"`
	IsDucking                bool     `json:"is_ducking,omitempty"`
}

const (
	EngagementPeekVelocityThresholdUPS = 100.0

	VelocityObservationCurrentTick = "current_tick"
	VelocityObservationLastAlive   = "last_alive"
	VelocityObservationUnavailable = "unavailable"

	ActiveWeaponObservationObservedCurrent = "observed_current"
	ActiveWeaponObservationLastObserved    = "last_observed"
	ActiveWeaponObservationUnavailable     = "unavailable"
)

// AI_DuelExchange represents a single damage/kill event within a duel
type AI_DuelExchange struct {
	Tick              int       `json:"tick"`
	Attacker          string    `json:"attacker"`
	Weapon            string    `json:"weapon"`
	Damage            int       `json:"damage"`
	Hitgroup          string    `json:"hitgroup"`
	IsKill            bool      `json:"is_kill,omitempty"`
	TimeToReaction    float64   `json:"time_to_reaction,omitempty"`
	TimeToFirstDamage float64   `json:"time_to_first_damage,omitempty"`
	HasBulletDamage   bool      `json:"has_bullet_damage"`
	BulletDistance    float64   `json:"bullet_distance,omitempty"`
	DamageDirection   AI_Vector `json:"damage_direction,omitempty"`
	Penetrations      int       `json:"penetrations,omitempty"`
	NoScope           bool      `json:"no_scope,omitempty"`
	AttackerInAir     bool      `json:"attacker_in_air,omitempty"`
}

// AI_DuelContext represents contextual information for the duel
type AI_DuelContext struct {
	Distance              float64 `json:"distance"`
	HeightDiff            float64 `json:"height_diff,omitempty"`
	IsTrade               bool    `json:"is_trade"`
	ThroughSmoke          bool    `json:"through_smoke"`
	IsWallbang            bool    `json:"is_wallbang"`
	PenetratedObjects     int     `json:"penetrated_objects"`
	NoScope               *bool   `json:"no_scope,omitempty"`   // Only for scoped weapons (AWP, Scout, etc.)
	ZoomLevel             *int    `json:"zoom_level,omitempty"` // 0=none, 1=first, 2=second (scoped weapons only)
	BombPlanted           bool    `json:"bomb_planted"`
	AliveCT               int     `json:"alive_ct"`
	AliveT                int     `json:"alive_t"`
	IsOpeningKill         bool    `json:"is_opening_kill"`
	EnemiesVisibleToLoser int     `json:"enemies_visible_to_loser,omitempty"`
	RoundTimeRemaining    float64 `json:"round_time_remaining"`
}

// AI_DuelRound groups duels by round
type AI_DuelRound struct {
	Round int       `json:"round"`
	Duels []AI_Duel `json:"duels"`
}

// AI_DuelExport is the new root structure for combat.json
type AI_DuelExport struct {
	Rounds []AI_DuelRound `json:"rounds"`
}

// RawCombatEvent is an intermediate struct to capture events before consolidation
// This is used to buffer damage/kill events during a round before grouping into duels
type RawCombatEvent struct {
	Tick   int
	Round  int
	IsKill bool

	// Attacker identity & state
	AttackerSteamID                  uint64
	AttackerName                     string
	AttackerTeam                     string
	AttackerMapArea                  string
	AttackerHealth                   int
	AttackerArmor                    int
	AttackerEquipmentValue           int
	AttackerAmmoInMagazine           int
	AttackerAmmoReserve              int
	AttackerVelocity                 *float64
	AttackerVelocityAvailable        bool
	AttackerVelocitySource           string
	AttackerVelocityObservation      string
	AttackerVelocityMeasurementTicks *int
	AttackerVelocityObservedTick     *int
	AttackerIsBlind                  bool
	AttackerIsDucking                bool
	AttackerShotsFired               int
	AttackerPosition                 AI_Vector // [NEW]

	// Victim identity & state
	VictimSteamID                  uint64
	VictimName                     string
	VictimTeam                     string
	VictimMapArea                  string
	VictimWeapon                   *string
	VictimWeaponObservation        string
	VictimWeaponObservedTick       *int
	VictimHealthBefore             int
	VictimHealthAfter              int
	VictimArmorBefore              int
	VictimArmorAfter               int
	VictimEquipmentValue           int
	VictimVelocity                 *float64
	VictimVelocityAvailable        bool
	VictimVelocitySource           string
	VictimVelocityObservation      string
	VictimVelocityMeasurementTicks *int
	VictimVelocityObservedTick     *int
	VictimIsBlind                  bool
	VictimIsDucking                bool
	VictimPosition                 AI_Vector // [NEW]

	// Combat details
	Weapon            string
	Damage            int
	Hitgroup          string
	Distance          float64
	HeightDiff        float64
	ThroughSmoke      bool
	IsWallbang        bool
	PenetratedObjects int
	IsHeadshot        bool
	NoScope           bool
	ZoomLevel         int // 0=none, 1=first, 2=second (only for scoped weapons)
	HasBulletDamage   bool
	BulletDistance    float64
	DamageDirection   AI_Vector
	AttackerInAir     bool

	// Aim metrics
	CrosshairError float64
	PitchError     float64
	YawError       float64
	TimeToReaction float64 // Time from visibility to first shot (ms)
	TimeToDamage   float64 // Time from visibility to this damage (ms)
	FirstSeenTick  int     // The tick when attacker first saw victim (for grouping visibility windows)

	// Round context
	BombPlanted            bool
	AliveCT                int
	AliveT                 int
	EnemiesVisibleToVictim int
	RoundTimeRemaining     float64
	IsTrade                bool
	IsOpeningKill          bool
}

// BulletDamageSnapshot is correlated with PlayerHurt/Kill because the native
// BulletDamage event is emitted separately and may arrive before or after it.
type BulletDamageSnapshot struct {
	Tick            int
	AttackerSteamID uint64
	VictimSteamID   uint64
	Distance        float64
	DamageDirection AI_Vector
	NumPenetrations int
	IsNoScope       bool
	IsAttackerInAir bool
}

// AI_GrenadeEvent represents a utility usage
type AI_GrenadeEvent struct {
	Round           int                `json:"-"`
	Type            string             `json:"type"` // "Flashbang", "Smoke", "HE", "Molotov"
	Thrower         string             `json:"thrower"`
	ThrowerSteamID  uint64             `json:"thrower_steam_id,string"`
	TickThrow       int                `json:"tick_throw"`
	TickExplode     int                `json:"tick_explode"`
	ThrowerAreaName string             `json:"thrower_area_name"`
	ThrowerSide     string             `json:"thrower_side,omitempty"` // "CT" or "T"
	LandArea        string             `json:"land_area"`
	StartPosition   AI_Vector          `json:"start_position"`
	EndPosition     AI_Vector          `json:"end_position"`
	BlindedPlayers  []AI_BlindedPlayer `json:"blinded_players,omitempty"`
	EnemiesBlinded  int                `json:"enemies_blinded,omitempty"`
	AlliesBlinded   int                `json:"allies_blinded,omitempty"`
	ThrowViewVector AI_Vector          `json:"throw_view_vector,omitempty"`
	DidBounce       bool               `json:"did_bounce"`

	// New fields for Molotov/HE
	DamageDealt         int                `json:"damage_dealt,omitempty"`
	ArmorDamageDealt    int                `json:"armor_damage_dealt,omitempty"`
	EnemyDamage         int                `json:"enemy_damage,omitempty"`
	EnemyArmorDamage    int                `json:"enemy_armor_damage,omitempty"`
	FriendlyDamage      int                `json:"friendly_damage,omitempty"`
	FriendlyArmorDamage int                `json:"friendly_armor_damage,omitempty"`
	SelfDamage          int                `json:"self_damage,omitempty"`
	SelfArmorDamage     int                `json:"self_armor_damage,omitempty"`
	EnemiesDamaged      int                `json:"enemies_damaged,omitempty"`
	AlliesDamaged       int                `json:"allies_damaged,omitempty"`
	SelfDamaged         bool               `json:"self_damaged,omitempty"`
	Kills               int                `json:"kills,omitempty"`
	Duration            float64            `json:"duration,omitempty"` // Seconds
	DamagedPlayers      []AI_DamagedPlayer `json:"damaged_players,omitempty"`

	// Molotov Extinguished
	Extinguished bool `json:"extinguished,omitempty"`
}

// AI_GrenadeRound represents all grenade events in a specific round
type AI_GrenadeRound struct {
	Round int `json:"round"`

	// Per-round utility counts
	GrenadesThrown          int `json:"grenades_thrown"`
	SmokeThrown             int `json:"smoke_thrown"`
	FlashbangThrown         int `json:"flashbang_thrown"`
	HEThrown                int `json:"he_thrown"`
	MolotovIncendiaryThrown int `json:"molotov_incendiary_thrown"`
	DecoyThrown             int `json:"decoy_thrown"`

	Events []AI_GrenadeEvent `json:"events"`
}

// AI_GrenadeTotals represents overall grenade counts for the entire match.
type AI_GrenadeTotals struct {
	GrenadesThrown          int `json:"grenades_thrown"`
	SmokeThrown             int `json:"smoke_thrown"`
	FlashbangThrown         int `json:"flashbang_thrown"`
	HEThrown                int `json:"he_thrown"`
	MolotovIncendiaryThrown int `json:"molotov_incendiary_thrown"`
	DecoyThrown             int `json:"decoy_thrown"`
}

// AI_GrenadesExport is the top-level structure for grenades.json.
// It contains match totals (outside rounds) plus the per-round breakdown.
type AI_GrenadesExport struct {
	Totals AI_GrenadeTotals  `json:"totals"`
	Rounds []AI_GrenadeRound `json:"rounds"`
}

type AI_BlindedPlayer struct {
	SteamID  uint64  `json:"steam_id,string"`
	Name     string  `json:"name"`
	Duration float32 `json:"duration"`
	Team     string  `json:"team"`
	IsEnemy  bool    `json:"is_enemy"`
	IsSelf   bool    `json:"is_self,omitempty"`
	Relation string  `json:"relation,omitempty"`
}

type AI_DamagedPlayer struct {
	SteamID     uint64 `json:"steam_id,string"`
	Name        string `json:"name"`
	Damage      int    `json:"damage"`
	ArmorDamage int    `json:"armor_damage"`
	Team        string `json:"team"`
	IsEnemy     bool   `json:"is_enemy"`
	IsSelf      bool   `json:"is_self,omitempty"`
	IsKill      bool   `json:"is_kill"`
	Relation    string `json:"relation,omitempty"`
}

type AI_ViewAngle struct {
	Pitch float32 `json:"pitch"`
	Yaw   float32 `json:"yaw"`
}

// AI_TrackingEvent represents a sampled position snapshot (2Hz)
type AI_TrackingEvent struct {
	Tick                         int       `json:"tick"`
	PlayerSteamID                uint64    `json:"player_steam_id,string"`
	Team                         string    `json:"team"` // "CT" or "T"
	Position                     AI_Vector `json:"pos"`
	AreaName                     string    `json:"area_name"`
	ViewAngleYaw                 float32   `json:"view_yaw"`
	ViewAnglePitch               float32   `json:"view_pitch"`
	VelocityVector               AI_Vector `json:"velocity_vector"`
	VelocityAvailable            bool      `json:"velocity_available"`
	VelocitySource               string    `json:"velocity_source"`
	VelocityMeasurementTicks     int       `json:"velocity_measurement_window_ticks"`
	IsWalking                    bool      `json:"is_walking"`
	IsDucking                    bool      `json:"is_ducking"`
	ActiveWeapon                 *string   `json:"active_weapon"`
	ActiveWeaponStatus           string    `json:"active_weapon_status"`
	LastObservedActiveWeapon     *string   `json:"last_observed_active_weapon"`
	LastObservedActiveWeaponTick *int      `json:"last_observed_active_weapon_tick"`
	HasC4                        bool      `json:"has_c4"`
	HasDefuseKit                 bool      `json:"has_defuse_kit"`
	IsPlanting                   bool      `json:"is_planting"`
	IsDefusing                   bool      `json:"is_defusing"`
	Health                       int       `json:"health"`
	Armor                        int       `json:"armor"`
	NearbyTeammates              int       `json:"nearby_teammates"`
	IsAlive                      bool      `json:"is_alive"`
	RoundTimeRemaining           float64   `json:"round_time_remaining"`
	ObjectivePhase               string    `json:"objective_phase"`
	PhaseTimeRemaining           *float64  `json:"phase_time_remaining"`
	RoundClockRemaining          *float64  `json:"round_clock_remaining"`
	BombTimeRemaining            *float64  `json:"bomb_time_remaining"`
}

// AI_TrackingTick represents all player states at a specific tick
type AI_TrackingTick struct {
	Tick    int                `json:"tick"`
	Players []AI_TrackingEvent `json:"players"`
}

// AI_TrackingRound represents a round with all tracking data grouped by tick
type AI_TrackingRound struct {
	Round int               `json:"round"`
	Ticks []AI_TrackingTick `json:"ticks"`
}

// AI_TrackingExport is the top-level structure for tracking.json
type AI_TrackingExport struct {
	Rounds []AI_TrackingRound `json:"rounds"`
}

// AI_TrackingEventWithRound holds an event with its round for grouping during processing
type AI_TrackingEventWithRound struct {
	Round int
	Event AI_TrackingEvent
}

type AI_Vector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// ============================================================================
// PLAYERS SUMMARY MODELS
// ============================================================================

// AI_PlayersSummaryExport represents the root structure for players_summary.json
type AI_PlayersSummaryExport struct {
	MatchID string           `json:"match_id"`
	Players []AI_PlayerStats `json:"players"`
}

// AI_PlayerStats contains comprehensive statistics for a single player
type AI_PlayerStats struct {
	SteamID                string               `json:"steam_id"`
	Name                   string               `json:"name"`
	Team                   string               `json:"team"` // "CT", "T" or "Mixed"
	NativeScoreboard       AI_NativePlayerStats `json:"native_scoreboard"`
	NativeScoreboardStatus string               `json:"native_scoreboard_status"`

	// === CORE STATS ===
	Kills                      int     `json:"kills"`
	Deaths                     int     `json:"deaths"`
	Assists                    int     `json:"assists"`
	KillsObserved              int     `json:"kills_observed"`
	DeathsObserved             int     `json:"deaths_observed"`
	AssistsObserved            int     `json:"assists_observed"`
	KillsNativeMinusObserved   int     `json:"kills_native_minus_observed"`
	DeathsNativeMinusObserved  int     `json:"deaths_native_minus_observed"`
	AssistsNativeMinusObserved int     `json:"assists_native_minus_observed"`
	KDRatio                    float64 `json:"kd_ratio"`
	ADR                        float64 `json:"adr"`
	HSPercentage               float64 `json:"hs_percentage"`
	Headshots                  int     `json:"headshots"`
	KAST                       float64 `json:"kast"`          // % rounds with Kill, Assist, Survival or Traded
	ImpactRating               float64 `json:"impact_rating"` // HLTV Impact

	// === HLTV 2.0 RATING ===
	// 0.0073*KAST + 0.3591*KPR - 0.5329*DPR + 0.2372*Impact + 0.0032*ADR + 0.1587
	HLTVRating float64 `json:"hltv_rating"`

	// === SIDE SPECIFIC ===
	CTRating float64 `json:"ct_rating"`
	TRating  float64 `json:"t_rating"`
	CTADR    float64 `json:"ct_adr"`
	TADR     float64 `json:"t_adr"`

	// === ENTRY / OPENING DUELS ===
	OpeningDuelsAttempted int     `json:"opening_duels_attempted"`
	OpeningDuelsWon       int     `json:"opening_duels_won"`
	OpeningDuelsLost      int     `json:"opening_duels_lost"`
	OpeningSuccessRate    float64 `json:"opening_success_rate"`

	// === TRADING & SUPPORT ===
	TradeKills              int `json:"trade_kills"`                // Killed the killer of a teammate (refrag)
	TradedDeaths            int `json:"traded_deaths"`              // Died and was avenged by teammate
	TradeAttempts           int `json:"trade_attempts"`             // Damaged or killed a teammate's killer in-window
	FailedTradeAttempts     int `json:"failed_trade_attempts"`      // Attempted but did not complete the trade
	UntradeableDeaths       int `json:"untradeable_deaths"`         // No living teammate was observed at death
	NonEvaluableTradeDeaths int `json:"non_evaluable_trade_deaths"` // Missing state, identity or temporal coverage
	FlashAssists            int `json:"flash_assists"`              // Kills on enemies that were blinded by this player

	// === CLUTCHING ===
	Clutches1v1Won     int `json:"clutches_1v1_won"`
	Clutches1v2Won     int `json:"clutches_1v2_won"`
	Clutches1v3Won     int `json:"clutches_1v3_won"`
	Clutches1v4Won     int `json:"clutches_1v4_won"`
	Clutches1v5Won     int `json:"clutches_1v5_won"`
	ClutchAttempts     int `json:"clutch_attempts"`
	ClutchWins         int `json:"clutch_wins"`
	ClutchLosses       int `json:"clutch_losses"`
	ClutchNotEvaluable int `json:"clutch_not_evaluable"`

	// === MULTIKILLS ===
	MultiKills map[string]int `json:"multikills"` // "1k", "2k", "3k", "4k", "ace"

	// === DAMAGE BREAKDOWN ===
	TotalDamage                   int            `json:"total_damage"`
	CombatDamageObserved          int            `json:"combat_damage_observed"`
	CombatDamageUnattributedDelta int            `json:"combat_damage_unattributed_delta"`
	FriendlyDamage                int            `json:"friendly_damage"`
	SelfDamage                    int            `json:"self_damage"`
	UtilityDamage                 int            `json:"utility_damage"`          // Authoritative native scoreboard value.
	UtilityDamageObserved         int            `json:"utility_damage_observed"` // Enemy health damage reconstructed from the utility ledger.
	GrenadeDamage                 map[string]int `json:"grenade_damage"`          // Ledger-derived breakdown: "he", "molotov", "unknown".

	// === AIM METRICS ===
	AvgTimeToReaction          float64        `json:"avg_time_to_reaction,omitempty"`
	TimeToDamageAvgMS          float64        `json:"time_to_damage_avg_ms"`
	CrosshairPlacementAvgError float64        `json:"crosshair_placement_avg_error"`
	CrosshairPlacementPeek     float64        `json:"crosshair_placement_peek"` // Error when peeking (>100 u/s)
	CrosshairPlacementHold     float64        `json:"crosshair_placement_hold"` // Error when holding (≤100 u/s)
	AvgCounterStrafeRating     float64        `json:"avg_counter_strafe_rating,omitempty"`
	ShotsFired                 int            `json:"shots_fired"`
	ShotsHit                   int            `json:"shots_hit"`
	ShotsMissed                int            `json:"shots_missed"`
	AccuracyOverall            float64        `json:"accuracy_overall"`
	BodyPartHits               map[string]int `json:"body_part_hits"` // "head", "chest", "stomach", "legs"

	// === UTILITY EFFICIENCY ===
	GrenadesThrownTotal int `json:"grenades_thrown_total"`

	FlashesThrown                int     `json:"flashes_thrown"`
	EnemiesFlashedTotal          int     `json:"enemies_flashed_total"`
	TeammatesFlashedTotal        int     `json:"teammates_flashed_total"`
	SelfFlashesTotal             int     `json:"self_flashes_total"`
	EnemiesFlashedPerFlash       float64 `json:"enemies_flashed_per_flash"`
	FlashDurationTotal           float64 `json:"flash_duration_total"`
	EnemyFlashDurationTotalMS    float64 `json:"enemy_flash_duration_total_ms"`
	TeammateFlashDurationTotalMS float64 `json:"teammate_flash_duration_total_ms"`
	SelfFlashDurationTotalMS     float64 `json:"self_flash_duration_total_ms"`
	BlindTimePerFlash            float64 `json:"blind_time_per_flash"`

	HEThrown        int     `json:"he_thrown"`
	HEDamagePerNade float64 `json:"he_damage_per_nade"`

	MolotovsThrown        int     `json:"molotovs_thrown"` // Legacy combined Molotov + incendiary count.
	MolotovGrenadesThrown int     `json:"molotov_grenades_thrown"`
	IncendiariesThrown    int     `json:"incendiaries_thrown"`
	MolotovDamagePerNade  float64 `json:"molotov_damage_per_nade"`

	SmokesThrown int `json:"smokes_thrown"`
	DecoysThrown int `json:"decoys_thrown"`

	// === ECONOMY ===
	RoundsPlayed   int `json:"rounds_played"`
	RoundsSurvived int `json:"rounds_survived"`

	// === WEAPON STATS ===
	WeaponStats map[string]AI_WeaponStat `json:"weapon_stats"`
}

// AI_NativePlayerStats exposes scoreboard counters maintained by the game and
// demoinfocs so downstream consumers can reconcile custom aggregations.
type AI_NativePlayerStats struct {
	Kills           int `json:"kills"`
	Deaths          int `json:"deaths"`
	Assists         int `json:"assists"`
	TotalDamage     int `json:"total_damage"`
	UtilityDamage   int `json:"utility_damage"`
	MVPs            int `json:"mvps"`
	Score           int `json:"score"`
	MoneySpentTotal int `json:"money_spent_total"`
}

type AI_WeaponStat struct {
	Kills       int     `json:"kills"`
	Headshots   int     `json:"headshots"`
	Damage      int     `json:"damage"`
	ShotsFired  int     `json:"shots_fired"`
	ShotsHit    int     `json:"shots_hit"`
	ShotsMissed int     `json:"shots_missed"`
	Accuracy    float64 `json:"accuracy"` // (ShotsHit / ShotsFired) * 100
}
