"""Approved region membership is numeric; selecting an attack remains optional."""
from copy import deepcopy
import json

import jsonschema
import pytest

from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation import replay_model_context as wire
from ai_coach.interpretation.place_volumes import load_reference, resolve
from ai_coach.interpretation.replay_attack_region import RegionAttackHypothesisSession, membership
from ai_coach.interpretation.replay_attack_hypothesis import FocusedAttackHypothesisSession, HYPOTHESIS
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_context_comparison_ollama import FakeClient
from ai_coach.tests.test_replay_attack_hypothesis import packet


def situated(points):
    data = packet(len(points))
    for player, (x, y, z) in zip(data['views'][0]['frame']['players'], points):
        player.update(x=x, y=y, z=z)
        player['place'] = {**resolve(player, load_reference(), map_name='de_mirage', xy_precision=.5),
                           'observation_tick': 12, 'availability_tick': 12, 'xy_precision': .5}
    return data


def empty():
    return {key: [] for key in ('situation', 'evolution', 'interpretation', 'uncertainty')}


@pytest.mark.parametrize('point,expected', [
    ((-1460,751,-48), 'inside'), ((-1343,796,-80), 'inside'),
    ((-1100,681,-40), 'inside'), ((-1027,598,-40), 'inside'),
    ((-1536,765,-21), 'outside'), ((-1900,750,-40), 'outside'),
    ((-1300,600,40), 'outside'),  # notch in the L, unique Apartments height
    ((-1300,600,-40), 'unknown'),  # overlapping B/Apps volumes stay ambiguous
    ((-1500,750,-40), 'boundary'), ((-1208,600,-40), 'boundary'),
    ((-1300,675,-40), 'boundary'), ((-1499.6,750,-40), 'boundary'),
    ((-1499,750,-40), 'inside'), ((-1501,750,-40), 'outside'),
    ((-1100,600,-200), 'outside'),  # same XY, downstairs
    ((-1100,600,500), 'outside'),
])
def test_membership_obeys_the_full_approved_contour_height_and_source_precision(point, expected):
    position = dict(zip(('x','y','z'), point))
    place = {**resolve(position, load_reference(), map_name='de_mirage', xy_precision=.5), 'xy_precision': .5}
    result = membership(position, place, map_name='de_mirage')
    assert result['relation'] == expected
    assert result['status'] == 'estimated' and result['build_compatibility'] == 'unverified'


@pytest.mark.parametrize('variant', ['reference', 'map', 'precision', 'nan', 'missing_z', 'inconsistent_place'])
def test_unverifiable_geometry_never_becomes_a_counted_inside_player(variant):
    position = {'x': -1400, 'y': 750, 'z': -40}
    place = {**resolve(position, load_reference(), map_name='de_mirage', xy_precision=.5), 'xy_precision': .5}
    map_name = 'de_mirage'
    if variant == 'reference': place['reference_id'] = 'different-reference'
    elif variant == 'map': map_name = 'de_other'
    elif variant == 'precision': place.pop('xy_precision')
    elif variant == 'nan': position['x'] = float('nan')
    elif variant == 'missing_z': position.pop('z')
    else: place['zone'] = 'BackAlley'
    assert membership(position, place, map_name=map_name)['relation'] == 'unknown'


def test_w01_pair_comes_from_region_not_nearest_two_or_third_player_veto():
    data = situated([(-1460,751,-48),(-1536,765,-21),(-1343,796,-80)])
    before = deepcopy(data)
    access = RegionAttackHypothesisSession(data).consultation()
    contract = access.initial()['hypothesis_contract']
    facts = {f['id']: f for f in access.initial()['situation']}
    assert [facts[p['player_fact_id']]['subject'] for p in contract['eligible_players']] == ['P01','P03']
    assert len(contract['valid_reference_combinations']) == 1
    assert sorted(p['membership']['relation'] for p in contract['spatial_review']['players']) == ['inside','inside','outside']
    assert before == data
    chosen = empty()
    chosen['interpretation'] = [{'hypothesis_id': HYPOTHESIS, 'evidence': contract['valid_reference_combinations'][0]}]
    reading, verified = access.render(chosen)
    assert access.verify(reading) == verified
    claim = reading['interpretation'][0]
    assert '2 T' in claim['text'] and 'tramo avanzado' in claim['text']
    assert 'estimada por el modelo' not in claim['text']
    assert len(claim['region_support']) == 2
    assert all(p['relation'] == 'inside' for p in claim['region_support'])
    baseline = ReplayReadingSession(data).consultation().render(empty())[0]
    assert all(reading[s] == baseline[s] for s in ('situation','evolution','uncertainty'))


