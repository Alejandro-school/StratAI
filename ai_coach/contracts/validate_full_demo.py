"""Validate the two non-canonical full-demo AI Coach diagnostic artifacts.

The reports are small JSON summaries. Their detailed rows remain in sibling
gzip JSONL files and are consumed incrementally.  Nothing in this module makes
the diagnostics canonical or suitable for training.
"""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import math
import stat
from collections import Counter
from collections.abc import Callable, Mapping
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, BinaryIO

from ai_coach.contracts import validate_navigation_smoke as navigation_validation
from ai_coach.contracts import validate_visual_contact_smoke as visual_validation

NAVIGATION_REPORT_FILENAME = "observed_navigation_full_demo.json"
NAVIGATION_ROWS_FILENAME = "observed_navigation_full_demo_rows.jsonl.gz"
VISUAL_REPORT_FILENAME = "observed_visual_contact_full_demo.json"
VISUAL_AUDIT_FILENAME = "observed_visual_contact_full_demo_audit.jsonl.gz"

NAVIGATION_REPORT_SCHEMA_ID = "stratai.observed_navigation_full_demo_sweep@1"
NAVIGATION_SCOPE = "temporal_full_demo_sweep"
NAVIGATION_SELECTION_CONTRACT = (
    "one_deterministic_stress_pair_per_eligible_replay_frame@1"
)
NAVIGATION_DISTRIBUTION_CONTRACT = "nearest_rank_p50_p95_max@1"
NAVIGATION_ROWS_DIGEST_CONTRACT = "sha256_uncompressed_jsonl_bytes@1"

VISUAL_REPORT_SCHEMA_ID = "stratai.observed_visual_contact_full_demo_validation@1"
VISUAL_SELECTION_CONTRACT = "all_valid_replay_rounds_per_round_t0@1"
VISUAL_AUDIT_RECORD_SCHEMA_ID = (
    "stratai.observed_visual_contact_full_demo_audit_record@1"
)

TIMING_POLICY = {
    "intended_use": "internal_decision_support_only",
    "user_facing_mode": "qualitative_conditional_with_uncertainty_only",
    "decision_semantics": "constraint_signal_not_recommendation",
    "estimate_semantics": "scenario_reference_not_arrival_prediction",
    "exact_user_facing_time_allowed": False,
    "arrival_prediction": False,
}

NAVIGATION_REPORT_KEYS = {
    "schema_id",
    "status",
    "source_alias",
    "source_sha256",
    "map_name",
    "scope",
    "selection_contract",
    "pair_exhaustive",
    "directional_pairs",
    "candidate_schema_id",
    "candidate_contract_version",
    "graph_contract_version",
    "graph_asset_reference",
    "graph_asset_sha256",
    "graph_node_count",
    "graph_edge_count",
    "round_count",
    "tick_rate",
    "sample_stride_ticks",
    "sample_interval_milliseconds",
    "distribution_contract",
    "timing_policy",
    "build_identifier",
    "processing_duration_ms",
    "navigation_evaluation_duration_ms",
    "rows_artifact",
    "summary",
    "rounds",
}
NAVIGATION_ROWS_ARTIFACT_KEYS = {
    "path",
    "format",
    "compression",
    "bytes",
    "sha256",
    "row_count",
    "logical_bytes",
    "logical_sha256",
    "digest_contract",
}
NAVIGATION_SUMMARY_KEYS = {
    "replay_frames",
    "eligible_frames",
    "ineligible_frames",
    "counts",
    "distributions",
}
NAVIGATION_ROUND_KEYS = {
    "round_number",
    "replay_frames",
    "eligible_frames",
    "ineligible_frames",
    "counts",
}
NAVIGATION_COUNTS_KEYS = {
    "queries",
    "reachable_queries",
    "disconnected_queries",
    "endpoint_unavailable_queries",
    "optimistic_baseline",
    "active_weapon_baseline",
    "observed_pace_projection",
    "walking_contexts",
    "non_walking_moving_contexts",
    "observed_moving_contexts",
    "observed_stationary_contexts",
    "motion_unavailable_contexts",
    "ducking_contexts",
    "scoped_contexts",
    "knife_contexts",
    "active_weapon_contexts",
    "weapon_baseline_speed_contexts",
    "abstention_reasons",
    "exact_user_facing_time_violations",
    "arrival_prediction_violations",
}
NAVIGATION_SCENARIO_COUNT_KEYS = {
    "estimated",
    "unavailable",
    "unavailability_reasons",
}
NAVIGATION_DISTRIBUTION_KEYS = {
    "direct_distance_world_units",
    "path_distance_world_units",
    "path_to_direct_distance_ratio",
    "observed_horizontal_speed_world_units_per_second",
    "weapon_baseline_speed_world_units_per_second",
    "optimistic_baseline_milliseconds",
    "active_weapon_baseline_milliseconds",
    "observed_pace_projection_milliseconds",
}
NUMERIC_DISTRIBUTION_KEYS = {"count", "p50", "p95", "max"}

VISUAL_REPORT_KEYS = {
    "schema_id",
    "status",
    "source_alias",
    "source_sha256",
    "map_name",
    "candidate_schema_id",
    "candidate_contract_version",
    "training_allowed",
    "build_identifier",
    "processing_duration_ms",
    "selection",
    "sampling",
    "geometry_assets",
    "summary",
    "rounds",
    "audit",
}
VISUAL_SELECTION_KEYS = {
    "contract",
    "round_count",
    "first_round",
    "last_round",
    "minimum_start_tick",
    "maximum_end_tick",
    "input_source_frames",
    "canonical_source_frames",
    "per_round_t0",
}
VISUAL_METRIC_KEYS = {
    "evaluations",
    "observed_evaluations",
    "unavailable_evaluations",
    "line_of_sight_clear_pair_samples",
    "line_of_sight_blocked_pair_samples",
    "line_of_sight_unavailable_pair_samples",
    "fov_inside_pair_samples",
    "fov_outside_pair_samples",
    "fov_unavailable_pair_samples",
    "smoke_pair_sample_intersections",
    "smoke_clear_pair_samples",
    "smoke_unavailable_pair_samples",
    "active_smoke_pair_sample_evaluations",
    "otherwise_visible_pair_samples_blocked_by_smoke",
    "visual_contact_pair_samples",
    "no_visual_contact_pair_samples",
    "visual_contact_unavailable_pair_samples",
    "input_source_frames",
    "canonical_source_frames",
    "source_frames_with_valid_active_smoke",
    "evaluated_source_frames",
    "evaluated_source_frames_with_active_smoke",
    "evaluated_source_frames_with_smoke_intersection",
    "unique_valid_active_smoke_effects",
    "directional_episodes",
    "closed_directional_episodes",
    "right_censored_directional_episodes",
    "boundaries",
    "abstention_reasons",
    "raw_steam_id_violations",
}
VISUAL_ROUND_KEYS = {
    "round_number",
    "start_tick",
    "end_tick",
    "metrics",
    "participants_evaluated",
    "participants_with_contact",
    "unique_directed_pairs_with_contact",
    "unique_unordered_pairs_with_contact",
    "unique_directed_pairs_smoke_intersected",
    "unique_unordered_pairs_smoke_intersected",
    "reacquisition_episodes_within_round",
    "audit_logical_records",
    "audit_logical_sha256",
}
VISUAL_SUMMARY_KEYS = {
    "metrics",
    "participant_aliases",
    "participants_evaluated",
    "participants_with_contact",
    "unique_match_directed_pairs_with_contact",
    "unique_match_unordered_pairs_with_contact",
    "unique_round_directed_pairs_with_contact",
    "unique_match_directed_pairs_smoke_intersected",
    "unique_match_unordered_pairs_smoke_intersected",
    "reacquisition_episodes_within_round",
    "rounds_with_evaluations",
    "rounds_with_visual_contact",
    "rounds_with_valid_active_smoke",
    "rounds_with_smoke_intersection",
    "rounds_with_otherwise_visible_smoke_block",
}
VISUAL_AUDIT_KEYS = {
    "record_schema_id",
    "encoding",
    "compression",
    "evaluation_records",
    "episode_records",
    "logical_records",
    "logical_bytes",
    "logical_sha256",
    "gzip_relative_path",
    "gzip_bytes",
    "gzip_sha256",
}
VISUAL_ENVELOPE_KEYS = {"record_type", "round", "value"}
BOUNDARIES = {"started", "continued", "ended", "inactive", "unavailable"}
MAX_JSONL_LINE_BYTES = 8 * 1024 * 1024


