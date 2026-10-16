package models

type SupportNativeMotionReference struct {
	SupportC4Reference
	ClassName *string `json:"class_name"`
}

type SupportNativeMotionState struct {
	Sequence       int                           `json:"sequence"`
	FirstCut       int                           `json:"first_cut"`
	Instance       int                           `json:"instance"`
	Epoch          int                           `json:"epoch"`
	Life           string                        `json:"entity_life"`
	Handle         uint64                        `json:"handle"`
	ClassName      string                        `json:"class_name"`
	CreatedFrame   int                           `json:"created_frame"`
	Fields         map[string]SupportClockScalar `json:"fields"`
	FieldSequences map[string]int                `json:"field_sequences"`
	Ground         SupportNativeMotionReference  `json:"ground"`
	Parent         SupportNativeMotionReference  `json:"parent"`
}

type SupportNativeMotionPlayer struct {
	NativeActor   uint64  `json:"-"`
	ActorAlias    string  `json:"actor_alias"`
	PawnHandle    *uint64 `json:"pawn_handle"`
	PawnLife      *string `json:"pawn_life"`
	StateSequence *int    `json:"state_sequence"`
}

type SupportNativeMotionCut struct {
	SourceSequence   int                         `json:"source_sequence"`
	Frame            int                         `json:"frame"`
	Tick             int                         `json:"tick"`
	Instance         int                         `json:"instance"`
	Epoch            int                         `json:"epoch"`
	ClockOrder       int                         `json:"clock_order"`
	DeliverySequence int                         `json:"delivery_sequence"`
	StateSequence    int                         `json:"state_sequence"`
	Players          []SupportNativeMotionPlayer `json:"players"`
}

type SupportNativeMotionObservations struct {
	Deliveries []SupportObjectivePositionDelivery `json:"deliveries"`
	States     []SupportNativeMotionState         `json:"states"`
	Cuts       []SupportNativeMotionCut           `json:"cuts"`
}
