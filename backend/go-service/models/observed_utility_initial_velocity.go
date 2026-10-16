package models

const (
	ObservedUtilityInitialVelocitySchemaID        = "stratai.observed_utility_initial_velocity@1"
	ObservedUtilityInitialVelocityContractVersion = "1.0.0"
	ObservedUtilityInitialVelocityCausalRole      = "launch_time_observation_only"

	ObservedUtilityInitialVelocityStatusObserved    = "observed"
	ObservedUtilityInitialVelocityStatusUnavailable = "unavailable"

	ObservedUtilityInitialVelocitySourceNative        = "demoinfocs_grenade_projectile_m_vecVelocity_at_throw"
	ObservedUtilityInitialVelocitySourcePositionDelta = "projectile_position_delta"
	ObservedUtilityInitialVelocitySourceUnavailable   = "unavailable"

	ObservedUtilityInitialVelocityUnits                     = "source2_world_units_per_second"
	ObservedUtilityInitialVelocityCoordinateFrame           = "source2_world_xyz"
	ObservedUtilityInitialVelocityWindowMethodNative        = "native_entity_property_at_projectile_throw"
	ObservedUtilityInitialVelocityWindowMethodPositionDelta = "two_position_delta"
	ObservedUtilityInitialVelocityWindowMethodUnavailable   = "unavailable"
	ObservedUtilityInitialVelocityWindowUnit                = "ingame_ticks"

	ObservedUtilityInitialVelocityReconciliationExact       = "exact"
	ObservedUtilityInitialVelocityReconciliationAmbiguous   = "ambiguous"
	ObservedUtilityInitialVelocityReconciliationUnavailable = "unavailable"
)

type ObservedUtilityInitialVelocityLedger struct {
	SchemaID          string                              `json:"schema_id"`
	ContractVersion   string                              `json:"contract_version"`
	IdentitySemantics string                              `json:"identity_semantics"`
	CausalRole        string                              `json:"causal_role"`
	Rows              []ObservedUtilityInitialVelocityRow `json:"rows"`
}

type ObservedUtilityInitialVelocityRow struct {
	RowID             string                                 `json:"row_id"`
	RoundNumber       int                                    `json:"round_number"`
	JoinKeys          ObservedUtilityInitialVelocityJoinKeys `json:"join_keys"`
	UtilityType       ObservedUtilityStringFact              `json:"utility_type"`
	Launch            ObservedUtilityLaunchFact              `json:"launch"`
	InitialVelocity   ObservedUtilityInitialVelocityFact     `json:"initial_velocity"`
	AbstentionReasons []string                               `json:"abstention_reasons"`
}

type ObservedUtilityInitialVelocityJoinKeys struct {
	ThrowID      string  `json:"throw_id"`
	ProjectileID *string `json:"projectile_id"`
	PlayerID     *string `json:"player_id"`
}

type ObservedUtilityStringFact struct {
	Status string  `json:"status"`
	Value  *string `json:"value"`
	Source string  `json:"source"`
}

type ObservedUtilityLaunchFact struct {
	Status string `json:"status"`
	Tick   *int   `json:"tick"`
	Source string `json:"source"`
}

type ObservedUtilityInitialVelocityVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ObservedUtilityInitialVelocityWindow struct {
	Method        string                                `json:"method"`
	StartTick     *int                                  `json:"start_tick"`
	EndTick       *int                                  `json:"end_tick"`
	TickCount     *int                                  `json:"tick_count"`
	TickRate      *float64                              `json:"tick_rate_hz"`
	StartPosition *ObservedUtilityInitialVelocityVector `json:"start_position"`
	EndPosition   *ObservedUtilityInitialVelocityVector `json:"end_position"`
	Unit          string                                `json:"unit"`
}

type ObservedUtilityInitialVelocityProvenance struct {
	ParserEvent    string `json:"parser_event"`
	EntityProperty string `json:"entity_property"`
	Reconciliation string `json:"reconciliation"`
}

type ObservedUtilityInitialVelocityFact struct {
	Status            string                                   `json:"status"`
	Vector            *ObservedUtilityInitialVelocityVector    `json:"vector"`
	HorizontalSpeed   *float64                                 `json:"horizontal_speed"`
	Units             string                                   `json:"units"`
	CoordinateFrame   string                                   `json:"coordinate_frame"`
	ObservedTick      *int                                     `json:"observed_tick"`
	AvailabilityTick  *int                                     `json:"availability_tick"`
	MeasurementWindow ObservedUtilityInitialVelocityWindow     `json:"measurement_window"`
	Source            string                                   `json:"source"`
	Provenance        ObservedUtilityInitialVelocityProvenance `json:"provenance"`
}
