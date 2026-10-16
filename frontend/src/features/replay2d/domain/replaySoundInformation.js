import catalog from '../../../../../backend/go-service/pkg/sound/cs2_profiles.json';
import {resolveReplayPlace} from './replayPlaces';

export const SOUND_INFORMATION_SCHEMA = 'stratai.replay_sound_information@1';
export const SOUND_TEAM_REPLAY_SCHEMA = 'stratai.team_replay_projection@11';
export const SOUND_CATALOG_SHA256 = 'be731fec74d709e3c2975a944b61a5d2858436071165e2be107bf2772076e689';
const profiles = new Map(catalog.events.map(p=>[p.event.toLowerCase(),p]));
const sources = ['net.CMsgSosStartSoundEvent','net.CCSUsrMsg_WeaponSound','net.CMsgTEFireBullets'];
const order = (a,b)=>a.localeCompare(b,'en',{numeric:true});
const point = p=>p && [p.X,p.Y,p.Z].every(Number.isFinite);
const at = (frames,tick)=>{let low=0,high=frames.length;while(low<high){const mid=(low+high)>>1;if(frames[mid].tick<=tick)low=mid+1;else high=mid;}return frames[low-1];};
// Native material/weapon profile variants are range evidence, not distinct
// perceived people or types. Retain the latest dated category per region.
const cell = o=>JSON.stringify([o.kind,o.source_side,o.region.candidates.map(c=>c.id)]);
const recent = rows=>{const map=new Map();for(const o of rows)map.set(cell(o),o);return [...map.values()].map(o=>o.id).sort(order);};

// Native points only verify geometric eligibility and a compatible region. They
// never become enemy markers, recognized identities or exact perceived points.
function soundRows(round,side,mapName) {
  if(!round||!['T','CT'].includes(side))return {rows:[],frames:[],start:0,valid:false};
  const start=round.start_tick,frames=[...(round.frames||[])].filter(f=>f.tick>=start).sort((a,b)=>a.tick-b.tick);
  const deaths=(round.events||[]).filter(e=>e.type==='kill'&&e.tick>=start);
  const source=round.sound_information;
  const valid=source?.schema_id==='stratai.calculated_sound_information@1'
    && source.producer==='go.replay.native_sound_profile_3d_v1' && source.catalog_id===catalog.catalog_id
    && source.catalog_sha256===SOUND_CATALOG_SHA256;
  const emissions=new Map(),duplicate=new Set();
  for(const e of valid?source.emissions||[]:[]) {
    if(emissions.has(e.id))duplicate.add(e.id);
    emissions.set(e.id,e);
  }
  const rows=[],ids=new Set();
  for(const o of valid?source.observations||[]:[]) {
    const e=emissions.get(o.emission_id),p=profiles.get(e?.profile?.toLowerCase());
    const when=o.tick,who=String(o.observer_id),basis=at(frames,when);
    const own=(basis?.players||[]).filter(p=>p.team===side),observer=e?.native_observers?.filter(p=>String(p.player_id)===who);
    if(!e||duplicate.has(e.id)||!p?.usable||p.local_only||p.silent||!p.broadcast||!sources.includes(e.source)
      ||e.family!==p.family||e.radius_units!==p.radius_units||e.radius_basis!==p.radius_basis
      ||!point(e.native_position)||!Number.isInteger(when)||when<start||e.tick!==when||e.availability_tick!==when
      ||o.availability_tick!==when||o.observer_side!==side||o.status!=='estimated_possible_reception'
      ||!['T','CT','unknown'].includes(e.emitter_side)||e.emitter_side===side||String(e.emitter_id)===who
      ||typeof o.id!=='string'||!/^sound-observation:\d+$/.test(o.id)||ids.has(o.id)
      ||!/^sound-emission:\d+$/.test(e.id)||observer?.length!==1||!observer[0].alive||observer[0].side!==side
      ||!point(observer[0].native_position)||!own.some(p=>String(p.steam_id)===who&&p.alive)
      ||deaths.some(d=>String(d.victim_id)===who&&d.tick<=when))continue;
    if(e.source==='net.CMsgSosStartSoundEvent'&&e.native_sound_hash!==p.hash)continue;
    if(e.source==='net.CMsgTEFireBullets'&&catalog.fire_sounds_by_item_definition[String(e.native_item_definition)]?.[String(e.native_sound_type)]!==p.event)continue;
    const volume=e.native_volume??p.volume;
    if(!Number.isFinite(volume)||volume<=0)continue;
    const a=e.native_position,b=observer[0].native_position,distance=Math.hypot(a.X-b.X,a.Y-b.Y,a.Z-b.Z);
    if(!Number.isFinite(o.distance_3d_units)||Math.abs(distance-o.distance_3d_units)>1e-7||distance>=p.radius_units)continue;
    ids.add(o.id);
    const recipients=own.map(p=>String(p.steam_id)).sort(order).map(player_id=>({player_id,
      kind:player_id===who?'individual_estimate':'simulated_team_communication',reception:'possible',availability_tick:when}));
    const region=resolveReplayPlace({x:a.X,y:a.Y,z:a.Z},mapName);
    rows.push({id:o.id,observer_id:who,kind:p.family,profile_hash:p.hash,source_side:e.emitter_side,
      side_status:'native_origin_estimate_not_recognition',region:{status:'compatible_region_estimate',
        reference_id:region.reference_id,candidates:region.candidates.map(c=>({id:c.id,label:c.label})),build_compatibility:'unverified'},
      observation_tick:when,availability_tick:when,status:'estimated_possible_reception',distance_3d_units:distance,
      radius_units:p.radius_units,radius_basis:p.radius_basis,
      source:{kind:'go_calculated_sound',reference:o.id,emission_reference:e.id,native_source:e.source,catalog_id:catalog.catalog_id,catalog_sha256:SOUND_CATALOG_SHA256},
      recipient_basis:{frame_tick:basis.tick,present_player_ids:recipients.map(r=>r.player_id)},recipients});
  }
  rows.sort((a,b)=>a.observation_tick-b.observation_tick||order(a.id,b.id));
  return {rows,frames,start,valid};
}

