from __future__ import annotations

import copy
import gzip
import hashlib
import json
import math
from pathlib import Path

import pytest

from ai_coach.contracts.validate_utility_initial_velocity_full_demo import (
    CONTRACT_VERSION,
    DELTA_SOURCE,
    DIGEST_CONTRACT,
    NATIVE_SOURCE,
    REPORT_SCHEMA_ID,
    ROWS_FILENAME,
    SCHEMA_ID,
    UtilityVelocityValidationError,
    _load_row_validator,
    _summary_for_rows,
    main,
    validate_utility_initial_velocity_full_demo_report,
    validate_utility_initial_velocity_row,
)

SOURCE_SHA256 = "a" * 64
SOURCE_ALIAS = f"demo-{SOURCE_SHA256[:20]}"
SCHEMA_PATH = (
    Path(__file__).parents[1]
    / "contracts"
    / "observed_utility_initial_velocity.schema.json"
)


def _native_row() -> dict[str, object]:
    return {
        "row_id": "utility-velocity-000001",
        "round_number": 1,
        "join_keys": {
            "throw_id": "utility-throw-000001",
            "projectile_id": "projectile-000001",
            "player_id": "participant-001",
        },
        "utility_type": {
            "status": "observed",
            "value": "smoke",
            "source": "weapon_instance",
        },
        "launch": {"status": "observed", "tick": 100, "source": "projectile_throw"},
        "initial_velocity": {
            "status": "observed",
            "vector": {"x": 0.0, "y": 700.0, "z": 180.0},
            "horizontal_speed": 700.0,
            "units": "source2_world_units_per_second",
            "coordinate_frame": "source2_world_xyz",
            "observed_tick": 100,
            "availability_tick": 100,
            "measurement_window": {
                "method": "native_entity_property_at_projectile_throw",
                "start_tick": 100,
                "end_tick": 100,
                "tick_count": 0,
                "tick_rate_hz": None,
                "start_position": None,
                "end_position": None,
                "unit": "ingame_ticks",
            },
            "source": NATIVE_SOURCE,
            "provenance": {
                "parser_event": "GrenadeProjectileThrow",
                "entity_property": "m_vecVelocity",
                "reconciliation": "exact",
            },
        },
        "abstention_reasons": [],
    }


def _delta_row() -> dict[str, object]:
    row = _native_row()
    row["row_id"] = "utility-velocity-000002"
    row["join_keys"] = {
        "throw_id": "utility-throw-000002",
        "projectile_id": "projectile-000002",
        "player_id": "participant-002",
    }
    row["launch"] = {"status": "observed", "tick": 110, "source": "projectile_throw"}
    row["utility_type"] = {
        "status": "observed",
        "value": "flashbang",
        "source": "weapon_instance",
    }
    row["initial_velocity"] = {
        "status": "observed",
        "vector": {"x": 64.0, "y": 0.0, "z": 32.0},
        "horizontal_speed": 64.0,
        "units": "source2_world_units_per_second",
        "coordinate_frame": "source2_world_xyz",
        "observed_tick": 112,
        "availability_tick": 112,
        "measurement_window": {
            "method": "two_position_delta",
            "start_tick": 110,
            "end_tick": 112,
            "tick_count": 2,
            "tick_rate_hz": 64.0,
            "start_position": {"x": 10.0, "y": 20.0, "z": 30.0},
            "end_position": {"x": 12.0, "y": 20.0, "z": 31.0},
            "unit": "ingame_ticks",
        },
        "source": DELTA_SOURCE,
        "provenance": {
            "parser_event": "GrenadeProjectileThrow+FrameDone",
            "entity_property": "projectile_position",
            "reconciliation": "exact",
        },
    }
    return row


def _unavailable_row() -> dict[str, object]:
    row = _native_row()
    row["row_id"] = "utility-velocity-000003"
    row["join_keys"] = {
        "throw_id": "utility-throw-000003",
        "projectile_id": "projectile-000003",
        "player_id": "participant-001",
    }
    row["launch"] = {"status": "observed", "tick": 120, "source": "projectile_throw"}
    row["initial_velocity"] = {
        "status": "unavailable",
        "vector": None,
        "horizontal_speed": None,
        "units": "source2_world_units_per_second",
        "coordinate_frame": "source2_world_xyz",
        "observed_tick": None,
        "availability_tick": None,
        "measurement_window": {
            "method": "unavailable",
            "start_tick": None,
            "end_tick": None,
            "tick_count": None,
            "tick_rate_hz": None,
            "start_position": None,
            "end_position": None,
            "unit": "ingame_ticks",
        },
        "source": "unavailable",
        "provenance": {
            "parser_event": "GrenadeProjectileThrow",
            "entity_property": "m_vecVelocity_or_projectile_position",
            "reconciliation": "exact",
        },
    }
    row["abstention_reasons"] = [
        "initial_velocity_unavailable",
        "insufficient_position_samples",
    ]
    return row


