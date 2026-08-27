import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import tacticalMapApi from '../api/tacticalMapApi';

const EMPTY_ARRAY = Object.freeze([]);
const EMPTY_OBJECT = Object.freeze({});
const EMPTY_GRENADE_DATA = Object.freeze({ smoke: [], flash: [], he: [], molotov: [] });
const EMPTY_SIDE_STATS = Object.freeze({ CT: null, T: null });
const EMPTY_MOVEMENT_METRICS = Object.freeze({
  avg_time_to_a: { ct: null, t: null },
  avg_time_to_b: { ct: null, t: null },
  top_positions: [],
  total_rounds: 0,
  total_samples: 0,
});

export const tacticalKeys = {
  all: ['tactical-map'],
  callouts: (mapName) => ['tactical-map', mapName, 'callouts'],
  grenades: (mapName) => ['tactical-map', mapName, 'grenades'],
  movement: (mapName) => ['tactical-map', mapName, 'movement'],
};

const tacticalQuery = ({ queryKey, queryFn, enabled }) => ({
  queryKey,
  queryFn,
  enabled,
  staleTime: 10 * 60 * 1000,
});

export const useCalloutStats = (mapName = 'de_dust2', { enabled = true } = {}) => {
  const query = useQuery(tacticalQuery({
    queryKey: tacticalKeys.callouts(mapName),
    queryFn: ({ signal }) => tacticalMapApi.getCallouts(mapName, { signal }),
    enabled: enabled && Boolean(mapName),
  }));
  const calloutStats = query.data?.callouts || EMPTY_OBJECT;
  const sortedCallouts = useMemo(() => (
    Object.entries(calloutStats)
      .map(([name, stats]) => ({ name, ...stats }))
      .sort((left, right) => right.sample_size - left.sample_size)
  ), [calloutStats]);

  return {
    calloutStats,
    sortedCallouts,
    heatmapData: query.data?.heatmap_data || EMPTY_ARRAY,
    matchesAnalyzed: query.data?.matches_analyzed || 0,
    sideStats: query.data?.side_stats || EMPTY_SIDE_STATS,
    bestCallout: sortedCallouts.find(({ rating }) => rating === 'good'),
    worstCallout: sortedCallouts.find(({ rating }) => rating === 'bad'),
    loading: query.isLoading,
    error: query.error?.message || null,
    refetch: query.refetch,
  };
};

export const useGrenadeStats = (mapName = 'de_dust2', { enabled = true } = {}) => {
  const query = useQuery(tacticalQuery({
    queryKey: tacticalKeys.grenades(mapName),
    queryFn: ({ signal }) => tacticalMapApi.getGrenades(mapName, { signal }),
    enabled: enabled && Boolean(mapName),
  }));
  const summary = query.data?.summary || EMPTY_OBJECT;
  const totalGrenades = useMemo(() => (
    Object.values(summary).reduce((total, stats) => total + (stats?.thrown || 0), 0)
  ), [summary]);

  return {
    grenadeData: query.data?.by_type || EMPTY_GRENADE_DATA,
    summary,
    insights: query.data?.insights || EMPTY_ARRAY,
    matchesAnalyzed: query.data?.matches_analyzed || 0,
    totalGrenades,
    loading: query.isLoading,
    error: query.error?.message || null,
    refetch: query.refetch,
  };
};

export const useMovementStats = (mapName = 'de_dust2', { enabled = true } = {}) => {
  const query = useQuery(tacticalQuery({
    queryKey: tacticalKeys.movement(mapName),
    queryFn: ({ signal }) => tacticalMapApi.getMovement(mapName, { signal }),
    enabled: enabled && Boolean(mapName),
  }));

  return {
    heatmapGrid: query.data?.heatmap_grid || EMPTY_ARRAY,
    flowLines: query.data?.flow_lines || EMPTY_ARRAY,
    metrics: query.data?.metrics || EMPTY_MOVEMENT_METRICS,
    matchesAnalyzed: query.data?.matches_analyzed || 0,
    loading: query.isLoading,
    error: query.error?.message || null,
    refetch: query.refetch,
  };
};
