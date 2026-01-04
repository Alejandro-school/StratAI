package handlers

import (
	"cs2-demo-service/models"
	"math"
	"strconv"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

type PlayerStatsHandler struct {
	stats map[uint64]*models.AI_PlayerStats

	// Round tracking
	currentRound  int
	roundKills    map[uint64]int
	roundDeaths   map[uint64]bool
	roundAssists  map[uint64]bool
	roundDamage   map[uint64]int
	roundSurvived map[uint64]bool
	roundTraded   map[uint64]bool // If player died and was traded

	// Side tracking (for CT/T stats)
	roundSide      map[uint64]common.Team // Player's side this round
	ctRoundDamage  map[uint64]int         // Damage dealt as CT
	tRoundDamage   map[uint64]int         // Damage dealt as T
	ctKills        map[uint64]int         // Kills as CT
	tKills         map[uint64]int         // Kills as T
	ctDeaths       map[uint64]int         // Deaths as CT
	tDeaths        map[uint64]int         // Deaths as T
	ctAssists      map[uint64]int         // Assists as CT
	tAssists       map[uint64]int         // Assists as T
	ctRoundsPlayed map[uint64]int         // Rounds played as CT
	tRoundsPlayed  map[uint64]int         // Rounds played as T
	ctKAST         map[uint64]float64     // KAST rounds as CT
	tKAST          map[uint64]float64     // KAST rounds as T

	// Entry tracking
	firstKillOccurred bool

	// Trade tracking (simplified, looking at recent deaths)
	recentDeaths []deathEvent

	// Multikill tracking for current round
	currentRoundKills map[uint64]int

	// Clutch tracking
	activeClutch *ClutchSituation // Current clutch situation (nil if none)
}

// ClutchSituation tracks a 1vX clutch in progress
type ClutchSituation struct {
	PlayerID       uint64
	PlayerTeam     common.Team
	EnemiesAtStart int // Number of enemies when clutch started
}

type deathEvent struct {
	victimID uint64
	killerID uint64
	tick     int
	team     common.Team
}

func NewPlayerStatsHandler() *PlayerStatsHandler {
	return &PlayerStatsHandler{
		stats:             make(map[uint64]*models.AI_PlayerStats),
		roundKills:        make(map[uint64]int),
		roundDeaths:       make(map[uint64]bool),
		roundAssists:      make(map[uint64]bool),
		roundDamage:       make(map[uint64]int),
		roundSurvived:     make(map[uint64]bool),
		roundTraded:       make(map[uint64]bool),
		recentDeaths:      make([]deathEvent, 0),
		roundSide:         make(map[uint64]common.Team),
		ctRoundDamage:     make(map[uint64]int),
		tRoundDamage:      make(map[uint64]int),
		ctKills:           make(map[uint64]int),
		tKills:            make(map[uint64]int),
		ctDeaths:          make(map[uint64]int),
		tDeaths:           make(map[uint64]int),
		ctAssists:         make(map[uint64]int),
		tAssists:          make(map[uint64]int),
		ctRoundsPlayed:    make(map[uint64]int),
		tRoundsPlayed:     make(map[uint64]int),
		ctKAST:            make(map[uint64]float64),
		tKAST:             make(map[uint64]float64),
		currentRoundKills: make(map[uint64]int),
	}
}

// RegisterPlayerStatsHandler registers the handler and returns it
func RegisterPlayerStatsHandler(ctx *models.DemoContext) *PlayerStatsHandler {
	h := NewPlayerStatsHandler()

	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) { h.HandleRoundStart(e, ctx) })
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) { h.HandleRoundEnd(e, ctx) })
	ctx.Parser.RegisterEventHandler(func(e events.Kill) { h.HandleKill(e, ctx) })
	ctx.Parser.RegisterEventHandler(h.HandleDamage)
	ctx.Parser.RegisterEventHandler(h.HandleWeaponFire)
	ctx.Parser.RegisterEventHandler(h.HandleGrenadeThrow)
	ctx.Parser.RegisterEventHandler(h.HandleBlind)

	return h
}

func (h *PlayerStatsHandler) GetStats() []models.AI_PlayerStats {
	var result []models.AI_PlayerStats
	for _, s := range h.stats {
		// Final calculations (Ratings, averages)
		h.calculateFinalStats(s)
		result = append(result, *s)
	}
	return result
}

