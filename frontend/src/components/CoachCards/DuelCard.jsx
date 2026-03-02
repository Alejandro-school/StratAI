import React from 'react';
import { Crosshair } from 'lucide-react';
import ReplayActionButton from './ReplayActionButton';

const DuelCard = ({ card, onOpenReview }) => {
  const {
    title,
    detail,
    reactionMs,
    velocity,
    distance,
    result,
    replayAction
  } = card;

  return (
    <article className="coach-card coach-card-duel">
      <div className="coach-card-header">
        <div className="coach-card-title-wrap">
          <Crosshair size={16} />
          <h4>{title}</h4>
        </div>
        <span className={`coach-card-result result-${(result || 'neutral').toLowerCase()}`}>{result}</span>
      </div>

      <p className="coach-card-detail">{detail}</p>

      <div className="duel-metrics-grid">
        <div><span>Reacción</span><strong>{reactionMs} ms</strong></div>
        <div><span>Velocidad</span><strong>{velocity} u/s</strong></div>
        <div><span>Distancia</span><strong>{distance} m</strong></div>
      </div>

      <ReplayActionButton onClick={() => onOpenReview?.(replayAction)} />
    </article>
  );
};

export default DuelCard;
