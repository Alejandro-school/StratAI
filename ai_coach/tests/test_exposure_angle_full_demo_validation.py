from __future__ import annotations

import copy
import gzip
import hashlib
import json
from collections.abc import Callable
from pathlib import Path

import pytest

from ai_coach.contracts.validate_exposure_angle_full_demo import (
    ExposureAngleValidationError,
    validate_exposure_angle_full_demo_report,
)
from ai_coach.spatial.exposure.loader import (
    load_package,
    relative_sectors,
    sector_contains,
)

ROOT = Path(__file__).parents[2]
REGISTRY_ROOT = ROOT / "ai_coach" / "spatial" / "exposure"
MAPS_ROOT = ROOT / "backend" / "data" / "maps"
SOURCE_SHA256 = "1" * 64
SOURCE_ALIAS = "demo-" + SOURCE_SHA256[:20]


def _compact_row(row: dict) -> bytes:
    return (
        json.dumps(
            row, ensure_ascii=False, separators=(",", ":"), allow_nan=False
        ).encode("utf-8")
        + b"\n"
    )


def _base_rows(package: dict, package_sha256: str) -> list[dict]:
    sample = package["geometry_samples"][0]
    yaw = 359.0
    provenance = {
        "position_source": "replay_player_state_position_16hz",
        "view_angles_source": "demoinfocs_view_angles_at_source_frame",
        "package_sha256": package_sha256,
        "geometry_revision": package["geometry_revision"],
        "package_contract_version": "1.0.0",
    }
    position = copy.deepcopy(sample["origin_world"])
    observed = {
        "schema_id": "stratai.observed_exposure_angle_diagnostic@2",
        "contract_version": "2.0.0",
        "source_alias": SOURCE_ALIAS,
        "map_id": "de_mirage",
        "asset_version": package["asset_version"],
        "round_number": 1,
        "t0_tick": 100,
        "availability_tick": 100,
        "participant_alias": "participant-001",
        "position_world": position,
        "view_yaw_degrees": yaw,
        "geometry_clearance": {
            "field_id": "FLD-GEOMETRIC-CLEARANCE-SECTORS",
            "status": "observed",
            "reason_codes": [],
            "sample_id": sample["id"],
            "sample_alias": sample["alias"],
            "horizontal_distance_units": 0.0,
            "vertical_distance_units": 0.0,
            "absolute_geometry_sectors": copy.deepcopy(sample["geometry_sectors"]),
            "view_relative_geometry_sectors": relative_sectors(
                sample["geometry_sectors"], yaw
            ),
            "view_inside_geometry_sector": any(
                sector_contains(sector, yaw) for sector in sample["geometry_sectors"]
            ),
        },
        "tactical_assessment": {
            "status": "unavailable",
            "reason_codes": ["tactical_decision_context_required"],
        },
        "provenance": provenance,
        "training_allowed": False,
    }
    outside = {
        "schema_id": "stratai.observed_exposure_angle_diagnostic@2",
        "contract_version": "2.0.0",
        "source_alias": SOURCE_ALIAS,
        "map_id": "de_mirage",
        "asset_version": package["asset_version"],
        "round_number": 1,
        "t0_tick": 100,
        "availability_tick": 100,
        "participant_alias": "participant-002",
        "position_world": copy.deepcopy(package["bounds"]["min"]),
        "view_yaw_degrees": 0.0,
        "geometry_clearance": {
            "field_id": "FLD-GEOMETRIC-CLEARANCE-SECTORS",
            "status": "unavailable",
            "reason_codes": ["outside_declared_sample_coverage"],
            "sample_id": None,
            "sample_alias": None,
            "horizontal_distance_units": None,
            "vertical_distance_units": None,
            "absolute_geometry_sectors": [],
            "view_relative_geometry_sectors": [],
            "view_inside_geometry_sector": None,
        },
        "tactical_assessment": {
            "status": "unavailable",
            "reason_codes": ["tactical_decision_context_required"],
        },
        "provenance": copy.deepcopy(provenance),
        "training_allowed": False,
    }
    return [observed, outside]


