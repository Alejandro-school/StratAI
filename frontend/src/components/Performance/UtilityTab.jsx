import React from 'react';
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts';
import {
  RECHARTS_TOOLTIP_STYLE,
  TOOLTIP_LABEL_STYLE,
  TOOLTIP_ITEM_STYLE,
  formatDecimal,
  formatInteger,
} from '../../utils/performanceFormatters';
import { getQualityLabel } from '../../utils/performanceBenchmarks';
import { MetricCell } from './StatPill';
import SectionBlock from './SectionBlock';

const UtilityTab = ({ utility = {}, combat = {}, economy = {} }) => {
  const grenadeDamage = utility.grenade_damage || {};
  const pieData = [
    { name: 'HE',      value: Number(grenadeDamage.he      || 0), color: '#f97316' },
    { name: 'Molotov', value: Number(grenadeDamage.molotov || 0), color: '#ef4444' },
  ].filter((d) => d.value > 0);

  const totalDmg = pieData.reduce((s, d) => s + d.value, 0);

  return (
    <div className="p-section">

      {/* */}
      <div className="p-metric-row">
        <MetricCell
          value={formatDecimal(utility.enemies_flashed_per_flash, 2)}
          label="Enemigos por flash"
          badge={getQualityLabel(Number(utility.enemies_flashed_per_flash || 0), 'enemies_per_flash')}
        />
        <MetricCell
          value={formatDecimal(utility.he_damage_per_nade, 1)}
          label="Daño por HE"
          badge={getQualityLabel(Number(utility.he_damage_per_nade || 0), 'he_damage_per_nade')}
        />
        <MetricCell
          value={formatDecimal(utility.molotov_damage_per_nade, 1)}
          label="Daño por molotov"
          badge={getQualityLabel(Number(utility.molotov_damage_per_nade || 0), 'molotov_damage_per_nade')}
        />
        <MetricCell
          value={formatInteger(utility.smokes_thrown)}
          label="Smokes lanzadas"
        />
      </div>

      {/* */}
      <SectionBlock title="Análisis de granadas">
        <div className="p-grid p-grid-wide">

          {/* Flash detail */}
          <div className="p-card">
            <p className="p-card-title">Flashes</p>
            <div className="p-stat-list">
              <div className="p-stat-row">
                <span className="p-stat-row-label">Flashes lanzadas</span>
                <span className="p-stat-row-value">{formatInteger(utility.flashes_thrown)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Enemigos cegados</span>
                <span className="p-stat-row-value">{formatInteger(utility.enemies_flashed_total)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Tiempo ciego promedio</span>
                <span className="p-stat-row-value">{formatDecimal(utility.blind_time_per_flash, 2, 's')}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Flash assists</span>
                <span className="p-stat-row-value">{formatInteger(combat.flash_assists)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">HE lanzadas</span>
                <span className="p-stat-row-value">{formatInteger(utility.he_thrown)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Molotovs lanzadas</span>
                <span className="p-stat-row-value">{formatInteger(utility.molotovs_thrown)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Total granadas</span>
                <span className="p-stat-row-value">{formatInteger(utility.grenades_thrown_total)}</span>
              </div>
            </div>
          </div>

          {/* Damage donut */}
          <div className="p-card p-card--chart">
            <p className="p-card-title">Distribución de daño</p>
            {pieData.length === 0 ? (
              <div className="p-empty p-empty--chart">
                <span>Sin daño de granadas registrado</span>
              </div>
            ) : (
              <div className="p-chart-wrap p-chart-wrap--fixed">
                <ResponsiveContainer width="100%" height={180}>
                  <PieChart>
                    <Pie data={pieData} dataKey="value" nameKey="name" innerRadius={50} outerRadius={74} paddingAngle={3}>
                      {pieData.map((entry) => (
                        <Cell key={entry.name} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={RECHARTS_TOOLTIP_STYLE}
                      labelStyle={TOOLTIP_LABEL_STYLE}
                      itemStyle={TOOLTIP_ITEM_STYLE}
                      cursor={false}
                      formatter={(value) => [formatInteger(value), 'Daño total']}
                    />
                  </PieChart>
                </ResponsiveContainer>
                <div className="p-chart-center">
                  <span className="p-chart-center-value">{formatInteger(totalDmg)}</span>
                  <span className="p-chart-center-label">Daño total</span>
                </div>

                {/* Legend */}
                <div className="p-chart-legend">
                  {pieData.map((d) => (
                    <div key={d.name} className="p-chart-legend-item">
                      <span className="p-chart-legend-dot" style={{ background: d.color }} />
                      {d.name}&nbsp;{formatInteger(d.value)}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </SectionBlock>

      {/* */}
      {economy.total_rounds > 0 && (
        <SectionBlock title="Supervivencia">
          <div className="p-card">
            <div className="p-grid p-grid-2">
              <div className="p-num-cell">
                <span className="p-num-cell-value">{formatDecimal(economy.survival_rate, 1)}%</span>
                <span className="p-num-cell-label">Tasa de supervivencia</span>
              </div>
              <div className="p-num-cell">
                <span className="p-num-cell-value">{formatInteger(economy.rounds_survived)} / {formatInteger(economy.total_rounds)}</span>
                <span className="p-num-cell-label">Rondas sobrevividas</span>
              </div>
            </div>
          </div>
        </SectionBlock>
      )}

    </div>
  );
};

export default UtilityTab;
