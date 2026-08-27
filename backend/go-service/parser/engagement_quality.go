package parser

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"

	"cs2-demo-service/models"
	engagementpkg "cs2-demo-service/pkg/engagement"
	"cs2-demo-service/pkg/maps"
)

type engagementQualityAssessment struct {
	engagements               int
	tradeCandidates           int
	tradeCompletions          int
	eventContract             int
	atomicProvenance          int
	participantReconciliation int
	roleConsistency           int
	temporalConsistency       int
	causalAvailability        int
	tradeReconciliation       int
	statsReconciliation       int
	determinism               int
	observationWarnings       int
	failureDetails            []string
}

func assessEngagementQuality(
	matchID string,
	tickRate float64,
	rounds []models.CanonicalRound,
	participants []models.CanonicalParticipant,
	combatEvents []models.CanonicalCombatEvent,
	statesByRound map[int][]models.CanonicalPlayerState,
	engagements models.CanonicalEngagementsExport,
	trades models.CanonicalTradesExport,
	stats models.CanonicalPlayerMatchStatsExport,
	visibility maps.VisibilityChecker,
) engagementQualityAssessment {
	assessment := engagementQualityAssessment{
		engagements:      len(engagements.Engagements),
		tradeCandidates:  len(trades.Candidates),
		tradeCompletions: len(trades.Completions),
	}
	eventByID := make(map[string]models.CanonicalCombatEvent, len(combatEvents))
	for _, event := range combatEvents {
		eventByID[event.EventID] = event
	}
	participantIDs := make(map[string]struct{}, len(participants))
	for _, participant := range participants {
		participantIDs[participant.PlayerID] = struct{}{}
	}
	stateByID := make(map[string]models.CanonicalPlayerState)
	for _, states := range statesByRound {
		for _, state := range states {
			stateByID[state.StateID] = state
		}
	}

	assessment.assessEventContract(matchID, tickRate, engagements, trades, eventByID)
	assessment.assessAtomicProvenance(engagements, eventByID)
	assessment.assessParticipants(engagements, participantIDs)
	assessment.assessRoles(engagements, eventByID)
	assessment.assessTemporal(tickRate, rounds, engagements, eventByID)
	assessment.assessCausalAvailability(engagements, stateByID)
	assessment.assessTrades(tickRate, trades, eventByID)
	assessment.assessStats(trades, combatEvents, stats)
	assessment.assessDeterminism(matchID, tickRate, rounds, participants, combatEvents, statesByRound, engagements, trades, visibility)
	assessment.assessObservationCoverage(engagements, trades)
	return assessment
}

func (assessment *engagementQualityAssessment) fail(counter *int, format string, args ...interface{}) {
	*counter++
	if len(assessment.failureDetails) < 50 {
		assessment.failureDetails = append(assessment.failureDetails, fmt.Sprintf(format, args...))
	}
}

func (assessment *engagementQualityAssessment) assessEventContract(
	matchID string,
	tickRate float64,
	engagements models.CanonicalEngagementsExport,
	trades models.CanonicalTradesExport,
	eventByID map[string]models.CanonicalCombatEvent,
) {
	if engagements.SchemaID != "stratai.engagements@2" || engagements.MatchID != matchID ||
		engagements.Config.AlgorithmVersion != engagementpkg.AlgorithmVersion ||
		engagements.Config.TickRateHz != tickRate ||
		engagements.Config.PairContinuationTicks != engagementTicks(engagements.Config.PairContinuationWindowMS, tickRate) ||
		engagements.Config.MultiTargetWindowTicks != engagementTicks(engagements.Config.MultiTargetWindowMS, tickRate) ||
		engagements.Config.MaxEngagementDurationTick != engagementTicks(engagements.Config.MaxEngagementDurationMS, tickRate) ||
		engagements.Config.AggressorPreludeTicks != engagementTicks(engagements.Config.AggressorPreludeWindowMS, tickRate) {
		assessment.fail(&assessment.eventContract, "engagement export/config contract mismatch")
	}
	if trades.SchemaID != "stratai.trades@1" || trades.MatchID != matchID ||
		trades.Config.AlgorithmVersion != engagementpkg.TradeAlgorithmVersion ||
		trades.Config.TickRateHz != tickRate ||
		trades.Config.TradeWindowTicks != engagementTicks(trades.Config.TradeWindowMS, tickRate) {
		assessment.fail(&assessment.eventContract, "trade export/config contract mismatch")
	}
	engagementIDs := make(map[string]struct{}, len(engagements.Engagements))
	exchangeIDs := make(map[string]struct{})
	for _, engagement := range engagements.Engagements {
		if engagement.EngagementID == "" || engagement.RoundID == "" || engagement.AlgorithmVersion != engagementpkg.AlgorithmVersion ||
			len(engagement.Exchanges) == 0 || len(engagement.Participants) < 2 || len(engagement.SourceEventIDs) == 0 {
			assessment.fail(&assessment.eventContract, "%s has incomplete required fields", engagement.EngagementID)
		}
		if _, duplicate := engagementIDs[engagement.EngagementID]; duplicate {
			assessment.fail(&assessment.eventContract, "duplicate engagement ID %s", engagement.EngagementID)
		}
		engagementIDs[engagement.EngagementID] = struct{}{}
		for _, exchange := range engagement.Exchanges {
			if _, duplicate := exchangeIDs[exchange.ExchangeID]; duplicate {
				assessment.fail(&assessment.eventContract, "duplicate exchange ID %s", exchange.ExchangeID)
			}
			exchangeIDs[exchange.ExchangeID] = struct{}{}
			hurt, exists := eventByID[exchange.ExchangeID]
			if !exists || hurt.EventType != "player_hurt" || hurt.Relation != "enemy" ||
				hurt.ActorPlayerID == nil || hurt.TargetPlayerID == nil ||
				exchange.ActorPlayerID != *hurt.ActorPlayerID || exchange.TargetPlayerID != *hurt.TargetPlayerID ||
				exchange.Tick != hurt.Tick || exchange.SequenceInTick != hurt.SequenceInTick ||
				exchange.SequenceInRound != hurt.SequenceInRound {
				assessment.fail(&assessment.eventContract, "%s is not an exact enemy player_hurt projection", exchange.ExchangeID)
			}
			if !engagementContains(exchange.SourceEventIDs, exchange.ExchangeID) {
				assessment.fail(&assessment.eventContract, "%s omits its atomic event from source_event_ids", exchange.ExchangeID)
			}
			if exchange.IsKill != (exchange.KillEventID != nil) {
				assessment.fail(&assessment.eventContract, "%s kill availability is contradictory", exchange.ExchangeID)
			}
		}
	}
	assessment.assessUniqueTradeIDs(trades)
}

