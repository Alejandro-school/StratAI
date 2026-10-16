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

	"cs2-demo-service/pkg/exposure"
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
	Resolution exposure.Resolution `json:"resolution"`
	Samples    []struct {
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
	var requestPath string
	var mapsRoot string
	var outputPath string
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
		fmt.Fprintf(os.Stderr, "generate exposure package: %v\n", err)
		os.Exit(1)
	}
	if err := exposure.ValidatePackage(&pkg); err != nil {
		fmt.Fprintf(os.Stderr, "validate generated exposure package: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}
	payload, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode exposure package: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputPath, append(payload, '\n'), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write exposure package: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("generated %s %s with %d samples at %s\n", pkg.MapID, pkg.AssetVersion, len(pkg.GeometrySamples), outputPath)
}

func generate(requestPath, mapsRoot string) (exposure.Package, error) {
	payload, err := os.ReadFile(requestPath)
	if err != nil {
		return exposure.Package{}, err
	}
	var request generationRequest
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return exposure.Package{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return exposure.Package{}, errors.New("generation request contains trailing JSON")
		}
		return exposure.Package{}, fmt.Errorf("generation request contains trailing JSON: %w", err)
	}
	if request.SchemaID != "stratai.exposure_angle_generation_request@1" ||
		request.ContractVersion != exposure.ContractVersion || len(request.Samples) == 0 {
		return exposure.Package{}, errors.New("generation request identity or samples are invalid")
	}

	gltfPath, err := safeAsset(mapsRoot, request.Sources.PhysicsGLTF)
	if err != nil {
		return exposure.Package{}, err
	}
	binPath, err := safeAsset(mapsRoot, request.Sources.PhysicsBin)
	if err != nil {
		return exposure.Package{}, err
	}
	seedPath, err := safeAsset(mapsRoot, request.Sources.SeedCatalog)
	if err != nil {
		return exposure.Package{}, err
	}
	mesh, err := geometry.LoadGLTF(gltfPath)
	if err != nil {
		return exposure.Package{}, fmt.Errorf("load physics GLTF: %w", err)
	}
	if mesh == nil || len(mesh.Triangles) == 0 || mesh.BVH == nil {
		return exposure.Package{}, errors.New("physics GLTF yielded no raycast geometry")
	}
	seedPayload, err := os.ReadFile(seedPath)
	if err != nil {
		return exposure.Package{}, err
	}
	var seeds []seedCoordinate
	if err := json.Unmarshal(seedPayload, &seeds); err != nil {
		return exposure.Package{}, err
	}
	seedByName := make(map[string]seedCoordinate, len(seeds))
	for _, seed := range seeds {
		if _, duplicate := seedByName[seed.Name]; duplicate {
			return exposure.Package{}, errors.New("seed coordinate names are not unique")
		}
		seedByName[seed.Name] = seed
	}

	bounds := geometry.CalculateBounds(mesh.Triangles)
	pkg := exposure.Package{
		SchemaID:        exposure.PackageSchemaID,
		ContractVersion: request.ContractVersion,
		AssetVersion:    request.AssetVersion,
		MapID:           request.MapID,
		Coordinates: exposure.Coordinates{
			System:            "source2_world",
			Units:             "source2_world_units",
			AxisConvention:    "right_handed_z_up_x_forward_y_left",
			OriginDescription: "map package coordinates are preserved in the Source 2 world frame",
		},
		Angles: exposure.AngularConvention{
			Units:             "degrees",
			ZeroDirection:     "+x",
			PositiveDirection: "counter_clockwise_toward_+y",
			Normalization:     "[0,360)",
			Interval:          "start_inclusive_end_exclusive",
			SectorEncoding:    "normalized_start_plus_width",
		},
		Resolution: request.Resolution,
		Bounds: exposure.Bounds{
			Min: exposure.Vector3{X: bounds.Min.X, Y: bounds.Min.Y, Z: bounds.Min.Z},
			Max: exposure.Vector3{X: bounds.Max.X, Y: bounds.Max.Y, Z: bounds.Max.Z},
		},
		Coverage: exposure.Coverage{
			Declaration:     "partial_reference_samples",
			FullMapCoverage: false,
			SampleCount:     len(request.Samples),
		},
		GeometryValidation: exposure.ReviewState{
			Status:     exposure.ReviewApproved,
			AuthorID:   stringPointer("stratai.geometry-generator@1"),
			ReviewerID: stringPointer("stratai.exposure-package-validator@1"),
			Evidence:   []string{"physics_gltf_sha256", "physics_buffer_sha256", "ray_grid_recomputation"},
		},
		TacticalReview: exposure.ReviewState{
			Status:   exposure.ReviewPending,
			Evidence: []string{},
		},
		Limitations: []string{
			"geometry sectors encode horizontal ray clearance only and are not tactical exposure labels",
			"sample coverage is partial and must not be extrapolated outside declared selection volumes",
			"cover and exit-route semantics are outside this package",
			"tactical use remains unavailable until independently attributable human review is approved",
		},
	}
	gltfHash, err := sha256File(gltfPath)
	if err != nil {
		return exposure.Package{}, err
	}
	binHash, err := sha256File(binPath)
	if err != nil {
		return exposure.Package{}, err
	}
	seedHash, err := sha256File(seedPath)
	if err != nil {
		return exposure.Package{}, err
	}
	revision := sha256.Sum256([]byte(gltfHash + "\n" + binHash + "\n"))
	pkg.GeometryRevision = hex.EncodeToString(revision[:])
	pkg.GeometryProvenance = []exposure.SourceAsset{
		{Role: "physics_gltf", Reference: request.Sources.PhysicsGLTF, SHA256: gltfHash},
		{Role: "physics_buffer", Reference: request.Sources.PhysicsBin, SHA256: binHash},
		{Role: "sample_seed_catalog", Reference: request.Sources.SeedCatalog, SHA256: seedHash},
	}

	requests := append([]struct {
		ID                     string  `json:"id"`
		Alias                  string  `json:"alias"`
		SourceCoordinate       string  `json:"source_coordinate"`
		SelectionRadiusUnits   float64 `json:"selection_radius_units"`
		VerticalToleranceUnits float64 `json:"vertical_tolerance_units"`
	}{}, request.Samples...)
	sort.Slice(requests, func(i, j int) bool { return requests[i].ID < requests[j].ID })
	annotationIndex := 0
	for _, requested := range requests {
		seed, ok := seedByName[requested.SourceCoordinate]
		if !ok {
			return exposure.Package{}, fmt.Errorf("sample %s references an unknown seed coordinate", requested.ID)
		}
		sample := exposure.GeometrySample{
			ID:                     requested.ID,
			Alias:                  requested.Alias,
			Origin:                 exposure.Vector3{X: seed.X, Y: seed.Y, Z: seed.Z},
			SelectionRadiusUnits:   requested.SelectionRadiusUnits,
			VerticalToleranceUnits: requested.VerticalToleranceUnits,
			OriginProvenance: exposure.OriginProvenance{
				SourceRole:       "sample_seed_catalog",
				SourceReference:  request.Sources.SeedCatalog,
				SourceCoordinate: requested.SourceCoordinate,
				Interpretation:   "coordinate seed only; source name is not a tactical label",
			},
		}
		rayCount := int(math.Round(360 / request.Resolution.AzimuthStepDegrees))
		if rayCount < 1 || math.Abs(float64(rayCount)*request.Resolution.AzimuthStepDegrees-360) > 1e-9 {
			return exposure.Package{}, errors.New("azimuth step must divide 360 exactly")
		}
		rayOrigin := r3.Vector{X: seed.X, Y: seed.Y, Z: seed.Z + request.Resolution.SamplingHeightUnits}
		for rayIndex := 0; rayIndex < rayCount; rayIndex++ {
			azimuth := float64(rayIndex) * request.Resolution.AzimuthStepDegrees
			radians := azimuth * math.Pi / 180
			direction := r3.Vector{X: math.Cos(radians), Y: math.Sin(radians), Z: 0}
			distance, _ := mesh.RayCast(rayOrigin, direction, request.Resolution.MaximumRayDistance)
			var hitDistance *float64
			if distance >= 0 {
				rounded := math.Round(distance*1e6) / 1e6
				hitDistance = &rounded
			}
			sample.Rays = append(sample.Rays, exposure.RaySample{
				AzimuthDegrees: azimuth,
				HitDistance:    hitDistance,
				ClearanceMet:   hitDistance == nil || *hitDistance >= request.Resolution.MinimumClearanceDistance,
			})
		}
		sectors, err := exposure.BuildSectors(sample.ID, sample.Rays, request.Resolution.AzimuthStepDegrees)
		if err != nil {
			return exposure.Package{}, err
		}
		sample.Sectors = sectors
		pkg.GeometrySamples = append(pkg.GeometrySamples, sample)
		pkg.Coverage.SectorCount += len(sectors)
		for _, sector := range sectors {
			annotationIndex++
			pkg.TacticalAnnotations = append(pkg.TacticalAnnotations, exposure.TacticalAnnotation{
				ID:             fmt.Sprintf("annotation-%03d", annotationIndex),
				GeometrySector: sector.ID,
				Provenance:     []string{"geometry_sector:" + sector.ID},
				Evidence:       []string{},
				ReviewStatus:   exposure.ReviewPending,
			})
		}
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
