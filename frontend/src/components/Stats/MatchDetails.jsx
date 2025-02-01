import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { useParams, Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { FiArrowLeft, FiUser, FiTarget, FiActivity, FiEyeOff} from 'react-icons/fi';
import '../../styles/Stats/MatchDetails.css';
import { FaGun } from "react-icons/fa6";
import { GiHeadshot, GiPositionMarker  } from "react-icons/gi";


const MatchDetails = () => {
  const { steamID, matchID } = useParams();
  const [match, setMatch] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchMatchData = async () => {
      try {
        const url = `http://localhost:8080/match/${steamID}/${matchID}`;
        const response = await axios.get(url);
        setMatch(response.data);
      } catch (err) {
        setError('Error al cargar los detalles de la partida.');
      } finally {
        setLoading(false);
      }
    };
    
    fetchMatchData();
  }, [steamID, matchID]);

  const splitPlayersByTeam = (players) => {
    console.log("Jugadores disponibles:", players);
  
    const userPlayer = players.find(p => String(p.steamID).trim() === String(steamID).trim());

    players.forEach(p => {
      console.log(`Comparando p.steamID="${p.steamID}" con param="${steamID}"`);
    });
    
  
    if (!userPlayer) {
      console.warn(`No se encontró el jugador con SteamID: ${steamID}`);
      return { myTeam: [], enemyTeam: players };
    }
  
    return {
      myTeam: players.filter(p => p.team === userPlayer.team),
      enemyTeam: players.filter(p => p.team !== userPlayer.team)
    };
  };

  if (loading) return <div className="match-details-container">Cargando...</div>;
  if (error) return <div className="match-details-container">{error}</div>;
  if (!match) return <div className="match-details-container">No se encontraron datos.</div>;

  const { myTeam, enemyTeam } = splitPlayersByTeam(match.players || []);

const getPerformanceClass = (kd) => {
  if (kd > 1.2) return "high-performer";
  if (kd < 0.8) return "low-performer";
  return "";
};

  // Función para detectar el MVP
  const getMVP = (players) => {
    return players.reduce((mvp, player) => (player.adr > mvp.adr ? player : mvp), players[0]);
  };

  const mvpPlayer = getMVP([...myTeam, ...enemyTeam]);

  const formatDuration = (duration) => {
    if (!duration || duration === "00:00") return "Desconocida";
    const [min, sec] = duration.split(":").map(Number);
    return `${min}m ${sec}s`;
};

  // Función para renderizar jugadores
  const renderPlayers = (players) => (
    
    <AnimatePresence>
      {players.map((player, index) => (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: index * 0.1 }}
          className={`player-card ${getPerformanceClass(player.kd_ratio)} ${player.steamID === mvpPlayer.steamID ? "mvp" : ""}`}
          >

            <div className="player-info">
              <img src={player.avatar ? player.avatar : "/assets/default_avatar.png"} alt="avatar" className="player-avatar" />
              <h3 style={{ color: "var(--text-primary)" }}>
                {player.name}
                {player.steamID === steamID && (
                  <span style={{ fontSize: "0.8em", color: "var(--accent-teal)", marginLeft: "0.5rem" }}>(TÚ)</span>
                )}
              </h3>
            </div>

            <div className="stat-item rank-container">
              <div className="stat-value">
                {player.rank ? (
                  <img src={`/assets/ranks/${player.rank}.png`} alt="Rango" className="rank-icon" />
                ) : (
                  "N/A"
                )}
              </div>
            </div>


          <div className="stats-grid">
            <div className="stat-item">
              <div className="stat-label"><FiTarget /> Kills</div>
              <div className="stat-value highlight-stat">{player.kills}</div>
            </div>
            <div className="stat-item">
              <div className="stat-label"><FiTarget /> Deaths</div>
              <div className="stat-value highlight-stat">{player.deaths}</div>
            </div>
            <div className="stat-item">
                <div className="stat-label"><FiActivity/>K/D</div>
                <div className="stat-value">{player.kd_ratio.toFixed(2)}</div>
                <div className="progress-bar">
                  <div className="progress-fill" style={{ width: `${(player.kd_ratio / 2) * 100}%` }}></div>
                </div>
            </div>
            <div className="stat-item">
              <div className="stat-label"><GiHeadshot  /> HS%</div>
              <div className="stat-value">{player.hs_percentage.toFixed(1)}%</div>
            </div>
            <div className="stat-item">
              <div className="stat-label">< FaGun /> ADR</div>
              <div className="stat-value">{player.adr.toFixed(1)}</div>
              <div className="progress-bar">
                <div className="progress-fill" style={{ width: `${player.adr / 2}%` }}></div>
              </div>
            </div>
            <div className="stat-item">
              <div className="stat-label"><FiEyeOff /> Flash</div>
              <div className="stat-value">{player.flash_assists}</div>
            </div>
            <div className="stat-item">
              <div className="stat-label"><GiPositionMarker />Posición</div>
              <div className="stat-value highlight-stat">{player.position ? `#${player.position}` : "N/A"} </div>
            </div>
          </div>
        </motion.div>
      ))}
    </AnimatePresence>
  );
 
  return (
    <div className="match-details-container">
      <div className="background-blobs"></div>
      
      <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.5 }}>
        <div className="match-header">
          <Link to="/HistoryGames" className="back-link">
            <FiArrowLeft /> Volver al historial
          </Link>
          <h1 style={{ color: 'var(--text-primary)', margin: '2rem 0' }}>
            {match.map_name} <span style={{ color: 'var(--text-secondary)', fontWeight: 300 }}>| {match.date}</span>
          </h1>
          
          <div className="match-info">
            <div className="info-box">
              <h3 style={{ color: 'var(--text-secondary)' }}>Resultado</h3>
              <p style={{ fontSize: '1.5rem', color: 'var(--accent-teal)' }}>
                {match.team_score} : {match.opponent_score}
              </p>
            </div>
            <div className="info-box">
              <h3 style={{ color: 'var(--text-secondary)' }}>Duración</h3>
              <p style={{ fontSize: '1.5rem', color: 'var(--text-primary)' }}>{formatDuration(match.duration)}</p>
            </div>
          </div>
        </div>
        <div className="teams-container">
          <div className="team-section my-team">
            <div className="team-header">
              <h2><FiUser /> Mi Equipo</h2>
            </div>
            {renderPlayers(myTeam)}
          </div>

          <div className="team-section enemy-team">
            <div className="team-header">
              <h2><FiUser /> Equipo Enemigo</h2>
            </div>
            {renderPlayers(enemyTeam)}
          </div>
        </div>

      </motion.div>
    </div>
  );


};

export default MatchDetails;