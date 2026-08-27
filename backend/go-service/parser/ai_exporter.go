package parser

import (
	"crypto/sha256"
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ExportMatchBundle writes the versioned canonical match bundle.
func ExportMatchBundle(ctx *models.DemoContext, matchID, outputDir string, matchDate ...string) error {
	playedAt := ""
	if len(matchDate) > 0 {
		playedAt = matchDate[0]
	}
	return exportMatchBundle(ctx, matchID, outputDir, playedAt, models.CanonicalExportProvenance{Source: "demo", DemoChecksum: strings.Repeat("0", 64)})
}

func ExportMatchBundleWithProvenance(
	ctx *models.DemoContext,
	matchID, outputDir, playedAt string,
	provenance models.CanonicalExportProvenance,
) error {
	return exportMatchBundle(ctx, matchID, outputDir, playedAt, provenance)
}

func exportMatchBundle(
	ctx *models.DemoContext,
	matchID, outputDir, playedAt string,
	provenance models.CanonicalExportProvenance,
) error {
	matchDir := filepath.Join(outputDir, fmt.Sprintf("match_%s", matchID))
	if err := os.MkdirAll(matchDir, 0o750); err != nil {
		return fmt.Errorf("failed to create match directory: %w", err)
	}

	tickRate := ctx.Parser.TickRate()
	durationSeconds := ctx.Parser.CurrentTime().Seconds()
	if ctx.ReplayData != nil {
		ctx.ReplayData.Metadata.MatchID = matchID
	}

	quality := buildQualityReport(ctx)
	if quality.hasHardObjectiveFailure() {
		return fmt.Errorf(
			"objective data failed pre-export validation: contract=%d round=%d terminal=%d lifecycle=%d replay_projection=%d tracking_carrier=%d replay_carrier=%d; %s",
			quality.ObjectiveContractViolations,
			quality.ObjectiveRoundMismatches,
			quality.ObjectiveTerminalMismatches,
			quality.ObjectiveLifecycleViolations,
			quality.ObjectiveReplayEventMismatches,
			quality.ObjectiveTrackingCarrierErrors,
			quality.ObjectiveReplayCarrierErrors,
			strings.Join(quality.objectiveFailureDetails, "; "),
		)
	}
	if quality.hasHardUtilityFailure() {
		return fmt.Errorf(
			"utility data failed pre-export validation: contract=%d reconciliation=%d player_stats=%d replay_projection=%d lifecycle=%d flash=%d damage=%d temporal_spatial=%d determinism=%d; %s",
			quality.UtilityContractViolations,
			quality.UtilityThrowReconciliationMismatches,
			quality.UtilityPlayerStatsMismatches,
			quality.UtilityReplayProjectionMismatches,
			quality.UtilityLifecycleViolations,
			quality.UtilityFlashAttributionMismatches,
			quality.UtilityDamageReconciliationMismatches,
			quality.UtilityTemporalSpatialViolations,
			quality.UtilityDeterminismViolations,
			strings.Join(quality.utilityFailureDetails, "; "),
		)
	}
	if quality.hasHardCombatFailure() {
		return fmt.Errorf(
			"combat data failed pre-export validation: contract=%d callbacks=%d player_stats=%d replay_projection=%d native_delta=%d determinism=%d; %s",
			quality.CombatContractViolations,
			quality.CombatCallbackAccountingViolations,
			quality.CombatPlayerStatsMismatches,
			quality.CombatReplayProjectionMismatches,
			quality.CombatNativeDeltaMismatches,
			quality.CombatDeterminismViolations,
			strings.Join(quality.combatFailureDetails, "; "),
		)
	}
	if err := ExportCanonicalBundle(ctx, matchID, matchDir, playedAt, durationSeconds, tickRate, quality, provenance); err != nil {
		return err
	}
	canonicalRounds, err := readCanonicalRounds(filepath.Join(matchDir, "canonical", "core", "rounds.json"))
	if err != nil {
		return err
	}
	combatLedger := filterCanonicalCombatLedgerToRounds(ctx.Combat.Snapshot(), canonicalRounds)
	combatEventIDs, combatShotIDs := canonicalCombatReferenceMaps(combatLedger, matchID)
	canonicalReplay := filterCanonicalReplayToRounds(ctx.ReplayData, canonicalRounds)
	if err := exportCanonicalReplayPresentation(matchDir, matchID, canonicalReplay, combatEventIDs, combatShotIDs); err != nil {
		return err
	}
	return finalizeCanonicalBlock7Quality(matchDir)
}

func readCanonicalRounds(path string) ([]models.CanonicalRound, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read canonical rounds: %w", err)
	}
	var export models.CanonicalRoundsExport
	if err := json.Unmarshal(payload, &export); err != nil {
		return nil, fmt.Errorf("decode canonical rounds: %w", err)
	}
	return export.Rounds, nil
}

