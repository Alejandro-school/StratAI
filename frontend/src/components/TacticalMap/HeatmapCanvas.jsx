import React, { useEffect, useMemo, useRef } from 'react';

const getPointWeight = (point) => {
  if (typeof point.weight === 'number') return point.weight;
  if (typeof point.count === 'number') return point.count;
  if (typeof point.samples === 'number') return point.samples;
  if (typeof point.value === 'number') return point.value;
  return 1;
};

const isHeatmapPointVisible = ({ point, hasLevels, currentLevel, zThreshold }) => {
  if (!hasLevels || zThreshold === undefined || zThreshold === null) return true;

  if (point.avg_z === undefined || point.avg_z === null) {
    return true;
  }

  const isUpper = point.avg_z >= zThreshold;
  return currentLevel === 'upper' ? isUpper : !isUpper;
};

const createHeatmapPalette = () => {
  const paletteCanvas = document.createElement('canvas');
  paletteCanvas.width = 256;
  paletteCanvas.height = 1;
  const paletteCtx = paletteCanvas.getContext('2d');

  if (!paletteCtx) {
    return new Uint8ClampedArray(256 * 4);
  }

  const gradient = paletteCtx.createLinearGradient(0, 0, 256, 0);
  gradient.addColorStop(0.0, 'rgba(0, 0, 0, 0)');
  gradient.addColorStop(0.15, '#1d4ed8');
  gradient.addColorStop(0.35, '#06b6d4');
  gradient.addColorStop(0.55, '#22c55e');
  gradient.addColorStop(0.72, '#facc15');
  gradient.addColorStop(0.86, '#fb923c');
  gradient.addColorStop(1.0, '#ef4444');

  paletteCtx.fillStyle = gradient;
  paletteCtx.fillRect(0, 0, 256, 1);

  return paletteCtx.getImageData(0, 0, 256, 1).data;
};

const getPercentile = (values, percentile) => {
  if (!values.length) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const index = Math.min(sorted.length - 1, Math.max(0, Math.floor((percentile / 100) * sorted.length)));
  return sorted[index];
};

const getMaxValue = (values, fallback = 1) => {
  if (!values || values.length === 0) return fallback;
  let max = fallback;
  for (let i = 0; i < values.length; i += 1) {
    if (values[i] > max) max = values[i];
  }
  return max;
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

  const filteredPoints = useMemo(() => {
    return (points || []).filter((point) => {
      if (activeSide !== 'all') {
        const side = point.side?.toLowerCase();
        if (side && side !== activeSide) {
          return false;
        }
      }

      return isHeatmapPointVisible({ point, hasLevels, currentLevel, zThreshold });
    });
  }, [points, activeSide, hasLevels, currentLevel, zThreshold]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const context = canvas.getContext('2d', { willReadFrequently: true });
    if (!context) return;

    const parent = canvas.parentElement;
    if (!parent) return;

    const parentRect = parent.getBoundingClientRect();
    const width = Math.max(1, Math.floor(parentRect.width));
    const height = Math.max(1, Math.floor(parentRect.height));

    const dpr = window.devicePixelRatio || 1;
    canvas.width = Math.floor(width * dpr);
    canvas.height = Math.floor(height * dpr);
    canvas.style.width = `${width}px`;
    canvas.style.height = `${height}px`;
    context.setTransform(dpr, 0, 0, dpr, 0, 0);

    context.clearRect(0, 0, width, height);

    if (!visible || filteredPoints.length === 0) {
      return;
    }

    const bufferCanvas = document.createElement('canvas');
    bufferCanvas.width = width;
    bufferCanvas.height = height;
    const bufferCtx = bufferCanvas.getContext('2d', { willReadFrequently: true });
    if (!bufferCtx) return;

    const maxWeight = getMaxValue(filteredPoints.map(getPointWeight), 1);
    const alphaScale = Math.max(0.35, Math.min(1, intensity / 100));
    const radiusScale = Math.max(0.7, Math.min(1.15, 1 - ((intensity - 70) / 300)));

    bufferCtx.clearRect(0, 0, width, height);
    bufferCtx.globalCompositeOperation = 'source-over';

    filteredPoints.forEach((point) => {
      const x = (Math.max(0, Math.min(100, point.x || 0)) / 100) * width;
      const y = (Math.max(0, Math.min(100, point.y || 0)) / 100) * height;

      const normalized = Math.max(0.04, getPointWeight(point) / maxWeight);
      const focus = Math.pow(normalized, 0.7);
      const radius = (10 + (focus * 24)) * radiusScale;
      const alpha = Math.min(0.9, (0.09 + focus * 0.62) * alphaScale);

      const gradient = bufferCtx.createRadialGradient(x, y, 0, x, y, radius);
      gradient.addColorStop(0, `rgba(255, 255, 255, ${alpha})`);
      gradient.addColorStop(0.35, `rgba(255, 255, 255, ${alpha * 0.68})`);
      gradient.addColorStop(1, 'rgba(255, 255, 255, 0)');

      bufferCtx.fillStyle = gradient;
      bufferCtx.beginPath();
      bufferCtx.arc(x, y, radius, 0, Math.PI * 2);
      bufferCtx.fill();
    });

    const imageData = bufferCtx.getImageData(0, 0, width, height);
    const { data } = imageData;
    const alphaValues = [];

    for (let index = 3; index < data.length; index += 4) {
      if (data[index] > 0) alphaValues.push(data[index]);
    }

    if (alphaValues.length === 0) return;

    const palette = createHeatmapPalette();
    const p98 = getPercentile(alphaValues, 98);
    const maxAlpha = getMaxValue(alphaValues, 1);
    const normalizationTop = Math.max(1, Math.min(maxAlpha, p98 || maxAlpha));

    for (let index = 0; index < data.length; index += 4) {
      const alpha = data[index + 3];
      if (alpha === 0) continue;

      const normalized = Math.min(1, alpha / normalizationTop);
      const shaped = Math.pow(normalized, 0.72);

      if (shaped < 0.035) {
        data[index + 3] = 0;
        continue;
      }

      const paletteIndex = Math.min(255, Math.floor(shaped * 255));
      const colorOffset = paletteIndex * 4;

      data[index] = palette[colorOffset];
      data[index + 1] = palette[colorOffset + 1];
      data[index + 2] = palette[colorOffset + 2];
      data[index + 3] = Math.min(255, Math.floor(255 * Math.pow(shaped, 0.92) * alphaScale));
    }

    context.putImageData(imageData, 0, 0);
  }, [filteredPoints, visible, intensity]);

  return <canvas ref={canvasRef} className="movement-heatmap-canvas" />;
};

export default HeatmapCanvas;
