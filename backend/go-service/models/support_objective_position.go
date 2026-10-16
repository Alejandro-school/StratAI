package models

// Native object geometry is evidence, not automatically player information.
type SupportObjectivePositionDelivery struct {
	Sequence int                `json:"sequence"`
	Frame    int                `json:"frame"`
	Tick     int                `json:"tick"`
	Life     string             `json:"entity_life"`
	Kind     string             `json:"kind"`
	Property string             `json:"property"`
	Previous SupportClockScalar `json:"previous"`
	Value    SupportClockScalar `json:"value"`
}

type SupportObjectivePositionObject struct {
	Life           string                        `json:"entity_life"`
	Handle         uint64                        `json:"handle"`
	ClassName      string                        `json:"class_name"`
	CreatedFrame   int                           `json:"created_frame"`
	Fields         map[string]SupportClockScalar `json:"fields"`
	FieldSequences map[string]int                `json:"field_sequences"`
	Owner          SupportC4Reference            `json:"owner"`
	Defuser        SupportC4Reference            `json:"defuser"`
	Parent         SupportC4Reference            `json:"parent"`
}

type SupportObjectivePositionBirth struct {
	Sequence         int                            `json:"sequence"`
	Frame            int                            `json:"frame"`
	Tick             int                            `json:"tick"`
	Instance         int                            `json:"instance"`
	Epoch            int                            `json:"epoch"`
	DeliverySequence int                            `json:"delivery_sequence"`
	Object           SupportObjectivePositionObject `json:"object"`
}

type SupportObjectivePositionState struct {
	Sequence int                              `json:"sequence"`
	FirstCut int                              `json:"first_cut"`
	Objects  []SupportObjectivePositionObject `json:"objects"`
}

type SupportObjectivePositionCut struct {
	SourceSequence   int `json:"source_sequence"`
	Frame            int `json:"frame"`
	Tick             int `json:"tick"`
	Instance         int `json:"instance"`
	Epoch            int `json:"epoch"`
	ClockOrder       int `json:"clock_order"`
	StateSequence    int `json:"state_sequence"`
	DeliverySequence int `json:"delivery_sequence"`
	BirthSequence    int `json:"birth_sequence"`
}

type SupportObjectivePositionObservations struct {
	Deliveries []SupportObjectivePositionDelivery `json:"deliveries"`
	Births     []SupportObjectivePositionBirth    `json:"births"`
	States     []SupportObjectivePositionState    `json:"states"`
	Cuts       []SupportObjectivePositionCut      `json:"cuts"`
}
