package parser

import (
	"cs2-demo-service/models"
	"fmt"
	"sort"
)

const (
	block7DecisionSchema        = "stratai.decision@1"
	block7DecisionFeatureSchema = "stratai.decision_features@1"
	block7OracleContextSchema   = "stratai.oracle_context@1"
	block7OutcomeSchema         = "stratai.decision_outcome@1"
	block7QualityMaskSchema     = "stratai.quality_mask@1"
	block7EngagementDecision    = "peek_hold_or_reposition"
	block7TradeDecision         = "spacing_or_trade_connection"
)

type canonicalCausalPartitions struct {
	Decisions        []models.CanonicalDecision
	DecisionFeatures []models.CanonicalDecisionFeatures
	OracleContext    []models.CanonicalOracleContext
	Outcomes         []models.CanonicalDecisionOutcome
	QualityMasks     []models.CanonicalQualityMask
}

func buildCanonicalCausalPartitions(
	matchID string,
	engagements []models.CanonicalEngagement,
	tradeExports ...models.CanonicalTradesExport,
) (canonicalCausalPartitions, error) {
	tradeCount := 0
	if len(tradeExports) > 0 {
		for _, candidate := range tradeExports[0].Candidates {
			tradeCount += len(candidate.Connections)
		}
	}
	result := canonicalCausalPartitions{
		Decisions:        make([]models.CanonicalDecision, 0, len(engagements)+tradeCount),
		DecisionFeatures: make([]models.CanonicalDecisionFeatures, 0, len(engagements)+tradeCount),
		OracleContext:    make([]models.CanonicalOracleContext, 0, len(engagements)+tradeCount),
		Outcomes:         make([]models.CanonicalDecisionOutcome, 0, len(engagements)+tradeCount),
		QualityMasks:     make([]models.CanonicalQualityMask, 0, len(engagements)+tradeCount),
	}
	for _, engagement := range engagements {
		if engagement.EngagementID == "" {
			return canonicalCausalPartitions{}, fmt.Errorf("engagement without stable ID")
		}
		// An engagement without an evidence-backed actor at T0 is useful as a
		// retrospective combat fact, but it is not a decision row. Abstain instead
		// of inventing an actor or failing publication of the whole match.
		actor := causalDecisionActor(engagement)
		if actor.PlayerID == nil || *actor.PlayerID == "" || actor.AvailabilityTick == nil ||
			*actor.AvailabilityTick > engagement.CausalContext.T0Tick {
			continue
		}
		decision, err := canonicalDecisionProjection(matchID, engagement)
		if err != nil {
			return canonicalCausalPartitions{}, err
		}
		features, mask, err := causalDecisionProjection(matchID, engagement)
		if err != nil {
			return canonicalCausalPartitions{}, err
		}
		result.Decisions = append(result.Decisions, decision)
		result.DecisionFeatures = append(result.DecisionFeatures, features)
		result.OracleContext = append(result.OracleContext, models.CanonicalOracleContext{
			SchemaID: block7OracleContextSchema, MatchID: matchID, DecisionID: engagement.EngagementID,
			RoundNumber: engagement.RoundNumber, T0Tick: engagement.CausalContext.T0Tick,
			Status: "unavailable", Available: false, FieldNames: []string{},
			Abstentions: []string{"hidden-state oracle is not available from the canonical parser"},
		})
		result.Outcomes = append(result.Outcomes, models.CanonicalDecisionOutcome{
			SchemaID: block7OutcomeSchema, MatchID: matchID, DecisionID: engagement.EngagementID,
			RoundNumber: engagement.RoundNumber, T0Tick: engagement.CausalContext.T0Tick,
			Outcome: engagement.OutcomeContext.Outcome, OutcomeTick: engagement.EndTick,
			DurationMS:           engagement.DurationMS,
			WinnerObserved:       engagement.OutcomeContext.WinnerPlayerID != nil,
			LoserCount:           len(engagement.OutcomeContext.LoserPlayerIDs),
			TerminalKillCount:    len(engagement.OutcomeContext.TerminalKillEventIDs),
			TradeCandidateCount:  len(engagement.OutcomeContext.TradeCandidateIDs),
			TradeCompletionCount: len(engagement.OutcomeContext.TradeCompletionIDs),
			SurvivalStatus:       engagement.OutcomeContext.SurvivalStatus,
			DisengagementStatus:  engagement.OutcomeContext.DisengagementStatus,
			Horizons:             engagementHorizonOutcomes(engagement),
		})
		result.QualityMasks = append(result.QualityMasks, mask)
	}
	if len(tradeExports) > 0 {
		appendTradeCausalPartitions(matchID, tradeExports[0], &result)
	}
	sortCanonicalCausalPartitions(&result)
	return result, nil
}

