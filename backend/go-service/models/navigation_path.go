package models

import "github.com/golang/geo/r3"

const (
	ObservedNavigationPathSchemaID        = "stratai.observed_navigation_path@2"
	ObservedNavigationPathContractVersion = "2.0.0"
	ObservedNavigationPathCausalRole      = "model_input_observation"
	ObservedNavigationPathSource          = "replay_position_pair_navigation_derivation"

	NavigationFactDerived     = "derived"
	NavigationFactEstimated   = "estimated"
	NavigationFactObserved    = "observed"
	NavigationFactUnavailable = "unavailable"

	NavigationTimingIntendedUse             = "internal_decision_support_only"
	NavigationTimingUserFacingMode          = "qualitative_conditional_with_uncertainty_only"
	NavigationTimingDecisionSemantics       = "constraint_signal_not_recommendation"
	NavigationTimingScenarioNotAPrediction  = "scenario_reference_not_arrival_prediction"
	NavigationIdealSpeedWorldUnitsPerSecond = 250.0
)

// ObservedNavigationPathRequest contains only causal inputs available at T0.
// It has no future, oracle, intent, outcome, recommendation, or tactical-label
// fields by construction.
type ObservedNavigationPathRequest struct {
	JoinKeys               ObservedNavigationPathJoinKeys
	RoundNumber            int
	T0Tick                 int
	OriginPosition         r3.Vector
	TargetPosition         r3.Vector
	OriginAvailabilityTick int
	TargetAvailabilityTick int
	OriginMotion           *ObservedNavigationMotionInput
}

// ObservedNavigationMotionInput is an optional T0 snapshot. Horizontal speed
// may be derived causally from an earlier position, while weapon and movement
// flags are direct observations at AvailabilityTick. Nil values remain
// unavailable; callers must never substitute defaults.
type ObservedNavigationMotionInput struct {
	AvailabilityTick                   int
	HorizontalSpeedWorldUnitsPerSecond *float64
	HorizontalSpeedSource              string
	ActiveWeapon                       *string
	IsWalking                          *bool
	IsDucking                          *bool
	IsScoped                           *bool
}

// ObservedNavigationPathJoinKeys isolate identifiers from every factual value.
type ObservedNavigationPathJoinKeys struct {
	MatchID        string `json:"match_id"`
	RoundID        string `json:"round_id"`
	QueryID        string `json:"query_id"`
	OriginPlayerID string `json:"origin_player_id"`
	TargetPlayerID string `json:"target_player_id"`
}

type ObservedNavigationAvailability struct {
	OriginTick int  `json:"origin_tick"`
	TargetTick int  `json:"target_tick"`
	GraphTick  *int `json:"graph_tick"`
}

type ObservedNavigationDistanceFact struct {
	Status          string   `json:"status"`
	ValueWorldUnits *float64 `json:"value_world_units"`
}

type ObservedNavigationBooleanFact struct {
	Status string `json:"status"`
	Value  *bool  `json:"value"`
}

// ObservedNavigationMotionContext makes the movement assumptions inspectable.
// Walking, ducking and scoped are context only: no undocumented multiplier is
// applied to them, and observed speed already reflects their current effect.
type ObservedNavigationMotionContext struct {
	AvailabilityTick                   *int     `json:"availability_tick"`
	HorizontalSpeedStatus              string   `json:"horizontal_speed_status"`
	HorizontalSpeedWorldUnitsPerSecond *float64 `json:"horizontal_speed_world_units_per_second"`
	HorizontalSpeedSource              *string  `json:"horizontal_speed_source"`
	ActiveWeaponStatus                 string   `json:"active_weapon_status"`
	ActiveWeapon                       *string  `json:"active_weapon"`
	IsWalking                          *bool    `json:"is_walking"`
	IsDucking                          *bool    `json:"is_ducking"`
	IsScoped                           *bool    `json:"is_scoped"`
}

