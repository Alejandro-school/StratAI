#!/usr/bin/env python3
"""Reprocess and validate the selected golden demos without persisting private paths."""

from __future__ import annotations

import argparse
import contextlib
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
from collections.abc import Callable, Iterator, Mapping, Sequence
from dataclasses import dataclass
from pathlib import Path

import canonical_export_validator
import golden_demo_corpus_validator as corpus_validator

RUNNER_VERSION = corpus_validator.SEMANTIC_RUNNER_VERSION
RESULT_PREFIX = "STRATAI_RESULT_JSON="
DEFAULT_TIMEOUT_SECONDS = 1_500
DEFAULT_BUILD_ID = "golden-demo-semantic-runner@1"


class RunnerError(RuntimeError):
    """Raised for an operational failure that must not quarantine a demo."""


@dataclass(frozen=True)
class ExportAttempt:
    return_code: int
    stage: str
    error_code: str
    stderr: str


@dataclass(frozen=True)
class RunSummary:
    status: str
    evaluated_count: int
    passed_count: int
    quarantined_count: int
    processed_count: int
    skipped_count: int


def selected_sources(
    sources: Sequence[Mapping[str, object]],
) -> list[Mapping[str, object]]:
    return [
        source
        for source in sources
        if isinstance(source.get("selection"), Mapping)
        and source["selection"].get("selected") is True
    ]


def current_case(case: Mapping[str, object]) -> bool:
    if case.get("runner_version") != RUNNER_VERSION:
        return False
    checks = case.get("checks")
    if not isinstance(checks, list):
        return False
    if case.get("status") == "passed":
        return corpus_validator.REQUIRED_PASS_CHECKS <= set(checks)
    return case.get("status") == "quarantined" and "source_identity" in checks


def semantic_status(
    selected: Sequence[Mapping[str, object]],
    cases: Sequence[Mapping[str, object]],
) -> str:
    if not cases:
        return "not_evaluated"
    selected_aliases = {str(source.get("source_alias")) for source in selected}
    evaluated_aliases = {str(case.get("source_alias")) for case in cases}
    if evaluated_aliases != selected_aliases:
        return "partial"
    if any(case.get("status") == "quarantined" for case in cases):
        return "blocked"
    return "passed"


def refresh_semantic_manifest(
    manifest: dict[str, object],
    sources: Sequence[Mapping[str, object]],
    cases_by_alias: Mapping[str, Mapping[str, object]],
) -> None:
    selected = selected_sources(sources)
    ordered_cases = [
        dict(cases_by_alias[alias])
        for source in selected
        if (alias := str(source.get("source_alias"))) in cases_by_alias
    ]
    status = semantic_status(selected, ordered_cases)
    passed_count = sum(case.get("status") == "passed" for case in ordered_cases)
    quarantined_count = sum(
        case.get("status") == "quarantined" for case in ordered_cases
    )
    manifest["semantic_validation"] = {
        "status": status,
        "evaluated_count": len(ordered_cases),
        "passed_count": passed_count,
        "quarantined_count": quarantined_count,
        "cases": ordered_cases,
    }
    manifest["gate_1"] = corpus_validator.expected_gate_declaration(status)
    manifest["training_allowed"] = False


def write_manifest_atomic(path: Path, manifest: Mapping[str, object]) -> None:
    errors, _ = corpus_validator.validate_manifest_metadata(manifest)
    if errors:
        raise RunnerError("refusing to write invalid semantic manifest: " + "; ".join(errors))
    path = Path(path)
    descriptor, temporary_name = tempfile.mkstemp(
        prefix=f".{path.name}.", suffix=".tmp", dir=path.parent
    )
    temporary = Path(temporary_name)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as handle:
            json.dump(manifest, handle, ensure_ascii=True, indent=2)
            handle.write("\n")
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)


