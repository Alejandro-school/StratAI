from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import math
import re
from collections import Counter
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator

from ai_coach.spatial.exposure.loader import (
    CONTRACT_VERSION as PACKAGE_CONTRACT_VERSION,
)
from ai_coach.spatial.exposure.loader import (
    ExposurePackageError,
    load_package,
    relative_sectors,
    sector_contains,
)

CONTRACT_VERSION = "2.0.0"
REPORT_SCHEMA_ID = "stratai.observed_exposure_angle_full_demo_report@2"
ROW_SCHEMA_ID = "stratai.observed_exposure_angle_diagnostic@2"
ROW_SCHEMA = Path(__file__).resolve().parent / "observed_exposure_angle_diagnostic_v2.schema.json"
ROWS_FILENAME = "observed_exposure_angle_diagnostic.jsonl.gz"
DIGEST_CONTRACT = "sha256_over_compact_jsonl_with_lf@1"
ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
PARTICIPANT_PATTERN = re.compile(r"^participant-[0-9]{3}$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")


class ExposureAngleValidationError(ValueError):
    """Raised when the real sidecar artifact fails independent validation."""


def validate_exposure_angle_full_demo_report(
    report_path: Path,
    *,
    registry_root: Path,
    maps_root: Path,
    expected_alias: str | None = None,
    expected_sha256: str | None = None,
    expected_map_id: str | None = None,
    require_geometry_observed: bool = False,
    require_geometry_unavailable: bool = False,
    require_tactical_unavailable: bool = False,
) -> dict[str, Any]:
    report_path = report_path.resolve(strict=True)
    report = _load_json(report_path)
    _exact_keys(
        report,
        {
            "schema_id",
            "contract_version",
            "candidate_schema_id",
            "source_alias",
            "source_sha256",
            "map_id",
            "build_identifier",
            "training_allowed",
            "package_resolution",
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
        or not ALIAS_PATTERN.fullmatch(report["source_alias"])
        or report["source_alias"] != "demo-" + report["source_sha256"][:20]
        or not SHA256_PATTERN.fullmatch(report["source_sha256"])
        or not isinstance(report["build_identifier"], str)
        or not report["build_identifier"].strip()
    ):
        raise ExposureAngleValidationError(
            "report identity or safety metadata mismatch"
        )
    if expected_alias is not None and report["source_alias"] != expected_alias:
        raise ExposureAngleValidationError("unexpected source alias")
    if expected_sha256 is not None and report["source_sha256"] != expected_sha256:
        raise ExposureAngleValidationError("unexpected source SHA-256")
    if expected_map_id is not None and report["map_id"] != expected_map_id:
        raise ExposureAngleValidationError("unexpected map id")

    package_summary = report["package_resolution"]
    _exact_keys(
        package_summary,
        {
            "status",
            "reason_codes",
            "contract_version",
            "asset_version",
            "package_sha256",
            "geometry_revision",
            "product_supported",
        },
        "package_resolution",
    )
    if package_summary["status"] != "available":
        raise ExposureAngleValidationError(
            f"real smoke package is unavailable: {package_summary['reason_codes']}"
        )
    resolution = load_package(
        registry_root,
        maps_root,
        report["map_id"],
        contract_version=package_summary["contract_version"],
        asset_version=package_summary["asset_version"],
    )
    if resolution.status != "available" or resolution.package is None:
        raise ExposureAngleValidationError(
            f"independent package resolution failed: {resolution.reason_codes}"
        )
    package = resolution.package
    if (
        package_summary["reason_codes"] != []
        or package_summary["contract_version"] != PACKAGE_CONTRACT_VERSION
        or package_summary["package_sha256"] != resolution.package_sha256
        or package_summary["geometry_revision"] != package["geometry_revision"]
        or package_summary["product_supported"]
        is not resolution.support["product_supported"]
    ):
        raise ExposureAngleValidationError(
            "package resolution summary does not recompute"
        )

    sampling = report["sampling"]
    _exact_keys(
        sampling, {"target_hz", "tick_rate_hz", "period_ticks", "strategy"}, "sampling"
    )
    tick_rate = _finite_number(sampling["tick_rate_hz"], "sampling.tick_rate_hz")
    if (
        sampling["target_hz"] != 1
        or tick_rate <= 0
        or not math.isclose(sampling["period_ticks"], tick_rate, abs_tol=1e-9)
        or sampling["strategy"]
        != "round_anchored_first_frame_at_or_after_period_no_carry_forward"
    ):
        raise ExposureAngleValidationError("sampling contract mismatch")

    artifact = report["rows_artifact"]
    _exact_keys(
        artifact,
        {
            "path",
            "format",
            "compression",
            "bytes",
            "sha256",
            "record_count",
            "logical_bytes",
            "logical_sha256",
            "digest_contract",
        },
        "rows_artifact",
    )
    if (
        artifact["path"] != ROWS_FILENAME
        or Path(artifact["path"]).name != artifact["path"]
        or artifact["format"] != "jsonl"
        or artifact["compression"] != "gzip"
        or artifact["digest_contract"] != DIGEST_CONTRACT
        or not SHA256_PATTERN.fullmatch(artifact["sha256"])
        or not SHA256_PATTERN.fullmatch(artifact["logical_sha256"])
    ):
        raise ExposureAngleValidationError("rows artifact metadata mismatch")
    rows_path = (report_path.parent / artifact["path"]).resolve(strict=True)
    if rows_path.parent != report_path.parent:
        raise ExposureAngleValidationError("rows artifact escapes report directory")

    package_samples = {sample["id"]: sample for sample in package["geometry_samples"]}
    row_validator = _row_validator()
    logical_hash = hashlib.sha256()
    logical_bytes = 0
    physical_hash = hashlib.sha256()
    physical_bytes = 0
    records = 0
    rounds: set[int] = set()
    aliases: set[str] = set()
    geometry_observed = 0
    geometry_unavailable = 0
    tactical_unavailable = 0
    samples_hit: Counter[str] = Counter()
    geometry_reasons: Counter[str] = Counter()
    tactical_reasons: Counter[str] = Counter()
    selected_row_frames: set[tuple[int, int]] = set()
    previous_key: tuple[int, int, str] | None = None
    with rows_path.open("rb") as raw_stream:
        for block in iter(lambda: raw_stream.read(1024 * 1024), b""):
            physical_hash.update(block)
            physical_bytes += len(block)
    with gzip.open(rows_path, "rb") as stream:
        for line_number, raw_line in enumerate(stream, start=1):
            if not raw_line.endswith(b"\n"):
                raise ExposureAngleValidationError(
                    f"row {line_number}: JSONL record lacks LF"
                )
            logical_hash.update(raw_line)
            logical_bytes += len(raw_line)
            row = _decode_json(raw_line)
            errors = sorted(
                row_validator.iter_errors(row), key=lambda error: list(error.path)
            )
            if errors:
                raise ExposureAngleValidationError(
                    f"row {line_number}: schema validation failed: {errors[0].message}"
                )
            _require_finite(row, f"row {line_number}")
            compact = (
                json.dumps(
                    row, ensure_ascii=False, separators=(",", ":"), allow_nan=False
                ).encode("utf-8")
                + b"\n"
            )
            if compact != raw_line:
                raise ExposureAngleValidationError(
                    f"row {line_number}: record is not compact deterministic JSON"
                )
            if (
                row["source_alias"] != report["source_alias"]
                or row["map_id"] != report["map_id"]
                or row["contract_version"] != CONTRACT_VERSION
                or row["asset_version"] != package["asset_version"]
                or row["training_allowed"] is not False
                or row["availability_tick"] > row["t0_tick"]
                or not PARTICIPANT_PATTERN.fullmatch(row["participant_alias"])
                or row["provenance"]["package_sha256"]
                != package_summary["package_sha256"]
                or row["provenance"]["geometry_revision"]
                != package_summary["geometry_revision"]
            ):
                raise ExposureAngleValidationError(
                    f"row {line_number}: identity, causal or provenance mismatch"
                )
            key = (row["round_number"], row["t0_tick"], row["participant_alias"])
            if previous_key is not None and key <= previous_key:
                raise ExposureAngleValidationError(
                    f"row {line_number}: records are not strictly deterministic"
                )
            previous_key = key
            if any(
                forbidden in raw_line.lower()
                for forbidden in (b"steam_id", b"player_name", b"oracle", b"outcome")
            ):
                raise ExposureAngleValidationError(
                    f"row {line_number}: forbidden identity/oracle field"
                )
            _validate_row_semantics(row, package, package_samples, line_number)
            records += 1
            rounds.add(row["round_number"])
            selected_row_frames.add((row["round_number"], row["t0_tick"]))
            aliases.add(row["participant_alias"])
            geometry = row["geometry_clearance"]
            tactical = row["tactical_assessment"]
            if geometry["status"] == "observed":
                geometry_observed += 1
                samples_hit[geometry["sample_id"]] += 1
            else:
                geometry_unavailable += 1
                geometry_reasons.update(geometry["reason_codes"])
            tactical_unavailable += 1
            tactical_reasons.update(tactical["reason_codes"])

    hashes = {
        "bytes": physical_bytes,
        "sha256": physical_hash.hexdigest(),
        "logical_bytes": logical_bytes,
        "logical_sha256": logical_hash.hexdigest(),
    }
    if records != artifact["record_count"] or any(
        artifact[key] != hashes[key] for key in hashes
    ):
        raise ExposureAngleValidationError(
            "rows artifact hashes/count do not recompute"
        )
    recomputed_coverage = {
        "round_numbers": sorted(rounds),
        "input_frames": report["coverage"]["input_frames"],
        "selected_frames": report["coverage"]["selected_frames"],
        "participant_aliases": len(aliases),
        "records": records,
        "geometry_observed": geometry_observed,
        "geometry_unavailable": geometry_unavailable,
        "tactical_judgment_unavailable": tactical_unavailable,
        "samples_hit": dict(sorted(samples_hit.items())),
        "physical_absence_reason_counts": dict(sorted(geometry_reasons.items())),
        "tactical_judgment_reason_counts": dict(sorted(tactical_reasons.items())),
    }
    for producer_only in ("input_frames", "selected_frames"):
        if (
            not isinstance(recomputed_coverage[producer_only], int)
            or isinstance(recomputed_coverage[producer_only], bool)
            or recomputed_coverage[producer_only] < 1
        ):
            raise ExposureAngleValidationError(
                f"coverage.{producer_only} is not a positive integer"
            )
    if (
        recomputed_coverage["selected_frames"] > recomputed_coverage["input_frames"]
        or len(selected_row_frames) > recomputed_coverage["selected_frames"]
    ):
        raise ExposureAngleValidationError(
            "producer frame counts violate replay bounds"
        )
    if recomputed_coverage != report["coverage"]:
        raise ExposureAngleValidationError("coverage does not independently recompute")
    if require_geometry_observed and geometry_observed == 0:
        raise ExposureAngleValidationError("real smoke contains no observed geometry")
    if require_geometry_unavailable and geometry_unavailable == 0:
        raise ExposureAngleValidationError(
            "real smoke contains no explicit geometry absence"
        )
    if require_tactical_unavailable and (
        tactical_unavailable != records
        or tactical_reasons["tactical_decision_context_required"] != records
    ):
        raise ExposureAngleValidationError(
            "tactical judgment was inferred outside a decision contract"
        )
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
        "coverage": recomputed_coverage,
        "rows_artifact": hashes,
        "causal_checks": {
            "future_ticks": 0,
            "oracle_or_outcome_fields": 0,
            "raw_identity_fields": 0,
            "angle_recompute_mismatches": 0,
            "bounds_or_sample_mismatches": 0,
        },
    }


