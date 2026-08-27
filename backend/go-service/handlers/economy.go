package handlers

import (
	"cs2-demo-service/models"
	"cs2-demo-service/pkg/economy"
	"cs2-demo-service/pkg/playerstate"
	"fmt"
	"sort"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

const (
	maximumPlayerMoney       = 16000
	economyDropMatchSeconds  = 10.0
	economyPriceTableVersion = economy.PriceTableVersion
)

// RegisterEconomyHandlers registra handlers de economía
func RegisterEconomyHandlers(ctx *models.DemoContext) {
	freezeCapturePending := false
	purchasedEntities := make(map[int64]struct{})
	recordedPurchasePickups := make(map[string]struct{})

	// Round start - capturar economía inicial
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		gs := ctx.Parser.GameState()
		if gs.IsWarmupPeriod() {
			return
		}
		roundNumber := gs.TotalRoundsPlayed() + 1
		ctx.MatchData.Economy = withoutLegacyEconomyRound(ctx.MatchData.Economy, roundNumber)

		// Economía CT
		ctEconomy := models.RoundEconomyStats{
			Round:          roundNumber,
			Team:           "CT",
			StartMoney:     0,
			EquipmentValue: 0,
			MoneySpent:     0,
			FullBuys:       0,
			PartialBuys:    0,
			Saves:          0,
		}

		ctPlayers := 0
		for _, player := range uniqueEconomyPlayers(gs.TeamCounterTerrorists().Members()) {
			if player.IsConnected {
				ctPlayers++
				ctEconomy.StartMoney += player.Money()
				ctEconomy.EquipmentValue += player.EquipmentValueCurrent()

				// Clasificar tipo de compra
				equipValue := player.EquipmentValueCurrent()
				if equipValue >= 4000 {
					ctEconomy.FullBuys++
				} else if equipValue >= 2000 {
					ctEconomy.PartialBuys++
				} else {
					ctEconomy.Saves++
				}
			}
		}

		if ctPlayers > 0 {
			ctEconomy.StartMoney /= ctPlayers
			ctEconomy.EquipmentValue /= ctPlayers
		}

		ctx.MatchData.Economy = append(ctx.MatchData.Economy, ctEconomy)

		// Economía T
		tEconomy := models.RoundEconomyStats{
			Round:          roundNumber,
			Team:           "T",
			StartMoney:     0,
			EquipmentValue: 0,
			MoneySpent:     0,
			FullBuys:       0,
			PartialBuys:    0,
			Saves:          0,
		}

		tPlayers := 0
		for _, player := range uniqueEconomyPlayers(gs.TeamTerrorists().Members()) {
			if player.IsConnected {
				tPlayers++
				tEconomy.StartMoney += player.Money()
				tEconomy.EquipmentValue += player.EquipmentValueCurrent()

				equipValue := player.EquipmentValueCurrent()
				if equipValue >= 4000 {
					tEconomy.FullBuys++
				} else if equipValue >= 2000 {
					tEconomy.PartialBuys++
				} else {
					tEconomy.Saves++
				}
			}
		}

		if tPlayers > 0 {
			tEconomy.StartMoney /= tPlayers
			tEconomy.EquipmentValue /= tPlayers
		}

		ctx.MatchData.Economy = append(ctx.MatchData.Economy, tEconomy)
	})

	// ItemPickup is also emitted for purchases. Freeze time covers most buys, but
	// legal purchases can continue after it ends while the player is in a buy zone.
	ctx.Parser.RegisterEventHandler(func(e events.ItemPickup) {
		recordEconomyPurchasePickup(ctx, e, purchasedEntities, recordedPurchasePickups)
	})

	ctx.Parser.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		ctx.FreezeTimeEnded = true

		gs := ctx.Parser.GameState()

		// Actualizar MoneySpent para economía CT
		for i := range ctx.MatchData.Economy {
			econ := &ctx.MatchData.Economy[i]
			if econ.Round == ctx.CurrentRound && econ.Team == "CT" {
				totalSpent := 0
				ctPlayers := 0
				for _, player := range uniqueEconomyPlayers(gs.TeamCounterTerrorists().Members()) {
					if player.IsConnected {
						totalSpent += player.MoneySpentThisRound()
						ctPlayers++
					}
				}
				if ctPlayers > 0 {
					econ.MoneySpent = totalSpent / ctPlayers
				}
			}
		}

		// Actualizar MoneySpent para economía T
		for i := range ctx.MatchData.Economy {
			econ := &ctx.MatchData.Economy[i]
			if econ.Round == ctx.CurrentRound && econ.Team == "T" {
				totalSpent := 0
				tPlayers := 0
				for _, player := range uniqueEconomyPlayers(gs.TeamTerrorists().Members()) {
					if player.IsConnected {
						totalSpent += player.MoneySpentThisRound()
						tPlayers++
					}
				}
				if tPlayers > 0 {
					econ.MoneySpent = totalSpent / tPlayers
				}
			}
		}

		// The event is emitted before all player netvars are settled. Mark the
		// snapshot for the immediately following FrameDone event.
		freezeCapturePending = true
	})

	ctx.Parser.RegisterEventHandler(func(events.FrameDone) {
		if !freezeCapturePending {
			return
		}
		captureFreezeTimeEquipment(ctx)
		freezeCapturePending = false
	})

	// AI Economy Handler - Capture per-round economy data
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		ctx.FreezeTimeEnded = false // Reset freeze time flag
		gs := ctx.Parser.GameState()

		// Skip warmup
		if gs.IsWarmupPeriod() {
			return
		}

		// Calculate actual round number (1-based, same as timeline.go)
		currentRound := gs.TotalRoundsPlayed() + 1
		ctx.AI_EconomyRounds = withoutDetailedEconomyRound(ctx.AI_EconomyRounds, currentRound)
		ctLossBonusLevel := ctx.CTConsecutiveLosses
		tLossBonusLevel := ctx.TConsecutiveLosses
		if isLossBonusResetRound(currentRound) {
			ctLossBonusLevel = initialLossBonusLevel
			tLossBonusLevel = initialLossBonusLevel
		}

		// Reset RoundPurchases
		ctx.RoundPurchases = make(map[uint64][]models.AI_WeaponItem)
		purchasedEntities = make(map[int64]struct{})
		recordedPurchasePickups = make(map[string]struct{})

		// Reset economy drop tracking for new round
		ctx.PendingDrops = []models.AI_EconomyDrop{}
		ctx.RoundDrops = []models.AI_EconomyDrop{}
		ctx.RoundPickups = []models.AI_EconomyPickup{}
		ctx.RoundRefunds = []models.AI_EconomyRefund{}
		ctx.PlayerDropsGiven = make(map[uint64][]string)
		ctx.PlayerDropsReceived = make(map[uint64][]string)
		ctx.PlayerRefunds = make(map[uint64][]string)

		// ENTITY-BASED TRACKING: Record original owner of each weapon at round start
		ctx.WeaponOriginalOwner = make(map[int64]uint64)
		for _, player := range uniqueEconomyPlayers(gs.Participants().Playing()) {
			if player.SteamID64 == 0 {
				continue
			}
			for _, weapon := range player.Weapons() {
				if weapon != nil {
					ctx.WeaponOriginalOwner[playerstate.EquipmentID(weapon)] = player.SteamID64
				}
			}
		}

		// Initialize new round data
		roundData := models.AI_EconomyRound{
			Round:   currentRound,
			Teams:   make(map[string]models.AI_EconomyTeam),
			Players: []models.AI_EconomyPlayer{},
		}

		// Calculate Team Data with balance metrics
		tMoney := 0
		ctMoney := 0
		tMoneyList := []int{}
		ctMoneyList := []int{}

		for _, p := range uniqueEconomyPlayers(gs.TeamTerrorists().Members()) {
			if p.IsConnected {
				tMoney += p.Money()
				tMoneyList = append(tMoneyList, p.Money())
			}
		}
		for _, p := range uniqueEconomyPlayers(gs.TeamCounterTerrorists().Members()) {
			if p.IsConnected {
				ctMoney += p.Money()
				ctMoneyList = append(ctMoneyList, p.Money())
			}
		}

		// Helper: Calculate balance statistics for a team
		calcBalanceStats := func(moneys []int) (avg, spread int, gini float64) {
			if len(moneys) == 0 {
				return 0, 0, 0
			}
			// Calculate total and average
			total := 0
			for _, m := range moneys {
				total += m
			}
			avg = total / len(moneys)

			// Calculate spread (max - min)
			minM, maxM := moneys[0], moneys[0]
			for _, m := range moneys {
				if m < minM {
					minM = m
				}
				if m > maxM {
					maxM = m
				}
			}
			spread = maxM - minM

			// Calculate Gini coefficient
			n := float64(len(moneys))
			if n > 1 && total > 0 {
				var sumDiff float64
				for _, mi := range moneys {
					for _, mj := range moneys {
						if mi > mj {
							sumDiff += float64(mi - mj)
						} else {
							sumDiff += float64(mj - mi)
						}
					}
				}
				gini = sumDiff / (2 * n * float64(total))
			}
			return
		}

		// Calculate T team stats
		tAvg, tSpread, tGini := calcBalanceStats(tMoneyList)

		// Calculate CT team stats
		ctAvg, ctSpread, ctGini := calcBalanceStats(ctMoneyList)

		roundData.Teams["T"] = models.AI_EconomyTeam{
			TeamID:          gs.TeamTerrorists().ID(),
			ClanName:        gs.TeamTerrorists().ClanName(),
			TotalMoney:      tMoney,
			LossBonus:       calculateLossBonus(tLossBonusLevel),
			AverageMoney:    tAvg,
			MoneySpread:     tSpread,
			GiniCoefficient: tGini,
			RoundsWon:       gs.TeamTerrorists().Score(),
		}
		roundData.Teams["CT"] = models.AI_EconomyTeam{
			TeamID:          gs.TeamCounterTerrorists().ID(),
			ClanName:        gs.TeamCounterTerrorists().ClanName(),
			TotalMoney:      ctMoney,
			LossBonus:       calculateLossBonus(ctLossBonusLevel),
			AverageMoney:    ctAvg,
			MoneySpread:     ctSpread,
			GiniCoefficient: ctGini,
			RoundsWon:       gs.TeamCounterTerrorists().Score(),
		}

		// Process Players
		for _, player := range uniqueEconomyPlayers(gs.Participants().Playing()) {
			if player.SteamID64 == 0 {
				continue
			}

			teamName := "Spectator"
			if player.Team == 2 {
				teamName = "T"
			} else if player.Team == 3 {
				teamName = "CT"
			}

			spawnArea := teamName + "Spawn"
			if player.LastPlaceName() != "" {
				spawnArea = player.LastPlaceName()
			}

			equipValue := player.EquipmentValueRoundStart()

			// Create start items with EntityID tracking (exclude Knife)
			startItems := []models.AI_WeaponItem{}
			for _, weapon := range player.Weapons() {
				if weapon == nil {
					continue
				}
				weaponName := weapon.String()
				if weaponName == "Knife" {
					continue // Skip knife - always present, not useful data
				}
				startItems = append(startItems, economyWeaponItem(weaponName, playerstate.EquipmentID(weapon), player.SteamID64, "observed"))
			}
			startItems = appendMissingFlashbangs(startItems, int(player.FlashbangCount()), player.SteamID64, "observed")

			// Add armor (Kevlar / Kevlar + Helmet)
			if player.Armor() > 0 {
				if player.HasHelmet() {
					startItems = append(startItems, economyWeaponItem("Kevlar + Helmet", 0, 0, "not_observed"))
				} else {
					startItems = append(startItems, economyWeaponItem("Kevlar Vest", 0, 0, "not_observed"))
				}
			}

			// Add defuse kit (CT only)
			if player.HasDefuseKit() {
				startItems = append(startItems, economyWeaponItem("Defuse Kit", 0, 0, "not_observed"))
			}

			pData := models.AI_EconomyPlayer{
				SteamID:                       player.SteamID64,
				Name:                          player.Name,
				Team:                          teamName,
				SpawnArea:                     spawnArea,
				InitialMoney:                  player.Money(),
				NextRoundMinMoney:             0, // Deprecated legacy field; canonical next-round money remains unavailable unless observed.
				StartRoundItems:               startItems,
				EquipmentValueStart:           equipValue,
				EquipmentValueStartCalculated: equipmentItemsValue(startItems),
				SpentInBuy:                    0,
				Purchases:                     []models.AI_WeaponItem{},
				FinalEquipment:                []models.AI_WeaponItem{}, // Will be filled at FreezetimeEnd
				FinalEquipmentValue:           0,
				FinalMoney:                    0,
				EquipmentValueEnd:             0,
				EndEquipment:                  []models.AI_WeaponItem{}, // Will be filled at RoundEnd
				Outcome:                       "",
				WinReason:                     "",
				Survived:                      false,
			}
			roundData.Players = append(roundData.Players, pData)
		}

		// Sort players by team (CT first, then T)
		sort.SliceStable(roundData.Players, func(i, j int) bool {
			if roundData.Players[i].Team == "CT" && roundData.Players[j].Team != "CT" {
				return true
			}
			if roundData.Players[i].Team != "CT" && roundData.Players[j].Team == "CT" {
				return false
			}
			return false // Keep original order within same team
		})

		ctx.AI_EconomyRounds = append(ctx.AI_EconomyRounds, roundData)
	})

	// Update AI Economy with round results
	ctx.Parser.RegisterEventHandler(func(e events.RoundEnd) {
		gs := ctx.Parser.GameState()
		updateLegacyTeamMoneySpent(ctx, "CT", gs.TeamCounterTerrorists().Members())
		updateLegacyTeamMoneySpent(ctx, "T", gs.TeamTerrorists().Members())
		winningTeam := ""
		if e.Winner == 2 {
			winningTeam = "T"
		} else if e.Winner == 3 {
			winningTeam = "CT"
		}

		winReason := fmt.Sprintf("%d", e.Reason)
		switch e.Reason {
		case events.RoundEndReasonTargetBombed:
			winReason = "BombExploded"
		case events.RoundEndReasonBombDefused:
			winReason = "BombDefused"
		case events.RoundEndReasonCTWin:
			winReason = "CTWin"
		case events.RoundEndReasonTerroristsWin:
			winReason = "TWin"
		case events.RoundEndReasonTargetSaved:
			winReason = "TargetSaved"
		}

		if len(ctx.AI_EconomyRounds) == 0 {
			return
		}
		roundData := &ctx.AI_EconomyRounds[len(ctx.AI_EconomyRounds)-1]
		players := gs.Participants().All()
		playersByID := make(map[uint64]*common.Player, len(players))
		for _, player := range players {
			if player == nil || player.SteamID64 == 0 {
				continue
			}
			playersByID[player.SteamID64] = player
		}

		for i := range roundData.Players {
			pData := &roundData.Players[i]
			player := playersByID[pData.SteamID]
			if player == nil {
				continue
			}

			pData.Outcome = "unknown"
			if winningTeam != "" && pData.Team == winningTeam {
				pData.Outcome = "win"
			} else if winningTeam != "" {
				pData.Outcome = "loss"
			}
			pData.WinReason = winReason
			pData.Survived = player.IsAlive()
			pData.MoneyAtRoundEnd = player.Money()
			pData.FinalMoney = pData.MoneyAtRoundEnd
			pData.EquipmentValueEndNative = player.EquipmentValueCurrent()
			// demoinfocs exposes the previous round's freeze-time value while the
			// RoundFreezetimeEnd event is being dispatched. At RoundEnd the native
			// netvar is settled and can be reconciled with the captured inventory.
			pData.FinalEquipmentValue = player.EquipmentValueFreezeTimeEnd()
			pData.SpentInBuy = player.MoneySpentThisRound()
			if purchases, ok := ctx.RoundPurchases[pData.SteamID]; ok {
				pData.Purchases = purchases
			}
			pData.PurchasesObservedValue = equipmentItemsValue(pData.Purchases)
			pData.PurchasesVsSpentDelta = pData.SpentInBuy - pData.PurchasesObservedValue

			pData.EndEquipment, pData.EquipmentValueEndCalculated = equipmentSnapshot(ctx, player)
			pData.EquipmentValueEndNative = player.EquipmentValueCurrent()
			pData.EquipmentValueEnd = pData.EquipmentValueEndCalculated

			refundValue := playerRefundValue(ctx.RoundRefunds, player.SteamID64)
			if refunds, ok := ctx.PlayerRefunds[player.SteamID64]; ok {
				pData.Refunds = refunds
			}
			pData.MoneyAfterBuyCalculated = calculateMoneyAfterBuy(pData.InitialMoney, pData.SpentInBuy, refundValue)
			pData.NextRoundMinMoney = 0
		}

		ctx.RoundDrops = append(ctx.RoundDrops, ctx.PendingDrops...)
		ctx.PendingDrops = nil

		// Add economy events to round data if any exist
		if len(ctx.RoundDrops) > 0 || len(ctx.RoundPickups) > 0 || len(ctx.RoundRefunds) > 0 {
			roundData.Events = &models.AI_EconomyRoundEvents{
				Drops:   ctx.RoundDrops,
				Pickups: ctx.RoundPickups,
				Refunds: ctx.RoundRefunds,
			}
		}
	})

	// Track weapon drops
	ctx.Parser.RegisterEventHandler(func(e events.ItemDrop) {
		if e.Player == nil || e.Weapon == nil {
			return
		}
		gs := ctx.Parser.GameState()
		if gs.IsWarmupPeriod() || ctx.CurrentRound < 1 {
			return
		}

		weaponName := e.Weapon.String()
		weaponValue, priceStatus := getWeaponPriceQuote(weaponName)

		drop := models.AI_EconomyDrop{
			Tick:              gs.IngameTick(),
			Dropper:           e.Player.Name,
			DropperID:         e.Player.SteamID64,
			DropperMoney:      e.Player.Money(),
			Weapon:            weaponName,
			WeaponValue:       weaponValue,
			PriceStatus:       priceStatus,
			PriceTableVersion: economyPriceTableVersion,
			EntityID:          playerstate.EquipmentID(e.Weapon),
			PickedUp:          false,
		}

		// Add to pending drops for matching with pickups
		ctx.PendingDrops = append(ctx.PendingDrops, drop)

		// Track drops given by player
		ctx.PlayerDropsGiven[e.Player.SteamID64] = append(ctx.PlayerDropsGiven[e.Player.SteamID64], weaponName)
	})

	// Track pickups (outside of freezetime = ground pickups, not purchases)
	ctx.Parser.RegisterEventHandler(func(e events.ItemPickup) {
		if e.Player == nil || e.Weapon == nil {
			return
		}

		weaponName := e.Weapon.String()
		weaponValue, priceStatus := getWeaponPriceQuote(weaponName)
		entityID := playerstate.EquipmentID(e.Weapon)
		if recordEconomyPurchasePickup(ctx, e, purchasedEntities, recordedPurchasePickups) {
			return
		}

		gs := ctx.Parser.GameState()
		currentTick := gs.IngameTick()
		// Try to match with a pending drop
		matched := false
		var matchedDrop *models.AI_EconomyDrop
		var matchedIdx int

		matchWindow := economyDropMatchWindowTicks(ctx.Parser.TickRate())
		for i, drop := range ctx.PendingDrops {
			if economyDropMatches(drop, currentTick, entityID, weaponName, matchWindow) {
				matchedDrop = &ctx.PendingDrops[i]
				matchedIdx = i
				matched = true
				break
			}
		}

		if matched && matchedDrop != nil {
			// Check if it's a teammate pickup
			dropperTeam := 0
			pickerTeam := int(e.Player.Team)

			for _, p := range gs.Participants().All() {
				if p.SteamID64 == matchedDrop.DropperID {
					dropperTeam = int(p.Team)
					break
				}
			}

			matchedDrop.Receiver = e.Player.Name
			matchedDrop.ReceiverID = e.Player.SteamID64
			matchedDrop.ReceiverMoney = e.Player.Money()
			matchedDrop.PickedUp = true
			if dropperTeam == pickerTeam && matchedDrop.DropperID != e.Player.SteamID64 {
				ctx.PlayerDropsReceived[e.Player.SteamID64] = append(ctx.PlayerDropsReceived[e.Player.SteamID64], weaponName)
			}
			ctx.RoundDrops = append(ctx.RoundDrops, *matchedDrop)
			ctx.RoundPickups = append(ctx.RoundPickups, models.AI_EconomyPickup{
				Tick:              currentTick,
				Player:            e.Player.Name,
				PlayerID:          e.Player.SteamID64,
				Weapon:            weaponName,
				WeaponValue:       weaponValue,
				PriceStatus:       priceStatus,
				PriceTableVersion: economyPriceTableVersion,
				EntityID:          entityID,
				FromDrop:          true,
				FromPlayer:        matchedDrop.Dropper,
				FromPlayerID:      matchedDrop.DropperID,
			})

			// Remove from pending
			ctx.PendingDrops = append(ctx.PendingDrops[:matchedIdx], ctx.PendingDrops[matchedIdx+1:]...)
		} else {
			if entityID != 0 {
				if _, purchase := purchasedEntities[entityID]; purchase {
					return
				}
			}
			if !ctx.FreezeTimeEnded {
				return
			}
			// Ground pickup (not from a tracked drop)
			pickup := models.AI_EconomyPickup{
				Tick:              currentTick,
				Player:            e.Player.Name,
				PlayerID:          e.Player.SteamID64,
				Weapon:            weaponName,
				WeaponValue:       weaponValue,
				PriceStatus:       priceStatus,
				PriceTableVersion: economyPriceTableVersion,
				EntityID:          entityID,
				FromDrop:          false,
			}
			ctx.RoundPickups = append(ctx.RoundPickups, pickup)
		}
	})

	// Track refunds (CS2 only)
	ctx.Parser.RegisterEventHandler(func(e events.ItemRefund) {
		if e.Player == nil || e.Weapon == nil {
			return
		}

		weaponName := e.Weapon.String()
		refundValue, priceStatus := getWeaponPriceQuote(weaponName)

		refund := models.AI_EconomyRefund{
			Tick:              ctx.Parser.GameState().IngameTick(),
			Player:            e.Player.Name,
			PlayerID:          e.Player.SteamID64,
			Weapon:            weaponName,
			RefundValue:       refundValue,
			PriceStatus:       priceStatus,
			PriceTableVersion: economyPriceTableVersion,
		}

		ctx.RoundRefunds = append(ctx.RoundRefunds, refund)
		ctx.PlayerRefunds[e.Player.SteamID64] = append(ctx.PlayerRefunds[e.Player.SteamID64], weaponName)
	})
}

