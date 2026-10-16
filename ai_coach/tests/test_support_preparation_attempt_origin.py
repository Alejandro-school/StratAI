import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import preparation as c
from ai_coach.support import preparation_attempt_origin as p
from ai_coach.tests import test_support_attempt_origin as legacy
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF


def features(args, index=2, **overrides):
    cut = args[0]["observations"]["frames"][index]
    keys = {
        "identifier": c.candidate_id(args[0]["source_alias"], cut, ACTOR, ALLY),
        "actor": ACTOR,
        "ally": ALLY,
        "sequence": cut["sequence"],
        "tick": cut["tick"],
        "frame": cut["frame"],
    }
    keys.update(overrides)
    return p.initial_features(*args, **keys)


def legacy_features(args):
    return {role: value["attempt_origin"] for role, value in features(args).items()}


def test_first_samples_population_numeric_progress_and_provenance():
    args = legacy.inputs()
    rows = list(p.iter_rows(*args))
    expected = [v["candidate_id"] for _, pairs in c.iter_census(args[0]) for v in pairs]
    assert [v["candidate_id"] for row in rows for v in row["candidates"]] == expected
    coverage = p.Coverage()
    for i, row in enumerate(rows):
        a.schema_validator(p.ROW_SCHEMA).validate(row)
        coverage.add(row)
        result = features(args, index=i)
        for actor in row["actors"]:
            role = "self" if actor["actor_alias"] == ACTOR else "ally"
            assert actor["initial"] == result[role]["attempt_origin"]
    assert (
        features(args, index=0)["self"]["attempt_origin"]["planting_progress"]["status"]
        == "not_applicable"
    )
    result = features(args)["self"]["attempt_origin"]
    duration = 106.25 - 6602 / 64
    assert result["planting_duration_seconds"]["value"] == duration
    assert result["planting_elapsed_seconds"]["value"] == 6 / 64
    assert result["planting_progress"]["value"] == 1 - 3 / duration
    assert rows[2]["actors"][0]["origin_reference"]["frame"] == 102
    for cell in coverage.result()["matrix"]:
        assert all(
            sum(v.values()) == cell["candidates"]
            for v in cell["fact_status_counts"].values()
        )


@pytest.mark.parametrize(
    "mode",
    [
        "paused",
        "pause_missing",
        "tick_absent",
        "tick_disagrees",
        "multiple_packet_ticks",
        "interval_change",
        "rules_life",
        "round",
        "epoch",
        "rewind",
        "native_type",
        "undeclared_deadline",
        "inventory_missing",
        "owner_serial",
        "duplicate_slot",
        "frame_gap",
    ],
)
def test_c0_origin_intermediate_ambiguities(monkeypatch, mode):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_positive_origin_is_invalidated_by_intermediate_ambiguity(mode)


@pytest.mark.parametrize(
    "mode",
    [
        "forged_baseline",
        "future_delivery",
        "predecessor",
        "snapshot",
        "cut_frontier",
        "clock_join",
        "unknown_field",
        "life_reuse",
    ],
)
def test_c0_rejects_fabricated_sources(monkeypatch, mode):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_malformed_or_fabricated_sources_are_rejected(mode)


@pytest.mark.parametrize(
    "mode",
    [
        "baseline_active",
        "creation_replay",
        "stale_deadline",
        "zero_duration",
        "negative_duration",
        "multiple_starts",
        "deadline_changed",
        "cancelled",
        "restart",
        "deadline_before_flag",
        "expired",
    ],
)
def test_c0_start_duration_and_attempt_boundaries(monkeypatch, mode):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_start_duration_and_attempt_boundaries(mode)


