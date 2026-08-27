import { useEffect, useMemo, useState } from 'react';
import {
  Activity,
  ArrowUpRight,
  Bomb,
  ChevronLeft,
  ChevronRight,
  Coins,
  Crosshair,
  Eye,
  Flame,
  Gauge,
  MousePointer2,
  Play,
  Shield,
  Skull,
  Sparkles,
  Target,
  Trophy,
  UserRound,
  Zap,
} from 'lucide-react';
import {
  formatDecimal,
  formatInteger,
  formatPercent,
  formatWeaponName,
  getWeaponIconPath,
} from '../../utils/performanceFormatters';

const numeric = (value, fallback = 0) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const isWin = (round) => ['win', 'won', 'victory'].includes(String(round?.user_outcome || '').toLowerCase());

const playerSide = (round) => round?.user_economy?.team || '';

const opponentSide = (side) => (side === 'CT' ? 'T' : 'CT');

const teamEquipment = (round) => {
  const side = playerSide(round);
  return {
    mine: numeric(round?.team_equipment?.[side]),
    theirs: numeric(round?.team_equipment?.[opponentSide(side)]),
  };
};

const ratingLabel = (rating) => {
  if (rating >= 1.35) return 'Partida dominante';
  if (rating >= 1.12) return 'Impacto positivo';
  if (rating >= 0.92) return 'Partida estable';
  return 'Partida por corregir';
};

const ratingTone = (rating) => {
  if (rating >= 1.12) return 'positive';
  if (rating >= 0.92) return 'neutral';
  return 'negative';
};

const trend = (value, good, warning, reverse = false) => {
  const score = reverse ? -value : value;
  const goodScore = reverse ? -good : good;
  const warningScore = reverse ? -warning : warning;
  if (score >= goodScore) return 'positive';
  if (score >= warningScore) return 'neutral';
  return 'negative';
};

const initials = (name) => String(name || 'Jugador').trim().slice(0, 2).toUpperCase();

export function PremierBadge({ profile }) {
  const rating = numeric(profile?.premier_rating, null);
  if (rating !== null && rating > 0) {
    return (
      <span className="md-premier" title="Rating Premier">
        <Sparkles size={10} />
        {formatInteger(rating)}
      </span>
    );
  }
  return (
    <span className="md-premier is-muted" title="Rating privado, sin calibrar o no disponible">
      Premier —
    </span>
  );
}

export function PlayerIdentity({ player, profile, isCurrent, isLeader, onSelect }) {
  return (
    <button
      type="button"
      className="md-player"
      onClick={() => onSelect?.(player)}
      title={isCurrent ? 'Abrir tu análisis' : `Analizar a ${player?.name || 'este jugador'}`}
    >
      <span className="md-player-avatar">
        {profile?.avatar
          ? <img src={profile.avatar} alt="" loading="lazy" />
          : initials(player?.name)}
      </span>
      <span className="md-player-copy">
        <span className="md-player-name">
          {isLeader && <Trophy size={11} aria-label="Mejor valoración del equipo" />}
          <strong>{player?.name || 'Jugador'}</strong>
          {isCurrent && <em>Tú</em>}
        </span>
        <PremierBadge profile={profile} />
      </span>
      <ArrowUpRight size={13} />
    </button>
  );
}

