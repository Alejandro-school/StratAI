import React, { useState } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { SceneBoundary, Loading } from './SceneStatus';

describe('landing scene recovery', () => {
  it('keeps sign-in and product access available while the 3D scene is loading', () => {
    const login = vi.fn(event => event.preventDefault()), explore = vi.fn();
    render(<Loading ready={false} loginHref="/auth/steam/login" onLogin={login} onExplore={explore} />);
    const link = screen.getByRole('link', { name:'Conectar Steam' });
    expect(link).toHaveAttribute('href', '/auth/steam/login');
    fireEvent.click(link);
    expect(login).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name:'Ver funciones mientras carga' }));
    expect(explore).toHaveBeenCalledOnce();
  });

  it('exposes recovery and product navigation if the scene throws before becoming ready', () => {
    const expectedError = vi.spyOn(console, 'error').mockImplementation(() => {});
    const navigate = vi.fn();
    function BrokenScene() { throw new Error('Scene failed before readiness'); }
    function Page() {
      const [ready, setReady] = useState(false);
      return <>
        <SceneBoundary onError={() => setReady(true)} onFallback={navigate}><BrokenScene /></SceneBoundary>
        <Loading ready={ready} />
      </>;
    }
    try {
      render(<Page />);
      expect(screen.queryByText('Preparando Mirage')).not.toBeInTheDocument();
      expect(screen.getByRole('heading', { name: 'No se ha podido abrir el entorno 3D.' })).toBeVisible();
      expect(screen.getByRole('button', { name: 'Volver a cargar' })).toBeEnabled();
      fireEvent.click(screen.getByRole('button', { name: 'Ver funciones' }));
      expect(navigate).toHaveBeenCalledOnce();
    } finally {
      expectedError.mockRestore();
    }
  });
});
