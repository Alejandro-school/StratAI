from __future__ import annotations

import argparse
import hashlib
import json
import math
import re
import tempfile
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker

from ai_coach.spatial.cover.loader import load_package as load_cover_package
from ai_coach.spatial.exit_route.loader import load_package as load_exit_route_package
from ai_coach.spatial.exposure.loader import load_package as load_exposure_package

_ROOT = Path(__file__).resolve().parents[2]
_SCHEMA = Path(__file__).with_name("spatial_context_review_batch.schema.json")
_PROTOCOL_ROOT = _ROOT / "ai_coach" / "spatial" / "review" / "protocols" / "2.0.0"
_RADAR = _ROOT / "frontend" / "public" / "maps" / "de_mirage_radar_psd.webp"
_MAPS_ROOT = _ROOT / "backend" / "data" / "maps"
_SPATIAL_ROOT = _ROOT / "ai_coach" / "spatial"
_FAMILIES = ("exposure", "cover", "exit_route")
_SEED_ID = "mirage-contextual-review-pilot-repair-v2"
_ABSOLUTE_PATH = re.compile(r"^(?:[A-Za-z]:[\\/]|/home/|/Users/|/tmp/|\\\\)")


class SpatialContextReviewValidationError(ValueError):
    """Raised when a contextual spatial-review batch cannot be certified."""


