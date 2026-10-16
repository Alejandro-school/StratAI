from __future__ import annotations

import hashlib
import json
import re
from datetime import datetime, timezone
from pathlib import Path

import pytest

from ai_coach.spatial.review.build_pilot import build_pilot_batch
from ai_coach.spatial.review.build_review_forms import (
    EXPECTED_BATCH_SHA256,
    SpatialReviewFormError,
    build_review_forms,
)

ROOT = Path(__file__).resolve().parents[2]
PACKET_PATTERN = re.compile(
    r'<script type="application/json" id="review-packet">(.*?)</script>', re.DOTALL
)


def _empty_run(tmp_path: Path, name: str) -> Path:
    path = tmp_path / name
    path.mkdir()
    return path


def _packet(html: str) -> dict:
    match = PACKET_PATTERN.search(html)
    assert match is not None
    return json.loads(match.group(1))


def test_builds_two_distinct_self_contained_blinded_forms(tmp_path: Path) -> None:
    batch_path, batch = build_pilot_batch(_empty_run(tmp_path, "batch"))
    destination = _empty_run(tmp_path, "forms")
    fixed_time = datetime(2026, 9, 2, 9, 15, tzinfo=timezone.utc)

    result = build_review_forms(
        batch_path, destination, created_at=fixed_time
    )

    assert result["batch_sha256"] == EXPECTED_BATCH_SHA256
    assert result["review_units"] == 12
    assert result["manifest"] == destination / "coordinator" / "session_manifest.json"
    manifest = json.loads(result["manifest"].read_text(encoding="utf-8"))
    assert manifest["created_at"] == "2026-09-02T09:15:00Z"
    assert manifest["privacy"] == {
        "public_identities_recorded": False,
        "absolute_paths_recorded": False,
        "reviewer_forms_contain_audit_context": False,
        "reviewer_forms_contain_other_responses": False,
    }
    assert set(manifest["authorization"].values()) == {False}
    assert str(tmp_path) not in result["manifest"].read_text(encoding="utf-8")

    packets = {}
    html_by_slot = {}
    for delivery in manifest["deliveries"]:
        slot = delivery["reviewer_slot"]
        html_path = destination / delivery["relative_path"]
        html = html_path.read_text(encoding="utf-8")
        html_by_slot[slot] = html
        packets[slot] = _packet(html)
        assert hashlib.sha256(html_path.read_bytes()).hexdigest() == delivery[
            "html_sha256"
        ]
        assert "Content-Security-Policy" in html
        assert "fetch(" not in html
        assert "XMLHttpRequest" not in html
        assert "WebSocket" not in html
        assert '"audit_context"' not in html
        assert '"package_sha256"' not in html
        assert '"support_manifest_sha256"' not in html
        assert '"source_sample_ref"' not in html
        assert '"source_candidate_ref"' not in html
        assert '"family"' not in html
        assert '"map_id"' not in html
        assert '"review_responses"' not in html

    assert html_by_slot["reviewer_a"] != html_by_slot["reviewer_b"]
    assert packets["reviewer_a"]["reviewer_slot"] == "reviewer_a"
    assert packets["reviewer_b"]["reviewer_slot"] == "reviewer_b"
    assert packets["reviewer_a"]["reviewer_pseudonym"] != packets["reviewer_b"][
        "reviewer_pseudonym"
    ]
    for packet in packets.values():
        assert re.fullmatch(r"reviewer-[0-9a-f]{16}", packet["reviewer_pseudonym"])
        assert packet["batch_id"] == batch["batch_id"]
        assert packet["protocol_version"] == "1.0.0"
        assert packet["handbook_version"] == "1.0.0"
        assert len(packet["review_units"]) == 12
        assert packet["review_units"] == [
            {
                "review_unit_id": unit["review_unit_id"],
                "display_order": unit["display_order"],
                "reviewer_view": unit["reviewer_view"],
                "visible_evidence_sha256": unit["visible_evidence_sha256"],
            }
            for unit in batch["review_units"]
        ]


def test_rejects_non_temporary_non_empty_and_non_offline_destinations(
    tmp_path: Path,
) -> None:
    batch_path, _ = build_pilot_batch(_empty_run(tmp_path, "batch"))

    with pytest.raises(SpatialReviewFormError, match="under_system_temp"):
        build_review_forms(batch_path, ROOT / "ai_coach")

    non_empty = _empty_run(tmp_path, "non-empty")
    keep = non_empty / "keep.txt"
    keep.write_text("keep", encoding="utf-8")
    with pytest.raises(SpatialReviewFormError, match="must_be_empty"):
        build_review_forms(batch_path, non_empty)
    assert keep.read_text(encoding="utf-8") == "keep"

    unsafe_template = tmp_path / "unsafe.html"
    unsafe_template.write_text(
        "<script>fetch('elsewhere')</script>__STRATAI_REVIEW_PACKET__",
        encoding="utf-8",
    )
    unsafe_output = _empty_run(tmp_path, "unsafe-output")
    with pytest.raises(SpatialReviewFormError, match="not_offline_safe"):
        build_review_forms(
            batch_path, unsafe_output, template_path=unsafe_template
        )
    assert list(unsafe_output.iterdir()) == []


def test_rejects_batch_with_wrong_frozen_hash_before_writing(tmp_path: Path) -> None:
    batch_path, _ = build_pilot_batch(_empty_run(tmp_path, "batch"))
    original = batch_path.read_text(encoding="utf-8")
    batch_path.write_text(original + " ", encoding="utf-8")
    destination = _empty_run(tmp_path, "forms")

    with pytest.raises(SpatialReviewFormError, match="frozen_batch_hash_mismatch"):
        build_review_forms(batch_path, destination)
    assert list(destination.iterdir()) == []
