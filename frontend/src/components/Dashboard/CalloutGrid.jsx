// frontend/src/components/Dashboard/CalloutGrid.jsx
// Grid of callout chips showing K/D and rating at a glance

import React from 'react';
import { TrendingUp, TrendingDown, Minus, AlertTriangle, Target } from 'lucide-react';
import '../../styles/Dashboard/calloutGrid.css';

/**
 * Individual callout chip component
 */
const CalloutChip = ({ callout, isSelected, isBest, isWorst, onClick }) => {
  const { name, kills, deaths, kd, win_rate, rating, sample_size } = callout;
  
  const getRatingIcon = () => {
    if (rating === 'good') return <TrendingUp size={12} />;
    if (rating === 'bad') return <TrendingDown size={12} />;
    return <Minus size={12} />;
  };
  
  const handleClick = () => {
    onClick?.(callout);
  };
  
  return (
    <button
      className={`callout-chip ${rating} ${isSelected ? 'selected' : ''} ${isBest ? 'best' : ''} ${isWorst ? 'worst' : ''}`}
      onClick={handleClick}
      title={`${name}: ${kills}K/${deaths}D - ${win_rate}% WR (${sample_size} duelos)`}
    >
      <div className="chip-header">
        <span className="chip-name">{name}</span>
        {isBest && <span className="chip-badge best">★</span>}
        {isWorst && <AlertTriangle size={10} className="chip-badge worst" />}
      </div>
      
      <div className="chip-stats">
        <span className="chip-kd">{kd}</span>
        <span className="chip-icon">{getRatingIcon()}</span>
      </div>
      
      <div className="chip-wr">
        <span className={`wr-value ${rating}`}>{win_rate}%</span>
      </div>
      
      <div className="chip-sample">
        {sample_size} duelos
      </div>
    </button>
  );
};

/**
 * CalloutGrid - Shows all callouts as clickable chips
 * @param {Object} props
 * @param {Array} props.callouts - Array of callout objects with stats
 * @param {string} props.selectedCallout - Currently selected callout name
 * @param {string} props.bestCallout - Name of best performing callout
 * @param {string} props.worstCallout - Name of worst performing callout
 * @param {Function} props.onCalloutSelect - Callback when callout is clicked
 * @param {boolean} props.compact - Use compact view
 */
const CalloutGrid = ({ 
  callouts = [], 
  selectedCallout = null,
  bestCallout = null,
  worstCallout = null,
  onCalloutSelect,
  compact = false
}) => {
  if (!callouts || callouts.length === 0) {
    return (
      <div className="callout-grid-empty">
        <Target size={24} opacity={0.3} />
        <p>No hay datos de callouts disponibles</p>
      </div>
    );
  }
  
  // Sort by sample size (most data first)
  const sortedCallouts = [...callouts].sort((a, b) => b.sample_size - a.sample_size);
  
  return (
    <div className={`callout-grid ${compact ? 'compact' : ''}`}>
      <div className="grid-header">
        <h4 className="grid-title">Rendimiento por Callout</h4>
        <div className="grid-legend">
          <span className="legend-item good">
            <TrendingUp size={12} /> Fuerte
          </span>
          <span className="legend-item neutral">
            <Minus size={12} /> Neutral
          </span>
          <span className="legend-item bad">
            <TrendingDown size={12} /> Débil
          </span>
        </div>
      </div>
      
      <div className="grid-chips">
        {sortedCallouts.map((callout) => (
          <CalloutChip
            key={callout.name}
            callout={callout}
            isSelected={selectedCallout === callout.name}
            isBest={bestCallout?.name === callout.name}
            isWorst={worstCallout?.name === callout.name}
            onClick={onCalloutSelect}
          />
        ))}
      </div>
    </div>
  );
};

export default CalloutGrid;
