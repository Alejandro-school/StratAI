import { queryOptions } from '@tanstack/react-query';
import matchesApi from '../api/matchesApi';

const MATCH_STALE_TIME = 5 * 60 * 1000;
const MATCH_DETAIL_STALE_TIME = 30 * 60 * 1000;

export const matchKeys = {
  all: ['matches'],
  list: (steamId) => ['matches', 'list', steamId || 'anonymous'],
  detail: (steamId, matchId) => ['matches', 'detail', steamId || 'anonymous', matchId || 'missing'],
};

export const matchesQueryOptions = (steamId) => queryOptions({
  queryKey: matchKeys.list(steamId),
  queryFn: ({ signal }) => matchesApi.getAll({ signal }),
  enabled: Boolean(steamId),
  staleTime: MATCH_STALE_TIME,
});

export const matchDetailsQueryOptions = (matchId, steamId) => queryOptions({
  queryKey: matchKeys.detail(steamId, matchId),
  queryFn: ({ signal }) => matchesApi.getDetails(matchId, { signal }),
  enabled: Boolean(matchId),
  staleTime: MATCH_DETAIL_STALE_TIME,
});
