import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { Collapse } from '@mui/material';
import SidebarComponent from '../Layout/Sidebar'; // Ajusta la ruta si es distinta
import '../../styles/Stats/historyGames.css';

const HistoryGames = ({ user }) => {
  // Estado donde guardamos las partidas
  const [games, setGames] = useState([]);
  // Estado para controlar si está cargando
  const [loading, setLoading] = useState(true);
  // Estado para capturar errores
  const [error, setError] = useState(null);
  // Estado para controlar la expansión de detalles de cada partida
  const [expandedGameIndex, setExpandedGameIndex] = useState(null);

  // useEffect para traer las demos de nuestro endpoint de FastAPI
  useEffect(() => {
    const fetchProcessedDemos = async () => {
      try {
        const response = await axios.get('http://localhost:8000/steam/get-processed-demos', {
          withCredentials: true,
        });
        // 'response.data.demos' contendrá la lista de partidas
        setGames(response.data.demos);
      } catch (err) {
        setError('Error al obtener las partidas.');
        console.error('❌ Error al obtener las partidas:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchProcessedDemos();
  }, []);

  // Función para manejar la expansión/colapso de los detalles de una partida
  const handleToggle = (index) => {
    setExpandedGameIndex(expandedGameIndex === index ? null : index);
  };

  return (
    <div className="container">
      {/* Sidebar */}
      <SidebarComponent user={user} />

      {/* Contenido principal */}
      <div className="stats-container">
        <h2>📂 Historial de Partidas</h2>

        {loading && <p>Cargando partidas...</p>}
        {error && <p className="error-message">{error}</p>}

        {/* Si no hay error y ya cargó, mostramos las partidas */}
        {!loading && !error && games.length > 0 ? (
          <div className="games-list">
            {games.map((game, index) => (
              <div key={index} className="game-card">
                {/* Encabezado principal: resultado y marcador */}
                <h3 className={`result ${game.result?.toLowerCase()}`}>
                  {game.result?.toUpperCase()} {game.score}
                </h3>

                {/* Fecha y duración */}
                <p><strong>Fecha:</strong> {game.date}</p>
                <p><strong>Duración:</strong> {game.duration}</p>

                {/* Sección con código de compartición y kills-deaths-assists */}
                <div className="icons">
                  <span className="share-code">🔑 {game.share_code}</span>
                  <span className="kda">⭐ {game.kills} / {game.deaths} / {game.assists}</span>
                </div>

                {/* Botón para mostrar/ocultar detalles de la partida */}
                <button
                  className={`collapsible-trigger ${expandedGameIndex === index ? 'open' : ''}`}
                  onClick={() => handleToggle(index)}
                >
                  {expandedGameIndex === index ? 'Ocultar detalles de la partida' : 'Ver detalles de la partida'}
                </button>

                {/* Componente Collapse para mostrar/ocultar detalles */}
                <Collapse in={expandedGameIndex === index}>
                  <div className="collapsible-content">
                    {/* Ejemplo de scoreboard: Suponemos que game.players es un array de jugadores */}
                    {game.players && game.players.length > 0 ? (
                      <table className="scoreboard-table">
                        <thead>
                          <tr>
                            <th>Jugador</th>
                            <th>Kills</th>
                            <th>Assists</th>
                            <th>Deaths</th>
                            <th>K/D Ratio</th>
                          </tr>
                        </thead>
                        <tbody>
                          {game.players.map((player, pIndex) => (
                            <tr key={pIndex}>
                              <td>{player.name}</td>
                              <td>{player.kills}</td>
                              <td>{player.assists}</td>
                              <td>{player.deaths}</td>
                              <td>{player.kdRatio?.toFixed(2)}</td>
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
