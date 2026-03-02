// frontend/src/pages/CoachDashboard.jsx
import React, { useEffect, useMemo, useRef, useState } from 'react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useUser } from '../context/UserContext';
import useCoachChat from '../hooks/useCoachChat';
import { API_URL } from '../utils/api';
import Replay2DViewer from '../components/Stats/Replay2DViewer';
import {
  Brain, Sparkles,
  Send,
  Clock3,
  ChevronRight,
  Loader2,
  Search,
  Shield,
  Target,
  Crosshair,
  Trophy,
  Skull,
  PlayCircle,
  X,
  MonitorPlay
} from 'lucide-react';
import useReplaySyncStore from '../stores/useReplaySyncStore';
import '../styles/pages/coachDashboard.css';

const getMatchId = (match) => match.match_id || match.matchID || match.id;
const getMatchMap = (match) => (match.map_name || match.map || 'Desconocido');
const getMapImage = (mapName) => `/images/maps/${mapName || 'de_dust2'}.png`;

const formatMatchDate = (dateValue) => {
  if (!dateValue) return 'Sin fecha';
  const parsed = new Date(dateValue);
  if (Number.isNaN(parsed.getTime())) return 'Sin fecha';
  return parsed.toLocaleDateString('es-ES', {
    day: '2-digit',
    month: 'short',
    year: 'numeric'
  });
};

/* ── SVG Animated Border for selected match ── */
const AnimatedBorderSVG = ({ isWin }) => (
  <svg className="hero-svg-border" viewBox="0 0 400 220" fill="none" preserveAspectRatio="none">
    <rect
      x="1" y="1" width="398" height="218" rx="19"
      stroke={isWin ? 'url(#winGrad)' : 'url(#lossGrad)'}
      strokeWidth="2"
      className="hero-svg-stroke"
    />
    <defs>
      <linearGradient id="winGrad" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stopColor="#22d3ee" />
        <stop offset="50%" stopColor="#34d399" />
        <stop offset="100%" stopColor="#22d3ee" />
      </linearGradient>
      <linearGradient id="lossGrad" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stopColor="#f87171" />
        <stop offset="50%" stopColor="#fb923c" />
        <stop offset="100%" stopColor="#f87171" />
      </linearGradient>
    </defs>
  </svg>
);

/* ── Radar Sweep Icon ── */
const RadarIcon = () => (
  <svg className="radar-icon" width="18" height="18" viewBox="0 0 24 24" fill="none">
    <circle cx="12" cy="12" r="10" stroke="rgba(34,211,238,0.3)" strokeWidth="1.5" />
    <circle cx="12" cy="12" r="6" stroke="rgba(34,211,238,0.2)" strokeWidth="1" />
    <circle cx="12" cy="12" r="2" fill="#22d3ee" />
    <line x1="12" y1="12" x2="12" y2="2" stroke="#22d3ee" strokeWidth="1.5" strokeLinecap="round" className="radar-sweep" />
  </svg>
);

/* ── Avatar Ring SVG ── */
const AvatarRingSVG = () => (
  <svg className="avatar-ring-svg" width="52" height="52" viewBox="0 0 52 52" fill="none">
    <circle cx="26" cy="26" r="24" stroke="url(#avatarRingGrad)" strokeWidth="2" strokeDasharray="6 4" className="avatar-ring-dash" />
    <defs>
      <linearGradient id="avatarRingGrad" x1="0" y1="0" x2="52" y2="52">
        <stop offset="0%" stopColor="#22d3ee" />
        <stop offset="100%" stopColor="#8b5cf6" />
      </linearGradient>
    </defs>
  </svg>
);

