package engagement

import (
	"fmt"
	"sort"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/maps"
)

type damageSeed struct {
	hurt    models.CanonicalCombatEvent
	kill    *models.CanonicalCombatEvent
	closure []models.CanonicalCombatEvent
}

type engagementGroup struct {
	seeds   []damageSeed
	prelude *models.CanonicalCombatEvent
}

func Derive(
	matchID string,
	tickRate float64,
	rounds []models.CanonicalRound,
	participants []models.CanonicalParticipant,
	combatEvents []models.CanonicalCombatEvent,
	statesByRound map[int][]models.CanonicalPlayerState,
	visibility ...maps.VisibilityChecker,
) (models.CanonicalEngagementsExport, models.CanonicalTradesExport, error) {
	if tickRate <= 0 {
		return models.CanonicalEngagementsExport{}, models.CanonicalTradesExport{}, fmt.Errorf("tick rate must be positive")
	}
	events := sortedEvents(combatEvents)
	byID := make(map[string]models.CanonicalCombatEvent, len(events))
	for _, event := range events {
		if event.EventID == "" {
			return models.CanonicalEngagementsExport{}, models.CanonicalTradesExport{}, fmt.Errorf("combat event has empty ID")
		}
		if _, exists := byID[event.EventID]; exists {
			return models.CanonicalEngagementsExport{}, models.CanonicalTradesExport{}, fmt.Errorf("duplicate combat event ID %s", event.EventID)
		}
		byID[event.EventID] = event
	}

	seeds := buildDamageSeeds(events, byID)
	config := engagementConfig(tickRate)
	groups := groupDamageSeeds(seeds, config)
	attachAggressorPreludes(groups, events, config)
	engagements := buildEngagements(matchID, tickRate, groups, statesByRound)
	var visibilityChecker maps.VisibilityChecker
	if len(visibility) > 0 {
		visibilityChecker = visibility[0]
	}
	trades := buildTrades(matchID, tickRate, rounds, participants, events, statesByRound, byID, visibilityChecker)
	linkTradesToEngagements(engagements, trades)

	return models.CanonicalEngagementsExport{
		SchemaID:    "stratai.engagements@2",
		MatchID:     matchID,
		Config:      config,
		Engagements: engagements,
	}, trades, nil
}

func buildDamageSeeds(
	events []models.CanonicalCombatEvent,
	byID map[string]models.CanonicalCombatEvent,
) []damageSeed {
	killByHurt := make(map[string]models.CanonicalCombatEvent)
	for _, event := range events {
		if event.EventType != "kill" {
			continue
		}
		for _, sourceID := range event.SourceEventIDs {
			source, exists := byID[sourceID]
			if exists && source.EventType == "player_hurt" {
				killByHurt[sourceID] = event
			}
		}
	}

	seeds := make([]damageSeed, 0)
	for _, event := range events {
		if event.EventType != "player_hurt" || event.Relation != "enemy" ||
			event.ActorPlayerID == nil || event.TargetPlayerID == nil {
			continue
		}
		seed := damageSeed{hurt: event, closure: eventClosure(event, byID)}
		if kill, exists := killByHurt[event.EventID]; exists {
			killCopy := kill
			seed.kill = &killCopy
			seed.closure = mergeEvents(seed.closure, eventClosure(kill, byID))
		}
		seeds = append(seeds, seed)
	}
	return seeds
}

func mergeEvents(groups ...[]models.CanonicalCombatEvent) []models.CanonicalCombatEvent {
	byID := make(map[string]models.CanonicalCombatEvent)
	for _, group := range groups {
		for _, event := range group {
			byID[event.EventID] = event
		}
	}
	result := make([]models.CanonicalCombatEvent, 0, len(byID))
	for _, event := range byID {
		result = append(result, event)
	}
	sort.Slice(result, func(i, j int) bool { return eventLess(result[i], result[j]) })
	return result
}

func groupDamageSeeds(seeds []damageSeed, config models.CanonicalEngagementConfig) []*engagementGroup {
	groups := make([]*engagementGroup, 0)
	for _, seed := range seeds {
		bestIndex := -1
		bestRelation := 0
		for index, group := range groups {
			relation := groupRelation(group, seed, config)
			if relation > bestRelation {
				bestIndex = index
				bestRelation = relation
			}
		}
		if bestIndex < 0 {
			groups = append(groups, &engagementGroup{seeds: []damageSeed{seed}})
			continue
		}
		groups[bestIndex].seeds = append(groups[bestIndex].seeds, seed)
	}
	return groups
}

