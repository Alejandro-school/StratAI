import React, { useEffect, useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import {
  ArrowLeft,
  Banknote,
  Crosshair,
  Flame,
  GitCompareArrows,
  LoaderCircle,
  Map,
  Repeat2,
  ShieldAlert,
  Swords,
  Target,
  Trophy,
  Users,
} from 'lucide-react';
import {
  COMPARISON_ATTRIBUTE_LABELS,
  buildComparisonDashboard,
} from '../data/performanceComparisonAdapters';
import { usePerformanceOverview, usePerformancePlayer } from '../hooks/usePerformanceData';
import PerformanceWeaponIcon from './statistics/PerformanceWeaponIcon';
import '../styles/performanceComparison.css';

const SECTION_ICONS = {
  aim: Crosshair,
  utility: ShieldAlert,
  clutching: Trophy,
  positioning: Repeat2,
  openings: Swords,
  economy: Banknote,
};

const formatMetricValue = (metric, value) => (
  `${metric.prefix || ''}${value.toFixed(metric.digits)}${metric.suffix || ''}`
);

const isUserWinner = (metric) => (
  metric.lowerIsBetter ? metric.user < metric.rival : metric.user > metric.rival
);

const MetricComparisonRow = ({ metric }) => {
  const scaleMax = Math.max(Math.abs(metric.user), Math.abs(metric.rival), 0.01);
  const userWidth = Math.max(3, (Math.abs(metric.user) / scaleMax) * 100);
  const rivalWidth = Math.max(3, (Math.abs(metric.rival) / scaleMax) * 100);
  const isTie = metric.user === metric.rival;
  const userLeads = isUserWinner(metric);

  return (
    <div className="pfcmp-metric-row">
      <div className={`pfcmp-metric-side is-user ${isTie ? 'is-tied' : userLeads ? 'is-leading' : ''}`}>
        <span className="pfcmp-metric-track"><i style={{ width: `${userWidth}%` }} /></span>
        <strong>{formatMetricValue(metric, metric.user)}</strong>
      </div>
      <div className="pfcmp-metric-label">
        <span>{metric.label}</span>
        {metric.lowerIsBetter && <small>menor es mejor</small>}
      </div>
      <div className={`pfcmp-metric-side is-rival ${isTie ? 'is-tied' : !userLeads ? 'is-leading' : ''}`}>
        <strong>{formatMetricValue(metric, metric.rival)}</strong>
        <span className="pfcmp-metric-track"><i style={{ width: `${rivalWidth}%` }} /></span>
      </div>
    </div>
  );
};

const ComparisonProfile = ({ profile, rival = false, accent }) => (
  <article className={`pfcmp-profile ${rival ? 'is-rival' : 'is-user'}`}>
    <header>
      <span className="pfcmp-profile-avatar" style={{ '--profile-accent': accent }}>
        {profile.initials}
      </span>
      <div>
        <small>{rival ? 'Referencia' : 'Tu rendimiento'}</small>
        <h2>{profile.handle}</h2>
      </div>
    </header>
    <div className="pfcmp-profile-facts">
      <div><small>Partidas</small><strong>{profile.matches}</strong></div>
      <div><small>Máx. bajas</small><strong>{profile.highKills}</strong></div>
      <div><small>Mejor racha</small><strong>{profile.streak}</strong></div>
    </div>
    <div className="pfcmp-profile-core">
      <div>
        <small>Rating</small>
        <strong>{profile.rating.toFixed(2)}</strong>
      </div>
      <div>
        <small>Win rate</small>
        <strong>{profile.winRate}%</strong>
      </div>
      <div>
        <small>Fortalezas</small>
        <span>{profile.topSkills.join(' · ')}</span>
      </div>
    </div>
  </article>
);

const AttributeMatrix = ({ dashboard }) => (
  <section className="pfcmp-attributes" id="pfcmp-overview">
    <header>
      <div>
        <span className="pfcmp-eyebrow">Lectura global</span>
        <h2>Atributos competitivos</h2>
      </div>
      <p>La misma escala para ambos jugadores. No son percentiles mezclados.</p>
    </header>
    <div>
      {COMPARISON_ATTRIBUTE_LABELS.map((label, index) => {
        const userValue = dashboard.user.attributes[index];
        const rivalValue = dashboard.rival.attributes[index];
        const verdict = userValue === rivalValue
          ? 'Igualado'
          : userValue > rivalValue
            ? 'Ventaja tuya'
            : `Ventaja ${dashboard.rival.name}`;
        return (
          <article key={label}>
            <strong className={userValue > rivalValue ? 'is-leading' : ''}>{userValue}</strong>
            <span className="pfcmp-attribute-track is-user"><i style={{ width: `${userValue}%` }} /></span>
            <div><span>{label}</span><small>{verdict}</small></div>
            <span className="pfcmp-attribute-track is-rival"><i style={{ width: `${rivalValue}%` }} /></span>
            <strong className={rivalValue > userValue ? 'is-leading' : ''}>{rivalValue}</strong>
          </article>
        );
      })}
    </div>
  </section>
);

const WeaponComparison = ({ weapons, rivalName }) => (
  <section className="pfcmp-weapons">
    <header>
      <span className="pfcmp-eyebrow">Arsenal principal</span>
      <h3>Rendimiento por arma</h3>
    </header>
    <div className="pfcmp-weapon-grid">
      {weapons.map(({ user, rival }) => (
        <article key={`${user.name}-${rival.name}`}>
          <div className="pfcmp-weapon-head">
            <div>
              <PerformanceWeaponIcon weapon={user.name} />
              <strong>{user.name}</strong>
              <small>Tú</small>
            </div>
            <span>VS</span>
            <div>
              <PerformanceWeaponIcon weapon={rival.name} />
              <strong>{rival.name}</strong>
              <small>{rivalName}</small>
            </div>
          </div>
          <div className="pfcmp-weapon-stats">
            <span><strong>{user.kills}</strong><small>Bajas</small><strong>{rival.kills}</strong></span>
            <span><strong>{user.damagePerKill.toFixed(1)}</strong><small>Daño / baja</small><strong>{rival.damagePerKill.toFixed(1)}</strong></span>
            <span><strong>{user.accuracy}%</strong><small>Precisión</small><strong>{rival.accuracy}%</strong></span>
            <span><strong>{user.headshots}%</strong><small>Headshots</small><strong>{rival.headshots}%</strong></span>
          </div>
        </article>
      ))}
    </div>
  </section>
);

const MapComparison = ({ dashboard, rivalName }) => (
  <section className="pfcmp-section pfcmp-maps" id="pfcmp-maps">
    <header className="pfcmp-section-head">
      <div className="pfcmp-section-title">
        <span><Map size={20} aria-hidden="true" /></span>
        <div><small>Pool procesado</small><h2>Mapas</h2></div>
      </div>
      <p>Rating, win rate y tamaño real de la muestra por escenario.</p>
    </header>
    <div className="pfcmp-map-grid">
      {dashboard.maps.map(({ contextId, presentation, user, rival }) => {
        const userRating = Number(user.avg_rating) || 0;
        const rivalRating = Number(rival.avg_rating) || 0;
        const isTie = userRating === rivalRating;
        const userLeads = userRating > rivalRating;
        return (
          <article key={contextId} className={isTie ? 'is-tied' : userLeads ? 'user-leads' : 'rival-leads'}>
            <img src={presentation.radar} alt="" aria-hidden="true" />
            <header>
              <strong>{presentation.name}</strong>
              <span>{isTie ? 'Igualado' : userLeads ? 'Ventaja tuya' : `Ventaja ${rivalName}`}</span>
            </header>
            <div className="pfcmp-map-rating">
              <div><small>Tú</small><strong>{userRating.toFixed(2)}</strong><span>{Number(user.win_rate) || 0}% WR</span></div>
              <i>{(userRating - rivalRating) >= 0 ? '+' : ''}{(userRating - rivalRating).toFixed(2)}</i>
              <div><small>{rivalName}</small><strong>{rivalRating.toFixed(2)}</strong><span>{Number(rival.win_rate) || 0}% WR</span></div>
            </div>
            <footer>{Number(user.matches) || 0} frente a {Number(rival.matches) || 0} partidas</footer>
          </article>
        );
      })}
    </div>
  </section>
);

const DetailSection = ({ section, rivalName }) => {
  const Icon = SECTION_ICONS[section.id] || Target;
  const decisiveMetrics = section.metrics.filter((metric) => metric.user !== metric.rival);
  const userWins = decisiveMetrics.filter(isUserWinner).length;
  const rivalWins = decisiveMetrics.length - userWins;
  const verdict = userWins === rivalWins
    ? 'Muy igualado'
    : userWins > rivalWins
      ? userWins - rivalWins >= 3 ? 'Ventaja clara' : 'Ventaja ligera'
      : rivalWins - userWins >= 3 ? `Ventaja clara ${rivalName}` : `Ventaja ligera ${rivalName}`;

  return (
    <section className="pfcmp-section" id={`pfcmp-${section.id}`}>
      <header className="pfcmp-section-head">
        <div className="pfcmp-section-title">
          <span><Icon size={20} aria-hidden="true" /></span>
          <div><small>{section.subtitle}</small><h2>{section.title}</h2></div>
        </div>
        <div className="pfcmp-section-verdict">
          <small>{verdict}</small>
          <strong>{section.scoreUser}</strong>
          <i />
          <strong>{section.scoreRival}</strong>
        </div>
      </header>
      <div className="pfcmp-metrics">
        {section.metrics.map((item) => (
          <MetricComparisonRow key={item.label} metric={item} />
        ))}
      </div>
    </section>
  );
};

const PerformanceComparisonView = ({
  player,
  onExit,
  onChangePlayer,
}) => {
  const [range, setRange] = useState(30);
  const [activeNav, setActiveNav] = useState('overview');
  const userQuery = usePerformanceOverview({ limit: range });
  const rivalQuery = usePerformancePlayer(player.steam_id || player.id, { limit: range });
  const dashboard = useMemo(
    () => (userQuery.data && rivalQuery.data
      ? buildComparisonDashboard(userQuery.data, rivalQuery.data, player, range)
      : null),
    [player, range, rivalQuery.data, userQuery.data],
  );
  const attributeWins = dashboard?.user.attributes.filter(
    (value, index) => value > dashboard.rival.attributes[index],
  ).length || 0;
  const rivalAttributeWins = dashboard?.rival.attributes.filter(
    (value, index) => value > dashboard.user.attributes[index],
  ).length || 0;
  const tiedAttributes = dashboard
    ? dashboard.user.attributes.length - attributeWins - rivalAttributeWins
    : 0;

  const navItems = [
    { id: 'overview', label: 'Resumen' },
    { id: 'aim', label: 'Puntería' },
    { id: 'utility', label: 'Utilidad' },
    { id: 'maps', label: 'Mapas' },
    { id: 'clutching', label: 'Clutches' },
    { id: 'positioning', label: 'Posicionamiento' },
    { id: 'openings', label: 'Aperturas' },
    { id: 'economy', label: 'Economía' },
  ];

  useEffect(() => {
    const sections = navItems
      .map(({ id }) => document.getElementById(`pfcmp-${id}`))
      .filter(Boolean);
    const updateActiveSection = () => {
      const currentSection = sections.reduce((current, section) => (
        section.getBoundingClientRect().top <= 150 ? section : current
      ), sections[0]);
      if (currentSection) {
        setActiveNav(currentSection.id.replace('pfcmp-', ''));
      }
    };

    updateActiveSection();
    window.addEventListener('scroll', updateActiveSection, { passive: true });
    return () => window.removeEventListener('scroll', updateActiveSection);
  }, [dashboard]);

  const scrollToSection = (sectionId) => {
    const section = document.getElementById(`pfcmp-${sectionId}`);
    if (!section) return;
    setActiveNav(sectionId);
    section.scrollIntoView();
  };

  return (
    <main className="pfcmp">
      <header className="pfcmp-toolbar">
        <button type="button" onClick={onExit}>
          <ArrowLeft size={17} aria-hidden="true" /> Volver a mi rendimiento
        </button>
        <div>
          <span><GitCompareArrows size={15} aria-hidden="true" /> Centro comparativo</span>
          <strong>Tú vs {player.name}</strong>
        </div>
        <div className="pfcmp-toolbar-actions">
          <div className="pfcmp-range" aria-label="Ventana de partidas">
            {[30, 60].map((value) => (
              <button
                type="button"
                key={value}
                className={range === value ? 'is-active' : ''}
                onClick={() => setRange(value)}
              >
                {value} partidas
              </button>
            ))}
          </div>
          <button type="button" onClick={onChangePlayer}>
            <Users size={16} aria-hidden="true" /> Cambiar jugador
          </button>
        </div>
      </header>

      {!dashboard && (
        <section className="pfcmp-data-state" role="status">
          <LoaderCircle size={26} className="pf3-spin" />
          <div>
            <strong>
              {userQuery.error || rivalQuery.error
                ? 'No se pudo construir la comparación'
                : 'Agregando las dos muestras reales'}
            </strong>
            <span>
              {userQuery.error || rivalQuery.error
                ? 'Revisa que ambos jugadores tengan demos procesadas.'
                : 'Duelos, mapas, armas, utilidad y economía.'}
            </span>
          </div>
          {(userQuery.error || rivalQuery.error) && (
            <button
              type="button"
              onClick={() => {
                userQuery.refetch();
                rivalQuery.refetch();
              }}
            >
              Reintentar
            </button>
          )}
        </section>
      )}

      {dashboard && (
        <>
          <motion.section
            className="pfcmp-hero"
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
          >
            <ComparisonProfile profile={dashboard.user} accent="#59e3ff" />
            <div className="pfcmp-hero-verdict">
              <span>VS</span>
              <strong>{attributeWins}–{rivalAttributeWins}</strong>
              <small>
                categorías a tu favor{tiedAttributes ? ` · ${tiedAttributes} igualada` : ''}
              </small>
              <p>
                {attributeWins >= 4
                  ? `Tu perfil es más completo; ${player.name} concentra su ventaja en áreas concretas.`
                  : 'Comparación igualada: las diferencias dependen de cada contexto.'}
              </p>
            </div>
            <ComparisonProfile profile={dashboard.rival} rival accent={player.accent} />
          </motion.section>

          <nav className="pfcmp-nav" aria-label="Secciones comparativas">
            {navItems.map((item) => (
              <button
                type="button"
                key={item.id}
                className={activeNav === item.id ? 'is-active' : ''}
                onClick={() => scrollToSection(item.id)}
              >
                {item.label}
              </button>
            ))}
          </nav>

          <AttributeMatrix dashboard={dashboard} />

          {dashboard.sections.slice(0, 2).map((section) => (
            <React.Fragment key={section.id}>
              <DetailSection section={section} rivalName={player.name} />
              {section.id === 'aim' && (
                <WeaponComparison weapons={dashboard.weapons} rivalName={player.name} />
              )}
            </React.Fragment>
          ))}

          <MapComparison dashboard={dashboard} rivalName={player.name} />

          {dashboard.sections.slice(2).map((section) => (
            <DetailSection key={section.id} section={section} rivalName={player.name} />
          ))}

          <footer className="pfcmp-footer">
            <Flame size={17} aria-hidden="true" />
            Datos reales procesados por Go sobre una ventana equivalente. Las muestras inferiores a cinco partidas se interpretan como orientativas.
          </footer>
        </>
      )}
    </main>
  );
};

export default PerformanceComparisonView;
