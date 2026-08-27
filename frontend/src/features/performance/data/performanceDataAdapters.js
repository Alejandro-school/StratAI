const MAP_PRESENTATION = {
  de_mirage: { name: 'Mirage', radar: '/maps/de_mirage_radar_psd.webp' },
  de_inferno: { name: 'Inferno', radar: '/maps/de_inferno_radar_psd.webp' },
  de_anubis: { name: 'Anubis', radar: '/maps/de_anubis_radar_psd.webp' },
  de_nuke: { name: 'Nuke', radar: '/maps/de_nuke_radar_psd.webp' },
  de_dust2: { name: 'Dust II', radar: '/maps/de_dust2_radar_transparent.png' },
  de_ancient: { name: 'Ancient', radar: '/maps/de_ancient_radar_psd.webp' },
  de_cache: { name: 'Cache', radar: '/maps/de_cache_radar_psd.png' },
  de_train: { name: 'Train', radar: '/maps/de_train_radar_psd.webp' },
  de_overpass: { name: 'Overpass', radar: '/maps/de_overpass_radar_psd.webp' },
  de_vertigo: { name: 'Vertigo', radar: '/maps/de_vertigo_radar_psd.webp' },
};

const BUY_LABELS = {
  full_buy: 'Full buy',
  partial_buy: 'Compra parcial',
  eco: 'Eco',
  force_buy: 'Force buy',
};

const number = (value, fallback = 0) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const round = (value, precision = 0) => Number(number(value).toFixed(precision));
const perSample = (value, samples, precision = 1) => (
  samples > 0 ? round(number(value) / samples, precision) : 0
);
const percent = (value, total, precision = 1) => (
  total > 0 ? round(number(value) / total * 100, precision) : 0
);
const clamp = (value, min = 0, max = 100) => Math.min(max, Math.max(min, number(value)));
const formatMoney = (value) => `€${Math.round(number(value)).toLocaleString('es-ES')}`;

const createHistory = (history = []) => (
  [...history]
    .slice(0, 12)
    .reverse()
    .map((match, index) => ({
      id: match.match_id || index,
      value: number(match.hltv_rating),
      result: match.result || 'L',
      map: match.map,
      date: match.date,
    }))
);

const calculateTrend = (history) => {
  if (history.length < 4) return 0;
  const midpoint = Math.floor(history.length / 2);
  const previous = history.slice(0, midpoint);
  const recent = history.slice(midpoint);
  const average = (values) => values.reduce((sum, item) => sum + item.value, 0) / values.length;
  const baseline = average(previous);
  return baseline > 0 ? round((average(recent) - baseline) / baseline * 100, 1) : 0;
};

const contextFromOverview = (payload) => {
  const overview = payload?.overview || {};
  const sides = payload?.sides || {};
  const history = createHistory(payload?.match_history);
  return {
    id: 'general',
    name: 'Todos los mapas',
    matches: number(overview.total_matches),
    wins: number(overview.wins),
    losses: number(overview.losses),
    winRate: number(overview.win_rate),
    rating: number(overview.hltv_rating),
    adr: number(overview.adr),
    kast: number(overview.kast),
    kd: number(overview.kd_ratio),
    impact: number(overview.impact_rating),
    trend: calculateTrend(history),
    ct: {
      rating: number(sides.ct_rating),
      adr: number(sides.ct_adr),
    },
    t: {
      rating: number(sides.t_rating),
      adr: number(sides.t_adr),
    },
    history,
  };
};

const contextFromMap = (map) => {
  const presentation = MAP_PRESENTATION[map.map] || {
    name: map.map?.replace(/^de_/, '').replace(/^\w/, (letter) => letter.toUpperCase()) || 'Mapa',
    radar: `/maps/${map.map}_radar_psd.webp`,
  };
  return {
    id: map.map,
    name: presentation.name,
    radar: presentation.radar,
    matches: number(map.matches),
    wins: number(map.wins),
    losses: number(map.losses),
    winRate: number(map.win_rate),
    rating: number(map.avg_rating),
    adr: number(map.avg_adr),
    kast: number(map.avg_kast),
    kd: number(map.avg_kd),
    impact: number(map.avg_impact),
    trend: 0,
    ct: {
      rating: number(map.sides?.ct_rating),
      adr: number(map.sides?.ct_adr),
    },
    t: {
      rating: number(map.sides?.t_rating),
      adr: number(map.sides?.t_adr),
    },
    history: [],
    dominantWeapon: map.dominant_weapon,
  };
};

