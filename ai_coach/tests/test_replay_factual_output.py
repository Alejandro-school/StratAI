"""Independent regression cases for protected, reference-only replay readings.

These fixtures describe facts directly, not by calling the new renderer or
loading a retained corpus. Wire tests use the real adapter without a model.
"""
from copy import deepcopy
import json

import pytest

from ai_coach.interpretation import context_comparison as comparison
from ai_coach.interpretation import context_comparison_ollama as adapter
from ai_coach.interpretation.replay_factual_reading import StrictConsultation


def fact(identity, subject, key, value, *, status="observed", seconds=15):
    return {"id": identity, "subject": subject, "key": key, "value": value,
            "status": status, "seconds_in_window": seconds}


class FactualSession:
    """C4 handoff plus an alive, injured player and uncertain geometry."""

    def __init__(self):
        self._version = "stratai.replay_context_input@6"
        start = [
            fact("S0:0", None, "objective", {"state": "carried", "carrier_id": "7", "objective_phase": "preplant"}, seconds=0),
            fact("S0:1", "P07", "own_player", {"team": "T", "alive": True, "health": 100,
                 "has_c4": True, "weapons": ["Glock-18", "C4", "Knife"]}, seconds=0),
            fact("S0:2", "P06", "own_player", {"team": "T", "alive": True, "health": 100,
                 "has_c4": False, "weapons": ["Glock-18", "Knife"]}, seconds=0),
            fact("S0:3", None, "round_remaining_seconds", 115, seconds=0),
        ]
        middle = [fact("S1:0", "P09", "own_player", {"team": "T", "alive": True, "health": 85}, seconds=14.59375)]
        current = [
            fact("S2:0", None, "objective", {"state": "carried", "carrier_id": "6", "objective_phase": "preplant"}),
            fact("S2:1", "P07", "own_player", {"team": "T", "alive": True, "health": 100,
                 "has_c4": False, "weapons": ["Glock-18", "Knife"]}),
            fact("S2:2", "P06", "own_player", {"team": "T", "alive": True, "health": 100,
                 "has_c4": True, "weapons": ["Glock-18", "C4", "Knife"]}),
            fact("S2:3", "P09", "own_player", {"team": "T", "alive": True, "health": 70,
                 "has_c4": False, "weapons": ["Glock-18", "Knife"]}),
            fact("S2:4", "P09", "place_context", {"status": "mapped", "zone": "Apartments",
                 "reference_id": "local-mirage-reference", "build_compatibility": "unverified"}, status="estimated"),
            fact("S2:5", None, "own_utility_tracking", {"projectiles": [{"id": "1", "type": "smoke",
                 "thrower_id": "1", "x": -806.28125, "y": 461.6875, "z": 934.46875,
                 "trajectory": [-405.40625, 726.125, -806.28125, 461.6875],
                 "knowledge": "simulated_team_utility_tracking"}], "active_effects": []}, status="simulated"),
            fact("S2:6", "P07", "place_context", {"status": "unavailable", "zone": None,
                 "candidates": [], "reference_id": "local-mirage-reference", "build_compatibility": "unverified",
                 "reason": "outside_place_volumes"}, status="unknown"),
            fact("S2:7", None, "own_alive_count", 5),
            fact("S2:8", None, "round_remaining_seconds", 100),
        ]
        self._events = [
            fact("E1", "P07", "bomb_drop", {"tick": 7278, "availability_tick": 7278,
                 "state_after": "dropped", "objective_phase": "preplant",
                 "knowledge": "own_team_objective_event"}, seconds=3.765625),
            fact("E2", "P06", "bomb_pickup", {"tick": 7285, "availability_tick": 7285,
                 "state_after": "carried", "objective_phase": "preplant",
                 "knowledge": "own_team_objective_event"}, seconds=3.875),
        ]
        self._archive = [start, middle, current]
        self._view = {"version": self._version, "instrument": self._version, "side": "T", "map": "de_mirage",
                      "through_index": 2, "situation": current, "events": self._events,
                      "sample_times_seconds": [0, 14.59375, 15],
                      "evolution": {"sample_count": 3, "duration_seconds": 15, "boundaries": {"start": start}},
                      "bank": {"principles": deepcopy(list(comparison.BANK)), "examples": deepcopy(list(comparison.EXAMPLES))},
                      "reading_constraints": [],
                      "consultation": {"operation": "sample(index)", "frontier": 2,
                                       "max_queries": comparison.QUERY_LIMIT, "max_response_bytes": comparison.QUERY_BYTES}}

    def consultation(self):
        return StrictConsultation(self)

    def receipt(self):
        return {"version": self._version, "training_allowed": False, "product_supported": False,
                "independent_evaluation": False}


