"""C0 design cases, causal boundary and adversarial artifact verification."""

import copy
import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import preparation as p
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF, fixture


def data_fixture(n=35):
    data = fixture(n)
    for cut in data["observations"]["frames"]:
        for player in cut["players"]:
            player["position"] = {"x": 0, "y": 0, "z": 0}
    return data


def field(frames, i, player, key, value):
    frames[i]["players"][player][key] = value


DESIGN_CASES = [
    ("native_zero_stationary", lambda fs: None, 2, "no_sampled_displacement", 2),
    ("first_cut_without_history", lambda fs: None, 2, "no_sampled_displacement", 0),
    (
        "future_movement",
        lambda fs: [
            field(fs, i, 0, "position", {"x": 32, "y": 0, "z": 0}) for i in range(3, 35)
        ],
        2,
        "sampled_displacement",
        2,
    ),
    (
        "movement_already_in_progress",
        lambda fs: [
            field(fs, i, 0, "position", {"x": i, "y": 0, "z": 0}) for i in range(35)
        ],
        2,
        "sampled_displacement",
        2,
    ),
    (
        "far_separation_not_thresholded",
        lambda fs: [
            field(fs, i, 1, "position", {"x": 4096, "y": 0, "z": 0}) for i in range(35)
        ],
        2,
        None,
        2,
    ),
    ("current_not_alive", lambda fs: field(fs, 2, 0, "alive", False), 0, None, 2),
    (
        "current_disconnected",
        lambda fs: field(fs, 2, 0, "connected", False),
        0,
        None,
        2,
    ),
    (
        "current_missing_position",
        lambda fs: field(fs, 2, 0, "position", None),
        0,
        None,
        2,
    ),
    (
        "current_nonfinite_position",
        lambda fs: field(fs, 2, 0, "position", {"x": float("nan"), "y": 0, "z": 0}),
        0,
        None,
        2,
    ),
    (
        "current_bool_coordinate",
        lambda fs: field(fs, 2, 0, "position", {"x": True, "y": 0, "z": 0}),
        0,
        None,
        2,
    ),
    ("current_alive_unknown", lambda fs: field(fs, 2, 0, "alive", None), 0, None, 2),
    (
        "current_identity_unknown",
        lambda fs: field(fs, 2, 0, "identity_status", "unavailable"),
        0,
        None,
        2,
    ),
    (
        "shared_pawn_life",
        lambda fs: field(fs, 2, 1, "pawn_life", "support-pawn-000001"),
        0,
        None,
        2,
    ),
    (
        "shared_full_handle",
        lambda fs: field(fs, 2, 1, "pawn_handle", 16385),
        0,
        None,
        2,
    ),
    (
        "duplicate_actor_alias",
        lambda fs: fs[2]["players"].append(copy.deepcopy(fs[2]["players"][0])),
        0,
        None,
        2,
    ),
    ("no_usable_ally", lambda fs: fs[2]["players"].pop(1), 0, None, 2),
    ("current_postround", lambda fs: fs[2].update(phase="postround"), 0, None, 2),
    ("current_bad_clock", lambda fs: fs[2].update(tick_rate=0), 0, None, 2),
    ("current_bad_sample_step", lambda fs: fs[2].update(sample_step=0), 0, None, 2),
    (
        "new_epoch_current",
        lambda fs: [fs[i].update(epoch=1) for i in range(2, 35)],
        2,
        None,
        2,
    ),
    (
        "new_pawn_current",
        lambda fs: [
            field(fs, i, 0, "pawn_life", "support-pawn-999999") for i in range(2, 35)
        ],
        2,
        None,
        2,
    ),
    (
        "same_tick_distinct_frames",
        lambda fs: fs[2].update(tick=fs[1]["tick"]),
        2,
        None,
        2,
    ),
    (
        "future_disconnection",
        lambda fs: field(fs, 8, 0, "connected", False),
        2,
        "censored:disconnected",
        2,
    ),
    (
        "future_not_alive",
        lambda fs: field(fs, 8, 0, "alive", False),
        2,
        "censored:not_alive",
        2,
    ),
    (
        "future_life_change",
        lambda fs: field(fs, 8, 0, "pawn_life", "support-pawn-999999"),
        2,
        "censored:identity_unavailable_or_changed",
        2,
    ),
    (
        "future_side_change",
        lambda fs: field(fs, 8, 0, "side", "T"),
        2,
        "censored:identity_unavailable_or_changed",
        2,
    ),
    (
        "future_gap",
        lambda fs: fs[8].update(tick=1033),
        2,
        "censored:horizon_gap_or_boundary",
        2,
    ),
    (
        "future_phase_boundary",
        lambda fs: fs[8].update(phase="postround"),
        2,
        "censored:horizon_gap_or_boundary",
        2,
    ),
    (
        "missing_exact_horizon",
        lambda fs: fs.pop(),
        2,
        "censored:missing_exact_horizon_sample",
        2,
    ),
    (
        "declared_freeze_does_not_promote_or_remove",
        lambda fs: fs[2].update(native_freeze=True),
        2,
        None,
        2,
    ),
    (
        "possible_hold_function_does_not_label_intent",
        lambda fs: fs[2].update(context_declared="protect_plant"),
        2,
        None,
        2,
    ),
]


