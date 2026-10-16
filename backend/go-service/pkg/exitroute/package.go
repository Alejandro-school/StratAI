// Package exitroute owns the map-agnostic, fail-closed contract for objective
// candidate routes. It deliberately does not infer whether a route is safe,
// reasonable, an escape, or a recommendation.
package exitroute

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cs2-demo-service/pkg/navigation"
	"github.com/golang/geo/r3"
)

const (
	ContractVersion         = "1.0.0"
	PackageSchemaID         = "stratai.exit_route_map_package@1"
	SupportManifestSchemaID = "stratai.exit_route_support_manifest@1"
	SupportManifestFilename = "support_manifest.json"
	ReviewPending           = "pending_review"
	ReviewApproved          = "approved"
	Available               = "available"
	Unavailable             = "unavailable"

	ReasonMapNotRegistered = "map_not_registered"
	ReasonPackageMissing   = "package_file_missing"
	ReasonPackageHash      = "package_hash_mismatch"
	ReasonContractVersion  = "contract_version_incompatible"
	ReasonAssetVersion     = "asset_version_incompatible"
	ReasonMapMismatch      = "map_id_mismatch"
	ReasonPackageInvalid   = "package_invalid"
	ReasonSourceAsset      = "source_asset_mismatch"
	ReasonOutsideCoverage  = "outside_declared_sample_coverage"
	ReasonAmbiguousSample  = "ambiguous_sample_overlap"
	ReasonPosition         = "position_unavailable"
	ReasonHumanReview      = "human_review_pending"
)

