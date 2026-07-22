import React, { useMemo } from 'react';
import { MapPin, Shield, Trophy } from 'lucide-react';
import { getWeaponIconPath, formatDecimal, formatMapName, formatWeaponName } from '../../utils/performanceFormatters';
import { getMapImage } from '../../utils/mapConfig';
import SectionBlock from './SectionBlock';
import { BriefingPanel, InsightCard } from './PerformanceBriefing';

const MapsTab = ({ maps = [] }) => {
  const bestMap = useMemo(() => {
    if (!maps.length) return null;
    return [...maps].sort((left, right) => Number(right.avg_rating || 0) - Number(left.avg_rating || 0))[0];
  }, [maps]);

  return (
    <div className="p-section">
      <BriefingPanel eyebrow="Briefing de mapas" title="Pool competitivo y zonas de confianza">
        <div className="p-briefing-grid-3">
          <InsightCard
            tone={bestMap ? 'good' : 'neutral'}
            icon={Trophy}
            label="Mapa lider"
            value={bestMap ? formatMapName(bestMap.map) : 'Sin muestra'}
            detail={bestMap ? `${formatDecimal(bestMap.win_rate, 0)}% WR, ${formatDecimal(bestMap.avg_rating, 2)} rating` : 'Aun no hay historial suficiente'}
          />
          <InsightCard
            tone="neutral"
            icon={MapPin}
            label="Pool registrado"
            value={maps.length}
            detail="mapas con estadistica agregada"
          />
          <InsightCard
            tone="warning"
            icon={Shield}
            label="Lectura clave"
            value="Split CT/T"
            detail="Compara si tu rating cae por bando en cada mapa"
          />
        </div>
      </BriefingPanel>

      <SectionBlock title="Pool competitivo" count={`${maps.length} mapas`}>
        {maps.length === 0 ? (
          <div className="p-card">
            <div className="p-empty">
              <MapPin size={28} className="p-empty-icon" />
              <span>Todavía no hay estadísticas por mapa.</span>
            </div>
          </div>
        ) : (
          <div className="p-map-card-grid">
            {maps.map((mapData) => {
              const wr = Number(mapData.win_rate || 0);
              const ringColor = wr >= 55 ? 'var(--p-green)' : wr >= 45 ? 'var(--p-accent)' : 'var(--p-red)';
              const ringAngle = `${Math.min(Math.max(wr, 0), 100) * 3.6}deg`;
              const isBest = bestMap?.map === mapData.map;
              const mapCover = getMapImage(mapData.map);
              const sideStats = mapData.sides || {};
              const dominantWeapon = mapData.dominant_weapon;
              const strongestSide = sideStats.strongest_side || 'Balanced';

              return (
                <div key={mapData.map} className="p-card p-map-card">
                  <div className="p-map-card-bg" style={{ backgroundImage: `url(${mapCover})` }} aria-hidden="true" />
                  <div className="p-map-card-overlay" />
                  <div className="p-map-card-head">
                    <div>
                      <h3 className="p-map-card-title">{formatMapName(mapData.map)}</h3>
                      <span className="p-map-card-sub">{Number(mapData.matches || 0)} partidas · {Number(mapData.wins || 0)}V / {Number(mapData.losses || 0)}D</span>
                    </div>
                    {isBest && (
                      <span className="p-map-best-badge">
                        <Trophy size={12} />
                        Mejor mapa
                      </span>
                    )}
                  </div>

                  <div className="p-map-card-body">
                    <div className="p-score-ring p-score-ring--sm" style={{ '--p-ring-angle': ringAngle, '--p-ring-color': ringColor }}>
                      <div className="p-score-ring-core p-score-ring-core--sm">
                        <span className="p-score-ring-value p-score-ring-value--sm">{formatDecimal(wr, 0)}%</span>
                        <span className="p-score-ring-label">WR</span>
                      </div>
                    </div>

                    <div className="p-map-card-stats">
                      <div className="p-map-card-stat">
                        <strong>{formatDecimal(mapData.avg_kd, 2)}</strong>
                        <span>K / D</span>
                      </div>
                      <div className="p-map-card-stat">
                        <strong>{formatDecimal(mapData.avg_adr, 1)}</strong>
                        <span>ADR</span>
                      </div>
                      <div className="p-map-card-stat">
                        <strong>{formatDecimal(mapData.avg_rating, 2)}</strong>
                        <span>Rating</span>
                      </div>
                    </div>
                  </div>

                  <div className="p-map-card-lower">
                    <div className="p-map-side-panel">
                      <div className="p-map-side-panel-head">
                        <span>Split por bando</span>
                        <strong className={`p-map-side-badge ${strongestSide === 'CT' ? 'ct' : strongestSide === 'T' ? 't' : 'balanced'}`}>
                          {strongestSide === 'Balanced' ? 'Equilibrado' : `Mejor ${strongestSide}`}
                        </strong>
                      </div>
                      <div className="p-map-side-grid">
                        <div className="p-map-side-cell ct">
                          <span>Valoración CT</span>
                          <strong>{formatDecimal(sideStats.ct_rating, 2)}</strong>
                          <small>{formatDecimal(sideStats.ct_adr, 1)} ADR</small>
                        </div>
                        <div className="p-map-side-cell t">
                          <span>Valoración T</span>
                          <strong>{formatDecimal(sideStats.t_rating, 2)}</strong>
                          <small>{formatDecimal(sideStats.t_adr, 1)} ADR</small>
                        </div>
                      </div>
                    </div>

                    <div className="p-map-weapon-panel">
                      <div className="p-map-weapon-panel-head">
                        <span>Armas dominantes</span>
                        {dominantWeapon && <strong>{formatWeaponName(dominantWeapon.weapon)}</strong>}
                      </div>

                      {Array.isArray(mapData.top_weapons) && mapData.top_weapons.length > 0 ? (
                        <div className="p-map-weapon-list">
                          {mapData.top_weapons.map((weapon) => {
                            const weaponIcon = getWeaponIconPath(weapon.weapon);
                            const weaponLabel = formatWeaponName(weapon.weapon);

                            return (
                              <div key={`${mapData.map}-${weapon.weapon}`} className="p-map-weapon-chip">
                                {weaponIcon && (
                                  <img
                                    src={weaponIcon}
                                    alt={weaponLabel}
                                    className="p-map-weapon-icon"
                                    loading="lazy"
                                  />
                                )}
                                <div className="p-map-weapon-copy">
                                  <span>{weaponLabel}</span>
                                  <small>{weapon.kills || 0} bajas · {formatDecimal(weapon.hs_pct, 0)}% HS</small>
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      ) : (
                        <div className="p-map-weapon-empty">Sin suficiente muestra de armas</div>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </SectionBlock>
    </div>
  );
};

export default MapsTab;
