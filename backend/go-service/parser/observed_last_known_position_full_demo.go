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
)

const (
	ObservedLastKnownPositionFullDemoSchemaID       = "stratai.observed_last_known_position_full_demo@1"
	ObservedLastKnownPositionFullDemoReportFilename = "observed_last_known_position_full_demo.json"
	ObservedLastKnownPositionFullDemoRowsFilename   = "observed_last_known_position_rows.jsonl.gz"
	ObservedLastKnownPositionRowsDigestContract     = "sha256_compressed_and_decompressed_jsonl_v1"
)

var lastKnownFullDemoSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type ObservedLastKnownPositionRowsArtifact struct {
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

type ObservedLastKnownPositionCoverage struct {
	RoundNumbers     []int `json:"round_numbers"`
	RoundCount       int   `json:"round_count"`
	RowCount         int   `json:"row_count"`
	ParticipantCount int   `json:"participant_count"`
	DirectedPairs    int   `json:"directed_pairs"`
	ContactUpdates   int   `json:"contact_updates"`
	ContactRefreshes int   `json:"contact_refreshes"`
	Carries          int   `json:"carries"`
	Unavailable      int   `json:"unavailable"`
	RoundResets      int   `json:"round_resets"`
	LifeResets       int   `json:"life_resets"`
}

type ObservedInformationAgeDistribution struct {
	ObservedCount int            `json:"observed_count"`
	ZeroCount     int            `json:"zero_count"`
	PositiveCount int            `json:"positive_count"`
	MinimumMS     *float64       `json:"minimum_ms"`
	P50MS         *float64       `json:"p50_ms"`
	P95MS         *float64       `json:"p95_ms"`
	MaximumMS     *float64       `json:"maximum_ms"`
	Buckets       map[string]int `json:"buckets"`
}

type ObservedLastKnownPositionSummary struct {
	Status              map[string]int                     `json:"status"`
	Derivations         map[string]int                     `json:"derivations"`
	ContactObservations map[string]int                     `json:"contact_observations"`
	CarrySources        map[string]int                     `json:"carry_sources"`
	AbstentionReasons   map[string]int                     `json:"abstention_reasons"`
	AgeDistribution     ObservedInformationAgeDistribution `json:"age_distribution"`
}

type ObservedLastKnownPositionFullDemoReport struct {
	SchemaID          string                                `json:"schema_id"`
	ContractVersion   string                                `json:"contract_version"`
	CandidateSchemaID string                                `json:"candidate_schema_id"`
	SourceAlias       string                                `json:"source_alias"`
	SourceSHA256      string                                `json:"source_sha256"`
	BuildIdentifier   string                                `json:"build_identifier"`
	TrainingAllowed   bool                                  `json:"training_allowed"`
	TickRateHz        float64                               `json:"tick_rate_hz"`
	RowsArtifact      ObservedLastKnownPositionRowsArtifact `json:"rows_artifact"`
	Coverage          ObservedLastKnownPositionCoverage     `json:"coverage"`
	Summary           ObservedLastKnownPositionSummary      `json:"summary"`
}

type observedLastKnownPositionCounter struct{ bytes int64 }

func (counter *observedLastKnownPositionCounter) Write(payload []byte) (int, error) {
	counter.bytes += int64(len(payload))
	return len(payload), nil
}

// BuildObservedLastKnownPositionFullDemoReport evaluates every round from the
// already parsed replay. It reuses BuildObservedVisualContactLedger and never
// opens the source demo or writes into the canonical bundle.
func BuildObservedLastKnownPositionFullDemoReport(
	replay *models.ReplayData,
	geometry TacticalGeometry,
	sourceAlias, sourceSHA256 string,
	rowsWriter io.Writer,
) (ObservedLastKnownPositionFullDemoReport, error) {
	if replay == nil || rowsWriter == nil {
		return ObservedLastKnownPositionFullDemoReport{}, fmt.Errorf("build last-known full-demo: replay or rows writer is nil")
	}
	if !lastKnownSourceAliasPattern.MatchString(strings.TrimSpace(sourceAlias)) ||
		!lastKnownFullDemoSHA256Pattern.MatchString(sourceSHA256) {
		return ObservedLastKnownPositionFullDemoReport{}, fmt.Errorf("build last-known full-demo: invalid source provenance")
	}
	if geometry == nil || !geometry.IsLoaded() {
		return ObservedLastKnownPositionFullDemoReport{}, fmt.Errorf("build last-known full-demo: loaded tactical geometry is required")
	}
	if math.IsNaN(replay.Metadata.TickRate) || math.IsInf(replay.Metadata.TickRate, 0) || replay.Metadata.TickRate <= 0 {
		return ObservedLastKnownPositionFullDemoReport{}, fmt.Errorf("build last-known full-demo: invalid tick rate")
	}
	rounds, err := canonicalObservedVisualContactFullDemoRounds(replay.Rounds)
	if err != nil {
		return ObservedLastKnownPositionFullDemoReport{}, err
	}
	syntheticByRaw, participantAliases := observedVisualContactFullDemoAliases(rounds)
	logicalHash := sha256.New()
	logicalCounter := &observedLastKnownPositionCounter{}
	encoder := json.NewEncoder(io.MultiWriter(rowsWriter, logicalHash, logicalCounter))
	encoder.SetEscapeHTML(false)

	coverage := ObservedLastKnownPositionCoverage{
		RoundNumbers: make([]int, 0, len(rounds)),
		RoundCount:   len(rounds), RoundResets: len(rounds),
	}
	summary := newObservedLastKnownPositionSummary()
	directedPairs := make(map[string]struct{})
	participants := make(map[string]struct{})
	ageValues := make([]float64, 0)
	rowSequence := 0
	for _, round := range rounds {
		roundReplay, err := observedVisualContactFullDemoRoundReplay(replay, round, sourceAlias, syntheticByRaw)
		if err != nil {
			return ObservedLastKnownPositionFullDemoReport{}, err
		}
		visualLedger, err := BuildObservedVisualContactLedger(roundReplay, geometry, round.EndTick)
		if err != nil {
			return ObservedLastKnownPositionFullDemoReport{}, fmt.Errorf("build last-known full-demo: round %d visual producer: %w", round.Round, err)
		}
		ledger, err := BuildObservedLastKnownPositionLedger(roundReplay, visualLedger, participantAliases)
		if err != nil {
			return ObservedLastKnownPositionFullDemoReport{}, fmt.Errorf("build last-known full-demo: round %d candidate producer: %w", round.Round, err)
		}
		coverage.RoundNumbers = append(coverage.RoundNumbers, round.Round)
		seenUpdate := make(map[string]struct{})
		lifeByPair := make(map[string][2]int)
		for _, sourceRow := range ledger.Rows {
			row := sourceRow
			rowSequence++
			row.RowID = fmt.Sprintf("last-known-%09d", rowSequence)
			pair := row.JoinKeys.ObserverAlias + "\x00" + row.JoinKeys.SubjectAlias
			directedPairs[pair] = struct{}{}
			participants[row.JoinKeys.ObserverAlias] = struct{}{}
			participants[row.JoinKeys.SubjectAlias] = struct{}{}
			if row.ContactObservation.ObserverLifeSequence != nil && row.ContactObservation.SubjectLifeSequence != nil {
				life := [2]int{*row.ContactObservation.ObserverLifeSequence, *row.ContactObservation.SubjectLifeSequence}
				if previous, exists := lifeByPair[pair]; exists && previous != life {
					coverage.LifeResets++
				}
				lifeByPair[pair] = life
			}
			if row.Derivation == models.ObservedLastKnownPositionDerivationUpdate {
				coverage.ContactUpdates++
				if _, exists := seenUpdate[pair]; exists {
					coverage.ContactRefreshes++
				}
				seenUpdate[pair] = struct{}{}
			} else if row.Derivation == models.ObservedLastKnownPositionDerivationCarry {
				coverage.Carries++
			} else {
				coverage.Unavailable++
			}
			accumulateObservedLastKnownPositionSummary(&summary, row, &ageValues)
			if err := encoder.Encode(row); err != nil {
				return ObservedLastKnownPositionFullDemoReport{}, fmt.Errorf("write last-known row %d: %w", rowSequence, err)
			}
		}
	}
	if rowSequence == 0 || coverage.ContactUpdates == 0 {
		return ObservedLastKnownPositionFullDemoReport{}, fmt.Errorf("build last-known full-demo: candidate coverage is empty")
	}
	coverage.RowCount = rowSequence
	coverage.ParticipantCount = len(participants)
	coverage.DirectedPairs = len(directedPairs)
	summary.AgeDistribution = observedLastKnownPositionAgeDistribution(ageValues)
	report := ObservedLastKnownPositionFullDemoReport{
		SchemaID:          ObservedLastKnownPositionFullDemoSchemaID,
		ContractVersion:   models.ObservedLastKnownPositionContractVersion,
		CandidateSchemaID: models.ObservedLastKnownPositionSchemaID,
		SourceAlias:       sourceAlias,
		SourceSHA256:      sourceSHA256,
		TrainingAllowed:   false,
		TickRateHz:        replay.Metadata.TickRate,
		RowsArtifact: ObservedLastKnownPositionRowsArtifact{
			Path:   ObservedLastKnownPositionFullDemoRowsFilename,
			Format: "jsonl", Compression: "gzip", RecordCount: rowSequence,
			LogicalBytes:   logicalCounter.bytes,
			LogicalSHA256:  hex.EncodeToString(logicalHash.Sum(nil)),
			DigestContract: ObservedLastKnownPositionRowsDigestContract,
		},
		Coverage: coverage,
		Summary:  summary,
	}
	if err := validateObservedLastKnownPositionFullDemoReport(report, false); err != nil {
		return ObservedLastKnownPositionFullDemoReport{}, err
	}
	return report, nil
}

func newObservedLastKnownPositionSummary() ObservedLastKnownPositionSummary {
	return ObservedLastKnownPositionSummary{
		Status: map[string]int{
			models.ObservedLastKnownPositionStatusObserved:    0,
			models.ObservedLastKnownPositionStatusUnavailable: 0,
		},
		Derivations: map[string]int{
			models.ObservedLastKnownPositionDerivationUpdate:      0,
			models.ObservedLastKnownPositionDerivationCarry:       0,
			models.ObservedLastKnownPositionDerivationUnavailable: 0,
		},
		ContactObservations: map[string]int{"true": 0, "false": 0, "unavailable": 0},
		CarrySources:        map[string]int{"negative_contact": 0, "unavailable_contact": 0},
		AbstentionReasons:   make(map[string]int),
	}
}

func accumulateObservedLastKnownPositionSummary(
	summary *ObservedLastKnownPositionSummary,
	row models.ObservedLastKnownPositionRow,
	ageValues *[]float64,
) {
	summary.Status[row.LastKnownPosition.Status]++
	summary.Derivations[row.Derivation]++
	if row.ContactObservation.Status == models.ObservedLastKnownPositionStatusUnavailable {
		summary.ContactObservations["unavailable"]++
		if row.Derivation == models.ObservedLastKnownPositionDerivationCarry {
			summary.CarrySources["unavailable_contact"]++
		}
	} else if row.ContactObservation.VisualContact != nil && *row.ContactObservation.VisualContact {
		summary.ContactObservations["true"]++
	} else {
		summary.ContactObservations["false"]++
		if row.Derivation == models.ObservedLastKnownPositionDerivationCarry {
			summary.CarrySources["negative_contact"]++
		}
	}
	for _, reason := range row.AbstentionReasons {
		summary.AbstentionReasons[reason]++
	}
	if row.InformationAge.ValueMS != nil {
		*ageValues = append(*ageValues, *row.InformationAge.ValueMS)
	}
}

func observedLastKnownPositionAgeDistribution(values []float64) ObservedInformationAgeDistribution {
	buckets := map[string]int{
		"0": 0, "gt_0_le_250": 0, "gt_250_le_1000": 0,
		"gt_1000_le_5000": 0, "gt_5000": 0,
	}
	for _, value := range values {
		switch {
		case value == 0:
			buckets["0"]++
		case value <= 250:
			buckets["gt_0_le_250"]++
		case value <= 1000:
			buckets["gt_250_le_1000"]++
		case value <= 5000:
			buckets["gt_1000_le_5000"]++
		default:
			buckets["gt_5000"]++
		}
	}
	result := ObservedInformationAgeDistribution{
		ObservedCount: len(values), ZeroCount: buckets["0"],
		PositiveCount: len(values) - buckets["0"], Buckets: buckets,
	}
	if len(values) == 0 {
		return result
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	result.MinimumMS = lastKnownFloatPointer(sorted[0])
	result.P50MS = lastKnownFloatPointer(sorted[lastKnownNearestRank(len(sorted), 0.50)])
	result.P95MS = lastKnownFloatPointer(sorted[lastKnownNearestRank(len(sorted), 0.95)])
	result.MaximumMS = lastKnownFloatPointer(sorted[len(sorted)-1])
	return result
}

func lastKnownNearestRank(length int, quantile float64) int {
	index := int(math.Ceil(float64(length)*quantile)) - 1
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func ValidateObservedLastKnownPositionFullDemoReport(report ObservedLastKnownPositionFullDemoReport) error {
	return validateObservedLastKnownPositionFullDemoReport(report, true)
}

func validateObservedLastKnownPositionFullDemoReport(
	report ObservedLastKnownPositionFullDemoReport,
	requirePhysical bool,
) error {
	if report.SchemaID != ObservedLastKnownPositionFullDemoSchemaID ||
		report.ContractVersion != models.ObservedLastKnownPositionContractVersion ||
		report.CandidateSchemaID != models.ObservedLastKnownPositionSchemaID || report.TrainingAllowed ||
		!lastKnownSourceAliasPattern.MatchString(report.SourceAlias) ||
		!lastKnownFullDemoSHA256Pattern.MatchString(report.SourceSHA256) ||
		math.IsNaN(report.TickRateHz) || math.IsInf(report.TickRateHz, 0) || report.TickRateHz <= 0 {
		return fmt.Errorf("validate last-known full-demo: contract metadata mismatch")
	}
	artifact := report.RowsArtifact
	if artifact.Path != ObservedLastKnownPositionFullDemoRowsFilename || artifact.Format != "jsonl" ||
		artifact.Compression != "gzip" || artifact.RecordCount <= 0 || artifact.LogicalBytes <= 0 ||
		!lastKnownFullDemoSHA256Pattern.MatchString(artifact.LogicalSHA256) ||
		artifact.DigestContract != ObservedLastKnownPositionRowsDigestContract {
		return fmt.Errorf("validate last-known full-demo: invalid logical artifact")
	}
	if requirePhysical && (strings.TrimSpace(report.BuildIdentifier) == "" || artifact.Bytes <= 0 ||
		!lastKnownFullDemoSHA256Pattern.MatchString(artifact.SHA256)) {
		return fmt.Errorf("validate last-known full-demo: invalid physical artifact")
	}
	coverage := report.Coverage
	if coverage.RoundCount <= 0 || len(coverage.RoundNumbers) != coverage.RoundCount ||
		!sort.IntsAreSorted(coverage.RoundNumbers) || coverage.RowCount != artifact.RecordCount ||
		coverage.ParticipantCount <= 0 || coverage.DirectedPairs <= 0 || coverage.ContactUpdates <= 0 ||
		coverage.ContactRefreshes < 0 || coverage.ContactRefreshes > coverage.ContactUpdates ||
		coverage.Carries < 0 || coverage.Unavailable < 0 ||
		coverage.ContactUpdates+coverage.Carries+coverage.Unavailable != coverage.RowCount ||
		coverage.RoundResets != coverage.RoundCount || coverage.LifeResets < 0 {
		return fmt.Errorf("validate last-known full-demo: invalid coverage")
	}
	if report.Summary.Status[models.ObservedLastKnownPositionStatusObserved] != coverage.ContactUpdates+coverage.Carries ||
		report.Summary.Status[models.ObservedLastKnownPositionStatusUnavailable] != coverage.Unavailable ||
		report.Summary.Derivations[models.ObservedLastKnownPositionDerivationUpdate] != coverage.ContactUpdates ||
		report.Summary.Derivations[models.ObservedLastKnownPositionDerivationCarry] != coverage.Carries ||
		report.Summary.Derivations[models.ObservedLastKnownPositionDerivationUnavailable] != coverage.Unavailable {
		return fmt.Errorf("validate last-known full-demo: summary mismatch")
	}
	contactTotal := 0
	for _, count := range report.Summary.ContactObservations {
		if count < 0 {
			return fmt.Errorf("validate last-known full-demo: negative contact count")
		}
		contactTotal += count
	}
	if contactTotal != coverage.RowCount ||
		report.Summary.CarrySources["negative_contact"]+report.Summary.CarrySources["unavailable_contact"] != coverage.Carries {
		return fmt.Errorf("validate last-known full-demo: contact/carry summary mismatch")
	}
	age := report.Summary.AgeDistribution
	if age.ObservedCount != coverage.ContactUpdates+coverage.Carries || age.ZeroCount < 0 || age.PositiveCount < 0 ||
		age.ZeroCount+age.PositiveCount != age.ObservedCount || age.MinimumMS == nil || age.P50MS == nil ||
		age.P95MS == nil || age.MaximumMS == nil || *age.MinimumMS < 0 || *age.MinimumMS > *age.P50MS ||
		*age.P50MS > *age.P95MS || *age.P95MS > *age.MaximumMS {
		return fmt.Errorf("validate last-known full-demo: age distribution mismatch")
	}
	bucketTotal := 0
	for _, count := range age.Buckets {
		if count < 0 {
			return fmt.Errorf("validate last-known full-demo: negative age bucket")
		}
		bucketTotal += count
	}
	if bucketTotal != age.ObservedCount {
		return fmt.Errorf("validate last-known full-demo: age buckets mismatch")
	}
	return nil
}
