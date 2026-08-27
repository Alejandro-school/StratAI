package utility

import (
	"reflect"
	"testing"
)

func TestTrackerLifecycleAndConfirmedExtinguish(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(1)
	smokeID, _ := tracker.RecordThrow(testThrowInput(1, 10, TypeSmoke, 1, Vector{X: 100}))
	tracker.StartEffect(EffectInput{
		Hint:                  testHint(1, 10, TypeSmoke, 1, 110, Vector{X: 100}),
		RuntimeEffectEntityID: 10,
		Source:                SourceSmokeStart,
	})
	fireID, _ := tracker.RecordThrow(testThrowInput(1, 20, TypeMolotov, 2, Vector{X: 120}))
	tracker.StartEffect(EffectInput{
		Hint:                  testHint(1, 0, TypeMolotov, 2, 112, Vector{X: 120}),
		RuntimeEffectEntityID: 70,
		Source:                SourceInfernoStart,
	})

	marked := tracker.MarkExtinguishCandidates(
		smokeID,
		1,
		113,
		Vector{X: 100},
		180,
		Correlation{Status: CorrelationObserved, Source: SourceGrenadeEntityID},
	)
	if marked != 1 {
		t.Fatalf("expected one extinguish candidate, got %d", marked)
	}
	before := throwByID(t, tracker.Snapshot(), fireID)
	if before.Lifecycle.EndReason == EndReasonSmokeExtinguished {
		t.Fatal("spatial overlap alone must not confirm an extinguish")
	}

	tracker.ExpireEffect(EffectInput{
		Hint:                  testHint(1, 0, TypeMolotov, 2, 114, Vector{X: 120}),
		RuntimeEffectEntityID: 70,
		Source:                SourceInfernoExpired,
	}, 64)
	after := throwByID(t, tracker.Snapshot(), fireID)
	if after.Lifecycle.Status != LifecycleEffectExpired ||
		after.Lifecycle.EndReason != EndReasonSmokeExtinguished ||
		after.Lifecycle.ExtinguishedByThrowID.Value != smokeID ||
		after.Lifecycle.ExtinguishAttribution.Status != CorrelationInferred ||
		after.Lifecycle.ExtinguishAttribution.Source != SourceSpatialSmokeOverlap ||
		after.Lifecycle.Duration.Value != 2.0/64.0 {
		t.Fatalf("unexpected confirmed lifecycle: %+v", after.Lifecycle)
	}
}

func TestTrackerCorrelatesPlayerFlashedByExactProjectile(t *testing.T) {
	tracker := NewTracker()
	firstID, _ := tracker.RecordThrow(testThrowInput(2, 11, TypeFlashbang, 9, Vector{X: 0}))
	secondID, _ := tracker.RecordThrow(testThrowInput(2, 12, TypeFlashbang, 9, Vector{X: 500}))

	tracker.RecordFlash(FlashInput{
		Round: 2, RuntimeEntityID: 12, Tick: 220,
		Victim:   testPlayer(8, "enemy", "CT"),
		Relation: RelationEnemy,
		Duration: ScalarObservation{Value: 1.5, Status: AvailabilityObserved, Source: SourcePlayerFlashed},
	})

	first := throwByID(t, tracker.Snapshot(), firstID)
	second := throwByID(t, tracker.Snapshot(), secondID)
	if len(first.Flashes) != 0 || len(second.Flashes) != 1 || second.Flashes[0].Victim.ID != 8 ||
		second.Flashes[0].Correlation.Status != CorrelationObserved ||
		second.Flashes[0].Correlation.Source != SourceProjectileEntity {
		t.Fatalf("flash was not correlated exactly: first=%+v second=%+v", first.Flashes, second.Flashes)
	}
}

