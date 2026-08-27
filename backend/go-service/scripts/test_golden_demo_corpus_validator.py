import copy
import hashlib
import json
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

import golden_demo_corpus_validator as validator


def write_json(path: Path, value: object) -> None:
    path.write_text(
        json.dumps(value, ensure_ascii=True, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )


def refresh_digests(manifest: dict[str, object]) -> None:
    sources = manifest["sources"]
    assert isinstance(sources, list)
    manifest["candidate_pool_sha256"] = validator.candidate_pool_digest(sources)
    manifest["selection_sha256"] = validator.selection_digest(sources)


def build_fixture(tmp_path: Path) -> tuple[Path, dict[str, Path], dict[str, object]]:
    roots = {
        "stratai_runtime": tmp_path / "stratai-demos",
        "faceit_offline": tmp_path / "faceit-demos",
    }
    for root in roots.values():
        root.mkdir()
    sources: list[dict[str, object]] = []
    for map_index, map_name in enumerate(validator.MAP_ORDER):
        quota = validator.MAP_QUOTAS[map_name]
        map_sources: list[dict[str, object]] = []
        for item_index in range(quota):
            pool = validator.SOURCE_POOLS[(map_index + item_index) % 2]
            payload = (
                validator.DEMO_MAGIC
                + map_name.encode("ascii")
                + b"\x00"
                + f"fixture-{map_index}-{item_index}".encode("ascii")
                + b"x" * (map_index * 1_000 + item_index * 10)
            )
            checksum = hashlib.sha256(payload).hexdigest()
            path = roots[pool] / f"fixture-{map_index}-{item_index}.dem"
            path.write_bytes(payload)
            if pool == "stratai_runtime":
                provenance = {
                    "inventory_method": "sha256_file_audit@1",
                    "checksum_evidence": "persisted_sidecar_verified",
                    "ownership_evidence": "unrecorded",
                    "bundle_linkage": "checksum_linked_legacy_bundle",
                }
            else:
                provenance = {
                    "inventory_method": "sha256_file_audit@1",
                    "checksum_evidence": "audit_recomputed",
                    "ownership_evidence": "unrecorded",
                    "bundle_linkage": "none",
                }
            map_sources.append(
                {
                    "source_alias": f"demo-{checksum[:20]}",
                    "sha256": checksum,
                    "bytes": len(payload),
                    "map": map_name,
                    "source_pool": pool,
                    "provenance": provenance,
                    "selection": {},
                    "_path": path,
                }
            )
        map_sources.sort(key=lambda item: str(item["sha256"]))
        for rank, source in enumerate(map_sources, 1):
            source["selection"] = {
                "selected": True,
                "rank_within_map": rank,
                "reason": "selected_within_map_quota",
            }
            source.pop("_path")
        sources.extend(map_sources)
    manifest: dict[str, object] = {
        "schema_id": validator.SCHEMA_ID,
        "corpus_id": validator.CORPUS_ID,
        "version": validator.CORPUS_VERSION,
        "training_allowed": False,
        "candidate_count": len(sources),
        "selected_count": len(sources),
        "candidate_pool_sha256": "",
        "selection_sha256": "",
        "contracts": copy.deepcopy(validator.EXPECTED_CONTRACTS),
        "selection_algorithm": copy.deepcopy(validator.EXPECTED_ALGORITHM),
        "gate_1": copy.deepcopy(validator.PARTIAL_GATE_DECLARATION),
        "semantic_validation": {
            "status": "not_evaluated",
            "evaluated_count": 0,
            "passed_count": 0,
            "quarantined_count": 0,
            "cases": [],
        },
        "sources": sources,
    }
    refresh_digests(manifest)
    manifest_path = tmp_path / "manifest.json"
    write_json(manifest_path, manifest)
    return manifest_path, roots, manifest


def test_committed_manifest_is_metadata_valid_but_does_not_pass_gate_1() -> None:
    manifest_path = (
        SCRIPT_DIR.parent
        / "testdata"
        / "golden-demos"
        / "v2"
        / "manifest.json"
    )
    result = validator.validate_golden_demo_corpus(
        manifest_path, metadata_only=True
    )
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    semantic = manifest["semantic_validation"]
    assert result["status"] == "metadata_valid"
    assert result["gate_1_proven"] is False
    assert result["semantic_validation_status"] == "partial"
    assert result["semantic_evaluated_count"] == semantic["evaluated_count"]
    assert result["semantic_quarantined_count"] == semantic["quarantined_count"]
    assert result["selected_count"] == 40
    assert result["errors"] == []


def test_resolved_fixture_inventory_is_valid_without_claiming_gate_1(
    tmp_path: Path,
) -> None:
    manifest_path, roots, _ = build_fixture(tmp_path)
    result = validator.validate_golden_demo_corpus(manifest_path, roots=roots)
    assert result["status"] == "source_inventory_valid"
    assert result["gate_1_proven"] is False
    assert result["errors"] == []


def test_rejects_source_hash_mismatch(tmp_path: Path) -> None:
    manifest_path, roots, manifest = build_fixture(tmp_path)
    first = manifest["sources"][0]
    assert isinstance(first, dict)
    source_path = next(
        path
        for path in roots[str(first["source_pool"])].glob("*.dem")
        if path.stat().st_size == first["bytes"]
        and hashlib.sha256(path.read_bytes()).hexdigest() == first["sha256"]
    )
    payload = bytearray(source_path.read_bytes())
    payload[-1] ^= 1
    source_path.write_bytes(payload)
    result = validator.validate_golden_demo_corpus(manifest_path, roots=roots)
    assert any("sha256 no coincide" in error for error in result["errors"])


def test_rejects_source_size_mismatch(tmp_path: Path) -> None:
    manifest_path, roots, manifest = build_fixture(tmp_path)
    first = manifest["sources"][0]
    assert isinstance(first, dict)
    source_path = next(
        path
        for path in roots[str(first["source_pool"])].glob("*.dem")
        if path.stat().st_size == first["bytes"]
        and hashlib.sha256(path.read_bytes()).hexdigest() == first["sha256"]
    )
    source_path.write_bytes(source_path.read_bytes() + b"x")
    result = validator.validate_golden_demo_corpus(manifest_path, roots=roots)
    assert any("bytes no coincide" in error for error in result["errors"])


def test_rejects_selected_count_outside_30_to_50(tmp_path: Path) -> None:
    manifest_path, _, manifest = build_fixture(tmp_path)
    manifest["selected_count"] = 29
    write_json(manifest_path, manifest)
    result = validator.validate_golden_demo_corpus(
        manifest_path, metadata_only=True
    )
    assert any("entre 30 y 50" in error for error in result["errors"])


def test_rejects_duplicate_source(tmp_path: Path) -> None:
    manifest_path, _, manifest = build_fixture(tmp_path)
    sources = manifest["sources"]
    assert isinstance(sources, list)
    sources[1] = copy.deepcopy(sources[0])
    refresh_digests(manifest)
    write_json(manifest_path, manifest)
    result = validator.validate_golden_demo_corpus(
        manifest_path, metadata_only=True
    )
    assert any("sha256 duplicado" in error for error in result["errors"])
    assert any("source_alias duplicado" in error for error in result["errors"])


def test_rejects_private_path_or_identifier_field(tmp_path: Path) -> None:
    manifest_path, _, manifest = build_fixture(tmp_path)
    sources = manifest["sources"]
    assert isinstance(sources, list)
    sources[0]["path"] = "C:\\private\\match_identifier.dem"
    write_json(manifest_path, manifest)
    result = validator.validate_golden_demo_corpus(
        manifest_path, metadata_only=True
    )
    assert any("referencia privada" in error for error in result["errors"])


def test_rejects_incompatible_contract_version(tmp_path: Path) -> None:
    manifest_path, _, manifest = build_fixture(tmp_path)
    manifest["contracts"]["parser_schema_version"] = "v15"
    write_json(manifest_path, manifest)
    result = validator.validate_golden_demo_corpus(
        manifest_path, metadata_only=True
    )
    assert any("version incompatible" in error for error in result["errors"])


def test_rejects_selection_algorithm_change(tmp_path: Path) -> None:
    manifest_path, _, manifest = build_fixture(tmp_path)
    manifest["selection_algorithm"]["order_key"] = "filename"
    write_json(manifest_path, manifest)
    result = validator.validate_golden_demo_corpus(
        manifest_path, metadata_only=True
    )
    assert any("algoritmo o cuotas incompatibles" in error for error in result["errors"])


def test_rejects_semantic_evidence_for_unknown_source(tmp_path: Path) -> None:
    manifest_path, _, manifest = build_fixture(tmp_path)
    manifest["semantic_validation"] = {
        "status": "partial",
        "evaluated_count": 1,
        "passed_count": 1,
        "quarantined_count": 0,
        "cases": [
            {
                "source_alias": "demo-00000000000000000000",
                "status": "passed",
                "checks": ["canonical_export"],
                "failure_domain": None,
                "reason": None,
                "notes": [],
                "runner_version": validator.LEGACY_MANUAL_RUNNER_VERSION,
            }
        ],
    }
    write_json(manifest_path, manifest)
    result = validator.validate_golden_demo_corpus(
        manifest_path, metadata_only=True
    )
    assert any("no referencia una fuente seleccionada" in error for error in result["errors"])