def selection(**sections):
    return {key: sections.get(key, []) for key in comparison.SECTIONS}


def answer(reading):
    return {"action": "finish", "indices": [], "reading": reading}


def consume(access, chosen):
    response = {"model": adapter.MODEL, "done": True, "done_reason": "stop", "prompt_eval_count": 100,
                "message": {"role": "assistant", "content": json.dumps(answer(chosen))}}
    return adapter.consume(access, response, adapter.initial_messages(access), adapter.Config())


def facts_in(reading, section):
    return [item for claim in reading[section] for item in claim["facts"]]


def test_current_c4_and_historical_c4_keep_different_people_and_times():
    session = FactualSession()
    chosen = selection(situation=["S2:0", "S2:1", "S2:2"], evolution=["S0:0", "S0:1", "E1", "E2"])
    rendered, verification = consume(session.consultation(), chosen)
    assert facts_in(rendered, "situation") == session._archive[-1][:3]
    assert facts_in(rendered, "evolution") == session._archive[0][:2] + session._events
    assert rendered["situation"][1]["facts"][0]["value"]["has_c4"] is False
    assert verification["semantic_truth"] == "not_evaluated"
    assert verification["tactical_quality"] == "not_evaluated"


def test_start_facts_are_citable_without_redundant_consultation():
    access = FactualSession().consultation()
    initial = access.initial()
    assert initial["evolution"]["boundaries"]["start"][0]["id"] == "S0:0"
    rendered, verification = consume(access, selection(evolution=["S0:0", "S0:3"]))
    assert [f["seconds_in_window"] for f in facts_in(rendered, "evolution")] == [0, 0]
    assert verification["queries"] == []


@pytest.mark.parametrize("historical", ["S0:1", "E1"])
def test_historical_carrier_or_drop_cannot_be_selected_as_current_situation(historical):
    with pytest.raises((ValueError, adapter.jsonschema.ValidationError)):
        consume(FactualSession().consultation(), selection(situation=[historical]))


def test_intermediate_facts_require_consultation_before_they_can_be_selected():
    access = FactualSession().consultation()
    with pytest.raises((ValueError, adapter.jsonschema.ValidationError)):
        consume(access, selection(evolution=["S1:0"]))
    access.sample(1)
    reading, _ = consume(access, selection(evolution=["S1:0"]))
    assert facts_in(reading, "evolution")[0]["value"]["health"] == 85
    assert facts_in(reading, "evolution")[0]["seconds_in_window"] == 14.59375


def test_alive_health_cannot_be_relabelled_unknown_by_selecting_it_as_uncertainty():
    with pytest.raises((ValueError, adapter.jsonschema.ValidationError)):
        consume(FactualSession().consultation(), selection(uncertainty=["S2:3"]))
    reading, _ = consume(FactualSession().consultation(), selection(situation=["S2:3"]))
    value = facts_in(reading, "situation")[0]["value"]
    assert value["alive"] is True and value["health"] == 70
    assert "70" in reading["situation"][0]["text"]


