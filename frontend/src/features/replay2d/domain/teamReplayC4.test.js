import { describe, expect, it } from "vitest";
import { normalizeRound } from "./replayModel";
import { createTeamReplayProjector } from "./teamReplayProjection";

const player = (steam_id, team, extra = {}) => ({
  steam_id, team, name: steam_id, alive: true, health: 100,
  x: team === "T" ? 100 : -100, y: 200, z: 0, has_c4: false, ...extra,
});
const players = () => [player("1", "T"), player("2", "T"), player("3", "CT")];
const dropped = (extra = {}) => ({
  state: "dropped", objective_phase: "preplant", is_planted_now: false,
  was_planted_this_round: false, carrier_id: null, defuser_id: null,
  position_status: "observed", position_source: "demoinfocs_native_snapshot",
  x: 880.28125, y: 622.59375, ...extra,
});
const roundWith = (bomb, extra = {}) => ({
  round: 3, start_tick: 100, end_tick: 108,
  frames: [{ tick: 100, players: players(), bomb }], events: [], ...extra,
});

describe("causal C4 team projection", () => {
  it("does not publish an event before its explicit availability", () => {
    const raw=roundWith(dropped(), {events:[{type:"bomb_pickup",tick:104,availability_tick:110,player_id:"1"}]});
    expect(createTeamReplayProjector(raw,"T")(104).frame.bomb.state).toBe("dropped");
  });
  it("keeps pickup historical when its carrier later dies without a captured drop", () => {
    const raw=roundWith(dropped(), {events:[{type:"bomb_pickup",tick:104,player_id:"2"},
      {type:"kill",tick:106,victim_id:"2",victim_team:"T"}]});
    const view=createTeamReplayProjector(raw,"T")(106);
    expect(view.frame.bomb.carrier_id).toBeUndefined();
    expect(view.frame.bomb.state).toBe("unknown");
    expect(view.frame.players.filter((p)=>p.has_c4)).toHaveLength(0);
    expect(view.events.some((e)=>e.type==="bomb_pickup")).toBe(true);
  });
  it("invalidates a snapshot carrier killed before the next C4 sample without inventing a drop", () => {
    const raw=roundWith(dropped({state:"carried",carrier_id:"2"}), {events:[
      {type:"kill",tick:106,victim_id:"2",victim_team:"T"}]});
    const bomb=createTeamReplayProjector(raw,"T")(106).frame.bomb;
    expect(bomb.state).toBe("unknown");
    expect(bomb.x).toBeUndefined();
    expect(bomb.carrier_id).toBeUndefined();
  });
  it("preserves same-tick start then abort before the next sampled frame", () => {
    const raw=roundWith(dropped({state:"carried",carrier_id:"1"}), {events:[
      {type:"bomb_plant_start",tick:104,player_id:"1",objective_sequence:10},
      {type:"bomb_plant_abort",tick:104,player_id:"1",objective_sequence:11}]});
    expect(createTeamReplayProjector(raw,"T")(104).frame.bomb.state).toBe("carried");
  });
  it("retains the N03 native dropped position for T without requiring a carrier", () => {
    const round = normalizeRound(roundWith(dropped()));
    const bomb = createTeamReplayProjector(round, "T")(100).frame.bomb;
    expect(bomb).toMatchObject({
      state: "dropped", objective_phase: "preplant", is_planted_now: false,
      x: 880.28125, y: 622.59375, position_status: "observed", availability_tick: 100,
    });
    expect(bomb.source.reference).toBe("round:3:frame:100:bomb");
  });

  it("retains the tracker carrier through normalization and does not invent one from another inventory", () => {
    const raw = roundWith(dropped({ state: "carried", carrier_id: "1" }));
    raw.frames[0].players[0].has_c4 = true;
    raw.frames[0].players[0].inventory = ["C4"];
    raw.frames[0].players[1].inventory = ["C4"];
    const bomb = createTeamReplayProjector(normalizeRound(raw), "T")(100).frame.bomb;
    expect(bomb).toMatchObject({ state: "carried", carrier_id: "1", objective_phase: "preplant" });
  });

  it("keeps the own tracker carrier when its position is unavailable", () => {
    const bomb = createTeamReplayProjector(roundWith({
      state: "carried", objective_phase: "preplant", carrier_id: "1", position_status: "unknown",
    }), "T")(100).frame.bomb;
    expect(bomb).toMatchObject({ state: "carried", carrier_id: "1", position_status: "unknown" });
    expect(bomb.x).toBeUndefined();
    expect(bomb.y).toBeUndefined();
  });

  it("distinguishes a known dropped state from an unknown location and ignores the dropper position", () => {
    const round = roundWith({ state: "dropped", objective_phase: "preplant", position_status: "unknown" }, {
      events: [{ id: "drop", type: "bomb_drop", tick: 100, player_id: "1", x: 777, y: 888 }],
    });
    const bomb = createTeamReplayProjector(round, "T")(100).frame.bomb;
    expect(bomb).toMatchObject({ state: "dropped", objective_phase: "preplant", position_status: "unknown" });
    expect(bomb.x).toBeUndefined();
    expect(bomb.y).toBeUndefined();
  });

  it.each([dropped(), dropped({ state: "carried", carrier_id: "1" })])(
    "does not expose hidden preplant C4, carrier or coordinates to CT (%j)", (bomb) => {
      const round = roundWith(bomb);
      const changed = structuredClone(round);
      changed.frames[0].bomb.x = 99881;
      changed.frames[0].bomb.y = 99882;
      changed.frames[0].bomb.carrier_id = "hidden-carrier";
      const project = (source) => createTeamReplayProjector(source, "CT")(100);
      expect(project(round).frame.bomb).toBeNull();
      expect(project(changed)).toEqual(project(round));
    },
  );

  it.each(["T", "CT"])("does not turn N10 resolution back into a live planted bomb for %s", (side) => {
    const resolved = dropped({
      state: "resolved", objective_phase: "resolved", is_planted_now: false,
      was_planted_this_round: true, plant_tick: 100, x: -1897.59375, y: 242.0625,
    });
    const round = roundWith(resolved, {
      frames: [{ tick: 108, players: players(), bomb: resolved }],
      events: [{ type: "bomb_plant", tick: 100, player_id: "1", x: -1897.58935546875, y: 242.04856872558594 }],
    });
    const bomb = createTeamReplayProjector(round, side)(108).frame.bomb;
    expect(bomb).toMatchObject({ state: "resolved", objective_phase: "resolved", is_planted_now: false });
  });

  it("does not interpolate, hold old C4 coordinates as current, or use a future pickup", () => {
    const raw = roundWith(dropped(), {
      frames: [
        { tick: 100, players: players(), bomb: dropped() },
        { tick: 108, players: players(), bomb: dropped({ state: "carried", carrier_id: "2", x: 900, y: 700 }) },
      ],
    });
    const project = createTeamReplayProjector(raw, "T");
    expect(project(100).frame.bomb).toMatchObject({ state: "dropped", x: 880.28125, y: 622.59375 });
    const between = project(104).frame.bomb;
    expect(between.state).toBe("dropped");
    expect(between.x).toBeUndefined();
    expect(between.y).toBeUndefined();
    expect(between.carrier_id ?? null).toBeNull();
    expect(project(108).frame.bomb).toMatchObject({ state: "carried", carrier_id: "2", x: 900, y: 700 });
    expect(project(100).frame.bomb).toMatchObject({ state: "dropped", x: 880.28125, y: 622.59375 });
  });

  it("does not restore an older observed position after a newer sample explicitly lacks coordinates", () => {
    const raw = roundWith(dropped(), {
      frames: [
        { tick: 100, players: players(), bomb: dropped() },
        { tick: 108, players: players(), bomb: { state: "dropped", objective_phase: "preplant", position_status: "unknown" } },
      ],
    });
    const bomb = createTeamReplayProjector(raw, "T")(108).frame.bomb;
    expect(bomb).toMatchObject({ state: "dropped", position_status: "unknown" });
    expect(bomb.x).toBeUndefined();
    expect(bomb.y).toBeUndefined();
  });

  it("keeps C4 projection invariant to future plant, pickup and resolution changes", () => {
    const raw = roundWith(dropped(), {
      frames: [{ tick: 100, players: players(), bomb: dropped() }, { tick: 108, players: players(), bomb: null }],
      events: [{ type: "bomb_plant", tick: 108, player_id: "1", x: 12, y: 34 }],
    });
    const changed = structuredClone(raw);
    changed.frames[1].bomb = dropped({ state: "resolved", objective_phase: "resolved", x: 99881, y: 99882 });
    changed.events[0] = { type: "bomb_defuse", tick: 108, player_id: "3", x: -123, y: -456 };
    const project = (source) => createTeamReplayProjector(normalizeRound(source), "T")(100);
    expect(project(raw).frame.bomb).toMatchObject({ state: "dropped", x: 880.28125, y: 622.59375 });
    expect(project(changed)).toEqual(project(raw));
  });

  it("applies an own drop causally between samples without treating actor coordinates as C4 coordinates", () => {
    const raw = roundWith(dropped({ state: "carried", carrier_id: "1" }), {
      events: [{ id: "own-drop", type: "bomb_drop", tick: 104, player_id: "1", x: 777, y: 888 }],
    });
    const project = createTeamReplayProjector(raw, "T");
    expect(project(100).frame.bomb).toMatchObject({ state: "carried", carrier_id: "1" });
    const bomb = project(104).frame.bomb;
    expect(bomb.state).toBe("dropped");
    expect(bomb.carrier_id ?? null).toBeNull();
    expect(bomb.x).toBeUndefined();
    expect(bomb.y).toBeUndefined();
    expect(project(104).events).toContainEqual(expect.objectContaining({ type: "bomb_drop", tick: 104 }));
    expect(project(103).events.some((event) => event.type === "bomb_drop")).toBe(false);
  });

  it("applies an own pickup to the unique carrier without exposing it to CT", () => {
    const raw = roundWith(dropped(), {
      events: [{ id: "own-pickup", type: "bomb_pickup", tick: 104, player_id: "2", x: 777, y: 888 }],
    });
    const bomb = createTeamReplayProjector(raw, "T")(104).frame.bomb;
    expect(bomb).toMatchObject({ state: "carried", carrier_id: "2" });
    expect(bomb.x).toBeUndefined();
    expect(bomb.y).toBeUndefined();
    const ct = createTeamReplayProjector(raw, "CT")(104);
    expect(ct.frame.bomb).toBeNull();
    expect(ct.events.some((event) => event.type === "bomb_pickup")).toBe(false);
  });
});
