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
	"math"
	"regexp"
	"sort"
	"strings"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/exposure"
)

const (
	ObservedExposureAngleDiagnosticSchemaID = "stratai.observed_exposure_angle_diagnostic@2"
	ObservedExposureAngleFullDemoSchemaID   = "stratai.observed_exposure_angle_full_demo_report@2"
	ObservedExposureAngleFullDemoFilename   = "observed_exposure_angle_full_demo.json"
	ObservedExposureAngleRowsFilename       = "observed_exposure_angle_diagnostic.jsonl.gz"
	exposureAngleDigestContract             = "sha256_over_compact_jsonl_with_lf@1"
	exposureAngleSamplingTargetHz           = 1
)

var (
	exposureParticipantAliasPattern = regexp.MustCompile(`^participant-[0-9]{3}$`)
	exposureSourceAliasPattern      = regexp.MustCompile(`^demo-[0-9a-f]{20}$`)
	exposureSHA256Pattern           = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type ExposureAngleFact struct {
	FieldID          string            `json:"field_id"`
	Status           string            `json:"status"`
	ReasonCodes      []string          `json:"reason_codes"`
	SampleID         *string           `json:"sample_id"`
	SampleAlias      *string           `json:"sample_alias"`
	HorizontalUnits  *float64          `json:"horizontal_distance_units"`
	VerticalUnits    *float64          `json:"vertical_distance_units"`
	AbsoluteSectors  []exposure.Sector `json:"absolute_geometry_sectors"`
	RelativeSectors  []exposure.Sector `json:"view_relative_geometry_sectors"`
	ViewInsideSector *bool             `json:"view_inside_geometry_sector"`
}

type ExposureAngleProvenance struct {
	PositionSource         string  `json:"position_source"`
	ViewAnglesSource       string  `json:"view_angles_source"`
	PackageSHA256          *string `json:"package_sha256"`
	GeometryRevision       *string `json:"geometry_revision"`
	PackageContractVersion string  `json:"package_contract_version"`
}

type ObservedExposureAngleRecord struct {
	SchemaID         string                  `json:"schema_id"`
	ContractVersion  string                  `json:"contract_version"`
	SourceAlias      string                  `json:"source_alias"`
	MapID            string                  `json:"map_id"`
	AssetVersion     *string                 `json:"asset_version"`
	RoundNumber      int                     `json:"round_number"`
	T0Tick           int                     `json:"t0_tick"`
	AvailabilityTick int                     `json:"availability_tick"`
	ParticipantAlias string                  `json:"participant_alias"`
	Position         exposure.Vector3        `json:"position_world"`
	ViewYawDegrees   *float64                `json:"view_yaw_degrees"`
	Geometry         ExposureAngleFact       `json:"geometry_clearance"`
	Tactical         SpatialTacticalJudgment `json:"tactical_assessment"`
	Provenance       ExposureAngleProvenance `json:"provenance"`
	TrainingAllowed  bool                    `json:"training_allowed"`
}

type ExposurePackageResolutionSummary struct {
	Status           string   `json:"status"`
	ReasonCodes      []string `json:"reason_codes"`
	ContractVersion  string   `json:"contract_version"`
	AssetVersion     *string  `json:"asset_version"`
	PackageSHA256    *string  `json:"package_sha256"`
	GeometryRevision *string  `json:"geometry_revision"`
	ProductSupported bool     `json:"product_supported"`
}

type ExposureAngleSampling struct {
	TargetHz    int     `json:"target_hz"`
	TickRateHz  float64 `json:"tick_rate_hz"`
	PeriodTicks float64 `json:"period_ticks"`
	Strategy    string  `json:"strategy"`
}

type ExposureAngleArtifact struct {
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

type ExposureAngleCoverage struct {
	RoundNumbers         []int          `json:"round_numbers"`
	InputFrames          int            `json:"input_frames"`
	SelectedFrames       int            `json:"selected_frames"`
	ParticipantAliases   int            `json:"participant_aliases"`
	Records              int            `json:"records"`
	GeometryObserved     int            `json:"geometry_observed"`
	GeometryUnavailable  int            `json:"geometry_unavailable"`
	TacticalUnavailable  int            `json:"tactical_judgment_unavailable"`
	SamplesHit           map[string]int `json:"samples_hit"`
	GeometryReasonCounts map[string]int `json:"physical_absence_reason_counts"`
	TacticalReasonCounts map[string]int `json:"tactical_judgment_reason_counts"`
}

type ObservedExposureAngleFullDemoReport struct {
	SchemaID          string                           `json:"schema_id"`
	ContractVersion   string                           `json:"contract_version"`
	CandidateSchemaID string                           `json:"candidate_schema_id"`
	SourceAlias       string                           `json:"source_alias"`
	SourceSHA256      string                           `json:"source_sha256"`
	MapID             string                           `json:"map_id"`
	BuildIdentifier   string                           `json:"build_identifier"`
	TrainingAllowed   bool                             `json:"training_allowed"`
	PackageResolution ExposurePackageResolutionSummary `json:"package_resolution"`
	Sampling          ExposureAngleSampling            `json:"sampling"`
	RowsArtifact      ExposureAngleArtifact            `json:"rows_artifact"`
	Coverage          ExposureAngleCoverage            `json:"coverage"`
}

type exposureAngleEmitter struct {
	writer       *bufio.Writer
	hash         hash.Hash
	logicalBytes int64
	records      int
}

func BuildObservedExposureAngleFullDemoReport(
	replay *models.ReplayData,
	resolution exposure.PackageResolution,
	sourceAlias, sourceSHA256 string,
	rowsWriter io.Writer,
) (ObservedExposureAngleFullDemoReport, error) {
	if replay == nil || rowsWriter == nil {
		return ObservedExposureAngleFullDemoReport{}, errors.New("build exposure angles: replay and rows writer are required")
	}
	if !validObservedVisualContactSmokeIdentity(sourceAlias, sourceSHA256) {
		return ObservedExposureAngleFullDemoReport{}, errors.New("build exposure angles: source identity is invalid")
	}
	mapID := strings.TrimSpace(replay.Metadata.MapName)
	if mapID == "" || !finiteExposureNumber(replay.Metadata.TickRate) || replay.Metadata.TickRate <= 0 {
		return ObservedExposureAngleFullDemoReport{}, errors.New("build exposure angles: map id or tick rate is invalid")
	}
	if resolution.Status != exposure.GeometryAvailable && resolution.Status != exposure.GeometryUnavailable {
		return ObservedExposureAngleFullDemoReport{}, errors.New("build exposure angles: package resolution status is invalid")
	}
	if resolution.Status == exposure.GeometryAvailable {
		if resolution.Package == nil || resolution.Package.MapID != mapID {
			return ObservedExposureAngleFullDemoReport{}, errors.New("build exposure angles: package map mismatch")
		}
		if err := exposure.ValidatePackage(resolution.Package); err != nil {
			return ObservedExposureAngleFullDemoReport{}, fmt.Errorf("build exposure angles: invalid package: %w", err)
		}
	} else if len(resolution.ReasonCodes) == 0 || resolution.Package != nil {
		return ObservedExposureAngleFullDemoReport{}, errors.New("build exposure angles: unavailable resolution lacks a reason")
	}

	rounds, err := canonicalExposureRounds(replay.Rounds)
	if err != nil {
		return ObservedExposureAngleFullDemoReport{}, err
	}
	aliases, err := exposureParticipantAliases(rounds)
	if err != nil {
		return ObservedExposureAngleFullDemoReport{}, err
	}
	emitter := newExposureAngleEmitter(rowsWriter)
	report := ObservedExposureAngleFullDemoReport{
		SchemaID:          ObservedExposureAngleFullDemoSchemaID,
		ContractVersion:   SpatialFactContractVersion,
		CandidateSchemaID: ObservedExposureAngleDiagnosticSchemaID,
		SourceAlias:       sourceAlias,
		SourceSHA256:      sourceSHA256,
		MapID:             mapID,
		TrainingAllowed:   false,
		PackageResolution: exposureResolutionSummary(resolution),
		Sampling: ExposureAngleSampling{
			TargetHz:    exposureAngleSamplingTargetHz,
			TickRateHz:  replay.Metadata.TickRate,
			PeriodTicks: replay.Metadata.TickRate / exposureAngleSamplingTargetHz,
			Strategy:    "round_anchored_first_frame_at_or_after_period_no_carry_forward",
		},
		Coverage: ExposureAngleCoverage{
			RoundNumbers:         make([]int, 0, len(rounds)),
			ParticipantAliases:   len(aliases),
			SamplesHit:           make(map[string]int),
			GeometryReasonCounts: make(map[string]int),
			TacticalReasonCounts: make(map[string]int),
		},
	}

	for _, round := range rounds {
		report.Coverage.RoundNumbers = append(report.Coverage.RoundNumbers, round.Round)
		report.Coverage.InputFrames += len(round.Frames)
		selectedFrames := selectExposureFrames(round.Frames, report.Sampling.PeriodTicks)
		report.Coverage.SelectedFrames += len(selectedFrames)
		for _, frame := range selectedFrames {
			players := append([]models.ReplayPlayerState(nil), frame.Players...)
			sort.Slice(players, func(i, j int) bool { return aliases[players[i].SteamID] < aliases[players[j].SteamID] })
			for _, player := range players {
				if !player.Alive {
					continue
				}
				alias, ok := aliases[player.SteamID]
				if !ok {
					return ObservedExposureAngleFullDemoReport{}, errors.New("build exposure angles: player alias is unavailable")
				}
				record, err := BuildObservedExposureAngleRecord(
					mapID, sourceAlias, alias, round.Round, frame.Tick, player, resolution,
				)
				if err != nil {
					return ObservedExposureAngleFullDemoReport{}, err
				}
				if err := emitter.emit(record); err != nil {
					return ObservedExposureAngleFullDemoReport{}, err
				}
				countExposureRecord(&report.Coverage, record)
			}
		}
	}
	if err := emitter.close(); err != nil {
		return ObservedExposureAngleFullDemoReport{}, err
	}
	report.RowsArtifact = ExposureAngleArtifact{
		Path:           ObservedExposureAngleRowsFilename,
		Format:         "jsonl",
		Compression:    "gzip",
		RecordCount:    emitter.records,
		LogicalBytes:   emitter.logicalBytes,
		LogicalSHA256:  emitter.logicalSHA256(),
		DigestContract: exposureAngleDigestContract,
	}
	if report.Coverage.Records == 0 || report.Coverage.Records != emitter.records {
		return ObservedExposureAngleFullDemoReport{}, errors.New("build exposure angles: real replay produced no candidate records")
	}
	if err := validateObservedExposureAngleFullDemoReport(report, false); err != nil {
		return ObservedExposureAngleFullDemoReport{}, err
	}
	return report, nil
}

func BuildObservedExposureAngleRecord(
	mapID, sourceAlias, participantAlias string,
	roundNumber, t0Tick int,
	player models.ReplayPlayerState,
	resolution exposure.PackageResolution,
) (ObservedExposureAngleRecord, error) {
	if strings.TrimSpace(mapID) == "" || !exposureSourceAliasPattern.MatchString(sourceAlias) ||
		!exposureParticipantAliasPattern.MatchString(participantAlias) || roundNumber < 1 || t0Tick < 0 {
		return ObservedExposureAngleRecord{}, errors.New("build exposure angle record: join keys are invalid")
	}
	position := exposure.Vector3{X: float64(player.X), Y: float64(player.Y), Z: player.Z}
	record := ObservedExposureAngleRecord{
		SchemaID:         ObservedExposureAngleDiagnosticSchemaID,
		ContractVersion:  SpatialFactContractVersion,
		SourceAlias:      sourceAlias,
		MapID:            mapID,
		RoundNumber:      roundNumber,
		T0Tick:           t0Tick,
		AvailabilityTick: t0Tick,
		ParticipantAlias: participantAlias,
		Position:         position,
		Geometry: ExposureAngleFact{
			FieldID:         "FLD-GEOMETRIC-CLEARANCE-SECTORS",
			Status:          exposure.GeometryUnavailable,
			ReasonCodes:     []string{},
			AbsoluteSectors: []exposure.Sector{},
			RelativeSectors: []exposure.Sector{},
		},
		Tactical: SpatialTacticalJudgment{
			Status:      exposure.GeometryUnavailable,
			ReasonCodes: []string{SpatialTacticalJudgmentUnavailableReason},
		},
		Provenance: ExposureAngleProvenance{
			PositionSource:         "replay_player_state_position_16hz",
			ViewAnglesSource:       "demoinfocs_view_angles_at_source_frame",
			PackageContractVersion: exposure.ContractVersion,
		},
		TrainingAllowed: false,
	}
	if resolution.Status != exposure.GeometryAvailable || resolution.Package == nil {
		record.Geometry.ReasonCodes = exposureSortedUniqueStrings(resolution.ReasonCodes)
		return record, validateObservedExposureAngleRecord(record)
	}
	pkg := resolution.Package
	assetVersion := pkg.AssetVersion
	record.AssetVersion = &assetVersion
	packageHash := resolution.PackageSHA256
	record.Provenance.PackageSHA256 = &packageHash
	geometryRevision := pkg.GeometryRevision
	record.Provenance.GeometryRevision = &geometryRevision
	if !finiteExposureVector(position) {
		record.Geometry.ReasonCodes = []string{exposure.ReasonPosition}
		return record, validateObservedExposureAngleRecord(record)
	}
	if !player.ViewAnglesAvailable || !finiteExposureNumber(float64(player.Yaw)) {
		record.Geometry.ReasonCodes = []string{exposure.ReasonViewAngles}
		return record, validateObservedExposureAngleRecord(record)
	}
	yaw := exposure.NormalizeDegrees(float64(player.Yaw))
	record.ViewYawDegrees = &yaw
	sample, reason := exposure.SelectSample(pkg, position)
	if sample == nil {
		record.Geometry.ReasonCodes = []string{reason}
		return record, validateObservedExposureAngleRecord(record)
	}
	record.Geometry.Status = "observed"
	sampleID, sampleAlias := sample.ID, sample.Alias
	record.Geometry.SampleID = &sampleID
	record.Geometry.SampleAlias = &sampleAlias
	horizontal := math.Hypot(position.X-sample.Origin.X, position.Y-sample.Origin.Y)
	vertical := math.Abs(position.Z - sample.Origin.Z)
	record.Geometry.HorizontalUnits = &horizontal
	record.Geometry.VerticalUnits = &vertical
	record.Geometry.AbsoluteSectors = append([]exposure.Sector(nil), sample.Sectors...)
	record.Geometry.RelativeSectors = exposure.RelativeSectors(sample.Sectors, yaw)
	inside := false
	for _, sector := range sample.Sectors {
		if exposure.SectorContains(sector, yaw) {
			inside = true
			break
		}
	}
	record.Geometry.ViewInsideSector = &inside
	return record, validateObservedExposureAngleRecord(record)
}

func ValidateObservedExposureAngleFullDemoReport(report ObservedExposureAngleFullDemoReport) error {
	return validateObservedExposureAngleFullDemoReport(report, true)
}

func validateObservedExposureAngleFullDemoReport(report ObservedExposureAngleFullDemoReport, requirePhysical bool) error {
	if report.SchemaID != ObservedExposureAngleFullDemoSchemaID || report.ContractVersion != SpatialFactContractVersion ||
		report.CandidateSchemaID != ObservedExposureAngleDiagnosticSchemaID || report.TrainingAllowed ||
		!validObservedVisualContactSmokeIdentity(report.SourceAlias, report.SourceSHA256) ||
		strings.TrimSpace(report.MapID) == "" || !finiteExposureNumber(report.Sampling.TickRateHz) ||
		report.Sampling.TickRateHz <= 0 || report.Sampling.TargetHz != exposureAngleSamplingTargetHz ||
		report.Sampling.PeriodTicks != report.Sampling.TickRateHz/exposureAngleSamplingTargetHz ||
		strings.TrimSpace(report.Sampling.Strategy) == "" {
		return errors.New("exposure angle full-demo report metadata is invalid")
	}
	if report.PackageResolution.Status != exposure.GeometryAvailable &&
		report.PackageResolution.Status != exposure.GeometryUnavailable {
		return errors.New("exposure package resolution status is invalid")
	}
	if report.PackageResolution.Status == exposure.GeometryAvailable {
		if report.PackageResolution.AssetVersion == nil || report.PackageResolution.PackageSHA256 == nil ||
			report.PackageResolution.GeometryRevision == nil ||
			!exposureSHA256Pattern.MatchString(*report.PackageResolution.PackageSHA256) ||
			!exposureSHA256Pattern.MatchString(*report.PackageResolution.GeometryRevision) {
			return errors.New("available exposure package metadata is incomplete")
		}
	} else if len(report.PackageResolution.ReasonCodes) == 0 {
		return errors.New("unavailable exposure package lacks reason codes")
	}
	if report.RowsArtifact.Path != ObservedExposureAngleRowsFilename || report.RowsArtifact.Format != "jsonl" ||
		report.RowsArtifact.Compression != "gzip" || report.RowsArtifact.DigestContract != exposureAngleDigestContract ||
		report.RowsArtifact.RecordCount < 1 || report.RowsArtifact.LogicalBytes < 1 ||
		!exposureSHA256Pattern.MatchString(report.RowsArtifact.LogicalSHA256) {
		return errors.New("exposure angle rows artifact metadata is invalid")
	}
	if requirePhysical && (strings.TrimSpace(report.BuildIdentifier) == "" || report.RowsArtifact.Bytes < 1 ||
		!exposureSHA256Pattern.MatchString(report.RowsArtifact.SHA256)) {
		return errors.New("exposure angle physical artifact metadata is invalid")
	}
	coverage := report.Coverage
	if len(coverage.RoundNumbers) == 0 || coverage.InputFrames < coverage.SelectedFrames || coverage.SelectedFrames < 1 ||
		coverage.ParticipantAliases < 1 || coverage.Records != report.RowsArtifact.RecordCount || coverage.Records < 1 ||
		coverage.GeometryObserved+coverage.GeometryUnavailable != coverage.Records ||
		coverage.TacticalUnavailable != coverage.Records ||
		coverage.TacticalReasonCounts[SpatialTacticalJudgmentUnavailableReason] != coverage.Records {
		return errors.New("exposure angle coverage counts are inconsistent")
	}
	return nil
}

func validateObservedExposureAngleRecord(record ObservedExposureAngleRecord) error {
	if record.SchemaID != ObservedExposureAngleDiagnosticSchemaID || record.ContractVersion != SpatialFactContractVersion ||
		!exposureSourceAliasPattern.MatchString(record.SourceAlias) || strings.TrimSpace(record.MapID) == "" ||
		record.RoundNumber < 1 || record.T0Tick < 0 || record.AvailabilityTick > record.T0Tick ||
		!exposureParticipantAliasPattern.MatchString(record.ParticipantAlias) || record.TrainingAllowed ||
		!finiteExposureVector(record.Position) || record.Provenance.PositionSource != "replay_player_state_position_16hz" ||
		record.Provenance.ViewAnglesSource != "demoinfocs_view_angles_at_source_frame" ||
		record.Provenance.PackageContractVersion != exposure.ContractVersion {
		return errors.New("exposure angle record metadata is invalid")
	}
	if record.Geometry.FieldID != "FLD-GEOMETRIC-CLEARANCE-SECTORS" {
		return errors.New("exposure geometry field id is invalid")
	}
	if record.Geometry.Status == "observed" {
		if record.AssetVersion == nil || record.ViewYawDegrees == nil || record.Geometry.SampleID == nil ||
			record.Geometry.SampleAlias == nil || record.Geometry.HorizontalUnits == nil || record.Geometry.VerticalUnits == nil ||
			record.Geometry.ViewInsideSector == nil || len(record.Geometry.ReasonCodes) != 0 ||
			len(record.Geometry.AbsoluteSectors) == 0 || len(record.Geometry.RelativeSectors) == 0 ||
			record.Provenance.PackageSHA256 == nil || record.Provenance.GeometryRevision == nil {
			return errors.New("observed exposure geometry is incomplete")
		}
		if !sameExposureSectorWidths(record.Geometry.AbsoluteSectors, record.Geometry.RelativeSectors) {
			return errors.New("relative exposure sectors do not preserve absolute sector widths")
		}
	} else if record.Geometry.Status != exposure.GeometryUnavailable || len(record.Geometry.ReasonCodes) == 0 ||
		record.Geometry.SampleID != nil || record.Geometry.SampleAlias != nil || record.Geometry.HorizontalUnits != nil ||
		record.Geometry.VerticalUnits != nil || record.Geometry.ViewInsideSector != nil ||
		len(record.Geometry.AbsoluteSectors) != 0 || len(record.Geometry.RelativeSectors) != 0 {
		return errors.New("unavailable exposure geometry is not fail-closed")
	}
	if record.Tactical.Status != exposure.GeometryUnavailable || len(record.Tactical.ReasonCodes) != 1 ||
		record.Tactical.ReasonCodes[0] != SpatialTacticalJudgmentUnavailableReason {
		return errors.New("tactical assessment must remain unavailable outside a decision contract")
	}
	return nil
}

func canonicalExposureRounds(input []models.ReplayRound) ([]models.ReplayRound, error) {
	rounds := append([]models.ReplayRound(nil), input...)
	sort.Slice(rounds, func(i, j int) bool { return rounds[i].Round < rounds[j].Round })
	seen := make(map[int]struct{}, len(rounds))
	for index := range rounds {
		if rounds[index].Round < 1 || len(rounds[index].Frames) == 0 {
			return nil, errors.New("exposure replay contains an invalid or empty round")
		}
		if _, duplicate := seen[rounds[index].Round]; duplicate {
			return nil, errors.New("exposure replay contains duplicate rounds")
		}
		seen[rounds[index].Round] = struct{}{}
		rounds[index].Frames = append([]models.ReplayFrame(nil), rounds[index].Frames...)
		sort.Slice(rounds[index].Frames, func(i, j int) bool { return rounds[index].Frames[i].Tick < rounds[index].Frames[j].Tick })
		for frameIndex := 1; frameIndex < len(rounds[index].Frames); frameIndex++ {
			if rounds[index].Frames[frameIndex].Tick <= rounds[index].Frames[frameIndex-1].Tick {
				return nil, errors.New("exposure replay frames are not strictly ordered")
			}
		}
	}
	return rounds, nil
}

func exposureParticipantAliases(rounds []models.ReplayRound) (map[uint64]string, error) {
	ids := make(map[uint64]struct{})
	for _, round := range rounds {
		for _, frame := range round.Frames {
			for _, player := range frame.Players {
				if player.SteamID == 0 {
					return nil, errors.New("exposure replay contains an unreconciled participant")
				}
				ids[player.SteamID] = struct{}{}
			}
		}
	}
	ordered := make([]uint64, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	aliases := make(map[uint64]string, len(ordered))
	for index, id := range ordered {
		aliases[id] = fmt.Sprintf("participant-%03d", index+1)
	}
	return aliases, nil
}

func selectExposureFrames(frames []models.ReplayFrame, periodTicks float64) []models.ReplayFrame {
	if len(frames) == 0 || periodTicks <= 0 {
		return nil
	}
	selected := make([]models.ReplayFrame, 0, int(math.Ceil(float64(len(frames))/16)))
	nextTick := float64(frames[0].Tick)
	for _, frame := range frames {
		if float64(frame.Tick)+1e-9 < nextTick {
			continue
		}
		selected = append(selected, frame)
		for nextTick <= float64(frame.Tick)+1e-9 {
			nextTick += periodTicks
		}
	}
	return selected
}

func exposureResolutionSummary(resolution exposure.PackageResolution) ExposurePackageResolutionSummary {
	summary := ExposurePackageResolutionSummary{
		Status:          resolution.Status,
		ReasonCodes:     exposureSortedUniqueStrings(resolution.ReasonCodes),
		ContractVersion: exposure.ContractVersion,
	}
	if resolution.Package != nil {
		assetVersion := resolution.Package.AssetVersion
		packageSHA := resolution.PackageSHA256
		geometryRevision := resolution.Package.GeometryRevision
		summary.AssetVersion = &assetVersion
		summary.PackageSHA256 = &packageSHA
		summary.GeometryRevision = &geometryRevision
		if resolution.Support != nil {
			summary.ProductSupported = resolution.Support.ProductSupported
		}
	}
	return summary
}

func countExposureRecord(coverage *ExposureAngleCoverage, record ObservedExposureAngleRecord) {
	coverage.Records++
	if record.Geometry.Status == "observed" {
		coverage.GeometryObserved++
		coverage.SamplesHit[*record.Geometry.SampleID]++
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

func newExposureAngleEmitter(writer io.Writer) *exposureAngleEmitter {
	return &exposureAngleEmitter{writer: bufio.NewWriter(writer), hash: sha256.New()}
}

func (emitter *exposureAngleEmitter) emit(record ObservedExposureAngleRecord) error {
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

func (emitter *exposureAngleEmitter) close() error { return emitter.writer.Flush() }

func (emitter *exposureAngleEmitter) logicalSHA256() string {
	return hex.EncodeToString(emitter.hash.Sum(nil))
}

func sameExposureSectorWidths(left, right []exposure.Sector) bool {
	if len(left) != len(right) {
		return false
	}
	leftWidths := make(map[string]float64, len(left))
	for _, sector := range left {
		leftWidths[sector.ID] = sector.WidthDegrees
	}
	for _, sector := range right {
		if width, ok := leftWidths[sector.ID]; !ok || math.Abs(width-sector.WidthDegrees) > 1e-9 {
			return false
		}
	}
	return true
}

func exposureSortedUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func finiteExposureNumber(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func finiteExposureVector(value exposure.Vector3) bool {
	return finiteExposureNumber(value.X) && finiteExposureNumber(value.Y) && finiteExposureNumber(value.Z)
}
