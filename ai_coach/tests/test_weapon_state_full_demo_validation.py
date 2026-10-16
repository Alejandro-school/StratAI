from __future__ import annotations

import gzip
import hashlib
import json
import math
import sys
from collections import Counter
from pathlib import Path

import pytest

from ai_coach.contracts.runtime_contracts import (
    OBSERVED_WEAPON_STATE_CONTRACT_VERSION,
    OBSERVED_WEAPON_STATE_SCHEMA_ID,
    is_non_ammunition_weapon,
)
from ai_coach.contracts.validate_weapon_state_full_demo import (
    DIGEST_CONTRACT,
    FACT_FIELDS,
    FACT_STATUSES,
    RECONCILIATIONS,
    REPORT_FILENAME,
    REPORT_SCHEMA_ID,
    ROWS_FILENAME,
    WeaponStateFullDemoValidationError,
    main,
    validate_weapon_state_full_demo_report,
)

SOURCE_SHA256 = "a" * 64
SOURCE_ALIAS = f"demo-{SOURCE_SHA256[:20]}"


def _observed(
    value: object,
    source: str,
    tick: int,
    reconciliation: str = "not_applicable",
) -> dict[str, object]:
    return {
        "status": "observed",
        "value": value,
        "availability_tick": tick,
        "provenance": {
            "source": source,
            "source_tick": tick,
            "reconciliation": reconciliation,
        },
    }


def _unavailable(reconciliation: str = "unavailable") -> dict[str, object]:
    return {
        "status": "unavailable",
        "value": None,
        "availability_tick": None,
        "provenance": {
            "source": "unavailable",
            "source_tick": None,
            "reconciliation": reconciliation,
        },
    }


def _row(
    tick: int,
    *,
    weapon: str | None,
    magazine: int | None,
    reserve: int | None,
    reload: bool | None,
    reconciliation: str = "not_applicable",
    reasons: list[str] | None = None,
) -> dict[str, object]:
    player_id = "participant-001"
    round_id = f"{SOURCE_ALIAS}:round:001"
    active = (
        _observed(
            weapon,
            "replay_active_weapon_snapshot",
            tick,
            reconciliation,
        )
        if weapon is not None
        else _unavailable()
    )
    ammo_reconciliation = reconciliation
    reload_reconciliation = (
        reconciliation if reconciliation != "confirmed_weapon_fire" else "not_applicable"
    )
    return {
        "schema_id": OBSERVED_WEAPON_STATE_SCHEMA_ID,
        "contract_version": OBSERVED_WEAPON_STATE_CONTRACT_VERSION,
        "identity_semantics": "join_only",
        "causal_role": "model_input_observation",
        "join_keys": {
            "match_id": SOURCE_ALIAS,
            "round_id": round_id,
            "player_id": player_id,
            "observation_id": f"{round_id}:weapon-state:{tick}:{player_id}",
        },
        "round_number": 1,
        "tick": tick,
        "t0_tick": tick,
        "active_weapon": active,
        "ammo_in_magazine": (
            _observed(
                magazine,
                "demoinfocs_active_weapon_ammo",
                tick,
                ammo_reconciliation,
            )
            if magazine is not None
            else _unavailable(
                "not_applicable" if weapon and is_non_ammunition_weapon(weapon) else "unavailable"
            )
        ),
        "ammo_reserve": (
            _observed(
                reserve,
                "demoinfocs_active_weapon_ammo",
                tick,
                ammo_reconciliation,
            )
            if reserve is not None
            else _unavailable(
                "not_applicable" if weapon and is_non_ammunition_weapon(weapon) else "unavailable"
            )
        ),
        "is_reloading": (
            _observed(
                reload,
                "demoinfocs_causal_reload_tracker",
                tick,
                reload_reconciliation,
            )
            if reload is not None
            else _unavailable(
                "not_applicable" if weapon and is_non_ammunition_weapon(weapon) else "unavailable"
            )
        ),
        "abstention_reasons": reasons or [],
    }


