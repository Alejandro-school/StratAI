package parser

import (
	"cs2-demo-service/models"
	economyconfig "cs2-demo-service/pkg/economy"
	"fmt"
	"sort"
	"strings"
)

type block6QualityAssessment struct {
	teamIdentity                int
	nativeCalculatedViolations  int
	nativeCalculatedDifferences int
	moneyTransitions            int
	purchaseProvenance          int
	priceTableVersion           int
	scoreboardReconciliation    int
	utilityReconciliation       int
	clutchReconciliation        int
	warmupContamination         int
	metadataProvenance          int
	metadataChecksumLineage     int
	economyDeterminism          int
	statsDeterminism            int
	observationWarnings         int
	details                     []string
}

func assessBlock6Quality(
	ctx *models.DemoContext,
	rounds []models.CanonicalRound,
	participants []models.CanonicalParticipant,
	combatEvents []models.CanonicalCombatEvent,
	economyRounds models.CanonicalEconomyRoundExport,
	economyPlayers models.CanonicalEconomyPlayerExport,
	stats models.CanonicalPlayerStatsExport,
	clutches models.CanonicalClutchEventExport,
	metadata models.CanonicalMatchMetadata,
) block6QualityAssessment {
	assessment := block6QualityAssessment{}
	teamIDs := make(map[string]struct{})
	playerTeams := make(map[string]string)
	for _, participant := range participants {
		teamIDs[participant.TeamID] = struct{}{}
		playerTeams[participant.PlayerID] = participant.TeamID
		if participant.TeamID == "ct" || participant.TeamID == "t" || participant.TeamID == "CT" || participant.TeamID == "T" || participant.TeamID == "" {
			assessment.teamIdentity++
		}
	}
	sidesByRound := make(map[string]string)
	for _, round := range rounds {
		for _, assignment := range round.SideAssignments {
			sidesByRound[fmt.Sprintf("%d|%s", round.RoundNumber, assignment.TeamID)] = assignment.Side
		}
	}
	seenTransactionIDs := make(map[string]struct{})
	for _, row := range economyPlayers.Players {
		if _, ok := teamIDs[row.TeamID]; !ok || playerTeams[row.PlayerID] != row.TeamID || !validCanonicalSide(row.Side) ||
			sidesByRound[fmt.Sprintf("%d|%s", row.RoundNumber, row.TeamID)] != row.Side {
			assessment.teamIdentity++
		}
		if row.RoundNumber < 1 || row.RoundNumber > ctx.CurrentRound {
			assessment.warmupContamination++
		}
		if row.Money.AfterBuyObserved.Amount != nil && row.Money.AfterBuyCalculated.Amount != nil {
			if *row.Money.AfterBuyObserved.Amount != *row.Money.AfterBuyCalculated.Amount {
				assessment.nativeCalculatedDifferences++
			}
			if row.Money.NativeCalculatedDelta == nil || *row.Money.NativeCalculatedDelta != *row.Money.AfterBuyObserved.Amount-*row.Money.AfterBuyCalculated.Amount {
				assessment.nativeCalculatedViolations++
			}
		}
		for _, inventory := range []models.CanonicalEconomyInventory{row.InventoryStart, row.InventoryFreeze, row.InventoryEnd} {
			if inventory.Status == "not_observed" {
				if inventory.NativeValue != nil || inventory.CalculatedValue != nil || len(inventory.Items) != 0 {
					assessment.nativeCalculatedViolations++
				}
				assessment.observationWarnings++
			} else if inventory.Status != "observed_with_calculated_valuation" || inventory.NativeValue == nil || inventory.CalculatedValue == nil {
				assessment.nativeCalculatedViolations++
			}
			for _, item := range inventory.Items {
				assessment.validatePrice(item.Price)
				if item.OriginalOwnerState == "unavailable" || item.OriginalOwnerState == "not_observed" {
					assessment.observationWarnings++
				}
			}
		}
		for _, value := range []*int{
			row.Money.RoundStartObserved.Amount, row.Money.FreezeEndObserved.Amount,
			row.Money.AfterBuyObserved.Amount, row.Money.AfterBuyCalculated.Amount,
			row.Money.RoundEndObserved.Amount, row.Money.NextRoundObserved.Amount,
			row.Money.NextRoundCalculated.Amount,
		} {
			if value != nil && (*value < 0 || *value > 16000) {
				assessment.moneyTransitions++
			}
		}
		refunds := 0
		for _, transaction := range row.Transactions {
			if transaction.TransactionID == "" {
				assessment.economyDeterminism++
			} else if _, duplicate := seenTransactionIDs[transaction.TransactionID]; duplicate {
				assessment.economyDeterminism++
			}
			seenTransactionIDs[transaction.TransactionID] = struct{}{}
			if transaction.ActorPlayerID != row.PlayerID || transaction.Source == "" || len(transaction.SourceEventIDs) == 0 {
				assessment.purchaseProvenance++
			}
			if transaction.Type == "purchase" && transaction.Item.PurchasedItem == nil {
				assessment.purchaseProvenance++
			}
			if transaction.Type == "pickup" && transaction.Item.OriginalOwnerState == "observed" && transaction.OtherPlayerID == nil {
				assessment.purchaseProvenance++
			}
			if transaction.Type == "refund" && transaction.Amount != nil {
				refunds += *transaction.Amount
			}
			assessment.validatePrice(transaction.Item.Price)
		}
		if row.Money.AfterBuyCalculated.Amount != nil && row.Money.RoundStartObserved.Amount != nil && row.SpentInBuy.Amount != nil {
			expected := *row.Money.RoundStartObserved.Amount - *row.SpentInBuy.Amount + refunds
			if expected < 0 {
				expected = 0
			}
			if expected > 16000 {
				expected = 16000
			}
			if *row.Money.AfterBuyCalculated.Amount != expected {
				assessment.moneyTransitions++
			}
		}
		if row.Money.FreezeEndObserved.Amount == nil {
			assessment.observationWarnings++
		}
	}
	seenRewardIDs := make(map[string]struct{})
	for _, row := range economyRounds.Rounds {
		if _, ok := teamIDs[row.TeamID]; !ok || !validCanonicalSide(row.Side) ||
			sidesByRound[fmt.Sprintf("%d|%s", row.RoundNumber, row.TeamID)] != row.Side {
			assessment.teamIdentity++
		}
		if row.RoundNumber < 1 || row.RoundNumber > ctx.CurrentRound {
			assessment.warmupContamination++
		}
		for _, reward := range row.Rewards {
			if reward.RewardID == "" {
				assessment.economyDeterminism++
			} else if _, duplicate := seenRewardIDs[reward.RewardID]; duplicate {
				assessment.economyDeterminism++
			}
			seenRewardIDs[reward.RewardID] = struct{}{}
			if reward.ObservedAmount != nil && reward.CalculatedAmount != nil && reward.Status != "reconciled" {
				assessment.nativeCalculatedViolations++
			}
		}
	}
	if economyPlayers.PriceTable.Version != block6PriceTableVersion ||
		economyPlayers.PriceTable.ChecksumSHA256 != economyconfig.PriceTableChecksum() ||
		economyPlayers.EconomyRules.Version != block6EconomyRulesVersion ||
		economyPlayers.EconomyRules.ChecksumSHA256 != economyconfig.RulesChecksum() ||
		economyRounds.EconomyRules.Version != block6EconomyRulesVersion ||
		economyRounds.EconomyRules.ChecksumSHA256 != economyconfig.RulesChecksum() {
		assessment.priceTableVersion++
	}
	for index := 1; index < len(economyPlayers.Players); index++ {
		left, right := economyPlayers.Players[index-1], economyPlayers.Players[index]
		if left.RoundNumber > right.RoundNumber || left.RoundNumber == right.RoundNumber && left.PlayerID >= right.PlayerID {
			assessment.economyDeterminism++
		}
	}
	for index := 1; index < len(economyRounds.Rounds); index++ {
		left, right := economyRounds.Rounds[index-1], economyRounds.Rounds[index]
		if left.RoundNumber > right.RoundNumber || left.RoundNumber == right.RoundNumber && left.TeamID >= right.TeamID {
			assessment.economyDeterminism++
		}
	}
	for index, player := range stats.Players {
		if index > 0 && stats.Players[index-1].PlayerID >= player.PlayerID {
			assessment.statsDeterminism++
		}
		if player.TeamID == "" || playerTeams[player.PlayerID] != player.TeamID {
			assessment.teamIdentity++
		}
		if player.NativeScoreboardStatus == "observed" {
			if player.NativeScoreboard == nil || player.Metrics.Kills != player.NativeScoreboard.Kills ||
				player.Metrics.Deaths != player.NativeScoreboard.Deaths || player.Metrics.Assists != player.NativeScoreboard.Assists ||
				player.Metrics.TotalDamage != player.NativeScoreboard.TotalDamage || player.Metrics.UtilityDamage != player.NativeScoreboard.UtilityDamage {
				assessment.scoreboardReconciliation++
			}
		} else if player.NativeScoreboard != nil {
			assessment.scoreboardReconciliation++
		}
		if player.Derived.UtilityDamageObserved != player.Metrics.UtilityDamageObserved ||
			player.NativeScoreboard != nil && player.NativeScoreboard.UtilityDamage != player.Metrics.UtilityDamage {
			assessment.utilityReconciliation++
		}
		if !player.Rating.Approximate || player.Rating.AlgorithmVersion != block6RatingVersion || player.Rating.Formula == "" {
			assessment.statsDeterminism++
		}
	}
	assessment.clutchReconciliation += validateClutchReconciliation(rounds, participants, combatEvents, stats, clutches)
	for _, event := range combatEvents {
		if event.RoundNumber < 1 || event.RoundNumber > ctx.CurrentRound {
			assessment.warmupContamination++
		}
	}
	if metadata.SchemaID != "stratai.match_metadata@1" || metadata.Source.Source == "" || metadata.ParserVersion != block6ParserVersion ||
		metadata.ExportFormatVersion != canonicalExportFormatVersion || metadata.QualitySchemaVersion != block6QualitySchema ||
		len(metadata.ConfigurationHashes) < 5 || len(metadata.TransformationVersions) < 4 ||
		metadata.PriceTable.Version != block6PriceTableVersion || metadata.PriceTable.ChecksumSHA256 != economyconfig.PriceTableChecksum() ||
		metadata.EconomyRules.Version != block6EconomyRulesVersion || metadata.EconomyRules.ChecksumSHA256 != economyconfig.RulesChecksum() ||
		metadata.Algorithms["clutch"].Version != block6ClutchVersion || metadata.Algorithms["rating"].Version != block6RatingVersion ||
		metadata.Algorithms["stats"].Version != "stratai.player_stats_ledger@1" {
		assessment.metadataProvenance++
	}
	if metadata.PlayedAt == nil && (metadata.PlayedAtStatus != "unavailable" || metadata.PlayedAtSource != nil || metadata.OriginDate != nil) ||
		metadata.PlayedAt != nil && (metadata.PlayedAtStatus != "observed" || metadata.PlayedAtSource == nil || *metadata.PlayedAtSource != metadata.Source.Source) {
		assessment.metadataProvenance++
	}
	if !validSHA256(metadata.Source.Checksum) {
		assessment.metadataChecksumLineage++
	}
	for _, checksum := range metadata.ConfigurationHashes {
		if !validSHA256(checksum) {
			assessment.metadataChecksumLineage++
		}
	}
	return assessment
}