@contextlib.contextmanager
def exclusive_runner_lock(work_root: Path) -> Iterator[None]:
    work_root.mkdir(parents=True, exist_ok=True)
    lock_path = work_root / "semantic-runner.lock"
    with lock_path.open("a+b") as handle:
        handle.seek(0, os.SEEK_END)
        if handle.tell() == 0:
            handle.write(b"0")
            handle.flush()
        handle.seek(0)
        if os.name == "nt":
            import msvcrt

            try:
                msvcrt.locking(handle.fileno(), msvcrt.LK_NBLCK, 1)
            except OSError as error:
                raise RunnerError("another semantic runner is active") from error
            try:
                yield
            finally:
                handle.seek(0)
                msvcrt.locking(handle.fileno(), msvcrt.LK_UNLCK, 1)
        else:
            import fcntl

            try:
                fcntl.flock(handle.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
            except OSError as error:
                raise RunnerError("another semantic runner is active") from error
            try:
                yield
            finally:
                fcntl.flock(handle.fileno(), fcntl.LOCK_UN)


def build_exporter(go_root: Path, work_root: Path) -> Path:
    binary_dir = work_root / "bin"
    binary_dir.mkdir(parents=True, exist_ok=True)
    suffix = ".exe" if os.name == "nt" else ""
    binary = binary_dir / f"canonical-export{suffix}"
    result = subprocess.run(
        ["go", "build", "-o", str(binary), "./cmd/canonical-export"],
        cwd=go_root,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        check=False,
    )
    if result.returncode != 0:
        raise RunnerError("could not build the canonical exporter")
    return binary.resolve()


def parse_export_result(stdout: str) -> Mapping[str, object]:
    for line in reversed(stdout.splitlines()):
        if not line.startswith(RESULT_PREFIX):
            continue
        try:
            value = json.loads(line[len(RESULT_PREFIX) :])
        except json.JSONDecodeError as error:
            raise RunnerError("canonical exporter returned invalid result JSON") from error
        if isinstance(value, Mapping):
            return value
    raise RunnerError("canonical exporter returned no structured result")


def invoke_exporter(
    exporter: Path,
    source: Mapping[str, object],
    demo_path: Path,
    output_root: Path,
    maps_root: Path | None,
    gomaxprocs: int,
    timeout_seconds: int,
) -> ExportAttempt:
    alias = str(source.get("source_alias"))
    checksum = str(source.get("sha256"))
    command = [
        str(exporter),
        "--demo",
        str(demo_path),
        "--output",
        str(output_root),
        "--alias",
        alias,
        "--checksum",
        checksum,
        "--build-id",
        DEFAULT_BUILD_ID,
    ]
    if maps_root is not None:
        command.extend(["--maps-root", str(maps_root)])
    environment = os.environ.copy()
    environment["GOMAXPROCS"] = str(gomaxprocs)
    try:
        result = subprocess.run(
            command,
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            timeout=timeout_seconds,
            env=environment,
            check=False,
        )
    except subprocess.TimeoutExpired as error:
        raise RunnerError(f"export timed out for {alias}") from error
    structured = parse_export_result(result.stdout)
    if structured.get("source_alias") not in {None, alias}:
        raise RunnerError(f"exporter alias mismatch for {alias}")
    if result.returncode == 0 and (
        structured.get("status") != "exported"
        or structured.get("sha256") != checksum
        or structured.get("bundle_dir") != f"match_{alias}"
    ):
        raise RunnerError(f"exporter success contract mismatch for {alias}")
    return ExportAttempt(
        return_code=result.returncode,
        stage=str(structured.get("stage") or "unknown"),
        error_code=str(structured.get("error_code") or ""),
        stderr=result.stderr,
    )


def classify_export_failure(attempt: ExportAttempt) -> tuple[str, str] | None:
    if attempt.return_code == 4 and attempt.stage == "parse":
        return "demo_parse", "parser_rejected_demo"
    if attempt.return_code != 5 or attempt.stage != "export":
        return None
    lowered = attempt.stderr.lower()
    classifications = (
        (
            "objective data failed pre-export validation",
            "objective_round_reconciliation",
            "objective_quality_gate_rejected",
        ),
        ("utility data failed pre-export validation", "utility_quality", "utility_quality_gate_rejected"),
        ("combat data failed pre-export validation", "combat_quality", "combat_quality_gate_rejected"),
        (
            "canonical roster identity quality gates failed",
            "roster_identity",
            "unreconciled_playing_participant",
        ),
        ("canonical engagement quality gates failed", "engagement_quality", "engagement_quality_gate_rejected"),
        ("canonical block 6 quality gates failed", "block6_quality", "block6_quality_gate_rejected"),
        ("build tactical export", "tactical_export", "tactical_export_rejected"),
    )
    for marker, domain, reason in classifications:
        if marker in lowered:
            return domain, reason
    return "canonical_contract", "canonical_validator_rejected_bundle"


def hash_tree(root: Path) -> dict[str, str]:
    hashes: dict[str, str] = {}
    for path in sorted(root.rglob("*")):
        if not path.is_file():
            continue
        digest = hashlib.sha256()
        with path.open("rb") as handle:
            while chunk := handle.read(1024 * 1024):
                digest.update(chunk)
        hashes[path.relative_to(root).as_posix()] = digest.hexdigest()
    return hashes


def bundle_notes(bundle: Path) -> list[str]:
    notes: set[str] = set()
    manifest_path = bundle / "canonical" / "manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    lineage = manifest.get("lineage") if isinstance(manifest, Mapping) else None
    input_hashes = lineage.get("input_hashes") if isinstance(lineage, Mapping) else None
    physics = input_hashes.get("physics_map") if isinstance(input_hashes, Mapping) else None
    if isinstance(physics, Mapping) and physics.get("status") == "observed":
        notes.add("map_geometry_loaded")
    else:
        notes.add("map_geometry_unavailable_tradeability_abstained")

    decision_path = bundle / "canonical" / "causal" / "decisions.jsonl"
    with decision_path.open("r", encoding="utf-8") as handle:
        for line in handle:
            if not line.strip():
                continue
            decision = json.loads(line)
            decision_type = decision.get("decision_type")
            if decision_type == "spacing_or_trade_connection":
                notes.add("decision_family_spacing_or_trade_connection_observed")
            elif decision_type == "peek_hold_or_reposition":
                notes.add("decision_family_peek_hold_or_reposition_observed")
    return sorted(notes)


def validate_direct_bundle(
    match_dir: Path,
    expected_match_id: str | None,
    expected_demo_checksum: str | None,
) -> list[str]:
    """Validate a direct CLI export, which intentionally has no publication catalog."""

    if expected_match_id is None:
        return ["expected_match_id is required"]
    errors = canonical_export_validator.ValidationErrors()
    canonical_export_validator.validate_canonical_bundle(
        Path(match_dir), expected_match_id, errors
    )
    manifest_path = Path(match_dir) / "canonical" / "manifest.json"
    if manifest_path.is_file():
        try:
            manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        except (OSError, UnicodeError, json.JSONDecodeError):
            errors.append("canonical/manifest.json: no se pudo verificar el checksum")
        else:
            if manifest.get("demo_checksum_sha256") != expected_demo_checksum:
                errors.append("canonical/manifest.json: demo_checksum_sha256 no coincide")
    return errors.as_report()


def passed_case(alias: str, notes: Sequence[str], deterministic: bool) -> dict[str, object]:
    checks = set(corpus_validator.REQUIRED_PASS_CHECKS)
    if deterministic:
        checks.add("gomaxprocs_1_4_tree_hash_equality")
    return {
        "source_alias": alias,
        "status": "passed",
        "checks": sorted(checks),
        "failure_domain": None,
        "reason": None,
        "notes": sorted(set(notes)),
        "runner_version": RUNNER_VERSION,
    }


def quarantine_case(
    alias: str,
    checks: Sequence[str],
    domain: str,
    reason: str,
    existing: Mapping[str, object] | None = None,
) -> dict[str, object]:
    notes: list[str] = []
    if (
        existing is not None
        and existing.get("status") == "quarantined"
        and existing.get("failure_domain") == domain
        and existing.get("reason") == "incomplete_final_round"
    ):
        reason = "incomplete_final_round"
        notes = ["objective_round_mismatch_detected"]
    return {
        "source_alias": alias,
        "status": "quarantined",
        "checks": sorted(set(checks)),
        "failure_domain": domain,
        "reason": reason,
        "notes": notes,
        "runner_version": RUNNER_VERSION,
    }


def process_source(
    exporter: Path,
    source: Mapping[str, object],
    demo_path: Path,
    work_root: Path,
    maps_root: Path | None,
    timeout_seconds: int,
    deterministic: bool,
    existing: Mapping[str, object] | None,
    validator: Callable[[Path, str | None, str | None], list[str]] = (
        validate_direct_bundle
    ),
) -> dict[str, object]:
    alias = str(source.get("source_alias"))
    checksum = str(source.get("sha256"))
    first_root = Path(tempfile.mkdtemp(prefix=f"{alias}-p1-", dir=work_root))
    second_root: Path | None = None
    try:
        first = invoke_exporter(
            exporter, source, demo_path, first_root, maps_root, 1, timeout_seconds
        )
        if first.return_code != 0:
            classification = classify_export_failure(first)
            if classification is None:
                raise RunnerError(f"operational exporter failure for {alias}")
            domain, reason = classification
            return quarantine_case(
                alias, ["source_identity"], domain, reason, existing
            )
        first_bundle = first_root / f"match_{alias}"
        validation_errors = validator(first_bundle, alias, checksum)
        if validation_errors:
            return quarantine_case(
                alias,
                ["canonical_export", "source_identity"],
                "canonical_contract",
                "canonical_validator_rejected_bundle",
                existing,
            )
        notes = bundle_notes(first_bundle)

        if deterministic:
            second_root = Path(tempfile.mkdtemp(prefix=f"{alias}-p4-", dir=work_root))
            second = invoke_exporter(
                exporter, source, demo_path, second_root, maps_root, 4, timeout_seconds
            )
            if second.return_code != 0:
                classification = classify_export_failure(second)
                if classification is None:
                    raise RunnerError(f"operational determinism export failure for {alias}")
                domain, reason = classification
                return quarantine_case(alias, ["source_identity"], domain, reason, existing)
            second_bundle = second_root / f"match_{alias}"
            second_errors = validator(second_bundle, alias, checksum)
            if second_errors:
                return quarantine_case(
                    alias,
                    ["canonical_export", "source_identity"],
                    "canonical_contract",
                    "canonical_validator_rejected_bundle",
                    existing,
                )
            if hash_tree(first_bundle / "canonical") != hash_tree(
                second_bundle / "canonical"
            ):
                return quarantine_case(
                    alias,
                    sorted(corpus_validator.REQUIRED_PASS_CHECKS),
                    "determinism",
                    "determinism_mismatch",
                    existing,
                )
        return passed_case(alias, notes, deterministic)
    finally:
        shutil.rmtree(first_root, ignore_errors=True)
        if second_root is not None:
            shutil.rmtree(second_root, ignore_errors=True)


def run_corpus(
    manifest_path: Path,
    roots: Mapping[str, Path],
    work_root: Path,
    maps_root: Path | None,
    exporter: Path | None = None,
    timeout_seconds: int = DEFAULT_TIMEOUT_SECONDS,
    force: bool = False,
    source_aliases: Sequence[str] = (),
    limit: int | None = None,
    determinism_alias: str | None = None,
    processor: Callable[..., dict[str, object]] = process_source,
) -> RunSummary:
    manifest, load_errors = corpus_validator.load_manifest(manifest_path)
    if manifest is None or load_errors:
        raise RunnerError("golden manifest could not be loaded")
    metadata_errors, sources = corpus_validator.validate_manifest_metadata(manifest)
    if metadata_errors:
        raise RunnerError("golden manifest metadata is invalid: " + "; ".join(metadata_errors))
    resolved, source_errors = corpus_validator.resolve_local_sources(sources, roots)
    if source_errors:
        raise RunnerError("golden source inventory is invalid: " + "; ".join(source_errors))

    selected = selected_sources(sources)
    selected_alias_set = {str(source.get("source_alias")) for source in selected}
    requested = set(source_aliases)
    if requested - selected_alias_set:
        raise RunnerError("a requested alias is not part of the selected corpus")
    if determinism_alias is not None and determinism_alias not in selected_alias_set:
        raise RunnerError("the determinism alias is not part of the selected corpus")

    semantic = manifest.get("semantic_validation")
    raw_cases = semantic.get("cases") if isinstance(semantic, Mapping) else []
    cases_by_alias: dict[str, Mapping[str, object]] = {
        str(case.get("source_alias")): case
        for case in raw_cases
        if isinstance(case, Mapping)
    }
    current_determinism_aliases = {
        alias
        for alias, case in cases_by_alias.items()
        if current_case(case)
        and isinstance(case.get("checks"), list)
        and "gomaxprocs_1_4_tree_hash_equality" in case["checks"]
    }
    if len(current_determinism_aliases) > 1:
        raise RunnerError("more than one current determinism case exists")
    assigned_determinism = next(iter(current_determinism_aliases), None)
    target_determinism = determinism_alias or assigned_determinism

    if exporter is None:
        exporter = build_exporter(Path(__file__).resolve().parent.parent, work_root)
    elif not exporter.is_file():
        raise RunnerError("configured canonical exporter does not exist")

    processed_count = 0
    skipped_count = 0
    for source in selected:
        alias = str(source.get("source_alias"))
        if requested and alias not in requested:
            continue
        existing = cases_by_alias.get(alias)
        needs_determinism = target_determinism == alias and assigned_determinism != alias
        if not force and existing is not None and current_case(existing) and not needs_determinism:
            skipped_count += 1
            continue
        if limit is not None and processed_count >= limit:
            break

        use_determinism = needs_determinism
        if target_determinism is None and assigned_determinism is None:
            use_determinism = True
        print(
            json.dumps(
                {"event": "case_started", "source_alias": alias},
                ensure_ascii=True,
                sort_keys=True,
            ),
            flush=True,
        )
        case = processor(
            exporter=exporter,
            source=source,
            demo_path=resolved[alias],
            work_root=work_root,
            maps_root=maps_root,
            timeout_seconds=timeout_seconds,
            deterministic=use_determinism,
            existing=existing,
        )
        if (
            use_determinism
            and case.get("status") == "passed"
            and isinstance(case.get("checks"), list)
            and "gomaxprocs_1_4_tree_hash_equality" in case["checks"]
        ):
            assigned_determinism = alias
            target_determinism = alias
        cases_by_alias[alias] = case
        refresh_semantic_manifest(manifest, sources, cases_by_alias)
        write_manifest_atomic(manifest_path, manifest)
        processed_count += 1
        print(
            json.dumps(
                {
                    "event": "case_finished",
                    "source_alias": alias,
                    "status": case.get("status"),
                },
                ensure_ascii=True,
                sort_keys=True,
            ),
            flush=True,
        )

    semantic = manifest["semantic_validation"]
    assert isinstance(semantic, Mapping)
    return RunSummary(
        status=str(semantic.get("status")),
        evaluated_count=int(semantic.get("evaluated_count", 0)),
        passed_count=int(semantic.get("passed_count", 0)),
        quarantined_count=int(semantic.get("quarantined_count", 0)),
        processed_count=processed_count,
        skipped_count=skipped_count,
    )


def default_manifest_path() -> Path:
    return corpus_validator.default_manifest_path()


def default_work_root() -> Path:
    return Path(__file__).resolve().parents[2] / "data" / "golden-demo-runs" / "v2"


def parse_arguments(argv: Sequence[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", nargs="?", type=Path, default=default_manifest_path())
    parser.add_argument("--stratai-root", type=Path, required=True)
    parser.add_argument("--faceit-root", type=Path, required=True)
    parser.add_argument("--maps-root", type=Path)
    parser.add_argument("--work-root", type=Path, default=default_work_root())
    parser.add_argument("--exporter", type=Path)
    parser.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT_SECONDS)
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--source-alias", action="append", default=[])
    parser.add_argument("--limit", type=int)
    parser.add_argument("--determinism-alias")
    arguments = parser.parse_args(argv)
    if arguments.timeout <= 0:
        parser.error("--timeout must be positive")
    if arguments.limit is not None and arguments.limit <= 0:
        parser.error("--limit must be positive")
    return arguments


def main(argv: Sequence[str] | None = None) -> int:
    arguments = parse_arguments(argv)
    work_root = arguments.work_root.resolve()
    roots = {
        "stratai_runtime": arguments.stratai_root.resolve(),
        "faceit_offline": arguments.faceit_root.resolve(),
    }
    try:
        with exclusive_runner_lock(work_root):
            summary = run_corpus(
                arguments.manifest.resolve(),
                roots,
                work_root,
                arguments.maps_root.resolve() if arguments.maps_root else None,
                exporter=arguments.exporter.resolve() if arguments.exporter else None,
                timeout_seconds=arguments.timeout,
                force=arguments.force,
                source_aliases=arguments.source_alias,
                limit=arguments.limit,
                determinism_alias=arguments.determinism_alias,
            )
    except RunnerError as error:
        print(
            json.dumps(
                {"status": "runner_error", "reason": str(error)},
                ensure_ascii=True,
                sort_keys=True,
            ),
            file=sys.stderr,
        )
        return 2
    print(json.dumps(summary.__dict__, ensure_ascii=True, sort_keys=True))
    return 1 if summary.status == "blocked" else 0


if __name__ == "__main__":
    raise SystemExit(main())
