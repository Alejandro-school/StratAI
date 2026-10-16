"""The first expert-framed hypothesis never owns numeric or objective facts."""
from copy import deepcopy
import math

import jsonschema
import pytest

from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation.replay_attack_hypothesis import AttackHypothesisSession, HYPOTHESIS
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.interpretation.place_volumes import load_reference, resolve
from ai_coach.tests.test_context_comparison_ollama import FakeClient
from ai_coach.tests.test_replay_factual_player_guards import factual_packet


def packet(count=2):
    data = factual_packet()
    view = data['views'][0]
    view['events'] = []
    view['frame']['round_context'].update(phase='preplant', end_tick=None, winner='', decision_status='active_round')
    view['frame']['bomb'].update(state='dropped', objective_phase='preplant')
    first = view['frame']['players'][0]
    view['frame']['players'] = []
    for i in range(count):
        player = deepcopy(first)
        player.update(steam_id=str(i+1), x=-1500+i*10)
        player['source']['reference'] = f'round:1:frame:12:player:{i+1}'
        view['frame']['players'].append(player)
    return data


def selection(access, count=2):
    result = {key: [] for key in ('situation', 'evolution', 'interpretation', 'uncertainty')}
    ids = [row['player_fact_id'] for row in access.initial()['hypothesis_contract']['eligible_players']]
    result['interpretation'] = [{'hypothesis_id': HYPOTHESIS, 'evidence': ids[:count]}]
    return result


@pytest.mark.parametrize('count', [2, 3])
def test_group_is_a_dated_possible_attack_with_exact_count_and_original_facts(count):
    data = packet(count)
    original = ReplayReadingSession(data)
    session = AttackHypothesisSession(data)
    access = session.consultation()
    assert access.initial()['situation'] == original.consultation().initial()['situation']
    assert session.receipt()['projected_packet_hash'] == original.receipt()['projected_packet_hash']
    assert session.receipt()['input_hash'] != original.receipt()['input_hash']
    reading, verified = access.render(selection(access, count))
    plain, _ = original.consultation().render({key: [] for key in reading})
    assert all(reading[key] == plain[key] for key in ('situation', 'evolution', 'uncertainty'))
    claim = reading['interpretation'][0]
    assert claim['kind'] == 'inferred' and f'{count} T' in claim['text']
    assert 'Posible ataque a B' in claim['text'] and 'No confirma intención de plantar' in claim['text']
    assert claim['hypothesis']['player_fact_ids'] == selection(access, count)['interpretation'][0]['evidence']
    assert len([f for f in claim['facts'] if f['key'] == 'own_player']) == count
    assert any(f['key'] == 'objective' for f in claim['facts'])
    assert access.verify(reading) == verified
    assert verified['tactical_quality'] == 'not_evaluated'


def test_geometry_is_recomputed_without_promoting_distance_to_a_tactical_threshold():
    access = AttackHypothesisSession(packet()).consultation()
    contract = access.initial()['hypothesis_contract']
    assert contract['advanced_reference']['position_xy'] == [-1500, 750]
    assert contract['spatial_classification'] == 'model_estimate_no_metric_threshold'
    assert [r['distance_xy_units'] for r in contract['eligible_players']] == [0, 10]
    for row in contract['eligible_players']:
        point = access._facts[row['player_fact_id']]['value']
        assert row['distance_xy_units'] == math.hypot(point['x']+1500, point['y']-750)


@pytest.mark.parametrize('bad_evidence', [[], ['S0:3'], ['S0:3', 'S0:3'], ['S0:3', 'S0:2'], ['future', 'S0:3']])
def test_unknown_duplicate_nonplayer_or_too_few_references_cannot_count_players(bad_evidence):
    access = AttackHypothesisSession(packet()).consultation()
    chosen = selection(access)
    chosen['interpretation'][0]['evidence'] = bad_evidence
    with pytest.raises(ValueError):
        access.render(chosen)


@pytest.mark.parametrize('count', [1, 4])
def test_player_count_cannot_be_extended_to_unframed_rules(count):
    access = AttackHypothesisSession(packet(count)).consultation()
    with pytest.raises(ValueError):
        access.render(selection(access, count))


