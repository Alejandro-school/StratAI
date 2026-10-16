package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/utility"
)

const (
	ObservedUtilityInitialVelocityFullDemoSchemaID       = "stratai.observed_utility_initial_velocity_full_demo@1"
	ObservedUtilityInitialVelocityFullDemoReportFilename = "observed_utility_initial_velocity_full_demo.json"
	ObservedUtilityInitialVelocityFullDemoRowsFilename   = "observed_utility_initial_velocity_rows.jsonl.gz"
	ObservedUtilityInitialVelocityRowsDigestContract     = "sha256_compressed_and_decompressed_jsonl_v1"

	utilityVelocityReasonThrowIdentityUnavailable = "throw_identity_unavailable"
	utilityVelocityReasonThrowIdentityAmbiguous   = "throw_identity_ambiguous"
	utilityVelocityReasonEntityUnavailable        = "projectile_entity_unavailable"
	utilityVelocityReasonEntityAmbiguous          = "projectile_identity_ambiguous"
	utilityVelocityReasonLauncherUnavailable      = "launcher_unavailable_at_launch"
	utilityVelocityReasonTypeUnavailable          = "utility_type_unavailable_at_launch"
	utilityVelocityReasonLaunchTickUnavailable    = "launch_tick_unavailable"
	utilityVelocityReasonVelocityUnavailable      = "initial_velocity_unavailable"
	utilityVelocityReasonVelocityNonFinite        = "initial_velocity_non_finite"
	utilityVelocityReasonVelocitySentinel         = "initial_velocity_sentinel"
	utilityVelocityReasonVelocityTickMismatch     = "initial_velocity_tick_mismatch"
	utilityVelocityReasonVelocityWindowInvalid    = "initial_velocity_window_invalid"
	utilityVelocityReasonHorizontalMismatch       = "horizontal_speed_mismatch"
)

var (
	observedUtilityVelocitySHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	observedUtilityVelocityRowIDPattern  = regexp.MustCompile(`^utility-velocity-[0-9]{6}$`)
	observedUtilityVelocityThrowPattern  = regexp.MustCompile(`^utility-throw-[0-9]{6}$`)
	observedUtilityVelocityEntityPattern = regexp.MustCompile(`^projectile-[0-9]{6}$`)
	observedUtilityVelocityPlayerPattern = regexp.MustCompile(`^participant-[0-9]{3}$`)
)

type ObservedUtilityInitialVelocityRowsArtifact struct {
	Path           string `json:"path"`
	Format         string `json:"format"`
	Compression    string `json:"compression"`
	Bytes          int64  `json:"bytes"`
	SHA256         string `json:"sha256"`
	RecordCount    int    `json:"record_count"`
	LogicalBytes   int64  `json:"logical_bytes"`
	LogicalSHA256  string `json:"logical_sha256"`
	DigestContract string `json:"digest_contract"`
}

type ObservedUtilityInitialVelocityCoverage struct {
	RoundNumbers     []int `json:"round_numbers"`
	LaunchCount      int   `json:"launch_count"`
	RowCount         int   `json:"row_count"`
	ParticipantCount int   `json:"participant_count"`
	ObservedCount    int   `json:"observed_count"`
	UnavailableCount int   `json:"unavailable_count"`
}

type ObservedUtilityInitialVelocityComponentZeros struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

type ObservedUtilityInitialVelocitySummary struct {
	Status             map[string]int                               `json:"status"`
	Sources            map[string]int                               `json:"sources"`
	MeasurementMethods map[string]int                               `json:"measurement_methods"`
	UtilityTypes       map[string]int                               `json:"utility_types"`
	AbstentionReasons  map[string]int                               `json:"abstention_reasons"`
	ComponentZeros     ObservedUtilityInitialVelocityComponentZeros `json:"component_zeros"`
}

type ObservedUtilityInitialVelocityFullDemoReport struct {
	SchemaID          string                                     `json:"schema_id"`
	ContractVersion   string                                     `json:"contract_version"`
	CandidateSchemaID string                                     `json:"candidate_schema_id"`
	SourceAlias       string                                     `json:"source_alias"`
	SourceSHA256      string                                     `json:"source_sha256"`
	BuildIdentifier   string                                     `json:"build_identifier"`
	TrainingAllowed   bool                                       `json:"training_allowed"`
	RowsArtifact      ObservedUtilityInitialVelocityRowsArtifact `json:"rows_artifact"`
	Coverage          ObservedUtilityInitialVelocityCoverage     `json:"coverage"`
	Summary           ObservedUtilityInitialVelocitySummary      `json:"summary"`
}

type observedUtilityVelocityCounter struct {
	bytes int64
}

func (counter *observedUtilityVelocityCounter) Write(payload []byte) (int, error) {
	counter.bytes += int64(len(payload))
	return len(payload), nil
}

// BuildObservedUtilityInitialVelocityLedger projects only the candidate
// measurement frozen online by the utility tracker. It never reads trajectory,
// lifecycle, bounce, detonation, effect, or outcome fields.
func BuildObservedUtilityInitialVelocityLedger(
	throws []utility.Throw,
) (models.ObservedUtilityInitialVelocityLedger, error) {
	ledger := models.ObservedUtilityInitialVelocityLedger{
		SchemaID:          models.ObservedUtilityInitialVelocitySchemaID,
		ContractVersion:   models.ObservedUtilityInitialVelocityContractVersion,
		IdentitySemantics: models.TacticalIdentityJoinOnly,
		CausalRole:        models.ObservedUtilityInitialVelocityCausalRole,
		Rows:              make([]models.ObservedUtilityInitialVelocityRow, 0, len(throws)),
	}
	ordered := append([]utility.Throw(nil), throws...)
	sort.SliceStable(ordered, func(left, right int) bool {
		return observedUtilityVelocityThrowLess(ordered[left], ordered[right])
	})

	playerAliases := observedUtilityVelocityPlayerAliases(ordered)

	for index, throw := range ordered {
		row := observedUtilityVelocityRow(
			throw,
			index+1,
			playerAliases,
		)
		ledger.Rows = append(ledger.Rows, row)
	}
	if err := ValidateObservedUtilityInitialVelocityLedger(ledger); err != nil {
		return models.ObservedUtilityInitialVelocityLedger{}, err
	}
	return ledger, nil
}

