from __future__ import annotations

import copy
import gzip
import hashlib
import io
import json
import sys
from dataclasses import dataclass
from pathlib import Path

import pytest

from ai_coach.contracts.validate_full_demo import (
    NAVIGATION_REPORT_FILENAME,
    NAVIGATION_ROWS_FILENAME,
    VISUAL_AUDIT_FILENAME,
    VISUAL_REPORT_FILENAME,
    FullDemoValidationError,
    main,
    validate_full_demo_reports,
)
from ai_coach.tests import test_navigation_smoke_validation as navigation_fixture
from ai_coach.tests import test_visual_contact_smoke_validation as visual_fixture


@dataclass
class FullDemoFixture:
    root: Path
    maps_root: Path
    navigation_path: Path
    visual_path: Path
    navigation: dict
    visual: dict
    navigation_rows: list[dict]
    visual_records: list[dict]


def _jsonl(records: list[dict]) -> bytes:
    return b"".join(
        json.dumps(record, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
        + b"\n"
        for record in records
    )


def _gzip(payload: bytes) -> bytes:
    output = io.BytesIO()
    with gzip.GzipFile(filename="", mode="wb", fileobj=output, mtime=0) as writer:
        writer.write(payload)
    return output.getvalue()


def _write_navigation_stream(fixture: FullDemoFixture) -> None:
    logical = _jsonl(fixture.navigation_rows)
    physical = _gzip(logical)
    (fixture.root / NAVIGATION_ROWS_FILENAME).write_bytes(physical)
    artifact = fixture.navigation["rows_artifact"]
    artifact.update(
        {
            "bytes": len(physical),
            "sha256": hashlib.sha256(physical).hexdigest(),
            "row_count": len(fixture.navigation_rows),
            "logical_bytes": len(logical),
            "logical_sha256": hashlib.sha256(logical).hexdigest(),
        }
    )


def _write_visual_stream(fixture: FullDemoFixture) -> None:
    logical = _jsonl(fixture.visual_records)
    physical = _gzip(logical)
    (fixture.root / VISUAL_AUDIT_FILENAME).write_bytes(physical)
    evaluations = sum(
        record["record_type"] == "evaluation" for record in fixture.visual_records
    )
    episodes = sum(
        record["record_type"] == "episode" for record in fixture.visual_records
    )
    fixture.visual["audit"].update(
        {
            "evaluation_records": evaluations,
            "episode_records": episodes,
            "logical_records": len(fixture.visual_records),
            "logical_bytes": len(logical),
            "logical_sha256": hashlib.sha256(logical).hexdigest(),
            "gzip_bytes": len(physical),
            "gzip_sha256": hashlib.sha256(physical).hexdigest(),
        }
    )
    lines = _jsonl(fixture.visual_records).splitlines(keepends=True)
    for round_summary in fixture.visual["rounds"]:
        round_number = round_summary["round_number"]
        round_payload = b"".join(
            line
            for record, line in zip(fixture.visual_records, lines, strict=True)
            if record["round"] == round_number
        )
        round_summary["audit_logical_records"] = sum(
            record["round"] == round_number for record in fixture.visual_records
        )
        round_summary["audit_logical_sha256"] = hashlib.sha256(
            round_payload
        ).hexdigest()


def _write_reports(fixture: FullDemoFixture) -> None:
    fixture.navigation_path.write_text(
        json.dumps(fixture.navigation, separators=(",", ":")) + "\n",
        encoding="utf-8",
    )
    fixture.visual_path.write_text(
        json.dumps(fixture.visual, separators=(",", ":")) + "\n",
        encoding="utf-8",
    )


def _distribution(values: list[float]) -> dict:
    ordered = sorted(values)
    return {
        "count": len(values),
        "p50": ordered[(len(ordered) + 1) // 2 - 1] if ordered else None,
        "p95": ordered[-1] if ordered else None,
        "max": ordered[-1] if ordered else None,
    }


def _navigation_report(maps_root: Path) -> tuple[dict, list[dict]]:
    smoke = navigation_fixture._valid_report()
    rows = copy.deepcopy(smoke["rows"])
    nav_payload = b"real nav fixture"
    nav_path = maps_root / "de_mirage" / "de_mirage.nav"
    nav_path.parent.mkdir(parents=True, exist_ok=True)
    nav_path.write_bytes(nav_payload)
    nav_sha = hashlib.sha256(nav_payload).hexdigest()
    for row in rows:
        row["provenance"]["graph_asset"]["sha256"] = nav_sha
    active_times = [
        row["internal_timing"]["scenarios"]["active_weapon_uninterrupted_baseline"][
            "value_milliseconds"
        ]
        for row in rows
    ]
    counts = {
        "queries": 2,
        "reachable_queries": 2,
        "disconnected_queries": 0,
        "endpoint_unavailable_queries": 0,
        "optimistic_baseline": {
            "estimated": 2,
            "unavailable": 0,
            "unavailability_reasons": {},
        },
        "active_weapon_baseline": {
            "estimated": 2,
            "unavailable": 0,
            "unavailability_reasons": {},
        },
        "observed_pace_projection": {
            "estimated": 2,
            "unavailable": 0,
            "unavailability_reasons": {},
        },
        "walking_contexts": 1,
        "non_walking_moving_contexts": 1,
        "observed_moving_contexts": 2,
        "observed_stationary_contexts": 0,
        "motion_unavailable_contexts": 0,
        "ducking_contexts": 0,
        "scoped_contexts": 0,
        "knife_contexts": 1,
        "active_weapon_contexts": {"AK-47": 1, "Knife": 1},
        "weapon_baseline_speed_contexts": {"215": 1, "250": 1},
        "abstention_reasons": {},
        "exact_user_facing_time_violations": 0,
        "arrival_prediction_violations": 0,
    }
    distributions = {
        "direct_distance_world_units": _distribution([80, 80]),
        "path_distance_world_units": _distribution([100, 100]),
        "path_to_direct_distance_ratio": _distribution([1.25, 1.25]),
        "observed_horizontal_speed_world_units_per_second": _distribution([100, 100]),
        "weapon_baseline_speed_world_units_per_second": _distribution([215, 250]),
        "optimistic_baseline_milliseconds": _distribution([400, 400]),
        "active_weapon_baseline_milliseconds": _distribution(active_times),
        "observed_pace_projection_milliseconds": _distribution([1000, 1000]),
    }
    return (
        {
            "schema_id": "stratai.observed_navigation_full_demo_sweep@1",
            "status": "passed",
            "source_alias": smoke["source_alias"],
            "source_sha256": smoke["source_sha256"],
            "map_name": "de_mirage",
            "scope": "temporal_full_demo_sweep",
            "selection_contract": (
                "one_deterministic_stress_pair_per_eligible_replay_frame@1"
            ),
            "pair_exhaustive": False,
            "directional_pairs": True,
            "candidate_schema_id": smoke["candidate_schema_id"],
            "candidate_contract_version": smoke["candidate_contract_version"],
            "graph_contract_version": smoke["graph_contract_version"],
            "graph_asset_reference": "de_mirage/de_mirage.nav",
            "graph_asset_sha256": nav_sha,
            "graph_node_count": 10,
            "graph_edge_count": 20,
            "round_count": 1,
            "tick_rate": 64,
            "sample_stride_ticks": 4,
            "sample_interval_milliseconds": 62.5,
            "distribution_contract": "nearest_rank_p50_p95_max@1",
            "timing_policy": {
                "intended_use": "internal_decision_support_only",
                "user_facing_mode": ("qualitative_conditional_with_uncertainty_only"),
                "decision_semantics": "constraint_signal_not_recommendation",
                "estimate_semantics": "scenario_reference_not_arrival_prediction",
                "exact_user_facing_time_allowed": False,
                "arrival_prediction": False,
            },
            "build_identifier": "full-demo-test-build",
            "processing_duration_ms": 20,
            "navigation_evaluation_duration_ms": 10,
            "rows_artifact": {
                "path": NAVIGATION_ROWS_FILENAME,
                "format": "jsonl",
                "compression": "gzip",
                "bytes": 0,
                "sha256": "",
                "row_count": 0,
                "logical_bytes": 0,
                "logical_sha256": "",
                "digest_contract": "sha256_uncompressed_jsonl_bytes@1",
            },
            "summary": {
                "replay_frames": 3,
                "eligible_frames": 2,
                "ineligible_frames": 1,
                "counts": copy.deepcopy(counts),
                "distributions": distributions,
            },
            "rounds": [
                {
                    "round_number": 1,
                    "replay_frames": 3,
                    "eligible_frames": 2,
                    "ineligible_frames": 1,
                    "counts": copy.deepcopy(counts),
                }
            ],
        },
        rows,
    )


def _visual_report(maps_root: Path) -> tuple[dict, list[dict]]:
    smoke = visual_fixture._valid_report(maps_root)
    evaluations = copy.deepcopy(smoke["ledger"]["evaluations"])
    episode = copy.deepcopy(smoke["ledger"]["contacts"][0])
    for evaluation in evaluations:
        if evaluation["join_keys"]["contact_id"] is not None:
            evaluation["join_keys"]["contact_id"] = "contact-000001"
    episode["join_keys"]["contact_id"] = "contact-000001"
    records = [
        {"record_type": "evaluation", "round": 1, "value": evaluation}
        for evaluation in evaluations
    ] + [{"record_type": "episode", "round": 1, "value": episode}]
    metrics = {
        "evaluations": 6,
        "observed_evaluations": 5,
        "unavailable_evaluations": 1,
        "line_of_sight_clear_pair_samples": 4,
        "line_of_sight_blocked_pair_samples": 1,
        "line_of_sight_unavailable_pair_samples": 1,
        "fov_inside_pair_samples": 4,
        "fov_outside_pair_samples": 1,
        "fov_unavailable_pair_samples": 1,
        "smoke_pair_sample_intersections": 1,
        "smoke_clear_pair_samples": 4,
        "smoke_unavailable_pair_samples": 1,
        "active_smoke_pair_sample_evaluations": 1,
        "otherwise_visible_pair_samples_blocked_by_smoke": 1,
        "visual_contact_pair_samples": 2,
        "no_visual_contact_pair_samples": 3,
        "visual_contact_unavailable_pair_samples": 1,
        "input_source_frames": 10,
        "canonical_source_frames": 3,
        "source_frames_with_valid_active_smoke": 1,
        "evaluated_source_frames": 3,
        "evaluated_source_frames_with_active_smoke": 1,
        "evaluated_source_frames_with_smoke_intersection": 1,
        "unique_valid_active_smoke_effects": 1,
        "directional_episodes": 1,
        "closed_directional_episodes": 1,
        "right_censored_directional_episodes": 0,
        "boundaries": {
            "continued": 1,
            "ended": 1,
            "inactive": 2,
            "started": 1,
            "unavailable": 1,
        },
        "abstention_reasons": copy.deepcopy(smoke["summary"]["abstention_reasons"]),
        "raw_steam_id_violations": 0,
    }
    round_summary = {
        "round_number": 1,
        "start_tick": 100,
        "end_tick": 108,
        "metrics": copy.deepcopy(metrics),
        "participants_evaluated": 8,
        "participants_with_contact": 2,
        "unique_directed_pairs_with_contact": 1,
        "unique_unordered_pairs_with_contact": 1,
        "unique_directed_pairs_smoke_intersected": 1,
        "unique_unordered_pairs_smoke_intersected": 1,
        "reacquisition_episodes_within_round": 0,
        "audit_logical_records": 0,
        "audit_logical_sha256": "",
    }
    return (
        {
            "schema_id": ("stratai.observed_visual_contact_full_demo_validation@1"),
            "status": "passed",
            "source_alias": smoke["source_alias"],
            "source_sha256": smoke["source_sha256"],
            "map_name": "de_mirage",
            "candidate_schema_id": smoke["candidate_schema_id"],
            "candidate_contract_version": smoke["candidate_contract_version"],
            "training_allowed": False,
            "build_identifier": "full-demo-test-build",
            "processing_duration_ms": 25,
            "selection": {
                "contract": "all_valid_replay_rounds_per_round_t0@1",
                "round_count": 1,
                "first_round": 1,
                "last_round": 1,
                "minimum_start_tick": 100,
                "maximum_end_tick": 108,
                "input_source_frames": 10,
                "canonical_source_frames": 3,
                "per_round_t0": "round_end_tick",
            },
            "sampling": copy.deepcopy(smoke["ledger"]["sampling"]),
            "geometry_assets": copy.deepcopy(smoke["geometry_assets"]),
            "summary": {
                "metrics": copy.deepcopy(metrics),
                "participant_aliases": 8,
                "participants_evaluated": 8,
                "participants_with_contact": 2,
                "unique_match_directed_pairs_with_contact": 1,
                "unique_match_unordered_pairs_with_contact": 1,
                "unique_round_directed_pairs_with_contact": 1,
                "unique_match_directed_pairs_smoke_intersected": 1,
                "unique_match_unordered_pairs_smoke_intersected": 1,
                "reacquisition_episodes_within_round": 0,
                "rounds_with_evaluations": 1,
                "rounds_with_visual_contact": 1,
                "rounds_with_valid_active_smoke": 1,
                "rounds_with_smoke_intersection": 1,
                "rounds_with_otherwise_visible_smoke_block": 1,
            },
            "rounds": [round_summary],
            "audit": {
                "record_schema_id": (
                    "stratai.observed_visual_contact_full_demo_audit_record@1"
                ),
                "encoding": "jsonl",
                "compression": "gzip",
                "evaluation_records": 0,
                "episode_records": 0,
                "logical_records": 0,
                "logical_bytes": 0,
                "logical_sha256": "",
                "gzip_relative_path": VISUAL_AUDIT_FILENAME,
                "gzip_bytes": 0,
                "gzip_sha256": "",
            },
        },
        records,
    )


def _fixture(tmp_path: Path) -> FullDemoFixture:
    root = tmp_path / "validation"
    maps_root = tmp_path / "maps"
    root.mkdir(parents=True)
    maps_root.mkdir(parents=True)
    navigation, navigation_rows = _navigation_report(maps_root)
    visual, visual_records = _visual_report(maps_root)
    fixture = FullDemoFixture(
        root=root,
        maps_root=maps_root,
        navigation_path=root / NAVIGATION_REPORT_FILENAME,
        visual_path=root / VISUAL_REPORT_FILENAME,
        navigation=navigation,
        visual=visual,
        navigation_rows=navigation_rows,
        visual_records=visual_records,
    )
    _write_navigation_stream(fixture)
    _write_visual_stream(fixture)
    _write_reports(fixture)
    return fixture


def _add_second_visual_round(fixture: FullDemoFixture) -> None:
    first_records = copy.deepcopy(fixture.visual_records)
    for record in first_records:
        record["round"] = 2
        value = record["value"]
        value["round_number"] = 2
        value["t0_tick"] = 208
        value["join_keys"]["round_id"] = f"{fixture.visual['source_alias']}:round:002"
        value["join_keys"]["contact_id"] = (
            "contact-000002" if value["join_keys"]["contact_id"] is not None else None
        )
        if record["record_type"] == "evaluation":
            value["tick"] += 100
            value["availability_tick"] += 100
            value["provenance"]["source_round"] = 2
            value["provenance"]["source_frame_tick"] += 100
        else:
            for key in (
                "start_tick",
                "end_tick",
                "last_confirmed_tick",
                "start_availability_tick",
                "end_availability_tick",
            ):
                value[key] += 100
    fixture.visual_records.extend(first_records)

    second_round = copy.deepcopy(fixture.visual["rounds"][0])
    second_round.update(
        {
            "round_number": 2,
            "start_tick": 200,
            "end_tick": 208,
            "audit_logical_records": 0,
            "audit_logical_sha256": "",
        }
    )
    fixture.visual["rounds"].append(second_round)
    selection = fixture.visual["selection"]
    selection.update(
        {
            "round_count": 2,
            "last_round": 2,
            "maximum_end_tick": 208,
            "input_source_frames": 20,
            "canonical_source_frames": 6,
        }
    )
    metrics = fixture.visual["summary"]["metrics"]
    for key, value in list(metrics.items()):
        if key in {"boundaries", "abstention_reasons"}:
            metrics[key] = {name: count * 2 for name, count in value.items()}
        else:
            metrics[key] = value * 2
    fixture.visual["summary"].update(
        {
            "unique_round_directed_pairs_with_contact": 2,
            "reacquisition_episodes_within_round": 0,
            "rounds_with_evaluations": 2,
            "rounds_with_visual_contact": 2,
            "rounds_with_valid_active_smoke": 2,
            "rounds_with_smoke_intersection": 2,
            "rounds_with_otherwise_visible_smoke_block": 2,
        }
    )


def _validate(fixture: FullDemoFixture) -> dict:
    return validate_full_demo_reports(
        fixture.navigation_path, fixture.visual_path, fixture.maps_root
    )


def test_full_demo_reports_and_streams_are_independently_recomputed(
    tmp_path: Path,
) -> None:
    fixture = _fixture(tmp_path)

    result = _validate(fixture)

    assert result["training_allowed"] is False
    assert result["navigation"]["rows"] == 2
    assert result["navigation"]["reachable_queries"] == 2
    assert result["visual"]["evaluations"] == 6
    assert result["visual"]["episodes"] == 1
    assert result["visual"]["smoke_intersections"] == 1
    assert result["visual"]["smoke_blocks"] == 1


def test_visual_stream_is_reconciled_round_by_round(tmp_path: Path) -> None:
    fixture = _fixture(tmp_path)
    _add_second_visual_round(fixture)
    _write_visual_stream(fixture)
    _write_reports(fixture)

    result = _validate(fixture)

    assert result["visual"]["evaluations"] == 12
    assert result["visual"]["episodes"] == 2


@pytest.mark.parametrize(
    "mutation, message",
    [
        (
            lambda fixture: fixture.navigation["timing_policy"].__setitem__(
                "exact_user_facing_time_allowed", True
            ),
            "timing_policy",
        ),
        (
            lambda fixture: fixture.navigation["summary"]["counts"].__setitem__(
                "walking_contexts", 99
            ),
            "streamed navigation rows",
        ),
        (
            lambda fixture: fixture.navigation["summary"]["distributions"][
                "path_distance_world_units"
            ].__setitem__("p95", 999),
            "differs from streamed rows",
        ),
        (
            lambda fixture: fixture.visual["summary"]["metrics"].__setitem__(
                "otherwise_visible_pair_samples_blocked_by_smoke", 0
            ),
            "per-round streamed totals",
        ),
        (
            lambda fixture: fixture.visual.__setitem__("training_allowed", True),
            "training policy",
        ),
    ],
)
def test_report_counter_distribution_smoke_and_policy_tampering_fail(
    tmp_path: Path, mutation, message: str
) -> None:
    fixture = _fixture(tmp_path)
    mutation(fixture)
    _write_reports(fixture)

    with pytest.raises(FullDemoValidationError, match=message):
        _validate(fixture)


def test_navigation_stream_order_and_visual_privacy_are_not_trusted(
    tmp_path: Path,
) -> None:
    fixture = _fixture(tmp_path)
    fixture.navigation_rows.reverse()
    _write_navigation_stream(fixture)
    _write_reports(fixture)
    with pytest.raises(FullDemoValidationError, match="strictly ordered"):
        _validate(fixture)

    fixture = _fixture(tmp_path / "second")
    fixture.visual_records[0]["value"]["join_keys"]["observer_id"] = "76561198000000000"
    _write_visual_stream(fixture)
    _write_reports(fixture)
    with pytest.raises(FullDemoValidationError, match="raw Steam ID"):
        _validate(fixture)


def test_visual_envelope_extras_and_episode_state_tampering_fail(
    tmp_path: Path,
) -> None:
    fixture = _fixture(tmp_path)
    fixture.visual_records[0]["unexpected"] = True
    _write_visual_stream(fixture)
    _write_reports(fixture)
    with pytest.raises(FullDemoValidationError, match=r"unknown=\['unexpected'\]"):
        _validate(fixture)

    fixture = _fixture(tmp_path / "second")
    fixture.visual_records[5]["value"]["join_keys"]["contact_id"] = None
    _write_visual_stream(fixture)
    _write_reports(fixture)
    with pytest.raises(FullDemoValidationError, match="ended boundary"):
        _validate(fixture)


def test_truncated_gzip_and_logical_digest_tampering_fail(tmp_path: Path) -> None:
    fixture = _fixture(tmp_path)
    path = fixture.root / VISUAL_AUDIT_FILENAME
    path.write_bytes(path.read_bytes()[:-7])
    with pytest.raises(FullDemoValidationError, match="truncated gzip|integrity"):
        _validate(fixture)

    fixture = _fixture(tmp_path / "second")
    fixture.navigation["rows_artifact"]["logical_sha256"] = "f" * 64
    _write_reports(fixture)
    with pytest.raises(FullDemoValidationError, match="integrity mismatch"):
        _validate(fixture)


def test_sidecars_and_assets_must_be_safe_regular_hashed_files(tmp_path: Path) -> None:
    fixture = _fixture(tmp_path)
    fixture.navigation["rows_artifact"]["path"] = "../escape.jsonl.gz"
    _write_reports(fixture)
    with pytest.raises(FullDemoValidationError, match="contract mismatch"):
        _validate(fixture)

    fixture = _fixture(tmp_path / "second")
    nav_asset = fixture.maps_root / "de_mirage" / "de_mirage.nav"
    nav_asset.write_bytes(b"tampered")
    with pytest.raises(FullDemoValidationError, match="asset SHA-256 mismatch"):
        _validate(fixture)


def test_cli_requires_maps_root_and_reports_matching_identity(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    fixture = _fixture(tmp_path)
    monkeypatch.setattr(
        sys,
        "argv",
        [
            "validate_full_demo.py",
            str(fixture.navigation_path),
            str(fixture.visual_path),
            "--maps-root",
            str(fixture.maps_root),
        ],
    )
    assert main() == 0
    assert json.loads(capsys.readouterr().out)["ok"] is True

    fixture.visual["build_identifier"] = "another-build"
    _write_reports(fixture)
    assert main() == 1
    output = json.loads(capsys.readouterr().out)
    assert output["ok"] is False
    assert "build_identifier differ" in output["error"]
