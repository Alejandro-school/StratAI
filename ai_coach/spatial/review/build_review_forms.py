from __future__ import annotations

import argparse
import hashlib
import json
import secrets
import stat
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from ai_coach.contracts.validate_spatial_review_pilot import (
    validate_spatial_review_pilot,
)

EXPECTED_BATCH_SHA256 = (
    "19648364aee7716221eba7ebded9c0c44a71ad6c513e9a6b9a5f7f9a21bf1b06"
)
SESSION_MANIFEST_NAME = "session_manifest.json"
REVIEW_FORM_NAME = "review.html"
REVIEWER_SLOTS = ("reviewer_a", "reviewer_b")

_TEMPLATE_PATH = Path(__file__).resolve().parent / "templates" / "review_form.html"


class SpatialReviewFormError(ValueError):
    """Raised before an unsafe or leaking reviewer form is published."""


def build_review_forms(
    batch_path: Path,
    output_root: Path,
    *,
    template_path: Path | None = None,
    created_at: datetime | None = None,
) -> dict[str, Any]:
    """Build two self-contained, blinded forms from a validated pilot batch."""

    destination = _validate_empty_temporary_destination(output_root)
    validation = validate_spatial_review_pilot(batch_path)
    batch_payload = batch_path.resolve(strict=True).read_bytes()
    batch_sha256 = hashlib.sha256(batch_payload).hexdigest()
    if batch_sha256 != EXPECTED_BATCH_SHA256:
        raise SpatialReviewFormError("frozen_batch_hash_mismatch")
    if validation.get("review_units") != 12 or validation.get("responses") != 0:
        raise SpatialReviewFormError("batch_not_ready_for_blinded_delivery")

    batch = _decode_object(batch_payload, "batch")
    if any(unit["state"] != "pending_review" for unit in batch["review_units"]):
        raise SpatialReviewFormError("batch_contains_non_pending_unit")

    source_template = template_path or _TEMPLATE_PATH
    template = source_template.read_text(encoding="utf-8")
    placeholder = "__STRATAI_REVIEW_PACKET__"
    if template.count(placeholder) != 1:
        raise SpatialReviewFormError("review_form_template_placeholder_invalid")
    _assert_offline_template(template)

    timestamp = _utc_timestamp(created_at)
    session_id = f"spatial-review-session-{secrets.token_hex(12)}"
    reviewer_pseudonyms = {
        slot: f"reviewer-{secrets.token_hex(8)}" for slot in REVIEWER_SLOTS
    }
    adjudicator_pseudonym = f"adjudicator-{secrets.token_hex(8)}"
    visible_units = [
        {
            "review_unit_id": unit["review_unit_id"],
            "display_order": unit["display_order"],
            "reviewer_view": unit["reviewer_view"],
            "visible_evidence_sha256": unit["visible_evidence_sha256"],
        }
        for unit in batch["review_units"]
    ]

    rendered_forms: dict[str, bytes] = {}
    for slot in REVIEWER_SLOTS:
        packet = {
            "packet_version": "1.0.0",
            "session_id": session_id,
            "batch_id": batch["batch_id"],
            "protocol_version": batch["protocol"]["protocol_version"],
            "handbook_version": batch["protocol"]["handbook_version"],
            "response_schema_id": "stratai.spatial_human_review_response@1",
            "reviewer_slot": slot,
            "reviewer_pseudonym": reviewer_pseudonyms[slot],
            "review_units": visible_units,
        }
        embedded_packet = _json_for_inline_script(packet)
        rendered = template.replace(placeholder, embedded_packet).encode("utf-8")
        _assert_no_coordinator_leak(rendered)
        rendered_forms[slot] = rendered

    deliveries = []
    for slot in REVIEWER_SLOTS:
        relative_path = f"{slot}/{REVIEW_FORM_NAME}"
        deliveries.append(
            {
                "reviewer_slot": slot,
                "reviewer_pseudonym": reviewer_pseudonyms[slot],
                "relative_path": relative_path,
                "html_sha256": hashlib.sha256(rendered_forms[slot]).hexdigest(),
                "review_units": len(visible_units),
                "delivery_policy": "deliver_only_this_html_to_assigned_reviewer",
            }
        )

    manifest = {
        "schema_id": "stratai.spatial_human_review_session_manifest@1",
        "contract_version": "1.0.0",
        "session_id": session_id,
        "created_at": timestamp,
        "batch_id": batch["batch_id"],
        "batch_sha256": batch_sha256,
        "protocol_version": batch["protocol"]["protocol_version"],
        "handbook_version": batch["protocol"]["handbook_version"],
        "review_units": len(visible_units),
        "deliveries": deliveries,
        "adjudication": {
            "adjudicator_pseudonym": adjudicator_pseudonym,
            "status": "reserved_for_conflicts",
            "form_generation": "after_two_valid_human_response_files",
        },
        "privacy": {
            "public_identities_recorded": False,
            "absolute_paths_recorded": False,
            "reviewer_forms_contain_audit_context": False,
            "reviewer_forms_contain_other_responses": False,
        },
        "authorization": {
            "training_use_authorized": False,
            "product_supported": False,
            "canonical_materialization": False,
            "release_authorized": False,
            "corpus_reprocessing_authorized": False,
        },
    }

    for slot in REVIEWER_SLOTS:
        review_directory = destination / slot
        review_directory.mkdir()
        _write_new(review_directory / REVIEW_FORM_NAME, rendered_forms[slot])
    coordinator_directory = destination / "coordinator"
    coordinator_directory.mkdir()
    manifest_path = coordinator_directory / SESSION_MANIFEST_NAME
    _write_new(manifest_path, _canonical_json_bytes(manifest))

    return {
        "session_id": session_id,
        "batch_id": batch["batch_id"],
        "batch_sha256": batch_sha256,
        "review_units": len(visible_units),
        "manifest": manifest_path,
        "reviewer_forms": {
            slot: destination / slot / REVIEW_FORM_NAME for slot in REVIEWER_SLOTS
        },
    }


