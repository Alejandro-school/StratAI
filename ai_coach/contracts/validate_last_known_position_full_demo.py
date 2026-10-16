#!/usr/bin/env python3
"""Fail-closed validator for causal last-known position and information age."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import math
import re
from collections import Counter
from collections.abc import Iterable, Iterator
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator
from jsonschema.exceptions import SchemaError

SCHEMA_ID = "stratai.observed_last_known_position@1"
REPORT_SCHEMA_ID = "stratai.observed_last_known_position_full_demo@1"
CONTRACT_VERSION = "1.0.0"
ROWS_FILENAME = "observed_last_known_position_rows.jsonl.gz"
REPORT_FILENAME = "observed_last_known_position_full_demo.json"
DIGEST_CONTRACT = "sha256_compressed_and_decompressed_jsonl_v1"
CONTACT_SOURCE = "stratai.observed_visual_contact@1"
POSITION_SOURCE = "confirmed_visual_contact_subject_pawn_origin"
AGE_SOURCE = "tick_delta_from_confirmed_visual_contact"
UNAVAILABLE = "unavailable"
ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
PARTICIPANT_PATTERN = re.compile(r"^participant-[0-9]{3,}$")
ROW_PATTERN = re.compile(r"^last-known-[0-9]{9}$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
RAW_STEAM_PATTERN = re.compile(r"(?<![0-9])[0-9]{17}(?![0-9])")
WINDOWS_ABSOLUTE_PATTERN = re.compile(r"^[A-Za-z]:[\\/]")
MAX_JSONL_LINE_BYTES = 1_048_576
FORBIDDEN_KEYS = {
    "name",
    "steam_id",
    "observer_id",
    "subject_id",
    "entity_id",
    "serial",
    "oracle",
    "outcomes",
    "future_position",
    "current_subject_position",
}
INPUT_FIELDS = [
    "replay.subject_pawn_origin",
    "visual_contact.availability_tick",
    "visual_contact.provenance.source_frame_tick",
    "visual_contact.status",
    "visual_contact.value",
]


class LastKnownPositionValidationError(ValueError):
    """Raised when a candidate artifact fails closed."""


def _reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise LastKnownPositionValidationError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def _load_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(
            path.read_text(encoding="utf-8"),
            object_pairs_hook=_reject_duplicate_keys,
        )
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise LastKnownPositionValidationError(
            f"cannot read JSON {path.name}: {exc}"
        ) from exc
    if not isinstance(value, dict):
        raise LastKnownPositionValidationError("JSON root must be an object")
    return value


def _exact_keys(value: dict[str, Any], expected: set[str], context: str) -> None:
    actual = set(value)
    if actual != expected:
        raise LastKnownPositionValidationError(
            f"{context} keys mismatch: missing={sorted(expected - actual)} "
            f"extra={sorted(actual - expected)}"
        )


def _strict_int(value: Any, context: str, *, positive: bool = False) -> int:
    if isinstance(value, bool) or not isinstance(value, int):
        raise LastKnownPositionValidationError(f"{context} must be an integer")
    if value < (1 if positive else 0):
        raise LastKnownPositionValidationError(f"{context} is out of range")
    return value


def _finite_number(value: Any, context: str) -> float:
    if (
        isinstance(value, bool)
        or not isinstance(value, (int, float))
        or not math.isfinite(value)
    ):
        raise LastKnownPositionValidationError(f"{context} must be finite")
    return float(value)


def _vector(value: Any, context: str) -> tuple[float, float, float]:
    if not isinstance(value, dict) or set(value) != {"x", "y", "z"}:
        raise LastKnownPositionValidationError(
            f"{context} must be a complete XYZ vector"
        )
    return tuple(  # type: ignore[return-value]
        _finite_number(value[axis], f"{context}.{axis}")
        for axis in ("x", "y", "z")
    )


def _strict_equal(left: Any, right: Any) -> bool:
    if (
        isinstance(left, (int, float))
        and not isinstance(left, bool)
        and isinstance(right, (int, float))
        and not isinstance(right, bool)
    ):
        return left == right
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        return set(left) == set(right) and all(
            _strict_equal(left[key], right[key]) for key in left
        )
    if isinstance(left, list):
        return len(left) == len(right) and all(
            _strict_equal(a, b) for a, b in zip(left, right, strict=True)
        )
    return left == right


def _walk_privacy(value: Any, context: str = "row") -> None:
    if isinstance(value, dict):
        for key, child in value.items():
            if key.casefold() in FORBIDDEN_KEYS:
                raise LastKnownPositionValidationError(
                    f"privacy violation: forbidden key {context}.{key}"
                )
            _walk_privacy(child, f"{context}.{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            _walk_privacy(child, f"{context}[{index}]")
    elif isinstance(value, str):
        if RAW_STEAM_PATTERN.search(value):
            raise LastKnownPositionValidationError(
                f"privacy violation: raw Steam-like identifier at {context}"
            )
        if value.startswith(("/", "\\\\")) or WINDOWS_ABSOLUTE_PATTERN.match(value):
            raise LastKnownPositionValidationError(
                f"privacy violation: absolute path at {context}"
            )


def _load_row_validator(schema_path: Path) -> Draft202012Validator:
    schema = _load_json(schema_path)
    if (
        schema.get("$id") != SCHEMA_ID
        or schema.get("contract_version") != CONTRACT_VERSION
        or not isinstance(schema.get("$defs", {}).get("row"), dict)
    ):
        raise LastKnownPositionValidationError("candidate schema metadata mismatch")
    row_schema = {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$defs": schema["$defs"],
        "$ref": "#/$defs/row",
    }
    try:
        Draft202012Validator.check_schema(schema)
        Draft202012Validator.check_schema(row_schema)
    except SchemaError as exc:
        raise LastKnownPositionValidationError(
            f"candidate schema is invalid: {exc.message}"
        ) from exc
    return Draft202012Validator(row_schema)


def _schema_validate_row(
    row: dict[str, Any], validator: Draft202012Validator
) -> None:
    errors = sorted(
        validator.iter_errors(row),
        key=lambda error: ([str(item) for item in error.absolute_path], error.message),
    )
    if errors:
        error = errors[0]
        location = ".".join(str(item) for item in error.absolute_path)
        raise LastKnownPositionValidationError(
            f"schema violation at {location or 'row'}: {error.message}"
        )


def _nearest_rank(values: list[float], quantile: float) -> float:
    index = max(0, min(len(values) - 1, math.ceil(len(values) * quantile) - 1))
    return values[index]


def _age_distribution(values: list[float]) -> dict[str, Any]:
    buckets = Counter(
        {
            "0": 0,
            "gt_0_le_250": 0,
            "gt_250_le_1000": 0,
            "gt_1000_le_5000": 0,
            "gt_5000": 0,
        }
    )
    for value in values:
        if value == 0:
            buckets["0"] += 1
        elif value <= 250:
            buckets["gt_0_le_250"] += 1
        elif value <= 1000:
            buckets["gt_250_le_1000"] += 1
        elif value <= 5000:
            buckets["gt_1000_le_5000"] += 1
        else:
            buckets["gt_5000"] += 1
    sorted_values = sorted(values)
    return {
        "observed_count": len(values),
        "zero_count": buckets["0"],
        "positive_count": len(values) - buckets["0"],
        "minimum_ms": sorted_values[0] if sorted_values else None,
        "p50_ms": _nearest_rank(sorted_values, 0.50) if sorted_values else None,
        "p95_ms": _nearest_rank(sorted_values, 0.95) if sorted_values else None,
        "maximum_ms": sorted_values[-1] if sorted_values else None,
        "buckets": dict(buckets),
    }


def _load_rows(
    rows_path: Path,
    validator: Draft202012Validator,
    tick_rate_hz: float,
) -> tuple[int, dict[str, Any], dict[str, Any]]:
    physical_hash = hashlib.sha256()
    logical_hash = hashlib.sha256()
    logical_bytes = 0

    def iter_rows() -> Iterator[dict[str, Any]]:
        nonlocal logical_bytes
        with gzip.open(rows_path, mode="rb") as stream:
            for line_number, line in enumerate(stream, start=1):
                if len(line) > MAX_JSONL_LINE_BYTES:
                    raise LastKnownPositionValidationError(
                        f"row {line_number} exceeds the line limit"
                    )
                logical_hash.update(line)
                logical_bytes += len(line)
                try:
                    row = json.loads(
                        line.decode("utf-8"),
                        object_pairs_hook=_reject_duplicate_keys,
                    )
                except (UnicodeDecodeError, json.JSONDecodeError) as exc:
                    raise LastKnownPositionValidationError(
                        f"row {line_number} is invalid JSON: {exc}"
                    ) from exc
                if not isinstance(row, dict):
                    raise LastKnownPositionValidationError(
                        f"row {line_number} must be an object"
                    )
                _schema_validate_row(row, validator)
                _walk_privacy(row)
                yield row

    try:
        with rows_path.open("rb") as raw:
            while chunk := raw.read(1024 * 1024):
                physical_hash.update(chunk)
        coverage, summary = _recompute_rows(iter_rows(), tick_rate_hz)
    except (OSError, EOFError) as exc:
        raise LastKnownPositionValidationError(
            f"cannot read rows artifact: {exc}"
        ) from exc

    row_count = coverage["row_count"]
    if row_count == 0:
        raise LastKnownPositionValidationError("rows artifact is empty")
    hashes = {
        "bytes": rows_path.stat().st_size,
        "sha256": physical_hash.hexdigest(),
        "logical_bytes": logical_bytes,
        "logical_sha256": logical_hash.hexdigest(),
    }
    return row_count, coverage, {**summary, "hashes": hashes}


def _recompute_rows(
    rows: Iterable[dict[str, Any]], tick_rate_hz: float
) -> tuple[dict[str, Any], dict[str, Any]]:
    rounds: set[int] = set()
    participants: set[str] = set()
    directed_pairs: set[tuple[str, str]] = set()
    statuses: Counter[str] = Counter()
    derivations: Counter[str] = Counter()
    contacts: Counter[str] = Counter()
    carry_sources: Counter[str] = Counter(
        {"negative_contact": 0, "unavailable_contact": 0}
    )
    abstentions: Counter[str] = Counter()
    ages: list[float] = []
    states: dict[
        tuple[int, str, str],
        tuple[dict[str, Any], int, int, tuple[int, int]],
    ] = {}
    last_life: dict[tuple[int, str, str], tuple[int, int]] = {}
    update_seen: set[tuple[int, str, str]] = set()
    life_resets = 0
    refreshes = 0
    row_count = 0
    previous_sort_key: tuple[int, int, str, str] | None = None

    for index, row in enumerate(rows, start=1):
        row_count = index
        if row["row_id"] != f"last-known-{index:09d}" or not ROW_PATTERN.fullmatch(
            row["row_id"]
        ):
            raise LastKnownPositionValidationError("row IDs are not deterministic")
        round_number = _strict_int(row["round_number"], "round_number", positive=True)
        t0_tick = _strict_int(row["t0_tick"], "t0_tick")
        joins = row["join_keys"]
        source_alias = joins["source_alias"]
        observer = joins["observer_alias"]
        subject = joins["subject_alias"]
        if (
            not ALIAS_PATTERN.fullmatch(source_alias)
            or not PARTICIPANT_PATTERN.fullmatch(observer)
            or not PARTICIPANT_PATTERN.fullmatch(subject)
            or observer == subject
            or joins["round_id"] != f"{source_alias}:round:{round_number:03d}"
        ):
            raise LastKnownPositionValidationError("opaque join mismatch")
        sort_key = (round_number, t0_tick, observer, subject)
        if previous_sort_key is not None and sort_key <= previous_sort_key:
            raise LastKnownPositionValidationError(
                "rows are duplicated or not causally ordered"
            )
        previous_sort_key = sort_key
        rounds.add(round_number)
        participants.update((observer, subject))
        directed_pairs.add((observer, subject))
        pair = (round_number, observer, subject)

        contact = row["contact_observation"]
        availability_tick = _strict_int(
            contact["availability_tick"], "contact.availability_tick"
        )
        source_tick = _strict_int(
            contact["source_frame_tick"], "contact.source_frame_tick"
        )
        if availability_tick != source_tick or source_tick > t0_tick:
            raise LastKnownPositionValidationError(
                "contact availability/source tick exceeds current T0"
            )
        if contact["source"] != CONTACT_SOURCE:
            raise LastKnownPositionValidationError("contact source mismatch")

        observer_life = contact["observer_life_sequence"]
        subject_life = contact["subject_life_sequence"]
        identity_exact = observer_life is not None and subject_life is not None
        if identity_exact:
            life = (
                _strict_int(observer_life, "observer_life_sequence", positive=True),
                _strict_int(subject_life, "subject_life_sequence", positive=True),
            )
            if pair in last_life and last_life[pair] != life:
                life_resets += 1
                states.pop(pair, None)
            last_life[pair] = life
        else:
            if observer_life is not None or subject_life is not None:
                raise LastKnownPositionValidationError("partial life identity")
            states.pop(pair, None)
            life = None

        if contact["status"] == "observed":
            if not isinstance(contact["visual_contact"], bool):
                raise LastKnownPositionValidationError(
                    "observed contact lacks a boolean value"
                )
            contact_key = "true" if contact["visual_contact"] else "false"
        elif contact["status"] == UNAVAILABLE:
            if contact["visual_contact"] is not None:
                raise LastKnownPositionValidationError(
                    "unavailable contact carries a value"
                )
            contact_key = UNAVAILABLE
        else:
            raise LastKnownPositionValidationError("unknown contact status")
        contacts[contact_key] += 1

        contact_position = contact["subject_position_at_contact"]
        if contact_key != "true" and contact_position is not None:
            raise LastKnownPositionValidationError(
                "hidden current position appears without true contact"
            )
        if contact_key == "true":
            if contact_position is not None and identity_exact:
                _vector(contact_position, "contact.subject_position_at_contact")
                states[pair] = (
                    contact_position,
                    source_tick,
                    availability_tick,
                    life,  # type: ignore[arg-type]
                )
            else:
                states.pop(pair, None)

        state = states.get(pair)
        position = row["last_known_position"]
        age = row["information_age"]
        reasons = row["abstention_reasons"]
        if reasons != sorted(set(reasons)):
            raise LastKnownPositionValidationError(
                "abstention reasons are not sorted and unique"
            )
        abstentions.update(reasons)
        statuses[position["status"]] += 1
        derivations[row["derivation"]] += 1

        if state is None:
            if (
                position["status"] != UNAVAILABLE
                or position["value"] is not None
                or position["observed_tick"] is not None
                or position["availability_tick"] is not None
                or position["source"] != UNAVAILABLE
                or age["status"] != UNAVAILABLE
                or age["value_ms"] is not None
                or age["availability_tick"] is not None
                or age["source"] != UNAVAILABLE
                or row["derivation"] != UNAVAILABLE
                or "last_known_position_unavailable" not in reasons
                or "information_age_unavailable" not in reasons
            ):
                raise LastKnownPositionValidationError(
                    "unavailable state was not reproduced"
                )
            continue

        state_position, observed_tick, original_availability, _ = state
        expected_age = (t0_tick - observed_tick) / tick_rate_hz * 1000
        if expected_age < 0 or not math.isfinite(expected_age):
            raise LastKnownPositionValidationError("recomputed age is invalid")
        actual_age = _finite_number(age["value_ms"], "information_age.value_ms")
        expected_derivation = (
            "contact_update"
            if contact_key == "true" and contact_position is not None
            else "carry_from_prior_contact"
        )
        if (
            position["status"] != "observed"
            or not _strict_equal(position["value"], state_position)
            or position["observed_tick"] != observed_tick
            or position["availability_tick"] != original_availability
            or position["source"] != POSITION_SOURCE
            or age["status"] != "observed"
            or age["availability_tick"] != t0_tick
            or age["source"] != AGE_SOURCE
            or not math.isclose(actual_age, expected_age, rel_tol=1e-12, abs_tol=1e-9)
            or row["derivation"] != expected_derivation
            or reasons
        ):
            raise LastKnownPositionValidationError(
                "position/age did not independently reproduce from prior contact"
            )
        _vector(position["value"], "last_known_position.value")
        ages.append(actual_age)
        if expected_derivation == "contact_update":
            if pair in update_seen:
                refreshes += 1
            update_seen.add(pair)
        elif contact_key == "false":
            carry_sources["negative_contact"] += 1
        elif contact_key == UNAVAILABLE:
            carry_sources["unavailable_contact"] += 1

    coverage = {
        "round_numbers": sorted(rounds),
        "round_count": len(rounds),
        "row_count": row_count,
        "participant_count": len(participants),
        "directed_pairs": len(directed_pairs),
        "contact_updates": derivations["contact_update"],
        "contact_refreshes": refreshes,
        "carries": derivations["carry_from_prior_contact"],
        "unavailable": derivations[UNAVAILABLE],
        "round_resets": len(rounds),
        "life_resets": life_resets,
    }
    summary = {
        "status": {
            "observed": statuses["observed"],
            UNAVAILABLE: statuses[UNAVAILABLE],
        },
        "derivations": {
            "contact_update": derivations["contact_update"],
            "carry_from_prior_contact": derivations["carry_from_prior_contact"],
            UNAVAILABLE: derivations[UNAVAILABLE],
        },
        "contact_observations": {
            "true": contacts["true"],
            "false": contacts["false"],
            UNAVAILABLE: contacts[UNAVAILABLE],
        },
        "carry_sources": dict(carry_sources),
        "abstention_reasons": dict(abstentions),
        "age_distribution": _age_distribution(ages),
    }
    return coverage, summary


def validate_last_known_position_full_demo_report(
    report_path: Path,
    *,
    expected_alias: str | None = None,
    expected_sha256: str | None = None,
    expected_rounds: list[int] | None = None,
    require_update: bool = False,
    require_carry: bool = False,
    require_unavailable: bool = False,
    require_zero_age: bool = False,
    require_positive_age: bool = False,
) -> dict[str, Any]:
    report_path = report_path.resolve(strict=True)
    report = _load_json(report_path)
    _exact_keys(
        report,
        {
            "schema_id",
            "contract_version",
            "candidate_schema_id",
            "source_alias",
            "source_sha256",
            "build_identifier",
            "training_allowed",
            "tick_rate_hz",
            "rows_artifact",
            "coverage",
            "summary",
        },
        "report",
    )
    tick_rate = _finite_number(report["tick_rate_hz"], "report.tick_rate_hz")
    if (
        report["schema_id"] != REPORT_SCHEMA_ID
        or report["contract_version"] != CONTRACT_VERSION
        or report["candidate_schema_id"] != SCHEMA_ID
        or report["training_allowed"] is not False
        or tick_rate <= 0
        or not isinstance(report["build_identifier"], str)
        or not report["build_identifier"].strip()
        or not ALIAS_PATTERN.fullmatch(report["source_alias"])
        or not SHA256_PATTERN.fullmatch(report["source_sha256"])
    ):
        raise LastKnownPositionValidationError("report metadata mismatch")
    if expected_alias is not None and report["source_alias"] != expected_alias:
        raise LastKnownPositionValidationError("unexpected source alias")
    if expected_sha256 is not None and report["source_sha256"] != expected_sha256:
        raise LastKnownPositionValidationError("unexpected source SHA-256")

    artifact = report["rows_artifact"]
    _exact_keys(
        artifact,
        {
            "path",
            "format",
            "compression",
            "bytes",
            "sha256",
            "record_count",
            "logical_bytes",
            "logical_sha256",
            "digest_contract",
        },
        "rows_artifact",
    )
    for key in ("bytes", "record_count", "logical_bytes"):
        _strict_int(artifact[key], f"rows_artifact.{key}", positive=True)
    if (
        artifact["path"] != ROWS_FILENAME
        or Path(artifact["path"]).is_absolute()
        or Path(artifact["path"]).name != artifact["path"]
        or artifact["format"] != "jsonl"
        or artifact["compression"] != "gzip"
        or artifact["digest_contract"] != DIGEST_CONTRACT
        or not SHA256_PATTERN.fullmatch(artifact["sha256"])
        or not SHA256_PATTERN.fullmatch(artifact["logical_sha256"])
    ):
        raise LastKnownPositionValidationError("rows artifact metadata mismatch")
    rows_path = (report_path.parent / artifact["path"]).resolve(strict=True)
    if rows_path.parent != report_path.parent:
        raise LastKnownPositionValidationError("rows artifact escapes report directory")

    schema_path = Path(__file__).with_name("observed_last_known_position.schema.json")
    validator = _load_row_validator(schema_path)
    row_count, coverage, summary_with_hashes = _load_rows(
        rows_path, validator, tick_rate
    )
    hashes = summary_with_hashes.pop("hashes")
    summary = summary_with_hashes
    if row_count != artifact["record_count"] or any(
        hashes[key] != artifact[key]
        for key in ("bytes", "sha256", "logical_bytes", "logical_sha256")
    ):
        raise LastKnownPositionValidationError("physical/logical artifact hash mismatch")
    if not _strict_equal(report["coverage"], coverage):
        raise LastKnownPositionValidationError("coverage does not recompute")
    if not _strict_equal(report["summary"], summary):
        raise LastKnownPositionValidationError("summary does not recompute")
    if expected_rounds is not None and coverage["round_numbers"] != expected_rounds:
        raise LastKnownPositionValidationError("unexpected round coverage")
    requirements = {
        "contact update": (not require_update or coverage["contact_updates"] > 0),
        "carry": (not require_carry or coverage["carries"] > 0),
        "unavailable": (not require_unavailable or coverage["unavailable"] > 0),
        "zero age": (
            not require_zero_age or summary["age_distribution"]["zero_count"] > 0
        ),
        "positive age": (
            not require_positive_age
            or summary["age_distribution"]["positive_count"] > 0
        ),
    }
    missing = [name for name, present in requirements.items() if not present]
    if missing:
        raise LastKnownPositionValidationError(
            f"required real coverage is absent: {', '.join(missing)}"
        )
    return {
        "status": "passed",
        "schema_id": report["schema_id"],
        "candidate_schema_id": report["candidate_schema_id"],
        "source_alias": report["source_alias"],
        "source_sha256": report["source_sha256"],
        "tick_rate_hz": tick_rate,
        "coverage": coverage,
        "summary": summary,
        "rows_artifact": hashes,
        "causal_checks": {
            "future_source_ticks": 0,
            "hidden_current_positions": 0,
            "age_recompute_mismatches": 0,
            "carry_recompute_mismatches": 0,
            "raw_identity_hits": 0,
        },
    }


def _parse_rounds(value: str) -> list[int]:
    try:
        result = [int(item) for item in value.split(",") if item]
    except ValueError as exc:
        raise argparse.ArgumentTypeError("rounds must be comma-separated integers") from exc
    if not result or any(item <= 0 for item in result):
        raise argparse.ArgumentTypeError("rounds must be positive")
    return result


def build_argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser()
    parser.add_argument("--report", type=Path, required=True)
    parser.add_argument("--expected-alias")
    parser.add_argument("--expected-sha256")
    parser.add_argument("--expected-rounds", type=_parse_rounds)
    parser.add_argument("--require-update", action="store_true")
    parser.add_argument("--require-carry", action="store_true")
    parser.add_argument("--require-unavailable", action="store_true")
    parser.add_argument("--require-zero-age", action="store_true")
    parser.add_argument("--require-positive-age", action="store_true")
    return parser


def main() -> int:
    args = build_argument_parser().parse_args()
    try:
        result = validate_last_known_position_full_demo_report(
            args.report,
            expected_alias=args.expected_alias,
            expected_sha256=args.expected_sha256,
            expected_rounds=args.expected_rounds,
            require_update=args.require_update,
            require_carry=args.require_carry,
            require_unavailable=args.require_unavailable,
            require_zero_age=args.require_zero_age,
            require_positive_age=args.require_positive_age,
        )
    except LastKnownPositionValidationError as exc:
        print(json.dumps({"status": "failed", "error": str(exc)}, sort_keys=True))
        return 1
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
