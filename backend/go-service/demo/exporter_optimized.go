package demo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"cs2-demo-service/models"
)

// ============================================================================
// NUEVA ESTRUCTURA DE EXPORTACIÓN OPTIMIZADA PARA IA
// ============================================================================
// Se exportan 7 JSONs separados y especializados:
// 1. match_info.json - Información general de la partida
// 2. players_summary.json - Stats básicas de todos los jugadores
// 3. economy.json - Historial económico completo por ronda
// 4. events.json - Eventos críticos optimizados para IA
// 5. movement_analysis.json - Datos de movimiento y posicionamiento
// 6. combat_analytics.json - Combat metrics for AI (aim, reaction, visibility)
// 7. grenades.json - Todas las granadas con trayectorias
// ============================================================================

// ExportOptimizedData es el punto de entrada principal para exportación optimizada
func ExportOptimizedData(result *models.DemoParseResult, outputDir string) error {
	// Crear directorio si no existe
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio: %w", err)
	}

	// Exportar cada archivo
	exportFuncs := []struct {
		name string
		fn   func(*models.DemoParseResult, string) error
	}{
		{"match_info.json", exportMatchInfo},
		{"players_summary.json", exportPlayersSummary},
		{"economy.json", exportEconomyOptimized},
		{"events.json", exportEventsEnhanced},
		{"movement_analysis.json", exportMovementAnalysis},
		{"combat_analytics.json", exportCombatAnalytics},
		{"grenades.json", exportGrenadesData},
	}

	for _, ef := range exportFuncs {
		if err := ef.fn(result, outputDir); err != nil {
			return fmt.Errorf("error exportando %s: %w", ef.name, err)
		}
		fmt.Printf("✅ Exportado: %s\n", ef.name)
	}

	return nil
}

