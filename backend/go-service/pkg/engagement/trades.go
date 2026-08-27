package engagement

import (
	"math"
	"sort"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/maps"

	"github.com/golang/geo/r3"
)

type tradeCandidateWork struct {
	candidate models.CanonicalTradeCandidate
	kill      models.CanonicalCombatEvent
	endTick   *int
}

func buildTrades(
	matchID string,
	tickRate float64,
	rounds []models.CanonicalRound,
	participants []models.CanonicalParticipant,
	events []models.CanonicalCombatEvent,
	statesByRound map[int][]models.CanonicalPlayerState,
	byID map[string]models.CanonicalCombatEvent,
	visibility maps.VisibilityChecker,
) models.CanonicalTradesExport {
	config := tradeConfig(tickRate)
	roundEnds := make(map[int]*int, len(rounds))
	for _, round := range rounds {
		roundEnds[round.RoundNumber] = round.EndTick
	}
	kills := enemyKills(events)
	works := make([]tradeCandidateWork, 0, len(kills))
	for index, kill := range kills {
		candidate := newTradeCandidate(matchID, index+1, kill, config, statesByRound[kill.RoundNumber], participants, byID, visibility)
		works = append(works, tradeCandidateWork{candidate: candidate, kill: kill, endTick: roundEnds[kill.RoundNumber]})
	}
	completions := matchTradeCompletions(matchID, tickRate, works, kills, byID)
	completionByCandidate := make(map[string]models.CanonicalTradeCompletion, len(completions))
	completionByResponse := make(map[string]models.CanonicalTradeCompletion, len(completions))
	for _, completion := range completions {
		completionByCandidate[completion.TradeCandidateID] = completion
		completionByResponse[completion.ResponseKillEventID] = completion
	}
	candidates := make([]models.CanonicalTradeCandidate, 0, len(works))
	for _, work := range works {
		candidate := work.candidate
		if completion, exists := completionByCandidate[candidate.TradeCandidateID]; exists {
			candidate.Evaluation = "completed"
			candidate.TradeCompletionID = stringPointer(completion.TradeCompletionID)
		} else {
			candidate.Evaluation = evaluateCandidate(candidate, work.endTick)
		}
		if prior, exists := completionByResponse[candidate.OriginalKillEventID]; exists {
			candidate.CounterTradeOfCompletionID = stringPointer(prior.TradeCompletionID)
		}
		candidates = append(candidates, candidate)
	}
	return models.CanonicalTradesExport{
		SchemaID:    "stratai.trades@1",
		MatchID:     matchID,
		Config:      config,
		Candidates:  candidates,
		Completions: completions,
	}
}

func enemyKills(events []models.CanonicalCombatEvent) []models.CanonicalCombatEvent {
	kills := make([]models.CanonicalCombatEvent, 0)
	for _, event := range events {
		if event.EventType == "kill" && event.Relation == "enemy" &&
			event.ActorPlayerID != nil && event.TargetPlayerID != nil {
			kills = append(kills, event)
		}
	}
	return kills
}

func newTradeCandidate(
	matchID string,
	sequence int,
	kill models.CanonicalCombatEvent,
	config models.CanonicalTradeConfig,
	states []models.CanonicalPlayerState,
	participants []models.CanonicalParticipant,
	byID map[string]models.CanonicalCombatEvent,
	visibility maps.VisibilityChecker,
) models.CanonicalTradeCandidate {
	eligible, stateIDs, connections, tradePossible, status := eligibleTeammates(kill, states, participants, config, visibility)
	attemptIDs := tradeAttemptEventIDs(kill, eligible, config.TradeWindowTicks, byID)
	sources := eventClosure(kill, byID)
	for _, attemptID := range attemptIDs {
		if event, exists := byID[attemptID]; exists {
			sources = mergeEvents(sources, eventClosure(event, byID))
		}
	}
	sourceIDs := make([]string, 0, len(sources))
	for _, event := range sources {
		sourceIDs = append(sourceIDs, event.EventID)
	}
	return models.CanonicalTradeCandidate{
		TradeCandidateID:          matchID + ":trade_candidate:" + sequenceID(sequence),
		RoundID:                   kill.RoundID,
		RoundNumber:               kill.RoundNumber,
		DeathTick:                 kill.Tick,
		DeathSequenceInTick:       kill.SequenceInTick,
		OriginalKillEventID:       kill.EventID,
		OriginalVictimPlayerID:    *kill.TargetPlayerID,
		OriginalKillerPlayerID:    *kill.ActorPlayerID,
		PlayerIDUsage:             "join_only",
		EligibleTeammatePlayerIDs: eligible,
		EligibilityStatus:         status,
		EligibilitySource:         eligibilitySource(status),
		EligibilityStateIDs:       stateIDs,
		Connections:               connections,
		TradePossible:             tradePossible,
		TradePossibleStatus:       status,
		AttemptEventIDs:           attemptIDs,
		WindowMS:                  config.TradeWindowMS,
		WindowTicks:               config.TradeWindowTicks,
		WindowEndTick:             kill.Tick + config.TradeWindowTicks,
		SourceEventIDs:            sourceIDs,
	}
}