func recordEconomyPurchasePickup(
	ctx *models.DemoContext,
	event events.ItemPickup,
	purchasedEntities map[int64]struct{},
	recordedPickups map[string]struct{},
) bool {
	if event.Player == nil || event.Weapon == nil || ctx == nil || ctx.Parser == nil {
		return false
	}
	gameState := ctx.Parser.GameState()
	if gameState.IsWarmupPeriod() || ctx.CurrentRound < 1 || ctx.FreezeTimeEnded && !event.Player.IsInBuyZone() {
		return false
	}
	weaponName := event.Weapon.String()
	weaponPrice, priceStatus := getWeaponPriceQuote(weaponName)
	equipmentID := playerstate.EquipmentID(event.Weapon)
	if equipmentID != 0 {
		if _, existed := ctx.WeaponOriginalOwner[equipmentID]; existed {
			return false
		}
		if _, purchased := purchasedEntities[equipmentID]; purchased {
			return true
		}
	}
	pickupKey := economyPickupKey(gameState.IngameTick(), event.Player.SteamID64, equipmentID, weaponName)
	if _, duplicate := recordedPickups[pickupKey]; duplicate {
		return true
	}
	item := economyWeaponItem(
		weaponName,
		equipmentID,
		event.Player.SteamID64,
		"observed",
	)
	item.Price = weaponPrice
	item.PriceStatus = priceStatus
	ctx.RoundPurchases[event.Player.SteamID64] = append(ctx.RoundPurchases[event.Player.SteamID64], item)
	recordedPickups[pickupKey] = struct{}{}
	if equipmentID != 0 {
		purchasedEntities[equipmentID] = struct{}{}
	}
	return true
}

