import React, { memo, useEffect, useRef } from 'react';
import { Loader2, RefreshCcw } from 'lucide-react';
import { getMatchId, getMatchSummary } from './matchPresentation';

const MatchArchiveRail = memo(({
  matches,
  selectedMatchId,
  loading,
  error,
  onSelectMatch,
  onRetry
}) => {
  const railRef = useRef(null);

  useEffect(() => {
    const selectedItem = railRef.current?.querySelector('[aria-current="true"]');
    selectedItem?.scrollIntoView({ block: 'nearest', inline: 'center' });
  }, [selectedMatchId]);

  const selectSibling = (direction) => {
    const currentIndex = matches.findIndex((match) => getMatchId(match) === selectedMatchId);
    if (currentIndex < 0) return;

    const nextIndex = Math.min(Math.max(currentIndex + direction, 0), matches.length - 1);
    onSelectMatch(getMatchId(matches[nextIndex]));
  };

  const handleKeyDown = (event) => {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
    event.preventDefault();
    selectSibling(event.key === 'ArrowLeft' ? -1 : 1);
  };

  if (loading) {
    return (
      <div className="vault-rail-state" role="status">
        <Loader2 size={16} className="spin" aria-hidden="true" />
        Recuperando demos procesadas…
      </div>
    );
  }

  if (error) {
    return (
      <div className="vault-rail-state vault-rail-error" role="alert">
        <span>No se pudo abrir el archivo de partidas.</span>
        <button type="button" onClick={onRetry}>
          <RefreshCcw size={14} aria-hidden="true" /> Reintentar
        </button>
      </div>
    );
  }

  if (!matches.length) {
    return (
      <div className="vault-rail-state">
        No hay partidas que coincidan con la búsqueda.
      </div>
    );
  }

  return (
    <div
      className="vault-rail"
      ref={railRef}
      role="group"
      aria-label="Partidas procesadas"
      onKeyDown={handleKeyDown}
    >
      {matches.map((match, index) => {
        const summary = getMatchSummary(match);
        const isSelected = selectedMatchId === summary.id;

        return (
          <button
            key={summary.id || `${summary.rawMapName}-${index}`}
            type="button"
            aria-pressed={isSelected}
            aria-current={isSelected ? 'true' : undefined}
            className={`vault-match-item ${isSelected ? 'is-selected' : ''}`}
            onClick={() => onSelectMatch(summary.id)}
          >
            <span className="vault-item-index">{String(index + 1).padStart(2, '0')}</span>
            <span className={`vault-item-result ${summary.isWin ? 'win' : 'loss'}`} aria-hidden="true" />
            <span className="vault-item-identity">
              <strong>{summary.mapName}</strong>
              <small>{summary.dateLabel}</small>
            </span>
            <span className={`vault-item-score ${summary.isWin ? 'win' : 'loss'}`}>{summary.score}</span>
          </button>
        );
      })}
    </div>
  );
});

MatchArchiveRail.displayName = 'MatchArchiveRail';

export default MatchArchiveRail;
