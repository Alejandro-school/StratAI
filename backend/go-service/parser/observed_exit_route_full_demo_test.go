package parser

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/exitroute"
	mapspkg "cs2-demo-service/pkg/maps"
	"cs2-demo-service/pkg/navigation"
)

func exitRouteTestRegistryRoot() string {
	return filepath.Join("..", "..", "..", "ai_coach", "spatial", "exit_route")
}
func exitRouteTestMapsRoot() string { return filepath.Join("..", "..", "data", "maps") }
func exitRouteTestInputs(t *testing.T) (exitroute.PackageResolution, *navigation.Graph) {
	t.Helper()
	resolution := exitroute.ResolvePackage(exitRouteTestRegistryRoot(), exitRouteTestMapsRoot(), "de_mirage", exitroute.ContractVersion, "")
	if resolution.Status != exitroute.Available || resolution.Package == nil {
		t.Fatalf("package unavailable: %+v", resolution)
	}
	manager := mapspkg.NewMapManager(exitRouteTestMapsRoot())
	if err := manager.LoadMap("de_mirage"); err != nil {
		t.Fatal(err)
	}
	graph, err := manager.NavigationGraphSnapshot()
	if err != nil || graph == nil {
		t.Fatalf("graph: %v", err)
	}
	return resolution, graph
}

func TestObservedExitRouteSeparatesObjectivePhysicalAndPendingHumanEvaluation(t *testing.T) {
	resolution, graph := exitRouteTestInputs(t)
	prepared, err := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	sample := resolution.Package.RouteSamples[0]
	player := models.ReplayPlayerState{SteamID: 7, X: int(sample.Origin.X), Y: int(sample.Origin.Y), Z: sample.Origin.Z, Alive: true, IsDucking: true, IsWalking: true, ObservedActiveWeapon: "Knife", ActiveWeaponObservationStatus: "observed", ActiveWeaponAvailabilityTick: 64, MotionHorizontalSpeed: 125, MotionAvailable: true, MotionSource: "causal_position_delta", MotionAvailabilityTick: 64}
	record, err := BuildObservedExitRouteRecord("de_mirage", "demo-aaaaaaaaaaaaaaaaaaaa", 1, 64, "participant-001", player, graph, prepared, resolution)
	if err != nil {
		t.Fatal(err)
	}
	if record.Sample.Status != "observed" || len(record.Candidates) != 1 {
		t.Fatalf("candidate unavailable: %+v", record)
	}
	candidate := record.Candidates[0]
	if candidate.Objective.ReasonCodes == nil {
		t.Fatal("available objective reasons must be an explicit empty array")
	}
	if candidate.Objective.Reachability.Value == nil || !*candidate.Objective.Reachability.Value || candidate.PhysicalCost.PathDistance.ValueWorldUnits == nil {
		t.Fatalf("objective/physical route unavailable: %+v", candidate)
	}
	if candidate.PhysicalCost.InternalTiming.ExactUserFacingTimeAllowed || candidate.PhysicalCost.InternalTiming.ArrivalPrediction || candidate.PhysicalCost.InternalTiming.DecisionSemantics != models.NavigationTimingDecisionSemantics {
		t.Fatal("navigation timing policy changed")
	}
	if candidate.PhysicalCost.InternalTiming.Scenarios.ActiveWeaponUninterruptedBaseline.Status != models.NavigationFactEstimated || candidate.PhysicalCost.InternalTiming.Scenarios.ObservedPaceProjection.Status != models.NavigationFactEstimated {
		t.Fatalf("T0 context was not used: %+v", candidate.PhysicalCost.InternalTiming)
	}
	if candidate.Objective.FieldID != "FLD-NAV-EXIT-PATH" {
		t.Fatalf("wrong objective field id: %+v", candidate.Objective)
	}
	if candidate.Tactical.Status != "unavailable" || !reflect.DeepEqual(candidate.Tactical.ReasonCodes, []string{SpatialTacticalJudgmentUnavailableReason}) {
		t.Fatalf("human evaluation was fabricated: %+v", candidate.Tactical)
	}
}

