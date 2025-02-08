package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"cs2-demo-service/db"
	"cs2-demo-service/demo"
	"cs2-demo-service/models"

	"github.com/gorilla/mux"
)

// HandleProcessDownloadedDemo -> POST /process-downloaded-demo
func HandleProcessDownloadedDemo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		SteamID  string `json:"steam_id"`
		Filename string `json:"filename"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Error al parsear JSON", http.StatusBadRequest)
		return
	}
	if payload.SteamID == "" || payload.Filename == "" {
		http.Error(w, "Faltan steam_id o filename", http.StatusBadRequest)
		return
	}

	demoPath := filepath.Join("..", "data", "demos", payload.Filename)
	if _, err := os.Stat(demoPath); os.IsNotExist(err) {
		http.Error(w, "No se encontró el archivo .dem", http.StatusNotFound)
		return
	}

	parseResult, err := demo.ProcessDemoFile(demoPath, payload.SteamID)
	if err != nil {
		log.Printf("Error al procesar la demo: %v", err)
		http.Error(w, "Error interno al procesar la demo", http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf("processed_demos:%s", payload.SteamID)
	entries, _ := db.Rdb.LRange(db.Ctx, key, 0, -1).Result()
	for _, e := range entries {
		var existing models.DemoParseResult
		if json.Unmarshal([]byte(e), &existing) == nil {
			if existing.MatchID == parseResult.MatchID {
				response := map[string]interface{}{
					"status":  "warning",
					"message": "Partida ya existente en Redis.",
					"data":    existing,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}
		}
	}

	data, _ := json.Marshal(parseResult)
	db.Rdb.RPush(db.Ctx, key, string(data))

	response := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Demo %s procesada y almacenada", payload.Filename),
		"data": map[string]interface{}{
			"match_id":       parseResult.MatchID,
			"map_name":       parseResult.MapName,
			"date":           parseResult.Date,
			"duration":       parseResult.Duration,
			"team_score":     parseResult.TeamScore,
			"opponent_score": parseResult.OpponentScore,
			"players":        parseResult.Players,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetProcessedDemos -> GET /get-processed-demos?steam_id=XXXX
func HandleGetProcessedDemos(w http.ResponseWriter, r *http.Request) {
	steamID := r.URL.Query().Get("steam_id")
	if steamID == "" {
		http.Error(w, "Falta el parámetro steam_id", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("processed_demos:%s", steamID)
	entries, err := db.Rdb.LRange(db.Ctx, key, 0, -1).Result()
	if err != nil {
		http.Error(w, "Error al leer Redis", http.StatusInternalServerError)
		return
	}

	demosMap := make(map[string]models.DemoParseResult)
	for _, e := range entries {
		var d models.DemoParseResult
		if json.Unmarshal([]byte(e), &d) == nil {
			demosMap[d.MatchID] = d
		}
	}

	var demos []models.DemoParseResult
	for _, v := range demosMap {
		demos = append(demos, v)
	}

	resp := map[string]interface{}{
		"status": "success",
		"demos":  demos,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGetMatchByID -> GET /match/{steamID}/{matchID}
func HandleGetMatchByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamID"]
	matchID := vars["matchID"]
	if steamID == "" || matchID == "" {
		http.Error(w, "Faltan parámetros /match/{steamID}/{matchID}", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("processed_demos:%s", steamID)
	entries, err := db.Rdb.LRange(db.Ctx, key, 0, -1).Result()
	if err != nil {
		http.Error(w, "Error al leer Redis", http.StatusInternalServerError)
		return
	}

	var found *models.DemoParseResult
	for _, e := range entries {
		var demoData models.DemoParseResult
		if json.Unmarshal([]byte(e), &demoData) == nil {
			if demoData.MatchID == matchID {
				found = &demoData
				break
			}
		}
	}

	if found == nil {
		http.Error(w, "No se encontró ese match", http.StatusNotFound)
		return
	}

	for _, player := range found.Players {
		log.Printf("Jugador: %s | Posición: %d", player.Name, player.Position)
	}

	response := map[string]interface{}{
		"match_id":       found.MatchID,
		"map_name":       found.MapName,
		"date":           found.Date,
		"duration":       found.Duration,
		"team_score":     found.TeamScore,
		"opponent_score": found.OpponentScore,
		"players":        found.Players,
	}
	log.Printf("Datos enviados al frontend: %+v", found)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
