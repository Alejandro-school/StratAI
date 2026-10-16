package models

const (
	ObservedLastKnownPositionSchemaID        = "stratai.observed_last_known_position@1"
	ObservedLastKnownPositionContractVersion = "1.0.0"
	ObservedLastKnownPositionCausalRole      = "model_input_observation"
	ObservedLastKnownPositionIdentity        = "local_opaque_aliases"

	ObservedLastKnownPositionStatusObserved    = "observed"
	ObservedLastKnownPositionStatusUnavailable = "unavailable"

	ObservedLastKnownPositionSourceContact     = "confirmed_visual_contact_subject_pawn_origin"
	ObservedLastKnownPositionSourceAge         = "tick_delta_from_confirmed_visual_contact"
	ObservedLastKnownPositionSourceUnavailable = "unavailable"
	ObservedLastKnownPositionCoordinateFrame   = "source2_world_xyz"
	ObservedLastKnownPositionSpatialSemantics  = "subject_pawn_origin_at_confirmed_visual_contact"
	ObservedInformationAgeUnits                = "milliseconds"

	ObservedLastKnownPositionDerivationUpdate      = "contact_update"
	ObservedLastKnownPositionDerivationCarry       = "carry_from_prior_contact"
	ObservedLastKnownPositionDerivationUnavailable = "unavailable"

	ObservedLastKnownPositionContactSource = ObservedVisualContactSchemaID
)

// ObservedLastKnownPositionLedger is a causal projection of the visual-contact
// ledger. It is a candidate contract and is intentionally outside canonical
// export format 3.8.0.
type ObservedLastKnownPositionLedger struct {
	SchemaID          string                         `json:"schema_id"`
	ContractVersion   string                         `json:"contract_version"`
	IdentitySemantics string                         `json:"identity_semantics"`
	CausalRole        string                         `json:"causal_role"`
	TickRateHz        float64                        `json:"tick_rate_hz"`
	Rows              []ObservedLastKnownPositionRow `json:"rows"`
}

type ObservedLastKnownPositionJoinKeys struct {
	SourceAlias   string `json:"source_alias"`
	RoundID       string `json:"round_id"`
	ObserverAlias string `json:"observer_alias"`
	SubjectAlias  string `json:"subject_alias"`
}

type ObservedLastKnownPositionVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// ObservedLastKnownPositionContactObservation retains only what is needed to
// reproduce the derivation. SubjectPosition is populated exclusively when the
// upstream visual contact is observed true; false and unavailable rows never
// retain the subject's current hidden position.
type ObservedLastKnownPositionContactObservation struct {
	Status                   string                           `json:"status"`
	VisualContact            *bool                            `json:"visual_contact"`
	AvailabilityTick         *int                             `json:"availability_tick"`
	SourceFrameTick          *int                             `json:"source_frame_tick"`
	SubjectPositionAtContact *ObservedLastKnownPositionVector `json:"subject_position_at_contact"`
	ObserverLifeSequence     *int                             `json:"observer_life_sequence"`
	SubjectLifeSequence      *int                             `json:"subject_life_sequence"`
	Source                   string                           `json:"source"`
}

type ObservedLastKnownPositionFact struct {
	Status           string                           `json:"status"`
	Value            *ObservedLastKnownPositionVector `json:"value"`
	ObservedTick     *int                             `json:"observed_tick"`
	AvailabilityTick *int                             `json:"availability_tick"`
	CoordinateFrame  string                           `json:"coordinate_frame"`
	SpatialSemantics string                           `json:"spatial_semantics"`
	Source           string                           `json:"source"`
}

type ObservedInformationAgeFact struct {
	Status           string   `json:"status"`
	ValueMS          *float64 `json:"value_ms"`
	AvailabilityTick *int     `json:"availability_tick"`
	Units            string   `json:"units"`
	Source           string   `json:"source"`
}

type ObservedLastKnownPositionProvenance struct {
	VisualContactSchemaID        string   `json:"visual_contact_schema_id"`
	VisualContactContractVersion string   `json:"visual_contact_contract_version"`
	PositionSourceArtifact       string   `json:"position_source_artifact"`
	InputFields                  []string `json:"input_fields"`
}

type ObservedLastKnownPositionRow struct {
	RowID              string                                      `json:"row_id"`
	RoundNumber        int                                         `json:"round_number"`
	T0Tick             int                                         `json:"t0_tick"`
	JoinKeys           ObservedLastKnownPositionJoinKeys           `json:"join_keys"`
	ContactObservation ObservedLastKnownPositionContactObservation `json:"contact_observation"`
	LastKnownPosition  ObservedLastKnownPositionFact               `json:"last_known_position"`
	InformationAge     ObservedInformationAgeFact                  `json:"information_age"`
	Derivation         string                                      `json:"derivation"`
	AbstentionReasons  []string                                    `json:"abstention_reasons"`
	Provenance         ObservedLastKnownPositionProvenance         `json:"provenance"`
}
