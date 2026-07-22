import React from 'react';
import { Activity, Crosshair, Map, Radar, Shield, Target, Zap } from 'lucide-react';
import { formatDecimal, formatInteger, formatMapName, formatPercent } from '../../utils/performanceFormatters';

const AREA_ICONS = {
  overview: Radar,
  combat: Shield,
  aim: Crosshair,
  weapons: Target,
  maps: Map,
  utility: Zap,
  compare: Activity,
};

export const BriefingPanel = ({ eyebrow, title, children, action, className = '' }) => (
  <section className={`p-briefing-panel ${className}`}>
    <div className="p-briefing-panel-head">
      <div>
        {eyebrow && <span className="p-briefing-eyebrow">{eyebrow}</span>}
        {title && <h2 className="p-briefing-title">{title}</h2>}
      </div>
      {action}
    </div>
    {children}
  </section>
);

export const InsightCard = ({ tone = 'neutral', label, value, detail, icon: Icon = Activity }) => (
  <article className={`p-insight-tile ${tone}`}>
    <span className="p-insight-icon"><Icon size={16} /></span>
    <span className="p-insight-label">{label}</span>
    <strong className="p-insight-value">{value}</strong>
    {detail && <span className="p-insight-detail">{detail}</span>}
  </article>
);

export const MetricRibbon = ({ metrics = [] }) => (
  <div className="p-command-ribbon">
    {metrics.map((metric) => (
      <div key={metric.label} className="p-command-ribbon-cell">
        <span>{metric.label}</span>
        <strong>{metric.value}</strong>
        {metric.detail && <small>{metric.detail}</small>}
      </div>
    ))}
  </div>
);

export const TrendStrip = ({ matches = [] }) => (
  <div className="p-trend-strip" aria-label="Forma reciente">
    {matches.slice(0, 10).map((match, index) => (
      <span
        key={match.match_id || `${match.map}-${index}`}
        className={`p-trend-node ${match.result === 'W' ? 'win' : 'loss'}`}
        title={`${formatMapName(match.map)} ${match.team_score ?? 0}-${match.opponent_score ?? 0}`}
      />
    ))}
    {matches.length === 0 && <span className="p-trend-empty">Sin historial reciente</span>}
  </div>
);

export const ImpactMatrix = ({ areas = [], onSelectArea }) => (
  <div className="p-impact-matrix">
    {areas.map((area) => {
      const Icon = AREA_ICONS[area.id] || Activity;
      return (
        <button
          key={area.id}
          type="button"
          className="p-impact-card"
          onClick={() => onSelectArea?.(area.id)}
        >
          <span className="p-impact-card-icon"><Icon size={16} /></span>
          <span className="p-impact-card-label">{area.label}</span>
          <strong className="p-impact-card-score">{formatDecimal(area.score, 0)}</strong>
          <span className="p-impact-card-track"><span style={{ width: `${area.score}%` }} /></span>
          <small>{area.detail}</small>
        </button>
      );
    })}
  </div>
);

export const TrainingAreaNav = ({ tabs = [], activeTab, onChange, areas = [] }) => (
  <nav className="p-command-nav" role="tablist" aria-label="Secciones de rendimiento">
    {tabs.map((tab) => {
      const area = areas.find((item) => item.id === tab.id);
      return (
        <button
          key={tab.id}
          id={`p-tab-${tab.id}`}
          type="button"
          role="tab"
          aria-selected={activeTab === tab.id}
          aria-controls={`p-panel-${tab.id}`}
          className={`p-command-tab ${activeTab === tab.id ? 'active' : ''}`}
          onClick={() => onChange(tab.id)}
        >
          <tab.icon size={15} />
          <span>{tab.label}</span>
          {area && <small>{formatDecimal(area.score, 0)}</small>}
        </button>
      );
    })}
  </nav>
);

export const PerformanceHero = ({ viewModel, overview = {}, maps = [], onSelectArea }) => {
  const bestMapName = viewModel.bestMap ? formatMapName(viewModel.bestMap.map) : 'Sin mapa lider';

  return (
    <header className="p-command-center">
      <div className="p-command-copy">
        <span className="p-command-kicker">Performance Command Center</span>
        <h1>{viewModel.verdict}</h1>
        <p>{viewModel.summary}</p>
        <div className="p-command-chips">
          <span>{viewModel.form.label}: {formatDecimal(viewModel.form.winRate, 0)}%</span>
          <span>Mejor mapa: {bestMapName}</span>
          <span>Foco: {viewModel.focusArea?.label || 'Sin foco'}</span>
        </div>
      </div>

      <div className="p-command-rating">
        <div
          className="p-command-ring"
          style={{
            '--p-ring-angle': viewModel.ratingAngle,
            '--p-ring-color': viewModel.rating >= 1 ? 'var(--p-cyan)' : 'var(--p-amber)',
          }}
        >
          <div>
            <span>HLTV</span>
            <strong>{formatDecimal(viewModel.rating, 2)}</strong>
            <small>{viewModel.ratingBadge.label}</small>
          </div>
        </div>
      </div>

      <div className="p-command-side">
        <MetricRibbon
          metrics={[
            { label: 'Partidas', value: formatInteger(overview.total_matches), detail: `${formatInteger(overview.wins)}V ${formatInteger(overview.losses)}D` },
            { label: 'K/D', value: formatDecimal(overview.kd_ratio, 2), detail: `${formatInteger(overview.kills)} bajas` },
            { label: 'KAST', value: formatPercent(overview.kast), detail: viewModel.sideRead },
          ]}
        />
        <TrendStrip matches={viewModel.form.recent} />
        <ImpactMatrix areas={viewModel.areas} onSelectArea={onSelectArea} />
        {maps.length === 0 && <span className="p-command-footnote">Juega mas mapas para desbloquear lecturas de pool.</span>}
      </div>
    </header>
  );
};
