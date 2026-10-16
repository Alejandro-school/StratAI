import copy
import gzip
import json
from pathlib import Path
from types import SimpleNamespace

import pytest

from ai_coach.support import anchors as s

ALIAS = "demo-" + "a" * 20
ACTOR = "support-actor-" + "1" * 20
ALLY = "support-actor-" + "2" * 20
ENEMY = "support-actor-" + "3" * 20
REF = {"path": s.DATA, "bytes": 1, "sha256": "b" * 64}


def player(alias, life, x, side="CT"):
    return {
        "actor_alias": alias,
        "side": side,
        "connected": True,
        "pawn_handle": 16384 + life,
        "pawn_life": f"support-pawn-{life:06d}",
        "identity_status": "observed",
        "position": {"x": x, "y": 0, "z": 0.0},
        "alive": True,
        "alive_status": "observed",
        "position_status": "observed",
    }


def fixture(n=36):
    frames = []
    for i in range(n):
        # Stable, then increasing separation. One onset per directed pair.
        x = 10 if i < 2 else 10 + i
        frames.append(
            {
                "sequence": i + 1,
                "frame": 100 + i * 4,
                "instance": 1,
                "round_number": 1,
                "tick": 200 + i * 4,
                "epoch": 0,
                "sample_step": 4,
                "tick_rate": 64.0,
                "phase": "playing",
                "players": [
                    player(ACTOR, 1, 0),
                    player(ALLY, 2, x),
                    player(ENEMY, 3, 99, "T"),
                ],
            }
        )
    return {
        "schema_id": "stratai.support_observations@1",
        "contract_version": "1.0.0-candidate.1",
        "source_alias": ALIAS,
        "source_sha256": "a" * 64,
        "build_identifier": "fixture",
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "base_manifest": {
            "path": f"match_{ALIAS}/canonical/manifest.json",
            "bytes": 1,
            "sha256": "c" * 64,
        },
        "capture_contract": "fresh_replay_16hz_FrameDone_with_full_pawn_life@1",
        "observations": {"frames": frames},
    }


def build(data):
    return s.build_report(data, REF, require_observed=True)


def test_onset_without_death_has_exact_causal_history_and_separate_horizon():
    data = fixture()
    out = build(data)
    assert len(out["anchors"]) == 2
    a = out["anchors"][0]
    assert (a["t0_tick"], a["source_sequence"]) == (208, 3)
    assert a["observed"]["distance_squared_world_units"] == [100, 100, 144]
    assert a["later_observation"]["end_tick"] == 336
    assert a["later_observation"]["not_alive_sample_observed"] is False
    assert out["summary"]["physical_entries_accredited"] == 0
    assert a["decision_id"] is None and a["physical_entry_status"] == "not_accredited"
    assert s.initial_features(data, a["anchor_id"], ACTOR, 208, 108) == a["observed"]
    assert (
        "later_observation" not in a["observed"] and "actor_alias" not in a["observed"]
    )


@pytest.mark.parametrize("mutation", ["death", "position", "damage", "winner", "enemy"])
def test_future_and_oracle_mutations_never_change_prior_selection_or_features(mutation):
    original = fixture()
    expected = build(original)["anchors"][0]
    data = copy.deepcopy(original)
    for f in data["observations"]["frames"]:
        if mutation == "enemy":
            f["players"][2]["position"]["x"] = -9999
            f["players"][2]["alive"] = False
        elif f["sequence"] > 3:
            if mutation == "death":
                f["players"][1]["alive"] = False
            elif mutation == "position":
                f["players"][1]["position"]["x"] = 0
            elif mutation == "damage":
                f["damage"] = {"amount": 99}
            elif mutation == "winner":
                f["winner"] = "T"
    # Selector reads a projection, not arbitrary execution/outcome/oracle fields.
    got, _ = s.select_anchors(data["observations"]["frames"], ALIAS)
    got = next(a for a in got if a["anchor_id"] == expected["anchor_id"])
    assert got == {k: v for k, v in expected.items() if k != "later_observation"}
    assert (
        s.initial_features(data, expected["anchor_id"], ACTOR, 208, 108)
        == expected["observed"]
    )


@pytest.mark.parametrize(
    "field,value", [("frame", 107), ("tick", 207), ("actor", ENEMY)]
)
def test_initial_access_requires_exact_actor_tick_and_delivery(field, value):
    data = fixture()
    a = build(data)["anchors"][0]
    query = {"frame": 108, "tick": 208, "actor": ACTOR}
    query[field] = value
    with pytest.raises(ValueError, match="no exact"):
        s.initial_features(
            data, a["anchor_id"], query["actor"], query["tick"], query["frame"]
        )


