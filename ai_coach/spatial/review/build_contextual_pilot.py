from __future__ import annotations

import argparse
import copy
import gzip
import hashlib
import json
import math
import stat
import subprocess
import tempfile
from collections.abc import Callable, Iterator
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator

from ai_coach.contracts.validate_cover_full_demo import (
    validate_cover_full_demo_report,
)
from ai_coach.contracts.validate_exit_route_full_demo import (
    validate_exit_route_full_demo_report,
)
from ai_coach.contracts.validate_exposure_angle_full_demo import (
    validate_exposure_angle_full_demo_report,
)
from ai_coach.spatial.cover.loader import load_package as load_cover_package
from ai_coach.spatial.exit_route.loader import (
    load_package as load_exit_route_package,
)
from ai_coach.spatial.exposure.loader import (
    load_package as load_exposure_package,
)

ROOT = Path(__file__).resolve().parents[3]
PROTOCOL_ROOT = Path(__file__).resolve().parent / "protocols" / "2.0.0"
PROTOCOL_PATH = PROTOCOL_ROOT / "protocol.json"
HANDBOOK_PATH = PROTOCOL_ROOT / "HANDBOOK.md"
PROTOCOL_SCHEMA = ROOT / "ai_coach" / "contracts" / "spatial_context_review_protocol.schema.json"
BATCH_SCHEMA = ROOT / "ai_coach" / "contracts" / "spatial_context_review_batch.schema.json"
RADAR_PATH = ROOT / "frontend" / "public" / "maps" / "de_mirage_radar_psd.webp"
MAPS_ROOT = ROOT / "backend" / "data" / "maps"
GO_SERVICE_ROOT = ROOT / "backend" / "go-service"
EXPOSURE_ROOT = ROOT / "ai_coach" / "spatial" / "exposure"
COVER_ROOT = ROOT / "ai_coach" / "spatial" / "cover"
EXIT_ROUTE_ROOT = ROOT / "ai_coach" / "spatial" / "exit_route"

BATCH_FILENAME = "spatial_context_review_batch.json"
TRACE_REQUEST_FILENAME = "route_trace_request.json"
TRACE_OUTPUT_FILENAME = "route_traces.json"
SEED_ID = "mirage-contextual-review-pilot-repair-v2"
FAMILIES = ("exposure", "cover", "exit_route")
ITEMS_PER_FAMILY = 4


class ContextualPilotError(ValueError):
    """Raised when a situated review batch cannot be built safely."""


TraceRunner = Callable[[Path, Path], None]


