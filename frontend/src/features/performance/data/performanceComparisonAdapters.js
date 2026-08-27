import { mapPresentation } from './performanceDataAdapters';

export const COMPARISON_ATTRIBUTE_LABELS = [
  'Puntería',
  'Duelos iniciales',
  'Utilidad',
  'Posicionamiento',
  'Clutches',
  'Economía',
];

const number = (value, fallback = 0) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};
const clamp = (value) => Math.round(Math.min(100, Math.max(0, number(value))));
const average = (values) => values.reduce((sum, value) => sum + number(value), 0) / values.length;
const perSample = (value, samples) => (samples > 0 ? number(value) / samples : 0);
const buy = (payload, type) => (
  payload?.economy?.buy_types?.find((item) => item.type === type) || {}
);

const metric = (
  label,
  user,
  rival,
  {
    digits = 0,
    suffix = '',
    prefix = '',
    lowerIsBetter = false,
  } = {},
) => ({
  label,
  user: number(user),
  rival: number(rival),
  digits,
  suffix,
  prefix,
  lowerIsBetter,
});

const profileStreak = (history = []) => {
  let current = 0;
  let best = 0;
  history.forEach((match) => {
    current = match.result === 'W' ? current + 1 : 0;
    best = Math.max(best, current);
  });
  return best;
};

const attributesFor = (payload) => {
  const overview = payload.overview || {};
  const aim = payload.aim || {};
  const utility = payload.utility || {};
  const combat = payload.combat || {};
  const economy = payload.economy || {};
  const matches = number(overview.total_matches);
  const rounds = number(economy.rounds || economy.total_rounds);
  const clutchWins = Object.values(combat.clutches || {}).reduce((sum, value) => sum + number(value), 0);

  return [
    clamp(average([
      number(aim.accuracy_overall) / 45 * 100,
      number(overview.hs_pct) / 65 * 100,
      (1 - number(aim.time_to_damage_avg_ms) / 1200) * 100,
      (1 - number(aim.crosshair_placement_avg_error) / 25) * 100,
    ])),
    clamp(average([
      number(combat.opening_success_rate) / 70 * 100,
      perSample(combat.opening_duels_won, matches) / 4 * 100,
    ])),
    clamp(average([
      number(utility.enemies_flashed_per_flash) / 2 * 100,
      number(utility.he_damage_per_nade) / 40 * 100,
      perSample(utility.utility_damage, rounds) / 12 * 100,
    ])),
    clamp(average([
      number(overview.kast) / 90 * 100,
      number(economy.survival_rate) / 55 * 100,
      perSample(combat.trade_kills, matches) / 3 * 100,
    ])),
    clamp(perSample(clutchWins, matches) / 1.5 * 100),
    clamp(average([
      number(buy(payload, 'full_buy').win_rate),
      number(buy(payload, 'force_buy').win_rate),
      number(economy.survival_rate) / 55 * 100,
    ])),
  ];
};

