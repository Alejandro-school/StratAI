import React, { useMemo, useState } from 'react';
import { motion } from 'framer-motion';

const WIDTH = 760;
const HEIGHT = 230;
const PADDING = 24;

const createPoints = (values, min, max) => values.map((value, index) => ({
    x: PADDING + (index / Math.max(1, values.length - 1)) * (WIDTH - PADDING * 2),
    y: HEIGHT - PADDING - ((value - min) / (max - min)) * (HEIGHT - PADDING * 2),
    value,
  }));

const TrendChart = ({ context, comparison, compareMode }) => {
  const [hovered, setHovered] = useState(null);
  const currentValues = useMemo(() => context.history.map((item) => item.value), [context]);
  const comparisonValues = useMemo(
    () => comparison?.history?.slice(0, context.history.length) || [],
    [comparison, context.history.length],
  );
  const scale = useMemo(() => {
    const values = compareMode ? [...currentValues, ...comparisonValues] : currentValues;
    return { min: Math.min(...values) - 0.08, max: Math.max(...values) + 0.08 };
  }, [compareMode, currentValues, comparisonValues]);
  const points = useMemo(
    () => createPoints(currentValues, scale.min, scale.max),
    [currentValues, scale],
  );
  const comparisonPoints = useMemo(
    () => createPoints(comparisonValues, scale.min, scale.max),
    [comparisonValues, scale],
  );
  const path = points.map((point) => `${point.x},${point.y}`).join(' ');
  const comparisonPath = comparisonPoints.map((point) => `${point.x},${point.y}`).join(' ');

  return (
    <section className="pf3-panel pf3-trend-panel" aria-labelledby="trend-title">
      <div className="pf3-section-heading">
        <div>
          <span className="pf3-kicker">Últimas partidas</span>
          <h2 id="trend-title">Evolución del rating</h2>
        </div>
        <div className="pf3-chart-key">
          <span><i /> Tú</span>
          {compareMode && <span><i /> {comparison.name}</span>}
        </div>
      </div>

      <div className="pf3-chart-wrap">
        <svg viewBox={`0 0 ${WIDTH} ${HEIGHT}`} role="img" aria-label={`Evolución del rating en ${context.name}`}>
          <defs>
            <linearGradient id="pf3-area" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="#54dbff" stopOpacity="0.35" />
              <stop offset="1" stopColor="#54dbff" stopOpacity="0" />
            </linearGradient>
            <filter id="pf3-glow">
              <feGaussianBlur stdDeviation="4" result="blur" />
              <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
            </filter>
          </defs>
          {[45, 90, 135, 180].map((y) => <line key={y} x1="24" y1={y} x2="736" y2={y} className="pf3-chart-grid" />)}
          <polygon points={`${PADDING},${HEIGHT - PADDING} ${path} ${WIDTH - PADDING},${HEIGHT - PADDING}`} fill="url(#pf3-area)" />
          {compareMode && <polyline points={comparisonPath} className="pf3-comparison-line" />}
          <motion.polyline
            key={context.id}
            points={path}
            className="pf3-primary-line"
            filter="url(#pf3-glow)"
            initial={{ pathLength: 0 }}
            animate={{ pathLength: 1 }}
            transition={{ duration: 0.9, ease: 'easeOut' }}
          />
          {points.map((point, index) => (
            <circle
              key={`${context.id}-${index}`}
              cx={point.x}
              cy={point.y}
              r={hovered === index ? 7 : 4.5}
              className={context.history[index].result === 'W' ? 'pf3-point win' : 'pf3-point loss'}
              tabIndex="0"
              aria-label={`Partida ${index + 1}: rating ${point.value.toFixed(2)}, ${context.history[index].result === 'W' ? 'victoria' : 'derrota'}`}
              onMouseEnter={() => setHovered(index)}
              onMouseLeave={() => setHovered(null)}
              onFocus={() => setHovered(index)}
              onBlur={() => setHovered(null)}
            />
          ))}
        </svg>
        {hovered !== null && (
          <div
            className="pf3-chart-tooltip"
            style={{ left: `${(points[hovered].x / WIDTH) * 100}%`, top: `${(points[hovered].y / HEIGHT) * 100}%` }}
          >
            <small>Partida {hovered + 1}</small>
            <strong>{points[hovered].value.toFixed(2)}</strong>
            <span>{context.history[hovered].result === 'W' ? 'Victoria' : 'Derrota'}</span>
          </div>
        )}
      </div>
    </section>
  );
};

export default TrendChart;
