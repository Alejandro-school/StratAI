// frontend/src/components/Dashboard/AdaptiveHotspot.jsx
// Adaptive Callout Hotspots - Smart sizing for dense maps
import React, { useMemo, useState } from 'react';
import { calculateMarkerSize, getMapProfile } from '../../utils/adaptiveClustering';
import '../../styles/Dashboard/adaptiveHotspot.css';

/**
 * AdaptiveHotspot - Smart callout marker that adapts to map density
 * Uses smaller markers on compact maps like Nuke
 * 
 * Features:
 * - Adaptive sizing based on map profile
 * - Micro mode for ultra-compact display
 * - Hover expansion for details
 * - Anti-overlap position adjustments
 */
const AdaptiveHotspot = ({ 
  callout, 
  mapName,
  isSelected, 
  isBest, 
  isWorst, 
  onClick,
  zoomLevel = 1,
  showConnector = false
}) => {
  const [isHovered, setIsHovered] = useState(false);
  const { name, position, rating, win_rate, kd, kills = 0, deaths = 0 } = callout;
  
  // Get map-aware sizing (must be called before any early returns)
  const sizeConfig = useMemo(() => 
    calculateMarkerSize(mapName, zoomLevel, isSelected),
    [mapName, zoomLevel, isSelected]
  );
  
  // Redesigned AdaptiveHotspot - Professional Pill Shape
  // Always show full name for better readability (no truncation)
  const isCompact = false; // Disable compact mode for cleaner design
  
  // Use adjusted position if available
  const rawX = position.x ?? callout.adjustedX ?? callout.x ?? 0;
  const rawY = position.y ?? callout.adjustedY ?? callout.y ?? 0;
  const displayX = Math.max(2, Math.min(98, rawX));
  const displayY = Math.max(2, Math.min(98, rawY));
  const wasAdjusted = position.wasAdjusted || false;
  
  const ratingClass = rating === 'good' ? 'good' : rating === 'bad' ? 'bad' : 'neutral';
  
  return (
    <button
      className={`
        adaptive-hotspot 
        ${ratingClass} 
        ${isSelected ? 'selected' : ''} 
        ${isBest ? 'best' : ''} 
        ${isWorst ? 'worst' : ''}
        ${isHovered ? 'hovered' : ''}
        ${wasAdjusted ? 'adjusted' : ''}
      `}
      style={{ 
        left: `${displayX}%`, 
        top: `${displayY}%`,
        '--marker-scale': `${sizeConfig.size / 36}` // Scale factor
      }}
      onClick={() => onClick(callout)}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {/* Pulse ring - only on active/best/worst */}
      {(isBest || isWorst || isSelected) && (
        <div className="hotspot-pulse" />
      )}
      
      {/* Core marker - Pill Shape */}
      <div className="hotspot-core">
        <span className="hotspot-label">{name}</span>
        
        {/* Integrated simple stat */}
        <div className="mini-stat-pill">
          {win_rate}%
        </div>
      </div>
      
      {/* Connector line if position was adjusted */}
      {showConnector && wasAdjusted && callout.originalPosition && (
        <svg className="position-connector" viewBox="0 0 100 100" preserveAspectRatio="none">
          <line 
            x1="50" y1="50"
            x2={50 + (callout.originalPosition.x - displayX) * 2}
            y2={50 + (callout.originalPosition.y - displayY) * 2}
            stroke="currentColor"
            strokeWidth="1"
            strokeDasharray="2 2"
            opacity="0.4"
          />
        </svg>
      )}
      
      {/* Tooltip - expands on hover */}
      <div className="hotspot-tooltip">
        <div className="tooltip-header">
          <span className="tooltip-name">{name}</span>
          {isBest && <span className="tooltip-badge best">★ BEST</span>}
          {isWorst && <span className="tooltip-badge worst">⚠ WEAK</span>}
        </div>
        
        <div className="tooltip-stats">
          <div className="stat-item">
            <span className={`stat-value ${ratingClass}`}>{win_rate}%</span>
            <span className="stat-label">WIN RATE</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{kd}</span>
            <span className="stat-label">K/D</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{kills}/{deaths}</span>
            <span className="stat-label">K/D SPLIT</span>
          </div>
        </div>
        
        {/* Quick insight */}
        {(isBest || isWorst) && (
          <div className={`tooltip-insight ${isBest ? 'positive' : 'negative'}`}>
            {isBest 
              ? '🎯 Your strongest position on this map'
              : '⚠️ Consider practicing this area'
            }
          </div>
        )}
      </div>
      
      {/* Best/Worst indicator badges */}
      {isBest && <div className="badge-indicator best">★</div>}
      {isWorst && <div className="badge-indicator worst">!</div>}
    </button>
  );
};

/**
 * AdaptiveHotspotLayer - Renders all hotspots with collision detection
 */
export const AdaptiveHotspotLayer = ({ 
  callouts, 
  mapName,
  selectedCallout,
  bestCallout,
  worstCallout,
  onSelect,
  zoomLevel = 1
}) => {
  const profile = getMapProfile(mapName);
  
  // Sort: selected first, then best/worst, then by activity
  const sortedCallouts = useMemo(() => {
    return [...callouts].sort((a, b) => {
      if (a.name === selectedCallout) return 1;
      if (b.name === selectedCallout) return -1;
      if (a.name === bestCallout?.name) return 1;
      if (b.name === bestCallout?.name) return -1;
      if (a.name === worstCallout?.name) return 1;
      if (b.name === worstCallout?.name) return -1;
      return (b.sample_size || 0) - (a.sample_size || 0);
    });
  }, [callouts, selectedCallout, bestCallout, worstCallout]);
  
  return (
    <div className={`adaptive-hotspot-layer ${profile.density}`}>
      {sortedCallouts.map(callout => (
        <AdaptiveHotspot
          key={callout.name}
          callout={callout}
          mapName={mapName}
          isSelected={selectedCallout === callout.name}
          isBest={bestCallout?.name === callout.name}
          isWorst={worstCallout?.name === callout.name}
          onClick={onSelect}
          zoomLevel={zoomLevel}
        />
      ))}
    </div>
  );
};

export default AdaptiveHotspot;
