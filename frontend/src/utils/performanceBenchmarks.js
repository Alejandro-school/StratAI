export const PERFORMANCE_BENCHMARKS = {
  hltv_rating: { poor: 0.8, average: 1.0, good: 1.2 },
  kd_ratio: { poor: 0.85, average: 1.0, good: 1.2 },
  impact_rating: { poor: 0.8, average: 1.0, good: 1.2 },
  win_rate: { poor: 40, average: 50, good: 60 },
  kast: { poor: 60, average: 70, good: 80 },
  adr: { poor: 55, average: 75, good: 90 },
  hs_pct: { poor: 30, average: 45, good: 55 },
  accuracy: { poor: 15, average: 25, good: 35 },
  opening_success: { poor: 40, average: 50, good: 60 },
  he_damage_per_nade: { poor: 6, average: 12, good: 18 },
  molotov_damage_per_nade: { poor: 10, average: 20, good: 30 },
  ttd_ms: { poor: 650, average: 450, good: 320 },
  crosshair_error: { poor: 8, average: 4, good: 2 },
  enemies_per_flash: { poor: 0.5, average: 1.0, good: 1.5 },
};

export const getQualityLabel = (value, metric, reverse = false) => {
  const thresholds = PERFORMANCE_BENCHMARKS[metric];
  if (!thresholds || typeof value !== 'number') return { label: 'N/A', tone: 'neutral' };

  if (!reverse) {
    if (value >= thresholds.good) return { label: 'Excelente', tone: 'excellent' };
    if (value >= thresholds.average) return { label: 'Bueno', tone: 'good' };
    if (value >= thresholds.poor) return { label: 'Promedio', tone: 'average' };
    return { label: 'Mejorable', tone: 'poor' };
  }

  if (value <= thresholds.good) return { label: 'Excelente', tone: 'excellent' };
  if (value <= thresholds.average) return { label: 'Bueno', tone: 'good' };
  if (value <= thresholds.poor) return { label: 'Promedio', tone: 'average' };
  return { label: 'Mejorable', tone: 'poor' };
};
