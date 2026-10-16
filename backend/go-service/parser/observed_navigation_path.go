package parser

import (
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/navigation"
)

const (
	navigationSourceArtifact         = "replay_positions_motion_context_and_static_nav_in_memory"
	navigationAssetLoaded            = "loaded"
	navigationAssetMissing           = "unavailable"
	navigationTimingScenarioContract = "navigation_internal_timing_scenarios@1"
	navigationIdealSpeedSource       = "optimistic_uninterrupted_250ups_assumption"
	navigationWeaponSpeedSource      = "active_weapon_unscoped_max_speed_table"
	navigationObservedPaceSource     = "causal_horizontal_speed_at_t0"
	navigationIdealAssumption        = "constant_250ups_without_stops_combat_angle_clearing_acceleration_or_equipment_changes"
	navigationWeaponAssumption       = "constant_active_weapon_unscoped_max_speed_without_stops_combat_angle_clearing_acceleration_or_equipment_changes"
	navigationObservedPaceAssumption = "constant_t0_horizontal_pace_without_predicting_future_movement_or_interruptions"
)

var observedNavigationInputFields = []string{
	"active_weapon_at_t0",
	"horizontal_speed_at_t0",
	"is_ducking_at_t0",
	"is_scoped_at_t0",
	"is_walking_at_t0",
	"nav_area_connections",
	"origin_availability_tick",
	"origin_position",
	"target_availability_tick",
	"target_position",
}

// BuildObservedNavigationPath derives factual position-to-position navigation
// measurements and explicit internal timing scenarios using only values
// available at T0. None of the exact times is an arrival prediction or
// permitted user-facing copy. The result remains outside canonical bundle 3.8.0.
func BuildObservedNavigationPath(
	request models.ObservedNavigationPathRequest,
	graph *navigation.Graph,
) (models.ObservedNavigationPath, error) {
	return buildObservedNavigationPathWithPreparedEvaluator(request, graph, nil)
}

