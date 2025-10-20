package demo

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"cs2-demo-service/models"

	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
)

// ProcessDemoFile es la función principal
func ProcessDemoFile(filePath, userSteamID string, jsonData string) (*models.DemoParseResult, error) {
	log.Println("DEBUG: --> Entró a ProcessDemoFile")

	matchDuration, err := ExtractMatchDuration(jsonData)
	if err != nil {
		log.Printf("⚠️ error extrayendo match_duration: %v", err)
		matchDuration = "00:00"
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error al abrir la demo %s: %v", filePath, err)
	}
	defer file.Close()

	parser := dem.NewParser(file)
	defer parser.Close()

	// Creamos el contexto
	ctx := NewDemoContext()
	ctx.parser = parser

	// Registramos handlers
	registerRoundHandlers(ctx)
	registerBombHandlers(ctx)
	registerGrenadeHandlers(ctx)
	registerPlayerHandlers(ctx)
	registerMovementHandlers(ctx)

	// Parseamos
	if err := parser.ParseToEnd(); err != nil {
		log.Printf("Error parsing demo: %v", err)
	}

	// Compilamos resultados
	parseResult, err := compileResults(ctx, filePath, userSteamID, matchDuration)
	if err != nil {
		return nil, err
	}

	// Exportamos
	outputDir := filepath.Join("../data/exports", parseResult.MatchID)
	err = ExportSegmentedResultsOneMovementFile(parseResult, outputDir)
	if err != nil {
		log.Printf("Error exportando resultados segmentados: %v", err)
	}
	return parseResult, nil
}

// compileResults arma el DemoParseResult usando la info de ctx
func compileResults(ctx *DemoContext, filePath string, userSteamID string, matchDuration string) (*models.DemoParseResult, error) {
	header := ctx.parser.Header()
	mapName := header.MapName
	gs := ctx.parser.GameState()
	tScore := gs.TeamTerrorists().Score()
	ctScore := gs.TeamCounterTerrorists().Score()

	userTeam := determineUserTeam(ctx, userSteamID)

	roundsPlayed := ctx.RoundNumber
	for sid, ps := range ctx.PlayerStatsMap {
		if roundsPlayed > 0 {
			ps.ADR = ctx.DamageDone[sid] / float64(roundsPlayed)
		}
		if ps.Deaths > 0 {
			ps.KDRatio = float64(ps.Kills) / float64(ps.Deaths)
		} else {
			ps.KDRatio = float64(ps.Kills)
		}
		if ps.Kills > 0 {
			ps.HSPercentage = float64(ctx.HeadshotsCount[sid]) / float64(ps.Kills) * 100
		}
		if ps.ShotsFired > 0 {
			ps.Accuracy = (float64(ps.ShotsConnected) / float64(ps.ShotsFired)) * 100
		} else {
			ps.Accuracy = 0
		}
	}

	var sortedPlayers []models.PlayerStats
	for sid, ps := range ctx.PlayerStatsMap {
		ps.SteamID = fmt.Sprintf("%d", sid)

		// Asignar el equipo del jugador
		for _, pl := range gs.Participants().All() {
			if pl.SteamID64 == sid {
				ps.Team = teamString(pl)
				break
			}
		}

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
	}

	for i := range sortedPlayers {
		steamIDStr := sortedPlayers[i].SteamID
		avatarURL, err := getSteamAvatar(steamIDStr)
		if err != nil {
			log.Printf("error retrieving steam avatar for %s: %v", steamIDStr, err)
		} else {
			sortedPlayers[i].Avatar = avatarURL
		}
	}

	matchID, err := ComputeFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("error generando matchID: %v", err)
	}

	teamScore, opponentScore := tScore, ctScore
	if userTeam == 3 {
		teamScore, opponentScore = ctScore, tScore
	}

	result := "defeat"
	if teamScore > opponentScore {
		result = "victory"
	}

	// Obtener la fecha del archivo .dem combinada con la hora real del partido
	matchDate := time.Now().Format("2006-01-02 15:04:05")
	if fileInfo, err := os.Stat(filePath); err == nil {
		fileDate := fileInfo.ModTime()
		
		// Calcular la hora real del partido restando la duración
		// Parsear la duración en formato MM:SS
		var mins, secs int
		if matchDuration != "00:00" && matchDuration != "" {
			fmt.Sscanf(matchDuration, "%d:%d", &mins, &secs)
			durationInSeconds := mins*60 + secs
			
			// Restar la duración del partido a la fecha del archivo
			actualMatchTime := fileDate.Add(-time.Duration(durationInSeconds) * time.Second)
			matchDate = actualMatchTime.Format("2006-01-02 15:04:05")
		} else {
			// Si no hay duración, usar la fecha del archivo directamente
			matchDate = fileDate.Format("2006-01-02 15:04:05")
		}
	}

	/* ----------------------------------------------------------------
	   1)  Empaquetamos las GRANADAS de la última ronda (aún en memoria)
	----------------------------------------------------------------- */
	for key, meta := range ctx.GrenadeMetas {
		traj := ctx.GrenadeTrajectories[key]

		ctx.AllGrenadeTrajectories = append(ctx.AllGrenadeTrajectories,
			models.GrenadeMetadata{
				ProjectileID:      meta.ProjectileID,
				Round:             meta.Round,
				ThrowerName:       meta.ThrowerName,
				ThrowerTeam:       meta.ThrowerTeam,
				NadeType:          meta.NadeType,
				Exploded:          meta.Exploded,
				ExplosionPosition: meta.ExplosionPosition,
				Trajectory:        traj,
			})
	}

	/* ----------------------------------------------------------------
	   2)  Creamos el resultado final
	----------------------------------------------------------------- */
	parseResult := &models.DemoParseResult{
		MatchID:       matchID,
		MapName:       mapName,
		Duration:      matchDuration,
		Result:        result,
		TeamScore:     teamScore,
		OpponentScore: opponentScore,
		Players:       sortedPlayers,
		EventLogs:     ctx.EventLogs,
		Filename:      filepath.Base(filePath),
		Date:          matchDate,

		EconomyHistory: ctx.EconomyHistory,
		Grenades:       ctx.AllGrenadeTrajectories, // ← única fuente que usa exporter
	}

	return parseResult, nil
}

