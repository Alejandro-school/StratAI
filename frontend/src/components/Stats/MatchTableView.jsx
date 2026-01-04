// frontend/src/components/Stats/MatchTableView.jsx
// Compact table view for matches
import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Eye, Play, Trophy, Skull, ChevronRight } from 'lucide-react';

const MatchTableView = ({ games, getPlayerStats, onViewDetails }) => {
  const navigate = useNavigate();

  // Format date nicely
  const formatDate = (dateStr) => {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);
    
    if (date.toDateString() === today.toDateString()) {
      return `Hoy ${date.toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit' })}`;
    }
    if (date.toDateString() === yesterday.toDateString()) {
      return `Ayer ${date.toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit' })}`;
    }
    return date.toLocaleDateString('es-ES', { 
      day: '2-digit',
      month: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  // Get K/D class
  const getKDClass = (kd) => {
    if (kd >= 1.5) return 'excellent';
    if (kd >= 1.0) return 'good';
    if (kd >= 0.8) return 'average';
    return 'poor';
  };

  if (games.length === 0) {
    return (
      <div className="table-empty">
        <p>No hay partidas para mostrar</p>
      </div>
    );
  }

  return (
    <div className="match-table-container">
      <table className="match-table">
        <thead>
          <tr>
            <th className="col-map">Mapa</th>
            <th className="col-date">Fecha</th>
            <th className="col-score">Score</th>
            <th className="col-kd">K/D</th>
            <th className="col-adr">ADR</th>
            <th className="col-hs">HS%</th>
            <th className="col-kills">Kills</th>
            <th className="col-result">Resultado</th>
            <th className="col-actions">Acciones</th>
          </tr>
        </thead>
        <tbody>
          {games.map(game => {
            const playerStats = getPlayerStats(game);
            const isVictory = game.result === 'victory';
            const kdRatio = playerStats?.kd_ratio || 0;

            return (
              <tr 
                key={game.match_id} 
                className={`table-row ${isVictory ? 'victory' : 'defeat'}`}
                onClick={() => onViewDetails(game.match_id)}
              >
                <td className="col-map">
                  <div className="map-cell">
                    <div 
                      className="map-mini-icon"
                      style={{ 
                        backgroundImage: `url(/images/maps/${game.map_name}.png)` 
                      }}
                    />
                    <span className="map-name">
                      {game.map_name?.replace('de_', '').toUpperCase()}
                    </span>
                  </div>
                </td>
                <td className="col-date">{formatDate(game.match_date)}</td>
                <td className="col-score">
                  <span className={`score ${isVictory ? 'win' : 'loss'}`}>
                    {game.team_score}:{game.opponent_score}
                  </span>
                </td>
                <td className={`col-kd ${getKDClass(kdRatio)}`}>
                  {kdRatio.toFixed(2)}
                </td>
                <td className="col-adr">
                  {Math.round(playerStats?.adr || 0)}
                </td>
                <td className="col-hs">
                  {Math.round(playerStats?.hs_percentage || 0)}%
                </td>
                <td className="col-kills">
                  {playerStats?.kills || 0}
                </td>
                <td className="col-result">
                  <div className={`result-badge-compact ${isVictory ? 'win' : 'loss'}`}>
                    {isVictory ? (
                      <>
                        <Trophy size={12} />
                        <span>WIN</span>
                      </>
                    ) : (
                      <>
                        <Skull size={12} />
                        <span>LOSS</span>
                      </>
                    )}
                  </div>
                </td>
                <td className="col-actions">
                  <div className="action-buttons">
                    <button 
                      className="action-btn primary"
                      onClick={(e) => {
                        e.stopPropagation();
                        onViewDetails(game.match_id);
                      }}
                      title="Ver detalles"
                    >
                      <Eye size={14} />
                      <span>Ver</span>
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

export default MatchTableView;