var (
	mapIDPattern      = regexp.MustCompile(`^de_[a-z0-9_]+$`)
	versionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+-candidate\.[0-9]+$`)
	shaPattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	samplePattern     = regexp.MustCompile(`^exit-sample-[0-9]{3}$`)
	candidatePattern  = regexp.MustCompile(`^exit-sample-[0-9]{3}/candidate-[0-9]{3}$`)
	annotationPattern = regexp.MustCompile(`^exit-annotation-[0-9]{4}$`)
)

type Vector3 struct{ X, Y, Z float64 }

func (value Vector3) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	}{value.X, value.Y, value.Z})
}
func (value *Vector3) UnmarshalJSON(data []byte) error {
	var raw struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	value.X, value.Y, value.Z = raw.X, raw.Y, raw.Z
	return nil
}

type Bounds struct {
	Min Vector3 `json:"min"`
	Max Vector3 `json:"max"`
}
type Coordinates struct {
	System            string `json:"system"`
	Units             string `json:"units"`
	AxisConvention    string `json:"axis_convention"`
	OriginDescription string `json:"origin_description"`
	RouteDirection    string `json:"route_direction"`
}
type NavigationContract struct {
	GraphContractVersion      string  `json:"graph_contract_version"`
	ShortestPathAlgorithm     string  `json:"shortest_path_algorithm"`
	EndpointAssociationMethod string  `json:"endpoint_association_method"`
	EdgeWeightMethod          string  `json:"edge_weight_method"`
	EndpointTolerance         float64 `json:"endpoint_tolerance_world_units"`
	EndpointVerticalTolerance float64 `json:"endpoint_vertical_tolerance_world_units"`
}
type SourceAsset struct {
	Role      string `json:"role"`
	Reference string `json:"reference"`
	SHA256    string `json:"sha256"`
	Use       string `json:"use"`
}
type RelatedGeometryAudit struct {
	Role           string `json:"role"`
	SchemaID       string `json:"schema_id"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	SemanticUse    string `json:"semantic_use"`
}
type SeedProvenance struct {
	SourceRole       string `json:"source_role"`
	SourceReference  string `json:"source_reference"`
	SourceCoordinate string `json:"source_coordinate"`
	Interpretation   string `json:"interpretation"`
}
type ObjectiveConnectivity struct {
	Status                 string   `json:"status"`
	ReasonCodes            []string `json:"reason_codes"`
	DirectDistance         float64  `json:"direct_distance_world_units"`
	Reachable              *bool    `json:"reachable"`
	PathDistance           *float64 `json:"path_distance_world_units"`
	OriginAssociation      *float64 `json:"origin_association_world_units"`
	DestinationAssociation *float64 `json:"destination_association_world_units"`
}
type Candidate struct {
	ID                    string                `json:"id"`
	Alias                 string                `json:"alias"`
	Destination           Vector3               `json:"destination_world"`
	DestinationProvenance SeedProvenance        `json:"destination_provenance"`
	Direction             string                `json:"direction"`
	Objective             ObjectiveConnectivity `json:"objective_connectivity"`
}
type RouteSample struct {
	ID                string         `json:"id"`
	Alias             string         `json:"alias"`
	Origin            Vector3        `json:"origin_world"`
	SelectionRadius   float64        `json:"selection_radius_units"`
	VerticalTolerance float64        `json:"vertical_tolerance_units"`
	OriginProvenance  SeedProvenance `json:"origin_provenance"`
	Candidates        []Candidate    `json:"candidates"`
}
type Coverage struct {
	Declaration               string `json:"declaration"`
	FullMapCoverage           bool   `json:"full_map_coverage"`
	SampleCount               int    `json:"sample_count"`
	CandidateCount            int    `json:"candidate_count"`
	MinimumObservedCandidates int    `json:"minimum_observed_candidates"`
}
type ReviewState struct {
	Status     string   `json:"status"`
	AuthorID   *string  `json:"author_id"`
	ReviewerID *string  `json:"reviewer_id"`
	Evidence   []string `json:"evidence"`
}
type TacticalAnnotation struct {
	ID                string   `json:"id"`
	AnnotationVersion string   `json:"annotation_version"`
	CandidateRef      string   `json:"candidate_ref"`
	Classification    *string  `json:"classification"`
	Provenance        []string `json:"provenance"`
	Evidence          []string `json:"evidence"`
	AuthorID          *string  `json:"author_id"`
	ReviewerID        *string  `json:"reviewer_id"`
	ReviewStatus      string   `json:"review_status"`
}
type Package struct {
	SchemaID             string                 `json:"schema_id"`
	ContractVersion      string                 `json:"contract_version"`
	AssetVersion         string                 `json:"asset_version"`
	MapID                string                 `json:"map_id"`
	GeometryRevision     string                 `json:"geometry_revision"`
	Coordinates          Coordinates            `json:"coordinates"`
	Navigation           NavigationContract     `json:"navigation_contract"`
	Bounds               Bounds                 `json:"bounds"`
	SourceAssets         []SourceAsset          `json:"source_assets"`
	RelatedGeometryAudit []RelatedGeometryAudit `json:"related_geometry_audit"`
	RouteSamples         []RouteSample          `json:"route_samples"`
	Coverage             Coverage               `json:"coverage"`
	ObjectiveValidation  ReviewState            `json:"objective_validation"`
	TacticalReview       ReviewState            `json:"tactical_review"`
	TacticalAnnotations  []TacticalAnnotation   `json:"tactical_annotations"`
	Limitations          []string               `json:"limitations"`
}
type SupportEntry struct {
	MapID                 string `json:"map_id"`
	ContractVersion       string `json:"contract_version"`
	AssetVersion          string `json:"asset_version"`
	PackagePath           string `json:"package_path"`
	PackageSHA256         string `json:"package_sha256"`
	ImplementationStatus  string `json:"implementation_status"`
	ObjectiveStatus       string `json:"objective_status"`
	TacticalReviewStatus  string `json:"tactical_review_status"`
	ProductSupported      bool   `json:"product_supported"`
	TrainingUseAuthorized bool   `json:"training_use_authorized"`
}
type SupportManifest struct {
	SchemaID        string         `json:"schema_id"`
	ContractVersion string         `json:"contract_version"`
	Packages        []SupportEntry `json:"packages"`
}
type PackageResolution struct {
	Status        string
	ReasonCodes   []string
	Package       *Package
	PackageSHA256 string
	Support       *SupportEntry
}

func SelectSample(pkg *Package, point Vector3) (*RouteSample, string) {
	if pkg == nil || !finiteVector(point) {
		return nil, ReasonPosition
	}
	selected := -1
	for index, sample := range pkg.RouteSamples {
		if math.Hypot(point.X-sample.Origin.X, point.Y-sample.Origin.Y) <= sample.SelectionRadius && math.Abs(point.Z-sample.Origin.Z) <= sample.VerticalTolerance {
			if selected >= 0 {
				return nil, ReasonAmbiguousSample
			}
			selected = index
		}
	}
	if selected < 0 {
		return nil, ReasonOutsideCoverage
	}
	return &pkg.RouteSamples[selected], ""
}

