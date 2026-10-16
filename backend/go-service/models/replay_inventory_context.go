package models

// ReplayInventoryContext dates the native equipment observation independently
// of the visual corpse cache. Null fields never mean an observed empty value.
type ReplayInventoryContext struct {
	SchemaID         string            `json:"schema_id"`
	Status           string            `json:"status"`
	ObservationTick  *int              `json:"observation_tick"`
	AvailabilityTick *int              `json:"availability_tick"`
	Source           map[string]string `json:"source,omitempty"`
	Weapon           *string           `json:"weapon"`
	Weapons          []string          `json:"weapons"`
	HasHelmet        *bool             `json:"has_helmet"`
}
