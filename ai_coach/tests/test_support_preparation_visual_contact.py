import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import preparation as c
from ai_coach.support import preparation_visual_contact as p
from ai_coach.tests import test_support_visual_contact as legacy
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF

PHYSICS = {"checked": 8, "clear": 8, "blocked": 0, "unavailable": 0, "mismatches": 0}


def sources(args):
    return args[0], args[2], args[3]


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
    return p.initial_features(*sources(args), **keys)


def legacy_features(args):
    return {role: value["visual_inputs"] for role, value in features(args).items()}


def mock_load(monkeypatch, args):
    refs = {
        n: REF
        for n in a.schema_validator(p.MANIFEST).schema["properties"]["references"][
            "required"
        ]
    }
    refs["preparation_protocol"] = c.protocol()[1]
    monkeypatch.setattr(
        p, "load_inputs", lambda *unused: (sources(args), refs, PHYSICS.copy())
    )


def test_first_samples_population_facts_and_matrix():
    args = legacy.inputs()
    rows = list(p.iter_rows(*sources(args)))
    assert [x["candidate_id"] for row in rows for x in row["candidates"]] == [
        x["candidate_id"] for _, pairs in c.iter_census(args[0]) for x in pairs
    ]
    coverage = p.Coverage()
    for index, row in enumerate(rows):
        a.schema_validator(p.ROW_SCHEMA).validate(row)
        coverage.add(row)
        result = features(args, index)
        for actor in row["actors"]:
            role = "self" if actor["actor_alias"] == ACTOR else "ally"
            assert result[role]["visual_inputs"] == actor["initial"]
    assert rows[0]["candidates"]
    assert features(args)["self"]["visual_inputs"]["native_eye_offset"]["value"] == {
        "x": 0,
        "y": 0,
        "z": 64,
    }
    for cell in coverage.result()["matrix"]:
        assert all(
            sum(v.values()) == cell["candidates"]
            for v in cell["fact_status_counts"].values()
        )


@pytest.mark.parametrize("index", [0, 1, 2, 3])
@pytest.mark.parametrize(
    "future",
    [None, {"source_sequence": 3, "frame": 108, "tick": 208, "malformed": True}],
)
def test_three_prefixes_hidden_geometry_rays_and_detached_copies(index, future):
    args = legacy.inputs()
    before = features(args, index)
    seq = index + 1
    support, timing, data = copy.deepcopy(p.prefix_inputs(*sources(args), seq))
    for source, key in [(support, "frames"), (timing, "cuts"), (data, "cuts")]:
        source["observations"][key].append(copy.deepcopy(future))
    for source in (timing, data):
        for key in ("states", "deliveries"):
            source["observations"][key].append(copy.deepcopy(future))
    for base, cut in zip(
        support["observations"]["frames"][:seq],
        data["observations"]["cuts"][:seq],
        strict=True,
    ):
        for i, actor in enumerate(base["players"]):
            if actor["actor_alias"] == ENEMY:
                actor["position"]["x"] += 10
                cut["players"][i]["coordinates"]["CBodyComponent.m_vecX"]["value"] += 10
        for ray in cut["rays"]:
            ray["clear"] = not ray["clear"]
    changed = support, args[1], timing, data
    assert features(changed, index) == before
    for secret in (
        "support-actor",
        "pawn_handle",
        "pawn_life",
        "visual_reference",
        "state_sequence",
        "provenance",
        "rays",
        "subject_index",
        "coordinates",
        "evaluator_diagnostics",
        "matrix",
        "candidate_id",
        "execution",
        "geometry",
    ):
        assert secret not in json.dumps(before)
    before["self"]["visual_inputs"]["native_eye_offset"]["value"]["z"] = 999
    assert features(changed, index) == features(args, index)


@pytest.mark.parametrize(
    "key,value",
    [
        ("identifier", "wrong"),
        ("actor", ALLY),
        ("ally", ACTOR),
        ("sequence", 2),
        ("sequence", 999),
        ("tick", 207),
        ("frame", 107),
    ],
)
def test_exact_membership(key, value):
    with pytest.raises((ValueError, ValidationError)):
        features(legacy.inputs(), **{key: value})


