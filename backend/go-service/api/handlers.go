package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cs2-demo-service/db"
	"cs2-demo-service/models"
	"cs2-demo-service/parser"

	"github.com/gorilla/mux"
)

// matchIDRegex only allows alphanumeric, hyphens, underscores
var matchIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
var checksumRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)

var analysisSlots = make(chan struct{}, configuredConcurrency())

const (
	defaultParserSchemaVersion = "v16"
	exportFormatVersion        = "3.8.0"
)

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
	ExportFormatVersion string           `json:"export_format_version,omitempty"`
	ValidatorVersion    string           `json:"validator_version"`
	ValidationStatus    string           `json:"validation_status"`
	Artifacts           []exportArtifact `json:"artifacts,omitempty"`
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

func logJSON(level, event string, fields map[string]interface{}) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level,
		"service":   "cs2-demo-analyzer",
		"event":     event,
	}
	for key, value := range fields {
		entry[key] = value
	}
	if data, err := json.Marshal(entry); err == nil {
		log.Print(string(data))
	}
}

func configuredConcurrency() int {
	value, err := strconv.Atoi(os.Getenv("GO_MAX_CONCURRENT_DEMOS"))
	if err != nil || value < 1 {
		return 2
	}
	return value
}

// sanitizeMatchID validates and sanitizes a match ID to prevent path traversal.
func sanitizeMatchID(id string) (string, error) {
	if !matchIDRegex.MatchString(id) {
		return "", fmt.Errorf("invalid match ID format")
	}
	return id, nil
}

// isPathSafe checks that resolved path is under the base directory.
func isPathSafe(basePath, targetPath string) bool {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) || absTarget == absBase
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func canonicalArtifactMetadataByPath(stagedDir string) (map[string]canonicalArtifactMetadata, string, error) {
	manifestPath := filepath.Join(stagedDir, "canonical", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return map[string]canonicalArtifactMetadata{}, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("read canonical manifest: %w", err)
	}

	var manifest canonicalManifestMetadata
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, "", fmt.Errorf("decode canonical manifest: %w", err)
	}
	if manifest.SchemaID == "" {
		return nil, "", fmt.Errorf("canonical manifest is missing schema_id")
	}

	artifacts := make(map[string]canonicalArtifactMetadata, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		canonicalPath := filepath.Clean(filepath.FromSlash(artifact.Path))
		if artifact.Path == "" || filepath.IsAbs(canonicalPath) || canonicalPath == ".." || strings.HasPrefix(canonicalPath, ".."+string(filepath.Separator)) {
			return nil, "", fmt.Errorf("canonical manifest contains invalid artifact path %q", artifact.Path)
		}
		rootRelativePath := filepath.ToSlash(filepath.Join("canonical", canonicalPath))
		artifacts[rootRelativePath] = artifact
	}

	return artifacts, manifest.SchemaID, nil
}

func inferArtifactType(relativePath string) string {
	switch relativePath {
	case "canonical/manifest.json":
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
	switch {
	case strings.HasSuffix(pathWithoutCompression, ".zst"):
		compression = "zstd"
		pathWithoutCompression = strings.TrimSuffix(pathWithoutCompression, ".zst")
	case strings.HasSuffix(pathWithoutCompression, ".gz"):
		compression = "gzip"
		pathWithoutCompression = strings.TrimSuffix(pathWithoutCompression, ".gz")
	}

	switch filepath.Ext(pathWithoutCompression) {
	case ".json":
		return "json", compression
	case ".jsonl":
		return "jsonl", compression
	case ".parquet":
		return "parquet", compression
	case ".csv":
		return "csv", compression
	default:
		format := strings.TrimPrefix(filepath.Ext(pathWithoutCompression), ".")
		if format == "" {
			format = "binary"
		}
		return format, compression
	}
}

func buildArtifactCatalog(stagedDir string) ([]exportArtifact, error) {
	canonicalArtifacts, canonicalManifestSchemaID, err := canonicalArtifactMetadataByPath(stagedDir)
	if err != nil {
		return nil, err
	}

	artifacts := make([]exportArtifact, 0)
	err = filepath.Walk(stagedDir, func(artifactPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported staged artifact %s", artifactPath)
		}

		relativePath, err := filepath.Rel(stagedDir, artifactPath)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if relativePath == "manifest.json" {
			return nil
		}

		format, compression := inferArtifactEncoding(relativePath)
		artifact := exportArtifact{
			ArtifactType: inferArtifactType(relativePath),
			Path:         relativePath,
			Format:       format,
			Compression:  compression,
			Bytes:        info.Size(),
		}
		if metadata, ok := canonicalArtifacts[relativePath]; ok {
			if metadata.ArtifactType != "" {
				artifact.ArtifactType = metadata.ArtifactType
			}
			if metadata.Format != "" {
				artifact.Format = metadata.Format
			}
			if metadata.Compression != "" {
				artifact.Compression = metadata.Compression
			}
			artifact.SchemaID = metadata.SchemaID
			artifact.RecordCount = metadata.RecordCount
		} else if relativePath == "canonical/manifest.json" {
			artifact.SchemaID = canonicalManifestSchemaID
		}

		artifact.SHA256, err = hashFile(artifactPath)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, artifact)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("catalog staged artifacts: %w", err)
	}

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Path < artifacts[j].Path
	})
	return artifacts, nil
}

