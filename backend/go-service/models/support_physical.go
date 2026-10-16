package models

// Facts are frozen in the support/replay callback, separate from the v1 source.
type SupportPhysicalFact struct {
	Status string  `json:"status"`
	Value  any     `json:"value"`
	Reason *string `json:"reason"`
}

type SupportPhysicalState struct {
	WeaponHandle         *uint64                        `json:"weapon_handle"`
	WeaponLife           *string                        `json:"weapon_life"`
	WeaponIdentityStatus string                         `json:"weapon_identity_status"`
	RawClip              *uint32                        `json:"raw_clip"`
	RawReserve           *int32                         `json:"raw_reserve"`
	ReserveSource        *string                        `json:"reserve_source"`
	ReloadSignal         *string                        `json:"reload_signal"`
	ReloadSignalTick     *int                           `json:"reload_signal_tick"`
	Facts                map[string]SupportPhysicalFact `json:"facts"`
}

type SupportPhysicalPlayer struct {
	NativeActor uint64                `json:"-"`
	ActorAlias  string                `json:"actor_alias"`
	Context     *SupportPhysicalState `json:"context"`
}

type SupportPhysicalFrame struct {
	SourceSequence int                     `json:"source_sequence"`
	Players        []SupportPhysicalPlayer `json:"players"`
}

type SupportPhysicalObservations struct {
	Frames []SupportPhysicalFrame `json:"frames"`
}
