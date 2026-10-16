"""Kill identity, callback weapon/position and causal review cuts."""
from copy import deepcopy

import pytest

from ai_coach.interpretation.kill_context import actor_alias, project

SHA='a'*64


def fixture():
    def player(native,side,alive):
        return dict(actor_alias=actor_alias(SHA,native),side=side,connected=True,
                    pawn_handle=10 if side=='T' else 20,pawn_life=side+'-life',
                    identity_status='observed',alive=alive,alive_status='observed')
    frames=[dict(sequence=i+1,tick=100+4*i,tick_rate=64,round_number=1,instance=2,epoch=0,
                 players=[player('steam:10','T',i==0),player('steam:20','CT',True)]) for i in range(3)]
    support=dict(source_alias='demo-test',source_sha256=SHA,observations={'frames':frames})
    card={'data':{'frames':[{'round':1,'continuity':'adjacent_samples','actors':[
        {'label':'T1','identity':{k:f['players'][0][k] for k in ['side','pawn_handle','pawn_life','identity_status']},'alive':i==0}
    ]} for i,f in enumerate(frames)]},'round-data':{'frames':[
        {'source_sequence':i+1,'eligibility':{'kind':'bomb_objective'},'round_elapsed_seconds':{'status':'observed','value':i/16},'native_clock':{'pause_state':{'value':'running'}}}
        for i in range(3)]}}
    event=dict(event_id='synthetic-kill',round_number=1,tick=102,sequence_in_tick=1,tick_status='observed',source='demoinfocs.kill',
        actor_player_id='steam:20',actor_side='ct',actor_status='observed',target_player_id='steam:10',target_side='t',target_status='observed',
        actor_position={'x':10,'y':20,'z':30},actor_position_status='observed',actor_position_source='callback_player_position',
        target_position={'x':1,'y':2,'z':3},target_position_status='observed',target_position_source='callback_player_position',
        weapon='AWP',weapon_status='observed',weapon_source='callback_weapon',weapon_is_utility=False,relation='enemy',through_smoke=True,penetrated_objects=1)
    refs=dict(support_sha256=SHA,canonical_manifest_sha256=SHA,combat_sha256=SHA,combat_record_count=1)
    return support,[event],card,refs


def result(s,e,c,r):
    return project(s,e,c,'E03',r)


def test_callback_identity_weapon_position_and_preceding_clock():
    value=result(*fixture());e=value['events'][0]
    assert e['attacker']['label']=='CT1' and e['victim']['label']=='T1'
    assert e['weapon']=='AWP' and e['attacker']['position']=={'x':10,'y':20,'z':30}
    assert e['available_index']==1 and e['available_sequence']==2
    assert e['elapsed_seconds']==2/64
    assert e['through_smoke'] and e['penetrated_objects']==1
    assert not value['perception_proven']
    assert 'steam:' not in str(value)


def test_future_kills_do_not_enter_prefix_and_duplicate_ids_fail():
    s,e,c,r=fixture();future={**e[0],'event_id':'future','tick':110}
    assert result(s,e+[future],c,r)==result(s,e,c,r)
    with pytest.raises(ValueError,match='duplicate'):
        result(s,e+e,c,r)
    prefix=deepcopy(c)
    for block in prefix.values():block['frames']=block['frames'][:1]
    assert result(s,e,prefix,r)['events']==[]


def test_missing_weapon_position_clock_are_not_filled_from_future():
    s,e,c,r=fixture();e[0].update(weapon_status='unavailable',actor_position_status='unavailable')
    c['round-data']['frames'][0]['round_elapsed_seconds']={'status':'unavailable','value':None}
    value=result(s,e,c,r)['events'][0]
    assert value['weapon'] is None and value['attacker']['position'] is None and value['elapsed_seconds'] is None


@pytest.mark.parametrize('mode',['gap','life','alive','knife'])
def test_incompatible_cuts_are_not_joined(mode):
    s,e,c,r=fixture()
    if mode=='gap':c['data']['frames'][1]['continuity']='unavailable'
    if mode=='life':
        s['observations']['frames'][1]['players'][0]['pawn_life']='other'
        c['data']['frames'][1]['actors'][0]['identity']['pawn_life']='other'
    if mode=='alive':
        s['observations']['frames'][1]['players'][0]['alive']=True
        c['data']['frames'][1]['actors'][0]['alive']=True
    if mode=='knife':c['round-data']['frames'][1]['eligibility']['kind']='knife'
    value=result(s,e,c,r)
    assert not value['events'] and sum(value['omissions'].values())==1


def test_world_and_suicide_are_not_invented_enemy_attackers():
    s,e,c,r=fixture();e[0].update(actor_player_id=None,actor_status='unavailable',relation='unknown',weapon='World')
    value=result(s,e,c,r)['events'][0]
    assert value['attacker'] is None and value['weapon']=='World'
    e[0].update(actor_player_id='steam:10',actor_side='t',actor_status='observed',relation='self')
    assert result(s,e,c,r)['events'][0]['attacker']['label']=='T1'
