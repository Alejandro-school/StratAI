import React, { Suspense, lazy, useCallback, useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  ArrowLeft,
  BarChart3,
  Clock3,
  Crosshair,
  LayoutGrid,
  ListVideo,
  Play,
  Shield,
  Swords,
  Trophy,
  Users,
} from 'lucide-react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import {
  EmptyPlayerState,
  MatchAnalysis,
  MatchBrief,
  MatchRounds,
  MatchWeapons,
  PlayerIdentity,
} from '../features/match-details/MatchDetailSections';
import { matchDetailsQueryOptions } from '../features/matches/queries/matchQueries';
import { getMapImage } from '../utils/mapConfig';
import {
  formatDecimal,
  formatInteger,
  formatMapName,
} from '../utils/performanceFormatters';
import '../styles/Match/matchDetails.css';

const Replay2DViewer = lazy(() => import('../features/replay2d/components/Replay2DViewerV2'));

const TABS = [
  { id: 'overview', label: 'Resumen', icon: LayoutGrid },
  { id: 'scoreboard', label: 'Marcador', icon: Trophy },
  { id: 'rounds', label: 'Rondas', icon: ListVideo },
  { id: 'analysis', label: 'Análisis', icon: BarChart3 },
  { id: 'weapons', label: 'Armas', icon: Crosshair },
  { id: 'replay', label: 'Replay', icon: Play },
];

const LEGACY_TABS = new Set(['economy', 'performance', 'compare']);

const number = (value) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

const duration = (seconds) => {
  const total = Math.max(0, Math.round(number(seconds)));
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
};

const resultTone = (rating) => {
  if (rating >= 1.12) return 'positive';
  if (rating >= 0.92) return 'neutral';
  return 'negative';
};

