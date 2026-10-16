package parser

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/exitroute"
	"cs2-demo-service/pkg/navigation"
	"github.com/golang/geo/r3"
)

const (
	ObservedExitRouteDiagnosticSchemaID = "stratai.observed_exit_route_diagnostic@2"
	ObservedExitRouteFullDemoSchemaID   = "stratai.observed_exit_route_full_demo_report@2"
	ObservedExitRouteFullDemoFilename   = "observed_exit_route_full_demo.json"
	ObservedExitRouteRowsFilename       = "observed_exit_route_diagnostic.jsonl.gz"
	exitRouteDigestContract             = "sha256_over_compact_jsonl_with_lf@1"
	exitRouteSamplingTargetHz           = 1
)

type ExitRouteSampleFact struct {
	Status      string   `json:"status"`
	ReasonCodes []string `json:"reason_codes"`
	SampleID    *string  `json:"sample_id"`
	SampleAlias *string  `json:"sample_alias"`
}
type ExitRouteObjectiveFact struct {
	FieldID                string                               `json:"field_id"`
	Direction              string                               `json:"direction"`
	ReasonCodes            []string                             `json:"reason_codes"`
	Reachability           models.ObservedNavigationBooleanFact `json:"reachability"`
	OriginAssociation      *float64                             `json:"origin_association_world_units"`
	DestinationAssociation *float64                             `json:"destination_association_world_units"`
}
type ExitRoutePhysicalCost struct {
	NavigationSchemaID        string                                  `json:"navigation_schema_id"`
	NavigationContractVersion string                                  `json:"navigation_contract_version"`
	DirectDistance            models.ObservedNavigationDistanceFact   `json:"direct_distance"`
	PathDistance              models.ObservedNavigationDistanceFact   `json:"path_distance"`
	InternalTiming            models.ObservedNavigationInternalTiming `json:"internal_timing"`
}
type ObservedExitRouteCandidate struct {
	CandidateID    string                  `json:"candidate_id"`
	CandidateAlias string                  `json:"candidate_alias"`
	Destination    exitroute.Vector3       `json:"destination_world"`
	Objective      ExitRouteObjectiveFact  `json:"objective_viability"`
	PhysicalCost   ExitRoutePhysicalCost   `json:"physical_cost"`
	Tactical       SpatialTacticalJudgment `json:"tactical_assessment"`
}
type ExitRouteProvenance struct {
	PositionSource         string   `json:"position_source"`
	PostureSource          string   `json:"posture_source"`
	MotionSource           string   `json:"motion_source"`
	PackageSchemaID        string   `json:"package_schema_id"`
	PackageContractVersion string   `json:"package_contract_version"`
	PackageAssetVersion    *string  `json:"package_asset_version"`
	PackageSHA256          *string  `json:"package_sha256"`
	GeometryRevision       *string  `json:"geometry_revision"`
	NavAssetSHA256         *string  `json:"nav_asset_sha256"`
	InputFields            []string `json:"input_fields"`
}
type ObservedExitRouteRecord struct {
	SchemaID         string                                 `json:"schema_id"`
	ContractVersion  string                                 `json:"contract_version"`
	SourceAlias      string                                 `json:"source_alias"`
	MapID            string                                 `json:"map_id"`
	RoundNumber      int                                    `json:"round_number"`
	T0Tick           int                                    `json:"t0_tick"`
	AvailabilityTick int                                    `json:"availability_tick"`
	ParticipantAlias string                                 `json:"participant_alias"`
	TrainingAllowed  bool                                   `json:"training_allowed"`
	ProductSupported bool                                   `json:"product_supported"`
	Position         *exitroute.Vector3                     `json:"position_world"`
	PostureContext   models.ObservedNavigationMotionContext `json:"posture_context"`
	Sample           ExitRouteSampleFact                    `json:"sample"`
	Candidates       []ObservedExitRouteCandidate           `json:"candidates"`
	Provenance       ExitRouteProvenance                    `json:"provenance"`
}
type ExitRoutePackageResolutionSummary struct {
	Status           string   `json:"status"`
	ReasonCodes      []string `json:"reason_codes"`
	ContractVersion  string   `json:"contract_version"`
	AssetVersion     *string  `json:"asset_version"`
	PackageSHA256    *string  `json:"package_sha256"`
	GeometryRevision *string  `json:"geometry_revision"`
	ProductSupported bool     `json:"product_supported"`
}
type ExitRouteSampling struct {
	TargetHz    int     `json:"target_hz"`
	TickRateHz  float64 `json:"tick_rate_hz"`
	PeriodTicks float64 `json:"period_ticks"`
	Strategy    string  `json:"strategy"`
}
type ExitRouteArtifact struct {
	Path           string `json:"path"`
	Format         string `json:"format"`
	Compression    string `json:"compression"`
	Bytes          int64  `json:"bytes"`
	SHA256         string `json:"sha256"`
	LogicalRecords int    `json:"logical_records"`
	LogicalBytes   int64  `json:"logical_bytes"`
	LogicalSHA256  string `json:"logical_sha256"`
	DigestContract string `json:"digest_contract"`
}
type ExitRouteCoverage struct {
	Rounds                          int            `json:"rounds"`
	Records                         int            `json:"records"`
	SamplesObserved                 int            `json:"samples_observed"`
	SamplesUnavailable              int            `json:"samples_unavailable"`
	CandidateEvaluations            int            `json:"candidate_evaluations"`
	Reachable                       int            `json:"reachable"`
	Unreachable                     int            `json:"unreachable"`
	ReachabilityUnavailable         int            `json:"reachability_unavailable"`
	PhysicalPathAvailable           int            `json:"physical_path_available"`
	PhysicalPathUnavailable         int            `json:"physical_path_unavailable"`
	ActiveWeaponScenarioAvailable   int            `json:"active_weapon_scenario_available"`
	ActiveWeaponScenarioUnavailable int            `json:"active_weapon_scenario_unavailable"`
	ObservedPaceScenarioAvailable   int            `json:"observed_pace_scenario_available"`
	ObservedPaceScenarioUnavailable int            `json:"observed_pace_scenario_unavailable"`
	TacticalUnavailable             int            `json:"tactical_judgment_unavailable"`
	SamplesHit                      map[string]int `json:"samples_hit"`
	SampleAbsenceReasonCounts       map[string]int `json:"sample_absence_reason_counts"`
	ObjectiveReasonCounts           map[string]int `json:"objective_unavailability_reason_counts"`
	TacticalReasonCounts            map[string]int `json:"tactical_judgment_reason_counts"`
}
type ObservedExitRouteFullDemoReport struct {
	SchemaID                  string                            `json:"schema_id"`
	ContractVersion           string                            `json:"contract_version"`
	CandidateSchemaID         string                            `json:"candidate_schema_id"`
	NavigationSchemaID        string                            `json:"navigation_schema_id"`
	NavigationContractVersion string                            `json:"navigation_contract_version"`
	BuildIdentifier           string                            `json:"build_identifier"`
	SourceAlias               string                            `json:"source_alias"`
	SourceSHA256              string                            `json:"source_sha256"`
	MapID                     string                            `json:"map_id"`
	TrainingAllowed           bool                              `json:"training_allowed"`
	ProductSupported          bool                              `json:"product_supported"`
	CanonicalBundleVersion    string                            `json:"canonical_bundle_version"`
	PackageResolution         ExitRoutePackageResolutionSummary `json:"package_resolution"`
	NavigationContract        exitroute.NavigationContract      `json:"navigation_contract"`
	Sampling                  ExitRouteSampling                 `json:"sampling"`
	RowsArtifact              ExitRouteArtifact                 `json:"rows_artifact"`
	Coverage                  ExitRouteCoverage                 `json:"coverage"`
}