export const buildPerformanceContexts = (payload) => [
  contextFromOverview(payload),
  ...(payload?.maps || []).map(contextFromMap),
];

export const hydrateSelectedContext = (context, payload) => {
  if (!payload?.overview?.total_matches) return context;
  const actual = contextFromOverview(payload);
  return {
    ...context,
    matches: actual.matches,
    wins: actual.wins,
    losses: actual.losses,
    winRate: actual.winRate,
    rating: actual.rating,
    adr: actual.adr,
    kast: actual.kast,
    kd: actual.kd,
    impact: actual.impact,
    trend: actual.trend,
    ct: actual.ct,
    t: actual.t,
    history: actual.history,
  };
};

const buildOverviewDetail = (payload) => {
  const overview = payload.overview || {};
  const combat = payload.combat || {};
  const weapons = payload.weapons || [];
  const matches = number(overview.total_matches);
  const openings = number(combat.opening_duels_attempted);
  const clutches = Object.values(combat.clutches || {}).reduce((sum, value) => sum + number(value), 0);
  const favoriteWeapon = weapons[0] || {};

  return {
    metrics: [
      { label: 'Bajas', value: number(overview.kills), note: `${perSample(overview.kills, matches)} por partida` },
      { label: 'Headshots', value: `${round(overview.hs_pct, 1)}%`, note: `${number(overview.headshots)} bajas` },
      { label: 'Asistencias', value: number(overview.assists), note: `${perSample(overview.assists, matches)} por partida` },
      {
        label: 'Aperturas',
        value: `${round(combat.opening_success_rate, 1)}%`,
        note: `${number(combat.opening_duels_won)} ganadas / ${number(combat.opening_duels_lost)} perdidas`,
      },
      { label: 'Trades', value: number(combat.trade_kills), note: `${perSample(combat.trade_kills, matches, 2)} por partida` },
      { label: 'Clutches ganados', value: clutches, note: 'Suma de situaciones 1vX convertidas' },
    ],
    favoriteWeapon: {
      name: favoriteWeapon.weapon || 'Sin datos',
      kills: number(favoriteWeapon.kills),
      damagePerKill: favoriteWeapon.kills
        ? round(number(favoriteWeapon.damage) / number(favoriteWeapon.kills), 1)
        : 0,
      accuracy: number(favoriteWeapon.accuracy),
      headshots: number(favoriteWeapon.hs_pct),
    },
    multikills: Object.entries(combat.multikills || {}).map(([label, value]) => ({
      label: label.toUpperCase(),
      value: number(value),
    })),
    samples: { openings },
  };
};

const buildUtilityDetail = (payload) => {
  const utility = payload.utility || {};
  const overview = payload.overview || {};
  const combat = payload.combat || {};
  const rounds = number(payload.economy?.total_rounds || payload.economy?.rounds);
  const grenadesTotal = number(utility.grenades_thrown_total);

  return {
    metrics: [
      { label: 'Granadas usadas', value: grenadesTotal, note: `${perSample(grenadesTotal, overview.total_matches)} por partida` },
      { label: 'Daño de utilidad', value: number(utility.utility_damage), note: `${perSample(utility.utility_damage, rounds)} por ronda` },
      { label: 'Asistencias de flash', value: number(combat.flash_assists), note: `${perSample(combat.flash_assists, overview.total_matches)} por partida` },
      { label: 'Enemigos cegados', value: number(utility.enemies_flashed_total), note: `${number(utility.enemies_flashed_per_flash)} por flash` },
    ],
    grenades: [
      {
        name: 'Flashbang',
        used: number(utility.flashes_thrown),
        primary: number(utility.enemies_flashed_per_flash).toFixed(2),
        primaryLabel: 'enemigos / flash',
        secondary: `${number(utility.blind_time_per_flash).toFixed(2)} s de ceguera`,
        score: clamp(number(utility.enemies_flashed_per_flash) / 2 * 100),
        scoreLabel: 'Impacto',
      },
      {
        name: 'HE Grenade',
        used: number(utility.he_thrown),
        primary: number(utility.he_damage_per_nade).toFixed(1),
        primaryLabel: 'daño / granada',
        secondary: `${number(utility.grenade_damage?.he)} de daño total`,
        score: clamp(number(utility.he_damage_per_nade) / 40 * 100),
        scoreLabel: 'Daño',
      },
      {
        name: 'Molotov',
        used: number(utility.molotovs_thrown),
        primary: number(utility.molotov_damage_per_nade).toFixed(1),
        primaryLabel: 'daño / granada',
        secondary: `${number(utility.grenade_damage?.molotov)} de daño total`,
        score: clamp(number(utility.molotov_damage_per_nade) / 35 * 100),
        scoreLabel: 'Daño',
      },
      {
        name: 'Smoke Grenade',
        used: number(utility.smokes_thrown),
        primary: `${percent(utility.smokes_thrown, grenadesTotal)}%`,
        primaryLabel: 'del uso de utilidad',
        secondary: 'Uso registrado por Go',
        score: percent(utility.smokes_thrown, grenadesTotal),
        scoreLabel: 'Cuota de uso',
      },
    ],
  };
};

