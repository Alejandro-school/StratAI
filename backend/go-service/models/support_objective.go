package models

// Event order is delivery order, not physical subtick order. Public events omit
// actors, sites and coordinates; native player properties are filtered per side.
type SupportObjectiveEvent struct {
	Sequence int    `json:"sequence"`
	Order    int    `json:"order"`
	Frame    int    `json:"frame"`
	Tick     int    `json:"tick"`
	Instance int    `json:"instance"`
	Epoch    int    `json:"epoch"`
	Kind     string `json:"kind"`
}

type SupportObjectivePlayer struct {
	NativeActor uint64                        `json:"-"`
	ActorAlias  string                        `json:"actor_alias"`
	PawnLife    *string                       `json:"pawn_life"`
	Fields      map[string]EconomyEntryScalar `json:"fields"`
}

type SupportObjectiveFrame struct {
	SourceSequence int                           `json:"source_sequence"`
	Frame          int                           `json:"frame"`
	Tick           int                           `json:"tick"`
	Instance       int                           `json:"instance"`
	Epoch          int                           `json:"epoch"`
	Order          int                           `json:"order"`
	EventSequence  int                           `json:"event_sequence"`
	Rules          map[string]EconomyEntryScalar `json:"rules"`
	Players        []SupportObjectivePlayer      `json:"players"`
}

type SupportObjectiveObservations struct {
	Events []SupportObjectiveEvent `json:"events"`
	Frames []SupportObjectiveFrame `json:"frames"`
}
