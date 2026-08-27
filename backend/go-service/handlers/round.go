package handlers

import (
	"cs2-demo-service/models"
	"fmt"

	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

// RegisterRoundHandlers registra handlers para eventos de ronda
func RegisterRoundHandlers(ctx *models.DemoContext) {
	// Round start
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		gs := ctx.Parser.GameState()
		if gs.IsWarmupPeriod() {
			return
		}
		roundNumber := gs.TotalRoundsPlayed() + 1

		ctx.InRound = true
		ensureObjectiveTracker(ctx).BeginRound(roundNumber, gs.IngameTick())
		ctx.LastActiveWeapon = make(map[uint64]models.ActiveWeaponObservation, 16)
		ctx.PlayerMotion.Reset()

		// Clear completed grenade trajectories from previous round
		ctx.CompletedGrenadeTrajectories = make(map[int]*models.GrenadeTrajectoryEvent)

		// Clear reaction time tracking
		ctx.EnemyFirstSeenTick = make(map[uint64]map[uint64]models.FirstSeenData)
		ctx.LastVisibleEnemies = make(map[uint64]map[uint64]bool)
		ctx.FirstDamageTick = make(map[uint64]map[uint64]int) // Clear damage tracking

		// Clear round damage tracking
		ctx.RoundDamage = make(map[uint64]map[uint64]int)

		// FIX Bug 3: Clear HP tracking for new round
		ctx.LastKnownHealth = make(map[uint64]int)

		ensureMatchRound(ctx.MatchData, roundNumber)
	})

	// Freeze time end
	ctx.Parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		// Marcar inicio de ronda activa
		gs := ctx.Parser.GameState()
		if !objectiveRoundActive(ctx) {
			return
		}
		ctx.InRound = true
		roundNumber := currentObjectiveRound(ctx)
		ensureObjectiveTracker(ctx).NativeSnapshot(
			nativeObjectiveObservation(ctx, roundNumber, gs.IngameTick()),
		)

		// FIX: Capture all players' active weapons BEFORE combat starts
		// This ensures victims who don't deal damage have their weapon tracked
		for _, player := range ctx.Parser.GameState().Participants().Playing() {
			if player == nil || !player.IsAlive() {
				continue
			}
			sid := player.SteamID64
			if sid == 0 {
				continue
			}
			cacheActiveWeapon(ctx, player, ctx.Parser.GameState().IngameTick())
		}
	})

	// Round end
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) {
		gs := ctx.Parser.GameState()
		roundNumber := gs.TotalRoundsPlayed()
		tracker := ensureObjectiveTracker(ctx)
		tracker.EndRound(roundNumber, gs.IngameTick())
		objectiveSummary, ok := tracker.RoundSummary(roundNumber)
		if !ok {
			objectiveSummary.Round = roundNumber
		}
		applyMatchRoundResult(
			ctx.MatchData,
			roundNumber,
			roundWinnerLabel(e.Winner),
			fmt.Sprintf("%v", e.Reason),
			gs.TeamCounterTerrorists().Score(),
			gs.TeamTerrorists().Score(),
			objectiveSummary.WasPlanted,
			objectiveSummary.Site,
			objectiveSummary.PlantTick,
		)

		// NEW: Consolidate raw combat events into duels
		ConsolidateDuels(ctx)
	})
}

func ensureMatchRound(match *models.MatchData, roundNumber int) {
	for _, round := range match.Rounds {
		if round.Round == roundNumber {
			return
		}
	}
	match.Rounds = append(match.Rounds, models.RoundData{Round: roundNumber})
}

func roundWinnerLabel(winner common.Team) string {
	switch winner {
	case common.TeamTerrorists:
		return "T"
	case common.TeamCounterTerrorists:
		return "CT"
	default:
		return ""
	}
}

func applyMatchRoundResult(
	match *models.MatchData,
	roundNumber int,
	winner, reason string,
	ctScore, tScore int,
	bombPlanted bool,
	bombSite string,
	bombTick int,
) {
	for index := range match.Rounds {
		if match.Rounds[index].Round != roundNumber {
			continue
		}
		round := &match.Rounds[index]
		round.Winner = winner
		round.Reason = reason
		round.CTScore = ctScore
		round.TScore = tScore
		if bombPlanted {
			round.BombPlanted = true
			round.BombSite = bombSite
			round.BombTick = bombTick
		}
		return
	}
}

// captureSurvivors captura el estado de todos los jugadores al final de la ronda
// registrando qué equipo sobrevivió (para cálculos económicos futuros)
func captureSurvivors(ctx *models.DemoContext) {
	if len(ctx.RoundTimelines) == 0 {
		return
	}

	currentTimeline := &ctx.RoundTimelines[len(ctx.RoundTimelines)-1]
	gs := ctx.Parser.GameState()

	var survivors []models.PlayerSurvivalSnapshot

	// Iterar sobre todos los jugadores
	for _, player := range gs.Participants().Playing() {
		if player == nil {
			continue
		}

		teamName := "Unknown"
		switch player.Team {
		case common.TeamTerrorists:
			teamName = "T"
		case common.TeamCounterTerrorists:
			teamName = "CT"
		}

		// Capturar inventario actual
		inventory := []string{}
		for _, weapon := range player.Weapons() {
			if weapon != nil && weapon.String() != "" {
				inventory = append(inventory, weapon.String())
			}
		}

		// Agregar equipo defensivo si existe
		if player.HasDefuseKit() {
			inventory = append(inventory, "Defuse Kit")
		}
		if player.Armor() > 0 {
			if player.HasHelmet() {
				inventory = append(inventory, "Kevlar + Helmet")
			} else {
				inventory = append(inventory, "Kevlar Vest")
			}
		}

		// Usar EquipmentValueCurrent() de la librería
		equipValue := player.EquipmentValueCurrent()

		snapshot := models.PlayerSurvivalSnapshot{
			SteamID:                player.SteamID64,
			Name:                   player.Name,
			Team:                   teamName,
			Survived:               player.IsAlive(),
			EndRoundItems:          inventory,
			EquipmentValueSurvived: 0, // Solo cuenta si sobrevivió
		}

		// Si sobrevivió, el equipo se guarda
		if player.IsAlive() {
			snapshot.EquipmentValueSurvived = equipValue
		}

		survivors = append(survivors, snapshot)
	}

	// Buscar el último evento round_end y añadirle survivors
	// (el evento ya fue creado por timeline.go, solo lo modificamos)
	for i := len(currentTimeline.Events) - 1; i >= 0; i-- {
		if currentTimeline.Events[i].Type == "round_end" && currentTimeline.Events[i].RoundEnd != nil {
			currentTimeline.Events[i].RoundEnd.Survivors = survivors
			break
		}
	}
}
