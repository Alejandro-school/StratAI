package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	"github.com/redis/go-redis/v9"
)

// PlayerStats: estadísticas de un jugador concreto
type PlayerStats struct {
	Name         string  `json:"name"`
	Kills        int     `json:"kills"`
	Assists      int     `json:"assists"`
	Deaths       int     `json:"deaths"`
	ADR          float64 `json:"adr"`
	HSPercentage float64 `json:"hs_percentage"`
	KDRatio      float64 `json:"kd_ratio"`
	FlashAssists int     `json:"flash_assists"`
}

// DemoParseResult: estructura global para cada partida
type DemoParseResult struct {
	MatchID       string        `json:"match_id"`
	MapName       string        `json:"map_name"`
	Duration      string        `json:"duration"`
	Result        string        `json:"result"`
	TeamScore     int           `json:"team_score"`
	OpponentScore int           `json:"opponent_score"`
	Players       []PlayerStats `json:"players"`
	Filename      string        `json:"filename,omitempty"`
	Date          string        `json:"date"` // Puedes setearlo desde tu front/otra parte
}

// Redis global
var rdb *redis.Client
var ctx = context.Background()

func withCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Aquí permites que tu frontend (localhost:3000) acceda
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Si el método es OPTIONS, retornas 200 de inmediato
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// initRedis: Conecta a Redis
func initRedis() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("❌ No se pudo conectar a Redis: %v", err)
	}
	log.Println("✅ Conectado a Redis en", redisAddr)
}

// processDemoFile: Lee y procesa una demo con demoinfocs
func processDemoFile(filePath string) (*DemoParseResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("❌ Error al abrir la demo %s: %v", filePath, err)
	}
	defer file.Close()

	parser := demoinfocs.NewParser(file)
	defer parser.Close()

	// Mapa <SteamID64> -> stats del jugador
	playerStatsMap := make(map[uint64]*PlayerStats)

	// Para calcular ADR, necesitamos registrar el daño hecho
	// Podríamos llevar un "damageDone" por jugador, luego lo dividimos entre rondas
	damageDone := make(map[uint64]float64)

	var roundsPlayed int
	// Para almacenar kills totales por jugador (para luego calcular HS% = headshots / kills)
	headshotsCount := make(map[uint64]int)

	// Registramos los eventos
	parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer != nil {
			sid := e.Killer.SteamID64
			// Inicializa si no existe
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &PlayerStats{Name: e.Killer.Name}
			}
			playerStatsMap[sid].Kills++
			if e.IsHeadshot {
				headshotsCount[sid]++
			}
		}
		if e.Victim != nil {
			sid := e.Victim.SteamID64
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &PlayerStats{Name: e.Victim.Name}
			}
			playerStatsMap[sid].Deaths++
		}
		if e.Assister != nil {
			sid := e.Assister.SteamID64
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &PlayerStats{Name: e.Assister.Name}
			}
			playerStatsMap[sid].Assists++
		}
	})

	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker != nil && e.Attacker != e.Player {
			sid := e.Attacker.SteamID64
			damageDone[sid] += float64(e.HealthDamage)
		}
	})

	parser.RegisterEventHandler(func(e events.FlashExplode) {
		if e.Thrower != nil {
			sid := e.Thrower.SteamID64
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &PlayerStats{Name: e.Thrower.Name}
			}
			// No hay un “flash assist” directo en demoinfocs,
			// pero si decides contarlo aquí, incrementa:
			playerStatsMap[sid].FlashAssists++
		}
	})

	parser.RegisterEventHandler(func(e events.RoundEnd) {
		roundsPlayed++
	})

	// Parseamos frames
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			more, err2 := parser.ParseNextFrame()
			if !more || err2 != nil {
				break
			}
		}
	}()
	wg.Wait()

	// Sacamos la info del header
	header := parser.Header()
	mapName := header.MapName
	durationSeconds := header.PlaybackTime // segundos

	// Marcador final (asumiendo T es “nuestro” team)
	gameState := parser.GameState()
	tScore := gameState.TeamTerrorists().Score()
	ctScore := gameState.TeamCounterTerrorists().Score()

	// Decide a cuál llamas “TeamScore” y “OpponentScore”.
	teamScore := tScore
	opponentScore := ctScore
	result := "defeat"
	if teamScore > opponentScore {
		result = "victory"
	}

	// Calculamos ADR, KD ratio, HS% para cada jugador
	players := make([]PlayerStats, 0, len(playerStatsMap))
	for sid, pStats := range playerStatsMap {
		// ADR = damageDone / roundsPlayed
		if roundsPlayed > 0 {
			pStats.ADR = damageDone[sid] / float64(roundsPlayed)
		}
		// KDRatio
		if pStats.Deaths > 0 {
			pStats.KDRatio = float64(pStats.Kills) / float64(pStats.Deaths)
		} else {
			pStats.KDRatio = float64(pStats.Kills)
		}
		// HS%
		if pStats.Kills > 0 {
			pStats.HSPercentage = float64(headshotsCount[sid]) / float64(pStats.Kills) * 100
		}
		players = append(players, *pStats)
	}

	// Formateamos duración en mm:ss
	mins := int(durationSeconds / 60)
	secs := int(durationSeconds) % 60
	duration := fmt.Sprintf("%d:%02d", mins, secs)

	// Construimos el resultado final
	parseResult := &DemoParseResult{
		MatchID:       "DemoID-123", // genera un ID real si lo deseas
		MapName:       mapName,
		Duration:      duration,
		Result:        result,
		TeamScore:     teamScore,
		OpponentScore: opponentScore,
		Players:       players,
		// Date, Filename: se completan después
	}
	return parseResult, nil
}

