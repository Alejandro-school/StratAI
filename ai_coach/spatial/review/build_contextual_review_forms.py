from __future__ import annotations

import argparse
import base64
import hashlib
import json
import secrets
import stat
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from ai_coach.contracts.validate_spatial_context_review_pilot import (
    validate_spatial_context_review_pilot,
)

ROOT = Path(__file__).resolve().parents[3]
RADAR_PATH = ROOT / "frontend" / "public" / "maps" / "de_mirage_radar_psd.webp"
TEMPLATE_PATH = Path(__file__).resolve().parent / "templates" / "contextual_review_form.html"
REVIEWER_SLOTS = ("reviewer_a", "reviewer_b")
FORM_NAME = "review.html"


class ContextualReviewFormError(ValueError):
    """Raised before a broken or leaking contextual form is delivered."""


def build_contextual_review_forms(
    batch_path: Path,
    output_root: Path,
    *,
    radar_path: Path = RADAR_PATH,
    template_path: Path = TEMPLATE_PATH,
    created_at: datetime | None = None,
) -> dict[str, Any]:
    destination = _empty_temp_destination(output_root)
    validation = validate_spatial_context_review_pilot(
        batch_path, radar_path=radar_path
    )
    batch_payload = batch_path.resolve(strict=True).read_bytes()
    batch = _object(batch_payload, "batch")
    batch_sha256 = hashlib.sha256(batch_payload).hexdigest()
    if validation["review_units"] != 12 or validation["responses"] != 0:
        raise ContextualReviewFormError("batch_not_ready")

    radar = radar_path.resolve(strict=True).read_bytes()
    if hashlib.sha256(radar).hexdigest() != batch["presentation"]["radar_asset"]["sha256"]:
        raise ContextualReviewFormError("radar_hash_mismatch")
    radar_data_uri = "data:image/webp;base64," + base64.b64encode(radar).decode("ascii")

    template = template_path.resolve(strict=True).read_text(encoding="utf-8")
    placeholder = "__STRATAI_CONTEXT_REVIEW_PACKET__"
    if template.count(placeholder) != 1:
        raise ContextualReviewFormError("template_placeholder_invalid")
    _offline_template(template)

    session_id = "spatial-context-session-" + secrets.token_hex(12)
    timestamp = _timestamp(created_at)
    pseudonyms = {slot: "reviewer-" + secrets.token_hex(8) for slot in REVIEWER_SLOTS}
    visible_units = [
        {
            "review_unit_id": unit["review_unit_id"],
            "display_order": unit["display_order"],
            "reviewer_view": unit["reviewer_view"],
            "visible_evidence_sha256": unit["visible_evidence_sha256"],
        }
        for unit in batch["review_units"]
    ]
    rendered: dict[str, bytes] = {}
    for slot in REVIEWER_SLOTS:
        packet = {
            "packet_version": "2.0.0",
            "session_id": session_id,
            "batch_id": batch["batch_id"],
            "protocol_version": batch["protocol"]["protocol_version"],
            "handbook_version": batch["protocol"]["handbook_version"],
            "response_schema_id": "stratai.spatial_context_review_response@2",
            "reviewer_slot": slot,
            "reviewer_pseudonym": pseudonyms[slot],
            "map_label": batch["presentation"]["map_label"],
            "coordinate_transform": batch["presentation"]["coordinate_transform"],
            "render_contract": batch["presentation"]["render_contract"],
            "radar_data_uri": radar_data_uri,
            "review_units": visible_units,
        }
        html = template.replace(placeholder, _inline_json(packet)).encode("utf-8")
        _no_coordinator_leak(html)
        rendered[slot] = html

    deliveries = []
    for slot in REVIEWER_SLOTS:
        directory = destination / slot
        directory.mkdir()
        form_path = directory / FORM_NAME
        _write_new(form_path, rendered[slot])
        deliveries.append(
            {
                "reviewer_slot": slot,
                "reviewer_pseudonym": pseudonyms[slot],
                "relative_path": f"{slot}/{FORM_NAME}",
                "html_sha256": hashlib.sha256(rendered[slot]).hexdigest(),
                "review_units": 12,
                "delivery_policy": "deliver_only_this_html_to_assigned_reviewer",
            }
        )

    coordinator = destination / "coordinator"
    coordinator.mkdir()
    manifest = {
        "schema_id": "stratai.spatial_context_review_session_manifest@2",
        "contract_version": "2.0.0",
        "session_id": session_id,
        "created_at": timestamp,
        "batch_id": batch["batch_id"],
        "batch_sha256": batch_sha256,
        "protocol_version": "2.0.0",
        "handbook_version": "2.0.0",
        "render_contract": batch["presentation"]["render_contract"],
        "review_units": 12,
        "deliveries": deliveries,
        "adjudication": {
            "status": "reserved_for_conflicts",
            "form_generation": "after_two_valid_human_response_files",
        },
        "privacy": {
            "public_identities_recorded": False,
            "absolute_paths_recorded": False,
            "reviewer_forms_contain_audit_context": False,
            "reviewer_forms_contain_other_responses": False,
        },
        "authorization": batch["authorization"],
    }
    manifest_path = coordinator / "session_manifest.json"
    _write_new(manifest_path, _canonical(manifest))
    return {
        "session_id": session_id,
        "batch_id": batch["batch_id"],
        "batch_sha256": batch_sha256,
        "review_units": 12,
        "manifest": manifest_path,
        "reviewer_forms": {
            slot: destination / slot / FORM_NAME for slot in REVIEWER_SLOTS
        },
    }