func TestTrackerDoesNotCorrelateDelayedFlashToReusedSmokeEntity(t *testing.T) {
	tracker := NewTracker()
	flashInput := testThrowInput(2, 11, TypeFlashbang, 9, Vector{X: 0})
	flashInput.Launch.Tick.Tick = 200
	flashID, _ := tracker.RecordThrow(flashInput)
	tracker.RecordDetonation(CallbackHint{
		Round: 2, RuntimeEntityID: 11, EntitySource: SourceGrenadeEntityID,
		Type: TypeFlashbang, ActorID: 9, Tick: 220, TickRate: 64,
		Position: Vector{X: 10}, PositionStatus: AvailabilityObserved,
	}, SourceFlashExplode)

	smokeInput := testThrowInput(2, 11, TypeSmoke, 7, Vector{X: 500})
	smokeInput.Launch.Tick.Tick = 300
	smokeID, _ := tracker.RecordThrow(smokeInput)

	resolvedID, recorded := tracker.RecordFlash(FlashInput{
		Round: 2, RuntimeEntityID: 11, ActorID: 9, Tick: 320, TickRate: 64,
		Actor:  testPlayer(9, "flash-thrower", "CT"),
		Victim: testPlayer(8, "victim", "T"), Relation: RelationEnemy,
		Duration: ScalarObservation{Value: 1.5, Status: AvailabilityObserved, Source: SourcePlayerFlashed},
	})

	flash := throwByID(t, tracker.Snapshot(), flashID)
	smoke := throwByID(t, tracker.Snapshot(), smokeID)
	if !recorded || resolvedID != flashID || len(flash.Flashes) != 1 {
		t.Fatalf("delayed flash did not return to its compatible throw: resolved=%q recorded=%t flash=%+v", resolvedID, recorded, flash.Flashes)
	}
	if len(smoke.Flashes) != 0 {
		t.Fatalf("reused smoke entity was contaminated by flash effects: %+v", smoke.Flashes)
	}
	if flash.Flashes[0].Correlation.Status != CorrelationInferred ||
		flash.Flashes[0].Correlation.Source != SourceThrowerTypePositionTick {
		t.Fatalf("fallback flash correlation provenance is wrong: %+v", flash.Flashes[0].Correlation)
	}
}

func TestTrackerAnchorsFlashEffectsToDetonationAcrossCallbackOrder(t *testing.T) {
	for _, detonationFirst := range []bool{false, true} {
		tracker := NewTracker()
		id, _ := tracker.RecordThrow(testThrowInput(2, 12, TypeFlashbang, 9, Vector{X: 500}))
		detonate := func() {
			tracker.RecordDetonation(CallbackHint{
				Round: 2, RuntimeEntityID: 12, EntitySource: SourceProjectileEntity,
				Type: TypeFlashbang, ActorID: 9, Tick: 220, PositionStatus: AvailabilityUnavailable,
			}, SourceFlashExplode)
		}
		flash := func() {
			tracker.RecordFlash(FlashInput{
				Round: 2, RuntimeEntityID: 12, ActorID: 9, Tick: 420,
				Victim: testPlayer(8, "enemy", "CT"), Relation: RelationEnemy,
				Duration: ScalarObservation{Value: 1.5, Status: AvailabilityObserved, Source: SourcePlayerFlashed},
			})
		}
		if detonationFirst {
			detonate()
			flash()
		} else {
			flash()
			detonate()
		}
		entry := throwByID(t, tracker.Snapshot(), id)
		if len(entry.Flashes) != 1 || entry.Flashes[0].Tick != entry.Lifecycle.Detonation.Tick || entry.Flashes[0].Tick != 220 {
			t.Fatalf("detonationFirst=%t did not anchor flash causally: lifecycle=%+v flashes=%+v", detonationFirst, entry.Lifecycle, entry.Flashes)
		}
	}
}

func TestTrackerFindsNilThrowerInfernoByDeterministicCausalEvidence(t *testing.T) {
	tracker := NewTracker()
	firstID, _ := tracker.RecordThrow(testThrowInput(3, 31, TypeIncendiary, 1, Vector{X: 0}))
	secondID, _ := tracker.RecordThrow(testThrowInput(3, 32, TypeIncendiary, 2, Vector{X: 900}))
	tracker.RecordDestroy(testHint(3, 31, TypeIncendiary, 1, 330, Vector{X: 50}))
	tracker.RecordDestroy(testHint(3, 32, TypeIncendiary, 2, 331, Vector{X: 950}))

	resolvedID, _ := tracker.StartEffect(EffectInput{
		Hint: CallbackHint{
			Round: 3, Type: TypeUnknown, TypeFamily: TypeFamilyFire, Tick: 332,
			Position: Vector{X: 948}, PositionStatus: AvailabilityObserved,
		},
		RuntimeEffectEntityID: 80,
		Source:                SourceInfernoStart,
	})
	if resolvedID != secondID || resolvedID == firstID {
		t.Fatalf("nil-thrower inferno resolved to %q, want %q", resolvedID, secondID)
	}
	resolved := throwByID(t, tracker.Snapshot(), secondID)
	if resolved.Type != TypeIncendiary || resolved.TypeSource != SourceWeaponInstance {
		t.Fatalf("inferred inferno overwrote observed throw type: %+v", resolved)
	}
	if resolved.Lifecycle.Correlation.Status != CorrelationInferred ||
		resolved.Lifecycle.Correlation.Source != SourceTypePositionTick {
		t.Fatalf("missing explicit inferred provenance: %+v", resolved.Lifecycle.Correlation)
	}
}

