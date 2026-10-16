package utility

import (
	"math"
	"reflect"
	"testing"
)

func TestInitialVelocityCandidatePrefersNativeAndPreservesZeroComponent(t *testing.T) {
	tracker := NewTracker()
	input := testThrowInput(1, 17, TypeSmoke, 9, Vector{X: 100, Y: 200, Z: 300})
	input.Launch.ProjectileInitialVelocity = VelocityObservation{
		Vector: Vector{X: 0, Y: 700, Z: 180}, HorizontalSpeed: 700,
		ObservedTick: 100, MeasurementWindowTicks: 0,
		Status: AvailabilityObserved, Source: SourceProjectileVelocity,
	}
	id, _ := tracker.RecordThrow(input)

	got := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Status != AvailabilityObserved || got.Source != SourceProjectileVelocity ||
		got.Vector.X != 0 || got.Vector.Y != 700 || got.Vector.Z != 180 ||
		got.ObservedTick != 100 || got.AvailabilityTick != 100 || got.MeasurementWindowTicks != 0 {
		t.Fatalf("unexpected native candidate: %+v", got)
	}
	future := testInitialVelocityPosition(input, 101, Vector{X: float64(math.MaxFloat32)}, 64)
	future.ActorID = 999
	future.Type = TypeHE
	tracker.RecordInitialVelocityPosition(future)
	after := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if !reflect.DeepEqual(got, after) {
		t.Fatalf("future position changed native fact: before=%+v after=%+v", got, after)
	}
}

func TestInitialVelocityCandidateFallsBackCausallyAtSecondPosition(t *testing.T) {
	tracker := NewTracker()
	input := testThrowInput(2, 21, TypeFlashbang, 7, Vector{X: 10, Y: 20, Z: 30})
	input.Launch.ProjectileInitialVelocity = VelocityObservation{
		Vector: Vector{}, ObservedTick: 200, Status: AvailabilityObserved, Source: SourceProjectileVelocity,
	}
	id, _ := tracker.RecordThrow(input)
	before := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if before.Status != AvailabilityUnavailable || before.Reason != InitialVelocityReasonNativeZeroAmbiguous {
		t.Fatalf("native zero was trusted: %+v", before)
	}
	if tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 200, input.Launch.Position.Value, 64)) {
		t.Fatal("same-tick position fabricated an initial velocity")
	}
	if !tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 202, Vector{X: 10, Y: 40, Z: 35}, 64)) {
		t.Fatal("causal position delta was not recorded")
	}

	got := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Status != AvailabilityObserved || got.Source != SourceProjectilePositionDelta ||
		got.ObservedTick != 202 || got.AvailabilityTick != 202 || got.MeasurementStartTick != 200 ||
		got.MeasurementEndTick != 202 || got.MeasurementWindowTicks != 2 || got.TickRate != 64 ||
		got.Vector.X != 0 || got.Vector.Y != 640 || got.Vector.Z != 160 {
		t.Fatalf("unexpected derived candidate: %+v", got)
	}
	future := testInitialVelocityPosition(input, 203, Vector{X: float64(math.MaxFloat32)}, 64)
	future.ActorID = 999
	future.Type = TypeHE
	tracker.RecordInitialVelocityPosition(future)
	tracker.RecordTrajectoryPosition(2, 21, 204, Vector{X: -1000})
	tracker.RecordBounce(
		input.Round,
		input.RuntimeEntityID,
		input.RuntimeEntitySerial,
		input.RuntimeEntitySerialStatus,
		203,
		Vector{X: 5000},
		AvailabilityObserved,
		1,
	)
	tracker.RecordDestroy(CallbackHint{
		Round: input.Round, RuntimeEntityID: input.RuntimeEntityID,
		RuntimeEntitySerial:       input.RuntimeEntitySerial,
		RuntimeEntitySerialStatus: input.RuntimeEntitySerialStatus,
		EntitySource:              SourceProjectileEntity, Type: input.Type,
		ActorID: input.Actor.ID, Tick: 204,
	})
	after := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if !reflect.DeepEqual(got, after) {
		t.Fatalf("future trajectory changed frozen fact: before=%+v after=%+v", got, after)
	}
}

