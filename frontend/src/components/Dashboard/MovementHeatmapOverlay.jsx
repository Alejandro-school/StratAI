// frontend/src/components/Dashboard/MovementHeatmapOverlay.jsx
// Professional heatmap like HLTV - solid colors, no ghost effect

import React, { useRef, useEffect, useMemo } from 'react';
import '../../styles/Dashboard/movementHeatmap.css';

const MovementHeatmapOverlay = ({ 
  heatmapGrid = [], 
  heatmapIntensity = 70,
  showHeatmap = true,
  activeSide = 'all',
  currentLevel = 'upper',
  zThreshold = null
}) => {
  const canvasRef = useRef(null);
  
  // Filter heatmap data based on side AND level (for multi-level maps like Nuke)
  const filteredGrid = useMemo(() => {
    let grid = heatmapGrid;
    
    // 1. Filter by level (Z coordinate) for multi-level maps
    if (zThreshold !== null) {
      grid = grid.filter(point => {
        const avgZ = point.avg_z;
        // If no Z data, show on upper by default
        if (avgZ === undefined || avgZ === null) {
          return currentLevel === 'upper';
        }
        // Filter based on level
        if (currentLevel === 'upper') {
          return avgZ >= zThreshold;
        } else {
          return avgZ < zThreshold;
        }
      });
    }
    
    // 2. Adjust intensity by CT/T side
    if (activeSide !== 'all') {
      grid = grid.map(point => {
        const ctRatio = point.ct_ratio || 50;
        let adjustedIntensity;
        
        if (activeSide === 'ct') {
          adjustedIntensity = point.intensity * (ctRatio / 100);
        } else {
          adjustedIntensity = point.intensity * ((100 - ctRatio) / 100);
        }
        
        return {
          ...point,
          intensity: adjustedIntensity
        };
      }).filter(point => point.intensity > 5);
    }
    
    return grid;
  }, [heatmapGrid, activeSide, currentLevel, zThreshold]);
  
  useEffect(() => {
    if (!showHeatmap || !canvasRef.current || filteredGrid.length === 0) return;
    
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    
    const parent = canvas.parentElement;
    const width = parent.offsetWidth;
    const height = parent.offsetHeight;
    
    canvas.width = width;
    canvas.height = height;
    
    // Step 1: Create intensity map (grayscale)
    const intensityCanvas = document.createElement('canvas');
    intensityCanvas.width = width;
    intensityCanvas.height = height;
    const intensityCtx = intensityCanvas.getContext('2d');
    
    // Black background
    intensityCtx.fillStyle = 'black';
    intensityCtx.fillRect(0, 0, width, height);
    
    // Draw white circles for each point (additive)
    intensityCtx.globalCompositeOperation = 'lighter';
    
    const radius = Math.max(width, height) / 14;
    
    filteredGrid.forEach(point => {
      const x = (point.x / 100) * width;
      const y = (point.y / 100) * height;
      const intensity = point.intensity / 100;
      
      // Create radial gradient from white to transparent
      const gradient = intensityCtx.createRadialGradient(x, y, 0, x, y, radius);
      const alpha = Math.min(1, intensity * 1.5);
      gradient.addColorStop(0, `rgba(255, 255, 255, ${alpha})`);
      gradient.addColorStop(0.2, `rgba(255, 255, 255, ${alpha * 0.8})`);
      gradient.addColorStop(0.5, `rgba(255, 255, 255, ${alpha * 0.4})`);
      gradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
      
      intensityCtx.fillStyle = gradient;
      intensityCtx.beginPath();
      intensityCtx.arc(x, y, radius, 0, Math.PI * 2);
      intensityCtx.fill();
    });
    
    // Step 2: Get intensity data and colorize
    const imageData = intensityCtx.getImageData(0, 0, width, height);
    const data = imageData.data;
    
    // Colorize based on intensity value
    const alphaMultiplier = heatmapIntensity / 100;
    
    for (let i = 0; i < data.length; i += 4) {
      const value = data[i]; // Red channel = intensity (grayscale)
      
      if (value > 10) { // Threshold to avoid noise
        const color = getHeatmapColor(value / 255);
        data[i] = color.r;
        data[i + 1] = color.g;
        data[i + 2] = color.b;
        data[i + 3] = Math.min(255, value * 2 * alphaMultiplier); // Alpha
      } else {
        data[i + 3] = 0; // Transparent
      }
    }
    
    // Step 3: Draw colorized heatmap
    ctx.clearRect(0, 0, width, height);
    ctx.putImageData(imageData, 0, 0);
    
  }, [filteredGrid, showHeatmap, heatmapIntensity]);

  // HLTV-style color palette: Blue → Cyan → Green → Yellow → Orange → Red
  function getHeatmapColor(t) {
    // t is 0-1
    if (t < 0.2) {
      // Blue
      return { r: 0, g: 0, b: 255 };
    } else if (t < 0.35) {
      // Blue to Cyan
      const p = (t - 0.2) / 0.15;
      return {
        r: 0,
        g: Math.round(255 * p),
        b: 255
      };
    } else if (t < 0.5) {
      // Cyan to Green
      const p = (t - 0.35) / 0.15;
      return {
        r: 0,
        g: 255,
        b: Math.round(255 * (1 - p))
      };
    } else if (t < 0.65) {
      // Green to Yellow
      const p = (t - 0.5) / 0.15;
      return {
        r: Math.round(255 * p),
        g: 255,
        b: 0
      };
    } else if (t < 0.8) {
      // Yellow to Orange
      const p = (t - 0.65) / 0.15;
      return {
        r: 255,
        g: Math.round(255 - 128 * p),
        b: 0
      };
    } else {
      // Orange to Red
      const p = (t - 0.8) / 0.2;
      return {
        r: 255,
        g: Math.round(127 - 127 * p),
        b: 0
      };
    }
  }

  if (!showHeatmap || filteredGrid.length === 0) {
    return null;
  }

  return (
    <canvas 
      ref={canvasRef}
      className="movement-heatmap-canvas"
    />
  );
};

export default MovementHeatmapOverlay;
