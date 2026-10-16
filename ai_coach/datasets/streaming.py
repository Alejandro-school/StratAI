"""Manifest-only artifact discovery and bounded JSONL/gzip streaming."""

from __future__ import annotations

import gzip
import hashlib
import json
import re
import stat
from collections import defaultdict
from collections.abc import Iterator, Mapping
from dataclasses import dataclass
from pathlib import Path
from typing import Any, BinaryIO

MAX_MANIFEST_BYTES = 16 * 1024 * 1024
MAX_JSONL_LINE_BYTES = 16 * 1024 * 1024
EXPECTED_EXPORT_VERSION = "3.8.0"
EXPECTED_PARSER_VERSION = "v16"
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
SOURCE_ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
DIRECT_CANONICAL_KEYS = {
    "schema_id",
    "export_format_version",
    "match_id",
    "demo_checksum_sha256",
    "parser_version",
    "configuration_hashes",
    "transformation_versions",
    "lineage",
    "artifacts",
}
DIRECT_ARTIFACT_REQUIRED_KEYS = {
    "artifact_type",
    "path",
    "schema_id",
    "format",
    "record_count",
    "sha256",
    "bytes",
    "sort_order",
}
DIRECT_ARTIFACT_OPTIONAL_KEYS = {"compression"}


class DatasetContractError(RuntimeError):
    """Data error with a stable reason code and source location."""

    def __init__(
        self,
        reason_code: str,
        message: str,
        *,
        path: str | Path | None = None,
        line: int | None = None,
    ) -> None:
        self.reason_code = reason_code
        self.path = str(path) if path is not None else None
        self.line = line
        location = self.path or "input"
        if line is not None:
            location = f"{location}, line {line}"
        super().__init__(f"{reason_code}: {location}: {message}")

    def as_dict(self) -> dict[str, Any]:
        return {
            "reason_code": self.reason_code,
            "path": self.path,
            "line": self.line,
            "message": str(self),
        }


@dataclass(frozen=True)
class Artifact:
    bundle_id: str
    artifact_type: str
    schema_id: str
    relative_path: str
    physical_path: Path
    format: str
    compression: str
    record_count: int | None

    @property
    def evidence_path(self) -> str:
        return f"{self.bundle_id}/canonical/{self.relative_path}"


@dataclass(frozen=True)
class LocatedRow:
    value: Mapping[str, Any]
    artifact: Artifact
    line: int

    def reference(self) -> dict[str, Any]:
        return {
            "artifact_type": self.artifact.artifact_type,
            "schema_id": self.artifact.schema_id,
            "path": self.artifact.evidence_path,
            "line": self.line,
        }


@dataclass(frozen=True)
class RawRow:
    raw: bytes
    artifact: Artifact
    line: int

    def decode(self) -> LocatedRow:
        try:
            value = json.loads(self.raw)
        except (UnicodeError, json.JSONDecodeError) as exc:
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"invalid JSONL object: {exc}",
                path=self.artifact.evidence_path,
                line=self.line,
            ) from exc
        if not isinstance(value, Mapping):
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                "JSONL row must be an object",
                path=self.artifact.evidence_path,
                line=self.line,
            )
        return LocatedRow(value=value, artifact=self.artifact, line=self.line)


@dataclass(frozen=True)
class BundleManifest:
    bundle_dir: Path
    bundle_id: str
    export_format_version: str
    parser_schema_version: str
    validation_status: str
    artifacts: Mapping[str, tuple[Artifact, ...]]

    def one(self, artifact_type: str) -> Artifact:
        matches = self.artifacts.get(artifact_type, ())
        if len(matches) != 1:
            raise DatasetContractError(
                "REQUIRED_ARTIFACT_MISSING",
                f"expected exactly one manifest artifact of type {artifact_type}; found {len(matches)}",
                path=self.bundle_dir / "canonical" / "manifest.json",
            )
        return matches[0]

    def many(self, artifact_type: str) -> tuple[Artifact, ...]:
        matches = self.artifacts.get(artifact_type, ())
        if not matches:
            raise DatasetContractError(
                "REQUIRED_ARTIFACT_MISSING",
                f"manifest does not declare artifact type {artifact_type}",
                path=self.bundle_dir / "canonical" / "manifest.json",
            )
        return matches


