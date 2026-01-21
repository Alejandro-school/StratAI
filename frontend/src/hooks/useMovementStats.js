// frontend/src/hooks/useMovementStats.js
// Hook for fetching movement heatmap and flow data

import { useState, useEffect, useCallback } from 'react';

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

/**
 * Hook to fetch movement statistics for hybrid flow + heatmap visualization
 * @param {string} mapName - Map name (e.g. 'de_dust2')
 * @returns {Object} { heatmapGrid, flowLines, metrics, matchesAnalyzed, loading, error, refetch }
 */
export const useMovementStats = (mapName = 'de_dust2') => {
  const [heatmapGrid, setHeatmapGrid] = useState([]);
  const [flowLines, setFlowLines] = useState([]);
  const [metrics, setMetrics] = useState({
    avg_time_to_a: { ct: null, t: null },
    avg_time_to_b: { ct: null, t: null },
    top_positions: [],
    total_rounds: 0,
    total_samples: 0
  });
  const [matchesAnalyzed, setMatchesAnalyzed] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchMovementStats = useCallback(async () => {
    if (!mapName) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(
        `${API_URL}/steam/get-movement-stats?map_name=${mapName}`,
        { credentials: 'include' }
      );
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      setHeatmapGrid(data.heatmap_grid || []);
      setFlowLines(data.flow_lines || []);
      setMetrics(data.metrics || {
        avg_time_to_a: { ct: null, t: null },
        avg_time_to_b: { ct: null, t: null },
        top_positions: [],
        total_rounds: 0,
        total_samples: 0
      });
      setMatchesAnalyzed(data.matches_analyzed || 0);
    } catch (err) {
      console.error('[useMovementStats] Error:', err);
      setError(err.message);
      setHeatmapGrid([]);
      setFlowLines([]);
    } finally {
      setLoading(false);
    }
  }, [mapName]);

  useEffect(() => {
    fetchMovementStats();
  }, [fetchMovementStats]);

  return {
    heatmapGrid,
    flowLines,
    metrics,
    matchesAnalyzed,
    loading,
    error,
    refetch: fetchMovementStats
  };
};

export default useMovementStats;
