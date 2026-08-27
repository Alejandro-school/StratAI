import copy
import json
import sys
from pathlib import Path

import pytest

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

import golden_demo_corpus_validator as corpus_validator
import golden_demo_semantic_runner as runner
from test_golden_demo_corpus_validator import build_fixture


def fake_exporter(tmp_path: Path) -> Path:
    path = tmp_path / "canonical-export-test-double"
    path.write_text("test double", encoding="utf-8")
    return path


def passing_processor(**kwargs: object) -> dict[str, object]:
    source = kwargs["source"]
    assert isinstance(source, dict)
    return runner.passed_case(
        str(source["source_alias"]),
        ["map_geometry_loaded"],
        bool(kwargs["deterministic"]),
    )


def test_classify_export_failure_recognizes_unreconciled_playing_participant() -> None:
    attempt = runner.ExportAttempt(
        return_code=5,
        stage="export",
        error_code="canonical_export_rejected",
        stderr="canonical roster identity quality gates failed: prolonged participant",
    )

    assert runner.classify_export_failure(attempt) == (
        "roster_identity",
        "unreconciled_playing_participant",
    )


def test_runner_resumes_and_produces_the_same_manifest_as_a_clean_run(
    tmp_path: Path,
) -> None:
    interrupted_root = tmp_path / "interrupted"
    clean_root = tmp_path / "clean"
    interrupted_root.mkdir()
    clean_root.mkdir()
    interrupted_manifest, interrupted_roots, _ = build_fixture(interrupted_root)
    clean_manifest, clean_roots, _ = build_fixture(clean_root)
    exporter = fake_exporter(tmp_path)

    first = runner.run_corpus(
        interrupted_manifest,
        interrupted_roots,
        tmp_path / "work-interrupted",
        None,
        exporter=exporter,
        limit=3,
        processor=passing_processor,
    )
    assert first.status == "partial"
    assert first.processed_count == 3

    resumed = runner.run_corpus(
        interrupted_manifest,
        interrupted_roots,
        tmp_path / "work-interrupted",
        None,
        exporter=exporter,
        processor=passing_processor,
    )
    clean = runner.run_corpus(
        clean_manifest,
        clean_roots,
        tmp_path / "work-clean",
        None,
        exporter=exporter,
        processor=passing_processor,
    )
    assert resumed.status == clean.status == "passed"
    assert resumed.processed_count == 37
    assert resumed.skipped_count == 3
    assert json.loads(interrupted_manifest.read_text(encoding="utf-8")) == json.loads(
        clean_manifest.read_text(encoding="utf-8")
    )

    manifest = json.loads(interrupted_manifest.read_text(encoding="utf-8"))
    assert manifest["gate_1"] == corpus_validator.MANUAL_REVIEW_GATE_DECLARATION
    assert manifest["training_allowed"] is False
    deterministic = [
        case
        for case in manifest["semantic_validation"]["cases"]
        if "gomaxprocs_1_4_tree_hash_equality" in case["checks"]
    ]
    assert len(deterministic) == 1


def test_runner_finishes_blocked_when_a_selected_demo_is_quarantined(
    tmp_path: Path,
) -> None:
    manifest_path, roots, manifest = build_fixture(tmp_path)
    selected = runner.selected_sources(manifest["sources"])
    quarantine_alias = str(selected[1]["source_alias"])

    def processor(**kwargs: object) -> dict[str, object]:
        source = kwargs["source"]
        assert isinstance(source, dict)
        alias = str(source["source_alias"])
        if alias == quarantine_alias:
            return runner.quarantine_case(
                alias,
                ["source_identity"],
                "objective_round_reconciliation",
                "objective_quality_gate_rejected",
            )
        return passing_processor(**kwargs)

    summary = runner.run_corpus(
        manifest_path,
        roots,
        tmp_path / "work",
        None,
        exporter=fake_exporter(tmp_path),
        processor=processor,
    )
    assert summary.status == "blocked"
    assert summary.quarantined_count == 1
    saved = json.loads(manifest_path.read_text(encoding="utf-8"))
    assert saved["gate_1"] == corpus_validator.BLOCKED_GATE_DECLARATION
    assert saved["training_allowed"] is False


def test_operational_failure_leaves_the_current_case_pending(tmp_path: Path) -> None:
    manifest_path, roots, original = build_fixture(tmp_path)

    def processor(**_: object) -> dict[str, object]:
        raise runner.RunnerError("temporary exporter outage")

    with pytest.raises(runner.RunnerError, match="temporary exporter outage"):
        runner.run_corpus(
            manifest_path,
            roots,
            tmp_path / "work",
            None,
            exporter=fake_exporter(tmp_path),
            processor=processor,
        )
    saved = json.loads(manifest_path.read_text(encoding="utf-8"))
    assert saved == original


def test_runner_rejects_non_selected_alias(tmp_path: Path) -> None:
    manifest_path, roots, manifest = build_fixture(tmp_path)
    source = copy.deepcopy(manifest["sources"][0])
    source["source_alias"] = "demo-00000000000000000000"
    with pytest.raises(runner.RunnerError, match="not part of the selected corpus"):
        runner.run_corpus(
            manifest_path,
            roots,
            tmp_path / "work",
            None,
            exporter=fake_exporter(tmp_path),
            source_aliases=[str(source["source_alias"])],
            processor=passing_processor,
        )


def test_current_passed_case_requires_all_semantic_checks() -> None:
    case = runner.passed_case("demo-00000000000000000000", [], False)
    assert runner.current_case(case) is True
    case["checks"] = ["canonical_export"]
    assert runner.current_case(case) is False


def test_direct_bundle_validation_caps_retained_errors(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    def fake_validator(
        _match_dir: Path, _match_id: str, errors: list[str]
    ) -> None:
        for index in range(8):
            errors.append(f"error:{index}")

    monkeypatch.setattr(
        runner.canonical_export_validator, "MAX_RETAINED_VALIDATION_ERRORS", 3
    )
    monkeypatch.setattr(
        runner.canonical_export_validator,
        "validate_canonical_bundle",
        fake_validator,
    )

    assert runner.validate_direct_bundle(
        tmp_path / "match_test",
        "test",
        "a" * 64,
    ) == [
        "error:0",
        "error:1",
        "error:2",
        "validation: se omitieron 5 errores adicionales tras alcanzar el limite de 3",
    ]
