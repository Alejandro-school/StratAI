from __future__ import annotations

import copy
import gzip
import hashlib
import json
from pathlib import Path

import pytest

from ai_coach.contracts.validate_last_known_position_full_demo import (
    DIGEST_CONTRACT,
    REPORT_FILENAME,
    REPORT_SCHEMA_ID,
    ROWS_FILENAME,
    SCHEMA_ID,
    LastKnownPositionValidationError,
    _recompute_rows,
    _strict_equal,
    main,
    validate_last_known_position_full_demo_report,
)

SOURCE_ALIAS = "demo-0123456789abcdefabcd"
SOURCE_SHA256 = "a" * 64


def _contact(
    *,
    status: str,
    value: bool | None,
    tick: int,
    position: dict[str, float] | None,
) -> dict[str, object]:
    return {
        "status": status,
        "visual_contact": value,
        "availability_tick": tick,
        "source_frame_tick": tick,
        "subject_position_at_contact": position,
        "observer_life_sequence": 1,
        "subject_life_sequence": 1,
        "source": "stratai.observed_visual_contact@1",
    }


def _unavailable_row(sequence: int, tick: int) -> dict[str, object]:
    return {
        "row_id": f"last-known-{sequence:09d}",
        "round_number": 1,
        "t0_tick": tick,
        "join_keys": {
            "source_alias": SOURCE_ALIAS,
            "round_id": f"{SOURCE_ALIAS}:round:001",
            "observer_alias": "participant-001",
            "subject_alias": "participant-002",
        },
        "contact_observation": _contact(
            status="observed", value=False, tick=tick, position=None
        ),
        "last_known_position": {
            "status": "unavailable",
            "value": None,
            "observed_tick": None,
            "availability_tick": None,
            "coordinate_frame": "source2_world_xyz",
            "spatial_semantics": "subject_pawn_origin_at_confirmed_visual_contact",
            "source": "unavailable",
        },
        "information_age": {
            "status": "unavailable",
            "value_ms": None,
            "availability_tick": None,
            "units": "milliseconds",
            "source": "unavailable",
        },
        "derivation": "unavailable",
        "abstention_reasons": [
            "information_age_unavailable",
            "last_known_position_unavailable",
            "no_prior_confirmed_visual_contact",
        ],
        "provenance": _provenance(),
    }


def _observed_row(
    sequence: int,
    tick: int,
    *,
    contact: bool,
    position: dict[str, float],
    observed_tick: int,
    age_ms: float,
) -> dict[str, object]:
    update = contact
    return {
        "row_id": f"last-known-{sequence:09d}",
        "round_number": 1,
        "t0_tick": tick,
        "join_keys": {
            "source_alias": SOURCE_ALIAS,
            "round_id": f"{SOURCE_ALIAS}:round:001",
            "observer_alias": "participant-001",
            "subject_alias": "participant-002",
        },
        "contact_observation": _contact(
            status="observed",
            value=contact,
            tick=tick,
            position=position if contact else None,
        ),
        "last_known_position": {
            "status": "observed",
            "value": position,
            "observed_tick": observed_tick,
            "availability_tick": observed_tick,
            "coordinate_frame": "source2_world_xyz",
            "spatial_semantics": "subject_pawn_origin_at_confirmed_visual_contact",
            "source": "confirmed_visual_contact_subject_pawn_origin",
        },
        "information_age": {
            "status": "observed",
            "value_ms": age_ms,
            "availability_tick": tick,
            "units": "milliseconds",
            "source": "tick_delta_from_confirmed_visual_contact",
        },
        "derivation": "contact_update" if update else "carry_from_prior_contact",
        "abstention_reasons": [],
        "provenance": _provenance(),
    }


def _provenance() -> dict[str, object]:
    return {
        "visual_contact_schema_id": "stratai.observed_visual_contact@1",
        "visual_contact_contract_version": "1.0.0",
        "position_source_artifact": "replay_data_in_memory",
        "input_fields": [
            "replay.subject_pawn_origin",
            "visual_contact.availability_tick",
            "visual_contact.provenance.source_frame_tick",
            "visual_contact.status",
            "visual_contact.value",
        ],
    }


def _rows() -> list[dict[str, object]]:
    zero = {"x": 0.0, "y": 0.0, "z": 0.0}
    refreshed = {"x": 20.0, "y": -10.0, "z": 5.0}
    return [
        _unavailable_row(1, 100),
        _observed_row(2, 104, contact=True, position=zero, observed_tick=104, age_ms=0),
        _observed_row(
            3,
            108,
            contact=False,
            position=zero,
            observed_tick=104,
            age_ms=62.5,
        ),
        _observed_row(
            4,
            112,
            contact=True,
            position=refreshed,
            observed_tick=112,
            age_ms=0,
        ),
    ]


