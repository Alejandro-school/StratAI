"""Causal, attribution, continuity and hostile-input tests for ICT-I1.A."""

from copy import deepcopy

import pytest
from jsonschema import ValidationError

from ai_coach.interpretation import temporal as t
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, fixture
from ai_coach.tests.test_support_physical import physical
from ai_coach.tests.test_support_objective import objective
from ai_coach.tests.test_support_c4 import ownership
from ai_coach.interpretation import ledger_io


def inputs(n=5):
    source = fixture(n)
    obj = objective(source)
    return source, dict(
        physical_source=physical(source),
        objective_source=obj,
        c4_source=ownership(source, obj),
    )


def records(source, sources=None, sequence=3):
    return list(
        t.iter_records(source, side="CT", through_sequence=sequence, **(sources or {}))
    )


def test_full_team_description_resources_provenance_and_immutable_link():
    source, sources = inputs()
    original = deepcopy((source, sources))
    rows = records(source, sources)
    assert (source, sources) == original
    for row in rows:
        t.validate_record(row)
        assert [a["actor_alias"] for a in row["actors"]] == [ACTOR, ALLY]
        assert not row["recommendations"] and not row["training_allowed"]
        assert all(h["status"] == "unavailable" for h in row["hypotheses"])
    assert rows[2]["previous_record_id"] == rows[1]["record_id"]
    actor, ally = rows[-1]["actors"]
    assert actor["has_c4"]["value"] is True
    assert ally["has_c4"]["value"] is False
    assert actor["physical"]["ammo_in_magazine"]["value"] == 0
    assert actor["physical"]["reloading"]["value"] is False
    assert ally["sampled_displacement"]["value"] == {"x": 2, "y": 0, "z": 0}
    assert (
        rows[0]["actors"][0]["sampled_displacement"]["reason"] == "no_previous_sample"
    )


@pytest.mark.parametrize(
    "mutation", ["future", "hidden", "human", "execution", "file_hashes"]
)
def test_same_prefix_is_invariant_to_excluded_information(mutation):
    source, sources = inputs()
    expected = records(source, sources)
    if mutation == "future":
        source["observations"]["frames"][3:] = [None]
        for name in ("physical_source", "objective_source"):
            sources[name]["observations"]["frames"][3:] = [None]
        sources["c4_source"]["observations"]["cuts"][3:] = [None]
        sources["c4_source"]["observations"]["states"][3:] = [None]
    elif mutation == "hidden":
        for f in source["observations"]["frames"]:
            enemy = f["players"][-1]
            enemy.update(position={"x": float("nan")}, alive="unknown", pawn_handle=-1)
        for f in sources["physical_source"]["observations"]["frames"]:
            f["players"][-1]["context"] = "hidden poison"
        for f in sources["c4_source"]["observations"]["states"]:
            f["players"][-1]["slots"] = ["hidden poison"]
    elif mutation in ("human", "execution"):
        source["expert_answer"] = "push B; fake; rotate"
        for f in source["observations"]["frames"]:
            f.update(
                winner="T", future_death=True, title="F01-K2", execution={"end": "A"}
            )
            for actor in f["players"]:
                actor.update(role="information", known_enemy_count=3, zone="B")
    else:
        source["base_manifest"]["sha256"] = "d" * 64
        for name in sources:
            if "support_source" in sources[name]:
                sources[name]["support_source"]["sha256"] = "e" * 64
    assert records(source, sources) == expected
    assert records(source, sources, 2) == expected[:2]


@pytest.mark.parametrize(
    "change,reason",
    [
        ("gap", "sample_gap_or_boundary"),
        ("round", "sample_gap_or_boundary"),
        ("epoch", "sample_gap_or_boundary"),
        ("life", "identity_changed"),
        ("absent", "previous_actor_unavailable"),
        ("shared", "previous_actor_unavailable"),
        ("dead", "not_alive"),
        ("position", "position_unavailable"),
    ],
)
def test_discontinuity_never_becomes_a_tactical_transition(change, reason):
    source = fixture(3)
    prev, current = source["observations"]["frames"][1:]
    if change == "gap":
        current["tick"] += 4
    if change == "round":
        current["instance"] += 1
    if change == "epoch":
        current.update(epoch=1, tick=1)
    if change == "life":
        current["players"][1]["pawn_life"] = "support-pawn-000099"
    if change == "absent":
        prev["players"].pop(1)
    if change == "shared":
        prev["players"][0].update(
            pawn_life=prev["players"][1]["pawn_life"],
            pawn_handle=prev["players"][1]["pawn_handle"],
        )
    if change == "dead":
        current["players"][1]["alive"] = False
    if change == "position":
        current["players"][1].update(
            position=None, position_status="native_fields_absent"
        )
    row = records(source)[-1]
    t.validate_record(row)
    assert row["actors"][1]["sampled_displacement"]["reason"] == reason
    assert all(h["status"] == "unavailable" for h in row["hypotheses"])


