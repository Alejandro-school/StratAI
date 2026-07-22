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

const MiniSparkline = ({ data = [], tone = 'var(--p-accent)' }) => {
  const values = data
    .map((value) => Number(value))
    .filter((value) => Number.isFinite(value));

  if (values.length < 2) return null;

  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;

  const points = values
    .map((value, index) => {
      const x = (index / (values.length - 1)) * 100;
      const y = 24 - ((value - min) / range) * 20 - 2;
      return `${x},${y}`;
    })
    .join(' ');

  return (
    <span className="p-metric-sparkline" aria-hidden="true">
      <svg viewBox="0 0 100 24" preserveAspectRatio="none">
        <polyline
          fill="none"
          stroke={tone}
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
          points={points}
        />
      </svg>
    </span>
  );
};

export const MetricCell = ({ value, label, sub, badge, trend, sparkData, sparkTone }) => (
  <div className="p-metric-cell">
    <div className="p-metric-head">
      <span className="p-metric-value">
        {value}
        <Trend value={trend} />
      </span>
      <MiniSparkline data={sparkData} tone={sparkTone} />
    </div>
    <span className="p-metric-label">{label}</span>
    {sub && <span className="p-metric-sub">{sub}</span>}
    {badge && <span className={`p-badge ${badge.tone}`}>{badge.label}</span>}
  </div>
);

export default MetricCell;
