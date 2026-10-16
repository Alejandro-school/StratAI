package exposure

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
	ContractVersion          = "1.0.0"
	PackageSchemaID          = "stratai.exposure_angle_map_package@1"
	SupportManifestSchemaID  = "stratai.exposure_angle_support_manifest@1"
	SupportManifestFilename  = "support_manifest.json"
	ReviewPending            = "pending_review"
	ReviewApproved           = "approved"
	GeometryAvailable        = "available"
	GeometryUnavailable      = "unavailable"
	ReasonMapNotRegistered   = "map_not_registered"
	ReasonPackageMissing     = "package_file_missing"
	ReasonPackageHash        = "package_hash_mismatch"
	ReasonContractVersion    = "contract_version_incompatible"
	ReasonAssetVersion       = "asset_version_incompatible"
	ReasonMapMismatch        = "map_id_mismatch"
	ReasonPackageInvalid     = "package_invalid"
	ReasonGeometrySource     = "geometry_source_mismatch"
	ReasonOutsideCoverage    = "outside_declared_sample_coverage"
	ReasonAmbiguousSample    = "ambiguous_sample_overlap"
	ReasonPosition           = "position_unavailable"
	ReasonViewAngles         = "view_angles_unavailable"
	ReasonHumanReviewPending = "human_review_pending"
)

