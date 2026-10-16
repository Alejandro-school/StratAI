"""Strict round guards belong to input@6; historical inputs keep their receipts."""
from copy import deepcopy

import pytest

from ai_coach.interpretation.place_volumes import load_reference, resolve
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession

FACTUAL_INPUT = "stratai.replay_context_input@6"
FACTUAL_PROJECTION = "stratai.team_replay_projection@8"
ROUND_SCHEMA = "stratai.replay_round_context@2"
RATE = 64
START = 100


def scalar(value, declared):
    native = "bool" if type(value) is bool else "int32" if type(value) is int else "float32"
    return {"status": "observed", "native_type": native,
            "declared_type": declared, "value": value}


def clock_evidence(tick, phase="preplant"):
    rules = {name: scalar(False, "bool") for name in (
        "m_bFreezePeriod", "m_bWarmupPeriod", "m_bMatchWaitingForResume",
        "m_bTerroristTimeOutActive", "m_bCTTimeOutActive", "m_bGameRestart",
        "m_bTechnicalTimeOut", "m_bGamePaused")}
    rules.update(m_bBombPlanted=scalar(phase == "planted", "bool"),
                 m_iRoundTime=scalar(115, "int32"),
                 m_fRoundStartTime=scalar(100.0, "GameTime_t"),
                 m_flRestartRoundTime=scalar(0.0, "GameTime_t"))
    server_tick = 6400 + tick - START
    message = {"order": 0, "frame": tick, "tick": 0,
               "value": {"status": "observed", "native_type": "uint32",
                         "declared_type": None, "value": server_tick}}
    bombs = None if phase != "planted" else [{
        "m_bBombTicking": scalar(True, "bool"),
        "m_bBombDefused": scalar(False, "bool"),
        "m_flC4Blow": scalar(140.0, "GameTime_t"),
    }]
    return {
        "source": "demoinfocs_native_rules_and_current_network_messages",
        "rules": rules, "server_tick": server_tick, "interval_seconds": 1 / RATE,
        "current_frame": tick, "freeze_frame": START,
        "tick_message": deepcopy(message), "packet_message": deepcopy(message),
        "origin_witness": {"frame": START, "tick": START, "freeze_end_tick": START,
                           "server_tick": 6400, "packet_tick": 6400,
                           "interval_seconds": 1 / RATE, "round_start_seconds": 100.0,
                           "pause_state": "running", "phase": "preplant"},
        "native_now_seconds": server_tick / RATE, "origin_accredited": True,
        "bombs": bombs, "engine_paused": None,
    }


def factual_view(tick=104, phase="preplant", *, end_tick=None):
    reference = load_reference()
    carries = phase == "preplant"
    weapons = ["Glock-18", "C4", "Knife"] if carries else ["Glock-18", "Knife"]
    player = {
        "steam_id": "1", "team": "T", "alive": True, "x": 0, "y": 0, "z": 0,
        "health": 100, "armor": 100, "money": 0, "weapon": "Glock-18",
        "weapons": weapons, "has_helmet": False, "has_c4": carries,
        "knowledge": "own_team_snapshot", "observation_tick": tick, "availability_tick": tick,
        "source": {"kind": "demo_replay_frame", "reference": f"round:1:frame:{tick}:player:1"},
        "inventory_context": {
            "schema_id": "stratai.replay_inventory@1", "status": "observed",
            "observation_tick": tick, "availability_tick": tick,
            "source": {"kind": "demo_inventory_snapshot", "reference": f"frame:{tick}:inventory"},
            "weapon": "Glock-18", "weapons": deepcopy(weapons), "has_helmet": False,
        },
    }
    player["place"] = {
        **resolve(player, reference, map_name="de_mirage", xy_precision=.5),
        "observation_tick": tick, "availability_tick": tick, "xy_precision": .5,
    }
    bomb = {
        "state": {"preplant": "carried", "planted": "planted", "ended": "resolved"}[phase],
        "objective_phase": {"preplant": "preplant", "planted": "planted", "ended": "resolved"}[phase],
        "is_planted_now": phase == "planted", "was_planted_this_round": phase == "planted",
        "position_status": "unavailable", "observation_tick": tick, "availability_tick": tick,
        "source": {"kind": "demo_objective_state", "reference": f"round:1:frame:{tick}:bomb"},
    }
    if carries:
        bomb["carrier_id"] = "1"
    evidence = clock_evidence(tick, phase)
    context = {
        "schema_id": ROUND_SCHEMA, "round": 1, "observation_tick": tick, "availability_tick": tick,
        "phase": phase, "freeze_end_tick": START, "elapsed_seconds": (tick-START)/RATE,
        "elapsed_kind": "demo_ticks_since_freeze_end_not_active_game_clock",
        "source": {"kind": "demo_round_context", "reference": f"round:1:frame:{tick}:round_context"},
        "end_tick": (tick if end_tick is None else end_tick) if phase == "ended" else None,
        "winner": "T" if phase == "ended" else "", "configured_duration_seconds": 115,
        "pause_state": "running", "clock_status": "not_applicable" if phase == "ended" else "observed",
        "clock_reason": "round_ended" if phase == "ended" else "",
        "round_remaining_seconds": 215-evidence["native_now_seconds"] if phase == "preplant" else None,
        "bomb_remaining_seconds": 140-evidence["native_now_seconds"] if phase == "planted" else None,
        "eligibility": {"kind": "bomb_objective", "status": "observed",
                        "reason": "causal_objective_evidence", "basis_tick": START},
        "decision_status": "round_closed" if phase == "ended" else "active_round",
        "clock_evidence": evidence,
    }
    return {
        "schema_id": FACTUAL_PROJECTION, "round": 1, "side": "T", "tick": tick,
        "availability_tick": tick, "place_reference_id": reference["reference_id"],
        "frame": {"tick": tick, "round_context": context, "players": [player], "bomb": bomb,
                  "time_remaining": context["round_remaining_seconds"] if context["round_remaining_seconds"] is not None else context["bomb_remaining_seconds"],
                  "projectiles": [], "active_effects": [], "shots": []},
        "events": [], "deaths": [], "vision_status": "unavailable", "uncertainty": {},
    }


