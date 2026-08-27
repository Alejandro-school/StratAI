package handlers

import (
	"testing"

	"cs2-demo-service/pkg/objective"
)

func TestObjectiveBombPositionPrefersTrackedBombOverEventActor(t *testing.T) {
	tracked := objective.ObservedPosition(10, 20, 30, objective.SourceDemoinfocsNativeSnapshot)
	actor := objective.ObservedPosition(900, 800, 700, objective.SourceDemoinfocsEvent)

	actual := objectiveBombPosition(objective.Snapshot{Position: tracked}, actor)

	if actual != tracked {
		t.Fatalf("explosion used actor position instead of tracked bomb: %+v", actual)
	}
}

func TestObjectiveBombPositionFallsBackWhenBombWasNotObserved(t *testing.T) {
	fallback := objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsEvent)

	actual := objectiveBombPosition(objective.Snapshot{}, fallback)

	if actual != fallback {
		t.Fatalf("unavailable bomb position discarded fallback: %+v", actual)
	}
}