// GetStatsWithContext returns stats with TTD and crosshair error calculated from combat duels
func (h *PlayerStatsHandler) GetStatsWithContext(ctx *models.DemoContext) []models.AI_PlayerStats {
	// First calculate TTD and crosshair error averages from combat data
	h.calculateCombatMetrics(ctx)

	// Then return the stats with final calculations
	return h.GetStats()
}

// calculateCombatMetrics aggregates time_to_damage and crosshair_placement from ReactionTimes
func (h *PlayerStatsHandler) calculateCombatMetrics(ctx *models.DemoContext) {
	// Iterate through all players in match data
	for steamID, playerData := range ctx.MatchData.Players {
		if playerData == nil {
			continue
		}

		// Check if this player exists in our stats
		playerStats, exists := h.stats[steamID]
		if !exists {
			continue
		}

		// Calculate averages from ReactionTimes
		var ttdSum float64
		var ttdCount int
		var crosshairSum float64
		var crosshairCount int

		for _, rt := range playerData.ReactionTimes {
			// Time to damage (only if > 0, meaning it was calculated)
			if rt.TimeToDamage > 0 {
				ttdSum += rt.TimeToDamage
				ttdCount++
			}

			// Crosshair placement error (only if > 0)
			if rt.CrosshairPlacementError > 0 {
				crosshairSum += rt.CrosshairPlacementError
				crosshairCount++
			}
		}

		// Apply averages to player stats
		if ttdCount > 0 {
			playerStats.TimeToDamageAvgMS = ttdSum / float64(ttdCount)
		}

		if crosshairCount > 0 {
			playerStats.CrosshairPlacementAvgError = crosshairSum / float64(crosshairCount)
		}
	}
}

func (h *PlayerStatsHandler) HandleRoundStart(e events.RoundStart, ctx *models.DemoContext) {
	h.currentRound++
	h.firstKillOccurred = false
	h.recentDeaths = make([]deathEvent, 0)

	// Reset round maps
	h.roundKills = make(map[uint64]int)
	h.roundDeaths = make(map[uint64]bool)
	h.roundAssists = make(map[uint64]bool)
	h.roundDamage = make(map[uint64]int)
	h.roundTraded = make(map[uint64]bool)
	h.currentRoundKills = make(map[uint64]int)

	// Track player sides at round start
	gs := ctx.Parser.GameState()
	if gs != nil {
		for _, p := range gs.Participants().Playing() {
			if p != nil {
				h.roundSide[p.SteamID64] = p.Team
				// Count rounds played per side
				if p.Team == common.TeamCounterTerrorists {
					h.ctRoundsPlayed[p.SteamID64]++
				} else if p.Team == common.TeamTerrorists {
					h.tRoundsPlayed[p.SteamID64]++
				}
			}
		}
	}
}