func TestInitialVelocityCandidateDuplicateSameTickPreservesLegacyThrowDedup(t *testing.T) {
	tracker := NewTracker()
	input := testThrowInput(13, 131, TypeSmoke, 23, Vector{X: 1, Y: 2, Z: 3})
	id, created := tracker.RecordThrow(input)
	if !created {
		t.Fatal("first throw was not created")
	}
	conflict := input
	conflict.RuntimeEntitySerial++
	conflict.Actor.ID++
	conflict.Type = TypeHE
	duplicateID, duplicateCreated := tracker.RecordThrow(conflict)
	if duplicateCreated || duplicateID != id {
		t.Fatalf("same-tick duplicate escaped legacy dedup: id=%q created=%v", duplicateID, duplicateCreated)
	}
	throws := tracker.Snapshot()
	if len(throws) != 1 || throws[0].ID != id || throws[0].Sequence != 1 ||
		throws[0].SourceEntityGeneration != 1 || throws[0].Actor.ID != input.Actor.ID ||
		throws[0].Type != input.Type {
		t.Fatalf("candidate metadata changed legacy throw shape: %+v", throws)
	}
}

func TestInitialVelocityCandidateRejectsSentinelInvalidRateAndSameTickConflict(t *testing.T) {
	tracker := NewTracker()
	sentinel := float64(math.MaxFloat32)
	input := testThrowInput(3, 31, TypeHE, 5, Vector{X: 1})
	input.Launch.ProjectileInitialVelocity = VelocityObservation{
		Vector:          Vector{X: sentinel, Y: sentinel, Z: sentinel},
		HorizontalSpeed: math.Hypot(sentinel, sentinel), ObservedTick: 300,
		Status: AvailabilityObserved, Source: SourceProjectileVelocity,
	}
	id, _ := tracker.RecordThrow(input)
	got := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Status != AvailabilityUnavailable || got.Reason != InitialVelocityReasonNativeSentinel {
		t.Fatalf("sentinel was not rejected: %+v", got)
	}
	if tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 301, Vector{X: 2}, math.NaN())) {
		t.Fatal("invalid tick rate was accepted")
	}
	got = throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Reason != InitialVelocityReasonTickRateUnavailable || !got.Terminal {
		t.Fatalf("invalid tick rate did not fail closed: %+v", got)
	}

	tracker = NewTracker()
	input = testThrowInput(4, 41, TypeDecoy, 6, Vector{})
	id, _ = tracker.RecordThrow(input)
	if !tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 401, Vector{X: 1}, 64)) {
		t.Fatal("first valid position delta was not accepted")
	}
	if tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 401, Vector{X: 2}, 64)) {
		t.Fatal("conflicting same-tick position was accepted")
	}
	got = throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Status != AvailabilityUnavailable || got.Reason != InitialVelocityReasonSameTickConflict || !got.Terminal {
		t.Fatalf("same-tick conflict did not abstain: %+v", got)
	}
}

func TestInitialVelocityCandidateRejectsSentinelEndpointsAndLaunchTickConflict(t *testing.T) {
	sentinel := float64(math.MaxFloat32)
	tracker := NewTracker()
	input := testThrowInput(5, 51, TypeSmoke, 15, Vector{X: sentinel, Y: sentinel, Z: sentinel})
	id, _ := tracker.RecordThrow(input)
	if tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 501, Vector{X: 1}, 64)) {
		t.Fatal("sentinel launch position was accepted")
	}
	got := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Reason != InitialVelocityReasonNativeSentinel || !got.Terminal {
		t.Fatalf("sentinel launch endpoint did not fail closed: %+v", got)
	}

	tracker = NewTracker()
	input = testThrowInput(6, 61, TypeHE, 16, Vector{X: 1, Y: 2, Z: 3})
	id, _ = tracker.RecordThrow(input)
	if tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(
		input,
		601,
		Vector{X: sentinel, Y: sentinel, Z: sentinel},
		64,
	)) {
		t.Fatal("sentinel end position was accepted")
	}
	got = throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Reason != InitialVelocityReasonNativeSentinel || !got.Terminal {
		t.Fatalf("sentinel end endpoint did not fail closed: %+v", got)
	}

	tracker = NewTracker()
	input = testThrowInput(7, 71, TypeDecoy, 17, Vector{X: 10})
	id, _ = tracker.RecordThrow(input)
	if tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 700, Vector{X: 11}, 64)) {
		t.Fatal("different same-launch-tick position was accepted")
	}
	got = throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Reason != InitialVelocityReasonSameTickConflict || !got.Terminal {
		t.Fatalf("launch-tick conflict did not abstain: %+v", got)
	}
}

