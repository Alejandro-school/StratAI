from __future__ import annotations

import gzip
import hashlib
import json
from pathlib import Path

import pytest

from ai_coach.contracts.validate_cover_full_demo import (
    CoverValidationError,
    validate_cover_full_demo_report,
)
from ai_coach.spatial.cover.loader import normalize_degrees, profile_for_posture

ROOT = Path(__file__).parents[2]
REGISTRY_ROOT = ROOT / "ai_coach" / "spatial" / "cover"
MAPS_ROOT = ROOT / "backend" / "data" / "maps"
PACKAGE_PATH = (
    REGISTRY_ROOT / "packages" / "de_mirage" / "1.0.0-candidate.1" / "package.json"
)
CHECKSUM = "a" * 64
ALIAS = "demo-" + CHECKSUM[:20]


def _package() -> dict:
    return json.loads(PACKAGE_PATH.read_text(encoding="utf-8"))


def _provenance(package: dict) -> dict:
    return {
        "position_source": "replay_player_state_position_16hz",
        "posture_source": "replay_player_state_is_ducking_16hz",
        "view_source": "replay_player_state_view_yaw_16hz",
        "package_schema_id": "stratai.cover_map_package@1",
        "package_contract_version": "1.0.0",
        "package_asset_version": package["asset_version"],
        "package_sha256": hashlib.sha256(PACKAGE_PATH.read_bytes()).hexdigest(),
        "geometry_revision": package["geometry_revision"],
    }


def _observed_row(package: dict) -> dict:
    sample = package["geometry_samples"][0]
    yaw = 15.0
    directions = []
    for direction in sample["directions"]:
        profile = profile_for_posture(direction, "standing")
        assert profile is not None
        directions.append(
            {
                "geometry_direction_ref": direction["id"],
                "absolute_azimuth_degrees": direction["azimuth_degrees"],
                "view_relative_azimuth_degrees": normalize_degrees(
                    direction["azimuth_degrees"] - yaw
                ),
                "probe_count": profile["probe_count"],
                "occluded_probe_count": profile["occluded_probe_count"],
                "complete_volume_occlusion": profile[
                    "complete_volume_occlusion"
                ],
            }
        )
    return {
        "schema_id": "stratai.observed_cover_diagnostic@2",
        "contract_version": "2.0.0",
        "source_alias": ALIAS,
        "map_id": "de_mirage",
        "round_number": 1,
        "t0_tick": 64,
        "availability_tick": 64,
        "participant_alias": "participant-001",
        "training_allowed": False,
        "position_world": sample["origin_world"],
        "view_yaw_degrees": yaw,
        "posture_id": "standing",
        "geometry_occlusion": {
            "field_id": "FLD-GEOMETRIC-OCCLUSION",
            "status": "observed",
            "reason_codes": [],
            "sample_id": sample["id"],
            "sample_alias": sample["alias"],
            "directions": directions,
        },
        "tactical_assessment": {
            "status": "unavailable",
            "reason_codes": ["tactical_decision_context_required"],
        },
        "provenance": _provenance(package),
    }


def _unavailable_row(package: dict) -> dict:
    return {
        "schema_id": "stratai.observed_cover_diagnostic@2",
        "contract_version": "2.0.0",
        "source_alias": ALIAS,
        "map_id": "de_mirage",
        "round_number": 1,
        "t0_tick": 128,
        "availability_tick": 128,
        "participant_alias": "participant-002",
        "training_allowed": False,
        "position_world": {"x": 999999, "y": 999999, "z": 0},
        "view_yaw_degrees": 0,
        "posture_id": "crouching",
        "geometry_occlusion": {
            "field_id": "FLD-GEOMETRIC-OCCLUSION",
            "status": "unavailable",
            "reason_codes": ["outside_declared_sample_coverage"],
            "sample_id": None,
            "sample_alias": None,
            "directions": [],
        },
        "tactical_assessment": {
            "status": "unavailable",
            "reason_codes": ["tactical_decision_context_required"],
        },
        "provenance": _provenance(package),
    }


