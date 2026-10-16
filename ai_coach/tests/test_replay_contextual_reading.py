"""Context is evidence, never a position-to-intent tactical classifier."""
from copy import deepcopy
import json

import jsonschema
import pytest

from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation import replay_model_context as wire
from ai_coach.interpretation.replay_contextual_reading import ContextualReadingSession
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_context_comparison_ollama import FakeClient
from ai_coach.tests.test_replay_attack_region import situated, empty


def decision(access):
    ids = [f['id'] for f in access.initial()['situation'] if f['key'] == 'own_player']
    selection = empty()
    selection['interpretation'] = {
        'status': 'hypotheses', 'missing_information': 'La intención definitiva permanece desconocida.',
        'hypotheses': [{
            'hypothesis': 'Posible preparación de ataque a B.', 'plausibility': 'possible',
            'evidence': ids, 'counterevidence': [],
            'limits': 'También puede tratarse de espera o presión.',
            'would_change': 'Una progresión posterior o repliegue cambiaría esta lectura.',
        }],
    }
    return selection


def test_general_hypotheses_are_not_filtered_by_the_b_apps_polygon():
    data = situated([(-1900,750,-40),(-1800,750,-40)])
    access = ContextualReadingSession(data).consultation()
    assert 'hypothesis_contract' not in access.initial()
    assert all(p['membership']['relation'] == 'outside' for p in access.initial()['spatial_measurements'])
    selected = decision(access)
    selected['interpretation']['hypotheses'][0]['hypothesis'] = 'Posible espera de coordinación.'
    reading, verification = access.render(selected)
    claim = reading['interpretation'][0]
    assert 'Posible espera de coordinación.' in claim['text']
    assert claim['semantic_status'] == 'unverified_model_inference'
    assert verification['contextual_factual_integrity'] == 'not_mechanically_established'
    assert verification['factual_integrity_scope'] == 'situation_evolution_uncertainty_only'
    assert access.verify(reading) == verification
    baseline = ReplayReadingSession(data).consultation().render(empty())[0]
    assert all(reading[s] == baseline[s] for s in ('situation','evolution','uncertainty'))


def test_compatible_hypotheses_are_shown_without_a_second_selection_or_fake_percentages():
    access = ContextualReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    second = deepcopy(selected['interpretation']['hypotheses'][0])
    second['hypothesis'] = 'Posible espera de coordinación antes de continuar.'
    second['plausibility'] = 'preferred'
    selected['interpretation']['hypotheses'].append(second)
    reading, _ = access.render(selected)
    assert len(reading['interpretation']) == 2
    assert [c['contextual_hypothesis'] for c in reading['interpretation']] == selected['interpretation']['hypotheses']
    assert all('pendiente de revisión' in c['text'] for c in reading['interpretation'])


@pytest.mark.parametrize('bad', ['empty_supported','insufficient_with_hypothesis','missing_reason','unknown_status'])
def test_decision_cannot_claim_support_while_omitting_hypotheses(bad):
    access = ContextualReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    if bad == 'empty_supported': selected['interpretation']['hypotheses'] = []
    elif bad == 'insufficient_with_hypothesis': selected['interpretation']['status'] = 'insufficient'
    elif bad == 'missing_reason': selected['interpretation'] = {'status':'insufficient','hypotheses':[], 'missing_information':''}
    else: selected['interpretation']['status'] = 'invented'
    with pytest.raises((ValueError, jsonschema.ValidationError)):
        access.render(selected)


def test_insufficiency_has_a_visible_reason_and_no_tactical_autofill():
    access = ContextualReadingSession(situated([(-1400,750,-40),(-1380,750,-40)])).consultation()
    selected = empty()
    selected['interpretation'] = {'status':'insufficient','hypotheses':[],
        'missing_information':'No puedo preferir una intención con estas muestras.'}
    reading, checked = access.render(selected)
    assert len(reading['interpretation']) == 1
    assert reading['interpretation'][0]['contextual_decision']['status'] == 'insufficient'
    assert 'No puedo preferir' in reading['interpretation'][0]['text']
    assert access.verify(reading) == checked


@pytest.mark.parametrize('text', ['Hay dos C4.', 'Hay 2 C4.', 'Hay ２ C4.', 'Quedan tres CT.', 'Probabilidad del 70%.'])
def test_unprotected_quantities_are_rejected_in_contextual_prose(text):
    access = ContextualReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    selected['interpretation']['hypotheses'][0]['hypothesis'] = text
    with pytest.raises(ValueError, match='quantit'):
        access.render(selected)


@pytest.mark.parametrize('field', ['evidence','counterevidence'])
def test_future_or_undelivered_citations_are_rejected(field):
    access = ContextualReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    selected['interpretation']['hypotheses'][0][field] = ['S99:3']
    with pytest.raises((ValueError, jsonschema.ValidationError)):
        access.render(selected)


