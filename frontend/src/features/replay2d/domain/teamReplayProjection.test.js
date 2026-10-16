import { describe, expect, it } from "vitest";
import { createTeamReplayProjector } from "./teamReplayProjection";
import { normalizeRound } from "./replayModel";

const identity = { match_id: "opaque-demo", demo_sha256: "a".repeat(64) };
const source = { reference: "independent-source:fixture-only" };
const player = (id, team, x, extra = {}) => ({ steam_id: id, team, x, y: 12, z: 0, yaw: 30,
  name: id, alive: true, health: 100, armor: 80, money: 5000, weapon: "AK-47", has_c4: team === "T", ...extra });
const fixture = () => normalizeRound({ round: 2, start_tick: 10, end_tick: 50, frames: [
  { tick: 10, players: [player("own", "T", 100), player("hidden", "CT", 800)], shots: [{ shooter_id: "hidden", from_x: 800 }],
    active_effects: [{ type: "smoke", x: 800, y: 12 }], projectiles: [{ x: 800, y: 12 }], time_remaining: 80 },
  { tick: 30, players: [player("own", "T", 110), player("hidden", "CT", 900)], time_remaining: 70 },
  { tick: 50, players: [player("own", "T", 999), player("hidden", "CT", 1234, { alive: false })], time_remaining: 60 },
], events: [
  { id: "death", type: "kill", tick: 20, killer_id: "secret-killer", killer_name: "Hidden Killer", killer_x: 500,
    victim_id: "hidden", victim_name: "Rival", victim_team: "CT", victim_x: 810, victim_y: 15, weapon: "AWP" },
  { id: "future", type: "kill", tick: 45, victim_id: "own", victim_team: "T", victim_x: 115, victim_y: 16 },
] });
const vision = (observations) => ({ schema_id: "stratai.confirmed_vision_source@1", ...identity, round: 2, source, observations });
const contact = (extra = {}) => ({ tick: 10, availability_tick: 10, observer_id: "own", observer_side: "T",
  subject_id: "visible", subject_side: "CT", status: "observed", visual_contact: true, source,
  position: { x: 777, y: 55, z: 2 }, subject_name: "Rival visible", ...extra });

