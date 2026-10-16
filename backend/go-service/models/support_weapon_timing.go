package models

// Native object lives are provenance, never model features.
type SupportWeaponTimingState struct {
	Sequence       int                           `json:"sequence"`
	FirstCut       int                           `json:"first_cut"`
	Instance       int                           `json:"instance"`
	Epoch          int                           `json:"epoch"`
	Life           string                        `json:"entity_life"`
	Handle         uint64                        `json:"handle"`
	ClassName      string                        `json:"class_name"`
	Kind           string                        `json:"kind"`
	CreatedFrame   int                           `json:"created_frame"`
	Fields         map[string]SupportClockScalar `json:"fields"`
	FieldSequences map[string]int                `json:"field_sequences"`
	Reference      SupportC4Reference            `json:"reference"`
}

type SupportWeaponTimingPlayer struct {
	NativeActor         uint64  `json:"-"`
	ActorAlias          string  `json:"actor_alias"`
	PawnHandle          *uint64 `json:"pawn_handle"`
	PawnLife            *string `json:"pawn_life"`
	PawnStateSequence   *int    `json:"pawn_state_sequence"`
	WeaponStateSequence *int    `json:"weapon_state_sequence"`
	WeaponLife          *string `json:"weapon_life"`
	WeaponStatus        string  `json:"weapon_status"`
}

type SupportWeaponTimingCut struct {
	SourceSequence   int                         `json:"source_sequence"`
	Frame            int                         `json:"frame"`
	Tick             int                         `json:"tick"`
	Instance         int                         `json:"instance"`
	Epoch            int                         `json:"epoch"`
	ClockOrder       int                         `json:"clock_order"`
	DeliverySequence int                         `json:"delivery_sequence"`
	StateSequence    int                         `json:"state_sequence"`
	Players          []SupportWeaponTimingPlayer `json:"players"`
}

type SupportWeaponTimingObservations struct {
	Deliveries []SupportObjectivePositionDelivery `json:"deliveries"`
	States     []SupportWeaponTimingState         `json:"states"`
	Cuts       []SupportWeaponTimingCut           `json:"cuts"`
}
