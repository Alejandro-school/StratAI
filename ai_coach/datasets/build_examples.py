"""Build and audit structural DecisionExample rows without creating a training dataset.

Artifacts are discovered only through committed manifests. Every JSONL file is
read line by line, gzip is never expanded on disk, and one bundle is released
before the next bundle is opened. Individual examples are validated and then
discarded; only aggregate counters and a small anonymous sample are returned.
"""

from __future__ import annotations

import argparse
import copy
import json
import re
import sys
from collections import Counter, defaultdict
from collections.abc import Callable, Iterator, Mapping, Sequence
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from ai_coach.contracts.runtime_contracts import (
    CONTRACT_VERSION,
    DECISION_EXAMPLE_SCHEMA_ID,
    RuntimeContractError,
    validate_decision_example,
)
from ai_coach.datasets.eligibility import evaluate_task_eligibility
from ai_coach.datasets.streaming import (
    BundleManifest,
    DatasetContractError,
    LocatedRow,
    RawRow,
    discover_bundle_dirs,
    iter_jsonl,
    iter_jsonl_raw,
    load_bundle_manifest,
    load_direct_bundle_manifest,
)

REPORT_SCHEMA_ID = "stratai.ai_coach_decision_example_coverage@1"
REPORT_VERSION = "1.0.0"
MAX_SAMPLES_PER_FAMILY = 2
MATERIALIZED_FAMILIES = (
    "peek_hold_or_reposition",
    "spacing_or_trade_connection",
)
_TACTICAL_SELF_RE = re.compile(rb'"visibility_scope"\s*:\s*"self"')
_TACTICAL_OBSERVER_RE = re.compile(rb'"observer_id"\s*:\s*"([^"\\]+)"')
_TACTICAL_SUBJECT_RE = re.compile(rb'"subject_id"\s*:\s*"([^"\\]+)"')
_TACTICAL_ROUND_RE = re.compile(rb'"round_number"\s*:\s*([0-9]+)')
_TACTICAL_AVAILABILITY_RE = re.compile(rb'"availability_tick"\s*:\s*([0-9]+)')
EXPECTED_SCHEMAS = {
    "decisions": "stratai.decision@1",
    "decision_features": "stratai.decision_features@1",
    "oracle_context": "stratai.oracle_context@1",
    "decision_outcomes": "stratai.decision_outcome@1",
    "quality_masks": "stratai.quality_mask@1",
    "player_states": "stratai.player_state@3",
    "tactical_observations": "stratai.tactical_physical_observation@1",
}
LINK_FIELDS = ("match_id", "decision_id", "round_number", "t0_tick")
_PARTITION_METADATA_KEYS = {
    "schema_id",
    "match_id",
    "decision_id",
    "decision_type",
    "round_id",
    "round_number",
    "state_id",
    "player_id",
    "actor_player_id",
    "observer_id",
    "subject_id",
    "source_record_id",
    "source_event_ids",
    "t0_tick",
    "availability_tick",
    "availability_tick_max",
    "tick",
    "identity_semantics",
    "join_keys",
}


def _is_int(value: Any) -> bool:
    return isinstance(value, int) and not isinstance(value, bool)


def _require_string(row: LocatedRow, key: str, *, identifier: bool = False) -> str:
    value = row.value.get(key)
    if not isinstance(value, str) or not value:
        code = "INVALID_IDENTIFIER_TYPE" if identifier else "INVALID_FIELD_TYPE"
        raise DatasetContractError(
            code,
            f"{key} must be a non-empty string",
            path=row.artifact.evidence_path,
            line=row.line,
        )
    return value


def _require_int(row: LocatedRow, key: str, *, minimum: int = 0) -> int:
    value = row.value.get(key)
    if not _is_int(value) or value < minimum:
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            f"{key} must be an integer >= {minimum}",
            path=row.artifact.evidence_path,
            line=row.line,
        )
    return value


def _check_schema(row: LocatedRow) -> None:
    expected = EXPECTED_SCHEMAS.get(row.artifact.artifact_type)
    if expected is None:
        return
    if row.artifact.schema_id != expected or row.value.get("schema_id") != expected:
        raise DatasetContractError(
            "UNSUPPORTED_SCHEMA",
            f"expected {expected} in descriptor and row",
            path=row.artifact.evidence_path,
            line=row.line,
        )


def _index_partition(
    manifest: BundleManifest, artifact_type: str
) -> dict[str, LocatedRow]:
    artifact = manifest.one(artifact_type)
    rows: dict[str, LocatedRow] = {}
    for located in iter_jsonl(artifact):
        _check_schema(located)
        decision_id = _require_string(located, "decision_id", identifier=True)
        if decision_id in rows:
            raise DatasetContractError(
                "DUPLICATE_IDENTIFIER",
                f"duplicate decision_id {decision_id}",
                path=artifact.evidence_path,
                line=located.line,
            )
        rows[decision_id] = located
    return rows


def _assert_partition_keys(
    decisions: Mapping[str, LocatedRow],
    partitions: Mapping[str, Mapping[str, LocatedRow]],
) -> None:
    expected = set(decisions)
    for name, partition in partitions.items():
        actual = set(partition)
        if actual != expected:
            missing = sorted(expected - actual)
            extra = sorted(actual - expected)
            detail = f"partition ids differ; missing={missing[:2]}, extra={extra[:2]}"
            artifact_path = (
                next(iter(partition.values())).artifact.evidence_path
                if partition
                else name
            )
            raise DatasetContractError(
                "PARTITION_KEY_MISMATCH", detail, path=artifact_path
            )


def _validate_join_fields(decision: LocatedRow, other: LocatedRow) -> None:
    for key in LINK_FIELDS:
        if other.value.get(key) != decision.value.get(key):
            raise DatasetContractError(
                "PARTITION_KEY_MISMATCH",
                f"{key} does not match decisions partition",
                path=other.artifact.evidence_path,
                line=other.line,
            )


