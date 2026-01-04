// frontend/src/components/Dashboard/CalloutDetailPanel.jsx
// Enhanced detail panel for selected callout with weapon stats and context

import React from 'react';
import { 
  Target, Shield, Clock, Crosshair, Eye, Zap,
  TrendingUp, TrendingDown, AlertTriangle, 
  Swords, Wind, Gauge, Users
} from 'lucide-react';
import { generateCalloutInsights } from '../../utils/InsightEngine';
import '../../styles/Dashboard/calloutDetailPanel.css';

/**
 * WeaponStatRow - Shows weapon performance
 */
const WeaponStatRow = ({ weapon, kills, deaths, kd }) => {
  const kdClass = kd >= 1.5 ? 'good' : kd < 0.8 ? 'bad' : 'neutral';
  
  // Generate stars based on K/D (1-4 stars)
  const stars = Math.min(4, Math.max(1, Math.round(kd * 1.5)));
  
  return (
    <div className="weapon-row">
      <span className="weapon-name">{weapon}</span>
      <span className="weapon-kd">
        <span className="kd-kills">{kills}K</span>
        <span className="kd-sep">/</span>
        <span className="kd-deaths">{deaths}D</span>
      </span>
      <span className={`weapon-rating ${kdClass}`}>
        {'★'.repeat(stars)}{'☆'.repeat(4 - stars)}
      </span>
    </div>
  );
};

/**
 * ContextStatItem - Individual context stat
 */
const ContextStatItem = ({ icon: Icon, label, value, subValue }) => (
  <div className="context-stat">
    <Icon size={18} className="context-icon" />
    <div className="context-info">
      <span className="context-value">{value}</span>
      {subValue && <span className="context-sub">{subValue}</span>}
    </div>
    <span className="context-label">{label}</span>
  </div>
);

/**
 * CalloutDetailPanel - Shows detailed stats for selected callout
 */
const CalloutDetailPanel = ({ callout, onClose }) => {
  if (!callout) {
    return (
      <div className="callout-detail-panel empty">
        <div className="empty-state">
          <Target size={32} opacity={0.3} />
          <h4>Selecciona un HotPoint</h4>
          <p>Haz clic en un punto del mapa para ver estadísticas detalladas</p>
        </div>
      </div>
    );
  }
  
  const {
    name,
    kills,
    deaths,
    kd,
    win_rate,
    rating,
    sample_size,
    weapon_stats = [],
    context_stats = {},
    avg_distance,
    ct_t_split = {},
    avg_time_to_damage,
    flash_death_pct
  } = callout;
  
  const insights = generateCalloutInsights(callout, name);
  const ratingClass = rating === 'good' ? 'good' : rating === 'bad' ? 'bad' : 'neutral';
  const RatingIcon = rating === 'good' ? TrendingUp : rating === 'bad' ? TrendingDown : Target;
  
  // Calculate CT/T performance
  const ctKD = ct_t_split.ct_deaths > 0 
    ? (ct_t_split.ct_kills / ct_t_split.ct_deaths).toFixed(2) 
    : ct_t_split.ct_kills || 0;
  const tKD = ct_t_split.t_deaths > 0 
    ? (ct_t_split.t_kills / ct_t_split.t_deaths).toFixed(2) 
    : ct_t_split.t_kills || 0;
  
  // Opening duel stats
  const openingWR = context_stats.opening_attempts > 0 
    ? Math.round((context_stats.opening_kills / context_stats.opening_attempts) * 100)
    : null;
  
  return (
    <div className={`callout-detail-panel v2 ${ratingClass}`}>
      {/* Header */}
      <div className="detail-header">
        <div className="header-main">
          <div className={`rating-indicator ${ratingClass}`}>
            <RatingIcon size={24} />
          </div>
          <div className="header-text">
            <h3 className="callout-name">{name}</h3>
            <span className="sample-size">{sample_size} duelos analizados</span>
          </div>
        </div>
        {onClose && (
          <button className="close-btn" onClick={onClose}>×</button>
        )}
      </div>
      
      {/* Main Stats - Compact */}
      <div className="main-stats-compact">
        <div className="big-stat primary">
          <span className="big-value">{win_rate}%</span>
          <span className="big-label">Win Rate</span>
        </div>
        <div className="big-stat">
          <span className="big-value">{kd}</span>
          <span className="big-label">K/D</span>
        </div>
        <div className="big-stat">
          <span className="big-value kills">{kills}</span>
          <span className="big-label">Kills</span>
        </div>
        <div className="big-stat">
          <span className="big-value deaths">{deaths}</span>
          <span className="big-label">Deaths</span>
        </div>
      </div>
      
      {/* Weapon Performance */}
      {weapon_stats.length > 0 && (
        <div className="panel-section">
          <h5 className="section-title">
            <Swords size={18} />
            Rendimiento por Arma
          </h5>
          <div className="weapon-list">
            {weapon_stats.map((w, idx) => (
              <WeaponStatRow 
                key={idx}
                weapon={w.weapon}
                kills={w.kills}
                deaths={w.deaths}
                kd={w.kd}
              />
            ))}
          </div>
        </div>
      )}
      
      {/* Context Stats */}
      <div className="panel-section">
        <h5 className="section-title">
          <Crosshair size={18} />
          Contexto de Duelos
        </h5>
        <div className="context-grid">
          {openingWR !== null && (
            <ContextStatItem 
              icon={Zap}
              label="First Kills"
              value={`${context_stats.opening_kills}/${context_stats.opening_attempts}`}
              subValue={`${openingWR}% WR`}
            />
          )}
          {(context_stats.trade_kills > 0 || context_stats.trade_deaths > 0) && (
            <ContextStatItem 
              icon={Users}
              label="Trades"
              value={`${context_stats.trade_kills}K / ${context_stats.trade_deaths}D`}
            />
          )}
          {context_stats.smoke_kills > 0 && (
            <ContextStatItem 
              icon={Wind}
              label="Smoke Kills"
              value={context_stats.smoke_kills}
            />
          )}
          {avg_time_to_damage && (
            <ContextStatItem 
              icon={Clock}
              label="TTD"
              value={`${Math.round(avg_time_to_damage)}ms`}
            />
          )}
          {avg_distance && (
            <ContextStatItem 
              icon={Gauge}
              label="Distancia"
              value={`${Math.round(avg_distance)}u`}
            />
          )}
          {flash_death_pct > 0 && (
            <ContextStatItem 
              icon={Eye}
              label="Flash Deaths"
              value={`${flash_death_pct}%`}
            />
          )}
        </div>
      </div>
      
      {/* Insight - Just 1 actionable */}
      {insights.length > 0 && (
        <div className="callout-insight-single">
          <AlertTriangle size={18} style={{ color: insights[0].type.color }} />
          <p>{insights[0].text}</p>
        </div>
      )}
    </div>
  );
};

export default CalloutDetailPanel;