def test_opt_in_adapter_roundtrip_preserves_model_and_historical_defaults():
    session = ContextualReadingSession(situated([(-1400,750,-40)]))
    access = session.consultation()
    messages = local.initial_messages(access, version=local.CONTEXTUAL_VERSION)
    assert wire.unpack(json.loads(messages[1]['content'])) == access.initial()
    selected = decision(access)
    client = FakeClient([{'action':'finish','indices':[],'reading':selected}])
    result = local.interpret(session, client, version=local.CONTEXTUAL_VERSION)
    assert result['status'] == 'mechanically_valid'
    assert local.verify_replay(session, result)['causal_replay']
    assert local.VERSION == '1.2.0-local.1'
    assert result['receipt']['reading_contract'] == 'stratai.replay_contextual_reading@1'
    assert client.bodies[0]['options']['num_ctx'] == 65536
    altered = deepcopy(result['reading'])
    altered['interpretation'][0]['facts'][0]['value']['has_c4'] = True
    with pytest.raises(ValueError): access.verify(altered)


def historical_packet(previous_x=-1900):
    data = situated([(-1400,750,-40)])
    first = deepcopy(data['views'][0])
    first.update(tick=11, availability_tick=11)
    first['frame']['tick'] = 11
    context = first['frame']['round_context']
    context.update(observation_tick=11, availability_tick=11, elapsed_seconds=1/64)
    context['source']['reference'] = 'round:1:frame:11:round_context'
    context['eligibility']['basis_tick'] = 11
    bomb = first['frame']['bomb']
    bomb.update(observation_tick=11, availability_tick=11)
    bomb['source']['reference'] = 'round:1:frame:11:bomb'
    player = first['frame']['players'][0]
    player.update(x=previous_x, observation_tick=11, availability_tick=11)
    player['source']['reference'] = 'round:1:frame:11:player:1'
    player['inventory_context'].update(observation_tick=11, availability_tick=11)
    player['inventory_context']['source']['reference'] = 'frame:11:inventory'
    from ai_coach.interpretation.place_volumes import resolve, load_reference
    player['place'].update(resolve(player, load_reference(), map_name='de_mirage', xy_precision=.5),
                           observation_tick=11, availability_tick=11)
    data['views'].insert(0, first)
    return data


def test_same_final_positions_with_different_pasts_deliver_different_actual_history():
    first = historical_packet(-1900)
    second = historical_packet(-1400)
    saved = deepcopy(first)
    session = ContextualReadingSession(first)
    left = session.consultation().initial()
    right = ContextualReadingSession(second).consultation().initial()
    assert left['situation'] == right['situation']
    assert left['own_context_history'] != right['own_context_history']
    assert left['own_context_history'] == [f for f in session._archive[0]
        if f['key'] in ('own_player','place_context','objective','round_context')]
    assert left['history_coverage']['continuity'] == 'discrete_samples_only'
    assert left['history_coverage']['sample_times_seconds'] == left['sample_times_seconds']
    assert len(left['sample_times_seconds']) == 2
    assert first == saved
    access = session.consultation()
    selected = decision(access)
    selected['interpretation']['hypotheses'][0]['evidence'].append(
        next(f['id'] for f in left['own_context_history'] if f['key']=='own_player'))
    reading, checked = access.render(selected)
    assert access.verify(reading) == checked


def test_single_cut_has_no_invented_earlier_positions_and_future_packet_is_rejected():
    data = situated([(-1400,750,-40)])
    access = ContextualReadingSession(data).consultation()
    assert access.initial()['own_context_history'] == []
    data['views'][0]['tick'] = data['cut_tick'] + 1
    with pytest.raises(ValueError): ContextualReadingSession(data)


@pytest.mark.parametrize('field', ['hypothesis','limits','would_change','missing_information'])
def test_quantities_are_guarded_in_every_contextual_text_field(field):
    access = ContextualReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    target = selected['interpretation'] if field == 'missing_information' else selected['interpretation']['hypotheses'][0]
    target[field] = 'Dos C4.'
    with pytest.raises(ValueError, match='quantit'): access.render(selected)


def test_adapter_mismatch_and_free_factual_text_cannot_bypass_protection():
    access = ContextualReadingSession(situated([(-1400,750,-40)])).consultation()
    with pytest.raises(ValueError): local.initial_messages(access)
    selected = decision(access)
    selected['situation'] = ['Hay dos C4.']
    with pytest.raises(jsonschema.ValidationError): access.render(selected)


def intermediate_packet():
    data = historical_packet()
    earlier = deepcopy(data['views'][0])
    def restamp(node):
        if isinstance(node, dict):
            for key, value in node.items():
                if key in ('tick','observation_tick','availability_tick','basis_tick') and value == 11:
                    node[key] = 10
                elif key == 'reference' and isinstance(value, str):
                    node[key] = value.replace('frame:11:', 'frame:10:')
                else: restamp(value)
        elif isinstance(node, list):
            for value in node: restamp(value)
    restamp(earlier)
    earlier['frame']['round_context']['elapsed_seconds'] = 0
    data['views'].insert(0, earlier)
    return data