def _validate_decision_rows(
    decision: LocatedRow,
    features: LocatedRow,
    oracle: LocatedRow,
    outcome: LocatedRow,
    quality: LocatedRow,
) -> None:
    for row in (features, oracle, outcome, quality):
        _validate_join_fields(decision, row)
    family = _require_string(decision, "decision_type")
    if family not in MATERIALIZED_FAMILIES:
        raise DatasetContractError(
            "UNSUPPORTED_SCHEMA",
            f"unsupported materialized decision family {family}",
            path=decision.artifact.evidence_path,
            line=decision.line,
        )
    if features.value.get("decision_type") != family:
        raise DatasetContractError(
            "PARTITION_KEY_MISMATCH",
            "decision_type differs between decisions and decision_features",
            path=features.artifact.evidence_path,
            line=features.line,
        )
    t0_tick = _require_int(decision, "t0_tick")
    availability_tick = _require_int(decision, "availability_tick")
    feature_tick = _require_int(features, "availability_tick_max")
    if availability_tick > t0_tick:
        raise DatasetContractError(
            "T0_AVAILABILITY_VIOLATION",
            "decision availability_tick is after t0_tick",
            path=decision.artifact.evidence_path,
            line=decision.line,
        )
    if feature_tick > t0_tick:
        raise DatasetContractError(
            "T0_AVAILABILITY_VIOLATION",
            "decision_features availability_tick_max is after t0_tick",
            path=features.artifact.evidence_path,
            line=features.line,
        )
    outcome_tick = _require_int(outcome, "outcome_tick")
    if outcome_tick < t0_tick:
        raise DatasetContractError(
            "OUTCOME_BEFORE_T0",
            "outcome_tick is before t0_tick",
            path=outcome.artifact.evidence_path,
            line=outcome.line,
        )
    _require_string(decision, "actor_player_id", identifier=True)
    _require_string(decision, "observed_state_ref", identifier=True)
    if decision.value.get("actor_id_usage") != "join_only":
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "actor_player_id must be declared join_only",
            path=decision.artifact.evidence_path,
            line=decision.line,
        )
    if decision.value.get("state_availability_status") != "observed":
        raise DatasetContractError(
            "REQUIRED_FIELD_UNAVAILABLE",
            "the referenced actor state is not observed",
            path=decision.artifact.evidence_path,
            line=decision.line,
        )
    if not isinstance(oracle.value.get("available"), bool) or oracle.value.get(
        "status"
    ) not in {
        "available",
        "unavailable",
    }:
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "oracle availability must be explicit",
            path=oracle.artifact.evidence_path,
            line=oracle.line,
        )
    mask_sets: list[set[str]] = []
    for key in ("available_fields", "unavailable_fields", "inferred_fields"):
        values = quality.value.get(key)
        if not isinstance(values, list) or any(
            not isinstance(item, str) for item in values
        ):
            raise DatasetContractError(
                "INVALID_FIELD_TYPE",
                f"{key} must be a list of strings",
                path=quality.artifact.evidence_path,
                line=quality.line,
            )
        if values != sorted(set(values)):
            raise DatasetContractError(
                "DUPLICATE_IDENTIFIER",
                f"{key} must be sorted and unique",
                path=quality.artifact.evidence_path,
                line=quality.line,
            )
        mask_sets.append(set(values))
    if any(
        mask_sets[left] & mask_sets[right] for left, right in ((0, 1), (0, 2), (1, 2))
    ):
        raise DatasetContractError(
            "PARTITION_KEY_MISMATCH",
            "quality masks overlap",
            path=quality.artifact.evidence_path,
            line=quality.line,
        )


def _join_referenced_states(
    manifest: BundleManifest,
    decisions: Mapping[str, LocatedRow],
) -> dict[str, LocatedRow]:
    targets: dict[str, list[str]] = defaultdict(list)
    for decision_id, decision in decisions.items():
        targets[
            _require_string(decision, "observed_state_ref", identifier=True)
        ].append(decision_id)
    joined: dict[str, LocatedRow] = {}
    seen_state_ids: set[str] = set()
    for artifact in manifest.many("player_states"):
        for state in iter_jsonl(artifact):
            _check_schema(state)
            state_id = _require_string(state, "state_id", identifier=True)
            if state_id in seen_state_ids:
                raise DatasetContractError(
                    "DUPLICATE_IDENTIFIER",
                    f"duplicate state_id {state_id}",
                    path=artifact.evidence_path,
                    line=state.line,
                )
            seen_state_ids.add(state_id)
            for decision_id in targets.get(state_id, ()):
                decision = decisions[decision_id]
                if state.value.get("match_id") != decision.value.get("match_id"):
                    raise DatasetContractError(
                        "PARTITION_KEY_MISMATCH",
                        "referenced state match_id differs",
                        path=artifact.evidence_path,
                        line=state.line,
                    )
                if state.value.get("round_number") != decision.value.get(
                    "round_number"
                ):
                    raise DatasetContractError(
                        "PARTITION_KEY_MISMATCH",
                        "referenced state round_number differs",
                        path=artifact.evidence_path,
                        line=state.line,
                    )
                if state.value.get("player_id") != decision.value.get(
                    "actor_player_id"
                ):
                    raise DatasetContractError(
                        "PARTITION_KEY_MISMATCH",
                        "referenced state player_id differs from actor_player_id",
                        path=artifact.evidence_path,
                        line=state.line,
                    )
                state_tick = _require_int(state, "tick")
                if state_tick > _require_int(decision, "t0_tick"):
                    raise DatasetContractError(
                        "T0_AVAILABILITY_VIOLATION",
                        "referenced player state is after t0_tick",
                        path=artifact.evidence_path,
                        line=state.line,
                    )
                joined[decision_id] = state
    missing = sorted(set(decisions) - set(joined))
    if missing:
        decision = decisions[missing[0]]
        raise DatasetContractError(
            "JOIN_REFERENCE_MISSING",
            f"observed_state_ref does not resolve for decision {missing[0]}",
            path=decision.artifact.evidence_path,
            line=decision.line,
        )
    return joined


