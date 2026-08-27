import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import TacticalPresencePanel from './TacticalPresencePanel';
import TacticalUtilityPanel from './TacticalUtilityPanel';
import TacticalZonePanel from './TacticalZonePanel';

describe('tactical analysis panels', () => {
  it('separates positioning controls from CT/T distribution and normalized conclusions', () => {
    render(
      <TacticalPresencePanel
        activeSide="ct"
        metrics={{
          total_rounds: 12,
          total_samples: 100,
          top_positions: [{ area: 'TopofMid', sample_count: 60, time_percent: 60 }],
        }}
        matchesAnalyzed={4}
        sideDistribution={{ ct: 62, t: 38 }}
        onIntensityChange={() => {}}
        onToggleHeatmap={() => {}}
        onToggleRoutes={() => {}}
      />,
    );

    expect(screen.getByText('Distribución CT / T')).toBeInTheDocument();
    expect(screen.getByText('62%')).toBeInTheDocument();
    expect(screen.getByText('Capas visuales')).toBeInTheDocument();
    expect(screen.getByText('Top of Mid')).toBeInTheDocument();
    expect(screen.getByText('CT · Por zonas')).toBeInTheDocument();
  });

  it('changes combat metric from one panel and keeps the shared evidence visible', () => {
    const onMetricChange = vi.fn();
    render(
      <TacticalZonePanel
        metric="volume"
        activeSide="t"
        callouts={[{
          name: 'ExtendedA',
          sample_size: 12,
          adjustedWinRate: 54,
          confidenceLabel: 'medium',
          impactScore: 62,
          riskScore: 31,
        }]}
        baselineWinRate={50}
        matchesAnalyzed={6}
        onMetricChange={onMetricChange}
        onSelect={() => {}}
      />,
    );

    expect(screen.getByText('Extended A')).toBeInTheDocument();
    expect(screen.getByText('Zonas · Volumen · T')).toBeInTheDocument();
    expect(screen.getByText(/12 duelos/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Riesgo' }));
    expect(onMetricChange).toHaveBeenCalledWith('risk');
  });

  it('prioritizes measured utility effect and does not invent smoke coverage', () => {
    const onClusterSelect = vi.fn();
    render(
      <TacticalUtilityPanel
        mapName="de_dust2"
        activeSide="ct"
        grenadeData={{
          smoke: [
            { x: 40, y: 40, count: 5, side: 'ct', areas: ['ExtendedA'] },
            { x: 42, y: 41, count: 2, side: 'ct', areas: ['ExtendedA'] },
          ],
          flash: [
            { x: 50, y: 50, count: 10, side: 'ct', avg_blinded: 1.6, areas: ['TopofMid'] },
            { x: 30, y: 30, count: 99, side: 't', avg_blinded: 2.2, areas: ['BDoors'] },
          ],
          he: [],
          molotov: [],
        }}
        matchesAnalyzed={5}
        visibleTypes={{ smoke: true, flash: true, he: true, molotov: true }}
        onToggleType={() => {}}
        onClusterSelect={onClusterSelect}
        onClusterClose={() => {}}
      />,
    );

    expect(screen.getByText('cobertura no disponible')).toBeInTheDocument();
    expect(screen.getByText('rivales cegados / uso')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Humo7 usos/ }).querySelector('img'))
      .toHaveAttribute('src', '/images/weapons/weapon_smokegrenade.png');
    expect(screen.getByRole('button', { name: /^Cegadora10 usos/ }).querySelector('img'))
      .toHaveAttribute('src', '/images/weapons/weapon_flashbang.png');
    expect(screen.getByRole('button', { name: /Extended A.*7 usos/ })).toBeInTheDocument();
    expect(screen.queryByText('B Doors')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Top of Mid/ }));
    expect(onClusterSelect).toHaveBeenCalledWith(expect.objectContaining({ count: 10 }), 'flash');
  });
});
