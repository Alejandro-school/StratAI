package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"cs2-demo-service/db"
	"cs2-demo-service/parser"

	"github.com/gorilla/mux"
)

// HandleProcessDemo procesa una demo y devuelve el JSON
func HandleProcessDemo(w http.ResponseWriter, r *http.Request) {
	log.Println("📥 Procesando demo...")

	// Leer archivo demo del body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error leyendo body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Extraer matchID del header o generar uno
	matchID := r.Header.Get("X-Match-ID")
	if matchID == "" {
		matchID = fmt.Sprintf("match_%d", len(body)%100000)
	}

	// Guardar demo temporal
	demoPath := filepath.Join("../data/demos", fmt.Sprintf("match_%s.dem", matchID))
	err = os.WriteFile(demoPath, body, 0644)
	if err != nil {
		http.Error(w, "Error guardando demo", http.StatusInternalServerError)
		return
	}

	// Parsear demo usando nueva arquitectura
	ctx, err := parser.ParseDemo(demoPath)
	if err != nil {
		log.Printf("❌ Error parseando demo: %v", err)
		http.Error(w, fmt.Sprintf("Error parseando: %v", err), http.StatusInternalServerError)
		return
	}

	matchData := ctx.MatchData

	// Asignar matchID
	matchData.MatchID = matchID

	// NUEVO: Exportar Timeline (formato coaching)
	exportBaseDir := "../data/exports"
	err = parser.ExportTimelineData(ctx, matchID, exportBaseDir)
	if err != nil {
		log.Printf("⚠️  Error exportando timeline: %v", err)
	}

	// NUEVO: Exportar Analysis (optimización queries)
	err = parser.ExportAnalysisData(ctx, matchID, exportBaseDir)
	if err != nil {
		log.Printf("⚠️  Error exportando analysis: %v", err)
	}

	// LEGACY: REMOVED
	// exportDir := filepath.Join("../data/exports", fmt.Sprintf("%s_%d-%d", matchData.MapName, matchData.CTScore, matchData.TScore))
	// err = parser.ExportOptimizedData(matchData, exportDir)
	// if err != nil {
	// 	log.Printf("⚠️  Error exportando archivos legacy: %v", err)
	// }

	// Guardar en Redis
	err = db.SaveMatchData(matchID, matchData)
	if err != nil {
		log.Printf("⚠️  Error guardando en Redis: %v", err)
	}

	// Devolver JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchData)

	log.Printf("✅ Demo procesada: %s (%d kills, %d rounds, %d timeline events)", matchID, len(matchData.Kills), len(matchData.Rounds), len(ctx.Timeline))
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

	matchData, err := db.GetMatchData(matchID)
	if err != nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchData)
}