def _validate_row_semantics(
    row: dict[str, Any],
    package: dict[str, Any],
    samples: dict[str, dict[str, Any]],
    line_number: int,
) -> None:
    geometry = row["geometry_clearance"]
    tactical = row["tactical_assessment"]
    position = row["position_world"]
    matches = []
    for sample in samples.values():
        horizontal = math.hypot(
            position["x"] - sample["origin_world"]["x"],
            position["y"] - sample["origin_world"]["y"],
        )
        vertical = abs(position["z"] - sample["origin_world"]["z"])
        if (
            horizontal <= sample["selection_radius_units"]
            and vertical <= sample["vertical_tolerance_units"]
        ):
            matches.append((sample, horizontal, vertical))
    if geometry["status"] == "unavailable":
        reason_codes = geometry["reason_codes"]
        expected_empty = {
            "field_id": "FLD-GEOMETRIC-CLEARANCE-SECTORS",
            "status": "unavailable",
            "reason_codes": reason_codes,
            "sample_id": None,
            "sample_alias": None,
            "horizontal_distance_units": None,
            "vertical_distance_units": None,
            "absolute_geometry_sectors": [],
            "view_relative_geometry_sectors": [],
            "view_inside_geometry_sector": None,
        }
        if geometry != expected_empty:
            raise ExposureAngleValidationError(
                f"row {line_number}: unavailable geometry is not fail-closed"
            )
        if reason_codes == ["outside_declared_sample_coverage"]:
            valid_absence = not matches and row["view_yaw_degrees"] is not None
        elif reason_codes == ["view_angles_unavailable"]:
            valid_absence = row["view_yaw_degrees"] is None
        else:
            valid_absence = False
        if not valid_absence:
            raise ExposureAngleValidationError(
                f"row {line_number}: unavailable sample selection does not recompute"
            )
    else:
        if geometry["reason_codes"] or row["view_yaw_degrees"] is None:
            raise ExposureAngleValidationError(
                f"row {line_number}: observed geometry has absence metadata"
            )
        if len(matches) != 1:
            raise ExposureAngleValidationError(
                f"row {line_number}: observed sample selection is ambiguous"
            )
        sample, horizontal, vertical = matches[0]
        if (
            geometry["sample_id"] != sample["id"]
            or geometry["sample_alias"] != sample["alias"]
        ):
            raise ExposureAngleValidationError(
                f"row {line_number}: sample id/alias mismatch"
            )
        if not math.isclose(
            geometry["horizontal_distance_units"], horizontal, abs_tol=1e-9
        ) or not math.isclose(
            geometry["vertical_distance_units"], vertical, abs_tol=1e-9
        ):
            raise ExposureAngleValidationError(
                f"row {line_number}: sample distances do not recompute"
            )
        if geometry["absolute_geometry_sectors"] != sample["geometry_sectors"]:
            raise ExposureAngleValidationError(
                f"row {line_number}: absolute sectors differ from package"
            )
        expected_relative = relative_sectors(
            sample["geometry_sectors"], row["view_yaw_degrees"]
        )
        if not _numeric_equal(
            geometry["view_relative_geometry_sectors"], expected_relative
        ):
            raise ExposureAngleValidationError(
                f"row {line_number}: relative sectors do not recompute"
            )
        expected_inside = any(
            sector_contains(sector, row["view_yaw_degrees"])
            for sector in sample["geometry_sectors"]
        )
        if geometry["view_inside_geometry_sector"] is not expected_inside:
            raise ExposureAngleValidationError(
                f"row {line_number}: half-open angle membership does not recompute"
            )
    if tactical != {
        "status": "unavailable",
        "reason_codes": ["tactical_decision_context_required"],
    }:
        raise ExposureAngleValidationError(
            f"row {line_number}: tactical judgment crossed the factual boundary"
        )


