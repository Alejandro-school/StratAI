package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/navigation"
)

const (
	ObservedNavigationFullDemoSchemaID             = "stratai.observed_navigation_full_demo_sweep@1"
	ObservedNavigationFullDemoScope                = "temporal_full_demo_sweep"
	ObservedNavigationFullDemoSelectionContract    = "one_deterministic_stress_pair_per_eligible_replay_frame@1"
	ObservedNavigationFullDemoDistributionContract = "nearest_rank_p50_p95_max@1"
	ObservedNavigationFullDemoRowsDigestContract   = "sha256_uncompressed_jsonl_bytes@1"
	ObservedNavigationFullDemoReportFilename       = "observed_navigation_full_demo.json"
	ObservedNavigationFullDemoRowsFilename         = "observed_navigation_full_demo_rows.jsonl.gz"
)

// ObservedNavigationFullDemoRowsArtifact describes the streamed row sidecar.
// The builder fills the logical fields. The command owns compression and fills
// Path, Bytes, and SHA256 only after it closes and hashes the gzip artifact.
type ObservedNavigationFullDemoRowsArtifact struct {
	Path           string `json:"path"`
	Format         string `json:"format"`
	Compression    string `json:"compression"`
	Bytes          int64  `json:"bytes"`
	SHA256         string `json:"sha256"`
	RowCount       int    `json:"row_count"`
	LogicalBytes   int64  `json:"logical_bytes"`
	LogicalSHA256  string `json:"logical_sha256"`
	DigestContract string `json:"digest_contract"`
}

type ObservedNavigationFullDemoNumericDistribution struct {
	Count int      `json:"count"`
	P50   *float64 `json:"p50"`
	P95   *float64 `json:"p95"`
	Max   *float64 `json:"max"`
}

type ObservedNavigationFullDemoDistributions struct {
	DirectDistanceWorldUnits                   ObservedNavigationFullDemoNumericDistribution `json:"direct_distance_world_units"`
	PathDistanceWorldUnits                     ObservedNavigationFullDemoNumericDistribution `json:"path_distance_world_units"`
	PathToDirectDistanceRatio                  ObservedNavigationFullDemoNumericDistribution `json:"path_to_direct_distance_ratio"`
	ObservedHorizontalSpeedWorldUnitsPerSecond ObservedNavigationFullDemoNumericDistribution `json:"observed_horizontal_speed_world_units_per_second"`
	WeaponBaselineSpeedWorldUnitsPerSecond     ObservedNavigationFullDemoNumericDistribution `json:"weapon_baseline_speed_world_units_per_second"`
	OptimisticBaselineMilliseconds             ObservedNavigationFullDemoNumericDistribution `json:"optimistic_baseline_milliseconds"`
	ActiveWeaponBaselineMilliseconds           ObservedNavigationFullDemoNumericDistribution `json:"active_weapon_baseline_milliseconds"`
	ObservedPaceProjectionMilliseconds         ObservedNavigationFullDemoNumericDistribution `json:"observed_pace_projection_milliseconds"`
}

type ObservedNavigationFullDemoScenarioCounts struct {
	Estimated             int            `json:"estimated"`
	Unavailable           int            `json:"unavailable"`
	UnavailabilityReasons map[string]int `json:"unavailability_reasons"`
}

type ObservedNavigationFullDemoCounts struct {
	Queries                    int `json:"queries"`
	ReachableQueries           int `json:"reachable_queries"`
	DisconnectedQueries        int `json:"disconnected_queries"`
	EndpointUnavailableQueries int `json:"endpoint_unavailable_queries"`

	OptimisticBaseline     ObservedNavigationFullDemoScenarioCounts `json:"optimistic_baseline"`
	ActiveWeaponBaseline   ObservedNavigationFullDemoScenarioCounts `json:"active_weapon_baseline"`
	ObservedPaceProjection ObservedNavigationFullDemoScenarioCounts `json:"observed_pace_projection"`

	WalkingContexts            int `json:"walking_contexts"`
	NonWalkingMovingContexts   int `json:"non_walking_moving_contexts"`
	ObservedMovingContexts     int `json:"observed_moving_contexts"`
	ObservedStationaryContexts int `json:"observed_stationary_contexts"`
	MotionUnavailableContexts  int `json:"motion_unavailable_contexts"`
	DuckingContexts            int `json:"ducking_contexts"`
	ScopedContexts             int `json:"scoped_contexts"`
	KnifeContexts              int `json:"knife_contexts"`

	ActiveWeaponContexts          map[string]int `json:"active_weapon_contexts"`
	WeaponBaselineSpeedContexts   map[string]int `json:"weapon_baseline_speed_contexts"`
	AbstentionReasons             map[string]int `json:"abstention_reasons"`
	ExactUserFacingTimeViolations int            `json:"exact_user_facing_time_violations"`
	ArrivalPredictionViolations   int            `json:"arrival_prediction_violations"`
}

