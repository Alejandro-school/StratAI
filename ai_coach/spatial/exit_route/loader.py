from __future__ import annotations

import hashlib
import json
import math
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator
from jsonschema.exceptions import SchemaError, ValidationError

CONTRACT_VERSION = "1.0.0"
PACKAGE_SCHEMA_ID = "stratai.exit_route_map_package@1"
SUPPORT_SCHEMA_ID = "stratai.exit_route_support_manifest@1"
SUPPORT_MANIFEST = "support_manifest.json"
_ROOT = Path(__file__).resolve().parents[2]
_PACKAGE_SCHEMA = _ROOT / "contracts" / "exit_route_map_package.schema.json"
_SUPPORT_SCHEMA = _ROOT / "contracts" / "exit_route_support_manifest.schema.json"


class ExitRoutePackageError(ValueError):
    """Raised when an objective route package violates its frozen contract."""


@dataclass(frozen=True)
class PackageResolution:
    status: str
    reason_codes: tuple[str, ...]
    package: dict[str, Any] | None = None
    package_sha256: str | None = None
    support: dict[str, Any] | None = None


def select_sample(
    package: dict[str, Any], point: dict[str, float]
) -> tuple[dict[str, Any] | None, str | None]:
    if any(not math.isfinite(float(point[axis])) for axis in ("x", "y", "z")):
        return None, "position_unavailable"
    candidates = []
    for sample in package["route_samples"]:
        origin = sample["origin_world"]
        horizontal = math.hypot(point["x"] - origin["x"], point["y"] - origin["y"])
        vertical = abs(point["z"] - origin["z"])
        if (
            horizontal <= sample["selection_radius_units"]
            and vertical <= sample["vertical_tolerance_units"]
        ):
            candidates.append(sample)
    if not candidates:
        return None, "outside_declared_sample_coverage"
    if len(candidates) > 1:
        return None, "ambiguous_sample_overlap"
    return candidates[0], None


def validate_package(
    package: dict[str, Any], *, maps_root: Path | None = None
) -> dict[str, int]:
    try:
        _validator(_PACKAGE_SCHEMA).validate(package)
    except (SchemaError, ValidationError) as exc:
        raise ExitRoutePackageError(f"package schema violation: {exc.message}") from exc
    _require_finite(package, "package")
    bounds = package["bounds"]
    for axis in ("x", "y", "z"):
        if not bounds["min"][axis] < bounds["max"][axis]:
            raise ExitRoutePackageError(f"bounds.{axis} is not ordered")

    expected_uses = {
        "nav_mesh": "objective_navigation",
        "physics_gltf": "audited_collision_only_not_route_truth",
        "physics_buffer": "audited_collision_only_not_route_truth",
        "sample_seed_catalog": "coordinate_seeds_only",
    }
    sources = {source["role"]: source for source in package["source_assets"]}
    if set(sources) != set(expected_uses):
        raise ExitRoutePackageError("source asset roles are incomplete or duplicated")
    for role, source in sources.items():
        if source["use"] != expected_uses[role]:
            raise ExitRoutePackageError("source asset semantic use is invalid")
        _safe_relative(source["reference"])
        if maps_root is not None:
            path = _safe_child(maps_root, source["reference"])
            if _sha256_file(path) != source["sha256"]:
                raise ExitRoutePackageError(f"source hash mismatch: {role}")

    related = {item["role"]: item for item in package["related_geometry_audit"]}
    if set(related) != {"exposure_package", "cover_package"}:
        raise ExitRoutePackageError("related geometry audit is incomplete")

    sample_ids: set[str] = set()
    aliases: set[str] = set()
    candidate_refs: set[str] = set()
    candidate_count = 0
    samples = package["route_samples"]
    for sample_index, sample in enumerate(samples, start=1):
        expected_sample = f"exit-sample-{sample_index:03d}"
        if sample["id"] != expected_sample or sample["id"] in sample_ids:
            raise ExitRoutePackageError("sample ids are not stable and ordered")
        if sample["alias"] in aliases:
            raise ExitRoutePackageError("sample aliases are duplicated")
        sample_ids.add(sample["id"])
        aliases.add(sample["alias"])
        _validate_point(sample["origin_world"], bounds)
        _validate_seed(sample["origin_provenance"], sources)
        for candidate_index, candidate in enumerate(sample["candidates"], start=1):
            expected_candidate = f"{sample['id']}/candidate-{candidate_index:03d}"
            if (
                candidate["id"] != expected_candidate
                or candidate["id"] in candidate_refs
            ):
                raise ExitRoutePackageError("candidate ids are not stable and ordered")
            if candidate["alias"] in aliases:
                raise ExitRoutePackageError("candidate aliases are duplicated")
            aliases.add(candidate["alias"])
            candidate_refs.add(candidate["id"])
            _validate_point(candidate["destination_world"], bounds)
            _validate_seed(candidate["destination_provenance"], sources)
            _validate_objective(candidate["objective_connectivity"])
            candidate_count += 1
    for left_index, left in enumerate(samples):
        for right in samples[left_index + 1 :]:
            if _volumes_overlap(left, right):
                raise ExitRoutePackageError("sample volumes overlap ambiguously")

    coverage = package["coverage"]
    if (
        coverage["full_map_coverage"] is not False
        or coverage["sample_count"] != len(samples)
        or coverage["candidate_count"] != candidate_count
        or coverage["minimum_observed_candidates"] > candidate_count
    ):
        raise ExitRoutePackageError("declared coverage differs from package")
    _validate_review(package["objective_validation"], approved=True)
    _validate_review(package["tactical_review"], approved=False)

    annotation_ids: set[str] = set()
    annotated: set[str] = set()
    for annotation in package["tactical_annotations"]:
        if (
            annotation["id"] in annotation_ids
            or annotation["candidate_ref"] in annotated
        ):
            raise ExitRoutePackageError("tactical annotation is duplicated")
        if annotation["candidate_ref"] not in candidate_refs:
            raise ExitRoutePackageError("tactical annotation reference is unknown")
        annotation_ids.add(annotation["id"])
        annotated.add(annotation["candidate_ref"])
        if (
            annotation["review_status"] != "pending_review"
            or any(
                annotation[field] is not None
                for field in ("classification", "author_id", "reviewer_id")
            )
            or annotation["evidence"]
        ):
            raise ExitRoutePackageError("pending annotation contains fabricated review")
    if annotated != candidate_refs:
        raise ExitRoutePackageError("tactical annotations do not cover candidates")
    return {
        "samples": len(samples),
        "candidates": candidate_count,
        "annotations": len(annotated),
    }


