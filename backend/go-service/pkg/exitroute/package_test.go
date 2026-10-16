package exitroute

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cs2-demo-service/pkg/navigation"
	"github.com/golang/geo/r3"
)

func TestObjectiveRoutesAreDirectedRecomputableAndTacticalReviewStaysPending(t *testing.T) {
	graph := testGraph([]navigation.Edge{{FromID: 1, ToID: 2, Weight: 100}})
	pkg := testPackage(t, graph)
	if err := ValidatePackage(&pkg); err != nil {
		t.Fatalf("validate package: %v", err)
	}
	if err := ValidateObjectiveRoutes(&pkg, graph); err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if pkg.TacticalReview.Status != ReviewPending || pkg.TacticalAnnotations[0].Classification != nil || pkg.TacticalAnnotations[0].AuthorID != nil || pkg.TacticalAnnotations[0].ReviewerID != nil || len(pkg.TacticalAnnotations[0].Evidence) != 0 {
		t.Fatal("pending human review was fabricated")
	}

	reverse := testGraph([]navigation.Edge{{FromID: 2, ToID: 1, Weight: 100}})
	if err := ValidateObjectiveRoutes(&pkg, reverse); err == nil {
		t.Fatal("direction reversal must change the objective result")
	}
	disconnected := testGraph(nil)
	if err := ValidateObjectiveRoutes(&pkg, disconnected); err == nil {
		t.Fatal("disconnected graph must reject stale connected result")
	}
}

func TestValidatePackageRejectsCoverageBoundsOverlapAndFabricatedReview(t *testing.T) {
	graph := testGraph([]navigation.Edge{{FromID: 1, ToID: 2, Weight: 100}})
	tests := []struct {
		name   string
		mutate func(*Package)
	}{
		{"destination outside bounds", func(pkg *Package) { pkg.RouteSamples[0].Candidates[0].Destination.X = 999 }},
		{"coverage mismatch", func(pkg *Package) { pkg.Coverage.CandidateCount = 2 }},
		{"overlap", func(pkg *Package) {
			second := pkg.RouteSamples[0]
			second.ID = "exit-sample-002"
			second.Alias = "sample-002"
			second.Candidates[0].ID = "exit-sample-002/candidate-001"
			second.Candidates[0].Alias = "candidate-002"
			pkg.RouteSamples = append(pkg.RouteSamples, second)
			pkg.Coverage.SampleCount = 2
			pkg.Coverage.CandidateCount = 2
			pkg.TacticalAnnotations = append(pkg.TacticalAnnotations, TacticalAnnotation{ID: "exit-annotation-0002", AnnotationVersion: pkg.AssetVersion, CandidateRef: second.Candidates[0].ID, Provenance: []string{"human_review_required"}, Evidence: []string{}, ReviewStatus: ReviewPending})
		}},
		{"fabricated human label", func(pkg *Package) { label := "safe"; pkg.TacticalAnnotations[0].Classification = &label }},
		{"broken candidate ref", func(pkg *Package) { pkg.TacticalAnnotations[0].CandidateRef = "exit-sample-999/candidate-001" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := testPackage(t, graph)
			test.mutate(&pkg)
			if err := ValidatePackage(&pkg); err == nil {
				t.Fatal("invalid package accepted")
			}
		})
	}
}

func TestSelectSampleFailsClosedOutsideAndOnAmbiguity(t *testing.T) {
	pkg := testPackage(t, testGraph([]navigation.Edge{{FromID: 1, ToID: 2, Weight: 100}}))
	if sample, reason := SelectSample(&pkg, Vector3{X: 500, Y: 500, Z: 0}); sample != nil || reason != ReasonOutsideCoverage {
		t.Fatalf("outside result = %#v %s", sample, reason)
	}
	second := pkg.RouteSamples[0]
	second.ID = "exit-sample-002"
	second.Alias = "sample-002"
	pkg.RouteSamples = append(pkg.RouteSamples, second)
	if sample, reason := SelectSample(&pkg, Vector3{X: 0, Y: 0, Z: 0}); sample != nil || reason != ReasonAmbiguousSample {
		t.Fatalf("ambiguous result = %#v %s", sample, reason)
	}
}

