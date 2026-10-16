package parser

import (
	"bytes"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
)

func TestObservedUtilityInitialVelocityLedgerPublishesNativeAndCausalDelta(t *testing.T) {
	native := observedUtilityVelocityTestThrow(1, 1, 10, 7, utility.TypeSmoke, 100)
	native.InitialVelocityCandidate = observedUtilityVelocityCandidate(native, utility.InitialVelocityCandidate{
		Vector: utility.Vector{X: 0, Y: 700, Z: 180}, HorizontalSpeed: 700,
		ObservedTick: 100, AvailabilityTick: 100,
		MeasurementStartTick: 100, MeasurementEndTick: 100,
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity,
	})
	delta := observedUtilityVelocityTestThrow(1, 2, 11, 8, utility.TypeFlashbang, 110)
	delta.InitialVelocityCandidate = observedUtilityVelocityCandidate(delta, utility.InitialVelocityCandidate{
		Vector: utility.Vector{X: 64, Y: 0, Z: 32}, HorizontalSpeed: 64,
		ObservedTick: 112, AvailabilityTick: 112,
		MeasurementStartTick: 110, MeasurementEndTick: 112, MeasurementWindowTicks: 2,
		TickRate:      64,
		StartPosition: utility.Vector{X: 10, Y: 20, Z: 30},
		EndPosition:   utility.Vector{X: 12, Y: 20, Z: 31},
		Status:        utility.AvailabilityObserved, Source: utility.SourceProjectilePositionDelta,
	})

	ledger, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{delta, native})
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Rows) != 2 {
		t.Fatalf("rows = %d", len(ledger.Rows))
	}
	first, second := ledger.Rows[0], ledger.Rows[1]
	if first.InitialVelocity.Status != models.ObservedUtilityInitialVelocityStatusObserved ||
		first.InitialVelocity.Source != models.ObservedUtilityInitialVelocitySourceNative ||
		first.InitialVelocity.Vector == nil || first.InitialVelocity.Vector.X != 0 ||
		first.InitialVelocity.AvailabilityTick == nil || *first.InitialVelocity.AvailabilityTick != 100 ||
		first.InitialVelocity.MeasurementWindow.TickCount == nil || *first.InitialVelocity.MeasurementWindow.TickCount != 0 {
		t.Fatalf("unexpected native row: %+v", first)
	}
	if second.InitialVelocity.Source != models.ObservedUtilityInitialVelocitySourcePositionDelta ||
		second.InitialVelocity.AvailabilityTick == nil || *second.InitialVelocity.AvailabilityTick != 112 ||
		second.InitialVelocity.MeasurementWindow.TickRate == nil || *second.InitialVelocity.MeasurementWindow.TickRate != 64 ||
		second.InitialVelocity.Vector == nil || second.InitialVelocity.Vector.Y != 0 ||
		second.InitialVelocity.Vector.X != 64 || second.InitialVelocity.Vector.Z != 32 {
		t.Fatalf("unexpected delta row: %+v", second)
	}
	for _, row := range ledger.Rows {
		if row.JoinKeys.PlayerID == nil || strings.Contains(*row.JoinKeys.PlayerID, "7") ||
			strings.Contains(row.JoinKeys.ThrowID, "r1") || strings.Contains(row.JoinKeys.ThrowID, "10") {
			t.Fatalf("raw identity leaked outside opaque joins: %+v", row.JoinKeys)
		}
	}
}

func TestObservedUtilityInitialVelocityLedgerAbstainsOnMissingAndAmbiguousIdentity(t *testing.T) {
	first := observedUtilityVelocityTestThrow(2, 1, 20, 9, utility.TypeHE, 200)
	second := observedUtilityVelocityTestThrow(2, 2, 20, 10, utility.TypeHE, 201)
	first.SourceEntityGeneration = 1
	second.SourceEntityGeneration = 1
	first.ID = "duplicate"
	second.ID = "duplicate"
	first.InitialVelocityCandidate = utility.InitialVelocityCandidate{
		Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
		Reason: utility.InitialVelocityReasonIdentityConflict, Terminal: true,
	}
	second.InitialVelocityCandidate = utility.InitialVelocityCandidate{
		Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
		Reason: utility.InitialVelocityReasonEntityReused, Terminal: true,
	}
	ledger, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{second, first})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range ledger.Rows {
		ambiguousCauseCount := 0
		for _, reason := range []string{
			utility.InitialVelocityReasonIdentityConflict,
			utility.InitialVelocityReasonEntityReused,
		} {
			if slices.Contains(row.AbstentionReasons, reason) {
				ambiguousCauseCount++
			}
		}
		if row.InitialVelocity.Status != models.ObservedUtilityInitialVelocityStatusUnavailable ||
			row.InitialVelocity.Vector != nil ||
			ambiguousCauseCount != 1 ||
			!slices.Contains(row.AbstentionReasons, utilityVelocityReasonVelocityUnavailable) ||
			row.JoinKeys.ProjectileID == nil ||
			row.InitialVelocity.Provenance.Reconciliation != models.ObservedUtilityInitialVelocityReconciliationAmbiguous {
			t.Fatalf("ambiguous identity did not abstain: %+v", row)
		}
	}
}

