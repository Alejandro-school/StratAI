import React, { Suspense, lazy, useCallback, useMemo, useState } from 'react';
import { AnimatePresence } from 'framer-motion';
import {
  AlertTriangle,
  ArrowLeft,
  GitCompareArrows,
  LoaderCircle,
  MousePointer2,
  Radio,
} from 'lucide-react';
import ContextRail from './ContextRail';
import PerformanceComparisonView from './PerformanceComparisonView';
import PlayerComparisonPicker from './PlayerComparisonPicker';
import SummaryPanel from './SummaryPanel';
import StatisticsWorkspace from './statistics/StatisticsWorkspace';
import {
  buildPerformanceContexts,
  hydrateSelectedContext,
} from '../data/performanceDataAdapters';
import { usePerformanceOverview } from '../hooks/usePerformanceData';
import '../styles/performancePrototype.css';

const PerformanceScene = lazy(() => import('../scene/PerformanceScene'));
const SAMPLE_LIMIT = 60;

const SceneFallback = () => (
  <div className="pf3-scene-fallback" role="status">
    <span />
    Cargando entorno…
  </div>
);

const PerformanceDataState = ({ error, onRetry }) => (
  <section className="pf3-real-data-state" role="status">
    {error ? <AlertTriangle size={28} /> : <LoaderCircle size={28} className="pf3-spin" />}
    <div>
      <strong>{error ? 'No se pudieron cargar tus estadísticas' : 'Procesando tus estadísticas reales'}</strong>
      <span>
        {error
          ? 'La API de rendimiento no ha respondido correctamente.'
          : 'Agregando mapas, duelos, utilidad, armas y economía.'}
      </span>
    </div>
    {error && <button type="button" onClick={onRetry}>Reintentar</button>}
  </section>
);

const PerformancePrototype = () => {
  const [selectedId, setSelectedId] = useState('general');
  const [activeSide, setActiveSide] = useState('all');
  const [comparisonPlayer, setComparisonPlayer] = useState(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const overviewQuery = usePerformanceOverview({ limit: SAMPLE_LIMIT });
  const mapQuery = usePerformanceOverview({
    mapName: selectedId === 'general' ? undefined : selectedId,
    limit: SAMPLE_LIMIT,
    enabled: Boolean(overviewQuery.data) && selectedId !== 'general' && !comparisonPlayer,
  });

  const contexts = useMemo(
    () => (overviewQuery.data ? buildPerformanceContexts(overviewQuery.data) : []),
    [overviewQuery.data],
  );
  const baseContext = useMemo(
    () => contexts.find((context) => context.id === selectedId) || contexts[0],
    [contexts, selectedId],
  );
  const selectedPayload = selectedId === 'general' ? overviewQuery.data : mapQuery.data;
  const selectedContext = useMemo(
    () => (baseContext && selectedPayload
      ? hydrateSelectedContext(baseContext, selectedPayload)
      : baseContext),
    [baseContext, selectedPayload],
  );

  const handleContextSelect = useCallback((contextId) => {
    setSelectedId(contextId);
    setActiveSide('all');
  }, []);

  const closeComparison = useCallback(() => {
    setComparisonPlayer(null);
    setPickerOpen(false);
  }, []);

  const selectComparisonPlayer = useCallback((player) => {
    setComparisonPlayer(player);
    setPickerOpen(false);
  }, []);

  const hasData = Boolean(overviewQuery.data && contexts.length && selectedContext);
  const comparisonOpen = Boolean(comparisonPlayer);

  return (
    <div className={`pf3 ${comparisonOpen ? 'pf3-comparison-shell' : ''}`}>
      {!hasData && (
        <PerformanceDataState
          error={overviewQuery.error}
          onRetry={overviewQuery.refetch}
        />
      )}

      {hasData && comparisonOpen && (
        <>
          <PerformanceComparisonView
            player={comparisonPlayer}
            onExit={closeComparison}
            onChangePlayer={() => setPickerOpen(true)}
          />
          <AnimatePresence>
            {pickerOpen && (
              <PlayerComparisonPicker
                activePlayerId={comparisonPlayer.id}
                currentSteamId={overviewQuery.data.steam_id}
                onClose={() => setPickerOpen(false)}
                onSelect={selectComparisonPlayer}
              />
            )}
          </AnimatePresence>
        </>
      )}

      {hasData && !comparisonOpen && (
        <>
          <section className="pf3-stage" aria-labelledby="performance-title">
            <div className="pf3-canvas" aria-hidden="true">
              <Suspense fallback={<SceneFallback />}>
                <PerformanceScene
                  contexts={contexts}
                  selectedId={selectedId}
                  onSelect={handleContextSelect}
                />
              </Suspense>
            </div>

            <div className="pf3-stage-vignette" />
            <header className="pf3-hero">
              <div className="pf3-hero-copy">
                <span className="pf3-live"><i /> Datos procesados</span>
                <h1 id="performance-title">Tu nivel, mapa a mapa</h1>
                <p>Resultados reales de las demos procesadas por Stratai.</p>
              </div>

              <div className="pf3-hero-actions">
                <button
                  type="button"
                  className="pf3-compare-button"
                  aria-expanded={pickerOpen}
                  aria-controls="comparison-picker"
                  onClick={() => setPickerOpen((current) => !current)}
                >
                  <GitCompareArrows size={17} aria-hidden="true" />
                  Comparar jugador
                </button>
                <span className="pf3-session">
                  <Radio size={15} aria-hidden="true" />
                  {overviewQuery.data.overview.total_matches} partidas disponibles
                </span>
              </div>
            </header>

            <SummaryPanel context={selectedContext} />

            {selectedId !== 'general' && (
              <button
                type="button"
                className="pf3-back-to-pool"
                onClick={() => handleContextSelect('general')}
              >
                <ArrowLeft size={16} aria-hidden="true" />
                Todos los mapas
              </button>
            )}

            <div className="pf3-scene-hint">
              <MousePointer2 size={16} aria-hidden="true" />
              Pasa el cursor para ampliar · Haz clic para analizar
            </div>

            <div className="pf3-rail-wrap">
              <ContextRail
                contexts={contexts}
                selectedId={selectedId}
                onSelect={handleContextSelect}
              />
            </div>

            <AnimatePresence>
              {pickerOpen && (
                <PlayerComparisonPicker
                  currentSteamId={overviewQuery.data.steam_id}
                  onClose={() => setPickerOpen(false)}
                  onSelect={selectComparisonPlayer}
                />
              )}
            </AnimatePresence>
          </section>

          <main className="pf3-data-deck">
            {selectedId !== 'general' && mapQuery.isFetching && (
              <div className="pf3-data-refresh" role="status">
                <LoaderCircle size={16} className="pf3-spin" /> Actualizando {selectedContext.name}
              </div>
            )}
            {selectedPayload && (
              <StatisticsWorkspace
                context={selectedContext}
                payload={selectedPayload}
                activeSide={activeSide}
                onSideChange={setActiveSide}
              />
            )}
          </main>
        </>
      )}
    </div>
  );
};

export default PerformancePrototype;
