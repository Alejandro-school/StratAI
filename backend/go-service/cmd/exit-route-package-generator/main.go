package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cs2-demo-service/pkg/exitroute"
	mapspkg "cs2-demo-service/pkg/maps"
	"cs2-demo-service/pkg/navigation"
	"github.com/golang/geo/r3"
)

type generationRequest struct {
	SchemaID        string `json:"schema_id"`
	ContractVersion string `json:"contract_version"`
	AssetVersion    string `json:"asset_version"`
	MapID           string `json:"map_id"`
	Sources         struct {
		NavMesh       string `json:"nav_mesh"`
		PhysicsGLTF   string `json:"physics_gltf"`
		PhysicsBuffer string `json:"physics_buffer"`
		SeedCatalog   string `json:"sample_seed_catalog"`
	} `json:"sources"`
	RelatedGeometry []exitroute.RelatedGeometryAudit `json:"related_geometry_audit"`
	Samples         []struct {
		ID                string  `json:"id"`
		Alias             string  `json:"alias"`
		OriginCoordinate  string  `json:"origin_coordinate"`
		SelectionRadius   float64 `json:"selection_radius_units"`
		VerticalTolerance float64 `json:"vertical_tolerance_units"`
		Destinations      []struct {
			ID               string `json:"id"`
			Alias            string `json:"alias"`
			SourceCoordinate string `json:"source_coordinate"`
		} `json:"destinations"`
	} `json:"samples"`
}
type seedCoordinate struct {
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Z    float64 `json:"z"`
}

