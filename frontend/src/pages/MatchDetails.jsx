import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import axios from 'axios';
import {
  ArrowLeft,
  Clock,
  Crosshair,
  Play,
  Shield,
  Sparkles,
  Sword,
  Trophy,
  Users,
  Zap,
} from 'lucide-react';
import {
  Cell,
  Pie,
  PieChart,
  PolarAngleAxis,
  PolarGrid,
  Radar,
  RadarChart,
  ResponsiveContainer,
  Tooltip,
} from 'recharts';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { MetricCell } from '../components/Performance/StatPill';
import Replay2DViewer from '../components/Stats/Replay2DViewer';
import { API_URL } from '../utils/api';
import { getMapImage } from '../utils/mapConfig';
import { PERFORMANCE_BENCHMARKS, getQualityLabel } from '../utils/performanceBenchmarks';
import {
  formatDecimal,
  formatDegrees,
  formatInteger,
  formatMapName,
  formatMilliseconds,
  formatPercent,
  formatWeaponName,
  getWeaponIconPath,
  RECHARTS_TOOLTIP_STYLE,
  TOOLTIP_ITEM_STYLE,
  TOOLTIP_LABEL_STYLE,
} from '../utils/performanceFormatters';
import '../styles/pages/performance.css';
import '../styles/Match/matchDetails.css';

const WEAPON_SORT_OPTIONS = [
  { key: 'kills', label: 'Bajas' },
  { key: 'accuracy', label: 'Precisión' },
  { key: 'hs_pct', label: 'HS %' },
  { key: 'damage', label: 'Daño' },
];

const TOP_BADGES = ['#1', '#2', '#3'];

const MULTIKILL_DEFS = [
  { key: '2k', label: '2K', className: 'mk-2k' },
  { key: '3k', label: '3K', className: 'mk-3k' },
  { key: '4k', label: '4K', className: 'mk-4k' },
  { key: '5k', label: 'ACE', className: 'mk-ace' },
];

const CLUTCH_DEFS = [
  { key: 'clutches_1v1_won', label: '1v1' },
  { key: 'clutches_1v2_won', label: '1v2' },
  { key: 'clutches_1v3_won', label: '1v3' },
  { key: 'clutches_1v4_won', label: '1v4' },
  { key: 'clutches_1v5_won', label: '1v5' },
];

const COMPARE_METRIC_GROUPS = [
  {
    id: 'output',
    label: 'Producción',
    tone: 'purple',
    metrics: [
      { key: 'rating', label: 'Valoración', formatter: (value) => formatDecimal(value, 2), benchmark: 'hltv_rating' },
      { key: 'impactRating', label: 'Impacto', formatter: (value) => formatDecimal(value, 2), benchmark: 'impact_rating' },
      { key: 'adr', label: 'ADR', formatter: (value) => formatInteger(Math.round(value)), benchmark: 'adr' },
      { key: 'kast', label: 'KAST', formatter: (value) => formatPercent(value, 0), scaleFloor: 100, benchmark: 'kast' },
      { key: 'kdRatio', label: 'K/D', formatter: (value) => formatDecimal(value, 2), benchmark: 'kd_ratio' },
      { key: 'survivalRate', label: 'Supervivencia', formatter: (value) => formatPercent(value, 0), scaleFloor: 100 },
    ],
  },
  {
    id: 'mechanics',
    label: 'Puntería y mecánica',
    tone: 'blue',
    metrics: [
      { key: 'accuracy', label: 'Precisión', formatter: (value) => formatPercent(value, 1), benchmark: 'accuracy', scaleFloor: 40 },
      { key: 'hsPct', label: 'HS %', formatter: (value) => formatPercent(value, 0), benchmark: 'hs_pct', scaleFloor: 100 },
      { key: 'timeToDamage', label: 'TTD', formatter: (value) => formatMilliseconds(value), benchmark: 'ttd_ms', reverse: true },
      { key: 'reactionTime', label: 'Reacción', formatter: (value) => formatMilliseconds(value), reverse: true, alwaysShow: true, scaleFloor: 700 },
      { key: 'crosshairError', label: 'Mira', formatter: (value) => formatDegrees(value, 1), benchmark: 'crosshair_error', reverse: true },
      { key: 'crosshairPeek', label: 'Mira al asomar', formatter: (value) => formatDegrees(value, 1), benchmark: 'crosshair_error', reverse: true },
      { key: 'crosshairHold', label: 'Mira al mantener', formatter: (value) => formatDegrees(value, 1), benchmark: 'crosshair_error', reverse: true },
      { key: 'counterStrafeRating', label: 'Frenado', formatter: (value) => formatPercent(value, 0), scaleFloor: 100, alwaysShow: true },
    ],
  },
  {
    id: 'duels',
    label: 'Duelos y clutch',
    tone: 'orange',
    metrics: [
      { key: 'openingAttempted', label: 'Aperturas jugadas', formatter: (value) => formatInteger(Math.round(value)), useRoundsMax: true },
      { key: 'openingWon', label: 'Aperturas ganadas', formatter: (value) => formatInteger(Math.round(value)), useRoundsMax: true },
      { key: 'openingSuccess', label: '% de aperturas', formatter: (value) => formatPercent(value, 0), benchmark: 'opening_success', scaleFloor: 100 },
      { key: 'tradeKills', label: 'Bajas de intercambio', formatter: (value) => formatInteger(Math.round(value)), useRoundsMax: true },
      { key: 'tradedDeaths', label: 'Muertes intercambiadas', formatter: (value) => formatInteger(Math.round(value)), reverse: true, useRoundsMax: true },
      { key: 'totalClutchesWon', label: 'Clutches ganados', formatter: (value) => formatInteger(Math.round(value)), useRoundsMax: true },
      { key: 'clutchConversion', label: 'Conversión clutch', formatter: (value) => formatPercent(value, 0), scaleFloor: 100 },
    ],
  },
  {
    id: 'utility',
    label: 'Utilidad',
    tone: 'green',
    metrics: [
      { key: 'utilityDamage', label: 'Daño útil', formatter: (value) => formatInteger(Math.round(value)), scaleFloor: 250 },
      { key: 'flashAssists', label: 'Asistencias con cegadora', formatter: (value) => formatInteger(Math.round(value)), useRoundsMax: true },
      { key: 'enemiesPerFlash', label: 'Rivales/cegadora', formatter: (value) => formatDecimal(value, 2), benchmark: 'enemies_per_flash', scaleFloor: 2 },
      { key: 'blindTimePerFlash', label: 'Seg cegados/ceg.', formatter: (value) => formatDecimal(value, 1, ' s'), scaleFloor: 4 },
      { key: 'heDamagePerNade', label: 'Daño HE/nade', formatter: (value) => formatDecimal(value, 1), benchmark: 'he_damage_per_nade', scaleFloor: 24 },
      { key: 'molotovDamagePerNade', label: 'Daño molo/nade', formatter: (value) => formatDecimal(value, 1), benchmark: 'molotov_damage_per_nade', scaleFloor: 30 },
    ],
  },
];

const toNumber = (value, fallback = 0) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const clamp = (value, min = 0, max = 100) => Math.max(min, Math.min(max, value));

const hasNumericValue = (value) => Number.isFinite(Number(value));

const toNullableNumber = (value) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

const formatDuration = (seconds) => {
  if (!seconds) return '0:00';
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};

const getKDColor = (kd) => {
  if (kd >= 1.5) return 'var(--p-green)';
  if (kd >= 1.0) return 'var(--p-blue)';
  if (kd >= 0.8) return 'var(--p-yellow)';
  return 'var(--p-red)';
};

const getRatingColor = (rating) => {
  if (rating >= 1.3) return 'var(--p-green)';
  if (rating >= 1.0) return 'var(--p-blue)';
  if (rating >= 0.8) return 'var(--p-yellow)';
  return 'var(--p-red)';
};

const zonePercent = (value, total) => (total > 0 ? (value / total) * 100 : 0);