func captureFreezeTimeEquipment(ctx *models.DemoContext) {
	if len(ctx.AI_EconomyRounds) == 0 {
		return
	}

	gameState := ctx.Parser.GameState()
	players := uniqueEconomyPlayers(gameState.Participants().Playing())
	playersByID := make(map[uint64]*common.Player, len(players))
	for _, player := range players {
		if player == nil || player.SteamID64 == 0 {
			continue
		}
		playersByID[player.SteamID64] = player
	}

	roundData := &ctx.AI_EconomyRounds[len(ctx.AI_EconomyRounds)-1]
	for i := range roundData.Players {
		playerData := &roundData.Players[i]
		player := playersByID[playerData.SteamID]
		if player == nil {
			continue
		}

		playerData.FinalEquipment, playerData.FinalEquipmentValueCalculated = equipmentSnapshot(ctx, player)
		playerData.FinalEquipmentValue = player.EquipmentValueFreezeTimeEnd()
		playerData.MoneyAfterBuy = player.Money()
		playerData.MoneyAfterBuyObserved = true
		if purchases, ok := ctx.RoundPurchases[playerData.SteamID]; ok {
			playerData.Purchases = purchases
		}
	}

	updateLegacyTeamEconomy(ctx, "CT", gameState.TeamCounterTerrorists().Members())
	updateLegacyTeamEconomy(ctx, "T", gameState.TeamTerrorists().Members())
}

