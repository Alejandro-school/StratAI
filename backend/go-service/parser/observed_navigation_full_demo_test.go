package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/navigation"

	"github.com/golang/geo/r3"
)

func TestObservedNavigationFullDemoSweepsEveryEligibleFrameWithPseudonymousRows(t *testing.T) {
	replay := observedNavigationFullDemoReplayFixture()
	var rows bytes.Buffer
	report, err := BuildObservedNavigationFullDemoReport(
		replay,
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		&rows,
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || report.Scope != ObservedNavigationFullDemoScope ||
		report.SelectionContract != ObservedNavigationFullDemoSelectionContract ||
		report.PairExhaustive || !report.DirectionalPairs {
		t.Fatalf("unexpected full-demo semantics: %#v", report)
	}
	if report.RoundCount != 2 || report.Summary.ReplayFrames != 5 ||
		report.Summary.EligibleFrames != 4 || report.Summary.IneligibleFrames != 1 ||
		report.Summary.Counts.Queries != 4 || report.RowsArtifact.RowCount != 4 {
		t.Fatalf("unexpected frame coverage: %#v", report.Summary)
	}
	if len(report.Rounds) != 2 || report.Rounds[0].RoundNumber != 1 ||
		report.Rounds[0].EligibleFrames != 3 || report.Rounds[1].RoundNumber != 2 ||
		report.Rounds[1].ReplayFrames != 2 || report.Rounds[1].EligibleFrames != 1 ||
		report.Rounds[1].IneligibleFrames != 1 {
		t.Fatalf("unexpected round coverage: %#v", report.Rounds)
	}
	if report.SampleStrideTicks != 4 || report.SampleIntervalMilliseconds != 62.5 {
		t.Fatalf("unexpected sampling metadata: stride=%d interval=%f", report.SampleStrideTicks, report.SampleIntervalMilliseconds)
	}
	if report.Summary.Counts.ReachableQueries != 4 ||
		report.Summary.Counts.OptimisticBaseline.Estimated != 4 ||
		report.Summary.Counts.ActiveWeaponBaseline.Estimated != 4 ||
		report.Summary.Counts.ObservedPaceProjection.Estimated != 4 ||
		report.Summary.Counts.ExactUserFacingTimeViolations != 0 ||
		report.Summary.Counts.ArrivalPredictionViolations != 0 {
		t.Fatalf("unexpected timing summary: %#v", report.Summary.Counts)
	}
	if report.Summary.Distributions.PathDistanceWorldUnits.Count != 4 ||
		report.Summary.Distributions.ObservedHorizontalSpeedWorldUnitsPerSecond.P95 == nil {
		t.Fatalf("missing distributions: %#v", report.Summary.Distributions)
	}

	logicalDigest := sha256.Sum256(rows.Bytes())
	if report.RowsArtifact.LogicalBytes != int64(rows.Len()) ||
		report.RowsArtifact.LogicalSHA256 != hex.EncodeToString(logicalDigest[:]) ||
		bytes.Count(rows.Bytes(), []byte{'\n'}) != 4 {
		t.Fatalf("logical row integrity mismatch: %#v", report.RowsArtifact)
	}
	if strings.Contains(rows.String(), "steam") || strings.Contains(rows.String(), "secret-player") {
		t.Fatalf("raw identity leaked into streamed rows: %s", rows.String())
	}

	decoder := json.NewDecoder(bytes.NewReader(rows.Bytes()))
	var ticks []int
	for decoder.More() {
		var row models.ObservedNavigationPath
		if err := decoder.Decode(&row); err != nil {
			t.Fatal(err)
		}
		ticks = append(ticks, row.T0Tick)
		if row.JoinKeys.MatchID != "demo-aaaaaaaaaaaaaaaaaaaa" ||
			!strings.HasPrefix(row.JoinKeys.OriginPlayerID, "participant-") ||
			!strings.HasPrefix(row.JoinKeys.TargetPlayerID, "participant-") ||
			row.JoinKeys.OriginPlayerID == row.JoinKeys.TargetPlayerID {
			t.Fatalf("row identifiers are not local aliases: %#v", row.JoinKeys)
		}
	}
	if !reflect.DeepEqual(ticks, []int{100, 104, 108, 204}) {
		t.Fatalf("stream order ticks = %v", ticks)
	}

	report.RowsArtifact.Path = ObservedNavigationFullDemoRowsFilename
	report.RowsArtifact.Bytes = 123
	report.RowsArtifact.SHA256 = strings.Repeat("c", 64)
	report.BuildIdentifier = "test-full-navigation"
	if err := ValidateObservedNavigationFullDemoReport(report); err != nil {
		t.Fatalf("completed report rejected: %v", err)
	}
	report.Summary.Counts.Queries--
	if err := ValidateObservedNavigationFullDemoReport(report); err == nil {
		t.Fatal("tampered completed report was accepted")
	}
}

func TestObservedNavigationFullDemoSeparatesDisconnectedAndUnavailable(t *testing.T) {
	replay := observedNavigationFullDemoReachabilityFixture()
	graph := &navigation.Graph{
		Version:          navigation.GraphContractVersion,
		AssetReference:   "de_mirage/de_mirage.nav",
		AssetSHA256:      strings.Repeat("b", 64),
		AvailabilityTick: 0,
		Nodes: []navigation.Node{
			{ID: 1, Position: r3.Vector{}},
			{ID: 2, Position: r3.Vector{X: 20}},
			{ID: 3, Position: r3.Vector{X: 200}},
			{ID: 4, Position: r3.Vector{X: 220}},
		},
		Edges: []navigation.Edge{
			{FromID: 1, ToID: 2, Weight: 20},
			{FromID: 2, ToID: 1, Weight: 20},
		},
	}
	var rows bytes.Buffer
	report, err := BuildObservedNavigationFullDemoReport(
		replay,
		graph,
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		&rows,
	)
	if err != nil {
		t.Fatal(err)
	}
	counts := report.Summary.Counts
	if counts.Queries != 3 || counts.ReachableQueries != 1 ||
		counts.DisconnectedQueries != 1 || counts.EndpointUnavailableQueries != 1 {
		t.Fatalf("reachability categories were conflated: %#v", counts)
	}
	if counts.AbstentionReasons["no_navigable_connection"] != 1 ||
		counts.AbstentionReasons["both_endpoints_outside_nav_tolerance"] != 1 {
		t.Fatalf("unexpected abstentions: %#v", counts.AbstentionReasons)
	}
	if counts.OptimisticBaseline.Estimated != 1 || counts.OptimisticBaseline.Unavailable != 2 ||
		counts.OptimisticBaseline.UnavailabilityReasons["path_distance_unavailable"] != 2 {
		t.Fatalf("scenario availability mismatch: %#v", counts.OptimisticBaseline)
	}
	if report.Summary.Distributions.DirectDistanceWorldUnits.Count != 3 ||
		report.Summary.Distributions.PathDistanceWorldUnits.Count != 1 ||
		report.Summary.Distributions.PathToDirectDistanceRatio.Count != 1 {
		t.Fatalf("distribution reachability mismatch: %#v", report.Summary.Distributions)
	}
}

func TestObservedNavigationFullDemoIsDeterministicAcrossReplayOrder(t *testing.T) {
	leftReplay := observedNavigationFullDemoReplayFixture()
	rightReplay := observedNavigationFullDemoReplayFixture()
	rightReplay.Rounds[0], rightReplay.Rounds[1] = rightReplay.Rounds[1], rightReplay.Rounds[0]
	for roundIndex := range rightReplay.Rounds {
		frames := rightReplay.Rounds[roundIndex].Frames
		for left, right := 0, len(frames)-1; left < right; left, right = left+1, right-1 {
			frames[left], frames[right] = frames[right], frames[left]
		}
		for frameIndex := range frames {
			players := frames[frameIndex].Players
			for left, right := 0, len(players)-1; left < right; left, right = left+1, right-1 {
				players[left], players[right] = players[right], players[left]
			}
		}
	}

	var leftRows, rightRows bytes.Buffer
	leftReport, err := BuildObservedNavigationFullDemoReport(
		leftReplay,
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		&leftRows,
	)
	if err != nil {
		t.Fatal(err)
	}
	rightReport, err := BuildObservedNavigationFullDemoReport(
		rightReplay,
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		&rightRows,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(leftReport, rightReport) || !bytes.Equal(leftRows.Bytes(), rightRows.Bytes()) {
		t.Fatalf("full sweep differs after replay reordering\nleft:  %#v\nright: %#v", leftReport, rightReport)
	}
}

func TestObservedNavigationFullDemoFailsOnWriterError(t *testing.T) {
	_, err := BuildObservedNavigationFullDemoReport(
		observedNavigationFullDemoReplayFixture(),
		navigationSmokeGraphFixture(),
		"demo-aaaaaaaaaaaaaaaaaaaa",
		strings.Repeat("a", 64),
		observedNavigationFullDemoFailingWriter{},
	)
	if err == nil || !strings.Contains(err.Error(), "write query") {
		t.Fatalf("writer error = %v", err)
	}
}

func TestObservedNavigationFullDemoRejectsInvalidRoundInsteadOfSkippingIt(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*models.ReplayData)
		want   string
	}{
		{
			name: "non-positive round",
			mutate: func(replay *models.ReplayData) {
				replay.Rounds = append(replay.Rounds, models.ReplayRound{Round: 0})
			},
			want: "invalid round number",
		},
		{
			name: "reversed bounds",
			mutate: func(replay *models.ReplayData) {
				replay.Rounds[0].EndTick = replay.Rounds[0].StartTick - 1
			},
			want: "invalid tick bounds",
		},
		{
			name: "frame outside bounds",
			mutate: func(replay *models.ReplayData) {
				replay.Rounds[0].Frames[0].Tick = replay.Rounds[0].EndTick + 1
			},
			want: "outside round",
		},
		{
			name: "duplicate frame tick",
			mutate: func(replay *models.ReplayData) {
				replay.Rounds[0].Frames[1].Tick = replay.Rounds[0].Frames[0].Tick
			},
			want: "duplicate frame tick",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			replay := observedNavigationFullDemoReplayFixture()
			testCase.mutate(replay)
			var rows bytes.Buffer
			_, err := BuildObservedNavigationFullDemoReport(
				replay,
				navigationSmokeGraphFixture(),
				"demo-aaaaaaaaaaaaaaaaaaaa",
				strings.Repeat("a", 64),
				&rows,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("invalid replay coverage was not rejected: %v", err)
			}
			if rows.Len() != 0 {
				t.Fatalf("rows were emitted before invalid round coverage was rejected: %d bytes", rows.Len())
			}
		})
	}
}

