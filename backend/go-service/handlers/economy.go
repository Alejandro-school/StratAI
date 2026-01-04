package handlers

import (
	"cs2-demo-service/models"
	"fmt"
	"sort"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// RegisterEconomyHandlers registra handlers de economía
func RegisterEconomyHandlers(ctx *models.DemoContext) {
	// Round start - capturar economía inicial
	ctx.Parser.RegisterEventHandler(func(e events.RoundStart) {
		gs := ctx.Parser.GameState()

		// Economía CT
		ctEconomy := models.RoundEconomyStats{
			Round:          ctx.CurrentRound,
			Team:           "CT",
			StartMoney:     0,
			EquipmentValue: 0,
			MoneySpent:     0,
			FullBuys:       0,
			PartialBuys:    0,
			Saves:          0,
		}

		ctPlayers := 0
		for _, player := range gs.TeamCounterTerrorists().Members() {
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
			Round:          ctx.CurrentRound,
			Team:           "T",
			StartMoney:     0,
			EquipmentValue: 0,
			MoneySpent:     0,
			FullBuys:       0,
			PartialBuys:    0,
			Saves:          0,
		}

		tPlayers := 0
		for _, player := range gs.TeamTerrorists().Members() {
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

	// Track purchases (using ItemPickup during freezetime)
	// IMPORTANT: We need to filter out items the player already had at round start
	ctx.Parser.RegisterEventHandler(func(e events.ItemPickup) {
		if e.Player == nil || e.Weapon == nil {
			return
		}

		// Only track during freeze time (buy period)
		if ctx.FreezeTimeEnded {
			return
		}

		weaponName := e.Weapon.String()
		weaponPrice := getWeaponPrice(weaponName)

		// Skip free items (knife, C4) and items with no price
		if weaponPrice == 0 {
			return
		}

		// CRITICAL: Check if this weapon already existed at round start (dropped weapon, not purchased)
		// If the weapon's UniqueID is in WeaponOriginalOwner, it's a dropped weapon being picked up
		if _, existed := ctx.WeaponOriginalOwner[e.Weapon.UniqueID()]; existed {
			// This is a pickup of a dropped weapon, NOT a purchase
			return
		}

		// Check if this item was already in player's inventory at round start
		if len(ctx.AI_EconomyRounds) > 0 {
			roundData := &ctx.AI_EconomyRounds[len(ctx.AI_EconomyRounds)-1]
			for _, pData := range roundData.Players {
				if pData.SteamID == e.Player.SteamID64 {
					// Check if weapon was in start items
					alreadyHad := false
					for _, item := range pData.StartRoundItems {
						if item.Weapon == weaponName {
							alreadyHad = true
							break
						}
					}
					// Also check if already purchased this round (avoid duplicates)
					if !alreadyHad {
						for _, item := range ctx.RoundPurchases[e.Player.SteamID64] {
							if item.Weapon == weaponName {
								alreadyHad = true
								break
							}
						}
					}
					if alreadyHad {
						return
					}
					break
				}
			}
		}

		// Add to purchases with EntityID (this is a genuine purchase)
		item := models.AI_WeaponItem{
			Weapon:        weaponName,
			Price:         weaponPrice,
			EntityID:      e.Weapon.UniqueID(),
			OriginalOwner: "purchased",
		}
		ctx.RoundPurchases[e.Player.SteamID64] = append(ctx.RoundPurchases[e.Player.SteamID64], item)
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
				for _, player := range gs.TeamCounterTerrorists().Members() {
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
				for _, player := range gs.TeamTerrorists().Members() {
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

		// ==========================================
		// CAPTURE FINAL EQUIPMENT AT FREEZETIME END
		// ==========================================
		// Record what each player has at the end of buy time
		// The AI can deduce drops by comparing EntityIDs across players' equipment

		roundData := &ctx.AI_EconomyRounds[len(ctx.AI_EconomyRounds)-1]

		// Build player name to SteamID map for getting original owner names
		playerNames := make(map[uint64]string)
		for _, player := range gs.Participants().Playing() {
			if player.SteamID64 != 0 {
				playerNames[player.SteamID64] = player.Name
			}
		}

		// Update each player with FinalEquipment and Purchases
		for i := range roundData.Players {
			pData := &roundData.Players[i]

			// Find the player in game state
			var player *common.Player
			for _, p := range gs.Participants().Playing() {
				if p.SteamID64 == pData.SteamID {
					player = p
					break
				}
			}
			if player == nil {
				continue
			}

			// Capture FinalEquipment with EntityID and OriginalOwner (exclude Knife)
			finalEquipment := []models.AI_WeaponItem{}
			totalValue := 0
			for _, weapon := range player.Weapons() {
				if weapon == nil {
					continue
				}

				weaponName := weapon.String()
				if weaponName == "Knife" {
					continue // Skip knife
				}

				// Determine original owner
				originalOwner := "purchased" // Default for newly bought weapons
				if originalOwnerID, existed := ctx.WeaponOriginalOwner[weapon.UniqueID()]; existed {
					if name, ok := playerNames[originalOwnerID]; ok {
						originalOwner = name
					}
				}

				price := getWeaponPrice(weaponName)
				finalEquipment = append(finalEquipment, models.AI_WeaponItem{
					Weapon:        weaponName,
					Price:         price,
					EntityID:      weapon.UniqueID(),
					OriginalOwner: originalOwner,
				})
				if price > 0 {
					totalValue += price
				}
			}

			// Add armor (Kevlar / Kevlar + Helmet)
			if player.Armor() > 0 {
				if player.HasHelmet() {
					finalEquipment = append(finalEquipment, models.AI_WeaponItem{
						Weapon:        "Kevlar + Helmet",
						Price:         1000,
						EntityID:      0,
						OriginalOwner: "purchased",
					})
					totalValue += 1000
				} else {
					finalEquipment = append(finalEquipment, models.AI_WeaponItem{
						Weapon:        "Kevlar Vest",
						Price:         650,
						EntityID:      0,
						OriginalOwner: "purchased",
					})
					totalValue += 650
				}
			}

			// Add defuse kit (CT only)
			if player.HasDefuseKit() {
				finalEquipment = append(finalEquipment, models.AI_WeaponItem{
					Weapon:        "Defuse Kit",
					Price:         400,
					EntityID:      0,
					OriginalOwner: "purchased",
				})
				totalValue += 400
			}

			pData.FinalEquipment = finalEquipment
			pData.FinalEquipmentValue = totalValue

			// Set purchases from RoundPurchases
			if purchases, ok := ctx.RoundPurchases[pData.SteamID]; ok {
				pData.Purchases = purchases
			}
		}
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

		// Reset RoundPurchases
		ctx.RoundPurchases = make(map[uint64][]models.AI_WeaponItem)

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
		for _, player := range gs.Participants().Playing() {
			if player.SteamID64 == 0 {
				continue
			}
			for _, weapon := range player.Weapons() {
				if weapon != nil {
					ctx.WeaponOriginalOwner[weapon.UniqueID()] = player.SteamID64
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
		tEquipList := []int{}
		ctEquipList := []int{}

		for _, p := range gs.TeamTerrorists().Members() {
			if p.IsConnected {
				tMoney += p.Money()
				tMoneyList = append(tMoneyList, p.Money())
				tEquipList = append(tEquipList, p.EquipmentValueCurrent())
			}
		}
		for _, p := range gs.TeamCounterTerrorists().Members() {
			if p.IsConnected {
				ctMoney += p.Money()
				ctMoneyList = append(ctMoneyList, p.Money())
				ctEquipList = append(ctEquipList, p.EquipmentValueCurrent())
			}
		}

		// Calculate loss bonus
		calcLossBonus := func(losses int) int {
			bonus := 1400 + (500 * losses)
			if bonus > 3400 {
				bonus = 3400
			}
			return bonus
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
			TotalMoney:      tMoney,
			LossBonus:       calcLossBonus(ctx.TConsecutiveLosses),
			AverageMoney:    tAvg,
			MoneySpread:     tSpread,
			GiniCoefficient: tGini,
			RoundsWon:       ctx.TRoundsWon,
		}
		roundData.Teams["CT"] = models.AI_EconomyTeam{
			TotalMoney:      ctMoney,
			LossBonus:       calcLossBonus(ctx.CTConsecutiveLosses),
			AverageMoney:    ctAvg,
			MoneySpread:     ctSpread,
			GiniCoefficient: ctGini,
			RoundsWon:       ctx.CTRoundsWon,
		}

		// Process Players
		for _, player := range gs.Participants().Playing() {
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

			equipValue := player.EquipmentValueCurrent()

			// Create start items with EntityID tracking (exclude Knife)
			startItems := []models.AI_WeaponItem{}
			for _, weapon := range player.Weapons() {
				weaponName := weapon.String()
				if weaponName == "Knife" {
					continue // Skip knife - always present, not useful data
				}
				startItems = append(startItems, models.AI_WeaponItem{
					Weapon:        weaponName,
					Price:         getWeaponPrice(weaponName),
					EntityID:      weapon.UniqueID(),
					OriginalOwner: player.Name,
				})
			}

			// Add armor (Kevlar / Kevlar + Helmet)
			if player.Armor() > 0 {
				if player.HasHelmet() {
					startItems = append(startItems, models.AI_WeaponItem{
						Weapon:        "Kevlar + Helmet",
						Price:         1000,
						EntityID:      0,
						OriginalOwner: "purchased",
					})
				} else {
					startItems = append(startItems, models.AI_WeaponItem{
						Weapon:        "Kevlar Vest",
						Price:         650,
						EntityID:      0,
						OriginalOwner: "purchased",
					})
				}
			}

			// Add defuse kit (CT only)
			if player.HasDefuseKit() {
				startItems = append(startItems, models.AI_WeaponItem{
					Weapon:        "Defuse Kit",
					Price:         400,
					EntityID:      0,
					OriginalOwner: "purchased",
				})
			}

			lossBonus := 1400
			if teamName == "T" {
				lossBonus = calcLossBonus(ctx.TConsecutiveLosses + 1)
			} else if teamName == "CT" {
				lossBonus = calcLossBonus(ctx.CTConsecutiveLosses + 1)
			}
			nextRoundMin := player.Money() + lossBonus

			pData := models.AI_EconomyPlayer{
				SteamID:             player.SteamID64,
				Name:                player.Name,
				Team:                teamName,
				SpawnArea:           spawnArea,
				InitialMoney:        player.Money(),
				NextRoundMinMoney:   nextRoundMin,
				StartRoundItems:     startItems,
				EquipmentValueStart: equipValue,
				SpentInBuy:          0,
				Purchases:           []models.AI_WeaponItem{},
				FinalEquipment:      []models.AI_WeaponItem{}, // Will be filled at FreezetimeEnd
				FinalEquipmentValue: 0,
				FinalMoney:          0,
				EquipmentValueEnd:   0,
				EndEquipment:        []models.AI_WeaponItem{}, // Will be filled at RoundEnd
				Outcome:             "",
				WinReason:           "",
				Survived:            false,
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

		for i := range roundData.Players {
			pData := &roundData.Players[i]

			var player *common.Player
			for _, p := range gs.Participants().All() {
				if p.Name == pData.Name {
					player = p
					break
				}
			}
			if player == nil {
				continue
			}

			pData.Outcome = "loss"
			if pData.Team == winningTeam {
				pData.Outcome = "win"
			}
			pData.WinReason = winReason
			pData.Survived = player.IsAlive()
			pData.FinalMoney = player.Money()
			pData.EquipmentValueEnd = player.EquipmentValueCurrent()
			pData.SpentInBuy = player.MoneySpentThisRound()

			// Capture EndEquipment with EntityID (exclude Knife)
			endEquipment := []models.AI_WeaponItem{}
			for _, weapon := range player.Weapons() {
				if weapon == nil {
					continue
				}
				weaponName := weapon.String()
				if weaponName == "Knife" {
					continue // Skip knife
				}
				// Determine original owner
				originalOwner := "purchased"
				if originalOwnerID, existed := ctx.WeaponOriginalOwner[weapon.UniqueID()]; existed {
					for _, p := range gs.Participants().All() {
						if p.SteamID64 == originalOwnerID {
							originalOwner = p.Name
							break
						}
					}
				}
				endEquipment = append(endEquipment, models.AI_WeaponItem{
					Weapon:        weaponName,
					Price:         getWeaponPrice(weaponName),
					EntityID:      weapon.UniqueID(),
					OriginalOwner: originalOwner,
				})
			}

			// Add armor (Kevlar / Kevlar + Helmet)
			if player.Armor() > 0 {
				if player.HasHelmet() {
					endEquipment = append(endEquipment, models.AI_WeaponItem{
						Weapon:        "Kevlar + Helmet",
						Price:         1000,
						EntityID:      0,
						OriginalOwner: "purchased",
					})
				} else {
					endEquipment = append(endEquipment, models.AI_WeaponItem{
						Weapon:        "Kevlar Vest",
						Price:         650,
						EntityID:      0,
						OriginalOwner: "purchased",
					})
				}
			}

			// Add defuse kit (CT only)
			if player.HasDefuseKit() {
				endEquipment = append(endEquipment, models.AI_WeaponItem{
					Weapon:        "Defuse Kit",
					Price:         400,
					EntityID:      0,
					OriginalOwner: "purchased",
				})
			}

			pData.EndEquipment = endEquipment

			// Add refunds
			if refunds, ok := ctx.PlayerRefunds[player.SteamID64]; ok {
				pData.Refunds = refunds
			}
		}

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

		weaponName := e.Weapon.String()
		weaponValue := getWeaponPrice(weaponName)

		// Only track significant weapon drops (rifles, SMGs, etc.)
		if weaponValue < 500 {
			return
		}

		drop := models.AI_EconomyDrop{
			Tick:         ctx.Parser.CurrentFrame(),
			Dropper:      e.Player.Name,
			DropperID:    e.Player.SteamID64,
			DropperMoney: e.Player.Money(),
			Weapon:       weaponName,
			WeaponValue:  weaponValue,
			PickedUp:     false,
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

		// Only track pickups outside of freezetime (ground pickups)
		if !ctx.FreezeTimeEnded {
			return // Already tracked as purchase
		}

		weaponName := e.Weapon.String()
		weaponValue := getWeaponPrice(weaponName)

		// Only track significant weapons
		if weaponValue < 500 {
			return
		}

		currentTick := ctx.Parser.CurrentFrame()

		// Try to match with a pending drop
		matched := false
		var matchedDrop *models.AI_EconomyDrop
		var matchedIdx int

		for i, drop := range ctx.PendingDrops {
			// Match if weapon name matches and within reasonable tick window
			if drop.Weapon == weaponName && !drop.PickedUp && currentTick-drop.Tick < 640 { // ~10 seconds at 64 tick
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

			gs := ctx.Parser.GameState()
			for _, p := range gs.Participants().All() {
				if p.SteamID64 == matchedDrop.DropperID {
					dropperTeam = int(p.Team)
					break
				}
			}

			if dropperTeam == pickerTeam && matchedDrop.DropperID != e.Player.SteamID64 {
				// Teammate picked up the drop
				matchedDrop.Receiver = e.Player.Name
				matchedDrop.ReceiverID = e.Player.SteamID64
				matchedDrop.ReceiverMoney = e.Player.Money()
				matchedDrop.PickedUp = true

				// Track drops received by player
				ctx.PlayerDropsReceived[e.Player.SteamID64] = append(ctx.PlayerDropsReceived[e.Player.SteamID64], weaponName)

				// Move to RoundDrops
				ctx.RoundDrops = append(ctx.RoundDrops, *matchedDrop)
			}

			// Remove from pending
			ctx.PendingDrops = append(ctx.PendingDrops[:matchedIdx], ctx.PendingDrops[matchedIdx+1:]...)
		} else {
			// Ground pickup (not from a tracked drop)
			pickup := models.AI_EconomyPickup{
				Tick:        currentTick,
				Player:      e.Player.Name,
				PlayerID:    e.Player.SteamID64,
				Weapon:      weaponName,
				WeaponValue: weaponValue,
				FromDrop:    false,
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
		refundValue := getWeaponPrice(weaponName)

		refund := models.AI_EconomyRefund{
			Tick:        ctx.Parser.CurrentFrame(),
			Player:      e.Player.Name,
			PlayerID:    e.Player.SteamID64,
			Weapon:      weaponName,
			RefundValue: refundValue,
		}

		ctx.RoundRefunds = append(ctx.RoundRefunds, refund)
		ctx.PlayerRefunds[e.Player.SteamID64] = append(ctx.PlayerRefunds[e.Player.SteamID64], weaponName)
	})
}

// getWeaponPrice returns the price of a weapon by its name (CS2 prices - December 2025)
func getWeaponPrice(weaponName string) int {
	prices := map[string]int{
		// Pistols
		"Glock-18":      200,
		"USP-S":         200,
		"P2000":         200,
		"Dual Berettas": 300,
		"P250":          300,
		"Tec-9":         500,
		"Five-SeveN":    500,
		"CZ75-Auto":     500,
		"Desert Eagle":  700,
		"R8 Revolver":   600,

		// SMGs
		"MAC-10":   1050,
		"MP9":      1250,
		"MP7":      1500,
		"MP5-SD":   1500,
		"UMP-45":   1200,
		"P90":      2350,
		"PP-Bizon": 1400,

		// Rifles
		"Galil AR": 1800,
		"FAMAS":    1950,
		"AK-47":    2700,
		"M4A4":     2900,
		"M4A1-S":   2900,
		"M4A1":     2900, // Alias for M4A1-S
		"SG 553":   3000,
		"AUG":      3300,
		"SSG 08":   1700,
		"AWP":      4750,
		"G3SG1":    5000,
		"SCAR-20":  5000,

		// Heavy
		"Nova":      1050,
		"XM1014":    2000,
		"Sawed-Off": 1100,
		"MAG-7":     1300,
		"M249":      5200,
		"Negev":     1700,

		// Grenades
		"Flashbang":          200,
		"Smoke Grenade":      300,
		"HE Grenade":         300,
		"Incendiary Grenade": 500,
		"Molotov":            400,
		"Decoy Grenade":      50,

		// Equipment
		"Kevlar Vest":     650,
		"Kevlar + Helmet": 1000,
		"Zeus x27":        200,
		"Defuse Kit":      400,

		// Free items
		"Knife": 0,
		"C4":    0,
	}

	if price, ok := prices[weaponName]; ok {
		return price
	}
	return 0
}