type ObservedNavigationFullDemoSummary struct {
	ReplayFrames     int                                     `json:"replay_frames"`
	EligibleFrames   int                                     `json:"eligible_frames"`
	IneligibleFrames int                                     `json:"ineligible_frames"`
	Counts           ObservedNavigationFullDemoCounts        `json:"counts"`
	Distributions    ObservedNavigationFullDemoDistributions `json:"distributions"`
}

type ObservedNavigationFullDemoRoundSummary struct {
	RoundNumber      int                              `json:"round_number"`
	ReplayFrames     int                              `json:"replay_frames"`
	EligibleFrames   int                              `json:"eligible_frames"`
	IneligibleFrames int                              `json:"ineligible_frames"`
	Counts           ObservedNavigationFullDemoCounts `json:"counts"`
}

type ObservedNavigationFullDemoTimingPolicy struct {
	IntendedUse                string `json:"intended_use"`
	UserFacingMode             string `json:"user_facing_mode"`
	DecisionSemantics          string `json:"decision_semantics"`
	EstimateSemantics          string `json:"estimate_semantics"`
	ExactUserFacingTimeAllowed bool   `json:"exact_user_facing_time_allowed"`
	ArrivalPrediction          bool   `json:"arrival_prediction"`
}

// ObservedNavigationFullDemoReport is a compact non-canonical audit. It covers
// every eligible 16 Hz replay frame but deliberately evaluates only one
// directional stress pair per frame; it is not participant-pair exhaustive.
type ObservedNavigationFullDemoReport struct {
	SchemaID          string `json:"schema_id"`
	Status            string `json:"status"`
	SourceAlias       string `json:"source_alias"`
	SourceSHA256      string `json:"source_sha256"`
	MapName           string `json:"map_name"`
	Scope             string `json:"scope"`
	SelectionContract string `json:"selection_contract"`
	PairExhaustive    bool   `json:"pair_exhaustive"`
	DirectionalPairs  bool   `json:"directional_pairs"`

	CandidateSchemaID        string `json:"candidate_schema_id"`
	CandidateContractVersion string `json:"candidate_contract_version"`
	GraphContractVersion     string `json:"graph_contract_version"`
	GraphAssetReference      string `json:"graph_asset_reference"`
	GraphAssetSHA256         string `json:"graph_asset_sha256"`
	GraphNodeCount           int    `json:"graph_node_count"`
	GraphEdgeCount           int    `json:"graph_edge_count"`

	RoundCount                 int                                    `json:"round_count"`
	TickRate                   float64                                `json:"tick_rate"`
	SampleStrideTicks          int                                    `json:"sample_stride_ticks"`
	SampleIntervalMilliseconds float64                                `json:"sample_interval_milliseconds"`
	DistributionContract       string                                 `json:"distribution_contract"`
	TimingPolicy               ObservedNavigationFullDemoTimingPolicy `json:"timing_policy"`

	BuildIdentifier                string                                   `json:"build_identifier"`
	ProcessingDurationMS           int64                                    `json:"processing_duration_ms"`
	NavigationEvaluationDurationMS int64                                    `json:"navigation_evaluation_duration_ms"`
	RowsArtifact                   ObservedNavigationFullDemoRowsArtifact   `json:"rows_artifact"`
	Summary                        ObservedNavigationFullDemoSummary        `json:"summary"`
	Rounds                         []ObservedNavigationFullDemoRoundSummary `json:"rounds"`
}

type observedNavigationFullDemoDistributionAccumulator struct {
	directDistances   []float64
	pathDistances     []float64
	pathRatios        []float64
	observedSpeeds    []float64
	weaponSpeeds      []float64
	optimisticTimes   []float64
	weaponTimes       []float64
	observedPaceTimes []float64
}

type observedNavigationFullDemoRoundAccumulator struct {
	replayFrames   int
	eligibleFrames int
	counts         ObservedNavigationFullDemoCounts
}

type observedNavigationFullDemoLogicalWriter struct {
	destination io.Writer
	hasher      io.Writer
	bytes       int64
}

func (writer *observedNavigationFullDemoLogicalWriter) Write(payload []byte) (int, error) {
	written, err := writer.destination.Write(payload)
	if written > 0 {
		writer.bytes += int64(written)
		if _, hashErr := writer.hasher.Write(payload[:written]); hashErr != nil && err == nil {
			err = hashErr
		}
	}
	if err == nil && written != len(payload) {
		err = io.ErrShortWrite
	}
	return written, err
}

