import { describe, expect, it } from "vitest";
import {
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

  it("generates a deterministic HE shake and disables it for reduced motion", () => {
    const events = [normalizeReplayEvent({ type: "utility_detonate", utility_type: "he", tick: 100 })];
    expect(getHeShake(events, 104, 64, false)).not.toEqual({ x: 0, y: 0 });
    expect(getHeShake(events, 104, 64, true)).toEqual({ x: 0, y: 0 });
  });

  it("changes director speed only outside event windows", () => {
    const events = [{ tick: 640 }];
    expect(directorRate(events, 640, 64, true, 1)).toBe(0.75);
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