type exitRouteEmitter struct {
	writer       *bufio.Writer
	hash         hash.Hash
	logicalBytes int64
	records      int
}

func newExitRouteEmitter(writer io.Writer) *exitRouteEmitter {
	return &exitRouteEmitter{writer: bufio.NewWriter(writer), hash: sha256.New()}
}
func (emitter *exitRouteEmitter) emit(record ObservedExitRouteRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if _, err = emitter.writer.Write(payload); err != nil {
		return err
	}
	if _, err = emitter.hash.Write(payload); err != nil {
		return err
	}
	emitter.logicalBytes += int64(len(payload))
	emitter.records++
	return nil
}
func (emitter *exitRouteEmitter) close() error { return emitter.writer.Flush() }
func (emitter *exitRouteEmitter) logicalSHA256() string {
	return hex.EncodeToString(emitter.hash.Sum(nil))
}

func BuildObservedExitRouteFullDemoReport(replay *models.ReplayData, graph *navigation.Graph, resolution exitroute.PackageResolution, sourceAlias, sourceSHA256 string, rowsWriter io.Writer) (ObservedExitRouteFullDemoReport, error) {
	if replay == nil || rowsWriter == nil {
		return ObservedExitRouteFullDemoReport{}, errors.New("build exit route: replay and rows writer are required")
	}
	if !exposureSourceAliasPattern.MatchString(sourceAlias) || !exposureSHA256Pattern.MatchString(sourceSHA256) || sourceAlias != "demo-"+sourceSHA256[:20] {
		return ObservedExitRouteFullDemoReport{}, errors.New("build exit route: source identity is invalid")
	}
	mapID := strings.TrimSpace(replay.Metadata.MapName)
	if mapID == "" || !finiteExposureNumber(replay.Metadata.TickRate) || replay.Metadata.TickRate <= 0 {
		return ObservedExitRouteFullDemoReport{}, errors.New("build exit route: map or tick rate is invalid")
	}
	if graph == nil || len(graph.Nodes) == 0 || graph.Version != navigation.GraphContractVersion {
		return ObservedExitRouteFullDemoReport{}, errors.New("build exit route: directed NAV graph is unavailable")
	}
	if resolution.Status == exitroute.Available {
		if resolution.Package == nil || resolution.Package.MapID != mapID {
			return ObservedExitRouteFullDemoReport{}, errors.New("build exit route: package map mismatch")
		}
		if err := exitroute.ValidatePackage(resolution.Package); err != nil {
			return ObservedExitRouteFullDemoReport{}, err
		}
		if err := exitroute.ValidateObjectiveRoutes(resolution.Package, graph); err != nil {
			return ObservedExitRouteFullDemoReport{}, err
		}
	} else if resolution.Status != exitroute.Unavailable || len(resolution.ReasonCodes) == 0 {
		return ObservedExitRouteFullDemoReport{}, errors.New("build exit route: invalid package resolution")
	}
	prepared, err := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	if err != nil {
		return ObservedExitRouteFullDemoReport{}, err
	}
	rounds, err := canonicalExposureRounds(replay.Rounds)
	if err != nil {
		return ObservedExitRouteFullDemoReport{}, err
	}
	aliases, err := exposureParticipantAliases(rounds)
	if err != nil {
		return ObservedExitRouteFullDemoReport{}, err
	}
	report := ObservedExitRouteFullDemoReport{SchemaID: ObservedExitRouteFullDemoSchemaID, ContractVersion: SpatialFactContractVersion, CandidateSchemaID: ObservedExitRouteDiagnosticSchemaID, NavigationSchemaID: models.ObservedNavigationPathSchemaID, NavigationContractVersion: models.ObservedNavigationPathContractVersion, SourceAlias: sourceAlias, SourceSHA256: sourceSHA256, MapID: mapID, TrainingAllowed: false, ProductSupported: false, CanonicalBundleVersion: "3.8.0_unmodified_sidecar_only", PackageResolution: exitRouteResolutionSummary(resolution), NavigationContract: exitroute.NavigationContract{GraphContractVersion: navigation.GraphContractVersion, ShortestPathAlgorithm: navigation.ShortestPathAlgorithm, EndpointAssociationMethod: navigation.EndpointAssociationMethod, EdgeWeightMethod: navigation.EdgeWeightMethod, EndpointTolerance: navigation.DefaultEndpointTolerance, EndpointVerticalTolerance: navigation.DefaultEndpointVerticalTolerance}, Sampling: ExitRouteSampling{TargetHz: exitRouteSamplingTargetHz, TickRateHz: replay.Metadata.TickRate, PeriodTicks: replay.Metadata.TickRate / exitRouteSamplingTargetHz, Strategy: "round_anchored_right_closed_alive_players"}, Coverage: ExitRouteCoverage{Rounds: len(rounds), SamplesHit: map[string]int{}, SampleAbsenceReasonCounts: map[string]int{}, ObjectiveReasonCounts: map[string]int{}, TacticalReasonCounts: map[string]int{}}}
	emitter := newExitRouteEmitter(rowsWriter)
	for _, round := range rounds {
		for _, frame := range selectExposureFrames(round.Frames, report.Sampling.PeriodTicks) {
			players := append([]models.ReplayPlayerState(nil), frame.Players...)
			sort.Slice(players, func(i, j int) bool { return aliases[players[i].SteamID] < aliases[players[j].SteamID] })
			for _, player := range players {
				if !player.Alive {
					continue
				}
				alias := aliases[player.SteamID]
				if alias == "" {
					return ObservedExitRouteFullDemoReport{}, errors.New("build exit route: alias unavailable")
				}
				record, err := BuildObservedExitRouteRecord(mapID, sourceAlias, round.Round, frame.Tick, alias, player, graph, prepared, resolution)
				if err != nil {
					return ObservedExitRouteFullDemoReport{}, err
				}
				if err := emitter.emit(record); err != nil {
					return ObservedExitRouteFullDemoReport{}, err
				}
				countExitRouteRecord(&report.Coverage, record)
			}
		}
	}
	if err := emitter.close(); err != nil {
		return ObservedExitRouteFullDemoReport{}, err
	}
	report.RowsArtifact = ExitRouteArtifact{Path: ObservedExitRouteRowsFilename, Format: "jsonl", Compression: "gzip", LogicalRecords: emitter.records, LogicalBytes: emitter.logicalBytes, LogicalSHA256: emitter.logicalSHA256(), DigestContract: exitRouteDigestContract}
	if report.Coverage.Records == 0 || report.Coverage.CandidateEvaluations == 0 {
		return ObservedExitRouteFullDemoReport{}, errors.New("build exit route: real replay produced no affected candidate records")
	}
	if err := validateObservedExitRouteFullDemoReport(report, false); err != nil {
		return ObservedExitRouteFullDemoReport{}, err
	}
	return report, nil
}