// buildObservedNavigationPathWithPreparedEvaluator preserves the public
// producer contract while allowing a full replay sweep to reuse one immutable
// NAV index and its directed shortest-path cache.
func buildObservedNavigationPathWithPreparedEvaluator(
	request models.ObservedNavigationPathRequest,
	graph *navigation.Graph,
	evaluator *navigation.PreparedEvaluator,
) (models.ObservedNavigationPath, error) {
	if err := validateNavigationRequest(request); err != nil {
		return models.ObservedNavigationPath{}, err
	}
	if err := validateObservedNavigationGraph(request, graph); err != nil {
		return models.ObservedNavigationPath{}, err
	}

	config := navigation.DefaultConfig()
	if evaluator == nil {
		var err error
		evaluator, err = navigation.NewPreparedEvaluator(graph, config)
		if err != nil {
			return models.ObservedNavigationPath{}, fmt.Errorf("build observed navigation path: %w", err)
		}
	} else if evaluator.Config() != config {
		return models.ObservedNavigationPath{}, fmt.Errorf(
			"build observed navigation path: prepared evaluator config differs from the contract",
		)
	}
	measurement, err := evaluator.Evaluate(request.OriginPosition, request.TargetPosition)
	if err != nil {
		return models.ObservedNavigationPath{}, fmt.Errorf("build observed navigation path: %w", err)
	}

	observedPaceSource := navigationObservedPaceSource
	if request.OriginMotion != nil && request.OriginMotion.HorizontalSpeedWorldUnitsPerSecond != nil {
		observedPaceSource += ":" + request.OriginMotion.HorizontalSpeedSource
	}
	result := models.ObservedNavigationPath{
		SchemaID:          models.ObservedNavigationPathSchemaID,
		ContractVersion:   models.ObservedNavigationPathContractVersion,
		IdentitySemantics: models.TacticalIdentityJoinOnly,
		CausalRole:        models.ObservedNavigationPathCausalRole,
		Source:            models.ObservedNavigationPathSource,
		JoinKeys:          request.JoinKeys,
		RoundNumber:       request.RoundNumber,
		T0Tick:            request.T0Tick,
		Availability: models.ObservedNavigationAvailability{
			OriginTick: request.OriginAvailabilityTick,
			TargetTick: request.TargetAvailabilityTick,
		},
		OriginMotion: motionContextFromNavigationRequest(request.OriginMotion),
		DirectDistance: models.ObservedNavigationDistanceFact{
			Status:          models.NavigationFactDerived,
			ValueWorldUnits: float64PointerNavigation(measurement.DirectDistance),
		},
		Reachability: models.ObservedNavigationBooleanFact{
			Status: models.NavigationFactUnavailable,
		},
		PathDistance: models.ObservedNavigationDistanceFact{
			Status: models.NavigationFactUnavailable,
		},
		InternalTiming: models.ObservedNavigationInternalTiming{
			IntendedUse:                models.NavigationTimingIntendedUse,
			UserFacingMode:             models.NavigationTimingUserFacingMode,
			DecisionSemantics:          models.NavigationTimingDecisionSemantics,
			EstimateSemantics:          models.NavigationTimingScenarioNotAPrediction,
			ExactUserFacingTimeAllowed: false,
			ArrivalPrediction:          false,
			Scenarios: models.ObservedNavigationTimingScenarios{
				OptimisticUninterrupted250UPSBaseline: unavailableNavigationTimingScenario(
					navigationIdealSpeedSource,
					navigationIdealAssumption,
					"path_distance_unavailable",
				),
				ActiveWeaponUninterruptedBaseline: unavailableNavigationTimingScenario(
					navigationWeaponSpeedSource,
					navigationWeaponAssumption,
					"path_distance_unavailable",
				),
				ObservedPaceProjection: unavailableNavigationTimingScenario(
					observedPaceSource,
					navigationObservedPaceAssumption,
					"path_distance_unavailable",
				),
			},
		},
		AbstentionReasons: []string{},
		Provenance: models.ObservedNavigationProvenance{
			SourceArtifact:                      navigationSourceArtifact,
			GraphContractVersion:                navigation.GraphContractVersion,
			ShortestPathAlgorithm:               navigation.ShortestPathAlgorithm,
			EndpointAssociationMethod:           navigation.EndpointAssociationMethod,
			EndpointToleranceWorldUnits:         config.EndpointTolerance,
			EndpointVerticalToleranceWorldUnits: config.EndpointVerticalTolerance,
			EdgeWeightMethod:                    navigation.EdgeWeightMethod,
			TimingScenarioContract:              navigationTimingScenarioContract,
			WeaponSpeedTableContract:            models.WeaponMaxSpeedContractVersion,
			OriginAssociationWorldUnits:         measurement.OriginAssociationDistance,
			TargetAssociationWorldUnits:         measurement.TargetAssociationDistance,
			GraphAsset: models.ObservedNavigationAsset{
				Status: navigationAssetMissing,
			},
			InputFields: append([]string(nil), observedNavigationInputFields...),
		},
	}

	if graph != nil && len(graph.Nodes) > 0 {
		assetReference := graph.AssetReference
		result.Availability.GraphTick = intPointerNavigation(graph.AvailabilityTick)
		result.Provenance.GraphAsset.Status = navigationAssetLoaded
		result.Provenance.GraphAsset.Reference = &assetReference
		assetSHA := graph.AssetSHA256
		result.Provenance.GraphAsset.SHA256 = &assetSHA
	}
	if measurement.Reachable != nil {
		result.Reachability.Status = models.NavigationFactDerived
		result.Reachability.Value = measurement.Reachable
	}
	if measurement.PathDistance != nil {
		pathDistance := *measurement.PathDistance
		result.PathDistance.Status = models.NavigationFactDerived
		result.PathDistance.ValueWorldUnits = measurement.PathDistance
		result.InternalTiming.Scenarios = navigationTimingScenarios(pathDistance, request.OriginMotion)
	}
	if measurement.UnavailableReason != "" {
		result.AbstentionReasons = append(result.AbstentionReasons, measurement.UnavailableReason)
	}
	sort.Strings(result.AbstentionReasons)

	if err := validateObservedNavigationPath(result); err != nil {
		return models.ObservedNavigationPath{}, fmt.Errorf("build observed navigation path: %w", err)
	}
	return result, nil
}

func validateObservedNavigationGraph(
	request models.ObservedNavigationPathRequest,
	graph *navigation.Graph,
) error {
	if graph == nil {
		return nil
	}
	if graph.Version != navigation.GraphContractVersion {
		return fmt.Errorf(
			"build observed navigation path: graph version must be %s",
			navigation.GraphContractVersion,
		)
	}
	if graph.AvailabilityTick < 0 || graph.AvailabilityTick > request.T0Tick {
		return fmt.Errorf(
			"build observed navigation path: graph availability_tick must be between 0 and t0_tick",
		)
	}
	if strings.TrimSpace(graph.AssetReference) == "" {
		return fmt.Errorf(
			"build observed navigation path: loaded graph asset reference is empty",
		)
	}
	decoded, err := hex.DecodeString(graph.AssetSHA256)
	if err != nil || len(decoded) != 32 || graph.AssetSHA256 != strings.ToLower(graph.AssetSHA256) {
		return fmt.Errorf(
			"build observed navigation path: loaded graph asset sha256 must be 64 lowercase hexadecimal characters",
		)
	}
	return nil
}

