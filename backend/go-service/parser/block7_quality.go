package parser

import (
	"cs2-demo-service/models"
	mapassets "cs2-demo-service/pkg/maps"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type mapInputProvenanceProvider interface {
	InputProvenance() map[string]mapassets.InputProvenance
}

const (
	block7ValidatorVersion       = "stratai.canonical_validator@2"
	block7GoldenCorpusVersion    = "stratai.golden_demo_corpus@2"
	block7GoldenCorpusManifestID = "golden-demos-v2"
	block7TickRateRulesVersion   = "stratai.tick_rate_rules@1"
)

var block7RequiredDomains = []string{
	"bundle_manifest_contract",
	"artifact_catalog_integrity",
	"artifact_hash_integrity",
	"artifact_record_count",
	"cross_artifact_references",
	"roster_consistency",
	"round_consistency",
	"objective_consistency",
	"utility_consistency",
	"combat_consistency",
	"engagement_consistency",
	"economy_consistency",
	"player_state_consistency",
	"replay_projection_consistency",
	"causal_availability",
	"future_leakage",
	"schema_version_compatibility",
	"determinism",
	"lineage_completeness",
	"corpus_quality",
}

type qualityDomain struct {
	Name               string   `json:"name"`
	Status             string   `json:"status"`
	Severity           string   `json:"severity"`
	Expected           string   `json:"expected"`
	Actual             string   `json:"actual"`
	Coverage           float64  `json:"coverage"`
	UnavailableCount   int      `json:"unavailable_count"`
	InferredCount      int      `json:"inferred_count"`
	WarningDetails     []string `json:"warning_details"`
	HardFailureDetails []string `json:"hard_failure_details"`
	SourceArtifacts    []string `json:"source_artifacts"`
	SchemaVersions     []string `json:"schema_versions"`
}

func buildCanonicalLineage(
	ctx *models.DemoContext,
	tickRate float64,
	provenance models.CanonicalExportProvenance,
	metadata models.CanonicalMatchMetadata,
	report qualityReport,
) models.CanonicalLineage {
	buildIdentifier := unavailableLineageValue("build identifier was not supplied")
	if value := strings.TrimSpace(provenance.BuildIdentifier); value != "" {
		buildIdentifier = observedLineageValue(value, "STRATAI_BUILD_ID")
	}
	algorithmVersions := make(map[string]string, len(metadata.Algorithms)+4)
	for name, value := range metadata.Algorithms {
		algorithmVersions[name] = value.Version
	}
	algorithmVersions["engagement"] = "engagement_causal@2"
	algorithmVersions["trade"] = "trade_response@2"
	algorithmVersions["causal_partition"] = "stratai.causal_partition@1"
	algorithmVersions["quality_gate"] = "stratai.quality_gate@1"
	qualityFlags := make([]string, 0)
	for _, check := range report.Checks {
		if check.Status != "pass" {
			qualityFlags = append(qualityFlags, check.Name+":"+check.Status)
		}
	}
	sort.Strings(qualityFlags)
	warnings := append([]string(nil), report.Warnings...)
	sort.Strings(warnings)
	inputHashes, inputAbstentions := canonicalMapInputHashes(ctx.MapManager)
	return models.CanonicalLineage{
		DemoChecksumSHA256: provenance.DemoChecksum,
		ParserVersion:      block6ParserVersion, DemoinfocsVersion: report.DemoinfocsVersion,
		ExportFormatVersion: canonicalExportFormatVersion, BuildIdentifier: buildIdentifier,
		MapName: ctx.MatchData.MapName, TickRateHz: tickRate,
		TickRateRulesVersion: block7TickRateRulesVersion,
		PriceTableVersion:    metadata.PriceTable.Version, PriceTableSHA256: metadata.PriceTable.ChecksumSHA256,
		AlgorithmVersions: algorithmVersions, SchemaVersions: map[string]string{},
		ValidatorVersion: block7ValidatorVersion,
		ProcessingTimestamp: models.CanonicalLineageValue{
			Value: nil, Status: "operational_only", Source: nil,
		},
		MetadataSource:      metadata.Source,
		InputHashes:         inputHashes,
		ConfigurationHashes: metadata.ConfigurationHashes,
		QualityFlags:        qualityFlags, Warnings: warnings,
		GoldenCorpusVersion:    block7GoldenCorpusVersion,
		GoldenCorpusManifestID: block7GoldenCorpusManifestID,
		Abstentions: append([]string{
			"processing timestamp excluded from deterministic canonical tree",
		}, inputAbstentions...),
	}
}

func canonicalMapInputHashes(checker interface{}) (map[string]models.CanonicalLineageValue, []string) {
	result := make(map[string]models.CanonicalLineageValue, 3)
	abstentions := make([]string, 0, 3)
	provider, ok := checker.(mapInputProvenanceProvider)
	inputs := map[string]mapassets.InputProvenance{}
	if ok {
		inputs = provider.InputProvenance()
	}
	for _, name := range []string{"physics_map", "nav_mesh", "callouts"} {
		input, found := inputs[name]
		if found && input.SHA256 != "" {
			source := "map_manager:" + input.RelativePath
			if !input.Used {
				source += "; inspected_not_used"
			}
			result[name] = observedLineageValue(input.SHA256, source)
			continue
		}
		reason := name + " was not available to the map loader"
		if found && input.LoadError != "" {
			reason = name + " could not be hashed: " + input.LoadError
		}
		result[name] = unavailableLineageValue(reason)
		abstentions = append(abstentions, reason)
	}
	return result, abstentions
}

func observedLineageValue(value, source string) models.CanonicalLineageValue {
	valueCopy := value
	sourceCopy := source
	return models.CanonicalLineageValue{Value: &valueCopy, Status: "observed", Source: &sourceCopy}
}

func unavailableLineageValue(reason string) models.CanonicalLineageValue {
	reasonCopy := reason
	return models.CanonicalLineageValue{Value: nil, Status: "unavailable", Source: &reasonCopy}
}

func finalizeCanonicalBlock7Quality(matchDir string) error {
	canonicalDir := filepath.Join(matchDir, "canonical")
	manifestPath := filepath.Join(canonicalDir, "manifest.json")
	qualityPath := filepath.Join(canonicalDir, "diagnostics", "quality_report.json")
	var manifest models.CanonicalManifest
	if err := decodeCanonicalJSON(manifestPath, &manifest); err != nil {
		return fmt.Errorf("finalize block 7 manifest: %w", err)
	}
	var qualityExport models.CanonicalQualityReportExport
	if err := decodeCanonicalJSON(qualityPath, &qualityExport); err != nil {
		return fmt.Errorf("finalize block 7 quality: %w", err)
	}
	report, err := canonicalQualityReport(qualityExport.Report)
	if err != nil {
		return err
	}
	manifest.Lineage.SchemaVersions = block7SchemaVersions(manifest.Artifacts)
	artifactViolations, artifactFailures := verifyCanonicalArtifactIntegrity(canonicalDir, manifest.Artifacts)
	report.Block7ArtifactIntegrityViolations = artifactViolations
	report.Domains = buildBlock7Domains(report, manifest, artifactFailures)
	if report.hasHardBlock7Failure() {
		return fmt.Errorf("canonical block 7 quality gates failed: %s", strings.Join(report.block7FailureDetails(), "; "))
	}
	qualityExport.Report = report
	if err := writeJSON(qualityPath, qualityExport); err != nil {
		return fmt.Errorf("rewrite block 7 quality report: %w", err)
	}
	if err := refreshCanonicalArtifact(&manifest, canonicalDir, "diagnostics/quality_report.json"); err != nil {
		return err
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		return fmt.Errorf("rewrite canonical manifest with block 7 lineage: %w", err)
	}
	return nil
}

func decodeCanonicalJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}

