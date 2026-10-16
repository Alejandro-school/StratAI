const finite = value => typeof value === 'number' && Number.isFinite(value);
const identity = value => String(value ?? '');
const position = row => Object.fromEntries(['x','y','z'].filter(k => finite(row?.[k])).map(k => [k,row[k]]));
const latest = (rows,tick) => {
  let low=0,high=rows.length;
  while(low<high){const mid=(low+high)>>1;if(rows[mid].tick<=tick)low=mid+1;else high=mid;}
  return rows[low-1];
};

// The callback extension is deliberately independent from final combat/utility
// summaries. A contact at the final cut cannot authorize an earlier action.
export function createReplayEventsProjector(round, side, contactPolicy = 'calculated', factual = false) {
  const supported = ['T','CT'].includes(side) && round?.event_context?.schema_id === 'stratai.replay_events@1';
  const frames = [...(round?.frames || [])].sort((a,b)=>a.tick-b.tick);
  const start = round?.start_tick;
  const deaths = (round?.events || []).filter(e => e.type === 'kill' && e.tick >= start);
  const vision = round?.calculated_vision;
  const samples = contactPolicy === 'calculated' && vision?.schema_id === 'stratai.calculated_vision@2'
    && vision.producer === 'go.reaction.world_occlusion_v2' && vision.sample_stride_ticks === 4
    ? [...(vision.samples || [])].filter(s => finite(s.tick) && s.tick >= start && s.availability_tick === s.tick).sort((a,b)=>a.tick-b.tick) : [];
  const alive = (id,tick) => !deaths.some(d => identity(d.victim_id) === id && d.tick <= tick);
  function ownAt(id,tick) {
    return latest(frames,tick)?.players?.find(p => identity(p.steam_id) === id && p.team === side);
  }
  function contactAt(id,tick) {
    const sample=latest(samples,tick);
    if(sample?.status !== 'calculated' || tick >= sample.tick+4 || !alive(id,tick))return null;
    const contact=(sample.contacts || []).find(c => identity(c.subject_id) === id && c.subject_side !== side
      && ['T','CT'].includes(c.subject_side) && c.observer_side === side
      && ownAt(identity(c.observer_id),tick)?.alive && alive(identity(c.observer_id),tick)
      && finite(c.x) && finite(c.y));
    return contact ? {contact,sample} : null;
  }
  const raw = supported ? [...(round.event_context.events || [])].filter(e => finite(e.tick) && e.tick >= start
    && finite(e.availability_tick) && e.availability_tick >= e.tick && e.id && e.source).sort((a,b)=>a.tick-b.tick) : [];
  // Each record is projected once, using only its event-time prefix. This keeps
  // seeking and playback from repeatedly traversing the full event ledger.
  const projected = raw.flatMap(e => {
    if (e.type === 'weapon_fire' && /knife|bayonet|grenade|flashbang|molotov|incendiary|decoy|^c4$/i.test(e.weapon || '')) return [];
    const actor=identity(e.actor_id),own=ownAt(actor,e.tick),seen=own ? null : contactAt(actor,e.tick);
    const victim=identity(e.victim_id),ownVictim=ownAt(victim,e.tick);
    const utilityEffect=['utility_start','utility_end'].includes(e.type);
    if(e.type === 'player_hurt' ? !ownVictim : !own && (!seen || utilityEffect))return [];
    if(!['weapon_fire','player_hurt','grenade_throw','utility_start','utility_end'].includes(e.type))return [];
    const base={id:e.id,type:e.type,tick:e.tick,availability_tick:e.availability_tick,
      knowledge:seen ? 'calculated_visual_event' : 'own_team_event',
      source:{kind:'demo_atomic_event',reference:e.id,native_source:typeof e.source==='string'?e.source:e.source.reference},
      lane:e.type==='weapon_fire'||e.type==='player_hurt'?'combat':'utility',
      label:{weapon_fire:'Disparo',player_hurt:'Daño recibido',grenade_throw:'Granada lanzada',utility_start:'Efecto iniciado',utility_end:'Efecto terminado'}[e.type]};
    if(own || seen){
      base.actor_id=actor;base.actor_team=own ? side : seen.contact.subject_side;
      if(e.type === 'weapon_fire' || e.type === 'player_hurt')if(e.weapon)base.weapon=e.weapon;
    }
    if(seen)base.observation={observer_id:identity(seen.contact.observer_id),tick:seen.sample.tick,
      source:{kind:'go_calculated_contact',reference:`round:${round.round}:contact:${seen.sample.tick}:${seen.contact.observer_id}:${actor}`}};
    if(e.type==='player_hurt'){
      base.victim_id=victim;base.victim_team=side;
      if(finite(e.damage)&&e.damage>=0)base.damage=e.damage;
      if(finite(e.health)&&e.health>=0)base.health=e.health;
      // Callback position is the victim position for this type.
      Object.assign(base,position(e));
      if (factual && finite(base.x) && finite(base.y)) base.position_tick=e.tick;
    }else{
      Object.assign(base,position(seen ? seen.contact : e));
      if (!factual || finite(base.x) && finite(base.y)) base.position_tick=seen ? seen.sample.tick : e.tick;
    }
    if(['grenade_throw','utility_start','utility_end'].includes(e.type)){
      if(e.utility_type)base.utility_type=e.utility_type;
      if(own && e.projectile_id != null)base.projectile_id=identity(e.projectile_id);
      if(own && e.effect_id)base.effect_id=identity(e.effect_id);
    }
    return [base];
  });
  return tick => {
    const events=projected.filter(e=>e.tick<=tick&&e.availability_tick<=tick);
    const snapshot=latest(frames,tick);
    const launches=new Map(events.filter(e=>e.type==='grenade_throw'&&e.actor_team===side&&e.projectile_id)
      .map(e=>[e.projectile_id,e]));
    const projectiles=supported ? (snapshot?.projectiles || []).flatMap(p=>{
      const launch=launches.get(identity(p.id));
      if(!launch || identity(p.thrower_id)!==launch.actor_id || !finite(p.x)||!finite(p.y)
        || events.some(e=>['utility_start','utility_end'].includes(e.type)&&e.projectile_id===launch.projectile_id))return [];
      return [{id:p.id,type:p.type,thrower_id:launch.actor_id,...position(p),
        trajectory:Array.isArray(p.trajectory)&&p.trajectory.every(finite)?[...p.trajectory]:[],
        observation_tick:snapshot.tick,availability_tick:snapshot.tick,knowledge:'simulated_team_utility_tracking',
        source:{kind:'demo_projectile_frame',reference:`round:${round.round}:frame:${snapshot.tick}:projectile:${p.id}`}}];
    }) : [];
    const active_effects=supported ? (snapshot?.active_effects || []).flatMap(effect=>{
      const launch=launches.get(identity(effect.projectile_id));
      const observedStart=events.find(e=>e.type==='utility_start'&& e.actor_team===side
        && (effect.effect_id ? e.effect_id===identity(effect.effect_id) : launch && e.projectile_id===launch.projectile_id)
        && e.actor_id===identity(effect.thrower_id) && e.tick===effect.start_tick);
      if(!observedStart
        || events.some(e=>e.type==='utility_end' && (observedStart.effect_id ? e.effect_id===observedStart.effect_id : e.projectile_id===observedStart.projectile_id) && e.tick>=observedStart.tick)
        || !finite(effect.x)||!finite(effect.y))return [];
      return [{type:effect.type,...(observedStart.effect_id?{effect_id:observedStart.effect_id}:{}),
        ...(observedStart.projectile_id?{projectile_id:observedStart.projectile_id}:{}),thrower_id:observedStart.actor_id,
        ...position(effect),start_tick:effect.start_tick,
        ...Object.fromEntries(['hull','points'].filter(k=>Array.isArray(effect[k])&&effect[k].length%2===0&&effect[k].every(finite)).map(k=>[k,[...effect[k]]])),
        ...(finite(effect.radius)?{radius:effect.radius}:{}),
        observation_tick:snapshot.tick,availability_tick:snapshot.tick,knowledge:'simulated_team_utility_tracking',
        source:{kind:'demo_effect_frame',reference:`round:${round.round}:frame:${snapshot.tick}:effect:${observedStart.id}`}}];
    }) : [];
    return {events,shots:events.filter(e=>e.type==='weapon_fire'&&tick-e.tick<32),projectiles,active_effects,
      limitations:{legacy_events:supported?'not_used':'atomic_source_unavailable',
        enemy_utility_geometry:'observation_source_unavailable',own_utility_tracking:'simulated_team_information_not_confirmed_view',
        sound_audibility:'not_supplied',tactical_efficacy:'not_established',final_aggregates:'excluded'}};
  };
}
