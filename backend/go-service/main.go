package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

type AdvancedStats struct {
	MatchID      string  `json:"match_id"`
	Kills        int     `json:"kills"`
	Deaths       int     `json:"deaths"`
	Assists      int     `json:"assists"`
	Headshots    int     `json:"headshots"`
	ADR          float64 `json:"adr"`
	HSPercentage float64 `json:"hs_percentage"`
	KDRatio      float64 `json:"kd_ratio"`
}

func processDemoFile(filePath string) (*AdvancedStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("❌ Error al abrir el archivo: %v", err)
	}
	defer file.Close()

	parser := demoinfocs.NewParser(file)
	defer parser.Close()

	var kills, deaths, assists, headshots int
	var totalDamage float64
	var roundsPlayed int

	parser.RegisterEventHandler(func(e events.Kill) {
		kills++
		if e.IsHeadshot {
			headshots++
		}
		if e.Assister != nil {
			assists++
		}
	})

	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker != nil && e.Player != nil && e.Attacker != e.Player {
			totalDamage += float64(e.HealthDamage)
		}
	})

	parser.RegisterEventHandler(func(e events.RoundEnd) {
		roundsPlayed++
	})

	for {
		moreFrames, err := parser.ParseNextFrame()
		if err != nil {
			break
		}
		if !moreFrames {
			break
		}
	}

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
	}

	return stats, nil
}

func handleProcessDemo(w http.ResponseWriter, r *http.Request) {
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

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "❌ Error al copiar archivo temporal", http.StatusInternalServerError)
		return
	}

	stats, err := processDemoFile(tempFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("❌ Error al procesar la demo: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func main() {
	http.HandleFunc("/process-demo", handleProcessDemo)
	fmt.Println("✅ Servidor Go ejecutándose en el puerto 8080...")
	http.ListenAndServe(":8080", nil)
}
