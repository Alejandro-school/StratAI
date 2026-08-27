import { describe, expect, it } from 'vitest';
import { getTacticalMapBootstrap, selectInitialTacticalMap } from './tacticalMapBootstrap';

const supportedMaps = [
  { id: 'de_dust2' },
  { id: 'de_nuke' },
];

describe('tactical map bootstrap', () => {
  it('selects the first map in match order that the tactical map supports', () => {
    const matches = [
      { match_id: 'cache-match', map_name: 'de_cache' },
      { match_id: 'nuke-match', map_name: 'de_nuke' },
    ];

    expect(selectInitialTacticalMap(matches, supportedMaps)).toBe('de_nuke');
  });

  it('defers route canonicalization until the inferred map has been applied', () => {
    expect(getTacticalMapBootstrap({
      currentMap: 'de_dust2',
      hasExplicitMap: false,
      isLoading: false,
      matches: [{ map_name: 'de_nuke' }],
      supportedMaps,
    })).toEqual({
      targetMap: 'de_nuke',
      deferRouteWrite: true,
    });

    expect(getTacticalMapBootstrap({
      currentMap: 'de_nuke',
      hasExplicitMap: false,
      isLoading: false,
      matches: [{ map_name: 'de_nuke' }],
      supportedMaps,
    })).toEqual({
      targetMap: null,
      deferRouteWrite: false,
    });
  });

  it('never replaces or delays an explicit map deep link', () => {
    expect(getTacticalMapBootstrap({
      currentMap: 'de_mirage',
      hasExplicitMap: true,
      isLoading: true,
      matches: [{ map_name: 'de_nuke' }],
      supportedMaps,
    })).toEqual({
      targetMap: null,
      deferRouteWrite: false,
    });
  });
});
