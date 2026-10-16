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
	"cs2-demo-service/pkg/cover"
)

const (
	ObservedCoverDiagnosticSchemaID = "stratai.observed_cover_diagnostic@2"
	ObservedCoverFullDemoSchemaID   = "stratai.observed_cover_full_demo_report@2"
	ObservedCoverFullDemoFilename   = "observed_cover_full_demo.json"
	ObservedCoverRowsFilename       = "observed_cover_diagnostic.jsonl.gz"
	coverDigestContract             = "sha256_over_compact_jsonl_with_lf@1"
	coverSamplingTargetHz           = 1
)

type ObservedCoverDirection struct {
	GeometryDirectionRef       string  `json:"geometry_direction_ref"`
	AbsoluteAzimuthDegrees     float64 `json:"absolute_azimuth_degrees"`
	ViewRelativeAzimuthDegrees float64 `json:"view_relative_azimuth_degrees"`
	ProbeCount                 int     `json:"probe_count"`
	OccludedProbeCount         int     `json:"occluded_probe_count"`
	CompleteVolumeOcclusion    bool    `json:"complete_volume_occlusion"`
}

type ObservedCoverGeometryFact struct {
	FieldID     string                   `json:"field_id"`
	Status      string                   `json:"status"`
	ReasonCodes []string                 `json:"reason_codes"`
	SampleID    *string                  `json:"sample_id"`
	SampleAlias *string                  `json:"sample_alias"`
	Directions  []ObservedCoverDirection `json:"directions"`
}

type ObservedCoverProvenance struct {
	PositionSource         string  `json:"position_source"`
	PostureSource          string  `json:"posture_source"`
	ViewSource             string  `json:"view_source"`
	PackageSchemaID        string  `json:"package_schema_id"`
	PackageContractVersion string  `json:"package_contract_version"`
	PackageAssetVersion    *string `json:"package_asset_version"`
	PackageSHA256          *string `json:"package_sha256"`
	GeometryRevision       *string `json:"geometry_revision"`
}

type ObservedCoverRecord struct {
	SchemaID         string                    `json:"schema_id"`
	ContractVersion  string                    `json:"contract_version"`
	SourceAlias      string                    `json:"source_alias"`
	MapID            string                    `json:"map_id"`
	RoundNumber      int                       `json:"round_number"`
	T0Tick           int                       `json:"t0_tick"`
	AvailabilityTick int                       `json:"availability_tick"`
	ParticipantAlias string                    `json:"participant_alias"`
	TrainingAllowed  bool                      `json:"training_allowed"`
	Position         *cover.Vector3            `json:"position_world"`
	ViewYawDegrees   *float64                  `json:"view_yaw_degrees"`
	PostureID        string                    `json:"posture_id"`
	Geometry         ObservedCoverGeometryFact `json:"geometry_occlusion"`
	Tactical         SpatialTacticalJudgment   `json:"tactical_assessment"`
	Provenance       ObservedCoverProvenance   `json:"provenance"`
}

type CoverPackageResolutionSummary struct {
	Status           string   `json:"status"`
	ReasonCodes      []string `json:"reason_codes"`
	ContractVersion  string   `json:"contract_version"`
	AssetVersion     *string  `json:"asset_version"`
	PackageSHA256    *string  `json:"package_sha256"`
	GeometryRevision *string  `json:"geometry_revision"`
	ProductSupported bool     `json:"product_supported"`
}

