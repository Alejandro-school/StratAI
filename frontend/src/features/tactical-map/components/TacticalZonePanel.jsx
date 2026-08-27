import React, { useMemo } from 'react';
import { AlertTriangle, Crosshair, ShieldCheck, Sparkles, X } from 'lucide-react';
import { formatCalloutName } from '../../../components/TacticalMap/tacticalPresentation';

const numberFormatter = new Intl.NumberFormat('es-ES', { maximumFractionDigits: 1 });

const CONFIDENCE_COPY = {
  high: 'Confianza alta',
  medium: 'Confianza media',
  low: 'Señal inicial',
};

const COMBAT_METRICS = {
  volume: {
    label: 'Volumen',
    description: 'El tamaño representa duelos observados; el color compara la eficiencia con tu media.',
    score: (callout) => callout.sample_size ?? 0,
    value: (callout) => callout.sample_size ?? 0,
  },
  efficiency: {
    label: 'Eficiencia',
    description: 'Porcentaje de duelos ajustado hacia tu media para limitar conclusiones por muestras pequeñas.',
    score: (callout, baseline) => Math.abs((callout.adjustedWinRate ?? 50) - baseline),
    value: (callout) => `${Math.round(callout.adjustedWinRate ?? callout.win_rate ?? 50)}%`,
  },
  impact: {
    label: 'Impacto',
    description: 'Índice 0–100 que combina duelos, aperturas y trades; siempre acompañado por su muestra.',
    score: (callout) => callout.impactScore ?? 0,
    value: (callout) => Math.round(callout.impactScore ?? 0),
  },
  risk: {
    label: 'Riesgo',
    description: 'Índice 0–100 de ventaja cedida. Una señal inicial nunca se presenta como certeza.',
    score: (callout) => callout.riskScore ?? 0,
    value: (callout) => Math.round(callout.riskScore ?? 0),
  },
};

const getRecommendation = (callout) => {
  if (callout.context_scope === 'all-sides-only') {
    return (callout.adjustedWinRate ?? 50) < 47
      ? 'Revisa el primer ángulo y define una ruta de salida antes de volver a disputar esta zona.'
      : 'Mantén esta lectura como hipótesis y suma más demos con el bando seleccionado.';
  }
  const context = callout.context_stats ?? {};
  const openingLosses = Math.max(0, (context.opening_attempts ?? 0) - (context.opening_kills ?? 0));

  if ((callout.flash_death_pct ?? 0) >= 25) {
    return 'Espera la utilidad rival o solicita una contra-flash antes de entrar en esta zona.';
  }
  if (openingLosses > (context.opening_kills ?? 0)) {
    return 'Reduce primeros contactos sin apoyo y prepara una ruta de salida antes de disputar.';
  }
  if ((context.trade_deaths ?? 0) > (context.trade_kills ?? 0)) {
    return 'Acorta la distancia con tu compañero para mejorar la conversión de trades.';
  }
  if ((callout.impactScore ?? 0) >= 65) {
    return 'Conserva el primer ángulo y el apoyo que ya convierten esta zona en ventaja.';
  }
  return 'Suma más encuentros antes de cambiar el plan; la señal todavía es moderada.';
};

const Metric = ({ label, value, hint }) => (
  <div className="tactical-zone-metric">
    <span>{label}</span>
    <strong>{value}</strong>
    {hint ? <small>{hint}</small> : null}
  </div>
);

const MetricSelector = ({ metric, onMetricChange }) => (
  <div className="tactical-combat-metrics" role="group" aria-label="Métrica de combate">
    {Object.entries(COMBAT_METRICS).map(([id, config]) => (
      <button
        key={id}
        type="button"
        aria-pressed={metric === id}
        className={metric === id ? 'is-active' : ''}
        onClick={() => onMetricChange(id)}
      >
        {config.label}
      </button>
    ))}
  </div>
);