// BuildObservedNavigationFullDemoReport streams one pseudonymous candidate row
// per eligible replay frame in deterministic round/tick order and retains only
// compact aggregate evidence in memory.
func BuildObservedNavigationFullDemoReport(
	replay *models.ReplayData,
	graph *navigation.Graph,
	sourceAlias string,
	sourceSHA256 string,
	rowsWriter io.Writer,
) (ObservedNavigationFullDemoReport, error) {
	if replay == nil {
		return ObservedNavigationFullDemoReport{}, fmt.Errorf("build full navigation sweep: replay data is nil")
	}
	if rowsWriter == nil {
		return ObservedNavigationFullDemoReport{}, fmt.Errorf("build full navigation sweep: rows writer is nil")
	}
	if !validLowerSHA256(sourceSHA256) || sourceAlias != "demo-"+sourceSHA256[:20] {
		return ObservedNavigationFullDemoReport{}, fmt.Errorf("build full navigation sweep: source alias/checksum mismatch")
	}
	if strings.TrimSpace(replay.Metadata.MapName) == "" {
		return ObservedNavigationFullDemoReport{}, fmt.Errorf("build full navigation sweep: map name is empty")
	}
	if !finiteNavigation(replay.Metadata.TickRate) || replay.Metadata.TickRate <= 0 {
		return ObservedNavigationFullDemoReport{}, fmt.Errorf("build full navigation sweep: tick rate must be finite and positive")
	}
	if graph == nil || len(graph.Nodes) == 0 || len(graph.Edges) == 0 {
		return ObservedNavigationFullDemoReport{}, fmt.Errorf("build full navigation sweep: loaded NAV graph is unavailable or empty")
	}
	if graph.Version != navigation.GraphContractVersion || strings.TrimSpace(graph.AssetReference) == "" ||
		!validLowerSHA256(graph.AssetSHA256) {
		return ObservedNavigationFullDemoReport{}, fmt.Errorf("build full navigation sweep: NAV contract or provenance is invalid")
	}

	prepared, err := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	if err != nil {
		return ObservedNavigationFullDemoReport{}, fmt.Errorf("build full navigation sweep: prepare NAV graph: %w", err)
	}
	frames := eligibleNavigationSmokeFrames(replay)
	participantAliases := navigationSmokeParticipantAliases(frames)
	roundAccumulators, replayFrames, err := navigationFullDemoRoundCoverage(replay)
	if err != nil {
		return ObservedNavigationFullDemoReport{}, err
	}

	logicalHash := sha256.New()
	logicalWriter := &observedNavigationFullDemoLogicalWriter{
		destination: rowsWriter,
		hasher:      logicalHash,
	}
	encoder := json.NewEncoder(logicalWriter)
	encoder.SetEscapeHTML(false)

	globalCounts := newObservedNavigationFullDemoCounts()
	distributions := observedNavigationFullDemoDistributionAccumulator{}
	seenFrames := make(map[string]struct{}, len(frames))
	for _, frame := range frames {
		roundAccumulator, knownRound := roundAccumulators[frame.roundNumber]
		if !knownRound {
			return ObservedNavigationFullDemoReport{}, fmt.Errorf(
				"build full navigation sweep: eligible frame references unknown round %d",
				frame.roundNumber,
			)
		}
		frameKey := fmt.Sprintf("%03d:%d", frame.roundNumber, frame.tick)
		if _, duplicate := seenFrames[frameKey]; duplicate {
			return ObservedNavigationFullDemoReport{}, fmt.Errorf(
				"build full navigation sweep: duplicate eligible frame %s",
				frameKey,
			)
		}
		seenFrames[frameKey] = struct{}{}

		origin, target, ok := navigationSmokePair(frame.players)
		if !ok {
			return ObservedNavigationFullDemoReport{}, fmt.Errorf(
				"build full navigation sweep: eligible frame %s lost its deterministic pair",
				frameKey,
			)
		}
		originAlias, originKnown := participantAliases[origin.SteamID]
		targetAlias, targetKnown := participantAliases[target.SteamID]
		if !originKnown || !targetKnown || originAlias == targetAlias {
			return ObservedNavigationFullDemoReport{}, fmt.Errorf(
				"build full navigation sweep: participant alias is unavailable",
			)
		}
		roundID := fmt.Sprintf("%s:round:%03d", sourceAlias, frame.roundNumber)
		queryID := fmt.Sprintf("%s:path:%s:%s:%d", roundID, originAlias, targetAlias, frame.tick)
		request := models.ObservedNavigationPathRequest{
			JoinKeys: models.ObservedNavigationPathJoinKeys{
				MatchID:        sourceAlias,
				RoundID:        roundID,
				QueryID:        queryID,
				OriginPlayerID: originAlias,
				TargetPlayerID: targetAlias,
			},
			RoundNumber:            frame.roundNumber,
			T0Tick:                 frame.tick,
			OriginPosition:         replayPlayerNavigationPosition(origin),
			TargetPosition:         replayPlayerNavigationPosition(target),
			OriginAvailabilityTick: frame.tick,
			TargetAvailabilityTick: frame.tick,
			OriginMotion:           replayNavigationMotionInput(origin, frame.tick),
		}
		row, err := buildObservedNavigationPathWithPreparedEvaluator(request, graph, prepared)
		if err != nil {
			return ObservedNavigationFullDemoReport{}, fmt.Errorf(
				"build full navigation sweep: query %s: %w",
				queryID,
				err,
			)
		}
		if err := encoder.Encode(row); err != nil {
			return ObservedNavigationFullDemoReport{}, fmt.Errorf(
				"build full navigation sweep: write query %s: %w",
				queryID,
				err,
			)
		}

		roundAccumulator.eligibleFrames++
		accumulateObservedNavigationFullDemoRow(&globalCounts, &distributions, row)
		accumulateObservedNavigationFullDemoRow(&roundAccumulator.counts, nil, row)
	}

	rounds := make([]ObservedNavigationFullDemoRoundSummary, 0, len(roundAccumulators))
	roundNumbers := make([]int, 0, len(roundAccumulators))
	for roundNumber := range roundAccumulators {
		roundNumbers = append(roundNumbers, roundNumber)
	}
	sort.Ints(roundNumbers)
	for _, roundNumber := range roundNumbers {
		accumulator := roundAccumulators[roundNumber]
		rounds = append(rounds, ObservedNavigationFullDemoRoundSummary{
			RoundNumber:      roundNumber,
			ReplayFrames:     accumulator.replayFrames,
			EligibleFrames:   accumulator.eligibleFrames,
			IneligibleFrames: accumulator.replayFrames - accumulator.eligibleFrames,
			Counts:           accumulator.counts,
		})
	}

	report := ObservedNavigationFullDemoReport{
		SchemaID:                 ObservedNavigationFullDemoSchemaID,
		Status:                   "failed",
		SourceAlias:              sourceAlias,
		SourceSHA256:             sourceSHA256,
		MapName:                  replay.Metadata.MapName,
		Scope:                    ObservedNavigationFullDemoScope,
		SelectionContract:        ObservedNavigationFullDemoSelectionContract,
		PairExhaustive:           false,
		DirectionalPairs:         true,
		CandidateSchemaID:        models.ObservedNavigationPathSchemaID,
		CandidateContractVersion: models.ObservedNavigationPathContractVersion,
		GraphContractVersion:     graph.Version,
		GraphAssetReference:      graph.AssetReference,
		GraphAssetSHA256:         graph.AssetSHA256,
		GraphNodeCount:           len(graph.Nodes),
		GraphEdgeCount:           len(graph.Edges),
		RoundCount:               len(rounds),
		TickRate:                 replay.Metadata.TickRate,
		SampleStrideTicks:        observedNavigationFullDemoSampleStride(replay),
		DistributionContract:     ObservedNavigationFullDemoDistributionContract,
		TimingPolicy: ObservedNavigationFullDemoTimingPolicy{
			IntendedUse:                models.NavigationTimingIntendedUse,
			UserFacingMode:             models.NavigationTimingUserFacingMode,
			DecisionSemantics:          models.NavigationTimingDecisionSemantics,
			EstimateSemantics:          models.NavigationTimingScenarioNotAPrediction,
			ExactUserFacingTimeAllowed: false,
			ArrivalPrediction:          false,
		},
		RowsArtifact: ObservedNavigationFullDemoRowsArtifact{
			Format:         "jsonl",
			Compression:    "gzip",
			RowCount:       globalCounts.Queries,
			LogicalBytes:   logicalWriter.bytes,
			LogicalSHA256:  hex.EncodeToString(logicalHash.Sum(nil)),
			DigestContract: ObservedNavigationFullDemoRowsDigestContract,
		},
		Summary: ObservedNavigationFullDemoSummary{
			ReplayFrames:     replayFrames,
			EligibleFrames:   len(frames),
			IneligibleFrames: replayFrames - len(frames),
			Counts:           globalCounts,
			Distributions:    distributions.finish(),
		},
		Rounds: rounds,
	}
	if report.SampleStrideTicks > 0 {
		report.SampleIntervalMilliseconds = float64(report.SampleStrideTicks) * 1000 / report.TickRate
	}
	if err := validateObservedNavigationFullDemoReport(report, false); err != nil {
		return ObservedNavigationFullDemoReport{}, err
	}
	report.Status = "passed"
	return report, nil
}

