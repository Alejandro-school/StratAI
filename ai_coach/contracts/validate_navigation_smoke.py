"""Validate one isolated real-demo navigation smoke artifact.

This validator is dependency-free and intentionally stricter than checking an
exit code. It validates the candidate rows, causal ticks, internal-only timing
policy, graph provenance, and the report's recomputed coverage counts.
"""

from __future__ import annotations

import argparse
import json
import math
import re
from collections import Counter
from collections.abc import Mapping
from pathlib import Path
from typing import Any

REPORT_SCHEMA_ID = "stratai.observed_navigation_single_demo_smoke@1"
NAVIGATION_SCHEMA_ID = "stratai.observed_navigation_path@2"
NAVIGATION_CONTRACT_VERSION = "2.0.0"
GRAPH_CONTRACT_VERSION = "nav_area_connections_with_polygons@2"
SOURCE_ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
PARTICIPANT_ALIAS_PATTERN = re.compile(r"^participant-[0-9]{3,}$")
NAVIGATION_SOURCE = "replay_position_pair_navigation_derivation"
NAVIGATION_INPUT_FIELDS = [
    "active_weapon_at_t0",
    "horizontal_speed_at_t0",
    "is_ducking_at_t0",
    "is_scoped_at_t0",
    "is_walking_at_t0",
    "nav_area_connections",
    "origin_availability_tick",
    "origin_position",
    "target_availability_tick",
    "target_position",
]
WEAPON_MAX_SPEED = {
    "ak-47": 215.0,
    "m4a4": 225.0,
    "m4a1": 225.0,
    "m4a1-s": 225.0,
    "galil ar": 220.0,
    "famas": 220.0,
    "sg 553": 210.0,
    "aug": 220.0,
    "awp": 200.0,
    "ssg 08": 230.0,
    "g3sg1": 215.0,
    "scar-20": 215.0,
    "mp9": 240.0,
    "mac-10": 240.0,
    "mp7": 220.0,
    "mp5-sd": 235.0,
    "ump-45": 230.0,
    "pp-bizon": 240.0,
    "p90": 230.0,
    "nova": 220.0,
    "xm1014": 215.0,
    "mag-7": 225.0,
    "sawed-off": 220.0,
    "m249": 195.0,
    "negev": 150.0,
    "glock-18": 240.0,
    "usp-s": 240.0,
    "p2000": 240.0,
    "p250": 240.0,
    "five-seven": 240.0,
    "tec-9": 240.0,
    "cz75-auto": 240.0,
    "desert eagle": 230.0,
    "dual berettas": 240.0,
    "r8 revolver": 220.0,
    "knife": 250.0,
    "zeus x27": 240.0,
}


class NavigationSmokeValidationError(ValueError):
    """Raised when the isolated smoke artifact is incomplete or unsafe."""


def _fail(message: str) -> None:
    raise NavigationSmokeValidationError(message)


def _object(value: Any, location: str) -> Mapping[str, Any]:
    if not isinstance(value, Mapping):
        _fail(f"{location}: expected object")
    return value


def _exact_keys(value: Mapping[str, Any], expected: set[str], location: str) -> None:
    missing = sorted(expected - set(value))
    unknown = sorted(set(value) - expected)
    if missing or unknown:
        _fail(f"{location}: missing={missing}, unknown={unknown}")


def _finite_number(value: Any, location: str, *, positive: bool = False) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        _fail(f"{location}: expected number")
    result = float(value)
    if not math.isfinite(result) or result < 0 or (positive and result <= 0):
        _fail(
            f"{location}: expected finite {'positive' if positive else 'non-negative'} number"
        )
    return result


def _non_empty_string(value: Any, location: str) -> str:
    if not isinstance(value, str) or not value.strip():
        _fail(f"{location}: expected non-empty string")
    return value


