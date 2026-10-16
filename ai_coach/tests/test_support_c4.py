import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import c4
from ai_coach.support import objective as o
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF, fixture
from ai_coach.tests.test_support_objective import build as objective_report
from ai_coach.tests.test_support_objective import objective, scalar


def ref(handle=None, life=None, status=None):
    return {
        "status": status or ("resolved" if life else "absent"),
        "handle": handle,
        "target_life": life,
    }


def ownership(support, obj):
    states, cuts = [], []
    for f, of in zip(
        support["observations"]["frames"], obj["observations"]["frames"], strict=True
    ):
        seq = f["sequence"]
        actor = a.actor_index(f)[ACTOR]
        entity = {
            "entity_life": "support-item-000001",
            "handle": 32778,
            "class_name": "CC4",
            "definition": scalar(49, "uint64"),
            "owner": ref(actor["pawn_handle"], actor["pawn_life"]),
        }
        states.append(
            {
                "sequence": seq,
                "first_cut": seq,
                "players": [
                    {
                        "actor_alias": p["actor_alias"],
                        "pawn_life": p["pawn_life"],
                        "pawn_handle": p["pawn_handle"],
                        "inventory_status": "observed",
                        "inventory_length": int(p["actor_alias"] == ACTOR),
                        "slots": [ref(32778, entity["entity_life"])]
                        if p["actor_alias"] == ACTOR
                        else [],
                    }
                    for p in f["players"]
                ],
                "entities": [entity],
            }
        )
        cuts.append(
            {
                "source_sequence": seq,
                **{k: f[k] for k in ("frame", "tick", "instance", "epoch")},
                "objective_order": of["order"],
                "state_sequence": seq,
            }
        )
    return {
        "schema_id": "stratai.support_c4_observations@1",
        **{
            k: support[k]
            for k in (
                "source_alias",
                "source_sha256",
                "build_identifier",
                "contract_version",
                "comparison_allowed",
                "training_allowed",
                "product_supported",
            )
        },
        "support_source": REF,
        "objective_source": REF,
        "capture_contract": "same_support_FrameDone_native_inventory_owner@1",
        "observations": {"cuts": cuts, "states": states},
    }


def inputs():
    support = fixture()
    obj = objective(support)
    return support, obj, ownership(support, obj)


def features(support, obj, data):
    anchor = a.select_anchors(
        support["observations"]["frames"], support["source_alias"]
    )[0][0]
    return c4.initial_features(
        support,
        obj,
        data,
        anchor["anchor_id"],
        ACTOR,
        anchor["t0_tick"],
        anchor["t0_frame"],
    )


def build(support, obj, data):
    refs = {
        k: REF
        for k in (
            "support_source",
            "objective_source",
            "c4_source",
            "anchors",
            "physical_context",
            "navigation_context",
            "objective_context",
        )
    }
    return c4.build_report(
        support,
        obj,
        data,
        a.build_report(support, REF),
        objective_report(support, obj),
        refs,
    )


def test_c4_positive_negative_zero_position_and_matrix_denominators():
    support, obj, data = inputs()
    snapshot = copy.deepcopy(data)
    report = build(support, obj, data)
    initial = features(support, obj, data)
    assert initial == report["rows"][0]["initial"]
    assert initial["self"]["has_c4"] == o.fact(True)
    assert initial["ally"]["has_c4"] == o.fact(False)
    assert initial["self"]["carrier_player_position"] == o.fact(
        {"x": 0, "y": 0, "z": 0}
    )
    assert initial["ally"]["carrier_player_position"]["status"] == "not_applicable"
    assert initial["self"]["physical_c4_position"]["value"] is None
    assert report["summary"]["anchors"] == 2
    assert all(
        sum(v.values()) == 2 for v in report["summary"]["fact_status_counts"].values()
    )
    assert all(
        sum(v.values()) == 2 for v in report["summary"]["possession_counts"].values()
    )
    assert (
        not report["training_allowed"] and report["summary"]["eligible_decisions"] == 0
    )
    assert data == snapshot
    initial["self"]["carrier_player_position"]["value"]["x"] = 999
    assert (
        features(support, obj, data)["self"]["carrier_player_position"]["value"]["x"]
        == 0
    )
    assert not any(
        x in json.dumps(initial)
        for x in (
            "actor_alias",
            "pawn_life",
            "support-item",
            "support-actor",
            "winner",
            "later_observation",
        )
    )


