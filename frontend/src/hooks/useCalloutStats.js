// frontend/src/hooks/useCalloutStats.js
// Hook for fetching granular per-callout statistics (React Query)

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { API_URL } from '../utils/api';

const fetchCalloutStats = async (mapName) => {
  const response = await fetch(
    `${API_URL}/steam/get-callout-stats?map_name=${mapName}`,
    { credentials: 'include' }
  );
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
};

/**
 * Hook to fetch granular callout statistics for the interactive map
 * @param {string} mapName - Map name (e.g. 'de_dust2')
 */
export const useCalloutStats = (mapName = 'de_dust2') => {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['callout-stats', mapName],
    queryFn: () => fetchCalloutStats(mapName),
    enabled: !!mapName,
  });

  const calloutStats = useMemo(() => data?.callouts || {}, [data]);
  const heatmapData = data?.heatmap_data || [];
  const matchesAnalyzed = data?.matches_analyzed || 0;
  const sideStats = data?.side_stats || { CT: null, T: null };

  const sortedCallouts = useMemo(() =>
    Object.entries(calloutStats)
      .map(([name, stats]) => ({ name, ...stats }))
      .sort((a, b) => b.sample_size - a.sample_size),
    [calloutStats]
  );

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
    loading: isLoading,
    error: error?.message || null,
    refetch,
  };
};

export default useCalloutStats;