@pytest.mark.parametrize("index", [0, 1, 2, 3])
def test_exact_prefix_ignores_uninspectable_future_and_returns_copies(index):
    args = legacy.inputs()
    expected = features(args, index=index)
    sequence = args[0]["observations"]["frames"][index]["sequence"]
    prefix = [
        *copy.deepcopy(p.clock.prefix_inputs(*args[:4], sequence)),
        copy.deepcopy(p.prefix_source(args[4], sequence)),
    ]
    for source, key in zip(
        prefix, ("frames", "frames", "cuts", "frames", "cuts"), strict=True
    ):
        source["observations"][key].append(None)
    prefix[1]["observations"]["events"].append(None)
    prefix[2]["observations"]["states"].append(None)
    prefix[4]["observations"]["frames"].append(None)
    prefix[4]["observations"]["deliveries"].append(None)
    for row in prefix[0]["observations"]["frames"][:sequence]:
        row["players"][2]["position"] = {"x": 1e8, "y": -1e8, "z": 1000}
    assert features(prefix, index=index) == expected
    for secret in (
        ACTOR,
        ALLY,
        ENEMY,
        "origin_reference",
        "delivery_sequence",
        "frame_sequence",
        "clock_order",
        "pawn_life",
        "pawn_handle",
        "objective_clock",
        "matrix",
        "execution",
    ):
        assert secret not in json.dumps(expected)
    expected["self"]["attempt_origin"]["planting_progress"]["value"] = 999
    assert features(prefix, index=index) != expected


@pytest.mark.parametrize(
    "key,value",
    [
        ("identifier", "wrong"),
        ("actor", ENEMY),
        ("ally", ACTOR),
        ("sequence", True),
        ("sequence", 2),
        ("tick", 200),
        ("frame", 100),
    ],
)
def test_exact_c0_candidate_membership(key, value):
    with pytest.raises(ValueError):
        features(legacy.inputs(), **{key: value})


def test_cut_frontiers_exclude_later_deliveries_and_frames_even_with_same_tick():
    args = legacy.inputs()
    expected = features(args, index=1)
    cut = args[4]["observations"]["cuts"][1]
    args[4]["observations"]["frames"][cut["frame_sequence"] :] = [
        {"frame": cut["frame"], "tick": cut["tick"], "malformed_later_frame": True}
    ]
    args[4]["observations"]["deliveries"][cut["delivery_sequence"] :] = [
        {"frame": cut["frame"], "tick": cut["tick"], "malformed_later_delivery": True}
    ]
    assert features(args, index=1) == expected


def test_later_clock_origin_cannot_accredit_earlier_native_start():
    args = legacy.inputs()
    later = args[3]["observations"]["frames"][1]
    origin = later["network"]["tick"]["value"]["value"] / 64
    args[1]["observations"]["events"][0].update(
        frame=later["frame"], tick=later["tick"], order=15
    )
    args[1]["observations"]["frames"][0]["event_sequence"] = 0
    for source in (args[3], args[4]):
        for row in source["observations"]["frames"]:
            row["rules"]["fields"]["m_fRoundStartTime"] = legacy.scalar(
                origin, declared="GameTime_t"
            )
    # Correspondence now exists at T0, but was unavailable at the native start.
    assert (
        p.clock.initial_features(
            *args[:4],
            c.candidate_id(
                args[0]["source_alias"],
                args[0]["observations"]["frames"][2],
                ACTOR,
                ALLY,
            ),
            ACTOR,
            ALLY,
            3,
            208,
            108,
        )["self"]["objective_clock"]["clock_state"]["value"]
        == "aligned_native_round_origin"
    )
    assert (
        features(args)["self"]["attempt_origin"]["planting_progress"]["status"]
        == "unavailable"
    )


def test_unrelated_enemy_native_clocks_do_not_change_c0_progress(monkeypatch):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_unrelated_player_clocks_do_not_change_own_or_ally_planting_progress()


