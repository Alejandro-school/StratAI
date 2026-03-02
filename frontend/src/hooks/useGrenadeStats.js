// frontend/src/hooks/useGrenadeStats.js
// Hook for fetching aggregate grenade statistics per map

import { useState, useEffect, useCallback } from 'react';
import { API_URL } from '../utils/api';

/**
 * Hook to fetch aggregate grenade statistics for the grenade heatmap
 * @param {string} mapName - Map name (e.g. 'de_dust2')
 * @param {Object} options - hook options
 * @param {boolean} options.enabled - if false, skip fetch and expose empty defaults
 * @returns {Object} { grenadeData, summary, insights, matchesAnalyzed, loading, error, refetch }
 */
export const useGrenadeStats = (mapName = 'de_dust2', options = {}) => {
  const { enabled = true } = options;
  const [grenadeData, setGrenadeData] = useState({
    smoke: [],
    flash: [],
    he: [],
    molotov: []
  });
  const [summary, setSummary] = useState({});
  const [insights, setInsights] = useState([]);
  const [matchesAnalyzed, setMatchesAnalyzed] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchGrenadeStats = useCallback(async () => {
    if (!enabled || !mapName) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(
        `${API_URL}/steam/get-aggregate-grenades?map_name=${mapName}`,
        { credentials: 'include' }
      );
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      setGrenadeData(data.by_type || { smoke: [], flash: [], he: [], molotov: [] });
      setSummary(data.summary || {});
      setInsights(data.insights || []);
      setMatchesAnalyzed(data.matches_analyzed || 0);
    } catch (err) {
      console.error('[useGrenadeStats] Error:', err);
      setError(err.message);
      setGrenadeData({ smoke: [], flash: [], he: [], molotov: [] });
      setSummary({});
      setInsights([]);
    } finally {
      setLoading(false);
    }
  }, [enabled, mapName]);

  useEffect(() => {
    if (!enabled) {
      setLoading(false);
      return;
    }
    fetchGrenadeStats();
  }, [enabled, fetchGrenadeStats]);

  // Computed values
  const totalGrenades = Object.values(summary).reduce(
    (sum, s) => sum + (s?.thrown || 0), 0
  );

  return {
    grenadeData,
    summary,
    insights,
    matchesAnalyzed,
    totalGrenades,
    loading,
    error,
    refetch: fetchGrenadeStats
  };
};

export default useGrenadeStats;