func navigationTimingScenarios(
	pathDistance float64,
	motion *models.ObservedNavigationMotionInput,
) models.ObservedNavigationTimingScenarios {
	result := models.ObservedNavigationTimingScenarios{
		OptimisticUninterrupted250UPSBaseline: estimatedNavigationTimingScenario(
			pathDistance,
			models.NavigationIdealSpeedWorldUnitsPerSecond,
			navigationIdealSpeedSource,
			navigationIdealAssumption,
		),
		ActiveWeaponUninterruptedBaseline: unavailableNavigationTimingScenario(
			navigationWeaponSpeedSource,
			navigationWeaponAssumption,
			"active_weapon_unavailable",
		),
		ObservedPaceProjection: unavailableNavigationTimingScenario(
			navigationObservedPaceSource,
			navigationObservedPaceAssumption,
			"observed_horizontal_speed_unavailable",
		),
	}
	if motion == nil {
		return result
	}

	if motion.ActiveWeapon != nil {
		if speed, ok := models.LookupWeaponMaxSpeed(*motion.ActiveWeapon); ok {
			result.ActiveWeaponUninterruptedBaseline = estimatedNavigationTimingScenario(
				pathDistance,
				speed,
				navigationWeaponSpeedSource,
				navigationWeaponAssumption,
			)
		} else {
			result.ActiveWeaponUninterruptedBaseline = unavailableNavigationTimingScenario(
				navigationWeaponSpeedSource,
				navigationWeaponAssumption,
				"active_weapon_speed_unknown",
			)
		}
	}
	if motion.HorizontalSpeedWorldUnitsPerSecond != nil {
		speed := *motion.HorizontalSpeedWorldUnitsPerSecond
		if speed > 0 {
			result.ObservedPaceProjection = estimatedNavigationTimingScenario(
				pathDistance,
				speed,
				navigationObservedPaceSource+":"+motion.HorizontalSpeedSource,
				navigationObservedPaceAssumption,
			)
		} else {
			result.ObservedPaceProjection = unavailableNavigationTimingScenario(
				navigationObservedPaceSource+":"+motion.HorizontalSpeedSource,
				navigationObservedPaceAssumption,
				"observed_stationary_not_projectable",
			)
		}
	}
	return result
}

func estimatedNavigationTimingScenario(
	pathDistance, speed float64,
	source, assumption string,
) models.ObservedNavigationTimingScenario {
	milliseconds := pathDistance / speed * 1000
	return models.ObservedNavigationTimingScenario{
		Status:                   models.NavigationFactEstimated,
		ValueMilliseconds:        float64PointerNavigation(milliseconds),
		SpeedWorldUnitsPerSecond: float64PointerNavigation(speed),
		SpeedSource:              source,
		Assumption:               assumption,
	}
}

func unavailableNavigationTimingScenario(
	source, assumption, reason string,
) models.ObservedNavigationTimingScenario {
	return models.ObservedNavigationTimingScenario{
		Status:               models.NavigationFactUnavailable,
		SpeedSource:          source,
		Assumption:           assumption,
		UnavailabilityReason: stringPointerNavigation(reason),
	}
}

func motionContextFromNavigationRequest(
	motion *models.ObservedNavigationMotionInput,
) models.ObservedNavigationMotionContext {
	result := models.ObservedNavigationMotionContext{
		HorizontalSpeedStatus: models.NavigationFactUnavailable,
		ActiveWeaponStatus:    models.NavigationFactUnavailable,
	}
	if motion == nil {
		return result
	}
	result.AvailabilityTick = intPointerNavigation(motion.AvailabilityTick)
	result.IsWalking = cloneBoolPointerNavigation(motion.IsWalking)
	result.IsDucking = cloneBoolPointerNavigation(motion.IsDucking)
	result.IsScoped = cloneBoolPointerNavigation(motion.IsScoped)
	if motion.HorizontalSpeedWorldUnitsPerSecond != nil {
		result.HorizontalSpeedStatus = models.NavigationFactDerived
		result.HorizontalSpeedWorldUnitsPerSecond = float64PointerNavigation(
			*motion.HorizontalSpeedWorldUnitsPerSecond,
		)
		result.HorizontalSpeedSource = stringPointerNavigation(motion.HorizontalSpeedSource)
	}
	if motion.ActiveWeapon != nil {
		result.ActiveWeaponStatus = models.NavigationFactObserved
		result.ActiveWeapon = stringPointerNavigation(strings.TrimSpace(*motion.ActiveWeapon))
	}
	return result
}

