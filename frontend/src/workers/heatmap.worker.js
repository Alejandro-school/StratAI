// frontend/src/workers/heatmap.worker.js
// Off-main-thread heatmap pixel computation.
// Receives filtered points + canvas dimensions, returns colored ImageData.

/* eslint-disable no-restricted-globals */

const getPointWeight = (point) => {
  if (typeof point.weight === 'number') return point.weight;
  if (typeof point.sample_count === 'number') return point.sample_count;
  if (typeof point.intensity === 'number') return point.intensity;
  if (typeof point.count === 'number') return point.count;
  if (typeof point.samples === 'number') return point.samples;
  if (typeof point.value === 'number') return point.value;
  return 1;
};

const getMaxValue = (values, fallback = 1) => {
  let max = fallback;
  for (let i = 0; i < values.length; i++) {
    if (values[i] > max) max = values[i];
  }
  return max;
};

/**
 * Build the 256-color heatmap palette as an array of [r,g,b,a] x 256
 */
const buildPalette = () => {
  // We can't use Canvas in a worker (no DOM), so build the gradient mathematically.
  // Presence palette: cool territory tones with a warm peak.
  // It deliberately avoids the green/red success/failure language used elsewhere.
  const stops = [
    { pos: 0.00, r: 0,   g: 0,   b: 0   },
    { pos: 0.14, r: 6,   g: 62,  b: 86  },
    { pos: 0.34, r: 14,  g: 116, b: 144 },
    { pos: 0.58, r: 34,  g: 211, b: 238 },
    { pos: 0.78, r: 103, g: 232, b: 249 },
    { pos: 0.92, r: 251, g: 191, b: 36  },
    { pos: 1.00, r: 255, g: 243, b: 196 },
  ];

  const palette = new Uint8ClampedArray(256 * 4);

  for (let i = 0; i < 256; i++) {
    const t = i / 255;
    // Find the two surrounding stops
    let lower = stops[0];
    let upper = stops[stops.length - 1];
    for (let s = 0; s < stops.length - 1; s++) {
      if (t >= stops[s].pos && t <= stops[s + 1].pos) {
        lower = stops[s];
        upper = stops[s + 1];
        break;
      }
    }
    const range = upper.pos - lower.pos || 1;
    const f = (t - lower.pos) / range;
    palette[i * 4]     = Math.round(lower.r + (upper.r - lower.r) * f);
    palette[i * 4 + 1] = Math.round(lower.g + (upper.g - lower.g) * f);
    palette[i * 4 + 2] = Math.round(lower.b + (upper.b - lower.b) * f);
    palette[i * 4 + 3] = 255;
  }

  return palette;
};

const PALETTE = buildPalette();

/**
 * Main computation: take points and render them into a raw RGBA buffer.
 */
const computeHeatmap = ({ points, width, height, intensity }) => {
  if (!points.length || width < 1 || height < 1) return null;

  const alphaScale = Math.max(0.35, Math.min(1, intensity / 100));
  const radiusScale = Math.max(0.7, Math.min(1.15, 1 - ((intensity - 70) / 300)));
  const maxWeight = getMaxValue(points.map(getPointWeight), 1);

  // Build an alpha accumulation buffer (float32 for accuracy)
  const alphaBuffer = new Float32Array(width * height);

  for (let i = 0; i < points.length; i++) {
    const point = points[i];
    const cx = (Math.max(0, Math.min(100, point.x || 0)) / 100) * width;
    const cy = (Math.max(0, Math.min(100, point.y || 0)) / 100) * height;

    const normalized = Math.max(0.04, getPointWeight(point) / maxWeight);
    const focus = Math.pow(normalized, 0.7);
    const radius = (10 + focus * 24) * radiusScale;
    const alpha = Math.min(0.9, (0.09 + focus * 0.62) * alphaScale);

    // Rasterize radial gradient into alphaBuffer
    const r = Math.ceil(radius);
    const x0 = Math.max(0, Math.floor(cx - r));
    const y0 = Math.max(0, Math.floor(cy - r));
    const x1 = Math.min(width - 1, Math.ceil(cx + r));
    const y1 = Math.min(height - 1, Math.ceil(cy + r));
    const rSq = radius * radius;

    for (let y = y0; y <= y1; y++) {
      for (let x = x0; x <= x1; x++) {
        const dx = x - cx;
        const dy = y - cy;
        const distSq = dx * dx + dy * dy;
        if (distSq >= rSq) continue;

        const dist = Math.sqrt(distSq);
        const t = dist / radius;
        // Approximate the 3-stop gradient: full at center, 0.68 at 0.35, 0 at 1.0
        let a;
        if (t <= 0.35) {
          a = alpha * (1 - t / 0.35 * 0.32);
        } else {
          a = alpha * 0.68 * (1 - (t - 0.35) / 0.65);
        }
        if (a <= 0) continue;

        // Additive blend (mimics 'source-over' on white-alpha gradients)
        const idx = y * width + x;
        alphaBuffer[idx] = Math.min(1, alphaBuffer[idx] + a);
      }
    }
  }

  // A 256-bin histogram avoids allocating and sorting millions of JS numbers.
  const alphaHistogram = new Uint32Array(256);
  let nonZeroPixels = 0;
  let maxAlpha = 1;
  for (let i = 0; i < alphaBuffer.length; i++) {
    if (alphaBuffer[i] <= 0) continue;
    const alpha = Math.round(alphaBuffer[i] * 255);
    alphaHistogram[alpha] += 1;
    nonZeroPixels += 1;
    if (alpha > maxAlpha) maxAlpha = alpha;
  }
  if (nonZeroPixels === 0) return null;

  const percentileTarget = Math.ceil(nonZeroPixels * 0.98);
  let cumulativePixels = 0;
  let p98 = maxAlpha;
  for (let alpha = 1; alpha < alphaHistogram.length; alpha++) {
    cumulativePixels += alphaHistogram[alpha];
    if (cumulativePixels >= percentileTarget) {
      p98 = alpha;
      break;
    }
  }
  const normalizationTop = Math.max(1, Math.min(maxAlpha, p98 || maxAlpha));

  // Build final RGBA pixel buffer
  const pixels = new Uint8ClampedArray(width * height * 4);

  for (let i = 0; i < alphaBuffer.length; i++) {
    const rawAlpha = Math.round(alphaBuffer[i] * 255);
    if (rawAlpha === 0) continue;

    const normalized = Math.min(1, rawAlpha / normalizationTop);
    const shaped = Math.pow(normalized, 0.72);

    if (shaped < 0.035) continue;

    const paletteIndex = Math.min(255, Math.floor(shaped * 255));
    const co = paletteIndex * 4;
    const po = i * 4;

    pixels[po]     = PALETTE[co];
    pixels[po + 1] = PALETTE[co + 1];
    pixels[po + 2] = PALETTE[co + 2];
    pixels[po + 3] = Math.min(255, Math.floor(255 * Math.pow(shaped, 0.92) * alphaScale));
  }

  return pixels;
};

self.onmessage = function (e) {
  const { points, width, height, intensity, requestId } = e.data;
  const pixels = computeHeatmap({ points, width, height, intensity });

  if (pixels) {
    self.postMessage(
      { pixels: pixels.buffer, width, height, requestId },
      [pixels.buffer] // transfer ownership — zero-copy
    );
  } else {
    self.postMessage({ pixels: null, width, height, requestId });
  }
};
