import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import QuickStatsPanel from './QuickStatsPanel';

const games = [
  { match_id: '3', map_name: 'de_nuke', result: 'victory', players: [{ kd_ratio: 1.5, adr: 90, hs_percentage: 50 }] },
  { match_id: '2', map_name: 'de_mirage', result: 'defeat', players: [{ kd_ratio: 0.8, adr: 70, hs_percentage: 40 }] },
  { match_id: '1', map_name: 'de_dust2', result: 'victory', players: [{ kd_ratio: 1.2, adr: 80, hs_percentage: 45 }] },
];

describe('QuickStatsPanel', () => {
  it('keeps the aggregate summary and renders the lightweight trend chart', () => {
    render(<QuickStatsPanel games={games} getPlayerStats={(game) => game.players[0]} />);

    expect(screen.getByText('67%')).toBeInTheDocument();
    expect(screen.getByText('1.17')).toBeInTheDocument();
    expect(screen.getByRole('img', { name: 'Tendencia de K/D' })).toBeInTheDocument();
    expect(screen.getByText('K/D: 1.50')).toBeInTheDocument();
  });
});