function buildBrief(rounds, userStats, isVictory) {
  const rating = numeric(userStats?.hltv_rating);
  const sideLostWithEdge = rounds.filter((round) => {
    const equipment = teamEquipment(round);
    return !isWin(round) && equipment.mine - equipment.theirs >= 4500;
  }).length;
  const steals = rounds.filter((round) => {
    const equipment = teamEquipment(round);
    return isWin(round) && equipment.theirs - equipment.mine >= 4500;
  }).length;
  const openingAttempts = numeric(userStats?.opening_duels_attempted);
  const openingWins = numeric(userStats?.opening_duels_won);
  const openingRate = openingAttempts ? (openingWins / openingAttempts) * 100 : null;
  const utilityDamage = numeric(userStats?.utility_damage);

  const verdict = rating >= 1.12
    ? `Tu impacto individual estuvo por encima del resultado: ${formatDecimal(rating, 2)} de valoración.`
    : rating < 0.92
      ? 'El problema principal estuvo en tu impacto individual, no solo en el marcador.'
      : isVictory
        ? 'Partida resuelta con una aportación estable y sin grandes picos.'
        : 'Partida ajustada en impacto: faltó convertir mejor los momentos importantes.';

  const signals = [];
  if (sideLostWithEdge > 0) {
    signals.push({
      tone: 'negative',
      label: 'Ventajas desperdiciadas',
      value: `${sideLostWithEdge}`,
      copy: `${sideLostWithEdge === 1 ? 'Una ronda perdida' : `${sideLostWithEdge} rondas perdidas`} con una ventaja clara de equipamiento.`,
    });
  }
  if (steals > 0) {
    signals.push({
      tone: 'positive',
      label: 'Rondas robadas',
      value: `${steals}`,
      copy: `${steals === 1 ? 'Una ronda ganada' : `${steals} rondas ganadas`} partiendo con peor compra que el rival.`,
    });
  }
  signals.push({
    tone: openingRate === null ? 'neutral' : trend(openingRate, 55, 40),
    label: 'Primer contacto',
    value: openingRate === null ? '—' : formatPercent(openingRate, 0),
    copy: openingAttempts
      ? `${formatInteger(openingWins)} de ${formatInteger(openingAttempts)} duelos iniciales ganados.`
      : 'No hay una muestra suficiente de aperturas.',
  });
  signals.push({
    tone: trend(utilityDamage, 70, 30),
    label: 'Utilidad',
    value: formatInteger(utilityDamage),
    copy: utilityDamage
      ? 'Daño total generado con granadas durante la partida.'
      : 'No se registró daño de utilidad.',
  });

  return { verdict, signals: signals.slice(0, 3) };
}

export function MatchBrief({ rounds = [], userStats = {}, isVictory, score }) {
  const brief = useMemo(
    () => buildBrief(rounds, userStats, isVictory),
    [isVictory, rounds, userStats],
  );
  const wins = rounds.filter(isWin).length;

  return (
    <section className="md-section md-brief" role="tabpanel" id="match-panel-overview">
      <div className="md-brief-lead">
        <span className="md-eyebrow">Lectura de la partida</span>
        <h2>{brief.verdict}</h2>
        <p>
          El objetivo no es repetir el marcador, sino aislar las decisiones que realmente explican la partida.
        </p>
      </div>

      <div className="md-signal-list">
        {brief.signals.map((signal) => (
          <article className={`md-signal is-${signal.tone}`} key={signal.label}>
            <span>{signal.label}</span>
            <strong>{signal.value}</strong>
            <p>{signal.copy}</p>
          </article>
        ))}
      </div>

      {rounds.length > 0 && (
        <div className="md-match-pulse">
          <div>
            <span>Pulso</span>
            <strong>{score}</strong>
          </div>
          <div className="md-round-strip" aria-label={`${wins} rondas ganadas de ${rounds.length}`}>
            {rounds.map((round) => (
              <span
                key={round.round}
                className={isWin(round) ? 'is-win' : 'is-loss'}
                title={`Ronda ${round.round}: ${isWin(round) ? 'ganada' : 'perdida'}`}
              />
            ))}
          </div>
          <small>{wins} ganadas · {rounds.length - wins} perdidas</small>
        </div>
      )}
    </section>
  );
}