def build_contextual_pilot(
    exposure_report_path: Path,
    cover_report_path: Path,
    exit_route_report_path: Path,
    output_root: Path,
    *,
    maps_root: Path = MAPS_ROOT,
    exposure_root: Path = EXPOSURE_ROOT,
    cover_root: Path = COVER_ROOT,
    exit_route_root: Path = EXIT_ROUTE_ROOT,
    radar_path: Path = RADAR_PATH,
    go_service_root: Path = GO_SERVICE_ROOT,
    trace_runner: TraceRunner | None = None,
) -> tuple[Path, dict[str, Any]]:
    destination = _validate_empty_temporary_destination(output_root)
    protocol = _load_and_validate_protocol()
    reports = _validate_and_load_reports(
        exposure_report_path,
        cover_report_path,
        exit_route_report_path,
        maps_root=maps_root,
        exposure_root=exposure_root,
        cover_root=cover_root,
        exit_route_root=exit_route_root,
    )
    source_alias, source_sha256, tick_rate = _shared_source_identity(reports)
    packages = _load_packages(
        maps_root, exposure_root, cover_root, exit_route_root
    )

    questions = {item["family"]: item for item in protocol["question_sets"]}
    candidates = {
        "exposure": _exposure_candidates(
            exposure_report_path, reports["exposure"], packages["exposure"]
        ),
        "cover": _cover_candidates(
            cover_report_path, reports["cover"], packages["cover"]
        ),
        "exit_route": _exit_route_candidates(
            exit_route_report_path,
            reports["exit_route"],
            packages["exit_route"],
        ),
    }
    selected = {
        family: _select_diverse(family, values)
        for family, values in candidates.items()
    }

    evidence_dir = destination / "coordinator_evidence"
    evidence_dir.mkdir()
    trace_request_path = evidence_dir / TRACE_REQUEST_FILENAME
    trace_output_path = evidence_dir / TRACE_OUTPUT_FILENAME
    trace_request = {
        "schema_id": "stratai.spatial_review_route_trace_request@1",
        "map_id": "de_mirage",
        "routes": [
            {
                "trace_id": item["trace_id"],
                "origin_world": item["reviewer_view"]["objective_geometry"][
                    "origin_world"
                ],
                "destination_world": item["reviewer_view"][
                    "objective_geometry"
                ]["destination_world"],
            }
            for item in selected["exit_route"]
        ],
    }
    _write_new(trace_request_path, _canonical_bytes(trace_request))
    if trace_runner is None:
        _run_trace_builder(
            trace_request_path,
            trace_output_path,
            maps_root=maps_root,
            go_service_root=go_service_root,
        )
    else:
        trace_runner(trace_request_path, trace_output_path)
    traces = _load_json(trace_output_path)
    _attach_and_validate_traces(selected["exit_route"], traces)

    radar = radar_path.resolve(strict=True)
    radar_sha256 = _sha256_file(radar)
    protocol_sha256 = _sha256_file(PROTOCOL_PATH)
    handbook_sha256 = _sha256_file(HANDBOOK_PATH)
    selected_joins = [
        item["source_join"] for family in FAMILIES for item in selected[family]
    ]
    batch_id = "spatial-context-pilot-" + _hash_parts(
        "batch",
        protocol_sha256,
        source_sha256,
        radar_sha256,
        *sorted(selected_joins),
    )[:24]

    units: list[dict[str, Any]] = []
    for family in FAMILIES:
        question = questions[family]
        for item in selected[family]:
            view = item["reviewer_view"]
            view.update(
                {
                    "prompt_id": question["prompt_id"],
                    "question": question["question"],
                    "claim_boundary": question["claim_boundary"],
                    "options": copy.deepcopy(question["options"]),
                    "evidence_limits": [
                        "approved_demo_context_up_to_t0_only",
                        "registered_mirage_radar_and_objective_geometry",
                        "no_future_outcome_or_oracle_enemy_state",
                        "not_a_safety_optimality_or_recommendation_claim",
                    ],
                }
            )
            unit_id = "scu-" + _hash_parts(
                "review_unit", protocol_sha256, item["source_join"]
            )[:24]
            visible_hash = hashlib.sha256(
                _canonical_bytes(
                    {
                        "reviewer_view": view,
                        "radar_sha256": radar_sha256,
                        "render_contract": "mirage_radar_context_overlay@2",
                    }
                )
            ).hexdigest()
            units.append(
                {
                    "review_unit_id": unit_id,
                    "display_order": 0,
                    "state": "pending_review",
                    "audit_context": {
                        "access_scope": "coordinator_only",
                        "family": family,
                        "source_join": item["source_join"],
                        "package_sha256": packages[family]["package_sha256"],
                        "sidecar_row_sha256": item["sidecar_row_sha256"],
                    },
                    "reviewer_view": view,
                    "visible_evidence_sha256": visible_hash,
                    "review_responses": [],
                    "adjudication": None,
                }
            )
    units.sort(key=lambda unit: _hash_parts("display", SEED_ID, unit["review_unit_id"]))
    for index, unit in enumerate(units, start=1):
        unit["display_order"] = index

    batch = {
        "schema_id": "stratai.spatial_context_review_batch@2",
        "contract_version": "2.0.0",
        "batch_id": batch_id,
        "protocol": {
            "schema_id": protocol["schema_id"],
            "protocol_version": protocol["protocol_version"],
            "handbook_version": protocol["handbook"]["version"],
            "published_at": protocol["published_at"],
            "protocol_sha256": protocol_sha256,
            "handbook_sha256": handbook_sha256,
        },
        "source_demo": {
            "opaque_alias": source_alias,
            "sha256": source_sha256,
            "map_id": "de_mirage",
            "tick_rate_hz": tick_rate,
            "sidecar_report_sha256": {
                family: _sha256_file(path.resolve(strict=True))
                for family, path in {
                    "exposure": exposure_report_path,
                    "cover": cover_report_path,
                    "exit_route": exit_route_report_path,
                }.items()
            },
        },
        "presentation": {
            "map_label": "Mirage",
            "radar_asset": {
                "relative_path": "frontend/public/maps/de_mirage_radar_psd.webp",
                "sha256": radar_sha256,
                "media_type": "image/webp",
                "source_pixels": 4096,
            },
            "coordinate_transform": {
                "pos_x": -3230,
                "pos_y": 1713,
                "scale": 5,
                "logical_pixels": 1024,
                "formula": "pixel_x=(world_x-pos_x)/scale;pixel_y=(pos_y-world_y)/scale",
            },
            "render_contract": "mirage_radar_context_overlay@2",
        },
        "source_packages": [
            _package_summary(family, packages[family]) for family in FAMILIES
        ],
        "selection": {
            "algorithm": "sha256_rank_distinct_sample_context@2",
            "seed_id": SEED_ID,
            "items_per_family": ITEMS_PER_FAMILY,
            "selected_by_family": {family: 4 for family in FAMILIES},
        },
        "blinding": {
            "reviewer_visible_partition": "reviewer_view",
            "audit_partition_visible_to_reviewer": False,
            "prior_responses_visible": False,
        },
        "authorization": copy.deepcopy(protocol["authorization"]),
        "review_units": units,
        "summary": {
            "total_review_units": 12,
            "family_counts": {family: 4 for family in FAMILIES},
            "observed_context_available": 12,
            "radar_registered": 12,
            "route_polylines_available": 4,
            "responses": 0,
            "adjudications": 0,
            "training_allowed": False,
        },
    }
    _validate_schema(batch, BATCH_SCHEMA, "contextual batch")
    _assert_no_forbidden_visible_data(batch)
    batch_path = destination / BATCH_FILENAME
    _write_new(batch_path, _canonical_bytes(batch))
    return batch_path, batch


