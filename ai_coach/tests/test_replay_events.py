from copy import deepcopy
import pytest
from ai_coach.tests.test_replay_round_context import packet as round_packet
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession

def packet():
    p=round_packet();p['schema_id']='stratai.replay_context_input@5'
    v=p['views'][0];v['schema_id']='stratai.team_replay_projection@7'
    own=next(x for x in v['frame']['players'] if x['team']=='T')
    v['events'].append({'id':'a1','type':'weapon_fire','tick':10,'availability_tick':10,
        'actor_id':str(own['steam_id']),'actor_team':'T','knowledge':'own_team_event',
        'weapon':'Glock-18','source':{'kind':'demo_atomic_event','reference':'a1','native_source':'WeaponFire'},
        'hit':True,'victims':['hidden'],'future_damage':99})
    v['frame'].update(projectiles=[],active_effects=[])
    return p

def test_events_reach_context_without_final_aggregates():
    view=ReplayReadingSession(packet()).consultation().initial()
    fire=next(e for e in view['events'] if e['key']=='weapon_fire')
    assert fire['value']['weapon']=='Glock-18'
    assert 'hit' not in fire['value'] and 'future_damage' not in fire['value'] and 'victims' not in fire['value']
    assert fire['seconds_in_window']==0

@pytest.mark.parametrize('change',[
    {'availability_tick':13},{'tick':9},{'source':{}},
    {'actor_id':'76561198000000001'},{'knowledge':'omniscient'},
    {'actor_team':'CT'},{'knowledge':'calculated_visual_event','observation':{'tick':1}},
])
def test_events_reject_future_hidden_or_unattributed_rows(change):
    p=packet();p['views'][0]['events'][-1].update(change)
    with pytest.raises(ValueError):ReplayReadingSession(p)

def test_new_extension_does_not_change_round_v4_semantics():
    p=round_packet();baseline=ReplayReadingSession(p).receipt()
    p['views'][0]['events']+=deepcopy(packet()['views'][0]['events'][-1:])
    assert ReplayReadingSession(p).receipt()['input_hash']==baseline['input_hash']
