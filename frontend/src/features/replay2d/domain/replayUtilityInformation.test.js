import { describe, expect, it } from 'vitest';
import { createReplayUtilityInformationProjector } from './replayUtilityInformation';

const row = (object, tick, type = 'he', kind = 'launch', observer = '1') => ({
  id: `r1-utility-observation-${object}-${tick}-${observer}`, tick, availability_tick: tick,
  observer_id: observer, observer_side: 'T', object_id: `projectile:${object}`, kind,
  utility_type: type, type_status: 'estimated_native_type_proxy', x: 100, y: 20, z: 40,
  source: kind === 'launch' ? 'demoinfocs_event:GrenadeProjectileThrow' : kind === 'projectile'
    ? 'demoinfocs_state:GrenadeProjectiles' : 'demoinfocs_event:HeExplode',
  eye_source: 'native_eye_position', position_basis: kind === 'effect' ? 'native_explosion_point' : 'native_projectile_position',
});
const fixture = observations => ({round: 1, start_tick: 100, frames: [100, 420, 424, 900].map(tick => ({tick,
  players: [{steam_id:'1',team:'T',alive:true}, {steam_id:'2',team:'T',alive:false}, {steam_id:'6',team:'CT',alive:true}]})),
  events:[], event_context:{events:[]}, utility_vision:{schema_id:'stratai.calculated_utility_vision@1',
    producer:'go.replay.utility_point_occlusion_v1',observations}});
const project = (round, rate = 64) => createReplayUtilityInformationProjector(round, 'T', rate);

describe('causal visual utility and expert launcher estimates', () => {
  it('accepts five seconds inclusively, deduplicates observers and labels count, side and location separately', () => {
    const round = fixture([row(1,100), row(1,100,'he','launch','2'), row(2,260), row(3,420)]);
    round.frames.forEach(f=>{f.players[1].alive=true;});
    const get = project(round);
    expect(get(419).launcher_estimates).toEqual([]);
    expect(get(420).launcher_estimates[0]).toMatchObject({minimum_launchers:3,status:'estimated_expert_deduction',
      first_launch_tick:100,last_launch_tick:420,side_status:'unknown',origin_status:'unknown',current_location_status:'unknown'});
    expect(get(420).launcher_estimates).toHaveLength(1);
  });
  it('excludes flashes, mixed types, six seconds and effects/projectiles without observable launch times', () => {
    for (const rows of [[row(1,100,'flashbang'),row(2,150,'flashbang'),row(3,200,'flashbang')],
      [row(1,100),row(2,150,'smoke'),row(3,200)], [row(1,100),row(2,250),row(3,421)],
      [row(1,100,'he','effect'),row(2,150,'he','projectile'),row(3,200,'he','effect')]]) {
      expect(project(fixture(rows))(900).launcher_estimates).toEqual([]);
    }
  });
  it.each(['he','smoke','molotov','incendiary'])('applies the agreed criterion to %s', type => {
    expect(project(fixture([row(1,100,type),row(2,150,type),row(3,200,type)]))(200).launcher_estimates).toHaveLength(1);
  });
  it('dead present teammates receive possible reports but cannot originate them; absent recipients are excluded', () => {
    const round = fixture([row(1,100),row(2,150,'he','launch','2')]);
    const info = project(round)(200);
    expect(info.observations).toHaveLength(1);
    expect(info.observations[0].recipients.map(r=>r.player_id)).toEqual(['1','2']);
    expect(info.by_player.find(p=>p.player_id==='2').observation_ids).toHaveLength(1);
  });
  it('does not leak native thrower IDs, enemy resources or future observations; memory stays historical on reverse seek', () => {
    const round = fixture([row(1,100),row(1,420,'he','projectile')]);
    Object.assign(round.utility_vision.observations[0],{thrower_id:'6',thrower_team:'CT',health:88});
    const get = project(round);
    expect(JSON.stringify(get(200))).not.toMatch(/thrower|health|"CT"/);
    expect(get(424).objects[0]).toMatchObject({state:'last_observed',observation_tick:420,position:{x:100,y:20,z:40}});
    expect(get(100).objects[0].observation_tick).toBe(100);
  });
  it('rejects missing/wrong provenance, nonfinite coordinates, delayed availability, dead observers and unknown tick rates', () => {
    for (const change of [{source:'untrusted'},{x:NaN},{availability_tick:101},{type_status:'confirmed'},{observer_id:'6'},{eye_source:''}]) {
      const r = row(1,100); Object.assign(r,change);
      expect(project(fixture([r]))(200).observations).toEqual([]);
    }
    expect(project(fixture([row(1,100),row(2,150),row(3,200)]),0)(200).launcher_estimates).toEqual([]);
  });
  it('excludes positively linked own launches from unknown-side launcher deductions', () => {
    const round = fixture([row(1,100),row(2,150),row(3,200)]);
    round.event_context.events=[{type:'grenade_throw',tick:100,actor_team:'T',projectile_id:'1'}];
    expect(project(round)(200).launcher_estimates).toEqual([]);
  });
  it('keeps only positively sampled segments and never bridges an occluded gap or imports a final trajectory', () => {
    const round = fixture([row(1,100,'he','projectile'),row(1,104,'he','projectile'),row(1,120,'he','projectile')]);
    round.frames[0].projectiles=[{id:'1',trajectory:[{x:9999,y:8888}]}];
    const get = project(round);
    expect(get(103).segments).toEqual([]);
    expect(get(120).segments).toHaveLength(1);
    expect(get(120).segments[0]).toMatchObject({start_tick:100,end_tick:104,interpolation_status:'between_points_not_observed'});
    expect(JSON.stringify(get(120))).not.toContain('9999');
  });
  it('preserves duplicate native flash callbacks as one object with an ambiguous point, without averaging or crashing', () => {
    const a=row(1,100,'flashbang','effect'), b=row(1,100,'flashbang','effect');
    a.source=b.source='demoinfocs_event:FlashExplode'; b.id+='-duplicate'; b.x+=0.015625;
    const info=project(fixture([a,b]))(100);
    expect(info.objects).toHaveLength(1);expect(info.observations).toHaveLength(2);
    expect(info.objects[0]).toMatchObject({position_status:'ambiguous_same_tick',position:null});
    expect(info.objects[0].points).toHaveLength(2);expect(info.launcher_estimates).toEqual([]);
  });
  it('orders retained segments by observer and object even when compacting their earlier samples changes insertion order', () => {
    const rows=[row(2,100,'he','projectile'),row(2,104,'he','projectile'),
      row(1,108,'he','projectile'),row(1,112,'he','projectile'),
      row(2,116,'he','projectile'),row(2,120,'he','projectile')];
    const info=project(fixture(rows))(120);
    expect(info.segments.map(s=>s.object_id)).toEqual(['projectile:1','projectile:2']);
    expect(info.segments.map(s=>s.end_tick)).toEqual([112,120]);
    expect(info.observations.some(r=>r.observation_tick===100)).toBe(false);
    expect(info.segments.every(s=>s.observation_ids.every(id=>info.observations.some(r=>r.id===id)))).toBe(true);
  });
});
