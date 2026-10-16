"""Opt-in transport keeps the original facts, limits and historical reader."""
import json

import pytest

from ai_coach.interpretation import context_comparison_ollama as local
from ai_coach.interpretation import replay_model_context as wire
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_replay_utility_information import packet
from ai_coach.tests.test_context_comparison_ollama import FakeClient


def test_opt_in_template_adapter_preserves_queries_receipt_and_generation_limits():
    session = ReplayReadingSession(packet())
    original = session.consultation().initial()
    selection = {key: [] for key in ('situation', 'evolution', 'interpretation', 'uncertainty')}
    client = FakeClient([{'action': 'consult', 'indices': [0], 'reading': None},
                         {'action': 'finish', 'indices': [], 'reading': selection}])
    result = local.interpret(session, client, version=local.TEMPLATE_VERSION)
    assert result['version'] == local.TEMPLATE_VERSION
    assert result['status'] == 'mechanically_valid'
    assert result['receipt'] == session.receipt()
    assert wire.encoded(wire.unpack(json.loads(client.bodies[0]['messages'][1]['content']))) == wire.encoded(original)
    delivered = wire.unpack(json.loads(client.bodies[1]['messages'][-1]['content']))
    assert delivered['consultation_results'][0]['facts'] == session._archive[0]
    assert client.bodies[0]['options']['num_ctx'] == 65536
    assert client.bodies[0]['options']['num_predict'] == 8192
    assert client.bodies[0]['truncate'] is client.bodies[0]['shift'] is False
    assert local.verify_replay(session, result)['causal_replay']
    assert local.VERSION == '1.2.0-local.1'
    assert 'preventive' in client.bodies[0]['messages'][0]['content']
    assert 'preventive' not in local.initial_messages(session.consultation())[0]['content']


def test_unknown_adapter_cannot_silently_choose_a_transport():
    session = ReplayReadingSession(packet())
    with pytest.raises(ValueError, match='adapter version'):
        local.interpret(session, FakeClient([]), version='unknown')
    with pytest.raises(ValueError, match='adapter version'):
        local.initial_messages(session.consultation(), version='unknown')