func navigationFullDemoRoundCoverage(
	replay *models.ReplayData,
) (map[int]*observedNavigationFullDemoRoundAccumulator, int, error) {
	result := make(map[int]*observedNavigationFullDemoRoundAccumulator)
	totalFrames := 0
	for index, round := range replay.Rounds {
		if round.Round < 1 {
			return nil, 0, fmt.Errorf(
				"build full navigation sweep: invalid round number %d at index %d",
				round.Round,
				index,
			)
		}
		if round.StartTick < 0 || round.EndTick < round.StartTick {
			return nil, 0, fmt.Errorf(
				"build full navigation sweep: invalid tick bounds for round %d",
				round.Round,
			)
		}
		if _, duplicate := result[round.Round]; duplicate {
			return nil, 0, fmt.Errorf(
				"build full navigation sweep: duplicate round number %d",
				round.Round,
			)
		}
		seenTicks := make(map[int]struct{}, len(round.Frames))
		for _, frame := range round.Frames {
			if frame.Tick < round.StartTick || frame.Tick > round.EndTick {
				return nil, 0, fmt.Errorf(
					"build full navigation sweep: frame tick %d is outside round %d bounds [%d,%d]",
					frame.Tick,
					round.Round,
					round.StartTick,
					round.EndTick,
				)
			}
			if _, duplicate := seenTicks[frame.Tick]; duplicate {
				return nil, 0, fmt.Errorf(
					"build full navigation sweep: duplicate frame tick %d in round %d",
					frame.Tick,
					round.Round,
				)
			}
			seenTicks[frame.Tick] = struct{}{}
		}
		result[round.Round] = &observedNavigationFullDemoRoundAccumulator{
			replayFrames: len(round.Frames),
			counts:       newObservedNavigationFullDemoCounts(),
		}
		totalFrames += len(round.Frames)
	}
	return result, totalFrames, nil
}

