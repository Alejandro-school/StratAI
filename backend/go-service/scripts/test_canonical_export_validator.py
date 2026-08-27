import gc
import gzip
import hashlib
import importlib.util
import json
import sys
import weakref
from collections.abc import Iterator
from copy import deepcopy
from pathlib import Path

import pytest

SCRIPT_PATH = Path(__file__).with_name("canonical_export_validator.py")
SPEC = importlib.util.spec_from_file_location("canonical_export_validator", SCRIPT_PATH)
validator = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = validator
SPEC.loader.exec_module(validator)

MATCH_ID = "123"
PLAYER_ONE = "steam:1"
PLAYER_TWO = "steam:2"
ROUND_ID = f"{MATCH_ID}:round:001"
DEMO_CHECKSUM = "a" * 64


def valid_block7_domains() -> list[dict]:
    return [
        {
            "name": name,
            "status": "pass",
            "severity": "hard",
            "expected": "0 hard violations",
            "actual": "0 hard violations",
            "coverage": 1.0,
            "unavailable_count": 0,
            "inferred_count": 0,
            "warning_details": [],
            "hard_failure_details": [],
            "source_artifacts": ["canonical/manifest.json"],
            "schema_versions": [validator.MANIFEST_SCHEMA_ID],
        }
        for name in sorted(validator.REQUIRED_BLOCK7_DOMAINS)
    ]


def valid_combat_event(event_id: str) -> dict:
    return {
        "schema_id": "stratai.combat_event@2",
        "match_id": MATCH_ID,
        "event_id": event_id,
        "round_id": ROUND_ID,
        "round_number": 1,
        "tick": 100,
        "sequence_in_tick": 1,
        "sequence_in_round": 1,
        "event_type": "player_hurt",
        "source": "demoinfocs.player_hurt",
        "source_event_ids": [],
        "tick_status": "observed",
        "subtick": None,
        "subtick_status": "unavailable",
        "time_seconds": 1.5625,
        "time_seconds_status": "derived",
        "time_seconds_source": "tick_divided_by_tick_rate",
        "actor_player_id": PLAYER_ONE,
        "actor_side": "t",
        "actor_status": "observed",
        "actor_source": "callback_player",
        "target_player_id": PLAYER_TWO,
        "target_side": "ct",
        "target_status": "observed",
        "target_source": "callback_player",
        "assister_player_id": None,
        "assister_side": None,
        "assister_status": "unavailable",
        "assister_source": "unavailable",
        "relation": "enemy",
        "weapon": "ak47",
        "weapon_status": "observed",
        "weapon_source": "callback_weapon",
        "weapon_is_utility": False,
        "actor_position": {"x": 1.0, "y": 2.0, "z": 3.0},
        "actor_position_status": "observed",
        "actor_position_source": "callback_player_position",
        "target_position": {"x": 4.0, "y": 5.0, "z": 6.0},
        "target_position_status": "observed",
        "target_position_source": "callback_player_position",
        "shot_id": None,
        "correlation_status": "unavailable",
        "correlation_source": "unavailable",
        "shot_result": None,
        "shot_result_status": "unavailable",
        "shot_result_source": "unavailable",
        "shot_result_availability_tick": None,
        "view_yaw": None,
        "view_pitch": None,
        "impact_position": None,
        "impact_position_status": "unavailable",
        "impact_position_source": "unavailable",
        "bullet_distance_world_units": None,
        "damage_direction": None,
        "penetrated_objects": None,
        "no_scope": None,
        "attacker_in_air": None,
        "through_smoke": None,
        "attacker_blind": None,
        "kill_distance_world_units": None,
        "health_damage": 40,
        "health_damage_taken": 40,
        "armor_damage": 0,
        "armor_damage_taken": 0,
        "health_before": 100,
        "health_after": 60,
        "armor_before": 50,
        "armor_after": 50,
        "damage_status": "observed",
        "damage_source": "demoinfocs.player_hurt",
        "hitgroup": "chest",
        "hitgroup_status": "observed",
        "hitgroup_source": "demoinfocs.player_hurt",
        "is_headshot": False,
        "is_kill": False,
        "assisted_flash": None,
        "reload_phase": None,
        "reload_end_tick": None,
        "reload_end_status": "unavailable",
        "previous_weapon": None,
        "previous_weapon_status": "unavailable",
        "is_weapon_switch": None,
        "ammo_in_magazine": None,
        "ammo_reserve": None,
        "ammo_status": "unavailable",
        "ammo_source": "unavailable",
        "reaction_time_ms": None,
        "time_to_damage_ms": None,
    }


def valid_combat_metrics(player_id: str) -> dict:
    dealt_damage = player_id == PLAYER_ONE
    return {
        "steam_id": player_id.removeprefix("steam:"),
        "kills": 0,
        "deaths": 0,
        "assists": 0,
        "kills_observed": 0,
        "deaths_observed": 0,
        "assists_observed": 0,
        "kills_native_minus_observed": 0,
        "deaths_native_minus_observed": 0,
        "assists_native_minus_observed": 0,
        "flash_assists": 0,
        "headshots": 0,
        "total_damage": 40 if dealt_damage else 0,
        "combat_damage_observed": 40 if dealt_damage else 0,
        "combat_damage_unattributed_delta": 0,
        "friendly_damage": 0,
        "self_damage": 0,
		"utility_damage": 0,
		"utility_damage_observed": 0,
        "shots_fired": 0,
        "shots_hit": 0,
        "shots_missed": 0,
        "body_part_hits": {"chest": 1} if dealt_damage else {},
        "weapon_stats": {
            "ak47": {
                "kills": 0,
                "headshots": 0,
                "damage": 40,
                "shots_fired": 0,
                "shots_hit": 0,
                "shots_missed": 0,
                "accuracy": 0,
            }
        } if dealt_damage else {},
        "native_scoreboard": {
            "kills": 0,
            "deaths": 0,
            "assists": 0,
            "total_damage": 40 if dealt_damage else 0,
			"utility_damage": 0,
        },
    }


def valid_combat_quality() -> dict:
    diagnostics = {
        event_type: {
            "observed": 1 if event_type == "player_hurt" else 0,
            "recorded": 1 if event_type == "player_hurt" else 0,
            "discarded": 0,
        }
        for event_type in validator.COMBAT_CALLBACK_GROUPS
    }
    return {
        **{name: 0 for name in validator.REQUIRED_COMBAT_QUALITY_METRICS},
        "combat_ledger_events": 1,
        "combat_callback_diagnostics": diagnostics,
        "combat_discarded_callback_reasons": {},
    }


def valid_economy_player(player_id: str, team_id: str, side: str) -> dict:
    observed = {"amount": 800, "status": "observed", "source": "test"}
    calculated = {"amount": 800, "status": "calculated", "source": "test"}
    unavailable = {"amount": None, "status": "not_observed", "source": "test"}
    inventory = {
        "status": "observed_with_calculated_valuation",
        "native_value": 0,
        "calculated_value": 0,
        "items": [],
    }
    return {
        "round_id": ROUND_ID,
        "round_number": 1,
        "player_id": player_id,
        "team_id": team_id,
        "side": side,
        "outcome": "win" if team_id == "team_a" else "loss",
        "survived": True,
        "money": {
            "round_start_observed": observed,
            "freeze_end_observed": observed,
            "after_buy_observed": observed,
            "after_buy_calculated": calculated,
            "round_end_observed": observed,
            "next_round_observed": unavailable,
            "next_round_calculated": {"amount": None, "status": "not_evaluable", "source": "incomplete_rewards"},
            "native_calculated_delta": 0,
        },
        "inventory_start": inventory,
        "inventory_freeze_end": inventory,
        "inventory_round_end": inventory,
        "spent_in_buy": {"amount": 0, "status": "observed", "source": "test"},
        "transactions": [],
        "warnings": [],
    }


def valid_player_stats(player_id: str, team_id: str) -> dict:
    metrics = {
        **valid_combat_metrics(player_id),
        "trade_kills": 0,
        "traded_deaths": 0,
        "trade_attempts": 0,
        "failed_trade_attempts": 0,
        "untradeable_deaths": 0,
        "non_evaluable_trade_deaths": 0,
    }
    native = metrics["native_scoreboard"]
    return {
        "player_id": player_id,
        "team_id": team_id,
        "native_scoreboard_status": "observed",
        "native_scoreboard": native,
        "derived": {
            "kills_observed": 0,
            "deaths_observed": 0,
            "assists_observed": 0,
            "combat_damage_observed": metrics["combat_damage_observed"],
            "utility_damage_observed": 0,
            "grenade_damage_observed": {},
            "trade_kills": 0,
            "traded_deaths": 0,
            "opening_duels_attempted": 0,
            "opening_duels_won": 0,
            "opening_duels_lost": 0,
            "kd_ratio_observed": None,
        },
        "reconciliation": {
            "kills_native_minus_observed": 0,
            "deaths_native_minus_observed": 0,
            "assists_native_minus_observed": 0,
            "damage_native_minus_observed": 0,
            "utility_native_minus_observed": 0,
        },
        "clutch": {"attempts": 0, "wins": 0, "losses": 0, "not_evaluable": 0, "by_state": {"1v1": 0, "1v2": 0, "1v3": 0, "1v4": 0, "1v5": 0}},
        "rating": {"value": 1.0, "status": "calculated", "approximate": True, "algorithm_version": "stratai.rating_hltv2_approx@1", "formula": "test formula", "source": "test"},
        "metrics": metrics,
        "provenance": {"combat": "stratai.combat_event@2"},
    }


def valid_tactical_physical_state() -> dict:
    availability = {
        field: "unavailable"
        for field in (
            "team", "position", "velocity_world_units_per_second",
            "horizontal_velocity_world_units_per_second", "yaw", "pitch", "health",
            "armor", "alive", "active_weapon", "grenades", "has_c4",
            "has_defuse_kit", "ammo_in_magazine", "ammo_reserve", "is_ducking",
            "is_walking", "is_scoped", "is_reloading", "is_blind",
            "flash_duration_seconds", "money", "is_defusing",
        )
    }
    for field in (
        "team", "position", "yaw", "pitch", "health", "armor", "alive",
        "active_weapon", "grenades", "has_c4", "has_defuse_kit", "is_ducking",
        "is_walking", "is_scoped", "is_blind", "flash_duration_seconds", "money",
        "is_defusing",
    ):
        availability[field] = "observed"
    return {
        "team": "T",
        "position": {"x": 1.0, "y": 2.0, "z": 3.0},
        "velocity_world_units_per_second": None,
        "horizontal_velocity_world_units_per_second": None,
        "yaw": 0.0,
        "pitch": 0.0,
        "health": 100,
        "armor": 50,
        "alive": True,
        "active_weapon": "ak47",
        "grenades": [],
        "has_c4": True,
        "has_defuse_kit": False,
        "ammo_in_magazine": None,
        "ammo_reserve": None,
        "is_ducking": False,
        "is_walking": False,
        "is_scoped": False,
        "is_reloading": None,
        "is_blind": False,
        "flash_duration_seconds": 0.0,
        "money": 800,
        "is_defusing": False,
        "field_availability": availability,
    }


def write_json(path: Path, payload: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, separators=(",", ":")), encoding="utf-8")


