package exposure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSectorsWrapAroundAndHalfOpenBoundaries(t *testing.T) {
	rays := []RaySample{
		{AzimuthDegrees: 0, ClearanceMet: true},
		{AzimuthDegrees: 90, ClearanceMet: false, HitDistance: floatPointer(10)},
		{AzimuthDegrees: 180, ClearanceMet: false, HitDistance: floatPointer(10)},
		{AzimuthDegrees: 270, ClearanceMet: true},
	}
	sectors, err := BuildSectors("sample-001", rays, 90)
	if err != nil {
		t.Fatal(err)
	}
	if len(sectors) != 1 || sectors[0].StartDegrees != 225 || sectors[0].WidthDegrees != 180 {
		t.Fatalf("wrap sector = %#v", sectors)
	}
	for _, angle := range []float64{225, 359, 0, 44.999} {
		if !SectorContains(sectors[0], angle) {
			t.Fatalf("expected half-open wrap sector to contain %f", angle)
		}
	}
	if SectorContains(sectors[0], 45) || SectorContains(sectors[0], 224.999) {
		t.Fatal("half-open wrap boundary was not enforced")
	}
	relative := RelativeSectors(sectors, 225)
	if len(relative) != 1 || relative[0].StartDegrees != 0 || relative[0].WidthDegrees != 180 {
		t.Fatalf("relative sector = %#v", relative)
	}
}

func TestValidatePackageRejectsInvalidGeometryAndAmbiguity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Package)
	}{
		{name: "non finite", mutate: func(pkg *Package) { pkg.GeometrySamples[0].Origin.X = math.NaN() }},
		{name: "outside bounds", mutate: func(pkg *Package) { pkg.GeometrySamples[0].Origin.X = 101 }},
		{name: "sector tamper", mutate: func(pkg *Package) { pkg.GeometrySamples[0].Sectors[0].WidthDegrees = 12 }},
		{name: "ambiguous overlap", mutate: func(pkg *Package) {
			second := pkg.GeometrySamples[0]
			second.ID = "sample-002"
			second.Alias = "reference-002"
			for index := range second.Sectors {
				second.Sectors[index].ID = "sample-002/sector-001"
			}
			pkg.GeometrySamples = append(pkg.GeometrySamples, second)
			pkg.Coverage.SampleCount = 2
			pkg.Coverage.SectorCount = 2
			pkg.TacticalAnnotations = append(pkg.TacticalAnnotations, TacticalAnnotation{
				ID: "annotation-002", GeometrySector: "sample-002/sector-001",
				Provenance: []string{"geometry_sector:sample-002/sector-001"},
				Evidence:   []string{}, ReviewStatus: ReviewPending,
			})
		}},
		{name: "fabricated pending review", mutate: func(pkg *Package) {
			pkg.TacticalReview.AuthorID = testStringPointer("invented")
		}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			pkg := validTestPackage()
			testCase.mutate(&pkg)
			if err := ValidatePackage(&pkg); err == nil {
				t.Fatal("invalid package was accepted")
			}
		})
	}
}

func TestResolvePackageFailsClosedWithoutMirageFallback(t *testing.T) {
	registryRoot, mapsRoot := writeTestRegistry(t, validTestPackage())
	resolved := ResolvePackage(registryRoot, mapsRoot, "de_fixture", ContractVersion, "1.0.0-candidate.1")
	if resolved.Status != GeometryAvailable || resolved.Package == nil || resolved.Package.MapID != "de_fixture" {
		t.Fatalf("valid fixture package did not resolve: %#v", resolved)
	}
	for _, testCase := range []struct {
		name     string
		mapID    string
		contract string
		asset    string
		reason   string
	}{
		{name: "unknown map", mapID: "de_other", contract: ContractVersion, asset: "1.0.0-candidate.1", reason: ReasonMapNotRegistered},
		{name: "contract mismatch", mapID: "de_fixture", contract: "2.0.0", asset: "1.0.0-candidate.1", reason: ReasonContractVersion},
		{name: "asset mismatch", mapID: "de_fixture", contract: ContractVersion, asset: "9.9.9", reason: ReasonAssetVersion},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resolution := ResolvePackage(registryRoot, mapsRoot, testCase.mapID, testCase.contract, testCase.asset)
			if resolution.Status != GeometryUnavailable || resolution.Package != nil ||
				len(resolution.ReasonCodes) != 1 || resolution.ReasonCodes[0] != testCase.reason {
				t.Fatalf("resolution = %#v", resolution)
			}
		})
	}
}

