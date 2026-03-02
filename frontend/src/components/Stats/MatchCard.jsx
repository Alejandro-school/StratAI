// frontend/src/components/Stats/MatchCard.jsx
// Redesigned match card - compact, professional, modern
import React from 'react';
import { Clock, ChevronRight, Crosshair, Target, Zap, Skull } from 'lucide-react';
import { getMapImage } from '../../utils/mapConfig';

// Format duration
const formatDuration = (seconds) => {
  if (!seconds) return '';
  const mins = Math.floor(seconds / 60);
  return `${mins}m`;
};

// K/D color
const getKDColor = (kd) => {
  if (kd >= 1.5) return 'var(--color-success)';
  if (kd >= 1.0) return 'var(--color-primary-400)';
  if (kd >= 0.8) return 'var(--color-warning)';
  return 'var(--color-danger)';
};

const MatchCard = ({ match, playerStats, onViewDetails, style }) => {
  const isVictory = match.result === 'victory';
  const mapName = match.map_name?.replace('de_', '').charAt(0).toUpperCase() + 
                  match.map_name?.replace('de_', '').slice(1) || 'Unknown';
  
  const kd = playerStats?.kd_ratio || 0;
  const adr = Math.round(playerStats?.adr || 0);
  const hsPercent = Math.round(playerStats?.hs_percentage || 0);
  const kills = playerStats?.kills || 0;
  const deaths = playerStats?.deaths || 0;

  return (
    <div 
      className={`match-card-v2 ${isVictory ? 'victory' : 'defeat'}`}
      onClick={() => onViewDetails(match.match_id)}
      style={style}
    >
      {/* Background image with overlay */}
      <div 
        className="card-bg"
        style={{ backgroundImage: `url(${getMapImage(match.map_name)})` }}
      />
      <div className="card-gradient" />
      
      {/* Result indicator bar */}
      <div className={`result-bar ${isVictory ? 'win' : 'loss'}`} />
      
      {/* Card content */}
      <div className="card-body">
        {/* Header: Map + Result */}
        <div className="card-header">
          <span className="map-name">{mapName}</span>
          <span className={`result-pill ${isVictory ? 'win' : 'loss'}`}>
            {isVictory ? 'WIN' : 'LOSS'}
          </span>
        </div>
        
        {/* Score */}
        <div className="score-row">
          <span className={`score-num ${isVictory ? 'winner' : ''}`}>
            {match.team_score}
          </span>
          <span className="score-divider">-</span>
          <span className={`score-num ${!isVictory ? 'winner' : ''}`}>
            {match.opponent_score}
          </span>
        </div>
        
        {/* Stats row */}
        <div className="stats-row">
          <div className="stat">
            <Crosshair size={12} />
            <span className="stat-value" style={{ color: getKDColor(kd) }}>
              {kd.toFixed(2)}
            </span>
            <span className="stat-label">K/D</span>
          </div>
          <div className="stat">
            <Target size={12} />
            <span className="stat-value">{adr}</span>
            <span className="stat-label">ADR</span>
          </div>
          <div className="stat">
            <Zap size={12} />
            <span className="stat-value">{hsPercent}%</span>
            <span className="stat-label">HS</span>
          </div>
          <div className="stat">
            <Skull size={12} />
            <span className="stat-value">{kills}/{deaths}</span>
            <span className="stat-label">K/D</span>
          </div>
        </div>
        
        {/* Footer: Duration + CTA */}
        <div className="card-footer">
          {match.match_duration > 0 && (
            <span className="duration">
              <Clock size={12} />
              {formatDuration(match.match_duration)}
            </span>
          )}
          <span className="view-cta">
            Ver detalles
            <ChevronRight size={14} />
          </span>
        </div>
      </div>
    </div>
  );
};

export default MatchCard;
