import { useEffect, useRef, useState } from "react";
import { loadReviewNotes, makeReviewNote, reviewPacket, reviewStorageKey, saveReviewNotes } from "../domain/coachReviewNotes";
import { labelCoachText } from "../domain/replayPlayerLabels";

const TITLES = { general: "Lectura completa", situation: "Qué está pasando", evolution: "Cómo ha evolucionado",
  interpretation: "Qué deduce la IA", uncertainty: "Qué falta saber" };
function browserStorage() { return window.localStorage; }

export function ReplayCoachCorrections({ bundle, variant, context, target, onResetTarget, focusRequest = 0, playerLabels = null }) {
  const textRef = useRef(null);
  useEffect(() => {
    if (!focusRequest) return;
    textRef.current?.scrollIntoView?.({ block: "center" });
    textRef.current?.focus({ preventScroll: true });
  }, [focusRequest]);
  const [stored] = useState(() => {
    try { return loadReviewNotes(bundle, browserStorage()); }
    catch { return { notes: [], error: "El navegador no permite guardar correcciones. Puedes descargarlas al archivo." }; }
  });
  const [notes, setNotes] = useState(stored.notes);
  const [drafts, setDrafts] = useState({});
  const [message, setMessage] = useState(stored.error || "");
  const draftKey = `${variant.code}:${target.section}:${target.claim_index}`;
  const { corrected = "", reason = "" } = drafts[draftKey] || {};
  const updateDraft = (field, value) => setDrafts((current) => ({ ...current, [draftKey]: { ...current[draftKey], [field]: value } }));
  const claim = target.section === "general" ? null : variant.reading?.[target.section]?.[target.claim_index];
  const validBinding = Boolean(reviewStorageKey(bundle));
  const variantNotes = notes.filter((note) => note.target.variant_code === variant.code);
  function download() {
    const url = URL.createObjectURL(new Blob([JSON.stringify(reviewPacket(bundle, notes), null, 2)], { type: "application/json" }));
    const anchor = document.createElement("a"); anchor.href = url; anchor.download = `StratAI-correcciones-${bundle.case_id}.json`;
    anchor.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
  return <section className="r2-corrections" aria-label="Tus correcciones">
    <h3>Tus correcciones</h3>
    <p>Escribe qué está mal, qué falta o cómo interpretarías la situación. Se conserva lo que dijo la IA.</p>
    <p className="r2-reading-note">Se guardan en este navegador. Descarga el archivo para conservarlas y revisarlas en el proyecto.
      Guardarlas no entrena el modelo ni cambia esta lectura.</p>
    <form onSubmit={(event) => {
      event.preventDefault();
      if (!corrected.trim() || !validBinding || (target.section !== "general" && !claim)) return;
      const note = makeReviewNote(bundle, variant.code, target, corrected, reason, context,
        { id: crypto.randomUUID(), createdAt: new Date().toISOString() });
      const next = [...notes, note]; setNotes(next);
      let error = stored.error;
      if (!error) { try { error = saveReviewNotes(bundle, next, browserStorage()); } catch { error = "No se ha podido guardar. Descarga las correcciones para conservarlas."; } }
      setMessage(error ? `${error} Descarga las correcciones nuevas para conservarlas.` : "Corrección guardada en este navegador.");
      setDrafts((current) => ({ ...current, [draftKey]: {} }));
    }}>
      <p><strong>{variant.code} · {TITLES[target.section]}</strong></p>
      {claim && <blockquote>{labelCoachText(claim.text, playerLabels)}</blockquote>}
      {target.section !== "general" && <button type="button" onClick={onResetTarget}>Cambiar a corrección general</button>}
      <label htmlFor="coach-correction-text">Tu corrección o lectura</label>
      <textarea ref={textRef} id="coach-correction-text" value={corrected} onChange={(event) => updateDraft("corrected", event.target.value)} required rows={4}
        placeholder="Por ejemplo: T4 sigue vivo en este corte; esa muerte pertenece a otro jugador." />
      <label htmlFor="coach-correction-reason">Por qué / qué información lo respalda (opcional)</label>
      <textarea id="coach-correction-reason" value={reason} onChange={(event) => updateDraft("reason", event.target.value)} rows={3}
        placeholder="Indica lo que ves, el momento o la información que falta." />
      <button type="submit" disabled={!validBinding || !corrected.trim()}>Guardar corrección</button>
    </form>
    {message && <p role="status">{message}</p>}
    {!validBinding && <p>Falta una referencia de la lectura; no se pueden vincular correcciones.</p>}
    <div className="r2-corrections-actions"><strong>{notes.length} correcciones en este caso</strong>
      <button type="button" onClick={download} disabled={!notes.length}>Descargar correcciones</button></div>
    <p className="r2-reading-note">Se muestran las correcciones de {variant.code}. Cada una conserva la frase original, su referencia y el momento que estabas revisando.</p>
    {variantNotes.map((note) => <article className="r2-review-note" key={note.id}>
      <strong>{TITLES[note.target.section]}</strong>
      {note.target.original_claim && <details><summary>Frase original de la IA</summary><p>{note.target.original_claim.text}</p></details>}
      <p>{note.corrected_text}</p>{note.reason && <p><b>Motivo:</b> {note.reason}</p>}
    </article>)}
  </section>;
}
