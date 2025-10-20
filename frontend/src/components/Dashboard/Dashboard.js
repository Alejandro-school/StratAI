// frontend/src/components/Dashboard/Dashboard.js - DISEÑO MEJORADO CON AIMLAB BACKGROUND
import React from "react";
import { Link } from "react-router-dom";
import Header from "../Layout/Header";
import SidebarComponent from "../Layout/Sidebar";
import { useUser } from "../../context/UserContext";
import { useDashboardData } from "../../hooks/useDashboardData";
import AimLabBackground from "./AimLabBackground";
import { 
  TrendingUp, TrendingDown, Target, Award, Zap, Activity,
  Play, MapPin, Crosshair, BarChart3, ArrowRight, Upload
} from "lucide-react";
import { 
  AreaChart, Area, XAxis, YAxis, ResponsiveContainer, 
  Tooltip
} from "recharts";
import "../../styles/Dashboard/dashboard.css";
import "../../styles/Dashboard/aimLabBackground.css";

const API_URL = process.env.REACT_APP_API_URL || "http://localhost:8000";

// KPI Card Component MEJORADO con gradientes vibrantes
const KPICard = ({ icon: Icon, label, value, suffix = "", decimals = 0, change, trend, gradient }) => {
  return (
    <div className="kpi-card" style={{ '--gradient': gradient }}>
      <div className="kpi-card-bg"></div>
      <div className="kpi-icon" style={{ background: gradient }}>
        <Icon size={20} />
      </div>
      <div className="kpi-content">
        <span className="kpi-label">{label}</span>
        <span className="kpi-value">
          {decimals > 0 ? value.toFixed(decimals) : value}{suffix}
        </span>
        <div className={`kpi-trend ${trend}`}>
          {trend === "up" ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
          <span>{change > 0 ? '+' : ''}{decimals > 0 ? change.toFixed(2) : change}{suffix}</span>
        </div>
      </div>
    </div>
  );
};

const Dashboard = () => {
  const { user } = useUser();
  const { dashboardData, loading, error } = useDashboardData(user);

  if (loading) {
    return (
      <>
        <Header />
        <div className="layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="loading-container">
              <div className="spinner"></div>
              <p>Cargando estadísticas...</p>
            </div>
          </div>
        </div>
      </>
    );
  }

  if (error) {
    return (
      <>
        <Header />
        <div className="layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="error-container">
              <h3>Error al cargar el dashboard</h3>
              <p>{error}</p>
            </div>
          </div>
        </div>
      </>
    );
  }

  if (!dashboardData || dashboardData.totalMatches === 0) {
    return (
      <>
        <Header />
        <div className="layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="no-data-container">
              <h2>No hay partidas analizadas</h2>
              <p>Sube demos de partidas para ver tus estadísticas y rendimiento</p>
              <Link to="/MatchSummary" className="action-btn action-btn-primary">
                <Upload size={18} />
                <span>Subir Demo</span>
              </Link>
            </div>
          </div>
        </div>
      </>
    );
  }

  const { 
    mainStats, 
    performanceData,
    recentMatches,
    weaponStats,
    mapStats,
    advancedStats
  } = dashboardData;

  return (
    <>
      <Header />
      <div className="layout">
        <SidebarComponent />
        <div className="main-content">
          {/* FONDO INTERACTIVO AIM LAB */}
          <AimLabBackground />
          
          <div className="dashboard-improved">
            {/* Header mejorado */}
            <div className="dashboard-header">
              <div className="header-left">
                <h1 className="dashboard-title">
                  Hola, <span className="highlight">{user?.username || "Jugador"}</span>
                </h1>
                <p className="dashboard-subtitle">
                  Aquí está tu rendimiento reciente y estadísticas clave
                </p>
              </div>
              <div className="header-actions">
                <Link to="/history-games" className="action-btn action-btn-secondary">
                  <BarChart3 size={18} />
                  <span>Ver partidas</span>
                </Link>
                <Link to="/MatchSummary" className="action-btn action-btn-primary">
                  <Upload size={18} />
                  <span>Analizar demo</span>
                </Link>
              </div>
            </div>

            {/* KPIs Row con gradientes vibrantes */}
            <div className="kpi-row">
              <KPICard
                icon={Target}
                label="Headshot %"
                value={mainStats.headshot.value}
                suffix="%"
                change={mainStats.headshot.change}
                trend={mainStats.headshot.trend}
                gradient="linear-gradient(135deg, #667eea 0%, #764ba2 100%)"
              />
              <KPICard
                icon={Activity}
                label="ADR"
                value={mainStats.adr.value}
                change={mainStats.adr.change}
                trend={mainStats.adr.trend}
                gradient="linear-gradient(135deg, #f093fb 0%, #f5576c 100%)"
              />
              <KPICard
                icon={Award}
                label="K/D Ratio"
                value={mainStats.kd.value}
                decimals={2}
                change={mainStats.kd.change}
                trend={mainStats.kd.trend}
                gradient="linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)"
              />
              <KPICard
                icon={Zap}
                label="Win Rate"
                value={mainStats.winRate.value}
                suffix="%"
                change={mainStats.winRate.change}
                trend={mainStats.winRate.trend}
                gradient="linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)"
              />
            </div>

            {/* Main Dashboard Grid */}
            <div className="dashboard-grid">
              
              {/* Performance Chart */}
              <div className="grid-item grid-large">
                <div className="widget-header">
                  <div>
                    <h3 className="widget-title">Evolución de Rendimiento</h3>
                    <p className="widget-subtitle">
                      {performanceData.length > 0 
                        ? `Últimas ${performanceData.length} partida${performanceData.length > 1 ? 's' : ''} · K/D y ADR`
                        : 'Sin datos de partidas'
                      }
                    </p>
                  </div>
                  {performanceData.length > 0 && (
                    <div className="chart-legend-inline">
                      <span className="legend-item">
                        <span className="legend-dot" style={{ background: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)" }} />
                        K/D
                      </span>
                      <span className="legend-item">
                        <span className="legend-dot" style={{ background: "linear-gradient(135deg, #f093fb 0%, #f5576c 100%)" }} />
                        ADR
                      </span>
                    </div>
                  )}
                </div>
                {performanceData.length > 0 ? (
                  <div className="chart-wrapper">
                    <ResponsiveContainer width="100%" height={280}>
                      <AreaChart data={performanceData}>
                      <defs>
                        <linearGradient id="kdGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor="#667eea" stopOpacity={0.6} />
                          <stop offset="100%" stopColor="#764ba2" stopOpacity={0.1} />
                        </linearGradient>
                        <linearGradient id="adrGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor="#f093fb" stopOpacity={0.6} />
                          <stop offset="100%" stopColor="#f5576c" stopOpacity={0.1} />
                        </linearGradient>
                      </defs>
                      <XAxis 
                        dataKey="match" 
                        stroke="rgba(255,255,255,0.3)"
                        tick={{ fill: "rgba(255,255,255,0.6)", fontSize: 12 }}
                      />
                      <YAxis 
                        stroke="rgba(255,255,255,0.3)"
                        tick={{ fill: "rgba(255,255,255,0.6)", fontSize: 12 }}
                      />
                      <Tooltip
                        contentStyle={{
                          background: "rgba(15, 23, 42, 0.95)",
                          border: "1px solid rgba(102, 126, 234, 0.3)",
                          borderRadius: 12,
                          backdropFilter: "blur(12px)",
                          padding: "12px",
                        }}
                        labelStyle={{ color: "#f1f5f9", marginBottom: 8 }}
                      />
                      <Area
                        type="monotone"
                        dataKey="kd"
                        stroke="#667eea"
                        strokeWidth={3}
                        fill="url(#kdGrad)"
                        dot={{ fill: "#667eea", strokeWidth: 2, r: 5 }}
                      />
                      <Area
                        type="monotone"
                        dataKey="adr"
                        stroke="#f093fb"
                        strokeWidth={3}
                        fill="url(#adrGrad)"
                        dot={{ fill: "#f093fb", strokeWidth: 2, r: 5 }}
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
                ) : (
                  <div className="no-chart-data">
                    <p>No hay suficientes datos para mostrar el gráfico</p>
                  </div>
                )}
              </div>

              {/* Weapon Performance */}
              <div className="grid-item grid-medium">
                <div className="widget-header">
                  <h3 className="widget-title">Armas Preferidas</h3>
                  <p className="widget-subtitle">Top 4 por kills</p>
                </div>
                <div className="weapon-list">
                  {weaponStats.length > 0 ? weaponStats.map((weapon, idx) => {
                    const gradients = [
                      'linear-gradient(135deg, #ffd700 0%, #ffed4e 100%)',
                      'linear-gradient(135deg, #c0c0c0 0%, #e8e8e8 100%)',
                      'linear-gradient(135deg, #cd7f32 0%, #d4965f 100%)',
                      'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    ];
                    const maxKills = weaponStats[0]?.kills || 1;
                    return (
                      <div key={idx} className="weapon-item">
                        <div className="weapon-rank" style={{ background: gradients[idx] }}>
                          #{idx + 1}
                        </div>
                        <div className="weapon-info">
                          <Crosshair size={16} className="weapon-icon" />
                          <span className="weapon-name">{weapon.weapon}</span>
                        </div>
                        <div className="weapon-stats">
                          <span className="weapon-kills">{weapon.kills}</span>
                          <div className="weapon-bar">
                            <div 
                              className="weapon-bar-fill" 
                              style={{ 
                                width: `${(weapon.kills / maxKills) * 100}%`,
                                background: gradients[idx]
                              }}
                            />
                          </div>
                        </div>
                      </div>
                    );
                  }) : (
                    <div className="no-data-message">
                      <Target size={40} opacity={0.3} />
                      <p style={{ marginBottom: '8px', fontWeight: 600 }}>Sin datos de armas</p>
                      <p style={{ fontSize: '12px', color: 'rgba(255,255,255,0.5)' }}>
                        Las demos necesitan reprocesarse.<br/>
                        Ejecuta: <code style={{ background: 'rgba(0,0,0,0.3)', padding: '2px 6px', borderRadius: '4px' }}>
                          backend\go-service\scripts\run_reprocess.bat
                        </code>
                      </p>
                    </div>
                  )}
                </div>
              </div>

              {/* Map Performance */}
              <div className="grid-item grid-medium">
                <div className="widget-header">
                  <h3 className="widget-title">Rendimiento por Mapa</h3>
                  <p className="widget-subtitle">Win rate últimas 20 partidas</p>
                </div>
                <div className="map-list">
                  {mapStats.map((map, idx) => (
                    <div key={idx} className="map-item">
                      <div className="map-header">
                        <MapPin size={14} className="map-icon" />
                        <span className="map-name">{map.map.replace('de_', '').toUpperCase()}</span>
                        <span className="map-winrate">{map.winRate}%</span>
                      </div>
                      <div className="map-progress">
                        <div 
                          className="map-progress-bar" 
                          style={{ 
                            width: `${map.winRate}%`,
                            background: map.winRate >= 55 ? "linear-gradient(90deg, #10b981 0%, #34d399 100%)" : 
                                       map.winRate >= 50 ? "linear-gradient(90deg, #f59e0b 0%, #fbbf24 100%)" : 
                                       "linear-gradient(90deg, #ef4444 0%, #f87171 100%)"
                          }}
                        />
                      </div>
                      <div className="map-record">
                        <span className="wins">{map.wins}W</span>
                        <span className="losses">{map.losses}L</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Momentos Destacados */}
              <div className="grid-item grid-medium">
                <div className="widget-header">
                  <h3 className="widget-title">Momentos Destacados</h3>
                  <p className="widget-subtitle">Tus mejores jugadas</p>
                </div>
                <div className="highlights-grid">
                  <div className="highlight-card">
                    <div className="highlight-icon ace">
                      <Award size={20} />
                    </div>
                    <div className="highlight-content">
                      <div className="highlight-value">{advancedStats.multiKills.ace}</div>
                      <div className="highlight-label">ACEs</div>
                    </div>
                  </div>
                  
                  <div className="highlight-card">
                    <div className="highlight-icon quad">
                      <Target size={20} />
                    </div>
                    <div className="highlight-content">
                      <div className="highlight-value">{advancedStats.multiKills.quad}</div>
                      <div className="highlight-label">4K</div>
                    </div>
                  </div>

                  <div className="highlight-card">
                    <div className="highlight-icon triple">
                      <Crosshair size={20} />
                    </div>
                    <div className="highlight-content">
                      <div className="highlight-value">{advancedStats.multiKills.triple}</div>
                      <div className="highlight-label">3K</div>
                    </div>
                  </div>

                  <div className="highlight-card">
                    <div className="highlight-icon clutch">
                      <Zap size={20} />
                    </div>
                    <div className="highlight-content">
                      <div className="highlight-value">{advancedStats.clutchesWon}</div>
                      <div className="highlight-label">Clutches</div>
                    </div>
                  </div>

                  <div className="highlight-card">
                    <div className="highlight-icon entry">
                      <Activity size={20} />
                    </div>
                    <div className="highlight-content">
                      <div className="highlight-value">{advancedStats.entryFrags}</div>
                      <div className="highlight-label">Entry Kills</div>
                    </div>
                  </div>

                  <div className="highlight-card">
                    <div className="highlight-icon utility">
                      <Zap size={20} />
                    </div>
                    <div className="highlight-content">
                      <div className="highlight-value">{advancedStats.utilityDamage}</div>
                      <div className="highlight-label">Utility DMG</div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Recent Matches */}
              <div className="grid-item grid-full">
                <div className="widget-header">
                  <div>
                    <h3 className="widget-title">Partidas Recientes</h3>
                    <p className="widget-subtitle">
                      {recentMatches.length > 0 
                        ? `Últimas ${recentMatches.length} competitiva${recentMatches.length > 1 ? 's' : ''}`
                        : 'Sin partidas'
                      }
                    </p>
                  </div>
                  {recentMatches.length > 0 && (
                    <Link to="/history-games" className="view-all-link">
                      Ver todas <ArrowRight size={16} />
                    </Link>
                  )}
                </div>
                {recentMatches.length > 0 ? (
                  <div className="matches-table-wrapper">
                    <table className="matches-table">
                      <thead>
                        <tr>
                          <th>Mapa</th>
                          <th>Resultado</th>
                          <th>K</th>
                          <th>D</th>
                          <th>A</th>
                          <th>K/D</th>
                          <th>ADR</th>
                          <th>HS%</th>
                          <th>Fecha</th>
                        </tr>
                      </thead>
                      <tbody>
                        {recentMatches.map((match, idx) => (
                          <tr key={idx} className={match.result === "Victoria" ? "match-win" : "match-loss"}>
                            <td className="match-map">
                              <MapPin size={14} />
                              {match.map.replace('de_', '').toUpperCase()}
                            </td>
                            <td>
                              <span className={`match-result ${match.result === "Victoria" ? "win" : "loss"}`}>
                                {match.result === "Victoria" ? "Victoria" : "Derrota"}
                              </span>
                            </td>
                            <td className="stat-highlight">{match.kills || 0}</td>
                            <td>{match.deaths || 0}</td>
                            <td>{match.assists || 0}</td>
                            <td className="stat-highlight">{typeof match.kd === 'number' ? match.kd.toFixed(2) : '0.00'}</td>
                            <td>{match.adr ? Math.round(match.adr) : 0}</td>
                            <td>{match.hsp ? Math.round(match.hsp) : 0}%</td>
                            <td className="match-date">{match.date || 'N/A'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <div className="no-matches">
                    <Play size={40} opacity={0.3} />
                    <p>No hay partidas recientes</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
};

export default Dashboard;
