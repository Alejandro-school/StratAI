import React, { Component } from 'react';
import { useProgress } from '@react-three/drei';

export class SceneBoundary extends Component {
  state = { error: false };
  static getDerivedStateFromError() { return { error: true }; }
  componentDidCatch() { this.props.onError(); }
  render() {
    if (this.state.error) return <div className="world-fallback"><img src="/maps/de_mirage_radar_psd.webp" alt="Vista general de Mirage" /><div><h2>No se ha podido abrir el entorno 3D.</h2><p>Puedes volver a cargarlo o consultar las funciones desde los controles inferiores.</p><button onClick={() => window.location.reload()}>Volver a cargar</button><button onClick={this.props.onFallback}>Ver funciones</button></div></div>;
    return this.props.children;
  }
}

export function Loading({ ready, loginHref, onLogin, onExplore, suspended = false }) {
  const { progress, errors } = useProgress();
  if (ready) return null;
  return <div className="world-loading" role="status" inert={suspended ? '' : undefined} aria-hidden={suspended || undefined}><span className="loading-brand">Strat<span>AI</span></span><div className="loading-bar"><span style={{ transform: `scaleX(${progress / 100})` }} /></div><p>{errors.length ? 'El entorno no ha podido cargarse.' : 'Preparando Mirage'}</p><span>Tu siguiente perspectiva está a punto de empezar.</span><div className="world-loading-actions">{loginHref && <a href={loginHref} onClick={onLogin}>Conectar Steam</a>}{onExplore && <button onClick={onExplore}>Ver funciones mientras carga</button>}{errors.length > 0 && <button onClick={() => window.location.reload()}>Reintentar</button>}</div></div>;
}

