import React from 'react';
import { ArrowDownRight, ArrowUpRight } from 'lucide-react';

export const MetricGrid = ({ metrics, compact = false }) => (
  <div className={`pf3-stat-metrics ${compact ? 'is-compact' : ''}`}>
    {metrics.map((metric) => {
      const positive = metric.delta > 0;
      const negative = metric.delta < 0;
      return (
        <article className="pf3-stat-metric" key={metric.label}>
          <span>{metric.label}</span>
          <strong>{metric.value}</strong>
          <small>{metric.note}</small>
          {metric.delta !== undefined && (
            <em className={positive ? 'is-positive' : negative ? 'is-negative' : ''}>
              {positive
                ? <ArrowUpRight size={13} aria-hidden="true" />
                : <ArrowDownRight size={13} aria-hidden="true" />}
              {Math.abs(metric.delta)}%
            </em>
          )}
        </article>
      );
    })}
  </div>
);

export const StatPanel = ({ eyebrow, title, aside, children, className = '' }) => (
  <section className={`pf3-stat-panel ${className}`}>
    <header>
      <div>
        {eyebrow && <span>{eyebrow}</span>}
        <h3>{title}</h3>
      </div>
      {aside && <small>{aside}</small>}
    </header>
    {children}
  </section>
);

export const StatBar = ({ label, value, suffix = '%', tone = 'cyan', detail }) => (
  <div className="pf3-stat-bar">
    <div>
      <span>{label}</span>
      <strong>{value}{suffix}</strong>
    </div>
    <div className="pf3-stat-bar-track">
      <i className={`tone-${tone}`} style={{ width: `${Math.min(value, 100)}%` }} />
    </div>
    {detail && <small>{detail}</small>}
  </div>
);