func newObservedNavigationFullDemoCounts() ObservedNavigationFullDemoCounts {
	return ObservedNavigationFullDemoCounts{
		OptimisticBaseline: ObservedNavigationFullDemoScenarioCounts{
			UnavailabilityReasons: make(map[string]int),
		},
		ActiveWeaponBaseline: ObservedNavigationFullDemoScenarioCounts{
			UnavailabilityReasons: make(map[string]int),
		},
		ObservedPaceProjection: ObservedNavigationFullDemoScenarioCounts{
			UnavailabilityReasons: make(map[string]int),
		},
		ActiveWeaponContexts:        make(map[string]int),
		WeaponBaselineSpeedContexts: make(map[string]int),
		AbstentionReasons:           make(map[string]int),
	}
}

func accumulateObservedNavigationFullDemoRow(
	counts *ObservedNavigationFullDemoCounts,
	distributions *observedNavigationFullDemoDistributionAccumulator,
	row models.ObservedNavigationPath,
) {
	counts.Queries++
	switch {
	case row.Reachability.Status == models.NavigationFactDerived && row.Reachability.Value != nil && *row.Reachability.Value:
		counts.ReachableQueries++
	case row.Reachability.Status == models.NavigationFactDerived && row.Reachability.Value != nil && !*row.Reachability.Value:
		counts.DisconnectedQueries++
	default:
		counts.EndpointUnavailableQueries++
	}
	accumulateObservedNavigationFullDemoScenario(
		&counts.OptimisticBaseline,
		row.InternalTiming.Scenarios.OptimisticUninterrupted250UPSBaseline,
	)
	accumulateObservedNavigationFullDemoScenario(
		&counts.ActiveWeaponBaseline,
		row.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline,
	)
	accumulateObservedNavigationFullDemoScenario(
		&counts.ObservedPaceProjection,
		row.InternalTiming.Scenarios.ObservedPaceProjection,
	)

	if row.OriginMotion.IsWalking != nil && *row.OriginMotion.IsWalking {
		counts.WalkingContexts++
	}
	if row.OriginMotion.IsDucking != nil && *row.OriginMotion.IsDucking {
		counts.DuckingContexts++
	}
	if row.OriginMotion.IsScoped != nil && *row.OriginMotion.IsScoped {
		counts.ScopedContexts++
	}
	if row.OriginMotion.HorizontalSpeedStatus == models.NavigationFactDerived &&
		row.OriginMotion.HorizontalSpeedWorldUnitsPerSecond != nil {
		speed := *row.OriginMotion.HorizontalSpeedWorldUnitsPerSecond
		if speed > 0 {
			counts.ObservedMovingContexts++
			if row.OriginMotion.IsWalking != nil && !*row.OriginMotion.IsWalking {
				counts.NonWalkingMovingContexts++
			}
		} else {
			counts.ObservedStationaryContexts++
		}
	} else {
		counts.MotionUnavailableContexts++
	}
	if row.OriginMotion.ActiveWeaponStatus == models.NavigationFactObserved && row.OriginMotion.ActiveWeapon != nil {
		weapon := strings.TrimSpace(*row.OriginMotion.ActiveWeapon)
		counts.ActiveWeaponContexts[weapon]++
		if strings.EqualFold(weapon, "Knife") {
			counts.KnifeContexts++
		}
	}
	weaponScenario := row.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline
	if weaponScenario.Status == models.NavigationFactEstimated && weaponScenario.SpeedWorldUnitsPerSecond != nil {
		key := strconv.FormatFloat(*weaponScenario.SpeedWorldUnitsPerSecond, 'f', -1, 64)
		counts.WeaponBaselineSpeedContexts[key]++
	}
	if row.InternalTiming.ExactUserFacingTimeAllowed {
		counts.ExactUserFacingTimeViolations++
	}
	if row.InternalTiming.ArrivalPrediction {
		counts.ArrivalPredictionViolations++
	}
	for _, reason := range row.AbstentionReasons {
		counts.AbstentionReasons[reason]++
	}
	if distributions != nil {
		distributions.add(row)
	}
}

