import React from 'react';
import '@testing-library/jest-dom';
import { fireEvent, screen, render } from '@testing-library/react';
import Performance from './Performance';

jest.mock('../components/Layout/NavigationFrame', () => ({ children }) => (
  <div data-testid="navigation-frame">{children}</div>
));

jest.mock('../context/UserContext', () => ({
  useUser: () => ({ user: { steamid: '123' } }),
}));

jest.mock('../hooks/usePerformanceData', () => ({
  usePerformanceData: jest.fn(),
}));

const { usePerformanceData } = require('../hooks/usePerformanceData');

beforeAll(() => {
  global.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

const performancePayload = {
  overview: {
    total_matches: 12,
    wins: 7,
    losses: 5,
    win_rate: 58.3,
    kills: 240,
    deaths: 190,
    assists: 54,
    total_damage: 9600,
    kd_ratio: 1.26,
    adr: 83.2,
    hs_pct: 46,
    kast: 72,
    hltv_rating: 1.12,
    impact_rating: 1.2,
  },
  sides: { ct_rating: 1.16, t_rating: 1.05, ct_adr: 86, t_adr: 80 },
  aim: {
    accuracy_overall: 28,
    time_to_damage_avg_ms: 440,
    crosshair_placement_avg_error: 6.2,
    crosshair_placement_peek: 7,
    crosshair_placement_hold: 5.8,
    shots_fired: 1000,
    shots_hit: 280,
    body_part_hits: { head: 110, chest: 90, stomach: 40, left_arm: 10, right_arm: 10, left_leg: 8, right_leg: 12 },
  },
  combat: {
    opening_duels_attempted: 40,
    opening_duels_won: 23,
    opening_duels_lost: 17,
    opening_success_rate: 57.5,
    trade_kills: 30,
    traded_deaths: 22,
    flash_assists: 8,
    clutches: { '1v1': 3, '1v2': 1, '1v3': 0, '1v4': 0, '1v5': 0 },
    multikills: { '2k': 16, '3k': 4, '4k': 1, ace: 0 },
  },
  utility: {
    grenades_thrown_total: 180,
    flashes_thrown: 60,
    enemies_flashed_total: 82,
    flash_duration_total: 120,
    enemies_flashed_per_flash: 1.36,
    blind_time_per_flash: 2,
    he_thrown: 30,
    he_damage_per_nade: 19,
    molotovs_thrown: 24,
    molotov_damage_per_nade: 11,
    smokes_thrown: 42,
    utility_damage: 920,
    grenade_damage: { he: 570, molotov: 264, flash: 0, smoke: 0 },
  },
  weapons: [
    { weapon: 'ak47', kills: 90, damage: 3400, accuracy: 31, hs_pct: 55 },
    { weapon: 'm4a1', kills: 60, damage: 2400, accuracy: 29, hs_pct: 41 },
  ],
  maps: [
    { map: 'de_mirage', wins: 4, losses: 1, win_rate: 80, avg_kd: 1.4, avg_adr: 90, avg_rating: 1.26, matches: 5, sides: { ct_rating: 1.3, t_rating: 1.1, ct_adr: 92, t_adr: 84, strongest_side: 'CT' }, top_weapons: [] },
  ],
  match_history: [
    { match_id: 'm1', map: 'de_mirage', result: 'W', final_score: '13-9', kills: 22, deaths: 14, assists: 4, kd_ratio: 1.57, adr: 96, hltv_rating: 1.33, hs_percentage: 50, accuracy_overall: 30, date: '2026-01-01' },
  ],
  economy: { rounds_survived: 140, total_rounds: 260, survival_rate: 53.8 },
};

describe('Performance redesign', () => {
  it('renders the command center and switches to specialized briefings', () => {
    usePerformanceData.mockReturnValue({
      loading: false,
      error: null,
      performance: performancePayload,
      retry: jest.fn(),
    });

    render(<Performance />);

    expect(screen.getByText('Performance Command Center')).toBeInTheDocument();
    expect(screen.getByText('Lectura priorizada')).toBeInTheDocument();
    expect(screen.getByText('13-9')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('tab', { name: /Combate/i }));
    expect(screen.getByText('Iniciativa, trades y cierre de ronda')).toBeInTheDocument();
  });

  it('renders loading and error states', () => {
    usePerformanceData.mockReturnValue({ loading: true });
    const { rerender } = render(<Performance />);
    expect(document.querySelector('.p-skeleton-view')).toBeInTheDocument();

    usePerformanceData.mockReturnValue({
      loading: false,
      error: 'boom',
      performance: null,
      retry: jest.fn(),
    });
    rerender(<Performance />);
    expect(screen.getByText('Error cargando datos')).toBeInTheDocument();
  });
});
