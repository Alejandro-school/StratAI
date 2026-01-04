// frontend/src/components/Dashboard/InsightPanel.jsx
// Panel displaying AI-generated insights

import React, { useMemo } from 'react';
import { AlertTriangle, Target, Crosshair, TrendingUp, Info, ChevronRight } from 'lucide-react';
import { getTopInsights, INSIGHT_TYPES } from '../../utils/InsightEngine';
import '../../styles/Dashboard/insightPanel.css';

/**
 * Get icon component for insight type
 */
const getInsightIcon = (type) => {
  switch (type.id) {
    case 'warning': return AlertTriangle;
    case 'drill': return Target;
    case 'tactical': return Crosshair;
    case 'strength': return TrendingUp;
    default: return Info;
  }
};

/**
 * Individual insight card
 */
const InsightCard = ({ insight, onCalloutClick }) => {
  const Icon = getInsightIcon(insight.type);
  
  const handleClick = () => {
    if (insight.callout && onCalloutClick) {
      onCalloutClick(insight.callout);
    }
  };
  
  return (
    <div 
      className={`insight-card ${insight.type.id} ${insight.callout ? 'clickable' : ''}`}
      onClick={handleClick}
    >
      <div className="insight-icon" style={{ color: insight.type.color }}>
        <Icon size={22} />
      </div>
      
      <div className="insight-content">
        <p className="insight-text">{insight.text}</p>
        
        {insight.callout && (
          <div className="insight-callout">
            <span>{insight.callout}</span>
            <ChevronRight size={14} />
          </div>
        )}
        
        {insight.action && (
          <div className="insight-action">
            <span>💡 {insight.action}</span>
          </div>
        )}
      </div>
    </div>
  );
};

/**
 * InsightPanel - Shows top insights from analysis
 * @param {Object} props
 * @param {Object} props.calloutStats - All callout statistics
 * @param {Function} props.onCalloutClick - Callback when callout reference is clicked
 * @param {number} props.limit - Max insights to show
 */
const InsightPanel = ({ 
  calloutStats = {}, 
  onCalloutClick,
  limit = 5 
}) => {
  const insights = useMemo(() => {
    return getTopInsights(calloutStats, limit);
  }, [calloutStats, limit]);
  
  if (!insights || insights.length === 0) {
    return (
      <div className="insight-panel empty">
        <Info size={24} opacity={0.3} />
        <p>No hay suficientes datos para generar insights.</p>
        <p className="hint">Juega más partidas para obtener análisis detallado.</p>
      </div>
    );
  }
  
  // Count by type for summary
  const typeCounts = insights.reduce((acc, insight) => {
    acc[insight.type.id] = (acc[insight.type.id] || 0) + 1;
    return acc;
  }, {});
  
  return (
    <div className="insight-panel">
      <div className="panel-header">
        <h4 className="panel-title">
          <TrendingUp size={20} />
          Insights de Rendimiento
        </h4>
        <div className="insight-summary">
          {typeCounts.warning > 0 && (
            <span className="summary-badge warning">{typeCounts.warning} alertas</span>
          )}
          {typeCounts.strength > 0 && (
            <span className="summary-badge strength">{typeCounts.strength} fortalezas</span>
          )}
        </div>
      </div>
      
      <div className="insights-list">
        {insights.map((insight, idx) => (
          <InsightCard 
            key={`${insight.callout || 'global'}-${idx}`}
            insight={insight}
            onCalloutClick={onCalloutClick}
          />
        ))}
      </div>
    </div>
  );
};

export default InsightPanel;
