package handlers

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"cs2-demo-service/models"
	"cs2-demo-service/pkg/combat"
	"cs2-demo-service/pkg/objective"
	"cs2-demo-service/pkg/utility"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestProjectReplayUtilityMarkersUsesLedgerExactly(t *testing.T) {
	throws := []utility.Throw{{
		ID: "r2-u0001", Round: 2, Sequence: 1, Type: utility.TypeFlashbang,
		Actor: utility.PlayerRef{ID: 7, Status: utility.AvailabilityObserved},
		Lifecycle: utility.Lifecycle{
			Detonation: utility.TickPositionObservation{
				Tick: 220, Position: utility.Vector{X: 1, Y: 2, Z: 3},
				Status: utility.AvailabilityObserved, PositionStatus: utility.AvailabilityObserved,
				Source: utility.SourceFlashExplode,
			},
			Correlation: utility.Correlation{
				Status: utility.CorrelationObserved, Source: utility.SourceGrenadeEntityID,
			},
		},
		Flashes: []utility.FlashEffect{
			{Victim: utility.PlayerRef{ID: 9, Status: utility.AvailabilityObserved},
				Duration: utility.ScalarObservation{Value: 0.75, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed}},
			{Victim: utility.PlayerRef{ID: 8, Status: utility.AvailabilityObserved},
				Duration: utility.ScalarObservation{Value: 1.25, Status: utility.AvailabilityObserved, Source: utility.SourcePlayerFlashed}},
		},
		Damage: []utility.DamageEffect{{HealthDamage: 12}},
	}}
	raw := []models.ReplayEvent{
		{ID: "raw", Tick: 200, Type: "utility_detonate", DurationMS: 450},
		{ID: "kill", Tick: 230, Type: "kill"},
	}

	actual := projectReplayUtilityMarkers(raw, throws, 2)
	if len(actual) != 2 || actual[0].ID != "utility:r2-u0001" || actual[1].ID != "kill" {
		t.Fatalf("raw marker survived or ledger marker missing: %+v", actual)
	}
	marker := actual[0]
	if marker.Tick != 220 || marker.X != 1 || marker.Y != 2 || marker.Z != 3 ||
		marker.ActorID != 7 || marker.Damage != 12 || marker.DurationMS != 1250 ||
		marker.DurationStatus != string(utility.AvailabilityObserved) ||
		marker.DurationSource != utility.SourcePlayerFlashed ||
		marker.CorrelationStatus != string(utility.CorrelationObserved) ||
		!reflect.DeepEqual(marker.AffectedPlayerIDs, []string{"8", "9"}) {
		t.Fatalf("ledger marker lost exact utility facts: %+v", marker)
	}
}

func TestReplayUtilityMarkerPreservesObservedZeroPosition(t *testing.T) {
	throws := []utility.Throw{{
		ID: "r3-u0001", Round: 3, Type: utility.TypeHE,
		Lifecycle: utility.Lifecycle{
			Detonation: utility.TickPositionObservation{
				Tick: 300, Status: utility.AvailabilityObserved,
				PositionStatus: utility.AvailabilityObserved, Source: utility.SourceHEExplode,
			},
		},
	}}
	markers := projectReplayUtilityMarkers(nil, throws, 3)
	if len(markers) != 1 || markers[0].PositionStatus != string(utility.AvailabilityObserved) {
		t.Fatalf("observed origin marker missing: %+v", markers)
	}
	encoded, err := json.Marshal(markers[0])
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"x", "y", "z"} {
		if value, exists := raw[field]; !exists || value != float64(0) {
			t.Fatalf("observed zero coordinate %s was omitted: %s", field, encoded)
		}
	}
}

func TestProjectReplayUtilityFramesUsesObservedExpirationOnly(t *testing.T) {
	frames := []models.ReplayFrame{{
		Tick: 120,
		ActiveEffects: []models.ReplayActiveEffect{{
			Type: "smoke", X: 10, Y: 20, StartTick: 100,
		}},
	}}
	observed := utility.Throw{
		Round: 1, Sequence: 1, Type: utility.TypeSmoke,
		Lifecycle: utility.Lifecycle{
			EffectStart: utility.TickPositionObservation{
				Tick: 100, Position: utility.Vector{X: 10, Y: 20},
				Status: utility.AvailabilityObserved, PositionStatus: utility.AvailabilityObserved,
			},
			Expiration: utility.TickPositionObservation{Tick: 164, Status: utility.AvailabilityObserved},
		},
	}
	actual := projectReplayUtilityFrames(frames, []utility.Throw{observed}, 1, 64)
	effect := actual[0].ActiveEffects[0]
	if effect.TimeRemaining != 44.0/64.0 || effect.TimeRemainingStatus != string(utility.AvailabilityObserved) ||
		effect.TimeRemainingSource != utility.SourceCallbackTicks {
		t.Fatalf("observed expiry was not projected: %+v", effect)
	}

	observed.Lifecycle.Expiration = utility.TickPositionObservation{Status: utility.AvailabilityUnavailable}
	actual = projectReplayUtilityFrames(frames, []utility.Throw{observed}, 1, 64)
	effect = actual[0].ActiveEffects[0]
	if effect.TimeRemaining != 0 || effect.TimeRemainingStatus != string(utility.AvailabilityUnavailable) ||
		effect.TimeRemainingSource != utility.SourceUnavailable {
		t.Fatalf("missing expiry produced an estimated duration: %+v", effect)
	}
}

