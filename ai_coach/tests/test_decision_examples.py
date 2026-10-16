from __future__ import annotations

import copy
import gzip
import hashlib
import json
from pathlib import Path

import pytest

from ai_coach.contracts.runtime_contracts import (
    CONTRACT_VERSION,
    DECISION_EXAMPLE_SCHEMA_ID,
    TASK_ELIGIBILITY_SCHEMA_ID,
    RuntimeContractError,
    validate_decision_example,
    validate_task_eligibility,
)
from ai_coach.datasets.build_examples import (
    build_audit_summary,
    build_direct_bundle_audit_summary,
    iter_bundle_summaries,
    process_bundle,
)
from ai_coach.datasets.eligibility import evaluate_task_eligibility
from ai_coach.datasets.streaming import DatasetContractError

CATALOG_PATH = Path(__file__).parents[1] / "contracts" / "capability_catalog.json"
DIRECT_CHECKSUM = "a" * 64
DIRECT_ALIAS = "demo-" + DIRECT_CHECKSUM[:20]


def _write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, sort_keys=True), encoding="utf-8")


def _write_jsonl(
    path: Path, values: list[dict[str, object]], *, zipped: bool = False
) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    raw = "".join(json.dumps(value, sort_keys=True) + "\n" for value in values).encode()
    if zipped:
        path.write_bytes(gzip.compress(raw, mtime=0))
    else:
        path.write_bytes(raw)


def _artifact(
    artifact_type: str,
    path: str,
    schema_id: str,
    *,
    compression: str = "",
    record_count: int = 1,
) -> dict[str, object]:
    return {
        "artifact_type": artifact_type,
        "path": path,
        "schema_id": schema_id,
        "format": "jsonl",
        "compression": compression,
        "record_count": record_count,
        "sha256": "synthetic-not-verified-in-this-read-only-phase",
        "bytes": 1,
        "sort_order": [],
    }


