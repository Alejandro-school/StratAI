import { queryOptions } from '@tanstack/react-query';
import { replayApi } from '../api/replayApi';

export const replayKeys = {
  all: ['replay'],
  metadata: (matchId) => ['replay', matchId, 'metadata'],
  round: (matchId, round) => ['replay', matchId, 'round', round],
  annotations: (matchId) => ['replay', matchId, 'annotations'],
};

export const replayMetadataQueryOptions = (matchId) => queryOptions({
  queryKey: replayKeys.metadata(matchId),
  queryFn: ({ signal }) => replayApi.metadata(matchId, signal),
  staleTime: Infinity,
});

export const replayRoundQueryOptions = (matchId, round) => queryOptions({
  queryKey: replayKeys.round(matchId, round),
  queryFn: ({ signal }) => replayApi.round(matchId, round, signal),
  staleTime: Infinity,
});

export const replayAnnotationsQueryOptions = (matchId) => queryOptions({
  queryKey: replayKeys.annotations(matchId),
  queryFn: ({ signal }) => replayApi.annotations(matchId, signal),
  enabled: Boolean(matchId),
  staleTime: 30 * 1000,
});