describe("team replay causal projection", () => {
  it("excludes earlier deaths after a round identifier is reused on reset", () => {
    const round=fixture();
    round.events.unshift({type:"kill",tick:5,victim_id:"own",victim_team:"T",victim_x:999,victim_y:999});
    const view=createTeamReplayProjector(round,"T")(12);
    expect(view.frame.players[0].alive).toBe(true);
    expect(view.deaths).toHaveLength(0);
  });
  it("reveals live rivals from calculated contacts and hides them at loss, gaps, death and reverse seek", () => {
    const round = fixture();
    round.calculated_vision = { schema_id: "stratai.calculated_vision@2", producer: "go.reaction.world_occlusion_v2", sample_stride_ticks: 4,
      samples: [{ tick: 12, availability_tick: 12, status: "calculated", contacts: [
        { observer_id: "own", observer_side: "T", subject_id: "44", subject_side: "CT", x: 321, y: 123, z: 0, health: 80 },
      ] }, { tick: 16, availability_tick: 16, status: "calculated", contacts: [] }] };
    const project = createTeamReplayProjector(round, "T");
    expect(project(11).frame.players).toHaveLength(1);
    const enemy = project(12).frame.players.at(-1);
    expect(enemy.x).toBe(321); expect(enemy.knowledge).toBe("calculated_visual_contact");
    expect(enemy.health).toBeUndefined(); expect(project(15.99).frame.players).toHaveLength(2);
    expect(project(16).frame.players).toHaveLength(1); expect(project(12).frame.players).toHaveLength(2);
    expect(project(40).vision_status).toBe("unavailable");
    expect(createTeamReplayProjector(round, "T", null, null, "confirmed")(12).frame.players).toHaveLength(1);
    round.events.push({ type: "kill", tick: 13, victim_id: "44", victim_team: "CT" });
    expect(createTeamReplayProjector(round, "T")(13).frame.players).toHaveLength(1);
    round.calculated_vision.samples[0].availability_tick = 17;
    expect(createTeamReplayProjector(round, "T")(12).frame.players).toHaveLength(1);
  });
  it("does not reuse the old contact producer that omitted opaque world walls", () => {
    const round = fixture();
    round.calculated_vision = { schema_id: "stratai.calculated_vision@1", producer: "go.reaction.same_endpoint_v1", sample_stride_ticks: 4,
      samples: [{ tick: 12, availability_tick: 12, status: "calculated", contacts: [
        { observer_id: "own", observer_side: "T", subject_id: "44", subject_side: "CT", x: 321, y: 123 },
      ] }] };
    const view = createTeamReplayProjector(round, "T")(12);
    expect(view.frame.players).toHaveLength(1);
    expect(view.vision_status).toBe("unavailable");
  });
  it("hides all enemy channels, future events and interpolation into future snapshots", () => {
    const view = createTeamReplayProjector(fixture(), "T")(19);
    expect(view.frame.players.map((p) => p.steam_id)).toEqual(["own"]);
    expect(view.frame.players[0].x).toBe(100);
    expect(view.frame.shots).toEqual([]);
    expect(view.frame.active_effects).toEqual([]);
    expect(view.frame.projectiles).toEqual([]);
    expect(view.deaths).toEqual([]);
    expect(view.frames).toHaveLength(1);
    expect(view.vision_status).toBe("unavailable");
  });
  it("is invariant to hidden enemy, unknown layers and future mutations", () => {
    const round = fixture(); const changed = structuredClone(round);
    changed.frames[0].players[1] = player("leaked", "CT", 99999);
    changed.frames[0].shots = [{ shooter_id: "secret", from_x: -999 }];
    changed.frames[1].players[0].x = -999;
    changed.events.at(-1).victim_x = -999;
    expect(createTeamReplayProjector(changed, "T")(19)).toEqual(createTeamReplayProjector(round, "T")(19));
  });
  it("persists the event death point without revealing killer or corpse movement", () => {
    const project = createTeamReplayProjector(fixture(), "T");
    expect(project(25).frame.death_markers[0].position).toEqual({ x: 810, y: 15 });
    expect(project(40).frame.death_markers[0].position).toEqual({ x: 810, y: 15 });
    expect(JSON.stringify(project(40))).not.toContain("secret-killer");
    expect(JSON.stringify(project(40))).not.toContain("Hidden Killer");
    expect(project(40).events[0].killer_x).toBeUndefined();
    expect(project(19).deaths).toHaveLength(0);
  });
  it("keeps missing death coordinates unknown, preserving actual zero", () => {
    const round = fixture();
    round.events = [{ type: "kill", tick: 20, victim_team: "CT", victim_id: "one" },
      { type: "kill", tick: 21, victim_team: "T", victim_id: "two", victim_x: 0, victim_y: 0 }];
    const view = createTeamReplayProjector(round, "T")(25);
    expect(view.unknown_death_positions).toBe(1);
    expect(view.frame.death_markers[0].position).toEqual({ x: 0, y: 0 });
  });
  it("resets between rounds and reconstructs correctly after seeking backward", () => {
    const round = fixture(); const project = createTeamReplayProjector(round, "T");
    expect(project(40).deaths).toHaveLength(1);
    expect(project(19).deaths).toHaveLength(0);
    expect(createTeamReplayProjector({ ...round, round: 3, events: [] }, "T")(40).deaths).toHaveLength(0);
  });
  it("uses a bound confirmed source position, strips resources, and stops updating on lost contact", () => {
    const project = createTeamReplayProjector(fixture(), "T", vision([contact()]), identity);
    const enemy = project(19).frame.players.find((p) => p.steam_id === "visible");
    expect(enemy.x).toBe(777);
    expect(enemy.health).toBeUndefined(); expect(enemy.money).toBeUndefined(); expect(enemy.yaw).toBeUndefined();
    expect(project(30).frame.players.some((p) => p.steam_id === "visible")).toBe(false);
  });
  it.each([
    { status: "estimated" }, { visual_contact: null }, { availability_tick: 25 },
    { observer_side: "CT" }, { observer_id: "hidden" }, { source: null },
    { position: null }, { subject_side: "T" },
  ])("rejects an unusable contact %j", (extra) => {
    const view = createTeamReplayProjector(fixture(), "T", vision([contact(extra)]), identity)(19);
    expect(view.frame.players).toHaveLength(1);
  });
  it("rejects a source from another match or round and ignores legacy flags", () => {
    const round = fixture(); round.frames[0].enemy_los = true;
    const other = { ...vision([contact()]), match_id: "another" };
    expect(createTeamReplayProjector(round, "T", other, identity)(19).vision_status).toBe("unavailable");
    expect(createTeamReplayProjector(round, "T", { ...vision([contact()]), round: 1 }, identity)(19).vision_status).toBe("unavailable");
  });
  it("supports CT with the same factual rule, without adding tactical CT criteria", () => {
    const view = createTeamReplayProjector(fixture(), "CT")(19);
    expect(view.frame.players.map((p) => p.steam_id)).toEqual(["hidden"]);
    expect(view.frame.players[0].x).toBe(800);
  });
  it("integrates an own observed plant position while withholding hidden CT objective details", () => {
    const round=fixture();
    round.frames[0].bomb={state:"planted",position_status:"observed",x:456,y:123,defuser_id:"hidden"};
    round.events.push({id:"plant",type:"bomb_plant",tick:10,player_id:"own"});
    const t=createTeamReplayProjector(round,"T")(10);
    expect(t.frame.bomb.x).toBe(456); expect(t.frame.bomb.defuser_id).toBeUndefined();
    expect(createTeamReplayProjector(round,"T")(19).frame.bomb.x).toBeUndefined();
    round.frames[0].bomb.defuser_id=null;
    const ct=createTeamReplayProjector(round,"CT")(19);
    expect(ct.frame.bomb.state).toBe("planted"); expect(ct.frame.bomb.x).toBeUndefined();
  });
});