def _write_fixture(
    directory: Path,
    mutate_rows: Callable[[list[dict]], None] | None = None,
) -> Path:
    directory.mkdir(parents=True, exist_ok=True)
    resolution = load_package(REGISTRY_ROOT, MAPS_ROOT, "de_mirage")
    assert resolution.package is not None
    assert resolution.package_sha256 is not None
    package = resolution.package
    rows = _base_rows(package, resolution.package_sha256)
    if mutate_rows is not None:
        mutate_rows(rows)
    logical_payload = b"".join(_compact_row(row) for row in rows)
    rows_path = directory / "observed_exposure_angle_diagnostic.jsonl.gz"
    with gzip.open(rows_path, "wb", compresslevel=9) as stream:
        stream.write(logical_payload)
    physical_payload = rows_path.read_bytes()
    report = {
        "schema_id": "stratai.observed_exposure_angle_full_demo_report@2",
        "contract_version": "2.0.0",
        "candidate_schema_id": "stratai.observed_exposure_angle_diagnostic@2",
        "source_alias": SOURCE_ALIAS,
        "source_sha256": SOURCE_SHA256,
        "map_id": "de_mirage",
        "build_identifier": "fixture-build",
        "training_allowed": False,
        "package_resolution": {
            "status": "available",
            "reason_codes": [],
            "contract_version": "1.0.0",
            "asset_version": package["asset_version"],
            "package_sha256": resolution.package_sha256,
            "geometry_revision": package["geometry_revision"],
            "product_supported": False,
        },
        "sampling": {
            "target_hz": 1,
            "tick_rate_hz": 64.0,
            "period_ticks": 64.0,
            "strategy": "round_anchored_first_frame_at_or_after_period_no_carry_forward",
        },
        "rows_artifact": {
            "path": rows_path.name,
            "format": "jsonl",
            "compression": "gzip",
            "bytes": len(physical_payload),
            "sha256": hashlib.sha256(physical_payload).hexdigest(),
            "record_count": len(rows),
            "logical_bytes": len(logical_payload),
            "logical_sha256": hashlib.sha256(logical_payload).hexdigest(),
            "digest_contract": "sha256_over_compact_jsonl_with_lf@1",
        },
        "coverage": {
            "round_numbers": [1],
            "input_frames": 16,
            "selected_frames": 1,
            "participant_aliases": 2,
            "records": 2,
            "geometry_observed": 1,
            "geometry_unavailable": 1,
            "tactical_judgment_unavailable": 2,
            "samples_hit": {package["geometry_samples"][0]["id"]: 1},
            "physical_absence_reason_counts": {
                "outside_declared_sample_coverage": 1
            },
            "tactical_judgment_reason_counts": {
                "tactical_decision_context_required": 2,
            },
        },
    }
    report_path = directory / "observed_exposure_angle_full_demo.json"
    report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return report_path


def test_independent_validator_recomputes_geometry_angles_counts_and_hashes(
    tmp_path: Path,
) -> None:
    report_path = _write_fixture(tmp_path)
    result = validate_exposure_angle_full_demo_report(
        report_path,
        registry_root=REGISTRY_ROOT,
        maps_root=MAPS_ROOT,
        expected_alias=SOURCE_ALIAS,
        expected_sha256=SOURCE_SHA256,
        expected_map_id="de_mirage",
        require_geometry_observed=True,
        require_geometry_unavailable=True,
        require_tactical_unavailable=True,
    )
    assert result["status"] == "passed"
    assert result["coverage"]["geometry_observed"] == 1
    assert result["coverage"]["geometry_unavailable"] == 1
    assert result["coverage"]["tactical_judgment_unavailable"] == 2
    assert result["causal_checks"] == {
        "future_ticks": 0,
        "oracle_or_outcome_fields": 0,
        "raw_identity_fields": 0,
        "angle_recompute_mismatches": 0,
        "bounds_or_sample_mismatches": 0,
    }


@pytest.mark.parametrize(
    ("mutation", "message"),
    [
        (
            lambda rows: rows[0].__setitem__("availability_tick", 101),
            "identity, causal or provenance mismatch",
        ),
        (
            lambda rows: rows[0]["geometry_clearance"][
                "view_relative_geometry_sectors"
            ][0].__setitem__("start_degrees", 123.0),
            "relative sectors do not recompute",
        ),
        (
            lambda rows: rows[0].__setitem__(
                "tactical_assessment",
                {
                    "status": "observed",
                    "reason_codes": [],
                },
            ),
            "schema validation failed|tactical judgment crossed",
        ),
        (
            lambda rows: rows[1]["geometry_clearance"].__setitem__(
                "sample_id", "sample-001"
            ),
            "not fail-closed",
        ),
    ],
)
def test_independent_validator_rejects_causal_semantic_and_abstention_tampering(
    tmp_path: Path,
    mutation: Callable[[list[dict]], None],
    message: str,
) -> None:
    report_path = _write_fixture(tmp_path, mutation)
    with pytest.raises(ExposureAngleValidationError, match=message):
        validate_exposure_angle_full_demo_report(
            report_path,
            registry_root=REGISTRY_ROOT,
            maps_root=MAPS_ROOT,
        )
