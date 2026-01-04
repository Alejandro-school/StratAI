// frontend/src/components/Dashboard/HeatmapCanvas.jsx
// Adaptive heatmap visualization for kills/deaths with smart clustering

import React, { useRef, useMemo, useState } from 'react';
import { clusterHeatmapPoints } from '../../utils/mapCoordinates';
import { adaptiveCluster, getMapProfile } from '../../utils/adaptiveClustering';
import '../../styles/Dashboard/heatmapCanvas.css';

/**
 * HeatmapCanvas - Renders clustered kill/death points on the map
 * Now with adaptive clustering based on map density profile
 * 
 * @param {Object} props
 * @param {Array} props.heatmapData - Array of {x, y, type, callout} points
 * @param {boolean} props.showKills - Show kill points
 * @param {boolean} props.showDeaths - Show death points
 * @param {Function} props.onClusterClick - Callback when cluster is clicked
 * @param {string} props.mapName - Map name for adaptive clustering
 */
const HeatmapCanvas = ({ 
  heatmapData = [], 
  showKills = true, 
  showDeaths = true,
  onClusterClick,
  mapName = 'de_dust2'
}) => {
  const containerRef = useRef(null);
  const [hoveredCluster, setHoveredCluster] = useState(null);
  
  // Get map profile for adaptive behavior
  const mapProfile = useMemo(() => getMapProfile(mapName), [mapName]);
  const isCompactMap = mapProfile.density === 'compact';
  
  // Filter and cluster points using adaptive algorithm
  const clusters = useMemo(() => {
    let filtered = heatmapData;
    
    if (!showKills) {
      filtered = filtered.filter(p => p.type !== 'kill');
    }
    if (!showDeaths) {
      filtered = filtered.filter(p => p.type !== 'death');
    }
    
    // Use adaptive clustering for better results on dense maps
    return adaptiveCluster(filtered, mapName);
  }, [heatmapData, showKills, showDeaths, mapName]);
  
  if (clusters.length === 0) {
    return null;
  }
  
  // Calculate max for scaling
  const maxTotal = Math.max(...clusters.map(c => c.total), 1);
  
  // Adjusted sizes for compact maps
  const getClusterSize = (total) => {
    const baseMin = isCompactMap ? 14 : 18;
    const baseMax = isCompactMap ? 36 : 48;
    const range = baseMax - baseMin;
    return baseMin + (total / maxTotal) * range;
  };
  
  return (
    <div className={`heatmap-overlay ${isCompactMap ? 'compact' : ''}`} ref={containerRef}>
      {clusters.map((cluster, idx) => {
        const size = getClusterSize(cluster.total);
        const isHovered = hoveredCluster === idx;
        
        // Color based on kill ratio with improved gradient
        const killRatio = cluster.ratio;
        
        // Better color interpolation: full red -> yellow -> full green
        let red, green;
        if (killRatio < 0.5) {
          // Red to Yellow
          red = 255;
          green = Math.round(killRatio * 2 * 200);
        } else {
          // Yellow to Green
          red = Math.round((1 - killRatio) * 2 * 255);
          green = 200;
        }
        
        const baseOpacity = isCompactMap ? 0.6 : 0.7;
        const hoverOpacity = 0.9;
        const opacity = isHovered ? hoverOpacity : baseOpacity;
        
        const bgColor = `rgba(${red}, ${green}, 60, ${opacity})`;
        const borderColor = `rgba(${red}, ${green}, 60, 1)`;
        const glowColor = `rgba(${red}, ${green}, 60, 0.5)`;
        
        return (
          <button
            key={`cluster-${idx}`}
            className={`heatmap-cluster ${isHovered ? 'hovered' : ''} ${isCompactMap ? 'compact' : ''}`}
            style={{
              left: `${cluster.x}%`,
              top: `${cluster.y}%`,
              width: `${size}px`,
              height: `${size}px`,
              backgroundColor: bgColor,
              borderColor: borderColor,
              '--glow-color': glowColor,
              transform: 'translate(-50%, -50%)'
            }}
            onClick={() => onClusterClick?.(cluster)}
            onMouseEnter={() => setHoveredCluster(idx)}
            onMouseLeave={() => setHoveredCluster(null)}
            title={`${cluster.callout}: ${cluster.kills}K / ${cluster.deaths}D`}
          >
            <span className="cluster-count">{cluster.total}</span>
            
            {/* Hover tooltip for compact mode */}
            {isHovered && (
              <div className="cluster-tooltip">
                <span className="tooltip-callout">{cluster.callout || 'Unknown'}</span>
                <div className="tooltip-stats">
                  <span className="kills">{cluster.kills}K</span>
                  <span className="divider">/</span>
                  <span className="deaths">{cluster.deaths}D</span>
                </div>
              </div>
            )}
          </button>
        );
      })}
    </div>
  );
};

export default HeatmapCanvas;
