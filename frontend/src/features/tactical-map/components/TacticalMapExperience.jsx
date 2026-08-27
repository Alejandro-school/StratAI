import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { BarChart3, BookOpen, History, X } from 'lucide-react';
import { Link, useSearchParams } from 'react-router-dom';
import { useAuth } from '../../../auth/useAuth';
import ErrorBoundary from '../../../components/ErrorBoundary';
import NavigationFrame from '../../../components/Layout/NavigationFrame';
import TacticalInsightsPanel from '../../../components/TacticalMap/TacticalInsightsPanel';
import TacticalIntro from '../../../components/TacticalMap/TacticalIntro';
import { useTacticalMapState } from '../../../context/TacticalMapContext';
import { matchesQueryOptions } from '../../matches/queries/matchQueries';
import { TACTICAL_MAPS } from '../../../utils/mapConfig';
import { getTacticalMapBootstrap } from '../domain/tacticalMapBootstrap';
import { useTacticalExperienceData } from '../hooks/useTacticalExperienceData';
import { useTacticalRouteSync } from '../hooks/useTacticalRouteSync';
import TacticalMapStage from './TacticalMapStage';
import TacticalPresencePanel from './TacticalPresencePanel';
import TacticalToolbar from './TacticalToolbar';
import TacticalUtilityPanel from './TacticalUtilityPanel';
import TacticalZonePanel from './TacticalZonePanel';
import '../../../styles/TacticalMap/tacticalExperience.css';
import '../../../styles/TacticalMap/tacticalEditorial.css';

const INTRO_STORAGE_KEY = 'stratai:tactical-map:intro:v3';

const LENS_COPY = {
  briefing: ['Briefing', 'Tres señales para decidir qué conservar, variar y corregir.'],
  positioning: ['Posicionamiento', 'Densidad, transiciones y distribución CT/T sin mezclar controles con conclusiones.'],
  combat: ['Combate', 'Una lectura por zonas con volumen, eficiencia, impacto y riesgo sobre la misma evidencia.'],
  utility: ['Utilidad', 'Usos y efecto medido; lo que no existe en los datos se declara como no disponible.'],
};

const SIDE_COPY = {
  ct: { code: 'CT', role: 'Defensa' },
  t: { code: 'T', role: 'Ataque' },
};

const getInitialIntroState = () => {
  if (typeof window === 'undefined') return false;
  try {
    return window.localStorage.getItem(INTRO_STORAGE_KEY) !== 'seen';
  } catch {
    return true;
  }
};