def _empty_temp_destination(path: Path) -> Path:
    temp = Path(tempfile.gettempdir()).resolve(strict=True)
    destination = path.resolve(strict=False)
    try:
        destination.relative_to(temp)
    except ValueError as exc:
        raise ContextualReviewFormError("output_must_be_under_system_temp") from exc
    if destination == temp or not destination.is_dir():
        raise ContextualReviewFormError("dedicated_existing_output_required")
    current = destination
    while True:
        metadata = current.lstat()
        attributes = getattr(metadata, "st_file_attributes", 0)
        reparse = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
        if stat.S_ISLNK(metadata.st_mode) or bool(attributes & reparse):
            raise ContextualReviewFormError("output_contains_link_or_reparse_point")
        if current == temp:
            break
        current = current.parent
    if any(destination.iterdir()):
        raise ContextualReviewFormError("output_directory_must_be_empty")
    return destination


def _offline_template(template: str) -> None:
    inspected = template.replace("http://www.w3.org/2000/svg", "")
    forbidden = ("http://", "https://", "fetch(", "XMLHttpRequest", "WebSocket", "localStorage", "sessionStorage")
    if any(token in inspected for token in forbidden):
        raise ContextualReviewFormError("template_not_offline_safe")


def _no_coordinator_leak(rendered: bytes) -> None:
    forbidden = (
        b'"audit_context"',
        b'"source_demo"',
        b'"source_packages"',
        b'"package_sha256"',
        b'"sidecar_row_sha256"',
        b'"source_join"',
        b'"review_responses"',
        b'"adjudication"',
    )
    leaked = [token.decode("ascii") for token in forbidden if token in rendered]
    if leaked:
        raise ContextualReviewFormError("coordinator_data_leak:" + ",".join(leaked))


def _inline_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"), allow_nan=False).replace("<", "\\u003c").replace("&", "\\u0026").replace("\u2028", "\\u2028").replace("\u2029", "\\u2029")


def _canonical(value: Any) -> bytes:
    return (json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"), allow_nan=False) + "\n").encode("utf-8")


def _object(payload: bytes, label: str) -> dict[str, Any]:
    try:
        value = json.loads(payload.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ContextualReviewFormError(f"{label}_invalid_json") from exc
    if not isinstance(value, dict):
        raise ContextualReviewFormError(f"{label}_must_be_object")
    return value


def _timestamp(value: datetime | None) -> str:
    instant = value or datetime.now(timezone.utc)
    if instant.tzinfo is None or instant.utcoffset() is None:
        raise ContextualReviewFormError("created_at_must_be_timezone_aware")
    return instant.astimezone(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")


def _write_new(path: Path, payload: bytes) -> None:
    with path.open("xb") as handle:
        handle.write(payload)
        handle.flush()


def main() -> int:
    parser = argparse.ArgumentParser(description="Build two situated spatial-review forms")
    parser.add_argument("--batch", type=Path, required=True)
    parser.add_argument("--output-root", type=Path, required=True)
    args = parser.parse_args()
    result = build_contextual_review_forms(args.batch, args.output_root)
    print(json.dumps({key: ({slot: str(path) for slot, path in value.items()} if key == "reviewer_forms" else str(value) if isinstance(value, Path) else value) for key, value in result.items()}, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
