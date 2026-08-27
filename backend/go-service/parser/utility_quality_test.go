package parser

import (
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
)

func TestCanonicalUtilityProjectionPreservesCausalObservations(t *testing.T) {
	throw := validUtilityQualityThrow()
	event := projectCanonicalUtilityThrows("utility-test", []utility.Throw{throw})[0]

	if event.SchemaID != "stratai.utility_event@2" || event.EventID != "utility-test:utility:r1-u0001" ||
		event.UtilityTypeStatus != string(utility.AvailabilityObserved) {
		t.Fatalf("unstable utility identity: %+v", event)
	}
	if event.Launch.ThrowerVelocity.MeasurementWindowTicks == nil || *event.Launch.ThrowerVelocity.MeasurementWindowTicks != 1 {
		t.Fatalf("thrower measurement window was lost: %+v", event.Launch.ThrowerVelocity)
	}
	if event.Launch.ProjectileInitialVelocity.MeasurementWindowTicks == nil ||
		*event.Launch.ProjectileInitialVelocity.MeasurementWindowTicks != 0 {
		t.Fatalf("instant projectile velocity lost its zero-tick measurement window: %+v", event.Launch.ProjectileInitialVelocity)
	}
	unavailable := canonicalUtilityVelocityObservation(utility.VelocityObservation{
		Status: utility.AvailabilityUnavailable,
		Source: utility.SourceUnavailable,
	})
	if unavailable.Vector != nil || unavailable.HorizontalWorldUnitsPerS != nil ||
		unavailable.ObservedTick != nil || unavailable.MeasurementWindowTicks != nil {
		t.Fatalf("unavailable velocity acquired observed values: %+v", unavailable)
	}
	if len(event.Trajectory.Samples) != 2 || len(event.Trajectory.Bounces) != 1 ||
		event.Trajectory.BounceCount == nil || *event.Trajectory.BounceCount != 1 {
		t.Fatalf("trajectory timing was not preserved: %+v", event.Trajectory)
	}
	if event.AffectedPlayers[0].PlayerSource != utility.SourcePlayerFlashed ||
		event.AffectedPlayers[0].BlindDuration.Milliseconds == nil ||
		*event.AffectedPlayers[0].BlindDuration.Milliseconds != 750 ||
		event.AffectedPlayers[0].BlindCorrelation.Status != string(utility.CorrelationObserved) {
		t.Fatalf("flash provenance or native duration was lost: %+v", event.AffectedPlayers[0])
	}
	if event.FlashSummary.PlayersTotal != 1 || event.FlashSummary.EnemiesFlashed != 1 ||
		event.FlashSummary.TotalDurationMS == nil || *event.FlashSummary.TotalDurationMS != 750 {
		t.Fatalf("flash summary does not reconcile: %+v", event.FlashSummary)
	}
}

func TestCanonicalUtilityProjectionPreservesCausalDestroyTerminal(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Lifecycle.Status = utility.LifecycleDestroyedWithoutDetonation
	throw.Lifecycle.Detonation = unavailableUtilityMoment()
	throw.Lifecycle.EndReason = utility.EndReasonDestroyed
	throw.Lifecycle.EndReasonSource = utility.SourceProjectileDestroy
	throw.Flashes = nil
	throw.Trajectory.Samples = append(throw.Trajectory.Samples, utility.TrajectorySample{
		Tick: 15, Position: utility.Vector{X: 15}, Source: utility.SourceProjectileDestroy,
	})

	if violations := utilityThrowContractViolations(throw); violations != 0 {
		t.Fatalf("causal destroy trajectory contract violations = %d", violations)
	}
	event := projectCanonicalUtilityThrows("utility-destroy", []utility.Throw{throw})[0]
	if len(event.Trajectory.Samples) != 3 ||
		event.Trajectory.Samples[2].Source != utility.SourceProjectileDestroy ||
		event.Trajectory.Samples[2].Tick != 15 {
		t.Fatalf("causal destroy terminal was not preserved: %+v", event.Trajectory)
	}
}

