"""Validate a complete-demo observed weapon-state report and its gzip row stream."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import re
import stat
from collections import Counter
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from pathlib import Path
from typing import Any, BinaryIO

from ai_coach.contracts.runtime_contracts import (
    OBSERVED_WEAPON_STATE_CONTRACT_VERSION,
    OBSERVED_WEAPON_STATE_SCHEMA_ID,
    RuntimeContractError,
    is_non_ammunition_weapon,
    validate_observed_weapon_state,
)

REPORT_SCHEMA_ID = "stratai.observed_weapon_state_full_demo@1"
REPORT_FILENAME = "observed_weapon_state_full_demo.json"
ROWS_FILENAME = "observed_weapon_state_rows.jsonl.gz"
DIGEST_CONTRACT = "sha256_compressed_and_decompressed_jsonl_v1"
SOURCE_ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
MAX_JSONL_LINE_BYTES = 1024 * 1024

FACT_FIELDS = (
    "active_weapon",
    "ammo_in_magazine",
    "ammo_reserve",
    "is_reloading",
)
FACT_STATUSES = ("observed", "unavailable")
RECONCILIATIONS = (
    "not_applicable",
    "confirmed_weapon_fire",
    "confirmed_weapon_reload",
    "ambiguous",
    "unavailable",
)
REPORT_KEYS = {
    "schema_id",
    "contract_version",
    "candidate_schema_id",
    "source_alias",
    "source_sha256",
    "build_identifier",
    "sampling",
    "rows_artifact",
    "coverage",
    "summary",
}
ROWS_ARTIFACT_KEYS = {
    "path",
    "format",
    "compression",
    "sha256",
    "bytes",
    "record_count",
    "logical_sha256",
    "logical_bytes",
    "digest_contract",
}
COVERAGE_KEYS = {
    "rounds_eligible",
    "round_numbers",
    "target_sample_count",
    "sample_count",
    "player_count",
}
SUMMARY_KEYS = {
    "fact_status",
    "ammo_in_magazine_values",
    "ammo_reserve_values",
    "reload_values",
    "active_weapons",
    "non_ammunition_rows",
    "ambiguous_rows",
    "abstention_reasons",
    "reconciliations",
}


class WeaponStateFullDemoValidationError(ValueError):
    """Raised when a full-demo weapon-state diagnostic fails closed."""


def _fail(message: str) -> None:
    raise WeaponStateFullDemoValidationError(message)


def _object(value: Any, location: str) -> Mapping[str, Any]:
    if not isinstance(value, Mapping):
        _fail(f"{location}: expected an object")
    return value


def _exact_keys(value: Mapping[str, Any], expected: set[str], location: str) -> None:
    missing = sorted(expected - set(value))
    unknown = sorted(set(value) - expected)
    if missing or unknown:
        _fail(f"{location}: missing={missing}, unknown={unknown}")


def _string(value: Any, location: str) -> str:
    if not isinstance(value, str) or not value or value != value.strip():
        _fail(f"{location}: expected a canonical non-empty string")
    return value


def _integer(value: Any, location: str, *, minimum: int = 0) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or value < minimum:
        _fail(f"{location}: expected an integer >= {minimum}")
    return value


def _sha256(value: Any, location: str) -> str:
    if not isinstance(value, str) or SHA256_PATTERN.fullmatch(value) is None:
        _fail(f"{location}: expected lowercase SHA-256")
    return value


def _pairs_without_duplicates(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            _fail(f"JSON object: duplicate key {key!r}")
        result[key] = value
    return result


def _load_report(path: Path) -> Mapping[str, Any]:
    try:
        with path.open("r", encoding="utf-8") as handle:
            value = json.load(handle, object_pairs_hook=_pairs_without_duplicates)
    except WeaponStateFullDemoValidationError:
        raise
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        _fail(f"report: cannot decode JSON: {error}")
    return _object(value, "report")


def _is_symlink_or_reparse(path: Path) -> bool:
    if path.is_symlink():
        return True
    attributes = getattr(path.lstat(), "st_file_attributes", 0)
    flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    return bool(attributes & flag)


def _safe_report(path: Path) -> tuple[Path, Path]:
    logical = path.absolute()
    if logical.name != REPORT_FILENAME:
        _fail(f"report: expected filename {REPORT_FILENAME!r}")
    try:
        if _is_symlink_or_reparse(logical.parent) or _is_symlink_or_reparse(logical):
            _fail("report: symlinks, junctions and reparse points are forbidden")
        parent = logical.parent.resolve(strict=True)
        resolved = logical.resolve(strict=True)
        resolved.relative_to(parent)
    except WeaponStateFullDemoValidationError:
        raise
    except (OSError, ValueError) as error:
        _fail(f"report: cannot resolve regular file: {error}")
    if resolved.parent != parent or not resolved.is_file():
        _fail("report: expected a direct regular child of its directory")
    return resolved, parent


def _safe_rows_artifact(parent: Path, reference: Any) -> Path:
    if reference != ROWS_FILENAME:
        _fail(f"report.rows_artifact.path: expected {ROWS_FILENAME!r}")
    candidate = parent / ROWS_FILENAME
    try:
        if _is_symlink_or_reparse(candidate):
            _fail("report.rows_artifact.path: reparse points are forbidden")
        resolved = candidate.resolve(strict=True)
        resolved.relative_to(parent)
    except WeaponStateFullDemoValidationError:
        raise
    except (OSError, ValueError) as error:
        _fail(f"report.rows_artifact.path: cannot resolve sibling: {error}")
    if resolved.parent != parent or not resolved.is_file():
        _fail("report.rows_artifact.path: expected a direct regular sibling")
    return resolved


class _HashingReader:
    def __init__(self, source: BinaryIO) -> None:
        self.source = source
        self.digest = hashlib.sha256()
        self.bytes = 0

    def read(self, size: int = -1) -> bytes:
        payload = self.source.read(size)
        self.digest.update(payload)
        self.bytes += len(payload)
        return payload


@dataclass(frozen=True)
class _StreamDigest:
    compressed_bytes: int
    compressed_sha256: str
    logical_bytes: int
    logical_sha256: str
    records: int


def _consume_gzip_jsonl(
    path: Path,
    consume: Callable[[Mapping[str, Any], int], None],
) -> _StreamDigest:
    logical_hash = hashlib.sha256()
    logical_bytes = 0
    records = 0
    tracker: _HashingReader | None = None
    try:
        with path.open("rb", buffering=0) as source:
            tracker = _HashingReader(source)
            with gzip.GzipFile(fileobj=tracker, mode="rb") as decompressed:
                while True:
                    line = decompressed.readline(MAX_JSONL_LINE_BYTES + 1)
                    if not line:
                        break
                    if len(line) > MAX_JSONL_LINE_BYTES:
                        _fail(f"rows[{records}]: row exceeds safety limit")
                    if not line.endswith(b"\n"):
                        _fail(f"rows[{records}]: truncated row or missing newline")
                    if not line.strip():
                        _fail(f"rows[{records}]: blank rows are forbidden")
                    logical_hash.update(line)
                    logical_bytes += len(line)
                    try:
                        value = json.loads(
                            line.decode("utf-8"),
                            object_pairs_hook=_pairs_without_duplicates,
                        )
                    except WeaponStateFullDemoValidationError:
                        raise
                    except (UnicodeError, json.JSONDecodeError) as error:
                        _fail(f"rows[{records}]: invalid JSON: {error}")
                    consume(_object(value, f"rows[{records}]"), records)
                    records += 1
    except WeaponStateFullDemoValidationError:
        raise
    except (OSError, EOFError, gzip.BadGzipFile) as error:
        _fail(f"rows: invalid or truncated gzip stream: {error}")
    assert tracker is not None
    if tracker.bytes != path.stat().st_size:
        _fail("rows: gzip reader did not consume the complete physical artifact")
    return _StreamDigest(
        compressed_bytes=tracker.bytes,
        compressed_sha256=tracker.digest.hexdigest(),
        logical_bytes=logical_bytes,
        logical_sha256=logical_hash.hexdigest(),
        records=records,
    )


def _empty_fact_counts() -> dict[str, dict[str, int]]:
    return {
        field: {status: 0 for status in FACT_STATUSES} for field in FACT_FIELDS
    }


def _empty_reconciliation_counts() -> dict[str, dict[str, int]]:
    return {
        field: {name: 0 for name in RECONCILIATIONS} for field in FACT_FIELDS
    }


@dataclass
class _Accumulator:
    source_alias: str

    def __post_init__(self) -> None:
        self.fact_status = _empty_fact_counts()
        self.reconciliations = _empty_reconciliation_counts()
        self.magazine_values = {"zero": 0, "positive": 0}
        self.reserve_values = {"zero": 0, "positive": 0}
        self.reload_values = {"true": 0, "false": 0}
        self.active_weapons: Counter[str] = Counter()
        self.abstentions: Counter[str] = Counter()
        self.non_ammunition_rows = 0
        self.ambiguous_rows = 0
        self.round_numbers: set[int] = set()
        self.samples: set[tuple[int, int]] = set()
        self.players: set[str] = set()
        self.previous_key: tuple[int, int, str] | None = None

    def consume(self, row: Mapping[str, Any], index: int) -> None:
        try:
            validate_observed_weapon_state(row)
        except RuntimeContractError as error:
            _fail(f"rows[{index}]: {error}")
        joins = row["join_keys"]
        if joins["match_id"] != self.source_alias:
            _fail(f"rows[{index}].join_keys.match_id: source alias mismatch")
        expected_round_id = f"{self.source_alias}:round:{row['round_number']:03d}"
        if joins["round_id"] != expected_round_id:
            _fail(f"rows[{index}].join_keys.round_id: deterministic round mismatch")
        key = (int(row["round_number"]), int(row["tick"]), str(joins["player_id"]))
        if self.previous_key is not None and key <= self.previous_key:
            _fail(f"rows[{index}]: rows are not strictly deterministic")
        self.previous_key = key
        self.round_numbers.add(key[0])
        self.samples.add((key[0], key[1]))
        self.players.add(key[2])

        ambiguous = False
        for field in FACT_FIELDS:
            fact = row[field]
            status = str(fact["status"])
            reconciliation = str(fact["provenance"]["reconciliation"])
            self.fact_status[field][status] += 1
            self.reconciliations[field][reconciliation] += 1
            ambiguous = ambiguous or reconciliation == "ambiguous"
        if ambiguous:
            self.ambiguous_rows += 1

        active_weapon = row["active_weapon"]
        if active_weapon["status"] == "observed":
            weapon = str(active_weapon["value"])
            self.active_weapons[weapon] += 1
            if is_non_ammunition_weapon(weapon):
                self.non_ammunition_rows += 1

        magazine = row["ammo_in_magazine"]
        if magazine["status"] == "observed":
            bucket = "zero" if magazine["value"] == 0 else "positive"
            self.magazine_values[bucket] += 1
        reserve = row["ammo_reserve"]
        if reserve["status"] == "observed":
            bucket = "zero" if reserve["value"] == 0 else "positive"
            self.reserve_values[bucket] += 1
        reloading = row["is_reloading"]
        if reloading["status"] == "observed":
            bucket = "true" if reloading["value"] is True else "false"
            self.reload_values[bucket] += 1
        self.abstentions.update(row["abstention_reasons"])

    def coverage(self) -> dict[str, Any]:
        emitted_rounds = sorted(self.round_numbers)
        emitted_samples = len(self.samples)
        return {
            "rounds_eligible": emitted_rounds,
            "round_numbers": emitted_rounds,
            "target_sample_count": emitted_samples,
            "sample_count": emitted_samples,
            "player_count": len(self.players),
        }

    def summary(self) -> dict[str, Any]:
        return {
            "fact_status": self.fact_status,
            "ammo_in_magazine_values": self.magazine_values,
            "ammo_reserve_values": self.reserve_values,
            "reload_values": self.reload_values,
            "active_weapons": dict(sorted(self.active_weapons.items())),
            "non_ammunition_rows": self.non_ammunition_rows,
            "ambiguous_rows": self.ambiguous_rows,
            "abstention_reasons": dict(sorted(self.abstentions.items())),
            "reconciliations": self.reconciliations,
        }


def _count_map(
    value: Any,
    expected_keys: tuple[str, ...] | set[str],
    location: str,
) -> dict[str, int]:
    mapping = _object(value, location)
    _exact_keys(mapping, set(expected_keys), location)
    return {
        key: _integer(mapping[key], f"{location}.{key}") for key in expected_keys
    }


def _ordered_dynamic_counts(value: Any, location: str) -> dict[str, int]:
    mapping = _object(value, location)
    keys = list(mapping)
    if keys != sorted(keys):
        _fail(f"{location}: keys must be sorted deterministically")
    result: dict[str, int] = {}
    for key, count in mapping.items():
        canonical = _string(key, f"{location}.key")
        result[canonical] = _integer(count, f"{location}.{canonical}", minimum=1)
    return result


def _expected_coverage(value: Any) -> dict[str, Any]:
    coverage = _object(value, "report.coverage")
    _exact_keys(coverage, COVERAGE_KEYS, "report.coverage")

    def round_list(field: str) -> list[int]:
        raw = coverage[field]
        if not isinstance(raw, list):
            _fail(f"report.coverage.{field}: expected a list")
        result = [
            _integer(item, f"report.coverage.{field}[{index}]", minimum=1)
            for index, item in enumerate(raw)
        ]
        if not result or result != sorted(set(result)):
            _fail(f"report.coverage.{field}: expected non-empty sorted unique values")
        return result

    rounds_eligible = round_list("rounds_eligible")
    rounds = round_list("round_numbers")
    result = {
        "rounds_eligible": rounds_eligible,
        "round_numbers": rounds,
        "target_sample_count": _integer(
            coverage["target_sample_count"],
            "report.coverage.target_sample_count",
            minimum=1,
        ),
        "sample_count": _integer(
            coverage["sample_count"], "report.coverage.sample_count", minimum=1
        ),
        "player_count": _integer(
            coverage["player_count"], "report.coverage.player_count", minimum=1
        ),
    }
    if result["rounds_eligible"] != rounds:
        _fail("report.coverage: round_numbers must equal rounds_eligible")
    if result["target_sample_count"] != result["sample_count"]:
        _fail("report.coverage: every eligible target sample must be emitted")
    return result


def _expected_summary(value: Any) -> dict[str, Any]:
    summary = _object(value, "report.summary")
    _exact_keys(summary, SUMMARY_KEYS, "report.summary")
    fact_status = _object(summary["fact_status"], "report.summary.fact_status")
    _exact_keys(fact_status, set(FACT_FIELDS), "report.summary.fact_status")
    reconciliations = _object(
        summary["reconciliations"], "report.summary.reconciliations"
    )
    _exact_keys(
        reconciliations, set(FACT_FIELDS), "report.summary.reconciliations"
    )
    return {
        "fact_status": {
            field: _count_map(
                fact_status[field],
                FACT_STATUSES,
                f"report.summary.fact_status.{field}",
            )
            for field in FACT_FIELDS
        },
        "ammo_in_magazine_values": _count_map(
            summary["ammo_in_magazine_values"],
            {"zero", "positive"},
            "report.summary.ammo_in_magazine_values",
        ),
        "ammo_reserve_values": _count_map(
            summary["ammo_reserve_values"],
            {"zero", "positive"},
            "report.summary.ammo_reserve_values",
        ),
        "reload_values": _count_map(
            summary["reload_values"],
            {"true", "false"},
            "report.summary.reload_values",
        ),
        "active_weapons": _ordered_dynamic_counts(
            summary["active_weapons"], "report.summary.active_weapons"
        ),
        "non_ammunition_rows": _integer(
            summary["non_ammunition_rows"], "report.summary.non_ammunition_rows"
        ),
        "ambiguous_rows": _integer(
            summary["ambiguous_rows"], "report.summary.ambiguous_rows"
        ),
        "abstention_reasons": _ordered_dynamic_counts(
            summary["abstention_reasons"], "report.summary.abstention_reasons"
        ),
        "reconciliations": {
            field: _count_map(
                reconciliations[field],
                RECONCILIATIONS,
                f"report.summary.reconciliations.{field}",
            )
            for field in FACT_FIELDS
        },
    }


def _validate_diversity(summary: Mapping[str, Any]) -> None:
    required_buckets = (
        ("ammo_in_magazine_values", "zero"),
        ("ammo_in_magazine_values", "positive"),
        ("ammo_reserve_values", "zero"),
        ("ammo_reserve_values", "positive"),
        ("reload_values", "true"),
        ("reload_values", "false"),
    )
    for group, bucket in required_buckets:
        if summary[group][bucket] < 1:
            _fail(f"report.summary.{group}.{bucket}: real diversity is required")
    if summary["fact_status"]["active_weapon"]["unavailable"] < 1:
        _fail("report.summary: an explicit unavailable active weapon is required")
    if summary["non_ammunition_rows"] < 1:
        _fail("report.summary: non-ammunition equipment coverage is required")

    normalized_weapons = {
        "".join(character for character in weapon.casefold() if character.isalnum())
        for weapon in summary["active_weapons"]
    }
    required_equipment = {
        "knife": lambda name: "knife" in name or "bayonet" in name,
        "c4": lambda name: "c4" in name or "bomb" in name,
        "utility": lambda name: any(
            token in name
            for token in (
                "grenade",
                "flash",
                "smoke",
                "molotov",
                "incendiary",
                "decoy",
            )
        ),
    }
    for label, predicate in required_equipment.items():
        if not any(predicate(name) for name in normalized_weapons):
            _fail(f"report.summary.active_weapons: missing real {label} coverage")

    reconciliation_totals = Counter[str]()
    for fact_counts in summary["reconciliations"].values():
        reconciliation_totals.update(fact_counts)
    for reconciliation in (
        "not_applicable",
        "confirmed_weapon_fire",
        "confirmed_weapon_reload",
    ):
        if reconciliation_totals[reconciliation] < 1:
            _fail(
                "report.summary.reconciliations: missing real "
                f"{reconciliation} coverage"
            )


def validate_weapon_state_full_demo_report(
    report_path: Path,
    *,
    require_diversity: bool = True,
    expected_source_alias: str | None = None,
    expected_source_sha256: str | None = None,
    expected_round_count: int | None = None,
) -> dict[str, Any]:
    """Validate every full-demo row and independently recompute its report."""

    report_file, report_parent = _safe_report(report_path)
    report = _load_report(report_file)
    _exact_keys(report, REPORT_KEYS, "report")
    if report["schema_id"] != REPORT_SCHEMA_ID:
        _fail(f"report.schema_id: expected {REPORT_SCHEMA_ID}")
    if report["contract_version"] != OBSERVED_WEAPON_STATE_CONTRACT_VERSION:
        _fail(
            "report.contract_version: expected "
            f"{OBSERVED_WEAPON_STATE_CONTRACT_VERSION}"
        )
    if report["candidate_schema_id"] != OBSERVED_WEAPON_STATE_SCHEMA_ID:
        _fail(
            "report.candidate_schema_id: expected "
            f"{OBSERVED_WEAPON_STATE_SCHEMA_ID}"
        )
    checksum = _sha256(report["source_sha256"], "report.source_sha256")
    alias = report["source_alias"]
    if (
        not isinstance(alias, str)
        or SOURCE_ALIAS_PATTERN.fullmatch(alias) is None
        or alias != f"demo-{checksum[:20]}"
    ):
        _fail("report: source alias/checksum mismatch")
    if expected_source_alias is not None and alias != expected_source_alias:
        _fail("report.source_alias: differs from the approved smoke identity")
    if expected_source_sha256 is not None:
        approved_checksum = _sha256(
            expected_source_sha256, "expected_source_sha256"
        )
        if checksum != approved_checksum:
            _fail("report.source_sha256: differs from the approved smoke identity")
    _string(report["build_identifier"], "report.build_identifier")

    sampling = _object(report["sampling"], "report.sampling")
    _exact_keys(sampling, {"target_hz", "strategy"}, "report.sampling")
    if sampling != {
        "target_hz": 16,
        "strategy": "round_anchored_right_closed_no_carry_forward",
    }:
        _fail("report.sampling: invalid 16 Hz sampling contract")

    artifact = _object(report["rows_artifact"], "report.rows_artifact")
    _exact_keys(artifact, ROWS_ARTIFACT_KEYS, "report.rows_artifact")
    if artifact["format"] != "jsonl" or artifact["compression"] != "gzip":
        _fail("report.rows_artifact: expected gzip JSONL")
    if artifact["digest_contract"] != DIGEST_CONTRACT:
        _fail(f"report.rows_artifact.digest_contract: expected {DIGEST_CONTRACT}")
    expected_artifact = {
        "bytes": _integer(artifact["bytes"], "report.rows_artifact.bytes", minimum=1),
        "sha256": _sha256(artifact["sha256"], "report.rows_artifact.sha256"),
        "record_count": _integer(
            artifact["record_count"],
            "report.rows_artifact.record_count",
            minimum=1,
        ),
        "logical_bytes": _integer(
            artifact["logical_bytes"],
            "report.rows_artifact.logical_bytes",
            minimum=1,
        ),
        "logical_sha256": _sha256(
            artifact["logical_sha256"], "report.rows_artifact.logical_sha256"
        ),
    }
    rows_path = _safe_rows_artifact(report_parent, artifact["path"])
    expected_coverage = _expected_coverage(report["coverage"])
    expected_summary = _expected_summary(report["summary"])

    accumulator = _Accumulator(alias)
    digest = _consume_gzip_jsonl(rows_path, accumulator.consume)
    actual_artifact = {
        "bytes": digest.compressed_bytes,
        "sha256": digest.compressed_sha256,
        "record_count": digest.records,
        "logical_bytes": digest.logical_bytes,
        "logical_sha256": digest.logical_sha256,
    }
    if expected_artifact != actual_artifact:
        _fail("report.rows_artifact: physical/logical digest or count mismatch")
    actual_coverage = accumulator.coverage()
    if expected_coverage != actual_coverage:
        _fail("report.coverage: differs from complete streamed rows")
    if expected_round_count is not None:
        approved_round_count = _integer(
            expected_round_count, "expected_round_count", minimum=1
        )
        expected_rounds = list(range(1, approved_round_count + 1))
        if actual_coverage["round_numbers"] != expected_rounds:
            _fail(
                "report.coverage.round_numbers: does not cover every approved round"
            )
    actual_summary = accumulator.summary()
    if expected_summary != actual_summary:
        _fail("report.summary: differs from complete streamed rows")
    if require_diversity:
        _validate_diversity(actual_summary)

    return {
        "schema_id": REPORT_SCHEMA_ID,
        "candidate_schema_id": OBSERVED_WEAPON_STATE_SCHEMA_ID,
        "source_alias": alias,
        "source_sha256": checksum,
        "rows": digest.records,
        "coverage": actual_coverage,
        "summary": actual_summary,
        "rows_artifact": actual_artifact,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path)
    parser.add_argument("--expected-source-alias", required=True)
    parser.add_argument("--expected-source-sha256", required=True)
    parser.add_argument(
        "--expected-rounds",
        dest="expected_round_count",
        required=True,
        type=int,
    )
    arguments = parser.parse_args()
    try:
        result = validate_weapon_state_full_demo_report(
            arguments.report,
            expected_source_alias=arguments.expected_source_alias,
            expected_source_sha256=arguments.expected_source_sha256,
            expected_round_count=arguments.expected_round_count,
        )
    except (OSError, WeaponStateFullDemoValidationError) as error:
        print(json.dumps({"ok": False, "error": str(error)}, sort_keys=True))
        return 1
    print(json.dumps({"ok": True, "result": result}, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
