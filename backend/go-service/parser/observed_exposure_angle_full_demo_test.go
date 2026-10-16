package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/exposure"
)

func TestObservedExposureAngleFullDemoIsDeterministicCausalAndTacticallyUnavailable(t *testing.T) {
	replay := exposureReplayFixture()
	resolution := exposureResolutionFixture(t)
	firstReport, firstRows := buildExposureFixture(t, replay, resolution)
	secondReport, secondRows := buildExposureFixture(t, replay, resolution)
	if !bytes.Equal(firstRows, secondRows) {
		t.Fatal("exposure diagnostic JSONL is not deterministic")
	}
	firstReport.RowsArtifact.Bytes = 123
	firstReport.RowsArtifact.SHA256 = strings.Repeat("a", 64)
	firstReport.BuildIdentifier = "fixture-build"
	if err := ValidateObservedExposureAngleFullDemoReport(firstReport); err != nil {
		t.Fatal(err)
	}
	if firstReport.Coverage.GeometryObserved == 0 || firstReport.Coverage.GeometryUnavailable == 0 ||
		firstReport.Coverage.TacticalUnavailable != firstReport.Coverage.Records ||
		firstReport.Coverage.TacticalReasonCounts[SpatialTacticalJudgmentUnavailableReason] != firstReport.Coverage.Records {
		t.Fatalf("unexpected coverage: %#v", firstReport.Coverage)
	}
	if firstReport.RowsArtifact.LogicalSHA256 != secondReport.RowsArtifact.LogicalSHA256 {
		t.Fatal("logical hashes differ")
	}
	digest := sha256.Sum256(firstRows)
	if firstReport.RowsArtifact.LogicalSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatal("logical JSONL hash does not independently recompute")
	}

	oracleOnly := exposureReplayFixture()
	oracleOnly.Rounds[0].Winner = "oracle-mutated"
	oracleOnly.Rounds[0].Events = []models.ReplayEvent{{Tick: 9999, Type: "future_outcome"}}
	_, oracleOnlyRows := buildExposureFixture(t, oracleOnly, resolution)
	if !bytes.Equal(firstRows, oracleOnlyRows) {
		t.Fatal("oracle/outcome mutation changed observed exposure rows")
	}

	mutated := exposureReplayFixture()
	mutated.Rounds[0].Winner = "oracle-mutated"
	mutated.Rounds[0].Events = []models.ReplayEvent{{Tick: 9999, Type: "future_outcome"}}
	mutated.Rounds[0].Frames[1].Players[0].X = 999
	_, mutatedRows := buildExposureFixture(t, mutated, resolution)
	if !bytes.Equal(bytes.Split(firstRows, []byte{'\n'})[0], bytes.Split(mutatedRows, []byte{'\n'})[0]) {
		t.Fatal("future/oracle mutation changed the earlier T0 record")
	}
}

func TestObservedExposureAngleIgnoresLegacyTacticalApproval(t *testing.T) {
	resolution := exposureResolutionFixture(t)
	resolution.Package.TacticalReview.Status = exposure.ReviewApproved
	player := exposureReplayFixture().Rounds[0].Frames[0].Players[0]
	record, err := BuildObservedExposureAngleRecord(
		"de_fixture", "demo-0123456789abcdef0123", "participant-001", 1, 100, player, resolution,
	)
	if err != nil {
		t.Fatal(err)
	}
	if record.Geometry.Status != "observed" || record.Geometry.FieldID != "FLD-GEOMETRIC-CLEARANCE-SECTORS" {
		t.Fatalf("objective fact changed with legacy tactical approval: %#v", record.Geometry)
	}
	if record.Tactical.Status != "unavailable" || len(record.Tactical.ReasonCodes) != 1 || record.Tactical.ReasonCodes[0] != SpatialTacticalJudgmentUnavailableReason {
		t.Fatalf("legacy tactical approval leaked into v2: %#v", record.Tactical)
	}
}