func observedUtilityVelocitySortTick(throw utility.Throw) int {
	if throw.Launch.Tick.Status == utility.AvailabilityObserved && throw.Launch.Tick.Tick >= 0 {
		return throw.Launch.Tick.Tick
	}
	return int(^uint(0) >> 1)
}

func observedUtilityVelocityThrowLess(left, right utility.Throw) bool {
	if left.Round != right.Round {
		return left.Round < right.Round
	}
	leftTick := observedUtilityVelocitySortTick(left)
	rightTick := observedUtilityVelocitySortTick(right)
	if leftTick != rightTick {
		return leftTick < rightTick
	}
	leftEntity := observedUtilityVelocityEntityPlausible(left)
	rightEntity := observedUtilityVelocityEntityPlausible(right)
	if leftEntity != rightEntity {
		return leftEntity
	}
	if left.SourceEntityID != right.SourceEntityID {
		return left.SourceEntityID < right.SourceEntityID
	}
	leftSerialObserved := left.SourceEntitySerialStatus == utility.AvailabilityObserved
	rightSerialObserved := right.SourceEntitySerialStatus == utility.AvailabilityObserved
	if leftSerialObserved != rightSerialObserved {
		return leftSerialObserved
	}
	if left.SourceEntitySerial != right.SourceEntitySerial {
		return left.SourceEntitySerial < right.SourceEntitySerial
	}
	leftActor := observedUtilityVelocityLauncherExact(left.Actor)
	rightActor := observedUtilityVelocityLauncherExact(right.Actor)
	if leftActor != rightActor {
		return leftActor
	}
	if left.Actor.ID != right.Actor.ID {
		return left.Actor.ID < right.Actor.ID
	}
	if left.Type != right.Type {
		return string(left.Type) < string(right.Type)
	}
	leftPositionObserved := left.Launch.Position.Status == utility.AvailabilityObserved
	rightPositionObserved := right.Launch.Position.Status == utility.AvailabilityObserved
	if leftPositionObserved != rightPositionObserved {
		return leftPositionObserved
	}
	leftCoordinates := [...]uint64{
		math.Float64bits(left.Launch.Position.Value.X),
		math.Float64bits(left.Launch.Position.Value.Y),
		math.Float64bits(left.Launch.Position.Value.Z),
	}
	rightCoordinates := [...]uint64{
		math.Float64bits(right.Launch.Position.Value.X),
		math.Float64bits(right.Launch.Position.Value.Y),
		math.Float64bits(right.Launch.Position.Value.Z),
	}
	for index := range leftCoordinates {
		if leftCoordinates[index] != rightCoordinates[index] {
			return leftCoordinates[index] < rightCoordinates[index]
		}
	}
	if left.Sequence != right.Sequence {
		return left.Sequence < right.Sequence
	}
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	return left.SourceEntityGeneration < right.SourceEntityGeneration
}

func observedUtilityVelocityPlayerAliases(throws []utility.Throw) map[uint64]string {
	aliases := make(map[uint64]string)
	for _, throw := range throws {
		if !observedUtilityVelocityLauncherExact(throw.Actor) {
			continue
		}
		if _, exists := aliases[throw.Actor.ID]; !exists {
			aliases[throw.Actor.ID] = fmt.Sprintf("participant-%03d", len(aliases)+1)
		}
	}
	return aliases
}

func observedUtilityVelocityRow(
	throw utility.Throw,
	index int,
	playerAliases map[uint64]string,
) models.ObservedUtilityInitialVelocityRow {
	row := models.ObservedUtilityInitialVelocityRow{
		RowID:             fmt.Sprintf("utility-velocity-%06d", index),
		RoundNumber:       throw.Round,
		JoinKeys:          models.ObservedUtilityInitialVelocityJoinKeys{ThrowID: fmt.Sprintf("utility-throw-%06d", index)},
		UtilityType:       observedUtilityVelocityTypeFact(throw),
		Launch:            observedUtilityVelocityLaunchFact(throw),
		AbstentionReasons: make([]string, 0),
	}

	reconciliation := models.ObservedUtilityInitialVelocityReconciliationExact
	rawThrowID := strings.TrimSpace(throw.ID)
	if rawThrowID == "" || throw.Sequence <= 0 || throw.Round <= 0 {
		row.AbstentionReasons = append(row.AbstentionReasons, utilityVelocityReasonThrowIdentityUnavailable)
		reconciliation = models.ObservedUtilityInitialVelocityReconciliationUnavailable
	}

	entityExact := observedUtilityVelocityEntityPlausible(throw)
	if !entityExact {
		row.AbstentionReasons = append(row.AbstentionReasons, utilityVelocityReasonEntityUnavailable)
		if reconciliation == models.ObservedUtilityInitialVelocityReconciliationExact {
			reconciliation = models.ObservedUtilityInitialVelocityReconciliationUnavailable
		}
	} else {
		projectileID := fmt.Sprintf("projectile-%06d", index)
		row.JoinKeys.ProjectileID = &projectileID
	}
	if throw.InitialVelocityCandidate.Reason == utility.InitialVelocityReasonIdentityConflict ||
		throw.InitialVelocityCandidate.Reason == utility.InitialVelocityReasonEntityReused {
		reconciliation = models.ObservedUtilityInitialVelocityReconciliationAmbiguous
	}

	if observedUtilityVelocityLauncherExact(throw.Actor) {
		playerID := playerAliases[throw.Actor.ID]
		row.JoinKeys.PlayerID = &playerID
	} else {
		row.AbstentionReasons = append(row.AbstentionReasons, utilityVelocityReasonLauncherUnavailable)
		if reconciliation == models.ObservedUtilityInitialVelocityReconciliationExact {
			reconciliation = models.ObservedUtilityInitialVelocityReconciliationUnavailable
		}
	}
	if row.UtilityType.Status != models.ObservedUtilityInitialVelocityStatusObserved {
		row.AbstentionReasons = append(row.AbstentionReasons, utilityVelocityReasonTypeUnavailable)
		if reconciliation == models.ObservedUtilityInitialVelocityReconciliationExact {
			reconciliation = models.ObservedUtilityInitialVelocityReconciliationUnavailable
		}
	}
	if row.Launch.Status != models.ObservedUtilityInitialVelocityStatusObserved {
		row.AbstentionReasons = append(row.AbstentionReasons, utilityVelocityReasonLaunchTickUnavailable)
		if reconciliation == models.ObservedUtilityInitialVelocityReconciliationExact {
			reconciliation = models.ObservedUtilityInitialVelocityReconciliationUnavailable
		}
	}

	fact, measurementReasons := observedUtilityVelocityFact(throw, row.Launch, reconciliation)
	row.InitialVelocity = fact
	row.AbstentionReasons = append(row.AbstentionReasons, measurementReasons...)
	if row.InitialVelocity.Status == models.ObservedUtilityInitialVelocityStatusUnavailable {
		row.AbstentionReasons = append(row.AbstentionReasons, utilityVelocityReasonVelocityUnavailable)
	}
	row.AbstentionReasons = canonicalObservedUtilityVelocityReasons(row.AbstentionReasons)
	if row.InitialVelocity.Status == models.ObservedUtilityInitialVelocityStatusObserved && len(row.AbstentionReasons) != 0 {
		row.InitialVelocity = unavailableObservedUtilityVelocityFact(reconciliation)
		row.AbstentionReasons = canonicalObservedUtilityVelocityReasons(append(
			row.AbstentionReasons,
			utilityVelocityReasonVelocityUnavailable,
		))
	}
	return row
}

