import { afterEach, beforeEach, it, expect, vi } from "vitest";
import { fireEvent, render, screen, cleanup } from "@testing-library/react";
import { ReplayCoachReading } from "./ReplayCoachReading";
import { reviewStorageKey } from "../domain/coachReviewNotes";
const bundle = { case_id: "E06", round: 9, cut_seconds: 81, replay_cut_tick: 5200, default_variant: "V2", instrument: "review@9",
  context_sha256: "a".repeat(64), source_html_sha256: "b".repeat(64), variants: [
    { code: "V2", reading: { evolution: [{ kind: "observed", text: "J4 murió.", evidence: ["E1"] }] } },
    { code: "V1", reading: { situation: [{ kind: "observed", text: "Otra lectura." }] } },
  ] };
let storage;
beforeEach(() => {
  const values = new Map();
  storage = { getItem: (key) => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) };
  vi.stubGlobal("localStorage", storage);
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });
it("saves a phrase correction linked to the original variant and restores it after reopening", () => {
  const original = JSON.stringify(bundle);
  const { unmount } = render(<ReplayCoachReading bundle={bundle} context={{ round: 9, tick: 5200, side: "T" }} />);
  fireEvent.click(screen.getByRole("button", { name: "Corregir frase 1 de Cómo ha evolucionado" }));
  fireEvent.change(screen.getByLabelText("Tu corrección o lectura"), { target: { value: "J4 sigue vivo; murió J3." } });
  fireEvent.change(screen.getByLabelText(/Por qué/), { target: { value: "Lo veo en el corte." } });
  fireEvent.click(screen.getByRole("button", { name: "Guardar corrección" }));
  expect(screen.getByRole("status")).toHaveTextContent("guardada");
  const saved = JSON.parse(storage.getItem(reviewStorageKey(bundle)));
  expect(saved.notes[0].target.variant_code).toBe("V2"); expect(saved.notes[0].target.original_claim.text).toBe("J4 murió.");
  expect(saved.notes[0].reviewed_replay_point.tick).toBe(5200); expect(JSON.stringify(bundle)).toBe(original);
  unmount(); render(<ReplayCoachReading bundle={bundle} />);
  expect(screen.getByText("J4 sigue vivo; murió J3.")).toBeInTheDocument();
});
it("separates drafts and corrections when switching variants, and exports the complete provenance", () => {
  const createUrl = vi.fn(() => "blob:notes"), revoke = vi.fn();
  vi.stubGlobal("URL", { createObjectURL: createUrl, revokeObjectURL: revoke });
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
  render(<ReplayCoachReading bundle={bundle} />);
  fireEvent.change(screen.getByLabelText("Tu corrección o lectura"), { target: { value: "Mi lectura V2" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Lectura del coach" }), { target: { value: "V1" } });
  expect(screen.getByLabelText("Tu corrección o lectura")).toHaveValue("");
  fireEvent.change(screen.getByRole("combobox", { name: "Lectura del coach" }), { target: { value: "V2" } });
  expect(screen.getByLabelText("Tu corrección o lectura")).toHaveValue("Mi lectura V2");
  fireEvent.click(screen.getByRole("button", { name: "Guardar corrección" }));
  fireEvent.click(screen.getByRole("button", { name: "Descargar correcciones" }));
  expect(createUrl).toHaveBeenCalledWith(expect.any(Blob));
  vi.unstubAllGlobals();
});
it("keeps a new correction downloadable when browser storage fails", () => {
  vi.spyOn(storage, "setItem").mockImplementation(() => { throw Error("quota"); });
  render(<ReplayCoachReading bundle={bundle} />);
  fireEvent.change(screen.getByLabelText("Tu corrección o lectura"), { target: { value: "Falta información." } });
  fireEvent.click(screen.getByRole("button", { name: "Guardar corrección" }));
  expect(screen.getByRole("status")).toHaveTextContent("No se ha podido guardar");
  expect(screen.getByRole("button", { name: "Descargar correcciones" })).toBeEnabled();
});
it("focuses the requested editor and keeps original player aliases in saved provenance", () => {
  const withAliases = {...bundle, variants:[{code:"V2",reading:{evolution:[{kind:"observed",text:"P01 murió.",evidence:["E1"]}]}}]};
  render(<ReplayCoachReading bundle={withAliases} playerLabels={{1:"T1"}}/>);
  fireEvent.click(screen.getByRole("button",{name:"Añadir corrección general"}));
  expect(screen.getByLabelText("Tu corrección o lectura")).toHaveFocus();
  fireEvent.click(screen.getByRole("button",{name:"Corregir frase 1 de Cómo ha evolucionado"}));
  expect(screen.getByLabelText("Tu corrección o lectura")).toHaveFocus();
  fireEvent.change(screen.getByLabelText("Tu corrección o lectura"),{target:{value:"T1 sigue vivo."}});
  fireEvent.click(screen.getByRole("button",{name:"Guardar corrección"}));
  const saved=JSON.parse(storage.getItem(reviewStorageKey(withAliases)));
  expect(saved.notes[0].target.original_claim.text).toBe("P01 murió.");
  expect(saved.notes[0].corrected_text).toBe("T1 sigue vivo.");
});