// ============================================================================
// 1. MATCH INFO - Información general de la partida
// ============================================================================
func exportMatchInfo(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "match_info.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Calcular rondas reales (TeamScore + OpponentScore)
	totalRounds := result.TeamScore + result.OpponentScore

	matchInfo := struct {
		MatchID       string `json:"match_id"`
		MapName       string `json:"map_name"`
		Duration      string `json:"duration"`
		Date          string `json:"date"`
		Result        string `json:"result"`
		TeamScore     int    `json:"team_score"`
		OpponentScore int    `json:"opponent_score"`
		Filename      string `json:"filename"`
		TotalRounds   int    `json:"total_rounds"`
	}{
		MatchID:       result.MatchID,
		MapName:       result.MapName,
		Duration:      result.Duration,
		Date:          result.Date,
		Result:        result.Result,
		TeamScore:     result.TeamScore,
		OpponentScore: result.OpponentScore,
		Filename:      result.Filename,
		TotalRounds:   totalRounds,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(matchInfo)
}

// ============================================================================
// 2. PLAYERS SUMMARY - Stats básicas de todos los jugadores
// ============================================================================
func exportPlayersSummary(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "players_summary.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Estructura simplificada solo con stats principales
	type PlayerSummary struct {
		SteamID         string  `json:"steam_id"`
		Name            string  `json:"name"`
		Team            string  `json:"team"`
		Avatar          string  `json:"avatar,omitempty"`
		Kills           int     `json:"kills"`
		Deaths          int     `json:"deaths"`
		Assists         int     `json:"assists"`
		KDRatio         float64 `json:"kd_ratio"`
		HSPercentage    float64 `json:"hs_percentage"`
		ADR             float64 `json:"adr"`
		ShotsFired      int     `json:"shots_fired"`
		ShotsHit        int     `json:"shots_hit"`
		Accuracy        float64 `json:"accuracy"`
		FirstShotAcc    float64 `json:"first_shot_accuracy"`
		AvgReactionTime float64 `json:"avg_reaction_time_ms"` // NUEVO
		DoubleKills     int     `json:"double_kills"`
		TripleKills     int     `json:"triple_kills"`
		QuadKills       int     `json:"quad_kills"`
		Ace             int     `json:"ace"`
		ClutchWins      int     `json:"clutch_wins"`
		EntryKills      int     `json:"entry_kills"`
		BombPlants      int     `json:"bomb_plants"`
		BombDefuses     int     `json:"bomb_defuses"`
		// Stats de utility
		EnemiesFlashed int `json:"enemies_flashed"`
		FlashAssists   int `json:"flash_assists"`
	}

	// 🔥 OBTENER AVATARES DE STEAM API EN PARALELO
	steamIDs := make([]string, 0, len(result.Players))
	for _, p := range result.Players {
		steamIDs = append(steamIDs, p.SteamID)
	}
	avatars := GetSteamAvatars(steamIDs)

	var players []PlayerSummary
	for _, p := range result.Players {
		accuracy := p.Accuracy

		players = append(players, PlayerSummary{
			SteamID:         p.SteamID,
			Name:            p.Name,
			Team:            p.Team,
			Avatar:          avatars[p.SteamID], // 🔥 Avatar desde Steam API
			Kills:           p.Kills,
			Deaths:          p.Deaths,
			Assists:         p.Assists,
			KDRatio:         p.KDRatio,
			HSPercentage:    p.HSPercentage,
			ADR:             p.ADR,
			ShotsFired:      p.ShotsFired,
			ShotsHit:        p.ShotsConnected,
			Accuracy:        accuracy,
			FirstShotAcc:    p.FirstShotAccuracy,
			AvgReactionTime: p.AvgReactionTimeMs,
			DoubleKills:     p.DoubleKills,
			TripleKills:     p.TripleKills,
			QuadKills:       p.QuadKills,
			Ace:             p.Ace,
			ClutchWins:      p.ClutchWins,
			EntryKills:      p.EntryKills,
			BombPlants:      p.BombPlants,
			BombDefuses:     p.BombDefuses,
			EnemiesFlashed:  p.EnemiesFlashed,
			FlashAssists:    p.FlashAssists,
		})
	}

	payload := struct {
		MatchID string          `json:"match_id"`
		Players []PlayerSummary `json:"players"`
	}{
		MatchID: result.MatchID,
		Players: players,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// ============================================================================
// 3. ECONOMY - Historial económico optimizado
// ============================================================================
func exportEconomyOptimized(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "economy.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Agrupar por ronda para mejor organización
	type RoundEconomy struct {
		Round   int                         `json:"round"`
		Context models.RoundContext         `json:"context"`
		Players []*models.RoundEconomyStats `json:"players"`
	}

	// Crear copia sin el campo Context para evitar duplicación
	type PlayerEconomyNoDupe struct {
		RoundNumber     int                `json:"round_number"`
		Name            string             `json:"name"`
		Team            string             `json:"team,omitempty"`
		InitialMoney    int                `json:"initial_money"`
		LossBonus       int                `json:"loss_bonus"`
		StartRoundItems []models.RoundItem `json:"start_round_items,omitempty"`
		SpentInBuy      int                `json:"spent_in_buy"`
		Purchases       []models.RoundItem `json:"purchases,omitempty"`
		KillReward      int                `json:"kill_reward"`
		RewardforPlant  int                `json:"Reward_for_Plant"`
		EndRoundItems   []models.RoundItem `json:"end_round_items,omitempty"`
		FinalMoney      int                `json:"final_money"`
	}

	type RoundEconomyExport struct {
		Round   int                   `json:"round"`
		Context models.RoundContext   `json:"context"`
		Players []PlayerEconomyNoDupe `json:"players"`
	}

	roundsMap := make(map[int]*RoundEconomyExport)
	for _, econ := range result.EconomyHistory {
		if roundsMap[econ.RoundNumber] == nil {
			roundsMap[econ.RoundNumber] = &RoundEconomyExport{
				Round:   econ.RoundNumber,
				Context: econ.Context,
				Players: []PlayerEconomyNoDupe{},
			}
		}
		// Convertir sin el campo Context duplicado
		playerEcon := PlayerEconomyNoDupe{
			RoundNumber:     econ.RoundNumber,
			Name:            econ.Name,
			Team:            econ.Team,
			InitialMoney:    econ.InitialMoney,
			LossBonus:       econ.LossBonus,
			StartRoundItems: econ.StartRoundItems,
			SpentInBuy:      econ.SpentInBuy,
			Purchases:       econ.Purchases,
			KillReward:      econ.KillReward,
			RewardforPlant:  econ.RewardforPlant,
			EndRoundItems:   econ.EndRoundItems,
			FinalMoney:      econ.FinalMoney,
		}
		roundsMap[econ.RoundNumber].Players = append(roundsMap[econ.RoundNumber].Players, playerEcon)
	}

	// Ordenar por ronda
	var rounds []int
	for r := range roundsMap {
		rounds = append(rounds, r)
	}
	sort.Ints(rounds)

	var orderedRounds []*RoundEconomyExport
	for _, r := range rounds {
		orderedRounds = append(orderedRounds, roundsMap[r])
	}

	payload := struct {
		MatchID string                `json:"match_id"`
		Rounds  []*RoundEconomyExport `json:"rounds"`
	}{
		MatchID: result.MatchID,
		Rounds:  orderedRounds,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// ============================================================================
// 4. EVENTS - Eventos críticos optimizados para IA
// ============================================================================
func exportEventsEnhanced(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "events.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Agrupar eventos por ronda
	type RoundEvents struct {
		Round      int                          `json:"round"`
		EventCount int                          `json:"event_count"`
		ByType     map[string][]models.EventLog `json:"events_by_type"`
		Timeline   []models.EventLog            `json:"timeline"` // Ordenado por timestamp
	}

	roundsMap := make(map[int]*RoundEvents)
	for _, ev := range result.EventLogs {
		if roundsMap[ev.Round] == nil {
			roundsMap[ev.Round] = &RoundEvents{
				Round:    ev.Round,
				ByType:   make(map[string][]models.EventLog),
				Timeline: []models.EventLog{},
			}
		}
		roundsMap[ev.Round].ByType[ev.EventType] = append(roundsMap[ev.Round].ByType[ev.EventType], ev)
		roundsMap[ev.Round].Timeline = append(roundsMap[ev.Round].Timeline, ev)
	}

	// Ordenar timeline y eventos por tipo dentro de cada ronda
	for _, re := range roundsMap {
		// Ordenar timeline por timestamp
		sort.Slice(re.Timeline, func(i, j int) bool {
			return re.Timeline[i].Timestamp < re.Timeline[j].Timestamp
		})

		// Ordenar eventos dentro de cada tipo por timestamp
		for eventType := range re.ByType {
			sort.Slice(re.ByType[eventType], func(i, j int) bool {
				return re.ByType[eventType][i].Timestamp < re.ByType[eventType][j].Timestamp
			})
		}

		re.EventCount = len(re.Timeline)
	}

	// Ordenar por ronda
	var rounds []int
	for r := range roundsMap {
		rounds = append(rounds, r)
	}
	sort.Ints(rounds)

	var orderedRounds []*RoundEvents
	for _, r := range rounds {
		orderedRounds = append(orderedRounds, roundsMap[r])
	}

	payload := struct {
		MatchID string         `json:"match_id"`
		Rounds  []*RoundEvents `json:"rounds"`
	}{
		MatchID: result.MatchID,
		Rounds:  orderedRounds,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// ============================================================================
// 6. MOVEMENT ANALYSIS - Datos de movimiento agrupados por ronda y tick
// ============================================================================
func exportMovementAnalysis(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "movement_analysis.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Estructura: Round -> Tick -> [PlayerPosition...]
	type PlayerPosition struct {
		SteamID    string  `json:"steam_id"`
		Name       string  `json:"name"`
		Team       string  `json:"team"`
		X          float64 `json:"x"`
		Y          float64 `json:"y"`
		Z          float64 `json:"z"`
		Pitch      float32 `json:"pitch"` // ViewAngle vertical
		Yaw        float32 `json:"yaw"`   // ViewAngle horizontal
		Speed      float64 `json:"speed"`
		Zone       string  `json:"zone"`
		IsIsolated bool    `json:"is_isolated"`
		IsDucking  bool    `json:"is_ducking"`
	}

	type TickSnapshot struct {
		Tick      int              `json:"tick"`
		Positions []PlayerPosition `json:"positions"`
	}

	type RoundMovement struct {
		Round     int            `json:"round"`
		Snapshots []TickSnapshot `json:"snapshots"`
	}

	// Mapa temporal: Round -> Tick -> []PlayerPosition
	roundTickMap := make(map[int]map[int][]PlayerPosition)

	// Recopilar todas las posiciones
	for _, p := range result.Players {
		for _, m := range p.Movement {
			if roundTickMap[m.Round] == nil {
				roundTickMap[m.Round] = make(map[int][]PlayerPosition)
			}

			roundTickMap[m.Round][m.Tick] = append(roundTickMap[m.Round][m.Tick], PlayerPosition{
				SteamID:    p.SteamID,
				Name:       p.Name,
				Team:       p.Team,
				X:          m.X,
				Y:          m.Y,
				Z:          m.Z,
				Pitch:      m.Pitch,
				Yaw:        m.Yaw,
				Speed:      m.Speed,
				Zone:       m.Zone,
				IsIsolated: m.IsIsolated,
				IsDucking:  m.IsDucking,
			})
		}
	}

	// Convertir a estructura ordenada
	var rounds []RoundMovement
	for roundNum, tickMap := range roundTickMap {
		// Ordenar ticks
		var ticks []int
		for tick := range tickMap {
			ticks = append(ticks, tick)
		}
		sort.Ints(ticks)

		// Construir snapshots
		var snapshots []TickSnapshot
		for _, tick := range ticks {
			snapshots = append(snapshots, TickSnapshot{
				Tick:      tick,
				Positions: tickMap[tick],
			})
		}

		rounds = append(rounds, RoundMovement{
			Round:     roundNum,
			Snapshots: snapshots,
		})
	}

	// Ordenar rondas
	sort.Slice(rounds, func(i, j int) bool {
		return rounds[i].Round < rounds[j].Round
	})

	// === AGREGAR ROTACIONES (movido desde advanced_analytics) ===
	type PlayerRotations struct {
		SteamID          string                 `json:"steam_id"`
		Name             string                 `json:"name"`
		Team             string                 `json:"team"`
		Rotations        []models.RotationEvent `json:"rotations"`
		TotalRotations   int                    `json:"total_rotations"`
		CorrectRotations int                    `json:"correct_rotations"`
		LateRotations    int                    `json:"late_rotations"`
		AvgRotationTime  float64                `json:"avg_rotation_time_seconds"`
	}

	var playersRotations []PlayerRotations
	for _, p := range result.Players {
		correctCount := 0
		lateCount := 0
		totalTimeSeconds := 0.0

		for _, rot := range p.Rotations {
			if rot.WasCorrect {
				correctCount++
			}
			if rot.WasTooLate {
				lateCount++
			}
			// Convertir ticks a segundos (64 ticks/seg)
			durationTicks := rot.EndTick - rot.StartTick
			totalTimeSeconds += float64(durationTicks) / 64.0
		}

		avgTime := 0.0
		if len(p.Rotations) > 0 {
			avgTime = totalTimeSeconds / float64(len(p.Rotations))
		}

		playersRotations = append(playersRotations, PlayerRotations{
			SteamID:          p.SteamID,
			Name:             p.Name,
			Team:             p.Team,
			Rotations:        p.Rotations,
			TotalRotations:   len(p.Rotations),
			CorrectRotations: correctCount,
			LateRotations:    lateCount,
			AvgRotationTime:  avgTime,
		})
	}

	payload := struct {
		MatchID         string            `json:"match_id"`
		Rounds          []RoundMovement   `json:"rounds"`
		PlayerRotations []PlayerRotations `json:"player_rotations"`
	}{
		MatchID:         result.MatchID,
		Rounds:          rounds,
		PlayerRotations: playersRotations,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// ============================================================================
// 6. ADVANCED ANALYTICS - Solo AIM, REACTION y VISIBILITY
// ============================================================================
// REORGANIZACIÓN POR FAMILIAS:
// - AIM: Everything related to aim (crosshair, recoil, shooting)
// - REACTION: Reaction times and duels
// - VISIBILITY: What player sees (awareness) - OPTIMIZED
// - MOVEMENT & ROTATIONS: Moved to movement_analysis.json
// ============================================================================
func exportCombatAnalytics(result *models.DemoParseResult, outputDir string) error {
	path := filepath.Join(outputDir, "combat_analytics.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	type PlayerAnalytics struct {
		SteamID string `json:"steam_id"`
		Name    string `json:"name"`
		Team    string `json:"team"`

		// === 1. AIM ANALYSIS - Análisis profundo de puntería ===
		AimAnalysis struct {
			// Crosshair Placement Score basado en tiempo a altura de cabeza
			CrosshairPlacement struct {
				Score             float64 `json:"score"`                // Score 0-100
				TimeAtHeadLevel   int     `json:"time_at_head_level"`   // Ticks
				TimeAtBodyLevel   int     `json:"time_at_body_level"`   // Ticks
				TimeAtGroundLevel int     `json:"time_at_ground_level"` // Ticks
			} `json:"crosshair_placement"`

			// First Shot Accuracy
			FirstShot struct {
				Accuracy float64 `json:"accuracy"` // % de acierto
				Hits     int     `json:"hits"`
				Misses   int     `json:"misses"`
			} `json:"first_shot"`

			// Recoil Control (solo armas automáticas válidas)
			RecoilControl struct {
				Sprays          []models.RecoilSpray `json:"sprays,omitempty"`
				TotalSprays     int                  `json:"total_sprays"`
				ExcellentSprays int                  `json:"excellent_sprays"`
				AvgControl      float64              `json:"avg_control"` // 0-100
			} `json:"recoil_control"`

			// NUEVO: Shooting Habits - Agregado optimizado de shot_contexts
			ShootingHabits struct {
				TotalShots        int     `json:"total_shots"`
				ShotsWhileMoving  int     `json:"shots_while_moving"`
				PercentageMoving  float64 `json:"percentage_moving"`
				AvgVelocity       float64 `json:"avg_velocity_on_shot"`
				ShotsDucking      int     `json:"shots_ducking"`
				PercentageDucking float64 `json:"percentage_ducking"`
			} `json:"shooting_habits"`
		} `json:"aim_analysis"`

		// === 2. REACTION TIME ANALYSIS ===
		ReactionAnalysis struct {
			Events      []models.ReactionTimeEvent `json:"events,omitempty"`
			AvgTimeMs   float64                    `json:"avg_reaction_time_ms"`
			FastestMs   int                        `json:"fastest_ms,omitempty"`
			SlowestMs   int                        `json:"slowest_ms,omitempty"`
			TotalEvents int                        `json:"total_events"`
			WinRate     float64                    `json:"win_rate"` // % duelos ganados
		} `json:"reaction_analysis"`

		// === 3. VISIBILITY ANALYSIS - Awareness (OPTIMIZADO)===
		VisibilityAnalysis struct {
			KeyMoments        []models.VisibilitySnapshot `json:"key_moments,omitempty"` // Solo momentos importantes
			TotalEnemiesSeen  int                         `json:"total_enemies_seen"`
			AvgEnemiesVisible float64                     `json:"avg_enemies_visible"`
			TunnelVisionCount int                         `json:"tunnel_vision_count"` // Vio >1 pero ignoró
		} `json:"visibility_analysis"`
	}

	var playersAnalytics []PlayerAnalytics
	for _, p := range result.Players {
		pa := PlayerAnalytics{
			SteamID: p.SteamID,
			Name:    p.Name,
			Team:    p.Team,
		}

		// === AIM ANALYSIS ===
		// Crosshair Placement
		pa.AimAnalysis.CrosshairPlacement.Score = p.CrosshairScore
		pa.AimAnalysis.CrosshairPlacement.TimeAtHeadLevel = p.TimeAtHeadLevel
		pa.AimAnalysis.CrosshairPlacement.TimeAtBodyLevel = p.TimeAtBodyLevel
		pa.AimAnalysis.CrosshairPlacement.TimeAtGroundLevel = p.TimeAtGroundLevel

		// First Shot
		pa.AimAnalysis.FirstShot.Accuracy = p.FirstShotAccuracy
		pa.AimAnalysis.FirstShot.Hits = p.FirstShotHits
		pa.AimAnalysis.FirstShot.Misses = p.FirstShotMisses

		// Recoil Control - FILTRAR armas semi-automáticas
		validSprays := []models.RecoilSpray{}
		excellentCount := 0
		totalSprayScore := 0.0

		for _, spray := range p.RecoilSprays {
			// Ignorar pistolas y AWP (no tienen recoil patterns complejos)
			weapon := spray.Weapon
			if weapon == "Glock-18" || weapon == "USP-S" || weapon == "P2000" ||
				weapon == "P250" || weapon == "Tec-9" || weapon == "Five-SeveN" ||
				weapon == "CZ75 Auto" || weapon == "Deagle" || weapon == "R8 Revolver" ||
				weapon == "SSG 08" || weapon == "AWP" {
				continue
			}

			validSprays = append(validSprays, spray)
			if spray.ControlQuality == "excellent" {
				excellentCount++
			}

			// Score basado en calidad
			switch spray.ControlQuality {
			case "excellent":
				totalSprayScore += 100
			case "good":
				totalSprayScore += 75
			case "fair":
				totalSprayScore += 50
			case "poor":
				totalSprayScore += 25
			}
		}

		avgSprayControl := 0.0
		if len(validSprays) > 0 {
			avgSprayControl = totalSprayScore / float64(len(validSprays))
		}

		pa.AimAnalysis.RecoilControl.Sprays = validSprays
		pa.AimAnalysis.RecoilControl.TotalSprays = len(validSprays)
		pa.AimAnalysis.RecoilControl.ExcellentSprays = excellentCount
		pa.AimAnalysis.RecoilControl.AvgControl = avgSprayControl

		// NUEVO: Shooting Habits - Agregado de shot_contexts
		totalShots := len(p.Shots)
		shotsMoving := 0
		shotsDucking := 0
		totalVelocity := 0.0

		for _, shot := range p.Shots {
			if shot.WasMoving {
				shotsMoving++
			}
			if shot.WasDucking {
				shotsDucking++
			}
			totalVelocity += shot.Velocity
		}

		avgVelocity := 0.0
		percentageMoving := 0.0
		percentageDucking := 0.0
		if totalShots > 0 {
			avgVelocity = totalVelocity / float64(totalShots)
			percentageMoving = float64(shotsMoving) / float64(totalShots) * 100
			percentageDucking = float64(shotsDucking) / float64(totalShots) * 100
		}

		pa.AimAnalysis.ShootingHabits.TotalShots = totalShots
		pa.AimAnalysis.ShootingHabits.ShotsWhileMoving = shotsMoving
		pa.AimAnalysis.ShootingHabits.PercentageMoving = percentageMoving
		pa.AimAnalysis.ShootingHabits.AvgVelocity = avgVelocity
		pa.AimAnalysis.ShootingHabits.ShotsDucking = shotsDucking
		pa.AimAnalysis.ShootingHabits.PercentageDucking = percentageDucking

		// === REACTION ANALYSIS ===
		pa.ReactionAnalysis.Events = p.ReactionTimes
		pa.ReactionAnalysis.AvgTimeMs = p.AvgReactionTimeMs
		pa.ReactionAnalysis.TotalEvents = p.ReactionTimeCount

		// Calcular fastest/slowest y win rate
		killsInDuels := 0
		if len(p.ReactionTimes) > 0 {
			fastest := p.ReactionTimes[0].ReactionTimeMs
			slowest := p.ReactionTimes[0].ReactionTimeMs
			for _, rt := range p.ReactionTimes {
				if rt.ReactionTimeMs < fastest {
					fastest = rt.ReactionTimeMs
				}
				if rt.ReactionTimeMs > slowest {
					slowest = rt.ReactionTimeMs
				}
				if rt.KilledEnemy {
					killsInDuels++
				}
			}
			pa.ReactionAnalysis.FastestMs = fastest
			pa.ReactionAnalysis.SlowestMs = slowest
			pa.ReactionAnalysis.WinRate = float64(killsInDuels) / float64(len(p.ReactionTimes))
		}

		// === VISIBILITY ANALYSIS - OPTIMIZADO ===
		// Solo guardar momentos clave (no todos los snapshots)
		var keyMoments []models.VisibilitySnapshot
		var prevVisibleCount int
		var prevLookingAt *uint64
		totalEnemiesSeen := make(map[uint64]bool)
		totalVisibleCount := 0
		tunnelVisionCount := 0

		for i, snap := range p.Visibility {
			currentCount := len(snap.VisibleEnemies)

			// Registrar enemigos únicos vistos
			for _, enemyID := range snap.VisibleEnemies {
				totalEnemiesSeen[enemyID] = true
			}
			totalVisibleCount += currentCount

			// Guardar snapshot si:
			// 1. Primer snapshot o último snapshot
			// 2. Cambio en cantidad de enemigos visibles
			// 3. Cambio de target (looking_at)
			isKeyMoment := false

			if i == 0 || i == len(p.Visibility)-1 {
				isKeyMoment = true
			} else if currentCount != prevVisibleCount {
				isKeyMoment = true
			} else if snap.LookingAt != prevLookingAt {
				if snap.LookingAt == nil && prevLookingAt != nil {
					isKeyMoment = true
				} else if snap.LookingAt != nil && prevLookingAt == nil {
					isKeyMoment = true
				} else if snap.LookingAt != nil && prevLookingAt != nil && *snap.LookingAt != *prevLookingAt {
					isKeyMoment = true
				}
			}

			if isKeyMoment {
				keyMoments = append(keyMoments, snap)
			}

			// Detectar tunnel vision: ve >1 enemigo pero no cambia target
			if currentCount > 1 && snap.LookingAt != nil {
				allEnemiesTargeted := false
				for _, enemyID := range snap.VisibleEnemies {
					if enemyID == *snap.LookingAt {
						allEnemiesTargeted = true
						break
					}
				}
				// Si ve múltiples enemigos pero mantiene mismo target ignorando otros
				if !allEnemiesTargeted && prevLookingAt != nil && snap.LookingAt == prevLookingAt {
					tunnelVisionCount++
				}
			}

			prevVisibleCount = currentCount
			prevLookingAt = snap.LookingAt
		}

		avgEnemiesVisible := 0.0
		if len(p.Visibility) > 0 {
			avgEnemiesVisible = float64(totalVisibleCount) / float64(len(p.Visibility))
		}

		pa.VisibilityAnalysis.KeyMoments = keyMoments
		pa.VisibilityAnalysis.TotalEnemiesSeen = len(totalEnemiesSeen)
		pa.VisibilityAnalysis.AvgEnemiesVisible = avgEnemiesVisible
		pa.VisibilityAnalysis.TunnelVisionCount = tunnelVisionCount

		playersAnalytics = append(playersAnalytics, pa)
	}

	payload := struct {
		MatchID string            `json:"match_id"`
		Players []PlayerAnalytics `json:"players"`
	}{
		MatchID: result.MatchID,
		Players: playersAnalytics,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// ============================================================================
// 7. GRENADES - Todas las granadas lanzadas con trayectorias
// ============================================================================
func exportGrenadesData(result *models.DemoParseResult, outputDir string) error {
	fmt.Printf("💣 Exportando granadas: %d granadas encontradas\n", len(result.Grenades))

	path := filepath.Join(outputDir, "grenades.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Agrupar granadas por tipo para estadísticas
	grenadeStats := make(map[string]int)
	for _, nade := range result.Grenades {
		grenadeStats[nade.GrenadeType]++
	}

	payload := struct {
		MatchID        string                `json:"match_id"`
		TotalGrenades  int                   `json:"total_grenades"`
		GrenadesByType map[string]int        `json:"grenades_by_type"`
		Grenades       []models.GrenadeEvent `json:"grenades"`
	}{
		MatchID:        result.MatchID,
		TotalGrenades:  len(result.Grenades),
		GrenadesByType: grenadeStats,
		Grenades:       result.Grenades,
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