// ObservedNavigationTimingScenario is an exact internal calculation under one
// explicit assumption. It is not a prediction of when the player will arrive.
type ObservedNavigationTimingScenario struct {
	Status                   string   `json:"status"`
	ValueMilliseconds        *float64 `json:"value_milliseconds"`
	SpeedWorldUnitsPerSecond *float64 `json:"speed_world_units_per_second"`
	SpeedSource              string   `json:"speed_source"`
	Assumption               string   `json:"assumption"`
	UnavailabilityReason     *string  `json:"unavailability_reason"`
}

type ObservedNavigationTimingScenarios struct {
	OptimisticUninterrupted250UPSBaseline ObservedNavigationTimingScenario `json:"optimistic_uninterrupted_250ups_baseline"`
	ActiveWeaponUninterruptedBaseline     ObservedNavigationTimingScenario `json:"active_weapon_uninterrupted_baseline"`
	ObservedPaceProjection                ObservedNavigationTimingScenario `json:"observed_pace_projection"`
}

// ObservedNavigationInternalTiming embeds the product communication policy in
// the contract so an exact internal value cannot silently become user copy.
type ObservedNavigationInternalTiming struct {
	IntendedUse                string                            `json:"intended_use"`
	UserFacingMode             string                            `json:"user_facing_mode"`
	DecisionSemantics          string                            `json:"decision_semantics"`
	EstimateSemantics          string                            `json:"estimate_semantics"`
	ExactUserFacingTimeAllowed bool                              `json:"exact_user_facing_time_allowed"`
	ArrivalPrediction          bool                              `json:"arrival_prediction"`
	Scenarios                  ObservedNavigationTimingScenarios `json:"scenarios"`
}

type ObservedNavigationAsset struct {
	Status    string  `json:"status"`
	Reference *string `json:"reference"`
	SHA256    *string `json:"sha256"`
}

type ObservedNavigationProvenance struct {
	SourceArtifact                      string                  `json:"source_artifact"`
	GraphContractVersion                string                  `json:"graph_contract_version"`
	ShortestPathAlgorithm               string                  `json:"shortest_path_algorithm"`
	EndpointAssociationMethod           string                  `json:"endpoint_association_method"`
	EndpointToleranceWorldUnits         float64                 `json:"endpoint_tolerance_world_units"`
	EndpointVerticalToleranceWorldUnits float64                 `json:"endpoint_vertical_tolerance_world_units"`
	EdgeWeightMethod                    string                  `json:"edge_weight_method"`
	TimingScenarioContract              string                  `json:"timing_scenario_contract"`
	WeaponSpeedTableContract            string                  `json:"weapon_speed_table_contract"`
	OriginAssociationWorldUnits         *float64                `json:"origin_association_world_units"`
	TargetAssociationWorldUnits         *float64                `json:"target_association_world_units"`
	GraphAsset                          ObservedNavigationAsset `json:"graph_asset"`
	InputFields                         []string                `json:"input_fields"`
}

// ObservedNavigationPath is a factual candidate contract. It is intentionally
// not part of canonical bundle schema 3.8.0.
type ObservedNavigationPath struct {
	SchemaID          string                           `json:"schema_id"`
	ContractVersion   string                           `json:"contract_version"`
	IdentitySemantics string                           `json:"identity_semantics"`
	CausalRole        string                           `json:"causal_role"`
	Source            string                           `json:"source"`
	JoinKeys          ObservedNavigationPathJoinKeys   `json:"join_keys"`
	RoundNumber       int                              `json:"round_number"`
	T0Tick            int                              `json:"t0_tick"`
	Availability      ObservedNavigationAvailability   `json:"availability"`
	OriginMotion      ObservedNavigationMotionContext  `json:"origin_motion"`
	DirectDistance    ObservedNavigationDistanceFact   `json:"direct_distance"`
	Reachability      ObservedNavigationBooleanFact    `json:"reachability"`
	PathDistance      ObservedNavigationDistanceFact   `json:"path_distance"`
	InternalTiming    ObservedNavigationInternalTiming `json:"internal_timing"`
	AbstentionReasons []string                         `json:"abstention_reasons"`
	Provenance        ObservedNavigationProvenance     `json:"provenance"`
}