def _validate_and_load_reports(
    exposure_path: Path,
    cover_path: Path,
    exit_path: Path,
    *,
    maps_root: Path,
    exposure_root: Path,
    cover_root: Path,
    exit_route_root: Path,
) -> dict[str, dict[str, Any]]:
    exposure_path = exposure_path.resolve(strict=True)
    cover_path = cover_path.resolve(strict=True)
    exit_path = exit_path.resolve(strict=True)
    validate_exposure_angle_full_demo_report(
        exposure_path,
        registry_root=exposure_root,
        maps_root=maps_root,
        expected_map_id="de_mirage",
        require_geometry_observed=True,
        require_geometry_unavailable=True,
        require_pending_tactical_review=True,
    )
    validate_cover_full_demo_report(
        cover_path,
        cover_root,
        maps_root,
        expected_map_id="de_mirage",
        require_observed=True,
    )
    validate_exit_route_full_demo_report(
        exit_path,
        exit_route_root,
        maps_root,
        expected_map_id="de_mirage",
        require_observed=True,
    )
    return {
        "exposure": _load_json(exposure_path),
        "cover": _load_json(cover_path),
        "exit_route": _load_json(exit_path),
    }


def _shared_source_identity(
    reports: dict[str, dict[str, Any]],
) -> tuple[str, str, float]:
    identities = {
        (
            report["source_alias"],
            report["source_sha256"],
            report["map_id"],
            float(report["sampling"]["tick_rate_hz"]),
        )
        for report in reports.values()
    }
    if len(identities) != 1:
        raise ContextualPilotError("sidecars_do_not_share_one_demo")
    alias, sha256, map_id, tick_rate = identities.pop()
    if map_id != "de_mirage" or alias != "demo-" + sha256[:20]:
        raise ContextualPilotError("source_demo_identity_invalid")
    return alias, sha256, tick_rate


