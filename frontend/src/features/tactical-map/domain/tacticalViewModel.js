import { formatCalloutName } from '../../../components/TacticalMap/tacticalPresentation';

const CONFIDENCE_LABELS = {
  high: 'alta',
  medium: 'media',
  low: 'inicial',
};

const PRESENTATION_TYPES = {
  impact: 'strength',
  territory: 'habit',
  risk: 'risk',
};

const PRESENTATION_ORDER = {
  strength: 0,
  habit: 1,
  risk: 2,
};

const numberFormatter = new Intl.NumberFormat('es-ES', { maximumFractionDigits: 0 });

const getSideWeight = (position, activeSide) => {
  const samples = Number(position.sample_count ?? 0);
  if (activeSide === 'all') return samples;

  const ctRatio = Number(position.ct_percent ?? position.ct_ratio ?? 50) / 100;
  return activeSide === 'ct' ? samples * ctRatio : samples * (1 - ctRatio);
};

export const getMovementMetricsForSide = (metrics = {}, activeSide = 'all') => {
  const positions = Array.isArray(metrics.top_positions) ? metrics.top_positions : [];
  if (activeSide === 'all') return metrics;

  const weighted = positions.map((position) => ({
    ...position,
    sideSamples: getSideWeight(position, activeSide),
  }));
  const total = weighted.reduce((sum, position) => sum + position.sideSamples, 0);

  return {
    ...metrics,
    total_samples: Math.round(total),
    top_positions: weighted
      .filter((position) => position.sideSamples > 0)
      .map((position) => ({
        ...position,
        time_percent: total > 0 ? (position.sideSamples / total) * 100 : 0,
        sample_count: Math.round(position.sideSamples),
      }))
      .sort((a, b) => b.time_percent - a.time_percent),
  };
};

export const getMovementSideDistribution = (metrics = {}) => {
  const positions = Array.isArray(metrics.top_positions) ? metrics.top_positions : [];
  const totals = positions.reduce((result, position) => {
    const samples = Math.max(0, Number(position.sample_count ?? 0));
    const ctRatio = Math.min(1, Math.max(0, Number(position.ct_percent ?? position.ct_ratio ?? 50) / 100));
    return {
      ct: result.ct + samples * ctRatio,
      t: result.t + samples * (1 - ctRatio),
    };
  }, { ct: 0, t: 0 });
  const total = totals.ct + totals.t;

  return total > 0
    ? { ct: totals.ct / total * 100, t: totals.t / total * 100 }
    : { ct: 50, t: 50 };
};

const findCallout = (callouts, name) => {
  const normalizedName = String(name ?? '').trim().toLocaleLowerCase('es');
  return callouts.find((callout) => (
    String(callout.name ?? '').trim().toLocaleLowerCase('es') === normalizedName
  ));
};

const getRecommendation = (type, provisional) => {
  if (provisional) {
    return 'Repítelo como hipótesis y valida el patrón con más demos antes de cambiar tu plan.';
  }
  if (type === 'strength') {
    return 'Conserva el primer ángulo y el apoyo que ya convierten esta zona en ventaja.';
  }
  if (type === 'habit') {
    return 'Añade una segunda ruta de entrada para no convertir esta presencia en un patrón predecible.';
  }
  return 'Reduce el primer contacto sin apoyo y define una salida antes de volver a disputar la zona.';
};

const getInsightCopy = (insight, type) => {
  const zone = formatCalloutName(insight.calloutName);
  if (insight.status === 'insufficient-data') {
    return {
      title: type === 'strength' ? 'Fortaleza por confirmar' : type === 'habit' ? 'Hábito por descubrir' : 'Riesgo por confirmar',
      zone: 'Sin evidencia suficiente',
      value: 'Sin muestra',
      detail: 'Aún faltan observaciones válidas en este mapa y filtro.',
      recommendation: 'Sigue sumando demos antes de modificar tu forma de jugar.',
    };
  }

  const provisional = insight.status === 'provisional';
  const evidence = insight.evidence ?? {};
  const sample = type === 'habit'
    ? `${numberFormatter.format(evidence.sampleCount ?? 0)} muestras`
    : `${numberFormatter.format(evidence.sampleSize ?? 0)} duelos`;
  const demos = `${numberFormatter.format(evidence.matchesAnalyzed ?? 0)} demos`;

  if (type === 'habit') {
    return {
      title: `Hábito · ${zone}`,
      zone,
      value: `${Math.round(insight.score)}% del tiempo`,
      detail: `${sample} · ${demos}. ${provisional ? 'Patrón inicial' : 'Mayor concentración de presencia con vida'}.`,
      recommendation: getRecommendation(type, provisional),
    };
  }
  if (type === 'strength') {
    return {
      title: `Fortaleza · ${zone}`,
      zone,
      value: `${Math.round(insight.score)}/100 de impacto`,
      detail: `${sample} · ${demos}. ${provisional ? 'Señal provisional' : 'Combina duelos, aperturas y trades'}.`,
      recommendation: getRecommendation(type, provisional),
    };
  }
  return {
    title: `Riesgo · ${zone}`,
    zone,
    value: `${Math.round(insight.score)}/100 de riesgo`,
    detail: `${sample} · ${demos}. ${provisional ? 'Señal provisional' : 'Zona prioritaria para cambiar la decisión'}.`,
    recommendation: getRecommendation(type, provisional),
  };
};

export const buildTacticalPresentation = ({
  tacticalInsights = {},
  movementMetrics = {},
}) => {
  const enrichedCallouts = tacticalInsights.callouts ?? [];
  const topPositions = movementMetrics.top_positions ?? [];

  const insights = (tacticalInsights.insights ?? [])
    .map((insight) => {
      const type = PRESENTATION_TYPES[insight.type] ?? 'habit';
      const copy = getInsightCopy(insight, type);
      return {
        id: `${type}:${insight.calloutName ?? 'pending'}`,
        type,
        zone: copy.zone,
        value: copy.value,
        evidence: {
          ...(insight.evidence ?? {}),
          status: insight.status,
          confidence: insight.confidenceScore,
        },
        confidence: insight.confidenceScore,
        recommendation: copy.recommendation,
        position: null,
        sourceType: insight.type,
        status: insight.status,
        confidenceLabel: CONFIDENCE_LABELS[insight.confidenceLabel] ?? 'inicial',
        title: copy.title,
        name: copy.zone,
        detail: copy.detail,
        action: copy.recommendation,
        calloutName: insight.calloutName,
      };
    })
    .sort((left, right) => PRESENTATION_ORDER[left.type] - PRESENTATION_ORDER[right.type]);

  const signals = insights.reduce((result, insight) => {
    if (insight.status === 'insufficient-data') return result;

    const callout = findCallout(enrichedCallouts, insight.calloutName);
    const habitPosition = topPositions.find(
      (position) => String(position.area) === String(insight.calloutName)
    )?.position;
    const position = insight.type === 'habit'
      ? habitPosition ?? callout?.position
      : callout?.position;

    if (!Number.isFinite(position?.x) || !Number.isFinite(position?.y)) return result;

    result.push({
      ...insight,
      position: { x: position.x, y: position.y },
      x: position.x,
      y: position.y,
      callout,
    });
    return result;
  }, []);

  const readyConfidence = insights
    .filter((insight) => insight.status !== 'insufficient-data')
    .map((insight) => insight.confidence);
  const globalConfidence = readyConfidence.length
    ? Math.round(readyConfidence.reduce((sum, score) => sum + score, 0) / readyConfidence.length)
    : 0;

  return { insights, signals, globalConfidence };
};
