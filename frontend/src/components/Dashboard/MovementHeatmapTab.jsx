// frontend/src/components/Dashboard/MovementHeatmapTab.jsx
// Simple sidebar panel for movement heatmap

import React, { useState, useEffect } from 'react';
import { Activity, MapPin, Layers, Navigation } from 'lucide-react';
import '../../styles/Dashboard/movementHeatmap.css';

/**
 * MovementHeatmapTab - Simple sidebar panel with heatmap controls
 */
const MovementHeatmapTab = ({ 
  metrics = {},
  matchesAnalyzed = 0,
  heatmapIntensity = 70,
  onIntensityChange,
  showHeatmap = true,
  onToggleHeatmap,
  loading = false
}) => {
  const { top_positions = [], total_rounds = 0 } = metrics;
  
  // Local state for smooth slider interaction
  const [localIntensity, setLocalIntensity] = useState(heatmapIntensity);
  
  // Sync with prop when it changes externally
  useEffect(() => {
    setLocalIntensity(heatmapIntensity);
  }, [heatmapIntensity]);
  
  // Instant update - no debounce
  const handleSliderChange = (value) => {
    setLocalIntensity(value);
    onIntensityChange?.(value);
  };

  if (loading) {
    return (
      <div className="movement-tab-panel loading">
        <div className="loading-spinner" />
        <span>Analizando movimiento...</span>
      </div>
    );
  }

  const hasData = total_rounds > 0;

  return (
    <div className="movement-tab-panel">
      {/* Header */}
      <div className="movement-header">
        <div className="header-title">
          <Activity size={20} className="header-icon" />
          <h4>Mapa de Calor</h4>
        </div>
        <div className="header-meta">
          <span className="meta-item">{matchesAnalyzed} partidas</span>
          <span className="meta-item">{total_rounds} rondas</span>
        </div>
      </div>

      {/* Intensity Slider */}
      <div className="intensity-control">
        <div className="intensity-header">
          <Layers size={14} />
          <span>Intensidad</span>
          <span className="intensity-value">{localIntensity}%</span>
        </div>
        <input 
          type="range"
          min="20"
          max="100"
          value={localIntensity}
          onChange={(e) => handleSliderChange(parseInt(e.target.value))}
          className="intensity-slider"
        />
      </div>

      {/* Color Scale Legend */}
      <div className="color-scale-legend">
        <span className="legend-label">Poco tiempo</span>
        <div className="color-scale-bar" />
        <span className="legend-label">Mucho tiempo</span>
      </div>

      {!hasData ? (
        <div className="empty-state">
          <Navigation size={40} strokeWidth={1.5} />
          <h5>Sin datos de movimiento</h5>
          <p>Juega más partidas para ver dónde pasas más tiempo</p>
        </div>
      ) : (
        <>
          {/* Top Positions */}
          {top_positions.length > 0 && (
            <div className="top-positions">
              <h5>
                <MapPin size={14} />
                Zonas más frecuentes
              </h5>
              <div className="positions-list">
                {top_positions.slice(0, 8).map((pos, idx) => (
                  <div key={idx} className="position-item">
                    <div className="position-rank">{idx + 1}</div>
                    <span className="position-name">{pos.area}</span>
                    <span className="position-percent">{pos.time_percent}%</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default MovementHeatmapTab;
