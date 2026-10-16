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

	"cs2-demo-service/pkg/cover"
	"cs2-demo-service/pkg/geometry"
	"github.com/golang/geo/r3"
)

type generationRequest struct {
	SchemaID        string `json:"schema_id"`
	ContractVersion string `json:"contract_version"`
	AssetVersion    string `json:"asset_version"`
	MapID           string `json:"map_id"`
	Sources         struct {
		PhysicsGLTF string `json:"physics_gltf"`
		PhysicsBin  string `json:"physics_buffer"`
		SeedCatalog string `json:"sample_seed_catalog"`
	} `json:"sources"`
	PlayerVolume cover.PlayerVolume `json:"player_volume"`
	Resolution   cover.Resolution   `json:"resolution"`
	Samples      []struct {
		ID                     string  `json:"id"`
		Alias                  string  `json:"alias"`
		SourceCoordinate       string  `json:"source_coordinate"`
		SelectionRadiusUnits   float64 `json:"selection_radius_units"`
		VerticalToleranceUnits float64 `json:"vertical_tolerance_units"`
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
	pkg, err := generate(requestPath, mapsRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate cover package: %v\n", err)
		os.Exit(1)
	}
	if err := cover.ValidatePackage(&pkg); err != nil {
		fmt.Fprintf(os.Stderr, "validate generated cover package: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}
	payload, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode cover package: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputPath, append(payload, '\n'), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write cover package: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("generated %s %s with %d samples and %d directions at %s\n", pkg.MapID, pkg.AssetVersion, len(pkg.GeometrySamples), pkg.Coverage.DirectionCount, outputPath)
}

func generate(requestPath, mapsRoot string) (cover.Package, error) {
	payload, err := os.ReadFile(requestPath)
	if err != nil {
		return cover.Package{}, err
	}
	var request generationRequest
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return cover.Package{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return cover.Package{}, errors.New("generation request contains trailing JSON")
		}
		return cover.Package{}, err
	}
	if request.SchemaID != "stratai.cover_generation_request@1" || request.ContractVersion != cover.ContractVersion || len(request.Samples) == 0 {
		return cover.Package{}, errors.New("generation request identity or samples are invalid")
	}
	gltfPath, err := safeAsset(mapsRoot, request.Sources.PhysicsGLTF)
	if err != nil {
		return cover.Package{}, err
	}
	binPath, err := safeAsset(mapsRoot, request.Sources.PhysicsBin)
	if err != nil {
		return cover.Package{}, err
	}
	seedPath, err := safeAsset(mapsRoot, request.Sources.SeedCatalog)
	if err != nil {
		return cover.Package{}, err
	}
	mesh, err := geometry.LoadGLTF(gltfPath)
	if err != nil {
		return cover.Package{}, fmt.Errorf("load physics GLTF: %w", err)
	}
	if mesh == nil || len(mesh.Triangles) == 0 || mesh.BVH == nil {
		return cover.Package{}, errors.New("physics GLTF yielded no raycast geometry")
	}
	seedPayload, err := os.ReadFile(seedPath)
	if err != nil {
		return cover.Package{}, err
	}
	var seeds []seedCoordinate
	if err := json.Unmarshal(seedPayload, &seeds); err != nil {
		return cover.Package{}, err
	}
	seedByName := map[string]seedCoordinate{}
	for _, seed := range seeds {
		if _, duplicate := seedByName[seed.Name]; duplicate {
			return cover.Package{}, errors.New("seed coordinate names are not unique")
		}
		seedByName[seed.Name] = seed
	}
	bounds := geometry.CalculateBounds(mesh.Triangles)
	gltfHash, err := sha256File(gltfPath)
	if err != nil {
		return cover.Package{}, err
	}
	binHash, err := sha256File(binPath)
	if err != nil {
		return cover.Package{}, err
	}
	seedHash, err := sha256File(seedPath)
	if err != nil {
		return cover.Package{}, err
	}
	revision := sha256.Sum256([]byte(gltfHash + "\n" + binHash + "\n"))
	pkg := cover.Package{SchemaID: cover.PackageSchemaID, ContractVersion: request.ContractVersion, AssetVersion: request.AssetVersion, MapID: request.MapID, GeometryRevision: hex.EncodeToString(revision[:]), Coordinates: cover.Coordinates{System: "source2_world", Units: "source2_world_units", AxisConvention: "right_handed_z_up_x_forward_y_left", OriginDescription: "map package coordinates are preserved in the Source 2 world frame"}, DirectionConvention: cover.DirectionConvention{Units: "degrees", ZeroDirection: "+x", PositiveDirection: "counter_clockwise_toward_+y", Normalization: "[0,360)", Meaning: "direction_from_player_volume_toward_possible_incoming_line"}, PlayerVolume: request.PlayerVolume, Resolution: request.Resolution, Bounds: cover.Bounds{Min: cover.Vector3{X: bounds.Min.X, Y: bounds.Min.Y, Z: bounds.Min.Z}, Max: cover.Vector3{X: bounds.Max.X, Y: bounds.Max.Y, Z: bounds.Max.Z}}, GeometryProvenance: []cover.SourceAsset{{Role: "physics_gltf", Reference: request.Sources.PhysicsGLTF, SHA256: gltfHash}, {Role: "physics_buffer", Reference: request.Sources.PhysicsBin, SHA256: binHash}, {Role: "sample_seed_catalog", Reference: request.Sources.SeedCatalog, SHA256: seedHash}}, Coverage: cover.Coverage{Declaration: "partial_reference_samples", FullMapCoverage: false, SampleCount: len(request.Samples), PostureIDs: postureIDs(request.PlayerVolume.Postures)}, GeometryValidation: cover.ReviewState{Status: cover.ReviewApproved, AuthorID: stringPointer("stratai.geometry-generator@1"), ReviewerID: stringPointer("stratai.cover-package-validator@1"), Evidence: []string{"physics_gltf_sha256", "physics_buffer_sha256", "multi_height_multi_offset_probe_recomputation"}}, TacticalReview: cover.ReviewState{Status: cover.ReviewPending, Evidence: []string{}}, Limitations: []string{"volume occlusion is objective geometry only and is not a tactical cover label", "every direction requires all declared height and lateral probes; one blocked line never establishes complete volume occlusion", "sample coverage is partial and must not be extrapolated outside declared selection volumes", "exit routes, enemy state, tactical risk, decisions and recommendations are outside this package", "tactical cover remains unavailable until independently attributable human review is approved"}}
	requests := append([]struct {
		ID                     string  `json:"id"`
		Alias                  string  `json:"alias"`
		SourceCoordinate       string  `json:"source_coordinate"`
		SelectionRadiusUnits   float64 `json:"selection_radius_units"`
		VerticalToleranceUnits float64 `json:"vertical_tolerance_units"`
	}{}, request.Samples...)
	sort.Slice(requests, func(i, j int) bool { return requests[i].ID < requests[j].ID })
	directionSlots := int(math.Round(360 / request.Resolution.AzimuthStepDegrees))
	if directionSlots < 1 || math.Abs(float64(directionSlots)*request.Resolution.AzimuthStepDegrees-360) > 1e-9 {
		return cover.Package{}, errors.New("azimuth step must divide 360 exactly")
	}
	annotationIndex := 0
	for _, requested := range requests {
		seed, ok := seedByName[requested.SourceCoordinate]
		if !ok {
			return cover.Package{}, fmt.Errorf("sample %s references an unknown seed coordinate", requested.ID)
		}
		sample := cover.GeometrySample{ID: requested.ID, Alias: requested.Alias, Origin: cover.Vector3{X: seed.X, Y: seed.Y, Z: seed.Z}, SelectionRadiusUnits: requested.SelectionRadiusUnits, VerticalToleranceUnits: requested.VerticalToleranceUnits, OriginProvenance: cover.OriginProvenance{SourceRole: "sample_seed_catalog", SourceReference: request.Sources.SeedCatalog, SourceCoordinate: requested.SourceCoordinate, Interpretation: "coordinate seed only; source name is not a tactical label"}}
		for directionIndex := 0; directionIndex < directionSlots; directionIndex++ {
			azimuth := float64(directionIndex) * request.Resolution.AzimuthStepDegrees
			radians := azimuth * math.Pi / 180
			directionVector := r3.Vector{X: math.Cos(radians), Y: math.Sin(radians), Z: 0}
			perpendicular := r3.Vector{X: -math.Sin(radians), Y: math.Cos(radians), Z: 0}
			direction := cover.DirectionSample{ID: fmt.Sprintf("%s/direction-%03d", sample.ID, directionIndex), AzimuthDegrees: azimuth}
			for _, posture := range request.PlayerVolume.Postures {
				profile := cover.OcclusionProfile{PostureID: posture.ID}
				for _, height := range posture.ProbeHeightsUnits {
					for _, offset := range request.PlayerVolume.LateralOffsetsUnits {
						origin := r3.Vector{X: seed.X + perpendicular.X*offset, Y: seed.Y + perpendicular.Y*offset, Z: seed.Z + height}
						distance, _ := mesh.RayCast(origin, directionVector, request.Resolution.MaximumRayDistanceUnits)
						var hit *float64
						if distance >= 0 {
							rounded := math.Round(distance*1e6) / 1e6
							hit = &rounded
						}
						occluded := hit != nil && *hit <= request.Resolution.ProtectionDistanceUnits+1e-9
						profile.Probes = append(profile.Probes, cover.ProbeResult{HeightUnits: height, LateralOffsetUnits: offset, HitDistanceUnits: hit, OccludedWithinProtectionDistance: occluded})
						if occluded {
							profile.OccludedProbeCount++
						}
					}
				}
				profile.ProbeCount = len(profile.Probes)
				profile.CompleteVolumeOcclusion = profile.OccludedProbeCount == profile.ProbeCount
				direction.Profiles = append(direction.Profiles, profile)
				annotationIndex++
				directionRef := direction.ID
				pkg.TacticalAnnotations = append(pkg.TacticalAnnotations, cover.TacticalAnnotation{ID: fmt.Sprintf("cover-annotation-%04d", annotationIndex), AnnotationVersion: request.AssetVersion, GeometryDirectionRef: directionRef, PostureID: posture.ID, Provenance: []string{"geometry_profile:" + directionRef + "#" + posture.ID}, Evidence: []string{}, ReviewStatus: cover.ReviewPending})
			}
			sample.Directions = append(sample.Directions, direction)
			pkg.Coverage.DirectionCount++
		}
		pkg.GeometrySamples = append(pkg.GeometrySamples, sample)
	}
	return pkg, nil
}

func safeAsset(root, reference string) (string, error) {
	if strings.TrimSpace(reference) == "" || filepath.IsAbs(filepath.FromSlash(reference)) || strings.Contains(reference, "\\") {
		return "", errors.New("source reference must be a forward-slash relative path")
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
func stringPointer(value string) *string { return &value }
func postureIDs(postures []cover.Posture) []string {
	result := make([]string, len(postures))
	for index := range postures {
		result[index] = postures[index].ID
	}
	return result
}