def _row_validator() -> Draft202012Validator:
    schema = _load_json(ROW_SCHEMA)
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def _numeric_equal(left: Any, right: Any) -> bool:
    if isinstance(left, bool) or isinstance(right, bool):
        return left is right
    if isinstance(left, (int, float)) and isinstance(right, (int, float)):
        return math.isclose(float(left), float(right), abs_tol=1e-9)
    if type(left) is not type(right):
        return False
    if isinstance(left, list):
        return len(left) == len(right) and all(
            _numeric_equal(a, b) for a, b in zip(left, right, strict=True)
        )
    if isinstance(left, dict):
        return left.keys() == right.keys() and all(
            _numeric_equal(left[key], right[key]) for key in left
        )
    return left == right


def _load_json(path: Path) -> dict[str, Any]:
    return _decode_json(path.read_bytes())


def _decode_json(payload: bytes) -> dict[str, Any]:
    value = json.loads(payload.decode("utf-8"), parse_constant=_reject_constant)
    if not isinstance(value, dict):
        raise ExposureAngleValidationError("JSON root is not an object")
    return value


def _reject_constant(constant: str) -> None:
    raise ExposureAngleValidationError(f"non-finite JSON constant: {constant}")


def _exact_keys(value: Any, expected: set[str], path: str) -> None:
    if not isinstance(value, dict) or set(value) != expected:
        raise ExposureAngleValidationError(f"{path}: keys differ from contract")


