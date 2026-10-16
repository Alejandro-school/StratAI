export const UTILITY_INFORMATION_SCHEMA = 'stratai.replay_utility_information@1';
export const UTILITY_TEAM_REPLAY_SCHEMA = 'stratai.team_replay_projection@10';
const order = (a, b) => a.localeCompare(b, 'en', {numeric:true});
const types = ['he','smoke','molotov','incendiary','flashbang','decoy','inferno'];
const sources = {launch:['demoinfocs_event:GrenadeProjectileThrow'], projectile:['demoinfocs_state:GrenadeProjectiles'],
  effect:['demoinfocs_event:SmokeStart','demoinfocs_event:InfernoStart','demoinfocs_event:HeExplode','demoinfocs_event:FlashExplode']};
const effectTypes = {'demoinfocs_event:SmokeStart':'smoke','demoinfocs_event:InfernoStart':'inferno',
  'demoinfocs_event:HeExplode':'he','demoinfocs_event:FlashExplode':'flashbang'};
const latest = (rows, tick) => {
  let low=0, high=rows.length;
  while (low<high) { const mid=(low+high)>>1; if(rows[mid].tick<=tick) low=mid+1; else high=mid; }
  return rows[low-1];
};

// Deductions use only positively observed launch instants. Late sightings and
// explosions never recover a hidden launch clock or identify an enemy.
export function estimateUtilityLaunchers(rows, tickRate) {
  if (!Number.isFinite(tickRate) || tickRate<=0) return [];
  const unique = new Map();
  for (const row of rows) if(row.kind==='launch' && row.affiliation==='unknown' && ['he','smoke','molotov','incendiary'].includes(row.utility_type)) {
    const prior=unique.get(row.object_id);
    if(prior && (prior.observation_tick!==row.observation_tick || prior.utility_type!==row.utility_type)) throw new Error('Conflicting observable launch');
    if (!prior) unique.set(row.object_id,row);
  }
  const result=[];
  for(const type of ['he','smoke','molotov','incendiary']) {
    const launches=[...unique.values()].filter(r=>r.utility_type===type).sort((a,b)=>a.observation_tick-b.observation_tick||order(a.object_id,b.object_id));
    for(let i=2;i<launches.length;i++) {
      const group=launches.slice(i-2,i+1), first=group[0], last=group[2];
      if(last.observation_tick-first.observation_tick>5*tickRate) continue;
      const evidence=rows.filter(r=>r.kind==='launch' && group.some(g=>g.object_id===r.object_id));
      const recipients=group.map(g=>new Set(evidence.filter(r=>r.object_id===g.object_id).flatMap(r=>r.recipients.map(p=>p.player_id))));
      result.push({id:`launchers:${group.map(g=>g.object_id).join(':')}`,utility_type:type,
        minimum_launchers:3,status:'estimated_expert_deduction',rule:'three_same_type_observable_launches_within_five_seconds',
        first_launch_tick:first.observation_tick,last_launch_tick:last.observation_tick,maximum_interval_seconds:5,
        object_ids:group.map(g=>g.object_id),observation_ids:evidence.map(r=>r.id).sort(order),
        possible_recipient_ids:[...recipients[0]].filter(id=>recipients.every(set=>set.has(id))).sort(order),
        side_status:'unknown',origin_status:'unknown',current_location_status:'unknown',
        exceptions:['pickups_or_extra_supply_not_excluded','type_is_visual_estimate','overlapping_groups_do_not_add_launchers']});
    }
  }
  return result;
}

