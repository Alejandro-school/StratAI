"""Candidate factual context rejects mismatched provenance and future evidence."""

import hashlib
import json
from copy import deepcopy

import pytest
from jsonschema import ValidationError

from ai_coach.interpretation import candidate_context as candidate
from ai_coach.interpretation import review_packet_v3, temporal
from ai_coach.tests.test_contextual_interpretation import episode, inputs


def fixture(*, death=False):
    support, sources = inputs()
    if death:
        support["observations"]["frames"][4]["players"][1]["alive"] = False
        sources["physical_source"]["observations"]["frames"][4]["players"][1][
            "context"
        ] = None
        sources["c4_source"]["observations"]["states"][4]["players"][1].update(
            pawn_life=None,
            pawn_handle=None,
            inventory_status="pawn_unavailable",
            inventory_length=None,
        )
    value = episode(support, sources)
    packet = review_packet_v3.build(
        value,
        support,
        sources,
        None,
        "de_mirage",
        {review_packet_v3.rounds.DATA: None, review_packet_v3.rounds.clock.DATA: None},
    )
    common = {
        "case_id": "E03",
        "source_alias": value["source_alias"],
        "source_sha256": value["source_sha256"],
        "frame_count": 5,
        "side": "T",
    }
    refs = {"support_sha256": "a" * 64, "canonical_manifest_sha256": "b" * 64}
    kills = {
        **common,
        "schema_id": "stratai.review_kill_context@1",
        "events": [],
        "perception_proven": False,
        "omissions": {},
        "source_refs": {**refs, "combat_sha256": "c" * 64, "combat_record_count": 1},
    }
    extra = {
        **common,
        "schema_id": "stratai.review_extra_context@1",
        "damage": [],
        "c4": [],
        "perception_proven": False,
        "omissions": {},
        "source_refs": {
            **refs,
            "combat_events_sha256": "c" * 64,
            "combat_events_record_count": 1,
            "objective_events_sha256": "d" * 64,
            "objective_events_record_count": 0,
        },
    }
    bomb = {
        **common,
        "schema_id": "stratai.review_bomb_context@1",
        "source": "replay_bomb_native_snapshot_exact_tick",
        "source_refs": {
            **refs,
            "replay_round_sha256": "e" * 64,
            "replay_frame_count": 5,
        },
        "frames": [
            {
                "index": i,
                "source_sequence": row["cut"]["sequence"],
                "tick": row["cut"]["tick"],
                "round": row["cut"]["round_number"],
                "status": "not_dropped",
                "position": None,
            }
            for i, row in enumerate(value["samples"])
        ],
        "counts": {"not_dropped": 5},
    }
    return packet, {
        "case_id": "E03",
        "instrument": "stratai.review_web@9",
        "kills": kills,
        "extra": extra,
        "bomb": bomb,
    }


def rehash(packet):
    value = packet["context"]
    value["context_id"] = temporal.digest(
        {k: v for k, v in value.items() if k != "context_id"}
    )


def kill_event(packet):
    sample = packet["context"]["samples"][-1]
    victim = sample["actors"][1]
    return {
        "id": "kill-" + "1" * 20,
        "available_index": 4,
        "available_sequence": 5,
        "tick": sample["cut"]["tick"],
        "round": 1,
        "elapsed_seconds": packet["round_context"][-1]["round_elapsed_seconds"][
            "value"
        ],
        "attacker": {
            "label": "CT1",
            "side": "CT",
            "life": "ct-life",
            "position": {"x": 100, "y": 200, "z": 0},
        },
        "victim": {
            "label": "T2",
            "side": "T",
            "life": victim["identity"]["pawn_life"],
            "position": {"x": 20, "y": 0, "z": 0},
        },
        "relation": "enemy",
        "weapon": "AK-47",
        "weapon_is_utility": False,
        "through_smoke": False,
        "penetrated_objects": 0,
        "source": "kill_callback",
        "position_meaning": "event_position_not_current_or_confirmed_perception",
    }


def remove_c4(packet):
    for sample, objective in zip(
        packet["context"]["samples"], packet["objective"], strict=True
    ):
        objective["carrier_aliases"] = []
        for actor in sample["actors"]:
            if actor["has_c4"]["status"] == "observed":
                actor["has_c4"]["value"] = False
    rehash(packet)


def test_full_prefix_partial_counts_and_missing_clocks_are_preserved():
    packet, kwargs = fixture()
    before = deepcopy((packet, kwargs))
    value = candidate.build(packet, **kwargs)
    assert value["packet"] == packet
    assert value["snapshot"]["team_status"]["opponents"]["alive"] is None
    assert value["snapshot"]["team_status"]["opponents"]["unknown"] == 4
    assert (
        value["snapshot"]["round_context"]["native_clock"]["round_remaining_seconds"][
            "value"
        ]
        is None
    )
    assert value["snapshot"]["objective"]["state"] == "carried"
    assert value["input_files"] == {}
    assert candidate.validate(value) == value
    assert (packet, kwargs) == before
    value["packet"]["objective"][0]["planted"] = True
    assert (packet, kwargs) == before


