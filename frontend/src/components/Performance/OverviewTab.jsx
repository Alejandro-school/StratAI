import React from 'react';
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { TrendingUp } from 'lucide-react';
import {
  RECHARTS_TOOLTIP_STYLE,
  TOOLTIP_LABEL_STYLE,
  TOOLTIP_ITEM_STYLE,
  CHART_AXIS_TICK,
  CHART_GRID_STROKE,
  CHART_CURSOR_STYLE,
  formatDecimal,
  formatMapName,
  formatPercent,
  formatRelativeDate,
} from '../../utils/performanceFormatters';
import { getQualityLabel } from '../../utils/performanceBenchmarks';
import { MetricCell } from './StatPill';
import SectionBlock from './SectionBlock';

const OverviewTab = ({ overview = {}, sides = {}, matchHistory = [], trends = {} }) => {
  const chartData = [...matchHistory].reverse().map((m, i) => ({
    index: i + 1,
    rating: Number(m.hltv_rating || 0),
    adr: Number(m.adr || 0),
  }));

  const sideMetrics = [
    { label: 'Rating HLTV', ct: Number(sides.ct_rating || 0), t: Number(sides.t_rating || 0), decimals: 2 },
    { label: 'ADR',         ct: Number(sides.ct_adr || 0),    t: Number(sides.t_adr || 0),    decimals: 1 },
  ];

  const wr = Number(overview.win_rate || 0);
  const kd = Number(overview.kd_ratio || 0);

  return (
    <div className="p-section">

      {/* */}
      <div className="p-metric-row">
        <MetricCell
          value={formatDecimal(overview.hltv_rating, 2)}
          label="Rating HLTV"
          badge={getQualityLabel(Number(overview.hltv_rating || 0), 'hltv_rating')}
          trend={trends.hltv_rating}
        />
        <MetricCell
          value={formatDecimal(kd, 2)}
          label="K / D"
          badge={getQualityLabel(kd, 'kd_ratio')}
          trend={trends.kd_ratio}
        />
        <MetricCell
          value={formatDecimal(overview.adr, 1)}
          label="ADR"
          badge={getQualityLabel(Number(overview.adr || 0), 'adr')}
          trend={trends.adr}
        />
        <MetricCell
          value={formatPercent(wr)}
          label="Victorias"
          sub={`${overview.wins || 0}W · ${overview.losses || 0}L · ${overview.total_matches || 0} partidas`}
          badge={getQualityLabel(wr, 'win_rate')}
        />
        <MetricCell
          value={formatPercent(overview.kast)}
          label="KAST"
          badge={getQualityLabel(Number(overview.kast || 0), 'kast')}
        />
      </div>

      {/* -- Historial -- */}
      <SectionBlock title="Historial">
        <div className="p-grid p-grid-wide">

          {/* Progression chart */}
          <div className="p-card p-card--chart">
            <p className="p-card-title">Progresión de Rating y ADR</p>
            <div className="p-chart-wrap">
              {chartData.length > 0 ? (
                <ResponsiveContainer width="100%" height={180}>
                  <LineChart data={chartData} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
                    <CartesianGrid stroke={CHART_GRID_STROKE} horizontal vertical={false} />
                    <XAxis dataKey="index" tick={CHART_AXIS_TICK} axisLine={false} tickLine={false} />
                    <YAxis yAxisId="left"  tick={CHART_AXIS_TICK} axisLine={false} tickLine={false} domain={[0, 'auto']} />
                    <YAxis yAxisId="right" orientation="right" tick={CHART_AXIS_TICK} axisLine={false} tickLine={false} domain={[0, 'auto']} />
                    <Tooltip
                      contentStyle={RECHARTS_TOOLTIP_STYLE}
                      labelStyle={TOOLTIP_LABEL_STYLE}
                      itemStyle={TOOLTIP_ITEM_STYLE}
                      cursor={CHART_CURSOR_STYLE}
                      labelFormatter={(v) => `Partida ${v}`}
                      formatter={(value, key) =>
                        key === 'rating'
                          ? [formatDecimal(value, 2), 'Rating']
                          : [formatDecimal(value, 1), 'ADR']
                      }
                    />
                    <Line name="rating" yAxisId="left"  type="monotone" dataKey="rating" stroke="#8b5cf6" strokeWidth={2} dot={false} activeDot={{ r: 4, fill: '#8b5cf6' }} />
                    <Line name="adr"    yAxisId="right" type="monotone" dataKey="adr"    stroke="#f97316" strokeWidth={2} dot={false} activeDot={{ r: 4, fill: '#f97316' }} />
                  </LineChart>
                </ResponsiveContainer>
              ) : (
                <div className="p-empty p-empty--tall">
                  <TrendingUp className="p-empty-icon" size={28} />
                  <span>Juega partidas para ver tu progresión</span>
                </div>
              )}
            </div>
          </div>

          {/* Recent matches */}
          <div className="p-card">
            <p className="p-card-title">Últimas partidas</p>
            {matchHistory.length === 0 ? (
              <div className="p-empty">No hay partidas procesadas aún.</div>
            ) : (
              <div className="p-match-list">
                {matchHistory.slice(0, 6).map((match) => {
                  const win = match.result === 'W';
                  const ts = match.team_score ?? 0;
                  const os = match.opponent_score ?? 0;
                  return (
                    <div key={match.match_id} className="p-match-row">
                      <span className={`p-match-result-dot ${win ? 'win' : 'loss'}`} />
                      <span className="p-match-map">{formatMapName(match.map)}</span>
                      <span className="p-match-date">{formatRelativeDate(match.date)}</span>
                      <span className={`p-match-score ${win ? 'win' : 'loss'}`}>{ts}-{os}</span>
                      <div className="p-match-stats">
                        <span><span className="val">{formatDecimal(match.kd_ratio, 2)}</span> K/D</span>
                        <span><span className="val">{formatDecimal(match.adr, 1)}</span> ADR</span>
                        <span><span className="val">{formatDecimal(match.hltv_rating, 2)}</span> RT</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </SectionBlock>

      {/* */}
      <SectionBlock title="Rendimiento por bando">
        <div className="p-grid p-grid-2">
          {sideMetrics.map((row) => {
            const total = row.ct + row.t;
            const ctPct = total > 0 ? (row.ct / total) * 100 : 50;
            const tPct  = 100 - ctPct;
            return (
              <div key={row.label} className="p-card p-card--compact">
                <div className="p-side-block-label p-mb-6">{row.label}</div>
                <div className="p-split-bar p-mb-6">
                  <div className="p-split-bar-ct" style={{ width: `${ctPct}%` }} />
                  <div className="p-split-bar-t"  style={{ width: `${tPct}%` }} />
                </div>
                <div className="p-side-values">
                  <span className="p-side-val ct">CT&nbsp;&nbsp;{formatDecimal(row.ct, row.decimals)}</span>
                  <span className="p-side-val t">T&nbsp;&nbsp;{formatDecimal(row.t, row.decimals)}</span>
                </div>
              </div>
            );
          })}
        </div>
      </SectionBlock>

    </div>
  );
};

export default OverviewTab;
