import React from 'react';
import { Award, Crosshair, Flame, ShieldCheck, Swords, Trophy } from 'lucide-react';
import StatPill from './StatPill';
import { getQualityLabel } from '../../utils/performanceBenchmarks';
import { formatDecimal, formatPercent } from '../../utils/performanceFormatters';

const HeroStats = ({ overview = {}, trends = {} }) => {
  const rating = Number(overview.hltv_rating || 0);
  const kast = Number(overview.kast || 0);
  const kd = Number(overview.kd_ratio || 0);
  const adr = Number(overview.adr || 0);
  const hs = Number(overview.hs_pct || 0);
  const wr = Number(overview.win_rate || 0);

  return (
    <section className="hero-stats-grid">
      <StatPill
        label="Rating HLTV"
        value={formatDecimal(rating, 2)}
        icon={Award}
        tone="purple"
        helpText="Índice global de rendimiento por ronda. Combina daño, impacto y supervivencia."
        badge={getQualityLabel(rating, 'hltv_rating')}
        trend={trends.hltv_rating}
      />
      <StatPill
        label="KAST"
        value={formatPercent(kast)}
        icon={ShieldCheck}
        tone="green"
        helpText="Porcentaje de rondas con kill, asistencia, supervivencia o trade."
        badge={getQualityLabel(kast, 'kast')}
      />
      <StatPill
        label="K/D"
        value={formatDecimal(kd, 2)}
        icon={Swords}
        tone={kd >= 1 ? 'blue' : 'red'}
        helpText="Relación entre bajas y muertes. Valores por encima de 1 indican ventaja."
        badge={getQualityLabel(kd, 'kd_ratio')}
        trend={trends.kd_ratio}
      />
      <StatPill
        label="ADR"
        value={formatDecimal(adr, 1)}
        icon={Flame}
        tone="orange"
        helpText="Daño promedio por ronda."
        badge={getQualityLabel(adr, 'adr')}
        trend={trends.adr}
      />
      <StatPill
        label="% Headshot"
        value={formatPercent(hs)}
        icon={Crosshair}
        tone="purple"
        helpText="Porcentaje de bajas conseguidas con disparo a la cabeza."
        badge={getQualityLabel(hs, 'hs_pct')}
        trend={trends.hs_pct}
      />
      <StatPill
        label="% Victoria"
        value={formatPercent(wr)}
        subtext={`${overview.wins || 0}W - ${overview.losses || 0}L`}
        icon={Trophy}
        tone={wr >= 50 ? 'green' : 'red'}
        helpText="Partidas ganadas sobre el total analizado."
        badge={getQualityLabel(wr, 'win_rate')}
      />
    </section>
  );
};

export default HeroStats;
