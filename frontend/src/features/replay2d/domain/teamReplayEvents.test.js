import { describe, expect, it } from "vitest";
import { createTeamReplayProjector } from "./teamReplayProjection";

const source = { map_name: "de_mirage", tick_rate: 64 };
const player = (steam_id, team, x) => ({ steam_id, team, alive: true, x, y: 20, z: 0, health: 100 });
const roundContext = (tick) => ({
  schema_id: "stratai.replay_round_context@1", round: 1, tick, availability_tick: tick,
  phase: "preplant", freeze_end_tick: 100, elapsed_seconds: (tick - 100) / 64,
  end_tick: null, winner: "", configured_duration_seconds: 115,
  pause_state: "unknown", clock_status: "unavailable", clock_reason: "native_pause_controls_incomplete",
  round_remaining_seconds: null, bomb_remaining_seconds: null,
});
const action = (id, tick, extra = {}) => ({
  id, type: "weapon_fire", tick, availability_tick: tick,
  actor_id: "1", actor_team: "T", weapon: "Glock-18", x: 10, y: 20,
  source: "demoinfocs_event:WeaponFire", ...extra,
});
const fixture = () => ({
  round: 1, start_tick: 100, end_tick: 116, winner: "CT",
  frames: [100, 104, 108].map((tick) => ({
    tick, time_remaining: 115, players: [player("1", "T", 10), player("9", "T", 90), player("8", "CT", 80)],
    round_context: roundContext(tick),
    bomb: { state: "carried", objective_phase: "preplant", is_planted_now: false,
      was_planted_this_round: false, carrier_id: "1", position_status: "observed",
      position_source: "demoinfocs_native_snapshot", x: 10, y: 20, z: 0 },
  })),
  events: [],
  event_context: { schema_id: "stratai.replay_events@1", events: [action("own-fire", 102),
    action("enemy-fire", 103, { actor_id: "8", actor_team: "CT", weapon: "USP-S", x: 987654 })] },
  calculated_vision: {
    schema_id: "stratai.calculated_vision@2", producer: "go.reaction.world_occlusion_v2", sample_stride_ticks: 4,
    samples: [{ tick: 100, availability_tick: 100, status: "calculated", contacts: [
      { observer_id: "9", observer_side: "T", subject_id: "8", subject_side: "CT", x: 80, y: 20, z: 0 },
    ] }, { tick: 104, availability_tick: 104, status: "calculated", contacts: [] }],
  },
});

