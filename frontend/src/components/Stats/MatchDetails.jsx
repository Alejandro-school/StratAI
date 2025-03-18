import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { useParams, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import '../../styles/Stats/MatchDetails.css';

const MatchDetails = () => {
  const { steamID, matchID } = useParams();
  const [match, setMatch] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [currentPage, setCurrentPage] = useState(0); // 0: Básicas, 1: Avanzadas

  useEffect(() => {
    const fetchMatchData = async () => {
      try {
        const url = `http://localhost:8080/match/${steamID}/${matchID}`;
        const response = await axios.get(url, { withCredentials: true });
        setMatch(response.data);
      } catch (err) {
        console.error(err);
        setError('Error al cargar los detalles de la partida.');
      } finally {
        setLoading(false);
      }
    };
    fetchMatchData();
  }, [steamID, matchID]);

  const splitPlayersByTeam = (players) => {
    const userPlayer = players.find(p => String(p.steamID).trim() === String(steamID).trim());
    if (!userPlayer) {
      return { myTeam: [], enemyTeam: players };
    }
    return {
      myTeam: players.filter(p => p.team === userPlayer.team),
      enemyTeam: players.filter(p => p.team !== userPlayer.team),
    };
  };

  const getTop3Players = (allPlayers = []) => {
    const sorted = [...allPlayers].sort((a, b) => (b.rating || 0) - (a.rating || 0));
    return sorted.slice(0, 3);
  };

  const formatDuration = (dur) => {
    if (!dur || dur === "00:00") return "Desconocida";
    const [min, sec] = dur.split(":").map(Number);
    return `${min}m ${sec}s`;
  };

  if (loading) {
    return <div className="match-details-container">Cargando...</div>;
  }
  if (error) {
    return <div className="match-details-container">{error}</div>;
  }
  if (!match) {
    return <div className="match-details-container">No se encontraron datos.</div>;
  }

  const { map_name, date, team_score, opponent_score, duration, players = [] } = match;
  const { myTeam, enemyTeam } = splitPlayersByTeam(players);
  const top3Players = getTop3Players(players);
  const isMyTeamWinner = team_score > opponent_score;
  const isEnemyTeamWinner = opponent_score > team_score;

  const renderBasicTeamTable = (teamPlayers, teamLabel, isWinner) => (
    <div className="team-table-container">
      <div className="team-table-header">
        <h2>{teamLabel}</h2>
        <span className={`team-result-badge ${isWinner ? 'win' : 'loss'}`}>
          {isWinner ? 'WIN' : 'LOSS'}
        </span>
      </div>
      <table className="scoreboard-table">
        <thead>
          <tr>
            <th>Player</th>
            <th>Kills</th>
            <th>Deaths</th>
            <th>K/D</th>
            <th>HS%</th>
            <th>ADR</th>
            <th>Flash</th>
            <th>Posición</th>
          </tr>
        </thead>
        <tbody>
          {teamPlayers.map((player, i) => (
            <tr key={i} className={i === 0 ? 'top-performer' : (player.steamID === steamID ? 'current-player' : '')}>
              <td className="player-cell">
                <img
                  src={player.avatar || '/assets/default_avatar.png'}
                  alt="Avatar"
                  className="table-avatar"
                />
                <span>
                  {player.name}
                  {player.steamID === steamID && ' (TÚ)'}
                </span>
              </td>
              <td>{player.kills !== undefined ? player.kills : '-'}</td>
              <td>{player.deaths !== undefined ? player.deaths : '-'}</td>
              <td>{player.kd_ratio !== undefined ? player.kd_ratio.toFixed(2) : '-'}</td>
              <td>{player.hs_percentage !== undefined ? player.hs_percentage.toFixed(1) + '%' : '-'}</td>
              <td>{player.adr !== undefined ? player.adr.toFixed(1) : '-'}</td>
              <td>{player.EnemiesFlashed !== undefined ? player.EnemiesFlashed : '-'}</td>
              <td>{player.position ? `#${player.position}` : 'N/A'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );

  const renderAdvancedTeamTable = (teamPlayers, teamLabel, isWinner) => (
    <div className="team-table-container">
      <div className="team-table-header">
        <h2>{teamLabel}</h2>
        <span className={`team-result-badge ${isWinner ? 'win' : 'loss'}`}>
          {isWinner ? 'WIN' : 'LOSS'}
        </span>
      </div>
      <table className="scoreboard-table">
        <thead>
          <tr>
            <th>Player</th>
            <th>Disparos</th>
            <th>Mirilla</th>
            <th>2K</th>
            <th>3K</th>
            <th>4K</th>
            <th>Ace</th>
            <th>Clutch</th>
          </tr>
        </thead>
        <tbody>
          {teamPlayers.map((player, i) => (
            <tr key={i} className={i === 0 ? 'top-performer' : (player.steamID === steamID ? 'current-player' : '')}>
              <td className="player-cell">
                <img
                  src={player.avatar || '/assets/default_avatar.png'}
                  alt="Avatar"
                  className="table-avatar"
                />
                <span>
                  {player.name}
                  {player.steamID === steamID && ' (TÚ)'}
                </span>
              </td>
              <td>{player.shots_fired !== undefined ? player.shots_fired : '-'}</td>
              <td>{player.aim_placement !== undefined ? player.aim_placement.toFixed(1) : '-'}</td>
              <td>{player.double_kills !== undefined ? player.double_kills : '-'}</td>
              <td>{player.triple_kills !== undefined ? player.triple_kills : '-'}</td>
              <td>{player.quad_kills !== undefined ? player.quad_kills : '-'}</td>
              <td>{player.ace !== undefined ? player.ace : '-'}</td>
              <td>{player.clutch_wins !== undefined ? player.clutch_wins : '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );

  const renderTeamSection = () => {
    if (currentPage === 0) {
      return (
        <div className="teams-scoreboard">
          {renderBasicTeamTable(myTeam, 'My Team', isMyTeamWinner)}
          {renderBasicTeamTable(enemyTeam, 'Enemy Team', isEnemyTeamWinner)}
        </div>
      );
    } else {
      return (
        <div className="teams-scoreboard">
          {renderAdvancedTeamTable(myTeam, 'My Team', isMyTeamWinner)}
          {renderAdvancedTeamTable(enemyTeam, 'Enemy Team', isEnemyTeamWinner)}
        </div>
      );
    }
  };

  return (
    <div className="match-details-container">
      <motion.div
        className="scoreboard-content"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        <div className="header-back">
          <Link to="/HistoryGames" className="back-link">
            &larr; Volver al historial
          </Link>
        </div>

        <div className="match-top-header">
          <div className="match-main-info">
            <h1 className="map-name">
              {map_name.replace(/^de_/, '').charAt(0).toUpperCase() + map_name.replace(/^de_/, '').slice(1)}
            </h1>
            <span className="match-date">{date}</span>
            <div className="score-duration-boxes">
              <div className={`score-box ${isMyTeamWinner ? 'win' : 'loss'}`}>
                <p className="score-label">Resultado</p>
                <p className="score-value">
                  {team_score} : {opponent_score}
                </p>
              </div>
              <div className="score-box">
                <p className="score-label">Duración</p>
                <p className="score-value">{formatDuration(duration)}</p>
              </div>
            </div>
          </div>
          <div className="top3-wrapper">
            {top3Players.map((player, index) => (
              <div className="top3-card" key={player.steamID || index}>
                <h4 className="top3-rank">
                  {index + 1}º - {player.name}
                </h4>
                <img
                  src={player.avatar || '/assets/default_avatar.png'}
                  alt="Top player avatar"
                  className="top3-avatar"
                />
                {player.rating !== undefined && (
                  <p className="top3-sub">Rating: {player.rating.toFixed(2)}</p>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Navegación entre pestañas de estadísticas */}
        <div className="stats-tabs">
          <div
            className={`stats-tab ${currentPage === 0 ? 'active' : ''}`}
            onClick={() => setCurrentPage(0)}
          >
            Básicas
          </div>
          <div
            className={`stats-tab ${currentPage === 1 ? 'active' : ''}`}
            onClick={() => setCurrentPage(1)}
          >
            Avanzadas
          </div>
        </div>

        {renderTeamSection()}
      </motion.div>
    </div>
  );
};

export default MatchDetails;
