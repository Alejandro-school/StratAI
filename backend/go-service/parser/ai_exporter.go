package parser

import (
	"cs2-demo-service/models"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func countGrenadesByType(events []models.AI_GrenadeEvent) (total, smokes, flashes, hes, molotovs, decoys int) {
	for _, e := range events {
		total++
		typeKey := strings.ToLower(strings.TrimSpace(e.Type))
		switch typeKey {
		case "smoke", "smokegrenade", "smoke grenade":
			smokes++
		case "flashbang", "flash":
			flashes++
		case "he", "hegrenade", "he grenade", "high explosive":
			hes++
		case "molotov", "incendiary", "incendiary grenade":
			molotovs++
		case "decoy", "decoy grenade":
			decoys++
		}
	}

	return total, smokes, flashes, hes, molotovs, decoys
}

// ExportAIModels exports the data for AI agents
// matchDate is optional - the date from Steam GC in ISO 8601 format
func ExportAIModels(ctx *models.DemoContext, matchID string, outputDir string, matchDate ...string) error {
	matchDir := filepath.Join(outputDir, fmt.Sprintf("match_%s", matchID))
	if err := os.MkdirAll(matchDir, 0755); err != nil {
		return fmt.Errorf("failed to create match directory: %w", err)
	}

	// Get header info for duration and tick rate
	header := ctx.Parser.Header()
	tickRate := ctx.Parser.TickRate()

	// Calculate duration in seconds from PlaybackTime (time.Duration)
	durationSeconds := header.PlaybackTime.Seconds()

	// Use date from parameter if provided
	dateStr := ""
	if len(matchDate) > 0 && matchDate[0] != "" {
		dateStr = matchDate[0]
		fmt.Printf("📅 Match date: %s\n", dateStr)
	}

	// 1. Export Metadata
	// Construct score string
	score := fmt.Sprintf("%d-%d", ctx.MatchData.CTScore, ctx.MatchData.TScore) // CT-T convention usually

	metadata := models.AI_Metadata{
		MatchID:         matchID,
		MapName:         ctx.MatchData.MapName,
		FinalScore:      score,
		Winner:          ctx.MatchData.Winner,
		Date:            dateStr,
		DurationSeconds: durationSeconds,
		TickRate:        tickRate,
		TotalRounds:     ctx.CurrentRound,
	}

	if err := writeJSON(filepath.Join(matchDir, "metadata.json"), metadata); err != nil {
		return err
	}

	// 2. Export Tracking Data (Grouped by Round and Tick)
	trackingExport := buildTrackingExport(ctx)
	if err := writeJSON(filepath.Join(matchDir, "tracking.json"), trackingExport); err != nil {
		return err
	}

	// 3. Export Consolidated Duels (Grouped by Round) to combat.json
	// Group duels by round
	duelRoundMap := make(map[int][]models.AI_Duel)
	for _, duel := range ctx.AI_Duels {
		duelRoundMap[duel.Round] = append(duelRoundMap[duel.Round], duel)
	}

	// Create ordered list of duel rounds
	var duelRounds []models.AI_DuelRound
	duelCounter := 0 // Global counter for sequential duel IDs
	for round := 1; round <= ctx.CurrentRound; round++ {
		if duels, exists := duelRoundMap[round]; exists {
			// Sort duels by tick_start
			sort.Slice(duels, func(i, j int) bool {
				return duels[i].TickStart < duels[j].TickStart
			})
			// Reassign duel_id sequentially after sorting
			for i := range duels {
				duelCounter++
				duels[i].DuelID = fmt.Sprintf("duel_%d", duelCounter)
			}
			duelRounds = append(duelRounds, models.AI_DuelRound{
				Round: round,
				Duels: duels,
			})
		}
	}

	duelExport := models.AI_DuelExport{
		Rounds: duelRounds,
	}

	if err := writeJSON(filepath.Join(matchDir, "combat.json"), duelExport); err != nil {
		return err
	}
	// 4. Export Economy Data
	economyExport := []models.AI_EconomyMatch{
		{
			MatchID: matchID,
			Rounds:  ctx.AI_EconomyRounds,
		},
	}
	if err := writeEconomyJSON(filepath.Join(matchDir, "economy.json"), economyExport); err != nil {
		return err
	}

	// 5. Group and Export Grenade Events by Round
	sort.Slice(ctx.AI_GrenadeEvents, func(i, j int) bool {
		if ctx.AI_GrenadeEvents[i].Round != ctx.AI_GrenadeEvents[j].Round {
			return ctx.AI_GrenadeEvents[i].Round < ctx.AI_GrenadeEvents[j].Round
		}
		return ctx.AI_GrenadeEvents[i].TickThrow < ctx.AI_GrenadeEvents[j].TickThrow
	})

	var groupedGrenades []models.AI_GrenadeRound
	if len(ctx.AI_GrenadeEvents) > 0 {
		currentRound := -1
		var currentEvents []models.AI_GrenadeEvent

		for _, event := range ctx.AI_GrenadeEvents {
			if event.Round != currentRound {
				if currentRound != -1 {
					total, smokes, flashes, hes, molotovs, decoys := countGrenadesByType(currentEvents)
					groupedGrenades = append(groupedGrenades, models.AI_GrenadeRound{
						Round:                   currentRound,
						GrenadesThrown:          total,
						SmokeThrown:             smokes,
						FlashbangThrown:         flashes,
						HEThrown:                hes,
						MolotovIncendiaryThrown: molotovs,
						DecoyThrown:             decoys,
						Events:                  currentEvents,
					})
				}
				currentRound = event.Round
				currentEvents = []models.AI_GrenadeEvent{event}
			} else {
				currentEvents = append(currentEvents, event)
			}
		}
		// Add last round
		if currentRound != -1 {
			total, smokes, flashes, hes, molotovs, decoys := countGrenadesByType(currentEvents)
			groupedGrenades = append(groupedGrenades, models.AI_GrenadeRound{
				Round:                   currentRound,
				GrenadesThrown:          total,
				SmokeThrown:             smokes,
				FlashbangThrown:         flashes,
				HEThrown:                hes,
				MolotovIncendiaryThrown: molotovs,
				DecoyThrown:             decoys,
				Events:                  currentEvents,
			})
		}
	}

	grandTotal, grandSmokes, grandFlashes, grandHEs, grandMolotovs, grandDecoys := countGrenadesByType(ctx.AI_GrenadeEvents)
	grenadesExport := models.AI_GrenadesExport{
		Totals: models.AI_GrenadeTotals{
			GrenadesThrown:          grandTotal,
			SmokeThrown:             grandSmokes,
			FlashbangThrown:         grandFlashes,
			HEThrown:                grandHEs,
			MolotovIncendiaryThrown: grandMolotovs,
			DecoyThrown:             grandDecoys,
		},
		Rounds: groupedGrenades,
	}

	if err := writeJSON(filepath.Join(matchDir, "grenades.json"), grenadesExport); err != nil {
		return err
	}

	// 6. Export Players Summary
	summaryExport := models.AI_PlayersSummaryExport{
		MatchID: matchID,
		Players: ctx.AI_PlayersSummary,
	}
	if err := writeJSON(filepath.Join(matchDir, "players_summary.json"), summaryExport); err != nil {
		return err
	}

	return nil
}

// writeJSON writes any data structure to a JSON file with indentation
func writeJSON(filepath string, data interface{}) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filepath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filepath, err)
	}

	return nil
}