func updateLegacyTeamEconomy(ctx *models.DemoContext, team string, players []*common.Player) {
	players = uniqueEconomyPlayers(players)
	totalEquipment := 0
	totalSpent := 0
	connected := 0
	fullBuys := 0
	partialBuys := 0
	saves := 0
	for _, player := range players {
		if player == nil || !player.IsConnected {
			continue
		}
		connected++
		equipmentValue := player.EquipmentValueFreezeTimeEnd()
		totalEquipment += equipmentValue
		totalSpent += player.MoneySpentThisRound()
		switch {
		case equipmentValue >= 4000:
			fullBuys++
		case equipmentValue >= 2000:
			partialBuys++
		default:
			saves++
		}
	}
	if connected == 0 {
		return
	}
	for i := range ctx.MatchData.Economy {
		economy := &ctx.MatchData.Economy[i]
		if economy.Round != ctx.CurrentRound || economy.Team != team {
			continue
		}
		economy.EquipmentValue = totalEquipment / connected
		economy.MoneySpent = totalSpent / connected
		economy.FullBuys = fullBuys
		economy.PartialBuys = partialBuys
		economy.Saves = saves
		return
	}
}

func uniqueEconomyPlayers(players []*common.Player) []*common.Player {
	bySteamID := make(map[uint64]*common.Player, len(players))
	for _, player := range players {
		if player == nil || player.SteamID64 == 0 {
			continue
		}
		current := bySteamID[player.SteamID64]
		if current == nil || !current.IsConnected && player.IsConnected ||
			current.IsConnected == player.IsConnected && player.EntityID > current.EntityID {
			bySteamID[player.SteamID64] = player
		}
	}
	steamIDs := make([]uint64, 0, len(bySteamID))
	for steamID := range bySteamID {
		steamIDs = append(steamIDs, steamID)
	}
	sort.Slice(steamIDs, func(i, j int) bool { return steamIDs[i] < steamIDs[j] })
	unique := make([]*common.Player, 0, len(steamIDs))
	for _, steamID := range steamIDs {
		unique = append(unique, bySteamID[steamID])
	}
	return unique
}