def test_intermediate_history_stays_available_without_being_silently_delivered_or_cited():
    data = intermediate_packet()
    session = ContextualReadingSession(data)
    access = session.consultation()
    assert len(session._archive) == len(access.initial()['sample_times_seconds']) == 3
    assert len(access.initial()['own_context_history']) == 4
    assert access.initial()['history_coverage']['initial_history_indices'] == [0]
    middle = next(f['id'] for f in session._archive[1] if f['key']=='own_player')
    selected = decision(access)
    selected['interpretation']['hypotheses'][0]['evidence'] = [middle]
    with pytest.raises(jsonschema.ValidationError): access.render(selected)
    assert access.sample(1) == session._archive[1]
    reading, verification = access.render(selected)
    assert access.verify(reading) == verification


def test_provider_grammar_uses_object_schemas_for_array_items_even_when_no_optional_facts_exist():
    access = ContextualReadingSession(situated([(-1400,750,-40)])).consultation()
    schema = local.schema(access)
    def check(node):
        if isinstance(node, dict):
            if 'items' in node:
                assert isinstance(node['items'], dict), 'local provider rejects boolean item schemas'
            for value in node.values(): check(value)
        elif isinstance(node, list):
            for value in node: check(value)
    check(schema)


def test_protocol_never_reoffers_supplied_samples_and_finishes_after_all_queries():
    access = ContextualReadingSession(intermediate_packet()).consultation()
    schema = local.schema(access)
    with pytest.raises(jsonschema.ValidationError):
        jsonschema.validate({'action':'consult','indices':[2],'reading':None}, schema)
    history = access.query(1, 'history')
    assert all(f['key'] in ('own_player','place_context','round_context','objective') for f in history)
    assert access.available_indices('history') == []
    assert access.available_indices('full') == [1]
    access.query(1, 'full')
    assert len(local.schema(access)['anyOf']) == 1


def test_history_consultation_is_smaller_and_does_not_make_undelivered_inventory_citable():
    access = ContextualReadingSession(intermediate_packet()).consultation()
    history = access.query(1, 'history')
    assert len(history) < len(access._session._archive[1])
    inventory = next(f['id'] for f in access._session._archive[1] if f['key']=='own_inventory')
    assert inventory not in access._consulted
    access.query(1, 'full')
    assert inventory in access._consulted


def test_last_model_call_can_finish_with_insufficiency_but_cannot_request_more_data():
    session = ContextualReadingSession(intermediate_packet())
    access = session.consultation()
    access._remaining_model_calls = 1
    assert len(local.schema(access)['anyOf']) == 1
    selected = empty()
    selected['interpretation'] = {'status':'insufficient','hypotheses':[],
                                  'missing_information':'No puedo preferir una intención.'}
    jsonschema.validate({'action':'finish','indices':[],'reading':selected}, local.schema(access))


def test_context_budget_refusal_restores_citations_and_never_repeats_the_same_query():
    session = ContextualReadingSession(intermediate_packet())
    access = session.consultation()
    config = local.Config()
    messages = local.initial_messages(access, version=local.CONTEXTUAL_VERSION)
    answer = {'action':'consult_history','indices':[1],'reading':None}
    response = {'model':local.MODEL,'done':True,'done_reason':'stop',
        'prompt_eval_count':config.num_ctx-config.num_predict-10,
        'message':{'content':json.dumps(answer)}}
    local.consume(access,response,messages,config,version=local.CONTEXTUAL_VERSION)
    assert access._context_budget_refusals and access._queries == []
    assert access.available_indices('history') == []
    middle = next(f['id'] for f in session._archive[1] if f['key']=='own_player')
    assert middle not in access._consulted
    assert 'not_delivered' in messages[-1]['content']


def test_contextual_query_then_finish_replays_with_original_total_call_budget():
    session = ContextualReadingSession(intermediate_packet())
    access = session.consultation()
    client = FakeClient([{'action':'consult_history','indices':[1],'reading':None},
                         {'action':'finish','indices':[],'reading':decision(access)}])
    result = local.interpret(session, client, version=local.CONTEXTUAL_VERSION)
    assert result['status'] == 'mechanically_valid'
    assert local.verify_replay(session,result)['causal_replay']
    assert result['contextual_runtime']['query_attempts'] == [{'index':1,'scope':'history'}]


def test_readable_overview_is_only_a_factual_index_with_exact_protected_text_and_no_new_evidence():
    session = ContextualReadingSession(situated([(-1400,750,-40),(-1380,750,-40)]))
    access = session.consultation()
    initial = access.initial()
    overview = access.current_overview()
    expected = {f['id']:f for f in initial['situation']}
    catalogue = [identity for states in overview['catalogue'].values() for ids in states.values() for identity in ids.split()]
    assert sorted(catalogue) == sorted(expected)
    for row in overview['facts']:
        assert row['text'] == access._claim(expected[row['id']])['text']
    assert {expected[row['id']]['key'] for row in overview['facts']} == {
        'own_player','place_context','round_context','objective'}
    assert access.initial() == initial
    messages = local.initial_messages(access,version=local.CONTEXTUAL_VERSION)
    assert json.loads(messages[2]['content']) == overview
    assert wire.unpack(json.loads(messages[1]['content'])) == initial
