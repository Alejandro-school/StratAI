import React, { useEffect, useState } from 'react';
import {
  AlertTriangle,
  Banknote,
  Clock3,
  Crosshair,
  Flame,
  PlayCircle,
  ShieldAlert
} from 'lucide-react';

const EVIDENCE_ICONS = {
  error: ShieldAlert,
  economy: Banknote,
  utility: Flame,
  duel: Crosshair,
  timing: Clock3
};

const SEVERITY_LABELS = {
  critical: 'Crítica',
  high: 'Alta',
  medium: 'Media'
};

const EvidenceTray = ({
  evidenceList,
  selectedEvidence,
  status,
  onSelectEvidence,
  onPlayEvidence
}) => {
  const [isOpen, setIsOpen] = useState(true);
  const selectedEvidenceId = selectedEvidence?.id;

  useEffect(() => {
    if (selectedEvidenceId) setIsOpen(false);
  }, [selectedEvidenceId]);

  const activeLabel = selectedEvidence
    ? `R${selectedEvidence.round} · ${selectedEvidence.title}`
    : 'Selecciona un momento para revisarlo';

  return (
    <section className="evidence-tray" aria-label="Momentos clave">
      <details className="evidence-disclosure" open={isOpen} onToggle={(event) => setIsOpen(event.currentTarget.open)}>
        <summary>
          <span>
            <span className="analysis-kicker">Momentos clave</span>
            <strong>{activeLabel}</strong>
          </span>
          <span className="evidence-count">{evidenceList.length}</span>
        </summary>

        {evidenceList.length ? (
      <div className="evidence-list">
        {evidenceList.map((evidence) => {
          const Icon = EVIDENCE_ICONS[evidence.type] || AlertTriangle;
          const isSelected = selectedEvidence?.id === evidence.id;

          return (
            <article
              key={evidence.id}
              className={`evidence-card ${isSelected ? 'selected' : ''} severity-${evidence.severity}`}
            >
              <button type="button" className="evidence-card-main" onClick={() => onSelectEvidence(evidence.id)}>
                <div className="evidence-card-icon">
                  <Icon size={16} aria-hidden="true" />
                </div>
                <div>
                  <div className="evidence-card-meta">
                    <span>R{evidence.round}</span>
                    <span>{SEVERITY_LABELS[evidence.severity]}</span>
                  </div>
                  <strong>{evidence.title}</strong>
                  <p>{evidence.impact}</p>
                </div>
              </button>

              <button
                type="button"
                className="evidence-play-btn"
                onClick={() => onPlayEvidence(evidence)}
                disabled={!evidence.interaction}
                title={evidence.interaction ? 'Ver clip' : 'Clip no disponible'}
                aria-label={evidence.interaction ? `Ver momento: ${evidence.title}` : `Clip pendiente: ${evidence.title}`}
              >
                <PlayCircle size={15} aria-hidden="true" />
              </button>
            </article>
          );
        })}
      </div>
    ) : (
      <div className="evidence-empty">
        <AlertTriangle size={20} />
        <strong>{status === 'idle' ? 'Sin análisis activo' : 'Buscando patrones'}</strong>
        <p>Los hallazgos aparecerán aquí en cuanto la IA confirme evidencia útil.</p>
      </div>
    )}
      </details>
    </section>
  );
};

export default EvidenceTray;