func hasCommittedManifest(finalDir, checksum, schemaVersion string) bool {
	data, err := os.ReadFile(filepath.Join(finalDir, "manifest.json"))
	if err != nil {
		return false
	}
	var manifest exportManifest
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	if manifest.Checksum != checksum ||
		manifest.ParserSchemaVersion != schemaVersion ||
		manifest.ExportFormatVersion != exportFormatVersion ||
		manifest.ValidatorVersion != canonicalValidatorVersion ||
		manifest.ValidationStatus != "passed" || manifest.CommittedAt.IsZero() || len(manifest.Artifacts) == 0 {
		return false
	}
	declared := make(map[string]struct{}, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		clean := filepath.Clean(filepath.FromSlash(artifact.Path))
		if artifact.Path == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) ||
			(clean != "canonical" && !strings.HasPrefix(clean, "canonical"+string(filepath.Separator))) {
			return false
		}
		if _, exists := declared[filepath.ToSlash(clean)]; exists {
			return false
		}
		path := filepath.Join(finalDir, clean)
		if !isExactChild(finalDir, path) {
			return false
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() != artifact.Bytes {
			return false
		}
		actualHash, err := hashFile(path)
		if err != nil || actualHash != artifact.SHA256 {
			return false
		}
		declared[filepath.ToSlash(clean)] = struct{}{}
	}
	disk := make(map[string]struct{}, len(declared))
	if err := filepath.Walk(finalDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || path == filepath.Join(finalDir, "manifest.json") {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular artifact")
		}
		relative, err := filepath.Rel(finalDir, path)
		if err != nil {
			return err
		}
		disk[filepath.ToSlash(relative)] = struct{}{}
		return nil
	}); err != nil {
		return false
	}
	if len(disk) != len(declared) {
		return false
	}
	for path := range disk {
		if _, ok := declared[path]; !ok {
			return false
		}
	}
	return true
}

func writeIdempotentSuccess(w http.ResponseWriter, matchID string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success", "match_id": matchID, "idempotent": true,
	})
}

func commitExport(stagedDir, finalDir string) error {
	transaction, err := beginPublicationCommit(stagedDir, finalDir, fmt.Sprintf("legacy-%d", time.Now().UnixNano()))
	if err != nil {
		return err
	}
	return transaction.Finalize()
}

func analysisLockKey(matchID, checksum, schemaVersion string) string {
	namespace := os.Getenv("PIPELINE_NAMESPACE")
	if namespace == "" {
		namespace = "stratai:v2"
	}
	identity := sha256.Sum256([]byte(matchID))
	return fmt.Sprintf("%s:analysis-lock:%x", namespace, identity)
}

func releaseAnalysisLock(ctx context.Context, key, token string) {
	if db.Rdb == nil {
		return
	}
	releaseContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = db.Rdb.Eval(
		releaseContext,
		`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`,
		[]string{key},
		token,
	).Err()
}