func (assessment *engagementQualityAssessment) assessUniqueTradeIDs(trades models.CanonicalTradesExport) {
	ids := make(map[string]struct{}, len(trades.Candidates)+len(trades.Completions))
	for _, candidate := range trades.Candidates {
		if candidate.TradeCandidateID == "" || candidate.OriginalKillEventID == "" {
			assessment.fail(&assessment.eventContract, "trade candidate has incomplete required fields")
		}
		if _, duplicate := ids[candidate.TradeCandidateID]; duplicate {
			assessment.fail(&assessment.eventContract, "duplicate trade ID %s", candidate.TradeCandidateID)
		}
		ids[candidate.TradeCandidateID] = struct{}{}
	}
	for _, completion := range trades.Completions {
		if completion.TradeCompletionID == "" || completion.TradeCandidateID == "" || completion.ResponseKillEventID == "" {
			assessment.fail(&assessment.eventContract, "trade completion has incomplete required fields")
		}
		if _, duplicate := ids[completion.TradeCompletionID]; duplicate {
			assessment.fail(&assessment.eventContract, "duplicate trade ID %s", completion.TradeCompletionID)
		}
		ids[completion.TradeCompletionID] = struct{}{}
	}
}

func (assessment *engagementQualityAssessment) assessAtomicProvenance(
	engagements models.CanonicalEngagementsExport,
	eventByID map[string]models.CanonicalCombatEvent,
) {
	hurtOwners := make(map[string]string)
	sourceOwners := make(map[string]string)
	for _, engagement := range engagements.Engagements {
		expectedEngagementSources := make([]string, 0)
		for _, sourceID := range engagement.SourceEventIDs {
			if _, exists := eventByID[sourceID]; !exists {
				assessment.fail(&assessment.atomicProvenance, "%s references missing combat event %s", engagement.EngagementID, sourceID)
			}
			if owner, exists := sourceOwners[sourceID]; exists && owner != engagement.EngagementID {
				assessment.fail(&assessment.atomicProvenance, "combat event %s is reused by %s and %s", sourceID, owner, engagement.EngagementID)
			} else {
				sourceOwners[sourceID] = engagement.EngagementID
			}
		}
		for _, exchange := range engagement.Exchanges {
			if owner, exists := hurtOwners[exchange.ExchangeID]; exists {
				assessment.fail(&assessment.atomicProvenance, "player_hurt %s is projected by %s and %s", exchange.ExchangeID, owner, engagement.EngagementID)
			}
			hurtOwners[exchange.ExchangeID] = engagement.EngagementID
			expected := engagementEventClosureIDs(exchange.ExchangeID, eventByID)
			if exchange.KillEventID != nil {
				expected = engagementMergeStrings(expected, engagementEventClosureIDs(*exchange.KillEventID, eventByID))
			}
			if !reflect.DeepEqual(engagementSortedStrings(exchange.SourceEventIDs), engagementSortedStrings(expected)) {
				assessment.fail(&assessment.atomicProvenance, "%s source closure differs from combat_event@2", exchange.ExchangeID)
			}
			expectedEngagementSources = append(expectedEngagementSources, expected...)
		}
		if engagement.FirstAggressor.Status == "inferred" {
			expectedEngagementSources = append(expectedEngagementSources, engagement.FirstAggressor.SourceEventIDs...)
		}
		if !reflect.DeepEqual(engagementSortedStrings(engagement.SourceEventIDs), engagementSortedStrings(expectedEngagementSources)) {
			assessment.fail(&assessment.atomicProvenance, "%s source_event_ids differ from its exchange/role closures", engagement.EngagementID)
		}
	}
	for _, event := range eventByID {
		if event.EventType == "player_hurt" && event.Relation == "enemy" && event.ActorPlayerID != nil && event.TargetPlayerID != nil {
			if _, exists := hurtOwners[event.EventID]; !exists {
				assessment.fail(&assessment.atomicProvenance, "enemy player_hurt %s has no engagement exchange", event.EventID)
			}
		}
	}
}