const normalizeMetric = (value, metric, reverse = false) => {
  const numeric = toNumber(value);
  const benchmark = PERFORMANCE_BENCHMARKS[metric];

  if (!benchmark) return 0;

  const { poor, average, good } = benchmark;

  if (!reverse) {
    if (numeric >= good) return 100;
    if (numeric >= average) {
      const range = good - average || 1;
      return clamp(65 + ((numeric - average) / range) * 35);
    }
    if (numeric >= poor) {
      const range = average - poor || 1;
      return clamp(35 + ((numeric - poor) / range) * 30);
    }

    if (poor > 0) {
      return clamp((numeric / poor) * 35, 8, 35);
    }

    return 0;
  }

  if (numeric <= good) return 100;
  if (numeric <= average) {
    const range = average - good || 1;
    return clamp(65 + ((average - numeric) / range) * 35);
  }
  if (numeric <= poor) {
    const range = poor - average || 1;
    return clamp(35 + ((poor - numeric) / range) * 30);
  }

  if (numeric > 0) {
    return clamp((poor / numeric) * 35, 8, 35);
  }

  return 0;
};

const buildWeaponEntries = (weaponStats) => {
  if (!weaponStats) return [];

  if (Array.isArray(weaponStats)) {
    return weaponStats.map((weapon) => {
      const kills = toNumber(weapon.kills);
      const shots = toNumber(weapon.shots);
      const hits = toNumber(weapon.hits);
      const headshots = toNumber(weapon.headshots);
      const damage = toNumber(weapon.damage);
      const accuracy = shots > 0 ? (hits / shots) * 100 : toNumber(weapon.accuracy);
      const hsPct = kills > 0 ? (headshots / kills) * 100 : toNumber(weapon.hs_pct);

      return {
        ...weapon,
        weapon: weapon.weapon,
        kills,
        shots,
        hits,
        damage,
        headshots,
        accuracy,
        hs_pct: hsPct,
      };
    });
  }

  if (typeof weaponStats === 'object') {
    return Object.entries(weaponStats).map(([weaponName, stats]) => {
      const kills = toNumber(stats?.kills);
      const shots = toNumber(stats?.shots);
      const hits = toNumber(stats?.hits);
      const headshots = toNumber(stats?.headshots);
      const damage = toNumber(stats?.damage);
      const accuracy = shots > 0 ? (hits / shots) * 100 : toNumber(stats?.accuracy);
      const hsPct = kills > 0 ? (headshots / kills) * 100 : toNumber(stats?.hs_pct);

      return {
        ...stats,
        weapon: weaponName,
        kills,
        shots,
        hits,
        damage,
        headshots,
        accuracy,
        hs_pct: hsPct,
      };
    });
  }

  return [];
};

const getBadgeForMetric = (value, metric, reverse = false, hasSample = true) => {
  if (!hasSample) return { label: 'Sin muestra', tone: 'neutral' };
  return getQualityLabel(toNumber(value), metric, reverse);
};

const formatCompareMetricValue = (metric, value) => {
  if (!hasNumericValue(value)) return 'N/D';
  return metric.formatter(Number(value));
};

const getCompareMetricLeader = (metric, currentValue, compareValue) => {
  const hasCurrent = hasNumericValue(currentValue);
  const hasCompare = hasNumericValue(compareValue);

  if (!hasCurrent && !hasCompare) return 'no-data';
  if (hasCurrent && !hasCompare) return 'current';
  if (!hasCurrent && hasCompare) return 'compare';

  const left = Number(currentValue);
  const right = Number(compareValue);

  if (Math.abs(left - right) < 0.0001) return 'tie';
  if (metric.reverse) return left < right ? 'current' : 'compare';
  return left > right ? 'current' : 'compare';
};

const getCompareMetricScaleMax = (metric, currentValue, compareValue, roundsTotal) => {
  const values = [currentValue, compareValue]
    .filter((value) => hasNumericValue(value))
    .map((value) => Number(value));

  const benchmark = metric.benchmark ? PERFORMANCE_BENCHMARKS[metric.benchmark] : null;
  const benchmarkMax = benchmark
    ? Math.max(toNumber(benchmark.poor), toNumber(benchmark.average), toNumber(benchmark.good))
    : 0;
  const roundsScale = metric.useRoundsMax ? Math.max(roundsTotal, 1) : 0;

  return Math.max(...values, benchmarkMax, metric.scaleFloor ?? 0, roundsScale, 1);
};

const getCompareMetricWidth = (metric, value, scaleMax) => {
  if (!hasNumericValue(value) || scaleMax <= 0) return 0;

  const numericValue = Number(value);

  if (metric.reverse) {
    const bounded = Math.min(numericValue, scaleMax);
    return clamp(((scaleMax - bounded) / scaleMax) * 100, 0, 100);
  }

  return clamp((numericValue / scaleMax) * 100, 0, 100);
};

const getCompareMetricGapScore = (currentWidth, compareWidth) => Math.abs(currentWidth - compareWidth);

const buildPlayerProfile = (player, roundsTotal) => {
  if (!player) return null;

  const kills = toNumber(player.kills);
  const deaths = toNumber(player.deaths);
  const assists = toNumber(player.assists);
  const rating = toNumber(player.hltv_rating);
  const adr = toNumber(player.adr);
  const kast = toNumber(player.kast);
  const hsPct = toNumber(player.hs_percentage);
  const impactRating = toNumber(player.impact_rating);
  const roundsSurvived = toNumber(player.rounds_survived);
  const survivalRate = roundsTotal > 0 ? (roundsSurvived / roundsTotal) * 100 : 0;
  const openingWon = toNumber(player.opening_duels_won);
  const openingAttempted = toNumber(player.opening_duels_attempted);
  const openingSuccess = openingAttempted > 0 ? (openingWon / openingAttempted) * 100 : 0;
  const tradeKills = toNumber(player.trade_kills);
  const tradedDeaths = toNumber(player.traded_deaths);
  const utilityDamage = toNumber(player.utility_damage);
  const flashAssists = toNumber(player.flash_assists);
  const enemiesPerFlash = toNumber(player.enemies_flashed_per_flash);
  const blindTimePerFlash = toNullableNumber(player.blind_time_per_flash);
  const timeToDamage = toNullableNumber(player.time_to_damage_avg_ms);
  const reactionTime = toNullableNumber(
    player.avg_time_to_reaction
      ?? player.reaction_time_avg_ms
      ?? player.avg_reaction_time_ms
      ?? player.reaction_time_ms,
  );
  const crosshairError = toNullableNumber(player.crosshair_placement_avg_error);
  const crosshairPeek = toNullableNumber(player.crosshair_placement_peek);
  const crosshairHold = toNullableNumber(player.crosshair_placement_hold);
  const counterStrafeRating = toNullableNumber(
    player.avg_counter_strafe_rating
      ?? player.counter_strafe_rating
      ?? player.counter_strafing_rating,
  );
  const accuracy = toNullableNumber(player.accuracy_overall);
  const heDamagePerNade = toNullableNumber(player.he_damage_per_nade);
  const molotovDamagePerNade = toNullableNumber(player.molotov_damage_per_nade);
  const kdRatio = deaths > 0 ? kills / deaths : kills;
  const openingLost = Math.max(openingAttempted - openingWon, 0);
  const totalClutchesWon = [
    'clutches_1v1_won',
    'clutches_1v2_won',
    'clutches_1v3_won',
    'clutches_1v4_won',
    'clutches_1v5_won',
  ].reduce((sum, key) => sum + toNumber(player[key]), 0);
  const totalClutchesAttempted = [
    'clutches_1v1_attempted',
    'clutches_1v2_attempted',
    'clutches_1v3_attempted',
    'clutches_1v4_attempted',
    'clutches_1v5_attempted',
  ].reduce((sum, key) => sum + toNumber(player[key]), 0);
  const clutchConversion = totalClutchesAttempted > 0 ? (totalClutchesWon / totalClutchesAttempted) * 100 : 0;
  const weapons = buildWeaponEntries(player.weapon_stats)
    .filter((weapon) => weapon.kills > 0 || weapon.damage > 0 || weapon.shots > 0)
    .sort((left, right) => right.kills - left.kills);

  return {
    raw: player,
    steamId: String(player.steam_id ?? ''),
    name: player.name || 'Jugador',
    team: player.team || 'N/D',
    kills,
    deaths,
    assists,
    rating,
    adr,
    kast,
    hsPct,
    impactRating,
    roundsSurvived,
    survivalRate,
    openingWon,
    openingAttempted,
    openingSuccess,
    tradeKills,
    tradedDeaths,
    utilityDamage,
    flashAssists,
    enemiesPerFlash,
    blindTimePerFlash,
    timeToDamage,
    reactionTime,
    crosshairError,
    crosshairPeek,
    crosshairHold,
    counterStrafeRating,
    accuracy,
    heDamagePerNade,
    molotovDamagePerNade,
    kdRatio,
    openingLost,
    totalClutchesWon,
    totalClutchesAttempted,
    clutchConversion,
    weapons,
    topWeapon: weapons[0] ?? null,
  };
};

