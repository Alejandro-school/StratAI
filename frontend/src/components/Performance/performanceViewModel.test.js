import { buildPerformanceViewModel } from './performanceViewModel';

const payload = {
  overview: {
    total_matches: 20,
    wins: 12,
    losses: 8,
    win_rate: 60,
    kills: 420,
    kd_ratio: 1.18,
    adr: 86.4,
    hs_pct: 47,
    kast: 74,
    hltv_rating: 1.14,
    impact_rating: 1.22,
  },
  sides: { ct_rating: 1.18, t_rating: 1.06 },
  aim: {
    accuracy_overall: 29,
    time_to_damage_avg_ms: 430,
    body_part_hits: { head: 120 },
  },
  combat: {
    opening_success_rate: 57,
  },
  utility: {
    enemies_flashed_per_flash: 1.25,
    he_damage_per_nade: 18,
    molotov_damage_per_nade: 12,
  },
  weapons: [
    { weapon: 'ak47', kills: 150, accuracy: 31, hs_pct: 55 },
  ],
  maps: [
    { map: 'de_mirage', win_rate: 66, avg_rating: 1.21, matches: 8 },
    { map: 'de_inferno', win_rate: 45, avg_rating: 0.98, matches: 4 },
  ],
  history: [
    { match_id: '1', result: 'W' },
    { match_id: '2', result: 'W' },
    { match_id: '3', result: 'L' },
  ],
};

describe('buildPerformanceViewModel', () => {
  it('derives tactical summary without changing the payload contract', () => {
    const vm = buildPerformanceViewModel(payload);

    expect(vm.verdict).toBe('Base competitiva solida');
    expect(vm.bestMap.map).toBe('de_mirage');
    expect(vm.form.wins).toBe(2);
    expect(vm.form.losses).toBe(1);
    expect(vm.areas).toHaveLength(5);
    expect(vm.bestArea).toBeTruthy();
    expect(vm.focusArea).toBeTruthy();
  });

  it('handles empty payloads safely', () => {
    const vm = buildPerformanceViewModel({});

    expect(vm.rating).toBe(0);
    expect(vm.form.label).toBe('Sin muestra reciente');
    expect(vm.areas.every((area) => Number.isFinite(area.score))).toBe(true);
    expect(vm.bestMap).toBeNull();
  });
});
