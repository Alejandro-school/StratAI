// frontend/src/components/Dashboard/MapZoomControls.jsx
// Interactive Zoom Controls for Dense Map Areas
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ZoomIn, ZoomOut, Maximize2, Focus } from 'lucide-react';
import { getMapProfile } from '../../utils/adaptiveClustering';
import '../../styles/TacticalMap/mapZoomControls.css';

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
  viewKey,
  resetSignal,
  className = ''
}) => {
  const containerRef = useRef(null);
  const dragStateRef = useRef({ dragging: false, pointerId: null, lastX: 0, lastY: 0 });
  const [panOffset, setPanOffset] = useState({ x: 0, y: 0 });

  const clampPan = useCallback((offset, zoom) => {
    const el = containerRef.current;
    if (!el || zoom <= 1) {
      return { x: 0, y: 0 };
    }

    const rect = el.getBoundingClientRect();
    const maxX = ((zoom - 1) * rect.width) / 2;
    const maxY = ((zoom - 1) * rect.height) / 2;

    return {
      x: Math.max(-maxX, Math.min(maxX, offset.x)),
      y: Math.max(-maxY, Math.min(maxY, offset.y)),
    };
  }, []);

  useEffect(() => {
    if (zoomLevel <= 1) {
      setPanOffset({ x: 0, y: 0 });
      return;
    }

    setPanOffset((prev) => clampPan(prev, zoomLevel));
  }, [zoomLevel, clampPan]);

  useEffect(() => {
    setPanOffset({ x: 0, y: 0 });
  }, [viewKey, resetSignal]);

  const handlePointerDown = useCallback((event) => {
    if (zoomLevel <= 1) return;
    if (event.target.closest('button, a')) return;

    dragStateRef.current = {
      dragging: true,
      pointerId: event.pointerId,
      lastX: event.clientX,
      lastY: event.clientY,
    };

    event.currentTarget.setPointerCapture(event.pointerId);
  }, [zoomLevel]);

  const handlePointerMove = useCallback((event) => {
    const state = dragStateRef.current;
    if (!state.dragging || state.pointerId !== event.pointerId) return;

    const deltaX = event.clientX - state.lastX;
    const deltaY = event.clientY - state.lastY;

    dragStateRef.current.lastX = event.clientX;
    dragStateRef.current.lastY = event.clientY;

    setPanOffset((prev) => clampPan({ x: prev.x + deltaX, y: prev.y + deltaY }, zoomLevel));
  }, [clampPan, zoomLevel]);

  const handlePointerUp = useCallback((event) => {
    if (dragStateRef.current.pointerId === event.pointerId) {
      dragStateRef.current = { dragging: false, pointerId: null, lastX: 0, lastY: 0 };
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
  }, []);

  const transform = useMemo(() => {
    if (zoomLevel === 1) return 'none';
    return `translate(${panOffset.x}px, ${panOffset.y}px) scale(${zoomLevel})`;
  }, [panOffset.x, panOffset.y, zoomLevel]);
  
  return (
    <div 
      ref={containerRef}
      className={`zoomable-map-container ${className} ${zoomLevel > 1 ? 'zoomed' : ''}`}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
      style={{
        '--zoom-level': zoomLevel,
        transform,
        transformOrigin: 'center center',
        touchAction: zoomLevel > 1 ? 'none' : 'auto'
      }}
    >
      {children}
    </div>
  );
};

export default MapZoomControls;
