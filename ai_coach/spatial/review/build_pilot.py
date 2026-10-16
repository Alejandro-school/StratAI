from __future__ import annotations

import argparse
import copy
import hashlib
import json
import math
import os
import stat
import tempfile
from collections.abc import Callable
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker

from ai_coach.spatial.cover.loader import load_package as load_cover_package
from ai_coach.spatial.exit_route.loader import load_package as load_exit_route_package
from ai_coach.spatial.exposure.loader import load_package as load_exposure_package
from ai_coach.spatial.review.loader import LoadedProtocol, load_protocol

BATCH_SCHEMA_ID = "stratai.spatial_human_review_batch@1"
CONTRACT_VERSION = "1.0.0"
DEFAULT_MAP_ID = "de_mirage"
DEFAULT_ITEMS_PER_FAMILY = 4
SELECTION_SEED_ID = "mirage-spatial-review-pilot-1b10-v1"
OUTPUT_NAME = "spatial_review_pilot_batch.json"
FAMILIES = ("exposure", "cover", "exit_route")

_PROJECT_ROOT = Path(__file__).resolve().parents[3]
_SCHEMA_PATH = (
    _PROJECT_ROOT / "ai_coach" / "contracts" / "spatial_human_review_batch.schema.json"
)
_DEFAULT_SPATIAL_ROOT = _PROJECT_ROOT / "ai_coach" / "spatial"
_DEFAULT_MAPS_ROOT = _PROJECT_ROOT / "backend" / "data" / "maps"


class SpatialReviewBatchError(ValueError):
    """Raised before publishing an unsafe or untrustworthy review-only batch."""


def build_pilot_batch(
    output_root: Path,
    *,
    map_id: str = DEFAULT_MAP_ID,
    items_per_family: int = DEFAULT_ITEMS_PER_FAMILY,
    review_root: Path | None = None,
    exposure_root: Path | None = None,
    cover_root: Path | None = None,
    exit_route_root: Path | None = None,
    maps_root: Path | None = None,
) -> tuple[Path, dict[str, Any]]:
    destination = _validate_empty_temporary_destination(output_root)
    if not isinstance(items_per_family, int) or isinstance(items_per_family, bool):
        raise SpatialReviewBatchError("items_per_family_invalid")
    if not 1 <= items_per_family <= 16:
        raise SpatialReviewBatchError("items_per_family_invalid")

    loaded_protocol = load_protocol(review_root)
    roots = {
        "exposure": (exposure_root or _DEFAULT_SPATIAL_ROOT / "exposure").resolve(),
        "cover": (cover_root or _DEFAULT_SPATIAL_ROOT / "cover").resolve(),
        "exit_route": (
            exit_route_root or _DEFAULT_SPATIAL_ROOT / "exit_route"
        ).resolve(),
    }
    resolved_maps_root = (maps_root or _DEFAULT_MAPS_ROOT).resolve()
    packages = _load_source_packages(roots, resolved_maps_root, map_id)
    batch = _build_batch(
        loaded_protocol,
        packages,
        roots,
        map_id=map_id,
        items_per_family=items_per_family,
    )
    _batch_validator().validate(batch)
    _require_finite(batch, "batch")
    payload = _canonical_json_bytes(batch)
    output_path = destination / OUTPUT_NAME
    temporary_path = destination / f".{OUTPUT_NAME}.tmp"
    if output_path.exists() or temporary_path.exists():
        raise SpatialReviewBatchError("output_would_overwrite")
    try:
        with temporary_path.open("xb") as handle:
            handle.write(payload)
            handle.flush()
            os.fsync(handle.fileno())
        temporary_path.replace(output_path)
    except Exception:
        if temporary_path.exists() and temporary_path.is_file():
            temporary_path.unlink()
        raise
    return output_path, batch


def _load_source_packages(
    roots: dict[str, Path], maps_root: Path, map_id: str
) -> dict[str, dict[str, Any]]:
    loaders: dict[str, Callable[..., Any]] = {
        "exposure": load_exposure_package,
        "cover": load_cover_package,
        "exit_route": load_exit_route_package,
    }
    result: dict[str, dict[str, Any]] = {}
    for family in FAMILIES:
        resolution = loaders[family](roots[family], maps_root, map_id)
        if resolution.status != "available" or resolution.package is None:
            reason = ",".join(resolution.reason_codes) or "unknown"
            raise SpatialReviewBatchError(f"{family}_package_unavailable:{reason}")
        package = resolution.package
        support = resolution.support
        if support is None or resolution.package_sha256 is None:
            raise SpatialReviewBatchError(f"{family}_support_missing")
        if (
            support.get("tactical_review_status") != "pending_review"
            or support.get("product_supported") is not False
            or support.get("training_use_authorized") is not False
            or package.get("tactical_review", {}).get("status") != "pending_review"
        ):
            raise SpatialReviewBatchError(f"{family}_support_mismatch")
        _assert_unsigned_pending_annotations(family, package)
        result[family] = {
            "package": package,
            "support": support,
            "package_sha256": resolution.package_sha256,
        }
    return result