func TestObservedUtilityInitialVelocityIgnoresFutureAndIsDeterministic(t *testing.T) {
	base := observedUtilityVelocityTestThrow(3, 1, 30, 11, utility.TypeMolotov, 300)
	base.InitialVelocityCandidate = observedUtilityVelocityCandidate(base, utility.InitialVelocityCandidate{
		Vector: utility.Vector{X: 10, Y: 20, Z: 30}, HorizontalSpeed: math.Hypot(10, 20),
		ObservedTick: 300, AvailabilityTick: 300,
		MeasurementStartTick: 300, MeasurementEndTick: 300,
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity,
	})
	mutated := base
	mutated.Trajectory = utility.Trajectory{
		Samples: []utility.TrajectorySample{{Tick: 301, Position: utility.Vector{X: 999999}}, {Tick: 900, Position: utility.Vector{Z: -999999}}},
		Status:  utility.TrajectoryObserved, Source: utility.SourceProjectileFrames,
	}
	mutated.Lifecycle = utility.Lifecycle{
		Status:     utility.LifecycleEffectExpired,
		Detonation: utility.TickPositionObservation{Tick: 320, Status: utility.AvailabilityObserved, Source: utility.SourceHEExplode},
	}
	mutated.Flashes = []utility.FlashEffect{{Tick: 400}}
	mutated.Damage = []utility.DamageEffect{{Tick: 401, Kill: true, HealthDamage: 100}}

	before, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{base})
	if err != nil {
		t.Fatal(err)
	}
	after, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{mutated})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("future/oracle fields changed candidate:\nbefore=%+v\nafter=%+v", before, after)
	}
	futureConflict := base
	futureConflict.ID = "r3-u2"
	futureConflict.Sequence = 2
	futureConflict.Launch.Tick.Tick = 350
	futureConflict.InitialVelocityCandidate = utility.InitialVelocityCandidate{
		Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
		Reason: utility.InitialVelocityReasonIdentityConflict, Terminal: true,
	}
	withFutureConflict, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{futureConflict, base})
	if err != nil {
		t.Fatal(err)
	}
	if len(withFutureConflict.Rows) != 2 || !reflect.DeepEqual(before.Rows[0], withFutureConflict.Rows[0]) {
		t.Fatalf("future physical-identity conflict changed prior row:\nbefore=%+v\nafter=%+v", before.Rows[0], withFutureConflict.Rows)
	}
	futureLowerActorID := observedUtilityVelocityTestThrow(3, 2, 31, 1, utility.TypeSmoke, 360)
	futureLowerActorID.InitialVelocityCandidate = observedUtilityVelocityCandidate(futureLowerActorID, utility.InitialVelocityCandidate{
		Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
		Reason: utility.InitialVelocityReasonNativeUnavailable,
	})
	withFutureLowerActorID, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{futureLowerActorID, base})
	if err != nil {
		t.Fatal(err)
	}
	if len(withFutureLowerActorID.Rows) != 2 || !reflect.DeepEqual(before.Rows[0], withFutureLowerActorID.Rows[0]) {
		t.Fatalf("future participant changed prior alias/row:\nbefore=%+v\nafter=%+v", before.Rows[0], withFutureLowerActorID.Rows)
	}

	other := observedUtilityVelocityTestThrow(1, 1, 4, 2, utility.TypeDecoy, 50)
	other.InitialVelocityCandidate = observedUtilityVelocityCandidate(other, utility.InitialVelocityCandidate{
		Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
		Reason: utility.InitialVelocityReasonNativeUnavailable,
	})
	left, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{base, other})
	if err != nil {
		t.Fatal(err)
	}
	right, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{other, base})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("equivalent input order changed output:\nleft=%+v\nright=%+v", left, right)
	}

	entityLowFirst := observedUtilityVelocityTestThrow(6, 1, 61, 31, utility.TypeSmoke, 600)
	entityHighSecond := observedUtilityVelocityTestThrow(6, 2, 62, 32, utility.TypeHE, 600)
	entityLowFirst.InitialVelocityCandidate = observedUtilityVelocityCandidate(entityLowFirst, utility.InitialVelocityCandidate{
		Vector: utility.Vector{X: 10, Y: 20, Z: 30}, HorizontalSpeed: math.Hypot(10, 20),
		ObservedTick: 600, AvailabilityTick: 600,
		MeasurementStartTick: 600, MeasurementEndTick: 600,
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity,
	})
	entityHighSecond.InitialVelocityCandidate = observedUtilityVelocityCandidate(entityHighSecond, utility.InitialVelocityCandidate{
		Vector: utility.Vector{X: 40, Y: 50, Z: 60}, HorizontalSpeed: math.Hypot(40, 50),
		ObservedTick: 600, AvailabilityTick: 600,
		MeasurementStartTick: 600, MeasurementEndTick: 600,
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity,
	})
	entityLowSecond := entityLowFirst
	entityLowSecond.Sequence = 2
	entityLowSecond.ID = "r6-u2"
	entityHighFirst := entityHighSecond
	entityHighFirst.Sequence = 1
	entityHighFirst.ID = "r6-u1"
	callbackOrderLeft, err := BuildObservedUtilityInitialVelocityLedger(
		[]utility.Throw{entityLowFirst, entityHighSecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	callbackOrderRight, err := BuildObservedUtilityInitialVelocityLedger(
		[]utility.Throw{entityHighFirst, entityLowSecond},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(callbackOrderLeft, callbackOrderRight) {
		t.Fatalf("simultaneous callback order changed aliases/output:\nleft=%+v\nright=%+v", callbackOrderLeft, callbackOrderRight)
	}
}

func TestObservedUtilityInitialVelocityRejectsBadMeasurementsAndNominalInference(t *testing.T) {
	for _, utilityType := range []utility.Type{
		utility.TypeFlashbang, utility.TypeSmoke, utility.TypeHE,
		utility.TypeMolotov, utility.TypeIncendiary, utility.TypeDecoy,
	} {
		throw := observedUtilityVelocityTestThrow(4, 1, 40, 12, utilityType, 400)
		throw.InitialVelocityCandidate = observedUtilityVelocityCandidate(throw, utility.InitialVelocityCandidate{
			Status: utility.AvailabilityUnavailable, Source: utility.SourceUnavailable,
			Reason: utility.InitialVelocityReasonInsufficientSamples,
		})
		ledger, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{throw})
		if err != nil {
			t.Fatal(err)
		}
		if ledger.Rows[0].InitialVelocity.Status != models.ObservedUtilityInitialVelocityStatusUnavailable {
			t.Fatalf("utility type %s fabricated a nominal velocity", utilityType)
		}
	}

	bad := observedUtilityVelocityTestThrow(5, 1, 50, 13, utility.TypeSmoke, 500)
	bad.InitialVelocityCandidate = observedUtilityVelocityCandidate(bad, utility.InitialVelocityCandidate{
		Vector: utility.Vector{X: math.NaN(), Y: 1, Z: 2}, HorizontalSpeed: 1,
		ObservedTick: 500, AvailabilityTick: 500,
		MeasurementStartTick: 500, MeasurementEndTick: 500,
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity,
	})
	ledger, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{bad})
	if err != nil {
		t.Fatal(err)
	}
	if ledger.Rows[0].InitialVelocity.Status != models.ObservedUtilityInitialVelocityStatusUnavailable {
		t.Fatal("non-finite velocity was accepted")
	}

	sentinel := float64(math.MaxFloat32)
	badEndpoint := observedUtilityVelocityTestThrow(5, 2, 51, 14, utility.TypeHE, 510)
	badEndpoint.InitialVelocityCandidate = observedUtilityVelocityCandidate(badEndpoint, utility.InitialVelocityCandidate{
		Vector:          utility.Vector{X: -sentinel, Y: -sentinel, Z: -sentinel},
		HorizontalSpeed: math.Hypot(sentinel, sentinel),
		ObservedTick:    511, AvailabilityTick: 511,
		MeasurementStartTick: 510, MeasurementEndTick: 511, MeasurementWindowTicks: 1,
		TickRate:      1,
		StartPosition: utility.Vector{X: sentinel, Y: sentinel, Z: sentinel},
		EndPosition:   utility.Vector{},
		Status:        utility.AvailabilityObserved, Source: utility.SourceProjectilePositionDelta,
	})
	ledger, err = BuildObservedUtilityInitialVelocityLedger([]utility.Throw{badEndpoint})
	if err != nil {
		t.Fatal(err)
	}
	if ledger.Rows[0].InitialVelocity.Status != models.ObservedUtilityInitialVelocityStatusUnavailable {
		t.Fatal("FLT_MAX measurement endpoint was accepted")
	}
}