type qualityReport struct {
	SchemaVersion                         int                                          `json:"schema_version"`
	DemoinfocsVersion                     string                                       `json:"demoinfocs_version"`
	RatingModel                           string                                       `json:"rating_model"`
	IdentifierEncoding                    string                                       `json:"identifier_encoding"`
	Status                                string                                       `json:"status"`
	UsableForTraining                     bool                                         `json:"usable_for_training"`
	StatusMeaning                         string                                       `json:"status_meaning"`
	ParseCompleted                        bool                                         `json:"parse_completed"`
	ParserWarnings                        []string                                     `json:"parser_warnings"`
	ExpectedRounds                        int                                          `json:"expected_rounds"`
	ReplayRounds                          int                                          `json:"replay_rounds"`
	EconomyRounds                         int                                          `json:"economy_rounds"`
	TrackingEvents                        int                                          `json:"tracking_events"`
	CombatDuels                           int                                          `json:"combat_duels"`
	GrenadeEvents                         int                                          `json:"grenade_events"`
	PlayerSummaries                       int                                          `json:"player_summaries"`
	ReplayRoundSequenceErrors             int                                          `json:"replay_round_sequence_errors"`
	EconomyRoundSequenceErrors            int                                          `json:"economy_round_sequence_errors"`
	TrackingRoundSequenceErrors           int                                          `json:"tracking_round_sequence_errors"`
	CombatRoundSequenceErrors             int                                          `json:"combat_round_sequence_errors"`
	GrenadeRoundSequenceErrors            int                                          `json:"grenade_round_sequence_errors"`
	EconomyReconciliationMismatches       int                                          `json:"economy_reconciliation_mismatches"`
	EconomyStartMismatches                int                                          `json:"economy_start_reconciliation_mismatches"`
	EconomyEndMismatches                  int                                          `json:"economy_end_reconciliation_mismatches"`
	EconomyEndNativeMismatches            int                                          `json:"economy_end_native_reconciliation_mismatches"`
	PurchaseItemizationMismatches         int                                          `json:"purchase_itemization_mismatches"`
	NativeScoreboardMismatches            int                                          `json:"native_scoreboard_mismatches"`
	AccuracyRangeMismatches               int                                          `json:"accuracy_range_mismatches"`
	CrosshairVectorMismatches             int                                          `json:"crosshair_vector_mismatches"`
	ReplayFramesOutsideBounds             int                                          `json:"replay_frames_outside_bounds"`
	RaytracingAvailable                   bool                                         `json:"raytracing_available"`
	VisibilitySampledTicks                int                                          `json:"visibility_sampled_ticks"`
	VisibilitySkippedTicks                int                                          `json:"visibility_skipped_ticks"`
	VisibilityRaycasts                    int                                          `json:"visibility_raycasts"`
	VisibilityRefinementRaycasts          int                                          `json:"visibility_refinement_raycasts"`
	VisibilityModel                       string                                       `json:"visibility_model"`
	VisibilityReferenceAspect             string                                       `json:"visibility_reference_aspect"`
	VisibilityReferenceFOV                float64                                      `json:"visibility_reference_fov"`
	VisibilityScopedFOVs                  map[string]float64                           `json:"visibility_scoped_reference_fovs"`
	VisibilitySampleStride                int                                          `json:"visibility_sample_stride"`
	VisibilityDistanceLimit               string                                       `json:"visibility_distance_limit"`
	NativeEyePositions                    int                                          `json:"native_eye_positions"`
	FallbackEyePositions                  int                                          `json:"fallback_eye_positions"`
	TTDSamples                            int                                          `json:"ttd_samples"`
	CrosshairPlacementSamples             int                                          `json:"crosshair_placement_samples"`
	BulletDamageEvents                    int                                          `json:"bullet_damage_events"`
	BulletDamageCorrelated                int                                          `json:"bullet_damage_correlated"`
	TrackingAliveStates                   int                                          `json:"tracking_alive_states"`
	TrackingVelocityObserved              int                                          `json:"tracking_velocity_observed"`
	TrackingPositiveVelocityStates        int                                          `json:"tracking_positive_velocity_states"`
	TrackingVelocityCoverageRatio         float64                                      `json:"tracking_velocity_coverage_ratio"`
	TrackingActiveWeaponObserved          int                                          `json:"tracking_active_weapon_observed"`
	TrackingActiveWeaponCoverage          float64                                      `json:"tracking_active_weapon_coverage_ratio"`
	TrackingStateContractViolations       int                                          `json:"tracking_state_contract_violations"`
	TrackingMovingDatasetWithoutData      int                                          `json:"tracking_moving_dataset_without_velocity"`
	ObjectiveEvents                       int                                          `json:"objective_events"`
	ObjectiveCarrierSnapshots             int                                          `json:"objective_carrier_snapshots"`
	ObjectiveBombDrops                    int                                          `json:"objective_bomb_drops"`
	ObjectiveBombPickups                  int                                          `json:"objective_bomb_pickups"`
	ObjectivePlantStarts                  int                                          `json:"objective_plant_starts"`
	ObjectivePlantAborts                  int                                          `json:"objective_plant_aborts"`
	ObjectivePlants                       int                                          `json:"objective_plants"`
	ObjectiveDefuseStarts                 int                                          `json:"objective_defuse_starts"`
	ObjectiveDefuseAborts                 int                                          `json:"objective_defuse_aborts"`
	ObjectiveDefuses                      int                                          `json:"objective_defuses"`
	ObjectiveBombExplosions               int                                          `json:"objective_bomb_explosions"`
	ObjectiveMissingActors                int                                          `json:"objective_missing_actors"`
	ObjectiveMissingPositions             int                                          `json:"objective_missing_positions"`
	ObjectiveAttempts                     int                                          `json:"objective_attempts"`
	ObjectiveAttemptsMissingStart         int                                          `json:"objective_attempts_missing_start"`
	ObjectiveAttemptsUnclosed             int                                          `json:"objective_attempts_unclosed"`
	ObjectiveContractViolations           int                                          `json:"objective_contract_violations"`
	ObjectiveRoundMismatches              int                                          `json:"objective_round_mismatches"`
	ObjectiveTerminalMismatches           int                                          `json:"objective_terminal_mismatches"`
	ObjectiveLifecycleViolations          int                                          `json:"objective_lifecycle_violations"`
	ObjectiveTrackingCarrierSamples       int                                          `json:"objective_tracking_carrier_samples"`
	ObjectiveReplayCarrierSamples         int                                          `json:"objective_replay_carrier_samples"`
	ObjectiveReplayBombStateMissing       int                                          `json:"objective_replay_bomb_state_missing"`
	ObjectiveTrackingCarrierErrors        int                                          `json:"objective_tracking_carrier_mismatches"`
	ObjectiveReplayCarrierErrors          int                                          `json:"objective_replay_carrier_mismatches"`
	ObjectiveCarrierMismatches            int                                          `json:"objective_carrier_mismatches"`
	ObjectiveReplayEventMismatches        int                                          `json:"objective_replay_event_mismatches"`
	ObjectiveNativeRoleDisagreements      int                                          `json:"objective_native_role_disagreements"`
	UtilityThrows                         int                                          `json:"utility_throws"`
	UtilityCanonicalEvents                int                                          `json:"utility_canonical_events"`
	UtilityFlashEffects                   int                                          `json:"utility_flash_effects"`
	UtilityDamageEffects                  int                                          `json:"utility_damage_effects"`
	UtilityContractViolations             int                                          `json:"utility_contract_violations"`
	UtilityThrowReconciliationMismatches  int                                          `json:"utility_throw_reconciliation_mismatches"`
	UtilityPlayerStatsMismatches          int                                          `json:"utility_player_stats_mismatches"`
	UtilityReplayProjectionMismatches     int                                          `json:"utility_replay_projection_mismatches"`
	UtilityCallbackAccountingViolations   int                                          `json:"utility_callback_accounting_violations"`
	UtilityThrowCallbacks                 int                                          `json:"utility_throw_callbacks"`
	UtilityBounceCallbacks                int                                          `json:"utility_bounce_callbacks"`
	UtilityLifecycleCallbacks             int                                          `json:"utility_lifecycle_callbacks"`
	UtilityPlayerFlashedCallbacks         int                                          `json:"utility_player_flashed_callbacks"`
	UtilityDamageCallbacks                int                                          `json:"utility_damage_callbacks"`
	UtilityCallbackDiagnostics            map[string]utilityCallbackQualityDiagnostics `json:"utility_callback_diagnostics"`
	UtilityUnmatchedCallbacks             int                                          `json:"utility_unmatched_callbacks"`
	UtilityOrphanCallbacks                int                                          `json:"utility_orphan_callbacks"`
	UtilityInferredCallbacks              int                                          `json:"utility_inferred_callbacks"`
	UtilityDeduplicatedCallbacks          int                                          `json:"utility_deduplicated_callbacks"`
	UtilityLifecycleViolations            int                                          `json:"utility_lifecycle_violations"`
	UtilityFlashAttributionMismatches     int                                          `json:"utility_flash_attribution_mismatches"`
	UtilityDamageReconciliationMismatches int                                          `json:"utility_damage_reconciliation_mismatches"`
	UtilityTemporalSpatialViolations      int                                          `json:"utility_temporal_spatial_violations"`
	UtilityDeterminismViolations          int                                          `json:"utility_determinism_violations"`
	UtilityObservationWarnings            int                                          `json:"utility_observation_warnings"`
	UtilityMissingTypeObservations        int                                          `json:"utility_missing_type_observations"`
	UtilityMissingActorObservations       int                                          `json:"utility_missing_actor_observations"`
	UtilityMissingAffectedPlayers         int                                          `json:"utility_missing_affected_player_observations"`
	UtilityMissingFlashDurations          int                                          `json:"utility_missing_flash_duration_observations"`
	UtilityMissingLaunchTicks             int                                          `json:"utility_missing_launch_tick_observations"`
	UtilityMissingLaunchPositions         int                                          `json:"utility_missing_launch_position_observations"`
	UtilityMissingLaunchViews             int                                          `json:"utility_missing_launch_view_observations"`
	UtilityMissingThrowerVelocities       int                                          `json:"utility_missing_thrower_velocity_observations"`
	UtilityMissingProjectileVelocities    int                                          `json:"utility_missing_projectile_velocity_observations"`
	UtilityMissingTrajectoryObservations  int                                          `json:"utility_missing_trajectory_observations"`
	UtilityMissingLifecycleObservations   int                                          `json:"utility_missing_lifecycle_observations"`
	UtilityInferredCorrelations           int                                          `json:"utility_inferred_correlations"`
	UtilityObservedEffectCorrelations     int                                          `json:"utility_observed_effect_correlations"`
	UtilityInferredEffectCorrelations     int                                          `json:"utility_inferred_effect_correlations"`
	UtilityUnavailableEffectCorrelations  int                                          `json:"utility_unavailable_effect_correlations"`
	CombatLedgerEvents                    int                                          `json:"combat_ledger_events"`
	CombatContractViolations              int                                          `json:"combat_contract_violations"`
	CombatCallbackAccountingViolations    int                                          `json:"combat_callback_accounting_violations"`
	CombatPlayerStatsMismatches           int                                          `json:"combat_player_stats_mismatches"`
	CombatReplayProjectionMismatches      int                                          `json:"combat_replay_projection_mismatches"`
	CombatNativeDeltaMismatches           int                                          `json:"combat_native_delta_mismatches"`
	CombatDeterminismViolations           int                                          `json:"combat_determinism_violations"`
	CombatMissingImpactPositions          int                                          `json:"combat_missing_impact_positions"`
	CombatMissingReloadEnds               int                                          `json:"combat_missing_reload_ends"`
	CombatUnavailableShotResults          int                                          `json:"combat_unavailable_shot_results"`
	CombatDiscardedCallbacks              int                                          `json:"combat_discarded_callbacks"`
	CombatCallbackDiagnostics             map[string]combatCallbackQualityDiagnostics  `json:"combat_callback_diagnostics"`
	CombatDiscardedCallbackReasons        map[string]int                               `json:"combat_discarded_callback_reasons"`
	Engagements                           int                                          `json:"engagements"`
	TradeCandidates                       int                                          `json:"trade_candidates"`
	TradeCompletions                      int                                          `json:"trade_completions"`
	EngagementEventContractViolations     int                                          `json:"engagement_event_contract_violations"`
	EngagementAtomicProvenanceViolations  int                                          `json:"engagement_atomic_provenance_violations"`
	EngagementParticipantMismatches       int                                          `json:"engagement_participant_reconciliation_mismatches"`
	EngagementRoleConsistencyViolations   int                                          `json:"engagement_role_consistency_violations"`
	EngagementTemporalConsistencyErrors   int                                          `json:"engagement_temporal_consistency_violations"`
	EngagementCausalAvailabilityErrors    int                                          `json:"engagement_causal_availability_violations"`
	EngagementTradeReconciliationErrors   int                                          `json:"engagement_trade_reconciliation_mismatches"`
	EngagementStatsReconciliationErrors   int                                          `json:"engagement_stats_reconciliation_mismatches"`
	EngagementDeterminismViolations       int                                          `json:"engagement_determinism_violations"`
	EngagementObservationWarnings         int                                          `json:"engagement_observation_warnings"`
	EconomyTeamIdentityViolations         int                                          `json:"economy_team_identity_violations"`
	EconomyNativeCalculatedViolations     int                                          `json:"economy_native_calculated_reconciliation_violations"`
	EconomyNativeCalculatedDifferences    int                                          `json:"economy_native_calculated_differences"`
	EconomyMoneyTransitionViolations      int                                          `json:"economy_money_transition_violations"`
	EconomyPurchaseProvenanceViolations   int                                          `json:"economy_purchase_provenance_violations"`
	EconomyPriceTableVersionViolations    int                                          `json:"economy_price_table_version_violations"`
	StatsScoreboardReconciliationErrors   int                                          `json:"stats_scoreboard_reconciliation_mismatches"`
	StatsUtilityReconciliationErrors      int                                          `json:"stats_utility_reconciliation_mismatches"`
	ClutchAttemptReconciliationErrors     int                                          `json:"clutch_attempt_reconciliation_mismatches"`
	WarmupContaminationViolations         int                                          `json:"warmup_contamination_violations"`
	MetadataProvenanceViolations          int                                          `json:"metadata_provenance_violations"`
	MetadataChecksumLineageViolations     int                                          `json:"metadata_checksum_lineage_violations"`
	EconomyDeterminismViolations          int                                          `json:"economy_determinism_violations"`
	StatsDeterminismViolations            int                                          `json:"stats_determinism_violations"`
	EconomyObservationWarnings            int                                          `json:"economy_observation_warnings"`
	Block7ArtifactIntegrityViolations     int                                          `json:"block7_artifact_integrity_violations"`
	Block7CausalAvailabilityViolations    int                                          `json:"block7_causal_availability_violations"`
	Block7FutureLeakageViolations         int                                          `json:"block7_future_leakage_violations"`
	Block7SchemaCompatibilityViolations   int                                          `json:"block7_schema_compatibility_violations"`
	Block7DeterminismViolations           int                                          `json:"block7_determinism_violations"`
	Block7CorpusQualityViolations         int                                          `json:"block7_corpus_quality_violations"`
	Domains                               []qualityDomain                              `json:"domains"`
	Checks                                []qualityCheck                               `json:"checks"`
	Warnings                              []string                                     `json:"warnings"`
	objectiveFailureDetails               []string                                     `json:"-"`
	utilityFailureDetails                 []string                                     `json:"-"`
	combatFailureDetails                  []string                                     `json:"-"`
	engagementFailureDetails              []string                                     `json:"-"`
	block6FailureDetails                  []string                                     `json:"-"`
}

