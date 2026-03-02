import React from 'react';
import { MapPin } from 'lucide-react';
import { formatDecimal, formatMapName } from '../../utils/performanceFormatters';
import SectionBlock from './SectionBlock';

const MapsTab = ({ maps = [] }) => (
  <div className="p-section">

    <SectionBlock title="Mapas jugados" count={`${maps.length} mapas`}>
      {maps.length === 0 ? (
        <div className="p-empty">
          <MapPin size={28} className="p-empty-icon" />
          <span>Todavía no hay estadísticas por mapa.</span>
        </div>
      ) : (
        <div className="p-map-list">
          {maps.map((mapData) => {
            const wr = Number(mapData.win_rate || 0);
            const wrColor = wr >= 55
              ? 'var(--p-green-text)'
              : wr >= 45
                ? 'var(--p-text)'
                : 'var(--p-red-text)';

            return (
              <div key={mapData.map} className="p-map-row">
                <div
                  className="p-map-row-bg"
                  style={{ backgroundImage: `url(/images/maps/${mapData.map}.png)` }}
                  aria-hidden="true"
                />

                {/* Identity */}
                <div className="p-map-identity">
                  <span className="p-map-name">{formatMapName(mapData.map)}</span>
                  <span className="p-map-matches">{Number(mapData.matches || 0)} partidas</span>
                  <span className="p-map-wl">{Number(mapData.wins || 0)}V · {Number(mapData.losses || 0)}D</span>
                </div>

                {/* Stats */}
                <div className="p-map-stats">
                  <div className="p-map-stat">
                    <span className="p-map-stat-value">{formatDecimal(mapData.avg_kd, 2)}</span>
                    <span className="p-map-stat-label">K / D</span>
                  </div>
                  <div className="p-map-stat">
                    <span className="p-map-stat-value">{formatDecimal(mapData.avg_adr, 1)}</span>
                    <span className="p-map-stat-label">ADR</span>
                  </div>
                  <div className="p-map-stat">
                    <span className="p-map-stat-value">{formatDecimal(mapData.avg_rating, 2)}</span>
                    <span className="p-map-stat-label">Rating</span>
                  </div>
                </div>

                {/* Win rate */}
                <div className="p-map-winrate">
                  <span className="p-map-winrate-value" style={{ color: wrColor }}>
                    {formatDecimal(wr, 0)}%
                  </span>
                  <span className="p-map-winrate-label">Win rate</span>
                  <div className="p-progress-track" style={{ width: '100%' }}>
                    <div
                      className="p-progress-fill"
                      style={{
                        width: `${wr}%`,
                        background: wr >= 55
                          ? 'var(--p-green)'
                          : wr >= 45
                            ? 'var(--p-accent)'
                            : 'var(--p-red)',
                      }}
                    />
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

export default MapsTab;