type CoverSampling struct {
	TargetHz    int     `json:"target_hz"`
	TickRateHz  float64 `json:"tick_rate_hz"`
	PeriodTicks float64 `json:"period_ticks"`
	Strategy    string  `json:"strategy"`
}
type CoverArtifact struct {
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
type CoverCoverage struct {
	Rounds                      int            `json:"rounds"`
	Records                     int            `json:"records"`
	GeometryObserved            int            `json:"geometry_observed"`
	GeometryUnavailable         int            `json:"geometry_unavailable"`
	DirectionsObserved          int            `json:"directions_observed"`
	CompleteCrouchingDirections int            `json:"complete_crouching_directions"`
	CompleteStandingDirections  int            `json:"complete_standing_directions"`
	TacticalUnavailable         int            `json:"tactical_judgment_unavailable"`
	SamplesHit                  map[string]int `json:"samples_hit"`
	GeometryReasonCounts        map[string]int `json:"physical_absence_reason_counts"`
	TacticalReasonCounts        map[string]int `json:"tactical_judgment_reason_counts"`
}
type CoverVolumeSummary struct {
	ModelID                 string         `json:"model_id"`
	RadiusUnits             float64        `json:"radius_units"`
	LateralOffsetsUnits     []float64      `json:"lateral_offsets_units"`
	PostureIDs              []string       `json:"posture_ids"`
	ProbesPerPosture        map[string]int `json:"probes_per_posture"`
	AzimuthStepDegrees      float64        `json:"azimuth_step_degrees"`
	ProtectionDistanceUnits float64        `json:"protection_distance_units"`
}
type ObservedCoverFullDemoReport struct {
	SchemaID          string                        `json:"schema_id"`
	ContractVersion   string                        `json:"contract_version"`
	CandidateSchemaID string                        `json:"candidate_schema_id"`
	BuildIdentifier   string                        `json:"build_identifier"`
	SourceAlias       string                        `json:"source_alias"`
	SourceSHA256      string                        `json:"source_sha256"`
	MapID             string                        `json:"map_id"`
	TrainingAllowed   bool                          `json:"training_allowed"`
	ProductSupported  bool                          `json:"product_supported"`
	PackageResolution CoverPackageResolutionSummary `json:"package_resolution"`
	PlayerVolume      CoverVolumeSummary            `json:"player_volume"`
	Sampling          CoverSampling                 `json:"sampling"`
	RowsArtifact      CoverArtifact                 `json:"rows_artifact"`
	Coverage          CoverCoverage                 `json:"coverage"`
}

type coverEmitter struct {
	writer       *bufio.Writer
	hash         hash.Hash
	logicalBytes int64
	records      int
}

func BuildObservedCoverFullDemoReport(replay *models.ReplayData, resolution cover.PackageResolution, sourceAlias, sourceSHA256 string, rowsWriter io.Writer) (ObservedCoverFullDemoReport, error) {
	if replay == nil || rowsWriter == nil {
		return ObservedCoverFullDemoReport{}, errors.New("build cover: replay and rows writer are required")
	}
	if !exposureSourceAliasPattern.MatchString(sourceAlias) || !exposureSHA256Pattern.MatchString(sourceSHA256) || sourceAlias != "demo-"+sourceSHA256[:20] {
		return ObservedCoverFullDemoReport{}, errors.New("build cover: source identity is invalid")
	}
	mapID := strings.TrimSpace(replay.Metadata.MapName)
	if mapID == "" || !finiteExposureNumber(replay.Metadata.TickRate) || replay.Metadata.TickRate <= 0 {
		return ObservedCoverFullDemoReport{}, errors.New("build cover: map id or tick rate is invalid")
	}
	if resolution.Status != cover.GeometryAvailable && resolution.Status != cover.GeometryUnavailable {
		return ObservedCoverFullDemoReport{}, errors.New("build cover: package resolution status is invalid")
	}
	if resolution.Status == cover.GeometryAvailable {
		if resolution.Package == nil || resolution.Package.MapID != mapID {
			return ObservedCoverFullDemoReport{}, errors.New("build cover: package map mismatch")
		}
		if err := cover.ValidatePackage(resolution.Package); err != nil {
			return ObservedCoverFullDemoReport{}, fmt.Errorf("build cover: invalid package: %w", err)
		}
	} else if len(resolution.ReasonCodes) == 0 {
		return ObservedCoverFullDemoReport{}, errors.New("build cover: unavailable resolution lacks a reason")
	}
	rounds, err := canonicalExposureRounds(replay.Rounds)
	if err != nil {
		return ObservedCoverFullDemoReport{}, err
	}
	aliases, err := exposureParticipantAliases(rounds)
	if err != nil {
		return ObservedCoverFullDemoReport{}, err
	}
	emitter := newCoverEmitter(rowsWriter)
	report := ObservedCoverFullDemoReport{SchemaID: ObservedCoverFullDemoSchemaID, ContractVersion: SpatialFactContractVersion, CandidateSchemaID: ObservedCoverDiagnosticSchemaID, SourceAlias: sourceAlias, SourceSHA256: sourceSHA256, MapID: mapID, TrainingAllowed: false, ProductSupported: false, PackageResolution: coverResolutionSummary(resolution), PlayerVolume: coverVolumeSummary(resolution), Sampling: CoverSampling{TargetHz: coverSamplingTargetHz, TickRateHz: replay.Metadata.TickRate, PeriodTicks: replay.Metadata.TickRate / coverSamplingTargetHz, Strategy: "round_anchored_right_closed_alive_players"}, Coverage: CoverCoverage{Rounds: len(rounds), SamplesHit: map[string]int{}, GeometryReasonCounts: map[string]int{}, TacticalReasonCounts: map[string]int{}}}
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
					return ObservedCoverFullDemoReport{}, errors.New("build cover: player alias is unavailable")
				}
				record, err := BuildObservedCoverRecord(mapID, sourceAlias, round.Round, frame.Tick, alias, player, resolution)
				if err != nil {
					return ObservedCoverFullDemoReport{}, err
				}
				if err := emitter.emit(record); err != nil {
					return ObservedCoverFullDemoReport{}, err
				}
				countCoverRecord(&report.Coverage, record)
			}
		}
	}
	if err := emitter.close(); err != nil {
		return ObservedCoverFullDemoReport{}, err
	}
	report.RowsArtifact = CoverArtifact{Path: ObservedCoverRowsFilename, Format: "jsonl", Compression: "gzip", LogicalRecords: emitter.records, LogicalBytes: emitter.logicalBytes, LogicalSHA256: emitter.logicalSHA256(), DigestContract: coverDigestContract}
	if report.Coverage.Records == 0 || report.Coverage.GeometryObserved == 0 {
		return ObservedCoverFullDemoReport{}, errors.New("build cover: real replay produced no affected objective geometry records")
	}
	if err := validateObservedCoverFullDemoReport(report, false); err != nil {
		return ObservedCoverFullDemoReport{}, err
	}
	return report, nil
}

