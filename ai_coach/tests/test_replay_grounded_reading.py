"""Grounding is a separate bounded task, not citation existence or tactic labels."""
from copy import deepcopy

import pytest
import jsonschema

from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation.replay_grounded_reading import GroundedReadingSession, DIMENSIONS
from ai_coach.tests.test_context_comparison_ollama import FakeClient
from ai_coach.tests.test_replay_attack_region import situated
from ai_coach.tests.test_replay_contextual_reading import decision, historical_packet


def audit(access, verdict='supported', target=None):
    reference = next(f['id'] for f in access.initial()['situation'] if f['key'] == 'own_player')
    selected = access._draft or decision(access)
    checks=[]
    target = target or (access.audit_packet()['targets'][0] if access._draft else 'H1')
    card=({'missing_information':selected['interpretation']['missing_information']} if target=='decision' else
          selected['interpretation']['hypotheses'][int(target[1:])-1])
    dimensions={}
    for key in (('missing_information',) if target=='decision' else DIMENSIONS):
        text=' '.join(card[key]) if key=='counterevidence' else card[key]
        dimensions[key]={'verdict':verdict if text else 'not_asserted','quote':text,
            'evidence':[reference] if text else [],
            'reason':'Ejercicio sintético del protocolo, sin evaluación semántica.'}
    checks.append({'target':target,'dimensions':dimensions})
    return {'action':'audit','indices':[],'reading':{'checks':checks}}


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
    checked['reading']['checks'][0]['dimensions']['hypothesis'].update(
        verdict='unsupported', reason='Las citas no acreditan esta afirmación.')
    with pytest.raises(ValueError, match='grounding'):
        access.complete_audit(checked)


def test_audit_has_current_context_events_and_only_delivered_history():
    access = GroundedReadingSession(historical_packet()).consultation()
    access.submit(decision(access))
    packet = access.audit_packet()
    assert packet['target_card'] == access._draft['interpretation']['hypotheses'][0]
    assert 'draft' not in packet
    assert packet['current'] == access.initial()['situation']
    assert packet['events'] == access.initial()['events']
    assert all(f['id'] in access._consulted for f in packet['cited_history'])
    assert packet['history_status'] == 'unconsulted_samples_are_not_absent_data'


def test_readable_event_index_does_not_replace_or_trim_the_full_context():
    from ai_coach.interpretation import replay_model_context as wire
    access = GroundedReadingSession(historical_packet()).consultation()
    messages = local.initial_messages(access,version=local.GROUNDED_VERSION)
    import json
    assert wire.unpack(json.loads(messages[1]['content'])) == access.initial()
    assert access.current_overview()['event_index'] == [
        {k:f[k] for k in ('id','key','subject','status','seconds_in_window')}
        for f in access.initial()['events'] if f['key']=='death' or f['key'].startswith('bomb_')]


def test_public_killfeed_and_life_callback_are_not_erased_by_a_later_snapshot():
    from ai_coach.tests.test_replay_factual_life_context import after_death_packet
    access = GroundedReadingSession(after_death_packet()).consultation()
    kills = [f for f in access.initial()['events'] if f['key']=='kill']
    assert kills and kills[0]['value']['weapon']=='AK-47'
    assert access.current_overview()['public_killfeed'] == [
        {'id':f['id'],'text':access._claim(f)['text']} for f in kills]
    assert 'AK-47' in access.current_overview()['public_killfeed'][0]['text']
    assert not access.initial()['spatial_measurements']


def test_audit_keeps_delivered_historical_places_even_if_model_only_cites_positions():
    access = GroundedReadingSession(historical_packet()).consultation()
    selected = decision(access)
    selected['interpretation']['hypotheses'][0]['evidence'] = [
        f['id'] for f in access.initial()['situation'] if f['key']=='own_player']
    access.submit(selected)
    history = access.audit_packet()['cited_history']
    assert any(f['key']=='place_context' for f in history)
    assert all(f['id'] in access._consulted for f in history)


def test_typed_current_view_keeps_one_c4_entity_and_exact_dated_players():
    access = GroundedReadingSession(historical_packet()).consultation()
    view=access.current_overview();current=access.initial()['situation']
    own={f['subject']:f for f in current if f['key']=='own_player'}
    assert view['cut_seconds']==access.initial()['sample_times_seconds'][-1]
    for row in view['players']:
        fact=own[row['participant']]
        assert row['snapshot']['value']=={k:v for k,v in fact['value'].items() if k!='has_c4'}
        assert row['snapshot']['seconds']==fact['seconds_in_window']
        assert view['c4']['value']['player_flags'][row['participant']]['has_c4']==fact['value']['has_c4']
    assert 'facts' not in view


def test_rule_names_are_disjoint_from_participants_without_mutating_legacy_bank():
    from ai_coach.interpretation.replay_contextual_reading import ContextualReadingSession
    data=situated([(-1400,750,-40)])
    legacy=ContextualReadingSession(data).consultation().initial()['bank']
    current=GroundedReadingSession(data).consultation().initial()['bank']
    assert [{**r,'id':r['id'].removeprefix('RULE_')} for r in current['principles']]==legacy['principles']
    assert all(r['id'].startswith('RULE_') for r in current['principles'])
    assert ContextualReadingSession(data).consultation().initial()['bank']==legacy


