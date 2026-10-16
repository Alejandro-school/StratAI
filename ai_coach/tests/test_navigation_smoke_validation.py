from __future__ import annotations

import copy

import pytest

from ai_coach.contracts.validate_navigation_smoke import (
    NavigationSmokeValidationError,
    validate_navigation_smoke_report,
)


def _estimated(
    speed: float, milliseconds: float, speed_source: str, assumption: str
) -> dict:
    return {
        "status": "estimated",
        "value_milliseconds": milliseconds,
        "speed_world_units_per_second": speed,
        "speed_source": speed_source,
        "assumption": assumption,
        "unavailability_reason": None,
    }


def _valid_report() -> dict:
    nav_hash = "b" * 64
    alias = "demo-" + "a" * 20
    ideal_assumption = (
        "constant_250ups_without_stops_combat_angle_clearing_acceleration_or_"
        "equipment_changes"
    )
    weapon_assumption = (
        "constant_active_weapon_unscoped_max_speed_without_stops_combat_angle_"
        "clearing_acceleration_or_equipment_changes"
    )
    pace_assumption = (
        "constant_t0_horizontal_pace_without_predicting_future_movement_or_"
        "interruptions"
    )
    row = {
        "schema_id": "stratai.observed_navigation_path@2",
        "contract_version": "2.0.0",
        "identity_semantics": "join_only",
        "causal_role": "model_input_observation",
        "source": "replay_position_pair_navigation_derivation",
        "join_keys": {
            "match_id": alias,
            "round_id": f"{alias}:round:001",
            "query_id": (
                f"{alias}:round:001:path:participant-001:participant-002:100"
            ),
            "origin_player_id": "participant-001",
            "target_player_id": "participant-002",
        },
        "round_number": 1,
        "t0_tick": 100,
        "availability": {"origin_tick": 100, "target_tick": 100, "graph_tick": 0},
        "origin_motion": {
            "availability_tick": 100,
            "horizontal_speed_status": "derived",
            "horizontal_speed_world_units_per_second": 100,
            "horizontal_speed_source": "position_delta",
            "active_weapon_status": "observed",
            "active_weapon": "AK-47",
            "is_walking": True,
            "is_ducking": False,
            "is_scoped": False,
        },
        "direct_distance": {"status": "derived", "value_world_units": 80},
        "reachability": {"status": "derived", "value": True},
        "path_distance": {"status": "derived", "value_world_units": 100},
        "internal_timing": {
            "intended_use": "internal_decision_support_only",
            "user_facing_mode": "qualitative_conditional_with_uncertainty_only",
            "decision_semantics": "constraint_signal_not_recommendation",
            "estimate_semantics": "scenario_reference_not_arrival_prediction",
            "exact_user_facing_time_allowed": False,
            "arrival_prediction": False,
            "scenarios": {
                "optimistic_uninterrupted_250ups_baseline": _estimated(
                    250,
                    400,
                    "optimistic_uninterrupted_250ups_assumption",
                    ideal_assumption,
                ),
                "active_weapon_uninterrupted_baseline": _estimated(
                    215,
                    100 / 215 * 1000,
                    "active_weapon_unscoped_max_speed_table",
                    weapon_assumption,
                ),
                "observed_pace_projection": _estimated(
                    100,
                    1000,
                    "causal_horizontal_speed_at_t0:position_delta",
                    pace_assumption,
                ),
            },
        },
        "abstention_reasons": [],
        "provenance": {
            "source_artifact": "replay_positions_motion_context_and_static_nav_in_memory",
            "graph_contract_version": "nav_area_connections_with_polygons@2",
            "shortest_path_algorithm": "directed_dijkstra_stable_ties@1",
            "endpoint_association_method": "containing_nav_polygon_xy_then_nearest_center_3d@2",
            "endpoint_tolerance_world_units": 96,
            "endpoint_vertical_tolerance_world_units": 128,
            "edge_weight_method": "nav_area_center_euclidean_world_units@1",
            "timing_scenario_contract": "navigation_internal_timing_scenarios@1",
            "weapon_speed_table_contract": "cs2_weapon_max_speed_table@1",
            "origin_association_world_units": 20,
            "target_association_world_units": 20,
            "graph_asset": {
                "status": "loaded",
                "reference": "de_mirage/de_mirage.nav",
                "sha256": nav_hash,
            },
            "input_fields": [
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
            ],
        },
    }
    knife_row = copy.deepcopy(row)
    knife_row["t0_tick"] = 104
    knife_row["availability"] = {
        "origin_tick": 104,
        "target_tick": 104,
        "graph_tick": 0,
    }
    knife_row["origin_motion"]["availability_tick"] = 104
    knife_row["origin_motion"]["active_weapon"] = "Knife"
    knife_row["origin_motion"]["is_walking"] = False
    knife_row["join_keys"]["query_id"] = (
        f"{alias}:round:001:path:participant-001:participant-002:104"
    )
    knife_row["internal_timing"]["scenarios"][
        "active_weapon_uninterrupted_baseline"
    ] = _estimated(
        250,
        400,
        "active_weapon_unscoped_max_speed_table",
        weapon_assumption,
    )
    return {
        "schema_id": "stratai.observed_navigation_single_demo_smoke@1",
        "status": "passed",
        "source_alias": alias,
        "source_sha256": "a" * 64,
        "map_name": "de_mirage",
        "candidate_schema_id": "stratai.observed_navigation_path@2",
        "candidate_contract_version": "2.0.0",
        "graph_contract_version": "nav_area_connections_with_polygons@2",
        "graph_asset_reference": "de_mirage/de_mirage.nav",
        "graph_asset_sha256": nav_hash,
        "graph_node_count": 10,
        "graph_edge_count": 20,
        "query_limit": 2,
        "build_identifier": "test-build",
        "processing_duration_ms": 10,
        "summary": {
            "rows": 2,
            "reachable_rows": 2,
            "unavailable_rows": 0,
            "ideal_baseline_rows": 2,
            "active_weapon_baseline_rows": 2,
            "observed_pace_projection_rows": 2,
            "walking_contexts": 1,
            "non_walking_moving_contexts": 1,
            "ducking_contexts": 0,
            "scoped_contexts": 0,
            "knife_contexts": 1,
            "active_weapons": ["AK-47", "Knife"],
            "weapon_baseline_speeds_world_units_per_second": [215, 250],
            "exact_user_facing_time_violations": 0,
            "abstention_reasons": {},
            "timing_unavailability_reasons": {},
        },
        "rows": [row, knife_row],
    }