func BuildObservedCoverRecord(mapID, sourceAlias string, roundNumber, t0Tick int, participantAlias string, player models.ReplayPlayerState, resolution cover.PackageResolution) (ObservedCoverRecord, error) {
	if strings.TrimSpace(mapID) == "" || !exposureSourceAliasPattern.MatchString(sourceAlias) || !exposureParticipantAliasPattern.MatchString(participantAlias) || roundNumber < 1 || t0Tick < 0 {
		return ObservedCoverRecord{}, errors.New("build cover record: join keys are invalid")
	}
	position := cover.Vector3{X: float64(player.X), Y: float64(player.Y), Z: player.Z}
	var positionFact *cover.Vector3
	if finiteCoverVector(position) {
		positionCopy := position
		positionFact = &positionCopy
	}
	postureID := "standing"
	if player.IsDucking {
		postureID = "crouching"
	}
	record := ObservedCoverRecord{SchemaID: ObservedCoverDiagnosticSchemaID, ContractVersion: SpatialFactContractVersion, SourceAlias: sourceAlias, MapID: mapID, RoundNumber: roundNumber, T0Tick: t0Tick, AvailabilityTick: t0Tick, ParticipantAlias: participantAlias, TrainingAllowed: false, Position: positionFact, PostureID: postureID, Geometry: ObservedCoverGeometryFact{FieldID: "FLD-GEOMETRIC-OCCLUSION", Status: cover.GeometryUnavailable, ReasonCodes: []string{}, Directions: []ObservedCoverDirection{}}, Tactical: SpatialTacticalJudgment{Status: cover.GeometryUnavailable, ReasonCodes: []string{SpatialTacticalJudgmentUnavailableReason}}, Provenance: ObservedCoverProvenance{PositionSource: "replay_player_state_position_16hz", PostureSource: "replay_player_state_is_ducking_16hz", ViewSource: "replay_player_state_view_yaw_16hz", PackageSchemaID: cover.PackageSchemaID, PackageContractVersion: cover.ContractVersion}}
	if resolution.Status != cover.GeometryAvailable || resolution.Package == nil {
		record.Geometry.ReasonCodes = cover.SortedUnique(resolution.ReasonCodes)
		return record, validateObservedCoverRecord(record)
	}
	pkg := resolution.Package
	assetVersion := pkg.AssetVersion
	packageSHA := resolution.PackageSHA256
	revision := pkg.GeometryRevision
	record.Provenance.PackageAssetVersion = &assetVersion
	record.Provenance.PackageSHA256 = &packageSHA
	record.Provenance.GeometryRevision = &revision
	if positionFact == nil {
		record.Geometry.ReasonCodes = []string{cover.ReasonPosition}
		return record, validateObservedCoverRecord(record)
	}
	if !player.ViewAnglesAvailable || !finiteExposureNumber(float64(player.Yaw)) {
		record.Geometry.ReasonCodes = []string{cover.ReasonViewAngles}
		return record, validateObservedCoverRecord(record)
	}
	yaw := cover.NormalizeDegrees(float64(player.Yaw))
	record.ViewYawDegrees = &yaw
	sample, reason := cover.SelectSample(pkg, position)
	if sample == nil {
		record.Geometry.ReasonCodes = []string{reason}
		return record, validateObservedCoverRecord(record)
	}
	record.Geometry.Status = "observed"
	record.Geometry.SampleID = stringPointerCover(sample.ID)
	record.Geometry.SampleAlias = stringPointerCover(sample.Alias)
	for _, direction := range sample.Directions {
		profile, ok := cover.ProfileForPosture(direction, postureID)
		if !ok {
			return ObservedCoverRecord{}, errors.New("build cover record: package lacks current posture")
		}
		record.Geometry.Directions = append(record.Geometry.Directions, ObservedCoverDirection{GeometryDirectionRef: direction.ID, AbsoluteAzimuthDegrees: direction.AzimuthDegrees, ViewRelativeAzimuthDegrees: cover.NormalizeDegrees(direction.AzimuthDegrees - yaw), ProbeCount: profile.ProbeCount, OccludedProbeCount: profile.OccludedProbeCount, CompleteVolumeOcclusion: profile.CompleteVolumeOcclusion})
	}
	return record, validateObservedCoverRecord(record)
}

