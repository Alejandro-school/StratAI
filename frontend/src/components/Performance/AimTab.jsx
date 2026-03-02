import React from 'react';
import { Bar, BarChart, CartesianGrid, Cell, LabelList, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { Crosshair, Gauge, Timer } from 'lucide-react';
import StatPill from './StatPill';
import { getQualityLabel, PERFORMANCE_BENCHMARKS } from '../../utils/performanceBenchmarks';
import {
  RECHARTS_TOOLTIP_STYLE,
  TOOLTIP_LABEL_STYLE,
  TOOLTIP_ITEM_STYLE,
  CHART_AXIS_TICK,
  CHART_GRID_STROKE,
  CHART_CURSOR_BAR,
  formatDegrees,
  formatInteger,
  formatMilliseconds,
  formatPercent,
} from '../../utils/performanceFormatters';

const AimTab = ({ aim = {} }) => {
  const bodyParts = aim.body_part_hits || {};
  const totalHits =
    Number(bodyParts.head || 0)
    + Number(bodyParts.chest || 0)
    + Number(bodyParts.stomach || 0)
    + Number(bodyParts.left_arm || 0)
    + Number(bodyParts.right_arm || 0)
    + Number(bodyParts.left_leg || 0)
    + Number(bodyParts.right_leg || 0);

  const chartData = [
    { label: 'Cabeza', value: Number(bodyParts.head || 0), color: '#f59e0b' },
    { label: 'Pecho', value: Number(bodyParts.chest || 0), color: '#3b82f6' },
    { label: 'Abdomen', value: Number(bodyParts.stomach || 0), color: '#10b981' },
    { label: 'Brazos', value: Number(bodyParts.left_arm || 0) + Number(bodyParts.right_arm || 0), color: '#f97316' },
    { label: 'Piernas', value: Number(bodyParts.left_leg || 0) + Number(bodyParts.right_leg || 0), color: '#8b5cf6' },
  ].map((entry) => ({
    ...entry,
    pct: totalHits > 0 ? (entry.value / totalHits) * 100 : 0,
  }));

  const formatBodyPartLabel = (value) => {
    if (!Number.isFinite(value)) return '0%';
    return `${value.toFixed(1)}%`;
  };

  return (
    <div className="perf-tab-layout">
      <div className="perf-grid perf-grid-3">
        <StatPill
          label="Precisión"
          value={formatPercent(aim.accuracy_overall)}
          icon={Gauge}
          tone="blue"
          badge={getQualityLabel(Number(aim.accuracy_overall || 0), 'accuracy')}
        />
        <StatPill
          label="Tiempo al daño"
          value={formatMilliseconds(aim.time_to_damage_avg_ms)}
          icon={Timer}
          tone="orange"
          badge={getQualityLabel(Number(aim.time_to_damage_avg_ms || 0), 'ttd_ms', true)}
        />
        <StatPill
          label="Error de crosshair"
          value={formatDegrees(aim.crosshair_placement_avg_error)}
          icon={Crosshair}
          tone="purple"
          badge={getQualityLabel(Number(aim.crosshair_placement_avg_error || 0), 'crosshair_error', true)}
        />
      </div>

      <div className="perf-grid perf-grid-2">
        <div className="perf-card">
          <h3 className="perf-card-title">Crosshair por contexto</h3>
          <div className="split-stats">
            <div className="split-stat-block">
              <span>Error al peekeo</span>
              <strong>{formatDegrees(aim.crosshair_placement_peek)}</strong>
            </div>
            <div className="split-stat-block">
              <span>Error al holdear</span>
              <strong>{formatDegrees(aim.crosshair_placement_hold)}</strong>
            </div>
          </div>
        </div>
        <div className="perf-card">
          <h3 className="perf-card-title">Volumen de disparos</h3>
          <div className="split-stats">
            <div className="split-stat-block">
              <span>Disparos efectuados</span>
              <strong>{formatInteger(aim.shots_fired)}</strong>
            </div>
            <div className="split-stat-block">
              <span>Disparos impactados</span>
              <strong>{formatInteger(aim.shots_hit)}</strong>
            </div>
          </div>
        </div>
      </div>

      <div className="perf-card">
        <h3 className="perf-card-title">Distribución de impactos por zona corporal</h3>
        <div className="chart-wrap">
          <ResponsiveContainer width="100%" height={290}>
            <BarChart data={chartData} layout="vertical" margin={{ left: 10, right: 20 }}>
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
                  return [`${formatInteger(value)} · ${formatBodyPartLabel(pct)}`, 'Impactos'];
                }}
              />
              <Bar dataKey="value" radius={[0, 6, 6, 0]}>
                {chartData.map((entry) => (
                  <Cell key={entry.label} fill={entry.color} />
                ))}
                <LabelList
                  dataKey="pct"
                  position="right"
                  formatter={formatBodyPartLabel}
                  style={{ fill: 'var(--perf-muted)', fontSize: 11, fontWeight: 700 }}
                />
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* HS% benchmark bar */}
      {Number(aim.accuracy_overall || 0) > 0 && (() => {
        const hsPct = Number(aim.body_part_hits?.head || 0);
        const totalH = totalHits;
        const hsRate = totalH > 0 ? (hsPct / totalH) * 100 : 0;
        const benchAvg = PERFORMANCE_BENCHMARKS.hs_pct?.average || 45;
        const clampedPct = Math.min(hsRate, 100);
        const markerPos = Math.min(benchAvg, 100);

        return (
          <div className="perf-card">
            <h3 className="perf-card-title">% Headshot vs Promedio</h3>
            <div className="benchmark-bar-wrap">
              <div className="benchmark-bar-labels">
                <span>Tuyo: <strong>{clampedPct.toFixed(1)}%</strong></span>
                <span>Promedio: {benchAvg}%</span>
              </div>
              <div className="benchmark-bar-track">
                <div
                  className="benchmark-bar-fill"
                  style={{
                    width: `${clampedPct}%`,
                    background: clampedPct >= benchAvg
                      ? 'linear-gradient(90deg, #8b5cf6, #22c55e)'
                      : 'linear-gradient(90deg, #8b5cf6, #f59e0b)',
                  }}
                />
                <div
                  className="benchmark-bar-marker"
                  style={{ left: `${markerPos}%` }}
                  title={`Promedio: ${benchAvg}%`}
                />
              </div>
            </div>
          </div>
        );
      })()}
    </div>
  );
};

export default AimTab;