func (report qualityReport) hasHardObjectiveFailure() bool {
	return report.ObjectiveContractViolations > 0 ||
		report.ObjectiveRoundMismatches > 0 ||
		report.ObjectiveTerminalMismatches > 0 ||
		report.ObjectiveLifecycleViolations > 0 ||
		report.ObjectiveCarrierMismatches > 0 ||
		report.ObjectiveReplayEventMismatches > 0
}

func (report qualityReport) hasHardUtilityFailure() bool {
	return report.UtilityContractViolations > 0 ||
		report.UtilityThrowReconciliationMismatches > 0 ||
		report.UtilityPlayerStatsMismatches > 0 ||
		report.UtilityReplayProjectionMismatches > 0 ||
		report.UtilityLifecycleViolations > 0 ||
		report.UtilityFlashAttributionMismatches > 0 ||
		report.UtilityDamageReconciliationMismatches > 0 ||
		report.UtilityTemporalSpatialViolations > 0 ||
		report.UtilityDeterminismViolations > 0
}

func (report qualityReport) hasHardCombatFailure() bool {
	return report.CombatContractViolations > 0 ||
		report.CombatCallbackAccountingViolations > 0 ||
		report.CombatPlayerStatsMismatches > 0 ||
		report.CombatReplayProjectionMismatches > 0 ||
		report.CombatNativeDeltaMismatches > 0 ||
		report.CombatDeterminismViolations > 0
}

type qualityCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Message  string `json:"message"`
}

type trackingQualityStats struct {
	aliveStates                  int
	velocityObserved             int
	positiveVelocityStates       int
	activeWeaponObserved         int
	contractViolations           int
	movingDatasetWithoutVelocity int
}

type trackingSeriesKey struct {
	round    int
	playerID uint64
}

type trackingSeriesState struct {
	position        models.AI_Vector
	hasPosition     bool
	movingIntervals int
}

func assessTrackingQuality(events []models.AI_TrackingEventWithRound) trackingQualityStats {
	stats := trackingQualityStats{}
	series := make(map[trackingSeriesKey]trackingSeriesState)
	for _, wrapped := range events {
		event := wrapped.Event
		if trackingStateContractInvalid(event) {
			stats.contractViolations++
		}
		if !event.IsAlive {
			continue
		}
		stats.aliveStates++
		if event.VelocityAvailable {
			stats.velocityObserved++
			if math.Hypot(event.VelocityVector.X, event.VelocityVector.Y) > 1e-6 {
				stats.positiveVelocityStates++
			}
		}
		if event.ActiveWeaponStatus == models.ActiveWeaponStatusObserved {
			stats.activeWeaponObserved++
		}

		key := trackingSeriesKey{round: wrapped.Round, playerID: event.PlayerSteamID}
		state := series[key]
		if state.hasPosition && trackingPositionDistance(state.position, event.Position) > 0.01 {
			state.movingIntervals++
		}
		state.position = event.Position
		state.hasPosition = true
		series[key] = state
	}
	movingIntervals := 0
	for _, state := range series {
		movingIntervals += state.movingIntervals
	}
	if movingIntervals >= 2 && stats.positiveVelocityStates == 0 {
		stats.movingDatasetWithoutVelocity = 1
	}
	return stats
}

func trackingStateContractInvalid(event models.AI_TrackingEvent) bool {
	if !finiteAIVector(event.Position) {
		return true
	}
	if trackingObjectiveStateContractInvalid(event) {
		return true
	}
	source := playerstate.VelocitySource(event.VelocitySource)
	if event.IsAlive {
		if event.VelocityAvailable {
			if source != playerstate.VelocitySourceNative && source != playerstate.VelocitySourcePositionDelta {
				return true
			}
			horizontal := math.Hypot(event.VelocityVector.X, event.VelocityVector.Y)
			if !finiteAIVector(event.VelocityVector) || horizontal > playerstate.MaxPlausibleHorizontalSpeedUPS ||
				math.Abs(event.VelocityVector.Z) > playerstate.MaxPlausibleVerticalSpeedUPS {
				return true
			}
			if source == playerstate.VelocitySourceNative && event.VelocityMeasurementTicks != 0 {
				return true
			}
			if source == playerstate.VelocitySourcePositionDelta &&
				(event.VelocityMeasurementTicks < 1 || event.VelocityMeasurementTicks > playerstate.MaxPositionDeltaIntervalTicks) {
				return true
			}
		} else if !unavailableVelocitySource(source) || event.VelocityVector != (models.AI_Vector{}) || event.VelocityMeasurementTicks != 0 {
			return true
		}
		if event.ActiveWeaponStatus == models.ActiveWeaponStatusObserved {
			if event.ActiveWeapon == nil || strings.TrimSpace(*event.ActiveWeapon) == "" || event.LastObservedActiveWeapon != nil {
				return true
			}
		} else if event.ActiveWeaponStatus != models.ActiveWeaponStatusUnavailable || event.ActiveWeapon != nil {
			return true
		}
	} else {
		if event.VelocityAvailable || source != playerstate.VelocitySourceNotApplicable ||
			event.VelocityVector != (models.AI_Vector{}) || event.VelocityMeasurementTicks != 0 {
			return true
		}
		if event.ActiveWeaponStatus != models.ActiveWeaponStatusNotApplicable || event.ActiveWeapon != nil {
			return true
		}
	}
	if (event.LastObservedActiveWeapon == nil) != (event.LastObservedActiveWeaponTick == nil) {
		return true
	}
	return event.LastObservedActiveWeapon != nil &&
		(strings.TrimSpace(*event.LastObservedActiveWeapon) == "" || *event.LastObservedActiveWeaponTick > event.Tick)
}

func trackingObjectiveStateContractInvalid(event models.AI_TrackingEvent) bool {
	if event.PhaseTimeRemaining == nil || !finiteNonNegative(*event.PhaseTimeRemaining) ||
		!finiteNonNegative(event.RoundTimeRemaining) || math.Abs(event.RoundTimeRemaining-*event.PhaseTimeRemaining) > 1e-6 {
		return true
	}
	if !event.IsAlive && (event.HasC4 || event.IsPlanting || event.IsDefusing) || event.IsPlanting && event.IsDefusing {
		return true
	}
	if event.HasC4 && (event.Team != "T" || !event.IsAlive) {
		return true
	}
	if event.HasDefuseKit && event.Team != "CT" {
		return true
	}
	if event.IsPlanting && (!event.IsAlive || !event.HasC4 || event.Team != "T" || event.ObjectivePhase != "planting") {
		return true
	}
	if event.IsDefusing && (!event.IsAlive || event.HasC4 || event.Team != "CT" || event.ObjectivePhase != "defusing") {
		return true
	}

	phaseClockMatches := func(clock *float64) bool {
		return clock != nil && finiteNonNegative(*clock) && math.Abs(*clock-*event.PhaseTimeRemaining) <= 1e-6
	}
	switch event.ObjectivePhase {
	case "preplant":
		return !phaseClockMatches(event.RoundClockRemaining) || event.BombTimeRemaining != nil || event.IsPlanting || event.IsDefusing
	case "planting":
		return !phaseClockMatches(event.RoundClockRemaining) || event.BombTimeRemaining != nil || event.IsDefusing
	case "planted":
		return event.RoundClockRemaining != nil || !phaseClockMatches(event.BombTimeRemaining) || event.HasC4 || event.IsPlanting || event.IsDefusing
	case "defusing":
		return event.RoundClockRemaining != nil || !phaseClockMatches(event.BombTimeRemaining) || event.HasC4 || event.IsPlanting
	case "resolved":
		return event.RoundClockRemaining != nil || event.BombTimeRemaining != nil || *event.PhaseTimeRemaining != 0 ||
			event.HasC4 || event.IsPlanting || event.IsDefusing
	default:
		return true
	}
}