@pytest.mark.parametrize(
    "change",
    ["missing_sample", "same_tick", "epoch", "instance", "phase", "rate", "step"],
)
def test_incompatible_history_does_not_interpolate_or_backfill(change):
    data = fixture(3)
    f = data["observations"]["frames"][1]
    if change == "missing_sample":
        f["tick"] = 202
    if change == "same_tick":
        f["tick"] = 200
    if change == "epoch":
        f["epoch"] = data["observations"]["frames"][2]["epoch"] = 1
    if change == "instance":
        f["instance"] = data["observations"]["frames"][2]["instance"] = 2
    if change == "phase":
        f["phase"] = "postround"
    if change == "rate":
        f["tick_rate"] = None
    if change == "step":
        f["sample_step"] = 5
    out = s.build_report(data, REF)
    assert not out["anchors"]
    assert out["summary"]["reason_counts"]["incompatible_sample_window"] >= 2


@pytest.mark.parametrize(
    "change",
    [
        "missing",
        "disconnected",
        "dead",
        "side",
        "life",
        "position",
        "identity",
        "duplicate_actor",
        "duplicate_pawn",
    ],
)
def test_unavailable_or_changed_pair_preserves_abstention(change):
    data = fixture(3)
    f = data["observations"]["frames"][1]
    p = f["players"][1]
    if change == "missing":
        f["players"].pop(1)
    if change == "disconnected":
        p["connected"] = False
    if change == "dead":
        p["alive"] = False
    if change == "side":
        p["side"] = "T"
    if change == "life":
        p["pawn_life"] = "support-pawn-000099"
    if change == "position":
        p["position"] = None
        p["position_status"] = "native_fields_absent"
    if change == "identity":
        p.update(
            identity_status="unavailable", pawn_life=None, position=None, alive=None
        )
        p.update(
            position_status="identity_unavailable", alive_status="identity_unavailable"
        )
    if change == "duplicate_actor":
        f["players"].append(copy.deepcopy(p))
    if change == "duplicate_pawn":
        f["players"][2].update(pawn_life=p["pawn_life"], pawn_handle=p["pawn_handle"])
    out = s.build_report(data, REF)
    assert not out["anchors"]
    assert (
        sum(
            out["summary"]["reason_counts"][r]
            for r in ("pair_history_unavailable", "pair_life_or_side_changed")
        )
        >= 2
    )


def test_life_handle_cannot_change_even_when_low_serial_bits_match():
    data = fixture(3)
    data["observations"]["frames"][1]["players"][0]["pawn_handle"] += 1024 << 14
    with pytest.raises(ValueError, match="life handle changed"):
        build(data)


@pytest.mark.parametrize(
    "change",
    [
        "sequence",
        "frame",
        "tick_reset",
        "sentinel",
        "nonfinite",
        "gate",
        "future_field",
    ],
)
def test_invalid_source_is_rejected(change):
    data = fixture(3)
    f = data["observations"]["frames"][1]
    if change == "sequence":
        f["sequence"] = 8
    if change == "frame":
        f["frame"] = 99
    if change == "tick_reset":
        f["tick"] = 5
    if change == "sentinel":
        f["players"][0]["pawn_handle"] = 16383
    if change == "nonfinite":
        f["players"][0]["position"]["z"] = float("nan")
    if change == "gate":
        data["training_allowed"] = True
    if change == "future_field":
        f["winner"] = "CT"
    with pytest.raises(ValueError):
        build(data)


@pytest.mark.parametrize(
    "change,reason",
    [
        ("end_missing", "missing_exact_horizon_sample"),
        ("gap", "horizon_gap_or_boundary"),
        ("phase", "horizon_gap_or_boundary"),
        ("life", "horizon_pair_unavailable_or_changed"),
    ],
)
def test_horizon_boundaries_leave_unknown_instead_of_assuming_survival(change, reason):
    data = fixture(35)
    if change == "end_missing":
        data["observations"]["frames"].pop()
    if change == "gap":
        data["observations"]["frames"][10]["tick"] += 1
    if change == "phase":
        data["observations"]["frames"][10]["phase"] = "postround"
    if change == "life":
        data["observations"]["frames"][10]["players"][1]["pawn_life"] = (
            "support-pawn-000099"
        )
    a = build(data)["anchors"][0]
    assert a["later_observation"]["reason"] == reason
    assert a["later_observation"]["not_alive_sample_observed"] is None


def test_later_not_alive_sample_never_becomes_initial_feature():
    data = fixture()
    data["observations"]["frames"][10]["players"][1]["alive"] = False
    a = build(data)["anchors"][0]
    assert a["later_observation"]["not_alive_sample_observed"] is True
    assert all(
        h["ally_alive"]
        for h in s.initial_features(data, a["anchor_id"], ACTOR, 208, 108)["history"]
    )


