import React from 'react';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { API_URL } from './utils/api';
// Resolve the real lazy module during test setup; cold Vite transforms are not route latency.
import './features/landing-world';

vi.mock('./features/landing-world/WorldScene', () => ({ default: function Scene({ onReady }) {
  React.useEffect(() => { onReady(); }, [onReady]);
  return <div data-testid="mirage-scene" />;
} }));
vi.mock('./pages/CoachDashboard', () => ({ default: () => <h1>Cuenta conectada</h1> }));
vi.mock('./auth/HistoryCodeForm', () => ({ default: () => <h1>Conectar historial</h1> }));

beforeEach(() => {
  window.history.replaceState({}, '', '/');
  vi.stubGlobal('matchMedia', () => ({ matches:false, addEventListener:vi.fn(), removeEventListener:vi.fn() }));
});
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

function sessionApi({ authenticated = true, configured = false } = {}) {
  const fetcher = vi.fn(async (url, options) => {
    expect(options.credentials).toBe('include');
    if (url === `${API_URL}/auth/steam/status`) return { ok:true, json:async () => ({ authenticated, steam_id:authenticated ? '76561198000000000' : undefined }) };
    if (url === `${API_URL}/steam/pipeline-status`) return { ok:true, json:async () => ({ configured }) };
    throw new Error(`Unexpected API request: ${url}`);
  });
  vi.stubGlobal('fetch', fetcher);
  return fetcher;
}

describe('official Mirage landing and existing account routes', () => {
  it('serves Mirage at / with the real Steam endpoint and no public session request', async () => {
    const fetcher = sessionApi();
    render(<App />);
    expect(await screen.findByRole('heading', { name:'Tu partida, desde dentro.' }, { timeout:5000 })).toBeInTheDocument();
    expect(await screen.findByTestId('mirage-scene')).toBeInTheDocument();
    expect(screen.getByRole('link', { name:'Conectar Steam gratis' })).toHaveAttribute('href', `${API_URL}/auth/steam/login`);
    expect(fetcher).not.toHaveBeenCalled();
    expect(screen.queryByText('EXPLORACIÓN DE DISEÑO')).not.toBeInTheDocument();
  });

  it('returns an unauthenticated private visit to the official landing', async () => {
    sessionApi({ authenticated:false });
    window.history.replaceState({}, '', '/dashboard');
    render(<App />);
    expect(await screen.findByRole('heading', { name:'Tu partida, desde dentro.' })).toBeInTheDocument();
    expect(window.location.pathname).toBe('/');
    expect(screen.queryByText('Cuenta conectada')).not.toBeInTheDocument();
  });

  it.each([false, true])('keeps the verified Steam return and configured=%s account destination', async configured => {
    const fetcher = sessionApi({ configured });
    window.history.replaceState({}, '', '/steam-login-success');
    render(<App />);
    expect(await screen.findByRole('heading', { name:configured ? 'Cuenta conectada' : 'Conectar historial' })).toBeInTheDocument();
    await waitFor(() => expect(window.location.pathname).toBe(configured ? '/dashboard' : '/history-code'));
    expect(fetcher).toHaveBeenCalledWith(`${API_URL}/steam/pipeline-status`, expect.objectContaining({ credentials:'include' }));
    expect(screen.queryByTestId('mirage-scene')).not.toBeInTheDocument();
  });
});
