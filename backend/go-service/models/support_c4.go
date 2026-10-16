package models

// References are read again at every cut. Life aliases identify native objects,
// not players or features; a full handle alone cannot identify a reused object.
type SupportC4Reference struct {
	Status     string  `json:"status"`
	Handle     *uint64 `json:"handle"`
	TargetLife *string `json:"target_life"`
}

type SupportC4Entity struct {
	Life       string             `json:"entity_life"`
	Handle     uint64             `json:"handle"`
	ClassName  string             `json:"class_name"`
	Definition EconomyEntryScalar `json:"definition"`
	Owner      SupportC4Reference `json:"owner"`
}

type SupportC4Player struct {
	NativeActor     uint64               `json:"-"`
	ActorAlias      string               `json:"actor_alias"`
	PawnLife        *string              `json:"pawn_life"`
	PawnHandle      *uint64              `json:"pawn_handle"`
	InventoryStatus string               `json:"inventory_status"`
	InventoryLength *int                 `json:"inventory_length"`
	Slots           []SupportC4Reference `json:"slots"`
}

type SupportC4State struct {
	Sequence int               `json:"sequence"`
	FirstCut int               `json:"first_cut"`
	Players  []SupportC4Player `json:"players"`
	Entities []SupportC4Entity `json:"entities"`
}

type SupportC4Cut struct {
	SourceSequence int `json:"source_sequence"`
	Frame          int `json:"frame"`
	Tick           int `json:"tick"`
	Instance       int `json:"instance"`
	Epoch          int `json:"epoch"`
	ObjectiveOrder int `json:"objective_order"`
	StateSequence  int `json:"state_sequence"`
}

type SupportC4Observations struct {
	Cuts   []SupportC4Cut   `json:"cuts"`
	States []SupportC4State `json:"states"`
}
