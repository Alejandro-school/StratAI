package models

// CanonicalLineageValue records either a factual value or why that value is not
// available. A missing lineage input must never be replaced by a guessed value.
type CanonicalLineageValue struct {
	Value  *string `json:"value"`
	Status string  `json:"status"`
	Source *string `json:"source"`
}

// CanonicalLineage is the deterministic, non-operational provenance carried by
// every canonical bundle. Processing/commit timestamps live only in the root
// transactional manifest so identical parses can remain byte-identical.
type CanonicalLineage struct {
	DemoChecksumSHA256     string                           `json:"demo_checksum_sha256"`
	ParserVersion          string                           `json:"parser_version"`
	DemoinfocsVersion      string                           `json:"demoinfocs_version"`
	ExportFormatVersion    string                           `json:"export_format_version"`
	BuildIdentifier        CanonicalLineageValue            `json:"build_identifier"`
	MapName                string                           `json:"map_name"`
	TickRateHz             float64                          `json:"tick_rate_hz"`
	TickRateRulesVersion   string                           `json:"tick_rate_rules_version"`
	PriceTableVersion      string                           `json:"price_table_version"`
	PriceTableSHA256       string                           `json:"price_table_sha256"`
	AlgorithmVersions      map[string]string                `json:"algorithm_versions"`
	SchemaVersions         map[string]string                `json:"schema_versions"`
	ValidatorVersion       string                           `json:"validator_version"`
	ProcessingTimestamp    CanonicalLineageValue            `json:"processing_timestamp"`
	MetadataSource         CanonicalSourceProvenance        `json:"metadata_source"`
	InputHashes            map[string]CanonicalLineageValue `json:"input_hashes"`
	ConfigurationHashes    map[string]string                `json:"configuration_hashes"`
	QualityFlags           []string                         `json:"quality_flags"`
	Warnings               []string                         `json:"warnings"`
	Abstentions            []string                         `json:"abstentions"`
	GoldenCorpusVersion    string                           `json:"golden_corpus_version"`
	GoldenCorpusManifestID string                           `json:"golden_corpus_manifest_id"`
}

// CanonicalDecision is the join-only contract that connects an actor, the
// state used at T0, the observed action and the separately stored outcome.
// Actor and state identifiers live here, never in the trainable feature row.
type CanonicalDecision struct {
	SchemaID                string   `json:"schema_id"`
	MatchID                 string   `json:"match_id"`
	DecisionID              string   `json:"decision_id"`
	RoundNumber             int      `json:"round_number"`
	ActorPlayerID           string   `json:"actor_player_id"`
	ActorIDUsage            string   `json:"actor_id_usage"`
	ObservedStateRef        *string  `json:"observed_state_ref"`
	StateAvailabilityStatus string   `json:"state_availability_status"`
	T0Tick                  int      `json:"t0_tick"`
	DecisionType            string   `json:"decision_type"`
	ActionTaken             string   `json:"action_taken"`
	AvailabilityTick        int      `json:"availability_tick"`
	AvailabilityStatus      string   `json:"availability_status"`
	CausalRole              string   `json:"causal_role"`
	VisibilityScope         string   `json:"visibility_scope"`
	Source                  string   `json:"source"`
	SourceRecordID          string   `json:"source_record_id"`
	SourceEventIDs          []string `json:"source_event_ids"`
	AlgorithmVersion        string   `json:"algorithm_version"`
}

