// frontend/src/components/Dashboard/MapZoomControls.jsx
// Interactive Zoom Controls for Dense Map Areas
import React from 'react';
import { ZoomIn, ZoomOut, Maximize2, Focus } from 'lucide-react';
import { getMapProfile } from '../../utils/adaptiveClustering';
import '../../styles/Dashboard/mapZoomControls.css';

/**
 * MapZoomControls - Zoom and focus controls for the map
 * 
 * @param {number} zoomLevel - Current zoom level (1 = 100%)
 * @param {function} onZoomChange - Callback when zoom changes
 * @param {function} onReset - Callback to reset view
 * @param {string} mapName - Current map for profile-aware behavior
 * @param {Object} focusArea - Optional { x, y } to focus on
 */
const MapZoomControls = ({ 
  zoomLevel = 1, 
  onZoomChange, 
  onReset,
  mapName,
  focusArea = null,
  onFocusArea
}) => {
  const profile = getMapProfile(mapName);
  const isCompact = profile.density === 'compact';
  
  const minZoom = 1;
  const maxZoom = isCompact ? 2.5 : 2;
  const zoomStep = 0.25;
  
  const handleZoomIn = () => {
    const newZoom = Math.min(maxZoom, zoomLevel + zoomStep);
    onZoomChange?.(newZoom);
  };
  
  const handleZoomOut = () => {
    const newZoom = Math.max(minZoom, zoomLevel - zoomStep);
    onZoomChange?.(newZoom);
  };
  
  const handleReset = () => {
    onZoomChange?.(1);
    onReset?.();
  };
  
  const zoomPercentage = Math.round(zoomLevel * 100);
  const canZoomIn = zoomLevel < maxZoom;
  const canZoomOut = zoomLevel > minZoom;
  
  return (
    <div className="map-zoom-controls">
      {/* Zoom Level Indicator */}
      <div className="zoom-indicator">
        <span className="zoom-value">{zoomPercentage}%</span>
      </div>
      
      {/* Zoom Buttons */}
      <div className="zoom-buttons">
        <button 
          className={`zoom-btn ${!canZoomIn ? 'disabled' : ''}`}
          onClick={handleZoomIn}
          disabled={!canZoomIn}
          title="Zoom In"
        >
          <ZoomIn size={16} />
        </button>
        
        <button 
          className={`zoom-btn ${!canZoomOut ? 'disabled' : ''}`}
          onClick={handleZoomOut}
          disabled={!canZoomOut}
          title="Zoom Out"
        >
          <ZoomOut size={16} />
        </button>
        
        <button 
          className={`zoom-btn reset ${zoomLevel === 1 ? 'disabled' : ''}`}
          onClick={handleReset}
          disabled={zoomLevel === 1}
          title="Reset View"
        >
          <Maximize2 size={16} />
        </button>
      </div>
      
      {/* Focus hint for compact maps */}
      {isCompact && zoomLevel === 1 && (
        <div className="zoom-hint">
          <Focus size={12} />
          <span>Zoom for details</span>
        </div>
      )}
    </div>
  );
};

/**
 * DensityZoneIndicator - Shows clickable "zoom here" hotspots
 * for areas with many overlapping markers
 */
export const DensityZoneIndicator = ({ 
  zone, 
  onClick 
}) => {
  return (
    <button 
      className="density-zone-indicator"
      style={{ left: `${zone.x}%`, top: `${zone.y}%` }}
      onClick={() => onClick?.(zone)}
      title={`${zone.count} callouts - Click to zoom`}
    >
      <div className="zone-ring" />
      <div className="zone-core">
        <Focus size={14} />
        <span>{zone.count}</span>
      </div>
    </button>
  );
};

/**
 * ZoomableMapContainer - Wrapper that handles zoom/pan transforms
 */
export const ZoomableMapContainer = ({ 
  children, 
  zoomLevel = 1,
  panPosition = { x: 50, y: 50 },
  className = ''
}) => {
  const transform = zoomLevel === 1 
    ? 'none'
    : `scale(${zoomLevel}) translate(${50 - panPosition.x}%, ${50 - panPosition.y}%)`;
  
  return (
    <div 
      className={`zoomable-map-container ${className} ${zoomLevel > 1 ? 'zoomed' : ''}`}
      style={{
        '--zoom-level': zoomLevel,
        transform,
        transformOrigin: `${panPosition.x}% ${panPosition.y}%`
      }}
    >
      {children}
    </div>
  );
};

export default MapZoomControls;
