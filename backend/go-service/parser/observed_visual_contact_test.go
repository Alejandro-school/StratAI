package parser

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"

	"github.com/golang/geo/r3"
)

func TestObservedVisualContactRequiresFOVInAdditionToGeometricLOS(t *testing.T) {
	tests := []struct {
		name        string
		targetEyes  r3.Vector
		wantContact bool
	}{
		{name: "inside FOV", targetEyes: r3.Vector{X: 100, Z: 64}, wantContact: true},
		{name: "outside FOV", targetEyes: r3.Vector{Y: 100, Z: 64}, wantContact: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			replay := visualContactTestReplay(100, 100, []models.ReplayFrame{{
				Tick: 100,
				Players: []models.ReplayPlayerState{
					visualContactTestPlayer(1, "observer-name", "CT", r3.Vector{Z: 64}, 0),
					visualContactTestPlayer(2, "subject-name", "T", test.targetEyes, 180),
				},
				ActiveEffects: []models.ReplayActiveEffect{},
			}})
			geometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
			ledger, err := BuildObservedVisualContactLedger(&replay, geometry, 100)
			if err != nil {
				t.Fatalf("BuildObservedVisualContactLedger() error = %v", err)
			}
			evaluation := visualContactEvaluation(t, ledger, "steam:1", "steam:2", 100)
			if evaluation.LineOfSight.Value == nil || !*evaluation.LineOfSight.Value {
				t.Fatal("geometric LOS should be true in this fixture")
			}
			if evaluation.FieldOfView.Inside == nil || *evaluation.FieldOfView.Inside != test.wantContact {
				t.Fatalf("inside FOV = %v, want %v", evaluation.FieldOfView.Inside, test.wantContact)
			}
			if evaluation.VisualContact.Value == nil || *evaluation.VisualContact.Value != test.wantContact {
				t.Fatalf("visual contact = %v, want %v", evaluation.VisualContact.Value, test.wantContact)
			}
		})
	}
}

func TestObservedVisualContactDistinguishesClearAndBlockedRaycasts(t *testing.T) {
	for _, visible := range []bool{true, false} {
		t.Run(map[bool]string{true: "clear", false: "blocked"}[visible], func(t *testing.T) {
			replay := visualContactSingleFrameReplay([]models.ReplayActiveEffect{})
			geometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return visible }}
			ledger, err := BuildObservedVisualContactLedger(&replay, geometry, 100)
			if err != nil {
				t.Fatalf("BuildObservedVisualContactLedger() error = %v", err)
			}
			evaluation := visualContactEvaluation(t, ledger, "steam:1", "steam:2", 100)
			if evaluation.LineOfSight.Value == nil || *evaluation.LineOfSight.Value != visible {
				t.Fatalf("line of sight = %v, want %v", evaluation.LineOfSight.Value, visible)
			}
			if evaluation.VisualContact.Value == nil || *evaluation.VisualContact.Value != visible {
				t.Fatalf("visual contact = %v, want %v", evaluation.VisualContact.Value, visible)
			}
		})
	}
}

func TestObservedVisualContactDetectsSmokeBetweenObserverAndTarget(t *testing.T) {
	replay := visualContactSingleFrameReplay([]models.ReplayActiveEffect{{
		Type: "smoke", X: 50, Y: 0, Radius: 20, StartTick: 90,
	}})
	geometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	ledger, err := BuildObservedVisualContactLedger(&replay, geometry, 100)
	if err != nil {
		t.Fatalf("BuildObservedVisualContactLedger() error = %v", err)
	}
	evaluation := visualContactEvaluation(t, ledger, "steam:1", "steam:2", 100)
	if evaluation.SmokeOcclusion.Occluded == nil || !*evaluation.SmokeOcclusion.Occluded {
		t.Fatalf("smoke occlusion = %v, want true", evaluation.SmokeOcclusion.Occluded)
	}
	if evaluation.VisualContact.Value == nil || *evaluation.VisualContact.Value {
		t.Fatalf("smoked visual contact = %v, want false", evaluation.VisualContact.Value)
	}
}

func TestObservedVisualContactRejectsOcclusionWithoutActiveSmoke(t *testing.T) {
	replay := visualContactSingleFrameReplay([]models.ReplayActiveEffect{{
		Type: "smoke", X: 50, Y: 0, Radius: 20, StartTick: 90,
	}})
	geometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	ledger := mustBuildVisualContactLedger(t, &replay, geometry, 100)
	zero := 0
	ledger.Evaluations[0].SmokeOcclusion.ActiveSmokeCount = &zero
	if err := ValidateObservedVisualContactLedger(ledger); err == nil ||
		!strings.Contains(err.Error(), "occlusion requires at least one active smoke") {
		t.Fatalf("occlusion without active smoke was accepted: %v", err)
	}
}

