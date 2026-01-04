package handlers

import (
	"cs2-demo-service/models"
	"fmt"
	"log"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

const (
	GAME_STATE_SAMPLE_INTERVAL = 128 // 1 segundo a 128 tick rate
)

// Items de equipamiento inicial que NO deben registrarse como compras
var initialEquipment = map[string]bool{
	"Knife":    true,
	"USP-S":    true,
	"P2000":    true,
	"Glock-18": true,
	"C4":       true,
}

// RegisterTimelineHandlers registra todos los handlers para el sistema de timeline
func RegisterTimelineHandlers(ctx *models.DemoContext) {
	// GameState sampling cada 128 ticks (SOLO después de freeze time)
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		currentTick := ctx.Parser.GameState().IngameTick()

		// Update PreviousPlayerPosition for velocity calculation (every frame)
		for _, p := range ctx.Parser.GameState().Participants().Playing() {
			ctx.PreviousPlayerPosition[p.SteamID64] = p.Position()
		}

		// FLUSH PENDING COMBAT EVENTS (Buffer logic)
		// If an event is older than 3 seconds (3 * 128 = 384 ticks), commit it as a non-fatal duel
		// This handles the case where damage happens but no kill follows immediately
		tickRate := ctx.Parser.TickRate()
		if tickRate == 0 {
			tickRate = 128
		}
		timeoutTicks := int(3.0 * tickRate)

		activePending := []models.AI_CombatDuel{}
		for _, pending := range ctx.PendingCombatEvents {
			if currentTick-pending.Tick > timeoutTicks {
				// Timeout reached: Commit to main list
				ctx.AI_CombatDuels = append(ctx.AI_CombatDuels, pending)
			} else {
				// Keep waiting
				activePending = append(activePending, pending)
			}
		}
		ctx.PendingCombatEvents = activePending

		// Sample cada 128 ticks, solo durante ronda activa Y después de freeze time
		if ctx.InRound && ctx.FreezeTimeEnded && (currentTick-ctx.LastGameStateTick >= GAME_STATE_SAMPLE_INTERVAL) {
			captureGameState(ctx, currentTick)
			ctx.LastGameStateTick = currentTick
		}

		if currentTick%10000 == 0 {
			log.Printf("[DEBUG] Tick: %d\n", currentTick)
		}
	})

	// Round start - crear evento + resetear tracking
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		gs := ctx.Parser.GameState()
		roundNum := gs.TotalRoundsPlayed()

		// SKIP warmup
		if gs.IsWarmupPeriod() {
			log.Printf("[DEBUG] Skipping RoundStart (Warmup)\n")
			return
		}

		// Calculate actual round number (1-based)
		currentRound := roundNum + 1

		log.Printf("[DEBUG] RoundStart: %d (TotalPlayed: %d)\n", currentRound, roundNum)

		// Update ActualRoundNumber in context for other handlers
		ctx.ActualRoundNumber = currentRound

		// SKIP si ya procesamos esta ronda
		if currentRound == ctx.CurrentRound {
			return
		}

		ctx.InRound = true
		ctx.CurrentRound = currentRound
		ctx.CurrentRoundEvents = []models.TimelineEvent{} // Reset eventos de ronda
		ctx.FreezeTimeEnded = false                       // Reset freeze time flag
		ctx.PlayerMoneyBefore = make(map[uint64]int)      // Reset money tracking
		ctx.RoundDamage = make(map[uint64]map[uint64]int) // Reset damage tracking
		ctx.LastKnownHealth = make(map[uint64]int)        // FIX Bug 3: Reset HP tracking

		// Calcular dinero inicial de equipos Y guardar dinero individual
		ctMoney, tMoney := 0, 0
		playerStates := []models.PlayerEconomyState{}

		for _, player := range gs.Participants().Playing() {
			if player.Team == common.TeamCounterTerrorists {
				ctMoney += player.Money()
			} else if player.Team == common.TeamTerrorists {
				tMoney += player.Money()
			}
			// Guardar dinero inicial para calcular cost de compras
			ctx.PlayerMoneyBefore[player.SteamID64] = player.Money()

			// Capturar inventario inicial
			inventory := []string{}
			for _, weapon := range player.Weapons() {
				inventory = append(inventory, weapon.String())
			}

			teamStr := "Spectator"
			if player.Team == common.TeamCounterTerrorists {
				teamStr = "CT"
			} else if player.Team == common.TeamTerrorists {
				teamStr = "T"
			}

			playerStates = append(playerStates, models.PlayerEconomyState{
				SteamID:   player.SteamID64,
				Name:      player.Name,
				Team:      teamStr,
				AreaName:  player.LastPlaceName(), // Spawn area at round start
				Money:     player.Money(),
				Inventory: inventory,
			})
		}

		event := models.TimelineEvent{
			Type:  "round_start",
			Tick:  gs.IngameTick(),
			Round: ctx.CurrentRound,
			RoundStart: &models.RoundStartEvent{
				RoundNumber:  ctx.CurrentRound,
				CTStartMoney: ctMoney,
				TStartMoney:  tMoney,
				CTScore:      gs.TeamCounterTerrorists().Score(),
				TScore:       gs.TeamTerrorists().Score(),
				CTLossBonus:  calculateLossBonus(ctx.CTConsecutiveLosses),
				TLossBonus:   calculateLossBonus(ctx.TConsecutiveLosses),
				Players:      playerStates,
			},
		}

		ctx.CurrentRoundEvents = append(ctx.CurrentRoundEvents, event)
		ctx.Timeline = append(ctx.Timeline, event)

		// Reset bomb tracking
		ctx.BombPlanted = false
		ctx.BombSite = ""
	})

	// Freeze time end - habilitar game state sampling
	ctx.Parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		ctx.FreezeTimeEnded = true
		ctx.FreezeTimeEndTick = ctx.Parser.GameState().IngameTick() // Track round action start
	})

	// Round end - crear evento + guardar timeline de ronda
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) {
		gs := ctx.Parser.GameState()
		roundNum := gs.TotalRoundsPlayed()
		log.Printf("[DEBUG] Event RoundEnd. RoundNum: %d, Reason: %v, Winner: %v\n", roundNum, e.Reason, e.Winner)

		// SKIP warmup
		if roundNum == 0 {
			log.Printf("[DEBUG] Skipping RoundEnd for round 0\n")
			return
		}

		log.Printf("[DEBUG] RoundEnd: %d\n", roundNum)

		// SKIP si no estamos en ronda o si ya guardamos esta ronda
		if !ctx.InRound || roundNum != ctx.CurrentRound {
			return
		}

		winner := "Unknown"
		if e.Winner == common.TeamCounterTerrorists {
			winner = "CT"
			ctx.CTRoundsWon++
			ctx.CTConsecutiveLosses = 0
			ctx.TConsecutiveLosses++
		} else if e.Winner == common.TeamTerrorists {
			winner = "T"
			ctx.TRoundsWon++
			ctx.TConsecutiveLosses = 0
			ctx.CTConsecutiveLosses++
		} else {
			// Draw or other? Usually doesn't reset loss bonus unless specified
			// But in CS, draw usually means no one wins, but loss bonus might increase for both?
			// Or just keep as is.
		}

		event := models.TimelineEvent{
			Type:  "round_end",
			Tick:  gs.IngameTick(),
			Round: ctx.CurrentRound,
			RoundEnd: &models.RoundEndEvent{
				RoundNumber: ctx.CurrentRound,
				Winner:      winner,
				Reason:      fmt.Sprintf("%v", e.Reason),
				CTScore:     gs.TeamCounterTerrorists().Score(),
				TScore:      gs.TeamTerrorists().Score(),
			},
		}

		ctx.CurrentRoundEvents = append(ctx.CurrentRoundEvents, event)
		ctx.Timeline = append(ctx.Timeline, event)

		// Guardar timeline de esta ronda para export
		if len(ctx.CurrentRoundEvents) > 0 {
			roundTimeline := models.RoundTimeline{
				RoundNumber: ctx.CurrentRound,
				Events:      ctx.CurrentRoundEvents,
				StartTick:   ctx.CurrentRoundEvents[0].Tick,
				EndTick:     gs.IngameTick(),
			}
			ctx.RoundTimelines = append(ctx.RoundTimelines, roundTimeline)
		}

		ctx.InRound = false
	})

	// Buy events - capturar compras usando ItemPickup (más fiable que ItemEquip)
	ctx.Parser.RegisterEventHandler(func(e events.ItemPickup) {
		if !ctx.InRound || e.Player == nil || e.Weapon == nil {
			return
		}

		itemName := e.Weapon.String()

		// FILTRO 1: Ignorar equipamiento inicial (cuchillos, pistolas default)
		// Aunque ItemPickup no suele saltar para items iniciales, es bueno prevenir
		if initialEquipment[itemName] {
			return
		}

		gs := ctx.Parser.GameState()

		// Calcular cost real por diferencia de dinero
		moneyBefore := ctx.PlayerMoneyBefore[e.Player.SteamID64]
		moneyNow := e.Player.Money()
		cost := moneyBefore - moneyNow

		// FILTRO 2: Solo registrar si hubo cambio de dinero (compra real, no pickup del suelo)
		if cost <= 0 {
			return
		}

		// Actualizar dinero previo para próxima compra
		ctx.PlayerMoneyBefore[e.Player.SteamID64] = moneyNow

		event := models.TimelineEvent{
			Type:  "buy",
			Tick:  gs.IngameTick(),
			Round: ctx.CurrentRound,
			Buy: &models.BuyEvent{
				Player:    e.Player.Name,
				SteamID:   e.Player.SteamID64,
				Item:      itemName,
				Cost:      cost,
				MoneyLeft: moneyNow,
			},
		}

		ctx.CurrentRoundEvents = append(ctx.CurrentRoundEvents, event)
		ctx.Timeline = append(ctx.Timeline, event)
	})

	// Refund detection - Check money increase in FrameDone (during buy time)
	// We reuse the existing FrameDone handler logic by adding a check here
	// But since we can't easily merge with the top FrameDone, we add a new one specific for refunds
	ctx.Parser.RegisterEventHandler(func(e events.FrameDone) {
		if !ctx.InRound {
			return
		}

		gs := ctx.Parser.GameState()
		// Check refunds only during buy time (approximate check or if IsBuyPeriod is available)
		// IsBuyPeriod is not directly available on GameState in all versions, but we can check freeze time or time
		// Actually, refunds can happen as long as buy zone is active.
		// We'll check for money INCREASE without kills.

		for _, player := range gs.Participants().Playing() {
			moneyBefore := ctx.PlayerMoneyBefore[player.SteamID64]
			moneyNow := player.Money()

			if moneyNow > moneyBefore {
				// Money increased. Check if it's a refund.
				// Exclude: Kill rewards (usually happen on death event, but money updates instantly)
				// Exclude: Round end rewards (handled by round end)
				// Exclude: Bomb plant bonus (handled by bomb plant)

				// Simple heuristic: If in freeze time, it's likely a refund.
				// If live, it could be kill reward.
				// We can check if player has kills recently? No.

				// Better: If money increased and NO kill happened in this tick?
				// That's hard to sync.

				// Safe bet: Only detect refunds during freeze time + a few seconds?
				// Or just log it as "MoneyIncrease" and let frontend decide?
				// User asked for "Refunds".

				// Let's assume money increase during freeze time is a refund.
				if !ctx.FreezeTimeEnded {
					diff := moneyNow - moneyBefore

					event := models.TimelineEvent{
						Type:  "buy", // Use "buy" type but with Refund flag
						Tick:  gs.IngameTick(),
						Round: ctx.CurrentRound,
						Buy: &models.BuyEvent{
							Player:    player.Name,
							SteamID:   player.SteamID64,
							Item:      "Refund", // We don't know what item
							Cost:      -diff,    // Negative cost for refund
							MoneyLeft: moneyNow,
							Refund:    true,
						},
					}

					ctx.CurrentRoundEvents = append(ctx.CurrentRoundEvents, event)
					ctx.Timeline = append(ctx.Timeline, event)
				}

				// Always update money tracking
				ctx.PlayerMoneyBefore[player.SteamID64] = moneyNow
			} else if moneyNow < moneyBefore {
				// Money decreased but NO ItemPickup? (e.g. buying armor via menu might not trigger ItemPickup?)
				// Wait, ItemPickup DOES trigger for armor.
				// But if we missed an event, we should update moneyBefore to avoid accumulating diffs.
				// However, we want to capture the buy in ItemPickup.
				// If we update here, we might race with ItemPickup.
				// ItemPickup handler runs BEFORE FrameDone? Usually yes.
				// So if ItemPickup handled it, moneyBefore is already updated.
				// If ItemPickup didn't handle it (e.g. some weird item), we might want to sync.
				// But let's trust ItemPickup for now.
			}
		}
	})
}

