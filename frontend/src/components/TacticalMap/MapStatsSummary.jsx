import React from 'react';

const MapStatsSummary = ({ sideStats, matchesAnalyzed, mapName }) => {
  const displayName = mapName?.replace('de_', '').toUpperCase() || 'MAP';
  const ct = sideStats?.CT;
  const t = sideStats?.T;

  const totalKills = (ct?.kills || 0) + (t?.kills || 0);
  const totalDeaths = (ct?.deaths || 0) + (t?.deaths || 0);
  const combinedKD = totalDeaths > 0 ? (totalKills / totalDeaths).toFixed(2) : totalKills.toFixed(2);

  return (
    <div className="map-summary">
      <div className="summary-header">
        <h3>{displayName}</h3>
        <span className="summary-matches">{matchesAnalyzed} partidas • K/D {combinedKD}</span>
      </div>

      <div className="side-stats-container">
        <div className="side-stats ct">
          <div className="side-label">
            <span className="side-icon ct">CT</span>
          </div>
          <div className="side-numbers">
            <div className="side-stat">
              <span className="stat-value">{ct?.kd?.toFixed(2) || '0.00'}</span>
              <span className="stat-label">K/D</span>
            </div>
            <div className="side-stat">
              <span className="stat-value">{Math.round(ct?.adr || 0)}</span>
              <span className="stat-label">ADR</span>
            </div>
            <div className="side-stat">
              <span className="stat-value">{ct?.hs_pct?.toFixed(0) || 0}%</span>
              <span className="stat-label">HS</span>
            </div>
          </div>
        </div>

        <div className="side-stats t">
          <div className="side-label">
            <span className="side-icon t">T</span>
          </div>
          <div className="side-numbers">
            <div className="side-stat">
              <span className="stat-value">{t?.kd?.toFixed(2) || '0.00'}</span>
              <span className="stat-label">K/D</span>
            </div>
            <div className="side-stat">
              <span className="stat-value">{Math.round(t?.adr || 0)}</span>
              <span className="stat-label">ADR</span>
            </div>
            <div className="side-stat">
              <span className="stat-value">{t?.hs_pct?.toFixed(0) || 0}%</span>
              <span className="stat-label">HS</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default MapStatsSummary;
