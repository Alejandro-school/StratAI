import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import preparation as c
from ai_coach.support import preparation_objective as p
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF, fixture
from ai_coach.tests.test_support_objective import objective, scalar


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


def test_population_first_sample_false_and_zero():
    support = fixture()
    data = objective(support)
    data["observations"]["frames"][0]["rules"]["m_iRoundTime"] = scalar(0, "int32")
    first = features(support, data)
    assert first["global"]["round_duration_seconds"]["value"] == 0
    assert first["self"]["objective"]["defusing"]["value"] is False
    rows = list(p.iter_rows(support, data))
    assert [v["candidate_id"] for row in rows for v in row["candidates"]] == [
        v["candidate_id"] for _, pairs in c.iter_census(support) for v in pairs
    ]
    assert len(rows) == len(support["observations"]["frames"])


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
        features(support, objective(support), **{field: value})


@pytest.mark.parametrize("index", [0, 1, 2, 10])
def test_prefix_future_enemy_geometry_and_copy(index):
    support = fixture()
    data = objective(support)
    expected = features(support, data, index)
    support["observations"]["frames"][index + 1 :] = [None, {"garbage": True}]
    data["observations"]["frames"][index + 1 :] = [None, {"source_sequence": "invalid"}]
    data["observations"]["events"].extend([None, {"kind": "bad"}])
    for row in support["observations"]["frames"][: index + 1]:
        row["players"][2]["position"] = {"x": 1e8, "y": -1e8, "z": 123}
    for row in data["observations"]["frames"][: index + 1]:
        row["players"][2]["fields"]["m_bIsDefusing"] = scalar(True)
    assert features(support, data, index) == expected
    for secret in (
        "support-actor",
        "pawn_handle",
        "pawn_life",
        "source_sha256",
        "candidate_id",
        "matrix",
        "execution",
        "provenance",
        "native_player",
        "event_sequence",
        "competitive",
    ):
        assert secret not in json.dumps(expected)
    expected["self"]["objective"]["defusing"]["value"] = True
    expected["global"]["objective_phase"]["value"] = "resolved"
    assert (
        features(support, data, index)["self"]["objective"]["defusing"]["value"]
        is False
    )


@pytest.mark.parametrize(
    "kind", ["bomb_planted", "bomb_defused", "bomb_exploded", "round_end"]
)
def test_later_public_event_same_tick_only_changes_later_delivery(kind):
    support = fixture(2)
    support["observations"]["frames"][1]["tick"] = 200
    data = objective(support)
    expected = features(support, data)
    event = dict(
        data["observations"]["events"][0],
        sequence=2,
        order=15,
        frame=101,
        tick=200,
        kind=kind,
    )
    data["observations"]["events"].append(event)
    data["observations"]["frames"][1]["event_sequence"] = 2
    if kind == "bomb_planted":
        data["observations"]["frames"][1]["rules"]["m_bBombPlanted"] = scalar(True)
    assert features(support, data) == expected
    later = features(support, data, 1)
    if kind == "bomb_planted":
        assert later["global"]["objective_phase"]["value"] == "planted"
        assert later["global"]["bomb_elapsed_seconds"]["value"] == 0
    else:
        assert (
            later["global"]["objective_phase"]["reason"]
            == "public_replay_phase_conflict"
        )
    a.validate_source(support)
    p.o.validate_source(data, support)


def test_same_tick_other_framedone_not_borrowed():
    support = fixture(2)
    support["observations"]["frames"][1]["tick"] = 200
    data = objective(support)
    data["observations"]["frames"].pop()
    result = features(support, data, 1)
    assert all(
        v["reason"] == "missing_exact_objective_sample"
        for v in result["global"].values()
    )
    assert len(list(p.iter_rows(support, data))[-1]["candidates"]) == 2


@pytest.mark.parametrize(
    "change",
    [
        "frame",
        "tick",
        "instance",
        "epoch",
        "life",
        "handle",
        "order",
        "frontier",
        "actor",
        "source_alias",
        "source_sha256",
        "build_identifier",
    ],
)
def test_source_identity_and_delivery_mismatch_rejected(change):
    support = fixture(2)
    data = objective(support)
    row = data["observations"]["frames"][1]
    if change in ("frame", "tick", "instance", "epoch"):
        row[change] += 1
    elif change == "life":
        row["players"][0]["pawn_life"] = "support-pawn-000099"
    elif change == "handle":
        support["observations"]["frames"][1]["players"][0]["pawn_handle"] += 16384
    elif change == "order":
        row["order"] = 10
    elif change == "frontier":
        row["event_sequence"] = 2
    elif change == "actor":
        row["players"][0]["actor_alias"] = "support-actor-" + "d" * 20
    else:
        data[change] = "demo-" + "b" * 20 if change == "source_alias" else "b" * 64
    with pytest.raises((ValueError, ValidationError)):
        features(support, data, 1)