// determineUserTeam busca al userSteamID
func determineUserTeam(ctx *DemoContext, userSteamID string) int {
	userID64, _ := strconv.ParseUint(userSteamID, 10, 64)
	for _, pl := range ctx.parser.GameState().Participants().All() {
		if pl.SteamID64 == userID64 {
			return int(pl.Team)
		}
	}
	return 0
}

func getSteamAvatar(steamID string) (string, error) {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("steam_api_key no configurada")
	}
	urlAvatar := fmt.Sprintf(
		"http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%s",
		apiKey, steamID,
	)
	resp, err := http.Get(urlAvatar)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var avatarResp struct {
		Response struct {
			Players []struct {
				AvatarFull string `json:"avatarfull"`
			} `json:"players"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &avatarResp); err != nil {
		return "", fmt.Errorf("error parseando JSON: %v", err)
	}
	if len(avatarResp.Response.Players) == 0 {
		return "", nil
	}
	return avatarResp.Response.Players[0].AvatarFull, nil
}

// ExtractMatchDuration
func ExtractMatchDuration(jsonData string) (string, error) {
	if jsonData == "{}" || jsonData == "" {
		return "00:00", nil
	}
	var response struct {
		MatchDuration int `json:"matchDuration"`
	}
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		return "", fmt.Errorf("error parseando JSON: %v", err)
	}
	if response.MatchDuration == 0 {
		return "00:00", fmt.Errorf("⚠️ error: no se encontraron datos de matchDuration")
	}
	mins := response.MatchDuration / 60
	secs := response.MatchDuration % 60
	return fmt.Sprintf("%02d:%02d", mins, secs), nil
}