def _load_packages(
    maps_root: Path,
    exposure_root: Path,
    cover_root: Path,
    exit_route_root: Path,
) -> dict[str, dict[str, Any]]:
    loaders = {
        "exposure": (load_exposure_package, exposure_root),
        "cover": (load_cover_package, cover_root),
        "exit_route": (load_exit_route_package, exit_route_root),
    }
    result: dict[str, dict[str, Any]] = {}
    for family, (loader, root) in loaders.items():
        resolution = loader(root, maps_root, "de_mirage")
        if (
            resolution.status != "available"
            or resolution.package is None
            or resolution.package_sha256 is None
        ):
            raise ContextualPilotError(f"{family}_package_unavailable")
        result[family] = {
            "package": resolution.package,
            "package_sha256": resolution.package_sha256,
            "support_manifest_sha256": _sha256_file(root / "support_manifest.json"),
        }
    return result


def _exposure_candidates(
    report_path: Path, report: dict[str, Any], source: dict[str, Any]
) -> list[dict[str, Any]]:
    package = source["package"]
    samples = {sample["id"]: sample for sample in package["geometry_samples"]}
    result: list[dict[str, Any]] = []
    for row, row_hash in _iter_rows(report_path, report):
        geometry = row["geometry_clearance"]
        if (
            geometry["status"] != "observed"
            or row["view_yaw_degrees"] is None
            or geometry["sample_id"] not in samples
        ):
            continue
        sample = samples[geometry["sample_id"]]
        sectors = {sector["id"]: sector for sector in sample["geometry_sectors"]}
        relative = {
            sector["id"]: sector
            for sector in geometry["view_relative_geometry_sectors"]
        }
        for observed_sector in geometry["absolute_geometry_sectors"]:
            sector_id = observed_sector["id"]
            if sector_id not in sectors or sector_id not in relative:
                raise ContextualPilotError("exposure_sidecar_package_join_failed")
            sector = sectors[sector_id]
            rays = [
                copy.deepcopy(ray)
                for ray in sample["rays"]
                if _angle_in_sector(
                    float(ray["azimuth_degrees"]),
                    float(sector["start_degrees"]),
                    float(sector["width_degrees"]),
                )
            ]
            source_join = _source_join(row, "exposure", sector_id)
            result.append(
                {
                    "sample_key": sample["id"],
                    "stratum": str(geometry["view_inside_geometry_sector"]),
                    "source_join": source_join,
                    "sidecar_row_sha256": row_hash,
                    "reviewer_view": {
                        "case_kind": "exposure",
                        "observed_context": _observed_context(row),
                        "objective_geometry": {
                            "evidence_type": "situated_angular_clearance_sector",
                            "origin_world": copy.deepcopy(row["position_world"]),
                            "sample_origin_world": copy.deepcopy(sample["origin_world"]),
                            "sample_horizontal_offset_units": geometry[
                                "horizontal_distance_units"
                            ],
                            "sample_vertical_offset_units": geometry[
                                "vertical_distance_units"
                            ],
                            "view_yaw_degrees": row["view_yaw_degrees"],
                            "ray_limit_world_units": package["resolution"][
                                "maximum_ray_distance"
                            ],
                            "minimum_clearance_distance_units": package[
                                "resolution"
                            ]["minimum_clearance_distance"],
                            "sector": copy.deepcopy(sector),
                            "view_relative_sector": copy.deepcopy(relative[sector_id]),
                            "supporting_rays": rays,
                        },
                    },
                }
            )
    return result


