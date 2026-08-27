package models

import "cs2-demo-service/pkg/objective"

// ========================================
// REPLAY 2D DATA STRUCTURES
// For high-fidelity 2D replay visualization
// ========================================

// ReplayData is the main structure for 2D replay export
type ReplayData struct {
	Metadata ReplayMetadata `json:"metadata"`
	Rounds   []ReplayRound  `json:"rounds"`
}

// ReplayMetadata contains map info for coordinate translation
type ReplayMetadata struct {
	SchemaVersion int       `json:"schema_version"`
	MatchID       string    `json:"match_id"`
	MapName       string    `json:"map_name"`
	TickRate      float64   `json:"tick_rate"`
	SampleRate    int       `json:"sample_rate_ms"` // Milliseconds between frames
	MapConfig     MapConfig `json:"map_config"`
}

// MapConfig contains the coordinate transformation values
// From demoinfocs-golang examples/_assets/metadata/*.txt
type MapConfig struct {
	PosX  float64 `json:"pos_x"`
	PosY  float64 `json:"pos_y"`
	Scale float64 `json:"scale"`
}

// ReplayRound contains all frames and events for one round
type ReplayRound struct {
	Round       int           `json:"round"`
	StartTick   int           `json:"start_tick"`
	EndTick     int           `json:"end_tick"`
	Winner      string        `json:"winner"` // "CT", "T", or ""
	Frames      []ReplayFrame `json:"frames"`
	Events      []ReplayEvent `json:"events"`
	CombatShots []ReplayShot  `json:"combat_shots"`
}

// ReplayFrame is a snapshot at a specific tick
type ReplayFrame struct {
	Tick          int                  `json:"tick"`
	TimeRemaining float64              `json:"time_remaining"`
	Players       []ReplayPlayerState  `json:"players"`
	Projectiles   []ReplayProjectile   `json:"projectiles,omitempty"`
	ActiveEffects []ReplayActiveEffect `json:"active_effects,omitempty"`
	Shots         []ReplayShot         `json:"shots,omitempty"`
	Bomb          *ReplayBombState     `json:"bomb,omitempty"`
}

// ReplayPlayerState is the state of a player at a given tick
type ReplayPlayerState struct {
	SteamID       uint64   `json:"steam_id,string"`
	Name          string   `json:"name"`
	Team          string   `json:"team"` // "CT" or "T"
	X             int      `json:"x"`    // Rounded to int for smaller JSON
	Y             int      `json:"y"`    // Rounded to int for smaller JSON
	Z             float64  `json:"z"`
	Yaw           float32  `json:"yaw"`   // View direction (horizontal)
	Pitch         float32  `json:"pitch"` // View direction (vertical)
	Health        int      `json:"health"`
	Armor         int      `json:"armor"`
	Alive         bool     `json:"alive"`
	Weapon        string   `json:"weapon"`
	Weapons       []string `json:"weapons,omitempty"`
	HasDefuseKit  bool     `json:"has_defuse_kit,omitempty"`
	HasHelmet     bool     `json:"has_helmet,omitempty"`
	HasC4         bool     `json:"has_c4,omitempty"`
	FlashDuration float64  `json:"flash_duration,omitempty"` // Seconds remaining of flash blindness
	Money         int      `json:"money"`
	IsDucking     bool     `json:"is_ducking,omitempty"`
	IsWalking     bool     `json:"is_walking,omitempty"`
	IsScoped      bool     `json:"is_scoped,omitempty"`
	IsReloading   bool     `json:"is_reloading,omitempty"`
	IsDefusing    bool     `json:"is_defusing,omitempty"`
}

// ReplayProjectile is a grenade in flight
type ReplayProjectile struct {
	ID         int64     `json:"id,string"`
	Type       string    `json:"type"` // "smoke", "flashbang", "he", "molotov", "incendiary", "decoy"
	ThrowerID  uint64    `json:"thrower_id,string"`
	X          float64   `json:"x"`
	Y          float64   `json:"y"`
	Z          float64   `json:"z"`
	Trajectory []float64 `json:"trajectory,omitempty"` // Last N positions [x1,y1,x2,y2,...]
}

// ReplayActiveEffect is a deployed smoke, molotov fire, etc.
type ReplayActiveEffect struct {
	Type                string    `json:"type"` // "smoke", "inferno"
	X                   float64   `json:"x"`
	Y                   float64   `json:"y"`
	Radius              float64   `json:"radius,omitempty"`         // For smoke circles
	TimeRemaining       float64   `json:"time_remaining,omitempty"` // Seconds until expiry
	TimeRemainingStatus string    `json:"time_remaining_status"`
	TimeRemainingSource string    `json:"time_remaining_source"`
	StartTick           int       `json:"start_tick,omitempty"` // Stable origin for deterministic playback
	Hull                []float64 `json:"hull,omitempty"`       // For inferno polygons [x1,y1,x2,y2,...]
	Points              []float64 `json:"points,omitempty"`     // Actual inferno fire cells [x1,y1,x2,y2,...]
}

// ReplayShot represents a weapon fire event for visualization
type ReplayShot struct {
	Tick           int     `json:"tick,omitempty"`
	SourceEventID  string  `json:"source_event_id"`
	ShotID         string  `json:"shot_id"`
	ShooterID      uint64  `json:"shooter_id,string"`
	FromX          float64 `json:"from_x"`
	FromY          float64 `json:"from_y"`
	ToX            float64 `json:"to_x"` // Derived view-direction endpoint, never a factual impact.
	ToY            float64 `json:"to_y"`
	Weapon         string  `json:"weapon"`
	Hit            bool    `json:"hit,omitempty"`
	Result         string  `json:"result"`
	ResultStatus   string  `json:"result_status"`
	PositionStatus string  `json:"position_status"`
	PositionSource string  `json:"position_source"`
	EndpointStatus string  `json:"endpoint_status"`
	EndpointSource string  `json:"endpoint_source"`
}