// captureGameState captura el estado completo del juego en un tick
func captureGameState(ctx *models.DemoContext, tick int) {
	gs := ctx.Parser.GameState()

	// Capturar estado de todos los jugadores
	playerSnapshots := make([]models.PlayerStateSnapshot, 0, 10)

	ctAlive := 0
	tAlive := 0

	for _, player := range gs.Participants().Playing() {
		if player == nil {
			continue
		}

		// Contar vivos
		if player.IsAlive() {
			if player.Team == common.TeamCounterTerrorists {
				ctAlive++
			} else if player.Team == common.TeamTerrorists {
				tAlive++
			}
		}

		// Capturar weapons
		primary := ""
		secondary := ""
		grenades := []string{}

		for _, weapon := range player.Weapons() {
			if weapon == nil {
				continue
			}
			switch weapon.Class() {
			case common.EqClassPistols:
				secondary = weapon.String()
			case common.EqClassSMG, common.EqClassRifle, common.EqClassHeavy:
				primary = weapon.String()
			case common.EqClassGrenade:
				grenades = append(grenades, weapon.String())
			}
		}

		activeWeapon := ""
		if player.ActiveWeapon() != nil {
			activeWeapon = player.ActiveWeapon().String()
		}

		team := "Spectator"
		if player.Team == common.TeamCounterTerrorists {
			team = "CT"
		} else if player.Team == common.TeamTerrorists {
			team = "T"
		}

		pos := player.Position()
		vel := player.Velocity()
		viewX := player.ViewDirectionX()
		viewY := player.ViewDirectionY()

		snapshot := models.PlayerStateSnapshot{
			Name:            player.Name,
			SteamID:         player.SteamID64,
			Team:            team,
			IsAlive:         player.IsAlive(),
			HP:              player.Health(),
			Armor:           player.Armor(),
			HasHelmet:       player.HasHelmet(),
			Money:           player.Money(),
			EquipValue:      player.EquipmentValueCurrent(),
			X:               pos.X,
			Y:               pos.Y,
			Z:               pos.Z,
			VelocityX:       vel.X,
			VelocityY:       vel.Y,
			VelocityZ:       vel.Z,
			ViewX:           float64(viewX),
			ViewY:           float64(viewY),
			ActiveWeapon:    activeWeapon,
			PrimaryWeapon:   primary,
			SecondaryWeapon: secondary,
			Grenades:        grenades,
			HasDefuser:      player.HasDefuseKit(),
			FlashDuration:   float64(player.FlashDuration),
		}

		playerSnapshots = append(playerSnapshots, snapshot)
	}

	// Determinar phase
	phase := "live"
	if gs.IsWarmupPeriod() {
		phase = "warmup"
	} else if gs.IsFreezetimePeriod() {
		phase = "freezetime"
	}

	// Calcular tiempo restante aproximado (asumiendo tick rate 128)
	// Ronda típica = 115 segundos = 14720 ticks a 128 tick/s
	// Esto es aproximado, se puede mejorar con tracking de round start tick
	timeRemaining := 0.0
	if gs.TotalRoundsPlayed() > 0 {
		timeRemaining = float64(14720-gs.IngameTick()) / 128.0
		if timeRemaining < 0 {
			timeRemaining = 0
		}
	}

	// Crear game state snapshot
	gameState := models.GameStateSnapshot{
		CTScore:       gs.TeamCounterTerrorists().Score(),
		TScore:        gs.TeamTerrorists().Score(),
		TimeRemaining: timeRemaining,
		BombPlanted:   ctx.BombPlanted,
		BombSite:      ctx.BombSite,
		Players:       playerSnapshots,
		CTAlive:       ctAlive,
		TAlive:        tAlive,
		Phase:         phase,
	}

	// Crear timeline event
	event := models.TimelineEvent{
		Type:      "game_state",
		Tick:      tick,
		Round:     ctx.CurrentRound,
		GameState: &gameState,
	}

	ctx.CurrentRoundEvents = append(ctx.CurrentRoundEvents, event)
	ctx.Timeline = append(ctx.Timeline, event)
}

