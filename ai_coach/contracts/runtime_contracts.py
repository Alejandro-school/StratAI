"""Executable, dependency-free AI Coach runtime-contract validation."""

from __future__ import annotations

import json
import re
from collections.abc import Iterable, Mapping, Sequence
from pathlib import Path
from typing import Any

DECISION_EXAMPLE_SCHEMA_ID = "stratai.ai_coach_decision_example@1"
TASK_ELIGIBILITY_SCHEMA_ID = "stratai.ai_coach_task_eligibility@1"
OBSERVED_VISUAL_CONTACT_SCHEMA_ID = "stratai.observed_visual_contact@1"
OBSERVED_NAVIGATION_PATH_SCHEMA_ID = "stratai.observed_navigation_path@2"
OBSERVED_NAVIGATION_PATH_CONTRACT_VERSION = "2.0.0"
OBSERVED_WEAPON_STATE_SCHEMA_ID = "stratai.observed_weapon_state@1"
OBSERVED_WEAPON_STATE_CONTRACT_VERSION = "1.0.0"
OBSERVED_UTILITY_INITIAL_VELOCITY_SCHEMA_ID = (
    "stratai.observed_utility_initial_velocity@1"
)
OBSERVED_UTILITY_INITIAL_VELOCITY_CONTRACT_VERSION = "1.0.0"
OBSERVED_LAST_KNOWN_POSITION_SCHEMA_ID = "stratai.observed_last_known_position@1"
OBSERVED_LAST_KNOWN_POSITION_CONTRACT_VERSION = "1.0.0"
EXPOSURE_ANGLE_MAP_PACKAGE_SCHEMA_ID = "stratai.exposure_angle_map_package@1"
EXPOSURE_ANGLE_SUPPORT_MANIFEST_SCHEMA_ID = "stratai.exposure_angle_support_manifest@1"
OBSERVED_EXPOSURE_ANGLE_DIAGNOSTIC_SCHEMA_ID = (
    "stratai.observed_exposure_angle_diagnostic@1"
)
OBSERVED_EXPOSURE_ANGLE_DIAGNOSTIC_V2_SCHEMA_ID = (
    "stratai.observed_exposure_angle_diagnostic@2"
)
EXPOSURE_ANGLE_CONTRACT_VERSION = "1.0.0"
COVER_MAP_PACKAGE_SCHEMA_ID = "stratai.cover_map_package@1"
COVER_SUPPORT_MANIFEST_SCHEMA_ID = "stratai.cover_support_manifest@1"
OBSERVED_COVER_DIAGNOSTIC_SCHEMA_ID = "stratai.observed_cover_diagnostic@1"
OBSERVED_COVER_DIAGNOSTIC_V2_SCHEMA_ID = "stratai.observed_cover_diagnostic@2"
COVER_CONTRACT_VERSION = "1.0.0"
EXIT_ROUTE_MAP_PACKAGE_SCHEMA_ID = "stratai.exit_route_map_package@1"
EXIT_ROUTE_SUPPORT_MANIFEST_SCHEMA_ID = "stratai.exit_route_support_manifest@1"
OBSERVED_EXIT_ROUTE_DIAGNOSTIC_SCHEMA_ID = "stratai.observed_exit_route_diagnostic@1"
OBSERVED_EXIT_ROUTE_DIAGNOSTIC_V2_SCHEMA_ID = "stratai.observed_exit_route_diagnostic@2"
EXIT_ROUTE_CONTRACT_VERSION = "1.0.0"
SPATIAL_FACT_CONTRACT_VERSION = "2.0.0"
SPATIAL_HUMAN_REVIEW_PROTOCOL_SCHEMA_ID = "stratai.spatial_human_review_protocol@1"
SPATIAL_HUMAN_REVIEW_BATCH_SCHEMA_ID = "stratai.spatial_human_review_batch@1"
SPATIAL_HUMAN_REVIEW_RESPONSE_SCHEMA_ID = "stratai.spatial_human_review_response@1"
SPATIAL_HUMAN_REVIEW_ADJUDICATION_SCHEMA_ID = (
    "stratai.spatial_human_review_adjudication@1"
)
SPATIAL_HUMAN_REVIEW_CONTRACT_VERSION = "1.0.0"
SPATIAL_CONTEXT_REVIEW_PROTOCOL_SCHEMA_ID = "stratai.spatial_context_review_protocol@2"
SPATIAL_CONTEXT_REVIEW_BATCH_SCHEMA_ID = "stratai.spatial_context_review_batch@2"
SPATIAL_CONTEXT_REVIEW_RESPONSE_SCHEMA_ID = "stratai.spatial_context_review_response@2"
SPATIAL_CONTEXT_REVIEW_ADJUDICATION_SCHEMA_ID = (
    "stratai.spatial_context_review_adjudication@2"
)
SPATIAL_CONTEXT_REVIEW_CONTRACT_VERSION = "2.0.0"
REASON_CODES_SCHEMA_ID = "stratai.ai_coach_reason_codes@1"
CONTRACT_VERSION = "1.0.0"
CATALOG_SCHEMA_ID = "stratai.ai_coach_capability_catalog@1"
ALLOWED_DECISION_FAMILIES = {
    "spacing_or_trade_connection",
    "peek_hold_or_reposition",
}
ALLOWED_ELIGIBILITY_STATES = {"eligible", "ineligible", "not_materialized"}
ALLOWED_AVAILABILITY_STATES = {"observed", "derived", "inferred", "unavailable"}

_CONTRACTS_DIR = Path(__file__).resolve().parent
_REASON_CODES_PATH = _CONTRACTS_DIR / "reason_codes.json"
_DECISION_SCHEMA_PATH = _CONTRACTS_DIR / "decision_example.schema.json"
_ELIGIBILITY_SCHEMA_PATH = _CONTRACTS_DIR / "task_eligibility.schema.json"
_VISUAL_CONTACT_SCHEMA_PATH = _CONTRACTS_DIR / "observed_visual_contact.schema.json"
_NAVIGATION_PATH_SCHEMA_PATH = _CONTRACTS_DIR / "observed_navigation_path.schema.json"
_WEAPON_STATE_SCHEMA_PATH = _CONTRACTS_DIR / "observed_weapon_state.schema.json"
_UTILITY_INITIAL_VELOCITY_SCHEMA_PATH = (
    _CONTRACTS_DIR / "observed_utility_initial_velocity.schema.json"
)
_LAST_KNOWN_POSITION_SCHEMA_PATH = (
    _CONTRACTS_DIR / "observed_last_known_position.schema.json"
)
_EXPOSURE_ANGLE_MAP_PACKAGE_SCHEMA_PATH = (
    _CONTRACTS_DIR / "exposure_angle_map_package.schema.json"
)
_EXPOSURE_ANGLE_SUPPORT_MANIFEST_SCHEMA_PATH = (
    _CONTRACTS_DIR / "exposure_angle_support_manifest.schema.json"
)
_OBSERVED_EXPOSURE_ANGLE_SCHEMA_PATH = (
    _CONTRACTS_DIR / "observed_exposure_angle_diagnostic.schema.json"
)
_OBSERVED_EXPOSURE_ANGLE_V2_SCHEMA_PATH = (
    _CONTRACTS_DIR / "observed_exposure_angle_diagnostic_v2.schema.json"
)
_COVER_MAP_PACKAGE_SCHEMA_PATH = _CONTRACTS_DIR / "cover_map_package.schema.json"
_COVER_SUPPORT_MANIFEST_SCHEMA_PATH = (
    _CONTRACTS_DIR / "cover_support_manifest.schema.json"
)
_OBSERVED_COVER_SCHEMA_PATH = _CONTRACTS_DIR / "observed_cover_diagnostic.schema.json"
_OBSERVED_COVER_V2_SCHEMA_PATH = (
    _CONTRACTS_DIR / "observed_cover_diagnostic_v2.schema.json"
)
_EXIT_ROUTE_MAP_PACKAGE_SCHEMA_PATH = (
    _CONTRACTS_DIR / "exit_route_map_package.schema.json"
)
_EXIT_ROUTE_SUPPORT_MANIFEST_SCHEMA_PATH = (
    _CONTRACTS_DIR / "exit_route_support_manifest.schema.json"
)
_OBSERVED_EXIT_ROUTE_SCHEMA_PATH = (
    _CONTRACTS_DIR / "observed_exit_route_diagnostic.schema.json"
)
_OBSERVED_EXIT_ROUTE_V2_SCHEMA_PATH = (
    _CONTRACTS_DIR / "observed_exit_route_diagnostic_v2.schema.json"
)
_SPATIAL_HUMAN_REVIEW_PROTOCOL_SCHEMA_PATH = (
    _CONTRACTS_DIR / "spatial_human_review_protocol.schema.json"
)
_SPATIAL_HUMAN_REVIEW_BATCH_SCHEMA_PATH = (
    _CONTRACTS_DIR / "spatial_human_review_batch.schema.json"
)
_SPATIAL_HUMAN_REVIEW_RESPONSE_SCHEMA_PATH = (
    _CONTRACTS_DIR / "spatial_human_review_response.schema.json"
)
_SPATIAL_HUMAN_REVIEW_ADJUDICATION_SCHEMA_PATH = (
    _CONTRACTS_DIR / "spatial_human_review_adjudication.schema.json"
)
_SPATIAL_CONTEXT_REVIEW_PROTOCOL_SCHEMA_PATH = (
    _CONTRACTS_DIR / "spatial_context_review_protocol.schema.json"
)
_SPATIAL_CONTEXT_REVIEW_BATCH_SCHEMA_PATH = (
    _CONTRACTS_DIR / "spatial_context_review_batch.schema.json"
)
_SPATIAL_CONTEXT_REVIEW_RESPONSE_SCHEMA_PATH = (
    _CONTRACTS_DIR / "spatial_context_review_response.schema.json"
)
_SPATIAL_CONTEXT_REVIEW_ADJUDICATION_SCHEMA_PATH = (
    _CONTRACTS_DIR / "spatial_context_review_adjudication.schema.json"
)