def load_package(
    registry_root: Path,
    maps_root: Path,
    map_id: str,
    *,
    contract_version: str = CONTRACT_VERSION,
    asset_version: str | None = None,
) -> PackageResolution:
    if contract_version != CONTRACT_VERSION:
        return _unavailable("contract_version_incompatible")
    try:
        manifest = _load_json(registry_root / SUPPORT_MANIFEST)
        _validator(_SUPPORT_SCHEMA).validate(manifest)
        _require_finite(manifest, "support_manifest")
    except OSError:
        return _unavailable("package_file_missing")
    except (json.JSONDecodeError, ExitRoutePackageError, SchemaError, ValidationError):
        return _unavailable("package_invalid")
    entries = manifest["packages"]
    if len({entry["map_id"] for entry in entries}) != len(entries):
        return _unavailable("package_invalid")
    if any(not _valid_support(entry) for entry in entries):
        return _unavailable("package_invalid")
    support = next((entry for entry in entries if entry["map_id"] == map_id), None)
    if support is None:
        return _unavailable("map_not_registered")
    if support["contract_version"] != contract_version:
        return _unavailable("contract_version_incompatible")
    if asset_version is not None and support["asset_version"] != asset_version:
        return _unavailable("asset_version_incompatible")
    try:
        path = _safe_child(registry_root, support["package_path"])
        payload = path.read_bytes()
    except OSError:
        return _unavailable("package_file_missing")
    except ExitRoutePackageError:
        return _unavailable("package_invalid")
    digest = hashlib.sha256(payload).hexdigest()
    if digest != support["package_sha256"]:
        return _unavailable("package_hash_mismatch")
    try:
        package = _decode_json(payload)
    except (json.JSONDecodeError, ExitRoutePackageError):
        return _unavailable("package_invalid")
    if package.get("map_id") != map_id or package.get("map_id") != support["map_id"]:
        return _unavailable("map_id_mismatch")
    if package.get("contract_version") != support["contract_version"]:
        return _unavailable("contract_version_incompatible")
    if package.get("asset_version") != support["asset_version"]:
        return _unavailable("asset_version_incompatible")
    try:
        validate_package(package, maps_root=maps_root)
    except OSError:
        return _unavailable("source_asset_mismatch")
    except ExitRoutePackageError as exc:
        if "source hash mismatch" in str(exc):
            return _unavailable("source_asset_mismatch")
        return _unavailable("package_invalid")
    if (
        package["tactical_review"]["status"] != support["tactical_review_status"]
        or support["product_supported"]
        or support["training_use_authorized"]
    ):
        return _unavailable("package_invalid")
    return PackageResolution("available", (), package, digest, support)


