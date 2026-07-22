import React, { useEffect, useRef } from 'react';
import {
  CalendarDays,
  Map,
  RotateCcw,
  Search,
  SlidersHorizontal,
  Trophy
} from 'lucide-react';

const MatchArchiveFilters = ({
  query,
  mapFilter,
  resultFilter,
  dateFilter,
  sortBy,
  availableMaps,
  visibleCount,
  totalCount,
  hasActiveFilters,
  onQueryChange,
  onMapFilterChange,
  onResultFilterChange,
  onDateFilterChange,
  onSortChange,
  onResetFilters
}) => {
  const inputRef = useRef(null);

  useEffect(() => {
    const focusSearch = (event) => {
      if (!(event.metaKey || event.ctrlKey) || event.key.toLowerCase() !== 'k') return;
      event.preventDefault();
      inputRef.current?.focus();
    };

    window.addEventListener('keydown', focusSearch);
    return () => window.removeEventListener('keydown', focusSearch);
  }, []);

  return (
    <section className="vault-filter-panel vault-filter-panel-inline" aria-labelledby="vault-filter-title">
      <div className="vault-filter-heading">
        <div>
          <span>Explorador de demos</span>
          <strong id="vault-filter-title" className="vault-filter-title">Encuentra tu partida</strong>
          <small aria-live="polite">
            {visibleCount} de {totalCount} partidas visibles
          </small>
        </div>
        <kbd>⌘ K</kbd>
      </div>

      <label className="vault-search-field">
        <span className="vault-search-icon"><Search size={19} aria-hidden="true" /></span>
        <input
          ref={inputRef}
          type="search"
          name="match-search"
          aria-label="Buscar partidas"
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          placeholder="Mapa, marcador, fecha o ID…"
          autoComplete="off"
        />
      </label>

      <div className="vault-filter-grid">
        <label className="vault-filter-field">
          <span className="vault-filter-icon"><Map size={15} aria-hidden="true" /></span>
          <span className="vault-filter-copy">
            <small>Mapa</small>
            <select
              name="match-map"
              aria-label="Filtrar por mapa"
              value={mapFilter}
              onChange={(event) => onMapFilterChange(event.target.value)}
            >
              <option value="all">Todos los mapas</option>
              {availableMaps.map((mapName) => (
                <option key={mapName} value={mapName}>{mapName.replace('de_', '').toUpperCase()}</option>
              ))}
            </select>
          </span>
        </label>

        <label className="vault-filter-field">
          <span className="vault-filter-icon"><Trophy size={15} aria-hidden="true" /></span>
          <span className="vault-filter-copy">
            <small>Resultado</small>
            <select
              name="match-result"
              aria-label="Filtrar por resultado"
              value={resultFilter}
              onChange={(event) => onResultFilterChange(event.target.value)}
            >
              <option value="all">Todos</option>
              <option value="win">Victorias</option>
              <option value="loss">Derrotas</option>
            </select>
          </span>
        </label>

        <label className="vault-filter-field">
          <span className="vault-filter-icon"><CalendarDays size={15} aria-hidden="true" /></span>
          <span className="vault-filter-copy">
            <small>Periodo</small>
            <select
              name="match-date"
              aria-label="Filtrar por fecha"
              value={dateFilter}
              onChange={(event) => onDateFilterChange(event.target.value)}
            >
              <option value="all">Todo el historial</option>
              <option value="30">Últimos 30 días</option>
              <option value="90">Últimos 90 días</option>
              <option value="year">Este año</option>
            </select>
          </span>
        </label>

        <label className="vault-filter-field">
          <span className="vault-filter-icon"><SlidersHorizontal size={15} aria-hidden="true" /></span>
          <span className="vault-filter-copy">
            <small>Orden</small>
            <select
              name="match-sort"
              aria-label="Ordenar partidas"
              value={sortBy}
              onChange={(event) => onSortChange(event.target.value)}
            >
              <option value="date_desc">Más reciente</option>
              <option value="date_asc">Más antigua</option>
              <option value="map_asc">Mapa A–Z</option>
              <option value="map_desc">Mapa Z–A</option>
            </select>
          </span>
        </label>
      </div>

      <div className="vault-filter-footer">
        <span><i aria-hidden="true" /> El archivo se actualiza al instante</span>
        {hasActiveFilters ? (
          <button type="button" onClick={onResetFilters}>
            <RotateCcw size={14} aria-hidden="true" /> Limpiar filtros
          </button>
        ) : null}
      </div>
    </section>
  );
};

export default MatchArchiveFilters;
