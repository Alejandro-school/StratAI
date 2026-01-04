// frontend/src/components/Dashboard/DashboardMap.jsx
// Map-Based Dashboard - Interactive Zone Performance Analysis
import React, { useState, useMemo } from 'react';
import { Link } from 'react-router-dom';
import NavigationFrame from '../Layout/NavigationFrame';
import { useUser } from '../../context/UserContext';
import { useDashboardData } from '../../hooks/useDashboardData';
import { useMapZoneStats } from '../../hooks/useMapZoneStats';
import { 
  Target, Shield, TrendingUp, TrendingDown,
  MapPin, ChevronDown, Eye, AlertTriangle, Check
} from 'lucide-react';
import '../../styles/Dashboard/dashboardMap.css';

// Available maps configuration  
// img: filename suffix for radar image (stored in /public/maps/)
const AVAILABLE_MAPS = [
  { id: 'de_dust2', name: 'Dust II', img: 'de_dust2_radar_psd.png', available: true },
  { id: 'de_mirage', name: 'Mirage', img: 'de_mirage_radar_psd.png', available: true },
  { id: 'de_inferno', name: 'Inferno', img: 'de_inferno_radar_psd.png', available: true },
  { id: 'de_nuke', name: 'Nuke', img: 'de_nuke_radar_psd.png', available: true },
  { id: 'de_overpass', name: 'Overpass', img: 'de_overpass_radar_psd.png', available: true },
  { id: 'de_train', name: 'Train', img: 'de_train_radar_psd.png', available: true },
  { id: 'de_vertigo', name: 'Vertigo', img: 'de_vertigo_radar_psd.png', available: true },
  { id: 'de_anubis', name: 'Anubis', img: 'de_anubis_radar_psd.png', available: true },
  { id: 'de_ancient', name: 'Ancient', img: 'de_ancient_radar_psd.png', available: true },
];

// Zone configuration for Dust 2 (positions are % from top-left)
// Adjusted based on visual reference - temporary until we have game coordinate mapping
const DUST2_ZONES = [
  {
    id: 'site-a',
    label: 'A Site',
    shortLabel: 'A',
    x: 80,
    y: 18,
    color: '#ef4444',
    description: 'Bombsite A - Long A, Short A, CT Spawn access'
  },
  {
    id: 'site-b',
    label: 'B Site',
    shortLabel: 'B',
    x: 12,
    y: 18,
    color: '#22c55e',
    description: 'Bombsite B - Tunnels, Window, CT Mid access'
  },
  {
    id: 'mid',
    label: 'Mid',
    shortLabel: 'MID',
    x: 46,
    y: 55,
    color: '#f59e0b',
    description: 'Middle area - AWP battles, rotations, map control'
  },
  {
    id: 'long-a',
    label: 'Long A',
    shortLabel: 'LONG',
    x: 78,
    y: 58,
    color: '#8b5cf6',
    description: 'Long A corridor - Entry duels, AWP fights'
  },
  {
    id: 'b-tunnels',
    label: 'B Tunnels',
    shortLabel: 'Tunnel',
    x: 17,
    y: 50,
    color: '#06b6d4',
    description: 'Lower tunnels - B rushes, lurks'
  },
  {
    id: 'catwalk',
    label: 'Catwalk',
    shortLabel: 'Short',
    x: 62,
    y: 38,
    color: '#ec4899',
    description: 'Short A / Catwalk - Quick A access'
  },
  {
    id: 't-spawn',
    label: 'T Spawn',
    shortLabel: 'T',
    x: 45,
    y: 94,
    color: '#f97316',
    description: 'Terrorist spawn - Round start, eco saves'
  },
  {
    id: 'ct-spawn',
    label: 'CT Spawn',
    shortLabel: 'CT',
    x: 68,
    y: 27,
    color: '#3b82f6',
    description: 'Counter-Terrorist spawn - Rotations, retakes'
  }
];

// Default fallback stats when no data available
const DEFAULT_ZONE_STATS = {
  kills: 0, deaths: 0, duels_won: 0, duels_total: 0, 
  win_rate: 50, rating: 'neutral'
};

