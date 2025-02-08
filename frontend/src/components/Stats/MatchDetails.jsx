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

  /**
   * Separa a los jugadores en mi equipo y el equipo enemigo,
   * basándonos en la coincidencia de steamID con el jugador local.
   */
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

  /**
   * Ordena los jugadores por 'rating' y devuelve los 3 primeros.
   * (Si prefieres otra métrica, cámbiala aquí).
   */
  const getTop3Players = (allPlayers = []) => {
    const sorted = [...allPlayers].sort((a, b) => (b.rating || 0) - (a.rating || 0));
    return sorted.slice(0, 3);
  };

  /**
   * Da formato "mm:ss" -> "Xm Ys" para la duración.
   */
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

  // Calculamos el top 3 (por rating) para mostrarlos arriba
  const top3Players = getTop3Players(players);

  // Determinamos si mi equipo ganó o no (para la tabla)
  const isMyTeamWinner = team_score > opponent_score;
  const isEnemyTeamWinner = opponent_score > team_score;

  /**
   * Renderiza la tabla con tus estadísticas anteriores: 
   * Player, Kills, Deaths, K/D, HS%, ADR, Flash, Posición
   */
  const renderTeamTable = (teamPlayers, teamLabel, isWinner) => {
    return (
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
                <td>
                  {player.kd_ratio !== undefined
                    ? player.kd_ratio.toFixed(2)
                    : '-'
                  }
                </td>
                <td>
                  {player.hs_percentage !== undefined
                    ? player.hs_percentage.toFixed(1) + '%'
                    : '-'
                  }
                </td>
                <td>
                  {player.adr !== undefined
                    ? player.adr.toFixed(1)
                    : '-'
                  }
                </td>
                <td>
                  {player.flash_assists !== undefined
                    ? player.flash_assists
                    : '-'
                  }
                </td>
                <td>
                  {player.position
                    ? `#${player.position}`
                    : 'N/A'
                  }
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  };

  return (
    <div className="match-details-container">
      <motion.div
        className="scoreboard-content"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        {/* ENCABEZADO O BOTÓN DE VOLVER */}
        <div className="header-back">
          <Link to="/HistoryGames" className="back-link">
            &larr; Volver al historial
          </Link>
        </div>

        {/* ENCABEZADO PROFESIONAL */}
        <div className="match-top-header">
          {/* Info principal del match: Mapa, fecha, resultado, duración */}
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

          {/* Top 3 jugadores */}
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
                {/* Ejemplo: mostrar su rating */}
                {player.rating !== undefined && (
                  <p className="top3-sub">Rating: {player.rating.toFixed(2)}</p>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* SECCIÓN DE TABLAS (EQUIPOS) */}
        <div className="teams-scoreboard">
          {renderTeamTable(myTeam, 'My Team', isMyTeamWinner)}
          {renderTeamTable(enemyTeam, 'Enemy Team', isEnemyTeamWinner)}
        </div>
      </motion.div>
    </div>
  );
};

export default MatchDetails;