func (assessment *block6QualityAssessment) validatePrice(price models.CanonicalEconomyPrice) {
	if price.TableVersion != block6PriceTableVersion {
		assessment.priceTableVersion++
	}
	switch price.Status {
	case "known":
		if price.Amount == nil || *price.Amount <= 0 {
			assessment.priceTableVersion++
		}
	case "known_zero":
		if price.Amount == nil || *price.Amount != 0 {
			assessment.priceTableVersion++
		}
	case "unknown":
		if price.Amount != nil {
			assessment.priceTableVersion++
		}
		assessment.observationWarnings++
	default:
		assessment.priceTableVersion++
	}
}

func validateClutchReconciliation(
	rounds []models.CanonicalRound,
	participants []models.CanonicalParticipant,
	combatEvents []models.CanonicalCombatEvent,
	stats models.CanonicalPlayerStatsExport,
	clutches models.CanonicalClutchEventExport,
) int {
	errors := 0
	expectedEvents := buildCanonicalClutchEvents("quality", rounds, participants, combatEvents).ClutchEvents
	expectedByKey := make(map[string]models.CanonicalClutchEvent, len(expectedEvents))
	for _, event := range expectedEvents {
		expectedByKey[fmt.Sprintf("%d|%s", event.RoundNumber, event.TeamID)] = event
	}
	combatIDs := make(map[string]struct{}, len(combatEvents))
	for _, event := range combatEvents {
		combatIDs[event.EventID] = struct{}{}
	}
	roundWinners := make(map[int]*string)
	for _, round := range rounds {
		roundWinners[round.RoundNumber] = round.WinnerTeamID
	}
	seen := make(map[string]struct{})
	byPlayer := make(map[string][]models.CanonicalClutchEvent)
	for _, event := range clutches.ClutchEvents {
		key := fmt.Sprintf("%d|%s", event.RoundNumber, event.TeamID)
		if _, duplicate := seen[key]; duplicate {
			errors++
		}
		seen[key] = struct{}{}
		if !event.Attempt || event.PlayerID == "" || event.EnemiesAtStart < 1 || event.EnemiesAtStart > 5 || event.State != fmt.Sprintf("1v%d", event.EnemiesAtStart) || len(event.SourceEventIDs) == 0 || event.TriggerEventID == "" {
			errors++
		}
		expected, ok := expectedByKey[key]
		if !ok || event.PlayerID != expected.PlayerID || event.EnemiesAtStart != expected.EnemiesAtStart ||
			event.State != expected.State || event.StartTick != expected.StartTick || event.TriggerEventID != expected.TriggerEventID {
			errors++
		}
		triggerLinked := false
		for _, sourceID := range event.SourceEventIDs {
			if sourceID == event.TriggerEventID {
				triggerLinked = true
			}
			if _, ok := combatIDs[sourceID]; !ok {
				errors++
			}
		}
		if !triggerLinked {
			errors++
		}
		winner := roundWinners[event.RoundNumber]
		if winner == nil {
			if event.Result != "not_evaluable" || event.EvaluationStatus != "not_evaluable" {
				errors++
			}
		} else if *winner == event.TeamID && event.Result != "won" || *winner != event.TeamID && event.Result != "lost" {
			errors++
		}
		byPlayer[event.PlayerID] = append(byPlayer[event.PlayerID], event)
	}
	if len(seen) != len(expectedByKey) {
		errors++
	}
	for _, player := range stats.Players {
		expected := clutchSummary(byPlayer[player.PlayerID])
		if player.Clutch.Attempts != expected.Attempts || player.Clutch.Wins != expected.Wins || player.Clutch.Losses != expected.Losses || player.Clutch.NotEvaluable != expected.NotEvaluable {
			errors++
		}
	}
	return errors
}

