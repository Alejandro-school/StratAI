export const COACH_REVIEW_SCHEMA = "stratai.coach_review_notes@1";
const HASH = /^[a-f0-9]{64}$/;
export const REVIEW_SECTIONS = ["general", "situation", "evolution", "interpretation", "uncertainty"];

export function reviewBinding(bundle) {
  if (!bundle?.case_id || !HASH.test(bundle.context_sha256 || "") || !HASH.test(bundle.source_html_sha256 || "")) return null;
  return { case_id: bundle.case_id, round: bundle.round, side: "T", instrument: bundle.instrument,
    context_sha256: bundle.context_sha256, source_html_sha256: bundle.source_html_sha256,
    historical_cut_seconds: bundle.cut_seconds, nearest_replay_cut_tick: bundle.replay_cut_tick };
}

export function reviewStorageKey(bundle) {
  const binding = reviewBinding(bundle);
  return binding ? `stratai:coach-review:v1:${binding.case_id}:${binding.context_sha256}:${binding.source_html_sha256}` : null;
}

export function reviewPacket(bundle, notes) {
  return { schema_id: COACH_REVIEW_SCHEMA, binding: reviewBinding(bundle),
    review_kind: "assisted_known_development", independent_evaluation: false,
    training_authorized: false, notes };
}

export function loadReviewNotes(bundle, storage) {
  const key = reviewStorageKey(bundle);
  if (!key) return { notes: [], error: "Falta la referencia de la lectura; no se pueden vincular correcciones." };
  try {
    const raw = storage.getItem(key);
    if (!raw) return { notes: [], error: null };
    const packet = JSON.parse(raw);
    if (packet.schema_id !== COACH_REVIEW_SCHEMA || JSON.stringify(packet.binding) !== JSON.stringify(reviewBinding(bundle))
      || packet.review_kind !== "assisted_known_development" || packet.independent_evaluation !== false || packet.training_authorized !== false
      || !Array.isArray(packet.notes) || packet.notes.some((note) => !note.id || !note.created_at
        || !bundle.variants.some((variant) => variant.code === note.target?.variant_code)
        || !REVIEW_SECTIONS.includes(note.target?.section) || typeof note.corrected_text !== "string"
        || !note.corrected_text.trim() || typeof note.reason !== "string" || !noteMatchesReading(bundle, note))) throw new Error("invalid stored notes");
    return { notes: packet.notes, error: null };
  } catch {
    return { notes: [], error: "No se pueden leer las correcciones guardadas. No se sobrescribirá ese registro." };
  }
}

export function makeReviewNote(bundle, variantCode, target, correctedText, reason, context, { id, createdAt }) {
  const variant = bundle.variants.find((row) => row.code === variantCode);
  if (!reviewBinding(bundle) || !variant || !REVIEW_SECTIONS.includes(target.section) || !correctedText.trim()) throw new Error("Invalid correction");
  const claim = target.section === "general" ? null : variant.reading?.[target.section]?.[target.claim_index];
  if (target.section !== "general" && !claim) throw new Error("Missing original claim");
  return { id, created_at: createdAt,
    target: { variant_code: variantCode, section: target.section, claim_index: claim ? target.claim_index : null,
      original_claim: claim ? JSON.parse(JSON.stringify(claim)) : null },
    corrected_text: correctedText, reason,
    reviewed_replay_point: context && Number.isFinite(context.tick) ? { round: context.round, tick: context.tick, side: context.side } : null,
    reviewed_supplement: bundle.review_supplement ? JSON.parse(JSON.stringify(bundle.review_supplement)) : null,
    supplement_exposure: "contact_replay_and_original_reading_available", validation_status: "human_note_not_validated" };
}

function noteMatchesReading(bundle, note) {
  if (note.target.section === "general") return note.target.original_claim === null && note.target.claim_index === null;
  const original = bundle.variants.find((variant) => variant.code === note.target.variant_code)?.reading?.[note.target.section]?.[note.target.claim_index];
  return original && JSON.stringify(original) === JSON.stringify(note.target.original_claim);
}

export function saveReviewNotes(bundle, notes, storage) {
  try { storage.setItem(reviewStorageKey(bundle), JSON.stringify(reviewPacket(bundle, notes))); return null; }
  catch { return "No se ha podido guardar en este navegador. La corrección sigue en esta sesión; descarga el archivo para conservarla."; }
}