const diagnosticsFor = (player) => {
  const rating = numeric(player?.hltv_rating);
  const adr = numeric(player?.adr);
  const kast = numeric(player?.kast);
  const openingAttempts = numeric(player?.opening_duels_attempted);
  const openingWins = numeric(player?.opening_duels_won);
  const openingRate = openingAttempts ? (openingWins / openingAttempts) * 100 : 0;
  const accuracy = numeric(player?.accuracy_overall);

  return [
    {
      label: 'Impacto',
      value: formatDecimal(rating, 2),
      progress: Math.min(100, (rating / 1.5) * 100),
      tone: ratingTone(rating),
      copy: rating >= 1.12 ? 'Influencia real por encima de la media.' : 'Faltó peso en las rondas disputadas.',
    },
    {
      label: 'Presión',
      value: `${formatInteger(adr)} ADR`,
      progress: Math.min(100, (adr / 120) * 100),
      tone: trend(adr, 85, 65),
      copy: adr >= 85 ? 'Daño constante antes de que la ronda se cierre.' : 'Poco daño útil antes de morir o guardar.',
    },
    {
      label: 'Continuidad',
      value: `${formatInteger(kast)}% KAST`,
      progress: Math.min(100, kast),
      tone: trend(kast, 72, 62),
      copy: kast >= 72 ? 'Participación fiable en la mayoría de rondas.' : 'Demasiadas rondas sin impacto contabilizable.',
    },
    {
      label: 'Aperturas',
      value: openingAttempts ? formatPercent(openingRate, 0) : 'Sin muestra',
      progress: openingAttempts ? openingRate : 0,
      tone: openingAttempts ? trend(openingRate, 55, 40) : 'neutral',
      copy: openingAttempts
        ? `${formatInteger(openingWins)} de ${formatInteger(openingAttempts)} primeros duelos.`
        : 'No tomó suficientes primeros contactos.',
    },
    ...(accuracy > 0 ? [{
      label: 'Precisión',
      value: formatPercent(accuracy, 1),
      progress: Math.min(100, (accuracy / 50) * 100),
      tone: trend(accuracy, 32, 23),
      copy: 'Impactos sobre los disparos registrados.',
    }] : []),
  ].slice(0, 4);
};

const hasValue = (value) => value !== null && value !== undefined && value !== '';

const show = (value, formatter = formatInteger) => (hasValue(value) ? formatter(value) : '—');

const totalClutches = (player) => [1, 2, 3, 4, 5]
  .reduce((sum, size) => sum + numeric(player?.[`clutches_1v${size}_won`]), 0);

const totalMultikills = (player) => Object.values(player?.multikills || {})
  .reduce((sum, value) => sum + numeric(value), 0);

const analysisGroups = (player) => [
  {
    title: 'Combate', icon: Crosshair, note: 'Volumen e impacto directo sobre el rival.',
    metrics: [
      ['Bajas', show(player?.kills)], ['Muertes', show(player?.deaths)],
      ['Asistencias', show(player?.assists)], ['K/D', show(player?.kd_ratio, (value) => formatDecimal(value, 2))],
      ['Daño total', show(player?.total_damage)], ['Impact rating', show(player?.impact_rating, (value) => formatDecimal(value, 2))],
    ],
  },
  {
    title: 'Mecánicas', icon: MousePointer2, note: 'Calidad del primer disparo, colocación y ejecución.',
    metrics: [
      ['Precisión', show(player?.accuracy_overall, (value) => formatPercent(value, 1))], ['Disparos', show(player?.shots_fired)],
      ['Impactos', show(player?.shots_hit)], ['Error de mira', show(player?.crosshair_placement_avg_error, (value) => `${formatDecimal(value, 1)}°`)],
      ['Tiempo a daño', show(player?.time_to_damage_avg_ms, (value) => `${formatInteger(value)} ms`)], ['Reacción', show(player?.avg_time_to_reaction, (value) => `${formatInteger(value)} ms`)],
    ],
  },
  {
    title: 'Rondas decisivas', icon: Trophy, note: 'Aperturas, intercambios y momentos de alta presión.',
    metrics: [
      ['Aperturas', `${show(player?.opening_duels_won)} / ${show(player?.opening_duels_attempted)}`], ['Trade kills', show(player?.trade_kills)],
      ['Muertes tradeadas', show(player?.traded_deaths)], ['Clutches', formatInteger(totalClutches(player))],
      ['Multibajas', formatInteger(totalMultikills(player))], ['Rondas vivo', show(player?.rounds_survived)],
    ],
  },
  {
    title: 'Utilidad', icon: Flame, note: 'Valor creado antes y durante los duelos.',
    metrics: [
      ['Daño de utilidad', show(player?.utility_damage)], ['Granadas', show(player?.grenades_thrown_total)],
      ['Enemigos cegados', show(player?.enemies_flashed_total)], ['Asist. flash', show(player?.flash_assists)],
      ['Daño / HE', show(player?.he_damage_per_nade, (value) => formatDecimal(value, 1))], ['Daño / molotov', show(player?.molotov_damage_per_nade, (value) => formatDecimal(value, 1))],
    ],
  },
];