const topSkills = (attributes) => (
  attributes
    .map((value, index) => ({ value, label: COMPARISON_ATTRIBUTE_LABELS[index] }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 2)
    .map((item) => item.label)
);

const buildProfile = (payload, fallbackName) => {
  const overview = payload.overview || {};
  const history = payload.match_history || [];
  const attributes = attributesFor(payload);
  const handle = payload.player?.name || fallbackName || 'Jugador';
  return {
    name: handle,
    handle,
    initials: handle.slice(0, 2).toUpperCase(),
    matches: number(overview.total_matches),
    highKills: Math.max(0, ...history.map((match) => number(match.kills))),
    streak: profileStreak([...history].reverse()),
    rating: number(overview.hltv_rating),
    winRate: number(overview.win_rate),
    attributes,
    topSkills: topSkills(attributes),
  };
};

const createAimSection = (user, rival) => ({
  id: 'aim',
  title: 'Puntería',
  subtitle: 'Velocidad, precisión y colocación de mira',
  scoreUser: attributesFor(user)[0],
  scoreRival: attributesFor(rival)[0],
  metrics: [
    metric('Tiempo hasta daño', user.aim?.time_to_damage_avg_ms, rival.aim?.time_to_damage_avg_ms, { suffix: ' ms', lowerIsBetter: true }),
    metric('Tiempo de reacción', user.aim?.reaction_time_avg_ms, rival.aim?.reaction_time_avg_ms, { suffix: ' ms', lowerIsBetter: true }),
    metric('Error de crosshair', user.aim?.crosshair_placement_avg_error, rival.aim?.crosshair_placement_avg_error, { digits: 1, suffix: '°', lowerIsBetter: true }),
    metric('Precisión global', user.aim?.accuracy_overall, rival.aim?.accuracy_overall, { digits: 1, suffix: '%' }),
    metric('Headshots', user.overview?.hs_pct, rival.overview?.hs_pct, { digits: 1, suffix: '%' }),
    metric('Duelos en estático', user.mechanics?.stationary_pct, rival.mechanics?.stationary_pct, { digits: 1, suffix: '%' }),
    metric('Disparos registrados', user.aim?.shots_fired, rival.aim?.shots_fired),
    metric('Impactos registrados', user.aim?.shots_hit, rival.aim?.shots_hit),
  ],
});

const createUtilitySection = (user, rival) => {
  const userMatches = number(user.overview?.total_matches);
  const rivalMatches = number(rival.overview?.total_matches);
  const userRounds = number(user.economy?.rounds || user.economy?.total_rounds);
  const rivalRounds = number(rival.economy?.rounds || rival.economy?.total_rounds);
  return {
    id: 'utility',
    title: 'Utilidad',
    subtitle: 'Valor real generado con granadas',
    scoreUser: attributesFor(user)[2],
    scoreRival: attributesFor(rival)[2],
    metrics: [
      metric('Granadas por partida', perSample(user.utility?.grenades_thrown_total, userMatches), perSample(rival.utility?.grenades_thrown_total, rivalMatches), { digits: 1 }),
      metric('Daño de utilidad por ronda', perSample(user.utility?.utility_damage, userRounds), perSample(rival.utility?.utility_damage, rivalRounds), { digits: 1 }),
      metric('Enemigos por flash', user.utility?.enemies_flashed_per_flash, rival.utility?.enemies_flashed_per_flash, { digits: 2 }),
      metric('Ceguera por flash', user.utility?.blind_time_per_flash, rival.utility?.blind_time_per_flash, { digits: 2, suffix: ' s' }),
      metric('Daño por HE', user.utility?.he_damage_per_nade, rival.utility?.he_damage_per_nade, { digits: 1 }),
      metric('Daño por molotov', user.utility?.molotov_damage_per_nade, rival.utility?.molotov_damage_per_nade, { digits: 1 }),
      metric('Asistencias de flash', user.combat?.flash_assists, rival.combat?.flash_assists),
      metric('Smokes lanzadas', user.utility?.smokes_thrown, rival.utility?.smokes_thrown),
    ],
  };
};

const createClutchSection = (user, rival) => {
  const userClutches = user.combat?.clutches || {};
  const rivalClutches = rival.combat?.clutches || {};
  return {
    id: 'clutching',
    title: 'Clutches',
    subtitle: 'Situaciones 1vX convertidas',
    scoreUser: attributesFor(user)[4],
    scoreRival: attributesFor(rival)[4],
    metrics: [
      metric('1 vs 1 ganados', userClutches['1v1'], rivalClutches['1v1']),
      metric('1 vs 2 ganados', userClutches['1v2'], rivalClutches['1v2']),
      metric('1 vs 3 ganados', userClutches['1v3'], rivalClutches['1v3']),
      metric('1 vs 4 ganados', userClutches['1v4'], rivalClutches['1v4']),
      metric('1 vs 5 ganados', userClutches['1v5'], rivalClutches['1v5']),
      metric('Rondas sobrevividas', user.economy?.survived_rounds, rival.economy?.survived_rounds),
    ],
  };
};

const createPositioningSection = (user, rival) => {
  const userMatches = number(user.overview?.total_matches);
  const rivalMatches = number(rival.overview?.total_matches);
  return {
    id: 'positioning',
    title: 'Posicionamiento y trades',
    subtitle: 'Supervivencia, bandos e intercambios',
    scoreUser: attributesFor(user)[3],
    scoreRival: attributesFor(rival)[3],
    metrics: [
      metric('KAST', user.overview?.kast, rival.overview?.kast, { digits: 1, suffix: '%' }),
      metric('Supervivencia', user.economy?.survival_rate, rival.economy?.survival_rate, { digits: 1, suffix: '%' }),
      metric('Rating CT', user.sides?.ct_rating, rival.sides?.ct_rating, { digits: 2 }),
      metric('Rating T', user.sides?.t_rating, rival.sides?.t_rating, { digits: 2 }),
      metric('ADR CT', user.sides?.ct_adr, rival.sides?.ct_adr, { digits: 1 }),
      metric('ADR T', user.sides?.t_adr, rival.sides?.t_adr, { digits: 1 }),
      metric('Trades por partida', perSample(user.combat?.trade_kills, userMatches), perSample(rival.combat?.trade_kills, rivalMatches), { digits: 2 }),
      metric('Muertes tradeadas', user.combat?.traded_deaths, rival.combat?.traded_deaths),
      metric('Duelos manteniendo ángulo', user.mechanics?.hold_pct, rival.mechanics?.hold_pct, { digits: 1, suffix: '%' }),
      metric('Duelos iniciados con peek', user.mechanics?.peek_pct, rival.mechanics?.peek_pct, { digits: 1, suffix: '%' }),
    ],
  };
};

const createOpeningSection = (user, rival) => ({
  id: 'openings',
  title: 'Duelos iniciales',
  subtitle: 'Frecuencia y éxito de las primeras peleas',
  scoreUser: attributesFor(user)[1],
  scoreRival: attributesFor(rival)[1],
  metrics: [
    metric('Intentos', user.combat?.opening_duels_attempted, rival.combat?.opening_duels_attempted),
    metric('Ganados', user.combat?.opening_duels_won, rival.combat?.opening_duels_won),
    metric('Perdidos', user.combat?.opening_duels_lost, rival.combat?.opening_duels_lost, { lowerIsBetter: true }),
    metric('Éxito', user.combat?.opening_success_rate, rival.combat?.opening_success_rate, { digits: 1, suffix: '%' }),
    metric('Trade kills', user.combat?.trade_kills, rival.combat?.trade_kills),
    metric('Muertes tradeadas', user.combat?.traded_deaths, rival.combat?.traded_deaths, { lowerIsBetter: true }),
  ],
});

const createEconomySection = (user, rival) => ({
  id: 'economy',
  title: 'Economía',
  subtitle: 'Compras, supervivencia y conversión',
  scoreUser: attributesFor(user)[5],
  scoreRival: attributesFor(rival)[5],
  metrics: [
    metric('Gasto medio por ronda', user.economy?.avg_spent_per_round, rival.economy?.avg_spent_per_round, { prefix: '€' }),
    metric('Valor medio de equipo', user.economy?.avg_equipment_value, rival.economy?.avg_equipment_value, { prefix: '€' }),
    metric('Full buys ganadas', buy(user, 'full_buy').win_rate, buy(rival, 'full_buy').win_rate, { digits: 1, suffix: '%' }),
    metric('Force buys ganadas', buy(user, 'force_buy').win_rate, buy(rival, 'force_buy').win_rate, { digits: 1, suffix: '%' }),
    metric('Ecos ganadas', buy(user, 'eco').win_rate, buy(rival, 'eco').win_rate, { digits: 1, suffix: '%' }),
    metric('Conversión tras save', user.economy?.save_conversion_rate, rival.economy?.save_conversion_rate, { digits: 1, suffix: '%' }),
    metric('Valor conservado', user.economy?.saved_equipment_value, rival.economy?.saved_equipment_value, { prefix: '€' }),
    metric('Gini monetario', user.economy?.team_money_gini, rival.economy?.team_money_gini, { digits: 3, lowerIsBetter: true }),
  ],
});

const weaponPairs = (user, rival) => {
  const userWeapons = user.weapons || [];
  const rivalWeapons = rival.weapons || [];
  const length = Math.max(3, Math.min(5, Math.max(userWeapons.length, rivalWeapons.length)));
  return Array.from({ length }, (_, index) => {
    const adapt = (weapon = {}) => ({
      name: weapon.weapon || 'Sin datos',
      kills: number(weapon.kills),
      damagePerKill: weapon.kills ? number(weapon.damage) / number(weapon.kills) : 0,
      accuracy: number(weapon.accuracy),
      headshots: number(weapon.hs_pct),
    });
    return { user: adapt(userWeapons[index]), rival: adapt(rivalWeapons[index]) };
  });
};

const mapRows = (user, rival) => {
  const userMaps = Object.fromEntries((user.maps || []).map((map) => [map.map, map]));
  const rivalMaps = Object.fromEntries((rival.maps || []).map((map) => [map.map, map]));
  return [...new Set([...Object.keys(userMaps), ...Object.keys(rivalMaps)])].map((contextId) => ({
    contextId,
    presentation: mapPresentation(contextId),
    user: userMaps[contextId] || { map: contextId },
    rival: rivalMaps[contextId] || { map: contextId },
  }));
};

export const buildComparisonDashboard = (user, rival, player, range) => ({
  range,
  user: buildProfile(user, 'Tú'),
  rival: buildProfile(rival, player?.name),
  maps: mapRows(user, rival),
  sections: [
    createAimSection(user, rival),
    createUtilitySection(user, rival),
    createClutchSection(user, rival),
    createPositioningSection(user, rival),
    createOpeningSection(user, rival),
    createEconomySection(user, rival),
  ],
  weapons: weaponPairs(user, rival),
  sources: {
    user: user.sources,
    rival: rival.sources,
  },
});
