package models

// Versioned extension: legacy time_remaining and archived replay inputs retain
// their original meaning. Null countdowns are not observed zeroes.
type ReplayRoundContext struct {
	SchemaID                  string         `json:"schema_id"`
	Round                     int            `json:"round"`
	Tick                      int            `json:"tick"`
	AvailabilityTick          int            `json:"availability_tick"`
	Phase                     string         `json:"phase"`
	FreezeEndTick             int            `json:"freeze_end_tick"`
	EndTick                   *int           `json:"end_tick"`
	Winner                    string         `json:"winner"`
	ElapsedSeconds            *float64       `json:"elapsed_seconds"`
	ConfiguredDurationSeconds *float64       `json:"configured_duration_seconds"`
	PauseState                string         `json:"pause_state"`
	RoundRemainingSeconds     *float64       `json:"round_remaining_seconds"`
	BombRemainingSeconds      *float64       `json:"bomb_remaining_seconds"`
	ClockStatus               string         `json:"clock_status"`
	ClockReason               string         `json:"clock_reason"`
	ClockEvidence             map[string]any `json:"clock_evidence,omitempty"`
}