func canonicalDecisionProjection(
	matchID string,
	engagement models.CanonicalEngagement,
) (models.CanonicalDecision, error) {
	role := causalDecisionActor(engagement)
	if role.PlayerID == nil || *role.PlayerID == "" {
		return models.CanonicalDecision{}, fmt.Errorf("decision %s has no actor available at t0", engagement.EngagementID)
	}
	if role.AvailabilityTick == nil || *role.AvailabilityTick > engagement.CausalContext.T0Tick {
		return models.CanonicalDecision{}, fmt.Errorf("decision %s actor is not available at t0", engagement.EngagementID)
	}

	var stateRef *string
	stateStatus := "unavailable"
	action := "engage"
	for _, state := range engagement.CausalContext.ParticipantStates {
		if state.PlayerID != *role.PlayerID {
			continue
		}
		if state.AvailabilityTick != nil && *state.AvailabilityTick > engagement.CausalContext.T0Tick {
			return models.CanonicalDecision{}, fmt.Errorf("decision %s actor state is available after t0", engagement.EngagementID)
		}
		if state.StateID != nil && state.Status == "observed" {
			value := *state.StateID
			stateRef = &value
			stateStatus = "observed"
		}
		if state.MovementClassification != nil && (*state.MovementClassification == "peek" || *state.MovementClassification == "hold") {
			action = *state.MovementClassification
		}
		break
	}

	sourceEventIDs := sortedUniqueStrings(role.SourceEventIDs)
	return models.CanonicalDecision{
		SchemaID: block7DecisionSchema, MatchID: matchID, DecisionID: engagement.EngagementID,
		RoundNumber: engagement.RoundNumber, ActorPlayerID: *role.PlayerID,
		ActorIDUsage: "join_only", ObservedStateRef: stateRef, StateAvailabilityStatus: stateStatus,
		T0Tick: engagement.CausalContext.T0Tick, DecisionType: block7EngagementDecision,
		ActionTaken: action, AvailabilityTick: *role.AvailabilityTick,
		AvailabilityStatus: role.Status, CausalRole: "decision",
		VisibilityScope: "observable_proxy", Source: "engagements@2",
		SourceRecordID: engagement.EngagementID, SourceEventIDs: sourceEventIDs,
		AlgorithmVersion: "stratai.decision_projection@1",
	}, nil
}

func causalDecisionActor(engagement models.CanonicalEngagement) models.CanonicalRoleAssignment {
	roles := []models.CanonicalRoleAssignment{
		engagement.Initiator,
		engagement.FirstAggressor,
		engagement.FirstDamageDealer,
	}
	for _, role := range roles {
		if role.PlayerID != nil && *role.PlayerID != "" && role.AvailabilityTick != nil &&
			*role.AvailabilityTick <= engagement.CausalContext.T0Tick {
			return role
		}
	}
	return models.CanonicalRoleAssignment{}
}

func sortedUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func engagementHorizonOutcomes(engagement models.CanonicalEngagement) []models.CanonicalHorizonOutcome {
	result := make([]models.CanonicalHorizonOutcome, 0, 3)
	for _, seconds := range []int{2, 5, 10} {
		outcome := "engagement_ongoing"
		if engagement.DurationMS <= float64(seconds*1000) {
			outcome = engagement.OutcomeContext.Outcome
		}
		result = append(result, models.CanonicalHorizonOutcome{
			HorizonSeconds: seconds,
			Status:         "derived_outcome_only",
			Outcome:        outcome,
			Source:         "engagements@2",
		})
	}
	return result
}