func BuildObservedExitRouteRecord(mapID, sourceAlias string, roundNumber, t0Tick int, participantAlias string, player models.ReplayPlayerState, graph *navigation.Graph, prepared *navigation.PreparedEvaluator, resolution exitroute.PackageResolution) (ObservedExitRouteRecord, error) {
	if strings.TrimSpace(mapID) == "" || !exposureSourceAliasPattern.MatchString(sourceAlias) || !exposureParticipantAliasPattern.MatchString(participantAlias) || roundNumber < 1 || t0Tick < 0 {
		return ObservedExitRouteRecord{}, errors.New("build exit route record: join keys are invalid")
	}
	position := exitroute.Vector3{X: float64(player.X), Y: float64(player.Y), Z: player.Z}
	var positionFact *exitroute.Vector3
	if finiteExposureNumber(position.X) && finiteExposureNumber(position.Y) && finiteExposureNumber(position.Z) {
		copy := position
		positionFact = &copy
	}
	motionInput := replayNavigationMotionInput(player, t0Tick)
	motionContext := motionContextFromNavigationRequest(motionInput)
	record := ObservedExitRouteRecord{SchemaID: ObservedExitRouteDiagnosticSchemaID, ContractVersion: SpatialFactContractVersion, SourceAlias: sourceAlias, MapID: mapID, RoundNumber: roundNumber, T0Tick: t0Tick, AvailabilityTick: t0Tick, ParticipantAlias: participantAlias, TrainingAllowed: false, ProductSupported: false, Position: positionFact, PostureContext: motionContext, Sample: ExitRouteSampleFact{Status: exitroute.Unavailable, ReasonCodes: []string{}}, Candidates: []ObservedExitRouteCandidate{}, Provenance: ExitRouteProvenance{PositionSource: "replay_player_state_position_16hz", PostureSource: "replay_player_state_causal_context_16hz", MotionSource: "observed_navigation_path_at_t0", PackageSchemaID: exitroute.PackageSchemaID, PackageContractVersion: exitroute.ContractVersion, InputFields: []string{"active_weapon_at_t0", "directed_nav_graph", "horizontal_speed_at_t0", "is_ducking_at_t0", "is_scoped_at_t0", "is_walking_at_t0", "origin_position_at_t0", "static_candidate_destination"}}}
	if resolution.Status != exitroute.Available || resolution.Package == nil {
		record.Sample.ReasonCodes = exitroute.SortedUnique(resolution.ReasonCodes)
		return record, validateObservedExitRouteRecord(record)
	}
	pkg := resolution.Package
	record.Provenance.PackageAssetVersion = stringPointerExit(pkg.AssetVersion)
	record.Provenance.PackageSHA256 = stringPointerExit(resolution.PackageSHA256)
	record.Provenance.GeometryRevision = stringPointerExit(pkg.GeometryRevision)
	record.Provenance.NavAssetSHA256 = stringPointerExit(graph.AssetSHA256)
	if positionFact == nil {
		record.Sample.ReasonCodes = []string{exitroute.ReasonPosition}
		return record, validateObservedExitRouteRecord(record)
	}
	sample, reason := exitroute.SelectSample(pkg, position)
	if sample == nil {
		record.Sample.ReasonCodes = []string{reason}
		return record, validateObservedExitRouteRecord(record)
	}
	record.Sample.Status = "observed"
	record.Sample.SampleID = stringPointerExit(sample.ID)
	record.Sample.SampleAlias = stringPointerExit(sample.Alias)
	for _, candidate := range sample.Candidates {
		queryID := fmt.Sprintf("%s:round:%03d:exit:%s:%d", sourceAlias, roundNumber, candidate.ID, t0Tick)
		request := models.ObservedNavigationPathRequest{JoinKeys: models.ObservedNavigationPathJoinKeys{MatchID: sourceAlias, RoundID: fmt.Sprintf("%s:round:%03d", sourceAlias, roundNumber), QueryID: queryID, OriginPlayerID: participantAlias, TargetPlayerID: candidate.Alias}, RoundNumber: roundNumber, T0Tick: t0Tick, OriginPosition: r3.Vector{X: position.X, Y: position.Y, Z: position.Z}, TargetPosition: r3.Vector{X: candidate.Destination.X, Y: candidate.Destination.Y, Z: candidate.Destination.Z}, OriginAvailabilityTick: t0Tick, TargetAvailabilityTick: 0, OriginMotion: motionInput}
		path, err := buildObservedNavigationPathWithPreparedEvaluator(request, graph, prepared)
		if err != nil {
			return ObservedExitRouteRecord{}, err
		}
		reasons := append([]string{}, path.AbstentionReasons...)
		row := ObservedExitRouteCandidate{CandidateID: candidate.ID, CandidateAlias: candidate.Alias, Destination: candidate.Destination, Objective: ExitRouteObjectiveFact{FieldID: "FLD-NAV-EXIT-PATH", Direction: "origin_to_destination", ReasonCodes: reasons, Reachability: path.Reachability, OriginAssociation: path.Provenance.OriginAssociationWorldUnits, DestinationAssociation: path.Provenance.TargetAssociationWorldUnits}, PhysicalCost: ExitRoutePhysicalCost{NavigationSchemaID: path.SchemaID, NavigationContractVersion: path.ContractVersion, DirectDistance: path.DirectDistance, PathDistance: path.PathDistance, InternalTiming: path.InternalTiming}, Tactical: SpatialTacticalJudgment{Status: exitroute.Unavailable, ReasonCodes: []string{SpatialTacticalJudgmentUnavailableReason}}}
		record.Candidates = append(record.Candidates, row)
	}
	return record, validateObservedExitRouteRecord(record)
}

