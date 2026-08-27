#!/usr/bin/env python3
"""Fail-closed pre-publication validation for a staged canonical export."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from canonical_export_validator import VALIDATOR_VERSION, validate_match_export
from golden_corpus_validator import validate_golden_corpus


def classify_failure(errors: list[str]) -> str:
    semantic_markers = (
        "referencia",
        "reconciliation",
        "consistency",
        "causal",
        "futuro",
        "outcome",
        "gate duro",
    )
    return (
        "semantic_validation"
        if any(marker in error.lower() for error in errors for marker in semantic_markers)
        else "contract_validation"
    )


def validate_staged_export(
    match_dir: Path, match_id: str, checksum: str, corpus_dir: Path
) -> dict[str, object]:
    errors = validate_match_export(match_dir, match_id, checksum)
    errors.extend(validate_golden_corpus(corpus_dir))
    warnings: list[str] = []
    quality_path = match_dir / "canonical" / "diagnostics" / "quality_report.json"
    if not errors:
        try:
            quality = json.loads(quality_path.read_text(encoding="utf-8")).get("report", {})
            warnings = [
                str(domain.get("name"))
                for domain in quality.get("domains", [])
                if isinstance(domain, dict) and domain.get("status") == "warning"
            ]
        except (OSError, UnicodeError, json.JSONDecodeError, AttributeError):
            errors.append("quality report no pudo releerse para clasificar warnings")
    return {
        "status": "passed" if not errors else "failed",
        "validator_version": VALIDATOR_VERSION,
        "failure_type": None if not errors else classify_failure(errors),
        "errors": errors,
        "warnings": sorted(warnings),
    }


def default_corpus_dir() -> Path:
    return Path(__file__).resolve().parent.parent / "testdata" / "golden-corpus" / "v1"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("match_dir", type=Path)
    parser.add_argument("--match-id", required=True)
    parser.add_argument("--checksum", required=True)
    parser.add_argument("--corpus-dir", type=Path, default=default_corpus_dir())
    args = parser.parse_args()
    result = validate_staged_export(
        args.match_dir.resolve(), args.match_id, args.checksum, args.corpus_dir.resolve()
    )
    print(json.dumps(result, sort_keys=True, separators=(",", ":")))
    return 0 if result["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