def _finite_number(value: Any, path: str) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise ExposureAngleValidationError(f"{path}: expected a number")
    result = float(value)
    if not math.isfinite(result):
        raise ExposureAngleValidationError(f"{path}: non-finite number")
    return result


def _require_finite(value: Any, path: str) -> None:
    if isinstance(value, bool) or value is None:
        return
    if isinstance(value, (int, float)):
        _finite_number(value, path)
    elif isinstance(value, dict):
        for key, child in value.items():
            _require_finite(child, f"{path}.{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            _require_finite(child, f"{path}[{index}]")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("report", type=Path)
    parser.add_argument("--registry-root", type=Path, required=True)
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--expected-source-alias")
    parser.add_argument("--expected-source-sha256")
    parser.add_argument("--expected-map-id")
    parser.add_argument("--require-geometry-observed", action="store_true")
    parser.add_argument("--require-geometry-unavailable", action="store_true")
    parser.add_argument("--require-tactical-unavailable", action="store_true")
    arguments = parser.parse_args()
    try:
        result = validate_exposure_angle_full_demo_report(
            arguments.report,
            registry_root=arguments.registry_root,
            maps_root=arguments.maps_root,
            expected_alias=arguments.expected_source_alias,
            expected_sha256=arguments.expected_source_sha256,
            expected_map_id=arguments.expected_map_id,
            require_geometry_observed=arguments.require_geometry_observed,
            require_geometry_unavailable=arguments.require_geometry_unavailable,
            require_tactical_unavailable=arguments.require_tactical_unavailable,
        )
    except (ExposureAngleValidationError, ExposurePackageError, OSError) as exc:
        parser.error(str(exc))
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
