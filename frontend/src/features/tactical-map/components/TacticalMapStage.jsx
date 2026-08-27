import React, { lazy, Suspense } from 'react';
import { AlertCircle, LoaderCircle } from 'lucide-react';
import ErrorBoundary from '../../../components/ErrorBoundary';
import FlowLinesOverlay from '../../../components/TacticalMap/FlowLinesOverlay';
import HeatmapCanvas from '../../../components/TacticalMap/HeatmapCanvas';
import MapZoomControls, { ZoomableMapContainer } from '../../../components/TacticalMap/MapZoomControls';
import TacticalSignalLayer from '../../../components/TacticalMap/TacticalSignalLayer';
import TacticalDuelLayer from './TacticalDuelLayer';

const GrenadeOverlay = lazy(() => (
  import('../../../components/TacticalMap/GrenadeMapTab').then((module) => ({
    default: module.GrenadeOverlay,
  }))
));

const LENS_LEGENDS = {
  briefing: [
    ['strength', 'Fortaleza'],
    ['habit', 'Hábito'],
    ['risk', 'Riesgo'],
  ],
  positioning: [
    ['low', 'Menos tiempo'],
    ['high', 'Más tiempo'],
  ],
  combat: [
    ['volume', 'Tamaño = evidencia'],
    ['positive', 'Señal favorable'],
    ['negative', 'Señal de riesgo'],
  ],
  utility: [
    ['volume', 'Tamaño = usos'],
  ],
};

const TacticalMapStage = ({
  currentMap,
  mapInfo,
  mapImage,
  currentLevel,
  hasLevels,
  activeLens,
  activeSide,
  combatMetric,
  heatmapGrid,
  flowLines,
  heatmapIntensity,
  showHeatmap,
  showRoutes,
  tacticalCallouts,
  baselineWinRate,
  selectedCallout,
  signals,
  selectedSignalId,
  grenadeProps,
  loading,
  error,
  onRetry,
  onCalloutSelect,
  onSignalSelect,
  zoomLevel,
  zoomResetSignal,
  onZoomChange,
  onZoomReset,
}) => {
  const baseLegends = LENS_LEGENDS[activeLens] ?? [];
  const legends = activeLens === 'utility'
    ? [[activeSide, activeSide.toUpperCase()], ...baseLegends]
    : baseLegends;

  return (
    <section className="tactical-stage-card" aria-labelledby="tactical-stage-title">
      <header className="tactical-stage-card__header">
        <div>
          <span className="tactical-stage-card__kicker">Modelo espacial</span>
          <h2 id="tactical-stage-title">{mapInfo.name}</h2>
        </div>
        <div className="tactical-stage-card__coordinates" aria-label="Filtro actual">
          <span className={`is-${activeSide}`}>{activeSide === 'ct' ? 'CT · Defensa' : 'T · Ataque'}</span>
          {hasLevels ? <span>{currentLevel === 'upper' ? 'Planta superior' : 'Planta inferior'}</span> : null}
        </div>
      </header>

      <div className="tactical-stage">
        <ZoomableMapContainer
          zoomLevel={zoomLevel}
          className="tactical-stage__viewport"
          viewKey={`${currentMap}:${currentLevel}:${activeLens}:${combatMetric}`}
          resetSignal={zoomResetSignal}
        >
          <img
            src={`/maps/${mapImage}`}
            alt={`Radar de ${mapInfo.name}${hasLevels ? `, planta ${currentLevel === 'upper' ? 'superior' : 'inferior'}` : ''}`}
            className="tactical-stage__radar"
            width="3000"
            height="3000"
            fetchPriority="high"
            draggable="false"
          />
          <div className="tactical-stage__grid" aria-hidden="true" />

          <ErrorBoundary name="tactical-map-overlay" message="No se ha podido dibujar esta capa">
            {activeLens === 'briefing' || activeLens === 'positioning' ? (
              <HeatmapCanvas
                points={heatmapGrid}
                intensity={activeLens === 'briefing' ? 52 : heatmapIntensity}
                visible={activeLens === 'briefing' || showHeatmap}
                activeSide={activeSide}
                hasLevels={hasLevels}
                currentLevel={currentLevel}
                zThreshold={mapInfo.zThreshold}
              />
            ) : null}

            {activeLens === 'positioning' && showRoutes ? (
              <FlowLinesOverlay
                flowLines={flowLines}
                activeSide={activeSide}
                hasLevels={hasLevels}
                currentLevel={currentLevel}
                zThreshold={mapInfo.zThreshold}
                visible
              />
            ) : null}

            {activeLens === 'briefing' ? (
              <TacticalSignalLayer
                signals={signals}
                selectedSignalId={selectedSignalId}
                onSelect={onSignalSelect}
                toneMap={false}
              />
            ) : null}

            {activeLens === 'combat' ? (
              <TacticalDuelLayer
                callouts={tacticalCallouts}
                metric={combatMetric}
                baselineWinRate={baselineWinRate}
                selectedCallout={selectedCallout}
                onSelect={onCalloutSelect}
              />
            ) : null}

            {activeLens === 'utility' ? (
              <Suspense fallback={null}>
                <GrenadeOverlay {...grenadeProps} />
              </Suspense>
            ) : null}
          </ErrorBoundary>
        </ZoomableMapContainer>

        {loading ? (
          <div className="tactical-stage-state" role="status" aria-live="polite">
            <LoaderCircle className="is-spinning" size={24} aria-hidden="true" />
            <strong>Reconstruyendo tu lectura…</strong>
            <span>Agregando las demos de este mapa</span>
          </div>
        ) : null}

        {!loading && error ? (
          <div className="tactical-stage-state is-error" role="alert">
            <AlertCircle size={24} aria-hidden="true" />
            <strong>No se ha podido cargar esta lectura</strong>
            <span>Comprueba la conexión y vuelve a intentarlo.</span>
            <button type="button" onClick={onRetry}>Reintentar</button>
          </div>
        ) : null}

        <div className={`tactical-stage-legend is-${activeLens}`} aria-label="Leyenda del mapa">
          {legends.map(([tone, label]) => (
            <span key={tone}><i className={`legend-${tone}`} aria-hidden="true" />{label}</span>
          ))}
        </div>

        <MapZoomControls
          zoomLevel={zoomLevel}
          mapName={currentMap}
          onZoomChange={onZoomChange}
          onReset={onZoomReset}
        />
      </div>
    </section>
  );
};

export default React.memo(TacticalMapStage);