describe("events integrated with the versioned causal team replay", () => {
  it("retains an inferno expiry with no native position without inventing position provenance", () => {
    const round = fixture();
    for (const frame of round.frames) frame.round_context.schema_id = "stratai.replay_round_context@2";
    const expiry = action("expiry", 103, { type: "utility_end", utility_type: "inferno", effect_id: "fire-1" });
    delete expiry.x; delete expiry.y;
    round.event_context.events = [expiry];
    const event = createTeamReplayProjector(round, "T", null, source)(104).events[0];
    expect(event).toMatchObject({ type: "utility_end", tick: 103, effect_id: "fire-1" });
    for (const field of ["x", "y", "z", "position_tick"]) expect(event).not.toHaveProperty(field);
  });

  it.each([true, false])("dates the own victim position independently of attacker contact (%s)", (visible) => {
    const round = fixture();
    for (const frame of round.frames) frame.round_context.schema_id = "stratai.replay_round_context@2";
    if (!visible) round.calculated_vision.samples = [];
    round.event_context.events = [action("hurt", 103, { type: "player_hurt",
      actor_id: "8", actor_team: "CT", victim_id: "9", victim_team: "T",
      damage: 15, health: 85, x: 91, y: 21, z: 1 })];
    const projected = createTeamReplayProjector(round, "T", null, source)(104);
    expect(projected.events[0]).toMatchObject({ type: "player_hurt", victim_id: "9",
      x: 91, y: 21, z: 1, position_tick: 103 });
    if (visible) expect(projected.events[0].observation.tick).toBe(100);
    else expect(projected.events[0]).not.toHaveProperty("actor_id");
    // Earlier schemas keep their original projection and receipts.
    for (const frame of round.frames) frame.round_context.schema_id = "stratai.replay_round_context@1";
    expect(createTeamReplayProjector(round, "T", null, source)(104).events[0]).not.toHaveProperty("position_tick");
  });

  it("combines objective, round state, and dated actions without restoring unknown clocks or a future winner", () => {
    const round = fixture();
    round.events = [{ id: "pickup", type: "bomb_pickup", tick: 101, player_id: "1", player_team: "T" }];
    const projected = createTeamReplayProjector(round, "T", null, source)(104);
    expect(projected.schema_id).toBe("stratai.team_replay_projection@7");
    expect(projected.frame.round_context).toMatchObject({
      phase: "preplant", winner: "", decision_status: "active_round", eligibility: { kind: "bomb_objective" },
    });
    expect(projected.frame.time_remaining).toBeNull();
    expect(projected.frame.bomb).toMatchObject({ state: "carried", carrier_id: "1", position_status: "observed" });
    expect(projected.events.map((e) => e.type)).toEqual(["bomb_pickup", "weapon_fire", "weapon_fire"]);
    expect(projected.frame.shots).toHaveLength(2);
    expect(projected.frame.players.filter((p) => p.has_c4).map((p) => p.steam_id)).toEqual(["1"]);
    expect(projected.frame.players.some((p) => p.steam_id === "8")).toBe(false);
    expect(projected.events.find((e) => e.id === "enemy-fire")).toMatchObject({ x: 80, position_tick: 100 });
    expect(JSON.stringify(projected)).not.toContain("987654");
  });

  it("respects contactPolicy none for both current enemy players and past enemy actions", () => {
    const projected = createTeamReplayProjector(fixture(), "T", null, source, "none")(104);
    expect(projected.schema_id).toBe("stratai.team_replay_projection@7");
    expect(projected.events.map((e) => e.id)).toEqual(["own-fire"]);
    expect(projected.frame.shots).toHaveLength(1);
    expect(projected.frame.players.every((p) => p.team === "T")).toBe(true);
  });

  it("keeps knife and smoke callbacks out of firearm shots while retaining the actual throw", () => {
    const round = fixture();
    round.event_context.events = [action("knife", 101, { weapon: "Knife" }),
      action("smoke-weapon-fire", 101, { weapon: "Smoke Grenade" }),
      action("bayonet", 101, { weapon: "Bayonet" }), action("glock", 102),
      action("throw", 101, { type: "grenade_throw", utility_type: "smoke", projectile_id: "31", source: "demoinfocs_event:GrenadeProjectileThrow" })];
    const projected = createTeamReplayProjector(round, "T", null, source)(104);
    expect(projected.frame.shots.map((e) => e.id)).toEqual(["glock"]);
    expect(projected.events.map((e) => e.id)).toEqual(["throw", "glock"]);
  });

  it("publishes killfeed identities and weapon without using them as contact or killer coordinates", () => {
    const round = fixture();
    round.event_context.events = [];
    round.calculated_vision.samples = [];
    round.events = [{ id: "death", type: "kill", tick: 103, killer_id: "8", killer_team: "CT", killer_name: "Private name",
      killer_x: 987654, killer_y: 987655, victim_id: "9", victim_team: "T", victim_x: 90, victim_y: 20,
      assister_id: "7", assister_x: 987656, weapon: "USP-S" }];
    const projected = createTeamReplayProjector(round, "T", null, source)(104);
    expect(projected.events).toHaveLength(1);
    expect(projected.events[0]).toMatchObject({ type: "kill", knowledge: "public_killfeed", killer_id: "8", killer_team: "CT", weapon: "USP-S" });
    for (const key of ["killer_x", "killer_y", "killer_name", "assister_id", "assister_x", "x", "y"]) {
      expect(projected.events[0]).not.toHaveProperty(key);
    }
    expect(projected.frame.players.some((p) => p.steam_id === "8")).toBe(false);
    expect(projected.frame.death_markers[0].position).toEqual({ x: 90, y: 20 });
    expect(JSON.stringify(projected)).not.toContain("987654");
    expect(createTeamReplayProjector(round, "T", null, source)(102).events).toEqual([]);
  });

  it("does not activate event semantics for the earlier projection without native round context", () => {
    const round = fixture();
    for (const frame of round.frames) delete frame.round_context;
    const projected = createTeamReplayProjector(round, "T", null, source)(104);
    expect(projected.schema_id).toBe("stratai.team_replay_projection@5");
    expect(projected.events).toEqual([]);
    expect(projected.frame.shots).toEqual([]);
  });
});