func TestCanonicalUtilityProjectionIsHashStableAcrossPermutations(t *testing.T) {
	first := validUtilityQualityThrow()
	second := validUtilityQualityThrow()
	second.ID, second.Sequence = "r1-u0002", 2
	second.SourceEntityID = 43
	second.Flashes = []utility.FlashEffect{
		validUtilityFlashEffect(3, utility.RelationEnemy, 0.25),
		validUtilityFlashEffect(2, utility.RelationEnemy, 0.75),
	}
	second.Trajectory.Samples[0], second.Trajectory.Samples[1] = second.Trajectory.Samples[1], second.Trajectory.Samples[0]

	left, err := json.Marshal(projectCanonicalUtilityThrows("utility-order", []utility.Throw{first, second}))
	if err != nil {
		t.Fatal(err)
	}
	second.Flashes[0], second.Flashes[1] = second.Flashes[1], second.Flashes[0]
	right, err := json.Marshal(projectCanonicalUtilityThrows("utility-order", []utility.Throw{second, first}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) || sha256.Sum256(left) != sha256.Sum256(right) {
		t.Fatalf("permuted utility observations changed canonical bytes\nleft:  %s\nright: %s", left, right)
	}
}

func TestUtilityQualityAcceptsCoherentThrow(t *testing.T) {
	throw := validUtilityQualityThrow()
	if violations := utilityThrowContractViolations(throw); violations != 0 {
		t.Fatalf("contract violations = %d", violations)
	}
	if violations := utilityLifecycleViolations(throw); violations != 0 {
		t.Fatalf("lifecycle violations = %d", violations)
	}
	if mismatches := utilityFlashAttributionMismatches(throw); mismatches != 0 {
		t.Fatalf("flash mismatches = %d", mismatches)
	}
	if violations := utilityTemporalSpatialViolations(throw, &models.ReplayRound{StartTick: 1, EndTick: 100}); violations != 0 {
		t.Fatalf("temporal/spatial violations = %d", violations)
	}
}

func TestUtilityTemporalSpatialAllowsCoherentLifecycleTerminalAfterRoundEnd(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Lifecycle.Expiration = observedUtilityMoment(110, utility.SourceInfernoExpired)

	if violations := utilityTemporalSpatialViolations(throw, &models.ReplayRound{StartTick: 1, EndTick: 100}); violations != 0 {
		t.Fatalf("coherent lifecycle terminal after round end was rejected: %d", violations)
	}
}

func TestUtilityLifecycleTerminalRoundGraceIsNarrow(t *testing.T) {
	bounds := &models.ReplayRound{StartTick: 1, EndTick: 100}

	if !utilityLifecycleMomentInsideBounds(110, bounds, true, true) {
		t.Fatal("coherent lifecycle terminal after round end was rejected")
	}
	if utilityLifecycleMomentInsideBounds(110, bounds, false, true) {
		t.Fatal("non-terminal lifecycle moment after round end was accepted")
	}
	if utilityLifecycleMomentInsideBounds(110, bounds, true, false) {
		t.Fatal("terminal with invalid lifecycle order was accepted")
	}
	if utilityLifecycleMomentInsideBounds(0, bounds, true, true) {
		t.Fatal("lifecycle terminal before round start was accepted")
	}
}

func TestUtilityTemporalSpatialStillRejectsOutOfRoundEffectsAndTrajectory(t *testing.T) {
	t.Run("effect", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Lifecycle.Expiration = observedUtilityMoment(110, utility.SourceInfernoExpired)
		throw.Flashes[0].Tick = 101
		if violations := utilityTemporalSpatialViolations(throw, &models.ReplayRound{StartTick: 1, EndTick: 100}); violations != 1 {
			t.Fatalf("out-of-round effect violations = %d, want 1", violations)
		}
	})

	t.Run("trajectory", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Lifecycle.Expiration = observedUtilityMoment(110, utility.SourceInfernoExpired)
		throw.Trajectory.Samples[1].Tick = 101
		if violations := utilityTemporalSpatialViolations(throw, &models.ReplayRound{StartTick: 1, EndTick: 100}); violations != 1 {
			t.Fatalf("out-of-round trajectory violations = %d, want 1", violations)
		}
	})
}

func TestUtilityDeterminismRejectsDuplicateTrajectoryObservations(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Trajectory.Samples = append(
		throw.Trajectory.Samples,
		throw.Trajectory.Samples[0],
	)
	sortUtilityTrajectoryForTest(&throw.Trajectory)
	if utilityTrajectorySorted(throw.Trajectory, throw.Lifecycle) {
		t.Fatal("duplicate trajectory sample was accepted")
	}

	throw = validUtilityQualityThrow()
	throw.Trajectory.Bounces = append(
		throw.Trajectory.Bounces,
		throw.Trajectory.Bounces[0],
	)
	sortUtilityTrajectoryForTest(&throw.Trajectory)
	if utilityTrajectorySorted(throw.Trajectory, throw.Lifecycle) {
		t.Fatal("duplicate bounce observation was accepted")
	}
}

func sortUtilityTrajectoryForTest(trajectory *utility.Trajectory) {
	sort.Slice(trajectory.Samples, func(i, j int) bool {
		left, right := trajectory.Samples[i], trajectory.Samples[j]
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		if compared := compareUtilityVectors(left.Position, right.Position); compared != 0 {
			return compared < 0
		}
		return left.Source < right.Source
	})
	sort.Slice(trajectory.Bounces, func(i, j int) bool {
		left, right := trajectory.Bounces[i], trajectory.Bounces[j]
		if left.Tick != right.Tick {
			return left.Tick < right.Tick
		}
		if left.Number != right.Number {
			return left.Number < right.Number
		}
		if left.PositionStatus != right.PositionStatus {
			return left.PositionStatus < right.PositionStatus
		}
		if compared := compareUtilityVectors(left.Position, right.Position); compared != 0 {
			return compared < 0
		}
		return left.Source < right.Source
	})
}

