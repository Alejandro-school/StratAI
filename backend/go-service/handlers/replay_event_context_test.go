package handlers

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"github.com/golang/geo/r3"
)

func TestReplayEventContextRequiresOptInRoundAndExplicitSource(t *testing.T) {
	for _, mode := range []string{"disabled", "missing_round", "missing_source", "before_round_start"} {
		t.Run(mode, func(t *testing.T) {
			ctx := &models.DemoContext{CaptureCalculatedVision: mode != "disabled"}
			handler := NewReplayHandler(ctx)
			if mode != "missing_round" {
				handler.currentRound = &models.ReplayRoundData{Round: 6, StartTick: 100}
			}
			event := models.ReplayContextEvent{Type: "weapon_fire", Tick: 110, Source: "demoinfocs_event:WeaponFire"}
			if mode == "missing_source" {
				event.Source = ""
			}
			if mode == "before_round_start" {
				event.Tick = 99
			}
			handler.recordReplayContextEvent(event)
			if handler.currentRound != nil && handler.currentRound.EventContext != nil && len(handler.currentRound.EventContext.Events) != 0 {
				t.Fatalf("unavailable or out-of-scope event was recorded: %+v", handler.currentRound.EventContext)
			}
			if ctx.Combat != nil || ctx.Objectives != nil || ctx.Utilities != nil || ctx.SupportObservations != nil {
				t.Fatal("replay-only event recording enabled an unrelated canonical observer")
			}
		})
	}
}

func TestReplayEventContextPreservesSameTickOrderAvailabilityAndOpaqueIDs(t *testing.T) {
	ctx := &models.DemoContext{CaptureCalculatedVision: true}
	handler := NewReplayHandler(ctx)
	legacy := []models.ReplayEvent{{ID: "retained-original-event", Tick: 100, Type: "retained-original-fact"}}
	handler.currentRound = &models.ReplayRoundData{Round: 6, StartTick: 100, Events: append([]models.ReplayEvent(nil), legacy...)}
	events := []models.ReplayContextEvent{
		{Type: "weapon_fire", Tick: 100, AvailabilityTick: 999, ActorID: 76561198000000077, ActorTeam: "T", Weapon: "AK-47", Source: "demoinfocs_event:WeaponFire"},
		{Type: "player_hurt", Tick: 100, AvailabilityTick: 999, ActorID: 76561198000000077, ActorTeam: "T", VictimID: 76561198000000088, VictimTeam: "CT", Weapon: "AK-47", Damage: 19, Health: 81, Source: "demoinfocs_event:PlayerHurt"},
		{Type: "weapon_fire", Tick: 100, AvailabilityTick: 999, ActorID: 76561198000000077, ActorTeam: "T", Weapon: "AK-47", Source: "demoinfocs_event:WeaponFire"},
	}
	for _, event := range events {
		handler.recordReplayContextEvent(event)
	}
	context := handler.currentRound.EventContext
	if context == nil || context.SchemaID != "stratai.replay_events@1" || len(context.Events) != len(events) {
		t.Fatalf("observed events were lost or lack their versioned contract: %+v", context)
	}
	seen := make(map[string]bool)
	for i, got := range context.Events {
		if got.ID == "" || seen[got.ID] || strings.Contains(got.ID, "76561198000000077") || strings.Contains(got.ID, "76561198000000088") {
			t.Errorf("event ID is missing, duplicated or exposes participant identity: %q", got.ID)
		}
		seen[got.ID] = true
		want := events[i]
		want.ID, want.AvailabilityTick = got.ID, want.Tick
		if !reflect.DeepEqual(got, want) {
			t.Errorf("callback order, event detail or causal timestamp changed at index %d: got %+v, want %+v", i, got, want)
		}
	}
	if !reflect.DeepEqual(handler.currentRound.Events, legacy) {
		t.Fatalf("context capture replaced original replay events: %+v", handler.currentRound.Events)
	}
	if ctx.Combat != nil || ctx.Objectives != nil || ctx.Utilities != nil || ctx.SupportObservations != nil {
		t.Fatal("context capture created unrelated canonical observer state")
	}
}

