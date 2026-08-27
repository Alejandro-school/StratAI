import { describe, expect, it } from 'vitest';
import {
  formatResetCountdown,
  getMissionPoints,
  getMissionProgress,
  getNextWeeklyReset,
  replaceMission,
} from './progressModel';

const createMission = (overrides = {}) => ({
  id: 'mission',
  baseline: 40,
  current: 50,
  target: 60,
  maxPoints: 400,
  completed: false,
  ...overrides,
});

describe('progressModel', () => {
  it('normalizes direct and inverse metrics', () => {
    expect(getMissionProgress(createMission())).toBe(0.5);
    expect(getMissionProgress(createMission({ baseline: 20, current: 14, target: 8 }))).toBe(0.5);
  });

  it('reserves twenty percent of points for completion', () => {
    expect(getMissionPoints(createMission())).toBe(160);
    expect(getMissionPoints(createMission({ current: 60, completed: true }))).toBe(400);
  });

  it('preserves mission role and maximum points when rerolled', () => {
    const current = [createMission({ id: 'old', role: 'support', maxPoints: 300 })];
    const [replacement] = replaceMission(current, 'old', createMission({ id: 'new', maxPoints: 900 }));
    expect(replacement).toMatchObject({ id: 'new', role: 'support', maxPoints: 300 });
  });

  it('uses Monday at midnight UTC as the weekly reset', () => {
    const now = new Date('2026-08-09T10:30:00.000Z');
    const reset = getNextWeeklyReset(now);
    expect(reset.toISOString()).toBe('2026-08-10T00:00:00.000Z');
    expect(formatResetCountdown(reset, now)).toBe('0d 13h 30m');
  });
});
