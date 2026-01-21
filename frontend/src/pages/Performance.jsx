// frontend/src/pages/Performance.jsx
import React, { useState } from 'react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useUser } from '../context/UserContext';
import { usePerformanceData } from '../hooks/usePerformanceData';
import { 
  Target, Crosshair, Award, TrendingUp, 
  Map as MapIcon, Sword, Users, ChevronRight,
  Flame, Zap, Shield, AlertTriangle, CheckCircle, Clock
} from 'lucide-react';
import '../styles/pages/performance.css';

// --- SUB-COMPONENTS ---

const StatCard = ({ label, value, subtext, icon: Icon, trend, color = "blue" }) => (
  <div className={`stat-card ${color}`}>
    <div className="stat-icon-wrapper">
      <Icon size={24} />
    </div>
    <div className="stat-content">
      <span className="stat-value">{value}</span>
      <span className="stat-label">{label}</span>
      {subtext && <span className="stat-subtext">{subtext}</span>}
      {trend && (
        <div className={`stat-trend ${trend > 0 ? 'positive' : 'negative'}`}>
          <TrendingUp size={14} />
          <span>{trend > 0 ? '+' : ''}{trend}%</span>
        </div>
      )}
    </div>
  </div>
);

const AimSection = ({ stats }) => {
  if (!stats) return <div className='no-data'>No aim data available</div>;

  const { body_part_hits, accuracy_overall, time_to_damage_avg_ms, crosshair_placement_avg_error } = stats;
  
  // Calculate total hits for percentages
  const totalHits = Object.values(body_part_hits).reduce((a, b) => a + b, 0);
  const getPct = (part) => totalHits > 0 ? Math.round(((body_part_hits[part] || 0) / totalHits) * 100) : 0;

  return (
    <div className="aim-dashboard-grid">
      <div className="aim-metrics-row">
        <StatCard label="Accuracy" value={`${accuracy_overall}%`} icon={Target} color="purple" />
        <StatCard label="Time to Damage" value={`${time_to_damage_avg_ms}ms`} icon={Zap} color="yellow" />
        <StatCard label="Crosshair Error" value={`${crosshair_placement_avg_error}°`} icon={Crosshair} color="red" />
      </div>

      <div className="aim-visualization-section">
        <h3>Distribución de Impactos</h3>
        <div className="body-bars-container">
          {[
            { key: 'head', label: 'Cabeza', icon: '🧠' },
            { key: 'chest', label: 'Pecho', icon: '👕' },
            { key: 'stomach', label: 'Estómago', icon: '🎽' },
            { key: 'left_arm', label: 'Brazos', icon: '💪' }, // Combined for simplicity
            { key: 'right_leg', label: 'Piernas', icon: '🦵' } // Combined
          ].map((part) => {
            // Combine arms/legs for simplified view if needed
            let count = body_part_hits[part.key] || 0;
            if (part.key === 'left_arm') count += body_part_hits['right_arm'] || 0;
            if (part.key === 'right_leg') count += body_part_hits['left_leg'] || 0;
            
            const pct = totalHits > 0 ? Math.round((count / totalHits) * 100) : 0;
            
            return (
              <div key={part.key} className="body-bar-row">
                 <span className="body-icon">{part.icon}</span>
                 <span className="body-label">{part.label}</span>
                 <div className="progress-track">
                   <div className="progress-fill" style={{ width: `${pct}%` }}></div>
                 </div>
                 <span className="body-pct">{pct}%</span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};

const WeaponsTable = ({ weapons }) => (
  <div className="data-table-container">
    <table className="performance-table">
      <thead>
        <tr>
          <th>Arma</th>
          <th>Kills</th>
          <th>Precisión</th>
          <th>HS %</th>
          <th>Daño Total</th>
        </tr>
      </thead>
      <tbody>
        {weapons.map((w, i) => (
          <tr key={i}>
            <td className="fw-bold">{w.weapon}</td>
            <td>{w.kills}</td>
            <td>
              <div className="mini-bar-cell">
                <span>{w.accuracy}%</span>
                <div className="mini-bar" style={{ width: `${Math.min(w.accuracy, 100)}%` }}></div>
              </div>
            </td>
            <td>{w.hs_pct}%</td>
            <td>{w.damage?.toLocaleString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const MapsTable = ({ maps }) => (
  <div className="data-table-container">
    <table className="performance-table">
      <thead>
        <tr>
          <th>Mapa</th>
          <th>Partidas</th>
          <th>Wins</th>
          <th>Losses</th>
          <th>Win Rate</th>
        </tr>
      </thead>
      <tbody>
        {maps.map((m, i) => (
          <tr key={i}>
            <td className="text-capitalize">{m.map}</td>
            <td>{m.wins + m.losses}</td>
            <td className="text-green">{m.wins}</td>
            <td className="text-red">{m.losses}</td>
            <td>
              <span className={`badge ${m.win_rate >= 50 ? 'badge-good' : 'badge-bad'}`}>
                {m.win_rate}%
              </span>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const GrenadesSection = ({ data }) => {
  if (!data?.summary) return <div>No grenade data</div>;
  const { summary, insights } = data;

  return (
    <div className="grenades-grid">
      <div className="grenade-summary-cards">
        <div className="grenade-card flash">
          <Zap size={20} />
          <h4>Flashbangs</h4>
          <div className="g-stat">
            <span>Thrown</span> <strong>{summary.flash.thrown}</strong>
          </div>
          <div className="g-stat">
            <span>Avg Blind</span> <strong>{summary.flash.avg_blinded}</strong>
          </div>
        </div>
        <div className="grenade-card he">
          <Flame size={20} />
          <h4>HE Grenades</h4>
          <div className="g-stat">
            <span>Thrown</span> <strong>{summary.he.thrown}</strong>
          </div>
          <div className="g-stat">
            <span>Avg Dmg</span> <strong>{summary.he.avg_damage}</strong>
          </div>
        </div>
        <div className="grenade-card molotov">
          <Flame size={20} color="orange" />
          <h4>Molotovs</h4>
          <div className="g-stat">
            <span>Thrown</span> <strong>{summary.molotov.thrown}</strong>
          </div>
          <div className="g-stat">
            <span>Avg Dmg</span> <strong>{summary.molotov.avg_damage}</strong>
          </div>
        </div>
      </div>

      <div className="grenade-insights">
        <h3>Insights de Utilidad</h3>
        {insights && insights.length > 0 ? (
          <div className="insights-list">
            {insights.map((insight, i) => (
              <div key={i} className={`insight-item ${insight.type}`}>
                {insight.type === 'warning' ? <AlertTriangle size={16} /> : <CheckCircle size={16} />}
                <p>{insight.text}</p>
              </div>
            ))}
          </div>
        ) : (
          <p className="no-insights">Sigue jugando para generar insights sobre tus granadas.</p>
        )}
      </div>
    </div>
  );
};

// --- MAIN COMPONENT ---

const Performance = () => {
  const { user } = useUser();
  const { overview, grenades, loading, error } = usePerformanceData(user);
  const [activeTab, setActiveTab] = useState('overview');

  if (loading) return (
    <NavigationFrame>
      <div className="loading-container">
        <div className="spinner"></div>
        <p>Analizando rendimiento combate a combate...</p>
      </div>
    </NavigationFrame>
  );

  if (error || !overview) return (
    <NavigationFrame>
      <div className="error-container">
        <AlertTriangle size={48} />
        <h2>Error cargando datos</h2>
        <p>{error || "No se encontraron datos de rendimiento."}</p>
      </div>
    </NavigationFrame>
  );

  const { stats, aim_stats, weapon_stats, map_stats } = overview;

  return (
    <NavigationFrame>
      <div className="performance-container">
        {/* Header */}
        <header className="performance-header">
          <div className="header-content">
            <h1><Target className="header-icon" /> Performance Hub</h1>
            <p className="header-subtitle">Análisis profundo de tus mecánicas y decisiones tácticas</p>
          </div>
          
          <nav className="performance-tabs">
            {[
              { id: 'overview', label: 'General', icon: Target },
              { id: 'aim', label: 'Aim', icon: Crosshair },
              { id: 'maps', label: 'Mapas', icon: MapIcon },
              { id: 'weapons', label: 'Armas', icon: Sword },
              { id: 'grenades', label: 'Granadas', icon: Flame },
            ].map(tab => (
              <button
                key={tab.id}
                className={`perf-tab ${activeTab === tab.id ? 'active' : ''}`}
                onClick={() => setActiveTab(tab.id)}
              >
                <tab.icon size={16} />
                <span>{tab.label}</span>
              </button>
            ))}
          </nav>
        </header>

        {/* Dynamic Content */}
        <main className="performance-content">
          
          {/* OVERVIEW TAB */}
          {activeTab === 'overview' && (
            <div className="fade-in">
              <section className="stats-grid">
                <StatCard label="K/D Ratio" value={stats.avg_kd} icon={Sword} color="blue" />
                <StatCard label="ADR" value={stats.avg_adr} icon={Flame} color="orange" />
                <StatCard label="Headshot %" value={`${stats.avg_hs}%`} icon={Crosshair} color="purple" />
                <StatCard label="Win Rate" value={`${stats.win_rate}%`} subtext={`${stats.wins}W - ${stats.losses}L`} icon={Award} color={stats.win_rate >= 50 ? "green" : "red"} />
                <StatCard label="Partidas" value={stats.total_matches} icon={Clock} color="gray" />
              </section>

              <div className="overview-split">
                <div className="widget-card">
                  <div className="widget-header">
                    <h3><Sword size={18} /> Top Armas</h3>
                    <button className="link-btn" onClick={() => setActiveTab('weapons')}>Ver todas</button>
                  </div>
                  <WeaponsTable weapons={weapon_stats.slice(0, 3)} />
                </div>

                <div className="widget-card">
                  <div className="widget-header">
                    <h3><MapIcon size={18} /> Mejores Mapas</h3>
                    <button className="link-btn" onClick={() => setActiveTab('maps')}>Ver todos</button>
                  </div>
                  <MapsTable maps={map_stats.slice(0, 3)} />
                </div>
              </div>
            </div>
          )}

          {/* AIM TAB */}
          {activeTab === 'aim' && (
            <div className="fade-in">
               <AimSection stats={aim_stats} />
            </div>
          )}

          {/* WEAPONS TAB */}
          {activeTab === 'weapons' && (
            <div className="fade-in card-padded">
              <WeaponsTable weapons={weapon_stats} />
            </div>
          )}

           {/* MAPS TAB */}
           {activeTab === 'maps' && (
            <div className="fade-in card-padded">
              <MapsTable maps={map_stats} />
            </div>
          )}

          {/* GRENADES TAB */}
          {activeTab === 'grenades' && (
            <div className="fade-in">
              <GrenadesSection data={grenades} />
            </div>
          )}

        </main>
      </div>
    </NavigationFrame>
  );
};

export default Performance;
