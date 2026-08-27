import { useQuery } from '@tanstack/react-query';
import performanceApi from '../api/performanceApi';

const QUERY_OPTIONS = {
  staleTime: 2 * 60 * 1000,
  gcTime: 10 * 60 * 1000,
  refetchOnWindowFocus: false,
  retry: 1,
};

export const usePerformanceOverview = ({ mapName, limit, enabled = true } = {}) => (
  useQuery({
    queryKey: ['performance', 'overview', mapName || 'all', limit || 'all'],
    queryFn: () => performanceApi.getOverview({ mapName, limit }),
    enabled,
    ...QUERY_OPTIONS,
  })
);

export const usePerformancePlayer = (steamId, { mapName, limit, enabled = true } = {}) => (
  useQuery({
    queryKey: ['performance', 'player', steamId, mapName || 'all', limit || 'all'],
    queryFn: () => performanceApi.getPlayerStats(steamId, { mapName, limit }),
    enabled: enabled && Boolean(steamId),
    ...QUERY_OPTIONS,
  })
);

export const usePerformancePlayers = (query = '', enabled = true) => (
  useQuery({
    queryKey: ['performance', 'players', query.trim().toLowerCase()],
    queryFn: () => performanceApi.searchPlayers(query.trim()),
    enabled,
    ...QUERY_OPTIONS,
  })
);