func withoutLegacyEconomyRound(rounds []models.RoundEconomyStats, roundNumber int) []models.RoundEconomyStats {
	filtered := rounds[:0]
	for _, round := range rounds {
		if round.Round != roundNumber {
			filtered = append(filtered, round)
		}
	}
	return filtered
}

func withoutDetailedEconomyRound(rounds []models.AI_EconomyRound, roundNumber int) []models.AI_EconomyRound {
	filtered := rounds[:0]
	for _, round := range rounds {
		if round.Round != roundNumber {
			filtered = append(filtered, round)
		}
	}
	return filtered
}

func updateLegacyTeamMoneySpent(ctx *models.DemoContext, team string, players []*common.Player) {
	players = uniqueEconomyPlayers(players)
	totalSpent := 0
	connected := 0
	for _, player := range players {
		if player == nil || !player.IsConnected {
			continue
		}
		totalSpent += player.MoneySpentThisRound()
		connected++
	}
	if connected == 0 {
		return
	}
	for i := range ctx.MatchData.Economy {
		economy := &ctx.MatchData.Economy[i]
		if economy.Round == ctx.CurrentRound && economy.Team == team {
			economy.MoneySpent = totalSpent / connected
			return
		}
	}
}

func equipmentSnapshot(ctx *models.DemoContext, player *common.Player) ([]models.AI_WeaponItem, int) {
	items := make([]models.AI_WeaponItem, 0, len(player.Weapons())+2)
	totalValue := 0
	for _, weapon := range player.Weapons() {
		if weapon == nil || weapon.String() == "Knife" {
			continue
		}
		weaponName := weapon.String()
		entityID := playerstate.EquipmentID(weapon)
		ownerID, exists := ctx.WeaponOriginalOwner[entityID]
		ownerStatus := "not_observed"
		if exists && ownerID != 0 {
			ownerStatus = "observed"
		}
		item := economyWeaponItem(weaponName, entityID, ownerID, ownerStatus)
		items = append(items, item)
		totalValue += item.Price
	}
	items = appendMissingFlashbangs(items, int(player.FlashbangCount()), 0, "not_observed")
	totalValue = equipmentItemsValue(items)

	if player.Armor() > 0 {
		armorName := "Kevlar Vest"
		if player.HasHelmet() {
			armorName = "Kevlar + Helmet"
		}
		item := economyWeaponItem(armorName, 0, 0, "not_observed")
		items = append(items, item)
		totalValue += item.Price
	}
	if player.HasDefuseKit() {
		item := economyWeaponItem("Defuse Kit", 0, 0, "not_observed")
		items = append(items, item)
		totalValue += item.Price
	}

	return items, totalValue
}

