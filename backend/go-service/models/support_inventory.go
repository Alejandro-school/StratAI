package models

// Separate candidate: the legacy C4/replay contracts remain byte-compatible.
type SupportInventoryItem struct {
	Life   string                        `json:"entity_life"`
	Handle uint64                        `json:"handle"`
	Fields map[string]EconomyEntryScalar `json:"fields"`
}

type SupportInventoryState struct {
	Sequence int                    `json:"sequence"`
	FirstCut int                    `json:"first_cut"`
	Items    []SupportInventoryItem `json:"items"`
}

type SupportInventoryCut struct {
	SourceSequence  int `json:"source_sequence"`
	Frame           int `json:"frame"`
	Tick            int `json:"tick"`
	Instance        int `json:"instance"`
	Epoch           int `json:"epoch"`
	C4StateSequence int `json:"c4_state_sequence"`
	StateSequence   int `json:"state_sequence"`
}

type SupportInventoryObservations struct {
	Cuts   []SupportInventoryCut   `json:"cuts"`
	States []SupportInventoryState `json:"states"`
}