const buildWeaponsDetail = (payload) => {
  const aim = payload.aim || {};
  const overview = payload.overview || {};
  const weapons = payload.weapons || [];
  const totalWeaponKills = weapons.reduce((sum, weapon) => sum + number(weapon.kills), 0);
  const totalHits = number(aim.shots_hit);
  const bodyHits = aim.body_part_hits || {};
  const distributionTotal = Object.values(bodyHits).reduce((sum, value) => sum + number(value), 0);
  const hitValue = (...keys) => keys.reduce((sum, key) => sum + number(bodyHits[key]), 0);

  return {
    metrics: [
      { label: 'Disparos', value: number(aim.shots_fired), note: `${perSample(aim.shots_fired, overview.total_matches)} por partida` },
      { label: 'Impactos', value: totalHits, note: `${number(aim.accuracy_overall)}% de precisión` },
      { label: 'Daño / impacto', value: totalHits ? round(number(overview.total_damage) / totalHits, 1) : 0, note: 'todas las armas' },
      { label: 'Armas registradas', value: weapons.length, note: `${totalWeaponKills} bajas atribuidas` },
    ],
    rows: weapons.slice(0, 8).map((weapon) => ({
      name: weapon.weapon,
      kills: number(weapon.kills),
      damagePerKill: weapon.kills ? round(number(weapon.damage) / number(weapon.kills), 1) : 0,
      accuracy: number(weapon.accuracy),
      hs: number(weapon.hs_pct),
      damage: number(weapon.damage),
      usage: percent(weapon.kills, totalWeaponKills),
    })),
    hitDistribution: [
      { label: 'Cabeza', value: percent(hitValue('head'), distributionTotal) },
      { label: 'Pecho', value: percent(hitValue('chest'), distributionTotal) },
      { label: 'Estómago', value: percent(hitValue('stomach'), distributionTotal) },
      { label: 'Brazos', value: percent(hitValue('left_arm', 'right_arm'), distributionTotal) },
      { label: 'Piernas', value: percent(hitValue('left_leg', 'right_leg'), distributionTotal) },
    ],
  };
};

const encounterFlags = (encounter) => [
  encounter.openings ? `${encounter.openings} aperturas` : null,
  encounter.through_smoke ? `${encounter.through_smoke} a través de humo` : null,
  encounter.wallbangs ? `${encounter.wallbangs} wallbangs` : null,
  encounter.trades ? `${encounter.trades} trades` : null,
  encounter.user_blind ? `${encounter.user_blind} cegado` : null,
].filter(Boolean);

const buildDuelsDetail = (payload) => {
  const duels = payload.duels || {};
  const mechanics = payload.mechanics || {};
  const decisive = number(duels.kills_won) + number(duels.kills_lost);
  return {
    metrics: [
      { label: 'Duelos registrados', value: number(duels.total), note: `${decisive} terminaron en baja` },
      { label: 'Duelos ganados', value: `${number(duels.win_rate)}%`, note: `${number(duels.kills_won)} / ${decisive}` },
      { label: 'Primer daño', value: `${Math.round(number(mechanics.time_to_first_damage_avg_ms))} ms`, note: 'media en enfrentamientos' },
      { label: 'Precisión en duelo', value: `${number(mechanics.accuracy)}%`, note: `${number(mechanics.hits)} impactos registrados` },
    ],
    encounters: (duels.encounters || []).map((encounter) => {
      const result = encounter.wins === encounter.losses
        ? 'Igualado'
        : encounter.wins > encounter.losses ? 'Ganado' : 'Perdido';
      return {
        rival: encounter.name,
        result,
        score: `${encounter.wins}–${encounter.losses}`,
        weapon: `${encounter.user_weapon} vs ${encounter.rival_weapon}`,
        shots: `${encounter.user_shots} / ${encounter.rival_shots}`,
        accuracy: `${encounter.user_accuracy}% / ${encounter.rival_accuracy}%`,
        firstDamage: `${Math.round(encounter.user_first_damage_ms)} / ${Math.round(encounter.rival_first_damage_ms)} ms`,
        context: encounter.area,
        flags: encounterFlags(encounter),
      };
    }),
  };
};

