import React, { useMemo } from 'react';

const resolvePosition = (line, prefix) => {
  const nested = line[`${prefix}_pos`];
  if (nested && Number.isFinite(nested.x) && Number.isFinite(nested.y)) {
    return nested;
  }

  const x = Number(line[`${prefix}_x`]);
  const y = Number(line[`${prefix}_y`]);
  return Number.isFinite(x) && Number.isFinite(y) ? { x, y } : null;
};

const resolveLineCounts = (line, activeSide) => {
  const total = Number(line.count ?? 0);
  const ct = Number(line.ct_count ?? line.ct ?? 0);
  const t = Number(line.t_count ?? line.t ?? Math.max(0, total - ct));
  const visible = activeSide === 'ct' ? ct : activeSide === 't' ? t : total;
  return { total, ct, t, visible };
};

const resolveRouteColor = ({ ct, t }, activeSide) => {
  if (activeSide === 'ct') return { color: '#63b3ff', marker: 'ct' };
  if (activeSide === 't') return { color: '#f5ad58', marker: 't' };

  const ratio = ct + t > 0 ? ct / (ct + t) : 0.5;
  if (ratio > 0.65) return { color: '#63b3ff', marker: 'ct' };
  if (ratio < 0.35) return { color: '#f5ad58', marker: 't' };
  return { color: '#78d7e8', marker: 'mixed' };
};

const isRouteOnLevel = (line, currentLevel, zThreshold) => {
  const resolveLevel = (prefix) => {
    if (line[`${prefix}_level`]) return line[`${prefix}_level`];
    const avgZ = Number(line[`${prefix}_avg_z`]);
    return Number.isFinite(avgZ)
      ? (avgZ >= zThreshold ? 'upper' : 'lower')
      : null;
  };
  return resolveLevel('from') === currentLevel && resolveLevel('to') === currentLevel;
};

const FlowLinesOverlay = React.memo(({
  flowLines = [],
  activeSide = 'all',
  visible = true,
  hasLevels = false,
  currentLevel = 'upper',
  zThreshold = null,
}) => {
  const processedLines = useMemo(() => {
    const validLines = flowLines.reduce((result, line) => {
      const from = resolvePosition(line, 'from');
      const to = resolvePosition(line, 'to');
      const counts = resolveLineCounts(line, activeSide);

      if (
        !from
        || !to
        || counts.visible <= 0
        || (
          hasLevels
          && !isRouteOnLevel(line, currentLevel, zThreshold)
        )
      ) return result;

      result.push({
        from,
        to,
        counts,
        fromArea: line.from_area ?? line.from ?? 'Origen',
        toArea: line.to_area ?? line.to ?? 'Destino',
      });
      return result;
    }, []);

    const visibleLines = validLines
      .sort((a, b) => a.counts.visible - b.counts.visible)
      .slice(-10);
    const maxCount = Math.max(1, ...visibleLines.map(({ counts }) => counts.visible));

    return visibleLines.map((line) => {
      const normalized = line.counts.visible / maxCount;
      const dx = line.to.x - line.from.x;
      const dy = line.to.y - line.from.y;
      const distance = Math.max(1, Math.sqrt(dx * dx + dy * dy));
      const colorConfig = resolveRouteColor(line.counts, activeSide);

      return {
        ...line,
        ...colorConfig,
        key: `${line.fromArea}-${line.toArea}`,
        label: `${line.fromArea} → ${line.toArea}`,
        thickness: 0.45 + normalized * 1.1,
        opacity: 0.22 + normalized * 0.5,
        control: {
          x: (line.from.x + line.to.x) / 2 + (-dy / distance) * 1.5,
          y: (line.from.y + line.to.y) / 2 + (dx / distance) * 1.5,
        },
      };
    });
  }, [activeSide, currentLevel, flowLines, hasLevels, zThreshold]);

  if (!visible || processedLines.length === 0) return null;

  return (
    <svg
      className="flow-lines-overlay"
      viewBox="0 0 100 100"
      preserveAspectRatio="none"
      role="img"
      aria-label={`${processedLines.length} transiciones de zona observadas`}
    >
      <defs>
        {[
          ['ct', '#63b3ff'],
          ['t', '#f5ad58'],
          ['mixed', '#78d7e8'],
        ].map(([id, color]) => (
          <marker
            key={id}
            id={`flow-arrow-${id}`}
            viewBox="0 0 10 6"
            refX="9"
            refY="3"
            markerWidth="5"
            markerHeight="3"
            orient="auto"
          >
            <path d="M 0 0 L 10 3 L 0 6 z" fill={color} opacity="0.9" />
          </marker>
        ))}
      </defs>

      {processedLines.map((line, index) => {
        const path = `M ${line.from.x} ${line.from.y} Q ${line.control.x} ${line.control.y} ${line.to.x} ${line.to.y}`;

        return (
          <g key={line.key}>
            <title>{`${line.label}: ${line.counts.visible} transiciones observadas`}</title>
            <path
              d={path}
              fill="none"
              stroke={line.color}
              strokeWidth={line.thickness + 1.2}
              strokeLinecap="round"
              opacity={line.opacity * 0.2}
              vectorEffect="non-scaling-stroke"
            />
            <path
              className="flow-route"
              d={path}
              fill="none"
              stroke={line.color}
              strokeWidth={line.thickness}
              strokeLinecap="round"
              opacity={line.opacity}
              markerEnd={`url(#flow-arrow-${line.marker})`}
              vectorEffect="non-scaling-stroke"
              style={{ '--route-delay': `${Math.min(index * 45, 360)}ms` }}
            />
          </g>
        );
      })}
    </svg>
  );
});

FlowLinesOverlay.displayName = 'FlowLinesOverlay';

export default FlowLinesOverlay;