func (assessment *engagementQualityAssessment) assessParticipants(
	engagements models.CanonicalEngagementsExport,
	roster map[string]struct{},
) {
	for _, engagement := range engagements.Engagements {
		expected := make([]string, 0, len(engagement.Exchanges)*2)
		for _, exchange := range engagement.Exchanges {
			expected = append(expected, exchange.ActorPlayerID, exchange.TargetPlayerID)
		}
		expected = engagementSortedStrings(expected)
		actual := make([]string, 0, len(engagement.Participants))
		for _, participant := range engagement.Participants {
			actual = append(actual, participant.PlayerID)
			if _, exists := roster[participant.PlayerID]; !exists {
				assessment.fail(&assessment.participantReconciliation, "%s contains non-roster participant %s", engagement.EngagementID, participant.PlayerID)
			}
		}
		if !reflect.DeepEqual(actual, expected) {
			assessment.fail(&assessment.participantReconciliation, "%s participant set differs from exchanges", engagement.EngagementID)
		}
		for _, role := range []models.CanonicalRoleAssignment{engagement.Initiator, engagement.FirstAggressor, engagement.FirstDamageDealer} {
			if role.PlayerID != nil && !engagementContains(expected, *role.PlayerID) {
				assessment.fail(&assessment.participantReconciliation, "%s assigns a role outside the participant set", engagement.EngagementID)
			}
		}
	}
}

func (assessment *engagementQualityAssessment) assessRoles(
	engagements models.CanonicalEngagementsExport,
	eventByID map[string]models.CanonicalCombatEvent,
) {
	for _, engagement := range engagements.Engagements {
		first := engagement.Exchanges[0]
		if engagement.FirstDamageDealer.PlayerID == nil || *engagement.FirstDamageDealer.PlayerID != first.ActorPlayerID ||
			engagement.FirstDamageDealer.Status != "observed" ||
			!engagementContains(engagement.FirstDamageDealer.SourceEventIDs, first.ExchangeID) {
			assessment.fail(&assessment.roleConsistency, "%s first_damage_dealer is inconsistent with its first exchange", engagement.EngagementID)
		}
		if engagement.FirstAggressor.PlayerID != nil {
			if engagement.FirstAggressor.Status != "observed" && engagement.FirstAggressor.Status != "inferred" {
				assessment.fail(&assessment.roleConsistency, "%s first_aggressor has invalid status", engagement.EngagementID)
			}
			for _, sourceID := range engagement.FirstAggressor.SourceEventIDs {
				event, exists := eventByID[sourceID]
				if !exists || engagementEventAfter(event, eventByID[first.ExchangeID]) {
					assessment.fail(&assessment.roleConsistency, "%s first_aggressor uses non-causal source %s", engagement.EngagementID, sourceID)
				}
			}
		} else if engagement.FirstAggressor.Status != "unavailable" {
			assessment.fail(&assessment.roleConsistency, "%s has a null first_aggressor without unavailable status", engagement.EngagementID)
		}
		if engagement.Initiator.PlayerID == nil || engagement.Initiator.Source == "winner" {
			assessment.fail(&assessment.roleConsistency, "%s initiator is unavailable or outcome-derived", engagement.EngagementID)
		}

		winnerIDs := make(map[string]struct{})
		loserIDs := make([]string, 0)
		killIDs := make([]string, 0)
		for _, exchange := range engagement.Exchanges {
			if !exchange.IsKill || exchange.KillEventID == nil {
				continue
			}
			kill, exists := eventByID[*exchange.KillEventID]
			if !exists || kill.EventType != "kill" || kill.ActorPlayerID == nil || kill.TargetPlayerID == nil {
				assessment.fail(&assessment.roleConsistency, "%s references invalid terminal kill", engagement.EngagementID)
				continue
			}
			winnerIDs[*kill.ActorPlayerID] = struct{}{}
			loserIDs = append(loserIDs, *kill.TargetPlayerID)
			killIDs = append(killIDs, kill.EventID)
		}
		if len(killIDs) == 0 {
			if engagement.OutcomeContext.Outcome != "disengaged" || engagement.OutcomeContext.WinnerPlayerID != nil || len(engagement.OutcomeContext.LoserPlayerIDs) != 0 {
				assessment.fail(&assessment.roleConsistency, "%s non-kill outcome contains terminal winner/loser data", engagement.EngagementID)
			}
			continue
		}
		if engagement.OutcomeContext.Outcome != "kill" ||
			!reflect.DeepEqual(engagementSortedStrings(killIDs), engagementSortedStrings(engagement.OutcomeContext.TerminalKillEventIDs)) ||
			!reflect.DeepEqual(engagementSortedStrings(loserIDs), engagementSortedStrings(engagement.OutcomeContext.LoserPlayerIDs)) {
			assessment.fail(&assessment.roleConsistency, "%s terminal outcome differs from kill events", engagement.EngagementID)
		}
		if len(winnerIDs) == 1 {
			for winnerID := range winnerIDs {
				if engagement.OutcomeContext.WinnerPlayerID == nil || *engagement.OutcomeContext.WinnerPlayerID != winnerID {
					assessment.fail(&assessment.roleConsistency, "%s winner differs from terminal kill actor", engagement.EngagementID)
				}
			}
		} else if engagement.OutcomeContext.WinnerPlayerID != nil {
			assessment.fail(&assessment.roleConsistency, "%s exposes one winner for a multi-winner outcome", engagement.EngagementID)
		}
	}
}

