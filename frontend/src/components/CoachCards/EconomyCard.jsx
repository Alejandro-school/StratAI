import React from 'react';
import { Wallet } from 'lucide-react';
import ReplayActionButton from './ReplayActionButton';

const EconomyCard = ({ card, onOpenReview }) => {
  const { title, detail, ownMoney, enemyMoney, buyResult, buyType, replayAction } = card;
  const max = Math.max(ownMoney || 0, enemyMoney || 0, 1);

  return (
    <article className="coach-card coach-card-economy">
      <div className="coach-card-header">
        <div className="coach-card-title-wrap">
          <Wallet size={16} />
          <h4>{title}</h4>
        </div>
        <span className={`coach-card-result result-${(buyResult || 'neutral').toLowerCase()}`}>
          {buyResult || 'N/A'}
        </span>
      </div>

      <p className="coach-card-detail">{detail}</p>
      {buyType && <span className="coach-card-tag">{buyType}</span>}

      <div className="coach-money-bars">
        <div className="money-row">
          <span>Tu equipo</span>
          <div className="money-bar-track">
            <div className="money-bar-fill own" style={{ width: `${((ownMoney || 0) / max) * 100}%` }} />
          </div>
          <strong>${ownMoney || 0}</strong>
        </div>
        <div className="money-row">
          <span>Rival est.</span>
          <div className="money-bar-track">
            <div className="money-bar-fill enemy" style={{ width: `${((enemyMoney || 0) / max) * 100}%` }} />
          </div>
          <strong>${enemyMoney || 0}</strong>
        </div>
      </div>

      <ReplayActionButton onClick={() => onOpenReview?.(replayAction)} />
    </article>
  );
};

export default EconomyCard;