def _load_json_object(
    path: Path, *, max_bytes: int = MAX_MANIFEST_BYTES
) -> Mapping[str, Any]:
    try:
        size = path.stat().st_size
    except OSError as exc:
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING", str(exc), path=path
        ) from exc
    if size > max_bytes:
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            f"JSON exceeds the {max_bytes}-byte limit",
            path=path,
        )
    try:
        with path.open("r", encoding="utf-8") as handle:
            value = json.load(handle)
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise DatasetContractError(
            "INVALID_FIELD_TYPE", f"invalid JSON: {exc}", path=path
        ) from exc
    if not isinstance(value, Mapping):
        raise DatasetContractError(
            "INVALID_FIELD_TYPE", "expected a JSON object", path=path
        )
    return value


def _safe_resolve(
    base: Path, relative_path: Any, *, location: Path
) -> tuple[str, Path]:
    if not isinstance(relative_path, str) or not relative_path:
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "artifact path must be a non-empty string",
            path=location,
        )
    relative = Path(relative_path)
    if relative.is_absolute():
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            "absolute artifact path is forbidden",
            path=location,
        )
    resolved_base = base.resolve()
    resolved = (base / relative).resolve()
    try:
        resolved.relative_to(resolved_base)
    except ValueError as exc:
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            f"artifact path escapes its manifest root: {relative_path}",
            path=location,
        ) from exc
    return relative.as_posix(), resolved


def _is_symlink_or_reparse(path: Path) -> bool:
    metadata = path.lstat()
    if stat.S_ISLNK(metadata.st_mode):
        return True
    attributes = getattr(metadata, "st_file_attributes", 0)
    reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    return bool(attributes & reparse_flag)


def _strict_direct_root(bundle_dir: Path) -> Path:
    try:
        if _is_symlink_or_reparse(bundle_dir) or not bundle_dir.is_dir():
            raise DatasetContractError(
                "REQUIRED_ARTIFACT_MISSING",
                "direct bundle root must be a regular non-reparse directory",
                path=bundle_dir,
            )
        root = bundle_dir.resolve(strict=True)
    except DatasetContractError:
        raise
    except OSError as exc:
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            f"cannot resolve direct bundle root: {exc}",
            path=bundle_dir,
        ) from exc
    root_manifest = root / "manifest.json"
    try:
        root_manifest.lstat()
    except FileNotFoundError:
        pass
    except OSError as exc:
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            f"cannot inspect forbidden root manifest: {exc}",
            path=root_manifest,
        ) from exc
    else:
        raise DatasetContractError(
            "AMBIGUOUS_BUNDLE_DISCOVERY",
            "direct canonical bundle must not contain a root manifest.json",
            path=root_manifest,
        )
    return root


def _strict_direct_resolve(
    base: Path, relative_path: Any, *, location: Path
) -> tuple[str, Path]:
    if (
        not isinstance(relative_path, str)
        or not relative_path
        or "\\" in relative_path
        or ":" in relative_path
    ):
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "direct artifact path must be a normalized relative POSIX path",
            path=location,
        )
    parts = relative_path.split("/")
    if any(part in {"", ".", ".."} for part in parts):
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            "direct artifact path must not escape or contain empty/dot components",
            path=location,
        )
    candidate = base
    try:
        for index, part in enumerate(parts):
            candidate /= part
            if _is_symlink_or_reparse(candidate):
                raise DatasetContractError(
                    "REQUIRED_ARTIFACT_MISSING",
                    "direct bundle paths must not contain symlinks or reparse points",
                    path=candidate,
                )
            if index < len(parts) - 1 and not candidate.is_dir():
                raise DatasetContractError(
                    "REQUIRED_ARTIFACT_MISSING",
                    "direct artifact parent must be a directory",
                    path=candidate,
                )
        resolved = candidate.resolve(strict=True)
        resolved.relative_to(base)
    except DatasetContractError:
        raise
    except (OSError, ValueError) as exc:
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            f"direct artifact cannot be resolved inside its bundle: {exc}",
            path=candidate,
        ) from exc
    if not resolved.is_file():
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            "direct artifact must be a regular file",
            path=resolved,
        )
    return relative_path, resolved