def _assert_unsigned_pending_annotations(
    family: str, package: dict[str, Any]
) -> None:
    annotations = package.get("tactical_annotations")
    if not isinstance(annotations, list) or not annotations:
        raise SpatialReviewBatchError(f"{family}_annotations_missing")
    seen_ids: set[str] = set()
    for annotation in annotations:
        annotation_id = annotation.get("id")
        if not isinstance(annotation_id, str) or annotation_id in seen_ids:
            raise SpatialReviewBatchError(f"{family}_annotation_duplicate")
        seen_ids.add(annotation_id)
        if (
            annotation.get("classification") is not None
            or annotation.get("author_id") is not None
            or annotation.get("reviewer_id") is not None
            or annotation.get("review_status") != "pending_review"
            or annotation.get("evidence") != []
        ):
            raise SpatialReviewBatchError(f"{family}_annotation_already_populated")


def _build_batch(
    loaded_protocol: LoadedProtocol,
    packages: dict[str, dict[str, Any]],
    roots: dict[str, Path],
    *,
    map_id: str,
    items_per_family: int,
) -> dict[str, Any]:
    protocol = loaded_protocol.protocol
    questions = {item["family"]: item for item in protocol["question_sets"]}
    if set(questions) != set(FAMILIES):
        raise SpatialReviewBatchError("protocol_question_family_mismatch")

    candidates = {
        "exposure": _exposure_candidates(packages["exposure"]["package"]),
        "cover": _cover_candidates(packages["cover"]["package"]),
        "exit_route": _exit_route_candidates(
            packages["exit_route"]["package"]
        ),
    }
    selected: list[dict[str, Any]] = []
    populations: dict[str, int] = {}
    selected_counts: dict[str, int] = {}
    excluded_counts: dict[str, int] = {}
    for family in FAMILIES:
        population = candidates[family]
        if not population:
            raise SpatialReviewBatchError(f"{family}_candidate_population_empty")
        populations[family] = len(population)
        ranked = sorted(
            population,
            key=lambda item: _digest(
                "selection",
                SELECTION_SEED_ID,
                family,
                item["source_candidate_ref"],
                item["source_annotation_ref"],
                packages[family]["package_sha256"],
            ),
        )
        family_selection = ranked[: min(items_per_family, len(ranked))]
        selected.extend(family_selection)
        selected_counts[family] = len(family_selection)
        excluded_counts[family] = len(ranked) - len(family_selection)

    source_packages = [
        _source_package_summary(family, packages[family], roots[family])
        for family in FAMILIES
    ]
    source_fingerprint = "|".join(
        f"{item['family']}:{item['package_sha256']}:{item['support_manifest_sha256']}"
        for item in source_packages
    )
    batch_id = "spatial-pilot-" + _digest(
        "batch",
        loaded_protocol.protocol_sha256,
        map_id,
        str(items_per_family),
        source_fingerprint,
    )[:24]

    review_units: list[dict[str, Any]] = []
    for item in selected:
        family = item["family"]
        source = packages[family]
        source_join = "|".join(
            (
                map_id,
                family,
                item["source_sample_ref"],
                item["source_candidate_ref"],
                item["source_annotation_ref"],
                source["package_sha256"],
            )
        )
        review_unit_id = "sru-" + _digest(
            "review_unit", loaded_protocol.protocol_sha256, source_join
        )[:24]
        reviewer_view = {
            "prompt_id": questions[family]["prompt_id"],
            "question": questions[family]["question"],
            "claim_boundary": questions[family]["claim_boundary"],
            "options": copy.deepcopy(questions[family]["options"]),
            "objective_geometry": copy.deepcopy(item["objective_geometry"]),
            "observed_context": {
                "status": "unavailable",
                "reason_codes": ["package_only_pilot_no_observed_context"],
                "availability_cutoff": None,
                "facts": {},
                "source_kind": "none_demo_not_opened",
            },
            "evidence_limits": [
                "objective_geometry_only",
                "observed_context_unavailable",
                "no_oracle_future_or_outcomes",
                "not_a_safety_optimality_or_recommendation_claim",
            ],
        }
        review_units.append(
            {
                "review_unit_id": review_unit_id,
                "display_order": 0,
                "split_id": "pilot_review_only",
                "state": "pending_review",
                "audit_context": {
                    "access_scope": "coordinator_only",
                    "family": family,
                    "map_id": map_id,
                    "package_schema_id": source["package"]["schema_id"],
                    "package_sha256": source["package_sha256"],
                    "support_manifest_sha256": _sha256_file(
                        roots[family] / "support_manifest.json"
                    ),
                    "source_sample_ref": item["source_sample_ref"],
                    "source_candidate_ref": item["source_candidate_ref"],
                    "source_annotation_ref": item["source_annotation_ref"],
                    "source_group_id": "srcgrp-"
                    + _digest(
                        "source_group",
                        map_id,
                        family,
                        item["source_sample_ref"],
                    )[:24],
                },
                "reviewer_view": reviewer_view,
                "visible_evidence_sha256": hashlib.sha256(
                    _canonical_json_bytes(reviewer_view)
                ).hexdigest(),
                "review_responses": [],
                "adjudication": None,
            }
        )
    review_units.sort(
        key=lambda unit: _digest(
            "display", SELECTION_SEED_ID, unit["review_unit_id"]
        )
    )
    for index, unit in enumerate(review_units, start=1):
        unit["display_order"] = index

    total = len(review_units)
    return {
        "schema_id": BATCH_SCHEMA_ID,
        "contract_version": CONTRACT_VERSION,
        "batch_id": batch_id,
        "protocol": {
            "schema_id": protocol["schema_id"],
            "protocol_version": protocol["protocol_version"],
            "handbook_version": protocol["handbook"]["version"],
            "published_at": protocol["published_at"],
            "protocol_sha256": loaded_protocol.protocol_sha256,
            "handbook_sha256": loaded_protocol.handbook_sha256,
        },
        "map_id": map_id,
        "purpose": "mirage_spatial_human_review_pilot_input_only",
        "authorization": copy.deepcopy(protocol["authorization"]),
        "source_packages": source_packages,
        "selection": {
            "algorithm": "sha256_rank_without_replacement@1",
            "seed_id": SELECTION_SEED_ID,
            "items_per_family": items_per_family,
            "candidate_population_by_family": populations,
            "selected_by_family": selected_counts,
            "excluded_by_pilot_limit_by_family": excluded_counts,
        },
        "blinding": {
            "reviewer_visible_partition": "reviewer_view",
            "audit_partition_visible_to_reviewer": False,
            "family_order": "sha256_interleaved_deterministic@1",
            "prior_responses_visible": False,
        },
        "leakage_control": {
            "split_id": "pilot_review_only",
            "grouping_key": "map_id_family_source_sample_ref",
            "cross_split_duplicate_count": 0,
            "training_split": False,
        },
        "review_units": review_units,
        "summary": {
            "total_review_units": total,
            "families": 3,
            "family_counts": selected_counts,
            "objective_geometry_available": total,
            "observed_context_unavailable": total,
            "responses": 0,
            "reviewer_identities": 0,
            "classifications": 0,
            "adjudications": 0,
            "recommendations": 0,
        },
    }


