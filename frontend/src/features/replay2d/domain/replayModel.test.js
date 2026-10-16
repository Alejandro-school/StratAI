import { describe, expect, it } from "vitest";
import {
  activeCombatShotsAtTick,
  activeEventsAtTick,
  closestFrameIndex,
  deriveZThreshold,
  directorRate,
  effectSecondsRemaining,
  effectProgress,
  interpolateFrame,
  normalizeReplayEvent,
  normalizeRound,
  resolveLevel,
} from "./replayModel";
import { createViewport, screenToWorld, worldToScreen, zoomAtPoint } from "../renderer/replayViewport";
import { getHeShake } from "../renderer/renderReplayUtility";
import { buildPlayerLoadout, stableRosterPlayers } from "./weaponPresentation";

const frames = [
  { tick: 100, time_remaining: 90, players: [{ steam_id: 1, x: 0, y: 10, z: -100, yaw: 170, alive: true }] },
  { tick: 110, time_remaining: 89.8, players: [{ steam_id: 1, x: 100, y: 30, z: 100, yaw: -170, alive: true }] },
  { tick: 120, time_remaining: 89.6, players: [{ steam_id: 1, x: 200, y: 50, z: 100, yaw: -160, alive: true }] },
];

describe("replay model", () => {
  it("finds the closest frame with a binary search", () => {
    expect(closestFrameIndex(frames, 116)).toBe(2);
    expect(closestFrameIndex(frames, 103)).toBe(0);
  });

  it("interpolates position, clock and wrapped angles", () => {
    const frame = interpolateFrame(frames, 105);
    expect(frame.players[0].x).toBe(50);
    expect(frame.time_remaining).toBeCloseTo(89.9);
    expect(Math.abs(frame.players[0].yaw)).toBe(180);
  });

  it("normalizes historical and current utility events", () => {
    expect(normalizeReplayEvent({ type: "flash_explode", tick: 42 }).subtype).toBe("flashbang");
    expect(normalizeReplayEvent({ type: "grenade_explode", grenade_type: "he", tick: 42 }).type).toBe("utility_detonate");
    expect(normalizeReplayEvent({ type: "utility_detonate", utility_type: "smoke", tick: 42 }).lane).toBe("utility");
  });

  it("derives deterministic effect lifetimes", () => {
    const event = normalizeReplayEvent({ type: "utility_detonate", utility_type: "he", tick: 100, duration_ms: 1000 });
    expect(effectProgress(132, event, 64)).toBe(0.5);
    expect(activeEventsAtTick([event], 165, 64)).toHaveLength(0);
  });

  it("reconstructs smoke start ticks in historical replays", () => {
    const round = normalizeRound({
      frames: [
        { tick: 100, active_effects: [{ type: "smoke", x: 10, y: 20, time_remaining: 18 }] },
        { tick: 164, active_effects: [{ type: "smoke", x: 10, y: 20, time_remaining: 18 }] },
      ],
      events: [],
    });
    expect(round.frames[1].active_effects[0].start_tick).toBe(100);
    expect(effectSecondsRemaining(round.frames[1].active_effects[0], 164, 64)).toBe(17);
  });

  it("uses sampled frame bounds to skip buytime and retain post-round", () => {
    const round = normalizeRound({
      start_tick: 40,
      end_tick: 180,
      frames: [{ tick: 100 }, { tick: 220 }, { tick: 164 }],
      events: [],
    });
    expect(round.frames.map((frame) => frame.tick)).toEqual([100, 164, 220]);
    expect(round.start_tick).toBe(100);
    expect(round.end_tick).toBe(220);
  });

  it("retains the official boundary for replay rounds with atomic combat shots", () => {
    const round = normalizeRound({
      start_tick: 90,
      end_tick: 120,
      frames: [{ tick: 100 }, { tick: 110 }],
      events: [],
      combat_shots: [{ tick: 118, source_event_id: "fire-1" }],
    });
    expect(round.end_tick).toBe(120);
    expect(round.combat_shots).toEqual([{ tick: 118, source_event_id: "fire-1" }]);
  });

  it("projects atomic shots by causal tick without fabricating a carrier frame", () => {
    const shots = [{ tick: 118, source_event_id: "fire-1" }];
    expect(activeCombatShotsAtTick(shots, 117)).toEqual([]);
    expect(activeCombatShotsAtTick(shots, 118)).toEqual(shots);
    expect(activeCombatShotsAtTick(shots, 151)).toEqual([]);
  });

  it("reconstructs bomb state so the plant position is not visible early", () => {
    const leakedBomb = { state: "carried", x: 1208, y: 2561, site: "A", plant_tick: 200 };
    const round = normalizeRound({
      frames: [
        { tick: 100, players: [], bomb: leakedBomb },
        { tick: 220, players: [], bomb: leakedBomb },
      ],
      events: [{ type: "bomb_plant", tick: 200, x: 1208, y: 2561, site: "A" }],
    });
    expect(round.frames[0].bomb).toBeNull();
    expect(round.frames[1].bomb).toMatchObject({ state: "planted", x: 1208, y: 2561, plant_tick: 200 });
  });

  it("derives hit events from health changes in historical replay frames", () => {
    const round = normalizeRound({
      frames: [
        { tick: 100, players: [{ steam_id: 7, name: "Seven", team: "CT", health: 100, x: 10, y: 20 }] },
        { tick: 104, players: [{ steam_id: 7, name: "Seven", team: "CT", health: 72, x: 12, y: 22 }] },
      ],
      events: [],
    });
    expect(round.events).toContainEqual(expect.objectContaining({ type: "player_hurt", victim_id: 7, damage: 28 }));
  });

  it("generates a deterministic HE shake and disables it for reduced motion", () => {
    const events = [normalizeReplayEvent({ type: "utility_detonate", utility_type: "he", tick: 100 })];
    expect(getHeShake(events, 104, 64, false)).not.toEqual({ x: 0, y: 0 });
    expect(getHeShake(events, 104, 64, true)).toEqual({ x: 0, y: 0 });
  });

  it("changes director speed only outside event windows", () => {
    const events = [{ tick: 640 }];
    expect(directorRate(events, 640, 64, true, 1)).toBe(1);
    expect(directorRate(events, 0, 64, true, 1)).toBe(2);
    expect(directorRate(events, 0, 64, false, 1)).toBe(1);
  });

  it("derives and resolves map levels from Z", () => {
    const threshold = deriveZThreshold(frames);
    expect(threshold).toBe(0);
    expect(resolveLevel(frames[1].players, threshold, 1)).toBe("upper");
  });
});

