package models

// EconomyFact is a value copied at observation time, never a reference to the
// mutable, round-end economy aggregates. Missing native properties remain null.
type EconomyFact struct {
	Value            any    `json:"value"`
	Status           string `json:"status"`
	Source           string `json:"source"`
	ObservationTick  *int   `json:"observation_tick"`
	AvailabilityTick *int   `json:"availability_tick"`
}

type EconomyInventoryItem struct {
	EntityAlias string `json:"entity_alias"`
	Weapon      string `json:"weapon"`
}

type EconomyActorSnapshot struct {
	ActorAlias        string                 `json:"actor_alias"`
	Side              string                 `json:"side"`
	Connected         bool                   `json:"connected"`
	Facts             map[string]EconomyFact `json:"facts"`
	PrepurchaseStatus string                 `json:"prepurchase_status"`
	PrepurchaseReason string                 `json:"prepurchase_reason"`
}

type EconomySnapshot struct {
	Phase            string                 `json:"phase"`
	Tick             int                    `json:"tick"`
	CallbackSequence int                    `json:"callback_sequence"`
	Context          map[string]EconomyFact `json:"context"`
	Actors           []EconomyActorSnapshot `json:"actors"`
}

type EconomyCallbackEvent struct {
	EventID          string  `json:"event_id"`
	Kind             string  `json:"kind"`
	Tick             int     `json:"tick"`
	CallbackSequence int     `json:"callback_sequence"`
	ActorAlias       string  `json:"actor_alias"`
	Side             string  `json:"side"`
	Weapon           *string `json:"weapon"`
	EntityAlias      *string `json:"entity_alias"`
	PriorDropEventID *string `json:"prior_drop_event_id"`
	Attribution      string  `json:"attribution"`
}

// EconomyObservedRound preserves round instances, including restarts; builders
// must not merge instances just because their scoreboard round number matches.
type EconomyObservedRound struct {
	Instance      int                    `json:"instance"`
	RoundNumber   int                    `json:"round_number"`
	StartTick     int                    `json:"start_tick"`
	FreezeEndTick *int                   `json:"freeze_end_tick"`
	EndTick       *int                   `json:"end_tick"`
	Snapshots     []EconomySnapshot      `json:"snapshots"`
	Events        []EconomyCallbackEvent `json:"events"`
}

type EconomyObservationLedger struct {
	Rounds           []EconomyObservedRound
	ActorAliases     map[uint64]string // Internal only; export uses match-scoped hashes.
	InitialFrontiers []EconomyInitialFrontier
}

// Probe facts are copied in RoundStart. CS2 delays item callbacks until the end
// of that native frame, so callback order alone never certifies prebuy state.
type EconomyInitialFrontier struct {
	Instance          int                    `json:"instance"`
	RoundStartProbe   *EconomySnapshot       `json:"round_start_probe"`
	NativeRules       map[string]EconomyFact `json:"native_rules"`
	OfficialEndTick   *int                   `json:"official_end_tick"`
	PreProbeCallbacks []EconomyEarlyCallback `json:"pre_probe_callbacks"`
}

type EconomyEarlyCallback struct {
	ActorAlias       string `json:"actor_alias"`
	Kind             string `json:"kind"`
	Tick             int    `json:"tick"`
	CallbackSequence int    `json:"callback_sequence"`
}