def test_disappearance_is_explicit_not_a_death():
    source = fixture(3)
    source["observations"]["frames"][-1]["players"].pop(1)
    row = records(source)[-1]
    assert row["missing_previous_actors"] == [ALLY]
    assert row["missing_actor_meaning"] == "no_current_sample_not_a_death_or_retreat"


@pytest.mark.parametrize("criterion", list(t.REQUIREMENTS))
@pytest.mark.parametrize(
    "variant", ["present_label_only", "no_proof", "precondition_failed", "all_team_B"]
)
def test_expert_factors_require_evidence_not_injected_labels(criterion, variant):
    source = fixture(3)
    expected = records(source)
    source["human_hypotheses"] = {criterion: variant}
    for cut in source["observations"]["frames"]:
        cut.update(
            contacts=[1, 2, 3],
            utilities=[1, 2, 3],
            mid_kill=True,
            fake=True,
            ct_rotations=["A"],
            information_player_dead=True,
            all_team_at_B=variant == "all_team_B",
        )
    assert records(source) == expected
    e02 = expected[-1]["hypotheses"][1]
    assert "all_team_at_B_exception" in e02["unmet_requirements"]


@pytest.mark.parametrize(
    "change",
    ["source_identity", "future_reload", "life_handle", "c4_clock", "nan", "order"],
)
def test_incompatible_allowed_sources_are_rejected(change):
    source, sources = inputs()
    if change == "source_identity":
        sources["physical_source"]["source_sha256"] = "f" * 64
    if change == "future_reload":
        sources["physical_source"]["observations"]["frames"][0]["players"][0][
            "context"
        ]["reload_signal_tick"] = 9999
    if change == "life_handle":
        source["observations"]["frames"][1]["players"][0]["pawn_handle"] += 1
    if change == "c4_clock":
        sources["c4_source"]["observations"]["cuts"][1]["tick"] += 1
    if change == "nan":
        source["observations"]["frames"][1]["players"][0]["position"]["x"] = float(
            "nan"
        )
    if change == "order":
        source["observations"]["frames"][1]["sequence"] = 3
    with pytest.raises((ValueError, ValidationError)):
        records(source, sources)


def test_missing_resources_stay_unknown_and_record_tampering_fails():
    row = records(fixture(3))[-1]
    assert row["actors"][0]["has_c4"]["value"] is None
    assert row["actors"][0]["physical"]["ammo_in_magazine"]["value"] is None
    row["actors"][0]["position"]["value"]["x"] = 99
    with pytest.raises(ValueError, match="hash"):
        t.validate_record(row)


def test_future_provenance_cannot_be_rehashed_into_validity():
    row = records(fixture(3))[-1]
    row["actors"][0]["references"][0]["availability_tick"] += 1
    row["record_id"] = t.digest({k: v for k, v in row.items() if k != "record_id"})
    with pytest.raises(ValueError, match="provenance"):
        t.validate_record(row)


def test_consumer_mutation_cannot_rewrite_subsequent_records():
    source = fixture(3)
    expected = records(source)
    stream = t.iter_records(source, side="CT", through_sequence=3)
    row = next(stream)
    row["record_id"] = "0" * 64
    row["hypotheses"][0]["unmet_requirements"].clear()
    row["actors"][0]["position"]["value"]["x"] = 999
    assert list(stream) == expected[1:]


@pytest.mark.parametrize("case", ["non_temp", "not_empty"])
def test_output_guard_runs_before_any_source_read(tmp_path, monkeypatch, case):
    from pathlib import Path

    target = Path(__file__).parent if case == "non_temp" else tmp_path
    if case == "not_empty":
        (target / "retained.txt").write_text("keep")
    monkeypatch.setattr(
        ledger_io, "load_sources", lambda _: pytest.fail("read before guard")
    )
    with pytest.raises(ValueError):
        ledger_io.materialize(tmp_path / "absent", target)


def test_roundtrip_reader_recomputes_and_rejects_tampering(tmp_path, monkeypatch):
    source, sources = inputs()
    monkeypatch.setattr(
        ledger_io, "load_sources", lambda _: (source, sources, {}, None)
    )
    manifest = ledger_io.materialize(tmp_path, tmp_path, through_sequence=3)
    assert ledger_io.validate(tmp_path, tmp_path) == manifest
    manifest["training_allowed"] = True
    (tmp_path / ledger_io.MANIFEST).write_text(t.json.dumps(manifest))
    with pytest.raises(ValueError, match="manifest"):
        ledger_io.validate(tmp_path, tmp_path)
