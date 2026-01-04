// frontend/src/components/Dashboard/DuelAnalysisPanel.jsx
// Panel showing per-callout duel analysis and weapon performance

import React, { useMemo } from 'react';
import { 
  Swords, Clock, Crosshair, Target, Trophy, Skull,
  TrendingUp, TrendingDown, Minus, ChevronRight
} from 'lucide-react';
import '../../styles/Dashboard/duelAnalysisPanel.css';

/**
 * Get trend indicator based on value
 */
const getTrendIcon = (value, baseline = 50) => {
  if (value > baseline + 5) return { icon: TrendingUp, color: '#22c55e' };
  if (value < baseline - 5) return { icon: TrendingDown, color: '#ef4444' };
  return { icon: Minus, color: '#64748b' };
};

/**
 * Individual callout duel stat row
 */
const CalloutDuelRow = ({ callout, onClick, isSelected }) => {
  const winRate = callout.win_rate || 50;
  const trend = getTrendIcon(winRate);
  const TrendIcon = trend.icon;
  
  return (
    <button 
      className={`callout-duel-row ${isSelected ? 'selected' : ''} ${callout.rating}`}
      onClick={() => onClick(callout)}
    >
      <div className="row-left">
        <span className={`callout-indicator ${callout.rating}`} />
        <span className="callout-name">{callout.name}</span>
      </div>
      
      <div className="row-stats">
        <div className="stat-mini">
          <span className="stat-value">{callout.kills || 0}</span>
          <span className="stat-label">K</span>
        </div>
        <span className="stat-divider">/</span>
        <div className="stat-mini">
          <span className="stat-value">{callout.deaths || 0}</span>
          <span className="stat-label">D</span>
        </div>
        
        <div className="stat-winrate" style={{ color: trend.color }}>
          <TrendIcon size={14} />
          <span>{winRate.toFixed(0)}%</span>
        </div>
      </div>
      
      <ChevronRight size={14} className="row-arrow" />
    </button>
  );
};

/**
 * Detailed metrics for selected callout
 */
const DuelDetailCard = ({ callout, onClose }) => {
  if (!callout) return null;
  
  const winRate = callout.win_rate || 50;
  const trend = getTrendIcon(winRate);
  
  return (
    <div className={`duel-detail-card ${callout.rating}`}>
      <div className="detail-header">
        <div className="header-left">
          <Target size={18} />
          <h4>{callout.name}</h4>
        </div>
        <button className="close-btn" onClick={onClose}>×</button>
      </div>
      
      <div className="detail-stats-grid">
        {/* Main stats */}
        <div className="stat-card primary">
          <Swords size={18} />
          <div className="stat-content">
            <span className="stat-value">{callout.kd?.toFixed(2) || '0.00'}</span>
            <span className="stat-label">K/D Ratio</span>
          </div>
        </div>
        
        <div className="stat-card" style={{ borderColor: trend.color }}>
          <Trophy size={18} style={{ color: trend.color }} />
          <div className="stat-content">
            <span className="stat-value" style={{ color: trend.color }}>
              {winRate.toFixed(0)}%
            </span>
            <span className="stat-label">Win Rate</span>
          </div>
        </div>
        
        {/* Time to damage */}
        {callout.avg_time_to_damage && (
          <div className="stat-card">
            <Clock size={18} />
            <div className="stat-content">
              <span className="stat-value">{callout.avg_time_to_damage}ms</span>
              <span className="stat-label">Tiempo a Daño</span>
            </div>
          </div>
        )}
        
        {/* Crosshair error */}
        {callout.avg_crosshair_error && (
          <div className="stat-card">
            <Crosshair size={18} />
            <div className="stat-content">
              <span className="stat-value">{callout.avg_crosshair_error}°</span>
              <span className="stat-label">Error Crosshair</span>
            </div>
          </div>
        )}
      </div>
      
      {/* Opening vs Retake */}
      {(callout.opening_duel_wr !== null || callout.retake_wr !== null) && (
        <div className="duel-breakdown">
          <h5>Tipo de Duelo</h5>
          <div className="breakdown-row">
            {callout.opening_duel_wr !== null && (
              <div className="breakdown-item">
                <span className="breakdown-label">Opening</span>
                <span className={`breakdown-value ${callout.opening_duel_wr >= 50 ? 'good' : 'bad'}`}>
                  {callout.opening_duel_wr?.toFixed(0) || 0}%
                </span>
              </div>
            )}
            {callout.retake_wr !== null && (
              <div className="breakdown-item">
                <span className="breakdown-label">Retake</span>
                <span className={`breakdown-value ${callout.retake_wr >= 50 ? 'good' : 'bad'}`}>
                  {callout.retake_wr?.toFixed(0) || 0}%
                </span>
              </div>
            )}
          </div>
        </div>
      )}
      
      {/* Flash deaths */}
      {callout.flash_death_pct > 0 && (
        <div className="flash-warning">
          <Skull size={14} />
          <span>
            {callout.flash_death_pct.toFixed(0)}% de tus muertes aquí fueron mientras estabas cegado
          </span>
        </div>
      )}
    </div>
  );
};

