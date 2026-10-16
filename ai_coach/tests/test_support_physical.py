import copy
import json
from pathlib import Path

import pytest
from jsonschema.exceptions import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import physical as p
from ai_coach.tests.test_support_anchors import ENEMY, REF, fixture


def observed(v):
    return {"status": "observed", "value": v, "reason": None}


def physical(support):
    frames = []
    for f in support["observations"]["frames"]:
        players = []
        for i, base in enumerate(f["players"], 1):
            ctx = {
                "weapon_handle": 16384 + i,
                "weapon_life": f"support-weapon-{i:06d}",
                "weapon_identity_status": "observed",
                "raw_clip": 1,
                "raw_reserve": 0,
                "reserve_source": "m_pReserveAmmo.0000",
                "reload_signal": "weapon_fire",
                "reload_signal_tick": 100,
                "facts": {
                    "view_angles": observed([0, 0]),
                    "walking": observed(False),
                    "ducking": observed(False),
                    "active_weapon": observed("AK-47"),
                    "ammo_in_magazine": observed(0),
                    "ammo_reserve": observed(0),
                    "reloading": observed(False),
                },
            }
            players.append({"actor_alias": base["actor_alias"], "context": ctx})
        frames.append({"source_sequence": f["sequence"], "players": players})
    return {
        "schema_id": "stratai.support_physical_observations@1",
        **{
            k: support[k]
            for k in (
                "contract_version",
                "source_alias",
                "source_sha256",
                "build_identifier",
                "comparison_allowed",
                "training_allowed",
                "product_supported",
            )
        },
        "support_source": REF,
        "capture_contract": "same_support_FrameDone_native_physical@1",
        "observations": {"frames": frames},
    }


ROUNDS = [{"round_number": 1, "start_tick": 100, "end_tick": 1000}]
REFERENCES = {
    k: REF for k in ("physical_source", "support_source", "anchors", "rounds")
}


def build(support=None, data=None, rounds=ROUNDS):
    support = support or fixture()
    data = data or physical(support)
    anchors = a.build_report(support, REF, require_observed=True)
    return p.build_report(support, data, anchors, REFERENCES, rounds)


def test_movement_recomputed_and_zero_false_retained_at_unchanged_anchors():
    support = fixture()
    data = physical(support)
    original = copy.deepcopy(support)
    report = build(support, data)
    assert support == original
    assert report["summary"]["all_components_available"] == 2
    row = report["rows"][0]
    assert row["initial"]["ally"]["movement"]["velocity_world_units_per_second"] == [
        32,
        0,
        0,
    ]
    assert (
        row["initial"]["self"]["movement"]["horizontal_speed_world_units_per_second"]
        == 0
    )
    features = p.initial_features(
        support,
        data,
        row["anchor_id"],
        row["actor_alias"],
        row["t0_tick"],
        row["t0_frame"],
    )
    assert features == row["initial"]
    encoded = json.dumps(features)
    assert (
        "support-actor" not in encoded
        and "weapon_handle" not in encoded
        and "later_observation" not in encoded
    )
    features["self"]["facts"]["view_angles"]["value"][0] = 50
    assert data["observations"]["frames"][2]["players"][0]["context"]["facts"][
        "view_angles"
    ]["value"] == [0, 0]


@pytest.mark.parametrize(
    "change",
    ["future_death", "future_damage", "future_winner", "future_weapon", "enemy"],
)
def test_future_and_enemy_mutations_do_not_change_initial(change):
    support = fixture()
    data = physical(support)
    row = build(support, data)["rows"][0]
    if change == "enemy":
        for f in data["observations"]["frames"]:
            for actor in f["players"]:
                if actor["actor_alias"] == ENEMY:
                    actor["context"]["facts"]["view_angles"] = observed([179, 90])
    else:
        for f in data["observations"]["frames"][3:]:
            f[change] = "changed"
        for f in support["observations"]["frames"][3:]:
            for actor in f["players"]:
                actor["alive"] = False
    assert (
        p.initial_features(
            support,
            data,
            row["anchor_id"],
            row["actor_alias"],
            row["t0_tick"],
            row["t0_frame"],
        )
        == row["initial"]
    )


@pytest.mark.parametrize("missing", ["frame", "actor", "context"])
def test_missing_exact_context_abstains_without_changing_selection(missing):
    support = fixture()
    data = physical(support)
    if missing == "frame":
        data["observations"]["frames"].pop(2)
    elif missing == "actor":
        data["observations"]["frames"][2]["players"].pop(0)
    else:
        data["observations"]["frames"][2]["players"][0]["context"] = None
    report = build(support, data)
    assert (
        len(report["rows"]) == 2 and report["summary"]["all_components_available"] == 0
    )
    assert report["rows"][0]["initial"]["self"]["facts"]["walking"]["value"] is None


