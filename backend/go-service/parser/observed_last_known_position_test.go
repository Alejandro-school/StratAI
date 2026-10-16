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

func TestObservedLastKnownPositionUpdatesCarriesAndPreservesZero(t *testing.T) {
	replay := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(100, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{}, r3.Vector{X: 100, Z: 64}, 180, true), true),
		lastKnownFrame(104, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 20}, r3.Vector{X: 100, Z: 64}, 180, true), true),
		lastKnownFrame(108, lastKnownPlayer(1, "CT", r3.Vector{X: 999}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 9000}, r3.Vector{X: -100, Z: 64}, 180, true), true),
		lastKnownFrame(112, lastKnownPlayer(1, "CT", r3.Vector{X: -999}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: -9000}, r3.Vector{X: -100, Z: 64}, 180, true), false),
	})
	ledger := mustBuildLastKnownLedger(t, &replay)
	rows := lastKnownRowsForPair(ledger, "participant-001", "participant-002")
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(rows))
	}
	if rows[0].Derivation != models.ObservedLastKnownPositionDerivationUpdate ||
		rows[0].LastKnownPosition.Value == nil || *rows[0].LastKnownPosition.Value != (models.ObservedLastKnownPositionVector{}) ||
		rows[0].InformationAge.ValueMS == nil || *rows[0].InformationAge.ValueMS != 0 {
		t.Fatalf("first zero-valued contact was not preserved: %+v", rows[0])
	}
	if rows[1].LastKnownPosition.Value == nil || rows[1].LastKnownPosition.Value.X != 20 ||
		rows[1].LastKnownPosition.ObservedTick == nil || *rows[1].LastKnownPosition.ObservedTick != 104 {
		t.Fatalf("continued contact did not refresh: %+v", rows[1])
	}
	for index, wantAge := range map[int]float64{2: 62.5, 3: 125} {
		row := rows[index]
		if row.Derivation != models.ObservedLastKnownPositionDerivationCarry ||
			row.LastKnownPosition.Value == nil || row.LastKnownPosition.Value.X != 20 ||
			row.InformationAge.ValueMS == nil || *row.InformationAge.ValueMS != wantAge ||
			row.ContactObservation.SubjectPositionAtContact != nil {
			t.Fatalf("row %d did not carry causally: %+v", index, row)
		}
	}
	if rows[2].ContactObservation.VisualContact == nil || *rows[2].ContactObservation.VisualContact ||
		rows[3].ContactObservation.Status != models.ObservedLastKnownPositionStatusUnavailable {
		t.Fatalf("negative/unavailable contact semantics changed: %+v %+v", rows[2], rows[3])
	}
}

func TestObservedLastKnownPositionAbstainsBeforeContactAndOnBadPosition(t *testing.T) {
	replay := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(100, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 180, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 10}, r3.Vector{X: 100, Z: 64}, 180, true), true),
		lastKnownFrame(104, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 20}, r3.Vector{X: 100, Z: 64}, 180, true), true),
	})
	ledger := mustBuildLastKnownLedger(t, &replay)
	rows := lastKnownRowsForPair(ledger, "participant-001", "participant-002")
	if rows[0].LastKnownPosition.Status != models.ObservedLastKnownPositionStatusUnavailable ||
		rows[0].LastKnownPosition.Value != nil || rows[0].InformationAge.ValueMS != nil ||
		!containsLastKnownString(rows[0].AbstentionReasons, lastKnownReasonNoPriorContact) {
		t.Fatalf("pre-contact row invented information: %+v", rows[0])
	}
	if rows[1].LastKnownPosition.Status != models.ObservedLastKnownPositionStatusObserved {
		t.Fatalf("later true contact did not establish state: %+v", rows[1])
	}

	bad := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(100, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{Z: math.NaN()}, r3.Vector{X: 100, Z: 64}, 180, true), true),
	})
	badLedger := mustBuildLastKnownLedger(t, &bad)
	badRow := lastKnownRowsForPair(badLedger, "participant-001", "participant-002")[0]
	if badRow.LastKnownPosition.Status != models.ObservedLastKnownPositionStatusUnavailable ||
		!containsLastKnownString(badRow.AbstentionReasons, lastKnownReasonPositionNonFinite) {
		t.Fatalf("non-finite contact position was accepted: %+v", badRow)
	}
}