def _join_tactical_states(
    manifest: BundleManifest,
    decisions: Mapping[str, LocatedRow],
) -> tuple[dict[str, LocatedRow], int, int]:
    targets: dict[tuple[int, str], list[tuple[int, str]]] = defaultdict(list)
    for decision_id, decision in decisions.items():
        key = (
            _require_int(decision, "round_number", minimum=1),
            _require_string(decision, "actor_player_id", identifier=True),
        )
        targets[key].append((_require_int(decision, "t0_tick"), decision_id))
    for values in targets.values():
        values.sort()
    indexes = {key: 0 for key in targets}
    latest: dict[tuple[int, str], RawRow] = {}
    latest_tick: dict[tuple[int, str], int] = {}
    joined_raw: dict[str, RawRow] = {}
    rows_streamed = 0

    artifact = manifest.one("tactical_observations")
    for raw_state in iter_jsonl_raw(artifact):
        rows_streamed += 1
        raw = raw_state.raw
        if _TACTICAL_SELF_RE.search(raw) is None:
            continue
        observer_match = _TACTICAL_OBSERVER_RE.search(raw)
        subject_match = _TACTICAL_SUBJECT_RE.search(raw)
        round_match = _TACTICAL_ROUND_RE.search(raw)
        availability_match = _TACTICAL_AVAILABILITY_RE.search(raw)
        if not all((observer_match, subject_match, round_match, availability_match)):
            raise DatasetContractError(
                "INVALID_IDENTIFIER_TYPE",
                "tactical self observation has invalid join identifiers or ticks",
                path=artifact.evidence_path,
                line=raw_state.line,
            )
        observer_id = observer_match.group(1).decode("utf-8")
        subject_id = subject_match.group(1).decode("utf-8")
        round_number = int(round_match.group(1))
        availability_tick = int(availability_match.group(1))
        if observer_id != subject_id:
            continue
        key = (round_number, observer_id)
        if key not in targets:
            continue
        previous_tick = latest_tick.get(key)
        if previous_tick is not None and availability_tick < previous_tick:
            raise DatasetContractError(
                "PARTITION_KEY_MISMATCH",
                "tactical observations are not ordered by availability_tick",
                path=artifact.evidence_path,
                line=raw_state.line,
            )
        values = targets[key]
        index = indexes[key]
        while index < len(values) and values[index][0] < availability_tick:
            _, decision_id = values[index]
            if key in latest:
                joined_raw[decision_id] = latest[key]
            index += 1
        indexes[key] = index
        latest[key] = raw_state
        latest_tick[key] = availability_tick

    for key, values in targets.items():
        index = indexes[key]
        while index < len(values):
            t0_tick, decision_id = values[index]
            candidate = latest.get(key)
            candidate_tick = latest_tick.get(key)
            if (
                candidate is not None
                and candidate_tick is not None
                and candidate_tick <= t0_tick
            ):
                joined_raw[decision_id] = candidate
            index += 1
    missing = sorted(set(decisions) - set(joined_raw))
    if missing:
        decision = decisions[missing[0]]
        raise DatasetContractError(
            "JOIN_REFERENCE_MISSING",
            f"no legal tactical self observation exists at or before T0 for decision {missing[0]}",
            path=artifact.evidence_path,
            line=decision.line,
        )
    decoded_by_line: dict[int, LocatedRow] = {}
    joined: dict[str, LocatedRow] = {}
    for decision_id, raw_state in joined_raw.items():
        state = decoded_by_line.get(raw_state.line)
        if state is None:
            state = raw_state.decode()
            _check_schema(state)
            join_keys = state.value.get("join_keys")
            if not isinstance(join_keys, Mapping):
                raise DatasetContractError(
                    "INVALID_IDENTIFIER_TYPE",
                    "tactical observation join_keys must be an object",
                    path=artifact.evidence_path,
                    line=state.line,
                )
            match_id = join_keys.get("match_id")
            observer_id = join_keys.get("observer_id")
            subject_id = join_keys.get("subject_id")
            if not all(
                isinstance(value, str) and value
                for value in (match_id, observer_id, subject_id)
            ):
                raise DatasetContractError(
                    "INVALID_IDENTIFIER_TYPE",
                    "tactical join identifiers must be non-empty strings",
                    path=artifact.evidence_path,
                    line=state.line,
                )
            if match_id != manifest.bundle_id:
                raise DatasetContractError(
                    "PARTITION_KEY_MISMATCH",
                    "tactical match_id differs from root manifest",
                    path=artifact.evidence_path,
                    line=state.line,
                )
            decoded_by_line[raw_state.line] = state
        joined[decision_id] = state
    return joined, artifact.record_count or rows_streamed, len(decoded_by_line)


def _strip_partition_metadata(value: Any) -> Any:
    if isinstance(value, Mapping):
        return {
            str(key): _strip_partition_metadata(child)
            for key, child in value.items()
            if str(key) not in _PARTITION_METADATA_KEYS
            and not str(key).endswith("_id")
            and not str(key).endswith("_ids")
            and not str(key).endswith("_ref")
            and not str(key).endswith("_refs")
        }
    if isinstance(value, list):
        return [_strip_partition_metadata(child) for child in value]
    return copy.deepcopy(value)


