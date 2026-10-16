"""Generation grammar must express the same constraints as the verifier."""
import jsonschema
import pytest

from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation.replay_attack_hypothesis import BoundedAttackHypothesisSession
from ai_coach.tests.test_replay_attack_hypothesis import packet, selection
from ai_coach.tests.test_context_comparison_ollama import FakeClient


def answer(access):
    return {'action':'finish', 'indices':[], 'reading':selection(access)}


@pytest.mark.parametrize('variant', ['repeated_hypothesis', 'duplicate_player', 'one_player', 'finish_indices', 'consult_reading', 'consult_empty'])
def test_generation_schema_rejects_observed_loop_and_invalid_envelopes(variant):
    access=BoundedAttackHypothesisSession(packet()).consultation()
    chosen=answer(access)
    if variant=='repeated_hypothesis':chosen['reading']['interpretation']*=2
    elif variant=='duplicate_player':chosen['reading']['interpretation'][0]['evidence'].append(chosen['reading']['interpretation'][0]['evidence'][0])
    elif variant=='one_player':chosen['reading']['interpretation'][0]['evidence'].pop()
    elif variant=='finish_indices':chosen['indices']=[0]
    elif variant=='consult_reading':chosen.update(action='consult',indices=[0])
    else:chosen.update(action='consult',reading=None)
    with pytest.raises(jsonschema.ValidationError):jsonschema.validate(chosen,local.schema(access))


def test_bounded_selection_canonicalizes_only_identity_combinations_not_spatial_groups():
    access=BoundedAttackHypothesisSession(packet(3)).consultation()
    context=access.initial()['hypothesis_contract']
    groups=context['valid_reference_combinations']
    assert len(groups)==4 and all(2<=len(g)<=3 and len(set(g))==len(g) for g in groups)
    for group in groups:
        chosen=answer(access);chosen['reading']['interpretation'][0]['evidence']=group
        jsonschema.validate(chosen,local.schema(access))
        reading,verification=access.render(chosen['reading'])
        assert access.verify(reading)==verification
    assert context['spatial_classification']=='model_estimate_no_metric_threshold'
    assert 'not evidence of proximity' in context['reference_combination_scope']


def test_render_also_rejects_mixed_or_repeated_hypotheses_without_relying_on_schema():
    access=BoundedAttackHypothesisSession(packet()).consultation()
    chosen=selection(access);chosen['interpretation']*=2
    with pytest.raises(ValueError):access.render(chosen)


def test_bounded_adapter_preserves_budgets_and_replays_a_real_consultation_protocol():
    session=BoundedAttackHypothesisSession(packet());access=session.consultation()
    client=FakeClient([{'action':'consult','indices':[0],'reading':None},answer(access)])
    result=local.interpret(session,client,version=local.BOUNDED_HYPOTHESIS_VERSION)
    assert result['status']=='mechanically_valid' and local.verify_replay(session,result)['causal_replay']
    assert client.bodies[0]['options']['num_ctx']==65536 and client.bodies[0]['options']['num_predict']==8192
    assert result['receipt']['reading_contract']=='stratai.replay_attack_hypothesis@2'
    assert result['verification']['representation']=='stratai.replay_attack_hypothesis@2'
    assert 'valid_reference_combinations' in str(access.initial())
    assert local.VERSION=='1.2.0-local.1'


def test_old_conditional_bank_and_abstention_still_work_without_hypothesis_rows():
    from ai_coach.interpretation.context_comparison import BANK
    access=BoundedAttackHypothesisSession(packet()).consultation();chosen=selection(access)
    chosen['interpretation']=[{'principle_id':row['id'],'evidence':['S0:3']} for row in BANK]
    jsonschema.validate({'action':'finish','indices':[],'reading':chosen},local.schema(access))
    assert len(access.render(chosen)[0]['interpretation'])==8
    chosen['interpretation']=[]
    assert access.render(chosen)[0]['interpretation']==[]


def test_a_complete_valid_response_cannot_repeat_an_evidence_id_even_when_order_changes():
    access=BoundedAttackHypothesisSession(packet()).consultation();chosen=answer(access)
    chosen['reading']['interpretation'][0]['evidence'].reverse()
    with pytest.raises(jsonschema.ValidationError):jsonschema.validate(chosen,local.schema(access))
    with pytest.raises(ValueError):access.render(chosen['reading'])