const BODY_LABELS = {
  head: 'Cabeza', chest: 'Pecho', stomach: 'Abdomen', left_arm: 'Brazo izq.',
  right_arm: 'Brazo der.', left_leg: 'Pierna izq.', right_leg: 'Pierna der.', generic: 'Otros',
};

export function MatchAnalysis({ player, currentPlayer, profile, isCurrent }) {
  const diagnostics = diagnosticsFor(player);
  const groups = analysisGroups(player);
  const rating = numeric(player?.hltv_rating);
  const kills = numeric(player?.kills);
  const deaths = numeric(player?.deaths);
  const delta = kills - deaths;
  const comparison = currentPlayer && !isCurrent
    ? rating - numeric(currentPlayer.hltv_rating)
    : null;
  const bodyHits = Object.entries(player?.body_part_hits || {})
    .map(([key, value]) => ({ key, label: BODY_LABELS[key] || key, value: numeric(value) }))
    .filter((item) => item.value > 0)
    .sort((left, right) => right.value - left.value);
  const totalBodyHits = bodyHits.reduce((sum, item) => sum + item.value, 0);

  return (
    <section className="md-section md-analysis" role="tabpanel" id="match-panel-analysis">
      <header className="md-section-head">
        <div>
          <span className="md-eyebrow">{isCurrent ? 'Tu análisis' : 'Jugador seleccionado'}</span>
          <h2>{player?.name || 'Jugador'}</h2>
        </div>
        <PremierBadge profile={profile} />
      </header>

      <div className={`md-analysis-verdict is-${ratingTone(rating)}`}>
        <div className="md-rating-mark">
          <span>Rating</span>
          <strong>{formatDecimal(rating, 2)}</strong>
        </div>
        <div>
          <h3>{ratingLabel(rating)}</h3>
          <p>
            {delta >= 0
              ? `${formatInteger(kills)} bajas y un diferencial de +${formatInteger(delta)}.`
              : `${formatInteger(kills)} bajas y un diferencial de ${formatInteger(delta)}.`}
            {comparison !== null && ` Está ${comparison >= 0 ? '+' : ''}${formatDecimal(comparison, 2)} respecto a ti.`}
          </p>
        </div>
        <Gauge size={34} />
      </div>

      <div className="md-analysis-scoreline">
        {diagnostics.map((item, index) => (
          <article className={`md-diagnostic is-${item.tone}`} key={item.label}>
            <span className="md-diagnostic-index">0{index + 1}</span>
            <div className="md-diagnostic-copy">
              <span>{item.label}</span>
              <strong>{item.value}</strong>
              <p>{item.copy}</p>
            </div>
            <div className="md-diagnostic-track">
              <span style={{ width: `${item.progress}%` }} />
            </div>
          </article>
        ))}
      </div>

      <div className="md-analysis-ledger">
        {groups.map((group) => {
          const Icon = group.icon;
          return (
            <article className="md-ledger-group" key={group.title}>
              <header>
                <Icon size={16} />
                <div><strong>{group.title}</strong><span>{group.note}</span></div>
              </header>
              <dl>
                {group.metrics.map(([label, value]) => (
                  <div key={label}><dt>{label}</dt><dd>{value}</dd></div>
                ))}
              </dl>
            </article>
          );
        })}
      </div>

      {bodyHits.length > 0 && (
        <div className="md-hit-distribution">
          <header>
            <div><Eye size={16} /><strong>Distribución de impactos</strong></div>
            <span>{formatInteger(totalBodyHits)} impactos registrados</span>
          </header>
          <div>
            {bodyHits.map((item) => (
              <div className="md-hit-row" key={item.key}>
                <span>{item.label}</span>
                <div><i style={{ width: `${(item.value / totalBodyHits) * 100}%` }} /></div>
                <strong>{item.value}</strong>
                <small>{formatPercent((item.value / totalBodyHits) * 100, 0)}</small>
              </div>
            ))}
          </div>
        </div>
      )}

      <footer className="md-analysis-footer">
        <div><Target size={15} /><span>HS en bajas</span><strong>{formatInteger(player?.hs_percentage)}%</strong></div>
        <div><Shield size={15} /><span>KAST</span><strong>{formatInteger(player?.kast)}%</strong></div>
        <div><Zap size={15} /><span>ADR</span><strong>{formatInteger(player?.adr)}</strong></div>
      </footer>
    </section>
  );
}