def test_drop_is_rendered_as_drop_with_preplant_phase_and_exact_date():
    reading, _ = consume(FactualSession().consultation(), selection(evolution=["E1"]))
    drop = facts_in(reading, "evolution")[0]
    assert drop["key"] == "bomb_drop"
    assert drop["subject"] == "P07" and drop["seconds_in_window"] == 3.765625
    assert drop["value"]["state_after"] == "dropped" and drop["value"]["objective_phase"] == "preplant"
    assert "bomb_planted" not in reading["evolution"][0]["text"]


def test_estimated_unknown_and_simulated_places_and_utility_do_not_become_observed():
    session = FactualSession()
    reading, _ = consume(session.consultation(), selection(situation=["S2:4", "S2:5", "S2:6"],
                                                          uncertainty=["S2:4", "S2:5", "S2:6"]))
    for section in ("situation", "uncertainty"):
        assert facts_in(reading, section) == session._archive[-1][4:7]
        assert all(c["kind"] != "observed" for c in reading[section])
    smoke_claim = reading["situation"][1]
    assert "Apartments" not in smoke_claim["text"] and "BackAlley" not in smoke_claim["text"]


@pytest.mark.parametrize("section", comparison.SECTIONS)
@pytest.mark.parametrize("kind,text", [
    ("observed", "P07 tiene C4 en el inventario actual."),
    ("inferred", "El humo cruza Apartments; P09 está muerto."),
    ("limit", "No se sabe si P09 murió pese al daño recibido."),
])
def test_free_factual_prose_is_rejected_in_every_section_and_under_every_kind(section, kind, text):
    malicious = {"kind": kind, "text": text, "evidence": ["S2:3"], "principles": []}
    with pytest.raises((ValueError, adapter.jsonschema.ValidationError)):
        consume(FactualSession().consultation(), selection(**{section: [malicious]}))


@pytest.mark.parametrize("section", comparison.SECTIONS)
def test_unknown_fact_ids_are_rejected_everywhere(section):
    row = {"principle_id": "P02", "evidence": ["S999:99"]} if section == "interpretation" else "S999:99"
    with pytest.raises((ValueError, adapter.jsonschema.ValidationError)):
        consume(FactualSession().consultation(), selection(**{section: [row]}))


def test_interpretation_uses_only_existing_conditional_principles_not_new_factual_prose():
    chosen = selection(interpretation=[{"principle_id": "P02", "evidence": ["S2:4"]}])
    reading, verification = consume(FactualSession().consultation(), chosen)
    claim = reading["interpretation"][0]
    assert claim["principles"] == ["P02"] and claim["evidence"] == ["S2:4"]
    assert claim["kind"] != "observed"
    assert next(p["text"] for p in comparison.BANK if p["id"] == "P02") in claim["text"]
    assert verification["tactical_quality"] == "not_evaluated"
    assert verification["semantic_truth"] == "not_evaluated"


@pytest.mark.parametrize("field,value", [
    ("text", "Por tanto P07 aún tiene C4."),
    ("kind", "observed"),
    ("subject", "P07"),
    ("value", {"has_c4": True}),
])
def test_extra_fields_cannot_smuggle_false_premises_into_an_allowed_principle(field, value):
    chosen = selection(interpretation=[{"principle_id": "P02", "evidence": ["S2:1"], field: value}])
    with pytest.raises((ValueError, adapter.jsonschema.ValidationError)):
        consume(FactualSession().consultation(), chosen)


def test_a_new_unapproved_principle_is_rejected():
    with pytest.raises((ValueError, adapter.jsonschema.ValidationError)):
        consume(FactualSession().consultation(), selection(interpretation=[{"principle_id": "P999", "evidence": ["S2:1"]}]))


@pytest.mark.parametrize("section,chosen", [
    ("situation", selection(situation=["S2:3"])),
    ("evolution", selection(evolution=["E1"])),
    ("interpretation", selection(interpretation=[{"principle_id": "P02", "evidence": ["S2:3"]}])),
    ("uncertainty", selection(uncertainty=["S2:6"])),
])
def test_a_rendered_reading_cannot_be_tampered_with_after_reference_validation(section, chosen):
    access = FactualSession().consultation()
    rendered, _ = access.render(chosen)
    changed = deepcopy(rendered)
    changed[section][0]["text"] = "P09 está muerto y P07 conserva C4."
    with pytest.raises(ValueError):
        access.verify(changed)