@pytest.mark.parametrize(
    "mode",
    [
        "absent",
        "unsupported_type",
        "outside_probe",
        "partial_slot",
        "unresolved_slot",
        "unknown_definition",
        "zero_definition",
        "owner_absent",
        "owner_series",
        "owner_life_absent",
        "owner_only",
        "unknown_owned_c4",
    ],
)
def test_insufficient_inventory_or_uncorroborated_owner_never_becomes_absence(mode):
    support, obj, data = inputs()
    state = data["observations"]["states"][2]
    player, entity = state["players"][0], state["entities"][0]
    if mode in (
        "absent",
        "unsupported_type",
        "outside_probe",
        "owner_only",
        "unknown_owned_c4",
    ):
        player.update(inventory_status="absent", inventory_length=None, slots=[])
        if mode == "unsupported_type":
            player["inventory_status"] = mode
        if mode == "outside_probe":
            player.update(inventory_status=mode, inventory_length=65)
        if mode == "unknown_owned_c4":
            player.update(inventory_status="observed", inventory_length=0)
            entity["definition"] = {
                "status": "absent",
                "native_type": None,
                "value": None,
            }
    elif mode == "partial_slot":
        player["slots"][0] = ref()
    elif mode == "unresolved_slot":
        player["slots"][0] = ref(32778, status="entity_unavailable")
    elif mode in ("unknown_definition", "zero_definition"):
        entity["definition"] = (
            {"status": "absent", "native_type": None, "value": None}
            if mode == "unknown_definition"
            else scalar(0, "uint64")
        )
    elif mode == "owner_absent":
        entity["owner"] = ref()
    elif mode == "owner_series":
        entity["owner"] = ref(99999, status="serial_mismatch")
    else:
        entity["owner"] = ref(
            entity["owner"]["handle"], status="kind_or_life_unavailable"
        )
    result = features(support, obj, data)["self"]["has_c4"]
    assert result["status"] == "unavailable" and result["value"] is None


@pytest.mark.parametrize(
    "mode",
    [
        "other_owner",
        "invalid_owner",
        "owner_wrong_life",
        "inventory_excludes_owner_claim",
        "type_disagreement",
        "duplicate_slot",
        "two_c4",
    ],
)
def test_contradictions_are_preserved(mode):
    support, obj, data = inputs()
    state = data["observations"]["states"][2]
    player, entity = state["players"][0], state["entities"][0]
    if mode == "other_owner":
        ally = state["players"][1]
        entity["owner"] = ref(ally["pawn_handle"], ally["pawn_life"])
    elif mode == "invalid_owner":
        entity["owner"] = ref(4294967295, status="invalid_sentinel")
    elif mode == "owner_wrong_life":
        entity["owner"]["target_life"] = "support-pawn-999999"
    elif mode == "inventory_excludes_owner_claim":
        player.update(slots=[], inventory_length=0)
    elif mode == "type_disagreement":
        entity["definition"] = scalar(7, "uint64")
    elif mode == "duplicate_slot":
        player["slots"].append(copy.deepcopy(player["slots"][0]))
        player["inventory_length"] += 1
    else:
        second = {
            **copy.deepcopy(entity),
            "entity_life": "support-item-000002",
            "handle": 32779,
        }
        state["entities"].append(second)
        player["slots"].append(ref(32779, second["entity_life"]))
        player["inventory_length"] += 1
    assert features(support, obj, data)["self"]["has_c4"] == o.fact(
        reason="inventory_owner_or_type_conflict"
    )


def test_positive_proof_survives_partial_other_slot_but_partial_negative_does_not():
    support, obj, data = inputs()
    state = data["observations"]["states"][2]
    for player in state["players"][:2]:
        player["slots"].append(ref())
        player["inventory_length"] += 1
    initial = features(support, obj, data)
    assert initial["self"]["has_c4"]["value"] is True
    assert initial["ally"]["has_c4"] == o.fact(reason="inventory_coverage_incomplete")


@pytest.mark.parametrize("sentinel", [16383, 4294967295])
def test_complete_sentinel_slots_and_resolved_non_c4_confirm_absence(sentinel):
    support, obj, data = inputs()
    for state in data["observations"]["states"]:
        entity = state["entities"][0]
        entity.update(class_name="CAK47", definition=scalar(7, "uint64"))
        state["players"][0]["slots"].append(ref(sentinel, status="invalid_sentinel"))
        state["players"][0]["inventory_length"] += 1
    assert features(support, obj, data)["self"]["has_c4"] == o.fact(False)


@pytest.mark.parametrize(
    "mode",
    [
        "clock",
        "order",
        "future_state",
        "sequence",
        "state_boundary",
        "inventory_length",
        "slot_life",
        "pawn_life",
        "duplicate_actor",
        "changed_object_handle",
        "scalar_presence",
        "reference_presence",
        "extra_field",
        "training",
    ],
)
def test_malformed_sources_fail_closed(mode):
    support, obj, data = inputs()
    cut = data["observations"]["cuts"][2]
    state = data["observations"]["states"][2]
    if mode == "clock":
        cut["tick"] += 1
    elif mode == "order":
        cut["objective_order"] += 1
    elif mode == "future_state":
        cut["state_sequence"] += 1
    elif mode == "sequence":
        cut["source_sequence"] = 1
    elif mode == "state_boundary":
        data["observations"]["states"] = data["observations"]["states"][:1]
        for i, c in enumerate(data["observations"]["cuts"]):
            c["state_sequence"] = 1
            if i >= 2:
                c["instance"] = support["observations"]["frames"][i]["instance"] = 2
    elif mode == "inventory_length":
        state["players"][0]["inventory_length"] = 2
    elif mode == "slot_life":
        state["players"][0]["slots"][0]["target_life"] = "support-item-999999"
    elif mode == "pawn_life":
        state["players"][0]["pawn_life"] = "support-pawn-999999"
    elif mode == "duplicate_actor":
        state["players"].append(copy.deepcopy(state["players"][0]))
    elif mode == "changed_object_handle":
        state["entities"][0]["handle"] += 1
    elif mode == "scalar_presence":
        state["entities"][0]["definition"]["status"] = "absent"
    elif mode == "reference_presence":
        state["entities"][0]["owner"]["target_life"] = None
    elif mode == "extra_field":
        state["bomb_position"] = [1, 2, 3]
    else:
        data["training_allowed"] = True
    with pytest.raises((ValueError, ValidationError)):
        c4.validate_source(data, support, obj)