func TestUtilityQualityRejectsSemanticContradictions(t *testing.T) {
	t.Run("flash relation", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Flashes[0].Victim.Side = throw.Actor.Side
		if utilityFlashAttributionMismatches(throw) == 0 {
			t.Fatal("enemy relation with a teammate was accepted")
		}
	})

	t.Run("flash correlation", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Flashes[0].Correlation = utility.Correlation{Status: utility.CorrelationObserved, Source: utility.SourceEffectEntityID}
		if utilityFlashAttributionMismatches(throw) == 0 {
			t.Fatal("invalid flash-to-throw correlation was accepted")
		}
	})

	t.Run("instant active effect", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Lifecycle.Status = utility.LifecycleEffectActive
		throw.Lifecycle.EffectStart = observedUtilityMoment(15, utility.SourceFlashExplode)
		if utilityLifecycleViolations(throw) == 0 {
			t.Fatal("flashbang effect_active lifecycle was accepted")
		}
	})

	t.Run("bounce arithmetic", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Trajectory.BounceCount = 2
		if utilityThrowContractViolations(throw) == 0 {
			t.Fatal("bounce_count different from len(bounces) was accepted")
		}
	})

	t.Run("non-positive bounce number", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Trajectory.Bounces[0].Number = 0
		if utilityThrowContractViolations(throw) == 0 {
			t.Fatal("non-positive bounce number was accepted")
		}
	})

	t.Run("destroy sample after detonation", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Trajectory.Samples = append(throw.Trajectory.Samples, utility.TrajectorySample{
			Tick: 15, Position: utility.Vector{X: 15}, Source: utility.SourceProjectileDestroy,
		})
		if utilityThrowContractViolations(throw) == 0 {
			t.Fatal("destroy sample at an earlier detonation terminal was accepted")
		}
	})

	t.Run("partial trajectory despite observed flight terminal", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Trajectory.Status = utility.TrajectoryPartial
		if utilityThrowContractViolations(throw) == 0 {
			t.Fatal("partial trajectory with frame coverage and detonation terminal was accepted")
		}
	})

	t.Run("observed trajectory without causal terminal", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Lifecycle.Detonation = unavailableUtilityMoment()
		throw.Lifecycle.Destroy = unavailableUtilityMoment()
		if utilityThrowContractViolations(throw) == 0 {
			t.Fatal("observed trajectory without detonation, effect_start, or destroy was accepted")
		}
	})

	t.Run("unavailable trajectory loses exact destroy", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Lifecycle.Detonation = unavailableUtilityMoment()
		throw.Trajectory = unavailableUtilityTrajectory()
		if utilityThrowContractViolations(throw) == 0 {
			t.Fatal("unavailable trajectory discarded an exact causal destroy observation")
		}
	})

	t.Run("corrupt view vector", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Launch.View.Vector = utility.Vector{X: 2}
		if utilityThrowContractViolations(throw) == 0 {
			t.Fatal("non-unit view vector was accepted")
		}
	})

	t.Run("trajectory after terminal", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Trajectory.Samples[1].Tick = 15
		if utilityTemporalSpatialViolations(throw, &models.ReplayRound{StartTick: 1, EndTick: 100}) == 0 {
			t.Fatal("trajectory sample at the detonation cutoff was accepted")
		}
		if utilityTrajectorySorted(throw.Trajectory, throw.Lifecycle) {
			t.Fatal("determinism gate accepted a trajectory sample at the detonation cutoff")
		}
	})

	t.Run("bounce at terminal", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Trajectory.Bounces[0].Tick = 15
		if validUtilityTrajectory(throw.Trajectory, throw.Lifecycle) {
			t.Fatal("bounce at the detonation cutoff was accepted")
		}
		if utilityTrajectorySorted(throw.Trajectory, throw.Lifecycle) {
			t.Fatal("determinism gate accepted a bounce at the detonation cutoff")
		}
	})

	t.Run("invalid extinguish attribution", func(t *testing.T) {
		throw := validUtilityQualityThrow()
		throw.Type = utility.TypeMolotov
		throw.Lifecycle.Status = utility.LifecycleEffectExpired
		throw.Lifecycle.Detonation = unavailableUtilityMoment()
		throw.Lifecycle.EffectStart = observedUtilityMoment(15, utility.SourceInfernoStart)
		throw.Lifecycle.Expiration = observedUtilityMoment(20, utility.SourceInfernoExpired)
		throw.Lifecycle.EndReason = utility.EndReasonSmokeExtinguished
		throw.Lifecycle.EndReasonSource = utility.SourceSpatialSmokeOverlap
		if utilityLifecycleViolations(throw) == 0 {
			t.Fatal("smoke_extinguished without smoke throw link was accepted")
		}
	})
}