_WEAPON_LEDGER_KEYS = {
    "schema_id",
    "contract_version",
    "identity_semantics",
    "causal_role",
    "sampling",
    "rows",
}
_WEAPON_ROW_KEYS = {
    "schema_id",
    "contract_version",
    "identity_semantics",
    "causal_role",
    "join_keys",
    "round_number",
    "tick",
    "t0_tick",
    "active_weapon",
    "ammo_in_magazine",
    "ammo_reserve",
    "is_reloading",
    "abstention_reasons",
}
_WEAPON_JOIN_KEYS = {"match_id", "round_id", "player_id", "observation_id"}
_WEAPON_FACT_KEYS = {"status", "value", "availability_tick", "provenance"}
_WEAPON_PROVENANCE_KEYS = {"source", "source_tick", "reconciliation"}
_WEAPON_FACT_SOURCES = {
    "active_weapon": "replay_active_weapon_snapshot",
    "ammo_in_magazine": "demoinfocs_active_weapon_ammo",
    "ammo_reserve": "demoinfocs_active_weapon_ammo",
    "is_reloading": "demoinfocs_causal_reload_tracker",
}
_WEAPON_RECONCILIATIONS = {
    "not_applicable",
    "confirmed_weapon_fire",
    "confirmed_weapon_reload",
    "ambiguous",
    "unavailable",
}
_UNAVAILABLE_WEAPON_RECONCILIATIONS = {
    "not_applicable",
    "ambiguous",
    "unavailable",
}
_WEAPON_NAME_SENTINELS = {
    "",
    "<nil>",
    "invalid",
    "none",
    "null",
    "unavailable",
    "unknown",
    "weapon_none",
}
_WEAPON_PARTICIPANT_ALIAS = re.compile(r"^participant-[0-9]{3,}$")
_WEAPON_ABSTENTION_REASON = re.compile(r"^[a-z0-9_]+(?::[a-z0-9_]+)?$")

_DECISION_REQUIRED = {
    "schema_id",
    "contract_version",
    "example_id",
    "decision_family",
    "join_keys",
    "t0_tick",
    "observed",
    "oracle",
    "outcomes",
    "historical_profile",
    "references",
    "evidence",
}
_DECISION_OPTIONAL = {"limitations"}
_ELIGIBILITY_REQUIRED = {
    "schema_id",
    "contract_version",
    "catalog_schema_id",
    "catalog_version",
    "capability_id",
    "status",
    "reason_codes",
    "missing_required_fields",
    "materialized_decision_types",
    "blockers",
    "evidence_used",
}
_JOIN_KEYS = {
    "match_id",
    "decision_id",
    "round_number",
    "actor_player_id",
    "observed_state_ref",
}
_OBSERVED_KEYS = {"action", "referenced_state", "tactical_state", "context", "quality"}
_IDENTIFIER_KEYS = {
    "match_id",
    "decision_id",
    "round_id",
    "state_id",
    "player_id",
    "actor_player_id",
    "observer_id",
    "subject_id",
    "source_record_id",
    "source_event_ids",
}
_FUTURE_OR_ORACLE_KEYS = {
    "oracle",
    "oracle_context",
    "oracle_enemy_state",
    "outcome",
    "outcomes",
    "outcome_tick",
    "winner_observed",
    "loser_count",
    "terminal_kill_count",
    "trade_completion_count",
    "survival_status",
    "disengagement_status",
    "horizons",
}


class RuntimeContractError(ValueError):
    """Raised when a runtime contract fails closed."""


def _fail(message: str) -> None:
    raise RuntimeContractError(message)


def _is_int(value: Any) -> bool:
    return isinstance(value, int) and not isinstance(value, bool)


def _require_object(value: Any, location: str) -> Mapping[str, Any]:
    if not isinstance(value, Mapping):
        _fail(f"{location}: expected an object")
    return value


def _require_exact_keys(
    value: Mapping[str, Any],
    required: set[str],
    optional: set[str],
    location: str,
) -> None:
    missing = sorted(required - set(value))
    unknown = sorted(set(value) - required - optional)
    if missing:
        _fail(f"{location}: missing required fields {', '.join(missing)}")
    if unknown:
        _fail(f"{location}: unknown fields {', '.join(unknown)}")


def _require_non_empty_string(value: Any, location: str) -> str:
    if not isinstance(value, str) or not value:
        _fail(f"{location}: expected a non-empty string")
    return value


def _require_string_list(
    value: Any, location: str, *, non_empty: bool = False
) -> list[str]:
    if not isinstance(value, list) or any(
        not isinstance(item, str) or not item for item in value
    ):
        _fail(f"{location}: expected a list of non-empty strings")
    if non_empty and not value:
        _fail(f"{location}: expected at least one value")
    if len(value) != len(set(value)):
        _fail(f"{location}: duplicate values are forbidden")
    if value != sorted(value):
        _fail(f"{location}: values must be sorted deterministically")
    return value


def _nested_keys(value: Any) -> Iterable[str]:
    if isinstance(value, Mapping):
        for key, child in value.items():
            yield str(key)
            yield from _nested_keys(child)
    elif isinstance(value, list):
        for child in value:
            yield from _nested_keys(child)


def _validate_observed_safety(observed: Mapping[str, Any]) -> None:
    for key in _nested_keys(observed):
        lowered = key.lower()
        if lowered in _FUTURE_OR_ORACLE_KEYS or lowered.startswith(
            ("oracle_", "future_")
        ):
            _fail(f"observed: oracle/future field is forbidden: {key}")
        if lowered in _IDENTIFIER_KEYS or lowered.endswith(
            ("_id", "_ids", "_ref", "_refs")
        ):
            _fail(f"observed: join identifier is forbidden as a feature: {key}")