func TestObservedNavigationFullDemoSampleStrideUsesAllFrameDeltas(t *testing.T) {
	replay := observedNavigationFullDemoReplayFixture()
	replay.Rounds[0].Frames[1].Tick += 4
	if got := observedNavigationFullDemoSampleStride(replay); got != 4 {
		t.Fatalf("sample stride = %d, want greatest common divisor 4", got)
	}
}

func TestObservedNavigationFullDemoDistributionUsesNearestRank(t *testing.T) {
	distribution := observedNavigationFullDemoDistribution([]float64{100, 1, 4, 2, 3})
	if distribution.Count != 5 || distribution.P50 == nil || *distribution.P50 != 3 ||
		distribution.P95 == nil || *distribution.P95 != 100 ||
		distribution.Max == nil || *distribution.Max != 100 {
		t.Fatalf("unexpected nearest-rank distribution: %#v", distribution)
	}
	empty := observedNavigationFullDemoDistribution(nil)
	if empty.Count != 0 || empty.P50 != nil || empty.P95 != nil || empty.Max != nil {
		t.Fatalf("empty distribution published substitute values: %#v", empty)
	}
}

type observedNavigationFullDemoFailingWriter struct{}

func (observedNavigationFullDemoFailingWriter) Write([]byte) (int, error) {
	return 0, errors.New("intentional writer failure")
}