def _bundle(root: Path, name: str = "match-1") -> Path:
    bundle = root / name
    canonical = bundle / "canonical"
    match_id = name
    actor_id = "actor-1"
    state_id = f"{match_id}:state:001:000000090:{actor_id}"
    decision_id = f"{match_id}:trade:1"
    common = {
        "match_id": match_id,
        "decision_id": decision_id,
        "round_number": 1,
        "t0_tick": 100,
    }
    decision = {
        "schema_id": "stratai.decision@1",
        **common,
        "actor_player_id": actor_id,
        "actor_id_usage": "join_only",
        "observed_state_ref": state_id,
        "state_availability_status": "observed",
        "decision_type": "spacing_or_trade_connection",
        "action_taken": "connected",
        "availability_tick": 100,
        "availability_status": "observed",
        "causal_role": "decision",
        "visibility_scope": "observed_physical_proxy",
        "source": "trades@1",
        "source_record_id": decision_id,
        "source_event_ids": ["event-1"],
        "algorithm_version": "synthetic@1",
    }
    features = {
        "schema_id": "stratai.decision_features@1",
        **common,
        "decision_type": "spacing_or_trade_connection",
        "availability_tick_max": 90,
        "observed_participant_states": 2,
        "trade_possible": True,
        "trade_possible_status": "derived",
        "nearest_connection_time_ms": 50.0,
        "nearest_connection_time_status": "derived",
        "any_line_of_sight": True,
        "line_of_sight_status": "derived",
        "minimum_facing_delta_deg": 10.0,
        "facing_status": "derived",
        "round_clock_remaining_ms": 90000,
        "bomb_time_remaining_ms": None,
    }
    oracle = {
        "schema_id": "stratai.oracle_context@1",
        **common,
        "available": False,
        "status": "unavailable",
        "field_names": [],
        "abstentions": ["synthetic oracle absent"],
    }
    outcome = {
        "schema_id": "stratai.decision_outcome@1",
        **common,
        "outcome_tick": 110,
        "outcome": "completed",
        "horizons": [
            {
                "horizon_seconds": 2,
                "status": "derived_outcome_only",
                "outcome": "completed",
            },
            {
                "horizon_seconds": 5,
                "status": "derived_outcome_only",
                "outcome": "completed",
            },
            {
                "horizon_seconds": 10,
                "status": "derived_outcome_only",
                "outcome": "completed",
            },
        ],
    }
    quality = {
        "schema_id": "stratai.quality_mask@1",
        **common,
        "available_fields": ["any_line_of_sight", "trade_possible"],
        "unavailable_fields": ["path_distance"],
        "inferred_fields": [],
        "warning_flags": ["synthetic fixture"],
    }
    player_state = {
        "schema_id": "stratai.player_state@3",
        "match_id": match_id,
        "state_id": state_id,
        "round_id": f"{match_id}:round:001",
        "round_number": 1,
        "tick": 90,
        "player_id": actor_id,
        "team_id": "team-a",
        "health": 100,
        "objective_phase": "preplant",
    }
    tactical_state = {
        "schema_id": "stratai.tactical_physical_observation@1",
        "match_id": match_id,
        "join_keys": {
            "match_id": match_id,
            "round_id": f"{match_id}:round:001",
            "observer_id": actor_id,
            "subject_id": actor_id,
        },
        "round_number": 1,
        "tick": 96,
        "availability_tick": 95,
        "status": "observed",
        "causal_role": "model_input_observation",
        "visibility_scope": "self",
        "source": "synthetic",
        "state": {"health": 100, "field_availability": {"health": "observed"}},
    }
    artifacts = [
        _artifact(
            "decision_features",
            "causal/decision_features.jsonl",
            "stratai.decision_features@1",
        ),
        _artifact("decisions", "causal/decisions.jsonl", "stratai.decision@1"),
        _artifact(
            "oracle_context", "causal/oracle_context.jsonl", "stratai.oracle_context@1"
        ),
        _artifact(
            "decision_outcomes", "causal/outcomes.jsonl", "stratai.decision_outcome@1"
        ),
        _artifact(
            "quality_masks", "causal/quality_masks.jsonl", "stratai.quality_mask@1"
        ),
        _artifact(
            "player_states",
            "states/player_states/round_001.jsonl",
            "stratai.player_state@3",
        ),
        _artifact(
            "tactical_observations",
            "states/tactical/observed.jsonl.gz",
            "stratai.tactical_physical_observation@1",
            compression="gzip",
        ),
    ]
    _write_jsonl(canonical / "causal/decisions.jsonl", [decision])
    _write_jsonl(canonical / "causal/decision_features.jsonl", [features])
    _write_jsonl(canonical / "causal/oracle_context.jsonl", [oracle])
    _write_jsonl(canonical / "causal/outcomes.jsonl", [outcome])
    _write_jsonl(canonical / "causal/quality_masks.jsonl", [quality])
    _write_jsonl(canonical / "states/player_states/round_001.jsonl", [player_state])
    _write_jsonl(
        canonical / "states/tactical/observed.jsonl.gz", [tactical_state], zipped=True
    )
    _write_json(
        canonical / "manifest.json",
        {
            "schema_id": "stratai.canonical_manifest@3",
            "parser_version": "v16",
            "artifacts": artifacts,
        },
    )
    _write_json(
        bundle / "manifest.json",
        {
            "match_id": match_id,
            "export_format_version": "3.8.0",
            "parser_schema_version": "v16",
            "validation_status": "passed",
            "validation_receipt": {"status": "passed"},
            "artifacts": [
                {
                    "artifact_type": "canonical_manifest",
                    "path": "canonical/manifest.json",
                    "schema_id": "stratai.canonical_manifest@3",
                    "format": "json",
                }
            ],
        },
    )
    return bundle


def _add_peek_decision(bundle: Path) -> None:
    manifest_path = bundle / "canonical/manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    causal_types = {
        "decisions",
        "decision_features",
        "oracle_context",
        "decision_outcomes",
        "quality_masks",
    }
    for descriptor in manifest["artifacts"]:
        if descriptor["artifact_type"] not in causal_types:
            continue
        artifact_path = bundle / "canonical" / descriptor["path"]
        rows = [
            json.loads(line)
            for line in artifact_path.read_text(encoding="utf-8").splitlines()
            if line
        ]
        peek = copy.deepcopy(rows[0])
        peek_id = f"{peek['match_id']}:peek:1"
        peek["decision_id"] = peek_id
        if descriptor["artifact_type"] == "decisions":
            peek["decision_type"] = "peek_hold_or_reposition"
            peek["action_taken"] = "held"
            peek["source"] = "peek_windows@1"
            peek["source_record_id"] = peek_id
        elif descriptor["artifact_type"] == "decision_features":
            peek["decision_type"] = "peek_hold_or_reposition"
        rows.append(peek)
        _write_jsonl(artifact_path, rows)
        descriptor["record_count"] = 2
    _write_json(manifest_path, manifest)