def _diverse_rows() -> list[dict[str, object]]:
    rows = [
        _row(100, weapon="AK-47", magazine=30, reserve=90, reload=False),
        _row(
            104,
            weapon="AK-47",
            magazine=0,
            reserve=89,
            reload=False,
            reconciliation="confirmed_weapon_fire",
        ),
        _row(
            108,
            weapon="AK-47",
            magazine=30,
            reserve=0,
            reload=True,
            reconciliation="confirmed_weapon_reload",
        ),
        _row(
            112,
            weapon="Knife",
            magazine=None,
            reserve=None,
            reload=None,
            reasons=["active_weapon_does_not_use_ammo"],
        ),
        _row(
            116,
            weapon="C4",
            magazine=None,
            reserve=None,
            reload=None,
            reasons=["active_weapon_does_not_use_ammo"],
        ),
        _row(
            120,
            weapon="HE Grenade",
            magazine=None,
            reserve=None,
            reload=None,
            reasons=["active_weapon_does_not_use_ammo"],
        ),
        _row(
            124,
            weapon=None,
            magazine=None,
            reserve=None,
            reload=None,
            reasons=["active_weapon_snapshot_unavailable"],
        ),
        _row(128, weapon="M4A1", magazine=20, reserve=40, reload=False),
    ]
    rows[-1]["ammo_in_magazine"] = _unavailable("ambiguous")
    rows[-1]["abstention_reasons"] = ["weapon_fire_reconciliation_ambiguous"]
    return rows


def _summary(rows: list[dict[str, object]]) -> dict[str, object]:
    fact_status = {
        field: {status: 0 for status in FACT_STATUSES} for field in FACT_FIELDS
    }
    reconciliations = {
        field: {name: 0 for name in RECONCILIATIONS} for field in FACT_FIELDS
    }
    magazine_values = {"zero": 0, "positive": 0}
    reserve_values = {"zero": 0, "positive": 0}
    reload_values = {"true": 0, "false": 0}
    active_weapons: Counter[str] = Counter()
    abstentions: Counter[str] = Counter()
    non_ammunition_rows = 0
    ambiguous_rows = 0
    for row in rows:
        ambiguous = False
        for field in FACT_FIELDS:
            fact = row[field]
            fact_status[field][fact["status"]] += 1
            reconciliation = fact["provenance"]["reconciliation"]
            reconciliations[field][reconciliation] += 1
            ambiguous = ambiguous or reconciliation == "ambiguous"
        ambiguous_rows += int(ambiguous)
        if row["active_weapon"]["status"] == "observed":
            weapon = row["active_weapon"]["value"]
            active_weapons[weapon] += 1
            non_ammunition_rows += int(is_non_ammunition_weapon(weapon))
        if row["ammo_in_magazine"]["status"] == "observed":
            key = "zero" if row["ammo_in_magazine"]["value"] == 0 else "positive"
            magazine_values[key] += 1
        if row["ammo_reserve"]["status"] == "observed":
            key = "zero" if row["ammo_reserve"]["value"] == 0 else "positive"
            reserve_values[key] += 1
        if row["is_reloading"]["status"] == "observed":
            key = "true" if row["is_reloading"]["value"] is True else "false"
            reload_values[key] += 1
        abstentions.update(row["abstention_reasons"])
    return {
        "fact_status": fact_status,
        "ammo_in_magazine_values": magazine_values,
        "ammo_reserve_values": reserve_values,
        "reload_values": reload_values,
        "active_weapons": dict(sorted(active_weapons.items())),
        "non_ammunition_rows": non_ammunition_rows,
        "ambiguous_rows": ambiguous_rows,
        "abstention_reasons": dict(sorted(abstentions.items())),
        "reconciliations": reconciliations,
    }


def _write_stream(path: Path, rows: list[dict[str, object]]) -> tuple[bytes, bytes]:
    logical = b"".join(
        json.dumps(row, sort_keys=True, separators=(",", ":")).encode("utf-8")
        + b"\n"
        for row in rows
    )
    with (
        path.open("wb") as handle,
        gzip.GzipFile(filename="", mode="wb", fileobj=handle, mtime=0) as zipped,
    ):
        zipped.write(logical)
    return logical, path.read_bytes()


