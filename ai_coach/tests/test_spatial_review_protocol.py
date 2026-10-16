from __future__ import annotations

import hashlib
import json
import shutil
from pathlib import Path

import pytest
from jsonschema import Draft202012Validator

from ai_coach.contracts.runtime_contracts import validate_runtime_contract_files
from ai_coach.spatial.review.ground_truth import (
    DEPRECATED_SPATIAL_REVIEW_SCHEMA_IDS,
    TacticalGroundTruthSourceError,
    assert_tactical_ground_truth_source,
    validate_tactical_ground_truth_records,
)
from ai_coach.spatial.review.loader import (
    SpatialReviewProtocolError,
    load_protocol,
)

ROOT = Path(__file__).resolve().parents[2]
REVIEW_ROOT = ROOT / "ai_coach" / "spatial" / "review"
CONTRACTS = ROOT / "ai_coach" / "contracts"


def test_protocol_and_all_human_record_schemas_are_valid_draft_2020_12() -> None:
    names = (
        "spatial_human_review_protocol.schema.json",
        "spatial_human_review_batch.schema.json",
        "spatial_human_review_response.schema.json",
        "spatial_human_review_adjudication.schema.json",
        "spatial_context_review_protocol.schema.json",
        "spatial_context_review_batch.schema.json",
        "spatial_context_review_response.schema.json",
        "spatial_context_review_adjudication.schema.json",
    )
    identifiers: list[str] = []
    for name in names:
        schema = json.loads((CONTRACTS / name).read_text(encoding="utf-8"))
        assert schema["$schema"] == "https://json-schema.org/draft/2020-12/schema"
        assert schema["x-stratai-status"] == "deprecated_visual_qa_only"
        assert schema["x-stratai-ground-truth-authorized"] is False
        assert schema["x-stratai-evaluation-authorized"] is False
        assert schema["x-stratai-training-authorized"] is False
        Draft202012Validator.check_schema(schema)
        identifiers.append(schema["$id"])
    assert len(identifiers) == len(set(identifiers)) == 8
    assert set(identifiers) == set(DEPRECATED_SPATIAL_REVIEW_SCHEMA_IDS)
    # CS2 added the common factual context contract to the existing registry.
    assert validate_runtime_contract_files()["schemas"] == 116


@pytest.mark.parametrize("schema_id", sorted(DEPRECATED_SPATIAL_REVIEW_SCHEMA_IDS))
def test_legacy_spatial_review_records_are_rejected_as_ground_truth(
    schema_id: str,
) -> None:
    with pytest.raises(
        TacticalGroundTruthSourceError, match="deprecated_visual_qa_only"
    ):
        assert_tactical_ground_truth_source({"schema_id": schema_id})


def test_ground_truth_ingestion_is_empty_allowlist_and_fails_closed() -> None:
    with pytest.raises(TacticalGroundTruthSourceError, match="empty_source"):
        validate_tactical_ground_truth_records([])
    with pytest.raises(TacticalGroundTruthSourceError, match="schema_not_allowlisted"):
        assert_tactical_ground_truth_source({"schema_id": "stratai.unknown_label@1"})


def test_protocol_freezes_independence_states_blinding_and_authorization() -> None:
    loaded = load_protocol()
    protocol = loaded.protocol
    assert protocol["roles"] == {
        "independent_reviewer_count": 2,
        "reviewer_slots": ["reviewer_a", "reviewer_b"],
        "adjudicator_required_on_conflict": True,
        "adjudicator_must_be_distinct": True,
        "identity_policy": "opaque_pseudonym_no_public_identity_in_artifact",
    }
    assert protocol["review_unit"]["allowed_states"] == [
        "pending_review",
        "reviewed",
        "needs_adjudication",
        "adjudicated",
        "rejected",
        "unavailable",
    ]
    assert protocol["blinding"]["prior_response_visibility"] == "none"
    assert protocol["workflow"]["support_auto_approval_allowed"] is False
    assert set(protocol["scope"]["families"]) == {
        "exposure",
        "cover",
        "exit_route",
    }
    assert set(protocol["authorization"].values()) == {False}
    assert hashlib.sha256(loaded.handbook_path.read_bytes()).hexdigest() == (
        loaded.handbook_sha256
    )


def test_protocol_loader_rejects_handbook_tampering(tmp_path: Path) -> None:
    copied = tmp_path / "review"
    shutil.copytree(REVIEW_ROOT, copied)
    handbook = copied / "protocols" / "1.0.0" / "HANDBOOK.md"
    handbook.write_text(
        handbook.read_text(encoding="utf-8") + "\ntampered\n", encoding="utf-8"
    )
    with pytest.raises(SpatialReviewProtocolError, match="handbook_hash_mismatch"):
        load_protocol(copied)


def test_protocol_loader_rejects_schema_and_version_mismatch(tmp_path: Path) -> None:
    copied = tmp_path / "review"
    shutil.copytree(REVIEW_ROOT, copied)
    protocol_path = copied / "protocols" / "1.0.0" / "protocol.json"
    protocol = json.loads(protocol_path.read_text(encoding="utf-8"))
    protocol["roles"]["independent_reviewer_count"] = 1
    protocol_path.write_text(
        json.dumps(protocol, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    with pytest.raises(SpatialReviewProtocolError, match="protocol_invalid"):
        load_protocol(copied)
    with pytest.raises(SpatialReviewProtocolError, match="version_incompatible"):
        load_protocol(protocol_version="2.0.0")
