package engagement

import (
	"fmt"
	"math"
	"sort"

	"cs2-demo-service/models"
)

func buildEngagements(
	matchID string,
	tickRate float64,
	groups []*engagementGroup,
	statesByRound map[int][]models.CanonicalPlayerState,
) []models.CanonicalEngagement {
	engagements := make([]models.CanonicalEngagement, 0, len(groups))
	for _, group := range groups {
		engagements = append(engagements, buildEngagement(
			"",
			matchID,
			tickRate,
			group,
			statesByRound[group.seeds[0].hurt.RoundNumber],
		))
	}
	sort.Slice(engagements, func(i, j int) bool {
		left := engagements[i]
		right := engagements[j]
		if left.RoundNumber != right.RoundNumber {
			return left.RoundNumber < right.RoundNumber
		}
		if left.StartTick != right.StartTick {
			return left.StartTick < right.StartTick
		}
		if left.StartSequenceInTick != right.StartSequenceInTick {
			return left.StartSequenceInTick < right.StartSequenceInTick
		}
		return left.SourceEventIDs[0] < right.SourceEventIDs[0]
	})
	for index := range engagements {
		engagements[index].EngagementID = matchID + ":engagement:" + sequenceID(index+1)
	}
	return engagements
}

func buildEngagement(
	engagementID string,
	matchID string,
	tickRate float64,
	group *engagementGroup,
	states []models.CanonicalPlayerState,
) models.CanonicalEngagement {
	exchanges := make([]models.CanonicalEngagementExchange, 0, len(group.seeds))
	allEvents := make([]models.CanonicalCombatEvent, 0)
	for _, seed := range group.seeds {
		exchanges = append(exchanges, buildExchange(seed))
		allEvents = mergeEvents(allEvents, seed.closure)
	}
	if group.prelude != nil {
		allEvents = mergeEvents(allEvents, []models.CanonicalCombatEvent{*group.prelude})
	}
	start := allEvents[0]
	end := allEvents[len(allEvents)-1]
	participants := buildParticipants(group)
	firstDamage := roleFromEvent(group.seeds[0].hurt, "observed", "first_player_hurt", 1.0)
	firstAggressor := firstAggressorRole(group, allEvents)
	initiator := initiatorRole(firstAggressor, firstDamage)
	outcome := buildOutcome(group, states)
	applyRoles(participants, initiator, firstAggressor, firstDamage, outcome)
	causalContext := buildCausalContext(start, causalParticipants(start), exchanges, states)

	sourceIDs := make([]string, 0, len(allEvents))
	for _, event := range allEvents {
		sourceIDs = append(sourceIDs, event.EventID)
	}
	return models.CanonicalEngagement{
		EngagementID:        engagementID,
		RoundID:             matchID + ":round:" + roundID(group.seeds[0].hurt.RoundNumber),
		RoundNumber:         group.seeds[0].hurt.RoundNumber,
		StartTick:           start.Tick,
		StartSequenceInTick: start.SequenceInTick,
		EndTick:             end.Tick,
		EndSequenceInTick:   end.SequenceInTick,
		DurationMS:          float64(end.Tick-start.Tick) * 1000.0 / tickRate,
		EngagementType:      engagementType(group),
		Initiator:           initiator,
		FirstAggressor:      firstAggressor,
		FirstDamageDealer:   firstDamage,
		Participants:        participants,
		Exchanges:           exchanges,
		CausalContext:       causalContext,
		OutcomeContext:      outcome,
		SourceEventIDs:      sourceIDs,
		AlgorithmVersion:    AlgorithmVersion,
	}
}

func buildExchange(seed damageSeed) models.CanonicalEngagementExchange {
	hurt := seed.hurt
	exchange := models.CanonicalEngagementExchange{
		ExchangeID:           hurt.EventID,
		Tick:                 hurt.Tick,
		SequenceInTick:       hurt.SequenceInTick,
		SequenceInRound:      hurt.SequenceInRound,
		ActorPlayerID:        *hurt.ActorPlayerID,
		TargetPlayerID:       *hurt.TargetPlayerID,
		Weapon:               hurt.Weapon,
		WeaponStatus:         hurt.WeaponStatus,
		WeaponSource:         hurt.WeaponSource,
		HealthDamage:         hurt.HealthDamage,
		HealthDamageTaken:    hurt.HealthDamageTaken,
		ArmorDamage:          hurt.ArmorDamage,
		ArmorDamageTaken:     hurt.ArmorDamageTaken,
		HealthBefore:         hurt.HealthBefore,
		HealthAfter:          hurt.HealthAfter,
		ArmorBefore:          hurt.ArmorBefore,
		ArmorAfter:           hurt.ArmorAfter,
		Hitgroup:             hurt.Hitgroup,
		HitgroupStatus:       hurt.HitgroupStatus,
		IsHeadshot:           hurt.IsHeadshot,
		ShotID:               hurt.ShotID,
		ActorPosition:        hurt.ActorPosition,
		ActorPositionStatus:  hurt.ActorPositionStatus,
		ActorPositionSource:  hurt.ActorPositionSource,
		TargetPosition:       hurt.TargetPosition,
		TargetPositionStatus: hurt.TargetPositionStatus,
		TargetPositionSource: hurt.TargetPositionSource,
		ReactionTimeMS:       hurt.ReactionTimeMS,
		TimeToDamageMS:       hurt.TimeToDamageMS,
	}
	if seed.kill != nil {
		exchange.IsKill = true
		exchange.KillEventID = stringPointer(seed.kill.EventID)
	}
	for _, event := range seed.closure {
		exchange.SourceEventIDs = append(exchange.SourceEventIDs, event.EventID)
		if event.EventType == "bullet_damage" && exchange.FirstImpactEventID == nil {
			exchange.FirstImpactEventID = stringPointer(event.EventID)
		}
	}
	setExchangeDistance(&exchange)
	setExchangeTimingAvailability(&exchange, hurt.Tick)
	return exchange
}

