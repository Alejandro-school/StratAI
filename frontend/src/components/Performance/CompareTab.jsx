import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Gauge, Info, Search, Target, User, X } from 'lucide-react';
import { API_URL } from '../../utils/api';
import { formatDecimal, formatInteger, formatPercent } from '../../utils/performanceFormatters';
import SectionBlock from './SectionBlock';
import { BriefingPanel, InsightCard } from './PerformanceBriefing';

/* ── Premier Rating tiers ───────────────────────────────────────────── */

const PREMIER_TIERS = [
  { id: '5k',  label: '5,000',  short: '5K',  rating: 5000 },
  { id: '10k', label: '10,000', short: '10K', rating: 10000 },
  { id: '15k', label: '15,000', short: '15K', rating: 15000 },
  { id: '20k', label: '20,000', short: '20K', rating: 20000 },
  { id: '25k', label: '25,000', short: '25K', rating: 25000 },
  { id: '30k', label: '30,000', short: '30K', rating: 30000 },
];

const PREMIER_BENCHMARKS = {
  hltv_rating:     { '5k': 0.68, '10k': 0.85, '15k': 1.0,  '20k': 1.10, '25k': 1.20, '30k': 1.35 },
  kd_ratio:        { '5k': 0.72, '10k': 0.90, '15k': 1.04, '20k': 1.14, '25k': 1.24, '30k': 1.40 },
  adr:             { '5k': 52,   '10k': 66,   '15k': 76,   '20k': 84,   '25k': 90,   '30k': 98 },
  hs_pct:          { '5k': 28,   '10k': 36,   '15k': 44,   '20k': 49,   '25k': 53,   '30k': 58 },
  win_rate:        { '5k': 40,   '10k': 46,   '15k': 50,   '20k': 52,   '25k': 55,   '30k': 60 },
  kast:            { '5k': 55,   '10k': 64,   '15k': 70,   '20k': 75,   '25k': 79,   '30k': 85 },
  accuracy:        { '5k': 13,   '10k': 18,   '15k': 23,   '20k': 27,   '25k': 31,   '30k': 38 },
  opening_success: { '5k': 36,   '10k': 43,   '15k': 50,   '20k': 54,   '25k': 58,   '30k': 64 },
};

const METRIC_DEFS = [
  { key: 'hltv_rating', label: 'Rating HLTV', max: 1.8, format: (v) => formatDecimal(v, 2) },
  { key: 'kd_ratio',    label: 'Relación K/D', max: 1.8, format: (v) => formatDecimal(v, 2) },
  { key: 'adr',         label: 'ADR',          max: 120, format: (v) => formatDecimal(v, 1) },
  { key: 'hs_pct',      label: '% a la cabeza', max: 70,  format: (v) => formatPercent(v) },
  { key: 'win_rate',    label: '% de victorias', max: 75, format: (v) => formatPercent(v) },
  { key: 'kast',        label: 'KAST',         max: 100, format: (v) => formatPercent(v) },
  { key: 'accuracy',    label: 'Precisión',    max: 50,  format: (v) => formatPercent(v) },
  { key: 'opening_success', label: '% de aperturas', max: 80, format: (v) => formatPercent(v) },
];

/* ── Extract metric values from a performance payload ────────────────── */

const extractValues = (overview, aim, combat) => ({
  hltv_rating: Number(overview?.hltv_rating || 0),
  kd_ratio: Number(overview?.kd_ratio || 0),
  adr: Number(overview?.adr || 0),
  hs_pct: Number(overview?.hs_pct || 0),
  win_rate: Number(overview?.win_rate || 0),
  kast: Number(overview?.kast || 0),
  accuracy: Number(aim?.accuracy_overall || 0),
  opening_success: Number(combat?.opening_success_rate || 0),
});

/* ── Comparison summary calculator ───────────────────────────────────── */

const calcSummary = (playerVals, compareVals) => {
  let above = 0, below = 0, even = 0;
  for (const def of METRIC_DEFS) {
    const pv = playerVals[def.key];
    const cv = compareVals[def.key];
    const threshold = Math.abs(cv) * 0.04;
    if (pv > cv + threshold) above++;
    else if (pv < cv - threshold) below++;
    else even++;
  }
  return { above, below, even };
};

/* ── Player search hook ──────────────────────────────────────────────── */