def _fixture(
    root: Path,
    rows: list[dict[str, object]] | None = None,
) -> tuple[Path, dict[str, object]]:
    rows = rows or _diverse_rows()
    root.mkdir(parents=True, exist_ok=True)
    rows_path = root / ROWS_FILENAME
    logical, physical = _write_stream(rows_path, rows)
    report: dict[str, object] = {
        "schema_id": REPORT_SCHEMA_ID,
        "contract_version": OBSERVED_WEAPON_STATE_CONTRACT_VERSION,
        "candidate_schema_id": OBSERVED_WEAPON_STATE_SCHEMA_ID,
        "source_alias": SOURCE_ALIAS,
        "source_sha256": SOURCE_SHA256,
        "build_identifier": "fixture-build",
        "sampling": {
            "target_hz": 16,
            "strategy": "round_anchored_right_closed_no_carry_forward",
        },
        "rows_artifact": {
            "path": ROWS_FILENAME,
            "format": "jsonl",
            "compression": "gzip",
            "sha256": hashlib.sha256(physical).hexdigest(),
            "bytes": len(physical),
            "record_count": len(rows),
            "logical_sha256": hashlib.sha256(logical).hexdigest(),
            "logical_bytes": len(logical),
            "digest_contract": DIGEST_CONTRACT,
        },
        "coverage": {
            "rounds_eligible": [1],
            "round_numbers": [1],
            "target_sample_count": len(rows),
            "sample_count": len(rows),
            "player_count": 1,
        },
        "summary": _summary(rows),
    }
    report_path = root / REPORT_FILENAME
    report_path.write_text(
        json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )
    return report_path, report


def _rewrite_report(path: Path, report: dict[str, object]) -> None:
    path.write_text(
        json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )


def _rewrite_rows(
    report_path: Path,
    report: dict[str, object],
    rows: list[dict[str, object]],
) -> None:
    logical, physical = _write_stream(report_path.parent / ROWS_FILENAME, rows)
    descriptor = report["rows_artifact"]
    descriptor.update(
        {
            "sha256": hashlib.sha256(physical).hexdigest(),
            "bytes": len(physical),
            "record_count": len(rows),
            "logical_sha256": hashlib.sha256(logical).hexdigest(),
            "logical_bytes": len(logical),
        }
    )
    _rewrite_report(report_path, report)


def test_full_demo_validator_streams_every_row_and_recomputes_summary(
    tmp_path: Path,
) -> None:
    report_path, _ = _fixture(tmp_path)
    result = validate_weapon_state_full_demo_report(report_path)
    assert result["rows"] == 8
    assert result["coverage"] == {
        "rounds_eligible": [1],
        "round_numbers": [1],
        "target_sample_count": 8,
        "sample_count": 8,
        "player_count": 1,
    }
    assert result["summary"]["ammo_in_magazine_values"] == {
        "zero": 1,
        "positive": 2,
    }
    assert result["summary"]["ammo_reserve_values"] == {
        "zero": 1,
        "positive": 3,
    }
    assert result["summary"]["reload_values"] == {"true": 1, "false": 3}
    assert result["summary"]["non_ammunition_rows"] == 3
    assert result["summary"]["ambiguous_rows"] == 1