func (assessment *engagementQualityAssessment) assessTemporal(
	tickRate float64,
	rounds []models.CanonicalRound,
	engagements models.CanonicalEngagementsExport,
	eventByID map[string]models.CanonicalCombatEvent,
) {
	roundByNumber := make(map[int]models.CanonicalRound, len(rounds))
	for _, round := range rounds {
		roundByNumber[round.RoundNumber] = round
	}
	for index, engagement := range engagements.Engagements {
		if engagement.EndTick < engagement.StartTick ||
			engagement.EndTick-engagement.StartTick > engagements.Config.MaxEngagementDurationTick ||
			math.Abs(engagement.DurationMS-float64(engagement.EndTick-engagement.StartTick)*1000/tickRate) > 0.000001 {
			assessment.fail(&assessment.temporalConsistency, "%s has invalid bounds or duration", engagement.EngagementID)
		}
		if index > 0 && engagementArtifactAfter(engagements.Engagements[index-1], engagement) {
			assessment.fail(&assessment.temporalConsistency, "engagement ordering is unstable at %s", engagement.EngagementID)
		}
		round, exists := roundByNumber[engagement.RoundNumber]
		if !exists || engagement.RoundID != round.RoundID ||
			(round.StartTick != nil && engagement.StartTick < *round.StartTick) ||
			(round.EndTick != nil && engagement.EndTick > *round.EndTick) {
			assessment.fail(&assessment.temporalConsistency, "%s falls outside its canonical round", engagement.EngagementID)
		}
		if len(engagement.SourceEventIDs) > 0 {
			sources := make([]models.CanonicalCombatEvent, 0, len(engagement.SourceEventIDs))
			for _, sourceID := range engagement.SourceEventIDs {
				if event, exists := eventByID[sourceID]; exists {
					sources = append(sources, event)
				}
			}
			sort.Slice(sources, func(i, j int) bool { return engagementEventBefore(sources[i], sources[j]) })
			if len(sources) > 0 && (engagement.StartTick != sources[0].Tick || engagement.StartSequenceInTick != sources[0].SequenceInTick ||
				engagement.EndTick != sources[len(sources)-1].Tick || engagement.EndSequenceInTick != sources[len(sources)-1].SequenceInTick) {
				assessment.fail(&assessment.temporalConsistency, "%s bounds differ from its atomic source closure", engagement.EngagementID)
			}
		}
		for exchangeIndex, exchange := range engagement.Exchanges {
			if exchangeIndex > 0 {
				previous := eventByID[engagement.Exchanges[exchangeIndex-1].ExchangeID]
				current := eventByID[exchange.ExchangeID]
				if engagementEventAfter(previous, current) {
					assessment.fail(&assessment.temporalConsistency, "%s exchanges are not in atomic order", engagement.EngagementID)
				}
			}
		}
	}
}