@pytest.mark.parametrize(
    "name,mutate,count,expected_future,at",
    DESIGN_CASES,
    ids=[c[0] for c in DESIGN_CASES],
)
def test_frozen_design_cases(name, mutate, count, expected_future, at):
    data = data_fixture()
    frames = data["observations"]["frames"]
    mutate(frames)
    assert len(p.select_pairs(frames[at])) == count, name
    if expected_future:
        result = p.execution_descriptor(
            frames, [p.roster(f) for f in frames], at, ACTOR
        )
        assert result["status"] == expected_future, name


def test_prefix_features_never_consult_malformed_future_or_enemy_geometry():
    data = data_fixture()
    cut = data["observations"]["frames"][2]
    identifier = p.candidate_id(data["source_alias"], cut, ACTOR, ALLY)
    args = (identifier, ACTOR, ALLY, 3, cut["tick"], cut["frame"])
    expected = p.initial_features(data, *args)
    changed = copy.deepcopy(data)
    changed["observations"]["frames"][3:] = [None, {"malformed_future": True}]
    for row in changed["observations"]["frames"][:3]:
        row["players"][2]["position"]["x"] = -999999
        row["players"].reverse()
    assert p.initial_features(changed, *args) == expected
    expected["self"]["position"]["x"] = 999
    assert p.initial_features(data, *args)["self"]["position"]["x"] == 0
    assert set(expected) == {"self", "ally"}
    assert set(expected["self"]) == {
        "position",
        "alive",
        "availability_tick",
        "availability_frame",
    }


@pytest.mark.parametrize(
    "index,value",
    [(0, "bad-id"), (1, ENEMY), (2, ENEMY), (3, 0), (3, True), (4, 0), (5, 0)],
)
def test_initial_access_requires_exact_candidate_pair_delivery_and_tick(index, value):
    data = data_fixture()
    cut = data["observations"]["frames"][0]
    args = [
        p.candidate_id(data["source_alias"], cut, ACTOR, ALLY),
        ACTOR,
        ALLY,
        1,
        cut["tick"],
        cut["frame"],
    ]
    args[index] = value
    with pytest.raises(ValueError, match="no exact"):
        p.initial_features(data, *args)


def test_future_branches_leave_initial_rows_and_ids_unchanged():
    data = data_fixture()
    changed = copy.deepcopy(data)
    for cut in changed["observations"]["frames"][3:]:
        cut["players"][0]["position"]["x"] = 23
    before = list(p.iter_census(data))[:3]
    after = list(p.iter_census(changed))[:3]
    for (cut_a, pairs_a), (cut_b, pairs_b) in zip(before, after):
        assert cut_a == cut_b
        for a, b in zip(pairs_a, pairs_b):
            assert {k: v for k, v in a.items() if k != "execution"} == {
                k: v for k, v in b.items() if k != "execution"
            }
    assert before[0][1][0]["execution"]["self"]["sampled_displacement"] is False
    assert after[0][1][0]["execution"]["self"]["sampled_displacement"] is True


