import React from 'react';
import { EyeOff, ShieldCheck, Trophy } from 'lucide-react';
import ProgressDialog from './ProgressDialog';

const LeagueDialog = ({ data, onClose, onToggleAnonymous, onToggleOptIn }) => (
  <ProgressDialog ariaLabel="Clasificación semanal" eyebrow="Liga de 50" title={data.season.league} onClose={onClose} size="wide">
    <div className="op-privacy-controls">
      <div>
        <ShieldCheck size={18} aria-hidden="true" />
        <p><b>Privacidad competitiva</b>Tu actividad solo aparece después de aceptar la liga.</p>
      </div>
      <label>
        <input name="league-participation" type="checkbox" checked={data.user.isOptedIn} onChange={onToggleOptIn} />
        Participar
      </label>
      <label>
        <input name="anonymous-league-alias" type="checkbox" checked={data.user.isAnonymous} disabled={!data.user.isOptedIn} onChange={onToggleAnonymous} />
        Alias anónimo
      </label>
    </div>

    <div className="op-leaderboard-table">
      <table aria-label="Jugadores de la liga">
        <thead>
          <tr className="op-leaderboard-table__head">
            <th>Pos.</th><th>Jugador</th><th>Misiones</th><th>Partidas</th><th>Puntos</th>
          </tr>
        </thead>
        <tbody>
          {data.leaderboard.map((entry) => {
            const visibleName = entry.isCurrentUser && data.user.isAnonymous ? 'Analista ••••' : entry.name;
            return (
              <tr key={entry.id} className={`op-leaderboard-table__row ${entry.isCurrentUser ? 'is-you' : ''}`}>
                <td>{entry.rank <= 3 ? <Trophy size={13} aria-label={`Puesto ${entry.rank}`} /> : `#${entry.rank}`}</td>
                <td><strong>{visibleName}{entry.isCurrentUser ? <small>Tú</small> : null}</strong></td>
                <td>{entry.completedMissions}/3</td>
                <td>{entry.matches}/10</td>
                <td><b>{entry.points}</b></td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>

    {!data.user.isOptedIn ? <p className="op-anonymous-note"><EyeOff size={14} aria-hidden="true" /> Tu fila no será elegible para premio hasta activar la participación.</p> : null}
  </ProgressDialog>
);

export default LeagueDialog;
