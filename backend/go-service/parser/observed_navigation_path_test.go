package parser

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/navigation"

	"github.com/golang/geo/r3"
)

func TestObservedNavigationPathConnectedBuildsSeparateInternalScenarios(t *testing.T) {
	result, err := BuildObservedNavigationPath(navigationRequest(), obstacleNavigationGraph())
	if err != nil {
		t.Fatalf("BuildObservedNavigationPath() error = %v", err)
	}
	if result.Reachability.Value == nil || !*result.Reachability.Value {
		t.Fatalf("reachability = %#v, want true", result.Reachability)
	}
	if result.DirectDistance.ValueWorldUnits == nil || result.PathDistance.ValueWorldUnits == nil {
		t.Fatalf("distances must be available: direct=%#v path=%#v", result.DirectDistance, result.PathDistance)
	}
	if !almostEqualNavigation(*result.DirectDistance.ValueWorldUnits, math.Sqrt(200)) ||
		!almostEqualNavigation(*result.PathDistance.ValueWorldUnits, 20) {
		t.Fatalf("unexpected distances: direct=%v path=%v", *result.DirectDistance.ValueWorldUnits, *result.PathDistance.ValueWorldUnits)
	}
	if *result.PathDistance.ValueWorldUnits <= *result.DirectDistance.ValueWorldUnits {
		t.Fatal("obstacle graph path must be longer than direct distance")
	}

	ideal := result.InternalTiming.Scenarios.OptimisticUninterrupted250UPSBaseline
	weapon := result.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline
	pace := result.InternalTiming.Scenarios.ObservedPaceProjection
	assertEstimatedNavigationScenario(t, ideal, 250, 80)
	assertEstimatedNavigationScenario(t, weapon, 215, 20.0/215.0*1000)
	assertEstimatedNavigationScenario(t, pace, 100, 200)
	if result.InternalTiming.ExactUserFacingTimeAllowed || result.InternalTiming.ArrivalPrediction ||
		result.InternalTiming.UserFacingMode != models.NavigationTimingUserFacingMode {
		t.Fatalf("unsafe timing communication policy: %#v", result.InternalTiming)
	}
	if result.OriginMotion.HorizontalSpeedStatus != models.NavigationFactDerived ||
		result.OriginMotion.ActiveWeaponStatus != models.NavigationFactObserved ||
		result.OriginMotion.IsWalking == nil || !*result.OriginMotion.IsWalking ||
		result.OriginMotion.IsDucking == nil || !*result.OriginMotion.IsDucking ||
		result.OriginMotion.IsScoped == nil || !*result.OriginMotion.IsScoped {
		t.Fatalf("motion context was not preserved: %#v", result.OriginMotion)
	}
	if result.SchemaID != "stratai.observed_navigation_path@2" || result.ContractVersion != "2.0.0" {
		t.Fatalf("unexpected candidate version: %s %s", result.SchemaID, result.ContractVersion)
	}
}

func TestObservedNavigationPathPreparedEvaluatorMatchesPublicProducer(t *testing.T) {
	graph := obstacleNavigationGraph()
	prepared, err := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	requests := []models.ObservedNavigationPathRequest{navigationRequest()}
	reverse := navigationRequest()
	reverse.OriginPosition, reverse.TargetPosition = reverse.TargetPosition, reverse.OriginPosition
	reverse.JoinKeys.QueryID = "query-reverse"
	requests = append(requests, reverse)

	for _, request := range requests {
		want, err := BuildObservedNavigationPath(request, graph)
		if err != nil {
			t.Fatal(err)
		}
		got, err := buildObservedNavigationPathWithPreparedEvaluator(request, graph, prepared)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("prepared producer differs\ngot:  %#v\nwant: %#v", got, want)
		}
	}
}

func TestObservedNavigationPathWeaponBaselinesDifferAndUnknownFailsClosed(t *testing.T) {
	for _, fixture := range []struct {
		weapon string
		speed  float64
	}{
		{weapon: "Knife", speed: 250},
		{weapon: "AK-47", speed: 215},
		{weapon: "AWP", speed: 200},
	} {
		t.Run(fixture.weapon, func(t *testing.T) {
			request := navigationRequest()
			request.OriginMotion.ActiveWeapon = navigationStringPointer(fixture.weapon)
			result, err := BuildObservedNavigationPath(request, obstacleNavigationGraph())
			if err != nil {
				t.Fatal(err)
			}
			assertEstimatedNavigationScenario(
				t,
				result.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline,
				fixture.speed,
				20.0/fixture.speed*1000,
			)
		})
	}

	request := navigationRequest()
	request.OriginMotion.ActiveWeapon = navigationStringPointer("future-unknown-weapon")
	result, err := BuildObservedNavigationPath(request, obstacleNavigationGraph())
	if err != nil {
		t.Fatal(err)
	}
	assertUnavailableNavigationScenario(
		t,
		result.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline,
		"active_weapon_speed_unknown",
	)
}

