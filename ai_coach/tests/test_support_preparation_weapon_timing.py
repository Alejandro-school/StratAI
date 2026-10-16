import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import preparation as c
from ai_coach.support import preparation_weapon_timing as p
from ai_coach.tests import test_support_weapon_timing as legacy
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, REF


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
    return {role: value["weapon_timing"] for role, value in features(args).items()}


def test_first_samples_exact_population_and_pawn_clock():
    args = legacy.inputs()
    rows = list(p.iter_rows(*args))
    expected = [v["candidate_id"] for _, pairs in c.iter_census(args[0]) for v in pairs]
    assert [v["candidate_id"] for row in rows for v in row["candidates"]] == expected
    coverage = p.Coverage()
    for index, row in enumerate(rows):
        a.schema_validator(p.ROW_SCHEMA).validate(row)
        coverage.add(row)
        initial = features(args, index=index)
        for actor in row["actors"]:
            role = "self" if actor["actor_alias"] == ACTOR else "ally"
            assert actor["initial"] == initial[role]["weapon_timing"]
    assert rows[0]["candidates"] and rows[0]["provenance"]["clock_witness"]
    assert (
        features(args)["self"]["weapon_timing"]["pawn_remaining_seconds"]["value"]
        == 110 - 6608 / 64
    )
    for cell in coverage.result()["matrix"]:
        assert all(
            sum(v.values()) == cell["candidates"]
            for v in cell["fact_status_counts"].values()
        )


@pytest.mark.parametrize("index", [0, 1, 2, 3])
@pytest.mark.parametrize(
    "future", [None, {"frame": 108, "tick": 208, "oracle": "malformed"}]
)
def test_all_seven_prefixes_ignore_malformed_future_and_enemy_geometry(index, future):
    args = legacy.inputs()
    before = features(args, index=index)
    seq = index + 1
    prefix = copy.deepcopy(p.prefix_inputs(*args, seq))
    for source, key in zip(
        prefix, ("frames", "cuts", "frames", "frames", "cuts", "frames", "cuts")
    ):
        source["observations"][key].append(copy.deepcopy(future))
    for i, keys in [
        (1, ("states", "deliveries")),
        (3, ("events",)),
        (4, ("states",)),
        (6, ("states", "deliveries")),
    ]:
        for key in keys:
            prefix[i]["observations"][key].append(copy.deepcopy(future))
    for row in prefix[0]["observations"]["frames"][:seq]:
        for player in row["players"]:
            if player["actor_alias"] not in (ACTOR, ALLY):
                player["position"] = {"x": 1e7, "y": -1e7, "z": 0}
    assert features(prefix, index=index) == before
    for secret in (
        "support-actor",
        "pawn_handle",
        "pawn_life",
        "timing_reference",
        "clock_witness",
        "provenance",
        "entity_life",
        "candidate_id",
        "execution",
        "matrix",
        "oracle",
        "server_tick",
        "m_flNextAttack",
    ):
        assert secret not in json.dumps(before)
    before["self"]["weapon_timing"]["pawn_remaining_seconds"]["value"] = 999
    assert features(prefix, index=index) == features(args, index=index)


@pytest.mark.parametrize(
    "key,value",
    [
        ("identifier", "invalid"),
        ("actor", ALLY),
        ("ally", ACTOR),
        ("sequence", 2),
        ("tick", 207),
        ("frame", 107),
        ("sequence", 999),
    ],
)
def test_exact_c0_membership(key, value):
    with pytest.raises((ValueError, ValidationError)):
        features(legacy.inputs(), **{key: value})


@pytest.mark.parametrize(
    "field", ["side", "pawn_life", "pawn_handle", "instance", "epoch", "missing_actor"]
)
def test_retained_pawn_state_cannot_cross_identity_or_domain_then_return(field):
    args = legacy.inputs()
    support = copy.deepcopy(args[0])
    first = support["observations"]["frames"][0]
    if field in ("instance", "epoch"):
        first[field] += 1
    elif field == "missing_actor":
        first["players"] = [x for x in first["players"] if x["actor_alias"] != ACTOR]
    else:
        player = next(x for x in first["players"] if x["actor_alias"] == ACTOR)
        player[field] = (
            "T"
            if field == "side"
            else 99999
            if field == "pawn_handle"
            else "support-pawn-999999"
        )
    alignment, _ = p.k.c.alignment_prefixes(args[0], args[3], args[5])
    facts, ref = p.actor_context(
        ACTOR, support, args[6], args[2], args[5], 3, alignment[3]
    )
    assert ref is None
    assert all(
        f["reason"] == "native_timing_c0_actor_continuity_unaccredited"
        for f in facts.values()
    )


@pytest.mark.parametrize(
    "key",
    [
        "clock_order",
        "frame",
        "tick",
        "instance",
        "epoch",
        "source_sequence",
        "state_sequence",
        "delivery_sequence",
    ],
)
def test_common_cut_corruption_rejected(key):
    args = legacy.inputs()
    args[6]["observations"]["cuts"][0][key] += 1
    with pytest.raises((ValueError, ValidationError)):
        features(args)


@pytest.mark.parametrize(
    "name",
    [
        "test_positive_native_game_time_clock_and_no_general_readiness",
        "test_nonfinite_native_retained_as_unavailable",
        "test_enemy_and_malformed_future_cannot_change_initial_features",
        "test_non_ammunition_category_retains_native_without_firearm_deadline",
    ],
)
def test_native_semantics_at_c0(monkeypatch, name):
    monkeypatch.setattr(legacy, "features", legacy_features)
    getattr(legacy, name)()


