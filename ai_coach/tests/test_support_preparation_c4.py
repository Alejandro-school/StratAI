import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import preparation as c
from ai_coach.support import preparation_c4 as p
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF, fixture
from ai_coach.tests.test_support_c4 import ownership, ref
from ai_coach.tests.test_support_objective import objective, scalar


def inputs(count=4):
    support = fixture(count)
    obj = objective(support)
    return support, obj, ownership(support, obj)


def features(support, obj, data, index=0, **overrides):
    cut = support["observations"]["frames"][index]
    args = {
        "identifier": c.candidate_id(support["source_alias"], cut, ACTOR, ALLY),
        "actor": ACTOR,
        "ally": ALLY,
        "sequence": cut["sequence"],
        "tick": cut["tick"],
        "frame": cut["frame"],
    }
    args.update(overrides)
    return p.initial_features(support, obj, data, **args)


def test_all_pairs_first_cut_positive_negative_and_zero_carrier_position():
    support, obj, data = inputs()
    result = features(support, obj, data)
    assert result["self"]["c4"]["has_c4"]["value"] is True
    assert result["ally"]["c4"]["has_c4"]["value"] is False
    assert result["self"]["c4"]["carrier_player_position"]["value"] == {
        "x": 0,
        "y": 0,
        "z": 0,
    }
    assert result["ally"]["c4"]["carrier_player_position"]["status"] == "not_applicable"
    for role in c.ROLES:
        assert result[role]["c4"]["physical_c4_position"]["value"] is None
    expected = [
        pair["candidate_id"] for _, pairs in c.iter_census(support) for pair in pairs
    ]
    rows = list(p.iter_rows(support, data))
    assert [v["candidate_id"] for row in rows for v in row["candidates"]] == expected


@pytest.mark.parametrize(
    "field,value",
    [
        ("identifier", "wrong"),
        ("actor", ENEMY),
        ("ally", ACTOR),
        ("sequence", 2),
        ("tick", 201),
        ("frame", 101),
        ("sequence", True),
    ],
)
def test_exact_candidate_key(field, value):
    with pytest.raises(ValueError):
        features(*inputs(), **{field: value})


@pytest.mark.parametrize("index", [0, 1, 2, 10])
def test_prefix_malformed_future_enemy_geometry_and_copies(index):
    support, obj, data = inputs(12)
    expected = features(support, obj, data, index)
    support["observations"]["frames"][index + 1 :] = [None, {"garbage": True}]
    obj["observations"]["frames"][index + 1 :] = [None]
    obj["observations"]["events"].append(None)
    data["observations"]["cuts"][index + 1 :] = [None]
    data["observations"]["states"][index + 1 :] = [None, {"sequence": "invalid"}]
    for frame in support["observations"]["frames"][: index + 1]:
        frame["players"][2]["position"] = {"x": 1e8, "y": -1e8, "z": 123}
    assert features(support, obj, data, index) == expected
    for secret in (
        "support-actor",
        "pawn_handle",
        "pawn_life",
        "source_sha256",
        "candidate_id",
        "provenance",
        "native_player",
        "state_sequence",
        "objective_order",
        "matrix",
        "execution",
        "competitive",
        "entities",
    ):
        assert secret not in json.dumps(expected)
    expected["self"]["c4"]["carrier_player_position"]["value"]["x"] = 999
    expected["self"]["c4"]["has_c4"]["value"] = False
    assert features(support, obj, data, index)["self"]["c4"]["has_c4"]["value"] is True
    assert support["observations"]["frames"][index]["players"][0]["position"]["x"] == 0


