package demo

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

// generaIDUnico crea un ID pseudo-aleatorio para identificar la demo.
func generaIDUnico(filename, date string) string {
	hasher := sha1.New()
	hasher.Write([]byte(filename + date))
	return "match_" + hex.EncodeToString(hasher.Sum(nil))[:10] // Tomamos solo los primeros 10 caracteres
}

// getSteamData consulta la API de Steam para obtener el avatar y el rango de CS2 Premiere.
// Devuelve (avatarURL, rank, error).
func getSteamData(steamID string) (string, string, error) {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		return "", "", fmt.Errorf("STEAM_API_KEY no configurada")
	}

	// Consulta para el avatar.
	urlAvatar := fmt.Sprintf("http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%s", apiKey, steamID)
	resp, err := http.Get(urlAvatar)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	log.Printf("Respuesta avatar para %s: %s", steamID, string(body))
	var avatarResp struct {
		Response struct {
			Players []struct {
				AvatarFull string `json:"avatarfull"`
			} `json:"players"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &avatarResp); err != nil {
		return "", "", err
	}
	avatarURL := ""
	if len(avatarResp.Response.Players) > 0 {
		avatarURL = avatarResp.Response.Players[0].AvatarFull
	}

	// Consulta para el rango de CS2 Premiere.
	// NOTA: El endpoint utilizado ya no existe; por ello se manejará el error y se asignará "N/A"
	urlRank := fmt.Sprintf("https://api.steampowered.com/ICSGOPlayers_730/GetGamePersonalData/v1/?key=%s&steamid=%s", apiKey, steamID)
	resp2, err := http.Get(urlRank)
	if err != nil {
		// En caso de error, devolvemos el avatar y "N/A" para el rango.
		return avatarURL, "N/A", nil
	}
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return avatarURL, "N/A", nil
	}
	log.Printf("Respuesta rango para %s: %s", steamID, string(body2))
	var rankResp struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(body2, &rankResp); err != nil {
		// Si falla el unmarshalling, asignamos "N/A"
		return avatarURL, "N/A", nil
	}
	rankVal := ""
	if r, ok := rankResp.Result["rank_type"]; ok {
		if s, ok := r.(string); ok {
			rankVal = s
		}
	}
	if rankVal == "" {
		rankVal = "N/A"
	}

	return avatarURL, rankVal, nil
}

// ProcessDemoFile procesa la demo y devuelve el resultado parseado.
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

	// Convertimos el userSteamID a uint64.
	userID64, _ := strconv.ParseUint(userSteamID, 10, 64)
	var userTeam common.Team

	// Eventos de kills.
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

	// Eventos de daño (para ADR).
	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker != nil && e.Attacker != e.Player {
			sid := e.Attacker.SteamID64
			damageDone[sid] += float64(e.HealthDamage)
		}
	})

	// Evento Flash.
	parser.RegisterEventHandler(func(e events.FlashExplode) {
		if e.Thrower != nil {
			sid := e.Thrower.SteamID64
			if playerStatsMap[sid] == nil {
				playerStatsMap[sid] = &models.PlayerStats{Name: e.Thrower.Name}
			}
			playerStatsMap[sid].FlashAssists++
		}
	})

	// Incrementamos roundsPlayed en cada RoundEnd.
	parser.RegisterEventHandler(func(e events.RoundEnd) {
		roundsPlayed++
	})

	// (CAMBIO) Nueva variable para llevar la cuenta del tiempo de la demo
	var lastTime time.Duration

	// (CAMBIO) Registramos un evento para cada frame,
	//          e iremos guardando el tiempo actual de la demo
	parser.RegisterEventHandler(func(e events.FrameDone) {
		lastTime = parser.CurrentTime()
	})

	// Parseamos el demo.
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

	// Información del header.
	header := parser.Header()
	fmt.Printf("Header completo: %+v\n", header)

	mapName := header.MapName

	// (CAMBIO) En lugar de header.PlaybackTime, tomamos lastTime
	matchDurationSeconds := int(lastTime.Seconds())

	// (CAMBIO) Eliminamos el bloque que intentaba calcular con PlaybackTime y FrameRate.
	if matchDurationSeconds == 0 && header.FrameRate() > 0 {
		matchDurationSeconds = int(float64(header.PlaybackTicks) / header.FrameRate())
	}

	if matchDurationSeconds == 0 {
		log.Println("⚠ Advertencia: La duración de la partida es 0 segundos. Revisa la demo.")
	}

	matchDurationFormatted := fmt.Sprintf("%02d:%02d", matchDurationSeconds/60, matchDurationSeconds%60)

	// Obtener puntajes.
	gameState := parser.GameState()
	tScore := gameState.TeamTerrorists().Score()
	ctScore := gameState.TeamCounterTerrorists().Score()

	// Asignar "team" a cada jugador.
	for _, pl := range parser.GameState().Participants().All() {
		sid := pl.SteamID64
		if sid == 0 {
			continue
		}
		if playerStatsMap[sid] == nil {
			playerStatsMap[sid] = &models.PlayerStats{}
		}
		playerStatsMap[sid].SteamID = fmt.Sprintf("%d", sid)
		playerStatsMap[sid].Name = pl.Name

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

		if sid == userID64 {
			userTeam = pl.Team
		}
	}

	// Determinar puntajes del equipo.
	var teamScore, opponentScore int
	if userTeam == common.TeamTerrorists {
		teamScore = tScore
		opponentScore = ctScore
	} else if userTeam == common.TeamCounterTerrorists {
		teamScore = ctScore
		opponentScore = tScore
	} else {
		teamScore = tScore
		opponentScore = ctScore
	}

	result := "defeat"
	if teamScore > opponentScore {
		result = "victory"
	}

	// Calcular KDRatio, ADR y HS%
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

	// ** Enriquecer cada jugador con Avatar y Rango de CS2 Premiere **
	for sid, ps := range playerStatsMap {
		steamIDStr := fmt.Sprintf("%d", sid)
		avatarURL, rankVal, err := getSteamData(steamIDStr)
		if err != nil {
			log.Printf("Error obteniendo datos de Steam para %s: %v", steamIDStr, err)
			// Asignamos "N/A" en caso de error.
			ps.Rank = "N/A"
		} else {
			ps.Avatar = avatarURL
			ps.Rank = rankVal
		}
	}

	// Usamos header.FrameTime() para la fecha (ajusta según convenga).
	var matchDate string
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error al obtener metadata del archivo: %v", err)
		// En caso de error, se puede asignar la fecha actual o manejar el error según convenga.
		matchDate = time.Now().Format("2006-01-02 15:04:05")
	} else {
		matchDate = fileInfo.ModTime().Format("2006-01-02 15:04:05")
	}

	parseResult := &models.DemoParseResult{
		MatchID:       generaIDUnico(filepath.Base(filePath), matchDate),
		MapName:       mapName,
		Duration:      matchDurationFormatted,
		Result:        result,
		TeamScore:     teamScore,
		OpponentScore: opponentScore,
		Players:       make([]models.PlayerStats, 0, len(playerStatsMap)),
		Date:          matchDate,
		Filename:      filepath.Base(filePath),
	}

	var sortedPlayers []models.PlayerStats
	for _, ps := range playerStatsMap {
		sortedPlayers = append(sortedPlayers, *ps)
	}

	sort.Slice(sortedPlayers, func(i, j int) bool {
		if sortedPlayers[i].KDRatio != sortedPlayers[j].KDRatio {
			return sortedPlayers[i].KDRatio > sortedPlayers[j].KDRatio
		}
		if sortedPlayers[i].ADR != sortedPlayers[j].ADR {
			return sortedPlayers[i].ADR > sortedPlayers[j].ADR
		}
		return sortedPlayers[i].Kills > sortedPlayers[j].Kills
	})

	for i := range sortedPlayers {
		sortedPlayers[i].Position = i + 1
		log.Printf("Asignando posición: %d a jugador %s (SteamID: %s)", sortedPlayers[i].Position, sortedPlayers[i].Name, sortedPlayers[i].SteamID)
	}
	for _, ps := range sortedPlayers {
		parseResult.Players = append(parseResult.Players, ps)
	}

	return parseResult, nil
}
