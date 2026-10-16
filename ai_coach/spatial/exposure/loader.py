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
PACKAGE_SCHEMA_ID = "stratai.exposure_angle_map_package@1"
SUPPORT_SCHEMA_ID = "stratai.exposure_angle_support_manifest@1"
SUPPORT_MANIFEST = "support_manifest.json"

_ROOT = Path(__file__).resolve().parents[2]
_PACKAGE_SCHEMA = _ROOT / "contracts" / "exposure_angle_map_package.schema.json"
_SUPPORT_SCHEMA = _ROOT / "contracts" / "exposure_angle_support_manifest.schema.json"


class ExposurePackageError(ValueError):
    """Raised when a map package violates the common exposure contract."""


@dataclass(frozen=True)
class PackageResolution:
    status: str
    reason_codes: tuple[str, ...]
    package: dict[str, Any] | None = None
    package_sha256: str | None = None
    support: dict[str, Any] | None = None


def normalize_degrees(value: float) -> float:
    result = math.fmod(value, 360.0)
    if result < 0:
        result += 360.0
    return 0.0 if result == 360.0 or abs(result) < 1e-12 else result


def sector_contains(sector: dict[str, Any], angle_degrees: float) -> bool:
    width = float(sector["width_degrees"])
    if width >= 360:
        return True
    return normalize_degrees(angle_degrees - float(sector["start_degrees"])) < width


def relative_sectors(
    sectors: list[dict[str, Any]], view_yaw_degrees: float
) -> list[dict[str, Any]]:
    result = [
        {
            "id": sector["id"],
            "start_degrees": normalize_degrees(
                float(sector["start_degrees"]) - view_yaw_degrees
            ),
            "width_degrees": float(sector["width_degrees"]),
        }
        for sector in sectors
    ]
    return sorted(result, key=lambda sector: (sector["start_degrees"], sector["id"]))


def build_sectors(
    sample_id: str, rays: list[dict[str, Any]], step_degrees: float
) -> list[dict[str, Any]]:
    expected = round(360.0 / step_degrees)
    if (
        expected < 1
        or not math.isclose(expected * step_degrees, 360.0, abs_tol=1e-9)
        or len(rays) != expected
    ):
        raise ExposurePackageError("ray grid does not divide the full circle")
    clear: list[bool] = []
    for index, ray in enumerate(rays):
        if not math.isclose(
            float(ray["azimuth_degrees"]), index * step_degrees, abs_tol=1e-9
        ):
            raise ExposurePackageError("ray azimuth grid is incomplete or unordered")
        clear.append(ray["clearance_met"] is True)
    if not any(clear):
        return []
    if all(clear):
        return [
            {
                "id": f"{sample_id}/sector-001",
                "start_degrees": 0.0,
                "width_degrees": 360.0,
            }
        ]
    raw: list[tuple[float, float]] = []
    for index, is_clear in enumerate(clear):
        if not is_clear or clear[(index - 1) % len(clear)]:
            continue
        count = 0
        while count < len(clear) and clear[(index + count) % len(clear)]:
            count += 1
        raw.append(
            (
                normalize_degrees(index * step_degrees - step_degrees / 2.0),
                count * step_degrees,
            )
        )
    raw.sort()
    return [
        {
            "id": f"{sample_id}/sector-{index:03d}",
            "start_degrees": start,
            "width_degrees": width,
        }
        for index, (start, width) in enumerate(raw, start=1)
    ]


