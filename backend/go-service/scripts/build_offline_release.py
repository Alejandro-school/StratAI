#!/usr/bin/env python3
"""Build a content-addressed offline Go release for the FACEIT demo pipeline."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import urllib.parse
from datetime import datetime, timezone
from typing import Any, Iterable, Mapping


RELEASE_SCHEMA_ID = "stratai.go_offline_release@1"
TARGET_MAP = "de_mirage"
BINARY_RELATIVE_PATH = pathlib.PurePosixPath("bin/stratai-offline-demo-service.exe")
VALIDATOR_RELATIVE_PATH = pathlib.PurePosixPath("scripts/publication_validator.py")
PACKAGED_FILES = (
    pathlib.PurePosixPath("scripts/publication_validator.py"),
    pathlib.PurePosixPath("scripts/canonical_export_validator.py"),
    pathlib.PurePosixPath("scripts/golden_corpus_validator.py"),
    pathlib.PurePosixPath("testdata/golden-corpus/v1/cases.jsonl"),
    pathlib.PurePosixPath("testdata/golden-corpus/v1/manifest.json"),
)
GO_BUILD_ENVIRONMENT = {
    "GOOS": "windows",
    "GOARCH": "amd64",
    "CGO_ENABLED": "0",
    "GOAMD64": "v1",
    "GOEXPERIMENT": "",
    "GOFLAGS": "",
}


def sha256_file(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def source_files(go_root: pathlib.Path) -> list[pathlib.Path]:
    files = [path for path in go_root.rglob("*.go") if path.is_file()]
    files.extend(go_root / name for name in ("go.mod", "go.sum"))
    files.extend(go_root / pathlib.Path(str(relative)) for relative in PACKAGED_FILES)
    files.append(go_root / "scripts" / "build_offline_release.py")
    missing = [path for path in files if not path.is_file()]
    if missing:
        raise RuntimeError(f"release source is incomplete: {missing[0]}")
    return sorted(set(files), key=lambda path: path.relative_to(go_root).as_posix())


def source_digest(go_root: pathlib.Path, files: Iterable[pathlib.Path]) -> str:
    digest = hashlib.sha256()
    for path in files:
        relative = path.relative_to(go_root).as_posix().encode("utf-8")
        data = path.read_bytes()
        digest.update(len(relative).to_bytes(4, "big"))
        digest.update(relative)
        digest.update(len(data).to_bytes(8, "big"))
        digest.update(data)
    return digest.hexdigest()


def git_metadata(project_root: pathlib.Path) -> tuple[str, bool]:
    commit = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=project_root,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        check=False,
    )
    commit_id = commit.stdout.strip() if commit.returncode == 0 else "unavailable"
    status = subprocess.run(
        ["git", "status", "--porcelain", "--untracked-files=all", "--", "backend/go-service"],
        cwd=project_root,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        check=False,
    )
    return commit_id, status.returncode != 0 or bool(status.stdout.strip())


def controlled_go_environment() -> dict[str, str]:
    environment = os.environ.copy()
    environment.update(GO_BUILD_ENVIRONMENT)
    return environment


def go_build_environment(go_root: pathlib.Path, environment: Mapping[str, str]) -> dict[str, str]:
    keys = tuple(GO_BUILD_ENVIRONMENT)
    completed = subprocess.run(
        ["go", "env", *keys],
        cwd=go_root,
        env=environment,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        check=False,
    )
    values = completed.stdout.splitlines()
    if completed.returncode != 0 or len(values) != len(keys):
        raise RuntimeError("could not determine the controlled Go build environment")
    result = dict(zip(keys, (value.strip() for value in values), strict=True))
    if result != GO_BUILD_ENVIRONMENT:
        raise RuntimeError("Go did not accept the controlled build environment")
    return result


def go_version(go_root: pathlib.Path, environment: Mapping[str, str]) -> str:
    completed = subprocess.run(
        ["go", "version"],
        cwd=go_root,
        env=environment,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        check=False,
    )
    value = completed.stdout.strip()
    if completed.returncode != 0 or not value.startswith("go version go"):
        raise RuntimeError("could not determine Go toolchain version")
    return value


def map_asset_catalog(maps_root: pathlib.Path) -> list[dict[str, Any]]:
    maps_root = maps_root.resolve()
    map_dir = maps_root / TARGET_MAP
    physics = map_dir / f"{TARGET_MAP}_physics.gltf"
    nav = map_dir / f"{TARGET_MAP}.nav"
    places = map_dir / f"{TARGET_MAP}_places.json"
    try:
        physics_data = physics.read_bytes()
        gltf = json.loads(physics_data.decode("utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise RuntimeError(f"{TARGET_MAP} physics GLTF is unavailable") from exc
    candidates = {physics, nav, places}
    for collection in (gltf.get("buffers", []), gltf.get("images", [])):
        if not isinstance(collection, list):
            raise RuntimeError(f"{TARGET_MAP} GLTF resource inventory is invalid")
        for item in collection:
            uri = item.get("uri") if isinstance(item, dict) else None
            if not isinstance(uri, str) or not uri:
                continue
            if uri.startswith("data:"):
                continue
            decoded = urllib.parse.unquote(uri)
            pure = pathlib.PurePosixPath(decoded)
            if pure.is_absolute() or any(part in {"", ".", ".."} for part in pure.parts):
                raise RuntimeError(f"{TARGET_MAP} GLTF contains an unsafe resource URI")
            candidates.add(map_dir.joinpath(*pure.parts))
    catalog: list[dict[str, Any]] = []
    for path in sorted(candidates, key=lambda item: item.relative_to(maps_root).as_posix()):
        try:
            resolved = path.resolve(strict=True)
            resolved.relative_to(map_dir.resolve())
        except (OSError, ValueError) as exc:
            raise RuntimeError(f"{TARGET_MAP} map asset is unavailable: {path.name}") from exc
        if path.is_symlink() or not path.is_file():
            raise RuntimeError(f"{TARGET_MAP} map asset is not a regular file: {path.name}")
        data = physics_data if path == physics else path.read_bytes()
        if not data:
            raise RuntimeError(f"{TARGET_MAP} map asset is empty: {path.name}")
        catalog.append(
            {
                "path": path.relative_to(maps_root).as_posix(),
                "bytes": len(data),
                "sha256": hashlib.sha256(data).hexdigest(),
            }
        )
    return catalog


def catalog_digest(catalog: list[dict[str, Any]]) -> str:
    payload = json.dumps(catalog, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(payload).hexdigest()


def build_input_digest(
    source_sha256: str,
    map_assets_sha256: str,
    toolchain: str,
    build_environment: Mapping[str, str],
) -> str:
    payload = json.dumps(
        {
            "source_sha256": source_sha256,
            "map_assets_sha256": map_assets_sha256,
            "go_version": toolchain,
            "go_build_env": dict(build_environment),
        },
        sort_keys=True,
        separators=(",", ":"),
    ).encode("utf-8")
    return hashlib.sha256(payload).hexdigest()


def snapshot_sources(
    go_root: pathlib.Path,
    files: Iterable[pathlib.Path],
    snapshot_root: pathlib.Path,
) -> list[pathlib.Path]:
    """Copy each source once so hashing, compilation and packaging use identical bytes."""
    snapshot_files: list[pathlib.Path] = []
    for source in files:
        relative = source.relative_to(go_root)
        destination = snapshot_root / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(source.read_bytes())
        snapshot_files.append(destination)
    return snapshot_files


def verify_live_inputs_unchanged(
    go_root: pathlib.Path,
    expected_source_sha256: str,
    maps_root: pathlib.Path,
    expected_map_assets: list[dict[str, Any]],
) -> None:
    current_files = source_files(go_root)
    if source_digest(go_root, current_files) != expected_source_sha256:
        raise RuntimeError("Go release sources changed while the release was being built")
    if map_asset_catalog(maps_root) != expected_map_assets:
        raise RuntimeError("Mirage map assets changed while the release was being built")


def run_binary_metadata(binary: pathlib.Path) -> dict[str, Any]:
    completed = subprocess.run(
        [str(binary), "--release-metadata"],
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError("offline service did not report release metadata")
    try:
        payload = json.loads(completed.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError("offline service returned invalid release metadata") from exc
    if not isinstance(payload, dict) or payload.get("schema_id") != RELEASE_SCHEMA_ID:
        raise RuntimeError("offline service release metadata has the wrong schema")
    return payload


def artifact_catalog(release_dir: pathlib.Path) -> list[dict[str, Any]]:
    artifacts: list[dict[str, Any]] = []
    for path in sorted(release_dir.rglob("*"), key=lambda item: item.relative_to(release_dir).as_posix()):
        if not path.is_file() or path.name == "RELEASE.json":
            continue
        relative = path.relative_to(release_dir).as_posix()
        artifacts.append(
            {
                "path": relative,
                "bytes": path.stat().st_size,
                "sha256": sha256_file(path),
            }
        )
    return artifacts


def verify_existing_release(
    release_dir: pathlib.Path,
    build_id: str,
    source_sha256: str,
    input_sha256: str,
    toolchain: str,
    goos: str,
    goarch: str,
    build_environment: Mapping[str, str],
    map_assets: list[dict[str, Any]],
    map_assets_sha256: str,
) -> bool:
    manifest_path = release_dir / "RELEASE.json"
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError):
        return False
    if (
        not isinstance(manifest, dict)
        or manifest.get("schema_id") != RELEASE_SCHEMA_ID
        or manifest.get("build_id") != build_id
        or manifest.get("source_sha256") != source_sha256
        or manifest.get("build_input_sha256") != input_sha256
        or manifest.get("go_version") != toolchain
        or manifest.get("goos") != goos
        or manifest.get("goarch") != goarch
        or manifest.get("go_build_env") != dict(build_environment)
        or manifest.get("target_map") != TARGET_MAP
        or manifest.get("map_assets") != map_assets
        or manifest.get("map_assets_sha256") != map_assets_sha256
        or manifest.get("binary_path") != BINARY_RELATIVE_PATH.as_posix()
        or manifest.get("validator_path") != VALIDATOR_RELATIVE_PATH.as_posix()
    ):
        return False
    artifacts = manifest.get("artifacts")
    if not isinstance(artifacts, list) or not artifacts:
        return False
    declared: set[str] = set()
    for artifact in artifacts:
        if not isinstance(artifact, dict) or set(artifact) != {"path", "bytes", "sha256"}:
            return False
        relative = artifact.get("path")
        size = artifact.get("bytes")
        checksum = artifact.get("sha256")
        pure = pathlib.PurePosixPath(relative) if isinstance(relative, str) else None
        if (
            pure is None
            or pure.is_absolute()
            or "\\" in relative
            or any(part in {"", ".", ".."} for part in pure.parts)
            or relative in declared
            or not isinstance(size, int)
            or isinstance(size, bool)
            or size < 1
            or not isinstance(checksum, str)
            or len(checksum) != 64
        ):
            return False
        path = release_dir.joinpath(*pure.parts)
        try:
            resolved = path.resolve(strict=True)
            resolved.relative_to(release_dir.resolve())
        except (OSError, ValueError):
            return False
        if (
            not path.is_file()
            or path.is_symlink()
            or path.stat().st_size != size
            or sha256_file(path) != checksum
        ):
            return False
        declared.add(relative)
    required = {BINARY_RELATIVE_PATH.as_posix(), *(relative.as_posix() for relative in PACKAGED_FILES)}
    if not required.issubset(declared):
        return False
    disk_entries = list(release_dir.rglob("*"))
    if any(path.is_symlink() for path in disk_entries):
        return False
    disk_inventory = {
        path.relative_to(release_dir).as_posix()
        for path in disk_entries
        if path.is_file() and path.name != "RELEASE.json"
    }
    if disk_inventory != declared:
        return False
    try:
        metadata = run_binary_metadata(release_dir.joinpath(*BINARY_RELATIVE_PATH.parts))
    except RuntimeError:
        return False
    for field in (
        "build_id",
        "parser_schema_version",
        "export_format_version",
        "quality_schema_version",
        "validator_version",
    ):
        if metadata.get(field) != manifest.get(field):
            return False
    return True


def write_current_pointer(output_root: pathlib.Path, build_id: str) -> None:
    descriptor, temporary_name = tempfile.mkstemp(prefix=".CURRENT.", suffix=".tmp", dir=output_root)
    temporary = pathlib.Path(temporary_name)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as handle:
            handle.write(build_id + "\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, output_root / "CURRENT")
    finally:
        temporary.unlink(missing_ok=True)


def build_release(
    go_root: pathlib.Path, output_root: pathlib.Path, maps_root: pathlib.Path
) -> dict[str, Any]:
    if os.name != "nt":
        raise RuntimeError("the current MVP release builder targets Windows only")
    go_root = go_root.resolve()
    output_root = output_root.resolve()
    output_root.mkdir(parents=True, exist_ok=True)
    source_snapshot_parent = pathlib.Path(
        tempfile.mkdtemp(prefix=".stratai-go-source-", dir=output_root)
    )
    try:
        source_snapshot_root = source_snapshot_parent / "go-service"
        snapshot_files_list = snapshot_sources(
            go_root, source_files(go_root), source_snapshot_root
        )
        digest = source_digest(source_snapshot_root, snapshot_files_list)
        build_process_environment = controlled_go_environment()
        build_environment = go_build_environment(source_snapshot_root, build_process_environment)
        goos = build_environment["GOOS"]
        goarch = build_environment["GOARCH"]
        toolchain = go_version(source_snapshot_root, build_process_environment)
        map_assets = map_asset_catalog(maps_root)
        map_assets_sha256 = catalog_digest(map_assets)
        input_digest = build_input_digest(
            digest, map_assets_sha256, toolchain, build_environment
        )
        build_id = f"stratai-{input_digest[:16]}"
        release_dir = output_root / build_id

        if release_dir.exists():
            if not verify_existing_release(
                release_dir,
                build_id,
                digest,
                input_digest,
                toolchain,
                goos,
                goarch,
                build_environment,
                map_assets,
                map_assets_sha256,
            ):
                raise RuntimeError(
                    f"immutable release directory already exists but failed verification: {release_dir}"
                )
            verify_live_inputs_unchanged(go_root, digest, maps_root, map_assets)
            write_current_pointer(output_root, build_id)
            return {"status": "reused", "build_id": build_id, "release_dir": str(release_dir)}

        staging = pathlib.Path(tempfile.mkdtemp(prefix=f".{build_id}-", dir=output_root))
        try:
            binary = staging / pathlib.Path(str(BINARY_RELATIVE_PATH))
            binary.parent.mkdir(parents=True, exist_ok=True)
            completed = subprocess.run(
                [
                    "go",
                    "build",
                    "-buildvcs=false",
                    "-trimpath",
                    f"-ldflags=-X main.buildID={build_id}",
                    "-o",
                    str(binary),
                    "./cmd/offline-demo-service",
                ],
                cwd=source_snapshot_root,
                env=build_process_environment,
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
                check=False,
            )
            if completed.returncode != 0:
                raise RuntimeError(f"offline Go build failed: {completed.stderr.strip()}")
            metadata = run_binary_metadata(binary)
            if metadata.get("build_id") != build_id:
                raise RuntimeError("offline binary build identity does not match the release")

            for relative in PACKAGED_FILES:
                source = source_snapshot_root / pathlib.Path(str(relative))
                destination = staging / pathlib.Path(str(relative))
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(source, destination)

            verify_live_inputs_unchanged(go_root, digest, maps_root, map_assets)
            project_root = go_root.parent.parent
            commit_id, dirty = git_metadata(project_root)
            manifest = {
                "schema_id": RELEASE_SCHEMA_ID,
                "build_id": build_id,
                "created_at": datetime.now(timezone.utc).isoformat(),
                "source_commit": commit_id,
                "source_dirty": dirty,
                "source_sha256": digest,
                "build_input_sha256": input_digest,
                "go_version": toolchain,
                "goos": goos,
                "goarch": goarch,
                "go_build_env": build_environment,
                "target_map": TARGET_MAP,
                "map_assets_sha256": map_assets_sha256,
                "map_assets": map_assets,
                "binary_path": BINARY_RELATIVE_PATH.as_posix(),
                "validator_path": VALIDATOR_RELATIVE_PATH.as_posix(),
                "parser_schema_version": metadata["parser_schema_version"],
                "export_format_version": metadata["export_format_version"],
                "quality_schema_version": metadata["quality_schema_version"],
                "validator_version": metadata["validator_version"],
                "artifacts": artifact_catalog(staging),
            }
            (staging / "RELEASE.json").write_text(
                json.dumps(manifest, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
                encoding="utf-8",
            )
            os.replace(staging, release_dir)
            write_current_pointer(output_root, build_id)
            return {"status": "built", "build_id": build_id, "release_dir": str(release_dir)}
        except BaseException:
            shutil.rmtree(staging, ignore_errors=True)
            raise
    finally:
        shutil.rmtree(source_snapshot_parent, ignore_errors=True)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    go_root = pathlib.Path(__file__).resolve().parent.parent
    parser.add_argument(
        "--output-root",
        type=pathlib.Path,
        default=go_root.parent.parent / "stratai-go-release",
    )
    parser.add_argument(
        "--maps-root",
        type=pathlib.Path,
        default=go_root.parent.parent.parent / "Faceit-Demos" / "data" / "maps",
    )
    args = parser.parse_args(argv)
    try:
        result = build_release(go_root, args.output_root, args.maps_root)
    except (OSError, RuntimeError, subprocess.SubprocessError) as exc:
        print(json.dumps({"status": "failed", "error": str(exc)}, sort_keys=True), file=sys.stderr)
        return 1
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