def _direct_bundle(root: Path, match_id: str = DIRECT_ALIAS) -> Path:
    bundle = _bundle(root, match_id)
    _add_peek_decision(bundle)
    (bundle / "manifest.json").unlink()
    publication_bundle = root / f"match_{match_id}"
    bundle.rename(publication_bundle)
    bundle = publication_bundle
    manifest_path = bundle / "canonical/manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest.update(
        {
            "export_format_version": "3.8.0",
            "match_id": match_id,
            "demo_checksum_sha256": DIRECT_CHECKSUM,
            "parser_version": "v16",
            "configuration_hashes": {},
            "transformation_versions": {},
            "lineage": {},
        }
    )
    for descriptor in manifest["artifacts"]:
        artifact_path = bundle / "canonical" / descriptor["path"]
        payload = artifact_path.read_bytes()
        descriptor["bytes"] = len(payload)
        descriptor["sha256"] = hashlib.sha256(payload).hexdigest()
    _write_json(manifest_path, manifest)
    return bundle


def _runtime_example() -> dict[str, object]:
    ref = {
        "artifact_type": "fixture",
        "schema_id": "fixture@1",
        "path": "fixture.jsonl",
        "line": 1,
    }
    block = {"availability_tick": 9, "status": "observed", "values": {"health": 100}}
    return {
        "schema_id": DECISION_EXAMPLE_SCHEMA_ID,
        "contract_version": CONTRACT_VERSION,
        "example_id": "decision-1",
        "decision_family": "spacing_or_trade_connection",
        "join_keys": {
            "match_id": "match-1",
            "decision_id": "decision-1",
            "round_number": 1,
            "actor_player_id": "actor-1",
            "observed_state_ref": "state-1",
        },
        "t0_tick": 10,
        "observed": {
            "action": copy.deepcopy(block),
            "referenced_state": copy.deepcopy(block),
            "tactical_state": copy.deepcopy(block),
            "context": copy.deepcopy(block),
            "quality": {},
        },
        "oracle": {
            "audit_only": True,
            "available": False,
            "status": "unavailable",
            "values": {},
        },
        "outcomes": {
            "label_or_evaluation_only": True,
            "outcome_tick": 11,
            "values": {"result": "later"},
        },
        "historical_profile": {
            "strictly_before_t0": True,
            "status": "not_materialized",
            "values": None,
        },
        "references": {"observed_action": ref, "outcome": ref, "quality": ref},
        "evidence": [ref, ref, ref, ref, ref],
        "limitations": [],
    }


def test_valid_join_builds_one_structural_example(tmp_path: Path) -> None:
    bundle = _bundle(tmp_path / "bundles")
    result = process_bundle(bundle, catalog_required_fields=[])
    assert result.joins_succeeded == 1
    assert result.family_counts == {"spacing_or_trade_connection": 1}
    assert result.gzip_rows_streamed == 1


def test_missing_reference_is_rejected(tmp_path: Path) -> None:
    bundle = _bundle(tmp_path / "bundles")
    path = bundle / "canonical/causal/decisions.jsonl"
    decision = json.loads(path.read_text(encoding="utf-8"))
    decision["observed_state_ref"] = "missing-state"
    _write_jsonl(path, [decision])
    with pytest.raises(DatasetContractError, match="JOIN_REFERENCE_MISSING"):
        process_bundle(bundle, catalog_required_fields=[])


def test_duplicate_reference_is_rejected(tmp_path: Path) -> None:
    bundle = _bundle(tmp_path / "bundles")
    path = bundle / "canonical/states/player_states/round_001.jsonl"
    state = json.loads(path.read_text(encoding="utf-8"))
    _write_jsonl(path, [state, state])
    with pytest.raises(DatasetContractError, match="DUPLICATE_IDENTIFIER"):
        process_bundle(bundle, catalog_required_fields=[])


def test_wrong_identifier_type_is_rejected(tmp_path: Path) -> None:
    bundle = _bundle(tmp_path / "bundles")
    path = bundle / "canonical/causal/decisions.jsonl"
    decision = json.loads(path.read_text(encoding="utf-8"))
    decision["actor_player_id"] = 123
    _write_jsonl(path, [decision])
    with pytest.raises(DatasetContractError, match="INVALID_IDENTIFIER_TYPE"):
        process_bundle(bundle, catalog_required_fields=[])