func (h *PlayerStatsHandler) HandleKill(e events.Kill, ctx *models.DemoContext) {
	if e.Killer != nil {
		s := h.getOrCreateStats(e.Killer)
		s.Kills++
		h.roundKills[e.Killer.SteamID64]++
		h.currentRoundKills[e.Killer.SteamID64]++

		// Side-specific kills
		if e.Killer.Team == common.TeamCounterTerrorists {
			h.ctKills[e.Killer.SteamID64]++
		} else if e.Killer.Team == common.TeamTerrorists {
			h.tKills[e.Killer.SteamID64]++
		}

		// Headshot
		if e.IsHeadshot {
			s.Headshots++
		}

		// Weapon stats - SKIP GRENADES
		if e.Weapon != nil && !h.isGrenade(e.Weapon) {
			wName := e.Weapon.String()
			ws := s.WeaponStats[wName]
			ws.Kills++
			if e.IsHeadshot {
				ws.Headshots++
			}
			s.WeaponStats[wName] = ws
		}

		// Entry / Opening Kill
		if !h.firstKillOccurred {
			s.OpeningDuelsAttempted++
			s.OpeningDuelsWon++
			h.firstKillOccurred = true

			// Mark opening duel lost for victim
			if e.Victim != nil {
				v := h.getOrCreateStats(e.Victim)
				v.OpeningDuelsAttempted++
				v.OpeningDuelsLost++
			}
		}

		// Trade Kill Logic
		for _, d := range h.recentDeaths {
			if e.Victim != nil && d.killerID == e.Victim.SteamID64 {
				s.TradeKills++
				h.roundTraded[d.victimID] = true
				if tStats, exists := h.stats[d.victimID]; exists {
					tStats.TradedDeaths++
				}
				break // Only count one trade per kill
			}
		}
	}

	if e.Victim != nil {
		s := h.getOrCreateStats(e.Victim)
		s.Deaths++
		h.roundDeaths[e.Victim.SteamID64] = true

		// Side-specific deaths
		if e.Victim.Team == common.TeamCounterTerrorists {
			h.ctDeaths[e.Victim.SteamID64]++
		} else if e.Victim.Team == common.TeamTerrorists {
			h.tDeaths[e.Victim.SteamID64]++
		}

		// Record death for potential trade
		h.recentDeaths = append(h.recentDeaths, deathEvent{
			victimID: e.Victim.SteamID64,
			killerID: 0,
		})
		if e.Killer != nil {
			h.recentDeaths[len(h.recentDeaths)-1].killerID = e.Killer.SteamID64
		}

		// CLUTCH DETECTION: Check if someone is now in a 1vX situation
		// Only check if there's no active clutch already
		if h.activeClutch == nil {
			h.checkForClutchSituation(ctx)
		}
	}

	if e.Assister != nil {
		s := h.getOrCreateStats(e.Assister)
		s.Assists++
		h.roundAssists[e.Assister.SteamID64] = true

		// Side-specific assists
		if e.Assister.Team == common.TeamCounterTerrorists {
			h.ctAssists[e.Assister.SteamID64]++
		} else if e.Assister.Team == common.TeamTerrorists {
			h.tAssists[e.Assister.SteamID64]++
		}

		// Flash assist
		if e.AssistedFlash {
			s.FlashAssists++
		}
	}
}

// checkForClutchSituation checks if any player is now in a 1vX situation
func (h *PlayerStatsHandler) checkForClutchSituation(ctx *models.DemoContext) {
	gs := ctx.Parser.GameState()
	if gs == nil {
		return
	}

	// Count alive players per team
	var aliveCT, aliveT []uint64
	for _, p := range gs.Participants().Playing() {
		if p == nil || !p.IsAlive() {
			continue
		}
		if p.Team == common.TeamCounterTerrorists {
			aliveCT = append(aliveCT, p.SteamID64)
		} else if p.Team == common.TeamTerrorists {
			aliveT = append(aliveT, p.SteamID64)
		}
	}

	// Check for CT clutch (1 CT vs X Ts)
	if len(aliveCT) == 1 && len(aliveT) >= 1 {
		clutcherID := aliveCT[0]
		enemyCount := len(aliveT)

		// Register clutch attempt
		h.activeClutch = &ClutchSituation{
			PlayerID:       clutcherID,
			PlayerTeam:     common.TeamCounterTerrorists,
			EnemiesAtStart: enemyCount,
		}

		// Mark as attempted
		if playerStats, exists := h.stats[clutcherID]; exists {
			switch enemyCount {
			case 1:
				playerStats.Clutches1v1Attempted++
			case 2:
				playerStats.Clutches1v2Attempted++
			case 3:
				playerStats.Clutches1v3Attempted++
			case 4:
				playerStats.Clutches1v4Attempted++
			case 5:
				playerStats.Clutches1v5Attempted++
			}
		}
	}

	// Check for T clutch (1 T vs X CTs)
	if len(aliveT) == 1 && len(aliveCT) >= 1 {
		clutcherID := aliveT[0]
		enemyCount := len(aliveCT)

		// Register clutch attempt
		h.activeClutch = &ClutchSituation{
			PlayerID:       clutcherID,
			PlayerTeam:     common.TeamTerrorists,
			EnemiesAtStart: enemyCount,
		}

		// Mark as attempted
		if playerStats, exists := h.stats[clutcherID]; exists {
			switch enemyCount {
			case 1:
				playerStats.Clutches1v1Attempted++
			case 2:
				playerStats.Clutches1v2Attempted++
			case 3:
				playerStats.Clutches1v3Attempted++
			case 4:
				playerStats.Clutches1v4Attempted++
			case 5:
				playerStats.Clutches1v5Attempted++
			}
		}
	}
}