func TestUtilityQualityAcceptsDestroyOnlyPartialTrajectory(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Lifecycle.Detonation = unavailableUtilityMoment()
	throw.Trajectory = utility.Trajectory{
		Samples: []utility.TrajectorySample{{Tick: 15, Position: utility.Vector{X: 15}, Source: utility.SourceProjectileDestroy}},
		Bounces: []utility.BounceObservation{}, Status: utility.TrajectoryPartial, Source: utility.SourceProjectileDestroy,
		BounceStatus: utility.AvailabilityUnavailable, BounceSource: utility.SourceUnavailable,
	}
	if violations := utilityThrowContractViolations(throw); violations != 0 {
		t.Fatalf("destroy-only partial trajectory was rejected: %d", violations)
	}
}

func TestUtilityQualityUsesEarliestFlightTerminal(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Lifecycle.EffectStart = observedUtilityMoment(13, utility.SourceSmokeStart)
	if terminal := utilityTrajectoryTerminalTick(throw.Lifecycle); terminal != 13 {
		t.Fatalf("trajectory terminal = %d, want earliest detonation/effect_start tick 13", terminal)
	}
	if validUtilityTrajectory(throw.Trajectory, throw.Lifecycle) {
		t.Fatal("trajectory sample after the earliest lifecycle terminal was accepted")
	}
}

func TestUtilityQualityAcceptsObservedTrajectoryEndingAtDestroy(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Lifecycle.Detonation = unavailableUtilityMoment()
	throw.Trajectory.Samples = append(throw.Trajectory.Samples, utility.TrajectorySample{
		Tick: 15, Position: utility.Vector{X: 15}, Source: utility.SourceProjectileDestroy,
	})
	if !validUtilityTrajectory(throw.Trajectory, throw.Lifecycle) {
		t.Fatal("observed frame trajectory with a causal destroy terminal was rejected")
	}
}

func TestUtilityQualityTreatsDestroyWithoutPositionAsPartialCoverage(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Lifecycle.Detonation = unavailableUtilityMoment()
	throw.Lifecycle.Destroy.Position = utility.Vector{}
	throw.Lifecycle.Destroy.PositionStatus = utility.AvailabilityUnavailable
	throw.Trajectory.Status = utility.TrajectoryPartial
	if !validUtilityTrajectory(throw.Trajectory, throw.Lifecycle) {
		t.Fatal("frame trajectory with an observed destroy tick but unavailable position was rejected")
	}
}

func TestLegacyUtilityDetailsDoNotTreatProjectileDestroyAsDetonation(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Lifecycle.Status = utility.LifecycleDestroyedWithoutDetonation
	throw.Lifecycle.Detonation = unavailableUtilityMoment()
	throw.Lifecycle.EffectStart = unavailableUtilityMoment()
	throw.Lifecycle.Destroy = observedUtilityMoment(21, utility.SourceProjectileDestroy)

	details := projectLegacyUtilityDetails(throw)
	if details.TickExplode != 0 || details.EndPosition != (models.AI_Vector{}) {
		t.Fatalf("projectile destroy was fabricated as an explosion: %+v", details)
	}
}

func TestUtilityQualityTreatsSparseLifecycleAsCoverage(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Type = utility.TypeUnknown
	throw.TypeSource = utility.SourceUnavailable
	throw.EntityStatus = utility.AvailabilityUnavailable
	throw.EntitySource = utility.SourceUnavailable
	throw.SourceEntityID, throw.SourceEntityGeneration = 0, 0
	throw.Actor.Source = utility.SourceCallbackActor
	throw.Launch = unavailableUtilityLaunch()
	throw.Trajectory = unavailableUtilityTrajectory()
	throw.Flashes = nil
	throw.Lifecycle = unavailableUtilityLifecycle()
	throw.Lifecycle.Status = utility.LifecycleEffectExpired
	throw.Lifecycle.Expiration = observedUtilityMoment(30, utility.SourceInfernoExpired)
	throw.Lifecycle.EndReason = utility.EndReasonExpired
	throw.Lifecycle.EndReasonSource = utility.SourceExpirationCallback

	stats := utilityQualityStats{}
	stats.assessThrow(throw, &models.ReplayRound{StartTick: 1, EndTick: 100})
	if stats.hasHardFailure() {
		t.Fatalf("explicit sparse observations became a hard failure: %+v", stats)
	}
	if stats.observationWarnings == 0 || stats.checks()[7].Status != "warning" {
		t.Fatalf("sparse lifecycle was not reported as coverage: %+v", stats)
	}
}