func main() {
	var requestPath, mapsRoot, outputPath string
	flag.StringVar(&requestPath, "request", "", "generation request JSON")
	flag.StringVar(&mapsRoot, "maps-root", "", "read-only map asset root")
	flag.StringVar(&outputPath, "output", "", "new package JSON path")
	flag.Parse()
	if flag.NArg() != 0 || requestPath == "" || mapsRoot == "" || outputPath == "" {
		fmt.Fprintln(os.Stderr, "--request, --maps-root and --output are required; positional arguments are forbidden")
		os.Exit(2)
	}
	if _, err := os.Lstat(outputPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "output must not already exist")
		os.Exit(2)
	}
	pkg, graph, err := generate(requestPath, mapsRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate exit-route package: %v\n", err)
		os.Exit(1)
	}
	if err := exitroute.ValidatePackage(&pkg); err != nil {
		fmt.Fprintf(os.Stderr, "validate generated exit-route package: %v\n", err)
		os.Exit(1)
	}
	if err := exitroute.ValidateObjectiveRoutes(&pkg, graph); err != nil {
		fmt.Fprintf(os.Stderr, "recompute generated exit routes: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	payload, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputPath, append(payload, '\n'), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("generated %s %s with %d samples and %d directed candidates at %s\n", pkg.MapID, pkg.AssetVersion, pkg.Coverage.SampleCount, pkg.Coverage.CandidateCount, outputPath)
}

func generate(requestPath, mapsRoot string) (exitroute.Package, *navigation.Graph, error) {
	payload, err := os.ReadFile(requestPath)
	if err != nil {
		return exitroute.Package{}, nil, err
	}
	var request generationRequest
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return exitroute.Package{}, nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return exitroute.Package{}, nil, errors.New("generation request contains trailing JSON")
	}
	if request.SchemaID != "stratai.exit_route_generation_request@1" || request.ContractVersion != exitroute.ContractVersion || len(request.Samples) == 0 || len(request.RelatedGeometry) != 2 {
		return exitroute.Package{}, nil, errors.New("generation request identity or coverage is invalid")
	}
	refs := []struct{ role, reference, use string }{{"nav_mesh", request.Sources.NavMesh, "objective_navigation"}, {"physics_gltf", request.Sources.PhysicsGLTF, "audited_collision_only_not_route_truth"}, {"physics_buffer", request.Sources.PhysicsBuffer, "audited_collision_only_not_route_truth"}, {"sample_seed_catalog", request.Sources.SeedCatalog, "coordinate_seeds_only"}}
	sources := make([]exitroute.SourceAsset, 0, len(refs))
	for _, item := range refs {
		path, err := safeAsset(mapsRoot, item.reference)
		if err != nil {
			return exitroute.Package{}, nil, err
		}
		hash, err := sha256File(path)
		if err != nil {
			return exitroute.Package{}, nil, err
		}
		sources = append(sources, exitroute.SourceAsset{Role: item.role, Reference: item.reference, SHA256: hash, Use: item.use})
	}
	seedPath, err := safeAsset(mapsRoot, request.Sources.SeedCatalog)
	if err != nil {
		return exitroute.Package{}, nil, err
	}
	seedPayload, err := os.ReadFile(seedPath)
	if err != nil {
		return exitroute.Package{}, nil, err
	}
	var seedRows []seedCoordinate
	if err := json.Unmarshal(seedPayload, &seedRows); err != nil {
		return exitroute.Package{}, nil, err
	}
	seeds := map[string]seedCoordinate{}
	for _, seed := range seedRows {
		if _, ok := seeds[seed.Name]; ok {
			return exitroute.Package{}, nil, errors.New("duplicate seed coordinate")
		}
		seeds[seed.Name] = seed
	}
	manager := mapspkg.NewMapManager(mapsRoot)
	if err := manager.LoadMap(request.MapID); err != nil {
		return exitroute.Package{}, nil, fmt.Errorf("load map assets: %w", err)
	}
	graph, err := manager.NavigationGraphSnapshot()
	if err != nil || graph == nil {
		return exitroute.Package{}, nil, errors.New("directed NAV graph is unavailable")
	}
	prepared, err := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	if err != nil {
		return exitroute.Package{}, nil, err
	}
	bounds, err := graphBounds(graph)
	if err != nil {
		return exitroute.Package{}, nil, err
	}
	revisionDigest := sha256.Sum256([]byte(sources[0].SHA256 + "\n" + navigation.GraphContractVersion + "\n" + navigation.EndpointAssociationMethod + "\n"))
	author, reviewer := "stratai.exit-route-generator@1", "stratai.exit-route-package-validator@1"
	config := navigation.DefaultConfig()
	pkg := exitroute.Package{SchemaID: exitroute.PackageSchemaID, ContractVersion: request.ContractVersion, AssetVersion: request.AssetVersion, MapID: request.MapID, GeometryRevision: hex.EncodeToString(revisionDigest[:]), Coordinates: exitroute.Coordinates{System: "source2_world", Units: "source2_world_units", AxisConvention: "right_handed_z_up_x_forward_y_left", OriginDescription: "candidate endpoints preserve the Source 2 world frame", RouteDirection: "directed_origin_to_destination"}, Navigation: exitroute.NavigationContract{GraphContractVersion: navigation.GraphContractVersion, ShortestPathAlgorithm: navigation.ShortestPathAlgorithm, EndpointAssociationMethod: navigation.EndpointAssociationMethod, EdgeWeightMethod: navigation.EdgeWeightMethod, EndpointTolerance: config.EndpointTolerance, EndpointVerticalTolerance: config.EndpointVerticalTolerance}, Bounds: bounds, SourceAssets: sources, RelatedGeometryAudit: request.RelatedGeometry, Coverage: exitroute.Coverage{Declaration: "partial_reference_samples", FullMapCoverage: false, SampleCount: len(request.Samples), MinimumObservedCandidates: 1}, ObjectiveValidation: exitroute.ReviewState{Status: exitroute.ReviewApproved, AuthorID: &author, ReviewerID: &reviewer, Evidence: []string{"exact_nav_asset_sha256", "directed_endpoint_association_and_shortest_path_recomputation", "bounds_and_non_overlap_validation"}}, TacticalReview: exitroute.ReviewState{Status: exitroute.ReviewPending, Evidence: []string{}}, Limitations: []string{"NAV proves only polygon association, directed connectivity and path cost under its encoded graph", "the physics mesh is audited as collision geometry but is not accepted as navigability or tactical truth", "callout names are coordinate seeds only and are not labels of exits, safety or recommendations", "exposure and cover packages remain separate geometry and do not classify candidate routes", "partial reference volumes must not be extrapolated to full-map support", "all tactical classification remains unavailable until attributable human review"}}
	requests := append([]struct {
		ID                string  `json:"id"`
		Alias             string  `json:"alias"`
		OriginCoordinate  string  `json:"origin_coordinate"`
		SelectionRadius   float64 `json:"selection_radius_units"`
		VerticalTolerance float64 `json:"vertical_tolerance_units"`
		Destinations      []struct {
			ID               string `json:"id"`
			Alias            string `json:"alias"`
			SourceCoordinate string `json:"source_coordinate"`
		} `json:"destinations"`
	}{}, request.Samples...)
	sort.Slice(requests, func(i, j int) bool { return requests[i].ID < requests[j].ID })
	annotationIndex := 0
	seedReference := request.Sources.SeedCatalog
	interpretation := "coordinate_seed_only_source_name_is_not_a_tactical_label"
	for _, requested := range requests {
		origin, ok := seeds[requested.OriginCoordinate]
		if !ok {
			return exitroute.Package{}, nil, fmt.Errorf("unknown origin coordinate %s", requested.OriginCoordinate)
		}
		sample := exitroute.RouteSample{ID: requested.ID, Alias: requested.Alias, Origin: exitroute.Vector3{X: origin.X, Y: origin.Y, Z: origin.Z}, SelectionRadius: requested.SelectionRadius, VerticalTolerance: requested.VerticalTolerance, OriginProvenance: exitroute.SeedProvenance{SourceRole: "sample_seed_catalog", SourceReference: seedReference, SourceCoordinate: requested.OriginCoordinate, Interpretation: interpretation}}
		destinations := append([]struct {
			ID               string `json:"id"`
			Alias            string `json:"alias"`
			SourceCoordinate string `json:"source_coordinate"`
		}{}, requested.Destinations...)
		sort.Slice(destinations, func(i, j int) bool { return destinations[i].ID < destinations[j].ID })
		for _, destinationRequest := range destinations {
			destination, ok := seeds[destinationRequest.SourceCoordinate]
			if !ok {
				return exitroute.Package{}, nil, fmt.Errorf("unknown destination coordinate %s", destinationRequest.SourceCoordinate)
			}
			measurement, err := prepared.Evaluate(vector(sample.Origin), vector(exitroute.Vector3{X: destination.X, Y: destination.Y, Z: destination.Z}))
			if err != nil {
				return exitroute.Package{}, nil, err
			}
			candidate := exitroute.Candidate{ID: destinationRequest.ID, Alias: destinationRequest.Alias, Destination: exitroute.Vector3{X: destination.X, Y: destination.Y, Z: destination.Z}, DestinationProvenance: exitroute.SeedProvenance{SourceRole: "sample_seed_catalog", SourceReference: seedReference, SourceCoordinate: destinationRequest.SourceCoordinate, Interpretation: interpretation}, Direction: "origin_to_destination", Objective: exitroute.ObjectiveFromMeasurement(measurement)}
			sample.Candidates = append(sample.Candidates, candidate)
			pkg.Coverage.CandidateCount++
			annotationIndex++
			pkg.TacticalAnnotations = append(pkg.TacticalAnnotations, exitroute.TacticalAnnotation{ID: fmt.Sprintf("exit-annotation-%04d", annotationIndex), AnnotationVersion: request.AssetVersion, CandidateRef: candidate.ID, Provenance: []string{"objective_candidate:" + candidate.ID, "human_tactical_review_required"}, Evidence: []string{}, ReviewStatus: exitroute.ReviewPending})
		}
		pkg.RouteSamples = append(pkg.RouteSamples, sample)
	}
	return pkg, graph, nil
}

