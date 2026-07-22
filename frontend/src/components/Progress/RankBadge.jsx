import React from 'react';
import { getTierForPoints, TIER_THRESHOLDS } from '../../mocks/progressData';
import '../../styles/Progress/rankBadge.css';

const TIER_ICONS = {
  bronze:  '◆',
  silver:  '◆',
  gold:    '◆',
  diamond: '◇',
};

const RankBadge = ({ points, rank, size = 'default' }) => {
  const tier = getTierForPoints(points);
  const nextTier = TIER_THRESHOLDS.find((t) => t.min > points);
  const progressInTier = nextTier
    ? ((points - tier.min) / (nextTier.min - tier.min)) * 100
    : 100;

  return (
    <div
      className={`rank-badge rank-badge--${tier.id} rank-badge--${size}`}
      style={{ '--tier-color': tier.color, '--tier-glow': tier.glow }}
    >
      <div className="rank-badge__emblem">
        <span className="rank-badge__icon">{TIER_ICONS[tier.id]}</span>
        <span className="rank-badge__rank">#{rank}</span>
      </div>
      <div className="rank-badge__info">
        <span className="rank-badge__tier-label">{tier.label}</span>
        <div className="rank-badge__xp-track">
          <div
            className="rank-badge__xp-fill"
            style={{ width: `${progressInTier}%` }}
          />
        </div>
        <span className="rank-badge__points">
          {points.toLocaleString('es-ES')} pts
          {nextTier && (
            <span className="rank-badge__next"> / {nextTier.min.toLocaleString('es-ES')}</span>
          )}
        </span>
      </div>
    </div>
  );
};

export default RankBadge;
