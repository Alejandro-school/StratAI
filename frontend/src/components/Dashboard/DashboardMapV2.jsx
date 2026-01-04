// frontend/src/components/Dashboard/DashboardMapV2.jsx
// Map-Based Dashboard V2 - Granular Callout Performance Analysis
// Updated with Adaptive Clustering for dense maps like Nuke
import React, { useState, useMemo } from 'react';
import { Link } from 'react-router-dom';
import NavigationFrame from '../Layout/NavigationFrame';
import { useUser } from '../../context/UserContext';
import { useDashboardData } from '../../hooks/useDashboardData';
import { useCalloutStats } from '../../hooks/useCalloutStats';
import { useGrenadeStats } from '../../hooks/useGrenadeStats';
import { useMovementStats } from '../../hooks/useMovementStats';
import CalloutDetailPanel from './CalloutDetailPanel';
// HeatmapCanvas removed - ugly numbered clusters not suitable for professional UI
import GrenadeMapTab, { GrenadeOverlay } from './GrenadeMapTab';
import MovementHeatmapOverlay from './MovementHeatmapOverlay';
import MovementHeatmapTab from './MovementHeatmapTab';
// New adaptive components
import { AdaptiveHotspotLayer } from './AdaptiveHotspot';
import LevelSelector from './LevelSelector';
import { ZoomableMapContainer } from './MapZoomControls';
import { processCalloutsForDisplay, getMapProfile } from '../../utils/adaptiveClustering';
import { guessCalloutLevel } from '../../utils/mapLevelFallbacks';
import { 
  Target, MapPin, ChevronDown,
  Check, BarChart3, Flame, Bomb, Activity
} from 'lucide-react';
import '../../styles/Dashboard/dashboardMap.css';

// Available maps configuration  
const AVAILABLE_MAPS = [
  { id: 'de_dust2', name: 'Dust II', img: 'de_dust2_radar_psd.png', available: true },
  { id: 'de_mirage', name: 'Mirage', img: 'de_mirage_radar_psd.png', available: true },
  { id: 'de_inferno', name: 'Inferno', img: 'de_inferno_radar_psd.png', available: true },
  { id: 'de_nuke', name: 'Nuke', img: 'de_nuke_radar_psd.png', available: true, 
    levels: { upper: 'de_nuke_radar_psd.png', lower: 'de_nuke_lower_radar_psd.png' },
    zThreshold: -500 // Below this = lower, above = upper
  },
  { id: 'de_overpass', name: 'Overpass', img: 'de_overpass_radar_psd.png', available: true },
  { id: 'de_train', name: 'Train', img: 'de_train_radar_psd.png', available: true,
    levels: { upper: 'de_train_radar_psd.png', lower: 'de_train_lower_radar_psd.png' },
    zThreshold: -50 // Below this = lower, above = upper
  },
  { id: 'de_vertigo', name: 'Vertigo', img: 'de_vertigo_radar_psd.png', available: true,
    levels: { upper: 'de_vertigo_radar_psd.png', lower: 'de_vertigo_lower_radar_psd.png' },
    zThreshold: 11700 // Vertigo: upper floor is Z > 11700 (based on CS2 data)
  },
  { id: 'de_anubis', name: 'Anubis', img: 'de_anubis_radar_psd.png', available: true },
  { id: 'de_ancient', name: 'Ancient', img: 'de_ancient_radar_psd.png', available: true },
];

// Map sections (displayed on map header)
const MAP_SECTIONS = [
  { id: 'hotpoints', label: 'Duels', icon: Flame },
  { id: 'grenades', label: 'Grenades', icon: Bomb },
  { id: 'heatmap', label: 'HeatMap', icon: Activity },
];