func TestObservedVisualContactAbstainsWhenRequiredInputsAreMissing(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*models.ReplayData)
		geometry   TacticalGeometry
		wantReason string
	}{
		{
			name: "geometry", geometry: nil, wantReason: "geometry_unavailable",
			mutate: func(*models.ReplayData) {},
		},
		{
			name:       "orientation",
			geometry:   &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }},
			wantReason: "observer_orientation_unavailable",
			mutate: func(replay *models.ReplayData) {
				replay.Rounds[0].Frames[0].Players[0].ViewAnglesAvailable = false
			},
		},
		{
			name:       "smoke snapshot",
			geometry:   &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }},
			wantReason: "smoke_snapshot_unavailable",
			mutate: func(replay *models.ReplayData) {
				replay.Rounds[0].Frames[0].ActiveEffects = nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			replay := visualContactSingleFrameReplay([]models.ReplayActiveEffect{})
			test.mutate(&replay)
			ledger, err := BuildObservedVisualContactLedger(&replay, test.geometry, 100)
			if err != nil {
				t.Fatalf("BuildObservedVisualContactLedger() error = %v", err)
			}
			evaluation := visualContactEvaluation(t, ledger, "steam:1", "steam:2", 100)
			if evaluation.Status != models.TacticalStatusUnavailable || evaluation.VisualContact.Value != nil {
				t.Fatalf("missing input invented contact: %+v", evaluation)
			}
			if !containsVisualContactReason(evaluation.AbstentionReasons, test.wantReason) {
				t.Fatalf("reasons = %v, want %q", evaluation.AbstentionReasons, test.wantReason)
			}
		})
	}
}

func TestObservedVisualContactRejectsAvailabilityAfterT0(t *testing.T) {
	replay := visualContactSingleFrameReplay([]models.ReplayActiveEffect{{
		Type: "smoke", X: 50, Y: 0, Radius: 20, StartTick: 101,
	}})
	geometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	if _, err := BuildObservedVisualContactLedger(&replay, geometry, 100); err == nil || !strings.Contains(err.Error(), "availability") {
		t.Fatalf("future smoke availability was accepted: %v", err)
	}

	replay = visualContactSingleFrameReplay([]models.ReplayActiveEffect{})
	ledger, err := BuildObservedVisualContactLedger(&replay, geometry, 100)
	if err != nil {
		t.Fatalf("BuildObservedVisualContactLedger() error = %v", err)
	}
	late := 101
	ledger.Evaluations[0].AvailabilityTick = &late
	if err := ValidateObservedVisualContactLedger(ledger); err == nil || !strings.Contains(err.Error(), "availability_tick") {
		t.Fatalf("future row availability was accepted: %v", err)
	}
}

func TestObservedVisualContactIsInvariantToFutureMutation(t *testing.T) {
	base := visualContactTestReplay(100, 104, []models.ReplayFrame{
		{
			Tick: 100,
			Players: []models.ReplayPlayerState{
				visualContactTestPlayer(1, "observer", "CT", r3.Vector{Z: 64}, 0),
				visualContactTestPlayer(2, "subject", "T", r3.Vector{X: 100, Z: 64}, 180),
			},
			ActiveEffects: []models.ReplayActiveEffect{},
		},
		{
			Tick: 104,
			Players: []models.ReplayPlayerState{
				visualContactTestPlayer(1, "observer-future", "CT", r3.Vector{X: 999, Z: 64}, 90),
				visualContactTestPlayer(2, "subject-future", "T", r3.Vector{Y: -999, Z: 64}, 270),
			},
			ActiveEffects: []models.ReplayActiveEffect{{Type: "smoke", X: 1, Radius: 144, StartTick: 104}},
		},
	})
	mutated := base
	mutated.Rounds = append([]models.ReplayRound(nil), base.Rounds...)
	mutated.Rounds[0].Frames = append([]models.ReplayFrame(nil), base.Rounds[0].Frames...)
	mutated.Rounds[0].Frames[1].Players = []models.ReplayPlayerState{
		visualContactTestPlayer(1, "changed", "CT", r3.Vector{X: -5000, Z: 64}, 180),
		visualContactTestPlayer(2, "changed-again", "T", r3.Vector{X: 5000, Z: 64}, 0),
	}
	mutated.Rounds[0].Frames[1].ActiveEffects = []models.ReplayActiveEffect{{
		Type: "smoke", X: math.NaN(), Radius: -1, StartTick: 9999,
	}}
	geometryLeft := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	geometryRight := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	left := mustBuildVisualContactLedger(t, &base, geometryLeft, 100)
	right := mustBuildVisualContactLedger(t, &mutated, geometryRight, 100)
	assertVisualContactJSONEqual(t, left, right)
}