func TestObservedNavigationPathObservedPaceZeroOrMissingDoesNotInventETA(t *testing.T) {
	request := navigationRequest()
	zero := 0.0
	request.OriginMotion.HorizontalSpeedWorldUnitsPerSecond = &zero
	result, err := BuildObservedNavigationPath(request, obstacleNavigationGraph())
	if err != nil {
		t.Fatal(err)
	}
	if result.OriginMotion.HorizontalSpeedWorldUnitsPerSecond == nil ||
		*result.OriginMotion.HorizontalSpeedWorldUnitsPerSecond != 0 {
		t.Fatalf("observed stationary speed was lost: %#v", result.OriginMotion)
	}
	assertUnavailableNavigationScenario(
		t,
		result.InternalTiming.Scenarios.ObservedPaceProjection,
		"observed_stationary_not_projectable",
	)

	request = navigationRequest()
	request.OriginMotion.HorizontalSpeedWorldUnitsPerSecond = nil
	request.OriginMotion.HorizontalSpeedSource = ""
	result, err = BuildObservedNavigationPath(request, obstacleNavigationGraph())
	if err != nil {
		t.Fatal(err)
	}
	assertUnavailableNavigationScenario(
		t,
		result.InternalTiming.Scenarios.ObservedPaceProjection,
		"observed_horizontal_speed_unavailable",
	)
}

func TestObservedNavigationPathRejectsInvalidOrFutureMotion(t *testing.T) {
	request := navigationRequest()
	request.OriginMotion.AvailabilityTick = request.T0Tick + 1
	if _, err := BuildObservedNavigationPath(request, obstacleNavigationGraph()); err == nil || !strings.Contains(err.Error(), "motion availability_tick") {
		t.Fatalf("future motion error = %v", err)
	}

	for name, speed := range map[string]float64{
		"negative": -1,
		"nan":      math.NaN(),
		"infinite": math.Inf(1),
	} {
		t.Run(name, func(t *testing.T) {
			request := navigationRequest()
			request.OriginMotion.HorizontalSpeedWorldUnitsPerSecond = &speed
			if _, err := BuildObservedNavigationPath(request, obstacleNavigationGraph()); err == nil || !strings.Contains(err.Error(), "horizontal speed") {
				t.Fatalf("invalid speed error = %v", err)
			}
		})
	}
}

func TestObservedNavigationPathDisconnectedPublishesNoPathOrTiming(t *testing.T) {
	graph := navigationGraph(
		[]navigation.Node{{ID: 1, Position: r3.Vector{}}, {ID: 2, Position: r3.Vector{X: 10}}},
		nil,
	)
	request := navigationRequest()
	request.TargetPosition = r3.Vector{X: 10}
	result, err := BuildObservedNavigationPath(request, graph)
	if err != nil {
		t.Fatalf("BuildObservedNavigationPath() error = %v", err)
	}
	if result.Reachability.Value == nil || *result.Reachability.Value {
		t.Fatalf("reachability = %#v, want false", result.Reachability)
	}
	assertNavigationUnavailable(t, result)
	if got := result.InternalTiming.Scenarios.ObservedPaceProjection.SpeedSource; got != navigationObservedPaceSource+":"+request.OriginMotion.HorizontalSpeedSource {
		t.Fatalf("unavailable path lost observed-pace provenance: %q", got)
	}
	if !reflect.DeepEqual(result.AbstentionReasons, []string{"no_navigable_connection"}) {
		t.Fatalf("abstention reasons = %v", result.AbstentionReasons)
	}
}