def _validate_availability_block(value: Any, location: str, t0_tick: int) -> None:
    block = _require_object(value, location)
    _require_exact_keys(
        block, {"availability_tick", "status", "values"}, set(), location
    )
    availability_tick = block["availability_tick"]
    if not _is_int(availability_tick) or availability_tick < 0:
        _fail(f"{location}.availability_tick: expected a non-negative integer")
    if availability_tick > t0_tick:
        _fail(f"{location}.availability_tick: must be <= t0_tick")
    if block["status"] not in ALLOWED_AVAILABILITY_STATES:
        _fail(f"{location}.status: invalid availability state")
    _require_object(block["values"], f"{location}.values")


def _validate_artifact_reference(value: Any, location: str) -> None:
    ref = _require_object(value, location)
    required = {"artifact_type", "schema_id", "path", "line"}
    _require_exact_keys(ref, required, set(), location)
    for key in ("artifact_type", "schema_id", "path"):
        _require_non_empty_string(ref[key], f"{location}.{key}")
    if not _is_int(ref["line"]) or ref["line"] < 1:
        _fail(f"{location}.line: expected a positive integer")


def is_non_ammunition_weapon(active_weapon: str) -> bool:
    """Return whether a normalized CS2 equipment name has no ammo/reload facts."""

    normalized = "".join(
        character for character in active_weapon.casefold() if character.isalnum()
    )
    return (
        normalized == "c4"
        or "knife" in normalized
        or "bayonet" in normalized
        or "karambit" in normalized
        or "dagger" in normalized
        or "bomb" in normalized
        or "grenade" in normalized
        or "flash" in normalized
        or "zeus" in normalized
        or "taser" in normalized
        or normalized
        in {
            "decoy",
            "flashbang",
            "incendiary",
            "molotov",
            "smoke",
        }
    )


def _validate_weapon_fact(
    value: Any,
    *,
    field_name: str,
    value_kind: str,
    t0_tick: int,
    location: str,
) -> Mapping[str, Any]:
    fact = _require_object(value, location)
    _require_exact_keys(fact, _WEAPON_FACT_KEYS, set(), location)
    status = fact["status"]
    if status not in {"observed", "unavailable"}:
        _fail(f"{location}.status: expected observed or unavailable")

    observed_value = fact["value"]
    availability_tick = fact["availability_tick"]
    provenance = _require_object(fact["provenance"], f"{location}.provenance")
    _require_exact_keys(
        provenance,
        _WEAPON_PROVENANCE_KEYS,
        set(),
        f"{location}.provenance",
    )
    source = provenance["source"]
    source_tick = provenance["source_tick"]
    reconciliation = provenance["reconciliation"]
    if source not in {*_WEAPON_FACT_SOURCES.values(), "unavailable"}:
        _fail(f"{location}.provenance.source: unsupported value")
    if reconciliation not in _WEAPON_RECONCILIATIONS:
        _fail(f"{location}.provenance.reconciliation: unsupported value")

    if status == "unavailable":
        if observed_value is not None or availability_tick is not None:
            _fail(
                f"{location}: unavailable facts require null value and availability_tick"
            )
        if source != "unavailable" or source_tick is not None:
            _fail(
                f"{location}.provenance: unavailable facts require null-source provenance"
            )
        if reconciliation not in _UNAVAILABLE_WEAPON_RECONCILIATIONS:
            _fail(
                f"{location}.provenance.reconciliation: incompatible with unavailable"
            )
        return fact

    if not _is_int(availability_tick) or availability_tick < 0:
        _fail(f"{location}.availability_tick: expected a non-negative integer")
    if availability_tick > t0_tick:
        _fail(f"{location}.availability_tick: must be <= t0_tick")
    if source != _WEAPON_FACT_SOURCES[field_name]:
        _fail(f"{location}.provenance.source: incompatible with {field_name}")
    if not _is_int(source_tick) or source_tick < 0:
        _fail(f"{location}.provenance.source_tick: expected a non-negative integer")
    if source_tick != availability_tick:
        _fail(f"{location}.provenance.source_tick: must equal availability_tick")
    if reconciliation not in {
        "not_applicable",
        "confirmed_weapon_fire",
        "confirmed_weapon_reload",
    }:
        _fail(f"{location}.provenance.reconciliation: incompatible with observed")

    if value_kind == "string":
        weapon = _require_non_empty_string(observed_value, f"{location}.value")
        if weapon.casefold().strip() in _WEAPON_NAME_SENTINELS:
            _fail(f"{location}.value: sentinel weapon names are unavailable")
    elif value_kind == "integer":
        if not _is_int(observed_value) or observed_value < 0:
            _fail(f"{location}.value: expected a non-negative observed integer")
    elif value_kind == "boolean":
        if not isinstance(observed_value, bool):
            _fail(f"{location}.value: expected an observed boolean")
    else:  # pragma: no cover - guarded by internal callers
        raise AssertionError(f"unsupported weapon fact kind: {value_kind}")
    return fact


def validate_observed_weapon_state(row: Mapping[str, Any]) -> None:
    """Validate one causal 16 Hz weapon-state candidate row."""

    value = _require_object(row, "ObservedWeaponState")
    _require_exact_keys(value, _WEAPON_ROW_KEYS, set(), "ObservedWeaponState")
    if value["schema_id"] != OBSERVED_WEAPON_STATE_SCHEMA_ID:
        _fail(
            f"ObservedWeaponState.schema_id: expected {OBSERVED_WEAPON_STATE_SCHEMA_ID}"
        )
    if value["contract_version"] != OBSERVED_WEAPON_STATE_CONTRACT_VERSION:
        _fail(
            "ObservedWeaponState.contract_version: expected "
            f"{OBSERVED_WEAPON_STATE_CONTRACT_VERSION}"
        )
    if value["identity_semantics"] != "join_only":
        _fail("ObservedWeaponState.identity_semantics: expected join_only")
    if value["causal_role"] != "model_input_observation":
        _fail("ObservedWeaponState.causal_role: expected model_input_observation")

    joins = _require_object(value["join_keys"], "ObservedWeaponState.join_keys")
    _require_exact_keys(
        joins, _WEAPON_JOIN_KEYS, set(), "ObservedWeaponState.join_keys"
    )
    for key in _WEAPON_JOIN_KEYS:
        _require_non_empty_string(joins[key], f"ObservedWeaponState.join_keys.{key}")
    if _WEAPON_PARTICIPANT_ALIAS.fullmatch(str(joins["player_id"])) is None:
        _fail("ObservedWeaponState.join_keys.player_id: expected an opaque alias")

    round_number = value["round_number"]
    if not _is_int(round_number) or round_number < 1:
        _fail("ObservedWeaponState.round_number: expected a positive integer")
    tick = value["tick"]
    t0_tick = value["t0_tick"]
    if not _is_int(tick) or tick < 0:
        _fail("ObservedWeaponState.tick: expected a non-negative integer")
    if not _is_int(t0_tick) or t0_tick < 0:
        _fail("ObservedWeaponState.t0_tick: expected a non-negative integer")
    if tick != t0_tick:
        _fail("ObservedWeaponState.tick: must equal t0_tick for the sampled row")
    expected_observation_id = (
        f"{joins['round_id']}:weapon-state:{tick}:{joins['player_id']}"
    )
    if joins["observation_id"] != expected_observation_id:
        _fail("ObservedWeaponState.join_keys.observation_id: must be deterministic")

    active_weapon = _validate_weapon_fact(
        value["active_weapon"],
        field_name="active_weapon",
        value_kind="string",
        t0_tick=t0_tick,
        location="ObservedWeaponState.active_weapon",
    )
    magazine = _validate_weapon_fact(
        value["ammo_in_magazine"],
        field_name="ammo_in_magazine",
        value_kind="integer",
        t0_tick=t0_tick,
        location="ObservedWeaponState.ammo_in_magazine",
    )
    reserve = _validate_weapon_fact(
        value["ammo_reserve"],
        field_name="ammo_reserve",
        value_kind="integer",
        t0_tick=t0_tick,
        location="ObservedWeaponState.ammo_reserve",
    )
    reloading = _validate_weapon_fact(
        value["is_reloading"],
        field_name="is_reloading",
        value_kind="boolean",
        t0_tick=t0_tick,
        location="ObservedWeaponState.is_reloading",
    )
    reasons = _require_string_list(
        value["abstention_reasons"], "ObservedWeaponState.abstention_reasons"
    )
    if any(_WEAPON_ABSTENTION_REASON.fullmatch(reason) is None for reason in reasons):
        _fail("ObservedWeaponState.abstention_reasons: expected canonical reason codes")

    dependent_facts = (magazine, reserve, reloading)
    if active_weapon["status"] == "unavailable" and any(
        fact["status"] == "observed" for fact in dependent_facts
    ):
        _fail(
            "ObservedWeaponState: ammo/reload cannot be observed without an active weapon"
        )
    if active_weapon["status"] == "observed" and is_non_ammunition_weapon(
        str(active_weapon["value"])
    ):
        if any(fact["status"] == "observed" for fact in dependent_facts):
            _fail(
                "ObservedWeaponState: knife, C4 and utility cannot publish ammo/reload"
            )
        if any(
            fact["provenance"]["reconciliation"] != "not_applicable"
            for fact in dependent_facts
        ):
            _fail(
                "ObservedWeaponState: non-ammunition facts require not_applicable provenance"
            )
    reconciliations = {
        field: value[field]["provenance"]["reconciliation"]
        for field in _WEAPON_FACT_SOURCES
    }
    for field in ("ammo_in_magazine", "ammo_reserve", "is_reloading"):
        reconciliation = reconciliations[field]
        if reconciliation.startswith("confirmed_weapon_") and (
            reconciliations["active_weapon"] != reconciliation
        ):
            _fail(
                "ObservedWeaponState: event confirmation requires matching active weapon"
            )
    if reconciliations["is_reloading"] == "confirmed_weapon_fire":
        _fail("ObservedWeaponState: weapon fire cannot confirm reload state")
    if (
        reconciliations["is_reloading"] == "confirmed_weapon_reload"
        and reloading["value"] is not True
    ):
        _fail("ObservedWeaponState: reload confirmation requires observed reload=true")