func ResolvePackage(registryRoot, mapsRoot, mapID, contractVersion, assetVersion string) PackageResolution {
	if contractVersion != ContractVersion {
		return unavailable(ReasonContractVersion)
	}
	payload, err := os.ReadFile(filepath.Join(registryRoot, SupportManifestFilename))
	if err != nil {
		return unavailable(ReasonPackageMissing)
	}
	var manifest SupportManifest
	if decodeStrict(payload, &manifest) != nil || manifest.SchemaID != SupportManifestSchemaID || manifest.ContractVersion != ContractVersion {
		return unavailable(ReasonPackageInvalid)
	}
	seen := map[string]bool{}
	var support *SupportEntry
	for index := range manifest.Packages {
		entry := &manifest.Packages[index]
		if validateSupport(*entry) != nil || seen[entry.MapID] {
			return unavailable(ReasonPackageInvalid)
		}
		seen[entry.MapID] = true
		if entry.MapID == mapID {
			support = entry
		}
	}
	if support == nil {
		return unavailable(ReasonMapNotRegistered)
	}
	if support.ContractVersion != contractVersion {
		return unavailable(ReasonContractVersion)
	}
	if assetVersion != "" && support.AssetVersion != assetVersion {
		return unavailable(ReasonAssetVersion)
	}
	path, err := safeDescendant(registryRoot, support.PackagePath)
	if err != nil {
		return unavailable(ReasonPackageInvalid)
	}
	payload, err = os.ReadFile(path)
	if err != nil {
		return unavailable(ReasonPackageMissing)
	}
	digest := sha256.Sum256(payload)
	packageSHA := hex.EncodeToString(digest[:])
	if packageSHA != support.PackageSHA256 {
		return unavailable(ReasonPackageHash)
	}
	var pkg Package
	if decodeStrict(payload, &pkg) != nil || ValidatePackage(&pkg) != nil {
		return unavailable(ReasonPackageInvalid)
	}
	if pkg.MapID != mapID || pkg.MapID != support.MapID {
		return unavailable(ReasonMapMismatch)
	}
	if pkg.ContractVersion != support.ContractVersion {
		return unavailable(ReasonContractVersion)
	}
	if pkg.AssetVersion != support.AssetVersion {
		return unavailable(ReasonAssetVersion)
	}
	if pkg.TacticalReview.Status != support.TacticalReviewStatus || support.ProductSupported || support.TrainingUseAuthorized {
		return unavailable(ReasonPackageInvalid)
	}
	if ValidateSourceAssets(&pkg, mapsRoot) != nil {
		return unavailable(ReasonSourceAsset)
	}
	copy := *support
	return PackageResolution{Status: Available, Package: &pkg, PackageSHA256: packageSHA, Support: &copy}
}

