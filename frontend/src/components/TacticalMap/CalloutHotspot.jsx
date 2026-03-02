import React from 'react';

const CalloutHotspot = ({ callout, isSelected, isBest, isWorst, onClick }) => {
  const { name, position, rating, win_rate: winRate, kd } = callout;

  if (!position) return null;

  const ratingClass = rating === 'good' ? 'good' : rating === 'bad' ? 'bad' : 'neutral';

  return (
    <button
      className={`callout-hotspot compact ${ratingClass} ${isSelected ? 'selected' : ''} ${isWorst ? 'worst' : ''} ${isBest ? 'best' : ''}`}
      style={{ left: `${position.x}%`, top: `${position.y}%` }}
      onClick={() => onClick(callout)}
    >
      <div className="hotspot-ring"></div>
      <div className="hotspot-core">
        <span className="hotspot-label">{name.slice(0, 4)}</span>
      </div>
      <div className="hotspot-tooltip">
        <span className="tooltip-name">{name}</span>
        <div className="tooltip-stats">
          <span className={`stat-wr ${ratingClass}`}>{winRate}%</span>
          <span className="stat-divider">•</span>
          <span className="stat-kd">{kd} K/D</span>
        </div>
      </div>
    </button>
  );
};

export default CalloutHotspot;
