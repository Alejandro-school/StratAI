#!/usr/bin/env python3
"""Validate the immutable synthetic golden corpus used by Block 7 gates."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path

GOLDEN_CORPUS_SCHEMA = "stratai.golden_corpus@1"
REQUIRED_CASE_FIELDS = frozenset(
    {
        "case_id",
        "source",
        "tags",
        "input",
        "expected_facts",
        "expected_derived_values",
        "expected_warnings",
        "expected_hard_failures",
        "training_allowed",
    }
)


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def validate_golden_corpus(corpus_dir: Path) -> list[str]:
    corpus_dir = Path(corpus_dir)
    manifest_path = corpus_dir / "manifest.json"
    cases_path = corpus_dir / "cases.jsonl"
    errors: list[str] = []
    if not manifest_path.is_file() or not cases_path.is_file():
        return [f"{corpus_dir}: manifest.json y cases.jsonl son obligatorios"]
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        return [f"{manifest_path}: JSON invalido ({error})"]
    if manifest.get("schema_id") != GOLDEN_CORPUS_SCHEMA:
        errors.append(f"{manifest_path}: schema_id debe ser {GOLDEN_CORPUS_SCHEMA}")
    if manifest.get("corpus_id") != "golden-corpus-v1":
        errors.append(f"{manifest_path}: corpus_id invalido")
    if manifest.get("training_allowed") is not False:
        errors.append(f"{manifest_path}: training_allowed debe ser false")
    if manifest.get("cases_path") != "cases.jsonl":
        errors.append(f"{manifest_path}: cases_path debe ser cases.jsonl")
    expected_hash = manifest.get("cases_sha256")
    if expected_hash != sha256_file(cases_path):
        errors.append(f"{manifest_path}: cases_sha256 no coincide")
    cases: list[dict[str, object]] = []
    try:
        for number, line in enumerate(cases_path.read_text(encoding="utf-8").splitlines(), 1):
            if not line.strip():
                errors.append(f"{cases_path}:{number}: linea vacia")
                continue
            value = json.loads(line)
            if not isinstance(value, dict):
                errors.append(f"{cases_path}:{number}: cada caso debe ser un objeto")
                continue
            cases.append(value)
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        errors.append(f"{cases_path}: JSONL invalido ({error})")
        return errors
    if manifest.get("case_count") != len(cases):
        errors.append(f"{manifest_path}: case_count no coincide")
    identifiers: list[object] = []
    observed_tags: set[str] = set()
    for index, case in enumerate(cases):
        location = f"{cases_path}:{index + 1}"
        missing = REQUIRED_CASE_FIELDS - set(case)
        if missing:
            errors.append(f"{location}: faltan campos {sorted(missing)}")
        identifiers.append(case.get("case_id"))
        if case.get("training_allowed") is not False:
            errors.append(f"{location}: training_allowed debe ser false")
        tags = case.get("tags")
        if not isinstance(tags, list) or not tags or not all(
            isinstance(tag, str) and tag for tag in tags
        ):
            errors.append(f"{location}: tags debe ser una lista no vacia")
        else:
            if tags != sorted(set(tags)):
                errors.append(f"{location}: tags debe estar ordenado y sin duplicados")
            observed_tags.update(tags)
        for field in ("expected_warnings", "expected_hard_failures"):
            value = case.get(field)
            if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
                errors.append(f"{location}: {field} debe ser una lista")
    if len(identifiers) != len(set(identifiers)) or not all(
        isinstance(identifier, str) and identifier for identifier in identifiers
    ):
        errors.append(f"{cases_path}: case_id debe ser unico y no vacio")
    manifest_tags = manifest.get("tags")
    if manifest_tags != sorted(observed_tags):
        errors.append(f"{manifest_path}: tags no coincide con los casos")
    return errors


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("corpus_dir", type=Path)
    arguments = parser.parse_args()
    validation_errors = validate_golden_corpus(arguments.corpus_dir.resolve())
    print(json.dumps({"status": "passed" if not validation_errors else "failed", "errors": validation_errors}, sort_keys=True))
    raise SystemExit(0 if not validation_errors else 1)