@pytest.mark.parametrize('points', [
    [(-1460,751,-48),(-1536,765,-21)],
    [(-1900,750,-40),(-1800,750,-40)],
    [(-1500,750,-40),(-1400,750,-40)],
    [(-1400,750,-40),(-1400,750,-200)],
])
def test_outside_boundary_or_other_floor_cannot_supply_second_member(points):
    access = RegionAttackHypothesisSession(situated(points)).consultation()
    assert access.initial()['hypothesis_contract']['valid_reference_combinations'] == []
    chosen = empty()
    ids = [p['current_player_fact']['id'] for p in access.spatial_review()['players']]
    chosen['interpretation'] = [{'hypothesis_id': HYPOTHESIS, 'evidence': ids}]
    with pytest.raises(jsonschema.ValidationError):
        jsonschema.validate({'action':'finish','indices':[],'reading':chosen}, local.schema(access))
    with pytest.raises(ValueError): access.render(chosen)


@pytest.mark.parametrize('count', [2,3])
def test_region_adapter_keeps_original_facts_optional_attack_queries_and_model_limits(count):
    data = situated([(-1400+i*15,750,-40) for i in range(count)])
    session = RegionAttackHypothesisSession(data)
    access = session.consultation()
    messages = local.initial_messages(access, version=local.REGION_HYPOTHESIS_VERSION)
    assert wire.unpack(json.loads(messages[1]['content'])) == access.initial()
    assert json.loads(messages[2]['content']) == access.spatial_review()
    ids = access.initial()['hypothesis_contract']['valid_reference_combinations'][-1]
    chosen = empty()
    chosen['interpretation'] = [{'hypothesis_id':HYPOTHESIS, 'evidence':ids}]
    client = FakeClient([{'action':'consult','indices':[0],'reading':None},
                         {'action':'finish','indices':[],'reading':chosen}])
    result = local.interpret(session, client, version=local.REGION_HYPOTHESIS_VERSION)
    assert result['status'] == 'mechanically_valid'
    assert local.verify_replay(session,result)['causal_replay']
    assert result['receipt']['reading_contract'] == 'stratai.replay_attack_hypothesis@4'
    assert result['reading']['interpretation'][0]['hypothesis']['player_fact_ids'] == ids
    assert client.bodies[0]['options'] == {'num_ctx':65536,'num_predict':8192,'temperature':0,'seed':17,'presence_penalty':0}
    assert local.VERSION == '1.2.0-local.1'
    assert access.render(empty())[0]['interpretation'] == []


def test_closed_round_cannot_generate_current_groups_or_expose_enemy_coordinates():
    data = situated([(-1400,750,-40),(-1380,750,-40)])
    data['views'][0]['frame']['round_context'].update(phase='ended',end_tick=12,winner='CT',decision_status='round_closed')
    data['views'][0]['frame']['bomb'].update(state='resolved',objective_phase='resolved')
    access = RegionAttackHypothesisSession(data).consultation()
    assert access.spatial_review()['players'] == []
    assert not access.initial()['hypothesis_contract']['valid_reference_combinations']


def test_membership_evidence_is_immutable_and_tampering_is_detected():
    access = RegionAttackHypothesisSession(situated([(-1400,750,-40),(-1380,750,-40)])).consultation()
    copied = access.spatial_review()
    copied['players'][0]['membership']['relation'] = 'outside'
    assert access.spatial_review()['players'][0]['membership']['relation'] == 'inside'
    chosen = empty()
    chosen['interpretation'] = [{'hypothesis_id':HYPOTHESIS,'evidence':access.initial()['hypothesis_contract']['valid_reference_combinations'][0]}]
    reading, _ = access.render(chosen)
    reading['interpretation'][0]['region_support'][0]['relation'] = 'outside'
    with pytest.raises(ValueError): access.verify(reading)


def test_legacy_focused_contract_and_prompt_remain_unchanged():
    access = FocusedAttackHypothesisSession(packet()).consultation()
    assert access.initial()['hypothesis_contract']['spatial_classification'] == 'model_estimate_no_metric_threshold'
    assert 'Compare' in local.initial_messages(access,version=local.FOCUSED_HYPOTHESIS_VERSION)[0]['content']
