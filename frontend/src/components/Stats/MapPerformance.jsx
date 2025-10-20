// frontend/src/components/Stats/MapPerformance.jsx
import React, { useState } from "react";
import Header from "../Layout/Header";
import SidebarComponent from "../Layout/Sidebar";
import Card from "./common/Card";
import EmptyState from "./common/EmptyState";
import ErrorState from "./common/ErrorState";
import { MapPin, AlertTriangle } from "lucide-react";
import { useUser } from "../../context/UserContext";
import { useMapPerformance } from "../../hooks/useMapPerformance";
import "../../styles/Stats/mapPerformance.css";

export default function MapPerformance() {
  const { user } = useUser();
  const { data, loading, error } = useMapPerformance(user);
  const [selectedMetric, setSelectedMetric] = useState("winRate");

  // Estado de carga
  if (loading) {
    return (
      <>
        <Header />
        <div className="dashboard-layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="main-scroll">
              <div className="mp-container">
                <div className="mp-loading">
                  <div className="mp-spinner"></div>
                  <p>Cargando datos de mapas...</p>
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
              <div className="mp-container">
                <Card>
                  <ErrorState message={error} onRetry={() => window.location.reload()} />
                </Card>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  // Estado vacío
  if (!data || !data.maps || data.maps.length === 0) {
    return (
      <>
        <Header />
        <div className="dashboard-layout">
          <SidebarComponent />
          <div className="main-content">
            <div className="main-scroll">
              <div className="mp-container">
                <Card>
                  <EmptyState 
                    icon={MapPin}
                    title="No hay datos de mapas"
                    description="Juega partidas en diferentes mapas para ver tu rendimiento"
                  />
                </Card>
              </div>
            </div>
          </div>
        </div>
      </>
    );
  }

  const { maps, hotspots } = data;

  // Obtener valores para el gradiente de color
  const metricValues = maps.map(m => parseFloat(m[selectedMetric]));
  const maxValue = Math.max(...metricValues);
  const minValue = Math.min(...metricValues);

  const getMetricColor = (value) => {
    const normalized = (parseFloat(value) - minValue) / (maxValue - minValue || 1);
    if (normalized >= 0.7) return "linear-gradient(135deg, #10b981 0%, #34d399 100%)";
    if (normalized >= 0.4) return "linear-gradient(135deg, #f59e0b 0%, #fbbf24 100%)";
    return "linear-gradient(135deg, #ef4444 0%, #f87171 100%)";
  };

  const getRiskColor = (risk) => {
    if (risk === "high") return "#ef4444";
    if (risk === "medium") return "#f59e0b";
    return "#10b981";
  };

  const metricLabels = {
    winRate: "Win Rate",
    adr: "ADR",
    kd: "K/D",
    entrySuccess: "Entry Success",
    clutchRate: "Clutch Rate"
  };

  return (
    <>
      <Header />
      <div className="dashboard-layout">
        <SidebarComponent />
        <div className="main-content">
          <div className="main-scroll">
            <div className="mp-container">
              {/* Header */}
              <div className="dashboard-header">
                <div className="header-left">
                  <h1 className="dashboard-title">
                    Rendimiento por <span className="highlight">Mapa</span>
                  </h1>
                  <p className="dashboard-subtitle">
                    Análisis detallado de tu desempeño en cada mapa
                  </p>
                </div>
                <div className="header-actions">
                  <select 
                    className="mp-metric-select" 
                    value={selectedMetric}
                    onChange={(e) => setSelectedMetric(e.target.value)}
                  >
                    <option value="winRate">Win Rate</option>
                    <option value="adr">ADR</option>
                    <option value="kd">K/D Ratio</option>
                    <option value="entrySuccess">Entry Success</option>
                    <option value="clutchRate">Clutch Rate</option>
                  </select>
                </div>
              </div>

              {/* Maps Grid */}
              <div className="mp-maps-grid">
                {maps.map((map, idx) => (
                  <Card key={idx} className="mp-map-card" hoverable>
                    <div className="mp-map-header">
                      <MapPin size={20} className="mp-map-icon" />
                      <h3 className="mp-map-name">{map.name}</h3>
                      <span className="mp-map-matches">{map.matches} partidas</span>
                    </div>
                    
                    <div className="mp-map-metric-highlight">
                      <span className="mp-metric-label">{metricLabels[selectedMetric]}</span>
                      <span className="mp-metric-value">
                        {selectedMetric === 'kd' 
                          ? parseFloat(map[selectedMetric]).toFixed(2) 
                          : `${map[selectedMetric]}${selectedMetric === 'adr' ? '' : '%'}`}
                      </span>
                      <div className="mp-metric-bar">
                        <div 
                          className="mp-metric-bar-fill" 
                          style={{ 
                            width: `${((parseFloat(map[selectedMetric]) - minValue) / (maxValue - minValue || 1)) * 100}%`,
                            background: getMetricColor(map[selectedMetric])
                          }}
                        />
                      </div>
                    </div>

                    <div className="mp-map-stats">
                      <div className="mp-stat-item">
                        <span className="mp-stat-label">Win Rate</span>
                        <span className="mp-stat-value">{map.winRate}%</span>
                      </div>
                      <div className="mp-stat-item">
                        <span className="mp-stat-label">ADR</span>
                        <span className="mp-stat-value">{map.adr}</span>
                      </div>
                      <div className="mp-stat-item">
                        <span className="mp-stat-label">K/D</span>
                        <span className="mp-stat-value">{map.kd}</span>
                      </div>
                      <div className="mp-stat-item">
                        <span className="mp-stat-label">Entry</span>
                        <span className="mp-stat-value">{map.entrySuccess}%</span>
                      </div>
                      <div className="mp-stat-item">
                        <span className="mp-stat-label">Clutch</span>
                        <span className="mp-stat-value">{map.clutchRate}%</span>
                      </div>
                    </div>
                  </Card>
                ))}
              </div>

              {/* Hotspots */}
              {hotspots && hotspots.length > 0 && (
                <Card className="mp-hotspots-card">
                  <div className="mp-hotspots-header">
                    <div>
                      <h3 className="mp-card-title">Zonas problemáticas</h3>
                      <p className="mp-card-subtitle">Mapas donde necesitas mejorar</p>
                    </div>
                    <div className="mp-risk-legend">
                      <span className="mp-legend-item">
                        <span className="mp-legend-dot" style={{ background: "#ef4444" }}></span>
                        Alta
                      </span>
                      <span className="mp-legend-item">
                        <span className="mp-legend-dot" style={{ background: "#f59e0b" }}></span>
                        Media
                      </span>
                      <span className="mp-legend-item">
                        <span className="mp-legend-dot" style={{ background: "#10b981" }}></span>
                        Baja
                      </span>
                    </div>
                  </div>
                  
                  <div className="mp-hotspots-list">
                    {hotspots.map((hotspot, idx) => (
                      <div key={idx} className="mp-hotspot-item">
                        <div 
                          className="mp-hotspot-risk" 
                          style={{ background: getRiskColor(hotspot.risk) }}
                        >
                          <AlertTriangle size={20} />
                        </div>
                        <div className="mp-hotspot-content">
                          <div className="mp-hotspot-header-row">
                            <span className="mp-hotspot-map">{hotspot.map}</span>
                            <span className="mp-hotspot-deaths">{hotspot.deaths} problemas</span>
                          </div>
                          <div className="mp-hotspot-position">{hotspot.position}</div>
                          <div className="mp-hotspot-description">{hotspot.description}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
