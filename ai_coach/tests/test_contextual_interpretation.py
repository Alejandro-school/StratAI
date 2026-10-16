"""Causal invariance, positive signals, ambiguity, interruptions and real IO guards."""

import json
from copy import deepcopy

import pytest

from ai_coach.interpretation import (
    coaching,
    context,
    contextual_io,
    detector,
    evaluation,
    spatial_reference,
    temporal,
)
from ai_coach.tests.test_support_anchors import ACTOR, fixture, player
from ai_coach.tests.test_support_c4 import ownership
from ai_coach.tests.test_support_objective import objective
from ai_coach.tests.test_support_physical import physical

UTILITY_ACTOR = "support-actor-" + "4" * 20


def inputs(n=6, utility=False):
    source = fixture(n)
    for i, frame in enumerate(source["observations"]["frames"]):
        for actor in frame["players"]:
            actor["side"] = "T" if actor["side"] == "CT" else "CT"
        frame["players"][0]["position"]["x"] = i * 2
        frame["players"][1]["position"]["x"] = 10 + i * 2
        if utility:
            frame["players"].append(player(UTILITY_ACTOR, 4, -10, "T"))
    obj = objective(source)
    phy = physical(source)
    if utility:
        for frame in phy["observations"]["frames"]:
            frame["players"][-1]["context"]["facts"]["active_weapon"]["value"] = (
                "Smoke Grenade"
            )
    return source, {
        "physical_source": phy,
        "objective_source": obj,
        "c4_source": ownership(source, obj),
    }


def map_reference():
    package = {
        "schema_id": spatial_reference.SCHEMA_ID,
        "map_name": "de_mirage",
        "source": {
            "kind": "local_vmap_env_cs_place_origin",
            "artifact": "fixture.vmap",
            "sha256": "c" * 64,
            "bytes": 1,
        },
        "scope": "development_reference",
        "build_compatibility": "unverified",
        "regions": [],
        "physical_route_accredited": False,
        "training_allowed": False,
        "landmarks": [
            {
                "id": label,
                "source_label": "Bombsite" + label,
                "label": label,
                "position": {"x": x, "y": 0, "z": 0},
                "kind": "site_landmark",
            }
            for label, x in (("A", -100), ("B", 100))
        ],
    }
    package["reference_id"] = temporal.digest(package)
    return package


def episode(source, sources, end=5, start=1):
    return context.from_sources(
        source, side="T", start_sequence=start, end_sequence=end, **sources
    )


def infer(value, package=None, previous=None):
    return detector.interpret(
        value,
        map_name="de_mirage",
        map_reference=package or map_reference(),
        previous=previous,
    )


def test_full_sequence_and_resources_are_preserved_without_readiness_or_zone_claims():
    source, sources = inputs(utility=True)
    value = episode(source, sources)
    assert len(value["samples"]) == 5
    assert [r["cut"]["sequence"] for r in value["samples"]] == list(range(1, 6))
    assert len(context.trajectories(value)[ACTOR]) == 5
    reading = infer(value)
    assert reading["status"] == "compatible"
    assert [h["destination"] for h in reading["hypotheses"]] == ["B"]
    roles = reading["hypotheses"][0]["possible_functions"]
    assert any(
        r["actor_alias"] == UTILITY_ACTOR
        and r["function"] == "possible_utility_preparation"
        for r in roles
    )
    assert reading["recommendations"] == [] and reading["comparison_allowed"] is False
    assert coaching.assess_readiness(reading)["missions"] == []
    assert all(
        sample["actors"][0]["physical"]["ammo_in_magazine"]["value"] == 0
        for sample in value["samples"]
    )


@pytest.mark.parametrize("mutation", ["future", "enemy", "answers", "ammo"])
def test_plan_is_invariant_to_future_hidden_opponents_answers_and_ammunition(mutation):
    source, sources = inputs()
    expected = infer(episode(source, sources))
    if mutation == "future":
        source["observations"]["frames"][5:] = [None]
        for key in ("physical_source", "objective_source"):
            sources[key]["observations"]["frames"][5:] = [None]
        sources["c4_source"]["observations"]["cuts"][5:] = [None]
        sources["c4_source"]["observations"]["states"][5:] = [None]
    elif mutation == "enemy":
        for row in source["observations"]["frames"]:
            row["players"][2]["position"] = {"x": float("nan")}
    elif mutation == "answers":
        source.update(expert_answer="enter A", case="K2", winner="CT")
        for row in source["observations"]["frames"]:
            row["expert_plan"] = "fake"
    else:
        for row in sources["physical_source"]["observations"]["frames"]:
            for actor in row["players"]:
                actor["context"]["raw_clip"] = 21
                actor["context"]["facts"]["ammo_in_magazine"]["value"] = 20
    actual = infer(episode(source, sources))
    if mutation == "ammo":
        assert actual["hypotheses"] == expected["hypotheses"]
    else:
        assert actual == expected