def test_same_tick_handoff_and_objective_event_belong_to_exact_delivery():
    support = fixture(2)
    support["observations"]["frames"][1]["tick"] = 200
    obj = objective(support)
    data = ownership(support, obj)
    obj["observations"]["events"].append(
        {
            "sequence": 2,
            "order": 15,
            "frame": 101,
            "tick": 200,
            "instance": 1,
            "epoch": 0,
            "kind": "bomb_planted",
        }
    )
    obj["observations"]["frames"][1]["event_sequence"] = 2
    obj["observations"]["frames"][1]["rules"]["m_bBombPlanted"] = scalar(True)
    state = data["observations"]["states"][1]
    own, ally = state["players"][:2]
    entity = state["entities"][0]
    own.update(slots=[], inventory_length=0)
    ally.update(
        slots=[ref(entity["handle"], entity["entity_life"])], inventory_length=1
    )
    entity["owner"] = ref(ally["pawn_handle"], ally["pawn_life"])
    first, second = [features(support, obj, data, i) for i in (0, 1)]
    assert first["self"]["c4"]["has_c4"]["value"] is True
    assert second["self"]["c4"]["has_c4"]["value"] is False
    assert second["ally"]["c4"]["has_c4"]["value"] is True
    assert (
        second["ally"]["c4"]["carrier_player_position"]["value"]
        == second["ally"]["position"]
    )


@pytest.mark.parametrize("empty", [False, True])
def test_missing_cut_never_borrows_previous_state_or_same_tick(empty):
    support = fixture(2)
    support["observations"]["frames"][1]["tick"] = 200
    obj = objective(support)
    data = ownership(support, obj)
    for name in ("cuts", "states"):
        data["observations"][name] = data["observations"][name][: 0 if empty else 1]
    result = features(support, obj, data, 1)
    assert result["self"]["c4"]["has_c4"]["reason"] == "missing_exact_c4_sample"
    assert len(list(p.iter_rows(support, data))[-1]["candidates"]) == 2


@pytest.mark.parametrize(
    "mode",
    [
        "absent",
        "unsupported_type",
        "outside_probe",
        "slot_absent",
        "slot_unresolved",
        "unknown_definition",
        "zero_definition",
        "owner_absent",
        "owner_series",
        "owner_life_absent",
        "unknown_owned_c4",
    ],
)
def test_partial_inventory_or_owner_keeps_uncertainty_and_population(mode):
    support, obj, data = inputs(1)
    state = data["observations"]["states"][0]
    player, entity = state["players"][0], state["entities"][0]
    if mode in ("absent", "unsupported_type", "outside_probe", "unknown_owned_c4"):
        player.update(
            inventory_status=mode if mode != "unknown_owned_c4" else "observed",
            inventory_length=65
            if mode == "outside_probe"
            else 0
            if mode == "unknown_owned_c4"
            else None,
            slots=[],
        )
        if mode == "unknown_owned_c4":
            entity["definition"] = {
                "status": "absent",
                "native_type": None,
                "value": None,
            }
    elif mode == "slot_absent":
        player["slots"][0] = ref()
    elif mode == "slot_unresolved":
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
    result = features(support, obj, data)
    assert result["self"]["c4"]["has_c4"]["status"] == "unavailable"
    assert (
        result["self"]["c4"]["carrier_player_position"]["reason"]
        == "ownership_unavailable"
    )
    assert len(next(p.iter_rows(support, data))["candidates"]) == 2


@pytest.mark.parametrize(
    "mode",
    [
        "other_owner",
        "invalid_owner",
        "wrong_life",
        "excluded_claim",
        "type",
        "duplicate_slot",
        "two_c4",
    ],
)
def test_conflicts_remain_explicit(mode):
    support, obj, data = inputs(1)
    state = data["observations"]["states"][0]
    player, entity = state["players"][0], state["entities"][0]
    if mode == "other_owner":
        entity["owner"] = ref(
            state["players"][1]["pawn_handle"], state["players"][1]["pawn_life"]
        )
    elif mode == "invalid_owner":
        entity["owner"] = ref(4294967295, status="invalid_sentinel")
    elif mode == "wrong_life":
        entity["owner"]["target_life"] = "support-pawn-999999"
    elif mode == "excluded_claim":
        player.update(slots=[], inventory_length=0)
    elif mode == "type":
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
    assert (
        features(support, obj, data)["self"]["c4"]["has_c4"]["reason"]
        == "inventory_owner_or_type_conflict"
    )


