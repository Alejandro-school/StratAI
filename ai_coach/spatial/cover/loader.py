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
PACKAGE_SCHEMA_ID = "stratai.cover_map_package@1"
SUPPORT_SCHEMA_ID = "stratai.cover_support_manifest@1"
SUPPORT_MANIFEST = "support_manifest.json"

_ROOT = Path(__file__).resolve().parents[2]
_PACKAGE_SCHEMA = _ROOT / "contracts" / "cover_map_package.schema.json"
_SUPPORT_SCHEMA = _ROOT / "contracts" / "cover_support_manifest.schema.json"


class CoverPackageError(ValueError):
    """Raised when a map package violates the objective cover contract."""


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


def select_sample(
    package: dict[str, Any], point: dict[str, float]
) -> tuple[dict[str, Any] | None, str | None]:
    if any(not math.isfinite(float(point[axis])) for axis in ("x", "y", "z")):
        return None, "position_unavailable"
    candidates = []
    for sample in package["geometry_samples"]:
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


def profile_for_posture(
    direction: dict[str, Any], posture_id: str
) -> dict[str, Any] | None:
    return next(
        (
            profile
            for profile in direction["profiles"]
            if profile["posture_id"] == posture_id
        ),
        None,
    )


def validate_package(
    package: dict[str, Any], *, maps_root: Path | None = None
) -> dict[str, int]:
    try:
        _schema_validator(_PACKAGE_SCHEMA).validate(package)
    except (SchemaError, ValidationError) as exc:
        raise CoverPackageError(f"package schema violation: {exc.message}") from exc
    _require_finite(package, "package")
    bounds = package["bounds"]
    for axis in ("x", "y", "z"):
        if not bounds["min"][axis] < bounds["max"][axis]:
            raise CoverPackageError(f"bounds.{axis} is not ordered")
    resolution = package["resolution"]
    if (
        resolution["protection_distance_units"]
        > resolution["maximum_ray_distance_units"]
    ):
        raise CoverPackageError("protection distance exceeds maximum ray distance")
    slots = round(360.0 / resolution["azimuth_step_degrees"])
    if not math.isclose(
        slots * resolution["azimuth_step_degrees"], 360.0, abs_tol=1e-9
    ):
        raise CoverPackageError("direction step must divide 360")

    volume = package["player_volume"]
    offsets = volume["lateral_offsets_units"]
    if offsets != sorted(set(offsets)) or 0 not in offsets:
        raise CoverPackageError("lateral offsets must be ordered, unique and centered")
    if any(abs(offset) > volume["radius_units"] for offset in offsets):
        raise CoverPackageError("lateral probe offset exceeds player radius")
    posture_ids: list[str] = []
    postures: dict[str, dict[str, Any]] = {}
    for posture in volume["postures"]:
        if posture["id"] in postures:
            raise CoverPackageError("posture ids are not unique")
        heights = posture["probe_heights_units"]
        if heights != sorted(set(heights)) or any(
            height > posture["volume_height_units"] for height in heights
        ):
            raise CoverPackageError("posture probe heights are invalid")
        posture_ids.append(posture["id"])
        postures[posture["id"]] = posture
    if posture_ids != sorted(posture_ids):
        raise CoverPackageError("postures are not deterministically ordered")

    roles: dict[str, str] = {}
    for source in package["geometry_provenance"]:
        if source["role"] in roles:
            raise CoverPackageError("geometry provenance roles are not unique")
        roles[source["role"]] = source["reference"]
        _safe_relative(source["reference"])
        if maps_root is not None:
            source_path = _safe_child(maps_root, source["reference"])
            if _sha256_file(source_path) != source["sha256"]:
                raise CoverPackageError(
                    f"geometry source hash mismatch: {source['role']}"
                )
    if not {"physics_gltf", "physics_buffer"}.issubset(roles):
        raise CoverPackageError("physics geometry provenance is incomplete")

    samples = package["geometry_samples"]
    sample_ids: set[str] = set()
    aliases: set[str] = set()
    direction_profiles: set[tuple[str, str]] = set()
    direction_count = 0
    for sample_index, sample in enumerate(samples, start=1):
        expected_sample_id = f"cover-sample-{sample_index:03d}"
        if sample["id"] != expected_sample_id:
            raise CoverPackageError("sample ids are not stable and ordered")
        if sample["id"] in sample_ids or sample["alias"] in aliases:
            raise CoverPackageError("sample ids and aliases must be unique")
        sample_ids.add(sample["id"])
        aliases.add(sample["alias"])
        provenance = sample["origin_provenance"]
        if roles.get(provenance["source_role"]) != provenance["source_reference"]:
            raise CoverPackageError("sample origin provenance is not declared")
        for axis in ("x", "y", "z"):
            value = sample["origin_world"][axis]
            if value < bounds["min"][axis] or value > bounds["max"][axis]:
                raise CoverPackageError("sample origin is outside map bounds")
        if len(sample["directions"]) != slots:
            raise CoverPackageError("sample direction grid is incomplete")
        for direction_index, direction in enumerate(sample["directions"]):
            expected_ref = f"{sample['id']}/direction-{direction_index:03d}"
            expected_azimuth = direction_index * resolution["azimuth_step_degrees"]
            if direction["id"] != expected_ref or not math.isclose(
                direction["azimuth_degrees"], expected_azimuth, abs_tol=1e-9
            ):
                raise CoverPackageError("direction id or azimuth grid is invalid")
            if [profile["posture_id"] for profile in direction["profiles"]] != posture_ids:
                raise CoverPackageError("direction posture profiles are incomplete")
            for profile in direction["profiles"]:
                _validate_profile(profile, volume, postures, resolution)
                direction_profiles.add((direction["id"], profile["posture_id"]))
            direction_count += 1
    for left_index, left in enumerate(samples):
        for right in samples[left_index + 1 :]:
            if _sample_volumes_overlap(left, right):
                raise CoverPackageError("sample volumes overlap ambiguously")

    coverage = package["coverage"]
    if (
        coverage["sample_count"] != len(samples)
        or coverage["direction_count"] != direction_count
        or coverage["posture_ids"] != posture_ids
        or coverage["full_map_coverage"] is not False
    ):
        raise CoverPackageError("declared coverage differs from real package")
    _validate_review(package["geometry_validation"], approved_required=True)
    _validate_review(package["tactical_review"], approved_required=False)

    annotation_ids: set[str] = set()
    annotated_profiles: set[tuple[str, str]] = set()
    for annotation in package["tactical_annotations"]:
        if annotation["id"] in annotation_ids:
            raise CoverPackageError("tactical annotation ids are not unique")
        annotation_ids.add(annotation["id"])
        profile_ref = (
            annotation["geometry_direction_ref"],
            annotation["posture_id"],
        )
        if profile_ref not in direction_profiles:
            raise CoverPackageError("tactical annotation profile reference is unknown")
        if profile_ref in annotated_profiles:
            raise CoverPackageError("geometry profile is annotated more than once")
        annotated_profiles.add(profile_ref)
        if annotation["review_status"] == "pending_review":
            if (
                any(
                    annotation[field] is not None
                    for field in ("classification", "author_id", "reviewer_id")
                )
                or annotation["evidence"]
            ):
                raise CoverPackageError(
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
            raise CoverPackageError(
                "approved tactical annotation lacks attributable evidence"
            )
    if annotated_profiles != direction_profiles:
        raise CoverPackageError(
            "tactical annotations do not cover every geometry profile"
        )
    if package["tactical_review"]["status"] == "approved" and any(
        annotation["review_status"] != "approved"
        for annotation in package["tactical_annotations"]
    ):
        raise CoverPackageError(
            "approved tactical review contains pending annotations"
        )
    return {
        "samples": len(samples),
        "directions": direction_count,
        "profiles": len(direction_profiles),
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
    except (json.JSONDecodeError, CoverPackageError, SchemaError, ValidationError):
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
    except CoverPackageError:
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
    except CoverPackageError as exc:
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


def _validate_profile(
    profile: dict[str, Any],
    volume: dict[str, Any],
    postures: dict[str, dict[str, Any]],
    resolution: dict[str, Any],
) -> None:
    posture = postures[profile["posture_id"]]
    expected_grid = [
        (height, offset)
        for height in posture["probe_heights_units"]
        for offset in volume["lateral_offsets_units"]
    ]
    if profile["probe_count"] != len(expected_grid) or len(profile["probes"]) != len(
        expected_grid
    ):
        raise CoverPackageError("profile probe count is invalid")
    occluded = 0
    for probe, (height, offset) in zip(profile["probes"], expected_grid, strict=True):
        if not math.isclose(probe["height_units"], height, abs_tol=1e-9) or not math.isclose(
            probe["lateral_offset_units"], offset, abs_tol=1e-9
        ):
            raise CoverPackageError("profile probe grid is invalid")
        distance = probe["hit_distance_units"]
        if distance is not None and distance > resolution["maximum_ray_distance_units"]:
            raise CoverPackageError("probe hit exceeds maximum ray distance")
        expected = distance is not None and distance <= resolution["protection_distance_units"]
        if probe["occluded_within_protection_distance"] is not expected:
            raise CoverPackageError("probe occlusion does not match distance")
        occluded += int(expected)
    if profile["occluded_probe_count"] != occluded or profile[
        "complete_volume_occlusion"
    ] is not (occluded == len(expected_grid)):
        raise CoverPackageError("profile aggregate does not recompute")


def _validate_review(review: dict[str, Any], *, approved_required: bool) -> None:
    if approved_required and review["status"] != "approved":
        raise CoverPackageError("required geometry validation is not approved")
    if review["status"] == "pending_review":
        if review["author_id"] is not None or review["reviewer_id"] is not None:
            raise CoverPackageError("pending review is signed")
        if review["evidence"]:
            raise CoverPackageError("pending review contains fabricated evidence")
    elif not review["author_id"] or not review["reviewer_id"] or not review["evidence"]:
        raise CoverPackageError("approved review lacks provenance")


def _valid_support_entry(entry: dict[str, Any]) -> bool:
    expected_path = f"packages/{entry['map_id']}/{entry['asset_version']}/package.json"
    return (
        entry["package_path"] == expected_path
        and not entry["training_use_authorized"]
        and not (
            entry["product_supported"]
            and entry["tactical_review_status"] != "approved"
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


def _require_finite(value: Any, path: str) -> None:
    if isinstance(value, bool) or value is None:
        return
    if isinstance(value, (int, float)):
        if not math.isfinite(float(value)):
            raise CoverPackageError(f"{path} contains a non-finite number")
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
        raise CoverPackageError("JSON root must be an object")
    return value


def _safe_relative(reference: str) -> None:
    if (
        not reference
        or "\\" in reference
        or Path(reference).is_absolute()
        or ".." in Path(reference).parts
    ):
        raise CoverPackageError("asset reference is not a safe relative path")


def _safe_child(root: Path, reference: str) -> Path:
    _safe_relative(reference)
    resolved_root = root.resolve(strict=True)
    candidate = (resolved_root / reference).resolve(strict=True)
    if candidate == resolved_root or resolved_root not in candidate.parents:
        raise CoverPackageError("asset reference escapes its root")
    return candidate


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _unavailable(reason: str) -> PackageResolution:
    return PackageResolution(status="unavailable", reason_codes=(reason,))