func ValidateObservedExitRouteFullDemoReport(report ObservedExitRouteFullDemoReport) error {
	return validateObservedExitRouteFullDemoReport(report, true)
}
func validateObservedExitRouteFullDemoReport(report ObservedExitRouteFullDemoReport, physical bool) error {
	if report.SchemaID != ObservedExitRouteFullDemoSchemaID || report.ContractVersion != SpatialFactContractVersion || report.CandidateSchemaID != ObservedExitRouteDiagnosticSchemaID || report.NavigationSchemaID != models.ObservedNavigationPathSchemaID || report.NavigationContractVersion != models.ObservedNavigationPathContractVersion || report.TrainingAllowed || report.ProductSupported || report.CanonicalBundleVersion != "3.8.0_unmodified_sidecar_only" || !exposureSourceAliasPattern.MatchString(report.SourceAlias) || !exposureSHA256Pattern.MatchString(report.SourceSHA256) || report.Sampling.TargetHz != 1 || report.Sampling.Strategy != "round_anchored_right_closed_alive_players" {
		return errors.New("exit route report metadata is invalid")
	}
	if report.RowsArtifact.Path != ObservedExitRouteRowsFilename || report.RowsArtifact.Format != "jsonl" || report.RowsArtifact.Compression != "gzip" || report.RowsArtifact.LogicalRecords != report.Coverage.Records || report.RowsArtifact.LogicalBytes <= 0 || !exposureSHA256Pattern.MatchString(report.RowsArtifact.LogicalSHA256) {
		return errors.New("exit route artifact metadata is invalid")
	}
	if physical && (report.RowsArtifact.Bytes <= 0 || !exposureSHA256Pattern.MatchString(report.RowsArtifact.SHA256)) {
		return errors.New("exit route physical artifact is invalid")
	}
	c := report.Coverage
	if c.Rounds < 1 || c.Records < 1 || c.SamplesObserved+c.SamplesUnavailable != c.Records || c.CandidateEvaluations < 1 || c.Reachable+c.Unreachable+c.ReachabilityUnavailable != c.CandidateEvaluations || c.PhysicalPathAvailable+c.PhysicalPathUnavailable != c.CandidateEvaluations || c.TacticalUnavailable != c.CandidateEvaluations || c.TacticalReasonCounts[SpatialTacticalJudgmentUnavailableReason] != c.CandidateEvaluations {
		return errors.New("exit route coverage is inconsistent")
	}
	return nil
}
func validateObservedExitRouteRecord(record ObservedExitRouteRecord) error {
	if record.SchemaID != ObservedExitRouteDiagnosticSchemaID || record.ContractVersion != SpatialFactContractVersion || !exposureSourceAliasPattern.MatchString(record.SourceAlias) || !exposureParticipantAliasPattern.MatchString(record.ParticipantAlias) || record.RoundNumber < 1 || record.T0Tick < 0 || record.AvailabilityTick != record.T0Tick || record.TrainingAllowed || record.ProductSupported {
		return errors.New("exit route record metadata is invalid")
	}
	if record.Sample.Status == "observed" {
		if record.Position == nil || record.Sample.SampleID == nil || record.Sample.SampleAlias == nil || len(record.Sample.ReasonCodes) != 0 || len(record.Candidates) == 0 {
			return errors.New("observed exit route sample is incomplete")
		}
		for _, candidate := range record.Candidates {
			if candidate.Objective.FieldID != "FLD-NAV-EXIT-PATH" || candidate.Objective.Direction != "origin_to_destination" || candidate.PhysicalCost.NavigationSchemaID != models.ObservedNavigationPathSchemaID || candidate.PhysicalCost.NavigationContractVersion != models.ObservedNavigationPathContractVersion || candidate.PhysicalCost.InternalTiming.ExactUserFacingTimeAllowed || candidate.PhysicalCost.InternalTiming.ArrivalPrediction || candidate.Tactical.Status != exitroute.Unavailable || len(candidate.Tactical.ReasonCodes) != 1 || candidate.Tactical.ReasonCodes[0] != SpatialTacticalJudgmentUnavailableReason {
				return errors.New("candidate semantic partitions are invalid")
			}
		}
	} else if record.Sample.Status != exitroute.Unavailable || len(record.Sample.ReasonCodes) == 0 || record.Sample.SampleID != nil || record.Sample.SampleAlias != nil || len(record.Candidates) != 0 {
		return errors.New("unavailable exit route sample is not fail closed")
	}
	return nil
}
func exitRouteResolutionSummary(resolution exitroute.PackageResolution) ExitRoutePackageResolutionSummary {
	reasons := exitroute.SortedUnique(resolution.ReasonCodes)
	if reasons == nil {
		reasons = []string{}
	}
	summary := ExitRoutePackageResolutionSummary{Status: resolution.Status, ReasonCodes: reasons, ContractVersion: exitroute.ContractVersion}
	if resolution.Package != nil {
		summary.AssetVersion = stringPointerExit(resolution.Package.AssetVersion)
		summary.PackageSHA256 = stringPointerExit(resolution.PackageSHA256)
		summary.GeometryRevision = stringPointerExit(resolution.Package.GeometryRevision)
	}
	return summary
}
func countExitRouteRecord(coverage *ExitRouteCoverage, record ObservedExitRouteRecord) {
	coverage.Records++
	if record.Sample.Status == "observed" {
		coverage.SamplesObserved++
		coverage.SamplesHit[*record.Sample.SampleID]++
	} else {
		coverage.SamplesUnavailable++
		for _, reason := range record.Sample.ReasonCodes {
			coverage.SampleAbsenceReasonCounts[reason]++
		}
	}
	for _, candidate := range record.Candidates {
		coverage.CandidateEvaluations++
		if candidate.Objective.Reachability.Value == nil {
			coverage.ReachabilityUnavailable++
		} else if *candidate.Objective.Reachability.Value {
			coverage.Reachable++
		} else {
			coverage.Unreachable++
		}
		if candidate.PhysicalCost.PathDistance.ValueWorldUnits != nil {
			coverage.PhysicalPathAvailable++
		} else {
			coverage.PhysicalPathUnavailable++
		}
		scenarios := candidate.PhysicalCost.InternalTiming.Scenarios
		if scenarios.ActiveWeaponUninterruptedBaseline.Status == models.NavigationFactEstimated {
			coverage.ActiveWeaponScenarioAvailable++
		} else {
			coverage.ActiveWeaponScenarioUnavailable++
		}
		if scenarios.ObservedPaceProjection.Status == models.NavigationFactEstimated {
			coverage.ObservedPaceScenarioAvailable++
		} else {
			coverage.ObservedPaceScenarioUnavailable++
		}
		coverage.TacticalUnavailable++
		for _, reason := range candidate.Objective.ReasonCodes {
			coverage.ObjectiveReasonCounts[reason]++
		}
		for _, reason := range candidate.Tactical.ReasonCodes {
			coverage.TacticalReasonCounts[reason]++
		}
	}
}
func stringPointerExit(value string) *string { return &value }
