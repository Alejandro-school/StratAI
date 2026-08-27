package objective

import "testing"

func TestPlantDefuseLifecycleSeparatesCurrentFromHistoricalState(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(4, 100)

	planter := Actor{SteamID: 11, Name: "planter", Side: "T"}
	defuser := Actor{SteamID: 22, Name: "defuser", Side: "CT"}
	tracker.PlantStart(EventInput{
		Round: 4, Tick: 200, Actor: planter,
		Site: "A", Position: ObservedPosition(10, 20, 30, SourceDemoinfocsEvent),
	})
	plant := tracker.Plant(EventInput{
		Round: 4, Tick: 260, Actor: planter,
		Site: "A", Position: ObservedPosition(12, 22, 32, SourceDemoinfocsEvent),
	})
	hasKit := true
	tracker.DefuseStart(EventInput{
		Round: 4, Tick: 400, Actor: defuser, Site: "A", HasKit: &hasKit,
		Position: ObservedPosition(13, 23, 33, SourceDemoinfocsEvent),
	})
	defuse := tracker.Defuse(EventInput{
		Round: 4, Tick: 720, Actor: defuser, Site: "A",
		Position: ObservedPosition(13, 23, 33, SourceDemoinfocsEvent),
	})

	snapshot := tracker.Snapshot()
	if snapshot.IsPlantedNow {
		t.Fatal("resolved objective remained planted in current state")
	}
	if !snapshot.WasPlantedThisRound {
		t.Fatal("terminal transition erased historical plant state")
	}
	if snapshot.State != StateDefused || snapshot.Phase != PhaseResolved {
		t.Fatalf("unexpected terminal state: %+v", snapshot)
	}
	if snapshot.Site != "A" || snapshot.PlantTick != 260 {
		t.Fatalf("plant facts were not retained: %+v", snapshot)
	}
	if plant.DurationTicks == nil || *plant.DurationTicks != 60 {
		t.Fatalf("plant duration mismatch: %+v", plant)
	}
	if defuse.DurationTicks == nil || *defuse.DurationTicks != 320 {
		t.Fatalf("defuse duration mismatch: %+v", defuse)
	}
	if defuse.HasKit == nil || !*defuse.HasKit {
		t.Fatalf("defuse completion did not inherit kit state: %+v", defuse)
	}

	summary, ok := tracker.RoundSummary(4)
	if !ok || !summary.WasPlanted || summary.Resolution != EventDefuse || summary.FinalState != StateDefused {
		t.Fatalf("round summary is inconsistent: %+v, found=%v", summary, ok)
	}
}

func TestAbortedAttemptsAreLinkedAndTimed(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(9, 10)
	actor := Actor{SteamID: 31, Side: "T"}

	start := tracker.PlantStart(EventInput{Round: 9, Tick: 100, Actor: actor})
	abort := tracker.PlantAbort(EventInput{Round: 9, Tick: 145, Actor: actor})

	if start.AttemptID == "" || abort.AttemptID != start.AttemptID {
		t.Fatalf("attempt linkage mismatch: start=%+v abort=%+v", start, abort)
	}
	if start.RelatedEventID != "" || !start.AttemptStartObserved {
		t.Fatalf("start event has an invalid self-link or observation status: %+v", start)
	}
	if abort.RelatedEventID != start.ID {
		t.Fatalf("abort does not link to its start: %+v", abort)
	}
	if abort.DurationTicks == nil || *abort.DurationTicks != 45 {
		t.Fatalf("abort duration mismatch: %+v", abort)
	}
	if abort.AttemptOutcome != AttemptAborted {
		t.Fatalf("abort outcome mismatch: %+v", abort)
	}
	if snapshot := tracker.Snapshot(); snapshot.State != StateCarried || snapshot.WasPlantedThisRound {
		t.Fatalf("abort corrupted objective state: %+v", snapshot)
	}
}

func TestPlantAbortAfterDropKeepsBombDropped(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(1, 1)
	planter := Actor{SteamID: 7, Side: "T"}
	tracker.PlantStart(EventInput{Round: 1, Tick: 100, Actor: planter, Site: "A"})
	drop := tracker.Drop(EventInput{Round: 1, Tick: 120, Actor: planter})
	abort := tracker.PlantAbort(EventInput{Round: 1, Tick: 120, Actor: planter, Site: "A"})

	if drop.Site != "" {
		t.Fatalf("drop inherited a site from an unfinished plant attempt: %+v", drop)
	}
	if abort.StateAfter != StateDropped || abort.PhaseAfter != PhasePreplant {
		t.Fatalf("plant abort resurrected a dropped carrier: %+v", abort)
	}
	if snapshot := tracker.Snapshot(); snapshot.State != StateDropped || snapshot.Carrier.SteamID != 0 {
		t.Fatalf("dropped C4 gained a carrier after plant abort: %+v", snapshot)
	}
}

