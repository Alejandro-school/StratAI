import {describeReplayPlace, resolveReplayPlace} from '../domain/replayPlaces';

export function ReplayUtilityInformationPanel({information, tickRate, labels = {}, mapName}) {
  if (!information) return null;
  const seconds = ticks => Number.isFinite(tickRate) && tickRate > 0 ? `${Number((ticks / tickRate).toFixed(2))} s` : `${ticks} ticks`;
  const name = id => labels?.[id] || `P${String(id).padStart(2,'0')}`;
  const refs = new Map(information.observations.map(o=>[o.id,o]));
  return <details className="r2-information">
    <summary>Granadas · {information.objects.length} señales visuales estimadas</summary>
    <div className="r2-information-body">
      <p>Los rombos marcan puntos estimados; los discontinuos son posiciones históricas.
        El punto de un efecto no representa toda su extensión. Ver un impacto no identifica el origen ni al lanzador.
        La comunicación y sus receptores son posibles; sonido y eficacia no están acreditados.</p>
      {information.source_status === 'unavailable' && <p>Fuente visual de utilidad no disponible en esta captura.</p>}
      {information.source_status !== 'unavailable' && !information.objects.length && <p>No hay señales visuales registradas. Esto no demuestra ausencia de granadas.</p>}
      <ul>{information.objects.map(obj=><li key={obj.object_id}>
        <strong>{obj.utility_type}</strong>: {obj.kind === 'launch' ? 'lanzamiento observable' : obj.kind === 'effect' ? 'punto de efecto' : 'proyectil'},
        {' '}hace {seconds(obj.age_ticks)}. {obj.state === 'last_observed' ? 'Posición actual desconocida.' : 'Estimación en este corte.'}
        {obj.position_status === 'ambiguous_same_tick' && ' Los callbacks aportan puntos distintos en el mismo tick: posición ambigua; una sola señal.'}
        {' '}{obj.position && describeReplayPlace(resolveReplayPlace(obj.position,mapName,0))} Origen y lanzador desconocidos.
      </li>)}</ul>
      <ul>{information.launcher_estimates.map(group=><li key={group.id}>
        Tres lanzamientos observables de {group.utility_type} en {seconds(group.last_launch_tick-group.first_launch_tick)}:
        {' '}indicio de al menos tres lanzadores. Bando, origen y posición actual desconocidos.
        Recogidas o suministro adicional no descartados. Las flashes se excluyen; los grupos solapados no se suman.
      </li>)}</ul>
      <ul>{information.by_player.filter(p=>p.observation_ids.length).map(player=><li key={player.player_id}>
        <strong>{name(player.player_id)}</strong>: {player.observation_ids.map(ref=>{
          const o=refs.get(ref), receipt=o.recipients.find(r=>r.player_id===player.player_id);
          return `${o.utility_type}: ${receipt.reception==='origin'?'observación calculada':'posible recepción'}`;
        }).join('; ')}.
      </li>)}</ul>
    </div>
  </details>;
}