const buildMechanicsDetail = (payload) => {
  const aim = payload.aim || {};
  const mechanics = payload.mechanics || {};
  return {
    metrics: [
      { label: 'Tiempo de reacción', value: `${Math.round(number(aim.reaction_time_avg_ms || mechanics.reaction_time_avg_ms))} ms`, note: 'media registrada' },
      { label: 'Tiempo hasta daño', value: `${Math.round(number(aim.time_to_damage_avg_ms || mechanics.time_to_first_damage_avg_ms))} ms`, note: 'visión → impacto' },
      { label: 'Error de crosshair', value: `${round(aim.crosshair_placement_avg_error || mechanics.crosshair_error_avg, 1)}°`, note: 'media al detectar rival' },
      { label: 'Duelos en estático', value: `${number(mechanics.stationary_pct)}%`, note: 'velocidad inferior a 75 u/s' },
      { label: 'Duelos en movimiento', value: `${number(mechanics.moving_pct)}%`, note: 'velocidad superior a 75 u/s' },
      { label: 'Precisión global', value: `${number(aim.accuracy_overall)}%`, note: `${number(aim.shots_hit)} impactos` },
    ],
    exposure: [
      { label: 'Enfrentamientos en estático', value: number(mechanics.stationary_pct) },
      { label: 'Duelos manteniendo ángulo', value: number(mechanics.hold_pct) },
      { label: 'Duelos iniciados con peek', value: number(mechanics.peek_pct) },
      { label: 'Impactos sin estar cegado', value: 100 - number(mechanics.blind_pct) },
    ],
    states: [
      { label: 'En movimiento', value: `${number(mechanics.moving_pct)}%`, note: 'de los enfrentamientos' },
      { label: 'Agachado', value: `${number(mechanics.ducking_pct)}%`, note: 'de los enfrentamientos' },
      { label: 'Cegado', value: `${number(mechanics.blind_pct)}%`, note: 'al participar en el duelo' },
      { label: 'A través de humo', value: `${number(mechanics.through_smoke_pct)}%`, note: 'de los duelos registrados' },
      { label: 'Wallbang', value: `${number(mechanics.wallbang_pct)}%`, note: 'de los duelos registrados' },
    ],
  };
};

const buildEconomyDetail = (payload) => {
  const economy = payload.economy || {};
  return {
    metrics: [
      { label: 'Gasto por ronda', value: formatMoney(economy.avg_spent_per_round), note: `${formatMoney(economy.total_spent)} invertidos` },
      { label: 'Equipo medio', value: formatMoney(economy.avg_equipment_value), note: 'valor al cerrar la compra' },
      { label: 'Equipo conservado', value: formatMoney(economy.saved_equipment_value), note: `${number(economy.survived_rounds)} rondas sobrevividas` },
      { label: 'Conversión tras save', value: `${number(economy.save_conversion_rate)}%`, note: 'ronda siguiente ganada' },
    ],
    buyTypes: (economy.buy_types || []).map((buy) => ({
      label: BUY_LABELS[buy.type] || buy.type,
      value: number(buy.share),
      winRate: number(buy.win_rate),
      rounds: number(buy.rounds),
    })),
    decisions: [
      { label: 'Supervivencia por ronda', value: `${number(economy.survival_rate)}%`, tone: 'positive' },
      { label: 'Dinero medio al terminar', value: formatMoney(economy.avg_money_after_round), tone: 'neutral' },
      { label: 'Equilibrio monetario del equipo', value: `${number(economy.team_money_gini).toFixed(3)} Gini`, tone: 'neutral' },
      { label: 'Rondas económicas registradas', value: number(economy.rounds), tone: 'neutral' },
    ],
  };
};

export const buildPerformanceDetail = (payload) => ({
  overview: buildOverviewDetail(payload),
  utility: buildUtilityDetail(payload),
  weapons: buildWeaponsDetail(payload),
  duels: buildDuelsDetail(payload),
  mechanics: buildMechanicsDetail(payload),
  economy: buildEconomyDetail(payload),
});

export const mapPresentation = (mapId) => (
  MAP_PRESENTATION[mapId] || {
    name: mapId?.replace(/^de_/, '') || 'Mapa',
    radar: `/maps/${mapId}_radar_psd.webp`,
  }
);
