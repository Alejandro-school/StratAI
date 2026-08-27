import React from 'react';
import { Eye, MapPinned, Route, SlidersHorizontal } from 'lucide-react';
import { formatCalloutName } from '../../../components/TacticalMap/tacticalPresentation';

const integerFormatter = new Intl.NumberFormat('es-ES', { maximumFractionDigits: 0 });
const percentFormatter = new Intl.NumberFormat('es-ES', { maximumFractionDigits: 1 });

const TacticalPresencePanel = ({
  metrics = {},
  matchesAnalyzed = 0,
  activeSide = 'ct',
  sideDistribution = { ct: 50, t: 50 },
  heatmapIntensity = 70,
  showHeatmap = true,
  showRoutes = false,
  routeCount = 0,
  onIntensityChange,
  onToggleHeatmap,
  onToggleRoutes,
}) => {
  const {
    top_positions: topPositions = [],
    total_rounds: totalRounds = 0,
    total_samples: totalSamples = 0,
  } = metrics;
  const positions = topPositions;
  const hasData = totalRounds > 0 || totalSamples > 0 || positions.length > 0;

  return (
    <section className="tactical-panel tactical-presence-panel" aria-labelledby="presence-panel-title">
      <header className="tactical-panel__header">
        <div>
          <span className="tactical-eyebrow">Lectura espacial · {activeSide.toUpperCase()}</span>
          <h2 id="presence-panel-title">Posicionamiento</h2>
        </div>
        <div className="tactical-sample-chip">
          <strong>{matchesAnalyzed}</strong>
          <span>{matchesAnalyzed === 1 ? 'demo' : 'demos'}</span>
        </div>
      </header>

      <p className="tactical-panel__lead">
        Dónde permaneces con vida, qué transiciones repites y cómo cambia tu distribución entre CT y T.
      </p>

      <div className="tactical-presence-metrics" aria-label="Cobertura de la muestra">
        <div>
          <strong>{integerFormatter.format(totalRounds)}</strong>
          <span>rondas</span>
        </div>
        <div>
          <strong>{integerFormatter.format(totalSamples)}</strong>
          <span>muestras</span>
        </div>
      </div>

      <section className="tactical-side-distribution" aria-labelledby="side-distribution-title">
        <div className="tactical-ranking__heading">
          <h3 id="side-distribution-title">Distribución CT / T</h3>
          <span>Comparativa del mapa</span>
        </div>
        <div className="tactical-side-distribution__values">
          <span><strong>{percentFormatter.format(sideDistribution.ct)}%</strong> CT</span>
          <span><strong>{percentFormatter.format(sideDistribution.t)}%</strong> T</span>
        </div>
        <div className="tactical-side-distribution__bar" aria-hidden="true">
          <span style={{ width: `${sideDistribution.ct}%` }} />
        </div>
      </section>

      <div className="tactical-layer-controls">
        <h3>Capas visuales</h3>
        <label className="tactical-range-control" htmlFor="tactical-heat-intensity">
          <span>
            <SlidersHorizontal size={15} aria-hidden="true" />
            Intensidad Visual
          </span>
          <output htmlFor="tactical-heat-intensity">{heatmapIntensity}%</output>
          <input
            id="tactical-heat-intensity"
            name="tactical-heat-intensity"
            type="range"
            min="25"
            max="100"
            value={heatmapIntensity}
            onChange={(event) => onIntensityChange(Number(event.target.value))}
          />
        </label>

        <div className="tactical-toggle-row">
          <button
            type="button"
            aria-pressed={showHeatmap}
            className={showHeatmap ? 'is-active' : ''}
            onClick={onToggleHeatmap}
          >
            <Eye size={15} aria-hidden="true" />
            Densidad
          </button>
          <button
            type="button"
            aria-pressed={showRoutes}
            className={showRoutes ? 'is-active' : ''}
            onClick={onToggleRoutes}
            disabled={routeCount === 0}
          >
            <Route size={15} aria-hidden="true" />
            Transiciones
            {routeCount > 0 ? <span>{routeCount}</span> : null}
          </button>
        </div>
      </div>

      {!hasData ? (
        <div className="tactical-empty-state">
          <MapPinned size={28} aria-hidden="true" />
          <h3>Aún no hay recorrido suficiente</h3>
          <p>Analiza una demo de este mapa para empezar a construir tu huella.</p>
        </div>
      ) : (
        <div className="tactical-ranking">
          <div className="tactical-ranking__heading">
            <h3>Conclusiones · zonas habituales</h3>
            <span>{activeSide.toUpperCase()} · Por zonas</span>
          </div>
          <ol>
            {positions.slice(0, 6).map((position) => (
              <li key={position.area}>
                <span className="tactical-ranking__name">{formatCalloutName(position.area)}</span>
                <span className="tactical-ranking__bar" aria-hidden="true">
                  <span style={{ width: `${Math.min(100, position.time_percent ?? 0)}%` }} />
                </span>
                <strong>{percentFormatter.format(position.time_percent ?? 0)}%</strong>
              </li>
            ))}
          </ol>
        </div>
      )}
    </section>
  );
};

export default React.memo(TacticalPresencePanel);