describe("player presentation", () => {
  it("keeps roster order stable and identifies the active weapon", () => {
    const players = [
      { steam_id: "9", team: "CT" },
      { steam_id: "2", team: "CT" },
    ];
    expect(stableRosterPlayers(players, "CT").map((player) => player.steam_id)).toEqual(["2", "9"]);
    expect(buildPlayerLoadout({
      weapon: "AK-47",
      weapons: ["Glock-18", "AK-47", "Smoke Grenade"],
    })).toMatchObject({
      primary: "ak47",
      pistol: "glock",
      active: "ak47",
      utility: ["smokegrenade"],
    });
  });
});

describe("native C4 normalization", () => {
  const carried = {
    state: "carried", objective_phase: "preplant", carrier_id: "1",
    is_planted_now: false, was_planted_this_round: false,
    position_status: "observed", position_source: "demoinfocs_native_snapshot",
    x: 12, y: 24,
  };

  it("preserves a native carried snapshot when the causal prefix has no plant event", () => {
    const round = normalizeRound({ frames: [{ tick: 100, bomb: carried }], events: [] });
    expect(round.frames[0].bomb).toEqual(carried);
  });

  it("does not change an earlier native carrier when a future plant is added", () => {
    const prefix = { frames: [{ tick: 100, bomb: carried }], events: [] };
    const complete = { ...prefix, events: [{ type: "bomb_plant", tick: 200, x: 999, y: 888, site: "B" }] };
    expect(normalizeRound(complete).frames[0].bomb).toEqual(carried);
    expect(normalizeRound(complete).frames[0].bomb).toEqual(normalizeRound(prefix).frames[0].bomb);
  });

  it("keeps N10 resolved instead of reviving its earlier planted state or plant coordinates", () => {
    const resolved = {
      state: "resolved", objective_phase: "resolved", is_planted_now: false,
      was_planted_this_round: true, carrier_id: null, defuser_id: null,
      position_status: "observed", position_source: "demoinfocs_native_snapshot",
      x: -1897.59375, y: 242.0625, site: "B", plant_tick: 68795,
    };
    const round = normalizeRound({
      frames: [{ tick: 69957, bomb: resolved }],
      events: [{ type: "bomb_plant", tick: 68795, x: -1897.58935546875, y: 242.04856872558594, site: "B" }],
    });
    expect(round.frames[0].bomb).toEqual(resolved);
  });

  it("preserves known native state without substituting historical plant coordinates for missing current position", () => {
    const planted = {
      state: "planted", objective_phase: "postplant", is_planted_now: true,
      was_planted_this_round: true, position_status: "unknown", plant_tick: 100,
    };
    const round = normalizeRound({
      frames: [{ tick: 200, bomb: planted }],
      events: [{ type: "bomb_plant", tick: 100, x: 700, y: 800 }],
    });
    expect(round.frames[0].bomb).toMatchObject(planted);
    expect(round.frames[0].bomb.x).toBeUndefined();
    expect(round.frames[0].bomb.y).toBeUndefined();
  });
});

describe("replay viewport", () => {
  const config = { pos_x: -1000, pos_y: 1000, scale: 2 };

  it("round-trips game coordinates across resize and zoom", () => {
    const viewport = createViewport(1280, 720, 1.7, { x: 20, y: -15 });
    const world = { x: -200, y: 350 };
    const result = screenToWorld(worldToScreen(world, config, viewport), config, viewport);
    expect(result.x).toBeCloseTo(world.x);
    expect(result.y).toBeCloseTo(world.y);
  });

  it("keeps the map point under the cursor while zooming", () => {
    const viewport = createViewport(1000, 700, 1, { x: 0, y: 0 });
    const cursor = { x: 330, y: 280 };
    const before = screenToWorld(cursor, config, viewport);
    const next = zoomAtPoint(viewport, cursor, 2);
    const after = screenToWorld(cursor, config, createViewport(1000, 700, next.zoom, next.pan));
    expect(after.x).toBeCloseTo(before.x);
    expect(after.y).toBeCloseTo(before.y);
  });
});
