package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cs2-demo-service/db"
	"cs2-demo-service/parser"

	"github.com/gorilla/mux"
)

// matchIDRegex only allows alphanumeric, hyphens, underscores
var matchIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

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

// ProcessDemoRequest represents the JSON body from Node service
type ProcessDemoRequest struct {
	DemoPath      string `json:"demo_path"`
	SteamID       string `json:"steam_id"`
	MatchID       string `json:"match_id"`
	MatchDate     string `json:"match_date"`     // ISO 8601 date from Steam GC
	MatchDuration int    `json:"match_duration"` // Duration in seconds from GC
}

// HandleProcessDemo procesa una demo y devuelve el JSON
func HandleProcessDemo(w http.ResponseWriter, r *http.Request) {
	log.Println("📥 Procesando demo...")

	// Parse JSON body from Node service
	var req ProcessDemoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Error decodificando JSON: %v", err)
		http.Error(w, "Error decodificando JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if req.DemoPath == "" {
		http.Error(w, "Falta demo_path", http.StatusBadRequest)
		return
	}

	// Validate demo_path is under allowed directory (prevent path traversal)
	demosBaseDir := filepath.Join("..", "data", "demos")
	if !isPathSafe(demosBaseDir, req.DemoPath) {
		log.Printf("❌ Path traversal attempt blocked: %s", req.DemoPath)
		http.Error(w, "Invalid demo path", http.StatusBadRequest)
		return
	}

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(req.DemoPath), ".dem") {
		log.Printf("❌ Invalid file extension: %s", req.DemoPath)
		http.Error(w, "Invalid file type: only .dem files allowed", http.StatusBadRequest)
		return
	}

	// Use provided matchID or generate one
	matchID := req.MatchID
	if matchID == "" {
		matchID = fmt.Sprintf("match_%d", len(req.DemoPath)%100000)
	}

	// Sanitize matchID
	matchID, err := sanitizeMatchID(matchID)
	if err != nil {
		http.Error(w, "Invalid match_id format", http.StatusBadRequest)
		return
	}

	log.Printf("📁 Demo path: %s", req.DemoPath)
	log.Printf("📅 Match date: %s", req.MatchDate)

	// Check if file exists and is valid
	fileInfo, err := os.Stat(req.DemoPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("❌ Demo file does not exist: %s", req.DemoPath)
			http.Error(w, fmt.Sprintf("Demo file not found: %s", req.DemoPath), http.StatusNotFound)
			return
		}
		log.Printf("❌ Error accessing demo file: %v", err)
		http.Error(w, fmt.Sprintf("Error accessing file: %v", err), http.StatusInternalServerError)
		return
	}

	// Check file size - a valid demo should be at least a few MB
	if fileInfo.Size() < 1024*100 { // Less than 100KB is definitely corrupt
		log.Printf("❌ Demo file too small (%d bytes), likely corrupt or incomplete: %s", fileInfo.Size(), req.DemoPath)
		http.Error(w, fmt.Sprintf("Demo file too small (%d bytes), likely corrupt", fileInfo.Size()), http.StatusBadRequest)
		return
	}
	log.Printf("📊 Demo file size: %.2f MB", float64(fileInfo.Size())/(1024*1024))

	// ⏱️ TIMING: ParseDemo
	parseStart := time.Now()
	// Parsear demo usando nueva arquitectura
	ctx, err := parser.ParseDemo(req.DemoPath)
	if err != nil {
		log.Printf("❌ Error parseando demo: %v", err)
		http.Error(w, fmt.Sprintf("Error parseando: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("⏱️ ParseDemo took: %v", time.Since(parseStart))

	matchData := ctx.MatchData

	// Asignar matchID
	matchData.MatchID = matchID

	exportBaseDir := "../data/exports"

	// ⏱️ TIMING: ExportAIModels
	exportStart := time.Now()
	// Exportar AI Models (pass match_date directly)
	err = parser.ExportAIModels(ctx, matchID, exportBaseDir, req.MatchDate)
	if err != nil {
		log.Printf("⚠️  Error exportando AI models: %v", err)
	}
	log.Printf("⏱️ ExportAIModels took: %v", time.Since(exportStart))

	// Guardar en Redis

	err = db.SaveMatchData(matchID, matchData)
	if err != nil {
		log.Printf("⚠️  Error guardando en Redis: %v", err)
	}

	// Devolver JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"match_id": matchID,
		"kills":    len(matchData.Kills),
		"rounds":   len(matchData.Rounds),
	})

	log.Printf("✅ Demo procesada: %s (%d kills, %d rounds)", matchID, len(matchData.Kills), len(matchData.Rounds))
}

// HandleHealth retorna el estado del servicio
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "cs2-demo-parser",
	})
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
