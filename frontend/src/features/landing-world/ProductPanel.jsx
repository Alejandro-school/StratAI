import React, { useEffect, useRef, useState } from 'react';
import { ArrowUpRight, Check, Expand, Pause, Play, X } from 'lucide-react';
import { API_URL } from '../../utils/api';
import { LANDING_COPY as copy, PRODUCT_CONTENT, planPrice } from './landingContent';
import { LandingDemo } from './LandingDemo';

export const STEAM_LOGIN = `${API_URL}/auth/steam/login`;

function ProductScreenshot({ content }) {
  const dialog = useRef();
  return <figure className="product-screenshot">
    <button className="screenshot-open" onClick={() => dialog.current.showModal()} aria-label={`Ampliar captura: ${content.title}`}>
      <img src={content.image} alt={content.alt} decoding="async" />
      <span><Expand size={14} />Ampliar captura original</span>
    </button>
    <dialog ref={dialog} className="screenshot-dialog" aria-label={`Captura ampliada: ${content.title}`} onKeyDown={event => { if (event.key === 'Escape') event.stopPropagation(); }}>
      <div><span>{content.title} · Captura original</span><button autoFocus className="icon-button" onClick={() => dialog.current.close()} aria-label="Cerrar captura ampliada"><X size={22} /></button></div>
      <img src={content.image} alt={content.alt} />
    </dialog>
  </figure>;
}

function ProductContent({ panel, playback, onSelect, renderRequest, reducedMotion }) {
  const content = PRODUCT_CONTENT[panel.id];
  const [playing, setPlaying] = useState(false);
  const toggle = () => {
    if (playback.current.progress === 1) playback.current.progress = 0;
    playback.current.playing = !playback.current.playing;
    renderRequest?.current();
    setPlaying(playback.current.playing);
  };
  useEffect(() => {
    if (panel.id !== 'utility') return undefined;
    const timer = window.setInterval(() => setPlaying(playback.current.playing), 250);
    return () => window.clearInterval(timer);
  }, [panel.id, playback]);
  const reference = <><ProductScreenshot content={content} /><ul className="product-capabilities">{content.features.map(item => <li key={item}><Check size={15} /><span>{item}</span></li>)}</ul></>;
  return <>
    <div className="product-introduction"><h2 id="feature-title">{content.title}</h2>
    {content.prospective && <p className="development-status">{panel.id === 'coach' ? 'Coach IA en desarrollo y evaluación' : 'Propuesta de retos y progresión'}</p>}
    <p className="feature-description">{content.desc}</p></div>
    {panel.id === 'replay' && <LandingDemo reducedMotion={reducedMotion} />}
    {panel.id === 'replay' ? <details className="replay-reference"><summary>Captura original y funciones del replay</summary>{reference}</details> : reference}
    {panel.id === 'coach' && <><p className="content-disclosure">Esta captura presenta la visión del producto. La demo muestra comentarios preparados y no acredita recomendaciones automáticas.</p><button className="panel-action" onClick={() => onSelect('replay')}>Ver la demo táctica<ArrowUpRight size={17} /></button></>}
    {panel.id === 'utility' && <><p className="content-disclosure">El escenario permite visualizar el espacio de una jugada. El arco y el humo de esta exploración son ilustrativos.</p><button className="panel-action" onClick={toggle}>{playing ? <Pause size={16} /> : <Play size={16} />}{playing ? 'Pausar trayectoria en Mirage' : 'Ver trayectoria en Mirage'}</button></>}
    {panel.id === 'stats' && <button className="panel-action" onClick={() => onSelect('replay')}>Conecta las cifras con la ronda<ArrowUpRight size={17} /></button>}
    {panel.id === 'challenges' && <button className="panel-action" onClick={() => onSelect('guide')}>De tu partida a tu mejora<ArrowUpRight size={17} /></button>}
  </>;
}

function Guide({ onSelect, onLogin }) {
  return <>
    <h2 id="feature-title">{copy.howItWorks.title}</h2>
    <p className="feature-description">{copy.services.subtitle}</p>
    <p className="content-disclosure">Así se plantea la experiencia de StratAI. El Coach IA y sus recomendaciones siguen en desarrollo.</p>
    <ol className="product-steps">{[1, 2, 3].map(number => <li key={number}><span>{String(number).padStart(2, '0')}</span><div><h3>{copy.howItWorks[`step${number}Title`]}</h3><p>{copy.howItWorks[`step${number}Desc`]}</p></div></li>)}</ol>
    <div className="guide-outcome"><h3>{copy.cta.proofTitle}</h3><ul>{[1, 2, 3].map(number => <li key={number}><Check size={15} />{copy.cta[`proofItem${number}`]}</li>)}</ul></div>
    <div className="product-close"><h3>{copy.cta.title}</h3><p>{copy.cta.subtitle}</p><a className="panel-action" href={STEAM_LOGIN} onClick={onLogin}>{copy.cta.ctaPrimary}<ArrowUpRight size={17} /></a><button className="content-link" onClick={() => onSelect('replay')}>{copy.cta.ctaSecondary}<ArrowUpRight size={15} /></button></div>
  </>;
}

