import React from 'react';
import { render, fireEvent, screen, cleanup } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { LandingWorld } from './LandingWorld';
import { LANDING_COPY, PRODUCT_CONTENT } from './landingContent';

vi.mock('./WorldScene', () => ({ default: function Scene(props) {
  React.useEffect(() => { props.onReady(); }, [props.onReady]);
  return <button data-testid="arrive" onClick={() => { props.journey.current=props.flight.current.to; props.flight.current.running=false; props.onArrive(props.journey.current); }}>Finish flight</button>;
} }));
beforeEach(() => vi.stubGlobal('matchMedia', () => ({ matches:false, addEventListener:vi.fn(),removeEventListener:vi.fn() })));
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe('button-led Mirage experience', () => {
  it('enters only on intent, exposes exploration controls and returns to the aerial introduction', async () => {
    render(<LandingWorld />);
    const enter = await screen.findByRole('button', { name:'Estoy listo' });
    await screen.findByTestId('arrive');
    fireEvent.wheel(screen.getByLabelText(/Explorar el escenario/),{deltaY:800});
    expect(screen.getByRole('heading',{name:'Tu partida, desde dentro.'})).toBeTruthy();
    fireEvent.click(enter);
    expect(screen.getByText('Entrando en A')).toBeTruthy();
    expect(screen.queryByRole('button',{name:'Estoy listo'})).toBeNull();
    fireEvent.click(screen.getByTestId('arrive'));
    expect(screen.getByRole('button',{name:'Acercar cámara'})).toBeTruthy();
    fireEvent.click(screen.getByRole('button',{name:'Volver al aire'}));
    fireEvent.click(screen.getByTestId('arrive'));
    expect(screen.getByRole('button',{name:'Estoy listo'})).toBeTruthy();
  });
  it('keeps native panel scrolling and restores the connected trigger after closing', async () => {
    render(<LandingWorld />); await screen.findByTestId('arrive');
    const trigger=screen.getByRole('button',{name:'Mapa',exact:true}); trigger.focus(); fireEvent.click(trigger);
    fireEvent.click(screen.getByTestId('arrive'));
    expect(screen.getByRole('heading',{name:'Mapa interactivo'})).toBeTruthy();
    fireEvent.click(screen.getByRole('button',{name:'Cerrar información'}));
    expect(document.activeElement).toBe(trigger);
    expect(screen.getByRole('button',{name:'Restablecer vista de A'})).toBeTruthy();
  });

  it('dismisses on an outside press, keeps panel controls usable and allows direct tool switching', async () => {
    render(<LandingWorld />); await screen.findByTestId('arrive');
    const map = screen.getByRole('button',{name:'Mapa',exact:true});
    map.focus(); fireEvent.click(map); fireEvent.click(screen.getByTestId('arrive'));
    fireEvent.pointerDown(screen.getByRole('heading',{name:'Mapa interactivo'}));
    fireEvent.click(screen.getByRole('button',{name:'Ver trayectoria en Mirage'}));
    expect(screen.getByRole('button',{name:'Pausar trayectoria en Mirage'})).toBeTruthy();
    fireEvent.pointerDown(screen.getByLabelText(/Explorar el escenario/),{pointerType:'mouse',button:0});
    expect(screen.queryByRole('heading',{name:'Mapa interactivo'})).toBeNull();
    expect(document.activeElement).toBe(map);
    expect(screen.getByRole('button',{name:'Restablecer vista de A'})).toBeTruthy();

    fireEvent.click(map);
    const stats = screen.getByRole('button',{name:'Stats',exact:true});
    fireEvent.pointerDown(stats,{pointerType:'touch',button:0});
    stats.focus(); fireEvent.click(stats);
    expect(await screen.findByRole('heading',{name:'Estadísticas detalladas'})).toBeTruthy();
    fireEvent.click(screen.getByRole('button',{name:'Cerrar información'}));
    expect(document.activeElement).toBe(stats);
  });

  it('keeps only the latest tool request and cancels a pending replacement when dismissed', async () => {
    render(<LandingWorld />);await screen.findByTestId('arrive');
    fireEvent.click(screen.getByRole('button',{name:'Mapa',exact:true}));fireEvent.click(screen.getByTestId('arrive'));
    fireEvent.click(screen.getByRole('button',{name:'Stats',exact:true}));
    fireEvent.click(screen.getByRole('button',{name:'Coach',exact:true}));
    expect(await screen.findByRole('heading',{name:PRODUCT_CONTENT.coach.title})).toBeTruthy();
    expect(screen.queryByRole('heading',{name:PRODUCT_CONTENT.stats.title})).toBeNull();
    fireEvent.click(screen.getByRole('button',{name:'Mapa',exact:true}));fireEvent.keyDown(window,{key:'Escape'});
    await new Promise(resolve=>setTimeout(resolve,150));
    expect(screen.queryByRole('heading',{name:PRODUCT_CONTENT.utility.title})).toBeNull();
    expect(screen.getByRole('button',{name:'Restablecer vista de A'})).toBeTruthy();
  });

  it('announces Steam entry and lets Escape restore the original link without navigating', async () => {
    render(<LandingWorld />);await screen.findByTestId('arrive');
    const login=screen.getByRole('link',{name:LANDING_COPY.hero.ctaPrimary});login.focus();fireEvent.click(login);
    expect(screen.getByText('Conectando con Steam')).toBeTruthy();
    expect(screen.queryByRole('button',{name:'Estoy listo'})).toBeNull();
    fireEvent.keyDown(window,{key:'Escape'});
    expect(screen.queryByText('Conectando con Steam')).toBeNull();expect(document.activeElement).toBe(login);
  });
});
