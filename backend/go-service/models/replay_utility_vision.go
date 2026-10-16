package models

// Opt-in runtime candidate; it does not alter canonical bundle 3.8 schemas.
// Observations describe geometric possibilities, not human perception.
type ReplayUtilityVision struct {
	SchemaID     string                     `json:"schema_id"`
	Producer     string                     `json:"producer"`
	Observations []ReplayUtilityObservation `json:"observations"`
	Abstentions  map[string]int             `json:"abstentions"`
}

type ReplayUtilityObservation struct {
	ID               string  `json:"id"`
	Tick             int     `json:"tick"`
	AvailabilityTick int     `json:"availability_tick"`
	ObserverID       uint64  `json:"observer_id,string"`
	ObserverSide     string  `json:"observer_side"`
	ObjectID         string  `json:"object_id"`
	Kind             string  `json:"kind"` // launch, projectile or effect
	UtilityType      string  `json:"utility_type"`
	TypeStatus       string  `json:"type_status"`
	X                float64 `json:"x"`
	Y                float64 `json:"y"`
	Z                float64 `json:"z"`
	Source           string  `json:"source"`
	EyeSource        string  `json:"eye_source"`
	PositionBasis    string  `json:"position_basis"`
}