const WIN_REASON_LABELS = {
  CTWin: 'Eliminación / tiempo agotado',
  TWin: 'Eliminación',
  TargetBombed: 'Bomba detonada',
  BombDefused: 'Bomba desactivada',
  TargetSaved: 'Objetivo defendido',
};

const purchaseName = (purchase) => purchase?.weapon || purchase?.name || 'Compra';

const roundBuyLabel = (value) => {
  const equipment = numeric(value);
  if (equipment >= 4000) return 'Compra completa';
  if (equipment >= 2500) return 'Compra media';
  if (equipment >= 1200) return 'Force buy';
  return 'Eco';
};

const roundNarrative = (round) => {
  const side = playerSide(round);
  const equipment = teamEquipment(round);
  const edge = equipment.mine - equipment.theirs;
  const won = isWin(round);
  const openingForUs = round?.opening?.killer_team === side;
  if (!won && edge >= 4500) return 'Ronda perdida pese a partir con una ventaja clara de equipamiento.';
  if (won && edge <= -4500) return 'Ronda robada con una compra sensiblemente inferior a la rival.';
  if (!won && openingForUs) return 'El equipo consiguió la primera baja, pero no convirtió la ventaja.';
  if (won && !openingForUs && round?.opening) return 'El equipo remontó la desventaja tras perder el primer duelo.';
  if (numeric(round?.user_combat?.kills) >= 2) return 'Tu impacto directo fue decisivo para resolver esta ronda.';
  return won ? 'Ronda convertida sin una anomalía económica importante.' : 'Ronda cedida sin una ventaja inicial clara.';
};