func observedUtilityVelocityTypeFact(throw utility.Throw) models.ObservedUtilityStringFact {
	if throw.Type != utility.TypeUnknown && throw.TypeSource == utility.SourceWeaponInstance {
		value := string(throw.Type)
		return models.ObservedUtilityStringFact{
			Status: models.ObservedUtilityInitialVelocityStatusObserved,
			Value:  &value,
			Source: utility.SourceWeaponInstance,
		}
	}
	return models.ObservedUtilityStringFact{
		Status: models.ObservedUtilityInitialVelocityStatusUnavailable,
		Source: models.ObservedUtilityInitialVelocitySourceUnavailable,
	}
}

func observedUtilityVelocityLaunchFact(throw utility.Throw) models.ObservedUtilityLaunchFact {
	if throw.Launch.Tick.Status == utility.AvailabilityObserved &&
		throw.Launch.Tick.Source == utility.SourceProjectileThrow && throw.Launch.Tick.Tick >= 0 {
		tick := throw.Launch.Tick.Tick
		return models.ObservedUtilityLaunchFact{
			Status: models.ObservedUtilityInitialVelocityStatusObserved,
			Tick:   &tick,
			Source: utility.SourceProjectileThrow,
		}
	}
	return models.ObservedUtilityLaunchFact{
		Status: models.ObservedUtilityInitialVelocityStatusUnavailable,
		Source: models.ObservedUtilityInitialVelocitySourceUnavailable,
	}
}

func observedUtilityVelocityEntityPlausible(throw utility.Throw) bool {
	return throw.EntityStatus == utility.AvailabilityObserved &&
		throw.EntitySource == utility.SourceProjectileEntity &&
		throw.SourceEntityID > 0 && throw.SourceEntityGeneration > 0 &&
		throw.SourceEntitySerialStatus == utility.AvailabilityObserved
}

func observedUtilityVelocityLauncherExact(player utility.PlayerRef) bool {
	return player.Status == utility.AvailabilityObserved && player.ID != 0 &&
		(player.Source == utility.SourceProjectileThrower || player.Source == utility.SourceProjectileOwner)
}