func observedNavigationFullDemoReplayFixture() *models.ReplayData {
	replay := navigationSmokeReplayFixture()
	for roundIndex := range replay.Rounds {
		for frameIndex := range replay.Rounds[roundIndex].Frames {
			for playerIndex := range replay.Rounds[roundIndex].Frames[frameIndex].Players {
				replay.Rounds[roundIndex].Frames[frameIndex].Players[playerIndex].Name = "secret-player"
			}
		}
	}
	eligible := models.ReplayFrame{
		Tick: 204,
		Players: []models.ReplayPlayerState{
			observedNavigationFullDemoOrigin(10, 10, 204),
			{SteamID: 20, Name: "secret-player", Team: "CT", Alive: true, X: 20},
		},
	}
	ineligible := models.ReplayFrame{
		Tick: 200,
		Players: []models.ReplayPlayerState{
			observedNavigationFullDemoOrigin(10, 10, 200),
		},
	}
	replay.Rounds = []models.ReplayRound{
		{Round: 2, StartTick: 200, EndTick: 204, Frames: []models.ReplayFrame{eligible, ineligible}},
		replay.Rounds[0],
	}
	return replay
}

func observedNavigationFullDemoReachabilityFixture() *models.ReplayData {
	frames := []models.ReplayFrame{
		observedNavigationFullDemoPairFrame(100, 0, 20),
		observedNavigationFullDemoPairFrame(104, 200, 220),
		observedNavigationFullDemoPairFrame(108, 1000, 1020),
	}
	return &models.ReplayData{
		Metadata: models.ReplayMetadata{MapName: "de_mirage", TickRate: 64},
		Rounds: []models.ReplayRound{{
			Round: 1, StartTick: 100, EndTick: 108, Frames: frames,
		}},
	}
}

func observedNavigationFullDemoPairFrame(tick, originX, targetX int) models.ReplayFrame {
	return models.ReplayFrame{
		Tick: tick,
		Players: []models.ReplayPlayerState{
			observedNavigationFullDemoOrigin(10, originX, tick),
			{SteamID: 20, Team: "CT", Alive: true, X: targetX},
		},
	}
}

func observedNavigationFullDemoOrigin(id uint64, x, tick int) models.ReplayPlayerState {
	return models.ReplayPlayerState{
		SteamID:                       id,
		Name:                          "secret-player",
		Team:                          "CT",
		Alive:                         true,
		X:                             x,
		ObservedActiveWeapon:          "Knife",
		ActiveWeaponObservationStatus: models.ActiveWeaponStatusObserved,
		ActiveWeaponAvailabilityTick:  tick,
		MotionHorizontalSpeed:         100,
		MotionAvailable:               true,
		MotionSource:                  "position_delta",
		MotionAvailabilityTick:        tick,
	}
}