def _build_example(
    decision: LocatedRow,
    features: LocatedRow,
    oracle: LocatedRow,
    outcome: LocatedRow,
    quality: LocatedRow,
    referenced_state: LocatedRow,
    tactical_state: LocatedRow,
) -> dict[str, Any]:
    t0_tick = int(decision.value["t0_tick"])
    decision_id = str(decision.value["decision_id"])
    family = str(decision.value["decision_type"])
    oracle_values = _strip_partition_metadata(oracle.value)
    outcome_values = _strip_partition_metadata(outcome.value)
    quality_values = _strip_partition_metadata(quality.value)
    example = {
        "schema_id": DECISION_EXAMPLE_SCHEMA_ID,
        "contract_version": CONTRACT_VERSION,
        "example_id": decision_id,
        "decision_family": family,
        "join_keys": {
            "match_id": decision.value["match_id"],
            "decision_id": decision_id,
            "round_number": decision.value["round_number"],
            "actor_player_id": decision.value["actor_player_id"],
            "observed_state_ref": decision.value["observed_state_ref"],
        },
        "t0_tick": t0_tick,
        "observed": {
            "action": {
                "availability_tick": decision.value["availability_tick"],
                "status": decision.value["availability_status"],
                "values": {
                    "action_taken": decision.value["action_taken"],
                    "causal_role": decision.value["causal_role"],
                    "source": decision.value["source"],
                    "visibility_scope": decision.value["visibility_scope"],
                },
            },
            "referenced_state": {
                "availability_tick": referenced_state.value["tick"],
                "status": decision.value["state_availability_status"],
                "values": _strip_partition_metadata(referenced_state.value),
            },
            "tactical_state": {
                "availability_tick": tactical_state.value["availability_tick"],
                "status": tactical_state.value["status"],
                "values": {
                    "state": _strip_partition_metadata(
                        tactical_state.value.get("state", {})
                    )
                },
            },
            "context": {
                "availability_tick": features.value["availability_tick_max"],
                "status": "observed",
                "values": _strip_partition_metadata(features.value),
            },
            "quality": quality_values,
        },
        "oracle": {
            "audit_only": True,
            "available": oracle.value["available"],
            "status": oracle.value["status"],
            "values": oracle_values,
        },
        "outcomes": {
            "label_or_evaluation_only": True,
            "outcome_tick": outcome.value["outcome_tick"],
            "values": outcome_values,
        },
        "historical_profile": {
            "strictly_before_t0": True,
            "status": "not_materialized",
            "values": None,
        },
        "references": {
            "observed_action": decision.reference(),
            "outcome": outcome.reference(),
            "quality": quality.reference(),
        },
        "evidence": [
            decision.reference(),
            features.reference(),
            referenced_state.reference(),
            tactical_state.reference(),
            oracle.reference(),
            outcome.reference(),
            quality.reference(),
        ],
        "limitations": sorted(
            {
                "Los outcomes describen el futuro y no son features observed.",
                "No se genera una etiqueta de correcto o incorrecto.",
                "Oracle se conserva sólo para auditoría o evaluación.",
                "Los identificadores se conservan sólo en join_keys.",
                "enemy_los es un proxy geométrico parcial sin FOV ni humo completos.",
            }
        ),
    }
    try:
        validate_decision_example(example)
    except RuntimeContractError as exc:
        code = (
            "ORACLE_IN_OBSERVED"
            if "oracle/future" in str(exc)
            else "INVALID_FIELD_TYPE"
        )
        raise DatasetContractError(
            code,
            str(exc),
            path=decision.artifact.evidence_path,
            line=decision.line,
        ) from exc
    return example


def _probe_status_pair(
    values: Mapping[str, Any], value_key: str, status_key: str
) -> str:
    if value_key not in values or status_key not in values:
        return "missing"
    if values.get(status_key) == "unavailable" or values.get(value_key) is None:
        return "null"
    return "present"


_CANDIDATE_SPATIAL_FACT_FIELD_IDS = frozenset(
    {
        "FLD-GEOMETRIC-CLEARANCE-SECTORS",
        "FLD-GEOMETRIC-OCCLUSION",
        "FLD-NAV-EXIT-PATH",
    }
)


def _probe_field(
    field_id: str,
    *,
    features: Mapping[str, Any],
    referenced_state: Mapping[str, Any],
    tactical_state: Mapping[str, Any],
) -> str:
    if field_id in _CANDIDATE_SPATIAL_FACT_FIELD_IDS:
        # Canonical bundle 3.8 has no factual-sidecar join. Fail closed until a
        # successor contract materializes it; legacy review forms are not a
        # fallback source for objective facts or tactical labels.
        return "missing"
    if field_id == "FLD-SELF-STATE-16HZ":
        return (
            "present" if isinstance(tactical_state.get("state"), Mapping) else "missing"
        )
    if field_id == "FLD-TEAMMATE-STATES":
        value = features.get("observed_participant_states")
        return (
            "present"
            if _is_int(value) and value > 0
            else "null"
            if value is None
            else "missing"
        )
    if field_id == "FLD-TRADE-POSSIBLE":
        return _probe_status_pair(features, "trade_possible", "trade_possible_status")
    if field_id == "FLD-CONNECTION-TIME":
        return _probe_status_pair(
            features, "nearest_connection_time_ms", "nearest_connection_time_status"
        )
    if field_id in {"FLD-LINE-OF-SIGHT", "FLD-VISIBLE-CONTACT"}:
        return _probe_status_pair(features, "any_line_of_sight", "line_of_sight_status")
    if field_id == "FLD-FACING-DELTA":
        return _probe_status_pair(features, "minimum_facing_delta_deg", "facing_status")
    if field_id == "FLD-ROUND-CLOCK":
        return (
            "present"
            if features.get("round_clock_remaining_ms") is not None
            else "null"
        )
    if field_id == "FLD-BOMB-CLOCK":
        return (
            "present" if features.get("bomb_time_remaining_ms") is not None else "null"
        )
    if field_id == "FLD-OBJECTIVE-STATE":
        return (
            "present" if referenced_state.get("objective_phase") is not None else "null"
        )
    return "missing"