func setExchangeDistance(exchange *models.CanonicalEngagementExchange) {
	if exchange.ActorPosition == nil || exchange.TargetPosition == nil ||
		exchange.ActorPositionStatus != "observed" || exchange.TargetPositionStatus != "observed" {
		exchange.DistanceStatus = "unavailable"
		exchange.DistanceSource = "unavailable"
		return
	}
	dx := exchange.ActorPosition.X - exchange.TargetPosition.X
	dy := exchange.ActorPosition.Y - exchange.TargetPosition.Y
	dz := exchange.ActorPosition.Z - exchange.TargetPosition.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	exchange.DistanceWorldUnits = floatPointer(distance)
	exchange.DistanceStatus = "derived"
	exchange.DistanceSource = "combat_event_positions"
}

func setExchangeTimingAvailability(exchange *models.CanonicalEngagementExchange, tick int) {
	if exchange.ReactionTimeMS == nil {
		exchange.ReactionStatus = "unavailable"
		exchange.ReactionSource = "unavailable"
	} else {
		exchange.ReactionStatus = "derived"
		exchange.ReactionSource = "combat_event@2"
		exchange.ReactionAvailabilityTick = intPointer(tick)
	}
	if exchange.TimeToDamageMS == nil {
		exchange.TimeToDamageStatus = "unavailable"
		exchange.TimeToDamageSource = "unavailable"
	} else {
		exchange.TimeToDamageStatus = "derived"
		exchange.TimeToDamageSource = "combat_event@2"
		exchange.TimeToDamageAvailabilityTick = intPointer(tick)
	}
}

func buildParticipants(group *engagementGroup) []models.CanonicalEngagementParticipant {
	sides := make(map[string]*string)
	for _, seed := range group.seeds {
		sides[*seed.hurt.ActorPlayerID] = seed.hurt.ActorSide
		sides[*seed.hurt.TargetPlayerID] = seed.hurt.TargetSide
	}
	ids := make([]string, 0, len(sides))
	for playerID := range sides {
		ids = append(ids, playerID)
	}
	sort.Strings(ids)
	participants := make([]models.CanonicalEngagementParticipant, 0, len(ids))
	for _, playerID := range ids {
		participants = append(participants, models.CanonicalEngagementParticipant{
			PlayerID: playerID,
			Side:     sides[playerID],
			Roles:    []string{},
		})
	}
	return participants
}

// causalParticipants deliberately uses only the event that defines T0. The
// complete engagement participant list is an outcome-side projection and may
// include players who first appear after T0.
func causalParticipants(start models.CanonicalCombatEvent) []models.CanonicalEngagementParticipant {
	sides := make(map[string]*string, 2)
	if start.ActorPlayerID != nil && *start.ActorPlayerID != "" {
		sides[*start.ActorPlayerID] = start.ActorSide
	}
	if start.TargetPlayerID != nil && *start.TargetPlayerID != "" {
		sides[*start.TargetPlayerID] = start.TargetSide
	}
	ids := make([]string, 0, len(sides))
	for playerID := range sides {
		ids = append(ids, playerID)
	}
	sort.Strings(ids)
	participants := make([]models.CanonicalEngagementParticipant, 0, len(ids))
	for _, playerID := range ids {
		participants = append(participants, models.CanonicalEngagementParticipant{
			PlayerID: playerID,
			Side:     sides[playerID],
			Roles:    []string{},
		})
	}
	return participants
}

func sequenceID(sequence int) string {
	return fmt.Sprintf("%06d", sequence)
}

func roundID(round int) string {
	return fmt.Sprintf("%03d", round)
}
