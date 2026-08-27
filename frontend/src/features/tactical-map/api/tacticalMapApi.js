import apiClient from '../../../lib/apiClient';

export const tacticalMapApi = {
  getCallouts: (mapName, { signal } = {}) => apiClient.get('/steam/get-callout-stats', {
    params: { map_name: mapName },
    signal,
  }),
  getGrenades: (mapName, { signal } = {}) => apiClient.get('/steam/get-aggregate-grenades', {
    params: { map_name: mapName },
    signal,
  }),
  getMovement: (mapName, { signal } = {}) => apiClient.get('/steam/get-movement-stats', {
    params: { map_name: mapName },
    signal,
  }),
};

export default tacticalMapApi;
