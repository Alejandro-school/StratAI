package demo

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"cs2-demo-service/models"
)

// exportMatchSummary crea el fichero match_summary.json con la info principal (sin Movement).
func exportMatchSummary(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "match_summary.json")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("no se pudo crear match_summary.json: %w", err)
	}
	defer file.Close()

	// Copiamos la lista de jugadores y eliminamos el campo Movement.
	summaryPlayers := make([]models.PlayerStats, len(result.Players))
	copy(summaryPlayers, result.Players)
	for i := range summaryPlayers {
		summaryPlayers[i].Movement = nil
	}

	summary := struct {
		MatchID       string               `json:"match_id"`
		MapName       string               `json:"map_name"`
		Duration      string               `json:"match_duration"`
		Result        string               `json:"result"`
		TeamScore     int                  `json:"team_score"`
		OpponentScore int                  `json:"opponent_score"`
		Filename      string               `json:"filename"`
		Date          string               `json:"date"`
		Players       []models.PlayerStats `json:"players"`
	}{
		MatchID:       result.MatchID,
		MapName:       result.MapName,
		Duration:      result.Duration,
		Result:        result.Result,
		TeamScore:     result.TeamScore,
		OpponentScore: result.OpponentScore,
		Filename:      result.Filename,
		Date:          result.Date,
		Players:       summaryPlayers,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}

// exportEconomy genera economy.json con la historia de economía ordenada por ronda.
func exportEconomy(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "economy.json")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("no se pudo crear economy.json: %w", err)
	}
	defer file.Close()

	// Ordenamos EconomyHistory por RoundNumber
	economyHistory := result.EconomyHistory
	sort.Slice(economyHistory, func(i, j int) bool {
		return economyHistory[i].RoundNumber < economyHistory[j].RoundNumber
	})

	payload := struct {
		MatchID        string                      `json:"match_id"`
		EconomyHistory []*models.RoundEconomyStats `json:"economy_history"`
	}{
		MatchID:        result.MatchID,
		EconomyHistory: economyHistory,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// exportEvents genera events.json agrupando los EventLogs por ronda en orden ascendente.
func exportEvents(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "events.json")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("no se pudo crear events.json: %w", err)
	}
	defer file.Close()

	// Agrupar eventos por ronda en un mapa.
	eventsByRound := make(map[int][]models.EventLog)
	for _, ev := range result.EventLogs {
		eventsByRound[ev.Round] = append(eventsByRound[ev.Round], ev)
	}

	// Estructura auxiliar para ordenar.
	type RoundEvents struct {
		Round  int               `json:"round"`
		Events []models.EventLog `json:"events"`
	}

	// Extraer y ordenar las rondas.
	var rounds []int
	for round := range eventsByRound {
		rounds = append(rounds, round)
	}
	sort.Ints(rounds) // Orden ascendente: 1, 2, 3, ...

	var sortedEvents []RoundEvents
	for _, r := range rounds {
		sortedEvents = append(sortedEvents, RoundEvents{
			Round:  r,
			Events: eventsByRound[r],
		})
	}

	payload := struct {
		MatchID       string        `json:"match_id"`
		EventsByRound []RoundEvents `json:"events_by_round"`
	}{
		MatchID:       result.MatchID,
		EventsByRound: sortedEvents,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// exportAllMovementsInOneFile genera movement.json con todos los registros de movimiento ordenados por tick.
func exportAllMovementsInOneFile(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "movement.json")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("no se pudo crear movement.json: %w", err)
	}
	defer file.Close()

	// Agregamos todos los movimientos en un slice.
	var allMovements []struct {
		SteamID    string  `json:"steamID"`
		PlayerName string  `json:"player_name"`
		Tick       int     `json:"tick"`
		X          float64 `json:"x_position"`
		Y          float64 `json:"y_position"`
		Z          float64 `json:"z_position"`
		Speed      float64 `json:"speed"`
		IsDucking  bool    `json:"is_ducking"`
	}
	for _, player := range result.Players {
		for _, m := range player.Movement {
			allMovements = append(allMovements, struct {
				SteamID    string  `json:"steamID"`
				PlayerName string  `json:"player_name"`
				Tick       int     `json:"tick"`
				X          float64 `json:"x_position"`
				Y          float64 `json:"y_position"`
				Z          float64 `json:"z_position"`
				Speed      float64 `json:"speed"`
				IsDucking  bool    `json:"is_ducking"`
			}{
				SteamID:    player.SteamID,
				PlayerName: player.Name,
				Tick:       m.Tick,
				X:          m.X,
				Y:          m.Y,
				Z:          m.Z,
				Speed:      m.Speed,
				IsDucking:  m.IsDucking,
			})
		}
	}
	// Ordenamos los movimientos por Tick (ascendente).
	sort.Slice(allMovements, func(i, j int) bool {
		return allMovements[i].Tick < allMovements[j].Tick
	})

	payload := struct {
		MatchID   string      `json:"match_id"`
		Movements interface{} `json:"movements"`
	}{
		MatchID:   result.MatchID,
		Movements: allMovements,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// • Usa result.Grenades  (slice ya ordenable)  →  NO lee mapas internos.
// • Devuelve un array ordenado ascendente por ronda: 1, 2, 3, …
func exportTrajectories(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "trajectories.json")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("no se pudo crear trajectories.json: %w", err)
	}
	defer file.Close()

	// Copiamos para no mutar la referencia original
	nades := make([]models.GrenadeMetadata, len(result.Grenades))
	copy(nades, result.Grenades)

	// ➊  Ordenar por ronda
	sort.Slice(nades, func(i, j int) bool {
		if nades[i].Round != nades[j].Round {
			return nades[i].Round < nades[j].Round
		}
		if nades[i].ThrowTick != nades[j].ThrowTick { // ← nuevo criterio
			return nades[i].ThrowTick < nades[j].ThrowTick
		}
		return nades[i].ProjectileID < nades[j].ProjectileID
	})

	// ➋  Construir payload
	payload := struct {
		MatchID      string                   `json:"match_id"`
		Trajectories []models.GrenadeMetadata `json:"trajectories"`
	}{
		MatchID:      result.MatchID,
		Trajectories: nades,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

/* ──────────────────────────────   EXPORT ALL  ───────────────────────────── */

func ExportSegmentedResultsOneMovementFile(result *models.DemoParseResult, outputDir string) error {
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("no se pudo crear el directorio de salida: %w", err)
	}

	if err := exportMatchSummary(result, outputDir); err != nil {
		return err
	}
	if err := exportEconomy(result, outputDir); err != nil {
		return err
	}
	if err := exportEvents(result, outputDir); err != nil {
		return err
	}
	if err := exportAllMovementsInOneFile(result, outputDir); err != nil {
		return err
	}
	if err := exportTrajectories(result, outputDir); err != nil {
		return err
	}

	log.Println("[Export] Exportación segmentada completada con éxito en", outputDir)
	return nil
}
