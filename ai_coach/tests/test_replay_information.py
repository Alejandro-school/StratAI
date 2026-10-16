"""Causal information and possible reception, independent synthetic evidence."""
from copy import deepcopy

import pytest

from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_replay_factual_player_guards import factual_packet


def packet():
    data = factual_packet()
    data['schema_id'] = 'stratai.replay_context_input@7'
    data['cut_tick'] = 16
    view = data['views'][0]
    view.update(schema_id='stratai.team_replay_projection@9', tick=16)
    own = deepcopy(view['frame']['players'][0])
    own['steam_id'] = '2'
    own['source']['reference'] = 'round:1:frame:12:player:2'
    view['frame']['players'].append(own)
    ref = 'round:1:start:10:contact:12:1:6'
    observation = dict(id=ref, observer_id='1', subject_id='6', subject_side='CT',
        position={'x': -1000, 'y': 600, 'z': -40}, observation_tick=12, availability_tick=12,
        status='estimated', source={'kind': 'go_calculated_contact', 'reference': 'round:1:contact:12:1:6'},
        recipient_basis={'frame_tick': 12, 'source': {'kind': 'demo_replay_frame',
            'reference': 'round:1:frame:12:recipients:T'}, 'present_player_ids': ['1', '2']},
        recipients=[{'player_id': '1', 'kind': 'individual_observation', 'reception': 'origin', 'availability_tick': 12},
                    {'player_id': '2', 'kind': 'simulated_team_communication', 'reception': 'possible', 'availability_tick': 12}])
    contact = dict(subject_id='6', subject_side='CT', state='last_observed', position=deepcopy(observation['position']),
        observation_tick=12, availability_tick=12, age_ticks=4, observation_ids=[ref])
    view['information'] = dict(schema_id='stratai.replay_information@1', side='T', round_start_tick=10,
        communication_model='immediate_simulated_possible_reception', reception_status='possible_not_confirmed',
        own_team_state_scope='collective_demo_overview_not_individual_knowledge',
        enemy_identity_scope='opaque_demo_tracking_not_recognized_person',
        known_team_players=[{'player_id': str(i), 'team': 'T', 'observation_tick': 12,
            'source': {'kind': 'demo_replay_frame', 'reference': f'round:1:frame:12:player:{i}'}} for i in (1, 2)],
        observations=[observation], contacts=[contact], by_player=[
            {'player_id': '1', 'observation_ids': [ref]}, {'player_id': '2', 'observation_ids': [ref]}])
    view['frame']['last_seen_markers'] = [{**deepcopy(contact), 'name': 'P06', 'team': 'CT',
        'knowledge': 'historical_contact', 'current_position_status': 'unknown'}]
    return data


def test_information_input_keeps_old_contact_separate_from_live_players_and_renders_possible_receivers():
    session = ReplayReadingSession(packet())
    context = session.consultation().initial()
    memory = next(f for f in context['situation'] if f['key'] == 'enemy_information')
    assert memory['value']['state'] == 'last_observed'
    assert memory['value']['age_seconds'] == 4 / 64
    individual = [f for f in context['situation'] if f['key'] == 'possible_player_information']
    assert [f['subject'] for f in individual] == ['P01', 'P02']
    reading, verification = session.consultation().render({k: [] for k in ('situation', 'evolution', 'interpretation', 'uncertainty')})
    text = ' '.join(c['text'] for c in reading['situation'])
    assert 'posible' in text.lower() and 'P01' in text and 'P02' in text
    assert 'posición actual desconocida' in text.lower()
    assert verification['factual_integrity'] == 'checked_reference_rendering'


@pytest.mark.parametrize('mutation', [
    lambda v: v['information']['observations'][0].update(availability_tick=17),
    lambda v: v['information']['observations'][0]['recipients'][1].update(reception='confirmed'),
    lambda v: v['information']['observations'][0]['recipients'][1].update(player_id='6'),
    lambda v: v['information']['observations'][0].update(observer_id='6'),
    lambda v: v['information']['contacts'][0]['position'].update(x=123456),
    lambda v: v['information']['contacts'][0].update(state='current_contact'),
    lambda v: v['information']['contacts'][0].update(age_ticks=0),
    lambda v: v['information']['observations'][0].update(health=88),
    lambda v: v['information']['by_player'][1].update(observation_ids=[]),
    lambda v: v['frame']['last_seen_markers'][0].update(alive=True),
    lambda v: v['information']['observations'][0]['recipient_basis'].update(present_player_ids=['1']),
    lambda v: v['information'].update(enemy_identity_scope='recognized_person'),
])
def test_information_rejects_future_hidden_confirmed_or_inconsistent_claims(mutation):
    data = packet()
    mutation(data['views'][0])
    with pytest.raises((ValueError, KeyError)):
        ReplayReadingSession(data)


def test_closed_text_cannot_turn_a_possible_shared_contact_into_confirmed_knowledge():
    access = ReplayReadingSession(packet()).consultation()
    rendered, _ = access.render({k: [] for k in ('situation', 'evolution', 'interpretation', 'uncertainty')})
    changed = deepcopy(rendered)
    claim = next(c for c in changed['situation'] if c['facts'][0]['key'] == 'possible_player_information')
    claim['text'] = 'P02 conoce con certeza que CT2 sigue en corta.'
    with pytest.raises(ValueError, match='text differs'):
        access.verify(changed)


def test_information_survives_absence_of_a_teammate_without_claiming_current_resources():
    data = packet()
    view = data['views'][0]
    view['availability_tick'] = 16
    view['frame']['tick'] = 16
    view['frame']['players'].pop()
    own = view['frame']['players'][0]
    own.update(observation_tick=16, availability_tick=16,
        source={'kind': 'demo_replay_frame', 'reference': 'round:1:frame:16:player:1'})
    own['inventory_context']['status'] = 'last_observed'
    own['place'].update(observation_tick=16, availability_tick=16)
    view['frame']['round_context'].update(observation_tick=16, availability_tick=16,
        elapsed_seconds=6/64, source={'kind':'demo_round_context', 'reference':'round:1:frame:16:round_context'})
    session = ReplayReadingSession(data)
    facts = session.consultation().initial()['situation']
    assert any(f['subject'] == 'P02' and f['key'] == 'possible_player_information' for f in facts)
    assert not any(f['subject'] == 'P02' and f['key'] == 'own_player' for f in facts)


def dead_player(player):
    player.update(alive=False, health=0)
    player.pop('place', None)
    player['inventory_context'].update(status='last_observed', observation_tick=10, availability_tick=10,
        source={'kind':'demo_inventory_snapshot','reference':'frame:10:inventory'})


def test_dead_teammate_remains_a_possible_receiver_without_becoming_a_visual_observer():
    data = packet()
    dead_player(data['views'][0]['frame']['players'][1])
    session = ReplayReadingSession(data)
    info = next(f for f in session.consultation().initial()['situation'] if f['key']=='possible_player_information' and f['subject']=='P02')
    assert info['value']['observations'][0]['recipients'][1]['reception']=='possible'


def test_dead_origin_is_rejected_even_without_a_kill_callback():
    data = packet()
    dead_player(data['views'][0]['frame']['players'][0])
    with pytest.raises(ValueError, match='dead observer snapshot'):
        ReplayReadingSession(data)
