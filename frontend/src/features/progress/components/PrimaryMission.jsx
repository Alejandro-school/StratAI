import React from 'react';
import { motion, useReducedMotion } from 'framer-motion';
import { BrainCircuit, Crosshair, ScanSearch, Sparkles } from 'lucide-react';
import { formatMetricValue, getMissionPoints, getMissionProgress } from '../progressModel';

const MetricNode = ({ label, mission, value, variant }) => (
  <div className={`op-trajectory__node op-trajectory__node--${variant}`}>
    <span>{label}</span>
    <strong>{formatMetricValue(value, mission.unit)}</strong>
  </div>
);

const PrimaryMission = ({ mission, onOpenEvidence }) => {
  const shouldReduceMotion = useReducedMotion();
  const progress = getMissionProgress(mission);
  const points = getMissionPoints(mission);

  return (
    <motion.article
      className="op-primary"
      initial={shouldReduceMotion ? false : { opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.45, ease: [0.16, 1, 0.3, 1] }}
    >
      <div className="op-primary__scan" aria-hidden="true" />
      <div className="op-primary__topline">
        <span className="op-kicker"><BrainCircuit size={14} aria-hidden="true" /> Prioridad IA · {mission.category}</span>
        <span className="op-points"><Sparkles size={13} aria-hidden="true" /> {points}/{mission.maxPoints} pts</span>
      </div>

      <div className="op-primary__brief">
        <div className="op-primary__index" aria-hidden="true">01</div>
        <div>
          <h2>{mission.title}</h2>
          <p>{mission.diagnosis}</p>
        </div>
      </div>

      <div className="op-evidence-chip">
        <ScanSearch size={15} aria-hidden="true" />
        <span>Diagnóstico basado en <strong>{mission.evidenceMatches} partidas</strong></span>
      </div>

      <div
        className="op-trajectory"
        style={{ '--mission-progress': `${progress * 100}%` }}
        aria-label={`${Math.round(progress * 100)}% de progreso en la misión principal`}
      >
        <div className="op-trajectory__line" aria-hidden="true">
          <motion.span
            initial={shouldReduceMotion ? false : { width: 0 }}
            animate={{ width: `${progress * 100}%` }}
            transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
          />
        </div>
        <MetricNode label="Punto de partida" mission={mission} value={mission.baseline} variant="baseline" />
        <MetricNode label="Ahora" mission={mission} value={mission.current} variant="current" />
        <MetricNode label="Objetivo" mission={mission} value={mission.target} variant="target" />
      </div>

      <footer className="op-primary__footer">
        <div className="op-coach-note">
          <Crosshair size={17} aria-hidden="true" />
          <span><b>Próxima decisión:</b> {mission.coachNote}</span>
        </div>
        <button type="button" className="op-primary-action" onClick={() => onOpenEvidence(mission.id)}>
          Ver por qué <ScanSearch size={15} aria-hidden="true" />
        </button>
      </footer>
    </motion.article>
  );
};

export default PrimaryMission;