// handleProcessDownloadedDemo: POST /process-downloaded-demo
// Recibe steam_id y filename, procesa la demo y la guarda en Redis
func handleProcessDownloadedDemo(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "No se encontró el archivo .dem en data/demos", http.StatusNotFound)
		return
	}

	parseResult, err := processDemoFile(demoPath)
	if err != nil {
		log.Printf("❌ Error al procesar la demo: %v", err)
		http.Error(w, "Error interno al procesar la demo", http.StatusInternalServerError)
		return
	}
	// Añadimos algunos campos manualmente
	parseResult.Filename = payload.Filename
	parseResult.Date = "2025-01-26" // Ejemplo: Podrías setear la fecha real aquí

	// Guardar en Redis
	// processed_demos:SteamID -> lista de JSON
	data, _ := json.Marshal(parseResult)
	rdb.RPush(ctx, fmt.Sprintf("processed_demos:%s", payload.SteamID), string(data))

	response := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Demo %s procesada y almacenada en Redis", payload.Filename),
		"data":    parseResult,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetProcessedDemos: GET /get-processed-demos
// Devuelve la lista de demos procesadas para un SteamID (o para todos, según tu uso)
func handleGetProcessedDemos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Imaginemos que recibes un query param ?steam_id=xxx
	// o si no hay, devuelves un error, o devuelves de todos
	steamID := r.URL.Query().Get("steam_id")
	if steamID == "" {
		http.Error(w, "Falta el parámetro steam_id", http.StatusBadRequest)
		return
	}

	// Leemos toda la lista
	entries, err := rdb.LRange(ctx, fmt.Sprintf("processed_demos:%s", steamID), 0, -1).Result()
	if err != nil {
		http.Error(w, "Error al leer Redis", http.StatusInternalServerError)
		return
	}

	var demos []DemoParseResult
	for _, entry := range entries {
		var demo DemoParseResult
		if err := json.Unmarshal([]byte(entry), &demo); err == nil {
			demos = append(demos, demo)
		}
	}

	// Respondemos
	response := map[string]interface{}{
		"status": "success",
		"demos":  demos,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleProcessDemo (opcional, para subir .dem manualmente via form-data)
func handleProcessDemo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "❌ Error al leer el archivo demo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFilePath := "./temp_demo.dem"
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		http.Error(w, "❌ Error al crear archivo temporal", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFilePath)

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "❌ Error al copiar archivo temporal", http.StatusInternalServerError)
		return
	}

	if !strings.HasSuffix(tempFilePath, ".dem") {
		http.Error(w, "❌ Archivo no válido. Debe ser un archivo .dem", http.StatusBadRequest)
		return
	}

	stats, err := processDemoFile(tempFilePath)
	if err != nil {
		log.Printf("❌ Error al procesar la demo: %v", err)
		http.Error(w, "❌ Error interno al procesar la demo", http.StatusInternalServerError)
		return
	}

	response := struct {
		Status  string           `json:"status"`
		Message string           `json:"message"`
		Data    *DemoParseResult `json:"data"`
	}{
		Status:  "success",
		Message: "Demo procesada exitosamente",
		Data:    stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// healthCheck: ping simple
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Servidor Go funcionando correctamente"))
}

func main() {
	initRedis()
	port := os.Getenv("GO_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/process-demo", handleProcessDemo)
	mux.HandleFunc("/process-downloaded-demo", handleProcessDownloadedDemo)
	mux.HandleFunc("/get-processed-demos", handleGetProcessedDemos)
	mux.HandleFunc("/health", healthCheck)

	// Envuelves tu mux con el middleware CORS
	handlerWithCors := withCors(mux)

	fmt.Printf("✅ Servidor Go en puerto %s...\n", port)
	http.ListenAndServe(":"+port, handlerWithCors)
}