func getWeaponPrice(weaponName string) int {
	price, _ := getWeaponPriceQuote(weaponName)
	return price
}

func getWeaponPriceQuote(weaponName string) (int, string) {
	return economy.PriceQuote(weaponName)
}

func economyWeaponItem(weaponName string, entityID int64, ownerID uint64, ownerStatus string) models.AI_WeaponItem {
	price, priceStatus := getWeaponPriceQuote(weaponName)
	return models.AI_WeaponItem{
		Weapon: weaponName, Price: price, PriceStatus: priceStatus,
		PriceTableVersion: economyPriceTableVersion, EntityID: entityID,
		OriginalOwnerID: ownerID, OriginalOwnerStatus: ownerStatus,
		ObservationStatus: "observed",
	}
}

func equipmentItemsValue(items []models.AI_WeaponItem) int {
	total := 0
	for _, item := range items {
		total += item.Price
	}
	return total
}

func appendMissingFlashbangs(items []models.AI_WeaponItem, flashbangCount int, ownerID uint64, ownerStatus string) []models.AI_WeaponItem {
	captured := 0
	for _, item := range items {
		if item.Weapon == "Flashbang" {
			captured++
			if item.OriginalOwnerStatus == "observed" {
				ownerID = item.OriginalOwnerID
				ownerStatus = item.OriginalOwnerStatus
			}
		}
	}
	for captured < flashbangCount {
		items = append(items, economyWeaponItem("Flashbang", 0, ownerID, ownerStatus))
		captured++
	}
	return items
}

func economyPickupKey(tick int, steamID uint64, entityID int64, weapon string) string {
	return fmt.Sprintf("%d|%d|%d|%s", tick, steamID, entityID, weapon)
}

func playerRefundValue(refunds []models.AI_EconomyRefund, steamID uint64) int {
	total := 0
	for _, refund := range refunds {
		if refund.PlayerID == steamID {
			total += refund.RefundValue
		}
	}
	return total
}

func calculateMoneyAfterBuy(initialMoney, spentInBuy, refundValue int) int {
	money := initialMoney - spentInBuy + refundValue
	if money < 0 {
		return 0
	}
	if money > maximumPlayerMoney {
		return maximumPlayerMoney
	}
	return money
}

func economyDropMatchWindowTicks(tickRate float64) int {
	if tickRate <= 0 {
		tickRate = 64
	}
	return int(tickRate * economyDropMatchSeconds)
}

func economyDropMatches(drop models.AI_EconomyDrop, pickupTick int, entityID int64, weaponName string, windowTicks int) bool {
	age := pickupTick - drop.Tick
	if drop.PickedUp || age < 0 || age > windowTicks {
		return false
	}
	return entityID != 0 && drop.EntityID != 0 && entityID == drop.EntityID && drop.Weapon == weaponName
}
