import React, { useMemo, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useUser } from '../context/UserContext';
import { TacticalMapProvider, useTacticalMapState } from '../context/TacticalMapContext';
import { useTacticalMapData } from '../hooks/useTacticalMapData';
import { useCalloutStats } from '../hooks/useCalloutStats';
import { useMovementStats } from '../hooks/useMovementStats';
import { useGrenadeStats } from '../hooks/useGrenadeStats';
import CalloutDetailPanel from '../components/TacticalMap/CalloutDetailPanel';
import CalloutHotspot from '../components/TacticalMap/CalloutHotspot';
import MapStatsSummary from '../components/TacticalMap/MapStatsSummary';
import GrenadeMapTab, { GrenadeOverlay } from '../components/TacticalMap/GrenadeMapTab';
import MovementHeatmapTab from '../components/TacticalMap/MovementHeatmapTab';
import HeatmapCanvas from '../components/TacticalMap/HeatmapCanvas';
import { AdaptiveHotspotLayer } from '../components/TacticalMap/AdaptiveHotspot';
import LevelSelector from '../components/TacticalMap/LevelSelector';
import MapZoomControls, { ZoomableMapContainer } from '../components/TacticalMap/MapZoomControls';
import { processCalloutsForDisplay, getMapProfile } from '../utils/adaptiveClustering';
import { filterCalloutsBySide, filterCalloutsByLevel } from '../utils/tacticalFilters';
import { TACTICAL_MAPS } from '../utils/mapConfig';
import { 
  Target, MapPin, ChevronDown,
  Check, BarChart3, Flame, Bomb, Activity
} from 'lucide-react';
import '../styles/TacticalMap/tacticalMap.css';

const MAP_SECTIONS = [
  { id: 'hotpoints', label: 'Duels', icon: Flame },
  { id: 'grenades', label: 'Grenades', icon: Bomb },
  { id: 'heatmap', label: 'HeatMap', icon: Activity },
];

const MIN_DUEL_SAMPLE_BY_DENSITY = {
  compact: 3,
  standard: 2,
  sparse: 1,
};

const MAX_DUEL_MARKERS_BY_DENSITY = {
  compact: 20,
  standard: 28,
  sparse: 36,
};

