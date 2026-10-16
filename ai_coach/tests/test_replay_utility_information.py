"""Reject forged utility facts and independently check five-second deduction."""
from copy import deepcopy

import pytest

from ai_coach.tests.test_replay_information import packet as information_packet
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.interpretation.replay_utility_information import launcher_estimates


def packet():
    data = information_packet()
    data['schema_id'] = 'stratai.replay_context_input@8'
    view = data['views'][0]
    view['schema_id'] = 'stratai.team_replay_projection@10'
    rows = []
    for i, when in enumerate((12, 14, 16), 1):
        ref = f'r1-utility-observation-{i}'
        rows.append(dict(id=ref, observer_id='1', object_id=f'projectile:{i}', kind='launch', utility_type='he',
            type_status='estimated_native_type_proxy', position={'x':100,'y':20,'z':40},
            observation_tick=when, availability_tick=when, status='estimated', affiliation='unknown',
            source=dict(kind='go_calculated_utility',reference=ref,native_source='demoinfocs_event:GrenadeProjectileThrow',
                        eye_source='native_eye_position',position_basis='native_projectile_position'),
            recipient_basis=dict(frame_tick=12,present_player_ids=['1','2']),
            recipients=[dict(player_id=p,kind='individual_observation' if p=='1' else 'simulated_team_communication',
                             reception='origin' if p=='1' else 'possible',availability_tick=when) for p in ('1','2')]))
    objects = [dict(object_id=r['object_id'],utility_type='he',kind='launch',affiliation='unknown',
                   state='current_estimate' if r['observation_tick']==16 else 'last_observed',position=r['position'],position_status='estimated_point',
                   points=[dict(position=deepcopy(r['position']),observation_ids=[r['id']])],
                   observation_tick=r['observation_tick'],age_ticks=16-r['observation_tick'],observation_ids=[r['id']]) for r in rows]
    view['frame']['utility_markers'] = deepcopy(objects)
    view['frame']['utility_segments'] = []
    view['utility_information'] = dict(schema_id='stratai.replay_utility_information@1',side='T',round_start_tick=10,
        communication_model='immediate_simulated_possible_reception',reception_status='possible_not_confirmed',
        source_status='estimated_source_supplied',observations=rows,objects=objects,segments=[],
        by_player=[dict(player_id=p,observation_ids=[r['id'] for r in rows]) for p in ('1','2')],
        launcher_estimates=launcher_estimates(rows,64))
    return data


def test_utility_input_renders_estimate_without_enemy_count_identity_or_origin():
    session = ReplayReadingSession(packet())
    facts = session._archive[-1]
    estimate = next(f for f in facts if f['key']=='utility_launcher_estimate')
    assert estimate['status']=='estimated' and estimate['value']['minimum_launchers']==3
    access = session.consultation()
    reading, _ = access.render({key:[] for key in ('situation','evolution','interpretation','uncertainty')})
    text = ' '.join(row['text'] for row in reading['situation'])
    assert 'tres lanzadores' in text and 'Bando, origen' in text and 'flashes' in text
    assert access.verify(reading)['factual_integrity']=='checked_reference_rendering'


@pytest.mark.parametrize('mutation', [
    lambda v: v['utility_information']['observations'][0].update(thrower_id='6'),
    lambda v: v['utility_information']['observations'][0].update(status='observed'),
    lambda v: v['utility_information']['observations'][0].update(availability_tick=17),
    lambda v: v['utility_information']['observations'][0].update(observer_id='6'),
    lambda v: v['utility_information']['observations'][0]['source'].update(native_source='unknown'),
    lambda v: v['utility_information']['observations'][0]['recipients'][1].update(reception='confirmed'),
    lambda v: v['utility_information']['launcher_estimates'][0].update(minimum_launchers=4),
    lambda v: v['utility_information']['launcher_estimates'][0].update(side_status='CT'),
    lambda v: v['utility_information']['launcher_estimates'][0].update(origin_status='B'),
    lambda v: v['utility_information']['launcher_estimates'][0].update(maximum_interval_seconds=2),
    lambda v: v['utility_information']['by_player'][1].update(observation_ids=[]),
    lambda v: v['frame']['utility_markers'][0]['position'].update(x=999),
])
def test_rejects_forged_counts_locations_reception_or_sources(mutation):
    data = packet()
    mutation(data['views'][0])
    with pytest.raises((ValueError,KeyError)):
        ReplayReadingSession(data)


def test_five_seconds_inclusive_and_flashes_excluded_independent_recalculation():
    rows = packet()['views'][0]['utility_information']['observations']
    for r, when in zip(rows,(12,172,332)):
        r['observation_tick']=when
    assert len(launcher_estimates(rows,64))==1
    rows[-1]['observation_tick']=333
    assert launcher_estimates(rows,64)==[]
    rows[-1]['observation_tick']=332
    for r in rows:
        r['utility_type']='flashbang'
    assert launcher_estimates(rows,64)==[]