def test_pauses_refresh_from_trajectory_but_retreat_and_lost_context_do_not_persist():
    source, sources = inputs()
    prior = infer(episode(source, sources, end=4))
    for actor in source["observations"]["frames"][4]["players"][:2]:
        actor["position"]["x"] -= 2
    paused = infer(episode(source, sources), previous=prior)
    assert paused["update"]["kind"] == "maintained"
    source["observations"]["frames"][4]["players"][0]["position"]["x"] = -10
    retreat = infer(episode(source, sources), previous=prior)
    assert all(h["destination"] != "B" for h in retreat["hypotheses"])
    lost = infer(episode(source, sources, start=5), previous=prior)
    assert (
        lost["status"] == "abstained"
        and lost["update"]["kind"] == "prior_context_unavailable"
    )


@pytest.mark.parametrize("change", ["gap", "life", "round", "side", "missing", "dead"])
def test_continuity_breaks_block_using_earlier_carrier_movement(change):
    source, sources = inputs()
    row = source["observations"]["frames"][4]
    if change == "gap":
        # Test the boundary at the derived context layer; native joins keep their own clock validation.
        value = episode(source, sources)
        value["samples"][-1]["continuity"] = "unavailable"
        value["context_id"] = temporal.digest(
            {k: v for k, v in value.items() if k != "context_id"}
        )
    else:
        if change == "life":
            row["players"][0]["pawn_life"] = "support-pawn-000099"
        elif change == "round":
            row["instance"] += 1
        elif change == "side":
            row["players"][0]["side"] = "CT"
        elif change == "missing":
            row["players"].pop(0)
        else:
            row["players"][0]["alive"] = False
        # Basic trajectory is independent of absent physical/C4 joins.
        value = episode(source, {})
    paths = context.trajectories(value)
    assert ACTOR not in paths or len(paths[ACTOR]) <= 1
    assert infer(value)["status"] == "abstained"


def test_multiple_compatible_destinations_are_not_ranked_as_certainty():
    source, sources = inputs()
    package = map_reference()
    package["landmarks"][0]["position"]["x"] = 200
    package["reference_id"] = temporal.digest(
        {k: v for k, v in package.items() if k != "reference_id"}
    )
    reading = infer(episode(source, sources), package)
    assert reading["status"] == "ambiguous" and len(reading["hypotheses"]) == 2


def test_missing_map_keeps_destination_unknown_and_nearest_seed_never_names_zone():
    source, sources = inputs()
    value = episode(source, sources)
    reading = detector.interpret(value, map_name="de_mirage")
    assert (
        reading["status"] == "abstained"
        and "missing_site_reference" in reading["abstention_reasons"]
    )
    assert (
        spatial_reference.resolve(
            {"x": 100, "y": 0, "z": 0}, map_reference(), map_name="de_mirage"
        )["zone"]
        is None
    )
    with pytest.raises(ValueError, match="map reference mismatch"):
        spatial_reference.resolve(
            {"x": 0, "y": 0, "z": 0}, map_reference(), map_name="de_nuke"
        )


def test_map_extractor_keeps_native_source_and_rejects_unknown_transform(tmp_path):
    vmap = '"CMapEntity"\n{\n\t"classname" "string" "env_cs_place"\n\t"place_name" "string" "BombsiteB"\n\t"origin" "vector3" "1 2 3"\n\t"angles" "qangle" "0 0 0"\n\t"scales" "vector3" "1 1 1"\n\t"nodeID" "int" "9"\n}\n'
    path = tmp_path / "de_mirage_texto.vmap"
    path.write_text(vmap, encoding="utf-8")
    package = spatial_reference.extract_vmap(path, map_name="de_mirage")
    assert package["landmarks"][0]["label"] == "B" and package["regions"] == []
    path.write_text(vmap.replace('"0 0 0"', '"0 90 0"'), encoding="utf-8")
    with pytest.raises(ValueError, match="transform"):
        spatial_reference.extract_vmap(path, map_name="de_mirage")


