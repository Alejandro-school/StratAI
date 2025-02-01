package inspect

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// Estructuras para JSON
type DemoData struct {
	Map         string        `json:"map"`
	TotalRounds int           `json:"totalRounds"`
	Players     []PlayerStats `json:"players"`
	Events      []Event       `json:"events"`
}

type PlayerStats struct {
	Name      string `json:"name"`
	SteamID   string `json:"steamID"`
	Kills     int    `json:"kills"`
	Deaths    int    `json:"deaths"`
	Assists   int    `json:"assists"`
	Headshots int    `json:"headshots"`
}

type Event struct {
	Type   string `json:"type"`
	Time   int    `json:"time"`
	Killer string `json:"killer,omitempty"`
	Victim string `json:"victim,omitempty"`
	Weapon string `json:"weapon,omitempty"`
	Winner string `json:"winner,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// InspectDemo procesa una demo y exporta un JSON con sus datos.
func InspectDemo() {
	demoFile := "../data/demos/match_e7nbjoSRxWSLfXz6oLfvk3AkE.dem"

	f, err := os.Open(demoFile)
	if err != nil {
		log.Fatalf("Error abriendo demo: %v", err)
	}
	defer f.Close()

	p := dem.NewParser(f)

	var demoData DemoData
	demoData.Players = []PlayerStats{}
	demoData.Events = []Event{}

	// Extraer información general de la partida
	p.RegisterEventHandler(func(e events.MatchStart) {
		demoData.Map = p.Header().MapName
		fmt.Println("📌 Mapa detectado:", demoData.Map)
	})

	// Extraer estadísticas de jugadores
	p.RegisterEventHandler(func(e events.Kill) {
		if e.Killer != nil && e.Victim != nil {
			demoData.Events = append(demoData.Events, Event{
				Type:   "kill",
				Time:   int(p.CurrentTime().Seconds()),
				Killer: e.Killer.Name,
				Victim: e.Victim.Name,
				Weapon: e.Weapon.String(),
			})
		}
	})

	p.RegisterEventHandler(func(e events.RoundEnd) {
		winner := "Unknown"
		if e.Winner == common.TeamTerrorists {
			winner = "Terrorists"
		} else if e.Winner == common.TeamCounterTerrorists {
			winner = "Counter-Terrorists"
		}
		demoData.Events = append(demoData.Events, Event{
			Type:   "round_end",
			Winner: winner,
			Reason: fmt.Sprintf("%s", e.Reason),
		})
		demoData.TotalRounds++
	})

	p.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter != nil {
			demoData.Events = append(demoData.Events, Event{
				Type:   "weapon_fire",
				Time:   int(p.CurrentTime().Seconds()),
				Killer: e.Shooter.Name,
				Weapon: e.Weapon.String(),
			})
		}
	})

	// Capturar estadísticas finales de jugadores
	p.RegisterEventHandler(func(e events.RoundEnd) {
		for _, player := range p.GameState().Participants().All() {
			playerData := PlayerStats{
				Name:    player.Name,
				SteamID: strconv.FormatUint(player.SteamID64, 10),
				Kills:   player.Kills(),
				Deaths:  player.Deaths(),
				Assists: player.Assists(),
			}
			demoData.Players = append(demoData.Players, playerData)
		}
	})

	// Procesar la demo
	err = p.ParseToEnd()
	if err != nil {
		log.Fatalf("❌ Error al procesar demo: %v", err)
	}

	// Exportar JSON
	jsonData, err := json.MarshalIndent(demoData, "", "  ")
	if err != nil {
		log.Fatalf("❌ Error al generar JSON: %v", err)
	}

	// Guardar en archivo JSON
	jsonFile := "output_demo.json"
	err = os.WriteFile(jsonFile, jsonData, 0644)
	if err != nil {
		log.Fatalf("❌ Error guardando JSON: %v", err)
	}

	fmt.Println("✅ JSON generado:", jsonFile)
}