func block7SchemaVersions(artifacts []models.CanonicalArtifact) map[string]string {
	versions := make(map[string]string, len(artifacts)+1)
	versions["canonical_manifest"] = "stratai.canonical_manifest@3"
	for _, artifact := range artifacts {
		versions[artifact.ArtifactType] = artifact.SchemaID
	}
	return versions
}

func verifyCanonicalArtifactIntegrity(
	canonicalDir string,
	artifacts []models.CanonicalArtifact,
) (int, []string) {
	violations := 0
	failures := make([]string, 0)
	previousPath := ""
	seen := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Path == "" || filepath.IsAbs(artifact.Path) || strings.Contains(artifact.Path, "\\") || strings.Contains(artifact.Path, "..") {
			violations++
			failures = append(failures, fmt.Sprintf("unsafe artifact path %q", artifact.Path))
			continue
		}
		if artifact.Path < previousPath {
			violations++
			failures = append(failures, "artifact catalog is not ordered by path")
		}
		previousPath = artifact.Path
		if _, exists := seen[artifact.Path]; exists {
			violations++
			failures = append(failures, "duplicate artifact path: "+artifact.Path)
		}
		seen[artifact.Path] = struct{}{}
		path := filepath.Join(canonicalDir, filepath.FromSlash(artifact.Path))
		checksum, err := checksumFile(path)
		if err != nil {
			violations++
			failures = append(failures, fmt.Sprintf("cannot hash %s: %v", artifact.Path, err))
			continue
		}
		info, err := os.Stat(path)
		if err != nil || checksum != artifact.SHA256 || info.Size() != artifact.Bytes || artifact.RecordCount < 0 {
			violations++
			failures = append(failures, "artifact integrity mismatch: "+artifact.Path)
		}
	}
	return violations, failures
}