def validate_observed_weapon_state_ledger(value: Mapping[str, Any]) -> int:
    """Validate a deterministic candidate ledger and return its row count."""

    ledger = _require_object(value, "ObservedWeaponStateLedger")
    _require_exact_keys(ledger, _WEAPON_LEDGER_KEYS, set(), "ObservedWeaponStateLedger")
    if ledger["schema_id"] != OBSERVED_WEAPON_STATE_SCHEMA_ID:
        _fail(
            "ObservedWeaponStateLedger.schema_id: expected "
            f"{OBSERVED_WEAPON_STATE_SCHEMA_ID}"
        )
    if ledger["contract_version"] != OBSERVED_WEAPON_STATE_CONTRACT_VERSION:
        _fail(
            "ObservedWeaponStateLedger.contract_version: expected "
            f"{OBSERVED_WEAPON_STATE_CONTRACT_VERSION}"
        )
    if ledger["identity_semantics"] != "join_only":
        _fail("ObservedWeaponStateLedger.identity_semantics: expected join_only")
    if ledger["causal_role"] != "model_input_observation":
        _fail("ObservedWeaponStateLedger.causal_role: expected model_input_observation")
    sampling = _require_object(ledger["sampling"], "ObservedWeaponStateLedger.sampling")
    _require_exact_keys(
        sampling,
        {"target_hz", "strategy"},
        set(),
        "ObservedWeaponStateLedger.sampling",
    )
    if sampling["target_hz"] != 16:
        _fail("ObservedWeaponStateLedger.sampling.target_hz: expected 16")
    if sampling["strategy"] != "round_anchored_right_closed_no_carry_forward":
        _fail("ObservedWeaponStateLedger.sampling.strategy: unsupported value")

    rows = ledger["rows"]
    if not isinstance(rows, list):
        _fail("ObservedWeaponStateLedger.rows: expected a list")
    seen_observation_ids: set[str] = set()
    seen_samples: set[tuple[str, str, str, int]] = set()
    sort_keys: list[tuple[str, int, int, str, str]] = []
    for index, row in enumerate(rows):
        try:
            validate_observed_weapon_state(row)
        except RuntimeContractError as exc:
            _fail(f"ObservedWeaponStateLedger.rows[{index}]: {exc}")
        joins = row["join_keys"]
        observation_id = str(joins["observation_id"])
        if observation_id in seen_observation_ids:
            _fail(
                "ObservedWeaponStateLedger.rows: duplicate observation_id "
                f"{observation_id}"
            )
        seen_observation_ids.add(observation_id)
        sample_key = (
            str(joins["match_id"]),
            str(joins["round_id"]),
            str(joins["player_id"]),
            int(row["tick"]),
        )
        if sample_key in seen_samples:
            _fail("ObservedWeaponStateLedger.rows: duplicate player/tick sample")
        seen_samples.add(sample_key)
        sort_keys.append(
            (
                str(joins["match_id"]),
                int(row["round_number"]),
                int(row["tick"]),
                str(joins["player_id"]),
                observation_id,
            )
        )
    if sort_keys != sorted(sort_keys):
        _fail("ObservedWeaponStateLedger.rows: rows must be ordered deterministically")
    return len(rows)


def validate_decision_example(example: Mapping[str, Any]) -> None:
    """Validate one example, including causal partition separation."""

    value = _require_object(example, "DecisionExample")
    _require_exact_keys(
        value, _DECISION_REQUIRED, _DECISION_OPTIONAL, "DecisionExample"
    )
    if value["schema_id"] != DECISION_EXAMPLE_SCHEMA_ID:
        _fail(f"DecisionExample.schema_id: expected {DECISION_EXAMPLE_SCHEMA_ID}")
    if value["contract_version"] != CONTRACT_VERSION:
        _fail(f"DecisionExample.contract_version: expected {CONTRACT_VERSION}")
    _require_non_empty_string(value["example_id"], "DecisionExample.example_id")
    if value["decision_family"] not in ALLOWED_DECISION_FAMILIES:
        _fail("DecisionExample.decision_family: unsupported value")
    t0_tick = value["t0_tick"]
    if not _is_int(t0_tick) or t0_tick < 0:
        _fail("DecisionExample.t0_tick: expected a non-negative integer")

    joins = _require_object(value["join_keys"], "DecisionExample.join_keys")
    _require_exact_keys(joins, _JOIN_KEYS, set(), "DecisionExample.join_keys")
    for key in _JOIN_KEYS - {"round_number"}:
        _require_non_empty_string(joins[key], f"DecisionExample.join_keys.{key}")
    if not _is_int(joins["round_number"]) or joins["round_number"] < 1:
        _fail("DecisionExample.join_keys.round_number: expected a positive integer")
    if value["example_id"] != joins["decision_id"]:
        _fail("DecisionExample.example_id: must equal join_keys.decision_id")

    observed = _require_object(value["observed"], "DecisionExample.observed")
    _require_exact_keys(observed, _OBSERVED_KEYS, set(), "DecisionExample.observed")
    for key in ("action", "referenced_state", "tactical_state", "context"):
        _validate_availability_block(
            observed[key], f"DecisionExample.observed.{key}", t0_tick
        )
    _require_object(observed["quality"], "DecisionExample.observed.quality")
    _validate_observed_safety(observed)

    oracle = _require_object(value["oracle"], "DecisionExample.oracle")
    _require_exact_keys(
        oracle,
        {"audit_only", "available", "status", "values"},
        set(),
        "DecisionExample.oracle",
    )
    if oracle["audit_only"] is not True or not isinstance(oracle["available"], bool):
        _fail("DecisionExample.oracle: must be audit-only with explicit availability")
    if oracle["status"] not in {"available", "unavailable"}:
        _fail("DecisionExample.oracle.status: invalid value")
    _require_object(oracle["values"], "DecisionExample.oracle.values")

    outcomes = _require_object(value["outcomes"], "DecisionExample.outcomes")
    _require_exact_keys(
        outcomes,
        {"label_or_evaluation_only", "outcome_tick", "values"},
        set(),
        "DecisionExample.outcomes",
    )
    if outcomes["label_or_evaluation_only"] is not True:
        _fail("DecisionExample.outcomes: must be label/evaluation-only")
    if not _is_int(outcomes["outcome_tick"]):
        _fail("DecisionExample.outcomes.outcome_tick: expected an integer")
    if outcomes["outcome_tick"] < t0_tick:
        _fail("DecisionExample.outcomes.outcome_tick: must be >= t0_tick")
    _require_object(outcomes["values"], "DecisionExample.outcomes.values")

    profile = _require_object(
        value["historical_profile"], "DecisionExample.historical_profile"
    )
    _require_exact_keys(
        profile,
        {"strictly_before_t0", "status", "values"},
        set(),
        "DecisionExample.historical_profile",
    )
    if (
        profile["strictly_before_t0"] is not True
        or profile["status"] != "not_materialized"
        or profile["values"] is not None
    ):
        _fail(
            "DecisionExample.historical_profile: only an explicit not_materialized partition is allowed"
        )

    references = _require_object(value["references"], "DecisionExample.references")
    _require_exact_keys(
        references,
        {"observed_action", "outcome", "quality"},
        set(),
        "DecisionExample.references",
    )
    for key, reference in references.items():
        _validate_artifact_reference(reference, f"DecisionExample.references.{key}")
    evidence = value["evidence"]
    if not isinstance(evidence, list) or len(evidence) < 5:
        _fail("DecisionExample.evidence: expected at least five artifact references")
    for index, reference in enumerate(evidence):
        _validate_artifact_reference(reference, f"DecisionExample.evidence[{index}]")
    if "limitations" in value:
        _require_string_list(value["limitations"], "DecisionExample.limitations")


