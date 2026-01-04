// frontend/src/components/Match/MatchDetails.jsx
// Main page component for displaying match details
import React, { useState, useEffect, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import axios from 'axios';
import NavigationFrame from '../Layout/NavigationFrame';
import { useUser } from '../../context/UserContext';
import { 
  ArrowLeft, Clock, Users, Target, Crosshair, Zap, Trophy, 
  Sword, TrendingUp, Activity, Flame
} from 'lucide-react';

import '../../styles/Match/matchDetails.css';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

// Map images
const getMapImage = (mapName) => {
  const maps = {
    'de_dust2': '/images/maps/de_dust2.png',
    'de_mirage': '/images/maps/de_mirage.png',
    'de_inferno': '/images/maps/de_inferno.png',
    'de_nuke': '/images/maps/de_nuke.png',
    'de_overpass': '/images/maps/de_overpass.png',
    'de_ancient': '/images/maps/de_ancient.png',
    'de_anubis': '/images/maps/de_anubis.png',
    'de_vertigo': '/images/maps/de_vertigo.png',
  };
  return maps[mapName] || maps['de_dust2'];
};

// Format duration
const formatDuration = (seconds) => {
  if (!seconds) return '0m';
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};

// K/D color
const getKDColor = (kd) => {
  if (kd >= 1.5) return '#22c55e';
  if (kd >= 1.0) return '#84cc16';
  if (kd >= 0.8) return '#f59e0b';
  return '#ef4444';
};

// Rating color
const getRatingColor = (rating) => {
  if (rating >= 1.3) return '#22c55e';
  if (rating >= 1.0) return '#84cc16';
  if (rating >= 0.8) return '#f59e0b';
  return '#ef4444';
};

const MatchDetails = () => {
  const { matchID } = useParams();
  const navigate = useNavigate();
  const { user } = useUser();
  
  const [matchData, setMatchData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Fetch match details
  useEffect(() => {
    const fetchMatchDetails = async () => {
      if (!matchID) return;
      
      try {
        setLoading(true);
        console.log('📡 Fetching match details:', matchID);
        
        const response = await axios.get(
          `${API_URL}/steam/get-match-details/${matchID}`,
          { withCredentials: true }
        );
        
        console.log('✅ Match details loaded:', response.data);
        setMatchData(response.data);
        setError(null);
      } catch (err) {
        console.error('Error fetching match details:', err);
        setError('No se pudieron cargar los detalles de la partida.');
      } finally {
        setLoading(false);
      }
    };

    fetchMatchDetails();
  }, [matchID]);

  // Get current user stats
  const userStats = useMemo(() => {
    return matchData?.current_user || null;
  }, [matchData]);

  // Format map name
  const mapDisplayName = useMemo(() => {
    const name = matchData?.metadata?.map_name || 'unknown';
    return name.replace('de_', '').charAt(0).toUpperCase() + 
           name.replace('de_', '').slice(1);
  }, [matchData]);

  if (loading) {
    return (
      <NavigationFrame>
        <div className="match-details-container">
          <div className="loading-state">
            <div className="spinner" />
            <p>Cargando detalles...</p>
          </div>
        </div>
      </NavigationFrame>
    );
  }

  if (error || !matchData) {
    return (
      <NavigationFrame>
        <div className="match-details-container">
          <div className="error-state">
            <h2>Error</h2>
            <p>{error || 'Partida no encontrada'}</p>
            <button onClick={() => navigate('/history-games')}>
              <ArrowLeft size={16} />
              Volver al historial
            </button>
          </div>
        </div>
      </NavigationFrame>
    );
  }

  const { metadata, result, team_ct, team_t } = matchData;
  const isVictory = result === 'victory';

  return (
    <NavigationFrame>
      <div className="match-details-container">
        {/* Header */}
        <div 
          className={`match-header ${isVictory ? 'victory' : 'defeat'}`}
          style={{ backgroundImage: `url(${getMapImage(metadata.map_name)})` }}
        >
          <div className="header-overlay" />
          <div className="header-content">
            <button className="back-btn" onClick={() => navigate('/history-games')}>
              <ArrowLeft size={18} />
              Volver
            </button>
            
            <div className="match-result">
              <span className="map-name">{mapDisplayName}</span>
              <div className={`result-badge ${isVictory ? 'win' : 'loss'}`}>
                {isVictory ? 'VICTORIA' : 'DERROTA'}
              </div>
            </div>
            
            <div className="score-display">
              <span className={`team-score ${isVictory ? 'winner' : ''}`}>
                {metadata.team_score}
              </span>
              <span className="score-divider">-</span>
              <span className={`team-score ${!isVictory ? 'winner' : ''}`}>
                {metadata.opponent_score}
              </span>
            </div>
            
            <div className="match-meta">
              <span><Clock size={14} /> {formatDuration(metadata.duration_seconds)}</span>
              <span><Users size={14} /> {metadata.total_rounds} rounds</span>
            </div>
          </div>
        </div>

        {/* Scoreboard */}
        <section className="section scoreboard-section">
          <h2><Trophy size={20} /> Scoreboard</h2>
          
          {/* Determine which team is "my team" based on current user */}
          {(() => {
            const userTeam = matchData.current_user?.team;
            const myTeam = userTeam === "CT" ? team_ct : team_t;
            const enemyTeam = userTeam === "CT" ? team_t : team_ct;
            const myTeamWon = isVictory;
            
            return (
              <>
                {/* My Team */}
                <div className={`team-table ${myTeamWon ? 'winner-team' : 'loser-team'}`}>
                  <div className="team-header">
                    <Users size={16} />
                    <span>Mi Equipo</span>
                    <span className={`team-result-badge ${myTeamWon ? 'win' : 'loss'}`}>
                      {myTeamWon ? 'WIN' : 'LOSE'}
                    </span>
                  </div>
                  <table>
                    <thead>
                      <tr>
                        <th>Jugador</th>
                        <th>K</th>
                        <th>D</th>
                        <th>A</th>
                        <th>K/D</th>
                        <th>ADR</th>
                        <th>HS%</th>
                        <th>Rating</th>
                        <th>KAST</th>
                      </tr>
                    </thead>
                    <tbody>
                      {myTeam.map((player, idx) => (
                        <tr 
                          key={player.steam_id} 
                          className={player.steam_id === matchData.current_user_steam_id ? 'current-user' : ''}
                        >
                          <td className="player-name">
                            {idx === 0 && <span className="mvp-badge">★</span>}
                            {player.name}
                          </td>
                          <td>{player.kills}</td>
                          <td>{player.deaths}</td>
                          <td>{player.assists}</td>
                          <td style={{ color: getKDColor(player.kd_ratio) }}>
                            {player.kd_ratio?.toFixed(2)}
                          </td>
                          <td>{Math.round(player.adr)}</td>
                          <td>{Math.round(player.hs_percentage)}%</td>
                          <td style={{ color: getRatingColor(player.hltv_rating) }}>
                            {player.hltv_rating?.toFixed(2)}
                          </td>
                          <td>{Math.round(player.kast)}%</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                
                {/* Enemy Team */}
                <div className={`team-table ${!myTeamWon ? 'winner-team' : 'loser-team'}`}>
                  <div className="team-header enemy">
                    <Sword size={16} />
                    <span>Equipo Enemigo</span>
                    <span className={`team-result-badge ${!myTeamWon ? 'win' : 'loss'}`}>
                      {!myTeamWon ? 'WIN' : 'LOSE'}
                    </span>
                  </div>
                  <table>
                    <thead>
                      <tr>
                        <th>Jugador</th>
                        <th>K</th>
                        <th>D</th>
                        <th>A</th>
                        <th>K/D</th>
                        <th>ADR</th>
                        <th>HS%</th>
                        <th>Rating</th>
                        <th>KAST</th>
                      </tr>
                    </thead>
                    <tbody>
                      {enemyTeam.map((player, idx) => (
                        <tr key={player.steam_id}>
                          <td className="player-name">
                            {idx === 0 && <span className="mvp-badge">★</span>}
                            {player.name}
                          </td>
                          <td>{player.kills}</td>
                          <td>{player.deaths}</td>
                          <td>{player.assists}</td>
                          <td style={{ color: getKDColor(player.kd_ratio) }}>
                            {player.kd_ratio?.toFixed(2)}
                          </td>
                          <td>{Math.round(player.adr)}</td>
                          <td>{Math.round(player.hs_percentage)}%</td>
                          <td style={{ color: getRatingColor(player.hltv_rating) }}>
                            {player.hltv_rating?.toFixed(2)}
                          </td>
                          <td>{Math.round(player.kast)}%</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            );
          })()}
        </section>

        {/* Your Performance */}
        {userStats && (
          <section className="section performance-section">
            <h2><TrendingUp size={20} /> Tu Rendimiento</h2>
            
            <div className="stats-grid">
              {/* Overview Card */}
              <div className="stat-card overview">
                <h3><Activity size={16} /> Resumen</h3>
                <div className="stat-items">
                  <div className="stat-item">
                    <span className="value" style={{ color: getRatingColor(userStats.hltv_rating) }}>
                      {userStats.hltv_rating?.toFixed(2)}
                    </span>
                    <span className="label">Rating HLTV</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.impact_rating?.toFixed(2)}</span>
                    <span className="label">Impact</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{Math.round(userStats.kast)}%</span>
                    <span className="label">KAST</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.rounds_survived}</span>
                    <span className="label">Rondas vivo</span>
                  </div>
                </div>
              </div>

              {/* Combat Card */}
              <div className="stat-card combat">
                <h3><Crosshair size={16} /> Combate</h3>
                <div className="stat-items">
                  <div className="stat-item">
                    <span className="value">
                      {userStats.opening_duels_won}/{userStats.opening_duels_attempted}
                    </span>
                    <span className="label">Opening duels</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.trade_kills}</span>
                    <span className="label">Trade kills</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">
                      {(userStats.clutches_1v1_won || 0) + (userStats.clutches_1v2_won || 0)}
                    </span>
                    <span className="label">Clutches</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.traded_deaths}</span>
                    <span className="label">Traded deaths</span>
                  </div>
                </div>
                
                {/* Multikills */}
                {userStats.multikills && Object.keys(userStats.multikills).length > 0 && (
                  <div className="multikills">
                    {userStats.multikills['2k'] && <span className="mk">2K: {userStats.multikills['2k']}</span>}
                    {userStats.multikills['3k'] && <span className="mk highlight">3K: {userStats.multikills['3k']}</span>}
                    {userStats.multikills['4k'] && <span className="mk highlight">4K: {userStats.multikills['4k']}</span>}
                    {userStats.multikills['5k'] && <span className="mk ace">ACE: {userStats.multikills['5k']}</span>}
                  </div>
                )}
              </div>

              {/* Accuracy Card */}
              <div className="stat-card accuracy">
                <h3><Target size={16} /> Precisión</h3>
                <div className="stat-items">
                  <div className="stat-item">
                    <span className="value">{userStats.accuracy_overall?.toFixed(1)}%</span>
                    <span className="label">Accuracy</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.shots_hit}/{userStats.shots_fired}</span>
                    <span className="label">Hits/Shots</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.time_to_damage_avg_ms?.toFixed(0)}ms</span>
                    <span className="label">TTD avg</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.crosshair_placement_avg_error?.toFixed(1)}°</span>
                    <span className="label">Crosshair error</span>
                  </div>
                </div>
                
                {/* Body hits */}
                {userStats.body_part_hits && (
                  <div className="body-hits">
                    <span>Head: {userStats.body_part_hits.head || 0}</span>
                    <span>Chest: {userStats.body_part_hits.chest || 0}</span>
                    <span>Stomach: {userStats.body_part_hits.stomach || 0}</span>
                  </div>
                )}
              </div>

              {/* Utility Card */}
              <div className="stat-card utility">
                <h3><Flame size={16} /> Utilidad</h3>
                <div className="stat-items">
                  <div className="stat-item">
                    <span className="value">{userStats.utility_damage}</span>
                    <span className="label">Utility damage</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.enemies_flashed_per_flash?.toFixed(1)}</span>
                    <span className="label">Flashed/flash</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.flash_assists}</span>
                    <span className="label">Flash assists</span>
                  </div>
                  <div className="stat-item">
                    <span className="value">{userStats.grenades_thrown_total}</span>
                    <span className="label">Grenades</span>
                  </div>
                </div>
                
                <div className="grenade-breakdown">
                  <span>🔵 Flash: {userStats.flashes_thrown}</span>
                  <span>💥 HE: {userStats.he_thrown}</span>
                  <span>🔥 Molotov: {userStats.molotovs_thrown}</span>
                  <span>💨 Smoke: {userStats.smokes_thrown}</span>
                </div>
              </div>
            </div>
          </section>
        )}

        {/* Weapon Performance */}
        {userStats?.weapon_stats && Object.keys(userStats.weapon_stats).length > 0 && (
          <section className="section weapons-section">
            <h2><Zap size={20} /> Rendimiento por Arma</h2>
            <div className="weapons-grid">
              {Object.entries(userStats.weapon_stats)
                .filter(([_, stats]) => stats.kills > 0 || stats.damage > 0)
                .sort((a, b) => b[1].kills - a[1].kills)
                .slice(0, 8)
                .map(([weapon, stats]) => (
                  <div key={weapon} className="weapon-card">
                    <div className="weapon-name">{weapon}</div>
                    <div className="weapon-stats">
                      <div className="ws">
                        <span className="val">{stats.kills}</span>
                        <span className="lbl">Kills</span>
                      </div>
                      <div className="ws">
                        <span className="val">{stats.headshots}</span>
                        <span className="lbl">HS</span>
                      </div>
                      <div className="ws">
                        <span className="val">{Math.round(stats.accuracy)}%</span>
                        <span className="lbl">Acc</span>
                      </div>
                      <div className="ws">
                        <span className="val">{stats.damage}</span>
                        <span className="lbl">DMG</span>
                      </div>
                    </div>
                  </div>
                ))}
            </div>
          </section>
        )}
      </div>
    </NavigationFrame>
  );
};

export default MatchDetails;
