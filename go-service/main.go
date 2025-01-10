package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// Middleware para habilitar CORS
func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var statsSyncMap sync.Map

type Stats struct {
	Kills       int                    `json:"kills"`
	Assists     int                    `json:"assists"`
	Deaths      int                    `json:"deaths"`
	Headshots   int                    `json:"headshots"`
	Rounds      int                    `json:"rounds_played"`
	Damage      int                    `json:"damage"`
	ADR         float64                `json:"adr"`
	HSPercent   float64                `json:"hs_percent"`
	KD          float64                `json:"kd"`
	ImpactScore float64                `json:"impact_score"`
	Players     map[string]PlayerStats `json:"players"`
}

type PlayerStats struct {
	Kills     int `json:"kills"`
	Assists   int `json:"assists"`
	Deaths    int `json:"deaths"`
	Headshots int `json:"headshots"`
	Damage    int `json:"damage"`
}

func processDemo(filePath string) (*Stats, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	parser := demoinfocs.NewParser(f)
	defer parser.Close()

	stats := &Stats{Players: make(map[string]PlayerStats)}
	var totalDamage int

	parser.RegisterEventHandler(func(e events.Kill) {
		stats.Kills++
		if e.Assister != nil {
			stats.Assists++
		}
		if e.Victim != nil {
			stats.Deaths++
		}
		if e.IsHeadshot {
			stats.Headshots++
		}

		// Actualizar estadísticas por jugador
		if e.Killer != nil {
			pStats := stats.Players[e.Killer.Name]
			pStats.Kills++
			if e.IsHeadshot {
				pStats.Headshots++
			}
			stats.Players[e.Killer.Name] = pStats
		}

		if e.Assister != nil {
			pStats := stats.Players[e.Assister.Name]
			pStats.Assists++
			stats.Players[e.Assister.Name] = pStats
		}

		if e.Victim != nil {
			pStats := stats.Players[e.Victim.Name]
			pStats.Deaths++
			stats.Players[e.Victim.Name] = pStats
		}
	})

	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		totalDamage += e.HealthDamage
		if e.Player != nil {
			pStats := stats.Players[e.Player.Name]
			pStats.Damage += e.HealthDamage
			stats.Players[e.Player.Name] = pStats
		}
	})

	err = parser.ParseToEnd()
	if err != nil {
		return nil, err
	}

	// Métricas avanzadas
	stats.Rounds = parser.GameState().TotalRoundsPlayed()
	stats.Damage = totalDamage
	stats.ADR = float64(totalDamage) / float64(stats.Rounds)
	stats.HSPercent = (float64(stats.Headshots) / float64(stats.Kills)) * 100
	stats.KD = float64(stats.Kills) / float64(stats.Deaths)
	stats.ImpactScore = float64(stats.Kills)*1.5 + float64(stats.Assists)*1.2 + float64(stats.Headshots)*2.0

	return stats, nil
}

func handleDemoProcessing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error al leer archivo del formulario", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", header.Filename)
	if err != nil {
		http.Error(w, "Error al crear archivo temporal", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = tempFile.ReadFrom(file)
	if err != nil {
		http.Error(w, "Error al escribir archivo temporal", http.StatusInternalServerError)
		return
	}

	stats, err := processDemo(tempFile.Name())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al procesar demo: %v", err), http.StatusInternalServerError)
		return
	}

	statsJSON, _ := json.Marshal(stats)
	w.Header().Set("Content-Type", "application/json")
	w.Write(statsJSON)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/process-demo", handleDemoProcessing)

	fmt.Println("Servidor Go escuchando en el puerto 8080...")
	if err := http.ListenAndServe(":8080", enableCors(mux)); err != nil {
		fmt.Printf("Error al iniciar servidor: %v\n", err)
	}
}
