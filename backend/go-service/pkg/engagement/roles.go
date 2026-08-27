package engagement

import (
	"sort"

	"cs2-demo-service/models"
)

func roleFromEvent(
	event models.CanonicalCombatEvent,
	status string,
	source string,
	confidence float64,
) models.CanonicalRoleAssignment {
	return models.CanonicalRoleAssignment{
		PlayerID:         event.ActorPlayerID,
		Status:           status,
		Source:           source,
		AvailabilityTick: intPointer(event.Tick),
		SourceEventIDs:   []string{event.EventID},
		Confidence:       floatPointer(confidence),
	}
}

func unavailableRole() models.CanonicalRoleAssignment {
	return models.CanonicalRoleAssignment{
		Status:         "unavailable",
		Source:         "unavailable",
		SourceEventIDs: []string{},
	}
}

func firstAggressorRole(
	group *engagementGroup,
	events []models.CanonicalCombatEvent,
) models.CanonicalRoleAssignment {
	first := group.seeds[0].hurt
	if group.prelude != nil {
		role := roleFromEvent(*group.prelude, "inferred", "unambiguous_two_player_miss_prelude", 0.65)
		role.AvailabilityTick = intPointer(first.Tick)
		return role
	}
	for _, event := range events {
		if event.EventType != "weapon_fire" || event.ActorPlayerID == nil || eventAfter(event, first) {
			continue
		}
		role := roleFromEvent(event, "observed", "causal_shot_ancestor", 1.0)
		role.AvailabilityTick = intPointer(first.Tick)
		role.SourceEventIDs = []string{event.EventID, first.EventID}
		return role
	}
	return unavailableRole()
}

func initiatorRole(
	firstAggressor models.CanonicalRoleAssignment,
	firstDamage models.CanonicalRoleAssignment,
) models.CanonicalRoleAssignment {
	if firstAggressor.PlayerID != nil {
		role := firstAggressor
		role.Source = "first_causal_offensive_action"
		return role
	}
	if firstDamage.PlayerID == nil {
		return unavailableRole()
	}
	role := firstDamage
	role.Status = "inferred"
	role.Source = "first_damage_fallback"
	role.Confidence = floatPointer(0.75)
	return role
}

func buildOutcome(
	group *engagementGroup,
	states []models.CanonicalPlayerState,
) models.CanonicalEngagementOutcomeContext {
	kills := make([]models.CanonicalCombatEvent, 0)
	for _, seed := range group.seeds {
		if seed.kill != nil {
			kills = append(kills, *seed.kill)
		}
	}
	if len(kills) == 0 {
		return nonKillOutcome(group, states)
	}
	winnerIDs := make(map[string]struct{})
	loserIDs := make([]string, 0, len(kills))
	killIDs := make([]string, 0, len(kills))
	for _, kill := range kills {
		if kill.ActorPlayerID != nil {
			winnerIDs[*kill.ActorPlayerID] = struct{}{}
		}
		if kill.TargetPlayerID != nil {
			loserIDs = append(loserIDs, *kill.TargetPlayerID)
		}
		killIDs = append(killIDs, kill.EventID)
	}
	var winner *string
	if len(winnerIDs) == 1 {
		for playerID := range winnerIDs {
			winner = stringPointer(playerID)
		}
	}
	return models.CanonicalEngagementOutcomeContext{
		Outcome:              "kill",
		WinnerPlayerID:       winner,
		LoserPlayerIDs:       sortedUniqueStrings(loserIDs),
		TerminalKillEventIDs: sortedUniqueStrings(killIDs),
		TradeCandidateIDs:    []string{},
		TradeCompletionIDs:   []string{},
		SurvivalStatus:       "observed_terminal_kill",
		DisengagementStatus:  "not_applicable",
	}
}

func nonKillOutcome(
	group *engagementGroup,
	states []models.CanonicalPlayerState,
) models.CanonicalEngagementOutcomeContext {
	lastTick := group.seeds[len(group.seeds)-1].hurt.Tick
	participants := groupParticipants(group)
	lastStates := latestStates(states, lastTick)
	allAlive := len(lastStates) > 0
	for playerID := range participants {
		state, exists := lastStates[playerID]
		if !exists || !state.IsAlive {
			allAlive = false
			break
		}
	}
	outcome := "disengaged"
	survival := "unavailable"
	if allAlive {
		survival = "observed_alive_at_last_exchange"
	}
	return models.CanonicalEngagementOutcomeContext{
		Outcome:              outcome,
		LoserPlayerIDs:       []string{},
		TerminalKillEventIDs: []string{},
		TradeCandidateIDs:    []string{},
		TradeCompletionIDs:   []string{},
		SurvivalStatus:       survival,
		DisengagementStatus:  "derived_window_closed_without_kill",
	}
}

func applyRoles(
	participants []models.CanonicalEngagementParticipant,
	initiator models.CanonicalRoleAssignment,
	firstAggressor models.CanonicalRoleAssignment,
	firstDamage models.CanonicalRoleAssignment,
	outcome models.CanonicalEngagementOutcomeContext,
) {
	for index := range participants {
		participant := &participants[index]
		addRole(participant, "initiator", initiator.PlayerID)
		addRole(participant, "first_aggressor", firstAggressor.PlayerID)
		addRole(participant, "first_damage_dealer", firstDamage.PlayerID)
		addRole(participant, "winner", outcome.WinnerPlayerID)
		for _, loserID := range outcome.LoserPlayerIDs {
			if participant.PlayerID == loserID {
				participant.Roles = append(participant.Roles, "loser")
			}
		}
		sort.Strings(participant.Roles)
	}
}

func addRole(
	participant *models.CanonicalEngagementParticipant,
	role string,
	playerID *string,
) {
	if playerID != nil && participant.PlayerID == *playerID {
		participant.Roles = append(participant.Roles, role)
	}
}

func engagementType(group *engagementGroup) string {
	shotTargets := make(map[string]map[string]struct{})
	actorTargets := make(map[string]map[string]struct{})
	for _, seed := range group.seeds {
		actorID := *seed.hurt.ActorPlayerID
		targetID := *seed.hurt.TargetPlayerID
		if _, exists := actorTargets[actorID]; !exists {
			actorTargets[actorID] = make(map[string]struct{})
		}
		actorTargets[actorID][targetID] = struct{}{}
		if seed.hurt.ShotID != nil {
			if _, exists := shotTargets[*seed.hurt.ShotID]; !exists {
				shotTargets[*seed.hurt.ShotID] = make(map[string]struct{})
			}
			shotTargets[*seed.hurt.ShotID][targetID] = struct{}{}
		}
	}
	for _, targets := range shotTargets {
		if len(targets) > 1 {
			return "collateral"
		}
	}
	for _, targets := range actorTargets {
		if len(targets) > 1 {
			return "multi_target"
		}
	}
	if len(groupParticipants(group)) == 2 {
		return "duel"
	}
	return "engagement"
}
