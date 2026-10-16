from __future__ import annotations

import copy
import hashlib
import json
import math
import shutil
from pathlib import Path

import pytest

from ai_coach.spatial.cover.loader import (
    CoverPackageError,
    load_package,
    normalize_degrees,
    profile_for_posture,
    select_sample,
    validate_package,
)

ROOT = Path(__file__).parents[2]
REGISTRY_ROOT = ROOT / "ai_coach" / "spatial" / "cover"
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


def test_mirage_cover_package_is_partial_objective_and_tactically_unsigned() -> None:
    package = _load_json(PACKAGE_PATH)
    counts = validate_package(package, maps_root=MAPS_ROOT)
    resolution = load_package(REGISTRY_ROOT, MAPS_ROOT, "de_mirage")

    assert counts == {
        "samples": 8,
        "directions": 192,
        "profiles": 384,
        "annotations": 384,
    }
    assert resolution.status == "available"
    assert resolution.reason_codes == ()
    assert resolution.package == package
    assert resolution.package_sha256 == hashlib.sha256(PACKAGE_PATH.read_bytes()).hexdigest()
    assert package["coverage"] == {
        "declaration": "partial_reference_samples",
        "full_map_coverage": False,
        "sample_count": 8,
        "direction_count": 192,
        "posture_ids": ["crouching", "standing"],
    }
    assert package["tactical_review"] == {
        "status": "pending_review",
        "author_id": None,
        "reviewer_id": None,
        "evidence": [],
    }
    assert all(
        annotation["classification"] is None
        and annotation["author_id"] is None
        and annotation["reviewer_id"] is None
        and annotation["evidence"] == []
        and annotation["review_status"] == "pending_review"
        for annotation in package["tactical_annotations"]
    )


def test_multi_height_multi_offset_profiles_recompute_and_direction_wraps() -> None:
    package = _load_json(PACKAGE_PATH)
    direction = package["geometry_samples"][0]["directions"][0]
    for posture_id in ("crouching", "standing"):
        profile = profile_for_posture(direction, posture_id)
        assert profile is not None
        assert profile["probe_count"] == 9
        assert profile["occluded_probe_count"] == sum(
            probe["occluded_within_protection_distance"]
            for probe in profile["probes"]
        )
        assert profile["complete_volume_occlusion"] is (
            profile["occluded_probe_count"] == 9
        )
        assert len({probe["height_units"] for probe in profile["probes"]}) == 3
        assert len({probe["lateral_offset_units"] for probe in profile["probes"]}) == 3
    assert normalize_degrees(-15) == 345
    assert normalize_degrees(360) == 0
    assert normalize_degrees(725) == 5


def test_sample_selection_is_bounded_and_fails_closed_on_ambiguity() -> None:
    package = _load_json(PACKAGE_PATH)
    sample = package["geometry_samples"][0]
    selected, reason = select_sample(package, sample["origin_world"])
    assert selected == sample
    assert reason is None
    selected, reason = select_sample(package, {"x": 999999, "y": 999999, "z": 0})
    assert selected is None
    assert reason == "outside_declared_sample_coverage"
    selected, reason = select_sample(package, {"x": math.nan, "y": 0, "z": 0})
    assert selected is None
    assert reason == "position_unavailable"


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
            lambda package: package["coverage"].__setitem__("direction_count", 99),
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
            lambda package: package["player_volume"]["postures"][0].__setitem__(
                "probe_heights_units", [18, 60]
            ),
            "probe heights",
        ),
        (
            lambda package: package["geometry_samples"][0]["directions"][0][
                "profiles"
            ][0].__setitem__("occluded_probe_count", 99),
            "aggregate",
        ),
        (
            lambda package: package["tactical_annotations"][0].__setitem__(
                "author_id", "fabricated-human"
            ),
            "pending tactical annotation",
        ),
    ],
)
def test_package_validator_rejects_invalid_ambiguous_or_unreviewed_content(
    mutate, message: str
) -> None:
    package = _load_json(PACKAGE_PATH)
    mutate(package)
    with pytest.raises(CoverPackageError, match=message):
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
