import { API_URL } from '../utils/api';

export class ApiError extends Error {
  constructor(message, status, payload) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.payload = payload;
  }
}

const buildUrl = (path, params) => {
  const origin = typeof window === 'undefined' ? 'http://localhost' : window.location.origin;
  const url = new URL(`${API_URL}${path}`, origin);

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      url.searchParams.set(key, String(value));
    }
  });

  return url.toString();
};

const parseResponse = async (response) => {
  if (response.status === 204) return null;
  return response.json().catch(() => null);
};

export const apiRequest = async (path, options = {}) => {
  const { body, headers, params, ...requestOptions } = options;
  const isFormData = typeof FormData !== 'undefined' && body instanceof FormData;
  const hasJsonBody = body !== undefined && !isFormData;
  const response = await fetch(buildUrl(path, params), {
    credentials: 'include',
    ...requestOptions,
    ...(body !== undefined ? { body: hasJsonBody ? JSON.stringify(body) : body } : {}),
    headers: {
      ...(hasJsonBody ? { 'Content-Type': 'application/json' } : {}),
      ...headers,
    },
  });
  const payload = await parseResponse(response);

  if (!response.ok || payload?.error) {
    const message = payload?.detail || payload?.error || `HTTP ${response.status}`;
    throw new ApiError(message, response.status, payload);
  }

  return payload;
};

export const apiClient = {
  get: (path, options) => apiRequest(path, { ...options, method: 'GET' }),
  post: (path, body, options) => apiRequest(path, { ...options, method: 'POST', body }),
  patch: (path, body, options) => apiRequest(path, { ...options, method: 'PATCH', body }),
  delete: (path, options) => apiRequest(path, { ...options, method: 'DELETE' }),
};

export default apiClient;