def _ambiguous_unavailable_row(identity_cause: str) -> dict[str, object]:
    row = _unavailable_row()
    row["initial_velocity"]["provenance"]["reconciliation"] = "ambiguous"  # type: ignore[index]
    row["abstention_reasons"] = sorted(
        ["initial_velocity_unavailable", identity_cause]
    )
    return row


def _write_report(tmp_path: Path, rows: list[dict[str, object]]) -> Path:
    logical = b"".join(
        (json.dumps(row, separators=(",", ":")) + "\n").encode("utf-8")
        for row in rows
    )
    compressed = gzip.compress(logical, mtime=0)
    rows_path = tmp_path / ROWS_FILENAME
    rows_path.write_bytes(compressed)
    coverage, summary = _summary_for_rows(rows)
    report = {
        "schema_id": REPORT_SCHEMA_ID,
        "contract_version": CONTRACT_VERSION,
        "candidate_schema_id": SCHEMA_ID,
        "source_alias": SOURCE_ALIAS,
        "source_sha256": SOURCE_SHA256,
        "build_identifier": "test-build",
        "training_allowed": False,
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
    report_path = tmp_path / "observed_utility_initial_velocity_full_demo.json"
    report_path.write_text(json.dumps(report, separators=(",", ":")), encoding="utf-8")
    return report_path


def test_native_and_delta_rows_validate_with_causal_availability() -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    native = _native_row()
    delta = _delta_row()
    validate_utility_initial_velocity_row(native, validator)
    validate_utility_initial_velocity_row(delta, validator)
    assert native["initial_velocity"]["vector"]["x"] == 0.0  # type: ignore[index]
    assert delta["initial_velocity"]["availability_tick"] == 112  # type: ignore[index]
    assert delta["initial_velocity"]["availability_tick"] != delta["launch"]["tick"]  # type: ignore[index]


@pytest.mark.parametrize("bad_value", [math.nan, math.inf, -math.inf])
def test_non_finite_values_are_rejected(bad_value: float) -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _native_row()
    row["initial_velocity"]["vector"]["y"] = bad_value  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|finite"):
        validate_utility_initial_velocity_row(row, validator)


def test_partial_sentinel_future_and_raw_identity_are_rejected() -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    partial = _native_row()
    del partial["initial_velocity"]["vector"]["z"]  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation"):
        validate_utility_initial_velocity_row(partial, validator)

    sentinel = _native_row()
    sentinel["initial_velocity"]["vector"] = {  # type: ignore[index]
        "x": 3.4028234663852886e38,
        "y": 3.4028234663852886e38,
        "z": 3.4028234663852886e38,
    }
    sentinel["initial_velocity"]["horizontal_speed"] = math.hypot(  # type: ignore[index]
        3.4028234663852886e38, 3.4028234663852886e38
    )
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|sentinel"):
        validate_utility_initial_velocity_row(sentinel, validator)

    endpoint_sentinel = _delta_row()
    endpoint_sentinel["initial_velocity"]["measurement_window"]["start_position"] = {  # type: ignore[index]
        "x": 3.4028234663852886e38,
        "y": 3.4028234663852886e38,
        "z": 3.4028234663852886e38,
    }
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|endpoint.*sentinel"):
        validate_utility_initial_velocity_row(endpoint_sentinel, validator)

    future = _delta_row()
    future["initial_velocity"]["availability_tick"] = 113  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="timing"):
        validate_utility_initial_velocity_row(future, validator)

    raw_identity = _native_row()
    raw_identity["join_keys"]["entity_id"] = 42  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation"):
        validate_utility_initial_velocity_row(raw_identity, validator)

    unknown_type = _native_row()
    unknown_type["utility_type"]["value"] = "snowball"  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="utility_type"):
        validate_utility_initial_velocity_row(unknown_type, validator)

    wrong_clock = _delta_row()
    wrong_clock["initial_velocity"]["measurement_window"]["unit"] = "demo_ticks"  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|units"):
        validate_utility_initial_velocity_row(wrong_clock, validator)


@pytest.mark.parametrize(
    ("field", "bad_value"),
    [
        ("start_tick", 100.0),
        ("end_tick", 100.0),
        ("tick_count", False),
    ],
)
def test_native_window_requires_strict_json_integers(field: str, bad_value: object) -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _native_row()
    row["initial_velocity"]["measurement_window"][field] = bad_value  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|integer"):
        validate_utility_initial_velocity_row(row, validator)


