package handlers

import (
	"testing"

	"cs2-demo-service/pkg/objective"
)

func TestObjectiveClockUsesTheCausalClockForEveryPhase(t *testing.T) {
	const (
		currentTick = 704
		freezeTick  = 64
		tickRate    = 64.0
	)
	tests := []struct {
		name           string
		state          objective.Snapshot
		phase          ObjectiveClockPhase
		phaseRemaining float64
		roundRemaining *float64
		bombRemaining  *float64
	}{
		{
			name:           "preplant",
			state:          objective.Snapshot{State: objective.StateCarried, Phase: objective.PhasePreplant},
			phase:          ObjectiveClockPreplant,
			phaseRemaining: 105,
			roundRemaining: floatPointer(105),
		},
		{
			name:           "planting",
			state:          objective.Snapshot{State: objective.StatePlanting, Phase: objective.PhasePlanting},
			phase:          ObjectiveClockPlanting,
			phaseRemaining: 105,
			roundRemaining: floatPointer(105),
		},
		{
			name:           "planted",
			state:          objective.Snapshot{State: objective.StatePlanted, Phase: objective.PhasePlanted, IsPlantedNow: true, PlantTick: 640},
			phase:          ObjectiveClockPlanted,
			phaseRemaining: 39,
			bombRemaining:  floatPointer(39),
		},
		{
			name:           "defusing",
			state:          objective.Snapshot{State: objective.StateDefusing, Phase: objective.PhaseDefusing, IsPlantedNow: true, PlantTick: 640},
			phase:          ObjectiveClockDefusing,
			phaseRemaining: 39,
			bombRemaining:  floatPointer(39),
		},
		{
			name:           "resolved",
			state:          objective.Snapshot{State: objective.StateDefused, Phase: objective.PhaseResolved, WasPlantedThisRound: true},
			phase:          ObjectiveClockResolved,
			phaseRemaining: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := buildObjectiveClockSnapshot(test.state, currentTick, freezeTick, tickRate, 115, 40)
			if actual.Phase != test.phase || actual.PhaseTimeRemaining != test.phaseRemaining {
				t.Fatalf("unexpected phase clock: %+v", actual)
			}
			if test.roundRemaining == nil {
				if actual.RoundClockRemaining != nil {
					t.Fatalf("postplant/resolved phase exposed the round clock: %+v", actual)
				}
			} else if actual.RoundClockRemaining == nil || *actual.RoundClockRemaining != *test.roundRemaining {
				t.Fatalf("preplant round clock mismatch: %+v", actual)
			}
			if test.bombRemaining == nil {
				if actual.BombTimeRemaining != nil {
					t.Fatalf("non-postplant phase exposed a bomb clock: %+v", actual)
				}
			} else if actual.BombTimeRemaining == nil || *actual.BombTimeRemaining != *test.bombRemaining {
				t.Fatalf("postplant bomb clock mismatch: %+v", actual)
			}
		})
	}
}

func floatPointer(value float64) *float64 {
	return &value
}
