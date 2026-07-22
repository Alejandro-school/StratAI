import React, { useMemo, useState } from 'react';
import { Crosshair, Target, Zap } from 'lucide-react';
import { getWeaponIconPath, formatDecimal, formatInteger, formatPercent, formatWeaponName } from '../../utils/performanceFormatters';
import { MetricCell } from './StatPill';
import SectionBlock from './SectionBlock';
import { BriefingPanel, InsightCard } from './PerformanceBriefing';

const SORT_OPTIONS = [
  { key: 'kills', label: 'Bajas' },
  { key: 'accuracy', label: 'Precisión' },
  { key: 'hs_pct', label: 'HS %' },
  { key: 'damage', label: 'Daño' },
];

const TOP_BADGES = ['#1', '#2', '#3'];

const WeaponsTab = ({ weapons = [], overview = {} }) => {
  const [sortKey, setSortKey] = useState('kills');
  const [showTable, setShowTable] = useState(false);

  const sortedWeapons = useMemo(
    () => [...weapons].sort((left, right) => Number(right[sortKey] || 0) - Number(left[sortKey] || 0)),
    [weapons, sortKey],
  );
  const maxKills = Math.max(...weapons.map((weapon) => Number(weapon.kills || 0)), 1);
  const maxDamage = Math.max(...weapons.map((weapon) => Number(weapon.damage || 0)), 1);
  const topWeapon = sortedWeapons[0];

  return (
    <div className="p-section">
      <BriefingPanel eyebrow="Briefing de arsenal" title="Armas que realmente convierten rondas">
        <div className="p-briefing-grid-3">
          <InsightCard
            tone="good"
            icon={Target}
            label="Arma dominante"
            value={topWeapon ? formatWeaponName(topWeapon.weapon) : 'Sin muestra'}
            detail={topWeapon ? `${formatInteger(topWeapon.kills)} bajas, ${formatInteger(topWeapon.damage)} dano` : 'Juega mas partidas para abrir lectura'}
          />
          <InsightCard
            tone={Number(topWeapon?.accuracy || 0) >= 25 ? 'good' : 'warning'}
            icon={Crosshair}
            label="Precision top"
            value={topWeapon ? formatDecimal(topWeapon.accuracy, 1, '%') : '0%'}
            detail={topWeapon ? `${formatDecimal(topWeapon.hs_pct, 1, '%')} HS` : 'Sin arma principal'}
          />
          <InsightCard
            tone="neutral"
            icon={Zap}
            label="Variedad"
            value={formatInteger(weapons.length)}
            detail="armas con estadistica registrada"
          />
        </div>
      </BriefingPanel>

      <div className="p-metric-row">
        <MetricCell value={formatInteger(overview.kills)} label="Bajas totales" />
        <MetricCell value={formatInteger(overview.total_damage)} label="Daño total" />
        <MetricCell value={formatPercent(overview.hs_pct)} label="HS % global" />
        <MetricCell value={formatInteger(weapons.length)} label="Armas usadas" />
      </div>

      <SectionBlock title="Armas principales">
        <div className="p-card">
          <div className="p-card-toolbar">
            <div>
              <p className="p-card-title">Top 5 del arsenal</p>
              <p className="p-card-subtitle">Lectura rápida de dónde estás haciendo daño real</p>
            </div>
            <select value={sortKey} onChange={(event) => setSortKey(event.target.value)} className="p-select">
              {SORT_OPTIONS.map((option) => (
                <option key={option.key} value={option.key}>Ordenar por {option.label}</option>
              ))}
            </select>
          </div>

          {sortedWeapons.length === 0 ? (
            <div className="p-empty">
              <span>Todavía no hay estadísticas de armas suficientes.</span>
            </div>
          ) : (
            <div className="p-weapon-card-list">
              {sortedWeapons.slice(0, 5).map((weapon, index) => {
                const accuracyAngle = `${Math.min(Math.max(Number(weapon.accuracy || 0), 0), 100) * 3.6}deg`;
                const weaponLabel = formatWeaponName(weapon.weapon);
                const weaponIcon = getWeaponIconPath(weapon.weapon);

                return (
                  <div key={`${weapon.weapon}-${index}`} className="p-weapon-card">
                    <div className="p-weapon-card-main">
                      <div className="p-weapon-card-head">
                        <span className="p-weapon-rank-badge">{TOP_BADGES[index] || `#${index + 1}`}</span>
                        {weaponIcon && (
                          <div className="p-weapon-icon-shell">
                            <img
                              src={weaponIcon}
                              alt={weaponLabel}
                              className="p-weapon-icon"
                              loading="lazy"
                            />
                          </div>
                        )}
                        <div className="p-weapon-title-block">
                          <strong className="p-weapon-card-title">{weaponLabel}</strong>
                          <span className="p-weapon-card-sub">{formatInteger(weapon.kills)} bajas · {formatInteger(weapon.damage)} de daño</span>
                        </div>
                      </div>

                      <div className="p-weapon-bars">
                        <div className="p-weapon-bar-row">
                          <span>Kills</span>
                          <div className="p-progress-track">
                            <div className="p-progress-fill p-progress-fill--purple" style={{ width: `${(Number(weapon.kills || 0) / maxKills) * 100}%` }} />
                          </div>
                          <strong>{formatInteger(weapon.kills)}</strong>
                        </div>
                        <div className="p-weapon-bar-row">
                          <span>Daño</span>
                          <div className="p-progress-track">
                            <div className="p-progress-fill p-progress-fill--orange" style={{ width: `${(Number(weapon.damage || 0) / maxDamage) * 100}%` }} />
                          </div>
                          <strong>{formatInteger(weapon.damage)}</strong>
                        </div>
                      </div>
                    </div>

                    <div className="p-weapon-ring" style={{ '--p-ring-angle': accuracyAngle, '--p-ring-color': 'var(--p-blue)' }}>
                      <div className="p-weapon-ring-core">
                        <span>{formatDecimal(weapon.accuracy, 1)}%</span>
                        <small>ACC</small>
                      </div>
                    </div>

                    <div className="p-weapon-card-meta">
                      <span className="p-badge neutral">HS {formatDecimal(weapon.hs_pct, 1)}%</span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </SectionBlock>

      <SectionBlock title="Tabla completa" count={`${weapons.length} armas`}>
        <div className="p-card">
          <div className="p-card-toolbar">
            <p className="p-card-title">Detalle por arma</p>
            <button type="button" className="p-ghost-btn" onClick={() => setShowTable((current) => !current)}>
              {showTable ? 'Ocultar tabla' : 'Ver todas las armas'}
            </button>
          </div>

          {showTable && (
            <div className="p-table-wrap">
              <table className="p-table">
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
                  {sortedWeapons.map((weapon, index) => {
                    const weaponLabel = formatWeaponName(weapon.weapon);
                    const weaponIcon = getWeaponIconPath(weapon.weapon);

                    return (
                      <tr key={`${weapon.weapon}-${index}`}>
                        <td className="p-weapon-table-cell">
                          <div className="p-weapon-name">
                            {weaponIcon && (
                              <img
                                src={weaponIcon}
                                alt={weaponLabel}
                                className="p-weapon-table-icon"
                                loading="lazy"
                              />
                            )}
                            {index < 3 && <span className="p-weapon-rank">{TOP_BADGES[index]}</span>}
                            {weaponLabel}
                          </div>
                        </td>
                        <td>{formatInteger(weapon.kills)}</td>
                        <td>{formatDecimal(weapon.accuracy, 1)}%</td>
                        <td>{formatDecimal(weapon.hs_pct, 1)}%</td>
                        <td>{formatInteger(weapon.damage)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </SectionBlock>
    </div>
  );
};

export default WeaponsTab;
