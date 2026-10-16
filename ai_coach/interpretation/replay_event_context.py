"""Whitelist dated perspective events; never enrich them from final summaries."""
from copy import deepcopy
import math
from ai_coach.support.anchors import require

ATOMIC = ('weapon_fire','player_hurt','grenade_throw','utility_start','utility_end')

def local_id(value):
    return str(value).isdigit() and 1 <= int(value) <= 10

def checked_events(view, start):
    own={str(p['steam_id']) for p in view['frame']['players'] if p['team']=='T'}
    result=[]
    for e in view.get('events',[]):
        kind=e['type']
        if kind not in ATOMIC and kind!='kill':continue
        require(type(e.get('tick')) is int and start<=e['tick']<=e.get('availability_tick',-1)<=view['tick'], 'future event')
        require(e.get('id') and e.get('source',{}).get('reference'), 'event source absent')
        actor=e.get('actor_id');victim=e.get('victim_id')
        for value in (actor,victim,e.get('killer_id')):
            require(value is None or local_id(value), 'nonlocal event participant')
        if kind=='kill':
            require(e.get('knowledge')=='public_killfeed' and victim is not None, 'unprojected killfeed')
            keys=('id','type','tick','availability_tick','victim_id','victim_team','killer_id','killer_team','weapon','knowledge','source')
        else:
            require(e['source'].get('kind')=='demo_atomic_event' and e['source'].get('native_source'), 'event callback absent')
            knowledge=e.get('knowledge')
            require(knowledge in ('own_team_event','calculated_visual_event'), 'unprojected event')
            if kind=='player_hurt':
                require(str(victim) in own and e.get('victim_team')=='T', 'hidden victim health')
                for k in ('damage','health'):
                    require(type(e.get(k)) in (int,float) and math.isfinite(e[k]) and e[k]>=0, 'invalid damage')
            else:require(actor is not None, 'event actor absent')
            if actor is not None:
                if str(actor) in own:
                    require(e.get('actor_team')=='T' and knowledge=='own_team_event', 'own actor differs')
                else:
                    observation=e.get('observation',{})
                    observed=observation.get('tick')
                    require(knowledge=='calculated_visual_event' and e.get('actor_team')=='CT'
                        and str(observation.get('observer_id')) in own and type(observed) is int
                        and start<=observed<=e['tick']<observed+4 and observation.get('source',{}).get('reference'), 'unobserved enemy event')
                    require(kind not in ('utility_start','utility_end'), 'hidden enemy effect')
            keys=('id','type','tick','availability_tick','actor_id','actor_team','x','y','z','position_tick','knowledge','source','observation')
            if kind=='player_hurt':keys+=('victim_id','victim_team','damage','health')
            if kind in ('weapon_fire','player_hurt') and actor is not None:keys+=('weapon',)
            if kind in ('grenade_throw','utility_start','utility_end'):keys+=('utility_type',)
            if kind in ('grenade_throw','utility_start','utility_end') and str(actor) in own:keys+=('projectile_id','effect_id')
        clean={k:deepcopy(e[k]) for k in keys if k in e}
        if actor is None:clean.pop('actor_team',None)
        if str(actor) in own or actor is None:clean.pop('observation',None)
        clean['source']={k:e['source'][k] for k in ('kind','reference','native_source') if isinstance(e['source'].get(k),str)}
        if 'observation' in clean:
            o=e['observation'];clean['observation']={k:o[k] for k in ('observer_id','tick')}
            clean['observation']['source']={k:o['source'][k] for k in ('kind','reference') if isinstance(o['source'].get(k),str)}
        result.append(clean)
    return result

def checked_utility(view, events):
    result={}
    own={str(p['steam_id']) for p in view['frame']['players'] if p['team']=='T'}
    for channel in ('projectiles','active_effects'):
        rows=[]
        for u in view['frame'].get(channel,[]):
            require(str(u.get('thrower_id')) in own and u.get('knowledge')=='simulated_team_utility_tracking', 'hidden utility')
            require(u.get('source',{}).get('reference') and u.get('observation_tick')==u.get('availability_tick')==view['availability_tick'], 'utility source differs')
            require(all(type(u.get(k)) in (float,int) and math.isfinite(u[k]) for k in ('x','y')), 'utility position absent')
            if channel=='projectiles':
                origins=[e for e in events if e['type']=='grenade_throw' and str(e.get('projectile_id'))==str(u.get('id'))]
                require(not any(e['type'] in ('utility_start','utility_end') and str(e.get('projectile_id'))==str(u.get('id')) for e in events), 'projectile after detonation')
            else:
                origins=[e for e in events if e['type']=='utility_start' and e['tick']==u.get('start_tick')
                    and (e.get('effect_id')==u['effect_id'] if u.get('effect_id') else e.get('projectile_id')==u.get('projectile_id'))]
            require(any(e.get('actor_id')==str(u['thrower_id']) for e in origins), 'utility origin absent')
            if channel=='active_effects':
                require(not any(e['type']=='utility_end' and (e.get('effect_id')==u['effect_id'] if u.get('effect_id') else e.get('projectile_id')==u.get('projectile_id')) for e in events), 'effect after expiration')
            for k in ('trajectory','hull','points'):
                if k in u:require(isinstance(u[k],list) and len(u[k])%2==0 and all(type(n) in (float,int) and math.isfinite(n) for n in u[k]), 'invalid utility geometry')
            keys=('id','type','thrower_id','projectile_id','effect_id','x','y','z','trajectory','hull','points','radius',
                'start_tick','observation_tick','availability_tick','knowledge','source')
            clean={k:deepcopy(u[k]) for k in keys if k in u}
            clean['source']={k:u['source'][k] for k in ('kind','reference') if isinstance(u['source'].get(k),str)}
            rows.append(clean)
        result[channel]=rows
    return result