func ValidateObservedCoverFullDemoReport(report ObservedCoverFullDemoReport) error {
	return validateObservedCoverFullDemoReport(report, true)
}
func validateObservedCoverFullDemoReport(report ObservedCoverFullDemoReport, requirePhysical bool) error {
	if report.SchemaID != ObservedCoverFullDemoSchemaID || report.ContractVersion != SpatialFactContractVersion || report.CandidateSchemaID != ObservedCoverDiagnosticSchemaID || report.TrainingAllowed || report.ProductSupported || !exposureSourceAliasPattern.MatchString(report.SourceAlias) || !exposureSHA256Pattern.MatchString(report.SourceSHA256) || strings.TrimSpace(report.MapID) == "" || !finiteExposureNumber(report.Sampling.TickRateHz) || report.Sampling.TickRateHz <= 0 || report.Sampling.TargetHz != coverSamplingTargetHz || report.Sampling.PeriodTicks != report.Sampling.TickRateHz/coverSamplingTargetHz || report.Sampling.Strategy != "round_anchored_right_closed_alive_players" {
		return errors.New("cover full-demo report metadata is invalid")
	}
	if report.PackageResolution.Status != cover.GeometryAvailable && report.PackageResolution.Status != cover.GeometryUnavailable {
		return errors.New("cover package resolution status is invalid")
	}
	if report.PackageResolution.Status == cover.GeometryAvailable {
		if report.PackageResolution.AssetVersion == nil || report.PackageResolution.PackageSHA256 == nil || report.PackageResolution.GeometryRevision == nil || !exposureSHA256Pattern.MatchString(*report.PackageResolution.PackageSHA256) || !exposureSHA256Pattern.MatchString(*report.PackageResolution.GeometryRevision) {
			return errors.New("available cover package metadata is incomplete")
		}
	} else if len(report.PackageResolution.ReasonCodes) == 0 {
		return errors.New("unavailable cover package lacks reason codes")
	}
	if report.RowsArtifact.Path != ObservedCoverRowsFilename || report.RowsArtifact.Format != "jsonl" || report.RowsArtifact.Compression != "gzip" || report.RowsArtifact.DigestContract != coverDigestContract || report.RowsArtifact.LogicalRecords != report.Coverage.Records || report.RowsArtifact.LogicalBytes <= 0 || !exposureSHA256Pattern.MatchString(report.RowsArtifact.LogicalSHA256) {
		return errors.New("cover rows artifact metadata is invalid")
	}
	if requirePhysical && (report.RowsArtifact.Bytes <= 0 || !exposureSHA256Pattern.MatchString(report.RowsArtifact.SHA256)) {
		return errors.New("cover physical artifact metadata is invalid")
	}
	if report.Coverage.Rounds < 1 || report.Coverage.Records < 1 || report.Coverage.GeometryObserved < 1 || report.Coverage.GeometryObserved+report.Coverage.GeometryUnavailable != report.Coverage.Records || report.Coverage.TacticalUnavailable != report.Coverage.Records || report.Coverage.TacticalReasonCounts[SpatialTacticalJudgmentUnavailableReason] != report.Coverage.Records {
		return errors.New("cover coverage counts are inconsistent")
	}
	return nil
}

