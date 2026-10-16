import { memo, useState } from "react";
import { ReplayCoachCorrections } from "./ReplayCoachCorrections";
import { labelCoachText } from "../domain/replayPlayerLabels";

const SECTIONS = { situation: "Qué está pasando", evolution: "Cómo ha evolucionado", interpretation: "Qué deduce la IA", uncertainty: "Qué falta saber" };
const KINDS = { observed: "Descripción de la IA", inferred: "Inferencia de la IA", inference: "Inferencia de la IA", hypothesis: "Hipótesis de la IA", limit: "Límite declarado" };

export const ReplayCoachReading = memo(function ReplayCoachReading({ bundle, context, onGoToCut, onPlayFromStart, playerLabels = null }) {
  const [selected, setSelected] = useState(bundle?.default_variant || "V2");
  const [target, setTarget] = useState({ section: "general", claim_index: null });
  const [focusRequest, setFocusRequest] = useState(0);
  function openCorrection(next) { setTarget(next); setFocusRequest((value) => value + 1); }
  if (!bundle) return <aside className="r2-reading" aria-label="Análisis del coach"><h2>Análisis del coach</h2>
    <p>No hay una lectura del coach asociada a este replay.</p></aside>;
  const variant = bundle.variants.find((row) => row.code === selected) || bundle.variants[0];
  const protectedFacts = variant.verification?.factual_integrity === "checked_reference_rendering";
  const sections = protectedFacts ? { ...SECTIONS, interpretation: "Criterios para revisar" } : SECTIONS;
  const kinds = protectedFacts ? { observed: "Hecho registrado", limit: "Límite o estimación" } : KINDS;
  const nearCut = context?.round === bundle.round && Math.abs(context.tick - bundle.replay_cut_tick) < 4;
  const afterCut = context?.round === bundle.round && context.tick > bundle.replay_cut_tick;
  const c4Corrected = bundle.review_supplement?.corrected_objective_not_given_to_original_model === true;
  const placesCorrected = bundle.review_supplement?.corrected_places_not_given_to_original_model === true;
  return <aside className="r2-reading" aria-label="Análisis del coach">
    <span className="r2-reading-kicker">{protectedFacts ? "HECHOS CON REFERENCIAS" : variant.attempt_input_version ? "INTENTO CON CONTEXTO REPARADO" : bundle.input_mode === "enriched_replay" && !c4Corrected ? "LECTURA ACTUALIZADA" : "LECTURA ORIGINAL"} · {bundle.case_id} · T</span>
    <h2>Análisis del coach</h2>
    {variant.attempt_input_version && variant.failure
      ? <p className="r2-reading-note">Este intento usa el contexto reparado de esta etapa. No hay una lectura aceptada;
        el fallo se conserva en la comparación. Las lecturas originales siguen separadas en el selector.</p>
      : protectedFacts
      ? <p className="r2-reading-note">Los hechos se redactan a partir de datos y referencias fechadas; la IA puede seleccionar referencias y criterios.
        Los criterios son condicionales: esto no acredita su aplicación ni valida la interpretación táctica.</p>
      : <p className="r2-reading-note">Todo el texto de estas cuatro secciones lo genera la IA. Las descripciones también pueden contener errores;
        «Qué deduce la IA» recoge sus conclusiones e hipótesis. La lectura está pendiente de revisión.</p>}
    <p>Esta lectura se generó para el corte de {bundle.cut_seconds.toFixed(2)} s de la ronda {bundle.round}.
      {nearCut ? " El mapa está en la muestra más cercana a ese corte." : " Estás viendo otro momento del replay."}</p>
    {afterCut && <p className="r2-reading-note" role="status">Estás viendo información posterior al corte. La IA no dispuso de ella para esta lectura.</p>}
    <div className="r2-reading-actions">
      {onPlayFromStart && <button type="button" onClick={onPlayFromStart}>Reproducir ronda desde el inicio</button>}
      <button type="button" disabled={nearCut} onClick={onGoToCut}>{nearCut ? "Estás en el momento analizado" : "Ir al momento analizado"}</button>
      <button type="button" aria-controls="coach-correction-text" onClick={() => openCorrection({ section: "general", claim_index: null })}>Añadir corrección general</button>
    </div>
    {bundle.variants.length > 1 && <label>Lectura <select aria-label="Lectura del coach" value={variant.code} onChange={(event) => { setSelected(event.target.value); setTarget({ section: "general", claim_index: null }); }}>
      {bundle.variants.map((row) => <option key={row.code} value={row.code}>{row.label ? `${row.code} · ${row.label}` : row.code}</option>)}
    </select></label>}
    {protectedFacts
      ? <p className="r2-reading-note">Esta variante usa el contexto reparado hasta el corte indicado. Las lecturas originales y sus correcciones siguen disponibles en el selector.
        La lectura no se recalcula al mover el tiempo.</p>
      : variant.attempt_input_version
      ? <p className="r2-reading-note">La entrada preparada incluye los hechos de esta etapa, con sus estimaciones y límites.
        Un intento rechazado no demuestra cómo los interpretó el modelo.</p>
      : variant.historical_input_version
      ? <p className="r2-reading-note">Intento histórico conservado, con entrada {variant.historical_input_version}. Recibió las funciones de aquella etapa,
        pero todavía permitía texto factual libre. Sus afirmaciones no están verificadas por la protección actual.</p>
      : placesCorrected
      ? <p className="r2-reading-note">La IA no recibió las zonas ni la corrección del C4 que ahora muestra el mapa. Esta respuesta es la original;
        las nuevas lecturas de comparación se conservan por separado.</p>
      : c4Corrected
      ? <p className="r2-reading-note">La IA no recibió la corrección del C4 que ahora muestra el mapa. Esta respuesta conserva sus datos y referencias anteriores;
        no se ha generado de nuevo ni demuestra el efecto de la reparación.</p>
      : bundle.input_mode === "enriched_replay"
      ? <p className="r2-reading-note">La IA recibió la situación al corte y puede consultar la evolución de este replay hasta ese instante,
        incluidos contactos estimados y puntos de muerte comunicados. {playerLabels ? "T1–T5 y CT1–CT5 identifican a los mismos jugadores en mapa y análisis." : "P01–P10 identifican a los mismos jugadores en mapa y análisis."}
        La lectura no se recalcula al mover el tiempo.</p>
      : <p className="r2-reading-note">El análisis conserva su entrada original. La IA no recibió los contactos ni las cruces añadidos ahora.
        Las lecturas no se recalculan al mover el tiempo. Los alias J pertenecen a esa entrada; el mapa usa P/Rival.</p>}
    {context?.side === "CT" && <p className="r2-reading-note">Esta lectura analiza a los T. No es un análisis de la defensa CT.</p>}
    {variant.failure && <p>La generación de esta variante quedó incompleta.</p>}
    {Object.entries(sections).map(([key, title]) => <section key={key}><h3>{title}</h3>
      {(variant.reading?.[key] || []).length ? variant.reading[key].map((claim, index) => <div className="r2-reading-claim" key={index}>
        <small>{kinds[claim.kind] || claim.kind}</small><p>{labelCoachText(claim.text, playerLabels)}</p>
        {labelCoachText(claim.text, playerLabels) !== claim.text && <details><summary>Ver texto original</summary><p>{claim.text}</p></details>}
        <details><summary>Ver referencias originales</summary><p>{claim.evidence?.join(", ") || "Sin referencias"}</p></details>
        <button type="button" className="r2-correct-claim" aria-label={`Corregir frase ${index + 1} de ${title}`}
          onClick={() => openCorrection({ section: key, claim_index: index })}>Corregir esta frase</button>
      </div>) : <p>Esta lectura no contiene afirmaciones en esta parte.</p>}
    </section>)}
    <ReplayCoachCorrections key={`${bundle.case_id}:${bundle.context_sha256}:${bundle.source_html_sha256}`}
      bundle={bundle} variant={variant} context={context} target={target} focusRequest={focusRequest} playerLabels={playerLabels}
      onResetTarget={() => openCorrection({ section: "general", claim_index: null })} />
  </aside>;
});