func TestObservedVisualContactIsInvariantToOracleOnlyMutation(t *testing.T) {
	leftReplay := visualContactSingleFrameReplay([]models.ReplayActiveEffect{})
	rightReplay := visualContactSingleFrameReplay([]models.ReplayActiveEffect{})
	for index := range rightReplay.Rounds[0].Frames[0].Players {
		player := &rightReplay.Rounds[0].Frames[0].Players[index]
		player.Name = "private-name-changed"
		player.Health = 1
		player.Armor = 99
		player.Money = 16000
		player.Weapons = []string{"oracle_inventory_change"}
		player.HasC4 = !player.HasC4
		player.HasHelmet = !player.HasHelmet
		player.HasDefuseKit = !player.HasDefuseKit
	}
	leftGeometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	rightGeometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	left := mustBuildVisualContactLedger(t, &leftReplay, leftGeometry, 100)
	right := mustBuildVisualContactLedger(t, &rightReplay, rightGeometry, 100)
	assertVisualContactJSONEqual(t, left, right)
}

func TestObservedVisualContactKeepsIDsJoinOnlyAndIsDeterministic(t *testing.T) {
	leftReplay := visualContactTestReplay(100, 100, []models.ReplayFrame{{
		Tick: 100,
		Players: []models.ReplayPlayerState{
			visualContactTestPlayer(2, "secret-subject", "T", r3.Vector{X: 100, Z: 64}, 180),
			visualContactTestPlayer(1, "secret-observer", "CT", r3.Vector{Z: 64}, 0),
		},
		ActiveEffects: []models.ReplayActiveEffect{
			{Type: "smoke", X: 1000, Radius: 50, StartTick: 90},
			{Type: "smoke", X: 2000, Radius: 50, StartTick: 80},
		},
	}})
	rightReplay := visualContactTestReplay(100, 100, []models.ReplayFrame{{
		Tick: 100,
		Players: []models.ReplayPlayerState{
			visualContactTestPlayer(1, "renamed-observer", "CT", r3.Vector{Z: 64}, 0),
			visualContactTestPlayer(2, "renamed-subject", "T", r3.Vector{X: 100, Z: 64}, 180),
		},
		ActiveEffects: []models.ReplayActiveEffect{
			{Type: "smoke", X: 2000, Radius: 50, StartTick: 80},
			{Type: "smoke", X: 1000, Radius: 50, StartTick: 90},
		},
	}})
	leftGeometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	rightGeometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	left := mustBuildVisualContactLedger(t, &leftReplay, leftGeometry, 100)
	right := mustBuildVisualContactLedger(t, &rightReplay, rightGeometry, 100)
	assertVisualContactJSONEqual(t, left, right)

	encoded, err := json.Marshal(left)
	if err != nil {
		t.Fatalf("json.Marshal(ledger) error = %v", err)
	}
	for _, forbidden := range []string{"secret-observer", "secret-subject", `"steam_id"`, `"name"`} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("ledger leaks forbidden identity field/value %q: %s", forbidden, encoded)
		}
	}
	var payload any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("json.Unmarshal(ledger) error = %v", err)
	}
	assertVisualContactIDsAreJoinOnly(t, payload, nil)
}

func TestObservedVisualContactPublishesCausalStartAndEndBoundaries(t *testing.T) {
	replay := visualContactTestReplay(100, 104, []models.ReplayFrame{
		{
			Tick: 100,
			Players: []models.ReplayPlayerState{
				visualContactTestPlayer(1, "observer", "CT", r3.Vector{Z: 64}, 0),
				visualContactTestPlayer(2, "subject", "T", r3.Vector{X: 100, Z: 64}, 180),
			},
			ActiveEffects: []models.ReplayActiveEffect{},
		},
		{
			Tick: 104,
			Players: []models.ReplayPlayerState{
				visualContactTestPlayer(1, "observer", "CT", r3.Vector{Z: 64}, 0),
				visualContactTestPlayer(2, "subject", "T", r3.Vector{X: -100, Z: 64}, 180),
			},
			ActiveEffects: []models.ReplayActiveEffect{},
		},
	})
	geometry := &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
	ledger := mustBuildVisualContactLedger(t, &replay, geometry, 104)
	started := visualContactEvaluation(t, ledger, "steam:1", "steam:2", 100)
	ended := visualContactEvaluation(t, ledger, "steam:1", "steam:2", 104)
	if started.Boundary != models.VisualContactBoundaryStarted || ended.Boundary != models.VisualContactBoundaryEnded {
		t.Fatalf("boundaries = %q/%q, want started/ended", started.Boundary, ended.Boundary)
	}
	episode := visualContactEpisode(t, ledger, "steam:1", "steam:2")
	if episode.StartTick != 100 || episode.EndTick == nil || *episode.EndTick != 104 ||
		episode.Status != models.VisualContactEpisodeClosed {
		t.Fatalf("unexpected contact episode: %+v", episode)
	}
}