func validateObservedCoverRecord(record ObservedCoverRecord) error {
	if record.SchemaID != ObservedCoverDiagnosticSchemaID || record.ContractVersion != SpatialFactContractVersion || !exposureSourceAliasPattern.MatchString(record.SourceAlias) || strings.TrimSpace(record.MapID) == "" || record.RoundNumber < 1 || record.T0Tick < 0 || record.AvailabilityTick != record.T0Tick || !exposureParticipantAliasPattern.MatchString(record.ParticipantAlias) || record.TrainingAllowed || (record.Position != nil && !finiteCoverVector(*record.Position)) || (record.PostureID != "standing" && record.PostureID != "crouching") || record.Provenance.PositionSource != "replay_player_state_position_16hz" || record.Provenance.PostureSource != "replay_player_state_is_ducking_16hz" || record.Provenance.ViewSource != "replay_player_state_view_yaw_16hz" || record.Provenance.PackageSchemaID != cover.PackageSchemaID || record.Provenance.PackageContractVersion != cover.ContractVersion {
		return errors.New("cover record metadata is invalid")
	}
	if record.Geometry.FieldID != "FLD-GEOMETRIC-OCCLUSION" {
		return errors.New("cover geometry field id is invalid")
	}
	if record.Geometry.Status == "observed" {
		if record.Position == nil || record.Geometry.SampleID == nil || record.Geometry.SampleAlias == nil || len(record.Geometry.ReasonCodes) != 0 || record.ViewYawDegrees == nil || !finiteExposureNumber(*record.ViewYawDegrees) || *record.ViewYawDegrees < 0 || *record.ViewYawDegrees >= 360 || len(record.Geometry.Directions) == 0 {
			return errors.New("observed cover geometry is incomplete")
		}
		for _, direction := range record.Geometry.Directions {
			if direction.ProbeCount < 1 || direction.OccludedProbeCount < 0 || direction.OccludedProbeCount > direction.ProbeCount || direction.CompleteVolumeOcclusion != (direction.OccludedProbeCount == direction.ProbeCount) || direction.AbsoluteAzimuthDegrees < 0 || direction.AbsoluteAzimuthDegrees >= 360 || direction.ViewRelativeAzimuthDegrees < 0 || direction.ViewRelativeAzimuthDegrees >= 360 {
				return errors.New("observed cover direction is invalid")
			}
		}
	} else if record.Geometry.Status != cover.GeometryUnavailable || len(record.Geometry.ReasonCodes) == 0 || record.Geometry.SampleID != nil || record.Geometry.SampleAlias != nil || len(record.Geometry.Directions) != 0 {
		return errors.New("unavailable cover geometry is not fail-closed")
	}
	if record.Tactical.Status != cover.GeometryUnavailable || len(record.Tactical.ReasonCodes) != 1 || record.Tactical.ReasonCodes[0] != SpatialTacticalJudgmentUnavailableReason {
		return errors.New("tactical assessment must remain unavailable outside a decision contract")
	}
	return nil
}