// Legacy Callout Hotspot - kept for backwards compatibility
// New maps use AdaptiveHotspotLayer instead
const CalloutHotspot = ({ callout, isSelected, isBest, isWorst, onClick }) => {
  const { name, position, rating, win_rate, kd } = callout;
  
  if (!position) return null;
  
  const ratingClass = rating === 'good' ? 'good' : rating === 'bad' ? 'bad' : 'neutral';
  
  return (
    <button
      className={`callout-hotspot compact ${ratingClass} ${isSelected ? 'selected' : ''} ${isWorst ? 'worst' : ''} ${isBest ? 'best' : ''}`}
      style={{ left: `${position.x}%`, top: `${position.y}%` }}
      onClick={() => onClick(callout)}
    >
      <div className="hotspot-ring"></div>
      <div className="hotspot-core">
        <span className="hotspot-label">{name.slice(0, 4)}</span>
      </div>
      {/* Stats tooltip - visible on hover */}
      <div className="hotspot-tooltip">
        <span className="tooltip-name">{name}</span>
        <div className="tooltip-stats">
          <span className={`stat-wr ${ratingClass}`}>{win_rate}%</span>
          <span className="stat-divider">•</span>
          <span className="stat-kd">{kd} K/D</span>
        </div>
      </div>
    </button>
  );
};

