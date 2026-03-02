import React, { useMemo, useState } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  LabelList,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import {
  RECHARTS_TOOLTIP_STYLE,
  TOOLTIP_LABEL_STYLE,
  TOOLTIP_ITEM_STYLE,
  CHART_AXIS_TICK,
  CHART_GRID_STROKE,
  CHART_CURSOR_BAR,
  formatDecimal,
  formatDegrees,
  formatInteger,
  formatMilliseconds,
  formatPercent,
  formatWeaponName,
} from '../../utils/performanceFormatters';
import { getQualityLabel } from '../../utils/performanceBenchmarks';
import { MetricCell } from './StatPill';
import SectionBlock from './SectionBlock';

/* ── Weapons sub-components ────────────────────────────────── */
const SORT_OPTIONS = [
  { key: 'kills',    label: 'Bajas' },
  { key: 'accuracy', label: 'Precisión' },
  { key: 'hs_pct',  label: 'HS %' },
  { key: 'damage',  label: 'Daño' },
];

const TOP_BADGE = ['🥇', '🥈', '🥉'];

const BarCell = ({ value, maxValue, children }) => {
  const pct = maxValue > 0 ? Math.min((value / maxValue) * 100, 100) : 0;
  return (
    <td className="p-bar-cell">
      <div className="p-bar-bg" style={{ width: `${pct}%` }} />
      <span className="p-bar-val">{children}</span>
    </td>
  );
};

/* ── Main component ─────────────────────────────────────────── */
const clutchKeys = ['1v1', '1v2', '1v3', '1v4', '1v5'];
const mkDefs = [
  { key: '2k',  label: '2K',  cls: 'mk-2k' },
  { key: '3k',  label: '3K',  cls: 'mk-3k' },
  { key: '4k',  label: '4K',  cls: 'mk-4k' },
  { key: 'ace', label: 'ACE', cls: 'mk-ace' },
];

