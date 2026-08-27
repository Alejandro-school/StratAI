package playerstate

import (
	"math"
	"testing"

	"github.com/golang/geo/r3"
)

func TestMotionTrackerUsesCausalPositionDelta(t *testing.T) {
	var tracker MotionTracker
	first := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 10, TickRate: 64,
		Position: r3.Vector{}, IsAlive: true,
	})
	if first.Available || first.Source != VelocitySourceInsufficientHistory {
		t.Fatalf("first estimate = %+v", first)
	}

	second := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 12, TickRate: 64,
		Position: r3.Vector{X: 10, Y: 4, Z: 2}, IsAlive: true,
	})
	if !second.Available || second.Source != VelocitySourcePositionDelta || second.IntervalTicks != 2 {
		t.Fatalf("second estimate = %+v", second)
	}
	assertClose(t, second.Vector.X, 320)
	assertClose(t, second.Vector.Y, 128)
	assertClose(t, second.Vector.Z, 64)
	assertClose(t, second.HorizontalSpeed(), math.Hypot(320, 128))
}

func TestMotionTrackerTreatsStationaryAsObservedZero(t *testing.T) {
	var tracker MotionTracker
	observation := MotionObservation{
		PlayerID: 1, Round: 1, Tick: 10, TickRate: 64,
		Position: r3.Vector{X: 1, Y: 2, Z: 3}, IsAlive: true,
	}
	tracker.Observe(observation)
	observation.Tick = 11
	estimate := tracker.Observe(observation)
	if !estimate.Available || estimate.Source != VelocitySourcePositionDelta || estimate.HorizontalSpeed() != 0 {
		t.Fatalf("stationary estimate = %+v", estimate)
	}
}

func TestMotionTrackerIsIdempotentWithinTick(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 1, TickRate: 64, IsAlive: true})
	first := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 2, TickRate: 64,
		Position: r3.Vector{X: 2}, IsAlive: true,
	})
	second := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 2, TickRate: 64,
		Position: r3.Vector{X: 999}, IsAlive: true,
	})
	if first != second {
		t.Fatalf("same-tick estimate changed: first=%+v second=%+v", first, second)
	}
}

func TestMotionTrackerRejectsRegressiveTickWithoutPoisoningAnchor(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 10, TickRate: 64, IsAlive: true})
	tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 11, TickRate: 64,
		Position: r3.Vector{X: 1}, IsAlive: true,
	})
	regressive := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 9, TickRate: 64,
		Position: r3.Vector{X: 999}, IsAlive: true,
	})
	if regressive.Available || regressive.Source != VelocitySourceNonMonotonicTick {
		t.Fatalf("regressive estimate = %+v", regressive)
	}
	next := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 12, TickRate: 64,
		Position: r3.Vector{X: 2}, IsAlive: true,
	})
	if !next.Available || next.Vector.X != 64 {
		t.Fatalf("regressive observation poisoned anchor: %+v", next)
	}
}

func TestMotionTrackerDetectsEntityChangeWithinSameTick(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{
		PlayerID: 1, PawnID: 7, PawnSerial: 1, Round: 1, Tick: 10, TickRate: 64, IsAlive: true,
	})
	changed := tracker.Observe(MotionObservation{
		PlayerID: 1, PawnID: 8, PawnSerial: 1, Round: 1, Tick: 10, TickRate: 64,
		Position: r3.Vector{X: 10}, IsAlive: true,
	})
	if changed.Available || changed.Source != VelocitySourceEntityChanged {
		t.Fatalf("same-tick entity change = %+v", changed)
	}
}

func TestMotionTrackerUsesTickRateFallbackAndKeepsPlayersIsolated(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 1, IsAlive: true})
	tracker.Observe(MotionObservation{PlayerID: 2, Round: 1, Tick: 1, Position: r3.Vector{X: 100}, IsAlive: true})
	first := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 2, Position: r3.Vector{X: 1}, IsAlive: true,
	})
	second := tracker.Observe(MotionObservation{
		PlayerID: 2, Round: 1, Tick: 2, Position: r3.Vector{X: 102}, IsAlive: true,
	})
	if first.Vector.X != 64 || second.Vector.X != 128 {
		t.Fatalf("isolated fallback estimates: first=%+v second=%+v", first, second)
	}
}

func TestMotionTrackerResetsAcrossRoundsAndDeath(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 10, TickRate: 64, IsAlive: true})
	afterRoundChange := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 2, Tick: 20, TickRate: 64,
		Position: r3.Vector{X: 1000}, IsAlive: true,
	})
	if afterRoundChange.Available || afterRoundChange.Source != VelocitySourceInsufficientHistory {
		t.Fatalf("round reset estimate = %+v", afterRoundChange)
	}

	dead := tracker.Observe(MotionObservation{PlayerID: 1, Round: 2, Tick: 21, TickRate: 64})
	if dead.Available || dead.Source != VelocitySourceNotApplicable {
		t.Fatalf("dead estimate = %+v", dead)
	}
	afterDeath := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 2, Tick: 22, TickRate: 64,
		Position: r3.Vector{X: 1001}, IsAlive: true,
	})
	if afterDeath.Available {
		t.Fatalf("post-death estimate reused stale history: %+v", afterDeath)
	}
}

