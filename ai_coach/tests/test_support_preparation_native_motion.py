import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import preparation as c
from ai_coach.support import preparation_native_motion as p
from ai_coach.tests import test_support_native_motion as legacy
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
    return {role: value["native_motion"] for role, value in features(args).items()}


def test_c0_includes_first_samples_and_preserves_population_and_six_facts():
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
            assert actor["initial"] == result[role]["native_motion"]
    assert rows[0]["candidates"]
    for cell in coverage.result()["matrix"]:
        assert all(
            sum(v.values()) == cell["candidates"]
            for v in cell["fact_status_counts"].values()
        )
    assert features(args)["self"]["native_motion"]["native_speed"]["value"] == 13


@pytest.mark.parametrize("index", [0, 1, 2, 3])
@pytest.mark.parametrize(
    "future", [None, {"frame": 108, "tick": 208, "oracle": "malformed"}]
)
def test_all_prefixes_ignore_future_even_at_same_frame_and_enemy_geometry(
    index, future
):
    args = legacy.inputs()
    before = features(args, index=index)
    seq = index + 1
    prefix = copy.deepcopy(
        [
            args[0],
            p.position_c0.prefix_source(args[1], seq),
            p.prefix_source(args[2], seq),
        ]
    )
    prefix[0]["observations"]["frames"] = prefix[0]["observations"]["frames"][:seq]
    for source, key in zip(prefix, ("frames", "cuts", "cuts")):
        source["observations"][key].append(copy.deepcopy(future))
    for source, keys in [
        (prefix[1], ("states", "births", "deliveries")),
        (prefix[2], ("states", "deliveries")),
    ]:
        for key in keys:
            source["observations"][key].append(copy.deepcopy(future))
    for row in prefix[0]["observations"]["frames"][:seq]:
        for player in row["players"]:
            if player["actor_alias"] not in (ACTOR, ALLY):
                player["position"] = {"x": 1e7, "y": -1e7, "z": 0}
    assert features(prefix, index=index) == before
    for secret in (
        "support-actor",
        "pawn_handle",
        "pawn_life",
        "motion_reference",
        "provenance",
        "entity_life",
        "candidate_id",
        "execution",
        "matrix",
        "oracle",
    ):
        assert secret not in json.dumps(before)
    before["self"]["native_motion"]["native_velocity"]["value"]["x"] = 999
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
def test_retained_state_cannot_cross_actor_life_or_domain_then_return(field):
    args = legacy.inputs()
    frames = copy.deepcopy(args[0]["observations"]["frames"])
    if field in ("instance", "epoch"):
        frames[0][field] += 1
    elif field == "missing_actor":
        frames[0]["players"] = [
            x for x in frames[0]["players"] if x["actor_alias"] != ACTOR
        ]
    else:
        player = next(x for x in frames[0]["players"] if x["actor_alias"] == ACTOR)
        player[field] = (
            "T"
            if field == "side"
            else 99999
            if field == "pawn_handle"
            else "support-pawn-999999"
        )
    facts, ref = p.actor_context(
        ACTOR, args[2], args[2]["observations"]["cuts"][2], frames
    )
    assert ref is None
    assert all(
        f["reason"] == "native_motion_c0_actor_continuity_unaccredited"
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
    args[2]["observations"]["cuts"][0][key] += 1
    with pytest.raises((ValueError, ValidationError)):
        features(args)


@pytest.mark.parametrize(
    "kind",
    [
        "future_field",
        "predecessor",
        "creation_replay",
        "wrong_handle",
        "wrong_pawn",
        "epoch",
        "first_cut",
        "destroyed",
        "orphan_state",
    ],
)
def test_native_identity_deliveries_rejected_in_c0_consumer(monkeypatch, kind):
    validate = p.k.validate_source

    # Route the legacy corruption fixtures through the exact C0 public API.
    def validate_at_c0(data, support, position):
        with monkeypatch.context() as patch:
            patch.setattr(p.k, "validate_source", validate)
            features([support, position, data])

    monkeypatch.setattr(legacy.s, "validate_source", validate_at_c0)
    if kind == "orphan_state":
        # An undelivered trailing state is future data and must be ignored.
        args = legacy.inputs()
        before = features(args)
        args[2]["observations"]["states"].append(None)
        assert features(args) == before
    else:
        legacy.test_state_delivery_and_pawn_identity_corruption_is_rejected(kind)


@pytest.mark.parametrize(
    "name",
    [
        "test_native_world_vector_and_ground_are_independent_of_mean_motion",
        "test_current_airborne_proof_uses_false_flag_and_explicit_invalid_ground",
        "test_conflicting_world_velocity_sources_abstain",
        "test_dynamic_ground_contact_does_not_claim_world_floor_or_route",
        "test_retained_baseline_does_not_require_a_new_delivery_at_every_cut",
        "test_unavailable_pawn_keeps_all_motion_facts_unavailable",
        "test_malformed_future_and_enemy_motion_do_not_change_initial_features",
    ],
)
def test_native_semantics_at_c0(monkeypatch, name):
    monkeypatch.setattr(legacy, "features", legacy_features)
    getattr(legacy, name)()


SEMANTIC_CASES = (
    [
        ("test_present_native_zero_vector_is_distinct_from_absence", value)
        for value in legacy.test_present_native_zero_vector_is_distinct_from_absence.pytestmark[
            0
        ].args[1]
    ]
    + [
        ("test_base_and_historical_values_never_replace_current_velocity", value)
        for value in legacy.test_base_and_historical_values_never_replace_current_velocity.pytestmark[
            0
        ].args[1]
    ]
    + [
        ("test_vector_name_or_values_without_reference_do_not_accredit_velocity", value)
        for value in legacy.test_vector_name_or_values_without_reference_do_not_accredit_velocity.pytestmark[
            0
        ].args[1]
    ]
    + [
        ("test_partial_nonfinite_or_boolean_velocity_components_are_rejected", value)
        for value in legacy.test_partial_nonfinite_or_boolean_velocity_components_are_rejected.pytestmark[
            0
        ].args[1]
    ]
    + [
        ("test_world_velocity_requires_unambiguous_current_parent_context", value)
        for value in legacy.test_world_velocity_requires_unambiguous_current_parent_context.pytestmark[
            0
        ].args[1]
    ]
    + [
        (
            "test_ground_contact_requires_presence_types_and_concordant_current_reference",
            value,
        )
        for value in legacy.test_ground_contact_requires_presence_types_and_concordant_current_reference.pytestmark[
            0
        ].args[1]
    ]
)


@pytest.mark.parametrize("name,value", SEMANTIC_CASES)
def test_typed_native_semantics_at_c0(monkeypatch, name, value):
    monkeypatch.setattr(legacy, "features", legacy_features)
    getattr(legacy, name)(value)


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
def test_reader_recomputes_artifact_and_motion_reference(
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
            rows[2]["actors"][0]["initial"]["native_velocity"]["value"]["x"] += 0.1
        elif mutation == "origin":
            rows[2]["actors"][0]["motion_reference"]["frame"] += 1
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
        cell["fact_status_counts"]["self.ground_contact"].get(
            "unavailable:native_motion_c0_actor_continuity_unaccredited", 0
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
