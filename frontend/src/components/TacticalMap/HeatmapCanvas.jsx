import React, { useEffect, useMemo, useRef, useCallback } from 'react';

const isHeatmapPointVisible = ({ point, hasLevels, currentLevel, zThreshold }) => {
  if (!hasLevels || zThreshold === undefined || zThreshold === null) return true;
  if (point.avg_z === undefined || point.avg_z === null) return true;
  const isUpper = point.avg_z >= zThreshold;
  return currentLevel === 'upper' ? isUpper : !isUpper;
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

  const filteredPoints = useMemo(() => {
    return (points || []).filter((point) => {
      if (activeSide !== 'all') {
        const side = point.side?.toLowerCase();
        if (side && side !== activeSide) return false;
      }
      return isHeatmapPointVisible({ point, hasLevels, currentLevel, zThreshold });
    });
  }, [points, activeSide, hasLevels, currentLevel, zThreshold]);

  // Post work to the worker whenever inputs change
  const scheduleRender = useCallback(() => {
    const canvas = canvasRef.current;
    const worker = workerRef.current;
    if (!canvas || !worker) return;

    const parent = canvas.parentElement;
    if (!parent) return;

    const parentRect = parent.getBoundingClientRect();
    const width = Math.max(1, Math.floor(parentRect.width));
    const height = Math.max(1, Math.floor(parentRect.height));

    // Keep CSS size in sync
    canvas.style.width = `${width}px`;
    canvas.style.height = `${height}px`;

    if (!visible || filteredPoints.length === 0) {
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
    const serializablePoints = filteredPoints.map(p => ({
      x: p.x,
      y: p.y,
      weight: p.weight,
      count: p.count,
      samples: p.samples,
      value: p.value,
    }));

    worker.postMessage({
      points: serializablePoints,
      width,
      height,
      intensity,
      requestId,
    });
  }, [filteredPoints, visible, intensity]);

  useEffect(() => {
    scheduleRender();
  }, [scheduleRender]);

  return <canvas ref={canvasRef} className="movement-heatmap-canvas" />;
};

export default React.memo(HeatmapCanvas);
