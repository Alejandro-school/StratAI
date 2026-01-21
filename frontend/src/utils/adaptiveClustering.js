// frontend/src/utils/adaptiveClustering.js
// Adaptive Clustering System - Smart anti-overlap for dense maps like Nuke

/**
 * Map density profiles - determines clustering behavior
 * compact = high density, needs smaller markers
 * standard = normal density
 * sparse = low density, can use larger markers
 */
export const MAP_DENSITY_PROFILES = {
  de_nuke: { 
    density: 'compact', 
    clusterThreshold: 2.5, 
    markerScale: 0.9,  // Increased for better visibility
    hasLevels: true,
    minMarkerSize: 38, // Minimum size in pixels
    description: 'Very compact map with multiple levels'
  },
  de_vertigo: { 
    density: 'compact', 
    clusterThreshold: 2.8, 
    markerScale: 0.7,
    hasLevels: true,
    description: 'Compact vertical map'
  },
  de_train: { 
    density: 'standard', 
    clusterThreshold: 3.5, 
    markerScale: 0.85,
    hasLevels: true,
    description: 'Medium density with trains'
  },
  de_inferno: { 
    density: 'standard', 
    clusterThreshold: 3.5, 
    markerScale: 0.85,
    description: 'Medium density'
  },
  de_mirage: { 
    density: 'standard', 
    clusterThreshold: 4.0, 
    markerScale: 0.9,
    description: 'Balanced layout'
  },
  de_dust2: { 
    density: 'sparse', 
    clusterThreshold: 4.5, 
    markerScale: 1.0,
    description: 'Open layout'
  },
  de_overpass: { 
    density: 'standard', 
    clusterThreshold: 3.8, 
    markerScale: 0.85,
    description: 'Mixed density'
  },
  de_ancient: { 
    density: 'standard', 
    clusterThreshold: 4.0, 
    markerScale: 0.9,
    description: 'Open layout'
  },
  de_anubis: { 
    density: 'standard', 
    clusterThreshold: 3.8, 
    markerScale: 0.85,
    description: 'Medium density'
  }
};

/**
 * Get map profile with sensible defaults
 */
export const getMapProfile = (mapName) => {
  return MAP_DENSITY_PROFILES[mapName] || {
    density: 'standard',
    clusterThreshold: 4.0,
    markerScale: 1.0,
    description: 'Unknown map'
  };
};

/**
 * Calculate distance between two points (percentage-based)
 */
const distance = (p1, p2) => {
  const dx = p1.x - p2.x;
  const dy = p1.y - p2.y;
  return Math.sqrt(dx * dx + dy * dy);
};

/**
 * Detect overlapping markers and calculate offset positions
 * Uses force-directed approach to push overlapping markers apart
 * 
 * @param {Array} markers - Array of {x, y, name, ...} objects
 * @param {number} minDistance - Minimum distance between markers (in %)
 * @param {number} iterations - Number of repulsion iterations
 * @returns {Array} Markers with adjusted positions
 */
export const resolveOverlaps = (markers, minDistance = 5, iterations = 3) => {
  if (!markers || markers.length < 2) return markers;
  
  // Clone to avoid mutation - handle both {x,y} and {position:{x,y}} formats
  let adjusted = markers.map(m => {
    const x = m.position?.x ?? m.x ?? 0;
    const y = m.position?.y ?? m.y ?? 0;
    return { ...m, adjustedX: x, adjustedY: y, _originalX: x, _originalY: y };
  });
  
  for (let iter = 0; iter < iterations; iter++) {
    for (let i = 0; i < adjusted.length; i++) {
      for (let j = i + 1; j < adjusted.length; j++) {
        const a = adjusted[i];
        const b = adjusted[j];
        
        const dist = distance(
          { x: a.adjustedX, y: a.adjustedY },
          { x: b.adjustedX, y: b.adjustedY }
        );
        
        if (dist < minDistance && dist > 0) {
          // Calculate repulsion force
          const overlap = minDistance - dist;
          const force = overlap * 0.5;
          
          // Direction vector from a to b
          const dx = (b.adjustedX - a.adjustedX) / dist;
          const dy = (b.adjustedY - a.adjustedY) / dist;
          
          // Push markers apart
          a.adjustedX -= dx * force;
          a.adjustedY -= dy * force;
          b.adjustedX += dx * force;
          b.adjustedY += dy * force;
          
          // Clamp to bounds
          a.adjustedX = Math.max(2, Math.min(98, a.adjustedX));
          a.adjustedY = Math.max(2, Math.min(98, a.adjustedY));
          b.adjustedX = Math.max(2, Math.min(98, b.adjustedX));
          b.adjustedY = Math.max(2, Math.min(98, b.adjustedY));
        }
      }
    }
  }
  
  // Return with final positions and offset flags
  return adjusted.map(m => ({
    ...m,
    position: {
      x: m.adjustedX,
      y: m.adjustedY,
      wasAdjusted: Math.abs(m.adjustedX - m._originalX) > 0.1 || Math.abs(m.adjustedY - m._originalY) > 0.1
    },
    originalPosition: { x: m._originalX, y: m._originalY }
  }));
};