def test_rendered_facts_cannot_change_value_subject_status_or_time():
    access = FactualSession().consultation()
    rendered, _ = access.render(selection(situation=["S2:3"]))
    for field, value in [("value", {"alive": False, "health": 0}), ("subject", "P07"),
                         ("status", "unknown"), ("seconds_in_window", 0)]:
        changed = deepcopy(rendered)
        changed["situation"][0]["facts"][0][field] = value
        with pytest.raises(ValueError):
            access.verify(changed)


def test_rendering_is_deterministic_and_does_not_mutate_the_session_or_selection():
    session = FactualSession()
    chosen = selection(situation=["S2:3", "S2:4"], evolution=["S0:0", "E1"])
    before_view, before_archive, before_selection = deepcopy(session._view), deepcopy(session._archive), deepcopy(chosen)
    first, first_check = session.consultation().render(chosen)
    second, second_check = session.consultation().render(chosen)
    assert first == second and first_check == second_check
    assert session._view == before_view and session._archive == before_archive and chosen == before_selection


def test_historical_protocol_is_still_free_prose_and_unchanged_for_legacy_access():
    session = FactualSession()
    session._version = "stratai.replay_context_input@5"
    session._view["version"] = session._view["instrument"] = session._version
    legacy = comparison.Consultation(session)
    claim = {"kind": "observed", "text": "Texto histórico conservado sin reparación retroactiva.",
             "evidence": ["S2:3"], "principles": []}
    original = selection(situation=[claim])
    rendered, verification = consume(legacy, original)
    assert rendered == original
    assert verification["semantic_truth"] == "not_evaluated"


def test_mandatory_current_facts_and_events_are_preserved_even_when_model_omits_them():
    session = FactualSession()
    session._view["required_reading_facts"] = {
        "situation": ["S2:0", "S2:1", "S2:2", "S2:3"],
        "evolution": ["E1", "E2"],
        "uncertainty": ["S2:6"],
    }
    reading, verification = consume(session.consultation(), selection())
    assert facts_in(reading, "situation") == session._archive[-1][:4]
    assert facts_in(reading, "evolution") == session._events
    assert facts_in(reading, "uncertainty") == [session._archive[-1][6]]
    assert reading["interpretation"] == []
    assert verification["factual_integrity"] == "checked_reference_rendering"


@pytest.mark.parametrize("section", ["situation", "evolution", "uncertainty"])
def test_removing_a_mandatory_fact_after_render_is_rejected_by_final_verification(section):
    session = FactualSession()
    session._view["required_reading_facts"] = {
        "situation": ["S2:3"], "evolution": ["E1"], "uncertainty": ["S2:6"],
    }
    access = session.consultation()
    reading, _ = access.render(selection())
    changed = deepcopy(reading)
    changed[section].clear()
    with pytest.raises(ValueError):
        access.verify(changed)


def test_mandatory_facts_are_not_duplicated_when_selected_explicitly():
    session = FactualSession()
    session._view["required_reading_facts"] = {"situation": ["S2:3"]}
    reading, _ = session.consultation().render(selection(situation=["S2:3"]))
    assert len(reading["situation"]) == 1
    assert facts_in(reading, "situation") == [session._archive[-1][3]]


def test_new_generations_cannot_use_the_unprotected_historical_replay_protocol():
    session = FactualSession()
    session._version = "stratai.replay_context_input@5"
    class NoNetwork:
        config = adapter.Config()
        def preflight(self):
            raise AssertionError('must reject the legacy generation before contacting any model')
    with pytest.raises(ValueError, match='historical replay'):
        adapter.interpret(session, NoNetwork())
