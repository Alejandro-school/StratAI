import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { Collapse } from '@mui/material';
import SidebarComponent from '../Layout/Sidebar';
import '../../styles/Stats/historyGames.css';
import { useUser } from '../../context/UserContext';


const HistoryGames = () => {
  const [games, setGames] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [expandedGameIndex, setExpandedGameIndex] = useState(null);
  const {user} = useUser();


  useEffect(() => {
    const fetchProcessedDemos = async () => {
      try {
        // Aquí asumo que le pasas un steam_id válido.
        // Por ejemplo, si tu usuario es user.steam_id
        // Ajusta la URL según necesites:
        const steamID = user?.steam_id
        const url = `http://localhost:8080/get-processed-demos?steam_id=${steamID}`;

        const response = await axios.get(url);
        // "demos" viene del JSON => {"status": "success", "demos": [...]}
        setGames(response.data.demos || []);
      } catch (err) {
        setError('Error al obtener las partidas.');
        console.error('❌ Error al obtener las partidas:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchProcessedDemos();
  }, [user]);

  const handleToggle = (index) => {
    setExpandedGameIndex(expandedGameIndex === index ? null : index);
  };

  return (
    <div className="container-historyGames">
      <SidebarComponent user={user} />

      <div className="stats-container">
        <h2>📂 Historial de Partidas</h2>

        {loading && <p>Cargando partidas...</p>}
        {error && <p className="error-message">{error}</p>}

        {!loading && !error && games.length > 0 ? (
          <div className="games-list">
            {games.map((game, index) => (
              <div key={index} className="game-card">
                <div
                  className="game-background"
                  style={{
                    // "game.map_name" => la imagen se llamará p.ej. /images/maps/de_inferno.jpg
                    backgroundImage: `url(/images/maps/${game.map_name}.png)`,
                  }}
                >
                  <div className="game-overlay">
                    <h3 className={`result ${game.result?.toLowerCase()}`}>
                      {game.result?.toUpperCase()}
                      <span className="score">
                        {game.team_score} : {game.opponent_score}
                      </span>
                    </h3>

                    <div className="game-info">
                      <span className="date-time">
                        {/* game.date y game.duration */}
                        📅 {game.date} &nbsp;|&nbsp; ⏳ {game.duration}
                      </span>
                      {/* Podrías mostrar kills totales de tu equipo, etc. 
                          Pero en este ejemplo iremos a la tabla de players */}
                    </div>

                    <button
                      className={`collapsible-trigger ${expandedGameIndex === index ? 'open' : ''}`}
                      onClick={() => handleToggle(index)}
                    >
                      {expandedGameIndex === index ? 'Ocultar detalles' : 'Ver detalles'}
                    </button>

                    <Collapse in={expandedGameIndex === index}>
                      <div className="collapsible-content">
                        {game.players && game.players.length > 0 ? (
                          <table className="scoreboard-table">
                            <thead>
                              <tr>
                                <th>Jugador</th>
                                <th>Kills</th>
                                <th>Assists</th>
                                <th>Deaths</th>
                                <th>K/D</th>
                                <th>HS%</th>
                                <th>ADR</th>
                                <th>FlashAssists</th>
                              </tr>
                            </thead>
                            <tbody>
                              {game.players.map((player, pIndex) => (
                                <tr key={pIndex}>
                                  <td>{player.name}</td>
                                  <td>{player.kills}</td>
                                  <td>{player.assists}</td>
                                  <td>{player.deaths}</td>
                                  <td>{player.kd_ratio.toFixed(2)}</td>
                                  <td>{player.hs_percentage.toFixed(1)}%</td>
                                  <td>{player.adr.toFixed(2)}</td>
                                  <td>{player.flash_assists}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        ) : (
                          <p>No hay estadísticas de jugadores para esta partida.</p>
                        )}
                      </div>
                    </Collapse>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          !loading && <p>No hay partidas disponibles.</p>
        )}
      </div>
    </div>
  );
};

export default HistoryGames;
