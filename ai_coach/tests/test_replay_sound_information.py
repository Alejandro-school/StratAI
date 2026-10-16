"""Sound profiles cannot authorize heard identity, enemy counts or future facts."""
from copy import deepcopy
import hashlib
import json
from pathlib import Path

import pytest

from ai_coach.tests.test_replay_utility_information import packet as utility_packet
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession

RAW = (Path(__file__).resolve().parents[2] / 'backend/go-service/pkg/sound/cs2_profiles.json').read_bytes()
CATALOG = json.loads(RAW)


def packet():
    data = utility_packet()
    data['schema_id'] = 'stratai.replay_context_input@9'
    view = data['views'][0]
    view['schema_id'] = 'stratai.team_replay_projection@11'
    p = next(p for p in CATALOG['events'] if p['event'] == 'Base.Footstep')
    ref = 'sound-observation:1'
    row = dict(id=ref, observer_id='1', kind=p['family'], profile_hash=p['hash'], source_side='CT',
        side_status='native_origin_estimate_not_recognition', region=dict(status='compatible_region_estimate',
            reference_id=data['place_reference_id'], candidates=[], build_compatibility='unverified'),
        observation_tick=12, availability_tick=12, status='estimated_possible_reception', distance_3d_units=1099,
        radius_units=p['radius_units'], radius_basis=p['radius_basis'],
        source=dict(kind='go_calculated_sound', reference=ref, emission_reference='sound-emission:1',
            native_source='net.CMsgSosStartSoundEvent', catalog_id=CATALOG['catalog_id'], catalog_sha256=hashlib.sha256(RAW).hexdigest()),
        recipient_basis=dict(frame_tick=12, present_player_ids=['1','2']),
        recipients=[dict(player_id=i, kind='individual_estimate' if i=='1' else 'simulated_team_communication',
            reception='possible', availability_tick=12) for i in ('1','2')])
    view['sound_information'] = dict(schema_id='stratai.replay_sound_information@1', side='T', round_start_tick=10,
        source_status='estimated_source_supplied', catalog_id=CATALOG['catalog_id'], catalog_sha256=hashlib.sha256(RAW).hexdigest(),
        communication_model='immediate_simulated_possible_reception', reception_status='possible_not_confirmed',
        geometry_model='native_points_euclidean_3d_no_visual_occlusion', build_compatibility='unverified',
        perception={k:'unknown' for k in ('listening','identity','enemy_count','exact_position','current_enemy_position','masking')},
        observations=[row], by_player=[dict(player_id='1',individual_signal_ids=[ref],possible_shared_signal_ids=[]),
            dict(player_id='2',individual_signal_ids=[],possible_shared_signal_ids=[ref])])
    return data


def test_sound_input_renders_possible_reception_without_identity_count_or_exact_point():
    session = ReplayReadingSession(packet())
    facts = session._archive[-1]
    sound = next(f for f in facts if f['key']=='sound_signal')
    assert sound['value']['kind']=='footstep' and sound['status']=='estimated'
    reading, verification = session.consultation().render({k:[] for k in ('situation','evolution','interpretation','uncertainty')})
    text = ' '.join(c['text'] for c in reading['situation'])
    assert 'pasos' in text.lower() and 'posible' in text.lower() and 'no confirma' in text.lower()
    assert verification['factual_integrity']=='checked_reference_rendering'
    assert 'distance_3d_units' not in json.dumps(sound) and 'profile_hash' not in json.dumps(sound)


@pytest.mark.parametrize('mutation', [
    lambda v:v['sound_information']['observations'][0].update(emitter_id='6'),
    lambda v:v['sound_information']['observations'][0].update(observation_tick=17),
    lambda v:v['sound_information']['observations'][0].update(radius_units=2000),
    lambda v:v['sound_information']['observations'][0].update(distance_3d_units=1100),
    lambda v:v['sound_information']['observations'][0].update(observer_id='6'),
    lambda v:v['sound_information']['observations'][0].update(source_side='T'),
    lambda v:v['sound_information']['observations'][0]['source'].update(catalog_sha256='0'*64),
    lambda v:v['sound_information']['observations'][0]['recipients'][1].update(reception='confirmed'),
    lambda v:v['sound_information']['perception'].update(enemy_count=2),
    lambda v:v['sound_information']['by_player'][1].update(possible_shared_signal_ids=[]),
    lambda v:v['sound_information']['observations'][0]['region'].update(exact_position={'x':1,'y':2,'z':3}),
])
def test_sound_rejects_forged_reception_profile_future_identity_and_count(mutation):
    data = packet()
    mutation(data['views'][0])
    with pytest.raises((ValueError,KeyError)):
        ReplayReadingSession(data)


def test_historical_utility_receipt_unchanged_by_sound_opt_in():
    before = ReplayReadingSession(utility_packet()).receipt()
    data = deepcopy(packet())
    data['schema_id']='stratai.replay_context_input@8'
    data['views'][0]['schema_id']='stratai.team_replay_projection@10'
    del data['views'][0]['sound_information']
    assert ReplayReadingSession(data).receipt()==before


def test_sound_context_codec_is_lossless_with_all_dated_signals():
    from ai_coach.interpretation import replay_model_context as wire
    context=ReplayReadingSession(packet()).consultation().initial()
    assert wire.unpack(wire.pack(context))==context


def test_sound_memory_cannot_disappear_without_a_newer_signal():
    from ai_coach.interpretation.replay_sound_information import checked_sound_information
    view=packet()['views'][0]
    previous={}
    checked_sound_information(view,previous)
    view['tick']=17
    view['sound_information']['observations']=[]
    for p in view['sound_information']['by_player']:
        p['individual_signal_ids']=[]
        p['possible_shared_signal_ids']=[]
    with pytest.raises(ValueError,match='memory lost'):
        checked_sound_information(view,previous)


def test_same_tick_death_blocks_direct_sound_and_silent_profile_is_unusable():
    from ai_coach.interpretation.replay_sound_information import checked_sound_information
    view=packet()['views'][0]
    view['deaths'].append(dict(victim_id='1',tick=12))
    with pytest.raises(ValueError,match='after death'):
        checked_sound_information(view,{})
    view=packet()['views'][0]
    p=next(p for p in CATALOG['events'] if p['event']=='CT_Silent.StepLeft')
    view['sound_information']['observations'][0].update(profile_hash=p['hash'])
    with pytest.raises(ValueError,match='profile or range'):
        checked_sound_information(view,{})
