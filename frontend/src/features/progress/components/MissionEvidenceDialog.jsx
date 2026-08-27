import React from 'react';
import { BrainCircuit, Crosshair, ScanLine } from 'lucide-react';
import { formatMetricValue } from '../progressModel';
import ProgressDialog from './ProgressDialog';

const MissionEvidenceDialog = ({ mission, onClose }) => (
  <ProgressDialog
    ariaLabel={`Evidencia de ${mission.title}`}
    eyebrow="Explicabilidad IA"
    title={mission.title}
    onClose={onClose}
  >
    <div className="op-evidence-summary">
      <BrainCircuit size={24} aria-hidden="true" />
      <div><span>Diagnóstico</span><p>{mission.diagnosis}</p></div>
    </div>

    <div className="op-evidence-metric">
      <span>Inicial <b>{formatMetricValue(mission.baseline, mission.unit)}</b></span>
      <span>Actual <b>{formatMetricValue(mission.current, mission.unit)}</b></span>
      <span>Objetivo <b>{formatMetricValue(mission.target, mission.unit)}</b></span>
    </div>

    <section className="op-evidence-list" aria-labelledby="evidence-list-title">
      <h3 id="evidence-list-title"><ScanLine size={15} aria-hidden="true" /> Señales detectadas</h3>
      {mission.evidence.map((item) => (
        <article key={`${item.map}-${item.round}`}>
          <span>{item.map} · Ronda {item.round}</span>
          <strong>{item.note}</strong>
          <small>{item.impact}</small>
        </article>
      ))}
    </section>

    <div className="op-evidence-action">
      <Crosshair size={18} aria-hidden="true" />
      <p><b>Decisión recomendada</b>{mission.coachNote}</p>
    </div>
  </ProgressDialog>
);

export default MissionEvidenceDialog;