func observedUtilityVelocityFact(
	throw utility.Throw,
	launch models.ObservedUtilityLaunchFact,
	reconciliation string,
) (models.ObservedUtilityInitialVelocityFact, []string) {
	fact := unavailableObservedUtilityVelocityFact(reconciliation)
	reasons := make([]string, 0)
	measurement := throw.InitialVelocityCandidate
	if reconciliation != models.ObservedUtilityInitialVelocityReconciliationExact {
		if strings.TrimSpace(measurement.Reason) != "" {
			reasons = append(reasons, measurement.Reason)
		}
		return fact, append(reasons, utilityVelocityReasonVelocityUnavailable)
	}
	if measurement.Status != utility.AvailabilityObserved {
		if strings.TrimSpace(measurement.Reason) != "" {
			reasons = append(reasons, measurement.Reason)
		}
		return fact, append(reasons, utilityVelocityReasonVelocityUnavailable)
	}
	if measurement.IdentityStatus != utility.AvailabilityObserved ||
		measurement.RuntimeEntityID != throw.SourceEntityID ||
		throw.SourceEntitySerialStatus != utility.AvailabilityObserved ||
		measurement.RuntimeEntitySerial != throw.SourceEntitySerial ||
		measurement.ActorID == 0 || measurement.ActorID != throw.Actor.ID ||
		measurement.Type == utility.TypeUnknown || measurement.Type != throw.Type {
		return fact, append(reasons, utilityVelocityReasonEntityAmbiguous)
	}
	if !observedUtilityVelocityFiniteVector(measurement.Vector) || !isFiniteObservedUtilityVelocity(measurement.HorizontalSpeed) {
		return fact, append(reasons, utilityVelocityReasonVelocityNonFinite)
	}
	if observedUtilityVelocitySentinel(measurement.Vector) || observedUtilityVelocityAllZero(measurement.Vector) {
		return fact, append(reasons, utilityVelocityReasonVelocitySentinel)
	}
	wantHorizontal := math.Hypot(measurement.Vector.X, measurement.Vector.Y)
	if !observedUtilityVelocityClose(measurement.HorizontalSpeed, wantHorizontal) {
		return fact, append(reasons, utilityVelocityReasonHorizontalMismatch)
	}
	if launch.Tick == nil {
		return fact, append(reasons, utilityVelocityReasonVelocityTickMismatch)
	}

	window := models.ObservedUtilityInitialVelocityWindow{Unit: models.ObservedUtilityInitialVelocityWindowUnit}
	publicSource := ""
	parserEvent := ""
	entityProperty := ""
	switch measurement.Source {
	case utility.SourceProjectileVelocity:
		if measurement.ObservedTick != *launch.Tick || measurement.AvailabilityTick != *launch.Tick ||
			measurement.MeasurementStartTick != *launch.Tick || measurement.MeasurementEndTick != *launch.Tick ||
			measurement.MeasurementWindowTicks != 0 {
			return fact, append(reasons, utilityVelocityReasonVelocityTickMismatch)
		}
		start, end, count := *launch.Tick, *launch.Tick, 0
		window.Method = models.ObservedUtilityInitialVelocityWindowMethodNative
		window.StartTick, window.EndTick, window.TickCount = &start, &end, &count
		publicSource = models.ObservedUtilityInitialVelocitySourceNative
		parserEvent = "GrenadeProjectileThrow"
		entityProperty = "m_vecVelocity"
	case utility.SourceProjectilePositionDelta:
		if measurement.MeasurementWindowTicks <= 0 || measurement.MeasurementStartTick != *launch.Tick ||
			measurement.MeasurementEndTick <= measurement.MeasurementStartTick ||
			measurement.MeasurementWindowTicks != measurement.MeasurementEndTick-measurement.MeasurementStartTick ||
			measurement.ObservedTick != measurement.MeasurementEndTick ||
			measurement.AvailabilityTick != measurement.MeasurementEndTick ||
			!isFiniteObservedUtilityVelocity(measurement.TickRate) || measurement.TickRate <= 0 ||
			!observedUtilityVelocityFiniteVector(measurement.StartPosition) ||
			!observedUtilityVelocityFiniteVector(measurement.EndPosition) ||
			observedUtilityVelocitySentinel(measurement.StartPosition) ||
			observedUtilityVelocitySentinel(measurement.EndPosition) {
			return fact, append(reasons, utilityVelocityReasonVelocityWindowInvalid)
		}
		recomputed := observedUtilityVelocityPositionDelta(measurement)
		if !observedUtilityVelocityVectorsClose(recomputed, measurement.Vector) {
			return fact, append(reasons, utilityVelocityReasonVelocityWindowInvalid)
		}
		start, end, count, tickRate := measurement.MeasurementStartTick, measurement.MeasurementEndTick, measurement.MeasurementWindowTicks, measurement.TickRate
		startPosition := observedUtilityVelocityPublicVector(measurement.StartPosition)
		endPosition := observedUtilityVelocityPublicVector(measurement.EndPosition)
		window.Method = models.ObservedUtilityInitialVelocityWindowMethodPositionDelta
		window.StartTick, window.EndTick, window.TickCount, window.TickRate = &start, &end, &count, &tickRate
		window.StartPosition, window.EndPosition = &startPosition, &endPosition
		publicSource = models.ObservedUtilityInitialVelocitySourcePositionDelta
		parserEvent = "GrenadeProjectileThrow+FrameDone"
		entityProperty = "projectile_position"
	default:
		return fact, append(reasons, utilityVelocityReasonVelocityUnavailable)
	}

	vector := observedUtilityVelocityPublicVector(measurement.Vector)
	horizontal := measurement.HorizontalSpeed
	observedTick := measurement.ObservedTick
	availabilityTick := measurement.AvailabilityTick
	fact = models.ObservedUtilityInitialVelocityFact{
		Status:            models.ObservedUtilityInitialVelocityStatusObserved,
		Vector:            &vector,
		HorizontalSpeed:   &horizontal,
		Units:             models.ObservedUtilityInitialVelocityUnits,
		CoordinateFrame:   models.ObservedUtilityInitialVelocityCoordinateFrame,
		ObservedTick:      &observedTick,
		AvailabilityTick:  &availabilityTick,
		MeasurementWindow: window,
		Source:            publicSource,
		Provenance: models.ObservedUtilityInitialVelocityProvenance{
			ParserEvent: parserEvent, EntityProperty: entityProperty,
			Reconciliation: models.ObservedUtilityInitialVelocityReconciliationExact,
		},
	}
	return fact, reasons
}

func unavailableObservedUtilityVelocityFact(reconciliation string) models.ObservedUtilityInitialVelocityFact {
	return models.ObservedUtilityInitialVelocityFact{
		Status:          models.ObservedUtilityInitialVelocityStatusUnavailable,
		Units:           models.ObservedUtilityInitialVelocityUnits,
		CoordinateFrame: models.ObservedUtilityInitialVelocityCoordinateFrame,
		MeasurementWindow: models.ObservedUtilityInitialVelocityWindow{
			Method: models.ObservedUtilityInitialVelocityWindowMethodUnavailable,
			Unit:   models.ObservedUtilityInitialVelocityWindowUnit,
		},
		Source: models.ObservedUtilityInitialVelocitySourceUnavailable,
		Provenance: models.ObservedUtilityInitialVelocityProvenance{
			ParserEvent: "GrenadeProjectileThrow", EntityProperty: "m_vecVelocity_or_projectile_position",
			Reconciliation: reconciliation,
		},
	}
}

func observedUtilityVelocityPositionDelta(measurement utility.InitialVelocityCandidate) utility.Vector {
	scale := measurement.TickRate / float64(measurement.MeasurementWindowTicks)
	return utility.Vector{
		X: (measurement.EndPosition.X - measurement.StartPosition.X) * scale,
		Y: (measurement.EndPosition.Y - measurement.StartPosition.Y) * scale,
		Z: (measurement.EndPosition.Z - measurement.StartPosition.Z) * scale,
	}
}

func observedUtilityVelocityPublicVector(vector utility.Vector) models.ObservedUtilityInitialVelocityVector {
	return models.ObservedUtilityInitialVelocityVector{X: vector.X, Y: vector.Y, Z: vector.Z}
}

