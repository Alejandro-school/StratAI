import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import OperationProgress from './OperationProgress';

const renderProgress = (route = '/progress') => render(
  <MemoryRouter initialEntries={[route]}>
    <OperationProgress userName="Kerchak" />
  </MemoryRouter>,
);

describe('OperationProgress', () => {
  it('prioritizes the AI mission and shows the reachable league gap', () => {
    renderProgress();
    expect(screen.getByRole('heading', { name: 'Tu briefing de mejora' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Recupera el control del primer duelo' })).toBeInTheDocument();
    expect(screen.getByText(/pts al podio/i)).toBeInTheDocument();
    expect(screen.getByText('7 de 10 partidas puntuables analizadas esta semana.')).toBeInTheDocument();
  });

  it('opens evidence from the URL and exposes the diagnostic signals', () => {
    renderProgress('/progress?mission=opening-duels');
    expect(screen.getByRole('dialog', { name: /Evidencia de Recupera el control/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Señales detectadas' })).toBeInTheDocument();
    expect(screen.getByText('Entrada en corto sin flash de apoyo')).toBeInTheDocument();
  });

  it('uses the only weekly reroll and keeps the support slot value', () => {
    renderProgress();
    fireEvent.click(screen.getByRole('button', { name: /Cambiar/i }));
    expect(screen.getByRole('dialog', { name: 'Confirmar cambio de misión' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Usar mi cambio' }));
    expect(screen.getByRole('heading', { name: 'Protege 3 rondas de economía frágil' })).toBeInTheDocument();
    expect(screen.getByText('Cambio utilizado')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent('Misión sustituida');
  });

  it('supports opt-in and anonymous identity in the full league', () => {
    renderProgress('/progress?scenario=new&panel=league');
    expect(screen.getByRole('dialog', { name: 'Clasificación semanal' })).toBeInTheDocument();
    const participation = screen.getByRole('checkbox', { name: 'Participar' });
    expect(participation).not.toBeChecked();
    fireEvent.click(participation);
    const anonymous = screen.getByRole('checkbox', { name: 'Alias anónimo' });
    fireEvent.click(anonymous);
    expect(screen.getAllByText('Analista ••••').length).toBeGreaterThan(0);
  });

  it('renders the previous weekly recap as a deep-linked dialog', () => {
    renderProgress('/progress?recap=1');
    expect(screen.getByRole('dialog', { name: 'Resumen de la semana anterior' })).toBeInTheDocument();
    expect(screen.getByText('Insignia Operador Preciso')).toBeInTheDocument();
    expect(screen.getByText(/Redujiste un 23%/)).toBeInTheDocument();
  });
});