def test_availability_after_t0_is_rejected(tmp_path: Path) -> None:
    bundle = _bundle(tmp_path / "bundles")
    path = bundle / "canonical/causal/decision_features.jsonl"
    features = json.loads(path.read_text(encoding="utf-8"))
    features["availability_tick_max"] = 101
    _write_jsonl(path, [features])
    with pytest.raises(DatasetContractError, match="T0_AVAILABILITY_VIOLATION"):
        process_bundle(bundle, catalog_required_fields=[])


def test_future_mutation_does_not_change_observed_summary(tmp_path: Path) -> None:
    bundle = _bundle(tmp_path / "bundles")
    before = process_bundle(bundle, catalog_required_fields=[])
    outcome_path = bundle / "canonical/causal/outcomes.jsonl"
    outcome = json.loads(outcome_path.read_text(encoding="utf-8"))
    outcome["outcome"] = "failed"
    outcome["horizons"][0]["outcome"] = "failed"
    _write_jsonl(outcome_path, [outcome])
    after = process_bundle(bundle, catalog_required_fields=[])
    assert before.samples == after.samples
    assert before.field_coverage == after.field_coverage


def test_oracle_injected_into_observed_is_rejected() -> None:
    example = _runtime_example()
    example["observed"]["context"]["values"]["oracle_enemy_state"] = {"health": 1}
    with pytest.raises(RuntimeContractError, match="oracle/future field is forbidden"):
        validate_decision_example(example)


def test_corrupt_gzip_has_a_clear_error(tmp_path: Path) -> None:
    bundle = _bundle(tmp_path / "bundles")
    (bundle / "canonical/states/tactical/observed.jsonl.gz").write_bytes(b"not-gzip")
    with pytest.raises(DatasetContractError, match="GZIP_STREAM_CORRUPT"):
        process_bundle(bundle, catalog_required_fields=[])


def test_missing_required_contract_field_is_rejected() -> None:
    example = _runtime_example()
    del example["observed"]["context"]
    with pytest.raises(RuntimeContractError, match="missing required fields context"):
        validate_decision_example(example)


def test_unknown_reason_code_is_rejected() -> None:
    result = {
        "schema_id": TASK_ELIGIBILITY_SCHEMA_ID,
        "contract_version": CONTRACT_VERSION,
        "catalog_schema_id": "stratai.ai_coach_capability_catalog@1",
        "catalog_version": "1.0.0",
        "capability_id": "CAP-TEST-001",
        "status": "ineligible",
        "reason_codes": ["UNKNOWN-LOCAL-CODE"],
        "missing_required_fields": [],
        "materialized_decision_types": ["DEC-TEST-001"],
        "blockers": [],
        "evidence_used": [],
    }
    with pytest.raises(RuntimeContractError, match="unknown reason code"):
        validate_task_eligibility(result)


def test_all_capabilities_are_not_materialized_without_decisions() -> None:
    catalog = json.loads(CATALOG_PATH.read_text(encoding="utf-8"))
    results = evaluate_task_eligibility(
        catalog,
        family_counts={},
        field_coverage={},
        bundles_processed=0,
        temporal_violations=0,
        join_failures=0,
    )
    assert len(results) == 17
    assert {item["status"] for item in results} == {"not_materialized"}
    assert all(
        "DECISION_FAMILY_NOT_MATERIALIZED" in item["reason_codes"] for item in results
    )


def test_spatial_eligibility_uses_objective_fact_ids_not_legacy_human_labels() -> None:
    catalog = json.loads(CATALOG_PATH.read_text(encoding="utf-8"))
    results = evaluate_task_eligibility(
        catalog,
        family_counts={"peek_hold_or_reposition": 1},
        field_coverage={},
        bundles_processed=1,
        temporal_violations=0,
        join_failures=0,
    )
    peek = next(item for item in results if item["capability_id"] == "CAP-PEEK-001")
    assert peek["status"] == "ineligible"
    assert {
        "FLD-GEOMETRIC-CLEARANCE-SECTORS",
        "FLD-GEOMETRIC-OCCLUSION",
        "FLD-NAV-EXIT-PATH",
    }.issubset(peek["missing_required_fields"])
    assert {
        "FLD-EXPOSURE-ANGLES",
        "FLD-COVER",
        "FLD-EXIT-ROUTE",
    }.isdisjoint(peek["missing_required_fields"])