// ReplayBombState represents the bomb's current state
type ReplayBombState struct {
	State               objective.State `json:"state"`
	ObjectivePhase      objective.Phase `json:"objective_phase"`
	IsPlantedNow        bool            `json:"is_planted_now"`
	WasPlantedThisRound bool            `json:"was_planted_this_round"`
	X                   float64         `json:"x"`
	Y                   float64         `json:"y"`
	CarrierID           *uint64         `json:"carrier_id,string"`
	Site                string          `json:"site,omitempty"`
	PlantTick           int             `json:"plant_tick,omitempty"`
	DefuserID           *uint64         `json:"defuser_id,string"`
	PositionStatus      string          `json:"position_status,omitempty"`
	PositionSource      string          `json:"position_source,omitempty"`
}

// ReplayEvent is a discrete event for timeline markers
type ReplayEvent struct {
	ID             string   `json:"id,omitempty"`
	Tick           int      `json:"tick"`
	Type           string   `json:"type"` // "kill", "player_hurt", "bomb_plant", "bomb_defuse", "utility_detonate"
	SourceEventIDs []string `json:"source_event_ids,omitempty"`

	// For kills - includes position data for kill line visualization
	KillerID     uint64  `json:"killer_id,omitempty,string"`
	VictimID     uint64  `json:"victim_id,omitempty,string"`
	AssisterID   uint64  `json:"assister_id,omitempty,string"`
	KillerName   string  `json:"killer_name,omitempty"`
	VictimName   string  `json:"victim_name,omitempty"`
	AssisterName string  `json:"assister_name,omitempty"`
	KillerTeam   string  `json:"killer_team,omitempty"`   // "CT" or "T"
	VictimTeam   string  `json:"victim_team,omitempty"`   // "CT" or "T"
	AssisterTeam string  `json:"assister_team,omitempty"` // "CT" or "T"
	KillerX      float64 `json:"killer_x,omitempty"`      // Position for kill line
	KillerY      float64 `json:"killer_y,omitempty"`
	VictimX      float64 `json:"victim_x,omitempty"`
	VictimY      float64 `json:"victim_y,omitempty"`
	Weapon       string  `json:"weapon,omitempty"`
	Headshot     bool    `json:"headshot,omitempty"`
	Wallbang     bool    `json:"wallbang,omitempty"`
	NoScope      bool    `json:"noscope,omitempty"`

	// For grenades
	GrenadeType       string   `json:"grenade_type,omitempty"`
	UtilityType       string   `json:"utility_type,omitempty"`
	X                 float64  `json:"x"`
	Y                 float64  `json:"y"`
	Z                 float64  `json:"z"`
	ActorID           uint64   `json:"actor_id,omitempty,string"`
	AffectedPlayerIDs []string `json:"affected_player_ids,omitempty"`
	Damage            int      `json:"damage,omitempty"`
	DurationMS        int      `json:"duration_ms,omitempty"`
	DurationStatus    string   `json:"duration_status,omitempty"`
	DurationSource    string   `json:"duration_source,omitempty"`
	PositionStatus    string   `json:"position_status,omitempty"`
	PositionSource    string   `json:"position_source,omitempty"`
	CorrelationStatus string   `json:"correlation_status,omitempty"`
	CorrelationSource string   `json:"correlation_source,omitempty"`
	SourceThrowID     string   `json:"source_throw_id,omitempty"`

	// For bomb events
	Site     string `json:"site,omitempty"`
	PlayerID uint64 `json:"player_id,omitempty,string"`
}

// ========================================
// REPLAY CONTEXT FIELDS
// To be added to DemoContext
// ========================================

// ReplayRoundData is the working data for current round
type ReplayRoundData struct {
	Round     int
	StartTick int
	EndTick   int
	Winner    string
	Frames    []ReplayFrame
	Events    []ReplayEvent
}

// ========================================
// MAP CONFIGS
// Coordinate transformation values from demoinfocs-golang
// ========================================

// MapConfigs contains pre-defined map metadata for coordinate translation
var MapConfigs = map[string]MapConfig{
	"de_dust2":    {PosX: -2476, PosY: 3239, Scale: 4.4},
	"de_mirage":   {PosX: -3230, PosY: 1713, Scale: 5.0},
	"de_inferno":  {PosX: -2087, PosY: 3870, Scale: 4.9},
	"de_ancient":  {PosX: -2953, PosY: 2164, Scale: 5.0},
	"de_anubis":   {PosX: -2796, PosY: 3328, Scale: 5.22},
	"de_nuke":     {PosX: -3453, PosY: 2887, Scale: 7.0},
	"de_overpass": {PosX: -4831, PosY: 1781, Scale: 5.2},
	"de_vertigo":  {PosX: -3168, PosY: 1762, Scale: 4.0},
	"de_train":    {PosX: -2308, PosY: 2078, Scale: 4.082077},
	"de_cache":    {PosX: -2000, PosY: 3250, Scale: 5.5},
}

// GetMapConfig returns the map config for a given map name
// Returns a default config if map is not found
func GetMapConfig(mapName string) MapConfig {
	if config, ok := MapConfigs[mapName]; ok {
		return config
	}
	// Default fallback
	return MapConfig{PosX: -2500, PosY: 3000, Scale: 5.0}
}