func eligibleTeammates(
	kill models.CanonicalCombatEvent,
	states []models.CanonicalPlayerState,
	participants []models.CanonicalParticipant,
	config models.CanonicalTradeConfig,
	visibility maps.VisibilityChecker,
) ([]string, []string, []models.CanonicalTradeConnection, *bool, string) {
	if kill.TargetSide == nil || kill.TargetPlayerID == nil || kill.ActorPlayerID == nil {
		return []string{}, []string{}, []models.CanonicalTradeConnection{}, nil, "unavailable"
	}

	teamByPlayer := make(map[string]string, len(participants))
	for _, participant := range participants {
		if participant.PlayerID != "" {
			teamByPlayer[participant.PlayerID] = participant.TeamID
		}
	}
	victimTeamID := teamByPlayer[*kill.TargetPlayerID]
	if victimTeamID == "" {
		return []string{}, []string{}, []models.CanonicalTradeConnection{}, nil, "unavailable"
	}

	teammateIDs := make([]string, 0)
	for playerID, teamID := range teamByPlayer {
		if playerID != *kill.TargetPlayerID && teamID == victimTeamID {
			teammateIDs = append(teammateIDs, playerID)
		}
	}
	teammateIDs = sortedUniqueStrings(teammateIDs)
	if len(teammateIDs) == 0 {
		return []string{}, []string{}, []models.CanonicalTradeConnection{}, boolPointer(false), "derived"
	}

	latest := latestStates(states, kill.Tick)
	killerState, killerStateAvailable := latest[*kill.ActorPlayerID]
	stateIDs := make([]string, 0, len(teammateIDs)+1)
	if killerStateAvailable {
		stateIDs = append(stateIDs, killerState.StateID)
	}
	connections := make([]models.CanonicalTradeConnection, 0, len(teammateIDs))
	eligible := make([]string, 0, len(teammateIDs))
	hasUnavailable := false
	for _, teammateID := range teammateIDs {
		connection := physicalTradeConnection(
			teammateID,
			latest[teammateID],
			stateExists(latest, teammateID),
			killerState,
			killerStateAvailable,
			config,
			visibility,
		)
		if connection.PlayerStateID != nil {
			stateIDs = append(stateIDs, *connection.PlayerStateID)
		}
		if connection.EligibilityStatus == "unavailable" {
			hasUnavailable = true
		}
		if connection.Eligible != nil && *connection.Eligible {
			eligible = append(eligible, teammateID)
		}
		connections = append(connections, connection)
	}
	if len(eligible) > 0 {
		return sortedUniqueStrings(eligible), sortedUniqueStrings(stateIDs), connections, boolPointer(true), "derived"
	}
	if hasUnavailable {
		return []string{}, sortedUniqueStrings(stateIDs), connections, nil, "unavailable"
	}
	return []string{}, sortedUniqueStrings(stateIDs), connections, boolPointer(false), "derived"
}

func eligibilitySource(status string) string {
	if status == "derived" {
		return "player_state@3+physics_mesh_los"
	}
	return "unavailable"
}

func stateExists(states map[string]models.CanonicalPlayerState, playerID string) bool {
	_, exists := states[playerID]
	return exists
}

