package handlers

import (
	"cs2-demo-service/models"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"testing"
)

func TestUtilityVisionLifeRejectsSameTickDeathWithoutPreviousRoundContamination(t *testing.T) {
	round := &models.ReplayRoundData{StartTick: 100, EndTick: 200,
		Events: []models.ReplayEvent{{Type: "kill", Tick: 80, VictimID: 11}, {Type: "kill", Tick: 150, VictimID: 12}},
		UtilityVision: &models.ReplayUtilityVision{Abstentions: map[string]int{}, Observations: []models.ReplayUtilityObservation{
			{ID: "old-round-survivor", ObserverID: 11, Tick: 120},
			{ID: "before-death", ObserverID: 12, Tick: 149},
			{ID: "same-tick-death", ObserverID: 12, Tick: 150},
			{ID: "after-death", ObserverID: 12, Tick: 151},
		}},
	}
	reconcileReplayUtilityVisionLife(round)
	if len(round.UtilityVision.Observations) != 2 || round.UtilityVision.Observations[0].ID != "old-round-survivor" || round.UtilityVision.Observations[1].ID != "before-death" {
		t.Fatalf("wrong causal life prefix: %+v", round.UtilityVision.Observations)
	}
	if round.UtilityVision.Abstentions["observer_death_at_or_before_observation_tick"] != 2 {
		t.Fatal("missing life abstentions")
	}
}

func TestUtilityVisionUnknownSmokeCannotBecomeClearSight(t *testing.T) {
	h := &ReplayHandler{ctx: &models.DemoContext{CaptureUtilityVision: true}, currentRound: &models.ReplayRoundData{StartTick: 100}}
	h.startReplayUtilityVisualSmoke(events.SmokeStart{})
	if !h.utilitySmokeOcclusionUnavailable {
		t.Fatal("unlinked cloud lost its occlusion uncertainty")
	}
	h.observeReplayUtilityPoint("projectile", "projectile:1", "he", r3.Vector{}, "demoinfocs_state:GrenadeProjectiles", "native_projectile_position", "")
	if len(h.currentRound.UtilityVision.Observations) != 0 || h.currentRound.UtilityVision.Abstentions["smoke_occlusion_source_unavailable"] != 1 {
		t.Fatal("unknown smoke permitted new visual estimates")
	}
}