def _validate_seed(value: dict[str, Any], sources: dict[str, dict[str, Any]]) -> None:
    if value["source_reference"] != sources["sample_seed_catalog"]["reference"]:
        raise ExitRoutePackageError("seed provenance reference is not declared")


def _validate_point(
    point: dict[str, float], bounds: dict[str, dict[str, float]]
) -> None:
    if any(
        point[axis] < bounds["min"][axis] or point[axis] > bounds["max"][axis]
        for axis in ("x", "y", "z")
    ):
        raise ExitRoutePackageError("route endpoint is outside NAV bounds")


def _validate_objective(value: dict[str, Any]) -> None:
    if value["reachable"] is None:
        if (
            value["status"] != "unavailable"
            or value["path_distance_world_units"] is not None
            or not value["reason_codes"]
        ):
            raise ExitRoutePackageError("unavailable objective is inconsistent")
    elif value["reachable"]:
        if (
            value["status"] != "derived"
            or value["path_distance_world_units"] is None
            or value["origin_association_world_units"] is None
            or value["destination_association_world_units"] is None
        ):
            raise ExitRoutePackageError("reachable objective is incomplete")
    elif value["path_distance_world_units"] is not None or not value["reason_codes"]:
        raise ExitRoutePackageError("disconnected objective is inconsistent")


def _validate_review(value: dict[str, Any], *, approved: bool) -> None:
    if approved:
        if (
            value["status"] != "approved"
            or not value["author_id"]
            or not value["reviewer_id"]
            or not value["evidence"]
        ):
            raise ExitRoutePackageError("objective review is incomplete")
    elif (
        value["status"] != "pending_review"
        or value["author_id"] is not None
        or value["reviewer_id"] is not None
        or value["evidence"]
    ):
        raise ExitRoutePackageError("pending tactical review contains attribution")


def _valid_support(entry: dict[str, Any]) -> bool:
    return (
        entry["package_path"]
        == f"packages/{entry['map_id']}/{entry['asset_version']}/package.json"
        and not entry["product_supported"]
        and not entry["training_use_authorized"]
    )


def _volumes_overlap(left: dict[str, Any], right: dict[str, Any]) -> bool:
    a, b = left["origin_world"], right["origin_world"]
    return (
        math.hypot(a["x"] - b["x"], a["y"] - b["y"])
        < left["selection_radius_units"] + right["selection_radius_units"]
        and abs(a["z"] - b["z"])
        <= left["vertical_tolerance_units"] + right["vertical_tolerance_units"]
    )


def _validator(path: Path) -> Draft202012Validator:
    schema = _load_json(path)
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def _safe_relative(reference: str) -> None:
    if (
        not reference
        or "\\" in reference
        or Path(reference).is_absolute()
        or ".." in Path(reference).parts
    ):
        raise ExitRoutePackageError("asset reference is not safe and relative")


def _safe_child(root: Path, reference: str) -> Path:
    _safe_relative(reference)
    resolved_root = root.resolve(strict=True)
    candidate = (resolved_root / reference).resolve(strict=True)
    if candidate == resolved_root or resolved_root not in candidate.parents:
        raise ExitRoutePackageError("asset reference escapes its root")
    return candidate


def _require_finite(value: Any, path: str) -> None:
    if isinstance(value, bool) or value is None:
        return
    if isinstance(value, (int, float)):
        if not math.isfinite(float(value)):
            raise ExitRoutePackageError(f"{path} contains a non-finite number")
    elif isinstance(value, dict):
        for key, child in value.items():
            _require_finite(child, f"{path}.{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            _require_finite(child, f"{path}[{index}]")


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
        raise ExitRoutePackageError("JSON root must be an object")
    return value


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _unavailable(reason: str) -> PackageResolution:
    return PackageResolution("unavailable", (reason,))
