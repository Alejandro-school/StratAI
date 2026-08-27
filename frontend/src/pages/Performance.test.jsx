import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, vi } from 'vitest';
import Performance from './Performance';

vi.mock('../components/Layout/NavigationFrame', () => ({
  default: ({ children }) => <div data-testid="navigation-frame">{children}</div>,
}));

vi.mock('../features/performance/scene/PerformanceScene', () => ({
  default: () => <div data-testid="performance-scene" />,
}));

const createPayload = (name = 'Kerchak', steamId = '76561198116485358') => ({
  steam_id: steamId,
  player: { steam_id: steamId, name },
  filters: { available_matches: 18, limit: 60, map_name: null },
  overview: {
    total_matches: 18,
    wins: 10,
    losses: 8,
    win_rate: 55.6,
    kills: 406,
    deaths: 246,
    assists: 88,
    headshots: 180,
    total_damage: 48798,
    kd_ratio: 1.65,
    adr: 135.9,
    hs_pct: 44.3,
    kast: 78.3,
    hltv_rating: 1.7,
    impact_rating: 2.1,
  },
  sides: { ct_rating: 1.81, t_rating: 1.58, ct_adr: 141.2, t_adr: 128.4 },
  aim: {
    accuracy_overall: 34.1,
    reaction_time_avg_ms: 412,
    time_to_damage_avg_ms: 521,
    crosshair_placement_avg_error: 8.4,
    shots_fired: 1190,
    shots_hit: 406,
    body_part_hits: { head: 180, chest: 150, stomach: 30, left_arm: 15, right_arm: 15, left_leg: 8, right_leg: 8 },
  },
  combat: {
    opening_duels_attempted: 93,
    opening_duels_won: 54,
    opening_duels_lost: 39,
    opening_success_rate: 58.1,
    trade_kills: 62,
    traded_deaths: 41,
    flash_assists: 24,
    clutches: { '1v1': 8, '1v2': 3, '1v3': 1, '1v4': 0, '1v5': 0 },
    multikills: { '2k': 48, '3k': 15, '4k': 4, ace: 1 },
  },
  duels: {
    total: 210,
    kills_won: 118,
    kills_lost: 82,
    win_rate: 59,
    encounters: [{
      name: 'Rival Uno',
      wins: 7,
      losses: 5,
      user_weapon: 'AK-47',
      rival_weapon: 'M4A1-S',
      user_shots: 42,
      rival_shots: 39,
      user_accuracy: 35,
      rival_accuracy: 31,
      user_first_damage_ms: 420,
      rival_first_damage_ms: 470,
      area: 'Middle',
      openings: 2,
      through_smoke: 1,
      wallbangs: 0,
      trades: 1,
      user_blind: 0,
    }],
  },
  mechanics: {
    engagements: 210,
    reaction_time_avg_ms: 412,
    time_to_first_damage_avg_ms: 521,
    crosshair_error_avg: 8.4,
    accuracy: 35,
    shots: 600,
    hits: 210,
    stationary_pct: 76,
    moving_pct: 24,
    ducking_pct: 13,
    blind_pct: 5,
    through_smoke_pct: 7,
    wallbang_pct: 2,
    hold_pct: 62,
    peek_pct: 38,
  },
  utility: {
    grenades_thrown_total: 260,
    flashes_thrown: 92,
    enemies_flashed_total: 126,
    enemies_flashed_per_flash: 1.37,
    blind_time_per_flash: 2.8,
    he_thrown: 61,
    he_damage_per_nade: 24.2,
    molotovs_thrown: 47,
    molotov_damage_per_nade: 18.1,
    smokes_thrown: 60,
    utility_damage: 2630,
    grenade_damage: { he: 1476, molotov: 851, flash: 0, smoke: 0 },
  },
  weapons: [
    { weapon: 'AK-47', kills: 180, headshots: 95, damage: 18500, shots_fired: 590, shots_hit: 185, accuracy: 31.4, hs_pct: 52.8 },
    { weapon: 'M4A1-S', kills: 110, headshots: 44, damage: 12100, shots_fired: 360, shots_hit: 126, accuracy: 35, hs_pct: 40 },
    { weapon: 'AWP', kills: 55, headshots: 4, damage: 6800, shots_fired: 120, shots_hit: 58, accuracy: 48.3, hs_pct: 7.3 },
  ],
  maps: [
    { map: 'de_mirage', matches: 6, wins: 4, losses: 2, win_rate: 66.7, avg_kd: 1.4, avg_adr: 132, avg_kast: 78, avg_rating: 1.62, avg_impact: 1.9, sides: { ct_rating: 1.7, t_rating: 1.54, ct_adr: 137, t_adr: 126 } },
    { map: 'de_ancient', matches: 4, wins: 2, losses: 2, win_rate: 50, avg_kd: 1.2, avg_adr: 118, avg_kast: 74, avg_rating: 1.31, avg_impact: 1.5, sides: { ct_rating: 1.4, t_rating: 1.22, ct_adr: 124, t_adr: 112 } },
    { map: 'de_cache', matches: 3, wins: 2, losses: 1, win_rate: 66.7, avg_kd: 1.1, avg_adr: 106, avg_kast: 71, avg_rating: 1.18, avg_impact: 1.3, sides: { ct_rating: 1.22, t_rating: 1.14, ct_adr: 111, t_adr: 101 } },
  ],
  match_history: [
    { match_id: '1', result: 'W', hltv_rating: 1.8, kills: 29, map: 'de_mirage' },
    { match_id: '2', result: 'L', hltv_rating: 1.5, kills: 21, map: 'de_cache' },
    { match_id: '3', result: 'W', hltv_rating: 1.72, kills: 27, map: 'de_ancient' },
    { match_id: '4', result: 'W', hltv_rating: 1.61, kills: 25, map: 'de_mirage' },
  ],
  economy: {
    total_rounds: 359,
    rounds: 359,
    survived_rounds: 116,
    survival_rate: 32.3,
    total_spent: 984950,
    avg_spent_per_round: 2743.6,
    avg_equipment_value: 4201.1,
    avg_money_after_round: 5216.3,
    saved_equipment_value: 597950,
    save_conversion_rate: 65.1,
    team_money_gini: 0.096,
    buy_types: [
      { type: 'full_buy', rounds: 217, share: 60.4, win_rate: 57.1 },
      { type: 'partial_buy', rounds: 6, share: 1.7, win_rate: 66.7 },
      { type: 'eco', rounds: 89, share: 24.8, win_rate: 31.5 },
      { type: 'force_buy', rounds: 47, share: 13.1, win_rate: 48.9 },
    ],
  },
  sources: { summary_matches: 18, combat_matches: 18, economy_matches: 18 },
});

