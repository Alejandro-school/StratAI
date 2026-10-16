package models

// ReplayEventContext is an opt-in callback ledger. It never contains values
// enriched from the completed combat or utility summaries.
type ReplayEventContext struct {
	SchemaID    string               `json:"schema_id"`
	Events      []ReplayContextEvent `json:"events"`
	Abstentions map[string]int       `json:"abstentions,omitempty"`
}

type ReplayContextEvent struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Tick             int      `json:"tick"`
	AvailabilityTick int      `json:"availability_tick"`
	ActorID          uint64   `json:"actor_id,omitempty,string"`
	ActorTeam        string   `json:"actor_team,omitempty"`
	VictimID         uint64   `json:"victim_id,omitempty,string"`
	VictimTeam       string   `json:"victim_team,omitempty"`
	Weapon           string   `json:"weapon,omitempty"`
	UtilityType      string   `json:"utility_type,omitempty"`
	ProjectileID     string   `json:"projectile_id,omitempty"`
	EffectID         string   `json:"effect_id,omitempty"`
	Damage           int      `json:"damage"` // Meaningful only for player_hurt; zero is observed.
	Health           int      `json:"health"`
	X                *float64 `json:"x,omitempty"`
	Y                *float64 `json:"y,omitempty"`
	Z                *float64 `json:"z,omitempty"`
	Source           string   `json:"source"`
}