def write_jsonl(path: Path, records: list[dict]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    content = "".join(
        f"{json.dumps(record, separators=(',', ':'))}\n" for record in records
    )
    if path.suffix == ".gz":
        with gzip.open(path, "wt", encoding="utf-8") as file:
            file.write(content)
    else:
        path.write_text(content, encoding="utf-8")


def read_jsonl(path: Path) -> list[dict]:
    if path.suffix == ".gz":
        with gzip.open(path, "rt", encoding="utf-8") as file:
            return [json.loads(line) for line in file]
    return [
        json.loads(line)
        for line in path.read_text(encoding="utf-8").splitlines()
    ]


def test_iter_json_lines_rejects_truncated_gzip(tmp_path: Path) -> None:
    path = tmp_path / "observed.jsonl.gz"
    write_jsonl(path, [{"schema_id": "test@1", "match_id": MATCH_ID}])
    path.write_bytes(path.read_bytes()[:-4])

    errors: list[str] = []
    list(validator.iter_json_lines(path, errors))

    assert any("no se pudo leer" in error for error in errors)


def write_gzip_json(path: Path, payload: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with gzip.open(path, "wt", encoding="utf-8") as file:
        json.dump(payload, file, separators=(",", ":"))


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def build_payloads() -> dict[str, tuple[str, str, object, list[str]]]:
    combat_id = f"{MATCH_ID}:combat:001:100:001"
    carrier_id = f"{MATCH_ID}:objective:001:090:001"
    plant_start_id = f"{MATCH_ID}:objective:001:100:002"
    plant_id = f"{MATCH_ID}:objective:001:104:003"
    defuse_start_id = f"{MATCH_ID}:objective:001:110:004"
    defuse_id = f"{MATCH_ID}:objective:001:120:005"
    plant_attempt_id = f"{MATCH_ID}:objective-attempt:001:plant:001"
    defuse_attempt_id = f"{MATCH_ID}:objective-attempt:001:defuse:001"
    replay_events = [
		{
			"id": "combat-r001-e000001",
			"tick": 100,
			"type": "player_hurt",
			"source_event_ids": [combat_id],
			"killer_id": "1",
			"victim_id": "2",
			"killer_team": "T",
			"victim_team": "CT",
			"weapon": "ak47",
			"damage": 40,
			"duration_ms": 320,
		},
        {
            "id": "objective:bomb_plant:104",
            "tick": 104,
            "type": "bomb_plant",
            "actor_id": "1",
            "player_id": "1",
            "site": "A",
            "x": 10,
            "y": 20,
        },
        {
            "id": "objective:bomb_defuse:120",
            "tick": 120,
            "type": "bomb_defuse",
            "actor_id": "2",
            "player_id": "2",
            "site": "A",
            "x": 11,
            "y": 21,
        },
        {
            "id": "utility:r1-u0001",
            "tick": 120,
            "type": "utility_detonate",
            "grenade_type": "flashbang",
            "utility_type": "flashbang",
            "x": 15.0,
            "y": 25.0,
            "z": 30.0,
            "actor_id": "1",
            "player_id": "1",
            "affected_player_ids": ["2"],
            "duration_ms": 1251,
            "duration_status": "observed",
            "duration_source": "player_flashed",
            "position_status": "observed",
            "position_source": "flash_explode",
            "correlation_status": "observed",
            "correlation_source": "projectile_entity",
            "source_throw_id": "r1-u0001",
        },
    ]
    return {
        "core/match.json": (
            "match",
            "stratai.match@1",
            {
                "schema_id": "stratai.match@1",
                "match_id": MATCH_ID,
                "tick_rate_hz": 64.0,
                "round_count": 1,
                "ct_score": 1,
                "t_score": 0,
                "teams": [
                    {"team_id": "team_a", "score": 1},
                    {"team_id": "team_b", "score": 0},
                ],
            },
            [],
        ),
		"core/match_metadata.json": (
			"match_metadata",
			"stratai.match_metadata@1",
			{
				"schema_id": "stratai.match_metadata@1",
				"match_id": MATCH_ID,
				"played_at": None,
				"played_at_status": "unavailable",
				"played_at_source": None,
				"origin_date": None,
				"origin_date_status": "unavailable",
				"source": {"source": "demo", "endpoint": None, "queried_at": None, "version": None, "checksum_sha256": DEMO_CHECKSUM},
				"parser_version": validator.PARSER_SCHEMA_VERSION,
				"export_format_version": validator.EXPORT_FORMAT_VERSION,
				"quality_schema_version": validator.QUALITY_SCHEMA_VERSION,
				"price_table": {"version": "stratai.cs2_prices@1", "checksum_sha256": "b" * 64, "effective_from": "2026-08-19", "applicability_status": "unverified_match_date", "source": "test"},
				"economy_rules": {"version": "stratai.cs2_economy_rules@1", "checksum_sha256": "c" * 64, "effective_from": "2026-08-19", "applicability_status": "unverified_match_date", "source": "test"},
				"algorithms": {
					"clutch": {"version": "stratai.clutch_ledger@1", "checksum_sha256": "d" * 64, "effective_from": "2026-08-19", "applicability_status": "applicable", "source": "test"},
					"rating": {"version": "stratai.rating_hltv2_approx@1", "checksum_sha256": "e" * 64, "effective_from": "2026-08-19", "applicability_status": "approximate", "source": "test"},
					"stats": {"version": "stratai.player_stats_ledger@1", "checksum_sha256": "f" * 64, "effective_from": "2026-08-19", "applicability_status": "applicable", "source": "test"},
				},
				"configuration_hashes": {
					"price_table": "b" * 64,
					"economy_rules": "c" * 64,
					"clutch_algorithm": "d" * 64,
					"rating_algorithm": "e" * 64,
					"stats_algorithm": "f" * 64,
				},
				"transformation_versions": {"economy": "@1", "stats": "@1", "clutch": "@1", "metadata": "@1"},
				"availability": {"played_at": "unavailable"},
				"warnings": ["played_at unavailable"],
			},
			[],
		),
        "core/participants.json": (
            "participants",
            "stratai.participants@1",
            {
                "schema_id": "stratai.participants@1",
                "match_id": MATCH_ID,
                "players": [
                    {
                        "player_id": PLAYER_ONE,
                        "steam_id": "1",
                        "display_name": "one",
                        "team_id": "team_b",
                    },
                    {
                        "player_id": PLAYER_TWO,
                        "steam_id": "2",
                        "display_name": "two",
                        "team_id": "team_a",
                    },
                ],
            },
            ["player_id"],
        ),
        "core/rounds.json": (
            "rounds",
            "stratai.rounds@2",
            {
                "schema_id": "stratai.rounds@2",
                "match_id": MATCH_ID,
                "rounds": [
                    {
                        "round_id": ROUND_ID,
                        "round_number": 1,
                        "start_tick": 80,
                        "end_tick": 140,
                        "winner_side": "ct",
                        "winner_team_id": "team_a",
                        "side_assignments": [
                            {"team_id": "team_a", "side": "ct"},
                            {"team_id": "team_b", "side": "t"},
                        ],
                        "team_scores_after": [
                            {"team_id": "team_a", "score": 1},
                            {"team_id": "team_b", "score": 0},
                        ],
                        "win_reason": "bomb_defused",
                        "raw_win_reason_code": 7,
                        "ct_score_after": 1,
                        "t_score_after": 0,
                        "bomb_planted": True,
                        "bomb_site": "A",
                        "bomb_tick": 104,
                        "objective": {
                            "was_bomb_planted": True,
                            "plant_event_id": plant_id,
                            "site": "A",
                            "plant_tick": 104,
                            "planter_player_id": PLAYER_ONE,
                            "outcome": "defused",
                            "resolution_event_id": defuse_id,
                            "resolution_tick": 120,
                            "resolver_player_id": PLAYER_TWO,
                            "plant_attempts": 1,
                            "plant_aborts": 0,
                            "defuse_attempts": 1,
                            "defuse_aborts": 0,
                            "bomb_drops": 0,
                            "bomb_pickups": 0,
                        },
                    }
                ],
            },
            ["round_number"],
        ),
        "events/combat_events.jsonl": (
            "combat_events",
			"stratai.combat_event@2",
			[valid_combat_event(combat_id)],
			["round_number", "tick", "sequence_in_tick", "event_id"],
        ),
        "events/utility_events.jsonl": (
            "utility_events",
            "stratai.utility_event@2",
            [
                {
                    "schema_id": "stratai.utility_event@2",
                    "match_id": MATCH_ID,
                    "event_id": f"{MATCH_ID}:utility:r1-u0001",
                    "source_throw_id": "r1-u0001",
                    "source_entity": {
                        "round_number": 1,
                        "entity_id": 77,
                        "generation": 1,
                    },
                    "source_entity_status": "observed",
                    "source_entity_source": "projectile_entity",
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "sequence_in_round": 1,
                    "event_type": "utility_throw",
                    "utility_type": "flashbang",
                    "utility_type_status": "observed",
                    "utility_type_source": "weapon_instance",
                    "thrower_player_id": PLAYER_ONE,
                    "thrower_side": "t",
                    "thrower_status": "observed",
                    "thrower_source": "projectile_thrower",
                    "correlation": {
                        "status": "observed",
                        "source": "projectile_entity",
                    },
                    "launch": {
                        "tick": 110,
                        "tick_status": "observed",
                        "tick_source": "projectile_throw",
                        "position": {
                            "value": {"x": 10.0, "y": 20.0, "z": 30.0},
                            "status": "observed",
                            "source": "projectile_position",
                        },
                        "view": {
                            "yaw_deg": 0.0,
                            "pitch_deg": 0.0,
                            "vector": {"x": 1.0, "y": 0.0, "z": 0.0},
                            "status": "observed",
                            "source": "player_view",
                        },
                        "thrower_velocity": {
                            "vector_world_units_per_second": {
                                "x": 0.0,
                                "y": 0.0,
                                "z": 0.0,
                            },
                            "horizontal_world_units_per_second": 0.0,
                            "observed_tick": 110,
                            "measurement_window_ticks": 0,
                            "status": "observed",
                            "source": "native",
                        },
                        "projectile_initial_velocity": {
                            "vector_world_units_per_second": {
                                "x": 100.0,
                                "y": 0.0,
                                "z": 50.0,
                            },
                            "horizontal_world_units_per_second": 100.0,
                            "observed_tick": 110,
                            "measurement_window_ticks": 0,
                            "status": "observed",
                            "source": "projectile_velocity",
                        },
                        "stance": {
                            "value": "standing",
                            "status": "observed",
                            "source": "player_state",
                        },
                        "area": {
                            "value": "Mid",
                            "status": "observed",
                            "source": "map_callout",
                        },
                    },
                    "trajectory": {
                        "bounce_count": 1,
                        "bounce_status": "observed",
                        "bounce_source": "projectile_bounce",
                        "samples": [
                            {
                                "tick": 110,
                                "position": {"x": 10.0, "y": 20.0, "z": 30.0},
                                "source": "projectile_frames",
                            },
                            {
                                "tick": 119,
                                "position": {"x": 15.0, "y": 25.0, "z": 30.0},
                                "source": "projectile_frames",
                            },
                        ],
                        "bounces": [
                            {
                                "tick": 115,
                                "position": {"x": 12.0, "y": 22.0, "z": 30.0},
                                "position_status": "observed",
                                "number": 1,
                                "source": "projectile_bounce",
                            }
                        ],
                        "status": "observed",
                        "source": "projectile_frames",
                    },
                    "lifecycle": {
                        "status": "detonated",
                        "detonation": {
                            "tick": 120,
                            "position": {"x": 15.0, "y": 25.0, "z": 30.0},
                            "status": "observed",
                            "position_status": "observed",
                            "source": "flash_explode",
                        },
                        "effect_start": {
                            "tick": None,
                            "position": None,
                            "status": "not_applicable",
                            "position_status": "not_applicable",
                            "source": "unavailable",
                        },
                        "expiration": {
                            "tick": None,
                            "position": None,
                            "status": "not_applicable",
                            "position_status": "not_applicable",
                            "source": "unavailable",
                        },
                        "destroy": {
                            "tick": 120,
                            "position": {"x": 15.0, "y": 25.0, "z": 30.0},
                            "status": "observed",
                            "position_status": "observed",
                            "source": "projectile_destroy",
                        },
                        "extinguish": {
                            "tick": None,
                            "position": None,
                            "status": "not_applicable",
                            "position_status": "not_applicable",
                            "source": "unavailable",
                        },
                        "duration": {
                            "milliseconds": None,
                            "status": "not_applicable",
                            "source": "unavailable",
                        },
                        "area": {
                            "value": "Mid",
                            "status": "observed",
                            "source": "map_callout",
                        },
                        "end_reason": {
                            "value": None,
                            "status": "not_applicable",
                            "source": "unavailable",
                        },
                        "extinguished_by_throw_id": {
                            "value": None,
                            "status": "not_applicable",
                            "source": "unavailable",
                        },
                        "extinguish_correlation": {
                            "status": "unavailable",
                            "source": "unavailable",
                        },
                    },
                    "affected_players": [
                        {
                            "player_id": PLAYER_TWO,
                            "side": "ct",
                            "player_status": "observed",
                            "player_source": "player_flashed",
                            "relation": "enemy",
                            "is_enemy": True,
                            "is_self": False,
                            "blind_duration": {
                                "milliseconds": 1250.5,
                                "status": "observed",
                                "source": "player_flashed",
                            },
                            "blind_correlation": {
                                "status": "observed",
                                "source": "projectile_entity",
                            },
                            "damage": None,
                            "armor_damage": None,
                            "is_kill": None,
                            "damage_events": [],
                        }
                    ],
                    "flash_summary": {
                        "players_total": 1,
                        "enemies_flashed": 1,
                        "teammates_flashed": 0,
                        "self_flashed": 0,
                        "unknown_flashed": 0,
                        "total_duration_ms": 1250.5,
                        "enemy_duration_ms": 1250.5,
                        "teammate_duration_ms": 0.0,
                        "self_duration_ms": 0.0,
                        "unknown_duration_ms": 0.0,
                    },
                    "damage_summary": {
                        "total_damage": 0,
                        "enemy_damage": 0,
                        "teammate_damage": 0,
                        "self_damage": 0,
                        "unknown_damage": 0,
                        "total_armor_damage": 0,
                        "enemy_armor_damage": 0,
                        "teammate_armor_damage": 0,
                        "self_armor_damage": 0,
                        "unknown_armor_damage": 0,
                        "enemies_damaged": 0,
                        "teammates_damaged": 0,
                        "unknown_players_damaged": 0,
                        "self_damaged": False,
                        "enemy_kills": 0,
                        "teammate_kills": 0,
                        "self_kills": 0,
                        "unknown_kills": 0,
                    },
                    "details": {},
                }
            ],
            ["round_number", "sequence_in_round", "event_id"],
        ),
        "events/objective_events.jsonl": (
            "objective_events",
            "stratai.objective_event@2",
            [
                {
                    "schema_id": "stratai.objective_event@2",
                    "match_id": MATCH_ID,
                    "event_id": carrier_id,
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "tick": 90,
                    "sequence_in_tick": 1,
                    "event_type": "bomb_carrier_snapshot",
                    "actor_player_id": PLAYER_ONE,
                    "actor_side": "t",
                    "site": None,
                    "position": {"x": 1, "y": 2, "z": 3},
                    "position_status": "observed",
                    "source": "game_state_snapshot",
                    "state_after": "carried",
                    "phase_after": "preplant",
                    "attempt_id": None,
                    "attempt_outcome": None,
                    "attempt_start_observed": None,
                    "action_duration_ms": None,
                    "has_defuse_kit": None,
                    "bomb_entity_id": 77,
                },
                {
                    "schema_id": "stratai.objective_event@2",
                    "match_id": MATCH_ID,
                    "event_id": plant_start_id,
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "tick": 100,
                    "sequence_in_tick": 1,
                    "event_type": "bomb_plant_start",
                    "actor_player_id": PLAYER_ONE,
                    "actor_side": "t",
                    "site": "A",
                    "position": {"x": 10, "y": 20, "z": 30},
                    "position_status": "observed",
                    "source": "demoinfocs_event",
                    "state_after": "planting",
                    "phase_after": "planting",
                    "attempt_id": plant_attempt_id,
                    "attempt_outcome": "in_progress",
                    "attempt_start_observed": True,
                    "action_duration_ms": None,
                    "has_defuse_kit": None,
                    "bomb_entity_id": 77,
                },
                {
                    "schema_id": "stratai.objective_event@2",
                    "match_id": MATCH_ID,
                    "event_id": plant_id,
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "tick": 104,
                    "sequence_in_tick": 1,
                    "event_type": "bomb_plant",
                    "actor_player_id": PLAYER_ONE,
                    "actor_side": "t",
                    "site": "A",
                    "position": {"x": 10, "y": 20, "z": 30},
                    "position_status": "observed",
                    "source": "demoinfocs_event",
                    "state_after": "planted",
                    "phase_after": "planted",
                    "attempt_id": plant_attempt_id,
                    "attempt_outcome": "completed",
                    "attempt_start_observed": True,
                    "action_duration_ms": 63,
                    "has_defuse_kit": None,
                    "bomb_entity_id": 77,
                },
                {
                    "schema_id": "stratai.objective_event@2",
                    "match_id": MATCH_ID,
                    "event_id": defuse_start_id,
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "tick": 110,
                    "sequence_in_tick": 1,
                    "event_type": "bomb_defuse_start",
                    "actor_player_id": PLAYER_TWO,
                    "actor_side": "ct",
                    "site": "A",
                    "position": {"x": 11, "y": 21, "z": 30},
                    "position_status": "observed",
                    "source": "demoinfocs_event",
                    "state_after": "defusing",
                    "phase_after": "defusing",
                    "attempt_id": defuse_attempt_id,
                    "attempt_outcome": "in_progress",
                    "attempt_start_observed": True,
                    "action_duration_ms": None,
                    "has_defuse_kit": True,
                    "bomb_entity_id": 77,
                },
                {
                    "schema_id": "stratai.objective_event@2",
                    "match_id": MATCH_ID,
                    "event_id": defuse_id,
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "tick": 120,
                    "sequence_in_tick": 1,
                    "event_type": "bomb_defuse",
                    "actor_player_id": PLAYER_TWO,
                    "actor_side": "ct",
                    "site": "A",
                    "position": {"x": 11, "y": 21, "z": 30},
                    "position_status": "observed",
                    "source": "demoinfocs_event",
                    "state_after": "defused",
                    "phase_after": "resolved",
                    "attempt_id": defuse_attempt_id,
                    "attempt_outcome": "completed",
                    "attempt_start_observed": True,
                    "action_duration_ms": 156,
                    "has_defuse_kit": True,
                    "bomb_entity_id": 77,
                },
            ],
            ["round_number", "tick", "sequence_in_tick", "event_id"],
        ),
        "states/player_states/round_001.jsonl": (
            "player_states",
            "stratai.player_state@3",
            [
                {
                    "schema_id": "stratai.player_state@3",
                    "match_id": MATCH_ID,
                    "state_id": f"{MATCH_ID}:state:001:000000090:{PLAYER_ONE}",
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "tick": 90,
                    "player_id": PLAYER_ONE,
                    "team_id": "team_b",
                    "side": "t",
                    "position": {"x": 1, "y": 2, "z": 3},
                    "health": 100,
                    "armor": 0,
                    "is_alive": True,
                    "horizontal_velocity_world_units_per_second": 5.0,
                    "velocity_vector_world_units_per_second": {
                        "x": 3.0,
                        "y": 4.0,
                        "z": 0.0,
                    },
                    "velocity_source": "native",
                    "velocity_measurement_window_ticks": 0,
                    "active_weapon": "AK-47",
                    "active_weapon_status": "observed",
                    "last_observed_active_weapon": None,
                    "last_observed_active_weapon_tick": None,
                    "has_c4": True,
                    "has_defuse_kit": False,
                    "is_planting": False,
                    "is_defusing": False,
                    "round_time_remaining_ms": 100_000,
                    "objective_phase": "preplant",
                    "phase_time_remaining_ms": 100_000,
                    "round_clock_remaining_ms": 100_000,
                    "bomb_time_remaining_ms": None,
                },
                {
                    "schema_id": "stratai.player_state@3",
                    "match_id": MATCH_ID,
                    "state_id": f"{MATCH_ID}:state:001:000000090:{PLAYER_TWO}",
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "tick": 90,
                    "player_id": PLAYER_TWO,
                    "team_id": "team_a",
                    "side": "ct",
                    "position": {"x": 4, "y": 5, "z": 6},
                    "health": 100,
                    "armor": 0,
                    "is_alive": True,
                    "horizontal_velocity_world_units_per_second": None,
                    "velocity_vector_world_units_per_second": None,
                    "velocity_source": "insufficient_history",
                    "velocity_measurement_window_ticks": None,
                    "active_weapon": None,
                    "active_weapon_status": "unavailable",
                    "last_observed_active_weapon": "Knife",
                    "last_observed_active_weapon_tick": 80,
                    "has_c4": False,
                    "has_defuse_kit": True,
                    "is_planting": False,
                    "is_defusing": False,
                    "round_time_remaining_ms": 100_000,
                    "objective_phase": "preplant",
                    "phase_time_remaining_ms": 100_000,
                    "round_clock_remaining_ms": 100_000,
                    "bomb_time_remaining_ms": None,
                },
            ],
            ["tick", "player_id"],
        ),
        "states/tactical/sampling.json": (
            "tactical_sampling",
            "stratai.tactical_sampling@1",
            {
                "schema_id": "stratai.tactical_sampling@1",
                "match_id": MATCH_ID,
                "identity_semantics": "join_only",
                "join_keys": {"match_id": MATCH_ID},
                "sampling": {
                    "target_hz": 16,
                    "tick_rate": 64.0,
                    "period_ticks": 4.0,
                    "strategy": "round_anchored_right_closed_no_carry_forward",
                },
                "physical_row_count": 1,
                "oracle_row_count": 1,
                "gap_count": 15,
            },
            [],
        ),
        "states/tactical/observed.jsonl.gz": (
            "tactical_observations",
            "stratai.tactical_physical_observation@1",
            [{
                "schema_id": "stratai.tactical_physical_observation@1",
                "match_id": MATCH_ID,
                "identity_semantics": "join_only",
                "join_keys": {
                    "match_id": MATCH_ID,
                    "round_id": ROUND_ID,
                    "observer_id": PLAYER_ONE,
                    "subject_id": PLAYER_ONE,
                },
                "round_number": 1,
                "tick": 80,
                "availability_tick": 80,
                "status": "observed",
                "causal_role": "model_input_observation",
                "visibility_scope": "self",
                "source": "replay_player_state",
                "provenance": {
                    "source_artifact": "replay_data",
                    "source_schema_version": 1,
                    "source_round": 1,
                    "source_frame_tick": 80,
                    "geometry_status": "not_required",
                    "line_of_sight": None,
                },
                "state": valid_tactical_physical_state(),
            }],
            ["round_number", "tick", "join_keys.observer_id", "join_keys.subject_id"],
        ),
        "states/tactical/oracle.jsonl": (
            "tactical_oracle",
            "stratai.tactical_oracle_state@1",
            [{
                "schema_id": "stratai.tactical_oracle_state@1",
                "match_id": MATCH_ID,
                "identity_semantics": "join_only",
                "join_keys": {
                    "match_id": MATCH_ID,
                    "round_id": ROUND_ID,
                    "observer_id": None,
                    "subject_id": PLAYER_ONE,
                },
                "round_number": 1,
                "tick": 80,
                "availability_tick": 80,
                "status": "observed",
                "causal_role": "label_only_oracle",
                "visibility_scope": "oracle",
                "source": "replay_player_state",
                "provenance": {
                    "source_artifact": "replay_data",
                    "source_schema_version": 1,
                    "source_round": 1,
                    "source_frame_tick": 80,
                    "geometry_status": "not_applicable",
                    "line_of_sight": None,
                },
                "state": {
                    "physical": valid_tactical_physical_state(),
                    "health": 100,
                    "armor": 50,
                    "weapons": ["ak47"],
                    "has_defuse_kit": False,
                    "has_helmet": True,
                    "has_c4": True,
                    "flash_duration_seconds": 0.0,
                    "money": 800,
                },
            }],
            ["round_number", "tick", "join_keys.subject_id"],
        ),
        "states/tactical/gaps.jsonl": (
            "tactical_gaps",
            "stratai.tactical_sampling_gap@1",
            [
                {
                    "schema_id": "stratai.tactical_sampling_gap@1",
                    "match_id": MATCH_ID,
                    "identity_semantics": "join_only",
                    "join_keys": {
                        "match_id": MATCH_ID,
                        "round_id": ROUND_ID,
                        "observer_id": None,
                        "subject_id": None,
                    },
                    "round_number": 1,
                    "tick": tick,
                    "availability_tick": None,
                    "status": "unavailable",
                    "causal_role": "coverage_gap",
                    "visibility_scope": "sampling_gap",
                    "source": "replay_sampling",
                    "reason": "missing_replay_frame_for_target_window",
                    "provenance": {
                        "source_artifact": "replay_data",
                        "source_schema_version": 1,
                        "source_round": 1,
                        "source_frame_tick": None,
                        "geometry_status": "not_evaluated",
                        "line_of_sight": None,
                    },
                }
                for tick in range(84, 141, 4)
            ],
            ["round_number", "tick", "reason"],
        ),
        "derived/engagements.json": (
            "engagements",
            "stratai.engagements@2",
            {
                "schema_id": "stratai.engagements@2",
                "match_id": MATCH_ID,
                "config": {
                    "algorithm_version": "engagement_causal@2",
                    "tick_rate_hz": 64.0,
                    "pair_continuation_window_ms": 1500,
                    "pair_continuation_window_ticks": 96,
                    "multi_target_window_ms": 750,
                    "multi_target_window_ticks": 48,
                    "max_engagement_duration_ms": 5000,
                    "max_engagement_duration_ticks": 320,
                    "aggressor_prelude_window_ms": 500,
                    "aggressor_prelude_window_ticks": 32,
                },
                "engagements": [
                    {
                        "engagement_id": f"{MATCH_ID}:engagement:000001",
                        "round_id": ROUND_ID,
                        "round_number": 1,
                        "start_tick": 100,
                        "start_sequence_in_tick": 1,
                        "end_tick": 100,
                        "end_sequence_in_tick": 1,
                        "duration_ms": 0.0,
                        "engagement_type": "duel",
                        "initiator": {
                            "player_id": PLAYER_ONE,
                            "status": "inferred",
                            "source": "first_damage_fallback",
                            "availability_tick": 100,
                            "source_event_ids": [combat_id],
                            "confidence": 0.75,
                        },
                        "first_aggressor": {
                            "player_id": None,
                            "status": "unavailable",
                            "source": "unavailable",
                            "availability_tick": None,
                            "source_event_ids": [],
                            "confidence": None,
                        },
                        "first_damage_dealer": {
                            "player_id": PLAYER_ONE,
                            "status": "observed",
                            "source": "first_player_hurt",
                            "availability_tick": 100,
                            "source_event_ids": [combat_id],
                            "confidence": 1.0,
                        },
                        "participants": [
                            {
                                "player_id": PLAYER_ONE,
                                "side": "t",
                                "roles": ["first_damage_dealer", "initiator"],
                            },
                            {"player_id": PLAYER_TWO, "side": "ct", "roles": []},
                        ],
                        "exchanges": [
                            {
                                "exchange_id": combat_id,
                                "tick": 100,
                                "sequence_in_tick": 1,
                                "sequence_in_round": 1,
                                "actor_player_id": PLAYER_ONE,
                                "target_player_id": PLAYER_TWO,
                                "is_kill": False,
                                "kill_event_id": None,
                                "source_event_ids": [combat_id],
                            }
                        ],
                        "causal_context": {
                            "t0_tick": 100,
                            "t0_sequence_in_tick": 1,
                            "participant_states": [
                                {
                                    "player_id": PLAYER_ONE,
                                    "state_id": f"{MATCH_ID}:state:001:000000090:{PLAYER_ONE}",
                                    "availability_tick": 90,
                                    "status": "observed",
                                    "source": "player_state@3",
                                    "side": "t",
                                    "position": {"x": 1, "y": 2, "z": 3},
                                    "position_status": "observed",
                                    "horizontal_velocity_world_units_per_second": 5.0,
                                    "velocity_status": "observed",
                                    "velocity_source": "native",
                                    "velocity_measurement_window_ticks": 0,
                                    "movement_classification": "hold",
                                    "active_weapon": "AK-47",
                                    "active_weapon_status": "observed",
                                    "active_weapon_source": "player_state@3",
                                    "health": 100,
                                    "armor": 0,
                                    "is_alive": True,
                                    "objective_phase": "preplant",
                                    "round_clock_remaining_ms": 100_000,
                                    "bomb_time_remaining_ms": None,
                                },
                                {
                                    "player_id": PLAYER_TWO,
                                    "state_id": f"{MATCH_ID}:state:001:000000090:{PLAYER_TWO}",
                                    "availability_tick": 90,
                                    "status": "observed",
                                    "source": "player_state@3",
                                    "side": "ct",
                                    "position": {"x": 4, "y": 5, "z": 6},
                                    "position_status": "observed",
                                    "horizontal_velocity_world_units_per_second": None,
                                    "velocity_status": "unavailable",
                                    "velocity_source": "insufficient_history",
                                    "velocity_measurement_window_ticks": None,
                                    "movement_classification": None,
                                    "active_weapon": None,
                                    "active_weapon_status": "unavailable",
                                    "active_weapon_source": "unavailable",
                                    "health": 100,
                                    "armor": 0,
                                    "is_alive": True,
                                    "objective_phase": "preplant",
                                    "round_clock_remaining_ms": 100_000,
                                    "bomb_time_remaining_ms": None,
                                },
                            ],
                            "initial_distance_world_units": 5.196152422706632,
                            "initial_distance_status": "derived",
                            "initial_distance_source": "player_state@3",
                            "bomb_context_status": "observed",
                            "economy_context_status": "unavailable",
                            "enemies_exposed_count": None,
                            "enemies_exposed_status": "unavailable",
                            "source_state_ids": [
                                f"{MATCH_ID}:state:001:000000090:{PLAYER_ONE}",
                                f"{MATCH_ID}:state:001:000000090:{PLAYER_TWO}",
                            ],
                        },
                        "outcome_context": {
                            "outcome": "disengaged",
                            "winner_player_id": None,
                            "loser_player_ids": [],
                            "terminal_kill_event_ids": [],
                            "trade_candidate_ids": [],
                            "trade_completion_ids": [],
                            "survival_status": "observed_alive_at_last_exchange",
                            "disengagement_status": "derived_window_closed_without_kill",
                        },
                        "source_event_ids": [combat_id],
                        "algorithm_version": "engagement_causal@2",
                    }
                ],
            },
            ["round_number", "start_tick", "start_sequence_in_tick", "engagement_id"],
        ),
        "derived/trades.json": (
            "trades",
            "stratai.trades@1",
            {
                "schema_id": "stratai.trades@1",
                "match_id": MATCH_ID,
                "config": {
                    "algorithm_version": "trade_response@2",
                    "tick_rate_hz": 64.0,
                    "trade_window_ms": 5000,
                    "trade_window_ticks": 320,
                    "max_distance_world_units": 1250.0,
                    "assumed_movement_speed_world_units_per_second": 250.0,
                    "max_facing_delta_deg": 100.0,
                    "physical_evidence_requirement": "alive+distance+connection_time+physics_mesh_los+orientation",
                },
                "trade_candidates": [],
                "trade_completions": [],
            },
            ["round_number", "death_tick", "death_sequence_in_tick", "trade_candidate_id"],
        ),
		"derived/economy_rounds.json": (
			"economy_rounds",
			"stratai.economy_round@1",
            {
				"schema_id": "stratai.economy_round@1",
                "match_id": MATCH_ID,
				"economy_rules": {"version": "stratai.cs2_economy_rules@1", "checksum_sha256": "c" * 64, "effective_from": "2026-08-19", "applicability_status": "unverified_match_date", "source": "test"},
				"rounds": [
					{"round_id": ROUND_ID, "round_number": 1, "team_id": "team_a", "side": "ct", "outcome": "win", "win_reason": "bomb_defused", "loss_bonus": {"level": 1, "amount": 1900, "status": "calculated", "rules_version": "stratai.cs2_economy_rules@1"}, "money_start": {"amount": 800, "status": "observed", "source": "test"}, "money_freeze_end": {"amount": 800, "status": "observed", "source": "test"}, "money_round_end": {"amount": 800, "status": "observed", "source": "test"}, "rewards": [], "diagnostics": {}},
					{"round_id": ROUND_ID, "round_number": 1, "team_id": "team_b", "side": "t", "outcome": "loss", "win_reason": "bomb_defused", "loss_bonus": {"level": 1, "amount": 1900, "status": "calculated", "rules_version": "stratai.cs2_economy_rules@1"}, "money_start": {"amount": 800, "status": "observed", "source": "test"}, "money_freeze_end": {"amount": 800, "status": "observed", "source": "test"}, "money_round_end": {"amount": 800, "status": "observed", "source": "test"}, "rewards": [], "diagnostics": {}},
				],
            },
			["round_number", "team_id"],
        ),
		"derived/economy_players.json": (
			"economy_players",
			"stratai.economy_player@1",
			{
				"schema_id": "stratai.economy_player@1",
				"match_id": MATCH_ID,
				"price_table": {"version": "stratai.cs2_prices@1", "checksum_sha256": "b" * 64, "effective_from": "2026-08-19", "applicability_status": "unverified_match_date", "source": "test"},
				"economy_rules": {"version": "stratai.cs2_economy_rules@1", "checksum_sha256": "c" * 64, "effective_from": "2026-08-19", "applicability_status": "unverified_match_date", "source": "test"},
				"players": [valid_economy_player(PLAYER_ONE, "team_b", "t"), valid_economy_player(PLAYER_TWO, "team_a", "ct")],
			},
			["round_number", "player_id"],
		),
		"derived/player_stats.json": (
			"player_stats",
			"stratai.player_stats@1",
            {
				"schema_id": "stratai.player_stats@1",
                "match_id": MATCH_ID,
				"players": [
					valid_player_stats(PLAYER_ONE, "team_b"),
					valid_player_stats(PLAYER_TWO, "team_a"),
				],
            },
            ["player_id"],
        ),
		"derived/clutch_events.json": (
			"clutch_events",
			"stratai.clutch_event@1",
			{"schema_id": "stratai.clutch_event@1", "match_id": MATCH_ID, "algorithm": {"version": "stratai.clutch_ledger@1", "checksum_sha256": "d" * 64, "effective_from": "2026-08-19", "applicability_status": "applicable", "source": "test"}, "clutch_events": []},
			["round_number", "team_id", "clutch_id"],
		),
		"causal/decisions.jsonl": (
			"decisions",
			"stratai.decision@1",
			[{
				"schema_id": "stratai.decision@1",
				"match_id": MATCH_ID,
				"decision_id": f"{MATCH_ID}:engagement:000001",
				"round_number": 1,
				"actor_player_id": PLAYER_ONE,
				"actor_id_usage": "join_only",
				"observed_state_ref": f"{MATCH_ID}:state:001:000000090:{PLAYER_ONE}",
				"state_availability_status": "observed",
				"t0_tick": 100,
				"decision_type": "peek_hold_or_reposition",
				"action_taken": "hold",
				"availability_tick": 100,
				"availability_status": "observed",
				"causal_role": "decision",
				"visibility_scope": "observable_proxy",
				"source": "engagements@2",
				"source_record_id": f"{MATCH_ID}:engagement:000001",
				"source_event_ids": [combat_id],
				"algorithm_version": "stratai.decision_projection@1",
			}],
			["round_number", "t0_tick", "decision_id"],
		),
        "causal/decision_features.jsonl": (
            "decision_features",
            "stratai.decision_features@1",
            [{
                "schema_id": "stratai.decision_features@1",
                "match_id": MATCH_ID,
                "decision_id": f"{MATCH_ID}:engagement:000001",
				"decision_type": "peek_hold_or_reposition",
                "round_number": 1,
                "t0_tick": 100,
                "availability_tick_max": 90,
                "participant_count": 2,
                "observed_participant_states": 2,
                "alive_participant_count": 2,
                "initial_distance_world_units": None,
                "initial_distance_status": "unavailable",
                "bomb_context_status": "observed",
                "economy_context_status": "unavailable",
                "enemies_exposed_count": None,
                "enemies_exposed_status": "unavailable",
                "round_clock_remaining_ms": 100000,
                "bomb_time_remaining_ms": None,
                "source_state_count": 2,
                "trade_possible": None,
                "trade_possible_status": "unavailable",
                "nearest_teammate_distance_world_units": None,
                "nearest_teammate_distance_status": "unavailable",
                "nearest_connection_time_ms": None,
                "nearest_connection_time_status": "unavailable",
                "any_line_of_sight": None,
                "line_of_sight_status": "unavailable",
                "minimum_facing_delta_deg": None,
                "facing_status": "unavailable",
            }],
            ["round_number", "t0_tick", "decision_id"],
        ),
        "causal/oracle_context.jsonl": (
            "oracle_context",
            "stratai.oracle_context@1",
            [{
                "schema_id": "stratai.oracle_context@1",
                "match_id": MATCH_ID,
                "decision_id": f"{MATCH_ID}:engagement:000001",
                "round_number": 1,
                "t0_tick": 100,
                "status": "unavailable",
                "available": False,
                "field_names": [],
                "abstentions": ["hidden-state oracle unavailable"],
            }],
            ["round_number", "t0_tick", "decision_id"],
        ),
        "causal/outcomes.jsonl": (
            "decision_outcomes",
            "stratai.decision_outcome@1",
            [{
                "schema_id": "stratai.decision_outcome@1",
                "match_id": MATCH_ID,
                "decision_id": f"{MATCH_ID}:engagement:000001",
                "round_number": 1,
                "t0_tick": 100,
                "outcome": "disengaged",
                "outcome_tick": 100,
                "duration_ms": 0,
                "winner_observed": False,
                "loser_count": 0,
                "terminal_kill_count": 0,
                "trade_candidate_count": 0,
                "trade_completion_count": 0,
                "survival_status": "observed",
                "disengagement_status": "observed",
                "horizons": [
                    {"horizon_seconds": seconds, "status": "derived_outcome_only", "outcome": "disengaged", "source": "engagements@2"}
                    for seconds in (2, 5, 10)
                ],
            }],
            ["round_number", "t0_tick", "decision_id"],
        ),
        "causal/quality_masks.jsonl": (
            "quality_masks",
            "stratai.quality_mask@1",
            [{
                "schema_id": "stratai.quality_mask@1",
                "match_id": MATCH_ID,
                "decision_id": f"{MATCH_ID}:engagement:000001",
                "round_number": 1,
                "t0_tick": 100,
                "available_fields": ["alive_participant_count", "participant_count"],
                "unavailable_fields": ["bomb_time_remaining_ms", "economy_context"],
                "inferred_fields": [],
                "warning_flags": ["coverage gap"],
            }],
            ["round_number", "t0_tick", "decision_id"],
        ),
        "diagnostics/quality_report.json": (
            "quality_report",
            "stratai.quality_report@1",
            {
                "schema_id": "stratai.quality_report@1",
                "match_id": MATCH_ID,
                "report": {
                    "schema_version": validator.QUALITY_SCHEMA_VERSION,
                    "status": "warning",
                    "usable_for_training": True,
                    "parse_completed": True,
                    "checks": [
						{
							"name": name,
							"status": "warning"
							if name == "engagement_observation_coverage"
							else "pass",
							"actual": "2"
							if name == "engagement_observation_coverage"
							else "0",
						}
						for name in sorted(
						validator.REQUIRED_UTILITY_QUALITY_CHECKS
							| validator.REQUIRED_COMBAT_QUALITY_CHECKS
							| validator.REQUIRED_ENGAGEMENT_QUALITY_CHECKS
							| validator.REQUIRED_BLOCK6_QUALITY_CHECKS
						)
                    ],
                    **{name: 0 for name in validator.REQUIRED_UTILITY_QUALITY_METRICS},
                    "utility_throws": 1,
                    "utility_canonical_events": 1,
                    "utility_throw_callbacks": 1,
                    "utility_bounce_callbacks": 1,
                    "utility_lifecycle_callbacks": 2,
                    "utility_player_flashed_callbacks": 1,
                    "utility_flash_effects": 1,
                    "utility_observed_effect_correlations": 1,
                    "utility_callback_diagnostics": {
                        "throws": {
                            "observed": 1,
                            "exact_correlated": 1,
                            "inferred_correlated": 0,
                            "orphaned": 0,
                            "deduplicated": 0,
                            "unmatched": 0,
                        },
                        "bounces": {
                            "observed": 1,
                            "exact_correlated": 1,
                            "inferred_correlated": 0,
                            "orphaned": 0,
                            "deduplicated": 0,
                            "unmatched": 0,
                        },
                        "lifecycle": {
                            "observed": 2,
                            "exact_correlated": 2,
                            "inferred_correlated": 0,
                            "orphaned": 0,
                            "deduplicated": 0,
                            "unmatched": 0,
                        },
                        "player_flashed": {
                            "observed": 1,
                            "exact_correlated": 1,
                            "inferred_correlated": 0,
                            "orphaned": 0,
                            "deduplicated": 0,
                            "unmatched": 0,
                        },
                        "damage": {
                            "observed": 0,
                            "exact_correlated": 0,
                            "inferred_correlated": 0,
                            "orphaned": 0,
                            "deduplicated": 0,
                            "unmatched": 0,
                        },
                    },
					"grenade_events": 1,
					**valid_combat_quality(),
					**{name: 0 for name in validator.HARD_BLOCK6_QUALITY_METRICS},
					**{name: 0 for name in validator.HARD_ENGAGEMENT_QUALITY_METRICS},
					"engagements": 1,
					"trade_candidates": 0,
					"trade_completions": 0,
					"engagement_observation_warnings": 2,
                    **{
                        metric: 0
                        for metric in (
                            "block7_artifact_integrity_violations",
                            "block7_causal_availability_violations",
                            "block7_future_leakage_violations",
                            "block7_schema_compatibility_violations",
                            "block7_determinism_violations",
                            "block7_corpus_quality_violations",
                        )
                    },
                    "domains": valid_block7_domains(),
                },
            },
            [],
        ),
        "presentation/replay/round_001.json.gz": (
            "replay_round",
			"stratai.replay_round@5",
            {
				"schema_id": "stratai.replay_round@5",
                "match_id": MATCH_ID,
                "round": {
                    "round": 1,
                    "start_tick": 80,
                    "end_tick": 140,
                    "winner": "CT",
                    "frames": [],
                    "events": replay_events,
				"combat_shots": [],
                },
            },
            [],
        ),
        "presentation/replay/index.json": (
            "replay_index",
			"stratai.replay_index@5",
            {
				"schema_id": "stratai.replay_index@5",
                "match_id": MATCH_ID,
                "metadata": {
                    "schema_version": validator.REPLAY_SCHEMA_VERSION,
                    "match_id": MATCH_ID,
                    "map_name": "de_mirage",
                    "tick_rate": 64.0,
                    "sample_rate_ms": 62,
                    "map_config": {"pos_x": 0, "pos_y": 0, "scale": 1},
                },
                "sample_stride_ticks": 4,
                "sample_interval_ms": 62.5,
                "rounds": [
                    {
                        "round_number": 1,
                        "start_tick": 80,
                        "end_tick": 140,
                        "winner_side": "ct",
                        "frame_count": 0,
                        "event_count": len(replay_events),
                        "events": replay_events,
                        "path": "presentation/replay/round_001.json.gz",
                        "sha256": "0" * 64,
                        "bytes": 0,
                    }
                ],
            },
            [],
        ),
    }


def write_valid_bundle(tmp_path: Path) -> Path:
    match_dir = tmp_path / f"match_{MATCH_ID}"
    match_dir.mkdir()
    descriptors = []
    for relative_path, (
        artifact_type,
        schema_id,
        payload,
        sort_order,
    ) in build_payloads().items():
        artifact_path = match_dir / "canonical" / relative_path
        spec = validator.SPEC_BY_TYPE[artifact_type]
        file_format = spec.file_format
        if file_format == "jsonl":
            assert isinstance(payload, list)
            write_jsonl(artifact_path, payload)
            record_count = len(payload)
        else:
            if artifact_type == "replay_index":
                payload = deepcopy(payload)
                round_path = (
                    match_dir / "canonical/presentation/replay/round_001.json.gz"
                )
                payload["rounds"][0]["sha256"] = sha256(round_path)
                payload["rounds"][0]["bytes"] = round_path.stat().st_size
            if spec.compression == "gzip":
                write_gzip_json(artifact_path, payload)
            else:
                write_json(artifact_path, payload)
            record_count = len(payload[spec.records_field]) if spec.records_field else 1
        descriptor = {
            "artifact_type": artifact_type,
            "path": relative_path,
            "schema_id": schema_id,
            "format": file_format,
            "record_count": record_count,
            "sha256": sha256(artifact_path),
            "bytes": artifact_path.stat().st_size,
            "sort_order": sort_order,
        }
        if validator.SPEC_BY_TYPE[artifact_type].compression is not None:
            descriptor["compression"] = validator.SPEC_BY_TYPE[
                artifact_type
            ].compression
        descriptors.append(descriptor)
    sorted_descriptors = sorted(descriptors, key=lambda descriptor: descriptor["path"])
    configuration_hashes = {
        "price_table": "b" * 64,
        "economy_rules": "c" * 64,
        "clutch_algorithm": "d" * 64,
        "rating_algorithm": "e" * 64,
        "stats_algorithm": "f" * 64,
    }
    write_json(
        match_dir / "canonical" / "manifest.json",
        {
            "schema_id": validator.MANIFEST_SCHEMA_ID,
            "export_format_version": validator.EXPORT_FORMAT_VERSION,
            "match_id": MATCH_ID,
			"demo_checksum_sha256": DEMO_CHECKSUM,
			"parser_version": validator.PARSER_SCHEMA_VERSION,
			"configuration_hashes": configuration_hashes,
			"transformation_versions": {"economy": "@1", "stats": "@1", "clutch": "@1", "metadata": "@1"},
            "lineage": {
                "demo_checksum_sha256": DEMO_CHECKSUM,
                "parser_version": validator.PARSER_SCHEMA_VERSION,
                "demoinfocs_version": "v4-test",
                "export_format_version": validator.EXPORT_FORMAT_VERSION,
                "build_identifier": {"value": None, "status": "unavailable", "source": "test build id absent"},
                "map_name": "de_mirage",
                "tick_rate_hz": 64.0,
                "tick_rate_rules_version": "stratai.tick_rate_rules@1",
                "price_table_version": "stratai.cs2_prices@1",
                "price_table_sha256": "b" * 64,
                "algorithm_versions": {"engagement": "engagement_causal@2", "quality_gate": "stratai.quality_gate@1"},
                "schema_versions": {
                    **{descriptor["artifact_type"]: descriptor["schema_id"] for descriptor in sorted_descriptors},
                    "canonical_manifest": validator.MANIFEST_SCHEMA_ID,
                },
                "validator_version": validator.VALIDATOR_VERSION,
                "processing_timestamp": {"value": None, "status": "operational_only", "source": None},
                "metadata_source": {"source": "demo", "checksum": DEMO_CHECKSUM},
                "input_hashes": {
                    name: {"value": None, "status": "unavailable", "source": "not exposed in fixture"}
                    for name in ("physics_map", "nav_mesh", "callouts")
                },
                "configuration_hashes": configuration_hashes,
                "quality_flags": [],
                "warnings": [],
                "abstentions": ["input hashes unavailable"],
                "golden_corpus_version": validator.GOLDEN_CORPUS_VERSION,
                "golden_corpus_manifest_id": "golden-demos-v2",
            },
            "artifacts": sorted_descriptors,
        },
    )
    root_artifacts = []
    for artifact_path in sorted(
        path for path in match_dir.rglob("*") if path.is_file()
    ):
        relative_path = artifact_path.relative_to(match_dir).as_posix()
        if relative_path == "manifest.json":
            continue
        root_artifacts.append(
            {
                "artifact_type": "test_artifact",
                "path": relative_path,
                "format": "json",
                "compression": "none",
                "sha256": sha256(artifact_path),
                "bytes": artifact_path.stat().st_size,
            }
        )
    write_json(
        match_dir / "manifest.json",
        {
            "match_id": MATCH_ID,
            "checksum": DEMO_CHECKSUM,
            "parser_schema_version": validator.PARSER_SCHEMA_VERSION,
            "committed_at": "2026-08-13T00:00:00Z",
            "export_format_version": validator.EXPORT_FORMAT_VERSION,
            "validator_version": validator.VALIDATOR_VERSION,
            "validation_status": "passed",
            "artifacts": root_artifacts,
        },
    )
    return match_dir


def refresh_descriptor(match_dir: Path, relative_path: str) -> None:
    manifest_path = match_dir / "canonical" / "manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    artifact_path = match_dir / "canonical" / relative_path
    descriptor = next(
        item for item in manifest["artifacts"] if item["path"] == relative_path
    )
    descriptor["sha256"] = sha256(artifact_path)
    descriptor["bytes"] = artifact_path.stat().st_size
    if descriptor["format"] == "jsonl":
        descriptor["record_count"] = len(read_jsonl(artifact_path))
    write_json(manifest_path, manifest)
    refresh_root_catalog(match_dir)


def read_combat_events(match_dir: Path) -> tuple[Path, list[dict]]:
    path = match_dir / "canonical/events/combat_events.jsonl"
    records = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    return path, records


def write_combat_events(match_dir: Path, path: Path, records: list[dict]) -> None:
    write_jsonl(path, records)
    refresh_descriptor(match_dir, "events/combat_events.jsonl")


def read_player_states(match_dir: Path) -> tuple[Path, list[dict]]:
    path = match_dir / "canonical/states/player_states/round_001.jsonl"
    records = [
        json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()
    ]
    return path, records


def write_player_states(match_dir: Path, path: Path, records: list[dict]) -> None:
    write_jsonl(path, records)
    refresh_descriptor(match_dir, "states/player_states/round_001.jsonl")


def read_objective_events(match_dir: Path) -> tuple[Path, list[dict]]:
    path = match_dir / "canonical/events/objective_events.jsonl"
    records = [
        json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()
    ]
    return path, records


def write_objective_events(match_dir: Path, path: Path, records: list[dict]) -> None:
    write_jsonl(path, records)
    refresh_descriptor(match_dir, "events/objective_events.jsonl")


def read_utility_events(match_dir: Path) -> tuple[Path, list[dict]]:
    path = match_dir / "canonical/events/utility_events.jsonl"
    records = [
        json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()
    ]
    return path, records


def write_utility_events(match_dir: Path, path: Path, records: list[dict]) -> None:
    write_jsonl(path, records)
    refresh_descriptor(match_dir, "events/utility_events.jsonl")


def read_quality_report(match_dir: Path) -> tuple[Path, dict]:
    path = match_dir / "canonical/diagnostics/quality_report.json"
    return path, json.loads(path.read_text(encoding="utf-8"))


def write_quality_report(match_dir: Path, path: Path, payload: dict) -> None:
    write_json(path, payload)
    refresh_descriptor(match_dir, "diagnostics/quality_report.json")


def read_replay_round(match_dir: Path) -> tuple[Path, dict]:
    path = match_dir / "canonical/presentation/replay/round_001.json.gz"
    with gzip.open(path, "rt", encoding="utf-8") as file:
        return path, json.load(file)


def write_replay_round(match_dir: Path, path: Path, payload: dict) -> None:
    write_gzip_json(path, payload)
    refresh_descriptor(match_dir, "presentation/replay/round_001.json.gz")


def read_rounds(match_dir: Path) -> tuple[Path, dict]:
    path = match_dir / "canonical/core/rounds.json"
    return path, json.loads(path.read_text(encoding="utf-8"))


def write_rounds(match_dir: Path, path: Path, payload: dict) -> None:
    write_json(path, payload)
    refresh_descriptor(match_dir, "core/rounds.json")


def read_engagements(match_dir: Path) -> tuple[Path, dict]:
    path = match_dir / "canonical/derived/engagements.json"
    return path, json.loads(path.read_text(encoding="utf-8"))


def write_engagements(match_dir: Path, path: Path, payload: dict) -> None:
    write_json(path, payload)
    refresh_descriptor(match_dir, "derived/engagements.json")


def read_canonical_json(match_dir: Path, relative_path: str) -> tuple[Path, dict]:
    path = match_dir / "canonical" / relative_path
    return path, json.loads(path.read_text(encoding="utf-8"))


def write_canonical_json(
    match_dir: Path, relative_path: str, path: Path, payload: dict
) -> None:
    write_json(path, payload)
    refresh_descriptor(match_dir, relative_path)


def refresh_root_catalog(match_dir: Path) -> None:
    manifest_path = match_dir / "manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    for descriptor in manifest["artifacts"]:
        artifact_path = match_dir / descriptor["path"]
        descriptor["sha256"] = sha256(artifact_path)
        descriptor["bytes"] = artifact_path.stat().st_size
    write_json(manifest_path, manifest)


def test_valid_bundle_has_no_errors(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)

    assert validator.validate_match_export(match_dir) == []


def test_participants_reject_zero_steam_identity(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, participants = read_canonical_json(match_dir, "core/participants.json")
    participants["players"][0]["player_id"] = "steam:0"
    participants["players"][0]["steam_id"] = "0"
    write_canonical_json(match_dir, "core/participants.json", path, participants)

    errors = validator.validate_match_export(match_dir)

    assert any("decimal positivo" in error for error in errors)


def test_large_tactical_jsonl_bypass_list_loader(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    original_loader = validator.load_json_lines

    def guarded_loader(path: Path, errors: list[str]) -> list[dict]:
        if (
            path.name in {"observed.jsonl.gz", "oracle.jsonl"}
            and path.parent.name == "tactical"
        ):
            raise AssertionError("large tactical JSONL must not be retained as a list")
        return original_loader(path, errors)

    monkeypatch.setattr(validator, "load_json_lines", guarded_loader)

    assert validator.validate_match_export(match_dir) == []


@pytest.mark.parametrize(
    ("relative_path", "mutation", "expected_error"),
    [
        (
            "states/tactical/observed.jsonl.gz",
            "unknown_round",
            "canonical/tactical_observations: referencia un round_id desconocido",
        ),
        (
            "states/tactical/observed.jsonl.gz",
            "round_number_mismatch",
            "canonical/tactical_observations: round_id y round_number discrepan",
        ),
        (
            "states/tactical/observed.jsonl.gz",
            "unknown_player_id",
            "canonical/tactical_observations: owner_player_id referencia un jugador desconocido",
        ),
        (
            "states/tactical/oracle.jsonl",
            "unknown_player_ids",
            "canonical/tactical_oracle: related_player_ids contiene jugadores desconocidos",
        ),
        (
            "states/tactical/oracle.jsonl",
            "unknown_source_event",
            "canonical/tactical_oracle: source_event_ids contiene eventos desconocidos",
        ),
    ],
)
def test_streamed_tactical_records_preserve_generic_reference_checks(
    tmp_path: Path,
    relative_path: str,
    mutation: str,
    expected_error: str,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path = match_dir / "canonical" / relative_path
    rows = read_jsonl(path)
    row = rows[0]
    if mutation == "unknown_round":
        row["round_id"] = f"{MATCH_ID}:round:999"
    elif mutation == "round_number_mismatch":
        row["round_id"] = ROUND_ID
        row["round_number"] = 2
    elif mutation == "unknown_player_id":
        row["reference_probe"] = {"owner_player_id": "steam:999"}
    elif mutation == "unknown_player_ids":
        row["reference_probe"] = {"related_player_ids": ["steam:999"]}
    else:
        assert mutation == "unknown_source_event"
        row["source_event_ids"] = ["unknown-event"]
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert expected_error in validator.validate_match_export(match_dir)


def test_streamed_reference_validation_releases_rows_as_they_are_consumed() -> None:
    class TrackableRow(dict):
        pass

    references: list[weakref.ReferenceType[TrackableRow]] = []
    peak_alive = 0

    def rows() -> Iterator[dict]:
        nonlocal peak_alive
        for _ in range(250):
            row = TrackableRow(
                {
                    "round_id": ROUND_ID,
                    "round_number": 1,
                    "owner_player_id": PLAYER_ONE,
                    "related_player_ids": [PLAYER_TWO],
                    "source_event_ids": ["event:known"],
                }
            )
            references.append(weakref.ref(row))
            peak_alive = max(
                peak_alive,
                sum(reference() is not None for reference in references),
            )
            yield row

    errors: list[str] = []

    def consume() -> None:
        for _ in validator.iter_records_with_valid_references(
            "tactical_observations",
            rows(),
            {ROUND_ID: 1},
            {PLAYER_ONE, PLAYER_TWO},
            {"event:known"},
            errors,
        ):
            pass

    consume()
    gc.collect()

    assert errors == []
    assert peak_alive <= 2
    assert all(reference() is None for reference in references)


def test_streamed_reference_validation_matches_eager_reference_validation() -> None:
    rows = [
        {
            "round_id": f"{MATCH_ID}:round:999",
            "round_number": 1,
            "reference_probe": {
                "owner_player_id": "steam:999",
                "related_player_ids": ["steam:999"],
            },
            "source_event_ids": ["event:unknown"],
        },
        {"round_id": ROUND_ID, "round_number": 2},
    ]
    round_numbers = {ROUND_ID: 1}
    player_ids = {PLAYER_ONE, PLAYER_TWO}
    event_ids = {"event:known"}
    eager_errors: list[str] = []
    for row in rows:
        validator.validate_record_references(
            "tactical_observations",
            row,
            round_numbers,
            player_ids,
            event_ids,
            eager_errors,
        )

    streamed_errors: list[str] = []
    consumed = list(
        validator.iter_records_with_valid_references(
            "tactical_observations",
            iter(rows),
            round_numbers,
            player_ids,
            event_ids,
            streamed_errors,
        )
    )

    assert consumed == rows
    assert streamed_errors == eager_errors


def test_public_validator_caps_retained_errors_and_reports_omissions(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    match_dir = tmp_path / "match_test"
    match_dir.mkdir()

    def fake_root_validator(
        _match_dir: Path,
        _match_id: str,
        errors: list[str],
        _expected_checksum: str | None,
    ) -> None:
        for index in range(5):
            errors.append(f"root:{index}")

    def fake_canonical_validator(
        _match_dir: Path, _match_id: str, errors: list[str]
    ) -> None:
        for index in range(5):
            errors.append(f"canonical:{index}")

    monkeypatch.setattr(validator, "MAX_RETAINED_VALIDATION_ERRORS", 3)
    monkeypatch.setattr(validator, "validate_root_catalog", fake_root_validator)
    monkeypatch.setattr(
        validator, "validate_canonical_bundle", fake_canonical_validator
    )

    assert validator.validate_match_export(match_dir, expected_match_id="test") == [
        "root:0",
        "root:1",
        "root:2",
        "validation: se omitieron 7 errores adicionales tras alcanzar el limite de 3",
    ]


def test_block7_rejects_future_label_in_decision_features(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/decision_features.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["winner"] = "team_a"
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "campos futuros/prohibidos" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_block7_feature_allowlist_rejects_unknown_future_field(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/decision_features.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["health_after_5s"] = 0
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "campos fuera de allowlist" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_block7_observed_features_cannot_embed_oracle_or_outcomes(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/decision_features.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["oracle_context"] = {"enemy_position_x": 123.0}
    rows[0]["outcomes"] = {"health_after_5s": 0}
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    errors = validator.validate_match_export(match_dir)
    assert any("campos futuros/prohibidos" in error for error in errors)
    assert any("campos fuera de allowlist" in error for error in errors)


def test_block7_exact_actor_target_distance_requires_causal_visibility(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/decision_features.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["initial_distance_world_units"] = 320.0
    rows[0]["initial_distance_status"] = "derived"
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "distancia actor-target exacta pertenece a oracle" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_block7_unavailable_measurement_cannot_invent_zero(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/decision_features.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["enemies_exposed_count"] = 0
    rows[0]["enemies_exposed_status"] = "unavailable"
    rows[0]["trade_possible"] = False
    rows[0]["trade_possible_status"] = "unavailable"
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    errors = validator.validate_match_export(match_dir)
    assert any("enemies_exposed_count unavailable debe ser null" in error for error in errors)
    assert any("trade_possible unavailable debe ser null" in error for error in errors)


def test_block7_decision_type_must_align_across_partitions(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/decision_features.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["decision_type"] = "spacing_or_trade_connection"
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "decision_type no coincide con decisions" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_block7_outcomes_require_exact_2_5_10_second_horizons(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/outcomes.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["horizons"] = rows[0]["horizons"][:2]
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "faltan horizons 2/5/10" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_tactical_stream_rejects_silent_16hz_gap(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    gaps_relative = "states/tactical/gaps.jsonl"
    gaps_path = match_dir / "canonical" / gaps_relative
    gaps = [json.loads(line) for line in gaps_path.read_text(encoding="utf-8").splitlines()]
    gaps.pop(0)
    write_jsonl(gaps_path, gaps)
    refresh_descriptor(match_dir, gaps_relative)
    sampling_relative = "states/tactical/sampling.json"
    sampling_path, sampling = read_canonical_json(match_dir, sampling_relative)
    sampling["gap_count"] = len(gaps)
    write_canonical_json(match_dir, sampling_relative, sampling_path, sampling)

    assert any(
        "stream 16 Hz tiene huecos silenciosos" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_tactical_observed_partition_rejects_oracle_state(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "states/tactical/observed.jsonl.gz"
    path = match_dir / "canonical" / relative_path
    rows = read_jsonl(path)
    rows[0]["state"]["weapons"] = ["ak47", "knife"]
    rows[0]["state"]["has_helmet"] = True
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "schema/availability fisico incompleto" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_tactical_observation_cannot_be_available_after_tick(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "states/tactical/observed.jsonl.gz"
    path = match_dir / "canonical" / relative_path
    rows = read_jsonl(path)
    rows[0]["availability_tick"] = rows[0]["tick"] + 1
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "availability_tick debe ser <= tick" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_tactical_enemy_requires_loaded_geometry_and_positive_los(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "states/tactical/observed.jsonl.gz"
    path = match_dir / "canonical" / relative_path
    rows = read_jsonl(path)
    row = rows[0]
    row["join_keys"]["subject_id"] = PLAYER_TWO
    row["visibility_scope"] = "enemy_los"
    row["provenance"]["geometry_status"] = "not_loaded"
    row["provenance"]["line_of_sight"] = None
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "enemigo sin geometria/LOS fiable" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_block7_rejects_post_t0_availability(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/decision_features.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["availability_tick_max"] = rows[0]["t0_tick"] + 1
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "availability_tick_max debe ser <= t0_tick" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_block7_requires_all_twenty_quality_domains(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_quality_report(match_dir)
    payload["report"]["domains"] = payload["report"]["domains"][:-1]
    write_quality_report(match_dir, path, payload)

    assert any(
        "dominios Bloque 7 incorrectos" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_block7_rejects_non_string_quality_domain_name_without_crashing(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_quality_report(match_dir)
    payload["report"]["domains"][0]["name"] = ["not", "hashable"]
    write_quality_report(match_dir, path, payload)

    assert any(
        "name debe ser una cadena" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_block7_requires_explicit_lineage(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    manifest_path = match_dir / "canonical" / "manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest.pop("lineage")
    write_json(manifest_path, manifest)
    refresh_root_catalog(match_dir)

    assert any(
        "lineage debe ser un objeto" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_combat_relation_must_match_stable_ids_and_sides(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_combat_events(match_dir)
    records[0]["relation"] = "friendly"
    write_combat_events(match_dir, path, records)

    assert any(
        "relation no reconcilia" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_combat_health_and_armor_transitions_must_reconcile(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_combat_events(match_dir)
    records[0]["armor_after"] = 49
    write_combat_events(match_dir, path, records)

    assert any(
        "before/after no reconcilia" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_combat_hitgroup_requires_stable_semantic_label(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_combat_events(match_dir)
    records[0]["hitgroup"] = "1"
    records[0]["is_headshot"] = True
    write_combat_events(match_dir, path, records)

    assert any(
        "hitgroup/headshot no reconcilia" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_combat_source_event_ids_cannot_reference_unknown_events(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_combat_events(match_dir)
    records[0]["source_event_ids"] = ["unknown-combat-event"]
    write_combat_events(match_dir, path, records)

    assert any(
        "referencia causal invalida" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_combat_stats_are_recomputed_from_atomic_events(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "derived/player_stats.json"
    path = match_dir / "canonical" / relative_path
    payload = json.loads(path.read_text(encoding="utf-8"))
    payload["players"][0]["metrics"]["combat_damage_observed"] = 39
    write_json(path, payload)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "stats de combate no reconcilian" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_combat_callback_diagnostics_must_conserve_callbacks(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_quality_report(match_dir)
    payload["report"]["combat_callback_diagnostics"]["player_hurt"]["recorded"] = 0
    write_quality_report(match_dir, path, payload)

    assert any(
        "combat_callback_accounting_violations no reconcilia" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_combat_callback_diagnostics_accept_explicit_outside_round_discard(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_quality_report(match_dir)
    report = payload["report"]
    report["combat_callback_diagnostics"]["player_hurt"].update(
        observed=2, recorded=1, discarded=1
    )
    report["combat_discarded_callbacks"] = 1
    report["combat_discarded_callback_reasons"] = {
        "outside_official_round": 1
    }
    report["status"] = "warning"
    for check in report["checks"]:
        if check["name"] == "combat_observation_coverage":
            check["status"] = "warning"
    write_quality_report(match_dir, path, payload)

    assert validator.validate_match_export(match_dir) == []


def test_replay_player_references_reject_anonymous_player() -> None:
    replay = [{"round": {"frames": [{"players": [{"steam_id": "0"}]}]}}]
    errors: list[str] = []

    validator.validate_replay_player_references(replay, {PLAYER_ONE}, errors)

    assert errors == [
        "canonical/presentation/replay: frame contiene un jugador no reconciliado"
    ]


def test_combat_replay_projection_rejects_shot_provenance_drift() -> None:
    fire = {
        "event_id": "combat-fire-1",
        "event_type": "weapon_fire",
        "round_number": 1,
        "tick": 10,
        "shot_id": "shot-1",
        "actor_player_id": PLAYER_ONE,
        "weapon": "ak47",
        "actor_position_status": "observed",
        "actor_position_source": "callback_player_position",
        "shot_result": "miss",
        "shot_result_status": "derived",
    }
    shot = {
        "source_event_id": "combat-fire-1",
        "shot_id": "shot-1",
        "tick": 10,
        "shooter_id": "1",
        "weapon": "ak47",
        "position_status": "observed",
        "position_source": "callback_player_position",
        "endpoint_status": "unavailable",
        "endpoint_source": "unavailable",
        "result": "miss",
        "result_status": "derived",
        "hit": False,
    }
    combat_shot = dict(shot)
    shot["shooter_id"] = "2"
    replay = [{"round": {
        "round": 1,
        "events": [],
        "combat_shots": [combat_shot],
        "frames": [{"tick": 10, "shots": [shot]}],
    }}]
    errors: list[str] = []

    mismatches = validator.validate_replay_combat_projection([fire], replay, errors)

    assert mismatches == 1
    assert any("provenance invalida" in error for error in errors)


def test_combat_replay_projection_normalizes_unavailable_shot_outcome() -> None:
    fire = {
        "event_id": "combat-fire-1",
        "event_type": "weapon_fire",
        "round_number": 1,
        "tick": 10,
        "shot_id": "shot-1",
        "actor_player_id": PLAYER_ONE,
        "weapon": "ak47",
        "actor_position_status": "observed",
        "actor_position_source": "callback_player_position",
        "shot_result": None,
        "shot_result_status": "unavailable",
    }
    shot = {
        "source_event_id": "combat-fire-1",
        "shot_id": "shot-1",
        "tick": 10,
        "shooter_id": "1",
        "weapon": "ak47",
        "position_status": "observed",
        "position_source": "callback_player_position",
        "endpoint_status": "unavailable",
        "endpoint_source": "unavailable",
        "result": "unavailable",
        "result_status": "unavailable",
        "hit": False,
    }
    replay = [{"round": {
        "round": 1,
        "events": [],
        "combat_shots": [shot],
        "frames": [{"tick": 10, "shots": [shot]}],
    }}]
    errors: list[str] = []

    mismatches = validator.validate_replay_combat_projection([fire], replay, errors)

    assert mismatches == 0
    assert errors == []


def test_combat_replay_projection_accepts_zero_for_unavailable_shooter() -> None:
    fire = {
        "event_id": "combat-fire-1",
        "event_type": "weapon_fire",
        "round_number": 1,
        "tick": 10,
        "shot_id": "shot-1",
        "actor_status": "unavailable",
        "actor_source": "unavailable",
        "weapon": "ak47",
        "actor_position_status": "unavailable",
        "actor_position_source": "unavailable",
        "shot_result": "miss",
        "shot_result_status": "derived",
    }
    shot = {
        "source_event_id": "combat-fire-1",
        "shot_id": "shot-1",
        "tick": 10,
        "shooter_id": "0",
        "weapon": "ak47",
        "position_status": "unavailable",
        "position_source": "unavailable",
        "endpoint_status": "unavailable",
        "endpoint_source": "unavailable",
        "result": "miss",
        "result_status": "derived",
        "hit": False,
    }
    replay = [{"round": {
        "round": 1,
        "events": [],
        "combat_shots": [shot],
        "frames": [{"tick": 10, "shots": [shot]}],
    }}]
    errors: list[str] = []

    mismatches = validator.validate_replay_combat_projection([fire], replay, errors)

    assert mismatches == 0
    assert errors == []


def test_round_outcome_mismatch_is_rejected(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "core/rounds.json"
    path = match_dir / "canonical" / relative_path
    payload = json.loads(path.read_text(encoding="utf-8"))
    payload["rounds"][0]["winner_team_id"] = "team_b"
    write_json(path, payload)
    refresh_descriptor(match_dir, relative_path)

    errors = validator.validate_match_export(match_dir)

    assert any("scores por team_id no coinciden" in error for error in errors)


def test_canonical_only_bundle_does_not_require_legacy_data_files(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)

    legacy_data_files = {
        "metadata.json",
        "quality.json",
        "players_summary.json",
        "combat.json",
        "economy.json",
        "grenades.json",
        "tracking.json",
        "replay.json",
    }
    assert not any((match_dir / filename).exists() for filename in legacy_data_files)
    assert (
        validator.validate_match_export(match_dir, expected_demo_checksum=DEMO_CHECKSUM)
        == []
    )


def test_root_manifest_requires_v16_and_expected_demo_checksum(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    manifest_path = match_dir / "manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest["parser_schema_version"] = "v8"
    write_json(manifest_path, manifest)

    errors = validator.validate_match_export(match_dir, expected_demo_checksum="b" * 64)

    assert any("parser_schema_version debe ser v16" in error for error in errors)
    assert any("checksum no corresponde a la demo" in error for error in errors)


def test_tampered_artifact_is_detected_by_checksum(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path = match_dir / "canonical/events/combat_events.jsonl"
    path.write_text(path.read_text(encoding="utf-8") + "{}\n", encoding="utf-8")

    assert any(
        "sha256 no coincide" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_required_artifact_cannot_be_missing_from_staged_tree(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    (match_dir / "canonical/causal/decisions.jsonl").unlink()

    errors = validator.validate_match_export(match_dir)
    assert any("fichero declarado inexistente" in error for error in errors)


def test_artifact_declared_size_must_match_disk(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    manifest_path = match_dir / "canonical/manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    descriptor = next(
        item for item in manifest["artifacts"] if item["artifact_type"] == "decisions"
    )
    descriptor["bytes"] += 1
    write_json(manifest_path, manifest)
    refresh_root_catalog(match_dir)

    assert any(
        "canonical/causal/decisions.jsonl: bytes no coincide" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_artifact_record_count_must_match_decoded_rows(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    manifest_path = match_dir / "canonical/manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    descriptor = next(
        item for item in manifest["artifacts"] if item["artifact_type"] == "decisions"
    )
    descriptor["record_count"] += 1
    write_json(manifest_path, manifest)
    refresh_root_catalog(match_dir)

    assert any(
        "canonical/causal/decisions.jsonl: record_count no coincide" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_artifact_embedded_schema_must_match_descriptor(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "causal/decisions.jsonl"
    path = match_dir / "canonical" / relative_path
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    rows[0]["schema_id"] = "stratai.decision@999"
    write_jsonl(path, rows)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "registro 0 tiene schema_id incorrecto" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_unknown_player_reference_is_rejected(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "events/objective_events.jsonl"
    path = match_dir / "canonical" / relative_path
    records = [
        json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()
    ]
    records[0]["actor_player_id"] = "steam:999"
    write_jsonl(path, records)
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "jugador desconocido" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_declared_sort_order_is_enforced(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "states/player_states/round_001.jsonl"
    path = match_dir / "canonical" / relative_path
    records = [
        json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()
    ]
    write_jsonl(path, list(reversed(records)))
    refresh_descriptor(match_dir, relative_path)

    assert any(
        "fuera de sort_order" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_canonical_manifest_artifacts_must_be_sorted_by_path(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    manifest_path = match_dir / "canonical/manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest["artifacts"] = list(reversed(manifest["artifacts"]))
    write_json(manifest_path, manifest)
    refresh_root_catalog(match_dir)

    assert any(
        "artifacts no está ordenado por path" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_player_state_file_is_required_for_each_round(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path = match_dir / "canonical/states/player_states/round_001.jsonl"
    path.rename(path.with_name("round_002.jsonl"))
    manifest_path = match_dir / "canonical/manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    descriptor = next(
        item
        for item in manifest["artifacts"]
        if item["artifact_type"] == "player_states"
    )
    descriptor["path"] = "states/player_states/round_002.jsonl"
    write_json(manifest_path, manifest)

    assert any(
        "exactamente un fichero por ronda" in error
        for error in validator.validate_match_export(match_dir)
    )


def test_player_state_v3_requires_nullable_contract_fields(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    records[0].pop("last_observed_active_weapon")
    write_player_states(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("no contiene last_observed_active_weapon" in error for error in errors)


def test_dead_player_cannot_expose_velocity_or_current_weapon(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    records[0]["is_alive"] = False
    records[0]["active_weapon"] = "Knife"
    write_player_states(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any(
        "jugador muerto requiere velocity_source=not_applicable" in error
        for error in errors
    )
    assert any(
        "jugador muerto requiere active_weapon_status=not_applicable" in error
        for error in errors
    )


def test_dead_player_may_keep_last_observed_weapon_separate(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    records[1].update(
        {
            "is_alive": False,
            "health": 0,
            "velocity_source": "not_applicable",
            "active_weapon_status": "not_applicable",
        }
    )
    write_player_states(match_dir, path, records)
    engagement_path, engagement_payload = read_engagements(match_dir)
    causal_state = engagement_payload["engagements"][0]["causal_context"][
        "participant_states"
    ][1]
    causal_state.update(
        {
            "is_alive": False,
            "health": 0,
            "velocity_source": "not_applicable",
            "active_weapon_status": "not_applicable",
        }
    )
    write_engagements(match_dir, engagement_path, engagement_payload)

    assert validator.validate_match_export(match_dir) == []


def test_available_velocity_requires_finite_coherent_values(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    records[0]["horizontal_velocity_world_units_per_second"] = float("nan")
    records[0]["velocity_vector_world_units_per_second"] = {
        "x": 30,
        "y": 40,
        "z": 5000,
    }
    write_player_states(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any(
        "velocidad horizontal disponible debe ser finita" in error for error in errors
    )
    assert any("velocidad vertical no es plausible" in error for error in errors)


def test_velocity_scalar_must_match_vector_and_source_interval(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    records[0].update(
        {
            "horizontal_velocity_world_units_per_second": 4.0,
            "velocity_source": "position_delta",
            "velocity_measurement_window_ticks": 9,
        }
    )
    write_player_states(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("no coincide con hypot" in error for error in errors)
    assert any("requiere intervalo entre 1 y 8" in error for error in errors)


def test_unavailable_state_cannot_fabricate_knife_as_current_weapon(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    records[1]["active_weapon"] = "Knife"
    write_player_states(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("active_weapon debe ser null" in error for error in errors)


def test_last_observed_weapon_requires_coherent_pair_and_past_tick(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    records[1]["last_observed_active_weapon_tick"] = 100
    write_player_states(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any(
        "tick del ultimo arma observada no puede ser futuro" in error
        for error in errors
    )


def test_repeated_position_changes_cannot_have_only_zero_velocity(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    template = records[1]
    records[0].update(
        {
            "horizontal_velocity_world_units_per_second": 0.0,
            "velocity_vector_world_units_per_second": {"x": 0.0, "y": 0.0, "z": 0.0},
            "velocity_source": "position_delta",
            "velocity_measurement_window_ticks": 1,
        }
    )
    for tick, x in ((98, 14), (106, 24)):
        state = dict(template)
        state.update(
            {
                "state_id": f"{MATCH_ID}:state:001:{tick:09d}:{PLAYER_TWO}",
                "tick": tick,
                "position": {"x": x, "y": 5, "z": 6},
                "horizontal_velocity_world_units_per_second": 0.0,
                "velocity_vector_world_units_per_second": {
                    "x": 0.0,
                    "y": 0.0,
                    "z": 0.0,
                },
                "velocity_source": "position_delta",
                "velocity_measurement_window_ticks": 8,
            }
        )
        records.append(state)
    records.sort(key=lambda record: (record["tick"], record["player_id"]))
    write_player_states(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("posiciones cambian repetidamente" in error for error in errors)


def test_factual_weapon_is_independent_from_observed_active_weapon(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_engagements(match_dir)
    payload["engagements"][0]["exchanges"][0]["weapon"] = "Desert Eagle"
    write_engagements(match_dir, path, payload)

    assert validator.validate_match_export(match_dir) == []


def test_engagement_exchange_must_project_exact_atomic_hurt(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_engagements(match_dir)
    payload["engagements"][0]["exchanges"][0]["actor_player_id"] = PLAYER_TWO
    write_engagements(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("no proyecta player_hurt exacto" in error for error in errors)


def test_engagement_exchange_requires_complete_atomic_closure(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_engagements(match_dir)
    payload["engagements"][0]["exchanges"][0]["source_event_ids"] = []
    write_engagements(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("closure incompleto" in error for error in errors)


def test_engagement_participants_reconcile_with_exchange_union(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_engagements(match_dir)
    payload["engagements"][0]["participants"].pop()
    write_engagements(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("participantes no reconcilian" in error for error in errors)


def test_engagement_first_damage_is_not_rewritten_from_outcome(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_engagements(match_dir)
    payload["engagements"][0]["first_damage_dealer"]["player_id"] = PLAYER_TWO
    write_engagements(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("first_damage_dealer no coincide" in error for error in errors)


def test_engagement_causal_state_cannot_be_future(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_engagements(match_dir)
    state = payload["engagements"][0]["causal_context"]["participant_states"][0]
    state["availability_tick"] = 101
    write_engagements(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("estado ausente o futuro" in error for error in errors)


def test_engagement_causal_context_accepts_only_participants_known_at_t0() -> None:
    errors: list[str] = []
    context = {
        "t0_tick": 100,
        "t0_sequence_in_tick": 1,
        "participant_states": [
            {
                "player_id": PLAYER_ONE,
                "state_id": None,
                "availability_tick": None,
                "status": "unavailable",
                "movement_classification": None,
                "horizontal_velocity_world_units_per_second": None,
                "active_weapon": None,
            }
        ],
        "initial_distance_world_units": None,
        "initial_distance_status": "unavailable",
    }

    validator.validate_engagement_causal_context(
        context,
        {PLAYER_ONE},
        {},
        100,
        1,
        "engagement",
        errors,
    )

    assert not any("participantes causales" in error for error in errors)


def test_unavailable_engagement_velocity_cannot_fabricate_classification(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_engagements(match_dir)
    state = payload["engagements"][0]["causal_context"]["participant_states"][1]
    state["movement_classification"] = "hold"
    write_engagements(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("velocidad no disponible fabrica clasificacion" in error for error in errors)


def test_engagement_active_weapon_must_match_causal_state(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_engagements(match_dir)
    state = payload["engagements"][0]["causal_context"]["participant_states"][0]
    state["active_weapon"] = "AWP"
    write_engagements(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("proyeccion no coincide con player_state@3" in error for error in errors)


def test_trade_window_config_is_millisecond_derived(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative_path = "derived/trades.json"
    path = match_dir / "canonical" / relative_path
    payload = json.loads(path.read_text(encoding="utf-8"))
    payload["config"]["trade_window_ticks"] = 319
    write_json(path, payload)
    refresh_descriptor(match_dir, relative_path)

    errors = validator.validate_match_export(match_dir)

    assert any("config no cumple trade_response@2" in error for error in errors)


def test_objective_event_enums_and_lifecycle_projection_are_closed(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_objective_events(match_dir)
    records[0]["event_type"] = "carrier_snapshot"
    records[1].update(
        {
            "source": "native_snapshot",
            "position_status": "missing",
            "state_after": "unknown",
            "phase_after": "active",
            "site": "C",
        }
    )
    write_objective_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("event_type no es valido" in error for error in errors)
    assert any("source no es valido" in error for error in errors)
    assert any("position_status no es valido" in error for error in errors)
    assert any("state_after no es valido" in error for error in errors)
    assert any("phase_after no es valido" in error for error in errors)
    assert any("requiere site A o B" in error for error in errors)


def test_objective_event_preserves_unavailable_actor_without_dropping_fact() -> None:
    payloads = build_payloads()
    round_record = payloads["core/rounds.json"][2]["rounds"][0]
    event = dict(payloads["events/objective_events.jsonl"][2][0])
    event["actor_player_id"] = None
    event["actor_side"] = None
    errors: list[str] = []

    validator.validate_objective_event_fields(
        event,
        1,
        {1: round_record},
        {1: {PLAYER_ONE: "t", PLAYER_TWO: "ct"}},
        MATCH_ID,
        errors,
    )

    assert errors == []


def test_objective_source_entity_id_may_change_within_round(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_objective_events(match_dir)
    records[3]["bomb_entity_id"] = 78
    records[4]["bomb_entity_id"] = 78
    write_objective_events(match_dir, path, records)

    assert validator.validate_match_export(match_dir) == []


def test_objective_attempt_pair_requires_exact_duration_actor_site_and_kit(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_objective_events(match_dir)
    records[2]["action_duration_ms"] = 62
    records[2]["site"] = "B"
    records[4]["actor_player_id"] = PLAYER_ONE
    records[4]["has_defuse_kit"] = False
    records[3]["attempt_id"] = f"{MATCH_ID}:objective-attempt:001:defuse:003"
    records[4]["attempt_id"] = f"{MATCH_ID}:objective-attempt:001:defuse:003"
    write_objective_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("action_duration_ms no coincide" in error for error in errors)
    assert any("site cambia dentro del attempt_id" in error for error in errors)
    assert any(
        "actor_player_id cambia dentro del attempt_id" in error for error in errors
    )
    assert any(
        "has_defuse_kit cambia dentro del attempt_id" in error for error in errors
    )
    assert any(
        "actor_side no coincide con side_assignments" in error for error in errors
    )
    assert any("ordinals defuse no son contiguos" in error for error in errors)


def test_sparse_objective_attempts_preserve_observation_provenance() -> None:
    payloads = build_payloads()
    rounds = payloads["core/rounds.json"][2]["rounds"]
    events = payloads["events/objective_events.jsonl"][2]
    player_sides = {1: {PLAYER_ONE: "t", PLAYER_TWO: "ct"}}

    start_only_errors: list[str] = []
    validator.validate_objective_state_machine(
        1, [events[0], events[1]], 64.0, start_only_errors
    )

    terminal_without_start = dict(events[2])
    terminal_without_start["attempt_start_observed"] = False
    terminal_without_start["action_duration_ms"] = None
    sparse_errors: list[str] = []
    validator.validate_objective_event_fields(
        terminal_without_start,
        1,
        {1: rounds[0]},
        player_sides,
        MATCH_ID,
        sparse_errors,
    )
    validator.validate_objective_state_machine(
        1, [events[0], terminal_without_start], 64.0, sparse_errors
    )

    assert start_only_errors == []
    assert sparse_errors == []


def test_bundle_accepts_terminal_without_observed_attempt_start(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    events_path, events = read_objective_events(match_dir)
    events.pop(1)
    events[1].update(
        {
            "event_id": f"{MATCH_ID}:objective:001:104:002",
            "attempt_start_observed": False,
            "action_duration_ms": None,
        }
    )
    events[2]["event_id"] = f"{MATCH_ID}:objective:001:110:003"
    events[3]["event_id"] = f"{MATCH_ID}:objective:001:120:004"
    write_objective_events(match_dir, events_path, events)

    rounds_path, rounds_payload = read_rounds(match_dir)
    objective = rounds_payload["rounds"][0]["objective"]
    objective["plant_event_id"] = events[1]["event_id"]
    objective["resolution_event_id"] = events[3]["event_id"]
    objective["plant_attempts"] = 0
    write_rounds(match_dir, rounds_path, rounds_payload)

    assert validator.validate_match_export(match_dir) == []


def test_objective_attempt_provenance_contradictions_are_rejected(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_objective_events(match_dir)
    records[1]["attempt_id"] = f"{MATCH_ID}:objective-r001-plant-000001"
    records[2].update(
        {
            "attempt_id": f"{MATCH_ID}:objective-r001-plant-000001",
            "attempt_outcome": "in_progress",
            "attempt_start_observed": False,
        }
    )
    write_objective_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("attempt_id no sigue el formato canonico" in error for error in errors)
    assert any("requiere attempt_outcome=completed" in error for error in errors)
    assert any(
        "sin start observado requiere action_duration_ms=null" in error
        for error in errors
    )
    assert any("contradice un start observado" in error for error in errors)


def test_round_objective_summary_and_win_reason_must_match_ledger(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_rounds(match_dir)
    round_record = payload["rounds"][0]
    round_record["win_reason"] = "target_bombed"
    round_record["bomb_tick"] = 105
    round_record["objective"].update(
        {
            "outcome": "exploded",
            "resolution_event_id": "unknown-event",
            "plant_attempts": 2,
        }
    )
    write_rounds(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any(
        "win_reason no coincide con raw_win_reason_code" in error for error in errors
    )
    assert any("plant_attempts no coincide con el ledger" in error for error in errors)
    assert any("resolution_event_id no coincide" in error for error in errors)
    assert any("outcome no coincide con el ledger" in error for error in errors)
    assert any("bomb_tick legacy no coincide" in error for error in errors)


def test_bomb_explosion_position_must_remain_near_the_plant() -> None:
    payloads = build_payloads()
    round_record = payloads["core/rounds.json"][2]["rounds"][0]
    events = payloads["events/objective_events.jsonl"][2]
    plant = next(event for event in events if event["event_type"] == "bomb_plant")
    defuse = next(event for event in events if event["event_type"] == "bomb_defuse")
    explosion = dict(defuse)
    explosion.update(
        {
            "event_type": "bomb_explode",
            "actor_player_id": plant["actor_player_id"],
            "position": {
                **plant["position"],
                "x": plant["position"]["x"] + 32.0,
            },
        }
    )
    objective_events = [event if event is not defuse else explosion for event in events]
    errors: list[str] = []

    validator.validate_round_objective_summary(round_record, objective_events, errors)

    assert any(
        "bomb_explode no conserva la posicion plantada" in error for error in errors
    )


def test_objective_state_machine_allows_aborted_plant_at_another_site() -> None:
    actor = "steam:76561198000000001"
    records = [
        {
            "event_type": "bomb_plant_start",
            "attempt_id": "match:objective-r001-plant:001",
            "tick": 10,
            "actor_player_id": actor,
            "site": "B",
            "state_after": "planting",
        },
        {
            "event_type": "bomb_plant_abort",
            "attempt_id": "match:objective-r001-plant:001",
            "tick": 20,
            "actor_player_id": actor,
            "site": "B",
            "attempt_start_observed": True,
            "action_duration_ms": 156,
            "state_after": "carried",
        },
        {
            "event_type": "bomb_plant_start",
            "attempt_id": "match:objective-r001-plant:002",
            "tick": 30,
            "actor_player_id": actor,
            "site": "A",
            "state_after": "planting",
        },
        {
            "event_type": "bomb_plant",
            "attempt_id": "match:objective-r001-plant:002",
            "tick": 40,
            "actor_player_id": actor,
            "site": "A",
            "attempt_start_observed": True,
            "action_duration_ms": 156,
            "state_after": "planted",
        },
    ]
    errors: list[str] = []

    validator.validate_objective_state_machine(1, records, 64.0, errors)

    assert errors == []


def test_objective_ticks_sequence_site_and_actor_side_are_validated(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_objective_events(match_dir)
    records[0]["tick"] = 70
    records[1]["site"] = "C"
    records[3]["sequence_in_tick"] = 2
    records[4]["actor_player_id"] = PLAYER_ONE
    write_objective_events(match_dir, path, records)
    rounds_path, rounds_payload = read_rounds(match_dir)
    rounds_payload["rounds"][0]["end_tick"] = 79
    write_rounds(match_dir, rounds_path, rounds_payload)

    errors = validator.validate_match_export(match_dir)

    assert any("tick anterior al inicio de ronda" in error for error in errors)
    assert any("requiere site A o B" in error for error in errors)
    assert any("sequence_in_tick no es contiguo" in error for error in errors)
    assert any(
        "actor_side no coincide con side_assignments" in error for error in errors
    )
    assert any(
        "end_tick no puede ser anterior a start_tick" in error for error in errors
    )


def test_player_objective_phase_roles_and_explicit_clocks_are_validated(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_player_states(match_dir)
    records[0]["objective_phase"] = "planted"
    records[1]["is_defusing"] = True
    records[1]["round_time_remaining_ms"] = 99_999
    records[1]["round_clock_remaining_ms"] = 99_999
    write_player_states(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("fase planted requiere reloj de ronda null" in error for error in errors)
    assert any(
        "no puede haber portador C4 en fase planted" in error for error in errors
    )
    assert any(
        "legacy debe coincidir con phase_time_remaining_ms" in error for error in errors
    )
    assert any(
        "is_defusing requiere objective_phase=defusing" in error for error in errors
    )
    assert any("objective_phase no coincide con el ledger" in error for error in errors)


def test_dead_defuser_before_abort_callback_has_no_active_role() -> None:
    events_by_round = {
        1: [
            {
                "tick": 10,
                "event_type": "bomb_plant",
                "state_after": "planted",
                "actor_player_id": PLAYER_ONE,
            },
            {
                "tick": 20,
                "event_type": "bomb_defuse_start",
                "state_after": "defusing",
                "actor_player_id": PLAYER_TWO,
            },
            {
                "tick": 31,
                "event_type": "bomb_defuse_abort",
                "state_after": "planted",
                "actor_player_id": PLAYER_TWO,
            },
        ]
    }
    player_states = [
        {
            "round_number": 1,
            "tick": 30,
            "player_id": PLAYER_TWO,
            "is_alive": False,
            "has_c4": False,
            "is_planting": False,
            "is_defusing": False,
            "objective_phase": "defusing",
        }
    ]
    errors: list[str] = []

    validator.validate_states_against_objective_ledger(
        events_by_round, player_states, errors
    )

    assert errors == []


def test_live_defuser_without_active_role_is_rejected() -> None:
    events_by_round = {
        1: [
            {
                "tick": 10,
                "event_type": "bomb_plant",
                "state_after": "planted",
                "actor_player_id": PLAYER_ONE,
            },
            {
                "tick": 20,
                "event_type": "bomb_defuse_start",
                "state_after": "defusing",
                "actor_player_id": PLAYER_TWO,
            },
        ]
    }
    player_states = [
        {
            "round_number": 1,
            "tick": 30,
            "player_id": PLAYER_TWO,
            "is_alive": True,
            "has_c4": False,
            "is_planting": False,
            "is_defusing": False,
            "objective_phase": "defusing",
        }
    ]
    errors: list[str] = []

    validator.validate_states_against_objective_ledger(
        events_by_round, player_states, errors
    )

    assert errors == [
        "canonical/states/player_states: ronda 1, tick 30: "
        "is_defusing no coincide con el ledger"
    ]


def test_replay_carrier_and_kit_must_match_player_state_at_exact_tick() -> None:
    states = build_payloads()["states/player_states/round_001.jsonl"][2]
    replay_rounds = [
        {
            "round": {
                "round": 1,
                "frames": [
                    {
                        "tick": 90,
                        "players": [
                            {"steam_id": "1", "has_c4": False},
                            {"steam_id": "2", "has_defuse_kit": False},
                        ],
                    }
                ],
            }
        }
    ]
    errors: list[str] = []

    validator.validate_replay_objective_consistency(states, replay_rounds, errors)

    assert any("has_c4 discrepa de player_state" in error for error in errors)
    assert any("has_defuse_kit discrepa de player_state" in error for error in errors)


def test_replay_objective_markers_are_an_exact_projection_of_the_ledger() -> None:
    objective_events = build_payloads()["events/objective_events.jsonl"][2]
    markers = []
    for event in objective_events:
        if event["event_type"] not in validator.REPLAY_OBJECTIVE_EVENT_TYPES:
            continue
        position = event["position"]
        markers.append(
            {
                "type": event["event_type"],
                "tick": event["tick"],
                "actor_id": event["actor_player_id"].split(":", 1)[1],
                "site": event["site"],
                "x": position["x"],
                "y": position["y"],
            }
        )
    replay_rounds = [{"round": {"round": 1, "events": markers}}]
    errors: list[str] = []

    validator.validate_replay_objective_markers(objective_events, replay_rounds, errors)

    assert errors == []


def test_replay_objective_markers_reject_missing_and_post_resolution_events() -> None:
    objective_events = build_payloads()["events/objective_events.jsonl"][2]
    plant = next(
        event for event in objective_events if event["event_type"] == "bomb_plant"
    )
    position = plant["position"]
    extra_marker = {
        "type": "bomb_explode",
        "tick": plant["tick"] + 1,
        "site": plant["site"],
        "x": position["x"],
        "y": position["y"],
    }
    replay_rounds = [{"round": {"round": 1, "events": [extra_marker]}}]
    errors: list[str] = []

    validator.validate_replay_objective_markers(objective_events, replay_rounds, errors)

    assert any("faltan markers del ledger" in error for error in errors)
    assert any("markers extra o post-resolucion" in error for error in errors)


def test_replay_utility_marker_is_exact_projection_of_canonical_event(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_replay_round(match_dir)
    marker = next(
        event
        for event in payload["round"]["events"]
        if event["type"] == "utility_detonate"
    )
    marker["damage"] = 1
    marker["affected_player_ids"] = ["2", "3"]
    marker["position_source"] = "unavailable"
    write_replay_round(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("damage no coincide" in error for error in errors)
    assert any("affected_player_ids no coincide" in error for error in errors)
    assert any("position_source no coincide" in error for error in errors)
    assert any(
        "utility_replay_projection_mismatches no coincide" in error for error in errors
    )


def test_replay_utility_marker_rejects_missing_duplicate_and_provenance(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_replay_round(match_dir)
    marker = next(
        event
        for event in payload["round"]["events"]
        if event["type"] == "utility_detonate"
    )
    marker.pop("duration_source")
    payload["round"]["events"].append(deepcopy(marker))
    write_replay_round(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("duration_source requerido" in error for error in errors)
    assert any("marker duplicado" in error for error in errors)

    payload["round"]["events"] = [
        event
        for event in payload["round"]["events"]
        if event["type"] != "utility_detonate"
    ]
    write_replay_round(match_dir, path, payload)
    errors = validator.validate_match_export(match_dir)
    assert any("falta marker" in error for error in errors)


def test_replay_bomb_state_rejects_stale_carrier_and_dead_holder() -> None:
    players = [
        {"steam_id": "1", "alive": True, "has_c4": True},
        {"steam_id": "2", "alive": True, "has_c4": False},
    ]
    bomb = {
        "state": "carried",
        "objective_phase": "preplant",
        "is_planted_now": False,
        "was_planted_this_round": False,
        "x": 10,
        "y": 20,
        "carrier_id": "1",
        "defuser_id": None,
        "position_status": "observed",
        "position_source": "demoinfocs_native_snapshot",
    }
    errors: list[str] = []

    validator.validate_replay_bomb_state(bomb, players, 1, 90, errors)

    assert errors == []

    bomb["state"] = "planted"
    bomb["objective_phase"] = "planted"
    bomb["is_planted_now"] = True
    bomb["was_planted_this_round"] = True
    bomb["site"] = "A"
    bomb["plant_tick"] = 90
    players[0]["alive"] = False
    errors = []

    validator.validate_replay_bomb_state(bomb, players, 1, 90, errors)

    assert any("jugador muerto conserva has_c4" in error for error in errors)
    assert any("estado sin carrier conserva C4" in error for error in errors)


def test_utility_v2_identity_is_stable_and_one_to_one_with_entity(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    duplicate = deepcopy(records[0])
    duplicate["source_throw_id"] = "r1-u0002"
    duplicate["event_id"] = f"{MATCH_ID}:utility:r1-u0002"
    duplicate["sequence_in_round"] = 2
    duplicate["launch"]["tick"] = 111
    duplicate["lifecycle"]["detonation"]["tick"] = 121
    write_utility_events(match_dir, path, [records[0], duplicate])

    errors = validator.validate_match_export(match_dir)

    assert any(
        "entidad estable corresponde a mas de un throw" in error for error in errors
    )


def test_utility_launch_does_not_treat_zero_as_missing(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    launch = records[0]["launch"]
    launch["position"] = {
        "value": {"x": 0.0, "y": 0.0, "z": 0.0},
        "status": "unavailable",
        "source": "unavailable",
    }
    launch["thrower_velocity"] = {
        "vector_world_units_per_second": {"x": 0.0, "y": 0.0, "z": 0.0},
        "horizontal_world_units_per_second": 0.0,
        "observed_tick": 0,
        "measurement_window_ticks": 0,
        "status": "unavailable",
        "source": "insufficient_history",
    }
    write_utility_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("value debe ser null" in error for error in errors)
    assert any("valores deben ser null" in error for error in errors)


def test_utility_launch_allows_explicitly_unavailable_tick_and_snapshot() -> None:
    launch = deepcopy(build_payloads()["events/utility_events.jsonl"][2][0]["launch"])
    launch.update(
        {"tick": None, "tick_status": "unavailable", "tick_source": "unavailable"}
    )
    launch["position"] = {
        "value": None,
        "status": "unavailable",
        "source": "unavailable",
    }
    launch["view"] = {
        "yaw_deg": None,
        "pitch_deg": None,
        "vector": None,
        "status": "unavailable",
        "source": "unavailable",
    }
    for field in ("thrower_velocity", "projectile_initial_velocity"):
        launch[field] = {
            "vector_world_units_per_second": None,
            "horizontal_world_units_per_second": None,
            "observed_tick": None,
            "measurement_window_ticks": None,
            "status": "unavailable",
            "source": "unavailable",
        }
    launch["stance"] = {"value": None, "status": "unavailable", "source": "unavailable"}
    launch["area"] = {"value": None, "status": "unavailable", "source": "unavailable"}
    errors: list[str] = []

    tick = validator.validate_utility_launch(
        launch, {"start_tick": 80, "end_tick": 140}, "utility.launch", errors
    )

    assert tick is None
    assert errors == []


def test_utility_provenance_is_specific_to_each_observation(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    event = records[0]
    event["launch"]["position"]["source"] = "player_view"
    event["launch"]["thrower_velocity"]["source"] = "projectile_velocity"
    event["launch"]["stance"]["source"] = "map_callout"
    event["lifecycle"]["detonation"]["source"] = "he_explode"
    write_utility_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any(".launch.position: source no acredita" in error for error in errors)
    assert any(
        ".launch.thrower_velocity: source no acredita" in error for error in errors
    )
    assert any(".launch.stance: source no acredita" in error for error in errors)
    assert any(
        ".lifecycle.detonation: source no corresponde" in error for error in errors
    )


def test_utility_source_entity_status_and_source_are_coherent(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    records[0]["source_entity_status"] = "unavailable"
    write_utility_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("source_entity debe ser null" in error for error in errors)
    assert any(
        "source_entity no observado debe usar source unavailable" in error
        for error in errors
    )


def test_utility_contract_rejects_runtime_identity_fields(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    records[0]["runtime_entity_id"] = 77
    records[0]["launch"]["position"]["unique_id"] = 999
    write_utility_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("campos no permitidos runtime_entity_id" in error for error in errors)
    assert any("campos no permitidos unique_id" in error for error in errors)


def test_utility_trajectory_enforces_causal_order_and_bounce_count(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    trajectory = records[0]["trajectory"]
    trajectory["samples"][0]["tick"] = 121
    trajectory["bounce_count"] = 2
    write_utility_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("samples no tiene orden total determinista" in error for error in errors)
    assert any(
        "sample de vuelo no es anterior al terminal" in error for error in errors
    )
    assert any("bounce_count no coincide con bounces" in error for error in errors)


def test_utility_trajectory_frame_at_detonation_tick_is_not_in_flight() -> None:
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    trajectory["samples"][-1]["tick"] = 120
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        120,
        140,
        (50.0, 60.0, 70.0),
        "utility.trajectory",
        errors,
    )

    assert any(
        "sample de vuelo no es anterior al terminal" in error for error in errors
    )


def test_utility_trajectory_destroy_sample_matches_lifecycle_position() -> None:
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    trajectory["samples"].append(
        {
            "tick": 120,
            "position": {"x": 16.0, "y": 25.0, "z": 30.0},
            "source": "projectile_destroy",
        }
    )
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        None,
        120,
        (15.0, 25.0, 30.0),
        "utility.trajectory",
        errors,
    )

    assert any("posicion destroy no reconcilia" in error for error in errors)


def test_utility_trajectory_accepts_late_destroy_after_smoke_terminal() -> None:
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 200},
        120,
        180,
        (50.0, 60.0, 70.0),
        "utility.trajectory",
        errors,
    )

    assert errors == []

    trajectory["samples"].append(
        {
            "tick": 180,
            "position": {"x": 50.0, "y": 60.0, "z": 70.0},
            "source": "projectile_destroy",
        }
    )
    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 200},
        120,
        180,
        (50.0, 60.0, 70.0),
        "utility.trajectory",
        errors,
    )
    assert any("no aplica tras detonation/effect_start" in error for error in errors)


def test_utility_trajectory_marks_incomplete_flight_as_partial() -> None:
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    trajectory.update({"status": "partial", "source": "projectile_frames"})
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        None,
        None,
        None,
        "utility.trajectory",
        errors,
    )

    assert errors == []

    trajectory["status"] = "observed"
    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        None,
        None,
        None,
        "utility.trajectory",
        errors,
    )
    assert any("requiere destroy terminal" in error for error in errors)


def test_utility_trajectory_frames_are_partial_when_destroy_position_is_missing() -> (
    None
):
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    trajectory.update({"status": "partial", "source": "projectile_frames"})
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        None,
        120,
        None,
        "utility.trajectory",
        errors,
    )

    assert errors == []


def test_utility_trajectory_accepts_destroy_only_as_partial() -> None:
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    trajectory.update(
        {
            "samples": [
                {
                    "tick": 120,
                    "position": {"x": 15.0, "y": 25.0, "z": 30.0},
                    "source": "projectile_destroy",
                }
            ],
            "status": "partial",
            "source": "projectile_destroy",
        }
    )
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        None,
        120,
        (15.0, 25.0, 30.0),
        "utility.trajectory",
        errors,
    )

    assert errors == []


def test_utility_trajectory_unavailable_cannot_drop_exact_destroy_sample() -> None:
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    trajectory.update(
        {
            "samples": [],
            "status": "unavailable",
            "source": "unavailable",
        }
    )
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        None,
        120,
        (15.0, 25.0, 30.0),
        "utility.trajectory",
        errors,
    )

    assert any("omite destroy sample observado" in error for error in errors)


def test_utility_trajectory_bounce_must_precede_flight_terminal() -> None:
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    trajectory["bounces"][0]["tick"] = 120
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        120,
        140,
        (50.0, 60.0, 70.0),
        "utility.trajectory",
        errors,
    )

    assert any("tick no es anterior al terminal" in error for error in errors)


def test_utility_bounce_preserves_callback_when_position_is_unavailable(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    bounce = records[0]["trajectory"]["bounces"][0]
    bounce["position"] = None
    bounce["position_status"] = "unavailable"
    write_utility_events(match_dir, path, records)

    assert validator.validate_match_export(match_dir) == []


def test_utility_zero_bounces_cannot_encode_missing_observation() -> None:
    trajectory = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["trajectory"]
    )
    trajectory.update({"bounce_count": 0, "bounces": []})
    errors: list[str] = []

    validator.validate_utility_trajectory(
        trajectory,
        110,
        {"start_tick": 80, "end_tick": 140},
        120,
        120,
        (15.0, 25.0, 30.0),
        "utility.trajectory",
        errors,
    )

    assert any("bounce_count observado debe ser positivo" in error for error in errors)


def test_utility_sparse_lifecycle_keeps_exact_tick_without_position() -> None:
    lifecycle = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["lifecycle"]
    )
    lifecycle["detonation"]["position"] = None
    lifecycle["detonation"]["position_status"] = "unavailable"
    errors: list[str] = []

    validator.validate_utility_lifecycle(
        lifecycle,
        "flashbang",
        110,
        {"start_tick": 80, "end_tick": 140},
        64.0,
        "utility.lifecycle",
        errors,
    )

    assert errors == []


def test_utility_persistent_lifecycle_accepts_sparse_start_but_not_detonated() -> None:
    lifecycle = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["lifecycle"]
    )
    lifecycle["status"] = "effect_expired"
    lifecycle["detonation"]["source"] = "smoke_start"
    lifecycle["expiration"] = {
        "tick": 130,
        "position": {"x": 15.0, "y": 25.0, "z": 30.0},
        "status": "observed",
        "position_status": "observed",
        "source": "smoke_expired",
    }
    lifecycle["end_reason"] = {
        "value": "expired",
        "status": "observed",
        "source": "expiration_callback",
    }
    errors: list[str] = []

    validator.validate_utility_lifecycle(
        lifecycle,
        "smoke",
        110,
        {"start_tick": 80, "end_tick": 140},
        64.0,
        "utility.lifecycle",
        errors,
    )

    assert not any("requiere effect_start" in error for error in errors)
    assert errors == []

    for utility_type in ("smoke", "decoy", "molotov", "incendiary"):
        broken = deepcopy(lifecycle)
        broken["status"] = "detonated"
        broken_errors: list[str] = []
        validator.validate_utility_lifecycle(
            broken,
            utility_type,
            110,
            {"start_tick": 80, "end_tick": 140},
            64.0,
            "utility.lifecycle",
            broken_errors,
        )
        assert any("persistente no puede terminar" in error for error in broken_errors)


def test_utility_lifecycle_allows_coherent_expiration_after_round_end() -> None:
    lifecycle = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["lifecycle"]
    )
    lifecycle["status"] = "effect_expired"
    lifecycle["detonation"]["source"] = "smoke_start"
    lifecycle["expiration"] = {
        "tick": 150,
        "position": {"x": 15.0, "y": 25.0, "z": 30.0},
        "status": "observed",
        "position_status": "observed",
        "source": "smoke_expired",
    }
    lifecycle["end_reason"] = {
        "value": "expired",
        "status": "observed",
        "source": "expiration_callback",
    }
    errors: list[str] = []

    validator.validate_utility_lifecycle(
        lifecycle,
        "smoke",
        110,
        {"start_tick": 80, "end_tick": 140},
        64.0,
        "utility.lifecycle",
        errors,
    )

    assert errors == []


def test_utility_lifecycle_rejects_terminal_before_effect_start() -> None:
    lifecycle = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["lifecycle"]
    )
    lifecycle["status"] = "effect_expired"
    lifecycle["detonation"] = {
        "tick": None,
        "position": None,
        "status": "unavailable",
        "position_status": "unavailable",
        "source": "unavailable",
    }
    lifecycle["effect_start"] = {
        "tick": 130,
        "position": {"x": 15.0, "y": 25.0, "z": 30.0},
        "status": "observed",
        "position_status": "observed",
        "source": "smoke_start",
    }
    lifecycle["expiration"] = {
        "tick": 120,
        "position": {"x": 15.0, "y": 25.0, "z": 30.0},
        "status": "observed",
        "position_status": "observed",
        "source": "smoke_expired",
    }
    lifecycle["end_reason"] = {
        "value": "expired",
        "status": "observed",
        "source": "expiration_callback",
    }
    errors: list[str] = []

    validator.validate_utility_lifecycle(
        lifecycle,
        "smoke",
        110,
        {"start_tick": 80, "end_tick": 140},
        64.0,
        "utility.lifecycle",
        errors,
    )

    assert any("expiration es anterior a effect_start" in error for error in errors)


def test_utility_non_terminal_moments_remain_inside_round() -> None:
    cases = (
        ("detonation", "flashbang", "flash_explode"),
        ("effect_start", "smoke", "smoke_start"),
    )
    for moment_name, utility_type, source in cases:
        errors: list[str] = []
        validator.validate_utility_moment(
            {
                "tick": 141,
                "position": {"x": 15.0, "y": 25.0, "z": 30.0},
                "status": "observed",
                "position_status": "observed",
                "source": source,
            },
            moment_name,
            utility_type,
            110,
            {"start_tick": 80, "end_tick": 140},
            f"utility.lifecycle.{moment_name}",
            errors,
        )

        assert any("tick posterior al fin de ronda" in error for error in errors)


def test_utility_inferno_lifecycle_only_keeps_type_unknown() -> None:
    lifecycle = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["lifecycle"]
    )
    lifecycle["status"] = "effect_expired"
    lifecycle["detonation"] = {
        "tick": None,
        "position": None,
        "status": "unavailable",
        "position_status": "unavailable",
        "source": "unavailable",
    }
    lifecycle["expiration"] = {
        "tick": 130,
        "position": {"x": 15.0, "y": 25.0, "z": 30.0},
        "status": "observed",
        "position_status": "observed",
        "source": "inferno_expired",
    }
    lifecycle["end_reason"] = {
        "value": "expired",
        "status": "observed",
        "source": "expiration_callback",
    }
    errors: list[str] = []

    validator.validate_utility_lifecycle(
        lifecycle,
        "unknown",
        None,
        {"start_tick": 80, "end_tick": 140},
        64.0,
        "utility.lifecycle",
        errors,
    )

    assert errors == []


def test_utility_unknown_inferno_cannot_use_instant_detonated_status() -> None:
    lifecycle = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["lifecycle"]
    )
    lifecycle["detonation"]["source"] = "inferno_start"
    errors: list[str] = []

    validator.validate_utility_lifecycle(
        lifecycle,
        "unknown",
        None,
        {"start_tick": 80, "end_tick": 140},
        64.0,
        "utility.lifecycle",
        errors,
    )

    assert any("persistente no puede terminar" in error for error in errors)


def test_utility_lifecycle_and_damage_ticks_must_stay_inside_round() -> None:
    lifecycle = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["lifecycle"]
    )
    lifecycle["detonation"]["tick"] = 79
    lifecycle["destroy"]["tick"] = 79
    player = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["affected_players"][0]
    )
    player.update(
        {
            "player_source": "player_hurt",
            "blind_duration": {
                "milliseconds": None,
                "status": "not_applicable",
                "source": "unavailable",
            },
            "blind_correlation": {
                "status": "unavailable",
                "source": "unavailable",
            },
            "damage": 1,
            "armor_damage": 0,
            "is_kill": False,
            "damage_events": [
                {
                    "tick": 79,
                    "health_damage": 1,
                    "armor_damage": 0,
                    "is_kill": False,
                    "source": "player_hurt",
                    "correlation": {
                        "status": "inferred",
                        "source": "thrower_type_position_tick",
                    },
                }
            ],
        }
    )
    errors: list[str] = []

    validator.validate_utility_lifecycle(
        lifecycle,
        "flashbang",
        None,
        {"start_tick": 80, "end_tick": 140},
        64.0,
        "utility.lifecycle",
        errors,
    )
    validator.validate_utility_affected_player(
        player,
        "he",
        PLAYER_ONE,
        "t",
        {PLAYER_TWO: "ct"},
        None,
        {"start_tick": 80, "end_tick": 140},
        "utility.affected",
        errors,
    )

    assert sum("tick anterior al inicio de ronda" in error for error in errors) >= 3


def test_utility_unknown_type_requires_explicit_unavailable_status(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    records[0].update(
        {
            "utility_type": "unknown",
            "utility_type_status": "observed",
            "utility_type_source": "unavailable",
        }
    )
    write_utility_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any(
        "utility_type unknown debe declarar unavailable" in error for error in errors
    )


def test_utility_extinguish_requires_complete_nearby_smoke_attribution() -> None:
    lifecycle = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["lifecycle"]
    )
    lifecycle["status"] = "effect_expired"
    lifecycle["effect_start"] = {
        "tick": 115,
        "position": {"x": 15.0, "y": 25.0, "z": 30.0},
        "status": "observed",
        "position_status": "observed",
        "source": "inferno_start",
    }
    lifecycle["expiration"] = {
        "tick": 137,
        "position": {"x": 15.0, "y": 25.0, "z": 30.0},
        "status": "observed",
        "position_status": "observed",
        "source": "inferno_expired",
    }
    lifecycle["extinguish"] = {
        "tick": 120,
        "position": {"x": 15.0, "y": 25.0, "z": 30.0},
        "status": "observed",
        "position_status": "observed",
        "source": "spatial_smoke_overlap",
    }
    lifecycle["duration"] = {
        "milliseconds": 343.75,
        "status": "observed",
        "source": "callback_ticks",
    }
    lifecycle["end_reason"] = {
        "value": "smoke_extinguished",
        "status": "observed",
        "source": "spatial_smoke_overlap",
    }
    lifecycle["extinguished_by_throw_id"] = {
        "value": "r1-u0002",
        "status": "observed",
        "source": "spatial_smoke_overlap",
    }
    lifecycle["extinguish_correlation"] = {
        "status": "inferred",
        "source": "spatial_smoke_overlap",
    }
    errors: list[str] = []

    validator.validate_utility_lifecycle(
        lifecycle,
        "molotov",
        110,
        {"start_tick": 80, "end_tick": 140},
        64.0,
        "utility.lifecycle",
        errors,
    )

    assert any("no son temporalmente proximas" in error for error in errors)


def test_utility_flash_relation_side_and_source_are_exact(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    victim = records[0]["affected_players"][0]
    victim["side"] = "t"
    victim["relation"] = "teammate"
    victim["is_enemy"] = False
    victim["player_source"] = "player_hurt"
    write_utility_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("side no coincide con side_assignments" in error for error in errors)
    assert any("player_source no coincide" in error for error in errors)


def test_utility_effect_source_does_not_replace_throw_correlation(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, records = read_utility_events(match_dir)
    records[0]["affected_players"][0]["blind_correlation"] = {
        "status": "observed",
        "source": "player_flashed",
    }
    write_utility_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any(".blind_correlation: source no es valido" in error for error in errors)


def test_utility_flash_summary_preserves_missing_duration_as_null() -> None:
    player = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["affected_players"][0]
    )
    player["blind_duration"] = {
        "milliseconds": None,
        "status": "unavailable",
        "source": "unavailable",
    }
    errors: list[str] = []
    affected = validator.validate_utility_affected_player(
        player,
        "flashbang",
        PLAYER_ONE,
        "t",
        {PLAYER_TWO: "ct"},
        110,
        {"start_tick": 80, "end_tick": 140},
        "utility.affected",
        errors,
    )
    assert affected is not None
    summary = {
        "players_total": 1,
        "enemies_flashed": 1,
        "teammates_flashed": 0,
        "self_flashed": 0,
        "unknown_flashed": 0,
        "total_duration_ms": None,
        "enemy_duration_ms": None,
        "teammate_duration_ms": 0.0,
        "self_duration_ms": 0.0,
        "unknown_duration_ms": 0.0,
    }

    validator.validate_utility_flash_summary(
        summary, [affected], "utility.flash_summary", errors
    )

    assert errors == []
    event = deepcopy(build_payloads()["events/utility_events.jsonl"][2][0])
    event["affected_players"][0]["blind_duration"] = player["blind_duration"]
    assert (
        validator.utility_observation_metric_counts([event])[
            "utility_missing_flash_duration_observations"
        ]
        == 1
    )
    summary["enemy_duration_ms"] = 0.0
    validator.validate_utility_flash_summary(
        summary, [affected], "utility.flash_summary", errors
    )
    assert any("enemy_duration_ms debe ser null" in error for error in errors)


def test_utility_missing_thrower_side_is_explicit_coverage_warning(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    utility_path, records = read_utility_events(match_dir)
    records[0]["thrower_side"] = None
    affected = records[0]["affected_players"][0]
    affected.update({"relation": "unknown", "is_enemy": False})
    records[0]["flash_summary"].update(
        {
            "enemies_flashed": 0,
            "unknown_flashed": 1,
            "enemy_duration_ms": 0.0,
            "unknown_duration_ms": 1250.5,
        }
    )
    write_utility_events(match_dir, utility_path, records)
    quality_path, quality_payload = read_quality_report(match_dir)
    report = quality_payload["report"]
    report["status"] = "warning"
    report["utility_missing_actor_observations"] = 1
    report["utility_observation_warnings"] = 1
    next(
        check
        for check in report["checks"]
        if check["name"] == "utility_observation_coverage"
    )["status"] = "warning"
    write_quality_report(match_dir, quality_path, quality_payload)

    assert validator.validate_match_export(match_dir) == []


def test_utility_zero_damage_callback_and_unknown_relation_are_preserved() -> None:
    player = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["affected_players"][0]
    )
    player.update(
        {
            "player_source": "player_hurt",
            "relation": "unknown",
            "is_enemy": False,
            "blind_duration": {
                "milliseconds": None,
                "status": "not_applicable",
                "source": "unavailable",
            },
            "blind_correlation": {
                "status": "unavailable",
                "source": "unavailable",
            },
            "damage": 0,
            "armor_damage": 0,
            "is_kill": False,
            "damage_events": [
                {
                    "tick": 120,
                    "health_damage": 0,
                    "armor_damage": 0,
                    "is_kill": False,
                    "source": "player_hurt",
                    "correlation": {
                        "status": "inferred",
                        "source": "thrower_type_position_tick",
                    },
                }
            ],
        }
    )
    errors: list[str] = []
    affected = validator.validate_utility_affected_player(
        player,
        "he",
        None,
        None,
        {PLAYER_TWO: "ct"},
        110,
        {"start_tick": 80, "end_tick": 140},
        "utility.affected",
        errors,
    )
    assert affected is not None
    summary = {
        "total_damage": 0,
        "enemy_damage": 0,
        "teammate_damage": 0,
        "self_damage": 0,
        "unknown_damage": 0,
        "total_armor_damage": 0,
        "enemy_armor_damage": 0,
        "teammate_armor_damage": 0,
        "self_armor_damage": 0,
        "unknown_armor_damage": 0,
        "enemies_damaged": 0,
        "teammates_damaged": 0,
        "unknown_players_damaged": 0,
        "self_damaged": False,
        "enemy_kills": 0,
        "teammate_kills": 0,
        "self_kills": 0,
        "unknown_kills": 0,
    }
    validator.validate_utility_damage_summary(
        summary, [affected], "utility.damage_summary", errors
    )

    assert errors == []

    summary["unknown_damage"] = 1
    validator.validate_utility_damage_summary(
        summary, [affected], "utility.damage_summary", errors
    )
    assert any("unknown_damage no coincide" in error for error in errors)


def test_utility_damage_summary_keeps_teammate_and_self_kills_separate() -> None:
    base_summary = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["damage_summary"]
    )
    for relation, damage_field, armor_field, damaged_field, kill_field in (
        (
            "teammate",
            "teammate_damage",
            "teammate_armor_damage",
            "teammates_damaged",
            "teammate_kills",
        ),
        (
            "self",
            "self_damage",
            "self_armor_damage",
            "self_damaged",
            "self_kills",
        ),
    ):
        affected = {
            "relation": relation,
            "health_damage": 10,
            "armor_damage": 2,
            "is_kill": True,
            "has_positive_damage": True,
        }
        summary = deepcopy(base_summary)
        summary.update(
            {
                "total_damage": 10,
                "total_armor_damage": 2,
                damage_field: 10,
                armor_field: 2,
                damaged_field: True if relation == "self" else 1,
                kill_field: 1,
            }
        )
        errors: list[str] = []

        validator.validate_utility_damage_summary(
            summary, [affected], "utility.damage_summary", errors
        )

        assert errors == []
        summary[kill_field] = 0
        validator.validate_utility_damage_summary(
            summary, [affected], "utility.damage_summary", errors
        )
        assert any(f"{kill_field} no coincide" in error for error in errors)


def test_utility_callback_can_preserve_effect_when_victim_identity_is_unavailable() -> (
    None
):
    player = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["affected_players"][0]
    )
    player.update(
        {
            "player_id": None,
            "side": None,
            "player_status": "unavailable",
            "player_source": "player_flashed",
            "relation": "unknown",
            "is_enemy": False,
        }
    )
    errors: list[str] = []

    affected = validator.validate_utility_affected_player(
        player,
        "flashbang",
        PLAYER_ONE,
        "t",
        {PLAYER_TWO: "ct"},
        110,
        {"start_tick": 80, "end_tick": 140},
        "utility.affected",
        errors,
    )

    assert affected is not None
    assert affected["relation"] == "unknown"
    assert errors == []


def test_utility_player_source_prefers_hurt_when_effects_are_combined() -> None:
    player = deepcopy(
        build_payloads()["events/utility_events.jsonl"][2][0]["affected_players"][0]
    )
    player.update(
        {
            "player_source": "player_hurt",
            "damage": 1,
            "armor_damage": 0,
            "is_kill": False,
            "damage_events": [
                {
                    "tick": 120,
                    "health_damage": 1,
                    "armor_damage": 0,
                    "is_kill": False,
                    "source": "player_hurt",
                    "correlation": {
                        "status": "inferred",
                        "source": "thrower_type_position_tick",
                    },
                }
            ],
        }
    )
    errors: list[str] = []

    validator.validate_utility_affected_player(
        player,
        "flashbang",
        PLAYER_ONE,
        "t",
        {PLAYER_TWO: "ct"},
        110,
        {"start_tick": 80, "end_tick": 140},
        "utility.affected",
        errors,
    )

    assert not any("player_source no coincide" in error for error in errors)
    assert any("utility_type no puede causar dano" in error for error in errors)


def test_utility_quality_requires_all_gates_and_reconciled_metrics(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_quality_report(match_dir)
    report = payload["report"]
    report["checks"] = [
        check for check in report["checks"] if check["name"] != "utility_determinism"
    ]
    report["utility_contract_violations"] = 1
    write_quality_report(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any(
        "faltan checks utility requeridos: utility_determinism" in error
        for error in errors
    )
    assert any("utility_contract_violations debe ser cero" in error for error in errors)


def test_utility_quality_requires_callback_and_victim_coverage_metrics(
    tmp_path: Path,
) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_quality_report(match_dir)
    report = payload["report"]
    report.pop("utility_callback_accounting_violations")
    report["utility_orphan_callbacks"] = 1
    write_quality_report(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any(
        "faltan metricas utility requeridas: utility_callback_accounting_violations"
        in error
        for error in errors
    )
    assert any(
        "utility_observation_warnings no reconcilia" in error for error in errors
    )
    assert any("utility_observation_coverage no coincide" in error for error in errors)


def test_utility_quality_reconciles_raw_callback_diagnostics(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    path, payload = read_quality_report(match_dir)
    report = payload["report"]
    report["utility_callback_diagnostics"]["throws"]["exact_correlated"] = 0
    write_quality_report(match_dir, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any(".throws: callbacks no reconcilian" in error for error in errors)
    assert any(
        "utility_callback_accounting_violations no reconcilia" in error
        for error in errors
    )


def test_block6_accepts_distinct_native_and_calculated_money(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/economy_players.json"
    path, payload = read_canonical_json(match_dir, relative)
    payload["players"][0]["money"]["after_buy_observed"]["amount"] = 900
    payload["players"][0]["money"]["native_calculated_delta"] = 100
    write_canonical_json(match_dir, relative, path, payload)

    assert validator.validate_match_export(match_dir) == []


def test_block6_accepts_player_without_observable_inventory(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/economy_players.json"
    path, payload = read_canonical_json(match_dir, relative)
    payload["players"][0]["inventory_round_end"] = {
        "status": "not_observed",
        "native_value": None,
        "calculated_value": None,
        "items": [],
    }
    write_canonical_json(match_dir, relative, path, payload)

    assert validator.validate_match_export(match_dir) == []


def test_block6_rejects_ct_as_team_identity(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/economy_players.json"
    path, payload = read_canonical_json(match_dir, relative)
    payload["players"][0]["team_id"] = "CT"
    write_canonical_json(match_dir, relative, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("team_id no es estable" in error for error in errors)


def test_block6_distinguishes_unknown_and_real_zero_prices(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/economy_players.json"
    path, payload = read_canonical_json(match_dir, relative)
    player = payload["players"][0]
    base_item = {
        "observed_item": "c4",
        "purchased_item": "c4",
        "entity_id": None,
        "original_owner_player_id": None,
        "original_owner_status": "not_observed",
    }
    player["transactions"] = [
        {
            "transaction_id": "purchase:unknown",
            "type": "purchase",
            "tick": 10,
            "actor_player_id": player["player_id"],
            "other_player_id": None,
            "amount": None,
            "source": "test",
            "source_event_ids": ["purchase:unknown"],
            "item": {
                **base_item,
                "price": {"amount": None, "status": "unknown", "table_version": "stratai.cs2_prices@1"},
            },
        },
        {
            "transaction_id": "purchase:zero",
            "type": "purchase",
            "tick": 11,
            "actor_player_id": player["player_id"],
            "other_player_id": None,
            "amount": 0,
            "source": "test",
            "source_event_ids": ["purchase:zero"],
            "item": {
                **base_item,
                "price": {"amount": 0, "status": "known_zero", "table_version": "stratai.cs2_prices@1"},
            },
        },
    ]
    write_canonical_json(match_dir, relative, path, payload)

    assert validator.validate_match_export(match_dir) == []

    payload["players"][0]["transactions"][0]["item"]["price"]["amount"] = 0
    write_canonical_json(match_dir, relative, path, payload)
    errors = validator.validate_match_export(match_dir)
    assert any("precio unknown debe permanecer null" in error for error in errors)


def test_block6_rejects_impossible_money_transition(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/economy_players.json"
    path, payload = read_canonical_json(match_dir, relative)
    payload["players"][0]["money"]["round_end_observed"]["amount"] = 16_001
    write_canonical_json(match_dir, relative, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("dinero fuera de 0..16000" in error for error in errors)


def test_block6_rejects_duplicate_reward_ids(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/economy_rounds.json"
    path, payload = read_canonical_json(match_dir, relative)
    reward = {
        "reward_id": "reward:1",
        "type": "win",
        "observed_amount": None,
        "calculated_amount": None,
        "status": "unavailable",
        "source_event_ids": ["round:1"],
    }
    payload["rounds"][0]["rewards"] = [reward, deepcopy(reward)]
    write_canonical_json(match_dir, relative, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("premio duplicado" in error for error in errors)


def test_block6_rejects_purchase_and_pickup_without_player_ids(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/economy_players.json"
    path, payload = read_canonical_json(match_dir, relative)
    player = payload["players"][0]
    item = {
        "observed_item": "ak47",
        "purchased_item": "ak47",
        "entity_id": 7,
        "price": {"amount": 2700, "status": "known", "table_version": "stratai.cs2_prices@1"},
        "original_owner_player_id": PLAYER_TWO,
        "original_owner_status": "observed",
    }
    player["transactions"] = [
        {
            "transaction_id": "purchase:no-actor",
            "type": "purchase",
            "tick": 10,
            "actor_player_id": None,
            "other_player_id": None,
            "source": "test",
            "source_event_ids": ["purchase:no-actor"],
            "item": item,
        },
        {
            "transaction_id": "pickup:no-owner",
            "type": "pickup",
            "tick": 11,
            "actor_player_id": player["player_id"],
            "other_player_id": None,
            "source": "test",
            "source_event_ids": ["pickup:no-owner"],
            "item": item,
        },
    ]
    write_canonical_json(match_dir, relative, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("provenance de jugador ausente" in error for error in errors)
    assert any("pickup atribuido sin owner ID" in error for error in errors)


def test_block6_rejects_warmup_economy_rows(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/economy_players.json"
    path, payload = read_canonical_json(match_dir, relative)
    payload["players"][0]["round_number"] = 0
    write_canonical_json(match_dir, relative, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("fuera de ronda competitiva" in error for error in errors)


def test_block6_reconstructs_missing_clutch_attempt_from_atomic_kill(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    participants_relative = "core/participants.json"
    participants_path, participants = read_canonical_json(
        match_dir, participants_relative
    )
    participants["players"].extend(
        [
            {
                "player_id": "steam:3",
                "steam_id": "3",
                "display_name": "three",
                "team_id": "team_b",
            },
            {
                "player_id": "steam:4",
                "steam_id": "4",
                "display_name": "four",
                "team_id": "team_a",
            },
        ]
    )
    participants["players"].sort(key=lambda player: player["player_id"])
    write_canonical_json(
        match_dir, participants_relative, participants_path, participants
    )
    path, records = read_combat_events(match_dir)
    records[0]["is_kill"] = True
    write_combat_events(match_dir, path, records)

    errors = validator.validate_match_export(match_dir)

    assert any("falta attempt atomico" in error for error in errors)


def test_block6_rejects_clutch_without_attempt(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/clutch_events.json"
    path, payload = read_canonical_json(match_dir, relative)
    payload["clutch_events"] = [
        {
            "clutch_id": "clutch:1",
            "round_id": ROUND_ID,
            "round_number": 1,
            "player_id": PLAYER_ONE,
            "team_id": "team_b",
            "side": "t",
            "enemies_at_start": 1,
            "state": "1v1",
            "attempt": False,
            "result": "lost",
            "start_tick": 100,
            "trigger_event_id": f"{MATCH_ID}:combat:001:100:001",
            "source_event_ids": [f"{MATCH_ID}:combat:001:100:001"],
            "outcome_source": "rounds.winner_team_id",
            "evaluation_status": "evaluated",
        }
    ]
    write_canonical_json(match_dir, relative, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("clutch sin attempt" in error for error in errors)


def test_block6_rejects_scoreboard_and_utility_mismatches(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "derived/player_stats.json"
    path, payload = read_canonical_json(match_dir, relative)
    payload["players"][0]["native_scoreboard"]["kills"] = 9
    payload["players"][0]["derived"]["utility_damage_observed"] = 1
    write_canonical_json(match_dir, relative, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("scoreboard nativo no reconcilia" in error for error in errors)
    assert any("utilidad observada no reconcilia" in error for error in errors)


def test_block6_rejects_invented_dates_and_missing_lineage(tmp_path: Path) -> None:
    match_dir = write_valid_bundle(tmp_path)
    relative = "core/match_metadata.json"
    path, payload = read_canonical_json(match_dir, relative)
    payload["played_at"] = "2026-08-19T12:00:00Z"
    payload["played_at_status"] = "observed"
    payload["played_at_source"] = "processing"
    payload["source"]["source"] = "processing"
    payload["configuration_hashes"].pop("rating_algorithm")
    write_canonical_json(match_dir, relative, path, payload)

    errors = validator.validate_match_export(match_dir)

    assert any("played_at sin fuente fiable" in error for error in errors)
    assert any("hashes de configuracion invalidos" in error for error in errors)