func appendTradeCausalPartitions(
	matchID string,
	trades models.CanonicalTradesExport,
	result *canonicalCausalPartitions,
) {
	completionByCandidate := make(map[string]models.CanonicalTradeCompletion, len(trades.Completions))
	for _, completion := range trades.Completions {
		completionByCandidate[completion.TradeCandidateID] = completion
	}
	for _, candidate := range trades.Candidates {
		completion, completed := completionByCandidate[candidate.TradeCandidateID]
		for index, connection := range candidate.Connections {
			decisionID := fmt.Sprintf("%s:connection:%03d", candidate.TradeCandidateID, index+1)
			stateRef := connection.PlayerStateID
			stateStatus := connection.StateStatus
			action := "unclassified_connection"
			if connection.Eligible != nil {
				if *connection.Eligible {
					action = "connected"
				} else {
					action = "disconnected"
				}
			}
			result.Decisions = append(result.Decisions, models.CanonicalDecision{
				SchemaID: block7DecisionSchema, MatchID: matchID, DecisionID: decisionID,
				RoundNumber: candidate.RoundNumber, ActorPlayerID: connection.TeammatePlayerID,
				ActorIDUsage: "join_only", ObservedStateRef: stateRef, StateAvailabilityStatus: stateStatus,
				T0Tick: candidate.DeathTick, DecisionType: block7TradeDecision,
				ActionTaken: action, AvailabilityTick: candidate.DeathTick,
				AvailabilityStatus: connection.EligibilityStatus, CausalRole: "decision",
				VisibilityScope: "observed_physical_proxy", Source: "trades@1",
				SourceRecordID:   candidate.TradeCandidateID,
				SourceEventIDs:   []string{candidate.OriginalKillEventID},
				AlgorithmVersion: "stratai.decision_projection@1",
			})

			aliveCount := (*int)(nil)
			if connection.Alive != nil {
				value := 0
				if *connection.Alive {
					value = 1
				}
				aliveCount = &value
			}
			observedStates := 0
			if connection.StateStatus == "observed" {
				observedStates = 1
			}
			features := models.CanonicalDecisionFeatures{
				SchemaID: block7DecisionFeatureSchema, MatchID: matchID, DecisionID: decisionID,
				DecisionType: block7TradeDecision, RoundNumber: candidate.RoundNumber,
				T0Tick: candidate.DeathTick, AvailabilityTickMax: candidate.DeathTick,
				ParticipantCount: 1, ObservedParticipantStates: observedStates,
				AliveParticipantCount: aliveCount,
				// Exact actor-target distance is kept out of observed model features.
				// The physical teammate-connection measurements remain available in
				// their dedicated fields below.
				InitialDistanceWorldUnits: nil,
				InitialDistanceStatus:     "unavailable",
				BombContextStatus:         "unavailable", EconomyContextStatus: "unavailable",
				EnemiesExposedStatus: "unavailable", SourceStateCount: len(candidate.EligibilityStateIDs),
				TradePossible: connection.Eligible, TradePossibleStatus: connection.EligibilityStatus,
				NearestTeammateDistance: connection.DistanceWorldUnits,
				NearestDistanceStatus:   connection.DistanceStatus,
				NearestConnectionTimeMS: connection.ConnectionTimeMS,
				NearestConnectionStatus: connection.ConnectionTimeStatus,
				AnyLineOfSight:          connection.LineOfSight, LineOfSightStatus: connection.LineOfSightStatus,
				MinimumFacingDeltaDeg: connection.FacingDeltaDeg, FacingStatus: connection.FacingStatus,
			}
			result.DecisionFeatures = append(result.DecisionFeatures, features)
			result.OracleContext = append(result.OracleContext, models.CanonicalOracleContext{
				SchemaID: block7OracleContextSchema, MatchID: matchID, DecisionID: decisionID,
				RoundNumber: candidate.RoundNumber, T0Tick: candidate.DeathTick,
				Status: "unavailable", Available: false, FieldNames: []string{},
				Abstentions: []string{"hidden-state oracle is not available from the canonical parser"},
			})
			outcomeTick := candidate.WindowEndTick
			durationMS := float64(candidate.WindowMS)
			winnerObserved := false
			completionCount := 0
			if completed {
				outcomeTick = candidate.DeathTick + completion.ElapsedTicks
				durationMS = completion.ElapsedMS
				winnerObserved = completion.TraderPlayerID == connection.TeammatePlayerID
				completionCount = 1
			}
			result.Outcomes = append(result.Outcomes, models.CanonicalDecisionOutcome{
				SchemaID: block7OutcomeSchema, MatchID: matchID, DecisionID: decisionID,
				RoundNumber: candidate.RoundNumber, T0Tick: candidate.DeathTick,
				Outcome: candidate.Evaluation, OutcomeTick: outcomeTick, DurationMS: durationMS,
				WinnerObserved: winnerObserved, TradeCandidateCount: 1,
				TradeCompletionCount: completionCount, SurvivalStatus: "not_applicable",
				DisengagementStatus: "not_applicable",
				Horizons:            tradeHorizonOutcomes(candidate, connection.TeammatePlayerID, completion, completed),
			})
			result.QualityMasks = append(result.QualityMasks,
				causalQualityMask(matchID, decisionID, candidate.RoundNumber, candidate.DeathTick, features),
			)
		}
	}
}