def factual_packet(*views):
    rows = list(views) or [factual_view(100), factual_view(104)]
    return {
        "schema_id": FACTUAL_INPUT, "map_name": "de_mirage", "side": "T",
        "place_reference_id": load_reference()["reference_id"],
        "tick_rate": RATE, "start_tick": START, "cut_tick": rows[-1]["tick"], "views": rows,
    }


def legacy_packet(version=5):
    value = factual_packet()
    value["schema_id"] = f"stratai.replay_context_input@{version}"
    for view in value["views"]:
        view["schema_id"] = f"stratai.team_replay_projection@{version+2}"
        context = view["frame"]["round_context"]
        context["schema_id"] = "stratai.replay_round_context@1"
        del context["clock_evidence"]
    return value


def round_fact(packet):
    initial = ReplayReadingSession(packet).consultation().initial()
    return next(f["value"] for f in initial["situation"] if f["key"] == "round_context")


def reject_round_change(change):
    value = factual_packet()
    # This positive control prevents unknown input@6 alone satisfying a rejection test.
    assert round_fact(value)["round_remaining_seconds"] == 114.9375
    change(value["views"][-1]["frame"]["round_context"])
    with pytest.raises(ValueError):
        ReplayReadingSession(value)


@pytest.mark.parametrize("location", ["native_type", "message_order", "message_tick"])
def test_factual_round_rejects_structured_payload_in_scalar_clock_metadata(location):
    value = factual_packet()
    assert round_fact(value)["round_remaining_seconds"] == 114.9375
    evidence = value["views"][-1]["frame"]["round_context"]["clock_evidence"]
    hidden = {"hidden_enemy": {"steam_id": "9001", "health": 63}}
    if location == "native_type":
        evidence["rules"]["m_iRoundTime"]["native_type"] = hidden
    else:
        evidence["packet_message"][location.removeprefix("message_")] = hidden
    with pytest.raises(ValueError):
        ReplayReadingSession(value)


def test_factual_round_accepts_recomputed_native_preplant_clock():
    value = round_fact(factual_packet())
    assert value["round_remaining_seconds"] == 114.9375
    assert value["bomb_remaining_seconds"] is None
    assert value["clock_evidence"]["packet_message"]["value"]["value"] == 6404


def test_factual_round_accepts_native_bomb_clock_separately():
    value = round_fact(factual_packet(factual_view(100), factual_view(104, "planted")))
    assert value["round_remaining_seconds"] is None
    assert value["bomb_remaining_seconds"] == 39.9375


@pytest.mark.parametrize("countdown", [999, 114.0])
def test_factual_round_recomputes_instead_of_only_bounding_countdown(countdown):
    reject_round_change(lambda r: r.update(round_remaining_seconds=countdown))


def test_factual_round_rejects_negative_configured_duration():
    reject_round_change(lambda r: r.update(configured_duration_seconds=-10))