func validCanonicalSide(side string) bool {
	return side == "ct" || side == "t"
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func (report *qualityReport) applyBlock6Quality(assessment block6QualityAssessment) {
	report.SchemaVersion = block6QualitySchema
	report.EconomyTeamIdentityViolations = assessment.teamIdentity
	report.EconomyNativeCalculatedViolations = assessment.nativeCalculatedViolations
	report.EconomyNativeCalculatedDifferences = assessment.nativeCalculatedDifferences
	report.EconomyMoneyTransitionViolations = assessment.moneyTransitions
	report.EconomyPurchaseProvenanceViolations = assessment.purchaseProvenance
	report.EconomyPriceTableVersionViolations = assessment.priceTableVersion
	report.StatsScoreboardReconciliationErrors = assessment.scoreboardReconciliation
	report.StatsUtilityReconciliationErrors = assessment.utilityReconciliation
	report.ClutchAttemptReconciliationErrors = assessment.clutchReconciliation
	report.WarmupContaminationViolations = assessment.warmupContamination
	report.MetadataProvenanceViolations = assessment.metadataProvenance
	report.MetadataChecksumLineageViolations = assessment.metadataChecksumLineage
	report.EconomyDeterminismViolations = assessment.economyDeterminism
	report.StatsDeterminismViolations = assessment.statsDeterminism
	report.EconomyObservationWarnings = assessment.observationWarnings
	type gate struct {
		name    string
		value   int
		message string
	}
	gates := []gate{
		{"economy_team_identity", assessment.teamIdentity, "Economy must use stable team IDs and round-local CT/T sides."},
		{"economy_native_calculated_reconciliation", assessment.nativeCalculatedViolations, "Native and calculated economy values must remain separate and reconciled."},
		{"economy_money_transition", assessment.moneyTransitions, "Money transitions must be possible and preserve unavailable rewards."},
		{"economy_purchase_provenance", assessment.purchaseProvenance, "Purchases, pickups, drops, refunds and exchanges require player-ID provenance."},
		{"economy_price_table_version", assessment.priceTableVersion, "Every known or unknown price must use the versioned price contract."},
		{"stats_scoreboard_reconciliation", assessment.scoreboardReconciliation, "Public native statistics must match the preserved scoreboard."},
		{"stats_utility_reconciliation", assessment.utilityReconciliation, "Native and observed utility damage must remain separate and reconciled."},
		{"clutch_attempt_reconciliation", assessment.clutchReconciliation, "Clutch attempts and outcomes must reconcile with atomic kills and round winners."},
		{"warmup_contamination", assessment.warmupContamination, "Competitive statistics must not contain warmup or out-of-round events."},
		{"metadata_provenance", assessment.metadataProvenance, "Match metadata must declare source, versions, configuration and availability."},
		{"metadata_checksum_lineage", assessment.metadataChecksumLineage, "Demo and configuration checksums must be valid SHA-256 lineage."},
		{"economy_determinism", assessment.economyDeterminism, "Economy records and IDs must have deterministic ordering."},
		{"stats_determinism", assessment.statsDeterminism, "Statistics records and algorithms must have deterministic ordering."},
	}
	for _, item := range gates {
		status := "pass"
		if item.value > 0 {
			status = "fail"
			report.block6FailureDetails = append(report.block6FailureDetails, fmt.Sprintf("%s=%d", item.name, item.value))
			report.Warnings = append(report.Warnings, item.message)
		}
		report.Checks = append(report.Checks, qualityCheck{Name: item.name, Status: status, Expected: "0", Actual: fmt.Sprint(item.value), Message: item.message})
	}
	reconciliationStatus := "pass"
	if assessment.nativeCalculatedDifferences > 0 {
		reconciliationStatus = "warning"
		report.Warnings = append(report.Warnings, "Native and calculated economy values differ; both are preserved with deltas.")
	}
	report.Checks = append(report.Checks, qualityCheck{Name: "economy_native_calculated_differences", Status: reconciliationStatus, Expected: "diagnostic", Actual: fmt.Sprint(assessment.nativeCalculatedDifferences), Message: "Differences are diagnostics and never overwrite native values."})
	coverageStatus := "pass"
	if assessment.observationWarnings > 0 {
		coverageStatus = "warning"
		report.Warnings = append(report.Warnings, "Some economy prices, owners or freeze observations are unavailable and remain explicit.")
	}
	report.Checks = append(report.Checks, qualityCheck{Name: "economy_observation_coverage", Status: coverageStatus, Expected: "explicit", Actual: fmt.Sprint(assessment.observationWarnings), Message: "Unavailable economy observations must remain null with an explicit status."})
	sort.Slice(report.Checks, func(i, j int) bool { return report.Checks[i].Name < report.Checks[j].Name })
	report.Warnings = uniqueStrings(report.Warnings)
	if report.hasHardBlock6Failure() {
		report.Status = "fail"
		report.UsableForTraining = false
	} else if len(report.Warnings) > 0 {
		report.Status = "warning"
	}
}

func (report qualityReport) hasHardBlock6Failure() bool {
	return report.EconomyTeamIdentityViolations > 0 ||
		report.EconomyNativeCalculatedViolations > 0 ||
		report.EconomyMoneyTransitionViolations > 0 ||
		report.EconomyPurchaseProvenanceViolations > 0 ||
		report.EconomyPriceTableVersionViolations > 0 ||
		report.StatsScoreboardReconciliationErrors > 0 ||
		report.StatsUtilityReconciliationErrors > 0 ||
		report.ClutchAttemptReconciliationErrors > 0 ||
		report.WarmupContaminationViolations > 0 ||
		report.MetadataProvenanceViolations > 0 ||
		report.MetadataChecksumLineageViolations > 0 ||
		report.EconomyDeterminismViolations > 0 ||
		report.StatsDeterminismViolations > 0
}
