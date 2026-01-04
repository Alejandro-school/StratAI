package handlers

import (
	"cs2-demo-service/models"
	"fmt"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterRoundHandlers registra handlers para eventos de ronda
func RegisterRoundHandlers(ctx *models.DemoContext) {
	// Round start
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		// ctx.CurrentRound is managed by timeline handlers now (based on GameState)
		// ctx.CurrentRound++
		ctx.InRound = true
		ctx.BombPlanted = false
		ctx.BombSite = ""
		ctx.BombTick = 0

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

		roundData := models.RoundData{
			Round:   ctx.CurrentRound,
			Winner:  "",
			Reason:  "",
			CTScore: 0,
			TScore:  0,
		}

		ctx.MatchData.Rounds = append(ctx.MatchData.Rounds, roundData)
	})

	// Freeze time end
	ctx.Parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		// Marcar inicio de ronda activa
		ctx.InRound = true

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
			if activeWeapon := player.ActiveWeapon(); activeWeapon != nil {
				ctx.LastActiveWeapon[sid] = activeWeapon.String()
			}
		}
	})

	// Round end
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) {
		ctx.InRound = false

		if len(ctx.MatchData.Rounds) > 0 {
			lastRound := &ctx.MatchData.Rounds[len(ctx.MatchData.Rounds)-1]

			// Winner usando enum nativo
			switch e.Winner {
			case common.TeamTerrorists:
				lastRound.Winner = "T"
			case common.TeamCounterTerrorists:
				lastRound.Winner = "CT"
			}

			// Reason usando enum nativo
			lastRound.Reason = fmt.Sprintf("%v", e.Reason)

			// Scores actuales
			gs := ctx.Parser.GameState()
			lastRound.CTScore = gs.TeamCounterTerrorists().Score()
			lastRound.TScore = gs.TeamTerrorists().Score()

			// Bomba plantada en esta ronda
			if ctx.BombPlanted {
				lastRound.BombPlanted = true
				lastRound.BombSite = ctx.BombSite
				lastRound.BombTick = ctx.BombTick
			}
		}

		// NEW: Consolidate raw combat events into duels
		ConsolidateDuels(ctx)

		// Capturar supervivientes y su equipo al final de la ronda
		captureSurvivors(ctx)
	})
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