func (h *PlayerStatsHandler) HandleDamage(e events.PlayerHurt) {
	if e.Attacker != nil && e.Player != nil && e.Attacker.SteamID64 != e.Player.SteamID64 {
		s := h.getOrCreateStats(e.Attacker)
		damage := e.HealthDamage
		if damage > 100 {
			damage = 100
		}

		s.TotalDamage += damage
		h.roundDamage[e.Attacker.SteamID64] += damage

		// Side-specific damage
		if e.Attacker.Team == common.TeamCounterTerrorists {
			h.ctRoundDamage[e.Attacker.SteamID64] += damage
		} else if e.Attacker.Team == common.TeamTerrorists {
			h.tRoundDamage[e.Attacker.SteamID64] += damage
		}

		// Weapon stats - SKIP GRENADES for weapon_stats
		if e.Weapon != nil && !h.isGrenade(e.Weapon) {
			wName := e.Weapon.String()
			ws := s.WeaponStats[wName]
			ws.Damage += damage
			ws.ShotsHit++
			s.WeaponStats[wName] = ws

			// Track overall shots hit
			s.ShotsHit++
		}

		// Grenade damage - tracked separately
		if h.isGrenade(e.Weapon) {
			wName := h.normalizeGrenadeName(e.Weapon.String())
			s.GrenadeDamage[wName] += damage
			s.UtilityDamage += damage
		}

		// Body part hits (on the attacker's record)
		group := hitgroupToString(events.HitGroup(e.HitGroup))
		s.BodyPartHits[group]++
	}
}

func (h *PlayerStatsHandler) HandleWeaponFire(e events.WeaponFire) {
	if e.Shooter != nil && e.Weapon != nil {
		s := h.getOrCreateStats(e.Shooter)

		// SKIP GRENADES for weapon fire tracking in weapon_stats
		if !h.isGrenade(e.Weapon) {
			s.ShotsFired++

			wName := e.Weapon.String()
			ws := s.WeaponStats[wName]
			ws.ShotsFired++
			s.WeaponStats[wName] = ws
		}
	}
}

func (h *PlayerStatsHandler) HandleGrenadeThrow(e events.GrenadeProjectileThrow) {
	if e.Projectile != nil && e.Projectile.Thrower != nil {
		s := h.getOrCreateStats(e.Projectile.Thrower)
		s.GrenadesThrownTotal++

		wName := e.Projectile.WeaponInstance.String()
		// Normalize
		if wName == "Flashbang" {
			s.FlashesThrown++
		} else if wName == "Smoke Grenade" {
			s.SmokesThrown++
		} else if wName == "HE Grenade" {
			s.HEThrown++
		} else if wName == "Molotov" || wName == "Incendiary Grenade" {
			s.MolotovsThrown++
		}
	}
}

func (h *PlayerStatsHandler) HandleBlind(e events.PlayerFlashed) {
	if e.Attacker != nil && e.Player != nil && e.FlashDuration().Seconds() > 0 {
		// Attacker blinded someone
		s := h.getOrCreateStats(e.Attacker)

		isEnemy := e.Attacker.Team != e.Player.Team
		if isEnemy {
			s.EnemiesFlashedTotal++
			s.FlashDurationTotal += e.FlashDuration().Seconds()
		}
	}
}

// Helpers
func (h *PlayerStatsHandler) isGrenade(w *common.Equipment) bool {
	if w == nil {
		return false
	}
	return w.Type == common.EqHE ||
		w.Type == common.EqMolotov ||
		w.Type == common.EqIncendiary ||
		w.Type == common.EqFlash ||
		w.Type == common.EqSmoke ||
		w.Type == common.EqDecoy
}

func (h *PlayerStatsHandler) normalizeGrenadeName(name string) string {
	switch name {
	case "Incendiary Grenade", "Molotov":
		return "molotov"
	case "HE Grenade":
		return "he"
	case "Flashbang":
		return "flash"
	case "Smoke Grenade":
		return "smoke"
	case "Decoy Grenade":
		return "decoy"
	default:
		return "other"
	}
}