def _source_package_summary(
    family: str, source: dict[str, Any], root: Path
) -> dict[str, Any]:
    package = source["package"]
    support = source["support"]
    return {
        "family": family,
        "schema_id": package["schema_id"],
        "contract_version": package["contract_version"],
        "asset_version": package["asset_version"],
        "map_id": package["map_id"],
        "package_sha256": source["package_sha256"],
        "support_manifest_sha256": _sha256_file(root / "support_manifest.json"),
        "geometry_revision": package["geometry_revision"],
        "coverage_declaration": package["coverage"]["declaration"],
        "full_map_coverage": package["coverage"]["full_map_coverage"],
        "tactical_review_status": support["tactical_review_status"],
        "product_supported": support["product_supported"],
        "training_use_authorized": support["training_use_authorized"],
    }


def _exposure_candidates(package: dict[str, Any]) -> list[dict[str, Any]]:
    samples = {sample["id"]: sample for sample in package["geometry_samples"]}
    sectors = {
        sector["id"]: (sample, sector)
        for sample in package["geometry_samples"]
        for sector in sample["geometry_sectors"]
    }
    result: list[dict[str, Any]] = []
    seen_refs: set[str] = set()
    for annotation in package["tactical_annotations"]:
        ref = annotation["geometry_sector_ref"]
        if ref in seen_refs or ref not in sectors:
            raise SpatialReviewBatchError("exposure_orphan_or_duplicate_reference")
        seen_refs.add(ref)
        sample, sector = sectors[ref]
        sample_id = ref.split("/", 1)[0]
        if samples.get(sample_id) is not sample:
            raise SpatialReviewBatchError("exposure_sample_reference_mismatch")
        supporting_rays = [
            copy.deepcopy(ray)
            for ray in sample["rays"]
            if _sector_contains(sector, float(ray["azimuth_degrees"]))
        ]
        if not supporting_rays:
            raise SpatialReviewBatchError("exposure_sector_without_supporting_rays")
        result.append(
            {
                "family": "exposure",
                "source_sample_ref": sample_id,
                "source_candidate_ref": ref,
                "source_annotation_ref": annotation["id"],
                "objective_geometry": {
                    "evidence_type": "angular_clearance_sector",
                    "coordinates": "source2_xyz_world_units",
                    "origin_world": copy.deepcopy(sample["origin_world"]),
                    "selection_radius_units": sample["selection_radius_units"],
                    "vertical_tolerance_units": sample["vertical_tolerance_units"],
                    "sector": {
                        "start_degrees": sector["start_degrees"],
                        "width_degrees": sector["width_degrees"],
                        "interval_semantics": "start_inclusive_end_exclusive",
                    },
                    "supporting_rays": supporting_rays,
                    "geometry_only": True,
                },
            }
        )
    if len(result) != package["coverage"]["sector_count"]:
        raise SpatialReviewBatchError("exposure_coverage_count_mismatch")
    return result