func TestOrphanInfernoKeepsTypeUnavailable(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.StartEffect(EffectInput{
		Hint: CallbackHint{
			Round: 16, Type: TypeUnknown, TypeFamily: TypeFamilyFire, Tick: 1600,
			Position: Vector{X: 42}, PositionStatus: AvailabilityObserved,
		},
		RuntimeEffectEntityID: 88,
		Source:                SourceInfernoStart,
	})
	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Type != TypeUnknown || entry.TypeSource != SourceUnavailable ||
		entry.Launch.Tick.Status != AvailabilityUnavailable {
		t.Fatalf("orphan inferno fabricated a throw type or launch: %+v", entry)
	}
}

func TestCallbackEnrichesMissingActorWithoutOverwritingConflict(t *testing.T) {
	tracker := NewTracker()
	input := testThrowInput(18, 181, TypeFlashbang, 0, Vector{})
	input.Actor = PlayerRef{Status: AvailabilityUnavailable, Source: SourceUnavailable}
	id, _ := tracker.RecordThrow(input)
	tracker.RecordFlash(FlashInput{
		Round: 18, RuntimeEntityID: 181, Tick: 1810,
		Actor:  testPlayer(8, "observed", "T"),
		Victim: testPlayer(9, "victim", "CT"), Relation: RelationEnemy,
		Duration: ScalarObservation{Value: 1, Status: AvailabilityObserved, Source: SourcePlayerFlashed},
	})
	tracker.RecordDetonation(CallbackHint{
		Round: 18, RuntimeEntityID: 181, EntitySource: SourceProjectileEntity,
		Type: TypeFlashbang, ActorID: 10, Actor: testPlayer(10, "conflict", "CT"), Tick: 1810,
		PositionStatus: AvailabilityUnavailable,
	}, SourceFlashExplode)
	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Actor.ID != 8 || entry.Actor.Name != "observed" {
		t.Fatalf("callback actor was not enriched safely: %+v", entry.Actor)
	}
	diagnostics := tracker.Diagnostics()
	if diagnostics.ActorEnriched != 1 || diagnostics.ActorConflicts != 1 {
		t.Fatalf("actor diagnostics are incomplete: %+v", diagnostics)
	}
}

func TestTrackerNamespacesReusedEntityIDByRoundAndGeneration(t *testing.T) {
	tracker := NewTracker()
	first, _ := tracker.RecordThrow(testThrowInput(4, 7, TypeHE, 1, Vector{}))
	tracker.RecordDestroy(testHint(4, 7, TypeHE, 1, 410, Vector{X: 10}))
	secondInput := testThrowInput(4, 7, TypeSmoke, 1, Vector{X: 20})
	secondInput.Launch.Tick.Tick++
	second, _ := tracker.RecordThrow(secondInput)
	third, _ := tracker.RecordThrow(testThrowInput(5, 7, TypeFlashbang, 1, Vector{X: 30}))

	snapshot := tracker.Snapshot()
	firstThrow := throwByID(t, snapshot, first)
	secondThrow := throwByID(t, snapshot, second)
	thirdThrow := throwByID(t, snapshot, third)
	if firstThrow.SourceEntityGeneration != 1 || secondThrow.SourceEntityGeneration != 2 ||
		thirdThrow.SourceEntityGeneration != 1 {
		t.Fatalf("unexpected generations: %d %d %d", firstThrow.SourceEntityGeneration, secondThrow.SourceEntityGeneration, thirdThrow.SourceEntityGeneration)
	}
}

func TestSnapshotOrdersFlashEffectsIndependentlyOfCallbackPermutation(t *testing.T) {
	inputs := []FlashInput{
		{Round: 6, RuntimeEntityID: 60, Tick: 620, Victim: testPlayer(3, "c", "CT"), Relation: RelationEnemy},
		{Round: 6, RuntimeEntityID: 60, Tick: 620, Victim: testPlayer(1, "a", "CT"), Relation: RelationEnemy},
		{Round: 6, RuntimeEntityID: 60, Tick: 620, Victim: testPlayer(2, "b", "T"), Relation: RelationTeammate},
	}
	left := snapshotFlashEffects(inputs)
	inputs[0], inputs[2] = inputs[2], inputs[0]
	right := snapshotFlashEffects(inputs)
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("callback permutation changed snapshot:\nleft=%+v\nright=%+v", left, right)
	}
}