export function createReplayUtilityInformationProjector(round, side, tickRate) {
  const start=round.start_tick, frames=[...(round.frames||[])].filter(f=>f.tick>=start).sort((a,b)=>a.tick-b.tick);
  const ownAt=tick=>(latest(frames,tick)?.players||[]).filter(p=>p.team===side);
  const deaths=(round.events||[]).filter(e=>e.type==='kill'&&e.tick>=start);
  const vision=round.utility_vision;
  const validSource=vision?.schema_id==='stratai.calculated_utility_vision@1' && vision.producer==='go.replay.utility_point_occlusion_v1';
  const observations=[], ids=new Set();
  for(const row of validSource ? vision.observations||[] : []) {
    const who=String(row.observer_id), tick=row.tick, basis=latest(frames,tick), own=ownAt(tick);
    if(!Number.isInteger(tick)||tick<start||row.availability_tick!==tick||row.observer_side!==side
      ||!sources[row.kind]?.includes(row.source)||!types.includes(row.utility_type)
      ||(row.kind==='effect' ? effectTypes[row.source]!==row.utility_type : row.utility_type==='inferno')
      ||(row.kind==='effect' ? row.position_basis!==(row.utility_type==='he'||row.utility_type==='flashbang'?'native_explosion_point':'effect_representative_point_not_full_geometry')
        : row.position_basis!=='native_projectile_position')
      ||row.type_status!=='estimated_native_type_proxy'||!['native_eye_position','stance_eye_proxy'].includes(row.eye_source)
      ||!['native_projectile_position','native_explosion_point','effect_representative_point_not_full_geometry'].includes(row.position_basis)
      ||!row.object_id||typeof row.id!=='string'||ids.has(row.id)||![row.x,row.y,row.z].every(Number.isFinite)
      ||!own.some(p=>String(p.steam_id)===who&&p.alive)||deaths.some(d=>String(d.victim_id)===who&&d.tick<=tick)) continue;
    ids.add(row.id);
    const recipients=own.map(p=>String(p.steam_id)).sort(order).map(player_id=>({player_id,
      kind:player_id===who?'individual_observation':'simulated_team_communication',reception:player_id===who?'origin':'possible',availability_tick:tick}));
    const ownLaunch=(round.event_context?.events||[]).some(e=>e.type==='grenade_throw'&&e.tick<=tick&&e.tick>=start
      &&e.actor_team===side&&row.object_id===`projectile:${e.projectile_id}`);
    observations.push({id:row.id,observer_id:who,object_id:row.object_id,kind:row.kind,utility_type:row.utility_type,
      type_status:row.type_status,position:{x:row.x,y:row.y,z:row.z},observation_tick:tick,availability_tick:tick,
      status:'estimated',affiliation:ownLaunch?'own_team':'unknown',
      source:{kind:'go_calculated_utility',reference:row.id,native_source:row.source,eye_source:row.eye_source,position_basis:row.position_basis},
      recipient_basis:{frame_tick:basis.tick,present_player_ids:recipients.map(r=>r.player_id)},recipients});
  }
  observations.sort((a,b)=>a.observation_tick-b.observation_tick||order(a.id,b.id));
  return tick=>{
    const prefix=observations.filter(o=>o.availability_tick<=tick), group=rows=>{
      const map=new Map(); for(const o of rows){const old=map.get(o.object_id)||[];
        if(!old.length||o.observation_tick>old[0].observation_tick) map.set(o.object_id,[o]);
        else if(o.observation_tick===old[0].observation_tick) old.push(o);}
      return [...map.values()].sort((a,b)=>order(a[0].object_id,b[0].object_id));
    };
    const recent=group(prefix), kept=new Set(prefix.filter(o=>o.kind==='launch').map(o=>o.id));
    recent.flat().forEach(o=>kept.add(o.id));
    const segments = [], lastPoint = new Map(), lastSegment = new Map();
    const projectileGroups = new Map();
    for (const row of prefix.filter(o=>o.kind==='projectile')) {
      const key = `${row.observer_id}:${row.object_id}:${row.observation_tick}`;
      if (!projectileGroups.has(key)) projectileGroups.set(key, []);
      projectileGroups.get(key).push(row);
    }
    for (const rows of projectileGroups.values()) {
      const row = rows[0];
      const key = `${row.observer_id}:${row.object_id}`, before = lastPoint.get(key);
      if (new Set(rows.map(o=>JSON.stringify(o.position))).size > 1) { lastPoint.delete(key); continue; }
      if (before && row.observation_tick-before.observation_tick===4) lastSegment.set(key, {
        object_id:row.object_id, observer_id:row.observer_id, from:{...before.position}, to:{...row.position},
        start_tick:before.observation_tick,end_tick:row.observation_tick,status:'estimated_sampled_segment',
        observation_ids:[before.id,row.id],interpolation_status:'between_points_not_observed'});
      lastPoint.set(key,row);
    }
    for (const segment of lastSegment.values()) { segments.push(segment); segment.observation_ids.forEach(ref=>kept.add(ref)); }
    // Compaction changes the first retained sample of each object. Publish a
    // stable order independent of Map insertion history so consumers agree.
    segments.sort((a,b)=>order(a.observer_id,b.observer_id)||order(a.object_id,b.object_id));
    const known=new Set(frames.filter(f=>f.tick<=tick).flatMap(f=>f.players.filter(p=>p.team===side).map(p=>String(p.steam_id))));
    const by_player=[...known].sort(order).map(player_id=>{
      const received=prefix.filter(o=>o.recipients.some(r=>r.player_id===player_id));
      const last=[...group(received.filter(o=>o.observer_id===player_id)),...group(received.filter(o=>o.observer_id!==player_id))].flat();
      const retained=[...last,...received.filter(o=>o.kind==='launch')];
      retained.forEach(o=>kept.add(o.id));
      return {player_id,observation_ids:[...new Set(retained.map(o=>o.id))].sort(order)};
    });
    const objects=recent.map(rows=>{const first=rows[0];
      const positions = new Map();
      for (const row of rows) {
        const key = JSON.stringify(row.position);
        if (!positions.has(key)) positions.set(key,{position:{...row.position},observation_ids:[]});
        positions.get(key).observation_ids.push(row.id);
      }
      const points=[...positions.values()].map(p=>({...p,observation_ids:p.observation_ids.sort(order)}));
      return {object_id:first.object_id,utility_type:first.utility_type,kind:first.kind,affiliation:first.affiliation,
        state:tick===first.observation_tick?'current_estimate':'last_observed',
        position_status:points.length===1?'estimated_point':'ambiguous_same_tick',position:points.length===1?{...first.position}:null,points,
        observation_tick:first.observation_tick,age_ticks:tick-first.observation_tick,observation_ids:rows.map(o=>o.id).sort(order)};});
    return structuredClone({schema_id:UTILITY_INFORMATION_SCHEMA,side,round_start_tick:start,
      communication_model:'immediate_simulated_possible_reception',reception_status:'possible_not_confirmed',
      source_status:validSource?'estimated_source_supplied':'unavailable',
      observations:prefix.filter(o=>kept.has(o.id)),objects,segments,by_player,launcher_estimates:estimateUtilityLaunchers(prefix,tickRate)});
  };
}