func ValidatePackage(pkg *Package) error {
	if pkg == nil || pkg.SchemaID != PackageSchemaID || pkg.ContractVersion != ContractVersion || !versionPattern.MatchString(pkg.AssetVersion) || !mapIDPattern.MatchString(pkg.MapID) || !shaPattern.MatchString(pkg.GeometryRevision) {
		return errors.New("package identity is invalid")
	}
	if pkg.Coordinates.System != "source2_world" || pkg.Coordinates.Units != "source2_world_units" || pkg.Coordinates.AxisConvention != "right_handed_z_up_x_forward_y_left" || pkg.Coordinates.RouteDirection != "directed_origin_to_destination" || strings.TrimSpace(pkg.Coordinates.OriginDescription) == "" {
		return errors.New("coordinate contract is invalid")
	}
	config := navigation.DefaultConfig()
	if pkg.Navigation.GraphContractVersion != navigation.GraphContractVersion || pkg.Navigation.ShortestPathAlgorithm != navigation.ShortestPathAlgorithm || pkg.Navigation.EndpointAssociationMethod != navigation.EndpointAssociationMethod || pkg.Navigation.EdgeWeightMethod != navigation.EdgeWeightMethod || pkg.Navigation.EndpointTolerance != config.EndpointTolerance || pkg.Navigation.EndpointVerticalTolerance != config.EndpointVerticalTolerance {
		return errors.New("navigation contract is invalid")
	}
	if !validBounds(pkg.Bounds) {
		return errors.New("bounds are invalid")
	}
	roles := map[string]bool{}
	for _, source := range pkg.SourceAssets {
		if roles[source.Role] || !safeRelative(source.Reference) || !shaPattern.MatchString(source.SHA256) {
			return errors.New("source asset is invalid")
		}
		roles[source.Role] = true
		if !validSourceUse(source) {
			return errors.New("source asset use is invalid")
		}
	}
	for _, role := range []string{"nav_mesh", "physics_gltf", "physics_buffer", "sample_seed_catalog"} {
		if !roles[role] {
			return errors.New("source asset role is missing")
		}
	}
	if len(pkg.RelatedGeometryAudit) != 2 {
		return errors.New("related geometry audit is incomplete")
	}
	related := map[string]bool{}
	for _, item := range pkg.RelatedGeometryAudit {
		if related[item.Role] || (item.Role != "exposure_package" && item.Role != "cover_package") || strings.TrimSpace(item.SchemaID) == "" || !shaPattern.MatchString(item.ArtifactSHA256) || item.SemanticUse != "audited_separate_geometry_not_used_as_exit_route_truth" {
			return errors.New("related geometry audit is invalid")
		}
		related[item.Role] = true
	}
	if len(pkg.RouteSamples) == 0 || pkg.Coverage.Declaration != "partial_reference_samples" || pkg.Coverage.FullMapCoverage || pkg.Coverage.SampleCount != len(pkg.RouteSamples) || pkg.Coverage.MinimumObservedCandidates < 1 {
		return errors.New("coverage is invalid")
	}
	candidateRefs := map[string]bool{}
	aliases := map[string]bool{}
	candidateCount := 0
	for sampleIndex, sample := range pkg.RouteSamples {
		if sample.ID != fmt.Sprintf("exit-sample-%03d", sampleIndex+1) || !samplePattern.MatchString(sample.ID) || strings.TrimSpace(sample.Alias) == "" || aliases[sample.Alias] || !pointInBounds(sample.Origin, pkg.Bounds) || !finitePositive(sample.SelectionRadius) || !finitePositive(sample.VerticalTolerance) || !validSeed(sample.OriginProvenance) || len(sample.Candidates) == 0 {
			return errors.New("route sample is invalid")
		}
		aliases[sample.Alias] = true
		for candidateIndex, candidate := range sample.Candidates {
			want := fmt.Sprintf("%s/candidate-%03d", sample.ID, candidateIndex+1)
			if candidate.ID != want || !candidatePattern.MatchString(candidate.ID) || candidateRefs[candidate.ID] || strings.TrimSpace(candidate.Alias) == "" || aliases[candidate.Alias] || !pointInBounds(candidate.Destination, pkg.Bounds) || !validSeed(candidate.DestinationProvenance) || candidate.Direction != "origin_to_destination" || validateObjective(candidate.Objective) != nil {
				return errors.New("route candidate is invalid")
			}
			candidateRefs[candidate.ID] = true
			aliases[candidate.Alias] = true
			candidateCount++
		}
	}
	for left := 0; left < len(pkg.RouteSamples); left++ {
		for right := left + 1; right < len(pkg.RouteSamples); right++ {
			if volumesOverlap(pkg.RouteSamples[left], pkg.RouteSamples[right]) {
				return errors.New("sample volumes overlap ambiguously")
			}
		}
	}
	if pkg.Coverage.CandidateCount != candidateCount || pkg.Coverage.MinimumObservedCandidates > candidateCount {
		return errors.New("candidate coverage differs from package")
	}
	if validateReview(pkg.ObjectiveValidation, true) != nil || validateReview(pkg.TacticalReview, false) != nil {
		return errors.New("review state is invalid")
	}
	annotations := map[string]bool{}
	refs := map[string]bool{}
	for _, annotation := range pkg.TacticalAnnotations {
		if !annotationPattern.MatchString(annotation.ID) || annotations[annotation.ID] || !versionPattern.MatchString(annotation.AnnotationVersion) || !candidateRefs[annotation.CandidateRef] || refs[annotation.CandidateRef] || len(annotation.Provenance) == 0 {
			return errors.New("tactical annotation is invalid")
		}
		annotations[annotation.ID] = true
		refs[annotation.CandidateRef] = true
		if annotation.ReviewStatus == ReviewPending {
			if annotation.Classification != nil || annotation.AuthorID != nil || annotation.ReviewerID != nil || len(annotation.Evidence) != 0 {
				return errors.New("pending annotation contains fabricated review")
			}
		} else {
			return errors.New("candidate package may not assert human tactical review")
		}
	}
	if len(refs) != candidateCount || pkg.TacticalReview.Status != ReviewPending {
		return errors.New("tactical review coverage is invalid")
	}
	if len(pkg.Limitations) == 0 {
		return errors.New("limitations are missing")
	}
	return nil
}