func TestEndRoundClassifiesNonDetonatingThrows(t *testing.T) {
	tracker := NewTracker()
	destroyedID, _ := tracker.RecordThrow(testThrowInput(7, 71, TypeDecoy, 1, Vector{}))
	unresolvedID, _ := tracker.RecordThrow(testThrowInput(7, 72, TypeSmoke, 1, Vector{}))
	tracker.RecordDestroy(testHint(7, 71, TypeDecoy, 1, 710, Vector{}))
	tracker.EndRound(7)

	destroyed := throwByID(t, tracker.Snapshot(), destroyedID)
	unresolved := throwByID(t, tracker.Snapshot(), unresolvedID)
	if destroyed.Lifecycle.Status != LifecycleDestroyedWithoutDetonation ||
		unresolved.Lifecycle.Status != LifecycleRoundEndedUnresolved {
		t.Fatalf("wrong final states: %s %s", destroyed.Lifecycle.Status, unresolved.Lifecycle.Status)
	}
}

func TestDamageBeforeHEExplodeIsNotLost(t *testing.T) {
	tracker := NewTracker()
	throwInput := testThrowInput(8, 81, TypeHE, 4, Vector{X: 50})
	throwInput.Launch.Tick.Tick = 100
	id, _ := tracker.RecordThrow(throwInput)
	tracker.RecordDestroy(CallbackHint{
		Round: 8, RuntimeEntityID: 81, EntitySource: SourceProjectileEntity,
		Type: TypeHE, ActorID: 4, Tick: 200, TickRate: 64,
		Position: Vector{X: 80}, PositionStatus: AvailabilityObserved,
	})
	resolvedID, recorded := tracker.RecordDamage(DamageInput{
		Round: 8, Type: TypeHE, ActorID: 4, Tick: 201, TickRate: 64,
		Victim: testPlayer(5, "victim", "CT"), VictimPosition: Vector{X: 80},
		VictimPositionStatus: AvailabilityObserved,
		Relation:             RelationEnemy, HealthDamage: 48, ArmorDamage: 7,
	})
	if recorded || resolvedID != "" {
		t.Fatalf("pre-explode damage must remain pending: id=%q recorded=%v", resolvedID, recorded)
	}
	tracker.RecordDetonation(CallbackHint{
		Round: 8, RuntimeEntityID: 81, EntitySource: SourceProjectileEntity,
		Type: TypeHE, ActorID: 4, Tick: 201, TickRate: 64,
		Position: Vector{X: 80}, PositionStatus: AvailabilityObserved,
	}, SourceHEExplode)
	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Lifecycle.Status != LifecycleDetonated {
		t.Fatal("later explosion did not enrich the pre-lifecycle damage throw")
	}
	if len(entry.Damage) != 1 || entry.Damage[0].HealthDamage != 48 || entry.Damage[0].ArmorDamage != 7 {
		t.Fatalf("pending damage was not reconciled: %+v", entry.Damage)
	}
}

func TestInitialBounceAvailabilityIsNotFabricated(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(9, 91, TypeSmoke, 1, Vector{}))
	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Trajectory.BounceStatus != AvailabilityUnavailable || entry.Trajectory.BounceSource != SourceUnavailable {
		t.Fatalf("zero bounce was presented as observed: %+v", entry.Trajectory)
	}
}

func TestDestroyOnlyTrajectoryDoesNotFabricateFrameCoverage(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(17, 171, TypeSmoke, 1, Vector{}))
	tracker.RecordDestroy(testHint(17, 171, TypeSmoke, 1, 1710, Vector{X: 99}))
	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Trajectory.Status != TrajectoryPartial || entry.Trajectory.Source != SourceProjectileDestroy ||
		len(entry.Trajectory.Samples) != 1 || entry.Trajectory.Samples[0].Source != SourceProjectileDestroy {
		t.Fatalf("destroy-only trajectory fabricated frame coverage: %+v", entry.Trajectory)
	}
}

func TestTrajectoryDeduplicatesFinalHistoryAlreadySeenInFrames(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(18, 181, TypeHE, 1, Vector{}))

	first := Vector{X: 10, Y: 20, Z: 30}
	second := Vector{X: 40, Y: 50, Z: 60}
	tracker.RecordTrajectoryPosition(18, 181, 1801, first)
	tracker.RecordTrajectoryPosition(18, 181, 1802, second)
	tracker.RecordTrajectoryPosition(18, 181, 1801, first)
	tracker.RecordTrajectoryPosition(18, 181, 1802, second)
	tracker.RecordDestroy(testHint(18, 181, TypeHE, 1, 1803, second))

	entry := throwByID(t, tracker.Snapshot(), id)
	if len(entry.Trajectory.Samples) != 3 {
		t.Fatalf("trajectory history was duplicated: %+v", entry.Trajectory.Samples)
	}
	seen := make(map[trajectorySampleKey]struct{}, len(entry.Trajectory.Samples))
	for _, sample := range entry.Trajectory.Samples {
		key := trajectorySampleKey{tick: sample.Tick, position: sample.Position, source: sample.Source}
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("duplicate canonical trajectory sample: %+v", sample)
		}
		seen[key] = struct{}{}
	}
	if entry.Trajectory.Status != TrajectoryObserved || entry.Trajectory.Source != SourceProjectileFrames {
		t.Fatalf("trajectory provenance changed after deduplication: %+v", entry.Trajectory)
	}
}