def _file_sha256(path: Path) -> str:
    digest = hashlib.sha256()
    try:
        with path.open("rb") as handle:
            for chunk in iter(lambda: handle.read(1024 * 1024), b""):
                digest.update(chunk)
    except OSError as exc:
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            f"cannot hash direct artifact: {exc}",
            path=path,
        ) from exc
    return digest.hexdigest()


def _direct_physical_catalog(canonical_root: Path) -> set[str]:
    files: set[str] = set()
    pending = [canonical_root]
    while pending:
        directory = pending.pop()
        try:
            children = sorted(directory.iterdir(), key=lambda item: item.name)
        except OSError as exc:
            raise DatasetContractError(
                "REQUIRED_ARTIFACT_MISSING",
                f"cannot enumerate direct canonical bundle: {exc}",
                path=directory,
            ) from exc
        for child in children:
            try:
                if _is_symlink_or_reparse(child):
                    raise DatasetContractError(
                        "REQUIRED_ARTIFACT_MISSING",
                        "direct canonical bundle must not contain symlinks or reparse points",
                        path=child,
                    )
                if child.is_dir():
                    pending.append(child)
                elif child.is_file():
                    files.add(child.relative_to(canonical_root).as_posix())
                else:
                    raise DatasetContractError(
                        "REQUIRED_ARTIFACT_MISSING",
                        "direct canonical entries must be regular files or directories",
                        path=child,
                    )
            except DatasetContractError:
                raise
            except OSError as exc:
                raise DatasetContractError(
                    "REQUIRED_ARTIFACT_MISSING",
                    f"cannot inspect direct canonical entry: {exc}",
                    path=child,
                ) from exc
    return files


