package cover

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
)

const (
	ContractVersion           = "1.0.0"
	PackageSchemaID           = "stratai.cover_map_package@1"
	SupportManifestSchemaID   = "stratai.cover_support_manifest@1"
	SupportManifestFilename   = "support_manifest.json"
	ReviewPending             = "pending_review"
	ReviewApproved            = "approved"
	GeometryAvailable         = "available"
	GeometryUnavailable       = "unavailable"
	ReasonMapNotRegistered    = "map_not_registered"
	ReasonPackageMissing      = "package_file_missing"
	ReasonPackageHash         = "package_hash_mismatch"
	ReasonContractVersion     = "contract_version_incompatible"
	ReasonAssetVersion        = "asset_version_incompatible"
	ReasonMapMismatch         = "map_id_mismatch"
	ReasonPackageInvalid      = "package_invalid"
	ReasonGeometrySource      = "geometry_source_mismatch"
	ReasonOutsideCoverage     = "outside_declared_sample_coverage"
	ReasonAmbiguousSample     = "ambiguous_sample_overlap"
	ReasonPosition            = "position_unavailable"
	ReasonViewAngles          = "view_angles_unavailable"
	ReasonHumanReviewPending  = "human_review_pending"
	ReasonExitRouteOutOfScope = "exit_route_out_of_scope"
)

