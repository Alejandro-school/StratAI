from copy import deepcopy
import pytest
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession, VERSION, PROJECTION

def packet():
    own={"steam_id":"1","team":"T","alive":True,"x":10,"y":20,"health":80,"knowledge":"own_team_snapshot"}
    rival={"steam_id":"2","team":"CT","alive":True,"x":30,"y":40,"health":100,
           "knowledge":"calculated_visual_contact","availability_tick":12,"observation_tick":12}
    view={"schema_id":PROJECTION,"side":"T","tick":12,"availability_tick":10,
          "vision_status":"calculated_contact_visible","uncertainty":{"enemy_resources":"unknown"},
          "frame":{"players":[own,rival],"bomb":None,"time_remaining":110},"deaths":[]}
    return {"schema_id":VERSION,"side":"T","tick_rate":64,"start_tick":10,"cut_tick":12,"views":[view]}

def test_replay_input_preserves_aliases_and_does_not_ingest_hidden_enemy_resources():
    session=ReplayReadingSession(packet());view=session.consultation().initial()
    contact=next(f for f in view['situation'] if f['key']=='enemy_contact')
    assert contact['subject']=='P02' and contact['status']=='estimated'
    assert 'health' not in contact['value']
    assert view['prior_rounds_status']=='not_supplied'

@pytest.mark.parametrize('change',[lambda p:p['views'][0].update(tick=13),
    lambda p:p['views'][0]['frame']['players'][1].update(knowledge='raw_snapshot'),
    lambda p:p['views'][0]['frame']['players'][1].update(availability_tick=14),
    lambda p:p['views'][0].update(availability_tick=14)])
def test_replay_input_rejects_future_or_unprojected_data(change):
    value=packet();change(value)
    with pytest.raises(ValueError):ReplayReadingSession(value)

def test_consultation_has_causal_bounds_and_death_identity_and_time():
    value=packet();death={'tick':11,'availability_tick':11,'victim_id':'3','team':'CT','position':{'x':1,'y':2}}
    value['views'][0]['deaths']=[death]
    session=ReplayReadingSession(value);access=session.consultation()
    event=access.initial()['events'][0]
    assert event['subject']=='P03' and event['value']['seconds_in_window']==1/64
    with pytest.raises(ValueError):access.sample(1)
    changed=deepcopy(value);changed['views'][0]['deaths'][0]['tick']=13
    with pytest.raises(ValueError):ReplayReadingSession(changed)

def c4_packet():
    value=packet()
    value['schema_id']='stratai.replay_context_input@2'
    view=value['views'][0]
    view['schema_id']='stratai.team_replay_projection@4'
    view['frame']['players'][0]['has_c4']=False
    view['frame']['bomb']={'state':'dropped','objective_phase':'preplant',
        'position_status':'unavailable','observation_tick':10,'availability_tick':10,
        'source':{'reference':'round:3:frame:10:bomb','kind':'demo_objective_state'}}
    view['events']=[{'id':'drop-1','type':'bomb_drop','tick':11,'availability_tick':11,
        'player_id':'1','knowledge':'own_team_objective_event',
        'source':{'reference':'objective-drop-1','kind':'demoinfocs_event'}}]
    return value

def test_c4_input_preserves_state_carrier_flag_and_dated_evolution_without_position():
    view=ReplayReadingSession(c4_packet()).consultation().initial()
    bomb=next(f for f in view['situation'] if f['key']=='objective')
    assert bomb['status']=='observed' and bomb['value']['state']=='dropped'
    assert 'x' not in bomb['value'] and bomb['value']['position_status']=='unavailable'
    own=next(f for f in view['situation'] if f['key']=='own_player')
    assert own['value']['has_c4'] is False
    drop=next(f for f in view['events'] if f['key']=='bomb_drop')
    assert drop['subject']=='P01' and drop['seconds_in_window']==1/64
    assert drop['value']['source']['reference']=='objective-drop-1'
    assert view['version']=='stratai.replay_context_input@2'

@pytest.mark.parametrize('change',[
    lambda p:p['views'][0]['frame']['bomb'].update(availability_tick=13),
    lambda p:p['views'][0]['frame']['bomb'].update(x=99,y=88,position_status='observed',position_tick=10),
    lambda p:p['views'][0]['events'][0].update(tick=13),
    lambda p:p['views'][0]['events'][0].update(availability_tick=13),
    lambda p:p['views'][0]['frame']['players'][0].update(has_c4=True),
])
def test_c4_input_rejects_future_stale_geometry_or_conflicting_carrier(change):
    value=c4_packet();change(value)
    with pytest.raises(ValueError): ReplayReadingSession(value)

@pytest.mark.parametrize('position',[
    {'x':1,'y':2,'observation_tick':13,'source':{'reference':'future'}},
    {'x':1,'y':2,'observation_tick':10,'source':{}},
    {'x':float('nan'),'y':2,'observation_tick':10,'source':{'reference':'sample'}},
])
def test_c4_input_rejects_invalid_historical_geometry(position):
    value=c4_packet();value['views'][0]['frame']['bomb']['last_observed_position']=position
    with pytest.raises(ValueError): ReplayReadingSession(value)