def test_output_is_deterministic(tmp_path: Path) -> None:
    root = tmp_path / "bundles"
    _bundle(root)
    first = build_audit_summary(
        root,
        expected_bundles=1,
        documented_source_demos=1,
        documented_exclusions=0,
    )
    second = build_audit_summary(
        root,
        expected_bundles=1,
        documented_source_demos=1,
        documented_exclusions=0,
    )
    assert first == second
    assert first["streaming"]["artifact_discovery"] == (
        "root manifest -> canonical_manifest descriptor -> canonical artifacts[]"
    )
    assert "discovery" not in first["bundles"]
    assert "provenance" not in first
    assert (
        "Los hashes ya validados por el manifest no se recalculan para evitar repetir Gate 1A."
        in first["limitations"]
    )


def test_direct_bundle_reuses_audit_and_reports_strict_discovery(
    tmp_path: Path,
) -> None:
    direct = _direct_bundle(tmp_path / "direct")
    corpus_root = tmp_path / "corpus"
    corpus_bundle = _bundle(corpus_root, DIRECT_ALIAS)
    _add_peek_decision(corpus_bundle)
    direct_report = build_direct_bundle_audit_summary(
        direct,
        expected_match_id=DIRECT_ALIAS,
        expected_demo_checksum=DIRECT_CHECKSUM,
    )
    corpus_report = build_audit_summary(
        corpus_root,
        expected_bundles=1,
        documented_source_demos=1,
        documented_exclusions=0,
    )

    for section in (
        "joins",
        "temporal_checks",
        "decision_examples",
        "fields",
        "samples",
    ):
        assert direct_report[section] == corpus_report[section]
    direct_eligibility = copy.deepcopy(direct_report["task_eligibility"])
    corpus_eligibility = copy.deepcopy(corpus_report["task_eligibility"])
    for result in direct_eligibility["results"]:
        assert result["evidence_used"][0]["reference"] == (
            "direct canonical identity + artifact bytes/SHA-256 verified"
        )
        result["evidence_used"][0]["reference"] = (
            "root manifest validation_status=passed"
        )
    assert direct_eligibility == corpus_eligibility
    assert direct_report["bundles"]["discovery"] == "direct_canonical_bundle"
    assert direct_report["streaming"]["artifact_discovery"] == (
        "explicit bundle root -> canonical/manifest.json -> canonical artifacts[]"
    )
    assert direct_report["provenance"] == {
        "source": "explicit_direct_canonical_bundle",
        "root_manifest_used": False,
        "canonical_manifest": "canonical/manifest.json",
        "expected_match_id": DIRECT_ALIAS,
        "expected_demo_checksum_sha256": DIRECT_CHECKSUM,
    }
    assert direct_report["exclusions"] == {
        "documented_source_demos": 1,
        "approved_bundles": 0,
        "documented_source_exclusions": 0,
        "excluded_bundles_opened": 0,
        "approved_source_demos": 1,
        "temporary_direct_bundles_validated": 1,
        "note": (
            "Se validó 1 demo fuente aprobada mediante 1 bundle temporal directo; "
            "se aprobaron 0 output bundles."
        ),
    }
    assert all("no se recalculan" not in item for item in direct_report["limitations"])
    assert str(direct) not in repr(direct_report)
    assert "root manifest validation_status=passed" not in repr(direct_report)


def test_direct_bundle_is_deterministic(tmp_path: Path) -> None:
    direct = _direct_bundle(tmp_path / "direct")
    kwargs = {
        "expected_match_id": DIRECT_ALIAS,
        "expected_demo_checksum": DIRECT_CHECKSUM,
    }
    first = build_direct_bundle_audit_summary(direct, **kwargs)
    second = build_direct_bundle_audit_summary(direct, **kwargs)
    assert first == second