func unavailableVelocitySource(source playerstate.VelocitySource) bool {
	switch source {
	case playerstate.VelocitySourceInsufficientHistory,
		playerstate.VelocitySourceRejected,
		playerstate.VelocitySourceStaleGap,
		playerstate.VelocitySourceEntityChanged,
		playerstate.VelocitySourceNonMonotonicTick:
		return true
	default:
		return false
	}
}

func finiteAIVector(vector models.AI_Vector) bool {
	return !math.IsNaN(vector.X) && !math.IsInf(vector.X, 0) &&
		!math.IsNaN(vector.Y) && !math.IsInf(vector.Y, 0) &&
		!math.IsNaN(vector.Z) && !math.IsInf(vector.Z, 0)
}

func trackingPositionDistance(left, right models.AI_Vector) float64 {
	dx := right.X - left.X
	dy := right.Y - left.Y
	dz := right.Z - left.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func countCompleteRoundSequenceErrors(rounds []int, expected int) int {
	counts := make(map[int]int, len(rounds))
	errors := 0
	for _, round := range rounds {
		if round < 1 || round > expected {
			errors++
			continue
		}
		counts[round]++
	}
	for round := 1; round <= expected; round++ {
		switch count := counts[round]; {
		case count == 0:
			errors++
		case count > 1:
			errors += count - 1
		}
	}
	if errors == 0 {
		for index, round := range rounds {
			if round != index+1 {
				return 1
			}
		}
	}
	return errors
}

func sortedUniqueRounds(rounds []int) []int {
	seen := make(map[int]struct{}, len(rounds))
	for _, round := range rounds {
		seen[round] = struct{}{}
	}
	unique := make([]int, 0, len(seen))
	for round := range seen {
		unique = append(unique, round)
	}
	sort.Ints(unique)
	return unique
}

func countSparseRoundSequenceErrors(rounds []int, expected int) int {
	errors := 0
	for _, round := range sortedUniqueRounds(rounds) {
		if round < 1 || round > expected {
			errors++
		}
	}
	return errors
}

func validAccuracy(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 100
}

func crosshairVectorInconsistent(participant models.AI_DuelParticipant) bool {
	total := participant.InitialCrosshairError
	pitch := participant.PitchError
	yaw := participant.YawError
	if math.IsNaN(total) || math.IsNaN(pitch) || math.IsNaN(yaw) ||
		math.IsInf(total, 0) || math.IsInf(pitch, 0) || math.IsInf(yaw, 0) ||
		total < 0 || total > 180 || pitch < 0 || pitch > 180 || yaw < 0 || yaw > 180 {
		return true
	}

	// The total is a 3D angular distance, not hypot(pitch, yaw). These triangle
	// bounds are safe without knowing the two absolute pitch angles.
	const toleranceDegrees = 0.5
	return total > pitch+yaw+toleranceDegrees || total+toleranceDegrees < pitch
}

func buildQualityReport(ctx *models.DemoContext) qualityReport {
	trackingQuality := assessTrackingQuality(ctx.AI_TrackingEventsWithRound)
	objectiveQuality := assessObjectiveQuality(ctx)
	utilityQuality := assessUtilityQuality(ctx)
	combatQuality := assessCombatQuality(ctx)
	replayRounds := 0
	replayRoundNumbers := make([]int, 0)
	invalidReplayBounds := 0
	replayFramesOutsideBounds := 0
	duplicateReplayIDs := 0
	missingReplayIDs := 0
	seenEventIDs := make(map[string]struct{})
	if ctx.ReplayData != nil {
		replayRounds = len(ctx.ReplayData.Rounds)
		for _, round := range ctx.ReplayData.Rounds {
			replayRoundNumbers = append(replayRoundNumbers, round.Round)
			if round.EndTick < round.StartTick {
				invalidReplayBounds++
			}
			for _, frame := range round.Frames {
				if frame.Tick < round.StartTick || frame.Tick > round.EndTick {
					replayFramesOutsideBounds++
				}
			}
			for _, event := range round.Events {
				if event.ID == "" {
					missingReplayIDs++
					continue
				}
				if _, exists := seenEventIDs[event.ID]; exists {
					duplicateReplayIDs++
				}
				seenEventIDs[event.ID] = struct{}{}
			}
		}
	}

	economyMismatches := 0
	economyStartMismatches := 0
	economyEndMismatches := 0
	economyEndNativeMismatches := 0
	purchaseItemizationMismatches := 0
	economyRoundNumbers := make([]int, 0, len(ctx.AI_EconomyRounds))
	for _, round := range ctx.AI_EconomyRounds {
		economyRoundNumbers = append(economyRoundNumbers, round.Round)
		for _, player := range round.Players {
			if player.EquipmentValueStart != player.EquipmentValueStartCalculated {
				economyStartMismatches++
			}
			if player.FinalEquipmentValue != player.FinalEquipmentValueCalculated {
				economyMismatches++
			}
			endEquipmentValueCalculated := 0
			for _, item := range player.EndEquipment {
				endEquipmentValueCalculated += item.Price
			}
			if player.EquipmentValueEnd != endEquipmentValueCalculated {
				economyEndMismatches++
			}
			if player.EquipmentValueEndNative != player.EquipmentValueEndCalculated {
				economyEndNativeMismatches++
			}
			if player.PurchasesVsSpentDelta != 0 {
				purchaseItemizationMismatches++
			}
		}
	}
	ttdSamples := 0
	crosshairSamples := 0
	if ctx.MatchData != nil {
		for _, player := range ctx.MatchData.Players {
			for _, reaction := range player.ReactionTimes {
				if reaction.TimeToDamage >= 50 && reaction.TimeToDamage <= 2500 && !reaction.SmokeInPath && !reaction.WasFlashed {
					ttdSamples++
				}
				if reaction.CrosshairPlacementError >= 0 && reaction.CrosshairPlacementError <= 90 && !reaction.SmokeInPath && !reaction.WasFlashed {
					crosshairSamples++
				}
			}
		}
	}
	nativeScoreboardMismatches := 0
	accuracyRangeMismatches := 0
	for _, player := range ctx.AI_PlayersSummary {
		native := player.NativeScoreboard
		if player.Kills != native.Kills || player.Deaths != native.Deaths || player.Assists != native.Assists ||
			player.TotalDamage != native.TotalDamage || player.UtilityDamage != native.UtilityDamage {
			nativeScoreboardMismatches++
		}
		if !validAccuracy(player.AccuracyOverall) {
			accuracyRangeMismatches++
		}
		for _, weapon := range player.WeaponStats {
			if !validAccuracy(weapon.Accuracy) {
				accuracyRangeMismatches++
			}
		}
	}

	trackingRoundNumbers := make([]int, 0, len(ctx.AI_TrackingEventsWithRound))
	for _, event := range ctx.AI_TrackingEventsWithRound {
		trackingRoundNumbers = append(trackingRoundNumbers, event.Round)
	}
	combatRoundNumbers := make([]int, 0, len(ctx.AI_Duels))
	crosshairVectorMismatches := 0
	for _, duel := range ctx.AI_Duels {
		combatRoundNumbers = append(combatRoundNumbers, duel.Round)
		if crosshairVectorInconsistent(duel.Attacker) {
			crosshairVectorMismatches++
		}
		for _, victim := range duel.Victims {
			if crosshairVectorInconsistent(victim) {
				crosshairVectorMismatches++
			}
		}
	}
	grenadeRoundNumbers := make([]int, 0, len(ctx.AI_GrenadeEvents))
	for _, event := range ctx.AI_GrenadeEvents {
		grenadeRoundNumbers = append(grenadeRoundNumbers, event.Round)
	}

	replayRoundSequenceErrors := countCompleteRoundSequenceErrors(replayRoundNumbers, ctx.CurrentRound)
	economyRoundSequenceErrors := countCompleteRoundSequenceErrors(economyRoundNumbers, ctx.CurrentRound)
	trackingRoundSequenceErrors := countCompleteRoundSequenceErrors(sortedUniqueRounds(trackingRoundNumbers), ctx.CurrentRound)
	combatRoundSequenceErrors := countSparseRoundSequenceErrors(combatRoundNumbers, ctx.CurrentRound)
	grenadeRoundSequenceErrors := countSparseRoundSequenceErrors(grenadeRoundNumbers, ctx.CurrentRound)

	checks := make([]qualityCheck, 0, 24)
	warnings := make([]string, 0)
	hasFailure := false
	addCheck := func(name, status, expected, actual, message string) {
		checks = append(checks, qualityCheck{Name: name, Status: status, Expected: expected, Actual: actual, Message: message})
		if status == "fail" {
			hasFailure = true
			warnings = append(warnings, message)
		} else if status == "warning" {
			warnings = append(warnings, message)
		}
	}

	parseStatus := "pass"
	if !ctx.ParseCompleted {
		parseStatus = "fail"
	}
	addCheck("parse_completed", parseStatus, "true", fmt.Sprint(ctx.ParseCompleted), "The parser must reach the end of the demo.")
	addCountCheck := func(name string, expected, actual int, failureMessage string) {
		status := "pass"
		if expected != actual {
			status = "fail"
		}
		addCheck(name, status, fmt.Sprint(expected), fmt.Sprint(actual), failureMessage)
	}
	addCountCheck("replay_round_count", ctx.CurrentRound, replayRounds, "Replay rounds must match parsed rounds.")
	addCountCheck("economy_round_count", ctx.CurrentRound, len(ctx.AI_EconomyRounds), "Economy rounds must match parsed rounds.")
	addCountCheck("replay_round_sequence", 0, replayRoundSequenceErrors, "Replay round labels must be exactly 1..N, without gaps, duplicates or reordering.")
	addCountCheck("economy_round_sequence", 0, economyRoundSequenceErrors, "Economy round labels must be exactly 1..N, without gaps, duplicates or reordering.")
	addCountCheck("tracking_round_sequence", 0, trackingRoundSequenceErrors, "Tracking round labels must cover exactly 1..N.")
	addCountCheck("combat_round_sequence", 0, combatRoundSequenceErrors, "Combat round labels with events must be inside 1..N; empty rounds may be omitted.")
	addCountCheck("grenade_round_sequence", 0, grenadeRoundSequenceErrors, "Grenade round labels with events must be inside 1..N; empty rounds may be omitted.")
	addCountCheck("replay_tick_bounds", 0, invalidReplayBounds+replayFramesOutsideBounds, "Replay rounds and frames must stay inside their tick boundaries.")
	addCountCheck("replay_event_ids", 0, duplicateReplayIDs+missingReplayIDs, "Replay event IDs must be present and unique.")
	addCountCheck("accuracy_ranges", 0, accuracyRangeMismatches, "Player and per-weapon accuracy percentages must be finite values between 0 and 100.")
	addCountCheck("crosshair_vector_consistency", 0, crosshairVectorMismatches, "Initial crosshair error must be geometrically consistent with its pitch and yaw components.")
	addCountCheck("player_state_contract", 0, trackingQuality.contractViolations, "Player states must preserve velocity availability and current-versus-historical weapon semantics.")
	addCountCheck("player_state_motion_signal", 0, trackingQuality.movingDatasetWithoutVelocity, "A match with repeated position changes must contain a positive causal velocity observation.")
	for _, check := range objectiveQuality.checks() {
		addCheck(check.Name, check.Status, check.Expected, check.Actual, check.Message)
	}
	for _, check := range utilityQuality.checks() {
		addCheck(check.Name, check.Status, check.Expected, check.Actual, check.Message)
	}
	for _, check := range combatQuality.checks() {
		addCheck(check.Name, check.Status, check.Expected, check.Actual, check.Message)
	}
	velocityCoverage := ratio(trackingQuality.velocityObserved, trackingQuality.aliveStates)
	velocityCoverageStatus := "pass"
	if trackingQuality.aliveStates == 0 || trackingQuality.velocityObserved == 0 {
		velocityCoverageStatus = "fail"
	} else if velocityCoverage < 0.9 {
		velocityCoverageStatus = "warning"
	}
	addCheck("player_state_velocity_coverage", velocityCoverageStatus, ">=0.90", fmt.Sprintf("%.6f", velocityCoverage), "Alive player states should contain causal horizontal velocity with explicit availability.")
	weaponCoverage := ratio(trackingQuality.activeWeaponObserved, trackingQuality.aliveStates)
	weaponCoverageStatus := "pass"
	if trackingQuality.aliveStates == 0 || weaponCoverage < 0.9 {
		weaponCoverageStatus = "warning"
	}
	addCheck("player_state_active_weapon_coverage", weaponCoverageStatus, ">=0.90", fmt.Sprintf("%.6f", weaponCoverage), "Alive player states should contain directly observed active weapons; missing observations remain null.")
	addCountCheck("economy_round_end_reconciliation", 0, economyEndMismatches, "Round-end equipment value must equal the sum of the exported end_equipment item prices.")
	economyEndNativeStatus := "pass"
	if economyEndNativeMismatches > 0 {
		economyEndNativeStatus = "warning"
	}
	addCheck("economy_round_end_native_reconciliation", economyEndNativeStatus, "0", fmt.Sprint(economyEndNativeMismatches), "Native and calculated round-end equipment values differ for some player-rounds; both values remain separate and native is never overwritten.")

	parserWarningStatus := "pass"
	if len(ctx.ParserWarnings) > 0 {
		parserWarningStatus = "warning"
	}
	addCheck("parser_warnings", parserWarningStatus, "0", fmt.Sprint(len(ctx.ParserWarnings)), "Non-fatal parser warnings may indicate incomplete fields.")

	economyStatus := "pass"
	if economyMismatches > 0 {
		economyStatus = "warning"
	}
	addCheck("economy_freeze_native_reconciliation", economyStatus, "0", fmt.Sprint(economyMismatches), "Native and calculated freeze-time equipment values differ for some player-rounds.")
	economyStartStatus := "pass"
	if economyStartMismatches > 0 {
		economyStartStatus = "warning"
	}
	addCheck("economy_round_start_native_reconciliation", economyStartStatus, "0", fmt.Sprint(economyStartMismatches), "Native and calculated round-start equipment values differ for some player-rounds.")
	purchaseStatus := "pass"
	if purchaseItemizationMismatches > 0 {
		purchaseStatus = "warning"
	}
	addCheck("purchase_itemization_coverage", purchaseStatus, "0", fmt.Sprint(purchaseItemizationMismatches), "Some native buy spend could not be attributed to observed ItemPickup events; use spent_in_buy as authoritative and purchases_vs_spent_delta as coverage metadata.")
	nativeStatus := "pass"
	if nativeScoreboardMismatches > 0 {
		nativeStatus = "warning"
	}
	addCheck("native_scoreboard_reconciliation", nativeStatus, "0", fmt.Sprint(nativeScoreboardMismatches), "Public kills, deaths, assists, total damage or utility damage differ from the native scoreboard for some players.")

	raytracingAvailable := ctx.MapManager != nil && ctx.MapManager.IsLoaded()
	if ctx.MapManager != nil {
		rayStatus := "pass"
		rayMessage := "Physics-mesh raytracing was available for visibility metrics."
		if !raytracingAvailable {
			rayStatus = "warning"
			rayMessage = "Physics-mesh raytracing was unavailable; TTD and crosshair placement were not fabricated from fallback visibility."
			if ctx.MapLoadError != "" {
				rayMessage += " Map load error: " + ctx.MapLoadError
			}
		}
		addCheck("raytracing_visibility", rayStatus, "available", fmt.Sprint(raytracingAvailable), rayMessage)
	}
	addCheck("visibility_model", "pass", "canonical_16_9_frustum_scope_mask_aware", "canonical_16_9_frustum_scope_mask_aware", "Visibility uses a canonical 16:9 frustum, scoped-weapon masks and zoom levels, no artificial distance cutoff, and engagement-tick refinement.")
	aimStatus := "not_available"
	aimMessage := "No valid TTD or crosshair-placement samples were produced."
	if ttdSamples > 0 || crosshairSamples > 0 {
		aimStatus = "pass"
		aimMessage = "Valid visual samples are available; see the sample counters for coverage."
	}
	addCheck("aim_metric_coverage", aimStatus, ">0", fmt.Sprintf("ttd=%d,crosshair=%d", ttdSamples, crosshairSamples), aimMessage)

	bulletStatus := "not_available"
	bulletMessage := "The demo emitted no BulletDamage events; older CS2 demos may not contain them."
	if ctx.BulletDamageEvents > 0 {
		bulletStatus = "pass"
		bulletMessage = "BulletDamage events were correlated with combat impacts."
		if ctx.BulletDamageCorrelated != ctx.BulletDamageEvents {
			bulletStatus = "warning"
			bulletMessage = "Some BulletDamage events could not be correlated with a PlayerHurt or Kill event."
		}
	}
	addCheck("bullet_damage_correlation", bulletStatus, fmt.Sprint(ctx.BulletDamageEvents), fmt.Sprint(ctx.BulletDamageCorrelated), bulletMessage)

	status := "pass"
	if hasFailure {
		status = "fail"
	} else if len(warnings) > 0 {
		status = "warning"
	}
	return qualityReport{
		SchemaVersion:                   block6QualitySchema,
		DemoinfocsVersion:               "v5.2.0",
		RatingModel:                     "HLTV 2.0 approximation; not an official HLTV implementation",
		IdentifierEncoding:              "decimal strings",
		Status:                          status,
		UsableForTraining:               !hasFailure,
		StatusMeaning:                   "pass: all required checks passed; warning: usable with documented limitations; fail: do not use for training",
		ParseCompleted:                  ctx.ParseCompleted,
		ParserWarnings:                  ctx.ParserWarnings,
		ExpectedRounds:                  ctx.CurrentRound,
		ReplayRounds:                    replayRounds,
		EconomyRounds:                   len(ctx.AI_EconomyRounds),
		TrackingEvents:                  len(ctx.AI_TrackingEventsWithRound),
		CombatDuels:                     len(ctx.AI_Duels),
		GrenadeEvents:                   len(ctx.AI_GrenadeEvents),
		PlayerSummaries:                 len(ctx.AI_PlayersSummary),
		ReplayRoundSequenceErrors:       replayRoundSequenceErrors,
		EconomyRoundSequenceErrors:      economyRoundSequenceErrors,
		TrackingRoundSequenceErrors:     trackingRoundSequenceErrors,
		CombatRoundSequenceErrors:       combatRoundSequenceErrors,
		GrenadeRoundSequenceErrors:      grenadeRoundSequenceErrors,
		EconomyReconciliationMismatches: economyMismatches,
		EconomyStartMismatches:          economyStartMismatches,
		EconomyEndMismatches:            economyEndMismatches,
		EconomyEndNativeMismatches:      economyEndNativeMismatches,
		PurchaseItemizationMismatches:   purchaseItemizationMismatches,
		NativeScoreboardMismatches:      nativeScoreboardMismatches,
		AccuracyRangeMismatches:         accuracyRangeMismatches,
		CrosshairVectorMismatches:       crosshairVectorMismatches,
		ReplayFramesOutsideBounds:       replayFramesOutsideBounds,
		RaytracingAvailable:             raytracingAvailable,
		VisibilitySampledTicks:          ctx.VisibilitySampledTicks,
		VisibilitySkippedTicks:          ctx.VisibilitySkippedTicks,
		VisibilityRaycasts:              ctx.VisibilityRaycasts,
		VisibilityRefinementRaycasts:    ctx.VisibilityRefinementRaycasts,
		VisibilityModel:                 "canonical_16_9_frustum_scope_mask_aware",
		VisibilityReferenceAspect:       "16:9",
		VisibilityReferenceFOV:          90,
		VisibilityScopedFOVs: map[string]float64{
			"awp_zoom_1": 40, "awp_zoom_2": 10,
			"ssg08_zoom_1": 40, "ssg08_zoom_2": 15,
			"autosniper_zoom_1": 40, "autosniper_zoom_2": 15,
			"aug_sg553_zoom_1": 45,
		},
		VisibilitySampleStride:                4,
		VisibilityDistanceLimit:               "none",
		NativeEyePositions:                    ctx.NativeEyePositions,
		FallbackEyePositions:                  ctx.FallbackEyePositions,
		TTDSamples:                            ttdSamples,
		CrosshairPlacementSamples:             crosshairSamples,
		BulletDamageEvents:                    ctx.BulletDamageEvents,
		BulletDamageCorrelated:                ctx.BulletDamageCorrelated,
		TrackingAliveStates:                   trackingQuality.aliveStates,
		TrackingVelocityObserved:              trackingQuality.velocityObserved,
		TrackingPositiveVelocityStates:        trackingQuality.positiveVelocityStates,
		TrackingVelocityCoverageRatio:         velocityCoverage,
		TrackingActiveWeaponObserved:          trackingQuality.activeWeaponObserved,
		TrackingActiveWeaponCoverage:          weaponCoverage,
		TrackingStateContractViolations:       trackingQuality.contractViolations,
		TrackingMovingDatasetWithoutData:      trackingQuality.movingDatasetWithoutVelocity,
		ObjectiveEvents:                       objectiveQuality.events,
		ObjectiveCarrierSnapshots:             objectiveQuality.carrierSnapshots,
		ObjectiveBombDrops:                    objectiveQuality.bombDrops,
		ObjectiveBombPickups:                  objectiveQuality.bombPickups,
		ObjectivePlantStarts:                  objectiveQuality.plantStarts,
		ObjectivePlantAborts:                  objectiveQuality.plantAborts,
		ObjectivePlants:                       objectiveQuality.plants,
		ObjectiveDefuseStarts:                 objectiveQuality.defuseStarts,
		ObjectiveDefuseAborts:                 objectiveQuality.defuseAborts,
		ObjectiveDefuses:                      objectiveQuality.defuses,
		ObjectiveBombExplosions:               objectiveQuality.bombExplosions,
		ObjectiveMissingActors:                objectiveQuality.missingActors,
		ObjectiveMissingPositions:             objectiveQuality.missingPositions,
		ObjectiveAttempts:                     objectiveQuality.attempts,
		ObjectiveAttemptsMissingStart:         objectiveQuality.attemptsMissingStart,
		ObjectiveAttemptsUnclosed:             objectiveQuality.attemptsUnclosed,
		ObjectiveContractViolations:           objectiveQuality.contractViolations,
		ObjectiveRoundMismatches:              objectiveQuality.roundMismatches,
		ObjectiveTerminalMismatches:           objectiveQuality.terminalMismatches,
		ObjectiveLifecycleViolations:          objectiveQuality.lifecycleViolations,
		ObjectiveTrackingCarrierSamples:       objectiveQuality.trackingCarrierSamples,
		ObjectiveReplayCarrierSamples:         objectiveQuality.replayCarrierSamples,
		ObjectiveReplayBombStateMissing:       objectiveQuality.replayBombStateMissing,
		ObjectiveTrackingCarrierErrors:        objectiveQuality.trackingCarrierMismatches,
		ObjectiveReplayCarrierErrors:          objectiveQuality.replayCarrierMismatches,
		ObjectiveCarrierMismatches:            objectiveQuality.trackingCarrierMismatches + objectiveQuality.replayCarrierMismatches,
		ObjectiveReplayEventMismatches:        objectiveQuality.replayEventMismatches,
		ObjectiveNativeRoleDisagreements:      ctx.ObjectiveNativeRoleDisagreements,
		UtilityThrows:                         utilityQuality.throws,
		UtilityCanonicalEvents:                utilityQuality.canonicalEvents,
		UtilityFlashEffects:                   utilityQuality.flashEffects,
		UtilityDamageEffects:                  utilityQuality.damageEffects,
		UtilityContractViolations:             utilityQuality.contractViolations,
		UtilityThrowReconciliationMismatches:  utilityQuality.throwReconciliationMismatches,
		UtilityPlayerStatsMismatches:          utilityQuality.playerStatsMismatches,
		UtilityReplayProjectionMismatches:     utilityQuality.replayProjectionMismatches,
		UtilityCallbackAccountingViolations:   utilityQuality.callbackAccountingViolations,
		UtilityThrowCallbacks:                 utilityQuality.diagnostics.Throws.Observed,
		UtilityBounceCallbacks:                utilityQuality.diagnostics.Bounces.Observed,
		UtilityLifecycleCallbacks:             utilityQuality.diagnostics.Lifecycle.Observed,
		UtilityPlayerFlashedCallbacks:         utilityQuality.diagnostics.Flashes.Observed,
		UtilityDamageCallbacks:                utilityQuality.diagnostics.Damage.Observed,
		UtilityCallbackDiagnostics:            utilityQuality.callbackDiagnosticsReport(),
		UtilityUnmatchedCallbacks:             utilityQuality.unmatchedCallbacks,
		UtilityOrphanCallbacks:                utilityQuality.orphanCallbacks,
		UtilityInferredCallbacks:              utilityQuality.inferredCallbacks,
		UtilityDeduplicatedCallbacks:          utilityQuality.deduplicatedCallbacks,
		UtilityLifecycleViolations:            utilityQuality.lifecycleViolations,
		UtilityFlashAttributionMismatches:     utilityQuality.flashAttributionMismatches,
		UtilityDamageReconciliationMismatches: utilityQuality.damageReconciliationMismatches,
		UtilityTemporalSpatialViolations:      utilityQuality.temporalSpatialViolations,
		UtilityDeterminismViolations:          utilityQuality.determinismViolations,
		UtilityObservationWarnings:            utilityQuality.observationWarnings,
		UtilityMissingTypeObservations:        utilityQuality.missingTypeObservations,
		UtilityMissingActorObservations:       utilityQuality.missingActorObservations,
		UtilityMissingAffectedPlayers:         utilityQuality.missingAffectedPlayerObservations,
		UtilityMissingFlashDurations:          utilityQuality.missingFlashDurationObservations,
		UtilityMissingLaunchTicks:             utilityQuality.missingLaunchTicks,
		UtilityMissingLaunchPositions:         utilityQuality.missingLaunchPositions,
		UtilityMissingLaunchViews:             utilityQuality.missingLaunchViews,
		UtilityMissingThrowerVelocities:       utilityQuality.missingThrowerVelocities,
		UtilityMissingProjectileVelocities:    utilityQuality.missingProjectileVelocities,
		UtilityMissingTrajectoryObservations:  utilityQuality.missingTrajectoryObservations,
		UtilityMissingLifecycleObservations:   utilityQuality.missingLifecycleObservations,
		UtilityInferredCorrelations:           utilityQuality.inferredCorrelations,
		UtilityObservedEffectCorrelations:     utilityQuality.observedEffectCorrelations,
		UtilityInferredEffectCorrelations:     utilityQuality.inferredEffectCorrelations,
		UtilityUnavailableEffectCorrelations:  utilityQuality.unavailableEffectCorrelations,
		CombatLedgerEvents:                    combatQuality.events,
		CombatContractViolations:              combatQuality.contractViolations,
		CombatCallbackAccountingViolations:    combatQuality.callbackAccountingViolations,
		CombatPlayerStatsMismatches:           combatQuality.playerStatsMismatches,
		CombatReplayProjectionMismatches:      combatQuality.replayProjectionMismatches,
		CombatNativeDeltaMismatches:           combatQuality.nativeDeltaMismatches,
		CombatDeterminismViolations:           combatQuality.determinismViolations,
		CombatMissingImpactPositions:          combatQuality.missingImpactPositions,
		CombatMissingReloadEnds:               combatQuality.missingReloadEnds,
		CombatUnavailableShotResults:          combatQuality.unavailableShotResults,
		CombatDiscardedCallbacks:              combatQuality.discardedCallbacks,
		CombatCallbackDiagnostics:             combatQuality.callbackDiagnosticsReport(),
		CombatDiscardedCallbackReasons:        cloneStringIntMap(combatQuality.diagnostics.DiscardedByReason),
		Checks:                                checks,
		Warnings:                              warnings,
		objectiveFailureDetails:               append([]string(nil), objectiveQuality.failureDetails...),
		utilityFailureDetails:                 append([]string(nil), utilityQuality.failureDetails...),
		combatFailureDetails:                  append([]string(nil), combatQuality.failureDetails...),
	}
}

// writeJSON writes any data structure to a JSON file with indentation
func writeJSON(filepath string, data interface{}) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filepath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filepath, err)
	}

	return nil
}

func checksumFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
