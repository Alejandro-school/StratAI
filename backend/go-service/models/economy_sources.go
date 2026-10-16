package models

// EconomyNativeArray preserves observed array length and each native member.
// Empty is observed only when the native array itself is present.
type EconomyNativeArray struct {
	Status  string                   `json:"status"`
	Source  string                   `json:"source"`
	Reason  string                   `json:"reason"`
	Length  *int                     `json:"length"`
	Entries []map[string]EconomyFact `json:"entries"`
}

type EconomySourceSnapshot struct {
	ObservationID  string             `json:"observation_id"`
	Instance       int                `json:"instance"`
	Tick           int                `json:"tick"`
	ActorAlias     string             `json:"actor_alias"`
	Side           string             `json:"side"`
	Connected      bool               `json:"connected"`
	Money          EconomyFact        `json:"money"`
	SpentNative    EconomyFact        `json:"spent_native"`
	Sellback       EconomyNativeArray `json:"sellback"`
	PurchasesRound EconomyNativeArray `json:"purchases_round"`
	PurchasesMatch EconomyNativeArray `json:"purchases_match"`
}

type EconomyVersionObservation struct {
	Source string                 `json:"source"`
	Tick   int                    `json:"tick"`
	Fields map[string]EconomyFact `json:"fields"`
}

// This ledger is execution evidence, never a replacement for v2 initial state.
type EconomyNativeSources struct {
	VersionObservations []EconomyVersionObservation `json:"version_observations"`
	Snapshots           []EconomySourceSnapshot     `json:"snapshots"`
	GenericEventCounts  map[string]int              `json:"generic_event_counts"`
	SnapshotChecks      int                         `json:"snapshot_checks"`
}