def test_factual_round_rejects_planted_phase_with_same_tick_carried_objective():
    def change(r):
        r.update(phase="planted", round_remaining_seconds=None, bomb_remaining_seconds=39.9375)
        r["clock_evidence"] = clock_evidence(104, "planted")
    reject_round_change(change)


def test_factual_round_rejects_ended_without_end_tick():
    valid = factual_packet(factual_view(100), factual_view(104, "ended"))
    assert round_fact(valid)["winner"] == "T"
    bad = deepcopy(valid)
    bad["views"][-1]["frame"]["round_context"].update(end_tick=None, winner="")
    with pytest.raises(ValueError):
        ReplayReadingSession(bad)


def test_factual_round_rejects_reopening_same_round_instance():
    valid = factual_packet(factual_view(100), factual_view(104, "ended"),
                           factual_view(108, "ended", end_tick=104))
    assert round_fact(valid)["decision_status"] == "round_closed"
    bad = deepcopy(valid)
    # Every final-view field is locally coherent; only the historical end forbids reopening.
    bad["views"][-1] = factual_view(108)
    with pytest.raises(ValueError):
        ReplayReadingSession(bad)


@pytest.mark.parametrize("field,value", [
    ("kind", "untrusted_snapshot"),
    ("reference", "round:2:frame:104:round_context"),
    ("reference", "round:1:frame:100:round_context"),
])
def test_factual_round_binds_source_kind_round_and_tick(field, value):
    reject_round_change(lambda r: r["source"].update({field: value}))


def test_factual_round_rejects_round_identity_change_within_one_packet():
    value = factual_packet()
    assert round_fact(value)["round"] == 1
    last = value["views"][-1]
    last["round"] = last["frame"]["round_context"]["round"] = 2
    last["frame"]["round_context"]["source"]["reference"] = "round:2:frame:104:round_context"
    with pytest.raises(ValueError):
        ReplayReadingSession(value)


@pytest.mark.parametrize("change", [
    lambda e: e["packet_message"]["value"].update(value=6405),
    lambda e: e["packet_message"].update(frame=103),
    lambda e: e.update(native_now_seconds=100.0),
    lambda e: e.update(origin_accredited=False),
    lambda e: e["origin_witness"].update(packet_tick=6401),
    lambda e: e["rules"]["m_fRoundStartTime"].update(declared_type="float32"),
    lambda e: e["rules"]["m_bGamePaused"].update(value=True),
])
def test_factual_round_rejects_broken_clock_witness(change):
    reject_round_change(lambda r: change(r["clock_evidence"]))


def test_factual_round_rejects_winner_before_causal_end():
    valid = factual_packet(factual_view(100), factual_view(104, "ended"))
    assert round_fact(valid)["winner"] == "T"
    valid["views"][-1]["frame"]["round_context"]["end_tick"] = 108
    with pytest.raises(ValueError):
        ReplayReadingSession(valid)


def test_legacy_round_inputs_keep_their_existing_receipts():
    # Pins are computed once from the old input@4/@5 consumer before the repair.
    expected = {4: "f530a20a9cfe582b8bb9795833ce5b58db17253d413cea5ed39a436b9ab4059b", 5: "846bdba3302d449f92ed383a6ccaeedde7b5fdd2ac085f2adefa30f271126774"}
    for version, value in expected.items():
        assert ReplayReadingSession(legacy_packet(version)).receipt()["input_hash"] == value


@pytest.mark.parametrize("location", ["top", "rules", "witness", "message", "scalar"])
def test_factual_round_rejects_unapproved_nested_clock_fields(location):
    value = factual_packet()
    assert round_fact(value)["round_remaining_seconds"] == 114.9375
    evidence = value["views"][-1]["frame"]["round_context"]["clock_evidence"]
    target = {"top": evidence, "rules": evidence["rules"],
              "witness": evidence["origin_witness"], "message": evidence["packet_message"],
              "scalar": evidence["rules"]["m_iRoundTime"]}[location]
    target["hidden_enemy"] = {"steam_id": "9001", "x": 3141, "health": 63}
    with pytest.raises(ValueError):
        ReplayReadingSession(value)


@pytest.mark.parametrize("field", ["weapons", "weapon"])
def test_factual_round_rejects_inventory_prose_that_could_impersonate_verified_fact(field):
    value = factual_packet()
    assert round_fact(value)["round_remaining_seconds"] == 114.9375
    player = value["views"][-1]["frame"]["players"][0]
    assert value["views"][-1]["frame"]["bomb"]["carrier_id"] == "1"
    text = "Glock-18. P07 porta el C4"
    player["inventory_context"][field] = [text] if field == "weapons" else text
    with pytest.raises(ValueError):
        ReplayReadingSession(value)