def _manifest_from_canonical(
    *,
    bundle_dir: Path,
    canonical_path: Path,
    canonical: Mapping[str, Any],
    bundle_id: str,
    export_version: str,
    parser_version: str,
    validation_status: str,
    strict_integrity: bool,
) -> BundleManifest:
    if canonical.get("schema_id") != "stratai.canonical_manifest@3":
        raise DatasetContractError(
            "UNSUPPORTED_SCHEMA",
            "expected stratai.canonical_manifest@3",
            path=canonical_path,
        )
    descriptors = canonical.get("artifacts")
    if not isinstance(descriptors, list):
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "canonical artifacts must be a list",
            path=canonical_path,
        )

    canonical_root = canonical_path.parent
    artifacts_by_type: dict[str, list[Artifact]] = defaultdict(list)
    seen_paths: set[str] = set()
    descriptor_paths: list[str] = []
    for index, descriptor in enumerate(descriptors):
        if not isinstance(descriptor, Mapping):
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"artifact descriptor {index} must be an object",
                path=canonical_path,
            )
        if strict_integrity:
            keys = set(descriptor)
            missing = DIRECT_ARTIFACT_REQUIRED_KEYS - keys
            unknown = keys - (
                DIRECT_ARTIFACT_REQUIRED_KEYS | DIRECT_ARTIFACT_OPTIONAL_KEYS
            )
            if missing or unknown:
                raise DatasetContractError(
                    "INVALID_FIELD_TYPE",
                    f"direct artifact descriptor missing={sorted(missing)}, unknown={sorted(unknown)}",
                    path=canonical_path,
                )
        artifact_type = descriptor.get("artifact_type")
        schema_id = descriptor.get("schema_id")
        artifact_format = descriptor.get("format")
        compression = descriptor.get("compression") or "none"
        count = descriptor.get("record_count")
        if not isinstance(artifact_type, str) or not artifact_type:
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                "artifact_type must be a string",
                path=canonical_path,
            )
        if not isinstance(schema_id, str) or not schema_id:
            raise DatasetContractError(
                "INVALID_FIELD_TYPE", "schema_id must be a string", path=canonical_path
            )
        if artifact_format not in {"json", "jsonl"}:
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"unsupported artifact format {artifact_format!r}",
                path=canonical_path,
            )
        if compression not in {"none", "gzip"}:
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"unsupported artifact compression {compression!r}",
                path=canonical_path,
            )
        if count is not None and (
            not isinstance(count, int) or isinstance(count, bool) or count < 0
        ):
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                "record_count must be a non-negative integer",
                path=canonical_path,
            )
        if strict_integrity:
            byte_count = descriptor.get("bytes")
            checksum = descriptor.get("sha256")
            sort_order = descriptor.get("sort_order")
            if (
                not isinstance(byte_count, int)
                or isinstance(byte_count, bool)
                or byte_count < 0
            ):
                raise DatasetContractError(
                    "INVALID_FIELD_TYPE",
                    "direct artifact bytes must be a non-negative integer",
                    path=canonical_path,
                )
            if not isinstance(checksum, str) or not SHA256_PATTERN.fullmatch(checksum):
                raise DatasetContractError(
                    "INVALID_FIELD_TYPE",
                    "direct artifact sha256 must be lowercase hexadecimal",
                    path=canonical_path,
                )
            if not isinstance(sort_order, list) or any(
                not isinstance(field, str) or not field for field in sort_order
            ):
                raise DatasetContractError(
                    "INVALID_FIELD_TYPE",
                    "direct artifact sort_order must be a string list",
                    path=canonical_path,
                )
            relative_path, physical_path = _strict_direct_resolve(
                canonical_root,
                descriptor.get("path"),
                location=canonical_path,
            )
            try:
                actual_bytes = physical_path.stat().st_size
            except OSError as exc:
                raise DatasetContractError(
                    "REQUIRED_ARTIFACT_MISSING",
                    f"cannot inspect direct artifact: {exc}",
                    path=physical_path,
                ) from exc
            if actual_bytes != byte_count:
                raise DatasetContractError(
                    "ARTIFACT_INTEGRITY_MISMATCH",
                    f"artifact bytes mismatch: manifest={byte_count}, actual={actual_bytes}",
                    path=physical_path,
                )
            if _file_sha256(physical_path) != checksum:
                raise DatasetContractError(
                    "ARTIFACT_INTEGRITY_MISMATCH",
                    "artifact SHA-256 mismatch",
                    path=physical_path,
                )
        else:
            relative_path, physical_path = _safe_resolve(
                canonical_root,
                descriptor.get("path"),
                location=canonical_path,
            )
            if not physical_path.is_file():
                raise DatasetContractError(
                    "REQUIRED_ARTIFACT_MISSING",
                    f"declared artifact does not exist: {relative_path}",
                    path=canonical_path,
                )
        if relative_path in seen_paths:
            raise DatasetContractError(
                "DUPLICATE_IDENTIFIER",
                f"duplicate artifact path {relative_path}",
                path=canonical_path,
            )
        seen_paths.add(relative_path)
        descriptor_paths.append(relative_path)
        artifacts_by_type[artifact_type].append(
            Artifact(
                bundle_id=bundle_id,
                artifact_type=artifact_type,
                schema_id=schema_id,
                relative_path=relative_path,
                physical_path=physical_path,
                format=artifact_format,
                compression=compression,
                record_count=count,
            )
        )
    if strict_integrity:
        if descriptor_paths != sorted(descriptor_paths):
            raise DatasetContractError(
                "ARTIFACT_INTEGRITY_MISMATCH",
                "direct artifact descriptors must be ordered by path",
                path=canonical_path,
            )
        expected_files = {"manifest.json", *seen_paths}
        physical_files = _direct_physical_catalog(canonical_root)
        if physical_files != expected_files:
            raise DatasetContractError(
                "ARTIFACT_INTEGRITY_MISMATCH",
                "direct canonical catalog mismatch: "
                f"missing={sorted(expected_files - physical_files)}, "
                f"undeclared={sorted(physical_files - expected_files)}",
                path=canonical_path,
            )
    return BundleManifest(
        bundle_dir=bundle_dir,
        bundle_id=bundle_id,
        export_format_version=export_version,
        parser_schema_version=parser_version,
        validation_status=validation_status,
        artifacts={
            key: tuple(sorted(value, key=lambda item: item.relative_path))
            for key, value in artifacts_by_type.items()
        },
    )


