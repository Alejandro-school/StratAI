import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import preparation as c
from ai_coach.support import preparation_physical as p
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF, fixture
from ai_coach.tests.test_support_physical import physical


def features(support, data, index=0, **overrides):
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
    return p.initial_features(support, data, **args)


def test_first_sample_is_selected_without_history_and_all_pairs_are_preserved():
    support = fixture()
    data = physical(support)
    rows = list(p.iter_rows(support, data))
    expected = [pair for _, pairs in c.iter_census(support) for pair in pairs]
    assert [pair["candidate_id"] for row in rows for pair in row["candidates"]] == [
        pair["candidate_id"] for pair in expected
    ]
    first = features(support, data)
    assert first["self"]["interval_average_velocity"] == {
        "status": "unavailable",
        "value": None,
        "reason": "no_previous_sample",
    }
    assert first["self"]["facts"]["walking"]["value"] is False
    assert first["self"]["facts"]["ammo_in_magazine"]["value"] == 0
    assert len(expected) == 72


def test_interval_is_recomputed_from_two_positions_not_an_onset_or_instantaneous_speed():
    support = fixture()
    result = features(support, physical(support), 2)
    assert result["ally"]["interval_average_velocity"]["value"][
        "velocity_world_units_per_second"
    ] == [32, 0, 0]
    assert (
        result["self"]["interval_average_velocity"]["value"][
            "horizontal_speed_world_units_per_second"
        ]
        == 0
    )
    assert result["ally"]["interval_average_velocity"]["value"]["seconds"] == 0.0625


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
def test_exact_key_required(field, value):
    support = fixture()
    with pytest.raises(ValueError):
        features(support, physical(support), **{field: value})


@pytest.mark.parametrize("index", [0, 1, 2, 10])
def test_prefix_future_malformed_and_enemy_geometry_are_isolated(index):
    support = fixture()
    data = physical(support)
    expected = features(support, data, index)
    support["observations"]["frames"][index + 1 :] = [None, {"garbage": True}]
    data["observations"]["frames"][index + 1 :] = [None, {"source_sequence": "invalid"}]
    for frame in support["observations"]["frames"][: index + 1]:
        frame["players"][2]["position"] = {"x": 1e8, "y": -1e8, "z": 123}
    for frame in data["observations"]["frames"][: index + 1]:
        frame["players"][2]["context"]["facts"]["view_angles"]["value"] = [170, 90]
    assert features(support, data, index) == expected
    encoded = json.dumps(expected)
    for secret in (
        "support-actor",
        "pawn_handle",
        "pawn_life",
        "weapon_handle",
        "weapon_life",
        "source_sha256",
        "candidate_id",
        "matrix",
        "execution",
        "native_context",
    ):
        assert secret not in encoded
    expected["self"]["facts"]["view_angles"]["value"][0] = 50
    assert (
        features(support, data, index)["self"]["facts"]["view_angles"]["value"][0] == 0
    )


@pytest.mark.parametrize(
    "change,reason",
    [
        ("life", "pawn_life_handle_or_side_changed"),
        ("handle", "pawn_life_handle_or_side_changed"),
        ("side", "pawn_life_handle_or_side_changed"),
        ("epoch", "previous_cut_incompatible"),
        ("instance", "previous_cut_incompatible"),
        ("phase", "previous_cut_incompatible"),
        ("rate", "previous_cut_incompatible"),
        ("gap", "previous_cut_incompatible"),
        ("same_tick", "previous_cut_incompatible"),
        ("dead", "previous_actor_unavailable"),
        ("disconnected", "previous_actor_unavailable"),
        ("absent", "previous_actor_unavailable"),
    ],
)
def test_previous_boundary_never_removes_current_candidate(change, reason):
    support = fixture(2)
    old, now = support["observations"]["frames"]
    if change == "life":
        old["players"][0]["pawn_life"] = "support-pawn-000004"
    elif change == "handle":
        old["players"][0].update(pawn_handle=32769, pawn_life="support-pawn-000004")
    elif change == "side":
        old["players"][0]["side"] = "T"
    elif change in ("epoch", "instance"):
        now[change] += 1
    elif change == "phase":
        old["phase"] = "postround"
    elif change == "rate":
        old.update(tick_rate=128, sample_step=8)
    elif change == "gap":
        now["tick"] += 4
    elif change == "same_tick":
        now["tick"] = old["tick"]
    elif change == "dead":
        old["players"][0]["alive"] = False
    elif change == "disconnected":
        old["players"][0]["connected"] = False
    elif change == "absent":
        old["players"].pop(0)
    data = physical(support)
    # Unusable historical actors cannot carry physical context.
    if change in ("dead", "disconnected"):
        data["observations"]["frames"][0]["players"][0]["context"] = None
    result = features(support, data, 1)
    assert result["self"]["interval_average_velocity"]["reason"] == reason
    assert result["self"]["facts"]["active_weapon"]["value"] == "AK-47"
    assert len(list(p.iter_rows(support, data))[-1]["candidates"]) == 2


def test_same_tick_other_framedone_does_not_borrow_physical_row():
    support = fixture(2)
    support["observations"]["frames"][1]["tick"] = 200
    data = physical(support)
    data["observations"]["frames"].pop()
    result = features(support, data, 1)
    assert result["self"]["facts"]["walking"] == {
        "status": "unavailable",
        "value": None,
        "reason": "missing_exact_physical_sample",
    }
    assert features(support, data)["self"]["facts"]["walking"]["value"] is False


