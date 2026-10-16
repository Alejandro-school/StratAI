package models

import st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"

type EconomyCashflowController struct {
	Alias          string `json:"alias"`
	Index          int    `json:"index"`
	Serial         int    `json:"serial"`
	CreatedFrame   int    `json:"created_frame"`
	DestroyedFrame *int   `json:"destroyed_frame"`
}

// Runtime pointers are never exported. Aliases refer to the existing trace life.
type EconomyCashflowRuntime struct {
	NextFrame   int
	Controllers map[st.Entity]*EconomyCashflowController
	Live        map[int]st.Entity
	Fingerprint string
}

type EconomyCashflowDelivery struct {
	TraceSequence int                `json:"trace_sequence"`
	PawnHandle    EconomyEntryScalar `json:"pawn_handle"`
}

type EconomyCashflowPawn struct {
	PawnAlias       string             `json:"pawn_alias"`
	ControllerAlias *string            `json:"controller_alias"`
	PawnHandle      EconomyEntryScalar `json:"pawn_handle"`
	Account         EconomyEntryScalar `json:"account"`
	Spent           EconomyEntryScalar `json:"spent"`
}

type EconomyCashflowState struct {
	Sequence   int                   `json:"sequence"`
	FirstCut   int                   `json:"first_cut"`
	EntryState int                   `json:"entry_state_sequence"`
	Pawns      []EconomyCashflowPawn `json:"pawns"`
}

type EconomyCashflowCut struct {
	OwnershipCut   int `json:"ownership_cut"`
	StateSequence  int `json:"state_sequence"`
	TraceSequence  int `json:"trace_sequence"`
	TraceNextFrame int `json:"trace_next_frame"`
}

type EconomyCashflowObservations struct {
	Controllers []*EconomyCashflowController `json:"controllers"`
	Deliveries  []EconomyCashflowDelivery    `json:"deliveries"`
	Cuts        []EconomyCashflowCut         `json:"cuts"`
	States      []EconomyCashflowState       `json:"states"`
}
