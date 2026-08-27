import React, { useMemo } from 'react';
import { X } from 'lucide-react';
import GrenadeImage from '../../../components/TacticalMap/GrenadeImage';
import { formatCalloutName } from '../../../components/TacticalMap/tacticalPresentation';
import { filterGrenadeClusters } from '../../../utils/tacticalFilters';

const TYPE_CONFIG = {
  smoke: { label: 'Humo' },
  flash: { label: 'Cegadora' },
  he: { label: 'HE' },
  molotov: { label: 'Molotov' },
};

const numberFormatter = new Intl.NumberFormat('es-ES', { maximumFractionDigits: 1 });

const getQuality = (uses, matches) => {
  const density = matches > 0 ? uses / matches : 0;
  if (uses >= 20 && matches >= 6) return { label: 'Muestra sólida', tone: 'high' };
  if (uses >= 8 && density >= 0.6) return { label: 'Muestra media', tone: 'medium' };
  return { label: 'Muestra inicial', tone: 'low' };
};

const getEffect = (type, stats = {}) => {
  if (type === 'flash') {
    return { value: numberFormatter.format(stats.avgBlinded ?? 0), label: 'rivales cegados / uso' };
  }
  if (type === 'he' || type === 'molotov') {
    return { value: numberFormatter.format(stats.avgDamage ?? 0), label: 'daño medio / uso' };
  }
  return { value: '—', label: 'cobertura no disponible' };
};

const aggregateType = (clusters = []) => {
  const uses = clusters.reduce((sum, cluster) => sum + Number(cluster.count ?? 0), 0);
  const weightedDamage = clusters.reduce(
    (sum, cluster) => sum + Number(cluster.avg_damage ?? 0) * Number(cluster.count ?? 0),
    0
  );
  const weightedBlinded = clusters.reduce(
    (sum, cluster) => sum + Number(cluster.avg_blinded ?? 0) * Number(cluster.count ?? 0),
    0
  );
  return {
    uses,
    avgDamage: uses ? weightedDamage / uses : 0,
    avgBlinded: uses ? weightedBlinded / uses : 0,
  };
};

const clusterName = (cluster) => formatCalloutName(
  cluster?.lineup_name || cluster?.areas?.[0] || cluster?.from_area || 'Zona del mapa'
);

const getTopUtilityZones = (filtered, visibleTypes) => {
  const zones = new Map();

  Object.entries(filtered).forEach(([type, clusters]) => {
    if (visibleTypes[type] === false) return;
    clusters.forEach((cluster) => {
      const zone = clusterName(cluster);
      const uses = Number(cluster.count ?? 0);
      const current = zones.get(zone) ?? { zone, uses: 0, types: new Set(), representative: null };
      current.uses += uses;
      current.types.add(type);
      if (!current.representative || uses > Number(current.representative.count ?? 0)) {
        current.representative = { ...cluster, type };
      }
      zones.set(zone, current);
    });
  });

  return [...zones.values()].sort((a, b) => b.uses - a.uses).slice(0, 5);
};

