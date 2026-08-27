const DEFAULT_BASELINE = 50;
const BAYES_PRIOR_DUELS = 8;
const OPENING_PRIOR_ATTEMPTS = 4;

const clamp = (value, min = 0, max = 100) => Math.min(max, Math.max(min, value));
const round = (value) => Math.round(value * 10) / 10;
const numberOr = (value, fallback = 0) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};
const nonNegative = (value) => Math.max(0, numberOr(value));
const nameOf = (callout) => String(callout?.name || "");

const getDuelRecord = (callout) => {
  const kills = nonNegative(callout?.kills);
  const deaths = nonNegative(callout?.deaths);
  const countedDuels = kills + deaths;
  const sampleSize = countedDuels || nonNegative(callout?.sample_size);
  const rawWinRate = clamp(numberOr(callout?.win_rate, DEFAULT_BASELINE));
  const wins = countedDuels ? kills : sampleSize * rawWinRate / 100;

  return { callout, wins, sampleSize, rawWinRate };
};

const getConfidence = (sampleSize, matchesAnalyzed) => {
  const duelEvidence = 1 - Math.exp(-nonNegative(sampleSize) / 12);
  const matchEvidence = 1 - Math.exp(-nonNegative(matchesAnalyzed) / 6);
  const score = round(Math.sqrt(duelEvidence * matchEvidence) * 100);
  const label = score >= 75 ? "high" : score >= 45 ? "medium" : "low";
  return { score, label };
};

const getFlashDeathPct = (callout, context, deaths) => {
  const direct = Number(callout?.flash_death_pct ?? context.flash_death_pct);
  if (Number.isFinite(direct)) return clamp(direct);
  if (!deaths) return 0;
  return clamp(nonNegative(context.flash_deaths) / deaths * 100);
};

const getContextScores = (callout, baseline) => {
  const context = callout?.context_stats || {};
  const openingAttempts = nonNegative(context.opening_attempts);
  const openingKills = Math.min(openingAttempts, nonNegative(context.opening_kills));
  const tradeKills = nonNegative(context.trade_kills);
  const tradeDeaths = nonNegative(context.trade_deaths);
  const tradeTotal = tradeKills + tradeDeaths;
  const openingRate = openingAttempts
    ? (openingKills + OPENING_PRIOR_ATTEMPTS * baseline / 100)
      / (openingAttempts + OPENING_PRIOR_ATTEMPTS) * 100
    : baseline;
  const tradeEdge = tradeTotal ? (tradeKills - tradeDeaths) / (tradeTotal + 4) : 0;
  const deaths = nonNegative(callout?.deaths);

  return {
    openingAttempts,
    openingKills,
    openingRate,
    tradeKills,
    tradeDeaths,
    tradeEdge,
    flashDeathPct: getFlashDeathPct(callout, context, deaths),
  };
};

const getImpactScore = ({ adjustedWinRate, baseline, context }) => {
  const duelQuality = clamp(50 + (adjustedWinRate - baseline) * 2);
  const openingQuality = clamp(50 + (context.openingRate - baseline) * 2);
  const tradeQuality = clamp(50 + context.tradeEdge * 50);
  const performance = duelQuality * 0.65 + openingQuality * 0.2 + tradeQuality * 0.15;
  return round(performance);
};

const getRiskScore = ({ adjustedWinRate, baseline, context, evidence, activity }) => {
  const duelRisk = clamp((baseline - adjustedWinRate) / 25 * 100);
  const openingRisk = context.openingAttempts
    ? clamp((baseline - context.openingRate) / 25 * 100)
    : 0;
  const tradeRisk = clamp(-context.tradeEdge * 100);
  const flashRisk = clamp(context.flashDeathPct / 40 * 100);
  const composite = duelRisk * 0.55 + openingRisk * 0.2 + tradeRisk * 0.15 + flashRisk * 0.1;
  const severity = Math.max(composite, duelRisk * 0.75, openingRisk * 0.55, tradeRisk * 0.55, flashRisk * 0.55);
  return round(severity * (0.55 + evidence * 0.45) * (0.7 + activity * 0.3));
};