func coverResolutionSummary(resolution cover.PackageResolution) CoverPackageResolutionSummary {
	summary := CoverPackageResolutionSummary{Status: resolution.Status, ReasonCodes: cover.SortedUnique(resolution.ReasonCodes), ContractVersion: cover.ContractVersion}
	if resolution.Package != nil {
		asset := resolution.Package.AssetVersion
		digest := resolution.PackageSHA256
		revision := resolution.Package.GeometryRevision
		summary.AssetVersion = &asset
		summary.PackageSHA256 = &digest
		summary.GeometryRevision = &revision
		if resolution.Support != nil {
			summary.ProductSupported = resolution.Support.ProductSupported
		}
	}
	return summary
}
func coverVolumeSummary(resolution cover.PackageResolution) CoverVolumeSummary {
	summary := CoverVolumeSummary{PostureIDs: []string{}, LateralOffsetsUnits: []float64{}, ProbesPerPosture: map[string]int{}}
	if resolution.Package == nil {
		return summary
	}
	pkg := resolution.Package
	summary.ModelID = pkg.PlayerVolume.ModelID
	summary.RadiusUnits = pkg.PlayerVolume.RadiusUnits
	summary.LateralOffsetsUnits = append([]float64(nil), pkg.PlayerVolume.LateralOffsetsUnits...)
	summary.AzimuthStepDegrees = pkg.Resolution.AzimuthStepDegrees
	summary.ProtectionDistanceUnits = pkg.Resolution.ProtectionDistanceUnits
	for _, posture := range pkg.PlayerVolume.Postures {
		summary.PostureIDs = append(summary.PostureIDs, posture.ID)
		summary.ProbesPerPosture[posture.ID] = len(posture.ProbeHeightsUnits) * len(pkg.PlayerVolume.LateralOffsetsUnits)
	}
	return summary
}
func countCoverRecord(coverage *CoverCoverage, record ObservedCoverRecord) {
	coverage.Records++
	if record.Geometry.Status == "observed" {
		coverage.GeometryObserved++
		coverage.SamplesHit[*record.Geometry.SampleID]++
		coverage.DirectionsObserved += len(record.Geometry.Directions)
		for _, direction := range record.Geometry.Directions {
			if direction.CompleteVolumeOcclusion {
				if record.PostureID == "crouching" {
					coverage.CompleteCrouchingDirections++
				} else {
					coverage.CompleteStandingDirections++
				}
			}
		}
	} else {
		coverage.GeometryUnavailable++
		for _, reason := range record.Geometry.ReasonCodes {
			coverage.GeometryReasonCounts[reason]++
		}
	}
	coverage.TacticalUnavailable++
	for _, reason := range record.Tactical.ReasonCodes {
		coverage.TacticalReasonCounts[reason]++
	}
}

type coverHash struct {
	value interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	}
}

func newCoverEmitter(writer io.Writer) *coverEmitter {
	hash := sha256.New()
	return &coverEmitter{writer: bufio.NewWriter(writer), hash: hash}
}
func (emitter *coverEmitter) emit(record ObservedCoverRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if _, err := emitter.writer.Write(payload); err != nil {
		return err
	}
	if _, err := emitter.hash.Write(payload); err != nil {
		return err
	}
	emitter.logicalBytes += int64(len(payload))
	emitter.records++
	return nil
}
func (emitter *coverEmitter) close() error          { return emitter.writer.Flush() }
func (emitter *coverEmitter) logicalSHA256() string { return hex.EncodeToString(emitter.hash.Sum(nil)) }
func stringPointerCover(value string) *string       { return &value }
func finiteCoverVector(value cover.Vector3) bool {
	return finiteExposureNumber(value.X) && finiteExposureNumber(value.Y) && finiteExposureNumber(value.Z)
}