func TestObservedExposureAngleRecordFailsClosedForUnknownMapAndMapMismatch(t *testing.T) {
	player := exposureReplayFixture().Rounds[0].Frames[0].Players[0]
	unavailable := exposure.PackageResolution{
		Status: exposure.GeometryUnavailable, ReasonCodes: []string{exposure.ReasonMapNotRegistered},
	}
	record, err := BuildObservedExposureAngleRecord(
		"de_other", "demo-0123456789abcdef0123", "participant-001", 1, 100, player, unavailable,
	)
	if err != nil {
		t.Fatal(err)
	}
	if record.Geometry.Status != exposure.GeometryUnavailable ||
		record.Geometry.ReasonCodes[0] != exposure.ReasonMapNotRegistered || record.AssetVersion != nil {
		t.Fatalf("unknown-map record = %#v", record)
	}

	resolution := exposureResolutionFixture(t)
	replay := exposureReplayFixture()
	replay.Metadata.MapName = "de_other"
	var rows bytes.Buffer
	if _, err := BuildObservedExposureAngleFullDemoReport(
		replay, resolution, "demo-0123456789abcdef0123", strings.Repeat("a", 64), &rows,
	); err == nil {
		t.Fatal("map mismatch was accepted")
	}
}

func TestObservedExposureAngleRecordRejectsUnavailableAndAmbiguousInputs(t *testing.T) {
	resolution := exposureResolutionFixture(t)
	player := exposureReplayFixture().Rounds[0].Frames[0].Players[0]
	player.ViewAnglesAvailable = false
	record, err := BuildObservedExposureAngleRecord(
		"de_fixture", "demo-0123456789abcdef0123", "participant-001", 1, 100, player, resolution,
	)
	if err != nil || record.Geometry.ReasonCodes[0] != exposure.ReasonViewAngles {
		t.Fatalf("view unavailable record=%#v err=%v", record, err)
	}
	player.ViewAnglesAvailable = true
	player.X = 500
	record, err = BuildObservedExposureAngleRecord(
		"de_fixture", "demo-0123456789abcdef0123", "participant-001", 1, 100, player, resolution,
	)
	if err != nil || record.Geometry.ReasonCodes[0] != exposure.ReasonOutsideCoverage {
		t.Fatalf("outside record=%#v err=%v", record, err)
	}
}

func buildExposureFixture(
	t *testing.T,
	replay *models.ReplayData,
	resolution exposure.PackageResolution,
) (ObservedExposureAngleFullDemoReport, []byte) {
	t.Helper()
	var rows bytes.Buffer
	report, err := BuildObservedExposureAngleFullDemoReport(
		replay,
		resolution,
		"demo-0123456789abcdef0123",
		exposureFixtureSHA256(),
		&rows,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range bytes.Split(bytes.TrimSpace(rows.Bytes()), []byte{'\n'}) {
		var record ObservedExposureAngleRecord
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatal(err)
		}
		if err := validateObservedExposureAngleRecord(record); err != nil {
			t.Fatal(err)
		}
		if record.AvailabilityTick > record.T0Tick || strings.Contains(string(line), "steam") ||
			strings.Contains(string(line), "player-one") {
			t.Fatalf("causal/privacy violation: %s", line)
		}
	}
	return report, append([]byte(nil), rows.Bytes()...)
}

func exposureFixtureSHA256() string {
	return "0123456789abcdef0123" + strings.Repeat("a", 44)
}

func exposureReplayFixture() *models.ReplayData {
	playerOne := models.ReplayPlayerState{
		SteamID: 20, Name: "player-one", Alive: true, X: 0, Y: 0, Z: 0,
		Yaw: 359, ViewAnglesAvailable: true,
	}
	playerTwo := models.ReplayPlayerState{
		SteamID: 10, Name: "player-two", Alive: true, X: 500, Y: 500, Z: 0,
		Yaw: 0, ViewAnglesAvailable: true,
	}
	return &models.ReplayData{
		Metadata: models.ReplayMetadata{MapName: "de_fixture", TickRate: 64},
		Rounds: []models.ReplayRound{{
			Round: 1,
			Frames: []models.ReplayFrame{
				{Tick: 100, Players: []models.ReplayPlayerState{playerOne, playerTwo}},
				{Tick: 164, Players: []models.ReplayPlayerState{playerOne, playerTwo}},
			},
		}},
	}
}

