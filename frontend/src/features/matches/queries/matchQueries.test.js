import { QueryClient } from '@tanstack/react-query';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { matchesQueryOptions } from './matchQueries';

const response = (payload) => ({
  ok: true,
  status: 200,
  json: vi.fn().mockResolvedValue(payload),
});

describe('matchesQueryOptions', () => {
  afterEach(() => vi.restoreAllMocks());

  it('deduplicates concurrent consumers and reuses fresh match DTOs', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(response({
      matches: [{ match_id: 'match-1', map_name: 'de_nuke' }],
    }));
    const client = new QueryClient();
    const options = matchesQueryOptions('steam:1');

    const [historyMatches, coachMatches] = await Promise.all([
      client.fetchQuery(options),
      client.fetchQuery(options),
    ]);
    const cachedMatches = await client.fetchQuery(options);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(historyMatches).toEqual(coachMatches);
    expect(cachedMatches).toEqual([{ match_id: 'match-1', map_name: 'de_nuke' }]);
    client.clear();
  });
});
