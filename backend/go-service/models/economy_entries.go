package models

// Scalars retain the SDK type. Availability belongs to the fresh delivery cut.
type EconomyEntryScalar struct {
	Status     string  `json:"status"`
	NativeType *string `json:"native_type"`
	Value      any     `json:"value"`
}
type EconomyEntry struct {
	Index            int                           `json:"index"`
	Fields           map[string]EconomyEntryScalar `json:"fields"`
	Item             EconomyEntityReference        `json:"item"`
	EntityDefinition EconomyEntryScalar            `json:"entity_definition"`
}
type EconomyEntryPawn struct {
	EntityAlias string         `json:"entity_alias"`
	Status      string         `json:"status"`
	Length      *int           `json:"length"`
	Entries     []EconomyEntry `json:"entries"`
}
type EconomyEntryState struct {
	Sequence       int                `json:"sequence"`
	FirstCut       int                `json:"first_cut"`
	OwnershipState int                `json:"ownership_state_sequence"`
	Pawns          []EconomyEntryPawn `json:"pawns"`
}
type EconomyEntryCut struct {
	OwnershipCut  int `json:"ownership_cut"`
	StateSequence int `json:"state_sequence"`
}
type EconomyEntryObservations struct {
	Cuts   []EconomyEntryCut   `json:"cuts"`
	States []EconomyEntryState `json:"states"`
}
