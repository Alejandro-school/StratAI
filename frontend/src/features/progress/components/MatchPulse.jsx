import React from 'react';
import { Activity, CheckCircle2, Radio } from 'lucide-react';

const MatchPulse = ({ matches, scoringMatches, maxMatches }) => (
  <section className="op-match-pulse" aria-labelledby="match-pulse-title">
    <div className="op-match-pulse__copy">
      <span className="op-eyebrow"><Activity size={13} aria-hidden="true" /> Pulso de sesión</span>
      <h2 id="match-pulse-title">Cada partida deja una señal</h2>
      <p>{scoringMatches} de {maxMatches} partidas puntuables analizadas esta semana.</p>
    </div>
    <div className="op-match-pulse__timeline">
      {matches.map((match, index) => (
        <article key={match.id} className="op-match-node">
          <span className="op-match-node__line" aria-hidden="true" />
          <i aria-hidden="true">{index === matches.length - 1 ? <Radio size={12} /> : <CheckCircle2 size={12} />}</i>
          <div><span>{match.map}</span><strong>+{match.delta} pts</strong><small>{match.result}</small></div>
        </article>
      ))}
    </div>
  </section>
);

export default MatchPulse;