def validate_spatial_context_review_pilot(
    batch_path: Path,
    *,
    radar_path: Path | None = None,
    maps_root: Path | None = None,
    exposure_root: Path | None = None,
    cover_root: Path | None = None,
    exit_route_root: Path | None = None,
) -> dict[str, Any]:
    batch_path = _temporary_file(batch_path)
    payload = batch_path.read_bytes()
    batch = _object(payload, "batch")
    schema = _object(_SCHEMA.read_bytes(), "schema")
    Draft202012Validator.check_schema(schema)
    Draft202012Validator(schema, format_checker=FormatChecker()).validate(batch)
    _finite(batch, "batch")

    protocol_path = _PROTOCOL_ROOT / "protocol.json"
    handbook_path = _PROTOCOL_ROOT / "HANDBOOK.md"
    protocol = _object(protocol_path.read_bytes(), "protocol")
    protocol_ref = {
        "schema_id": protocol["schema_id"],
        "protocol_version": protocol["protocol_version"],
        "handbook_version": protocol["handbook"]["version"],
        "published_at": protocol["published_at"],
        "protocol_sha256": _sha256_file(protocol_path),
        "handbook_sha256": _sha256_file(handbook_path),
    }
    _equal(protocol["handbook"]["sha256"], protocol_ref["handbook_sha256"], "handbook hash")
    _equal(batch["protocol"], protocol_ref, "protocol provenance")
    _equal(batch["authorization"], protocol["authorization"], "authorization")

    resolved_radar = (radar_path or _RADAR).resolve(strict=True)
    _equal(batch["presentation"]["radar_asset"]["sha256"], _sha256_file(resolved_radar), "radar hash")
    if resolved_radar.stat().st_size < 100_000:
        _fail("radar asset is unexpectedly small")

    resolved_maps = (maps_root or _MAPS_ROOT).resolve(strict=True)
    roots = {
        "exposure": (exposure_root or _SPATIAL_ROOT / "exposure").resolve(strict=True),
        "cover": (cover_root or _SPATIAL_ROOT / "cover").resolve(strict=True),
        "exit_route": (exit_route_root or _SPATIAL_ROOT / "exit_route").resolve(strict=True),
    }
    loaders = {
        "exposure": load_exposure_package,
        "cover": load_cover_package,
        "exit_route": load_exit_route_package,
    }
    sources: dict[str, dict[str, Any]] = {}
    for family in _FAMILIES:
        resolution = loaders[family](roots[family], resolved_maps, "de_mirage")
        if resolution.status != "available" or resolution.package is None or resolution.package_sha256 is None:
            _fail(f"{family} package unavailable")
        sources[family] = {
            "package": resolution.package,
            "package_sha256": resolution.package_sha256,
        }
    expected_sources = [
        {
            "family": family,
            "schema_id": sources[family]["package"]["schema_id"],
            "asset_version": sources[family]["package"]["asset_version"],
            "package_sha256": sources[family]["package_sha256"],
            "support_manifest_sha256": _sha256_file(roots[family] / "support_manifest.json"),
            "geometry_revision": sources[family]["package"]["geometry_revision"],
        }
        for family in _FAMILIES
    ]
    _equal(batch["source_packages"], expected_sources, "source packages")

    questions = {item["family"]: item for item in protocol["question_sets"]}
    seen_ids: set[str] = set()
    seen_joins: set[str] = set()
    seen_orders: set[int] = set()
    expected_order: list[tuple[str, str]] = []
    counts = {family: 0 for family in _FAMILIES}
    route_points = 0
    for unit in batch["review_units"]:
        audit = unit["audit_context"]
        view = unit["reviewer_view"]
        family = audit["family"]
        if family != view["case_kind"]:
            _fail("audit and reviewer family mismatch")
        if audit["source_join"] in seen_joins:
            _fail("duplicate source join")
        seen_joins.add(audit["source_join"])
        expected_id = "scu-" + _hash_parts(
            "review_unit", protocol_ref["protocol_sha256"], audit["source_join"]
        )[:24]
        _equal(unit["review_unit_id"], expected_id, "review unit id")
        if expected_id in seen_ids:
            _fail("duplicate review unit id")
        seen_ids.add(expected_id)
        expected_order.append((_hash_parts("display", _SEED_ID, expected_id), expected_id))
        if unit["display_order"] in seen_orders:
            _fail("duplicate display order")
        seen_orders.add(unit["display_order"])
        _equal(audit["package_sha256"], sources[family]["package_sha256"], "audit package hash")
        question = questions[family]
        for key in ("prompt_id", "question", "claim_boundary", "options"):
            _equal(view[key], question[key], f"{family} {key}")
        context = view["observed_context"]
        if context["availability_tick"] > context["t0_tick"]:
            _fail("reviewer context uses future data")
        if context["position_world"] != view["objective_geometry"]["origin_world"]:
            _fail("observed and geometry origins differ")
        _privacy(view, "reviewer_view")
        _validate_geometry(family, view["objective_geometry"])
        if family == "exit_route":
            route_points += len(view["objective_geometry"]["path_points_world"])
        visible = {
            "reviewer_view": view,
            "radar_sha256": batch["presentation"]["radar_asset"]["sha256"],
            "render_contract": batch["presentation"]["render_contract"],
        }
        _equal(unit["visible_evidence_sha256"], hashlib.sha256(_canonical(visible)).hexdigest(), "visible evidence hash")
        if unit["state"] != "pending_review" or unit["review_responses"] or unit["adjudication"] is not None:
            _fail("batch already contains review data")
        counts[family] += 1

    ordered = [unit["review_unit_id"] for unit in sorted(batch["review_units"], key=lambda value: value["display_order"])]
    _equal(ordered, [value[1] for value in sorted(expected_order)], "display order")
    _equal(seen_orders, set(range(1, 13)), "contiguous display order")
    _equal(counts, {family: 4 for family in _FAMILIES}, "family counts")
    expected_batch_id = "spatial-context-pilot-" + _hash_parts(
        "batch",
        protocol_ref["protocol_sha256"],
        batch["source_demo"]["sha256"],
        batch["presentation"]["radar_asset"]["sha256"],
        *sorted(seen_joins),
    )[:24]
    _equal(batch["batch_id"], expected_batch_id, "batch id")
    return {
        "status": "valid",
        "batch_sha256": hashlib.sha256(payload).hexdigest(),
        "bytes": len(payload),
        "review_units": 12,
        "family_counts": counts,
        "observed_context_available": 12,
        "route_polylines_available": 4,
        "route_polyline_points": route_points,
        "responses": 0,
        "training_allowed": False,
    }