func TestInitialVelocityCandidateRequiresExactPhysicalActorAndTypeIdentity(t *testing.T) {
	tracker := NewTracker()
	input := testThrowInput(8, 81, TypeFlashbang, 18, Vector{})
	id, _ := tracker.RecordThrow(input)
	wrongSerial := testInitialVelocityPosition(input, 801, Vector{X: 1}, 64)
	wrongSerial.RuntimeEntitySerial++
	if tracker.RecordInitialVelocityPosition(wrongSerial) {
		t.Fatal("reused entity ID with another serial attached to the old throw")
	}
	before := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if before.Terminal {
		t.Fatalf("unknown future serial retroactively invalidated the old throw: %+v", before)
	}

	wrongActor := testInitialVelocityPosition(input, 801, Vector{X: 1}, 64)
	wrongActor.ActorID++
	if tracker.RecordInitialVelocityPosition(wrongActor) {
		t.Fatal("actor mismatch was accepted")
	}
	got := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Reason != InitialVelocityReasonIdentityConflict || !got.Terminal {
		t.Fatalf("actor mismatch did not fail closed: %+v", got)
	}

	tracker = NewTracker()
	input = testThrowInput(9, 91, TypeSmoke, 19, Vector{})
	id, _ = tracker.RecordThrow(input)
	wrongType := testInitialVelocityPosition(input, 901, Vector{X: 1}, 64)
	wrongType.Type = TypeHE
	if tracker.RecordInitialVelocityPosition(wrongType) {
		t.Fatal("utility type mismatch was accepted")
	}
	got = throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Reason != InitialVelocityReasonIdentityConflict || !got.Terminal {
		t.Fatalf("type mismatch did not fail closed: %+v", got)
	}

	empty := NewTracker()
	if empty.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 901, Vector{X: 1}, 64)) ||
		len(empty.Snapshot()) != 0 {
		t.Fatal("a FrameDone sample without a throw callback fabricated a throw")
	}
}

func TestInitialVelocityCandidateAllowsDistinctObservedSerialsWithoutChangingPriorFact(t *testing.T) {
	tracker := NewTracker()
	first := testThrowInput(10, 101, TypeSmoke, 20, Vector{})
	firstID, _ := tracker.RecordThrow(first)
	if !tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(first, 1001, Vector{X: 1}, 64)) {
		t.Fatal("first physical entity did not produce its causal fact")
	}
	before := throwByID(t, tracker.Snapshot(), firstID).InitialVelocityCandidate

	second := first
	second.RuntimeEntitySerial++
	second.Launch.Tick.Tick = 1005
	second.Launch.Position.Value = Vector{X: 10}
	secondID, _ := tracker.RecordThrow(second)
	if !tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(second, 1006, Vector{X: 11}, 64)) {
		t.Fatal("distinct observed serial did not produce an isolated candidate fact")
	}
	secondCandidate := throwByID(t, tracker.Snapshot(), secondID).InitialVelocityCandidate
	if secondCandidate.Status != AvailabilityObserved || secondCandidate.RuntimeEntitySerial != second.RuntimeEntitySerial {
		t.Fatalf("distinct observed serial was treated as ambiguous: %+v", secondCandidate)
	}
	after := throwByID(t, tracker.Snapshot(), firstID).InitialVelocityCandidate
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("distinct future serial changed the prior fact: before=%+v after=%+v", before, after)
	}
}

