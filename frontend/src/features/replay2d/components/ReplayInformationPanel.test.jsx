import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ReplayInformationPanel } from './ReplayInformationPanel';

describe('possible reception and dated memory presentation', () => {
  it('explains who observed and who could receive, without presenting memory as current location', () => {
    const observation = { id: 'o1', observer_id: '1', subject_id: '6', position: { x: 10, y: 20 },
      observation_tick: 100, availability_tick: 100, status: 'estimated', recipients: [
        { player_id: '1', reception: 'origin' }, { player_id: '2', reception: 'possible' }] };
    const information = { contacts: [{ subject_id: '6', state: 'last_observed', age_ticks: 512,
      position: observation.position, observation_ids: ['o1'] }], observations: [observation],
      by_player: [{ player_id: '1', observation_ids: ['o1'] }, { player_id: '2', observation_ids: ['o1'] }] };
    render(<ReplayInformationPanel information={information} tickRate={64} tick={612}
      labels={{ '1': 'T1', '2': 'T2', '6': 'CT1' }} />);
    expect(screen.getAllByText(/hace 8 s/)).toHaveLength(3);
    expect(screen.getByText(/Posición actual desconocida/)).toBeInTheDocument();
    expect(screen.getByText(/observación calculada/).closest('li')).toHaveTextContent('T1');
    expect(screen.getByText(/posible recepción/).closest('li')).toHaveTextContent('T2');
  });
  it('explains an empty contact register without declaring enemy absence', () => {
    render(<ReplayInformationPanel information={{ contacts: [], observations: [], by_player: [] }} tickRate={64} tick={100} />);
    expect(screen.getByText(/No hay contactos registrados/)).toBeInTheDocument();
    expect(screen.getByText(/no demuestra ausencia/)).toBeInTheDocument();
  });
  it('presents fractional age readably while retaining the exact source ticks', () => {
    const contact = { subject_id: '6', state: 'last_observed', age_ticks: 1423, position: { x: 10, y: 20 } };
    render(<ReplayInformationPanel information={{ contacts: [contact], observations: [], by_player: [] }} tickRate={64} tick={1523} />);
    expect(screen.getByText(/hace 22.2 s/)).toBeInTheDocument();
    expect(contact.age_ticks).toBe(1423);
  });
});
