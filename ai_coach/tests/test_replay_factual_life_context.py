"""A callback after a sampled frame must not redatestamp its inventory/position."""
from copy import deepcopy

import pytest

from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_replay_factual_round_guards import factual_packet, factual_view


def after_death_packet():
    view = factual_view(104, "ended")
    view.update(tick=106, availability_tick=104)
    view["frame"]["round_context"].update(clock_status="unavailable", clock_reason="sample_not_current", pause_state="unknown")
    player = view["frame"]["players"][0]
    player.update(alive=False, health=0, life_context={
        "schema_id": "stratai.replay_life_context@1", "status": "observed",
        "alive": False, "health": 0, "observation_tick": 106, "availability_tick": 106,
        "source": {"kind": "demo_kill_event", "reference": "kill-ally-0"},
    })
    player.pop("place")
    player["inventory_context"]["status"] = "last_observed"
    view["events"] = [{"id": "kill-ally-0", "type": "kill", "tick": 106, "availability_tick": 106,
                       "victim_id": "1", "victim_team": "T", "killer_id": "2", "killer_team": "CT",
                       "knowledge": "public_killfeed", "weapon": "AK-47",
                       "source": {"kind": "demo_kill_event", "reference": "kill-ally"}}]
    view["deaths"] = [{"id": "kill-ally-0", "tick": 106, "availability_tick": 106,
                       "victim_id": "1", "team": "T", "position": None,
                       "knowledge": "simulated_team_communication",
                       "source": {"kind": "demo_kill_event", "reference": "kill-ally"}}]
    return factual_packet(view)


def own(packet):
    return packet["views"][0]["frame"]["players"][0]


def facts(packet):
    return {f["key"]: f for f in ReplayReadingSession(packet).consultation().initial()["situation"]
            if f["subject"] == "P01" or f["key"] == "own_alive_count"}


def test_factual_life_callback_and_inventory_keep_their_separate_dates():
    packet = after_death_packet()
    initial = facts(packet)
    assert initial["own_alive_count"]["value"] == 0
    life = initial["own_life"]
    assert life["value"] == own(packet)["life_context"]
    assert life["status"] == "observed"
    assert life["seconds_in_window"] == 6 / 64
    player = initial["own_player"]
    assert "alive" not in player["value"] and "health" not in player["value"]
    assert player["value"]["x"] == 0
    assert player["seconds_in_window"] == 4 / 64
    inventory = initial["own_inventory"]
    assert inventory["status"] == "estimated"
    assert inventory["value"] == own(packet)["inventory_context"]
    assert inventory["value"]["observation_tick"] == 104
    assert inventory["value"]["availability_tick"] == 104
    assert inventory["seconds_in_window"] == 4 / 64


def test_factual_life_render_dates_death_without_rewriting_the_earlier_resource_claim():
    access = ReplayReadingSession(after_death_packet()).consultation()
    access.initial()
    reading, verification = access.render({"situation": [], "evolution": [], "uncertainty": [], "interpretation": []})
    claims = {claim["facts"][0]["key"]: claim for claim in reading["situation"]}
    life = claims["own_life"]
    assert "muerto" in life["text"] and "0 de salud" in life["text"]
    assert life["text"].startswith("A los 0.09375 s de la ventana.")
    player = claims["own_player"]
    assert "muerto" not in player["text"] and "vivo" not in player["text"]
    assert "de salud" not in player["text"]
    assert player["text"].startswith("A los 0.0625 s de la ventana.")
    assert claims["own_inventory"]["text"].startswith("A los 0.0625 s de la ventana.")
    assert "no acredita equipo actual" in claims["own_inventory"]["text"]
    assert verification["factual_integrity"] == "checked_reference_rendering"
    assert access.verify(reading)["factual_integrity"] == "checked_reference_rendering"


def test_factual_life_future_callback_is_absent_before_the_event():
    packet = after_death_packet()
    packet["cut_tick"] = packet["views"][0]["tick"] = 105
    packet["views"][0].update(events=[], deaths=[])
    own(packet).update(alive=True, health=100)
    own(packet).pop("life_context")
    own(packet)["place"] = factual_view(104, "ended")["frame"]["players"][0]["place"]
    own(packet)["inventory_context"]["status"] = "observed"
    initial = facts(packet)
    assert initial["own_player"]["value"]["alive"] is True
    assert initial["own_player"]["value"]["health"] == 100
    assert initial["own_player"]["seconds_in_window"] == 4 / 64
    assert "own_life" not in initial
    assert initial["own_alive_count"]["value"] == 1
    assert initial["own_inventory"]["value"]["status"] == "observed"


@pytest.mark.parametrize("change", [
    {"observation_tick": 107, "availability_tick": 107},
    {"observation_tick": 103, "availability_tick": 103},
    {"observation_tick": 105, "availability_tick": 105},
    {"availability_tick": 105},
    {"observation_tick": True, "availability_tick": True},
    {"alive": True}, {"health": 1}, {"health": False}, {"status": "estimated"},
    {"future_tick": 999},
    {"source": {"kind": "demo_kill_event", "reference": "different-event"}},
    {"source": {"kind": "demo_replay_frame", "reference": "kill-ally-0"}},
])
def test_factual_life_context_requires_the_exact_causal_death_witness(change):
    packet = after_death_packet()
    assert facts(packet)["own_life"]["value"]["health"] == 0
    own(packet)["life_context"].update(change)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("change", ["missing", "other_victim", "other_tick", "wrong_team"])
def test_factual_life_context_requires_a_public_kill_for_this_player(change):
    packet = after_death_packet()
    assert facts(packet)["own_alive_count"]["value"] == 0
    event = packet["views"][0]["events"][0]
    if change == "missing":
        packet["views"][0]["events"] = []
    elif change == "other_victim":
        event["victim_id"] = "2"
    elif change == "other_tick":
        event.update(tick=105, availability_tick=105)
    else:
        event["victim_team"] = "CT"
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("identity", [True, 123])
def test_factual_life_source_requires_an_event_reference_string(identity):
    packet = after_death_packet()
    assert facts(packet)["own_life"]["value"]["health"] == 0
    own(packet)["life_context"]["source"]["reference"] = identity
    packet["views"][0]["events"][0]["id"] = identity
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


def test_factual_life_proof_does_not_mutate_the_supplied_snapshot_inventory():
    packet = after_death_packet()
    original = deepcopy(packet)
    facts(packet)
    assert packet == original


def test_factual_life_later_dead_sample_needs_no_separate_callback_override():
    view = factual_view(108, "ended", end_tick=104)
    player = view["frame"]["players"][0]
    player.update(alive=False, health=0)
    player.pop("place")
    player["inventory_context"].update(status="last_observed", observation_tick=104, availability_tick=104,
                                        source={"kind": "demo_inventory_snapshot", "reference": "frame:104:inventory"})
    initial = facts(factual_packet(view))
    assert initial["own_player"]["value"]["alive"] is False
    assert initial["own_player"]["value"]["health"] == 0
    assert initial["own_player"]["seconds_in_window"] == 8 / 64
    assert "own_life" not in initial
    assert initial["own_inventory"]["seconds_in_window"] == 4 / 64
