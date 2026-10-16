"""Synthetic regression cases for the strict factual replay input.

Historical schemas have frozen receipts; no runtime capture or retained output
is used as a test dependency.
"""
from copy import deepcopy

import pytest

from ai_coach.interpretation.place_volumes import load_reference, resolve
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_replay_reading_context import packet as v1_packet, c4_packet as v2_packet
from ai_coach.tests.test_replay_place_context import packet as v3_packet
from ai_coach.tests.test_replay_round_context import packet as v4_packet
from ai_coach.tests.test_replay_events import packet as v5_packet


@pytest.mark.parametrize("factory,input_hash,packet_hash", [
    (v1_packet, "c9d2b7781664ea7e25ebb950e035cb2978eb552b0f435b46c782033a861792fb",
     "c4245ec98af5d68679b94428708c2c2fff50ae9e3c8a9e78eb022b09fcdde0c3"),
    (v2_packet, "eed7d641d72cf068e8cf8a25fd86da7685775b01d98d999419adebcebaf6540a",
     "2959ada78c2c70a125dddba8d775e913db25d344e68ec9b52257e6847b4e40ce"),
    (v3_packet, "5ee085efbce1c91527cee04d11ceb399cd625d8f0d808314b99baf26681c3e3b",
     "ef7a2a3ccb4c4f8b5410c6fd7f258ec9c9d798e0c916ba874c612e5a4304d347"),
    (v4_packet, "3afc79ae73b48f623e216222fab18411cfc8ee0fad970116bf3fdac99e7f47f5",
     "3259c990f2cbe55fe77215e410e41a4f6df93d0f4b455b16905dbc3439d243b3"),
    (v5_packet, "75bed1d4c05746ecea5fabc4236476be31de130ea918adebfb647c4e4b271916",
     "37079dc2db44bca5a45be7f9a6cf0a47f749306e6b8bf54778ad3dcc1f2de587"),
])
def test_factual_repair_preserves_all_historical_input_receipts(factory, input_hash, packet_hash):
    packet = factory()
    assert ReplayReadingSession(packet).receipt() == {
        "input_hash": input_hash,
        "projected_packet_hash": packet_hash,
        "version": packet["schema_id"],
        "training_allowed": False,
        "product_supported": False,
        "independent_evaluation": False,
    }


def factual_packet():
    """One valid synthetic own player; no retained replay data dependency."""
    packet = v5_packet()
    packet["schema_id"] = "stratai.replay_context_input@6"
    view = packet["views"][0]
    view.update(schema_id="stratai.team_replay_projection@8", tick=12, availability_tick=12)
    view["frame"].update(tick=12, time_remaining=None)
    own = view["frame"]["players"][0]
    view["frame"]["players"] = [own]
    own.update(alive=True, health=100, x=-1500, y=750, z=-40,
               armor=100, money=800, observation_tick=12, availability_tick=12,
               source={"kind": "demo_replay_frame", "reference": "round:1:frame:12:player:1"},
               weapon="LEGACY VISUAL CACHE", weapons=["LEGACY VISUAL CACHE"], has_helmet=False,
               inventory_context={"schema_id": "stratai.replay_inventory@1", "status": "observed",
                                  "observation_tick": 12, "availability_tick": 12,
                                  "source": {"kind": "demo_inventory_snapshot", "reference": "frame:12:inventory"},
                                  "weapon": "Glock-18", "weapons": ["Glock-18", "Knife"], "has_helmet": True})
    own["place"] = {**resolve(own, load_reference(), map_name="de_mirage", xy_precision=.5),
                    "observation_tick": 12, "availability_tick": 12, "xy_precision": .5}
    view["frame"]["round_context"].update(
        schema_id="stratai.replay_round_context@2", observation_tick=12, availability_tick=12,
        elapsed_seconds=2/64, clock_evidence={},
        source={"kind": "demo_round_context", "reference": "round:1:frame:12:round_context"})
    view["frame"]["bomb"].update(
        state="resolved", objective_phase="resolved", observation_tick=12, availability_tick=12,
        source={"kind": "demo_objective_state", "reference": "round:1:frame:12:bomb"})
    view["frame"]["round_context"]["eligibility"]["basis_tick"] = 12
    view["events"] = view["events"][-1:]
    event = view["events"][-1]
    event.update(tick=12, availability_tick=12, x=0, y=0, z=0, position_tick=12)
    return packet


def own_player(packet):
    return packet["views"][0]["frame"]["players"][0]


def inventory_fact(context):
    return next(f for f in context["situation"] if f["key"] == "own_inventory")


def test_factual_live_inventory_uses_the_separate_native_snapshot_not_visual_cache():
    packet = factual_packet()
    context = ReplayReadingSession(packet).consultation().initial()
    player = next(f for f in context["situation"] if f["key"] == "own_player")
    assert all(key not in player["value"] for key in ("weapon", "weapons", "has_helmet"))
    inventory = inventory_fact(context)
    assert inventory["subject"] == "P01"
    assert inventory["value"] == own_player(packet)["inventory_context"]
    assert inventory["status"] == "observed"
    assert inventory["seconds_in_window"] == 2/64
    assert "LEGACY VISUAL CACHE" not in str(context["situation"])


