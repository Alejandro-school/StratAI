// frontend/src/utils/mapCoordinates.js
// Utilities for converting CS2 game coordinates to radar image positions

/**
 * Map radar configuration from game files
 * origin_x, origin_y: World coordinates of top-left corner of radar
 * scale: Units per pixel (radar images are 1024x1024)
 */
export const MAP_RADAR_CONFIG = {
  de_dust2: { origin_x: -2476, origin_y: 3239, scale: 4.4 },
  de_mirage: { origin_x: -3230, origin_y: 1713, scale: 5.0 },
  de_inferno: { origin_x: -2087, origin_y: 3870, scale: 4.9 },
  de_nuke: { origin_x: -3453, origin_y: 2887, scale: 7.0 },
  de_overpass: { origin_x: -4831, origin_y: 1781, scale: 5.2 },
  de_ancient: { origin_x: -2953, origin_y: 2164, scale: 5.0 },
  de_anubis: { origin_x: -2796, origin_y: 3328, scale: 5.22 },
  de_vertigo: { origin_x: -3168, origin_y: 1762, scale: 4.0 },
  de_train: { origin_x: -2477, origin_y: 2392, scale: 4.7 },
};

/**
 * Convert game world coordinates to radar percentage (0-100)
 * @param {number} gameX - X coordinate in game units
 * @param {number} gameY - Y coordinate in game units
 * @param {string} mapName - Map identifier (e.g. 'de_dust2')
 * @returns {{x: number, y: number}} Percentage position on radar
 */
export const gameToRadarPercent = (gameX, gameY, mapName) => {
  const config = MAP_RADAR_CONFIG[mapName] || MAP_RADAR_CONFIG.de_dust2;
  
  // Convert game coords to pixel position, then to percentage
  const xPercent = ((gameX - config.origin_x) / config.scale) / 1024 * 100;
  const yPercent = ((config.origin_y - gameY) / config.scale) / 1024 * 100; // Y is inverted
  
  return {
    x: Math.max(0, Math.min(100, xPercent)),
    y: Math.max(0, Math.min(100, yPercent))
  };
};

/**
 * Convert radar percentage to CSS position
 * @param {number} percentX - X as percentage (0-100)
 * @param {number} percentY - Y as percentage (0-100)
 * @returns {{left: string, top: string}} CSS position values
 */
export const percentToCSS = (percentX, percentY) => ({
  left: `${percentX}%`,
  top: `${percentY}%`
});

/**
 * Calculate distance between two radar positions
 * @param {{x: number, y: number}} pos1 
 * @param {{x: number, y: number}} pos2 
 * @returns {number} Distance in percentage units
 */
export const radarDistance = (pos1, pos2) => {
  const dx = pos1.x - pos2.x;
  const dy = pos1.y - pos2.y;
  return Math.sqrt(dx * dx + dy * dy);
};

/**
 * Get color for heatmap point based on type
 * @param {'kill' | 'death'} type 
 * @param {number} opacity 
 * @returns {string} RGBA color string
 */
export const getHeatmapColor = (type, opacity = 0.6) => {
  if (type === 'kill') {
    return `rgba(34, 197, 94, ${opacity})`; // Green
  }
  return `rgba(239, 68, 68, ${opacity})`; // Red
};

/**
 * Group heatmap points by proximity for clustering
 * @param {Array} points - Array of {x, y, type} objects
 * @param {number} threshold - Distance threshold for grouping (in %)
 * @returns {Array} Clustered points with count
 */
export const clusterHeatmapPoints = (points, threshold = 3) => {
  const clusters = [];
  const used = new Set();
  
  points.forEach((point, i) => {
    if (used.has(i)) return;
    
    const cluster = {
      x: point.x,
      y: point.y,
      kills: point.type === 'kill' ? 1 : 0,
      deaths: point.type === 'death' ? 1 : 0,
      callout: point.callout
    };
    used.add(i);
    
    // Find nearby points
    points.forEach((other, j) => {
      if (used.has(j)) return;
      if (radarDistance(point, other) <= threshold) {
        if (other.type === 'kill') cluster.kills++;
        else cluster.deaths++;
        used.add(j);
      }
    });
    
    cluster.total = cluster.kills + cluster.deaths;
    cluster.ratio = cluster.kills / (cluster.total || 1);
    clusters.push(cluster);
  });
  
  return clusters;
};

export default {
  MAP_RADAR_CONFIG,
  gameToRadarPercent,
  percentToCSS,
  radarDistance,
  getHeatmapColor,
  clusterHeatmapPoints
};
