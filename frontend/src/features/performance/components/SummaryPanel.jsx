import React from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { Activity, ArrowDownRight, ArrowUpRight, Swords, Trophy } from 'lucide-react';

const METRICS = [
  { key: 'rating', label: 'Rating', digits: 2, max: 2.2 },
  { key: 'adr', label: 'ADR', digits: 1, max: 160 },
  { key: 'kast', label: 'KAST', digits: 1, suffix: '%', max: 100 },
  { key: 'kd', label: 'K / D', digits: 2, max: 2.2 },
  { key: 'impact', label: 'Impacto', digits: 2, max: 2.6 },
];

const SummaryPanel = ({ context }) => {
  const TrendIcon = context.trend >= 0 ? ArrowUpRight : ArrowDownRight;
  return (
    <AnimatePresence mode="wait">
      <motion.aside
        key={context.id}
        className="pf3-summary"
        initial={{ opacity: 0, x: 26, filter: 'blur(8px)' }}
        animate={{ opacity: 1, x: 0, filter: 'blur(0px)' }}
        exit={{ opacity: 0, x: -18, filter: 'blur(6px)' }}
        transition={{ duration: 0.42, ease: [0.22, 1, 0.36, 1] }}
      >
        <div className="pf3-summary-head">
          <div>
            <span className="pf3-kicker">
              <Activity size={14} aria-hidden="true" />
              Resumen seleccionado
            </span>
            <h2>{context.name}</h2>
            <p>{context.matches} partidas analizadas</p>
          </div>
          <div className={`pf3-form-badge ${context.trend < 0 ? 'is-negative' : ''}`}>
            <TrendIcon size={16} aria-hidden="true" />
            {context.trend >= 0 ? '+' : ''}{context.trend}%
            <small>forma</small>
          </div>
        </div>

        <div className="pf3-record">
          <div><Trophy size={17} aria-hidden="true" /><strong>{context.wins}</strong><span>victorias</span></div>
          <span className="pf3-record-divider" />
          <div><Swords size={17} aria-hidden="true" /><strong>{context.losses}</strong><span>derrotas</span></div>
          <strong className="pf3-win-rate">{context.winRate}%<small>win rate</small></strong>
        </div>

        <div className="pf3-metric-stack">
          {METRICS.map((metric, index) => {
            const value = Number(context[metric.key]) || 0;
            return (
              <motion.div
                key={metric.key}
                className="pf3-metric-row"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.08 + index * 0.045 }}
              >
                <span>{metric.label}</span>
                <div className="pf3-metric-track" aria-hidden="true">
                  <span style={{ width: `${Math.min(100, value / metric.max * 100)}%` }} />
                </div>
                <strong>{value.toFixed(metric.digits)}{metric.suffix || ''}</strong>
              </motion.div>
            );
          })}
        </div>
      </motion.aside>
    </AnimatePresence>
  );
};

export default SummaryPanel;
