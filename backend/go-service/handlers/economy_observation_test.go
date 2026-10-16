package handlers

import (
	"encoding/json"
	"testing"

	"cs2-demo-service/models"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func TestEconomyEarlyCallbackSurvivesRoundResetWithoutFutureMutation(t *testing.T) {
	tracker := newEconomyObservationTracker()
	tracker.rememberEarlyCallback(models.EconomyEarlyCallback{ActorAlias: "p0001", Kind: "refund", Tick: 100, CallbackSequence: 1})
	tracker.startRound(1, 100)
	initial := tracker.earlyCallbacksAt(100)
	if len(initial) != 1 {
		t.Fatal("native callback lost before RoundStart")
	}
	tracker.rememberEarlyCallback(models.EconomyEarlyCallback{ActorAlias: "p0001", Kind: "pickup", Tick: 101, CallbackSequence: 3})
	if len(initial) != 1 || initial[0].Kind != "refund" || len(tracker.earlyCallbacksAt(100)) != 0 {
		t.Fatal("future callback mutated retained boundary")
	}
}

func TestEconomyNativeZeroAndFalseRequirePropertyPresence(t *testing.T) {
	for _, test := range []struct {
		name    string
		entity  *initialVelocityTestEntity
		boolean bool
		status  string
		value   any
	}{
		{"missing", &initialVelocityTestEntity{}, false, "unavailable", nil},
		{"nil native value", &initialVelocityTestEntity{present: true}, false, "unavailable", nil},
		{"zero cash", &initialVelocityTestEntity{present: true, value: st.PropertyValue{Any: uint64(0)}}, false, "observed", 0},
		{"false kit", &initialVelocityTestEntity{present: true, value: st.PropertyValue{Any: false}}, true, "observed", false},
		{"cash sentinel", &initialVelocityTestEntity{present: true, value: st.PropertyValue{Any: uint64(4294967295)}}, false, "unavailable", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			fact := nativeEconomyFact(test.entity, "native", 42, test.boolean)
			if fact.Status != test.status || fact.Value != test.value {
				t.Fatalf("unexpected fact: %+v", fact)
			}
			if fact.Status == "observed" && (*fact.ObservationTick != 42 || *fact.AvailabilityTick != 42) {
				t.Fatal("observation frontier changed")
			}
		})
	}
	if fact := nativeEconomyFact(nil, "native", 42, false); fact.Value != nil || fact.AvailabilityTick != nil {
		t.Fatal("missing entity became zero")
	}
}

func validInitialEconomyActor() models.EconomyActorSnapshot {
	actor := unavailableEconomyActor("p0001", "CT")
	actor.Connected = true
	actor.Facts["money"] = observedEconomyFact(800, "money", 100)
	actor.Facts["spent_native"] = observedEconomyFact(0, "spent", 100)
	actor.Facts["inventory"] = observedEconomyFact([]models.EconomyInventoryItem{}, economyWeaponsProperty, 100)
	return actor
}

func TestEconomyPrepurchaseRejectsLateEarlyAndMissingStates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		tick     int
		activity bool
		mutate   func(*models.EconomyActorSnapshot)
		status   string
	}{
		{"observed frontier", 100, false, func(*models.EconomyActorSnapshot) {}, "available"},
		{"late frame", 101, false, func(*models.EconomyActorSnapshot) {}, "unavailable"},
		{"callback before frame", 100, true, func(*models.EconomyActorSnapshot) {}, "unavailable"},
		{"cash already spent", 100, false, func(a *models.EconomyActorSnapshot) { a.Facts["spent_native"] = observedEconomyFact(200, "spent", 100) }, "unavailable"},
		{"no inventory", 100, false, func(a *models.EconomyActorSnapshot) {
			a.Facts["inventory"] = missingEconomyFact(economyWeaponsProperty)
		}, "unavailable"},
		{"disconnected", 100, false, func(a *models.EconomyActorSnapshot) { a.Connected = false }, "unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actor := validInitialEconomyActor()
			tc.mutate(&actor)
			setEconomyPrepurchaseStatus(&actor, "initial_available", 100, tc.tick, tc.activity)
			if actor.PrepurchaseStatus != tc.status {
				t.Fatalf("invalid prebuy: %+v", actor)
			}
		})
	}
}

func TestEconomyLaterEventsCannotRewriteInitialSnapshot(t *testing.T) {
	tracker := newEconomyObservationTracker()
	tracker.startRound(1, 100)
	actor := validInitialEconomyActor()
	setEconomyPrepurchaseStatus(&actor, "initial_available", 100, 100, false)
	tracker.round().Snapshots = append(tracker.round().Snapshots, models.EconomySnapshot{Phase: "initial_available", Tick: 100, Actors: []models.EconomyActorSnapshot{actor}})
	before, _ := json.Marshal(tracker.round().Snapshots)
	for _, kind := range []string{"pickup", "refund", "drop"} {
		tracker.recordEvent(models.EconomyCallbackEvent{Kind: kind, Tick: 200, ActorAlias: "p0001", Side: "CT"})
	}
	after, _ := json.Marshal(tracker.round().Snapshots)
	if string(before) != string(after) {
		t.Fatal("later activity rewrote initial observation")
	}
	tracker.startRound(1, 500)
	if tracker.round().Instance != 2 || len(tracker.round().Events) != 0 || len(tracker.activity) != 0 {
		t.Fatal("round restart reused previous activity")
	}
}

func TestEconomyDropReferenceNeedsPriorSameEntityAndSide(t *testing.T) {
	tracker := newEconomyObservationTracker()
	tracker.startRound(1, 100)
	entity := "w00001"
	tracker.recordEvent(models.EconomyCallbackEvent{EventID: "e00000002", Kind: "drop", Tick: 110, ActorAlias: "p0001", Side: "CT", EntityAlias: &entity, Attribution: "callback_only"})
	tracker.recordEvent(models.EconomyCallbackEvent{Kind: "pickup", Tick: 111, ActorAlias: "p0002", Side: "CT", EntityAlias: &entity})
	tracker.recordEvent(models.EconomyCallbackEvent{Kind: "pickup", Tick: 112, ActorAlias: "p0003", Side: "CT", EntityAlias: &entity})
	if tracker.round().Events[1].PriorDropEventID == nil || tracker.round().Events[2].PriorDropEventID != nil {
		t.Fatal("drop was not consumed exactly once")
	}
	tracker.startRound(2, 200)
	tracker.recordEvent(models.EconomyCallbackEvent{Kind: "pickup", Tick: 201, ActorAlias: "p0002", Side: "CT", EntityAlias: &entity})
	if tracker.round().Events[0].PriorDropEventID != nil {
		t.Fatal("drop crossed round")
	}
}