def test_valid_navigation_smoke_is_recomputed() -> None:
    assert validate_navigation_smoke_report(_valid_report()) == {
        "rows": 2,
        "reachable_rows": 2,
        "unavailable_rows": 0,
        "ideal_baseline_rows": 2,
        "active_weapon_baseline_rows": 2,
        "observed_pace_projection_rows": 2,
        "walking_contexts": 1,
        "non_walking_moving_contexts": 1,
        "ducking_contexts": 0,
        "scoped_contexts": 0,
        "knife_contexts": 1,
        "exact_user_facing_time_violations": 0,
    }


def test_exact_user_facing_time_policy_violation_fails() -> None:
    report = _valid_report()
    report["rows"][0]["internal_timing"]["exact_user_facing_time_allowed"] = True
    with pytest.raises(NavigationSmokeValidationError, match="unsafe value"):
        validate_navigation_smoke_report(report)


def test_future_motion_or_tampered_summary_fails() -> None:
    future = _valid_report()
    future["rows"][0]["origin_motion"]["availability_tick"] = 101
    with pytest.raises(NavigationSmokeValidationError, match="availability_tick"):
        validate_navigation_smoke_report(future)

    tampered = copy.deepcopy(_valid_report())
    tampered["summary"]["reachable_rows"] = 0
    with pytest.raises(NavigationSmokeValidationError, match="recomputed counts"):
        validate_navigation_smoke_report(tampered)


def test_timing_formula_and_weapon_table_tampering_fail() -> None:
    bad_formula = _valid_report()
    bad_formula["rows"][0]["internal_timing"]["scenarios"][
        "observed_pace_projection"
    ]["value_milliseconds"] = 999
    with pytest.raises(NavigationSmokeValidationError, match="path_distance / speed"):
        validate_navigation_smoke_report(bad_formula)

    bad_weapon = _valid_report()
    scenario = bad_weapon["rows"][0]["internal_timing"]["scenarios"][
        "active_weapon_uninterrupted_baseline"
    ]
    scenario["speed_world_units_per_second"] = 1
    scenario["value_milliseconds"] = 100_000
    with pytest.raises(NavigationSmokeValidationError, match="weapon table"):
        validate_navigation_smoke_report(bad_weapon)


@pytest.mark.parametrize(
    ("mutate", "message"),
    [
        (lambda report: report["rows"][0].__setitem__("source", "future_oracle"), "producer"),
        (
            lambda report: report["rows"][0]["provenance"].__setitem__(
                "input_fields", ["future_outcome"]
            ),
            "contract drift",
        ),
        (
            lambda report: report["rows"][0]["provenance"]["graph_asset"].__setitem__(
                "reference", "another.nav"
            ),
            "loaded NAV hash",
        ),
        (
            lambda report: report["rows"][0]["join_keys"].__setitem__(
                "match_id", "demo-" + "c" * 20
            ),
            "differs from report",
        ),
    ],
)
def test_schema_provenance_and_single_demo_join_tampering_fail(
    mutate, message: str
) -> None:
    report = _valid_report()
    mutate(report)
    with pytest.raises(NavigationSmokeValidationError, match=message):
        validate_navigation_smoke_report(report)
