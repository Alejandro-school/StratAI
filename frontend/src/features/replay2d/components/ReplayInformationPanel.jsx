import { describeReplayPlace, resolveReplayPlace } from '../domain/replayPlaces';

export function ReplayInformationPanel({ information, tick, tickRate, labels = {}, mapName }) {
  if (!information) return null;
  const name = id => labels?.[id] || (String(id).length <= 3 ? `P${String(id).padStart(2, '0')}` : 'Jugador observado');
  const age = ticks => Number.isFinite(tickRate) && tickRate > 0 ? `${Number((ticks / tickRate).toFixed(1))} s` : `${ticks} ticks`;
  const refs = new Map(information.observations.map(o => [o.id, o]));
  const memories = information.contacts.filter(c => c.state === 'last_observed').length;
  return <details className="r2-information">
    <summary>Información disponible · {information.contacts.length - memories} contactos y {memories} recuerdos</summary>
    <div className="r2-information-body">
      <p>La comunicación inmediata es simulada. Poder recibir información no acredita escucha, atención ni recuerdo.
        Los círculos huecos marcan la última posición observada; su posición actual es desconocida.
        Las etiquetas rivales enlazan registros de la demo; no acreditan reconocimiento de una persona.</p>
      {!information.contacts.length && <p>No hay contactos registrados. Esto no demuestra ausencia de rivales.</p>}
      <ul>{information.contacts.map(contact => {
        const place = describeReplayPlace(resolveReplayPlace(contact.position, mapName, 0));
        return <li key={contact.subject_id}>
          <strong>{name(contact.subject_id)}</strong>: {contact.state === 'last_observed' ? 'última observación' : 'contacto estimado'}, hace {age(contact.age_ticks)}.
          {place && ` ${place}.`}{contact.state === 'last_observed' && ' Posición actual desconocida.'}
        </li>;
      })}</ul>
      <ul>{information.by_player.filter(player => player.observation_ids.length).map(player => <li key={player.player_id}>
        <strong>{name(player.player_id)}</strong>: {player.observation_ids.map(ref => {
          const observation = refs.get(ref);
          const receipt = observation.recipients.find(r => r.player_id === player.player_id);
          const mode = receipt.reception === 'origin' ? (observation.status === 'observed' ? 'observación acreditada por la fuente' : 'observación calculada') : 'posible recepción';
          return `${name(observation.subject_id)} (${mode}, hace ${age(tick - observation.observation_tick)})`;
        }).join('; ')}.
      </li>)}</ul>
    </div>
  </details>;
}
