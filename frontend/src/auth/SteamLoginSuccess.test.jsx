import React from 'react';
import { render, screen } from '@testing-library/react';
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

  it('enters the dashboard when the authenticated account is configured', async () => {
    useAuth.mockReturnValue({
      user: { authenticated: true, steam_id: '76561198116485358' },
      authError: null,
    });
    globalThis.fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ configured: true, temporary_interface_mode: true }),
    });

    renderCallback();

    expect(await screen.findByText('Dashboard')).toBeInTheDocument();
    expect(globalThis.fetch).toHaveBeenCalledTimes(1);
  });
});