def test_positive_evidence_survives_partial_other_slot_without_inventing_negative():
    support, obj, data = inputs(1)
    for player in data["observations"]["states"][0]["players"][:2]:
        player["slots"].append(ref())
        player["inventory_length"] += 1
    result = features(support, obj, data)
    assert result["self"]["c4"]["has_c4"]["value"] is True
    assert result["ally"]["c4"]["has_c4"]["reason"] == "inventory_coverage_incomplete"


@pytest.mark.parametrize("sentinel", [16383, 4294967295])
def test_complete_inventory_sentinels_and_non_c4_confirm_negative(sentinel):
    support, obj, data = inputs(1)
    state = data["observations"]["states"][0]
    state["entities"][0].update(class_name="CAK47", definition=scalar(7, "uint64"))
    state["players"][0]["slots"].append(ref(sentinel, status="invalid_sentinel"))
    state["players"][0]["inventory_length"] += 1
    assert features(support, obj, data)["self"]["c4"]["has_c4"]["value"] is False


@pytest.mark.parametrize("mode", ["life", "handle", "side", "instance", "epoch"])
def test_current_boundaries_use_fresh_state(mode):
    support = fixture(2)
    current = support["observations"]["frames"][1]
    if mode in ("life", "handle"):
        current["players"][0].update(pawn_life="support-pawn-000099", pawn_handle=32769)
    elif mode == "side":
        current["players"][0]["side"] = current["players"][1]["side"] = "T"
        current["players"][2]["side"] = "CT"
    else:
        current[mode] += 1
    obj = objective(support)
    data = ownership(support, obj)
    state = data["observations"]["states"][1]
    state["entities"] = []
    state["players"][0].update(slots=[], inventory_length=0)
    result = features(support, obj, data, 1)
    assert result["self"]["c4"]["has_c4"]["value"] is False
    assert len(list(p.iter_rows(support, data))[-1]["candidates"]) == 2


@pytest.mark.parametrize(
    "mode",
    [
        "clock",
        "order",
        "future_state",
        "first_cut",
        "pawn_life",
        "pawn_handle",
        "slot_life",
        "entity_handle",
        "inventory_length",
        "source_identity",
        "objective_missing",
        "same_tick_wrong_frame",
    ],
)
def test_bad_source_binding_fails_closed(mode):
    support, obj, data = inputs(2)
    cut, state = data["observations"]["cuts"][1], data["observations"]["states"][1]
    if mode == "clock":
        cut["tick"] += 1
    elif mode == "order":
        cut["objective_order"] += 1
    elif mode == "future_state":
        cut["state_sequence"] += 1
    elif mode == "first_cut":
        state["first_cut"] += 1
    elif mode in ("pawn_life", "pawn_handle"):
        state["players"][0][mode] = (
            "support-pawn-999999" if mode == "pawn_life" else 32769
        )
    elif mode == "slot_life":
        state["players"][0]["slots"][0]["target_life"] = "support-item-999999"
    elif mode == "entity_handle":
        state["entities"][0]["handle"] += 1
    elif mode == "inventory_length":
        state["players"][0]["inventory_length"] += 1
    elif mode == "source_identity":
        data["build_identifier"] = "wrong"
    elif mode == "objective_missing":
        obj["observations"]["frames"].pop()
    else:
        cut["frame"] -= 1
    with pytest.raises((ValueError, ValidationError)):
        features(support, obj, data, 1)


@pytest.mark.parametrize("boundary", ["instance", "epoch"])
def test_shared_state_cannot_cross_boundary(boundary):
    support, obj, data = inputs(2)
    support["observations"]["frames"][1][boundary] += 1
    obj["observations"]["frames"][1][boundary] += 1
    data["observations"]["cuts"][1][boundary] += 1
    data["observations"]["cuts"][1]["state_sequence"] = 1
    data["observations"]["states"].pop()
    with pytest.raises(ValueError, match="boundary"):
        features(support, obj, data, 1)