func TestObservedLastKnownPositionResetsAfterDeathRespawn(t *testing.T) {
	replay := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(100, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 10}, r3.Vector{X: 100, Z: 64}, 180, true), true),
		lastKnownFrame(104, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 10}, r3.Vector{X: 100, Z: 64}, 180, false), true),
		lastKnownFrame(108, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 999}, r3.Vector{X: -100, Z: 64}, 180, true), true),
	})
	ledger := mustBuildLastKnownLedger(t, &replay)
	rows := lastKnownRowsForPair(ledger, "participant-001", "participant-002")
	if len(rows) != 2 || rows[0].LastKnownPosition.Status != models.ObservedLastKnownPositionStatusObserved ||
		rows[1].LastKnownPosition.Status != models.ObservedLastKnownPositionStatusUnavailable ||
		rows[1].ContactObservation.SubjectLifeSequence == nil || *rows[1].ContactObservation.SubjectLifeSequence != 2 {
		t.Fatalf("death/respawn carried stale position: %+v", rows)
	}
}

func TestObservedLastKnownPositionDirectedPairsAreIndependent(t *testing.T) {
	replay := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(100, lastKnownPlayer(1, "CT", r3.Vector{X: 1}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 2}, r3.Vector{X: 100, Z: 64}, 180, true), true),
	})
	ledger := mustBuildLastKnownLedger(t, &replay)
	forward := lastKnownRowsForPair(ledger, "participant-001", "participant-002")
	reverse := lastKnownRowsForPair(ledger, "participant-002", "participant-001")
	if len(forward) != 1 || len(reverse) != 1 || forward[0].LastKnownPosition.Value == nil ||
		reverse[0].LastKnownPosition.Value == nil || forward[0].LastKnownPosition.Value.X != 2 ||
		reverse[0].LastKnownPosition.Value.X != 1 {
		t.Fatalf("directed states were conflated: forward=%+v reverse=%+v", forward, reverse)
	}
}

func TestObservedLastKnownPositionOrdersByOpaqueAlias(t *testing.T) {
	replay := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(100, lastKnownPlayer(10, "T", r3.Vector{X: 10}, r3.Vector{X: -100, Z: 64}, 180, true),
			lastKnownPlayer(2, "CT", r3.Vector{X: 2}, r3.Vector{X: 100, Z: 64}, 0, true), true),
	})
	visual := mustBuildVisualContactLedger(t, &replay, lastKnownGeometry(), 100)
	ledger, err := BuildObservedLastKnownPositionLedger(
		&replay,
		visual,
		map[string]string{
			"steam:2":  "participant-002",
			"steam:10": "participant-010",
		},
	)
	if err != nil {
		t.Fatalf("BuildObservedLastKnownPositionLedger() error = %v", err)
	}
	if len(ledger.Rows) != 2 ||
		ledger.Rows[0].JoinKeys.ObserverAlias != "participant-002" ||
		ledger.Rows[1].JoinKeys.ObserverAlias != "participant-010" {
		t.Fatalf("rows were not ordered by public aliases: %+v", ledger.Rows)
	}
}

func TestObservedLastKnownPositionIgnoresHiddenCurrentAndFutureMutation(t *testing.T) {
	base := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(100, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 10}, r3.Vector{X: 100, Z: 64}, 180, true), true),
		lastKnownFrame(104, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 1000}, r3.Vector{X: -100, Z: 64}, 180, true), true),
		lastKnownFrame(108, lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
			lastKnownPlayer(2, "T", r3.Vector{X: 2000}, r3.Vector{X: -100, Z: 64}, 180, true), true),
	})
	mutated := cloneLastKnownReplay(t, base)
	mutated.Rounds[0].Frames[1].Players[1].X = -7000
	mutated.Rounds[0].Frames[1].Players[1].Name = "hidden-name"
	mutated.Rounds[0].Frames[2].Players[1].X = 9000
	mutated.Rounds[0].Frames[2].Players[1].Health = 1
	left := mustBuildLastKnownLedger(t, &base)
	right := mustBuildLastKnownLedger(t, &mutated)
	if !reflect.DeepEqual(left, right) {
		leftJSON, _ := json.Marshal(left)
		rightJSON, _ := json.Marshal(right)
		t.Fatalf("hidden/future mutation changed causal output\nleft=%s\nright=%s", leftJSON, rightJSON)
	}
}

