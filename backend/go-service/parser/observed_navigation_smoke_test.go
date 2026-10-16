package parser

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/navigation"

	"github.com/golang/geo/r3"
)

func TestObservedNavigationSmokeReportExercisesRealMotionAndWeaponContext(t *testing.T) {
	report, err := BuildObservedNavigationSmokeReport(
		navigationSmokeReplayFixture(),
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		3,
	)
	if err != nil {
		t.Fatalf("BuildObservedNavigationSmokeReport() error = %v", err)
	}
	if report.Status != "passed" || report.Summary.Rows != 3 || report.Summary.ReachableRows != 3 {
		t.Fatalf("unexpected smoke report summary: %#v", report)
	}
	if report.Summary.IdealBaselineRows != 3 || report.Summary.ActiveWeaponBaselineRows != 3 ||
		report.Summary.ObservedPaceProjectionRows != 3 || report.Summary.ExactUserFacingTimeViolations != 0 {
		t.Fatalf("timing coverage is incomplete: %#v", report.Summary)
	}
	if report.Summary.WalkingContexts == 0 || report.Summary.NonWalkingMovingContexts == 0 ||
		report.Summary.KnifeContexts == 0 || len(report.Summary.ActiveWeapons) != 3 ||
		len(report.Summary.WeaponBaselineSpeeds) != 3 || report.Summary.DuckingContexts == 0 ||
		report.Summary.ScopedContexts == 0 {
		t.Fatalf("real movement/weapon diversity is incomplete: %#v", report.Summary)
	}
	for _, row := range report.Rows {
		if row.JoinKeys.OriginPlayerID != "participant-001" || row.JoinKeys.TargetPlayerID != "participant-002" ||
			strings.Contains(row.JoinKeys.QueryID, "steam:") {
			t.Fatalf("smoke query used a non-causal or unexpected pair: %#v", row.JoinKeys)
		}
		if row.InternalTiming.ExactUserFacingTimeAllowed || row.InternalTiming.ArrivalPrediction {
			t.Fatalf("unsafe row policy: %#v", row.InternalTiming)
		}
	}
}

func TestObservedNavigationSmokeReportRejectsMissingRealMovementOrKnifeDiversity(t *testing.T) {
	noWalking := navigationSmokeReplayFixture()
	for frameIndex := range noWalking.Rounds[0].Frames {
		noWalking.Rounds[0].Frames[frameIndex].Players[0].IsWalking = false
	}
	_, err := BuildObservedNavigationSmokeReport(
		noWalking,
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		3,
	)
	if err == nil || !strings.Contains(err.Error(), "walking and non-walking") {
		t.Fatalf("missing walking coverage error = %v", err)
	}

	noKnife := navigationSmokeReplayFixture()
	noKnife.Rounds[0].Frames[0].Players[0].ObservedActiveWeapon = "M4A4"
	_, err = BuildObservedNavigationSmokeReport(
		noKnife,
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		3,
	)
	if err == nil || !strings.Contains(err.Error(), "Knife") {
		t.Fatalf("missing Knife coverage error = %v", err)
	}
}

func TestObservedNavigationSmokeReportIsDeterministicAcrossReplayOrder(t *testing.T) {
	replay := navigationSmokeReplayFixture()
	left, err := BuildObservedNavigationSmokeReport(
		replay,
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		3,
	)
	if err != nil {
		t.Fatal(err)
	}
	replay.Rounds[0].Frames[0], replay.Rounds[0].Frames[2] = replay.Rounds[0].Frames[2], replay.Rounds[0].Frames[0]
	for index := range replay.Rounds[0].Frames {
		players := replay.Rounds[0].Frames[index].Players
		players[0], players[1] = players[1], players[0]
	}
	right, err := BuildObservedNavigationSmokeReport(
		replay,
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		3,
	)
	if err != nil {
		t.Fatal(err)
	}
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	if !reflect.DeepEqual(leftJSON, rightJSON) {
		t.Fatalf("smoke reports differ\nleft: %s\nright: %s", leftJSON, rightJSON)
	}
}

func TestObservedNavigationSmokeReportFailsClosedWithoutGraphOrCoverage(t *testing.T) {
	_, err := BuildObservedNavigationSmokeReport(
		navigationSmokeReplayFixture(),
		nil,
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		3,
	)
	if err == nil || !strings.Contains(err.Error(), "NAV graph") {
		t.Fatalf("missing graph error = %v", err)
	}

	replay := navigationSmokeReplayFixture()
	for frameIndex := range replay.Rounds[0].Frames {
		for playerIndex := range replay.Rounds[0].Frames[frameIndex].Players {
			replay.Rounds[0].Frames[frameIndex].Players[playerIndex].Team = ""
		}
	}
	_, err = BuildObservedNavigationSmokeReport(
		replay,
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		3,
	)
	if err == nil || !strings.Contains(err.Error(), "causal same-team queries") {
		t.Fatalf("missing causal coverage error = %v", err)
	}
}

func navigationSmokeReplayFixture() *models.ReplayData {
	frames := make([]models.ReplayFrame, 0, 3)
	weapons := []string{"Knife", "AK-47", "AWP"}
	for index, tick := range []int{100, 104, 108} {
		frames = append(frames, models.ReplayFrame{
			Tick: tick,
			Players: []models.ReplayPlayerState{
				{
					SteamID:                       10,
					Team:                          "CT",
					Alive:                         true,
					X:                             index,
					ObservedActiveWeapon:          weapons[index],
					ActiveWeaponObservationStatus: models.ActiveWeaponStatusObserved,
					ActiveWeaponAvailabilityTick:  tick,
					MotionHorizontalSpeed:         100,
					MotionAvailable:               true,
					MotionSource:                  "position_delta",
					MotionAvailabilityTick:        tick,
					IsWalking:                     index == 0,
					IsDucking:                     index == 1,
					IsScoped:                      index == 2,
				},
				{SteamID: 20, Team: "CT", Alive: true, X: 20},
				{SteamID: 30, Team: "T", Alive: true, X: 10},
			},
		})
	}
	return &models.ReplayData{
		Metadata: models.ReplayMetadata{MapName: "de_mirage", TickRate: 64},
		Rounds: []models.ReplayRound{{
			Round: 1, StartTick: 100, EndTick: 108, Frames: frames,
		}},
	}
}

func navigationSmokeGraphFixture() *navigation.Graph {
	return &navigation.Graph{
		Version:          navigation.GraphContractVersion,
		AssetReference:   "de_mirage/de_mirage.nav",
		AssetSHA256:      strings.Repeat("b", 64),
		AvailabilityTick: 0,
		Nodes: []navigation.Node{
			{ID: 1, Position: r3.Vector{}},
			{ID: 2, Position: r3.Vector{X: 10}},
			{ID: 3, Position: r3.Vector{X: 20}},
		},
		Edges: []navigation.Edge{
			{FromID: 1, ToID: 2, Weight: 10},
			{FromID: 2, ToID: 1, Weight: 10},
			{FromID: 2, ToID: 3, Weight: 10},
			{FromID: 3, ToID: 2, Weight: 10},
		},
	}
}
