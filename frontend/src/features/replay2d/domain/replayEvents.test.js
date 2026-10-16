import { describe, expect, it } from "vitest";
import { createReplayEventsProjector } from "./replayEvents";

// Synthetic ticks deliberately follow the real N01 shape: a teammate throws
// smoke, another teammate sees a CT firing, and that teammate takes damage.
// They are unit fixtures, not an alternative copy of the retained N01 demo.
const player = (id, team, extra = {}) => ({
  steam_id: id, team, name: `P${id.padStart(2, "0")}`, alive: true,
  x: Number(id) * 10, y: 20, z: 0, health: 100, ...extra,
});
const event = (id, type, tick, extra = {}) => ({
  id, type, tick, availability_tick: tick,
  actor_id: "1", actor_team: "T", source: `demoinfocs.${type}`, ...extra,
});
const contact = (extra = {}) => ({
  observer_id: "9", observer_side: "T", subject_id: "8", subject_side: "CT",
  x: 80, y: 20, z: 0, ...extra,
});
const sample = (tick, contacts = []) => ({
  tick, availability_tick: tick, status: "calculated", contacts,
});
const fixture = () => ({
  round: 1, start_tick: 100, end_tick: 116,
  frames: [100, 104, 108, 112].map((tick) => ({
    tick,
    players: [player("1", "T"), player("9", "T"), player("8", "CT"), player("2", "CT")],
    projectiles: [], active_effects: [], shots: [],
  })),
  events: [],
  calculated_vision: {
    schema_id: "stratai.calculated_vision@2", producer: "go.reaction.world_occlusion_v2",
    sample_stride_ticks: 4,
    samples: [sample(100, [contact()]), sample(104), sample(108, [contact()]), sample(112)],
  },
  event_context: { schema_id: "stratai.replay_events@1", events: [] },
});
const ids = (view) => view.events.map((item) => item.id);
const sourceReference = (item) => typeof item.source === "string" ? item.source : item.source?.reference;
const expectEstimated = (item) => {
  expect(item).toBeDefined();
  expect(item.knowledge).toMatch(/calculated|estimated/);
};
const ownThrow = () => event("own-smoke", "grenade_throw", 101, {
  utility_type: "smoke", projectile_id: "31", x: 10, y: 20, z: 0,
});
const projectile = (extra = {}) => ({
  id: "31", type: "smoke", thrower_id: "1", x: 11, y: 21, z: 3,
  trajectory: [10, 20, 11, 21], ...extra,
});

