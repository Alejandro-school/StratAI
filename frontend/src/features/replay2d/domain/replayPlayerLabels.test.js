import { expect, it } from "vitest";
import { createTeamPlayerLabels, labelCoachText, labelReplayEvents, labelReplayFrame } from "./replayPlayerLabels";

it("keeps team numbers stable when contacts disappear or the roster order changes", () => {
  const roster = [1, 10, 6, 7, 9].map((id) => ({ steam_id: String(id), team: "T" }))
    .concat([2, 3, 4, 5, 8].map((id) => ({ steam_id: String(id), team: "CT" })));
  const labels = createTeamPlayerLabels(roster);
  expect(labels).toEqual({ 1: "T1", 6: "T2", 7: "T3", 9: "T4", 10: "T5", 2: "CT1", 3: "CT2", 4: "CT3", 5: "CT4", 8: "CT5" });
  expect(createTeamPlayerLabels([...roster].reverse())).toEqual(labels);
  const frame = { tick: 100, players: [{ steam_id: "8", team: "CT", name: "P08", x: 10, knowledge: "calculated_visual_contact" }],
    death_markers: [{ victim_id: "6", name: "P06", tick: 50, source: { reference: "death-6" } }] };
  const original = JSON.stringify(frame);
  const displayed = labelReplayFrame(frame, labels);
  expect(displayed.players[0]).toEqual({ ...frame.players[0], name: "CT5", display_number: 5 });
  expect(displayed.players[0]).not.toHaveProperty("money");
  expect(displayed.death_markers[0]).toEqual({ ...frame.death_markers[0], name: "T2" });
  expect(JSON.stringify(frame)).toBe(original);
});

it("translates player mentions without changing references, unknown aliases or original claims", () => {
  const labels = { 1: "T1", 10: "T5", 8: "CT5" };
  const text = "P01 vio a P08; P10 espera. XP01, P0100 y P99 desconocidos.";
  expect(labelCoachText(text, labels)).toBe("T1 vio a CT5; T5 espera. XP01, P0100 y P99 desconocidos.");
  const event = { id: "event-P01", source: { reference: "P01:2" }, victim_id: "8", victim_name: "P08", tick: 10 };
  expect(labelReplayEvents([event], labels)).toEqual([{ ...event, victim_name: "CT5" }]);
  expect(event.victim_name).toBe("P08");
});
