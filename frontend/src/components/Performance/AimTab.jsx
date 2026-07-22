import React, { useMemo } from 'react';
import {
  PolarAngleAxis,
  PolarGrid,
  Radar,
  RadarChart,
  ResponsiveContainer,
  Tooltip,
} from 'recharts';
import { Crosshair, Gauge, Target } from 'lucide-react';
import { PERFORMANCE_BENCHMARKS, getQualityLabel } from '../../utils/performanceBenchmarks';
import {
  RECHARTS_TOOLTIP_STYLE,
  TOOLTIP_ITEM_STYLE,
  TOOLTIP_LABEL_STYLE,
  formatDecimal,
  formatDegrees,
  formatInteger,
  formatMilliseconds,
  formatPercent,
} from '../../utils/performanceFormatters';
import { MetricCell } from './StatPill';
import SectionBlock from './SectionBlock';
import { BriefingPanel, InsightCard } from './PerformanceBriefing';

const clamp = (value, min = 0, max = 100) => Math.max(min, Math.min(max, value));

const normalizeMetric = (value, metric, reverse = false) => {
  const numeric = Number(value || 0);
  const benchmark = PERFORMANCE_BENCHMARKS[metric];

  if (!benchmark) return 0;

  if (!reverse) {
    const range = benchmark.good - benchmark.poor || 1;
    return clamp(((numeric - benchmark.poor) / range) * 100);
  }

  const range = benchmark.poor - benchmark.good || 1;
  return clamp(((benchmark.poor - numeric) / range) * 100);
};

const zonePercent = (value, total) => (total > 0 ? (value / total) * 100 : 0);