const TacticalMapContent = () => {
  const { user } = useUser();
  const { dashboardData, loading: dashLoading } = useTacticalMapData(user);
  const {
    state: {
      currentMap,
      currentLevel,
      showMapDropdown,
      selectedCallout,
      activeSection,
      activeSide,
      grenadeVisibleTypes,
      selectedGrenadeCluster,
      selectedGrenadeType,
      movementHeatmapIntensity,
      showMovementHeatmap,
      zoomLevel,
    },
    setCurrentMap,
    setCurrentLevel,
    setShowMapDropdown,
    setActiveSection,
    setActiveSide,
    setSelectedCallout,
    toggleSelectedCallout,
    setSelectedGrenade,
    clearSelectedGrenade,
    toggleGrenadeType,
    setMovementHeatmapIntensity,
    toggleMovementHeatmap,
    setZoomLevel,
  } = useTacticalMapState();
  
  const mapViewRef = useRef(null);
  const [zoomResetSignal, setZoomResetSignal] = useState(0);

  const { 
    sortedCallouts, 
    matchesAnalyzed,
    sideStats,
    loading: calloutLoading 
  } = useCalloutStats(currentMap);
  
  const {
    heatmapGrid,
    flowLines,
    metrics: movementMetrics,
    matchesAnalyzed: movementMatchesAnalyzed,
    loading: movementLoading
  } = useMovementStats(currentMap);

  const {
    grenadeData,
    summary: grenadeSummary,
    matchesAnalyzed: grenadeMatchesAnalyzed,
    loading: grenadeLoading,
  } = useGrenadeStats(currentMap);

  useEffect(() => {
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

  const currentMapInfo = TACTICAL_MAPS.find(m => m.id === currentMap) || TACTICAL_MAPS[0];
  const hasLevels = !!currentMapInfo.levels;

  const currentMapImage = useMemo(() => {
    if (hasLevels && currentMapInfo.levels) {
      return currentMapInfo.levels[currentLevel] || currentMapInfo.img;
    }
    return currentMapInfo.img;
  }, [currentMapInfo, currentLevel, hasLevels]);

  const filteredCallouts = useMemo(
    () => filterCalloutsBySide(sortedCallouts, activeSide),
    [sortedCallouts, activeSide]
  );

  const mapProfile = useMemo(() => getMapProfile(currentMap), [currentMap]);
  const isCompactMap = mapProfile.density === 'compact';

  const calloutsWithPositions = useMemo(
    () =>
      filterCalloutsByLevel({
        callouts: filteredCallouts,
        hasLevels,
        zThreshold: currentMapInfo.zThreshold,
        currentLevel,
        currentMap,
      }),
    [filteredCallouts, hasLevels, currentMapInfo.zThreshold, currentLevel, currentMap]
  );

  const duelDisplayCallouts = useMemo(() => {
    const density = mapProfile.density || 'standard';
    const minSample = MIN_DUEL_SAMPLE_BY_DENSITY[density] ?? 2;
    const maxMarkers = MAX_DUEL_MARKERS_BY_DENSITY[density] ?? 28;

    const aboveThreshold = calloutsWithPositions
      .filter((callout) => (callout.sample_size || 0) >= minSample)
      .sort((a, b) => (b.sample_size || 0) - (a.sample_size || 0));

    const selectedCandidate = selectedCallout
      ? calloutsWithPositions.find((callout) => callout.name === selectedCallout)
      : null;

    const withSelected = selectedCandidate && !aboveThreshold.some((callout) => callout.name === selectedCandidate.name)
      ? [selectedCandidate, ...aboveThreshold]
      : aboveThreshold;

    const uniqueByName = [];
    const seen = new Set();
    withSelected.forEach((callout) => {
      if (!seen.has(callout.name)) {
        seen.add(callout.name);
        uniqueByName.push(callout);
      }
    });

    return uniqueByName.slice(0, maxMarkers);
  }, [calloutsWithPositions, mapProfile.density, selectedCallout]);

  const bestCalloutSide = useMemo(() => duelDisplayCallouts.find(c => c.rating === 'good'), [duelDisplayCallouts]);
  const worstCalloutSide = useMemo(() => duelDisplayCallouts.find(c => c.rating === 'bad'), [duelDisplayCallouts]);

  const processedCallouts = useMemo(() => {
    if (!isCompactMap) {
      return { callouts: duelDisplayCallouts, densityZones: [], profile: mapProfile };
    }
    return processCalloutsForDisplay(duelDisplayCallouts, currentMap);
  }, [duelDisplayCallouts, currentMap, isCompactMap, mapProfile]);

  const selectedCalloutData = useMemo(() => {
    if (!selectedCallout) return null;
    return filteredCallouts.find(c => c.name === selectedCallout) || null;
  }, [selectedCallout, filteredCallouts]);

  const handleCalloutSelect = (callout) => {
    const name = typeof callout === 'string' ? callout : callout.name;
    toggleSelectedCallout(name);
  };


  if (dashLoading && !dashboardData) {
    return (
      <NavigationFrame>
        <div className="map-dashboard loading">
          <div className="loading-spinner"></div>
          <span>Cargando mapa tÃ¡ctico...</span>
        </div>
      </NavigationFrame>
    );
  }

  return (
    <NavigationFrame>
      <div className="map-dashboard v2">
        <div className="map-main">
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
                  {TACTICAL_MAPS.map(map => (
                    <button
                      key={map.id}
                      className={`map-option ${currentMap === map.id ? 'active' : ''} ${!map.available ? 'disabled' : ''}`}
                      onClick={() => {
                        if (map.available) {
                          setCurrentMap(map.id);
                          setZoomLevel(1);
                          setZoomResetSignal((prev) => prev + 1);
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

            <div className="section-nav-wrapper">
              <div className="section-nav">
                {MAP_SECTIONS.map(section => (
                  <button
                    key={section.id}
                    className={`section-btn ${activeSection === section.id ? 'active' : ''}`}
                    onClick={() => {
                      setActiveSection(section.id);
                    }}
                  >
                    <section.icon size={18} />
                    <span>{section.label}</span>
                  </button>
                ))}
              </div>

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

            {hasLevels && (
              <LevelSelector
                mapName={currentMap}
                currentLevel={currentLevel}
                onLevelChange={(level) => {
                  setCurrentLevel(level);
                  setZoomLevel(1);
                  setZoomResetSignal((prev) => prev + 1);
                }}
                compact={true}
              />
            )}
          </div>

          <div className="map-view" ref={mapViewRef}>
            <ZoomableMapContainer
              zoomLevel={zoomLevel}
              className="map-container"
              viewKey={`${currentMap}:${currentLevel}:${activeSection}`}
              resetSignal={zoomResetSignal}
            >
              <img 
                src={`/maps/${currentMapImage}`}
                alt={`${currentMapInfo.name} Radar${hasLevels ? ` (${currentLevel})` : ''}`}
                className="map-image"
              />
              
              {activeSection === 'grenades' && (
                <GrenadeOverlay
                  mapName={currentMap}
                  activeSide={activeSide}
                  visibleTypes={grenadeVisibleTypes}
                  onClusterClick={(cluster, type) => {
                    setSelectedGrenade(cluster, type);
                  }}
                  selectedCluster={selectedGrenadeCluster}
                  selectedType={selectedGrenadeType}
                  currentLevel={currentLevel}
                  zThreshold={hasLevels ? currentMapInfo.zThreshold : null}
                  grenadeData={grenadeData}
                  loading={grenadeLoading}
                  zoomLevel={zoomLevel}
                />
              )}

              {activeSection === 'heatmap' && (
                <HeatmapCanvas
                  points={heatmapGrid}
                  intensity={movementHeatmapIntensity}
                  visible={showMovementHeatmap}
                  activeSide={activeSide}
                  hasLevels={hasLevels}
                  currentLevel={currentLevel}
                  zThreshold={currentMapInfo.zThreshold}
                />
              )}
              
            {activeSection === 'hotpoints' && (
                 isCompactMap ? (
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
                    {duelDisplayCallouts.map(callout => (
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

            {activeSection === 'heatmap' ? (
              <div className="map-legend heatmap-legend">
                <div className="legend-item low">
                  <div className="legend-dot"></div>
                  <span>Bajo</span>
                </div>
                <div className="legend-item medium">
                  <div className="legend-dot"></div>
                  <span>Medio</span>
                </div>
                <div className="legend-item high">
                  <div className="legend-dot"></div>
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
                  <span>DÃ©bil</span>
                </div>
              </div>
            )}

            <MapZoomControls
              zoomLevel={zoomLevel}
              mapName={currentMap}
              onZoomChange={setZoomLevel}
              onReset={() => {
                setZoomLevel(1);
                setZoomResetSignal((prev) => prev + 1);
              }}
            />
          </div>
        </div>

        <div className="map-sidebar">
          <div className="data-status">
            {calloutLoading ? (
              <span className="loading-text">Cargando datos...</span>
            ) : (
              <span className="matches-count">
                {matchesAnalyzed} {matchesAnalyzed === 1 ? 'partida' : 'partidas'} â€¢ {sortedCallouts.length} callouts
              </span>
            )}
          </div>

          <MapStatsSummary 
            sideStats={sideStats} 
            matchesAnalyzed={matchesAnalyzed}
            mapName={currentMap}
          />

          <div className="detail-content">
            {activeSection === 'hotpoints' && (
              <CalloutDetailPanel
                callout={selectedCalloutData}
                onClose={() => setSelectedCallout(null)}
              />
            )}
            
            {activeSection === 'grenades' && (
              <GrenadeMapTab 
                mapName={currentMap}
                activeSide={activeSide}
                visibleTypes={grenadeVisibleTypes}
                onToggleType={toggleGrenadeType}
                selectedCluster={selectedGrenadeCluster}
                selectedType={selectedGrenadeType}
                onClusterSelect={(cluster, type) => {
                  setSelectedGrenade(cluster, type);
                }}
                onClusterClose={clearSelectedGrenade}
                currentLevel={currentLevel}
                hasLevels={hasLevels}
                zThreshold={currentMapInfo.zThreshold}
                grenadeData={grenadeData}
                summary={grenadeSummary}
                matchesAnalyzed={grenadeMatchesAnalyzed}
                loading={grenadeLoading}
              />
            )}
            
            {activeSection === 'heatmap' && (
              <MovementHeatmapTab
                metrics={movementMetrics}
                matchesAnalyzed={movementMatchesAnalyzed}
                heatmapIntensity={movementHeatmapIntensity}
                onIntensityChange={setMovementHeatmapIntensity}
                showHeatmap={showMovementHeatmap}
                onToggleHeatmap={toggleMovementHeatmap}
                loading={movementLoading}
                flowLines={flowLines}
              />
            )}
          </div>

          <div className="sidebar-actions">
            <Link to="/map-performance" className="action-btn">
              <BarChart3 size={16} />
              Ver todas las estadÃ­sticas
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

const TacticalMap = () => (
  <TacticalMapProvider>
    <TacticalMapContent />
  </TacticalMapProvider>
);

export default TacticalMap;

