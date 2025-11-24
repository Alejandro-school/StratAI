package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"cs2-demo-service/demo"

	"github.com/redis/go-redis/v9"
)

// processingMutex protege el procesamiento concurrente de demos
var processingMutex sync.Mutex

// redisClient cliente global de Redis
var redisClient *redis.Client

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Advertencia: No se pudo conectar a Redis: %v", err)
	}
}

// HandleProcessDemo procesa una demo y devuelve el JSON directamente.
// Espera un payload con:
// - demo_path: ruta absoluta al archivo .dem
// - steam_id: Steam ID del jugador a analizar (opcional, puede ser "")
// - match_id: ID del match para identificación
func HandleProcessDemo(w http.ResponseWriter, r *http.Request) {
	log.Println("📥 Recibida petición para procesar demo")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		DemoPath string `json:"demo_path"`
		SteamID  string `json:"steam_id"`
		MatchID  string `json:"match_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("❌ Error parseando JSON: %v", err)
		http.Error(w, "Error al parsear JSON", http.StatusBadRequest)
		return
	}

	if payload.DemoPath == "" || payload.MatchID == "" {
		http.Error(w, "Faltan demo_path o match_id", http.StatusBadRequest)
		return
	}

	// Verificar que el archivo existe
	if _, err := os.Stat(payload.DemoPath); os.IsNotExist(err) {
		log.Printf("❌ No se encontró el archivo: %s", payload.DemoPath)
		http.Error(w, "No se encontró el archivo .dem", http.StatusNotFound)
		return
	}

	log.Printf("🔍 Procesando: %s", payload.DemoPath)
	log.Printf("   MatchID: %s", payload.MatchID)

	// Bloquear procesamiento concurrente (una demo a la vez)
	processingMutex.Lock()
	defer processingMutex.Unlock()

	log.Println("🔒 Mutex adquirido, iniciando procesamiento...")

	// Medir tiempo de procesamiento
	startTime := time.Now()

	// Procesar la demo
	result, err := demo.ProcessDemoFile(payload.DemoPath, payload.SteamID, "{}", payload.MatchID)
	if err != nil {
		log.Printf("❌ Error: %v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	elapsed := time.Since(startTime)
	log.Printf("⏱️  Tiempo de procesamiento: %.2f segundos", elapsed.Seconds())

	log.Printf("✅ Listo: %s | Duración: %s | Scores: %d-%d | Jugadores: %d",
		result.MapName, result.Duration, result.TeamScore, result.OpponentScore, len(result.Players))

	// Guardar en Redis si SteamID está presente
	if payload.SteamID != "" {
		ctx := context.Background()
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("⚠️  Error al serializar resultado: %v", err)
		} else {
			// Guardar en processed_demos:{steamID}
			key := fmt.Sprintf("processed_demos:%s", payload.SteamID)
			if err := redisClient.RPush(ctx, key, string(resultJSON)).Err(); err != nil {
				log.Printf("⚠️  Error al guardar en Redis: %v", err)
			} else {
				log.Printf("💾 Resultado guardado en Redis: %s", key)

				// === NUEVO: Invalidar caché del dashboard para que se actualice inmediatamente ===
				cacheKey := fmt.Sprintf("dashboard_stats:%s", payload.SteamID)
				if err := redisClient.Del(ctx, cacheKey).Err(); err != nil {
					log.Printf("⚠️  Error al invalidar caché del dashboard: %v", err)
				} else {
					log.Printf("🔄 Caché del dashboard invalidado: %s", cacheKey)
				}
			}
		}
	}

	// Devolver el resultado
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   result,
	})
}

// HandleHealth endpoint de salud para verificar que el servicio está activo
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "cs2-demo-analyzer",
	})
}

// HandleGetMatchDetails obtiene detalles de una partida desde exports/
// Endpoint: GET /match-details/{matchID}
// Retorna datos combinados de match_info.json y players_summary.json
func HandleGetMatchDetails(w http.ResponseWriter, r *http.Request) {
	// Extraer matchID de la URL
	// Formato esperado: /match-details/{matchID}
	pathParts := len(r.URL.Path)
	if pathParts < 15 { // "/match-details/" = 15 chars
		http.Error(w, "MatchID no proporcionado", http.StatusBadRequest)
		return
	}

	matchID := r.URL.Path[15:] // Después de "/match-details/"

	if matchID == "" {
		http.Error(w, "MatchID vacío", http.StatusBadRequest)
		return
	}

	log.Printf("📊 Solicitando detalles del match: %s", matchID)

	// Construir ruta a exports
	exportsPath := "../data/exports/" + matchID

	// Verificar que existe el directorio
	if _, err := os.Stat(exportsPath); os.IsNotExist(err) {
		log.Printf("❌ Match no encontrado: %s", matchID)
		http.Error(w, "Match no encontrado", http.StatusNotFound)
		return
	}

	// Leer match_info.json
	matchInfoPath := exportsPath + "/match_info.json"
	matchInfoData, err := os.ReadFile(matchInfoPath)
	if err != nil {
		log.Printf("❌ Error leyendo match_info.json: %v", err)
		http.Error(w, "Error leyendo información del match", http.StatusInternalServerError)
		return
	}

	var matchInfo map[string]interface{}
	if err := json.Unmarshal(matchInfoData, &matchInfo); err != nil {
		log.Printf("❌ Error parseando match_info.json: %v", err)
		http.Error(w, "Error parseando información del match", http.StatusInternalServerError)
		return
	}

	// Leer players_summary.json
	playersPath := exportsPath + "/players_summary.json"
	playersData, err := os.ReadFile(playersPath)
	if err != nil {
		log.Printf("❌ Error leyendo players_summary.json: %v", err)
		http.Error(w, "Error leyendo jugadores", http.StatusInternalServerError)
		return
	}

	var playersSummary map[string]interface{}
	if err := json.Unmarshal(playersData, &playersSummary); err != nil {
		log.Printf("❌ Error parseando players_summary.json: %v", err)
		http.Error(w, "Error parseando jugadores", http.StatusInternalServerError)
		return
	}

	// Combinar datos en respuesta
	response := map[string]interface{}{
		"match_id":       matchID,
		"map_name":       matchInfo["map_name"],
		"date":           matchInfo["date"],
		"duration":       matchInfo["duration"],
		"team_score":     matchInfo["team_score"],
		"opponent_score": matchInfo["opponent_score"],
		"result":         matchInfo["result"],
		"total_rounds":   matchInfo["total_rounds"],
		"players":        playersSummary["players"], // Ya incluye avatares
	}

	log.Printf("✅ Detalles enviados para match: %s", matchID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