def _write_fixture(tmp_path: Path) -> Path:
    package = _package()
    rows = [_observed_row(package), _unavailable_row(package)]
    logical = b"".join(
        json.dumps(row, separators=(",", ":"), sort_keys=True).encode() + b"\n"
        for row in rows
    )
    rows_path = tmp_path / "observed_cover_diagnostic.jsonl.gz"
    with gzip.GzipFile(filename=rows_path, mode="wb", mtime=0) as stream:
        stream.write(logical)
    complete_standing = sum(
        direction["complete_volume_occlusion"]
        for direction in rows[0]["geometry_occlusion"]["directions"]
    )
    volume = package["player_volume"]
    report = {
        "schema_id": "stratai.observed_cover_full_demo_report@2",
        "contract_version": "2.0.0",
        "candidate_schema_id": "stratai.observed_cover_diagnostic@2",
        "build_identifier": "test@1",
        "source_alias": ALIAS,
        "source_sha256": CHECKSUM,
        "map_id": "de_mirage",
        "training_allowed": False,
        "product_supported": False,
        "package_resolution": {
            "status": "available",
            "reason_codes": [],
            "contract_version": "1.0.0",
            "asset_version": package["asset_version"],
            "package_sha256": hashlib.sha256(PACKAGE_PATH.read_bytes()).hexdigest(),
            "geometry_revision": package["geometry_revision"],
            "product_supported": False,
        },
        "player_volume": {
            "model_id": volume["model_id"],
            "radius_units": volume["radius_units"],
            "lateral_offsets_units": volume["lateral_offsets_units"],
            "posture_ids": [posture["id"] for posture in volume["postures"]],
            "probes_per_posture": {
                posture["id"]: len(posture["probe_heights_units"])
                * len(volume["lateral_offsets_units"])
                for posture in volume["postures"]
            },
            "azimuth_step_degrees": package["resolution"]["azimuth_step_degrees"],
            "protection_distance_units": package["resolution"][
                "protection_distance_units"
            ],
        },
        "sampling": {
            "target_hz": 1,
            "tick_rate_hz": 64,
            "period_ticks": 64,
            "strategy": "round_anchored_right_closed_alive_players",
        },
        "rows_artifact": {
            "path": rows_path.name,
            "format": "jsonl",
            "compression": "gzip",
            "bytes": rows_path.stat().st_size,
            "sha256": hashlib.sha256(rows_path.read_bytes()).hexdigest(),
            "logical_records": 2,
            "logical_bytes": len(logical),
            "logical_sha256": hashlib.sha256(logical).hexdigest(),
            "digest_contract": "sha256_over_compact_jsonl_with_lf@1",
        },
        "coverage": {
            "rounds": 1,
            "records": 2,
            "geometry_observed": 1,
            "geometry_unavailable": 1,
            "directions_observed": 24,
            "complete_crouching_directions": 0,
            "complete_standing_directions": complete_standing,
            "tactical_judgment_unavailable": 2,
            "samples_hit": {"cover-sample-001": 1},
            "physical_absence_reason_counts": {
                "outside_declared_sample_coverage": 1
            },
            "tactical_judgment_reason_counts": {
                "tactical_decision_context_required": 2,
            },
        },
    }
    report_path = tmp_path / "observed_cover_full_demo.json"
    report_path.write_text(json.dumps(report), encoding="utf-8")
    return report_path


def test_independent_cover_validator_recomputes_geometry_and_abstention(
    tmp_path: Path,
) -> None:
    result = validate_cover_full_demo_report(
        _write_fixture(tmp_path),
        REGISTRY_ROOT,
        MAPS_ROOT,
        expected_alias=ALIAS,
        expected_sha256=CHECKSUM,
        expected_map_id="de_mirage",
        require_observed=True,
    )
    assert result["status"] == "passed"
    assert result["records"] == 2
    assert result["geometry_observed"] == 1
    assert result["directions_observed"] == 24
    assert result["tactical_judgment_unavailable"] == 2
    assert result["package_counts"] == {
        "samples": 8,
        "directions": 192,
        "profiles": 384,
        "annotations": 384,
    }


@pytest.mark.parametrize(
    ("mutate", "message"),
    [
        (
            lambda row: row.__setitem__("availability_tick", row["t0_tick"] + 1),
            "schema violation|causal tick",
        ),
        (
            lambda row: row["geometry_occlusion"]["directions"][0].__setitem__(
                "occluded_probe_count", 99
            ),
            "schema violation|does not recompute",
        ),
        (
            lambda row: row.__setitem__("oracle", {"enemy_position": [1, 2, 3]}),
            "schema violation|forbidden",
        ),
        (
            lambda row: row["tactical_assessment"].update(
                {
                    "status": "observed",
                    "reason_codes": [],
                }
            ),
            "schema violation|tactical judgment crossed",
        ),
    ],
)
def test_independent_validator_rejects_future_geometry_or_unreviewed_tactics(
    tmp_path: Path, mutate, message: str
) -> None:
    report_path = _write_fixture(tmp_path)
    report = json.loads(report_path.read_text())
    rows_path = tmp_path / report["rows_artifact"]["path"]
    with gzip.open(rows_path, "rb") as stream:
        rows = [json.loads(line) for line in stream]
    mutate(rows[0])
    logical = b"".join(
        json.dumps(row, separators=(",", ":"), sort_keys=True).encode() + b"\n"
        for row in rows
    )
    with gzip.GzipFile(filename=rows_path, mode="wb", mtime=0) as stream:
        stream.write(logical)
    report["rows_artifact"].update(
        {
            "bytes": rows_path.stat().st_size,
            "sha256": hashlib.sha256(rows_path.read_bytes()).hexdigest(),
            "logical_bytes": len(logical),
            "logical_sha256": hashlib.sha256(logical).hexdigest(),
        }
    )
    report_path.write_text(json.dumps(report))
    with pytest.raises(CoverValidationError, match=message):
        validate_cover_full_demo_report(report_path, REGISTRY_ROOT, MAPS_ROOT)