func exposureResolutionFixture(t *testing.T) exposure.PackageResolution {
	t.Helper()
	rays := []exposure.RaySample{
		{AzimuthDegrees: 0, ClearanceMet: true},
		{AzimuthDegrees: 90, ClearanceMet: true},
		{AzimuthDegrees: 180, ClearanceMet: true},
		{AzimuthDegrees: 270, ClearanceMet: true},
	}
	sectors, err := exposure.BuildSectors("sample-001", rays, 90)
	if err != nil {
		t.Fatal(err)
	}
	author, reviewer := "generator", "validator"
	pkg := exposure.Package{
		SchemaID: exposure.PackageSchemaID, ContractVersion: exposure.ContractVersion,
		AssetVersion: "1.0.0-candidate.1", MapID: "de_fixture", GeometryRevision: strings.Repeat("b", 64),
		Coordinates: exposure.Coordinates{
			System: "source2_world", Units: "source2_world_units",
			AxisConvention: "right_handed_z_up_x_forward_y_left", OriginDescription: "fixture",
		},
		Angles: exposure.AngularConvention{
			Units: "degrees", ZeroDirection: "+x", PositiveDirection: "counter_clockwise_toward_+y",
			Normalization: "[0,360)", Interval: "start_inclusive_end_exclusive", SectorEncoding: "normalized_start_plus_width",
		},
		Resolution: exposure.Resolution{
			AzimuthStepDegrees: 90, MaximumRayDistance: 1000,
			MinimumClearanceDistance: 100, SamplingHeightUnits: 64,
		},
		Bounds: exposure.Bounds{
			Min: exposure.Vector3{X: -100, Y: -100, Z: -100},
			Max: exposure.Vector3{X: 100, Y: 100, Z: 100},
		},
		GeometryProvenance: []exposure.SourceAsset{
			{Role: "physics_gltf", Reference: "de_fixture/physics.gltf", SHA256: strings.Repeat("c", 64)},
			{Role: "physics_buffer", Reference: "de_fixture/physics.bin", SHA256: strings.Repeat("d", 64)},
			{Role: "fixture", Reference: "de_fixture/seed.json", SHA256: strings.Repeat("f", 64)},
		},
		GeometrySamples: []exposure.GeometrySample{{
			ID: "sample-001", Alias: "fixture-reference", Origin: exposure.Vector3{},
			SelectionRadiusUnits: 50, VerticalToleranceUnits: 50,
			OriginProvenance: exposure.OriginProvenance{
				SourceRole: "fixture", SourceReference: "de_fixture/seed.json",
				SourceCoordinate: "origin", Interpretation: "coordinate only",
			},
			Rays: rays, Sectors: sectors,
		}},
		Coverage: exposure.Coverage{Declaration: "partial_reference_samples", SampleCount: 1, SectorCount: 1},
		GeometryValidation: exposure.ReviewState{
			Status: exposure.ReviewApproved, AuthorID: &author, ReviewerID: &reviewer, Evidence: []string{"fixture"},
		},
		TacticalReview: exposure.ReviewState{Status: exposure.ReviewPending, Evidence: []string{}},
		TacticalAnnotations: []exposure.TacticalAnnotation{{
			ID: "annotation-001", GeometrySector: "sample-001/sector-001",
			Provenance: []string{"geometry_sector:sample-001/sector-001"}, Evidence: []string{}, ReviewStatus: exposure.ReviewPending,
		}},
		Limitations: []string{"fixture"},
	}
	if err := exposure.ValidatePackage(&pkg); err != nil {
		t.Fatal(err)
	}
	support := exposure.SupportEntry{
		MapID: "de_fixture", ContractVersion: exposure.ContractVersion,
		AssetVersion: pkg.AssetVersion, TacticalReviewStatus: exposure.ReviewPending,
	}
	return exposure.PackageResolution{
		Status: exposure.GeometryAvailable, Package: &pkg,
		PackageSHA256: strings.Repeat("e", 64), Support: &support,
	}
}