// Map Stats Summary - Now shows CT/T split stats
const MapStatsSummary = ({ sideStats, matchesAnalyzed, mapName }) => {
  const displayName = mapName?.replace('de_', '').toUpperCase() || 'MAP';
  const ct = sideStats?.CT;
  const t = sideStats?.T;

  // Calculate combined stats for the header
  const totalKills = (ct?.kills || 0) + (t?.kills || 0);
  const totalDeaths = (ct?.deaths || 0) + (t?.deaths || 0);
  const combinedKD = totalDeaths > 0 ? (totalKills / totalDeaths).toFixed(2) : totalKills.toFixed(2);

  return (
    <div className="map-summary">
      <div className="summary-header">
        <h3>{displayName}</h3>
        <span className="summary-matches">{matchesAnalyzed} partidas • K/D {combinedKD}</span>
      </div>
      
      <div className="side-stats-container">
        {/* CT Stats */}
        <div className="side-stats ct">
          <div className="side-label">
            <span className="side-icon ct">CT</span>
          </div>
          <div className="side-numbers">
            <div className="side-stat">
              <span className="stat-value">{ct?.kd?.toFixed(2) || '0.00'}</span>
              <span className="stat-label">K/D</span>
            </div>
            <div className="side-stat">
              <span className="stat-value">{Math.round(ct?.adr || 0)}</span>
              <span className="stat-label">ADR</span>
            </div>
            <div className="side-stat">
              <span className="stat-value">{ct?.hs_pct?.toFixed(0) || 0}%</span>
              <span className="stat-label">HS</span>
            </div>
          </div>
        </div>

        {/* T Stats */}
        <div className="side-stats t">
          <div className="side-label">
            <span className="side-icon t">T</span>
          </div>
          <div className="side-numbers">
            <div className="side-stat">
              <span className="stat-value">{t?.kd?.toFixed(2) || '0.00'}</span>
              <span className="stat-label">K/D</span>
            </div>
            <div className="side-stat">
              <span className="stat-value">{Math.round(t?.adr || 0)}</span>
              <span className="stat-label">ADR</span>
            </div>
            <div className="side-stat">
              <span className="stat-value">{t?.hs_pct?.toFixed(0) || 0}%</span>
              <span className="stat-label">HS</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Main Component
const DashboardMapV2 = () => {
  const { user } = useUser();
  const { dashboardData, loading: dashLoading } = useDashboardData(user);
  
  // State
  const [currentMap, setCurrentMap] = useState('de_dust2');
  const [currentLevel, setCurrentLevel] = useState('upper'); // 'upper' or 'lower' for multi-level maps
  const [showMapDropdown, setShowMapDropdown] = useState(false);
  const [selectedCallout, setSelectedCallout] = useState(null);
  const [activeSection, setActiveSection] = useState('hotpoints');
  const [activeSide, setActiveSide] = useState('all'); // 'all', 'ct', 't'
  const [showHeatmap, setShowHeatmap] = useState(true);
  
  // Zoom state for adaptive maps
  const [zoomLevel, setZoomLevel] = useState(1);
  const [panPosition, setPanPosition] = useState({ x: 50, y: 50 });
  
  // Grenade visibility state
  const [grenadeVisibleTypes, setGrenadeVisibleTypes] = useState({
    smoke: true, flash: true, he: true, molotov: true
  });
  const [selectedGrenadeCluster, setSelectedGrenadeCluster] = useState(null);
  const [selectedGrenadeType, setSelectedGrenadeType] = useState(null);
  
  // Movement heatmap state
  const [movementHeatmapIntensity, setMovementHeatmapIntensity] = useState(70);
  const [showMovementHeatmap, setShowMovementHeatmap] = useState(true);
  const [showFlowLines, setShowFlowLines] = useState(true);
  
  // Use adaptive clustering based on map profile
  const [useAdaptiveMode, setUseAdaptiveMode] = useState(true);
  
  const mapViewRef = React.useRef(null);

  // Fetch callout stats from new API
  const { 
    calloutStats, 
    sortedCallouts, 
    heatmapData, 
    matchesAnalyzed,
    sideStats,
    bestCallout, 
    worstCallout,
    loading: calloutLoading 
  } = useCalloutStats(currentMap);
  
  // Fetch grenade stats for grenade tab
  const {
    grenadeData,
    summary: grenadeSummary,
    insights: grenadeInsights,
    matchesAnalyzed: grenadeMatchesAnalyzed,
    loading: grenadeLoading
  } = useGrenadeStats(currentMap);

  // Fetch movement stats for heatmap tab
  const {
    heatmapGrid,
    flowLines,
    metrics: movementMetrics,
    matchesAnalyzed: movementMatchesAnalyzed,
    loading: movementLoading
  } = useMovementStats(currentMap);

  // Parallax effect
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
  
  // Check if current map has multiple levels
  const hasLevels = !!currentMapInfo.levels;
  
  // Get the correct radar image based on level
  const currentMapImage = useMemo(() => {
    if (hasLevels && currentMapInfo.levels) {
      return currentMapInfo.levels[currentLevel] || currentMapInfo.img;
    }
    return currentMapInfo.img;
  }, [currentMapInfo, currentLevel, hasLevels]);

  // Computed filtered data based on activeSide
  const filteredCallouts = useMemo(() => {
    if (activeSide === 'all') return sortedCallouts;
    
    const mapped = sortedCallouts.map(c => {
      const ctSplit = c.ct_t_split || {};
      const sideKills = activeSide === 'ct' ? (ctSplit.ct_kills || 0) : (ctSplit.t_kills || 0);
      const sideDeaths = activeSide === 'ct' ? (ctSplit.ct_deaths || 0) : (ctSplit.t_deaths || 0);
      const total = sideKills + sideDeaths;
      
      // Debug log
      if (total === 0) {
        console.log(`[Side Filter] EXCLUDING ${c.name}: ${activeSide.toUpperCase()} has ${sideKills}K/${sideDeaths}D (ct_t_split:`, ctSplit, ')');
        return null;
      }
      
      const kd = sideDeaths > 0 ? (sideKills / sideDeaths).toFixed(2) : sideKills.toFixed(2);
      const win_rate = Math.round((sideKills / total) * 100);
      const rating = win_rate >= 55 ? 'good' : win_rate <= 45 ? 'bad' : 'neutral';
      
      console.log(`[Side Filter] INCLUDING ${c.name}: ${activeSide.toUpperCase()} has ${sideKills}K/${sideDeaths}D = ${rating}`);
      
      return {
        ...c,
        kills: sideKills,
        deaths: sideDeaths,
        kd,
        win_rate,
        rating,
        sample_size: total
      };
    }).filter(Boolean);
    
    console.log(`[Side Filter] RESULT: ${activeSide.toUpperCase()}, ${sortedCallouts.length} total -> ${mapped.length} visible`);
    return mapped;
  }, [sortedCallouts, activeSide]);

  const bestCalloutSide = useMemo(() => filteredCallouts.find(c => c.rating === 'good'), [filteredCallouts]);
  const worstCalloutSide = useMemo(() => filteredCallouts.find(c => c.rating === 'bad'), [filteredCallouts]);

  // Get map profile for adaptive behavior
  const mapProfile = useMemo(() => getMapProfile(currentMap), [currentMap]);
  const isCompactMap = mapProfile.density === 'compact';

  // Callouts with positions for map display (filtered by level if multi-level map)
  const calloutsWithPositions = useMemo(() => {
    const withPos = filteredCallouts.filter(c => c.position && c.position.x !== undefined && c.position.y !== undefined);
    
    // If map has levels, filter by Z coordinate
    if (hasLevels && currentMapInfo.zThreshold !== undefined) {
      const threshold = currentMapInfo.zThreshold;
      const levelFiltered = withPos.filter(c => {
        // Try Z-based filtering first (backend should provide this)
        if (c.avg_z !== null && c.avg_z !== undefined) {
          const isUpper = c.avg_z >= threshold;
          const shouldShow = currentLevel === 'upper' ? isUpper : !isUpper;
          const levelName = isUpper ? 'UPPER' : 'LOWER';
          console.log(`✓ [${c.name}] Z=${Math.round(c.avg_z)} → ${levelName} (threshold=${threshold}) ${shouldShow ? '✓ VISIBLE' : '✗ hidden'}`);
          return shouldShow;
        }
        
        // Fallback: Use name-based level detection (only if backend fails)
        const guessedLevel = guessCalloutLevel(c.name, currentMap);
        const shouldShow = guessedLevel === currentLevel;
        console.warn(`⚠ [${c.name}] NO Z DATA → guessed ${guessedLevel.toUpperCase()} ${shouldShow ? '✓ VISIBLE' : '✗ hidden'}`);
        return shouldShow;
      });
      console.log(`📊 ${currentLevel.toUpperCase()}: ${withPos.length} total → ${levelFiltered.length} visible`);
      return levelFiltered;
    }
    
    return withPos;
  }, [filteredCallouts, hasLevels, currentMapInfo.zThreshold, currentLevel, currentMap]);

  // Process callouts with adaptive anti-overlap for dense maps
  const processedCallouts = useMemo(() => {
    if (!useAdaptiveMode || !isCompactMap) {
      return { callouts: calloutsWithPositions, densityZones: [], profile: mapProfile };
    }
    return processCalloutsForDisplay(calloutsWithPositions, currentMap);
  }, [calloutsWithPositions, currentMap, useAdaptiveMode, isCompactMap, mapProfile]);

  // Get names of callouts in current level (for heatmap filtering)
  const currentLevelCalloutNames = useMemo(() => {
    return new Set(calloutsWithPositions.map(c => c.name));
  }, [calloutsWithPositions]);

  // Filter heatmap data by current level and side
  const filteredHeatmapData = useMemo(() => {
    const threshold = hasLevels ? currentMapInfo.zThreshold : null;
    
    return heatmapData.filter(point => {
      // 1. Level filtering (Z coordinate)
      if (threshold !== null && point.avg_z !== undefined && point.avg_z !== null) {
        if (currentLevel === 'upper') {
          if (point.avg_z < threshold) return false;
        } else {
          if (point.avg_z >= threshold) return false;
        }
      } else {
        // Fallback: Name-based filtering if no Z data
        if (!currentLevelCalloutNames.has(point.callout)) return false;
      }
      
      return true;
    });
  }, [heatmapData, currentLevelCalloutNames, currentLevel, hasLevels, currentMapInfo.zThreshold]);

  // Find selected callout full data
  const selectedCalloutData = useMemo(() => {
    if (!selectedCallout) return null;
    return filteredCallouts.find(c => c.name === selectedCallout) || null;
  }, [selectedCallout, filteredCallouts]);

  // Handle callout selection (from map or grid)
  const handleCalloutSelect = (callout) => {
    const name = typeof callout === 'string' ? callout : callout.name;
    setSelectedCallout(prev => prev === name ? null : name);
  };

  // Handle cluster click from heatmap
  const handleClusterClick = (cluster) => {
    if (cluster.callout) {
      setSelectedCallout(cluster.callout);
    }
  };

  // Loading state
  if (dashLoading && !dashboardData) {
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
      <div className="map-dashboard v2">
        {/* Map Container */}
        <div className="map-main">
          {/* Map Header */}
          <div className="map-header">
            <div className="map-selector-wrapper">
              <button 
                className={`map-selector ${showMapDropdown ? 'open' : ''}`}
                onClick={() => setShowMapDropdown(!showMapDropdown)}
              >
                <MapPin size={20} />
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
                          setSelectedCallout(null);
                        }
                      }}
                      disabled={!map.available}
                    >
                      <span>{map.name}</span>
                      {currentMap === map.id && <Check size={14} />}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Section Navigation */}
            <div className="section-nav-wrapper">
              <div className="section-nav">
                {MAP_SECTIONS.map(section => (
                  <button
                    key={section.id}
                    className={`section-btn ${activeSection === section.id ? 'active' : ''}`}
                    onClick={() => {
                      setActiveSection(section.id);
                      setSelectedCallout(null);
                      setSelectedGrenadeCluster(null);
                    }}
                  >
                    <section.icon size={18} />
                    <span>{section.label}</span>
                  </button>
                ))}
              </div>

              {/* Side Filter Toggle */}
              <div className="side-filter-toggle">
                <button 
                  className={`side-filter-btn all ${activeSide === 'all' ? 'active' : ''}`}
                  onClick={() => setActiveSide('all')}
                >
                  ALL
                </button>
                <button 
                  className={`side-filter-btn ct ${activeSide === 'ct' ? 'active' : ''}`}
                  onClick={() => setActiveSide('ct')}
                >
                  CT
                </button>
                <button 
                  className={`side-filter-btn t ${activeSide === 't' ? 'active' : ''}`}
                  onClick={() => setActiveSide('t')}
                >
                  T
                </button>
              </div>
            </div>

            {/* Level Selector for multi-level maps - New Professional Component */}
            {hasLevels && (
              <LevelSelector
                mapName={currentMap}
                currentLevel={currentLevel}
                onLevelChange={(level) => {
                  setCurrentLevel(level);
                  setZoomLevel(1); // Reset zoom when changing levels
                }}
                compact={true}
              />
            )}
          </div>

          {/* Map View */}
          <div className="map-view" ref={mapViewRef}>
            {/* Zoomable Map Container */}
            <ZoomableMapContainer 
              zoomLevel={zoomLevel} 
              panPosition={panPosition}
              className="map-container"
            >
              <img 
                src={`/maps/${currentMapImage}`}
                alt={`${currentMapInfo.name} Radar${hasLevels ? ` (${currentLevel})` : ''}`}
                className="map-image"
              />
              
              {/* Grenade Overlay - Grenades mode */}
              {activeSection === 'grenades' && (
                <GrenadeOverlay
                  mapName={currentMap}
                  activeSide={activeSide}
                  visibleTypes={grenadeVisibleTypes}
                  onClusterClick={(cluster, type) => {
                    setSelectedGrenadeCluster(cluster);
                    setSelectedGrenadeType(type);
                  }}
                  selectedCluster={selectedGrenadeCluster}
                  selectedType={selectedGrenadeType}
                  currentLevel={currentLevel}
                  zThreshold={hasLevels ? currentMapInfo.zThreshold : null}
                />
              )}
              
              {/* Movement Heatmap Overlay - Heatmap mode */}
              {activeSection === 'heatmap' && (
                <MovementHeatmapOverlay
                  heatmapGrid={heatmapGrid}
                  heatmapIntensity={movementHeatmapIntensity}
                  showHeatmap={showMovementHeatmap}
                  activeSide={activeSide}
                  currentLevel={currentLevel}
                  zThreshold={hasLevels ? currentMapInfo.zThreshold : null}
                />
              )}
              
              {/* Callout Hotspots - HotPoints mode */}
              {/* Uses new AdaptiveHotspotLayer for compact maps, legacy for others */}
              {activeSection === 'hotpoints' && (
                useAdaptiveMode && isCompactMap ? (
                  <AdaptiveHotspotLayer
                    callouts={processedCallouts.callouts}
                    mapName={currentMap}
                    selectedCallout={selectedCallout}
                    bestCallout={bestCalloutSide}
                    worstCallout={worstCalloutSide}
                    onSelect={handleCalloutSelect}
                    zoomLevel={zoomLevel}
                  />
                ) : (
                  <div className="map-hotspots">
                    {calloutsWithPositions.map(callout => (
                      <CalloutHotspot
                        key={callout.name}
                        callout={callout}
                        isSelected={selectedCallout === callout.name}
                        isBest={bestCalloutSide?.name === callout.name}
                        isWorst={worstCalloutSide?.name === callout.name}
                        onClick={handleCalloutSelect}
                      />
                    ))}
                  </div>
                )
              )}
            </ZoomableMapContainer>

            {/* Legend - changes based on active section */}
            {activeSection === 'heatmap' ? (
              <div className="map-legend heatmap-legend">
                <div className="legend-item" style={{ color: '#3b82f6' }}>
                  <div className="legend-dot" style={{ background: '#3b82f6' }}></div>
                  <span>Bajo</span>
                </div>
                <div className="legend-item" style={{ color: '#22c55e' }}>
                  <div className="legend-dot" style={{ background: '#22c55e' }}></div>
                  <span>Medio</span>
                </div>
                <div className="legend-item" style={{ color: '#ef4444' }}>
                  <div className="legend-dot" style={{ background: '#ef4444' }}></div>
                  <span>Alto</span>
                </div>
              </div>
            ) : (
              <div className="map-legend">
                <div className="legend-item good">
                  <div className="legend-dot"></div>
                  <span>Fuerte</span>
                </div>
                <div className="legend-item neutral">
                  <div className="legend-dot"></div>
                  <span>Neutral</span>
                </div>
                <div className="legend-item bad">
                  <div className="legend-dot"></div>
                  <span>Débil</span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Side Panel */}
        <div className="map-sidebar">
          {/* Data Status */}
          <div className="data-status">
            {calloutLoading ? (
              <span className="loading-text">Cargando datos...</span>
            ) : (
              <span className="matches-count">
                {matchesAnalyzed} {matchesAnalyzed === 1 ? 'partida' : 'partidas'} • {sortedCallouts.length} callouts
              </span>
            )}
          </div>

          {/* Map Stats Summary */}
          <MapStatsSummary 
            sideStats={sideStats} 
            matchesAnalyzed={matchesAnalyzed}
            mapName={currentMap}
          />

          {/* Detail Panel Content - Based on active section */}
          <div className="detail-content">
            {/* HotPoints Mode - Show callout details */}
            {activeSection === 'hotpoints' && (
              <CalloutDetailPanel
                callout={selectedCalloutData}
                onClose={() => setSelectedCallout(null)}
              />
            )}
            
            {/* Grenades Mode - Show grenade controls and popup */}
            {activeSection === 'grenades' && (
              <GrenadeMapTab 
                mapName={currentMap}
                activeSide={activeSide}
                visibleTypes={grenadeVisibleTypes}
                onToggleType={(type) => setGrenadeVisibleTypes(prev => ({ ...prev, [type]: !prev[type] }))}
                selectedCluster={selectedGrenadeCluster}
                selectedType={selectedGrenadeType}
                onClusterSelect={(cluster, type) => {
                  setSelectedGrenadeCluster(cluster);
                  setSelectedGrenadeType(type);
                }}
                onClusterClose={() => {
                  setSelectedGrenadeCluster(null);
                  setSelectedGrenadeType(null);
                }}
              />
            )}
            
            {/* HeatMap Mode - Movement Analysis */}
            {activeSection === 'heatmap' && (
              <MovementHeatmapTab
                metrics={movementMetrics}
                matchesAnalyzed={movementMatchesAnalyzed}
                heatmapIntensity={movementHeatmapIntensity}
                onIntensityChange={setMovementHeatmapIntensity}
                showHeatmap={showMovementHeatmap}
                onToggleHeatmap={() => setShowMovementHeatmap(prev => !prev)}
                loading={movementLoading}
              />
            )}
          </div>

          {/* Quick Actions */}
          <div className="sidebar-actions">
            <Link to="/map-performance" className="action-btn">
              <BarChart3 size={16} />
              Ver todas las estadísticas
            </Link>
            <Link to="/history-games" className="action-btn">
              <Target size={16} />
              Historial de partidas
            </Link>
          </div>
        </div>
      </div>
    </NavigationFrame>
  );
};

export default DashboardMapV2;
