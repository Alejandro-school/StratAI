// frontend/src/components/TacticalMap/FlowLinesOverlay.jsx
// SVG overlay rendering movement flow corridors between map areas
import React, { useMemo } from 'react';

/**
 * Renders animated SVG flow lines showing movement patterns.
 * Line thickness ∝ frequency, color = side-dependent.
 */
const FlowLinesOverlay = React.memo(({ flowLines = [], activeSide = 'all', visible = true }) => {
  const processedLines = useMemo(() => {
    if (!flowLines.length) return [];

    // Filter by side if needed
    let filtered = flowLines;
    if (activeSide === 'ct') {
      filtered = flowLines.filter(l => (l.ct_ratio || 0) > 0.55);
    } else if (activeSide === 't') {
      filtered = flowLines.filter(l => (l.ct_ratio || 0) < 0.45);
    }

    // Sort by count (draw thicker lines on top)
    const sorted = [...filtered].sort((a, b) => (a.count || 0) - (b.count || 0));

    // Normalize for thickness
    const maxCount = Math.max(1, ...sorted.map(l => l.count || 1));

    return sorted.map((line) => {
      const count = line.count || 1;
      const normalized = count / maxCount;
      const thickness = 1.5 + normalized * 4; // 1.5 – 5.5px
      const opacity = 0.15 + normalized * 0.5; // 0.15 – 0.65

      // Determine color from CT/T ratio
      const ctRatio = line.ct_ratio ?? 0.5;
      let color;
      if (ctRatio > 0.65) {
        color = '#60a5fa'; // CT blue
      } else if (ctRatio < 0.35) {
        color = '#fb923c'; // T orange
      } else {
        color = '#818cf8'; // mixed purple
      }

      // Positions: from_pos and to_pos are {x, y} in 0-100% coords
      const from = line.from_pos || { x: 0, y: 0 };
      const to = line.to_pos || { x: 0, y: 0 };

      // Create a slight curve via midpoint offset for visual separation
      const midX = (from.x + to.x) / 2;
      const midY = (from.y + to.y) / 2;
      const dx = to.x - from.x;
      const dy = to.y - from.y;
      // Perpendicular offset (alternating direction based on position)
      const perpScale = 1.5;
      const ctrlX = midX + (-dy / Math.max(1, Math.sqrt(dx * dx + dy * dy))) * perpScale;
      const ctrlY = midY + (dx / Math.max(1, Math.sqrt(dx * dx + dy * dy))) * perpScale;

      return {
        key: `${line.from}-${line.to}`,
        from,
        to,
        ctrl: { x: ctrlX, y: ctrlY },
        thickness,
        opacity,
        color,
        label: `${line.from} → ${line.to}`,
        count,
      };
    });
  }, [flowLines, activeSide]);

  if (!visible || processedLines.length === 0) return null;

  return (
    <svg
      className="flow-lines-overlay"
      viewBox="0 0 100 100"
      preserveAspectRatio="none"
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        pointerEvents: 'none',
        zIndex: 3,
      }}
    >
      <defs>
        <marker
          id="flow-arrow"
          viewBox="0 0 10 6"
          refX="9"
          refY="3"
          markerWidth="6"
          markerHeight="4"
          orient="auto"
        >
          <path d="M 0 0 L 10 3 L 0 6 z" fill="currentColor" opacity="0.7" />
        </marker>
      </defs>

      {processedLines.map((line) => (
        <g key={line.key}>
          {/* Glow effect */}
          <path
            d={`M ${line.from.x} ${line.from.y} Q ${line.ctrl.x} ${line.ctrl.y} ${line.to.x} ${line.to.y}`}
            fill="none"
            stroke={line.color}
            strokeWidth={line.thickness + 2}
            strokeLinecap="round"
            opacity={line.opacity * 0.3}
          />
          {/* Main line */}
          <path
            d={`M ${line.from.x} ${line.from.y} Q ${line.ctrl.x} ${line.ctrl.y} ${line.to.x} ${line.to.y}`}
            fill="none"
            stroke={line.color}
            strokeWidth={line.thickness}
            strokeLinecap="round"
            strokeDasharray={`${line.thickness * 2} ${line.thickness}`}
            opacity={line.opacity}
            style={{ color: line.color }}
            markerEnd="url(#flow-arrow)"
          >
            <animate
              attributeName="stroke-dashoffset"
              from={line.thickness * 6}
              to="0"
              dur="2s"
              repeatCount="indefinite"
            />
          </path>
        </g>
      ))}
    </svg>
  );
});

export default FlowLinesOverlay;
