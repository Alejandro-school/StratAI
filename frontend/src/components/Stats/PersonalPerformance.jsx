// frontend/src/components/Stats/PersonalPerformance.jsx
import React, { useState } from "react";
import Header from "../Layout/Header";
import SidebarComponent from "../Layout/Sidebar";
import Card from "./common/Card";
import EmptyState from "./common/EmptyState";
import ErrorState from "./common/ErrorState";
import { Target, Activity, Award, TrendingUp, TrendingDown, Crosshair, Zap } from "lucide-react";
import { useUser } from "../../context/UserContext";
import { usePersonalPerformance } from "../../hooks/usePersonalPerformance";
import "../../styles/Stats/personalPerformance.css";

/**
 * PersonalPerformance - Componente de resumen DETALLADO de desempeño personal del jugador
 * Muestra estadísticas MÁS profundas que el Dashboard
 */
export default function PersonalPerformance() {
  const { user } = useUser();
  const { data, loading, error } = usePersonalPerformance(user);
  const [selectedFilter, setSelectedFilter] = useState("all");
  const [selectedSide, setSelectedSide] = useState("both");

  // Estado de carga
  if (loading) {
    return (
      <>
        <Header />
        <div className="dashboard-layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="main-scroll">
              <div className="pp-container">
                <div className="pp-loading">
                  <div className="pp-spinner"></div>
                  <p>Cargando estadísticas detalladas...</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  // Estado de error
  if (error) {
    return (
      <>
        <Header />
        <div className="dashboard-layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="main-scroll">
              <div className="pp-container">
                <Card>
                  <ErrorState 
                    message={error} 
                    onRetry={() => window.location.reload()} 
                  />
                </Card>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  // Estado vacío
  if (!data) {
    return (
      <>
        <Header />
        <div className="dashboard-layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="main-scroll">
              <div className="pp-container">
                <Card>
                  <EmptyState 
                    icon={Target}
                    title="No hay datos de rendimiento"
                    description="Juega algunas partidas para ver tus estadísticas detalladas aquí"
                  />
                </Card>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  const { summary, trends, topWeapons } = data;

  return (
    <>
      <Header />
      <div className="dashboard-layout">
        <SidebarComponent />
        <div className="main-content">
          <div className="main-scroll">
            <div className="pp-container">
              {/* Header Section */}
              <div className="dashboard-header">
                <div className="header-left">
                  <h1 className="dashboard-title">
                    Desempeño <span className="highlight">Personal</span>
                  </h1>
                  <p className="dashboard-subtitle">
                    Análisis detallado de tu rendimiento en las últimas {data.totalMatches} partidas
                  </p>
                </div>
                <div className="header-actions pp-filters">
                  <select 
                    className="pp-filter-select" 
                    value={selectedFilter}
                    onChange={(e) => setSelectedFilter(e.target.value)}
                  >
                    <option value="all">Todas las fechas</option>
                    <option value="week">Última semana</option>
                    <option value="month">Último mes</option>
                    <option value="season">Temporada actual</option>
                  </select>
                  <select 
                    className="pp-filter-select" 
                    value={selectedSide}
                    onChange={(e) => setSelectedSide(e.target.value)}
                  >
                    <option value="both">Ambos lados</option>
                    <option value="t">Terroristas</option>
                    <option value="ct">Antiterroristas</option>
                  </select>
                </div>
              </div>

              {/* KPIs Grid */}
              <div className="pp-kpis-grid">
                <Card className="pp-kpi-card">
                  <div className="pp-kpi-icon" style={{ background: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)" }}>
                    <Target size={24} />
                  </div>
                  <div className="pp-kpi-content">
                    <span className="pp-kpi-label">K/D Ratio</span>
                    <span className="pp-kpi-value">{summary.kd.value.toFixed(2)}</span>
                    <div className={`pp-kpi-change ${summary.kd.trend}`}>
                      {summary.kd.trend === 'up' ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                      <span>{summary.kd.change > 0 ? '+' : ''}{summary.kd.change.toFixed(2)}</span>
                    </div>
                  </div>
                </Card>

                <Card className="pp-kpi-card">
                  <div className="pp-kpi-icon" style={{ background: "linear-gradient(135deg, #f093fb 0%, #f5576c 100%)" }}>
                    <Activity size={24} />
                  </div>
                  <div className="pp-kpi-content">
                    <span className="pp-kpi-label">ADR</span>
                    <span className="pp-kpi-value">{summary.adr.value.toFixed(1)}</span>
                    <div className={`pp-kpi-change ${summary.adr.trend}`}>
                      {summary.adr.trend === 'up' ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                      <span>{summary.adr.change > 0 ? '+' : ''}{summary.adr.change.toFixed(1)}</span>
                    </div>
                  </div>
                </Card>

                <Card className="pp-kpi-card">
                  <div className="pp-kpi-icon" style={{ background: "linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)" }}>
                    <Crosshair size={24} />
                  </div>
                  <div className="pp-kpi-content">
                    <span className="pp-kpi-label">HS%</span>
                    <span className="pp-kpi-value">{summary.hs.value.toFixed(1)}%</span>
                    <div className={`pp-kpi-change ${summary.hs.trend}`}>
                      {summary.hs.trend === 'up' ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                      <span>{summary.hs.change > 0 ? '+' : ''}{summary.hs.change.toFixed(1)}%</span>
                    </div>
                  </div>
                </Card>

                <Card className="pp-kpi-card">
                  <div className="pp-kpi-icon" style={{ background: "linear-gradient(135deg, #fa709a 0%, #fee140 100%)" }}>
                    <Zap size={24} />
                  </div>
                  <div className="pp-kpi-content">
                    <span className="pp-kpi-label">Impact</span>
                    <span className="pp-kpi-value">{summary.impact.value.toFixed(2)}</span>
                    <div className={`pp-kpi-change ${summary.impact.trend}`}>
                      {summary.impact.trend === 'up' ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                      <span>{summary.impact.change > 0 ? '+' : ''}{summary.impact.change.toFixed(2)}</span>
                    </div>
                  </div>
                </Card>

                <Card className="pp-kpi-card">
                  <div className="pp-kpi-icon" style={{ background: "linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)" }}>
                    <Award size={24} />
                  </div>
                  <div className="pp-kpi-content">
                    <span className="pp-kpi-label">Win Rate</span>
                    <span className="pp-kpi-value">{summary.winRate.value.toFixed(1)}%</span>
                    <div className={`pp-kpi-change ${summary.winRate.trend}`}>
                      {summary.winRate.trend === 'up' ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                      <span>{summary.winRate.change > 0 ? '+' : ''}{summary.winRate.change.toFixed(1)}%</span>
                    </div>
                  </div>
                </Card>

                <Card className="pp-kpi-card">
                  <div className="pp-kpi-icon" style={{ background: "linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%)" }}>
                    <Zap size={24} />
                  </div>
                  <div className="pp-kpi-content">
                    <span className="pp-kpi-label">Clutches</span>
                    <span className="pp-kpi-value">{summary.clutches.value}</span>
                    <div className={`pp-kpi-change ${summary.clutches.trend}`}>
                      {summary.clutches.trend === 'up' ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                      <span>{summary.clutches.change > 0 ? '+' : ''}{summary.clutches.change}</span>
                    </div>
                  </div>
                </Card>
              </div>

              {/* Trends y Weapons */}
              <div className="pp-content-grid">
                {/* Tendencias */}
                <Card className="pp-trends-card">
                  <h3 className="pp-card-title">Tendencias recientes</h3>
                  <p className="pp-card-subtitle">Evolución de métricas clave</p>
                  <div className="pp-trends-chart">
                    {trends.map((item, idx) => (
                      <div key={idx} className="pp-trend-bar">
                        <span className="pp-trend-label">{item.period}</span>
                        <div className="pp-trend-bars">
                          <div className="pp-bar-wrapper">
                            <div 
                              className="pp-bar pp-bar-kd" 
                              style={{ height: `${(item.kd / 2) * 100}%` }}
                              title={`K/D: ${item.kd.toFixed(2)}`}
                            />
                          </div>
                          <div className="pp-bar-wrapper">
                            <div 
                              className="pp-bar pp-bar-adr" 
                              style={{ height: `${(item.adr / 100) * 100}%` }}
                              title={`ADR: ${item.adr}`}
                            />
                          </div>
                          <div className="pp-bar-wrapper">
                            <div 
                              className="pp-bar pp-bar-hs" 
                              style={{ height: `${item.hs}%` }}
                              title={`HS%: ${item.hs}`}
                            />
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                  <div className="pp-trends-legend">
                    <span className="pp-legend-item">
                      <span className="pp-legend-dot pp-legend-kd"></span>
                      K/D
                    </span>
                    <span className="pp-legend-item">
                      <span className="pp-legend-dot pp-legend-adr"></span>
                      ADR
                    </span>
                    <span className="pp-legend-item">
                      <span className="pp-legend-dot pp-legend-hs"></span>
                      HS%
                    </span>
                  </div>
                </Card>

                {/* Top Weapons */}
                <Card className="pp-weapons-card">
                  <h3 className="pp-card-title">Top 5 armas</h3>
                  <p className="pp-card-subtitle">Rendimiento por arma</p>
                  {topWeapons && topWeapons.length > 0 ? (
                    <div className="pp-weapons-table">
                      <div className="pp-weapons-header">
                        <span>Arma</span>
                        <span>Precisión</span>
                        <span>HS%</span>
                        <span>Daño</span>
                        <span>Kills</span>
                      </div>
                      {topWeapons.map((weapon, idx) => (
                        <div key={idx} className="pp-weapon-row">
                          <div className="pp-weapon-name">
                            <Crosshair size={16} className="pp-weapon-icon" />
                            <span>{weapon.weapon}</span>
                          </div>
                          <span className="pp-weapon-stat">{weapon.accuracy}%</span>
                          <span className="pp-weapon-stat pp-stat-highlight">{weapon.hs}%</span>
                          <span className="pp-weapon-stat">{weapon.avgDamage}</span>
                          <span className="pp-weapon-stat pp-stat-bold">{weapon.kills}</span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <EmptyState 
                      icon={Crosshair}
                      title="Sin datos de armas"
                      description="Las estadísticas de armas aparecerán aquí"
                    />
                  )}
                </Card>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
