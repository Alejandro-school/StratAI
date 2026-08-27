import React from 'react';
import { ArrowDown, ArrowUp, ChevronRight, Crown, LockKeyhole, Radio, Trophy, Users } from 'lucide-react';

const PODIUM_LABELS = { 1: '14d', 2: '7d', 3: '3d' };

const getInitials = (name) => name.split(/\s+/).map((part) => part[0]).join('').slice(0, 2).toUpperCase();

const RankTrend = ({ value }) => (
  <span className={`op-rank-trend ${value < 0 ? 'is-down' : ''}`}>
    {value < 0 ? <ArrowDown size={11} aria-hidden="true" /> : <ArrowUp size={11} aria-hidden="true" />}
    {Math.abs(value)}
  </span>
);

const PodiumRow = ({ entry }) => (
  <div className={`op-podium-row op-podium-row--${entry.rank}`}>
    <span className="op-podium-row__rank">{entry.rank === 1 ? <Crown size={16} aria-label="Primer puesto" /> : `0${entry.rank}`}</span>
    <span className="op-player-orb" aria-hidden="true">{getInitials(entry.name)}</span>
    <span className="op-podium-row__name">{entry.name}</span>
    <strong>{entry.points}</strong>
    <span className="op-podium-row__reward">{PODIUM_LABELS[entry.rank]} Pro</span>
  </div>
);

const LeaguePanel = ({ data, nextRankGap, onOpenLeague, onOptIn, podiumGap }) => {
  const podium = data.leaderboard.slice(0, 3);
  const current = data.leaderboard.find((entry) => entry.isCurrentUser);
  const visibleName = data.user.isAnonymous ? 'Analista ••••' : data.user.alias;

  return (
    <aside className="op-league" aria-labelledby="league-title">
      <header className="op-league__header">
        <div>
          <span className="op-eyebrow"><Radio size={12} aria-hidden="true" /> Liga activa</span>
          <h2 id="league-title">{data.season.league}</h2>
        </div>
        <span className="op-league__population"><Users size={13} aria-hidden="true" /> 50</span>
      </header>

      {!data.user.isOptedIn ? (
        <div className="op-league-gate">
          <LockKeyhole size={22} aria-hidden="true" />
          <h3>Tu puesto aún es provisional</h3>
          <p>Activa la liga para aparecer con tu alias o competir de forma anónima.</p>
          <button type="button" onClick={onOptIn}>Entrar en la liga</button>
        </div>
      ) : (
        <>
          <div className="op-podium" aria-label="Podio semanal">
            {podium.map((entry) => <PodiumRow key={entry.id} entry={entry} />)}
          </div>

          <div className="op-your-rank">
            <div className="op-your-rank__topline"><span>Tu señal</span><RankTrend value={current?.trend ?? 0} /></div>
            <div className="op-your-rank__identity">
              <strong>#{data.user.rank}</strong>
              <span>{visibleName}</span>
              <b>{data.user.points} pts</b>
            </div>
            <div className="op-your-rank__gaps">
              <span><b>{nextRankGap}</b> pts al siguiente</span>
              <span><Trophy size={12} aria-hidden="true" /><b>{podiumGap}</b> pts al podio</span>
            </div>
          </div>
        </>
      )}

      <button type="button" className="op-league__open" onClick={onOpenLeague}>
        Ver clasificación completa <ChevronRight size={15} aria-hidden="true" />
      </button>

      <p className="op-league__fairness">
        <Trophy size={13} aria-hidden="true" /> Solo puntúan tus primeras {data.season.maxScoringMatches} partidas.
      </p>
    </aside>
  );
};

export default LeaguePanel;
