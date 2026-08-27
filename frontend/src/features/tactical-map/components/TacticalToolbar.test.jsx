import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import TacticalToolbar from './TacticalToolbar';

const maps = [
  { id: 'de_dust2', name: 'Dust II' },
  { id: 'de_mirage', name: 'Mirage' },
];

const renderToolbar = (overrides = {}) => {
  const props = {
    maps,
    currentMap: 'de_dust2',
    currentMapInfo: maps[0],
    activeLens: 'briefing',
    activeSide: 'ct',
    currentLevel: 'upper',
    hasLevels: false,
    isMapMenuOpen: false,
    onMapMenuChange: vi.fn(),
    onMapChange: vi.fn(),
    onLensChange: vi.fn(),
    onSideChange: vi.fn(),
    onLevelChange: vi.fn(),
    ...overrides,
  };
  render(<TacticalToolbar {...props} />);
  return props;
};

describe('TacticalToolbar', () => {
  it('moves through the four analysis tabs with the keyboard', () => {
    const props = renderToolbar();
    const briefing = screen.getByRole('tab', { name: 'Briefing' });

    fireEvent.keyDown(briefing, { key: 'ArrowRight' });
    expect(props.onLensChange).toHaveBeenCalledWith('positioning');
    expect(screen.getByRole('tab', { name: 'Posicionamiento' })).toHaveFocus();
  });

  it('opens the map menu from ArrowDown and exposes radio menu items', () => {
    const onMapMenuChange = vi.fn();
    renderToolbar({ onMapMenuChange });

    fireEvent.keyDown(screen.getByRole('button', { name: /Dust II/ }), { key: 'ArrowDown' });
    expect(onMapMenuChange).toHaveBeenCalledWith(true);
  });

  it('switches directly between CT and T without an aggregated option', () => {
    const onSideChange = vi.fn();
    renderToolbar({ onSideChange });

    expect(screen.queryByRole('button', { name: 'Ambos' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Analizar CT · Defensa/ })).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(screen.getByRole('button', { name: /Analizar T · Ataque/ }));
    expect(onSideChange).toHaveBeenCalledWith('t');
  });
});