def test_deduplication_rearms_only_after_nonincreasing_step():
    data = fixture(6)
    for f, x in zip(data["observations"]["frames"], [10, 10, 12, 13, 12, 13]):
        f["players"][1]["position"]["x"] = x
    out = build(data)
    assert [
        a["source_sequence"] for a in out["anchors"] if a["actor_alias"] == ACTOR
    ] == [3, 6]
    assert len({a["anchor_id"] for a in out["anchors"]}) == 4


def test_current_ally_absence_is_counted():
    data = fixture(3)
    data["observations"]["frames"][-1]["players"].pop(1)
    out = s.build_report(data, REF)
    assert out["summary"]["reason_counts"]["no_live_ally"] > 0 and not out["anchors"]


def test_forged_report_is_recomputed():
    data = fixture()
    out = build(data)
    out["anchors"][0]["observed"]["distance_squared_world_units"][2] += 1
    with pytest.raises(ValueError, match="differs from source"):
        s.validate_report(out, data, REF)


def test_report_and_feature_edits_cannot_mutate_source_snapshots():
    data = fixture()
    original = copy.deepcopy(data)
    out = build(data)
    a = out["anchors"][0]
    a["observed"]["history"][0]["self_position"]["x"] = 100
    assert data == original
    with pytest.raises(ValueError, match="differs from source"):
        s.validate_report(out, data, REF)
    features = s.initial_features(data, a["anchor_id"], ACTOR, 208, 108)
    features["history"][0]["ally_position"]["x"] = -100
    assert data == original


def test_partial_prefix_keeps_anchor_ids_and_features():
    data = fixture()
    long = build(data)
    short = copy.deepcopy(data)
    short["observations"]["frames"] = short["observations"]["frames"][:3]
    early = build(short)
    for a, b in zip(long["anchors"], early["anchors"]):
        assert {k: v for k, v in a.items() if k != "later_observation"} == {
            k: v for k, v in b.items() if k != "later_observation"
        }


def test_materializer_rejects_nonempty_output_before_reading_source(tmp_path):
    (tmp_path / "sentinel").write_text("preserve")
    with pytest.raises(ValueError):
        s.materialize(Path("missing/support_observations.json"), tmp_path)
    assert (tmp_path / "sentinel").read_text() == "preserve"


def test_source_hash_chain_rejects_modified_base(tmp_path, monkeypatch):
    data = fixture(3)
    base = tmp_path / f"match_{ALIAS}/canonical/manifest.json"
    base.parent.mkdir(parents=True)
    base.write_text("{}")
    path = tmp_path / s.DATA
    path.write_text(json.dumps(data))
    monkeypatch.setattr(s, "load_direct_bundle_manifest", lambda *a, **k: object())
    with pytest.raises(ValueError, match="base hash differs"):
        s.load_source(path)
    data["base_manifest"].update(bytes=2, sha256=s.digest(b"{}"))
    path.write_text(json.dumps(data))
    loaded, ref, _ = s.load_source(path)
    assert loaded == data and ref["sha256"] == s.digest(path.read_bytes())


def test_replay_crosscheck_compares_positions_and_aliases_without_outcomes(tmp_path):
    data = fixture(3)
    for f in data["observations"]["frames"]:
        for i, p in enumerate(f["players"], 1):
            p["actor_alias"] = (
                "support-actor-" + s.digest(("a" * 64 + f"\nsteam:{i}").encode())[:20]
            )
    replay = {
        "round": {
            "round": 1,
            "frames": [
                {
                    "tick": f["tick"],
                    "players": [
                        {
                            "steam_id": str(i),
                            "team": p["side"],
                            "alive": p["alive"],
                            **p["position"],
                        }
                        for i, p in enumerate(f["players"], 1)
                    ],
                }
                for f in data["observations"]["frames"]
            ],
        }
    }
    path = tmp_path / "replay.gz"
    with gzip.open(path, "wt") as stream:
        json.dump(replay, stream)
    base = SimpleNamespace(many=lambda _: [SimpleNamespace(physical_path=path)])
    assert s.validate_replay_overlap(data, base)["player_samples_checked"] == 9
    row = data["observations"]["frames"][0]["players"][0]
    row["alive"] = None
    assert s.validate_replay_overlap(data, base)["player_samples_checked"] == 9
    row["alive"] = False
    saved_position = row["position"]
    row["position"] = None
    with pytest.raises(ValueError, match="position/alive mismatch"):
        s.validate_replay_overlap(data, base)
    row["alive"] = True
    row["position"] = saved_position
    data["observations"]["frames"][0]["players"][0]["position"]["x"] = 1
    with pytest.raises(ValueError, match="position/alive mismatch"):
        s.validate_replay_overlap(data, base)


def test_protocol_is_frozen_and_has_no_prescriptive_decision():
    p, ref = s.protocol()
    assert p["history_samples"] == 3 and p["future_window_seconds"] == 2
    assert p["decision_id"] is None and ref["bytes"] > 0
    p["history_samples"] = 4
    assert list(s.schema_validator(s.PROTOCOL).iter_errors(p))
