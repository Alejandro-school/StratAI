package engagement

import (
	"math"
	"sort"

	"cs2-demo-service/models"
)

func buildCausalContext(
	start models.CanonicalCombatEvent,
	participants []models.CanonicalEngagementParticipant,
	exchanges []models.CanonicalEngagementExchange,
	states []models.CanonicalPlayerState,
) models.CanonicalEngagementCausalContext {
	latest := latestStates(states, start.Tick)
	participantStates := make([]models.CanonicalEngagementParticipantState, 0, len(participants))
	observedPositions := make([]models.CanonicalVector, 0, len(participants))
	sourceStateIDs := make([]string, 0, len(participants))
	bombStatus := "unavailable"
	for _, participant := range participants {
		state, exists := latest[participant.PlayerID]
		if !exists {
			participantStates = append(participantStates, unavailableParticipantState(participant.PlayerID))
			continue
		}
		participantState := causalParticipantState(state)
		participantStates = append(participantStates, participantState)
		observedPositions = append(observedPositions, state.Position)
		sourceStateIDs = append(sourceStateIDs, state.StateID)
		if state.ObjectivePhase != "" {
			bombStatus = "observed"
		}
	}
	context := models.CanonicalEngagementCausalContext{
		T0Tick:               start.Tick,
		T0SequenceInTick:     start.SequenceInTick,
		ParticipantStates:    participantStates,
		BombContextStatus:    bombStatus,
		EconomyContextStatus: "unavailable",
		EnemiesExposedStatus: "unavailable",
		SourceStateIDs:       sortedUniqueStrings(sourceStateIDs),
	}
	if len(participants) == 2 && len(observedPositions) == 2 {
		dx := observedPositions[0].X - observedPositions[1].X
		dy := observedPositions[0].Y - observedPositions[1].Y
		dz := observedPositions[0].Z - observedPositions[1].Z
		distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
		context.InitialDistanceWorldUnits = floatPointer(distance)
		context.InitialDistanceStatus = "derived"
		context.InitialDistanceSource = "player_state@3"
	} else if len(exchanges) > 0 &&
		exchanges[0].DistanceWorldUnits != nil &&
		(exchanges[0].Tick < start.Tick ||
			(exchanges[0].Tick == start.Tick && exchanges[0].SequenceInTick <= start.SequenceInTick)) {
		context.InitialDistanceWorldUnits = exchanges[0].DistanceWorldUnits
		context.InitialDistanceStatus = "derived"
		context.InitialDistanceSource = "first_exchange_positions"
	} else {
		context.InitialDistanceStatus = "unavailable"
		context.InitialDistanceSource = "unavailable"
	}
	return context
}

func latestStates(
	states []models.CanonicalPlayerState,
	tick int,
) map[string]models.CanonicalPlayerState {
	latest := make(map[string]models.CanonicalPlayerState)
	for _, state := range states {
		if state.Tick > tick {
			continue
		}
		current, exists := latest[state.PlayerID]
		if !exists || state.Tick > current.Tick || (state.Tick == current.Tick && state.StateID < current.StateID) {
			latest[state.PlayerID] = state
		}
	}
	return latest
}

func causalParticipantState(state models.CanonicalPlayerState) models.CanonicalEngagementParticipantState {
	position := state.Position
	health := state.Health
	armor := state.Armor
	isAlive := state.IsAlive
	objectivePhase := state.ObjectivePhase
	result := models.CanonicalEngagementParticipantState{
		PlayerID:                   state.PlayerID,
		StateID:                    stringPointer(state.StateID),
		AvailabilityTick:           intPointer(state.Tick),
		Status:                     "observed",
		Source:                     "player_state@3",
		Side:                       stringPointer(state.Side),
		Position:                   &position,
		PositionStatus:             "observed",
		HorizontalVelocityWorldUPS: state.HorizontalVelocityWorldUPS,
		VelocityMeasurementTicks:   state.VelocityMeasurementTicks,
		ActiveWeapon:               state.ActiveWeapon,
		ActiveWeaponStatus:         state.ActiveWeaponStatus,
		ActiveWeaponSource:         "unavailable",
		Health:                     &health,
		Armor:                      &armor,
		IsAlive:                    &isAlive,
		ObjectivePhase:             &objectivePhase,
		RoundClockRemainingMS:      state.RoundClockRemainingMS,
		BombTimeRemainingMS:        state.BombTimeRemainingMS,
	}
	if state.HorizontalVelocityWorldUPS == nil {
		result.VelocityStatus = "unavailable"
		result.VelocitySource = state.VelocitySource
	} else {
		result.VelocityStatus = "observed"
		result.VelocitySource = state.VelocitySource
		classification := "hold"
		if *state.HorizontalVelocityWorldUPS > PeekVelocityThresholdUPS {
			classification = "peek"
		}
		result.MovementClassification = stringPointer(classification)
	}
	if state.ActiveWeapon != nil && state.ActiveWeaponStatus == "observed" {
		result.ActiveWeaponSource = "player_state@3"
	}
	return result
}

func unavailableParticipantState(playerID string) models.CanonicalEngagementParticipantState {
	return models.CanonicalEngagementParticipantState{
		PlayerID:           playerID,
		Status:             "unavailable",
		Source:             "unavailable",
		PositionStatus:     "unavailable",
		VelocityStatus:     "unavailable",
		VelocitySource:     "unavailable",
		ActiveWeaponStatus: "unavailable",
		ActiveWeaponSource: "unavailable",
	}
}

func sortedPlayerStates(states []models.CanonicalPlayerState) []models.CanonicalPlayerState {
	result := append([]models.CanonicalPlayerState(nil), states...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].RoundNumber != result[j].RoundNumber {
			return result[i].RoundNumber < result[j].RoundNumber
		}
		if result[i].Tick != result[j].Tick {
			return result[i].Tick < result[j].Tick
		}
		return result[i].PlayerID < result[j].PlayerID
	})
	return result
}