var (
	mapIDPattern      = regexp.MustCompile(`^de_[a-z0-9_]+$`)
	versionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-candidate\.[0-9]+)?$`)
	sha256Pattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	sampleIDPattern   = regexp.MustCompile(`^sample-[0-9]{3}$`)
	sectorIDPattern   = regexp.MustCompile(`^sample-[0-9]{3}/sector-[0-9]{3}$`)
	annotationPattern = regexp.MustCompile(`^annotation-[0-9]{3}$`)
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

type AngularConvention struct {
	Units             string `json:"units"`
	ZeroDirection     string `json:"zero_direction"`
	PositiveDirection string `json:"positive_direction"`
	Normalization     string `json:"normalization"`
	Interval          string `json:"interval"`
	SectorEncoding    string `json:"sector_encoding"`
}

type Resolution struct {
	AzimuthStepDegrees       float64 `json:"azimuth_step_degrees"`
	MaximumRayDistance       float64 `json:"maximum_ray_distance"`
	MinimumClearanceDistance float64 `json:"minimum_clearance_distance"`
	SamplingHeightUnits      float64 `json:"sampling_height_units"`
}

type SourceAsset struct {
	Role      string `json:"role"`
	Reference string `json:"reference"`
	SHA256    string `json:"sha256"`
}

type RaySample struct {
	AzimuthDegrees float64  `json:"azimuth_degrees"`
	HitDistance    *float64 `json:"hit_distance_units"`
	ClearanceMet   bool     `json:"clearance_met"`
}

type Sector struct {
	ID           string  `json:"id"`
	StartDegrees float64 `json:"start_degrees"`
	WidthDegrees float64 `json:"width_degrees"`
}

type OriginProvenance struct {
	SourceRole       string `json:"source_role"`
	SourceReference  string `json:"source_reference"`
	SourceCoordinate string `json:"source_coordinate"`
	Interpretation   string `json:"interpretation"`
}

type GeometrySample struct {
	ID                     string           `json:"id"`
	Alias                  string           `json:"alias"`
	Origin                 Vector3          `json:"origin_world"`
	SelectionRadiusUnits   float64          `json:"selection_radius_units"`
	VerticalToleranceUnits float64          `json:"vertical_tolerance_units"`
	OriginProvenance       OriginProvenance `json:"origin_provenance"`
	Rays                   []RaySample      `json:"rays"`
	Sectors                []Sector         `json:"geometry_sectors"`
}

type ReviewState struct {
	Status     string   `json:"status"`
	AuthorID   *string  `json:"author_id"`
	ReviewerID *string  `json:"reviewer_id"`
	Evidence   []string `json:"evidence"`
}

type TacticalAnnotation struct {
	ID             string   `json:"id"`
	GeometrySector string   `json:"geometry_sector_ref"`
	Classification *string  `json:"classification"`
	Provenance     []string `json:"provenance"`
	Evidence       []string `json:"evidence"`
	AuthorID       *string  `json:"author_id"`
	ReviewerID     *string  `json:"reviewer_id"`
	ReviewStatus   string   `json:"review_status"`
}

type Coverage struct {
	Declaration     string `json:"declaration"`
	FullMapCoverage bool   `json:"full_map_coverage"`
	SampleCount     int    `json:"sample_count"`
	SectorCount     int    `json:"sector_count"`
}

type Package struct {
	SchemaID            string               `json:"schema_id"`
	ContractVersion     string               `json:"contract_version"`
	AssetVersion        string               `json:"asset_version"`
	MapID               string               `json:"map_id"`
	GeometryRevision    string               `json:"geometry_revision"`
	Coordinates         Coordinates          `json:"coordinates"`
	Angles              AngularConvention    `json:"angular_convention"`
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

func SectorContains(sector Sector, angleDegrees float64) bool {
	if sector.WidthDegrees >= 360 {
		return true
	}
	delta := NormalizeDegrees(angleDegrees - sector.StartDegrees)
	return delta >= 0 && delta < sector.WidthDegrees
}

func RelativeSectors(sectors []Sector, viewYawDegrees float64) []Sector {
	result := make([]Sector, len(sectors))
	for index, sector := range sectors {
		result[index] = sector
		result[index].StartDegrees = NormalizeDegrees(sector.StartDegrees - viewYawDegrees)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].StartDegrees != result[j].StartDegrees {
			return result[i].StartDegrees < result[j].StartDegrees
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func BuildSectors(sampleID string, rays []RaySample, stepDegrees float64) ([]Sector, error) {
	if !sampleIDPattern.MatchString(sampleID) || !finitePositive(stepDegrees) {
		return nil, errors.New("invalid sample id or angular step")
	}
	expected := int(math.Round(360 / stepDegrees))
	if expected < 1 || math.Abs(float64(expected)*stepDegrees-360) > 1e-9 || len(rays) != expected {
		return nil, errors.New("ray grid does not divide the full circle")
	}
	clear := make([]bool, len(rays))
	clearCount := 0
	for index, ray := range rays {
		wantAngle := float64(index) * stepDegrees
		if !finite(ray.AzimuthDegrees) || math.Abs(ray.AzimuthDegrees-wantAngle) > 1e-9 {
			return nil, errors.New("ray azimuths are not a complete ordered grid")
		}
		clear[index] = ray.ClearanceMet
		if clear[index] {
			clearCount++
		}
	}
	if clearCount == 0 {
		return []Sector{}, nil
	}
	if clearCount == len(clear) {
		return []Sector{{ID: sampleID + "/sector-001", StartDegrees: 0, WidthDegrees: 360}}, nil
	}

	type rawSector struct {
		start float64
		width float64
	}
	raw := make([]rawSector, 0)
	for index := range clear {
		previous := (index - 1 + len(clear)) % len(clear)
		if !clear[index] || clear[previous] {
			continue
		}
		count := 0
		for count < len(clear) && clear[(index+count)%len(clear)] {
			count++
		}
		raw = append(raw, rawSector{
			start: NormalizeDegrees(float64(index)*stepDegrees - stepDegrees/2),
			width: float64(count) * stepDegrees,
		})
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i].start < raw[j].start })
	sectors := make([]Sector, len(raw))
	for index, sector := range raw {
		sectors[index] = Sector{
			ID:           fmt.Sprintf("%s/sector-%03d", sampleID, index+1),
			StartDegrees: sector.start,
			WidthDegrees: sector.width,
		}
	}
	return sectors, nil
}

func SelectSample(pkg *Package, point Vector3) (*GeometrySample, string) {
	if pkg == nil || !finiteVector(point) {
		return nil, ReasonPosition
	}
	candidates := make([]int, 0, 1)
	for index, sample := range pkg.GeometrySamples {
		dx := point.X - sample.Origin.X
		dy := point.Y - sample.Origin.Y
		if math.Hypot(dx, dy) <= sample.SelectionRadiusUnits &&
			math.Abs(point.Z-sample.Origin.Z) <= sample.VerticalToleranceUnits {
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

func ResolvePackage(
	registryRoot, mapsRoot, mapID, requiredContractVersion, requiredAssetVersion string,
) PackageResolution {
	if requiredContractVersion != ContractVersion {
		return unavailable(ReasonContractVersion)
	}
	manifestPath := filepath.Join(registryRoot, SupportManifestFilename)
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		return unavailable(ReasonPackageMissing)
	}
	var manifest SupportManifest
	if err := decodeStrict(payload, &manifest); err != nil || manifest.SchemaID != SupportManifestSchemaID ||
		manifest.ContractVersion != ContractVersion {
		return unavailable(ReasonPackageInvalid)
	}
	seenMaps := make(map[string]struct{}, len(manifest.Packages))
	var support *SupportEntry
	for index := range manifest.Packages {
		entry := &manifest.Packages[index]
		if err := validateSupportEntry(*entry); err != nil {
			return unavailable(ReasonPackageInvalid)
		}
		if _, duplicate := seenMaps[entry.MapID]; duplicate {
			return unavailable(ReasonPackageInvalid)
		}
		seenMaps[entry.MapID] = struct{}{}
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
	packageHash := sha256.Sum256(packagePayload)
	packageSHA256 := hex.EncodeToString(packageHash[:])
	if !sha256Pattern.MatchString(support.PackageSHA256) || packageSHA256 != support.PackageSHA256 {
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
	if pkg.TacticalReview.Status != support.TacticalReviewStatus ||
		(support.ProductSupported && pkg.TacticalReview.Status != ReviewApproved) {
		return unavailable(ReasonPackageInvalid)
	}
	if err := ValidateSourceAssets(&pkg, mapsRoot); err != nil {
		return unavailable(ReasonGeometrySource)
	}
	copySupport := *support
	return PackageResolution{
		Status:        GeometryAvailable,
		Package:       &pkg,
		PackageSHA256: packageSHA256,
		Support:       &copySupport,
	}
}

func ValidatePackage(pkg *Package) error {
	if pkg == nil {
		return errors.New("package is nil")
	}
	if pkg.SchemaID != PackageSchemaID || pkg.ContractVersion != ContractVersion ||
		!versionPattern.MatchString(pkg.AssetVersion) || !mapIDPattern.MatchString(pkg.MapID) {
		return errors.New("package identity is invalid")
	}
	if !sha256Pattern.MatchString(pkg.GeometryRevision) {
		return errors.New("geometry revision is invalid")
	}
	if pkg.Coordinates.System != "source2_world" || pkg.Coordinates.Units != "source2_world_units" ||
		pkg.Coordinates.AxisConvention != "right_handed_z_up_x_forward_y_left" ||
		strings.TrimSpace(pkg.Coordinates.OriginDescription) == "" {
		return errors.New("coordinate contract is invalid")
	}
	if pkg.Angles.Units != "degrees" || pkg.Angles.ZeroDirection != "+x" ||
		pkg.Angles.PositiveDirection != "counter_clockwise_toward_+y" ||
		pkg.Angles.Normalization != "[0,360)" || pkg.Angles.Interval != "start_inclusive_end_exclusive" ||
		pkg.Angles.SectorEncoding != "normalized_start_plus_width" {
		return errors.New("angular contract is invalid")
	}
	if !validBounds(pkg.Bounds) || !finitePositive(pkg.Resolution.AzimuthStepDegrees) ||
		!finitePositive(pkg.Resolution.MaximumRayDistance) ||
		!finitePositive(pkg.Resolution.MinimumClearanceDistance) ||
		pkg.Resolution.MinimumClearanceDistance > pkg.Resolution.MaximumRayDistance ||
		!finitePositive(pkg.Resolution.SamplingHeightUnits) {
		return errors.New("bounds or resolution is invalid")
	}
	if len(pkg.GeometryProvenance) < 2 {
		return errors.New("geometry provenance is incomplete")
	}
	roles := make(map[string]string, len(pkg.GeometryProvenance))
	for _, source := range pkg.GeometryProvenance {
		if strings.TrimSpace(source.Role) == "" || !safeRelative(source.Reference) ||
			!sha256Pattern.MatchString(source.SHA256) {
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
	if len(pkg.GeometrySamples) == 0 || pkg.Coverage.Declaration != "partial_reference_samples" ||
		pkg.Coverage.FullMapCoverage || pkg.Coverage.SampleCount != len(pkg.GeometrySamples) {
		return errors.New("declared geometry coverage is invalid")
	}

	seenSamples := make(map[string]struct{}, len(pkg.GeometrySamples))
	seenAliases := make(map[string]struct{}, len(pkg.GeometrySamples))
	sectorRefs := make(map[string]struct{})
	sectorCount := 0
	for sampleIndex := range pkg.GeometrySamples {
		sample := &pkg.GeometrySamples[sampleIndex]
		if !sampleIDPattern.MatchString(sample.ID) || strings.TrimSpace(sample.Alias) == "" ||
			!finiteVector(sample.Origin) || !pointInBounds(sample.Origin, pkg.Bounds) ||
			!finitePositive(sample.SelectionRadiusUnits) || !finitePositive(sample.VerticalToleranceUnits) ||
			strings.TrimSpace(sample.OriginProvenance.SourceRole) == "" ||
			!safeRelative(sample.OriginProvenance.SourceReference) ||
			strings.TrimSpace(sample.OriginProvenance.SourceCoordinate) == "" ||
			strings.TrimSpace(sample.OriginProvenance.Interpretation) == "" {
			return fmt.Errorf("geometry sample %d is invalid", sampleIndex)
		}
		if _, duplicate := seenSamples[sample.ID]; duplicate {
			return errors.New("geometry sample id is duplicated")
		}
		if _, duplicate := seenAliases[sample.Alias]; duplicate {
			return errors.New("geometry sample alias is duplicated")
		}
		if reference, ok := roles[sample.OriginProvenance.SourceRole]; !ok ||
			reference != sample.OriginProvenance.SourceReference {
			return errors.New("geometry sample origin provenance is not declared")
		}
		seenSamples[sample.ID] = struct{}{}
		seenAliases[sample.Alias] = struct{}{}
		for _, ray := range sample.Rays {
			if ray.HitDistance != nil && (!finite(*ray.HitDistance) || *ray.HitDistance < 0 ||
				*ray.HitDistance > pkg.Resolution.MaximumRayDistance+1e-6) {
				return errors.New("ray hit distance is invalid")
			}
			wantClearance := ray.HitDistance == nil || *ray.HitDistance >= pkg.Resolution.MinimumClearanceDistance
			if ray.ClearanceMet != wantClearance {
				return errors.New("ray clearance classification does not match distance")
			}
		}
		expected, err := BuildSectors(sample.ID, sample.Rays, pkg.Resolution.AzimuthStepDegrees)
		if err != nil || !sameSectors(expected, sample.Sectors) {
			return errors.New("stored geometry sectors do not recompute")
		}
		for _, sector := range sample.Sectors {
			if !sectorIDPattern.MatchString(sector.ID) || !finite(sector.StartDegrees) ||
				sector.StartDegrees < 0 || sector.StartDegrees >= 360 ||
				!finitePositive(sector.WidthDegrees) || sector.WidthDegrees > 360 {
				return errors.New("geometry sector is invalid")
			}
			if _, duplicate := sectorRefs[sector.ID]; duplicate {
				return errors.New("geometry sector id is duplicated")
			}
			sectorRefs[sector.ID] = struct{}{}
			sectorCount++
		}
	}
	if pkg.Coverage.SectorCount != sectorCount {
		return errors.New("declared sector count does not match package")
	}
	for left := 0; left < len(pkg.GeometrySamples); left++ {
		for right := left + 1; right < len(pkg.GeometrySamples); right++ {
			if sampleVolumesOverlap(pkg.GeometrySamples[left], pkg.GeometrySamples[right]) {
				return errors.New("sample selection volumes overlap ambiguously")
			}
		}
	}
	if err := validateReview(pkg.GeometryValidation, true); err != nil || pkg.GeometryValidation.Status != ReviewApproved {
		return errors.New("geometry validation is not approved")
	}
	if err := validateReview(pkg.TacticalReview, false); err != nil {
		return err
	}
	seenAnnotations := make(map[string]struct{}, len(pkg.TacticalAnnotations))
	annotatedSectors := make(map[string]struct{}, len(pkg.TacticalAnnotations))
	for _, annotation := range pkg.TacticalAnnotations {
		if !annotationPattern.MatchString(annotation.ID) {
			return errors.New("tactical annotation id is invalid")
		}
		if _, duplicate := seenAnnotations[annotation.ID]; duplicate {
			return errors.New("tactical annotation id is duplicated")
		}
		seenAnnotations[annotation.ID] = struct{}{}
		if _, ok := sectorRefs[annotation.GeometrySector]; !ok || len(annotation.Provenance) == 0 {
			return errors.New("tactical annotation reference or provenance is invalid")
		}
		if _, duplicate := annotatedSectors[annotation.GeometrySector]; duplicate {
			return errors.New("tactical geometry sector is annotated more than once")
		}
		annotatedSectors[annotation.GeometrySector] = struct{}{}
		if annotation.ReviewStatus == ReviewApproved {
			if annotation.Classification == nil || strings.TrimSpace(*annotation.Classification) == "" ||
				annotation.AuthorID == nil || annotation.ReviewerID == nil || len(annotation.Evidence) == 0 {
				return errors.New("approved tactical annotation lacks human review evidence")
			}
		} else if annotation.ReviewStatus == ReviewPending {
			if annotation.Classification != nil || annotation.AuthorID != nil || annotation.ReviewerID != nil {
				return errors.New("pending tactical annotation must remain unclassified and unsigned")
			}
		} else {
			return errors.New("tactical annotation review status is invalid")
		}
	}
	if len(annotatedSectors) != len(sectorRefs) {
		return errors.New("tactical annotations do not cover the declared geometry sectors")
	}
	if pkg.TacticalReview.Status == ReviewApproved {
		for _, annotation := range pkg.TacticalAnnotations {
			if annotation.ReviewStatus != ReviewApproved {
				return errors.New("approved tactical review contains pending annotations")
			}
		}
	}
	if pkg.TacticalReview.Status == ReviewPending &&
		(pkg.TacticalReview.AuthorID != nil || pkg.TacticalReview.ReviewerID != nil || len(pkg.TacticalReview.Evidence) != 0) {
		return errors.New("pending tactical review must not fabricate human evidence")
	}
	if len(pkg.Limitations) == 0 {
		return errors.New("package limitations are required")
	}
	return nil
}

func validateSupportEntry(entry SupportEntry) error {
	expectedPath := filepath.ToSlash(filepath.Join("packages", entry.MapID, entry.AssetVersion, "package.json"))
	if !mapIDPattern.MatchString(entry.MapID) || entry.ContractVersion != ContractVersion ||
		!versionPattern.MatchString(entry.AssetVersion) || entry.PackagePath != expectedPath ||
		!sha256Pattern.MatchString(entry.PackageSHA256) || entry.ImplementationStatus != "candidate" ||
		entry.GeometryStatus != "validated" ||
		(entry.TacticalReviewStatus != ReviewPending && entry.TacticalReviewStatus != ReviewApproved) ||
		entry.TrainingUseAuthorized || (entry.ProductSupported && entry.TacticalReviewStatus != ReviewApproved) {
		return errors.New("support manifest entry is invalid")
	}
	return nil
}

func ValidateSourceAssets(pkg *Package, mapsRoot string) error {
	if pkg == nil || strings.TrimSpace(mapsRoot) == "" {
		return errors.New("maps root is required")
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
			return fmt.Errorf("source asset %s hash mismatch", source.Role)
		}
	}
	return nil
}

func unavailable(reason string) PackageResolution {
	return PackageResolution{Status: GeometryUnavailable, ReasonCodes: []string{reason}}
}

func decodeStrict(payload []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON")
		}
		return fmt.Errorf("unexpected trailing JSON: %w", err)
	}
	return nil
}

func safeDescendant(root, reference string) (string, error) {
	if !safeRelative(reference) {
		return "", errors.New("asset reference is not a safe relative path")
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
		return "", errors.New("asset reference escapes its root")
	}
	return candidate, nil
}

func safeRelative(reference string) bool {
	if strings.TrimSpace(reference) == "" || filepath.IsAbs(filepath.FromSlash(reference)) || strings.Contains(reference, "\\") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(reference)))
	return clean == reference && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func finite(value float64) bool         { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func finitePositive(value float64) bool { return finite(value) && value > 0 }
func finiteVector(value Vector3) bool   { return finite(value.X) && finite(value.Y) && finite(value.Z) }

func validBounds(bounds Bounds) bool {
	return finiteVector(bounds.Min) && finiteVector(bounds.Max) &&
		bounds.Min.X < bounds.Max.X && bounds.Min.Y < bounds.Max.Y && bounds.Min.Z < bounds.Max.Z
}

func pointInBounds(point Vector3, bounds Bounds) bool {
	return point.X >= bounds.Min.X && point.X <= bounds.Max.X &&
		point.Y >= bounds.Min.Y && point.Y <= bounds.Max.Y &&
		point.Z >= bounds.Min.Z && point.Z <= bounds.Max.Z
}

func sameSectors(left, right []Sector) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].ID != right[index].ID ||
			math.Abs(left[index].StartDegrees-right[index].StartDegrees) > 1e-9 ||
			math.Abs(left[index].WidthDegrees-right[index].WidthDegrees) > 1e-9 {
			return false
		}
	}
	return true
}

func sampleVolumesOverlap(left, right GeometrySample) bool {
	vertical := math.Abs(left.Origin.Z-right.Origin.Z) <=
		left.VerticalToleranceUnits+right.VerticalToleranceUnits
	horizontal := math.Hypot(left.Origin.X-right.Origin.X, left.Origin.Y-right.Origin.Y) <
		left.SelectionRadiusUnits+right.SelectionRadiusUnits
	return vertical && horizontal
}

func validateReview(review ReviewState, requireApproved bool) error {
	if review.Status != ReviewPending && review.Status != ReviewApproved {
		return errors.New("review status is invalid")
	}
	if requireApproved && review.Status != ReviewApproved {
		return errors.New("required review is not approved")
	}
	if review.Status == ReviewApproved {
		if review.AuthorID == nil || strings.TrimSpace(*review.AuthorID) == "" ||
			review.ReviewerID == nil || strings.TrimSpace(*review.ReviewerID) == "" || len(review.Evidence) == 0 {
			return errors.New("approved review lacks provenance")
		}
	} else if review.AuthorID != nil || review.ReviewerID != nil || len(review.Evidence) != 0 {
		return errors.New("pending review must be unsigned")
	}
	return nil
}