// ProcessDemoRequest represents the JSON body from Node service
type ProcessDemoRequest struct {
	DemoPath        string `json:"demo_path"`
	SteamID         string `json:"steam_id"`
	MatchID         string `json:"match_id"`
	MatchDate       string `json:"match_date"`     // ISO 8601 date from Steam GC
	MatchDuration   int    `json:"match_duration"` // Duration in seconds from GC
	Checksum        string `json:"checksum"`
	SchemaVersion   string `json:"parser_schema_version"`
	JobID           string `json:"job_id"`
	Force           bool   `json:"force"`
	Source          string `json:"source"`
	SourceEndpoint  string `json:"source_endpoint"`
	SourceQueriedAt string `json:"source_queried_at"`
	SourceVersion   string `json:"source_version"`
}

// HandleProcessDemo procesa una demo y devuelve el JSON
func HandleProcessDemo(w http.ResponseWriter, r *http.Request) {
	var req ProcessDemoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.DemoPath == "" || req.MatchID == "" || !checksumRegex.MatchString(req.Checksum) {
		http.Error(w, "demo_path, match_id and checksum are required", http.StatusBadRequest)
		return
	}
	if req.SchemaVersion == "" {
		req.SchemaVersion = defaultParserSchemaVersion
	}
	if req.JobID == "" {
		req.JobID = req.MatchID
	}
	logJSON("info", "demo_processing_started", map[string]interface{}{
		"job_id": req.JobID, "match_id": req.MatchID,
	})

	demosBaseDir := filepath.Join("..", "data", "demos")
	if !isPathSafe(demosBaseDir, req.DemoPath) {
		http.Error(w, "Invalid demo path", http.StatusBadRequest)
		return
	}
	if !strings.HasSuffix(strings.ToLower(req.DemoPath), ".dem") {
		http.Error(w, "Invalid file type: only .dem files allowed", http.StatusBadRequest)
		return
	}

	matchID, err := sanitizeMatchID(req.MatchID)
	if err != nil {
		http.Error(w, "Invalid match_id format", http.StatusBadRequest)
		return
	}

	select {
	case analysisSlots <- struct{}{}:
		activeProcessing.Inc()
		defer func() {
			<-analysisSlots
			activeProcessing.Dec()
		}()
	case <-r.Context().Done():
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
		return
	}

	fileInfo, err := os.Stat(req.DemoPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "demo file not found", http.StatusNotFound)
			return
		}
		processingErrors.WithLabelValues("validate").Inc()
		http.Error(w, "could not access demo file", http.StatusInternalServerError)
		return
	}
	if fileInfo.Size() < 1024*100 {
		http.Error(w, "demo file is incomplete", http.StatusBadRequest)
		return
	}
	demoSizeBytes.Observe(float64(fileInfo.Size()))

	exportBaseDir := filepath.Join("..", "data", "exports")
	finalDir := filepath.Join(exportBaseDir, "match_"+matchID)
	if !req.Force && hasCommittedManifest(finalDir, req.Checksum, req.SchemaVersion) {
		writeIdempotentSuccess(w, matchID)
		return
	}

	phaseStart := time.Now()
	actualChecksum, err := hashFile(req.DemoPath)
	processingDuration.WithLabelValues("checksum").Observe(time.Since(phaseStart).Seconds())
	if err != nil || actualChecksum != req.Checksum {
		processingErrors.WithLabelValues("checksum").Inc()
		http.Error(w, "demo checksum mismatch", http.StatusBadRequest)
		return
	}

	lockKey := analysisLockKey(matchID, req.Checksum, req.SchemaVersion)
	lockToken := fmt.Sprintf("%s-%d", req.JobID, time.Now().UnixNano())
	if db.Rdb == nil {
		processingErrors.WithLabelValues("lock").Inc()
		http.Error(w, "analysis lock unavailable", http.StatusServiceUnavailable)
		return
	}
	acquired, err := db.Rdb.SetNX(
		r.Context(),
		lockKey,
		lockToken,
		30*time.Minute,
	).Result()
	if err != nil {
		processingErrors.WithLabelValues("lock").Inc()
		http.Error(w, "analysis lock unavailable", http.StatusServiceUnavailable)
		return
	}
	if !acquired {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			if hasCommittedManifest(finalDir, req.Checksum, req.SchemaVersion) {
				writeIdempotentSuccess(w, matchID)
				return
			}
			select {
			case <-r.Context().Done():
				http.Error(w, "matching analysis is already in progress", http.StatusConflict)
				return
			case <-ticker.C:
			}
		}
	}
	defer releaseAnalysisLock(context.Background(), lockKey, lockToken)
	tempParent := filepath.Join(exportBaseDir, ".tmp")
	if err := os.MkdirAll(tempParent, 0o750); err != nil {
		processingErrors.WithLabelValues("rollback_failure").Inc()
		http.Error(w, "could not prepare publication recovery", http.StatusInternalServerError)
		return
	}
	if err := recoverInterruptedPublication(finalDir, tempParent, matchID, req.Checksum, req.SchemaVersion); err != nil {
		processingErrors.WithLabelValues("rollback_failure").Inc()
		logJSON("error", "publication_recovery_failed", map[string]interface{}{
			"job_id": req.JobID, "match_id": matchID, "error": err.Error(),
		})
		http.Error(w, "interrupted publication could not be recovered", http.StatusInternalServerError)
		return
	}
	if !req.Force && hasCommittedManifest(finalDir, req.Checksum, req.SchemaVersion) {
		writeIdempotentSuccess(w, matchID)
		return
	}

	if err := r.Context().Err(); err != nil {
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
		return
	}

	phaseStart = time.Now()
	demoContext, err := parser.ParseDemo(req.DemoPath)
	processingDuration.WithLabelValues("parse").Observe(time.Since(phaseStart).Seconds())
	if err != nil {
		processingErrors.WithLabelValues("parse").Inc()
		http.Error(w, "could not parse demo", http.StatusUnprocessableEntity)
		return
	}
	matchData := demoContext.MatchData
	matchData.MatchID = matchID
	if err := r.Context().Err(); err != nil {
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
		return
	}

	stagingRoot, err := os.MkdirTemp(tempParent, matchID+"-")
	if err != nil {
		processingErrors.WithLabelValues("export").Inc()
		http.Error(w, "could not prepare export", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(stagingRoot)

	phaseStart = time.Now()
	source := req.Source
	if strings.TrimSpace(source) == "" {
		source = "demo"
	}
	err = parser.ExportMatchBundleWithProvenance(
		demoContext,
		matchID,
		stagingRoot,
		req.MatchDate,
		models.CanonicalExportProvenance{
			DemoChecksum:    req.Checksum,
			BuildIdentifier: strings.TrimSpace(os.Getenv("STRATAI_BUILD_ID")),
			Source:          source,
			Endpoint:        req.SourceEndpoint,
			QueriedAt:       req.SourceQueriedAt,
			SourceVersion:   req.SourceVersion,
		},
	)
	processingDuration.WithLabelValues("export").Observe(time.Since(phaseStart).Seconds())
	if err != nil {
		failureType := classifyGoExportFailure(err)
		processingErrors.WithLabelValues(failureType).Inc()
		log.Printf("match %s export failed: %v", matchID, err)
		http.Error(w, "could not export required artifacts", http.StatusInternalServerError)
		return
	}
	stagedDir := filepath.Join(stagingRoot, "match_"+matchID)
	artifacts, err := buildArtifactCatalog(stagedDir)
	if err != nil {
		processingErrors.WithLabelValues("export").Inc()
		http.Error(w, "could not catalog export artifacts", http.StatusInternalServerError)
		return
	}
	manifest := exportManifest{
		MatchID:             matchID,
		Checksum:            req.Checksum,
		ParserSchemaVersion: req.SchemaVersion,
		CommittedAt:         time.Now().UTC(),
		ExportFormatVersion: exportFormatVersion,
		ValidatorVersion:    canonicalValidatorVersion,
		ValidationStatus:    "passed",
		Artifacts:           artifacts,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		processingErrors.WithLabelValues("export").Inc()
		http.Error(w, "could not create export manifest", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(filepath.Join(stagedDir, "manifest.json"), manifestBytes, 0o640); err != nil {
		processingErrors.WithLabelValues("export").Inc()
		http.Error(w, "could not create export manifest", http.StatusInternalServerError)
		return
	}

	phaseStart = time.Now()
	receipt, err := stagedBundleValidator(r.Context(), stagedDir, matchID, req.Checksum)
	processingDuration.WithLabelValues("validate_bundle").Observe(time.Since(phaseStart).Seconds())
	if err != nil {
		kind := publicationFailureKind(err, "contract_validation")
		processingErrors.WithLabelValues(kind).Inc()
		logJSON("error", "staged_bundle_rejected", map[string]interface{}{
			"job_id": req.JobID, "match_id": matchID, "failure_type": kind, "error": err.Error(),
		})
		http.Error(w, "staged export failed canonical validation", http.StatusUnprocessableEntity)
		return
	}

	phaseStart = time.Now()
	redisSnapshot, err := snapshotMatchDataWithRetry(r.Context(), matchID)
	processingDuration.WithLabelValues("redis_snapshot").Observe(time.Since(phaseStart).Seconds())
	if err != nil {
		processingErrors.WithLabelValues("commit_failure").Inc()
		http.Error(w, "could not prepare Redis publication transaction", http.StatusServiceUnavailable)
		return
	}
	phaseStart = time.Now()
	transaction, err := beginPublicationCommit(stagedDir, finalDir, lockToken)
	if err != nil {
		kind := publicationFailureKind(err, "commit_failure")
		processingErrors.WithLabelValues(kind).Inc()
		http.Error(w, "could not commit match artifacts", http.StatusInternalServerError)
		return
	}
	processingDuration.WithLabelValues("commit").Observe(time.Since(phaseStart).Seconds())

	phaseStart = time.Now()
	err = saveMatchDataWithRetry(r.Context(), matchID, matchData)
	processingDuration.WithLabelValues("redis").Observe(time.Since(phaseStart).Seconds())
	if err != nil {
		redisRollbackErr := restoreMatchDataWithRetry(context.Background(), matchID, redisSnapshot)
		filesystemRollbackErr := transaction.Rollback()
		if redisRollbackErr != nil || filesystemRollbackErr != nil {
			processingErrors.WithLabelValues("rollback_failure").Inc()
			logJSON("error", "publication_rollback_failed", map[string]interface{}{
				"job_id": req.JobID, "match_id": matchID, "redis_error": fmt.Sprint(redisRollbackErr),
				"filesystem_error": fmt.Sprint(filesystemRollbackErr),
			})
			http.Error(w, "publication rollback failed", http.StatusInternalServerError)
			return
		}
		processingErrors.WithLabelValues("commit_failure").Inc()
		http.Error(w, "could not persist match data; publication rolled back", http.StatusServiceUnavailable)
		return
	}
	if err := transaction.Finalize(); err != nil {
		logJSON("warning", "publication_backup_cleanup_warning", map[string]interface{}{
			"job_id": req.JobID, "match_id": matchID, "error": err.Error(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "success",
		"match_id":          matchID,
		"kills":             len(matchData.Kills),
		"rounds":            len(matchData.Rounds),
		"validator_version": receipt.ValidatorVersion,
		"coverage_warnings": receipt.Warnings,
	})
	logJSON("info", "demo_processing_committed", map[string]interface{}{
		"job_id": req.JobID, "match_id": matchID, "checksum": req.Checksum,
		"validator_version": receipt.ValidatorVersion, "validation_status": receipt.Status,
		"coverage_warnings": receipt.Warnings,
	})
}

// HandleHealth retorna el estado del servicio
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	redisStatus := "unavailable"
	status := "degraded"
	statusCode := http.StatusServiceUnavailable
	if db.Rdb != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Rdb.Ping(ctx).Err(); err == nil {
			redisStatus = "available"
			status = "ok"
			statusCode = http.StatusOK
		}
	}
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": status, "service": "cs2-demo-parser", "redis": redisStatus,
		"parser_schema_version": defaultParserSchemaVersion,
		"export_format_version": exportFormatVersion,
		"validator_version":     canonicalValidatorVersion,
	})
}

func HandleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if db.Rdb == nil || db.Rdb.Ping(r.Context()).Err() != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"not_ready"}`))
		return
	}
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

// HandleGetMatchDetails obtiene detalles de un match desde Redis
func HandleGetMatchDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["matchID"]

	// Sanitize matchID to prevent injection
	matchID, err := sanitizeMatchID(matchID)
	if err != nil {
		http.Error(w, "Invalid match ID format", http.StatusBadRequest)
		return
	}

	matchData, err := db.GetMatchData(matchID)
	if err != nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchData)
}