func TestReplayInfernoKeyRejectsNilCallback(t *testing.T) {
	if _, ok := replayInfernoKey(nil); ok {
		t.Fatal("nil InfernoExpired callback produced a runtime key")
	}
}

func TestRemoveDetonatedProjectileRemovesOnlyNearestMatchingGrenade(t *testing.T) {
	handler := &ReplayHandler{activeProjectiles: map[int64]*models.ReplayProjectile{
		1: {ID: 1, Type: "he", X: 100, Y: 100},
		2: {ID: 2, Type: "he", X: 900, Y: 900},
		3: {ID: 3, Type: "flashbang", X: 105, Y: 105},
	}}

	handler.removeDetonatedProjectile("he", 110, 110)

	if _, exists := handler.activeProjectiles[1]; exists {
		t.Fatal("detonated HE remained active")
	}
	if _, exists := handler.activeProjectiles[2]; !exists {
		t.Fatal("unrelated HE was removed")
	}
	if _, exists := handler.activeProjectiles[3]; !exists {
		t.Fatal("different grenade type was removed")
	}
}

func TestRemoveDetonatedProjectileBreaksDistanceTiesByStableSequence(t *testing.T) {
	handler := &ReplayHandler{activeProjectiles: map[int64]*models.ReplayProjectile{
		9001: {ID: 2, Type: "he", X: -10, Y: 0},
		1234: {ID: 1, Type: "he", X: 10, Y: 0},
	}}

	handler.removeDetonatedProjectile("he", 0, 0)

	if _, exists := handler.activeProjectiles[1234]; exists {
		t.Fatal("lower stable projectile sequence won the tie but was not removed")
	}
	if _, exists := handler.activeProjectiles[9001]; !exists {
		t.Fatal("higher stable projectile sequence was removed from an equal-distance tie")
	}
}

func TestReplayParticipantOrderIsIndependentOfInputPermutation(t *testing.T) {
	botTwo := &common.Player{SteamID64: 0, UserID: 2, EntityID: 12, Name: "bot-two"}
	humanThree := &common.Player{SteamID64: 3, UserID: 3, EntityID: 13, Name: "three"}
	botOne := &common.Player{SteamID64: 0, UserID: 1, EntityID: 11, Name: "bot-one"}
	humanTwo := &common.Player{SteamID64: 2, UserID: 4, EntityID: 14, Name: "two"}

	first := []*common.Player{humanThree, nil, botTwo, humanTwo, botOne}
	second := []*common.Player{botOne, humanTwo, botTwo, nil, humanThree}
	slices.SortFunc(first, compareReplayParticipants)
	slices.SortFunc(second, compareReplayParticipants)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("participant order depends on input permutation: first=%v second=%v", first, second)
	}
	if first[0] != botOne || first[1] != botTwo || first[2] != humanTwo || first[3] != humanThree || first[4] != nil {
		t.Fatalf("unexpected participant order: %v", first)
	}
}

func TestReplayProjectileIDsAndOrderIgnoreNativeRandomIDs(t *testing.T) {
	handler := &ReplayHandler{activeProjectiles: make(map[int64]*models.ReplayProjectile)}
	handler.activeProjectiles[9189867185372267349] = &models.ReplayProjectile{
		ID:   handler.nextReplayProjectileID(),
		Type: "he",
	}
	handler.activeProjectiles[3947307330693761937] = &models.ReplayProjectile{
		ID:   handler.nextReplayProjectileID(),
		Type: "smoke",
	}

	projectiles := []models.ReplayProjectile{
		*handler.activeProjectiles[3947307330693761937],
		*handler.activeProjectiles[9189867185372267349],
	}
	sortReplayProjectiles(projectiles)

	if projectiles[0].ID != 1 || projectiles[1].ID != 2 {
		t.Fatalf("projectile order leaked native random IDs: %+v", projectiles)
	}
}