func TestObservedNavigationPathOutsideEndpointToleranceIsUnavailable(t *testing.T) {
	graph := navigationGraph(
		[]navigation.Node{{ID: 1, Position: r3.Vector{}}, {ID: 2, Position: r3.Vector{X: 10}}},
		[]navigation.Edge{{FromID: 1, ToID: 2, Weight: 10}},
	)
	for name, fixture := range map[string]struct {
		origin r3.Vector
		target r3.Vector
		reason string
	}{
		"origin": {origin: r3.Vector{X: 500}, target: r3.Vector{X: 10}, reason: "origin_outside_nav_tolerance"},
		"target": {origin: r3.Vector{}, target: r3.Vector{X: 500}, reason: "target_outside_nav_tolerance"},
		"both":   {origin: r3.Vector{X: 500}, target: r3.Vector{X: 600}, reason: "both_endpoints_outside_nav_tolerance"},
	} {
		t.Run(name, func(t *testing.T) {
			request := navigationRequest()
			request.OriginPosition = fixture.origin
			request.TargetPosition = fixture.target
			result, err := BuildObservedNavigationPath(request, graph)
			if err != nil {
				t.Fatalf("BuildObservedNavigationPath() error = %v", err)
			}
			assertNavigationUnavailable(t, result)
			if !reflect.DeepEqual(result.AbstentionReasons, []string{fixture.reason}) {
				t.Fatalf("abstention reasons = %v", result.AbstentionReasons)
			}
		})
	}
}

func TestObservedNavigationPathMissingNavIsUnavailableButDirectDistanceRemainsSeparate(t *testing.T) {
	result, err := BuildObservedNavigationPath(navigationRequest(), nil)
	if err != nil {
		t.Fatalf("BuildObservedNavigationPath() error = %v", err)
	}
	assertNavigationUnavailable(t, result)
	if result.DirectDistance.ValueWorldUnits == nil || *result.DirectDistance.ValueWorldUnits <= 0 {
		t.Fatalf("direct distance must remain a separate derived fact: %#v", result.DirectDistance)
	}
	if result.Provenance.GraphAsset.Status != navigationAssetMissing || result.Provenance.GraphAsset.Reference != nil {
		t.Fatalf("missing graph provenance = %#v", result.Provenance.GraphAsset)
	}
}

func TestObservedNavigationPathRejectsEndpointOrGraphAvailabilityAfterT0(t *testing.T) {
	request := navigationRequest()
	request.OriginAvailabilityTick = request.T0Tick + 1
	if _, err := BuildObservedNavigationPath(request, obstacleNavigationGraph()); err == nil || !strings.Contains(err.Error(), "availability_tick") {
		t.Fatalf("future endpoint availability error = %v", err)
	}

	request = navigationRequest()
	graph := obstacleNavigationGraph()
	graph.AvailabilityTick = request.T0Tick + 1
	if _, err := BuildObservedNavigationPath(request, graph); err == nil || !strings.Contains(err.Error(), "graph availability_tick") {
		t.Fatalf("future graph availability error = %v", err)
	}
}

func TestObservedNavigationPathRejectsLoadedGraphWithoutHash(t *testing.T) {
	graph := obstacleNavigationGraph()
	graph.AssetSHA256 = ""
	if _, err := BuildObservedNavigationPath(navigationRequest(), graph); err == nil ||
		!strings.Contains(err.Error(), "asset sha256") {
		t.Fatalf("missing graph hash error = %v", err)
	}
}

func TestObservedNavigationPathIsInvariantToFutureAndOracleMutations(t *testing.T) {
	type nonCausalFixture struct {
		Request        models.ObservedNavigationPathRequest
		FuturePosition r3.Vector
		OracleOutcome  string
	}
	fixture := nonCausalFixture{
		Request:        navigationRequest(),
		FuturePosition: r3.Vector{X: 999},
		OracleOutcome:  "round_won",
	}
	before, err := BuildObservedNavigationPath(fixture.Request, obstacleNavigationGraph())
	if err != nil {
		t.Fatalf("before mutation error = %v", err)
	}
	fixture.FuturePosition = r3.Vector{X: -999, Y: 12345}
	fixture.OracleOutcome = "round_lost"
	after, err := BuildObservedNavigationPath(fixture.Request, obstacleNavigationGraph())
	if err != nil {
		t.Fatalf("after mutation error = %v", err)
	}
	assertNavigationJSONEqual(t, before, after)

	for _, requestType := range []reflect.Type{
		reflect.TypeOf(models.ObservedNavigationPathRequest{}),
		reflect.TypeOf(models.ObservedNavigationMotionInput{}),
	} {
		for index := 0; index < requestType.NumField(); index++ {
			name := strings.ToLower(requestType.Field(index).Name)
			if strings.Contains(name, "future") || strings.Contains(name, "oracle") || strings.Contains(name, "outcome") {
				t.Fatalf("causal request exposes forbidden field %s", requestType.Field(index).Name)
			}
		}
	}
}