func accumulateObservedNavigationFullDemoScenario(
	counts *ObservedNavigationFullDemoScenarioCounts,
	scenario models.ObservedNavigationTimingScenario,
) {
	if scenario.Status == models.NavigationFactEstimated {
		counts.Estimated++
		return
	}
	counts.Unavailable++
	if scenario.UnavailabilityReason != nil {
		counts.UnavailabilityReasons[*scenario.UnavailabilityReason]++
	}
}

func (accumulator *observedNavigationFullDemoDistributionAccumulator) add(
	row models.ObservedNavigationPath,
) {
	if row.DirectDistance.ValueWorldUnits != nil {
		direct := *row.DirectDistance.ValueWorldUnits
		accumulator.directDistances = append(accumulator.directDistances, direct)
		if row.PathDistance.ValueWorldUnits != nil {
			path := *row.PathDistance.ValueWorldUnits
			accumulator.pathDistances = append(accumulator.pathDistances, path)
			if direct > 0 {
				accumulator.pathRatios = append(accumulator.pathRatios, path/direct)
			}
		}
	}
	if row.OriginMotion.HorizontalSpeedStatus == models.NavigationFactDerived &&
		row.OriginMotion.HorizontalSpeedWorldUnitsPerSecond != nil {
		accumulator.observedSpeeds = append(
			accumulator.observedSpeeds,
			*row.OriginMotion.HorizontalSpeedWorldUnitsPerSecond,
		)
	}
	accumulator.addScenario(
		row.InternalTiming.Scenarios.OptimisticUninterrupted250UPSBaseline,
		&accumulator.optimisticTimes,
		false,
	)
	accumulator.addScenario(
		row.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline,
		&accumulator.weaponTimes,
		true,
	)
	accumulator.addScenario(
		row.InternalTiming.Scenarios.ObservedPaceProjection,
		&accumulator.observedPaceTimes,
		false,
	)
}

func (accumulator *observedNavigationFullDemoDistributionAccumulator) addScenario(
	scenario models.ObservedNavigationTimingScenario,
	times *[]float64,
	retainSpeed bool,
) {
	if scenario.Status != models.NavigationFactEstimated || scenario.ValueMilliseconds == nil {
		return
	}
	*times = append(*times, *scenario.ValueMilliseconds)
	if retainSpeed && scenario.SpeedWorldUnitsPerSecond != nil {
		accumulator.weaponSpeeds = append(accumulator.weaponSpeeds, *scenario.SpeedWorldUnitsPerSecond)
	}
}

func (accumulator observedNavigationFullDemoDistributionAccumulator) finish() ObservedNavigationFullDemoDistributions {
	return ObservedNavigationFullDemoDistributions{
		DirectDistanceWorldUnits:                   observedNavigationFullDemoDistribution(accumulator.directDistances),
		PathDistanceWorldUnits:                     observedNavigationFullDemoDistribution(accumulator.pathDistances),
		PathToDirectDistanceRatio:                  observedNavigationFullDemoDistribution(accumulator.pathRatios),
		ObservedHorizontalSpeedWorldUnitsPerSecond: observedNavigationFullDemoDistribution(accumulator.observedSpeeds),
		WeaponBaselineSpeedWorldUnitsPerSecond:     observedNavigationFullDemoDistribution(accumulator.weaponSpeeds),
		OptimisticBaselineMilliseconds:             observedNavigationFullDemoDistribution(accumulator.optimisticTimes),
		ActiveWeaponBaselineMilliseconds:           observedNavigationFullDemoDistribution(accumulator.weaponTimes),
		ObservedPaceProjectionMilliseconds:         observedNavigationFullDemoDistribution(accumulator.observedPaceTimes),
	}
}