def _cover_candidates(package: dict[str, Any]) -> list[dict[str, Any]]:
    directions = {
        direction["id"]: (sample, direction)
        for sample in package["geometry_samples"]
        for direction in sample["directions"]
    }
    result: list[dict[str, Any]] = []
    seen_refs: set[str] = set()
    for annotation in package["tactical_annotations"]:
        direction_ref = annotation["geometry_direction_ref"]
        posture_id = annotation["posture_id"]
        candidate_ref = f"{direction_ref}#{posture_id}"
        if candidate_ref in seen_refs or direction_ref not in directions:
            raise SpatialReviewBatchError("cover_orphan_or_duplicate_reference")
        seen_refs.add(candidate_ref)
        sample, direction = directions[direction_ref]
        profiles = {
            profile["posture_id"]: profile for profile in direction["profiles"]
        }
        if posture_id not in profiles:
            raise SpatialReviewBatchError("cover_posture_reference_missing")
        profile = profiles[posture_id]
        result.append(
            {
                "family": "cover",
                "source_sample_ref": sample["id"],
                "source_candidate_ref": candidate_ref,
                "source_annotation_ref": annotation["id"],
                "objective_geometry": {
                    "evidence_type": "player_volume_occlusion",
                    "coordinates": "source2_xyz_world_units",
                    "origin_world": copy.deepcopy(sample["origin_world"]),
                    "selection_radius_units": sample["selection_radius_units"],
                    "vertical_tolerance_units": sample["vertical_tolerance_units"],
                    "incoming_azimuth_degrees": direction["azimuth_degrees"],
                    "posture_id": posture_id,
                    "profile": copy.deepcopy(profile),
                    "geometry_only": True,
                },
            }
        )
    expected = package["coverage"]["direction_count"] * len(
        package["coverage"]["posture_ids"]
    )
    if len(result) != expected:
        raise SpatialReviewBatchError("cover_coverage_count_mismatch")
    return result


