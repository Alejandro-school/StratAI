import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ReplayCoachReading } from "./ReplayCoachReading";

function bundle() {
  return {
    case_id: "FactualOutputReview", round: 1, cut_seconds: 15, replay_cut_tick: 7997,
    input_mode: "enriched_replay", default_variant: "protected",
    variants: [
      {
        code: "original",
        reading: {
          situation: [{ kind: "observed", text: "Texto original conservado aunque pueda contener errores.", evidence: ["S240:9"] }],
          interpretation: [{ kind: "inferred", text: "Hipótesis experimental original.", evidence: ["S240:9"] }],
        },
      },
      {
        code: "protected",
        verification: {
          factual_integrity: "checked_reference_rendering",
          semantic_truth: "not_evaluated", tactical_quality: "not_evaluated",
        },
        reading: {
          situation: [{ kind: "observed", text: "P09 está vivo, con 70 de salud.", evidence: ["S240:9"] }],
          interpretation: [{ kind: "limit", text: "Criterio condicional para revisión: la distribución no prueba intención.", evidence: ["S240:9"], principles: ["P02"] }],
          uncertainty: [{ kind: "limit", text: "La zona procede de una referencia local estimada.", evidence: ["S240:10"] }],
        },
      },
    ],
  };
}

const context = { round: 1, tick: 7997, side: "T" };

describe("protected factual reading presentation", () => {
  it("labels reference-rendered facts separately from conditional review criteria", () => {
    render(<ReplayCoachReading bundle={bundle()} context={context} />);
    expect(screen.getByRole("heading", { name: "Criterios para revisar" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Qué deduce la IA" })).not.toBeInTheDocument();
    expect(screen.getByText("Hecho registrado")).toBeInTheDocument();
    expect(screen.getAllByText("Límite o estimación").length).toBeGreaterThan(0);
    expect(screen.queryByText(/Todo el texto de estas cuatro secciones lo genera la IA/)).not.toBeInTheDocument();
    expect(screen.getByText("P09 está vivo, con 70 de salud.")).toBeInTheDocument();
  });

  it("explains the factual guarantee without claiming validated tactical interpretation", () => {
    const { container } = render(<ReplayCoachReading bundle={bundle()} context={context} />);
    const notes = [...container.querySelectorAll(".r2-reading-note")].map((element) => element.textContent).join(" ");
    expect(notes).toMatch(/datos|referencias/i);
    expect(notes).toMatch(/condicional/i);
    expect(notes).toMatch(/no.{0,60}(acredita|valida|evalúa)|sin.{0,60}(acreditar|validar|evaluar)/i);
    expect(notes).not.toMatch(/todo.{0,20}(correcto|verificado)|interpretación táctica (validada|acreditada)/i);
  });

  it("keeps historical prose and labels intact when selecting the original variant", () => {
    const source = bundle();
    const originalReading = JSON.stringify(source.variants[0].reading);
    render(<ReplayCoachReading bundle={source} context={context} />);
    fireEvent.change(screen.getByRole("combobox", { name: "Lectura del coach" }), { target: { value: "original" } });
    expect(screen.getByRole("heading", { name: "Qué deduce la IA" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Criterios para revisar" })).not.toBeInTheDocument();
    expect(screen.getByText("Descripción de la IA")).toBeInTheDocument();
    expect(screen.getByText("Inferencia de la IA")).toBeInTheDocument();
    expect(screen.getByText(/Todo el texto de estas cuatro secciones lo genera la IA/)).toBeInTheDocument();
    expect(screen.getByText("Texto original conservado aunque pueda contener errores.")).toBeInTheDocument();
    expect(JSON.stringify(source.variants[0].reading)).toBe(originalReading);
    fireEvent.change(screen.getByRole("combobox", { name: "Lectura del coach" }), { target: { value: "protected" } });
    expect(screen.getByRole("heading", { name: "Criterios para revisar" })).toBeInTheDocument();
    expect(screen.queryByText(/Todo el texto de estas cuatro secciones lo genera la IA/)).not.toBeInTheDocument();
  });

  it.each([undefined, {}, { factual_integrity: "not_evaluated" }, { citation_integrity: true }])(
    "does not present an unverified variant as protected based on the surrounding bundle: %j", (verification) => {
      const source = bundle();
      source.verification = { factual_integrity: "checked_reference_rendering" };
      source.variants[1].verification = verification;
      render(<ReplayCoachReading bundle={source} context={context} />);
      expect(screen.getByRole("heading", { name: "Qué deduce la IA" })).toBeInTheDocument();
      expect(screen.queryByRole("heading", { name: "Criterios para revisar" })).not.toBeInTheDocument();
      expect(screen.queryByText("Hecho registrado")).not.toBeInTheDocument();
      expect(screen.getByText(/Todo el texto de estas cuatro secciones lo genera la IA/)).toBeInTheDocument();
    },
  );
});
