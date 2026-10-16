"""Protected facts must not consume the model's tactical selection channel."""
from copy import deepcopy
import json

import jsonschema
import pytest

from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation import replay_model_context as wire
from ai_coach.interpretation.replay_attack_hypothesis import FocusedAttackHypothesisSession as CandidateSession
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_replay_attack_hypothesis import packet, selection
from ai_coach.tests.test_context_comparison_ollama import FakeClient


def empty():
    return {key: [] for key in ('situation', 'evolution', 'interpretation', 'uncertainty')}


@pytest.mark.parametrize('section', ['situation', 'evolution', 'uncertainty'])
def test_generation_does_not_ask_model_to_copy_facts_that_renderer_already_includes(section):
    data = packet()
    if section == 'evolution':
        data['views'][0]['events'] = [{'id': 'drop-focus', 'type': 'bomb_drop', 'tick': 11,
            'availability_tick': 11, 'player_id': '1', 'knowledge': 'own_team_objective_event',
            'source': {'reference': 'objective-drop-focus', 'kind': 'demoinfocs_event'}}]
    access = CandidateSession(data).consultation()
    required = access.initial()['required_reading_facts'][section]
    assert required
    chosen = empty()
    chosen[section] = required[:1]
    with pytest.raises(jsonschema.ValidationError):
        jsonschema.validate({'action': 'finish', 'indices': [], 'reading': chosen}, local.schema(access))


@pytest.mark.parametrize('count', [2, 3])
def test_empty_factual_selection_keeps_every_original_fact_and_numeric_value(count):
    data = packet(count)
    access = CandidateSession(data).consultation()
    baseline = ReplayReadingSession(data).consultation().render(empty())[0]
    chosen = selection(access, count)
    chosen['interpretation'][0]['evidence'].sort()
    jsonschema.validate({'action': 'finish', 'indices': [], 'reading': chosen}, local.schema(access))
    reading, verified = access.render(chosen)
    for section in ('situation', 'evolution', 'uncertainty'):
        assert reading[section] == baseline[section]
    assert access.verify(reading) == verified
    assert len(reading['interpretation'][0]['hypothesis']['player_fact_ids']) == count


def test_abstention_is_not_replaced_by_an_automatic_attack_label():
    access = CandidateSession(packet()).consultation()
    reading, verified = access.render(empty())
    assert reading['interpretation'] == []
    assert access.verify(reading) == verified
    assert reading['situation'] and reading['uncertainty']


@pytest.mark.parametrize('variant', ['future', 'current_as_history', 'observed_as_unknown', 'free_prose'])
def test_output_schema_rejects_wrong_or_unavailable_factual_references(variant):
    access = CandidateSession(packet()).consultation()
    chosen = empty()
    if variant == 'future':
        chosen['situation'] = ['S999:3']
    elif variant == 'current_as_history':
        chosen['evolution'] = [access.initial()['situation'][0]['id']]
    elif variant == 'observed_as_unknown':
        chosen['uncertainty'] = [next(f['id'] for f in access.initial()['situation'] if f['status'] == 'observed')]
    else:
        chosen['interpretation'] = [{'text': 'dos C4 y plantado confirmado'}]
    with pytest.raises(jsonschema.ValidationError):
        jsonschema.validate({'action': 'finish', 'indices': [], 'reading': chosen}, local.schema(access))


def test_no_geometry_threshold_or_new_game_requirement_is_added_to_contract():
    old = CandidateSession(packet()).consultation().initial()['hypothesis_contract']
    assert old['advanced_reference']['position_xy'] == [-1500, 750]
    assert old['spatial_classification'] == 'model_estimate_no_metric_threshold'
    assert [r['distance_xy_units'] for r in old['eligible_players']] == [0, 10]
    assert old['alternatives'] == ['staging', 'withdrawal', 'another_destination']


def test_focused_adapter_keeps_full_lossless_input_and_causal_queries_with_original_limits():
    session = CandidateSession(packet())
    access = session.consultation()
    chosen = selection(access)
    chosen['interpretation'][0]['evidence'].sort()
    client = FakeClient([{'action': 'consult', 'indices': [0], 'reading': None},
                         {'action': 'finish', 'indices': [], 'reading': chosen}])
    result = local.interpret(session, client, version=local.FOCUSED_HYPOTHESIS_VERSION)
    assert result['status'] == 'mechanically_valid'
    assert local.verify_replay(session, result)['causal_replay']
    assert result['receipt']['reading_contract'] == 'stratai.replay_attack_hypothesis@3'
    assert result['verification']['representation'] == 'stratai.replay_attack_hypothesis@3'
    body = client.bodies[0]
    assert wire.unpack(json.loads(body['messages'][1]['content'])) == access.initial()
    assert body['options'] == {'num_ctx': 65536, 'num_predict': 8192, 'temperature': 0, 'seed': 17, 'presence_penalty': 0}
    assert result['queries'][0]['index'] == 0
    assert local.VERSION == '1.2.0-local.1'


