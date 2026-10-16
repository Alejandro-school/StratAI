import { render, screen, fireEvent } from "@testing-library/react";
import { it, expect, vi } from "vitest";
import { ReplayCoachReading } from "./ReplayCoachReading";
const bundle = { case_id: "E06", round: 9, cut_seconds: 81.1875, replay_cut_tick: 5200, default_variant: "V2",
  variants: [{ code: "V2", reading: { situation: [{ kind: "observed", text: "Dos T siguen vivos.", evidence: ["F1:2"] }],
    interpretation: [{ kind: "hypothesis", text: "Pueden reunirse para apoyarse." }] } }] };
it("does not label a rejected repaired-context attempt as the old uncorrected input", () => {
  render(<ReplayCoachReading bundle={{...bundle,default_variant:'C-IG',review_supplement:{
    corrected_objective_not_given_to_original_model:true,corrected_places_not_given_to_original_model:true},
    variants:[{code:'C-IG',attempt_input_version:'stratai.replay_context_input@8',reading:null,failure:'invalid finish envelope'}]}}
    context={{round:9,tick:5200}} />);
  expect(screen.getByText(/INTENTO CON CONTEXTO REPARADO/)).toBeInTheDocument();
  expect(screen.getByText(/No hay una lectura aceptada/)).toBeInTheDocument();
  expect(screen.queryByText(/La IA no recibió las zonas/)).not.toBeInTheDocument();
});
it("distinguishes original readings from the corrected pair of C4 and places", () => {
  render(<ReplayCoachReading bundle={{ ...bundle, input_mode: "enriched_replay", review_supplement: {
    corrected_objective_not_given_to_original_model: true, corrected_places_not_given_to_original_model: true,
  } }} context={{ round: 9, tick: 5200 }} />);
  expect(screen.getByText(/La IA no recibió las zonas ni la corrección del C4/)).toBeInTheDocument();
  expect(screen.queryByText(/LECTURA ACTUALIZADA/)).not.toBeInTheDocument();
});
it("keeps original readings explicitly separate from repaired C4 facts", () => {
  render(<ReplayCoachReading bundle={{...bundle,input_mode:"enriched_replay",
    review_supplement:{corrected_objective_not_given_to_original_model:true}}} context={{round:9,tick:5200}}/>);
  expect(screen.getByText(/La IA no recibió la corrección del C4/)).toBeInTheDocument();
  expect(screen.queryByText(/LECTURA ACTUALIZADA/)).not.toBeInTheDocument();
  expect(screen.getByText("Dos T siguen vivos.")).toBeInTheDocument();
});
it("shows the original natural-language reading next to a labelled historical cut and seeks on request", () => {
  const go = vi.fn(); render(<ReplayCoachReading bundle={bundle} context={{ round: 9, tick: 5000, side: "T" }} onGoToCut={go} />);
  expect(screen.getByText("Dos T siguen vivos.")).toBeInTheDocument();
  expect(screen.getByText("Pueden reunirse para apoyarse.")).toBeInTheDocument();
  expect(screen.getByText(/Estás viendo otro momento/)).toBeInTheDocument();
  expect(screen.getByText(/La IA no recibió los contactos/)).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Ir al momento analizado" })); expect(go).toHaveBeenCalledOnce();
});
it("does not present a T reading as an analysis of CT or fabricate a missing reading", () => {
  const { rerender } = render(<ReplayCoachReading bundle={bundle} context={{ round: 9, tick: 5200, side: "CT" }} />);
  expect(screen.getByText(/No es un análisis de la defensa CT/)).toBeInTheDocument();
  rerender(<ReplayCoachReading />); expect(screen.getByText(/No hay una lectura/)).toBeInTheDocument();
});
it("identifies an enriched input without attributing it to an older reading", () => {
  render(<ReplayCoachReading bundle={{ ...bundle, input_mode: "enriched_replay" }} context={{round:9,tick:5200,side:"T"}} />);
  expect(screen.getByText(/La IA recibió la situación al corte/)).toBeInTheDocument();
  expect(screen.queryByText(/La IA no recibió los contactos/)).not.toBeInTheDocument();
  expect(screen.getByText(/LECTURA ACTUALIZADA/)).toBeInTheDocument();
});
it("makes the seek state explicit and explains post-cut replay exposure", () => {
  const play=vi.fn(), go=vi.fn();
  const {rerender}=render(<ReplayCoachReading bundle={bundle} context={{round:9,tick:5200}} onGoToCut={go} onPlayFromStart={play}/>);
  expect(screen.getByRole("button",{name:"Estás en el momento analizado"})).toBeDisabled();
  expect(screen.queryByRole("combobox",{name:"Lectura del coach"})).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button",{name:"Reproducir ronda desde el inicio"}));
  expect(play).toHaveBeenCalledOnce();
  rerender(<ReplayCoachReading bundle={bundle} context={{round:9,tick:5300}} onGoToCut={go}/>);
  expect(screen.getByText(/Estás viendo información posterior al corte/)).toHaveAttribute("role","status");
  fireEvent.click(screen.getByRole("button",{name:"Ir al momento analizado"}));
  expect(go).toHaveBeenCalledOnce();
});
