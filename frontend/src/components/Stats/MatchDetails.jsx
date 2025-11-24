import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { useParams, useNavigate } from 'react-router-dom';
import SidebarComponent from '../Layout/Sidebar';
import { useUser } from '../../context/UserContext';
import '../../styles/Stats/MatchDetails.css';

const MatchDetails = () => {
  const { steamID, matchID } = useParams();
  const { user } = useUser();
  const navigate = useNavigate();
  const [match, setMatch] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [activeTab, setActiveTab] = useState('overview'); // overview, performance, detailed

  useEffect(() => {
    const fetchMatchData = async () => {
      try {
        // 1. Intentar cargar desde caché primero (navegación instantánea)
        const cacheKey = `match_details_${matchID}`;
        const cached = localStorage.getItem(cacheKey);
        
        if (cached) {
          const { data, timestamp } = JSON.parse(cached);
          const isRecent = (Date.now() - timestamp) < 3600000; // 1 hora
          
          if (isRecent) {
            console.log('✅ Cargando desde caché:', matchID);
            setMatch(data);
            setLoading(false);
            return; // Carga instantánea desde caché
          }
        }
        
        // 2. Si no hay caché válido, cargar desde servidor Go (más rápido)
        console.log('📡 Cargando desde Go service:', matchID);
        const url = `http://localhost:8080/match-details/${matchID}`;
        const response = await axios.get(url);
        
        // 3. Guardar en caché para próximas visitas
        localStorage.setItem(cacheKey, JSON.stringify({
          data: response.data,
          timestamp: Date.now()
        }));
        
        setMatch(response.data);
      } catch (err) {
        console.error('Error al cargar detalles:', err);
        setError('Error al cargar los detalles de la partida.');
      } finally {
        setLoading(false);
      }
    };
    fetchMatchData();
  }, [steamID, matchID]);

  const splitPlayersByTeam = (players) => {
    // Encontrar al jugador autenticado
    const userPlayer = players.find(p => 
      String(p.steamID || p.steam_id || '').trim() === String(steamID).trim()
    );
    
    if (!userPlayer) {
      console.warn('No se encontró el jugador autenticado en los datos de la partida');
      // Si no encontramos al usuario, intentamos dividir por el campo team
      const terrorists = players.filter(p => p.team === 'Terrorist' || p.team === 'T' || p.team === 2);
      const counterTerrorists = players.filter(p => p.team === 'CounterTerrorist' || p.team === 'CT' || p.team === 3);
      
      return { 
        myTeam: terrorists.sort((a, b) => (b.kills || 0) - (a.kills || 0)),
        enemyTeam: counterTerrorists.sort((a, b) => (b.kills || 0) - (a.kills || 0))
      };
    }

    // Normalizar el valor del team del usuario
    const userTeam = userPlayer.team;
    
    console.log('Usuario encontrado:', userPlayer.name, 'Team:', userTeam);
    console.log('Todos los jugadores:', players.map(p => ({ name: p.name, team: p.team })));
    
    // Filtrar jugadores por equipo
    const myTeam = players.filter(p => p.team === userTeam);
    const enemyTeam = players.filter(p => p.team !== userTeam);
    
    return {
      myTeam: myTeam.sort((a, b) => (b.kills || 0) - (a.kills || 0)),
      enemyTeam: enemyTeam.sort((a, b) => (b.kills || 0) - (a.kills || 0)),
    };
  };

  const formatDuration = (dur) => {
    if (!dur || dur === "00:00") return "Desconocida";
    const [min, sec] = dur.split(":").map(Number);
    return `${min}m ${sec}s`;
  };

  const getMVP = (players = []) => {
    if (!players.length) return null;
    return [...players].sort((a, b) => {
      const scoreA = (a.kills || 0) * 2 + (a.assists || 0) - (a.deaths || 0);
      const scoreB = (b.kills || 0) * 2 + (b.assists || 0) - (b.deaths || 0);
      return scoreB - scoreA;
    })[0];
  };

  if (loading) {
    return (
      <div className="match-details-wrapper">
        <SidebarComponent user={user} />
        <div className="match-details-content">
          <div className="loading-state">
            <div className="loading-spinner"></div>
            <p>Cargando detalles de la partida...</p>
          </div>
        </div>
      </div>
    );
  }

  if (error || !match) {
    return (
      <div className="match-details-wrapper">
        <SidebarComponent user={user} />
        <div className="match-details-content">
          <div className="error-state">
            <h3>{error || 'No se encontraron datos de la partida'}</h3>
            <button className="back-btn" onClick={() => navigate('/history-games')}>
              Volver al historial
            </button>
          </div>
        </div>
      </div>
    );
  }

  const { map_name, date, team_score, opponent_score, duration, players = [] } = match;
  const { myTeam, enemyTeam } = splitPlayersByTeam(players);
  const isMyTeamWinner = team_score > opponent_score;
  const mvpPlayer = getMVP(players);

  const renderPlayerRow = (player, index) => {
    const playerSteamID = String(player.steamID || player.steam_id || '').trim();
    const isCurrentUser = playerSteamID === String(steamID).trim();
    const mvpSteamID = mvpPlayer ? String(mvpPlayer.steamID || mvpPlayer.steam_id || '').trim() : '';
    const isMVP = mvpPlayer && playerSteamID === mvpSteamID;
    
    return (
      <tr key={index} className={`${isCurrentUser ? 'current-user' : ''} ${isMVP ? 'mvp' : ''}`}>
        <td className="player-info-cell">
          <div className="player-info">
            {isMVP && <div className="mvp-badge">MVP</div>}
            <img
              src={player.avatar || '/assets/default_avatar.png'}
              alt="Avatar"
              className="player-avatar"
            />
            <div className="player-name-wrapper">
              <span className="player-name">{player.name}</span>
              {isCurrentUser && <span className="you-badge">TÚ</span>}
            </div>
          </div>
        </td>
        <td className="stat-cell">
          <div className="stat-value">{player.kills || 0}</div>
        </td>
        <td className="stat-cell">
          <div className="stat-value">{player.deaths || 0}</div>
        </td>
        <td className="stat-cell">
          <div className="stat-value">{player.assists || 0}</div>
        </td>
        <td className="stat-cell">
          <div className={`stat-value ${(player.kd_ratio || 0) >= 1 ? 'positive' : 'negative'}`}>
            {player.kd_ratio?.toFixed(2) || '0.00'}
          </div>
        </td>
        <td className="stat-cell">
          <div className="stat-value">{player.adr?.toFixed(0) || '0'}</div>
        </td>
        <td className="stat-cell">
          <div className="stat-value">{player.hs_percentage?.toFixed(0) || '0'}%</div>
        </td>
      </tr>
    );
  };

  const renderTeamTable = (teamPlayers, teamName, isWinner) => (
    <div className="team-section">
      <div className="team-header">
        <div className="team-info">
          <h3>{teamName}</h3>
          <div className={`team-badge ${isWinner ? 'winner' : 'loser'}`}>
            {isWinner ? 'VICTORIA' : 'DERROTA'}
          </div>
        </div>
      </div>
      
      <div className="team-table-wrapper">
        <table className="team-table">
          <thead>
            <tr>
              <th className="player-header">Jugador</th>
              <th>K</th>
              <th>D</th>
              <th>A</th>
              <th>K/D</th>
              <th>ADR</th>
              <th>HS%</th>
            </tr>
          </thead>
          <tbody>
            {teamPlayers.map((player, index) => renderPlayerRow(player, index))}
          </tbody>
        </table>
      </div>
    </div>
  );

  return (
    <div className="match-details-wrapper">
      <SidebarComponent user={user} />
      <div className="match-details-content">
        {/* Back button */}
        <button className="back-button" onClick={() => navigate('/history-games')}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <line x1="19" y1="12" x2="5" y2="12"/>
            <polyline points="12 19 5 12 12 5"/>
          </svg>
          Volver al historial
        </button>

        {/* Match Header */}
        <div className="match-header-card">
          <div 
            className="match-header-bg"
            style={{
              backgroundImage: `url(/images/maps/${map_name}.png)`,
            }}
          />
          <div className="match-header-overlay">
            <div className="match-header-info">
              <h1 className="match-map-title">
                {map_name?.replace('de_', '').toUpperCase()}
              </h1>
              <div className="match-metadata">
                <span className="metadata-item">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                    <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                    <line x1="16" y1="2" x2="16" y2="6"/>
                    <line x1="8" y1="2" x2="8" y2="6"/>
                    <line x1="3" y1="10" x2="21" y2="10"/>
                  </svg>
                  {date}
                </span>
                <span className="metadata-item">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                    <circle cx="12" cy="12" r="10"/>
                    <polyline points="12 6 12 12 16 14"/>
                  </svg>
                  {formatDuration(duration)}
                </span>
              </div>
            </div>
            
            <div className="match-score-display">
              <div className="score-section">
                <div className="score-label">Marcador Final</div>
                <div className="score-numbers">
                  <span className={`score ${isMyTeamWinner ? 'win-score' : ''}`}>{team_score}</span>
                  <span className="score-divider">:</span>
                  <span className={`score ${!isMyTeamWinner ? 'win-score' : ''}`}>{opponent_score}</span>
                </div>
                <div className={`result-badge ${isMyTeamWinner ? 'victory' : 'defeat'}`}>
                  {isMyTeamWinner ? 'VICTORIA' : 'DERROTA'}
                </div>
              </div>
            </div>

            {/* MVP Section */}
            {mvpPlayer && (
              <div className="mvp-section">
                <div className="mvp-label">MVP</div>
                <img
                  src={mvpPlayer.avatar || '/assets/default_avatar.png'}
                  alt="MVP"
                  className="mvp-avatar"
                />
                <div className="mvp-name">{mvpPlayer.name}</div>
                <div className="mvp-stats">
                  {mvpPlayer.kills || 0}K / {mvpPlayer.deaths || 0}D / {mvpPlayer.assists || 0}A
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Teams Section */}
        <div className="teams-container">
          {renderTeamTable(myTeam, 'Mi Equipo', isMyTeamWinner)}
          {renderTeamTable(enemyTeam, 'Equipo Enemigo', !isMyTeamWinner)}
        </div>
      </div>
    </div>
  );
};

export default MatchDetails;