func TestPostPlantPickupAndDropCallbacksAreIgnored(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(1, 1)
	planter := Actor{SteamID: 7, Side: "T"}
	tracker.Plant(EventInput{Round: 1, Tick: 100, Actor: planter, Site: "B"})
	eventCount := len(tracker.Events())

	if event := tracker.Pickup(EventInput{Round: 1, Tick: 100, Actor: planter}); event.ID != "" {
		t.Fatalf("post-plant pickup was recorded: %+v", event)
	}
	if event := tracker.Drop(EventInput{Round: 1, Tick: 101, Actor: planter}); event.ID != "" {
		t.Fatalf("post-plant drop was recorded: %+v", event)
	}
	if snapshot := tracker.Snapshot(); snapshot.State != StatePlanted || snapshot.Carrier.SteamID != 0 {
		t.Fatalf("post-plant callback changed objective state: %+v", snapshot)
	}
	if len(tracker.Events()) != eventCount {
		t.Fatal("ignored post-plant callbacks leaked into the objective ledger")
	}
}

func TestAnonymousPickupDoesNotInventCarrier(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(1, 1)
	carrier := Actor{SteamID: 7, Side: "T"}
	tracker.Drop(EventInput{Round: 1, Tick: 10, Actor: carrier})
	eventCount := len(tracker.Events())

	if event := tracker.Pickup(EventInput{Round: 1, Tick: 20}); event.ID != "" {
		t.Fatalf("anonymous pickup was recorded: %+v", event)
	}
	if snapshot := tracker.Snapshot(); snapshot.State != StateDropped || snapshot.Carrier.SteamID != 0 {
		t.Fatalf("anonymous pickup invented a carrier: %+v", snapshot)
	}
	if len(tracker.Events()) != eventCount {
		t.Fatal("anonymous pickup leaked into the objective ledger")
	}

	tracker.NativeSnapshot(NativeObservation{Round: 1, Tick: 21, Carrier: carrier})
	if snapshot := tracker.Snapshot(); snapshot.State != StateCarried || snapshot.Carrier.SteamID != carrier.SteamID {
		t.Fatalf("later native observation did not recover the real carrier: %+v", snapshot)
	}
}

func TestDuplicateDropDoesNotCreateInvalidTransition(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(1, 1)
	carrier := Actor{SteamID: 7, Side: "T"}
	first := tracker.Drop(EventInput{Round: 1, Tick: 10, Actor: carrier})
	eventCount := len(tracker.Events())

	duplicate := tracker.Drop(EventInput{Round: 1, Tick: 11, Actor: carrier})
	if first.ID == "" || duplicate.ID != "" {
		t.Fatalf("duplicate drop was not ignored: first=%+v duplicate=%+v", first, duplicate)
	}
	if len(tracker.Events()) != eventCount {
		t.Fatal("duplicate drop leaked into the objective ledger")
	}
	if snapshot := tracker.Snapshot(); snapshot.State != StateDropped || snapshot.Carrier.SteamID != 0 {
		t.Fatalf("duplicate drop changed objective state: %+v", snapshot)
	}
}

func TestAnonymousPlantAbortDoesNotInventCarrier(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(1, 1)
	tracker.PlantStart(EventInput{Round: 1, Tick: 10})

	abort := tracker.PlantAbort(EventInput{Round: 1, Tick: 20})
	if abort.StateAfter != StateUnknown || abort.PhaseAfter != PhasePreplant {
		t.Fatalf("anonymous plant abort invented objective state: %+v", abort)
	}
	if snapshot := tracker.Snapshot(); snapshot.State != StateUnknown || snapshot.Carrier.SteamID != 0 {
		t.Fatalf("anonymous plant abort invented a carrier: %+v", snapshot)
	}
}

func TestSameTickPlantAbortAndPickupAreCoalescedInEitherCallbackOrder(t *testing.T) {
	for _, pickupFirst := range []bool{false, true} {
		tracker := NewTracker()
		tracker.BeginRound(1, 1)
		planter := Actor{SteamID: 7, Side: "T"}
		tracker.PlantStart(EventInput{Round: 1, Tick: 100, Actor: planter, Site: "A"})
		if pickupFirst {
			tracker.Pickup(EventInput{Round: 1, Tick: 120, Actor: planter})
			tracker.PlantAbort(EventInput{Round: 1, Tick: 120, Actor: planter, Site: "A"})
		} else {
			tracker.PlantAbort(EventInput{Round: 1, Tick: 120, Actor: planter, Site: "A"})
			tracker.Pickup(EventInput{Round: 1, Tick: 120, Actor: planter})
		}

		for _, event := range tracker.Events() {
			if event.Type == EventPickup && event.Tick == 120 {
				t.Fatalf("pickup artifact survived callback order pickupFirst=%t: %+v", pickupFirst, tracker.Events())
			}
		}
		if snapshot := tracker.Snapshot(); snapshot.State != StateCarried || snapshot.Carrier.SteamID != planter.SteamID {
			t.Fatalf("plant abort state differs for pickupFirst=%t: %+v", pickupFirst, snapshot)
		}
	}
}