def validate_decision_examples(examples: Iterable[Mapping[str, Any]]) -> int:
    """Validate an iterable and reject duplicate example IDs without retaining rows."""

    seen: set[str] = set()
    count = 0
    for example in examples:
        validate_decision_example(example)
        example_id = str(example["example_id"])
        if example_id in seen:
            _fail(f"DecisionExample.example_id: duplicate id {example_id}")
        seen.add(example_id)
        count += 1
    return count


def load_reason_codes(path: Path = _REASON_CODES_PATH) -> frozenset[str]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    if (
        payload.get("schema_id") != REASON_CODES_SCHEMA_ID
        or payload.get("version") != CONTRACT_VERSION
    ):
        _fail("reason_codes.json: schema_id or version is invalid")
    rows = payload.get("reason_codes")
    if not isinstance(rows, list):
        _fail("reason_codes.json: reason_codes must be a list")
    codes: list[str] = []
    for index, row in enumerate(rows):
        item = _require_object(row, f"reason_codes[{index}]")
        _require_exact_keys(item, {"id", "meaning"}, set(), f"reason_codes[{index}]")
        codes.append(_require_non_empty_string(item["id"], f"reason_codes[{index}].id"))
        _require_non_empty_string(item["meaning"], f"reason_codes[{index}].meaning")
    if len(codes) != len(set(codes)):
        _fail("reason_codes.json: duplicate reason code")
    if codes != sorted(codes):
        _fail("reason_codes.json: reason codes must be sorted")
    return frozenset(codes)


def validate_task_eligibility(
    eligibility: Mapping[str, Any],
    *,
    allowed_reason_codes: frozenset[str] | None = None,
) -> None:
    """Validate one deterministic per-capability eligibility result."""

    value = _require_object(eligibility, "TaskEligibility")
    _require_exact_keys(value, _ELIGIBILITY_REQUIRED, set(), "TaskEligibility")
    if value["schema_id"] != TASK_ELIGIBILITY_SCHEMA_ID:
        _fail(f"TaskEligibility.schema_id: expected {TASK_ELIGIBILITY_SCHEMA_ID}")
    if value["contract_version"] != CONTRACT_VERSION:
        _fail(f"TaskEligibility.contract_version: expected {CONTRACT_VERSION}")
    if value["catalog_schema_id"] != CATALOG_SCHEMA_ID:
        _fail(f"TaskEligibility.catalog_schema_id: expected {CATALOG_SCHEMA_ID}")
    _require_non_empty_string(
        value["catalog_version"], "TaskEligibility.catalog_version"
    )
    _require_non_empty_string(value["capability_id"], "TaskEligibility.capability_id")
    status = value["status"]
    if status not in ALLOWED_ELIGIBILITY_STATES:
        _fail("TaskEligibility.status: invalid value")
    reasons = _require_string_list(
        value["reason_codes"], "TaskEligibility.reason_codes", non_empty=True
    )
    known = allowed_reason_codes or load_reason_codes()
    unknown = sorted(set(reasons) - known)
    if unknown:
        _fail(f"TaskEligibility.reason_codes: unknown reason code {', '.join(unknown)}")
    missing = _require_string_list(
        value["missing_required_fields"], "TaskEligibility.missing_required_fields"
    )
    materialized = _require_string_list(
        value["materialized_decision_types"],
        "TaskEligibility.materialized_decision_types",
    )
    blockers = value["blockers"]
    if not isinstance(blockers, list):
        _fail("TaskEligibility.blockers: expected a list")
    for index, blocker in enumerate(blockers):
        item = _require_object(blocker, f"TaskEligibility.blockers[{index}]")
        _require_exact_keys(
            item,
            {"reason_code", "field_id", "detail"},
            set(),
            f"TaskEligibility.blockers[{index}]",
        )
        if item["reason_code"] not in known:
            _fail(f"TaskEligibility.blockers[{index}].reason_code: unknown reason code")
        if item["field_id"] is not None:
            _require_non_empty_string(
                item["field_id"], f"TaskEligibility.blockers[{index}].field_id"
            )
        _require_non_empty_string(
            item["detail"], f"TaskEligibility.blockers[{index}].detail"
        )
    evidence = value["evidence_used"]
    if not isinstance(evidence, list):
        _fail("TaskEligibility.evidence_used: expected a list")
    for index, entry in enumerate(evidence):
        item = _require_object(entry, f"TaskEligibility.evidence_used[{index}]")
        _require_exact_keys(
            item,
            {"kind", "reference", "count"},
            set(),
            f"TaskEligibility.evidence_used[{index}]",
        )
        if item["kind"] not in {
            "catalog",
            "decision_family",
            "field_coverage",
            "bundle_validation",
        }:
            _fail(f"TaskEligibility.evidence_used[{index}].kind: invalid value")
        _require_non_empty_string(
            item["reference"], f"TaskEligibility.evidence_used[{index}].reference"
        )
        if not _is_int(item["count"]) or item["count"] < 0:
            _fail(
                f"TaskEligibility.evidence_used[{index}].count: expected a non-negative integer"
            )

    if status == "eligible" and (missing or blockers):
        _fail("TaskEligibility: eligible cannot contain missing fields or blockers")
    if status == "eligible" and reasons != ["ELIGIBLE_REQUIRED_EVIDENCE_PRESENT"]:
        _fail("TaskEligibility: eligible requires its stable success reason")
    if (
        status == "not_materialized"
        and "DECISION_FAMILY_NOT_MATERIALIZED" not in reasons
    ):
        _fail(
            "TaskEligibility: not_materialized requires DECISION_FAMILY_NOT_MATERIALIZED"
        )
    if status != "not_materialized" and not materialized:
        _fail(
            "TaskEligibility: a materialized state requires at least one decision type"
        )