func observedNavigationFullDemoDistribution(
	values []float64,
) ObservedNavigationFullDemoNumericDistribution {
	if len(values) == 0 {
		return ObservedNavigationFullDemoNumericDistribution{}
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	p50 := ordered[observedNavigationFullDemoNearestRank(len(ordered), 0.50)]
	p95 := ordered[observedNavigationFullDemoNearestRank(len(ordered), 0.95)]
	maximum := ordered[len(ordered)-1]
	return ObservedNavigationFullDemoNumericDistribution{
		Count: len(ordered),
		P50:   float64PointerNavigation(p50),
		P95:   float64PointerNavigation(p95),
		Max:   float64PointerNavigation(maximum),
	}
}

func observedNavigationFullDemoNearestRank(length int, percentile float64) int {
	index := int(math.Ceil(percentile*float64(length))) - 1
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func observedNavigationFullDemoSampleStride(replay *models.ReplayData) int {
	rounds := append([]models.ReplayRound(nil), replay.Rounds...)
	sort.Slice(rounds, func(i, j int) bool {
		if rounds[i].Round != rounds[j].Round {
			return rounds[i].Round < rounds[j].Round
		}
		return rounds[i].StartTick < rounds[j].StartTick
	})
	stride := 0
	for _, round := range rounds {
		frames := append([]models.ReplayFrame(nil), round.Frames...)
		sort.SliceStable(frames, func(i, j int) bool { return frames[i].Tick < frames[j].Tick })
		for index := 1; index < len(frames); index++ {
			if delta := frames[index].Tick - frames[index-1].Tick; delta > 0 {
				stride = greatestCommonDivisorObservedNavigation(stride, delta)
			}
		}
	}
	return stride
}

func greatestCommonDivisorObservedNavigation(left, right int) int {
	for right != 0 {
		left, right = right, left%right
	}
	if left < 0 {
		return -left
	}
	return left
}

// ValidateObservedNavigationFullDemoReport validates the completed compact
// report after the caller has closed, sized, and hashed the gzip sidecar.
func ValidateObservedNavigationFullDemoReport(report ObservedNavigationFullDemoReport) error {
	return validateObservedNavigationFullDemoReport(report, true)
}

func validateObservedNavigationFullDemoReport(
	report ObservedNavigationFullDemoReport,
	requirePhysicalArtifact bool,
) error {
	if report.SchemaID != ObservedNavigationFullDemoSchemaID ||
		!validLowerSHA256(report.SourceSHA256) ||
		report.SourceAlias != "demo-"+report.SourceSHA256[:20] ||
		report.Scope != ObservedNavigationFullDemoScope ||
		report.SelectionContract != ObservedNavigationFullDemoSelectionContract ||
		report.PairExhaustive || !report.DirectionalPairs ||
		report.CandidateSchemaID != models.ObservedNavigationPathSchemaID ||
		report.CandidateContractVersion != models.ObservedNavigationPathContractVersion ||
		report.GraphContractVersion != navigation.GraphContractVersion ||
		report.DistributionContract != ObservedNavigationFullDemoDistributionContract ||
		report.TimingPolicy.IntendedUse != models.NavigationTimingIntendedUse ||
		report.TimingPolicy.UserFacingMode != models.NavigationTimingUserFacingMode ||
		report.TimingPolicy.DecisionSemantics != models.NavigationTimingDecisionSemantics ||
		report.TimingPolicy.EstimateSemantics != models.NavigationTimingScenarioNotAPrediction ||
		report.TimingPolicy.ExactUserFacingTimeAllowed || report.TimingPolicy.ArrivalPrediction {
		return fmt.Errorf("build full navigation sweep: report contract metadata is invalid")
	}
	if strings.TrimSpace(report.MapName) == "" || strings.TrimSpace(report.GraphAssetReference) == "" ||
		!validLowerSHA256(report.GraphAssetSHA256) || report.GraphNodeCount < 1 || report.GraphEdgeCount < 1 ||
		!finiteNavigation(report.TickRate) || report.TickRate <= 0 || report.SampleStrideTicks < 1 ||
		!finiteNavigation(report.SampleIntervalMilliseconds) || report.SampleIntervalMilliseconds <= 0 {
		return fmt.Errorf("build full navigation sweep: report provenance or sampling metadata is invalid")
	}
	if report.RowsArtifact.Format != "jsonl" || report.RowsArtifact.Compression != "gzip" ||
		report.RowsArtifact.DigestContract != ObservedNavigationFullDemoRowsDigestContract {
		return fmt.Errorf("build full navigation sweep: rows artifact contract is invalid")
	}
	if requirePhysicalArtifact {
		if report.Status != "passed" || strings.TrimSpace(report.BuildIdentifier) == "" ||
			report.ProcessingDurationMS < 0 || report.NavigationEvaluationDurationMS < 0 ||
			report.RowsArtifact.Path != ObservedNavigationFullDemoRowsFilename ||
			report.RowsArtifact.Bytes < 1 || !validLowerSHA256(report.RowsArtifact.SHA256) {
			return fmt.Errorf("build full navigation sweep: completed physical artifact metadata is invalid")
		}
	} else if report.Status != "failed" || report.RowsArtifact.Path != "" ||
		report.RowsArtifact.Bytes != 0 || report.RowsArtifact.SHA256 != "" {
		return fmt.Errorf("build full navigation sweep: pending physical artifact metadata is invalid")
	}
	if report.Summary.ReplayFrames < 1 || report.Summary.EligibleFrames < 1 ||
		report.Summary.IneligibleFrames != report.Summary.ReplayFrames-report.Summary.EligibleFrames {
		return fmt.Errorf("build full navigation sweep: replay frame coverage is invalid")
	}
	if report.Summary.Counts.Queries != report.Summary.EligibleFrames ||
		report.RowsArtifact.RowCount != report.Summary.Counts.Queries {
		return fmt.Errorf("build full navigation sweep: one query per eligible frame is required")
	}
	if report.Summary.Counts.ReachableQueries+report.Summary.Counts.DisconnectedQueries+
		report.Summary.Counts.EndpointUnavailableQueries != report.Summary.Counts.Queries {
		return fmt.Errorf("build full navigation sweep: reachability categories do not cover all queries")
	}
	if report.Summary.Counts.ReachableQueries == 0 ||
		report.Summary.Counts.OptimisticBaseline.Estimated != report.Summary.Counts.ReachableQueries ||
		report.Summary.Counts.ActiveWeaponBaseline.Estimated == 0 ||
		report.Summary.Counts.ObservedPaceProjection.Estimated == 0 {
		return fmt.Errorf("build full navigation sweep: real navigation or timing coverage is empty")
	}
	if report.Summary.Counts.ExactUserFacingTimeViolations != 0 ||
		report.Summary.Counts.ArrivalPredictionViolations != 0 {
		return fmt.Errorf("build full navigation sweep: exact user-facing timing policy was violated")
	}
	for _, scenarios := range []ObservedNavigationFullDemoScenarioCounts{
		report.Summary.Counts.OptimisticBaseline,
		report.Summary.Counts.ActiveWeaponBaseline,
		report.Summary.Counts.ObservedPaceProjection,
	} {
		if scenarios.Estimated+scenarios.Unavailable != report.Summary.Counts.Queries {
			return fmt.Errorf("build full navigation sweep: timing scenario counts are incomplete")
		}
	}
	if report.RowsArtifact.LogicalBytes < 1 || !validLowerSHA256(report.RowsArtifact.LogicalSHA256) {
		return fmt.Errorf("build full navigation sweep: logical rows digest is invalid")
	}
	if report.RoundCount != len(report.Rounds) || report.RoundCount < 1 {
		return fmt.Errorf("build full navigation sweep: round coverage is invalid")
	}
	frames, eligible, queries := 0, 0, 0
	for index, round := range report.Rounds {
		if round.RoundNumber < 1 || (index > 0 && report.Rounds[index-1].RoundNumber >= round.RoundNumber) {
			return fmt.Errorf("build full navigation sweep: rounds are not strictly ordered")
		}
		if round.EligibleFrames+round.IneligibleFrames != round.ReplayFrames ||
			round.Counts.Queries != round.EligibleFrames ||
			round.Counts.ReachableQueries+round.Counts.DisconnectedQueries+
				round.Counts.EndpointUnavailableQueries != round.Counts.Queries {
			return fmt.Errorf("build full navigation sweep: round %d coverage is inconsistent", round.RoundNumber)
		}
		frames += round.ReplayFrames
		eligible += round.EligibleFrames
		queries += round.Counts.Queries
	}
	if frames != report.Summary.ReplayFrames || eligible != report.Summary.EligibleFrames ||
		queries != report.Summary.Counts.Queries {
		return fmt.Errorf("build full navigation sweep: round totals differ from global totals")
	}
	return nil
}