def _cover_candidates(
    report_path: Path, report: dict[str, Any], source: dict[str, Any]
) -> list[dict[str, Any]]:
    package = source["package"]
    samples = {sample["id"]: sample for sample in package["geometry_samples"]}
    directions = {
        direction["id"]: (sample, direction)
        for sample in package["geometry_samples"]
        for direction in sample["directions"]
    }
    result: list[dict[str, Any]] = []
    for row, row_hash in _iter_rows(report_path, report):
        geometry = row["geometry_occlusion"]
        if (
            geometry["status"] != "observed"
            or row["position_world"] is None
            or row["view_yaw_degrees"] is None
            or geometry["sample_id"] not in samples
        ):
            continue
        for observed in geometry["directions"]:
            ref = observed["geometry_direction_ref"]
            if ref not in directions:
                raise ContextualPilotError("cover_sidecar_package_join_failed")
            sample, direction = directions[ref]
            profiles = {
                profile["posture_id"]: profile
                for profile in direction["profiles"]
            }
            posture = row["posture_id"]
            if posture not in profiles:
                raise ContextualPilotError("cover_posture_profile_missing")
            profile = profiles[posture]
            if (
                profile["occluded_probe_count"]
                != observed["occluded_probe_count"]
                or profile["complete_volume_occlusion"]
                != observed["complete_volume_occlusion"]
            ):
                raise ContextualPilotError("cover_sidecar_measurement_mismatch")
            count = int(profile["occluded_probe_count"])
            stratum = "none" if count == 0 else "complete" if count == 9 else "partial"
            source_join = _source_join(row, "cover", f"{ref}#{posture}")
            result.append(
                {
                    "sample_key": sample["id"],
                    "stratum": stratum,
                    "source_join": source_join,
                    "sidecar_row_sha256": row_hash,
                    "reviewer_view": {
                        "case_kind": "cover",
                        "observed_context": _observed_context(row),
                        "objective_geometry": {
                            "evidence_type": "situated_player_volume_occlusion",
                            "origin_world": copy.deepcopy(row["position_world"]),
                            "sample_origin_world": copy.deepcopy(sample["origin_world"]),
                            "view_yaw_degrees": row["view_yaw_degrees"],
                            "incoming_azimuth_degrees": observed[
                                "absolute_azimuth_degrees"
                            ],
                            "view_relative_azimuth_degrees": observed[
                                "view_relative_azimuth_degrees"
                            ],
                            "protection_distance_world_units": package[
                                "resolution"
                            ]["protection_distance_units"],
                            "maximum_ray_distance_world_units": package[
                                "resolution"
                            ]["maximum_ray_distance_units"],
                            "posture_id": posture,
                            "profile": copy.deepcopy(profile),
                        },
                    },
                }
            )
    return result


