import { describe, expect, it } from 'vitest';
import { formatCalloutName, groupSignalsByPosition } from './tacticalPresentation';

describe('tactical presentation helpers', () => {
  it.each([
    ['ExtendedA', 'Extended A'],
    ['TopofMid', 'Top of Mid'],
    ['CTSpawn', 'CT Spawn'],
    ['outside_long', 'outside long'],
  ])('normalizes %s without changing the underlying identifier', (raw, expected) => {
    expect(formatCalloutName(raw)).toBe(expected);
  });

  it('groups equal coordinates and assigns separated display positions', () => {
    const groups = groupSignalsByPosition([
      { id: 'strength:a', position: { x: 50, y: 50 } },
      { id: 'risk:a', position: { x: 50, y: 50 } },
      { id: 'habit:b', position: { x: 20, y: 30 } },
    ]);

    expect(groups).toHaveLength(2);
    expect(groups[0].signals).toHaveLength(2);
    expect(groups[0].signals[0].displayPosition)
      .not.toEqual(groups[0].signals[1].displayPosition);
    expect(groups[1].signals[0].displayPosition).toEqual({ x: 20, y: 30 });
  });
});
