package parser

import (
	"encoding/json"
	"strings"
	"testing"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
)

type observedVisualContactSmokeGeometry struct {
	loaded      bool
	alwaysClear bool
}

func (geometry observedVisualContactSmokeGeometry) IsLoaded() bool {
	return geometry.loaded
}

func (geometry observedVisualContactSmokeGeometry) IsVisible(start, end r3.Vector) bool {
	if geometry.alwaysClear {
		return true
	}
	return start.X >= 0 && end.X >= 0
}

func TestBuildObservedVisualContactSmokeReportCoversCausalRoundPrefix(t *testing.T) {
	replay := observedVisualContactSmokeReplayFixture()
	report, err := BuildObservedVisualContactSmokeReport(
		&replay,
		observedVisualContactSmokeGeometry{loaded: true},
		observedVisualContactSmokeAlias(),
		observedVisualContactSmokeSHA256(),
		16,
		observedVisualContactSmokeAssets(),
	)
	if err != nil {
		t.Fatalf("BuildObservedVisualContactSmokeReport() error = %v", err)
	}

	if report.SchemaID != ObservedVisualContactSmokeSchemaID || report.Status != "passed" {
		t.Fatalf("unexpected report identity/status: %#v", report)
	}
	if report.Selection != (ObservedVisualContactSmokeSelection{
		Contract:         ObservedVisualContactSmokeSelectionContract,
		RoundNumber:      1,
		StartTick:        100,
		EndTick:          160,
		T0Tick:           160,
		InputRoundFrames: 17,
		SelectedFrames:   16,
		FrameLimit:       16,
	}) {
		t.Fatalf("unexpected selection: %#v", report.Selection)
	}
	if report.Summary.ObservedEvaluations == 0 ||
		report.Summary.LineOfSightClear == 0 || report.Summary.LineOfSightBlocked == 0 ||
		report.Summary.FOVInside == 0 || report.Summary.FOVOutside == 0 ||
		report.Summary.SmokeOccluded == 0 || report.Summary.SmokeClear == 0 ||
		report.Summary.VisualContactTrue == 0 || report.Summary.VisualContactFalse == 0 ||
		report.Summary.ActiveSmokeEvaluations == 0 || report.Summary.Episodes == 0 ||
		report.Summary.Boundaries[models.VisualContactBoundaryStarted] == 0 ||
		report.Summary.Boundaries[models.VisualContactBoundaryEnded] == 0 ||
		report.Summary.Boundaries[models.VisualContactBoundaryInactive] == 0 {
		t.Fatalf("required real-branch coverage is missing: %#v", report.Summary)
	}
	if report.Summary.RawSteamIDViolations != 0 {
		t.Fatalf("raw Steam ID violations = %d, want 0", report.Summary.RawSteamIDViolations)
	}
	if err := ValidateObservedVisualContactSmokeReport(report); err != nil {
		t.Fatalf("ValidateObservedVisualContactSmokeReport() error = %v", err)
	}

	var continuedAtSmoke *models.ObservedVisualContactEvaluation
	for index := range report.Ledger.Evaluations {
		evaluation := &report.Ledger.Evaluations[index]
		if evaluation.JoinKeys.ObserverID == "participant-003" &&
			evaluation.JoinKeys.SubjectID == "participant-001" &&
			evaluation.Provenance.SourceFrameTick == 140 {
			continuedAtSmoke = evaluation
			break
		}
	}
	if continuedAtSmoke == nil || continuedAtSmoke.Boundary != models.VisualContactBoundaryContinued {
		t.Fatalf("pre-smoke contact at smoke tick = %#v, want continued", continuedAtSmoke)
	}
	var warmedEpisode *models.ObservedVisualContactEpisode
	for index := range report.Ledger.Contacts {
		episode := &report.Ledger.Contacts[index]
		if episode.JoinKeys.ObserverID == "participant-003" &&
			episode.JoinKeys.SubjectID == "participant-001" {
			warmedEpisode = episode
			break
		}
	}
	if warmedEpisode == nil || warmedEpisode.StartTick != report.Selection.StartTick {
		t.Fatalf("pre-smoke episode = %#v, want real round-prefix start %d", warmedEpisode, report.Selection.StartTick)
	}
}

