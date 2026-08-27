package parser

import (
	"crypto/sha256"
	"cs2-demo-service/models"
	economyconfig "cs2-demo-service/pkg/economy"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	block6ParserVersion       = "v16"
	block6QualitySchema       = 12
	block6PriceTableVersion   = economyconfig.PriceTableVersion
	block6EconomyRulesVersion = economyconfig.RulesVersion
	block6RatingVersion       = "stratai.rating_hltv2_approx@1"
	block6ClutchVersion       = "stratai.clutch_ledger@1"
)

const block6RatingFormula = "0.0073*KAST + 0.3591*KPR - 0.5329*DPR + 0.2372*Impact + 0.0032*ADR + 0.1587"

func block6Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func block6Configurations(playedAt *string) (
	models.CanonicalVersionedConfig,
	models.CanonicalVersionedConfig,
	map[string]models.CanonicalVersionedConfig,
	map[string]string,
	map[string]string,
) {
	priceEffectiveFrom := economyconfig.PriceTableEffectiveFrom
	rulesEffectiveFrom := economyconfig.RulesEffectiveFrom
	priceApplicability := "unverified_match_date"
	if playedAt != nil && strings.Split(*playedAt, "T")[0] >= priceEffectiveFrom {
		priceApplicability = "verified_by_played_at"
	} else if playedAt != nil {
		priceApplicability = "unverified_outside_effective_range"
	}
	rulesApplicability := "unverified_match_date"
	if playedAt != nil && strings.Split(*playedAt, "T")[0] >= rulesEffectiveFrom {
		rulesApplicability = "verified_by_played_at"
	} else if playedAt != nil {
		rulesApplicability = "unverified_outside_effective_range"
	}
	price := models.CanonicalVersionedConfig{
		Version: block6PriceTableVersion, ChecksumSHA256: economyconfig.PriceTableChecksum(),
		EffectiveFrom: &priceEffectiveFrom, ApplicabilityStatus: priceApplicability,
		Source: "versioned_stratai_price_table",
	}
	rules := models.CanonicalVersionedConfig{
		Version: block6EconomyRulesVersion, ChecksumSHA256: economyconfig.RulesChecksum(),
		EffectiveFrom: &rulesEffectiveFrom, ApplicabilityStatus: rulesApplicability,
		Source: "valve_loss_bonus_rule_with_explicit_unavailable_rewards",
	}
	algorithmEffectiveFrom := "2026-08-19"
	algorithms := map[string]models.CanonicalVersionedConfig{
		"clutch": {
			Version: block6ClutchVersion, ChecksumSHA256: block6Hash(block6ClutchVersion + "|atomic_kills|stable_teams|round_winner"),
			EffectiveFrom: &algorithmEffectiveFrom, ApplicabilityStatus: "applicable", Source: "canonical_combat_event@2+rounds@2",
		},
		"rating": {
			Version: block6RatingVersion, ChecksumSHA256: block6Hash(block6RatingVersion + "|" + block6RatingFormula),
			EffectiveFrom: &algorithmEffectiveFrom, ApplicabilityStatus: "approximate", Source: "non_official_hltv2_approximation",
		},
		"stats": {
			Version: "stratai.player_stats_ledger@1", ChecksumSHA256: block6Hash("stratai.player_stats_ledger@1|combat@2|utility@2|trades@1"),
			EffectiveFrom: &algorithmEffectiveFrom, ApplicabilityStatus: "applicable", Source: "canonical_atomic_ledgers",
		},
	}
	hashes := map[string]string{
		"economy_rules":    rules.ChecksumSHA256,
		"price_table":      price.ChecksumSHA256,
		"clutch_algorithm": algorithms["clutch"].ChecksumSHA256,
		"rating_algorithm": algorithms["rating"].ChecksumSHA256,
		"stats_algorithm":  algorithms["stats"].ChecksumSHA256,
	}
	transformations := map[string]string{
		"economy":  "stratai.economy_projection@1",
		"stats":    "stratai.player_stats_projection@1",
		"clutch":   block6ClutchVersion,
		"metadata": "stratai.match_metadata_projection@1",
	}
	return price, rules, algorithms, hashes, transformations
}

func canonicalPlayedAt(value string) (*string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, "unavailable"
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			normalized := parsed.UTC().Format(time.RFC3339)
			if layout == "2006-01-02" {
				normalized = parsed.Format("2006-01-02")
			}
			return &normalized, "observed"
		}
	}
	return nil, "unavailable"
}