@pytest.mark.parametrize("after", [False, True])
def test_same_tick_handoff_obeys_exact_frame_cut(after):
    support, obj, data = inputs()
    # Current anchor is sequence 3. A later frame can have the same demo tick.
    support["observations"]["frames"][3]["tick"] = 208
    obj["observations"]["frames"][3]["tick"] = 208
    data["observations"]["cuts"][3]["tick"] = 208
    for state in data["observations"]["states"][3 if after else 2 :]:
        own, ally = state["players"][:2]
        entity = state["entities"][0]
        own.update(slots=[], inventory_length=0)
        ally.update(
            slots=[ref(entity["handle"], entity["entity_life"])], inventory_length=1
        )
        entity["owner"] = ref(ally["pawn_handle"], ally["pawn_life"])
    result = features(support, obj, data)
    assert result["self"]["has_c4"]["value"] is after
    assert result["ally"]["has_c4"]["value"] is not after


def test_future_and_unrelated_enemy_state_cannot_contaminate_features():
    support, obj, data = inputs()
    expected = features(support, obj, data)
    for state in data["observations"]["states"][:3]:
        enemy = next(p for p in state["players"] if p["actor_alias"] == ENEMY)
        hidden = {
            "entity_life": "support-item-000099",
            "handle": 100,
            "class_name": "CC4",
            "definition": scalar(49, "uint64"),
            "owner": ref(enemy["pawn_handle"], enemy["pawn_life"]),
        }
        state["entities"].append(hidden)
        enemy.update(slots=[ref(100, hidden["entity_life"])], inventory_length=1)
    for f in support["observations"]["frames"][3:]:
        for p in f["players"]:
            p["position"] = {"x": 99999, "y": 99999, "z": 99999}
    # Even malformed future payloads are outside this access boundary.
    data["observations"]["states"][3:] = [{"oracle": "future winner"}]
    obj["observations"]["events"].append({"oracle": "future enemy"})
    assert features(support, obj, data) == expected


def test_missing_exact_cut_abstains_without_reusing_previous_state():
    support, obj, data = inputs()
    data["observations"]["cuts"] = data["observations"]["cuts"][:2]
    data["observations"]["states"] = data["observations"]["states"][:2]
    assert features(support, obj, data)["self"]["has_c4"] == o.fact(
        reason="missing_exact_c4_sample"
    )


@pytest.mark.parametrize("mode", ["nonempty", "nontemporary"])
def test_materializer_rejects_destination_before_reading_source(
    tmp_path, monkeypatch, mode
):
    from ai_coach.support import c4 as module

    called = []
    monkeypatch.setattr(module, "load_source", lambda *args: called.append(args))
    destination = tmp_path if mode == "nonempty" else Path.cwd()
    if mode == "nonempty":
        (destination / "sentinel").write_text("keep")
    with pytest.raises(ValueError):
        module.materialize(
            Path("missing"), destination, Path("missing"), Path("missing")
        )
    assert called == []


@pytest.mark.parametrize(
    "mode", ["dead", "disconnected", "new_pawn", "new_object", "new_epoch"]
)
def test_fresh_boundaries_do_not_inherit_possession(mode):
    support, _obj, data = inputs()
    base = support["observations"]["frames"][2]
    state = data["observations"]["states"][2]
    player = state["players"][0]
    if mode in ("dead", "disconnected"):
        player.update(
            pawn_life=None,
            pawn_handle=None,
            inventory_status="pawn_unavailable",
            inventory_length=None,
            slots=[],
        )
        assert c4.possession(player, state["entities"]) == o.fact(
            reason="pawn_unavailable"
        )
        return
    if mode == "new_pawn":
        # Direct factual evaluation is independent of anchor reselection.
        player["pawn_life"] = "support-pawn-000100"
        assert c4.possession(player, state["entities"])["value"] is None
    elif mode == "new_object":
        state["entities"][0]["entity_life"] = "support-item-000100"
        player["slots"][0]["target_life"] = "support-item-000100"
        state["entities"][0]["owner"] = ref()
        assert c4.possession(player, state["entities"])["value"] is None
    else:
        base["epoch"] = 1
        assert (
            c4.initial_context({"actor_alias": ACTOR, "ally_alias": ALLY}, base, None)[
                "self"
            ]["has_c4"]["value"]
            is None
        )