func TestObservedUtilityInitialVelocityValidatorRejectsContractParityViolations(t *testing.T) {
	newLedger := func(t *testing.T) models.ObservedUtilityInitialVelocityLedger {
		t.Helper()
		throw := observedUtilityVelocityTestThrow(7, 1, 70, 41, utility.TypeSmoke, 700)
		throw.InitialVelocityCandidate = observedUtilityVelocityCandidate(throw, utility.InitialVelocityCandidate{
			Vector: utility.Vector{X: 0, Y: 700, Z: 180}, HorizontalSpeed: 700,
			ObservedTick: 700, AvailabilityTick: 700,
			MeasurementStartTick: 700, MeasurementEndTick: 700,
			Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity,
		})
		ledger, err := BuildObservedUtilityInitialVelocityLedger([]utility.Throw{throw})
		if err != nil {
			t.Fatal(err)
		}
		return ledger
	}

	tests := map[string]func(*models.ObservedUtilityInitialVelocityRow){
		"missing projectile join": func(row *models.ObservedUtilityInitialVelocityRow) {
			row.JoinKeys.ProjectileID = nil
		},
		"missing player join": func(row *models.ObservedUtilityInitialVelocityRow) {
			row.JoinKeys.PlayerID = nil
		},
		"unavailable utility type": func(row *models.ObservedUtilityInitialVelocityRow) {
			row.UtilityType = models.ObservedUtilityStringFact{Status: models.ObservedUtilityInitialVelocityStatusUnavailable, Source: models.ObservedUtilityInitialVelocitySourceUnavailable}
		},
		"whitespace utility type": func(row *models.ObservedUtilityInitialVelocityRow) {
			value := " smoke"
			row.UtilityType.Value = &value
		},
		"unavailable launch": func(row *models.ObservedUtilityInitialVelocityRow) {
			row.Launch = models.ObservedUtilityLaunchFact{Status: models.ObservedUtilityInitialVelocityStatusUnavailable, Source: models.ObservedUtilityInitialVelocitySourceUnavailable}
		},
		"wrong parser event": func(row *models.ObservedUtilityInitialVelocityRow) {
			row.InitialVelocity.Provenance.ParserEvent = "FrameDone"
		},
		"wrong entity property": func(row *models.ObservedUtilityInitialVelocityRow) {
			row.InitialVelocity.Provenance.EntityProperty = "velocity"
		},
		"sentinel vector": func(row *models.ObservedUtilityInitialVelocityRow) {
			row.InitialVelocity.Vector.X = float64(math.MaxFloat32)
			value := float64(math.MaxFloat32)
			row.InitialVelocity.HorizontalSpeed = &value
		},
		"non-finite horizontal": func(row *models.ObservedUtilityInitialVelocityRow) {
			value := math.Inf(1)
			row.InitialVelocity.HorizontalSpeed = &value
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			ledger := newLedger(t)
			mutate(&ledger.Rows[0])
			if err := ValidateObservedUtilityInitialVelocityLedger(ledger); err == nil {
				t.Fatal("validator accepted contract parity violation")
			}
		})
	}

	unavailable := newLedger(t)
	unavailable.Rows[0].InitialVelocity = unavailableObservedUtilityVelocityFact(models.ObservedUtilityInitialVelocityReconciliationExact)
	unavailable.Rows[0].AbstentionReasons = []string{utilityVelocityReasonVelocityUnavailable}
	unavailable.Rows[0].InitialVelocity.Provenance.EntityProperty = "m_vecVelocity"
	if err := ValidateObservedUtilityInitialVelocityLedger(unavailable); err == nil {
		t.Fatal("validator accepted incorrect unavailable provenance")
	}

	validUnavailable := newLedger(t)
	validUnavailable.Rows[0].InitialVelocity = unavailableObservedUtilityVelocityFact(models.ObservedUtilityInitialVelocityReconciliationExact)
	validUnavailable.Rows[0].AbstentionReasons = []string{utilityVelocityReasonVelocityUnavailable}
	if err := ValidateObservedUtilityInitialVelocityLedger(validUnavailable); err != nil {
		t.Fatalf("validator rejected valid unavailable fact: %v", err)
	}
	unknownReason := validUnavailable
	unknownReason.Rows = append([]models.ObservedUtilityInitialVelocityRow(nil), validUnavailable.Rows...)
	unknownReason.Rows[0].AbstentionReasons = []string{"player-76561198000000000"}
	if err := ValidateObservedUtilityInitialVelocityLedger(unknownReason); err == nil {
		t.Fatal("validator accepted arbitrary/private abstention reason")
	}
	duplicateReason := validUnavailable
	duplicateReason.Rows = append([]models.ObservedUtilityInitialVelocityRow(nil), validUnavailable.Rows...)
	duplicateReason.Rows[0].AbstentionReasons = []string{
		utilityVelocityReasonVelocityUnavailable,
		utilityVelocityReasonVelocityUnavailable,
	}
	if err := ValidateObservedUtilityInitialVelocityLedger(duplicateReason); err == nil {
		t.Fatal("validator accepted duplicate abstention reason")
	}
	unsortedReasons := validUnavailable
	unsortedReasons.Rows = append([]models.ObservedUtilityInitialVelocityRow(nil), validUnavailable.Rows...)
	unsortedReasons.Rows[0].AbstentionReasons = []string{
		utility.InitialVelocityReasonNativeUnavailable,
		utilityVelocityReasonVelocityUnavailable,
	}
	if err := ValidateObservedUtilityInitialVelocityLedger(unsortedReasons); err == nil {
		t.Fatal("validator accepted non-canonical abstention reason order")
	}
}

