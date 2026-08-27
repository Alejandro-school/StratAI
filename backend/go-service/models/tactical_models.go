package models

const (
	TacticalExportSchemaID      = "stratai.tactical_export@1"
	TacticalSamplingSchemaID    = "stratai.tactical_sampling@1"
	TacticalPhysicalSchemaID    = "stratai.tactical_physical_observation@1"
	TacticalOracleSchemaID      = "stratai.tactical_oracle_state@1"
	TacticalSamplingGapSchemaID = "stratai.tactical_sampling_gap@1"

	TacticalTargetHz = 16

	TacticalIdentityJoinOnly = "join_only"

	TacticalStatusObserved    = "observed"
	TacticalStatusUnavailable = "unavailable"

	TacticalCausalRoleModelInput = "model_input_observation"
	TacticalCausalRoleOracle     = "label_only_oracle"
	TacticalCausalRoleGap        = "coverage_gap"

	TacticalVisibilitySelf     = "self"
	TacticalVisibilityTeam     = "team"
	TacticalVisibilityEnemyLOS = "enemy_los"
	TacticalVisibilityOracle   = "oracle"
	TacticalVisibilityGap      = "sampling_gap"

	TacticalSourceReplayPlayerState = "replay_player_state"
	TacticalSourceReplaySampling    = "replay_sampling"

	TacticalSamplingStrategy = "round_anchored_right_closed_no_carry_forward"
)

// TacticalExport keeps causally available observations separate from oracle
// labels. Identity fields are join keys only and must not be used as features.
type TacticalExport struct {
	SchemaID          string                 `json:"schema_id"`
	IdentitySemantics string                 `json:"identity_semantics"`
	JoinKeys          TacticalExportJoinKeys `json:"join_keys"`
	Sampling          TacticalSampling       `json:"sampling"`
	PhysicalRows      []TacticalPhysicalRow  `json:"physical_rows"`
	OracleRows        []TacticalOracleRow    `json:"oracle_rows"`
	Gaps              []TacticalSamplingGap  `json:"gaps"`
}

type TacticalSampling struct {
	TargetHz    int     `json:"target_hz"`
	TickRate    float64 `json:"tick_rate"`
	PeriodTicks float64 `json:"period_ticks"`
	Strategy    string  `json:"strategy"`
}

type TacticalSamplingManifest struct {
	SchemaID          string                 `json:"schema_id"`
	MatchID           string                 `json:"match_id"`
	IdentitySemantics string                 `json:"identity_semantics"`
	JoinKeys          TacticalExportJoinKeys `json:"join_keys"`
	Sampling          TacticalSampling       `json:"sampling"`
	PhysicalRowCount  int                    `json:"physical_row_count"`
	OracleRowCount    int                    `json:"oracle_row_count"`
	GapCount          int                    `json:"gap_count"`
}

// TacticalExportJoinKeys keeps match identity out of sampling configuration
// and state payloads.
type TacticalExportJoinKeys struct {
	MatchID string `json:"match_id"`
}

// TacticalJoinKeys deliberately isolates identifiers from model features.
type TacticalJoinKeys struct {
	MatchID    string  `json:"match_id"`
	RoundID    string  `json:"round_id"`
	ObserverID *string `json:"observer_id"`
	SubjectID  *string `json:"subject_id"`
}

type TacticalVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// TacticalPhysicalState contains only state that can be treated as physical
// observation. Private oracle-only values such as health, armor and money do
// not exist in this type.
type TacticalPhysicalState struct {
	Team                 *string           `json:"team"`
	Position             *TacticalVector   `json:"position"`
	Velocity             *TacticalVector   `json:"velocity_world_units_per_second"`
	HorizontalVelocity   *float64          `json:"horizontal_velocity_world_units_per_second"`
	Yaw                  *float64          `json:"yaw"`
	Pitch                *float64          `json:"pitch"`
	Health               *int              `json:"health"`
	Armor                *int              `json:"armor"`
	Alive                *bool             `json:"alive"`
	ActiveWeapon         *string           `json:"active_weapon"`
	Grenades             []string          `json:"grenades"`
	HasC4                *bool             `json:"has_c4"`
	HasDefuseKit         *bool             `json:"has_defuse_kit"`
	AmmoInMagazine       *int              `json:"ammo_in_magazine"`
	AmmoReserve          *int              `json:"ammo_reserve"`
	IsDucking            *bool             `json:"is_ducking"`
	IsWalking            *bool             `json:"is_walking"`
	IsScoped             *bool             `json:"is_scoped"`
	IsReloading          *bool             `json:"is_reloading"`
	IsBlind              *bool             `json:"is_blind"`
	FlashDurationSeconds *float64          `json:"flash_duration_seconds"`
	Money                *int              `json:"money"`
	IsDefusing           *bool             `json:"is_defusing"`
	FieldAvailability    map[string]string `json:"field_availability"`
}

// TacticalOracleState is label-only and is never mixed into PhysicalRows.
type TacticalOracleState struct {
	Physical             TacticalPhysicalState `json:"physical"`
	Health               *int                  `json:"health"`
	Armor                *int                  `json:"armor"`
	Weapons              []string              `json:"weapons"`
	HasDefuseKit         *bool                 `json:"has_defuse_kit"`
	HasHelmet            *bool                 `json:"has_helmet"`
	HasC4                *bool                 `json:"has_c4"`
	FlashDurationSeconds *float64              `json:"flash_duration_seconds"`
	Money                *int                  `json:"money"`
}

type TacticalProvenance struct {
	SourceArtifact      string `json:"source_artifact"`
	SourceSchemaVersion *int   `json:"source_schema_version"`
	SourceRound         int    `json:"source_round"`
	SourceFrameTick     *int   `json:"source_frame_tick"`
	GeometryStatus      string `json:"geometry_status"`
	LineOfSight         *bool  `json:"line_of_sight"`
}

type TacticalPhysicalRow struct {
	SchemaID          string                `json:"schema_id"`
	MatchID           string                `json:"match_id"`
	IdentitySemantics string                `json:"identity_semantics"`
	JoinKeys          TacticalJoinKeys      `json:"join_keys"`
	RoundNumber       int                   `json:"round_number"`
	Tick              int                   `json:"tick"`
	AvailabilityTick  *int                  `json:"availability_tick"`
	Status            string                `json:"status"`
	CausalRole        string                `json:"causal_role"`
	VisibilityScope   string                `json:"visibility_scope"`
	Source            string                `json:"source"`
	Provenance        TacticalProvenance    `json:"provenance"`
	State             TacticalPhysicalState `json:"state"`
}

type TacticalOracleRow struct {
	SchemaID          string              `json:"schema_id"`
	MatchID           string              `json:"match_id"`
	IdentitySemantics string              `json:"identity_semantics"`
	JoinKeys          TacticalJoinKeys    `json:"join_keys"`
	RoundNumber       int                 `json:"round_number"`
	Tick              int                 `json:"tick"`
	AvailabilityTick  *int                `json:"availability_tick"`
	Status            string              `json:"status"`
	CausalRole        string              `json:"causal_role"`
	VisibilityScope   string              `json:"visibility_scope"`
	Source            string              `json:"source"`
	Provenance        TacticalProvenance  `json:"provenance"`
	State             TacticalOracleState `json:"state"`
}

type TacticalSamplingGap struct {
	SchemaID          string             `json:"schema_id"`
	MatchID           string             `json:"match_id"`
	IdentitySemantics string             `json:"identity_semantics"`
	JoinKeys          TacticalJoinKeys   `json:"join_keys"`
	RoundNumber       int                `json:"round_number"`
	Tick              int                `json:"tick"`
	AvailabilityTick  *int               `json:"availability_tick"`
	Status            string             `json:"status"`
	CausalRole        string             `json:"causal_role"`
	VisibilityScope   string             `json:"visibility_scope"`
	Source            string             `json:"source"`
	Reason            string             `json:"reason"`
	Provenance        TacticalProvenance `json:"provenance"`
}