const RendimientoTab = ({ aim = {}, combat = {}, weapons = [], overview = {} }) => {
  const [sortKey, setSortKey] = useState('kills');

  /* ── Aim data ──────────────────────────────────────────────── */
  const bodyParts = aim.body_part_hits || {};
  const totalHits =
    Number(bodyParts.head || 0)
    + Number(bodyParts.chest || 0)
    + Number(bodyParts.stomach || 0)
    + Number(bodyParts.left_arm || 0)
    + Number(bodyParts.right_arm || 0)
    + Number(bodyParts.left_leg || 0)
    + Number(bodyParts.right_leg || 0);

  const bodyChartData = [
    { label: 'Cabeza',  value: Number(bodyParts.head || 0),    color: '#f59e0b' },
    { label: 'Pecho',   value: Number(bodyParts.chest || 0),   color: '#3b82f6' },
    { label: 'Abdomen', value: Number(bodyParts.stomach || 0), color: '#10b981' },
    { label: 'Brazos',  value: Number(bodyParts.left_arm  || 0) + Number(bodyParts.right_arm || 0), color: '#f97316' },
    { label: 'Piernas', value: Number(bodyParts.left_leg  || 0) + Number(bodyParts.right_leg || 0), color: '#8b5cf6' },
  ].map((e) => ({ ...e, pct: totalHits > 0 ? (e.value / totalHits) * 100 : 0 }));

  /* ── Combat data ───────────────────────────────────────────── */
  const clutches   = combat.clutches   || {};
  const multikills = combat.multikills || {};
  const clutchVals = clutchKeys.map((k) => Number(clutches[k] || 0));
  const maxClutch  = Math.max(...clutchVals, 1);
  const mkVals     = mkDefs.map((m)   => Number(multikills[m.key] || 0));
  const maxMk      = Math.max(...mkVals, 1);

  /* ── Weapons data ──────────────────────────────────────────── */
  const sortedWeapons = useMemo(
    () => [...weapons].sort((a, b) => Number(b[sortKey] || 0) - Number(a[sortKey] || 0)),
    [weapons, sortKey],
  );
  const maxKills  = useMemo(() => Math.max(...weapons.map((w) => Number(w.kills  || 0)), 1), [weapons]);
  const maxDamage = useMemo(() => Math.max(...weapons.map((w) => Number(w.damage || 0)), 1), [weapons]);

  return (
    <div className="p-section">

      {/* ── Key metrics band ─────────────────────────────────── */}
      <div className="p-metric-row">
        <MetricCell
          value={formatPercent(aim.accuracy_overall)}
          label="Precisión"
          badge={getQualityLabel(Number(aim.accuracy_overall || 0), 'accuracy')}
        />
        <MetricCell
          value={formatMilliseconds(aim.time_to_damage_avg_ms)}
          label="Tiempo al daño"
          badge={getQualityLabel(Number(aim.time_to_damage_avg_ms || 0), 'ttd_ms', true)}
        />
        <MetricCell
          value={formatPercent(combat.opening_success_rate)}
          label="Éxito en aperturas"
          badge={getQualityLabel(Number(combat.opening_success_rate || 0), 'opening_success')}
        />
        <MetricCell
          value={formatDecimal(overview.impact_rating, 2)}
          label="Rating de impacto"
          badge={getQualityLabel(Number(overview.impact_rating || 0), 'impact_rating')}
        />
      </div>

      {/* ── Puntería ─────────────────────────────────────────── */}
      <SectionBlock title="Puntería">
        <div className="p-grid p-grid-wide">

          {/* Body-part chart */}
          <div className="p-card p-card--chart">
            <p className="p-card-title">Distribución de impactos</p>
            <div className="p-chart-wrap">
              <ResponsiveContainer width="100%" height={170}>
                <BarChart data={bodyChartData} layout="vertical" margin={{ left: 10, right: 32, top: 4, bottom: 0 }}>
                  <CartesianGrid stroke={CHART_GRID_STROKE} horizontal={false} vertical />
                  <XAxis type="number" tick={CHART_AXIS_TICK} axisLine={false} tickLine={false} />
                  <YAxis type="category" dataKey="label" tick={{ fill: '#94a3b8', fontSize: 12, fontWeight: 600 }} axisLine={false} tickLine={false} />
                  <Tooltip
                    contentStyle={RECHARTS_TOOLTIP_STYLE}
                    labelStyle={TOOLTIP_LABEL_STYLE}
                    itemStyle={TOOLTIP_ITEM_STYLE}
                    cursor={CHART_CURSOR_BAR}
                    formatter={(value, _, payload) => {
                      const pct = payload?.payload?.pct ?? 0;
                      return [`${formatInteger(value)} · ${pct.toFixed(1)}%`, 'Impactos'];
                    }}
                  />
                  <Bar dataKey="value" radius={[0, 6, 6, 0]}>
                    {bodyChartData.map((e) => <Cell key={e.label} fill={e.color} />)}
                    <LabelList
                      dataKey="pct"
                      position="right"
                      formatter={(v) => `${v.toFixed(1)}%`}
                      style={{ fill: 'var(--p-text-muted)', fontSize: 11, fontWeight: 700 }}
                    />
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>

          {/* Aim stats */}
          <div className="p-card">
            <p className="p-card-title">Métricas de puntería</p>
            <div className="p-stat-list">
              <div className="p-stat-row">
                <span className="p-stat-row-label">Error de crosshair</span>
                <span className="p-stat-row-value">{formatDegrees(aim.crosshair_placement_avg_error)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Error al peekear</span>
                <span className="p-stat-row-value">{formatDegrees(aim.crosshair_placement_peek)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Error al holdear</span>
                <span className="p-stat-row-value">{formatDegrees(aim.crosshair_placement_hold)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Disparos efectuados</span>
                <span className="p-stat-row-value">{formatInteger(aim.shots_fired)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Disparos impactados</span>
                <span className="p-stat-row-value">{formatInteger(aim.shots_hit)}</span>
              </div>
            </div>
          </div>
        </div>
      </SectionBlock>

      {/* ── Combate ──────────────────────────────────────────── */}
      <SectionBlock title="Combate">
        <div className="p-grid p-grid-2">

          {/* Opening duels */}
          <div className="p-card">
            <p className="p-card-title">Duelos de apertura</p>
            <div className="p-num-grid" style={{ gridTemplateColumns: 'repeat(3, 1fr)' }}>
              <div className="p-num-cell">
                <span className="p-num-cell-value">{formatInteger(combat.opening_duels_attempted)}</span>
                <span className="p-num-cell-label">Intentados</span>
              </div>
              <div className="p-num-cell">
                <span className="p-num-cell-value" style={{ color: 'var(--p-green-text)' }}>
                  {formatInteger(combat.opening_duels_won)}
                </span>
                <span className="p-num-cell-label">Ganados</span>
              </div>
              <div className="p-num-cell">
                <span className="p-num-cell-value" style={{ color: 'var(--p-red-text)' }}>
                  {formatInteger(combat.opening_duels_lost)}
                </span>
                <span className="p-num-cell-label">Perdidos</span>
              </div>
            </div>

            <div style={{ marginTop: 12 }}>
              <p className="p-card-title" style={{ marginBottom: 8 }}>Trades</p>
              <div className="p-num-grid" style={{ gridTemplateColumns: 'repeat(2, 1fr)' }}>
                <div className="p-num-cell">
                  <span className="p-num-cell-value">{formatInteger(combat.trade_kills)}</span>
                  <span className="p-num-cell-label">Bajas de trade</span>
                </div>
                <div className="p-num-cell">
                  <span className="p-num-cell-value">{formatInteger(combat.traded_deaths)}</span>
                  <span className="p-num-cell-label">Muertes tradeadas</span>
                </div>
              </div>
            </div>
          </div>

          {/* Clutches */}
          <div className="p-card">
            <p className="p-card-title">Clutches</p>
            <div className="p-clutch-grid">
              {clutchKeys.map((key, i) => {
                const val = clutchVals[i];
                const opacity = val > 0 ? (val / maxClutch) * 0.85 + 0.15 : 0;
                return (
                  <div
                    key={key}
                    className={`p-mk-item ${val === 0 ? 'zero' : ''}`}
                    style={{ '--mk-opacity': opacity }}
                  >
                    <span className="p-mk-item-label">{key}</span>
                    <span className="p-mk-item-value">{val}</span>
                  </div>
                );
              })}
            </div>

            <p className="p-card-title" style={{ marginTop: 14, marginBottom: 8 }}>Multi-kills</p>
            <div className="p-mk-grid">
              {mkDefs.map((mk, i) => {
                const val = mkVals[i];
                const opacity = val > 0 ? (val / maxMk) * 0.85 + 0.15 : 0;
                return (
                  <div
                    key={mk.key}
                    className={`p-mk-item ${mk.cls} ${val === 0 ? 'zero' : ''}`}
                    style={{ '--mk-opacity': opacity }}
                  >
                    <span className="p-mk-item-label">{mk.label}</span>
                    <span className="p-mk-item-value">{val}</span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </SectionBlock>

      {/* ── Armas ────────────────────────────────────────────── */}
      <SectionBlock title="Arsenal">
        <div className="p-card">
          <div className="p-table-toolbar">
            <span className="p-card-title" style={{ margin: 0 }}>Armas utilizadas</span>
            <select
              value={sortKey}
              onChange={(e) => setSortKey(e.target.value)}
              className="p-select"
            >
              {SORT_OPTIONS.map((opt) => (
                <option key={opt.key} value={opt.key}>Ordenar por {opt.label}</option>
              ))}
            </select>
          </div>

          <div className="p-table-wrap">
            <table className="p-table">
              <thead>
                <tr>
                  <th>Arma</th>
                  <th>Bajas</th>
                  <th>Precisión</th>
                  <th>HS %</th>
                  <th>Daño</th>
                </tr>
              </thead>
              <tbody>
                {sortedWeapons.length === 0 && (
                  <tr><td colSpan={5} className="p-empty-cell">Sin estadísticas de armas todavía.</td></tr>
                )}
                {sortedWeapons.map((weapon, index) => (
                  <tr key={`${weapon.weapon}-${index}`}>
                    <td>
                      <div className="p-weapon-name">
                        {index < 3 && <span className="p-weapon-rank">{TOP_BADGE[index]}</span>}
                        {formatWeaponName(weapon.weapon)}
                      </div>
                    </td>
                    <BarCell value={Number(weapon.kills || 0)} maxValue={maxKills}>
                      {formatInteger(weapon.kills)}
                    </BarCell>
                    <td>{formatDecimal(weapon.accuracy, 1, '%')}</td>
                    <td>{formatDecimal(weapon.hs_pct, 1, '%')}</td>
                    <BarCell value={Number(weapon.damage || 0)} maxValue={maxDamage}>
                      {formatInteger(weapon.damage)}
                    </BarCell>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </SectionBlock>

    </div>
  );
};

export default RendimientoTab;