def _validate_distance_fact(value: Any, location: str) -> float | None:
    fact = _object(value, location)
    _exact_keys(fact, {"status", "value_world_units"}, location)
    if fact["status"] == "unavailable":
        if fact["value_world_units"] is not None:
            _fail(f"{location}: unavailable value must be null")
        return None
    if fact["status"] != "derived":
        _fail(f"{location}.status: invalid value")
    return _finite_number(fact["value_world_units"], f"{location}.value_world_units")


def _validate_timing_scenario(
    value: Any,
    location: str,
    *,
    path_distance: float | None,
) -> tuple[bool, float | None]:
    scenario = _object(value, location)
    _exact_keys(
        scenario,
        {
            "status",
            "value_milliseconds",
            "speed_world_units_per_second",
            "speed_source",
            "assumption",
            "unavailability_reason",
        },
        location,
    )
    _non_empty_string(scenario["speed_source"], f"{location}.speed_source")
    _non_empty_string(scenario["assumption"], f"{location}.assumption")
    if scenario["status"] == "unavailable":
        if (
            scenario["value_milliseconds"] is not None
            or scenario["speed_world_units_per_second"] is not None
        ):
            _fail(f"{location}: unavailable scenario published a substitute value")
        _non_empty_string(
            scenario["unavailability_reason"], f"{location}.unavailability_reason"
        )
        return False, None
    if (
        scenario["status"] != "estimated"
        or scenario["unavailability_reason"] is not None
    ):
        _fail(f"{location}: invalid estimated scenario")
    milliseconds = _finite_number(
        scenario["value_milliseconds"], f"{location}.value_milliseconds"
    )
    speed = _finite_number(
        scenario["speed_world_units_per_second"],
        f"{location}.speed_world_units_per_second",
        positive=True,
    )
    if path_distance is None:
        _fail(f"{location}: estimated scenario requires a path distance")
    expected_milliseconds = path_distance / speed * 1000
    if not math.isclose(
        milliseconds, expected_milliseconds, rel_tol=1e-9, abs_tol=1e-6
    ):
        _fail(
            f"{location}: value does not equal path_distance / speed under its explicit assumption"
        )
    return True, speed


def _validate_motion(value: Any, t0_tick: int, location: str) -> None:
    motion = _object(value, location)
    _exact_keys(
        motion,
        {
            "availability_tick",
            "horizontal_speed_status",
            "horizontal_speed_world_units_per_second",
            "horizontal_speed_source",
            "active_weapon_status",
            "active_weapon",
            "is_walking",
            "is_ducking",
            "is_scoped",
        },
        location,
    )
    tick = motion["availability_tick"]
    if tick is not None and (
        not isinstance(tick, int)
        or isinstance(tick, bool)
        or tick < 0
        or tick > t0_tick
    ):
        _fail(f"{location}.availability_tick: must be null or <= t0_tick")
    if motion["horizontal_speed_status"] == "unavailable":
        if (
            motion["horizontal_speed_world_units_per_second"] is not None
            or motion["horizontal_speed_source"] is not None
        ):
            _fail(f"{location}: unavailable speed must be null")
    elif motion["horizontal_speed_status"] == "derived":
        _finite_number(
            motion["horizontal_speed_world_units_per_second"],
            f"{location}.horizontal_speed_world_units_per_second",
        )
        _non_empty_string(
            motion["horizontal_speed_source"], f"{location}.horizontal_speed_source"
        )
        if tick is None:
            _fail(f"{location}: derived speed requires availability_tick")
    else:
        _fail(f"{location}.horizontal_speed_status: invalid value")
    if motion["active_weapon_status"] == "unavailable":
        if motion["active_weapon"] is not None:
            _fail(f"{location}: unavailable weapon must be null")
    elif motion["active_weapon_status"] == "observed":
        _non_empty_string(motion["active_weapon"], f"{location}.active_weapon")
        if tick is None:
            _fail(f"{location}: observed weapon requires availability_tick")
    else:
        _fail(f"{location}.active_weapon_status: invalid value")
    for key in ("is_walking", "is_ducking", "is_scoped"):
        if motion[key] is not None and not isinstance(motion[key], bool):
            _fail(f"{location}.{key}: expected boolean or null")