func validateNavigationRequest(request models.ObservedNavigationPathRequest) error {
	joinValues := []struct {
		name  string
		value string
	}{
		{name: "match_id", value: request.JoinKeys.MatchID},
		{name: "round_id", value: request.JoinKeys.RoundID},
		{name: "query_id", value: request.JoinKeys.QueryID},
		{name: "origin_player_id", value: request.JoinKeys.OriginPlayerID},
		{name: "target_player_id", value: request.JoinKeys.TargetPlayerID},
	}
	for _, join := range joinValues {
		name, value := join.name, join.value
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("build observed navigation path: join_keys.%s is empty", name)
		}
	}
	if request.RoundNumber < 1 {
		return fmt.Errorf("build observed navigation path: round_number must be positive")
	}
	if request.T0Tick < 0 {
		return fmt.Errorf("build observed navigation path: t0_tick must be non-negative")
	}
	availabilityValues := []struct {
		name string
		tick int
	}{
		{name: "origin", tick: request.OriginAvailabilityTick},
		{name: "target", tick: request.TargetAvailabilityTick},
	}
	for _, availability := range availabilityValues {
		name, tick := availability.name, availability.tick
		if tick < 0 || tick > request.T0Tick {
			return fmt.Errorf(
				"build observed navigation path: %s availability_tick must be between 0 and t0_tick",
				name,
			)
		}
	}
	if request.OriginMotion == nil {
		return nil
	}
	motion := request.OriginMotion
	if motion.AvailabilityTick < 0 || motion.AvailabilityTick > request.T0Tick {
		return fmt.Errorf(
			"build observed navigation path: motion availability_tick must be between 0 and t0_tick",
		)
	}
	if motion.HorizontalSpeedWorldUnitsPerSecond == nil {
		if strings.TrimSpace(motion.HorizontalSpeedSource) != "" {
			return fmt.Errorf("build observed navigation path: speed source requires horizontal speed")
		}
	} else {
		speed := *motion.HorizontalSpeedWorldUnitsPerSecond
		if !finiteNavigation(speed) || speed < 0 {
			return fmt.Errorf("build observed navigation path: horizontal speed must be finite and non-negative")
		}
		if strings.TrimSpace(motion.HorizontalSpeedSource) == "" {
			return fmt.Errorf("build observed navigation path: horizontal speed source is empty")
		}
	}
	if motion.ActiveWeapon != nil && strings.TrimSpace(*motion.ActiveWeapon) == "" {
		return fmt.Errorf("build observed navigation path: active weapon cannot be empty")
	}
	return nil
}

func validateObservedNavigationPath(result models.ObservedNavigationPath) error {
	if result.Availability.OriginTick > result.T0Tick || result.Availability.TargetTick > result.T0Tick {
		return fmt.Errorf("endpoint availability must be <= t0_tick")
	}
	if result.Availability.GraphTick != nil && *result.Availability.GraphTick > result.T0Tick {
		return fmt.Errorf("graph availability must be <= t0_tick")
	}
	if result.OriginMotion.AvailabilityTick != nil && *result.OriginMotion.AvailabilityTick > result.T0Tick {
		return fmt.Errorf("motion availability must be <= t0_tick")
	}
	if err := validateNavigationMotionContext(result.OriginMotion); err != nil {
		return err
	}
	if err := validateNavigationDistanceFact(result.DirectDistance, "direct_distance"); err != nil {
		return err
	}
	if err := validateNavigationDistanceFact(result.PathDistance, "path_distance"); err != nil {
		return err
	}
	if err := validateNavigationBooleanFact(result.Reachability); err != nil {
		return err
	}
	if result.InternalTiming.IntendedUse != models.NavigationTimingIntendedUse ||
		result.InternalTiming.UserFacingMode != models.NavigationTimingUserFacingMode ||
		result.InternalTiming.DecisionSemantics != models.NavigationTimingDecisionSemantics ||
		result.InternalTiming.EstimateSemantics != models.NavigationTimingScenarioNotAPrediction ||
		result.InternalTiming.ExactUserFacingTimeAllowed || result.InternalTiming.ArrivalPrediction {
		return fmt.Errorf("internal timing communication policy is invalid")
	}
	scenarios := []models.ObservedNavigationTimingScenario{
		result.InternalTiming.Scenarios.OptimisticUninterrupted250UPSBaseline,
		result.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline,
		result.InternalTiming.Scenarios.ObservedPaceProjection,
	}
	for _, scenario := range scenarios {
		if err := validateNavigationTimingScenario(scenario); err != nil {
			return err
		}
	}
	if result.Reachability.Value == nil || !*result.Reachability.Value {
		if result.PathDistance.ValueWorldUnits != nil {
			return fmt.Errorf("unreachable or unavailable result cannot publish path distance")
		}
		for _, scenario := range scenarios {
			if scenario.Status != models.NavigationFactUnavailable {
				return fmt.Errorf("unreachable or unavailable result cannot publish timing scenarios")
			}
		}
	} else if result.InternalTiming.Scenarios.OptimisticUninterrupted250UPSBaseline.Status != models.NavigationFactEstimated {
		return fmt.Errorf("reachable result must publish the internal optimistic baseline")
	}
	return nil
}