func TestResolvePackageRejectsMapMismatchAndTampering(t *testing.T) {
	pkg := validTestPackage()
	registryRoot, mapsRoot := writeTestRegistry(t, pkg)
	packagePath := filepath.Join(registryRoot, "packages", "de_fixture", "1.0.0-candidate.1", "package.json")
	payload, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	payload[0] ^= 1
	if err := os.WriteFile(packagePath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	resolution := ResolvePackage(registryRoot, mapsRoot, "de_fixture", ContractVersion, "1.0.0-candidate.1")
	if resolution.Status != GeometryUnavailable || resolution.ReasonCodes[0] != ReasonPackageHash {
		t.Fatalf("tampered resolution = %#v", resolution)
	}

	registryRoot, mapsRoot = writeTestRegistry(t, pkg)
	packagePath = filepath.Join(registryRoot, "packages", "de_fixture", "1.0.0-candidate.1", "package.json")
	pkg.MapID = "de_mismatch"
	writeJSON(t, packagePath, pkg)
	manifestPath := filepath.Join(registryRoot, SupportManifestFilename)
	var manifest SupportManifest
	manifestPayload, _ := os.ReadFile(manifestPath)
	if err := json.Unmarshal(manifestPayload, &manifest); err != nil {
		t.Fatal(err)
	}
	packagePayload, _ := os.ReadFile(packagePath)
	packageDigest := sha256.Sum256(packagePayload)
	manifest.Packages[0].PackageSHA256 = hex.EncodeToString(packageDigest[:])
	writeJSON(t, manifestPath, manifest)
	resolution = ResolvePackage(registryRoot, mapsRoot, "de_fixture", ContractVersion, "1.0.0-candidate.1")
	if resolution.Status != GeometryUnavailable || resolution.ReasonCodes[0] != ReasonMapMismatch {
		t.Fatalf("map mismatch resolution = %#v", resolution)
	}
}

func TestSelectSampleIsDeterministicAndExplicitlyUnavailable(t *testing.T) {
	pkg := validTestPackage()
	sample, reason := SelectSample(&pkg, Vector3{})
	if sample == nil || reason != "" || sample.ID != "sample-001" {
		t.Fatalf("sample=%#v reason=%q", sample, reason)
	}
	if sample, reason = SelectSample(&pkg, Vector3{X: 90, Y: 90}); sample != nil || reason != ReasonOutsideCoverage {
		t.Fatalf("outside sample=%#v reason=%q", sample, reason)
	}
	if sample, reason = SelectSample(&pkg, Vector3{X: math.Inf(1)}); sample != nil || reason != ReasonPosition {
		t.Fatalf("invalid sample=%#v reason=%q", sample, reason)
	}
}

func validTestPackage() Package {
	rays := []RaySample{
		{AzimuthDegrees: 0, ClearanceMet: true},
		{AzimuthDegrees: 90, ClearanceMet: true},
		{AzimuthDegrees: 180, ClearanceMet: true},
		{AzimuthDegrees: 270, ClearanceMet: true},
	}
	sectors, _ := BuildSectors("sample-001", rays, 90)
	return Package{
		SchemaID: PackageSchemaID, ContractVersion: ContractVersion,
		AssetVersion: "1.0.0-candidate.1", MapID: "de_fixture",
		GeometryRevision: strings.Repeat("0", 64),
		Coordinates: Coordinates{
			System: "source2_world", Units: "source2_world_units",
			AxisConvention: "right_handed_z_up_x_forward_y_left", OriginDescription: "fixture",
		},
		Angles: AngularConvention{
			Units: "degrees", ZeroDirection: "+x", PositiveDirection: "counter_clockwise_toward_+y",
			Normalization: "[0,360)", Interval: "start_inclusive_end_exclusive",
			SectorEncoding: "normalized_start_plus_width",
		},
		Resolution: Resolution{
			AzimuthStepDegrees: 90, MaximumRayDistance: 1000,
			MinimumClearanceDistance: 100, SamplingHeightUnits: 64,
		},
		Bounds: Bounds{Min: Vector3{-100, -100, -100}, Max: Vector3{100, 100, 100}},
		GeometryProvenance: []SourceAsset{
			{Role: "physics_gltf", Reference: "de_fixture/physics.gltf", SHA256: strings.Repeat("0", 64)},
			{Role: "physics_buffer", Reference: "de_fixture/physics.bin", SHA256: strings.Repeat("0", 64)},
			{Role: "fixture", Reference: "de_fixture/seed.json", SHA256: strings.Repeat("0", 64)},
		},
		GeometrySamples: []GeometrySample{{
			ID: "sample-001", Alias: "reference-001", Origin: Vector3{},
			SelectionRadiusUnits: 10, VerticalToleranceUnits: 10,
			OriginProvenance: OriginProvenance{
				SourceRole: "fixture", SourceReference: "de_fixture/seed.json",
				SourceCoordinate: "origin", Interpretation: "coordinate only",
			},
			Rays: rays, Sectors: sectors,
		}},
		Coverage: Coverage{Declaration: "partial_reference_samples", SampleCount: 1, SectorCount: 1},
		GeometryValidation: ReviewState{
			Status: ReviewApproved, AuthorID: testStringPointer("generator"), ReviewerID: testStringPointer("validator"),
			Evidence: []string{"fixture"},
		},
		TacticalReview: ReviewState{Status: ReviewPending, Evidence: []string{}},
		TacticalAnnotations: []TacticalAnnotation{{
			ID: "annotation-001", GeometrySector: "sample-001/sector-001",
			Provenance: []string{"geometry_sector:sample-001/sector-001"}, Evidence: []string{}, ReviewStatus: ReviewPending,
		}},
		Limitations: []string{"fixture only"},
	}
}

func writeTestRegistry(t *testing.T, pkg Package) (string, string) {
	t.Helper()
	registryRoot := t.TempDir()
	mapsRoot := t.TempDir()
	for index := range pkg.GeometryProvenance {
		path := filepath.Join(mapsRoot, filepath.FromSlash(pkg.GeometryProvenance[index].Reference))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		payload := []byte(pkg.GeometryProvenance[index].Role)
		if err := os.WriteFile(path, payload, 0o600); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		pkg.GeometryProvenance[index].SHA256 = hex.EncodeToString(digest[:])
	}
	packageRelative := filepath.ToSlash(filepath.Join("packages", pkg.MapID, pkg.AssetVersion, "package.json"))
	packagePath := filepath.Join(registryRoot, filepath.FromSlash(packageRelative))
	writeJSON(t, packagePath, pkg)
	packagePayload, _ := os.ReadFile(packagePath)
	digest := sha256.Sum256(packagePayload)
	manifest := SupportManifest{
		SchemaID: SupportManifestSchemaID, ContractVersion: ContractVersion,
		Packages: []SupportEntry{{
			MapID: pkg.MapID, ContractVersion: pkg.ContractVersion, AssetVersion: pkg.AssetVersion,
			PackagePath: packageRelative, PackageSHA256: hex.EncodeToString(digest[:]),
			ImplementationStatus: "candidate", GeometryStatus: "validated",
			TacticalReviewStatus: ReviewPending, ProductSupported: false, TrainingUseAuthorized: false,
		}},
	}
	writeJSON(t, filepath.Join(registryRoot, SupportManifestFilename), manifest)
	return registryRoot, mapsRoot
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func floatPointer(value float64) *float64    { return &value }
func testStringPointer(value string) *string { return &value }
