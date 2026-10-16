package handlers

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/objective"
)

func TestReplayObjectiveC4SameTickUsesLedgerSequence(t *testing.T) {
	ledger := []objective.Event{
		{ID: "start", Sequence: 10, Type: objective.EventPlantStart, Round: 1, Tick: 104},
		{ID: "abort", Sequence: 11, Type: objective.EventPlantAbort, Round: 1, Tick: 104},
	}
	actual := projectReplayObjectiveMarkers(nil, ledger, 1)
	if len(actual) != 2 || actual[0].ID != "start" || actual[1].ID != "abort" {
		t.Fatalf("same-tick causal order changed: %+v", actual)
	}
}

func TestReplayObjectiveC4MarkerPreservesEveryObservedTransition(t *testing.T) {
	tests := []struct {
		eventType objective.EventType
		state     objective.State
		phase     objective.Phase
		side      string
	}{
		{objective.EventCarrierSnapshot, objective.StateCarried, objective.PhasePreplant, "T"},
		{objective.EventDrop, objective.StateDropped, objective.PhasePreplant, "T"},
		{objective.EventPickup, objective.StateCarried, objective.PhasePreplant, "T"},
		{objective.EventPlantStart, objective.StatePlanting, objective.PhasePlanting, "T"},
		{objective.EventPlantAbort, objective.StateCarried, objective.PhasePreplant, "T"},
		{objective.EventPlant, objective.StatePlanted, objective.PhasePlanted, "T"},
		{objective.EventDefuseStart, objective.StateDefusing, objective.PhaseDefusing, "CT"},
		{objective.EventDefuseAbort, objective.StatePlanted, objective.PhasePlanted, "CT"},
		{objective.EventDefuse, objective.StateDefused, objective.PhaseResolved, "CT"},
		{objective.EventExplode, objective.StateExploded, objective.PhaseResolved, ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			source := objective.Event{
				ID: "objective-" + string(tt.eventType), Type: tt.eventType,
				Round: 4, Tick: 8123, Actor: objective.Actor{SteamID: 77, Side: tt.side},
				Site: "B", Source: objective.SourceDemoinfocsEvent,
				StateAfter: tt.state, PhaseAfter: tt.phase,
				Position: objective.UnavailablePosition(objective.SourceDemoinfocsEvent),
			}
			if tt.eventType == objective.EventCarrierSnapshot {
				source.Source = objective.SourceDemoinfocsNativeSnapshot
			}
			if tt.eventType == objective.EventExplode {
				source.Actor = objective.Actor{}
			}

			marker, ok := replayObjectiveMarker(source)
			if !ok {
				t.Fatalf("observed %s transition disappeared from replay", tt.eventType)
			}
			if marker.ID != source.ID || marker.Tick != source.Tick || marker.Type != "bomb_"+string(tt.eventType) ||
				marker.PlayerID != source.Actor.SteamID || marker.Site != source.Site {
				t.Fatalf("replay changed the objective event identity: %+v", marker)
			}
			if !reflect.DeepEqual(marker.SourceEventIDs, []string{source.ID}) {
				t.Errorf("source event provenance lost: got %v", marker.SourceEventIDs)
			}
			payload := replayC4JSON(t, marker)
			for key, want := range map[string]any{
				"availability_tick": float64(source.Tick),
				"state_after":       string(tt.state), "objective_phase": string(tt.phase), "source": source.Source,
			} {
				if payload[key] != want {
					t.Errorf("%s = %v, want %v", key, payload[key], want)
				}
			}
			if tt.side != "" && payload["player_team"] != tt.side {
				t.Errorf("player_team = %v, want %s", payload["player_team"], tt.side)
			}
			if tt.eventType == objective.EventExplode && marker.DurationMS != 1200 {
				t.Errorf("explosion visual duration = %d, want 1200", marker.DurationMS)
			}
		})
	}
}