def validate_package(
    package: dict[str, Any], *, maps_root: Path | None = None
) -> dict[str, int]:
    try:
        _schema_validator(_PACKAGE_SCHEMA).validate(package)
    except (SchemaError, ValidationError) as exc:
        raise ExposurePackageError(f"package schema violation: {exc.message}") from exc
    _require_finite(package, "package")
    bounds = package["bounds"]
    for axis in ("x", "y", "z"):
        if not bounds["min"][axis] < bounds["max"][axis]:
            raise ExposurePackageError(f"bounds.{axis} is not ordered")
    resolution = package["resolution"]
    if resolution["minimum_clearance_distance"] > resolution["maximum_ray_distance"]:
        raise ExposurePackageError("minimum clearance exceeds maximum ray distance")
    roles: dict[str, str] = {}
    for source in package["geometry_provenance"]:
        if source["role"] in roles:
            raise ExposurePackageError("geometry provenance roles are not unique")
        roles[source["role"]] = source["reference"]
        _safe_relative(source["reference"])
        if maps_root is not None:
            source_path = _safe_child(maps_root, source["reference"])
            if _sha256_file(source_path) != source["sha256"]:
                raise ExposurePackageError(
                    f"geometry source hash mismatch: {source['role']}"
                )
    if not {"physics_gltf", "physics_buffer"}.issubset(roles):
        raise ExposurePackageError("physics geometry provenance is incomplete")

    sample_ids: set[str] = set()
    sample_aliases: set[str] = set()
    sector_refs: set[str] = set()
    sector_count = 0
    samples = package["geometry_samples"]
    for sample in samples:
        if sample["id"] in sample_ids or sample["alias"] in sample_aliases:
            raise ExposurePackageError("sample ids and aliases must be unique")
        sample_ids.add(sample["id"])
        sample_aliases.add(sample["alias"])
        provenance = sample["origin_provenance"]
        if roles.get(provenance["source_role"]) != provenance["source_reference"]:
            raise ExposurePackageError("sample origin provenance is not declared")
        for axis in ("x", "y", "z"):
            value = sample["origin_world"][axis]
            if value < bounds["min"][axis] or value > bounds["max"][axis]:
                raise ExposurePackageError("sample origin is outside map bounds")
        rays = sample["rays"]
        for ray in rays:
            distance = ray["hit_distance_units"]
            if distance is not None and distance > resolution["maximum_ray_distance"]:
                raise ExposurePackageError("ray hit exceeds maximum distance")
            clearance = (
                distance is None or distance >= resolution["minimum_clearance_distance"]
            )
            if ray["clearance_met"] is not clearance:
                raise ExposurePackageError("ray clearance does not match distance")
        expected = build_sectors(sample["id"], rays, resolution["azimuth_step_degrees"])
        if not _strict_equal(expected, sample["geometry_sectors"]):
            raise ExposurePackageError("stored geometry sectors do not recompute")
        for sector in sample["geometry_sectors"]:
            if sector["id"] in sector_refs:
                raise ExposurePackageError("geometry sector ids are not unique")
            sector_refs.add(sector["id"])
            sector_count += 1
    for left_index, left in enumerate(samples):
        for right in samples[left_index + 1 :]:
            if _sample_volumes_overlap(left, right):
                raise ExposurePackageError("sample volumes overlap ambiguously")
    coverage = package["coverage"]
    if (
        coverage["sample_count"] != len(samples)
        or coverage["sector_count"] != sector_count
        or coverage["full_map_coverage"] is not False
    ):
        raise ExposurePackageError("declared coverage differs from real package")
    _validate_review(package["geometry_validation"], approved_required=True)
    _validate_review(package["tactical_review"], approved_required=False)
    annotation_ids: set[str] = set()
    annotated_sectors: set[str] = set()
    for annotation in package["tactical_annotations"]:
        if annotation["id"] in annotation_ids:
            raise ExposurePackageError("tactical annotation ids are not unique")
        annotation_ids.add(annotation["id"])
        if annotation["geometry_sector_ref"] not in sector_refs:
            raise ExposurePackageError("tactical annotation reference is unknown")
        if annotation["geometry_sector_ref"] in annotated_sectors:
            raise ExposurePackageError("tactical sector is annotated more than once")
        annotated_sectors.add(annotation["geometry_sector_ref"])
        if annotation["review_status"] == "pending_review":
            if (
                any(
                    annotation[field] is not None
                    for field in ("classification", "author_id", "reviewer_id")
                )
                or annotation["evidence"]
            ):
                raise ExposurePackageError(
                    "pending tactical annotation contains fabricated review"
                )
        elif not all(
            (
                annotation["classification"],
                annotation["author_id"],
                annotation["reviewer_id"],
                annotation["evidence"],
            )
        ):
            raise ExposurePackageError(
                "approved tactical annotation lacks attributable evidence"
            )
    if annotated_sectors != sector_refs:
        raise ExposurePackageError(
            "tactical annotations do not cover the declared geometry sectors"
        )
    if package["tactical_review"]["status"] == "approved" and any(
        annotation["review_status"] != "approved"
        for annotation in package["tactical_annotations"]
    ):
        raise ExposurePackageError(
            "approved tactical review contains pending annotations"
        )
    return {
        "samples": len(samples),
        "sectors": sector_count,
        "annotations": len(package["tactical_annotations"]),
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
        _schema_validator(_SUPPORT_SCHEMA).validate(manifest)
        _require_finite(manifest, "support_manifest")
    except OSError:
        return _unavailable("package_file_missing")
    except (json.JSONDecodeError, ExposurePackageError, SchemaError, ValidationError):
        return _unavailable("package_invalid")
    entries = manifest["packages"]
    if len({entry["map_id"] for entry in entries}) != len(entries):
        return _unavailable("package_invalid")
    if any(not _valid_support_entry(entry) for entry in entries):
        return _unavailable("package_invalid")
    support = next((entry for entry in entries if entry["map_id"] == map_id), None)
    if support is None:
        return _unavailable("map_not_registered")
    if support["contract_version"] != contract_version:
        return _unavailable("contract_version_incompatible")
    if asset_version is not None and support["asset_version"] != asset_version:
        return _unavailable("asset_version_incompatible")
    try:
        package_path = _safe_child(registry_root, support["package_path"])
        payload = package_path.read_bytes()
    except OSError:
        return _unavailable("package_file_missing")
    except ExposurePackageError:
        return _unavailable("package_invalid")
    digest = hashlib.sha256(payload).hexdigest()
    if digest != support["package_sha256"]:
        return _unavailable("package_hash_mismatch")
    try:
        package = _decode_json(payload)
    except json.JSONDecodeError:
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
        return _unavailable("geometry_source_mismatch")
    except ExposurePackageError as exc:
        if "source hash mismatch" in str(exc):
            return _unavailable("geometry_source_mismatch")
        return _unavailable("package_invalid")
    if package["tactical_review"]["status"] != support["tactical_review_status"] or (
        support["product_supported"]
        and package["tactical_review"]["status"] != "approved"
    ):
        return _unavailable("package_invalid")
    return PackageResolution(
        status="available",
        reason_codes=(),
        package=package,
        package_sha256=digest,
        support=support,
    )


def _validate_review(review: dict[str, Any], *, approved_required: bool) -> None:
    if approved_required and review["status"] != "approved":
        raise ExposurePackageError("required geometry validation is not approved")
    if review["status"] == "pending_review":
        if review["author_id"] is not None or review["reviewer_id"] is not None:
            raise ExposurePackageError("pending review is signed")
        if review["evidence"]:
            raise ExposurePackageError("pending review contains fabricated evidence")
    elif not review["author_id"] or not review["reviewer_id"] or not review["evidence"]:
        raise ExposurePackageError("approved review lacks provenance")


def _valid_support_entry(entry: dict[str, Any]) -> bool:
    expected_path = f"packages/{entry['map_id']}/{entry['asset_version']}/package.json"
    return (
        entry["package_path"] == expected_path
        and not entry["training_use_authorized"]
        and not (
            entry["product_supported"] and entry["tactical_review_status"] != "approved"
        )
    )


def _sample_volumes_overlap(left: dict[str, Any], right: dict[str, Any]) -> bool:
    left_origin, right_origin = left["origin_world"], right["origin_world"]
    horizontal = math.hypot(
        left_origin["x"] - right_origin["x"],
        left_origin["y"] - right_origin["y"],
    )
    vertical = abs(left_origin["z"] - right_origin["z"])
    return (
        horizontal < left["selection_radius_units"] + right["selection_radius_units"]
        and vertical
        <= left["vertical_tolerance_units"] + right["vertical_tolerance_units"]
    )


def _strict_equal(left: Any, right: Any) -> bool:
    if type(left) is not type(right):
        if isinstance(left, (int, float)) and isinstance(right, (int, float)):
            return math.isclose(float(left), float(right), abs_tol=1e-9)
        return False
    if isinstance(left, dict):
        return left.keys() == right.keys() and all(
            _strict_equal(left[key], right[key]) for key in left
        )
    if isinstance(left, list):
        return len(left) == len(right) and all(
            _strict_equal(a, b) for a, b in zip(left, right, strict=True)
        )
    if isinstance(left, float):
        return math.isclose(left, right, abs_tol=1e-9)
    return left == right


def _require_finite(value: Any, path: str) -> None:
    if isinstance(value, bool) or value is None:
        return
    if isinstance(value, (int, float)):
        if not math.isfinite(float(value)):
            raise ExposurePackageError(f"{path} contains a non-finite number")
        return
    if isinstance(value, dict):
        for key, child in value.items():
            _require_finite(child, f"{path}.{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            _require_finite(child, f"{path}[{index}]")


def _schema_validator(path: Path) -> Draft202012Validator:
    schema = _load_json(path)
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


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
        raise ExposurePackageError("JSON root must be an object")
    return value


def _safe_relative(reference: str) -> None:
    if (
        not reference
        or "\\" in reference
        or Path(reference).is_absolute()
        or ".." in Path(reference).parts
    ):
        raise ExposurePackageError("asset reference is not a safe relative path")


def _safe_child(root: Path, reference: str) -> Path:
    _safe_relative(reference)
    resolved_root = root.resolve(strict=True)
    candidate = (resolved_root / reference).resolve(strict=True)
    if candidate == resolved_root or resolved_root not in candidate.parents:
        raise ExposurePackageError("asset reference escapes its root")
    return candidate


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _unavailable(reason: str) -> PackageResolution:
    return PackageResolution(status="unavailable", reason_codes=(reason,))
