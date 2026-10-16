import { describe, expect, it } from "vitest";
import { createTeamReplayProjector } from "./teamReplayProjection";

const identity = { map_name: "de_mirage" };
const own = { steam_id: "1", team: "T", alive: true, x: -1500, y: 750, z: 0, place: { zone: "FORGED" } };
const enemy = { steam_id: "2", team: "CT", alive: true, x: -400, y: -400, z: -100 };
const round = () => ({ round: 1, start_tick: 100, frames: [
  { tick: 100, players: [own, enemy] }, { tick: 108, players: [own, enemy] },
], events: [], calculated_vision: { schema_id: "stratai.calculated_vision@2",
  producer: "go.reaction.world_occlusion_v2", sample_stride_ticks: 4, samples: [] } });
const project = (data, tick = 100) => createTeamReplayProjector(data, "T", null, identity)(tick);

describe("places after causal team projection", () => {
  it("ignores raw labels and never locates a hidden enemy", () => {
    const data = round(), view = project(data);
    expect(view.schema_id).toBe("stratai.team_replay_projection@5");
    expect(view.frame.players).toHaveLength(1);
    expect(view.frame.players[0].place).toMatchObject({ zone: "Apartments", observation_tick: 100, xy_precision: 0.5 });
    expect(JSON.stringify(view)).not.toContain("FORGED");
    const changed = structuredClone(data);
    changed.frames[0].players[1] = { ...enemy, x: -9999, y: 9999, place: { zone: "BombsiteA" } };
    expect(project(changed)).toEqual(view);
  });
  it("uses the allowed contact position and its own timestamp, not the raw enemy position", () => {
    const data = round();
    data.calculated_vision.samples.push({ tick: 100, availability_tick: 100, status: "calculated", contacts: [
      { observer_id: "1", observer_side: "T", subject_id: "2", subject_side: "CT", x: -1500, y: 750, z: 0 },
    ] });
    const contact = project(data, 102).frame.players.find((p) => p.steam_id === "2");
    expect(contact.place).toMatchObject({ zone: "Apartments", observation_tick: 100, availability_tick: 100, xy_precision: 0 });
    expect(project(data, 104).frame.players).toHaveLength(1);
    delete data.calculated_vision.samples[0].contacts[0].z;
    const missing = project(data).frame.players.find((p) => p.steam_id === "2");
    expect(missing.z).toBeUndefined();
    expect(missing.place).toMatchObject({ status: "unavailable", reason: "position_unavailable", zone: null });
  });
  it("removes a living place on death and cannot see future moves", () => {
    const data = round();
    data.events.push({ type: "kill", tick: 104, victim_id: "1", victim_team: "T" });
    expect(project(data, 104).frame.players[0].place).toBeUndefined();
    const before = project(data);
    data.frames[1].players = [{ ...own, x: 8888 }];
    expect(project(data)).toEqual(before);
  });
  it("retains version 4 when a Mirage map reference was not selected", () => {
    for (const source of [null, { map_name: "de_other" }]) {
      const view = createTeamReplayProjector(round(), "T", null, source)(100);
      expect(view.schema_id).toBe("stratai.team_replay_projection@4");
      expect(view.place_reference_id).toBeUndefined();
    }
  });
});
