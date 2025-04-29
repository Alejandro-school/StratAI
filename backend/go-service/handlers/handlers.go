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
	"github.com/redis/go-redis/v9"
)

// GetMatchJSONFromRedis intenta obtener de Redis el JSON con matchDuration.
func GetMatchJSONFromRedis(matchID string) (string, error) {
	jsonData, err := db.Rdb.Get(db.Ctx, "match_data:"+matchID).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("No hay datos en Redis para matchID: %s", matchID)
	} else if err != nil {
		return "", fmt.Errorf("Error obteniendo JSON desde Redis: %v", err)
	}
	return jsonData, nil
}

// HandleProcessDownloadedDemo procesa una demo descargada y almacena el resultado básico en Redis.
// Se usa la función ProcessDemoFileBasic para obtener solo los datos necesarios para el frontend.
func HandleProcessDownloadedDemo(w http.ResponseWriter, r *http.Request) {
	log.Println("DEBUG: --> Entró a HandleProcessDownloadedDemo")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		SteamID  string `json:"steam_id"`
		Filename string `json:"filename"`
		MatchID  string `json:"match_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Error al parsear JSON", http.StatusBadRequest)
		return
	}

	if payload.SteamID == "" || payload.Filename == "" || payload.MatchID == "" {
		http.Error(w, "Faltan steam_id, filename o match_id", http.StatusBadRequest)
		return
	}

	// 1) Comprobar si ya procesamos esta demo en Redis
	key := fmt.Sprintf("processed_demos:%s", payload.SteamID)
	entries, _ := db.Rdb.LRange(db.Ctx, key, 0, -1).Result()
	for _, e := range entries {
		var existing models.BasicDemoParseResult
		if json.Unmarshal([]byte(e), &existing) == nil {
			if existing.MatchID == payload.MatchID {
				// Ya existe => devolvemos
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

	// 2) Intentar leer de Redis el JSON con la duración: "match_data:<matchID>"
	matchDataJSON, err := GetMatchJSONFromRedis(payload.MatchID)
	if err == redis.Nil {
		log.Printf("No se encontró match_data para matchID=%s en Redis", payload.MatchID)
		matchDataJSON = "{}"
	} else if err != nil {
		log.Printf("Error leyendo match_data:%s => %v", payload.MatchID, err)
		http.Error(w, "Error al obtener match_data de Redis", http.StatusInternalServerError)
		return
	}

	// 3) Procesamos la demo usando la función básica.
	demoPath := filepath.Join("..", "data", "demos", payload.Filename)
	if _, err := os.Stat(demoPath); os.IsNotExist(err) {
		http.Error(w, "No se encontró el archivo .dem", http.StatusNotFound)
		return
	}

	basicResult, err := demo.ProcessDemoFileBasic(demoPath, payload.SteamID, matchDataJSON)
	if err != nil {
		log.Printf("Error al procesar la demo: %v", err)
		http.Error(w, "Error interno al procesar la demo", http.StatusInternalServerError)
		return
	}

	// Asignamos el matchID que nos mandó el frontend.
	basicResult.MatchID = payload.MatchID

	// 4) Guardamos en Redis la partida parseada (versión básica).
	data, marshalErr := json.Marshal(basicResult)
	if marshalErr != nil {
		log.Printf("[ERROR] Marshal falló: %v", marshalErr)
	} else {
		db.Rdb.RPush(db.Ctx, key, string(data))
		log.Printf("[OK] Procesado matchID=%s para steamID=%s", basicResult.MatchID, payload.SteamID)
	}

	// 5) Respuesta al frontend.
	response := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Demo %s procesada y almacenada", payload.Filename),
		"data": map[string]interface{}{
			"match_id":       basicResult.MatchID,
			"map_name":       basicResult.MapName,
			"date":           basicResult.Date,
			"duration":       basicResult.Duration,
			"team_score":     basicResult.TeamScore,
			"opponent_score": basicResult.OpponentScore,
			"players":        basicResult.Players,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetProcessedDemos retorna todas las demos procesadas (versión básica) para un steam_id.
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

	demosMap := make(map[string]models.BasicDemoParseResult)
	for _, e := range entries {
		var d models.BasicDemoParseResult
		if json.Unmarshal([]byte(e), &d) == nil {
			demosMap[d.MatchID] = d
		}
	}

	var demos []models.BasicDemoParseResult
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

// HandleGetMatchByID retorna los detalles (versión básica) de una demo específica.
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

	var found *models.BasicDemoParseResult
	for _, e := range entries {
		var demoData models.BasicDemoParseResult
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