const usePlayerSearch = () => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const debounceRef = useRef(null);

  const search = useCallback((q) => {
    setQuery(q);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (!q.trim() && q !== '') { setResults([]); return; }

    debounceRef.current = setTimeout(async () => {
      setLoading(true);
      try {
        const encoded = encodeURIComponent(q.trim());
        const res = await fetch(`${API_URL}/steam/player-search?q=${encoded}`, { credentials: 'include' });
        if (res.ok) {
          const data = await res.json();
          setResults(data.players || []);
        }
      } catch { /* ignore */ }
      setLoading(false);
    }, 300);
  }, []);

  return { query, search, results, loading };
};

/* ── Fetch opponent stats ────────────────────────────────────────────── */

const useOpponentStats = () => {
  const [opponent, setOpponent] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const loadPlayer = useCallback(async (steamId) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_URL}/steam/player-stats/${steamId}`, { credentials: 'include' });
      if (!res.ok) throw new Error('No se pudo cargar al jugador');
      const data = await res.json();
      if (data.error) { setError(data.error); setOpponent(null); }
      else setOpponent(data);
    } catch (err) {
      setError(err.message);
      setOpponent(null);
    }
    setLoading(false);
  }, []);

  const clear = useCallback(() => { setOpponent(null); setError(null); }, []);

  return { opponent, loading, error, loadPlayer, clear };
};

/* ── Comparison bar row ──────────────────────────────────────────────── */

const CompareRow = ({ def, yours, theirs, theirLabel }) => {
  const yoursWidth = Math.min((yours / def.max) * 100, 100);
  const theirsPos = Math.min((theirs / def.max) * 100, 100);
  return (
    <div className="p-compare-row">
      <span className="p-compare-row-label">{def.label}</span>
      <div className="p-compare-bar-container">
        <div className="p-compare-bar-you" style={{ width: `${yoursWidth}%` }} />
        <div className="p-compare-bar-marker rank-marker" style={{ left: `${theirsPos}%` }} data-label={theirLabel} />
      </div>
      <span className="p-compare-you-val">{def.format(yours)}</span>
      <span className="p-compare-rank-val">{def.format(theirs)}</span>
    </div>
  );
};

/* ── Main component ──────────────────────────────────────────────────── */

const CompareTab = ({ overview = {}, aim = {}, combat = {} }) => {
  const [mode, setMode] = useState('rank');
  const [selectedTier, setSelectedTier] = useState('15k');
  const [selectedPlayer, setSelectedPlayer] = useState(null);
  const [showSearch, setShowSearch] = useState(false);

  const { query, search, results, loading: searchLoading } = usePlayerSearch();
  const { opponent, loading: playerLoading, error: playerError, loadPlayer, clear: clearOpponent } = useOpponentStats();

  const playerValues = useMemo(() => extractValues(overview, aim, combat), [overview, aim, combat]);

  const rankComparison = useMemo(() => {
    const data = {};
    for (const key of Object.keys(PREMIER_BENCHMARKS)) data[key] = PREMIER_BENCHMARKS[key][selectedTier] || 0;
    return data;
  }, [selectedTier]);

  const opponentValues = useMemo(() => {
    if (!opponent) return null;
    return extractValues(opponent.overview, opponent.aim, opponent.combat);
  }, [opponent]);

  const compareTarget = mode === 'player' && opponentValues ? opponentValues : rankComparison;
  const selectedTierObj = PREMIER_TIERS.find((t) => t.id === selectedTier);
  const compareLabel = mode === 'player' && selectedPlayer
    ? selectedPlayer.name
    : selectedTierObj?.short || '';

  const summary = useMemo(() => calcSummary(playerValues, compareTarget), [playerValues, compareTarget]);

  const handleSelectPlayer = (player) => {
    setSelectedPlayer(player);
    setShowSearch(false);
    search('');
    loadPlayer(player.steam_id);
  };

  const handleClearPlayer = () => {
    setSelectedPlayer(null);
    clearOpponent();
    setMode('rank');
  };

  useEffect(() => {
    if (showSearch && results.length === 0 && !query) search('');
  }, [showSearch, results.length, query, search]);

  return (
    <div className="p-section">
      <BriefingPanel eyebrow="Briefing comparativo" title="Benchmarks para calibrar tu nivel real">
        <div className="p-briefing-grid-3">
          <InsightCard
            tone={summary.above >= summary.below ? 'good' : 'warning'}
            icon={Target}
            label="Por encima"
            value={summary.above}
            detail="metricas donde superas la referencia"
          />
          <InsightCard
            tone="neutral"
            icon={Gauge}
            label="Referencia"
            value={compareLabel}
            detail={mode === 'player' ? 'jugador seleccionado' : 'rating Premier'}
          />
          <InsightCard
            tone={summary.below > summary.above ? 'danger' : 'neutral'}
            icon={Info}
            label="Brechas"
            value={summary.below}
            detail="metricas que requieren ajuste"
          />
        </div>
      </BriefingPanel>

      <div className="p-compare-intro">
        <Info className="p-compare-intro-icon" size={18} />
        <p className="p-compare-intro-text">
          Compara tu rendimiento con los <strong>benchmarks de Premier Rating</strong> o
          busca un jugador de tus partidas para un <strong>cara a cara</strong> directo.
        </p>
      </div>

      {/* Mode toggle */}
      <div className="p-compare-mode-row">
        <button type="button" className={`p-rank-btn ${mode === 'rank' ? 'active' : ''}`} onClick={() => setMode('rank')}>
          Por rating Premier
        </button>
        <button
          type="button"
          className={`p-rank-btn ${mode === 'player' ? 'active' : ''}`}
          onClick={() => { setMode('player'); setShowSearch(true); }}
        >
          <Search size={13} /> Buscar jugador
        </button>
      </div>

      {/* Rank selector */}
      {mode === 'rank' && (
        <div className="p-rank-selector" role="radiogroup" aria-label="Seleccionar rango Premier">
          {PREMIER_TIERS.map((tier) => (
            <button
              key={tier.id}
              type="button"
              role="radio"
              aria-checked={selectedTier === tier.id}
              className={`p-rank-btn ${selectedTier === tier.id ? 'active' : ''}`}
              onClick={() => setSelectedTier(tier.id)}
            >
              {tier.label}
            </button>
          ))}
        </div>
      )}

      {/* Player search */}
      {mode === 'player' && (
        <div className="p-player-search-area">
          {selectedPlayer && (
            <div className="p-selected-player">
              <User size={14} />
              <span className="p-selected-player-name">{selectedPlayer.name}</span>
              <span className="p-selected-player-matches">{selectedPlayer.matches} partidas</span>
              <button type="button" className="p-selected-player-clear" onClick={handleClearPlayer}><X size={13} /></button>
            </div>
          )}

          {showSearch && (
            <div className="p-search-dropdown">
              <div className="p-search-input-wrap">
                <Search size={14} className="p-search-input-icon" />
                <input
                  type="text"
                  className="p-search-input"
                  placeholder="Buscar por nombre..."
                  value={query}
                  onChange={(e) => search(e.target.value)}
                  autoFocus
                />
                <button type="button" className="p-search-close" onClick={() => setShowSearch(false)}><X size={14} /></button>
              </div>
              <div className="p-search-results">
                {searchLoading && <div className="p-search-status">Buscando...</div>}
                {!searchLoading && results.length === 0 && query && (
                  <div className="p-search-status">Sin resultados para "{query}"</div>
                )}
                {results.map((p) => (
                  <button key={p.steam_id} type="button" className="p-search-result-row" onClick={() => handleSelectPlayer(p)}>
                    <User size={14} />
                    <span className="p-search-result-name">{p.name}</span>
                    <span className="p-search-result-meta">{p.matches} partidas</span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {!showSearch && !selectedPlayer && (
            <button type="button" className="p-search-trigger" onClick={() => setShowSearch(true)}>
              <Search size={14} /> Buscar jugador para comparar...
            </button>
          )}

          {playerLoading && (
            <div className="p-card" style={{ textAlign: 'center', padding: 20 }}>
              <div className="p-spinner" style={{ margin: '0 auto' }} />
              <p className="p-card-subtitle" style={{ marginTop: 8 }}>Cargando estadísticas...</p>
            </div>
          )}

          {playerError && (
            <div className="p-card" style={{ textAlign: 'center', padding: 20 }}>
              <p className="p-bad-text" style={{ fontSize: 13 }}>{playerError}</p>
            </div>
          )}
        </div>
      )}

      {/* Comparison table */}
      {(mode === 'rank' || (mode === 'player' && opponentValues)) && (
        <>
          <SectionBlock
            title={mode === 'player' && selectedPlayer
              ? `Tu vs ${selectedPlayer.name}`
              : `Tu rendimiento vs Premier ${selectedTierObj?.label || ''}`}
          >
            <div className="p-card">
              <div className="p-compare-header-row">
                <span className="p-compare-header-cell">Métrica</span>
                <span className="p-compare-header-cell">Comparación</span>
                <span className="p-compare-header-cell">Tu</span>
                <span className="p-compare-header-cell">
                  {mode === 'player' && selectedPlayer ? selectedPlayer.name.slice(0, 8) : selectedTierObj?.short}
                </span>
              </div>
              {METRIC_DEFS.map((def) => (
                <CompareRow key={def.key} def={def} yours={playerValues[def.key]} theirs={compareTarget[def.key]} theirLabel={compareLabel} />
              ))}
            </div>
          </SectionBlock>

          <SectionBlock title="Resumen">
            <div className="p-compare-summary">
              <div className="p-compare-verdict">
                <span className={`p-compare-verdict-value ${summary.above > 0 ? 'above' : 'even'}`}>{summary.above}</span>
                <span className="p-compare-verdict-label">Por encima</span>
                <span className="p-compare-verdict-sub">métricas donde superas</span>
              </div>
              <div className="p-compare-verdict">
                <span className={`p-compare-verdict-value ${summary.even > 0 ? 'even' : ''}`}>{summary.even}</span>
                <span className="p-compare-verdict-label">Al nivel</span>
                <span className="p-compare-verdict-sub">métricas equilibradas</span>
              </div>
              <div className="p-compare-verdict">
                <span className={`p-compare-verdict-value ${summary.below > 0 ? 'below' : 'even'}`}>{summary.below}</span>
                <span className="p-compare-verdict-label">Por debajo</span>
                <span className="p-compare-verdict-sub">métricas a mejorar</span>
              </div>
            </div>
          </SectionBlock>

          {mode === 'player' && opponentValues && selectedPlayer && (
            <SectionBlock title="Detalle del oponente">
              <div className="p-card">
                <div className="p-kpi-strip">
                  <div className="p-kpi-cell">
                    <span className="p-kpi-cell-value">{formatInteger(opponent?.overview?.total_matches)}</span>
                    <span className="p-kpi-cell-label">Partidas</span>
                  </div>
                  <div className="p-kpi-cell">
                    <span className="p-kpi-cell-value">{formatInteger(opponent?.overview?.kills)}</span>
                    <span className="p-kpi-cell-label">Bajas totales</span>
                  </div>
                  <div className="p-kpi-cell">
                    <span className="p-kpi-cell-value">{formatDecimal(opponent?.overview?.adr, 1)}</span>
                    <span className="p-kpi-cell-label">ADR</span>
                  </div>
                  <div className="p-kpi-cell">
                    <span className="p-kpi-cell-value">{formatPercent(opponent?.overview?.win_rate)}</span>
                    <span className="p-kpi-cell-label">% de victorias</span>
                  </div>
                </div>
              </div>
            </SectionBlock>
          )}

          <SectionBlock title="Lectura comparativa">
            <div className="p-card">
              <p className="p-insight-copy">
                {mode === 'player' && selectedPlayer
                  ? summary.above >= 6
                    ? `Dominas a ${selectedPlayer.name} en la mayoría de métricas. Eres el jugador más completo en esta comparación.`
                    : summary.above >= 4
                      ? `Ventaja sobre ${selectedPlayer.name} en la mayoría de áreas. Identifica las métricas donde pierdes para cerrar la brecha.`
                      : summary.below >= 5
                        ? `${selectedPlayer.name} te supera en varias métricas. Analiza sus áreas fuertes para encontrar cómo mejorar.`
                        : `Nivel equilibrado con ${selectedPlayer.name}. Las diferencias son menores — la consistencia decidirá quién rinde más.`
                  : summary.above >= 6
                    ? `Estás claramente por encima del nivel ${selectedTierObj?.label}. Sube al próximo tier para ver tu brecha real.`
                    : summary.above >= 4
                      ? `Rendimiento sólido para ${selectedTierObj?.label}. Pocas áreas de mejora para consolidar este nivel.`
                      : summary.below >= 5
                        ? `Necesitas trabajo para alcanzar ${selectedTierObj?.label}. Enfócate en las métricas con mayor diferencia.`
                        : `Equilibrado con ${selectedTierObj?.label}. La consistencia es buena — eleva las 2-3 métricas donde estás por debajo.`}
              </p>
            </div>
          </SectionBlock>
        </>
      )}
    </div>
  );
};

export default CompareTab;
