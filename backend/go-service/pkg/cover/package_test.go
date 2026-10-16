package cover

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func testRegistryRoot() string {
	return filepath.Join("..", "..", "..", "..", "ai_coach", "spatial", "cover")
}

func testMapsRoot() string {
	return filepath.Join("..", "..", "..", "data", "maps")
}

func loadTestPackage(t *testing.T) *Package {
	t.Helper()
	path := filepath.Join(testRegistryRoot(), "packages", "de_mirage", "1.0.0-candidate.1", "package.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var pkg Package
	if err := json.Unmarshal(payload, &pkg); err != nil {
		t.Fatal(err)
	}
	return &pkg
}

func TestMiragePackageValidatesAsPartialObjectiveGeometry(t *testing.T) {
	resolution := ResolvePackage(testRegistryRoot(), testMapsRoot(), "de_mirage", ContractVersion, "")
	if resolution.Status != GeometryAvailable || resolution.Package == nil || resolution.Support == nil {
		t.Fatalf("unexpected resolution: %+v", resolution)
	}
	pkg := resolution.Package
	if err := ValidatePackage(pkg); err != nil {
		t.Fatal(err)
	}
	if len(pkg.GeometrySamples) != 8 || pkg.Coverage.DirectionCount != 192 || len(pkg.TacticalAnnotations) != 384 {
		t.Fatalf("unexpected package counts: %+v", pkg.Coverage)
	}
	if pkg.Coverage.FullMapCoverage || pkg.TacticalReview.Status != ReviewPending || resolution.Support.ProductSupported || resolution.Support.TrainingUseAuthorized {
		t.Fatal("partial candidate was promoted or tactically signed")
	}
	for _, annotation := range pkg.TacticalAnnotations {
		if annotation.ReviewStatus != ReviewPending || annotation.Classification != nil || annotation.AuthorID != nil || annotation.ReviewerID != nil || len(annotation.Evidence) != 0 {
			t.Fatal("pending annotation contains fabricated human review")
		}
	}
}

func TestProfileRequiresEveryHeightAndLateralProbe(t *testing.T) {
	pkg := loadTestPackage(t)
	direction := pkg.GeometrySamples[0].Directions[0]
	for _, postureID := range []string{"crouching", "standing"} {
		profile, ok := ProfileForPosture(direction, postureID)
		if !ok || profile.ProbeCount != 9 || len(profile.Probes) != 9 {
			t.Fatalf("missing nine-probe profile for %s", postureID)
		}
		count := 0
		for _, probe := range profile.Probes {
			if probe.OccludedWithinProtectionDistance {
				count++
			}
		}
		if profile.OccludedProbeCount != count || profile.CompleteVolumeOcclusion != (count == 9) {
			t.Fatalf("profile aggregate did not recompute: %+v", profile)
		}
	}
}

func TestSelectSampleBoundsAndAmbiguity(t *testing.T) {
	pkg := loadTestPackage(t)
	origin := pkg.GeometrySamples[0].Origin
	selected, reason := SelectSample(pkg, origin)
	if selected == nil || selected.ID != pkg.GeometrySamples[0].ID || reason != "" {
		t.Fatalf("sample origin was not selected: %s", reason)
	}
	if selected, reason = SelectSample(pkg, Vector3{X: 999999, Y: 999999}); selected != nil || reason != ReasonOutsideCoverage {
		t.Fatalf("outside point did not fail closed: %s", reason)
	}
	if selected, reason = SelectSample(pkg, Vector3{X: math.NaN()}); selected != nil || reason != ReasonPosition {
		t.Fatalf("non-finite point did not fail closed: %s", reason)
	}
	clone := *pkg
	clone.GeometrySamples = append([]GeometrySample(nil), pkg.GeometrySamples...)
	clone.GeometrySamples[1].Origin = clone.GeometrySamples[0].Origin
	if err := ValidatePackage(&clone); err == nil {
		t.Fatal("ambiguous overlapping selection volumes were accepted")
	}
}

func TestUnknownMapsAndVersionsNeverReceiveMirageFallback(t *testing.T) {
	for _, mapID := range []string{"de_dust2", "de_inferno", "de_ancient", "de_nuke", "de_anubis", "de_train", "de_unknown"} {
		resolution := ResolvePackage(testRegistryRoot(), testMapsRoot(), mapID, ContractVersion, "")
		if resolution.Status != GeometryUnavailable || len(resolution.ReasonCodes) != 1 || resolution.ReasonCodes[0] != ReasonMapNotRegistered || resolution.Package != nil {
			t.Fatalf("%s received unexpected package: %+v", mapID, resolution)
		}
	}
	if resolution := ResolvePackage(testRegistryRoot(), testMapsRoot(), "de_mirage", "2.0.0", ""); resolution.ReasonCodes[0] != ReasonContractVersion {
		t.Fatalf("incompatible contract did not fail closed: %+v", resolution)
	}
	if resolution := ResolvePackage(testRegistryRoot(), testMapsRoot(), "de_mirage", ContractVersion, "9.9.9"); resolution.ReasonCodes[0] != ReasonAssetVersion {
		t.Fatalf("incompatible asset did not fail closed: %+v", resolution)
	}
}

func TestNormalizeDegreesIsStableAtWrapBoundaries(t *testing.T) {
	for input, want := range map[float64]float64{-15: 345, 0: 0, 360: 0, 725: 5} {
		if got := NormalizeDegrees(input); got != want {
			t.Fatalf("NormalizeDegrees(%v)=%v want %v", input, got, want)
		}
	}
}
