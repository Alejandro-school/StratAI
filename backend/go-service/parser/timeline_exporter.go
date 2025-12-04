package parser

import (
	"cs2-demo-service/models"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ExportMatchData es la función principal que orquesta la exportación de datos
func ExportMatchData(ctx *models.DemoContext, matchID string, outputBaseDir string) error {
	matchDir := filepath.Join(outputBaseDir, "match_"+matchID)
	if err := os.MkdirAll(matchDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio match: %w", err)
	}

	// 1. Combat (Kills, Damages, Shots, Player Stats)
	if err := exportCombat(ctx, matchDir); err != nil {
		return err
	}

	// 2. Grenades (Trajectories + Flash effects)
	if err := exportGrenades(ctx, matchDir); err != nil {
		return err
	}

	// 3. Movement (High frequency player positions)
	if err := exportMovement(ctx, matchDir); err != nil {
		return err
	}

	// 4. Economy (Buys, Money, Equipment Value)
	if err := exportEconomy(ctx, matchDir); err != nil {
		return err
	}

	// 5. Summary (Metadata, Scores, Final Stats)
	if err := exportSummary(ctx, matchID, matchDir); err != nil {
		return err
	}

	fmt.Printf("✅ Exportación completada en %s\n", matchDir)
	return nil
}

// ExportTimelineData exporta la estructura de timeline optimizada para coaching
// Estructura: metadata.json + timeline/round_N.json
func ExportTimelineData(ctx *models.DemoContext, matchID string, outputBaseDir string) error {
	// Crear directorio match_id/
	matchDir := filepath.Join(outputBaseDir, "match_"+matchID)
	if err := os.MkdirAll(matchDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio match: %w", err)
	}

	// 1. Export metadata.json
	if err := exportTimelineMetadata(ctx, matchID, matchDir); err != nil {
		return fmt.Errorf("error exportando metadata: %w", err)
	}

	// 2. Export timeline/round_N.json para cada ronda
	timelineDir := filepath.Join(matchDir, "timeline")
	if err := os.MkdirAll(timelineDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio timeline: %w", err)
	}

	for _, roundTimeline := range ctx.RoundTimelines {
		if err := exportRoundTimeline(roundTimeline, timelineDir); err != nil {
			return fmt.Errorf("error exportando ronda %d: %w", roundTimeline.RoundNumber, err)
		}
	}

	fmt.Printf("✅ Timeline exportada: %d rondas en %s\n", len(ctx.RoundTimelines), matchDir)
	return nil
}

func ExportAnalysisData(ctx *models.DemoContext, matchID string, outputBaseDir string) error {
	matchDir := filepath.Join(outputBaseDir, "match_"+matchID)
	analysisDir := filepath.Join(matchDir, "analysis")
	if err := os.MkdirAll(analysisDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio analysis: %w", err)
	}

	// Exportar todos los archivos de análisis en el subdirectorio analysis/
	if err := exportCombat(ctx, analysisDir); err != nil {
		return err
	}
	if err := exportGrenades(ctx, analysisDir); err != nil {
		return err
	}
	if err := exportEconomyAnalysis(ctx, "economy.json", analysisDir); err != nil {
		return err
	}
	if err := exportPlayersAnalysis(ctx, analysisDir); err != nil {
		return err
	}

	// Exportar chat si hay mensajes
	chatEvents := []models.TimelineEvent{}
	for _, rt := range ctx.RoundTimelines {
		for _, event := range rt.Events {
			if event.Type == "chat" {
				chatEvents = append(chatEvents, event)
			}
		}
	}
	if len(chatEvents) > 0 {
		chatPath := filepath.Join(analysisDir, "chat.json")
		if err := writeJSON(chatPath, chatEvents); err != nil {
			return err
		}
		fmt.Printf("✅ Exportado: analysis/chat.json (%d eventos)\n", len(chatEvents))
	}

	fmt.Printf("✅ Análisis exportado en %s\n", analysisDir)
	return nil
}

// exportTimelineMetadata crea el archivo metadata.json con información de la partida
func exportTimelineMetadata(ctx *models.DemoContext, matchID string, matchDir string) error {
	gs := ctx.Parser.GameState()

	// Construir lista de jugadores con estadísticas
	playerMetadata := []models.PlayerMetadata{}
	for steamID, playerData := range ctx.MatchData.Players {
		kdRatio := 0.0
		if playerData.Deaths > 0 {
			kdRatio = float64(playerData.Kills) / float64(playerData.Deaths)
		}

		// Calcular ADR (Average Damage per Round)
		totalRounds := len(ctx.RoundTimelines)
		adr := 0.0
		if totalRounds > 0 {
			adr = float64(playerData.Damage) / float64(totalRounds)
		}

		playerMetadata = append(playerMetadata, models.PlayerMetadata{
			Name:    playerData.Name,
			SteamID: steamID,
			Team:    playerData.Team,
			Kills:   playerData.Kills,
			Deaths:  playerData.Deaths,
			Assists: playerData.Assists,
			Damage:  playerData.Damage,
			MVPs:    playerData.MVPs,
			KDRatio: kdRatio,
			ADR:     adr,
		})
	}

	// Construir resumen de rondas
	roundSummaries := []models.RoundSummary{}
	for _, r := range ctx.MatchData.Rounds {
		roundSummaries = append(roundSummaries, models.RoundSummary{
			Round:   r.Round,
			Winner:  r.Winner,
			Reason:  r.Reason,
			CTScore: r.CTScore,
			TScore:  r.TScore,
		})
	}

	// Determinar ganador
	ctScore := gs.TeamCounterTerrorists().Score()
	tScore := gs.TeamTerrorists().Score()
	winner := "Unknown"
	if ctScore > tScore {
		winner = "CT"
	} else if tScore > ctScore {
		winner = "T"
	} else {
		winner = "Draw"
	}

	header := ctx.Parser.Header()

	metadata := models.MatchMetadata{
		MatchID:      matchID,
		Map:          header.MapName,
		Date:         "",
		TickRate:     header.FrameRate(),
		Duration:     0,
		CTScore:      ctScore,
		TScore:       tScore,
		Winner:       winner,
		Players:      playerMetadata,
		RoundSummary: roundSummaries,
		EconomyStats: ctx.MatchData.Economy,
		TotalRounds:  len(ctx.RoundTimelines),
	}

	metadataPath := filepath.Join(matchDir, "metadata.json")
	return writeJSON(metadataPath, metadata)
}

// exportRoundTimeline exporta una ronda completa en timeline/round_N.json
func exportRoundTimeline(roundTimeline models.RoundTimeline, timelineDir string) error {
	filename := fmt.Sprintf("round_%02d.json", roundTimeline.RoundNumber)
	filePath := filepath.Join(timelineDir, filename)

	if err := writeJSON(filePath, roundTimeline); err != nil {
		return err
	}

	fmt.Printf("✅ Exportado: timeline/%s (%d eventos)\n", filename, len(roundTimeline.Events))
	return nil
}

func exportPlayersAnalysis(ctx *models.DemoContext, analysisDir string) error {
	// Crear versión limpia sin el array de movimiento (demasiado pesado)
	type PlayerStats struct {
		SteamID       uint64                     `json:"steam_id"`
		Name          string                     `json:"name"`
		Team          string                     `json:"team"`
		Kills         int                        `json:"kills"`
		Deaths        int                        `json:"deaths"`
		Assists       int                        `json:"assists"`
		Damage        int                        `json:"damage"`
		MVPs          int                        `json:"mvps"`
		HeadshotPct   float64                    `json:"headshot_percentage,omitempty"`
		Crosshair     *models.CrosshairStats     `json:"crosshair,omitempty"`
		Mechanics     *models.MechanicsStats     `json:"mechanics,omitempty"`
		ReactionTimes []models.ReactionTimeEvent `json:"reaction_times,omitempty"`
		Sprays        []models.SprayAnalysis     `json:"sprays,omitempty"`
	}

	cleanPlayers := make(map[uint64]PlayerStats)

	for steamID, p := range ctx.MatchData.Players {
		headshotPct := 0.0
		if p.Kills > 0 {
			// Calcular headshots desde kills en timeline
			headshots := 0
			for _, rt := range ctx.RoundTimelines {
				for _, e := range rt.Events {
					if e.Type == "kill" && e.Kill != nil && e.Kill.KillerSteamID == steamID && e.Kill.IsHeadshot {
						headshots++
					}
				}
			}
			headshotPct = float64(headshots) / float64(p.Kills) * 100.0
		}

		cleanPlayers[steamID] = PlayerStats{
			SteamID:       p.SteamID,
			Name:          p.Name,
			Team:          p.Team,
			Kills:         p.Kills,
			Deaths:        p.Deaths,
			Assists:       p.Assists,
			Damage:        p.Damage,
			MVPs:          p.MVPs,
			HeadshotPct:   headshotPct,
			Crosshair:     p.Crosshair,
			Mechanics:     p.Mechanics,
			ReactionTimes: p.ReactionTimes,
			Sprays:        p.Sprays,
		}
	}

	filePath := filepath.Join(analysisDir, "players.json")
	return writeJSON(filePath, cleanPlayers)
}

// --- 1. COMBAT ---

func exportCombat(ctx *models.DemoContext, outputDir string) error {
	type Position struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	}

	type CombatEvent struct {
		Tick int         `json:"tick"`
		Type string      `json:"type"`
		Data interface{} `json:"data,omitempty"`
	}

	type DamageData struct {
		Attacker     string `json:"attacker"`
		Victim       string `json:"victim"`
		Weapon       string `json:"weapon"`
		Damage       int    `json:"damage"`
		ArmorDamage  int    `json:"armor_damage"`
		HitGroup     string `json:"hit_group"`
		VictimHealth int    `json:"victim_health"`
	}

	type KillData struct {
		Killer          string              `json:"killer"`
		KillerSteamID   uint64              `json:"killer_steam_id"`
		Victim          string              `json:"victim"`
		VictimSteamID   uint64              `json:"victim_steam_id"`
		Assister        string              `json:"assister,omitempty"`
		AssisterSteamID uint64              `json:"assister_steam_id,omitempty"`
		Weapon          string              `json:"weapon"`
		Headshot        bool                `json:"headshot"`
		Wallbang        bool                `json:"wallbang"`
		ThroughSmoke    bool                `json:"through_smoke"`
		AttackerBlind   bool                `json:"attacker_blind"`
		NoScope         bool                `json:"no_scope"`
		Distance        float64             `json:"distance"`
		Position        map[string]Position `json:"position"`
		CounterStrafe   float64             `json:"counter_strafe_rating,omitempty"`
	}

	type CombatRound struct {
		Round  int           `json:"round"`
		Events []CombatEvent `json:"events"`
	}

	type CombatSummary struct {
		TotalKills         int     `json:"total_kills"`
		TotalDamages       int     `json:"total_damages"`
		HeadshotPercentage float64 `json:"headshot_percentage"`
		WallbangKills      int     `json:"wallbang_kills"`
		ThroughSmokeKills  int     `json:"through_smoke_kills"`
	}

	type CombatAnalysis struct {
		Rounds  []CombatRound `json:"rounds"`
		Summary CombatSummary `json:"summary"`
	}

	analysis := CombatAnalysis{
		Rounds: []CombatRound{},
	}

	totalKills := 0
	headshotKills := 0
	wallbangKills := 0
	smokekills := 0
	totalDamages := 0

	for _, rt := range ctx.RoundTimelines {
		round := CombatRound{
			Round:  rt.RoundNumber,
			Events: []CombatEvent{},
		}

		for _, e := range rt.Events {
			if e.Type == "damage" && e.Damage != nil {
				totalDamages++
				round.Events = append(round.Events, CombatEvent{
					Tick: e.Tick,
					Type: "damage",
					Data: DamageData{
						Attacker:     e.Damage.Attacker,
						Victim:       e.Damage.Victim,
						Weapon:       e.Damage.Weapon,
						Damage:       e.Damage.HealthDamage,
						ArmorDamage:  e.Damage.ArmorDamage,
						HitGroup:     e.Damage.HitGroup,
						VictimHealth: e.Damage.VictimHealth,
					},
				})
			} else if e.Type == "kill" && e.Kill != nil {
				totalKills++
				if e.Kill.IsHeadshot {
					headshotKills++
				}
				if e.Kill.IsWallbang {
					wallbangKills++
				}
				if e.Kill.ThroughSmoke {
					smokekills++
				}

				round.Events = append(round.Events, CombatEvent{
					Tick: e.Tick,
					Type: "kill",
					Data: KillData{
						Killer:          e.Kill.Killer,
						KillerSteamID:   e.Kill.KillerSteamID,
						Victim:          e.Kill.Victim,
						VictimSteamID:   e.Kill.VictimSteamID,
						Assister:        e.Kill.Assister,
						AssisterSteamID: e.Kill.AssisterSteamID,
						Weapon:          e.Kill.Weapon,
						Headshot:        e.Kill.IsHeadshot,
						Wallbang:        e.Kill.IsWallbang,
						ThroughSmoke:    e.Kill.ThroughSmoke,
						AttackerBlind:   e.Kill.AttackerBlind,
						NoScope:         e.Kill.NoScope,
						Distance:        float64(e.Kill.Distance),
						Position: map[string]Position{
							"killer": {X: e.Kill.KillerX, Y: e.Kill.KillerY, Z: e.Kill.KillerZ},
							"victim": {X: e.Kill.VictimX, Y: e.Kill.VictimY, Z: e.Kill.VictimZ},
						},
						CounterStrafe: e.Kill.CounterStrafeRating,
					},
				})
			}
		}

		analysis.Rounds = append(analysis.Rounds, round)
	}

	// Calcular summary
	headshotPct := 0.0
	if totalKills > 0 {
		headshotPct = float64(headshotKills) / float64(totalKills) * 100.0
	}

	analysis.Summary = CombatSummary{
		TotalKills:         totalKills,
		TotalDamages:       totalDamages,
		HeadshotPercentage: headshotPct,
		WallbangKills:      wallbangKills,
		ThroughSmokeKills:  smokekills,
	}

	return writeJSON(filepath.Join(outputDir, "combat.json"), analysis)
}

