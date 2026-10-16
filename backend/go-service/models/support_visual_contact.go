package models

// The complete census and ray endpoints are evaluator-only provenance. They
// must never be returned as legal initial information about hidden opponents.
type SupportVisualState struct {
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
}

type SupportVisualPlayer struct {
	SupportNativeMotionPlayer
	// These six freshly read scalars preserve the SDK cell decoder inputs.
	Coordinates map[string]SupportClockScalar `json:"coordinates"`
}

type SupportVisualRay struct {
	Observer int   `json:"observer_index"`
	Subject  int   `json:"subject_index"`
	Clear    *bool `json:"clear"`
}

type SupportVisualCut struct {
	SourceSequence   int                   `json:"source_sequence"`
	Frame            int                   `json:"frame"`
	Tick             int                   `json:"tick"`
	Instance         int                   `json:"instance"`
	Epoch            int                   `json:"epoch"`
	ClockOrder       int                   `json:"clock_order"`
	DeliverySequence int                   `json:"delivery_sequence"`
	StateSequence    int                   `json:"state_sequence"`
	GeometryLoaded   bool                  `json:"geometry_loaded"`
	Players          []SupportVisualPlayer `json:"players"`
	Rays             []SupportVisualRay    `json:"rays"`
}

type SupportVisualObservations struct {
	Deliveries []SupportObjectivePositionDelivery `json:"deliveries"`
	States     []SupportVisualState               `json:"states"`
	Cuts       []SupportVisualCut                 `json:"cuts"`
}
