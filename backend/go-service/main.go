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

/*
AdvancedStats:
Estructura para almacenar estadísticas avanzadas de la demo,
como kills, deaths, headshots, etc.
*/
type AdvancedStats struct {
	MatchID      string  `json:"match_id"`
	Kills        int     `json:"kills"`
	Deaths       int     `json:"deaths"`
	Assists      int     `json:"assists"`
	Headshots    int     `json:"headshots"`
	ADR          float64 `json:"adr"`
	HSPercentage float64 `json:"hs_percentage"`
	KDRatio      float64 `json:"kd_ratio"`
	FlashAssists int     `json:"flash_assists"`
	Filename     string  `json:"filename,omitempty"`
}

var rdb *redis.Client
var ctx = context.Background()

// initRedis se encarga de conectar con Redis.
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

/*
processDemoFile:
Procesa un archivo .dem con demoinfocs-golang.
*/
func processDemoFile(filePath string) (*AdvancedStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("❌ Error al abrir el archivo %s: %v", filePath, err)
	}
	defer file.Close()

	parser := demoinfocs.NewParser(file)
	defer parser.Close()

	var kills, deaths, assists, headshots, flashAssists int
	var totalDamage float64
	var roundsPlayed int

	var wg sync.WaitGroup
	eventsChan := make(chan interface{}, 100)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range eventsChan {
			switch ev := e.(type) {
			case events.Kill:
				kills++
				if ev.Victim != nil {
					deaths++
				}
				if ev.IsHeadshot {
					headshots++
				}
				if ev.Assister != nil {
					assists++
				}
			case events.PlayerHurt:
				if ev.Attacker != nil && ev.Player != nil && ev.Attacker != ev.Player {
					totalDamage += float64(ev.HealthDamage)
				}
			case events.FlashExplode:
				if ev.Thrower != nil {
					flashAssists++
				}
			}
		}
	}()

	parser.RegisterEventHandler(func(e events.Kill) {
		eventsChan <- e
	})
	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		eventsChan <- e
	})
	parser.RegisterEventHandler(func(e events.FlashExplode) {
		eventsChan <- e
	})
	parser.RegisterEventHandler(func(e events.RoundEnd) {
		roundsPlayed++
	})

	for {
		more, err := parser.ParseNextFrame()
		if err != nil || !more {
			break
		}
	}

	close(eventsChan)
	wg.Wait()

	hsPercentage := 0.0
	if kills > 0 {
		hsPercentage = (float64(headshots) / float64(kills)) * 100
	}

	kdRatio := 0.0
	if deaths > 0 {
		kdRatio = float64(kills) / float64(deaths)
	}

	adr := 0.0
	if roundsPlayed > 0 {
		adr = totalDamage / float64(roundsPlayed)
	}

	stats := &AdvancedStats{
		MatchID:      "DemoID",
		Kills:        kills,
		Deaths:       deaths,
		Assists:      assists,
		Headshots:    headshots,
		ADR:          adr,
		HSPercentage: hsPercentage,
		KDRatio:      kdRatio,
		FlashAssists: flashAssists,
	}

	return stats, nil
}

/*
handleProcessDownloadedDemo:
POST /process-downloaded-demo
Recibe steam_id y filename (p.ej. "match_xxx.dem"), parsea la demo y
almacena stats en Redis -> processed_demos:<steam_id>.
*/
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

	// Aseguramos la ruta relativa a data/demos
	// Ajusta si el binario Go corre en otra ubicación
	demoPath := filepath.Join("..", "data", "demos", payload.Filename)
	// Si tu main.go reside en /go-service y data/demos está en /backend/data/demos,
	// puede que necesites "../../" en vez de "..". Ajusta según tu caso real.

	if _, err := os.Stat(demoPath); os.IsNotExist(err) {
		http.Error(w, "No se encontró el archivo .dem en data/demos", http.StatusNotFound)
		return
	}

	stats, err := processDemoFile(demoPath)
	if err != nil {
		log.Printf("❌ Error al procesar la demo: %v", err)
		http.Error(w, "Error interno al procesar la demo", http.StatusInternalServerError)
		return
	}

	stats.Filename = payload.Filename

	statsJSON, _ := json.Marshal(stats)
	rdb.RPush(ctx, fmt.Sprintf("processed_demos:%s", payload.SteamID), string(statsJSON))

	response := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Demo %s procesada y almacenada en Redis", payload.Filename),
		"data":    stats,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleProcessDemo: para subir demos manualmente (opcional)
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
		Status  string         `json:"status"`
		Message string         `json:"message"`
		Data    *AdvancedStats `json:"data"`
	}{
		Status:  "success",
		Message: "Demo procesada exitosamente",
		Data:    stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// healthCheck: comprueba si está activo
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Servidor Go funcionando correctamente"))
}

// main: inicia el servidor HTTP en :8080
func main() {
	initRedis()

	port := os.Getenv("GO_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/process-demo", handleProcessDemo)
	http.HandleFunc("/process-downloaded-demo", handleProcessDownloadedDemo)
	http.HandleFunc("/health", healthCheck)

	fmt.Printf("✅ Servidor Go ejecutándose en el puerto %s...\n", port)
	http.ListenAndServe(":"+port, nil)
}
