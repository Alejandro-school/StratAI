import React from 'react';
import { MetricGrid, StatBar, StatPanel } from './StatisticsPrimitives';

const EconomySection = ({ data }) => (
  <div className="pf3-stat-layout">
    <MetricGrid metrics={data.metrics} />
    <StatPanel eyebrow="Distribución de compras" title="Rendimiento por tipo de buy">
      <div className="pf3-buy-list">
        {data.buyTypes.map((buy) => (
          <StatBar
            key={buy.label}
            label={buy.label}
            value={buy.value}
            detail={`${buy.winRate}% de rondas ganadas`}
          />
        ))}
      </div>
    </StatPanel>
    <StatPanel eyebrow="Disciplina de equipo" title="Decisiones económicas">
      <div className="pf3-decision-list">
        {data.decisions.map((decision) => (
          <article key={decision.label}>
            <span>{decision.label}</span>
            <strong className={`is-${decision.tone}`}>{decision.value}</strong>
          </article>
        ))}
      </div>
    </StatPanel>
  </div>
);

export default EconomySection;