const CoachDashboard = () => {
  const { user } = useUser();
  const { playAiClip, stopAiClip, isPlaying, activeClip } = useReplaySyncStore();
  const [allMatches, setAllMatches] = useState([]);
  const [loadingMatches, setLoadingMatches] = useState(true);
  const [selectedMatchId, setSelectedMatchId] = useState(null);
  const [matchFilterQuery, setMatchFilterQuery] = useState('');
  const [sortBy, setSortBy] = useState('date_desc');
  const [heroAnimKey, setHeroAnimKey] = useState(0);
  const chatEndRef = useRef(null);
  const [showReplayViewer, setShowReplayViewer] = useState(false);
  const [chatHistory] = useState([
    'Mejorar decisiones de utilidad',
    'Rutina de warmup para aim',
    'Plan semanal de consistencia'
  ]);

  // When AI sets an activeClip, open the replay viewer
  useEffect(() => {
    if (activeClip) {
      setShowReplayViewer(true);
    }
  }, [activeClip]);

  const handleCloseReplay = () => {
    setShowReplayViewer(false);
    stopAiClip();
  };

  const {
    messages,
    chatInput,
    setChatInput,
    isAiTyping,
    submitMessage,
    sendProactiveGreeting,
    analyzeMatch
  } = useCoachChat(user);

  useEffect(() => {
    const fetchMatches = async () => {
      try {
        setLoadingMatches(true);
        const response = await fetch(`${API_URL}/steam/get-processed-demos`, {
          credentials: 'include'
        });
        if (!response.ok) throw new Error('No se pudieron cargar las partidas');
        const data = await response.json();
        const fetchedMatches = Array.isArray(data.matches) ? data.matches : [];
        setAllMatches(fetchedMatches);
        if (fetchedMatches[0]) {
          const firstId = getMatchId(fetchedMatches[0]);
          setSelectedMatchId(firstId || null);
        }
        // AI takes initiative: proactive greeting with pattern analysis
        if (fetchedMatches.length > 0) {
          sendProactiveGreeting(fetchedMatches);
        }
      } catch (error) {
        console.error(error);
        setAllMatches([]);
      } finally {
        setLoadingMatches(false);
      }
    };

    fetchMatches();
    // Intentional run-once: sendProactiveGreeting identity depends on messages.length
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!allMatches.length) {
      setSelectedMatchId(null);
      return;
    }

    const selectedStillVisible = allMatches.some((match) => getMatchId(match) === selectedMatchId);
    if (!selectedStillVisible) {
      setSelectedMatchId(getMatchId(allMatches[0]));
    }
  }, [allMatches, selectedMatchId]);

  const filteredMatches = useMemo(() => {
    const query = matchFilterQuery.trim().toLowerCase();
    if (!query) return allMatches;

    return allMatches.filter((match) => {
      const mapName = getMatchMap(match);
      const score = `${match.team_score ?? '?'}-${match.opponent_score ?? '?'}`;
      const dateText = formatMatchDate(match.match_date);
      const matchId = String(getMatchId(match) || '');
      const haystack = `${mapName} ${score} ${dateText} ${matchId}`.toLowerCase();
      return haystack.includes(query);
    });
  }, [allMatches, matchFilterQuery]);

  const sortedMatches = useMemo(() => {
    const matchesCopy = [...filteredMatches];

    const getDateValue = (match) => {
      if (!match.match_date) return 0;
      const parsed = new Date(match.match_date).getTime();
      return Number.isNaN(parsed) ? 0 : parsed;
    };

    const getMapValue = (match) => getMatchMap(match).toLowerCase();

    switch (sortBy) {
      case 'date_asc':
        matchesCopy.sort((a, b) => getDateValue(a) - getDateValue(b));
        break;
      case 'map_asc':
        matchesCopy.sort((a, b) => getMapValue(a).localeCompare(getMapValue(b)));
        break;
      case 'map_desc':
        matchesCopy.sort((a, b) => getMapValue(b).localeCompare(getMapValue(a)));
        break;
      case 'date_desc':
      default:
        matchesCopy.sort((a, b) => getDateValue(b) - getDateValue(a));
        break;
    }

    return matchesCopy;
  }, [filteredMatches, sortBy]);

  useEffect(() => {
    if (!sortedMatches.length) {
      setSelectedMatchId(null);
      return;
    }

    const selectedStillVisible = sortedMatches.some((match) => getMatchId(match) === selectedMatchId);
    if (!selectedStillVisible) {
      setSelectedMatchId(getMatchId(sortedMatches[0]));
    }
  }, [sortedMatches, selectedMatchId]);

  const selectedMatch = useMemo(
    () => sortedMatches.find((match) => getMatchId(match) === selectedMatchId),
    [sortedMatches, selectedMatchId]
  );

  const selectedMapName = selectedMatch ? getMatchMap(selectedMatch) : null;

  const handleSelectMatch = (matchId) => {
    setSelectedMatchId(matchId);
    setHeroAnimKey((k) => k + 1);
  };

  const submitCoachMessage = (text) => {
    if (!text?.trim()) return;
    submitMessage(text);
  };

  const handleSendMessage = (e) => {
    e.preventDefault();
    submitCoachMessage(chatInput);
  };

  const handleAnalyzeSelectedMatch = () => {
    if (!selectedMatch) return;
    analyzeMatch(selectedMatch);
  };

  // Auto-scroll chat
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, isAiTyping]);

  // Hero card data
  const heroData = useMemo(() => {
    if (!selectedMatch) return null;
    const rawMapName = getMatchMap(selectedMatch);
    const mapName = rawMapName.replace('de_', '');
    const score = `${selectedMatch.team_score ?? '?'}-${selectedMatch.opponent_score ?? '?'}`;
    const isWin = selectedMatch.result === 'W' || selectedMatch.result === 'victory';
    const dateLabel = formatMatchDate(selectedMatch.match_date);
    const totalRounds = Number(selectedMatch.total_rounds) > 0 ? selectedMatch.total_rounds : '—';
    const teamLabel = selectedMatch.user_team || '—';
    return { rawMapName, mapName, score, isWin, dateLabel, totalRounds, teamLabel };
  }, [selectedMatch]);

  return (
    <NavigationFrame>
      <div className="coach-dashboard coach-clean">
        <header className="coach-clean-header">
          <div className="coach-header-copy">
            <span className="coach-header-kicker">
              <Crosshair size={10} />
              STRATAI COMMAND CENTER
            </span>
            <h1>Coach IA</h1>
            <p>Panel de coaching táctico para análisis y mejora de rendimiento.</p>
          </div>
          <div className="coach-header-meta">
            <span className="coach-online-badge">
              <span className="online-dot" />
              <Sparkles size={14} />
              Asistente activo
            </span>
            <span className="coach-header-count">
              <Target size={11} />
              {allMatches.length} partidas listas
            </span>
          </div>
        </header>

        <div className={`coach-clean-layout ${showReplayViewer ? 'replay-active' : ''}`}>

          {/* ── REPLAY 2D PANEL (slides in when active) ── */}
          {showReplayViewer && (
            <section className="coach-replay-panel">
              <div className="replay-panel-header">
                <div className="replay-panel-title">
                  <MonitorPlay size={16} />
                  <span>Análisis Visual 2D</span>
                </div>
                <button
                  type="button"
                  className="replay-panel-close"
                  onClick={handleCloseReplay}
                  title="Cerrar replay"
                >
                  <X size={16} />
                </button>
              </div>
              <div className="replay-panel-body">
                <Replay2DViewer matchId={selectedMatchId} initialRound={1} />
              </div>
            </section>
          )}

          {/* ── SIDEBAR (hidden when replay is active) ── */}
          {!showReplayViewer && (
          <aside className="coach-orientation-panel">

            {/* ── SELECTED MATCH HERO CARD ── */}
            {heroData && (
              <section className="selected-match-hero" key={heroAnimKey}>
                <div
                  className="hero-bg"
                  style={{ backgroundImage: `url(${getMapImage(heroData.rawMapName)})` }}
                />
                <AnimatedBorderSVG isWin={heroData.isWin} />
                <div className="hero-content">
                  <div className="hero-top-row">
                    <span className="hero-map-label">{heroData.mapName.toUpperCase()}</span>
                    <span className={`hero-result-badge ${heroData.isWin ? 'win' : 'loss'}`}>
                      {heroData.isWin ? <Trophy size={12} /> : <Skull size={12} />}
                      {heroData.isWin ? 'VICTORIA' : 'DERROTA'}
                    </span>
                  </div>
                  <div className={`hero-score ${heroData.isWin ? 'win' : 'loss'}`}>
                    {heroData.score}
                  </div>
                  <div className="hero-meta-row">
                    <span className="hero-meta-pill">
                      <Shield size={10} />
                      {heroData.teamLabel}
                    </span>
                    <span className="hero-meta-pill">{heroData.dateLabel}</span>
                    <span className="hero-meta-pill">R{heroData.totalRounds}</span>
                  </div>
                </div>
              </section>
            )}

            {/* ── MATCH SELECTOR ── */}
            <section className="coach-orientation-card match-selector-card">
              <div className="orientation-title">
                <RadarIcon />
                Selecciona partida para analizar
              </div>

              {loadingMatches ? (
                <div className="matches-loading">
                  <Loader2 size={15} className="spin" />
                  Cargando partidas...
                </div>
              ) : (
                <>
                  <div className="match-controls-row">
                    <div className="matches-count-pill">
                      {sortedMatches.length} de {allMatches.length} partidas
                    </div>
                    <div className="match-sort-wrap">
                      <select
                        id="coach-match-sort"
                        className="match-sort-select"
                        value={sortBy}
                        onChange={(e) => setSortBy(e.target.value)}
                      >
                        <option value="date_desc">Más reciente</option>
                        <option value="date_asc">Más antigua</option>
                        <option value="map_asc">Mapa (A-Z)</option>
                        <option value="map_desc">Mapa (Z-A)</option>
                      </select>
                    </div>
                  </div>

                  <div className="match-filter-wrap">
                    <Search size={13} />
                    <input
                      type="text"
                      className="match-filter-input"
                      placeholder="Filtrar por mapa, score, fecha o ID"
                      value={matchFilterQuery}
                      onChange={(e) => setMatchFilterQuery(e.target.value)}
                    />
                  </div>

                  {sortedMatches.length ? (
                    <>
                      <div className="match-list-grid">
                        {sortedMatches.map((match, idx) => {
                          const matchId = getMatchId(match);
                          const isSelected = selectedMatchId === matchId;
                          const rawMapName = getMatchMap(match);
                          const mapName = rawMapName.replace('de_', '');
                          const score = `${match.team_score ?? '?'}-${match.opponent_score ?? '?'}`;
                          const isWin = match.result === 'W' || match.result === 'victory';
                          const dateLabel = formatMatchDate(match.match_date);
                          const totalRounds = Number(match.total_rounds) > 0 ? match.total_rounds : '—';
                          const teamLabel = match.user_team || '—';

                          return (
                            <button
                              key={matchId || `${mapName}-${score}`}
                              type="button"
                              className={`match-item-btn map-card ${isSelected ? 'selected' : ''}`}
                              onClick={() => handleSelectMatch(matchId)}
                              style={{
                                backgroundImage: `url(${getMapImage(rawMapName)})`,
                                '--stagger': `${idx * 60}ms`
                              }}
                            >
                              <div className="match-item-overlay">
                                <div className="match-item-top">
                                  <span className="match-map">{mapName.toUpperCase()}</span>
                                  <span className={`match-result ${isWin ? 'win' : 'loss'}`}>
                                    {isWin ? 'WIN' : 'LOSS'}
                                  </span>
                                </div>
                                <div className={`match-score ${isWin ? 'win' : 'loss'}`}>{score}</div>
                                <div className="match-meta-row">
                                  <span className="match-date-label">{dateLabel}</span>
                                  <span className="match-meta-pill">{teamLabel}</span>
                                  <span className="match-meta-pill">R{totalRounds}</span>
                                </div>
                              </div>
                              {isSelected && (
                                <div className="selected-card-glow" />
                              )}
                            </button>
                          );
                        })}
                      </div>

                      <button
                        type="button"
                        className="analyze-match-btn gradient-border-btn"
                        onClick={handleAnalyzeSelectedMatch}
                        disabled={!selectedMatch}
                      >
                        <Crosshair size={14} />
                        Analizar partida seleccionada
                        <ChevronRight size={14} />
                      </button>
                    </>
                  ) : (
                    <div className="matches-empty-pro">
                      <p>No hay partidas procesadas todavía.</p>
                      <span>Cuando el pipeline termine nuevos demos, aparecerán aquí automáticamente.</span>
                    </div>
                  )}
                </>
              )}
            </section>

            {/* ── CHAT HISTORY ── */}
            <section className="coach-orientation-card chat-history-card">
              <div className="orientation-title">
                <Clock3 size={15} />
                Historial de chats
              </div>
              <ul className="chat-history-list">
                {chatHistory.map((item) => (
                  <li key={item}>
                    <button type="button" onClick={() => submitCoachMessage(`Quiero retomar: ${item}`)}>
                      <ChevronRight size={12} className="history-arrow" />
                      {item}
                    </button>
                  </li>
                ))}
              </ul>
            </section>
          </aside>
          )}

          {/* ── CHAT SHELL ── */}
          <main className="coach-chat-shell">
            <div
              className="chat-interface-header coach-chat-header-clean map-context-header"
              style={
                selectedMapName
                  ? {
                      backgroundImage: `linear-gradient(130deg, rgba(6,182,212,0.18), rgba(139,92,246,0.14), rgba(249,115,22,0.09)), linear-gradient(180deg, rgba(10,18,30,0.2), rgba(12,18,25,0.88)), url(${getMapImage(selectedMapName)})`
                    }
                  : undefined
              }
            >
              <div className="ai-status clean">
                <div className="ai-avatar-wrap">
                  <AvatarRingSVG />
                  <div className="ai-avatar">
                    <Brain size={20} color="var(--color-text-main)" />
                  </div>
                </div>
                <div className="ai-info">
                  <h3>StratAI Coach</h3>
                  <span>
                    {selectedMatch
                      ? `Partida seleccionada: ${getMatchMap(selectedMatch).replace('de_', '').toUpperCase()} ${selectedMatch.team_score ?? '?'}-${selectedMatch.opponent_score ?? '?'}`
                      : 'Bienvenido. ¿En qué quieres trabajar hoy?'}
                  </span>
                </div>
              </div>
            </div>

            <div className="chat-window clean">
              {messages.map((msg) => (
                <div
                  key={msg.id}
                  className={`chat-bubble ${msg.sender}`}
                >
                  {msg.context && <span className="context-pill">{msg.context}</span>}
                  <p className="chat-bubble-text">
                    {msg.isTyping && msg.visibleChars !== undefined ? (
                      <>
                        <span>{msg.text.substring(0, msg.visibleChars)}</span>
                        <span style={{ opacity: 0 }}>{msg.text.substring(msg.visibleChars)}</span>
                      </>
                    ) : (
                      msg.text
                    )}
                  </p>
                  
                  {/* --- AI INTERACTION PAYLOAD --- */}
                  {msg.interaction && (!msg.isTyping || (msg.isTyping && msg.visibleChars === msg.text.length)) && (
                    <div className="chat-ai-interaction">
                      <div className="interaction-header">
                        <PlayCircle size={14} className="interaction-icon" />
                        <span>Replay 2D Disponible</span>
                      </div>
                      <p className="interaction-desc">
                        Tick {msg.interaction.startTick} - {msg.interaction.endTick}
                      </p>
                      <button 
                        type="button"
                        className={`interaction-play-btn ${isPlaying && activeClip?.startTick === msg.interaction.startTick ? 'playing' : ''}`}
                        onClick={() => playAiClip(msg.interaction)}
                      >
                        {isPlaying && activeClip?.startTick === msg.interaction.startTick 
                          ? 'Reproduciendo Clip...' 
                          : '▶ Ver fragmento en 2D'}
                      </button>
                    </div>
                  )}
                </div>
              ))}
              <div ref={chatEndRef} />
            </div>


            <form className="chat-input-area" onSubmit={handleSendMessage}>
              <input 
                type="text" 
                className="chat-input-field" 
                placeholder="Escribe..."
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
              />
              <button type="submit" className="send-btn">
                <Send size={20} />
              </button>
            </form>
          </main>
        </div>
      </div>
    </NavigationFrame>
  );
};

export default CoachDashboard;