func ValidateSourceAssets(pkg *Package, mapsRoot string) error {
	for _, source := range pkg.SourceAssets {
		path, err := safeDescendant(mapsRoot, source.Reference)
		if err != nil {
			return err
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(payload)
		if hex.EncodeToString(digest[:]) != source.SHA256 {
			return fmt.Errorf("source hash mismatch: %s", source.Role)
		}
	}
	return nil
}

// ValidateObjectiveRoutes independently recomputes every static candidate on
// the exact directed NAV graph and rejects stale or substituted measurements.
func ValidateObjectiveRoutes(pkg *Package, graph *navigation.Graph) error {
	if pkg == nil || graph == nil || graph.Version != navigation.GraphContractVersion {
		return errors.New("objective route graph is unavailable")
	}
	prepared, err := navigation.NewPreparedEvaluator(graph, navigation.DefaultConfig())
	if err != nil {
		return err
	}
	for _, sample := range pkg.RouteSamples {
		for _, candidate := range sample.Candidates {
			measurement, err := prepared.Evaluate(vector(sample.Origin), vector(candidate.Destination))
			if err != nil {
				return err
			}
			if !sameMeasurement(candidate.Objective, measurement) {
				return fmt.Errorf("objective route differs from NAV recomputation: %s", candidate.ID)
			}
		}
	}
	return nil
}

func ObjectiveFromMeasurement(value navigation.Measurement) ObjectiveConnectivity {
	status := "derived"
	reasons := []string{}
	if value.Reachable == nil {
		status = "unavailable"
	}
	if value.UnavailableReason != "" {
		reasons = []string{value.UnavailableReason}
	}
	return ObjectiveConnectivity{Status: status, ReasonCodes: reasons, DirectDistance: value.DirectDistance, Reachable: cloneBool(value.Reachable), PathDistance: cloneFloat(value.PathDistance), OriginAssociation: cloneFloat(value.OriginAssociationDistance), DestinationAssociation: cloneFloat(value.TargetAssociationDistance)}
}

func SortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	output := result[:0]
	for _, value := range result {
		if value != "" && (len(output) == 0 || output[len(output)-1] != value) {
			output = append(output, value)
		}
	}
	return output
}