func (assessment *engagementQualityAssessment) assessCausalAvailability(
	engagements models.CanonicalEngagementsExport,
	stateByID map[string]models.CanonicalPlayerState,
) {
	for _, engagement := range engagements.Engagements {
		context := engagement.CausalContext
		if context.T0Tick != engagement.StartTick || context.T0SequenceInTick != engagement.StartSequenceInTick {
			assessment.fail(&assessment.causalAvailability, "%s causal t0 differs from engagement start", engagement.EngagementID)
		}
		participantIDs := make([]string, 0, len(engagement.Participants))
		for _, participant := range engagement.Participants {
			participantIDs = append(participantIDs, participant.PlayerID)
		}
		statePlayerIDs := make([]string, 0, len(context.ParticipantStates))
		for _, participantState := range context.ParticipantStates {
			statePlayerIDs = append(statePlayerIDs, participantState.PlayerID)
			if participantState.Status == "unavailable" {
				if participantState.StateID != nil || participantState.AvailabilityTick != nil || participantState.MovementClassification != nil {
					assessment.fail(&assessment.causalAvailability, "%s fabricates fields for unavailable state %s", engagement.EngagementID, participantState.PlayerID)
				}
				continue
			}
			if participantState.Status != "observed" || participantState.StateID == nil || participantState.AvailabilityTick == nil {
				assessment.fail(&assessment.causalAvailability, "%s has invalid state availability for %s", engagement.EngagementID, participantState.PlayerID)
				continue
			}
			state, exists := stateByID[*participantState.StateID]
			if !exists || state.PlayerID != participantState.PlayerID || state.Tick != *participantState.AvailabilityTick || state.Tick > context.T0Tick {
				assessment.fail(&assessment.causalAvailability, "%s uses missing/future state %s", engagement.EngagementID, *participantState.StateID)
			}
			if participantState.HorizontalVelocityWorldUPS == nil {
				if participantState.VelocityStatus != "unavailable" || participantState.MovementClassification != nil {
					assessment.fail(&assessment.causalAvailability, "%s fabricates movement without velocity for %s", engagement.EngagementID, participantState.PlayerID)
				}
			} else if participantState.VelocityStatus != "observed" || participantState.MovementClassification == nil {
				assessment.fail(&assessment.causalAvailability, "%s omits classification for observed velocity of %s", engagement.EngagementID, participantState.PlayerID)
			}
			if participantState.ActiveWeapon == nil && participantState.ActiveWeaponStatus == "observed" {
				assessment.fail(&assessment.causalAvailability, "%s marks a missing active weapon observed", engagement.EngagementID)
			}
		}
		participantSet := make(map[string]struct{}, len(participantIDs))
		for _, playerID := range participantIDs {
			participantSet[playerID] = struct{}{}
		}
		for _, playerID := range statePlayerIDs {
			if _, exists := participantSet[playerID]; !exists {
				assessment.fail(&assessment.causalAvailability, "%s causal state contains non-participant %s", engagement.EngagementID, playerID)
			}
		}
		if engagement.Initiator.PlayerID != nil && !containsEngagementString(statePlayerIDs, *engagement.Initiator.PlayerID) {
			assessment.fail(&assessment.causalAvailability, "%s causal state omits the t0 actor", engagement.EngagementID)
		}
		if context.InitialDistanceWorldUnits == nil && context.InitialDistanceStatus != "unavailable" {
			assessment.fail(&assessment.causalAvailability, "%s has contradictory initial-distance availability", engagement.EngagementID)
		}
		for _, role := range []models.CanonicalRoleAssignment{engagement.Initiator, engagement.FirstAggressor, engagement.FirstDamageDealer} {
			if role.AvailabilityTick != nil && *role.AvailabilityTick < engagement.StartTick {
				assessment.fail(&assessment.causalAvailability, "%s publishes role before its declared t0", engagement.EngagementID)
			}
		}
	}
}