// writeEconomyJSON writes economy data with compact array formatting
// Items in arrays are written one per line but inline
func writeEconomyJSON(filepath string, data interface{}) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Post-process to compact weapon item arrays
	content := string(bytes)
	content = compactWeaponArrays(content)

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filepath, err)
	}

	return nil
}

// compactWeaponArrays converts multi-line weapon objects to single lines
func compactWeaponArrays(jsonStr string) string {
	// This is a simple approach: find patterns like
	// {\n  "weapon": ...\n  "price": ...\n  "entity_id": ...\n  "original_owner": ...\n  }
	// and compact them to single lines

	result := strings.Builder{}
	lines := strings.Split(jsonStr, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Detect start of weapon object within an array
		if trimmed == "{" && i+4 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			// Check if it's a weapon item (first field is "weapon")
			if strings.HasPrefix(nextLine, "\"weapon\":") {
				// Collect all lines until closing brace
				objLines := []string{}
				indent := strings.TrimSuffix(line, "{")
				j := i + 1
				for j < len(lines) {
					objLine := strings.TrimSpace(lines[j])
					if objLine == "}," || objLine == "}" {
						// Build compact object
						compactObj := indent + "{"
						for k, ol := range objLines {
							compactObj += strings.TrimSpace(ol)
							if k < len(objLines)-1 {
								compactObj += " "
							}
						}
						if objLine == "}," {
							compactObj += "},"
						} else {
							compactObj += "}"
						}
						result.WriteString(compactObj + "\n")
						i = j + 1
						break
					}
					objLines = append(objLines, objLine)
					j++
				}
				continue
			}
		}

		result.WriteString(line + "\n")
		i++
	}

	// Remove trailing newline
	s := result.String()
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s
}

// buildTrackingExport groups tracking events by round and tick
func buildTrackingExport(ctx *models.DemoContext) models.AI_TrackingExport {
	// Group events by round, then by tick
	roundMap := make(map[int]map[int][]models.AI_TrackingEvent)

	for _, e := range ctx.AI_TrackingEventsWithRound {
		round := e.Round
		tick := e.Event.Tick

		if _, ok := roundMap[round]; !ok {
			roundMap[round] = make(map[int][]models.AI_TrackingEvent)
		}
		roundMap[round][tick] = append(roundMap[round][tick], e.Event)
	}

	// Build ordered structure
	var rounds []models.AI_TrackingRound

	// Get sorted round numbers
	roundNums := make([]int, 0, len(roundMap))
	for r := range roundMap {
		roundNums = append(roundNums, r)
	}
	sort.Ints(roundNums)

	for _, round := range roundNums {
		tickMap := roundMap[round]

		// Get sorted ticks
		tickNums := make([]int, 0, len(tickMap))
		for t := range tickMap {
			tickNums = append(tickNums, t)
		}
		sort.Ints(tickNums)

		var ticks []models.AI_TrackingTick
		for _, tick := range tickNums {
			ticks = append(ticks, models.AI_TrackingTick{
				Tick:    tick,
				Players: tickMap[tick],
			})
		}

		rounds = append(rounds, models.AI_TrackingRound{
			Round: round,
			Ticks: ticks,
		})
	}

	return models.AI_TrackingExport{
		Rounds: rounds,
	}
}
