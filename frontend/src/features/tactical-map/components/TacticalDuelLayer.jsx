import React, { useMemo } from 'react';
import { formatCalloutName } from '../../../components/TacticalMap/tacticalPresentation';

const clamp = (value, min, max) => Math.min(max, Math.max(min, value));

const METRICS = {
  volume: {
    label: 'volumen',
    value: (callout) => callout.sample_size ?? 0,
    sort: (callout) => callout.sample_size ?? 0,
  },
  efficiency: {
    label: 'eficiencia ajustada',
    value: (callout) => `${Math.round(callout.adjustedWinRate ?? callout.win_rate ?? 50)}%`,
    sort: (callout, baseline) => Math.abs((callout.adjustedWinRate ?? 50) - baseline),
  },
  impact: {
    label: 'impacto',
    value: (callout) => Math.round(callout.impactScore ?? 0),
    sort: (callout) => callout.impactScore ?? 0,
  },
  risk: {
    label: 'riesgo',
    value: (callout) => Math.round(callout.riskScore ?? 0),
    sort: (callout) => callout.riskScore ?? 0,
  },
};

const getTone = (callout, metric, baselineWinRate) => {
  if (metric === 'risk') return (callout.riskScore ?? 0) >= 55 ? 'negative' : 'neutral';
  if (metric === 'impact') return (callout.impactScore ?? 0) >= 65 ? 'positive' : 'neutral';

  const delta = (callout.adjustedWinRate ?? callout.win_rate ?? 50) - baselineWinRate;
  if (delta >= 3) return 'positive';
  if (delta <= -3) return 'negative';
  return 'neutral';
};

const TacticalDuelLayer = ({
  callouts = [],
  metric = 'volume',
  baselineWinRate = 50,
  selectedCallout,
  onSelect,
}) => {
  const metricConfig = METRICS[metric] ?? METRICS.volume;
  const visibleCallouts = useMemo(() => {
    const positioned = callouts.filter((callout) => (
      Number.isFinite(callout.position?.x)
      && Number.isFinite(callout.position?.y)
    ));
    const sorted = [...positioned].sort(
      (a, b) => metricConfig.sort(b, baselineWinRate) - metricConfig.sort(a, baselineWinRate)
    );
    const limited = sorted.slice(0, metric === 'volume' ? 16 : 12);
    const selected = sorted.find((callout) => callout.name === selectedCallout);

    if (selected && !limited.some((callout) => callout.name === selected.name)) {
      return [selected, ...limited.slice(0, limited.length - 1)];
    }
    return limited;
  }, [baselineWinRate, callouts, metric, metricConfig]);

  return (
    <div className={`tactical-duel-layer is-${metric}`} aria-label={`${metricConfig.label} por zona`}>
      {visibleCallouts.map((callout) => {
        const tone = getTone(callout, metric, baselineWinRate);
        const markerSize = clamp(42 + ((callout.volumeShare ?? 0) / 100) * 150, 42, 64);
        const selected = selectedCallout === callout.name;
        const adjustedRate = Math.round(callout.adjustedWinRate ?? callout.win_rate ?? 50);
        const confidence = callout.confidenceLabel ?? 'low';
        const name = formatCalloutName(callout.name);
        const value = metricConfig.value(callout);

        return (
          <button
            key={callout.name}
            type="button"
            className={`tactical-zone-node is-${tone} confidence-${confidence} ${selected ? 'is-selected' : ''}`}
            style={{
              '--node-x': `${clamp(callout.position.x, 2, 98)}%`,
              '--node-y': `${clamp(callout.position.y, 2, 98)}%`,
              '--node-size': `${markerSize}px`,
            }}
            aria-pressed={selected}
            aria-label={`${name}: ${metricConfig.label} ${value}; ${callout.sample_size ?? 0} duelos; confianza ${confidence}.`}
            onClick={() => onSelect(callout)}
          >
            <span className="tactical-zone-node__core" aria-hidden="true">{value}</span>
            <span className="tactical-zone-node__label">
              <strong>{name}</strong>
              <small>{adjustedRate}% ajustado · {callout.sample_size ?? 0} duelos</small>
            </span>
          </button>
        );
      })}
    </div>
  );
};

export default React.memo(TacticalDuelLayer);