def test_tampered_summary_and_coverage_are_rejected(tmp_path: Path) -> None:
    report_path, report = _fixture(tmp_path)
    report["summary"]["ammo_in_magazine_values"]["zero"] += 1
    _rewrite_report(report_path, report)
    with pytest.raises(WeaponStateFullDemoValidationError, match="report.summary"):
        validate_weapon_state_full_demo_report(report_path)

    report_path, report = _fixture(tmp_path / "coverage")
    report["coverage"]["sample_count"] += 1
    _rewrite_report(report_path, report)
    with pytest.raises(WeaponStateFullDemoValidationError, match="report.coverage"):
        validate_weapon_state_full_demo_report(report_path)

    report_path, report = _fixture(tmp_path / "rounds")
    report["coverage"]["rounds_eligible"] = [1, 2]
    _rewrite_report(report_path, report)
    with pytest.raises(
        WeaponStateFullDemoValidationError, match="round_numbers must equal"
    ):
        validate_weapon_state_full_demo_report(report_path)

    report_path, report = _fixture(tmp_path / "targets")
    report["coverage"]["target_sample_count"] += 1
    _rewrite_report(report_path, report)
    with pytest.raises(
        WeaponStateFullDemoValidationError, match="eligible target sample"
    ):
        validate_weapon_state_full_demo_report(report_path)

    report_path, _ = _fixture(tmp_path / "expected-rounds")
    with pytest.raises(WeaponStateFullDemoValidationError, match="approved round"):
        validate_weapon_state_full_demo_report(
            report_path, expected_round_count=24
        )


def test_every_streamed_row_is_validated_causally(tmp_path: Path) -> None:
    rows = _diverse_rows()
    report_path, report = _fixture(tmp_path, rows)
    rows[3]["active_weapon"]["availability_tick"] = rows[3]["t0_tick"] + 1
    rows[3]["active_weapon"]["provenance"]["source_tick"] = rows[3]["t0_tick"] + 1
    _rewrite_rows(report_path, report, rows)
    with pytest.raises(WeaponStateFullDemoValidationError, match="must be <= t0_tick"):
        validate_weapon_state_full_demo_report(report_path)


@pytest.mark.parametrize("sentinel", [math.nan, math.inf, -math.inf])
def test_full_stream_rejects_non_finite_ammo(
    tmp_path: Path, sentinel: float
) -> None:
    rows = _diverse_rows()
    report_path, report = _fixture(tmp_path, rows)
    rows[0]["ammo_in_magazine"]["value"] = sentinel
    _rewrite_rows(report_path, report, rows)
    with pytest.raises(
        WeaponStateFullDemoValidationError, match="non-negative observed integer"
    ):
        validate_weapon_state_full_demo_report(report_path)


def test_artifact_hash_order_and_path_fail_closed(tmp_path: Path) -> None:
    report_path, report = _fixture(tmp_path / "hash")
    report["rows_artifact"]["sha256"] = "b" * 64
    _rewrite_report(report_path, report)
    with pytest.raises(WeaponStateFullDemoValidationError, match="digest or count"):
        validate_weapon_state_full_demo_report(report_path)

    rows = _diverse_rows()
    rows[0], rows[1] = rows[1], rows[0]
    report_path, report = _fixture(tmp_path / "order")
    _rewrite_rows(report_path, report, rows)
    with pytest.raises(WeaponStateFullDemoValidationError, match="strictly deterministic"):
        validate_weapon_state_full_demo_report(report_path)

    report_path, report = _fixture(tmp_path / "path")
    report["rows_artifact"]["path"] = "../" + ROWS_FILENAME
    _rewrite_report(report_path, report)
    with pytest.raises(WeaponStateFullDemoValidationError, match="expected"):
        validate_weapon_state_full_demo_report(report_path)


def test_diversity_gate_is_explicit(tmp_path: Path) -> None:
    rows = [_row(100, weapon="AK-47", magazine=30, reserve=90, reload=False)]
    report_path, _ = _fixture(tmp_path, rows)
    result = validate_weapon_state_full_demo_report(
        report_path, require_diversity=False
    )
    assert result["rows"] == 1
    with pytest.raises(WeaponStateFullDemoValidationError, match="diversity"):
        validate_weapon_state_full_demo_report(report_path)


def test_cli_reports_machine_readable_success(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    report_path, _ = _fixture(tmp_path)
    monkeypatch.setattr(
        sys,
        "argv",
        [
            "validate_weapon_state_full_demo",
            str(report_path),
            "--expected-source-alias",
            SOURCE_ALIAS,
            "--expected-source-sha256",
            SOURCE_SHA256,
            "--expected-rounds",
            "1",
        ],
    )
    assert main() == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["ok"] is True
    assert payload["result"]["rows"] == 8
