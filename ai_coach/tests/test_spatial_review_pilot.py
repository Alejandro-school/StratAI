from __future__ import annotations

import hashlib
import json
import shutil
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.contracts.validate_spatial_review_pilot import (
    SpatialReviewPilotValidationError,
    validate_spatial_review_pilot,
)
from ai_coach.spatial.review.build_pilot import (
    OUTPUT_NAME,
    SpatialReviewBatchError,
    build_pilot_batch,
)

ROOT = Path(__file__).resolve().parents[2]
SPATIAL = ROOT / "ai_coach" / "spatial"
MAPS = ROOT / "backend" / "data" / "maps"
PACKAGE_FILES = (
    SPATIAL / "exposure" / "packages" / "de_mirage" / "1.0.0-candidate.1" / "package.json",
    SPATIAL / "cover" / "packages" / "de_mirage" / "1.0.0-candidate.1" / "package.json",
    SPATIAL / "exit_route" / "packages" / "de_mirage" / "1.0.0-candidate.1" / "package.json",
)


def _empty_run(tmp_path: Path, name: str) -> Path:
    path = tmp_path / name
    path.mkdir()
    return path


def _load(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def _write(path: Path, value: dict) -> None:
    path.write_text(
        json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
        + "\n",
        encoding="utf-8",
    )


def _hashes(paths: tuple[Path, ...]) -> dict[str, str]:
    return {
        str(path.relative_to(ROOT)): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in paths
    }


def test_double_build_is_byte_identical_read_only_and_independently_validated(
    tmp_path: Path,
) -> None:
    before = _hashes(PACKAGE_FILES)
    first_path, first = build_pilot_batch(_empty_run(tmp_path, "first"))
    second_path, second = build_pilot_batch(_empty_run(tmp_path, "second"))
    assert first_path.read_bytes() == second_path.read_bytes()
    assert first == second
    assert _hashes(PACKAGE_FILES) == before

    result = validate_spatial_review_pilot(first_path)
    assert result == {
        "status": "valid",
        "batch_sha256": hashlib.sha256(first_path.read_bytes()).hexdigest(),
        "bytes": len(first_path.read_bytes()),
        "review_units": 12,
        "family_counts": {"exposure": 4, "cover": 4, "exit_route": 4},
        "candidate_population_by_family": {
            "exposure": 25,
            "cover": 384,
            "exit_route": 8,
        },
        "excluded_by_pilot_limit_by_family": {
            "exposure": 21,
            "cover": 380,
            "exit_route": 4,
        },
        "objective_geometry_available": 12,
        "observed_context_unavailable": 12,
        "responses": 0,
        "reviewer_identities": 0,
        "classifications": 0,
        "adjudications": 0,
        "recommendations": 0,
    }


def test_batch_separates_audit_geometry_context_and_empty_human_partitions(
    tmp_path: Path,
) -> None:
    output, batch = build_pilot_batch(_empty_run(tmp_path, "separation"))
    assert output.name == OUTPUT_NAME
    assert set(batch["authorization"].values()) == {False}
    assert [unit["display_order"] for unit in batch["review_units"]] == list(
        range(1, 13)
    )
    assert {unit["audit_context"]["family"] for unit in batch["review_units"]} == {
        "exposure",
        "cover",
        "exit_route",
    }
    for unit in batch["review_units"]:
        assert unit["state"] == "pending_review"
        assert unit["review_responses"] == []
        assert unit["adjudication"] is None
        assert unit["split_id"] == "pilot_review_only"
        view = unit["reviewer_view"]
        assert view["observed_context"] == {
            "status": "unavailable",
            "reason_codes": ["package_only_pilot_no_observed_context"],
            "availability_cutoff": None,
            "facts": {},
            "source_kind": "none_demo_not_opened",
        }
        assert "family" not in view
        assert "map_id" not in view
        assert "package_sha256" not in view
        assert "source_candidate_ref" not in view


def test_builder_rejects_non_temp_non_empty_and_unknown_map_before_write(
    tmp_path: Path,
) -> None:
    with pytest.raises(SpatialReviewBatchError, match="under_system_temp"):
        build_pilot_batch(ROOT / "ai_coach")

    non_empty = _empty_run(tmp_path, "non-empty")
    (non_empty / "keep.txt").write_text("keep", encoding="utf-8")
    with pytest.raises(SpatialReviewBatchError, match="must_be_empty"):
        build_pilot_batch(non_empty)
    assert [item.name for item in non_empty.iterdir()] == ["keep.txt"]

    unknown = _empty_run(tmp_path, "unknown-map")
    with pytest.raises(SpatialReviewBatchError, match="map_not_registered"):
        build_pilot_batch(unknown, map_id="de_unknown")
    assert list(unknown.iterdir()) == []


def test_builder_rejects_support_hash_mismatch_without_writing(tmp_path: Path) -> None:
    copied = tmp_path / "exposure-copy"
    shutil.copytree(SPATIAL / "exposure", copied)
    manifest_path = copied / "support_manifest.json"
    manifest = _load(manifest_path)
    manifest["packages"][0]["package_sha256"] = "0" * 64
    _write(manifest_path, manifest)
    output = _empty_run(tmp_path, "bad-support-output")
    with pytest.raises(SpatialReviewBatchError, match="package_hash_mismatch"):
        build_pilot_batch(output, exposure_root=copied)
    assert list(output.iterdir()) == []


def test_builder_rejects_orphan_reference_without_writing(tmp_path: Path) -> None:
    copied = tmp_path / "exit-copy"
    shutil.copytree(SPATIAL / "exit_route", copied)
    package_path = (
        copied
        / "packages"
        / "de_mirage"
        / "1.0.0-candidate.1"
        / "package.json"
    )
    package = _load(package_path)
    package["tactical_annotations"][0]["candidate_ref"] = (
        "exit-sample-999/candidate-999"
    )
    _write(package_path, package)
    manifest_path = copied / "support_manifest.json"
    manifest = _load(manifest_path)
    manifest["packages"][0]["package_sha256"] = hashlib.sha256(
        package_path.read_bytes()
    ).hexdigest()
    _write(manifest_path, manifest)
    output = _empty_run(tmp_path, "orphan-output")
    with pytest.raises(SpatialReviewBatchError, match="package_invalid"):
        build_pilot_batch(output, exit_route_root=copied, maps_root=MAPS)
    assert list(output.iterdir()) == []


@pytest.mark.parametrize("mutation", ["response", "geometry", "privacy", "split"])
def test_independent_validator_rejects_human_data_tampering_and_leakage(
    tmp_path: Path, mutation: str
) -> None:
    valid_path, _ = build_pilot_batch(_empty_run(tmp_path, f"valid-{mutation}"))
    batch = _load(valid_path)
    unit = batch["review_units"][0]
    if mutation == "response":
        unit["review_responses"] = [{}]
    elif mutation == "geometry":
        unit["reviewer_view"]["objective_geometry"]["selection_radius_units"] += 1
    elif mutation == "privacy":
        unit["reviewer_view"]["steam_id"] = "76561198000000000"
    else:
        unit["split_id"] = "training"
    tampered_root = _empty_run(tmp_path, f"tampered-{mutation}")
    tampered_path = tampered_root / OUTPUT_NAME
    _write(tampered_path, batch)
    with pytest.raises((SpatialReviewPilotValidationError, ValidationError)):
        validate_spatial_review_pilot(tampered_path)


def test_validator_rejects_batch_outside_temp_without_reading() -> None:
    with pytest.raises(SpatialReviewPilotValidationError, match="system temp"):
        validate_spatial_review_pilot(
            ROOT / "ai_coach" / "contracts" / "spatial_human_review_batch.schema.json"
        )