/**
 * Smart clustering that adapts to map density
 * 
 * @param {Array} points - Array of {x, y, type, callout} objects
 * @param {string} mapName - Map identifier
 * @returns {Array} Clustered points
 */
export const adaptiveCluster = (points, mapName) => {
  const profile = getMapProfile(mapName);
  const threshold = profile.clusterThreshold;
  
  const clusters = [];
  const used = new Set();
  
  // Sort by density to prioritize high-activity areas
  const sortedPoints = [...points].sort((a, b) => {
    const aNeighbors = points.filter(p => distance(a, p) < threshold * 2).length;
    const bNeighbors = points.filter(p => distance(b, p) < threshold * 2).length;
    return bNeighbors - aNeighbors;
  });
  
  sortedPoints.forEach((point, i) => {
    if (used.has(i)) return;
    
    const cluster = {
      x: point.x,
      y: point.y,
      kills: point.type === 'kill' ? 1 : 0,
      deaths: point.type === 'death' ? 1 : 0,
      callout: point.callout,
      points: [point]
    };
    used.add(i);
    
    // Find nearby points
    let sumX = point.x;
    let sumY = point.y;
    let count = 1;
    
    sortedPoints.forEach((other, j) => {
      if (used.has(j)) return;
      if (distance(point, other) <= threshold) {
        if (other.type === 'kill') cluster.kills++;
        else cluster.deaths++;
        cluster.points.push(other);
        sumX += other.x;
        sumY += other.y;
        count++;
        used.add(j);
      }
    });
    
    // Use centroid position
    cluster.x = sumX / count;
    cluster.y = sumY / count;
    cluster.total = cluster.kills + cluster.deaths;
    cluster.ratio = cluster.kills / (cluster.total || 1);
    clusters.push(cluster);
  });
  
  // Resolve overlaps between clusters
  const resolved = resolveOverlaps(clusters, profile.clusterThreshold * 1.5);
  
  return resolved.map(c => ({
    ...c,
    x: c.position?.x ?? c.x,
    y: c.position?.y ?? c.y
  }));
};

/**
 * Group callouts and detect density zones
 * Useful for showing "zoom here for details" indicators
 * 
 * @param {Array} callouts - Array of callout objects with positions
 * @param {string} mapName - Map identifier
 * @returns {Object} { callouts, densityZones }
 */
export const processCalloutsForDisplay = (callouts, mapName) => {
  const profile = getMapProfile(mapName);
  const minMarkerDistance = profile.density === 'compact' ? 4 : 
                            profile.density === 'sparse' ? 6 : 5;
  
  // Resolve overlaps
  const resolved = resolveOverlaps(
    callouts.filter(c => c.position), 
    minMarkerDistance
  );
  
  // Detect high-density zones
  const densityZones = [];
  const zoneThreshold = 15; // % of map
  
  for (let i = 0; i < resolved.length; i++) {
    const neighbors = resolved.filter((c, j) => 
      j !== i && distance(
        { x: c.position?.x ?? c.x, y: c.position?.y ?? c.y },
        { x: resolved[i].position?.x ?? resolved[i].x, y: resolved[i].position?.y ?? resolved[i].y }
      ) < zoneThreshold
    );
    
    if (neighbors.length >= 3) {
      const zone = {
        x: resolved[i].position?.x ?? resolved[i].x,
        y: resolved[i].position?.y ?? resolved[i].y,
        count: neighbors.length + 1,
        callouts: [resolved[i].name, ...neighbors.map(n => n.name)]
      };
      
      // Check if zone overlaps with existing
      const overlaps = densityZones.some(z => 
        distance(z, zone) < zoneThreshold * 0.5
      );
      
      if (!overlaps) {
        densityZones.push(zone);
      }
    }
  }
  
  return {
    callouts: resolved,
    densityZones,
    profile
  };
};

/**
 * Calculate optimal marker size based on map and zoom level
 * 
 * @param {string} mapName - Map identifier
 * @param {number} zoomLevel - Current zoom (1 = 100%)
 * @param {boolean} isSelected - Is the marker selected
 * @returns {Object} { size, fontSize, showLabel }
 */
export const calculateMarkerSize = (mapName, zoomLevel = 1, isSelected = false) => {
  const profile = getMapProfile(mapName);
  const baseSize = 42; // Base marker size in pixels (increased for better visibility)
  
  // Scale by map profile and zoom
  let size = baseSize * profile.markerScale * zoomLevel;
  
  // Use minMarkerSize from profile if available
  const minSize = profile.minMarkerSize || 28;
  size = Math.max(minSize, Math.min(60, size));
  
  // Selected markers are slightly larger
  if (isSelected) {
    size *= 1.15;
  }
  
  // Calculate font size proportionally
  const fontSize = Math.max(8, Math.min(12, size * 0.28));
  
  // Only show full label if marker is large enough or selected
  const showLabel = size >= 32 || isSelected;
  
  return { size, fontSize, showLabel, profile };
};

export default {
  MAP_DENSITY_PROFILES,
  getMapProfile,
  resolveOverlaps,
  adaptiveCluster,
  processCalloutsForDisplay,
  calculateMarkerSize
};