func containsEngagementString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func (assessment *engagementQualityAssessment) assessTrades(
	tickRate float64,
	trades models.CanonicalTradesExport,
	eventByID map[string]models.CanonicalCombatEvent,
) {
	candidateByID := make(map[string]models.CanonicalTradeCandidate, len(trades.Candidates))
	candidateByKill := make(map[string]int)
	for _, candidate := range trades.Candidates {
		candidateByID[candidate.TradeCandidateID] = candidate
		candidateByKill[candidate.OriginalKillEventID]++
		kill, exists := eventByID[candidate.OriginalKillEventID]
		if !exists || kill.EventType != "kill" || kill.Relation != "enemy" || kill.ActorPlayerID == nil || kill.TargetPlayerID == nil ||
			candidate.RoundNumber != kill.RoundNumber || candidate.DeathTick != kill.Tick || candidate.DeathSequenceInTick != kill.SequenceInTick ||
			candidate.OriginalVictimPlayerID != *kill.TargetPlayerID || candidate.OriginalKillerPlayerID != *kill.ActorPlayerID ||
			candidate.WindowTicks != trades.Config.TradeWindowTicks || candidate.WindowEndTick != candidate.DeathTick+candidate.WindowTicks ||
			!engagementContains(candidate.SourceEventIDs, candidate.OriginalKillEventID) {
			assessment.fail(&assessment.tradeReconciliation, "%s differs from its original kill", candidate.TradeCandidateID)
		}
		validEvaluation := map[string]bool{"completed": true, "failed": true, "not_attempted": true, "not_tradeable": true, "not_evaluable": true}
		if !validEvaluation[candidate.Evaluation] || (candidate.Evaluation == "completed") != (candidate.TradeCompletionID != nil) {
			assessment.fail(&assessment.tradeReconciliation, "%s has contradictory evaluation", candidate.TradeCandidateID)
		}
	}
	for _, event := range eventByID {
		if event.EventType == "kill" && event.Relation == "enemy" && event.ActorPlayerID != nil && event.TargetPlayerID != nil && candidateByKill[event.EventID] != 1 {
			assessment.fail(&assessment.tradeReconciliation, "enemy kill %s has %d trade candidates", event.EventID, candidateByKill[event.EventID])
		}
	}
	usedCandidates := make(map[string]struct{})
	usedResponses := make(map[string]struct{})
	completionByID := make(map[string]models.CanonicalTradeCompletion, len(trades.Completions))
	for _, completion := range trades.Completions {
		completionByID[completion.TradeCompletionID] = completion
		candidate, exists := candidateByID[completion.TradeCandidateID]
		original := eventByID[completion.OriginalKillEventID]
		response, responseExists := eventByID[completion.ResponseKillEventID]
		if !exists || !responseExists || response.EventType != "kill" || response.ActorPlayerID == nil || response.TargetPlayerID == nil ||
			completion.OriginalKillEventID != candidate.OriginalKillEventID || completion.RoundNumber != candidate.RoundNumber ||
			!engagementEventBefore(original, response) || response.Tick-candidate.DeathTick > candidate.WindowTicks ||
			*response.TargetPlayerID != candidate.OriginalKillerPlayerID || *response.ActorPlayerID != completion.TraderPlayerID ||
			!engagementContains(candidate.EligibleTeammatePlayerIDs, completion.TraderPlayerID) ||
			completion.ElapsedTicks != response.Tick-candidate.DeathTick ||
			math.Abs(completion.ElapsedMS-float64(completion.ElapsedTicks)*1000/tickRate) > 0.000001 {
			assessment.fail(&assessment.tradeReconciliation, "%s is not a valid one-to-one response kill", completion.TradeCompletionID)
		}
		if _, duplicate := usedCandidates[completion.TradeCandidateID]; duplicate {
			assessment.fail(&assessment.tradeReconciliation, "candidate %s has multiple completions", completion.TradeCandidateID)
		}
		if _, duplicate := usedResponses[completion.ResponseKillEventID]; duplicate {
			assessment.fail(&assessment.tradeReconciliation, "response kill %s completes multiple candidates", completion.ResponseKillEventID)
		}
		usedCandidates[completion.TradeCandidateID] = struct{}{}
		usedResponses[completion.ResponseKillEventID] = struct{}{}
	}
	for _, candidate := range trades.Candidates {
		if candidate.TradeCompletionID != nil {
			completion, exists := completionByID[*candidate.TradeCompletionID]
			if !exists || completion.TradeCandidateID != candidate.TradeCandidateID {
				assessment.fail(&assessment.tradeReconciliation, "%s references a missing completion", candidate.TradeCandidateID)
			}
		}
		if candidate.CounterTradeOfCompletionID != nil {
			completion, exists := completionByID[*candidate.CounterTradeOfCompletionID]
			if !exists || completion.ResponseKillEventID != candidate.OriginalKillEventID {
				assessment.fail(&assessment.tradeReconciliation, "%s has an invalid counter-trade link", candidate.TradeCandidateID)
			}
		}
	}
}

func (assessment *engagementQualityAssessment) assessStats(
	trades models.CanonicalTradesExport,
	combatEvents []models.CanonicalCombatEvent,
	stats models.CanonicalPlayerMatchStatsExport,
) {
	type expectedStats struct {
		tradeKills, tradedDeaths, attempts, failed, untradeable, nonEvaluable int
	}
	expected := make(map[string]*expectedStats)
	eventByID := make(map[string]models.CanonicalCombatEvent, len(combatEvents))
	for _, event := range combatEvents {
		eventByID[event.EventID] = event
	}
	completionByCandidate := make(map[string]models.CanonicalTradeCompletion, len(trades.Completions))
	for _, completion := range trades.Completions {
		completionByCandidate[completion.TradeCandidateID] = completion
	}
	ensure := func(playerID string) *expectedStats {
		if expected[playerID] == nil {
			expected[playerID] = &expectedStats{}
		}
		return expected[playerID]
	}
	for _, candidate := range trades.Candidates {
		switch candidate.Evaluation {
		case "not_tradeable":
			ensure(candidate.OriginalVictimPlayerID).untradeable++
		case "not_evaluable":
			ensure(candidate.OriginalVictimPlayerID).nonEvaluable++
		}
		actors := make(map[string]struct{})
		for _, eventID := range candidate.AttemptEventIDs {
			if event := eventByID[eventID]; event.ActorPlayerID != nil {
				actors[*event.ActorPlayerID] = struct{}{}
			}
		}
		if completion, exists := completionByCandidate[candidate.TradeCandidateID]; exists {
			actors[completion.TraderPlayerID] = struct{}{}
		}
		for playerID := range actors {
			ensure(playerID).attempts++
			if candidate.Evaluation == "failed" {
				ensure(playerID).failed++
			}
		}
	}
	for _, completion := range trades.Completions {
		ensure(completion.TraderPlayerID).tradeKills++
		ensure(completion.OriginalVictimPlayerID).tradedDeaths++
	}
	for _, player := range stats.Players {
		wanted := ensure(player.PlayerID)
		actual := player.Metrics
		if actual.TradeKills != wanted.tradeKills || actual.TradedDeaths != wanted.tradedDeaths ||
			actual.TradeAttempts != wanted.attempts || actual.FailedTradeAttempts != wanted.failed ||
			actual.UntradeableDeaths != wanted.untradeable || actual.NonEvaluableTradeDeaths != wanted.nonEvaluable {
			assessment.fail(&assessment.statsReconciliation, "trade stats mismatch for %s", player.PlayerID)
		}
	}
}

