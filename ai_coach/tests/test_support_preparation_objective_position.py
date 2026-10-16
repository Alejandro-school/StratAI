import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import preparation as c
from ai_coach.support import preparation_objective_position as p
from ai_coach.tests import test_support_objective_position as legacy
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, REF


def features(args, index=2, actor=ACTOR, **overrides):
    cut = args[0]["observations"]["frames"][index]
    ally = ALLY if actor == ACTOR else ACTOR
    keys = {
        "identifier": c.candidate_id(args[0]["source_alias"], cut, actor, ally),
        "actor": actor,
        "ally": ally,
        "sequence": cut["sequence"],
        "tick": cut["tick"],
        "frame": cut["frame"],
    }
    keys.update(overrides)
    return p.initial_features(*args, **keys)


def legacy_features(args, actor=ACTOR):
    return {
        role: value["objective_position"]
        for role, value in features(args, actor=actor).items()
    }


@pytest.mark.parametrize(
    "factory",
    [
        legacy.inputs,
        lambda: legacy.inputs(defusing=True),
        legacy.completed_plant_inputs,
    ],
)
def test_c0_all_first_samples_and_three_legal_bases(factory):
    args = factory()
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
            assert actor["initial"] == result[role]["objective_position"]
    assert rows[0]["candidates"]
    for cell in coverage.result()["matrix"]:
        assert all(
            sum(v.values()) == cell["candidates"]
            for v in cell["fact_status_counts"].values()
        )
    assert features(args)["self"]["objective_position"]["physical_objective_position"][
        "value"
    ] == {"x": 0, "y": 0, "z": 0}


@pytest.mark.parametrize(
    "name",
    [
        "test_root_zero_is_native_object_position_with_bilateral_own_possession",
        "test_current_defuser_contact_accredits_only_its_object_and_roles",
        "test_global_plant_and_spotted_do_not_grant_exact_position",
        "test_unrelated_native_object_geometry_is_not_initial_information",
        "test_exact_native_completion_accredits_stationary_plant_position",
        "test_returning_to_same_position_does_not_repair_lost_plant_witness",
        "test_later_motion_does_not_retroactively_remove_legal_plant_position",
    ],
)
def test_legal_position_regressions_at_c0(monkeypatch, name):
    monkeypatch.setattr(legacy, "features", legacy_features)
    getattr(legacy, name)()


@pytest.mark.parametrize("native_type", ["[3]float32", "[3]float64"])
def test_typed_native_rotation_at_c0(monkeypatch, native_type):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_fixed_native_rotation_is_retained_and_validated(native_type)


@pytest.mark.parametrize("index", [0, 1, 2, 3])
@pytest.mark.parametrize("factory", [legacy.inputs, legacy.completed_plant_inputs])
def test_all_six_prefixes_ignore_completely_malformed_future(factory, index):
    args = factory()
    before = features(args, index=index)
    seq = index + 1
    prefix = copy.deepcopy(
        [
            *p.clock.prefix_inputs(*args[:4], seq),
            p.origin.prefix_source(args[4], seq),
            p.prefix_source(args[5], seq),
        ]
    )
    for source, key in zip(
        prefix, ("frames", "frames", "cuts", "frames", "cuts", "cuts")
    ):
        source["observations"][key].append(None)
    for source, keys in [
        (prefix[1], ("events",)),
        (prefix[2], ("states",)),
        (prefix[4], ("frames", "deliveries")),
        (prefix[5], ("states", "births", "deliveries")),
    ]:
        for key in keys:
            source["observations"][key].append(None)
    for row in prefix[0]["observations"]["frames"][:seq]:
        for player in row["players"]:
            if player["actor_alias"] not in (ACTOR, ALLY):
                player["position"] = {"x": 1e7, "y": -1e7, "z": 0}
    assert features(prefix, index=index) == before
    for secret in (
        "support-actor",
        "pawn_handle",
        "pawn_life",
        "position_reference",
        "provenance",
        "birth_sequence",
        "entity_life",
        "native_geometry",
        "candidate_id",
        "execution",
        "matrix",
    ):
        assert secret not in json.dumps(before)
    before["self"]["objective_position"]["physical_objective_position"]["value"] = (
        "mutated"
    )
    assert features(prefix, index=index) == features(args, index=index)