// --- 2. GRENADES ---

func exportGrenades(ctx *models.DemoContext, outputDir string) error {
	type GrenadeEvent struct {
		Type         string      `json:"type"`
		Thrower      string      `json:"thrower"`
		TickExplode  int         `json:"tick_explode"`
		StartPos     models.XYZ  `json:"start_position,omitempty"`
		EndPos       models.XYZ  `json:"end_position"`
		FlashVictims interface{} `json:"blinded_players,omitempty"`
	}

	type GrenadeRound struct {
		RoundNumber int            `json:"round"`
		Grenades    []GrenadeEvent `json:"grenades"`
	}

	analysis := struct {
		Rounds []GrenadeRound `json:"rounds"`
	}{Rounds: []GrenadeRound{}}

	for _, rt := range ctx.RoundTimelines {
		round := GrenadeRound{
			RoundNumber: rt.RoundNumber,
			Grenades:    []GrenadeEvent{},
		}

		trajectories := []models.TimelineEvent{}
		flashes := []models.TimelineEvent{}

		for _, e := range rt.Events {
			if e.Type == "grenade_trajectory" {
				trajectories = append(trajectories, e)
			} else if e.Type == "flash" {
				flashes = append(flashes, e)
			}
		}

		for _, t := range trajectories {
			traj := t.GrenadeTrajectory
			ge := GrenadeEvent{
				Type:        traj.GrenadeType,
				Thrower:     traj.Thrower,
				TickExplode: t.Tick,
				EndPos:      traj.LandPosition,
			}
			if len(traj.Positions) > 0 {
				ge.StartPos = traj.Positions[0]
			}

			if traj.GrenadeType == "Flashbang" {
				for _, f := range flashes {
					if abs(f.Tick-t.Tick) <= 10 && f.Flash.Thrower == traj.Thrower {
						ge.FlashVictims = f.Flash.Victims
						break
					}
				}
			}
			round.Grenades = append(round.Grenades, ge)
		}
		analysis.Rounds = append(analysis.Rounds, round)
	}

	return writeJSON(filepath.Join(outputDir, "grenades.json"), analysis)
}

