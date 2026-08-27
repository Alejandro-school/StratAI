import React, { useCallback, useEffect, useMemo, useRef } from 'react';

const isHeatmapPointVisible = ({ point, hasLevels, currentLevel, zThreshold }) => {
  if (!hasLevels || zThreshold === undefined || zThreshold === null) return true;
  if (point.avg_z === undefined || point.avg_z === null) return false;
  const isUpper = point.avg_z >= zThreshold;
  return currentLevel === 'upper' ? isUpper : !isUpper;
};

const getSideWeight = (point, activeSide) => {
  const total = Number(point.sample_count ?? point.count ?? point.intensity ?? 0);
  if (activeSide === 'all') return total;

  if (activeSide === 'ct') {
    return Number(point.ct_count ?? total * (Number(point.ct_ratio ?? 50) / 100));
  }

  return Number(
    point.t_count
      ?? total * (1 - (Number(point.ct_ratio ?? 50) / 100))
  );
};

const HeatmapCanvas = ({
  points = [],
  intensity = 70,
  visible = true,
  activeSide = 'all',
  hasLevels = false,
  currentLevel = 'upper',
  zThreshold = null,
}) => {
  const canvasRef = useRef(null);
  const workerRef = useRef(null);
  const requestIdRef = useRef(0);
  const resizeFrameRef = useRef(null);

  // Initialize worker once
  useEffect(() => {
    workerRef.current = new Worker(
      new URL('../../workers/heatmap.worker.js', import.meta.url)
    );

    workerRef.current.onmessage = (e) => {
      const { pixels, width, height, requestId } = e.data;
      const canvas = canvasRef.current;
      if (!canvas) return;

      // Discard stale results
      if (requestId !== requestIdRef.current) return;

      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      if (!pixels) {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        return;
      }

      const imageData = new ImageData(
        new Uint8ClampedArray(pixels),
        width,
        height
      );
      // Size canvas to match worker output
      canvas.width = width;
      canvas.height = height;
      ctx.putImageData(imageData, 0, 0);
    };

    return () => {
      workerRef.current?.terminate();
      workerRef.current = null;
    };
  }, []);

  const weightedPoints = useMemo(() => {
    return (points || []).reduce((visiblePoints, point) => {
      if (!isHeatmapPointVisible({ point, hasLevels, currentLevel, zThreshold })) {
        return visiblePoints;
      }

      const weight = getSideWeight(point, activeSide);
      if (weight <= 0) return visiblePoints;

      visiblePoints.push({
        x: point.x,
        y: point.y,
        weight,
      });

      return visiblePoints;
    }, []);
  }, [points, activeSide, hasLevels, currentLevel, zThreshold]);

  // Post work to the worker whenever inputs change
  const scheduleRender = useCallback(() => {
    const canvas = canvasRef.current;
    const worker = workerRef.current;
    if (!canvas || !worker) return;

    const parent = canvas.parentElement;
    if (!parent) return;

    const parentRect = parent.getBoundingClientRect();
    const cssWidth = Math.max(1, Math.floor(parentRect.width));
    const cssHeight = Math.max(1, Math.floor(parentRect.height));
    const deviceRatio = Math.min(1.75, window.devicePixelRatio || 1);
    const outputRatio = Math.min(
      deviceRatio,
      1400 / Math.max(cssWidth, cssHeight)
    );
    const width = Math.max(1, Math.floor(cssWidth * outputRatio));
    const height = Math.max(1, Math.floor(cssHeight * outputRatio));

    // Keep CSS size in sync
    canvas.style.width = `${cssWidth}px`;
    canvas.style.height = `${cssHeight}px`;

    if (!visible || weightedPoints.length === 0) {
      requestIdRef.current += 1;
      const ctx = canvas.getContext('2d');
      if (ctx) {
        canvas.width = width;
        canvas.height = height;
        ctx.clearRect(0, 0, width, height);
      }
      return;
    }

    const requestId = ++requestIdRef.current;

    // Send serializable points (strip any non-cloneable refs)
    worker.postMessage({
      points: weightedPoints,
      width,
      height,
      intensity,
      requestId,
    });
  }, [weightedPoints, visible, intensity]);

  useEffect(() => {
    scheduleRender();
  }, [scheduleRender]);

  useEffect(() => {
    const canvas = canvasRef.current;
    const parent = canvas?.parentElement;
    if (!parent || typeof ResizeObserver === 'undefined') return undefined;

    const observer = new ResizeObserver(() => {
      if (resizeFrameRef.current) cancelAnimationFrame(resizeFrameRef.current);
      resizeFrameRef.current = requestAnimationFrame(scheduleRender);
    });
    observer.observe(parent);

    return () => {
      observer.disconnect();
      if (resizeFrameRef.current) cancelAnimationFrame(resizeFrameRef.current);
    };
  }, [scheduleRender]);

  return (
    <canvas
      ref={canvasRef}
      className="movement-heatmap-canvas"
      role="img"
      aria-label="Distribución espacial del tiempo con vida en el mapa"
    />
  );
};

export default React.memo(HeatmapCanvas);