func physicalTradeConnection(
	teammateID string,
	teammateState models.CanonicalPlayerState,
	teammateStateAvailable bool,
	killerState models.CanonicalPlayerState,
	killerStateAvailable bool,
	config models.CanonicalTradeConfig,
	visibility maps.VisibilityChecker,
) models.CanonicalTradeConnection {
	connection := models.CanonicalTradeConnection{
		TeammatePlayerID:     teammateID,
		PlayerIDUsage:        "join_only",
		StateStatus:          "unavailable",
		AliveStatus:          "unavailable",
		DistanceStatus:       "unavailable",
		ConnectionTimeStatus: "unavailable",
		LineOfSightStatus:    "unavailable",
		FacingStatus:         "unavailable",
		MapGeometryStatus:    "unavailable",
		EligibilityStatus:    "unavailable",
		IneligibilityReasons: []string{},
	}
	if !teammateStateAvailable {
		connection.IneligibilityReasons = []string{"teammate_state_unavailable"}
		return connection
	}
	connection.PlayerStateID = stringPointer(teammateState.StateID)
	connection.StateAvailabilityTick = intPointer(teammateState.Tick)
	connection.StateStatus = "observed"
	connection.Alive = boolPointer(teammateState.IsAlive)
	connection.AliveStatus = "observed"
	if !killerStateAvailable {
		connection.IneligibilityReasons = []string{"killer_state_unavailable"}
		return connection
	}

	distance := vectorDistance(teammateState.Position, killerState.Position)
	connection.DistanceWorldUnits = floatPointer(distance)
	connection.DistanceStatus = "derived"
	if config.AssumedMovementSpeedWorldUPS <= 0 {
		connection.IneligibilityReasons = []string{"movement_speed_config_unavailable"}
		return connection
	}
	connectionTimeMS := distance * 1000.0 / config.AssumedMovementSpeedWorldUPS
	connection.ConnectionTimeMS = floatPointer(connectionTimeMS)
	connection.ConnectionTimeStatus = "derived"
	facingDelta := facingDeltaDegrees(teammateState, killerState.Position)
	connection.FacingDeltaDeg = floatPointer(facingDelta)
	connection.FacingStatus = "derived"
	if visibility == nil || !visibility.IsLoaded() {
		connection.IneligibilityReasons = []string{"map_geometry_unavailable"}
		return connection
	}
	connection.MapGeometryStatus = "observed"
	lineOfSight := visibility.IsVisible(eyePosition(teammateState.Position), eyePosition(killerState.Position))
	connection.LineOfSight = boolPointer(lineOfSight)
	connection.LineOfSightStatus = "derived"

	reasons := make([]string, 0, 5)
	if !teammateState.IsAlive {
		reasons = append(reasons, "teammate_dead")
	}
	if distance > config.MaxDistanceWorldUnits {
		reasons = append(reasons, "distance_exceeds_threshold")
	}
	if connectionTimeMS > float64(config.TradeWindowMS) {
		reasons = append(reasons, "connection_time_exceeds_window")
	}
	if !lineOfSight {
		reasons = append(reasons, "no_line_of_sight")
	}
	if facingDelta > config.MaxFacingDeltaDeg {
		reasons = append(reasons, "facing_away")
	}
	connection.IneligibilityReasons = reasons
	connection.Eligible = boolPointer(len(reasons) == 0)
	connection.EligibilityStatus = "derived"
	return connection
}