func TestInitialVelocityCandidateAbstainsOnExactPhysicalScopeReuse(t *testing.T) {
	tracker := NewTracker()
	first := testThrowInput(11, 111, TypeSmoke, 21, Vector{})
	firstID, _ := tracker.RecordThrow(first)
	if !tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(first, 1101, Vector{X: 1}, 64)) {
		t.Fatal("first physical scope did not produce its causal fact")
	}
	before := throwByID(t, tracker.Snapshot(), firstID).InitialVelocityCandidate

	reused := first
	reused.Launch.Tick.Tick = 1105
	reused.Launch.Position.Value = Vector{X: 10}
	reusedID, _ := tracker.RecordThrow(reused)
	reusedCandidate := throwByID(t, tracker.Snapshot(), reusedID).InitialVelocityCandidate
	if reusedCandidate.Reason != InitialVelocityReasonEntityReused || !reusedCandidate.Terminal {
		t.Fatalf("same exact physical scope reuse did not abstain: %+v", reusedCandidate)
	}
	after := throwByID(t, tracker.Snapshot(), firstID).InitialVelocityCandidate
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("future exact-scope reuse changed the prior fact: before=%+v after=%+v", before, after)
	}
}

func TestInitialVelocityCandidateBoundaryOrderAndZeroConflictAreDeterministic(t *testing.T) {
	for _, boundary := range []string{"bounce", "destroy"} {
		t.Run(boundary, func(t *testing.T) {
			left := initialVelocityBoundaryPermutation(t, boundary, true)
			right := initialVelocityBoundaryPermutation(t, boundary, false)
			if !reflect.DeepEqual(left, right) || left.Status != AvailabilityUnavailable ||
				left.Reason != InitialVelocityReasonFlightEnded || !left.Terminal {
				t.Fatalf("boundary order changed result: frame-first=%+v boundary-first=%+v", left, right)
			}
		})
	}

	tracker := NewTracker()
	input := testThrowInput(12, 121, TypeDecoy, 22, Vector{X: 5})
	id, _ := tracker.RecordThrow(input)
	if tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 1201, Vector{X: 5}, 64)) {
		t.Fatal("zero delta was accepted")
	}
	if tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 1201, Vector{X: 6}, 64)) {
		t.Fatal("second incompatible endpoint in the same tick was accepted")
	}
	got := throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
	if got.Reason != InitialVelocityReasonSameTickConflict || !got.Terminal {
		t.Fatalf("zero-to-nonzero same-tick conflict did not abstain: %+v", got)
	}
}

func initialVelocityBoundaryPermutation(t *testing.T, boundary string, frameFirst bool) InitialVelocityCandidate {
	t.Helper()
	tracker := NewTracker()
	input := testThrowInput(11, 111, TypeMolotov, 21, Vector{})
	id, _ := tracker.RecordThrow(input)
	frame := func() {
		tracker.RecordInitialVelocityPosition(testInitialVelocityPosition(input, 1101, Vector{X: 1}, 64))
	}
	boundaryEvent := func() {
		switch boundary {
		case "bounce":
			tracker.RecordBounce(
				input.Round,
				input.RuntimeEntityID,
				input.RuntimeEntitySerial,
				input.RuntimeEntitySerialStatus,
				1101,
				Vector{X: 1},
				AvailabilityObserved,
				1,
			)
		case "destroy":
			tracker.RecordDestroy(CallbackHint{
				Round: input.Round, RuntimeEntityID: input.RuntimeEntityID,
				RuntimeEntitySerial:       input.RuntimeEntitySerial,
				RuntimeEntitySerialStatus: input.RuntimeEntitySerialStatus,
				EntitySource:              SourceProjectileEntity, Type: input.Type,
				ActorID: input.Actor.ID, Tick: 1101,
			})
		default:
			t.Fatalf("unknown boundary %q", boundary)
		}
	}
	if frameFirst {
		frame()
		boundaryEvent()
	} else {
		boundaryEvent()
		frame()
	}
	return throwByID(t, tracker.Snapshot(), id).InitialVelocityCandidate
}

func testInitialVelocityPosition(
	input ThrowInput,
	tick int,
	position Vector,
	tickRate float64,
) InitialVelocityPositionInput {
	return InitialVelocityPositionInput{
		Round:                     input.Round,
		RuntimeEntityID:           input.RuntimeEntityID,
		RuntimeEntitySerial:       input.RuntimeEntitySerial,
		RuntimeEntitySerialStatus: input.RuntimeEntitySerialStatus,
		ActorID:                   input.Actor.ID,
		Type:                      input.Type,
		Tick:                      tick,
		Position:                  position,
		TickRate:                  tickRate,
	}
}