const renderPerformance = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <Performance />
    </QueryClientProvider>,
  );
};

describe('Performance', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn(async (input) => {
      const url = new URL(String(input));
      if (url.pathname.endsWith('/steam/player-search')) {
        return {
          ok: true,
          json: async () => ({
            players: [{ steam_id: '76561198065602953', name: 'Rival Uno', matches: 12 }],
          }),
        };
      }
      if (url.pathname.includes('/steam/player-stats/')) {
        return { ok: true, json: async () => createPayload('Rival Uno', '76561198065602953') };
      }
      return { ok: true, json: async () => createPayload() };
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders real performance data, map filters and player comparison', async () => {
    renderPerformance();

    expect(screen.getByTestId('navigation-frame')).toBeInTheDocument();
    expect(await screen.findByText('Tu nivel, mapa a mapa', { selector: 'h1' })).toBeInTheDocument();
    expect(screen.getAllByText('Todos los mapas', { selector: 'h2' }).length).toBeGreaterThan(0);
    expect(screen.getByRole('tab', { name: /Utilidad táctica/i })).toBeInTheDocument();
    expect(screen.getByText(/18 partidas · datos reales/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('tab', { name: /Mirage/i }));
    expect((await screen.findAllByText('Mirage', { selector: 'h2' })).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: 'Todos los mapas' }));
    expect(screen.getAllByText('Todos los mapas', { selector: 'h2' }).length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('tab', { name: /Cache/i }));
    expect((await screen.findAllByText('Cache', { selector: 'h2' })).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('tab', { name: /Utilidad/i }));
    expect(await screen.findByTitle('Flashbang')).toHaveAttribute(
      'src',
      '/images/cs2/equipment/flashbang.svg',
    );
    expect(await screen.findByTitle('Smoke Grenade')).toHaveAttribute(
      'src',
      '/images/cs2/equipment/smokegrenade.svg',
    );

    fireEvent.click(screen.getByRole('button', { name: /Comparar jugador/i }));
    expect(await screen.findByRole('dialog', { name: /Elige una referencia real/i })).toBeInTheDocument();
    fireEvent.click(await screen.findByRole('button', { name: /Rival Uno/i }));
    expect((await screen.findAllByText('Tú vs Rival Uno')).length).toBeGreaterThan(0);
    expect(await screen.findByRole('heading', { name: 'Atributos competitivos' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Puntería' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Utilidad' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Mapas' })).toBeInTheDocument();
    expect(screen.getAllByText('Tiempo hasta daño').length).toBeGreaterThan(0);
    expect(screen.getAllByText('AK-47').length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: 'Volver a mi rendimiento' }));
    expect(screen.getByText('Tu nivel, mapa a mapa', { selector: 'h1' })).toBeInTheDocument();
  });
});