func TestTrajectoryDestroyReplacesSameTickFrameAndRemainsTerminal(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(19, 191, TypeHE, 1, Vector{}))
	first := Vector{X: 10, Y: 20, Z: 30}
	terminal := Vector{X: 40, Y: 50, Z: 60}
	tracker.RecordTrajectoryPosition(19, 191, 1901, first)
	tracker.RecordTrajectoryPosition(19, 191, 1902, terminal)
	tracker.RecordDestroy(testHint(19, 191, TypeHE, 1, 1902, terminal))

	if tracker.RecordTrajectoryPosition(19, 191, 1902, terminal) ||
		tracker.RecordTrajectoryPosition(19, 191, 1903, terminal) {
		t.Fatal("trajectory accepted a frame at or after projectile destroy")
	}
	entry := throwByID(t, tracker.Snapshot(), id)
	if len(entry.Trajectory.Samples) != 2 ||
		entry.Trajectory.Samples[0].Source != SourceProjectileFrames ||
		entry.Trajectory.Samples[1].Source != SourceProjectileDestroy ||
		entry.Trajectory.Samples[1].Tick != entry.Lifecycle.Destroy.Tick {
		t.Fatalf("destroy observation is not the unique terminal sample: %+v", entry.Trajectory.Samples)
	}
}

func TestDetonationPrunesTrajectoryAtAndAfterFlightTerminal(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(21, 211, TypeHE, 1, Vector{}))
	for tick := 2101; tick <= 2103; tick++ {
		tracker.RecordTrajectoryPosition(21, 211, tick, Vector{X: float64(tick)})
	}
	tracker.RecordDetonation(
		testHint(21, 211, TypeHE, 1, 2102, Vector{X: 2102}),
		SourceHEExplode,
	)
	if tracker.RecordTrajectoryPosition(21, 211, 2102, Vector{X: 999}) ||
		tracker.RecordTrajectoryPosition(21, 211, 2104, Vector{X: 999}) {
		t.Fatal("trajectory accepted a sample at or after detonation")
	}
	tracker.RecordDestroy(testHint(21, 211, TypeHE, 1, 2110, Vector{X: 2110}))

	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Lifecycle.Destroy.Status != AvailabilityObserved ||
		entry.Trajectory.Status != TrajectoryObserved ||
		entry.Trajectory.Source != SourceProjectileFrames ||
		len(entry.Trajectory.Samples) != 1 || entry.Trajectory.Samples[0].Tick != 2101 {
		t.Fatalf("detonation did not close flight trajectory exactly: %+v", entry)
	}
}

func TestEffectStartPrunesTrajectoryAtAndAfterFlightTerminal(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(22, 221, TypeSmoke, 1, Vector{}))
	for tick := 2201; tick <= 2203; tick++ {
		tracker.RecordTrajectoryPosition(22, 221, tick, Vector{X: float64(tick)})
	}
	tracker.StartEffect(EffectInput{
		Hint:                  testHint(22, 221, TypeSmoke, 1, 2202, Vector{X: 2202}),
		RuntimeEffectEntityID: 222,
		Source:                SourceSmokeStart,
	})

	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Trajectory.Status != TrajectoryObserved ||
		entry.Trajectory.Source != SourceProjectileFrames ||
		len(entry.Trajectory.Samples) != 1 || entry.Trajectory.Samples[0].Tick != 2201 {
		t.Fatalf("effect start did not close flight trajectory exactly: %+v", entry.Trajectory)
	}
}

