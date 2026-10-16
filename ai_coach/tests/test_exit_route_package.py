from __future__ import annotations

import copy
import hashlib
import json
from pathlib import Path

import pytest
from jsonschema import Draft202012Validator, ValidationError

from ai_coach.spatial.exit_route.loader import (
    ExitRoutePackageError,
    load_package,
    select_sample,
    validate_package,
)

ROOT = Path(__file__).resolve().parents[2]
REGISTRY = ROOT / "ai_coach" / "spatial" / "exit_route"
MAPS = ROOT / "backend" / "data" / "maps"
PACKAGE_PATH = (
    REGISTRY / "packages" / "de_mirage" / "1.0.0-candidate.1" / "package.json"
)


def _package() -> dict:
    return json.loads(PACKAGE_PATH.read_text(encoding="utf-8"))


def test_real_schema_manifest_loader_and_go_contract_parity() -> None:
    package = _package()
    counts = validate_package(package, maps_root=MAPS)
    assert counts == {"samples": 8, "candidates": 8, "annotations": 8}
    resolved = load_package(REGISTRY, MAPS, "de_mirage")
    assert resolved.status == "available"
    assert (
        resolved.package_sha256 == hashlib.sha256(PACKAGE_PATH.read_bytes()).hexdigest()
    )
    assert resolved.support is not None
    assert resolved.support["product_supported"] is False
    assert resolved.support["training_use_authorized"] is False
    assert package["navigation_contract"] == {
        "graph_contract_version": "nav_area_connections_with_polygons@2",
        "shortest_path_algorithm": "directed_dijkstra_stable_ties@1",
        "endpoint_association_method": "containing_nav_polygon_xy_then_nearest_center_3d@2",
        "edge_weight_method": "nav_area_center_euclidean_world_units@1",
        "endpoint_tolerance_world_units": 96,
        "endpoint_vertical_tolerance_world_units": 128,
    }


def test_unknown_map_and_versions_fail_closed_without_mirage_fallback() -> None:
    assert load_package(REGISTRY, MAPS, "de_inferno").reason_codes == (
        "map_not_registered",
    )
    assert load_package(
        REGISTRY, MAPS, "de_mirage", contract_version="2.0.0"
    ).reason_codes == ("contract_version_incompatible",)
    assert load_package(
        REGISTRY, MAPS, "de_mirage", asset_version="9.0.0-candidate.1"
    ).reason_codes == ("asset_version_incompatible",)


def test_hash_map_reference_and_source_tampering_fail_closed(tmp_path: Path) -> None:
    registry = tmp_path / "registry"
    maps = tmp_path / "maps"
    _copy_registry(registry)
    _copy_sources(maps)
    package_path = (
        registry / "packages" / "de_mirage" / "1.0.0-candidate.1" / "package.json"
    )
    package_path.write_bytes(package_path.read_bytes() + b" ")
    assert load_package(registry, maps, "de_mirage").reason_codes == (
        "package_hash_mismatch",
    )

    _copy_registry(registry, reset=True)
    manifest_path = registry / "support_manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest["packages"][0]["package_path"] = "../escape.json"
    manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
    assert load_package(registry, maps, "de_mirage").reason_codes == (
        "package_invalid",
    )

    _copy_registry(registry, reset=True)
    (maps / "de_mirage" / "de_mirage.nav").write_bytes(b"tampered")
    assert load_package(registry, maps, "de_mirage").reason_codes == (
        "source_asset_mismatch",
    )


def test_bounds_overlap_coverage_and_human_label_are_rejected() -> None:
    base = _package()
    cases = []
    outside = copy.deepcopy(base)
    outside["route_samples"][0]["candidates"][0]["destination_world"]["x"] = 999999
    cases.append(outside)
    coverage = copy.deepcopy(base)
    coverage["coverage"]["candidate_count"] += 1
    cases.append(coverage)
    overlap = copy.deepcopy(base)
    overlap["route_samples"][1]["origin_world"] = copy.deepcopy(
        overlap["route_samples"][0]["origin_world"]
    )
    cases.append(overlap)
    fabricated = copy.deepcopy(base)
    fabricated["tactical_annotations"][0]["classification"] = "safe"
    cases.append(fabricated)
    broken = copy.deepcopy(base)
    broken["tactical_annotations"][0]["candidate_ref"] = "exit-sample-999/candidate-001"
    cases.append(broken)
    for package in cases:
        with pytest.raises(ExitRoutePackageError):
            validate_package(package)


def test_selection_is_partial_height_aware_and_ambiguous_fail_closed() -> None:
    package = _package()
    origin = package["route_samples"][0]["origin_world"]
    selected, reason = select_sample(package, origin)
    assert selected is not None and reason is None
    outside = dict(origin)
    outside["z"] += package["route_samples"][0]["vertical_tolerance_units"] + 1
    assert select_sample(package, outside) == (None, "outside_declared_sample_coverage")
    ambiguous = copy.deepcopy(package)
    ambiguous["route_samples"][1]["origin_world"] = copy.deepcopy(origin)
    assert select_sample(ambiguous, origin) == (None, "ambiguous_sample_overlap")


def test_observed_schema_forbids_future_oracle_outcomes_raw_identity_and_human_claims() -> (
    None
):
    schema = json.loads(
        (
            ROOT
            / "ai_coach"
            / "contracts"
            / "observed_exit_route_diagnostic_v2.schema.json"
        ).read_text(encoding="utf-8")
    )
    Draft202012Validator.check_schema(schema)
    candidate_schema = schema["$defs"]["candidate"]
    forbidden = {
        "future",
        "oracle",
        "outcomes",
        "steam_id",
        "player_name",
        "recommendation",
    }
    assert forbidden.isdisjoint(candidate_schema["properties"])
    tactical = schema["$defs"]["tactical_assessment"]["properties"]
    assert tactical["status"]["const"] == "unavailable"
    assert tactical["reason_codes"]["const"] == [
        "tactical_decision_context_required"
    ]
    assert set(tactical) == {"status", "reason_codes"}
    invalid = {key: None for key in schema["required"]}
    invalid["oracle"] = {}
    with pytest.raises(ValidationError):
        Draft202012Validator(schema).validate(invalid)


def _copy_registry(destination: Path, *, reset: bool = False) -> None:
    import shutil

    if reset and destination.exists():
        shutil.rmtree(destination)
    shutil.copytree(REGISTRY, destination, dirs_exist_ok=True)


def _copy_sources(destination: Path) -> None:
    import shutil

    for reference in [source["reference"] for source in _package()["source_assets"]]:
        target = destination / reference
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(MAPS / reference, target)
