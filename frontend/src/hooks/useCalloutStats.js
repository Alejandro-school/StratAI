// frontend/src/hooks/useCalloutStats.js
// Hook for fetching granular per-callout statistics

import { useState, useEffect, useCallback } from 'react';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

/**
 * Hook to fetch granular callout statistics for the interactive map
 * @param {string} mapName - Map name (e.g. 'de_dust2')
 * @returns {Object} { calloutStats, heatmapData, matchesAnalyzed, loading, error, refetch }
 */
export const useCalloutStats = (mapName = 'de_dust2') => {
  const [calloutStats, setCalloutStats] = useState({});
  const [heatmapData, setHeatmapData] = useState([]);
  const [matchesAnalyzed, setMatchesAnalyzed] = useState(0);
  const [sideStats, setSideStats] = useState({ CT: null, T: null });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchCalloutStats = useCallback(async () => {
    if (!mapName) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(
        `${API_URL}/steam/get-callout-stats?map_name=${mapName}`,
        { credentials: 'include' }
      );
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      setCalloutStats(data.callouts || {});
      setHeatmapData(data.heatmap_data || []);
      setMatchesAnalyzed(data.matches_analyzed || 0);
      setSideStats(data.side_stats || { CT: null, T: null });
    } catch (err) {
      console.error('[useCalloutStats] Error:', err);
      setError(err.message);
      setCalloutStats({});
      setHeatmapData([]);
      setSideStats({ CT: null, T: null });
    } finally {
      setLoading(false);
    }
  }, [mapName]);

  useEffect(() => {
    fetchCalloutStats();
  }, [fetchCalloutStats]);

  // Computed values
  const sortedCallouts = Object.entries(calloutStats)
    .map(([name, stats]) => ({ name, ...stats }))
    .sort((a, b) => b.sample_size - a.sample_size);

  const bestCallout = sortedCallouts.find(c => c.rating === 'good');
  const worstCallout = sortedCallouts.find(c => c.rating === 'bad');

  return {
    calloutStats,
    sortedCallouts,
    heatmapData,
    matchesAnalyzed,
    sideStats,
    bestCallout,
    worstCallout,
    loading,
    error,
    refetch: fetchCalloutStats
  };
};

export default useCalloutStats;