@pytest.mark.parametrize(
    *legacy.test_zero_absence_sentinel_type_and_elapsed_deadline.pytestmark[0].args
)
def test_zero_absence_sentinel_type_and_elapsed_deadline_at_c0(
    monkeypatch, value, typ, declared, expected
):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_zero_absence_sentinel_type_and_elapsed_deadline(
        value, typ, declared, expected
    )


@pytest.mark.parametrize(
    *legacy.test_clock_proof_invalidations_and_pause.pytestmark[0].args
)
def test_clock_proof_invalidations_and_pause_at_c0(monkeypatch, mutation):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_clock_proof_invalidations_and_pause(mutation)


@pytest.mark.parametrize(
    *legacy.test_tick_fraction_never_assumes_unproved_formula.pytestmark[0].args
)
def test_tick_fraction_never_assumes_unproved_formula_at_c0(monkeypatch, tick, ratio):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_tick_fraction_never_assumes_unproved_formula(tick, ratio)


@pytest.mark.parametrize(
    *legacy.test_adversarial_provenance_rejected.pytestmark[0].args
)
def test_adversarial_provenance_rejected_at_c0(monkeypatch, mutation):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_adversarial_provenance_rejected(mutation)


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
        "witness",
    ],
)
def test_reader_recomputes_artifact_and_timing_reference(
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
            "position_source",
            "physical_source",
            "timing_source",
            "motion_source",
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
            rows[2]["actors"][0]["initial"]["pawn_remaining_seconds"]["value"] += 0.1
        elif mutation == "witness":
            rows[2]["provenance"]["clock_witness"]["server_tick"] += 1
        elif mutation == "origin":
            rows[2]["actors"][0]["timing_reference"]["frame"] += 1
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


def test_manifest_and_reader_retain_c0_continuity_abstention(tmp_path, monkeypatch):
    args = legacy.inputs()
    prior = args[0]["observations"]["frames"][0]
    next(x for x in prior["players"] if x["actor_alias"] == ACTOR)["side"] = "T"
    schema = a.schema_validator(p.MANIFEST).schema
    refs = {name: REF for name in schema["properties"]["references"]["required"]}
    refs["preparation_protocol"] = c.protocol()[1]
    monkeypatch.setattr(p, "load_inputs", lambda _: (args, refs))
    report = p.materialize(Path("unused"), tmp_path, require_observed=True)
    assert p.validate_artifact(Path("unused"), tmp_path) == report
    assert any(
        cell["fact_status_counts"]["self.pawn_remaining_seconds"].get(
            "unavailable:native_timing_c0_actor_continuity_unaccredited", 0
        )
        for cell in report["coverage"]["matrix"]
    )


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


@pytest.mark.parametrize("kind", ["pawn", "weapon"])
def test_missing_native_object_preserves_c0_and_independent_pawn_deadline(kind):
    args = legacy.inputs()
    obs = args[6]["observations"]
    removed = {1, 2} if kind == "pawn" else {2}
    mapping = {}
    states = []
    for state in obs["states"]:
        if state["sequence"] not in removed:
            mapping[state["sequence"]] = len(states) + 1
            state["sequence"] = len(states) + 1
            states.append(state)
    obs["states"] = states
    for cut in obs["cuts"]:
        cut["state_sequence"] = len(states)
        for player in cut["players"]:
            if player["actor_alias"] == ACTOR:
                player["weapon_life"] = player["weapon_state_sequence"] = None
                player["weapon_status"] = "current_weapon_unavailable"
                if kind == "pawn":
                    player["pawn_life"] = player["pawn_handle"] = player[
                        "pawn_state_sequence"
                    ] = None
                    player["weapon_status"] = "current_pawn_unavailable"
            for name in ("pawn_state_sequence", "weapon_state_sequence"):
                if player[name] is not None:
                    player[name] = mapping[player[name]]
    actual = features(args)["self"]["weapon_timing"]
    if kind == "pawn":
        assert all(
            v["reason"] == "current_timing_pawn_unavailable" for v in actual.values()
        )
    else:
        assert (
            actual["primary_remaining_seconds"]["reason"]
            == "current_weapon_unavailable"
        )
        assert actual["pawn_remaining_seconds"]["value"] == 110 - 6608 / 64
    assert [
        v["candidate_id"] for row in p.iter_rows(*args) for v in row["candidates"]
    ] == [v["candidate_id"] for _, pairs in c.iter_census(args[0]) for v in pairs]


@pytest.mark.parametrize("mutation", ["sequence", "missing", "reordered"])
def test_physical_snapshot_must_be_current_exact_cut(mutation):
    args = legacy.inputs()
    frames = args[2]["observations"]["frames"]
    if mutation == "sequence":
        frames[2]["source_sequence"] -= 1
    elif mutation == "missing":
        frames.pop(2)
    else:
        frames[1], frames[2] = frames[2], frames[1]
    with pytest.raises((ValueError, ValidationError)):
        features(args)


@pytest.mark.parametrize("kind", ["other_weapon", "serial", "physical_life"])
def test_current_weapon_cannot_borrow_another_object_state(kind):
    args = legacy.inputs()
    if kind == "other_weapon":
        args[6]["observations"]["cuts"][2]["players"][0]["weapon_state_sequence"] = 4
    elif kind == "serial":
        args[6]["observations"]["states"][1]["handle"] += 16384
    else:
        args[2]["observations"]["frames"][2]["players"][0]["context"]["weapon_life"] = (
            "support-weapon-999999"
        )
    with pytest.raises((ValueError, ValidationError)):
        features(args)