// CanonicalDecisionFeatures contains only join keys and trainable facts
// available no later than T0Tick. Labels, results, names, ratings, scores and
// player IDs are intentionally absent from this physical partition.
type CanonicalDecisionFeatures struct {
	SchemaID                  string   `json:"schema_id"`
	MatchID                   string   `json:"match_id"`
	DecisionID                string   `json:"decision_id"`
	DecisionType              string   `json:"decision_type"`
	RoundNumber               int      `json:"round_number"`
	T0Tick                    int      `json:"t0_tick"`
	AvailabilityTickMax       int      `json:"availability_tick_max"`
	ParticipantCount          int      `json:"participant_count"`
	ObservedParticipantStates int      `json:"observed_participant_states"`
	AliveParticipantCount     *int     `json:"alive_participant_count"`
	InitialDistanceWorldUnits *float64 `json:"initial_distance_world_units"`
	InitialDistanceStatus     string   `json:"initial_distance_status"`
	BombContextStatus         string   `json:"bomb_context_status"`
	EconomyContextStatus      string   `json:"economy_context_status"`
	EnemiesExposedCount       *int     `json:"enemies_exposed_count"`
	EnemiesExposedStatus      string   `json:"enemies_exposed_status"`
	RoundClockRemainingMS     *int64   `json:"round_clock_remaining_ms"`
	BombTimeRemainingMS       *int64   `json:"bomb_time_remaining_ms"`
	SourceStateCount          int      `json:"source_state_count"`
	TradePossible             *bool    `json:"trade_possible"`
	TradePossibleStatus       string   `json:"trade_possible_status"`
	NearestTeammateDistance   *float64 `json:"nearest_teammate_distance_world_units"`
	NearestDistanceStatus     string   `json:"nearest_teammate_distance_status"`
	NearestConnectionTimeMS   *float64 `json:"nearest_connection_time_ms"`
	NearestConnectionStatus   string   `json:"nearest_connection_time_status"`
	AnyLineOfSight            *bool    `json:"any_line_of_sight"`
	LineOfSightStatus         string   `json:"line_of_sight_status"`
	MinimumFacingDeltaDeg     *float64 `json:"minimum_facing_delta_deg"`
	FacingStatus              string   `json:"facing_status"`
}

// CanonicalOracleContext is deliberately separate from decision features. The
// current parser does not publish a trustworthy hidden-state oracle, so rows
// abstain explicitly instead of copying post-T0 information.
type CanonicalOracleContext struct {
	SchemaID    string   `json:"schema_id"`
	MatchID     string   `json:"match_id"`
	DecisionID  string   `json:"decision_id"`
	RoundNumber int      `json:"round_number"`
	T0Tick      int      `json:"t0_tick"`
	Status      string   `json:"status"`
	Available   bool     `json:"available"`
	FieldNames  []string `json:"field_names"`
	Abstentions []string `json:"abstentions"`
}

// CanonicalDecisionOutcome contains information that is forbidden from the
// decision-feature partition and is only available at/after the outcome.
type CanonicalDecisionOutcome struct {
	SchemaID             string                    `json:"schema_id"`
	MatchID              string                    `json:"match_id"`
	DecisionID           string                    `json:"decision_id"`
	RoundNumber          int                       `json:"round_number"`
	T0Tick               int                       `json:"t0_tick"`
	Outcome              string                    `json:"outcome"`
	OutcomeTick          int                       `json:"outcome_tick"`
	DurationMS           float64                   `json:"duration_ms"`
	WinnerObserved       bool                      `json:"winner_observed"`
	LoserCount           int                       `json:"loser_count"`
	TerminalKillCount    int                       `json:"terminal_kill_count"`
	TradeCandidateCount  int                       `json:"trade_candidate_count"`
	TradeCompletionCount int                       `json:"trade_completion_count"`
	SurvivalStatus       string                    `json:"survival_status"`
	DisengagementStatus  string                    `json:"disengagement_status"`
	Horizons             []CanonicalHorizonOutcome `json:"horizons"`
}

// CanonicalHorizonOutcome is a retrospective label. It is physically stored
// only in the outcome partition and is forbidden from decision features.
type CanonicalHorizonOutcome struct {
	HorizonSeconds int    `json:"horizon_seconds"`
	Status         string `json:"status"`
	Outcome        string `json:"outcome"`
	Source         string `json:"source"`
}

// CanonicalQualityMask keeps availability and inference separate from both
// features and outcomes. Consumers must apply the mask rather than treating a
// missing observation as a factual zero.
type CanonicalQualityMask struct {
	SchemaID          string   `json:"schema_id"`
	MatchID           string   `json:"match_id"`
	DecisionID        string   `json:"decision_id"`
	RoundNumber       int      `json:"round_number"`
	T0Tick            int      `json:"t0_tick"`
	AvailableFields   []string `json:"available_fields"`
	UnavailableFields []string `json:"unavailable_fields"`
	InferredFields    []string `json:"inferred_fields"`
	WarningFlags      []string `json:"warning_flags"`
}
