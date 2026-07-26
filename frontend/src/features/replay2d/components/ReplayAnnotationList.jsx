import { Check, Trash2 } from "lucide-react";
import { useState } from "react";

function AnnotationItem({ annotation, onUpdate, onDelete }) {
  const [text, setText] = useState(annotation.text || "");
  return (
    <li>
      <span style={{ "--annotation-color": annotation.color }} />
      <input
        value={text}
        onChange={(event) => setText(event.target.value)}
        onBlur={() => {
          if (text !== annotation.text) onUpdate(annotation.id, { text });
        }}
        aria-label="Texto de anotación"
        placeholder={annotation.type}
      />
      <button type="button" onClick={() => onUpdate(annotation.id, { text })} aria-label="Guardar anotación"><Check size={13} /></button>
      <button type="button" className="danger" onClick={() => onDelete(annotation.id)} aria-label="Eliminar anotación"><Trash2 size={13} /></button>
    </li>
  );
}

export function ReplayAnnotationList({ annotations, isSaving, error, onUpdate, onDelete }) {
  if (!annotations.length && !error) return null;
  return (
    <details className="r2-annotation-list">
      <summary>Anotaciones <span>{annotations.length}</span>{isSaving && <i>guardando…</i>}</summary>
      {error && <p role="alert">{error}</p>}
      <ul>
        {annotations.map((annotation) => (
          <AnnotationItem key={annotation.id} annotation={annotation} onUpdate={onUpdate} onDelete={onDelete} />
        ))}
      </ul>
    </details>
  );
}
