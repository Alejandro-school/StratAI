"""Grounding is a separate bounded task, not citation existence or tactic labels."""
from copy import deepcopy

import pytest
import jsonschema

from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation.replay_grounded_reading import GroundedReadingSession
from ai_coach.tests.test_context_comparison_ollama import FakeClient
from ai_coach.tests.test_replay_attack_region import situated
from ai_coach.tests.test_replay_contextual_reading import decision, historical_packet


def audit(access, verdict='supported'):
    reference = next(f['id'] for f in access.initial()['situation'] if f['key'] == 'own_player')
    return {'action':'audit', 'indices':[], 'reading':{'checks':[
        {'target':target, 'verdict':verdict, 'evidence':[reference],
         'reason':'La hipótesis conserva su carácter condicional; no acredita intención.'}
        for target in ['decision','H1']]}}


def test_draft_cannot_be_emitted_without_the_grounding_pass():
    access = GroundedReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    with pytest.raises(ValueError, match='audit'):
        access.render(selected)
    access.submit(selected)
    with pytest.raises(ValueError, match='audit'):
        access.render(selected)


@pytest.mark.parametrize('claim',[
    'P09 fue eliminado por fuego enemigo en su posición de inicio.',
    'P02 espera en una zona segura para plantar en BackAlley.',
])
def test_unsupported_factual_assertion_is_blocked_even_with_valid_citations(claim):
    access = GroundedReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    who = next(f['subject'] for f in access.initial()['situation'] if f['key'] == 'own_player')
    claim = claim.replace('P09',who).replace('P02',who)
    selected['interpretation']['hypotheses'][0]['hypothesis'] = claim
    access.submit(selected)
    checked = audit(access)
    checked['reading']['checks'][1].update(verdict='unsupported', reason='Las citas no acreditan esta afirmación.')
    with pytest.raises(ValueError, match='grounding'):
        access.complete_audit(checked)


def test_audit_has_current_context_events_and_only_delivered_history():
    access = GroundedReadingSession(historical_packet()).consultation()
    access.submit(decision(access))
    packet = access.audit_packet()
    assert packet['draft'] == access._draft
    assert packet['current'] == access.initial()['situation']
    assert packet['events'] == access.initial()['events']
    assert all(f['id'] in access._consulted for f in packet['cited_history'])
    assert packet['history_status'] == 'unconsulted_samples_are_not_absent_data'


def test_audit_must_cover_every_row_and_the_missing_information_claim():
    access = GroundedReadingSession(situated([(-1400,750,-40)])).consultation()
    access.submit(decision(access))
    incomplete = audit(access)
    incomplete['reading']['checks'].pop()
    with pytest.raises((ValueError,jsonschema.ValidationError)):
        access.complete_audit(incomplete)


def test_same_model_budget_causal_replay_and_no_tactical_autofill():
    session = GroundedReadingSession(situated([(-1900,750,-40)]))
    access = session.consultation()
    selected = decision(access)
    responses = [{'action':'finish','indices':[],'reading':selected}, audit(access)]
    client = FakeClient(responses)
    result = local.interpret(session, client, version=local.GROUNDED_VERSION)
    assert result['local_calls'] == 2
    assert result['status'] == 'mechanically_valid'
    assert local.verify_replay(session,result)['causal_replay']
    assert result['verification']['grounding_review'] == 'same_model_assisted_not_independent'
    assert result['reading']['interpretation'][0]['contextual_hypothesis'] == selected['interpretation']['hypotheses'][0]
    assert all(b['options']['num_ctx'] == 65536 and b['options']['num_predict'] == 8192 for b in client.bodies)
    assert local.VERSION == '1.2.0-local.1'
    tampered = deepcopy(result)
    tampered['reading']['interpretation'][0]['text'] += ' alterado'
    with pytest.raises(ValueError): local.verify_replay(session,tampered)


def test_queries_reserve_a_call_for_drafting_and_auditing():
    access = GroundedReadingSession(historical_packet()).consultation()
    access._remaining_model_calls = 2
    schema = access.envelope_schema()
    assert len(schema['anyOf']) == 1
    assert schema['anyOf'][0]['properties']['action']['enum'] == ['finish']


def test_insufficiency_is_audited_too_and_unknown_cannot_pass():
    access = GroundedReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    selected['interpretation'].update(status='insufficient', hypotheses=[])
    access.submit(selected)
    checked = audit(access, 'unknown')
    checked['reading']['checks'] = checked['reading']['checks'][:1]
    with pytest.raises(ValueError, match='grounding'):
        access.complete_audit(checked)