func (assessment *engagementQualityAssessment) assessDeterminism(
	matchID string,
	tickRate float64,
	rounds []models.CanonicalRound,
	participants []models.CanonicalParticipant,
	combatEvents []models.CanonicalCombatEvent,
	statesByRound map[int][]models.CanonicalPlayerState,
	engagements models.CanonicalEngagementsExport,
	trades models.CanonicalTradesExport,
	visibility maps.VisibilityChecker,
) {
	reversedEvents := append([]models.CanonicalCombatEvent(nil), combatEvents...)
	engagementReverse(reversedEvents)
	reversedRounds := append([]models.CanonicalRound(nil), rounds...)
	engagementReverse(reversedRounds)
	reversedParticipants := append([]models.CanonicalParticipant(nil), participants...)
	engagementReverse(reversedParticipants)
	reversedStates := make(map[int][]models.CanonicalPlayerState, len(statesByRound))
	for roundNumber, states := range statesByRound {
		reversedStates[roundNumber] = append([]models.CanonicalPlayerState(nil), states...)
		engagementReverse(reversedStates[roundNumber])
	}
	rederivedEngagements, rederivedTrades, err := engagementpkg.Derive(
		matchID, tickRate, reversedRounds, reversedParticipants, reversedEvents, reversedStates, visibility,
	)
	if err != nil || !reflect.DeepEqual(engagements, rederivedEngagements) || !reflect.DeepEqual(trades, rederivedTrades) {
		assessment.fail(&assessment.determinism, "engagement/trade derivation changed under input permutation")
	}
}

func (assessment *engagementQualityAssessment) assessObservationCoverage(
	engagements models.CanonicalEngagementsExport,
	trades models.CanonicalTradesExport,
) {
	for _, engagement := range engagements.Engagements {
		if engagement.FirstAggressor.Status != "observed" {
			assessment.observationWarnings++
		}
		for _, state := range engagement.CausalContext.ParticipantStates {
			if state.Status != "observed" || state.HorizontalVelocityWorldUPS == nil || state.ActiveWeapon == nil {
				assessment.observationWarnings++
			}
		}
		if engagement.CausalContext.InitialDistanceWorldUnits == nil {
			assessment.observationWarnings++
		}
	}
	for _, candidate := range trades.Candidates {
		if candidate.Evaluation == "not_evaluable" {
			assessment.observationWarnings++
		}
	}
}

func (assessment engagementQualityAssessment) checks() []qualityCheck {
	return []qualityCheck{
		engagementQualityCountCheck("engagement_event_contract", assessment.eventContract, "Engagement and trade rows must satisfy the frozen versioned contract."),
		engagementQualityCountCheck("engagement_atomic_provenance", assessment.atomicProvenance, "Every exchange must project one exact atomic hurt with a complete, exclusive source closure."),
		engagementQualityCountCheck("engagement_participant_reconciliation", assessment.participantReconciliation, "Participants must be roster members and equal the exchange actor/target union."),
		engagementQualityCountCheck("engagement_role_consistency", assessment.roleConsistency, "Initiator, aggressor, first damage, winner and losers must retain distinct evidence-based semantics."),
		engagementQualityCountCheck("engagement_temporal_consistency", assessment.temporalConsistency, "Bounds, durations, exchanges and stable ordering must agree with atomic time."),
		engagementQualityCountCheck("engagement_causal_availability", assessment.causalAvailability, "Causal features must use observations available at t0 and expose unavailable values explicitly."),
		engagementQualityCountCheck("engagement_trade_reconciliation", assessment.tradeReconciliation, "Every enemy kill must have one trade candidate and response kills must match one-to-one."),
		engagementQualityCountCheck("engagement_stats_reconciliation", assessment.statsReconciliation, "Trade statistics must be a lossless projection of trade candidates and completions."),
		engagementQualityCountCheck("engagement_determinism", assessment.determinism, "Input permutations must produce byte-equivalent engagement and trade models."),
		engagementObservationCheck(assessment.observationWarnings),
	}
}

func engagementQualityCountCheck(name string, violations int, message string) qualityCheck {
	status := "pass"
	if violations > 0 {
		status = "fail"
	}
	return qualityCheck{Name: name, Status: status, Expected: "0", Actual: strconv.Itoa(violations), Message: message}
}

