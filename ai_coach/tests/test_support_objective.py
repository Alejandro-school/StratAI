import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import objective as o
from ai_coach.tests.test_support_anchors import ACTOR, ENEMY, REF, fixture
from ai_coach.tests.test_support_physical import build as physical_report


def scalar(value=False, typ="bool"):
    return {"status": "observed", "value": value, "native_type": typ}


def objective(support):
    event = {
        "sequence": 1,
        "order": 1,
        "frame": 99,
        "tick": 196,
        "instance": 1,
        "epoch": 0,
        "kind": "freeze_end",
    }
    frames = [
        {
            "source_sequence": f["sequence"],
            **{k: f[k] for k in ("frame", "tick", "instance", "epoch")},
            "order": 10 + i * 10,
            "event_sequence": 1,
            "rules": {
                k: scalar(115, "int32") if k == "m_iRoundTime" else scalar()
                for k in o.RULES.values()
            },
            "players": [
                {
                    "actor_alias": p["actor_alias"],
                    "pawn_life": p["pawn_life"],
                    "fields": {k: scalar() for k in o.PLAYER.values()},
                }
                for p in f["players"]
            ],
        }
        for i, f in enumerate(support["observations"]["frames"])
    ]
    return {
        "schema_id": "stratai.support_objective_observations@1",
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
        "capture_contract": "same_support_FrameDone_public_objective_prefix@1",
        "observations": {"frames": frames, "events": [event]},
    }


REFS = {
    k: REF
    for k in (
        "objective_source",
        "support_source",
        "anchors",
        "physical_context",
        "navigation_context",
    )
}


def build(support, data):
    return o.build_report(
        support, data, a.build_report(support, REF), physical_report(support), REFS
    )


def features(support, data):
    row = a.select_anchors(support["observations"]["frames"], support["source_alias"])[
        0
    ][0]
    return o.initial_features(
        support, data, row["anchor_id"], ACTOR, row["t0_tick"], row["t0_frame"]
    )


def add_event(data, *, after=False, kind="bomb_planted"):
    data["observations"]["events"].append(
        {
            "sequence": 2,
            "order": 31 if after else 29,
            "frame": 108,
            "tick": 208,
            "instance": 1,
            "epoch": 0,
            "kind": kind,
        }
    )
    for f in data["observations"]["frames"]:
        if f["order"] > (31 if after else 29):
            f["event_sequence"] = 2
            f["rules"]["m_bBombPlanted"] = scalar(kind == "bomb_planted")


def test_exact_objective_facts_and_independent_elapsed_arithmetic():
    support = fixture()
    data = objective(support)
    original = copy.deepcopy(data)
    report = build(support, data)
    initial = features(support, data)
    assert data == original
    assert initial == report["rows"][0]["initial"]
    assert initial["global"]["round_elapsed_seconds"] == o.fact(12 / 64)
    assert initial["global"]["objective_phase"]["value"] == "preplant"
    assert initial["global"]["bomb_elapsed_seconds"]["status"] == "not_applicable"
    assert initial["self"]["in_bomb_zone"]["value"] is False
    assert report["summary"]["anchors"] == report["summary"]["phase_observed"] == 2
    assert all(
        sum(c.values()) == 2 for c in report["summary"]["fact_status_counts"].values()
    )
    encoded = json.dumps(initial)
    assert all(
        s not in encoded
        for s in (
            "actor_alias",
            "support-actor",
            "pawn_life",
            "winner",
            "position",
            "competitive",
            "later_observation",
        )
    )
    initial["self"]["defusing"]["value"] = True
    assert not features(support, data)["self"]["defusing"]["value"]


@pytest.mark.parametrize("after", [False, True])
def test_same_tick_events_obey_delivery_order_and_preserve_zero(after):
    support = fixture()
    data = objective(support)
    add_event(data, after=after)
    result = features(support, data)
    assert result["global"]["objective_phase"]["value"] == (
        "preplant" if after else "planted"
    )
    if not after:
        assert result["global"]["bomb_elapsed_seconds"] == o.fact(0.0)


@pytest.mark.parametrize(
    "mutation",
    ["enemy", "future_death", "future_winner", "future_fields", "future_event"],
)
def test_future_and_enemy_information_does_not_change_features(mutation):
    support = fixture()
    data = objective(support)
    expected = features(support, data)
    if mutation == "enemy":
        for f in data["observations"]["frames"]:
            for actor in f["players"]:
                if actor["actor_alias"] == ENEMY:
                    actor["fields"] = {k: scalar(True) for k in o.PLAYER.values()}
        for f in support["observations"]["frames"]:
            f["players"][-1]["position"]["x"] = 98765
    elif mutation == "future_event":
        add_event(data, after=True, kind="bomb_exploded")
    else:
        for f in data["observations"]["frames"][3:]:
            f[mutation] = "oracle"
        for f in support["observations"]["frames"][3:]:
            f[mutation] = "future"
    assert features(support, data) == expected


@pytest.mark.parametrize(
    "mode",
    [
        "absent",
        "unsupported_type",
        "integer_bool",
        "negative_duration",
        "zero_duration",
    ],
)
def test_native_presence_type_and_zero_are_distinct(mode):
    support = fixture()
    data = objective(support)
    row = data["observations"]["frames"][2]
    if mode in ("absent", "unsupported_type"):
        row["rules"]["m_bBombPlanted"] = {
            "status": mode,
            "native_type": None if mode == "absent" else "float32",
            "value": None,
        }
    elif mode == "integer_bool":
        row["rules"]["m_bBombPlanted"] = scalar(0, "uint64")
    else:
        row["rules"]["m_iRoundTime"] = scalar(
            -1 if mode == "negative_duration" else 0, "int32"
        )
    result = features(support, data)["global"]
    if mode == "zero_duration":
        assert result["round_duration_seconds"] == o.fact(0)
    elif mode == "negative_duration":
        assert result["round_duration_seconds"]["status"] == "unavailable"
    else:
        assert result["bomb_planted"]["value"] is None
        assert result["objective_phase"]["value"] == "preplant"