def test_delta_window_rejects_boolean_tick_count_even_when_arithmetic_matches() -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _delta_row()
    row["initial_velocity"]["observed_tick"] = 111  # type: ignore[index]
    row["initial_velocity"]["availability_tick"] = 111  # type: ignore[index]
    window = row["initial_velocity"]["measurement_window"]  # type: ignore[index]
    window["end_tick"] = 111
    window["tick_count"] = True
    window["end_position"] = {"x": 11.0, "y": 20.0, "z": 30.5}
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|integer"):
        validate_utility_initial_velocity_row(row, validator)


def test_utility_type_is_a_closed_untrimmed_enum() -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _native_row()
    row["utility_type"]["value"] = " smoke "  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|utility_type"):
        validate_utility_initial_velocity_row(row, validator)


@pytest.mark.parametrize("axis", ["x", "y", "z"])
@pytest.mark.parametrize("sign", [-1.0, 1.0])
def test_any_signed_flt_max_velocity_component_is_rejected(axis: str, sign: float) -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _native_row()
    vector = row["initial_velocity"]["vector"]  # type: ignore[index]
    vector[axis] = sign * 3.4028234663852886e38
    row["initial_velocity"]["horizontal_speed"] = math.hypot(  # type: ignore[index]
        vector["x"], vector["y"]
    )
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|sentinel"):
        validate_utility_initial_velocity_row(row, validator)


@pytest.mark.parametrize("endpoint", ["start_position", "end_position"])
@pytest.mark.parametrize("sign", [-1.0, 1.0])
def test_any_signed_flt_max_endpoint_component_is_rejected(
    endpoint: str,
    sign: float,
) -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _delta_row()
    row["initial_velocity"]["measurement_window"][endpoint]["y"] = (  # type: ignore[index]
        sign * 3.4028234663852886e38
    )
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|endpoint.*sentinel"):
        validate_utility_initial_velocity_row(row, validator)


def test_schema_discriminates_status_source_provenance_and_private_reasons() -> None:
    validator = _load_row_validator(SCHEMA_PATH)

    wrong_source_window = _native_row()
    wrong_source_window["initial_velocity"]["measurement_window"]["method"] = "two_position_delta"  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation"):
        validate_utility_initial_velocity_row(wrong_source_window, validator)

    private_provenance = _unavailable_row()
    private_provenance["initial_velocity"]["provenance"]["entity_property"] = "player_76561198000000000"  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation"):
        validate_utility_initial_velocity_row(private_provenance, validator)

    private_reason = _unavailable_row()
    private_reason["abstention_reasons"] = ["76561198000000000"]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation"):
        validate_utility_initial_velocity_row(private_reason, validator)

    fabricated = _unavailable_row()
    fabricated["initial_velocity"]["observed_tick"] = 120  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation"):
        validate_utility_initial_velocity_row(fabricated, validator)


def test_unavailable_is_null_and_does_not_infer_from_utility_type() -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _unavailable_row()
    validate_utility_initial_velocity_row(row, validator)
    assert row["utility_type"]["value"] == "smoke"  # type: ignore[index]
    assert row["initial_velocity"]["vector"] is None  # type: ignore[index]

    fabricated = copy.deepcopy(row)
    fabricated["initial_velocity"]["vector"] = {"x": 750, "y": 0, "z": 0}  # type: ignore[index]
    with pytest.raises(UtilityVelocityValidationError, match="fabricated|schema violation"):
        validate_utility_initial_velocity_row(fabricated, validator)


def test_every_unavailable_velocity_requires_the_generic_abstention() -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _unavailable_row()
    row["abstention_reasons"] = ["insufficient_position_samples"]
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|generic"):
        validate_utility_initial_velocity_row(row, validator)


@pytest.mark.parametrize(
    "identity_cause",
    [
        "projectile_observation_identity_conflict",
        "projectile_entity_reused",
    ],
)
def test_ambiguous_unavailable_requires_one_specific_identity_cause(
    identity_cause: str,
) -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    validate_utility_initial_velocity_row(
        _ambiguous_unavailable_row(identity_cause),
        validator,
    )


@pytest.mark.parametrize(
    "reasons",
    [
        ["initial_velocity_unavailable"],
        ["projectile_entity_reused"],
        ["initial_velocity_unavailable", "projectile_identity_ambiguous"],
        [
            "initial_velocity_unavailable",
            "projectile_entity_reused",
            "projectile_observation_identity_conflict",
        ],
    ],
)
def test_ambiguous_unavailable_rejects_missing_or_multiple_identity_causes(
    reasons: list[str],
) -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _ambiguous_unavailable_row("projectile_entity_reused")
    row["abstention_reasons"] = sorted(reasons)
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|ambiguous"):
        validate_utility_initial_velocity_row(row, validator)