def test_planted_e24_like_phase_is_explicit_without_inventing_one_v_one():
    packet, kwargs = fixture()
    remove_c4(packet)
    packet["objective"][-1]["planted"] = True
    value = candidate.build(packet, **kwargs)
    assert value["snapshot"]["objective"]["phase"] == "postplant"
    assert value["snapshot"]["objective"]["state"] == "planted"
    assert value["snapshot"]["team_status"]["opponents"]["status"] == "partial"


def test_unknown_round_is_preserved_not_labeled_knife():
    packet, kwargs = fixture()
    for row in packet["round_context"]:
        row["eligibility"] = {
            "kind": "unknown",
            "analysis_allowed": False,
            "reason": "round_kind_unconfirmed",
            "basis_sequence": row["source_sequence"],
        }
    packet["analysis_scope"] = "pending_context"
    value = candidate.build(packet, **kwargs)
    assert value["snapshot"]["round_context"]["eligibility"]["kind"] == "unknown"
    assert value["packet"]["analysis_scope"] == "pending_context"


def test_no_carrier_and_no_plant_does_not_invent_ground_bomb():
    packet, kwargs = fixture()
    remove_c4(packet)
    value = candidate.build(packet, **kwargs)
    assert value["snapshot"]["objective"]["phase"] == "unknown"
    assert value["snapshot"]["objective"]["state"] == "not_planted_unresolved"


def test_ground_position_requires_exact_current_object_and_no_carrier():
    packet, kwargs = fixture()
    remove_c4(packet)
    kwargs["bomb"]["frames"][-1].update(
        status="dropped_position", position={"x": 111, "y": 222}
    )
    kwargs["bomb"]["counts"] = {"not_dropped": 4, "dropped_position": 1}
    value = candidate.build(packet, **kwargs)
    assert value["snapshot"]["objective"]["state"] == "dropped"
    assert value["snapshot"]["objective"]["ground"]["position"] == {"x": 111, "y": 222}
    packet["objective"][-1]["planted"] = True
    with pytest.raises(ValueError, match="ground C4 conflicts"):
        candidate.build(packet, **kwargs)


def test_e03_historical_instrument_excludes_later_layers():
    packet, kwargs = fixture()
    kwargs["instrument"] = "stratai.review_web@5"
    with pytest.raises(ValueError, match="@5 excludes"):
        candidate.build(packet, **kwargs)
    kwargs.update(extra=None, bomb=None)
    value = candidate.build(packet, **kwargs)
    assert value["supplements"] == {
        "kills": kwargs["kills"],
        "extra": None,
        "bomb": None,
    }
    assert set(value["input_content_hashes"]) == {"packet", "kills"}
    kwargs["instrument"] = "stratai.review_web@9"
    with pytest.raises(ValueError, match="@9 requires"):
        candidate.build(packet, **kwargs)


def test_enemy_event_is_dated_and_never_current_tracking_or_perception():
    packet, kwargs = fixture(death=True)
    kwargs["kills"]["events"] = [kill_event(packet)]
    value = candidate.build(packet, **kwargs)
    event = value["events"][0]
    assert event["event"]["attacker"]["position"] == {"x": 100, "y": 200, "z": 0}
    assert event["availability_cut"] == packet["context"]["samples"][-1]["cut"]
    assert event["demo_seconds_since_event"] == 0
    assert (
        event["position_meaning"]
        == "event_position_not_current_or_confirmed_perception"
    )
    assert all(a["identity"]["side"] == "T" for a in value["snapshot"]["actors"])
    assert value["perception_proven"] is False


@pytest.mark.parametrize(
    "field,new",
    [
        ("available_index", 5),
        ("available_sequence", 6),
        ("tick", 9999),
        ("tick", 210),
        ("round", 2),
        ("elapsed_seconds", 9999),
    ],
)
def test_future_or_misaligned_event_is_rejected(field, new):
    packet, kwargs = fixture(death=True)
    event = kill_event(packet)
    event[field] = new
    kwargs["kills"]["events"] = [event]
    with pytest.raises(ValueError):
        candidate.build(packet, **kwargs)


@pytest.mark.parametrize(
    "layer,field,new",
    [
        ("kills", "source_sha256", "f" * 64),
        ("kills", "case_id", "E04"),
        ("extra", "side", "CT"),
        ("bomb", "frame_count", 6),
    ],
)
def test_wrong_source_case_side_and_prefix_are_rejected(layer, field, new):
    packet, kwargs = fixture()
    kwargs[layer][field] = new
    with pytest.raises(ValueError):
        candidate.build(packet, **kwargs)


def test_changed_life_and_duplicate_events_fail_closed():
    packet, kwargs = fixture(death=True)
    event = kill_event(packet)
    event["victim"]["life"] = "another-life"
    kwargs["kills"]["events"] = [event]
    with pytest.raises(ValueError, match="actor identity"):
        candidate.build(packet, **kwargs)
    kwargs["kills"]["events"] = [kill_event(packet), kill_event(packet)]
    with pytest.raises(ValueError, match="duplicate"):
        candidate.build(packet, **kwargs)


