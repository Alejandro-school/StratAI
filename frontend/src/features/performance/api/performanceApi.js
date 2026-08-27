import apiClient from '../../../lib/apiClient';

export const performanceApi = {
  getOverview: ({ mapName, limit } = {}) => apiClient.get('/steam/performance-overview', {
    params: { map_name: mapName, limit },
  }),
  getPlayerStats: (steamId, { mapName, limit } = {}) => (
    apiClient.get(`/steam/player-stats/${encodeURIComponent(steamId)}`, {
      params: { map_name: mapName, limit },
    })
  ),
  searchPlayers: (query = '') => apiClient.get('/steam/player-search', {
    params: { q: query },
  }),
};

export default performanceApi;
