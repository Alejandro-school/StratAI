from __future__ import annotations

import argparse
import copy
import hashlib
import json
import math
import re
import stat
import tempfile
from collections.abc import Callable
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker

from ai_coach.spatial.cover.loader import load_package as load_cover_package
from ai_coach.spatial.exit_route.loader import load_package as load_exit_route_package
from ai_coach.spatial.exposure.loader import load_package as load_exposure_package
from ai_coach.spatial.review.loader import load_protocol

_PROJECT_ROOT = Path(__file__).resolve().parents[2]
_BATCH_SCHEMA = Path(__file__).resolve().with_name(
    "spatial_human_review_batch.schema.json"
)
_SPATIAL_ROOT = _PROJECT_ROOT / "ai_coach" / "spatial"
_MAPS_ROOT = _PROJECT_ROOT / "backend" / "data" / "maps"
_FAMILIES = ("exposure", "cover", "exit_route")
_SEED_ID = "mirage-spatial-review-pilot-1b10-v1"
_ABSOLUTE_PATH = re.compile(r"^(?:[A-Za-z]:[\\/]|/home/|/Users/|/tmp/|\\\\)")


class SpatialReviewPilotValidationError(ValueError):
    """Raised when a review pilot cannot be independently certified."""


def validate_spatial_review_pilot(
    batch_path: Path,
    *,
    review_root: Path | None = None,
    exposure_root: Path | None = None,
    cover_root: Path | None = None,
    exit_route_root: Path | None = None,
    maps_root: Path | None = None,
) -> dict[str, Any]:
    batch_path = _validate_temporary_batch_path(batch_path)
    payload = batch_path.read_bytes()
    batch = _decode_json(payload, "batch")
    schema = _decode_json(_BATCH_SCHEMA.read_bytes(), "batch_schema")
    Draft202012Validator.check_schema(schema)
    Draft202012Validator(
        schema, format_checker=FormatChecker()
    ).validate(batch)
    _require_finite(batch, "batch")

    loaded_protocol = load_protocol(review_root)
    protocol = loaded_protocol.protocol
    expected_protocol = {
        "schema_id": protocol["schema_id"],
        "protocol_version": protocol["protocol_version"],
        "handbook_version": protocol["handbook"]["version"],
        "published_at": protocol["published_at"],
        "protocol_sha256": loaded_protocol.protocol_sha256,
        "handbook_sha256": loaded_protocol.handbook_sha256,
    }
    _equal(batch["protocol"], expected_protocol, "protocol provenance")
    _equal(batch["authorization"], protocol["authorization"], "authorization")

    roots = {
        "exposure": (exposure_root or _SPATIAL_ROOT / "exposure").resolve(),
        "cover": (cover_root or _SPATIAL_ROOT / "cover").resolve(),
        "exit_route": (exit_route_root or _SPATIAL_ROOT / "exit_route").resolve(),
    }
    resolved_maps_root = (maps_root or _MAPS_ROOT).resolve()
    sources = _load_sources(roots, resolved_maps_root, batch["map_id"])
    expected_source_packages = [
        _source_summary(family, sources[family], roots[family])
        for family in _FAMILIES
    ]
    _equal(batch["source_packages"], expected_source_packages, "source packages")

    inventories = {
        "exposure": _inventory_exposure(sources["exposure"]["package"]),
        "cover": _inventory_cover(sources["cover"]["package"]),
        "exit_route": _inventory_exit_route(sources["exit_route"]["package"]),
    }
    items_per_family = batch["selection"]["items_per_family"]
    expected_selected: dict[str, dict[str, dict[str, Any]]] = {}
    populations: dict[str, int] = {}
    selected_counts: dict[str, int] = {}
    excluded_counts: dict[str, int] = {}
    for family in _FAMILIES:
        inventory = inventories[family]
        populations[family] = len(inventory)
        ranked = sorted(
            inventory.values(),
            key=lambda item: _hash_parts(
                "selection",
                _SEED_ID,
                family,
                item["source_candidate_ref"],
                item["source_annotation_ref"],
                sources[family]["package_sha256"],
            ),
        )
        chosen = ranked[: min(items_per_family, len(ranked))]
        expected_selected[family] = {
            item["source_candidate_ref"]: item for item in chosen
        }
        selected_counts[family] = len(chosen)
        excluded_counts[family] = len(ranked) - len(chosen)

    expected_selection = {
        "algorithm": "sha256_rank_without_replacement@1",
        "seed_id": _SEED_ID,
        "items_per_family": items_per_family,
        "candidate_population_by_family": populations,
        "selected_by_family": selected_counts,
        "excluded_by_pilot_limit_by_family": excluded_counts,
    }
    _equal(batch["selection"], expected_selection, "selection")

    source_fingerprint = "|".join(
        f"{item['family']}:{item['package_sha256']}:{item['support_manifest_sha256']}"
        for item in expected_source_packages
    )
    expected_batch_id = "spatial-pilot-" + _hash_parts(
        "batch",
        loaded_protocol.protocol_sha256,
        batch["map_id"],
        str(items_per_family),
        source_fingerprint,
    )[:24]
    _equal(batch["batch_id"], expected_batch_id, "batch id")

    questions = {item["family"]: item for item in protocol["question_sets"]}
    seen_unit_ids: set[str] = set()
    seen_source_joins: set[tuple[str, ...]] = set()
    seen_display_orders: set[int] = set()
    expected_display: list[tuple[str, str]] = []
    actual_family_counts = {family: 0 for family in _FAMILIES}
    for unit in batch["review_units"]:
        audit = unit["audit_context"]
        family = audit["family"]
        if family not in expected_selected:
            _fail(f"unknown family {family}")
        candidate_ref = audit["source_candidate_ref"]
        expected_item = expected_selected[family].get(candidate_ref)
        if expected_item is None:
            _fail(f"unselected or unknown source candidate {family}:{candidate_ref}")
        source = sources[family]
        source_join = (
            batch["map_id"],
            family,
            expected_item["source_sample_ref"],
            candidate_ref,
            expected_item["source_annotation_ref"],
            source["package_sha256"],
        )
        if source_join in seen_source_joins:
            _fail("duplicate source join")
        seen_source_joins.add(source_join)
        expected_unit_id = "sru-" + _hash_parts(
            "review_unit",
            loaded_protocol.protocol_sha256,
            "|".join(source_join),
        )[:24]
        if unit["review_unit_id"] in seen_unit_ids:
            _fail("duplicate review_unit_id")
        seen_unit_ids.add(unit["review_unit_id"])
        _equal(unit["review_unit_id"], expected_unit_id, "review unit id")
        expected_audit = {
            "access_scope": "coordinator_only",
            "family": family,
            "map_id": batch["map_id"],
            "package_schema_id": source["package"]["schema_id"],
            "package_sha256": source["package_sha256"],
            "support_manifest_sha256": _sha256_file(
                roots[family] / "support_manifest.json"
            ),
            "source_sample_ref": expected_item["source_sample_ref"],
            "source_candidate_ref": candidate_ref,
            "source_annotation_ref": expected_item["source_annotation_ref"],
            "source_group_id": "srcgrp-"
            + _hash_parts(
                "source_group",
                batch["map_id"],
                family,
                expected_item["source_sample_ref"],
            )[:24],
        }
        _equal(audit, expected_audit, f"audit join {expected_unit_id}")
        question = questions[family]
        expected_view = {
            "prompt_id": question["prompt_id"],
            "question": question["question"],
            "claim_boundary": question["claim_boundary"],
            "options": copy.deepcopy(question["options"]),
            "objective_geometry": copy.deepcopy(expected_item["objective_geometry"]),
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
        _equal(unit["reviewer_view"], expected_view, f"reviewer view {expected_unit_id}")
        expected_view_sha = hashlib.sha256(
            _canonical_bytes(expected_view)
        ).hexdigest()
        _equal(
            unit["visible_evidence_sha256"],
            expected_view_sha,
            f"visible evidence hash {expected_unit_id}",
        )
        if (
            unit["state"] != "pending_review"
            or unit["review_responses"] != []
            or unit["adjudication"] is not None
        ):
            _fail("pilot contains human review or adjudication data")
        if unit["split_id"] != "pilot_review_only":
            _fail("cross-split leakage")
        order = unit["display_order"]
        if order in seen_display_orders:
            _fail("duplicate display order")
        seen_display_orders.add(order)
        expected_display.append(
            (
                _hash_parts("display", _SEED_ID, expected_unit_id),
                expected_unit_id,
            )
        )
        actual_family_counts[family] += 1
        _validate_reviewer_privacy(unit["reviewer_view"])

    ordered_ids = [item[1] for item in sorted(expected_display)]
    actual_ordered_ids = [
        unit["review_unit_id"]
        for unit in sorted(batch["review_units"], key=lambda item: item["display_order"])
    ]
    _equal(actual_ordered_ids, ordered_ids, "deterministic display order")
    _equal(
        seen_display_orders,
        set(range(1, len(batch["review_units"]) + 1)),
        "contiguous display order",
    )
    _equal(actual_family_counts, selected_counts, "family counts")

    total = len(batch["review_units"])
    expected_summary = {
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
    }
    _equal(batch["summary"], expected_summary, "summary")
    _equal(
        batch["leakage_control"],
        {
            "split_id": "pilot_review_only",
            "grouping_key": "map_id_family_source_sample_ref",
            "cross_split_duplicate_count": 0,
            "training_split": False,
        },
        "leakage controls",
    )
    return {
        "status": "valid",
        "batch_sha256": hashlib.sha256(payload).hexdigest(),
        "bytes": len(payload),
        "review_units": total,
        "family_counts": actual_family_counts,
        "candidate_population_by_family": populations,
        "excluded_by_pilot_limit_by_family": excluded_counts,
        "objective_geometry_available": total,
        "observed_context_unavailable": total,
        "responses": 0,
        "reviewer_identities": 0,
        "classifications": 0,
        "adjudications": 0,
        "recommendations": 0,
    }


def _load_sources(
    roots: dict[str, Path], maps_root: Path, map_id: str
) -> dict[str, dict[str, Any]]:
    loaders: dict[str, Callable[..., Any]] = {
        "exposure": load_exposure_package,
        "cover": load_cover_package,
        "exit_route": load_exit_route_package,
    }
    result: dict[str, dict[str, Any]] = {}
    for family in _FAMILIES:
        resolution = loaders[family](roots[family], maps_root, map_id)
        if (
            resolution.status != "available"
            or resolution.package is None
            or resolution.support is None
            or resolution.package_sha256 is None
        ):
            _fail(f"{family} package unavailable during independent validation")
        package = resolution.package
        support = resolution.support
        if (
            support.get("tactical_review_status") != "pending_review"
            or support.get("product_supported") is not False
            or support.get("training_use_authorized") is not False
            or package.get("tactical_review", {}).get("status") != "pending_review"
        ):
            _fail(f"{family} support state changed")
        for annotation in package["tactical_annotations"]:
            if (
                annotation.get("classification") is not None
                or annotation.get("author_id") is not None
                or annotation.get("reviewer_id") is not None
                or annotation.get("review_status") != "pending_review"
                or annotation.get("evidence") != []
            ):
                _fail(f"{family} source annotation contains human data")
        result[family] = {
            "package": package,
            "support": support,
            "package_sha256": resolution.package_sha256,
        }
    return result


def _source_summary(
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


def _inventory_exposure(package: dict[str, Any]) -> dict[str, dict[str, Any]]:
    sectors = {
        sector["id"]: (sample, sector)
        for sample in package["geometry_samples"]
        for sector in sample["geometry_sectors"]
    }
    inventory: dict[str, dict[str, Any]] = {}
    seen_annotations: set[str] = set()
    for annotation in package["tactical_annotations"]:
        ref = annotation["geometry_sector_ref"]
        if ref in inventory or annotation["id"] in seen_annotations or ref not in sectors:
            _fail("exposure orphan or duplicate")
        seen_annotations.add(annotation["id"])
        sample, sector = sectors[ref]
        rays = [
            copy.deepcopy(ray)
            for ray in sample["rays"]
            if (float(ray["azimuth_degrees"]) - float(sector["start_degrees"]))
            % 360.0
            < float(sector["width_degrees"])
        ]
        if not rays:
            _fail("exposure sector lacks rays")
        inventory[ref] = {
            "source_sample_ref": sample["id"],
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
                "supporting_rays": rays,
                "geometry_only": True,
            },
        }
    if len(inventory) != package["coverage"]["sector_count"]:
        _fail("exposure coverage mismatch")
    return inventory


def _inventory_cover(package: dict[str, Any]) -> dict[str, dict[str, Any]]:
    directions = {
        direction["id"]: (sample, direction)
        for sample in package["geometry_samples"]
        for direction in sample["directions"]
    }
    inventory: dict[str, dict[str, Any]] = {}
    seen_annotations: set[str] = set()
    for annotation in package["tactical_annotations"]:
        direction_ref = annotation["geometry_direction_ref"]
        posture = annotation["posture_id"]
        ref = f"{direction_ref}#{posture}"
        if (
            ref in inventory
            or annotation["id"] in seen_annotations
            or direction_ref not in directions
        ):
            _fail("cover orphan or duplicate")
        seen_annotations.add(annotation["id"])
        sample, direction = directions[direction_ref]
        profiles = {item["posture_id"]: item for item in direction["profiles"]}
        if posture not in profiles:
            _fail("cover posture missing")
        inventory[ref] = {
            "source_sample_ref": sample["id"],
            "source_candidate_ref": ref,
            "source_annotation_ref": annotation["id"],
            "objective_geometry": {
                "evidence_type": "player_volume_occlusion",
                "coordinates": "source2_xyz_world_units",
                "origin_world": copy.deepcopy(sample["origin_world"]),
                "selection_radius_units": sample["selection_radius_units"],
                "vertical_tolerance_units": sample["vertical_tolerance_units"],
                "incoming_azimuth_degrees": direction["azimuth_degrees"],
                "posture_id": posture,
                "profile": copy.deepcopy(profiles[posture]),
                "geometry_only": True,
            },
        }
    expected = package["coverage"]["direction_count"] * len(
        package["coverage"]["posture_ids"]
    )
    if len(inventory) != expected:
        _fail("cover coverage mismatch")
    return inventory


def _inventory_exit_route(package: dict[str, Any]) -> dict[str, dict[str, Any]]:
    candidates = {
        candidate["id"]: (sample, candidate)
        for sample in package["route_samples"]
        for candidate in sample["candidates"]
    }
    inventory: dict[str, dict[str, Any]] = {}
    seen_annotations: set[str] = set()
    for annotation in package["tactical_annotations"]:
        ref = annotation["candidate_ref"]
        if ref in inventory or annotation["id"] in seen_annotations or ref not in candidates:
            _fail("exit-route orphan or duplicate")
        seen_annotations.add(annotation["id"])
        sample, candidate = candidates[ref]
        inventory[ref] = {
            "source_sample_ref": sample["id"],
            "source_candidate_ref": ref,
            "source_annotation_ref": annotation["id"],
            "objective_geometry": {
                "evidence_type": "directed_navigation_candidate",
                "coordinates": "source2_xyz_world_units",
                "origin_world": copy.deepcopy(sample["origin_world"]),
                "destination_world": copy.deepcopy(candidate["destination_world"]),
                "selection_radius_units": sample["selection_radius_units"],
                "vertical_tolerance_units": sample["vertical_tolerance_units"],
                "direction": candidate["direction"],
                "objective_connectivity": copy.deepcopy(
                    candidate["objective_connectivity"]
                ),
                "geometry_only": True,
                "exact_user_facing_time_allowed": False,
                "arrival_prediction": False,
            },
        }
    if len(inventory) != package["coverage"]["candidate_count"]:
        _fail("exit-route coverage mismatch")
    return inventory


def _validate_reviewer_privacy(value: Any, path: str = "reviewer_view") -> None:
    if isinstance(value, dict):
        forbidden_keys = {
            "family",
            "map_id",
            "package_path",
            "package_sha256",
            "support_manifest_sha256",
            "source_sample_ref",
            "source_candidate_ref",
            "source_annotation_ref",
            "author_id",
            "reviewer_id",
            "steam_id",
            "faceit_id",
        }
        overlap = forbidden_keys.intersection(value)
        if overlap:
            _fail(f"{path}: reviewer-visible forbidden keys {sorted(overlap)}")
        for key, item in value.items():
            _validate_reviewer_privacy(item, f"{path}.{key}")
    elif isinstance(value, list):
        for index, item in enumerate(value):
            _validate_reviewer_privacy(item, f"{path}[{index}]")
    elif isinstance(value, str):
        if _ABSOLUTE_PATH.search(value) or "steamcommunity.com" in value.lower():
            _fail(f"{path}: retained path or public identity")


def _validate_temporary_batch_path(path: Path) -> Path:
    temporary_root = Path(tempfile.gettempdir()).resolve()
    resolved = path.resolve(strict=True)
    try:
        resolved.relative_to(temporary_root)
    except ValueError as exc:
        raise SpatialReviewPilotValidationError(
            "batch must stay under system temp"
        ) from exc
    if resolved.name != "spatial_review_pilot_batch.json" or not resolved.is_file():
        _fail("unexpected batch path")
    current = resolved
    while True:
        if _is_symlink_or_reparse(current):
            _fail("batch path contains link or reparse point")
        if current == temporary_root:
            break
        if current.parent == current:
            _fail("batch escaped system temp")
        current = current.parent
    return resolved


def _is_symlink_or_reparse(path: Path) -> bool:
    metadata = path.lstat()
    attributes = getattr(metadata, "st_file_attributes", 0)
    reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    return stat.S_ISLNK(metadata.st_mode) or bool(attributes & reparse_flag)


def _decode_json(payload: bytes, location: str) -> dict[str, Any]:
    try:
        value = json.loads(
            payload.decode("utf-8"), object_pairs_hook=_pairs_without_duplicates
        )
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise SpatialReviewPilotValidationError(
            f"{location}: invalid JSON"
        ) from exc
    if not isinstance(value, dict):
        _fail(f"{location}: expected object")
    return value


def _pairs_without_duplicates(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            _fail(f"duplicate JSON key {key}")
        result[key] = value
    return result


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
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _require_finite(value: Any, path: str) -> None:
    if value is None or isinstance(value, (str, bool, int)):
        return
    if isinstance(value, float):
        if not math.isfinite(value):
            _fail(f"{path}: non-finite number")
        return
    if isinstance(value, list):
        for index, item in enumerate(value):
            _require_finite(item, f"{path}[{index}]")
        return
    if isinstance(value, dict):
        for key, item in value.items():
            _require_finite(item, f"{path}.{key}")
        return
    _fail(f"{path}: unsupported value")


def _equal(actual: Any, expected: Any, location: str) -> None:
    if actual != expected:
        _fail(f"{location}: mismatch")


def _fail(message: str) -> None:
    raise SpatialReviewPilotValidationError(message)


def _argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Independently validate a package-only spatial review pilot"
    )
    parser.add_argument("batch", type=Path)
    parser.add_argument("--review-root", type=Path)
    parser.add_argument("--exposure-root", type=Path)
    parser.add_argument("--cover-root", type=Path)
    parser.add_argument("--exit-route-root", type=Path)
    parser.add_argument("--maps-root", type=Path)
    return parser


def main() -> int:
    args = _argument_parser().parse_args()
    result = validate_spatial_review_pilot(
        args.batch,
        review_root=args.review_root,
        exposure_root=args.exposure_root,
        cover_root=args.cover_root,
        exit_route_root=args.exit_route_root,
        maps_root=args.maps_root,
    )
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
