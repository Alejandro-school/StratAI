import React from 'react';
import { ArrowRight, RefreshCcw, Shield, Zap } from 'lucide-react';
import {
  canRerollMission,
  formatMetricValue,
  getMissionPoints,
  getMissionProgress,
} from '../progressModel';

const SupportMission = ({ hasReroll, mission, onOpenEvidence, onRequestReroll }) => {
  const progress = getMissionProgress(mission);
  const canReroll = canRerollMission(mission, hasReroll);

  return (
    <article className="op-support-card">
      <div className="op-support-card__rail" aria-hidden="true"><Zap size={14} /></div>
      <div className="op-support-card__content">
        <header>
          <span>{mission.category}</span>
          <strong>+{mission.maxPoints} máx.</strong>
        </header>
        <h3>{mission.title}</h3>
        <div className="op-support-card__metric">
          <span>{mission.metricLabel}</span>
          <b>{formatMetricValue(mission.current, mission.unit)} <i>/ {formatMetricValue(mission.target, mission.unit)}</i></b>
        </div>
        <div className="op-support-card__progress" aria-label={`${Math.round(progress * 100)}% completado`}>
          <span style={{ width: `${progress * 100}%` }} />
        </div>
        <footer>
          <span>{getMissionPoints(mission)} pts ganados</span>
          <div>
            {canReroll ? (
              <button type="button" onClick={() => onRequestReroll(mission)}>
                <RefreshCcw size={13} aria-hidden="true" /> Cambiar
              </button>
            ) : null}
            <button type="button" onClick={() => onOpenEvidence(mission.id)}>
              Evidencia <ArrowRight size={13} aria-hidden="true" />
            </button>
          </div>
        </footer>
      </div>
    </article>
  );
};

const SupportMissions = (props) => (
  <section className="op-support" aria-labelledby="support-title">
    <div className="op-section-heading">
      <div>
        <span className="op-eyebrow"><Shield size={13} aria-hidden="true" /> Refuerzos</span>
        <h2 id="support-title">2 misiones que sostienen tu mejora</h2>
      </div>
      <span className="op-reroll-count">{props.hasReroll ? '1 cambio disponible' : 'Cambio utilizado'}</span>
    </div>
    <div className="op-support__grid">
      {props.missions.map((mission) => <SupportMission key={mission.id} mission={mission} {...props} />)}
    </div>
  </section>
);

export default SupportMissions;