func canonicalObservedUtilityVelocityReasons(reasons []string) []string {
	set := make(map[string]struct{}, len(reasons))
	for _, reason := range reasons {
		if reason = strings.TrimSpace(reason); reason != "" {
			set[reason] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for reason := range set {
		result = append(result, reason)
	}
	sort.Strings(result)
	return result
}

func isFiniteObservedUtilityVelocity(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func observedUtilityVelocityFiniteVector(vector utility.Vector) bool {
	return isFiniteObservedUtilityVelocity(vector.X) && isFiniteObservedUtilityVelocity(vector.Y) &&
		isFiniteObservedUtilityVelocity(vector.Z)
}

func observedUtilityVelocityAllZero(vector utility.Vector) bool {
	return vector.X == 0 && vector.Y == 0 && vector.Z == 0
}

func observedUtilityVelocitySentinel(vector utility.Vector) bool {
	sentinel := float64(math.MaxFloat32)
	return math.Abs(vector.X) >= sentinel || math.Abs(vector.Y) >= sentinel || math.Abs(vector.Z) >= sentinel
}

func observedUtilityVelocityClose(left, right float64) bool {
	tolerance := 1e-9 * math.Max(1, math.Max(math.Abs(left), math.Abs(right)))
	return math.Abs(left-right) <= tolerance
}

func observedUtilityVelocityVectorsClose(left, right utility.Vector) bool {
	return observedUtilityVelocityClose(left.X, right.X) &&
		observedUtilityVelocityClose(left.Y, right.Y) &&
		observedUtilityVelocityClose(left.Z, right.Z)
}

func ValidateObservedUtilityInitialVelocityLedger(ledger models.ObservedUtilityInitialVelocityLedger) error {
	if ledger.SchemaID != models.ObservedUtilityInitialVelocitySchemaID ||
		ledger.ContractVersion != models.ObservedUtilityInitialVelocityContractVersion ||
		ledger.IdentitySemantics != models.TacticalIdentityJoinOnly ||
		ledger.CausalRole != models.ObservedUtilityInitialVelocityCausalRole {
		return fmt.Errorf("validate utility velocity ledger: contract metadata mismatch")
	}
	seenRows := make(map[string]struct{}, len(ledger.Rows))
	seenThrows := make(map[string]struct{}, len(ledger.Rows))
	previousRound, previousTick, previousRow := -1, -1, ""
	for index, row := range ledger.Rows {
		if err := validateObservedUtilityVelocityRow(row); err != nil {
			return fmt.Errorf("validate utility velocity ledger row %d: %w", index, err)
		}
		if _, exists := seenRows[row.RowID]; exists {
			return fmt.Errorf("validate utility velocity ledger: duplicate row_id %s", row.RowID)
		}
		if _, exists := seenThrows[row.JoinKeys.ThrowID]; exists {
			return fmt.Errorf("validate utility velocity ledger: duplicate throw_id %s", row.JoinKeys.ThrowID)
		}
		seenRows[row.RowID] = struct{}{}
		seenThrows[row.JoinKeys.ThrowID] = struct{}{}
		tick := int(^uint(0) >> 1)
		if row.Launch.Tick != nil {
			tick = *row.Launch.Tick
		}
		if index > 0 && (row.RoundNumber < previousRound ||
			(row.RoundNumber == previousRound && (tick < previousTick ||
				(tick == previousTick && row.RowID <= previousRow)))) {
			return fmt.Errorf("validate utility velocity ledger: rows are not deterministic")
		}
		previousRound, previousTick, previousRow = row.RoundNumber, tick, row.RowID
	}
	return nil
}

func validateObservedUtilityVelocityRow(row models.ObservedUtilityInitialVelocityRow) error {
	if row.RoundNumber <= 0 || !observedUtilityVelocityRowIDPattern.MatchString(row.RowID) ||
		!observedUtilityVelocityThrowPattern.MatchString(row.JoinKeys.ThrowID) {
		return fmt.Errorf("invalid local row identity")
	}
	if row.JoinKeys.ProjectileID != nil && !observedUtilityVelocityEntityPattern.MatchString(*row.JoinKeys.ProjectileID) {
		return fmt.Errorf("invalid projectile join alias")
	}
	if row.JoinKeys.PlayerID != nil && !observedUtilityVelocityPlayerPattern.MatchString(*row.JoinKeys.PlayerID) {
		return fmt.Errorf("invalid player join alias")
	}
	if err := validateObservedUtilityVelocityStringFact(row.UtilityType); err != nil {
		return err
	}
	if err := validateObservedUtilityVelocityLaunchFact(row.Launch); err != nil {
		return err
	}
	seenReasons := make(map[string]struct{}, len(row.AbstentionReasons))
	previousReason := ""
	for index, reason := range row.AbstentionReasons {
		if !observedUtilityVelocityKnownReason(reason) {
			return fmt.Errorf("invalid abstention reason")
		}
		if index > 0 && reason <= previousReason {
			return fmt.Errorf("abstention reasons are not canonical")
		}
		if _, exists := seenReasons[reason]; exists {
			return fmt.Errorf("duplicate abstention reason")
		}
		seenReasons[reason] = struct{}{}
		previousReason = reason
	}
	if row.InitialVelocity.Units != models.ObservedUtilityInitialVelocityUnits ||
		row.InitialVelocity.CoordinateFrame != models.ObservedUtilityInitialVelocityCoordinateFrame ||
		row.InitialVelocity.MeasurementWindow.Unit != models.ObservedUtilityInitialVelocityWindowUnit {
		return fmt.Errorf("invalid velocity units or coordinate frame")
	}
	if row.InitialVelocity.Status == models.ObservedUtilityInitialVelocityStatusUnavailable {
		if row.InitialVelocity.Vector != nil || row.InitialVelocity.HorizontalSpeed != nil ||
			row.InitialVelocity.ObservedTick != nil || row.InitialVelocity.AvailabilityTick != nil ||
			row.InitialVelocity.Source != models.ObservedUtilityInitialVelocitySourceUnavailable ||
			row.InitialVelocity.MeasurementWindow.Method != models.ObservedUtilityInitialVelocityWindowMethodUnavailable ||
			row.InitialVelocity.MeasurementWindow.StartTick != nil || row.InitialVelocity.MeasurementWindow.EndTick != nil ||
			row.InitialVelocity.MeasurementWindow.TickCount != nil || row.InitialVelocity.MeasurementWindow.TickRate != nil ||
			row.InitialVelocity.MeasurementWindow.StartPosition != nil || row.InitialVelocity.MeasurementWindow.EndPosition != nil ||
			len(row.AbstentionReasons) == 0 {
			return fmt.Errorf("invalid unavailable velocity fact")
		}
		if row.InitialVelocity.Provenance.Reconciliation != models.ObservedUtilityInitialVelocityReconciliationUnavailable &&
			row.InitialVelocity.Provenance.Reconciliation != models.ObservedUtilityInitialVelocityReconciliationAmbiguous &&
			row.InitialVelocity.Provenance.Reconciliation != models.ObservedUtilityInitialVelocityReconciliationExact {
			return fmt.Errorf("invalid unavailable reconciliation")
		}
		if _, exists := seenReasons[utilityVelocityReasonVelocityUnavailable]; !exists {
			return fmt.Errorf("unavailable velocity is missing its generic reason")
		}
		ambiguousCauseCount := 0
		for _, reason := range []string{
			utility.InitialVelocityReasonIdentityConflict,
			utility.InitialVelocityReasonEntityReused,
		} {
			if _, exists := seenReasons[reason]; exists {
				ambiguousCauseCount++
			}
		}
		if row.InitialVelocity.Provenance.Reconciliation == models.ObservedUtilityInitialVelocityReconciliationAmbiguous {
			if ambiguousCauseCount != 1 {
				return fmt.Errorf("ambiguous reconciliation requires exactly one identity cause")
			}
		} else if ambiguousCauseCount != 0 {
			return fmt.Errorf("identity ambiguity reason requires ambiguous reconciliation")
		}
		if row.InitialVelocity.Provenance.ParserEvent != "GrenadeProjectileThrow" ||
			row.InitialVelocity.Provenance.EntityProperty != "m_vecVelocity_or_projectile_position" {
			return fmt.Errorf("invalid unavailable provenance")
		}
		return nil
	}
	if row.InitialVelocity.Status != models.ObservedUtilityInitialVelocityStatusObserved ||
		row.InitialVelocity.Vector == nil || row.InitialVelocity.HorizontalSpeed == nil ||
		row.InitialVelocity.ObservedTick == nil || row.InitialVelocity.AvailabilityTick == nil ||
		row.JoinKeys.ProjectileID == nil || row.JoinKeys.PlayerID == nil ||
		row.UtilityType.Status != models.ObservedUtilityInitialVelocityStatusObserved ||
		row.Launch.Status != models.ObservedUtilityInitialVelocityStatusObserved || row.Launch.Tick == nil ||
		len(row.AbstentionReasons) != 0 ||
		row.InitialVelocity.Provenance.Reconciliation != models.ObservedUtilityInitialVelocityReconciliationExact {
		return fmt.Errorf("invalid observed velocity shape")
	}
	vector := row.InitialVelocity.Vector
	if !isFiniteObservedUtilityVelocity(vector.X) || !isFiniteObservedUtilityVelocity(vector.Y) ||
		!isFiniteObservedUtilityVelocity(vector.Z) || !isFiniteObservedUtilityVelocity(*row.InitialVelocity.HorizontalSpeed) ||
		observedUtilityVelocitySentinel(utility.Vector{X: vector.X, Y: vector.Y, Z: vector.Z}) ||
		observedUtilityVelocityAllZero(utility.Vector{X: vector.X, Y: vector.Y, Z: vector.Z}) ||
		!observedUtilityVelocityClose(*row.InitialVelocity.HorizontalSpeed, math.Hypot(vector.X, vector.Y)) {
		return fmt.Errorf("invalid observed velocity value")
	}
	window := row.InitialVelocity.MeasurementWindow
	if *row.InitialVelocity.AvailabilityTick < *row.InitialVelocity.ObservedTick {
		return fmt.Errorf("availability precedes observation")
	}
	switch row.InitialVelocity.Source {
	case models.ObservedUtilityInitialVelocitySourceNative:
		if window.Method != models.ObservedUtilityInitialVelocityWindowMethodNative ||
			window.StartTick == nil || window.EndTick == nil || window.TickCount == nil ||
			*window.StartTick != *row.Launch.Tick || *window.EndTick != *row.Launch.Tick || *window.TickCount != 0 ||
			window.TickRate != nil || window.StartPosition != nil || window.EndPosition != nil ||
			row.InitialVelocity.Provenance.ParserEvent != "GrenadeProjectileThrow" ||
			row.InitialVelocity.Provenance.EntityProperty != "m_vecVelocity" ||
			*row.InitialVelocity.ObservedTick != *row.Launch.Tick || *row.InitialVelocity.AvailabilityTick != *row.Launch.Tick {
			return fmt.Errorf("invalid native measurement window")
		}
	case models.ObservedUtilityInitialVelocitySourcePositionDelta:
		if window.Method != models.ObservedUtilityInitialVelocityWindowMethodPositionDelta ||
			window.StartTick == nil || window.EndTick == nil || window.TickCount == nil || window.TickRate == nil ||
			window.StartPosition == nil || window.EndPosition == nil || *window.TickCount <= 0 ||
			*window.StartTick != *row.Launch.Tick || *window.EndTick-*window.StartTick != *window.TickCount ||
			*row.InitialVelocity.ObservedTick != *window.EndTick || *row.InitialVelocity.AvailabilityTick != *window.EndTick ||
			!isFiniteObservedUtilityVelocity(*window.TickRate) || *window.TickRate <= 0 ||
			row.InitialVelocity.Provenance.ParserEvent != "GrenadeProjectileThrow+FrameDone" ||
			row.InitialVelocity.Provenance.EntityProperty != "projectile_position" {
			return fmt.Errorf("invalid position-delta measurement window")
		}
		startPosition := utility.Vector{
			X: window.StartPosition.X, Y: window.StartPosition.Y, Z: window.StartPosition.Z,
		}
		endPosition := utility.Vector{
			X: window.EndPosition.X, Y: window.EndPosition.Y, Z: window.EndPosition.Z,
		}
		if !observedUtilityVelocityFiniteVector(startPosition) ||
			!observedUtilityVelocityFiniteVector(endPosition) ||
			observedUtilityVelocitySentinel(startPosition) ||
			observedUtilityVelocitySentinel(endPosition) {
			return fmt.Errorf("invalid position-delta endpoint")
		}
		scale := *window.TickRate / float64(*window.TickCount)
		recomputed := utility.Vector{
			X: (window.EndPosition.X - window.StartPosition.X) * scale,
			Y: (window.EndPosition.Y - window.StartPosition.Y) * scale,
			Z: (window.EndPosition.Z - window.StartPosition.Z) * scale,
		}
		if !observedUtilityVelocityVectorsClose(recomputed, utility.Vector{X: vector.X, Y: vector.Y, Z: vector.Z}) {
			return fmt.Errorf("position-delta value mismatch")
		}
	default:
		return fmt.Errorf("invalid velocity source")
	}
	return nil
}

func validateObservedUtilityVelocityStringFact(fact models.ObservedUtilityStringFact) error {
	if fact.Status == models.ObservedUtilityInitialVelocityStatusObserved {
		if fact.Value == nil || !observedUtilityVelocityKnownType(*fact.Value) ||
			fact.Source != utility.SourceWeaponInstance {
			return fmt.Errorf("invalid observed utility type")
		}
		return nil
	}
	if fact.Status != models.ObservedUtilityInitialVelocityStatusUnavailable || fact.Value != nil ||
		fact.Source != models.ObservedUtilityInitialVelocitySourceUnavailable {
		return fmt.Errorf("invalid unavailable utility type")
	}
	return nil
}

func observedUtilityVelocityKnownType(value string) bool {
	switch value {
	case string(utility.TypeFlashbang), string(utility.TypeSmoke), string(utility.TypeHE),
		string(utility.TypeMolotov), string(utility.TypeIncendiary), string(utility.TypeDecoy):
		return true
	default:
		return false
	}
}

func observedUtilityVelocityKnownReason(value string) bool {
	switch value {
	case utilityVelocityReasonThrowIdentityUnavailable,
		utilityVelocityReasonThrowIdentityAmbiguous,
		utilityVelocityReasonEntityUnavailable,
		utilityVelocityReasonEntityAmbiguous,
		utilityVelocityReasonLauncherUnavailable,
		utilityVelocityReasonTypeUnavailable,
		utilityVelocityReasonLaunchTickUnavailable,
		utilityVelocityReasonVelocityUnavailable,
		utilityVelocityReasonVelocityNonFinite,
		utilityVelocityReasonVelocitySentinel,
		utilityVelocityReasonVelocityTickMismatch,
		utilityVelocityReasonVelocityWindowInvalid,
		utilityVelocityReasonHorizontalMismatch,
		utility.InitialVelocityReasonNativeUnavailable,
		utility.InitialVelocityReasonNativeNonFinite,
		utility.InitialVelocityReasonNativeSentinel,
		utility.InitialVelocityReasonNativeZeroAmbiguous,
		utility.InitialVelocityReasonLaunchPositionUnavailable,
		utility.InitialVelocityReasonTickRateUnavailable,
		utility.InitialVelocityReasonInsufficientSamples,
		utility.InitialVelocityReasonSameTickConflict,
		utility.InitialVelocityReasonFlightEnded,
		utility.InitialVelocityReasonIdentityUnavailable,
		utility.InitialVelocityReasonIdentityConflict,
		utility.InitialVelocityReasonEntityReused:
		return true
	default:
		return false
	}
}

func validateObservedUtilityVelocityLaunchFact(fact models.ObservedUtilityLaunchFact) error {
	if fact.Status == models.ObservedUtilityInitialVelocityStatusObserved {
		if fact.Tick == nil || *fact.Tick < 0 || fact.Source != utility.SourceProjectileThrow {
			return fmt.Errorf("invalid observed launch")
		}
		return nil
	}
	if fact.Status != models.ObservedUtilityInitialVelocityStatusUnavailable || fact.Tick != nil ||
		fact.Source != models.ObservedUtilityInitialVelocitySourceUnavailable {
		return fmt.Errorf("invalid unavailable launch")
	}
	return nil
}

func BuildObservedUtilityInitialVelocityFullDemoReport(
	throws []utility.Throw,
	sourceAlias string,
	sourceSHA256 string,
	rowsWriter io.Writer,
) (ObservedUtilityInitialVelocityFullDemoReport, error) {
	if rowsWriter == nil {
		return ObservedUtilityInitialVelocityFullDemoReport{}, fmt.Errorf("build utility velocity full-demo report: rows writer is nil")
	}
	sourceAlias = strings.TrimSpace(sourceAlias)
	if sourceAlias == "" || !observedUtilityVelocitySHA256Pattern.MatchString(sourceSHA256) {
		return ObservedUtilityInitialVelocityFullDemoReport{}, fmt.Errorf("build utility velocity full-demo report: invalid source provenance")
	}
	ledger, err := BuildObservedUtilityInitialVelocityLedger(throws)
	if err != nil {
		return ObservedUtilityInitialVelocityFullDemoReport{}, err
	}
	if len(ledger.Rows) == 0 {
		return ObservedUtilityInitialVelocityFullDemoReport{}, fmt.Errorf("build utility velocity full-demo report: candidate produced no rows")
	}
	logicalHash := sha256.New()
	logicalCounter := &observedUtilityVelocityCounter{}
	encoder := json.NewEncoder(io.MultiWriter(rowsWriter, logicalHash, logicalCounter))
	encoder.SetEscapeHTML(false)
	for index := range ledger.Rows {
		if err := encoder.Encode(ledger.Rows[index]); err != nil {
			return ObservedUtilityInitialVelocityFullDemoReport{}, fmt.Errorf("write utility velocity row %d: %w", index, err)
		}
	}
	coverage, summary := summarizeObservedUtilityVelocityRows(ledger.Rows)
	report := ObservedUtilityInitialVelocityFullDemoReport{
		SchemaID:          ObservedUtilityInitialVelocityFullDemoSchemaID,
		ContractVersion:   models.ObservedUtilityInitialVelocityContractVersion,
		CandidateSchemaID: models.ObservedUtilityInitialVelocitySchemaID,
		SourceAlias:       sourceAlias,
		SourceSHA256:      sourceSHA256,
		TrainingAllowed:   false,
		RowsArtifact: ObservedUtilityInitialVelocityRowsArtifact{
			Path: ObservedUtilityInitialVelocityFullDemoRowsFilename, Format: "jsonl", Compression: "gzip",
			RecordCount: len(ledger.Rows), LogicalBytes: logicalCounter.bytes,
			LogicalSHA256:  hex.EncodeToString(logicalHash.Sum(nil)),
			DigestContract: ObservedUtilityInitialVelocityRowsDigestContract,
		},
		Coverage: coverage,
		Summary:  summary,
	}
	if err := validateObservedUtilityInitialVelocityFullDemoReport(report, false); err != nil {
		return ObservedUtilityInitialVelocityFullDemoReport{}, err
	}
	return report, nil
}

func summarizeObservedUtilityVelocityRows(
	rows []models.ObservedUtilityInitialVelocityRow,
) (ObservedUtilityInitialVelocityCoverage, ObservedUtilityInitialVelocitySummary) {
	rounds := make(map[int]struct{})
	players := make(map[string]struct{})
	summary := ObservedUtilityInitialVelocitySummary{
		Status: map[string]int{
			models.ObservedUtilityInitialVelocityStatusObserved:    0,
			models.ObservedUtilityInitialVelocityStatusUnavailable: 0,
		},
		Sources: make(map[string]int), MeasurementMethods: make(map[string]int),
		UtilityTypes: make(map[string]int), AbstentionReasons: make(map[string]int),
	}
	for _, row := range rows {
		rounds[row.RoundNumber] = struct{}{}
		if row.JoinKeys.PlayerID != nil {
			players[*row.JoinKeys.PlayerID] = struct{}{}
		}
		summary.Status[row.InitialVelocity.Status]++
		summary.Sources[row.InitialVelocity.Source]++
		summary.MeasurementMethods[row.InitialVelocity.MeasurementWindow.Method]++
		if row.UtilityType.Value != nil {
			summary.UtilityTypes[*row.UtilityType.Value]++
		} else {
			summary.UtilityTypes[models.ObservedUtilityInitialVelocitySourceUnavailable]++
		}
		for _, reason := range row.AbstentionReasons {
			summary.AbstentionReasons[reason]++
		}
		if row.InitialVelocity.Vector != nil {
			if row.InitialVelocity.Vector.X == 0 {
				summary.ComponentZeros.X++
			}
			if row.InitialVelocity.Vector.Y == 0 {
				summary.ComponentZeros.Y++
			}
			if row.InitialVelocity.Vector.Z == 0 {
				summary.ComponentZeros.Z++
			}
		}
	}
	roundNumbers := make([]int, 0, len(rounds))
	for round := range rounds {
		roundNumbers = append(roundNumbers, round)
	}
	sort.Ints(roundNumbers)
	coverage := ObservedUtilityInitialVelocityCoverage{
		RoundNumbers: roundNumbers, LaunchCount: len(rows), RowCount: len(rows), ParticipantCount: len(players),
		ObservedCount:    summary.Status[models.ObservedUtilityInitialVelocityStatusObserved],
		UnavailableCount: summary.Status[models.ObservedUtilityInitialVelocityStatusUnavailable],
	}
	return coverage, summary
}

func ValidateObservedUtilityInitialVelocityFullDemoReport(report ObservedUtilityInitialVelocityFullDemoReport) error {
	return validateObservedUtilityInitialVelocityFullDemoReport(report, true)
}

func validateObservedUtilityInitialVelocityFullDemoReport(
	report ObservedUtilityInitialVelocityFullDemoReport,
	requirePhysical bool,
) error {
	if report.SchemaID != ObservedUtilityInitialVelocityFullDemoSchemaID ||
		report.ContractVersion != models.ObservedUtilityInitialVelocityContractVersion ||
		report.CandidateSchemaID != models.ObservedUtilityInitialVelocitySchemaID || report.TrainingAllowed {
		return fmt.Errorf("validate utility velocity full-demo report: contract metadata mismatch")
	}
	if strings.TrimSpace(report.SourceAlias) == "" || !observedUtilityVelocitySHA256Pattern.MatchString(report.SourceSHA256) {
		return fmt.Errorf("validate utility velocity full-demo report: invalid source provenance")
	}
	artifact := report.RowsArtifact
	if artifact.Path != ObservedUtilityInitialVelocityFullDemoRowsFilename || artifact.Format != "jsonl" ||
		artifact.Compression != "gzip" || artifact.RecordCount <= 0 || artifact.LogicalBytes <= 0 ||
		!observedUtilityVelocitySHA256Pattern.MatchString(artifact.LogicalSHA256) ||
		artifact.DigestContract != ObservedUtilityInitialVelocityRowsDigestContract {
		return fmt.Errorf("validate utility velocity full-demo report: invalid logical artifact")
	}
	if requirePhysical && (strings.TrimSpace(report.BuildIdentifier) == "" || artifact.Bytes <= 0 ||
		!observedUtilityVelocitySHA256Pattern.MatchString(artifact.SHA256)) {
		return fmt.Errorf("validate utility velocity full-demo report: invalid physical artifact")
	}
	coverage := report.Coverage
	if len(coverage.RoundNumbers) == 0 || !sort.IntsAreSorted(coverage.RoundNumbers) ||
		coverage.LaunchCount != artifact.RecordCount || coverage.RowCount != artifact.RecordCount ||
		coverage.ObservedCount < 0 || coverage.UnavailableCount < 0 ||
		coverage.ObservedCount+coverage.UnavailableCount != artifact.RecordCount || coverage.ParticipantCount < 0 {
		return fmt.Errorf("validate utility velocity full-demo report: invalid coverage")
	}
	if report.Summary.Status[models.ObservedUtilityInitialVelocityStatusObserved] != coverage.ObservedCount ||
		report.Summary.Status[models.ObservedUtilityInitialVelocityStatusUnavailable] != coverage.UnavailableCount {
		return fmt.Errorf("validate utility velocity full-demo report: status summary mismatch")
	}
	sourceTotal, methodTotal := 0, 0
	for _, count := range report.Summary.Sources {
		if count < 0 {
			return fmt.Errorf("validate utility velocity full-demo report: invalid source count")
		}
		sourceTotal += count
	}
	for _, count := range report.Summary.MeasurementMethods {
		if count < 0 {
			return fmt.Errorf("validate utility velocity full-demo report: invalid method count")
		}
		methodTotal += count
	}
	if sourceTotal != artifact.RecordCount || methodTotal != artifact.RecordCount {
		return fmt.Errorf("validate utility velocity full-demo report: summary total mismatch")
	}
	return nil
}