describe("causal replay event projection", () => {
  it("recovers the N01 factual shape without claiming a smoke effect or tactical efficacy", () => {
    const round = fixture();
    round.event_context.events = [
      ownThrow(),
      event("ct-fire", "weapon_fire", 103, { actor_id: "8", actor_team: "CT", weapon: "USP-S", x: 80, y: 20, z: 0 }),
      event("own-fire", "weapon_fire", 103, { actor_id: "9", weapon: "Glock-18", x: 90, y: 20, z: 0 }),
      event("own-hurt", "player_hurt", 103, { actor_id: "8", actor_team: "CT", victim_id: "9", victim_team: "T", damage: 15, health: 85 }),
    ];
    round.frames[1].projectiles = [projectile()];
    const projected = createReplayEventsProjector(round, "T")(104);
    expect(ids(projected)).toEqual(expect.arrayContaining(["own-smoke", "ct-fire", "own-fire", "own-hurt"]));
    expect(projected.shots).toHaveLength(2);
    expect(projected.projectiles).toHaveLength(1);
    expect(projected.active_effects).toEqual([]);
    expect(projected.events.filter((item) => item.type === "utility_start")).toEqual([]);
    expectEstimated(projected.events.find((item) => item.id === "ct-fire"));
    for (const item of projected.events) {
      expect(sourceReference(item)).toBeTruthy();
      expect(item.tick).toBeLessThanOrEqual(item.availability_tick);
      expect(item.availability_tick).toBeLessThanOrEqual(104);
      expect(item).not.toHaveProperty("efficacy");
      expect(item).not.toHaveProperty("intent");
      expect(item).not.toHaveProperty("heard_by");
    }
  });

  it("keeps own fire without vision and rejects a hidden enemy fire event", () => {
    const round = fixture();
    round.calculated_vision.samples = [];
    round.event_context.events = [
      event("own-fire", "weapon_fire", 101, { weapon: "Glock-18" }),
      event("hidden-fire", "weapon_fire", 101, { actor_id: "2", actor_team: "CT", weapon: "secret-weapon" }),
    ];
    const projected = createReplayEventsProjector(round, "T")(104);
    expect(ids(projected)).toEqual(["own-fire"]);
    expect(projected.shots).toHaveLength(1);
    expect(JSON.stringify(projected)).not.toContain("secret-weapon");
  });

  it("gates enemy actions at their event tick and retains dated observations after contact loss", () => {
    const round = fixture();
    round.event_context.events = [
      event("seen-then", "weapon_fire", 103, { actor_id: "8", actor_team: "CT" }),
      event("hidden-then", "weapon_fire", 105, { actor_id: "8", actor_team: "CT" }),
    ];
    const project = createReplayEventsProjector(round, "T");
    expect(ids(project(104))).toEqual(["seen-then"]);
    expect(ids(project(108))).toEqual(["seen-then"]);
    expect(ids(project(102))).toEqual([]);
    expect(ids(project(103))).toEqual(["seen-then"]);
  });

  it("does not extend a calculated contact beyond the half-open four-tick sample interval", () => {
    const round = fixture();
    round.calculated_vision.samples = [sample(100, [contact()])];
    round.event_context.events = [103, 104].map((tick) => event(`shot-${tick}`, "weapon_fire", tick, { actor_id: "8", actor_team: "CT" }));
    expect(ids(createReplayEventsProjector(round, "T")(108))).toEqual(["shot-103"]);
  });

  it.each([
    (round) => { round.calculated_vision.producer = "go.reaction.same_endpoint_v1"; },
    (round) => { round.calculated_vision.samples[0].availability_tick = 104; },
    (round) => { round.calculated_vision.samples[0].status = "unavailable"; },
    (round) => { round.calculated_vision.samples[0].contacts[0].observer_id = "2"; },
    (round) => { round.frames[0].players.find((item) => item.steam_id === "9").alive = false; },
    (round) => { round.events.push({ id: "observer-death", type: "kill", tick: 102, victim_id: "9", victim_team: "T" }); },
  ])("rejects an enemy event when its contact source or observer is unusable (%#)", (invalidate) => {
    const round = fixture();
    round.event_context.events = [event("enemy-fire", "weapon_fire", 103, { actor_id: "8", actor_team: "CT" })];
    invalidate(round);
    expect(createReplayEventsProjector(round, "T")(104).events).toEqual([]);
  });

  it("preserves own damage while redacting an attacker absent from the event-time perspective", () => {
    const round = fixture();
    round.event_context.events = [event("received", "player_hurt", 103, {
      actor_id: "2", actor_team: "CT", victim_id: "9", victim_team: "T", damage: 15, health: 85,
      weapon: "hidden-weapon", actor_name: "Hidden attacker", killer_id: "2", killer_name: "Hidden attacker",
      killer_x: 987654, killer_y: 987655, actor_position: { x: 987654, y: 987655 },
      assister_id: "8", headshot: true, wallbang: true, noscope: true,
    })];
    const projected = createReplayEventsProjector(round, "T")(104);
    expect(projected.events).toHaveLength(1);
    expect(projected.events[0]).toMatchObject({ id: "received", type: "player_hurt", victim_id: "9", damage: 15 });
    expect(projected.events[0].actor_id == null).toBe(true);
    expect(projected.events[0].weapon == null).toBe(true);
    expect(projected.events[0].actor_team == null).toBe(true);
    const serialized = JSON.stringify(projected);
    for (const secret of ["Hidden attacker", "hidden-weapon", "987654", "987655", "assister_id", "wallbang", "noscope", "headshot"]) {
      expect(serialized).not.toContain(secret);
    }
  });

  it("retains the estimated attacker when a valid contact exists at the own-damage tick", () => {
    const round = fixture();
    round.event_context.events = [event("received", "player_hurt", 103, {
      actor_id: "8", actor_team: "CT", victim_id: "9", victim_team: "T", damage: 15, health: 85,
    })];
    const projected = createReplayEventsProjector(round, "T")(108).events[0];
    expect(projected).toMatchObject({ actor_id: "8", victim_id: "9", damage: 15 });
    expectEstimated(projected);
  });

  it("never exports a hidden enemy victim's exact damage or remaining health", () => {
    const round = fixture();
    round.event_context.events = [event("enemy-hurt", "player_hurt", 103, {
      victim_id: "2", victim_team: "CT", damage: 73, health: 27,
    })];
    const projected = createReplayEventsProjector(round, "T")(104);
    expect(projected.events.filter((item) => item.type === "player_hurt")).toEqual([]);
  });

  it("respects delayed availability without borrowing a later contact to authorize the event", () => {
    const round = fixture();
    round.event_context.events = [
      event("delayed-own", "weapon_fire", 101, { availability_tick: 108 }),
      event("delayed-hidden", "weapon_fire", 105, { availability_tick: 108, actor_id: "8", actor_team: "CT" }),
    ];
    const project = createReplayEventsProjector(round, "T");
    expect(project(104).events).toEqual([]);
    expect(ids(project(108))).toEqual(["delayed-own"]);
  });

  it("excludes previous round instances, future events, and impossible availability", () => {
    const round = fixture();
    round.event_context.events = [
      event("previous-instance", "weapon_fire", 99),
      event("past", "weapon_fire", 101),
      event("future", "weapon_fire", 108),
      event("backdated", "weapon_fire", 103, { availability_tick: 102 }),
    ];
    expect(ids(createReplayEventsProjector(round, "T")(104))).toEqual(["past"]);
  });

  it("uses per-type whitelists instead of serializing final shot or utility aggregates", () => {
    const round = fixture();
    const forbidden = {
      hit: true, result: "hit", result_status: "derived", affected_player_ids: ["hidden-aggregate"],
      damage: 9999, health: 8888, duration_ms: 18000, durationMs: 18000,
      duration_status: "observed", duration_source: "final_expiration", time_remaining: 13,
      target_id: "hidden-target", statistics: { future: "final-ledger-secret" },
    };
    round.event_context.events = [
      event("shot", "weapon_fire", 101, forbidden),
      { ...ownThrow(), ...forbidden },
    ];
    const projected = createReplayEventsProjector(round, "T")(104);
    expect(projected.events).toHaveLength(2);
    for (const item of [...projected.events, ...projected.shots]) {
      for (const key of Object.keys(forbidden)) expect(item).not.toHaveProperty(key);
    }
    const serialized = JSON.stringify(projected);
    for (const secret of ["hidden-aggregate", "hidden-target", "final-ledger-secret", "final_expiration"]) {
      expect(serialized).not.toContain(secret);
    }
  });

  it("tracks only the causally sampled projectile of a positively linked own throw", () => {
    const round = fixture();
    round.event_context.events = [ownThrow(), event("visible-enemy-throw", "grenade_throw", 103, {
      actor_id: "8", actor_team: "CT", utility_type: "he", projectile_id: "32",
    })];
    round.frames[1].projectiles = [
      projectile(),
      projectile({ id: "32", thrower_id: "8", type: "he", x: 987654, trajectory: [987653, 20, 987654, 21] }),
    ];
    round.frames[2].projectiles = [projectile({ x: 999999, trajectory: [11, 21, 999999, 21] })];
    const project = createReplayEventsProjector(round, "T");
    expect(project(101).projectiles).toEqual([]);
    const projected = project(105);
    expect(projected.projectiles).toHaveLength(1);
    expect(projected.projectiles[0]).toMatchObject({
      id: "31", x: 11, trajectory: [10, 20, 11, 21], knowledge: "simulated_team_utility_tracking",
    });
    expect(ids(projected)).toContain("visible-enemy-throw");
    expect(JSON.stringify(projected)).not.toContain("987654");
    expect(JSON.stringify(projected)).not.toContain("999999");
  });

  it("retains an own linked effect without the retrospectively computed countdown", () => {
    const round = fixture();
    round.event_context.events = [ownThrow(), event("smoke-start", "utility_start", 104, {
      utility_type: "smoke", projectile_id: "31", x: 11, y: 21, z: 3,
    })];
    round.frames[1].active_effects = [{
      type: "smoke", start_tick: 104, projectile_id: "31", thrower_id: "1", x: 11, y: 21, radius: 144,
      time_remaining: 18, time_remaining_status: "observed", time_remaining_source: "callback_ticks",
      duration_ms: 18000, expiration_tick: 1256,
    }];
    const projected = createReplayEventsProjector(round, "T")(104);
    expect(projected.active_effects).toHaveLength(1);
    expect(projected.active_effects[0]).toMatchObject({ type: "smoke", x: 11, y: 21 });
    expect(projected.active_effects[0].time_remaining == null).toBe(true);
    expect(projected.active_effects[0]).not.toHaveProperty("duration_ms");
    expect(projected.active_effects[0]).not.toHaveProperty("expiration_tick");
  });

  it("does not attribute an effect by proximity or reveal an enemy effect through its visible actor", () => {
    const round = fixture();
    round.event_context.events = [ownThrow(), event("enemy-start", "utility_start", 103, {
      actor_id: "8", actor_team: "CT", utility_type: "smoke", projectile_id: "32", x: 987654, y: 21,
    })];
    round.frames[1].active_effects = [
      { type: "smoke", start_tick: 103, x: 10, y: 20, radius: 144 },
      { type: "smoke", start_tick: 103, projectile_id: "32", thrower_id: "8", x: 987654, y: 21, radius: 144 },
    ];
    const projected = createReplayEventsProjector(round, "T")(104);
    expect(projected.active_effects).toEqual([]);
    expect(JSON.stringify(projected)).not.toContain("987654");
  });

  it("is invariant to hidden raw channels, completed aggregates, and everything after the cut", () => {
    const round = fixture();
    round.event_context.events = [ownThrow(), event("own-shot", "weapon_fire", 103)];
    round.frames[1].projectiles = [projectile()];
    const changed = structuredClone(round);
    changed.frames[2].players[0].x = 987654;
    changed.frames[2].projectiles = [projectile({ x: 987655 })];
    changed.frames[1].players.find((item) => item.steam_id === "2").health = 1;
    changed.frames[1].shots = [{ shooter_id: "2", from_x: 987656 }];
    changed.frames[1].active_effects = [{ type: "smoke", x: 987657, y: 0 }];
    changed.event_context.events[0].damage = 9999;
    changed.event_context.events[0].duration_ms = 99999;
    changed.event_context.events.push(event("future-start", "utility_start", 108, {
      projectile_id: "31", utility_type: "smoke", x: 987658, y: 0,
    }));
    changed.calculated_vision.samples[2].contacts = [contact({ subject_id: "2", x: 987659 })];
    const before = structuredClone(round);
    expect(createReplayEventsProjector(changed, "T")(104)).toEqual(createReplayEventsProjector(round, "T")(104));
    expect(round).toEqual(before);
  });

  it("applies the own/enemy boundary symmetrically to the CT side", () => {
    const round = fixture();
    round.event_context.events = [
      event("t-fire", "weapon_fire", 103),
      event("ct-fire", "weapon_fire", 103, { actor_id: "8", actor_team: "CT" }),
      event("ct-hurt", "player_hurt", 103, { victim_id: "8", victim_team: "CT", damage: 15, health: 85 }),
    ];
    const projected = createReplayEventsProjector(round, "CT")(104);
    expect(ids(projected)).toEqual(expect.arrayContaining(["ct-fire", "ct-hurt"]));
    expect(ids(projected)).not.toContain("t-fire");
    expect(projected.events.find((item) => item.id === "ct-hurt").actor_id == null).toBe(true);
  });

  it("stops a sampled own flight at a causal effect start between replay frames", () => {
    const round = fixture();
    round.event_context.events = [ownThrow(), event("smoke-start", "utility_start", 105, {
      utility_type: "smoke", projectile_id: "31", x: 11, y: 21, z: 3,
    })];
    round.frames[1].projectiles = [projectile()];
    const project = createReplayEventsProjector(round, "T");
    expect(project(104).projectiles).toHaveLength(1);
    expect(project(105).projectiles).toEqual([]);
    expect(ids(project(105))).toContain("smoke-start");
    expect(project(104).projectiles).toHaveLength(1);
  });

  it("preserves causal hull and fire cells for a positively linked own inferno", () => {
    const round = fixture();
    round.event_context.events = [
      { ...ownThrow(), utility_type: "molotov" },
      event("own-fire-start", "utility_start", 104, {
        utility_type: "inferno", projectile_id: "31", x: 11, y: 21, z: 3,
      }),
    ];
    const hull = [10, 20, 14, 20, 14, 24, 10, 24];
    const points = [11, 21, 12, 22];
    round.frames[1].active_effects = [{
      type: "inferno", projectile_id: "31", thrower_id: "1", start_tick: 104,
      x: 11, y: 21, hull, points,
    }];
    round.frames[2].active_effects = [{
      ...round.frames[1].active_effects[0], hull: [999, 999, 1001, 999, 1001, 1001], points: [999, 999],
    }];
    const projected = createReplayEventsProjector(round, "T")(105);
    expect(projected.active_effects).toHaveLength(1);
    expect(projected.active_effects[0]).toMatchObject({
      type: "inferno", hull, points, knowledge: "simulated_team_utility_tracking", observation_tick: 104,
    });
  });

  it("uses the native inferno actor and effect identity without inventing a projectile linkage", () => {
    const round = fixture();
    round.event_context.events = [event("native-inferno-start", "utility_start", 104, {
      effect_id: "effect-own", utility_type: "inferno", x: 11, y: 21, z: 3,
      source: "demoinfocs_event:InfernoStart",
    })];
    round.frames[1].active_effects = [{
      type: "inferno", effect_id: "effect-own", thrower_id: "1", start_tick: 104,
      source: "demoinfocs_event:InfernoStart", x: 11, y: 21,
      hull: [10, 20, 14, 20, 14, 24], points: [11, 21],
    }];
    const projected = createReplayEventsProjector(round, "T")(105);
    expect(projected.active_effects).toHaveLength(1);
    expect(projected.active_effects[0]).toMatchObject({
      effect_id: "effect-own", thrower_id: "1", type: "inferno", x: 11, y: 21,
      hull: [10, 20, 14, 20, 14, 24], points: [11, 21], knowledge: "simulated_team_utility_tracking",
    });
    expect(projected.active_effects[0].projectile_id == null).toBe(true);
    expect(projected.events[0].projectile_id == null).toBe(true);
  });

  it.each(["different-own-actor", "missing-start", "missing-source"])(
    "rejects an unbound native inferno effect (%s)", (invalid) => {
      const round = fixture();
      round.event_context.events = [event("native-inferno-start", "utility_start", 104, {
        effect_id: "effect-own", utility_type: "inferno", source: "demoinfocs_event:InfernoStart",
      })];
      round.frames[1].active_effects = [{
        type: "inferno", effect_id: "effect-own", thrower_id: "1", start_tick: 104,
        source: "demoinfocs_event:InfernoStart", x: 11, y: 21,
      }];
      if (invalid === "different-own-actor") round.frames[1].active_effects[0].thrower_id = "9";
      if (invalid === "missing-start") round.event_context.events = [];
      if (invalid === "missing-source") round.event_context.events[0].source = "";
      expect(createReplayEventsProjector(round, "T")(105).active_effects).toEqual([]);
    },
  );

  it("applies a native effect end between frames and reconstructs the earlier active effect on reverse seek", () => {
    const round = fixture();
    round.event_context.events = [
      event("native-inferno-start", "utility_start", 104, {
        effect_id: "effect-own", utility_type: "inferno", source: "demoinfocs_event:InfernoStart",
      }),
      event("native-inferno-end", "utility_end", 105, {
        effect_id: "effect-own", utility_type: "inferno", source: "demoinfocs_event:InfernoExpired",
      }),
    ];
    round.frames[1].active_effects = [{
      type: "inferno", effect_id: "effect-own", thrower_id: "1", start_tick: 104,
      source: "demoinfocs_event:InfernoStart", x: 11, y: 21,
    }];
    const project = createReplayEventsProjector(round, "T");
    expect(project(104).active_effects).toHaveLength(1);
    expect(project(105).active_effects).toEqual([]);
    expect(ids(project(105))).toContain("native-inferno-end");
    expect(project(104).active_effects).toHaveLength(1);
  });

  it.each([undefined, { schema_id: "stratai.replay_events@0", events: [] }])(
    "abstains for legacy or unsupported event contexts without adapting omniscient raw layers", (context) => {
      const round = fixture();
      round.event_context = context;
      round.events = [event("legacy-hurt", "player_hurt", 101, { victim_id: "9", victim_team: "T", damage: 15 })];
      round.frames[1].shots = [{ tick: 103, shooter_id: "1", source_event_id: "legacy-shot" }];
      round.frames[1].projectiles = [projectile()];
      const projected = createReplayEventsProjector(round, "T")(104);
      for (const channel of ["events", "shots", "projectiles", "active_effects"]) expect(projected[channel]).toEqual([]);
      expect(projected.limitations).toBeTruthy();
      expect(JSON.stringify(projected.limitations)).not.toMatch(/^(?:\[\]|\{\}|null|"")$/);
    },
  );
});