def _exit_route_candidates(
    report_path: Path, report: dict[str, Any], source: dict[str, Any]
) -> list[dict[str, Any]]:
    package = source["package"]
    samples = {sample["id"]: sample for sample in package["route_samples"]}
    result: list[dict[str, Any]] = []
    for row, row_hash in _iter_rows(report_path, report):
        if (
            row["sample"]["status"] != "observed"
            or row["position_world"] is None
            or row["sample"]["sample_id"] not in samples
        ):
            continue
        sample = samples[row["sample"]["sample_id"]]
        known = {
            candidate["id"]: candidate for candidate in sample["candidates"]
        }
        for candidate in row["candidates"]:
            candidate_id = candidate["candidate_id"]
            if candidate_id not in known:
                raise ContextualPilotError("exit_route_sidecar_package_join_failed")
            reachability = candidate["objective_viability"]["reachability"]["value"]
            path_distance = candidate["physical_cost"]["path_distance"][
                "value_world_units"
            ]
            if reachability is not True or path_distance is None:
                continue
            source_join = _source_join(row, "exit_route", candidate_id)
            trace_id = "trace-" + _hash_parts("trace", source_join)[:24]
            posture = row["posture_context"]
            context = _observed_context(row)
            context["posture_id"] = (
                "crouching" if posture.get("is_ducking") else "standing"
                if posture.get("is_ducking") is not None
                else None
            )
            context["active_weapon"] = posture.get("active_weapon")
            context["horizontal_speed_world_units_per_second"] = posture.get(
                "horizontal_speed_world_units_per_second"
            )
            result.append(
                {
                    "sample_key": sample["id"],
                    "stratum": "reachable",
                    "trace_id": trace_id,
                    "source_join": source_join,
                    "sidecar_row_sha256": row_hash,
                    "reviewer_view": {
                        "case_kind": "exit_route",
                        "observed_context": context,
                        "objective_geometry": {
                            "evidence_type": "situated_directed_navigation_path",
                            "origin_world": copy.deepcopy(row["position_world"]),
                            "sample_origin_world": copy.deepcopy(sample["origin_world"]),
                            "destination_world": copy.deepcopy(
                                candidate["destination_world"]
                            ),
                            "direction": "origin_to_destination",
                            "objective_connectivity": {
                                "reachable": reachability,
                                "direct_distance_world_units": candidate[
                                    "physical_cost"
                                ]["direct_distance"]["value_world_units"],
                                "path_distance_world_units": path_distance,
                                "origin_association_world_units": candidate[
                                    "objective_viability"
                                ]["origin_association_world_units"],
                                "destination_association_world_units": candidate[
                                    "objective_viability"
                                ]["destination_association_world_units"],
                            },
                            "trace_id": trace_id,
                            "path_points_world": [],
                            "trace_contract": "pending",
                        },
                    },
                }
            )
    return result


def _select_diverse(family: str, candidates: list[dict[str, Any]]) -> list[dict[str, Any]]:
    if not candidates:
        raise ContextualPilotError(f"{family}_has_no_reviewable_candidates")
    ranked = sorted(
        candidates,
        key=lambda item: _hash_parts("select", SEED_ID, family, item["source_join"]),
    )
    selected: list[dict[str, Any]] = []
    used_samples: set[str] = set()
    used_strata: set[str] = set()
    for require_new_stratum in (True, False):
        for item in ranked:
            if item["sample_key"] in used_samples or item in selected:
                continue
            if require_new_stratum and item["stratum"] in used_strata:
                continue
            selected.append(copy.deepcopy(item))
            used_samples.add(item["sample_key"])
            used_strata.add(item["stratum"])
            if len(selected) == ITEMS_PER_FAMILY:
                return selected
    raise ContextualPilotError(f"{family}_lacks_four_distinct_observed_samples")


def _attach_and_validate_traces(
    selected: list[dict[str, Any]], traces: dict[str, Any]
) -> None:
    if (
        traces.get("schema_id") != "stratai.spatial_review_route_traces@1"
        or traces.get("map_id") != "de_mirage"
        or traces.get("graph_contract_version")
        != "nav_area_connections_with_polygons@2"
        or traces.get("shortest_path_algorithm")
        != "directed_dijkstra_stable_ties@1"
    ):
        raise ContextualPilotError("route_trace_identity_invalid")
    by_id = {item["trace_id"]: item for item in traces.get("routes", [])}
    if set(by_id) != {item["trace_id"] for item in selected}:
        raise ContextualPilotError("route_trace_join_mismatch")
    for item in selected:
        geometry = item["reviewer_view"]["objective_geometry"]
        trace = by_id[item["trace_id"]]
        measured = trace["measurement"]
        expected = geometry["objective_connectivity"]
        for key in (
            "direct_distance_world_units",
            "path_distance_world_units",
            "origin_association_world_units",
            "destination_association_world_units",
        ):
            if not math.isclose(
                float(measured[key]), float(expected[key]), abs_tol=1e-6
            ):
                raise ContextualPilotError("route_trace_measurement_mismatch")
        points = trace.get("path_points_world")
        if not isinstance(points, list) or len(points) < 2:
            raise ContextualPilotError("route_trace_polyline_missing")
        if points[0] != geometry["origin_world"] or points[-1] != geometry[
            "destination_world"
        ]:
            raise ContextualPilotError("route_trace_endpoints_mismatch")
        geometry["path_points_world"] = copy.deepcopy(points)
        geometry["trace_contract"] = "nav_area_centres_polyline@1"