func TestResolvePackageFailsClosedForRegistryPackageAndSourceIdentity(t *testing.T) {
	graph := testGraph([]navigation.Edge{{FromID: 1, ToID: 2, Weight: 100}})
	pkg := testPackage(t, graph)
	registry, maps := writeRegistry(t, pkg)
	resolved := ResolvePackage(registry, maps, "de_test", ContractVersion, pkg.AssetVersion)
	if resolved.Status != Available {
		t.Fatalf("resolve: %#v", resolved)
	}
	if got := ResolvePackage(registry, maps, "de_unknown", ContractVersion, ""); got.Status != Unavailable || got.ReasonCodes[0] != ReasonMapNotRegistered {
		t.Fatalf("unknown map: %#v", got)
	}
	if got := ResolvePackage(registry, maps, "de_test", "9.0.0", ""); got.ReasonCodes[0] != ReasonContractVersion {
		t.Fatalf("contract: %#v", got)
	}
	if got := ResolvePackage(registry, maps, "de_test", ContractVersion, "9.0.0-candidate.1"); got.ReasonCodes[0] != ReasonAssetVersion {
		t.Fatalf("asset version: %#v", got)
	}
	if err := os.WriteFile(filepath.Join(maps, "de_test", "map.nav"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ResolvePackage(registry, maps, "de_test", ContractVersion, ""); got.ReasonCodes[0] != ReasonSourceAsset {
		t.Fatalf("source hash: %#v", got)
	}
}

func testGraph(edges []navigation.Edge) *navigation.Graph {
	return &navigation.Graph{Version: navigation.GraphContractVersion, AssetReference: "de_test/map.nav", AssetSHA256: hex.EncodeToString(make([]byte, 32)), Nodes: []navigation.Node{{ID: 1, Position: r3.Vector{X: 0, Y: 0, Z: 0}, Corners: []r3.Vector{{X: -10, Y: -10}, {X: 10, Y: -10}, {X: 10, Y: 10}, {X: -10, Y: 10}}}, {ID: 2, Position: r3.Vector{X: 100, Y: 0, Z: 0}, Corners: []r3.Vector{{X: 90, Y: -10}, {X: 110, Y: -10}, {X: 110, Y: 10}, {X: 90, Y: 10}}}}, Edges: edges}
}
func testPackage(t *testing.T, graph *navigation.Graph) Package {
	t.Helper()
	measurement, err := navigation.Evaluate(graph, r3.Vector{}, r3.Vector{X: 100}, navigation.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	author, reviewer := "generator", "validator"
	sourceHash := hashBytes([]byte("nav"))
	return Package{SchemaID: PackageSchemaID, ContractVersion: ContractVersion, AssetVersion: "1.0.0-candidate.1", MapID: "de_test", GeometryRevision: hashBytes([]byte("revision")), Coordinates: Coordinates{System: "source2_world", Units: "source2_world_units", AxisConvention: "right_handed_z_up_x_forward_y_left", OriginDescription: "test", RouteDirection: "directed_origin_to_destination"}, Navigation: NavigationContract{GraphContractVersion: navigation.GraphContractVersion, ShortestPathAlgorithm: navigation.ShortestPathAlgorithm, EndpointAssociationMethod: navigation.EndpointAssociationMethod, EdgeWeightMethod: navigation.EdgeWeightMethod, EndpointTolerance: 96, EndpointVerticalTolerance: 128}, Bounds: Bounds{Min: Vector3{X: -20, Y: -20, Z: -20}, Max: Vector3{X: 120, Y: 20, Z: 20}}, SourceAssets: []SourceAsset{{Role: "nav_mesh", Reference: "de_test/map.nav", SHA256: sourceHash, Use: "objective_navigation"}, {Role: "physics_gltf", Reference: "de_test/map.gltf", SHA256: sourceHash, Use: "audited_collision_only_not_route_truth"}, {Role: "physics_buffer", Reference: "de_test/map.bin", SHA256: sourceHash, Use: "audited_collision_only_not_route_truth"}, {Role: "sample_seed_catalog", Reference: "de_test/places.json", SHA256: sourceHash, Use: "coordinate_seeds_only"}}, RelatedGeometryAudit: []RelatedGeometryAudit{{Role: "exposure_package", SchemaID: "exposure@1", ArtifactSHA256: hashBytes([]byte("exposure")), SemanticUse: "audited_separate_geometry_not_used_as_exit_route_truth"}, {Role: "cover_package", SchemaID: "cover@1", ArtifactSHA256: hashBytes([]byte("cover")), SemanticUse: "audited_separate_geometry_not_used_as_exit_route_truth"}}, RouteSamples: []RouteSample{{ID: "exit-sample-001", Alias: "sample-001", Origin: Vector3{}, SelectionRadius: 20, VerticalTolerance: 20, OriginProvenance: testSeed(), Candidates: []Candidate{{ID: "exit-sample-001/candidate-001", Alias: "candidate-001", Destination: Vector3{X: 100}, DestinationProvenance: testSeed(), Direction: "origin_to_destination", Objective: ObjectiveFromMeasurement(measurement)}}}}, Coverage: Coverage{Declaration: "partial_reference_samples", SampleCount: 1, CandidateCount: 1, MinimumObservedCandidates: 1}, ObjectiveValidation: ReviewState{Status: ReviewApproved, AuthorID: &author, ReviewerID: &reviewer, Evidence: []string{"recomputed"}}, TacticalReview: ReviewState{Status: ReviewPending, Evidence: []string{}}, TacticalAnnotations: []TacticalAnnotation{{ID: "exit-annotation-0001", AnnotationVersion: "1.0.0-candidate.1", CandidateRef: "exit-sample-001/candidate-001", Provenance: []string{"human_review_required"}, Evidence: []string{}, ReviewStatus: ReviewPending}}, Limitations: []string{"objective only"}}
}
func testSeed() SeedProvenance {
	return SeedProvenance{SourceRole: "sample_seed_catalog", SourceReference: "de_test/places.json", SourceCoordinate: "seed", Interpretation: "coordinate_seed_only_source_name_is_not_a_tactical_label"}
}
func writeRegistry(t *testing.T, pkg Package) (string, string) {
	t.Helper()
	root := t.TempDir()
	registry := filepath.Join(root, "registry")
	maps := filepath.Join(root, "maps")
	if err := os.MkdirAll(filepath.Join(registry, "packages", "de_test", pkg.AssetVersion), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(maps, "de_test"), 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"map.nav", "map.gltf", "map.bin", "places.json"} {
		if err := os.WriteFile(filepath.Join(maps, "de_test", name), []byte("nav"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	payload, err := json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.Join(registry, "packages", "de_test", pkg.AssetVersion, "package.json")
	if err := os.WriteFile(packagePath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := SupportManifest{SchemaID: SupportManifestSchemaID, ContractVersion: ContractVersion, Packages: []SupportEntry{{MapID: "de_test", ContractVersion: ContractVersion, AssetVersion: pkg.AssetVersion, PackagePath: "packages/de_test/" + pkg.AssetVersion + "/package.json", PackageSHA256: hashBytes(payload), ImplementationStatus: "candidate", ObjectiveStatus: "validated", TacticalReviewStatus: ReviewPending}}}
	manifestPayload, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(registry, SupportManifestFilename), manifestPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	return registry, maps
}
func hashBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