func TestObservedUtilityInitialVelocityFullDemoReportStreamsAndRejectsTampering(t *testing.T) {
	throw := observedUtilityVelocityTestThrow(1, 1, 10, 7, utility.TypeSmoke, 100)
	throw.InitialVelocityCandidate = observedUtilityVelocityCandidate(throw, utility.InitialVelocityCandidate{
		Vector: utility.Vector{X: 10, Y: 20, Z: 30}, HorizontalSpeed: math.Hypot(10, 20),
		ObservedTick: 100, AvailabilityTick: 100,
		MeasurementStartTick: 100, MeasurementEndTick: 100,
		Status: utility.AvailabilityObserved, Source: utility.SourceProjectileVelocity,
	})
	var rows bytes.Buffer
	report, err := BuildObservedUtilityInitialVelocityFullDemoReport(
		[]utility.Throw{throw}, "demo-0123456789abcdef0123", strings.Repeat("a", 64), &rows,
	)
	if err != nil {
		t.Fatal(err)
	}
	if rows.Len() == 0 || report.RowsArtifact.RecordCount != 1 || report.Coverage.ObservedCount != 1 ||
		report.Summary.ComponentZeros.X != 0 || report.TrainingAllowed {
		t.Fatalf("unexpected report: %+v rows=%q", report, rows.String())
	}
	report.BuildIdentifier = "test-build"
	report.RowsArtifact.Bytes = 10
	report.RowsArtifact.SHA256 = strings.Repeat("b", 64)
	if err := ValidateObservedUtilityInitialVelocityFullDemoReport(report); err != nil {
		t.Fatal(err)
	}
	tampered := report
	tampered.Coverage.ObservedCount = 0
	if err := ValidateObservedUtilityInitialVelocityFullDemoReport(tampered); err == nil {
		t.Fatal("tampered coverage was accepted")
	}
}

