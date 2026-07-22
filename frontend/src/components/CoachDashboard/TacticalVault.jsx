import React, { lazy, Suspense, useEffect, useState } from 'react';
import { Archive } from 'lucide-react';
import ErrorBoundary from '../ErrorBoundary';
import MatchArchiveFilters from './MatchArchiveFilters';
import MatchArchiveRail from './MatchArchiveRail';
import MatchFocusPanel from './MatchFocusPanel';
import { getMatchSummary } from './matchPresentation';

const loadTacticalScene = () => import('./TacticalMatchScene');
const TacticalMatchScene = lazy(loadTacticalScene);

const useEnhancedVisuals = () => {
  const [isEnabled, setIsEnabled] = useState(false);

  useEffect(() => {
    const desktopQuery = window.matchMedia('(min-width: 900px)');
    const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');

    const updatePreference = () => {
      setIsEnabled(desktopQuery.matches && !motionQuery.matches && !document.hidden);
    };

    updatePreference();
    desktopQuery.addEventListener('change', updatePreference);
    motionQuery.addEventListener('change', updatePreference);
    document.addEventListener('visibilitychange', updatePreference);

    return () => {
      desktopQuery.removeEventListener('change', updatePreference);
      motionQuery.removeEventListener('change', updatePreference);
      document.removeEventListener('visibilitychange', updatePreference);
    };
  }, []);

  return isEnabled;
};

const ScenePoster = ({ summary }) => (
  <div className="vault-scene-poster">
    {summary ? <img src={summary.mapImage} alt="" width="1600" height="900" aria-hidden="true" /> : null}
    <div className="vault-poster-grid" aria-hidden="true" />
  </div>
);

const TacticalVisual = ({ match }) => {
  const summary = getMatchSummary(match);
  const canRenderScene = useEnhancedVisuals();
  const hasSummary = Boolean(summary);

  useEffect(() => {
    if (canRenderScene && hasSummary) loadTacticalScene();
  }, [canRenderScene, hasSummary]);

  return (
    <div className={`vault-visual ${summary?.isWin ? 'is-win' : 'is-loss'}`} aria-hidden="true">
      <div className="vault-visual-corner vault-visual-corner-top" />
      <div className="vault-visual-corner vault-visual-corner-bottom" />

      {canRenderScene && summary ? (
        <ErrorBoundary name="tactical-vault-scene" fallback={<ScenePoster summary={summary} />}>
          <Suspense fallback={<ScenePoster summary={summary} />}>
            <div className="vault-canvas" key={summary.id}>
              <TacticalMatchScene mapImage={summary.mapImage} isWin={summary.isWin} />
            </div>
          </Suspense>
        </ErrorBoundary>
      ) : <ScenePoster summary={summary} />}

      {summary ? (
        <div className="vault-visual-telemetry">
          <span>STRAT/AI · DEMO</span>
          <span>{String(summary.id || 'READY').slice(-12)}</span>
        </div>
      ) : null}
    </div>
  );
};

const TacticalVault = ({
  matches,
  allMatchesCount,
  selectedMatch,
  selectedMatchId,
  loading,
  error,
  query,
  mapFilter,
  resultFilter,
  dateFilter,
  sortBy,
  availableMaps,
  hasActiveFilters,
  analysisStatus,
  onQueryChange,
  onMapFilterChange,
  onResultFilterChange,
  onDateFilterChange,
  onSortChange,
  onResetFilters,
  onSelectMatch,
  onAnalyze,
  onRetry
}) => (
    <main className="tactical-vault">
      <header className="vault-header">
        <div className="vault-branding">
          <span className="vault-brand-mark"><Archive size={15} aria-hidden="true" /></span>
          <div>
            <span>Coach IA</span>
            <strong>Archivo táctico</strong>
          </div>
        </div>

        <div className="vault-header-actions">
          <span className="vault-match-count">
            <strong>{allMatchesCount}</strong> partidas listas
          </span>
        </div>
      </header>

      <div className="vault-stage">
        <div className="vault-control-column">
          <MatchArchiveFilters
            query={query}
            mapFilter={mapFilter}
            resultFilter={resultFilter}
            dateFilter={dateFilter}
            sortBy={sortBy}
            availableMaps={availableMaps}
            visibleCount={matches.length}
            totalCount={allMatchesCount}
            hasActiveFilters={hasActiveFilters}
            onQueryChange={onQueryChange}
            onMapFilterChange={onMapFilterChange}
            onResultFilterChange={onResultFilterChange}
            onDateFilterChange={onDateFilterChange}
            onSortChange={onSortChange}
            onResetFilters={onResetFilters}
          />
          <MatchFocusPanel
            match={selectedMatch}
            loading={loading}
            error={error}
            isAnalyzing={analysisStatus === 'analyzing'}
            onAnalyze={onAnalyze}
            onRetry={onRetry}
          />
        </div>
        <TacticalVisual match={selectedMatch} />
      </div>

      <section className="vault-archive" aria-labelledby="vault-archive-title">
        <div className="vault-archive-label">
          <span id="vault-archive-title">Partidas procesadas</span>
          <small>Usa ← → para recorrer</small>
        </div>
        <MatchArchiveRail
          matches={matches}
          selectedMatchId={selectedMatchId}
          loading={loading}
          error={error}
          onSelectMatch={onSelectMatch}
          onRetry={onRetry}
        />
      </section>

    </main>
  );

export default TacticalVault;
