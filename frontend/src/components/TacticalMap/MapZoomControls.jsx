// frontend/src/components/TacticalMap/MapZoomControls.jsx
// Zoom controls + zoomable container for tactical map
import React, { useRef, useState, useEffect, useCallback } from 'react';
import { ZoomIn, ZoomOut, Maximize2, Focus } from 'lucide-react';
import { getMapProfile } from '../../utils/adaptiveClustering';
import '../../styles/TacticalMap/mapZoomControls.css';

const MIN_ZOOM = 1;
const MAX_ZOOM = 3;
const ZOOM_STEP = 0.25;

/**
 * ZoomableMapContainer — wraps the map image + overlays.
 * Handles CSS zoom transform, wheel-to-zoom, drag-to-pan,
 * and resets via viewKey / resetSignal changes.
 */
export const ZoomableMapContainer = ({
  zoomLevel = 1,
  className = '',
  viewKey,
  resetSignal,
  children,
}) => {
  const containerRef = useRef(null);
  const [translate, setTranslate] = useState({ x: 0, y: 0 });
  const dragState = useRef({
    dragging: false,
    element: null,
    pointerId: null,
    startX: 0,
    startY: 0,
    origX: 0,
    origY: 0,
  });

  const stopDragging = useCallback(() => {
    const pointerId = dragState.current.pointerId;
    const element = dragState.current.element;

    if (
      pointerId !== null
      && element?.hasPointerCapture?.(pointerId)
    ) {
      element.releasePointerCapture(pointerId);
    }

    dragState.current.dragging = false;
    dragState.current.element = null;
    dragState.current.pointerId = null;
  }, []);

  // Reset pan on view change or explicit reset
  useEffect(() => {
    stopDragging();
    setTranslate({ x: 0, y: 0 });
  }, [viewKey, resetSignal, stopDragging]);

  useEffect(() => stopDragging, [stopDragging]);

  // Clamp translation so the map doesn't go out of bounds
  const clampTranslate = useCallback((tx, ty, zoom) => {
    if (zoom <= 1) return { x: 0, y: 0 };
    const maxPan = ((zoom - 1) / zoom) * 50; // percentage-based
    return {
      x: Math.max(-maxPan, Math.min(maxPan, tx)),
      y: Math.max(-maxPan, Math.min(maxPan, ty)),
    };
  }, []);

  // Mouse drag handlers
  const handlePointerDown = useCallback((e) => {
    if (zoomLevel <= 1) return;
    // Only respond to primary button
    if (e.button !== 0) return;
    if (e.target.closest('button, a, input')) return;
    stopDragging();
    dragState.current = {
      dragging: true,
      element: e.currentTarget,
      pointerId: e.pointerId,
      startX: e.clientX,
      startY: e.clientY,
      origX: translate.x,
      origY: translate.y,
    };
    e.currentTarget.setPointerCapture(e.pointerId);
  }, [zoomLevel, translate, stopDragging]);

  const handlePointerMove = useCallback((e) => {
    if (!dragState.current.dragging) return;
    const el = containerRef.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const dx = ((e.clientX - dragState.current.startX) / rect.width) * 100;
    const dy = ((e.clientY - dragState.current.startY) / rect.height) * 100;
    const clamped = clampTranslate(
      dragState.current.origX + dx,
      dragState.current.origY + dy,
      zoomLevel
    );
    setTranslate(clamped);
  }, [zoomLevel, clampTranslate]);

  const isZoomed = zoomLevel > 1;

  const handleKeyDown = useCallback((event) => {
    if (!isZoomed) return;

    const offsets = {
      ArrowLeft: { x: 4, y: 0 },
      ArrowRight: { x: -4, y: 0 },
      ArrowUp: { x: 0, y: 4 },
      ArrowDown: { x: 0, y: -4 },
    };
    const offset = offsets[event.key];

    if (event.key === 'Home') {
      event.preventDefault();
      setTranslate({ x: 0, y: 0 });
      return;
    }

    if (!offset) return;
    event.preventDefault();
    setTranslate((current) => clampTranslate(
      current.x + offset.x,
      current.y + offset.y,
      zoomLevel
    ));
  }, [clampTranslate, isZoomed, zoomLevel]);

  return (
    <div
      ref={containerRef}
      className={`zoomable-map-container ${className} ${isZoomed ? 'zoomed' : ''}`}
      style={{
        transform: isZoomed
          ? `scale(${zoomLevel}) translate(${translate.x}%, ${translate.y}%)`
          : undefined,
      }}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={stopDragging}
      onPointerCancel={stopDragging}
      onLostPointerCapture={stopDragging}
      onKeyDown={handleKeyDown}
      role="region"
      aria-label={
        isZoomed
          ? 'Radar ampliado. Arrastra o usa las flechas para desplazarte; Inicio centra el mapa.'
          : 'Radar táctico interactivo'
      }
      tabIndex={0}
    >
      {children}
    </div>
  );
};

/**
 * MapZoomControls — zoom +/–/reset buttons and indicator.
 * Sits absolutely inside .map-view.
 */
const MapZoomControls = ({
  zoomLevel = 1,
  mapName,
  onZoomChange,
  onReset,
}) => {
  const profile = getMapProfile(mapName);
  const showHint = profile.density === 'compact' && zoomLevel === 1;

  const zoomIn = () => {
    const next = Math.min(MAX_ZOOM, +(zoomLevel + ZOOM_STEP).toFixed(2));
    onZoomChange(next);
  };

  const zoomOut = () => {
    const next = Math.max(MIN_ZOOM, +(zoomLevel - ZOOM_STEP).toFixed(2));
    onZoomChange(next);
  };

  return (
    <div className="map-zoom-controls">
      {showHint && (
        <div className="zoom-hint">
          <Focus size={12} aria-hidden="true" />
          <span>Haz zoom para ver detalle</span>
        </div>
      )}

      <div className="zoom-indicator">
        <span className="zoom-value">{zoomLevel.toFixed(2)}×</span>
      </div>

      <div className="zoom-buttons">
        <button
          type="button"
          className={`zoom-btn ${zoomLevel >= MAX_ZOOM ? 'disabled' : ''}`}
          onClick={zoomIn}
          disabled={zoomLevel >= MAX_ZOOM}
          aria-label="Acercar"
        >
          <ZoomIn size={16} aria-hidden="true" />
        </button>
        <button
          type="button"
          className={`zoom-btn ${zoomLevel <= MIN_ZOOM ? 'disabled' : ''}`}
          onClick={zoomOut}
          disabled={zoomLevel <= MIN_ZOOM}
          aria-label="Alejar"
        >
          <ZoomOut size={16} aria-hidden="true" />
        </button>
        <button
          type="button"
          className="zoom-btn reset"
          onClick={onReset}
          aria-label="Restablecer zoom"
        >
          <Maximize2 size={14} aria-hidden="true" />
        </button>
      </div>
    </div>
  );
};

export default MapZoomControls;
