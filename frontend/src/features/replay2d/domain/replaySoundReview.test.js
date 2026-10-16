import {describe,it,expect} from 'vitest';
import catalog from '../../../../../backend/go-service/pkg/sound/cs2_profiles.json';
import {createReplaySoundInformationProjector,createReplaySoundReviewProjector,SOUND_CATALOG_SHA256} from './replaySoundInformation';

function fixture() {
  const p=catalog.events.find(p=>p.event==='Base.Footstep');
  return {start_tick:100,events:[],frames:[{tick:100,players:[{steam_id:'1',team:'T',alive:true}]}],sound_information:{
    schema_id:'stratai.calculated_sound_information@1',producer:'go.replay.native_sound_profile_3d_v1',
    catalog_id:catalog.catalog_id,catalog_sha256:SOUND_CATALOG_SHA256,
    emissions:[{id:'sound-emission:1',tick:100,availability_tick:100,profile:p.event,family:p.family,
      source:'net.CMsgSosStartSoundEvent',native_sound_hash:p.hash,native_position:{X:0,Y:0,Z:100},
      emitter_id:'99',emitter_side:'CT',radius_units:p.radius_units,radius_basis:p.radius_basis,
      native_observers:[{player_id:'1',side:'T',alive:true,native_position:{X:0,Y:0,Z:0}},
        {player_id:'99',side:'CT',alive:true,native_position:{X:20,Y:30,Z:110}}]}],
    observations:[{id:'sound-observation:1',emission_id:'sound-emission:1',observer_id:'1',observer_side:'T',
      tick:100,availability_tick:100,distance_3d_units:100,status:'estimated_possible_reception'}]}};
}

function add(round,id,emitter='99',tick=100) {
  const e=structuredClone(round.sound_information.emissions[0]);
  Object.assign(e,{id:`sound-emission:${id}`,emitter_id:emitter,tick,availability_tick:tick});
  e.native_observers[1].player_id=emitter;
  e.native_observers[1].native_position.X=id*20;
  round.sound_information.emissions.push(e);
  round.sound_information.observations.push({...round.sound_information.observations[0],
    id:`sound-observation:${id}`,emission_id:e.id,tick,availability_tick:tick});
}

describe('demo sound points for map review, separate from model information',()=>{
  it('marks distinct native emitters even for simultaneous footsteps in the same region',()=>{
    const round=fixture();add(round,2,'98');
    const view=createReplaySoundReviewProjector(round,'T','de_mirage')(100,64);
    expect(view).toHaveLength(2);
    expect(new Set(view.map(m=>m.id)).size).toBe(2);
    expect(view.map(m=>m.x)).toEqual([20,40]);
    expect(view.every(m=>m.position_basis==='demo_emitter_at_sound_tick')).toBe(true);
  });
  it('keeps a dated emitter point, updates on sound only, and respects backward seeks and display expiry',()=>{
    const round=fixture();add(round,2,'99',116);
    round.frames.push({tick:117,players:[{steam_id:'99',team:'CT',alive:true,x:999,y:999,z:999}]});
    const project=createReplaySoundReviewProjector(round,'T','de_mirage');
    expect(project(99,64)).toEqual([]);
    expect(project(100,64)[0]).toMatchObject({x:20,y:30,z:110,age_seconds:0});
    expect(project(132,64)).toHaveLength(1);
    expect(project(132,64)[0]).toMatchObject({x:40,observation_tick:116,age_seconds:0.25});
    expect(project(213,64)).toEqual([]);
    expect(project(100,64)[0].x).toBe(20);
  });
  it('deduplicates several possible listeners of the same emitting player',()=>{
    const round=fixture(),e=round.sound_information.emissions[0];
    round.frames[0].players.push({steam_id:'2',team:'T',alive:true});
    e.native_observers.push({player_id:'2',side:'T',alive:true,native_position:{X:0,Y:0,Z:0}});
    round.sound_information.observations.push({...round.sound_information.observations[0],id:'sound-observation:2',observer_id:'2'});
    expect(createReplaySoundReviewProjector(round,'T','de_mirage')(100,64)).toHaveLength(1);
  });
  it.each(['missing-actor','unknown-side','outside-radius','dead-listener','bad-catalog','nonfinite-actor'])('abstains the map point for %s',kind=>{
    const round=fixture(),e=round.sound_information.emissions[0];
    if(kind==='missing-actor')e.native_observers.pop();
    if(kind==='unknown-side')e.emitter_side='unknown';
    if(kind==='outside-radius'){e.native_position.Z=1100;round.sound_information.observations[0].distance_3d_units=1100;}
    if(kind==='dead-listener')round.events=[{type:'kill',victim_id:'1',tick:100}];
    if(kind==='bad-catalog')round.sound_information.catalog_id='other';
    if(kind==='nonfinite-actor')e.native_observers[1].native_position.X=NaN;
    expect(createReplaySoundReviewProjector(round,'T','de_mirage')(100,64)).toEqual([]);
  });
  it('leaves the original factual/model projection byte-identical and does not expose raw emitter identity',()=>{
    const round=fixture(),before=JSON.stringify(round);
    const original=JSON.stringify(createReplaySoundInformationProjector(round,'T','de_mirage')(100));
    const markers=createReplaySoundReviewProjector(round,'T','de_mirage')(100,64);
    expect(JSON.stringify(createReplaySoundInformationProjector(round,'T','de_mirage')(100))).toBe(original);
    expect(JSON.stringify(round)).toBe(before);
    expect(JSON.stringify(markers)).not.toContain('emitter_id');
    expect(JSON.stringify(markers)).not.toContain('"99"');
    expect(original).not.toContain('demo_emitter_at_sound_tick');
  });
});