func TestReplayObjectiveC4ProjectionReplacesRawTransitionsWithSameRoundLedger(t *testing.T) {
	eventTypes := []objective.EventType{
		objective.EventCarrierSnapshot, objective.EventDrop, objective.EventPickup,
		objective.EventPlantStart, objective.EventPlantAbort, objective.EventPlant,
		objective.EventDefuseStart, objective.EventDefuseAbort, objective.EventDefuse, objective.EventExplode,
	}
	raw := []models.ReplayEvent{{ID: "unrelated-kill", Tick: 25, Type: "kill"}}
	ledger := make([]objective.Event, 0, len(eventTypes)+1)
	for i, eventType := range eventTypes {
		raw = append(raw, models.ReplayEvent{ID: "raw-" + string(eventType), Tick: i + 10, Type: "bomb_" + string(eventType)})
		ledger = append(ledger, objective.Event{ID: "ledger-" + string(eventType), Type: eventType, Round: 4, Tick: i + 10})
	}
	ledger = append(ledger, objective.Event{ID: "another-round", Type: objective.EventDrop, Round: 5, Tick: 12})

	actual := projectReplayObjectiveMarkers(raw, ledger, 4)
	if len(actual) != len(eventTypes)+1 {
		t.Fatalf("expected every C4 transition once and the unrelated kill; got %+v", actual)
	}
	for i, eventType := range eventTypes {
		if actual[i].ID != "ledger-"+string(eventType) {
			t.Errorf("event %d = %s; want same-round ledger event %s", i, actual[i].ID, eventType)
		}
	}
	if actual[len(actual)-1].ID != "unrelated-kill" {
		t.Errorf("unrelated replay event was not preserved: %+v", actual)
	}
	if again := projectReplayObjectiveMarkers(actual, ledger, 4); !reflect.DeepEqual(again, actual) {
		t.Errorf("projecting an existing replay duplicated or changed C4 events: %+v", again)
	}
}

func TestReplayObjectiveC4ActorPositionDoesNotBecomeBombObjectPosition(t *testing.T) {
	for _, eventType := range []objective.EventType{
		objective.EventDrop, objective.EventPickup, objective.EventPlantStart, objective.EventPlantAbort,
		objective.EventPlant, objective.EventDefuseStart, objective.EventDefuseAbort, objective.EventDefuse,
	} {
		t.Run(string(eventType), func(t *testing.T) {
			marker, ok := replayObjectiveMarker(objective.Event{
				ID: "actor-position-" + string(eventType), Type: eventType, Round: 4, Tick: 100,
				Actor: objective.Actor{SteamID: 77, Side: "T"}, Source: objective.SourceDemoinfocsEvent,
				Position: objective.ObservedPosition(120, 340, 560, objective.SourceDemoinfocsEvent),
			})
			if !ok {
				t.Fatalf("event %s disappeared", eventType)
			}
			payload := replayC4JSON(t, marker)
			if payload["position_status"] != "unavailable" {
				t.Errorf("actor coordinates must not certify an observed bomb location: %v", payload)
			}
			if payload["position_source"] != "actor_event_not_object_position" {
				t.Errorf("missing explanation of unavailable object position: %v", payload["position_source"])
			}
			for _, axis := range []string{"x", "y", "z"} {
				if payload[axis] != float64(0) {
					t.Errorf("actor position leaked into bomb object %s: %v", axis, payload[axis])
				}
			}
		})
	}
}

func TestReplayObjectiveC4SnapshotDoesNotReuseLastObservedObjectPosition(t *testing.T) {
	tracker := objective.NewTracker()
	tracker.Drop(objective.EventInput{
		Round: 4, Tick: 100, Actor: objective.Actor{SteamID: 77, Side: "T"},
		Position: objective.ObservedPosition(10, 20, 30, objective.SourceDemoinfocsEvent),
		Source:   objective.SourceDemoinfocsEvent,
	})
	observed := replayBombObservation(tracker, objective.NativeObservation{
		Round: 4, Tick: 101,
		Position: objective.ObservedPosition(120, 340, 560, objective.SourceDemoinfocsNativeSnapshot),
	})
	if observed.State != objective.StateDropped || observed.PositionStatus != "observed" ||
		observed.X != 120 || observed.Y != 340 || observed.PositionSource != objective.SourceDemoinfocsNativeSnapshot {
		t.Fatalf("current native ground C4 observation was not preserved: %+v", observed)
	}

	unavailable := replayBombObservation(tracker, objective.NativeObservation{
		Round: 4, Tick: 102, Position: objective.UnavailablePosition(objective.SourceDemoinfocsNativeSnapshot),
	})
	if unavailable.State != objective.StateDropped || unavailable.PositionStatus != "unavailable" ||
		unavailable.X != 0 || unavailable.Y != 0 {
		t.Errorf("unobserved current C4 location reused the previous frame: %+v", unavailable)
	}
	if canonical := tracker.Snapshot(); !canonical.Position.Available() || canonical.Position.X != 120 || canonical.Position.Y != 340 {
		t.Errorf("replay presentation unexpectedly changed canonical tracker history: %+v", canonical.Position)
	}

	observedAgain := replayBombObservation(tracker, objective.NativeObservation{
		Round: 4, Tick: 103,
		Position: objective.ObservedPosition(125, 345, 560, objective.SourceDemoinfocsNativeSnapshot),
	})
	if observedAgain.PositionStatus != "observed" || observedAgain.X != 125 || observedAgain.Y != 345 {
		t.Errorf("new native observation failed to restore the current C4 position: %+v", observedAgain)
	}
}

func replayC4JSON(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(fmt.Errorf("encode replay C4: %w", err))
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(fmt.Errorf("decode replay C4: %w", err))
	}
	return payload
}