def discover_bundle_dirs(bundles_root: Path) -> tuple[Path, ...]:
    """Discover bundles only through direct child root manifests."""

    if not bundles_root.is_dir():
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            "bundle root directory does not exist",
            path=bundles_root,
        )
    discovered: list[Path] = []
    for child in sorted(bundles_root.iterdir(), key=lambda item: item.name):
        if child.is_dir() and (child / "manifest.json").is_file():
            discovered.append(child)
    return tuple(discovered)


def load_bundle_manifest(bundle_dir: Path) -> BundleManifest:
    """Load the root and canonical manifests, then resolve only declared artifacts."""

    root_path = bundle_dir / "manifest.json"
    root = _load_json_object(root_path)
    bundle_id = root.get("match_id")
    if not isinstance(bundle_id, str) or not bundle_id:
        raise DatasetContractError(
            "INVALID_IDENTIFIER_TYPE",
            "root manifest match_id must be a non-empty string",
            path=root_path,
        )
    validation_status = root.get("validation_status")
    receipt = root.get("validation_receipt")
    receipt_status = receipt.get("status") if isinstance(receipt, Mapping) else None
    if validation_status != "passed" or receipt_status != "passed":
        raise DatasetContractError(
            "BUNDLE_VALIDATION_FAILED",
            "root manifest and validation receipt must both be passed",
            path=root_path,
        )
    export_version = root.get("export_format_version")
    if export_version != EXPECTED_EXPORT_VERSION:
        raise DatasetContractError(
            "UNSUPPORTED_SCHEMA",
            f"expected export format {EXPECTED_EXPORT_VERSION}; found {export_version!r}",
            path=root_path,
        )
    parser_version = root.get("parser_schema_version")
    if not isinstance(parser_version, str) or not parser_version:
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "parser_schema_version must be a non-empty string",
            path=root_path,
        )

    root_artifacts = root.get("artifacts")
    if not isinstance(root_artifacts, list):
        raise DatasetContractError(
            "INVALID_FIELD_TYPE", "root artifacts must be a list", path=root_path
        )
    canonical_descriptors = [
        item
        for item in root_artifacts
        if isinstance(item, Mapping)
        and item.get("artifact_type") == "canonical_manifest"
    ]
    if len(canonical_descriptors) != 1:
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            f"expected one canonical_manifest descriptor; found {len(canonical_descriptors)}",
            path=root_path,
        )
    _, canonical_path = _safe_resolve(
        bundle_dir,
        canonical_descriptors[0].get("path"),
        location=root_path,
    )
    if not canonical_path.is_file():
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            "canonical manifest path does not exist",
            path=canonical_path,
        )
    canonical = _load_json_object(canonical_path)
    return _manifest_from_canonical(
        bundle_dir=bundle_dir,
        canonical_path=canonical_path,
        canonical=canonical,
        bundle_id=bundle_id,
        export_version=export_version,
        parser_version=parser_version,
        validation_status=validation_status,
        strict_integrity=False,
    )