def test_context_chain_and_provenance_cannot_be_tampered_or_joined_across_sides():
    source, sources = inputs()
    rows = list(temporal.iter_records(source, side="T", through_sequence=5, **sources))
    with pytest.raises(ValueError, match="missing context record"):
        context.interval([rows[0], *rows[2:]], start_sequence=1, end_sequence=5)
    value = episode(source, sources)
    value["samples"][0]["actors"][0]["references"][0]["availability_tick"] += 1
    value["context_id"] = temporal.digest(
        {k: v for k, v in value.items() if k != "context_id"}
    )
    with pytest.raises(ValueError, match="provenance"):
        context.validate(value)


def test_evaluation_without_labels_never_reports_accuracy_or_independence():
    source, sources = inputs()
    reading = infer(episode(source, sources))
    match = reading["source_sha256"]
    cases = [
        {"source_sha256": match, "side": "T", "start_sequence": 1, "end_sequence": 5}
    ]
    manifest = evaluation.freeze(
        cases,
        development_matches=[match],
        detector_configuration=detector.CONFIGURATION,
        exposure={match: "development_known"},
    )
    id_ = manifest["cases"][0]["episode_id"]
    result = evaluation.evaluate(manifest, {id_: reading}, [])
    assert result["agreement"] is None and result["counts"]["unreviewed"] == 1
    assert result["independent_generalization_accredited"] is False
    review = {
        "episode_id": id_,
        "reviewer": "expert-local",
        "blind": False,
        "status": "compatible",
        "acceptable_destinations": ["B"],
        "reason": "Own movement with the objective",
    }
    assert (
        evaluation.evaluate(manifest, {id_: reading}, [review])["counts"][
            "compatible_readings"
        ]
        == 1
    )
    with pytest.raises(ValueError, match="duplicate"):
        evaluation.evaluate(manifest, {id_: reading}, [review, review])
    changed = deepcopy(reading)
    changed["interval"]["start_sequence"] = 2
    with pytest.raises(ValueError, match="case mismatch"):
        evaluation.evaluate(manifest, {id_: changed}, [])


def test_output_guard_precedes_source_reads_and_roundtrip_recomputes(
    tmp_path, monkeypatch
):
    source, sources = inputs(utility=True)
    monkeypatch.setattr(
        contextual_io,
        "load_inputs",
        lambda *_: (source, sources, {}, "de_mirage", map_reference()),
    )
    manifest = contextual_io.materialize(
        tmp_path, tmp_path, side="T", intervals=[(1, 4), (1, 5)]
    )
    assert contextual_io.validate(tmp_path, tmp_path) == manifest
    with pytest.raises(ValueError, match="empty"):
        contextual_io.materialize(tmp_path, tmp_path, side="T", intervals=[(1, 5)])
    rows_path = tmp_path / "interpretations.jsonl"
    lines = rows_path.read_text(encoding="utf-8").splitlines()
    row = json.loads(lines[0])
    row["hypotheses"][0]["destination"] = "A"
    lines[0] = json.dumps(row)
    rows_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    manifest["artifacts"][1] = contextual_io.physical.file_reference(rows_path)
    (tmp_path / contextual_io.MANIFEST).write_text(
        json.dumps(manifest), encoding="utf-8"
    )
    with pytest.raises(ValueError, match="recomputation"):
        contextual_io.validate(tmp_path, tmp_path)


def test_non_temp_and_invalid_intervals_do_not_read_sources(tmp_path, monkeypatch):
    monkeypatch.setattr(
        contextual_io, "load_inputs", lambda *_: pytest.fail("source read before guard")
    )
    with pytest.raises(ValueError):
        contextual_io.materialize(
            tmp_path, contextual_io.Path(__file__).parent, side="T", intervals=[(1, 5)]
        )
    for bounds in ([], [(2, 1)], [(1, 5), (1, 4)], [(True, 4)]):
        with pytest.raises(ValueError):
            contextual_io.materialize(tmp_path, tmp_path, side="T", intervals=bounds)
