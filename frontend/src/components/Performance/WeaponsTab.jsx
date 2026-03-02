import React, { useMemo, useState } from 'react';
import { formatDecimal, formatInteger, formatWeaponName } from '../../utils/performanceFormatters';

const SORT_OPTIONS = {
  kills: { key: 'kills', label: 'Bajas' },
  accuracy: { key: 'accuracy', label: 'Precisión' },
  hs_pct: { key: 'hs_pct', label: 'HS %' },
  damage: { key: 'damage', label: 'Daño' },
};

const TOP_WEAPON_BADGES = ['🥇', '🥈', '🥉'];

const BarCell = ({ value, maxValue, children }) => {
  const pct = maxValue > 0 ? Math.min((value / maxValue) * 100, 100) : 0;
  return (
    <td className="weapon-bar-cell">
      <div className="bar-fill" style={{ width: `${pct}%` }} />
      <span className="bar-value">{children}</span>
    </td>
  );
};

const WeaponsTab = ({ weapons = [], overview = {} }) => {
  const [sortKey, setSortKey] = useState('kills');

  const sortedWeapons = useMemo(() => {
    return [...weapons].sort((a, b) => Number(b[sortKey] || 0) - Number(a[sortKey] || 0));
  }, [weapons, sortKey]);

  // Compute column maximums for bar fills
  const maxKills = useMemo(() => Math.max(...weapons.map((w) => Number(w.kills || 0)), 1), [weapons]);
  const maxDamage = useMemo(() => Math.max(...weapons.map((w) => Number(w.damage || 0)), 1), [weapons]);

  return (
    <div className="perf-tab-layout">
      <div className="perf-card">
        <div className="table-toolbar">
          <h3 className="perf-card-title">Arsenal completo</h3>
          <select value={sortKey} onChange={(event) => setSortKey(event.target.value)} className="perf-select">
            {Object.entries(SORT_OPTIONS).map(([key, opt]) => (
              <option key={key} value={opt.key}>{`Ordenar por ${opt.label}`}</option>
            ))}
          </select>
        </div>

        <div className="table-wrapper">
          <table className="perf-table">
            <thead>
              <tr>
                <th>Arma</th>
                <th>Bajas</th>
                <th>Precisión</th>
                <th>HS %</th>
                <th>Daño</th>
              </tr>
            </thead>
            <tbody>
              {sortedWeapons.length === 0 && (
                <tr>
                  <td colSpan={5} className="perf-empty-cell">Todavía no hay estadísticas de armas.</td>
                </tr>
              )}

              {sortedWeapons.map((weapon, index) => (
                <tr key={`${weapon.weapon}-${index}`}>
                  <td>
                    <div className={`weapon-label ${index < 3 ? `weapon-top-${index + 1}` : ''}`}>
                      {index < 3 && <span className="weapon-medal">{TOP_WEAPON_BADGES[index]}</span>}
                      {formatWeaponName(weapon.weapon)}
                    </div>
                  </td>
                  <BarCell value={Number(weapon.kills || 0)} maxValue={maxKills}>
                    {formatInteger(weapon.kills)}
                  </BarCell>
                  <td>{formatDecimal(weapon.accuracy, 1, '%')}</td>
                  <td>{formatDecimal(weapon.hs_pct, 1, '%')}</td>
                  <BarCell value={Number(weapon.damage || 0)} maxValue={maxDamage}>
                    {formatInteger(weapon.damage)}
                  </BarCell>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default WeaponsTab;