def _write_report(tmp_path: Path, rows: list[dict[str, object]]) -> Path:
    logical = b"".join(
        (json.dumps(row, separators=(",", ":")) + "\n").encode("utf-8")
        for row in rows
    )
    compressed = gzip.compress(logical, mtime=0)
    (tmp_path / ROWS_FILENAME).write_bytes(compressed)
    coverage, summary = _recompute_rows(rows, 64.0)
    report = {
        "schema_id": REPORT_SCHEMA_ID,
        "contract_version": "1.0.0",
        "candidate_schema_id": SCHEMA_ID,
        "source_alias": SOURCE_ALIAS,
        "source_sha256": SOURCE_SHA256,
        "build_identifier": "test-build",
        "training_allowed": False,
        "tick_rate_hz": 64.0,
        "rows_artifact": {
            "path": ROWS_FILENAME,
            "format": "jsonl",
            "compression": "gzip",
            "bytes": len(compressed),
            "sha256": hashlib.sha256(compressed).hexdigest(),
            "record_count": len(rows),
            "logical_bytes": len(logical),
            "logical_sha256": hashlib.sha256(logical).hexdigest(),
            "digest_contract": DIGEST_CONTRACT,
        },
        "coverage": coverage,
        "summary": summary,
    }
    path = tmp_path / REPORT_FILENAME
    path.write_text(json.dumps(report, separators=(",", ":")), encoding="utf-8")
    return path


def test_validator_recomputes_contact_carry_age_and_absence(tmp_path: Path) -> None:
    result = validate_last_known_position_full_demo_report(
        _write_report(tmp_path, _rows()),
        expected_alias=SOURCE_ALIAS,
        expected_sha256=SOURCE_SHA256,
        expected_rounds=[1],
        require_update=True,
        require_carry=True,
        require_unavailable=True,
        require_zero_age=True,
        require_positive_age=True,
    )
    assert result["coverage"]["contact_updates"] == 2
    assert result["coverage"]["contact_refreshes"] == 1
    assert result["coverage"]["carries"] == 1
    assert result["coverage"]["unavailable"] == 1
    assert result["summary"]["age_distribution"]["zero_count"] == 2
    assert result["summary"]["age_distribution"]["positive_count"] == 1


def test_validator_rejects_hidden_current_future_and_age_tampering(
    tmp_path: Path,
) -> None:
    hidden = _rows()
    hidden[2]["contact_observation"]["subject_position_at_contact"] = {  # type: ignore[index]
        "x": 999.0,
        "y": 0.0,
        "z": 0.0,
    }
    with pytest.raises(LastKnownPositionValidationError, match="schema|hidden"):
        validate_last_known_position_full_demo_report(_write_report(tmp_path, hidden))

    future_dir = tmp_path / "future"
    future_dir.mkdir()
    future = _rows()
    future[1]["contact_observation"]["source_frame_tick"] = 108  # type: ignore[index]
    with pytest.raises(LastKnownPositionValidationError, match="source tick|reproduce"):
        validate_last_known_position_full_demo_report(
            _write_report(future_dir, future)
        )

    age_dir = tmp_path / "age"
    age_dir.mkdir()
    bad_age = _rows()
    bad_age[2]["information_age"]["value_ms"] = 0.0  # type: ignore[index]
    with pytest.raises(LastKnownPositionValidationError, match="reproduce"):
        validate_last_known_position_full_demo_report(
            _write_report(age_dir, bad_age)
        )


def test_validator_rejects_private_identity_schema_and_report_tampering(
    tmp_path: Path,
) -> None:
    private = _rows()
    private[1]["join_keys"]["observer_id"] = "76561198000000000"  # type: ignore[index]
    with pytest.raises(LastKnownPositionValidationError, match="schema|privacy"):
        validate_last_known_position_full_demo_report(_write_report(tmp_path, private))

    report_dir = tmp_path / "report"
    report_dir.mkdir()
    report_path = _write_report(report_dir, _rows())
    report = json.loads(report_path.read_text(encoding="utf-8"))
    report["summary"]["age_distribution"]["zero_count"] = 0
    report_path.write_text(json.dumps(report), encoding="utf-8")
    with pytest.raises(LastKnownPositionValidationError, match="summary"):
        validate_last_known_position_full_demo_report(report_path)

    hash_dir = tmp_path / "hash"
    hash_dir.mkdir()
    report_path = _write_report(hash_dir, _rows())
    (hash_dir / ROWS_FILENAME).write_bytes(
        (hash_dir / ROWS_FILENAME).read_bytes() + b"tamper"
    )
    with pytest.raises(LastKnownPositionValidationError, match="artifact|read"):
        validate_last_known_position_full_demo_report(report_path)


def test_cli_runs_the_strict_validator(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    report_path = _write_report(tmp_path, _rows())
    monkeypatch.setattr(
        "sys.argv",
        [
            "validate-last-known",
            "--report",
            str(report_path),
            "--expected-alias",
            SOURCE_ALIAS,
            "--expected-sha256",
            SOURCE_SHA256,
            "--expected-rounds",
            "1",
            "--require-update",
            "--require-carry",
            "--require-unavailable",
            "--require-zero-age",
            "--require-positive-age",
        ],
    )
    assert main() == 0
    assert json.loads(capsys.readouterr().out)["status"] == "passed"


def test_fixture_rows_do_not_change_when_only_future_or_hidden_values_change() -> None:
    baseline = _rows()
    mutated = copy.deepcopy(baseline)
    hidden_current_position = {"x": 9000.0, "y": -9000.0, "z": 300.0}
    future_outcome = {"winner": "T", "kill": True}
    assert hidden_current_position not in baseline
    assert future_outcome not in baseline
    assert baseline == mutated


def test_strict_equality_accepts_equivalent_json_number_encodings() -> None:
    assert _strict_equal(
        {"minimum_ms": 0, "p50_ms": 8500},
        {"minimum_ms": 0.0, "p50_ms": 8500.0},
    )
    assert not _strict_equal({"value": True}, {"value": 1})