func TestReplayActiveEffectOrderIsIndependentOfInputPermutation(t *testing.T) {
	firstEffect := models.ReplayActiveEffect{
		Type: "inferno", X: 10, Y: 20, StartTick: 100, Hull: []float64{1, 2}, Points: []float64{3, 4},
	}
	secondEffect := models.ReplayActiveEffect{
		Type: "inferno", X: 10, Y: 20, StartTick: 100, Hull: []float64{1, 2}, Points: []float64{3, 5},
	}
	thirdEffect := models.ReplayActiveEffect{Type: "smoke", X: 5, Y: 6, StartTick: 90, Radius: 144}
	first := []models.ReplayActiveEffect{thirdEffect, secondEffect, firstEffect}
	second := []models.ReplayActiveEffect{firstEffect, thirdEffect, secondEffect}
	slices.SortFunc(first, compareReplayActiveEffects)
	slices.SortFunc(second, compareReplayActiveEffects)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("active-effect order depends on input permutation: first=%+v second=%+v", first, second)
	}
}

func TestProjectReplayObjectiveMarkersDropsRawExtrasAndUsesLedger(t *testing.T) {
	rawEvents := []models.ReplayEvent{
		{ID: "raw-postresolved", Tick: 300, Type: "bomb_explode", Site: "B"},
		{ID: "kill-2", Tick: 200, Type: "kill"},
		{ID: "hurt-1", Tick: 100, Type: "player_hurt"},
		{ID: "raw-accepted", Tick: 200, Type: "bomb_plant", Site: "B", PlayerID: 9},
	}
	ledger := []objective.Event{
		{
			ID:       "objective-plant",
			Type:     objective.EventPlant,
			Round:    4,
			Tick:     200,
			Actor:    objective.Actor{SteamID: 42, Name: "planter", Side: "T"},
			Site:     "A",
			Position: objective.ObservedPosition(10, 20, 30, objective.SourceDemoinfocsEvent),
		},
		{ID: "objective-start", Type: objective.EventPlantStart, Round: 4, Tick: 150},
		{ID: "other-round", Type: objective.EventExplode, Round: 5, Tick: 300, Site: "B"},
	}

	actual := projectReplayObjectiveMarkers(rawEvents, ledger, 4)

	if len(actual) != 3 {
		t.Fatalf("expected two non-objective events and one ledger marker, got %+v", actual)
	}
	if actual[0].ID != "hurt-1" || actual[1].ID != "objective-plant" || actual[2].ID != "kill-2" {
		t.Fatalf("events are not totally ordered by tick, type and ID: %+v", actual)
	}
	marker := actual[1]
	if marker.Type != "bomb_plant" || marker.PlayerID != 42 || marker.Site != "A" ||
		marker.X != 10 || marker.Y != 20 || marker.Z != 30 {
		t.Fatalf("replay marker was not projected from the ledger: %+v", marker)
	}
	for _, event := range actual {
		if event.ID == "raw-postresolved" || event.ID == "raw-accepted" {
			t.Fatalf("raw objective marker survived ledger projection: %+v", actual)
		}
	}
}

func TestReplayObjectiveMarkerProjectsSupportedTerminalEvents(t *testing.T) {
	tests := []struct {
		name       string
		typeValue  objective.EventType
		replayType string
		durationMS int
	}{
		{name: "plant", typeValue: objective.EventPlant, replayType: "bomb_plant"},
		{name: "defuse", typeValue: objective.EventDefuse, replayType: "bomb_defuse"},
		{name: "explode", typeValue: objective.EventExplode, replayType: "bomb_explode", durationMS: 1200},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := objective.Event{
				ID:       "objective-" + test.name,
				Type:     test.typeValue,
				Round:    7,
				Tick:     123,
				Actor:    objective.Actor{SteamID: 88},
				Site:     "B",
				Position: objective.ObservedPosition(1, 2, 3, objective.SourceDemoinfocsEvent),
			}

			marker, ok := replayObjectiveMarker(source)
			if !ok {
				t.Fatal("supported objective event was not projected")
			}
			if marker.ID != source.ID || marker.Tick != source.Tick || marker.Type != test.replayType ||
				marker.PlayerID != 88 || marker.Site != "B" || marker.X != 1 || marker.Y != 2 || marker.Z != 3 ||
				marker.DurationMS != test.durationMS {
				t.Fatalf("incorrect objective projection: %+v", marker)
			}
		})
	}

	if _, ok := replayObjectiveMarker(objective.Event{Type: objective.EventPlantStart}); ok {
		t.Fatal("non-terminal objective event produced a replay marker")
	}
}