var (
	mapIDPattern       = regexp.MustCompile(`^de_[a-z0-9_]+$`)
	versionPattern     = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+-candidate\.[0-9]+$`)
	sha256Pattern      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	sampleIDPattern    = regexp.MustCompile(`^cover-sample-[0-9]{3}$`)
	directionIDPattern = regexp.MustCompile(`^cover-sample-[0-9]{3}/direction-[0-9]{3}$`)
	annotationPattern  = regexp.MustCompile(`^cover-annotation-[0-9]{4}$`)
	postureIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
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
}
type DirectionConvention struct {
	Units             string `json:"units"`
	ZeroDirection     string `json:"zero_direction"`
	PositiveDirection string `json:"positive_direction"`
	Normalization     string `json:"normalization"`
	Meaning           string `json:"meaning"`
}
type Posture struct {
	ID                string    `json:"id"`
	VolumeHeightUnits float64   `json:"volume_height_units"`
	ProbeHeightsUnits []float64 `json:"probe_heights_units"`
}
type PlayerVolume struct {
	ModelID             string    `json:"model_id"`
	RadiusUnits         float64   `json:"radius_units"`
	LateralOffsetsUnits []float64 `json:"lateral_offsets_units"`
	Postures            []Posture `json:"postures"`
}
type Resolution struct {
	AzimuthStepDegrees      float64 `json:"azimuth_step_degrees"`
	MaximumRayDistanceUnits float64 `json:"maximum_ray_distance_units"`
	ProtectionDistanceUnits float64 `json:"protection_distance_units"`
}
type SourceAsset struct {
	Role      string `json:"role"`
	Reference string `json:"reference"`
	SHA256    string `json:"sha256"`
}
type ProbeResult struct {
	HeightUnits                      float64  `json:"height_units"`
	LateralOffsetUnits               float64  `json:"lateral_offset_units"`
	HitDistanceUnits                 *float64 `json:"hit_distance_units"`
	OccludedWithinProtectionDistance bool     `json:"occluded_within_protection_distance"`
}
type OcclusionProfile struct {
	PostureID               string        `json:"posture_id"`
	ProbeCount              int           `json:"probe_count"`
	OccludedProbeCount      int           `json:"occluded_probe_count"`
	CompleteVolumeOcclusion bool          `json:"complete_volume_occlusion"`
	Probes                  []ProbeResult `json:"probes"`
}
type DirectionSample struct {
	ID             string             `json:"id"`
	AzimuthDegrees float64            `json:"azimuth_degrees"`
	Profiles       []OcclusionProfile `json:"profiles"`
}
type OriginProvenance struct {
	SourceRole       string `json:"source_role"`
	SourceReference  string `json:"source_reference"`
	SourceCoordinate string `json:"source_coordinate"`
	Interpretation   string `json:"interpretation"`
}
type GeometrySample struct {
	ID                     string            `json:"id"`
	Alias                  string            `json:"alias"`
	Origin                 Vector3           `json:"origin_world"`
	SelectionRadiusUnits   float64           `json:"selection_radius_units"`
	VerticalToleranceUnits float64           `json:"vertical_tolerance_units"`
	OriginProvenance       OriginProvenance  `json:"origin_provenance"`
	Directions             []DirectionSample `json:"directions"`
}
type ReviewState struct {
	Status     string   `json:"status"`
	AuthorID   *string  `json:"author_id"`
	ReviewerID *string  `json:"reviewer_id"`
	Evidence   []string `json:"evidence"`
}
type TacticalAnnotation struct {
	ID                   string   `json:"id"`
	AnnotationVersion    string   `json:"annotation_version"`
	GeometryDirectionRef string   `json:"geometry_direction_ref"`
	PostureID            string   `json:"posture_id"`
	Classification       *string  `json:"classification"`
	Provenance           []string `json:"provenance"`
	Evidence             []string `json:"evidence"`
	AuthorID             *string  `json:"author_id"`
	ReviewerID           *string  `json:"reviewer_id"`
	ReviewStatus         string   `json:"review_status"`
}
type Coverage struct {
	Declaration     string   `json:"declaration"`
	FullMapCoverage bool     `json:"full_map_coverage"`
	SampleCount     int      `json:"sample_count"`
	DirectionCount  int      `json:"direction_count"`
	PostureIDs      []string `json:"posture_ids"`
}

type Package struct {
	SchemaID            string               `json:"schema_id"`
	ContractVersion     string               `json:"contract_version"`
	AssetVersion        string               `json:"asset_version"`
	MapID               string               `json:"map_id"`
	GeometryRevision    string               `json:"geometry_revision"`
	Coordinates         Coordinates          `json:"coordinates"`
	DirectionConvention DirectionConvention  `json:"direction_convention"`
	PlayerVolume        PlayerVolume         `json:"player_volume"`
	Resolution          Resolution           `json:"resolution"`
	Bounds              Bounds               `json:"bounds"`
	GeometryProvenance  []SourceAsset        `json:"geometry_provenance"`
	GeometrySamples     []GeometrySample     `json:"geometry_samples"`
	Coverage            Coverage             `json:"coverage"`
	GeometryValidation  ReviewState          `json:"geometry_validation"`
	TacticalReview      ReviewState          `json:"tactical_review"`
	TacticalAnnotations []TacticalAnnotation `json:"tactical_annotations"`
	Limitations         []string             `json:"limitations"`
}

type SupportEntry struct {
	MapID                 string `json:"map_id"`
	ContractVersion       string `json:"contract_version"`
	AssetVersion          string `json:"asset_version"`
	PackagePath           string `json:"package_path"`
	PackageSHA256         string `json:"package_sha256"`
	ImplementationStatus  string `json:"implementation_status"`
	GeometryStatus        string `json:"geometry_status"`
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

func NormalizeDegrees(value float64) float64 {
	result := math.Mod(value, 360)
	if result < 0 {
		result += 360
	}
	if result == 360 || math.Abs(result) < 1e-12 {
		return 0
	}
	return result
}

func ProfileForPosture(direction DirectionSample, postureID string) (*OcclusionProfile, bool) {
	for index := range direction.Profiles {
		if direction.Profiles[index].PostureID == postureID {
			return &direction.Profiles[index], true
		}
	}
	return nil, false
}

func SelectSample(pkg *Package, point Vector3) (*GeometrySample, string) {
	if pkg == nil || !finiteVector(point) {
		return nil, ReasonPosition
	}
	candidates := make([]int, 0, 1)
	for index, sample := range pkg.GeometrySamples {
		if math.Hypot(point.X-sample.Origin.X, point.Y-sample.Origin.Y) <= sample.SelectionRadiusUnits && math.Abs(point.Z-sample.Origin.Z) <= sample.VerticalToleranceUnits {
			candidates = append(candidates, index)
		}
	}
	if len(candidates) == 0 {
		return nil, ReasonOutsideCoverage
	}
	if len(candidates) > 1 {
		return nil, ReasonAmbiguousSample
	}
	return &pkg.GeometrySamples[candidates[0]], ""
}

func ResolvePackage(registryRoot, mapsRoot, mapID, requiredContractVersion, requiredAssetVersion string) PackageResolution {
	if requiredContractVersion != ContractVersion {
		return unavailable(ReasonContractVersion)
	}
	payload, err := os.ReadFile(filepath.Join(registryRoot, SupportManifestFilename))
	if err != nil {
		return unavailable(ReasonPackageMissing)
	}
	var manifest SupportManifest
	if err := decodeStrict(payload, &manifest); err != nil || manifest.SchemaID != SupportManifestSchemaID || manifest.ContractVersion != ContractVersion {
		return unavailable(ReasonPackageInvalid)
	}
	seen := map[string]struct{}{}
	var support *SupportEntry
	for index := range manifest.Packages {
		entry := &manifest.Packages[index]
		if err := validateSupportEntry(*entry); err != nil {
			return unavailable(ReasonPackageInvalid)
		}
		if _, duplicate := seen[entry.MapID]; duplicate {
			return unavailable(ReasonPackageInvalid)
		}
		seen[entry.MapID] = struct{}{}
		if entry.MapID == mapID {
			support = entry
		}
	}
	if support == nil {
		return unavailable(ReasonMapNotRegistered)
	}
	if support.ContractVersion != requiredContractVersion {
		return unavailable(ReasonContractVersion)
	}
	if requiredAssetVersion != "" && support.AssetVersion != requiredAssetVersion {
		return unavailable(ReasonAssetVersion)
	}
	packagePath, err := safeDescendant(registryRoot, support.PackagePath)
	if err != nil {
		return unavailable(ReasonPackageInvalid)
	}
	packagePayload, err := os.ReadFile(packagePath)
	if err != nil {
		return unavailable(ReasonPackageMissing)
	}
	digest := sha256.Sum256(packagePayload)
	packageSHA := hex.EncodeToString(digest[:])
	if !sha256Pattern.MatchString(support.PackageSHA256) || support.PackageSHA256 != packageSHA {
		return unavailable(ReasonPackageHash)
	}
	var pkg Package
	if err := decodeStrict(packagePayload, &pkg); err != nil {
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
	if err := ValidatePackage(&pkg); err != nil {
		return unavailable(ReasonPackageInvalid)
	}
	if pkg.TacticalReview.Status != support.TacticalReviewStatus || (support.ProductSupported && pkg.TacticalReview.Status != ReviewApproved) {
		return unavailable(ReasonPackageInvalid)
	}
	if err := ValidateSourceAssets(&pkg, mapsRoot); err != nil {
		return unavailable(ReasonGeometrySource)
	}
	copySupport := *support
	return PackageResolution{Status: GeometryAvailable, Package: &pkg, PackageSHA256: packageSHA, Support: &copySupport}
}

func ValidatePackage(pkg *Package) error {
	if pkg == nil {
		return errors.New("package is nil")
	}
	if pkg.SchemaID != PackageSchemaID || pkg.ContractVersion != ContractVersion || !versionPattern.MatchString(pkg.AssetVersion) || !mapIDPattern.MatchString(pkg.MapID) || !sha256Pattern.MatchString(pkg.GeometryRevision) {
		return errors.New("package identity is invalid")
	}
	if pkg.Coordinates.System != "source2_world" || pkg.Coordinates.Units != "source2_world_units" || pkg.Coordinates.AxisConvention != "right_handed_z_up_x_forward_y_left" || strings.TrimSpace(pkg.Coordinates.OriginDescription) == "" {
		return errors.New("coordinate contract is invalid")
	}
	if pkg.DirectionConvention.Units != "degrees" || pkg.DirectionConvention.ZeroDirection != "+x" || pkg.DirectionConvention.PositiveDirection != "counter_clockwise_toward_+y" || pkg.DirectionConvention.Normalization != "[0,360)" || pkg.DirectionConvention.Meaning != "direction_from_player_volume_toward_possible_incoming_line" {
		return errors.New("direction contract is invalid")
	}
	if !validBounds(pkg.Bounds) || !finitePositive(pkg.Resolution.AzimuthStepDegrees) || !finitePositive(pkg.Resolution.MaximumRayDistanceUnits) || !finitePositive(pkg.Resolution.ProtectionDistanceUnits) || pkg.Resolution.ProtectionDistanceUnits > pkg.Resolution.MaximumRayDistanceUnits {
		return errors.New("bounds or resolution is invalid")
	}
	directionSlots := int(math.Round(360 / pkg.Resolution.AzimuthStepDegrees))
	if directionSlots < 1 || math.Abs(float64(directionSlots)*pkg.Resolution.AzimuthStepDegrees-360) > 1e-9 {
		return errors.New("direction step must divide 360")
	}
	postures, err := validatePlayerVolume(pkg.PlayerVolume)
	if err != nil {
		return err
	}
	roles := map[string]string{}
	if len(pkg.GeometryProvenance) < 2 {
		return errors.New("geometry provenance is incomplete")
	}
	for _, source := range pkg.GeometryProvenance {
		if strings.TrimSpace(source.Role) == "" || !safeRelative(source.Reference) || !sha256Pattern.MatchString(source.SHA256) {
			return errors.New("geometry provenance entry is invalid")
		}
		if _, duplicate := roles[source.Role]; duplicate {
			return errors.New("geometry provenance role is duplicated")
		}
		roles[source.Role] = source.Reference
	}
	if _, ok := roles["physics_gltf"]; !ok {
		return errors.New("physics GLTF provenance is missing")
	}
	if _, ok := roles["physics_buffer"]; !ok {
		return errors.New("physics buffer provenance is missing")
	}
	if len(pkg.GeometrySamples) == 0 || pkg.Coverage.Declaration != "partial_reference_samples" || pkg.Coverage.FullMapCoverage || pkg.Coverage.SampleCount != len(pkg.GeometrySamples) {
		return errors.New("declared geometry coverage is invalid")
	}
	if !sameStrings(pkg.Coverage.PostureIDs, postureIDs(pkg.PlayerVolume.Postures)) {
		return errors.New("declared posture coverage differs from volume contract")
	}
	seenSamples := map[string]struct{}{}
	seenAliases := map[string]struct{}{}
	directionRefs := map[string]map[string]struct{}{}
	directionCount := 0
	for sampleIndex := range pkg.GeometrySamples {
		sample := &pkg.GeometrySamples[sampleIndex]
		if !sampleIDPattern.MatchString(sample.ID) || strings.TrimSpace(sample.Alias) == "" || !finiteVector(sample.Origin) || !pointInBounds(sample.Origin, pkg.Bounds) || !finitePositive(sample.SelectionRadiusUnits) || !finitePositive(sample.VerticalToleranceUnits) || roles[sample.OriginProvenance.SourceRole] != sample.OriginProvenance.SourceReference || strings.TrimSpace(sample.OriginProvenance.SourceCoordinate) == "" || strings.TrimSpace(sample.OriginProvenance.Interpretation) == "" {
			return fmt.Errorf("geometry sample %d is invalid", sampleIndex)
		}
		if _, duplicate := seenSamples[sample.ID]; duplicate {
			return errors.New("geometry sample id is duplicated")
		}
		if _, duplicate := seenAliases[sample.Alias]; duplicate {
			return errors.New("geometry sample alias is duplicated")
		}
		seenSamples[sample.ID] = struct{}{}
		seenAliases[sample.Alias] = struct{}{}
		if len(sample.Directions) != directionSlots {
			return errors.New("sample direction grid is incomplete")
		}
		for directionIndex := range sample.Directions {
			direction := &sample.Directions[directionIndex]
			wantID := fmt.Sprintf("%s/direction-%03d", sample.ID, directionIndex)
			wantAzimuth := float64(directionIndex) * pkg.Resolution.AzimuthStepDegrees
			if direction.ID != wantID || !directionIDPattern.MatchString(direction.ID) || !finite(direction.AzimuthDegrees) || math.Abs(direction.AzimuthDegrees-wantAzimuth) > 1e-9 {
				return errors.New("sample direction id or azimuth grid is invalid")
			}
			if len(direction.Profiles) != len(pkg.PlayerVolume.Postures) {
				return errors.New("direction posture profiles are incomplete")
			}
			directionRefs[direction.ID] = map[string]struct{}{}
			for profileIndex := range direction.Profiles {
				if err := validateProfile(direction.Profiles[profileIndex], pkg.PlayerVolume, postures, pkg.Resolution); err != nil {
					return fmt.Errorf("%s: %w", direction.ID, err)
				}
				if direction.Profiles[profileIndex].PostureID != pkg.PlayerVolume.Postures[profileIndex].ID {
					return errors.New("direction posture profiles are not deterministically ordered")
				}
				directionRefs[direction.ID][direction.Profiles[profileIndex].PostureID] = struct{}{}
			}
			directionCount++
		}
	}
	if pkg.Coverage.DirectionCount != directionCount {
		return errors.New("declared direction count differs from real package")
	}
	for left := 0; left < len(pkg.GeometrySamples); left++ {
		for right := left + 1; right < len(pkg.GeometrySamples); right++ {
			if sampleVolumesOverlap(pkg.GeometrySamples[left], pkg.GeometrySamples[right]) {
				return errors.New("sample selection volumes overlap ambiguously")
			}
		}
	}
	if err := validateReview(pkg.GeometryValidation, true); err != nil {
		return err
	}
	if err := validateReview(pkg.TacticalReview, false); err != nil {
		return err
	}
	seenAnnotations := map[string]struct{}{}
	annotated := map[string]struct{}{}
	for _, annotation := range pkg.TacticalAnnotations {
		if !annotationPattern.MatchString(annotation.ID) || !versionPattern.MatchString(annotation.AnnotationVersion) {
			return errors.New("tactical annotation identity is invalid")
		}
		if _, duplicate := seenAnnotations[annotation.ID]; duplicate {
			return errors.New("tactical annotation id is duplicated")
		}
		seenAnnotations[annotation.ID] = struct{}{}
		postureSet, ok := directionRefs[annotation.GeometryDirectionRef]
		if !ok {
			return errors.New("tactical annotation geometry reference is unknown")
		}
		if _, ok := postureSet[annotation.PostureID]; !ok {
			return errors.New("tactical annotation posture reference is unknown")
		}
		key := annotation.GeometryDirectionRef + "#" + annotation.PostureID
		if _, duplicate := annotated[key]; duplicate {
			return errors.New("geometry profile is annotated more than once")
		}
		annotated[key] = struct{}{}
		if len(annotation.Provenance) == 0 {
			return errors.New("tactical annotation provenance is missing")
		}
		if annotation.ReviewStatus == ReviewPending {
			if annotation.Classification != nil || annotation.AuthorID != nil || annotation.ReviewerID != nil || len(annotation.Evidence) != 0 {
				return errors.New("pending tactical annotation contains fabricated review")
			}
		} else if annotation.ReviewStatus == ReviewApproved {
			if annotation.Classification == nil || strings.TrimSpace(*annotation.Classification) == "" || annotation.AuthorID == nil || annotation.ReviewerID == nil || len(annotation.Evidence) == 0 {
				return errors.New("approved tactical annotation lacks attributable evidence")
			}
		} else {
			return errors.New("tactical annotation review status is invalid")
		}
	}
	if len(annotated) != directionCount*len(pkg.PlayerVolume.Postures) {
		return errors.New("tactical annotations do not cover every geometry profile")
	}
	if pkg.TacticalReview.Status == ReviewApproved {
		for _, annotation := range pkg.TacticalAnnotations {
			if annotation.ReviewStatus != ReviewApproved {
				return errors.New("approved tactical review contains pending annotations")
			}
		}
	}
	if len(pkg.Limitations) == 0 {
		return errors.New("package limitations are missing")
	}
	return nil
}

func ValidateSourceAssets(pkg *Package, mapsRoot string) error {
	if pkg == nil {
		return errors.New("package is nil")
	}
	for _, source := range pkg.GeometryProvenance {
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
			return fmt.Errorf("source hash mismatch for %s", source.Role)
		}
	}
	return nil
}

func validatePlayerVolume(volume PlayerVolume) (map[string]Posture, error) {
	if volume.ModelID != "cylinder_cross_section_probes@1" || !finitePositive(volume.RadiusUnits) || len(volume.LateralOffsetsUnits) < 3 || len(volume.Postures) == 0 {
		return nil, errors.New("player volume contract is invalid")
	}
	for index, value := range volume.LateralOffsetsUnits {
		if !finite(value) || math.Abs(value) > volume.RadiusUnits {
			return nil, errors.New("lateral probe offset exceeds player radius")
		}
		if index > 0 && value <= volume.LateralOffsetsUnits[index-1] {
			return nil, errors.New("lateral offsets are not strictly ordered")
		}
	}
	if !containsFloat(volume.LateralOffsetsUnits, 0) {
		return nil, errors.New("player volume requires a centerline probe")
	}
	result := map[string]Posture{}
	for index, posture := range volume.Postures {
		if !postureIDPattern.MatchString(posture.ID) || !finitePositive(posture.VolumeHeightUnits) || len(posture.ProbeHeightsUnits) < 2 {
			return nil, errors.New("posture contract is invalid")
		}
		if _, duplicate := result[posture.ID]; duplicate {
			return nil, errors.New("posture id is duplicated")
		}
		for heightIndex, height := range posture.ProbeHeightsUnits {
			if !finitePositive(height) || height > posture.VolumeHeightUnits {
				return nil, errors.New("posture probe height exceeds volume")
			}
			if heightIndex > 0 && height <= posture.ProbeHeightsUnits[heightIndex-1] {
				return nil, errors.New("posture probe heights are not strictly ordered")
			}
		}
		if index > 0 && posture.ID <= volume.Postures[index-1].ID {
			return nil, errors.New("postures are not deterministically ordered")
		}
		result[posture.ID] = posture
	}
	return result, nil
}

func validateProfile(profile OcclusionProfile, volume PlayerVolume, postures map[string]Posture, resolution Resolution) error {
	posture, ok := postures[profile.PostureID]
	if !ok {
		return errors.New("unknown posture profile")
	}
	expected := len(posture.ProbeHeightsUnits) * len(volume.LateralOffsetsUnits)
	if profile.ProbeCount != expected || len(profile.Probes) != expected {
		return errors.New("profile probe count is invalid")
	}
	occluded := 0
	probeIndex := 0
	for _, height := range posture.ProbeHeightsUnits {
		for _, offset := range volume.LateralOffsetsUnits {
			probe := profile.Probes[probeIndex]
			probeIndex++
			if math.Abs(probe.HeightUnits-height) > 1e-9 || math.Abs(probe.LateralOffsetUnits-offset) > 1e-9 {
				return errors.New("profile probe grid is invalid")
			}
			if probe.HitDistanceUnits != nil && (!finite(*probe.HitDistanceUnits) || *probe.HitDistanceUnits < 0 || *probe.HitDistanceUnits > resolution.MaximumRayDistanceUnits+1e-6) {
				return errors.New("probe hit distance is invalid")
			}
			want := probe.HitDistanceUnits != nil && *probe.HitDistanceUnits <= resolution.ProtectionDistanceUnits+1e-9
			if probe.OccludedWithinProtectionDistance != want {
				return errors.New("probe occlusion does not match distance")
			}
			if want {
				occluded++
			}
		}
	}
	if profile.OccludedProbeCount != occluded || profile.CompleteVolumeOcclusion != (occluded == expected) {
		return errors.New("profile aggregate does not recompute")
	}
	return nil
}

func validateReview(review ReviewState, approvedRequired bool) error {
	if review.Status == ReviewPending {
		if approvedRequired {
			return errors.New("required geometry validation is not approved")
		}
		if review.AuthorID != nil || review.ReviewerID != nil || len(review.Evidence) != 0 {
			return errors.New("pending review contains fabricated evidence")
		}
		return nil
	}
	if review.Status != ReviewApproved || review.AuthorID == nil || review.ReviewerID == nil || len(review.Evidence) == 0 {
		return errors.New("approved review lacks attributable evidence")
	}
	return nil
}
func validateSupportEntry(entry SupportEntry) error {
	if !mapIDPattern.MatchString(entry.MapID) || entry.ContractVersion != ContractVersion || !versionPattern.MatchString(entry.AssetVersion) || entry.PackagePath != fmt.Sprintf("packages/%s/%s/package.json", entry.MapID, entry.AssetVersion) || !sha256Pattern.MatchString(entry.PackageSHA256) || entry.ImplementationStatus != "candidate" || entry.GeometryStatus != "validated" || entry.TrainingUseAuthorized {
		return errors.New("support entry is invalid")
	}
	if entry.TacticalReviewStatus != ReviewPending && entry.TacticalReviewStatus != ReviewApproved {
		return errors.New("support review status is invalid")
	}
	if entry.ProductSupported && entry.TacticalReviewStatus != ReviewApproved {
		return errors.New("unsupported tactical review cannot be product supported")
	}
	return nil
}
func unavailable(reason string) PackageResolution {
	return PackageResolution{Status: GeometryUnavailable, ReasonCodes: []string{reason}}
}
func decodeStrict(payload []byte, destination any) error {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
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
func safeRelative(reference string) bool {
	return strings.TrimSpace(reference) != "" && !filepath.IsAbs(filepath.FromSlash(reference)) && !strings.Contains(reference, "\\") && !containsParent(filepath.ToSlash(reference))
}
func containsParent(reference string) bool {
	for _, part := range strings.Split(reference, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
func safeDescendant(root, reference string) (string, error) {
	if !safeRelative(reference) {
		return "", errors.New("unsafe relative path")
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
		return "", errors.New("path escapes root")
	}
	return candidate, nil
}
func validBounds(bounds Bounds) bool {
	return finiteVector(bounds.Min) && finiteVector(bounds.Max) && bounds.Min.X < bounds.Max.X && bounds.Min.Y < bounds.Max.Y && bounds.Min.Z < bounds.Max.Z
}
func pointInBounds(point Vector3, bounds Bounds) bool {
	return point.X >= bounds.Min.X && point.X <= bounds.Max.X && point.Y >= bounds.Min.Y && point.Y <= bounds.Max.Y && point.Z >= bounds.Min.Z && point.Z <= bounds.Max.Z
}
func sampleVolumesOverlap(left, right GeometrySample) bool {
	return math.Hypot(left.Origin.X-right.Origin.X, left.Origin.Y-right.Origin.Y) < left.SelectionRadiusUnits+right.SelectionRadiusUnits && math.Abs(left.Origin.Z-right.Origin.Z) <= left.VerticalToleranceUnits+right.VerticalToleranceUnits
}
func postureIDs(postures []Posture) []string {
	result := make([]string, len(postures))
	for index := range postures {
		result[index] = postures[index].ID
	}
	return result
}
func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
func containsFloat(values []float64, want float64) bool {
	for _, value := range values {
		if math.Abs(value-want) < 1e-9 {
			return true
		}
	}
	return false
}
func finite(value float64) bool         { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func finitePositive(value float64) bool { return finite(value) && value > 0 }
func finiteVector(value Vector3) bool   { return finite(value.X) && finite(value.Y) && finite(value.Z) }

func SortedUnique(values []string) []string {
	seen := map[string]struct{}{}
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
