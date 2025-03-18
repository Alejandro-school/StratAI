import React, { useEffect, useState } from 'react';
import axios from 'axios';
import SidebarComponent from '../Layout/Sidebar';
import '../../styles/Stats/historyGames.css';
import { useUser } from '../../context/UserContext';
import { useNavigate } from 'react-router-dom';
import FilterBar from './FilterBar';

const HistoryGames = () => {
  const [games, setGames] = useState([]);
  const [filteredGames, setFilteredGames] = useState([]);
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

        // Suponiendo que response.data.demos trae la fecha real en "match_date"
        setGames(response.data.demos || []);
        setFilteredGames(response.data.demos || []);
      } catch (err) {
        setError('Error al obtener las partidas.');
      } finally {
        setLoading(false);
      }
    };

    fetchProcessedDemos();
  }, [user]);

  // Función para filtrar las partidas
  const handleFilterChange = ({ map, dateRange, durationRange }) => {
    const { start, end } = dateRange;
    const { min, max } = durationRange;

    const filtered = games.filter((game) => {
      // Filtrar por mapa
      if (map && game.map_name !== map) {
        return false;
      }
      // Filtrar por fecha (suponiendo que match_date es un string tipo 'YYYY-MM-DD')
      if (start && game.match_date < start) {
        return false;
      }
      if (end && game.match_date > end) {
        return false;
      }
      // Filtrar por duración (asumiendo match_duration está en minutos)
      const duration = parseInt(game.match_duration, 10);
      if (duration < min || duration > max) {
        return false;
      }
      return true;
    });

    setFilteredGames(filtered);
  };

  const handleViewDetails = (matchID) => {
    navigate(`/match/${user.steam_id}/${matchID}`);
  };

  return (
    <div className="container-historyGames">
      <SidebarComponent user={user} />
      <div className="stats-container">
        <h2>📂 Historial de Partidas</h2>

        {loading && <p>Cargando partidas...</p>}
        {error && <p className="error-message">{error}</p>}

        {/* Barra de filtros */}
        {!loading && !error && games.length > 0 && (
          <FilterBar
            onFilterChange={handleFilterChange}
            availableMaps={[...new Set(games.map((g) => g.map_name))]}
          />
        )}

        {!loading && !error && filteredGames.length > 0 ? (
          <div className="games-list">
            {filteredGames.map((game) => (
              <div key={game.match_id} className="game-card">
                <div
                  className="game-background"
                  style={{
                    backgroundImage: `url(/images/maps/${game.map_name}.png)`,
                  }}
                >
                  <div className="game-overlay">
                    <div className="header">
                      <h3 className={`result ${game.result?.toLowerCase()}`}>
                        {game.result?.toUpperCase()}
                      </h3>
                      <span className="score">
                        {game.team_score} : {game.opponent_score}
                      </span>
                    </div>

                    <div className="info">
                      <span className="date-time">
                        📅 {game.match_date}
                      </span>
                      <span className="duration">
                        ⏳ {game.match_duration} min
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
