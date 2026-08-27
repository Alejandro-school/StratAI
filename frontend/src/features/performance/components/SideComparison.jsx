import React from 'react';
import { motion } from 'framer-motion';
import { Shield, Zap } from 'lucide-react';

const SIDE_METRICS = [
  { key: 'rating', label: 'Rating', max: 2.2, digits: 2 },
  { key: 'adr', label: 'ADR', max: 160, digits: 1 },
];

const SideComparison = ({ context, activeSide, onSideChange }) => (
  <section className="pf3-panel pf3-side-panel" aria-labelledby="side-comparison-title">
    <div className="pf3-section-heading">
      <div>
        <span className="pf3-kicker">Rendimiento por bando</span>
        <h2 id="side-comparison-title">CT vs. T</h2>
      </div>
      <div className="pf3-side-switch" aria-label="Filtrar por bando">
        <button
          type="button"
          className={activeSide === 'ct' ? 'is-active ct' : ''}
          aria-pressed={activeSide === 'ct'}
          onClick={() => onSideChange(activeSide === 'ct' ? 'all' : 'ct')}
        >
          <Shield size={15} aria-hidden="true" /> CT
        </button>
        <button
          type="button"
          className={activeSide === 't' ? 'is-active t' : ''}
          aria-pressed={activeSide === 't'}
          onClick={() => onSideChange(activeSide === 't' ? 'all' : 't')}
        >
          <Zap size={15} aria-hidden="true" /> T
        </button>
      </div>
    </div>

    <div className="pf3-side-table">
      <div className="pf3-side-header">
        <strong>CT</strong><span>Métrica</span><strong>T</strong>
      </div>
      {SIDE_METRICS.map((metric, index) => {
        const ct = context.ct[metric.key];
        const t = context.t[metric.key];
        return (
          <motion.div
            key={metric.key}
            className="pf3-side-row"
            initial={{ opacity: 0, scaleX: 0.94 }}
            animate={{ opacity: 1, scaleX: 1 }}
            transition={{ delay: index * 0.05 }}
          >
            <strong>{ct.toFixed(metric.digits)}{metric.suffix || ''}</strong>
            <div className="pf3-side-bars" aria-hidden="true">
              <span className="ct"><i style={{ width: `${(ct / metric.max) * 100}%` }} /></span>
              <span className="t"><i style={{ width: `${(t / metric.max) * 100}%` }} /></span>
            </div>
            <span>{metric.label}</span>
            <strong>{t.toFixed(metric.digits)}{metric.suffix || ''}</strong>
          </motion.div>
        );
      })}
    </div>
  </section>
);

export default SideComparison;