export function createReplaySoundInformationProjector(round,side,mapName) {
  const {rows,frames,start,valid}=soundRows(round,side,mapName);
  return tick=>{
    const prefix=rows.filter(o=>o.availability_tick<=tick),kept=new Set(recent(prefix));
    const known=[...new Set(frames.filter(f=>f.tick<=tick).flatMap(f=>f.players.filter(p=>p.team===side).map(p=>String(p.steam_id))))].sort(order);
    const by_player=known.map(player_id=>{
      const received=prefix.filter(o=>o.recipients.some(r=>r.player_id===player_id));
      const individual_signal_ids=recent(received.filter(o=>o.observer_id===player_id));
      const possible_shared_signal_ids=recent(received.filter(o=>o.observer_id!==player_id));
      [...individual_signal_ids,...possible_shared_signal_ids].forEach(id=>kept.add(id));
      return {player_id,individual_signal_ids,possible_shared_signal_ids};
    });
    return structuredClone({schema_id:SOUND_INFORMATION_SCHEMA,side,round_start_tick:start,
      source_status:valid?'estimated_source_supplied':'unavailable',catalog_id:catalog.catalog_id,catalog_sha256:SOUND_CATALOG_SHA256,
      communication_model:'immediate_simulated_possible_reception',reception_status:'possible_not_confirmed',
      geometry_model:'native_points_euclidean_3d_no_visual_occlusion',build_compatibility:'unverified',
      perception:{listening:'unknown',identity:'unknown',enemy_count:'unknown',exact_position:'unknown',current_enemy_position:'unknown',masking:'unknown'},
      observations:prefix.filter(o=>kept.has(o.id)),by_player});
  };
}

// Review-only demo overlay, explicitly excluded from the factual/model packet.
// The point is the actor's snapshot at emission, never a later player position.
// 1.5 seconds is a display lifetime, not a claim about knowledge or audibility.
export function createReplaySoundReviewProjector(round,side,mapName) {
  const {rows}=soundRows(round,side,mapName);
  const emissions=new Map((round?.sound_information?.emissions||[]).map(e=>[e.id,e]));
  const actors=new Map(),points=[];
  for(const row of rows) {
    const e=emissions.get(row.source.emission_reference);
    if(!['T','CT'].includes(e.emitter_side)||!e.emitter_id||String(e.emitter_id)==='0')continue;
    const actor=e.native_observers.filter(p=>String(p.player_id)===String(e.emitter_id));
    if(actor.length!==1||actor[0].side!==e.emitter_side||!actor[0].alive||!point(actor[0].native_position))continue;
    if(!actors.has(String(e.emitter_id)))actors.set(String(e.emitter_id),`sound-marker:${actors.size+1}`);
    const a=actor[0].native_position;
    points.push({id:actors.get(String(e.emitter_id)),x:a.X,y:a.Y,z:a.Z,side:e.emitter_side,kind:row.kind,
      observation_tick:row.observation_tick,emission_reference:e.id,
      position_basis:'demo_emitter_at_sound_tick',reception:'estimated_possible_reception'});
  }
  return (tick,tickRate)=>{
    if(!Number.isFinite(tick)||!Number.isFinite(tickRate)||tickRate<=0)return [];
    const latest=new Map();
    for(const p of points) {
      const age_seconds=(tick-p.observation_tick)/tickRate;
      if(age_seconds>=0&&age_seconds<=1.5)latest.set(p.id,{...p,age_seconds});
    }
    return [...latest.values()];
  };
}
