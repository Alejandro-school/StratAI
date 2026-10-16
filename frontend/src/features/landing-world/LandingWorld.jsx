import React, { Suspense, lazy, useCallback, useEffect, useRef, useState } from 'react';
import { ArrowDown, ArrowUpRight, ArrowUpLeft, BarChart3, BrainCircuit, CircleDot, Map, Minus, Plus, RotateCcw, Trophy } from 'lucide-react';
import { SceneBoundary, Loading } from './SceneStatus';
import { FEATURES, getFeature } from './journey';
import { INFORMATION_PANELS, LANDING_COPY as copy } from './landingContent';
import { ProductPanel, STEAM_LOGIN } from './ProductPanel';
import { WorldPins } from './WorldPins';
import { createNavigation, panNavigation, rotateNavigation, zoomNavigation } from './cameraNavigation';
import { useExplorationInput } from './useExplorationInput';
import { useSteamEntry } from './useSteamEntry';
import { TOOL_EXIT_MS } from './sceneTransitions';

const WorldScene = lazy(() => import('./WorldScene'));
const icons = { replay: CircleDot, utility: Map, stats: BarChart3, coach: BrainCircuit, challenges: Trophy };

export function LandingWorld() {
  const [selected, setSelected] = useState(null);
  const [destination, setDestination] = useState(null);
  const [panelLeaving, setPanelLeaving] = useState(false);
  const [mode, setMode] = useState('aerial');
  const [ready, setReady] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(() => window.matchMedia('(prefers-reduced-motion: reduce)').matches);
  const phase = mode === 'aerial' ? 0 : mode === 'exploring' ? 2 : 1;
  const pins = useRef({}), journey = useRef(0), pointer = useRef({ x: 0, y: 0 });
  const navigation = useRef(createNavigation()), interaction = useRef(false);
  const flight = useRef({ running: false, from: 0, to: 0, elapsed: 0, fresh: false });
  const playback = useRef({ playing: false, progress: 0 });
  const stage = useRef(), renderRequest = useRef(() => {}), lastFocus = useRef();
  const selectionTimer = useRef(), sceneAvailable = useRef(false);
  const startEntry = useCallback(() => { window.clearTimeout(selectionTimer.current); setPanelLeaving(false); playback.current.playing = false; }, []);
  const steam = useSteamEntry({ href:STEAM_LOGIN, canAnimate:ready && sceneAvailable.current && !flight.current.running, reducedMotion, renderRequest, onStart:startEntry });
  const onReady = useCallback(() => { sceneAvailable.current = true; setReady(true); }, []);
  const onSceneError = useCallback(() => { sceneAvailable.current = false; setReady(true); }, []);
  const onArrive = useCallback(destination => {
    setMode(destination === 1 ? 'exploring' : 'aerial');
    if (destination === 1 && !selected && !steam.transition.current.active) stage.current?.focus({ preventScroll: true });
  }, [selected]);
  useExplorationInput({ stage, enabled: mode === 'exploring' && !selected && !steam.entering, navigation, renderRequest, interaction });
  useEffect(() => () => window.clearTimeout(selectionTimer.current), []);
  useEffect(() => {
    const query = window.matchMedia('(prefers-reduced-motion: reduce)');
    const change = () => setReducedMotion(query.matches);
    query.addEventListener('change', change);
    return () => query.removeEventListener('change', change);
  }, []);
  const travel = useCallback(destination => {
    if (steam.transition.current.active) return;
    window.clearTimeout(selectionTimer.current); setPanelLeaving(false); setDestination(null);
    playback.current.playing = false; setSelected(null);
    if (destination === 0 && journey.current === 0 && !flight.current.running) { setMode('aerial'); return; }
    if (destination === 1) navigation.current = createNavigation();
    if (reducedMotion) { journey.current = destination; flight.current.running = false; setMode(destination === 1 ? 'exploring' : 'aerial'); }
    else {
      flight.current = { running: true, from: journey.current, to: destination, elapsed: 0, fresh: true };
      setMode(destination === 1 ? 'entering' : 'leaving');
    }
    renderRequest.current();
  }, [reducedMotion]);
  const select = useCallback(id => {
    if (steam.entering || (!getFeature(id) && !INFORMATION_PANELS.some(panel => panel.id === id))) return;
    if (!document.activeElement?.closest('.feature-panel')) lastFocus.current = document.activeElement;
    if (selected === id && destination === id && !panelLeaving) return;
    window.clearTimeout(selectionTimer.current);
    if (journey.current < 1) travel(1);
    playback.current = { playing: false, progress: 0 }; setDestination(id);
    if (selected && selected !== id && !reducedMotion) {
      setPanelLeaving(true);
      selectionTimer.current = window.setTimeout(() => { setSelected(id); setPanelLeaving(false); }, TOOL_EXIT_MS);
    } else { setSelected(id); setPanelLeaving(false); }
    renderRequest.current();
  }, [travel, selected, destination, panelLeaving, reducedMotion, steam.entering]);
  const close = useCallback(() => {
    window.clearTimeout(selectionTimer.current); setPanelLeaving(false); setDestination(null);
    playback.current.playing = false; setSelected(null); renderRequest.current();
    const destination = lastFocus.current?.isConnected ? lastFocus.current : document.querySelector('#product-navigation button');
    destination?.focus?.({ preventScroll: true });
  }, []);
  const reset = () => { navigation.current = createNavigation(); pointer.current = { x: 0, y: 0 }; renderRequest.current(); };
  const zoom = delta => { zoomNavigation(navigation.current, delta); renderRequest.current(); };
  useEffect(() => {
    const key = event => { if (event.key === 'Escape') { if (steam.transition.current.active) steam.cancel(); else close(); } };
    const visibility = () => { if (document.hidden) playback.current.playing = false; else renderRequest.current(); };
    window.addEventListener('keydown', key); document.addEventListener('visibilitychange', visibility);
    return () => { window.removeEventListener('keydown', key); document.removeEventListener('visibilitychange', visibility); };
  }, [close, steam.cancel, steam.transition]);
  const feature = getFeature(selected) || INFORMATION_PANELS.find(panel => panel.id === selected);
  const interfaceHidden = steam.entering || !ready;
  return <div className={`landing-world phase-${phase} mode-${mode} ${selected ? 'feature-open' : ''} ${ready ? 'is-ready' : ''} ${reducedMotion ? 'motion-reduced' : ''} ${steam.entering ? 'steam-entering' : ''}`}>
    <a className="world-skip" inert={interfaceHidden ? '' : undefined} href="#product-navigation">Ir a las funciones del producto</a>
    <div ref={stage} className="world-stage" tabIndex={0} inert={interfaceHidden ? '' : undefined} aria-hidden={interfaceHidden || undefined} aria-label="Explorar el escenario de Mirage. Flechas para desplazarte, más y menos para zoom, R para restablecer."
      onKeyDown={event => {
        if (mode !== 'exploring' || selected || event.altKey || event.ctrlKey || event.metaKey) return;
        if (event.key === '+' || event.key === '=') zoom(-100);
        else if (event.key === '-') zoom(100);
        else if (event.key.toLowerCase() === 'r') reset();
        else if (['ArrowLeft','ArrowRight','ArrowUp','ArrowDown'].includes(event.key)) {
          const dx = event.key === 'ArrowLeft' ? 35 : event.key === 'ArrowRight' ? -35 : 0;
          const dy = event.key === 'ArrowUp' ? 35 : event.key === 'ArrowDown' ? -35 : 0;
          if (event.shiftKey) rotateNavigation(navigation.current, dx, dy); else panNavigation(navigation.current, dx, dy);
          renderRequest.current();
        } else return;
        event.preventDefault();
      }}
      onPointerMove={event => {
        if (event.pointerType === 'touch' || interaction.current) return;
        pointer.current = { x: event.clientX / window.innerWidth * 2 - 1, y: -(event.clientY / window.innerHeight * 2 - 1) };
        if (!reducedMotion && !selected) renderRequest.current();
      }} onPointerLeave={() => { pointer.current = { x: 0, y: 0 }; renderRequest.current(); }}>
      <SceneBoundary onError={onSceneError} onFallback={() => { setReady(true); journey.current = 1; flight.current.running = false; setMode('exploring'); }}>
        <Suspense fallback={null}><WorldScene journey={journey} flight={flight} navigation={navigation} interaction={interaction} onArrive={onArrive} selected={destination} pointer={pointer} pins={pins} onReady={onReady} reducedMotion={reducedMotion} playback={playback} stage={stage} entered={phase === 2 && ready && !steam.entering} renderRequest={renderRequest} entry={steam.transition} /></Suspense>
      </SceneBoundary>
    </div>
    <WorldPins elements={pins} enabled={phase === 2 && ready && !steam.entering} selected={destination} onSelect={select} />
    <div className="world-vignette" aria-hidden="true" />
    <header className="world-header" inert={interfaceHidden ? '' : undefined} aria-hidden={interfaceHidden || undefined}><button className="world-brand" onClick={() => travel(0)} aria-label="StratAI, volver a la vista aérea">Strat<span>AI</span><span className="brand-pixel" /></button><span className="world-location"><span />MIRAGE <span className="location-divider">/</span> COUNTER-STRIKE 2</span><nav aria-label="Navegación del recorrido"><button onClick={() => select('guide')}>Cómo funciona</button><button onClick={() => select('plans')}>Planes y FAQ</button><button className="header-explore" onClick={() => travel(phase === 0 ? 1 : 0)} disabled={!ready || phase === 1}>{phase === 0 ? 'Entrar a A' : phase === 1 ? 'En vuelo' : 'Vista aérea'}{phase === 0 ? <ArrowDown size={14} /> : <ArrowUpLeft size={14} />}</button></nav></header>
    <main className="world-overlay" inert={interfaceHidden ? '' : undefined} aria-hidden={interfaceHidden || undefined}>
      {phase === 0 && <section className="world-intro" aria-labelledby="intro-title"><h1 id="intro-title">Tu partida,<br />{' '}<span>desde dentro.</span></h1><p>{copy.hero.subtitle}</p><div className="hero-capabilities">{[copy.hero.badge1, copy.hero.badge2, copy.hero.badge3].map(item => <span key={item}>{item}</span>)}</div><div className="hero-actions"><button className="world-primary" onClick={() => travel(1)} disabled={!ready}>Estoy listo<ArrowDown size={17} /></button><a href={STEAM_LOGIN} onClick={steam.onLogin}>{copy.hero.ctaPrimary}<ArrowUpRight size={16} /></a></div><span className="hero-development">Explora StratAI en Mirage · Coach IA en desarrollo</span></section>}
      {phase === 1 && !selected && <div className="flight-copy" role="status"><p>{mode === 'entering' ? 'Entrando en A' : 'Volviendo al aire'}</p><span>Una nueva perspectiva de tu partida.</span></div>}
      {phase === 2 && !selected && <div className="inside-copy"><h2>Ahora, explora A.</h2><p>Muévete por el punto.<br />Descubre cada herramienta.</p></div>}
      {feature && <ProductPanel key={feature.id} panel={feature} playback={playback} onClose={close} onSelect={select} renderRequest={renderRequest} reducedMotion={reducedMotion} onLogin={steam.onLogin} leaving={panelLeaving} suspended={steam.entering} />}
    </main>
    {phase === 2 && !selected && <aside className="camera-tools" inert={interfaceHidden ? '' : undefined} aria-hidden={interfaceHidden || undefined} aria-label="Controles de la cámara"><p>Arrastra para explorar<span>Rueda o pellizco para acercarte · Giro con botón derecho</span></p><div><button onClick={() => zoom(-130)} aria-label="Acercar cámara"><Plus size={17} /></button><button onClick={() => zoom(130)} aria-label="Alejar cámara"><Minus size={17} /></button><button onClick={reset} aria-label="Restablecer vista de A"><RotateCcw size={16} /></button><button onClick={() => travel(0)}><ArrowUpLeft size={16} />Volver al aire</button></div></aside>}
    <footer className="world-bottom" inert={interfaceHidden ? '' : undefined} aria-hidden={interfaceHidden || undefined}><div className="world-bottom-note"><span className="preview-pixel" />ANALIZA TU PARTIDA<span className="bottom-note-detail">Mirage · Descubre StratAI</span></div><nav id="product-navigation" className="product-navigation" tabIndex={-1} aria-label="Funciones de StratAI">{FEATURES.map(item => { const Icon = icons[item.id]; return <button key={item.id} disabled={!ready} onClick={() => select(item.id)} aria-pressed={destination === item.id}><Icon size={16} /><span>{item.short}</span></button>; })}</nav><button className="motion-toggle" onClick={() => setReducedMotion(value => !value)} aria-pressed={reducedMotion}>Movimiento {reducedMotion ? 'reducido' : 'completo'}<span /></button></footer>
    <Loading ready={ready} suspended={steam.entering} loginHref={STEAM_LOGIN} onLogin={steam.onLogin} onExplore={() => { setReady(true); journey.current = 1; flight.current.running = false; setMode('exploring'); }} />
    {steam.entering && <section ref={steam.status} className="steam-entry" tabIndex={-1} aria-labelledby="steam-entry-status" style={{ '--entry-duration':`${steam.transition.current.duration}ms` }}>
      <div className="steam-entry-mark"><div className="steam-entry-brand" aria-hidden="true">Strat<span>AI</span><span className="brand-pixel" /></div><p id="steam-entry-status" role="status">Conectando con Steam</p>
        {steam.handoff && <div className="steam-entry-actions"><a href={STEAM_LOGIN}>Continuar a Steam<ArrowUpRight size={15} /></a><button onClick={steam.cancel}>Volver a Mirage</button></div>}
      </div>
    </section>}
    <span className="world-sr-only" role="status">{ready ? phase === 0 ? 'Vista aérea de Mirage. Pulsa Estoy listo para entrar.' : phase === 1 ? 'Recorriendo Mirage.' : 'Dentro de A. Puedes mover la cámara y explorar las funciones.' : 'Cargando el escenario tridimensional.'}</span>
  </div>;
}