func validateSupport(entry SupportEntry) error {
	expected := fmt.Sprintf("packages/%s/%s/package.json", entry.MapID, entry.AssetVersion)
	if !mapIDPattern.MatchString(entry.MapID) || entry.ContractVersion != ContractVersion || !versionPattern.MatchString(entry.AssetVersion) || entry.PackagePath != expected || !shaPattern.MatchString(entry.PackageSHA256) || entry.ImplementationStatus != "candidate" || entry.ObjectiveStatus != "validated" || (entry.TacticalReviewStatus != ReviewPending && entry.TacticalReviewStatus != ReviewApproved) || entry.ProductSupported || entry.TrainingUseAuthorized {
		return errors.New("support entry is invalid")
	}
	return nil
}
func validateObjective(value ObjectiveConnectivity) error {
	if !finite(value.DirectDistance) || value.DirectDistance < 0 || (value.Status != "derived" && value.Status != "unavailable") {
		return errors.New("objective is invalid")
	}
	if value.Reachable == nil {
		if value.Status != "unavailable" || value.PathDistance != nil || len(value.ReasonCodes) == 0 {
			return errors.New("unavailable objective is invalid")
		}
		return nil
	}
	if *value.Reachable {
		if value.Status != "derived" || value.PathDistance == nil || !finite(*value.PathDistance) || *value.PathDistance < 0 || value.OriginAssociation == nil || value.DestinationAssociation == nil {
			return errors.New("reachable objective is invalid")
		}
	} else if value.PathDistance != nil || len(value.ReasonCodes) == 0 {
		return errors.New("unreachable objective is invalid")
	}
	return nil
}
func validateReview(review ReviewState, approved bool) error {
	if approved {
		if review.Status != ReviewApproved || review.AuthorID == nil || review.ReviewerID == nil || len(review.Evidence) == 0 {
			return errors.New("approved review is incomplete")
		}
		return nil
	}
	if review.Status != ReviewPending || review.AuthorID != nil || review.ReviewerID != nil || len(review.Evidence) != 0 {
		return errors.New("pending review contains attribution")
	}
	return nil
}
func validSeed(value SeedProvenance) bool {
	return value.SourceRole == "sample_seed_catalog" && strings.TrimSpace(value.SourceReference) != "" && strings.TrimSpace(value.SourceCoordinate) != "" && value.Interpretation == "coordinate_seed_only_source_name_is_not_a_tactical_label"
}
func validSourceUse(source SourceAsset) bool {
	switch source.Role {
	case "nav_mesh":
		return source.Use == "objective_navigation"
	case "sample_seed_catalog":
		return source.Use == "coordinate_seeds_only"
	case "physics_gltf", "physics_buffer":
		return source.Use == "audited_collision_only_not_route_truth"
	}
	return false
}
func validBounds(value Bounds) bool {
	return finiteVector(value.Min) && finiteVector(value.Max) && value.Min.X < value.Max.X && value.Min.Y < value.Max.Y && value.Min.Z < value.Max.Z
}
func pointInBounds(point Vector3, bounds Bounds) bool {
	return finiteVector(point) && point.X >= bounds.Min.X && point.X <= bounds.Max.X && point.Y >= bounds.Min.Y && point.Y <= bounds.Max.Y && point.Z >= bounds.Min.Z && point.Z <= bounds.Max.Z
}
func volumesOverlap(left, right RouteSample) bool {
	return math.Hypot(left.Origin.X-right.Origin.X, left.Origin.Y-right.Origin.Y) < left.SelectionRadius+right.SelectionRadius && math.Abs(left.Origin.Z-right.Origin.Z) <= left.VerticalTolerance+right.VerticalTolerance
}
func sameMeasurement(expected ObjectiveConnectivity, actual navigation.Measurement) bool {
	computed := ObjectiveFromMeasurement(actual)
	return expected.Status == computed.Status && sameStrings(expected.ReasonCodes, computed.ReasonCodes) && close(expected.DirectDistance, computed.DirectDistance) && sameBool(expected.Reachable, computed.Reachable) && sameFloat(expected.PathDistance, computed.PathDistance) && sameFloat(expected.OriginAssociation, computed.OriginAssociation) && sameFloat(expected.DestinationAssociation, computed.DestinationAssociation)
}
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func sameBool(a, b *bool) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func sameFloat(a, b *float64) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && close(*a, *b))
}
func close(a, b float64) bool        { return math.Abs(a-b) <= 1e-6 }
func vector(value Vector3) r3.Vector { return r3.Vector{X: value.X, Y: value.Y, Z: value.Z} }
func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func finiteVector(value Vector3) bool   { return finite(value.X) && finite(value.Y) && finite(value.Z) }
func finite(value float64) bool         { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func finitePositive(value float64) bool { return finite(value) && value > 0 }
func safeRelative(reference string) bool {
	return reference != "" && !filepath.IsAbs(filepath.FromSlash(reference)) && !strings.Contains(reference, "\\") && !strings.Contains("/"+reference+"/", "/../")
}
func safeDescendant(root, reference string) (string, error) {
	if !safeRelative(reference) {
		return "", errors.New("unsafe reference")
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
		return "", errors.New("reference escapes root")
	}
	return candidate, nil
}
func decodeStrict(payload []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON")
		}
		return err
	}
	return nil
}
func unavailable(reason string) PackageResolution {
	return PackageResolution{Status: Unavailable, ReasonCodes: []string{reason}}
}
