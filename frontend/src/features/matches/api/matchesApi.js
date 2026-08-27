import apiClient from '../../../lib/apiClient';

const getMatchList = (payload) => (
  Array.isArray(payload?.matches) ? payload.matches : []
);

export const matchesApi = {
  getAll: async ({ signal } = {}) => getMatchList(
    await apiClient.get('/steam/get-processed-demos', { signal }),
  ),
  getDetails: (matchId, { signal } = {}) => apiClient.get(
    `/steam/get-match-details/${encodeURIComponent(matchId)}`,
    { signal },
  ),
  discover: ({ signal } = {}) => apiClient.post('/steam/discovery', undefined, { signal }),
};

export default matchesApi;