const TacticalUtilityPanel = ({
  mapName,
  activeSide = 'ct',
  currentLevel = 'upper',
  hasLevels = false,
  zThreshold = null,
  grenadeData = {},
  matchesAnalyzed = 0,
  visibleTypes = {},
  selectedCluster,
  selectedType,
  onToggleType,
  onClusterSelect,
  onClusterClose,
}) => {
  const filtered = useMemo(() => filterGrenadeClusters({
    grenadeData,
    activeSide,
    hasLevels,
    zThreshold,
    currentLevel,
    mapName,
  }), [activeSide, currentLevel, grenadeData, hasLevels, mapName, zThreshold]);

  const types = useMemo(() => Object.entries(TYPE_CONFIG).map(([type, config]) => {
    const stats = aggregateType(filtered[type] ?? []);
    return {
      type,
      ...config,
      ...stats,
      effect: getEffect(type, stats),
      quality: getQuality(stats.uses, matchesAnalyzed),
    };
  }), [filtered, matchesAnalyzed]);

  const topZones = useMemo(
    () => getTopUtilityZones(filtered, visibleTypes),
    [filtered, visibleTypes]
  );
  const selectedStats = selectedCluster ? aggregateType([selectedCluster]) : null;
  const selectedEffect = selectedStats ? getEffect(selectedType, selectedStats) : null;
  const selectedQuality = selectedStats ? getQuality(selectedStats.uses, matchesAnalyzed) : null;

  return (
    <section className="tactical-panel tactical-utility-panel" aria-labelledby="utility-panel-title">
      <header className="tactical-panel__header">
        <div>
          <span className="tactical-eyebrow">Eficacia observada · {activeSide.toUpperCase()}</span>
          <h2 id="utility-panel-title">Utilidad</h2>
        </div>
        <div className="tactical-sample-chip">
          <strong>{matchesAnalyzed}</strong>
          <span>{matchesAnalyzed === 1 ? 'demo' : 'demos'}</span>
        </div>
      </header>

      <p className="tactical-panel__lead">
        Los usos aportan contexto; la lectura principal es el efecto medido. Si una métrica no existe, se indica expresamente.
      </p>

      <div className="tactical-utility-types" aria-label="Tipos de granada">
        {types.map(({ type, label, uses, effect, quality }) => (
          <button
            key={type}
            type="button"
            aria-pressed={visibleTypes[type] !== false}
            className={visibleTypes[type] !== false ? 'is-active' : ''}
            onClick={() => onToggleType(type)}
          >
            <span className="tactical-utility-types__icon" aria-hidden="true">
              <GrenadeImage type={type} size={46} />
            </span>
            <span>
              <strong>{label}</strong>
              <small>{uses} usos</small>
            </span>
            <span className="tactical-utility-types__effect">
              <strong>{effect.value}</strong>
              <small>{effect.label}</small>
            </span>
            <i className={`confidence-${quality.tone}`}>{quality.label}</i>
          </button>
        ))}
      </div>

      {selectedCluster && selectedType ? (
        <article className="tactical-utility-detail" aria-labelledby="utility-detail-title">
          <header>
            <GrenadeImage type={selectedType} size={48} className="tactical-utility-detail__grenade" />
            <div className="tactical-utility-detail__title">
              <span>{TYPE_CONFIG[selectedType]?.label ?? 'Granada'} seleccionada</span>
              <h3 id="utility-detail-title">{clusterName(selectedCluster)}</h3>
            </div>
            <button type="button" onClick={onClusterClose} aria-label="Cerrar detalle de utilidad">
              <X size={17} aria-hidden="true" />
            </button>
          </header>
          <div>
            <span><strong>{selectedStats.uses}</strong> usos</span>
            <span><strong>{selectedEffect.value}</strong> {selectedEffect.label}</span>
          </div>
          <p className={`confidence-${selectedQuality.tone}`}>{selectedQuality.label}. Interpreta el efecto dentro de esta muestra.</p>
        </article>
      ) : null}

      {topZones.length ? (
        <div className="tactical-zone-shortlist tactical-utility-ranking">
          <h3>Uso por zonas · {activeSide.toUpperCase()}</h3>
          {topZones.map(({ zone, uses, types: zoneTypes, representative }) => {
            const typeLabels = [...zoneTypes].map((type) => TYPE_CONFIG[type]?.label ?? type);
            return (
              <button
                key={zone}
                type="button"
                onClick={() => onClusterSelect(representative, representative.type)}
              >
                <span>
                  <strong>{zone}</strong>
                  <small>{typeLabels.join(' + ')} · {uses} usos</small>
                </span>
                <b>{uses}</b>
              </button>
            );
          })}
        </div>
      ) : (
        <div className="tactical-empty-state">
          <GrenadeImage type="he" size={54} className="tactical-utility-empty__grenade" />
          <h3>Sin utilidad observada</h3>
          <p>No hay lanzamientos para este mapa, bando y planta.</p>
        </div>
      )}
    </section>
  );
};

export default React.memo(TacticalUtilityPanel);