@dataclass
class CountMetric:
    present: int = 0
    missing: int = 0
    null: int = 0

    def observe(self, state: str) -> None:
        if state not in {"present", "missing", "null"}:
            raise ValueError(f"unknown field state {state}")
        setattr(self, state, getattr(self, state) + 1)

    def as_dict(self) -> dict[str, int]:
        return {"present": self.present, "missing": self.missing, "null": self.null}


@dataclass
class BundleSummary:
    family_counts: Counter[str] = field(default_factory=Counter)
    action_counts: dict[str, Counter[str]] = field(
        default_factory=lambda: defaultdict(Counter)
    )
    joins_succeeded: int = 0
    jsonl_rows_streamed: int = 0
    gzip_rows_streamed: int = 0
    gzip_rows_json_decoded: int = 0
    field_coverage: dict[str, dict[str, CountMetric]] = field(
        default_factory=lambda: defaultdict(lambda: defaultdict(CountMetric))
    )
    structural_fields: dict[str, CountMetric] = field(
        default_factory=lambda: defaultdict(CountMetric)
    )
    samples: dict[str, list[dict[str, Any]]] = field(
        default_factory=lambda: defaultdict(list)
    )
    export_format_version: str = ""
    parser_schema_version: str = ""


def _anonymous_sample(example: Mapping[str, Any]) -> dict[str, Any]:
    observed = example["observed"]
    return {
        "decision_family": example["decision_family"],
        "action_taken": observed["action"]["values"]["action_taken"],
        "t0_tick": example["t0_tick"],
        "observed_availability_ticks": {
            "action": observed["action"]["availability_tick"],
            "context": observed["context"]["availability_tick"],
            "referenced_state": observed["referenced_state"]["availability_tick"],
            "tactical_state": observed["tactical_state"]["availability_tick"],
        },
        "oracle_status": example["oracle"]["status"],
        "outcome_tick": example["outcomes"]["outcome_tick"],
        "quality_warning_flags": observed["quality"].get("warning_flags", []),
    }


def process_manifest(
    manifest: BundleManifest,
    *,
    catalog_required_fields: Sequence[str],
    spatial_candidate: Any = None,
    example_sink: Callable[[dict[str, Any]], None] | None = None,
) -> BundleSummary:
    """Apply the shared row, join, causal, streaming, and coverage validations."""

    decisions = _index_partition(manifest, "decisions")
    partitions = {
        "decision_features": _index_partition(manifest, "decision_features"),
        "oracle_context": _index_partition(manifest, "oracle_context"),
        "decision_outcomes": _index_partition(manifest, "decision_outcomes"),
        "quality_masks": _index_partition(manifest, "quality_masks"),
    }
    _assert_partition_keys(decisions, partitions)
    for decision_id, decision in decisions.items():
        _validate_decision_rows(
            decision,
            partitions["decision_features"][decision_id],
            partitions["oracle_context"][decision_id],
            partitions["decision_outcomes"][decision_id],
            partitions["quality_masks"][decision_id],
        )
    referenced_states = _join_referenced_states(manifest, decisions)
    tactical_states, tactical_rows, tactical_rows_decoded = _join_tactical_states(
        manifest, decisions
    )

    result = BundleSummary(
        export_format_version=manifest.export_format_version,
        parser_schema_version=manifest.parser_schema_version,
        gzip_rows_streamed=tactical_rows,
        gzip_rows_json_decoded=tactical_rows_decoded,
    )
    result.jsonl_rows_streamed = sum(
        artifact.record_count or 0
        for artifact_type in (
            "decisions",
            "decision_features",
            "oracle_context",
            "decision_outcomes",
            "quality_masks",
            "player_states",
        )
        for artifact in manifest.many(artifact_type)
    )
    for decision_id in sorted(decisions):
        decision = decisions[decision_id]
        features = partitions["decision_features"][decision_id]
        oracle = partitions["oracle_context"][decision_id]
        outcome = partitions["decision_outcomes"][decision_id]
        quality = partitions["quality_masks"][decision_id]
        referenced_state = referenced_states[decision_id]
        tactical_state = tactical_states[decision_id]
        example = _build_example(
            decision,
            features,
            oracle,
            outcome,
            quality,
            referenced_state,
            tactical_state,
        )
        if spatial_candidate is not None:
            example = spatial_candidate.attach(example)
        family = str(example["decision_family"])
        action = str(example["observed"]["action"]["values"]["action_taken"])
        result.family_counts[family] += 1
        result.action_counts[family][action] += 1
        result.joins_succeeded += 1
        result.structural_fields["decision.actor_player_id"].observe("present")
        result.structural_fields["decision.observed_state_ref"].observe("present")
        result.structural_fields["decision.availability_tick"].observe("present")
        result.structural_fields["decision_features.availability_tick_max"].observe(
            "present"
        )
        result.structural_fields["observed.referenced_state"].observe("present")
        result.structural_fields["observed.tactical_state"].observe("present")
        result.structural_fields["outcomes.outcome_tick"].observe("present")
        result.structural_fields["oracle.available"].observe(
            "present" if oracle.value.get("available") is not None else "missing"
        )
        for optional_path, value in (
            (
                "decision_features.round_clock_remaining_ms",
                features.value.get("round_clock_remaining_ms"),
            ),
            (
                "decision_features.bomb_time_remaining_ms",
                features.value.get("bomb_time_remaining_ms"),
            ),
        ):
            result.structural_fields[optional_path].observe(
                "present" if value is not None else "null"
            )
        for field_id in catalog_required_fields:
            result.field_coverage[field_id][family].observe(
                spatial_candidate.field_status(example, field_id)
                if spatial_candidate is not None and field_id in _CANDIDATE_SPATIAL_FACT_FIELD_IDS
                else _probe_field(
                    field_id,
                    features=features.value,
                    referenced_state=referenced_state.value,
                    tactical_state=tactical_state.value,
                )
            )
        if len(result.samples[family]) < MAX_SAMPLES_PER_FAMILY:
            result.samples[family].append(_anonymous_sample(example))
        if example_sink is not None:
            example_sink(example)
        del example
    return result