def _validate_geometry(family: str, geometry: dict[str, Any]) -> None:
    if family == "exposure":
        _equal(geometry.get("evidence_type"), "situated_angular_clearance_sector", "exposure type")
        sector = geometry.get("sector", {})
        rays = geometry.get("supporting_rays")
        if not rays or not 0 < float(sector.get("width_degrees", 0)) <= 360:
            _fail("exposure sector has no usable rays")
        for ray in rays:
            if (float(ray["azimuth_degrees"]) - float(sector["start_degrees"])) % 360 >= float(sector["width_degrees"]):
                _fail("exposure ray falls outside sector")
    elif family == "cover":
        _equal(geometry.get("evidence_type"), "situated_player_volume_occlusion", "cover type")
        profile = geometry.get("profile", {})
        probes = profile.get("probes")
        if profile.get("probe_count") != 9 or not isinstance(probes, list) or len(probes) != 9:
            _fail("cover evidence does not contain nine probes")
        observed = sum(bool(probe["occluded_within_protection_distance"]) for probe in probes)
        _equal(profile.get("occluded_probe_count"), observed, "cover probe count")
        _equal(profile.get("complete_volume_occlusion"), observed == 9, "cover completeness")
    elif family == "exit_route":
        _equal(geometry.get("evidence_type"), "situated_directed_navigation_path", "route type")
        _equal(geometry.get("trace_contract"), "nav_area_centres_polyline@1", "route trace contract")
        points = geometry.get("path_points_world")
        if not isinstance(points, list) or len(points) < 3:
            _fail("route polyline is not reviewable")
        _equal(points[0], geometry["origin_world"], "route origin")
        _equal(points[-1], geometry["destination_world"], "route destination")
        connectivity = geometry["objective_connectivity"]
        if connectivity["reachable"] is not True:
            _fail("unreachable route selected")
        direct = float(connectivity["direct_distance_world_units"])
        path = float(connectivity["path_distance_world_units"])
        if path + 1e-6 < direct:
            _fail("path distance shorter than direct distance")
    else:
        _fail(f"unknown family {family}")


def _privacy(value: Any, path: str) -> None:
    forbidden = {"steam_id", "steamid", "name", "winner", "outcome", "future", "enemy_position", "audit_context"}
    if isinstance(value, dict):
        overlap = forbidden.intersection(key.lower() for key in value)
        if overlap:
            _fail(f"visible forbidden keys at {path}: {sorted(overlap)}")
        for key, item in value.items():
            _privacy(item, f"{path}.{key}")
    elif isinstance(value, list):
        for index, item in enumerate(value):
            _privacy(item, f"{path}[{index}]")
    elif isinstance(value, str) and (_ABSOLUTE_PATH.match(value) or "steamcommunity" in value.lower()):
        _fail(f"visible absolute path or identity at {path}")


def _temporary_file(path: Path) -> Path:
    resolved = path.resolve(strict=True)
    if not resolved.is_file():
        _fail("batch path is not a file")
    try:
        resolved.relative_to(Path(tempfile.gettempdir()).resolve(strict=True))
    except ValueError as exc:
        raise SpatialContextReviewValidationError("batch must remain under system temp") from exc
    return resolved


def _object(payload: bytes, label: str) -> dict[str, Any]:
    try:
        value = json.loads(payload.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise SpatialContextReviewValidationError(f"invalid {label} json") from exc
    if not isinstance(value, dict):
        _fail(f"{label} must be an object")
    return value


def _finite(value: Any, path: str) -> None:
    if isinstance(value, float) and not math.isfinite(value):
        _fail(f"non-finite number at {path}")
    if isinstance(value, dict):
        for key, item in value.items():
            _finite(item, f"{path}.{key}")
    elif isinstance(value, list):
        for index, item in enumerate(value):
            _finite(item, f"{path}[{index}]")


def _canonical(value: Any) -> bytes:
    return (json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"), allow_nan=False) + "\n").encode("utf-8")


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.resolve(strict=True).open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _hash_parts(*parts: str) -> str:
    return hashlib.sha256("\x1f".join(parts).encode("utf-8")).hexdigest()


def _equal(actual: Any, expected: Any, label: str) -> None:
    if actual != expected:
        _fail(f"{label} mismatch")


def _fail(message: str) -> None:
    raise SpatialContextReviewValidationError(message)


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate a contextual spatial-review pilot")
    parser.add_argument("batch", type=Path)
    args = parser.parse_args()
    print(json.dumps(validate_spatial_context_review_pilot(args.batch), sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