@pytest.mark.parametrize("change", ["life", "side", "epoch", "instance"])
def test_boundary_current_sample_keeps_population_without_carrying_old_state(change):
    support = fixture(2)
    current = support["observations"]["frames"][1]
    if change == "life":
        current["players"][0].update(pawn_life="support-pawn-000099", pawn_handle=32769)
    elif change == "side":
        current["players"][0]["side"] = current["players"][1]["side"] = "T"
        current["players"][2]["side"] = "CT"
    else:
        current[change] += 1
    data = objective(support)
    data["observations"]["frames"][0]["players"][0]["fields"]["m_bIsDefusing"] = scalar(
        True
    )
    result = features(support, data, 1)
    assert result["self"]["objective"]["defusing"]["value"] is False
    if change in ("epoch", "instance"):
        assert (
            result["global"]["objective_phase"]["reason"]
            == "public_phase_origin_absent"
        )
    assert len(list(p.iter_rows(support, data))[-1]["candidates"]) == 2


@pytest.mark.parametrize(
    "kind,reason",
    [
        ("missing", "actor_objective_context_unavailable"),
        ("null_life", "actor_objective_context_unavailable"),
        ("absent", "native_absent"),
        ("unsupported", "native_unsupported_type"),
        ("int_zero", "native_type_or_range_unsupported"),
    ],
)
def test_unavailable_keeps_candidate_and_no_defaults(kind, reason):
    support = fixture(1)
    data = objective(support)
    row = data["observations"]["frames"][0]
    player = row["players"][0]
    if kind == "missing":
        row["players"].pop(0)
    elif kind == "null_life":
        player["pawn_life"] = None
        player["fields"] = {
            k: {"status": "absent", "native_type": None, "value": None}
            for k in p.o.PLAYER.values()
        }
    else:
        player["fields"]["m_bIsDefusing"] = {
            "absent": {"status": "absent", "native_type": None, "value": None},
            "unsupported": {
                "status": "unsupported_type",
                "native_type": "string",
                "value": None,
            },
            "int_zero": scalar(0, "int32"),
        }[kind]
    assert features(support, data)["self"]["objective"]["defusing"] == {
        "status": "unavailable",
        "value": None,
        "reason": reason,
    }
    assert len(next(p.iter_rows(support, data))["candidates"]) == 2


def test_phase_conflict_and_playing_does_not_imply_freeze_end():
    support = fixture(1)
    data = objective(support)
    data["observations"]["events"][0]["kind"] = "round_start"
    result = features(support, data)
    assert (
        result["global"]["round_elapsed_seconds"]["reason"]
        == "public_clock_origin_absent"
    )
    data["observations"]["frames"][0]["rules"]["m_bBombPlanted"] = scalar(True)
    result = features(support, data)
    assert (
        result["global"]["objective_phase"]["reason"] == "native_public_phase_conflict"
    )
    assert (
        result["global"]["bomb_elapsed_seconds"]["reason"]
        == "objective_phase_unavailable"
    )


@pytest.fixture
def materialized(tmp_path, monkeypatch):
    support = fixture(4)
    data = objective(support)
    refs = {
        k: REF.copy()
        for k in ("objective_source", "support_source", "preparation_protocol")
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


def test_reader_roundtrip_and_matrix(materialized):
    root, result = materialized
    assert p.validate_artifact(Path("unused"), root) == result
    assert result["coverage"]["counts"] == {"cuts": 4, "actor_cuts": 8, "candidates": 8}
    for cell in result["coverage"]["matrix"]:
        assert cell["phase_observed"] == 4
        assert cell["both_available"]["defusing"] == 4
        assert cell["fact_status_counts"]["self.defusing"] == {"observed:present": 4}


@pytest.mark.parametrize(
    "mutation",
    [
        "identity",
        "cut",
        "fact",
        "raw",
        "event",
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
            rows[0]["actors"][0]["initial"]["defusing"]["value"] = True
        elif mutation == "raw":
            rows[0]["actors"][0]["native_player"]["fields"]["m_bIsDefusing"][
                "value"
            ] = True
        elif mutation == "event":
            rows[0]["provenance"]["event_sequence"] += 1
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


def test_preflight_precedes_source_and_rejects_overwrite(tmp_path, monkeypatch):
    monkeypatch.setattr(
        p, "load_inputs", lambda source: pytest.fail("source before preflight")
    )
    (tmp_path / "retained").write_text("keep")
    for destination in (tmp_path, Path.cwd()):
        with pytest.raises(ValueError):
            p.materialize(Path("unused"), destination)


def test_excluded_and_empty_cuts_are_retained_without_features():
    support = fixture(3)
    support["observations"]["frames"][0]["phase"] = "postround"
    support["observations"]["frames"][1]["players"] = []
    data = objective(support)
    rows = list(p.iter_rows(support, data))
    for row in rows[:2]:
        assert row["candidates"] == row["actors"] == []
        assert row["global"] is row["provenance"] is None
        a.schema_validator(p.ROW_SCHEMA).validate(row)