func observedUtilityVelocityTestThrow(
	round, sequence, entityID int,
	actorID uint64,
	utilityType utility.Type,
	launchTick int,
) utility.Throw {
	return utility.Throw{
		ID:    "r" + string(rune('0'+round)) + "-u" + string(rune('0'+sequence)),
		Round: round, Sequence: sequence,
		SourceEntityID: entityID, SourceEntityGeneration: 1,
		SourceEntitySerial: entityID + 1000, SourceEntitySerialStatus: utility.AvailabilityObserved,
		EntityStatus: utility.AvailabilityObserved, EntitySource: utility.SourceProjectileEntity,
		Type: utilityType, TypeSource: utility.SourceWeaponInstance,
		Actor: utility.PlayerRef{ID: actorID, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrower},
		Launch: utility.ThrowSnapshot{
			Tick:     utility.TickObservation{Tick: launchTick, Status: utility.AvailabilityObserved, Source: utility.SourceProjectileThrow},
			Position: utility.VectorObservation{Value: utility.Vector{X: 1, Y: 2, Z: 3}, Status: utility.AvailabilityObserved, Source: utility.SourceProjectilePosition},
		},
	}
}

func observedUtilityVelocityCandidate(
	throw utility.Throw,
	candidate utility.InitialVelocityCandidate,
) utility.InitialVelocityCandidate {
	candidate.IdentityStatus = utility.AvailabilityObserved
	candidate.RuntimeEntityID = throw.SourceEntityID
	candidate.RuntimeEntitySerial = throw.SourceEntitySerial
	candidate.ActorID = throw.Actor.ID
	candidate.Type = throw.Type
	return candidate
}
