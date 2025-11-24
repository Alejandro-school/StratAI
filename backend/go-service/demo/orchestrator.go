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
func ProcessDemoFile(filePath, userSteamID string, jsonData string, matchIDFromSteam string) (*models.DemoParseResult, error) {
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

	// Parsear header primero para obtener información de la demo
	log.Printf("⏳ Parseando header...")
	header, err := parser.ParseHeader()
	if err != nil {
		return nil, fmt.Errorf("error al parsear header: %v", err)
	}

	log.Printf("📋 Info de la demo:")
	log.Printf("   Mapa: %s", header.MapName)
	log.Printf("   Duración: %v", header.PlaybackTime)
	log.Printf("   Ticks totales: %d", header.PlaybackTicks)
	log.Printf("   Framerate: %f", header.FrameRate())

	// Creamos el contexto
	log.Printf("⏳ Creando contexto y registrando handlers...")
	ctx := NewDemoContext()
	ctx.parser = parser

	// Registramos handlers
	registerRoundHandlers(ctx)
	registerBombHandlers(ctx)
	registerPlayerHandlers(ctx)
	registerMovementHandlers(ctx)
	registerGrenadeHandlers(ctx)

	// Parseamos
	log.Printf("⏳ Iniciando ParseToEnd (esto puede tardar)...")
	parseStart := time.Now()
	if err := parser.ParseToEnd(); err != nil {
		log.Printf("⚠️ Error parsing demo: %v", err)
		// No retornamos error, continuamos para ver qué datos se obtuvieron
	}
	parseElapsed := time.Since(parseStart)
	log.Printf("✅ ParseToEnd completado en %.2f segundos", parseElapsed.Seconds())

	log.Printf("   Rondas detectadas: %d", ctx.RoundNumber)
	log.Printf("   Jugadores en PlayerStatsMap: %d", len(ctx.PlayerStatsMap))

	// Compilamos resultados
	log.Printf("⏳ Compilando resultados...")
	parseResult, err := compileResults(ctx, filePath, userSteamID, matchDuration, matchIDFromSteam)
	if err != nil {
		return nil, err
	}

	// Exportar archivos JSON optimizados
	// Directorio de exportación en data/exports/{match_id}/
	log.Printf("⏳ Exportando datos optimizados...")
	currentDir, _ := os.Getwd()
	var outputDir string

	// Detectar si estamos en go-service o en la raíz del proyecto
	if filepath.Base(currentDir) == "go-service" {
		outputDir = filepath.Join(currentDir, "..", "data", "exports", parseResult.MatchID)
	} else {
		// Si estamos en otro directorio, usar ruta relativa
		outputDir = filepath.Join(currentDir, "data", "exports", parseResult.MatchID)
	}

	exportStart := time.Now()
	err = ExportOptimizedData(parseResult, outputDir)
	exportElapsed := time.Since(exportStart)

	if err != nil {
		log.Printf("⚠️ Error exportando resultados: %v", err)
		// No retornamos error, continuamos
	} else {
		log.Printf("✅ Exportación: 7 archivos JSON en %s (%.2fs)", outputDir, exportElapsed.Seconds())
	}

	log.Printf("✅ Parsing completado para %s", parseResult.MatchID)
	return parseResult, nil
}