@pytest.mark.parametrize("boundary", ["epoch", "instance"])
def test_no_objective_state_inherited_across_boundaries(boundary):
    support = fixture()
    for frame in support["observations"]["frames"]:
        frame[boundary] += 1
    data = objective(support)
    # Deliberately retain an earlier epoch/instance public clock origin.
    result = features(support, data)["global"]
    assert result["objective_phase"]["reason"] == "public_phase_origin_absent"
    assert result["round_elapsed_seconds"]["value"] is None


def test_missing_exact_cut_never_uses_previous_sample():
    support = fixture()
    data = objective(support)
    del data["observations"]["frames"][2]
    result = features(support, data)
    assert all(
        f["reason"] == "missing_exact_objective_sample"
        for group in result.values()
        for f in group.values()
    )
    assert build(support, data)["summary"]["anchors"] == 2


def test_positive_native_event_conflict_abstains():
    support = fixture()
    data = objective(support)
    data["observations"]["frames"][2]["rules"]["m_bBombPlanted"] = scalar(True)
    assert (
        features(support, data)["global"]["objective_phase"]["reason"]
        == "native_public_phase_conflict"
    )


def test_terminal_phase_never_leaks_into_restarted_preliminary_playing_instance():
    support = fixture()
    data = objective(support)
    add_event(data, kind="round_end")
    # A second freeze end without RoundStart is observed in the real
    # preliminary instance. It must not inherit the old resolved phase.
    data["observations"]["events"].append(
        {
            "sequence": 3,
            "order": 30,
            "frame": 108,
            "tick": 208,
            "instance": 1,
            "epoch": 0,
            "kind": "freeze_end",
        }
    )
    for cut in data["observations"]["frames"][2:]:
        cut["order"] += 1
        cut["event_sequence"] = 3
    result = features(support, data)["global"]
    assert result["objective_phase"]["reason"] == "public_replay_phase_conflict"
    assert result["bomb_elapsed_seconds"]["status"] == "unavailable"
    assert result["round_elapsed_seconds"] == o.fact(0)


@pytest.mark.parametrize(
    "mutation",
    [
        "tick",
        "frame",
        "instance",
        "epoch",
        "order",
        "frontier",
        "future_event",
        "life",
        "actor",
        "duplicate",
        "presence",
        "type",
        "source",
        "eligible",
        "hidden_carrier",
    ],
)
def test_invalid_identity_clock_frontier_and_oracle_are_rejected(mutation):
    support = fixture()
    data = objective(support)
    row = data["observations"]["frames"][2]
    if mutation in ("tick", "frame", "instance", "epoch"):
        row[mutation] += 1
    elif mutation == "order":
        row["order"] = 1
    elif mutation == "frontier":
        row["event_sequence"] = 0
    elif mutation == "future_event":
        data["observations"]["events"][0]["tick"] = 1000
    elif mutation == "life":
        row["players"][0]["pawn_life"] = "support-pawn-009999"
    elif mutation == "actor":
        row["players"][0]["actor_alias"] = "support-actor-" + "9" * 20
    elif mutation == "duplicate":
        row["players"].append(copy.deepcopy(row["players"][0]))
    elif mutation == "presence":
        row["rules"]["m_bBombPlanted"]["value"] = None
    elif mutation == "type":
        row["rules"]["m_bBombPlanted"]["native_type"] = None
    elif mutation == "source":
        data["source_sha256"] = "9" * 64
    elif mutation == "eligible":
        data["training_allowed"] = True
    else:
        row["hidden_carrier"] = {"position": [0, 0, 0]}
    with pytest.raises((ValueError, ValidationError)):
        o.validate_source(data, support)


def test_absent_pawn_and_reused_identity_never_inherit_fields():
    support = fixture()
    data = objective(support)
    row = data["observations"]["frames"][2]["players"][0]
    row["pawn_life"] = None
    row["fields"] = {
        k: {"status": "absent", "native_type": None, "value": None}
        for k in o.PLAYER.values()
    }
    assert (
        features(support, data)["self"]["defusing"]["reason"]
        == "actor_objective_context_unavailable"
    )
    row["fields"]["m_bIsDefusing"] = scalar(False)
    with pytest.raises(ValueError, match="without pawn"):
        o.validate_source(data, support)


@pytest.mark.parametrize("mode", ["nonempty", "nontemporary"])
def test_materializer_protects_output_before_reading_source(tmp_path, mode):
    output = tmp_path if mode == "nonempty" else Path.cwd()
    if mode == "nonempty":
        (output / "sentinel").write_text("retain")
    with pytest.raises(ValueError):
        o.materialize(
            Path("missing") / o.DATA, output, Path("missing.exe"), Path("maps")
        )


def test_report_checks_unchanged_anchor_and_competitive_boundary():
    support = fixture()
    data = objective(support)
    phys = physical_report(support, rounds=[])
    report = o.build_report(support, data, a.build_report(support, REF), phys, REFS)
    assert report["summary"]["canonical_interval_matched"] == 0
    assert report["summary"]["phase_observed"] == 2
    assert all(not x["decision_eligible"] for x in report["rows"])
    phys["rows"][0]["t0_tick"] += 1
    with pytest.raises(ValueError, match="anchor mismatch"):
        o.build_report(support, data, a.build_report(support, REF), phys, REFS)