const TacticalZonePanel = ({
  metric = 'volume',
  activeSide = 'ct',
  callouts = [],
  selectedCallout = null,
  baselineWinRate = 50,
  matchesAnalyzed = 0,
  onMetricChange,
  onSelect,
  onClose,
}) => {
  const metricConfig = COMBAT_METRICS[metric] ?? COMBAT_METRICS.volume;
  const candidates = useMemo(() => [...callouts]
    .sort((a, b) => metricConfig.score(b, baselineWinRate) - metricConfig.score(a, baselineWinRate))
    .slice(0, 5), [baselineWinRate, callouts, metricConfig]);

  if (!selectedCallout) {
    return (
      <section className="tactical-panel tactical-zone-panel" aria-labelledby="zone-panel-title">
        <header className="tactical-panel__header">
          <div>
            <span className="tactical-eyebrow">Combate por zonas · {activeSide.toUpperCase()}</span>
            <h2 id="zone-panel-title">Dónde Peleas</h2>
          </div>
          <div className="tactical-sample-chip">
            <strong>{matchesAnalyzed}</strong>
            <span>{matchesAnalyzed === 1 ? 'demo' : 'demos'}</span>
          </div>
        </header>

        <MetricSelector metric={metric} onMetricChange={onMetricChange} />
        <p className="tactical-panel__lead">{metricConfig.description}</p>
        <div className="tactical-baseline">
          <span>Referencia del mapa</span>
          <strong>{numberFormatter.format(baselineWinRate)}%</strong>
        </div>

        {candidates.length === 0 ? (
          <div className="tactical-empty-state">
            <Crosshair size={28} aria-hidden="true" />
            <h3>Sin duelos registrados</h3>
            <p>Cuando haya combates en este mapa, podrás comparar sus zonas.</p>
          </div>
        ) : (
          <div className="tactical-zone-shortlist">
            <h3>Zonas · {metricConfig.label} · {activeSide.toUpperCase()}</h3>
            {candidates.map((callout) => (
              <button key={callout.name} type="button" onClick={() => onSelect(callout)}>
                <span>
                  <strong>{formatCalloutName(callout.name)}</strong>
                  <small>{callout.sample_size ?? 0} duelos · {CONFIDENCE_COPY[callout.confidenceLabel] ?? 'Señal inicial'}</small>
                </span>
                <b>{metricConfig.value(callout)}</b>
              </button>
            ))}
          </div>
        )}
      </section>
    );
  }

  const context = selectedCallout.context_stats ?? {};
  const hasSideContext = selectedCallout.context_scope !== 'all-sides-only';
  const adjustedRate = selectedCallout.adjustedWinRate ?? selectedCallout.win_rate ?? 50;
  const rawRate = selectedCallout.win_rate ?? 50;
  const openingLosses = Math.max(0, (context.opening_attempts ?? 0) - (context.opening_kills ?? 0));

  return (
    <section className="tactical-panel tactical-zone-panel is-detail" aria-labelledby="selected-zone-title">
      <header className="tactical-panel__header tactical-zone-detail__header">
        <div>
          <span className="tactical-eyebrow">Detalle de combate</span>
          <h2 id="selected-zone-title">{formatCalloutName(selectedCallout.name)}</h2>
        </div>
        <button type="button" className="tactical-icon-button" onClick={onClose} aria-label="Cerrar detalle de zona">
          <X size={17} aria-hidden="true" />
        </button>
      </header>

      <MetricSelector metric={metric} onMetricChange={onMetricChange} />
      <div className={`tactical-confidence-card confidence-${selectedCallout.confidenceLabel ?? 'low'}`}>
        {(selectedCallout.confidenceLabel ?? 'low') === 'high'
          ? <ShieldCheck size={18} aria-hidden="true" />
          : <AlertTriangle size={18} aria-hidden="true" />}
        <div>
          <strong>{CONFIDENCE_COPY[selectedCallout.confidenceLabel] ?? 'Señal inicial'}</strong>
          <span>{selectedCallout.sample_size ?? 0} duelos · {matchesAnalyzed} {matchesAnalyzed === 1 ? 'demo' : 'demos'}</span>
        </div>
      </div>

      <div className="tactical-zone-scoreboard">
        <Metric label="Eficiencia ajustada" value={`${numberFormatter.format(adjustedRate)}%`} hint={`Observado: ${numberFormatter.format(rawRate)}%`} />
        <Metric label="K/D" value={numberFormatter.format(Number(selectedCallout.kd ?? 0))} hint={`${selectedCallout.kills ?? 0} / ${selectedCallout.deaths ?? 0}`} />
        <Metric label="Impacto" value={Math.round(selectedCallout.impactScore ?? 0)} hint="índice 0–100" />
        <Metric label="Riesgo" value={Math.round(selectedCallout.riskScore ?? 0)} hint="índice 0–100" />
      </div>

      <div className="tactical-context-grid">
        <Metric label="Openings" value={hasSideContext ? `${context.opening_kills ?? 0}–${openingLosses}` : '—'} />
        <Metric label="Trades" value={hasSideContext ? `${context.trade_kills ?? 0}–${context.trade_deaths ?? 0}` : '—'} />
        <Metric label="Muertes cegado" value={hasSideContext ? `${numberFormatter.format(selectedCallout.flash_death_pct ?? 0)}%` : '—'} />
        <Metric label="Cuota de duelos" value={`${numberFormatter.format(selectedCallout.volumeShare ?? 0)}%`} />
      </div>

      <div className="tactical-action-note">
        <Sparkles size={17} aria-hidden="true" />
        <div>
          <strong>Siguiente ajuste</strong>
          <p>{getRecommendation(selectedCallout)}</p>
        </div>
      </div>

      {Math.abs(rawRate - adjustedRate) >= 1 ? (
        <p className="tactical-method-note">
          El {numberFormatter.format(rawRate)}% observado se ajusta hacia tu media de {numberFormatter.format(baselineWinRate)}% para no sobrerrepresentar pocos duelos.
        </p>
      ) : null}
      {!hasSideContext ? (
        <p className="tactical-method-note">Con CT/T activo se omite el contexto combinado que no puede atribuirse al bando correcto.</p>
      ) : null}
    </section>
  );
};

export default React.memo(TacticalZonePanel);
