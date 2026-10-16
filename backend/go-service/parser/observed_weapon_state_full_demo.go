package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
)

const (
	ObservedWeaponStateFullDemoSchemaID       = "stratai.observed_weapon_state_full_demo@1"
	ObservedWeaponStateFullDemoReportFilename = "observed_weapon_state_full_demo.json"
	ObservedWeaponStateFullDemoRowsFilename   = "observed_weapon_state_rows.jsonl.gz"
	ObservedWeaponStateRowsDigestContract     = "sha256_compressed_and_decompressed_jsonl_v1"
)

type ObservedWeaponStateFullDemoRowsArtifact struct {
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

type ObservedWeaponStateFullDemoCoverage struct {
	RoundsEligible    []int `json:"rounds_eligible"`
	RoundNumbers      []int `json:"round_numbers"`
	TargetSampleCount int   `json:"target_sample_count"`
	SampleCount       int   `json:"sample_count"`
	PlayerCount       int   `json:"player_count"`
}

type ObservedWeaponStateFactStatusCounts struct {
	Observed    int `json:"observed"`
	Unavailable int `json:"unavailable"`
}

type ObservedWeaponStateAmmoValueCounts struct {
	Zero     int `json:"zero"`
	Positive int `json:"positive"`
}

type ObservedWeaponStateReloadValueCounts struct {
	True  int `json:"true"`
	False int `json:"false"`
}

type ObservedWeaponStateFullDemoSummary struct {
	FactStatus           map[string]ObservedWeaponStateFactStatusCounts `json:"fact_status"`
	AmmoInMagazineValues ObservedWeaponStateAmmoValueCounts             `json:"ammo_in_magazine_values"`
	AmmoReserveValues    ObservedWeaponStateAmmoValueCounts             `json:"ammo_reserve_values"`
	ReloadValues         ObservedWeaponStateReloadValueCounts           `json:"reload_values"`
	ActiveWeapons        map[string]int                                 `json:"active_weapons"`
	NonAmmunitionRows    int                                            `json:"non_ammunition_rows"`
	AmbiguousRows        int                                            `json:"ambiguous_rows"`
	AbstentionReasons    map[string]int                                 `json:"abstention_reasons"`
	Reconciliations      map[string]map[string]int                      `json:"reconciliations"`
}

type ObservedWeaponStateFullDemoReport struct {
	SchemaID          string                                  `json:"schema_id"`
	ContractVersion   string                                  `json:"contract_version"`
	CandidateSchemaID string                                  `json:"candidate_schema_id"`
	SourceAlias       string                                  `json:"source_alias"`
	SourceSHA256      string                                  `json:"source_sha256"`
	BuildIdentifier   string                                  `json:"build_identifier"`
	Sampling          models.ObservedWeaponSampling           `json:"sampling"`
	RowsArtifact      ObservedWeaponStateFullDemoRowsArtifact `json:"rows_artifact"`
	Coverage          ObservedWeaponStateFullDemoCoverage     `json:"coverage"`
	Summary           ObservedWeaponStateFullDemoSummary      `json:"summary"`
}

type observedWeaponLogicalCounter struct {
	bytes int64
}

type observedWeaponExpectedCoverage struct {
	roundNumbers []int
	samples      map[string]struct{}
}

func (counter *observedWeaponLogicalCounter) Write(payload []byte) (int, error) {
	counter.bytes += int64(len(payload))
	return len(payload), nil
}

// BuildObservedWeaponStateFullDemoReport emits every candidate row as logical
// JSONL. Physical gzip metadata is completed atomically by the CLI.
func BuildObservedWeaponStateFullDemoReport(
	replay *models.ReplayData,
	events []combat.Event,
	sourceAlias string,
	sourceSHA256 string,
	rowsWriter io.Writer,
) (ObservedWeaponStateFullDemoReport, error) {
	if replay == nil {
		return ObservedWeaponStateFullDemoReport{}, fmt.Errorf("build weapon full-demo report: replay data is nil")
	}
	if rowsWriter == nil {
		return ObservedWeaponStateFullDemoReport{}, fmt.Errorf("build weapon full-demo report: rows writer is nil")
	}
	sourceAlias = strings.TrimSpace(sourceAlias)
	if sourceAlias == "" {
		return ObservedWeaponStateFullDemoReport{}, fmt.Errorf("build weapon full-demo report: source alias is empty")
	}
	if !observedWeaponSHA256Pattern.MatchString(sourceSHA256) {
		return ObservedWeaponStateFullDemoReport{}, fmt.Errorf("build weapon full-demo report: source SHA-256 is invalid")
	}

	cutoffTick, ok := observedWeaponFullDemoCutoff(replay.Rounds)
	if !ok {
		return ObservedWeaponStateFullDemoReport{}, fmt.Errorf("build weapon full-demo report: no complete rounds")
	}
	replayCopy := *replay
	replayCopy.Metadata = replay.Metadata
	replayCopy.Metadata.MatchID = sourceAlias
	ledger, err := BuildObservedWeaponStateLedger(&replayCopy, events, cutoffTick)
	if err != nil {
		return ObservedWeaponStateFullDemoReport{}, err
	}
	if len(ledger.Rows) == 0 {
		return ObservedWeaponStateFullDemoReport{}, fmt.Errorf("build weapon full-demo report: candidate produced no rows")
	}
	expectedCoverage, err := expectedObservedWeaponFullDemoCoverage(&replayCopy, cutoffTick)
	if err != nil {
		return ObservedWeaponStateFullDemoReport{}, err
	}
	coverage, err := validateObservedWeaponFullDemoCoverage(expectedCoverage, ledger.Rows)
	if err != nil {
		return ObservedWeaponStateFullDemoReport{}, err
	}

	logicalHash := sha256.New()
	logicalCounter := &observedWeaponLogicalCounter{}
	encoder := json.NewEncoder(io.MultiWriter(rowsWriter, logicalHash, logicalCounter))
	encoder.SetEscapeHTML(false)
	for index := range ledger.Rows {
		if err := encoder.Encode(ledger.Rows[index]); err != nil {
			return ObservedWeaponStateFullDemoReport{}, fmt.Errorf("write weapon row %d: %w", index, err)
		}
	}

	report := ObservedWeaponStateFullDemoReport{
		SchemaID:          ObservedWeaponStateFullDemoSchemaID,
		ContractVersion:   models.ObservedWeaponStateContractVersion,
		CandidateSchemaID: models.ObservedWeaponStateSchemaID,
		SourceAlias:       sourceAlias,
		SourceSHA256:      sourceSHA256,
		Sampling:          ledger.Sampling,
		RowsArtifact: ObservedWeaponStateFullDemoRowsArtifact{
			Path:           ObservedWeaponStateFullDemoRowsFilename,
			Format:         "jsonl",
			Compression:    "gzip",
			RecordCount:    len(ledger.Rows),
			LogicalBytes:   logicalCounter.bytes,
			LogicalSHA256:  hex.EncodeToString(logicalHash.Sum(nil)),
			DigestContract: ObservedWeaponStateRowsDigestContract,
		},
		Coverage: coverage,
		Summary:  summarizeObservedWeaponRows(ledger.Rows),
	}
	if err := validateObservedWeaponStateFullDemoReport(report, false); err != nil {
		return ObservedWeaponStateFullDemoReport{}, err
	}
	return report, nil
}

func ValidateObservedWeaponStateFullDemoReport(report ObservedWeaponStateFullDemoReport) error {
	return validateObservedWeaponStateFullDemoReport(report, true)
}

func expectedObservedWeaponFullDemoCoverage(
	replay *models.ReplayData,
	cutoffTick int,
) (observedWeaponExpectedCoverage, error) {
	result := observedWeaponExpectedCoverage{samples: make(map[string]struct{})}
	if replay == nil {
		return result, fmt.Errorf("expected weapon coverage: replay is nil")
	}
	tickRate := replay.Metadata.TickRate
	if tickRate < models.TacticalTargetHz {
		return result, fmt.Errorf("expected weapon coverage: invalid tick rate")
	}
	periodTicks := tickRate / float64(models.TacticalTargetHz)
	roundSet := make(map[int]struct{})
	for _, round := range replay.Rounds {
		if round.Round <= 0 || round.EndTick < round.StartTick {
			continue
		}
		frames, err := canonicalObservedWeaponFrames(round)
		if err != nil {
			return result, err
		}
		frameIndex := 0
		previousTarget := round.StartTick
		for targetIndex, targetTick := range tacticalTargetTicks(round.StartTick, round.EndTick, periodTicks) {
			if targetTick > cutoffTick {
				break
			}
			var selected *observedWeaponFrameSample
			for frameIndex < len(frames) && frames[frameIndex].tick <= targetTick {
				candidate := &frames[frameIndex]
				if targetIndex == 0 || candidate.tick > previousTarget {
					selected = candidate
				}
				frameIndex++
			}
			if selected != nil && len(selected.players) > 0 {
				result.samples[fmt.Sprintf("%d:%d", round.Round, targetTick)] = struct{}{}
				roundSet[round.Round] = struct{}{}
			}
			previousTarget = targetTick
		}
	}
	for round := range roundSet {
		result.roundNumbers = append(result.roundNumbers, round)
	}
	sort.Ints(result.roundNumbers)
	return result, nil
}

func validateObservedWeaponFullDemoCoverage(
	expected observedWeaponExpectedCoverage,
	rows []models.ObservedWeaponStateRow,
) (ObservedWeaponStateFullDemoCoverage, error) {
	rounds := make(map[int]struct{})
	samples := make(map[string]struct{})
	players := make(map[string]struct{})
	for _, row := range rows {
		rounds[row.RoundNumber] = struct{}{}
		samples[fmt.Sprintf("%d:%d", row.RoundNumber, row.Tick)] = struct{}{}
		players[row.JoinKeys.PlayerID] = struct{}{}
	}
	roundNumbers := make([]int, 0, len(rounds))
	for round := range rounds {
		roundNumbers = append(roundNumbers, round)
	}
	sort.Ints(roundNumbers)
	if !equalObservedWeaponIntSlices(expected.roundNumbers, roundNumbers) {
		return ObservedWeaponStateFullDemoCoverage{}, fmt.Errorf(
			"weapon full-demo coverage: emitted rounds %v, expected %v",
			roundNumbers,
			expected.roundNumbers,
		)
	}
	if len(samples) != len(expected.samples) {
		return ObservedWeaponStateFullDemoCoverage{}, fmt.Errorf(
			"weapon full-demo coverage: emitted %d samples, expected %d",
			len(samples),
			len(expected.samples),
		)
	}
	for sample := range expected.samples {
		if _, exists := samples[sample]; !exists {
			return ObservedWeaponStateFullDemoCoverage{}, fmt.Errorf("weapon full-demo coverage: missing sample %s", sample)
		}
	}
	return ObservedWeaponStateFullDemoCoverage{
		RoundsEligible:    append([]int(nil), expected.roundNumbers...),
		RoundNumbers:      roundNumbers,
		TargetSampleCount: len(expected.samples),
		SampleCount:       len(samples),
		PlayerCount:       len(players),
	}, nil
}

func equalObservedWeaponIntSlices(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func summarizeObservedWeaponRows(rows []models.ObservedWeaponStateRow) ObservedWeaponStateFullDemoSummary {
	summary := newObservedWeaponStateFullDemoSummary()
	for _, row := range rows {
		addObservedWeaponFactStatus(summary.FactStatus, "active_weapon", row.ActiveWeapon.Status)
		addObservedWeaponFactStatus(summary.FactStatus, "ammo_in_magazine", row.AmmoInMagazine.Status)
		addObservedWeaponFactStatus(summary.FactStatus, "ammo_reserve", row.AmmoReserve.Status)
		addObservedWeaponFactStatus(summary.FactStatus, "is_reloading", row.IsReloading.Status)
		addObservedWeaponReconciliation(summary.Reconciliations, "active_weapon", row.ActiveWeapon.Provenance.Reconciliation)
		addObservedWeaponReconciliation(summary.Reconciliations, "ammo_in_magazine", row.AmmoInMagazine.Provenance.Reconciliation)
		addObservedWeaponReconciliation(summary.Reconciliations, "ammo_reserve", row.AmmoReserve.Provenance.Reconciliation)
		addObservedWeaponReconciliation(summary.Reconciliations, "is_reloading", row.IsReloading.Provenance.Reconciliation)

		if row.ActiveWeapon.Status == models.ObservedWeaponFactObserved && row.ActiveWeapon.Value != nil {
			summary.ActiveWeapons[*row.ActiveWeapon.Value]++
		}
		countObservedWeaponAmmoValue(row.AmmoInMagazine, &summary.AmmoInMagazineValues)
		countObservedWeaponAmmoValue(row.AmmoReserve, &summary.AmmoReserveValues)
		if row.IsReloading.Status == models.ObservedWeaponFactObserved && row.IsReloading.Value != nil {
			if *row.IsReloading.Value {
				summary.ReloadValues.True++
			} else {
				summary.ReloadValues.False++
			}
		}

		ambiguous := false
		for _, reconciliation := range []string{
			row.ActiveWeapon.Provenance.Reconciliation,
			row.AmmoInMagazine.Provenance.Reconciliation,
			row.AmmoReserve.Provenance.Reconciliation,
			row.IsReloading.Provenance.Reconciliation,
		} {
			ambiguous = ambiguous || reconciliation == models.ObservedWeaponReconciliationAmbiguous
		}
		for _, reason := range row.AbstentionReasons {
			summary.AbstentionReasons[reason]++
			if reason == weaponAbstentionNonAmmunition {
				summary.NonAmmunitionRows++
			}
		}
		if ambiguous {
			summary.AmbiguousRows++
		}
	}
	return summary
}

func newObservedWeaponStateFullDemoSummary() ObservedWeaponStateFullDemoSummary {
	facts := []string{"active_weapon", "ammo_in_magazine", "ammo_reserve", "is_reloading"}
	reconciliations := []string{
		models.ObservedWeaponReconciliationNotApplicable,
		models.ObservedWeaponReconciliationFire,
		models.ObservedWeaponReconciliationReload,
		models.ObservedWeaponReconciliationAmbiguous,
		models.ObservedWeaponReconciliationUnavailable,
	}
	summary := ObservedWeaponStateFullDemoSummary{
		FactStatus:        make(map[string]ObservedWeaponStateFactStatusCounts, len(facts)),
		ActiveWeapons:     make(map[string]int),
		AbstentionReasons: make(map[string]int),
		Reconciliations:   make(map[string]map[string]int, len(facts)),
	}
	for _, fact := range facts {
		summary.FactStatus[fact] = ObservedWeaponStateFactStatusCounts{}
		summary.Reconciliations[fact] = make(map[string]int, len(reconciliations))
		for _, reconciliation := range reconciliations {
			summary.Reconciliations[fact][reconciliation] = 0
		}
	}
	return summary
}

func addObservedWeaponFactStatus(
	counts map[string]ObservedWeaponStateFactStatusCounts,
	fact string,
	status string,
) {
	value := counts[fact]
	if status == models.ObservedWeaponFactObserved {
		value.Observed++
	} else {
		value.Unavailable++
	}
	counts[fact] = value
}

func addObservedWeaponReconciliation(counts map[string]map[string]int, fact, reconciliation string) {
	counts[fact][reconciliation]++
}

func countObservedWeaponAmmoValue(
	fact models.ObservedWeaponIntFact,
	counts *ObservedWeaponStateAmmoValueCounts,
) {
	if fact.Status != models.ObservedWeaponFactObserved || fact.Value == nil {
		return
	}
	if *fact.Value == 0 {
		counts.Zero++
	} else if *fact.Value > 0 {
		counts.Positive++
	}
}

func observedWeaponFullDemoCutoff(rounds []models.ReplayRound) (int, bool) {
	cutoff := 0
	found := false
	for _, round := range rounds {
		if round.Round <= 0 || round.EndTick < round.StartTick {
			continue
		}
		if !found || round.EndTick > cutoff {
			cutoff = round.EndTick
			found = true
		}
	}
	return cutoff, found
}

var observedWeaponSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validateObservedWeaponStateFullDemoReport(
	report ObservedWeaponStateFullDemoReport,
	requirePhysical bool,
) error {
	if report.SchemaID != ObservedWeaponStateFullDemoSchemaID ||
		report.ContractVersion != models.ObservedWeaponStateContractVersion ||
		report.CandidateSchemaID != models.ObservedWeaponStateSchemaID {
		return fmt.Errorf("validate weapon full-demo report: contract metadata mismatch")
	}
	if strings.TrimSpace(report.SourceAlias) == "" || !observedWeaponSHA256Pattern.MatchString(report.SourceSHA256) {
		return fmt.Errorf("validate weapon full-demo report: invalid source provenance")
	}
	if report.Sampling.TargetHz != models.TacticalTargetHz || report.Sampling.Strategy != models.TacticalSamplingStrategy {
		return fmt.Errorf("validate weapon full-demo report: sampling mismatch")
	}
	artifact := report.RowsArtifact
	if artifact.Path != ObservedWeaponStateFullDemoRowsFilename || artifact.Format != "jsonl" ||
		artifact.Compression != "gzip" || artifact.RecordCount <= 0 || artifact.LogicalBytes <= 0 ||
		!observedWeaponSHA256Pattern.MatchString(artifact.LogicalSHA256) ||
		artifact.DigestContract != ObservedWeaponStateRowsDigestContract {
		return fmt.Errorf("validate weapon full-demo report: invalid logical rows artifact")
	}
	if requirePhysical && (strings.TrimSpace(report.BuildIdentifier) == "" || artifact.Bytes <= 0 ||
		!observedWeaponSHA256Pattern.MatchString(artifact.SHA256)) {
		return fmt.Errorf("validate weapon full-demo report: invalid physical rows artifact")
	}
	if len(report.Coverage.RoundsEligible) == 0 || !sort.IntsAreSorted(report.Coverage.RoundsEligible) ||
		!equalObservedWeaponIntSlices(report.Coverage.RoundsEligible, report.Coverage.RoundNumbers) ||
		report.Coverage.TargetSampleCount <= 0 ||
		report.Coverage.TargetSampleCount != report.Coverage.SampleCount || report.Coverage.PlayerCount <= 0 {
		return fmt.Errorf("validate weapon full-demo report: invalid coverage")
	}
	for index, round := range report.Coverage.RoundsEligible {
		if round <= 0 || (index > 0 && round == report.Coverage.RoundNumbers[index-1]) {
			return fmt.Errorf("validate weapon full-demo report: invalid round coverage")
		}
	}
	for _, fact := range []string{"active_weapon", "ammo_in_magazine", "ammo_reserve", "is_reloading"} {
		status, ok := report.Summary.FactStatus[fact]
		if !ok || status.Observed < 0 || status.Unavailable < 0 ||
			status.Observed+status.Unavailable != artifact.RecordCount {
			return fmt.Errorf("validate weapon full-demo report: invalid status summary for %s", fact)
		}
		reconciliations, ok := report.Summary.Reconciliations[fact]
		if !ok {
			return fmt.Errorf("validate weapon full-demo report: missing reconciliation summary for %s", fact)
		}
		total := 0
		for _, key := range []string{
			models.ObservedWeaponReconciliationNotApplicable,
			models.ObservedWeaponReconciliationFire,
			models.ObservedWeaponReconciliationReload,
			models.ObservedWeaponReconciliationAmbiguous,
			models.ObservedWeaponReconciliationUnavailable,
		} {
			value, exists := reconciliations[key]
			if !exists || value < 0 {
				return fmt.Errorf("validate weapon full-demo report: invalid reconciliation %s.%s", fact, key)
			}
			total += value
		}
		if total != artifact.RecordCount {
			return fmt.Errorf("validate weapon full-demo report: reconciliation total mismatch for %s", fact)
		}
	}
	return nil
}