func TestBuildObservedVisualContactSmokeReportFailsClosedOnCoverageGap(t *testing.T) {
	replay := observedVisualContactSmokeReplayFixture()
	_, err := BuildObservedVisualContactSmokeReport(
		&replay,
		observedVisualContactSmokeGeometry{loaded: true, alwaysClear: true},
		observedVisualContactSmokeAlias(),
		observedVisualContactSmokeSHA256(),
		16,
		observedVisualContactSmokeAssets(),
	)
	if err == nil || !strings.Contains(err.Error(), "line_of_sight_blocked") {
		t.Fatalf("BuildObservedVisualContactSmokeReport() error = %v, want LOS coverage rejection", err)
	}
}

func TestBuildObservedVisualContactSmokeReportTriesLaterCausalPrefix(t *testing.T) {
	replay := observedVisualContactSmokeReplayFixture()
	validRound := replay.Rounds[0]
	validRound.Round = 2
	weakFrames := make([]models.ReplayFrame, 0, 16)
	for tick := 0; tick <= 60; tick += 4 {
		effects := []models.ReplayActiveEffect{}
		if tick >= 20 {
			effects = []models.ReplayActiveEffect{{
				Type: "smoke", X: 1000, Y: 1000, Radius: 20, StartTick: 20,
			}}
		}
		weakFrames = append(weakFrames, models.ReplayFrame{
			Tick: tick,
			Players: []models.ReplayPlayerState{
				observedVisualContactSmokePlayer(76561198000000011, "CT", 0, 0, 0),
				observedVisualContactSmokePlayer(76561198000000012, "T", 100, 0, 180),
			},
			ActiveEffects: effects,
		})
	}
	replay.Rounds = []models.ReplayRound{
		{Round: 1, StartTick: 0, EndTick: 60, Frames: weakFrames},
		validRound,
	}

	report, err := BuildObservedVisualContactSmokeReport(
		&replay,
		observedVisualContactSmokeGeometry{loaded: true},
		observedVisualContactSmokeAlias(),
		observedVisualContactSmokeSHA256(),
		16,
		observedVisualContactSmokeAssets(),
	)
	if err != nil {
		t.Fatalf("BuildObservedVisualContactSmokeReport() error = %v", err)
	}
	if report.Selection.RoundNumber != 2 || report.Selection.StartTick != validRound.StartTick {
		t.Fatalf("selection = %#v, want later valid round 2 causal prefix", report.Selection)
	}
}

func TestObservedVisualContactSmokeIsPrivateAndDeterministic(t *testing.T) {
	leftReplay := observedVisualContactSmokeReplayFixture()
	rightReplay := observedVisualContactSmokeReplayFixture()
	reverseObservedVisualContactSmokeFixture(&rightReplay)

	left, err := BuildObservedVisualContactSmokeReport(
		&leftReplay,
		observedVisualContactSmokeGeometry{loaded: true},
		observedVisualContactSmokeAlias(),
		observedVisualContactSmokeSHA256(),
		16,
		observedVisualContactSmokeAssets(),
	)
	if err != nil {
		t.Fatalf("left BuildObservedVisualContactSmokeReport() error = %v", err)
	}
	assets := observedVisualContactSmokeAssets()
	assets[0], assets[1] = assets[1], assets[0]
	right, err := BuildObservedVisualContactSmokeReport(
		&rightReplay,
		observedVisualContactSmokeGeometry{loaded: true},
		observedVisualContactSmokeAlias(),
		observedVisualContactSmokeSHA256(),
		16,
		assets,
	)
	if err != nil {
		t.Fatalf("right BuildObservedVisualContactSmokeReport() error = %v", err)
	}

	leftJSON, err := json.Marshal(left)
	if err != nil {
		t.Fatalf("json.Marshal(left): %v", err)
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		t.Fatalf("json.Marshal(right): %v", err)
	}
	if string(leftJSON) != string(rightJSON) {
		t.Fatalf("visual contact smoke is not deterministic\nleft:  %s\nright: %s", leftJSON, rightJSON)
	}
	for _, forbidden := range []string{
		"76561198000000001",
		"76561198000000002",
		"76561198000000003",
		"76561198000000004",
		"steam:",
		"Raw Player",
	} {
		if strings.Contains(string(leftJSON), forbidden) {
			t.Fatalf("serialized smoke leaked %q", forbidden)
		}
	}
	for _, evaluation := range left.Ledger.Evaluations {
		if !strings.HasPrefix(evaluation.JoinKeys.ObserverID, "participant-") ||
			!strings.HasPrefix(evaluation.JoinKeys.SubjectID, "participant-") {
			t.Fatalf("non-pseudonymous evaluation join keys: %#v", evaluation.JoinKeys)
		}
		if evaluation.JoinKeys.ContactID != nil && !strings.HasPrefix(*evaluation.JoinKeys.ContactID, "contact-") {
			t.Fatalf("non-pseudonymous contact join key: %#v", evaluation.JoinKeys)
		}
	}

	tampered := left
	tampered.Ledger.Evaluations = append([]models.ObservedVisualContactEvaluation(nil), left.Ledger.Evaluations...)
	tampered.Ledger.Evaluations[0].JoinKeys.ObserverID = "steam:76561198000000001"
	tampered.Summary = summarizeObservedVisualContactSmoke(tampered.Ledger)
	if err := ValidateObservedVisualContactSmokeReport(tampered); err == nil ||
		!strings.Contains(err.Error(), "non-pseudonymous") {
		t.Fatalf("raw identity tampering was accepted: %v", err)
	}
}