def process_bundle(
    bundle_dir: Path,
    *,
    catalog_required_fields: Sequence[str],
) -> BundleSummary:
    return process_manifest(
        load_bundle_manifest(bundle_dir),
        catalog_required_fields=catalog_required_fields,
    )


def iter_bundle_summaries(
    bundle_dirs: Sequence[Path],
    *,
    catalog_required_fields: Sequence[str],
) -> Iterator[BundleSummary]:
    """Process and yield exactly one completed bundle summary at a time."""

    for bundle_dir in bundle_dirs:
        yield process_bundle(
            bundle_dir, catalog_required_fields=catalog_required_fields
        )


@dataclass
class AuditAggregate:
    family_counts: Counter[str] = field(default_factory=Counter)
    action_counts: dict[str, Counter[str]] = field(
        default_factory=lambda: defaultdict(Counter)
    )
    joins_succeeded: int = 0
    joins_failed: int = 0
    temporal_violations: int = 0
    jsonl_rows_streamed: int = 0
    gzip_rows_streamed: int = 0
    gzip_rows_json_decoded: int = 0
    bundles_processed: int = 0
    format_versions: Counter[str] = field(default_factory=Counter)
    parser_versions: Counter[str] = field(default_factory=Counter)
    field_coverage: dict[str, dict[str, CountMetric]] = field(
        default_factory=lambda: defaultdict(lambda: defaultdict(CountMetric))
    )
    structural_fields: dict[str, CountMetric] = field(
        default_factory=lambda: defaultdict(CountMetric)
    )
    samples: dict[str, list[dict[str, Any]]] = field(
        default_factory=lambda: defaultdict(list)
    )

    def merge(self, bundle: BundleSummary) -> None:
        self.bundles_processed += 1
        self.joins_succeeded += bundle.joins_succeeded
        self.jsonl_rows_streamed += bundle.jsonl_rows_streamed
        self.gzip_rows_streamed += bundle.gzip_rows_streamed
        self.gzip_rows_json_decoded += bundle.gzip_rows_json_decoded
        self.format_versions[bundle.export_format_version] += 1
        self.parser_versions[bundle.parser_schema_version] += 1
        self.family_counts.update(bundle.family_counts)
        for family, counts in bundle.action_counts.items():
            self.action_counts[family].update(counts)
        for field_id, families in bundle.field_coverage.items():
            for family, metric in families.items():
                target = self.field_coverage[field_id][family]
                target.present += metric.present
                target.missing += metric.missing
                target.null += metric.null
        for path, metric in bundle.structural_fields.items():
            target = self.structural_fields[path]
            target.present += metric.present
            target.missing += metric.missing
            target.null += metric.null
        for family, samples in bundle.samples.items():
            remaining = MAX_SAMPLES_PER_FAMILY - len(self.samples[family])
            if remaining > 0:
                self.samples[family].extend(samples[:remaining])


def _capability_field_payload(
    aggregate: AuditAggregate,
    catalog: Mapping[str, Any],
) -> tuple[list[dict[str, Any]], dict[str, dict[str, dict[str, int]]]]:
    catalog_fields = {
        str(item["id"]): item
        for item in catalog.get("fields", [])
        if isinstance(item, Mapping) and isinstance(item.get("id"), str)
    }
    serialized: dict[str, dict[str, dict[str, int]]] = {}
    rows: list[dict[str, Any]] = []
    for field_id in sorted(aggregate.field_coverage):
        families: dict[str, dict[str, int]] = {}
        for family in MATERIALIZED_FAMILIES:
            metric = aggregate.field_coverage[field_id].get(family, CountMetric())
            families[family] = metric.as_dict()
        serialized[field_id] = families
        field_contract = catalog_fields.get(field_id, {})
        rows.append(
            {
                "field_id": field_id,
                "catalog_availability_status": field_contract.get(
                    "availability_status", "DESCONOCIDO"
                ),
                "by_family": families,
            }
        )
    return rows, serialized


