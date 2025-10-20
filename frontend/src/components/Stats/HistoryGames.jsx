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
  const [sortBy, setSortBy] = useState('date'); // date, performance, result
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
        setFilteredGames(response.data.demos || []);
      } catch (err) {
        setError('Error al obtener las partidas.');
      } finally {
        setLoading(false);
      }
    };

    fetchProcessedDemos();
  }, [user]);

  const handleFilterChange = ({ map, dateRange, durationRange }) => {
    const { start, end } = dateRange;
    const { min, max } = durationRange;

    const filtered = games.filter((game) => {
      if (map && game.map_name !== map) return false;
      if (start && game.match_date < start) return false;
      if (end && game.match_date > end) return false;
      const duration = parseInt(game.match_duration, 10);
      if (duration < min || duration > max) return false;
      return true;
    });

    setFilteredGames(filtered);
  };

  const handleSort = (type) => {
    setSortBy(type);
    let sorted = [...filteredGames];
    
    switch(type) {
      case 'date':
        sorted.sort((a, b) => new Date(b.match_date) - new Date(a.match_date));
        break;
      case 'performance':
        sorted.sort((a, b) => (b.kd_ratio || 0) - (a.kd_ratio || 0));
        break;
      case 'result':
        sorted.sort((a, b) => {
          if (a.result === 'victory' && b.result !== 'victory') return -1;
          if (a.result !== 'victory' && b.result === 'victory') return 1;
          return 0;
        });
        break;
      default:
        break;
    }
    
    setFilteredGames(sorted);
  };

  const handleViewDetails = (matchID) => {
    navigate(`/match/${user.steam_id}/${matchID}`);
  };

  const getPlayerStats = (game) => {
    const player = game.players?.find(p => 
      String(p.steamID || p.steam_id || '').trim() === String(user.steam_id || '').trim()
    );
    return player || {};
  };

  const calculateWinRate = () => {
    if (filteredGames.length === 0) return 0;
    const wins = filteredGames.filter(g => g.result === 'victory').length;
    return ((wins / filteredGames.length) * 100).toFixed(0);
  };

  const calculateAvgKD = () => {
    if (filteredGames.length === 0) return 0;
    const totalKD = filteredGames.reduce((sum, game) => {
      const stats = getPlayerStats(game);
      return sum + (stats.kd_ratio || 0);
    }, 0);
    return (totalKD / filteredGames.length).toFixed(2);
  };

  return (
    <div className="history-games-container">
      <SidebarComponent user={user} />
      <div className="history-content">
        {/* Header con estadísticas rápidas */}
        <div className="history-header">
          <div className="history-title-section">
            <h1>Historial de Partidas</h1>
            <p className="history-subtitle">Análisis detallado de tus últimas {games.length} partidas</p>
          </div>
          
          <div className="history-quick-stats">
            <div className="quick-stat-card">
              <div className="quick-stat-value">{filteredGames.length}</div>
              <div className="quick-stat-label">Partidas</div>
            </div>
            <div className="quick-stat-card">
              <div className="quick-stat-value">{calculateWinRate()}%</div>
              <div className="quick-stat-label">Win Rate</div>
            </div>
            <div className="quick-stat-card">
              <div className="quick-stat-value">{calculateAvgKD()}</div>
              <div className="quick-stat-label">K/D Promedio</div>
            </div>
          </div>
        </div>

        {loading && (
          <div className="loading-state">
            <div className="loading-spinner"></div>
            <p>Cargando partidas...</p>
          </div>
        )}

        {error && <div className="error-state">{error}</div>}

        {!loading && !error && games.length > 0 && (
          <>
            {/* Barra de filtros y ordenamiento */}
            <div className="history-controls">
              <FilterBar
                onFilterChange={handleFilterChange}
                availableMaps={[...new Set(games.map((g) => g.map_name))]}
              />
              
              <div className="sort-controls">
                <label>Ordenar por:</label>
                <div className="sort-buttons">
                  <button 
                    className={`sort-btn ${sortBy === 'date' ? 'active' : ''}`}
                    onClick={() => handleSort('date')}
                  >
                    Fecha
                  </button>
                  <button 
                    className={`sort-btn ${sortBy === 'performance' ? 'active' : ''}`}
                    onClick={() => handleSort('performance')}
                  >
                    Rendimiento
                  </button>
                  <button 
                    className={`sort-btn ${sortBy === 'result' ? 'active' : ''}`}
                    onClick={() => handleSort('result')}
                  >
                    Resultado
                  </button>
                </div>
              </div>
            </div>

            {/* Lista de partidas */}
            {filteredGames.length > 0 ? (
              <div className="games-grid">
                {filteredGames.map((game) => {
                  const playerStats = getPlayerStats(game);
                  const isVictory = game.result === 'victory';
                  
                  return (
                    <div 
                      key={game.match_id} 
                      className={`match-card ${isVictory ? 'victory' : 'defeat'}`}
                      onClick={() => handleViewDetails(game.match_id)}
                    >
                      {/* Header con mapa y resultado */}
                      <div className="match-card-header">
                        <div 
                          className="match-map-bg"
                          style={{
                            backgroundImage: `url(/images/maps/${game.map_name}.png)`,
                          }}
                        />
                        <div className="match-header-content">
                          <div className="match-map-name">
                            {game.map_name?.replace('de_', '').toUpperCase()}
                          </div>
                          <div className={`match-result-badge ${isVictory ? 'win' : 'loss'}`}>
                            {isVictory ? 'VICTORIA' : 'DERROTA'}
                          </div>
                        </div>
                      </div>

                      {/* Score y información */}
                      <div className="match-card-body">
                        <div className="match-score">
                          <span className={`score-team ${isVictory ? 'win' : ''}`}>
                            {game.team_score}
                          </span>
                          <span className="score-separator">:</span>
                          <span className={`score-opponent ${!isVictory ? 'loss' : ''}`}>
                            {game.opponent_score}
                          </span>
                        </div>

                        {/* Estadísticas del jugador */}
                        <div className="match-player-stats">
                          <div className="player-stat">
                            <span className="stat-label">K/D</span>
                            <span className="stat-value">
                              {playerStats.kd_ratio?.toFixed(2) || '0.00'}
                            </span>
                          </div>
                          <div className="player-stat">
                            <span className="stat-label">ADR</span>
                            <span className="stat-value">
                              {playerStats.adr?.toFixed(0) || '0'}
                            </span>
                          </div>
                          <div className="player-stat">
                            <span className="stat-label">HS%</span>
                            <span className="stat-value">
                              {playerStats.hs_percentage?.toFixed(0) || '0'}%
                            </span>
                          </div>
                          <div className="player-stat">
                            <span className="stat-label">Kills</span>
                            <span className="stat-value">
                              {playerStats.kills || '0'}
                            </span>
                          </div>
                        </div>

                        {/* Información adicional */}
                        <div className="match-meta">
                          <span className="meta-item">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                              <circle cx="12" cy="12" r="10"/>
                              <polyline points="12 6 12 12 16 14"/>
                            </svg>
                            {game.match_duration} min
                          </span>
                          <span className="meta-item">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                              <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                              <line x1="16" y1="2" x2="16" y2="6"/>
                              <line x1="8" y1="2" x2="8" y2="6"/>
                              <line x1="3" y1="10" x2="21" y2="10"/>
                            </svg>
                            {game.match_date}
                          </span>
                        </div>
                      </div>

                      {/* Hover overlay */}
                      <div className="match-card-hover">
                        <div className="hover-text">Ver detalles completos</div>
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                          <line x1="5" y1="12" x2="19" y2="12"/>
                          <polyline points="12 5 19 12 12 19"/>
                        </svg>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="empty-state">
                <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="8" x2="12" y2="12"/>
                  <line x1="12" y1="16" x2="12.01" y2="16"/>
                </svg>
                <h3>No se encontraron partidas</h3>
                <p>Intenta ajustar los filtros para ver más resultados</p>
              </div>
            )}
          </>
        )}

        {!loading && !error && games.length === 0 && (
          <div className="empty-state">
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            <h3>No hay partidas analizadas</h3>
            <p>Las partidas aparecerán aquí una vez que el sistema las procese</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default HistoryGames;