@pytest.mark.parametrize(
    "kind",
    [
        "missing_row",
        "missing_actor",
        "null_context",
        "missing_signal",
        "knife",
        "c4",
        "grenade",
        "unknown",
    ],
)
def test_missing_and_not_applicable_remain_distinct_without_changing_population(kind):
    support = fixture(1)
    data = physical(support)
    ctx = data["observations"]["frames"][0]["players"][0]["context"]
    if kind == "missing_row":
        data["observations"]["frames"] = []
    elif kind == "missing_actor":
        data["observations"]["frames"][0]["players"].pop(0)
    elif kind == "null_context":
        data["observations"]["frames"][0]["players"][0]["context"] = None
    elif kind == "missing_signal":
        ctx["facts"]["reloading"] = {
            "status": "unavailable",
            "value": None,
            "reason": "reload_signal_unavailable",
        }
        ctx.update(reload_signal=None, reload_signal_tick=None)
    else:
        ctx["facts"]["active_weapon"]["value"] = {
            "knife": "Knife",
            "c4": "C4",
            "grenade": "Flashbang",
            "unknown": "Unknown",
        }[kind]
        for field in p.p.FIELDS[4:]:
            ctx["facts"][field] = {
                "status": "unavailable" if kind == "unknown" else "not_applicable",
                "value": None,
                "reason": "unknown_weapon_class"
                if kind == "unknown"
                else "non_ammunition_weapon",
            }
        ctx.update(
            raw_clip=None,
            raw_reserve=None,
            reserve_source=None,
            reload_signal=None,
            reload_signal_tick=None,
        )
    result = features(support, data)
    assert result["self"]["facts"]["reloading"]["value"] is None
    assert result["self"]["facts"]["reloading"]["status"] == (
        "not_applicable" if kind in ("knife", "c4", "grenade") else "unavailable"
    )
    assert len(next(iter(p.iter_rows(support, data)))["candidates"]) == 2


@pytest.fixture
def materialized(tmp_path, monkeypatch):
    support = fixture(4)
    data = physical(support)
    refs = {
        k: REF.copy()
        for k in ("physical_source", "support_source", "preparation_protocol")
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


def test_roundtrip_reconciles_every_cut_pair_and_component(materialized):
    root, result = materialized
    assert p.validate_artifact(Path("unused"), root) == result
    assert result["coverage"]["counts"] == {"cuts": 4, "actor_cuts": 8, "candidates": 8}
    for cell in result["coverage"]["matrix"]:
        assert cell["candidates"] == 4
        assert cell["both_available"]["interval_average_velocity"] == 3
        assert cell["self"]["walking"] == {"observed:present": 4}


@pytest.mark.parametrize(
    "mutation",
    [
        "identity",
        "cut",
        "fact",
        "raw",
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
        manifest = json.loads(target.read_bytes())
        if mutation == "coverage":
            manifest["coverage"]["counts"]["candidates"] += 1
        else:
            manifest["artifact"]["sha256"] = "0" * 64
        target.write_bytes(c.encoded_row(manifest))
    else:
        target = root / p.ROWS
        rows = [json.loads(line) for line in target.read_bytes().splitlines()]
        if mutation == "identity":
            rows[0]["actors"][0]["pawn_handle"] += 1
        elif mutation == "cut":
            rows[0]["cut"]["frame"] += 1
        elif mutation == "fact":
            rows[0]["actors"][0]["initial"]["facts"]["walking"]["value"] = True
        elif mutation == "raw":
            rows[0]["actors"][0]["native_context"]["raw_clip"] += 1
        elif mutation == "candidate":
            rows[0]["candidates"].pop()
        elif mutation == "missing":
            rows.pop()
        elif mutation == "extra":
            rows.append(rows[0])
        elif mutation == "reorder":
            rows.reverse()
        elif mutation == "unknown":
            rows[0]["enemy"] = True
        target.write_bytes(b"".join(c.encoded_row(row) for row in rows))
    with pytest.raises((ValueError, ValidationError)):
        p.validate_artifact(Path("unused"), root)


def test_output_preflight_precedes_source_and_rejects_overwrite(tmp_path, monkeypatch):
    monkeypatch.setattr(
        p, "load_inputs", lambda source: pytest.fail("source read before preflight")
    )
    (tmp_path / "retained").write_text("keep")
    for destination in (tmp_path, Path.cwd()):
        with pytest.raises(ValueError):
            p.materialize(Path("unused"), destination)
    assert (tmp_path / "retained").read_text() == "keep"


@pytest.mark.parametrize(
    "field,value", [("source_sha256", "b" * 64), ("build_identifier", "different")]
)
def test_physical_source_identity_must_match(field, value):
    support = fixture(1)
    data = physical(support)
    data[field] = value
    with pytest.raises(ValueError, match="identity"):
        features(support, data)


def test_future_reload_signal_is_rejected_at_current_cut():
    support = fixture(1)
    data = physical(support)
    data["observations"]["frames"][0]["players"][0]["context"]["reload_signal_tick"] = (
        201
    )
    with pytest.raises(ValueError, match="future"):
        features(support, data)
