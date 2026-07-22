// frontend/src/hooks/useMovementStats.js
// Hook for fetching movement heatmap and flow data (React Query)

import { useQuery } from '@tanstack/react-query';
import { API_URL } from '../utils/api';

const EMPTY_METRICS = {
  avg_time_to_a: { ct: null, t: null },
  avg_time_to_b: { ct: null, t: null },
  top_positions: [],
  total_rounds: 0,
  total_samples: 0,
};

const fetchMovementStats = async (mapName) => {
  const response = await fetch(
    `${API_URL}/steam/get-movement-stats?map_name=${mapName}`,
    { credentials: 'include' }
  );
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
};

/**
 * Hook to fetch movement statistics for hybrid flow + heatmap visualization
 * @param {string} mapName - Map name (e.g. 'de_dust2')
 */
export const useMovementStats = (mapName = 'de_dust2') => {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['movement-stats', mapName],
    queryFn: () => fetchMovementStats(mapName),
    enabled: !!mapName,
  });

  return {
    heatmapGrid: data?.heatmap_grid || [],
    flowLines: data?.flow_lines || [],
    metrics: data?.metrics || EMPTY_METRICS,
    matchesAnalyzed: data?.matches_analyzed || 0,
    loading: isLoading,
    error: error?.message || null,
    refetch,
  };
};

export default useMovementStats;
