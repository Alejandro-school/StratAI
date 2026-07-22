import React, { memo } from 'react';
import { ArrowUpRight, Check, Crosshair, Loader2, RefreshCcw } from 'lucide-react';
import { getMatchSummary } from './matchPresentation';

const MatchFocusPanel = memo(({ match, loading, error, isAnalyzing, onAnalyze, onRetry }) => {
  const summary = getMatchSummary(match);

  if (loading) {
    return (
      <section className="vault-focus vault-focus-loading" aria-live="polite" aria-busy="true">
        <span className="vault-eyebrow">Abriendo archivo táctico</span>
        <div className="vault-title-skeleton" />
        <div className="vault-score-skeleton" />
      </section>
    );
  }

  if (error) {
    return (
      <section className="vault-focus vault-focus-empty" role="alert">
        <span className="vault-eyebrow">Conexión interrumpida</span>
        <h1>No pudimos recuperar tus partidas.</h1>
        <p>Comprueba la conexión y vuelve a abrir el archivo.</p>
        <button type="button" className="vault-retry-action" onClick={onRetry}>
          <RefreshCcw size={15} aria-hidden="true" /> Reintentar
        </button>
      </section>
    );
  }

  if (!summary) {
    return (
      <section className="vault-focus vault-focus-empty" aria-live="polite">
        <span className="vault-eyebrow">Archivo táctico</span>
        <h1>Sin partidas en el radar.</h1>
        <p>Cuando una demo termine de procesarse aparecerá aquí, lista para revisar.</p>
      </section>
    );
  }

  return (
    <section className="vault-focus" aria-labelledby="vault-match-title" aria-live="polite">
      <div className="vault-focus-copy" key={summary.id}>
        <div className="vault-status-row">
          <span className="vault-eyebrow">Dossier de partida</span>
          <span className="vault-ready"><Check size={13} aria-hidden="true" /> Lista</span>
        </div>

        <div className="vault-title-lockup">
          <span className="vault-section-label">Mapa analizado</span>
          <h1 id="vault-match-title">{summary.mapName}</h1>
          <p className="vault-date">{summary.dateLabel}</p>
        </div>

        <div className={`vault-score-card ${summary.isWin ? 'is-win' : 'is-loss'}`}>
          <div>
            <span className="vault-section-label">Marcador final</span>
            <div className="vault-score" aria-label={`Marcador ${summary.teamScore} a ${summary.opponentScore}`}>
              <strong>{summary.teamScore}</strong>
              <span>:</span>
              <strong>{summary.opponentScore}</strong>
            </div>
          </div>
          <div className={`vault-result vault-result-${summary.isWin ? 'win' : 'loss'}`}>
            <span aria-hidden="true" />
            {summary.isWin ? 'Victoria' : 'Derrota'}
          </div>
        </div>

        <dl className="vault-match-meta">
          <div>
            <dt>Lado</dt>
            <dd>{summary.teamLabel}</dd>
          </div>
          <div>
            <dt>Rondas</dt>
            <dd>{summary.totalRounds}</dd>
          </div>
          <div>
            <dt>Estado</dt>
            <dd className="vault-meta-ready">Procesada</dd>
          </div>
        </dl>
      </div>

      <button
        type="button"
        className="vault-primary-action"
        onClick={onAnalyze}
        disabled={isAnalyzing}
      >
        <span className="vault-action-icon">
          {isAnalyzing
            ? <Loader2 size={18} className="spin" aria-hidden="true" />
            : <Crosshair size={18} aria-hidden="true" />}
        </span>
        <span>
          <strong>{isAnalyzing ? 'Preparando revisión' : 'Iniciar revisión con IA'}</strong>
          <small>Replay y evidencias sincronizadas</small>
        </span>
        <ArrowUpRight size={18} aria-hidden="true" />
      </button>
    </section>
  );
});

MatchFocusPanel.displayName = 'MatchFocusPanel';

export default MatchFocusPanel;
