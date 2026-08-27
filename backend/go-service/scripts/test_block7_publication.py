import json
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

from golden_corpus_validator import validate_golden_corpus
from publication_validator import validate_staged_export
from test_canonical_export_validator import (
    DEMO_CHECKSUM,
    MATCH_ID,
    write_valid_bundle,
)


def test_versioned_golden_corpus_is_valid() -> None:
    corpus_dir = SCRIPT_DIR.parent / "testdata" / "golden-corpus" / "v1"
    assert validate_golden_corpus(corpus_dir) == []


def test_golden_corpus_rejects_training_use(tmp_path: Path) -> None:
    source = SCRIPT_DIR.parent / "testdata" / "golden-corpus" / "v1"
    corpus_dir = tmp_path / "v1"
    corpus_dir.mkdir()
    (corpus_dir / "manifest.json").write_bytes((source / "manifest.json").read_bytes())
    rows = (source / "cases.jsonl").read_text(encoding="utf-8").splitlines()
    case = json.loads(rows[0])
    case["training_allowed"] = True
    rows[0] = json.dumps(case, sort_keys=True, separators=(",", ":"))
    (corpus_dir / "cases.jsonl").write_text("\n".join(rows) + "\n", encoding="utf-8")

    errors = validate_golden_corpus(corpus_dir)
    assert any("training_allowed debe ser false" in error for error in errors)
    assert any("cases_sha256 no coincide" in error for error in errors)


def test_publication_validator_accepts_complete_staging(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    corpus_dir = SCRIPT_DIR.parent / "testdata" / "golden-corpus" / "v1"

    result = validate_staged_export(match_dir, MATCH_ID, DEMO_CHECKSUM, corpus_dir)

    assert result["status"] == "passed"
    assert result["failure_type"] is None


def test_publication_validator_rejects_partial_staging(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    corpus_dir = SCRIPT_DIR.parent / "testdata" / "golden-corpus" / "v1"
    (match_dir / "canonical/causal/decisions.jsonl").unlink()

    result = validate_staged_export(match_dir, MATCH_ID, DEMO_CHECKSUM, corpus_dir)

    assert result["status"] == "failed"
    assert result["failure_type"] is not None
    assert any("fichero declarado inexistente" in error for error in result["errors"])