def test_direct_bundle_rejects_root_manifest_identity_and_artifact_tampering(
    tmp_path: Path,
) -> None:
    ambiguous = _bundle(tmp_path / "ambiguous", DIRECT_ALIAS)
    with pytest.raises(DatasetContractError, match="AMBIGUOUS_BUNDLE_DISCOVERY"):
        build_direct_bundle_audit_summary(
            ambiguous,
            expected_match_id=DIRECT_ALIAS,
            expected_demo_checksum=DIRECT_CHECKSUM,
        )

    direct = _direct_bundle(tmp_path / "direct")
    with pytest.raises(DatasetContractError, match="20 lowercase hex"):
        build_direct_bundle_audit_summary(
            direct,
            expected_match_id="match-2",
            expected_demo_checksum=DIRECT_CHECKSUM,
        )
    with pytest.raises(DatasetContractError, match="approved demo checksum"):
        build_direct_bundle_audit_summary(
            direct,
            expected_match_id=DIRECT_ALIAS,
            expected_demo_checksum="b" * 64,
        )

    wrong_directory = _direct_bundle(tmp_path / "wrong-directory")
    renamed = wrong_directory.with_name("unexpected-bundle-name")
    wrong_directory.rename(renamed)
    with pytest.raises(DatasetContractError, match="match_<expected_match_id>"):
        build_direct_bundle_audit_summary(
            renamed,
            expected_match_id=DIRECT_ALIAS,
            expected_demo_checksum=DIRECT_CHECKSUM,
        )

    artifact_path = direct / "canonical/causal/decisions.jsonl"
    payload = bytearray(artifact_path.read_bytes())
    payload[0] ^= 1
    artifact_path.write_bytes(payload)
    with pytest.raises(DatasetContractError, match="SHA-256 mismatch"):
        build_direct_bundle_audit_summary(
            direct,
            expected_match_id=DIRECT_ALIAS,
            expected_demo_checksum=DIRECT_CHECKSUM,
        )


def test_direct_bundle_rejects_undeclared_canonical_files(tmp_path: Path) -> None:
    direct = _direct_bundle(tmp_path / "direct")
    (direct / "canonical/undeclared.bin").write_bytes(b"not declared")
    with pytest.raises(DatasetContractError, match="catalog mismatch"):
        build_direct_bundle_audit_summary(
            direct,
            expected_match_id=DIRECT_ALIAS,
            expected_demo_checksum=DIRECT_CHECKSUM,
        )


def test_direct_bundle_rejects_empty_causal_partitions(tmp_path: Path) -> None:
    direct = _direct_bundle(tmp_path / "direct")
    manifest_path = direct / "canonical/manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    causal_types = {
        "decisions",
        "decision_features",
        "oracle_context",
        "decision_outcomes",
        "quality_masks",
    }
    empty_checksum = hashlib.sha256(b"").hexdigest()
    for descriptor in manifest["artifacts"]:
        if descriptor["artifact_type"] not in causal_types:
            continue
        (direct / "canonical" / descriptor["path"]).write_bytes(b"")
        descriptor["record_count"] = 0
        descriptor["bytes"] = 0
        descriptor["sha256"] = empty_checksum
    _write_json(manifest_path, manifest)

    with pytest.raises(DatasetContractError, match="BUNDLE_VALIDATION_FAILED"):
        build_direct_bundle_audit_summary(
            direct,
            expected_match_id=DIRECT_ALIAS,
            expected_demo_checksum=DIRECT_CHECKSUM,
        )


def test_direct_bundle_rejects_noncanonical_descriptor_order(tmp_path: Path) -> None:
    direct = _direct_bundle(tmp_path / "direct")
    manifest_path = direct / "canonical/manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest["artifacts"].reverse()
    _write_json(manifest_path, manifest)

    with pytest.raises(DatasetContractError, match="ordered by path"):
        build_direct_bundle_audit_summary(
            direct,
            expected_match_id=DIRECT_ALIAS,
            expected_demo_checksum=DIRECT_CHECKSUM,
        )


def test_bundle_generator_does_not_read_ahead(tmp_path: Path) -> None:
    root = tmp_path / "bundles"
    first_bundle = _bundle(root, "match-1")
    second_bundle = _bundle(root, "match-2")
    (second_bundle / "canonical/states/tactical/observed.jsonl.gz").write_bytes(
        b"not-gzip"
    )
    stream = iter_bundle_summaries(
        [first_bundle, second_bundle], catalog_required_fields=[]
    )
    assert next(stream).joins_succeeded == 1
    with pytest.raises(DatasetContractError, match="GZIP_STREAM_CORRUPT"):
        next(stream)