@pytest.mark.parametrize(
    "field", ["side", "pawn_life", "pawn_handle", "instance", "epoch", "missing_actor"]
)
def test_retained_state_continuity_fails_closed(field):
    args = legacy.inputs()
    support = copy.deepcopy(args[0])
    first = support["observations"]["frames"][0]
    if field in ("instance", "epoch"):
        first[field] += 1
    elif field == "missing_actor":
        first["players"] = [x for x in first["players"] if x["actor_alias"] != ACTOR]
    else:
        actor = next(x for x in first["players"] if x["actor_alias"] == ACTOR)
        actor[field] = (
            "T"
            if field == "side"
            else 99999
            if field == "pawn_handle"
            else "support-pawn-999999"
        )
    facts, ref = p.actor_context(ACTOR, support, args[3], 3)
    assert ref is None
    assert all(
        facts[n]["reason"] == "native_visual_c0_actor_continuity_unaccredited"
        for n in p.k.PLAYER[:2]
    )
    assert all(facts[n] == p.k.native_facts(None)[n] for n in p.k.PLAYER[2:])


@pytest.mark.parametrize(
    *legacy.test_camera_presence_default_range_transition_and_scope.pytestmark[0].args
)
def test_camera_semantics_at_c0(monkeypatch, name, value, reason):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_camera_presence_default_range_transition_and_scope(name, value, reason)


@pytest.mark.parametrize("duration", [0, 0.001, 2, 20])
def test_flash_never_unlocks_contact(monkeypatch, duration):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_flash_duration_and_max_alpha_never_stand_in_for_rendered_clearance(
        duration
    )


@pytest.mark.parametrize(
    *legacy.test_forged_causal_identity_or_physics_provenance_is_rejected.pytestmark[
        0
    ].args
)
def test_forged_provenance_rejected_at_c0(monkeypatch, mutation):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_forged_causal_identity_or_physics_provenance_is_rejected(mutation)


def test_missing_geometry_and_eye_remain_distinct(monkeypatch):
    monkeypatch.setattr(legacy, "features", legacy_features)
    legacy.test_geometry_absent_and_native_eye_absent_are_distinct_from_obstruction()


@pytest.mark.parametrize("value", [True, False])
@pytest.mark.parametrize("name", ["flash_clear", "smoke_clear", "visual_contact"])
def test_schema_rejects_fabricated_positive_and_negative(name, value):
    row = next(p.iter_rows(*sources(legacy.inputs())))
    row["actors"][0]["initial"][name] = p.k.o.fact(value)
    with pytest.raises(ValidationError):
        a.schema_validator(p.ROW_SCHEMA).validate(row)


@pytest.mark.parametrize(
    "value",
    [
        legacy.scalar(),
        legacy.scalar(2, "uint64", "uint32"),
        dict(
            legacy.scalar(typ="float32", declared="CNetworkedQuantizedFloat"),
            status="nonfinite",
        ),
    ],
)
def test_eye_absence_type_and_nonfinite(value):
    args = legacy.inputs()
    legacy.replace_field(args, "m_vecZ", value)
    for cut in args[3]["observations"]["cuts"]:
        for ray in cut["rays"]:
            if 0 in (ray["observer_index"], ray["subject_index"]):
                ray["clear"] = None
    assert (
        features(args)["self"]["visual_inputs"]["native_eye_offset"]["status"]
        == "unavailable"
    )


@pytest.mark.parametrize(
    "mutation",
    [
        "none",
        "fact",
        "reference",
        "cut",
        "frontier",
        "candidate",
        "missing",
        "extra",
        "reorder",
        "hash",
        "coverage",
        "physics",
    ],
)
def test_reader_recomputes_every_artifact(tmp_path, monkeypatch, mutation):
    args = legacy.inputs()
    mock_load(monkeypatch, args)
    report = p.materialize(Path("unused"), tmp_path, None, None, require_observed=True)
    if mutation == "none":
        assert p.validate_artifact(Path("unused"), tmp_path, None, None) == report
        return
    if mutation in ("hash", "coverage", "physics"):
        target = tmp_path / p.MANIFEST
        value = json.loads(target.read_bytes())
        if mutation == "hash":
            value["artifact"]["sha256"] = "0" * 64
        elif mutation == "physics":
            value["evaluator_diagnostics"]["static_ray_recomputation"]["clear"] += 1
        else:
            value["coverage"]["counts"]["candidates"] += 1
        target.write_text(json.dumps(value))
    else:
        target = tmp_path / p.ROWS
        rows = [json.loads(line) for line in target.read_bytes().splitlines()]
        if mutation == "fact":
            rows[2]["actors"][0]["initial"]["reference_fov_degrees"]["value"] += 1
        elif mutation == "reference":
            rows[2]["actors"][0]["visual_reference"]["frame"] += 1
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
        p.validate_artifact(Path("unused"), tmp_path, None, None)


