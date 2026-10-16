from __future__ import annotations

import copy
import hashlib
import json
import math
import shutil
from pathlib import Path

import pytest

from ai_coach.spatial.exposure.loader import (
    CONTRACT_VERSION,
    ExposurePackageError,
    build_sectors,
    load_package,
    relative_sectors,
    sector_contains,
    validate_package,
)

ROOT = Path(__file__).parents[2]
REGISTRY_ROOT = ROOT / "ai_coach" / "spatial" / "exposure"
MAPS_ROOT = ROOT / "backend" / "data" / "maps"
PACKAGE_PATH = (
    REGISTRY_ROOT / "packages" / "de_mirage" / "1.0.0-candidate.1" / "package.json"
)


def _load_json(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def _write_json(path: Path, value: dict) -> None:
    path.write_text(
        json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )


def _copied_registry(tmp_path: Path) -> Path:
    destination = tmp_path / "registry"
    shutil.copytree(REGISTRY_ROOT, destination)
    return destination


def test_mirage_reference_package_is_valid_partial_and_unsigned_tactically() -> None:
    package = _load_json(PACKAGE_PATH)
    counts = validate_package(package, maps_root=MAPS_ROOT)
    resolution = load_package(REGISTRY_ROOT, MAPS_ROOT, "de_mirage")

    assert counts == {"samples": 8, "sectors": 25, "annotations": 25}
    assert resolution.status == "available"
    assert resolution.reason_codes == ()
    assert resolution.package == package
    assert (
        resolution.package_sha256
        == hashlib.sha256(PACKAGE_PATH.read_bytes()).hexdigest()
    )
    assert resolution.support == {
        "map_id": "de_mirage",
        "contract_version": CONTRACT_VERSION,
        "asset_version": "1.0.0-candidate.1",
        "package_path": "packages/de_mirage/1.0.0-candidate.1/package.json",
        "package_sha256": resolution.package_sha256,
        "implementation_status": "candidate",
        "geometry_status": "validated",
        "tactical_review_status": "pending_review",
        "product_supported": False,
        "training_use_authorized": False,
    }
    assert package["coverage"] == {
        "declaration": "partial_reference_samples",
        "full_map_coverage": False,
        "sample_count": 8,
        "sector_count": 25,
    }
    assert package["tactical_review"] == {
        "status": "pending_review",
        "author_id": None,
        "reviewer_id": None,
        "evidence": [],
    }
    assert [sample["id"] for sample in package["geometry_samples"]] == [
        f"sample-{index:03d}" for index in range(1, 9)
    ]


def test_common_angle_math_wraps_and_uses_half_open_intervals() -> None:
    rays = [
        {"azimuth_degrees": 0, "hit_distance_units": None, "clearance_met": True},
        {"azimuth_degrees": 90, "hit_distance_units": 10, "clearance_met": False},
        {"azimuth_degrees": 180, "hit_distance_units": 10, "clearance_met": False},
        {"azimuth_degrees": 270, "hit_distance_units": None, "clearance_met": True},
    ]
    sectors = build_sectors("sample-001", rays, 90)
    assert sectors == [
        {
            "id": "sample-001/sector-001",
            "start_degrees": 225.0,
            "width_degrees": 180,
        }
    ]
    assert all(sector_contains(sectors[0], angle) for angle in (225, 359, 0, 44.999))
    assert not sector_contains(sectors[0], 45)
    assert not sector_contains(sectors[0], 224.999)
    assert relative_sectors(sectors, 225) == [
        {
            "id": "sample-001/sector-001",
            "start_degrees": 0.0,
            "width_degrees": 180.0,
        }
    ]


@pytest.mark.parametrize(
    ("mutate", "message"),
    [
        (
            lambda package: package["geometry_samples"][0]["origin_world"].__setitem__(
                "x", math.nan
            ),
            "schema violation|non-finite",
        ),
        (
            lambda package: package["geometry_samples"][0]["origin_world"].__setitem__(
                "x", package["bounds"]["max"]["x"] + 1
            ),
            "outside map bounds",
        ),
        (
            lambda package: package["coverage"].__setitem__("sample_count", 99),
            "declared coverage",
        ),
        (
            lambda package: package["geometry_samples"][1].__setitem__(
                "origin_world",
                copy.deepcopy(package["geometry_samples"][0]["origin_world"]),
            ),
            "overlap",
        ),
        (
            lambda package: package["tactical_review"].__setitem__(
                "author_id", "fabricated-human"
            ),
            "pending review",
        ),
        (
            lambda package: package["geometry_samples"][0][
                "origin_provenance"
            ].__setitem__("source_role", "undeclared"),
            "not declared",
        ),
    ],
)
def test_package_validator_rejects_invalid_or_ambiguous_content(
    mutate, message: str
) -> None:
    package = _load_json(PACKAGE_PATH)
    mutate(package)
    with pytest.raises(ExposurePackageError, match=message):
        validate_package(package)


def test_loader_fails_closed_without_cross_map_fallback() -> None:
    for map_id in (
        "de_dust2",
        "de_inferno",
        "de_ancient",
        "de_nuke",
        "de_anubis",
        "de_train",
        "de_unknown",
    ):
        resolution = load_package(REGISTRY_ROOT, MAPS_ROOT, map_id)
        assert resolution.status == "unavailable"
        assert resolution.reason_codes == ("map_not_registered",)
        assert resolution.package is None
    assert load_package(
        REGISTRY_ROOT, MAPS_ROOT, "de_mirage", contract_version="2.0.0"
    ).reason_codes == ("contract_version_incompatible",)
    assert load_package(
        REGISTRY_ROOT, MAPS_ROOT, "de_mirage", asset_version="9.9.9"
    ).reason_codes == ("asset_version_incompatible",)


def test_loader_rejects_missing_tampered_mismatched_and_inconsistent_packages(
    tmp_path: Path,
) -> None:
    registry = _copied_registry(tmp_path / "missing")
    (registry / "packages/de_mirage/1.0.0-candidate.1/package.json").unlink()
    assert load_package(registry, MAPS_ROOT, "de_mirage").reason_codes == (
        "package_file_missing",
    )

    registry = _copied_registry(tmp_path / "tampered")
    package_path = registry / "packages/de_mirage/1.0.0-candidate.1/package.json"
    package_path.write_bytes(package_path.read_bytes() + b" ")
    assert load_package(registry, MAPS_ROOT, "de_mirage").reason_codes == (
        "package_hash_mismatch",
    )

    registry = _copied_registry(tmp_path / "map-mismatch")
    package_path = registry / "packages/de_mirage/1.0.0-candidate.1/package.json"
    package = _load_json(package_path)
    package["map_id"] = "de_other"
    _write_json(package_path, package)
    manifest_path = registry / "support_manifest.json"
    manifest = _load_json(manifest_path)
    manifest["packages"][0]["package_sha256"] = hashlib.sha256(
        package_path.read_bytes()
    ).hexdigest()
    _write_json(manifest_path, manifest)
    assert load_package(registry, MAPS_ROOT, "de_mirage").reason_codes == (
        "map_id_mismatch",
    )

    registry = _copied_registry(tmp_path / "review-mismatch")
    manifest_path = registry / "support_manifest.json"
    manifest = _load_json(manifest_path)
    manifest["packages"][0]["tactical_review_status"] = "approved"
    _write_json(manifest_path, manifest)
    assert load_package(registry, MAPS_ROOT, "de_mirage").reason_codes == (
        "package_invalid",
    )

    empty_maps = tmp_path / "empty-maps"
    empty_maps.mkdir()
    assert load_package(REGISTRY_ROOT, empty_maps, "de_mirage").reason_codes == (
        "geometry_source_mismatch",
    )