def test_legacy_bounded_adapter_keeps_its_original_prompt_and_copying_schema():
    from ai_coach.interpretation.replay_attack_hypothesis import BoundedAttackHypothesisSession
    access = BoundedAttackHypothesisSession(packet()).consultation()
    body = local.initial_messages(access, version=local.BOUNDED_HYPOTHESIS_VERSION)
    assert body[0]['content'].endswith('Do not force an interpretation when delivered facts do not distinguish alternatives.')
    assert local.schema(access)['anyOf'][0]['properties']['reading']['properties']['situation']['maxItems'] == 256
    chosen = empty()
    chosen['situation'] = access.initial()['required_reading_facts']['situation']
    jsonschema.validate({'action': 'finish', 'indices': [], 'reading': chosen}, local.schema(access))


def test_invalid_model_copy_is_rejected_without_retry_or_losing_the_original_response():
    session = CandidateSession(packet())
    chosen = empty()
    chosen['situation'] = session.consultation().initial()['required_reading_facts']['situation']
    client = FakeClient([{'action': 'finish', 'indices': [], 'reading': chosen}])
    result = local.interpret(session, client, version=local.FOCUSED_HYPOTHESIS_VERSION)
    assert result['status'] == 'rejected_or_incomplete' and result['reading'] is None
    assert result['local_calls'] == 1 and result['transcript'][0]['response']
    assert local.verify_replay(session, result)['causal_replay']


def test_current_spatial_review_is_plain_exact_data_not_a_tactical_answer():
    access = CandidateSession(packet(3)).consultation()
    messages = local.initial_messages(access, version=local.FOCUSED_HYPOTHESIS_VERSION)
    assert len(messages) == 3
    review = json.loads(messages[2]['content'])
    facts = {f['id']: f for f in access.initial()['situation']}
    original = access.initial()['hypothesis_contract']
    assert review == original['spatial_review']
    assert review['valid_reference_combinations'] == original['valid_reference_combinations']
    assert len(review['eligible_players']) == 3
    for row in review['eligible_players']:
        assert row['current_player_fact'] == facts[row['player_fact_id']]
        assert row['current_place_fact'] == facts[row['place_fact_id']]
        assert row['distance_xy_units'] == next(r['distance_xy_units'] for r in original['eligible_players']
                                               if r['player_fact_id'] == row['player_fact_id'])
        assert row['current_place_fact']['value']['zone'] == 'Apartments'
    assert 'assessment' not in review and 'selected_hypothesis' not in review


def test_empty_spatial_review_never_supplies_historical_or_hidden_enemy_coordinates():
    data = packet()
    data['views'][0]['frame']['round_context'].update(phase='ended', end_tick=12, winner='CT', decision_status='round_closed')
    data['views'][0]['frame']['bomb'].update(state='resolved', objective_phase='resolved')
    access = CandidateSession(data).consultation()
    messages = local.initial_messages(access, version=local.FOCUSED_HYPOTHESIS_VERSION)
    assert json.loads(messages[2]['content'])['eligible_players'] == []


def test_optional_history_is_bounded_without_truncating_the_protected_facts():
    data = packet(3)
    later = deepcopy(data['views'][0])
    def advance(value):
        if isinstance(value, dict):
            for key, item in value.items():
                if key in ('tick', 'observation_tick', 'availability_tick', 'basis_tick', 'position_tick') and item == 12:
                    value[key] = 14
                elif key == 'reference' and isinstance(item, str):
                    value[key] = item.replace('frame:12:', 'frame:14:')
                else:
                    advance(item)
        elif isinstance(value, list):
            for item in value:
                advance(item)
    advance(later)
    later['frame']['round_context']['elapsed_seconds'] = 4/64
    data.update(cut_tick=14, views=[data['views'][0], later])
    access = CandidateSession(data).consultation()
    schema = local.schema(access)
    available = schema['anyOf'][0]['properties']['reading']['properties']['evolution']['items']['enum']
    assert len(available) >= 9
    chosen = empty()
    chosen['evolution'] = available[:8]
    jsonschema.validate({'action': 'finish', 'indices': [], 'reading': chosen}, schema)
    reading, verified = access.render(chosen)
    assert access.verify(reading) == verified
    assert len(reading['situation']) == len(access.initial()['required_reading_facts']['situation'])
    chosen['evolution'] = available[:9]
    with pytest.raises(jsonschema.ValidationError):
        jsonschema.validate({'action': 'finish', 'indices': [], 'reading': chosen}, schema)
    with pytest.raises(ValueError, match='too many optional'):
        access.render(chosen)


def test_spatial_review_copies_cannot_mutate_the_dated_factual_source():
    access = CandidateSession(packet()).consultation()
    before = access.initial()
    copied = access.spatial_review()
    copied['eligible_players'][0]['current_player_fact']['value']['has_c4'] = True
    copied['eligible_players'][0]['current_place_fact']['value']['zone'] = 'BackAlley'
    assert access.initial() == before