@pytest.mark.parametrize("reconciliation", ["exact", "unavailable"])
@pytest.mark.parametrize(
    "identity_cause",
    [
        "projectile_observation_identity_conflict",
        "projectile_entity_reused",
    ],
)
def test_non_ambiguous_reconciliation_rejects_identity_ambiguity_causes(
    reconciliation: str,
    identity_cause: str,
) -> None:
    validator = _load_row_validator(SCHEMA_PATH)
    row = _unavailable_row()
    row["initial_velocity"]["provenance"]["reconciliation"] = reconciliation  # type: ignore[index]
    validate_utility_initial_velocity_row(row, validator)

    row["abstention_reasons"] = sorted(
        [*row["abstention_reasons"], identity_cause]  # type: ignore[arg-type]
    )
    with pytest.raises(UtilityVelocityValidationError, match="schema violation|ambiguous"):
        validate_utility_initial_velocity_row(row, validator)


def test_full_demo_validator_streams_and_recomputes_everything(tmp_path: Path) -> None:
    report_path = _write_report(tmp_path, [_native_row(), _delta_row(), _unavailable_row()])
    result = validate_utility_initial_velocity_full_demo_report(
        report_path,
        expected_alias=SOURCE_ALIAS,
        expected_sha256=SOURCE_SHA256,
        expected_rounds=[1],
        require_observed=True,
    )
    assert result["coverage"] == {
        "round_numbers": [1],
        "launch_count": 3,
        "row_count": 3,
        "participant_count": 2,
        "observed_count": 2,
        "unavailable_count": 1,
    }
    assert result["summary"]["component_zeros"] == {"x": 1, "y": 1, "z": 0}


def test_cli_executes_the_draft_2020_12_backed_validator(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    report_path = _write_report(tmp_path, [_native_row()])
    monkeypatch.setattr(
        "sys.argv",
        [
            "validate-utility-initial-velocity",
            "--report",
            str(report_path),
            "--expected-alias",
            SOURCE_ALIAS,
            "--expected-sha256",
            SOURCE_SHA256,
            "--expected-rounds",
            "1",
            "--require-observed",
        ],
    )
    assert main() == 0
    result = json.loads(capsys.readouterr().out)
    assert result["status"] == "passed"
    assert result["coverage"]["observed_count"] == 1


def test_full_demo_validator_rejects_hash_and_summary_tampering(tmp_path: Path) -> None:
    report_path = _write_report(tmp_path, [_native_row()])
    report = json.loads(report_path.read_text(encoding="utf-8"))
    report["summary"]["status"]["observed"] = 0
    report_path.write_text(json.dumps(report), encoding="utf-8")
    with pytest.raises(UtilityVelocityValidationError, match="summary"):
        validate_utility_initial_velocity_full_demo_report(report_path)

    report_path = _write_report(tmp_path, [_native_row()])
    rows_path = tmp_path / ROWS_FILENAME
    rows_path.write_bytes(rows_path.read_bytes() + b"tampered")
    with pytest.raises(UtilityVelocityValidationError, match="physical"):
        validate_utility_initial_velocity_full_demo_report(report_path)


def test_full_demo_metadata_and_recomputed_counts_are_type_strict(tmp_path: Path) -> None:
    report_path = _write_report(tmp_path, [_native_row()])
    report = json.loads(report_path.read_text(encoding="utf-8"))
    report["rows_artifact"]["record_count"] = True
    report_path.write_text(json.dumps(report), encoding="utf-8")
    with pytest.raises(UtilityVelocityValidationError, match="metadata"):
        validate_utility_initial_velocity_full_demo_report(report_path)

    report_path = _write_report(tmp_path, [_native_row()])
    report = json.loads(report_path.read_text(encoding="utf-8"))
    report["rows_artifact"]["bytes"] = float(report["rows_artifact"]["bytes"])
    report_path.write_text(json.dumps(report), encoding="utf-8")
    with pytest.raises(UtilityVelocityValidationError, match="metadata"):
        validate_utility_initial_velocity_full_demo_report(report_path)

    report_path = _write_report(tmp_path, [_native_row()])
    report = json.loads(report_path.read_text(encoding="utf-8"))
    report["coverage"]["observed_count"] = True
    report["summary"]["status"]["observed"] = True
    report["summary"]["component_zeros"]["x"] = True
    report_path.write_text(json.dumps(report), encoding="utf-8")
    with pytest.raises(UtilityVelocityValidationError, match="coverage|summary"):
        validate_utility_initial_velocity_full_demo_report(report_path)
