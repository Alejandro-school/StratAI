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
	"strconv"
	"strings"
	"time"

	"cs2-demo-service/db"
	"cs2-demo-service/parser"

	"github.com/gorilla/mux"
)

// matchIDRegex only allows alphanumeric, hyphens, underscores
var matchIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
var checksumRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)

var analysisSlots = make(chan struct{}, configuredConcurrency())

const defaultParserSchemaVersion = "1"

type exportManifest struct {
	MatchID             string    `json:"match_id"`
	Checksum            string    `json:"checksum"`
	ParserSchemaVersion string    `json:"parser_schema_version"`
	CommittedAt         time.Time `json:"committed_at"`
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

func hasCommittedManifest(finalDir, checksum, schemaVersion string) bool {
	data, err := os.ReadFile(filepath.Join(finalDir, "manifest.json"))
	if err != nil {
		return false
	}
	var manifest exportManifest
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	return manifest.Checksum == checksum && manifest.ParserSchemaVersion == schemaVersion
}

func writeIdempotentSuccess(w http.ResponseWriter, matchID string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success", "match_id": matchID, "idempotent": true,
	})
}

func commitExport(stagedDir, finalDir string) error {
	backupDir := ""
	if _, err := os.Stat(finalDir); err == nil {
		backupDir = fmt.Sprintf("%s.replaced.%d", finalDir, time.Now().UnixNano())
		if err := os.Rename(finalDir, backupDir); err != nil {
			return fmt.Errorf("could not preserve previous export: %w", err)
		}
	}
	if err := os.Rename(stagedDir, finalDir); err != nil {
		if backupDir != "" {
			_ = os.Rename(backupDir, finalDir)
		}
		return fmt.Errorf("could not commit export: %w", err)
	}
	if backupDir != "" {
		_ = os.RemoveAll(backupDir)
	}
	return nil
}

func analysisLockKey(matchID, checksum, schemaVersion string) string {
	namespace := os.Getenv("PIPELINE_NAMESPACE")
	if namespace == "" {
		namespace = "stratai:v2"
	}
	identity := sha256.Sum256([]byte(matchID + ":" + checksum + ":" + schemaVersion))
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
	DemoPath      string `json:"demo_path"`
	SteamID       string `json:"steam_id"`
	MatchID       string `json:"match_id"`
	MatchDate     string `json:"match_date"`     // ISO 8601 date from Steam GC
	MatchDuration int    `json:"match_duration"` // Duration in seconds from GC
	Checksum      string `json:"checksum"`
	SchemaVersion string `json:"parser_schema_version"`
	JobID         string `json:"job_id"`
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
	if hasCommittedManifest(finalDir, req.Checksum, req.SchemaVersion) {
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

	tempParent := filepath.Join(exportBaseDir, ".tmp")
	if err := os.MkdirAll(tempParent, 0o750); err != nil {
		processingErrors.WithLabelValues("export").Inc()
		http.Error(w, "could not prepare export", http.StatusInternalServerError)
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
	err = parser.ExportAIModels(demoContext, matchID, stagingRoot, req.MatchDate)
	processingDuration.WithLabelValues("export").Observe(time.Since(phaseStart).Seconds())
	if err != nil {
		processingErrors.WithLabelValues("export").Inc()
		http.Error(w, "could not export required artifacts", http.StatusInternalServerError)
		return
	}
	stagedDir := filepath.Join(stagingRoot, "match_"+matchID)
	manifest := exportManifest{
		MatchID:             matchID,
		Checksum:            req.Checksum,
		ParserSchemaVersion: req.SchemaVersion,
		CommittedAt:         time.Now().UTC(),
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
	err = db.SaveMatchData(matchID, matchData)
	processingDuration.WithLabelValues("redis").Observe(time.Since(phaseStart).Seconds())
	if err != nil {
		processingErrors.WithLabelValues("redis").Inc()
		http.Error(w, "could not persist match data", http.StatusServiceUnavailable)
		return
	}

	phaseStart = time.Now()
	if err := commitExport(stagedDir, finalDir); err != nil {
		processingErrors.WithLabelValues("commit").Inc()
		http.Error(w, "could not commit match artifacts", http.StatusInternalServerError)
		return
	}
	processingDuration.WithLabelValues("commit").Observe(time.Since(phaseStart).Seconds())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"match_id": matchID,
		"kills":    len(matchData.Kills),
		"rounds":   len(matchData.Rounds),
	})
	logJSON("info", "demo_processing_committed", map[string]interface{}{
		"job_id": req.JobID, "match_id": matchID, "checksum": req.Checksum,
	})
}

// HandleHealth retorna el estado del servicio
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "cs2-demo-parser",
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