func TestFlightTerminalIsIndependentOfCallbackPermutation(t *testing.T) {
	buildSnapshot := func(terminalFirst bool) (Throw, CallbackDiagnostics) {
		tracker := NewTracker()
		id, _ := tracker.RecordThrow(testThrowInput(23, 231, TypeHE, 1, Vector{}))
		detonate := func() {
			tracker.RecordDetonation(
				testHint(23, 231, TypeHE, 1, 2302, Vector{X: 2302}),
				SourceHEExplode,
			)
		}
		if terminalFirst {
			detonate()
		}
		for number, tick := range []int{2301, 2302, 2303} {
			position := Vector{X: float64(tick)}
			tracker.RecordTrajectoryPosition(23, 231, tick, position)
			tracker.RecordBounce(23, 231, tick, position, AvailabilityObserved, number+1)
		}
		tracker.RecordDestroy(testHint(23, 231, TypeHE, 1, 2304, Vector{X: 2304}))
		if !terminalFirst {
			detonate()
		}
		return throwByID(t, tracker.Snapshot(), id), tracker.Diagnostics().Bounces
	}

	early, earlyDiagnostics := buildSnapshot(true)
	late, lateDiagnostics := buildSnapshot(false)
	if !reflect.DeepEqual(early, late) || !reflect.DeepEqual(earlyDiagnostics, lateDiagnostics) {
		t.Fatalf("callback permutation changed trajectory:\nearly=%+v\nlate=%+v\nearly diagnostics=%+v\nlate diagnostics=%+v", early, late, earlyDiagnostics, lateDiagnostics)
	}
	if len(early.Trajectory.Samples) != 1 || early.Trajectory.Samples[0].Tick != 2301 ||
		len(early.Trajectory.Bounces) != 1 || early.Trajectory.Bounces[0].Tick != 2301 ||
		early.Trajectory.BounceCount != 1 || earlyDiagnostics.ExactCorrelated != 3 ||
		earlyDiagnostics.Unmatched != 0 {
		t.Fatalf("flight terminal retained invalid samples or bounces: %+v diagnostics=%+v", early.Trajectory, earlyDiagnostics)
	}
}

func TestDestroyWithoutPositionLeavesFrameTrajectoryPartial(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(24, 241, TypeDecoy, 1, Vector{}))
	tracker.RecordTrajectoryPosition(24, 241, 2401, Vector{X: 1})
	hint := testHint(24, 241, TypeDecoy, 1, 2402, Vector{})
	hint.PositionStatus = AvailabilityUnavailable
	tracker.RecordDestroy(hint)

	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Trajectory.Status != TrajectoryPartial ||
		entry.Trajectory.Source != SourceProjectileFrames ||
		len(entry.Trajectory.Samples) != 1 {
		t.Fatalf("destroy without position fabricated complete trajectory coverage: %+v", entry.Trajectory)
	}
}

func TestDestroyPrunesBouncesAtAndAfterTerminal(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(25, 251, TypeDecoy, 1, Vector{}))
	tracker.RecordBounce(25, 251, 2501, Vector{X: 1}, AvailabilityObserved, 1)
	tracker.RecordBounce(25, 251, 2502, Vector{X: 2}, AvailabilityObserved, 2)
	tracker.RecordDestroy(testHint(25, 251, TypeDecoy, 1, 2502, Vector{X: 2}))
	if tracker.RecordBounce(25, 251, 2503, Vector{X: 3}, AvailabilityObserved, 3) {
		t.Fatal("bounce after destroy was accepted")
	}

	entry := throwByID(t, tracker.Snapshot(), id)
	diagnostics := tracker.Diagnostics().Bounces
	if len(entry.Trajectory.Bounces) != 1 || entry.Trajectory.Bounces[0].Tick != 2501 ||
		entry.Trajectory.BounceCount != 1 || entry.Trajectory.BounceStatus != AvailabilityObserved ||
		diagnostics.Observed != 3 || diagnostics.ExactCorrelated != 3 || diagnostics.Unmatched != 0 {
		t.Fatalf("destroy cutoff retained invalid bounce coverage: %+v diagnostics=%+v", entry.Trajectory, diagnostics)
	}
}

func TestSparseDamageUsesCallbackActorProvenance(t *testing.T) {
	tracker := NewTracker()
	tracker.BeginRound(20)
	tracker.RecordDamage(DamageInput{
		Round: 20, Type: TypeHE, ActorID: 7,
		Actor:  PlayerRef{ID: 7, Name: "thrower", Side: "T", Status: AvailabilityObserved, Source: SourcePlayerHurt},
		Victim: PlayerRef{ID: 8, Name: "victim", Side: "CT", Status: AvailabilityObserved, Source: SourcePlayerHurt},
		Tick:   2001, TickRate: 64, Relation: RelationEnemy, HealthDamage: 10,
	})
	tracker.EndRound(20)

	throws := tracker.Snapshot()
	if len(throws) != 1 || throws[0].Actor.Status != AvailabilityObserved ||
		throws[0].Actor.Source != SourceCallbackActor {
		t.Fatalf("sparse callback actor kept effect provenance: %+v", throws)
	}
}

func TestInferredCorrelationRejectsOldOrDistantThrow(t *testing.T) {
	tracker := NewTracker()
	oldID, _ := tracker.RecordThrow(testThrowInput(10, 101, TypeMolotov, 0, Vector{}))
	resolvedID, _ := tracker.StartEffect(EffectInput{
		Hint: CallbackHint{
			Round: 10, Type: TypeMolotov, Tick: 2000, TickRate: 64,
			Position: Vector{X: 5000}, PositionStatus: AvailabilityObserved,
		},
		RuntimeEffectEntityID: 500,
		Source:                SourceInfernoStart,
	})
	if resolvedID == oldID {
		t.Fatal("an old and distant throw was correlated to an unrelated inferno")
	}
	if len(tracker.Snapshot()) != 2 {
		t.Fatalf("sparse lifecycle callback was not preserved: %+v", tracker.Snapshot())
	}
}