class FullDemoValidationError(ValueError):
    """Raised when a complete-demo diagnostic is unsafe or inconsistent."""


def _fail(message: str) -> None:
    raise FullDemoValidationError(message)


def _object(value: Any, location: str) -> Mapping[str, Any]:
    if not isinstance(value, Mapping):
        _fail(f"{location}: expected object")
    return value


def _exact_keys(value: Mapping[str, Any], expected: set[str], location: str) -> None:
    missing = sorted(expected - set(value))
    unknown = sorted(set(value) - expected)
    if missing or unknown:
        _fail(f"{location}: missing={missing}, unknown={unknown}")


def _string(value: Any, location: str) -> str:
    if not isinstance(value, str) or not value.strip() or value != value.strip():
        _fail(f"{location}: expected canonical non-empty string")
    return value


def _integer(value: Any, location: str, *, minimum: int = 0) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or value < minimum:
        _fail(f"{location}: expected integer >= {minimum}")
    return value


def _number(value: Any, location: str, *, positive: bool = False) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        _fail(f"{location}: expected number")
    result = float(value)
    if not math.isfinite(result) or result < 0 or (positive and result <= 0):
        qualifier = "positive" if positive else "non-negative"
        _fail(f"{location}: expected finite {qualifier} number")
    return result


def _sha256(value: Any, location: str) -> str:
    if not isinstance(value, str) or not navigation_validation.SHA256_PATTERN.fullmatch(
        value
    ):
        _fail(f"{location}: expected lowercase SHA-256")
    return value


def _identity(report: Mapping[str, Any], location: str) -> tuple[str, str]:
    checksum = _sha256(report["source_sha256"], f"{location}.source_sha256")
    alias = report["source_alias"]
    if (
        not isinstance(alias, str)
        or not navigation_validation.SOURCE_ALIAS_PATTERN.fullmatch(alias)
        or alias != f"demo-{checksum[:20]}"
    ):
        _fail(f"{location}: source alias/checksum mismatch")
    return alias, checksum


def _relative_reference(value: Any, location: str) -> str:
    reference = _string(value, location)
    parts = reference.split("/")
    if (
        "\\" in reference
        or ":" in reference
        or reference.startswith("/")
        or any(part in {"", ".", ".."} for part in parts)
    ):
        _fail(f"{location}: expected normalized relative path")
    return reference


def _is_symlink_or_reparse(path: Path) -> bool:
    if path.is_symlink():
        return True
    attributes = getattr(path.lstat(), "st_file_attributes", 0)
    flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    return bool(attributes & flag)


def _safe_report(path: Path, expected_name: str, location: str) -> tuple[Path, Path]:
    if not isinstance(path, Path):
        _fail(f"{location}: expected pathlib.Path")
    logical = path.absolute()
    if logical.name != expected_name:
        _fail(f"{location}: expected filename {expected_name!r}")
    try:
        if _is_symlink_or_reparse(logical.parent) or _is_symlink_or_reparse(logical):
            _fail(f"{location}: symlinks, junctions, and reparse points are forbidden")
        parent = logical.parent.resolve(strict=True)
        resolved = logical.resolve(strict=True)
        resolved.relative_to(parent)
    except FullDemoValidationError:
        raise
    except (OSError, ValueError) as error:
        _fail(f"{location}: cannot resolve regular sibling file: {error}")
    if resolved.parent != parent or not resolved.is_file():
        _fail(f"{location}: expected a direct regular child of its report directory")
    return resolved, parent


def _safe_sibling(
    report_parent: Path, relative: Any, expected_name: str, location: str
) -> Path:
    reference = _relative_reference(relative, location)
    if reference != expected_name:
        _fail(f"{location}: expected {expected_name!r}")
    candidate = report_parent / reference
    try:
        if _is_symlink_or_reparse(candidate):
            _fail(f"{location}: symlinks, junctions, and reparse points are forbidden")
        resolved = candidate.resolve(strict=True)
        resolved.relative_to(report_parent)
    except FullDemoValidationError:
        raise
    except (OSError, ValueError) as error:
        _fail(f"{location}: cannot resolve sibling sidecar: {error}")
    if resolved.parent != report_parent or not resolved.is_file():
        _fail(f"{location}: sidecar must be a direct regular sibling")
    return resolved


def _safe_asset(maps_root: Path, reference: str, checksum: str, location: str) -> None:
    try:
        root = visual_validation._geometry_root(maps_root)
    except visual_validation.VisualContactSmokeValidationError as error:
        _fail(str(error))
    candidate = root
    try:
        for part in reference.split("/"):
            candidate /= part
            if _is_symlink_or_reparse(candidate):
                _fail(f"{location}: symlink or reparse component is forbidden")
        resolved = candidate.resolve(strict=True)
        resolved.relative_to(root)
    except FullDemoValidationError:
        raise
    except (OSError, ValueError) as error:
        _fail(f"{location}: cannot resolve inside maps_root: {error}")
    if not resolved.is_file():
        _fail(f"{location}: expected regular file")
    digest = hashlib.sha256()
    try:
        with resolved.open("rb") as handle:
            for chunk in iter(lambda: handle.read(1024 * 1024), b""):
                digest.update(chunk)
    except OSError as error:
        _fail(f"{location}: cannot read asset: {error}")
    if digest.hexdigest() != checksum:
        _fail(f"{location}: asset SHA-256 mismatch")