func TestCompletionWithoutStartKeepsDurationUnknown(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(1, 1)
	plant := tracker.Plant(EventInput{Round: 1, Tick: 80, Actor: Actor{SteamID: 7, Side: "T"}, Site: "B"})

	attempts := tracker.Attempts()
	if len(attempts) != 1 {
		t.Fatalf("expected one inferred attempt, got %d", len(attempts))
	}
	attempt := attempts[0]
	if attempt.StartObserved || attempt.StartEventID != "" {
		t.Fatalf("missing begin callback was presented as observed: %+v", attempt)
	}
	if attempt.DurationTicks != nil || plant.DurationTicks != nil || plant.AttemptStartObserved {
		t.Fatalf("missing start callback was converted into an observed duration: attempt=%+v event=%+v", attempt, plant)
	}
}

func TestNativeCarrierSnapshotIsDeduplicatedAndNeverOverridesPlantedState(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(2, 10)
	carrier := Actor{SteamID: 99, Name: "carrier", Side: "T"}
	observation := NativeObservation{
		Round: 2, Tick: 20, Carrier: carrier,
		Position: ObservedPosition(1, 2, 3, SourceDemoinfocsNativeSnapshot),
	}
	tracker.NativeSnapshot(observation)
	observation.Tick = 24
	tracker.NativeSnapshot(observation)

	events := tracker.Events()
	if len(events) != 1 || events[0].Type != EventCarrierSnapshot {
		t.Fatalf("native carrier observation was duplicated: %+v", events)
	}
	tracker.Plant(EventInput{Round: 2, Tick: 100, Actor: carrier, Site: "A"})
	observation.Tick = 104
	tracker.NativeSnapshot(observation)

	snapshot := tracker.Snapshot()
	if snapshot.State != StatePlanted || snapshot.Carrier.SteamID != 0 {
		t.Fatalf("stale native carrier overrode planted state: %+v", snapshot)
	}
}

func TestReadAPIsReturnPointerIndependentCopies(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(3, 1)
	entityID := 42
	event := tracker.Drop(EventInput{Round: 3, Tick: 2, EntityID: &entityID})

	events := tracker.Events()
	*events[0].EntityID = 500
	again := tracker.Events()
	if event.EntityID == nil || *again[0].EntityID != 42 {
		t.Fatalf("caller mutated tracker ledger: %+v", again[0])
	}
}

func TestRoundEndResolvesObjectiveWithoutErasingPlantHistory(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(12, 10)
	actor := Actor{SteamID: 8, Side: "T"}
	tracker.Plant(EventInput{Round: 12, Tick: 100, Actor: actor, Site: "A"})
	tracker.EndRound(12, 180)

	snapshot := tracker.Snapshot()
	if snapshot.State != StateResolved || snapshot.Phase != PhaseResolved || snapshot.IsPlantedNow || !snapshot.WasPlantedThisRound {
		t.Fatalf("round end did not reconcile current/history state: %+v", snapshot)
	}
	summary, ok := tracker.RoundSummary(12)
	if !ok || !summary.ResolvedAtRoundEnd || summary.RoundEndTick != 180 || !summary.WasPlanted {
		t.Fatalf("round-end reconciliation missing from summary: %+v", summary)
	}
}

func TestRoundEndLeavesStartOnlyAttemptIncomplete(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(7, 1)
	tracker.PlantStart(EventInput{Round: 7, Tick: 50, Actor: Actor{SteamID: 4, Side: "T"}})
	tracker.EndRound(7, 70)

	attempts := tracker.Attempts()
	if len(attempts) != 1 || attempts[0].Outcome != AttemptInProgress {
		t.Fatalf("start-only attempt was assigned a fabricated terminal outcome: %+v", attempts)
	}
	if attempts[0].DurationTicks != nil || attempts[0].EndTick != nil || attempts[0].EndEventID != "" {
		t.Fatalf("start-only attempt was assigned a fabricated terminal boundary: %+v", attempts[0])
	}
}