def test_factual_dead_inventory_is_dated_history_and_never_current_equipment():
    packet = factual_packet()
    player = own_player(packet)
    player.update(alive=False, health=0)
    player.pop("place")
    player["inventory_context"].update(status="last_observed", observation_tick=10, availability_tick=10,
                                       source={"kind": "demo_inventory_snapshot", "reference": "frame:10:inventory"})
    context = ReplayReadingSession(packet).consultation().initial()
    inventory = inventory_fact(context)
    assert inventory["status"] == "estimated"
    assert inventory["value"]["status"] == "last_observed"
    assert inventory["value"]["observation_tick"] == 10
    assert inventory["value"]["weapons"] == ["Glock-18", "Knife"]
    assert inventory["seconds_in_window"] == 0
    player_fact = next(f for f in context["situation"] if f["key"] == "own_player")
    assert player_fact["value"]["alive"] is False
    assert player_fact["value"]["health"] == 0
    assert "weapons" not in player_fact["value"]


def test_factual_alive_missing_inventory_observation_remains_historical():
    packet = factual_packet()
    own_player(packet)["inventory_context"].update(
        status="last_observed", observation_tick=10, availability_tick=10,
        source={"kind": "demo_inventory_snapshot", "reference": "frame:10:inventory"})
    inventory = inventory_fact(ReplayReadingSession(packet).consultation().initial())
    assert inventory["status"] == "estimated"
    assert inventory["seconds_in_window"] == 0


def test_factual_unavailable_inventory_cannot_be_filled_from_legacy_cache():
    packet = factual_packet()
    own_player(packet)["inventory_context"].update(status="unavailable", observation_tick=None,
                                                  availability_tick=None, source=None,
                                                  weapon=None, weapons=[], has_helmet=None)
    inventory = inventory_fact(ReplayReadingSession(packet).consultation().initial())
    assert inventory["status"] == "unknown"
    assert inventory["value"]["weapons"] == []
    assert inventory["value"]["weapon"] is None
    assert inventory["value"]["observation_tick"] is None


@pytest.mark.parametrize("source", [None, {}, "absent"])
def test_factual_unavailable_inventory_allows_only_absent_provenance(source):
    packet = factual_packet()
    value = own_player(packet)["inventory_context"]
    value.update(status="unavailable", observation_tick=None, availability_tick=None,
                 weapon=None, weapons=[], has_helmet=None)
    if source == "absent":
        value.pop("source")
    else:
        value["source"] = source
    assert inventory_fact(ReplayReadingSession(packet).consultation().initial())["status"] == "unknown"


@pytest.mark.parametrize("source", [
    {"kind": "demo_inventory_snapshot", "reference": "frame:999:inventory"},
    {"kind": "demo_inventory_snapshot", "reference": "frame:12:inventory"},
    "frame:12:inventory", True,
])
def test_factual_unavailable_inventory_rejects_manufactured_provenance(source):
    packet = factual_packet()
    value = own_player(packet)["inventory_context"]
    value.update(status="unavailable", observation_tick=None, availability_tick=None,
                 weapon=None, weapons=[], has_helmet=None, source=None)
    ReplayReadingSession(packet)
    value["source"] = source
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("status", ["observed", "unavailable"])
def test_factual_inventory_cannot_smuggle_unsupported_fields_into_facts(status):
    packet = factual_packet()
    value = own_player(packet)["inventory_context"]
    if status == "unavailable":
        value.update(status=status, observation_tick=None, availability_tick=None,
                     weapon=None, weapons=[], has_helmet=None, source=None)
    ReplayReadingSession(packet)
    value.update(current_ammo=30, enemy_has_awp=True, future_tick=999)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("source", [
    None, {},
    {"kind": "raw_enemy_snapshot", "reference": "round:1:frame:12:player:1"},
    {"kind": "demo_replay_frame", "reference": "round:1:frame:13:player:1"},
    {"kind": "demo_replay_frame", "reference": "round:1:frame:12:player:2"},
])
def test_factual_own_snapshot_source_binds_kind_time_and_participant(source):
    packet = factual_packet()
    ReplayReadingSession(packet)
    own_player(packet)["source"] = source
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("changes", [
    {"observation_tick": 13, "availability_tick": 13},
    {"observation_tick": 12, "availability_tick": 11},
    {"status": "observed", "observation_tick": 10, "availability_tick": 10},
    {"source": None},
    {"source": {"kind": "demo_inventory_snapshot", "reference": ""}},
])
def test_factual_inventory_rejects_future_stale_or_unattributed_current_observation(changes):
    packet = factual_packet()
    own_player(packet)["inventory_context"].update(changes)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