export function MatchRounds({ rounds = [], initialRound = 1, onOpenReplay }) {
  const [selectedNumber, setSelectedNumber] = useState(initialRound);

  useEffect(() => {
    if (!rounds.some((round) => round.round === selectedNumber)) {
      setSelectedNumber(rounds[0]?.round || 1);
    }
  }, [rounds, selectedNumber]);

  const selectedIndex = Math.max(0, rounds.findIndex((round) => round.round === selectedNumber));
  const round = rounds[selectedIndex];
  if (!round) {
    return (
      <section className="md-section md-empty" role="tabpanel" id="match-panel-rounds">
        <Activity size={26} /><h2>Sin detalle de rondas</h2><p>Esta demo no contiene datos ronda a ronda.</p>
      </section>
    );
  }

  const won = isWin(round);
  const side = playerSide(round);
  const equipment = teamEquipment(round);
  const combat = round.user_combat || {};
  const utility = round.user_utility || {};
  const economy = round.user_economy || {};
  const winsSoFar = rounds.slice(0, selectedIndex + 1).filter(isWin).length;
  const lossesSoFar = selectedIndex + 1 - winsSoFar;
  const purchases = economy.purchases || [];

  const move = (direction) => {
    const next = Math.max(0, Math.min(rounds.length - 1, selectedIndex + direction));
    setSelectedNumber(rounds[next].round);
  };

  return (
    <section className="md-section md-rounds" role="tabpanel" id="match-panel-rounds">
      <header className="md-section-head">
        <div><span className="md-eyebrow">Explorador de rondas</span><h2>Entiende qué cambió la partida</h2></div>
        <div className="md-round-nav">
          <button type="button" onClick={() => move(-1)} disabled={selectedIndex === 0}><ChevronLeft size={15} /></button>
          <span>Ronda {round.round} de {rounds.length}</span>
          <button type="button" onClick={() => move(1)} disabled={selectedIndex === rounds.length - 1}><ChevronRight size={15} /></button>
        </div>
      </header>

      <div className="md-round-filmstrip" aria-label="Seleccionar ronda">
        {rounds.map((item) => (
          <button
            type="button"
            key={item.round}
            className={`${isWin(item) ? 'is-win' : 'is-loss'} ${item.round === round.round ? 'is-active' : ''}`}
            onClick={() => setSelectedNumber(item.round)}
          >
            <span>{item.round}</span><i />
          </button>
        ))}
      </div>

      <div className="md-round-stage">
        <div className={`md-round-verdict ${won ? 'is-win' : 'is-loss'}`}>
          <span>Ronda {round.round} · lado {side || '—'}</span>
          <strong>{won ? 'Ganada' : 'Perdida'}</strong>
          <p>{roundNarrative(round)}</p>
          <small>{winsSoFar}:{lossesSoFar} tras esta ronda</small>
        </div>

        <div className="md-round-events">
          <article>
            <Crosshair size={16} /><span>Primera baja</span>
            <strong>{round.opening ? `${round.opening.killer} → ${round.opening.victim}` : 'Sin registro'}</strong>
            <small>{round.opening?.weapon || 'No disponible'}</small>
          </article>
          <article>
            <Skull size={16} /><span>Tu actuación</span>
            <strong>{formatInteger(combat.kills)} K · {formatInteger(combat.deaths)} D · {formatInteger(combat.damage)} daño</strong>
            <small>{combat.trade_kills ? `${combat.trade_kills} baja de intercambio` : economy.survived ? 'Sobreviviste' : 'Sin trade registrado'}</small>
          </article>
          <article>
            <Flame size={16} /><span>Tu utilidad</span>
            <strong>{formatInteger(utility.count)} granadas · {formatInteger(utility.damage)} daño</strong>
            <small>{(utility.types || []).join(' · ') || 'Sin utilidad registrada'}</small>
          </article>
          <article>
            <Bomb size={16} /><span>Resolución</span>
            <strong>{WIN_REASON_LABELS[round.win_reason] || round.win_reason || 'Sin detalle'}</strong>
            <small>{formatInteger(round.survivors?.[side])} supervivientes de tu lado</small>
          </article>
        </div>

        <aside className="md-round-loadout">
          <header><Coins size={16} /><div><span>Contexto económico</span><strong>{roundBuyLabel(economy.equipment_value)}</strong></div></header>
          <dl>
            <div><dt>Tu equipo</dt><dd>{formatInteger(equipment.mine)} $</dd></div>
            <div><dt>Rival</dt><dd>{formatInteger(equipment.theirs)} $</dd></div>
            <div><dt>Tu gasto</dt><dd>{formatInteger(economy.spent)} $</dd></div>
            <div><dt>Dinero final</dt><dd>{formatInteger(economy.final_money)} $</dd></div>
          </dl>
          <div className="md-purchase-list">
            <span>Compras</span>
            {purchases.length
              ? purchases.slice(0, 5).map((purchase, index) => <em key={`${purchaseName(purchase)}-${index}`}>{purchaseName(purchase)}</em>)
              : <small>Sin compras registradas</small>}
          </div>
          <button type="button" className="md-open-replay" onClick={() => onOpenReplay?.(round.round, round.opening?.tick)}>
            <Play size={14} /> Abrir ronda en replay
          </button>
        </aside>
      </div>
    </section>
  );
}