func calculateLossBonus(losses int) int {
	if losses > 4 {
		losses = 4
	}
	return 1400 + (losses * 500)
}

// AddTimelineEvent es un helper para agregar eventos desde otros handlers
func AddTimelineEvent(ctx *models.DemoContext, event models.TimelineEvent) {
	event.Round = ctx.CurrentRound
	ctx.CurrentRoundEvents = append(ctx.CurrentRoundEvents, event)
	ctx.Timeline = append(ctx.Timeline, event)
}

// Helpers para crear eventos rápidamente desde otros handlers

func AddKillToTimeline(ctx *models.DemoContext, tick int, kill *models.KillEvent) {
	event := models.TimelineEvent{
		Type:  "kill",
		Tick:  tick,
		Round: ctx.CurrentRound,
		Kill:  kill,
	}
	AddTimelineEvent(ctx, event)
}

func AddDamageToTimeline(ctx *models.DemoContext, tick int, damage *models.DamageEvent) {
	event := models.TimelineEvent{
		Type:   "damage",
		Tick:   tick,
		Round:  ctx.CurrentRound,
		Damage: damage,
	}
	AddTimelineEvent(ctx, event)
}

func AddGrenadeToTimeline(ctx *models.DemoContext, tick int, grenadeType string, data interface{}) {
	event := models.TimelineEvent{
		Type:  grenadeType,
		Tick:  tick,
		Round: ctx.CurrentRound,
	}

	switch grenadeType {
	case "flash":
		event.Flash = data.(*models.FlashEvent)
	case "he":
		event.HE = data.(*models.HEEvent)
	case "smoke":
		event.Smoke = data.(*models.SmokeEvent)
	case "molotov":
		event.Molotov = data.(*models.MolotovEvent)
	}

	AddTimelineEvent(ctx, event)
}

func AddBombToTimeline(ctx *models.DemoContext, tick int, bomb *models.BombEvent) {
	event := models.TimelineEvent{
		Type:  "bomb",
		Tick:  tick,
		Round: ctx.CurrentRound,
		Bomb:  bomb,
	}
	AddTimelineEvent(ctx, event)

	// Update bomb tracking in context
	if bomb.EventType == "plant" {
		ctx.BombPlanted = true
		ctx.BombSite = bomb.Site
	} else if bomb.EventType == "defuse" || bomb.EventType == "explode" {
		ctx.BombPlanted = false
	}
}