def test_preflight_before_loading_or_physics(tmp_path, monkeypatch):
    monkeypatch.setattr(
        p, "load_inputs", lambda *args: pytest.fail("source read before preflight")
    )
    (tmp_path / "retained").write_text("keep")
    for destination in (tmp_path, Path.cwd()):
        with pytest.raises(ValueError):
            p.materialize(Path("unused"), destination, None, None)


def test_manifest_retains_continuity_reason_with_digit(tmp_path, monkeypatch):
    args = legacy.inputs()
    args[0]["observations"]["frames"][0]["players"][0]["side"] = "T"
    mock_load(monkeypatch, args)
    report = p.materialize(Path("unused"), tmp_path, None, None, require_observed=True)
    assert p.validate_artifact(Path("unused"), tmp_path, None, None) == report
    assert any(
        cell["fact_status_counts"]["self.native_eye_offset"].get(
            "unavailable:native_visual_c0_actor_continuity_unaccredited", 0
        )
        for cell in report["coverage"]["matrix"]
    )


def test_excluded_cut_retained():
    args = legacy.inputs()
    args[0]["observations"]["frames"][0]["phase"] = "postround"
    rows = list(p.iter_rows(*sources(args)))
    assert len(rows) == len(args[0]["observations"]["frames"])
    assert rows[0]["actors"] == rows[0]["candidates"] == []
    assert rows[0]["provenance"] is None and rows[1]["candidates"]


def test_loader_rechecks_physics_and_references(tmp_path, monkeypatch):
    args = legacy.inputs()
    source = tmp_path / p.k.DATA
    source.write_text(json.dumps(args[3]))
    monkeypatch.setattr(p.k, "load_source", lambda unused: (*args, None))
    seen = []

    def verify(*values):
        seen.append(values)
        return PHYSICS.copy()

    monkeypatch.setattr(p.k, "verify_physics", verify)
    inputs, refs, physics = p.load_inputs(source, Path("verifier"), Path("maps"))
    assert inputs == sources(args) and physics == PHYSICS
    assert refs["visual_source"]["sha256"] == a.digest(source.read_bytes())
    assert seen == [(source, args[3], Path("maps"), Path("verifier"))]


def test_missing_native_pawn_preserves_candidates():
    args = legacy.inputs()
    obs = args[3]["observations"]
    obs["states"].pop(0)
    for state in obs["states"]:
        state["sequence"] -= 1
    for cut in obs["cuts"]:
        cut["state_sequence"] -= 1
        for player in cut["players"]:
            if player["actor_alias"] == ACTOR:
                player.update(
                    pawn_handle=None,
                    pawn_life=None,
                    state_sequence=None,
                    coordinates={},
                )
            else:
                player["state_sequence"] -= 1
        cut["rays"] = [
            ray
            for ray in cut["rays"]
            if 0 not in (ray["observer_index"], ray["subject_index"])
        ]
    assert features(args)["self"]["visual_inputs"] == p.k.native_facts(None)
    assert [
        x["candidate_id"]
        for row in p.iter_rows(*sources(args))
        for x in row["candidates"]
    ] == [x["candidate_id"] for _, pairs in c.iter_census(args[0]) for x in pairs]


@pytest.mark.parametrize("key", ["frame", "clock_order", "source_sequence", "epoch"])
def test_timing_common_cut_cannot_be_substituted(key):
    args = legacy.inputs()
    args[2]["observations"]["cuts"][2][key] += 1
    with pytest.raises((ValueError, ValidationError)):
        features(args)


def test_native_life_substitution_within_same_support_life_is_rejected():
    args = legacy.inputs()
    obs = args[3]["observations"]
    state = copy.deepcopy(obs["states"][0])
    state.update(
        sequence=len(obs["states"]) + 1,
        first_cut=3,
        entity_life="support-visual-pawn-999999",
        created_frame=108,
    )
    for name, value in state["fields"].items():
        sequence = len(obs["deliveries"]) + 1
        obs["deliveries"].append(
            {
                "sequence": sequence,
                "frame": 108,
                "tick": 208,
                "entity_life": state["entity_life"],
                "kind": "created",
                "property": name,
                "previous": legacy.scalar(declared=value["declared_type"]),
                "value": copy.deepcopy(value),
            }
        )
        state["field_sequences"][name] = sequence
    obs["states"].append(state)
    for cut in obs["cuts"][2:]:
        cut["state_sequence"] = len(obs["states"])
        cut["delivery_sequence"] = len(obs["deliveries"])
        cut["players"][0]["state_sequence"] = state["sequence"]
    with pytest.raises(ValueError, match="substituted"):
        features(args)
