#!/usr/bin/env python3
"""Fail-closed validator for the utility initial-velocity candidate artifacts."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import math
import re
from collections import Counter
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator
from jsonschema.exceptions import SchemaError

SCHEMA_ID = "stratai.observed_utility_initial_velocity@1"
REPORT_SCHEMA_ID = "stratai.observed_utility_initial_velocity_full_demo@1"
CONTRACT_VERSION = "1.0.0"
ROWS_FILENAME = "observed_utility_initial_velocity_rows.jsonl.gz"
DIGEST_CONTRACT = "sha256_compressed_and_decompressed_jsonl_v1"
NATIVE_SOURCE = "demoinfocs_grenade_projectile_m_vecVelocity_at_throw"
DELTA_SOURCE = "projectile_position_delta"
UNAVAILABLE = "unavailable"
UNITS = "source2_world_units_per_second"
FRAME = "source2_world_xyz"
WINDOW_UNIT = "ingame_ticks"
FLOAT32_MAX = 3.4028234663852886e38
UTILITY_TYPES = {"flashbang", "smoke", "he", "molotov", "incendiary", "decoy"}
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
MAX_JSONL_LINE_BYTES = 1_048_576
ROW_ID_PATTERN = re.compile(r"^utility-velocity-[0-9]{6}$")
THROW_ID_PATTERN = re.compile(r"^utility-throw-[0-9]{6}$")
PROJECTILE_ID_PATTERN = re.compile(r"^projectile-[0-9]{6}$")
PLAYER_ID_PATTERN = re.compile(r"^participant-[0-9]{3}$")
REASON_PATTERN = re.compile(r"^[a-z0-9_]+$")


class UtilityVelocityValidationError(ValueError):
    """Raised when a candidate artifact fails closed."""


def _reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise UtilityVelocityValidationError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def _load_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"), object_pairs_hook=_reject_duplicate_keys)
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise UtilityVelocityValidationError(f"cannot read report {path.name}: {exc}") from exc
    if not isinstance(value, dict):
        raise UtilityVelocityValidationError("report must be a JSON object")
    return value


def _exact_keys(value: dict[str, Any], expected: set[str], context: str) -> None:
    actual = set(value)
    if actual != expected:
        raise UtilityVelocityValidationError(
            f"{context} keys mismatch: missing={sorted(expected - actual)} extra={sorted(actual - expected)}"
        )


def _finite_number(value: Any, context: str) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value):
        raise UtilityVelocityValidationError(f"{context} must be finite")
    return float(value)


def _finite_vector(value: Any, context: str) -> tuple[float, float, float]:
    if not isinstance(value, dict) or set(value) != {"x", "y", "z"}:
        raise UtilityVelocityValidationError(
            f"schema violation: {context} must be a complete XYZ vector"
        )
    return tuple(_finite_number(value[axis], f"{context}.{axis}") for axis in ("x", "y", "z"))  # type: ignore[return-value]


def _non_negative_integer(value: Any, context: str) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or value < 0:
        raise UtilityVelocityValidationError(f"{context} must be a non-negative integer")
    return value


def _positive_integer(value: Any) -> bool:
    return not isinstance(value, bool) and isinstance(value, int) and value > 0


def _contains_float32_max(vector: tuple[float, float, float]) -> bool:
    return any(abs(component) >= FLOAT32_MAX for component in vector)


def _strict_json_equal(left: Any, right: Any) -> bool:
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        return set(left) == set(right) and all(
            _strict_json_equal(left[key], right[key]) for key in left
        )
    if isinstance(left, list):
        return len(left) == len(right) and all(
            _strict_json_equal(left_item, right_item)
            for left_item, right_item in zip(left, right, strict=True)
        )
    return left == right


def _close(left: float, right: float) -> bool:
    return math.isclose(left, right, rel_tol=1e-9, abs_tol=1e-9)


def _load_row_validator(schema_path: Path) -> Draft202012Validator:
    schema = _load_json(schema_path)
    if schema.get("$id") != SCHEMA_ID or schema.get("contract_version") != CONTRACT_VERSION:
        raise UtilityVelocityValidationError("candidate schema metadata mismatch")
    if not isinstance(schema.get("$defs", {}).get("row"), dict):
        raise UtilityVelocityValidationError("candidate schema lacks row definition")
    row_schema = {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "x-contract-schema-id": SCHEMA_ID,
        "$defs": schema["$defs"],
        "$ref": "#/$defs/row",
    }
    try:
        Draft202012Validator.check_schema(schema)
        Draft202012Validator.check_schema(row_schema)
    except SchemaError as exc:
        raise UtilityVelocityValidationError(f"candidate schema is invalid: {exc.message}") from exc
    return Draft202012Validator(row_schema)


def validate_utility_initial_velocity_row(
    row: dict[str, Any],
    validator: Draft202012Validator,
) -> None:
    if (
        not isinstance(validator, Draft202012Validator)
        or validator.schema.get("x-contract-schema-id") != SCHEMA_ID
    ):
        raise UtilityVelocityValidationError("schema violation: validator identity mismatch")
    schema_errors = sorted(
        validator.iter_errors(row),
        key=lambda error: ([str(item) for item in error.absolute_path], error.message),
    )
    if schema_errors:
        error = schema_errors[0]
        location = ".".join(str(item) for item in error.absolute_path)
        suffix = f" at {location}" if location else ""
        raise UtilityVelocityValidationError(f"schema violation{suffix}: {error.message}")
    _exact_keys(
        row,
        {
            "row_id", "round_number", "join_keys", "utility_type", "launch",
            "initial_velocity", "abstention_reasons",
        },
        "schema violation at row",
    )
    if not isinstance(row["row_id"], str) or not ROW_ID_PATTERN.fullmatch(row["row_id"]):
        raise UtilityVelocityValidationError("schema violation at row_id")
    if isinstance(row["round_number"], bool) or not isinstance(row["round_number"], int) or row["round_number"] <= 0:
        raise UtilityVelocityValidationError("schema violation at round_number")
    joins = row["join_keys"]
    if not isinstance(joins, dict):
        raise UtilityVelocityValidationError("schema violation at join_keys")
    _exact_keys(joins, {"throw_id", "projectile_id", "player_id"}, "schema violation at join_keys")
    if not isinstance(joins["throw_id"], str) or not THROW_ID_PATTERN.fullmatch(joins["throw_id"]):
        raise UtilityVelocityValidationError("schema violation at join_keys.throw_id")
    for key, pattern in (("projectile_id", PROJECTILE_ID_PATTERN), ("player_id", PLAYER_ID_PATTERN)):
        value = joins[key]
        if value is not None and (not isinstance(value, str) or not pattern.fullmatch(value)):
            raise UtilityVelocityValidationError(f"schema violation at join_keys.{key}")
    utility_type = row["utility_type"]
    launch = row["launch"]
    fact = row["initial_velocity"]
    if not isinstance(utility_type, dict) or not isinstance(launch, dict) or not isinstance(fact, dict):
        raise UtilityVelocityValidationError("schema violation at causal facts")
    _exact_keys(utility_type, {"status", "value", "source"}, "schema violation at utility_type")
    _exact_keys(launch, {"status", "tick", "source"}, "schema violation at launch")
    _exact_keys(
        fact,
        {
            "status", "vector", "horizontal_speed", "units", "coordinate_frame",
            "observed_tick", "availability_tick", "measurement_window", "source", "provenance",
        },
        "schema violation at initial_velocity",
    )
    if utility_type["status"] == "observed":
        if utility_type["value"] not in UTILITY_TYPES or utility_type["source"] != "weapon_instance":
            raise UtilityVelocityValidationError("schema violation at utility_type observed value")
    elif utility_type["status"] != UNAVAILABLE or utility_type["value"] is not None or utility_type["source"] != UNAVAILABLE:
        raise UtilityVelocityValidationError("schema violation at utility_type unavailable value")
    if launch["status"] == "observed":
        if isinstance(launch["tick"], bool) or not isinstance(launch["tick"], int) or launch["tick"] < 0 or launch["source"] != "projectile_throw":
            raise UtilityVelocityValidationError("schema violation at launch observed value")
    elif launch["status"] != UNAVAILABLE or launch["tick"] is not None or launch["source"] != UNAVAILABLE:
        raise UtilityVelocityValidationError("schema violation at launch unavailable value")
    window = fact["measurement_window"]
    provenance = fact["provenance"]
    if not isinstance(window, dict) or not isinstance(provenance, dict):
        raise UtilityVelocityValidationError("schema violation at velocity provenance")
    _exact_keys(
        window,
        {"method", "start_tick", "end_tick", "tick_count", "tick_rate_hz", "start_position", "end_position", "unit"},
        "schema violation at measurement_window",
    )
    _exact_keys(
        provenance,
        {"parser_event", "entity_property", "reconciliation"},
        "schema violation at provenance",
    )
    if fact["status"] not in {"observed", UNAVAILABLE}:
        raise UtilityVelocityValidationError("schema violation at initial_velocity.status")
    if not isinstance(provenance["parser_event"], str) or not provenance["parser_event"] or not isinstance(provenance["entity_property"], str) or not provenance["entity_property"]:
        raise UtilityVelocityValidationError("schema violation at provenance strings")
    if provenance["reconciliation"] not in {"exact", "ambiguous", UNAVAILABLE}:
        raise UtilityVelocityValidationError("schema violation at reconciliation")
    reasons = row["abstention_reasons"]
    if not isinstance(reasons, list) or any(not isinstance(reason, str) or not REASON_PATTERN.fullmatch(reason) for reason in reasons) or len(reasons) != len(set(reasons)):
        raise UtilityVelocityValidationError("schema violation at abstention_reasons")
    if fact["vector"] is not None:
        _finite_vector(fact["vector"], "initial_velocity.vector")

    if reasons != sorted(reasons):
        raise UtilityVelocityValidationError("abstention reasons must be sorted")
    if fact["units"] != UNITS or fact["coordinate_frame"] != FRAME or window["unit"] != WINDOW_UNIT:
        raise UtilityVelocityValidationError("units or coordinate frame mismatch")

    if fact["status"] == UNAVAILABLE:
        nullable = (
            fact["vector"],
            fact["horizontal_speed"],
            fact["observed_tick"],
            fact["availability_tick"],
            window["start_tick"],
            window["end_tick"],
            window["tick_count"],
            window["tick_rate_hz"],
            window["start_position"],
            window["end_position"],
        )
        if any(value is not None for value in nullable) or fact["source"] != UNAVAILABLE:
            raise UtilityVelocityValidationError("unavailable fact carries a fabricated value")
        if window["method"] != UNAVAILABLE or not reasons:
            raise UtilityVelocityValidationError("unavailable fact lacks an explicit abstention")
        if provenance["reconciliation"] not in {"exact", "ambiguous", "unavailable"}:
            raise UtilityVelocityValidationError("invalid unavailable reconciliation")
        if "initial_velocity_unavailable" not in reasons:
            raise UtilityVelocityValidationError(
                "unavailable velocity lacks its generic abstention"
            )
        identity_causes = {
            "projectile_observation_identity_conflict",
            "projectile_entity_reused",
        }.intersection(reasons)
        if provenance["reconciliation"] == "ambiguous":
            if len(identity_causes) != 1:
                raise UtilityVelocityValidationError(
                    "ambiguous velocity requires its generic abstention and exactly one identity cause"
                )
        elif identity_causes:
            raise UtilityVelocityValidationError(
                "identity ambiguity abstentions require ambiguous reconciliation"
            )
        return

    if reasons:
        raise UtilityVelocityValidationError("observed fact cannot carry abstentions")
    if launch["status"] != "observed" or row["utility_type"]["status"] != "observed":
        raise UtilityVelocityValidationError("observed velocity lacks causal launch/type identity")
    if row["join_keys"]["projectile_id"] is None or row["join_keys"]["player_id"] is None:
        raise UtilityVelocityValidationError("observed velocity lacks opaque identity joins")
    if provenance["reconciliation"] != "exact":
        raise UtilityVelocityValidationError("observed velocity reconciliation is not exact")
    vector = _finite_vector(fact["vector"], "initial_velocity.vector")
    if vector == (0.0, 0.0, 0.0) or _contains_float32_max(vector):
        raise UtilityVelocityValidationError("observed velocity is a zero/default or FLT_MAX sentinel")
    horizontal = _finite_number(fact["horizontal_speed"], "initial_velocity.horizontal_speed")
    if horizontal < 0 or not _close(horizontal, math.hypot(vector[0], vector[1])):
        raise UtilityVelocityValidationError("horizontal speed does not match XYZ")
    observed_tick = fact["observed_tick"]
    availability_tick = fact["availability_tick"]
    launch_tick = launch["tick"]
    observed_tick = _non_negative_integer(observed_tick, "initial_velocity.observed_tick")
    availability_tick = _non_negative_integer(
        availability_tick, "initial_velocity.availability_tick"
    )
    launch_tick = _non_negative_integer(launch_tick, "launch.tick")
    if availability_tick < observed_tick:
        raise UtilityVelocityValidationError("availability precedes observation")

    if fact["source"] == NATIVE_SOURCE:
        start_tick = _non_negative_integer(
            window["start_tick"], "measurement_window.start_tick"
        )
        end_tick = _non_negative_integer(
            window["end_tick"], "measurement_window.end_tick"
        )
        tick_count = _non_negative_integer(
            window["tick_count"], "measurement_window.tick_count"
        )
        if (
            window["method"] != "native_entity_property_at_projectile_throw"
            or start_tick != launch_tick
            or end_tick != launch_tick
            or tick_count != 0
            or window["tick_rate_hz"] is not None
            or window["start_position"] is not None
            or window["end_position"] is not None
            or observed_tick != launch_tick
            or availability_tick != launch_tick
            or provenance["parser_event"] != "GrenadeProjectileThrow"
            or provenance["entity_property"] != "m_vecVelocity"
        ):
            raise UtilityVelocityValidationError("native measurement timing/provenance mismatch")
        return

    if fact["source"] != DELTA_SOURCE:
        raise UtilityVelocityValidationError("unknown observed velocity source")
    start_tick = window["start_tick"]
    end_tick = window["end_tick"]
    tick_count = window["tick_count"]
    start_tick = _non_negative_integer(start_tick, "measurement_window.start_tick")
    end_tick = _non_negative_integer(end_tick, "measurement_window.end_tick")
    tick_count = _non_negative_integer(tick_count, "measurement_window.tick_count")
    tick_rate = _finite_number(window["tick_rate_hz"], "measurement_window.tick_rate_hz")
    if (
        window["method"] != "two_position_delta"
        or start_tick != launch_tick
        or tick_count <= 0
        or end_tick - start_tick != tick_count
        or observed_tick != end_tick
        or availability_tick != end_tick
        or tick_rate <= 0
        or provenance["parser_event"] != "GrenadeProjectileThrow+FrameDone"
        or provenance["entity_property"] != "projectile_position"
    ):
        raise UtilityVelocityValidationError("position-delta timing/provenance mismatch")
    start = _finite_vector(window["start_position"], "measurement_window.start_position")
    end = _finite_vector(window["end_position"], "measurement_window.end_position")
    if _contains_float32_max(start) or _contains_float32_max(end):
        raise UtilityVelocityValidationError("position-delta endpoint is an FLT_MAX sentinel")
    recomputed = tuple((end[index] - start[index]) * tick_rate / tick_count for index in range(3))
    if not all(_close(recomputed[index], vector[index]) for index in range(3)):
        raise UtilityVelocityValidationError("position-delta vector was not independently reproduced")


def _summary_for_rows(rows: list[dict[str, Any]]) -> tuple[dict[str, Any], dict[str, Any]]:
    rounds: set[int] = set()
    players: set[str] = set()
    statuses: Counter[str] = Counter()
    sources: Counter[str] = Counter()
    methods: Counter[str] = Counter()
    utility_types: Counter[str] = Counter()
    abstentions: Counter[str] = Counter()
    zeros = Counter({"x": 0, "y": 0, "z": 0})
    for row in rows:
        rounds.add(row["round_number"])
        player_id = row["join_keys"]["player_id"]
        if player_id is not None:
            players.add(player_id)
        fact = row["initial_velocity"]
        statuses[fact["status"]] += 1
        sources[fact["source"]] += 1
        methods[fact["measurement_window"]["method"]] += 1
        utility_types[row["utility_type"]["value"] or UNAVAILABLE] += 1
        abstentions.update(row["abstention_reasons"])
        if fact["vector"] is not None:
            for axis in ("x", "y", "z"):
                if fact["vector"][axis] == 0:
                    zeros[axis] += 1
    coverage = {
        "round_numbers": sorted(rounds),
        "launch_count": len(rows),
        "row_count": len(rows),
        "participant_count": len(players),
        "observed_count": statuses["observed"],
        "unavailable_count": statuses[UNAVAILABLE],
    }
    summary = {
        "status": {"observed": statuses["observed"], UNAVAILABLE: statuses[UNAVAILABLE]},
        "sources": dict(sources),
        "measurement_methods": dict(methods),
        "utility_types": dict(utility_types),
        "abstention_reasons": dict(abstentions),
        "component_zeros": dict(zeros),
    }
    return coverage, summary


def validate_utility_initial_velocity_full_demo_report(
    report_path: Path,
    *,
    expected_alias: str | None = None,
    expected_sha256: str | None = None,
    expected_rounds: list[int] | None = None,
    require_observed: bool = False,
) -> dict[str, Any]:
    report_path = report_path.resolve(strict=True)
    report = _load_json(report_path)
    _exact_keys(
        report,
        {
            "schema_id", "contract_version", "candidate_schema_id", "source_alias", "source_sha256",
            "build_identifier", "training_allowed", "rows_artifact", "coverage", "summary",
        },
        "report",
    )
    if (
        report["schema_id"] != REPORT_SCHEMA_ID
        or report["contract_version"] != CONTRACT_VERSION
        or report["candidate_schema_id"] != SCHEMA_ID
        or report["training_allowed"] is not False
        or not isinstance(report["build_identifier"], str)
        or not report["build_identifier"].strip()
        or not ALIAS_PATTERN.fullmatch(report["source_alias"])
        or not SHA256_PATTERN.fullmatch(report["source_sha256"])
    ):
        raise UtilityVelocityValidationError("report metadata mismatch")
    if expected_alias is not None and report["source_alias"] != expected_alias:
        raise UtilityVelocityValidationError("source alias mismatch")
    if expected_sha256 is not None and report["source_sha256"] != expected_sha256:
        raise UtilityVelocityValidationError("source SHA-256 mismatch")

    artifact = report["rows_artifact"]
    if not isinstance(artifact, dict):
        raise UtilityVelocityValidationError("rows_artifact must be an object")
    _exact_keys(
        artifact,
        {"path", "format", "compression", "bytes", "sha256", "record_count", "logical_bytes", "logical_sha256", "digest_contract"},
        "rows_artifact",
    )
    if (
        artifact["path"] != ROWS_FILENAME
        or artifact["format"] != "jsonl"
        or artifact["compression"] != "gzip"
        or artifact["digest_contract"] != DIGEST_CONTRACT
        or not _positive_integer(artifact["bytes"])
        or not _positive_integer(artifact["record_count"])
        or not _positive_integer(artifact["logical_bytes"])
        or not isinstance(artifact["sha256"], str)
        or not SHA256_PATTERN.fullmatch(artifact["sha256"])
        or not isinstance(artifact["logical_sha256"], str)
        or not SHA256_PATTERN.fullmatch(artifact["logical_sha256"])
    ):
        raise UtilityVelocityValidationError("rows artifact metadata mismatch")
    rows_path = report_path.parent / artifact["path"]
    if rows_path.parent.resolve(strict=True) != report_path.parent or not rows_path.is_file():
        raise UtilityVelocityValidationError("rows artifact path escapes or is not a regular file")
    compressed = rows_path.read_bytes()
    if len(compressed) != artifact["bytes"] or hashlib.sha256(compressed).hexdigest() != artifact["sha256"]:
        raise UtilityVelocityValidationError("physical rows artifact digest mismatch")

    schema_path = Path(__file__).with_name("observed_utility_initial_velocity.schema.json")
    row_validator = _load_row_validator(schema_path)
    rows: list[dict[str, Any]] = []
    logical_hash = hashlib.sha256()
    logical_bytes = 0
    seen_rows: set[str] = set()
    seen_throws: set[str] = set()
    previous_key: tuple[int, int, str] | None = None
    try:
        with gzip.open(rows_path, "rb") as stream:
            for index, raw_line in enumerate(stream, start=1):
                if len(raw_line) > MAX_JSONL_LINE_BYTES or not raw_line.endswith(b"\n") or not raw_line.strip():
                    raise UtilityVelocityValidationError(f"invalid JSONL framing at row {index}")
                logical_hash.update(raw_line)
                logical_bytes += len(raw_line)
                try:
                    row = json.loads(raw_line, object_pairs_hook=_reject_duplicate_keys)
                except (UnicodeDecodeError, json.JSONDecodeError) as exc:
                    raise UtilityVelocityValidationError(f"invalid JSON row {index}: {exc}") from exc
                if not isinstance(row, dict):
                    raise UtilityVelocityValidationError(f"row {index} is not an object")
                validate_utility_initial_velocity_row(row, row_validator)
                row_id = row["row_id"]
                throw_id = row["join_keys"]["throw_id"]
                if row_id in seen_rows or throw_id in seen_throws:
                    raise UtilityVelocityValidationError("duplicate row or throw join identity")
                seen_rows.add(row_id)
                seen_throws.add(throw_id)
                launch_tick = row["launch"]["tick"]
                key = (row["round_number"], launch_tick if launch_tick is not None else 2**63 - 1, row_id)
                if previous_key is not None and key <= previous_key:
                    raise UtilityVelocityValidationError("rows are not in deterministic order")
                previous_key = key
                rows.append(row)
    except (OSError, EOFError) as exc:
        raise UtilityVelocityValidationError(f"cannot stream gzip rows: {exc}") from exc

    if (
        len(rows) != artifact["record_count"]
        or logical_bytes != artifact["logical_bytes"]
        or logical_hash.hexdigest() != artifact["logical_sha256"]
    ):
        raise UtilityVelocityValidationError("logical rows artifact digest/count mismatch")
    coverage, summary = _summary_for_rows(rows)
    if not _strict_json_equal(report["coverage"], coverage) or not _strict_json_equal(
        report["summary"], summary
    ):
        raise UtilityVelocityValidationError("coverage or summary was not independently reproduced")
    if expected_rounds is not None and coverage["round_numbers"] != expected_rounds:
        raise UtilityVelocityValidationError("eligible round coverage mismatch")
    if require_observed and coverage["observed_count"] <= 0:
        raise UtilityVelocityValidationError("real diversity gate requires at least one observed velocity")
    return {
        "status": "passed",
        "schema_id": REPORT_SCHEMA_ID,
        "source_alias": report["source_alias"],
        "source_sha256": report["source_sha256"],
        "coverage": coverage,
        "summary": summary,
        "rows_artifact": artifact,
    }


def _parse_rounds(value: str) -> list[int]:
    try:
        rounds = [int(item) for item in value.split(",") if item]
    except ValueError as exc:
        raise argparse.ArgumentTypeError("rounds must be comma-separated integers") from exc
    if not rounds or rounds != sorted(set(rounds)) or any(round_number <= 0 for round_number in rounds):
        raise argparse.ArgumentTypeError("rounds must be unique, sorted, and positive")
    return rounds


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--expected-alias")
    parser.add_argument("--expected-sha256")
    parser.add_argument("--expected-rounds", type=_parse_rounds)
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    try:
        result = validate_utility_initial_velocity_full_demo_report(
            args.report,
            expected_alias=args.expected_alias,
            expected_sha256=args.expected_sha256,
            expected_rounds=args.expected_rounds,
            require_observed=args.require_observed,
        )
    except UtilityVelocityValidationError as exc:
        print(json.dumps({"status": "failed", "error": str(exc)}, sort_keys=True))
        return 1
    print(json.dumps(result, sort_keys=True, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