def _run_trace_builder(
    request_path: Path,
    output_path: Path,
    *,
    maps_root: Path,
    go_service_root: Path,
) -> None:
    command = [
        "go",
        "run",
        "./cmd/spatial-review-paths",
        "--request",
        str(request_path.resolve(strict=True)),
        "--maps-root",
        str(maps_root.resolve(strict=True)),
        "--output",
        str(output_path.resolve(strict=False)),
    ]
    completed = subprocess.run(
        command,
        cwd=go_service_root.resolve(strict=True),
        check=False,
        capture_output=True,
        text=True,
        timeout=120,
    )
    if completed.returncode != 0:
        raise ContextualPilotError(
            "route_trace_builder_failed:" + completed.stderr.strip()
        )


def _iter_rows(
    report_path: Path, report: dict[str, Any]
) -> Iterator[tuple[dict[str, Any], str]]:
    rows_path = (
        report_path.resolve(strict=True).parent / report["rows_artifact"]["path"]
    ).resolve(strict=True)
    with gzip.open(rows_path, "rb") as handle:
        for line in handle:
            if not line.endswith(b"\n"):
                raise ContextualPilotError("sidecar_row_missing_newline")
            try:
                row = json.loads(line)
            except (UnicodeDecodeError, json.JSONDecodeError) as exc:
                raise ContextualPilotError("sidecar_row_invalid_json") from exc
            yield row, hashlib.sha256(line).hexdigest()


def _observed_context(row: dict[str, Any]) -> dict[str, Any]:
    if row["availability_tick"] > row["t0_tick"]:
        raise ContextualPilotError("future_context_detected")
    posture = row.get("posture_id")
    return {
        "status": "observed",
        "source_kind": "approved_demo_causal_sidecar",
        "round_number": row["round_number"],
        "t0_tick": row["t0_tick"],
        "availability_tick": row["availability_tick"],
        "participant_alias": row["participant_alias"],
        "position_world": copy.deepcopy(row["position_world"]),
        "view_yaw_degrees": row.get("view_yaw_degrees"),
        "posture_id": posture,
        "active_weapon": None,
        "horizontal_speed_world_units_per_second": None,
    }


def _source_join(row: dict[str, Any], family: str, candidate_ref: str) -> str:
    return "|".join(
        [
            row["source_alias"],
            family,
            f"round:{row['round_number']}",
            f"tick:{row['t0_tick']}",
            row["participant_alias"],
            candidate_ref,
        ]
    )


def _package_summary(family: str, source: dict[str, Any]) -> dict[str, Any]:
    package = source["package"]
    return {
        "family": family,
        "schema_id": package["schema_id"],
        "asset_version": package["asset_version"],
        "package_sha256": source["package_sha256"],
        "support_manifest_sha256": source["support_manifest_sha256"],
        "geometry_revision": package["geometry_revision"],
    }


def _angle_in_sector(angle: float, start: float, width: float) -> bool:
    return (angle - start) % 360.0 < width


def _load_and_validate_protocol() -> dict[str, Any]:
    protocol = _load_json(PROTOCOL_PATH)
    if _sha256_file(HANDBOOK_PATH) != protocol["handbook"]["sha256"]:
        raise ContextualPilotError("handbook_hash_mismatch")
    _validate_schema(protocol, PROTOCOL_SCHEMA, "contextual protocol")
    return protocol


def _validate_schema(value: Any, schema_path: Path, label: str) -> None:
    schema = _load_json(schema_path)
    validator = Draft202012Validator(schema)
    errors = sorted(validator.iter_errors(value), key=lambda error: list(error.path))
    if errors:
        first = errors[0]
        path = ".".join(str(part) for part in first.path) or "$"
        raise ContextualPilotError(f"{label}_schema:{path}:{first.message}")