def load_direct_bundle_manifest(
    bundle_dir: Path,
    *,
    expected_match_id: str,
    expected_demo_checksum: str,
) -> BundleManifest:
    """Load one explicit canonical-only bundle and verify every artifact byte/hash."""

    if not isinstance(expected_match_id, str) or not SOURCE_ALIAS_PATTERN.fullmatch(
        expected_match_id
    ):
        raise DatasetContractError(
            "INVALID_IDENTIFIER_TYPE",
            "expected_match_id must match demo-<20 lowercase hex>",
            path=bundle_dir,
        )
    if not isinstance(expected_demo_checksum, str) or not SHA256_PATTERN.fullmatch(
        expected_demo_checksum
    ):
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "expected_demo_checksum must be a lowercase SHA-256",
            path=bundle_dir,
        )
    if expected_match_id != f"demo-{expected_demo_checksum[:20]}":
        raise DatasetContractError(
            "ARTIFACT_INTEGRITY_MISMATCH",
            "expected_match_id must be derived from the approved demo checksum",
            path=bundle_dir,
        )
    root = _strict_direct_root(bundle_dir)
    if root.name != f"match_{expected_match_id}":
        raise DatasetContractError(
            "AMBIGUOUS_BUNDLE_DISCOVERY",
            "direct bundle directory must be named match_<expected_match_id>",
            path=root,
        )
    _, canonical_path = _strict_direct_resolve(
        root,
        "canonical/manifest.json",
        location=root,
    )
    canonical = _load_json_object(canonical_path)
    keys = set(canonical)
    missing = DIRECT_CANONICAL_KEYS - keys
    unknown = keys - DIRECT_CANONICAL_KEYS
    if missing or unknown:
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            f"direct canonical manifest missing={sorted(missing)}, unknown={sorted(unknown)}",
            path=canonical_path,
        )
    if canonical.get("schema_id") != "stratai.canonical_manifest@3":
        raise DatasetContractError(
            "UNSUPPORTED_SCHEMA",
            "expected stratai.canonical_manifest@3",
            path=canonical_path,
        )
    if canonical.get("export_format_version") != EXPECTED_EXPORT_VERSION:
        raise DatasetContractError(
            "UNSUPPORTED_SCHEMA",
            f"expected export format {EXPECTED_EXPORT_VERSION}",
            path=canonical_path,
        )
    if canonical.get("parser_version") != EXPECTED_PARSER_VERSION:
        raise DatasetContractError(
            "UNSUPPORTED_SCHEMA",
            f"expected parser version {EXPECTED_PARSER_VERSION}",
            path=canonical_path,
        )
    if canonical.get("match_id") != expected_match_id:
        raise DatasetContractError(
            "INVALID_IDENTIFIER_TYPE",
            "canonical match_id does not match the expected smoke alias",
            path=canonical_path,
        )
    if canonical.get("demo_checksum_sha256") != expected_demo_checksum:
        raise DatasetContractError(
            "ARTIFACT_INTEGRITY_MISMATCH",
            "canonical demo checksum does not match the approved source",
            path=canonical_path,
        )
    for key in ("configuration_hashes", "transformation_versions", "lineage"):
        if not isinstance(canonical.get(key), Mapping):
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"direct canonical {key} must be an object",
                path=canonical_path,
            )
    return _manifest_from_canonical(
        bundle_dir=root,
        canonical_path=canonical_path,
        canonical=canonical,
        bundle_id=expected_match_id,
        export_version=EXPECTED_EXPORT_VERSION,
        parser_version=EXPECTED_PARSER_VERSION,
        validation_status="direct_integrity_verified",
        strict_integrity=True,
    )


def _binary_jsonl_rows(
    handle: BinaryIO,
    artifact: Artifact,
    *,
    max_line_bytes: int,
    required_fragments_any: tuple[bytes, ...] = (),
) -> Iterator[LocatedRow]:
    physical_line = 0
    record_count = 0
    while True:
        raw = handle.readline(max_line_bytes + 1)
        if not raw:
            break
        physical_line += 1
        if len(raw) > max_line_bytes and not raw.endswith(b"\n"):
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"JSONL line exceeds {max_line_bytes} bytes",
                path=artifact.evidence_path,
                line=physical_line,
            )
        if not raw.strip():
            continue
        record_count += 1
        if required_fragments_any and not any(
            fragment in raw for fragment in required_fragments_any
        ):
            continue
        try:
            value = json.loads(raw)
        except (UnicodeError, json.JSONDecodeError) as exc:
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"invalid JSONL object: {exc}",
                path=artifact.evidence_path,
                line=physical_line,
            ) from exc
        if not isinstance(value, Mapping):
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                "JSONL row must be an object",
                path=artifact.evidence_path,
                line=physical_line,
            )
        yield LocatedRow(value=value, artifact=artifact, line=physical_line)
    if artifact.record_count is not None and record_count != artifact.record_count:
        raise DatasetContractError(
            "REQUIRED_FIELD_MISSING",
            f"record_count mismatch: manifest={artifact.record_count}, streamed={record_count}",
            path=artifact.evidence_path,
        )


