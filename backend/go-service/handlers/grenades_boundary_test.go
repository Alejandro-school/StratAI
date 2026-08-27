package handlers

import (
	"reflect"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
)

func TestUtilityRoundBoundaryKeepsPostRoundCallbacksUntilOfficialEnd(t *testing.T) {
	handler := newBoundaryUtilityHandler()
	handler.beginRound(1, false)
	throwID, _ := handler.ctx.Utilities.RecordThrow(boundaryThrowInput(1, 10, 100))

	if !handler.state.markRoundEnd() || !handler.state.records(false) {
		t.Fatal("RoundEnd must enter an accepting post-round phase")
	}
	handler.ctx.Utilities.RecordDetonation(utility.CallbackHint{
		Round: 1, RuntimeEntityID: 10, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeFlashbang, ActorID: 1, Tick: 110,
		PositionStatus: utility.AvailabilityUnavailable,
	}, utility.SourceFlashExplode)
	if !handler.finishCurrentRound() || handler.state.records(false) {
		t.Fatal("RoundEndOfficial must close the accepting boundary exactly once")
	}

	entry := boundaryThrowByID(t, handler.ctx.Utilities.Snapshot(), throwID)
	if entry.Lifecycle.Status != utility.LifecycleDetonated {
		t.Fatalf("post-round detonation was lost: %+v", entry.Lifecycle)
	}
	if handler.finishCurrentRound() {
		t.Fatal("official round close must be idempotent")
	}
}

func TestUtilityRoundBoundaryFallsBackAtNextRoundStart(t *testing.T) {
	handler := newBoundaryUtilityHandler()
	handler.beginRound(1, false)
	firstID, _ := handler.ctx.Utilities.RecordThrow(boundaryThrowInput(1, 10, 100))
	handler.state.markRoundEnd()

	handler.beginRound(2, false)
	secondID, _ := handler.ctx.Utilities.RecordThrow(boundaryThrowInput(2, 10, 200))

	first := boundaryThrowByID(t, handler.ctx.Utilities.Snapshot(), firstID)
	second := boundaryThrowByID(t, handler.ctx.Utilities.Snapshot(), secondID)
	if first.Lifecycle.Status != utility.LifecycleRoundEndedUnresolved ||
		first.Lifecycle.EndReason != utility.EndReasonRoundEnd {
		t.Fatalf("previous round was not closed by the next RoundStart: %+v", first.Lifecycle)
	}
	if second.Round != 2 || handler.state.round != 2 || handler.state.phase != utilityRoundLive {
		t.Fatalf("next round state was contaminated: state=%+v throw=%+v", handler.state, second)
	}
}

func TestUtilityRoundBoundaryFinalizesLastRoundAndTracksDiscards(t *testing.T) {
	handler := newBoundaryUtilityHandler()
	handler.beginRound(1, false)
	throwID, _ := handler.ctx.Utilities.RecordThrow(boundaryThrowInput(1, 10, 100))

	handler.Finalize()
	handler.Finalize()
	entry := boundaryThrowByID(t, handler.ctx.Utilities.Snapshot(), throwID)
	if entry.Lifecycle.Status != utility.LifecycleRoundEndedUnresolved || handler.state.phase != utilityRoundInactive {
		t.Fatalf("last round remained active after Finalize: state=%+v lifecycle=%+v", handler.state, entry.Lifecycle)
	}
	if len(handler.ctx.AI_GrenadeEvents) != 1 {
		t.Fatalf("final projection was not refreshed: %+v", handler.ctx.AI_GrenadeEvents)
	}

	handler.ctx.Utilities.RecordDiscardedCallback(
		utility.CallbackGroupLifecycle,
		utility.CallbackDiscardOutsideRound,
	)
	handler.invalidCallback(utility.CallbackGroupThrows)
	diagnostics := handler.ctx.Utilities.Diagnostics()
	if diagnostics.Discarded.Lifecycle.OutsideRound != 1 ||
		diagnostics.Discarded.Throws.Invalid != 1 {
		t.Fatalf("discarded callbacks are invisible: %+v", diagnostics.Discarded)
	}
	if diagnostics.Lifecycle.Observed != 0 || diagnostics.Throws.Observed != 2 ||
		diagnostics.Throws.Unmatched != 1 {
		t.Fatalf("intentional and invalid discards were not separated: %+v", diagnostics)
	}
}

func TestUtilityRoundBoundaryIsDeterministic(t *testing.T) {
	firstThrows, firstDiagnostics := runBoundarySequence()
	secondThrows, secondDiagnostics := runBoundarySequence()
	if !reflect.DeepEqual(firstThrows, secondThrows) || !reflect.DeepEqual(firstDiagnostics, secondDiagnostics) {
		t.Fatalf("round boundary changed deterministic output:\nfirst=%+v %+v\nsecond=%+v %+v",
			firstThrows, firstDiagnostics, secondThrows, secondDiagnostics)
	}
}

func runBoundarySequence() ([]utility.Throw, utility.Diagnostics) {
	handler := newBoundaryUtilityHandler()
	handler.beginRound(1, false)
	handler.ctx.Utilities.RecordThrow(boundaryThrowInput(1, 10, 100))
	handler.state.markRoundEnd()
	handler.ctx.Utilities.RecordDetonation(utility.CallbackHint{
		Round: 1, RuntimeEntityID: 10, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeFlashbang, ActorID: 1, Tick: 110,
	}, utility.SourceFlashExplode)
	handler.finishCurrentRound()
	handler.ctx.Utilities.RecordDiscardedCallback(
		utility.CallbackGroupLifecycle,
		utility.CallbackDiscardOutsideRound,
	)
	return handler.ctx.Utilities.Snapshot(), handler.ctx.Utilities.Diagnostics()
}

func newBoundaryUtilityHandler() *UtilityHandler {
	tracker := utility.NewTracker()
	return &UtilityHandler{
		ctx: &models.DemoContext{Utilities: tracker, MatchData: &models.MatchData{}},
	}
}

func boundaryThrowInput(round, entityID, tick int) utility.ThrowInput {
	return utility.ThrowInput{
		Round: round, RuntimeEntityID: entityID, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeFlashbang, TypeSource: utility.SourceWeaponInstance,
		Actor: utility.PlayerRef{
			ID: 1, Name: "thrower", Side: "T",
			Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrower,
		},
		Launch: utility.ThrowSnapshot{
			Tick: utility.TickObservation{
				Tick: tick, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow,
			},
		},
	}
}

func boundaryThrowByID(t *testing.T, throws []utility.Throw, id string) utility.Throw {
	t.Helper()
	for _, entry := range throws {
		if entry.ID == id {
			return entry
		}
	}
	t.Fatalf("throw %q not found", id)
	return utility.Throw{}
}
