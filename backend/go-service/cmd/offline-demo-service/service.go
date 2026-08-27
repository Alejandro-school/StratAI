package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"cs2-demo-service/models"
	"cs2-demo-service/parser"
)

const (
	parserVersion    = "v16"
	exportVersion    = "3.8.0"
	qualityVersion   = 12
	validatorVersion = "stratai.canonical_validator@2"
	releaseSchemaID  = "stratai.go_offline_release@1"
)

var (
	matchIDRegex  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)
	checksumRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type offlineConfig struct {
	address       string
	demosRoot     string
	exportRoot    string
	maxConcurrent int
}

type processDemoRequest struct {
	DemoPath       string `json:"demo_path"`
	MatchID        string `json:"match_id"`
	MatchDate      string `json:"match_date"`
	Checksum       string `json:"checksum"`
	SchemaVersion  string `json:"parser_schema_version"`
	Source         string `json:"source"`
	SourceEndpoint string `json:"source_endpoint"`
	SourceQueried  string `json:"source_queried_at"`
	SourceVersion  string `json:"source_version"`
}

type processingResult struct {
	Kills  int
	Rounds int
}

type demoProcessor func(context.Context, processDemoRequest, string, string) (processingResult, error)

type offlineService struct {
	config    offlineConfig
	buildID   string
	process   demoProcessor
	slots     chan struct{}
	pathLocks sync.Map
}

type exportArtifact struct {
	ArtifactType string `json:"artifact_type"`
	Path         string `json:"path"`
	Format       string `json:"format"`
	Compression  string `json:"compression"`
	SchemaID     string `json:"schema_id,omitempty"`
	RecordCount  *int   `json:"record_count,omitempty"`
	SHA256       string `json:"sha256"`
	Bytes        int64  `json:"bytes"`
}

type exportManifest struct {
	MatchID             string           `json:"match_id"`
	Checksum            string           `json:"checksum"`
	ParserSchemaVersion string           `json:"parser_schema_version"`
	CommittedAt         time.Time        `json:"committed_at"`
	ExportFormatVersion string           `json:"export_format_version"`
	ValidatorVersion    string           `json:"validator_version"`
	ValidationStatus    string           `json:"validation_status"`
	Artifacts           []exportArtifact `json:"artifacts"`
}

type canonicalArtifactMetadata struct {
	ArtifactType string `json:"artifact_type"`
	Path         string `json:"path"`
	Format       string `json:"format"`
	Compression  string `json:"compression"`
	SchemaID     string `json:"schema_id"`
	RecordCount  *int   `json:"record_count"`
}

type canonicalManifestMetadata struct {
	SchemaID  string                      `json:"schema_id"`
	Artifacts []canonicalArtifactMetadata `json:"artifacts"`
}

func newOfflineService(config offlineConfig, buildIdentifier string, processor demoProcessor) *offlineService {
	if processor == nil {
		processor = parseAndExportDemo
	}
	return &offlineService{
		config: config, buildID: buildIdentifier, process: processor,
		slots: make(chan struct{}, config.maxConcurrent),
	}
}

func releaseMetadata(buildIdentifier string) map[string]interface{} {
	return map[string]interface{}{
		"schema_id":              releaseSchemaID,
		"build_id":               buildIdentifier,
		"parser_schema_version":  parserVersion,
		"export_format_version":  exportVersion,
		"quality_schema_version": qualityVersion,
		"validator_version":      validatorVersion,
	}
}

func (service *offlineService) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", service.handleHealth)
	mux.HandleFunc("/process-demo", service.handleProcessDemo)
	return mux
}