func TestReplayEventContextKeepsWorldDamageWithoutInventingAttacker(t *testing.T) {
	handler := NewReplayHandler(&models.DemoContext{CaptureCalculatedVision: true})
	handler.currentRound = &models.ReplayRoundData{Round: 6, StartTick: 100}
	handler.recordReplayContextEvent(models.ReplayContextEvent{
		Type: "player_hurt", Tick: 110, VictimID: 76561198000000088, VictimTeam: "CT",
		Damage: 12, Health: 88, Source: "demoinfocs_event:PlayerHurt",
	})
	context := handler.currentRound.EventContext
	if context == nil || len(context.Events) != 1 {
		t.Fatalf("damage with no attacker was discarded: %+v", context)
	}
	got := context.Events[0]
	if got.ActorID != 0 || got.ActorTeam != "" || got.VictimID != 76561198000000088 || got.VictimTeam != "CT" || got.Damage != 12 || got.Health != 88 {
		t.Fatalf("world damage facts were changed or an attacker was invented: %+v", got)
	}
}

func TestReplayEventContextWireFormatPreservesIdentitiesAndObservedCoordinateZero(t *testing.T) {
	handler := NewReplayHandler(&models.DemoContext{CaptureCalculatedVision: true})
	handler.currentRound = &models.ReplayRoundData{Round: 6, StartTick: 100}
	x, y, z := 0.0, 40.0, 0.0
	handler.recordReplayContextEvent(models.ReplayContextEvent{
		Type: "utility_throw", Tick: 110, ActorID: 76561198000000077, ActorTeam: "T", VictimID: 76561198000000088,
		ProjectileID: "context-projectile-6-1", UtilityType: "smoke", X: &x, Y: &y, Z: &z, Source: "demoinfocs_event:GrenadeProjectileThrow",
	})
	handler.recordReplayContextEvent(models.ReplayContextEvent{Type: "weapon_fire", Tick: 111, Source: "demoinfocs_event:WeaponFire"})
	encoded, err := json.Marshal(handler.currentRound.EventContext)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	got := payload.Events[0]
	if got["actor_id"] != "76561198000000077" || got["victim_id"] != "76561198000000088" || got["projectile_id"] != "context-projectile-6-1" {
		t.Fatalf("wire format loses 64-bit identities or the exact projectile link: %v", got)
	}
	if got["x"] != float64(0) || got["y"] != float64(40) || got["z"] != float64(0) {
		t.Fatalf("observed zero coordinates were lost: %v", got)
	}
	for _, axis := range []string{"x", "y", "z"} {
		if payload.Events[1][axis] != nil {
			t.Errorf("absent coordinate became an observed value: %s=%v", axis, payload.Events[1][axis])
		}
	}
}

func TestReplayEventContextSurvivesRoundFinalizationWithoutReplacingOriginalFacts(t *testing.T) {
	handler := NewReplayHandler(&models.DemoContext{CaptureCalculatedVision: true})
	handler.currentRound = &models.ReplayRoundData{
		Round: 6, StartTick: 100,
		Frames: []models.ReplayFrame{{Tick: 140, TimeRemaining: 71, RoundContext: &models.ReplayRoundContext{Round: 6, Tick: 140, Phase: "preplant"}}},
		Events: []models.ReplayEvent{{ID: "original-fact", Tick: 120, Type: "retained-original-fact"}},
	}
	handler.recordReplayContextEvent(models.ReplayContextEvent{Type: "weapon_fire", Tick: 120, Weapon: "AK-47", Source: "demoinfocs_event:WeaponFire"})
	handler.finishCurrentRound(150)
	handler.Finalize()
	handler.Finalize()
	if len(handler.Rounds) != 1 {
		t.Fatalf("finalization duplicated or discarded the round: %+v", handler.Rounds)
	}
	round := handler.Rounds[0]
	if round.EventContext == nil || len(round.EventContext.Events) != 1 || round.EventContext.Events[0].Type != "weapon_fire" {
		t.Fatalf("versioned event context was lost during final round projection: %+v", round.EventContext)
	}
	if len(round.Events) != 1 || round.Events[0].ID != "original-fact" || len(round.Frames) != 1 ||
		round.Frames[0].TimeRemaining != 71 || round.Frames[0].RoundContext == nil || round.Frames[0].RoundContext.Phase != "preplant" {
		t.Fatalf("event context extension damaged original events or round frames: %+v", round)
	}
	encoded, err := json.Marshal(round)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if _, exists := wire["event_context"]; !exists {
		t.Fatal("final round wire contract omitted event_context")
	}

	handler.currentRound = &models.ReplayRoundData{Round: 7, StartTick: 200}
	handler.recordReplayContextEvent(models.ReplayContextEvent{Type: "player_hurt", Tick: 210, Damage: 8, Source: "demoinfocs_event:PlayerHurt"})
	handler.finishCurrentRound(250)
	if handler.Rounds[0].EventContext == handler.Rounds[1].EventContext || len(handler.Rounds[0].EventContext.Events) != 1 {
		t.Fatal("next-round context mutated an already completed round")
	}
}