@pytest.mark.parametrize("variant", ["future", "same_frame_future"])
def test_later_position_delivery_frontiers(monkeypatch, variant):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_prefix_excludes_even_malformed_future_deliveries_births_and_states(
        variant
    )


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
def test_exact_c0_candidate_membership(key, value):
    args = legacy.inputs()
    overrides = {key: value}
    if key == "actor":
        overrides["identifier"] = c.candidate_id(
            args[0]["source_alias"], args[0]["observations"]["frames"][2], ACTOR, ALLY
        )
    with pytest.raises((ValueError, ValidationError)):
        features(args, **overrides)


@pytest.mark.parametrize(
    "field", ["side", "pawn_life", "pawn_handle", "instance", "epoch", "missing_actor"]
)
def test_stationary_witness_c0_continuity_starts_at_native_onset(field):
    args = legacy.completed_plant_inputs()
    witnesses = p.k.planting_witnesses(*args)
    frames = copy.deepcopy(args[0]["observations"]["frames"])
    cut = args[5]["observations"]["cuts"][2]
    public = p.k.o.initial_context(
        {"actor_alias": ACTOR, "ally_alias": ACTOR},
        frames[2],
        args[1]["observations"]["frames"][2],
        p.k.o.public_prefixes(args[1])[3],
    )
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
        ACTOR,
        public,
        args[2]["observations"]["states"][2],
        args[5]["observations"]["states"][2],
        cut,
        witnesses,
        p.k.position_changes(args[5]),
        frames,
        [x["frame"] for x in frames],
        args[4]["observations"]["deliveries"],
    )
    assert ref is None
    assert all(
        f["reason"] == "native_position_c0_actor_continuity_unaccredited"
        for f in facts.values()
    )


@pytest.mark.parametrize(
    "variant",
    [
        "predecessor",
        "frontier",
        "birth_frame",
        "life",
        "census",
        "owner",
        "parent",
        "epoch",
        "missing_cut",
        "future_state",
    ],
)
def test_position_source_corruption_is_rejected_at_c0(variant):
    args = legacy.inputs()
    obs = args[5]["observations"]
    if variant == "predecessor":
        obs["deliveries"][0]["previous"]["status"] = "observed"
    elif variant == "frontier":
        obs["cuts"][0]["delivery_sequence"] += 1
    elif variant == "birth_frame":
        obs["births"][0]["frame"] += 1
    elif variant == "life":
        obs["states"][2]["objects"][0]["handle"] += 16384
    elif variant == "census":
        obs["states"][2]["objects"] = []
    elif variant == "owner":
        obs["states"][2]["objects"][0]["owner"]["handle"] += 16384
    elif variant == "parent":
        obs["states"][2]["objects"][0]["parent"]["handle"] = 0
    elif variant == "epoch":
        obs["cuts"][2]["epoch"] += 1
    elif variant == "missing_cut":
        obs["cuts"].pop(2)
    else:
        obs["cuts"][2]["state_sequence"] += 1
    with pytest.raises((ValueError, ValidationError)):
        features(args)


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
def test_reader_recomputes_artifact_and_position_reference(
    tmp_path, monkeypatch, mutation
):
    args = legacy.completed_plant_inputs()
    refs = {
        name: REF
        for name in (
            "support_source",
            "objective_source",
            "c4_source",
            "clock_source",
            "attempt_source",
            "position_source",
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
            rows[2]["actors"][0]["initial"]["physical_objective_position"]["value"][
                "x"
            ] += 0.1
        elif mutation == "origin":
            rows[2]["actors"][0]["position_reference"]["frame"] += 1
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
    args = legacy.completed_plant_inputs()
    args[0]["observations"]["frames"][0]["phase"] = "postround"
    rows = list(p.iter_rows(*args))
    assert (
        rows[0]["actors"] == rows[0]["candidates"] == []
        and rows[0]["provenance"] is None
    )
    assert len(rows) == len(args[0]["observations"]["frames"])
    assert rows[1]["candidates"]