func TestSparseExpirationKeepsDurationUnavailable(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.ExpireEffect(EffectInput{
		Hint: CallbackHint{
			Round: 11, Type: TypeSmoke, Tick: 1100, TickRate: 64,
			Position: Vector{X: 12}, PositionStatus: AvailabilityObserved,
		},
		RuntimeEffectEntityID: 12,
		Source:                SourceSmokeExpired,
	}, 64)
	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Lifecycle.Status != LifecycleEffectExpired ||
		entry.Lifecycle.EffectStart.Status != AvailabilityUnavailable ||
		entry.Lifecycle.Duration.Status != AvailabilityUnavailable {
		t.Fatalf("sparse expiration invented lifecycle values: %+v", entry.Lifecycle)
	}
}

func TestPlayerFlashedWithoutFlashExplodeStillProvesDetonation(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(12, 121, TypeFlashbang, 1, Vector{}))
	for tick := 1209; tick <= 1211; tick++ {
		tracker.RecordTrajectoryPosition(12, 121, tick, Vector{X: float64(tick)})
	}
	tracker.RecordFlash(FlashInput{
		Round: 12, RuntimeEntityID: 121, Tick: 1210,
		Victim: testPlayer(2, "victim", "CT"), Relation: RelationEnemy,
		Duration: ScalarObservation{Value: 1, Status: AvailabilityObserved, Source: SourcePlayerFlashed},
	})
	tracker.EndRound(12)
	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.Lifecycle.Status != LifecycleDetonated ||
		entry.Lifecycle.Detonation.Status != AvailabilityObserved ||
		entry.Lifecycle.Detonation.PositionStatus != AvailabilityUnavailable ||
		entry.Trajectory.Status != TrajectoryObserved ||
		len(entry.Trajectory.Samples) != 1 || entry.Trajectory.Samples[0].Tick != 1209 {
		t.Fatalf("PlayerFlashed proof was discarded: %+v", entry.Lifecycle)
	}
}

func TestDiagnosticsAccountForEveryCallback(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordThrow(testThrowInput(13, 131, TypeFlashbang, 1, Vector{}))
	tracker.RecordThrow(testThrowInput(13, 131, TypeFlashbang, 1, Vector{}))
	tracker.RecordFlash(FlashInput{
		Round: 13, RuntimeEntityID: 131, Tick: 1310,
		Victim: testPlayer(2, "victim", "CT"), Relation: RelationEnemy,
	})
	tracker.RecordFlash(FlashInput{
		Round: 13, RuntimeEntityID: 131, Tick: 1310,
		Victim: testPlayer(2, "victim", "CT"), Relation: RelationEnemy,
	})
	diagnostics := tracker.Diagnostics()
	for name, metric := range map[string]CallbackDiagnostics{
		"throws":  diagnostics.Throws,
		"flashes": diagnostics.Flashes,
	} {
		accounted := metric.ExactCorrelated + metric.InferredCorrelated + metric.Orphaned +
			metric.Deduplicated + metric.Unmatched
		if metric.Observed != accounted {
			t.Fatalf("%s diagnostics do not reconcile: %+v", name, metric)
		}
	}
}

func TestGrenadeEntityProvenanceIsScopedAndPersisted(t *testing.T) {
	tracker := NewTracker()
	id, _ := tracker.RecordDetonation(CallbackHint{
		Round: 14, RuntimeEntityID: 44, EntitySource: SourceGrenadeEntityID,
		Type: TypeHE, Tick: 1400, PositionStatus: AvailabilityUnavailable,
	}, SourceHEExplode)
	entry := throwByID(t, tracker.Snapshot(), id)
	if entry.SourceEntityID != 44 || entry.SourceEntityGeneration != 1 ||
		entry.EntityStatus != AvailabilityObserved || entry.EntitySource != SourceGrenadeEntityID {
		t.Fatalf("grenade entity provenance was lost: %+v", entry)
	}
}

