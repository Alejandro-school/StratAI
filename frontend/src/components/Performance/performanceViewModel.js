import { getQualityLabel } from '../../utils/performanceBenchmarks';
import { formatDecimal, formatInteger, formatMapName, formatPercent } from '../../utils/performanceFormatters';

const number = (value, fallback = 0) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const clamp = (value, min = 0, max = 100) => Math.min(Math.max(number(value), min), max);

const scoreFromMetric = (value, max, inverse = false) => {
  const score = clamp((number(value) / max) * 100);
  return inverse ? 100 - score : score;
};

const getForm = (history = []) => {
  const recent = history.slice(0, 10);
  const wins = recent.filter((match) => match.result === 'W').length;
  const losses = recent.length - wins;
  const winRate = recent.length > 0 ? (wins / recent.length) * 100 : 0;
  const label = recent.length === 0
    ? 'Sin muestra reciente'
    : winRate >= 65
      ? 'Racha fuerte'
      : winRate >= 50
        ? 'Forma estable'
        : 'Necesita ajuste';

  return { recent, wins, losses, winRate, label };
};

const getBestMap = (maps = []) => {
  if (!maps.length) return null;

  return [...maps].sort((left, right) => {
    const leftScore = number(left.avg_rating) * 80 + number(left.win_rate) + number(left.matches) * 2;
    const rightScore = number(right.avg_rating) * 80 + number(right.win_rate) + number(right.matches) * 2;
    return rightScore - leftScore;
  })[0];
};

const getAreaScores = ({ overview = {}, aim = {}, combat = {}, utility = {}, maps = [], weapons = [] }) => {
  const bestMap = getBestMap(maps);
  const topWeapon = weapons[0] || {};

  return [
    {
      id: 'combat',
      label: 'Combate',
      score: Math.round((
        scoreFromMetric(overview.impact_rating, 1.6)
        + scoreFromMetric(combat.opening_success_rate, 70)
        + scoreFromMetric(overview.kast, 85)
      ) / 3),
      primary: formatDecimal(overview.impact_rating, 2),
      detail: 'Impacto, aperturas y KAST',
    },
    {
      id: 'aim',
      label: 'Punteria',
      score: Math.round((
        scoreFromMetric(aim.accuracy_overall, 40)
        + scoreFromMetric(overview.hs_pct, 60)
        + scoreFromMetric(aim.time_to_damage_avg_ms, 650, true)
      ) / 3),
      primary: formatPercent(aim.accuracy_overall),
      detail: 'Precision, HS y tiempo al dano',
    },
    {
      id: 'weapons',
      label: 'Arsenal',
      score: Math.round((
        scoreFromMetric(topWeapon.kills, Math.max(number(overview.kills), 1))
        + scoreFromMetric(topWeapon.accuracy, 40)
        + scoreFromMetric(topWeapon.hs_pct, 60)
      ) / 3),
      primary: topWeapon.weapon ? formatInteger(topWeapon.kills) : '0',
      detail: topWeapon.weapon ? `Dominante: ${topWeapon.weapon}` : 'Sin arma dominante',
    },
    {
      id: 'maps',
      label: 'Mapas',
      score: Math.round((
        scoreFromMetric(bestMap?.win_rate, 70)
        + scoreFromMetric(bestMap?.avg_rating, 1.5)
        + scoreFromMetric(bestMap?.matches, Math.max(number(overview.total_matches), 1))
      ) / 3),
      primary: bestMap ? formatMapName(bestMap.map) : 'Sin muestra',
      detail: bestMap ? `${formatPercent(bestMap.win_rate)} WR, ${formatDecimal(bestMap.avg_rating, 2)} rating` : 'Falta historial por mapa',
    },
    {
      id: 'utility',
      label: 'Utilidad',
      score: Math.round((
        scoreFromMetric(utility.enemies_flashed_per_flash, 1.8)
        + scoreFromMetric(utility.he_damage_per_nade, 35)
        + scoreFromMetric(utility.molotov_damage_per_nade, 25)
      ) / 3),
      primary: formatDecimal(utility.enemies_flashed_per_flash, 2),
      detail: 'Cegadoras y dano de granadas',
    },
  ].map((area) => ({ ...area, score: clamp(area.score) }));
};

const getStrengthsAndWeaknesses = (areas = []) => {
  const ranked = [...areas].sort((left, right) => right.score - left.score);
  return {
    strengths: ranked.slice(0, 2),
    weaknesses: ranked.slice(-2).reverse(),
    bestArea: ranked[0],
    focusArea: ranked[ranked.length - 1],
  };
};

export const buildPerformanceViewModel = ({
  overview = {},
  sides = {},
  aim = {},
  combat = {},
  utility = {},
  weapons = [],
  maps = [],
  history = [],
  trends = {},
}) => {
  const rating = number(overview.hltv_rating);
  const ratingBadge = getQualityLabel(rating, 'hltv_rating');
  const form = getForm(history);
  const bestMap = getBestMap(maps);
  const areas = getAreaScores({ overview, aim, combat, utility, maps, weapons });
  const { strengths, weaknesses, bestArea, focusArea } = getStrengthsAndWeaknesses(areas);
  const sideGap = number(sides.ct_rating) - number(sides.t_rating);

  return {
    rating,
    ratingBadge,
    ratingAngle: `${clamp((rating / 2) * 360, 0, 360)}deg`,
    form,
    bestMap,
    areas,
    strengths,
    weaknesses,
    bestArea,
    focusArea,
    trends,
    verdict: rating >= 1.15
      ? 'Rendimiento con ventaja clara'
      : rating >= 1
        ? 'Base competitiva solida'
        : 'Briefing de ajuste activo',
    summary: `${formatInteger(overview.total_matches)} partidas analizadas, ${formatPercent(overview.win_rate)} de victorias y ${formatDecimal(overview.adr, 1)} ADR.`,
    sideRead: Math.abs(sideGap) < 0.03
      ? 'Tu rendimiento por bando esta equilibrado.'
      : sideGap > 0
        ? `Mejor lectura CT por ${formatDecimal(Math.abs(sideGap), 2)} puntos de rating.`
        : `Mejor lectura T por ${formatDecimal(Math.abs(sideGap), 2)} puntos de rating.`,
  };
};

export { clamp, number };