def _binary_raw_rows(
    handle: BinaryIO,
    artifact: Artifact,
    *,
    max_line_bytes: int,
) -> Iterator[RawRow]:
    physical_line = 0
    record_count = 0
    while True:
        raw = handle.readline(max_line_bytes + 1)
        if not raw:
            break
        physical_line += 1
        if len(raw) > max_line_bytes and not raw.endswith(b"\n"):
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"JSONL line exceeds {max_line_bytes} bytes",
                path=artifact.evidence_path,
                line=physical_line,
            )
        if not raw.strip():
            continue
        record_count += 1
        yield RawRow(raw=raw, artifact=artifact, line=physical_line)
    if artifact.record_count is not None and record_count != artifact.record_count:
        raise DatasetContractError(
            "REQUIRED_FIELD_MISSING",
            f"record_count mismatch: manifest={artifact.record_count}, streamed={record_count}",
            path=artifact.evidence_path,
        )


def iter_jsonl(
    artifact: Artifact,
    *,
    max_line_bytes: int = MAX_JSONL_LINE_BYTES,
    required_fragments_any: tuple[bytes, ...] = (),
) -> Iterator[LocatedRow]:
    """Yield manifest-declared JSONL rows without persistent decompression."""

    if artifact.format != "jsonl":
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "artifact must declare format=jsonl",
            path=artifact.evidence_path,
        )
    try:
        if artifact.compression == "gzip":
            with gzip.open(artifact.physical_path, "rb") as handle:
                yield from _binary_jsonl_rows(
                    handle,
                    artifact,
                    max_line_bytes=max_line_bytes,
                    required_fragments_any=required_fragments_any,
                )
        elif artifact.compression == "none":
            with artifact.physical_path.open("rb") as handle:
                yield from _binary_jsonl_rows(
                    handle,
                    artifact,
                    max_line_bytes=max_line_bytes,
                    required_fragments_any=required_fragments_any,
                )
        else:
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"unsupported compression {artifact.compression!r}",
                path=artifact.evidence_path,
            )
    except DatasetContractError:
        raise
    except (OSError, EOFError, gzip.BadGzipFile) as exc:
        code = (
            "GZIP_STREAM_CORRUPT"
            if artifact.compression == "gzip"
            else "INVALID_FIELD_TYPE"
        )
        raise DatasetContractError(
            code, f"corrupt or unreadable stream: {exc}", path=artifact.evidence_path
        ) from exc


def iter_jsonl_raw(
    artifact: Artifact,
    *,
    max_line_bytes: int = MAX_JSONL_LINE_BYTES,
) -> Iterator[RawRow]:
    """Stream every raw JSONL record while deferring JSON decoding to selected rows."""

    if artifact.format != "jsonl":
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "artifact must declare format=jsonl",
            path=artifact.evidence_path,
        )
    try:
        if artifact.compression == "gzip":
            with gzip.open(artifact.physical_path, "rb") as handle:
                yield from _binary_raw_rows(
                    handle, artifact, max_line_bytes=max_line_bytes
                )
        elif artifact.compression == "none":
            with artifact.physical_path.open("rb") as handle:
                yield from _binary_raw_rows(
                    handle, artifact, max_line_bytes=max_line_bytes
                )
        else:
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"unsupported compression {artifact.compression!r}",
                path=artifact.evidence_path,
            )
    except DatasetContractError:
        raise
    except (OSError, EOFError, gzip.BadGzipFile) as exc:
        code = (
            "GZIP_STREAM_CORRUPT"
            if artifact.compression == "gzip"
            else "INVALID_FIELD_TYPE"
        )
        raise DatasetContractError(
            code, f"corrupt or unreadable stream: {exc}", path=artifact.evidence_path
        ) from exc