func TestObservedExitRouteAbsentContextAndUnknownCoverageFailClosed(t *testing.T) {
	resolution, graph := exitRouteTestInputs(t)
	prepared, _ := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	sample := resolution.Package.RouteSamples[0]
	record, err := BuildObservedExitRouteRecord("de_mirage", "demo-aaaaaaaaaaaaaaaaaaaa", 1, 1, "participant-001", models.ReplayPlayerState{X: int(sample.Origin.X), Y: int(sample.Origin.Y), Z: sample.Origin.Z, Alive: true}, graph, prepared, resolution)
	if err != nil {
		t.Fatal(err)
	}
	scenarios := record.Candidates[0].PhysicalCost.InternalTiming.Scenarios
	if scenarios.ActiveWeaponUninterruptedBaseline.Status != "unavailable" || scenarios.ObservedPaceProjection.Status != "unavailable" {
		t.Fatalf("missing T0 context got nominal fallbacks: %+v", scenarios)
	}
	unknown := exitroute.PackageResolution{Status: exitroute.Unavailable, ReasonCodes: []string{exitroute.ReasonMapNotRegistered}}
	record, err = BuildObservedExitRouteRecord("de_inferno", "demo-aaaaaaaaaaaaaaaaaaaa", 1, 1, "participant-001", models.ReplayPlayerState{Alive: true}, graph, prepared, unknown)
	if err != nil {
		t.Fatal(err)
	}
	if record.Sample.Status != "unavailable" || !reflect.DeepEqual(record.Sample.ReasonCodes, []string{exitroute.ReasonMapNotRegistered}) || len(record.Candidates) != 0 {
		t.Fatalf("unknown map did not fail closed: %+v", record)
	}
}

func TestObservedExitRouteFullDemoIsDeterministicCausalAndPrivate(t *testing.T) {
	resolution, graph := exitRouteTestInputs(t)
	sample := resolution.Package.RouteSamples[0]
	player := models.ReplayPlayerState{SteamID: 76561199999999999, Name: "raw-name", X: int(sample.Origin.X), Y: int(sample.Origin.Y), Z: sample.Origin.Z, Alive: true, ObservedActiveWeapon: "Knife", ActiveWeaponObservationStatus: "observed", MotionHorizontalSpeed: 100, MotionAvailable: true, MotionSource: "causal_position_delta"}
	replay := &models.ReplayData{Metadata: models.ReplayMetadata{MapName: "de_mirage", TickRate: 64}, Rounds: []models.ReplayRound{{Round: 1, StartTick: 0, EndTick: 128, Frames: []models.ReplayFrame{{Tick: 0, Players: []models.ReplayPlayerState{player}}, {Tick: 64, Players: []models.ReplayPlayerState{player}}}}}}
	checksum := strings.Repeat("a", 64)
	var first, second bytes.Buffer
	report1, err := BuildObservedExitRouteFullDemoReport(replay, graph, resolution, "demo-aaaaaaaaaaaaaaaaaaaa", checksum, &first)
	if err != nil {
		t.Fatal(err)
	}
	replay.Rounds[0].Frames[1].Players[0].Health = 99
	replay.Rounds[0].Frames[1].Players[0].Name = "future-oracle-mutation"
	report2, err := BuildObservedExitRouteFullDemoReport(replay, graph, resolution, "demo-aaaaaaaaaaaaaaaaaaaa", checksum, &second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) || report1.RowsArtifact.LogicalSHA256 != report2.RowsArtifact.LogicalSHA256 {
		t.Fatal("unrelated future/outcome mutation changed causal output")
	}
	if report1.Coverage.Records != 2 || report1.Coverage.CandidateEvaluations != 2 || report1.Coverage.TacticalUnavailable != 2 || report1.Coverage.TacticalReasonCounts[SpatialTacticalJudgmentUnavailableReason] != 2 {
		t.Fatalf("counts: %+v", report1.Coverage)
	}
	if report1.PackageResolution.ReasonCodes == nil || len(report1.PackageResolution.ReasonCodes) != 0 {
		t.Fatalf("available package reasons must be an explicit empty array: %#v", report1.PackageResolution.ReasonCodes)
	}
	for _, line := range bytes.Split(bytes.TrimSpace(first.Bytes()), []byte{'\n'}) {
		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatal(err)
		}
		payload := string(line)
		for _, forbidden := range []string{"76561199999999999", "raw-name", "future-oracle-mutation"} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("forbidden value leaked: %s", forbidden)
			}
		}
	}
}