func validateNavigationMotionContext(value models.ObservedNavigationMotionContext) error {
	if value.HorizontalSpeedStatus == models.NavigationFactUnavailable {
		if value.HorizontalSpeedWorldUnitsPerSecond != nil || value.HorizontalSpeedSource != nil {
			return fmt.Errorf("unavailable horizontal speed must be null")
		}
	} else if value.HorizontalSpeedStatus != models.NavigationFactDerived ||
		value.HorizontalSpeedWorldUnitsPerSecond == nil ||
		!finiteNavigation(*value.HorizontalSpeedWorldUnitsPerSecond) ||
		*value.HorizontalSpeedWorldUnitsPerSecond < 0 ||
		value.HorizontalSpeedSource == nil || strings.TrimSpace(*value.HorizontalSpeedSource) == "" {
		return fmt.Errorf("derived horizontal speed is invalid")
	}
	if value.ActiveWeaponStatus == models.NavigationFactUnavailable {
		if value.ActiveWeapon != nil {
			return fmt.Errorf("unavailable active weapon must be null")
		}
	} else if value.ActiveWeaponStatus != models.NavigationFactObserved ||
		value.ActiveWeapon == nil || strings.TrimSpace(*value.ActiveWeapon) == "" {
		return fmt.Errorf("observed active weapon is invalid")
	}
	return nil
}

func validateNavigationTimingScenario(value models.ObservedNavigationTimingScenario) error {
	if strings.TrimSpace(value.SpeedSource) == "" || strings.TrimSpace(value.Assumption) == "" {
		return fmt.Errorf("timing scenario provenance is incomplete")
	}
	if value.Status == models.NavigationFactUnavailable {
		if value.ValueMilliseconds != nil || value.SpeedWorldUnitsPerSecond != nil ||
			value.UnavailabilityReason == nil || strings.TrimSpace(*value.UnavailabilityReason) == "" {
			return fmt.Errorf("unavailable timing scenario must contain only a reason")
		}
		return nil
	}
	if value.Status != models.NavigationFactEstimated || value.ValueMilliseconds == nil ||
		!finiteNavigation(*value.ValueMilliseconds) || *value.ValueMilliseconds < 0 ||
		value.SpeedWorldUnitsPerSecond == nil ||
		!finiteNavigation(*value.SpeedWorldUnitsPerSecond) || *value.SpeedWorldUnitsPerSecond <= 0 ||
		value.UnavailabilityReason != nil {
		return fmt.Errorf("estimated timing scenario must contain finite non-negative time and positive speed")
	}
	return nil
}

func validateNavigationDistanceFact(value models.ObservedNavigationDistanceFact, name string) error {
	if value.Status == models.NavigationFactUnavailable {
		if value.ValueWorldUnits != nil {
			return fmt.Errorf("unavailable %s must be null", name)
		}
		return nil
	}
	if value.Status != models.NavigationFactDerived || value.ValueWorldUnits == nil ||
		!finiteNavigation(*value.ValueWorldUnits) || *value.ValueWorldUnits < 0 {
		return fmt.Errorf("%s must be finite and non-negative when derived", name)
	}
	return nil
}

func validateNavigationBooleanFact(value models.ObservedNavigationBooleanFact) error {
	if value.Status == models.NavigationFactUnavailable {
		if value.Value != nil {
			return fmt.Errorf("unavailable reachability must be null")
		}
		return nil
	}
	if value.Status != models.NavigationFactDerived || value.Value == nil {
		return fmt.Errorf("reachability must contain a boolean when derived")
	}
	return nil
}

func finiteNavigation(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func cloneBoolPointerNavigation(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func float64PointerNavigation(value float64) *float64 { return &value }
func intPointerNavigation(value int) *int             { return &value }
func stringPointerNavigation(value string) *string    { return &value }