// Convert API response to UI format
const formatZoneStats = (apiStats) => {
  if (!apiStats) return DEFAULT_ZONE_STATS;
  return {
    kills: apiStats.kills || 0,
    deaths: apiStats.deaths || 0,
    winRate: apiStats.winRate || 50,
    rating: apiStats.rating || 'neutral',
    ctStats: apiStats.ct_stats || { kills: 0, deaths: 0, win_rate: 50 },
    tStats: apiStats.t_stats || { kills: 0, deaths: 0, win_rate: 50 },
    duelsWon: apiStats.duels_won || 0,
    duelsTotal: apiStats.duels_total || 0
  };
};

// Zone Hotspot Component
const ZoneHotspot = ({ zone, stats, isSelected, isWorst, isBest, onClick }) => {
  const ratingClass = stats.rating === 'good' ? 'good' : stats.rating === 'bad' ? 'bad' : 'neutral';
  
  return (
    <button
      className={`zone-hotspot ${ratingClass} ${isSelected ? 'selected' : ''} ${isWorst ? 'worst' : ''} ${isBest ? 'best' : ''}`}
      style={{ left: `${zone.x}%`, top: `${zone.y}%`, '--zone-color': zone.color }}
      onClick={() => onClick(zone.id)}
    >
      <div className="hotspot-ring"></div>
      <div className="hotspot-core">
        <span className="hotspot-label">{zone.shortLabel}</span>
      </div>
      <div className="hotspot-stats">
        <span className={`stat-wr ${ratingClass}`}>{stats.winRate}%</span>
      </div>
      {isWorst && (
        <div className="hotspot-badge worst">
          <AlertTriangle size={10} />
        </div>
      )}
      {isBest && (
        <div className="hotspot-badge best">
          <TrendingUp size={10} />
        </div>
      )}
    </button>
  );
};

// Zone Detail Panel
const ZoneDetailPanel = ({ zone, stats, onClose }) => {
  if (!zone) {
    return (
      <div className="zone-panel empty">
        <div className="panel-header">
          <MapPin size={18} />
          <span>Selecciona una zona</span>
        </div>
        <p className="panel-hint">Haz clic en cualquier zona del mapa para ver estadísticas detalladas</p>
      </div>
    );
  }

  const kd = stats.deaths > 0 ? (stats.kills / stats.deaths).toFixed(2) : stats.kills.toFixed(2);
  const ratingClass = stats.rating === 'good' ? 'good' : stats.rating === 'bad' ? 'bad' : 'neutral';

  return (
    <div className={`zone-panel ${ratingClass}`}>
      <div className="panel-header">
        <div className="panel-zone-info">
          <div className="zone-indicator" style={{ background: zone.color }}></div>
          <div>
            <h3>{zone.label}</h3>
            <p>{zone.description}</p>
          </div>
        </div>
      </div>

      <div className="panel-stats-grid">
        <div className="panel-stat">
          <span className="stat-value">{stats.winRate}%</span>
          <span className="stat-label">Win Rate</span>
        </div>
        <div className="panel-stat">
          <span className="stat-value">{kd}</span>
          <span className="stat-label">K/D</span>
        </div>
        <div className="panel-stat">
          <span className="stat-value">{stats.kills}</span>
          <span className="stat-label">Kills</span>
        </div>
        <div className="panel-stat">
          <span className="stat-value">{stats.deaths}</span>
          <span className="stat-label">Deaths</span>
        </div>
      </div>

      <div className="panel-side-split">
        <div className="side-stat ct">
          <div className="side-header">
            <Shield size={12} />
            <span>CT SIDE</span>
          </div>
          <div className="side-values">
            <span className="side-wr">{stats.ctStats.win_rate}% WR</span>
            <span className="side-kd">{stats.ctStats.kills}/{stats.ctStats.deaths} K/D</span>
          </div>
        </div>
        <div className="side-stat t">
          <div className="side-header">
            <Target size={12} />
            <span>T SIDE</span>
          </div>
          <div className="side-values">
            <span className="side-wr">{stats.tStats.win_rate}% WR</span>
            <span className="side-kd">{stats.tStats.kills}/{stats.tStats.deaths} K/D</span>
          </div>
        </div>
      </div>

      <div className="panel-insight">
        {stats.rating === 'good' && (
          <>
            <TrendingUp size={16} className="good" />
            <span>Esta es una de tus <strong>zonas fuertes</strong>. Sigue jugando agresivo aquí.</span>
          </>
        )}
        {stats.rating === 'bad' && (
          <>
            <TrendingDown size={16} className="bad" />
            <span>Esta zona <strong>necesita trabajo</strong>. Revisa tus demos para identificar errores.</span>
          </>
        )}
        {stats.rating === 'neutral' && (
          <>
            <Eye size={16} className="neutral" />
            <span>Tu rendimiento aquí es <strong>consistente</strong>. Oportunidad de mejora.</span>
          </>
        )}
      </div>
    </div>
  );
};