def test_shared_state_is_legal_but_current_carrier_position_is_fresh():
    support, obj, data = inputs(2)
    support["observations"]["frames"][1]["players"][0]["position"]["x"] = 4
    data["observations"]["cuts"][1]["state_sequence"] = 1
    data["observations"]["states"].pop()
    assert (
        features(support, obj, data, 1)["self"]["c4"]["carrier_player_position"][
            "value"
        ]["x"]
        == 4
    )


@pytest.fixture
def materialized(tmp_path, monkeypatch):
    support, _, data = inputs()
    refs = {
        key: REF.copy()
        for key in (
            "c4_source",
            "objective_source",
            "support_source",
            "preparation_protocol",
        )
    }
    monkeypatch.setattr(
        p,
        "load_inputs",
        lambda source: (
            copy.deepcopy(data),
            copy.deepcopy(support),
            copy.deepcopy(refs),
        ),
    )
    return tmp_path, p.materialize(Path("unused"), tmp_path, require_observed=True)


def test_reader_roundtrip_and_full_matrix(materialized):
    root, result = materialized
    assert p.validate_artifact(Path("unused"), root) == result
    assert result["coverage"]["counts"] == {"cuts": 4, "actor_cuts": 8, "candidates": 8}
    for cell in result["coverage"]["matrix"]:
        assert cell["both_available"] == {
            "has_c4": 4,
            "carrier_player_position": 4,
            "physical_c4_position": 0,
        }
        assert sum(cell["possession_counts"]["self"].values()) == 4


@pytest.mark.parametrize(
    "mutation",
    [
        "identity",
        "cut",
        "fact",
        "position",
        "raw",
        "provenance",
        "candidate",
        "missing",
        "extra",
        "reorder",
        "coverage",
        "hash",
        "unknown",
    ],
)
def test_reader_rejects_corruption(materialized, mutation):
    root, _ = materialized
    if mutation in ("coverage", "hash"):
        target = root / p.MANIFEST
        data = json.loads(target.read_bytes())
        if mutation == "coverage":
            data["coverage"]["counts"]["candidates"] += 1
        else:
            data["artifact"]["sha256"] = "0" * 64
        target.write_bytes(c.encoded_row(data))
    else:
        target = root / p.ROWS
        rows = [json.loads(line) for line in target.read_bytes().splitlines()]
        if mutation == "identity":
            rows[0]["actors"][0]["pawn_handle"] += 1
        elif mutation == "cut":
            rows[0]["cut"]["frame"] += 1
        elif mutation == "fact":
            rows[0]["actors"][0]["initial"]["has_c4"]["value"] = False
        elif mutation == "position":
            rows[0]["actors"][0]["initial"]["carrier_player_position"]["value"][
                "x"
            ] += 1
        elif mutation == "raw":
            rows[0]["actors"][0]["native_player"]["slots"][0]["handle"] += 1
        elif mutation == "provenance":
            rows[0]["provenance"]["objective_order"] += 1
        elif mutation == "candidate":
            rows[0]["candidates"].pop()
        elif mutation == "missing":
            rows.pop()
        elif mutation == "extra":
            rows.append(rows[0])
        elif mutation == "reorder":
            rows.reverse()
        else:
            rows[0]["enemy"] = True
        target.write_bytes(b"".join(c.encoded_row(row) for row in rows))
    with pytest.raises((ValueError, ValidationError)):
        p.validate_artifact(Path("unused"), root)


def test_preflight_before_source_and_no_overwrite(tmp_path, monkeypatch):
    monkeypatch.setattr(
        p, "load_inputs", lambda source: pytest.fail("source read before preflight")
    )
    (tmp_path / "retained").write_text("keep")
    for path in (tmp_path, Path.cwd()):
        with pytest.raises(ValueError):
            p.materialize(Path("unused"), path)


def test_excluded_cut_is_preserved_without_features():
    support, _, data = inputs(1)
    support["observations"]["frames"][0]["phase"] = "postround"
    row = next(p.iter_rows(support, data))
    assert row["actors"] == row["candidates"] == [] and row["provenance"] is None
    a.schema_validator(p.ROW_SCHEMA).validate(row)
