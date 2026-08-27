import React, { useEffect, useMemo, useState } from 'react';
import { Check, LoaderCircle, Search, ShieldCheck, UserRoundSearch, X } from 'lucide-react';
import { motion } from 'framer-motion';
import { usePerformancePlayers } from '../hooks/usePerformanceData';

const ACCENTS = ['#ffb66e', '#b899ff', '#ff7f91', '#5cf2cd', '#75dced'];

const decoratePlayer = (player) => {
  const accentIndex = [...String(player.steam_id)].reduce(
    (sum, digit) => sum + Number(digit || 0),
    0,
  ) % ACCENTS.length;
  return {
    ...player,
    id: String(player.steam_id),
    initials: String(player.name || 'JG').slice(0, 2).toUpperCase(),
    accent: ACCENTS[accentIndex],
  };
};

const PlayerComparisonPicker = ({
  activePlayerId,
  currentSteamId,
  onSelect,
  onClose,
}) => {
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const { data, isFetching, error } = usePerformancePlayers(debouncedQuery);

  useEffect(() => {
    const timeout = window.setTimeout(() => setDebouncedQuery(query), 250);
    return () => window.clearTimeout(timeout);
  }, [query]);

  const players = useMemo(
    () => (data?.players || [])
      .filter((player) => String(player.steam_id) !== String(currentSteamId))
      .map(decoratePlayer),
    [currentSteamId, data],
  );

  return (
    <motion.section
      id="comparison-picker"
      className="pf3-player-picker"
      initial={{ opacity: 0, x: '-50%', y: -14, scale: 0.97 }}
      animate={{ opacity: 1, x: '-50%', y: 0, scale: 1 }}
      exit={{ opacity: 0, x: '-50%', y: -8, scale: 0.98 }}
      transition={{ duration: 0.24, ease: [0.22, 1, 0.36, 1] }}
      role="dialog"
      aria-modal="false"
      aria-labelledby="comparison-picker-title"
    >
      <header>
        <div>
          <span><UserRoundSearch size={15} aria-hidden="true" /> Comparación competitiva</span>
          <h2 id="comparison-picker-title">Elige una referencia real</h2>
          <p>Jugadores presentes en tus partidas procesadas</p>
        </div>
        <button type="button" onClick={onClose} aria-label="Cerrar selector">
          <X size={18} aria-hidden="true" />
        </button>
      </header>

      <label className="pf3-player-search">
        <Search size={17} aria-hidden="true" />
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Buscar por nombre"
          autoFocus
        />
        {isFetching && <LoaderCircle size={16} className="pf3-spin" aria-label="Buscando" />}
      </label>

      <div className="pf3-player-options">
        {players.map((player) => {
          const selected = player.id === String(activePlayerId);
          return (
            <button
              type="button"
              key={player.id}
              className={selected ? 'is-selected' : ''}
              onClick={() => onSelect(player)}
            >
              <span
                className="pf3-player-avatar"
                style={{ '--player-accent': player.accent }}
                aria-hidden="true"
              >
                {player.initials}
              </span>
              <span className="pf3-player-option-copy">
                <strong>{player.name}</strong>
                <small>Jugador detectado en tus demos</small>
              </span>
              <span className="pf3-player-option-stats">
                <strong>{player.matches}</strong>
                <small>partidas registradas</small>
              </span>
              {selected && <Check size={17} aria-label="Seleccionado" />}
            </button>
          );
        })}
        {!isFetching && !error && players.length === 0 && (
          <p className="pf3-player-empty">No hay jugadores que coincidan con la búsqueda.</p>
        )}
        {error && (
          <p className="pf3-player-empty">No se pudo consultar el índice de jugadores.</p>
        )}
      </div>

      <footer>
        <ShieldCheck size={15} aria-hidden="true" />
        Solo se comparan datos procesados por Go. Las muestras pequeñas se indican en pantalla.
      </footer>
    </motion.section>
  );
};

export default PlayerComparisonPicker;