func (service *offlineService) handleHealth(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload := releaseMetadata(service.buildID)
	payload["status"] = "ok"
	payload["service"] = "stratai-offline-demo-service"
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (service *offlineService) handleProcessDemo(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, request.Body, 64*1024))
	decoder.DisallowUnknownFields()
	var payload processDemoRequest
	if err := decoder.Decode(&payload); err != nil || ensureJSONEOF(decoder) != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if !matchIDRegex.MatchString(payload.MatchID) || !checksumRegex.MatchString(payload.Checksum) {
		http.Error(w, "valid match_id and checksum are required", http.StatusBadRequest)
		return
	}
	if payload.SchemaVersion == "" {
		payload.SchemaVersion = parserVersion
	}
	if payload.SchemaVersion != parserVersion {
		http.Error(w, "unsupported parser schema version", http.StatusBadRequest)
		return
	}
	if !pathWithin(service.config.demosRoot, payload.DemoPath) ||
		!strings.EqualFold(filepath.Ext(payload.DemoPath), ".dem") {
		http.Error(w, "invalid demo path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(payload.DemoPath)
	if err != nil || !info.Mode().IsRegular() {
		http.Error(w, "demo file not found", http.StatusNotFound)
		return
	}
	if info.Size() < 100*1024 {
		http.Error(w, "demo file is incomplete", http.StatusBadRequest)
		return
	}
	actualChecksum, err := hashFile(payload.DemoPath)
	if err != nil || actualChecksum != payload.Checksum {
		http.Error(w, "demo checksum mismatch", http.StatusBadRequest)
		return
	}

	select {
	case service.slots <- struct{}{}:
		defer func() { <-service.slots }()
	case <-request.Context().Done():
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
		return
	}
	lockValue, _ := service.pathLocks.LoadOrStore(payload.MatchID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	finalBundle := filepath.Join(service.config.exportRoot, "match_"+payload.MatchID)
	if _, err := os.Lstat(finalBundle); err == nil {
		http.Error(w, "staging bundle already exists", http.StatusConflict)
		return
	} else if !os.IsNotExist(err) {
		http.Error(w, "could not inspect export root", http.StatusInternalServerError)
		return
	}
	stagingRoot, err := os.MkdirTemp(service.config.exportRoot, ".offline-export-")
	if err != nil {
		http.Error(w, "could not prepare export root", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(stagingRoot)

	result, err := service.process(request.Context(), payload, stagingRoot, service.buildID)
	if err != nil {
		log.Printf("event=canonical_export_failed match_id=%s error=%q", payload.MatchID, err)
		http.Error(w, "could not export required artifacts", http.StatusUnprocessableEntity)
		return
	}
	if err := request.Context().Err(); err != nil {
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
		return
	}
	stagedBundle := filepath.Join(stagingRoot, "match_"+payload.MatchID)
	artifacts, err := buildArtifactCatalog(stagedBundle)
	if err != nil {
		log.Printf("event=artifact_catalog_failed match_id=%s error=%q", payload.MatchID, err)
		http.Error(w, "could not catalog export artifacts", http.StatusInternalServerError)
		return
	}
	manifest := exportManifest{
		MatchID: payload.MatchID, Checksum: payload.Checksum,
		ParserSchemaVersion: parserVersion, CommittedAt: time.Now().UTC(),
		ExportFormatVersion: exportVersion, ValidatorVersion: validatorVersion,
		ValidationStatus: "pending", Artifacts: artifacts,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil || os.WriteFile(filepath.Join(stagedBundle, "manifest.json"), manifestBytes, 0o640) != nil {
		http.Error(w, "could not create operational manifest", http.StatusInternalServerError)
		return
	}
	if err := os.Rename(stagedBundle, finalBundle); err != nil {
		http.Error(w, "could not publish staging bundle", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "exported", "match_id": payload.MatchID,
		"bundle_name": filepath.Base(finalBundle), "kills": result.Kills, "rounds": result.Rounds,
	})
}

func parseAndExportDemo(
	ctx context.Context, payload processDemoRequest, outputRoot, buildIdentifier string,
) (processingResult, error) {
	if err := ctx.Err(); err != nil {
		return processingResult{}, err
	}
	demoContext, err := parser.ParseDemo(payload.DemoPath)
	if err != nil {
		return processingResult{}, fmt.Errorf("parse demo: %w", err)
	}
	demoContext.MatchData.MatchID = payload.MatchID
	if strings.TrimSpace(payload.Source) == "" {
		payload.Source = "faceit"
	}
	provenance := models.CanonicalExportProvenance{
		DemoChecksum: payload.Checksum, BuildIdentifier: buildIdentifier,
		Source: payload.Source, Endpoint: payload.SourceEndpoint,
		QueriedAt: payload.SourceQueried, SourceVersion: payload.SourceVersion,
	}
	if err := parser.ExportMatchBundleWithProvenance(
		demoContext, payload.MatchID, outputRoot, payload.MatchDate, provenance,
	); err != nil {
		return processingResult{}, fmt.Errorf("export canonical bundle: %w", err)
	}
	return processingResult{
		Kills: len(demoContext.MatchData.Kills), Rounds: len(demoContext.MatchData.Rounds),
	}, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("trailing JSON value")
}

func pathWithin(root, target string) bool {
	if strings.TrimSpace(target) == "" {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	targetInfo, err := os.Lstat(targetAbs)
	if err != nil || !targetInfo.Mode().IsRegular() || targetInfo.Mode()&os.ModeSymlink != 0 {
		return false
	}
	targetResolved, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(root, targetResolved)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", digest.Sum(nil)), nil
}

func canonicalArtifactMetadataByPath(bundle string) (map[string]canonicalArtifactMetadata, string, error) {
	data, err := os.ReadFile(filepath.Join(bundle, "canonical", "manifest.json"))
	if err != nil {
		return nil, "", fmt.Errorf("read canonical manifest: %w", err)
	}
	var manifest canonicalManifestMetadata
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.SchemaID == "" {
		return nil, "", errors.New("decode canonical manifest")
	}
	artifacts := make(map[string]canonicalArtifactMetadata, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		clean := filepath.Clean(filepath.FromSlash(artifact.Path))
		if artifact.Path == "" || filepath.IsAbs(clean) || clean == ".." ||
			strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, "", errors.New("invalid canonical artifact path")
		}
		artifacts[filepath.ToSlash(filepath.Join("canonical", clean))] = artifact
	}
	return artifacts, manifest.SchemaID, nil
}

func buildArtifactCatalog(bundle string) ([]exportArtifact, error) {
	canonicalArtifacts, canonicalManifestSchemaID, err := canonicalArtifactMetadataByPath(bundle)
	if err != nil {
		return nil, err
	}
	artifacts := make([]exportArtifact, 0)
	err = filepath.Walk(bundle, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("unsupported staged artifact")
		}
		relative, err := filepath.Rel(bundle, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "manifest.json" {
			return nil
		}
		format, compression := inferArtifactEncoding(relative)
		artifact := exportArtifact{
			ArtifactType: inferArtifactType(relative), Path: relative,
			Format: format, Compression: compression, Bytes: info.Size(),
		}
		if metadata, ok := canonicalArtifacts[relative]; ok {
			artifact.ArtifactType, artifact.Format, artifact.Compression =
				metadata.ArtifactType, metadata.Format, metadata.Compression
			artifact.SchemaID, artifact.RecordCount = metadata.SchemaID, metadata.RecordCount
		} else if relative == "canonical/manifest.json" {
			artifact.SchemaID = canonicalManifestSchemaID
		}
		artifact.SHA256, err = hashFile(path)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, artifact)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	return artifacts, nil
}

func inferArtifactType(relativePath string) string {
	if relativePath == "canonical/manifest.json" {
		return "canonical_manifest"
	}
	name := strings.ToLower(relativePath)
	for _, suffix := range []string{".zst", ".gz"} {
		name = strings.TrimSuffix(name, suffix)
	}
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.NewReplacer("/", "_", "\\", "_", "-", "_", ".", "_").Replace(name)
	name = strings.Trim(name, "_")
	if name == "" {
		return "artifact"
	}
	return name
}

func inferArtifactEncoding(relativePath string) (string, string) {
	pathWithoutCompression := strings.ToLower(relativePath)
	compression := "none"
	if strings.HasSuffix(pathWithoutCompression, ".zst") {
		compression, pathWithoutCompression = "zstd", strings.TrimSuffix(pathWithoutCompression, ".zst")
	} else if strings.HasSuffix(pathWithoutCompression, ".gz") {
		compression, pathWithoutCompression = "gzip", strings.TrimSuffix(pathWithoutCompression, ".gz")
	}
	format := strings.TrimPrefix(filepath.Ext(pathWithoutCompression), ".")
	if format == "" {
		format = "binary"
	}
	return format, compression
}