func TestObservedNavigationPathIdentifiersAreJoinOnlyAndExactTimeIsInternal(t *testing.T) {
	result, err := BuildObservedNavigationPath(navigationRequest(), obstacleNavigationGraph())
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	assertNavigationIDsAreJoinOnly(t, value, nil)
	root := value.(map[string]any)
	if _, exists := root["travel_time_estimate"]; exists {
		t.Fatal("legacy single ETA field must not be published")
	}
	assertNavigationTimingValuesAreInternal(t, value, nil)
}

func TestObservedNavigationPathDeterministicAcrossNodeAndEdgeOrder(t *testing.T) {
	ordered := obstacleNavigationGraph()
	reversed := navigationGraph(
		[]navigation.Node{ordered.Nodes[2], ordered.Nodes[1], ordered.Nodes[0]},
		[]navigation.Edge{ordered.Edges[1], ordered.Edges[0]},
	)
	left, err := BuildObservedNavigationPath(navigationRequest(), ordered)
	if err != nil {
		t.Fatal(err)
	}
	right, err := BuildObservedNavigationPath(navigationRequest(), reversed)
	if err != nil {
		t.Fatal(err)
	}
	assertNavigationJSONEqual(t, left, right)
}

func TestObservedNavigationPathRejectsInvalidEdgeWeights(t *testing.T) {
	for name, weight := range map[string]float64{
		"negative":          -1,
		"zero":              0,
		"positive_infinity": math.Inf(1),
		"negative_infinity": math.Inf(-1),
		"nan":               math.NaN(),
	} {
		t.Run(name, func(t *testing.T) {
			graph := navigationGraph(
				[]navigation.Node{{ID: 1, Position: r3.Vector{}}, {ID: 2, Position: r3.Vector{X: 10}}},
				[]navigation.Edge{{FromID: 1, ToID: 2, Weight: weight}},
			)
			request := navigationRequest()
			request.TargetPosition = r3.Vector{X: 10}
			if _, err := BuildObservedNavigationPath(request, graph); err == nil || !strings.Contains(err.Error(), "weight must be finite and positive") {
				t.Fatalf("invalid weight %v error = %v", weight, err)
			}
		})
	}
}

func TestDirectRayOrEuclideanDistanceDoesNotProveReachabilityOrTiming(t *testing.T) {
	directRayIsClear := true
	graph := navigationGraph(
		[]navigation.Node{{ID: 1, Position: r3.Vector{}}, {ID: 2, Position: r3.Vector{X: 10}}},
		nil,
	)
	request := navigationRequest()
	request.TargetPosition = r3.Vector{X: 10}
	result, err := BuildObservedNavigationPath(request, graph)
	if err != nil {
		t.Fatal(err)
	}
	if !directRayIsClear || result.DirectDistance.ValueWorldUnits == nil {
		t.Fatal("test setup must contain a clear direct segment and euclidean distance")
	}
	if result.Reachability.Value == nil || *result.Reachability.Value {
		t.Fatalf("direct segment incorrectly proved reachability: %#v", result)
	}
	assertNavigationUnavailable(t, result)
}

func navigationRequest() models.ObservedNavigationPathRequest {
	speed := 100.0
	weapon := "AK-47"
	walking, ducking, scoped := true, true, true
	return models.ObservedNavigationPathRequest{
		JoinKeys: models.ObservedNavigationPathJoinKeys{
			MatchID:        "match-1",
			RoundID:        "match-1:round:001",
			QueryID:        "match-1:round:001:path:origin:target:100",
			OriginPlayerID: "origin",
			TargetPlayerID: "target",
		},
		RoundNumber:            1,
		T0Tick:                 100,
		OriginPosition:         r3.Vector{},
		TargetPosition:         r3.Vector{X: 10, Y: 10},
		OriginAvailabilityTick: 100,
		TargetAvailabilityTick: 100,
		OriginMotion: &models.ObservedNavigationMotionInput{
			AvailabilityTick:                   100,
			HorizontalSpeedWorldUnitsPerSecond: &speed,
			HorizontalSpeedSource:              "position_delta",
			ActiveWeapon:                       &weapon,
			IsWalking:                          &walking,
			IsDucking:                          &ducking,
			IsScoped:                           &scoped,
		},
	}
}

func obstacleNavigationGraph() *navigation.Graph {
	return navigationGraph(
		[]navigation.Node{
			{ID: 1, Position: r3.Vector{}},
			{ID: 2, Position: r3.Vector{Y: 10}},
			{ID: 3, Position: r3.Vector{X: 10, Y: 10}},
		},
		[]navigation.Edge{
			{FromID: 1, ToID: 2, Weight: 10},
			{FromID: 2, ToID: 3, Weight: 10},
		},
	)
}

