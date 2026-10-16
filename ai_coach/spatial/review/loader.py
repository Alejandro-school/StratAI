from __future__ import annotations

import hashlib
import json
import math
import stat
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker
from jsonschema.exceptions import SchemaError, ValidationError

PROTOCOL_VERSION = "1.0.0"
PROTOCOL_SCHEMA_ID = "stratai.spatial_human_review_protocol@1"
_PROTOCOL_FILE = "protocol.json"
_SCHEMA_PATH = (
    Path(__file__).resolve().parents[2]
    / "contracts"
    / "spatial_human_review_protocol.schema.json"
)


class SpatialReviewProtocolError(ValueError):
    """Raised when the versioned human-review protocol is not trustworthy."""


@dataclass(frozen=True)
class LoadedProtocol:
    protocol: dict[str, Any]
    protocol_sha256: str
    handbook_sha256: str
    protocol_path: Path
    handbook_path: Path


def load_protocol(
    review_root: Path | None = None,
    *,
    protocol_version: str = PROTOCOL_VERSION,
) -> LoadedProtocol:
    if protocol_version != PROTOCOL_VERSION:
        raise SpatialReviewProtocolError("protocol_version_incompatible")
    root = (review_root or Path(__file__).resolve().parent).resolve()
    protocol_path = _safe_child(
        root, f"protocols/{protocol_version}/{_PROTOCOL_FILE}"
    )
    try:
        payload = protocol_path.read_bytes()
        protocol = _decode_json(payload)
        schema = _decode_json(_SCHEMA_PATH.read_bytes())
        Draft202012Validator.check_schema(schema)
        Draft202012Validator(
            schema, format_checker=FormatChecker()
        ).validate(protocol)
        _require_finite(protocol, "protocol")
    except OSError as exc:
        raise SpatialReviewProtocolError("protocol_file_missing") from exc
    except (json.JSONDecodeError, SchemaError, ValidationError) as exc:
        raise SpatialReviewProtocolError("protocol_invalid") from exc

    if protocol["schema_id"] != PROTOCOL_SCHEMA_ID:
        raise SpatialReviewProtocolError("protocol_schema_mismatch")
    if protocol["protocol_version"] != protocol_version:
        raise SpatialReviewProtocolError("protocol_version_incompatible")

    handbook_reference = protocol["handbook"]["relative_path"]
    handbook_path = _safe_child(protocol_path.parent, handbook_reference)
    try:
        handbook_payload = handbook_path.read_bytes()
    except OSError as exc:
        raise SpatialReviewProtocolError("handbook_file_missing") from exc
    handbook_sha256 = hashlib.sha256(handbook_payload).hexdigest()
    if handbook_sha256 != protocol["handbook"]["sha256"]:
        raise SpatialReviewProtocolError("handbook_hash_mismatch")
    return LoadedProtocol(
        protocol=protocol,
        protocol_sha256=hashlib.sha256(payload).hexdigest(),
        handbook_sha256=handbook_sha256,
        protocol_path=protocol_path,
        handbook_path=handbook_path,
    )


def _decode_json(payload: bytes) -> dict[str, Any]:
    value = json.loads(
        payload.decode("utf-8"), object_pairs_hook=_pairs_without_duplicates
    )
    if not isinstance(value, dict):
        raise SpatialReviewProtocolError("expected JSON object")
    return value


def _pairs_without_duplicates(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    value: dict[str, Any] = {}
    for key, item in pairs:
        if key in value:
            raise SpatialReviewProtocolError(f"duplicate JSON key: {key}")
        value[key] = item
    return value


def _safe_child(root: Path, reference: str) -> Path:
    relative = PurePosixPath(reference)
    if relative.is_absolute() or not relative.parts or ".." in relative.parts:
        raise SpatialReviewProtocolError("unsafe_relative_path")
    candidate = root.joinpath(*relative.parts)
    resolved = candidate.resolve(strict=False)
    try:
        resolved.relative_to(root.resolve())
    except ValueError as exc:
        raise SpatialReviewProtocolError("unsafe_relative_path") from exc
    current = candidate
    while current != root.parent and current.exists():
        metadata = current.lstat()
        attributes = getattr(metadata, "st_file_attributes", 0)
        reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
        if stat.S_ISLNK(metadata.st_mode) or bool(attributes & reparse_flag):
            raise SpatialReviewProtocolError("unsafe_relative_path")
        if current == root:
            break
        current = current.parent
    return resolved


def _require_finite(value: Any, path: str) -> None:
    if isinstance(value, bool) or value is None or isinstance(value, (str, int)):
        return
    if isinstance(value, float):
        if not math.isfinite(value):
            raise SpatialReviewProtocolError(f"{path}: non-finite number")
        return
    if isinstance(value, list):
        for index, item in enumerate(value):
            _require_finite(item, f"{path}[{index}]")
        return
    if isinstance(value, dict):
        for key, item in value.items():
            _require_finite(item, f"{path}.{key}")
        return
    raise SpatialReviewProtocolError(f"{path}: unsupported value")