func tradeHorizonOutcomes(
	candidate models.CanonicalTradeCandidate,
	actorPlayerID string,
	completion models.CanonicalTradeCompletion,
	completed bool,
) []models.CanonicalHorizonOutcome {
	result := make([]models.CanonicalHorizonOutcome, 0, 3)
	for _, seconds := range []int{2, 5, 10} {
		horizonMS := float64(seconds * 1000)
		outcome := "no_trade_completion_yet"
		if completed && completion.ElapsedMS <= horizonMS {
			outcome = "trade_completed_by_teammate"
			if completion.TraderPlayerID == actorPlayerID {
				outcome = "trade_completed_by_actor"
			}
		} else if horizonMS >= float64(candidate.WindowMS) {
			outcome = candidate.Evaluation
		}
		result = append(result, models.CanonicalHorizonOutcome{
			HorizonSeconds: seconds, Status: "derived_outcome_only",
			Outcome: outcome, Source: "trades@1",
		})
	}
	return result
}

func sortCanonicalCausalPartitions(partitions *canonicalCausalPartitions) {
	type alignedRow struct {
		decision models.CanonicalDecision
		features models.CanonicalDecisionFeatures
		oracle   models.CanonicalOracleContext
		outcome  models.CanonicalDecisionOutcome
		mask     models.CanonicalQualityMask
	}
	rows := make([]alignedRow, len(partitions.Decisions))
	for index := range rows {
		rows[index] = alignedRow{
			decision: partitions.Decisions[index], features: partitions.DecisionFeatures[index],
			oracle: partitions.OracleContext[index], outcome: partitions.Outcomes[index],
			mask: partitions.QualityMasks[index],
		}
	}
	sort.Slice(rows, func(left, right int) bool {
		if rows[left].decision.RoundNumber != rows[right].decision.RoundNumber {
			return rows[left].decision.RoundNumber < rows[right].decision.RoundNumber
		}
		if rows[left].decision.T0Tick != rows[right].decision.T0Tick {
			return rows[left].decision.T0Tick < rows[right].decision.T0Tick
		}
		return rows[left].decision.DecisionID < rows[right].decision.DecisionID
	})
	for index, row := range rows {
		partitions.Decisions[index], partitions.DecisionFeatures[index] = row.decision, row.features
		partitions.OracleContext[index], partitions.Outcomes[index] = row.oracle, row.outcome
		partitions.QualityMasks[index] = row.mask
	}
}

