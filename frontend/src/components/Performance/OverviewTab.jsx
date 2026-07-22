import React, { useMemo, useState } from 'react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  ReferenceLine,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { Award, Crosshair, Swords, Trophy } from 'lucide-react';
import {
  CHART_AXIS_TICK,
  CHART_CURSOR_STYLE,
  CHART_GRID_STROKE,
  RECHARTS_TOOLTIP_STYLE,
  TOOLTIP_ITEM_STYLE,
  TOOLTIP_LABEL_STYLE,
  formatDecimal,
  formatInteger,
  formatMapName,
  formatPercent,
  formatRelativeDate,
} from '../../utils/performanceFormatters';
import { getQualityLabel } from '../../utils/performanceBenchmarks';
import { MetricCell } from './StatPill';
import SectionBlock from './SectionBlock';
import { BriefingPanel, ImpactMatrix, InsightCard, TrendStrip } from './PerformanceBriefing';

const MAX_HISTORY_ROWS = 20;

const TONE_TO_COLOR = {
  excellent: 'var(--p-green)',
  good: 'var(--p-blue)',
  average: 'var(--p-yellow)',
  poor: 'var(--p-red)',
  neutral: 'var(--p-accent)',
};

const buildSpark = (matches = [], field) => matches.slice(0, 10).reverse().map((match) => Number(match[field] || 0));

const parseScorePair = (value) => {
  if (typeof value !== 'string' || !value.includes('-')) return null;
  const [left, right] = value.split('-').map((part) => Number(part.trim()));
  return Number.isFinite(left) && Number.isFinite(right) ? [left, right] : null;
};

const getMatchScore = (match = {}) => {
  if (match.team_score != null || match.opponent_score != null) {
    return `${match.team_score ?? 0}-${match.opponent_score ?? 0}`;
  }

  const direct = parseScorePair(match.final_score || match.score);
  if (direct) return `${direct[0]}-${direct[1]}`;

  const ctScore = match.ct_score ?? match.score_ct;
  const tScore = match.t_score ?? match.score_t;
  if (ctScore != null || tScore != null) {
    const isT = match.user_team === 'T' || match.team === 'T';
    return isT ? `${tScore ?? 0}-${ctScore ?? 0}` : `${ctScore ?? 0}-${tScore ?? 0}`;
  }

  return '0-0';
};

const getStreak = (matches = []) => {
  if (!matches.length) {
    return { count: 0, result: null, label: 'Sin racha activa' };
  }

  const result = matches[0].result;
  let count = 0;

  for (const match of matches) {
    if (match.result !== result) {
      break;
    }

    count += 1;
  }

  return {
    count,
    result,
    label: `${count}${result === 'W' ? 'V' : 'D'} seguidas`,
  };
};