// --- 3. MOVEMENT ---

func exportMovement(ctx *models.DemoContext, outputDir string) error {
	type PlayerPos struct {
		SteamID uint64  `json:"steam_id"`
		Name    string  `json:"name"`
		Team    string  `json:"team"`
		X       float64 `json:"x"`
		Y       float64 `json:"y"`
		Z       float64 `json:"z"`
		Yaw     float64 `json:"yaw"`
		Pitch   float64 `json:"pitch"`
	}

	type TickMovement struct {
		Tick    int         `json:"tick"`
		Players []PlayerPos `json:"players"`
	}

	type MovementRound struct {
		RoundNumber int            `json:"round"`
		Ticks       []TickMovement `json:"ticks"`
	}

	analysis := struct {
		Rounds []MovementRound `json:"rounds"`
	}{Rounds: []MovementRound{}}

	for _, rt := range ctx.RoundTimelines {
		round := MovementRound{
			RoundNumber: rt.RoundNumber,
			Ticks:       []TickMovement{},
		}

		for _, e := range rt.Events {
			if e.Type == "game_state" && e.GameState != nil {
				tm := TickMovement{
					Tick:    e.Tick,
					Players: []PlayerPos{},
				}
				for _, p := range e.GameState.Players {
					if p.IsAlive {
						tm.Players = append(tm.Players, PlayerPos{
							SteamID: p.SteamID,
							Name:    p.Name,
							Team:    p.Team,
							X:       p.X,
							Y:       p.Y,
							Z:       p.Z,
							Yaw:     p.ViewY,
							Pitch:   p.ViewX,
						})
					}
				}
				if len(tm.Players) > 0 {
					round.Ticks = append(round.Ticks, tm)
				}
			}
		}
		analysis.Rounds = append(analysis.Rounds, round)
	}

	return writeJSON(filepath.Join(outputDir, "movement.json"), analysis)
}