func TestReplayEventContextInvalidEffectPositionNeverReusesLaunchCoordinates(t *testing.T) {
	for _, invalid := range []r3.Vector{{X: math.NaN(), Y: 200, Z: 300}, {X: 100, Y: math.Inf(1), Z: 300}, {X: 100, Y: 200, Z: math.Inf(-1)}} {
		x, y, z := 10.0, 20.0, 30.0
		effect := models.ReplayContextEvent{Type: "utility_start", X: &x, Y: &y, Z: &z}
		replayEventPosition(&effect, invalid)
		if effect.X != nil || effect.Y != nil || effect.Z != nil {
			t.Fatalf("unavailable effect position retained the launch location: %+v", effect)
		}
	}
}

func TestReplayEventContextLinksTheNativeProjectileWhenEquipmentHasTwoThrows(t *testing.T) {
	first := models.ReplayContextEvent{Type: "grenade_throw", Tick: 110, ProjectileID: "41", UtilityType: "flashbang", Source: "demoinfocs_event:GrenadeProjectileThrow"}
	second := models.ReplayContextEvent{Type: "grenade_throw", Tick: 111, ProjectileID: "42", UtilityType: "flashbang", Source: "demoinfocs_event:GrenadeProjectileThrow"}
	handler := NewReplayHandler(&models.DemoContext{CaptureCalculatedVision: true})
	handler.currentRound = &models.ReplayRoundData{Round: 6, StartTick: 100}
	// Two grenades from the same inventory instance do not have two distinct
	// equipment identities. The native projectile entity disambiguates them.
	handler.replayEventThrows = map[int64][]models.ReplayContextEvent{5001: {first, second}}
	handler.replayEventProjectileEntities = map[int][]models.ReplayContextEvent{17: {first}, 18: {second}}
	for nativeEntity, expected := range map[int]models.ReplayContextEvent{17: first, 18: second} {
		got, ok := handler.replayUtilityLaunch(nativeEntity, 5001)
		if !ok || !reflect.DeepEqual(got, expected) {
			t.Errorf("native projectile %d was linked to the wrong launch: got %+v, available=%v", nativeEntity, got, ok)
		}
	}
	if got, ok := handler.replayUtilityLaunch(0, 5001); ok {
		t.Fatalf("missing projectile identity silently selected one of two equipment launches: %+v", got)
	}
}

func TestReplayEventContextDoesNotResolveUnknownConflictingOrReusedNativeIdentity(t *testing.T) {
	first := models.ReplayContextEvent{Type: "grenade_throw", Tick: 110, ProjectileID: "41", Source: "demoinfocs_event:GrenadeProjectileThrow"}
	second := models.ReplayContextEvent{Type: "grenade_throw", Tick: 111, ProjectileID: "42", Source: "demoinfocs_event:GrenadeProjectileThrow"}
	handler := NewReplayHandler(&models.DemoContext{CaptureCalculatedVision: true})
	handler.currentRound = &models.ReplayRoundData{Round: 6, StartTick: 100}
	handler.replayEventThrows = map[int64][]models.ReplayContextEvent{5001: {first}, 5002: {second}, 5003: {first, second}}
	handler.replayEventProjectileEntities = map[int][]models.ReplayContextEvent{17: {first}, 18: {second}, 19: {first, second}}
	for _, ids := range []struct {
		entity    int
		equipment int64
	}{
		{17, 5002}, // Positive identities conflict; equipment cannot override it.
		{99, 5001}, // A provided, unknown projectile cannot use an equipment fallback.
		{19, 5003}, // Native entity reuse has more than one causal candidate.
		{0, 0},     // No identity source exists.
	} {
		if got, ok := handler.replayUtilityLaunch(ids.entity, ids.equipment); ok {
			t.Errorf("unaccredited native identity produced a launch (entity=%d equipment=%d): %+v", ids.entity, ids.equipment, got)
		}
	}
	if got, ok := handler.replayUtilityLaunch(0, 5001); !ok || got.ProjectileID != "41" {
		t.Errorf("an absent projectile ID should still retain an unambiguous exact equipment witness: %+v available=%v", got, ok)
	}
	// A positively observed projectile entity with exactly one launch is itself
	// an identity witness. Missing equipment is different from conflicting it.
	if got, ok := handler.replayUtilityLaunch(17, 0); !ok || got.ProjectileID != "41" {
		t.Errorf("unique native projectile identity was lost merely because equipment is unavailable: %+v available=%v", got, ok)
	}
}