def _exit_route_candidates(package: dict[str, Any]) -> list[dict[str, Any]]:
    candidates = {
        candidate["id"]: (sample, candidate)
        for sample in package["route_samples"]
        for candidate in sample["candidates"]
    }
    result: list[dict[str, Any]] = []
    seen_refs: set[str] = set()
    for annotation in package["tactical_annotations"]:
        ref = annotation["candidate_ref"]
        if ref in seen_refs or ref not in candidates:
            raise SpatialReviewBatchError("exit_route_orphan_or_duplicate_reference")
        seen_refs.add(ref)
        sample, candidate = candidates[ref]
        result.append(
            {
                "family": "exit_route",
                "source_sample_ref": sample["id"],
                "source_candidate_ref": ref,
                "source_annotation_ref": annotation["id"],
                "objective_geometry": {
                    "evidence_type": "directed_navigation_candidate",
                    "coordinates": "source2_xyz_world_units",
                    "origin_world": copy.deepcopy(sample["origin_world"]),
                    "destination_world": copy.deepcopy(
                        candidate["destination_world"]
                    ),
                    "selection_radius_units": sample["selection_radius_units"],
                    "vertical_tolerance_units": sample[
                        "vertical_tolerance_units"
                    ],
                    "direction": candidate["direction"],
                    "objective_connectivity": copy.deepcopy(
                        candidate["objective_connectivity"]
                    ),
                    "geometry_only": True,
                    "exact_user_facing_time_allowed": False,
                    "arrival_prediction": False,
                },
            }
        )
    if len(result) != package["coverage"]["candidate_count"]:
        raise SpatialReviewBatchError("exit_route_coverage_count_mismatch")
    return result


def _sector_contains(sector: dict[str, Any], angle: float) -> bool:
    start = float(sector["start_degrees"])
    width = float(sector["width_degrees"])
    return (angle - start) % 360.0 < width


def _validate_empty_temporary_destination(output_root: Path) -> Path:
    temporary_root = Path(tempfile.gettempdir()).resolve()
    destination = output_root.resolve(strict=False)
    try:
        destination.relative_to(temporary_root)
    except ValueError as exc:
        raise SpatialReviewBatchError("output_must_be_under_system_temp") from exc
    if destination == temporary_root:
        raise SpatialReviewBatchError("output_must_be_dedicated_directory")
    if not destination.exists() or not destination.is_dir():
        raise SpatialReviewBatchError("output_directory_must_exist")
    current = destination
    while True:
        if _is_symlink_or_reparse(current):
            raise SpatialReviewBatchError("output_contains_link_or_reparse_point")
        if current == temporary_root:
            break
        if current.parent == current:
            raise SpatialReviewBatchError("output_must_be_under_system_temp")
        current = current.parent
    if any(destination.iterdir()):
        raise SpatialReviewBatchError("output_directory_must_be_empty")
    return destination


def _is_symlink_or_reparse(path: Path) -> bool:
    metadata = path.lstat()
    if stat.S_ISLNK(metadata.st_mode):
        return True
    attributes = getattr(metadata, "st_file_attributes", 0)
    reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    return bool(attributes & reparse_flag)


def _batch_validator() -> Draft202012Validator:
    schema = json.loads(_SCHEMA_PATH.read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema, format_checker=FormatChecker())


def _canonical_json_bytes(value: Any) -> bytes:
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


def _digest(*parts: str) -> str:
    return hashlib.sha256("\x1f".join(parts).encode("utf-8")).hexdigest()


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _require_finite(value: Any, path: str) -> None:
    if value is None or isinstance(value, (str, bool, int)):
        return
    if isinstance(value, float):
        if not math.isfinite(value):
            raise SpatialReviewBatchError(f"{path}: non-finite number")
        return
    if isinstance(value, list):
        for index, item in enumerate(value):
            _require_finite(item, f"{path}[{index}]")
        return
    if isinstance(value, dict):
        for key, item in value.items():
            _require_finite(item, f"{path}.{key}")
        return
    raise SpatialReviewBatchError(f"{path}: unsupported value")


def _argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Build a deterministic package-only spatial review pilot"
    )
    parser.add_argument("--output-root", type=Path, required=True)
    parser.add_argument("--map-id", default=DEFAULT_MAP_ID)
    parser.add_argument("--items-per-family", type=int, default=DEFAULT_ITEMS_PER_FAMILY)
    parser.add_argument("--review-root", type=Path)
    parser.add_argument("--exposure-root", type=Path)
    parser.add_argument("--cover-root", type=Path)
    parser.add_argument("--exit-route-root", type=Path)
    parser.add_argument("--maps-root", type=Path)
    return parser


def main() -> int:
    args = _argument_parser().parse_args()
    output_path, batch = build_pilot_batch(
        args.output_root,
        map_id=args.map_id,
        items_per_family=args.items_per_family,
        review_root=args.review_root,
        exposure_root=args.exposure_root,
        cover_root=args.cover_root,
        exit_route_root=args.exit_route_root,
        maps_root=args.maps_root,
    )
    print(
        json.dumps(
            {
                "output": output_path.name,
                "sha256": _sha256_file(output_path),
                "summary": batch["summary"],
            },
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