const MatchDetails = () => {
  const { matchID } = useParams();
  const navigate = useNavigate();

  const [matchData, setMatchData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [activeTab, setActiveTab] = useState('scoreboard');
  const [comparePlayerId, setComparePlayerId] = useState('');
  const [weaponSortKey, setWeaponSortKey] = useState('kills');
  const [showWeaponTable, setShowWeaponTable] = useState(false);

  useEffect(() => {
    const fetchMatchDetails = async () => {
      if (!matchID) return;

      try {
        setLoading(true);

        const response = await axios.get(
          `${API_URL}/steam/get-match-details/${matchID}`,
          { withCredentials: true },
        );

        setMatchData(response.data);
        setError(null);
      } catch (requestError) {
        console.error('Error fetching match details:', requestError);
        setError('No se pudieron cargar los detalles de la partida.');
      } finally {
        setLoading(false);
      }
    };

    fetchMatchDetails();
  }, [matchID]);

  const metadata = matchData?.metadata ?? {};
  const result = matchData?.result ?? 'defeat';
  const isVictory = result === 'victory';

  const allPlayers = useMemo(
    () => [...(matchData?.team_ct ?? []), ...(matchData?.team_t ?? [])],
    [matchData],
  );

  const scoreboardPlayer = useMemo(
    () => allPlayers.find((player) => String(player.steam_id) === String(matchData?.current_user_steam_id)),
    [allPlayers, matchData],
  );

  const userStats = useMemo(
    () => ({ ...(scoreboardPlayer ?? {}), ...(matchData?.current_user ?? {}) }),
    [matchData, scoreboardPlayer],
  );

  const mapDisplayName = useMemo(() => formatMapName(metadata.map_name), [metadata.map_name]);

  const userTeam = userStats?.team;
  const myTeam = userTeam === 'CT' ? matchData?.team_ct ?? [] : matchData?.team_t ?? [];
  const enemyTeam = userTeam === 'CT' ? matchData?.team_t ?? [] : matchData?.team_ct ?? [];

  const roundsTotal = toNumber(metadata.total_rounds);
  const kills = toNumber(userStats.kills);
  const deaths = toNumber(userStats.deaths);
  const assists = toNumber(userStats.assists);
  const kdRatio = deaths > 0 ? kills / deaths : kills;
  const rating = toNumber(userStats.hltv_rating);
  const impactRating = toNumber(userStats.impact_rating);
  const adr = toNumber(userStats.adr);
  const kast = toNumber(userStats.kast);
  const roundsSurvived = toNumber(userStats.rounds_survived);
  const survivalRate = roundsTotal > 0 ? (roundsSurvived / roundsTotal) * 100 : 0;
  const killShare = myTeam.length > 0
    ? (kills / Math.max(myTeam.reduce((sum, player) => sum + toNumber(player.kills), 0), 1)) * 100
    : 0;

  const openingWon = toNumber(userStats.opening_duels_won);
  const openingAttempted = toNumber(userStats.opening_duels_attempted);
  const openingLost = Math.max(openingAttempted - openingWon, 0);
  const openingSuccess = openingAttempted > 0 ? (openingWon / openingAttempted) * 100 : 0;
  const tradeKills = toNumber(userStats.trade_kills);
  const tradedDeaths = toNumber(userStats.traded_deaths);
  const tradeRatio = tradedDeaths > 0 ? tradeKills / tradedDeaths : tradeKills;

  const utilityDamage = toNumber(userStats.utility_damage);
  const enemiesPerFlash = toNumber(userStats.enemies_flashed_per_flash);
  const flashAssists = toNumber(userStats.flash_assists);
  const grenadesThrown = toNumber(userStats.grenades_thrown_total);
  const utilityDamagePerGrenade = grenadesThrown > 0 ? utilityDamage / grenadesThrown : 0;

  const timeToDamage = toNumber(userStats.time_to_damage_avg_ms, null);
  const crosshairError = toNumber(userStats.crosshair_placement_avg_error, null);
  const accuracy = toNumber(userStats.accuracy_overall);
  const shotsFired = toNumber(userStats.shots_fired);
  const shotsHit = toNumber(userStats.shots_hit);

  const bodyZones = useMemo(() => {
    const bodyParts = userStats.body_part_hits ?? {};
    const head = toNumber(bodyParts.head);
    const chest = toNumber(bodyParts.chest);
    const stomach = toNumber(bodyParts.stomach);
    const arms = toNumber(bodyParts.left_arm) + toNumber(bodyParts.right_arm);
    const legs = toNumber(bodyParts.left_leg) + toNumber(bodyParts.right_leg);
    const totalHits = head + chest + stomach + arms + legs;

    return {
      totalHits,
      zones: [
        { label: 'Cabeza', color: '#f59e0b', count: head, pct: zonePercent(head, totalHits) },
        { label: 'Pecho', color: '#3b82f6', count: chest, pct: zonePercent(chest, totalHits) },
        { label: 'Estómago', color: '#10b981', count: stomach, pct: zonePercent(stomach, totalHits) },
        { label: 'Brazos', color: '#f97316', count: arms, pct: zonePercent(arms, totalHits) },
        { label: 'Piernas', color: '#6366f1', count: legs, pct: zonePercent(legs, totalHits) },
      ],
    };
  }, [userStats.body_part_hits]);

  const hsRate = bodyZones.totalHits > 0
    ? zonePercent(bodyZones.zones[0].count, bodyZones.totalHits)
    : toNumber(userStats.hs_percentage);

  const radarData = useMemo(
    () => [
      { metric: 'Precisión', value: normalizeMetric(accuracy, 'accuracy') },
      { metric: 'Mira', value: normalizeMetric(crosshairError, 'crosshair_error', true) },
      { metric: 'TTD', value: normalizeMetric(timeToDamage, 'ttd_ms', true) },
      { metric: 'HS %', value: normalizeMetric(hsRate, 'hs_pct') },
      { metric: 'Apertura', value: normalizeMetric(openingSuccess, 'opening_success') },
    ],
    [accuracy, crosshairError, hsRate, openingSuccess, timeToDamage],
  );

  const duelData = useMemo(
    () => [
      { name: 'Ganados', value: openingWon, color: '#22c55e' },
      { name: 'Perdidos', value: openingLost, color: '#ef4444' },
    ].filter((item) => item.value > 0),
    [openingLost, openingWon],
  );

  const grenadeCards = useMemo(
    () => [
      {
        label: 'Cegadora',
        value: toNumber(userStats.flashes_thrown),
        sub: `${formatDecimal(enemiesPerFlash, 1)} rivales/cegadora`,
      },
      {
        label: 'HE',
        value: toNumber(userStats.he_thrown),
        sub: `${formatInteger(utilityDamage)} de daño útil`,
      },
      {
        label: 'Molotov',
        value: toNumber(userStats.molotovs_thrown),
        sub: 'Control de espacio',
      },
      {
        label: 'Humo',
        value: toNumber(userStats.smokes_thrown),
        sub: `${formatInteger(flashAssists)} asistencias con cegadora`,
      },
    ],
    [enemiesPerFlash, flashAssists, userStats, utilityDamage],
  );

  const clutchValues = useMemo(
    () => CLUTCH_DEFS.map((item) => toNumber(userStats[item.key])),
    [userStats],
  );
  const maxClutch = Math.max(...clutchValues, 1);

  const multikills = userStats.multikills ?? {};
  const multikillValues = MULTIKILL_DEFS.map((item) => toNumber(multikills[item.key]));
  const maxMultikill = Math.max(...multikillValues, 1);

  const weaponEntries = useMemo(() => {
    const entries = buildWeaponEntries(userStats.weapon_stats)
      .filter((weapon) => weapon.kills > 0 || weapon.damage > 0 || weapon.shots > 0)
      .sort((left, right) => toNumber(right[weaponSortKey]) - toNumber(left[weaponSortKey]));

    return entries;
  }, [userStats.weapon_stats, weaponSortKey]);

  const maxWeaponKills = Math.max(...weaponEntries.map((weapon) => weapon.kills), 1);
  const maxWeaponDamage = Math.max(...weaponEntries.map((weapon) => weapon.damage), 1);
  const totalWeaponKills = weaponEntries.reduce((sum, weapon) => sum + weapon.kills, 0);
  const totalWeaponDamage = weaponEntries.reduce((sum, weapon) => sum + weapon.damage, 0);
  const bestWeaponAccuracy = weaponEntries.length > 0
    ? Math.max(...weaponEntries.map((weapon) => weapon.accuracy))
    : 0;
  const topWeapon = weaponEntries[0] ?? null;

  const compareCandidates = useMemo(
    () => allPlayers.filter((player) => String(player.steam_id) !== String(matchData?.current_user_steam_id)),
    [allPlayers, matchData],
  );

  useEffect(() => {
    if (compareCandidates.length === 0) {
      setComparePlayerId('');
      return;
    }

    const selectedExists = compareCandidates.some((player) => String(player.steam_id) === String(comparePlayerId));
    if (!selectedExists) {
      setComparePlayerId(String(compareCandidates[0].steam_id));
    }
  }, [compareCandidates, comparePlayerId]);

  const comparePlayer = useMemo(
    () => compareCandidates.find((player) => String(player.steam_id) === String(comparePlayerId)) ?? compareCandidates[0] ?? null,
    [compareCandidates, comparePlayerId],
  );

  const currentProfile = useMemo(() => buildPlayerProfile(userStats, roundsTotal), [roundsTotal, userStats]);
  const compareProfile = useMemo(() => buildPlayerProfile(comparePlayer, roundsTotal), [comparePlayer, roundsTotal]);

  const compareMetricGroups = useMemo(() => {
    if (!currentProfile || !compareProfile) return [];

    return COMPARE_METRIC_GROUPS.map((group) => {
      const metrics = group.metrics
        .map((metric) => {
          const currentValue = currentProfile[metric.key];
          const compareValue = compareProfile[metric.key];
          const hasAnyValue = hasNumericValue(currentValue) || hasNumericValue(compareValue);

          if (!hasAnyValue && !metric.alwaysShow) {
            return null;
          }

          const scaleMax = getCompareMetricScaleMax(metric, currentValue, compareValue, roundsTotal);
          const currentWidth = getCompareMetricWidth(metric, currentValue, scaleMax);
          const compareWidth = getCompareMetricWidth(metric, compareValue, scaleMax);
          const leader = getCompareMetricLeader(metric, currentValue, compareValue);

          return {
            ...metric,
            currentValue,
            compareValue,
            currentWidth,
            compareWidth,
            scaleMax,
            leader,
            gapScore: getCompareMetricGapScore(currentWidth, compareWidth),
          };
        })
        .filter(Boolean);

      const currentWins = metrics.filter((metric) => metric.leader === 'current').length;
      const compareWins = metrics.filter((metric) => metric.leader === 'compare').length;

      return {
        ...group,
        metrics,
        currentWins,
        compareWins,
      };
    }).filter((group) => group.metrics.length > 0);
  }, [compareProfile, currentProfile, roundsTotal]);

  const compareMetricsFlat = useMemo(
    () => compareMetricGroups.flatMap((group) => group.metrics.map((metric) => ({ ...metric, groupLabel: group.label }))),
    [compareMetricGroups],
  );

  const compareHighlights = useMemo(
    () => compareMetricsFlat
      .filter((metric) => metric.leader === 'current' || metric.leader === 'compare')
      .sort((left, right) => right.gapScore - left.gapScore)
      .slice(0, 6),
    [compareMetricsFlat],
  );

  const compareSummary = useMemo(() => {
    if (!compareProfile || !currentProfile || compareMetricsFlat.length === 0) return null;

    const currentWins = compareMetricsFlat.filter((metric) => metric.leader === 'current').length;
    const compareWins = compareMetricsFlat.filter((metric) => metric.leader === 'compare').length;
    const biggestGap = [...compareMetricsFlat].sort((left, right) => right.gapScore - left.gapScore)[0] ?? null;

    return {
      currentWins,
      compareWins,
      biggestGap,
      sameTeam: currentProfile.team === compareProfile.team,
    };
  }, [compareMetricsFlat, compareProfile, currentProfile]);

  const tabs = useMemo(() => {
    const items = [
      { id: 'scoreboard', label: 'Marcador', icon: Trophy },
      { id: 'performance', label: 'Rendimiento', icon: Crosshair, hidden: !userStats },
      { id: 'compare', label: 'Comparar', icon: Users, hidden: compareCandidates.length === 0 || !currentProfile },
      { id: 'weapons', label: 'Armas', icon: Zap, hidden: weaponEntries.length === 0 },
      { id: 'replay', label: 'Repetición', icon: Play },
    ];

    return items.filter((item) => !item.hidden);
  }, [compareCandidates.length, currentProfile, userStats, weaponEntries.length]);

  if (loading) {
    return (
      <NavigationFrame>
        <div className="match-details-container">
          <div className="loading-state">
            <div className="spinner" />
            <p>Cargando detalles...</p>
          </div>
        </div>
      </NavigationFrame>
    );
  }

  if (error || !matchData) {
    return (
      <NavigationFrame>
        <div className="match-details-container">
          <div className="error-state">
            <h2>Error</h2>
            <p>{error || 'Partida no encontrada'}</p>
            <button type="button" onClick={() => navigate('/history-games')}>
              <ArrowLeft size={16} />
              Volver al historial
            </button>
          </div>
        </div>
      </NavigationFrame>
    );
  }

  return (
    <NavigationFrame>
      <div className="match-details-container">
        <div
          className={`match-header ${isVictory ? 'victory' : 'defeat'}`}
          style={{ backgroundImage: `url(${getMapImage(metadata.map_name)})` }}
        >
          <div className="header-overlay" />
          <div className="header-content">
            <button className="back-btn" type="button" onClick={() => navigate('/history-games')}>
              <ArrowLeft size={18} />
              Volver
            </button>

            <div className="match-result">
              <span className="map-name">{mapDisplayName}</span>
              <div className={`result-badge ${isVictory ? 'win' : 'loss'}`}>
                {isVictory ? 'Victoria' : 'Derrota'}
              </div>
            </div>

            <div className="score-display">
              <span className={`team-score ${isVictory ? 'winner' : ''}`}>{metadata.team_score}</span>
              <span className="score-divider">:</span>
              <span className={`team-score ${!isVictory ? 'winner' : ''}`}>{metadata.opponent_score}</span>
            </div>

            <div className="match-meta">
              <span><Clock size={14} /> {formatDuration(metadata.duration_seconds)}</span>
              <span><Users size={14} /> {metadata.total_rounds} rondas</span>
              <span><Sparkles size={14} /> {userTeam || 'Equipo no detectado'}</span>
            </div>
          </div>
        </div>

        <div className="match-tabs-shell">
          <div className="p-tabs match-tabs" role="tablist" aria-label="Secciones del detalle de la partida">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === tab.id}
                  className={`p-tab ${activeTab === tab.id ? 'active' : ''}`}
                  onClick={() => setActiveTab(tab.id)}
                >
                  <Icon size={15} />
                  {tab.label}
                </button>
              );
            })}
          </div>
        </div>

        {activeTab === 'scoreboard' && (
          <section className="section scoreboard-section p-panel">
            <div className="match-section-heading match-section-heading--compact">
              <h2><Trophy size={20} /> Marcador</h2>
              <span className={`match-section-tag ${isVictory ? 'win' : 'loss'}`}>
                {isVictory ? 'Partida ganada' : 'Partida perdida'}
              </span>
            </div>

            <div className={`team-table ${isVictory ? 'winner-team' : 'loser-team'}`}>
              <div className="team-header">
                <Users size={16} />
                <span>Mi equipo</span>
                <span className={`team-result-badge ${isVictory ? 'win' : 'loss'}`}>
                  {isVictory ? 'VICTORIA' : 'DERROTA'}
                </span>
              </div>
              <table>
                <thead>
                  <tr>
                    <th>Jugador</th>
                    <th>K</th>
                    <th>D</th>
                    <th>A</th>
                    <th>K/D</th>
                    <th>ADR</th>
                    <th>HS%</th>
                    <th>Valoración</th>
                    <th>KAST</th>
                  </tr>
                </thead>
                <tbody>
                  {myTeam.map((player, index) => (
                    <tr
                      key={player.steam_id}
                      className={String(player.steam_id) === String(matchData.current_user_steam_id) ? 'current-user' : ''}
                    >
                      <td className="player-name">
                        {index === 0 && <span className="mvp-badge">★</span>}
                        {player.name}
                      </td>
                      <td>{player.kills}</td>
                      <td>{player.deaths}</td>
                      <td>{player.assists}</td>
                      <td style={{ color: getKDColor(toNumber(player.kd_ratio)) }}>{formatDecimal(player.kd_ratio, 2)}</td>
                      <td>{formatInteger(Math.round(toNumber(player.adr)))}</td>
                      <td>{formatInteger(Math.round(toNumber(player.hs_percentage)))}%</td>
                      <td style={{ color: getRatingColor(toNumber(player.hltv_rating)) }}>{formatDecimal(player.hltv_rating, 2)}</td>
                      <td>{formatInteger(Math.round(toNumber(player.kast)))}%</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className={`team-table ${!isVictory ? 'winner-team' : 'loser-team'}`}>
              <div className="team-header enemy">
                <Sword size={16} />
                <span>Equipo enemigo</span>
                <span className={`team-result-badge ${!isVictory ? 'win' : 'loss'}`}>
                  {!isVictory ? 'VICTORIA' : 'DERROTA'}
                </span>
              </div>
              <table>
                <thead>
                  <tr>
                    <th>Jugador</th>
                    <th>K</th>
                    <th>D</th>
                    <th>A</th>
                    <th>K/D</th>
                    <th>ADR</th>
                    <th>HS%</th>
                    <th>Valoración</th>
                    <th>KAST</th>
                  </tr>
                </thead>
                <tbody>
                  {enemyTeam.map((player, index) => (
                    <tr key={player.steam_id}>
                      <td className="player-name">
                        {index === 0 && <span className="mvp-badge">★</span>}
                        {player.name}
                      </td>
                      <td>{player.kills}</td>
                      <td>{player.deaths}</td>
                      <td>{player.assists}</td>
                      <td style={{ color: getKDColor(toNumber(player.kd_ratio)) }}>{formatDecimal(player.kd_ratio, 2)}</td>
                      <td>{formatInteger(Math.round(toNumber(player.adr)))}</td>
                      <td>{formatInteger(Math.round(toNumber(player.hs_percentage)))}%</td>
                      <td style={{ color: getRatingColor(toNumber(player.hltv_rating)) }}>{formatDecimal(player.hltv_rating, 2)}</td>
                      <td>{formatInteger(Math.round(toNumber(player.kast)))}%</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        )}

        {activeTab === 'performance' && userStats && (
          <section className="section performance-section p-panel">
            <div className="match-section-heading match-section-heading--compact">
              <h2><Crosshair size={20} /> Rendimiento</h2>
              <span className="match-section-tag neutral">{userStats.name || 'Jugador actual'}</span>
            </div>

            <div className="p-metric-row match-metric-row">
              <MetricCell
                value={formatDecimal(rating, 2)}
                label="Valoración HLTV"
                sub={`Impacto ${formatDecimal(impactRating, 2)}`}
                badge={getBadgeForMetric(rating, 'hltv_rating')}
              />
              <MetricCell
                value={formatPercent(kast, 0)}
                label="KAST"
                sub={`${formatInteger(roundsSurvived)} rondas vivo`}
                badge={getBadgeForMetric(kast, 'kast')}
              />
              <MetricCell
                value={formatDecimal(kdRatio, 2)}
                label="K/D"
                sub={`${formatInteger(kills)} bajas · ${formatInteger(deaths)} muertes`}
                badge={getBadgeForMetric(kdRatio, 'kd_ratio')}
              />
              <MetricCell
                value={formatInteger(Math.round(adr))}
                label="ADR"
                sub={`${formatInteger(assists)} asistencias`}
                badge={getBadgeForMetric(adr, 'adr')}
              />
              <MetricCell
                value={formatPercent(hsRate, 0)}
                label="HS %"
                sub={`${formatInteger(bodyZones.totalHits)} impactos analizados`}
                badge={getBadgeForMetric(hsRate, 'hs_pct', false, bodyZones.totalHits > 0 || toNumber(userStats.hs_percentage) > 0)}
              />
            </div>

            <div className="p-grid p-grid-dashboard match-performance-grid">
              <div className="p-card p-card--chart match-feature-card">
                <div className="p-card-toolbar">
                  <div>
                    <p className="p-card-title">Aperturas y control del primer duelo</p>
                  </div>
                </div>

                <div className="p-duel-layout">
                  <div className="p-donut-wrap">
                    {duelData.length > 0 ? (
                      <ResponsiveContainer width="100%" height={220}>
                        <PieChart>
                          <Pie data={duelData} dataKey="value" innerRadius={58} outerRadius={84} paddingAngle={4}>
                            {duelData.map((item) => (
                              <Cell key={item.name} fill={item.color} />
                            ))}
                          </Pie>
                          <Tooltip
                            contentStyle={RECHARTS_TOOLTIP_STYLE}
                            labelStyle={TOOLTIP_LABEL_STYLE}
                            itemStyle={TOOLTIP_ITEM_STYLE}
                            formatter={(value) => [formatInteger(value), 'Duelos']}
                          />
                        </PieChart>
                      </ResponsiveContainer>
                    ) : (
                      <div className="p-empty p-empty--chart">
                        <span>Sin duelos iniciales suficientes para una lectura fiable.</span>
                      </div>
                    )}

                    <div className="p-donut-center">
                      <strong>{formatPercent(openingSuccess, 0)}</strong>
                      <span>aperturas</span>
                    </div>
                  </div>

                  <div className="p-stat-list p-stat-list--dense">
                    <div className="p-stat-row">
                      <span className="p-stat-row-label">Intentados</span>
                      <span className="p-stat-row-value">{formatInteger(openingAttempted)}</span>
                    </div>
                    <div className="p-stat-row">
                      <span className="p-stat-row-label">Ganados</span>
                      <span className="p-stat-row-value p-good-text">{formatInteger(openingWon)}</span>
                    </div>
                    <div className="p-stat-row">
                      <span className="p-stat-row-label">Perdidos</span>
                      <span className="p-stat-row-value p-bad-text">{formatInteger(openingLost)}</span>
                    </div>
                    <div className="p-stat-row">
                      <span className="p-stat-row-label">Bajas de intercambio</span>
                      <span className="p-stat-row-value">{formatInteger(tradeKills)}</span>
                    </div>
                    <div className="p-stat-row">
                      <span className="p-stat-row-label">Muertes intercambiadas</span>
                      <span className="p-stat-row-value">{formatInteger(tradedDeaths)}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div className="p-stack-col">
                <div className="p-card match-side-card">
                  <p className="p-card-title">Intercambios</p>
                  <div className="p-compare-grid">
                    <div className="p-compare-block good">
                      <span className="p-compare-label">Bajas de intercambio</span>
                      <strong className="p-compare-value">{formatInteger(tradeKills)}</strong>
                    </div>
                    <div className="p-compare-block bad">
                      <span className="p-compare-label">Muertes intercambiadas</span>
                      <strong className="p-compare-value">{formatInteger(tradedDeaths)}</strong>
                    </div>
                  </div>
                  <div className="p-progress-wrap p-mt-14">
                    <div className="p-progress-head">
                      <span className="p-progress-head-label">Eficacia en intercambios</span>
                      <span className="p-progress-head-value">{formatDecimal(tradeRatio, 2)}</span>
                    </div>
                    <div className="p-split-bar p-split-bar--lg">
                      <div
                        className="p-split-bar-ct p-split-bar-success"
                        style={{ width: `${(() => {
                          const total = tradeKills + tradedDeaths;
                          return total > 0 ? (tradeKills / total) * 100 : 50;
                        })()}%` }}
                      />
                      <div
                        className="p-split-bar-t p-split-bar-danger"
                        style={{ width: `${(() => {
                          const total = tradeKills + tradedDeaths;
                          return total > 0 ? (tradedDeaths / total) * 100 : 50;
                        })()}%` }}
                      />
                    </div>
                  </div>
                </div>

                <div className="p-card match-side-card">
                  <p className="p-card-title">Resumen</p>
                  <div className="match-side-stat-list">
                    <div className="match-side-stat">
                      <span>Contribución al equipo</span>
                      <strong>{formatPercent(killShare, 0)}</strong>
                    </div>
                    <div className="match-side-stat">
                      <span>Rondas vivo</span>
                      <strong>{formatPercent(survivalRate, 0)}</strong>
                    </div>
                    <div className="match-side-stat">
                      <span>Impacto</span>
                      <strong>{formatDecimal(impactRating, 2)}</strong>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="p-grid p-grid-2 match-performance-grid">
              <div className="p-card p-card--chart match-feature-card">
                <p className="p-card-title">Radar mecánico</p>
                <div className="p-radar-wrap">
                  <ResponsiveContainer width="100%" height={300}>
                    <RadarChart data={radarData} outerRadius={110}>
                      <PolarGrid stroke="rgba(148, 163, 184, 0.14)" />
                      <PolarAngleAxis dataKey="metric" tick={{ fill: '#94a3b8', fontSize: 12, fontWeight: 700 }} />
                      <Tooltip
                        contentStyle={RECHARTS_TOOLTIP_STYLE}
                        labelStyle={TOOLTIP_LABEL_STYLE}
                        itemStyle={TOOLTIP_ITEM_STYLE}
                        formatter={(value) => [`${Number(value).toFixed(0)}/100`, 'Puntuación']}
                      />
                      <Radar dataKey="value" stroke="#6366f1" fill="#6366f1" fillOpacity={0.28} strokeWidth={2.5} />
                    </RadarChart>
                  </ResponsiveContainer>
                </div>

                <div className="p-benchmark-list">
                  <div className="p-benchmark-item">
                    <div className="p-progress-head">
                      <span className="p-progress-head-label">Precisión aplicada</span>
                      <span className="p-progress-head-value">{formatPercent(accuracy, 1)}</span>
                    </div>
                    <div className="p-progress-track">
                      <div className="p-progress-fill p-progress-fill--green" style={{ width: `${clamp(accuracy)}%` }} />
                    </div>
                  </div>
                  <div className="p-benchmark-item">
                    <div className="p-progress-head">
                      <span className="p-progress-head-label">Error de mira</span>
                      <span className="p-progress-head-value">{formatDegrees(crosshairError, 1)}</span>
                    </div>
                    <div className="p-progress-track">
                      <div className="p-progress-fill p-progress-fill--purple" style={{ width: `${normalizeMetric(crosshairError, 'crosshair_error', true)}%` }} />
                    </div>
                  </div>
                  <div className="p-benchmark-item">
                    <div className="p-progress-head">
                      <span className="p-progress-head-label">Tiempo al daño</span>
                      <span className="p-progress-head-value">{formatMilliseconds(timeToDamage)}</span>
                    </div>
                    <div className="p-progress-track">
                      <div className="p-progress-fill p-progress-fill--orange" style={{ width: `${normalizeMetric(timeToDamage, 'ttd_ms', true)}%` }} />
                    </div>
                  </div>
                </div>
              </div>

              <div className="p-card match-feature-card">
                <p className="p-card-title">Distribución de impactos</p>
                <div className="p-silhouette-layout">
                  <svg viewBox="0 0 220 320" className="p-aim-silhouette" aria-hidden="true">
                    <circle cx="110" cy="40" r="26" fill="rgba(245, 158, 11, 0.16)" stroke="#f59e0b" strokeWidth="2" style={{ opacity: 0.25 + bodyZones.zones[0].pct / 100 }} />
                    <rect x="82" y="78" width="56" height="74" rx="18" fill="rgba(59, 130, 246, 0.16)" stroke="#3b82f6" strokeWidth="2" style={{ opacity: 0.25 + bodyZones.zones[1].pct / 100 }} />
                    <rect x="92" y="154" width="36" height="42" rx="14" fill="rgba(16, 185, 129, 0.16)" stroke="#10b981" strokeWidth="2" style={{ opacity: 0.25 + bodyZones.zones[2].pct / 100 }} />
                    <rect x="44" y="86" width="30" height="108" rx="14" fill="rgba(249, 115, 22, 0.16)" stroke="#f97316" strokeWidth="2" style={{ opacity: 0.25 + bodyZones.zones[3].pct / 100 }} />
                    <rect x="146" y="86" width="30" height="108" rx="14" fill="rgba(249, 115, 22, 0.16)" stroke="#f97316" strokeWidth="2" style={{ opacity: 0.25 + bodyZones.zones[3].pct / 100 }} />
                    <rect x="88" y="202" width="22" height="96" rx="12" fill="rgba(99, 102, 241, 0.16)" stroke="#6366f1" strokeWidth="2" style={{ opacity: 0.25 + bodyZones.zones[4].pct / 100 }} />
                    <rect x="112" y="202" width="22" height="96" rx="12" fill="rgba(99, 102, 241, 0.16)" stroke="#6366f1" strokeWidth="2" style={{ opacity: 0.25 + bodyZones.zones[4].pct / 100 }} />

                    <text x="110" y="45" textAnchor="middle" className="p-silhouette-text">{bodyZones.zones[0].pct.toFixed(1)}%</text>
                    <text x="110" y="118" textAnchor="middle" className="p-silhouette-text">{bodyZones.zones[1].pct.toFixed(1)}%</text>
                    <text x="110" y="180" textAnchor="middle" className="p-silhouette-text">{bodyZones.zones[2].pct.toFixed(1)}%</text>
                    <text x="58" y="142" textAnchor="middle" className="p-silhouette-text">{bodyZones.zones[3].pct.toFixed(1)}%</text>
                    <text x="162" y="142" textAnchor="middle" className="p-silhouette-text">{bodyZones.zones[3].pct.toFixed(1)}%</text>
                    <text x="110" y="256" textAnchor="middle" className="p-silhouette-text">{bodyZones.zones[4].pct.toFixed(1)}%</text>
                  </svg>

                  <div className="p-aim-legend">
                    {bodyZones.zones.map((zone) => (
                      <div key={zone.label} className="p-aim-legend-item">
                        <span className="p-aim-legend-dot" style={{ background: zone.color }} />
                        <span>{zone.label}</span>
                        <strong>{formatInteger(zone.count)}</strong>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="p-num-grid p-grid-2 match-aim-summary">
                  <div className="p-num-cell">
                    <span className="p-num-cell-value">{formatInteger(shotsFired)}</span>
                    <span className="p-num-cell-label">Disparos efectuados</span>
                  </div>
                  <div className="p-num-cell">
                    <span className="p-num-cell-value">{formatInteger(shotsHit)}</span>
                    <span className="p-num-cell-label">Disparos impactados</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="p-grid p-grid-2 match-performance-grid">
              <div className="p-card match-feature-card">
                <p className="p-card-title">Utilidad que sí deja huella</p>

                <div className="p-metric-row match-inline-metric-row">
                  <MetricCell
                    value={formatInteger(utilityDamage)}
                    label="Daño de utilidad"
                    badge={getBadgeForMetric(utilityDamagePerGrenade, 'he_damage_per_nade')}
                  />
                  <MetricCell
                    value={formatDecimal(enemiesPerFlash, 1)}
                    label="Rivales por cegadora"
                    badge={getBadgeForMetric(enemiesPerFlash, 'enemies_per_flash', false, toNumber(userStats.flashes_thrown) > 0)}
                  />
                  <MetricCell value={formatInteger(flashAssists)} label="Asistencias con cegadora" />
                </div>

                <div className="match-grenade-grid">
                  {grenadeCards.map((item) => (
                    <div key={item.label} className="match-grenade-card">
                      <span className="match-grenade-label">{item.label}</span>
                      <strong className="match-grenade-value">{formatInteger(item.value)}</strong>
                      <span className="match-grenade-sub">{item.sub}</span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="p-card match-feature-card">
                <p className="p-card-title">Cierre de rondas</p>

                <div className="match-closing-grid">
                  <div>
                    <div className="match-subsection-heading">Clutches</div>
                    <div className="p-clutch-grid">
                      {CLUTCH_DEFS.map((item, index) => {
                        const value = clutchValues[index];
                        const opacity = value > 0 ? (value / maxClutch) * 0.85 + 0.15 : 0;

                        return (
                          <div
                            key={item.key}
                            className={`p-mk-item ${value === 0 ? 'zero' : ''}`}
                            style={{ '--mk-glow': 'rgba(59, 130, 246, 0.24)', '--mk-opacity': opacity }}
                          >
                            <span className="p-mk-item-label">{item.label}</span>
                            <span className="p-mk-item-value">{value}</span>
                          </div>
                        );
                      })}
                    </div>
                  </div>

                  <div>
                    <div className="match-subsection-heading">Multi-bajas</div>
                    <div className="p-mk-grid">
                      {MULTIKILL_DEFS.map((item, index) => {
                        const value = multikillValues[index];
                        const opacity = value > 0 ? (value / maxMultikill) * 0.85 + 0.15 : 0;

                        return (
                          <div
                            key={item.key}
                            className={`p-mk-item ${item.className} ${value === 0 ? 'zero' : ''}`}
                            style={{ '--mk-opacity': opacity }}
                          >
                            <span className="p-mk-item-label">{item.label}</span>
                            <span className="p-mk-item-value">{value}</span>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>
        )}

        {activeTab === 'weapons' && weaponEntries.length > 0 && (
          <section className="section weapons-section p-panel">
            <div className="match-section-heading match-section-heading--compact">
              <h2><Zap size={20} /> Armas</h2>
              <span className="match-section-tag neutral">
                {topWeapon ? formatWeaponName(topWeapon.weapon) : 'Sin armas'}
              </span>
            </div>

            <div className="p-metric-row match-metric-row">
              <MetricCell value={formatInteger(totalWeaponKills)} label="Bajas con armas" />
              <MetricCell value={formatInteger(totalWeaponDamage)} label="Daño total" />
              <MetricCell value={formatPercent(bestWeaponAccuracy, 1)} label="Mejor precisión" />
              <MetricCell value={formatInteger(weaponEntries.length)} label="Armas usadas" />
            </div>

            <div className="p-card match-weapon-card-shell">
              <div className="p-table-toolbar match-weapon-toolbar">
                <div>
                  <p className="p-card-title">Top del arsenal</p>
                </div>
                <select value={weaponSortKey} onChange={(event) => setWeaponSortKey(event.target.value)} className="p-select">
                  {WEAPON_SORT_OPTIONS.map((option) => (
                    <option key={option.key} value={option.key}>Ordenar por {option.label}</option>
                  ))}
                </select>
              </div>

              <div className="p-weapon-card-list">
                {weaponEntries.slice(0, 6).map((weapon, index) => {
                  const accuracyAngle = `${clamp(weapon.accuracy, 0, 100) * 3.6}deg`;
                  const weaponLabel = formatWeaponName(weapon.weapon);
                  const weaponIcon = getWeaponIconPath(weapon.weapon);

                  return (
                    <div key={`${weapon.weapon}-${index}`} className="p-weapon-card">
                      <div className="match-weapon-card-main">
                        <div className="p-weapon-card-head">
                          <span className="p-weapon-rank-badge">{TOP_BADGES[index] || `#${index + 1}`}</span>
                          {weaponIcon && (
                            <div className="p-weapon-icon-shell">
                              <img src={weaponIcon} alt={weaponLabel} className="p-weapon-icon" loading="lazy" />
                            </div>
                          )}
                          <div className="p-weapon-title-block">
                            <strong className="p-weapon-card-title">{weaponLabel}</strong>
                            <span className="p-weapon-card-sub">
                              {formatInteger(weapon.kills)} bajas · {formatInteger(weapon.damage)} de daño · {formatInteger(weapon.headshots)} HS
                            </span>
                          </div>
                        </div>

                        <div className="p-weapon-bars">
                          <div className="p-weapon-bar-row">
                            <span>Bajas</span>
                            <div className="p-progress-track">
                              <div className="p-progress-fill p-progress-fill--purple" style={{ width: `${(weapon.kills / maxWeaponKills) * 100}%` }} />
                            </div>
                            <strong>{formatInteger(weapon.kills)}</strong>
                          </div>
                          <div className="p-weapon-bar-row">
                            <span>Daño</span>
                            <div className="p-progress-track">
                              <div className="p-progress-fill p-progress-fill--orange" style={{ width: `${(weapon.damage / maxWeaponDamage) * 100}%` }} />
                            </div>
                            <strong>{formatInteger(weapon.damage)}</strong>
                          </div>
                        </div>
                      </div>

                      <div className="p-weapon-ring" style={{ '--p-ring-angle': accuracyAngle, '--p-ring-color': 'var(--p-blue)' }}>
                        <div className="p-weapon-ring-core">
                          <span>{formatDecimal(weapon.accuracy, 1)}%</span>
                          <small>PREC.</small>
                        </div>
                      </div>

                      <div className="match-weapon-meta">
                        <span className="p-badge neutral">HS {formatDecimal(weapon.hs_pct, 1)}%</span>
                        <span className="p-badge good">Impactos {formatInteger(weapon.hits)}</span>
                        <span className="p-badge average">Disparos {formatInteger(weapon.shots)}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            <div className="p-card match-weapon-table-shell">
              <div className="p-table-toolbar">
                <p className="p-card-title">Detalle por arma</p>
                <button type="button" className="p-ghost-btn" onClick={() => setShowWeaponTable((current) => !current)}>
                  {showWeaponTable ? 'Ocultar tabla' : 'Ver tabla completa'}
                </button>
              </div>

              {showWeaponTable && (
                <div className="p-table-wrap">
                  <table className="p-table">
                    <thead>
                      <tr>
                        <th>Arma</th>
                        <th>Bajas</th>
                        <th>Daño</th>
                        <th>Precisión</th>
                        <th>HS %</th>
                        <th>Impactos</th>
                        <th>Disparos</th>
                      </tr>
                    </thead>
                    <tbody>
                      {weaponEntries.map((weapon, index) => {
                        const weaponLabel = formatWeaponName(weapon.weapon);
                        const weaponIcon = getWeaponIconPath(weapon.weapon);

                        return (
                          <tr key={`${weapon.weapon}-table-${index}`}>
                            <td className="p-weapon-table-cell">
                              <div className="p-weapon-name">
                                {weaponIcon && (
                                  <img src={weaponIcon} alt={weaponLabel} className="p-weapon-table-icon" loading="lazy" />
                                )}
                                {index < 3 && <span className="p-weapon-rank">{TOP_BADGES[index]}</span>}
                                {weaponLabel}
                              </div>
                            </td>
                            <td>{formatInteger(weapon.kills)}</td>
                            <td>{formatInteger(weapon.damage)}</td>
                            <td>{formatDecimal(weapon.accuracy, 1)}%</td>
                            <td>{formatDecimal(weapon.hs_pct, 1)}%</td>
                            <td>{formatInteger(weapon.hits)}</td>
                            <td>{formatInteger(weapon.shots)}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </section>
        )}

        {activeTab === 'compare' && currentProfile && compareProfile && compareSummary && (
          <section className="section compare-section p-panel">
            <div className="match-section-heading match-section-heading--compact">
              <h2><Users size={20} /> Comparativa de jugadores</h2>
              <span className={`match-section-tag ${compareSummary.sameTeam ? 'neutral' : 'loss'}`}>
                {compareSummary.sameTeam ? 'Comparando con compañero' : 'Comparando con rival'}
              </span>
            </div>

            <div className="match-compare-hero">
              <div className="match-compare-player-card current">
                <span className="match-compare-kicker">Tú</span>
                <h3 className="match-compare-player-name">{currentProfile.name}</h3>
                <div className="match-compare-player-meta">
                  <span className="match-compare-team-pill">{currentProfile.team}</span>
                  <span className="match-compare-main-metric">Valoración {formatDecimal(currentProfile.rating, 2)}</span>
                </div>
                <div className="match-compare-player-mini-grid">
                  <div>
                    <strong>{formatInteger(currentProfile.kills)}</strong>
                    <span>Bajas</span>
                  </div>
                  <div>
                    <strong>{formatInteger(Math.round(currentProfile.adr))}</strong>
                    <span>ADR</span>
                  </div>
                  <div>
                    <strong>{formatPercent(currentProfile.kast, 0)}</strong>
                    <span>KAST</span>
                  </div>
                </div>
              </div>

              <div className="match-compare-control-card">
                <span className="match-compare-control-label">Jugador a comparar</span>
                <select
                  value={comparePlayerId}
                  onChange={(event) => setComparePlayerId(event.target.value)}
                  className="p-select match-compare-select"
                >
                  {compareCandidates.map((player) => (
                    <option key={player.steam_id} value={player.steam_id}>
                      {player.name} · {player.team}
                    </option>
                  ))}
                </select>

                <div className="match-compare-summary-pill">
                  <strong>{compareSummary.currentWins}</strong>
                  <span>métricas a favor</span>
                </div>

                {compareSummary.biggestGap && (
                  <p className="match-compare-summary-copy">
                    Mayor diferencia en <strong>{compareSummary.biggestGap.label}</strong>.
                  </p>
                )}
              </div>

              <div className="match-compare-player-card compare">
                <span className="match-compare-kicker">{compareSummary.sameTeam ? 'Compañero' : 'Rival'}</span>
                <h3 className="match-compare-player-name">{compareProfile.name}</h3>
                <div className="match-compare-player-meta">
                  <span className="match-compare-team-pill is-compare">{compareProfile.team}</span>
                  <span className="match-compare-main-metric">Valoración {formatDecimal(compareProfile.rating, 2)}</span>
                </div>
                <div className="match-compare-player-mini-grid">
                  <div>
                    <strong>{formatInteger(compareProfile.kills)}</strong>
                    <span>Bajas</span>
                  </div>
                  <div>
                    <strong>{formatInteger(Math.round(compareProfile.adr))}</strong>
                    <span>ADR</span>
                  </div>
                  <div>
                    <strong>{formatPercent(compareProfile.kast, 0)}</strong>
                    <span>KAST</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="match-compare-main-grid">
              <div className="p-card match-compare-atlas-card">
                <div className="match-compare-atlas-head">
                  <div>
                    <p className="p-card-title">Panel comparativo</p>
                    <p className="match-compare-atlas-copy">Valores brutos, más contexto y escalas dinámicas por métrica.</p>
                  </div>

                  <div className="match-compare-radar-legend">
                    <span className="match-compare-radar-pill current">{currentProfile.name}</span>
                    <span className="match-compare-radar-pill compare">{compareProfile.name}</span>
                  </div>
                </div>

                <div className="match-compare-group-grid">
                  {compareMetricGroups.map((group) => (
                    <article key={group.id} className={`match-compare-group match-compare-group--${group.tone}`}>
                      <div className="match-compare-group-head">
                        <div>
                          <span className="match-compare-group-label">{group.label}</span>
                          <strong className="match-compare-group-score">{group.currentWins} · {group.compareWins}</strong>
                        </div>
                        <span className="match-compare-group-meta">
                          {group.currentWins === group.compareWins
                            ? 'Parejo'
                            : group.currentWins > group.compareWins
                              ? `Ventaja ${currentProfile.name}`
                              : `Ventaja ${compareProfile.name}`}
                        </span>
                      </div>

                      <div className="match-compare-lane-list">
                        {group.metrics.map((metric) => (
                          <div
                            key={`${group.id}-${metric.key}`}
                            className={`match-compare-lane ${metric.leader === 'no-data' ? 'is-muted' : ''}`}
                          >
                            <div className="match-compare-lane-head">
                              <span className="match-compare-lane-player current">{currentProfile.name}</span>
                              <span className="match-compare-lane-label">{metric.label}</span>
                              <span className="match-compare-lane-player compare">{compareProfile.name}</span>
                            </div>

                            <div className="match-compare-lane-track">
                              <div className="match-compare-lane-column current">
                                <strong className={`match-compare-lane-value ${metric.leader === 'current' ? 'is-leading' : ''}`}>
                                  {formatCompareMetricValue(metric, metric.currentValue)}
                                </strong>
                                <div className="match-compare-lane-bar-shell">
                                  <div className="match-compare-lane-bar current" style={{ width: `${metric.currentWidth}%` }} />
                                </div>
                              </div>

                              <div className="match-compare-lane-column compare">
                                <strong className={`match-compare-lane-value ${metric.leader === 'compare' ? 'is-leading' : ''}`}>
                                  {formatCompareMetricValue(metric, metric.compareValue)}
                                </strong>
                                <div className="match-compare-lane-bar-shell">
                                  <div className="match-compare-lane-bar compare" style={{ width: `${metric.compareWidth}%` }} />
                                </div>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </article>
                  ))}
                </div>
              </div>

              <div className="match-compare-side-stack">
                <div className="p-card match-compare-summary-card">
                  <p className="p-card-title">Balance global</p>
                  <div className="match-compare-scoreline">
                    <div className="match-compare-score-block current">
                      <strong>{compareSummary.currentWins}</strong>
                      <span>{currentProfile.name}</span>
                    </div>
                    <div className="match-compare-score-divider">VS</div>
                    <div className="match-compare-score-block compare">
                      <strong>{compareSummary.compareWins}</strong>
                      <span>{compareProfile.name}</span>
                    </div>
                  </div>

                  {compareSummary.biggestGap && (
                    <div className="match-compare-big-gap">
                      <span className="match-compare-big-gap-label">Mayor diferencia</span>
                      <strong>{compareSummary.biggestGap.label}</strong>
                      <p>
                        {compareSummary.biggestGap.leader === 'current' ? currentProfile.name : compareProfile.name} marca la ventaja más clara en esta métrica.
                      </p>
                    </div>
                  )}
                </div>

                <div className="p-card match-compare-highlights-card">
                  <p className="p-card-title">Ventajas claras</p>
                  <div className="match-compare-highlight-list">
                    {compareHighlights.map((metric) => (
                      <div key={`${metric.groupLabel}-${metric.key}`} className="match-compare-highlight-row">
                        <div>
                          <span className="match-compare-highlight-group">{metric.groupLabel}</span>
                          <strong>{metric.label}</strong>
                        </div>
                        <span className={`match-compare-highlight-pill ${metric.leader}`}>
                          {metric.leader === 'current' ? currentProfile.name : compareProfile.name}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="p-card match-compare-snapshot-card">
                  <p className="p-card-title">Contexto rápido</p>
                  <div className="match-compare-context-grid match-compare-context-grid--stacked">
                    <div className="match-compare-context-card">
                      <span className="match-subsection-heading">Aperturas</span>
                      <div className="match-compare-context-values">
                        <div>
                          <strong>{formatPercent(currentProfile.openingSuccess, 0)}</strong>
                          <span>{currentProfile.name}</span>
                        </div>
                        <div>
                          <strong>{formatPercent(compareProfile.openingSuccess, 0)}</strong>
                          <span>{compareProfile.name}</span>
                        </div>
                      </div>
                    </div>

                    <div className="match-compare-context-card">
                      <span className="match-subsection-heading">Asistencias con cegadora</span>
                      <div className="match-compare-context-values">
                        <div>
                          <strong>{formatInteger(currentProfile.flashAssists)}</strong>
                          <span>{currentProfile.name}</span>
                        </div>
                        <div>
                          <strong>{formatInteger(compareProfile.flashAssists)}</strong>
                          <span>{compareProfile.name}</span>
                        </div>
                      </div>
                    </div>

                    <div className="match-compare-context-card">
                      <span className="match-subsection-heading">Clutches</span>
                      <div className="match-compare-context-values">
                        <div>
                          <strong>{formatInteger(currentProfile.totalClutchesWon)}</strong>
                          <span>{currentProfile.totalClutchesAttempted > 0 ? `${formatPercent(currentProfile.clutchConversion, 0)} conversión` : 'Sin intentos'}</span>
                        </div>
                        <div>
                          <strong>{formatInteger(compareProfile.totalClutchesWon)}</strong>
                          <span>{compareProfile.totalClutchesAttempted > 0 ? `${formatPercent(compareProfile.clutchConversion, 0)} conversión` : 'Sin intentos'}</span>
                        </div>
                      </div>
                    </div>

                    <div className="match-compare-context-card match-compare-weapon-card">
                      <span className="match-subsection-heading">Arma principal</span>
                      <div className="match-compare-weapon-duel">
                        <div>
                          <strong>{currentProfile.topWeapon ? formatWeaponName(currentProfile.topWeapon.weapon) : 'N/D'}</strong>
                          <span>{currentProfile.topWeapon ? `${formatInteger(currentProfile.topWeapon.kills)} bajas` : 'Sin datos'}</span>
                        </div>
                        <div>
                          <strong>{compareProfile.topWeapon ? formatWeaponName(compareProfile.topWeapon.weapon) : 'N/D'}</strong>
                          <span>{compareProfile.topWeapon ? `${formatInteger(compareProfile.topWeapon.kills)} bajas` : 'Sin datos'}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>
        )}

        {activeTab === 'replay' && (
          <section className="section replay-section p-panel">
            <div className="match-section-heading match-section-heading--compact">
              <h2><Shield size={20} /> Repetición 2D</h2>
            </div>
            <Replay2DViewer matchId={matchID} initialRound={1} />
          </section>
        )}
      </div>
    </NavigationFrame>
  );
};

export default MatchDetails;