func TestPendingMolotovDamageReconcilesIntoSingleThrow(t *testing.T) {
	tracker := NewTracker()
	throwInput := testThrowInput(15, 151, TypeMolotov, 7, Vector{})
	throwInput.Launch.Tick.Tick = 100
	id, _ := tracker.RecordThrow(throwInput)
	tracker.RecordDestroy(CallbackHint{
		Round: 15, RuntimeEntityID: 151, EntitySource: SourceProjectileEntity,
		Type: TypeMolotov, ActorID: 7, Tick: 200, TickRate: 64,
		Position: Vector{X: 500}, PositionStatus: AvailabilityObserved,
	})
	for tick, damage := range map[int]int{511: 7, 512: 9} {
		tracker.RecordDamage(DamageInput{
			Round: 15, Type: TypeMolotov, ActorID: 7, Tick: tick, TickRate: 64,
			Victim: testPlayer(8, "victim", "CT"), VictimPosition: Vector{X: 510},
			VictimPositionStatus: AvailabilityObserved,
			Relation:             RelationEnemy, HealthDamage: damage,
		})
	}
	tracker.StartEffect(EffectInput{
		Hint: CallbackHint{
			Round: 15, Type: TypeMolotov, ActorID: 7, Tick: 512, TickRate: 64,
			Position: Vector{X: 500}, PositionStatus: AvailabilityObserved,
		},
		RuntimeEffectEntityID: 901,
		Source:                SourceInfernoStart,
	})
	entry := throwByID(t, tracker.Snapshot(), id)
	if len(tracker.Snapshot()) != 1 || len(entry.Damage) != 2 {
		t.Fatalf("pending damage did not reconcile into one throw: %+v", tracker.Snapshot())
	}
	diagnostics := tracker.Diagnostics().Damage
	if diagnostics.Observed != 2 || diagnostics.InferredCorrelated != 2 || diagnostics.Unmatched != 0 {
		t.Fatalf("pending damage diagnostics did not settle: %+v", diagnostics)
	}
}

func TestDiscardedCallbackDiagnosticsSeparateIntentionalAndInvalidInput(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordDiscardedCallback(CallbackGroupLifecycle, CallbackDiscardWarmup)
	tracker.RecordDiscardedCallback(CallbackGroupLifecycle, CallbackDiscardOutsideRound)
	tracker.RecordDiscardedCallback(CallbackGroupLifecycle, CallbackDiscardInvalid)

	diagnostics := tracker.Diagnostics()
	if diagnostics.Discarded.Lifecycle.Warmup != 1 ||
		diagnostics.Discarded.Lifecycle.OutsideRound != 1 ||
		diagnostics.Discarded.Lifecycle.Invalid != 1 {
		t.Fatalf("discard reasons were not retained: %+v", diagnostics.Discarded.Lifecycle)
	}
	if diagnostics.Lifecycle.Observed != 1 || diagnostics.Lifecycle.Unmatched != 1 {
		t.Fatalf("only invalid input should enter hard callback accounting: %+v", diagnostics.Lifecycle)
	}
}

func snapshotFlashEffects(inputs []FlashInput) []FlashEffect {
	tracker := NewTracker()
	id, _ := tracker.RecordThrow(testThrowInput(6, 60, TypeFlashbang, 9, Vector{}))
	for _, input := range inputs {
		input.Duration = ScalarObservation{Value: 1, Status: AvailabilityObserved, Source: SourcePlayerFlashed}
		tracker.RecordFlash(input)
	}
	return throwByID(nil, tracker.Snapshot(), id).Flashes
}

func testThrowInput(round, entityID int, utilityType Type, actorID uint64, position Vector) ThrowInput {
	return ThrowInput{
		Round: round, RuntimeEntityID: entityID, EntitySource: SourceProjectileEntity,
		Type: utilityType, TypeSource: SourceWeaponInstance,
		Actor: testPlayer(actorID, "thrower", "T"),
		Launch: ThrowSnapshot{
			Tick:     TickObservation{Tick: round * 100, Status: AvailabilityObserved, Source: SourceProjectileThrow},
			Position: VectorObservation{Value: position, Status: AvailabilityObserved, Source: SourceProjectilePosition},
		},
	}
}

func testHint(round, entityID int, utilityType Type, actorID uint64, tick int, position Vector) CallbackHint {
	return CallbackHint{
		Round: round, RuntimeEntityID: entityID, EntitySource: SourceProjectileEntity,
		Type: utilityType, ActorID: actorID, Tick: tick,
		Position: position, PositionStatus: AvailabilityObserved,
	}
}

func testPlayer(id uint64, name, side string) PlayerRef {
	return PlayerRef{ID: id, Name: name, Side: side, Status: AvailabilityObserved, Source: SourceProjectileThrower}
}

func throwByID(t *testing.T, snapshot []Throw, id string) Throw {
	for _, entry := range snapshot {
		if entry.ID == id {
			return entry
		}
	}
	if t != nil {
		t.Fatalf("throw %q not found", id)
	}
	panic("throw not found")
}