func TestUtilityQualityCountsIndependentInferredCorrelations(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Lifecycle.Correlation = utility.Correlation{
		Status: utility.CorrelationInferred,
		Source: utility.SourceThrowerTypePositionTick,
	}
	throw.Lifecycle.ExtinguishAttribution = utility.Correlation{
		Status: utility.CorrelationInferred,
		Source: utility.SourceSpatialSmokeOverlap,
	}

	stats := utilityQualityStats{}
	stats.countObservationWarnings(throw)

	if stats.inferredCorrelations != 2 {
		t.Fatalf("independent inferred correlations were collapsed: got %d, want 2", stats.inferredCorrelations)
	}
}

func TestCanonicalUtilityKeepsAmbiguousOrphanTypeUnavailable(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Type = utility.TypeUnknown
	throw.TypeSource = utility.SourceUnavailable
	event := projectCanonicalUtilityThrows("orphan-type", []utility.Throw{throw})[0]
	if event.UtilityType != string(utility.TypeUnknown) || event.UtilityTypeStatus != string(utility.AvailabilityUnavailable) ||
		event.UtilityTypeSource != utility.SourceUnavailable {
		t.Fatalf("ambiguous orphan type was fabricated: %+v", event)
	}
}

func TestUtilityContractRejectsEffectSourceAsThrowerProvenance(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Actor.Source = utility.SourcePlayerHurt
	if violations := utilityThrowContractViolations(throw); violations == 0 {
		t.Fatal("effect provenance was accepted as thrower provenance")
	}
	throw.Actor.Source = utility.SourceCallbackActor
	if violations := utilityThrowContractViolations(throw); violations != 0 {
		t.Fatalf("callback actor provenance was rejected: %d", violations)
	}
}

func TestUtilityDamageSummaryDoesNotCountZeroDamageAsDamagedPlayer(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Type = utility.TypeHE
	throw.Flashes = nil
	throw.Lifecycle.Detonation.Source = utility.SourceHEExplode
	throw.Damage = []utility.DamageEffect{{
		Victim:   utility.PlayerRef{ID: 2, Side: "CT", Status: utility.AvailabilityObserved, Source: utility.SourcePlayerHurt},
		Relation: utility.RelationEnemy, Tick: 15, Source: utility.SourcePlayerHurt,
		Correlation: utility.Correlation{Status: utility.CorrelationInferred, Source: utility.SourceThrowerTypePositionTick},
	}}
	if mismatches := utilityDamageMismatches(throw); mismatches != 0 {
		t.Fatalf("zero-valued observed damage callback was rejected: %d", mismatches)
	}
	event := projectCanonicalUtilityThrows("zero-damage", []utility.Throw{throw})[0]
	if event.DamageSummary.TotalDamage != 0 || event.DamageSummary.TotalArmorDamage != 0 ||
		event.DamageSummary.EnemiesDamaged != 0 || len(event.AffectedPlayers[0].DamageEvents) != 1 {
		t.Fatalf("zero damage callback was lost or inflated summary counts: %+v", event)
	}
}

func TestCanonicalFlashSummaryKeepsIncompleteDurationNull(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Flashes[0].Duration = utility.ScalarObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable}
	if mismatches := utilityFlashAttributionMismatches(throw); mismatches != 0 {
		t.Fatalf("explicit unavailable flash duration was rejected: %d", mismatches)
	}
	event := projectCanonicalUtilityThrows("flash-coverage", []utility.Throw{throw})[0]
	if event.FlashSummary.TotalDurationMS != nil || event.FlashSummary.EnemyDurationMS != nil ||
		event.FlashSummary.TeammateDurationMS == nil || *event.FlashSummary.TeammateDurationMS != 0 {
		t.Fatalf("incomplete duration was replaced by a fabricated aggregate: %+v", event.FlashSummary)
	}
	stats := utilityQualityStats{}
	stats.countObservationWarnings(throw)
	if stats.missingFlashDurationObservations != 1 || stats.observationWarnings == 0 {
		t.Fatalf("missing flash duration was not reported as coverage: %+v", stats)
	}
}