def test_conditional_intent_does_not_require_an_observed_cognitive_field():
    access=GroundedReadingSession(situated([(-1400,750,-40)])).consultation()
    selected=decision(access);access.submit(selected)
    answer=audit(access)
    answer['reading']['checks'][0]['dimensions']['hypothesis']={
        'verdict':'conditional_inference','quote':selected['interpretation']['hypotheses'][0]['hypothesis'],
        'evidence':selected['interpretation']['hypotheses'][0]['evidence'],
        'reason':'Intención condicional compatible; no afirmación de un pensamiento observado.'}
    assert access.complete_audit(answer) is None
    reading,verification=access.complete_audit(audit(access))
    assert reading['interpretation'][0]['semantic_status']=='unverified_model_inference'
    assert verification['contextual_factual_integrity']=='not_mechanically_established'


def test_conditional_inference_cannot_excuse_invalid_counterevidence():
    access=GroundedReadingSession(situated([(-1400,750,-40)])).consultation()
    selected=decision(access);access.submit(selected)
    answer=audit(access)
    answer['reading']['checks'][0]['dimensions']['counterevidence']={
        'verdict':'conditional_inference','quote':selected['interpretation']['hypotheses'][0]['hypothesis'],
        'evidence':selected['interpretation']['hypotheses'][0]['evidence'],'reason':'No debe permitirse.'}
    with pytest.raises(jsonschema.ValidationError):access.complete_audit(answer)


@pytest.mark.parametrize('bad',['P99 participa.','Hay dos C4.','Hay 2 AK-47.'])
def test_public_identifiers_are_allowed_without_allowing_counts_or_unknown_aliases(bad):
    from ai_coach.tests.test_replay_factual_life_context import after_death_packet
    access = GroundedReadingSession(after_death_packet()).consultation()
    access._checked_text('P02 aparece en el feed con AK-47.')
    with pytest.raises(ValueError,match='quantit'):
        access._checked_text(bad)


def test_historical_query_moves_the_same_current_boundary_after_history():
    from ai_coach.tests.test_replay_contextual_reading import intermediate_packet
    session = GroundedReadingSession(intermediate_packet())
    access = session.consultation()
    selected = decision(access)
    client = FakeClient([{'action':'consult_history','indices':[1],'reading':None},
        {'action':'finish','indices':[],'reading':selected},audit(access),audit(access,target='decision')])
    result = local.interpret(session,client,version=local.GROUNDED_VERSION)
    assert result['status']=='mechanically_valid'
    boundary={'role':'user','content':local.compact(access.current_overview())}
    messages=client.bodies[1]['messages']
    assert messages[-1]==boundary and messages.count(boundary)==1
    assert messages[1]==client.bodies[0]['messages'][1]
    assert local.verify_replay(session,result)['causal_replay']


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
    responses = [{'action':'finish','indices':[],'reading':selected}, audit(access), audit(access,target='decision')]
    client = FakeClient(responses)
    result = local.interpret(session, client, version=local.GROUNDED_VERSION)
    assert result['local_calls'] == 3
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


def test_rejected_grounding_is_preserved_and_replays_without_becoming_a_reading():
    session = GroundedReadingSession(situated([(-1400,750,-40)]))
    access = session.consultation()
    selected = decision(access)
    client = FakeClient([{'action':'finish','indices':[],'reading':selected},audit(access,'unsupported')])
    result = local.interpret(session,client,version=local.GROUNDED_VERSION)
    assert result['status'] == 'rejected_or_incomplete'
    assert result['reading'] is result['verification'] is None
    assert result['contextual_runtime']['draft'] == selected
    assert result['contextual_runtime']['audit']['checks']
    assert local.verify_replay(session,result)['causal_replay']


@pytest.mark.parametrize('bad',['invented_quote','duplicate_target'])
def test_assessment_cannot_quote_a_different_claim_or_omit_a_target(bad):
    access = GroundedReadingSession(situated([(-1400,750,-40)])).consultation()
    access.submit(decision(access))
    answer = audit(access)
    if bad == 'invented_quote':
        answer['reading']['checks'][0]['dimensions']['hypothesis']['quote'] = 'Texto que el borrador no contiene.'
    else:
        answer['reading']['checks'][0]['target'] = 'decision'
    with pytest.raises((ValueError,jsonschema.ValidationError)):
        access.complete_audit(answer)


def test_insufficiency_is_audited_too_and_unknown_cannot_pass():
    access = GroundedReadingSession(situated([(-1400,750,-40)])).consultation()
    selected = decision(access)
    selected['interpretation'].update(status='insufficient', hypotheses=[])
    access.submit(selected)
    checked = audit(access, 'unknown')
    checked['reading']['checks'] = checked['reading']['checks'][:1]
    with pytest.raises(ValueError, match='grounding'):
        access.complete_audit(checked)