func groupRelation(group *engagementGroup, seed damageSeed, config models.CanonicalEngagementConfig) int {
	first := group.seeds[0].hurt
	if first.RoundNumber != seed.hurt.RoundNumber || seed.hurt.Tick-first.Tick > config.MaxEngagementDurationTick {
		return 0
	}
	last := group.seeds[len(group.seeds)-1].hurt
	for _, existing := range group.seeds {
		if sameShot(existing.hurt, seed.hurt) {
			return 3
		}
	}
	if samePair(last, seed.hurt) && seed.hurt.Tick-last.Tick <= config.PairContinuationTicks {
		return 2
	}
	if sameActor(last, seed.hurt) && seed.hurt.Tick-last.Tick <= config.MultiTargetWindowTicks {
		return 1
	}
	return 0
}

func sameShot(left, right models.CanonicalCombatEvent) bool {
	return left.ShotID != nil && right.ShotID != nil && *left.ShotID == *right.ShotID
}

func samePair(left, right models.CanonicalCombatEvent) bool {
	if left.ActorPlayerID == nil || left.TargetPlayerID == nil || right.ActorPlayerID == nil || right.TargetPlayerID == nil {
		return false
	}
	return (*left.ActorPlayerID == *right.ActorPlayerID && *left.TargetPlayerID == *right.TargetPlayerID) ||
		(*left.ActorPlayerID == *right.TargetPlayerID && *left.TargetPlayerID == *right.ActorPlayerID)
}

func sameActor(left, right models.CanonicalCombatEvent) bool {
	return left.ActorPlayerID != nil && right.ActorPlayerID != nil &&
		*left.ActorPlayerID == *right.ActorPlayerID && left.TargetPlayerID != nil && right.TargetPlayerID != nil &&
		*left.TargetPlayerID != *right.TargetPlayerID
}

func attachAggressorPreludes(
	groups []*engagementGroup,
	events []models.CanonicalCombatEvent,
	config models.CanonicalEngagementConfig,
) {
	type candidate struct {
		group *engagementGroup
		event models.CanonicalCombatEvent
	}
	candidates := make([]candidate, 0)
	ownerCounts := make(map[string]int)
	for _, group := range groups {
		if len(groupParticipants(group)) != 2 || hasCausalShotAncestor(group) {
			continue
		}
		first := group.seeds[0].hurt
		for _, event := range events {
			if event.RoundNumber != first.RoundNumber || event.Tick > first.Tick || event.Tick < first.Tick-config.AggressorPreludeTicks {
				continue
			}
			last := group.seeds[len(group.seeds)-1].hurt
			if !eventBefore(event, first) || last.Tick-event.Tick > config.MaxEngagementDurationTick {
				continue
			}
			if !validPrelude(event, group, groups) {
				continue
			}
			candidates = append(candidates, candidate{group: group, event: event})
			ownerCounts[event.EventID]++
		}
	}
	assigned := make(map[*engagementGroup]struct{})
	for _, candidate := range candidates {
		if ownerCounts[candidate.event.EventID] != 1 {
			continue
		}
		if _, exists := assigned[candidate.group]; exists {
			continue
		}
		eventCopy := candidate.event
		candidate.group.prelude = &eventCopy
		assigned[candidate.group] = struct{}{}
	}
}

func hasCausalShotAncestor(group *engagementGroup) bool {
	first := group.seeds[0].hurt
	for _, seed := range group.seeds {
		for _, event := range seed.closure {
			if event.EventType == "weapon_fire" && event.ActorPlayerID != nil && !eventAfter(event, first) {
				return true
			}
		}
	}
	return false
}

func validPrelude(event models.CanonicalCombatEvent, group *engagementGroup, groups []*engagementGroup) bool {
	if event.EventType != "weapon_fire" || event.ActorPlayerID == nil || event.ActorSide == nil ||
		event.ShotResult == nil || *event.ShotResult != "miss" {
		return false
	}
	participants := groupParticipants(group)
	if _, exists := participants[*event.ActorPlayerID]; !exists {
		return false
	}
	first := group.seeds[0].hurt
	var opposingSide *string
	if *event.ActorPlayerID == *first.ActorPlayerID {
		opposingSide = first.TargetSide
	} else if *event.ActorPlayerID == *first.TargetPlayerID {
		opposingSide = first.ActorSide
	}
	if opposingSide == nil || *opposingSide == *event.ActorSide {
		return false
	}
	for _, other := range groups {
		if other == group || other.seeds[0].hurt.RoundNumber != event.RoundNumber {
			continue
		}
		otherParticipants := groupParticipants(other)
		if _, exists := otherParticipants[*event.ActorPlayerID]; !exists {
			continue
		}
		if event.Tick >= other.seeds[0].hurt.Tick && event.Tick <= other.seeds[len(other.seeds)-1].hurt.Tick {
			return false
		}
	}
	return true
}

func groupParticipants(group *engagementGroup) map[string]struct{} {
	participants := make(map[string]struct{})
	for _, seed := range group.seeds {
		participants[*seed.hurt.ActorPlayerID] = struct{}{}
		participants[*seed.hurt.TargetPlayerID] = struct{}{}
	}
	return participants
}