func TestCanonicalDamageSummaryPreservesFriendlyKillBreakdown(t *testing.T) {
	throw := validUtilityQualityThrow()
	throw.Type = utility.TypeMolotov
	throw.Flashes = nil
	throw.Damage = []utility.DamageEffect{
		{
			Victim:   utility.PlayerRef{ID: 3, Side: "T", Status: utility.AvailabilityObserved, Source: utility.SourcePlayerHurt},
			Relation: utility.RelationTeammate, HealthDamage: 20, Kill: true, Tick: 15, Source: utility.SourcePlayerHurt,
			Correlation: utility.Correlation{Status: utility.CorrelationInferred, Source: utility.SourceThrowerTypePositionTick},
		},
		{
			Victim: throw.Actor, Relation: utility.RelationSelf, HealthDamage: 0, Kill: true, Tick: 15, Source: utility.SourcePlayerHurt,
			Correlation: utility.Correlation{Status: utility.CorrelationInferred, Source: utility.SourceThrowerTypePositionTick},
		},
	}
	throw.Damage[1].Victim.Source = utility.SourcePlayerHurt
	event := projectCanonicalUtilityThrows("friendly-kills", []utility.Throw{throw})[0]
	if event.DamageSummary.TeammateKills != 1 || event.DamageSummary.SelfKills != 1 ||
		event.DamageSummary.SelfDamaged || event.DamageSummary.TeammatesDamaged != 1 {
		t.Fatalf("friendly kill breakdown is inconsistent: %+v", event.DamageSummary)
	}
	if mismatches := utilityDamageMismatches(throw); mismatches != 0 {
		t.Fatalf("friendly kill breakdown did not reconcile: %d", mismatches)
	}
}

func TestUtilityPlayerStatsReconcileWithLedger(t *testing.T) {
	flash := validUtilityQualityThrow()
	teammateFlash := validUtilityFlashEffect(3, utility.RelationTeammate, 0.25)
	teammateFlash.Victim.Side = "T"
	selfFlash := validUtilityFlashEffect(1, utility.RelationSelf, 0.1)
	selfFlash.Victim.Side = "T"
	flash.Flashes = append(flash.Flashes, teammateFlash, selfFlash)
	types := []utility.Type{
		utility.TypeFlashbang, utility.TypeHE, utility.TypeSmoke, utility.TypeMolotov,
		utility.TypeIncendiary, utility.TypeDecoy, utility.TypeUnknown,
	}
	throws := make([]utility.Throw, 0, len(types))
	for index, typeName := range types {
		entry := flash
		entry.ID = "r1-u000" + strconv.Itoa(index+1)
		entry.Sequence = index + 1
		entry.Type = typeName
		if index > 0 {
			entry.Flashes = nil
		}
		switch typeName {
		case utility.TypeHE:
			entry.Damage = []utility.DamageEffect{{Relation: utility.RelationEnemy, HealthDamage: 15}}
		case utility.TypeMolotov:
			entry.Damage = []utility.DamageEffect{{Relation: utility.RelationEnemy, HealthDamage: 7}}
		case utility.TypeIncendiary:
			entry.Damage = []utility.DamageEffect{{Relation: utility.RelationEnemy, HealthDamage: 8}}
		case utility.TypeUnknown:
			entry.Damage = []utility.DamageEffect{{Relation: utility.RelationEnemy, HealthDamage: 4}}
		}
		throws = append(throws, entry)
	}
	player := models.AI_PlayerStats{
		SteamID: "1", GrenadesThrownTotal: 7, FlashesThrown: 1, HEThrown: 1, SmokesThrown: 1,
		MolotovsThrown: 2, MolotovGrenadesThrown: 1, IncendiariesThrown: 1, DecoysThrown: 1,
		EnemiesFlashedTotal: 1, TeammatesFlashedTotal: 1, SelfFlashesTotal: 1,
		EnemyFlashDurationTotalMS: 750, TeammateFlashDurationTotalMS: 250, SelfFlashDurationTotalMS: 100,
		FlashDurationTotal: 0.75, EnemiesFlashedPerFlash: 1, BlindTimePerFlash: 0.75,
		UtilityDamage: 99, NativeScoreboard: models.AI_NativePlayerStats{UtilityDamage: 99},
		UtilityDamageObserved: 34, GrenadeDamage: map[string]int{"he": 15, "molotov": 15, "unknown": 4},
		HEDamagePerNade: 15, MolotovDamagePerNade: 7.5,
	}
	stats := utilityQualityStats{}
	stats.assessPlayerStats(throws, []models.AI_PlayerStats{player})
	if stats.playerStatsMismatches != 0 || stats.hasHardFailure() {
		t.Fatalf("coherent ledger-derived stats did not reconcile: %+v", stats)
	}
	player.DecoysThrown = 0
	stats = utilityQualityStats{}
	stats.assessPlayerStats(throws, []models.AI_PlayerStats{player})
	if stats.playerStatsMismatches != 1 || !stats.hasHardFailure() {
		t.Fatalf("player stats drift was not a hard failure: %+v", stats)
	}
	player.DecoysThrown = 1
	player.UtilityDamageObserved++
	stats = utilityQualityStats{}
	stats.assessPlayerStats(throws, []models.AI_PlayerStats{player})
	if stats.playerStatsMismatches != 1 || !stats.hasHardFailure() {
		t.Fatalf("ledger utility damage drift was not a hard failure: %+v", stats)
	}
	player.UtilityDamageObserved--
	player.GrenadeDamage["molotov"]++
	stats = utilityQualityStats{}
	stats.assessPlayerStats(throws, []models.AI_PlayerStats{player})
	if stats.playerStatsMismatches != 1 || !stats.hasHardFailure() {
		t.Fatalf("ledger utility damage breakdown drift was not a hard failure: %+v", stats)
	}
}

