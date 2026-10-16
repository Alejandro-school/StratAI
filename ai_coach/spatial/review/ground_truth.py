"""Fail-closed boundary for future tactical ground-truth ingestion."""

from __future__ import annotations

from collections.abc import Iterable, Mapping
from typing import Any

DEPRECATED_SPATIAL_REVIEW_SCHEMA_IDS = frozenset(
    {
        "stratai.spatial_human_review_protocol@1",
        "stratai.spatial_human_review_batch@1",
        "stratai.spatial_human_review_response@1",
        "stratai.spatial_human_review_adjudication@1",
        "stratai.spatial_context_review_protocol@2",
        "stratai.spatial_context_review_batch@2",
        "stratai.spatial_context_review_response@2",
        "stratai.spatial_context_review_adjudication@2",
    }
)

# The successor tactical decision contract has not been designed yet. Keeping
# this allowlist empty prevents an unknown or legacy record from becoming a
# training/evaluation label by convention or filename.
AUTHORIZED_TACTICAL_GROUND_TRUTH_SCHEMA_IDS: frozenset[str] = frozenset()
REJECTION_REASON_CODE = "TACTICAL_GROUND_TRUTH_SOURCE_FORBIDDEN"


class TacticalGroundTruthSourceError(ValueError):
    """Raised when a record is not an explicitly authorized tactical label."""


def assert_tactical_ground_truth_source(record: Mapping[str, Any]) -> None:
    """Accept only explicitly allowlisted future tactical-label schemas."""

    schema_id = record.get("schema_id")
    if not isinstance(schema_id, str) or not schema_id:
        raise TacticalGroundTruthSourceError(
            f"{REJECTION_REASON_CODE}:missing_schema_id"
        )
    if schema_id in DEPRECATED_SPATIAL_REVIEW_SCHEMA_IDS:
        raise TacticalGroundTruthSourceError(
            f"{REJECTION_REASON_CODE}:deprecated_visual_qa_only:{schema_id}"
        )
    if schema_id not in AUTHORIZED_TACTICAL_GROUND_TRUTH_SCHEMA_IDS:
        raise TacticalGroundTruthSourceError(
            f"{REJECTION_REASON_CODE}:schema_not_allowlisted:{schema_id}"
        )


def validate_tactical_ground_truth_records(
    records: Iterable[Mapping[str, Any]],
) -> int:
    """Validate a complete label stream before any downstream ingestion."""

    count = 0
    for record in records:
        assert_tactical_ground_truth_source(record)
        count += 1
    if count == 0:
        raise TacticalGroundTruthSourceError(
            f"{REJECTION_REASON_CODE}:empty_source"
        )
    return count