func (h *PlayerStatsHandler) HandleRoundEnd(e events.RoundEnd, ctx *models.DemoContext) {
	// CLUTCH: Check if the clutcher won the round
	if h.activeClutch != nil {
		clutchWon := false

		// Determine if clutch was won based on round winner
		if e.Winner == h.activeClutch.PlayerTeam {
			clutchWon = true
		}

		// Record clutch result
		if playerStats, exists := h.stats[h.activeClutch.PlayerID]; exists && clutchWon {
			switch h.activeClutch.EnemiesAtStart {
			case 1:
				playerStats.Clutches1v1Won++
			case 2:
				playerStats.Clutches1v2Won++
			case 3:
				playerStats.Clutches1v3Won++
			case 4:
				playerStats.Clutches1v4Won++
			case 5:
				playerStats.Clutches1v5Won++
			}
		}

		// Reset clutch tracking for next round
		h.activeClutch = nil
	}

	// Calculate KAST for this round
	for steamID, playerStats := range h.stats {
		hasKill := h.roundKills[steamID] > 0
		hasAssist := h.roundAssists[steamID]
		survived := !h.roundDeaths[steamID]
		traded := h.roundTraded[steamID]

		if hasKill || hasAssist || survived || traded {
			playerStats.KAST += 1.0

			// Side-specific KAST
			if h.roundSide[steamID] == common.TeamCounterTerrorists {
				h.ctKAST[steamID]++
			} else if h.roundSide[steamID] == common.TeamTerrorists {
				h.tKAST[steamID]++
			}
		}

		if survived {
			playerStats.RoundsSurvived++
		}

		// Count multikills for this round
		kills := h.currentRoundKills[steamID]
		if kills >= 2 {
			if playerStats.MultiKills == nil {
				playerStats.MultiKills = make(map[string]int)
			}
			switch kills {
			case 2:
				playerStats.MultiKills["2k"]++
			case 3:
				playerStats.MultiKills["3k"]++
			case 4:
				playerStats.MultiKills["4k"]++
			case 5:
				playerStats.MultiKills["ace"]++
			}
		}
	}
}

func (h *PlayerStatsHandler) getOrCreateStats(player *common.Player) *models.AI_PlayerStats {
	if player == nil {
		return nil
	}

	id := player.SteamID64
	if _, exists := h.stats[id]; !exists {
		team := "Mixed"
		if player.Team == common.TeamCounterTerrorists {
			team = "CT"
		} else if player.Team == common.TeamTerrorists {
			team = "T"
		}

		h.stats[id] = &models.AI_PlayerStats{
			SteamID:              strconv.FormatUint(player.SteamID64, 10),
			Name:                 player.Name,
			Team:                 team,
			MultiKills:           make(map[string]int),
			GrenadeDamage:        make(map[string]int),
			BodyPartHits:         make(map[string]int),
			UtilityUsagePerRound: make(map[string]float64),
			WeaponStats:          make(map[string]models.AI_WeaponStat),
		}
	}
	return h.stats[id]
}

