import { getMissionPoints, getNextWeeklyReset, getWeeklyScore } from './progressModel';
import { createLeaderboard } from './progressLeagueData';

const PRIMARY_MISSION = {
  id: 'opening-duels',
  role: 'primary',
  category: 'Duelo de apertura',
  title: 'Recupera el control del primer duelo',
  diagnosis: 'La IA detectó que tus aperturas agresivas pierden valor cuando haces wide peek sin utilidad previa.',
  coachNote: 'Frena medio paso antes del contacto y exige una flash o una segunda línea de tiro.',
  metricLabel: 'Duelos de apertura ganados',
  baseline: 41,
  current: 53,
  target: 56,
  unit: '%',
  evidenceMatches: 8,
  evidence: [
    { map: 'Mirage', round: 7, note: 'Entrada en corto sin flash de apoyo', impact: '-18% prob. de ronda' },
    { map: 'Ancient', round: 19, note: 'Wide peek con velocidad residual', impact: 'Primer daño tardío' },
    { map: 'Dust II', round: 4, note: 'Apertura correcta tras flash alta', impact: '+1 ventaja inicial' },
  ],
  maxPoints: 400,
  completed: false,
};

const SUPPORT_MISSIONS = [
  {
    id: 'flash-conversion',
    role: 'support',
    category: 'Utilidad',
    title: 'Convierte 6 flashes en ventaja real',
    diagnosis: 'Ciegas rivales, pero tu equipo entra tarde en el espacio que generas.',
    coachNote: 'Lanza la flash con una cuenta clara y entra antes de que el rival recupere visión.',
    metricLabel: 'Asistencias de flash',
    baseline: 0,
    current: 6,
    target: 6,
    unit: 'count',
    evidenceMatches: 5,
    evidence: [{ map: 'Inferno', round: 12, note: 'Flash efectiva sin entrada posterior', impact: '2.1 s desaprovechados' }],
    maxPoints: 300,
    completed: true,
  },
  {
    id: 'counter-strafe',
    role: 'support',
    category: 'Mecánicas',
    title: 'Reduce el error al detenerte antes de disparar',
    diagnosis: 'Tu primera bala pierde precisión por velocidad residual en los picos laterales.',
    coachNote: 'Practica bloques cortos: strafe, parada completa, ráfaga de 2 balas.',
    metricLabel: 'Error de counter-strafe',
    baseline: 18,
    current: 16.3,
    target: 8,
    unit: '%',
    evidenceMatches: 7,
    evidence: [{ map: 'Nuke', round: 16, note: 'Disparo a 31 u/s de velocidad', impact: 'Primera bala desviada' }],
    maxPoints: 300,
    completed: false,
  },
];

export const REROLL_REPLACEMENT = {
  id: 'economy-discipline',
  category: 'Economía',
  title: 'Protege 3 rondas de economía frágil',
  diagnosis: 'Tus compras reactivas reducen la calidad del siguiente full buy.',
  coachNote: 'Si el equipo no alcanza utilidad completa, guarda el rifle y sincroniza la compra.',
  metricLabel: 'Compras disciplinadas',
  baseline: 0,
  current: 0,
  target: 3,
  unit: 'count',
  evidenceMatches: 6,
  evidence: [{ map: 'Anubis', round: 15, note: 'Force buy aislada tras perder pistola', impact: '-$2.900 siguiente ronda' }],
  completed: false,
};

const createBaseData = (userName) => {
  const missions = [PRIMARY_MISSION, ...SUPPORT_MISSIONS];
  const points = getWeeklyScore(missions);
  return {
    user: {
      id: 'current-user',
      displayName: userName,
      alias: userName,
      isOptedIn: true,
      isAnonymous: false,
      rank: 6,
      points,
      hasReroll: true,
      scoringMatches: 7,
    },
    season: {
      id: '2026-W32',
      label: 'Semana 32',
      league: 'DELTA–07',
      resetAt: getNextWeeklyReset(),
      maxScoringMatches: 10,
      rewards: [
        { rank: 1, label: '14 días Pro' },
        { rank: 2, label: '7 días Pro' },
        { rank: 3, label: '3 días Pro' },
      ],
    },
    missions,
    leaderboard: createLeaderboard(userName, points),
    recentMatches: [
      { id: 'm-1', map: 'Mirage', delta: 42, result: 'Victoria' },
      { id: 'm-2', map: 'Ancient', delta: 27, result: 'Derrota' },
      { id: 'm-3', map: 'Inferno', delta: 64, result: 'Victoria' },
      { id: 'm-4', map: 'Nuke', delta: 18, result: 'Derrota' },
      { id: 'm-5', map: 'Dust II', delta: 51, result: 'Victoria' },
    ],
    recap: {
      season: 'Semana 31',
      finalRank: 4,
      points: 826,
      missionsCompleted: 2,
      headline: 'Tu disciplina táctica subió un nivel',
      improvement: 'Redujiste un 23% las muertes sin intercambio.',
      reward: 'Insignia Operador Preciso',
    },
  };
};

const createNewPlayerData = (data) => ({
  ...data,
  user: { ...data.user, isOptedIn: false, rank: null, points: 0, scoringMatches: 1 },
  season: { ...data.season, league: 'LIGA PROVISIONAL' },
  missions: data.missions.map((mission) => ({ ...mission, current: mission.baseline })),
  leaderboard: createLeaderboard(data.user.displayName, 0, 50),
});

const createPodiumData = (data) => ({
  ...data,
  user: { ...data.user, rank: 1, points: 1000, hasReroll: false },
  missions: data.missions.map((mission) => ({ ...mission, current: mission.target, completed: true })),
  leaderboard: createLeaderboard(data.user.displayName, 1000, 1),
});

const createOutsideData = (data) => ({
  ...data,
  user: { ...data.user, rank: 27, points: 338 },
  leaderboard: createLeaderboard(data.user.displayName, 338, 27),
});

export const getProgressPrototypeData = (scenario = 'near-podium', userName = 'Tu perfil') => {
  const baseData = createBaseData(userName);
  if (scenario === 'new') return createNewPlayerData(baseData);
  if (scenario === 'podium') return createPodiumData(baseData);
  if (scenario === 'outside') return createOutsideData(baseData);
  return baseData;
};

export const syncUserScore = (data, missions) => {
  const points = getWeeklyScore(missions);
  const leaderboard = data.leaderboard
    .map((entry) => (entry.isCurrentUser ? { ...entry, points } : entry))
    .sort((a, b) => b.points - a.points)
    .map((entry, index) => ({ ...entry, rank: index + 1 }));
  const currentPlayer = leaderboard.find((entry) => entry.isCurrentUser);
  return {
    ...data,
    missions,
    user: { ...data.user, points, rank: currentPlayer?.rank ?? data.user.rank },
    leaderboard,
  };
};

export const getPodiumGap = (data) => {
  const third = data.leaderboard.find((entry) => entry.rank === 3);
  return Math.max((third?.points ?? 0) - data.user.points, 0);
};

export const getNextRankGap = (data) => {
  if (!data.user.rank || data.user.rank <= 1) return 0;
  const next = data.leaderboard.find((entry) => entry.rank === data.user.rank - 1);
  return Math.max((next?.points ?? 0) - data.user.points, 0);
};

export const getMissionDisplayPoints = (mission) => getMissionPoints(mission);