@pytest.mark.parametrize(
    "axis", ["availability_tick", "availability_frame", "instance", "epoch"]
)
def test_context_provenance_checks_frame_tick_and_instance(axis):
    packet, kwargs = fixture()
    packet["context"]["samples"][-1]["actors"][0]["references"][0][axis] += 1
    rehash(packet)
    with pytest.raises(ValueError, match="provenance"):
        candidate.build(packet, **kwargs)


def test_cross_round_and_nonincreasing_cut_are_rejected():
    packet, kwargs = fixture()
    packet["context"]["samples"][-1]["cut"]["round_number"] = 2
    rehash(packet)
    with pytest.raises(ValueError, match="crosses round"):
        candidate.build(packet, **kwargs)


def test_partial_inventory_remains_partial_and_exact_counts_cannot_be_fabricated():
    packet, kwargs = fixture()
    item = next(iter(packet["inventory"][-1].values()))
    item.update(
        status="partial",
        complete=False,
        quantity_complete=False,
        reason="quantity_unavailable",
    )
    value = candidate.build(packet, **kwargs)
    assert value["snapshot"]["actors"][0]["inventory"]["status"] == "partial"
    packet["team_status"][-1]["opponents"]["alive"] = 1
    with pytest.raises((ValueError, ValidationError)):
        candidate.build(packet, **kwargs)


def test_human_text_and_predictions_are_not_accepted():
    packet, kwargs = fixture()
    packet["human_reading"] = "A"
    with pytest.raises(ValidationError):
        candidate.build(packet, **kwargs)
    del packet["human_reading"]
    kwargs["kills"]["prediction"] = "B"
    with pytest.raises(ValidationError):
        candidate.build(packet, **kwargs)


def test_load_binds_raw_bytes_and_preserves_sources(tmp_path):
    packet, kwargs = fixture()
    hashes, arguments = {}, {"case_id": "E03", "instrument": kwargs["instrument"]}
    for name, payload in {
        "packet": packet,
        **{k: kwargs[k] for k in ("kills", "extra", "bomb")},
    }.items():
        path = tmp_path / (name + ".json")
        path.write_text(json.dumps(payload), encoding="utf-8")
        hashes[name] = hashlib.sha256(path.read_bytes()).hexdigest()
        arguments[name + "_path"] = path
        arguments[name + "_sha256"] = hashes[name]
    value = candidate.load(**arguments)
    assert candidate.validate(value) == value
    assert set(value["input_files"]) == {"packet", "kills", "extra", "bomb"}
    for name, expected_hash in hashes.items():
        assert (
            hashlib.sha256(arguments[name + "_path"].read_bytes()).hexdigest()
            == expected_hash
        )
    arguments["kills_path"].write_text("{}", encoding="utf-8")
    with pytest.raises(ValueError, match="kills file hash"):
        candidate.load(**arguments)


def test_validation_recomputes_snapshot_and_content_id():
    packet, kwargs = fixture()
    value = candidate.build(packet, **kwargs)
    value["snapshot"]["objective"]["phase"] = "postplant"
    value["candidate_context_id"] = candidate.content_hash(
        {k: v for k, v in value.items() if k != "candidate_context_id"}
    )
    with pytest.raises(ValueError, match="derived facts"):
        candidate.validate(value)


def test_last_known_inventory_is_dated_same_life_and_never_current():
    packet, kwargs = fixture()
    actor = packet["context"]["samples"][-1]["actors"][0]
    alias = actor["actor_alias"]
    item = packet["inventory"][-1][alias]
    previous_cut = packet["context"]["samples"][-2]["cut"]
    item.update(
        status="unavailable",
        complete=False,
        quantity_complete=False,
        reason="missing_inventory",
        last_known={
            "identity": deepcopy(actor["identity"]),
            "source_sequence": previous_cut["sequence"],
            "tick": previous_cut["tick"],
            "items": [],
        },
    )
    value = candidate.build(packet, **kwargs)
    assert value["snapshot"]["actors"][0]["inventory"]["status"] == "unavailable"
    item["last_known"]["tick"] += 100
    with pytest.raises(ValueError, match="future last-known"):
        candidate.build(packet, **kwargs)
    item["last_known"]["tick"] = previous_cut["tick"]
    item["last_known"]["identity"]["pawn_life"] = "different-life"
    with pytest.raises(ValueError, match="identity or continuity"):
        candidate.build(packet, **kwargs)


def test_explicit_ground_state_conflict_stays_unresolved():
    packet, kwargs = fixture()
    kwargs["bomb"]["frames"][-1]["status"] = "state_conflict"
    kwargs["bomb"]["counts"] = {"not_dropped": 4, "state_conflict": 1}
    value = candidate.build(packet, **kwargs)
    assert value["snapshot"]["objective"]["phase"] == "unknown"
    assert value["snapshot"]["objective"]["state"] == "conflicting"