// --- 4. ECONOMY ---

func exportEconomy(ctx *models.DemoContext, outputDir string) error {
	return exportEconomyAnalysis(ctx, "economy.json", outputDir)
}

// --- 5. SUMMARY ---

func exportSummary(ctx *models.DemoContext, matchID string, outputDir string) error {
	return exportMetadata(ctx, matchID, outputDir, "summary.json")
}

// --- HELPERS ---

func writeJSON(path string, data interface{}) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// exportMetadata crea el archivo summary.json
func exportMetadata(ctx *models.DemoContext, matchID string, matchDir string, filename string) error {
	gs := ctx.Parser.GameState()

	playerMetadata := []models.PlayerMetadata{}
	for steamID, playerData := range ctx.MatchData.Players {
		kdRatio := 0.0
		if playerData.Deaths > 0 {
			kdRatio = float64(playerData.Kills) / float64(playerData.Deaths)
		}

		totalRounds := len(ctx.RoundTimelines)
		adr := 0.0
		if totalRounds > 0 {
			adr = float64(playerData.Damage) / float64(totalRounds)
		}

		playerMetadata = append(playerMetadata, models.PlayerMetadata{
			Name:    playerData.Name,
			SteamID: steamID,
			Team:    playerData.Team,
			Kills:   playerData.Kills,
			Deaths:  playerData.Deaths,
			Assists: playerData.Assists,
			Damage:  playerData.Damage,
			MVPs:    playerData.MVPs,
			KDRatio: kdRatio,
			ADR:     adr,
		})
	}

	roundSummaries := []models.RoundSummary{}
	for _, r := range ctx.MatchData.Rounds {
		roundSummaries = append(roundSummaries, models.RoundSummary{
			Round:   r.Round,
			Winner:  r.Winner,
			Reason:  r.Reason,
			CTScore: r.CTScore,
			TScore:  r.TScore,
		})
	}

	ctScore := gs.TeamCounterTerrorists().Score()
	tScore := gs.TeamTerrorists().Score()
	winner := "Unknown"
	if ctScore > tScore {
		winner = "CT"
	} else if tScore > ctScore {
		winner = "T"
	} else {
		winner = "Draw"
	}

	header := ctx.Parser.Header()

	metadata := models.MatchMetadata{
		MatchID:      matchID,
		Map:          header.MapName,
		Date:         "",
		TickRate:     header.FrameRate(),
		Duration:     0,
		CTScore:      ctScore,
		TScore:       tScore,
		Winner:       winner,
		Players:      playerMetadata,
		RoundSummary: roundSummaries,
		EconomyStats: ctx.MatchData.Economy,
		TotalRounds:  len(ctx.RoundTimelines),
	}

	return writeJSON(filepath.Join(matchDir, filename), metadata)
}