def _assert_no_forbidden_visible_data(batch: dict[str, Any]) -> None:
    forbidden_keys = {
        "steam_id",
        "steamid",
        "name",
        "winner",
        "outcome",
        "future",
        "enemy_position",
        "audit_context",
    }
    for unit in batch["review_units"]:
        _scan_visible(unit["reviewer_view"], forbidden_keys, "reviewer_view")


def _scan_visible(value: Any, forbidden: set[str], path: str) -> None:
    if isinstance(value, dict):
        overlap = forbidden.intersection(key.lower() for key in value)
        if overlap:
            raise ContextualPilotError(f"visible_forbidden_keys:{path}:{sorted(overlap)}")
        for key, item in value.items():
            _scan_visible(item, forbidden, f"{path}.{key}")
    elif isinstance(value, list):
        for index, item in enumerate(value):
            _scan_visible(item, forbidden, f"{path}[{index}]")
    elif isinstance(value, str):
        if ":\\" in value or value.startswith("/") or "steamcommunity" in value.lower():
            raise ContextualPilotError(f"visible_path_or_identity:{path}")


def _validate_empty_temporary_destination(output_root: Path) -> Path:
    temporary_root = Path(tempfile.gettempdir()).resolve()
    destination = output_root.resolve(strict=False)
    try:
        destination.relative_to(temporary_root)
    except ValueError as exc:
        raise ContextualPilotError("output_must_be_under_system_temp") from exc
    if destination == temporary_root or not destination.exists() or not destination.is_dir():
        raise ContextualPilotError("output_must_be_existing_dedicated_directory")
    current = destination
    while True:
        if _is_symlink_or_reparse(current):
            raise ContextualPilotError("output_contains_link_or_reparse_point")
        if current == temporary_root:
            break
        if current.parent == current:
            raise ContextualPilotError("output_escaped_system_temp")
        current = current.parent
    if any(destination.iterdir()):
        raise ContextualPilotError("output_directory_must_be_empty")
    return destination


def _is_symlink_or_reparse(path: Path) -> bool:
    metadata = path.lstat()
    attributes = getattr(metadata, "st_file_attributes", 0)
    reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    return stat.S_ISLNK(metadata.st_mode) or bool(attributes & reparse_flag)


def _load_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.resolve(strict=True).read_text(encoding="utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ContextualPilotError(f"invalid_json:{path.name}") from exc
    if not isinstance(value, dict):
        raise ContextualPilotError(f"json_object_required:{path.name}")
    return value


def _canonical_bytes(value: Any) -> bytes:
    return (
        json.dumps(
            value,
            ensure_ascii=False,
            sort_keys=True,
            separators=(",", ":"),
            allow_nan=False,
        )
        + "\n"
    ).encode("utf-8")


def _hash_parts(*parts: str) -> str:
    return hashlib.sha256("\x1f".join(parts).encode("utf-8")).hexdigest()


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.resolve(strict=True).open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _write_new(path: Path, payload: bytes) -> None:
    with path.open("xb") as handle:
        handle.write(payload)
        handle.flush()


def _argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Build a demo-contextual spatial human-review pilot"
    )
    parser.add_argument("--exposure-report", type=Path, required=True)
    parser.add_argument("--cover-report", type=Path, required=True)
    parser.add_argument("--exit-route-report", type=Path, required=True)
    parser.add_argument("--output-root", type=Path, required=True)
    return parser


def main() -> int:
    args = _argument_parser().parse_args()
    path, batch = build_contextual_pilot(
        args.exposure_report,
        args.cover_report,
        args.exit_route_report,
        args.output_root,
    )
    print(
        json.dumps(
            {
                "batch": str(path),
                "batch_id": batch["batch_id"],
                "batch_sha256": _sha256_file(path),
                "review_units": len(batch["review_units"]),
            },
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