const OverviewTab = ({
  overview = {},
  sides = {},
  matchHistory = [],
  maps = [],
  aim = {},
  trends = {},
  viewModel,
  onSelectArea,
}) => {
  const [expandedMatchId, setExpandedMatchId] = useState(null);
  const [showAllHistory, setShowAllHistory] = useState(false);
  const [seriesVisibility, setSeriesVisibility] = useState({ rating: true, adr: true });

  const chartData = useMemo(
    () => [...matchHistory].slice(0, 12).reverse().map((match, index) => ({
      index: index + 1,
      map: formatMapName(match.map),
      score: getMatchScore(match),
      rating: Number(match.hltv_rating || 0),
      adr: Number(match.adr || 0),
      kd: Number(match.kd_ratio || 0),
      result: match.result,
    })),
    [matchHistory],
  );

  const bestMap = useMemo(() => {
    if (!maps.length) return null;

    return [...maps].sort((left, right) => {
      const leftScore = Number(left.avg_rating || 0) * 100 + Number(left.win_rate || 0) + Number(left.matches || 0);
      const rightScore = Number(right.avg_rating || 0) * 100 + Number(right.win_rate || 0) + Number(right.matches || 0);
      return rightScore - leftScore;
    })[0];
  }, [maps]);

  const displayedHistory = showAllHistory ? matchHistory.slice(0, MAX_HISTORY_ROWS) : matchHistory.slice(0, 10);
  const streak = getStreak(matchHistory);
  const ratingBadge = getQualityLabel(Number(overview.hltv_rating || 0), 'hltv_rating');
  const ratingColor = TONE_TO_COLOR[ratingBadge.tone] || 'var(--p-accent)';
  const ratingAngle = `${Math.min(Math.max((Number(overview.hltv_rating || 0) / 2) * 360, 0), 360)}deg`;
  const sideGap = Number(sides.ct_rating || 0) - Number(sides.t_rating || 0);
  const sideLeader = sideGap === 0 ? 'equilibrado' : sideGap > 0 ? 'CT' : 'T';

  const toggleSeries = (key) => {
    setSeriesVisibility((current) => ({
      ...current,
      [key]: !current[key],
    }));
  };

  return (
    <div className="p-section">
      {viewModel && (
        <BriefingPanel eyebrow="Diagnostico tactico" title="Lectura priorizada">
          <div className="p-diagnosis-grid">
            <div className="p-diagnosis-main">
              <div className="p-diagnosis-verdict">
                <span className="p-overview-kicker">Estado competitivo</span>
                <h2>{viewModel.verdict}</h2>
                <p>{viewModel.summary}</p>
              </div>
              <TrendStrip matches={viewModel.form.recent} />
              <ImpactMatrix areas={viewModel.areas} onSelectArea={onSelectArea} />
            </div>
            <div className="p-diagnosis-side">
              <InsightCard
                tone="good"
                icon={Trophy}
                label="Fortaleza principal"
                value={viewModel.bestArea?.label || 'Sin muestra'}
                detail={viewModel.bestArea?.detail}
              />
              <InsightCard
                tone="warning"
                icon={Crosshair}
                label="Foco de entrenamiento"
                value={viewModel.focusArea?.label || 'Sin foco'}
                detail={viewModel.focusArea?.detail}
              />
              <InsightCard
                tone="neutral"
                icon={Swords}
                label="Lectura por bando"
                value={sideLeader === 'equilibrado' ? 'Equilibrado' : sideLeader}
                detail={viewModel.sideRead}
              />
            </div>
          </div>
        </BriefingPanel>
      )}

      <div className="p-overview-hero-grid">
        <div className="p-card p-overview-lead-card">
          <div className="p-overview-lead-copy">
            <span className="p-overview-kicker">Perfil competitivo</span>
            <h2 className="p-overview-lead-title">Tu rendimiento actual está en {ratingBadge.label.toLowerCase()}</h2>
            <p className="p-overview-lead-text">
              {overview.total_matches || 0} partidas analizadas, {overview.wins || 0} victorias y un KAST del {formatPercent(overview.kast)}.
            </p>
            <div className="p-overview-chip-row">
              <span className="p-overview-chip">
                <Trophy size={14} />
                {bestMap ? `Mejor mapa: ${formatMapName(bestMap.map)}` : 'Todavía sin mejor mapa'}
              </span>
              <span className="p-overview-chip">
                <Swords size={14} />
                {formatInteger(overview.kills)} bajas totales
              </span>
            </div>
          </div>

          <div className="p-score-ring" style={{ '--p-ring-angle': ratingAngle, '--p-ring-color': ratingColor }}>
            <div className="p-score-ring-core">
              <span className="p-score-ring-label">HLTV</span>
              <span className="p-score-ring-value">{formatDecimal(overview.hltv_rating, 2)}</span>
              <span className={`p-badge ${ratingBadge.tone}`}>{ratingBadge.label}</span>
            </div>
          </div>
        </div>

        <div className="p-overview-insight-grid">
          <div className="p-card p-insight-card">
              <span className="p-insight-label">% de victorias</span>
            <strong className="p-insight-value">{formatPercent(overview.win_rate)}</strong>
            <span className="p-insight-sub">{overview.wins || 0}V · {overview.losses || 0}D</span>
          </div>
          <div className="p-card p-insight-card">
            <span className="p-insight-label">Headshots</span>
            <strong className="p-insight-value">{formatPercent(overview.hs_pct)}</strong>
            <span className="p-insight-sub">{formatInteger(overview.total_damage)} de daño total</span>
          </div>
          <div className="p-card p-insight-card">
            <span className="p-insight-label">Precisión</span>
            <strong className="p-insight-value">{formatPercent(aim.accuracy_overall)}</strong>
            <span className="p-insight-sub">TTD {formatDecimal(aim.time_to_damage_avg_ms, 0, ' ms')}</span>
          </div>
          <div className="p-card p-insight-card">
            <span className="p-insight-label">Impacto</span>
            <strong className="p-insight-value">{formatDecimal(overview.impact_rating, 2)}</strong>
            <span className="p-insight-sub">ADR {formatDecimal(overview.adr, 1)}</span>
          </div>
        </div>
      </div>

      <div className="p-metric-row">
        <MetricCell
          value={formatDecimal(overview.hltv_rating, 2)}
          label="Rating HLTV"
          badge={ratingBadge}
          trend={trends.hltv_rating}
          sparkData={buildSpark(matchHistory, 'hltv_rating')}
          sparkTone="var(--p-accent)"
        />
        <MetricCell
          value={formatDecimal(overview.kd_ratio, 2)}
          label="K / D"
          badge={getQualityLabel(Number(overview.kd_ratio || 0), 'kd_ratio')}
          trend={trends.kd_ratio}
          sparkData={buildSpark(matchHistory, 'kd_ratio')}
          sparkTone="var(--p-blue)"
        />
        <MetricCell
          value={formatDecimal(overview.adr, 1)}
          label="ADR"
          badge={getQualityLabel(Number(overview.adr || 0), 'adr')}
          trend={trends.adr}
          sparkData={buildSpark(matchHistory, 'adr')}
          sparkTone="var(--p-orange)"
        />
        <MetricCell
          value={formatPercent(overview.win_rate)}
          label="Victorias"
          sub={`${overview.wins || 0}V · ${overview.losses || 0}D · ${overview.total_matches || 0} partidas`}
          badge={getQualityLabel(Number(overview.win_rate || 0), 'win_rate')}
          sparkData={matchHistory.slice(0, 10).reverse().map((match) => (match.result === 'W' ? 1 : 0))}
          sparkTone="var(--p-green)"
        />
        <MetricCell
          value={formatPercent(overview.kast)}
          label="KAST"
          badge={getQualityLabel(Number(overview.kast || 0), 'kast')}
          sparkData={buildSpark(matchHistory, 'accuracy_overall')}
          sparkTone="var(--p-yellow)"
        />
      </div>

      <SectionBlock title="Lectura de forma">
        <div className="p-grid p-grid-dashboard">
          <div className="p-card p-card--chart p-trend-card">
            <div className="p-card-toolbar">
              <div>
                <p className="p-card-title">Progresión reciente</p>
                <p className="p-card-subtitle">Últimas 12 partidas con referencias de nivel medio</p>
              </div>
              <div className="p-toggle-row">
                <button
                  type="button"
                  className={`p-chart-toggle ${seriesVisibility.rating ? 'is-active' : ''}`}
                  onClick={() => toggleSeries('rating')}
                >
                  Rating
                </button>
                <button
                  type="button"
                  className={`p-chart-toggle ${seriesVisibility.adr ? 'is-active' : ''}`}
                  onClick={() => toggleSeries('adr')}
                >
                  ADR
                </button>
              </div>
            </div>

            <div className="p-chart-wrap p-chart-wrap--tall">
              {chartData.length > 0 ? (
                <ResponsiveContainer width="100%" height={240}>
                  <AreaChart data={chartData} margin={{ top: 8, right: 8, left: -18, bottom: 0 }}>
                    <defs>
                      <linearGradient id="p-rating-gradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="#6366f1" stopOpacity={0.42} />
                        <stop offset="100%" stopColor="#6366f1" stopOpacity={0.02} />
                      </linearGradient>
                      <linearGradient id="p-adr-gradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="#f97316" stopOpacity={0.34} />
                        <stop offset="100%" stopColor="#f97316" stopOpacity={0.02} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid stroke={CHART_GRID_STROKE} vertical={false} />
                    <XAxis dataKey="index" tick={CHART_AXIS_TICK} axisLine={false} tickLine={false} />
                    <YAxis yAxisId="rating" tick={CHART_AXIS_TICK} axisLine={false} tickLine={false} domain={[0.5, 'auto']} />
                    <YAxis yAxisId="adr" orientation="right" tick={CHART_AXIS_TICK} axisLine={false} tickLine={false} domain={[40, 'auto']} />
                    <ReferenceLine yAxisId="rating" y={1} stroke="rgba(148, 163, 184, 0.25)" strokeDasharray="4 4" />
                    <ReferenceLine yAxisId="adr" y={75} stroke="rgba(148, 163, 184, 0.18)" strokeDasharray="4 4" />
                    <Tooltip
                      contentStyle={RECHARTS_TOOLTIP_STYLE}
                      labelStyle={TOOLTIP_LABEL_STYLE}
                      itemStyle={TOOLTIP_ITEM_STYLE}
                      cursor={CHART_CURSOR_STYLE}
                      labelFormatter={(value, payload) => {
                        const row = payload?.[0]?.payload;
                        return row ? `${row.map} · ${row.score}` : `Partida ${value}`;
                      }}
                      formatter={(value, key, payload) => {
                        if (key === 'rating') return [formatDecimal(value, 2), 'Rating'];
                        if (key === 'adr') return [formatDecimal(value, 1), 'ADR'];
                        return [value, payload?.dataKey];
                      }}
                    />
                    {seriesVisibility.rating && (
                      <Area
                        yAxisId="rating"
                        type="monotone"
                        dataKey="rating"
                        stroke="#6366f1"
                        fill="url(#p-rating-gradient)"
                        strokeWidth={2.2}
                      />
                    )}
                    {seriesVisibility.adr && (
                      <Area
                        yAxisId="adr"
                        type="monotone"
                        dataKey="adr"
                        stroke="#f97316"
                        fill="url(#p-adr-gradient)"
                        strokeWidth={2.2}
                      />
                    )}
                  </AreaChart>
                </ResponsiveContainer>
              ) : (
                <div className="p-empty p-empty--tall">
                  <Award className="p-empty-icon" size={28} />
                  <span>Juega más partidas para ver una progresión consistente.</span>
                </div>
              )}
            </div>
          </div>

          <div className="p-stack-col">
            <div className="p-card p-form-card">
              <p className="p-card-title">Estado de forma</p>
              <div className="p-form-meta">
                <span className={`p-form-streak ${streak.result === 'W' ? 'win' : streak.result === 'L' ? 'loss' : ''}`}>
                  {streak.result === 'W' ? 'Caliente' : streak.result === 'L' ? 'En ajuste' : 'Neutro'}
                </span>
                <strong>{streak.label}</strong>
              </div>
              <div className="p-form-strip">
                {matchHistory.slice(0, 20).map((match) => (
                  <span
                    key={match.match_id}
                    className={`p-form-dot ${match.result === 'W' ? 'win' : 'loss'}`}
                    title={`${formatMapName(match.map)} · ${match.result === 'W' ? 'Victoria' : 'Derrota'}`}
                  />
                ))}
              </div>
            </div>

            <div className="p-card">
              <p className="p-card-title">Rendimiento por bando</p>
              <div className="p-side-compare-list">
                <div className="p-side-compare-item">
                  <div className="p-side-compare-head">
                    <span>Rating</span>
                    <strong>{formatDecimal(sides.ct_rating, 2)} CT · {formatDecimal(sides.t_rating, 2)} T</strong>
                  </div>
                  <div className="p-split-bar p-split-bar--lg">
                    <div
                      className="p-split-bar-ct"
                      style={{ width: `${(() => {
                        const ct = Number(sides.ct_rating || 0);
                        const t = Number(sides.t_rating || 0);
                        const total = ct + t;
                        return total > 0 ? (ct / total) * 100 : 50;
                      })()}%` }}
                    />
                    <div
                      className="p-split-bar-t"
                      style={{ width: `${(() => {
                        const ct = Number(sides.ct_rating || 0);
                        const t = Number(sides.t_rating || 0);
                        const total = ct + t;
                        return total > 0 ? (t / total) * 100 : 50;
                      })()}%` }}
                    />
                  </div>
                </div>
                <div className="p-side-compare-item">
                  <div className="p-side-compare-head">
                    <span>ADR</span>
                    <strong>{formatDecimal(sides.ct_adr, 1)} CT · {formatDecimal(sides.t_adr, 1)} T</strong>
                  </div>
                  <div className="p-split-bar p-split-bar--lg">
                    <div
                      className="p-split-bar-ct"
                      style={{ width: `${(() => {
                        const ct = Number(sides.ct_adr || 0);
                        const t = Number(sides.t_adr || 0);
                        const total = ct + t;
                        return total > 0 ? (ct / total) * 100 : 50;
                      })()}%` }}
                    />
                    <div
                      className="p-split-bar-t"
                      style={{ width: `${(() => {
                        const ct = Number(sides.ct_adr || 0);
                        const t = Number(sides.t_adr || 0);
                        const total = ct + t;
                        return total > 0 ? (t / total) * 100 : 50;
                      })()}%` }}
                    />
                  </div>
                </div>
              </div>
              <p className="p-side-summary">
                {sideLeader === 'equilibrado'
                  ? 'Tu impacto se reparte de forma muy pareja entre CT y T.'
                  : `Ahora mismo rindes mejor en ${sideLeader} por ${formatDecimal(Math.abs(sideGap), 2)} puntos de rating.`}
              </p>
            </div>
          </div>
        </div>
      </SectionBlock>

      <SectionBlock title="Historial reciente" count={`${Math.min(matchHistory.length, MAX_HISTORY_ROWS)} partidas`}>
        {displayedHistory.length === 0 ? (
          <div className="p-card">
            <div className="p-empty">
              <Crosshair className="p-empty-icon" size={28} />
              <span>Aun no hay suficiente historial procesado para construir patrones.</span>
            </div>
          </div>
        ) : (
          <div className="p-card p-history-card">
            <div className="p-history-list">
              {displayedHistory.map((match) => {
                const isExpanded = expandedMatchId === match.match_id;
                const isWin = match.result === 'W';

                return (
                  <button
                    key={match.match_id}
                    type="button"
                    className={`p-history-row ${isExpanded ? 'is-expanded' : ''}`}
                    onClick={() => setExpandedMatchId(isExpanded ? null : match.match_id)}
                  >
                    <div className="p-history-row-main">
                      <span className={`p-match-result-dot ${isWin ? 'win' : 'loss'}`} />
                      <span className="p-history-map">{formatMapName(match.map)}</span>
                      <span className={`p-history-score ${isWin ? 'win' : 'loss'}`}>
                        {getMatchScore(match)}
                      </span>
                      <span className="p-history-kda">
                        {match.kills ?? 0}/{match.deaths ?? 0}/{match.assists ?? 0}
                      </span>
                      <span className="p-history-stat">{formatDecimal(match.kd_ratio, 2)} K/D</span>
                      <span className="p-history-stat">{formatDecimal(match.adr, 1)} ADR</span>
                      <span className="p-history-stat">{formatDecimal(match.hltv_rating, 2)} RT</span>
                      <span className="p-history-date">{formatRelativeDate(match.date)}</span>
                    </div>

                    {isExpanded && (
                      <div className="p-history-details">
                        <div className="p-num-grid p-grid-4">
                          <div className="p-num-cell">
                            <span className="p-num-cell-value">{formatPercent(match.hs_percentage)}</span>
                            <span className="p-num-cell-label">Headshots</span>
                          </div>
                          <div className="p-num-cell">
                            <span className="p-num-cell-value">{formatPercent(match.accuracy_overall)}</span>
                            <span className="p-num-cell-label">Precisión</span>
                          </div>
                          <div className="p-num-cell">
                            <span className="p-num-cell-value">{formatDecimal(match.hltv_rating, 2)}</span>
                            <span className="p-num-cell-label">Rating HLTV</span>
                          </div>
                          <div className="p-num-cell">
                            <span className="p-num-cell-value">{formatDecimal(match.adr, 1)}</span>
                            <span className="p-num-cell-label">ADR</span>
                          </div>
                        </div>
                      </div>
                    )}
                  </button>
                );
              })}
            </div>

            {matchHistory.length > 10 && (
              <div className="p-history-actions">
                <button
                  type="button"
                  className="p-ghost-btn"
                  onClick={() => setShowAllHistory((current) => !current)}
                >
                  {showAllHistory ? 'Mostrar menos' : 'Ver hasta 20 partidas'}
                </button>
              </div>
            )}
          </div>
        )}
      </SectionBlock>
    </div>
  );
};

export default OverviewTab;