function MatchScoreboard({
  myTeam,
  enemyTeam,
  currentUserId,
  playerProfiles,
  isVictory,
  onSelect,
}) {
  const renderTeam = (players, ownTeam) => (
    <div className={`md-team ${ownTeam ? 'is-own' : 'is-enemy'}`}>
      <header>
        <span>{ownTeam ? <Shield size={15} /> : <Swords size={15} />}</span>
        <div>
          <strong>{ownTeam ? 'Tu equipo' : 'Rival'}</strong>
          <small>{ownTeam ? (isVictory ? 'Ganador' : 'Derrota') : (isVictory ? 'Derrota' : 'Ganador')}</small>
        </div>
      </header>
      <div className="md-table-scroll">
        <table>
          <thead>
            <tr>
              <th>Jugador</th>
              <th>K–D</th>
              <th>+/-</th>
              <th>ADR</th>
              <th>KAST</th>
              <th>HS</th>
              <th>Rating</th>
            </tr>
          </thead>
          <tbody>
            {players.map((player, index) => {
              const isCurrent = String(player.steam_id) === String(currentUserId);
              const delta = number(player.kills) - number(player.deaths);
              return (
                <tr key={player.steam_id} className={isCurrent ? 'is-current' : ''}>
                  <td>
                    <PlayerIdentity
                      player={player}
                      profile={playerProfiles[String(player.steam_id)]}
                      isCurrent={isCurrent}
                      isLeader={index === 0}
                      onSelect={onSelect}
                    />
                  </td>
                  <td><strong>{formatInteger(player.kills)}</strong><span>–</span>{formatInteger(player.deaths)}</td>
                  <td className={delta > 0 ? 'is-good' : delta < 0 ? 'is-bad' : ''}>
                    {delta > 0 ? '+' : ''}{formatInteger(delta)}
                  </td>
                  <td>{formatInteger(player.adr)}</td>
                  <td>{formatInteger(player.kast)}%</td>
                  <td>{formatInteger(player.hs_percentage)}%</td>
                  <td>
                    <span className={`md-rating is-${resultTone(number(player.hltv_rating))}`}>
                      {formatDecimal(player.hltv_rating, 2)}
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );

  return (
    <section className="md-section md-scoreboard" role="tabpanel" id="match-panel-scoreboard">
      <header className="md-section-head">
        <div>
          <span className="md-eyebrow">Marcador completo</span>
          <h2>Quién tuvo impacto de verdad</h2>
        </div>
        <p>Pulsa un jugador para abrir su lectura individual.</p>
      </header>
      {renderTeam(myTeam, true)}
      {renderTeam(enemyTeam, false)}
    </section>
  );
}

const MatchDetails = () => {
  const { matchID, steamID } = useParams();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedTab = searchParams.get('tab');
  const initialTab = TABS.some((tab) => tab.id === requestedTab) ? requestedTab : 'overview';

  const matchQuery = useQuery(matchDetailsQueryOptions(matchID, steamID));
  const matchData = matchQuery.data;
  const loading = Boolean(matchID) && matchQuery.isPending;
  const error = matchQuery.error ? 'No se han podido cargar los datos de esta partida.' : '';
  const [activeTab, setActiveTab] = useState(initialTab);
  const [selectedPlayerId, setSelectedPlayerId] = useState(searchParams.get('player') || '');

  const updateQuery = useCallback((patch) => {
    setSearchParams((current) => {
      const next = new URLSearchParams(current);
      Object.entries(patch).forEach(([key, value]) => {
        if (value === null || value === undefined || value === '') next.delete(key);
        else next.set(key, String(value));
      });
      return next;
    }, { replace: true });
  }, [setSearchParams]);

  const selectTab = useCallback((tab) => {
    setActiveTab(tab);
    updateQuery({ tab: tab === 'overview' ? null : tab });
  }, [updateQuery]);

  useEffect(() => {
    if (LEGACY_TABS.has(requestedTab)) {
      selectTab('overview');
      return;
    }
    if (requestedTab && TABS.some((tab) => tab.id === requestedTab)) {
      setActiveTab(requestedTab);
    }
  }, [requestedTab, selectTab]);

  const allPlayers = useMemo(
    () => [...(matchData?.team_ct || []), ...(matchData?.team_t || [])],
    [matchData],
  );
  const currentUserId = matchData?.current_user_steam_id || '';
  const currentPlayer = matchData?.current_user
    || allPlayers.find((player) => String(player.steam_id) === String(currentUserId));
  const selectedPlayer = allPlayers.find(
    (player) => String(player.steam_id) === String(selectedPlayerId || currentUserId),
  ) || currentPlayer;
  const selectedProfile = matchData?.player_profiles?.[String(selectedPlayer?.steam_id)] || {};
  const userTeam = currentPlayer?.team;
  const myTeam = userTeam === 'CT' ? matchData?.team_ct || [] : matchData?.team_t || [];
  const enemyTeam = userTeam === 'CT' ? matchData?.team_t || [] : matchData?.team_ct || [];
  const metadata = matchData?.metadata || {};
  const isVictory = matchData?.result === 'victory';

  const selectPlayer = (player) => {
    setSelectedPlayerId(String(player.steam_id));
    setActiveTab('analysis');
    updateQuery({ tab: 'analysis', player: player.steam_id });
  };

  const openReplayRound = (round, tick) => {
    setActiveTab('replay');
    updateQuery({
      tab: 'replay',
      round,
      tick: Number.isFinite(Number(tick)) ? Math.round(Number(tick)) : 0,
    });
  };

  if (loading) {
    return (
      <NavigationFrame>
        <main className="match-detail">
          <div className="md-loading" role="status">
            <span />
            <strong />
            <div><i /><i /><i /></div>
            <p>Preparando la lectura de la partida…</p>
          </div>
        </main>
      </NavigationFrame>
    );
  }

  if (error || !matchData) {
    return (
      <NavigationFrame>
        <main className="match-detail">
          <div className="md-error">
            <span>PARTIDA NO DISPONIBLE</span>
            <h1>{error || 'No se ha encontrado la partida.'}</h1>
            <div>
              <button type="button" onClick={() => matchQuery.refetch()}>Reintentar</button>
              <button type="button" onClick={() => navigate('/history-games')}>Volver</button>
            </div>
          </div>
        </main>
      </NavigationFrame>
    );
  }

  return (
    <NavigationFrame>
      <main className="match-detail">
        <header
          className="md-hero"
          style={{ '--md-map': `url(${getMapImage(metadata.map_name)})` }}
        >
          <button type="button" className="md-back" onClick={() => navigate('/history-games')}>
            <ArrowLeft size={15} />
            Partidas
          </button>

          <div className="md-hero-map">
            <span>Partida competitiva</span>
            <h1>{formatMapName(metadata.map_name)}</h1>
            <div>
              <span><Clock3 size={13} /> {duration(metadata.duration_seconds)}</span>
              <span><Users size={13} /> {metadata.total_rounds} rondas</span>
            </div>
          </div>

          <div className="md-hero-score">
            <span className={isVictory ? 'is-win' : 'is-loss'}>
              {isVictory ? 'Victoria' : 'Derrota'}
            </span>
            <div>
              <strong>{metadata.team_score}</strong>
              <i>:</i>
              <strong>{metadata.opponent_score}</strong>
            </div>
          </div>

          <div className="md-hero-player">
            <span>Tu partida</span>
            <strong>{formatDecimal(currentPlayer?.hltv_rating, 2)}</strong>
            <small>Rating · {formatInteger(currentPlayer?.adr)} ADR · {formatInteger(currentPlayer?.kills)} K</small>
          </div>
        </header>

        <nav className="md-tabs" role="tablist" aria-label="Secciones de la partida">
          {TABS.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                type="button"
                role="tab"
                key={tab.id}
                id={`match-tab-${tab.id}`}
                aria-selected={activeTab === tab.id}
                aria-controls={`match-panel-${tab.id}`}
                tabIndex={activeTab === tab.id ? 0 : -1}
                className={activeTab === tab.id ? 'is-active' : ''}
                onClick={() => selectTab(tab.id)}
                onKeyDown={(event) => {
                  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
                  event.preventDefault();
                  const index = TABS.findIndex((item) => item.id === activeTab);
                  const next = event.key === 'Home'
                    ? 0
                    : event.key === 'End'
                      ? TABS.length - 1
                      : (index + (event.key === 'ArrowRight' ? 1 : -1) + TABS.length) % TABS.length;
                  selectTab(TABS[next].id);
                  event.currentTarget.parentElement?.querySelectorAll('[role="tab"]')?.[next]?.focus();
                }}
              >
                <Icon size={14} />
                {tab.label}
              </button>
            );
          })}
        </nav>

        {activeTab === 'overview' && (
          <MatchBrief
            rounds={matchData.rounds || []}
            userStats={currentPlayer || {}}
            isVictory={isVictory}
            score={`${metadata.team_score}:${metadata.opponent_score}`}
          />
        )}

        {activeTab === 'scoreboard' && (
          <MatchScoreboard
            myTeam={myTeam}
            enemyTeam={enemyTeam}
            currentUserId={currentUserId}
            playerProfiles={matchData.player_profiles || {}}
            isVictory={isVictory}
            onSelect={selectPlayer}
          />
        )}

        {activeTab === 'rounds' && (
          <MatchRounds
            rounds={matchData.rounds || []}
            initialRound={number(searchParams.get('round')) || 1}
            onOpenReplay={openReplayRound}
          />
        )}

        {activeTab === 'analysis' && (
          selectedPlayer
            ? (
              <MatchAnalysis
                player={selectedPlayer}
                currentPlayer={currentPlayer}
                profile={selectedProfile}
                isCurrent={String(selectedPlayer.steam_id) === String(currentUserId)}
              />
            )
            : <EmptyPlayerState />
        )}

        {activeTab === 'weapons' && (
          selectedPlayer ? <MatchWeapons player={selectedPlayer} /> : <EmptyPlayerState />
        )}

        {activeTab === 'replay' && (
          <section className="md-section md-replay" role="tabpanel" id="match-panel-replay">
            <Suspense fallback={<div className="md-loading" role="status">Cargando replay…</div>}>
              <Replay2DViewer matchId={matchID} initialRound={1} />
            </Suspense>
          </section>
        )}
      </main>
    </NavigationFrame>
  );
};

export default MatchDetails;