// exportEconomyAnalysis genera el archivo economy.json
func exportEconomyAnalysis(ctx *models.DemoContext, filename string, outputDir string) error {
	type Purchase struct {
		Name   string `json:"name"`
		Price  int    `json:"price"`
		Refund bool   `json:"refund,omitempty"`
	}

	type PlayerEconomy struct {
		RoundNumber     int    `json:"round_number"`
		Name            string `json:"name"`
		Team            string `json:"team"`
		InitialMoney    int    `json:"initial_money"`
		StartRoundItems []struct {
			Name  string `json:"name"`
			Price int    `json:"price"`
		} `json:"start_round_items"`
		SpentInBuy int        `json:"spent_in_buy"`
		Purchases  []Purchase `json:"purchases"`
		FinalMoney int        `json:"final_money"`
	}

	type TeamEconomy struct {
		TotalMoney int    `json:"total_money"`
		LossBonus  int    `json:"loss_bonus"`
		BuyType    string `json:"buy_type"`
	}

	type RoundEconomy struct {
		Round   int                    `json:"round"`
		Teams   map[string]TeamEconomy `json:"teams"`
		Players []PlayerEconomy        `json:"players"`
	}

	type EconomyAnalysis struct {
		MatchID string         `json:"match_id"`
		Rounds  []RoundEconomy `json:"rounds"`
	}

	analysis := EconomyAnalysis{
		MatchID: ctx.MatchData.MatchID,
		Rounds:  []RoundEconomy{},
	}

	for _, rt := range ctx.RoundTimelines {
		roundEco := RoundEconomy{
			Round:   rt.RoundNumber,
			Teams:   make(map[string]TeamEconomy),
			Players: []PlayerEconomy{},
		}

		var roundStart *models.RoundStartEvent
		for _, e := range rt.Events {
			if e.Type == "round_start" {
				roundStart = e.RoundStart
				break
			}
		}

		if roundStart != nil {
			ctBuyType := "Unknown"
			if roundStart.CTStartMoney < 10000 {
				ctBuyType = "Eco"
			} else if roundStart.CTStartMoney < 20000 {
				ctBuyType = "Force Buy"
			} else {
				ctBuyType = "Full Buy"
			}

			tBuyType := "Unknown"
			if roundStart.TStartMoney < 10000 {
				tBuyType = "Eco"
			} else if roundStart.TStartMoney < 20000 {
				tBuyType = "Force Buy"
			} else {
				tBuyType = "Full Buy"
			}

			roundEco.Teams["CT"] = TeamEconomy{
				TotalMoney: roundStart.CTStartMoney,
				LossBonus:  roundStart.CTLossBonus,
				BuyType:    ctBuyType,
			}
			roundEco.Teams["T"] = TeamEconomy{
				TotalMoney: roundStart.TStartMoney,
				LossBonus:  roundStart.TLossBonus,
				BuyType:    tBuyType,
			}

			playerMap := make(map[uint64]*PlayerEconomy)

			for _, p := range roundStart.Players {
				items := []struct {
					Name  string `json:"name"`
					Price int    `json:"price"`
				}{}
				for _, item := range p.Inventory {
					items = append(items, struct {
						Name  string `json:"name"`
						Price int    `json:"price"`
					}{Name: item, Price: 0})
				}

				pe := &PlayerEconomy{
					RoundNumber:     rt.RoundNumber,
					Name:            p.Name,
					Team:            p.Team,
					InitialMoney:    p.Money,
					StartRoundItems: items,
					SpentInBuy:      0,
					Purchases:       []Purchase{},
					FinalMoney:      p.Money,
				}
				playerMap[p.SteamID] = pe
			}

			for _, e := range rt.Events {
				if e.Type == "buy" && e.Buy != nil {
					if pe, ok := playerMap[e.Buy.SteamID]; ok {
						pe.Purchases = append(pe.Purchases, Purchase{
							Name:   e.Buy.Item,
							Price:  e.Buy.Cost,
							Refund: e.Buy.Refund,
						})
						pe.SpentInBuy += e.Buy.Cost
						pe.FinalMoney = e.Buy.MoneyLeft
					}
				}
			}

			for _, pe := range playerMap {
				roundEco.Players = append(roundEco.Players, *pe)
			}
		}

		analysis.Rounds = append(analysis.Rounds, roundEco)
	}

	return writeJSON(filepath.Join(outputDir, filename), analysis)
}