func TestUtilityThrowGateUsesRawCallbackDiagnostics(t *testing.T) {
	diagnostics := utility.Diagnostics{Throws: utility.CallbackDiagnostics{Observed: 2, ExactCorrelated: 1, Deduplicated: 1}}
	throw := validUtilityQualityThrow()
	stats := utilityQualityStats{}
	stats.assessDiagnostics(diagnostics)
	stats.assessDiagnosticThrowReconciliation([]utility.Throw{throw})
	if stats.callbackAccountingViolations != 0 || stats.throwReconciliationMismatches != 0 {
		t.Fatalf("coherent callback accounting did not reconcile: %+v", stats)
	}
	throw.Launch.Tick = utility.TickObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable}
	stats = utilityQualityStats{}
	stats.assessDiagnostics(diagnostics)
	stats.assessDiagnosticThrowReconciliation([]utility.Throw{throw})
	if stats.throwReconciliationMismatches == 0 || !stats.hasHardFailure() {
		t.Fatalf("missing launch row was not detected from raw callbacks: %+v", stats)
	}
	report := stats.callbackDiagnosticsReport()
	if report["throws"].Observed != 2 || report["throws"].Deduplicated != 1 {
		t.Fatalf("raw callback diagnostics were not preserved: %+v", report)
	}
}

func TestUtilityReplayProjectionGateRejectsMissingAndDriftedMarkers(t *testing.T) {
	throw := validUtilityQualityThrow()
	moment, ok := utilityReplayEffectMoment(throw.Lifecycle)
	if !ok {
		t.Fatal("coherent throw has no replay effect moment")
	}
	marker := expectedReplayUtilityMarker(throw, moment)
	replay := &models.ReplayData{Rounds: []models.ReplayRound{{
		Round: throw.Round, Events: []models.ReplayEvent{marker},
	}}}

	stats := utilityQualityStats{}
	stats.assessReplayProjection([]utility.Throw{throw}, replay)
	if stats.replayProjectionMismatches != 0 || stats.hasHardFailure() {
		t.Fatalf("coherent replay marker did not reconcile: %+v", stats)
	}

	replay.Rounds[0].Events[0].Damage++
	stats = utilityQualityStats{}
	stats.assessReplayProjection([]utility.Throw{throw}, replay)
	if stats.replayProjectionMismatches != 1 || !stats.hasHardFailure() {
		t.Fatalf("replay marker drift was not a hard failure: %+v", stats)
	}

	stats = utilityQualityStats{}
	stats.assessReplayProjection([]utility.Throw{throw}, &models.ReplayData{})
	if stats.replayProjectionMismatches != 1 || !stats.hasHardFailure() {
		t.Fatalf("missing replay marker was not a hard failure: %+v", stats)
	}
}