func TestObservedLastKnownPositionIsDeterministicPrivateAndFailsClosed(t *testing.T) {
	base := lastKnownReplay([]models.ReplayFrame{
		lastKnownFrame(104, lastKnownPlayer(2, "T", r3.Vector{X: 20}, r3.Vector{X: -100, Z: 64}, 180, true),
			lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true), true),
		lastKnownFrame(100, lastKnownPlayer(2, "T", r3.Vector{X: 10}, r3.Vector{X: 100, Z: 64}, 180, true),
			lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true), true),
	})
	left := mustBuildLastKnownLedger(t, &base)
	permuted := cloneLastKnownReplay(t, base)
	permuted.Rounds[0].Frames[0], permuted.Rounds[0].Frames[1] = permuted.Rounds[0].Frames[1], permuted.Rounds[0].Frames[0]
	for index := range permuted.Rounds[0].Frames {
		players := permuted.Rounds[0].Frames[index].Players
		players[0], players[1] = players[1], players[0]
	}
	right := mustBuildLastKnownLedger(t, &permuted)
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	if !reflect.DeepEqual(leftJSON, rightJSON) {
		t.Fatalf("causally equivalent permutations differ\nleft=%s\nright=%s", leftJSON, rightJSON)
	}
	for _, forbidden := range []string{"secret", "steam_id", "observer_id", "subject_id", "entity_id", "serial"} {
		if strings.Contains(strings.ToLower(string(leftJSON)), forbidden) {
			t.Fatalf("candidate leaked forbidden identity %q: %s", forbidden, leftJSON)
		}
	}

	visual := mustBuildVisualContactLedger(t, &base, lastKnownGeometry(), 104)
	visual.Evaluations = append(visual.Evaluations, visual.Evaluations[0])
	if _, err := BuildObservedLastKnownPositionLedger(&base, visual, lastKnownAliases()); err == nil {
		t.Fatal("duplicate evaluation was accepted")
	}
	badRate := cloneLastKnownReplay(t, base)
	badRate.Metadata.TickRate = math.NaN()
	if _, err := BuildObservedLastKnownPositionLedger(&badRate, models.ObservedVisualContactLedger{}, lastKnownAliases()); err == nil {
		t.Fatal("non-finite tick rate was accepted")
	}

	ambiguous := lastKnownReplay([]models.ReplayFrame{lastKnownFrame(
		100,
		lastKnownPlayer(1, "CT", r3.Vector{}, r3.Vector{Z: 64}, 0, true),
		lastKnownPlayer(2, "T", r3.Vector{X: 10}, r3.Vector{X: 100, Z: 64}, 180, true),
		true,
	)})
	ambiguous.Rounds[0].Frames[0].Players = append(
		ambiguous.Rounds[0].Frames[0].Players,
		lastKnownPlayer(2, "T", r3.Vector{X: 11}, r3.Vector{X: 100, Z: 64}, 180, true),
	)
	ambiguousLedger := mustBuildLastKnownLedger(t, &ambiguous)
	row := lastKnownRowsForPair(ambiguousLedger, "participant-001", "participant-002")[0]
	if row.LastKnownPosition.Status != models.ObservedLastKnownPositionStatusUnavailable ||
		!containsLastKnownString(row.AbstentionReasons, lastKnownReasonIdentityAmbiguous) {
		t.Fatalf("ambiguous position join was accepted: %+v", row)
	}
}