def _validate_navigation_row(
    value: Any,
    index: int,
    *,
    source_alias: str,
    graph_asset_reference: str,
    graph_asset_sha256: str,
) -> dict[str, Any]:
    location = f"rows[{index}]"
    row = _object(value, location)
    _exact_keys(
        row,
        {
            "schema_id",
            "contract_version",
            "identity_semantics",
            "causal_role",
            "source",
            "join_keys",
            "round_number",
            "t0_tick",
            "availability",
            "origin_motion",
            "direct_distance",
            "reachability",
            "path_distance",
            "internal_timing",
            "abstention_reasons",
            "provenance",
        },
        location,
    )
    if (
        row["schema_id"] != NAVIGATION_SCHEMA_ID
        or row["contract_version"] != NAVIGATION_CONTRACT_VERSION
    ):
        _fail(f"{location}: candidate schema/version mismatch")
    if row["source"] != NAVIGATION_SOURCE:
        _fail(f"{location}.source: producer mismatch")
    if (
        row["identity_semantics"] != "join_only"
        or row["causal_role"] != "model_input_observation"
    ):
        _fail(f"{location}: identity or causal role mismatch")
    if "travel_time_estimate" in row:
        _fail(f"{location}: legacy single ETA field is forbidden")

    joins = _object(row["join_keys"], f"{location}.join_keys")
    _exact_keys(
        joins,
        {"match_id", "round_id", "query_id", "origin_player_id", "target_player_id"},
        f"{location}.join_keys",
    )
    for key, item in joins.items():
        _non_empty_string(item, f"{location}.join_keys.{key}")
    t0_tick = row["t0_tick"]
    if not isinstance(t0_tick, int) or isinstance(t0_tick, bool) or t0_tick < 0:
        _fail(f"{location}.t0_tick: expected non-negative integer")
    if (
        not isinstance(row["round_number"], int)
        or isinstance(row["round_number"], bool)
        or row["round_number"] < 1
    ):
        _fail(f"{location}.round_number: expected positive integer")
    expected_round_id = f"{source_alias}:round:{row['round_number']:03d}"
    if joins["match_id"] != source_alias or joins["round_id"] != expected_round_id:
        _fail(f"{location}.join_keys: match or round differs from report source")
    origin_alias = str(joins["origin_player_id"])
    target_alias = str(joins["target_player_id"])
    if (
        not PARTICIPANT_ALIAS_PATTERN.fullmatch(origin_alias)
        or not PARTICIPANT_ALIAS_PATTERN.fullmatch(target_alias)
        or origin_alias == target_alias
    ):
        _fail(f"{location}.join_keys: player identifiers must be distinct local aliases")
    expected_query_id = (
        f"{expected_round_id}:path:{origin_alias}:{target_alias}:{t0_tick}"
    )
    if joins["query_id"] != expected_query_id:
        _fail(f"{location}.join_keys.query_id: inconsistent local join")

    availability = _object(row["availability"], f"{location}.availability")
    _exact_keys(
        availability,
        {"origin_tick", "target_tick", "graph_tick"},
        f"{location}.availability",
    )
    for key in ("origin_tick", "target_tick"):
        tick = availability[key]
        if (
            not isinstance(tick, int)
            or isinstance(tick, bool)
            or tick < 0
            or tick > t0_tick
        ):
            _fail(f"{location}.availability.{key}: must be <= t0_tick")
    graph_tick = availability["graph_tick"]
    if graph_tick is not None and (
        not isinstance(graph_tick, int)
        or isinstance(graph_tick, bool)
        or graph_tick < 0
        or graph_tick > t0_tick
    ):
        _fail(f"{location}.availability.graph_tick: must be null or <= t0_tick")
    _validate_motion(row["origin_motion"], t0_tick, f"{location}.origin_motion")
    _validate_distance_fact(row["direct_distance"], f"{location}.direct_distance")
    path_distance = _validate_distance_fact(
        row["path_distance"], f"{location}.path_distance"
    )

    reachability = _object(row["reachability"], f"{location}.reachability")
    _exact_keys(reachability, {"status", "value"}, f"{location}.reachability")
    if reachability["status"] == "unavailable":
        if reachability["value"] is not None:
            _fail(f"{location}.reachability: unavailable value must be null")
        reachable = False
    elif reachability["status"] == "derived" and isinstance(
        reachability["value"], bool
    ):
        reachable = reachability["value"]
    else:
        _fail(f"{location}.reachability: invalid fact")

    timing = _object(row["internal_timing"], f"{location}.internal_timing")
    _exact_keys(
        timing,
        {
            "intended_use",
            "user_facing_mode",
            "decision_semantics",
            "estimate_semantics",
            "exact_user_facing_time_allowed",
            "arrival_prediction",
            "scenarios",
        },
        f"{location}.internal_timing",
    )
    expected_policy = {
        "intended_use": "internal_decision_support_only",
        "user_facing_mode": "qualitative_conditional_with_uncertainty_only",
        "decision_semantics": "constraint_signal_not_recommendation",
        "estimate_semantics": "scenario_reference_not_arrival_prediction",
        "exact_user_facing_time_allowed": False,
        "arrival_prediction": False,
    }
    for key, expected in expected_policy.items():
        if timing[key] != expected:
            _fail(f"{location}.internal_timing.{key}: unsafe value")
    scenarios = _object(timing["scenarios"], f"{location}.internal_timing.scenarios")
    scenario_keys = {
        "optimistic_uninterrupted_250ups_baseline",
        "active_weapon_uninterrupted_baseline",
        "observed_pace_projection",
    }
    _exact_keys(scenarios, scenario_keys, f"{location}.internal_timing.scenarios")
    scenario_results = {
        key: _validate_timing_scenario(
            scenarios[key],
            f"{location}.internal_timing.scenarios.{key}",
            path_distance=path_distance,
        )
        for key in scenario_keys
    }
    estimated = {key: result[0] for key, result in scenario_results.items()}
    scenario_speeds = {key: result[1] for key, result in scenario_results.items()}
    expected_scenario_provenance = {
        "optimistic_uninterrupted_250ups_baseline": (
            "optimistic_uninterrupted_250ups_assumption",
            "constant_250ups_without_stops_combat_angle_clearing_acceleration_or_equipment_changes",
        ),
        "active_weapon_uninterrupted_baseline": (
            "active_weapon_unscoped_max_speed_table",
            "constant_active_weapon_unscoped_max_speed_without_stops_combat_angle_clearing_acceleration_or_equipment_changes",
        ),
    }
    for key, (speed_source, assumption) in expected_scenario_provenance.items():
        if (
            scenarios[key]["speed_source"] != speed_source
            or scenarios[key]["assumption"] != assumption
        ):
            _fail(f"{location}.internal_timing.scenarios.{key}: provenance drift")
    pace_scenario = scenarios["observed_pace_projection"]
    motion = row["origin_motion"]
    expected_pace_source = "causal_horizontal_speed_at_t0"
    if motion["horizontal_speed_status"] == "derived":
        expected_pace_source += f":{motion['horizontal_speed_source']}"
    if (
        pace_scenario["speed_source"] != expected_pace_source
        or pace_scenario["assumption"]
        != "constant_t0_horizontal_pace_without_predicting_future_movement_or_interruptions"
    ):
        _fail(f"{location}.internal_timing.scenarios.observed_pace_projection: provenance drift")
    if reachable:
        if (
            path_distance is None
            or not estimated["optimistic_uninterrupted_250ups_baseline"]
        ):
            _fail(f"{location}: reachable row lacks path or optimistic baseline")
        ideal_speed = scenarios["optimistic_uninterrupted_250ups_baseline"][
            "speed_world_units_per_second"
        ]
        if ideal_speed != 250:
            _fail(f"{location}: optimistic baseline must use explicit 250 UPS")
    elif path_distance is not None or any(estimated.values()):
        _fail(f"{location}: unavailable route published path/timing")

    weapon_key = "active_weapon_uninterrupted_baseline"
    if estimated[weapon_key] and motion["active_weapon_status"] != "observed":
        _fail(f"{location}: weapon baseline requires an observed active weapon")
    weapon_name = motion["active_weapon"]
    expected_weapon_speed = (
        WEAPON_MAX_SPEED.get(str(weapon_name).strip().casefold())
        if weapon_name is not None
        else None
    )
    if estimated[weapon_key]:
        if expected_weapon_speed is None or not math.isclose(
            float(scenario_speeds[weapon_key]), expected_weapon_speed, rel_tol=1e-12
        ):
            _fail(f"{location}: weapon baseline differs from the versioned weapon table")
    elif reachable:
        weapon_reason = scenarios[weapon_key]["unavailability_reason"]
        expected_weapon_reason = (
            "active_weapon_unavailable"
            if motion["active_weapon_status"] == "unavailable"
            else "active_weapon_speed_unknown"
        )
        if expected_weapon_speed is not None or weapon_reason != expected_weapon_reason:
            _fail(f"{location}: weapon baseline availability is inconsistent")
    pace_key = "observed_pace_projection"
    if estimated[pace_key]:
        observed_speed = motion["horizontal_speed_world_units_per_second"]
        if motion["horizontal_speed_status"] != "derived" or observed_speed is None:
            _fail(f"{location}: pace projection requires a derived causal speed")
        if not math.isclose(
            float(observed_speed), float(scenario_speeds[pace_key]), rel_tol=1e-12
        ):
            _fail(f"{location}: pace projection speed differs from the causal observation")
    elif reachable:
        expected_pace_reason = (
            "observed_stationary_not_projectable"
            if motion["horizontal_speed_status"] == "derived"
            and motion["horizontal_speed_world_units_per_second"] == 0
            else "observed_horizontal_speed_unavailable"
        )
        if pace_scenario["unavailability_reason"] != expected_pace_reason:
            _fail(f"{location}: pace projection availability is inconsistent")

    reasons = row["abstention_reasons"]
    if not isinstance(reasons, list) or any(
        not isinstance(item, str) or not item for item in reasons
    ):
        _fail(f"{location}.abstention_reasons: invalid list")
    if reasons != sorted(set(reasons)):
        _fail(f"{location}.abstention_reasons: must be sorted and unique")

    provenance = _object(row["provenance"], f"{location}.provenance")
    required_provenance = {
        "source_artifact",
        "graph_contract_version",
        "shortest_path_algorithm",
        "endpoint_association_method",
        "endpoint_tolerance_world_units",
        "endpoint_vertical_tolerance_world_units",
        "edge_weight_method",
        "timing_scenario_contract",
        "weapon_speed_table_contract",
        "origin_association_world_units",
        "target_association_world_units",
        "graph_asset",
        "input_fields",
    }
    _exact_keys(provenance, required_provenance, f"{location}.provenance")
    expected_provenance = {
        "source_artifact": "replay_positions_motion_context_and_static_nav_in_memory",
        "graph_contract_version": GRAPH_CONTRACT_VERSION,
        "shortest_path_algorithm": "directed_dijkstra_stable_ties@1",
        "endpoint_association_method": "containing_nav_polygon_xy_then_nearest_center_3d@2",
        "endpoint_tolerance_world_units": 96,
        "endpoint_vertical_tolerance_world_units": 128,
        "edge_weight_method": "nav_area_center_euclidean_world_units@1",
        "timing_scenario_contract": "navigation_internal_timing_scenarios@1",
        "weapon_speed_table_contract": "cs2_weapon_max_speed_table@1",
    }
    for key, expected in expected_provenance.items():
        if provenance[key] != expected:
            _fail(f"{location}.provenance.{key}: unexpected value")
    for key in ("origin_association_world_units", "target_association_world_units"):
        if provenance[key] is not None:
            _finite_number(provenance[key], f"{location}.provenance.{key}")
    if reachable and (
        provenance["origin_association_world_units"] is None
        or provenance["target_association_world_units"] is None
    ):
        _fail(f"{location}.provenance: reachable route requires endpoint associations")
    if provenance["input_fields"] != NAVIGATION_INPUT_FIELDS:
        _fail(f"{location}.provenance.input_fields: contract drift")
    asset = _object(provenance["graph_asset"], f"{location}.provenance.graph_asset")
    _exact_keys(
        asset, {"status", "reference", "sha256"}, f"{location}.provenance.graph_asset"
    )
    if (
        asset["status"] != "loaded"
        or asset["reference"] != graph_asset_reference
        or asset["sha256"] != graph_asset_sha256
        or not SHA256_PATTERN.fullmatch(str(asset["sha256"]))
    ):
        _fail(f"{location}.provenance.graph_asset: loaded NAV hash required")

    return {
        "reachable": reachable,
        "ideal": estimated["optimistic_uninterrupted_250ups_baseline"],
        "weapon": estimated["active_weapon_uninterrupted_baseline"],
        "pace": estimated["observed_pace_projection"],
        "walking": row["origin_motion"]["is_walking"] is True,
        "non_walking_moving": (
            row["origin_motion"]["is_walking"] is False
            and row["origin_motion"]["horizontal_speed_status"] == "derived"
            and row["origin_motion"]["horizontal_speed_world_units_per_second"] > 0
        ),
        "ducking": row["origin_motion"]["is_ducking"] is True,
        "scoped": row["origin_motion"]["is_scoped"] is True,
        "knife": (
            estimated[weapon_key]
            and str(row["origin_motion"]["active_weapon"]).casefold() == "knife"
        ),
        "active_weapon": (
            row["origin_motion"]["active_weapon"] if estimated[weapon_key] else None
        ),
        "weapon_speed": scenario_speeds[weapon_key],
        "reasons": reasons,
        "timing_reasons": [
            scenario["unavailability_reason"]
            for scenario in scenarios.values()
            if scenario["status"] == "unavailable"
        ],
        "asset_sha256": asset["sha256"],
    }


