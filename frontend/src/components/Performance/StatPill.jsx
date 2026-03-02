import React from 'react';
import { TrendingUp, TrendingDown } from 'lucide-react';

/**
 * MetricCell — a single cell inside a p-metric-row band.
 * Used at the top of every tab to show the 3-4 key numbers for that section.
 */
const Trend = ({ value }) => {
  if (value == null || !Number.isFinite(value) || Math.abs(value) < 0.01) return null;
  const up = value > 0;
  const Icon = up ? TrendingUp : TrendingDown;
  const sign = up ? '+' : '';
  const abs = Math.abs(value);
  const display = abs >= 10 ? `${sign}${Math.round(value)}` : `${sign}${value.toFixed(abs >= 1 ? 1 : 2)}`;
  return (
    <span className={`p-trend ${up ? 'up' : 'down'}`} aria-label={`Tendencia: ${display}`}>
      <Icon />
      {display}
    </span>
  );
};

export const MetricCell = ({ value, label, sub, badge, trend }) => (
  <div className="p-metric-cell">
    <span className="p-metric-value">
      {value}
      <Trend value={trend} />
    </span>
    <span className="p-metric-label">{label}</span>
    {sub && <span className="p-metric-sub">{sub}</span>}
    {badge && <span className={`p-badge ${badge.tone}`}>{badge.label}</span>}
  </div>
);

export default MetricCell;
