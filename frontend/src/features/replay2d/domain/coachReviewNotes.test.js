import { describe, it, expect } from "vitest";
import { loadReviewNotes, makeReviewNote, reviewPacket, reviewStorageKey, saveReviewNotes } from "./coachReviewNotes";
const bundle = { case_id: "E06", round: 9, cut_seconds: 81.1875, replay_cut_tick: 5200, instrument: "review@9",
  context_sha256: "a".repeat(64), source_html_sha256: "b".repeat(64), variants: [
    { code: "V2", reading: { evolution: [{ kind: "observed", text: "J4 murió.", evidence: ["E1"] }] } },
  ] };
const create = () => makeReviewNote(bundle, "V2", { section: "evolution", claim_index: 0 }, "J4 sigue vivo.\nLa muerte corresponde a J3.", "Ver el corte.",
  { round: 9, tick: 5210, side: "T" }, { id: "note1", createdAt: "2026-10-06T17:00:00Z" });
describe("assisted coach correction notes", () => {
  it("preserves literal corrections and the original claim without treating either as training or independent labels", () => {
    const original = JSON.stringify(bundle), note = create(), packet = reviewPacket(bundle, [note]);
    expect(JSON.stringify(bundle)).toBe(original); expect(note.target.original_claim.text).toBe("J4 murió.");
    expect(note.corrected_text).toContain("\n"); expect(note.reviewed_replay_point.tick).toBe(5210);
    expect(packet.independent_evaluation).toBe(false); expect(packet.training_authorized).toBe(false);
    expect(packet.binding.context_sha256).toBe(bundle.context_sha256);
  });
  it("restores matching notes and keeps different sources/versions separate", () => {
    const values = new Map(), storage = { getItem: (key) => values.get(key), setItem: (key, value) => values.set(key, value) };
    expect(saveReviewNotes(bundle, [create()], storage)).toBeNull();
    expect(loadReviewNotes(bundle, storage).notes).toHaveLength(1);
    expect(loadReviewNotes({ ...bundle, context_sha256: "c".repeat(64) }, storage).notes).toEqual([]);
    const wrong = reviewPacket(bundle, [create()]); wrong.notes[0].target.original_claim.text = "A different original";
    storage.setItem(reviewStorageKey(bundle), JSON.stringify(wrong));
    expect(loadReviewNotes(bundle, storage).error).toBeTruthy();
  });
  it("reports storage failures and corrupt records, rather than pretending a note was persisted", () => {
    const failing = { getItem() { throw Error(); }, setItem() { throw Error(); } };
    expect(loadReviewNotes(bundle, failing).error).toBeTruthy();
    expect(saveReviewNotes(bundle, [create()], failing)).toMatch(/No se ha podido guardar/);
    expect(loadReviewNotes(bundle, { getItem: () => "broken-json" }).error).toBeTruthy();
    expect(() => makeReviewNote(bundle, "V2", { section: "evolution", claim_index: 99 }, "Correct", "", null, {})).toThrow();
  });
});