func validUtilityQualityThrow() utility.Throw {
	return utility.Throw{
		ID: "r1-u0001", Round: 1, Sequence: 1,
		SourceEntityID: 42, SourceEntityGeneration: 1,
		EntityStatus: utility.AvailabilityObserved, EntitySource: utility.SourceProjectileEntity,
		Type: utility.TypeFlashbang, TypeSource: utility.SourceWeaponInstance,
		Actor: utility.PlayerRef{ID: 1, Name: "thrower", Side: "T", Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrower},
		Launch: utility.ThrowSnapshot{
			Tick:                      utility.TickObservation{Tick: 10, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow},
			Position:                  utility.VectorObservation{Value: utility.Vector{X: 1, Y: 2, Z: 3}, Status: utility.AvailabilityObserved, Source: utility.SourceProjectilePosition},
			View:                      utility.ViewObservation{Yaw: 5, Pitch: -2, Vector: utility.Vector{X: 1}, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerView},
			ThrowerVelocity:           utility.VelocityObservation{Vector: utility.Vector{X: 10}, HorizontalSpeed: 10, ObservedTick: 10, MeasurementWindowTicks: 1, Status: utility.AvailabilityObserved, Source: utility.SourceVelocityPositionDelta},
			ProjectileInitialVelocity: utility.VelocityObservation{Vector: utility.Vector{X: 100}, HorizontalSpeed: 100, ObservedTick: 10, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity},
			Stance:                    utility.StanceObservation{Value: utility.StanceWalking, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerState},
			Area:                      utility.StringObservation{Value: "A Ramp", Status: utility.AvailabilityObserved, Source: utility.SourcePlayerLastPlace},
		},
		Trajectory: utility.Trajectory{
			Samples: []utility.TrajectorySample{
				{Tick: 10, Position: utility.Vector{X: 1}, Source: utility.SourceProjectileFrames},
				{Tick: 14, Position: utility.Vector{X: 5}, Source: utility.SourceProjectileFrames},
			},
			Bounces: []utility.BounceObservation{{Tick: 12, Position: utility.Vector{X: 3}, PositionStatus: utility.AvailabilityObserved, Number: 1, Source: utility.SourceProjectileBounce}},
			Status:  utility.TrajectoryObserved, Source: utility.SourceProjectileFrames,
			BounceCount: 1, BounceStatus: utility.AvailabilityObserved, BounceSource: utility.SourceProjectileBounce,
		},
		Lifecycle: utility.Lifecycle{
			Status: utility.LifecycleDetonated, Detonation: observedUtilityMoment(15, utility.SourceFlashExplode),
			EffectStart: unavailableUtilityMoment(), Expiration: unavailableUtilityMoment(), Destroy: observedUtilityMoment(15, utility.SourceProjectileDestroy), Extinguish: unavailableUtilityMoment(),
			Duration:  utility.ScalarObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
			Area:      utility.StringObservation{Value: "A Site", Status: utility.AvailabilityObserved, Source: utility.SourceMapCallout},
			EndReason: utility.EndReasonUnavailable, EndReasonSource: utility.SourceUnavailable,
			Correlation:           utility.Correlation{Status: utility.CorrelationObserved, Source: utility.SourceProjectileEntity},
			ExtinguishedByThrowID: utility.StringObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
			ExtinguishAttribution: utility.Correlation{Status: utility.CorrelationUnavailable, Source: utility.SourceUnavailable},
		},
		Flashes: []utility.FlashEffect{validUtilityFlashEffect(2, utility.RelationEnemy, 0.75)},
		Damage:  []utility.DamageEffect{},
	}
}

func validUtilityFlashEffect(playerID uint64, relation utility.Relation, duration float64) utility.FlashEffect {
	return utility.FlashEffect{
		Victim:   utility.PlayerRef{ID: playerID, Name: "victim", Side: "CT", Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed},
		Relation: relation, Duration: utility.ScalarObservation{Value: duration, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed},
		Tick: 15, Source: utility.SourcePlayerFlashed,
		Correlation: utility.Correlation{Status: utility.CorrelationObserved, Source: utility.SourceProjectileEntity},
	}
}

func observedUtilityMoment(tick int, source string) utility.TickPositionObservation {
	return utility.TickPositionObservation{
		Tick: tick, Position: utility.Vector{X: float64(tick)}, Status: utility.AvailabilityObserved,
		PositionStatus: utility.AvailabilityObserved, Source: source,
	}
}

func unavailableUtilityMoment() utility.TickPositionObservation {
	return utility.TickPositionObservation{Status: utility.AvailabilityUnavailable, PositionStatus: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable}
}

func unavailableUtilityLaunch() utility.ThrowSnapshot {
	return utility.ThrowSnapshot{
		Tick:                      utility.TickObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		Position:                  utility.VectorObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		View:                      utility.ViewObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		ThrowerVelocity:           utility.VelocityObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		ProjectileInitialVelocity: utility.VelocityObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		Stance:                    utility.StanceObservation{Value: utility.StanceUnknown, Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		Area:                      utility.StringObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
	}
}

func unavailableUtilityTrajectory() utility.Trajectory {
	return utility.Trajectory{
		Samples: []utility.TrajectorySample{}, Bounces: []utility.BounceObservation{},
		Status: utility.TrajectoryUnavailable, Source: utility.SourceUnavailable,
		BounceStatus: utility.AvailabilityUnavailable, BounceSource: utility.SourceUnavailable,
	}
}

func unavailableUtilityLifecycle() utility.Lifecycle {
	return utility.Lifecycle{
		Status:     utility.LifecycleThrown,
		Detonation: unavailableUtilityMoment(), EffectStart: unavailableUtilityMoment(), Expiration: unavailableUtilityMoment(), Destroy: unavailableUtilityMoment(), Extinguish: unavailableUtilityMoment(),
		Duration:  utility.ScalarObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		Area:      utility.StringObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		EndReason: utility.EndReasonUnavailable, EndReasonSource: utility.SourceUnavailable,
		Correlation:           utility.Correlation{Status: utility.CorrelationUnavailable, Source: utility.SourceUnavailable},
		ExtinguishedByThrowID: utility.StringObservation{Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable},
		ExtinguishAttribution: utility.Correlation{Status: utility.CorrelationUnavailable, Source: utility.SourceUnavailable},
	}
}