func mustBuildLastKnownLedger(t *testing.T, replay *models.ReplayData) models.ObservedLastKnownPositionLedger {
	t.Helper()
	visual := mustBuildVisualContactLedger(t, replay, lastKnownGeometry(), replay.Rounds[0].EndTick)
	ledger, err := BuildObservedLastKnownPositionLedger(replay, visual, lastKnownAliases())
	if err != nil {
		t.Fatalf("BuildObservedLastKnownPositionLedger() error = %v", err)
	}
	return ledger
}

func lastKnownRowsForPair(
	ledger models.ObservedLastKnownPositionLedger,
	observer, subject string,
) []models.ObservedLastKnownPositionRow {
	rows := make([]models.ObservedLastKnownPositionRow, 0)
	for _, row := range ledger.Rows {
		if row.JoinKeys.ObserverAlias == observer && row.JoinKeys.SubjectAlias == subject {
			rows = append(rows, row)
		}
	}
	return rows
}

func lastKnownReplay(frames []models.ReplayFrame) models.ReplayData {
	startTick, endTick := frames[0].Tick, frames[0].Tick
	for _, frame := range frames[1:] {
		if frame.Tick < startTick {
			startTick = frame.Tick
		}
		if frame.Tick > endTick {
			endTick = frame.Tick
		}
	}
	return models.ReplayData{
		Metadata: models.ReplayMetadata{
			SchemaVersion: 1,
			MatchID:       "demo-0123456789abcdefabcd",
			MapName:       "de_test",
			TickRate:      64,
		},
		Rounds: []models.ReplayRound{{Round: 1, StartTick: startTick, EndTick: endTick, Frames: frames}},
	}
}

func lastKnownFrame(
	tick int,
	observer, subject models.ReplayPlayerState,
	smokeAvailable bool,
) models.ReplayFrame {
	var effects []models.ReplayActiveEffect
	if smokeAvailable {
		effects = []models.ReplayActiveEffect{}
	}
	return models.ReplayFrame{Tick: tick, Players: []models.ReplayPlayerState{observer, subject}, ActiveEffects: effects}
}

func lastKnownPlayer(
	id uint64,
	team string,
	pawn, eyes r3.Vector,
	yaw float32,
	alive bool,
) models.ReplayPlayerState {
	return models.ReplayPlayerState{
		SteamID: id,
		Name:    "secret-player",
		Team:    team,
		X:       int(pawn.X), Y: int(pawn.Y), Z: pawn.Z,
		Yaw: yaw, Pitch: 0, Alive: alive, Health: 100,
		Weapon: "weapon_knife",
		EyeX:   eyes.X, EyeY: eyes.Y, EyeZ: eyes.Z,
		EyePositionAvailable: true, EyePositionSource: "native_eye_position",
		ViewAnglesAvailable: true, ZoomLevelAvailable: true,
	}
}

func lastKnownGeometry() TacticalGeometry {
	return &tacticalGeometryStub{loaded: true, visible: func(_, _ r3.Vector) bool { return true }}
}

func lastKnownAliases() map[string]string {
	return map[string]string{"steam:1": "participant-001", "steam:2": "participant-002"}
}

func cloneLastKnownReplay(t *testing.T, value models.ReplayData) models.ReplayData {
	t.Helper()
	result := value
	result.Rounds = append([]models.ReplayRound(nil), value.Rounds...)
	for roundIndex := range value.Rounds {
		result.Rounds[roundIndex].Frames = append(
			[]models.ReplayFrame(nil), value.Rounds[roundIndex].Frames...,
		)
		for frameIndex := range value.Rounds[roundIndex].Frames {
			original := value.Rounds[roundIndex].Frames[frameIndex]
			target := &result.Rounds[roundIndex].Frames[frameIndex]
			target.Players = make([]models.ReplayPlayerState, len(original.Players))
			copy(target.Players, original.Players)
			if original.ActiveEffects != nil {
				target.ActiveEffects = make([]models.ReplayActiveEffect, len(original.ActiveEffects))
				copy(target.ActiveEffects, original.ActiveEffects)
			}
		}
	}
	return result
}