def test_closed_text_cannot_promote_launcher_hypothesis_to_three_ct_in_b():
    access = ReplayReadingSession(packet()).consultation()
    reading, _ = access.render({key:[] for key in ('situation','evolution','interpretation','uncertainty')})
    claim = next(c for c in reading['situation'] if c['facts'][0]['key']=='utility_launcher_estimate')
    claim['text']='Hay tres CT en B.'
    with pytest.raises(ValueError,match='text differs'):
        access.verify(reading)


def test_new_input_keeps_sealed_local_model_and_reference_only_output():
    from ai_coach.tests.test_context_comparison_ollama import FakeClient
    from ai_coach.interpretation.context_comparison_ollama import interpret, verify_replay, MODEL, DIGEST
    session = ReplayReadingSession(packet())
    answer = {'action':'finish','indices':[], 'reading':{key:[] for key in ('situation','evolution','interpretation','uncertainty')}}
    client = FakeClient([answer])
    result = interpret(session, client)
    assert result['status']=='mechanically_valid'
    assert client.config.model == MODEL == 'qwen3.5:9b' and client.config.digest == DIGEST
    assert result['local_calls']==1 and result['external_calls']==0
    assert verify_replay(session, result)['causal_replay']


def test_duplicate_flash_callback_is_one_signal_with_ambiguous_position_and_original_points():
    data = packet()
    view = data['views'][0]
    rows = view['utility_information']['observations'][:2]
    for i, row in enumerate(rows):
        row.update(object_id='flash:179:tick:16', kind='effect', utility_type='flashbang', observation_tick=16,availability_tick=16)
        row['source'].update(native_source='demoinfocs_event:FlashExplode',position_basis='native_explosion_point')
        row['position']['x'] += i * 0.015625
        for receipt in row['recipients']:
            receipt['availability_tick']=16
    obj = dict(object_id=rows[0]['object_id'],utility_type='flashbang',kind='effect',affiliation='unknown',
        state='current_estimate',position_status='ambiguous_same_tick',position=None,
        points=[dict(position=deepcopy(r['position']),observation_ids=[r['id']]) for r in rows],
        observation_tick=16,age_ticks=0,observation_ids=[r['id'] for r in rows])
    view['utility_information'].update(observations=rows,objects=[obj],launcher_estimates=[],
        by_player=[dict(player_id=p,observation_ids=[r['id'] for r in rows]) for p in ('1','2')])
    view['frame']['utility_markers']=[deepcopy(obj)]
    session=ReplayReadingSession(data)
    reading,_=session.consultation().render({key:[] for key in ('situation','evolution','interpretation','uncertainty')})
    assert any('posición ambigua' in c['text'] for c in reading['situation'])
    view['utility_information']['objects'][0]['position']=deepcopy(rows[0]['position'])
    with pytest.raises(ValueError,match='memory or markers'):
        ReplayReadingSession(data)


def test_segment_order_is_canonical_when_compact_samples_have_different_chronology():
    from ai_coach.interpretation.replay_utility_information import checked_utility_information
    view = packet()['views'][0]
    view['tick'] = 18
    template = view['utility_information']['observations'][0]
    rows = []
    for index, (obj, tick) in enumerate(((2,12),(1,14),(2,16),(1,18)), 1):
        row = deepcopy(template)
        row.update(id=f'utility-{index}', object_id=f'projectile:{obj}', kind='projectile',
                   observation_tick=tick, availability_tick=tick)
        row['source'].update(reference=row['id'], native_source='demoinfocs_state:GrenadeProjectiles')
        for receipt in row['recipients']:
            receipt['availability_tick'] = tick
        rows.append(row)
    segments, objects = [], []
    for obj in (1,2):
        before, after = [r for r in rows if r['object_id']==f'projectile:{obj}']
        segments.append(dict(object_id=after['object_id'],observer_id='1',
            start_tick=before['observation_tick'],end_tick=after['observation_tick'],
            status='estimated_sampled_segment',observation_ids=[before['id'],after['id']],
            interpolation_status='between_points_not_observed',to=after['position'],**{'from':before['position']}))
        objects.append(dict(object_id=after['object_id'],utility_type='he',kind='projectile',affiliation='unknown',
            state='current_estimate' if after['observation_tick']==18 else 'last_observed',
            position_status='estimated_point',position=after['position'],
            points=[dict(position=after['position'],observation_ids=[after['id']])],
            observation_tick=after['observation_tick'],age_ticks=18-after['observation_tick'],observation_ids=[after['id']]))
    view['utility_information'].update(observations=rows,objects=objects,segments=segments,launcher_estimates=[],
        by_player=[dict(player_id=p,observation_ids=['utility-3','utility-4']) for p in ('1','2')])
    view['frame'].update(utility_markers=deepcopy(objects),utility_segments=deepcopy(segments))
    assert checked_utility_information(view,64,{})['segments'] == segments
    view['utility_information']['segments'][0]['end_tick'] = 19
    with pytest.raises(ValueError,match='sampled segments'):
        checked_utility_information(view,64,{})