def _build_audit_summary(
    bundles_root: Path,
    *,
    expected_bundles: int = 44,
    documented_source_demos: int = 50,
    documented_exclusions: int = 6,
    catalog_path: Path | None = None,
    progress: bool = False,
    direct_manifest: BundleManifest | None = None,
    direct_provenance: Mapping[str, str] | None = None,
) -> dict[str, Any]:
    project_root = Path(__file__).resolve().parents[2]
    catalog_file = (
        catalog_path
        or project_root / "ai_coach" / "contracts" / "capability_catalog.json"
    )
    try:
        catalog = json.loads(catalog_file.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            f"invalid capability catalog: {exc}",
            path=catalog_file,
        ) from exc
    if not isinstance(catalog, Mapping):
        raise DatasetContractError(
            "INVALID_FIELD_TYPE",
            "capability catalog must be an object",
            path=catalog_file,
        )
    required_fields = sorted(
        {
            str(field_id)
            for capability in catalog.get("capabilities", [])
            if isinstance(capability, Mapping)
            for field_id in (
                capability.get("task_eligibility", {}).get("required_fields", [])
                if isinstance(capability.get("task_eligibility"), Mapping)
                else []
            )
        }
    )
    if direct_manifest is None:
        bundle_dirs = discover_bundle_dirs(bundles_root)
        if len(bundle_dirs) != expected_bundles:
            raise DatasetContractError(
                "REQUIRED_ARTIFACT_MISSING",
                f"expected {expected_bundles} manifest-backed bundles; found {len(bundle_dirs)}",
                path=bundles_root,
            )
        bundle_summaries = iter_bundle_summaries(
            bundle_dirs, catalog_required_fields=required_fields
        )
    else:
        bundle_dirs = (direct_manifest.bundle_dir,)
        bundle_summaries = iter(
            (
                process_manifest(
                    direct_manifest, catalog_required_fields=required_fields
                ),
            )
        )
    aggregate = AuditAggregate()
    for index, bundle in enumerate(bundle_summaries, start=1):
        aggregate.merge(bundle)
        del bundle
        if progress:
            print(
                f"decision_example_progress {index}/{len(bundle_dirs)}",
                file=sys.stderr,
                flush=True,
            )

    if direct_manifest is not None:
        decision_examples = sum(aggregate.family_counts.values())
        missing_families = sorted(
            family
            for family in MATERIALIZED_FAMILIES
            if aggregate.family_counts.get(family, 0) == 0
        )
        if (
            aggregate.bundles_processed != 1
            or decision_examples == 0
            or missing_families
            or aggregate.joins_succeeded != decision_examples
            or aggregate.joins_failed != 0
        ):
            raise DatasetContractError(
                "BUNDLE_VALIDATION_FAILED",
                "direct bundle requires one processed bundle, every materialized "
                "family, and exactly one successful join per decision with no failures; "
                f"processed={aggregate.bundles_processed}, "
                f"examples={decision_examples}, joins_succeeded={aggregate.joins_succeeded}, "
                f"joins_failed={aggregate.joins_failed}, missing_families={missing_families}",
                path=bundles_root,
            )

    capability_fields, serialized_coverage = _capability_field_payload(
        aggregate, catalog
    )
    eligibility_arguments: dict[str, Any] = {}
    if direct_manifest is not None:
        eligibility_arguments["bundle_validation_reference"] = (
            "direct canonical identity + artifact bytes/SHA-256 verified"
        )
    eligibility = evaluate_task_eligibility(
        catalog,
        family_counts=aggregate.family_counts,
        field_coverage=serialized_coverage,
        bundles_processed=aggregate.bundles_processed,
        temporal_violations=aggregate.temporal_violations,
        join_failures=aggregate.joins_failed,
        **eligibility_arguments,
    )
    eligibility_statuses = Counter(item["status"] for item in eligibility)
    for status in ("eligible", "ineligible", "not_materialized"):
        eligibility_statuses.setdefault(status, 0)
    eligibility_reasons = Counter(
        reason for item in eligibility for reason in item["reason_codes"]
    )
    samples = [
        sample
        for family in MATERIALIZED_FAMILIES
        for sample in aggregate.samples.get(family, [])
    ]
    direct = direct_manifest is not None
    report = {
        "schema_id": REPORT_SCHEMA_ID,
        "version": REPORT_VERSION,
        "status": "passed",
        "training_allowed": False,
        "contracts": {
            "decision_example": DECISION_EXAMPLE_SCHEMA_ID,
            "task_eligibility": "stratai.ai_coach_task_eligibility@1",
            "contract_version": CONTRACT_VERSION,
            "catalog_schema_id": catalog.get("schema_id"),
            "catalog_version": catalog.get("version"),
        },
        "bundles": {
            "expected": expected_bundles,
            "found": len(bundle_dirs),
            "processed": aggregate.bundles_processed,
            "export_format_versions": dict(sorted(aggregate.format_versions.items())),
            "parser_schema_versions": dict(sorted(aggregate.parser_versions.items())),
        },
        "joins": {
            "succeeded": aggregate.joins_succeeded,
            "failed": aggregate.joins_failed,
        },
        "temporal_checks": {
            "availability_tick_after_t0": aggregate.temporal_violations,
            "outcome_before_t0": 0,
            "oracle_fields_in_observed": 0,
        },
        "decision_examples": {
            "total": sum(aggregate.family_counts.values()),
            "by_family": {
                family: aggregate.family_counts.get(family, 0)
                for family in MATERIALIZED_FAMILIES
            },
            "actions_by_family": {
                family: dict(sorted(counts.items()))
                for family, counts in sorted(aggregate.action_counts.items())
            },
            "persisted_individual_examples": 0,
        },
        "task_eligibility": {
            "capabilities_evaluated": len(eligibility),
            "by_status": dict(sorted(eligibility_statuses.items())),
            "by_reason_code": dict(sorted(eligibility_reasons.items())),
            "results": eligibility,
        },
        "fields": {
            "structural_required": [
                {"path": path, **metric.as_dict()}
                for path, metric in sorted(aggregate.structural_fields.items())
            ],
            "capability_required": capability_fields,
        },
        "exclusions": {
            "documented_source_demos": documented_source_demos,
            "approved_bundles": 0 if direct else expected_bundles,
            "documented_source_exclusions": documented_exclusions,
            "excluded_bundles_opened": 0,
            "note": (
                "El modo directo abre unicamente el bundle canonico indicado; no descubre ni abre otros bundles."
                if direct
                else "Las exclusiones de Gate 1A se conservan fuera de analyzed_demos y no se abren en esta fase."
            ),
        },
        "streaming": {
            "artifact_discovery": (
                "explicit bundle root -> canonical/manifest.json -> canonical artifacts[]"
                if direct
                else "root manifest -> canonical_manifest descriptor -> canonical artifacts[]"
            ),
            "bundle_processing": "one_at_a_time",
            "max_bundles_with_loaded_rows": 1,
            "jsonl_method": "binary line iterator with a bounded line size",
            "gzip_method": "gzip.open binary stream; no persistent decompression",
            "jsonl_rows_streamed": aggregate.jsonl_rows_streamed,
            "gzip_rows_streamed": aggregate.gzip_rows_streamed,
            "gzip_rows_json_decoded": aggregate.gzip_rows_json_decoded,
            "all_tactical_observation_rows_streamed": True,
            "individual_examples_retained": False,
            "anonymous_sample_limit": MAX_SAMPLES_PER_FAMILY
            * len(MATERIALIZED_FAMILIES),
        },
        "samples": samples,
        "errors": [],
        "limitations": [
            "La fase valida estructura, joins, tiempo y disponibilidad; no juzga si una acción fue correcta.",
            "No se crean candidatos, recomendaciones ni labels contrafactuales.",
            "enemy_los sigue siendo un proxy geométrico parcial sin FOV ni humo completos.",
            (
                "Los bytes y hashes SHA-256 de todos los artifacts canónicos se recalculan desde el bundle directo."
                if direct
                else "Los hashes ya validados por el manifest no se recalculan para evitar repetir Gate 1A."
            ),
            "El perfil histórico aún no está materializado.",
        ],
    }
    if direct:
        report["exclusions"].update(
            {
                "approved_source_demos": 1,
                "temporary_direct_bundles_validated": 1,
                "note": (
                    "Se validó 1 demo fuente aprobada mediante 1 bundle temporal "
                    "directo; se aprobaron 0 output bundles."
                ),
            }
        )
        report["bundles"]["discovery"] = "direct_canonical_bundle"
        report["streaming"]["artifact_integrity"] = (
            "all declared artifact bytes and SHA-256 independently recalculated"
        )
        report["provenance"] = {
            "source": "explicit_direct_canonical_bundle",
            "root_manifest_used": False,
            "canonical_manifest": "canonical/manifest.json",
            **dict(direct_provenance or {}),
        }
    return report


