// frontend/src/hooks/useGrenadeStats.js
// Hook for fetching aggregate grenade statistics per map (React Query)

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { API_URL } from '../utils/api';

const EMPTY_DATA = { smoke: [], flash: [], he: [], molotov: [] };

const fetchGrenadeStats = async (mapName) => {
  const response = await fetch(
    `${API_URL}/steam/get-aggregate-grenades?map_name=${mapName}`,
    { credentials: 'include' }
  );
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
};

/**
 * Hook to fetch aggregate grenade statistics for the grenade heatmap
 * @param {string} mapName - Map name (e.g. 'de_dust2')
 * @param {Object} options - hook options
 * @param {boolean} options.enabled - if false, skip fetch
 */
export const useGrenadeStats = (mapName = 'de_dust2', options = {}) => {
  const { enabled = true } = options;

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['grenade-stats', mapName],
    queryFn: () => fetchGrenadeStats(mapName),
    enabled: enabled && !!mapName,
  });

  const grenadeData = data?.by_type || EMPTY_DATA;
  const summary = useMemo(() => data?.summary || {}, [data]);
  const insights = data?.insights || [];
  const matchesAnalyzed = data?.matches_analyzed || 0;

  const totalGrenades = useMemo(
    () => Object.values(summary).reduce((sum, s) => sum + (s?.thrown || 0), 0),
    [summary]
  );

  return {
    grenadeData,
    summary,
    insights,
    matchesAnalyzed,
    totalGrenades,
    loading: isLoading,
    error: error?.message || null,
    refetch,
  };
};

export default useGrenadeStats;
