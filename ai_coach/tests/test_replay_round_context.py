from copy import deepcopy
import pytest
from ai_coach.tests.test_replay_reading_context import c4_packet
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.interpretation.place_volumes import load_reference, resolve

def packet():
    p = c4_packet()
    p.update(schema_id='stratai.replay_context_input@4', map_name='de_mirage', place_reference_id=load_reference()['reference_id'])
    v=p['views'][0]
    for player in v['frame']['players']:
        tick = player.get('observation_tick', 10)
        player.update(observation_tick=tick, availability_tick=tick)
        precision = .5 if player['team'] == 'T' else 0
        player['place'] = {**resolve(player, load_reference(), map_name='de_mirage', xy_precision=precision),
                           'observation_tick':tick, 'availability_tick':tick, 'xy_precision':precision}
    v.update(schema_id='stratai.team_replay_projection@6',place_reference_id=p['place_reference_id'],round=1)
    v['frame']['time_remaining']=None
    v['frame']['round_context']={
        'schema_id':'stratai.replay_round_context@1','round':1,'observation_tick':10,'availability_tick':10,
        'phase':'ended','freeze_end_tick':10,'elapsed_seconds':0,'end_tick':10,'winner':'T',
        'elapsed_kind':'demo_ticks_since_freeze_end_not_active_game_clock',
        'source':{'kind':'demo_round_context','reference':'round:1:frame:10:round_context'},
        'configured_duration_seconds':115,'pause_state':'unknown','clock_status':'unavailable',
        'clock_reason':'sample_not_current','round_remaining_seconds':None,'bomb_remaining_seconds':None,
        'eligibility':{'kind':'bomb_objective','status':'observed','reason':'causal_objective_evidence','basis_tick':10},
        'decision_status':'round_closed'}
    return p

def test_ended_round_is_explicit_and_legacy_countdown_cannot_return():
    p=packet()
    view=ReplayReadingSession(p).consultation().initial()
    context=next(f for f in view['situation'] if f['key']=='round_context')['value']
    assert context['phase']=='ended' and context['decision_status']=='round_closed'
    clock=next(f for f in view['situation'] if f['key']=='round_remaining_seconds')
    assert clock['value'] is None and clock['status']=='unknown'
    assert view['native_status']=='round_context_supplied'

@pytest.mark.parametrize('change',[
    {'availability_tick':13}, {'end_tick':13}, {'round_remaining_seconds':100},
    {'elapsed_seconds':10}, {'winner':'CT','phase':'preplant'}, {'source':{}},
])
def test_round_input_rejects_future_inconsistent_or_unattributed_context(change):
    p=packet();p['views'][0]['frame']['round_context'].update(change)
    with pytest.raises(ValueError): ReplayReadingSession(p)

def test_original_input_is_unchanged_when_new_source_is_not_selected():
    p=c4_packet(); baseline=ReplayReadingSession(p).receipt()
    p['views'][0]['frame']['round_context']=deepcopy(packet()['views'][0]['frame']['round_context'])
    assert ReplayReadingSession(p).receipt()['input_hash']==baseline['input_hash']
