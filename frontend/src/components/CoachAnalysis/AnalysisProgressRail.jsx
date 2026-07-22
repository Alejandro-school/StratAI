import React from 'react';
import { Check, ChevronDown, Loader2, RotateCcw, ScanLine } from 'lucide-react';

const getPhaseState = (phaseIndex, activeIndex, status) => {
  if (status === 'complete' || phaseIndex < activeIndex) return 'complete';
  if (phaseIndex === activeIndex) return 'active';
  return 'pending';
};

const PhaseIcon = ({ state }) => {
  if (state === 'complete') return <Check size={13} />;
  if (state === 'active') return <Loader2 size={13} className="spin" />;
  return <ScanLine size={13} />;
};

const AnalysisProgressRail = ({
  phases,
  activePhaseId,
  progress,
  status,
  evidenceCount,
  onReset
}) => {
  const activeIndex = Math.max(0, phases.findIndex((phase) => phase.id === activePhaseId));
  const statusLabel = status === 'complete' ? 'Análisis completo' : 'Analizando demo';

  const activePhase = phases[activeIndex];

  return (
    <aside className="analysis-progress-rail" aria-label="Estado del análisis">
      <div className="analysis-rail-summary">
        <div className="analysis-status-copy">
          <span className="analysis-kicker">Análisis IA</span>
          <strong>{statusLabel}</strong>
          <span>{status === 'complete' ? `${evidenceCount} hallazgos priorizados` : activePhase?.title}</span>
        </div>

        <div className="analysis-progress-cluster">
          <div
            className="analysis-progress-track"
            role="progressbar"
            aria-label="Progreso del análisis"
            aria-valuemin="0"
            aria-valuemax="100"
            aria-valuenow={progress}
          >
            <span style={{ width: `${progress}%` }} />
          </div>
          <span className="analysis-progress-value">{progress}%</span>
        </div>

        <details className="analysis-phase-disclosure">
          <summary>
            Ver proceso
            <ChevronDown size={14} aria-hidden="true" />
          </summary>
          <ol className="analysis-phase-list">
            {phases.map((phase, index) => {
              const phaseState = getPhaseState(index, activeIndex, status);

              return (
                <li key={phase.id} className={`analysis-phase ${phaseState}`}>
                  <div className="phase-marker">
                    <PhaseIcon state={phaseState} />
                  </div>
                  <div>
                    <strong>{phase.title}</strong>
                    <p>{phase.detail}</p>
                  </div>
                </li>
              );
            })}
          </ol>
        </details>

        <button type="button" className="analysis-reset-btn" onClick={onReset}>
          <RotateCcw size={14} aria-hidden="true" />
          Cambiar partida
        </button>
      </div>

      <span className="sr-only" aria-live="polite">
        {statusLabel}. {progress}% completado. {evidenceCount} hallazgos disponibles.
      </span>
    </aside>
  );
};

export default AnalysisProgressRail;