def test_same_tick_distinct_frames_have_distinct_ids_without_history_gate():
    data = data_fixture(2)
    frames = data["observations"]["frames"]
    frames[1]["tick"] = frames[0]["tick"]
    rows = list(p.iter_census(data))
    assert len(rows[0][1]) == len(rows[1][1]) == 2
    assert rows[0][1][0]["candidate_id"] != rows[1][1][0]["candidate_id"]
    assert (
        rows[0][1][0]["execution"]["self"]["status"]
        == "censored:missing_exact_horizon_sample"
    )


def test_coverage_counts_missing_allies_dead_and_postround_separately():
    data = data_fixture(4)
    frames = data["observations"]["frames"]
    frames[1]["players"][1]["alive"] = False
    frames[2]["players"].pop(1)
    frames[3]["phase"] = "postround"
    coverage = p.Coverage()
    for cut, pairs in p.iter_census(data):
        coverage.add(cut, pairs)
    result = coverage.result()
    assert result["counts"] == {
        "cuts_total": 4,
        "cuts_admissible": 3,
        "cuts_excluded_phase": 1,
        "cuts_excluded_clock": 0,
        "roster_rows_total": 11,
        "actor_alias_cuts": 8,
        "actor_usable": 7,
        "selected_actor_cuts": 2,
        "pairs_examined": 4,
        "pairs_selected": 2,
        "pairs_excluded_actor": 1,
        "pairs_excluded_ally": 1,
    }
    assert result["actor_unavailable"]["not_alive"] == 1
    assert result["actor_unavailable"]["disconnected"] == 0
    assert len(result["matrix"]) == 2


def test_missing_horizon_precedes_earlier_death_without_erasing_candidate():
    data = data_fixture(5)
    frames = data["observations"]["frames"]
    frames[1]["players"][0]["alive"] = False
    row = next(p.iter_census(data))[1][0]
    assert row["execution"]["self"]["status"] == "censored:missing_exact_horizon_sample"
    assert row["execution"]["self"]["sampled_displacement"] is None


@pytest.mark.parametrize(
    "change",
    [
        "first_failure",
        "same_handle_new_life",
        "removed_actor",
        "shared_pawn",
        "position_absent",
    ],
)
def test_future_censure_reports_first_available_failure(change):
    data = data_fixture()
    frames = data["observations"]["frames"]
    expected = "censored:identity_unavailable_or_changed"
    if change == "first_failure":
        frames[4]["players"][0]["connected"] = False
        frames[5]["players"][0]["alive"] = False
        expected = "censored:disconnected"
    elif change == "same_handle_new_life":
        frames[4]["players"][0]["pawn_life"] = "support-pawn-999999"
    elif change == "removed_actor":
        frames[4]["players"].pop(0)
    elif change == "shared_pawn":
        frames[4]["players"][1]["pawn_handle"] = frames[4]["players"][0]["pawn_handle"]
        expected = "censored:shared_pawn_reference"
    else:
        frames[4]["players"][0]["position"] = None
        expected = "censored:position_unavailable"
    descriptor = next(p.iter_census(data))[1][0]["execution"]["self"]
    assert descriptor["status"] == expected
    assert descriptor["availability_sequence"] == 5
    assert descriptor["sampled_displacement"] is None


@pytest.fixture
def materialized(tmp_path, monkeypatch):
    data = data_fixture()
    monkeypatch.setattr(
        p.anchors, "load_source", lambda path: (copy.deepcopy(data), REF.copy(), None)
    )
    result = p.materialize(
        Path("unused") / p.anchors.DATA, tmp_path, require_observed=True
    )
    return tmp_path, result


def test_roundtrip_streaming_reader_recomputes_complete_artifact(materialized):
    root, result = materialized
    assert p.validate_artifact(Path("unused"), root) == result
    assert result["coverage"]["counts"]["pairs_selected"] == 70
    assert result["coverage"]["counts"]["cuts_total"] == 35
    assert (root / p.PROTOCOL).read_bytes() == (
        p.anchors.CONTRACTS / p.PROTOCOL
    ).read_bytes()
    assert result["eligible_decisions"] == result["physical_entries_accredited"] == 0


