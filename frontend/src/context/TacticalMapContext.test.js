import { describe, expect, it } from 'vitest';
import { parseTacticalRouteState } from './TacticalMapContext';

describe('parseTacticalRouteState', () => {
  it.each([
    ['overview', 'briefing', 'volume'],
    ['presence', 'positioning', 'volume'],
    ['duels', 'combat', 'volume'],
    ['impact', 'combat', 'impact'],
  ])('migrates legacy lens %s', (legacy, lens, metric) => {
    expect(parseTacticalRouteState(`?lens=${legacy}`)).toMatchObject({
      activeSection: lens,
      combatMetric: metric,
    });
  });

  it('preserves canonical filters and validates the combat metric', () => {
    expect(parseTacticalRouteState('?map=de_mirage&lens=combat&metric=risk&side=ct&zone=TopofMid'))
      .toMatchObject({
        currentMap: 'de_mirage',
        activeSection: 'combat',
        combatMetric: 'risk',
        activeSide: 'ct',
        selectedCallout: 'TopofMid',
      });
  });

  it('defaults to CT and migrates the removed all-sides context', () => {
    expect(parseTacticalRouteState('')).toMatchObject({ activeSide: 'ct' });
    expect(parseTacticalRouteState('?side=all')).toMatchObject({ activeSide: 'ct' });
  });
});
