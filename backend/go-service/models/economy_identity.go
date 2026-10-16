package models

type EconomyIdentityClock struct {
	Sequence    int     `json:"sequence"`
	DemoTick    int     `json:"demo_tick"`
	Frame       int     `json:"frame"`
	NetworkTick *uint32 `json:"network_tick"`
	Epoch       int     `json:"epoch"`
	EpochReason string  `json:"epoch_reason"`
}

type EconomyIdentityRow struct {
	Sequence      int     `json:"sequence"`
	DemoTick      int     `json:"demo_tick"`
	Frame         int     `json:"frame"`
	ClockSequence *int    `json:"clock_sequence"`
	Kind          string  `json:"kind"`
	Source        string  `json:"source"`
	Instance      *int    `json:"instance"`
	ActorAlias    *string `json:"actor_alias"`
	Side          *string `json:"side"`
	EntityAlias   *string `json:"entity_alias"`
	Value         any     `json:"value"`
}

// Handles are ephemeral demo-local references, never account/item IDs.
type EconomyEntityReference struct {
	Handle      *uint64 `json:"handle"`
	Status      string  `json:"status"`
	TargetAlias *string `json:"target_alias"`
}

type EconomyNativeIdentity struct {
	Clocks            []EconomyIdentityClock `json:"clocks"`
	Rows              []EconomyIdentityRow   `json:"rows"`
	UserMessageCounts map[string]int         `json:"user_message_counts"`
}
