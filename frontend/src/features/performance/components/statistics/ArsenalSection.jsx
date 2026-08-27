import React from 'react';
import PerformanceWeaponIcon from './PerformanceWeaponIcon';
import { MetricGrid, StatBar, StatPanel } from './StatisticsPrimitives';

const ArsenalSection = ({ data }) => (
  <div className="pf3-stat-layout">
    <MetricGrid metrics={data.metrics} />
    <StatPanel
      eyebrow="Rendimiento individual"
      title="Armas principales"
      aside="Ordenadas por uso"
      className="pf3-weapons-panel"
    >
      <div className="pf3-weapon-table">
        <div className="pf3-weapon-head">
          <span>Arma</span><span>Bajas</span><span>Daño/baja</span>
          <span>Precisión</span><span>HS</span><span>Daño</span>
        </div>
        {data.rows.map((weapon) => (
          <div className="pf3-weapon-row" key={weapon.name}>
            <strong>
              <PerformanceWeaponIcon weapon={weapon.name} />
              <span>{weapon.name}</span>
              <i style={{ width: `${weapon.usage}%` }} />
            </strong>
            <span>{weapon.kills}</span>
            <span>{weapon.damagePerKill}</span>
            <span>{weapon.accuracy}%</span>
            <span>{weapon.hs}%</span>
            <span>{weapon.damage.toLocaleString('es-ES')}</span>
          </div>
        ))}
      </div>
    </StatPanel>
    <StatPanel eyebrow="Distribución corporal" title="Zonas de impacto" className="pf3-hit-panel">
      {data.hitDistribution.map((hit) => (
        <StatBar key={hit.label} label={hit.label} value={hit.value} />
      ))}
    </StatPanel>
  </div>
);

export default ArsenalSection;