func TestMotionTrackerReseedsAfterStaleGapAndEntityChange(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{
		PlayerID: 1, PawnID: 7, PawnSerial: 2, Round: 1, Tick: 1, TickRate: 64, IsAlive: true,
	})
	stale := tracker.Observe(MotionObservation{
		PlayerID: 1, PawnID: 7, PawnSerial: 2, Round: 1, Tick: 20, TickRate: 64,
		Position: r3.Vector{X: 1}, IsAlive: true,
	})
	if stale.Available || stale.Source != VelocitySourceStaleGap {
		t.Fatalf("stale estimate = %+v", stale)
	}
	changed := tracker.Observe(MotionObservation{
		PlayerID: 1, PawnID: 8, PawnSerial: 1, Round: 1, Tick: 21, TickRate: 64,
		Position: r3.Vector{X: 2}, IsAlive: true,
	})
	if changed.Available || changed.Source != VelocitySourceEntityChanged {
		t.Fatalf("entity change estimate = %+v", changed)
	}
}

func TestMotionTrackerKeepsLastAliveEstimateSeparateFromDeadState(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 1, TickRate: 64, IsAlive: true})
	tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 2, TickRate: 64,
		Position: r3.Vector{X: 1}, IsAlive: true,
	})
	tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 3, TickRate: 64})
	lastAlive, exists := tracker.LastAlive(1, 1)
	if !exists || !lastAlive.Available || lastAlive.Source != VelocitySourcePositionDelta || lastAlive.ObservedTick != 2 {
		t.Fatalf("last alive estimate = %+v, exists=%v", lastAlive, exists)
	}
	tracker.Reset()
	if _, exists := tracker.LastAlive(1, 1); exists {
		t.Fatal("reset retained last alive estimate")
	}
}

func TestMotionTrackerUsesNativeVelocityWithoutPositionHistory(t *testing.T) {
	var tracker MotionTracker
	native := r3.Vector{X: 12, Y: 5, Z: -1}
	estimate := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 10, TickRate: 64,
		Position: r3.Vector{X: 999}, IsAlive: true, NativeVelocity: &native,
	})
	if !estimate.Available || estimate.Source != VelocitySourceNative || estimate.Vector != native {
		t.Fatalf("native estimate = %+v", estimate)
	}
}

func TestMotionTrackerPrefersPositionDeltaOnceAvailable(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 1, TickRate: 64, IsAlive: true})
	native := r3.Vector{X: 999}
	estimate := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 2, TickRate: 64,
		Position: r3.Vector{X: 2}, IsAlive: true, NativeVelocity: &native,
	})
	if !estimate.Available || estimate.Source != VelocitySourcePositionDelta {
		t.Fatalf("position delta was not authoritative: %+v", estimate)
	}
}

func TestMotionTrackerRejectsStaleNativeZeroWhenPositionMoved(t *testing.T) {
	var tracker MotionTracker
	tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 1, TickRate: 64, IsAlive: true})
	staleNative := r3.Vector{}
	estimate := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 2, TickRate: 64,
		Position: r3.Vector{X: 2}, IsAlive: true, NativeVelocity: &staleNative,
	})
	if !estimate.Available || estimate.Source != VelocitySourcePositionDelta {
		t.Fatalf("stale native zero was trusted: %+v", estimate)
	}
}

func TestMotionTrackerDoesNotTrustNativeZeroWithoutHistory(t *testing.T) {
	var tracker MotionTracker
	zero := r3.Vector{}
	estimate := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 1, TickRate: 64,
		IsAlive: true, NativeVelocity: &zero,
	})
	if estimate.Available || estimate.Source != VelocitySourceInsufficientHistory {
		t.Fatalf("native zero was treated as observed: %+v", estimate)
	}
}

func TestMotionTrackerRejectsImplausibleTeleport(t *testing.T) {
	for _, test := range []struct {
		name     string
		position r3.Vector
	}{
		{name: "horizontal", position: r3.Vector{X: 1000}},
		{name: "vertical", position: r3.Vector{Z: 1000}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var tracker MotionTracker
			tracker.Observe(MotionObservation{PlayerID: 1, Round: 1, Tick: 1, TickRate: 64, IsAlive: true})
			estimate := tracker.Observe(MotionObservation{
				PlayerID: 1, Round: 1, Tick: 2, TickRate: 64,
				Position: test.position, IsAlive: true,
			})
			if estimate.Available || estimate.Source != VelocitySourceRejected {
				t.Fatalf("teleport estimate = %+v", estimate)
			}
		})
	}
}

func TestMotionTrackerDoesNotStoreInvalidPosition(t *testing.T) {
	var tracker MotionTracker
	rejected := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 1, TickRate: 64,
		Position: r3.Vector{X: math.NaN()}, IsAlive: true,
	})
	if rejected.Available || rejected.Source != VelocitySourceRejected {
		t.Fatalf("invalid position estimate = %+v", rejected)
	}
	firstValid := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 2, TickRate: 64,
		Position: r3.Vector{X: 1}, IsAlive: true,
	})
	if firstValid.Available {
		t.Fatalf("invalid position became an anchor: %+v", firstValid)
	}
}

func TestMotionTrackerDeadStateIsNotApplicableEvenWithoutPosition(t *testing.T) {
	var tracker MotionTracker
	estimate := tracker.Observe(MotionObservation{
		PlayerID: 1, Round: 1, Tick: 1,
		Position: r3.Vector{X: math.NaN()}, IsAlive: false,
	})
	if estimate.Available || estimate.Source != VelocitySourceNotApplicable {
		t.Fatalf("dead estimate = %+v", estimate)
	}
}

func assertClose(t *testing.T, actual, expected float64) {
	t.Helper()
	if math.Abs(actual-expected) > 1e-9 {
		t.Fatalf("actual=%f expected=%f", actual, expected)
	}
}