def test_non_ammunition_and_uninitialized_reload_are_distinct():
    support = fixture()
    data = physical(support)
    ctx = data["observations"]["frames"][2]["players"][0]["context"]
    ctx.update(
        raw_clip=None,
        raw_reserve=None,
        reserve_source=None,
        reload_signal=None,
        reload_signal_tick=None,
    )
    ctx["facts"]["active_weapon"] = observed("Knife")
    for k in p.FIELDS[4:]:
        ctx["facts"][k] = {
            "status": "not_applicable",
            "value": None,
            "reason": "non_ammunition_weapon",
        }
    assert build(support, data)["summary"]["all_components_available"] == 2
    ctx = data["observations"]["frames"][2]["players"][1]["context"]
    ctx["reload_signal"] = None
    ctx["reload_signal_tick"] = None
    ctx["facts"]["reloading"] = {
        "status": "unavailable",
        "value": None,
        "reason": "reload_signal_unavailable",
    }
    assert build(support, data)["summary"]["all_components_available"] == 0


@pytest.mark.parametrize(
    "bad",
    [
        "source_identity",
        "duplicate_sequence",
        "duplicate_actor",
        "future_sequence",
        "unknown_actor",
        "weapon_life",
        "sentinel",
        "unresolved",
        "raw_clip",
        "raw_reserve",
        "reload_signal",
        "presence",
        "reason",
        "nonfinite",
        "unknown_field",
        "training",
        "partial_applicability",
        "unavailable_active",
        "future_signal",
    ],
)
def test_invalid_physical_source_rejected(bad):
    support = fixture()
    data = physical(support)
    f = data["observations"]["frames"][2]
    ctx = f["players"][0]["context"]
    if bad == "source_identity":
        data["source_sha256"] = "b" * 64
    elif bad == "duplicate_sequence":
        f["source_sequence"] = 2
    elif bad == "duplicate_actor":
        f["players"].append(copy.deepcopy(f["players"][0]))
    elif bad == "future_sequence":
        f["source_sequence"] = 10000
    elif bad == "unknown_actor":
        f["players"][0]["actor_alias"] = "support-actor-" + "f" * 20
    elif bad == "weapon_life":
        ctx["weapon_handle"] += 1024 << 14
    elif bad == "sentinel":
        ctx["weapon_handle"] = 0xFFFFFFFF
    elif bad == "unresolved":
        ctx["weapon_identity_status"] = "unresolved"
    elif bad == "future_signal":
        ctx["reload_signal_tick"] = 100000
    elif bad == "raw_clip":
        ctx["raw_clip"] = 0
    elif bad == "raw_reserve":
        ctx["raw_reserve"] = -1
    elif bad == "reload_signal":
        ctx["reload_signal"] = None
    elif bad == "presence":
        ctx["facts"]["walking"]["value"] = None
    elif bad == "reason":
        ctx["facts"]["walking"]["reason"] = "native_fields_absent"
    elif bad == "nonfinite":
        ctx["facts"]["view_angles"]["value"][0] = float("nan")
    elif bad == "unknown_field":
        ctx["winner"] = "CT"
    elif bad == "training":
        data["training_allowed"] = True
    elif bad == "partial_applicability":
        ctx["facts"]["ammo_reserve"] = {
            "status": "not_applicable",
            "value": None,
            "reason": "non_ammunition_weapon",
        }
    elif bad == "unavailable_active":
        ctx["facts"]["active_weapon"] = {
            "status": "unavailable",
            "value": None,
            "reason": "weapon_identity_unavailable",
        }
    with pytest.raises((ValueError, ValidationError)):
        p.validate_source(data, support)


@pytest.mark.parametrize("bad", ["dead", "disconnected", "missing_position", "side"])
def test_context_requires_available_base_actor(bad):
    support = fixture()
    data = physical(support)
    actor = support["observations"]["frames"][2]["players"][0]
    if bad == "dead":
        actor["alive"] = False
    elif bad == "disconnected":
        actor["connected"] = False
    elif bad == "missing_position":
        actor["position"] = None
    else:
        actor["side"] = "unknown"
    with pytest.raises(ValueError):
        p.validate_source(data, support)


def test_preliminary_round_is_not_promoted():
    report = build(rounds=[])
    assert report["summary"]["canonical_interval_matched"] == 0
    assert report["summary"]["all_components_available"] == 2
    assert all(
        not r["decision_eligible"] and r["competitive_round_status"] == "not_accredited"
        for r in report["rows"]
    )


def test_forged_movement_and_anchor_rejected():
    support = fixture()
    data = physical(support)
    ar = a.build_report(support, REF, require_observed=True)
    report = build(support, data)
    report["rows"][0]["initial"]["self"]["movement"][
        "horizontal_speed_world_units_per_second"
    ] = 1
    with pytest.raises(ValueError, match="recomputation"):
        p.validate_report(report, support, data, ar, REFERENCES, ROUNDS)
    ar["anchors"][0]["t0_tick"] += 1
    with pytest.raises(ValueError, match="anchor mismatch"):
        p.build_report(support, data, ar, REFERENCES, ROUNDS)


def test_materializer_rejects_nonempty_destination_before_reading_source(tmp_path):
    (tmp_path / "sentinel").write_text("keep")
    with pytest.raises(ValueError):
        p.materialize(Path("absent"), tmp_path)


def test_loader_rejects_hash_mismatch(tmp_path):
    support = fixture()
    data = physical(support)
    (tmp_path / a.DATA).write_text(json.dumps(support))
    path = tmp_path / p.DATA
    path.write_text(json.dumps(data))
    with pytest.raises(ValueError, match="hash mismatch"):
        p.load_source(path)
