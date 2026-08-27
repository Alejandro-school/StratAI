package handlers

import (
	"encoding/json"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/playerstate"

	"github.com/golang/geo/r3"
)

func TestVelocityObservationSeparatesUnknownFromStationary(t *testing.T) {
	unknown := velocityObservationFromEstimate(playerstate.MotionEstimate{
		Source: playerstate.VelocitySourceInsufficientHistory,
	}, models.VelocityObservationUnavailable)
	if unknown.Available || unknown.Horizontal != nil || unknown.X != nil || unknown.MeasurementTicks != nil {
		t.Fatalf("unknown velocity has measured values: %+v", unknown)
	}
	if engagement := getEngagementType(unknown.Horizontal, unknown.Available); engagement != "" {
		t.Fatalf("unknown velocity classified as %q", engagement)
	}

	stationary := velocityObservationFromEstimate(playerstate.MotionEstimate{
		Vector:        r3.Vector{},
		Source:        playerstate.VelocitySourcePositionDelta,
		IntervalTicks: 1,
		ObservedTick:  42,
		Available:     true,
	}, models.VelocityObservationCurrentTick)
	if !stationary.Available || stationary.Horizontal == nil || *stationary.Horizontal != 0 {
		t.Fatalf("stationary velocity was not preserved as an observed zero: %+v", stationary)
	}
	if stationary.ObservedTick == nil || *stationary.ObservedTick != 42 {
		t.Fatalf("stationary observation lost its tick: %+v", stationary)
	}
	if engagement := getEngagementType(stationary.Horizontal, stationary.Available); engagement != "hold" {
		t.Fatalf("stationary velocity classified as %q", engagement)
	}
}

func TestLastAliveVelocityKeepsHistoricalTick(t *testing.T) {
	lastAlive := velocityObservationFromEstimate(playerstate.MotionEstimate{
		Vector:        r3.Vector{X: 128},
		Source:        playerstate.VelocitySourcePositionDelta,
		IntervalTicks: 1,
		ObservedTick:  99,
		Available:     true,
	}, models.VelocityObservationLastAlive)

	if lastAlive.Source != string(playerstate.VelocitySourcePositionDelta) ||
		lastAlive.Observation != models.VelocityObservationLastAlive ||
		lastAlive.ObservedTick == nil || *lastAlive.ObservedTick != 99 {
		t.Fatalf("last-alive provenance was lost: %+v", lastAlive)
	}
}

func TestLegacySnapshotsEncodeUnknownAsNullAndStationaryAsZero(t *testing.T) {
	zero := 0.0
	unknown := mustDecodeJSON(t, models.MovementLog{VelocityObservation: models.VelocityObservationUnavailable})
	stationary := mustDecodeJSON(t, models.MovementLog{
		Speed:               &zero,
		VelocityAvailable:   true,
		VelocityObservation: models.VelocityObservationCurrentTick,
	})
	if unknown["speed"] != nil {
		t.Fatalf("unknown movement speed encoded as %v", unknown["speed"])
	}
	if unknown["velocity_observation"] != models.VelocityObservationUnavailable ||
		stationary["speed"] != float64(0) || stationary["velocity_available"] != true ||
		stationary["velocity_observation"] != models.VelocityObservationCurrentTick {
		t.Fatalf("stationary movement speed encoded incorrectly: %+v", stationary)
	}

	unknownState := mustDecodeJSON(t, models.PlayerStateSnapshot{VelocityObservation: models.VelocityObservationUnavailable})
	stationaryState := mustDecodeJSON(t, models.PlayerStateSnapshot{
		VelocityX:           &zero,
		VelocityY:           &zero,
		VelocityZ:           &zero,
		VelocityAvailable:   true,
		VelocityObservation: models.VelocityObservationCurrentTick,
	})
	if unknownState["velocity_x"] != nil {
		t.Fatalf("unknown timeline velocity encoded as %v", unknownState["velocity_x"])
	}
	if stationaryState["velocity_x"] != float64(0) || stationaryState["velocity_available"] != true ||
		stationaryState["velocity_observation"] != models.VelocityObservationCurrentTick {
		t.Fatalf("stationary timeline velocity encoded incorrectly: %+v", stationaryState)
	}

	unknownDuelPlayer := mustDecodeJSON(t, models.AI_DuelParticipant{VelocityObservation: models.VelocityObservationUnavailable})
	stationaryDuelPlayer := mustDecodeJSON(t, models.AI_DuelParticipant{
		Velocity:            &zero,
		VelocityAvailable:   true,
		VelocityObservation: models.VelocityObservationCurrentTick,
	})
	if unknownDuelPlayer["velocity"] != nil {
		t.Fatalf("unknown duel velocity encoded as %v", unknownDuelPlayer["velocity"])
	}
	if stationaryDuelPlayer["velocity"] != float64(0) || stationaryDuelPlayer["velocity_available"] != true ||
		stationaryDuelPlayer["velocity_observation"] != models.VelocityObservationCurrentTick {
		t.Fatalf("stationary duel velocity encoded incorrectly: %+v", stationaryDuelPlayer)
	}
}

func TestRawCombatVelocityAvailabilitySurvivesAggregation(t *testing.T) {
	zero := 0.0
	window, observedTick := 1, 64
	unknown := aggregateAttackerStats([]models.RawCombatEvent{{
		AttackerSteamID:             7,
		AttackerVelocitySource:      string(playerstate.VelocitySourceInsufficientHistory),
		AttackerVelocityObservation: models.VelocityObservationUnavailable,
	}}, 7)
	if unknown.Velocity != nil || unknown.VelocityAvailable {
		t.Fatalf("unknown combat velocity became measured: %+v", unknown)
	}

	stationary := aggregateAttackerStats([]models.RawCombatEvent{{
		AttackerSteamID:                  7,
		AttackerVelocity:                 &zero,
		AttackerVelocityAvailable:        true,
		AttackerVelocitySource:           string(playerstate.VelocitySourcePositionDelta),
		AttackerVelocityObservation:      models.VelocityObservationCurrentTick,
		AttackerVelocityMeasurementTicks: &window,
		AttackerVelocityObservedTick:     &observedTick,
	}}, 7)
	if !stationary.VelocityAvailable || stationary.Velocity == nil || *stationary.Velocity != 0 {
		t.Fatalf("stationary combat velocity was lost: %+v", stationary)
	}
	if stationary.VelocityObservedTick == nil || *stationary.VelocityObservedTick != observedTick {
		t.Fatalf("combat velocity tick was lost: %+v", stationary)
	}
	if stationary.VelocitySource != string(playerstate.VelocitySourcePositionDelta) ||
		stationary.VelocityObservation != models.VelocityObservationCurrentTick {
		t.Fatalf("combat velocity provenance was lost: %+v", stationary)
	}
}

func mustDecodeJSON(t *testing.T, value any) map[string]any {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	decoded := make(map[string]any)
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}
