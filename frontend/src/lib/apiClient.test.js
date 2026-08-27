import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, apiClient } from './apiClient';

const response = (payload, status = 200) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(payload),
});

describe('apiClient', () => {
  afterEach(() => vi.restoreAllMocks());

  it('serializes params and JSON bodies with authenticated requests', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(response({ ok: true }));

    await apiClient.post('/resource', { name: 'demo' }, {
      params: { map_name: 'de_nuke', empty: null },
    });

    const [url, options] = fetchMock.mock.calls[0];
    expect(new URL(url).searchParams.get('map_name')).toBe('de_nuke');
    expect(new URL(url).searchParams.has('empty')).toBe(false);
    expect(options).toMatchObject({
      method: 'POST',
      credentials: 'include',
      body: JSON.stringify({ name: 'demo' }),
      headers: { 'Content-Type': 'application/json' },
    });
  });

  it('exposes the HTTP status and server payload on errors', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(response({ detail: 'No autorizado' }, 401));

    await expect(apiClient.get('/private')).rejects.toMatchObject({
      name: 'ApiError',
      message: 'No autorizado',
      status: 401,
      payload: { detail: 'No autorizado' },
    });
    await apiClient.get('/private').catch((error) => expect(error).toBeInstanceOf(ApiError));
  });
});