func navigationGraph(nodes []navigation.Node, edges []navigation.Edge) *navigation.Graph {
	return &navigation.Graph{
		Version:          navigation.GraphContractVersion,
		AssetReference:   "synthetic_nav_fixture",
		AssetSHA256:      strings.Repeat("a", 64),
		AvailabilityTick: 0,
		Nodes:            nodes,
		Edges:            edges,
	}
}

func assertNavigationUnavailable(t *testing.T, result models.ObservedNavigationPath) {
	t.Helper()
	if result.PathDistance.Status != models.NavigationFactUnavailable || result.PathDistance.ValueWorldUnits != nil {
		t.Fatalf("path distance must be unavailable: %#v", result.PathDistance)
	}
	for _, scenario := range []models.ObservedNavigationTimingScenario{
		result.InternalTiming.Scenarios.OptimisticUninterrupted250UPSBaseline,
		result.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline,
		result.InternalTiming.Scenarios.ObservedPaceProjection,
	} {
		if scenario.Status != models.NavigationFactUnavailable || scenario.ValueMilliseconds != nil ||
			scenario.SpeedWorldUnitsPerSecond != nil || scenario.UnavailabilityReason == nil {
			t.Fatalf("timing scenario must be unavailable without substitute values: %#v", scenario)
		}
	}
}

func assertEstimatedNavigationScenario(
	t *testing.T,
	scenario models.ObservedNavigationTimingScenario,
	speed, milliseconds float64,
) {
	t.Helper()
	if scenario.Status != models.NavigationFactEstimated || scenario.ValueMilliseconds == nil ||
		scenario.SpeedWorldUnitsPerSecond == nil || scenario.UnavailabilityReason != nil ||
		!almostEqualNavigation(*scenario.SpeedWorldUnitsPerSecond, speed) ||
		!almostEqualNavigation(*scenario.ValueMilliseconds, milliseconds) {
		t.Fatalf("unexpected estimated scenario: %#v", scenario)
	}
}

func assertUnavailableNavigationScenario(
	t *testing.T,
	scenario models.ObservedNavigationTimingScenario,
	reason string,
) {
	t.Helper()
	if scenario.Status != models.NavigationFactUnavailable || scenario.ValueMilliseconds != nil ||
		scenario.SpeedWorldUnitsPerSecond != nil || scenario.UnavailabilityReason == nil ||
		*scenario.UnavailabilityReason != reason {
		t.Fatalf("unexpected unavailable scenario: %#v", scenario)
	}
}

func assertNavigationJSONEqual(t *testing.T, left, right models.ObservedNavigationPath) {
	t.Helper()
	leftJSON, err := json.Marshal(left)
	if err != nil {
		t.Fatalf("json.Marshal(left) error = %v", err)
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		t.Fatalf("json.Marshal(right) error = %v", err)
	}
	if !reflect.DeepEqual(leftJSON, rightJSON) {
		t.Fatalf("navigation results differ\nleft:  %s\nright: %s", leftJSON, rightJSON)
	}
}

func assertNavigationIDsAreJoinOnly(t *testing.T, value any, path []string) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			childPath := append(append([]string(nil), path...), key)
			if key != "schema_id" && strings.HasSuffix(key, "_id") {
				if len(path) == 0 || path[len(path)-1] != "join_keys" {
					t.Fatalf("identifier outside join_keys at %s", strings.Join(childPath, "."))
				}
			}
			assertNavigationIDsAreJoinOnly(t, child, childPath)
		}
	case []any:
		for _, child := range typed {
			assertNavigationIDsAreJoinOnly(t, child, append(path, "[]"))
		}
	}
}

func assertNavigationTimingValuesAreInternal(t *testing.T, value any, path []string) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			childPath := append(append([]string(nil), path...), key)
			if (key == "value_milliseconds" || key == "speed_world_units_per_second") &&
				(len(path) == 0 || path[0] != "internal_timing") {
				t.Fatalf("exact timing value escaped internal_timing at %s", strings.Join(childPath, "."))
			}
			assertNavigationTimingValuesAreInternal(t, child, childPath)
		}
	case []any:
		for _, child := range typed {
			assertNavigationTimingValuesAreInternal(t, child, append(path, "[]"))
		}
	}
}

func navigationStringPointer(value string) *string { return &value }

func almostEqualNavigation(left, right float64) bool {
	return math.Abs(left-right) <= 1e-9
}
