from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import math
import re
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator
from jsonschema.exceptions import SchemaError, ValidationError

from ai_coach.spatial.cover.loader import (
    CONTRACT_VERSION as PACKAGE_CONTRACT_VERSION,
)
from ai_coach.spatial.cover.loader import (
    load_package,
    normalize_degrees,
    profile_for_posture,
    select_sample,
    validate_package,
)

CONTRACT_VERSION = "2.0.0"
REPORT_SCHEMA_ID = "stratai.observed_cover_full_demo_report@2"
ROW_SCHEMA_ID = "stratai.observed_cover_diagnostic@2"
ROW_FILENAME = "observed_cover_diagnostic.jsonl.gz"
DIGEST_CONTRACT = "sha256_over_compact_jsonl_with_lf@1"
ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
PARTICIPANT_PATTERN = re.compile(r"^participant-[0-9]{3}$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
FORBIDDEN_KEYS = {
    "steam_id",
    "steamid",
    "name",
    "enemy_position",
    "oracle",
    "outcome",
    "outcomes",
    "winner",
    "future",
}

ROOT = Path(__file__).resolve().parent
ROW_SCHEMA = ROOT / "observed_cover_diagnostic_v2.schema.json"


class CoverValidationError(ValueError):
    """Raised when a full-demo cover sidecar fails independent validation."""


def validate_cover_full_demo_report(
    report_path: Path,
    registry_root: Path,
    maps_root: Path,
    *,
    expected_alias: str | None = None,
    expected_sha256: str | None = None,
    expected_map_id: str | None = None,
    require_observed: bool = False,
) -> dict[str, Any]:
    report_path = report_path.resolve(strict=True)
    report = _load_json(report_path)
    _require_finite(report, "report")
    _exact_keys(
        report,
        {
            "schema_id",
            "contract_version",
            "candidate_schema_id",
            "build_identifier",
            "source_alias",
            "source_sha256",
            "map_id",
            "training_allowed",
            "product_supported",
            "package_resolution",
            "player_volume",
            "sampling",
            "rows_artifact",
            "coverage",
        },
        "report",
    )
    if (
        report["schema_id"] != REPORT_SCHEMA_ID
        or report["contract_version"] != CONTRACT_VERSION
        or report["candidate_schema_id"] != ROW_SCHEMA_ID
        or report["training_allowed"] is not False
        or report["product_supported"] is not False
        or not ALIAS_PATTERN.fullmatch(report["source_alias"])
        or not SHA256_PATTERN.fullmatch(report["source_sha256"])
        or report["source_alias"] != "demo-" + report["source_sha256"][:20]
        or not isinstance(report["build_identifier"], str)
        or not report["build_identifier"].strip()
    ):
        raise CoverValidationError("report identity or safety metadata mismatch")
    if expected_alias is not None and report["source_alias"] != expected_alias:
        raise CoverValidationError("unexpected source alias")
    if expected_sha256 is not None and report["source_sha256"] != expected_sha256:
        raise CoverValidationError("unexpected source SHA-256")
    if expected_map_id is not None and report["map_id"] != expected_map_id:
        raise CoverValidationError("unexpected map id")

    resolution = load_package(
        registry_root.resolve(strict=True),
        maps_root.resolve(strict=True),
        report["map_id"],
        contract_version=PACKAGE_CONTRACT_VERSION,
    )
    if resolution.status != "available" or resolution.package is None:
        raise CoverValidationError(
            f"package failed closed during independent load: {resolution.reason_codes}"
        )
    package = resolution.package
    package_counts = validate_package(package, maps_root=maps_root)
    summary = report["package_resolution"]
    expected_summary = {
        "status": "available",
        "reason_codes": [],
        "contract_version": PACKAGE_CONTRACT_VERSION,
        "asset_version": package["asset_version"],
        "package_sha256": resolution.package_sha256,
        "geometry_revision": package["geometry_revision"],
        "product_supported": False,
    }
    if summary != expected_summary:
        raise CoverValidationError("package resolution summary does not match registry")

    volume = package["player_volume"]
    expected_volume = {
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
    }
    if report["player_volume"] != expected_volume:
        raise CoverValidationError("player volume summary does not match package")
    sampling = report["sampling"]
    if (
        sampling["target_hz"] != 1
        or sampling["strategy"] != "round_anchored_right_closed_alive_players"
        or not _numeric_equal(
            sampling["period_ticks"], sampling["tick_rate_hz"]
        )
    ):
        raise CoverValidationError("sampling contract is invalid")

    artifact = report["rows_artifact"]
    _exact_keys(
        artifact,
        {
            "path",
            "format",
            "compression",
            "bytes",
            "sha256",
            "logical_records",
            "logical_bytes",
            "logical_sha256",
            "digest_contract",
        },
        "rows_artifact",
    )
    if (
        artifact["path"] != ROW_FILENAME
        or artifact["format"] != "jsonl"
        or artifact["compression"] != "gzip"
        or artifact["digest_contract"] != DIGEST_CONTRACT
        or not SHA256_PATTERN.fullmatch(artifact["sha256"])
        or not SHA256_PATTERN.fullmatch(artifact["logical_sha256"])
    ):
        raise CoverValidationError("rows artifact metadata is invalid")
    rows_path = (report_path.parent / artifact["path"]).resolve(strict=True)
    if rows_path.parent != report_path.parent:
        raise CoverValidationError("rows artifact escapes report directory")
    if rows_path.stat().st_size != artifact["bytes"]:
        raise CoverValidationError("rows artifact byte count mismatch")
    if _sha256_file(rows_path) != artifact["sha256"]:
        raise CoverValidationError("rows artifact SHA-256 mismatch")

    row_validator = _row_validator()
    logical_hash = hashlib.sha256()
    logical_bytes = 0
    counts: dict[str, Any] = {
        "rounds": set(),
        "records": 0,
        "geometry_observed": 0,
        "geometry_unavailable": 0,
        "directions_observed": 0,
        "complete_crouching_directions": 0,
        "complete_standing_directions": 0,
        "tactical_judgment_unavailable": 0,
        "samples_hit": {},
        "physical_absence_reason_counts": {},
        "tactical_judgment_reason_counts": {},
    }
    participants: set[str] = set()
    posture_counts: dict[str, int] = {"crouching": 0, "standing": 0}
    with gzip.open(rows_path, "rb") as stream:
        for line_number, raw_line in enumerate(stream, start=1):
            if not raw_line.endswith(b"\n"):
                raise CoverValidationError(f"row {line_number} lacks LF terminator")
            logical_hash.update(raw_line)
            logical_bytes += len(raw_line)
            try:
                row = _decode_json(raw_line)
                row_validator.validate(row)
            except (json.JSONDecodeError, SchemaError, ValidationError) as exc:
                raise CoverValidationError(
                    f"row {line_number} schema violation: {exc}"
                ) from exc
            _require_finite(row, f"row {line_number}")
            _reject_forbidden_keys(row, f"row {line_number}")
            _validate_row_semantics(row, report, package)
            _count_row(counts, row)
            participants.add(row["participant_alias"])
            posture_counts[row["posture_id"]] += 1

    if artifact["logical_records"] != counts["records"]:
        raise CoverValidationError("logical record count mismatch")
    if artifact["logical_bytes"] != logical_bytes:
        raise CoverValidationError("logical byte count mismatch")
    if artifact["logical_sha256"] != logical_hash.hexdigest():
        raise CoverValidationError("logical SHA-256 mismatch")
    recomputed_coverage = {
        **{key: value for key, value in counts.items() if key != "rounds"},
        "rounds": len(counts["rounds"]),
    }
    if recomputed_coverage != report["coverage"]:
        raise CoverValidationError("coverage summary does not recompute from rows")
    if require_observed and counts["geometry_observed"] == 0:
        raise CoverValidationError("no objective cover geometry was observed")
    if counts["tactical_judgment_unavailable"] != counts["records"]:
        raise CoverValidationError("tactical judgment crossed the factual boundary")
    return {
        "status": "passed",
        "schema_id": report["schema_id"],
        "candidate_schema_id": report["candidate_schema_id"],
        "source_alias": report["source_alias"],
        "source_sha256": report["source_sha256"],
        "map_id": report["map_id"],
        "contract_version": report["contract_version"],
        "asset_version": package["asset_version"],
        "package_sha256": resolution.package_sha256,
        "geometry_revision": package["geometry_revision"],
        "package_counts": package_counts,
        "records": counts["records"],
        "participants": len(participants),
        "posture_counts": posture_counts,
        "geometry_observed": counts["geometry_observed"],
        "geometry_unavailable": counts["geometry_unavailable"],
        "directions_observed": counts["directions_observed"],
        "complete_crouching_directions": counts[
            "complete_crouching_directions"
        ],
        "complete_standing_directions": counts["complete_standing_directions"],
        "tactical_judgment_unavailable": counts["tactical_judgment_unavailable"],
        "samples_hit": counts["samples_hit"],
        "logical_bytes": logical_bytes,
        "logical_sha256": logical_hash.hexdigest(),
        "physical_bytes": rows_path.stat().st_size,
        "physical_sha256": artifact["sha256"],
    }


def _validate_row_semantics(
    row: dict[str, Any], report: dict[str, Any], package: dict[str, Any]
) -> None:
    if (
        row["source_alias"] != report["source_alias"]
        or row["map_id"] != report["map_id"]
        or row["availability_tick"] > row["t0_tick"]
        or row["availability_tick"] != row["t0_tick"]
        or row["training_allowed"] is not False
        or not PARTICIPANT_PATTERN.fullmatch(row["participant_alias"])
    ):
        raise CoverValidationError("row identity, causal tick or privacy mismatch")
    provenance = row["provenance"]
    if (
        provenance["package_asset_version"] != package["asset_version"]
        or provenance["package_sha256"]
        != report["package_resolution"]["package_sha256"]
        or provenance["geometry_revision"] != package["geometry_revision"]
    ):
        raise CoverValidationError("row package provenance mismatch")
    geometry = row["geometry_occlusion"]
    if row["position_world"] is None:
        selected, reason = None, "position_unavailable"
    else:
        selected, reason = select_sample(package, row["position_world"])
    if geometry["status"] == "unavailable":
        if (
            selected is not None or geometry["reason_codes"] != [reason]
        ) and geometry["reason_codes"] != ["view_angles_unavailable"]:
            raise CoverValidationError("unavailable geometry does not recompute")
    else:
        if selected is None or geometry["sample_id"] != selected["id"] or geometry[
            "sample_alias"
        ] != selected["alias"]:
            raise CoverValidationError("selected geometry sample does not recompute")
        yaw = row["view_yaw_degrees"]
        if yaw is None or len(geometry["directions"]) != len(selected["directions"]):
            raise CoverValidationError("observed direction grid is incomplete")
        for actual, expected in zip(
            geometry["directions"], selected["directions"], strict=True
        ):
            profile = profile_for_posture(expected, row["posture_id"])
            if profile is None:
                raise CoverValidationError("current posture profile is missing")
            expected_row = {
                "geometry_direction_ref": expected["id"],
                "absolute_azimuth_degrees": expected["azimuth_degrees"],
                "view_relative_azimuth_degrees": normalize_degrees(
                    expected["azimuth_degrees"] - yaw
                ),
                "probe_count": profile["probe_count"],
                "occluded_probe_count": profile["occluded_probe_count"],
                "complete_volume_occlusion": profile["complete_volume_occlusion"],
            }
            if not _numeric_equal(actual, expected_row):
                raise CoverValidationError(
                    "objective direction/profile does not recompute"
                )
    tactical = row["tactical_assessment"]
    if tactical != {
        "status": "unavailable",
        "reason_codes": ["tactical_decision_context_required"],
    }:
        raise CoverValidationError("tactical judgment crossed the factual boundary")


def _count_row(counts: dict[str, Any], row: dict[str, Any]) -> None:
    counts["rounds"].add(row["round_number"])
    counts["records"] += 1
    geometry = row["geometry_occlusion"]
    if geometry["status"] == "observed":
        counts["geometry_observed"] += 1
        sample_id = geometry["sample_id"]
        counts["samples_hit"][sample_id] = counts["samples_hit"].get(sample_id, 0) + 1
        counts["directions_observed"] += len(geometry["directions"])
        key = f"complete_{row['posture_id']}_directions"
        counts[key] += sum(
            direction["complete_volume_occlusion"]
            for direction in geometry["directions"]
        )
    else:
        counts["geometry_unavailable"] += 1
        _count_reasons(counts["physical_absence_reason_counts"], geometry["reason_codes"])
    tactical = row["tactical_assessment"]
    counts["tactical_judgment_unavailable"] += 1
    _count_reasons(counts["tactical_judgment_reason_counts"], tactical["reason_codes"])


def _count_reasons(target: dict[str, int], reasons: list[str]) -> None:
    for reason in reasons:
        target[reason] = target.get(reason, 0) + 1


def _row_validator() -> Draft202012Validator:
    schema = _load_json(ROW_SCHEMA)
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def _reject_forbidden_keys(value: Any, path: str) -> None:
    if isinstance(value, dict):
        for key, child in value.items():
            lowered = key.casefold()
            if lowered in FORBIDDEN_KEYS or lowered.startswith(("oracle_", "future_")):
                raise CoverValidationError(f"{path} contains forbidden field {key}")
            _reject_forbidden_keys(child, f"{path}.{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            _reject_forbidden_keys(child, f"{path}[{index}]")


def _numeric_equal(left: Any, right: Any) -> bool:
    if isinstance(left, bool) or isinstance(right, bool):
        return left is right
    if isinstance(left, (int, float)) and isinstance(right, (int, float)):
        return math.isclose(float(left), float(right), abs_tol=1e-9)
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        return left.keys() == right.keys() and all(
            _numeric_equal(left[key], right[key]) for key in left
        )
    if isinstance(left, list):
        return len(left) == len(right) and all(
            _numeric_equal(a, b) for a, b in zip(left, right, strict=True)
        )
    return left == right


def _load_json(path: Path) -> dict[str, Any]:
    return _decode_json(path.read_bytes())


def _decode_json(payload: bytes) -> dict[str, Any]:
    value = json.loads(
        payload.decode("utf-8"),
        parse_constant=lambda constant: (_ for _ in ()).throw(
            json.JSONDecodeError(f"invalid constant {constant}", constant, 0)
        ),
    )
    if not isinstance(value, dict):
        raise CoverValidationError("JSON root must be an object")
    return value


def _exact_keys(value: Any, expected: set[str], path: str) -> None:
    if not isinstance(value, dict) or set(value) != expected:
        raise CoverValidationError(f"{path} has unexpected keys")


def _require_finite(value: Any, path: str) -> None:
    if isinstance(value, bool) or value is None:
        return
    if isinstance(value, (int, float)):
        if not math.isfinite(float(value)):
            raise CoverValidationError(f"{path} contains a non-finite number")
    elif isinstance(value, dict):
        for key, child in value.items():
            _require_finite(child, f"{path}.{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            _require_finite(child, f"{path}[{index}]")


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("report", type=Path)
    parser.add_argument("--registry-root", type=Path, required=True)
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--expected-source-alias")
    parser.add_argument("--expected-source-sha256")
    parser.add_argument("--expected-map-id")
    parser.add_argument("--require-observed", action="store_true")
    arguments = parser.parse_args()
    try:
        result = validate_cover_full_demo_report(
            arguments.report,
            arguments.registry_root,
            arguments.maps_root,
            expected_alias=arguments.expected_source_alias,
            expected_sha256=arguments.expected_source_sha256,
            expected_map_id=arguments.expected_map_id,
            require_observed=arguments.require_observed,
        )
    except (CoverValidationError, OSError, ValueError) as exc:
        parser.error(str(exc))
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