const enrichCallout = (record, baseline, totalDuels, maxSample, matchesAnalyzed) => {
  const { callout, wins, sampleSize, rawWinRate } = record;
  const adjustedWinRate = (wins + BAYES_PRIOR_DUELS * baseline / 100)
    / (sampleSize + BAYES_PRIOR_DUELS) * 100;
  const confidence = getConfidence(sampleSize, matchesAnalyzed);
  const volumeShare = totalDuels ? sampleSize / totalDuels * 100 : 0;
  const activity = maxSample ? Math.sqrt(sampleSize / maxSample) : 0;
  const context = getContextScores(callout, baseline);
  const scoreInput = {
    adjustedWinRate,
    baseline,
    context,
    evidence: confidence.score / 100,
    activity,
  };
  const impactScore = getImpactScore(scoreInput);
  const riskScore = getRiskScore(scoreInput);

  return {
    ...callout,
    adjustedWinRate: round(adjustedWinRate),
    confidenceScore: confidence.score,
    confidenceLabel: confidence.label,
    volumeShare: round(volumeShare),
    impactScore,
    impactLabel: impactScore >= 65 ? "high" : impactScore >= 45 ? "moderate" : "limited",
    riskScore,
    riskLabel: riskScore >= 65 ? "high" : riskScore >= 35 ? "elevated" : "low",
    tacticalEvidence: {
      rawWinRate: round(rawWinRate),
      sampleSize: round(sampleSize),
      openingAttempts: round(context.openingAttempts),
      tradeKills: round(context.tradeKills),
      tradeDeaths: round(context.tradeDeaths),
      flashDeathPct: round(context.flashDeathPct),
    },
  };
};

const compareByScore = (field) => (left, right) =>
  right[field] - left[field]
  || right.confidenceScore - left.confidenceScore
  || right.volumeShare - left.volumeShare
  || nameOf(left).localeCompare(nameOf(right), "en");

const unavailableInsight = (type, priority) => ({
  type,
  priority,
  status: "insufficient-data",
  calloutName: null,
  score: null,
  confidenceScore: 0,
  confidenceLabel: "low",
  evidence: {},
});

const getTerritoryInsight = (movementMetrics, matchesAnalyzed) => {
  const positions = Array.isArray(movementMetrics?.top_positions)
    ? movementMetrics.top_positions
    : [];
  const ranked = positions
    .filter((position) => String(position?.area || "").trim())
    .sort((left, right) =>
      nonNegative(right.time_percent) - nonNegative(left.time_percent)
      || nonNegative(right.sample_count) - nonNegative(left.sample_count)
      || String(left.area).localeCompare(String(right.area), "en"));
  const territory = ranked[0];
  if (!territory) return unavailableInsight("territory", 1);

  const confidence = getConfidence(territory.sample_count, matchesAnalyzed);
  return {
    type: "territory",
    priority: 1,
    status: confidence.label === "low" ? "provisional" : "ready",
    calloutName: String(territory.area),
    score: round(nonNegative(territory.time_percent)),
    confidenceScore: confidence.score,
    confidenceLabel: confidence.label,
    evidence: {
      timePercent: round(nonNegative(territory.time_percent)),
      sampleCount: round(nonNegative(territory.sample_count)),
      matchesAnalyzed: round(nonNegative(matchesAnalyzed)),
    },
  };
};

const getCalloutInsight = (type, priority, callouts, scoreField, matchesAnalyzed) => {
  if (!callouts.length) return unavailableInsight(type, priority);
  const callout = [...callouts].sort(compareByScore(scoreField))[0];
  return {
    type,
    priority,
    status: callout.confidenceLabel === "low" ? "provisional" : "ready",
    calloutName: nameOf(callout),
    score: callout[scoreField],
    confidenceScore: callout.confidenceScore,
    confidenceLabel: callout.confidenceLabel,
    evidence: {
      ...callout.tacticalEvidence,
      volumeShare: callout.volumeShare,
      matchesAnalyzed: round(nonNegative(matchesAnalyzed)),
    },
  };
};

export const buildTacticalInsights = ({
  callouts = [],
  matchesAnalyzed = 0,
  movementMatchesAnalyzed = matchesAnalyzed,
  movementMetrics = {},
} = {}) => {
  const records = (Array.isArray(callouts) ? callouts : []).map(getDuelRecord);
  const totalDuels = records.reduce((sum, record) => sum + record.sampleSize, 0);
  const totalWins = records.reduce((sum, record) => sum + record.wins, 0);
  const baseline = totalDuels ? totalWins / totalDuels * 100 : DEFAULT_BASELINE;
  const maxSample = Math.max(0, ...records.map((record) => record.sampleSize));
  const enrichedCallouts = records.map((record) =>
    enrichCallout(record, baseline, totalDuels, maxSample, matchesAnalyzed));

  return {
    baselineWinRate: round(baseline),
    totalDuels: round(totalDuels),
    callouts: enrichedCallouts,
    insights: [
      getTerritoryInsight(movementMetrics, movementMatchesAnalyzed),
      getCalloutInsight("impact", 2, enrichedCallouts, "impactScore", matchesAnalyzed),
      getCalloutInsight("risk", 3, enrichedCallouts, "riskScore", matchesAnalyzed),
    ],
  };
};