def test_factual_dead_player_cannot_claim_fresh_observed_inventory():
    packet = factual_packet()
    own_player(packet).update(alive=False, health=0)
    own_player(packet).pop("place")
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("changes", [
    {"health": -1}, {"health": 101}, {"health": True}, {"health": 80.5},
    {"health": 0, "alive": True}, {"health": 1, "alive": False}, {"alive": 1},
    {"x": float("inf")}, {"y": None}, {"z": True},
    {"observation_tick": 13, "availability_tick": 13},
    {"observation_tick": 12, "availability_tick": 11},
])
def test_factual_own_player_rejects_incoherent_health_position_or_time(changes):
    packet = factual_packet()
    own_player(packet).update(changes)
    if own_player(packet).get("alive") is False:
        own_player(packet).pop("place")
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


def test_factual_own_player_requires_complete_xyz():
    packet = factual_packet()
    own_player(packet).pop("z")
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("cross_team", [False, True])
def test_factual_frame_rejects_duplicate_identity_even_across_teams(cross_team):
    packet = factual_packet()
    duplicate = deepcopy(own_player(packet))
    if cross_team:
        duplicate.update(team="CT", knowledge="calculated_visual_contact")
    packet["views"][0]["frame"]["players"].append(duplicate)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


def test_factual_frame_rejects_distinct_raw_ids_that_collapse_to_one_local_alias():
    packet = factual_packet()
    duplicate = deepcopy(own_player(packet))
    duplicate["steam_id"] = "01"
    duplicate["source"]["reference"] = "round:1:frame:12:player:01"
    packet["views"][0]["frame"]["players"].append(duplicate)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


def test_factual_frame_allows_five_own_players_and_rejects_six():
    packet = factual_packet()
    players = packet["views"][0]["frame"]["players"]
    for identity in range(2, 6):
        player = deepcopy(players[0])
        player["steam_id"] = str(identity)
        player["source"]["reference"] = f"round:1:frame:12:player:{identity}"
        players.append(player)
    ReplayReadingSession(packet)
    sixth = deepcopy(players[0])
    sixth["steam_id"] = "6"
    sixth["source"]["reference"] = "round:1:frame:12:player:6"
    players.append(sixth)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("changes", [
    {"position_tick": 13}, {"position_tick": 11}, {"position_tick": True},
    {"x": None}, {"y": None}, {"z": float("inf")}, {"x": True},
])
def test_factual_own_event_position_requires_current_complete_finite_evidence(changes):
    packet = factual_packet()
    packet["views"][0]["events"][-1].update(changes)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("missing", ["x", "y", "z", "position_tick"])
def test_factual_gun_event_does_not_certify_partial_position(missing):
    packet = factual_packet()
    packet["views"][0]["events"][-1].pop(missing)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


def test_factual_zero_event_coordinates_are_observed_and_absence_stays_absent():
    packet = factual_packet()
    context = ReplayReadingSession(packet).consultation().initial()
    event = next(e for e in context["events"] if e["key"] == "weapon_fire")
    assert {k: event["value"][k] for k in ("x", "y", "z")} == {"x": 0, "y": 0, "z": 0}
    for key in ("x", "y", "z", "position_tick"):
        packet["views"][0]["events"][-1].pop(key)
    absent = next(e for e in ReplayReadingSession(packet).consultation().initial()["events"] if e["key"] == "weapon_fire")
    assert all(k not in absent["value"] for k in ("x", "y", "z", "position_tick"))


def test_factual_2d_inferno_event_does_not_require_invented_height():
    packet = factual_packet()
    event = packet["views"][0]["events"][-1]
    event.update(type="utility_start", utility_type="inferno", effect_id="r1-inferno-1")
    event.pop("z")
    context = ReplayReadingSession(packet).consultation().initial()
    effect = next(e for e in context["events"] if e["key"] == "utility_start")
    assert effect["value"]["x"] == 0 and effect["value"]["y"] == 0
    assert "z" not in effect["value"]


def test_factual_enemy_event_position_must_match_its_dated_contact():
    packet = factual_packet()
    event = packet["views"][0]["events"][-1]
    event.update(actor_id="2", actor_team="CT", knowledge="calculated_visual_event",
                 position_tick=10, observation={"observer_id": "1", "tick": 10,
                     "source": {"kind": "go_calculated_contact", "reference": "round:1:contact:10:1:2"}})
    ReplayReadingSession(packet)
    event["position_tick"] = 12
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("visible", [False, True])
def test_factual_damage_position_is_the_victim_at_callback_time(visible):
    packet = factual_packet()
    event = packet["views"][0]["events"][-1]
    event.update(type="player_hurt", victim_id="1", victim_team="T", damage=15, health=85)
    if visible:
        event.update(actor_id="2", actor_team="CT", knowledge="calculated_visual_event",
                     observation={"observer_id": "1", "tick": 10,
                                  "source": {"kind": "go_calculated_contact", "reference": "round:1:contact:10:1:2"}})
    ReplayReadingSession(packet)
    event["position_tick"] = 10
    with pytest.raises(ValueError, match="event position date differs"):
        ReplayReadingSession(packet)