func TestRoundEndPreservesObservedTerminalState(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(8, 1)
	planter := Actor{SteamID: 10, Side: "T"}
	defuser := Actor{SteamID: 20, Side: "CT"}
	tracker.Plant(EventInput{Round: 8, Tick: 100, Actor: planter, Site: "B"})
	tracker.Defuse(EventInput{Round: 8, Tick: 300, Actor: defuser, Site: "B"})
	tracker.EndRound(8, 304)

	snapshot := tracker.Snapshot()
	if snapshot.State != StateDefused || snapshot.Phase != PhaseResolved {
		t.Fatalf("round end overwrote observed terminal state: %+v", snapshot)
	}
	summary, ok := tracker.RoundSummary(8)
	if !ok || summary.FinalState != StateDefused || summary.Resolution != EventDefuse || summary.ResolvedAtRoundEnd {
		t.Fatalf("terminal round summary was overwritten: %+v", summary)
	}
}

func TestBeginRoundRestartsResolvedDuplicateRoundNumber(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(1, 10)
	tracker.NativeSnapshot(NativeObservation{
		Round: 1,
		Tick:  20,
		Carrier: Actor{
			SteamID: 7,
			Side:    "T",
		},
	})
	tracker.EndRound(1, 30)

	tracker.BeginRound(1, 100)

	snapshot := tracker.Snapshot()
	if snapshot.Round != 1 || snapshot.Tick != 100 || snapshot.State != StateUnknown || snapshot.Phase != PhasePreplant {
		t.Fatalf("resolved duplicate round was not restarted: %+v", snapshot)
	}
	if len(tracker.Events()) != 0 || len(tracker.Attempts()) != 0 {
		t.Fatalf("replaced round retained preliminary ledger entries: events=%+v attempts=%+v", tracker.Events(), tracker.Attempts())
	}
	if _, ok := tracker.RoundSummary(1); !ok {
		t.Fatal("replacement round did not receive a fresh summary")
	}
}

func TestBeginRoundKeepsActiveDuplicateIdempotent(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(2, 10)
	tracker.NativeSnapshot(NativeObservation{Round: 2, Tick: 20, Carrier: Actor{SteamID: 9, Side: "T"}})

	tracker.BeginRound(2, 30)

	snapshot := tracker.Snapshot()
	if snapshot.Tick != 20 || snapshot.Carrier.SteamID != 9 || len(tracker.Events()) != 1 {
		t.Fatalf("active duplicate round start was not idempotent: snapshot=%+v events=%+v", snapshot, tracker.Events())
	}
}

func TestCarrierAndDefuseAbortLifecyclePreservesNativeFacts(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(5, 1)
	firstCarrier := Actor{SteamID: 1, Side: "T"}
	secondCarrier := Actor{SteamID: 2, Side: "T"}
	defuser := Actor{SteamID: 3, Side: "CT"}
	tracker.NativeSnapshot(NativeObservation{Round: 5, Tick: 10, Carrier: firstCarrier})
	entityID := 77
	drop := tracker.Drop(EventInput{Round: 5, Tick: 20, Actor: firstCarrier, EntityID: &entityID})
	tracker.Pickup(EventInput{Round: 5, Tick: 30, Actor: secondCarrier})
	tracker.Plant(EventInput{Round: 5, Tick: 100, Actor: secondCarrier, Site: "A"})
	hasKit := false
	start := tracker.DefuseStart(EventInput{Round: 5, Tick: 200, Actor: defuser, Site: "A", HasKit: &hasKit})
	abort := tracker.DefuseAbort(EventInput{Round: 5, Tick: 240, Actor: defuser, Site: "A"})

	if drop.EntityID == nil || *drop.EntityID != 77 {
		t.Fatalf("drop entity identity was lost: %+v", drop)
	}
	if start.HasKit == nil || *start.HasKit || abort.HasKit == nil || *abort.HasKit {
		t.Fatalf("defuse kit observation was not linked: start=%+v abort=%+v", start, abort)
	}
	if abort.AttemptID != start.AttemptID || abort.DurationTicks == nil || *abort.DurationTicks != 40 {
		t.Fatalf("defuse abort was not linked and timed: start=%+v abort=%+v", start, abort)
	}
	snapshot := tracker.Snapshot()
	if snapshot.State != StatePlanted || !snapshot.IsPlantedNow || snapshot.Defuser.SteamID != 0 {
		t.Fatalf("defuse abort did not return to planted current state: %+v", snapshot)
	}

	events := tracker.Events()
	for index := 1; index < len(events); index++ {
		if events[index].Sequence <= events[index-1].Sequence {
			t.Fatalf("ledger sequence is not monotonic: %+v", events)
		}
	}
}