const normalizeWeapons = (weaponStats) => {
  if (!weaponStats || typeof weaponStats !== 'object') return [];
  return Object.entries(weaponStats)
    .map(([weapon, stats]) => {
      const shots = numeric(stats?.shots_fired ?? stats?.shots);
      const hits = numeric(stats?.shots_hit ?? stats?.hits);
      const kills = numeric(stats?.kills);
      const headshots = numeric(stats?.headshots);
      return {
        weapon,
        kills,
        damage: numeric(stats?.damage),
        shots,
        hits,
        headshots,
        accuracy: shots > 0 ? (hits / shots) * 100 : null,
        hsRate: kills > 0 ? (headshots / kills) * 100 : null,
      };
    })
    .filter((weapon) => weapon.kills > 0 || weapon.damage > 0 || weapon.shots > 0)
    .sort((left, right) => right.kills - left.kills || right.damage - left.damage);
};

export function MatchWeapons({ player }) {
  const weapons = useMemo(() => normalizeWeapons(player?.weapon_stats), [player?.weapon_stats]);
  const [selectedName, setSelectedName] = useState('');

  useEffect(() => {
    setSelectedName(weapons[0]?.weapon || '');
  }, [weapons]);

  const selected = weapons.find((weapon) => weapon.weapon === selectedName) || weapons[0];
  const totalKills = weapons.reduce((sum, weapon) => sum + weapon.kills, 0);

  if (!selected) {
    return (
      <section className="md-section md-empty" role="tabpanel" id="match-panel-weapons">
        <Crosshair size={26} />
        <h2>Sin datos de armamento</h2>
        <p>Esta demo no contiene disparos o daño desglosados por arma para {player?.name || 'este jugador'}.</p>
      </section>
    );
  }

  return (
    <section className="md-section md-armory" role="tabpanel" id="match-panel-weapons">
      <header className="md-section-head">
        <div>
          <span className="md-eyebrow">Armería de la partida</span>
          <h2>{player?.name || 'Jugador'}</h2>
        </div>
        <span className="md-armory-total">{formatInteger(totalKills)} bajas con armas</span>
      </header>

      <div className="md-armory-layout">
        <nav className="md-weapon-list" aria-label="Armas utilizadas">
          {weapons.map((weapon, index) => (
            <button
              type="button"
              key={weapon.weapon}
              className={weapon.weapon === selected.weapon ? 'is-active' : ''}
              onClick={() => setSelectedName(weapon.weapon)}
            >
              <span>{String(index + 1).padStart(2, '0')}</span>
              <strong>{formatWeaponName(weapon.weapon)}</strong>
              <em>{formatInteger(weapon.kills)} K</em>
            </button>
          ))}
        </nav>

        <div className="md-weapon-stage">
          <span className="md-weapon-watermark">{formatWeaponName(selected.weapon)}</span>
          <img src={getWeaponIconPath(selected.weapon)} alt={formatWeaponName(selected.weapon)} />
          <div className="md-weapon-title">
            <span>Arma seleccionada</span>
            <h3>{formatWeaponName(selected.weapon)}</h3>
          </div>
        </div>

        <aside className="md-weapon-specs">
          <div>
            <span>Bajas</span>
            <strong>{formatInteger(selected.kills)}</strong>
          </div>
          <div>
            <span>Daño</span>
            <strong>{formatInteger(selected.damage)}</strong>
          </div>
          <div>
            <span>Precisión</span>
            <strong>{selected.accuracy === null ? 'N/D' : formatPercent(selected.accuracy, 1)}</strong>
            <small>{selected.accuracy === null ? 'Sin disparos registrados' : `${formatInteger(selected.hits)} / ${formatInteger(selected.shots)} impactos`}</small>
          </div>
          <div>
            <span>HS en bajas</span>
            <strong>{selected.hsRate === null ? 'N/D' : formatPercent(selected.hsRate, 0)}</strong>
          </div>
        </aside>
      </div>
    </section>
  );
}

export function EmptyPlayerState() {
  return (
    <section className="md-section md-empty">
      <UserRound size={26} />
      <h2>Jugador no disponible</h2>
      <p>Selecciona otro jugador desde el marcador.</p>
    </section>
  );
}