func observedVisualContactSmokeReplayFixture() models.ReplayData {
	frames := make([]models.ReplayFrame, 0, 17)
	for tick := 100; tick <= 164; tick += 4 {
		effects := []models.ReplayActiveEffect{}
		if tick >= 140 {
			effects = []models.ReplayActiveEffect{{
				Type: "smoke", X: 50, Y: 0, Radius: 20, StartTick: 140,
			}}
		}
		frames = append(frames, models.ReplayFrame{
			Tick: tick,
			Players: []models.ReplayPlayerState{
				observedVisualContactSmokePlayer(76561198000000001, "CT", 0, 0, 0),
				observedVisualContactSmokePlayer(76561198000000002, "T", 100, 0, 180),
				observedVisualContactSmokePlayer(76561198000000003, "T", 100, 200, -116),
				observedVisualContactSmokePlayer(76561198000000004, "T", -100, 0, 0),
			},
			ActiveEffects: effects,
		})
	}
	return models.ReplayData{
		Metadata: models.ReplayMetadata{
			SchemaVersion: 5,
			MapName:       "de_smoke_fixture",
			TickRate:      64,
			SampleRate:    62,
		},
		Rounds: []models.ReplayRound{{
			Round: 1, StartTick: 100, EndTick: 164, Frames: frames,
		}},
	}
}

func observedVisualContactSmokePlayer(
	steamID uint64,
	team string,
	x, y float64,
	yaw float32,
) models.ReplayPlayerState {
	return models.ReplayPlayerState{
		SteamID:              steamID,
		Name:                 "Raw Player " + team,
		Team:                 team,
		Alive:                true,
		Weapon:               "AK-47",
		Yaw:                  yaw,
		Pitch:                0,
		EyeX:                 x,
		EyeY:                 y,
		EyeZ:                 64,
		EyePositionAvailable: true,
		EyePositionSource:    "native_eye_position",
		ViewAnglesAvailable:  true,
		ZoomLevelAvailable:   true,
	}
}

func observedVisualContactSmokeAlias() string {
	return "demo-" + observedVisualContactSmokeSHA256()[:20]
}

func observedVisualContactSmokeSHA256() string {
	return strings.Repeat("a", 64)
}

func observedVisualContactSmokeAssets() []ObservedVisualContactSmokeAsset {
	return []ObservedVisualContactSmokeAsset{
		{
			Reference: "de_smoke_fixture/de_smoke_fixture_physics.gltf",
			SHA256:    strings.Repeat("b", 64),
			Role:      ObservedVisualContactSmokePhysicsGLTFRole,
		},
		{
			Reference: "de_smoke_fixture/de_smoke_fixture_physics.bin",
			SHA256:    strings.Repeat("c", 64),
			Role:      ObservedVisualContactSmokePhysicsBufferRole,
		},
	}
}

func reverseObservedVisualContactSmokeFixture(replay *models.ReplayData) {
	frames := replay.Rounds[0].Frames
	for left, right := 0, len(frames)-1; left < right; left, right = left+1, right-1 {
		frames[left], frames[right] = frames[right], frames[left]
	}
	for frameIndex := range frames {
		players := frames[frameIndex].Players
		for left, right := 0, len(players)-1; left < right; left, right = left+1, right-1 {
			players[left], players[right] = players[right], players[left]
		}
		effects := frames[frameIndex].ActiveEffects
		for left, right := 0, len(effects)-1; left < right; left, right = left+1, right-1 {
			effects[left], effects[right] = effects[right], effects[left]
		}
	}
}