def _validate_empty_temporary_destination(output_root: Path) -> Path:
    temporary_root = Path(tempfile.gettempdir()).resolve()
    destination = output_root.resolve(strict=False)
    try:
        destination.relative_to(temporary_root)
    except ValueError as exc:
        raise SpatialReviewFormError("output_must_be_under_system_temp") from exc
    if destination == temporary_root:
        raise SpatialReviewFormError("output_must_be_dedicated_directory")
    if not destination.exists() or not destination.is_dir():
        raise SpatialReviewFormError("output_directory_must_exist")
    current = destination
    while True:
        if _is_symlink_or_reparse(current):
            raise SpatialReviewFormError("output_contains_link_or_reparse_point")
        if current == temporary_root:
            break
        if current.parent == current:
            raise SpatialReviewFormError("output_must_be_under_system_temp")
        current = current.parent
    if any(destination.iterdir()):
        raise SpatialReviewFormError("output_directory_must_be_empty")
    return destination


def _is_symlink_or_reparse(path: Path) -> bool:
    metadata = path.lstat()
    attributes = getattr(metadata, "st_file_attributes", 0)
    reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    return stat.S_ISLNK(metadata.st_mode) or bool(attributes & reparse_flag)


def _assert_offline_template(template: str) -> None:
    inspected = template.replace("http://www.w3.org/2000/svg", "")
    forbidden = (
        "http://",
        "https://",
        "fetch(",
        "XMLHttpRequest",
        "WebSocket",
        "localStorage",
        "sessionStorage",
    )
    if any(token in inspected for token in forbidden):
        raise SpatialReviewFormError("review_form_template_not_offline_safe")


def _assert_no_coordinator_leak(rendered: bytes) -> None:
    forbidden = (
        b'"audit_context"',
        b'"package_sha256"',
        b'"support_manifest_sha256"',
        b'"source_sample_ref"',
        b'"source_candidate_ref"',
        b'"source_annotation_ref"',
        b'"source_group_id"',
        b'"family"',
        b'"map_id"',
        b'"review_responses"',
        b'"adjudication"',
    )
    leaked = [token.decode("ascii") for token in forbidden if token in rendered]
    if leaked:
        raise SpatialReviewFormError(
            f"review_form_contains_coordinator_data:{','.join(leaked)}"
        )


def _json_for_inline_script(value: Any) -> str:
    return (
        json.dumps(
            value,
            ensure_ascii=False,
            sort_keys=True,
            separators=(",", ":"),
            allow_nan=False,
        )
        .replace("<", "\\u003c")
        .replace("&", "\\u0026")
        .replace("\u2028", "\\u2028")
        .replace("\u2029", "\\u2029")
    )


def _canonical_json_bytes(value: Any) -> bytes:
    return (
        json.dumps(
            value,
            ensure_ascii=False,
            sort_keys=True,
            separators=(",", ":"),
            allow_nan=False,
        )
        + "\n"
    ).encode("utf-8")


def _decode_object(payload: bytes, location: str) -> dict[str, Any]:
    try:
        value = json.loads(payload.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise SpatialReviewFormError(f"{location}_invalid_json") from exc
    if not isinstance(value, dict):
        raise SpatialReviewFormError(f"{location}_must_be_object")
    return value


def _utc_timestamp(value: datetime | None) -> str:
    instant = value or datetime.now(timezone.utc)
    if instant.tzinfo is None or instant.utcoffset() is None:
        raise SpatialReviewFormError("created_at_must_be_timezone_aware")
    return instant.astimezone(timezone.utc).isoformat(timespec="seconds").replace(
        "+00:00", "Z"
    )


def _write_new(path: Path, payload: bytes) -> None:
    with path.open("xb") as handle:
        handle.write(payload)
        handle.flush()


def _argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Build two blinded, offline spatial human-review forms"
    )
    parser.add_argument("--batch", type=Path, required=True)
    parser.add_argument("--output-root", type=Path, required=True)
    return parser


def main() -> int:
    args = _argument_parser().parse_args()
    result = build_review_forms(args.batch, args.output_root)
    print(
        json.dumps(
            {
                "session_id": result["session_id"],
                "batch_id": result["batch_id"],
                "batch_sha256": result["batch_sha256"],
                "review_units": result["review_units"],
                "manifest": str(result["manifest"]),
                "reviewer_forms": {
                    slot: str(path)
                    for slot, path in result["reviewer_forms"].items()
                },
            },
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
