import React from 'react';
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts';
import { Shield, Swords, TrendingUp } from 'lucide-react';
import {
  RECHARTS_TOOLTIP_STYLE,
  TOOLTIP_ITEM_STYLE,
  TOOLTIP_LABEL_STYLE,
  formatDecimal,
  formatInteger,
  formatPercent,
} from '../../utils/performanceFormatters';
import { getQualityLabel } from '../../utils/performanceBenchmarks';
import { MetricCell } from './StatPill';
import SectionBlock from './SectionBlock';
import { BriefingPanel, InsightCard } from './PerformanceBriefing';

const clutchKeys = ['1v1', '1v2', '1v3', '1v4', '1v5'];
const multikillDefs = [
  { key: '2k', label: '2K', cls: 'mk-2k' },
  { key: '3k', label: '3K', cls: 'mk-3k' },
  { key: '4k', label: '4K', cls: 'mk-4k' },
  { key: 'ace', label: 'ACE', cls: 'mk-ace' },
];

const CombatTab = ({ combat = {}, overview = {}, economy = {} }) => {
  const clutches = combat.clutches || {};
  const multikills = combat.multikills || {};
  const openingWon = Number(combat.opening_duels_won || 0);
  const openingLost = Number(combat.opening_duels_lost || 0);
  const tradeKills = Number(combat.trade_kills || 0);
  const tradedDeaths = Number(combat.traded_deaths || 0);
  const clutchValues = clutchKeys.map((key) => Number(clutches[key] || 0));
  const maxClutch = Math.max(...clutchValues, 1);
  const multikillValues = multikillDefs.map((item) => Number(multikills[item.key] || 0));
  const maxMultikill = Math.max(...multikillValues, 1);
  const tradeRatio = tradedDeaths > 0 ? tradeKills / tradedDeaths : tradeKills > 0 ? tradeKills : 0;
  const duelData = [
    { name: 'Ganados', value: openingWon, color: '#22c55e' },
    { name: 'Perdidos', value: openingLost, color: '#ef4444' },
  ].filter((item) => item.value > 0);

  const totalRounds = Number(economy.total_rounds || 0);
  const kills = Number(overview.kills || 0);
  const deaths = Number(overview.deaths || 0);
  const assists = Number(overview.assists || 0);
  const totalDamage = Number(overview.total_damage || 0);
  const killsPerRound = totalRounds > 0 ? kills / totalRounds : 0;
  const deathsPerRound = totalRounds > 0 ? deaths / totalRounds : 0;
  const assistsPerRound = totalRounds > 0 ? assists / totalRounds : 0;
  const damagePerKill = kills > 0 ? totalDamage / kills : 0;
  const totalClutchWins = clutchValues.reduce((sum, value) => sum + value, 0);
  const firstBloodRate = Number(overview.total_matches || 0) > 0
    ? (openingWon / Number(overview.total_matches)) * 100 : 0;

  return (
    <div className="p-section">
      <BriefingPanel eyebrow="Briefing de combate" title="Iniciativa, trades y cierre de ronda">
        <div className="p-briefing-grid-3">
          <InsightCard
            tone={Number(combat.opening_success_rate || 0) >= 50 ? 'good' : 'warning'}
            icon={Swords}
            label="Primer contacto"
            value={formatPercent(combat.opening_success_rate)}
            detail={`${formatInteger(openingWon)} ganados / ${formatInteger(openingLost)} perdidos`}
          />
          <InsightCard
            tone={tradeRatio >= 1 ? 'good' : 'danger'}
            icon={TrendingUp}
            label="Intercambios"
            value={formatDecimal(tradeRatio, 2)}
            detail={`${formatInteger(tradeKills)} trades ejecutados`}
          />
          <InsightCard
            tone={totalClutchWins > 0 ? 'good' : 'neutral'}
            icon={Shield}
            label="Cierre"
            value={formatInteger(totalClutchWins)}
            detail="clutches ganados registrados"
          />
        </div>
      </BriefingPanel>

      <div className="p-metric-row">
        <MetricCell
          value={formatPercent(combat.opening_success_rate)}
          label="Exito en aperturas"
          badge={getQualityLabel(Number(combat.opening_success_rate || 0), 'opening_success')}
        />
        <MetricCell
          value={formatDecimal(overview.impact_rating, 2)}
          label="Impacto"
          badge={getQualityLabel(Number(overview.impact_rating || 0), 'impact_rating')}
        />
        <MetricCell
          value={formatDecimal(tradeRatio, 2)}
          label="Ratio de intercambios"
          sub={`${formatInteger(tradeKills)} bajas · ${formatInteger(tradedDeaths)} muertes intercambiadas`}
        />
        <MetricCell
          value={formatPercent(overview.kast)}
          label="KAST"
          badge={getQualityLabel(Number(overview.kast || 0), 'kast')}
        />
      </div>

      <SectionBlock title="Resumen de combate">
        <div className="p-kpi-strip">
          <div className="p-kpi-cell">
            <span className="p-kpi-cell-value">{formatInteger(kills)}</span>
            <span className="p-kpi-cell-label">Bajas</span>
            <span className="p-kpi-cell-sub">{formatDecimal(killsPerRound, 2)} / ronda</span>
          </div>
          <div className="p-kpi-cell">
            <span className="p-kpi-cell-value">{formatInteger(deaths)}</span>
            <span className="p-kpi-cell-label">Muertes</span>
            <span className="p-kpi-cell-sub">{formatDecimal(deathsPerRound, 2)} / ronda</span>
          </div>
          <div className="p-kpi-cell">
            <span className="p-kpi-cell-value">{formatInteger(assists)}</span>
            <span className="p-kpi-cell-label">Asistencias</span>
            <span className="p-kpi-cell-sub">{formatDecimal(assistsPerRound, 2)} / ronda</span>
          </div>
          <div className="p-kpi-cell">
            <span className="p-kpi-cell-value">{formatDecimal(damagePerKill, 0)}</span>
            <span className="p-kpi-cell-label">Daño / baja</span>
            <span className="p-kpi-cell-sub">{formatInteger(totalDamage)} daño total</span>
          </div>
          <div className="p-kpi-cell">
            <span className="p-kpi-cell-value">{formatDecimal(firstBloodRate, 1)}%</span>
            <span className="p-kpi-cell-label">% de primeras bajas</span>
            <span className="p-kpi-cell-sub">{formatInteger(openingWon)} primeras bajas</span>
          </div>
          <div className="p-kpi-cell">
            <span className="p-kpi-cell-value">{formatInteger(totalClutchWins)}</span>
            <span className="p-kpi-cell-label">Clutches ganados</span>
            <span className="p-kpi-cell-sub">{formatInteger(Number(combat.flash_assists || 0))} asistencias con cegadora</span>
          </div>
        </div>
      </SectionBlock>

      <SectionBlock title="Duelos iniciales e intercambios">
        <div className="p-grid p-grid-dashboard">
          <div className="p-card p-card--chart">
            <div className="p-card-toolbar">
              <div>
                <p className="p-card-title">Duelos iniciales</p>
                <p className="p-card-subtitle">Cuánto castigas o cedes el primer duelo de la ronda</p>
              </div>
            </div>
            <div className="p-duel-layout">
              <div className="p-donut-wrap">
                {duelData.length > 0 ? (
                  <ResponsiveContainer width="100%" height={220}>
                    <PieChart>
                      <Pie data={duelData} dataKey="value" innerRadius={58} outerRadius={84} paddingAngle={4}>
                        {duelData.map((item) => (
                          <Cell key={item.name} fill={item.color} />
                        ))}
                      </Pie>
                      <Tooltip
                        contentStyle={RECHARTS_TOOLTIP_STYLE}
                        labelStyle={TOOLTIP_LABEL_STYLE}
                        itemStyle={TOOLTIP_ITEM_STYLE}
                        formatter={(value) => [formatInteger(value), 'Duelos']}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="p-empty p-empty--chart">
                    <span>Sin duelos iniciales registrados</span>
                  </div>
                )}
                <div className="p-donut-center">
                  <strong>{formatPercent(combat.opening_success_rate)}</strong>
                  <span>tasa de éxito</span>
                </div>
              </div>

              <div className="p-stat-list p-stat-list--dense">
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Intentados</span>
                  <span className="p-stat-row-value">{formatInteger(combat.opening_duels_attempted)}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Ganados</span>
                  <span className="p-stat-row-value p-good-text">{formatInteger(openingWon)}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Perdidos</span>
                  <span className="p-stat-row-value p-bad-text">{formatInteger(openingLost)}</span>
                </div>
                <div className="p-stat-row">
                  <span className="p-stat-row-label">Asistencias con cegadora</span>
                  <span className="p-stat-row-value">{formatInteger(combat.flash_assists)}</span>
                </div>
              </div>
            </div>
          </div>

          <div className="p-stack-col">
            <div className="p-card">
              <p className="p-card-title">Intercambios</p>
              <div className="p-compare-grid">
                <div className="p-compare-block good">
                  <span className="p-compare-label">Bajas de intercambio</span>
                  <strong className="p-compare-value">{formatInteger(tradeKills)}</strong>
                </div>
                <div className="p-compare-block bad">
                  <span className="p-compare-label">Muertes intercambiadas</span>
                  <strong className="p-compare-value">{formatInteger(tradedDeaths)}</strong>
                </div>
              </div>
              <div className="p-progress-wrap p-mt-14">
                <div className="p-progress-head">
                  <span className="p-progress-head-label">Eficacia en intercambios</span>
                  <span className="p-progress-head-value">{formatDecimal(tradeRatio, 2)}</span>
                </div>
                <div className="p-split-bar p-split-bar--lg">
                  <div
                    className="p-split-bar-ct p-split-bar-success"
                    style={{ width: `${(() => {
                      const total = tradeKills + tradedDeaths;
                      return total > 0 ? (tradeKills / total) * 100 : 50;
                    })()}%` }}
                  />
                  <div
                    className="p-split-bar-t p-split-bar-danger"
                    style={{ width: `${(() => {
                      const total = tradeKills + tradedDeaths;
                      return total > 0 ? (tradedDeaths / total) * 100 : 50;
                    })()}%` }}
                  />
                </div>
              </div>
            </div>

            <div className="p-card">
              <p className="p-card-title">Lectura rápida</p>
              <p className="p-insight-copy">
                {openingWon + openingLost === 0
                  ? 'Todavía no hay muestra suficiente para valorar tus duelos iniciales.'
                  : combat.opening_success_rate >= 55
                    ? 'Tu primera bala está generando ventaja real. Mantener este volumen de aperturas merece construcción táctica alrededor.'
                    : 'Tus primeras peleas están cediendo demasiada iniciativa. Aquí hay valor inmediato para coaching y toma de decisiones.'}
              </p>
            </div>
          </div>
        </div>
      </SectionBlock>

      <SectionBlock title="Cierre de rondas">
        <div className="p-grid p-grid-3">
          <div className="p-card">
            <div className="p-card-toolbar">
              <p className="p-card-title">Clutches</p>
              <span className="p-card-subtitle">{formatInteger(totalClutchWins)} clutches ganados</span>
            </div>
            <div className="p-clutch-grid">
              {clutchKeys.map((key, index) => {
                const value = clutchValues[index];
                const opacity = value > 0 ? (value / maxClutch) * 0.85 + 0.15 : 0;
                return (
                  <div
                    key={key}
                    className={`p-mk-item ${value === 0 ? 'zero' : ''}`}
                    style={{ '--mk-glow': 'rgba(59, 130, 246, 0.24)', '--mk-opacity': opacity }}
                  >
                    <span className="p-mk-item-label">{key}</span>
                    <span className="p-mk-item-value">{value}</span>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="p-card">
            <div className="p-card-toolbar">
              <p className="p-card-title">Multi-bajas</p>
              <span className="p-card-subtitle">Picos de impacto que ganan rondas</span>
            </div>
            <div className="p-mk-grid">
              {multikillDefs.map((item, index) => {
                const value = multikillValues[index];
                const opacity = value > 0 ? (value / maxMultikill) * 0.85 + 0.15 : 0;
                return (
                  <div
                    key={item.key}
                    className={`p-mk-item ${item.cls} ${value === 0 ? 'zero' : ''}`}
                    style={{ '--mk-opacity': opacity }}
                  >
                    <span className="p-mk-item-label">{item.label}</span>
                    <span className="p-mk-item-value">{value}</span>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="p-card">
            <p className="p-card-title">Eficiencia de combate</p>
            <div className="p-stat-list">
              <div className="p-stat-row">
                <span className="p-stat-row-label">Relación K/D</span>
                <span className="p-stat-row-value">{formatDecimal(overview.kd_ratio, 2)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">ADR</span>
                <span className="p-stat-row-value">{formatDecimal(overview.adr, 1)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Kills / ronda</span>
                <span className="p-stat-row-value">{formatDecimal(killsPerRound, 2)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Daño por baja</span>
                <span className="p-stat-row-value">{formatDecimal(damagePerKill, 0)}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Multi-bajas totales</span>
                <span className="p-stat-row-value">{formatInteger(multikillValues.reduce((s, v) => s + v, 0))}</span>
              </div>
              <div className="p-stat-row">
                <span className="p-stat-row-label">Partidas analizadas</span>
                <span className="p-stat-row-value">{formatInteger(overview.total_matches)}</span>
              </div>
            </div>
          </div>
        </div>
      </SectionBlock>
    </div>
  );
};

export default CombatTab;
