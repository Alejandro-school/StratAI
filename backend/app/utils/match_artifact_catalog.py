"""Resolve versioned match artifacts without coupling readers to disk layout."""

import json
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path
from typing import Any


@dataclass(frozen=True, slots=True)
class MatchArtifact:
    artifact_type: str
    relative_path: str


def load_match_manifest(match_dir: Path) -> dict[str, Any]:
    manifest_path = match_dir / "manifest.json"
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError):
        return {}
    return manifest if isinstance(manifest, dict) else {}


def resolve_match_artifact(
    match_dir: Path,
    artifact_type: str,
    legacy_path: str | None = None,
) -> Path | None:
    artifacts = resolve_match_artifacts(match_dir, artifact_type)
    if artifacts:
        return artifacts[0]

    if legacy_path is None:
        return None
    resolved = _resolve_safe_path(match_dir, legacy_path)
    return resolved if resolved is not None and resolved.is_file() else None


def resolve_match_artifacts(match_dir: Path, artifact_type: str) -> tuple[Path, ...]:
    match_dir = match_dir.resolve()
    manifest_path = match_dir / "manifest.json"
    try:
        stat = manifest_path.stat()
    except OSError:
        return ()

    descriptors = _load_artifact_descriptors(
        str(manifest_path),
        stat.st_mtime_ns,
        stat.st_size,
    )
    resolved_paths = []
    for descriptor in descriptors:
        if descriptor.artifact_type != artifact_type:
            continue
        resolved = _resolve_safe_path(match_dir, descriptor.relative_path)
        if resolved is not None and resolved.is_file():
            resolved_paths.append(resolved)
    return tuple(resolved_paths)


@lru_cache(maxsize=256)
def _load_artifact_descriptors(
    manifest_path: str,
    _mtime_ns: int,
    _size: int,
) -> tuple[MatchArtifact, ...]:
    try:
        payload = json.loads(Path(manifest_path).read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError):
        return ()

    artifacts = payload.get("artifacts", []) if isinstance(payload, dict) else []
    return tuple(
        MatchArtifact(
            artifact_type=str(descriptor["artifact_type"]),
            relative_path=str(descriptor["path"]),
        )
        for descriptor in artifacts
        if isinstance(descriptor, dict)
        and isinstance(descriptor.get("artifact_type"), str)
        and isinstance(descriptor.get("path"), str)
    )


def _resolve_safe_path(match_dir: Path, relative_path: object) -> Path | None:
    if not isinstance(relative_path, str) or not relative_path:
        return None
    candidate = (match_dir / relative_path).resolve()
    try:
        candidate.relative_to(match_dir)
    except ValueError:
        return None
    return candidate
