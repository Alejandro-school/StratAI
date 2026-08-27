import React from 'react';
import { Eye, ScanLine } from 'lucide-react';
import { MetricGrid, StatPanel } from './StatisticsPrimitives';

const DuelsSection = ({ data }) => (
  <div className="pf3-stat-layout">
    <MetricGrid metrics={data.metrics} />
    <StatPanel
      eyebrow="Cara a cara"
      title="Rivales más frecuentes"
      aside="Tus datos / datos del rival"
      className="pf3-duels-panel"
    >
      <div className="pf3-duel-list">
        {data.encounters.map((duel) => (
          <article key={duel.rival}>
            <div className="pf3-duel-rival">
              <span className={`pf3-duel-result is-${duel.result.toLowerCase()}`} />
              <div><small>contra</small><strong>{duel.rival}</strong></div>
              <b>{duel.score}</b>
              <em>{duel.result}</em>
            </div>
            <div className="pf3-duel-comparison">
              <span><small>Armas</small><strong>{duel.weapon}</strong></span>
              <span><small>Disparos</small><strong>{duel.shots}</strong></span>
              <span><small>Precisión</small><strong>{duel.accuracy}</strong></span>
              <span><small>Primer daño</small><strong>{duel.firstDamage}</strong></span>
            </div>
            <footer>
              <span><ScanLine size={14} /> {duel.context}</span>
              {duel.flags.map((flag) => <span key={flag}><Eye size={14} /> {flag}</span>)}
            </footer>
          </article>
        ))}
      </div>
    </StatPanel>
  </div>
);

export default DuelsSection;