func causalDecisionProjection(
	matchID string,
	engagement models.CanonicalEngagement,
) (models.CanonicalDecisionFeatures, models.CanonicalQualityMask, error) {
	context := engagement.CausalContext
	maxAvailabilityTick := 0
	observedStates := 0
	aliveCount := 0
	aliveObserved := false
	var roundClock *int64
	var bombClock *int64
	for _, state := range context.ParticipantStates {
		if state.AvailabilityTick != nil {
			if *state.AvailabilityTick > context.T0Tick {
				return models.CanonicalDecisionFeatures{}, models.CanonicalQualityMask{}, fmt.Errorf(
					"decision %s has availability tick %d after t0 %d",
					engagement.EngagementID, *state.AvailabilityTick, context.T0Tick,
				)
			}
			if *state.AvailabilityTick > maxAvailabilityTick {
				maxAvailabilityTick = *state.AvailabilityTick
			}
		}
		if state.Status == "observed" {
			observedStates++
		}
		if state.IsAlive != nil {
			aliveObserved = true
			if *state.IsAlive {
				aliveCount++
			}
		}
		roundClock = causalMinimumInt64(roundClock, state.RoundClockRemainingMS)
		bombClock = causalMinimumInt64(bombClock, state.BombTimeRemainingMS)
	}
	var aliveCountValue *int
	if aliveObserved {
		value := aliveCount
		aliveCountValue = &value
	}
	features := models.CanonicalDecisionFeatures{
		SchemaID: block7DecisionFeatureSchema, MatchID: matchID,
		DecisionID: engagement.EngagementID, DecisionType: block7EngagementDecision,
		RoundNumber: engagement.RoundNumber, T0Tick: context.T0Tick,
		AvailabilityTickMax: maxAvailabilityTick,
		ParticipantCount:    len(context.ParticipantStates), ObservedParticipantStates: observedStates,
		AliveParticipantCount: aliveCountValue,
		// Exact actor-target distance is engine ground truth unless a decision-time
		// visibility observation proves it. Keep it out of the observed partition;
		// the oracle partition retains the retrospective geometry.
		InitialDistanceWorldUnits: nil,
		InitialDistanceStatus:     "unavailable",
		BombContextStatus:         context.BombContextStatus, EconomyContextStatus: context.EconomyContextStatus,
		EnemiesExposedCount: context.EnemiesExposedCount, EnemiesExposedStatus: context.EnemiesExposedStatus,
		RoundClockRemainingMS: roundClock, BombTimeRemainingMS: bombClock,
		SourceStateCount:    len(context.SourceStateIDs),
		TradePossibleStatus: "unavailable", NearestDistanceStatus: "unavailable",
		NearestConnectionStatus: "unavailable", LineOfSightStatus: "unavailable",
		FacingStatus: "unavailable",
	}
	mask := causalQualityMask(matchID, engagement.EngagementID, engagement.RoundNumber, context.T0Tick, features)
	return features, mask, nil
}

func causalMinimumInt64(current, candidate *int64) *int64 {
	if candidate == nil {
		return current
	}
	if current == nil || *candidate < *current {
		value := *candidate
		return &value
	}
	return current
}

func causalQualityMask(
	matchID, decisionID string,
	roundNumber, t0Tick int,
	features models.CanonicalDecisionFeatures,
) models.CanonicalQualityMask {
	available := []string{"participant_count", "observed_participant_states", "source_state_count"}
	unavailable := make([]string, 0, 8)
	warnings := make([]string, 0, 4)
	appendAvailability := func(name string, isAvailable bool) {
		if isAvailable {
			available = append(available, name)
		} else {
			unavailable = append(unavailable, name)
		}
	}
	appendAvailability("alive_participant_count", features.AliveParticipantCount != nil)
	appendAvailability("initial_distance_world_units", features.InitialDistanceWorldUnits != nil && features.InitialDistanceStatus != "unavailable")
	appendAvailability("enemies_exposed_count", features.EnemiesExposedCount != nil && features.EnemiesExposedStatus != "unavailable")
	appendAvailability("round_clock_remaining_ms", features.RoundClockRemainingMS != nil)
	appendAvailability("bomb_time_remaining_ms", features.BombTimeRemainingMS != nil)
	appendAvailability("bomb_context", features.BombContextStatus != "unavailable")
	appendAvailability("economy_context", features.EconomyContextStatus != "unavailable")
	appendAvailability("trade_possible", features.TradePossible != nil && features.TradePossibleStatus != "unavailable")
	appendAvailability("nearest_teammate_distance_world_units", features.NearestTeammateDistance != nil && features.NearestDistanceStatus != "unavailable")
	appendAvailability("nearest_connection_time_ms", features.NearestConnectionTimeMS != nil && features.NearestConnectionStatus != "unavailable")
	appendAvailability("any_line_of_sight", features.AnyLineOfSight != nil && features.LineOfSightStatus != "unavailable")
	appendAvailability("minimum_facing_delta_deg", features.MinimumFacingDeltaDeg != nil && features.FacingStatus != "unavailable")
	if len(unavailable) > 0 {
		warnings = append(warnings, "one or more causal observations are unavailable")
	}
	sort.Strings(available)
	sort.Strings(unavailable)
	return models.CanonicalQualityMask{
		SchemaID: block7QualityMaskSchema, MatchID: matchID, DecisionID: decisionID,
		RoundNumber: roundNumber, T0Tick: t0Tick,
		AvailableFields: available, UnavailableFields: unavailable,
		InferredFields: []string{}, WarningFlags: warnings,
	}
}