func engagementObservationCheck(warnings int) qualityCheck {
	status := "pass"
	if warnings > 0 {
		status = "warning"
	}
	return qualityCheck{
		Name: "engagement_observation_coverage", Status: status, Expected: "explicit availability",
		Actual: strconv.Itoa(warnings), Message: "Sparse role, movement, weapon, distance and trade-window observations remain explicit and do not fabricate values.",
	}
}

func (report *qualityReport) applyEngagementQuality(assessment engagementQualityAssessment) {
	report.SchemaVersion = 10
	report.Engagements = assessment.engagements
	report.TradeCandidates = assessment.tradeCandidates
	report.TradeCompletions = assessment.tradeCompletions
	report.EngagementEventContractViolations = assessment.eventContract
	report.EngagementAtomicProvenanceViolations = assessment.atomicProvenance
	report.EngagementParticipantMismatches = assessment.participantReconciliation
	report.EngagementRoleConsistencyViolations = assessment.roleConsistency
	report.EngagementTemporalConsistencyErrors = assessment.temporalConsistency
	report.EngagementCausalAvailabilityErrors = assessment.causalAvailability
	report.EngagementTradeReconciliationErrors = assessment.tradeReconciliation
	report.EngagementStatsReconciliationErrors = assessment.statsReconciliation
	report.EngagementDeterminismViolations = assessment.determinism
	report.EngagementObservationWarnings = assessment.observationWarnings
	report.engagementFailureDetails = append([]string(nil), assessment.failureDetails...)
	for _, check := range assessment.checks() {
		report.Checks = append(report.Checks, check)
		if check.Status == "fail" {
			report.Status = "fail"
			report.UsableForTraining = false
			report.Warnings = append(report.Warnings, check.Message)
		} else if check.Status == "warning" {
			if report.Status == "pass" {
				report.Status = "warning"
			}
			report.Warnings = append(report.Warnings, check.Message)
		}
	}
}

func (report qualityReport) hasHardEngagementFailure() bool {
	return report.EngagementEventContractViolations > 0 ||
		report.EngagementAtomicProvenanceViolations > 0 ||
		report.EngagementParticipantMismatches > 0 ||
		report.EngagementRoleConsistencyViolations > 0 ||
		report.EngagementTemporalConsistencyErrors > 0 ||
		report.EngagementCausalAvailabilityErrors > 0 ||
		report.EngagementTradeReconciliationErrors > 0 ||
		report.EngagementStatsReconciliationErrors > 0 ||
		report.EngagementDeterminismViolations > 0
}

func engagementTicks(milliseconds int, tickRate float64) int {
	return int(math.Ceil(float64(milliseconds) * tickRate / 1000))
}

func engagementEventBefore(left, right models.CanonicalCombatEvent) bool {
	if left.RoundNumber != right.RoundNumber {
		return left.RoundNumber < right.RoundNumber
	}
	if left.Tick != right.Tick {
		return left.Tick < right.Tick
	}
	if left.SequenceInTick != right.SequenceInTick {
		return left.SequenceInTick < right.SequenceInTick
	}
	if left.SequenceInRound != right.SequenceInRound {
		return left.SequenceInRound < right.SequenceInRound
	}
	return left.EventID < right.EventID
}

func engagementEventAfter(left, right models.CanonicalCombatEvent) bool {
	return engagementEventBefore(right, left)
}

func engagementArtifactAfter(left, right models.CanonicalEngagement) bool {
	if left.RoundNumber != right.RoundNumber {
		return left.RoundNumber > right.RoundNumber
	}
	if left.StartTick != right.StartTick {
		return left.StartTick > right.StartTick
	}
	if left.StartSequenceInTick != right.StartSequenceInTick {
		return left.StartSequenceInTick > right.StartSequenceInTick
	}
	return left.EngagementID > right.EngagementID
}

func engagementEventClosureIDs(rootID string, eventByID map[string]models.CanonicalCombatEvent) []string {
	seen := make(map[string]struct{})
	var visit func(string)
	visit = func(eventID string) {
		if _, exists := seen[eventID]; exists {
			return
		}
		event, exists := eventByID[eventID]
		if !exists {
			return
		}
		seen[eventID] = struct{}{}
		for _, sourceID := range event.SourceEventIDs {
			visit(sourceID)
		}
	}
	visit(rootID)
	ids := make([]string, 0, len(seen))
	for eventID := range seen {
		ids = append(ids, eventID)
	}
	return ids
}

func engagementMergeStrings(groups ...[]string) []string {
	values := make([]string, 0)
	for _, group := range groups {
		values = append(values, group...)
	}
	return engagementSortedStrings(values)
}

func engagementSortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	if len(result) == 0 {
		return []string{}
	}
	write := 1
	for read := 1; read < len(result); read++ {
		if result[read] != result[write-1] {
			result[write] = result[read]
			write++
		}
	}
	return result[:write]
}

func engagementContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func engagementReverse[T any](values []T) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
