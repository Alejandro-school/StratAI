package models

type EconomyOwnershipPawn struct {
	EntityAlias string                   `json:"entity_alias"`
	ActorAlias  *string                  `json:"actor_alias"`
	Side        *string                  `json:"side"`
	Status      string                   `json:"inventory_status"`
	Length      *int                     `json:"inventory_length"`
	Slots       []EconomyEntityReference `json:"slots"`
}

type EconomyOwnershipEquipment struct {
	EntityAlias string                 `json:"entity_alias"`
	Owner       EconomyEntityReference `json:"owner"`
}

type EconomyOwnershipState struct {
	Sequence  int                         `json:"sequence"`
	FirstCut  int                         `json:"first_cut"`
	Pawns     []EconomyOwnershipPawn      `json:"pawns"`
	Equipment []EconomyOwnershipEquipment `json:"equipment"`
}

type EconomyOwnershipCut struct {
	Sequence         int  `json:"sequence"`
	Frame            int  `json:"frame"`
	DemoTick         int  `json:"demo_tick"`
	Instance         int  `json:"instance"`
	IdentitySequence int  `json:"identity_sequence"`
	ClockSequence    *int `json:"clock_sequence"`
	StateSequence    int  `json:"state_sequence"`
}

type EconomyOwnershipObservations struct {
	Cuts   []EconomyOwnershipCut   `json:"cuts"`
	States []EconomyOwnershipState `json:"states"`
}
