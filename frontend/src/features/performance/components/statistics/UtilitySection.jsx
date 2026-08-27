import React from 'react';
import { MetricGrid, StatBar, StatPanel } from './StatisticsPrimitives';
import PerformanceWeaponIcon from './PerformanceWeaponIcon';

const UtilitySection = ({ data }) => (
  <div className="pf3-stat-layout">
    <MetricGrid metrics={data.metrics} />
    <StatPanel
      eyebrow="Uso y resultado"
      title="Eficiencia por tipo de granada"
      aside="Agregado de la muestra seleccionada"
      className="pf3-utility-panel"
    >
      <div className="pf3-utility-list">
        {data.grenades.map((grenade) => (
          <article key={grenade.name}>
            <div className="pf3-utility-name">
              <PerformanceWeaponIcon weapon={grenade.name} />
              <div>
                <strong>{grenade.name}</strong>
                <small>{grenade.used} usadas</small>
              </div>
            </div>
            <div className="pf3-utility-result">
              <strong>{grenade.primary}</strong>
              <span>{grenade.primaryLabel}</span>
              <small>{grenade.secondary}</small>
            </div>
            <StatBar value={grenade.score} label={grenade.scoreLabel} />
          </article>
        ))}
      </div>
    </StatPanel>
  </div>
);

export default UtilitySection;