/**
 * Summary stats header
 */
const DuelSummary = ({ callouts }) => {
  const stats = useMemo(() => {
    if (!callouts || callouts.length === 0) return null;
    
    const totalKills = callouts.reduce((sum, c) => sum + (c.kills || 0), 0);
    const totalDeaths = callouts.reduce((sum, c) => sum + (c.deaths || 0), 0);
    const overallKD = totalDeaths > 0 ? (totalKills / totalDeaths).toFixed(2) : totalKills.toFixed(2);
    
    // Best and worst zones
    const sorted = [...callouts].filter(c => (c.kills + c.deaths) >= 3).sort((a, b) => b.win_rate - a.win_rate);
    const best = sorted[0];
    const worst = sorted[sorted.length - 1];
    
    // Average TTD
    const withTTD = callouts.filter(c => c.avg_time_to_damage);
    const avgTTD = withTTD.length > 0 
      ? Math.round(withTTD.reduce((sum, c) => sum + c.avg_time_to_damage, 0) / withTTD.length)
      : null;
    
    return { totalKills, totalDeaths, overallKD, best, worst, avgTTD };
  }, [callouts]);
  
  if (!stats) return null;
  
  return (
    <div className="duel-summary">
      <div className="summary-row">
        <div className="summary-stat">
          <span className="value">{stats.overallKD}</span>
          <span className="label">K/D Global</span>
        </div>
        <div className="summary-stat">
          <span className="value">{stats.totalKills}</span>
          <span className="label">Kills</span>
        </div>
        <div className="summary-stat">
          <span className="value">{stats.totalDeaths}</span>
          <span className="label">Deaths</span>
        </div>
        {stats.avgTTD && (
          <div className="summary-stat">
            <span className="value">{stats.avgTTD}ms</span>
            <span className="label">Avg TTD</span>
          </div>
        )}
      </div>
      
      {stats.best && stats.worst && stats.best.name !== stats.worst.name && (
        <div className="summary-zones">
          <div className="zone-badge good">
            <Trophy size={12} />
            <span>{stats.best.name}</span>
            <span className="zone-wr">{stats.best.win_rate?.toFixed(0)}%</span>
          </div>
          <div className="zone-badge bad">
            <Skull size={12} />
            <span>{stats.worst.name}</span>
            <span className="zone-wr">{stats.worst.win_rate?.toFixed(0)}%</span>
          </div>
        </div>
      )}
    </div>
  );
};

/**
 * DuelAnalysisPanel - Main component for duel analysis
 */
const DuelAnalysisPanel = ({ 
  calloutStats = {}, 
  onCalloutClick 
}) => {
  const [selectedCallout, setSelectedCallout] = React.useState(null);
  
  // Convert calloutStats object to sorted array
  const sortedCallouts = useMemo(() => {
    return Object.entries(calloutStats)
      .map(([name, stats]) => ({ name, ...stats }))
      .filter(c => (c.kills || 0) + (c.deaths || 0) >= 2) // Min 2 duels
      .sort((a, b) => ((b.kills || 0) + (b.deaths || 0)) - ((a.kills || 0) + (a.deaths || 0)));
  }, [calloutStats]);
  
  const handleRowClick = (callout) => {
    setSelectedCallout(prev => prev?.name === callout.name ? null : callout);
    if (onCalloutClick) {
      onCalloutClick(callout.name);
    }
  };
  
  if (sortedCallouts.length === 0) {
    return (
      <div className="duel-panel empty">
        <Swords size={32} opacity={0.3} />
        <p>No hay suficientes datos de duelos.</p>
        <p className="hint">Juega más partidas para ver análisis detallado.</p>
      </div>
    );
  }
  
  return (
    <div className="duel-panel">
      {/* Summary */}
      <DuelSummary callouts={sortedCallouts} />
      
      {/* Selected callout detail */}
      {selectedCallout && (
        <DuelDetailCard 
          callout={selectedCallout} 
          onClose={() => setSelectedCallout(null)} 
        />
      )}
      
      {/* Callout list */}
      <div className="duel-list">
        <h5 className="list-title">Performance por Zona</h5>
        {sortedCallouts.map(callout => (
          <CalloutDuelRow
            key={callout.name}
            callout={callout}
            onClick={handleRowClick}
            isSelected={selectedCallout?.name === callout.name}
          />
        ))}
      </div>
    </div>
  );
};

export default DuelAnalysisPanel;