def validate_task_eligibilities(results: Sequence[Mapping[str, Any]]) -> int:
    known = load_reason_codes()
    seen: set[str] = set()
    for result in results:
        validate_task_eligibility(result, allowed_reason_codes=known)
        capability_id = str(result["capability_id"])
        if capability_id in seen:
            _fail(f"TaskEligibility.capability_id: duplicate id {capability_id}")
        seen.add(capability_id)
    return len(results)


def validate_runtime_contract_files() -> dict[str, int]:
    """Validate static schema metadata and reason-code references without jsonschema."""

    reason_codes = load_reason_codes()
    decision_schema = json.loads(_DECISION_SCHEMA_PATH.read_text(encoding="utf-8"))
    eligibility_schema = json.loads(
        _ELIGIBILITY_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    visual_contact_schema = json.loads(
        _VISUAL_CONTACT_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    navigation_path_schema = json.loads(
        _NAVIGATION_PATH_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    weapon_state_schema = json.loads(
        _WEAPON_STATE_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    utility_velocity_schema = json.loads(
        _UTILITY_INITIAL_VELOCITY_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    last_known_position_schema = json.loads(
        _LAST_KNOWN_POSITION_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    exposure_map_package_schema = json.loads(
        _EXPOSURE_ANGLE_MAP_PACKAGE_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    exposure_support_manifest_schema = json.loads(
        _EXPOSURE_ANGLE_SUPPORT_MANIFEST_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    observed_exposure_angle_schema = json.loads(
        _OBSERVED_EXPOSURE_ANGLE_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    observed_exposure_angle_v2_schema = json.loads(
        _OBSERVED_EXPOSURE_ANGLE_V2_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    cover_map_package_schema = json.loads(
        _COVER_MAP_PACKAGE_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    cover_support_manifest_schema = json.loads(
        _COVER_SUPPORT_MANIFEST_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    observed_cover_schema = json.loads(
        _OBSERVED_COVER_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    observed_cover_v2_schema = json.loads(
        _OBSERVED_COVER_V2_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    exit_route_map_package_schema = json.loads(
        _EXIT_ROUTE_MAP_PACKAGE_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    exit_route_support_manifest_schema = json.loads(
        _EXIT_ROUTE_SUPPORT_MANIFEST_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    observed_exit_route_schema = json.loads(
        _OBSERVED_EXIT_ROUTE_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    observed_exit_route_v2_schema = json.loads(
        _OBSERVED_EXIT_ROUTE_V2_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    spatial_review_protocol_schema = json.loads(
        _SPATIAL_HUMAN_REVIEW_PROTOCOL_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    spatial_review_batch_schema = json.loads(
        _SPATIAL_HUMAN_REVIEW_BATCH_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    spatial_review_response_schema = json.loads(
        _SPATIAL_HUMAN_REVIEW_RESPONSE_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    spatial_review_adjudication_schema = json.loads(
        _SPATIAL_HUMAN_REVIEW_ADJUDICATION_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    spatial_context_protocol_schema = json.loads(
        _SPATIAL_CONTEXT_REVIEW_PROTOCOL_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    spatial_context_batch_schema = json.loads(
        _SPATIAL_CONTEXT_REVIEW_BATCH_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    spatial_context_response_schema = json.loads(
        _SPATIAL_CONTEXT_REVIEW_RESPONSE_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    spatial_context_adjudication_schema = json.loads(
        _SPATIAL_CONTEXT_REVIEW_ADJUDICATION_SCHEMA_PATH.read_text(encoding="utf-8")
    )
    if decision_schema.get("$id") != DECISION_EXAMPLE_SCHEMA_ID:
        _fail("decision_example.schema.json: invalid $id")
    if decision_schema.get("contract_version") != CONTRACT_VERSION:
        _fail("decision_example.schema.json: invalid contract_version")
    if eligibility_schema.get("$id") != TASK_ELIGIBILITY_SCHEMA_ID:
        _fail("task_eligibility.schema.json: invalid $id")
    if eligibility_schema.get("contract_version") != CONTRACT_VERSION:
        _fail("task_eligibility.schema.json: invalid contract_version")
    if visual_contact_schema.get("$id") != OBSERVED_VISUAL_CONTACT_SCHEMA_ID:
        _fail("observed_visual_contact.schema.json: invalid $id")
    if visual_contact_schema.get("contract_version") != CONTRACT_VERSION:
        _fail("observed_visual_contact.schema.json: invalid contract_version")
    if navigation_path_schema.get("$id") != OBSERVED_NAVIGATION_PATH_SCHEMA_ID:
        _fail("observed_navigation_path.schema.json: invalid $id")
    if (
        navigation_path_schema.get("contract_version")
        != OBSERVED_NAVIGATION_PATH_CONTRACT_VERSION
    ):
        _fail("observed_navigation_path.schema.json: invalid contract_version")
    if weapon_state_schema.get("$id") != OBSERVED_WEAPON_STATE_SCHEMA_ID:
        _fail("observed_weapon_state.schema.json: invalid $id")
    if (
        weapon_state_schema.get("contract_version")
        != OBSERVED_WEAPON_STATE_CONTRACT_VERSION
    ):
        _fail("observed_weapon_state.schema.json: invalid contract_version")
    if (
        utility_velocity_schema.get("$id")
        != OBSERVED_UTILITY_INITIAL_VELOCITY_SCHEMA_ID
    ):
        _fail("observed_utility_initial_velocity.schema.json: invalid $id")
    if (
        utility_velocity_schema.get("contract_version")
        != OBSERVED_UTILITY_INITIAL_VELOCITY_CONTRACT_VERSION
    ):
        _fail("observed_utility_initial_velocity.schema.json: invalid contract_version")
    if last_known_position_schema.get("$id") != OBSERVED_LAST_KNOWN_POSITION_SCHEMA_ID:
        _fail("observed_last_known_position.schema.json: invalid $id")
    if (
        last_known_position_schema.get("contract_version")
        != OBSERVED_LAST_KNOWN_POSITION_CONTRACT_VERSION
    ):
        _fail("observed_last_known_position.schema.json: invalid contract_version")
    if exposure_map_package_schema.get("$id") != EXPOSURE_ANGLE_MAP_PACKAGE_SCHEMA_ID:
        _fail("exposure_angle_map_package.schema.json: invalid $id")
    if (
        exposure_map_package_schema.get("contract_version")
        != EXPOSURE_ANGLE_CONTRACT_VERSION
    ):
        _fail("exposure_angle_map_package.schema.json: invalid contract_version")
    if (
        exposure_support_manifest_schema.get("$id")
        != EXPOSURE_ANGLE_SUPPORT_MANIFEST_SCHEMA_ID
    ):
        _fail("exposure_angle_support_manifest.schema.json: invalid $id")
    if (
        exposure_support_manifest_schema.get("contract_version")
        != EXPOSURE_ANGLE_CONTRACT_VERSION
    ):
        _fail("exposure_angle_support_manifest.schema.json: invalid contract_version")
    if (
        observed_exposure_angle_schema.get("$id")
        != OBSERVED_EXPOSURE_ANGLE_DIAGNOSTIC_SCHEMA_ID
    ):
        _fail("observed_exposure_angle_diagnostic.schema.json: invalid $id")
    if (
        observed_exposure_angle_schema.get("contract_version")
        != EXPOSURE_ANGLE_CONTRACT_VERSION
    ):
        _fail(
            "observed_exposure_angle_diagnostic.schema.json: invalid contract_version"
        )
    if (
        observed_exposure_angle_v2_schema.get("$id")
        != OBSERVED_EXPOSURE_ANGLE_DIAGNOSTIC_V2_SCHEMA_ID
        or observed_exposure_angle_v2_schema.get("contract_version")
        != SPATIAL_FACT_CONTRACT_VERSION
    ):
        _fail("observed_exposure_angle_diagnostic_v2.schema.json: invalid metadata")
    if cover_map_package_schema.get("$id") != COVER_MAP_PACKAGE_SCHEMA_ID:
        _fail("cover_map_package.schema.json: invalid $id")
    if cover_map_package_schema.get("contract_version") != COVER_CONTRACT_VERSION:
        _fail("cover_map_package.schema.json: invalid contract_version")
    if cover_support_manifest_schema.get("$id") != COVER_SUPPORT_MANIFEST_SCHEMA_ID:
        _fail("cover_support_manifest.schema.json: invalid $id")
    if cover_support_manifest_schema.get("contract_version") != COVER_CONTRACT_VERSION:
        _fail("cover_support_manifest.schema.json: invalid contract_version")
    if observed_cover_schema.get("$id") != OBSERVED_COVER_DIAGNOSTIC_SCHEMA_ID:
        _fail("observed_cover_diagnostic.schema.json: invalid $id")
    if observed_cover_schema.get("contract_version") != COVER_CONTRACT_VERSION:
        _fail("observed_cover_diagnostic.schema.json: invalid contract_version")
    if (
        observed_cover_v2_schema.get("$id") != OBSERVED_COVER_DIAGNOSTIC_V2_SCHEMA_ID
        or observed_cover_v2_schema.get("contract_version")
        != SPATIAL_FACT_CONTRACT_VERSION
    ):
        _fail("observed_cover_diagnostic_v2.schema.json: invalid metadata")
    if exit_route_map_package_schema.get("$id") != EXIT_ROUTE_MAP_PACKAGE_SCHEMA_ID:
        _fail("exit_route_map_package.schema.json: invalid $id")
    if (
        exit_route_map_package_schema.get("contract_version")
        != EXIT_ROUTE_CONTRACT_VERSION
    ):
        _fail("exit_route_map_package.schema.json: invalid contract_version")
    if (
        exit_route_support_manifest_schema.get("$id")
        != EXIT_ROUTE_SUPPORT_MANIFEST_SCHEMA_ID
    ):
        _fail("exit_route_support_manifest.schema.json: invalid $id")
    if (
        exit_route_support_manifest_schema.get("contract_version")
        != EXIT_ROUTE_CONTRACT_VERSION
    ):
        _fail("exit_route_support_manifest.schema.json: invalid contract_version")
    if (
        observed_exit_route_schema.get("$id")
        != OBSERVED_EXIT_ROUTE_DIAGNOSTIC_SCHEMA_ID
    ):
        _fail("observed_exit_route_diagnostic.schema.json: invalid $id")
    if (
        observed_exit_route_schema.get("contract_version")
        != EXIT_ROUTE_CONTRACT_VERSION
    ):
        _fail("observed_exit_route_diagnostic.schema.json: invalid contract_version")
    if (
        observed_exit_route_v2_schema.get("$id")
        != OBSERVED_EXIT_ROUTE_DIAGNOSTIC_V2_SCHEMA_ID
        or observed_exit_route_v2_schema.get("contract_version")
        != SPATIAL_FACT_CONTRACT_VERSION
    ):
        _fail("observed_exit_route_diagnostic_v2.schema.json: invalid metadata")
    spatial_review_schemas = (
        (
            spatial_review_protocol_schema,
            SPATIAL_HUMAN_REVIEW_PROTOCOL_SCHEMA_ID,
            "spatial_human_review_protocol.schema.json",
        ),
        (
            spatial_review_batch_schema,
            SPATIAL_HUMAN_REVIEW_BATCH_SCHEMA_ID,
            "spatial_human_review_batch.schema.json",
        ),
        (
            spatial_review_response_schema,
            SPATIAL_HUMAN_REVIEW_RESPONSE_SCHEMA_ID,
            "spatial_human_review_response.schema.json",
        ),
        (
            spatial_review_adjudication_schema,
            SPATIAL_HUMAN_REVIEW_ADJUDICATION_SCHEMA_ID,
            "spatial_human_review_adjudication.schema.json",
        ),
    )
    for schema, expected_id, filename in spatial_review_schemas:
        if schema.get("$id") != expected_id:
            _fail(f"{filename}: invalid $id")
        if schema.get("contract_version") != SPATIAL_HUMAN_REVIEW_CONTRACT_VERSION:
            _fail(f"{filename}: invalid contract_version")
    spatial_context_schemas = (
        (
            spatial_context_protocol_schema,
            SPATIAL_CONTEXT_REVIEW_PROTOCOL_SCHEMA_ID,
            "spatial_context_review_protocol.schema.json",
        ),
        (
            spatial_context_batch_schema,
            SPATIAL_CONTEXT_REVIEW_BATCH_SCHEMA_ID,
            "spatial_context_review_batch.schema.json",
        ),
        (
            spatial_context_response_schema,
            SPATIAL_CONTEXT_REVIEW_RESPONSE_SCHEMA_ID,
            "spatial_context_review_response.schema.json",
        ),
        (
            spatial_context_adjudication_schema,
            SPATIAL_CONTEXT_REVIEW_ADJUDICATION_SCHEMA_ID,
            "spatial_context_review_adjudication.schema.json",
        ),
    )
    for schema, expected_id, filename in spatial_context_schemas:
        if schema.get("$id") != expected_id:
            _fail(f"{filename}: invalid $id")
        version = schema.get("properties", {}).get("contract_version", {}).get("const")
        if version != SPATIAL_CONTEXT_REVIEW_CONTRACT_VERSION:
            _fail(f"{filename}: invalid contract_version")
    for schema, _, filename in spatial_review_schemas + spatial_context_schemas:
        if (
            schema.get("x-stratai-status") != "deprecated_visual_qa_only"
            or schema.get("x-stratai-ground-truth-authorized") is not False
            or schema.get("x-stratai-evaluation-authorized") is not False
            or schema.get("x-stratai-training-authorized") is not False
        ):
            _fail(f"{filename}: invalid ground-truth boundary metadata")
    schema_codes = set(
        eligibility_schema.get("properties", {})
        .get("reason_codes", {})
        .get("items", {})
        .get("enum", [])
    )
    if schema_codes != set(reason_codes):
        _fail(
            "task_eligibility.schema.json: reason-code enum differs from reason_codes.json"
        )
    schema_ids = [
        decision_schema["$id"],
        eligibility_schema["$id"],
        visual_contact_schema["$id"],
        navigation_path_schema["$id"],
        weapon_state_schema["$id"],
        utility_velocity_schema["$id"],
        last_known_position_schema["$id"],
        exposure_map_package_schema["$id"],
        exposure_support_manifest_schema["$id"],
        observed_exposure_angle_schema["$id"],
        observed_exposure_angle_v2_schema["$id"],
        cover_map_package_schema["$id"],
        cover_support_manifest_schema["$id"],
        observed_cover_schema["$id"],
        observed_cover_v2_schema["$id"],
        exit_route_map_package_schema["$id"],
        exit_route_support_manifest_schema["$id"],
        observed_exit_route_schema["$id"],
        observed_exit_route_v2_schema["$id"],
        spatial_review_protocol_schema["$id"],
        spatial_review_batch_schema["$id"],
        spatial_review_response_schema["$id"],
        spatial_review_adjudication_schema["$id"],
        spatial_context_protocol_schema["$id"],
        spatial_context_batch_schema["$id"],
        spatial_context_response_schema["$id"],
        spatial_context_adjudication_schema["$id"],
        REASON_CODES_SCHEMA_ID,
    ]
    if len(schema_ids) != len(set(schema_ids)):
        _fail("runtime contracts: duplicate schema id")
    from ai_coach.contracts.spatial_candidate import schema_validator

    for filename in (
        "spatial_candidate_artifact.schema.json",
        "spatial_participant_bridge.schema.json",
        "spatial_candidate_manifest.schema.json",
        "spatial_candidate_manifest_v2.schema.json",
        "spatial_fact_join.schema.json",
        "decision_example_v2.schema.json",
        "decision_example_v3.schema.json",
    ):
        schema = schema_validator(filename).schema
        if schema["$id"] in schema_ids:
            _fail("runtime contracts: duplicate candidate schema id")
        schema_ids.append(schema["$id"])
    from ai_coach.contracts.validate_economy_candidate import (
        schema_validator as economy_schema_validator,
    )

    for filename in (
        "observed_economy_candidate.schema.json",
        "economy_candidate_manifest.schema.json",
        "observed_economy_candidate_v2.schema.json",
        "economy_candidate_manifest_v2.schema.json",
    ):
        schema = economy_schema_validator(filename).schema
        if (
            schema["$id"] in schema_ids
            or schema.get("x-stratai-training-authorized") is not False
        ):
            _fail("runtime contracts: invalid economic candidate boundary")
        schema_ids.append(schema["$id"])
    from ai_coach.economy.candidate_io import (
        check_schema as check_opportunity_schema,
    )
    from ai_coach.economy.candidate_io import (
        schema_validator as opportunity_schema_validator,
    )

    for filename in (
        "economy_opportunities.schema.json",
        "economy_opportunities_manifest.schema.json",
        "economy_opportunity_protocol.schema.json",
    ):
        schema = opportunity_schema_validator(filename).schema
        if (
            schema["$id"] in schema_ids
            or schema.get("x-stratai-training-authorized") is not False
        ):
            _fail("runtime contracts: invalid economic opportunity boundary")
        schema_ids.append(schema["$id"])
    protocol = json.loads(
        (Path(__file__).parent / "economy_opportunity_protocol.json").read_text(
            encoding="utf-8"
        )
    )
    check_opportunity_schema(protocol, "economy_opportunity_protocol.schema.json")
    from ai_coach.economy.native_sources import (
        schema_validator as native_schema_validator,
    )

    for filename in (
        "economy_native_sources.schema.json",
        "economy_native_sources_manifest.schema.json",
        "economy_native_reconstruction.schema.json",
    ):
        schema = native_schema_validator(filename).schema
        if (
            schema["$id"] in schema_ids
            or schema.get("x-stratai-training-authorized") is not False
        ):
            _fail("runtime contracts: invalid native economic source boundary")
        schema_ids.append(schema["$id"])
    from ai_coach.economy.rule_assets import (
        REGISTRY as rule_registry_path,
    )
    from ai_coach.economy.rule_assets import (
        SCHEMAS as rule_schemas,
    )
    from ai_coach.economy.rule_assets import (
        check_schema as check_rule_schema,
    )
    from ai_coach.economy.rule_assets import (
        schema_validator as rule_schema_validator,
    )

    for filename in sorted(rule_schemas):
        schema = rule_schema_validator(filename).schema
        if (
            schema["$id"] in schema_ids
            or schema.get("x-stratai-training-authorized") is not False
        ):
            _fail("runtime contracts: invalid economic reference boundary")
        schema_ids.append(schema["$id"])
    check_rule_schema(
        json.loads(rule_registry_path.read_text(encoding="utf-8")),
        "economy_rule_source_registry.schema.json",
    )
    from ai_coach.economy.rule_context import SCHEMAS as context_schemas
    from ai_coach.economy.rule_context import probe as context_probe
    from ai_coach.economy.rule_context import schema_validator as context_validator

    context_probe()
    for filename in sorted(context_schemas):
        schema = context_validator(filename).schema
        if (
            schema["$id"] in schema_ids
            or schema.get("x-stratai-training-authorized") is not False
        ):
            _fail("runtime contracts: invalid native rule context boundary")
        schema_ids.append(schema["$id"])
    from ai_coach.economy.session_payload import probe as session_probe
    from ai_coach.economy.session_payload import schema_validator as session_validator

    session_probe()
    schema = session_validator().schema
    if (
        schema["$id"] in schema_ids
        or schema.get("x-stratai-training-authorized") is not False
    ):
        _fail("runtime contracts: invalid session payload boundary")
    schema_ids.append(schema["$id"])
    from ai_coach.economy.resource_manifest import (
        schema_validator as manifest_validator,
    )

    schema = manifest_validator().schema
    if (
        schema["$id"] in schema_ids
        or schema.get("x-stratai-training-authorized") is not False
    ):
        _fail("runtime contracts: invalid resource manifest boundary")
    schema_ids.append(schema["$id"])
    from ai_coach.economy.transaction_trace import schema_validator as trace_validator

    schema = trace_validator().schema
    if (
        schema["$id"] in schema_ids
        or schema.get("x-stratai-training-authorized") is not False
    ):
        _fail("runtime contracts: invalid transaction trace boundary")
    schema_ids.append(schema["$id"])
    from ai_coach.economy.native_identity import schema_validator as identity_validator

    schema = identity_validator().schema
    if (
        schema["$id"] in schema_ids
        or schema.get("x-stratai-training-authorized") is not False
    ):
        _fail("runtime contracts: invalid native identity boundary")
    schema_ids.append(schema["$id"])
    from ai_coach.economy.ownership import schema_validator as ownership_validator

    for filename in (
        "economy_ownership_observations.json",
        "economy_ownership_reconciliation.json",
        "economy_entry_observations.json",
        "economy_entry_identity_links.json",
        "economy_cashflow_observations.json",
        "economy_cashflow_reconciliation.json",
    ):
        schema = ownership_validator(filename).schema
        if (
            schema["$id"] in schema_ids
            or schema.get("x-stratai-training-authorized") is not False
        ):
            _fail("runtime contracts: invalid ownership boundary")
        schema_ids.append(schema["$id"])
    from ai_coach.support.anchors import schema_validator as support_validator

    for filename in (
        "support_observations.json",
        "support_anchors.json",
        "support_anchor_protocol.json",
        "support_physical_observations.json",
        "support_physical_context.json",
        "support_navigation_observations.json",
        "support_navigation_context.json",
        "support_objective_observations.json",
        "support_objective_context.json",
        "support_c4_observations.json",
        "support_c4_context.json",
        "support_objective_clock_observations.json",
        "support_objective_clock_context.json",
        "support_attempt_origin_observations.json",
        "support_attempt_origin_context.json",
        "support_objective_position_observations.json",
        "support_objective_position_context.json",
        "support_native_motion_observations.json",
        "support_native_motion_context.json",
        "support_weapon_timing_observations.json",
        "support_weapon_timing_context.json",
        "support_visual_contact_observations.json",
        "support_visual_contact_context.json",
        "support_preparation_protocol.json",
        "support_preparation_cut.json",
        "support_preparation_pair.json",
        "support_preparation_manifest.json",
        "support_preparation_physical_cut.json",
        "support_preparation_physical_manifest.json",
        "support_preparation_objective_cut.json",
        "support_preparation_objective_manifest.json",
        "support_preparation_c4_cut.json",
        "support_preparation_c4_manifest.json",
        "support_preparation_objective_clock_cut.json",
        "support_preparation_objective_clock_manifest.json",
        "support_preparation_attempt_origin_cut.json",
        "support_preparation_attempt_origin_manifest.json",
        "support_preparation_objective_position_cut.json",
        "support_preparation_objective_position_manifest.json",
        "support_preparation_native_motion_cut.json",
        "support_preparation_native_motion_manifest.json",
    "support_preparation_weapon_timing_cut.json",
    "support_preparation_weapon_timing_manifest.json",
    "support_preparation_visual_contact_cut.json",
    "support_preparation_visual_contact_manifest.json",
        "temporal_description.json",
        "contextual_input.json",
        "contextual_temporal_interpretation.json",
        "support_inventory_observations.json",
        "contextual_review.json",
        "contextual_review_packet.json",
        "support_round_context_observations.json",
        "contextual_review_packet_v2.json",
        "contextual_review_packet_v3.json",
        "common_factual_context.json",
    ):
        schema = support_validator(filename).schema
        if (
            schema["$id"] in schema_ids
            or schema.get("x-stratai-training-authorized") is not False
        ):
            _fail("runtime contracts: invalid support boundary")
        schema_ids.append(schema["$id"])
    from ai_coach.support.preparation import protocol as preparation_protocol

    preparation_protocol()
    return {"schemas": len(schema_ids), "reason_codes": len(reason_codes)}