def build_audit_summary(
    bundles_root: Path,
    *,
    expected_bundles: int = 44,
    documented_source_demos: int = 50,
    documented_exclusions: int = 6,
    catalog_path: Path | None = None,
    progress: bool = False,
) -> dict[str, Any]:
    """Audit the approved root-manifest corpus without changing its discovery."""

    return _build_audit_summary(
        bundles_root,
        expected_bundles=expected_bundles,
        documented_source_demos=documented_source_demos,
        documented_exclusions=documented_exclusions,
        catalog_path=catalog_path,
        progress=progress,
    )


def build_direct_bundle_audit_summary(
    bundle_dir: Path,
    *,
    expected_match_id: str,
    expected_demo_checksum: str,
    catalog_path: Path | None = None,
    progress: bool = False,
) -> dict[str, Any]:
    """Audit exactly one explicit canonical-only smoke bundle, fail closed."""

    manifest = load_direct_bundle_manifest(
        bundle_dir,
        expected_match_id=expected_match_id,
        expected_demo_checksum=expected_demo_checksum,
    )
    return _build_audit_summary(
        bundle_dir,
        expected_bundles=1,
        documented_source_demos=1,
        documented_exclusions=0,
        catalog_path=catalog_path,
        progress=progress,
        direct_manifest=manifest,
        direct_provenance={
            "expected_match_id": expected_match_id,
            "expected_demo_checksum_sha256": expected_demo_checksum,
        },
    )


def _resolve_output(path: Path) -> Path:
    project_root = Path(__file__).resolve().parents[2]
    evidence_root = (project_root / "docs" / "evidence").resolve()
    output = path.resolve()
    try:
        output.relative_to(evidence_root)
    except ValueError as exc:
        raise DatasetContractError(
            "REQUIRED_ARTIFACT_MISSING",
            "CLI output must stay under StratAI/docs/evidence",
            path=output,
        ) from exc
    return output


def parse_args(argv: Sequence[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("bundles_root", type=Path)
    parser.add_argument("--expected-bundles", type=int, default=44)
    parser.add_argument("--documented-source-demos", type=int, default=50)
    parser.add_argument("--documented-exclusions", type=int, default=6)
    parser.add_argument("--catalog", type=Path)
    parser.add_argument("--direct-bundle", action="store_true")
    parser.add_argument("--expected-match-id")
    parser.add_argument("--expected-demo-checksum")
    parser.add_argument(
        "--summary-only",
        action="store_true",
        help="retained for an explicit no-dataset invocation",
    )
    parser.add_argument("--progress", action="store_true")
    parser.add_argument("--output", type=Path, required=True)
    return parser.parse_args(argv)


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(argv)
    try:
        output = _resolve_output(args.output)
        if args.direct_bundle:
            if not args.expected_match_id or not args.expected_demo_checksum:
                raise DatasetContractError(
                    "INVALID_FIELD_TYPE",
                    "--direct-bundle requires --expected-match-id and --expected-demo-checksum",
                    path=args.bundles_root,
                )
            report = build_direct_bundle_audit_summary(
                args.bundles_root.resolve(),
                expected_match_id=args.expected_match_id,
                expected_demo_checksum=args.expected_demo_checksum,
                catalog_path=args.catalog.resolve() if args.catalog else None,
                progress=args.progress,
            )
        else:
            if args.expected_match_id or args.expected_demo_checksum:
                raise DatasetContractError(
                    "INVALID_FIELD_TYPE",
                    "direct identity arguments require --direct-bundle",
                    path=args.bundles_root,
                )
            report = build_audit_summary(
                args.bundles_root.resolve(),
                expected_bundles=args.expected_bundles,
                documented_source_demos=args.documented_source_demos,
                documented_exclusions=args.documented_exclusions,
                catalog_path=args.catalog.resolve() if args.catalog else None,
                progress=args.progress,
            )
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(
            json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )
    except DatasetContractError as exc:
        print(f"decision_example_audit_failed: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
