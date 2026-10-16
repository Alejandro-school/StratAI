import {describe,it,expect} from 'vitest';
import catalog from '../../../../../backend/go-service/pkg/sound/cs2_profiles.json';
import {createReplaySoundInformationProjector,SOUND_CATALOG_SHA256} from './replaySoundInformation';

const fixture=()=>{
  const p=catalog.events.find(p=>p.event==='Base.Footstep');
  return {round:1,start_tick:100,events:[],frames:[{tick:100,players:[
    {steam_id:'1',team:'T',alive:true},{steam_id:'2',team:'T',alive:false}]}],sound_information:{
    schema_id:'stratai.calculated_sound_information@1',producer:'go.replay.native_sound_profile_3d_v1',catalog_id:catalog.catalog_id,catalog_sha256:SOUND_CATALOG_SHA256,
    emissions:[{id:'sound-emission:1',tick:100,availability_tick:100,profile:p.event,family:p.family,
      source:'net.CMsgSosStartSoundEvent',native_sound_hash:p.hash,native_position:{X:0,Y:0,Z:1099},
      emitter_id:'99',emitter_side:'CT',radius_units:p.radius_units,radius_basis:p.radius_basis,
      native_observers:[{player_id:'1',side:'T',alive:true,native_position:{X:0,Y:0,Z:0}}]}],
    observations:[{id:'sound-observation:1',emission_id:'sound-emission:1',observer_id:'1',observer_side:'T',
      tick:100,availability_tick:100,distance_3d_units:1099,status:'estimated_possible_reception'}]}};
};

describe('estimated spatial sound information',()=>{
  it('keeps the native side as an estimate, strips hidden identities/points, and shares possibly with dead teammates',()=>{
    const view=createReplaySoundInformationProjector(fixture(),'T','de_mirage')(100);
    expect(view.observations).toHaveLength(1);
    expect(view.observations[0].source_side).toBe('CT');
    expect(view.observations[0].recipients.map(r=>r.player_id)).toEqual(['1','2']);
    expect(view.by_player[1].possible_shared_signal_ids).toEqual(['sound-observation:1']);
    const raw=JSON.stringify(view);
    for(const hidden of ['emitter_id','native_position','native_observers','"99"']) expect(raw).not.toContain(hidden);
    expect(view.perception.enemy_count).toBe('unknown');
    expect(view.perception.exact_position).toBe('unknown');
  });
  it.each(['height','radius','distance','future','death','local','silent','volume','self','catalog','boundary'])('rejects %s leakage or mismatched evidence',kind=>{
    const round=fixture(),e=round.sound_information.emissions[0],o=round.sound_information.observations[0];
    if(kind==='height') e.native_position.Z=1200;
    if(kind==='radius') e.radius_units=2000;
    if(kind==='distance') o.distance_3d_units=10;
    if(kind==='future') {e.tick=101;e.availability_tick=101;o.tick=101;o.availability_tick=101;}
    if(kind==='death') round.events=[{type:'kill',victim_id:'1',tick:100}];
    if(kind==='silent') {const p=catalog.events.find(p=>p.event==='CT_Silent.StepLeft');Object.assign(e,{profile:p.event,family:p.family,native_sound_hash:p.hash,radius_units:p.radius_units,radius_basis:p.radius_basis});}
    if(kind==='volume') e.native_volume=0;
    if(kind==='self') {e.emitter_side='unknown';e.emitter_id='1';}
    if(kind==='catalog') round.sound_information.catalog_sha256='0'.repeat(64);
    if(kind==='boundary') {e.native_position.Z=1100;o.distance_3d_units=1100;}
    if(kind==='local') {const p=catalog.events.find(p=>p.event==='HEGrenade.PullPin_Grenade');Object.assign(e,{profile:p.event,family:p.family,native_sound_hash:p.hash,radius_units:p.radius_units,radius_basis:p.radius_basis});}
    expect(createReplaySoundInformationProjector(round,'T','de_mirage')(100).observations).toHaveLength(0);
  });
  it('does not grant identity or count from repeated emissions, and keeps latest dated memory',()=>{
    const round=fixture();
    for(let i=2;i<=3;i++) {
      round.sound_information.emissions.push({...structuredClone(round.sound_information.emissions[0]),id:`sound-emission:${i}`,tick:99+i,availability_tick:99+i});
      round.sound_information.observations.push({...structuredClone(round.sound_information.observations[0]),id:`sound-observation:${i}`,emission_id:`sound-emission:${i}`,tick:99+i,availability_tick:99+i});
    }
    const project=createReplaySoundInformationProjector(round,'T','de_mirage');
    expect(project(100).observations[0].observation_tick).toBe(100);
    const later=project(102);
    expect(later.by_player[0].individual_signal_ids).toEqual(['sound-observation:3']);
    expect(later.perception.enemy_count).toBe('unknown');
  });
  it('keeps one latest footstep category rather than treating native material profiles as separate perceived information',()=>{
    const round=fixture(),p=catalog.events.find(p=>p.event==='CT_Concrete.StepLeft');
    round.sound_information.emissions.push({...structuredClone(round.sound_information.emissions[0]),id:'sound-emission:2',tick:101,availability_tick:101,profile:p.event,native_sound_hash:p.hash,radius_units:p.radius_units,radius_basis:p.radius_basis});
    round.sound_information.observations.push({...structuredClone(round.sound_information.observations[0]),id:'sound-observation:2',emission_id:'sound-emission:2',tick:101,availability_tick:101});
    const view=createReplaySoundInformationProjector(round,'T','de_mirage')(101);
    expect(view.by_player[0].individual_signal_ids).toEqual(['sound-observation:2']);
    expect(view.observations).toHaveLength(1);
    expect(round.sound_information.emissions).toHaveLength(2);
  });
});
