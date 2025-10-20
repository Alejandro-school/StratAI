// frontend/src/components/Stats/Progress.jsx
import React, { useState } from "react";
import SectionHeader from "./common/SectionHeader";
import Card from "./common/Card";
import EmptyState from "./common/EmptyState";
import ErrorState from "./common/ErrorState";
import { TrendingUp, TrendingDown, Award, MapPin, Calendar } from "lucide-react";
import "../../styles/Stats/progress.css";

/**
 * MOCK DATA - Ejemplo de estructura de datos esperada
 */
const MOCK_DATA = {
  series: {
    kd: { current: 1.25, previous: 1.18, change: 5.9, trend: "up" },
    adr: { current: 82.4, previous: 78.2, change: 5.4, trend: "up" },
    hs: { current: 48.5, previous: 49.2, change: -1.4, trend: "down" },
    impact: { current: 1.18, previous: 1.12, change: 5.4, trend: "up" }
  },
  milestones: [
    { 
      date: "2025-10-10", 
      title: "Primera ACE", 
      description: "5K en Mirage contra equipo completo",
      type: "achievement"
    },
    { 
      date: "2025-10-05", 
      title: "K/D > 1.20", 
      description: "Alcanzaste un K/D promedio superior a 1.20",
      type: "stat"
    },
    { 
      date: "2025-09-28", 
      title: "Win rate 60%", 
      description: "Lograste un 60% de victorias en Inferno",
      type: "milestone"
    },
    { 
      date: "2025-09-20", 
      title: "Clutch 1v3", 
      description: "Ganaste tu primer clutch 1v3 en Dust2",
      type: "achievement"
    }
  ],
  recent: [
    { 
      date: "2025-10-14", 
      map: "Mirage", 
      result: "Victoria", 
      score: "16-12",
      adr: 88.5, 
      kd: 1.42, 
      kills: 24, 
      deaths: 17 
    },
    { 
      date: "2025-10-13", 
      map: "Inferno", 
      result: "Derrota", 
      score: "13-16",
      adr: 76.2, 
      kd: 0.95, 
      kills: 18, 
      deaths: 19 
    },
    { 
      date: "2025-10-12", 
      map: "Dust2", 
      result: "Victoria", 
      score: "16-14",
      adr: 85.3, 
      kd: 1.35, 
      kills: 27, 
      deaths: 20 
    },
    { 
      date: "2025-10-11", 
      map: "Nuke", 
      result: "Victoria", 
      score: "16-10",
      adr: 79.8, 
      kd: 1.18, 
      kills: 21, 
      deaths: 18 
    },
    { 
      date: "2025-10-10", 
      map: "Overpass", 
      result: "Derrota", 
      score: "12-16",
      adr: 72.4, 
      kd: 0.88, 
      kills: 15, 
      deaths: 17 
    }
  ]
};

/**
 * Progress - Componente de evolución temporal del rendimiento
 * 
 * @param {Object} data - Datos de progreso { series, milestones, recent }
 * @param {boolean} loading - Estado de carga
 * @param {string} error - Mensaje de error si existe
 * @param {Function} onRetry - Función para reintentar carga
 */
