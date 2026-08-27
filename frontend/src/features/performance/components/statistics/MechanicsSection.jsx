import React from 'react';
import { MetricGrid, StatBar, StatPanel } from './StatisticsPrimitives';

const MechanicsSection = ({ data }) => (
  <div className="pf3-stat-layout">
    <MetricGrid metrics={data.metrics} />
    <StatPanel eyebrow="Calidad del enfrentamiento" title="Exposición y preparación">
      <div className="pf3-exposure-grid">
        {data.exposure.map((item) => (
          <StatBar key={item.label} label={item.label} value={item.value} tone="mint" />
        ))}
      </div>
    </StatPanel>
    <StatPanel eyebrow="Contexto al recibir daño" title="Estados de riesgo">
      <div className="pf3-state-grid">
        {data.states.map((state) => (
          <article key={state.label}>
            <span>{state.label}</span>
            <strong>{state.value}</strong>
            <small>{state.note}</small>
          </article>
        ))}
      </div>
    </StatPanel>
  </div>
);

export default MechanicsSection;
