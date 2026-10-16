"""Lossless transport must preserve causal evidence rather than prune it."""
from copy import deepcopy
import json

import pytest

from ai_coach.interpretation import replay_model_context as wire


def report():
    return {"id": "contact:64:9:2", "observer_id": "9", "subject_id": "2",
            "observation_tick": 64, "availability_tick": 64, "age_seconds": 1.015625,
            "position": {"x": -1765.505126953125, "y": 779.725830078125},
            "source": {"reference": "calculated_vision@2", "status": "estimated"},
            "recipients": [{"player_id": str(i), "reception": "possible",
                            "availability_tick": 64} for i in (1, 6, 7, 9, 10)]}


def test_repeated_report_keeps_all_recipients_times_and_exact_numbers():
    original = {"situation": [{"id": f"S240:{i}", "subject": f"P{i:02d}",
                               "observations": [report()]} for i in range(1, 6)],
                "earlier": {"observation_tick": 0}, "uncertainty": "current_position_unknown"}
    before = json.dumps(original, separators=(",", ":"))
    packed = wire.pack(original)
    assert packed["wire_schema"] == wire.VERSION
    assert len(json.dumps(packed)) < len(json.dumps(original)) * .75
    assert wire.unpack(json.loads(json.dumps(packed))) == original
    assert json.dumps(original, separators=(",", ":")) == before
    decoded = wire.unpack(packed)
    decoded["situation"][0]["observations"][0]["recipients"].clear()
    assert decoded["situation"][1]["observations"][0]["recipients"]


def test_reserved_markers_and_types_are_literal_source_data():
    original = {"$ref": "not_a_reference", "nested": {"$literal": [True, 1, 1.0, None]},
                "reports": [report()] * 5}
    packed = wire.pack(original)
    assert wire.encoded(wire.unpack(packed)) == wire.encoded(original)


@pytest.mark.parametrize("change", ["missing", "cycle", "extra", "hash", "version"])
def test_forged_dictionary_or_hash_is_rejected(change):
    packed = wire.pack({"reports": [report()] * 5})
    changed = deepcopy(packed)
    if change == "missing":
        changed["payload"] = {"$ref": "absent"}
    elif change == "cycle":
        changed["definitions"]["D0"] = {"$ref": "D0"}
    elif change == "extra":
        changed["definitions"]["unused"] = {"secret": "future"}
    elif change == "hash":
        changed["source_sha256"] = "0" * 64
    else:
        changed["wire_schema"] = "unknown"
    with pytest.raises(ValueError):
        wire.unpack(changed)


def test_small_packet_falls_back_without_losing_or_wrapping_fields():
    original = {"situation": [], "tick": 64, "unknown": None}
    assert wire.pack(original) == original
    assert wire.unpack(original) == original


def test_nonfinite_json_is_not_silently_encoded():
    with pytest.raises(ValueError):
        wire.pack({"tick": float("nan")})


def test_replay_transport_and_queries_preserve_receipt_and_causal_frontier():
    from ai_coach.interpretation import context_comparison_ollama as local
    from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
    from ai_coach.tests.test_replay_utility_information import packet
    from ai_coach.tests.test_context_comparison_ollama import FakeClient

    session = ReplayReadingSession(packet())
    original = session.consultation().initial()
    selection = {key: [] for key in ("situation", "evolution", "interpretation", "uncertainty")}
    client = FakeClient([{"action": "consult", "indices": [0], "reading": None},
                         {"action": "finish", "indices": [], "reading": selection}])
    result = local.interpret(session, client)
    assert wire.unpack(json.loads(client.bodies[0]["messages"][1]["content"])) == original
    delivered = wire.unpack(json.loads(client.bodies[1]["messages"][-1]["content"]))
    assert delivered["consultation_results"][0]["facts"] == session._archive[0]
    assert result["status"] == "mechanically_valid"
    assert result["receipt"] == session.receipt()
    assert local.verify_replay(session, result)["causal_replay"]
    with pytest.raises(ValueError, match="frontier"):
        session.consultation().sample(1)


def test_legacy_wire_and_invalid_finish_remain_verifiable_and_rejected():
    from ai_coach.interpretation import context_comparison_ollama as local
    from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
    from ai_coach.tests.test_replay_utility_information import packet
    from ai_coach.tests.test_context_comparison_ollama import FakeClient

    session = ReplayReadingSession(packet())
    access = session.consultation()
    legacy = local.initial_messages(access, version=local.LEGACY_VERSION)
    assert json.loads(legacy[1]["content"]) == access.initial()
    assert "$ref" not in legacy[0]["content"]
    answer = {"action": "finish", "indices": [0],
              "reading": {key: [] for key in ("situation", "evolution", "interpretation", "uncertainty")}}
    result = local.interpret(session, FakeClient([answer]))
    assert result["failure"] == "invalid finish envelope"
    assert result["reading"] is None
    assert local.verify_replay(session, result)["causal_replay"]


def test_source_wire_discriminator_is_escaped_even_without_savings():
    source = {"wire_schema": "a literal source field", "tick": 64}
    assert wire.unpack(wire.pack(source)) == source


def test_unique_events_share_columns_without_losing_any_event():
    original = {'events':[{"id": f"E{i}", "subject": "P01", "key": "weapon_fire",
                           "observation_tick": i, "availability_tick": i,
                           "seconds_in_window": i / 64, "position_status": "historical",
                           "source_reference": f"callback:{i}"} for i in range(89)]}
    packed=wire.pack(original)
    assert packed['columns']
    assert wire.encoded(wire.unpack(packed))==wire.encoded(original)
    assert len(wire.encoded(packed)) < len(wire.encoded(original)) * .65


def test_row_markers_remain_literal_and_dictionary_v1_is_reproducible():
    original={'$row':'literal','values':[1,2], 'nested':{'$row_literal':{'x':3}},'reports':[report()]*5}
    assert wire.encoded(wire.unpack(wire.pack(original)))==wire.encoded(original)
    previous=wire.pack(original,version=wire.SHARED_VERSION)
    assert previous['wire_schema']==wire.SHARED_VERSION
    assert wire.unpack(previous)==original


@pytest.mark.parametrize('mutation', ['missing', 'width'])
def test_missing_columns_and_wrong_row_width_are_rejected(mutation):
    original={'events':[dict(id=str(i),observation_tick=i,availability_tick=i,source_reference=f'event:{i}') for i in range(89)]}
    packed=wire.pack(original)
    assert packed['wire_schema']==wire.VERSION
    if mutation=='missing':packed['columns']={}
    else:next(iter(packed['columns'].values())).pop()
    with pytest.raises(ValueError):wire.unpack(packed)
