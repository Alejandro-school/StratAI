package models

// Native values retain presence and decoded type. DeclaredType is read from the
// demo's flattened serializer, never inferred from the value or property name.
type SupportClockScalar struct {
	Status       string  `json:"status"`
	NativeType   *string `json:"native_type"`
	DeclaredType *string `json:"declared_type"`
	Value        any     `json:"value"`
}

type SupportClockMessage struct {
	Order int                `json:"order"`
	Frame int                `json:"frame"`
	Tick  int                `json:"tick"`
	Value SupportClockScalar `json:"value"`
}

type SupportClockObject struct {
	Life      string                        `json:"entity_life"`
	Handle    uint64                        `json:"handle"`
	ClassName string                        `json:"class_name"`
	Fields    map[string]SupportClockScalar `json:"fields"`
	Defuser   *SupportC4Reference           `json:"defuser"`
}

type SupportClockPlayer struct {
	NativeActor uint64                        `json:"-"`
	ActorAlias  string                        `json:"actor_alias"`
	PawnLife    *string                       `json:"pawn_life"`
	PawnHandle  *uint64                       `json:"pawn_handle"`
	Fields      map[string]SupportClockScalar `json:"fields"`
}

type SupportObjectiveClockFrame struct {
	SourceSequence  int                             `json:"source_sequence"`
	Frame           int                             `json:"frame"`
	Tick            int                             `json:"tick"`
	Instance        int                             `json:"instance"`
	Epoch           int                             `json:"epoch"`
	ObjectiveOrder  int                             `json:"objective_order"`
	C4StateSequence int                             `json:"c4_state_sequence"`
	Order           int                             `json:"order"`
	Network         map[string]*SupportClockMessage `json:"network"`
	Rules           *SupportClockObject             `json:"rules"`
	Players         []SupportClockPlayer            `json:"players"`
	Bombs           []SupportClockObject            `json:"bombs"`
	Items           []SupportClockObject            `json:"items"`
}

type SupportObjectiveClockObservations struct {
	Frames []SupportObjectiveClockFrame `json:"frames"`
}