const TacticalMapExperience = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const { user } = useAuth();
  const hasExplicitMapOnEntry = useRef(searchParams.has('map')).current;
  const matchesQuery = useQuery({
    ...matchesQueryOptions(user?.steam_id),
    enabled: Boolean(user?.steam_id) && !hasExplicitMapOnEntry,
  });
  const [zoomResetSignal, setZoomResetSignal] = useState(0);
  const [selectedInsightId, setSelectedInsightId] = useState(() => searchParams.get('signal'));
  const [isIntroOpen, setIsIntroOpen] = useState(getInitialIntroState);
  const {
    state,
    dispatch,
    setCurrentMap,
    setCurrentLevel,
    setShowMapDropdown,
    setActiveSection,
    setCombatMetric,
    setActiveSide,
    setSelectedCallout,
    toggleSelectedCallout,
    setSelectedGrenade,
    clearSelectedGrenade,
    toggleGrenadeType,
    setMovementHeatmapIntensity,
    toggleMovementHeatmap,
    toggleMovementRoutes,
    setZoomLevel,
  } = useTacticalMapState();
  const {
    currentMap,
    currentLevel,
    showMapDropdown,
    selectedCallout,
    activeSection,
    combatMetric,
    activeSide,
    grenadeVisibleTypes,
    selectedGrenadeCluster,
    selectedGrenadeType,
    movementHeatmapIntensity,
    showMovementHeatmap,
    showMovementRoutes,
    zoomLevel,
  } = state;
  const mapBootstrap = useMemo(() => getTacticalMapBootstrap({
    currentMap,
    hasExplicitMap: hasExplicitMapOnEntry,
    isLoading: Boolean(user?.steam_id) && matchesQuery.isPending,
    matches: matchesQuery.data,
    supportedMaps: TACTICAL_MAPS,
  }), [currentMap, hasExplicitMapOnEntry, matchesQuery.data, matchesQuery.isPending, user?.steam_id]);
  const data = useTacticalExperienceData({
    currentMap,
    currentLevel,
    activeSection,
    activeSide,
    selectedCallout,
    enabled: !mapBootstrap.deferRouteWrite,
  });
  const { mapInfo, hasLevels } = data;
  const routeState = useMemo(() => ({
    currentMap,
    currentLevel,
    activeSection,
    combatMetric,
    activeSide,
    selectedCallout,
  }), [activeSection, activeSide, combatMetric, currentLevel, currentMap, selectedCallout]);
  useEffect(() => {
    if (mapBootstrap.targetMap) setCurrentMap(mapBootstrap.targetMap);
  }, [mapBootstrap.targetMap, setCurrentMap]);

  useTacticalRouteSync({
    searchParams,
    setSearchParams,
    dispatch,
    routeState,
    selectedSignalId: selectedInsightId,
    setSelectedSignalId: setSelectedInsightId,
    hasLevels,
    deferRouteWrite: mapBootstrap.deferRouteWrite,
  });

  const resetZoom = useCallback(() => {
    setZoomLevel(1);
    setZoomResetSignal((value) => value + 1);
  }, [setZoomLevel]);
  const clearContext = useCallback(() => {
    setSelectedInsightId(null);
    setSelectedCallout(null);
    clearSelectedGrenade();
  }, [clearSelectedGrenade, setSelectedCallout]);
  const handleMapChange = useCallback((mapName) => {
    setCurrentMap(mapName);
    setSelectedInsightId(null);
    resetZoom();
  }, [resetZoom, setCurrentMap]);
  const handleLensChange = useCallback((lens) => {
    setActiveSection(lens);
    setSelectedInsightId(null);
    resetZoom();
  }, [resetZoom, setActiveSection]);
  const handleSideChange = useCallback((side) => {
    setActiveSide(side);
    setSelectedInsightId(null);
  }, [setActiveSide]);
  const handleLevelChange = useCallback((level) => {
    setCurrentLevel(level);
    setSelectedInsightId(null);
    resetZoom();
  }, [resetZoom, setCurrentLevel]);
  const handleMetricChange = useCallback((metric) => {
    setCombatMetric(metric);
    setSelectedInsightId(null);
  }, [setCombatMetric]);
  const handleSignalSelect = useCallback((signal) => {
    setSelectedInsightId(signal.id);
    if (signal.callout?.name) setSelectedCallout(signal.callout.name);
  }, [setSelectedCallout]);
  const handleInsightSelect = useCallback((insight) => {
    setSelectedInsightId(insight.id);
    if (insight.calloutName) setSelectedCallout(insight.calloutName);
  }, [setSelectedCallout]);
  const dismissIntro = useCallback(() => {
    setIsIntroOpen(false);
    try {
      window.localStorage.setItem(INTRO_STORAGE_KEY, 'seen');
    } catch {
      // Local storage can be unavailable in private browsing.
    }
  }, []);

  const grenadeProps = useMemo(() => ({
    mapName: currentMap,
    activeSide,
    visibleTypes: grenadeVisibleTypes,
    onClusterClick: (cluster, type) => setSelectedGrenade(cluster, type),
    selectedCluster: selectedGrenadeCluster,
    selectedType: selectedGrenadeType,
    currentLevel,
    zThreshold: hasLevels ? mapInfo.zThreshold : null,
    grenadeData: data.grenadeQuery.grenadeData,
    loading: data.grenadeQuery.loading,
    zoomLevel,
  }), [
    activeSide,
    currentLevel,
    currentMap,
    data.grenadeQuery.grenadeData,
    data.grenadeQuery.loading,
    grenadeVisibleTypes,
    hasLevels,
    mapInfo.zThreshold,
    selectedGrenadeCluster,
    selectedGrenadeType,
    setSelectedGrenade,
    zoomLevel,
  ]);
  const [lensTitle, lensDescription] = LENS_COPY[activeSection] ?? LENS_COPY.briefing;
  const sideCopy = SIDE_COPY[activeSide] ?? SIDE_COPY.ct;
  const hasContextSelection = Boolean(
    (activeSection === 'briefing' && selectedInsightId)
    || (activeSection === 'combat' && selectedCallout)
    || (activeSection === 'utility' && selectedGrenadeCluster)
  );

  return (
    <NavigationFrame>
      <main className="tactical-experience" data-side={activeSide}>
        <header className="tactical-experience__header">
          <div>
            <span className="tactical-experience__eyebrow">Archivo de rendimiento espacial</span>
            <h1>Mapa Táctico</h1>
            <p><strong>{lensTitle}.</strong> {lensDescription}</p>
          </div>
          <button type="button" className="tactical-help-button" onClick={() => setIsIntroOpen(true)}>
            <BookOpen size={17} aria-hidden="true" />
            Cómo leerlo
          </button>
        </header>

        <TacticalToolbar
          maps={TACTICAL_MAPS}
          currentMap={currentMap}
          currentMapInfo={mapInfo}
          activeLens={activeSection}
          activeSide={activeSide}
          currentLevel={currentLevel}
          hasLevels={hasLevels}
          isMapMenuOpen={showMapDropdown}
          onMapMenuChange={setShowMapDropdown}
          onMapChange={handleMapChange}
          onLensChange={handleLensChange}
          onSideChange={handleSideChange}
          onLevelChange={handleLevelChange}
        />

        <div
          key={activeSide}
          className="tactical-side-context"
          role="status"
          aria-live="polite"
        >
          <strong className="tactical-side-context__code">{sideCopy.code}</strong>
          <span>
            <small>Contexto activo</small>
            <strong>{sideCopy.role}</strong>
          </span>
          <p>Mapa y conclusiones desglosados por zonas para el bando {sideCopy.code}.</p>
        </div>

        <div className="tactical-experience__workspace">
          <TacticalMapStage
            currentMap={currentMap}
            mapInfo={mapInfo}
            mapImage={data.mapImage}
            currentLevel={currentLevel}
            hasLevels={hasLevels}
            activeLens={activeSection}
            activeSide={activeSide}
            combatMetric={combatMetric}
            heatmapGrid={data.movementQuery.heatmapGrid}
            flowLines={data.movementQuery.flowLines}
            heatmapIntensity={movementHeatmapIntensity}
            showHeatmap={showMovementHeatmap}
            showRoutes={showMovementRoutes}
            tacticalCallouts={data.tacticalModel.callouts}
            baselineWinRate={data.tacticalModel.baselineWinRate}
            selectedCallout={selectedCallout}
            signals={data.presentation.signals}
            selectedSignalId={selectedInsightId}
            grenadeProps={grenadeProps}
            loading={data.loading}
            error={data.error}
            onRetry={data.retry}
            onCalloutSelect={(callout) => toggleSelectedCallout(callout.name)}
            onSignalSelect={handleSignalSelect}
            zoomLevel={zoomLevel}
            zoomResetSignal={zoomResetSignal}
            onZoomChange={setZoomLevel}
            onZoomReset={resetZoom}
          />

          {hasContextSelection ? (
            <button
              type="button"
              className="tactical-bottom-sheet-backdrop"
              aria-label="Cerrar detalle táctico"
              onClick={clearContext}
            />
          ) : null}
          <div
            id="tactical-lens-panel"
            className={`tactical-experience__side ${hasContextSelection ? 'has-context-selection' : ''}`}
            role="tabpanel"
            aria-labelledby={`tactical-tab-${activeSection}`}
          >
            {hasContextSelection ? (
              <div className="tactical-bottom-sheet-handle">
                <span>Detalle seleccionado</span>
                <button type="button" onClick={clearContext} aria-label="Cerrar detalle táctico">
                  <X size={18} aria-hidden="true" />
                </button>
              </div>
            ) : null}

            <ErrorBoundary name="tactical-sidebar" message="No se ha podido mostrar el análisis">
              {activeSection === 'briefing' ? (
                <TacticalInsightsPanel
                  mapName={currentMap}
                  activeSide={activeSide}
                  matchesAnalyzed={data.matchesAnalyzed}
                  confidence={data.presentation.globalConfidence}
                  insights={data.presentation.insights}
                  selectedInsightId={selectedInsightId}
                  onInsightSelect={handleInsightSelect}
                />
              ) : null}

              {activeSection === 'positioning' ? (
                <TacticalPresencePanel
                  metrics={data.movementMetrics}
                  matchesAnalyzed={data.movementQuery.matchesAnalyzed}
                  activeSide={activeSide}
                  sideDistribution={data.sideDistribution}
                  heatmapIntensity={movementHeatmapIntensity}
                  showHeatmap={showMovementHeatmap}
                  showRoutes={showMovementRoutes}
                  routeCount={data.movementQuery.flowLines.length}
                  onIntensityChange={setMovementHeatmapIntensity}
                  onToggleHeatmap={toggleMovementHeatmap}
                  onToggleRoutes={toggleMovementRoutes}
                />
              ) : null}

              {activeSection === 'combat' ? (
                <TacticalZonePanel
                  metric={combatMetric}
                  activeSide={activeSide}
                  callouts={data.tacticalModel.callouts}
                  selectedCallout={data.selectedCalloutData}
                  baselineWinRate={data.tacticalModel.baselineWinRate}
                  matchesAnalyzed={data.calloutQuery.matchesAnalyzed}
                  onMetricChange={handleMetricChange}
                  onSelect={(callout) => setSelectedCallout(callout.name)}
                  onClose={() => setSelectedCallout(null)}
                />
              ) : null}

              {activeSection === 'utility' ? (
                <TacticalUtilityPanel
                  mapName={currentMap}
                  activeSide={activeSide}
                  currentLevel={currentLevel}
                  hasLevels={hasLevels}
                  zThreshold={mapInfo.zThreshold}
                  grenadeData={data.grenadeQuery.grenadeData}
                  matchesAnalyzed={data.grenadeQuery.matchesAnalyzed}
                  visibleTypes={grenadeVisibleTypes}
                  selectedCluster={selectedGrenadeCluster}
                  selectedType={selectedGrenadeType}
                  onToggleType={toggleGrenadeType}
                  onClusterSelect={setSelectedGrenade}
                  onClusterClose={clearSelectedGrenade}
                />
              ) : null}
            </ErrorBoundary>

            <nav className="tactical-side-actions" aria-label="Más análisis">
              <Link to="/performance"><BarChart3 size={16} aria-hidden="true" />Rendimiento completo</Link>
              <Link to="/history-games"><History size={16} aria-hidden="true" />Historial</Link>
            </nav>
          </div>
        </div>
      </main>

      <TacticalIntro
        isOpen={isIntroOpen}
        mapName={currentMap}
        matchesAnalyzed={data.matchesAnalyzed}
        onDismiss={dismissIntro}
        onExplore={dismissIntro}
      />
    </NavigationFrame>
  );
};

export default TacticalMapExperience;