func graphBounds(graph *navigation.Graph) (exitroute.Bounds, error) {
	if graph == nil || len(graph.Nodes) == 0 {
		return exitroute.Bounds{}, errors.New("NAV graph has no nodes")
	}
	min := exitroute.Vector3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	max := exitroute.Vector3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	include := func(point r3.Vector) {
		min.X = math.Min(min.X, point.X)
		min.Y = math.Min(min.Y, point.Y)
		min.Z = math.Min(min.Z, point.Z)
		max.X = math.Max(max.X, point.X)
		max.Y = math.Max(max.Y, point.Y)
		max.Z = math.Max(max.Z, point.Z)
	}
	for _, node := range graph.Nodes {
		include(node.Position)
		for _, corner := range node.Corners {
			include(corner)
		}
	}
	return exitroute.Bounds{Min: min, Max: max}, nil
}
func vector(value exitroute.Vector3) r3.Vector { return r3.Vector{X: value.X, Y: value.Y, Z: value.Z} }
func safeAsset(root, reference string) (string, error) {
	if strings.TrimSpace(reference) == "" || filepath.IsAbs(filepath.FromSlash(reference)) || strings.Contains(reference, "\\") {
		return "", errors.New("source reference must be forward-slash relative")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(absoluteRoot, filepath.FromSlash(reference)))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(absoluteRoot, candidate)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("source reference escapes maps root")
	}
	return candidate, nil
}
func sha256File(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