// compileResults arma el DemoParseResult usando la info de ctx
func compileResults(ctx *DemoContext, filePath string, userSteamID string, matchDuration string, matchIDFromSteam string) (*models.DemoParseResult, error) {
	header := ctx.parser.Header()
	mapName := header.MapName
	gs := ctx.parser.GameState()
	tScore := gs.TeamTerrorists().Score()
	ctScore := gs.TeamCounterTerrorists().Score()

	// Si los scores están en 0, intentar obtenerlos de otra manera
	if tScore == 0 && ctScore == 0 {
		log.Printf("⚠️ Scores en 0, verificando GameState...")
		log.Printf("GameState válido: %v", gs != nil)
		log.Printf("Rondas jugadas: %d", ctx.RoundNumber)

		// Intentar obtener scores de los participantes
		participants := gs.Participants().All()
		log.Printf("Participantes encontrados: %d", len(participants))

		// Contar rondas ganadas por equipo manualmente si es necesario
		// Esto es un fallback si el GameState no tiene los scores correctos
	}

	// Extraer la duración real del Header del demo
	actualDuration := header.PlaybackTime
	durationString := fmt.Sprintf("%02d:%02d", int(actualDuration.Minutes()), int(actualDuration.Seconds())%60)

	// Si matchDuration del JSON estaba vacío, usar la duración real del Header
	if matchDuration == "00:00" || matchDuration == "" {
		matchDuration = durationString
	}

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

		// === NUEVO: Calcular First Shot Accuracy ===
		totalFirstShots := ps.FirstShotHits + ps.FirstShotMisses
		if totalFirstShots > 0 {
			ps.FirstShotAccuracy = (float64(ps.FirstShotHits) / float64(totalFirstShots)) * 100
		} else if ps.FirstShotHits > 0 {
			// Si solo tenemos hits (no trackeamos misses aún), usar como aproximación
			ps.FirstShotAccuracy = 100.0 // Aproximación optimista
		}

		// === NUEVO: Calcular Crosshair Placement Score ===
		totalCrosshairTicks := ps.TimeAtHeadLevel + ps.TimeAtBodyLevel + ps.TimeAtGroundLevel
		if totalCrosshairTicks > 0 {
			// Score basado en % de tiempo a head level
			headLevelPct := float64(ps.TimeAtHeadLevel) / float64(totalCrosshairTicks)
			bodyLevelPct := float64(ps.TimeAtBodyLevel) / float64(totalCrosshairTicks)

			// Head level = 100%, Body level = 60%, Ground = 0%
			ps.CrosshairScore = (headLevelPct * 100) + (bodyLevelPct * 60)
		}

		// === NUEVO: Calcular Average Reaction Time ===
		if ps.ReactionTimeCount > 0 {
			ps.AvgReactionTimeMs = float64(ps.TotalReactionTimeMs) / float64(ps.ReactionTimeCount)
		}

		// === NUEVO: Calcular Average Spray Control ===
		if ps.TotalSprays > 0 {
			// Score basado en % de sprays excelentes vs poor
			excellentPct := float64(ps.ExcellentSprays) / float64(ps.TotalSprays)
			poorPct := float64(ps.PoorSprays) / float64(ps.TotalSprays)

			// Excellent = 100 puntos, Poor = 0 puntos
			ps.AvgSprayControl = (excellentPct * 100) - (poorPct * 50)
			if ps.AvgSprayControl < 0 {
				ps.AvgSprayControl = 0
			}
		}

		// === NUEVO: Calcular estadísticas finales por arma ===
		if ps.WeaponStats != nil {
			for _, ws := range ps.WeaponStats {
				// Accuracy
				if ws.ShotsFired > 0 {
					ws.Accuracy = (float64(ws.ShotsHit) / float64(ws.ShotsFired)) * 100
				}

				// Headshot percentage
				if ws.Kills > 0 {
					ws.HeadshotPercent = (float64(ws.Headshots) / float64(ws.Kills)) * 100
				}
			}
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

	// DESACTIVADO: Las llamadas a Steam API tardan mucho, usar avatares vacíos
	/*
		for i := range sortedPlayers {
			steamIDStr := sortedPlayers[i].SteamID
			avatarURL, err := getSteamAvatar(steamIDStr)
			if err != nil {
				log.Printf("error retrieving steam avatar for %s: %v", steamIDStr, err)
			} else {
				sortedPlayers[i].Avatar = avatarURL
			}
		}
	*/

	// Usar el matchID que viene de Steam en lugar del hash del archivo
	// Esto asegura consistencia entre el nombre del archivo y la carpeta de exportación
	matchID := matchIDFromSteam // Usar el matchID que viene del parámetro

	// Si no hay matchID de Steam, usar el hash del archivo como fallback
	if matchID == "" {
		computedMatchID, err := ComputeFileHash(filePath)
		if err != nil {
			return nil, fmt.Errorf("error generando matchID: %v", err)
		}
		matchID = computedMatchID
	}

	teamScore, opponentScore := tScore, ctScore
	if userTeam == 3 {
		teamScore, opponentScore = ctScore, tScore
	}

	result := "defeat"
	if teamScore > opponentScore {
		result = "victory"
	}

	// Calcular la fecha y hora de la partida usando el timestamp del header del demo
	// El header del demo contiene información más precisa que la fecha del archivo
	matchDate := time.Now().Format("2006-01-02 15:04:05")

	// Intentar extraer la fecha del header del demo
	// Nota: El header de CS:GO no tiene timestamp directo, usamos fecha del archivo
	if fileInfo, err := os.Stat(filePath); err == nil {
		// Fallback: usar fecha de modificación del archivo
		fileDate := fileInfo.ModTime()

		// Restar la duración del partido para obtener la hora aproximada de inicio
		if actualDuration > 0 {
			approximateStartTime := fileDate.Add(-actualDuration)
			matchDate = approximateStartTime.Format("2006-01-02 15:04:05")
			log.Printf("📅 Usando fecha de archivo calculada: %s", matchDate)
		} else {
			matchDate = fileDate.Format("2006-01-02 15:04:05")
			log.Printf("📅 Usando fecha de archivo directa: %s", matchDate)
		}
	}

	/* ----------------------------------------------------------------
	   Creamos el resultado final
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
		Grenades:       ctx.Grenades,
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