func (h *PlayerStatsHandler) calculateFinalStats(s *models.AI_PlayerStats) {
	rounds := float64(h.currentRound)
	if rounds == 0 {
		return
	}

	steamID, _ := strconv.ParseUint(s.SteamID, 10, 64)

	// Set Team based on the LAST ROUND (roundSide tracks the team at each round start)
	// This is the team the player was on when the match ended
	if lastTeam, exists := h.roundSide[steamID]; exists {
		if lastTeam == common.TeamCounterTerrorists {
			s.Team = "CT"
		} else if lastTeam == common.TeamTerrorists {
			s.Team = "T"
		}
	}

	// Basic stats
	s.ADR = float64(s.TotalDamage) / rounds
	s.KDRatio = float64(s.Kills) / math.Max(1.0, float64(s.Deaths))
	if s.Kills > 0 {
		s.HSPercentage = (float64(s.Headshots) / float64(s.Kills)) * 100.0
	}

	// KAST percentage
	s.KAST = (s.KAST / rounds) * 100.0

	// Opening duels success rate
	if s.OpeningDuelsAttempted > 0 {
		s.OpeningSuccessRate = (float64(s.OpeningDuelsWon) / float64(s.OpeningDuelsAttempted)) * 100.0
	}

	// Utility efficiency
	if s.FlashesThrown > 0 {
		s.EnemiesFlashedPerFlash = float64(s.EnemiesFlashedTotal) / float64(s.FlashesThrown)
		s.BlindTimePerFlash = s.FlashDurationTotal / float64(s.FlashesThrown)
	}

	if s.HEThrown > 0 {
		heDmg := s.GrenadeDamage["he"]
		s.HEDamagePerNade = float64(heDmg) / float64(s.HEThrown)
	}

	if s.MolotovsThrown > 0 {
		moloDmg := s.GrenadeDamage["molotov"]
		s.MolotovDamagePerNade = float64(moloDmg) / float64(s.MolotovsThrown)
	}

	// Accuracy
	if s.ShotsFired > 0 {
		s.AccuracyOverall = (float64(s.ShotsHit) / float64(s.ShotsFired)) * 100.0
	}

	// DamagePerRound field - this is proper ADR
	s.DamagePerRound = s.ADR

	// Calculate accuracy for each weapon
	for wName, ws := range s.WeaponStats {
		if ws.ShotsFired > 0 {
			ws.Accuracy = (float64(ws.ShotsHit) / float64(ws.ShotsFired)) * 100.0
			s.WeaponStats[wName] = ws
		}
	}

	// CT/T specific stats (use float64 for rating calculations)
	ctRoundsF := float64(h.ctRoundsPlayed[steamID])
	tRoundsF := float64(h.tRoundsPlayed[steamID])

	// CT ADR
	if ctRoundsF > 0 {
		s.CTADR = float64(h.ctRoundDamage[steamID]) / ctRoundsF
	}

	// T ADR
	if tRoundsF > 0 {
		s.TADR = float64(h.tRoundDamage[steamID]) / tRoundsF
	}

	// Calculate CT Rating
	if ctRoundsF > 0 {
		ctKPR := float64(h.ctKills[steamID]) / ctRoundsF
		ctDPR := float64(h.ctDeaths[steamID]) / ctRoundsF
		ctKAST := (h.ctKAST[steamID] / ctRoundsF) * 100.0
		ctAssistPerRound := float64(h.ctAssists[steamID]) / ctRoundsF
		ctImpact := 2.13*ctKPR + 0.42*ctAssistPerRound - 0.41
		if ctImpact < 0 {
			ctImpact = 0
		}
		s.CTRating = 0.0073*ctKAST + 0.3591*ctKPR - 0.5329*ctDPR + 0.2372*ctImpact + 0.0032*s.CTADR + 0.1587
		if s.CTRating < 0 {
			s.CTRating = 0
		}
	}

	// Calculate T Rating
	if tRoundsF > 0 {
		tKPR := float64(h.tKills[steamID]) / tRoundsF
		tDPR := float64(h.tDeaths[steamID]) / tRoundsF
		tKAST := (h.tKAST[steamID] / tRoundsF) * 100.0
		tAssistPerRound := float64(h.tAssists[steamID]) / tRoundsF
		tImpact := 2.13*tKPR + 0.42*tAssistPerRound - 0.41
		if tImpact < 0 {
			tImpact = 0
		}
		s.TRating = 0.0073*tKAST + 0.3591*tKPR - 0.5329*tDPR + 0.2372*tImpact + 0.0032*s.TADR + 0.1587
		if s.TRating < 0 {
			s.TRating = 0
		}
	}

	// HLTV 2.0 Rating (overall)
	kpr := float64(s.Kills) / rounds
	dpr := float64(s.Deaths) / rounds

	// Impact Rating (approx): 2.13*KPR + 0.42*Assist% - 0.41
	assistPerRound := float64(s.Assists) / rounds
	s.ImpactRating = 2.13*kpr + 0.42*assistPerRound - 0.41
	if s.ImpactRating < 0 {
		s.ImpactRating = 0
	}

	// HLTV 2.0: 0.0073*KAST + 0.3591*KPR - 0.5329*DPR + 0.2372*Impact + 0.0032*ADR + 0.1587
	s.HLTVRating = 0.0073*s.KAST + 0.3591*kpr - 0.5329*dpr + 0.2372*s.ImpactRating + 0.0032*s.ADR + 0.1587
	if s.HLTVRating < 0 {
		s.HLTVRating = 0
	}
}