function Plans({ onLogin }) {
  const [yearly, setYearly] = useState(false);
  return <>
    <h2 id="feature-title">{copy.pricing.title}</h2>
    <p className="feature-description">{copy.pricing.subtitle}</p>
    <p className="content-disclosure">Planes de la propuesta comercial actual. Esta exploración no activa suscripciones ni cobros.</p>
    <div className="billing-switch" aria-label="Periodo del plan">{[false, true].map(value => <button key={String(value)} aria-pressed={yearly === value} onClick={() => setYearly(value)}>{value ? copy.pricing.yearly : copy.pricing.monthly}</button>)}</div>
    <div className="product-plans">{['free', 'pro', 'team'].map(tier => <article key={tier} data-tier={tier}>
      <div className="plan-name"><h3>{copy.pricing[tier].name}</h3>{tier === 'pro' && <span>{copy.pricing.popular}</span>}</div>
      <p className="plan-price">{planPrice(tier, yearly)}<span>€</span><small>{tier === 'free' ? copy.pricing[tier].period : yearly ? '/año' : copy.pricing[tier].period}</small></p>
      <p className="plan-description">{copy.pricing[tier].desc}</p>
      <ul>{copy.pricing[tier].features.map(item => <li key={item}><Check size={13} />{item}</li>)}</ul>
    </article>)}</div>
    <a className="panel-action" href={STEAM_LOGIN} onClick={onLogin}>{copy.hero.ctaPrimary}<ArrowUpRight size={17} /></a>
  </>;
}

function Questions() {
  return <><h2 id="feature-title">{copy.faq.title}</h2><p className="content-disclosure">Respuestas de la propuesta del landing actual. Las funciones del Coach IA descritas aquí siguen en desarrollo.</p><div className="product-faq">{copy.faq.items.map(item => <details key={item.q}><summary>{item.q}</summary><p>{item.a}</p></details>)}</div></>;
}

export function ProductPanel({ panel, playback, onClose, onSelect, onLogin, renderRequest, reducedMotion = false, leaving = false, suspended = false }) {
  const close = useRef(), container = useRef();
  const [tab, setTab] = useState(panel.id === 'plans' ? 'plans' : 'guide');
  const information = panel.id === 'guide' || panel.id === 'plans';
  useEffect(() => { close.current?.focus({ preventScroll: true }); }, [panel.id]);
  useEffect(() => {
    if (suspended) return undefined;
    const dismiss = event => {
      if (event.target.closest?.('#product-navigation button, .world-header nav button, .world-pin')) return;
      if (!container.current?.contains(event.target)) onClose();
    };
    document.addEventListener('pointerdown', dismiss, true);
    return () => document.removeEventListener('pointerdown', dismiss, true);
  }, [onClose, suspended]);
  return <aside ref={container} className={`feature-panel product-panel ${panel.id === 'replay' ? 'product-panel--replay' : ''} ${leaving ? 'is-leaving' : ''}`} inert={leaving || suspended ? '' : undefined} aria-hidden={leaving || suspended || undefined} aria-labelledby="feature-title" style={{ '--feature-color': panel.color }}>
    <div className="feature-chrome"><span>{panel.label}<span className="panel-location">{panel.place} / {information ? 'PRODUCTO' : 'MIRAGE'}</span></span><button ref={close} className="icon-button" onClick={onClose} aria-label="Cerrar información"><X size={20} /></button></div>
    {information && <nav className="content-tabs" aria-label="Información de StratAI">{[['guide', 'Cómo funciona'], ['plans', 'Planes'], ['faq', 'Preguntas frecuentes']].map(([id, label]) => <button key={id} aria-pressed={tab === id} onClick={() => setTab(id)}>{label}</button>)}</nav>}
    <div key={information ? tab : panel.id} className="feature-body product-body">
      {information ? tab === 'guide' ? <Guide onSelect={onSelect} onLogin={onLogin} /> : tab === 'plans' ? <Plans onLogin={onLogin} /> : <Questions /> : <ProductContent panel={panel} playback={playback} onSelect={onSelect} renderRequest={renderRequest} reducedMotion={reducedMotion} />}
      <div className="panel-end"><span>StratAI · Counter-Strike 2</span><span>{copy.footer.rights}</span></div>
    </div>
    <footer className="feature-footer"><span>{panel.status}</span><button onClick={() => onSelect(information ? 'replay' : 'guide')}>{information ? 'Ver demo táctica' : 'Cómo funciona'}<ArrowUpRight size={13} /></button></footer>
  </aside>;
}