def validate_navigation_smoke_report(value: Mapping[str, Any]) -> dict[str, int]:
    report = _object(value, "report")
    _exact_keys(
        report,
        {
            "schema_id",
            "status",
            "source_alias",
            "source_sha256",
            "map_name",
            "candidate_schema_id",
            "candidate_contract_version",
            "graph_contract_version",
            "graph_asset_reference",
            "graph_asset_sha256",
            "graph_node_count",
            "graph_edge_count",
            "query_limit",
            "build_identifier",
            "processing_duration_ms",
            "summary",
            "rows",
        },
        "report",
    )
    if report["schema_id"] != REPORT_SCHEMA_ID or report["status"] != "passed":
        _fail("report: schema or status mismatch")
    checksum = str(report["source_sha256"])
    alias = str(report["source_alias"])
    if (
        not SHA256_PATTERN.fullmatch(checksum)
        or not SOURCE_ALIAS_PATTERN.fullmatch(alias)
        or alias != f"demo-{checksum[:20]}"
    ):
        _fail("report: source alias/checksum mismatch")
    if (
        report["candidate_schema_id"] != NAVIGATION_SCHEMA_ID
        or report["candidate_contract_version"] != NAVIGATION_CONTRACT_VERSION
    ):
        _fail("report: candidate contract mismatch")
    if report["graph_contract_version"] != GRAPH_CONTRACT_VERSION:
        _fail("report: graph contract mismatch")
    _non_empty_string(report["map_name"], "report.map_name")
    graph_asset_reference = _non_empty_string(
        report["graph_asset_reference"], "report.graph_asset_reference"
    )
    graph_asset_sha256 = str(report["graph_asset_sha256"])
    if not SHA256_PATTERN.fullmatch(graph_asset_sha256):
        _fail("report.graph_asset_sha256: invalid hash")
    for key in ("graph_node_count", "graph_edge_count", "query_limit"):
        if (
            not isinstance(report[key], int)
            or isinstance(report[key], bool)
            or report[key] < 1
        ):
            _fail(f"report.{key}: expected positive integer")
    if report["query_limit"] > 256:
        _fail("report.query_limit: must be at most 256")
    _non_empty_string(report["build_identifier"], "report.build_identifier")
    if (
        not isinstance(report["processing_duration_ms"], int)
        or isinstance(report["processing_duration_ms"], bool)
        or report["processing_duration_ms"] < 0
    ):
        _fail("report.processing_duration_ms: expected non-negative integer")
    rows = report["rows"]
    minimum_rows = min(report["query_limit"], 8)
    if (
        not isinstance(rows, list)
        or len(rows) < minimum_rows
        or len(rows) > report["query_limit"]
    ):
        _fail("report.rows: expected a bounded non-empty list")

    facts = [
        _validate_navigation_row(
            row,
            index,
            source_alias=alias,
            graph_asset_reference=graph_asset_reference,
            graph_asset_sha256=graph_asset_sha256,
        )
        for index, row in enumerate(rows)
    ]
    if any(fact["asset_sha256"] != report["graph_asset_sha256"] for fact in facts):
        _fail("report: row NAV hash differs from report NAV hash")
    recomputed = {
        "rows": len(facts),
        "reachable_rows": sum(fact["reachable"] for fact in facts),
        "unavailable_rows": sum(not fact["reachable"] for fact in facts),
        "ideal_baseline_rows": sum(fact["ideal"] for fact in facts),
        "active_weapon_baseline_rows": sum(fact["weapon"] for fact in facts),
        "observed_pace_projection_rows": sum(fact["pace"] for fact in facts),
        "walking_contexts": sum(fact["walking"] for fact in facts),
        "non_walking_moving_contexts": sum(
            fact["non_walking_moving"] for fact in facts
        ),
        "ducking_contexts": sum(fact["ducking"] for fact in facts),
        "scoped_contexts": sum(fact["scoped"] for fact in facts),
        "knife_contexts": sum(fact["knife"] for fact in facts),
        "active_weapons": sorted(
            {fact["active_weapon"] for fact in facts if fact["active_weapon"]}
        ),
        "weapon_baseline_speeds_world_units_per_second": sorted(
            {fact["weapon_speed"] for fact in facts if fact["weapon_speed"] is not None}
        ),
        "exact_user_facing_time_violations": 0,
        "abstention_reasons": dict(
            sorted(
                Counter(reason for fact in facts for reason in fact["reasons"]).items()
            )
        ),
        "timing_unavailability_reasons": dict(
            sorted(
                Counter(
                    reason for fact in facts for reason in fact["timing_reasons"]
                ).items()
            )
        ),
    }
    summary = _object(report["summary"], "report.summary")
    _exact_keys(summary, set(recomputed), "report.summary")
    if dict(summary) != recomputed:
        _fail(f"report.summary: expected recomputed counts {recomputed}")
    if (
        recomputed["reachable_rows"] == 0
        or recomputed["active_weapon_baseline_rows"] == 0
        or recomputed["observed_pace_projection_rows"] == 0
        or recomputed["walking_contexts"] == 0
        or recomputed["non_walking_moving_contexts"] == 0
        or recomputed["knife_contexts"] == 0
        or len(recomputed["active_weapons"]) < 2
        or len(recomputed["weapon_baseline_speeds_world_units_per_second"]) < 2
    ):
        _fail("report: required real-demo timing coverage is zero")
    return {key: value for key, value in recomputed.items() if isinstance(value, int)}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path)
    args = parser.parse_args()
    try:
        payload = json.loads(args.report.read_text(encoding="utf-8"))
        counts = validate_navigation_smoke_report(payload)
    except (OSError, json.JSONDecodeError, NavigationSmokeValidationError) as exc:
        print(
            json.dumps(
                {"ok": False, "error": str(exc)}, ensure_ascii=False, sort_keys=True
            )
        )
        return 1
    print(
        json.dumps({"ok": True, "counts": counts}, ensure_ascii=False, sort_keys=True)
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