func vectorDistance(left, right models.CanonicalVector) float64 {
	dx := right.X - left.X
	dy := right.Y - left.Y
	dz := right.Z - left.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func facingDeltaDegrees(state models.CanonicalPlayerState, target models.CanonicalVector) float64 {
	desiredYaw := math.Atan2(target.Y-state.Position.Y, target.X-state.Position.X) * 180.0 / math.Pi
	delta := math.Mod(math.Abs(float64(state.ViewYawDeg)-desiredYaw), 360.0)
	if delta > 180.0 {
		delta = 360.0 - delta
	}
	return delta
}

func eyePosition(position models.CanonicalVector) r3.Vector {
	return r3.Vector{X: position.X, Y: position.Y, Z: position.Z + 64.0}
}

func boolPointer(value bool) *bool {
	copy := value
	return &copy
}

func tradeAttemptEventIDs(
	kill models.CanonicalCombatEvent,
	eligible []string,
	windowTicks int,
	byID map[string]models.CanonicalCombatEvent,
) []string {
	eligibleSet := make(map[string]struct{}, len(eligible))
	for _, playerID := range eligible {
		eligibleSet[playerID] = struct{}{}
	}
	attempts := make([]models.CanonicalCombatEvent, 0)
	for _, event := range byID {
		if event.EventType != "player_hurt" || event.RoundNumber != kill.RoundNumber ||
			!eventBefore(kill, event) || event.Tick > kill.Tick+windowTicks ||
			event.ActorPlayerID == nil || event.TargetPlayerID == nil ||
			*event.TargetPlayerID != *kill.ActorPlayerID {
			continue
		}
		if _, exists := eligibleSet[*event.ActorPlayerID]; exists {
			attempts = append(attempts, event)
		}
	}
	sort.Slice(attempts, func(i, j int) bool { return eventLess(attempts[i], attempts[j]) })
	ids := make([]string, 0, len(attempts))
	for _, event := range attempts {
		ids = append(ids, event.EventID)
	}
	return ids
}

func matchTradeCompletions(
	matchID string,
	tickRate float64,
	works []tradeCandidateWork,
	kills []models.CanonicalCombatEvent,
	byID map[string]models.CanonicalCombatEvent,
) []models.CanonicalTradeCompletion {
	consumedCandidates := make(map[string]struct{})
	completions := make([]models.CanonicalTradeCompletion, 0)
	for _, response := range kills {
		best := -1
		bestElapsed := 0
		for index := range works {
			candidate := works[index].candidate
			if _, consumed := consumedCandidates[candidate.TradeCandidateID]; consumed ||
				response.RoundNumber != candidate.RoundNumber || !eventBefore(works[index].kill, response) ||
				response.Tick-candidate.DeathTick > candidate.WindowTicks ||
				response.TargetPlayerID == nil || *response.TargetPlayerID != candidate.OriginalKillerPlayerID ||
				response.ActorSide == nil || works[index].kill.TargetSide == nil ||
				*response.ActorSide != *works[index].kill.TargetSide ||
				response.ActorPlayerID == nil || !containsString(candidate.EligibleTeammatePlayerIDs, *response.ActorPlayerID) {
				continue
			}
			elapsed := response.Tick - candidate.DeathTick
			if best < 0 || elapsed < bestElapsed || (elapsed == bestElapsed && candidate.TradeCandidateID < works[best].candidate.TradeCandidateID) {
				best = index
				bestElapsed = elapsed
			}
		}
		if best < 0 {
			continue
		}
		candidate := works[best].candidate
		consumedCandidates[candidate.TradeCandidateID] = struct{}{}
		completionID := matchID + ":trade_completion:" + sequenceID(len(completions)+1)
		sources := mergeEvents(eventClosure(works[best].kill, byID), eventClosure(response, byID))
		sourceIDs := make([]string, 0, len(sources))
		for _, event := range sources {
			sourceIDs = append(sourceIDs, event.EventID)
		}
		completions = append(completions, models.CanonicalTradeCompletion{
			TradeCompletionID:      completionID,
			TradeCandidateID:       candidate.TradeCandidateID,
			RoundID:                candidate.RoundID,
			RoundNumber:            candidate.RoundNumber,
			OriginalKillEventID:    candidate.OriginalKillEventID,
			ResponseKillEventID:    response.EventID,
			OriginalVictimPlayerID: candidate.OriginalVictimPlayerID,
			OriginalKillerPlayerID: candidate.OriginalKillerPlayerID,
			TraderPlayerID:         *response.ActorPlayerID,
			TradeRelation:          "teammate",
			ElapsedTicks:           bestElapsed,
			ElapsedMS:              float64(bestElapsed) * 1000.0 / tickRate,
			SourceEventIDs:         sourceIDs,
		})
	}
	return completions
}

func evaluateCandidate(candidate models.CanonicalTradeCandidate, roundEnd *int) string {
	if candidate.TradePossibleStatus == "unavailable" || candidate.TradePossible == nil {
		return "not_evaluable"
	}
	if !*candidate.TradePossible {
		return "not_tradeable"
	}
	if roundEnd == nil || candidate.WindowEndTick > *roundEnd {
		return "not_evaluable"
	}
	if len(candidate.AttemptEventIDs) > 0 {
		return "failed"
	}
	return "not_attempted"
}

func linkTradesToEngagements(
	engagements []models.CanonicalEngagement,
	trades models.CanonicalTradesExport,
) {
	candidateByKill := make(map[string]string, len(trades.Candidates))
	for _, candidate := range trades.Candidates {
		candidateByKill[candidate.OriginalKillEventID] = candidate.TradeCandidateID
	}
	completionByKill := make(map[string]string, len(trades.Completions))
	for _, completion := range trades.Completions {
		completionByKill[completion.ResponseKillEventID] = completion.TradeCompletionID
	}
	for index := range engagements {
		engagement := &engagements[index]
		for _, killID := range engagement.OutcomeContext.TerminalKillEventIDs {
			if candidateID, exists := candidateByKill[killID]; exists {
				engagement.OutcomeContext.TradeCandidateIDs = append(engagement.OutcomeContext.TradeCandidateIDs, candidateID)
			}
			if completionID, exists := completionByKill[killID]; exists {
				engagement.OutcomeContext.TradeCompletionIDs = append(engagement.OutcomeContext.TradeCompletionIDs, completionID)
			}
		}
		engagement.OutcomeContext.TradeCandidateIDs = sortedUniqueStrings(engagement.OutcomeContext.TradeCandidateIDs)
		engagement.OutcomeContext.TradeCompletionIDs = sortedUniqueStrings(engagement.OutcomeContext.TradeCompletionIDs)
	}
}
