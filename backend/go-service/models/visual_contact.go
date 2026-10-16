package models

const (
	ObservedVisualContactSchemaID        = "stratai.observed_visual_contact@1"
	ObservedVisualContactContractVersion = "1.0.0"

	VisualContactCausalRole      = "model_input_observation"
	VisualContactVisibilityScope = "enemy_visual_contact_evaluation"
	VisualContactSource          = "replay_visual_contact_derivation"

	VisualContactFactDerived     = "derived"
	VisualContactFactUnavailable = "unavailable"

	VisualContactBoundaryStarted     = "started"
	VisualContactBoundaryContinued   = "continued"
	VisualContactBoundaryEnded       = "ended"
	VisualContactBoundaryInactive    = "inactive"
	VisualContactBoundaryUnavailable = "unavailable"

	VisualContactEpisodeOpen   = "open_at_t0"
	VisualContactEpisodeClosed = "closed"
)

// ObservedVisualContactLedger is a causal, factual projection capped at T0.
// It is a new contract and is intentionally not written into canonical 3.8.0.
type ObservedVisualContactLedger struct {
	SchemaID          string                            `json:"schema_id"`
	ContractVersion   string                            `json:"contract_version"`
	IdentitySemantics string                            `json:"identity_semantics"`
	CausalRole        string                            `json:"causal_role"`
	JoinKeys          TacticalExportJoinKeys            `json:"join_keys"`
	T0Tick            int                               `json:"t0_tick"`
	Sampling          TacticalSampling                  `json:"sampling"`
	Evaluations       []ObservedVisualContactEvaluation `json:"evaluations"`
	Contacts          []ObservedVisualContactEpisode    `json:"contacts"`
}

// ObservedVisualContactJoinKeys isolate every identifier from factual values.
type ObservedVisualContactJoinKeys struct {
	MatchID    string  `json:"match_id"`
	RoundID    string  `json:"round_id"`
	ObserverID string  `json:"observer_id"`
	SubjectID  string  `json:"subject_id"`
	ContactID  *string `json:"contact_id"`
}

type ObservedVisualContactFact struct {
	Status string `json:"status"`
	Value  *bool  `json:"value"`
}

type ObservedVisualContactFOVFact struct {
	Status                 string   `json:"status"`
	Inside                 *bool    `json:"inside"`
	ReferenceHorizontalFOV *float64 `json:"reference_horizontal_fov_deg"`
}

type ObservedVisualContactSmokeFact struct {
	Status           string `json:"status"`
	Occluded         *bool  `json:"occluded"`
	ActiveSmokeCount *int   `json:"active_smoke_count"`
}

type ObservedVisualContactProvenance struct {
	SourceArtifact      string   `json:"source_artifact"`
	SourceSchemaVersion *int     `json:"source_schema_version"`
	SourceRound         int      `json:"source_round"`
	SourceFrameTick     int      `json:"source_frame_tick"`
	GeometryStatus      string   `json:"geometry_status"`
	EyePositionSource   string   `json:"eye_position_source"`
	RaycastMethod       string   `json:"raycast_method"`
	FOVMethod           string   `json:"fov_method"`
	SmokeMethod         string   `json:"smoke_method"`
	InputFields         []string `json:"input_fields"`
}

type ObservedVisualContactEvaluation struct {
	SchemaID          string                          `json:"schema_id"`
	ContractVersion   string                          `json:"contract_version"`
	IdentitySemantics string                          `json:"identity_semantics"`
	JoinKeys          ObservedVisualContactJoinKeys   `json:"join_keys"`
	RoundNumber       int                             `json:"round_number"`
	Tick              int                             `json:"tick"`
	T0Tick            int                             `json:"t0_tick"`
	AvailabilityTick  *int                            `json:"availability_tick"`
	Status            string                          `json:"status"`
	CausalRole        string                          `json:"causal_role"`
	VisibilityScope   string                          `json:"visibility_scope"`
	Source            string                          `json:"source"`
	LineOfSight       ObservedVisualContactFact       `json:"line_of_sight"`
	FieldOfView       ObservedVisualContactFOVFact    `json:"field_of_view"`
	SmokeOcclusion    ObservedVisualContactSmokeFact  `json:"smoke_occlusion"`
	VisualContact     ObservedVisualContactFact       `json:"visual_contact"`
	Boundary          string                          `json:"boundary"`
	AbstentionReasons []string                        `json:"abstention_reasons"`
	Provenance        ObservedVisualContactProvenance `json:"provenance"`
}

type ObservedVisualContactEpisode struct {
	SchemaID              string                        `json:"schema_id"`
	ContractVersion       string                        `json:"contract_version"`
	IdentitySemantics     string                        `json:"identity_semantics"`
	JoinKeys              ObservedVisualContactJoinKeys `json:"join_keys"`
	RoundNumber           int                           `json:"round_number"`
	T0Tick                int                           `json:"t0_tick"`
	StartTick             int                           `json:"start_tick"`
	EndTick               *int                          `json:"end_tick"`
	LastConfirmedTick     int                           `json:"last_confirmed_tick"`
	StartAvailabilityTick int                           `json:"start_availability_tick"`
	EndAvailabilityTick   *int                          `json:"end_availability_tick"`
	Status                string                        `json:"status"`
	CausalRole            string                        `json:"causal_role"`
}
