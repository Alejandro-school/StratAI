import React from 'react';
import { Star } from 'lucide-react';
import SideComparison from '../SideComparison';
import TrendChart from '../TrendChart';
import PerformanceWeaponIcon from './PerformanceWeaponIcon';
import { MetricGrid, StatPanel } from './StatisticsPrimitives';

const OverviewSection = ({
  context,
  activeSide,
  onSideChange,
  data,
}) => (
  <div className="pf3-stat-layout">
    <MetricGrid metrics={data.metrics} />

    <SideComparison
      context={context}
      activeSide={activeSide}
      onSideChange={onSideChange}
    />
    <TrendChart
      context={context}
    />

    <StatPanel
      eyebrow="Arma de referencia"
      title={data.favoriteWeapon.name}
      aside={<Star size={16} aria-hidden="true" />}
      className="pf3-favorite-weapon"
    >
      <PerformanceWeaponIcon weapon={data.favoriteWeapon.name} featured />
      <div>
        <strong>{data.favoriteWeapon.kills}</strong><span>bajas</span>
        <strong>{data.favoriteWeapon.damagePerKill}</strong><span>daño / baja</span>
        <strong>{data.favoriteWeapon.accuracy}%</strong><span>precisión</span>
        <strong>{data.favoriteWeapon.headshots}%</strong><span>headshots</span>
      </div>
    </StatPanel>

    <StatPanel eyebrow="Conversión por ronda" title="Multikills" className="pf3-multikills">
      <div>
        {data.multikills.map((item) => (
          <span key={item.label}>
            <strong>{item.value}</strong>
            <small>{item.label}</small>
          </span>
        ))}
      </div>
    </StatPanel>
  </div>
);

export default OverviewSection;