func visualContactSingleFrameReplay(effects []models.ReplayActiveEffect) models.ReplayData {
	return visualContactTestReplay(100, 100, []models.ReplayFrame{{
		Tick: 100,
		Players: []models.ReplayPlayerState{
			visualContactTestPlayer(1, "observer", "CT", r3.Vector{Z: 64}, 0),
			visualContactTestPlayer(2, "subject", "T", r3.Vector{X: 100, Z: 64}, 180),
		},
		ActiveEffects: effects,
	}})
}

func visualContactTestReplay(startTick, endTick int, frames []models.ReplayFrame) models.ReplayData {
	return models.ReplayData{
		Metadata: models.ReplayMetadata{
			SchemaVersion: 1,
			MatchID:       "match-visual-contact-test",
			MapName:       "de_test",
			TickRate:      64,
		},
		Rounds: []models.ReplayRound{{
			Round: 1, StartTick: startTick, EndTick: endTick, Frames: frames,
		}},
	}
}

func visualContactTestPlayer(id uint64, name, team string, eyes r3.Vector, yaw float32) models.ReplayPlayerState {
	return models.ReplayPlayerState{
		SteamID: id, Name: name, Team: team, Alive: true, Health: 100, Armor: 100,
		Weapon: "weapon_knife", Money: 800, Yaw: yaw, Pitch: 0,
		EyeX: eyes.X, EyeY: eyes.Y, EyeZ: eyes.Z,
		EyePositionAvailable: true, EyePositionSource: "native_eye_position",
		ViewAnglesAvailable: true, ZoomLevelAvailable: true,
	}
}

func mustBuildVisualContactLedger(
	t *testing.T,
	replay *models.ReplayData,
	geometry TacticalGeometry,
	t0Tick int,
) models.ObservedVisualContactLedger {
	t.Helper()
	ledger, err := BuildObservedVisualContactLedger(replay, geometry, t0Tick)
	if err != nil {
		t.Fatalf("BuildObservedVisualContactLedger() error = %v", err)
	}
	return ledger
}

func visualContactEvaluation(
	t *testing.T,
	ledger models.ObservedVisualContactLedger,
	observerID, subjectID string,
	tick int,
) models.ObservedVisualContactEvaluation {
	t.Helper()
	for _, evaluation := range ledger.Evaluations {
		if evaluation.JoinKeys.ObserverID == observerID && evaluation.JoinKeys.SubjectID == subjectID && evaluation.Tick == tick {
			return evaluation
		}
	}
	t.Fatalf("visual contact evaluation %s -> %s at %d not found", observerID, subjectID, tick)
	return models.ObservedVisualContactEvaluation{}
}

func visualContactEpisode(
	t *testing.T,
	ledger models.ObservedVisualContactLedger,
	observerID, subjectID string,
) models.ObservedVisualContactEpisode {
	t.Helper()
	for _, episode := range ledger.Contacts {
		if episode.JoinKeys.ObserverID == observerID && episode.JoinKeys.SubjectID == subjectID {
			return episode
		}
	}
	t.Fatalf("visual contact episode %s -> %s not found", observerID, subjectID)
	return models.ObservedVisualContactEpisode{}
}

func assertVisualContactJSONEqual(
	t *testing.T,
	left, right models.ObservedVisualContactLedger,
) {
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
		t.Fatalf("visual contact ledgers differ\nleft:  %s\nright: %s", leftJSON, rightJSON)
	}
}

func assertVisualContactIDsAreJoinOnly(t *testing.T, value any, path []string) {
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
			assertVisualContactIDsAreJoinOnly(t, child, childPath)
		}
	case []any:
		for index, child := range typed {
			assertVisualContactIDsAreJoinOnly(t, child, append(path, "[]"))
			_ = index
		}
	}
}