func TestReplayBombStateProjectsCurrentAndHistoricalFacts(t *testing.T) {
	snapshot := objective.Snapshot{
		State:               objective.StateDefused,
		Phase:               objective.PhaseResolved,
		IsPlantedNow:        false,
		WasPlantedThisRound: true,
		Site:                "B",
		PlantTick:           5304,
		Defuser:             objective.Actor{SteamID: 42},
		Position:            objective.ObservedPosition(1208, -340, 12, objective.SourceDemoinfocsEvent),
	}
	actual := replayBombState(snapshot)
	if actual.State != objective.StateDefused || actual.IsPlantedNow || !actual.WasPlantedThisRound {
		t.Fatalf("replay bomb state lost current/history semantics: %+v", actual)
	}
	if actual.DefuserID == nil || *actual.DefuserID != 42 || actual.CarrierID != nil ||
		actual.X != 1208 || actual.PositionStatus != objective.PositionObserved {
		t.Fatalf("replay bomb projection mismatch: %+v", actual)
	}
	encoded, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal replay bomb state: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decode replay bomb state: %v", err)
	}
	if carrier, exists := raw["carrier_id"]; !exists || carrier != nil {
		t.Fatalf("missing carrier must be explicit null: %s", encoded)
	}
}

func TestReplayPlayerHasC4UsesOnlyLiveNativeCarrier(t *testing.T) {
	snapshot := objective.Snapshot{Carrier: objective.Actor{SteamID: 20}}
	if replayPlayerHasC4(10, true, snapshot) {
		t.Fatal("previous carrier retained stale C4")
	}
	if !replayPlayerHasC4(20, true, snapshot) {
		t.Fatal("current native carrier lost C4")
	}
	if replayPlayerHasC4(20, false, snapshot) {
		t.Fatal("dead player retained C4")
	}
}

func TestMergePlayerEquipmentClearsDeadPlayerObjectiveItems(t *testing.T) {
	cached := replayPlayerEquipment{
		weapons:      []string{"AK-47", "C4"},
		activeWeapon: "AK-47",
		hasC4:        true,
		hasDefuseKit: true,
	}
	actual := mergePlayerEquipment(cached, false, nil, "", false, false, true)
	if actual.hasC4 || actual.hasDefuseKit {
		t.Fatalf("dead player retained cached objective equipment: %+v", actual)
	}
	if actual.activeWeapon != "AK-47" || len(actual.weapons) != 2 {
		t.Fatalf("C4 fix damaged the intentional loadout cache: %+v", actual)
	}
}

func TestFinalizePersistsLastValidRoundOnce(t *testing.T) {
	handler := &ReplayHandler{
		currentRound: &models.ReplayRoundData{
			Round:     23,
			StartTick: 100,
			Frames:    []models.ReplayFrame{{Tick: 140}},
		},
		Rounds: []models.ReplayRound{},
	}

	handler.Finalize()
	handler.Finalize()

	if len(handler.Rounds) != 1 || handler.Rounds[0].EndTick != 140 {
		t.Fatalf("last round was not finalized exactly once: %+v", handler.Rounds)
	}
}

func TestFinalizeReprojectsCombatAfterUnorderedRoundEndCallbacks(t *testing.T) {
	tracker := combat.NewTracker()
	tracker.RecordWeaponFire(combat.FireInput{
		Round: 1, Tick: 100,
		Actor:  combat.PlayerRef{ID: 7, Status: combat.AvailabilityObserved, Source: combat.SourceCallbackPlayer},
		Weapon: combat.WeaponRef{Name: "AK-47", Status: combat.AvailabilityObserved, Source: combat.SourceCallbackWeapon},
	})
	openFire := tracker.Snapshot()[0]
	handler := &ReplayHandler{
		ctx: &models.DemoContext{Combat: tracker},
		Rounds: []models.ReplayRound{{
			Round: 1,
			Frames: projectReplayCombatFrames(
				[]models.ReplayFrame{{Tick: 100}}, []combat.Event{openFire}, 1,
			),
		}},
	}
	if got := handler.Rounds[0].Frames[0].Shots[0].Result; got != string(combat.ShotUnavailable) {
		t.Fatalf("pre-closure replay result = %q", got)
	}

	tracker.EndRound(1, 120)
	handler.Finalize()
	shot := handler.Rounds[0].Frames[0].Shots[0]
	if shot.Result != string(combat.ShotMiss) || shot.ResultStatus != string(combat.AvailabilityDerived) {
		t.Fatalf("final replay retained callback-order state: %+v", shot)
	}
}

func TestReplayEventIDsRemainUniqueWithinSameTick(t *testing.T) {
	handler := &ReplayHandler{}
	first := handler.replayEventID(1, 100, "hurt", 7)
	second := handler.replayEventID(1, 100, "hurt", 7)
	if first == second {
		t.Fatalf("event IDs collided: %s", first)
	}
}

func TestReplayRoundNumberDoesNotDependOnSharedHandlerState(t *testing.T) {
	for totalRoundsPlayed := 0; totalRoundsPlayed < 22; totalRoundsPlayed++ {
		expected := totalRoundsPlayed + 1
		if actual := replayRoundNumber(totalRoundsPlayed); actual != expected {
			t.Fatalf("total rounds %d produced replay round %d; expected %d", totalRoundsPlayed, actual, expected)
		}
	}
}
