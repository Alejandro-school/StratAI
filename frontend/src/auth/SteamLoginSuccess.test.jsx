import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, beforeEach, vi } from 'vitest';
import SteamLoginSuccess from './SteamLoginSuccess';
import { useAuth } from './useAuth';

vi.mock('./useAuth', () => ({
  useAuth: vi.fn(),
}));

const renderCallback = () => render(
  <MemoryRouter initialEntries={['/steam-login-success']}>
    <Routes>
      <Route path="/" element={<div>Landing</div>} />
      <Route path="/dashboard" element={<div>Dashboard</div>} />
      <Route path="/history-code" element={<div>Conectar historial</div>} />
      <Route path="/steam-login-success" element={<SteamLoginSuccess />} />
    </Routes>
  </MemoryRouter>,
);

describe('SteamLoginSuccess', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows an actionable error instead of redirecting to the Landing', async () => {
    useAuth.mockReturnValue({ user: null, authError: null });

    renderCallback();

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'No se pudo confirmar la sesión de Steam',
    );
    expect(screen.queryByText('Landing')).not.toBeInTheDocument();
  });

  it('enters the dashboard without friendship checks', async () => {
    useAuth.mockReturnValue({
      user: { authenticated: true, steam_id: '76561198116485358' },
      authError: null,
    });
    globalThis.fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ configured: true, credential_status: 'configured' }),
    });

    renderCallback();

    expect(await screen.findByText('Dashboard')).toBeInTheDocument();
    expect(globalThis.fetch).toHaveBeenCalledTimes(1);
  });

  it('asks for history credentials when they are missing', async () => {
    useAuth.mockReturnValue({ user: { authenticated: true }, authError: null });
    globalThis.fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ configured: false, credential_status: 'missing' }),
    });

    renderCallback();

    expect(await screen.findByText('Conectar historial')).toBeInTheDocument();
    expect(globalThis.fetch).toHaveBeenCalledTimes(1);
  });

  it('retries an unavailable pipeline without restarting the Steam login', async () => {
    useAuth.mockReturnValue({ user: { authenticated: true }, authError: null });
    globalThis.fetch
      .mockResolvedValueOnce({ ok: false, status: 503 })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ configured: true }) });

    renderCallback();
    fireEvent.click(await screen.findByRole('button', { name: 'Reintentar' }));

    expect(await screen.findByText('Dashboard')).toBeInTheDocument();
    expect(globalThis.fetch).toHaveBeenCalledTimes(2);
  });
});