func buildCanonicalMatchMetadata(
	matchID, playedAt string,
	provenance models.CanonicalExportProvenance,
) models.CanonicalMatchMetadata {
	playedAtValue, playedAtStatus := canonicalPlayedAt(playedAt)
	price, rules, algorithms, hashes, transformations := block6Configurations(playedAtValue)
	var playedAtSource *string
	if playedAtValue != nil && provenance.Source != "" {
		value := provenance.Source
		playedAtSource = &value
	}
	var originDate *string
	originDateStatus := "unavailable"
	if playedAtValue != nil {
		value := strings.Split(*playedAtValue, "T")[0]
		originDate = &value
		originDateStatus = "derived_from_played_at"
	}
	endpoint := optionalString(provenance.Endpoint)
	queriedAt, _ := canonicalPlayedAt(provenance.QueriedAt)
	sourceVersion := optionalString(provenance.SourceVersion)
	warnings := make([]string, 0, 2)
	if playedAtValue == nil {
		warnings = append(warnings, "played_at unavailable: no reliable origin timestamp was supplied")
	}
	if price.ApplicabilityStatus != "verified_by_played_at" {
		warnings = append(warnings, "price table applicability cannot be verified for played_at")
	}
	return models.CanonicalMatchMetadata{
		SchemaID: "stratai.match_metadata@1", MatchID: matchID,
		PlayedAt: playedAtValue, PlayedAtStatus: playedAtStatus, PlayedAtSource: playedAtSource,
		OriginDate: originDate, OriginDateStatus: originDateStatus,
		Source: models.CanonicalSourceProvenance{
			Source: firstNonEmptyString(provenance.Source, "demo"), Endpoint: endpoint,
			QueriedAt: queriedAt, Version: sourceVersion, Checksum: provenance.DemoChecksum,
		},
		ParserVersion: block6ParserVersion, ExportFormatVersion: canonicalExportFormatVersion,
		QualitySchemaVersion: block6QualitySchema, PriceTable: price, EconomyRules: rules,
		Algorithms: algorithms, ConfigurationHashes: hashes, TransformationVersions: transformations,
		Availability: map[string]string{
			"played_at": playedAtStatus, "origin_date": originDateStatus,
			"native_reward_amounts": "unavailable", "processing_timestamp": "excluded_from_canonical_tree",
		},
		Warnings: warnings,
	}
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func observedMoney(amount int, source string) models.CanonicalAvailabilityValue {
	value := amount
	return models.CanonicalAvailabilityValue{Amount: &value, Status: "observed", Source: source}
}

func calculatedMoney(amount int, source string) models.CanonicalAvailabilityValue {
	value := amount
	return models.CanonicalAvailabilityValue{Amount: &value, Status: "calculated", Source: source}
}

func unavailableMoney(status, source string) models.CanonicalAvailabilityValue {
	return models.CanonicalAvailabilityValue{Status: status, Source: source}
}

func canonicalEconomyPrice(item models.AI_WeaponItem) models.CanonicalEconomyPrice {
	status := item.PriceStatus
	if status == "" {
		status = "unknown"
		if item.Price > 0 {
			status = "known"
		}
	}
	var amount *int
	if status == "known" || status == "known_zero" {
		value := item.Price
		amount = &value
	}
	version := item.PriceTableVersion
	if version == "" {
		version = block6PriceTableVersion
	}
	return models.CanonicalEconomyPrice{Amount: amount, Status: status, TableVersion: version}
}

func canonicalEconomyItem(item models.AI_WeaponItem, acquisition string) models.CanonicalEconomyItem {
	observed := item.Weapon
	var purchased *string
	if acquisition == "purchase" {
		purchased = &observed
	}
	var entityID *string
	if item.EntityID != 0 {
		value := strconv.FormatInt(item.EntityID, 10)
		entityID = &value
	}
	var ownerID *string
	if item.OriginalOwnerID != 0 {
		value := canonicalPlayerID(item.OriginalOwnerID)
		ownerID = &value
	}
	ownerStatus := item.OriginalOwnerStatus
	if ownerStatus == "" {
		ownerStatus = "not_observed"
	}
	observation := item.ObservationStatus
	if observation == "" {
		observation = "observed"
	}
	return models.CanonicalEconomyItem{
		ObservedItem: &observed, PurchasedItem: purchased, EntityID: entityID,
		OriginalOwnerID: ownerID, OriginalOwnerState: ownerStatus,
		ObservationStatus: observation, Price: canonicalEconomyPrice(item),
	}
}

func canonicalEconomyInventory(native, calculated int, items []models.AI_WeaponItem) models.CanonicalEconomyInventory {
	if items == nil {
		return models.CanonicalEconomyInventory{
			Status: "not_observed",
			Items:  make([]models.CanonicalEconomyItem, 0),
		}
	}
	nativeValue, calculatedValue := native, calculated
	converted := make([]models.CanonicalEconomyItem, 0, len(items))
	for _, item := range items {
		converted = append(converted, canonicalEconomyItem(item, "inventory"))
	}
	sort.Slice(converted, func(i, j int) bool {
		left, right := "", ""
		if converted[i].ObservedItem != nil {
			left = *converted[i].ObservedItem
		}
		if converted[j].ObservedItem != nil {
			right = *converted[j].ObservedItem
		}
		if left != right {
			return left < right
		}
		return stringValue(converted[i].EntityID) < stringValue(converted[j].EntityID)
	})
	return models.CanonicalEconomyInventory{
		Status: "observed_with_calculated_valuation", NativeValue: &nativeValue,
		CalculatedValue: &calculatedValue, Items: converted,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func buildCanonicalBlock6Economy(
	ctx *models.DemoContext,
	matchID string,
	rosters canonicalRosterInfo,
	rounds []models.CanonicalRound,
	objectiveEvents []models.CanonicalObjectiveEvent,
	combatEvents []models.CanonicalCombatEvent,
	playedAt string,
) (models.CanonicalEconomyRoundExport, models.CanonicalEconomyPlayerExport) {
	playedAtValue, _ := canonicalPlayedAt(playedAt)
	price, rules, _, _, _ := block6Configurations(playedAtValue)
	legacyRounds := append([]models.AI_EconomyRound(nil), ctx.AI_EconomyRounds...)
	sort.Slice(legacyRounds, func(i, j int) bool { return legacyRounds[i].Round < legacyRounds[j].Round })
	nextMoney := make(map[string]int)
	for _, round := range legacyRounds {
		for _, player := range round.Players {
			nextMoney[fmt.Sprintf("%d|%d", round.Round-1, player.SteamID)] = player.InitialMoney
		}
	}
	playerRows := make([]models.CanonicalEconomyPlayer, 0)
	for _, round := range legacyRounds {
		for _, player := range round.Players {
			playerID := canonicalPlayerID(player.SteamID)
			teamID := rosters.playerTeams[player.SteamID]
			side := normalizeSide(player.Team)
			warnings := make([]string, 0)
			freezeObserved := unavailableMoney("not_observed", "freeze_snapshot")
			afterBuyObserved := unavailableMoney("not_observed", "freeze_snapshot")
			if player.MoneyAfterBuyObserved {
				freezeObserved = observedMoney(player.MoneyAfterBuy, "demoinfocs.player_money_after_freeze")
				afterBuyObserved = observedMoney(player.MoneyAfterBuy, "demoinfocs.player_money_after_freeze")
			} else {
				warnings = append(warnings, "money after buy was not observable")
			}
			afterBuyCalculated := calculatedMoney(player.MoneyAfterBuyCalculated, "round_start-spent_in_buy+refunds")
			var moneyDelta *int
			if afterBuyObserved.Amount != nil {
				value := *afterBuyObserved.Amount - player.MoneyAfterBuyCalculated
				moneyDelta = &value
			}
			nextObserved := unavailableMoney("not_observed", "next_round_missing")
			if value, ok := nextMoney[fmt.Sprintf("%d|%d", round.Round, player.SteamID)]; ok {
				nextObserved = observedMoney(value, "next_round.round_start")
			}
			transactions := canonicalPlayerTransactions(matchID, round, player)
			for _, item := range append(append([]models.AI_WeaponItem{}, player.StartRoundItems...), append(player.FinalEquipment, player.EndEquipment...)...) {
				if canonicalEconomyPrice(item).Status == "unknown" {
					warnings = append(warnings, "inventory contains an item with unknown price: "+item.Weapon)
				}
			}
			sort.Strings(warnings)
			playerRows = append(playerRows, models.CanonicalEconomyPlayer{
				RoundID: canonicalRoundID(matchID, round.Round), RoundNumber: round.Round,
				PlayerID: playerID, TeamID: teamID, Side: side, Outcome: player.Outcome, Survived: player.Survived,
				Money: models.CanonicalEconomyMoney{
					RoundStartObserved: observedMoney(player.InitialMoney, "demoinfocs.player_money_round_start"),
					FreezeEndObserved:  freezeObserved, AfterBuyObserved: afterBuyObserved,
					AfterBuyCalculated:    afterBuyCalculated,
					RoundEndObserved:      observedMoney(player.MoneyAtRoundEnd, "demoinfocs.player_money_round_end"),
					NextRoundObserved:     nextObserved,
					NextRoundCalculated:   unavailableMoney("not_evaluable", "incomplete_reward_observation"),
					NativeCalculatedDelta: moneyDelta,
				},
				InventoryStart:  canonicalEconomyInventory(player.EquipmentValueStart, player.EquipmentValueStartCalculated, player.StartRoundItems),
				InventoryFreeze: canonicalEconomyInventory(player.FinalEquipmentValue, player.FinalEquipmentValueCalculated, player.FinalEquipment),
				InventoryEnd:    canonicalEconomyInventory(player.EquipmentValueEndNative, player.EquipmentValueEndCalculated, player.EndEquipment),
				SpentInBuy:      observedMoney(player.SpentInBuy, "demoinfocs.money_spent_this_round"),
				Transactions:    transactions, Warnings: uniqueStrings(warnings),
			})
		}
	}
	sort.Slice(playerRows, func(i, j int) bool {
		if playerRows[i].RoundNumber != playerRows[j].RoundNumber {
			return playerRows[i].RoundNumber < playerRows[j].RoundNumber
		}
		return playerRows[i].PlayerID < playerRows[j].PlayerID
	})
	roundRows := canonicalEconomyTeamRounds(matchID, legacyRounds, playerRows, rosters, rounds, objectiveEvents, combatEvents)
	return models.CanonicalEconomyRoundExport{
			SchemaID: "stratai.economy_round@1", MatchID: matchID, EconomyRules: rules, Rounds: roundRows,
		}, models.CanonicalEconomyPlayerExport{
			SchemaID: "stratai.economy_player@1", MatchID: matchID, PriceTable: price, EconomyRules: rules, Players: playerRows,
		}
}

func canonicalPlayerTransactions(matchID string, round models.AI_EconomyRound, player models.AI_EconomyPlayer) []models.CanonicalEconomyTransaction {
	type candidate struct {
		tick   int
		kind   string
		other  uint64
		item   models.AI_WeaponItem
		amount *int
		status string
		source string
	}
	candidates := make([]candidate, 0)
	for _, item := range player.Purchases {
		value := item.Price
		var amount *int
		if item.PriceStatus == "known" || item.PriceStatus == "known_zero" {
			amount = &value
		}
		candidates = append(candidates, candidate{kind: "purchase", item: item, amount: amount, status: "observed", source: "demoinfocs.ItemPickup_in_buy_zone"})
	}
	if round.Events != nil {
		for _, drop := range round.Events.Drops {
			if drop.DropperID != player.SteamID {
				continue
			}
			item := models.AI_WeaponItem{Weapon: drop.Weapon, Price: drop.WeaponValue, PriceStatus: drop.PriceStatus, PriceTableVersion: drop.PriceTableVersion, EntityID: drop.EntityID, OriginalOwnerID: drop.DropperID, OriginalOwnerStatus: "observed", ObservationStatus: "observed"}
			candidates = append(candidates, candidate{tick: drop.Tick, kind: "drop", other: drop.ReceiverID, item: item, status: "observed", source: "demoinfocs.ItemDrop"})
		}
		for _, pickup := range round.Events.Pickups {
			if pickup.PlayerID != player.SteamID {
				continue
			}
			ownerStatus := "unavailable"
			if pickup.FromPlayerID != 0 {
				ownerStatus = "observed"
			}
			item := models.AI_WeaponItem{Weapon: pickup.Weapon, Price: pickup.WeaponValue, PriceStatus: pickup.PriceStatus, PriceTableVersion: pickup.PriceTableVersion, EntityID: pickup.EntityID, OriginalOwnerID: pickup.FromPlayerID, OriginalOwnerStatus: ownerStatus, ObservationStatus: "observed"}
			candidates = append(candidates, candidate{tick: pickup.Tick, kind: "pickup", other: pickup.FromPlayerID, item: item, status: "observed", source: "demoinfocs.ItemPickup_ground"})
			if pickup.FromDrop && pickup.FromPlayerID != 0 {
				candidates = append(candidates, candidate{tick: pickup.Tick, kind: "exchange", other: pickup.FromPlayerID, item: item, status: "observed", source: "matched_ItemDrop_ItemPickup_entity_id"})
			}
		}
		for _, refund := range round.Events.Refunds {
			if refund.PlayerID != player.SteamID {
				continue
			}
			item := models.AI_WeaponItem{Weapon: refund.Weapon, Price: refund.RefundValue, PriceStatus: refund.PriceStatus, PriceTableVersion: refund.PriceTableVersion, OriginalOwnerStatus: "not_applicable", ObservationStatus: "observed"}
			value := refund.RefundValue
			var amount *int
			if refund.PriceStatus == "known" || refund.PriceStatus == "known_zero" {
				amount = &value
			}
			candidates = append(candidates, candidate{tick: refund.Tick, kind: "refund", item: item, amount: amount, status: "observed", source: "demoinfocs.ItemRefund"})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].tick != candidates[j].tick {
			return candidates[i].tick < candidates[j].tick
		}
		if candidates[i].kind != candidates[j].kind {
			return candidates[i].kind < candidates[j].kind
		}
		if candidates[i].item.EntityID != candidates[j].item.EntityID {
			return candidates[i].item.EntityID < candidates[j].item.EntityID
		}
		return candidates[i].item.Weapon < candidates[j].item.Weapon
	})
	transactions := make([]models.CanonicalEconomyTransaction, 0, len(candidates))
	for index, entry := range candidates {
		transactionID := fmt.Sprintf("%s:economy:%03d:%s:%s:%03d", matchID, round.Round, canonicalPlayerID(player.SteamID), entry.kind, index+1)
		var tick *int
		if entry.tick > 0 {
			value := entry.tick
			tick = &value
		}
		var other *string
		if entry.other != 0 {
			value := canonicalPlayerID(entry.other)
			other = &value
		}
		transactions = append(transactions, models.CanonicalEconomyTransaction{
			TransactionID: transactionID, Type: entry.kind, Tick: tick,
			ActorPlayerID: canonicalPlayerID(player.SteamID), OtherPlayerID: other,
			Item: canonicalEconomyItem(entry.item, entry.kind), Amount: entry.amount,
			Status: entry.status, Source: entry.source, SourceEventIDs: []string{transactionID + ":source"},
		})
	}
	return transactions
}

func canonicalEconomyTeamRounds(
	matchID string,
	legacy []models.AI_EconomyRound,
	players []models.CanonicalEconomyPlayer,
	rosters canonicalRosterInfo,
	rounds []models.CanonicalRound,
	objectiveEvents []models.CanonicalObjectiveEvent,
	combatEvents []models.CanonicalCombatEvent,
) []models.CanonicalEconomyRound {
	playerTeam := make(map[string]string)
	for steamID, teamID := range rosters.playerTeams {
		playerTeam[canonicalPlayerID(steamID)] = teamID
	}
	roundByNumber := make(map[int]models.CanonicalRound, len(rounds))
	for _, round := range rounds {
		roundByNumber[round.RoundNumber] = round
	}
	legacyByNumber := make(map[int]models.AI_EconomyRound, len(legacy))
	for _, round := range legacy {
		legacyByNumber[round.Round] = round
	}
	result := make([]models.CanonicalEconomyRound, 0, len(rounds)*2)
	for _, round := range rounds {
		for _, team := range rosters.teams {
			side := rosters.sides[round.RoundNumber][team.TeamID]
			row := models.CanonicalEconomyRound{
				RoundID: round.RoundID, RoundNumber: round.RoundNumber, TeamID: team.TeamID, Side: side,
				Outcome: "unknown", WinReason: round.WinReason,
				Rewards: make([]models.CanonicalEconomyReward, 0), Diagnostics: map[string]int{},
			}
			if round.WinnerTeamID != nil {
				if *round.WinnerTeamID == team.TeamID {
					row.Outcome = "win"
				} else {
					row.Outcome = "loss"
				}
			}
			legacyTeam := legacyByNumber[round.RoundNumber].Teams[strings.ToUpper(side)]
			row.LossBonus = models.CanonicalLossBonus{
				Level: lossBonusLevel(legacyTeam.LossBonus), Amount: legacyTeam.LossBonus,
				Status: "calculated", RulesVersion: block6EconomyRulesVersion,
			}
			start, freeze, end := 0, 0, 0
			freezeObserved := true
			for _, player := range players {
				if player.RoundNumber != round.RoundNumber || player.TeamID != team.TeamID {
					continue
				}
				start += intValue(player.Money.RoundStartObserved.Amount)
				end += intValue(player.Money.RoundEndObserved.Amount)
				if player.Money.FreezeEndObserved.Amount == nil {
					freezeObserved = false
				} else {
					freeze += *player.Money.FreezeEndObserved.Amount
				}
				if player.Money.NativeCalculatedDelta != nil && *player.Money.NativeCalculatedDelta != 0 {
					row.Diagnostics["money_after_buy_native_calculated_differences"]++
				}
			}
			row.MoneyStart = observedMoney(start, "sum(economy_player.money.round_start_observed)")
			row.MoneyRoundEnd = observedMoney(end, "sum(economy_player.money.round_end_observed)")
			if freezeObserved {
				row.MoneyFreezeEnd = observedMoney(freeze, "sum(economy_player.money.freeze_end_observed)")
			} else {
				row.MoneyFreezeEnd = unavailableMoney("not_observed", "incomplete_player_freeze_observation")
			}
			row.Rewards = canonicalTeamRewards(matchID, round, team.TeamID, side, row.Outcome, row.LossBonus, playerTeam, objectiveEvents, combatEvents)
			result = append(result, row)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].RoundNumber != result[j].RoundNumber {
			return result[i].RoundNumber < result[j].RoundNumber
		}
		return result[i].TeamID < result[j].TeamID
	})
	return result
}

func canonicalTeamRewards(
	matchID string,
	round models.CanonicalRound,
	teamID, side, outcome string,
	lossBonus models.CanonicalLossBonus,
	playerTeam map[string]string,
	objectiveEvents []models.CanonicalObjectiveEvent,
	combatEvents []models.CanonicalCombatEvent,
) []models.CanonicalEconomyReward {
	rewards := make([]models.CanonicalEconomyReward, 0)
	baseID := fmt.Sprintf("%s:reward:%03d:%s", matchID, round.RoundNumber, teamID)
	if outcome == "loss" {
		amount := lossBonus.Amount
		rewards = append(rewards, models.CanonicalEconomyReward{RewardID: baseID + ":loss", Type: "loss_bonus", CalculatedAmount: &amount, Status: "calculated", SourceEventIDs: []string{round.RoundID}})
	} else if outcome == "win" {
		rewards = append(rewards, models.CanonicalEconomyReward{RewardID: baseID + ":win", Type: "round_win", Status: "unavailable", SourceEventIDs: []string{round.RoundID}})
	}
	for _, event := range objectiveEvents {
		if event.RoundNumber != round.RoundNumber || (event.EventType != "bomb_plant" && event.EventType != "bomb_explode" && event.EventType != "bomb_defuse") {
			continue
		}
		eventTeam := ""
		if event.ActorPlayerID != nil {
			eventTeam = playerTeam[*event.ActorPlayerID]
		}
		if eventTeam == "" && event.ActorSide != nil && normalizeSide(*event.ActorSide) == side {
			eventTeam = teamID
		}
		if eventTeam != teamID {
			continue
		}
		typeName := strings.TrimPrefix(event.EventType, "bomb_")
		rewards = append(rewards, models.CanonicalEconomyReward{RewardID: baseID + ":" + typeName + ":" + event.EventID, Type: typeName, PlayerID: event.ActorPlayerID, Status: "unavailable", SourceEventIDs: []string{event.EventID}})
	}
	for _, event := range combatEvents {
		if event.RoundNumber != round.RoundNumber || !event.IsKill || event.ActorPlayerID == nil || playerTeam[*event.ActorPlayerID] != teamID || event.Relation != "enemy" {
			continue
		}
		rewards = append(rewards, models.CanonicalEconomyReward{RewardID: baseID + ":kill:" + event.EventID, Type: "kill", PlayerID: event.ActorPlayerID, Status: "unavailable", SourceEventIDs: []string{event.EventID}})
	}
	sort.Slice(rewards, func(i, j int) bool { return rewards[i].RewardID < rewards[j].RewardID })
	return rewards
}

func lossBonusLevel(amount int) int {
	switch amount {
	case 1400:
		return 0
	case 1900:
		return 1
	case 2400:
		return 2
	case 2900:
		return 3
	case 3400:
		return 4
	default:
		return -1
	}
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func buildCanonicalClutchEvents(
	matchID string,
	rounds []models.CanonicalRound,
	participants []models.CanonicalParticipant,
	combatEvents []models.CanonicalCombatEvent,
) models.CanonicalClutchEventExport {
	_, _, algorithms, _, _ := block6Configurations(nil)
	teamPlayers := make(map[string][]string)
	playerTeam := make(map[string]string)
	for _, participant := range participants {
		teamPlayers[participant.TeamID] = append(teamPlayers[participant.TeamID], participant.PlayerID)
		playerTeam[participant.PlayerID] = participant.TeamID
	}
	for teamID := range teamPlayers {
		sort.Strings(teamPlayers[teamID])
	}
	roundMap := make(map[int]models.CanonicalRound, len(rounds))
	for _, round := range rounds {
		roundMap[round.RoundNumber] = round
	}
	eventsByRound := make(map[int][]models.CanonicalCombatEvent)
	for _, event := range combatEvents {
		if event.IsKill && event.TargetPlayerID != nil {
			eventsByRound[event.RoundNumber] = append(eventsByRound[event.RoundNumber], event)
		}
	}
	result := make([]models.CanonicalClutchEvent, 0)
	for roundNumber, events := range eventsByRound {
		sort.Slice(events, func(i, j int) bool {
			if events[i].Tick != events[j].Tick {
				return events[i].Tick < events[j].Tick
			}
			if events[i].SequenceInTick != events[j].SequenceInTick {
				return events[i].SequenceInTick < events[j].SequenceInTick
			}
			return events[i].EventID < events[j].EventID
		})
		alive := make(map[string]map[string]bool)
		for teamID, players := range teamPlayers {
			alive[teamID] = make(map[string]bool, len(players))
			for _, playerID := range players {
				alive[teamID][playerID] = true
			}
		}
		attempted := make(map[string]bool)
		for eventIndex, event := range events {
			victimTeam := playerTeam[*event.TargetPlayerID]
			if victimTeam != "" {
				delete(alive[victimTeam], *event.TargetPlayerID)
			}
			teamIDs := sortedMapKeys(alive)
			for _, teamID := range teamIDs {
				if attempted[teamID] || len(alive[teamID]) != 1 {
					continue
				}
				enemies := 0
				for otherTeam, players := range alive {
					if otherTeam != teamID {
						enemies += len(players)
					}
				}
				if enemies < 1 || enemies > 5 {
					continue
				}
				playerID := onlyAlivePlayer(alive[teamID])
				round := roundMap[roundNumber]
				resultValue, evaluation := "not_evaluable", "not_evaluable"
				if round.WinnerTeamID != nil {
					resultValue, evaluation = "lost", "evaluated"
					if *round.WinnerTeamID == teamID {
						resultValue = "won"
					}
				}
				sources := make([]string, 0, len(events)-eventIndex)
				for _, source := range events[eventIndex:] {
					sources = append(sources, source.EventID)
				}
				side := sideForTeam(round, teamID)
				result = append(result, models.CanonicalClutchEvent{
					ClutchID: fmt.Sprintf("%s:clutch:%03d:%s", matchID, roundNumber, teamID),
					RoundID:  round.RoundID, RoundNumber: roundNumber, PlayerID: playerID,
					TeamID: teamID, Side: side, EnemiesAtStart: enemies,
					State: fmt.Sprintf("1v%d", enemies), Attempt: true, Result: resultValue,
					StartTick: event.Tick, TriggerEventID: event.EventID, SourceEventIDs: sources,
					OutcomeSource: "rounds.winner_team_id", EvaluationStatus: evaluation,
				})
				attempted[teamID] = true
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].RoundNumber != result[j].RoundNumber {
			return result[i].RoundNumber < result[j].RoundNumber
		}
		return result[i].TeamID < result[j].TeamID
	})
	return models.CanonicalClutchEventExport{SchemaID: "stratai.clutch_event@1", MatchID: matchID, Algorithm: algorithms["clutch"], ClutchEvents: result}
}

func sideForTeam(round models.CanonicalRound, teamID string) string {
	for _, assignment := range round.SideAssignments {
		if assignment.TeamID == teamID {
			return assignment.Side
		}
	}
	return "unknown"
}

func onlyAlivePlayer(players map[string]bool) string {
	for playerID := range players {
		return playerID
	}
	return ""
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func buildCanonicalBlock6PlayerStats(
	matchID string,
	rosters canonicalRosterInfo,
	legacy models.CanonicalPlayerMatchStatsExport,
	clutches models.CanonicalClutchEventExport,
) models.CanonicalPlayerStatsExport {
	clutchByPlayer := make(map[string][]models.CanonicalClutchEvent)
	for _, event := range clutches.ClutchEvents {
		clutchByPlayer[event.PlayerID] = append(clutchByPlayer[event.PlayerID], event)
	}
	players := make([]models.CanonicalPlayerStats, 0, len(legacy.Players))
	for _, record := range legacy.Players {
		metrics := record.Metrics
		steamID, _ := strconv.ParseUint(strings.TrimPrefix(record.PlayerID, "steam:"), 10, 64)
		teamID := rosters.playerTeams[steamID]
		summary := clutchSummary(clutchByPlayer[record.PlayerID])
		metrics.ClutchAttempts = summary.Attempts
		metrics.ClutchWins = summary.Wins
		metrics.ClutchLosses = summary.Losses
		metrics.ClutchNotEvaluable = summary.NotEvaluable
		metrics.Clutches1v1Won = clutchWinsForState(clutchByPlayer[record.PlayerID], "1v1")
		metrics.Clutches1v2Won = clutchWinsForState(clutchByPlayer[record.PlayerID], "1v2")
		metrics.Clutches1v3Won = clutchWinsForState(clutchByPlayer[record.PlayerID], "1v3")
		metrics.Clutches1v4Won = clutchWinsForState(clutchByPlayer[record.PlayerID], "1v4")
		metrics.Clutches1v5Won = clutchWinsForState(clutchByPlayer[record.PlayerID], "1v5")
		nativeStatus := metrics.NativeScoreboardStatus
		var native *models.AI_NativePlayerStats
		var killsDelta, deathsDelta, assistsDelta, damageDelta, utilityDelta *int
		if nativeStatus == "observed" {
			value := metrics.NativeScoreboard
			native = &value
			killsDelta = block6IntPointer(value.Kills - metrics.KillsObserved)
			deathsDelta = block6IntPointer(value.Deaths - metrics.DeathsObserved)
			assistsDelta = block6IntPointer(value.Assists - metrics.AssistsObserved)
			damageDelta = block6IntPointer(metrics.CombatDamageUnattributedDelta)
			utilityDelta = block6IntPointer(value.UtilityDamage - metrics.UtilityDamageObserved)
		} else {
			nativeStatus = "unavailable"
		}
		var kd *float64
		if metrics.DeathsObserved > 0 {
			value := float64(metrics.KillsObserved) / float64(metrics.DeathsObserved)
			kd = &value
		}
		var ratingValue *float64
		ratingStatus := "not_evaluable"
		if metrics.RoundsPlayed > 0 {
			value := metrics.HLTVRating
			ratingValue = &value
			ratingStatus = "calculated"
		}
		players = append(players, models.CanonicalPlayerStats{
			PlayerID: record.PlayerID, TeamID: teamID, NativeScoreboardStatus: nativeStatus, NativeScoreboard: native,
			Derived: models.CanonicalStatsDerived{
				KillsObserved: metrics.KillsObserved, DeathsObserved: metrics.DeathsObserved,
				AssistsObserved: metrics.AssistsObserved, CombatDamageObserved: metrics.CombatDamageObserved,
				UtilityDamageObserved: metrics.UtilityDamageObserved, GrenadeDamageObserved: metrics.GrenadeDamage,
				TradeKills: metrics.TradeKills, TradedDeaths: metrics.TradedDeaths,
				OpeningDuelsAttempted: metrics.OpeningDuelsAttempted, OpeningDuelsWon: metrics.OpeningDuelsWon,
				OpeningDuelsLost: metrics.OpeningDuelsLost, KDRatioObserved: kd,
			},
			Reconciliation: models.CanonicalStatsReconciliation{
				KillsNativeMinusObserved: killsDelta, DeathsNativeMinusObserved: deathsDelta,
				AssistsNativeMinusObserved: assistsDelta, DamageNativeMinusObserved: damageDelta,
				UtilityNativeMinusObserved: utilityDelta,
			},
			Clutch: summary,
			Rating: models.CanonicalApproximateRating{
				Value: ratingValue, Status: ratingStatus, Approximate: true,
				AlgorithmVersion: block6RatingVersion, Formula: block6RatingFormula,
				Source: "non_official_hltv2_approximation",
			},
			Metrics: metrics,
			Provenance: map[string]string{
				"native_scoreboard": "demoinfocs_final_scoreboard",
				"combat":            "stratai.combat_event@2", "utility": "stratai.utility_event@2",
				"trades": "stratai.trades@1", "clutch": "stratai.clutch_event@1",
				"algorithm": "stratai.player_stats_ledger@1",
			},
		})
	}
	sort.Slice(players, func(i, j int) bool { return players[i].PlayerID < players[j].PlayerID })
	return models.CanonicalPlayerStatsExport{SchemaID: "stratai.player_stats@1", MatchID: matchID, Players: players}
}

func block6IntPointer(value int) *int { return &value }

func clutchSummary(events []models.CanonicalClutchEvent) models.CanonicalClutchSummary {
	summary := models.CanonicalClutchSummary{ByState: map[string]int{"1v1": 0, "1v2": 0, "1v3": 0, "1v4": 0, "1v5": 0}}
	for _, event := range events {
		summary.Attempts++
		summary.ByState[event.State]++
		switch event.Result {
		case "won":
			summary.Wins++
		case "lost":
			summary.Losses++
		default:
			summary.NotEvaluable++
		}
	}
	return summary
}

func clutchWinsForState(events []models.CanonicalClutchEvent, state string) int {
	wins := 0
	for _, event := range events {
		if event.State == state && event.Result == "won" {
			wins++
		}
	}
	return wins
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}