@pytest.mark.parametrize(
    "mutation",
    [
        "position",
        "id",
        "execution",
        "unknown",
        "duplicate",
        "missing",
        "reverse",
        "cut_actor",
        "coverage",
        "hash",
        "protocol",
    ],
)
def test_reader_rejects_tampering_missing_and_extra_rows(materialized, mutation):
    root, _ = materialized
    if mutation in ("coverage", "hash"):
        path = root / p.MANIFEST
        data = json.loads(path.read_bytes())
        if mutation == "coverage":
            data["coverage"]["counts"]["pairs_selected"] += 1
        else:
            data["artifacts"][0]["sha256"] = "0" * 64
        path.write_bytes(p.encoded_row(data))
    elif mutation == "protocol":
        path = root / p.PROTOCOL
        data = json.loads(path.read_bytes())
        data["future_window_seconds"] = 3
        path.write_bytes(p.encoded_row(data))
    else:
        path = root / (p.CUTS if mutation == "cut_actor" else p.PAIRS)
        lines = path.read_bytes().splitlines(keepends=True)
        row = json.loads(lines[0])
        if mutation == "position":
            row["initial"]["self"]["position"]["x"] = 10
        elif mutation == "id":
            row["candidate_id"] = "support-preparation-" + "0" * 64
        elif mutation == "execution":
            row["execution"]["self"].update(
                status="sampled_displacement", sampled_displacement=True
            )
        elif mutation == "unknown":
            row["initial"]["enemy_position"] = 10
        elif mutation == "cut_actor":
            row["actors"][0]["reason"] = "not_alive"
        lines[0] = p.encoded_row(row)
        if mutation == "duplicate":
            lines.append(lines[0])
        elif mutation == "missing":
            lines.pop()
        elif mutation == "reverse":
            lines.reverse()
        path.write_bytes(b"".join(lines))
    with pytest.raises((ValueError, ValidationError)):
        p.validate_artifact(Path("unused"), root)


def test_materializer_rejects_nonempty_before_loading_source(tmp_path, monkeypatch):
    (tmp_path / "retained.txt").write_text("retained", encoding="utf-8")

    def forbidden(path):
        pytest.fail("source must not load before output preflight")

    monkeypatch.setattr(p.anchors, "load_source", forbidden)
    with pytest.raises(ValueError, match="empty"):
        p.materialize(Path("does-not-exist"), tmp_path)
    assert (tmp_path / "retained.txt").read_text() == "retained"


def test_materializer_rejects_repository_destination_before_source(monkeypatch):
    with pytest.raises(ValueError, match="TEMP"):
        p.materialize(Path("does-not-exist"), Path.cwd())


def test_source_schema_remains_strict_while_selector_excludes_invalid_fixture_states():
    data = data_fixture()
    data["observations"]["frames"][0]["players"][0]["position"]["x"] = True
    assert p.select_pairs(data["observations"]["frames"][0]) == []
    with pytest.raises(ValueError, match="schema"):
        p.anchors.validate_source(data)


@pytest.mark.parametrize("handle", [-1, 2**32, 16383, True, 16385.0])
def test_current_invalid_native_handle_never_selects(handle):
    data = data_fixture(1)
    cut = data["observations"]["frames"][0]
    cut["players"][0]["pawn_handle"] = handle
    assert p.select_pairs(cut) == []


def test_frozen_case_names_and_expected_results_match_documentary_contract():
    # tests/ -> ai_coach/ -> repository
    path = (
        Path(__file__).parents[2]
        / "docs/references/support-preparation-selection-cases-v1.json"
    )
    documented = json.loads(path.read_bytes())["cases"]
    actual = {
        name: (count, expected, at) for name, _, count, expected, at in DESIGN_CASES
    }
    assert len(actual) == len(documented) == 31
    for case in documented:
        assert actual[case["name"]] == (
            case["expected_selected_pairs"],
            case.get("expected_actor_execution"),
            case["current_zero_based_index"],
        )