def _pairs_without_duplicates(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            _fail(f"JSON object: duplicate key {key!r}")
        result[key] = value
    return result


def _load_report(path: Path, location: str) -> Mapping[str, Any]:
    try:
        with path.open("r", encoding="utf-8") as handle:
            value = json.load(handle, object_pairs_hook=_pairs_without_duplicates)
    except FullDemoValidationError:
        raise
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        _fail(f"{location}: cannot decode JSON report: {error}")
    return _object(value, location)


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
    location: str,
    consume: Callable[[Mapping[str, Any], int, bytes], None],
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
                        _fail(f"{location}[{records}]: JSONL row exceeds safety limit")
                    if not line.endswith(b"\n"):
                        _fail(
                            f"{location}[{records}]: truncated row or missing newline"
                        )
                    if not line.strip():
                        _fail(f"{location}[{records}]: blank JSONL rows are forbidden")
                    logical_hash.update(line)
                    logical_bytes += len(line)
                    try:
                        value = json.loads(
                            line.decode("utf-8"),
                            object_pairs_hook=_pairs_without_duplicates,
                        )
                    except FullDemoValidationError:
                        raise
                    except (UnicodeError, json.JSONDecodeError) as error:
                        _fail(f"{location}[{records}]: invalid JSON: {error}")
                    consume(_object(value, f"{location}[{records}]"), records, line)
                    records += 1
    except FullDemoValidationError:
        raise
    except (OSError, EOFError, gzip.BadGzipFile) as error:
        _fail(f"{location}: invalid or truncated gzip stream: {error}")
    assert tracker is not None
    physical_size = path.stat().st_size
    if tracker.bytes != physical_size:
        _fail(f"{location}: gzip reader did not consume the complete physical artifact")
    return _StreamDigest(
        compressed_bytes=tracker.bytes,
        compressed_sha256=tracker.digest.hexdigest(),
        logical_bytes=logical_bytes,
        logical_sha256=logical_hash.hexdigest(),
        records=records,
    )


def _empty_scenario_counts() -> dict[str, Any]:
    return {"estimated": 0, "unavailable": 0, "unavailability_reasons": {}}


def _empty_navigation_counts() -> dict[str, Any]:
    return {
        "queries": 0,
        "reachable_queries": 0,
        "disconnected_queries": 0,
        "endpoint_unavailable_queries": 0,
        "optimistic_baseline": _empty_scenario_counts(),
        "active_weapon_baseline": _empty_scenario_counts(),
        "observed_pace_projection": _empty_scenario_counts(),
        "walking_contexts": 0,
        "non_walking_moving_contexts": 0,
        "observed_moving_contexts": 0,
        "observed_stationary_contexts": 0,
        "motion_unavailable_contexts": 0,
        "ducking_contexts": 0,
        "scoped_contexts": 0,
        "knife_contexts": 0,
        "active_weapon_contexts": {},
        "weapon_baseline_speed_contexts": {},
        "abstention_reasons": {},
        "exact_user_facing_time_violations": 0,
        "arrival_prediction_violations": 0,
    }


def _counter_increment(mapping: dict[str, int], key: str) -> None:
    mapping[key] = mapping.get(key, 0) + 1


def _go_float_key(value: float) -> str:
    if value.is_integer():
        return str(int(value))
    return format(value, ".15f").rstrip("0").rstrip(".")


def _accumulate_navigation_counts(
    counts: dict[str, Any], row: Mapping[str, Any]
) -> None:
    counts["queries"] += 1
    reachability = row["reachability"]
    if reachability["status"] == "derived" and reachability["value"] is True:
        counts["reachable_queries"] += 1
    elif reachability["status"] == "derived":
        counts["disconnected_queries"] += 1
    else:
        counts["endpoint_unavailable_queries"] += 1

    scenario_names = {
        "optimistic_baseline": "optimistic_uninterrupted_250ups_baseline",
        "active_weapon_baseline": "active_weapon_uninterrupted_baseline",
        "observed_pace_projection": "observed_pace_projection",
    }
    scenarios = row["internal_timing"]["scenarios"]
    for count_name, row_name in scenario_names.items():
        scenario = scenarios[row_name]
        target = counts[count_name]
        if scenario["status"] == "estimated":
            target["estimated"] += 1
        else:
            target["unavailable"] += 1
            _counter_increment(
                target["unavailability_reasons"], scenario["unavailability_reason"]
            )

    motion = row["origin_motion"]
    if motion["is_walking"] is True:
        counts["walking_contexts"] += 1
    if motion["is_ducking"] is True:
        counts["ducking_contexts"] += 1
    if motion["is_scoped"] is True:
        counts["scoped_contexts"] += 1
    if motion["horizontal_speed_status"] == "derived":
        speed = float(motion["horizontal_speed_world_units_per_second"])
        if speed > 0:
            counts["observed_moving_contexts"] += 1
            if motion["is_walking"] is False:
                counts["non_walking_moving_contexts"] += 1
        else:
            counts["observed_stationary_contexts"] += 1
    else:
        counts["motion_unavailable_contexts"] += 1
    if motion["active_weapon_status"] == "observed":
        weapon = _string(motion["active_weapon"], "navigation row active_weapon")
        _counter_increment(counts["active_weapon_contexts"], weapon)
        if weapon.casefold() == "knife":
            counts["knife_contexts"] += 1
    weapon_scenario = scenarios["active_weapon_uninterrupted_baseline"]
    if weapon_scenario["status"] == "estimated":
        speed = float(weapon_scenario["speed_world_units_per_second"])
        _counter_increment(
            counts["weapon_baseline_speed_contexts"], _go_float_key(speed)
        )
    if row["internal_timing"]["exact_user_facing_time_allowed"]:
        counts["exact_user_facing_time_violations"] += 1
    if row["internal_timing"]["arrival_prediction"]:
        counts["arrival_prediction_violations"] += 1
    for reason in row["abstention_reasons"]:
        _counter_increment(counts["abstention_reasons"], reason)


def _canonicalize_count_maps(counts: dict[str, Any]) -> dict[str, Any]:
    result = json.loads(json.dumps(counts))
    for name in (
        "optimistic_baseline",
        "active_weapon_baseline",
        "observed_pace_projection",
    ):
        result[name]["unavailability_reasons"] = dict(
            sorted(result[name]["unavailability_reasons"].items())
        )
    for name in (
        "active_weapon_contexts",
        "weapon_baseline_speed_contexts",
        "abstention_reasons",
    ):
        result[name] = dict(sorted(result[name].items()))
    return result


def _validate_navigation_counts(
    value: Any, expected: Mapping[str, Any], location: str
) -> None:
    counts = _object(value, location)
    _exact_keys(counts, NAVIGATION_COUNTS_KEYS, location)
    normalized: dict[str, Any] = {}
    map_names = {
        "active_weapon_contexts",
        "weapon_baseline_speed_contexts",
        "abstention_reasons",
    }
    scenario_names = {
        "optimistic_baseline",
        "active_weapon_baseline",
        "observed_pace_projection",
    }
    for key in NAVIGATION_COUNTS_KEYS - map_names - scenario_names:
        normalized[key] = _integer(counts[key], f"{location}.{key}")
    for key in map_names:
        mapping = _object(counts[key], f"{location}.{key}")
        normalized[key] = {}
        for name, count in mapping.items():
            canonical_name = _string(name, f"{location}.{key}.key")
            normalized[key][canonical_name] = _integer(
                count, f"{location}.{key}.{canonical_name}", minimum=1
            )
        normalized[key] = dict(sorted(normalized[key].items()))
    for key in scenario_names:
        scenario = _object(counts[key], f"{location}.{key}")
        _exact_keys(scenario, NAVIGATION_SCENARIO_COUNT_KEYS, f"{location}.{key}")
        reasons = _object(
            scenario["unavailability_reasons"],
            f"{location}.{key}.unavailability_reasons",
        )
        normalized[key] = {
            "estimated": _integer(scenario["estimated"], f"{location}.{key}.estimated"),
            "unavailable": _integer(
                scenario["unavailable"], f"{location}.{key}.unavailable"
            ),
            "unavailability_reasons": {
                _string(name, f"{location}.{key}.unavailability_reasons.key"): _integer(
                    count,
                    f"{location}.{key}.unavailability_reasons.{name}",
                    minimum=1,
                )
                for name, count in reasons.items()
            },
        }
        normalized[key]["unavailability_reasons"] = dict(
            sorted(normalized[key]["unavailability_reasons"].items())
        )
    if normalized != dict(expected):
        _fail(f"{location}: differs from streamed navigation rows")


def _nearest_rank(values: list[float], percentile: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    index = math.ceil(percentile * len(ordered)) - 1
    return ordered[max(0, min(index, len(ordered) - 1))]


def _distribution(values: list[float]) -> dict[str, Any]:
    return {
        "count": len(values),
        "p50": _nearest_rank(values, 0.50),
        "p95": _nearest_rank(values, 0.95),
        "max": max(values) if values else None,
    }


def _validate_distribution(
    value: Any, expected: Mapping[str, Any], location: str
) -> None:
    distribution = _object(value, location)
    _exact_keys(distribution, NUMERIC_DISTRIBUTION_KEYS, location)
    if _integer(distribution["count"], f"{location}.count") != expected["count"]:
        _fail(f"{location}.count: differs from streamed rows")
    for key in ("p50", "p95", "max"):
        wanted = expected[key]
        actual = distribution[key]
        if wanted is None:
            if actual is not None:
                _fail(f"{location}.{key}: expected null for an empty distribution")
        elif actual is None or not math.isclose(
            _number(actual, f"{location}.{key}"),
            float(wanted),
            rel_tol=1e-12,
            abs_tol=1e-9,
        ):
            _fail(f"{location}.{key}: differs from streamed rows")


@dataclass
class _NavigationStreamState:
    alias: str
    graph_reference: str
    graph_sha256: str
    report_rounds: set[int]
    counts: dict[str, Any] = field(default_factory=_empty_navigation_counts)
    round_counts: dict[int, dict[str, Any]] = field(default_factory=dict)
    seen_queries: set[str] = field(default_factory=set)
    seen_frames: set[tuple[int, int]] = field(default_factory=set)
    previous_frame: tuple[int, int] | None = None
    distributions: dict[str, list[float]] = field(
        default_factory=lambda: {key: [] for key in NAVIGATION_DISTRIBUTION_KEYS}
    )

    def consume(self, row: Mapping[str, Any], index: int, _line: bytes) -> None:
        try:
            navigation_validation._validate_navigation_row(
                row,
                index,
                source_alias=self.alias,
                graph_asset_reference=self.graph_reference,
                graph_asset_sha256=self.graph_sha256,
            )
        except navigation_validation.NavigationSmokeValidationError as error:
            _fail(str(error))
        round_number = row["round_number"]
        tick = row["t0_tick"]
        frame = (round_number, tick)
        if round_number not in self.report_rounds:
            _fail(f"navigation.rows[{index}]: row references an unknown report round")
        if self.previous_frame is not None and frame <= self.previous_frame:
            _fail("navigation.rows: rows must be strictly ordered by round/t0_tick")
        if frame in self.seen_frames:
            _fail("navigation.rows: duplicate round/t0_tick frame")
        query_id = row["join_keys"]["query_id"]
        if query_id in self.seen_queries:
            _fail("navigation.rows: duplicate query_id")
        self.previous_frame = frame
        self.seen_frames.add(frame)
        self.seen_queries.add(query_id)
        round_counts = self.round_counts.setdefault(
            round_number, _empty_navigation_counts()
        )
        _accumulate_navigation_counts(self.counts, row)
        _accumulate_navigation_counts(round_counts, row)

        direct = row["direct_distance"]["value_world_units"]
        path = row["path_distance"]["value_world_units"]
        if direct is not None:
            direct_value = float(direct)
            self.distributions["direct_distance_world_units"].append(direct_value)
            if path is not None:
                path_value = float(path)
                self.distributions["path_distance_world_units"].append(path_value)
                if direct_value > 0:
                    self.distributions["path_to_direct_distance_ratio"].append(
                        path_value / direct_value
                    )
        motion = row["origin_motion"]
        if motion["horizontal_speed_status"] == "derived":
            self.distributions[
                "observed_horizontal_speed_world_units_per_second"
            ].append(float(motion["horizontal_speed_world_units_per_second"]))
        scenarios = row["internal_timing"]["scenarios"]
        scenario_distributions = {
            "optimistic_uninterrupted_250ups_baseline": "optimistic_baseline_milliseconds",
            "active_weapon_uninterrupted_baseline": "active_weapon_baseline_milliseconds",
            "observed_pace_projection": "observed_pace_projection_milliseconds",
        }
        for scenario_name, distribution_name in scenario_distributions.items():
            scenario = scenarios[scenario_name]
            if scenario["status"] == "estimated":
                self.distributions[distribution_name].append(
                    float(scenario["value_milliseconds"])
                )
        weapon = scenarios["active_weapon_uninterrupted_baseline"]
        if weapon["status"] == "estimated":
            self.distributions["weapon_baseline_speed_world_units_per_second"].append(
                float(weapon["speed_world_units_per_second"])
            )


def _validate_navigation_report(
    report: Mapping[str, Any], report_parent: Path, maps_root: Path
) -> dict[str, Any]:
    _exact_keys(report, NAVIGATION_REPORT_KEYS, "navigation")
    if (
        report["schema_id"] != NAVIGATION_REPORT_SCHEMA_ID
        or report["status"] != "passed"
    ):
        _fail("navigation: schema or status mismatch")
    alias, checksum = _identity(report, "navigation")
    expected_contract = {
        "scope": NAVIGATION_SCOPE,
        "selection_contract": NAVIGATION_SELECTION_CONTRACT,
        "pair_exhaustive": False,
        "directional_pairs": True,
        "candidate_schema_id": navigation_validation.NAVIGATION_SCHEMA_ID,
        "candidate_contract_version": navigation_validation.NAVIGATION_CONTRACT_VERSION,
        "graph_contract_version": navigation_validation.GRAPH_CONTRACT_VERSION,
        "distribution_contract": NAVIGATION_DISTRIBUTION_CONTRACT,
    }
    for key, expected in expected_contract.items():
        if report[key] != expected:
            _fail(f"navigation.{key}: contract mismatch")
    if (
        report["pair_exhaustive"] is not False
        or report["directional_pairs"] is not True
    ):
        _fail("navigation: pair scope flags must be booleans")
    timing_policy = _object(report["timing_policy"], "navigation.timing_policy")
    _exact_keys(timing_policy, set(TIMING_POLICY), "navigation.timing_policy")
    if dict(timing_policy) != TIMING_POLICY:
        _fail("navigation.timing_policy: unsafe or unknown policy")
    if (
        timing_policy["exact_user_facing_time_allowed"] is not False
        or timing_policy["arrival_prediction"] is not False
    ):
        _fail("navigation.timing_policy: policy flags must be booleans")
    map_name = _string(report["map_name"], "navigation.map_name")
    build = _string(report["build_identifier"], "navigation.build_identifier")
    graph_reference = _relative_reference(
        report["graph_asset_reference"], "navigation.graph_asset_reference"
    )
    graph_sha256 = _sha256(
        report["graph_asset_sha256"], "navigation.graph_asset_sha256"
    )
    _safe_asset(maps_root, graph_reference, graph_sha256, "navigation.graph_asset")
    for key in ("graph_node_count", "graph_edge_count", "round_count"):
        _integer(report[key], f"navigation.{key}", minimum=1)
    tick_rate = _number(report["tick_rate"], "navigation.tick_rate", positive=True)
    stride = _integer(
        report["sample_stride_ticks"], "navigation.sample_stride_ticks", minimum=1
    )
    interval = _number(
        report["sample_interval_milliseconds"],
        "navigation.sample_interval_milliseconds",
        positive=True,
    )
    if not math.isclose(interval, stride * 1000 / tick_rate, rel_tol=1e-12):
        _fail("navigation.sample_interval_milliseconds: inconsistent with stride/rate")
    for key in ("processing_duration_ms", "navigation_evaluation_duration_ms"):
        _integer(report[key], f"navigation.{key}")

    raw_rounds = report["rounds"]
    if not isinstance(raw_rounds, list) or not raw_rounds:
        _fail("navigation.rounds: expected non-empty list")
    if report["round_count"] != len(raw_rounds):
        _fail("navigation.round_count: differs from rounds")
    rounds: list[Mapping[str, Any]] = []
    previous_round = 0
    for index, value in enumerate(raw_rounds):
        location = f"navigation.rounds[{index}]"
        round_summary = _object(value, location)
        _exact_keys(round_summary, NAVIGATION_ROUND_KEYS, location)
        number = _integer(
            round_summary["round_number"], f"{location}.round_number", minimum=1
        )
        if number <= previous_round:
            _fail("navigation.rounds: must be strictly ordered")
        previous_round = number
        replay_frames = _integer(
            round_summary["replay_frames"], f"{location}.replay_frames"
        )
        eligible = _integer(
            round_summary["eligible_frames"], f"{location}.eligible_frames"
        )
        ineligible = _integer(
            round_summary["ineligible_frames"], f"{location}.ineligible_frames"
        )
        if eligible + ineligible != replay_frames:
            _fail(f"{location}: frame partition does not reconcile")
        rounds.append(round_summary)

    artifact = _object(report["rows_artifact"], "navigation.rows_artifact")
    _exact_keys(artifact, NAVIGATION_ROWS_ARTIFACT_KEYS, "navigation.rows_artifact")
    expected_artifact = {
        "path": NAVIGATION_ROWS_FILENAME,
        "format": "jsonl",
        "compression": "gzip",
        "digest_contract": NAVIGATION_ROWS_DIGEST_CONTRACT,
    }
    for key, expected in expected_artifact.items():
        if artifact[key] != expected:
            _fail(f"navigation.rows_artifact.{key}: contract mismatch")
    physical_bytes = _integer(
        artifact["bytes"], "navigation.rows_artifact.bytes", minimum=1
    )
    physical_sha = _sha256(artifact["sha256"], "navigation.rows_artifact.sha256")
    logical_bytes = _integer(
        artifact["logical_bytes"],
        "navigation.rows_artifact.logical_bytes",
        minimum=1,
    )
    logical_sha = _sha256(
        artifact["logical_sha256"], "navigation.rows_artifact.logical_sha256"
    )
    row_count = _integer(
        artifact["row_count"], "navigation.rows_artifact.row_count", minimum=1
    )
    sidecar = _safe_sibling(
        report_parent,
        artifact["path"],
        NAVIGATION_ROWS_FILENAME,
        "navigation.rows_artifact.path",
    )
    state = _NavigationStreamState(
        alias=alias,
        graph_reference=graph_reference,
        graph_sha256=graph_sha256,
        report_rounds={item["round_number"] for item in rounds},
    )
    digest = _consume_gzip_jsonl(sidecar, "navigation.rows", state.consume)
    if (
        digest.compressed_bytes != physical_bytes
        or digest.compressed_sha256 != physical_sha
        or digest.logical_bytes != logical_bytes
        or digest.logical_sha256 != logical_sha
        or digest.records != row_count
    ):
        _fail("navigation.rows_artifact: physical/logical integrity mismatch")

    global_counts = _canonicalize_count_maps(state.counts)
    if global_counts["queries"] != row_count:
        _fail("navigation.rows_artifact.row_count: differs from streamed queries")
    summary = _object(report["summary"], "navigation.summary")
    _exact_keys(summary, NAVIGATION_SUMMARY_KEYS, "navigation.summary")
    replay_frames = _integer(
        summary["replay_frames"], "navigation.summary.replay_frames"
    )
    eligible = _integer(
        summary["eligible_frames"], "navigation.summary.eligible_frames"
    )
    ineligible = _integer(
        summary["ineligible_frames"], "navigation.summary.ineligible_frames"
    )
    if eligible != row_count or replay_frames != eligible + ineligible:
        _fail("navigation.summary: frame coverage differs from streamed rows")
    _validate_navigation_counts(
        summary["counts"], global_counts, "navigation.summary.counts"
    )
    distributions = _object(
        summary["distributions"], "navigation.summary.distributions"
    )
    _exact_keys(
        distributions,
        NAVIGATION_DISTRIBUTION_KEYS,
        "navigation.summary.distributions",
    )
    for name in NAVIGATION_DISTRIBUTION_KEYS:
        _validate_distribution(
            distributions[name],
            _distribution(state.distributions[name]),
            f"navigation.summary.distributions.{name}",
        )

    round_frames = round_eligible = round_ineligible = 0
    for index, round_summary in enumerate(rounds):
        number = round_summary["round_number"]
        expected_counts = _canonicalize_count_maps(
            state.round_counts.get(number, _empty_navigation_counts())
        )
        _validate_navigation_counts(
            round_summary["counts"],
            expected_counts,
            f"navigation.rounds[{index}].counts",
        )
        if round_summary["eligible_frames"] != expected_counts["queries"]:
            _fail(f"navigation.rounds[{index}]: eligible frames differ from rows")
        round_frames += round_summary["replay_frames"]
        round_eligible += round_summary["eligible_frames"]
        round_ineligible += round_summary["ineligible_frames"]
    if (round_frames, round_eligible, round_ineligible) != (
        replay_frames,
        eligible,
        ineligible,
    ):
        _fail("navigation.rounds: totals differ from global summary")
    if (
        global_counts["reachable_queries"] == 0
        or global_counts["optimistic_baseline"]["estimated"]
        != global_counts["reachable_queries"]
        or global_counts["active_weapon_baseline"]["estimated"] == 0
        or global_counts["observed_pace_projection"]["estimated"] == 0
        or global_counts["exact_user_facing_time_violations"] != 0
        or global_counts["arrival_prediction_violations"] != 0
    ):
        _fail("navigation: coverage or exact-time policy gate failed")
    return {
        "source_alias": alias,
        "source_sha256": checksum,
        "map_name": map_name,
        "build_identifier": build,
        "rows": row_count,
        "rows_sha256": physical_sha,
        "reachable_queries": global_counts["reachable_queries"],
    }


def _empty_visual_metrics() -> dict[str, Any]:
    return {
        key: (
            {boundary: 0 for boundary in sorted(BOUNDARIES)}
            if key == "boundaries"
            else {}
        )
        if key in {"boundaries", "abstention_reasons"}
        else 0
        for key in VISUAL_METRIC_KEYS
    }


def _visual_pair(left: str, right: str, *, unordered: bool = False) -> tuple[str, str]:
    return tuple(sorted((left, right))) if unordered else (left, right)


def _adapt_full_contact_alias(value: Any, location: str) -> tuple[str | None, Any]:
    """Validate @1 full-demo aliases and adapt them for the shared row validator.

    The bounded smoke uses three-digit contact aliases, while the full-demo
    producer deliberately reserves six digits. The candidate schema is the
    same, so only the local diagnostic alias representation differs.
    """

    if value is None:
        return None, None
    contact_id = _string(value, location)
    if not contact_id.startswith("contact-"):
        _fail(f"{location}: expected full-demo contact alias")
    digits = contact_id.removeprefix("contact-")
    if not digits.isdigit() or len(digits) < 6:
        _fail(f"{location}: expected at least six canonical digits")
    number = int(digits)
    if number <= 0 or digits != f"{number:06d}":
        _fail(f"{location}: non-canonical full-demo contact alias")
    return contact_id, f"contact-{number:03d}"


def _adapt_visual_value_for_shared_validator(
    value: Any, location: str
) -> tuple[Mapping[str, Any], str | None]:
    row = _object(value, location)
    joins = _object(row.get("join_keys"), f"{location}.join_keys")
    original, adapted = _adapt_full_contact_alias(
        joins.get("contact_id"), f"{location}.join_keys.contact_id"
    )
    if original is None:
        return row, None
    adapted_joins = dict(joins)
    adapted_joins["contact_id"] = adapted
    adapted_row = dict(row)
    adapted_row["join_keys"] = adapted_joins
    return adapted_row, original


@dataclass
class _VisualRoundState:
    summary: Mapping[str, Any]
    evaluations: list[dict[str, Any]] = field(default_factory=list)
    episodes: list[dict[str, Any]] = field(default_factory=list)
    digest: Any = field(default_factory=hashlib.sha256)
    records: int = 0
    episodes_started: bool = False
    previous_evaluation: tuple[int, str, str] | None = None


@dataclass
class _VisualStreamState:
    alias: str
    rounds: dict[int, _VisualRoundState]
    round_order: dict[int, int]
    current_round_index: int = -1
    evaluation_records: int = 0
    episode_records: int = 0
    seen_contact_ids: set[str] = field(default_factory=set)
    participants_evaluated: set[str] = field(default_factory=set)
    participants_with_contact: set[str] = field(default_factory=set)
    directed_contact_pairs: set[tuple[str, str]] = field(default_factory=set)
    unordered_contact_pairs: set[tuple[str, str]] = field(default_factory=set)
    directed_smoke_pairs: set[tuple[str, str]] = field(default_factory=set)
    unordered_smoke_pairs: set[tuple[str, str]] = field(default_factory=set)
    round_directed_contact_pairs: set[tuple[int, str, str]] = field(default_factory=set)
    global_metrics: dict[str, Any] = field(default_factory=_empty_visual_metrics)
    round_summaries: dict[int, dict[str, Any]] = field(default_factory=dict)
    finalized_rounds: set[int] = field(default_factory=set)

    def _round_number_at(self, index: int) -> int:
        return min(
            number for number, order in self.round_order.items() if order == index
        )

    def _finalize_round_index(self, index: int) -> None:
        round_number = self._round_number_at(index)
        if round_number in self.finalized_rounds:
            return
        round_state = self.rounds[round_number]
        metrics, summary = _recompute_visual_round(round_state, index)
        _add_visual_metrics(self.global_metrics, metrics)
        self.round_summaries[round_number] = summary
        self.finalized_rounds.add(round_number)
        round_state.evaluations.clear()
        round_state.episodes.clear()

    def _finalize_before(self, order: int) -> None:
        start = max(self.current_round_index, 0)
        for index in range(start, order):
            self._finalize_round_index(index)

    def finish(self) -> None:
        start = max(self.current_round_index, 0)
        for index in range(start, len(self.round_order)):
            self._finalize_round_index(index)
        self.current_round_index = len(self.round_order)

    def consume(self, envelope: Mapping[str, Any], index: int, line: bytes) -> None:
        location = f"visual.audit[{index}]"
        _exact_keys(envelope, VISUAL_ENVELOPE_KEYS, location)
        record_type = envelope["record_type"]
        if record_type not in {"evaluation", "episode"}:
            _fail(f"{location}.record_type: unexpected value")
        round_number = _integer(envelope["round"], f"{location}.round", minimum=1)
        if round_number not in self.rounds:
            _fail(f"{location}.round: unknown report round")
        order = self.round_order[round_number]
        if order < self.current_round_index:
            _fail("visual.audit: rounds are not contiguous and ordered")
        if order > self.current_round_index:
            self._finalize_before(order)
        self.current_round_index = max(self.current_round_index, order)
        if round_number in self.finalized_rounds:
            _fail("visual.audit: record appears after its round was finalized")
        state = self.rounds[round_number]
        state.digest.update(line)
        state.records += 1
        selection = {
            "round_number": round_number,
            "start_tick": state.summary["start_tick"],
            "end_tick": state.summary["end_tick"],
        }
        if record_type == "evaluation":
            if state.episodes_started:
                _fail(f"{location}: evaluation appears after an episode")
            try:
                adapted_value, original_contact = (
                    _adapt_visual_value_for_shared_validator(
                        envelope["value"], f"{location}.value"
                    )
                )
                normalized = visual_validation._validate_evaluation(
                    adapted_value,
                    len(state.evaluations),
                    self.alias,
                    state.summary["end_tick"],
                    selection,
                )
            except visual_validation.VisualContactSmokeValidationError as error:
                _fail(str(error))
            normalized["contact_id"] = original_contact
            order_key = (
                normalized["tick"],
                normalized["observer"],
                normalized["subject"],
            )
            if (
                state.previous_evaluation is not None
                and order_key <= state.previous_evaluation
            ):
                _fail(f"{location}: evaluations are not in canonical order")
            state.previous_evaluation = order_key
            state.evaluations.append(normalized)
            self.evaluation_records += 1
            self.participants_evaluated.update(
                (normalized["observer"], normalized["subject"])
            )
            directed = _visual_pair(normalized["observer"], normalized["subject"])
            unordered = _visual_pair(
                normalized["observer"], normalized["subject"], unordered=True
            )
            if normalized["visual_contact"] is True:
                self.participants_with_contact.update(
                    (normalized["observer"], normalized["subject"])
                )
                self.directed_contact_pairs.add(directed)
                self.unordered_contact_pairs.add(unordered)
                self.round_directed_contact_pairs.add((round_number, *directed))
            if normalized["smoke_occluded"] is True:
                self.directed_smoke_pairs.add(directed)
                self.unordered_smoke_pairs.add(unordered)
        else:
            state.episodes_started = True
            try:
                adapted_value, original_contact = (
                    _adapt_visual_value_for_shared_validator(
                        envelope["value"], f"{location}.value"
                    )
                )
                normalized = visual_validation._validate_episode(
                    adapted_value,
                    len(state.episodes),
                    self.alias,
                    state.summary["end_tick"],
                    selection,
                )
            except visual_validation.VisualContactSmokeValidationError as error:
                _fail(str(error))
            normalized["contact_id"] = original_contact
            contact_id = normalized["contact_id"]
            if contact_id in self.seen_contact_ids:
                _fail(f"{location}: duplicate contact ID across rounds")
            expected_contact = f"contact-{self.episode_records + 1:06d}"
            if contact_id != expected_contact:
                _fail(f"{location}: contact aliases must be globally sequential")
            self.seen_contact_ids.add(contact_id)
            state.episodes.append(normalized)
            self.episode_records += 1


def _reported_frame_metrics(value: Mapping[str, Any], location: str) -> dict[str, int]:
    names = {
        "input_source_frames",
        "canonical_source_frames",
        "source_frames_with_valid_active_smoke",
        "unique_valid_active_smoke_effects",
    }
    return {name: _integer(value[name], f"{location}.{name}") for name in names}


def _normalize_visual_metrics(value: Any, location: str) -> dict[str, Any]:
    metrics = _object(value, location)
    _exact_keys(metrics, VISUAL_METRIC_KEYS, location)
    normalized = {
        key: _integer(metrics[key], f"{location}.{key}")
        for key in VISUAL_METRIC_KEYS - {"boundaries", "abstention_reasons"}
    }
    boundaries = _object(metrics["boundaries"], f"{location}.boundaries")
    _exact_keys(boundaries, BOUNDARIES, f"{location}.boundaries")
    normalized["boundaries"] = {
        boundary: _integer(boundaries[boundary], f"{location}.boundaries.{boundary}")
        for boundary in sorted(BOUNDARIES)
    }
    reasons = _object(metrics["abstention_reasons"], f"{location}.abstention_reasons")
    normalized["abstention_reasons"] = dict(
        sorted(
            (
                _string(reason, f"{location}.abstention_reasons.key"),
                _integer(
                    count,
                    f"{location}.abstention_reasons.{reason}",
                    minimum=1,
                ),
            )
            for reason, count in reasons.items()
        )
    )
    return normalized


def _recompute_visual_round(
    state: _VisualRoundState, index: int
) -> tuple[dict[str, Any], dict[str, Any]]:
    location = f"visual.rounds[{index}]"
    try:
        visual_validation._validate_episode_state(state.evaluations, state.episodes)
    except visual_validation.VisualContactSmokeValidationError as error:
        _fail(str(error))
    reported_metrics = _normalize_visual_metrics(
        state.summary["metrics"], f"{location}.metrics"
    )
    retained = _reported_frame_metrics(reported_metrics, f"{location}.metrics")
    metrics = _empty_visual_metrics()
    metrics.update(retained)
    evaluations = state.evaluations
    episodes = state.episodes
    metrics.update(
        {
            "evaluations": len(evaluations),
            "observed_evaluations": sum(
                row["status"] == "observed" for row in evaluations
            ),
            "unavailable_evaluations": sum(
                row["status"] == "unavailable" for row in evaluations
            ),
            "line_of_sight_clear_pair_samples": sum(
                row["line_of_sight"] is True for row in evaluations
            ),
            "line_of_sight_blocked_pair_samples": sum(
                row["line_of_sight"] is False for row in evaluations
            ),
            "line_of_sight_unavailable_pair_samples": sum(
                row["line_of_sight"] is None for row in evaluations
            ),
            "fov_inside_pair_samples": sum(
                row["inside_fov"] is True for row in evaluations
            ),
            "fov_outside_pair_samples": sum(
                row["inside_fov"] is False for row in evaluations
            ),
            "fov_unavailable_pair_samples": sum(
                row["inside_fov"] is None for row in evaluations
            ),
            "smoke_pair_sample_intersections": sum(
                row["smoke_occluded"] is True for row in evaluations
            ),
            "smoke_clear_pair_samples": sum(
                row["smoke_occluded"] is False for row in evaluations
            ),
            "smoke_unavailable_pair_samples": sum(
                row["smoke_occluded"] is None for row in evaluations
            ),
            "active_smoke_pair_sample_evaluations": sum(
                row["active_smoke_count"] is not None and row["active_smoke_count"] > 0
                for row in evaluations
            ),
            "otherwise_visible_pair_samples_blocked_by_smoke": sum(
                row["smoke_occluded"] is True
                and row["line_of_sight"] is True
                and row["inside_fov"] is True
                for row in evaluations
            ),
            "visual_contact_pair_samples": sum(
                row["visual_contact"] is True for row in evaluations
            ),
            "no_visual_contact_pair_samples": sum(
                row["visual_contact"] is False for row in evaluations
            ),
            "visual_contact_unavailable_pair_samples": sum(
                row["visual_contact"] is None for row in evaluations
            ),
            "directional_episodes": len(episodes),
            "closed_directional_episodes": sum(
                row["status"] == "closed" for row in episodes
            ),
            "right_censored_directional_episodes": sum(
                row["status"] == "open_at_t0" for row in episodes
            ),
            "boundaries": {
                boundary: sum(row["boundary"] == boundary for row in evaluations)
                for boundary in sorted(BOUNDARIES)
            },
            "abstention_reasons": dict(
                sorted(
                    Counter(
                        reason for row in evaluations for reason in row["reasons"]
                    ).items()
                )
            ),
            "raw_steam_id_violations": 0,
        }
    )
    frames = {row["source_frame_tick"] for row in evaluations}
    active_frames = {
        row["source_frame_tick"]
        for row in evaluations
        if row["active_smoke_count"] is not None and row["active_smoke_count"] > 0
    }
    intersection_frames = {
        row["source_frame_tick"] for row in evaluations if row["smoke_occluded"] is True
    }
    metrics["evaluated_source_frames"] = len(frames)
    metrics["evaluated_source_frames_with_active_smoke"] = len(active_frames)
    metrics["evaluated_source_frames_with_smoke_intersection"] = len(
        intersection_frames
    )
    if reported_metrics != metrics:
        _fail(f"{location}.metrics: differs from streamed audit records")
    if not (
        metrics["evaluated_source_frames_with_smoke_intersection"]
        <= metrics["evaluated_source_frames_with_active_smoke"]
        <= metrics["source_frames_with_valid_active_smoke"]
        <= metrics["canonical_source_frames"]
        <= metrics["input_source_frames"]
    ):
        _fail(f"{location}.metrics: source-frame denominators are inconsistent")
    if (
        metrics["smoke_pair_sample_intersections"] > 0
        and metrics["unique_valid_active_smoke_effects"] == 0
    ):
        _fail(f"{location}.metrics: smoke intersections require a valid smoke effect")

    participants = {
        participant
        for row in evaluations
        for participant in (row["observer"], row["subject"])
    }
    contact_participants = {
        participant
        for row in evaluations
        if row["visual_contact"] is True
        for participant in (row["observer"], row["subject"])
    }
    directed_contact = {
        _visual_pair(row["observer"], row["subject"])
        for row in evaluations
        if row["visual_contact"] is True
    }
    unordered_contact = {
        _visual_pair(row["observer"], row["subject"], unordered=True)
        for row in evaluations
        if row["visual_contact"] is True
    }
    directed_smoke = {
        _visual_pair(row["observer"], row["subject"])
        for row in evaluations
        if row["smoke_occluded"] is True
    }
    unordered_smoke = {
        _visual_pair(row["observer"], row["subject"], unordered=True)
        for row in evaluations
        if row["smoke_occluded"] is True
    }
    expected_summary = {
        "round_number": state.summary["round_number"],
        "start_tick": state.summary["start_tick"],
        "end_tick": state.summary["end_tick"],
        "metrics": metrics,
        "participants_evaluated": len(participants),
        "participants_with_contact": len(contact_participants),
        "unique_directed_pairs_with_contact": len(directed_contact),
        "unique_unordered_pairs_with_contact": len(unordered_contact),
        "unique_directed_pairs_smoke_intersected": len(directed_smoke),
        "unique_unordered_pairs_smoke_intersected": len(unordered_smoke),
        "reacquisition_episodes_within_round": len(episodes) - len(directed_contact),
        "audit_logical_records": state.records,
        "audit_logical_sha256": state.digest.hexdigest(),
    }
    for key in VISUAL_ROUND_KEYS - {
        "metrics",
        "audit_logical_sha256",
    }:
        _integer(state.summary[key], f"{location}.{key}")
    _sha256(state.summary["audit_logical_sha256"], f"{location}.audit_logical_sha256")
    if dict(state.summary) != expected_summary:
        _fail(f"{location}: summary/hash differs from streamed audit round")
    return metrics, expected_summary


def _add_visual_metrics(target: dict[str, Any], value: Mapping[str, Any]) -> None:
    for key in VISUAL_METRIC_KEYS - {"boundaries", "abstention_reasons"}:
        target[key] += value[key]
    for key in BOUNDARIES:
        target["boundaries"][key] += value["boundaries"][key]
    for reason, count in value["abstention_reasons"].items():
        target["abstention_reasons"][reason] = (
            target["abstention_reasons"].get(reason, 0) + count
        )
    target["abstention_reasons"] = dict(sorted(target["abstention_reasons"].items()))


def _validate_visual_report(
    report: Mapping[str, Any], report_parent: Path, maps_root: Path
) -> dict[str, Any]:
    _exact_keys(report, VISUAL_REPORT_KEYS, "visual")
    if report["schema_id"] != VISUAL_REPORT_SCHEMA_ID or report["status"] != "passed":
        _fail("visual: schema or status mismatch")
    alias, checksum = _identity(report, "visual")
    if (
        report["candidate_schema_id"] != visual_validation.CANDIDATE_SCHEMA_ID
        or report["candidate_contract_version"]
        != visual_validation.CANDIDATE_CONTRACT_VERSION
        or report["training_allowed"] is not False
    ):
        _fail("visual: candidate contract or training policy mismatch")
    map_name = _string(report["map_name"], "visual.map_name")
    build = _string(report["build_identifier"], "visual.build_identifier")
    _integer(report["processing_duration_ms"], "visual.processing_duration_ms")
    try:
        visual_validation._validate_sampling(report["sampling"])
        visual_validation._validate_geometry_assets(
            report["geometry_assets"], maps_root
        )
    except visual_validation.VisualContactSmokeValidationError as error:
        _fail(str(error))

    raw_rounds = report["rounds"]
    if not isinstance(raw_rounds, list) or not raw_rounds:
        _fail("visual.rounds: expected non-empty list")
    rounds: dict[int, _VisualRoundState] = {}
    round_order: dict[int, int] = {}
    previous = 0
    for index, value in enumerate(raw_rounds):
        location = f"visual.rounds[{index}]"
        summary = _object(value, location)
        _exact_keys(summary, VISUAL_ROUND_KEYS, location)
        number = _integer(
            summary["round_number"], f"{location}.round_number", minimum=1
        )
        start = _integer(summary["start_tick"], f"{location}.start_tick")
        end = _integer(summary["end_tick"], f"{location}.end_tick")
        if number <= previous or end < start:
            _fail(f"{location}: rounds must be ordered with valid bounds")
        previous = number
        rounds[number] = _VisualRoundState(summary=summary)
        round_order[number] = index

    selection = _object(report["selection"], "visual.selection")
    _exact_keys(selection, VISUAL_SELECTION_KEYS, "visual.selection")
    for key in VISUAL_SELECTION_KEYS - {"contract", "per_round_t0"}:
        _integer(selection[key], f"visual.selection.{key}")
    expected_selection = {
        "contract": VISUAL_SELECTION_CONTRACT,
        "round_count": len(raw_rounds),
        "first_round": raw_rounds[0]["round_number"],
        "last_round": raw_rounds[-1]["round_number"],
        "minimum_start_tick": min(item["start_tick"] for item in raw_rounds),
        "maximum_end_tick": max(item["end_tick"] for item in raw_rounds),
        "input_source_frames": sum(
            item["metrics"]["input_source_frames"] for item in raw_rounds
        ),
        "canonical_source_frames": sum(
            item["metrics"]["canonical_source_frames"] for item in raw_rounds
        ),
        "per_round_t0": "round_end_tick",
    }
    if dict(selection) != expected_selection:
        _fail("visual.selection: differs from complete ordered round coverage")

    audit = _object(report["audit"], "visual.audit")
    _exact_keys(audit, VISUAL_AUDIT_KEYS, "visual.audit")
    expected_audit_contract = {
        "record_schema_id": VISUAL_AUDIT_RECORD_SCHEMA_ID,
        "encoding": "jsonl",
        "compression": "gzip",
        "gzip_relative_path": VISUAL_AUDIT_FILENAME,
    }
    for key, expected in expected_audit_contract.items():
        if audit[key] != expected:
            _fail(f"visual.audit.{key}: contract mismatch")
    gzip_bytes = _integer(audit["gzip_bytes"], "visual.audit.gzip_bytes", minimum=1)
    gzip_sha = _sha256(audit["gzip_sha256"], "visual.audit.gzip_sha256")
    logical_records = _integer(
        audit["logical_records"], "visual.audit.logical_records", minimum=1
    )
    logical_bytes = _integer(
        audit["logical_bytes"], "visual.audit.logical_bytes", minimum=1
    )
    logical_sha = _sha256(audit["logical_sha256"], "visual.audit.logical_sha256")
    evaluation_records = _integer(
        audit["evaluation_records"], "visual.audit.evaluation_records", minimum=1
    )
    episode_records = _integer(audit["episode_records"], "visual.audit.episode_records")
    sidecar = _safe_sibling(
        report_parent,
        audit["gzip_relative_path"],
        VISUAL_AUDIT_FILENAME,
        "visual.audit.gzip_relative_path",
    )
    state = _VisualStreamState(alias=alias, rounds=rounds, round_order=round_order)
    digest = _consume_gzip_jsonl(sidecar, "visual.audit", state.consume)
    if (
        digest.compressed_bytes != gzip_bytes
        or digest.compressed_sha256 != gzip_sha
        or digest.logical_bytes != logical_bytes
        or digest.logical_sha256 != logical_sha
        or digest.records != logical_records
        or state.evaluation_records != evaluation_records
        or state.episode_records != episode_records
        or logical_records != evaluation_records + episode_records
    ):
        _fail("visual.audit: physical/logical integrity or record counts mismatch")

    state.finish()
    global_metrics = state.global_metrics
    round_summaries = [state.round_summaries[round_number] for round_number in rounds]
    summary = _object(report["summary"], "visual.summary")
    _exact_keys(summary, VISUAL_SUMMARY_KEYS, "visual.summary")
    reported_metrics = _normalize_visual_metrics(
        summary["metrics"], "visual.summary.metrics"
    )
    if reported_metrics != global_metrics:
        _fail("visual.summary.metrics: differs from per-round streamed totals")
    participant_aliases = _integer(
        summary["participant_aliases"], "visual.summary.participant_aliases", minimum=1
    )
    max_alias = max(
        (
            int(alias_value.removeprefix("participant-"))
            for alias_value in state.participants_evaluated
        ),
        default=0,
    )
    if (
        participant_aliases < len(state.participants_evaluated)
        or max_alias > participant_aliases
    ):
        _fail("visual.summary.participant_aliases: cannot cover retained aliases")
    expected_summary = {
        "metrics": global_metrics,
        "participant_aliases": participant_aliases,
        "participants_evaluated": len(state.participants_evaluated),
        "participants_with_contact": len(state.participants_with_contact),
        "unique_match_directed_pairs_with_contact": len(state.directed_contact_pairs),
        "unique_match_unordered_pairs_with_contact": len(state.unordered_contact_pairs),
        "unique_round_directed_pairs_with_contact": len(
            state.round_directed_contact_pairs
        ),
        "unique_match_directed_pairs_smoke_intersected": len(
            state.directed_smoke_pairs
        ),
        "unique_match_unordered_pairs_smoke_intersected": len(
            state.unordered_smoke_pairs
        ),
        "reacquisition_episodes_within_round": global_metrics["directional_episodes"]
        - len(state.round_directed_contact_pairs),
        "rounds_with_evaluations": sum(
            item["metrics"]["evaluations"] > 0 for item in round_summaries
        ),
        "rounds_with_visual_contact": sum(
            item["metrics"]["visual_contact_pair_samples"] > 0
            for item in round_summaries
        ),
        "rounds_with_valid_active_smoke": sum(
            item["metrics"]["source_frames_with_valid_active_smoke"] > 0
            for item in round_summaries
        ),
        "rounds_with_smoke_intersection": sum(
            item["metrics"]["smoke_pair_sample_intersections"] > 0
            for item in round_summaries
        ),
        "rounds_with_otherwise_visible_smoke_block": sum(
            item["metrics"]["otherwise_visible_pair_samples_blocked_by_smoke"] > 0
            for item in round_summaries
        ),
    }
    for key in VISUAL_SUMMARY_KEYS - {"metrics"}:
        _integer(summary[key], f"visual.summary.{key}")
    if dict(summary) != expected_summary:
        _fail("visual.summary: differs from streamed audit records")
    if (
        global_metrics["evaluations"] == 0
        or global_metrics["observed_evaluations"] == 0
    ):
        _fail("visual: real observed evaluation coverage is empty")
    return {
        "source_alias": alias,
        "source_sha256": checksum,
        "map_name": map_name,
        "build_identifier": build,
        "audit_records": logical_records,
        "audit_sha256": gzip_sha,
        "evaluations": evaluation_records,
        "episodes": episode_records,
        "smoke_intersections": global_metrics["smoke_pair_sample_intersections"],
        "smoke_blocks": global_metrics[
            "otherwise_visible_pair_samples_blocked_by_smoke"
        ],
    }


def validate_full_demo_reports(
    navigation_report: Path, visual_report: Path, maps_root: Path
) -> dict[str, Any]:
    """Validate both reports, their sibling streams, provenance, and joins."""

    navigation_path, navigation_parent = _safe_report(
        navigation_report, NAVIGATION_REPORT_FILENAME, "navigation_report"
    )
    visual_path, visual_parent = _safe_report(
        visual_report, VISUAL_REPORT_FILENAME, "visual_report"
    )
    if navigation_parent != visual_parent:
        _fail("reports: navigation and visual reports must be physical siblings")
    navigation = _validate_navigation_report(
        _load_report(navigation_path, "navigation"), navigation_parent, maps_root
    )
    visual = _validate_visual_report(
        _load_report(visual_path, "visual"), visual_parent, maps_root
    )
    for key in ("source_alias", "source_sha256", "map_name", "build_identifier"):
        if navigation[key] != visual[key]:
            _fail(f"reports: navigation and visual {key} differ")
    return {"navigation": navigation, "visual": visual, "training_allowed": False}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("navigation_report", type=Path)
    parser.add_argument("visual_report", type=Path)
    parser.add_argument(
        "--maps-root",
        required=True,
        type=Path,
        help="Read-only root containing NAV and geometry assets",
    )
    arguments = parser.parse_args()
    try:
        summary = validate_full_demo_reports(
            arguments.navigation_report,
            arguments.visual_report,
            arguments.maps_root,
        )
    except (FullDemoValidationError, OSError) as error:
        print(json.dumps({"ok": False, "error": str(error)}, sort_keys=True))
        return 1
    print(json.dumps({"ok": True, "summary": summary}, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
