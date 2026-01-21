// frontend/src/hooks/useMapZoneStats.js
// Hook for fetching zone-based combat stats for the map dashboard

import { useState, useEffect, useCallback } from 'react';

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

/**
 * Hook to fetch combat statistics per map zone
 * @param {string} mapName - Map name (e.g. 'de_dust2')
 * @returns {Object} { zoneStats, loading, error, refetch }
 */
export const useMapZoneStats = (mapName = 'de_dust2') => {
  const [zoneStats, setZoneStats] = useState(null);
  const [matchesAnalyzed, setMatchesAnalyzed] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchZoneStats = useCallback(async () => {
    if (!mapName) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(
        `${API_URL}/steam/get-map-zone-stats?map_name=${mapName}`,
        { credentials: 'include' }
      );
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      setZoneStats(data.zones || {});
      setMatchesAnalyzed(data.matches_analyzed || 0);
    } catch (err) {
      console.error('[useMapZoneStats] Error:', err);
      setError(err.message);
      // Set empty default stats so UI doesn't break
      setZoneStats({});
    } finally {
      setLoading(false);
    }
  }, [mapName]);

  useEffect(() => {
    fetchZoneStats();
  }, [fetchZoneStats]);

  return {
    zoneStats,
    matchesAnalyzed,
    loading,
    error,
    refetch: fetchZoneStats
  };
};

export default useMapZoneStats;