@pytest.mark.parametrize(
    "key", ["side", "pawn_handle", "pawn_life", "instance", "epoch"]
)
def test_origin_cannot_cross_c0_actor_or_round_boundary(key):
    args = legacy.inputs()
    rows = list(p.iter_rows(*args))
    assert rows[2]["actors"][0]["initial"]["planting_progress"]["status"] == "observed"
    original = p.k.origin_prefixes(*args)
    frames = args[0]["observations"]["frames"]
    previous = list(p.clock.iter_rows(*args[:4]))[2]["actors"][0]["initial"]
    cut = args[4]["observations"]["cuts"][2]
    row = args[4]["observations"]["frames"][cut["frame_sequence"] - 1]
    owner = args[2]["observations"]["states"][2]
    if key in ("instance", "epoch"):
        frames[0][key] += 1
    else:
        actor = frames[0]["players"][0]
        actor[key] = (
            ("T" if actor["side"] == "CT" else "CT")
            if key == "side"
            else 999
            if key == "pawn_handle"
            else "support-pawn-999999"
        )
    facts, proof = p.actor_context(
        ACTOR,
        previous,
        frames[2],
        row,
        original[cut["frame_sequence"]],
        owner,
        frames,
        [r["frame"] for r in frames],
    )
    assert proof is None
    assert all(
        f["reason"] == "native_origin_c0_actor_continuity_unaccredited"
        for f in facts.values()
    )


@pytest.mark.parametrize(
    "mutation",
    [
        "none",
        "fact",
        "origin",
        "cut",
        "frontier",
        "candidate",
        "missing",
        "extra",
        "reorder",
        "hash",
        "coverage",
    ],
)
def test_reader_recomputes_artifact_and_origin_reference(
    tmp_path, monkeypatch, mutation
):
    args = legacy.inputs()
    refs = {
        name: REF
        for name in (
            "support_source",
            "objective_source",
            "c4_source",
            "clock_source",
            "attempt_source",
        )
    }
    refs["preparation_protocol"] = c.protocol()[1]
    monkeypatch.setattr(p, "load_inputs", lambda _: (args, refs))
    report = p.materialize(Path("unused"), tmp_path, require_observed=True)
    if mutation == "none":
        assert p.validate_artifact(Path("unused"), tmp_path) == report
        return
    if mutation in ("hash", "coverage"):
        target = tmp_path / p.MANIFEST
        value = json.loads(target.read_bytes())
        if mutation == "hash":
            value["artifact"]["sha256"] = "0" * 64
        else:
            value["coverage"]["counts"]["candidates"] += 1
        target.write_text(json.dumps(value))
    else:
        target = tmp_path / p.ROWS
        rows = [json.loads(line) for line in target.read_bytes().splitlines()]
        if mutation == "fact":
            rows[2]["actors"][0]["initial"]["planting_progress"]["value"] += 0.1
        elif mutation == "origin":
            rows[2]["actors"][0]["origin_reference"]["frame"] += 1
        elif mutation == "cut":
            rows[2]["cut"]["frame"] += 1
        elif mutation == "frontier":
            rows[2]["provenance"]["delivery_sequence"] += 1
        elif mutation == "candidate":
            rows[0]["candidates"].pop()
        elif mutation == "missing":
            rows.pop()
        elif mutation == "extra":
            rows.append(rows[0])
        elif mutation == "reorder":
            rows.reverse()
        target.write_bytes(b"".join(c.encoded_row(row) for row in rows))
    with pytest.raises((ValueError, ValidationError)):
        p.validate_artifact(Path("unused"), tmp_path)


def test_preflight_precedes_source(tmp_path, monkeypatch):
    monkeypatch.setattr(
        p, "load_inputs", lambda _: pytest.fail("read before preflight")
    )
    (tmp_path / "retained").write_text("keep")
    for path in (tmp_path, Path.cwd()):
        with pytest.raises(ValueError):
            p.materialize(Path("unused"), path)


def test_excluded_and_first_cut_are_not_lost_for_absent_origin():
    args = legacy.inputs()
    args[0]["observations"]["frames"][0]["phase"] = "postround"
    rows = list(p.iter_rows(*args))
    assert (
        rows[0]["actors"] == rows[0]["candidates"] == []
        and rows[0]["provenance"] is None
    )
    assert len(rows) == len(args[0]["observations"]["frames"])
    assert rows[1]["candidates"]
