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

from ai_coach.spatial.exit_route.loader import (
    CONTRACT_VERSION as PACKAGE_CONTRACT_VERSION,
)
from ai_coach.spatial.exit_route.loader import (
    load_package,
    select_sample,
    validate_package,
)

CONTRACT_VERSION = "2.0.0"
REPORT_SCHEMA_ID = "stratai.observed_exit_route_full_demo_report@2"
ROW_SCHEMA_ID = "stratai.observed_exit_route_diagnostic@2"
ROW_FILENAME = "observed_exit_route_diagnostic.jsonl.gz"
ALIAS = re.compile(r"^demo-[0-9a-f]{20}$")
PARTICIPANT = re.compile(r"^participant-[0-9]{3}$")
SHA256 = re.compile(r"^[0-9a-f]{64}$")
FORBIDDEN_KEYS = {
    "steam_id",
    "steamid",
    "name",
    "oracle",
    "outcome",
    "outcomes",
    "winner",
    "future",
    "recommendation",
}
SCHEMA = Path(__file__).resolve().parent / "observed_exit_route_diagnostic_v2.schema.json"


class ExitRouteValidationError(ValueError):
    """Raised when a full-demo candidate route sidecar fails independent checks."""


def validate_exit_route_full_demo_report(
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
    report = _load(report_path)
    if (
        report.get("schema_id") != REPORT_SCHEMA_ID
        or report.get("contract_version") != CONTRACT_VERSION
        or report.get("candidate_schema_id") != ROW_SCHEMA_ID
        or report.get("navigation_schema_id") != "stratai.observed_navigation_path@2"
        or report.get("navigation_contract_version") != "2.0.0"
        or report.get("training_allowed") is not False
        or report.get("product_supported") is not False
        or report.get("canonical_bundle_version") != "3.8.0_unmodified_sidecar_only"
        or not ALIAS.fullmatch(report.get("source_alias", ""))
        or not SHA256.fullmatch(report.get("source_sha256", ""))
        or report["source_alias"] != "demo-" + report["source_sha256"][:20]
    ):
        raise ExitRouteValidationError("report identity or safety metadata mismatch")
    if expected_alias is not None and report["source_alias"] != expected_alias:
        raise ExitRouteValidationError("unexpected source alias")
    if expected_sha256 is not None and report["source_sha256"] != expected_sha256:
        raise ExitRouteValidationError("unexpected source SHA-256")
    if expected_map_id is not None and report["map_id"] != expected_map_id:
        raise ExitRouteValidationError("unexpected map id")

    resolution = load_package(
        registry_root.resolve(strict=True),
        maps_root.resolve(strict=True),
        report["map_id"],
    )
    if resolution.status != "available" or resolution.package is None:
        raise ExitRouteValidationError(
            f"package failed closed: {resolution.reason_codes}"
        )
    package = resolution.package
    package_counts = validate_package(package, maps_root=maps_root)
    summary = report["package_resolution"]
    if summary != {
        "status": "available",
        "reason_codes": [],
        "contract_version": PACKAGE_CONTRACT_VERSION,
        "asset_version": package["asset_version"],
        "package_sha256": resolution.package_sha256,
        "geometry_revision": package["geometry_revision"],
        "product_supported": False,
    }:
        raise ExitRouteValidationError("package resolution summary mismatch")
    if report["navigation_contract"] != package["navigation_contract"]:
        raise ExitRouteValidationError("runtime NAV contract differs from package")
    sampling = report["sampling"]
    if (
        sampling["target_hz"] != 1
        or sampling["strategy"] != "round_anchored_right_closed_alive_players"
        or not math.isclose(
            sampling["period_ticks"], sampling["tick_rate_hz"], abs_tol=1e-9
        )
    ):
        raise ExitRouteValidationError("sampling contract mismatch")

    artifact = report["rows_artifact"]
    rows_path = (report_path.parent / artifact["path"]).resolve(strict=True)
    if (
        rows_path.parent != report_path.parent
        or artifact["path"] != ROW_FILENAME
        or rows_path.stat().st_size != artifact["bytes"]
        or _sha(rows_path) != artifact["sha256"]
    ):
        raise ExitRouteValidationError("physical artifact mismatch")
    validator = Draft202012Validator(_load(SCHEMA))
    logical_hash = hashlib.sha256()
    logical_bytes = 0
    counts: dict[str, Any] = {
        "rounds": set(),
        "records": 0,
        "samples_observed": 0,
        "samples_unavailable": 0,
        "candidate_evaluations": 0,
        "reachable": 0,
        "unreachable": 0,
        "reachability_unavailable": 0,
        "physical_path_available": 0,
        "physical_path_unavailable": 0,
        "active_weapon_scenario_available": 0,
        "active_weapon_scenario_unavailable": 0,
        "observed_pace_scenario_available": 0,
        "observed_pace_scenario_unavailable": 0,
        "tactical_judgment_unavailable": 0,
        "samples_hit": {},
        "sample_absence_reason_counts": {},
        "objective_unavailability_reason_counts": {},
        "tactical_judgment_reason_counts": {},
    }
    participants: set[str] = set()
    with gzip.open(rows_path, "rb") as stream:
        for number, line in enumerate(stream, start=1):
            if not line.endswith(b"\n"):
                raise ExitRouteValidationError(f"row {number} lacks LF")
            logical_hash.update(line)
            logical_bytes += len(line)
            row = json.loads(line)
            validator.validate(row)
            _reject_forbidden(row)
            _validate_row(row, report, package)
            _count(counts, row)
            participants.add(row["participant_alias"])
    coverage = {**counts, "rounds": len(counts["rounds"])}
    if coverage != report["coverage"]:
        raise ExitRouteValidationError("coverage does not recompute from rows")
    if (
        artifact["logical_records"] != counts["records"]
        or artifact["logical_bytes"] != logical_bytes
        or artifact["logical_sha256"] != logical_hash.hexdigest()
    ):
        raise ExitRouteValidationError("logical artifact mismatch")
    if require_observed and counts["candidate_evaluations"] == 0:
        raise ExitRouteValidationError("no candidate routes observed")
    if counts["tactical_judgment_unavailable"] != counts["candidate_evaluations"]:
        raise ExitRouteValidationError("tactical judgment crossed the factual boundary")
    return {
        "status": "passed",
        "source_alias": report["source_alias"],
        "source_sha256": report["source_sha256"],
        "map_id": report["map_id"],
        "contract_version": CONTRACT_VERSION,
        "asset_version": package["asset_version"],
        "package_sha256": resolution.package_sha256,
        "geometry_revision": package["geometry_revision"],
        "package_counts": package_counts,
        "records": counts["records"],
        "participants": len(participants),
        "candidate_evaluations": counts["candidate_evaluations"],
        "reachable": counts["reachable"],
        "unreachable": counts["unreachable"],
        "reachability_unavailable": counts["reachability_unavailable"],
        "physical_path_available": counts["physical_path_available"],
        "physical_path_unavailable": counts["physical_path_unavailable"],
        "tactical_judgment_unavailable": counts["tactical_judgment_unavailable"],
        "samples_hit": counts["samples_hit"],
        "sample_absence_reason_counts": counts["sample_absence_reason_counts"],
        "objective_unavailability_reason_counts": counts["objective_unavailability_reason_counts"],
        "tactical_judgment_reason_counts": counts["tactical_judgment_reason_counts"],
        "logical_bytes": logical_bytes,
        "logical_sha256": logical_hash.hexdigest(),
        "physical_bytes": rows_path.stat().st_size,
        "physical_sha256": artifact["sha256"],
    }


def _validate_row(
    row: dict[str, Any], report: dict[str, Any], package: dict[str, Any]
) -> None:
    if (
        row["source_alias"] != report["source_alias"]
        or row["map_id"] != report["map_id"]
        or row["availability_tick"] != row["t0_tick"]
        or not PARTICIPANT.fullmatch(row["participant_alias"])
    ):
        raise ExitRouteValidationError("row identity, tick or privacy mismatch")
    provenance = row["provenance"]
    if (
        provenance["package_sha256"] != report["package_resolution"]["package_sha256"]
        or provenance["geometry_revision"] != package["geometry_revision"]
        or provenance["nav_asset_sha256"]
        != next(
            source["sha256"]
            for source in package["source_assets"]
            if source["role"] == "nav_mesh"
        )
    ):
        raise ExitRouteValidationError("row provenance mismatch")
    selected, reason = (
        (None, "position_unavailable")
        if row["position_world"] is None
        else select_sample(package, row["position_world"])
    )
    sample = row["sample"]
    if selected is None:
        if (
            sample
            != {
                "status": "unavailable",
                "reason_codes": [reason],
                "sample_id": None,
                "sample_alias": None,
            }
            or row["candidates"]
        ):
            raise ExitRouteValidationError("unavailable sample does not recompute")
        return
    if (
        sample["status"] != "observed"
        or sample["sample_id"] != selected["id"]
        or sample["sample_alias"] != selected["alias"]
        or len(row["candidates"]) != len(selected["candidates"])
    ):
        raise ExitRouteValidationError("selected route sample does not recompute")
    for actual, expected in zip(row["candidates"], selected["candidates"], strict=True):
        if (
            actual["candidate_id"] != expected["id"]
            or actual["destination_world"] != expected["destination_world"]
        ):
            raise ExitRouteValidationError("candidate destination mismatch")
        origin = row["position_world"]
        destination = actual["destination_world"]
        direct = math.sqrt(
            sum((origin[axis] - destination[axis]) ** 2 for axis in ("x", "y", "z"))
        )
        if not math.isclose(
            actual["physical_cost"]["direct_distance"]["value_world_units"],
            direct,
            abs_tol=1e-9,
        ):
            raise ExitRouteValidationError("direct distance does not recompute")
        path_distance = actual["physical_cost"]["path_distance"]["value_world_units"]
        for scenario in actual["physical_cost"]["internal_timing"][
            "scenarios"
        ].values():
            if scenario["status"] == "estimated":
                expected_ms = (
                    path_distance / scenario["speed_world_units_per_second"] * 1000
                )
                if path_distance is None or not math.isclose(
                    scenario["value_milliseconds"], expected_ms, abs_tol=1e-6
                ):
                    raise ExitRouteValidationError(
                        "physical timing scenario does not recompute"
                    )
            elif (
                scenario["value_milliseconds"] is not None
                or scenario["speed_world_units_per_second"] is not None
            ):
                raise ExitRouteValidationError(
                    "unavailable scenario contains nominal fallback"
                )
        tactical = actual["tactical_assessment"]
        if tactical != {
            "status": "unavailable",
            "reason_codes": ["tactical_decision_context_required"],
        }:
            raise ExitRouteValidationError("tactical judgment crossed the factual boundary")


def _count(counts: dict[str, Any], row: dict[str, Any]) -> None:
    counts["rounds"].add(row["round_number"])
    counts["records"] += 1
    if row["sample"]["status"] == "observed":
        counts["samples_observed"] += 1
        sample_id = row["sample"]["sample_id"]
        counts["samples_hit"][sample_id] = counts["samples_hit"].get(sample_id, 0) + 1
    else:
        counts["samples_unavailable"] += 1
        _count_reasons(
            counts["sample_absence_reason_counts"], row["sample"]["reason_codes"]
        )
    for candidate in row["candidates"]:
        counts["candidate_evaluations"] += 1
        reachable = candidate["objective_viability"]["reachability"]["value"]
        counts[
            "reachability_unavailable"
            if reachable is None
            else "reachable"
            if reachable
            else "unreachable"
        ] += 1
        counts[
            "physical_path_available"
            if candidate["physical_cost"]["path_distance"]["value_world_units"]
            is not None
            else "physical_path_unavailable"
        ] += 1
        scenarios = candidate["physical_cost"]["internal_timing"]["scenarios"]
        counts[
            "active_weapon_scenario_available"
            if scenarios["active_weapon_uninterrupted_baseline"]["status"]
            == "estimated"
            else "active_weapon_scenario_unavailable"
        ] += 1
        counts[
            "observed_pace_scenario_available"
            if scenarios["observed_pace_projection"]["status"] == "estimated"
            else "observed_pace_scenario_unavailable"
        ] += 1
        counts["tactical_judgment_unavailable"] += 1
        _count_reasons(
            counts["objective_unavailability_reason_counts"],
            candidate["objective_viability"]["reason_codes"],
        )
        _count_reasons(
            counts["tactical_judgment_reason_counts"],
            candidate["tactical_assessment"]["reason_codes"],
        )


def _count_reasons(target: dict[str, int], reasons: list[str]) -> None:
    for reason in reasons:
        target[reason] = target.get(reason, 0) + 1


def _reject_forbidden(value: Any) -> None:
    if isinstance(value, dict):
        for key, child in value.items():
            if key.casefold() in FORBIDDEN_KEYS or key.casefold().startswith(
                ("future_", "oracle_")
            ):
                raise ExitRouteValidationError(f"forbidden field {key}")
            _reject_forbidden(child)
    elif isinstance(value, list):
        for child in value:
            _reject_forbidden(child)


def _load(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ExitRouteValidationError("JSON root is not an object")
    return value


def _sha(path: Path) -> str:
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
    args = parser.parse_args()
    try:
        result = validate_exit_route_full_demo_report(
            args.report,
            args.registry_root,
            args.maps_root,
            expected_alias=args.expected_source_alias,
            expected_sha256=args.expected_source_sha256,
            expected_map_id=args.expected_map_id,
            require_observed=args.require_observed,
        )
    except (ExitRouteValidationError, OSError, ValueError) as exc:
        parser.error(str(exc))
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