export default function Progress({ data, loading = false, error = null, onRetry }) {
  const [selectedMetric, setSelectedMetric] = useState("kd");
  
  // Usar mock data si no se proveen datos
  const statsData = data || MOCK_DATA;

  // Estado de carga
  if (loading) {
    return (
      <div className="pr-container">
        <SectionHeader 
          title="Progreso" 
          description="Evolución de tus métricas clave"
        />
        <Card>
          <div className="pr-loading">
            <div className="pr-spinner"></div>
            <p>Cargando progreso...</p>
          </div>
        </Card>
      </div>
    );
  }

  // Estado de error
  if (error) {
    return (
      <div className="pr-container">
        <SectionHeader 
          title="Progreso" 
          description="Evolución de tus métricas clave"
        />
        <Card>
          <ErrorState message={error} onRetry={onRetry} />
        </Card>
      </div>
    );
  }

  // Estado vacío
  if (!statsData.series) {
    return (
      <div className="pr-container">
        <SectionHeader 
          title="Progreso" 
          description="Evolución de tus métricas clave"
        />
        <Card>
          <EmptyState 
            icon={TrendingUp}
            title="No hay datos de progreso"
            description="Juega más partidas para ver tu evolución"
          />
        </Card>
      </div>
    );
  }

  const { series, milestones, recent } = statsData;

  const metrics = {
    kd: { label: "K/D Ratio", icon: TrendingUp, color: "#667eea" },
    adr: { label: "ADR", icon: TrendingUp, color: "#f093fb" },
    hs: { label: "HS%", icon: TrendingUp, color: "#4facfe" },
    impact: { label: "Impact", icon: TrendingUp, color: "#fa709a" }
  };

  const getMilestoneIcon = (type) => {
    switch (type) {
      case "achievement": return "🏆";
      case "stat": return "📊";
      case "milestone": return "🎯";
      default: return "✨";
    }
  };

  const getMilestoneColor = (type) => {
    switch (type) {
      case "achievement": return "linear-gradient(135deg, #fbbf24, #f59e0b)";
      case "stat": return "linear-gradient(135deg, #8b5cf6, #7c3aed)";
      case "milestone": return "linear-gradient(135deg, #10b981, #059669)";
      default: return "linear-gradient(135deg, #94a3b8, #64748b)";
    }
  };

  return (
    <div className="pr-container">
      <SectionHeader 
        title="Progreso" 
        description="Evolución de tus métricas clave"
        actions={
          <select 
            className="pr-metric-select" 
            value={selectedMetric}
            onChange={(e) => setSelectedMetric(e.target.value)}
          >
            <option value="kd">K/D Ratio</option>
            <option value="adr">ADR</option>
            <option value="hs">HS%</option>
            <option value="impact">Impact</option>
          </select>
        }
      />

      {/* Metric Cards Grid */}
      <div className="pr-metrics-grid">
        {Object.entries(series).map(([key, metric]) => {
          const Icon = metrics[key]?.icon || TrendingUp;
          const isPositive = metric.change > 0;
          
          return (
            <Card key={key} className="pr-metric-card" hoverable>
              <div className="pr-metric-header">
                <span className="pr-metric-label">{metrics[key]?.label}</span>
                <div className="pr-metric-icon" style={{ background: metrics[key]?.color }}>
                  <Icon size={20} />
                </div>
              </div>
              <div className="pr-metric-value">{metric.current.toFixed(2)}</div>
              <div className="pr-metric-footer">
                <span className="pr-metric-previous">Anterior: {metric.previous.toFixed(2)}</span>
                <div className={`pr-metric-change ${isPositive ? 'pr-change-up' : 'pr-change-down'}`}>
                  {isPositive ? <TrendingUp size={14} /> : <TrendingDown size={14} />}
                  <span>{isPositive ? '+' : ''}{metric.change.toFixed(1)}%</span>
                </div>
              </div>
            </Card>
          );
        })}
      </div>

      {/* Main Content Grid */}
      <div className="pr-content-grid">
        {/* Timeline / Milestones */}
        <Card className="pr-timeline-card">
          <h3 className="pr-card-title">Hitos y logros</h3>
          <p className="pr-card-subtitle">Tu progreso a lo largo del tiempo</p>
          
          <div className="pr-timeline">
            {milestones && milestones.length > 0 ? (
              milestones.map((milestone, idx) => (
                <div key={idx} className="pr-milestone-item">
                  <div 
                    className="pr-milestone-icon" 
                    style={{ background: getMilestoneColor(milestone.type) }}
                  >
                    <span className="pr-milestone-emoji">{getMilestoneIcon(milestone.type)}</span>
                  </div>
                  <div className="pr-milestone-content">
                    <div className="pr-milestone-date">
                      <Calendar size={14} />
                      <span>{milestone.date}</span>
                    </div>
                    <div className="pr-milestone-title">{milestone.title}</div>
                    <div className="pr-milestone-description">{milestone.description}</div>
                  </div>
                </div>
              ))
            ) : (
              <div className="pr-timeline-empty">
                <Award size={32} opacity={0.3} />
                <p>No hay hitos registrados aún</p>
              </div>
            )}
          </div>
        </Card>

        {/* Recent Matches */}
        <Card className="pr-recent-card">
          <h3 className="pr-card-title">Últimas partidas</h3>
          <p className="pr-card-subtitle">Rendimiento reciente</p>
          
          <div className="pr-matches-list">
            {recent && recent.length > 0 ? (
              recent.map((match, idx) => (
                <div key={idx} className="pr-match-item">
                  <div className={`pr-match-result ${match.result === 'Victoria' ? 'pr-result-win' : 'pr-result-loss'}`}>
                    {match.result === 'Victoria' ? 'V' : 'D'}
                  </div>
                  <div className="pr-match-content">
                    <div className="pr-match-header">
                      <div className="pr-match-map">
                        <MapPin size={14} />
                        <span>{match.map}</span>
                      </div>
                      <span className="pr-match-score">{match.score}</span>
                    </div>
                    <div className="pr-match-stats">
                      <span className="pr-match-stat">
                        <strong>{match.kills}</strong> / {match.deaths}
                      </span>
                      <span className="pr-match-stat">
                        K/D: <strong>{match.kd.toFixed(2)}</strong>
                      </span>
                      <span className="pr-match-stat">
                        ADR: <strong>{match.adr.toFixed(1)}</strong>
                      </span>
                    </div>
                    <div className="pr-match-date">{match.date}</div>
                  </div>
                </div>
              ))
            ) : (
              <EmptyState 
                icon={Calendar}
                title="No hay partidas recientes"
                description="Las partidas aparecerán aquí"
              />
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}

/**
 * EJEMPLO DE USO EN ROUTER:
 * 
 * import Progress from './components/Stats/Progress';
 * 
 * const progressData = {
 *   series: { kd: {...}, adr: {...}, ... },
 *   milestones: [...],
 *   recent: [...]
 * };
 * 
 * <Progress 
 *   data={progressData} 
 *   loading={false}
 *   error={null}
 *   onRetry={() => fetchData()}
 * />
 */