// Map Stats Summary
const MapStatsSummary = ({ dashboardData }) => {
  const stats = dashboardData?.mainStats;
  if (!stats) return null;

  return (
    <div className="map-summary">
      <div className="summary-header">
        <h3>DUST II</h3>
        <span className="summary-matches">12 partidas analizadas</span>
      </div>
      <div className="summary-stats">
        <div className="summary-stat">
          <span className="stat-value">{stats.kd?.value?.toFixed(2) || '0.00'}</span>
          <span className="stat-label">K/D</span>
        </div>
        <div className="summary-stat">
          <span className="stat-value">{Math.round(stats.adr?.value || 0)}</span>
          <span className="stat-label">ADR</span>
        </div>
        <div className="summary-stat">
          <span className="stat-value">{stats.winRate?.value || 0}%</span>
          <span className="stat-label">Win</span>
        </div>
        <div className="summary-stat">
          <span className="stat-value">{stats.headshot?.value || 0}%</span>
          <span className="stat-label">HS</span>
        </div>
      </div>
    </div>
  );
};

// Main Component
const DashboardMap = () => {
  const { user } = useUser();
  const { dashboardData, loading } = useDashboardData(user);
  const [selectedZoneId, setSelectedZoneId] = useState(null);
  const [currentMap, setCurrentMap] = useState('de_dust2');
  const [showMapDropdown, setShowMapDropdown] = useState(false);
  const mapViewRef = React.useRef(null);

  // Parallax effect - move background with mouse
  React.useEffect(() => {
    const handleMouseMove = (e) => {
      if (!mapViewRef.current) return;
      const rect = mapViewRef.current.getBoundingClientRect();
      const x = ((e.clientX - rect.left) / rect.width) * 100;
      const y = ((e.clientY - rect.top) / rect.height) * 100;
      mapViewRef.current.style.setProperty('--mouse-x', `${x}%`);
      mapViewRef.current.style.setProperty('--mouse-y', `${y}%`);
    };

    const mapView = mapViewRef.current;
    if (mapView) {
      mapView.addEventListener('mousemove', handleMouseMove);
      return () => mapView.removeEventListener('mousemove', handleMouseMove);
    }
  }, []);

  // Get current map info
  const currentMapInfo = AVAILABLE_MAPS.find(m => m.id === currentMap) || AVAILABLE_MAPS[0];

  // Fetch real zone stats from API
  const { zoneStats: apiZoneStats, matchesAnalyzed, loading: zoneLoading } = useMapZoneStats(currentMap);

  // Format zone stats for UI
  const zoneStats = useMemo(() => {
    const stats = {};
    DUST2_ZONES.forEach(zone => {
      stats[zone.id] = formatZoneStats(apiZoneStats?.[zone.id]);
    });
    return stats;
  }, [apiZoneStats]);

  // Find best and worst zones
  const { bestZone, worstZone } = useMemo(() => {
    let best = null, worst = null;
    let bestWr = -1, worstWr = 101;
    
    Object.entries(zoneStats).forEach(([id, stats]) => {
      if (stats.winRate > bestWr) { bestWr = stats.winRate; best = id; }
      if (stats.winRate < worstWr) { worstWr = stats.winRate; worst = id; }
    });
    
    return { bestZone: best, worstZone: worst };
  }, [zoneStats]);

  const selectedZone = DUST2_ZONES.find(z => z.id === selectedZoneId);
  const selectedStats = selectedZoneId ? zoneStats[selectedZoneId] : null;

  // Show loading only for initial dashboard load
  if (loading && !dashboardData) {
    return (
      <NavigationFrame>
        <div className="map-dashboard loading">
          <div className="loading-spinner"></div>
          <span>Cargando mapa táctico...</span>
        </div>
      </NavigationFrame>
    );
  }

  return (
    <NavigationFrame>
      <div className="map-dashboard">
        {/* Map Container */}
        <div className="map-main">
          {/* Map Header */}
          <div className="map-header">
            <div className="map-selector-wrapper">
              <button 
                className={`map-selector ${showMapDropdown ? 'open' : ''}`}
                onClick={() => setShowMapDropdown(!showMapDropdown)}
              >
                <MapPin size={18} />
                <span className="map-name">{currentMapInfo.name.toUpperCase()}</span>
                <ChevronDown size={16} className={showMapDropdown ? 'rotated' : ''} />
              </button>

              {showMapDropdown && (
                <div className="map-dropdown">
                  {AVAILABLE_MAPS.map(map => (
                    <button
                      key={map.id}
                      className={`map-option ${currentMap === map.id ? 'active' : ''} ${!map.available ? 'disabled' : ''}`}
                      onClick={() => {
                        if (map.available) {
                          setCurrentMap(map.id);
                          setShowMapDropdown(false);
                          setSelectedZoneId(null);
                        }
                      }}
                      disabled={!map.available}
                    >
                      <span>{map.name}</span>
                      {currentMap === map.id && <Check size={14} />}
                      {!map.available && <span className="coming-soon">Pronto</span>}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Map View */}
          <div className="map-view" ref={mapViewRef}>
            <img 
              src={`/maps/${currentMapInfo.img}`}
              alt={`${currentMapInfo.name} Radar`}
              className="map-image"
            />
            
            {/* Zone Hotspots */}
            <div className="map-hotspots">
              {DUST2_ZONES.map(zone => (
                <ZoneHotspot
                  key={zone.id}
                  zone={zone}
                  stats={zoneStats[zone.id]}
                  isSelected={selectedZoneId === zone.id}
                  isBest={bestZone === zone.id}
                  isWorst={worstZone === zone.id}
                  onClick={setSelectedZoneId}
                />
              ))}
            </div>

            {/* Legend */}
            <div className="map-legend">
              <div className="legend-item good">
                <div className="legend-dot"></div>
                <span>Zona fuerte</span>
              </div>
              <div className="legend-item neutral">
                <div className="legend-dot"></div>
                <span>Neutral</span>
              </div>
              <div className="legend-item bad">
                <div className="legend-dot"></div>
                <span>Zona débil</span>
              </div>
            </div>
          </div>
        </div>

        {/* Side Panel */}
        <div className="map-sidebar">
          {/* Data Status */}
          <div className="data-status">
            {zoneLoading ? (
              <span className="loading-text">Cargando datos...</span>
            ) : (
              <span className="matches-count">
                {matchesAnalyzed} {matchesAnalyzed === 1 ? 'partida' : 'partidas'} analizadas
              </span>
            )}
          </div>

          {/* Map Stats Summary */}
          <MapStatsSummary dashboardData={dashboardData} />

          <ZoneDetailPanel
            zone={selectedZone}
            stats={selectedStats}
            onClose={() => setSelectedZoneId(null)}
          />

          {/* Quick Actions */}
          <div className="sidebar-actions">
            <Link to="/map-performance" className="action-btn">
              <Eye size={16} />
              Ver todas las estadísticas
            </Link>
            <Link to="/history-games" className="action-btn">
              <Target size={16} />
              Partidas en Dust II
            </Link>
          </div>
        </div>
      </div>
    </NavigationFrame>
  );
};

export default DashboardMap;
