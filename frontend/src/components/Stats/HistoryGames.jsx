import React, { useEffect, useState } from 'react';
import axios from 'axios';
import SidebarComponent from '../Layout/Sidebar';
import '../../styles/Stats/historyGames.css';
import { useUser } from '../../context/UserContext';
import { useNavigate } from 'react-router-dom';

const HistoryGames = () => {
  const [games, setGames] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const { user } = useUser();
  const navigate = useNavigate();

  useEffect(() => {
    const fetchProcessedDemos = async () => {
      try {
        if (!user?.steam_id) return;

        const steamID = user.steam_id;
        const url = `http://localhost:8080/get-processed-demos?steam_id=${steamID}`;
        const response = await axios.get(url);
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

  const handleViewDetails = (matchID) => {
    // Navegamos a la ruta /match/steamID/matchID
    navigate(`/match/${user.steam_id}/${matchID}`);
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
            {games.map((game) => (
              <div key={game.match_id} className="game-card">
                <div
                  className="game-background"
                  style={{
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
                        📅 {game.date} &nbsp;|&nbsp; ⏳ {game.duration}
                      </span>
                    </div>

                    <button
                      className="details-button"
                      onClick={() => handleViewDetails(game.match_id)}
                    >
                      Ver detalles
                    </button>
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
