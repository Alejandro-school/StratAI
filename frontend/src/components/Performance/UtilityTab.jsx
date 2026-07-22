import React from 'react';
import { Flame, Shield, Zap } from 'lucide-react';
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
import { BriefingPanel, InsightCard } from './PerformanceBriefing';

const UtilityTab = ({ utility = {}, combat = {}, economy = {} }) => {
  const grenadeDamage = utility.grenade_damage || {};
  const pieData = [
    { name: 'HE',      value: Number(grenadeDamage.he      || 0), color: '#f97316' },
    { name: 'Molotov', value: Number(grenadeDamage.molotov || 0), color: '#ef4444' },
  ].filter((d) => d.value > 0);

  const totalDmg = pieData.reduce((s, d) => s + d.value, 0);
  const totalRounds = Number(economy.total_rounds || 0);
  const grenadesPerRound = totalRounds > 0
    ? Number(utility.grenades_thrown_total || 0) / totalRounds
    : 0;
  const utilDmgPerRound = totalRounds > 0
    ? Number(utility.utility_damage || 0) / totalRounds
    : 0;
  const flashesPerRound = totalRounds > 0
    ? Number(utility.flashes_thrown || 0) / totalRounds
    : 0;
  const smokesPerRound = totalRounds > 0
    ? Number(utility.smokes_thrown || 0) / totalRounds
    : 0;

  return (
    <div className="p-section">
      <BriefingPanel eyebrow="Briefing de utilidad" title="Control de mapa, flashes y dano auxiliar">
        <div className="p-briefing-grid-3">
          <InsightCard
            tone={Number(utility.enemies_flashed_per_flash || 0) >= 1 ? 'good' : 'warning'}
            icon={Zap}
            label="Flash impact"
            value={formatDecimal(utility.enemies_flashed_per_flash, 2)}
            detail={`${formatInteger(utility.enemies_flashed_total)} enemigos cegados`}
          />
          <InsightCard
            tone={Number(utility.he_damage_per_nade || 0) >= 20 ? 'good' : 'neutral'}
            icon={Flame}
            label="Dano HE"
            value={formatDecimal(utility.he_damage_per_nade, 1)}
            detail={`${formatInteger(grenadeDamage.he)} dano total`}
          />
          <InsightCard
            tone={Number(economy.survival_rate || 0) >= 45 ? 'good' : 'warning'}
            icon={Shield}
            label="Supervivencia"
            value={formatDecimal(economy.survival_rate, 1, '%')}
            detail={`${formatInteger(economy.rounds_survived)} de ${formatInteger(economy.total_rounds)} rondas`}
          />
        </div>
      </BriefingPanel>

      <div className="p-metric-row">
        <MetricCell
          value={formatDecimal(utility.enemies_flashed_per_flash, 2)}
          label="Enemigos por cegadora"
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
          label="Humos lanzados"
        />
        <MetricCell
          value={formatDecimal(utilDmgPerRound, 1)}
          label="Daño de utilidad / ronda"
          sub={`${formatInteger(utility.utility_damage)} daño total`}
        />
      </div>

      <SectionBlock title="Impacto por tipo de granada">
        <div className="p-grid p-grid-4">
          <div className="p-card p-utility-card">
            <span className="p-utility-card-kicker">Cegadora</span>
            <strong className="p-utility-card-value">{formatInteger(utility.flashes_thrown)}</strong>
            <span className="p-utility-card-sub">{formatDecimal(utility.enemies_flashed_per_flash, 2)} enemigos / cegadora</span>
            <span className={`p-badge ${getQualityLabel(Number(utility.enemies_flashed_per_flash || 0), 'enemies_per_flash').tone}`}>
              {getQualityLabel(Number(utility.enemies_flashed_per_flash || 0), 'enemies_per_flash').label}
            </span>
          </div>
          <div className="p-card p-utility-card">
            <span className="p-utility-card-kicker">HE</span>
            <strong className="p-utility-card-value">{formatInteger(utility.he_thrown)}</strong>
            <span className="p-utility-card-sub">{formatDecimal(utility.he_damage_per_nade, 1)} daño / granada</span>
            <span className={`p-badge ${getQualityLabel(Number(utility.he_damage_per_nade || 0), 'he_damage_per_nade').tone}`}>
              {getQualityLabel(Number(utility.he_damage_per_nade || 0), 'he_damage_per_nade').label}
            </span>
          </div>
          <div className="p-card p-utility-card">
            <span className="p-utility-card-kicker">Molotov</span>
            <strong className="p-utility-card-value">{formatInteger(utility.molotovs_thrown)}</strong>
            <span className="p-utility-card-sub">{formatDecimal(utility.molotov_damage_per_nade, 1)} daño / molotov</span>
            <span className={`p-badge ${getQualityLabel(Number(utility.molotov_damage_per_nade || 0), 'molotov_damage_per_nade').tone}`}>
              {getQualityLabel(Number(utility.molotov_damage_per_nade || 0), 'molotov_damage_per_nade').label}
            </span>
          </div>
          <div className="p-card p-utility-card">
            <span className="p-utility-card-kicker">Humo</span>
            <strong className="p-utility-card-value">{formatInteger(utility.smokes_thrown)}</strong>
            <span className="p-utility-card-sub">{formatDecimal(grenadesPerRound, 2)} granadas / ronda</span>
            <span className="p-badge neutral">Volumen</span>
          </div>
        </div>
      </SectionBlock>

      <SectionBlock title="Eficiencia de utilidad">
        <div className="p-grid p-grid-dashboard">
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

          <div className="p-stack-col">
            <div className="p-card">
              <p className="p-card-title">Cegadoras</p>
              <div className="p-stat-list">
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Enemigos cegados</span>
                  <span className="p-stat-row-value">{formatInteger(utility.enemies_flashed_total)}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Tiempo ciego promedio</span>
                  <span className="p-stat-row-value">{formatDecimal(utility.blind_time_per_flash, 2, 's')}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Asistencias con cegadora</span>
                  <span className="p-stat-row-value">{formatInteger(combat.flash_assists)}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Cegadoras por ronda</span>
                  <span className="p-stat-row-value">{formatDecimal(flashesPerRound, 2)}</span>
                </div>
              </div>
            </div>

            <div className="p-card">
              <p className="p-card-title">Frecuencia de uso</p>
              <div className="p-stat-list">
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Granadas totales</span>
                  <span className="p-stat-row-value">{formatInteger(utility.grenades_thrown_total)}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Granadas por ronda</span>
                  <span className="p-stat-row-value">{formatDecimal(grenadesPerRound, 2)}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Humos por ronda</span>
                  <span className="p-stat-row-value">{formatDecimal(smokesPerRound, 2)}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Daño de utilidad</span>
                  <span className="p-stat-row-value">{formatInteger(utility.utility_damage)}</span>
                </div>
                {economy.total_rounds > 0 && (
                  <div className="p-stat-row">
                    <span className="p-stat-row-label">Tasa de supervivencia</span>
                    <span className="p-stat-row-value">{formatDecimal(economy.survival_rate, 1)}%</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </SectionBlock>

      {economy.total_rounds > 0 && (
        <SectionBlock title="Supervivencia y lectura">
          <div className="p-grid p-grid-2">
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
            <div className="p-card">
              <p className="p-card-title">Lectura rápida</p>
              <p className="p-insight-copy">
                {Number(utility.enemies_flashed_per_flash || 0) >= 1.2
                  ? 'Tus cegadoras están generando impacto real. El promedio de enemigos cegados muestra buen timing y colocación.'
                  : Number(utility.enemies_flashed_per_flash || 0) >= 0.7
                    ? 'Uso de cegadoras correcto, pero hay espacio para mejorar posicionamiento y timing de lanzamiento.'
                    : 'El impacto de tus cegadoras es bajo. Revisa lineups y timing: muchas no están conectando.'}
              </p>
              <p className="p-insight-copy p-mt-14">
                {grenadesPerRound >= 2.5
                  ? 'Alto volumen de utilidad por ronda: estás invirtiendo bien tu dinero en granadas.'
                  : 'Bajo uso de granadas por ronda. Comprar y usar más utilidad mejorará tu control de mapa.'}
              </p>
            </div>
          </div>
        </SectionBlock>
      )}

    </div>
  );
};

export default UtilityTab;
