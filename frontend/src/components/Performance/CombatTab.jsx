import React from 'react';
import { Handshake, ShieldAlert, Swords, UserRoundSearch } from 'lucide-react';
import StatPill from './StatPill';
import { getQualityLabel } from '../../utils/performanceBenchmarks';
import { formatDecimal, formatInteger, formatPercent } from '../../utils/performanceFormatters';

const clutchKeys = ['1v1', '1v2', '1v3', '1v4', '1v5'];
const mkKeys = [
  { key: '2k', label: '2K', cls: 'mk-2k' },
  { key: '3k', label: '3K', cls: 'mk-3k' },
  { key: '4k', label: '4K', cls: 'mk-4k' },
  { key: 'ace', label: 'ACE', cls: 'mk-ace' },
];

const CombatTab = ({ combat = {}, overview = {} }) => {
  const clutches = combat.clutches || {};
  const multikills = combat.multikills || {};

  // Compute max values for intensity fills
  const clutchValues = clutchKeys.map((k) => Number(clutches[k] || 0));
  const maxClutch = Math.max(...clutchValues, 1);
  const mkValues = mkKeys.map((m) => Number(multikills[m.key] || 0));
  const maxMk = Math.max(...mkValues, 1);

  return (
    <div className="perf-tab-layout">
      <div className="perf-grid perf-grid-4">
        <StatPill
          label="Éxito en aperturas"
          value={formatPercent(combat.opening_success_rate)}
          icon={Swords}
          tone="blue"
          badge={getQualityLabel(Number(combat.opening_success_rate || 0), 'opening_success')}
        />
        <StatPill
          label="Bajas de trade"
          value={formatInteger(combat.trade_kills)}
          icon={Handshake}
          tone="green"
        />
        <StatPill
          label="Muertes tradeadas"
          value={formatInteger(combat.traded_deaths)}
          icon={ShieldAlert}
          tone="orange"
        />
        <StatPill
          label="Rating de impacto"
          value={formatDecimal(overview.impact_rating, 2)}
          icon={UserRoundSearch}
          tone="purple"
          badge={getQualityLabel(Number(overview.impact_rating || 0), 'impact_rating')}
        />
      </div>

      <div className="perf-grid perf-grid-2">
        <div className="perf-card">
          <h3 className="perf-card-title">Duelos de apertura</h3>
          <div className="split-stats split-stats--3">
            <div className="split-stat-block">
              <span>Intentados</span>
              <strong>{formatInteger(combat.opening_duels_attempted)}</strong>
            </div>
            <div className="split-stat-block">
              <span>Ganados</span>
              <strong>{formatInteger(combat.opening_duels_won)}</strong>
            </div>
            <div className="split-stat-block">
              <span>Perdidos</span>
              <strong>{formatInteger(combat.opening_duels_lost)}</strong>
            </div>
          </div>
        </div>
        <div className="perf-card">
          <h3 className="perf-card-title">Clutches</h3>
          <div className="clutch-grid">
            {clutchKeys.map((key, i) => {
              const val = clutchValues[i];
              const fillOpacity = val > 0 ? (val / maxClutch) * 0.85 + 0.15 : 0;
              return (
                <div
                  key={key}
                  className={`clutch-item ${val === 0 ? 'is-zero' : ''}`}
                  style={{ '--fill-opacity': fillOpacity }}
                >
                  <span>{key}</span>
                  <strong>{val}</strong>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div className="perf-card">
        <h3 className="perf-card-title">Multi-kills</h3>
        <div className="multikill-grid">
          {mkKeys.map((mk, i) => {
            const val = mkValues[i];
            const fillOpacity = val > 0 ? (val / maxMk) * 0.85 + 0.15 : 0;
            return (
              <div
                key={mk.key}
                className={`multikill-item ${mk.cls} ${val === 0 ? 'is-zero' : ''}`}
                style={{ '--fill-opacity': fillOpacity }}
              >
                <span>{mk.label}</span>
                <strong>{val}</strong>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export default CombatTab;