def test_ended_round_never_offers_or_renders_a_current_attack():
    data = packet()
    data['views'][0]['frame']['round_context'].update(phase='ended', end_tick=12, winner='CT', decision_status='round_closed')
    data['views'][0]['frame']['bomb'].update(state='resolved', objective_phase='resolved')
    access = AttackHypothesisSession(data).consultation()
    assert access.initial()['hypothesis_contract']['eligible_players'] == []
    with pytest.raises(ValueError):
        access.render(selection(access))


@pytest.mark.parametrize('variant', ['dead', 'stale', 'other_zone', 'unknown_height', 'boundary', 'planting'])
def test_missing_current_unambiguous_own_position_cannot_support_the_group(variant):
    data = packet()
    view = data['views'][0]
    player = view['frame']['players'][1]
    if variant == 'dead':
        player.update(alive=False, health=0)
        player.pop('place')
        player['inventory_context'].update(status='last_observed', observation_tick=11, availability_tick=11)
        player['inventory_context']['source']['reference'] = 'frame:11:inventory'
    elif variant == 'stale':
        player.update(observation_tick=11, availability_tick=11)
        player['place'].update(observation_tick=11, availability_tick=11)
        player['inventory_context'].update(status='last_observed', observation_tick=11, availability_tick=11)
    elif variant in ('other_zone', 'unknown_height', 'boundary'):
        if variant == 'other_zone':
            player.update(x=0, y=0)
        elif variant == 'unknown_height':
            player.update(z=500)
        else:
            player.update(x=-1024, y=600)
        player['place'] = {**resolve(player, load_reference(), map_name='de_mirage', xy_precision=.5),
                           'observation_tick':12, 'availability_tick':12, 'xy_precision':.5}
    else:
        view['frame']['bomb'].update(state='planting')
    if variant == 'stale':
        with pytest.raises(ValueError, match='own snapshot date differs'):
            AttackHypothesisSession(data)
        return
    access = AttackHypothesisSession(data).consultation()
    assert len(access.initial()['hypothesis_contract']['eligible_players']) < 2
    with pytest.raises(ValueError):
        access.render(selection(access))


def test_old_conditional_principles_and_abstention_remain_available():
    from ai_coach.interpretation.context_comparison import BANK
    access = AttackHypothesisSession(packet()).consultation()
    empty = {key: [] for key in ('situation', 'evolution', 'interpretation', 'uncertainty')}
    assert access.render(empty)[0]['interpretation'] == []
    empty['interpretation'] = [{'principle_id': BANK[0]['id'], 'evidence': ['S0:3']}]
    reading, verified = access.render(empty)
    assert reading['interpretation'][0]['kind'] == 'limit'
    assert access.verify(reading) == verified


@pytest.mark.parametrize('mutation', [
    lambda c: c.update(text='Hay dos C4 y los T van a plantar.'),
    lambda c: c['facts'][0]['value'].update(x=123456),
    lambda c: c['hypothesis'].update(id='plant_b_confirmed'),
    lambda c: c['hypothesis']['player_fact_ids'].append('S0:999'),
])
def test_tampering_cannot_create_new_facts_or_a_plant_claim(mutation):
    access = AttackHypothesisSession(packet()).consultation()
    reading, _ = access.render(selection(access))
    mutation(reading['interpretation'][0])
    with pytest.raises(ValueError):
        access.verify(reading)


def test_opt_in_adapter_is_causally_replayable_and_old_adapter_is_unchanged():
    session = AttackHypothesisSession(packet())
    access = session.consultation()
    answer = {'action': 'finish', 'indices': [], 'reading': selection(access)}
    jsonschema.validate(answer, local.schema(access))
    result = local.interpret(session, FakeClient([answer]), version=local.HYPOTHESIS_VERSION)
    assert result['status'] == 'mechanically_valid' and local.verify_replay(session, result)['causal_replay']
    assert local.VERSION == '1.2.0-local.1'
    for wrong_session, version in [(session, local.TEMPLATE_VERSION), (ReplayReadingSession(packet()), local.HYPOTHESIS_VERSION)]:
        with pytest.raises(ValueError, match='hypothesis session'):
            local.initial_messages(wrong_session.consultation(), version=version)