func refreshCanonicalArtifact(
	manifest *models.CanonicalManifest,
	canonicalDir, relativePath string,
) error {
	for index := range manifest.Artifacts {
		if manifest.Artifacts[index].Path != relativePath {
			continue
		}
		path := filepath.Join(canonicalDir, filepath.FromSlash(relativePath))
		checksum, err := checksumFile(path)
		if err != nil {
			return fmt.Errorf("refresh %s checksum: %w", relativePath, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("refresh %s size: %w", relativePath, err)
		}
		manifest.Artifacts[index].SHA256 = checksum
		manifest.Artifacts[index].Bytes = info.Size()
		return nil
	}
	return fmt.Errorf("canonical manifest does not contain %s", relativePath)
}

func (report qualityReport) hasHardBlock7Failure() bool {
	for _, domain := range report.Domains {
		if domain.Severity == "hard" && domain.Status == "fail" {
			return true
		}
	}
	return false
}

func (report qualityReport) block7FailureDetails() []string {
	details := make([]string, 0)
	for _, domain := range report.Domains {
		if domain.Severity == "hard" && domain.Status == "fail" {
			if len(domain.HardFailureDetails) == 0 {
				details = append(details, domain.Name+" failed")
			} else {
				for _, detail := range domain.HardFailureDetails {
					details = append(details, domain.Name+": "+detail)
				}
			}
		}
	}
	return details
}

func buildBlock7Domains(
	report qualityReport,
	manifest models.CanonicalManifest,
	artifactFailures []string,
) []qualityDomain {
	domains := make([]qualityDomain, 0, len(block7RequiredDomains))
	addHard := func(name string, violations int, sources, schemas []string) {
		domains = append(domains, block7HardDomain(name, violations, sources, schemas, nil))
	}
	addHard("bundle_manifest_contract", report.Block7ArtifactIntegrityViolations, []string{"canonical/manifest.json"}, []string{"stratai.canonical_manifest@3"})
	addHard("artifact_catalog_integrity", report.Block7ArtifactIntegrityViolations, []string{"canonical/manifest.json"}, []string{"stratai.canonical_manifest@3"})
	domains = append(domains, block7HardDomain("artifact_hash_integrity", report.Block7ArtifactIntegrityViolations, []string{"canonical/manifest.json"}, []string{"sha256"}, artifactFailures))
	addHard("artifact_record_count", report.Block7ArtifactIntegrityViolations, []string{"canonical/manifest.json"}, []string{"stratai.canonical_manifest@3"})
	crossReferences := report.ObjectiveCarrierMismatches + report.ObjectiveReplayEventMismatches + report.UtilityThrowReconciliationMismatches + report.CombatReplayProjectionMismatches + report.EngagementAtomicProvenanceViolations
	addHard("cross_artifact_references", crossReferences, []string{"canonical/core", "canonical/events", "canonical/derived", "canonical/presentation"}, block7Schemas(manifest, "rounds", "combat_events", "utility_events", "objective_events", "engagements"))
	addHard("roster_consistency", report.EngagementParticipantMismatches+report.EconomyTeamIdentityViolations, []string{"canonical/core/participants.json", "canonical/core/rounds.json"}, block7Schemas(manifest, "participants", "rounds"))
	roundViolations := report.ReplayRoundSequenceErrors + report.EconomyRoundSequenceErrors + report.TrackingRoundSequenceErrors + report.CombatRoundSequenceErrors + report.GrenadeRoundSequenceErrors
	addHard("round_consistency", roundViolations, []string{"canonical/core/rounds.json"}, block7Schemas(manifest, "rounds"))
	objectiveViolations := report.ObjectiveContractViolations + report.ObjectiveRoundMismatches + report.ObjectiveTerminalMismatches + report.ObjectiveLifecycleViolations + report.ObjectiveCarrierMismatches + report.ObjectiveReplayEventMismatches
	objectiveUnavailable := report.ObjectiveMissingActors + report.ObjectiveMissingPositions + report.ObjectiveAttemptsMissingStart + report.ObjectiveNativeRoleDisagreements
	domains = append(domains, block7CoverageDomain("objective_consistency", objectiveViolations, report.ObjectiveEvents+report.TrackingEvents, objectiveUnavailable, 0, []string{"canonical/events/objective_events.jsonl", "canonical/core/rounds.json", "canonical/states/player_states"}, block7Schemas(manifest, "objective_events", "rounds", "player_states")))
	utilityViolations := report.UtilityContractViolations + report.UtilityThrowReconciliationMismatches + report.UtilityPlayerStatsMismatches + report.UtilityReplayProjectionMismatches + report.UtilityLifecycleViolations + report.UtilityFlashAttributionMismatches + report.UtilityDamageReconciliationMismatches + report.UtilityTemporalSpatialViolations + report.UtilityDeterminismViolations
	utilityObserved := report.UtilityCanonicalEvents + report.UtilityFlashEffects + report.UtilityDamageEffects
	domains = append(domains, block7CoverageDomain("utility_consistency", utilityViolations, utilityObserved, report.UtilityObservationWarnings, report.UtilityInferredCorrelations, []string{"canonical/events/utility_events.jsonl"}, block7Schemas(manifest, "utility_events")))
	combatViolations := report.CombatContractViolations + report.CombatCallbackAccountingViolations + report.CombatPlayerStatsMismatches + report.CombatReplayProjectionMismatches + report.CombatNativeDeltaMismatches + report.CombatDeterminismViolations
	domains = append(domains, block7CoverageDomain("combat_consistency", combatViolations, report.CombatLedgerEvents, report.CombatMissingImpactPositions+report.CombatMissingReloadEnds+report.CombatUnavailableShotResults, 0, []string{"canonical/events/combat_events.jsonl"}, block7Schemas(manifest, "combat_events")))
	engagementViolations := report.EngagementEventContractViolations + report.EngagementAtomicProvenanceViolations + report.EngagementParticipantMismatches + report.EngagementRoleConsistencyViolations + report.EngagementTemporalConsistencyErrors + report.EngagementCausalAvailabilityErrors + report.EngagementTradeReconciliationErrors + report.EngagementStatsReconciliationErrors + report.EngagementDeterminismViolations
	domains = append(domains, block7CoverageDomain("engagement_consistency", engagementViolations, report.Engagements, report.EngagementObservationWarnings, 0, []string{"canonical/derived/engagements.json", "canonical/derived/trades.json"}, block7Schemas(manifest, "engagements", "trades")))
	economyViolations := report.EconomyTeamIdentityViolations + report.EconomyNativeCalculatedViolations + report.EconomyMoneyTransitionViolations + report.EconomyPurchaseProvenanceViolations + report.EconomyPriceTableVersionViolations + report.StatsScoreboardReconciliationErrors + report.StatsUtilityReconciliationErrors + report.ClutchAttemptReconciliationErrors + report.WarmupContaminationViolations
	economyObserved := report.EconomyRounds + report.PlayerSummaries
	domains = append(domains, block7CoverageDomain("economy_consistency", economyViolations, economyObserved, report.EconomyObservationWarnings, 0, []string{"canonical/derived/economy_rounds.json", "canonical/derived/economy_players.json"}, block7Schemas(manifest, "economy_rounds", "economy_players")))
	playerStateUnavailable := report.TrackingAliveStates - report.TrackingVelocityObserved
	if playerStateUnavailable < 0 {
		playerStateUnavailable = 0
	}
	domains = append(domains, block7CoverageDomain("player_state_consistency", report.TrackingStateContractViolations+report.TrackingMovingDatasetWithoutData, report.TrackingAliveStates, playerStateUnavailable, 0, []string{"canonical/states/player_states"}, block7Schemas(manifest, "player_states")))
	addHard("replay_projection_consistency", report.ReplayFramesOutsideBounds+report.CombatReplayProjectionMismatches+report.UtilityReplayProjectionMismatches+report.ObjectiveReplayEventMismatches, []string{"canonical/presentation/replay"}, block7Schemas(manifest, "replay_index", "replay_round"))
	domains = append(domains, block7CoverageDomain("causal_availability", report.Block7CausalAvailabilityViolations, report.Engagements, report.EngagementObservationWarnings, 0, []string{"canonical/causal/decision_features.jsonl", "canonical/causal/quality_masks.jsonl"}, block7Schemas(manifest, "decision_features", "quality_masks")))
	addHard("future_leakage", report.Block7FutureLeakageViolations, []string{"canonical/causal/decision_features.jsonl", "canonical/causal/oracle_context.jsonl", "canonical/causal/outcomes.jsonl"}, block7Schemas(manifest, "decision_features", "oracle_context", "decision_outcomes"))
	addHard("schema_version_compatibility", report.Block7SchemaCompatibilityViolations, []string{"canonical/manifest.json"}, block7AllSchemas(manifest))
	determinismViolations := report.UtilityDeterminismViolations + report.CombatDeterminismViolations + report.EngagementDeterminismViolations + report.EconomyDeterminismViolations + report.StatsDeterminismViolations + report.Block7DeterminismViolations
	addHard("determinism", determinismViolations, []string{"canonical/manifest.json"}, []string{"stratai.deterministic_serialization@1"})
	lineageUnavailable := block7LineageUnavailableCount(manifest.Lineage)
	lineage := block7CoverageDomain("lineage_completeness", report.MetadataProvenanceViolations+report.MetadataChecksumLineageViolations, 15, lineageUnavailable, 0, []string{"canonical/manifest.json", "canonical/core/match_metadata.json"}, []string{"stratai.canonical_manifest@3", "stratai.match_metadata@1"})
	if lineageUnavailable > 0 && lineage.Status == "pass" {
		lineage.Status = "warning"
		lineage.Severity = "warning"
		lineage.WarningDetails = []string{"unavailable lineage inputs are explicit and were not invented"}
	}
	domains = append(domains, lineage)
	corpus := block7HardDomain("corpus_quality", report.Block7CorpusQualityViolations, []string{"backend/go-service/testdata/golden-demos/v2/manifest.json"}, []string{block7GoldenCorpusVersion}, nil)
	if report.Block7CorpusQualityViolations == 0 {
		corpus.Status = "warning"
		corpus.Severity = "warning"
		corpus.Coverage = 0
		corpus.UnavailableCount = 40
		corpus.WarningDetails = []string{"40-demo source inventory is valid; semantic reprocessing and Gate 1 proof remain not evaluated"}
	}
	domains = append(domains, corpus)
	return domains
}

func block7HardDomain(name string, violations int, sources, schemas, details []string) qualityDomain {
	status := "pass"
	if violations > 0 {
		status = "fail"
	}
	if details == nil {
		details = []string{}
	}
	return qualityDomain{
		Name: name, Status: status, Severity: "hard", Expected: "0 hard violations",
		Actual: fmt.Sprintf("%d hard violations", violations), Coverage: 1,
		UnavailableCount: 0, InferredCount: 0, WarningDetails: []string{},
		HardFailureDetails: details, SourceArtifacts: sources, SchemaVersions: schemas,
	}
}

func block7CoverageDomain(name string, violations, observed, unavailable, inferred int, sources, schemas []string) qualityDomain {
	domain := block7HardDomain(name, violations, sources, schemas, nil)
	total := observed + unavailable + inferred
	if total > 0 {
		domain.Coverage = float64(observed) / float64(total)
	}
	domain.UnavailableCount = unavailable
	domain.InferredCount = inferred
	if violations == 0 && (unavailable > 0 || inferred > 0) {
		domain.Status = "warning"
		domain.Severity = "warning"
		domain.WarningDetails = []string{"coverage gaps are explicit and must be applied through the quality mask"}
	}
	return domain
}

func block7Schemas(manifest models.CanonicalManifest, artifactTypes ...string) []string {
	wanted := make(map[string]struct{}, len(artifactTypes))
	for _, artifactType := range artifactTypes {
		wanted[artifactType] = struct{}{}
	}
	values := make([]string, 0, len(wanted))
	for _, artifact := range manifest.Artifacts {
		if _, ok := wanted[artifact.ArtifactType]; ok {
			values = append(values, artifact.SchemaID)
		}
	}
	sort.Strings(values)
	return block7UniqueStrings(values)
}

func block7AllSchemas(manifest models.CanonicalManifest) []string {
	values := make([]string, 0, len(manifest.Artifacts)+1)
	values = append(values, manifest.SchemaID)
	for _, artifact := range manifest.Artifacts {
		values = append(values, artifact.SchemaID)
	}
	sort.Strings(values)
	return block7UniqueStrings(values)
}

func block7UniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func block7LineageUnavailableCount(lineage models.CanonicalLineage) int {
	count := 0
	if lineage.BuildIdentifier.Status == "unavailable" {
		count++
	}
	if lineage.ProcessingTimestamp.Status != "observed" {
		count++
	}
	for _, value := range lineage.InputHashes {
		if value.Status == "unavailable" {
			count++
		}
	}
	return count
}