const AimTab = ({ aim = {}, combat = {}, overview = {} }) => {
  const bodyParts = aim.body_part_hits || {};
  const totalHits =
    Number(bodyParts.head || 0)
    + Number(bodyParts.chest || 0)
    + Number(bodyParts.stomach || 0)
    + Number(bodyParts.left_arm || 0)
    + Number(bodyParts.right_arm || 0)
    + Number(bodyParts.left_leg || 0)
    + Number(bodyParts.right_leg || 0);

  const hsRate = totalHits > 0 ? (Number(bodyParts.head || 0) / totalHits) * 100 : 0;
  const shotsFired = Number(aim.shots_fired || 0);
  const shotsHit = Number(aim.shots_hit || 0);
  const kills = Number(overview.kills || 0);
  const shotsPerKill = kills > 0 ? shotsFired / kills : 0;
  const damagePerShot = shotsFired > 0 ? Number(overview.total_damage || 0) / shotsFired : 0;
  const missRate = shotsFired > 0 ? ((shotsFired - shotsHit) / shotsFired) * 100 : 0;

  const radarData = useMemo(
    () => [
      { metric: 'Precisión', value: normalizeMetric(aim.accuracy_overall, 'accuracy') },
      { metric: 'Mira', value: normalizeMetric(aim.crosshair_placement_avg_error, 'crosshair_error', true) },
      { metric: 'TTD', value: normalizeMetric(aim.time_to_damage_avg_ms, 'ttd_ms', true) },
      { metric: 'Cabeza %', value: normalizeMetric(hsRate, 'hs_pct') },
      { metric: 'Aperturas', value: normalizeMetric(combat.opening_success_rate, 'opening_success') },
    ],
    [aim, combat.opening_success_rate, hsRate],
  );

  const zoneData = [
    {
      label: 'Cabeza',
      color: '#f59e0b',
      count: Number(bodyParts.head || 0),
      pct: zonePercent(Number(bodyParts.head || 0), totalHits),
    },
    {
      label: 'Pecho',
      color: '#3b82f6',
      count: Number(bodyParts.chest || 0),
      pct: zonePercent(Number(bodyParts.chest || 0), totalHits),
    },
    {
      label: 'Estómago',
      color: '#10b981',
      count: Number(bodyParts.stomach || 0),
      pct: zonePercent(Number(bodyParts.stomach || 0), totalHits),
    },
    {
      label: 'Brazos',
      color: '#f97316',
      count: Number(bodyParts.left_arm || 0) + Number(bodyParts.right_arm || 0),
      pct: zonePercent(Number(bodyParts.left_arm || 0) + Number(bodyParts.right_arm || 0), totalHits),
    },
    {
      label: 'Piernas',
      color: '#6366f1',
      count: Number(bodyParts.left_leg || 0) + Number(bodyParts.right_leg || 0),
      pct: zonePercent(Number(bodyParts.left_leg || 0) + Number(bodyParts.right_leg || 0), totalHits),
    },
  ];

  return (
    <div className="p-section">
      <BriefingPanel eyebrow="Briefing mecanico" title="Punteria aplicada y velocidad de dano">
        <div className="p-briefing-grid-3">
          <InsightCard
            tone={Number(aim.accuracy_overall || 0) >= 25 ? 'good' : 'warning'}
            icon={Target}
            label="Precision aplicada"
            value={formatPercent(aim.accuracy_overall)}
            detail={`${formatInteger(shotsHit)} impactos de ${formatInteger(shotsFired)} disparos`}
          />
          <InsightCard
            tone={hsRate >= 40 ? 'good' : 'neutral'}
            icon={Crosshair}
            label="Headshot rate"
            value={formatPercent(hsRate)}
            detail={`${formatInteger(bodyParts.head)} impactos a cabeza`}
          />
          <InsightCard
            tone={Number(aim.time_to_damage_avg_ms || 0) <= 550 ? 'good' : 'warning'}
            icon={Gauge}
            label="Tiempo al dano"
            value={formatMilliseconds(aim.time_to_damage_avg_ms)}
            detail={`${formatDegrees(aim.crosshair_placement_avg_error)} de error medio`}
          />
        </div>
      </BriefingPanel>

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
          value={formatPercent(hsRate)}
          label="Tasa de headshots"
          badge={getQualityLabel(Number(hsRate || 0), 'hs_pct')}
        />
        <MetricCell
          value={formatDegrees(aim.crosshair_placement_avg_error)}
          label="Error de mira"
          badge={getQualityLabel(Number(aim.crosshair_placement_avg_error || 0), 'crosshair_error', true)}
        />
        <MetricCell
          value={formatDecimal(shotsPerKill, 1)}
          label="Disparos / baja"
          sub={`${formatDecimal(damagePerShot, 1)} daño / disparo`}
        />
      </div>

      <SectionBlock title="Perfil de puntería">
        <div className="p-grid p-grid-dashboard">
          <div className="p-card p-card--chart">
            <p className="p-card-title">Radar mecánico</p>
            <p className="p-card-subtitle">Normalizado contra referencias para ver rápido dónde está tu brecha real</p>
            <div className="p-radar-wrap">
              <ResponsiveContainer width="100%" height={300}>
                <RadarChart data={radarData} outerRadius={110}>
                  <PolarGrid stroke="rgba(148, 163, 184, 0.14)" />
                  <PolarAngleAxis dataKey="metric" tick={{ fill: '#94a3b8', fontSize: 12, fontWeight: 700 }} />
                  <Tooltip
                    contentStyle={RECHARTS_TOOLTIP_STYLE}
                    labelStyle={TOOLTIP_LABEL_STYLE}
                    itemStyle={TOOLTIP_ITEM_STYLE}
                      formatter={(value) => [`${Number(value).toFixed(0)}/100`, 'Puntuación']}
                  />
                  <Radar dataKey="value" stroke="#6366f1" fill="#6366f1" fillOpacity={0.28} strokeWidth={2.5} />
                </RadarChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="p-card">
            <p className="p-card-title">Silueta de impactos</p>
            <div className="p-silhouette-layout">
              <svg viewBox="0 0 220 320" className="p-aim-silhouette" aria-hidden="true">
                <circle cx="110" cy="40" r="26" fill="rgba(245, 158, 11, 0.16)" stroke="#f59e0b" strokeWidth="2" style={{ opacity: 0.25 + zoneData[0].pct / 100 }} />
                <rect x="82" y="78" width="56" height="74" rx="18" fill="rgba(59, 130, 246, 0.16)" stroke="#3b82f6" strokeWidth="2" style={{ opacity: 0.25 + zoneData[1].pct / 100 }} />
                <rect x="92" y="154" width="36" height="42" rx="14" fill="rgba(16, 185, 129, 0.16)" stroke="#10b981" strokeWidth="2" style={{ opacity: 0.25 + zoneData[2].pct / 100 }} />
                <rect x="44" y="86" width="30" height="108" rx="14" fill="rgba(249, 115, 22, 0.16)" stroke="#f97316" strokeWidth="2" style={{ opacity: 0.25 + zoneData[3].pct / 100 }} />
                <rect x="146" y="86" width="30" height="108" rx="14" fill="rgba(249, 115, 22, 0.16)" stroke="#f97316" strokeWidth="2" style={{ opacity: 0.25 + zoneData[3].pct / 100 }} />
                <rect x="88" y="202" width="22" height="96" rx="12" fill="rgba(99, 102, 241, 0.16)" stroke="#6366f1" strokeWidth="2" style={{ opacity: 0.25 + zoneData[4].pct / 100 }} />
                <rect x="112" y="202" width="22" height="96" rx="12" fill="rgba(99, 102, 241, 0.16)" stroke="#6366f1" strokeWidth="2" style={{ opacity: 0.25 + zoneData[4].pct / 100 }} />
                <text x="110" y="45" textAnchor="middle" className="p-silhouette-text">{zoneData[0].pct.toFixed(1)}%</text>
                <text x="110" y="118" textAnchor="middle" className="p-silhouette-text">{zoneData[1].pct.toFixed(1)}%</text>
                <text x="110" y="180" textAnchor="middle" className="p-silhouette-text">{zoneData[2].pct.toFixed(1)}%</text>
                <text x="58" y="142" textAnchor="middle" className="p-silhouette-text">{zoneData[3].pct.toFixed(1)}%</text>
                <text x="162" y="142" textAnchor="middle" className="p-silhouette-text">{zoneData[3].pct.toFixed(1)}%</text>
                <text x="110" y="256" textAnchor="middle" className="p-silhouette-text">{zoneData[4].pct.toFixed(1)}%</text>
              </svg>

              <div className="p-aim-legend">
                {zoneData.map((zone) => (
                  <div key={zone.label} className="p-aim-legend-item">
                    <span className="p-aim-legend-dot" style={{ background: zone.color }} />
                    <span>{zone.label}</span>
                    <strong>{formatInteger(zone.count)}</strong>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </SectionBlock>

      <SectionBlock title="Lectura mecanica">
        <div className="p-grid p-grid-2">
          <div className="p-card">
            <p className="p-card-title">Mira y contexto</p>
            <div className="p-benchmark-list">
              <div className="p-benchmark-item">
                <div className="p-progress-head">
                  <span className="p-progress-head-label">Error general</span>
                  <span className="p-progress-head-value">{formatDegrees(aim.crosshair_placement_avg_error)}</span>
                </div>
                <div className="p-progress-track">
                  <div className="p-progress-fill p-progress-fill--purple" style={{ width: `${normalizeMetric(aim.crosshair_placement_avg_error, 'crosshair_error', true)}%` }} />
                </div>
              </div>
              <div className="p-benchmark-item">
                <div className="p-progress-head">
                  <span className="p-progress-head-label">Error al peekear</span>
                  <span className="p-progress-head-value">{formatDegrees(aim.crosshair_placement_peek)}</span>
                </div>
                <div className="p-progress-track">
                  <div className="p-progress-fill p-progress-fill--orange" style={{ width: `${normalizeMetric(aim.crosshair_placement_peek, 'crosshair_error', true)}%` }} />
                </div>
              </div>
              <div className="p-benchmark-item">
                <div className="p-progress-head">
                  <span className="p-progress-head-label">Error al holdear</span>
                  <span className="p-progress-head-value">{formatDegrees(aim.crosshair_placement_hold)}</span>
                </div>
                <div className="p-progress-track">
                  <div className="p-progress-fill p-progress-fill--blue" style={{ width: `${normalizeMetric(aim.crosshair_placement_hold, 'crosshair_error', true)}%` }} />
                </div>
              </div>
            </div>
          </div>

          <div className="p-card">
            <p className="p-card-title">Volumen de disparo</p>
            <div className="p-num-grid p-grid-2">
              <div className="p-num-cell">
                <span className="p-num-cell-value">{formatInteger(aim.shots_fired)}</span>
                <span className="p-num-cell-label">Disparos efectuados</span>
              </div>
              <div className="p-num-cell">
                <span className="p-num-cell-value">{formatInteger(aim.shots_hit)}</span>
                <span className="p-num-cell-label">Disparos impactados</span>
              </div>
            </div>
            <div className="p-progress-wrap p-mt-14">
              <div className="p-progress-head">
                <span className="p-progress-head-label">Precision aplicada</span>
                <span className="p-progress-head-value">{formatPercent(aim.accuracy_overall)}</span>
              </div>
              <div className="p-progress-track">
                <div className="p-progress-fill p-progress-fill--green" style={{ width: `${clamp(Number(aim.accuracy_overall || 0), 0, 100)}%` }} />
              </div>
            </div>
          </div>
        </div>
      </SectionBlock>

      <SectionBlock title="Eficiencia de disparo">
        <div className="p-grid p-grid-3">
          <div className="p-card">
            <p className="p-card-title">Conversión de disparos</p>
            <div className="p-stat-list">
              <div className="p-stat-row">
                <span className="p-stat-row-label">Disparos por baja</span>
                <span className="p-stat-row-value">{formatDecimal(shotsPerKill, 1)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Daño por disparo</span>
                <span className="p-stat-row-value">{formatDecimal(damagePerShot, 1)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Tasa de fallo</span>
                <span className="p-stat-row-value p-bad-text">{formatPercent(missRate)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Headshots totales</span>
                <span className="p-stat-row-value">{formatInteger(bodyParts.head)}</span>
              </div>
            </div>
          </div>
          <div className="p-card">
            <p className="p-card-title">Distribución de impactos</p>
            <div className="p-stat-list">
              {zoneData.map((zone) => (
                <div key={zone.label} className="p-stat-row">
                  <span className="p-stat-row-label" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span className="p-aim-legend-dot" style={{ background: zone.color, width: 8, height: 8, borderRadius: 2, flexShrink: 0 }} />
                    {zone.label}
                  </span>
                  <span className="p-stat-row-value">{formatInteger(zone.count)} ({zone.pct.toFixed(1)}%)</span>
                </div>
              ))}
            </div>
          </div>
          <div className="p-card">
            <p className="p-card-title">Lectura rápida</p>
            <p className="p-insight-copy">
              {hsRate >= 50
                ? 'Tu porcentaje de disparos a la cabeza es de élite. El enfoque a cabeza está generando ventaja táctica real en cada duelo.'
                : hsRate >= 40
                  ? 'Buen porcentaje de disparos a la cabeza. Hay margen para mejorar la colocación de mira y ganar consistencia.'
                  : hsRate >= 25
                    ? 'Tu porcentaje de disparos a la cabeza está en el rango medio. Trabajar el pre-aim y la altura de la mira mejorará tus duelos.'
                    : 'El porcentaje de disparos a la cabeza necesita atención. Practica colocación de mira en aim maps y pre-aim en deathmatch.'}
            </p>
            <p className="p-insight-copy p-mt-14">
              {Number(aim.accuracy_overall || 0) >= 30
                ? 'Precisión por encima de la media: tus disparos conectan con frecuencia.'
                : 'Precisión en rango mejorable: evita sprays innecesarios y prioriza taps controlados.'}
            </p>
          </div>
        </div>
      </SectionBlock>
    </div>
  );
};

export default AimTab;
