package demo

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"cs2-demo-service/models"

	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// generaIDUnico crea un ID pseudo-aleatorio para identificar la demo
func generaIDUnico(filename, date string) string {
	hasher := sha1.New()
	hasher.Write([]byte(filename + date))
	return "match_" + hex.EncodeToString(hasher.Sum(nil))[:10] // Tomamos solo los primeros 10 caracteres
}

// ProcessDemoFile procesa la demo. Recibe el steamID para saber en qué equipo está el usuario.
// (Es la misma función processDemoFile que estaba en main.go, con los mismos eventos y lógica)
func ProcessDemoFile(filePath, userSteamID string) (*models.DemoParseResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Error al abrir la demo %s: %v", filePath, err)
	}
	defer file.Close()

	parser := dem.NewParser(file)
	defer parser.Close()

	// Mapa <SteamID64> -> *PlayerStats
	playerStatsMap := make(map[uint64]*models.PlayerStats)
	damageDone := make(map[uint64]float64)
	headshotsCount := make(map[uint64]int)
	var roundsPlayed int

	// Convertimos el userSteamID a uint64
	userID64, _ := strconv.ParseUint(userSteamID, 10, 64)
	var userTeam common.Team // guardará TeamTerrorists o TeamCounterTerrorists

	// Eventos de kills
	parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer != nil {
			sid := e.Killer.SteamID64
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &models.PlayerStats{Name: e.Killer.Name}
			}
			playerStatsMap[sid].Kills++
			if e.IsHeadshot {
				headshotsCount[sid]++
			}
		}
		if e.Victim != nil {
			sid := e.Victim.SteamID64
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &models.PlayerStats{Name: e.Victim.Name}
			}
			playerStatsMap[sid].Deaths++
		}
		if e.Assister != nil {
			sid := e.Assister.SteamID64
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &models.PlayerStats{Name: e.Assister.Name}
			}
			playerStatsMap[sid].Assists++
		}
	})

	// Eventos de daño (para ADR)
	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker != nil && e.Attacker != e.Player {
			sid := e.Attacker.SteamID64
			damageDone[sid] += float64(e.HealthDamage)
		}
	})

	// Flash (ejemplo)
	parser.RegisterEventHandler(func(e events.FlashExplode) {
		if e.Thrower != nil {
			sid := e.Thrower.SteamID64
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &models.PlayerStats{Name: e.Thrower.Name}
			}
			playerStatsMap[sid].FlashAssists++
		}
	})

	// Al acabar cada ronda, incrementamos roundsPlayed
	parser.RegisterEventHandler(func(e events.RoundEnd) {
		roundsPlayed++
	})

	// Parseamos el archivo
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			more, errParse := parser.ParseNextFrame()
			if !more || errParse != nil {
				break
			}
		}
	}()
	wg.Wait()

	// Info del header
	header := parser.Header()
	mapName := header.MapName
	matchDurationSeconds := int(header.PlaybackTime.Seconds())
	if matchDurationSeconds == 0 {
		log.Println("⚠ Advertencia: La duración de la partida es 0 segundos. Revisa la demo.")
	}
	matchDurationFormatted := fmt.Sprintf("%02d:%02d", matchDurationSeconds/60, matchDurationSeconds%60)

	// Equipo T y CT
	gameState := parser.GameState()
	tScore := gameState.TeamTerrorists().Score()
	ctScore := gameState.TeamCounterTerrorists().Score()

	// *** Asignamos el “team” a cada jugador. ***
	// Mapa <SteamID64> -> *PlayerStats
	for _, pl := range parser.GameState().Participants().All() {
		sid := pl.SteamID64
		if sid == 0 {
			continue
		}

		if playerStatsMap[sid] == nil {
			playerStatsMap[sid] = &models.PlayerStats{} // <-- Fíjate en 'models.PlayerStats'
		}

		// Asignamos SIEMPRE SteamID y Name (aunque ya se haya creado con kills)
		playerStatsMap[sid].SteamID = fmt.Sprintf("%d", sid)
		playerStatsMap[sid].Name = pl.Name

		// Convertimos a string "T" o "CT"
		var tStr string
		switch pl.Team {
		case common.TeamTerrorists:
			tStr = "T"
		case common.TeamCounterTerrorists:
			tStr = "CT"
		default:
			tStr = "UNK"
		}
		playerStatsMap[sid].Team = tStr

		// Si es nuestro user, guardamos su Team real
		if sid == userID64 {
			userTeam = pl.Team
		}
	}

	// Decidimos el “teamScore” y “opponentScore”
	var teamScore, opponentScore int
	if userTeam == common.TeamTerrorists {
		teamScore = tScore
		opponentScore = ctScore
	} else if userTeam == common.TeamCounterTerrorists {
		teamScore = ctScore
		opponentScore = tScore
	} else {
		// fallback: asumimos T
		teamScore = tScore
		opponentScore = ctScore
	}

	// victory / defeat
	result := "defeat"
	if teamScore > opponentScore {
		result = "victory"
	}

	// Calculamos KDRatio, ADR, HS%
	for sid, pStats := range playerStatsMap {
		if roundsPlayed > 0 {
			pStats.ADR = damageDone[sid] / float64(roundsPlayed)
		}
		if pStats.Deaths > 0 {
			pStats.KDRatio = float64(pStats.Kills) / float64(pStats.Deaths)
		} else {
			pStats.KDRatio = float64(pStats.Kills)
		}
		if pStats.Kills > 0 {
			pStats.HSPercentage = float64(headshotsCount[sid]) / float64(pStats.Kills) * 100
		}
	}

	// Tomamos la fecha de modificación del archivo
	matchDate := time.Unix(int64(header.PlaybackTime), 0).Format("2006-01-02 15:04:05")

	// Montamos la estructura final
	parseResult := &models.DemoParseResult{
		MatchID:       generaIDUnico(filepath.Base(filePath), matchDate),
		MapName:       mapName,
		Duration:      matchDurationFormatted,
		Result:        result,
		TeamScore:     teamScore,
		OpponentScore: opponentScore,
		Players:       make([]models.PlayerStats, 0, len(playerStatsMap)),
		Date:          matchDate, // ✅ Ahora usa la fecha correcta de la partida
		Filename:      filepath.Base(filePath),
	}

	// Ordenamos los jugadores por rendimiento
	var sortedPlayers []models.PlayerStats
	for _, ps := range playerStatsMap {
		sortedPlayers = append(sortedPlayers, *ps)
	}

	// Ordenamos por Kill/Death Ratio, luego por ADR, luego por Kills
	sort.Slice(sortedPlayers, func(i, j int) bool {
		if sortedPlayers[i].KDRatio != sortedPlayers[j].KDRatio {
			return sortedPlayers[i].KDRatio > sortedPlayers[j].KDRatio
		}
		if sortedPlayers[i].ADR != sortedPlayers[j].ADR {
			return sortedPlayers[i].ADR > sortedPlayers[j].ADR
		}
		return sortedPlayers[i].Kills > sortedPlayers[j].Kills
	})

	// Asignamos posiciones
	for i := range sortedPlayers {
		sortedPlayers[i].Position = i + 1
		log.Printf("Asignando posición: %d a jugador %s (SteamID: %s)", sortedPlayers[i].Position, sortedPlayers[i].Name, sortedPlayers[i].SteamID)

	}
	for _, ps := range playerStatsMap {
		log.Printf("Jugador: %s | SteamID: %s | Equipo: %s | Posición: %d", ps.Name, ps.SteamID, ps.Team, ps.Position)
		parseResult.Players = append(parseResult.Players, *ps)
	}

	return parseResult, nil
}
