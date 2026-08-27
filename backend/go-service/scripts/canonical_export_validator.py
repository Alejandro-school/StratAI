#!/usr/bin/env python3
"""Valida estructura, integridad y referencias del export canónico."""

import argparse
import gzip
import hashlib
import json
import math
import re
from collections.abc import Iterable, Mapping
from dataclasses import dataclass
from datetime import datetime
from itertools import pairwise
from pathlib import Path, PurePosixPath

MANIFEST_SCHEMA_ID = "stratai.canonical_manifest@3"
EXPORT_FORMAT_VERSION = "3.8.0"
PARSER_SCHEMA_VERSION = "v16"
QUALITY_SCHEMA_VERSION = 12
REPLAY_SCHEMA_VERSION = 5
VALIDATOR_VERSION = "stratai.canonical_validator@2"
GOLDEN_CORPUS_VERSION = "stratai.golden_demo_corpus@2"
MAX_RETAINED_VALIDATION_ERRORS = 10_000
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
PLAYER_ID_PATTERN = re.compile(r"^steam:[1-9][0-9]*$")
AVAILABLE_VELOCITY_SOURCES = frozenset({"native", "position_delta"})
UNAVAILABLE_VELOCITY_SOURCES = frozenset(
    {
        "insufficient_history",
        "not_applicable",
        "position_delta_rejected",
        "stale_gap",
        "entity_changed",
        "non_monotonic_tick",
    }
)
VELOCITY_SOURCES = AVAILABLE_VELOCITY_SOURCES | UNAVAILABLE_VELOCITY_SOURCES
ACTIVE_WEAPON_STATUSES = frozenset({"observed", "unavailable", "not_applicable"})
AVAILABLE_VELOCITY_OBSERVATIONS = frozenset({"current_tick", "last_alive"})
ACTIVE_WEAPON_OBSERVATIONS = frozenset(
    {"observed_current", "last_observed", "unavailable"}
)
MAX_HORIZONTAL_VELOCITY_UPS = 2_000.0
MAX_VERTICAL_VELOCITY_UPS = 4_000.0
MAX_POSITION_DELTA_INTERVAL_TICKS = 8
ENGAGEMENT_PEEK_VELOCITY_THRESHOLD_UPS = 100.0
ENGAGEMENT_ALGORITHM_VERSION = "engagement_causal@2"
TRADE_ALGORITHM_VERSION = "trade_response@2"
ENGAGEMENT_PAIR_WINDOW_MS = 1_500
ENGAGEMENT_MULTI_TARGET_WINDOW_MS = 750
ENGAGEMENT_MAX_DURATION_MS = 5_000
ENGAGEMENT_PRELUDE_WINDOW_MS = 500
TRADE_WINDOW_MS = 5_000
TRADE_MAX_DISTANCE_WORLD_UNITS = 1_250.0
TRADE_ASSUMED_MOVEMENT_SPEED_UPS = 250.0
TRADE_MAX_FACING_DELTA_DEG = 100.0
TRADE_PHYSICAL_EVIDENCE_REQUIREMENT = (
    "alive+distance+connection_time+physics_mesh_los+orientation"
)
ENGAGEMENT_ROLE_STATUSES = frozenset({"observed", "inferred", "unavailable"})
ENGAGEMENT_OUTCOMES = frozenset({"kill", "disengaged"})
TRADE_EVALUATIONS = frozenset(
    {"completed", "failed", "not_attempted", "not_tradeable", "not_evaluable"}
)
REQUIRED_ENGAGEMENT_QUALITY_CHECKS = frozenset(
    {
        "engagement_event_contract",
        "engagement_atomic_provenance",
        "engagement_participant_reconciliation",
        "engagement_role_consistency",
        "engagement_temporal_consistency",
        "engagement_causal_availability",
        "engagement_trade_reconciliation",
        "engagement_stats_reconciliation",
        "engagement_determinism",
        "engagement_observation_coverage",
    }
)
REQUIRED_BLOCK6_QUALITY_CHECKS = frozenset(
    {
        "economy_team_identity",
        "economy_native_calculated_reconciliation",
        "economy_money_transition",
        "economy_purchase_provenance",
        "economy_price_table_version",
        "stats_scoreboard_reconciliation",
        "stats_utility_reconciliation",
        "clutch_attempt_reconciliation",
        "warmup_contamination",
        "metadata_provenance",
        "metadata_checksum_lineage",
        "economy_determinism",
        "stats_determinism",
        "economy_observation_coverage",
    }
)
HARD_BLOCK6_QUALITY_METRICS = frozenset(
    {
        "economy_team_identity_violations",
        "economy_native_calculated_reconciliation_violations",
        "economy_money_transition_violations",
        "economy_purchase_provenance_violations",
        "economy_price_table_version_violations",
        "stats_scoreboard_reconciliation_mismatches",
        "stats_utility_reconciliation_mismatches",
        "clutch_attempt_reconciliation_mismatches",
        "warmup_contamination_violations",
        "metadata_provenance_violations",
        "metadata_checksum_lineage_violations",
        "economy_determinism_violations",
        "stats_determinism_violations",
    }
)
HARD_ENGAGEMENT_QUALITY_METRICS = frozenset(
    {
        "engagement_event_contract_violations",
        "engagement_atomic_provenance_violations",
        "engagement_participant_reconciliation_mismatches",
        "engagement_role_consistency_violations",
        "engagement_temporal_consistency_violations",
        "engagement_causal_availability_violations",
        "engagement_trade_reconciliation_mismatches",
        "engagement_stats_reconciliation_mismatches",
        "engagement_determinism_violations",
    }
)
REQUIRED_BLOCK7_DOMAINS = frozenset(
    {
        "bundle_manifest_contract",
        "artifact_catalog_integrity",
        "artifact_hash_integrity",
        "artifact_record_count",
        "cross_artifact_references",
        "roster_consistency",
        "round_consistency",
        "objective_consistency",
        "utility_consistency",
        "combat_consistency",
        "engagement_consistency",
        "economy_consistency",
        "player_state_consistency",
        "replay_projection_consistency",
        "causal_availability",
        "future_leakage",
        "schema_version_compatibility",
        "determinism",
        "lineage_completeness",
        "corpus_quality",
    }
)
BLOCK7_DOMAIN_FIELDS = frozenset(
    {
        "name",
        "status",
        "severity",
        "expected",
        "actual",
        "coverage",
        "unavailable_count",
        "inferred_count",
        "warning_details",
        "hard_failure_details",
        "source_artifacts",
        "schema_versions",
    }
)
CAUSAL_LINK_FIELDS = frozenset(
    {"schema_id", "match_id", "decision_id", "round_number", "t0_tick"}
)
DECISION_FEATURE_PROHIBITED_KEYS = frozenset(
    {
        "player_id",
        "steam_id",
        "display_name",
        "name",
        "nickname",
        "winner",
        "winner_player_id",
        "loser_player_ids",
        "outcome",
        "outcome_tick",
        "result",
        "round_result",
        "score",
        "final_score",
        "rating",
        "label",
        "terminal_kill_event_ids",
        "trade_completion_ids",
		"actor_player_id",
		"observed_state_ref",
		"oracle_context",
		"oracle_state",
		"decision_outcomes",
		"outcomes",
    }
)
DECISION_FEATURE_ALLOWED_KEYS = frozenset(
    {
        "schema_id",
        "match_id",
        "decision_id",
        "decision_type",
        "round_number",
        "t0_tick",
        "availability_tick_max",
        "participant_count",
        "observed_participant_states",
        "alive_participant_count",
        "initial_distance_world_units",
        "initial_distance_status",
        "bomb_context_status",
        "economy_context_status",
        "enemies_exposed_count",
        "enemies_exposed_status",
        "round_clock_remaining_ms",
        "bomb_time_remaining_ms",
        "source_state_count",
        "trade_possible",
        "trade_possible_status",
        "nearest_teammate_distance_world_units",
        "nearest_teammate_distance_status",
        "nearest_connection_time_ms",
        "nearest_connection_time_status",
        "any_line_of_sight",
        "line_of_sight_status",
        "minimum_facing_delta_deg",
        "facing_status",
    }
)
MVP_DECISION_TYPES = frozenset(
    {"spacing_or_trade_connection", "peek_hold_or_reposition"}
)
MVP_DECISION_ACTIONS = {
    "peek_hold_or_reposition": frozenset({"peek", "hold", "engage"}),
    "spacing_or_trade_connection": frozenset(
        {"connected", "disconnected", "unclassified_connection"}
    ),
}
OBJECTIVE_EVENT_TYPES = frozenset(
    {
        "bomb_carrier_snapshot",
        "bomb_drop",
        "bomb_pickup",
        "bomb_plant_start",
        "bomb_plant_abort",
        "bomb_plant",
        "bomb_defuse_start",
        "bomb_defuse_abort",
        "bomb_defuse",
        "bomb_explode",
    }
)
REPLAY_OBJECTIVE_EVENT_TYPES = frozenset({"bomb_plant", "bomb_defuse", "bomb_explode"})
OBJECTIVE_EXPLOSION_POSITION_TOLERANCE_UNITS = 16.0
OBJECTIVE_EVENT_SOURCES = frozenset({"demoinfocs_event", "game_state_snapshot"})
OBJECTIVE_POSITION_STATUSES = frozenset({"observed", "unavailable"})
OBJECTIVE_STATES = frozenset(
    {
        "carried",
        "dropped",
        "planting",
        "planted",
        "defusing",
        "defused",
        "exploded",
        "resolved",
    }
)
OBJECTIVE_OUTCOMES = frozenset(
    {"not_planted", "elimination_after_plant", "exploded", "defused", "time_expired"}
)
OBJECTIVE_PHASES = frozenset(
    {"preplant", "planting", "planted", "defusing", "resolved"}
)
ROUND_WIN_REASON_BY_RAW_CODE = {
    0: "still_in_progress",
    1: "target_bombed",
    2: "vip_escaped",
    3: "vip_killed",
    4: "terrorists_escaped",
    5: "ct_stopped_escape",
    6: "terrorists_stopped",
    7: "bomb_defused",
    8: "ct_win",
    9: "terrorists_win",
    10: "draw",
    11: "hostages_rescued",
    12: "target_saved",
    13: "hostages_not_rescued",
    14: "terrorists_not_escaped",
    15: "vip_not_escaped",
    16: "game_start",
    17: "terrorists_surrender",
    18: "ct_surrender",
    19: "terrorists_planted",
    20: "cts_reached_hostage",
}
ROUND_WIN_REASONS = frozenset((*ROUND_WIN_REASON_BY_RAW_CODE.values(), "unknown"))
OBJECTIVE_STATE_AFTER_BY_EVENT = {
    "bomb_carrier_snapshot": "carried",
    "bomb_drop": "dropped",
    "bomb_pickup": "carried",
    "bomb_plant_start": "planting",
    "bomb_plant_abort": "carried",
    "bomb_plant": "planted",
    "bomb_defuse_start": "defusing",
    "bomb_defuse_abort": "planted",
    "bomb_defuse": "defused",
    "bomb_explode": "exploded",
}
OBJECTIVE_ACTOR_SIDE_BY_EVENT = {
    "bomb_carrier_snapshot": "t",
    "bomb_drop": "t",
    "bomb_pickup": "t",
    "bomb_plant_start": "t",
    "bomb_plant_abort": "t",
    "bomb_plant": "t",
    "bomb_defuse_start": "ct",
    "bomb_defuse_abort": "ct",
    "bomb_defuse": "ct",
}
OBJECTIVE_ATTEMPT_KIND_BY_EVENT = {
    "bomb_plant_start": "plant",
    "bomb_plant_abort": "plant",
    "bomb_plant": "plant",
    "bomb_defuse_start": "defuse",
    "bomb_defuse_abort": "defuse",
    "bomb_defuse": "defuse",
}
OBJECTIVE_ATTEMPT_START_EVENTS = frozenset({"bomb_plant_start", "bomb_defuse_start"})
OBJECTIVE_ATTEMPT_END_EVENTS = frozenset(
    {"bomb_plant_abort", "bomb_plant", "bomb_defuse_abort", "bomb_defuse"}
)
OBJECTIVE_ATTEMPT_OUTCOME_BY_EVENT = {
    "bomb_plant_start": "in_progress",
    "bomb_plant_abort": "aborted",
    "bomb_plant": "completed",
    "bomb_defuse_start": "in_progress",
    "bomb_defuse_abort": "aborted",
    "bomb_defuse": "completed",
}
OBJECTIVE_SITE_EVENTS = frozenset(
    {
        "bomb_plant_start",
        "bomb_plant_abort",
        "bomb_plant",
        "bomb_defuse_start",
        "bomb_defuse_abort",
        "bomb_defuse",
        "bomb_explode",
    }
)
OBJECTIVE_PHASE_BY_STATE = {
    "carried": "preplant",
    "dropped": "preplant",
    "planting": "planting",
    "planted": "planted",
    "defusing": "defusing",
    "defused": "resolved",
    "exploded": "resolved",
}
UTILITY_EVENT_TYPES = frozenset({"utility_throw"})
UTILITY_TYPES = frozenset(
    {"flashbang", "smoke", "he", "molotov", "incendiary", "decoy", "unknown"}
)
UTILITY_SOURCES = frozenset(
    {
        "unavailable",
        "weapon_instance",
        "callback_type",
        "projectile_entity",
        "grenade_entity_id",
        "effect_entity_id",
        "projectile_throw",
        "projectile_thrower",
        "projectile_owner",
        "projectile_position",
        "projectile_velocity",
        "player_view",
        "player_state",
        "native",
        "position_delta",
        "insufficient_history",
        "position_delta_rejected",
        "stale_gap",
        "entity_changed",
        "non_monotonic_tick",
        "not_applicable",
        "map_callout",
        "player_last_place",
        "projectile_frames",
        "projectile_bounce",
        "projectile_destroy",
        "flash_explode",
        "he_explode",
        "smoke_start",
        "smoke_expired",
        "inferno_start",
        "inferno_expired",
        "decoy_start",
        "decoy_expired",
        "player_flashed",
        "player_hurt",
        "callback_actor",
        "callback_ticks",
        "expiration_callback",
        "spatial_smoke_overlap",
        "round_boundary",
        "thrower_type_position_tick",
        "type_position_tick",
    }
)
UTILITY_TYPE_SOURCES = frozenset({"weapon_instance", "callback_type", "unavailable"})
UTILITY_THROWER_STATUSES = frozenset({"observed", "unavailable"})
UTILITY_THROWER_SOURCES = frozenset(
    {"projectile_thrower", "projectile_owner", "callback_actor", "unavailable"}
)
UTILITY_OBSERVATION_STATUSES = frozenset({"observed", "unavailable", "not_applicable"})
UTILITY_UNAVAILABLE_SOURCES = frozenset(
    {
        "unavailable",
        "insufficient_history",
        "position_delta_rejected",
        "stale_gap",
        "entity_changed",
        "non_monotonic_tick",
    }
)
UTILITY_TRAJECTORY_STATUSES = frozenset({"observed", "partial", "unavailable"})
UTILITY_STANCES = frozenset(
    {"standing", "walking", "crouching", "crouch_walking", "airborne", "unknown"}
)
UTILITY_LIFECYCLE_STATUSES = frozenset(
    {
        "thrown",
        "detonated",
        "effect_active",
        "effect_expired",
        "destroyed_without_detonation",
        "round_ended_unresolved",
    }
)
UTILITY_END_REASONS = frozenset(
    {"expired", "smoke_extinguished", "destroyed", "round_end", "unavailable"}
)
UTILITY_RELATIONS = frozenset({"self", "teammate", "enemy", "unknown"})
UTILITY_PLAYER_STATUSES = frozenset({"observed", "unavailable"})
UTILITY_PLAYER_SOURCES = frozenset({"player_flashed", "player_hurt", "unavailable"})
UTILITY_AREA_SOURCES = frozenset({"map_callout", "player_last_place"})
UTILITY_THROWER_VELOCITY_SOURCES = frozenset({"native", "position_delta"})
UTILITY_PROJECTILE_VELOCITY_SOURCES = frozenset({"projectile_velocity"})
UTILITY_VELOCITY_UNAVAILABLE_SOURCES = frozenset(
    {
        "unavailable",
        "insufficient_history",
        "position_delta_rejected",
        "stale_gap",
        "entity_changed",
        "non_monotonic_tick",
    }
)
UTILITY_CORRELATION_STATUSES = frozenset({"observed", "inferred", "unavailable"})
UTILITY_CORRELATION_SOURCES = frozenset(
    {
        "projectile_entity",
        "grenade_entity_id",
        "effect_entity_id",
        "thrower_type_position_tick",
        "type_position_tick",
        "unavailable",
    }
)
UTILITY_LIFECYCLE_MOMENT_SOURCES = {
    "detonation": {
        "flashbang": frozenset({"flash_explode", "player_flashed"}),
        "he": frozenset({"he_explode"}),
        "smoke": frozenset({"smoke_start"}),
        "decoy": frozenset({"decoy_start"}),
        "molotov": frozenset({"inferno_start"}),
        "incendiary": frozenset({"inferno_start"}),
        "unknown": frozenset({"inferno_start"}),
    },
    "effect_start": {
        "smoke": frozenset({"smoke_start"}),
        "decoy": frozenset({"decoy_start"}),
        "molotov": frozenset({"inferno_start"}),
        "incendiary": frozenset({"inferno_start"}),
        "unknown": frozenset({"inferno_start"}),
    },
    "expiration": {
        "smoke": frozenset({"smoke_expired"}),
        "decoy": frozenset({"decoy_expired"}),
        "molotov": frozenset({"inferno_expired"}),
        "incendiary": frozenset({"inferno_expired"}),
        "unknown": frozenset({"inferno_expired"}),
    },
    "destroy": {
        utility_type: frozenset({"projectile_destroy"})
        for utility_type in UTILITY_TYPES
    },
    "extinguish": {
        "molotov": frozenset({"spatial_smoke_overlap"}),
        "incendiary": frozenset({"spatial_smoke_overlap"}),
        "unknown": frozenset({"spatial_smoke_overlap"}),
    },
}
UTILITY_SORT_ORDER = ("round_number", "sequence_in_round", "event_id")
REQUIRED_UTILITY_QUALITY_CHECKS = frozenset(
    {
        "utility_event_contract",
        "utility_throw_reconciliation",
        "utility_lifecycle",
        "utility_flash_attribution",
        "utility_damage_reconciliation",
        "utility_temporal_spatial_consistency",
        "utility_determinism",
        "utility_observation_coverage",
    }
)
REQUIRED_UTILITY_QUALITY_METRICS = frozenset(
    {
        "utility_throws",
        "utility_canonical_events",
        "utility_throw_callbacks",
        "utility_bounce_callbacks",
        "utility_lifecycle_callbacks",
        "utility_player_flashed_callbacks",
        "utility_damage_callbacks",
        "utility_flash_effects",
        "utility_damage_effects",
        "utility_contract_violations",
        "utility_throw_reconciliation_mismatches",
        "utility_player_stats_mismatches",
        "utility_callback_accounting_violations",
        "utility_unmatched_callbacks",
        "utility_orphan_callbacks",
        "utility_inferred_callbacks",
        "utility_deduplicated_callbacks",
        "utility_lifecycle_violations",
        "utility_flash_attribution_mismatches",
        "utility_damage_reconciliation_mismatches",
        "utility_replay_projection_mismatches",
        "utility_temporal_spatial_violations",
        "utility_determinism_violations",
        "utility_observation_warnings",
        "utility_missing_type_observations",
        "utility_missing_actor_observations",
        "utility_missing_launch_tick_observations",
        "utility_missing_launch_position_observations",
        "utility_missing_launch_view_observations",
        "utility_missing_thrower_velocity_observations",
        "utility_missing_projectile_velocity_observations",
        "utility_missing_trajectory_observations",
        "utility_missing_lifecycle_observations",
        "utility_missing_affected_player_observations",
        "utility_missing_flash_duration_observations",
        "utility_inferred_correlations",
        "utility_observed_effect_correlations",
        "utility_inferred_effect_correlations",
        "utility_unavailable_effect_correlations",
    }
)
UTILITY_CALLBACK_GROUPS = (
    "throws",
    "bounces",
    "lifecycle",
    "player_flashed",
    "damage",
)
UTILITY_CALLBACK_FIELDS = (
    "observed",
    "exact_correlated",
    "inferred_correlated",
    "orphaned",
    "deduplicated",
    "unmatched",
)
HARD_UTILITY_QUALITY_METRICS = frozenset(
    {
        "utility_contract_violations",
        "utility_throw_reconciliation_mismatches",
        "utility_player_stats_mismatches",
        "utility_callback_accounting_violations",
        "utility_unmatched_callbacks",
        "utility_lifecycle_violations",
        "utility_flash_attribution_mismatches",
        "utility_damage_reconciliation_mismatches",
        "utility_replay_projection_mismatches",
        "utility_temporal_spatial_violations",
        "utility_determinism_violations",
    }
)
COMBAT_EVENT_TYPES = frozenset(
    {"weapon_equip", "weapon_reload", "weapon_fire", "bullet_damage", "player_hurt", "kill"}
)
COMBAT_AVAILABILITY = frozenset({"observed", "derived", "unavailable"})
COMBAT_RELATIONS = frozenset({"enemy", "friendly", "self", "world", "unknown"})
COMBAT_CORRELATIONS = frozenset({"exact", "inferred", "unavailable"})
COMBAT_HITGROUPS = frozenset({
    "generic", "head", "chest", "stomach", "left_arm", "right_arm",
    "left_leg", "right_leg", "neck", "gear",
})
REQUIRED_COMBAT_QUALITY_CHECKS = frozenset(
    {
        "combat_contract",
        "combat_callback_accounting",
        "combat_player_stats_projection",
        "combat_replay_projection",
        "combat_native_deltas",
        "combat_determinism",
        "combat_observation_coverage",
    }
)
REQUIRED_COMBAT_QUALITY_METRICS = frozenset(
    {
        "combat_ledger_events",
        "combat_contract_violations",
        "combat_callback_accounting_violations",
        "combat_player_stats_mismatches",
        "combat_replay_projection_mismatches",
        "combat_native_delta_mismatches",
        "combat_determinism_violations",
        "combat_missing_impact_positions",
        "combat_missing_reload_ends",
        "combat_unavailable_shot_results",
        "combat_discarded_callbacks",
    }
)
HARD_COMBAT_QUALITY_METRICS = frozenset(
    {
        "combat_contract_violations",
        "combat_callback_accounting_violations",
        "combat_player_stats_mismatches",
        "combat_replay_projection_mismatches",
        "combat_native_delta_mismatches",
        "combat_determinism_violations",
    }
)
COMBAT_CALLBACK_GROUPS = tuple(sorted(COMBAT_EVENT_TYPES))


class ValidationErrors(list[str]):
    """Retain a bounded error sample while counting every omitted message."""

    def __init__(self, limit: int | None = None) -> None:
        super().__init__()
        self.limit = MAX_RETAINED_VALIDATION_ERRORS if limit is None else limit
        if isinstance(self.limit, bool) or self.limit < 1:
            raise ValueError("validation error limit must be a positive integer")
        self.omitted_count = 0

    def append(self, message: str) -> None:
        if len(self) < self.limit:
            super().append(message)
            return
        self.omitted_count += 1

    def as_report(self) -> list[str]:
        report = list(self)
        if self.omitted_count:
            report.append(
                "validation: "
                f"se omitieron {self.omitted_count} errores adicionales "
                f"tras alcanzar el limite de {self.limit}"
            )
        return report


@dataclass(frozen=True)
class ArtifactSpec:
    artifact_type: str
    path_pattern: re.Pattern[str]
    schema_id: str
    file_format: str
    records_field: str | None = None
    is_required: bool = True
    is_singleton: bool = True
    compression: str | None = None


@dataclass(frozen=True)
class StreamedArtifact:
    file_path: Path
    record_count: int


STREAMED_JSONL_ARTIFACT_TYPES = frozenset(
    {"tactical_observations", "tactical_oracle"}
)


ARTIFACT_SPECS = (
    ArtifactSpec("match", re.compile(r"core/match\.json"), "stratai.match@1", "json"),
    ArtifactSpec(
        "participants",
        re.compile(r"core/participants\.json"),
        "stratai.participants@1",
        "json",
        "players",
    ),
    ArtifactSpec(
        "rounds", re.compile(r"core/rounds\.json"), "stratai.rounds@2", "json", "rounds"
    ),
    ArtifactSpec(
        "combat_events",
        re.compile(r"events/combat_events\.jsonl"),
        "stratai.combat_event@2",
        "jsonl",
    ),
    ArtifactSpec(
        "utility_events",
        re.compile(r"events/utility_events\.jsonl"),
        "stratai.utility_event@2",
        "jsonl",
    ),
    ArtifactSpec(
        "objective_events",
        re.compile(r"events/objective_events\.jsonl"),
        "stratai.objective_event@2",
        "jsonl",
    ),
    ArtifactSpec(
        "player_states",
        re.compile(r"states/player_states/round_[0-9]{3}\.jsonl"),
        "stratai.player_state@3",
        "jsonl",
        is_singleton=False,
    ),
    ArtifactSpec(
        "engagements",
        re.compile(r"derived/engagements\.json"),
        "stratai.engagements@2",
        "json",
        "engagements",
    ),
    ArtifactSpec(
        "trades",
        re.compile(r"derived/trades\.json"),
        "stratai.trades@1",
        "json",
        "trade_candidates",
    ),
    ArtifactSpec(
        "match_metadata",
        re.compile(r"core/match_metadata\.json"),
        "stratai.match_metadata@1",
        "json",
    ),
    ArtifactSpec(
        "economy_rounds",
        re.compile(r"derived/economy_rounds\.json"),
        "stratai.economy_round@1",
        "json",
        "rounds",
    ),
    ArtifactSpec(
        "economy_players",
        re.compile(r"derived/economy_players\.json"),
        "stratai.economy_player@1",
        "json",
        "players",
    ),
    ArtifactSpec(
        "player_stats",
        re.compile(r"derived/player_stats\.json"),
        "stratai.player_stats@1",
        "json",
        "players",
    ),
    ArtifactSpec(
        "clutch_events",
        re.compile(r"derived/clutch_events\.json"),
        "stratai.clutch_event@1",
        "json",
        "clutch_events",
    ),
    ArtifactSpec(
		"decisions",
		re.compile(r"causal/decisions\.jsonl"),
		"stratai.decision@1",
		"jsonl",
	),
    ArtifactSpec(
        "decision_features",
        re.compile(r"causal/decision_features\.jsonl"),
        "stratai.decision_features@1",
        "jsonl",
    ),
    ArtifactSpec(
        "oracle_context",
        re.compile(r"causal/oracle_context\.jsonl"),
        "stratai.oracle_context@1",
        "jsonl",
    ),
    ArtifactSpec(
        "decision_outcomes",
        re.compile(r"causal/outcomes\.jsonl"),
        "stratai.decision_outcome@1",
        "jsonl",
    ),
    ArtifactSpec(
        "quality_masks",
        re.compile(r"causal/quality_masks\.jsonl"),
        "stratai.quality_mask@1",
        "jsonl",
    ),
    ArtifactSpec(
        "tactical_sampling",
        re.compile(r"states/tactical/sampling\.json"),
        "stratai.tactical_sampling@1",
        "json",
    ),
    ArtifactSpec(
        "tactical_observations",
        re.compile(r"states/tactical/observed\.jsonl\.gz"),
        "stratai.tactical_physical_observation@1",
        "jsonl",
        compression="gzip",
    ),
    ArtifactSpec(
        "tactical_oracle",
        re.compile(r"states/tactical/oracle\.jsonl"),
        "stratai.tactical_oracle_state@1",
        "jsonl",
    ),
    ArtifactSpec(
        "tactical_gaps",
        re.compile(r"states/tactical/gaps\.jsonl"),
        "stratai.tactical_sampling_gap@1",
        "jsonl",
    ),
    ArtifactSpec(
        "quality_report",
        re.compile(r"diagnostics/quality_report\.json"),
        "stratai.quality_report@1",
        "json",
    ),
    ArtifactSpec(
        "replay_index",
        re.compile(r"presentation/replay/index\.json"),
        "stratai.replay_index@5",
        "json",
        is_required=False,
    ),
    ArtifactSpec(
        "replay_round",
        re.compile(r"presentation/replay/round_[0-9]{3}\.json\.gz"),
        "stratai.replay_round@5",
        "json",
        is_required=False,
        is_singleton=False,
        compression="gzip",
    ),
)
SPEC_BY_TYPE = {spec.artifact_type: spec for spec in ARTIFACT_SPECS}
REQUIRED_RECORD_FIELDS = {
    "participants": ("player_id", "steam_id", "display_name", "team_id"),
    "rounds": (
        "round_id",
        "round_number",
        "start_tick",
        "end_tick",
        "winner_side",
        "winner_team_id",
        "win_reason",
        "raw_win_reason_code",
        "bomb_planted",
        "bomb_site",
        "bomb_tick",
        "objective",
    ),
    "combat_events": (
		"schema_id",
		"match_id",
        "event_id",
        "round_id",
        "round_number",
        "tick",
		"sequence_in_tick",
		"sequence_in_round",
        "event_type",
		"source",
		"source_event_ids",
		"tick_status",
		"subtick",
		"subtick_status",
		"time_seconds",
		"time_seconds_status",
		"time_seconds_source",
		"actor_player_id",
		"actor_side",
		"actor_status",
		"actor_source",
		"target_player_id",
		"target_side",
		"target_status",
		"target_source",
		"assister_player_id",
		"assister_side",
		"assister_status",
		"assister_source",
		"relation",
		"weapon",
		"weapon_status",
		"weapon_source",
		"weapon_is_utility",
		"actor_position",
		"actor_position_status",
		"actor_position_source",
		"target_position",
		"target_position_status",
		"target_position_source",
		"shot_id",
		"correlation_status",
		"correlation_source",
		"shot_result",
		"shot_result_status",
		"shot_result_source",
		"shot_result_availability_tick",
		"view_yaw",
		"view_pitch",
		"impact_position",
		"impact_position_status",
		"impact_position_source",
		"bullet_distance_world_units",
		"damage_direction",
		"penetrated_objects",
		"no_scope",
		"attacker_in_air",
		"through_smoke",
		"attacker_blind",
		"kill_distance_world_units",
		"health_damage",
		"health_damage_taken",
		"armor_damage",
		"armor_damage_taken",
		"health_before",
		"health_after",
		"armor_before",
		"armor_after",
		"damage_status",
		"damage_source",
		"hitgroup",
		"hitgroup_status",
		"hitgroup_source",
		"is_headshot",
        "is_kill",
		"assisted_flash",
		"reload_phase",
		"reload_end_tick",
		"reload_end_status",
		"previous_weapon",
		"previous_weapon_status",
		"is_weapon_switch",
		"ammo_in_magazine",
		"ammo_reserve",
		"ammo_status",
		"ammo_source",
		"reaction_time_ms",
		"time_to_damage_ms",
    ),
    "utility_events": (
        "event_id",
        "source_throw_id",
        "source_entity",
        "source_entity_status",
        "source_entity_source",
        "round_id",
        "round_number",
        "sequence_in_round",
        "event_type",
        "utility_type",
        "utility_type_status",
        "utility_type_source",
        "thrower_player_id",
        "thrower_side",
        "thrower_status",
        "thrower_source",
        "correlation",
        "launch",
        "trajectory",
        "lifecycle",
        "affected_players",
        "flash_summary",
        "damage_summary",
        "details",
    ),
    "objective_events": (
        "event_id",
        "round_id",
        "round_number",
        "tick",
        "sequence_in_tick",
        "event_type",
        "actor_player_id",
        "actor_side",
        "site",
        "position",
        "position_status",
        "source",
        "state_after",
        "phase_after",
        "attempt_id",
        "attempt_outcome",
        "attempt_start_observed",
        "action_duration_ms",
        "has_defuse_kit",
        "bomb_entity_id",
    ),
    "player_states": (
        "state_id",
        "round_id",
        "round_number",
        "tick",
        "player_id",
        "team_id",
        "side",
        "position",
        "health",
        "is_alive",
        "horizontal_velocity_world_units_per_second",
        "velocity_vector_world_units_per_second",
        "velocity_source",
        "velocity_measurement_window_ticks",
        "active_weapon",
        "active_weapon_status",
        "last_observed_active_weapon",
        "last_observed_active_weapon_tick",
        "has_c4",
        "has_defuse_kit",
        "is_planting",
        "is_defusing",
        "objective_phase",
        "phase_time_remaining_ms",
        "round_time_remaining_ms",
        "round_clock_remaining_ms",
        "bomb_time_remaining_ms",
    ),
    "engagements": (
        "engagement_id",
        "round_id",
        "round_number",
        "start_tick",
        "start_sequence_in_tick",
        "end_tick",
        "end_sequence_in_tick",
        "duration_ms",
        "engagement_type",
        "initiator",
        "first_aggressor",
        "first_damage_dealer",
        "participants",
        "exchanges",
        "causal_context",
        "outcome_context",
        "source_event_ids",
        "algorithm_version",
    ),
    "trades": (
        "trade_candidate_id",
        "round_id",
        "round_number",
        "death_tick",
        "death_sequence_in_tick",
        "original_kill_event_id",
        "original_victim_player_id",
        "original_killer_player_id",
        "player_id_usage",
        "eligible_teammate_player_ids",
        "eligibility_status",
        "eligibility_source",
        "eligibility_state_ids",
        "connections",
        "trade_possible",
        "trade_possible_status",
        "attempt_event_ids",
        "evaluation",
        "trade_completion_id",
        "counter_trade_of_completion_id",
        "window_ms",
        "window_ticks",
        "window_end_tick",
        "source_event_ids",
    ),
    "economy_rounds": ("round_id", "round_number", "team_id", "side", "loss_bonus", "rewards"),
    "economy_players": ("round_id", "round_number", "player_id", "team_id", "side", "money", "transactions"),
    "player_stats": ("player_id", "team_id", "native_scoreboard_status", "derived", "clutch", "rating", "metrics", "provenance"),
    "clutch_events": ("clutch_id", "round_id", "round_number", "player_id", "team_id", "side", "enemies_at_start", "state", "attempt", "result", "source_event_ids"),
    "decision_features": (
        "schema_id", "match_id", "decision_id", "decision_type", "round_number", "t0_tick",
        "availability_tick_max", "participant_count", "observed_participant_states",
        "alive_participant_count", "initial_distance_world_units", "initial_distance_status",
        "bomb_context_status", "economy_context_status", "enemies_exposed_count",
        "enemies_exposed_status", "round_clock_remaining_ms", "bomb_time_remaining_ms",
        "source_state_count", "trade_possible", "trade_possible_status",
        "nearest_teammate_distance_world_units", "nearest_teammate_distance_status",
        "nearest_connection_time_ms", "nearest_connection_time_status",
        "any_line_of_sight", "line_of_sight_status", "minimum_facing_delta_deg",
        "facing_status",
    ),
    "decisions": (
        "schema_id", "match_id", "decision_id", "round_number",
        "actor_player_id", "actor_id_usage", "observed_state_ref",
        "state_availability_status", "t0_tick", "decision_type",
        "action_taken", "availability_tick", "availability_status",
        "causal_role", "visibility_scope", "source", "source_record_id",
        "source_event_ids", "algorithm_version",
    ),
    "oracle_context": (
        "schema_id", "match_id", "decision_id", "round_number", "t0_tick",
        "status", "available", "field_names", "abstentions",
    ),
    "decision_outcomes": (
        "schema_id", "match_id", "decision_id", "round_number", "t0_tick",
        "outcome", "outcome_tick", "duration_ms", "winner_observed",
        "loser_count", "terminal_kill_count", "trade_candidate_count",
        "trade_completion_count", "survival_status", "disengagement_status",
        "horizons",
    ),
    "quality_masks": (
        "schema_id", "match_id", "decision_id", "round_number", "t0_tick",
        "available_fields", "unavailable_fields", "inferred_fields", "warning_flags",
    ),
    "tactical_sampling": (
        "schema_id", "match_id", "identity_semantics", "join_keys", "sampling",
        "physical_row_count", "oracle_row_count", "gap_count",
    ),
    "tactical_observations": (
        "schema_id", "match_id", "identity_semantics", "join_keys", "round_number",
        "tick", "availability_tick", "status", "causal_role", "visibility_scope",
        "source", "provenance", "state",
    ),
    "tactical_oracle": (
        "schema_id", "match_id", "identity_semantics", "join_keys", "round_number",
        "tick", "availability_tick", "status", "causal_role", "visibility_scope",
        "source", "provenance", "state",
    ),
    "tactical_gaps": (
        "schema_id", "match_id", "identity_semantics", "join_keys", "round_number",
        "tick", "availability_tick", "status", "causal_role", "visibility_scope",
        "source", "reason", "provenance",
    ),
}
ENGAGEMENT_ROLE_FIELDS = (
    "player_id",
    "status",
    "source",
    "availability_tick",
    "source_event_ids",
    "confidence",
)
ENGAGEMENT_EXCHANGE_FIELDS = (
    "exchange_id",
    "tick",
    "sequence_in_tick",
    "sequence_in_round",
    "actor_player_id",
    "target_player_id",
    "is_kill",
    "kill_event_id",
    "source_event_ids",
)


def load_json_object(path: Path, errors: list[str]) -> dict | None:
    try:
        if path.suffix == ".gz":
            with gzip.open(path, "rt", encoding="utf-8") as file:
                value = json.load(file)
        else:
            value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        errors.append(f"{path}: JSON no válido ({error})")
        return None
    if not isinstance(value, dict):
        errors.append(f"{path}: la raíz debe ser un objeto JSON")
        return None
    return value


def iter_json_lines(
    path: Path, errors: list[str] | None
) -> Iterable[tuple[int, dict]]:
    try:
        if path.suffix == ".gz":
            handle_context = gzip.open(path, "rt", encoding="utf-8")
        else:
            handle_context = path.open("r", encoding="utf-8")
        with handle_context as handle:
            for line_number, line in enumerate(handle, start=1):
                if not line.strip():
                    if errors is not None:
                        errors.append(f"{path}:{line_number}: línea vacía")
                    continue
                try:
                    record = json.loads(line)
                except json.JSONDecodeError as error:
                    if errors is not None:
                        errors.append(
                            f"{path}:{line_number}: JSON no válido ({error})"
                        )
                    continue
                if not isinstance(record, dict):
                    if errors is not None:
                        errors.append(
                            f"{path}:{line_number}: cada línea debe ser un objeto"
                        )
                    continue
                yield line_number, record
    except (OSError, EOFError, UnicodeError) as error:
        if errors is not None:
            errors.append(f"{path}: no se pudo leer ({error})")


def load_json_lines(path: Path, errors: list[str]) -> list[dict]:
    return [record for _, record in iter_json_lines(path, errors)]


def validate_streamed_jsonl_artifact(
    file_path: Path,
    path: str,
    spec: ArtifactSpec,
    descriptor: Mapping[str, object],
    match_id: str,
    errors: list[str],
) -> int:
    raw_order = descriptor.get("sort_order")
    sort_order_valid = isinstance(raw_order, list) and all(
        isinstance(field, str) and field for field in raw_order
    )
    if not sort_order_valid:
        errors.append(f"canonical/{path}: sort_order debe ser una lista de campos")
    sort_order = raw_order if sort_order_valid else None
    sort_validation_active = bool(sort_order)
    sort_error_reported = False
    previous_sort_key: tuple | None = None
    required_fields = REQUIRED_RECORD_FIELDS.get(spec.artifact_type, ())
    record_count = 0

    for _, record in iter_json_lines(file_path, errors):
        index = record_count
        record_count += 1
        if record.get("schema_id") != spec.schema_id:
            errors.append(
                f"canonical/{path}: registro {index} tiene schema_id incorrecto"
            )
        if record.get("match_id") != match_id:
            errors.append(f"canonical/{path}: registro {index} tiene match_id incorrecto")
        missing = [field for field in required_fields if field not in record]
        if missing:
            errors.append(
                f"canonical/{path}: registro {index} no contiene {', '.join(missing)}"
            )
        if not sort_validation_active or sort_order is None:
            continue
        values = [nested_value(record, field) for field in sort_order]
        if any(value is None for value in values):
            if not sort_error_reported:
                errors.append(f"canonical/{path}: registro {index} no cumple sort_order")
                sort_error_reported = True
            sort_validation_active = False
            continue
        sort_key = tuple(sortable_value(value) for value in values)
        if previous_sort_key is not None and sort_key < previous_sort_key:
            if not sort_error_reported:
                errors.append(f"canonical/{path}: registros fuera de sort_order")
                sort_error_reported = True
            sort_validation_active = False
        previous_sort_key = sort_key

    if sort_order_valid and not sort_order and record_count >= 2:
        errors.append(f"canonical/{path}: una colección necesita sort_order")
    if descriptor.get("record_count") != record_count:
        errors.append(f"canonical/{path}: record_count no coincide")
    return record_count


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as file:
        for chunk in iter(lambda: file.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def validate_root_catalog(
    match_dir: Path,
    match_id: str,
    errors: list[str],
    expected_demo_checksum: str | None = None,
) -> None:
    manifest_path = match_dir / "manifest.json"
    if not manifest_path.is_file():
        errors.append("manifest.json: fichero requerido inexistente")
        return
    manifest = load_json_object(manifest_path, errors)
    if manifest is None:
        return
    if manifest.get("match_id") != match_id:
        errors.append("manifest.json: match_id no coincide")
    checksum = manifest.get("checksum")
    if not isinstance(checksum, str) or SHA256_PATTERN.fullmatch(checksum) is None:
        errors.append("manifest.json: checksum de demo no es un SHA-256 válido")
    elif expected_demo_checksum is not None and checksum != expected_demo_checksum:
        errors.append("manifest.json: checksum no corresponde a la demo")
    if manifest.get("parser_schema_version") != PARSER_SCHEMA_VERSION:
        errors.append(
            f"manifest.json: parser_schema_version debe ser {PARSER_SCHEMA_VERSION}"
        )
    if manifest.get("export_format_version") != EXPORT_FORMAT_VERSION:
        errors.append(
            f"manifest.json: export_format_version debe ser {EXPORT_FORMAT_VERSION}"
        )
    if manifest.get("validator_version") != VALIDATOR_VERSION:
        errors.append(f"manifest.json: validator_version debe ser {VALIDATOR_VERSION}")
    if manifest.get("validation_status") != "passed":
        errors.append("manifest.json: validation_status debe ser passed")
    if (
        not isinstance(manifest.get("committed_at"), str)
        or not manifest["committed_at"].strip()
    ):
        errors.append("manifest.json: committed_at debe ser una fecha no vacía")
    descriptors = manifest.get("artifacts")
    if not isinstance(descriptors, list) or not all(
        isinstance(item, dict) for item in descriptors
    ):
        errors.append("manifest.json: artifacts debe ser una lista de objetos")
        return

    declared_paths: list[str] = []
    for descriptor in descriptors:
        raw_path = descriptor.get("path")
        if not isinstance(raw_path, str) or not raw_path or "\\" in raw_path:
            errors.append("manifest.json: artifact.path no es una ruta POSIX válida")
            continue
        relative_path = PurePosixPath(raw_path)
        if (
            relative_path.is_absolute()
            or ".." in relative_path.parts
            or relative_path.as_posix() != raw_path
        ):
            errors.append(f"manifest.json: ruta insegura: {raw_path}")
            continue
        if not raw_path.startswith("canonical/"):
            errors.append(f"manifest.json: artefacto fuera de canonical/: {raw_path}")
            continue
        path = match_dir.joinpath(*relative_path.parts)
        declared_paths.append(raw_path)
        if not path.is_file():
            errors.append(f"manifest.json: fichero declarado inexistente: {raw_path}")
            continue
        if descriptor.get("bytes") != path.stat().st_size:
            errors.append(f"manifest.json: bytes no coincide para {raw_path}")
        checksum = descriptor.get("sha256")
        if not isinstance(checksum, str) or SHA256_PATTERN.fullmatch(checksum) is None:
            errors.append(f"manifest.json: sha256 no es válido para {raw_path}")
        elif checksum != sha256_file(path):
            errors.append(f"manifest.json: sha256 no coincide para {raw_path}")

    if declared_paths != sorted(declared_paths):
        errors.append("manifest.json: artifacts no está ordenado por path")
    if len(declared_paths) != len(set(declared_paths)):
        errors.append("manifest.json: hay paths duplicados")
    disk_paths = {
        path.relative_to(match_dir).as_posix()
        for path in match_dir.rglob("*")
        if path.is_file() and path != match_dir / "manifest.json"
    }
    if set(declared_paths) != disk_paths:
        errors.append(
            "manifest.json: el catálogo no coincide con los ficheros existentes"
        )


def resolve_artifact_path(
    canonical_dir: Path, raw_path: object, errors: list[str]
) -> tuple[str, Path] | None:
    if not isinstance(raw_path, str) or not raw_path or "\\" in raw_path:
        errors.append(
            "canonical/manifest.json: artifact.path no es una ruta POSIX válida"
        )
        return None
    artifact_path = PurePosixPath(raw_path)
    if (
        artifact_path.is_absolute()
        or ".." in artifact_path.parts
        or artifact_path.as_posix() != raw_path
    ):
        errors.append(f"canonical/manifest.json: ruta insegura: {raw_path}")
        return None
    return raw_path, canonical_dir.joinpath(*artifact_path.parts)


def get_artifact_spec(
    descriptor: Mapping[str, object], path: str, errors: list[str]
) -> ArtifactSpec | None:
    artifact_type = descriptor.get("artifact_type")
    spec = SPEC_BY_TYPE.get(artifact_type) if isinstance(artifact_type, str) else None
    if spec is None:
        errors.append(f"canonical/{path}: artifact_type desconocido: {artifact_type}")
        return None
    if spec.path_pattern.fullmatch(path) is None:
        errors.append(f"canonical/{path}: ruta inválida para {artifact_type}")
    if descriptor.get("schema_id") != spec.schema_id:
        errors.append(f"canonical/{path}: schema_id debe ser {spec.schema_id}")
    if descriptor.get("format") != spec.file_format:
        errors.append(f"canonical/{path}: format debe ser {spec.file_format}")
    if descriptor.get("compression") != spec.compression:
        errors.append(f"canonical/{path}: compression debe ser {spec.compression}")
    return spec


def nested_value(record: Mapping[str, object], field: str) -> object:
    value: object = record
    for part in field.split("."):
        if not isinstance(value, Mapping) or part not in value:
            return None
        value = value[part]
    return value


def sortable_value(value: object) -> tuple[int, object]:
    if isinstance(value, bool):
        return 0, int(value)
    if isinstance(value, (int, float)):
        return 1, value
    return 2, str(value)


def validate_sort_order(
    path: str, records: list[dict], raw_order: object, errors: list[str]
) -> None:
    if not isinstance(raw_order, list) or not all(
        isinstance(field, str) and field for field in raw_order
    ):
        errors.append(f"canonical/{path}: sort_order debe ser una lista de campos")
        return
    if len(records) < 2:
        return
    if not raw_order:
        errors.append(f"canonical/{path}: una colección necesita sort_order")
        return
    keys: list[tuple] = []
    for index, record in enumerate(records):
        values = [nested_value(record, field) for field in raw_order]
        if any(value is None for value in values):
            errors.append(f"canonical/{path}: registro {index} no cumple sort_order")
            return
        keys.append(tuple(sortable_value(value) for value in values))
    if keys != sorted(keys):
        errors.append(f"canonical/{path}: registros fuera de sort_order")


def validate_required_fields(
    spec: ArtifactSpec, path: str, records: list[dict], errors: list[str]
) -> None:
    required_fields = REQUIRED_RECORD_FIELDS.get(spec.artifact_type, ())
    for index, record in enumerate(records):
        missing = [field for field in required_fields if field not in record]
        if missing:
            errors.append(
                f"canonical/{path}: registro {index} no contiene {', '.join(missing)}"
            )


def is_finite_number(value: object) -> bool:
    return (
        not isinstance(value, bool)
        and isinstance(value, (int, float))
        and math.isfinite(float(value))
    )


def finite_vector(value: object) -> tuple[float, float, float] | None:
    if not isinstance(value, Mapping):
        return None
    coordinates = (value.get("x"), value.get("y"), value.get("z"))
    if not all(is_finite_number(coordinate) for coordinate in coordinates):
        return None
    return tuple(float(coordinate) for coordinate in coordinates)


def validate_velocity_contract(
    record: Mapping[str, object], location: str, is_alive: bool, errors: list[str]
) -> None:
    source = record.get("velocity_source")
    horizontal = record.get("horizontal_velocity_world_units_per_second")
    vector_value = record.get("velocity_vector_world_units_per_second")
    interval = record.get("velocity_measurement_window_ticks")

    if source not in VELOCITY_SOURCES:
        errors.append(f"{location}: velocity_source no es valido")
        return
    if not is_alive and source != "not_applicable":
        errors.append(
            f"{location}: un jugador muerto requiere velocity_source=not_applicable"
        )
    if is_alive and source == "not_applicable":
        errors.append(
            f"{location}: velocity_source=not_applicable solo admite jugadores muertos"
        )

    if source in UNAVAILABLE_VELOCITY_SOURCES:
        if horizontal is not None or vector_value is not None or interval is not None:
            errors.append(
                f"{location}: una velocidad no disponible debe tener valor, vector e intervalo null"
            )
        return

    vector = finite_vector(vector_value)
    if not is_finite_number(horizontal):
        errors.append(f"{location}: la velocidad horizontal disponible debe ser finita")
    elif not 0 <= float(horizontal) <= MAX_HORIZONTAL_VELOCITY_UPS:
        errors.append(f"{location}: la velocidad horizontal no es plausible")
    if vector is None:
        errors.append(f"{location}: el vector de velocidad disponible debe ser finito")
    else:
        expected_horizontal = math.hypot(vector[0], vector[1])
        if expected_horizontal > MAX_HORIZONTAL_VELOCITY_UPS:
            errors.append(
                f"{location}: el vector horizontal de velocidad no es plausible"
            )
        if abs(vector[2]) > MAX_VERTICAL_VELOCITY_UPS:
            errors.append(f"{location}: la velocidad vertical no es plausible")
        if is_finite_number(horizontal) and not math.isclose(
            float(horizontal), expected_horizontal, rel_tol=1e-9, abs_tol=1e-6
        ):
            errors.append(
                f"{location}: la velocidad horizontal no coincide con hypot(x, y) del vector"
            )

    if isinstance(interval, bool) or not isinstance(interval, int):
        errors.append(f"{location}: el intervalo de velocidad debe ser entero")
    elif source == "native" and interval != 0:
        errors.append(f"{location}: velocity_source=native requiere intervalo 0")
    elif source == "position_delta" and not (
        1 <= interval <= MAX_POSITION_DELTA_INTERVAL_TICKS
    ):
        errors.append(
            f"{location}: velocity_source=position_delta requiere intervalo entre 1 y {MAX_POSITION_DELTA_INTERVAL_TICKS}"
        )


def validate_active_weapon_contract(
    record: Mapping[str, object], location: str, is_alive: bool, errors: list[str]
) -> None:
    weapon = record.get("active_weapon")
    status = record.get("active_weapon_status")
    last_weapon = record.get("last_observed_active_weapon")
    last_tick = record.get("last_observed_active_weapon_tick")

    if status not in ACTIVE_WEAPON_STATUSES:
        errors.append(f"{location}: active_weapon_status no es valido")
        return
    if not is_alive and status != "not_applicable":
        errors.append(
            f"{location}: un jugador muerto requiere active_weapon_status=not_applicable"
        )
    if is_alive and status == "not_applicable":
        errors.append(
            f"{location}: active_weapon_status=not_applicable solo admite jugadores muertos"
        )

    if status == "observed":
        if not isinstance(weapon, str) or not weapon.strip():
            errors.append(
                f"{location}: active_weapon observado debe ser un nombre no vacio"
            )
    elif weapon is not None:
        errors.append(
            f"{location}: active_weapon debe ser null si no fue observado en el tick"
        )

    history_is_complete = (
        isinstance(last_weapon, str)
        and bool(last_weapon.strip())
        and isinstance(last_tick, int)
        and not isinstance(last_tick, bool)
    )
    if (last_weapon is None) != (last_tick is None):
        errors.append(
            f"{location}: last_observed_active_weapon y su tick deben ser ambos null o ambos validos"
        )
    elif last_weapon is not None and not history_is_complete:
        errors.append(
            f"{location}: el ultimo arma observada requiere nombre no vacio y tick entero"
        )
    elif history_is_complete:
        tick = record.get("tick")
        if last_tick < 0 or (
            isinstance(tick, int) and not isinstance(tick, bool) and last_tick > tick
        ):
            errors.append(
                f"{location}: el tick del ultimo arma observada no puede ser futuro"
            )
        if status == "observed":
            errors.append(
                f"{location}: un arma actual observada no debe duplicarse como historial"
            )


def validate_nullable_nonnegative_integer(
    value: object, field: str, location: str, errors: list[str]
) -> bool:
    if value is None:
        return True
    if isinstance(value, bool) or not isinstance(value, int) or value < 0:
        errors.append(f"{location}: {field} debe ser null o un entero no negativo")
        return False
    return True


def validate_player_objective_contract(
    record: Mapping[str, object], location: str, is_alive: bool, errors: list[str]
) -> None:
    boolean_fields = ("has_c4", "has_defuse_kit", "is_planting", "is_defusing")
    for field in boolean_fields:
        if not isinstance(record.get(field), bool):
            errors.append(f"{location}: {field} debe ser booleano")

    phase = record.get("objective_phase")
    if phase not in OBJECTIVE_PHASES:
        errors.append(f"{location}: objective_phase no es valido")

    round_time = record.get("round_time_remaining_ms")
    if (
        isinstance(round_time, bool)
        or not isinstance(round_time, int)
        or round_time < 0
    ):
        errors.append(
            f"{location}: round_time_remaining_ms debe ser un entero no negativo"
        )
    phase_time = record.get("phase_time_remaining_ms")
    round_clock = record.get("round_clock_remaining_ms")
    bomb_time = record.get("bomb_time_remaining_ms")
    validate_nullable_nonnegative_integer(
        phase_time, "phase_time_remaining_ms", location, errors
    )
    validate_nullable_nonnegative_integer(
        round_clock, "round_clock_remaining_ms", location, errors
    )
    validate_nullable_nonnegative_integer(
        bomb_time, "bomb_time_remaining_ms", location, errors
    )

    phase_time_is_valid = (
        isinstance(phase_time, int)
        and not isinstance(phase_time, bool)
        and phase_time >= 0
    )
    if not phase_time_is_valid:
        errors.append(
            f"{location}: phase_time_remaining_ms debe ser un entero no negativo"
        )
    if (
        phase_time_is_valid
        and isinstance(round_time, int)
        and not isinstance(round_time, bool)
        and round_time != phase_time
    ):
        errors.append(
            f"{location}: round_time_remaining_ms legacy debe coincidir con phase_time_remaining_ms"
        )

    if phase in {"preplant", "planting"}:
        if round_clock != phase_time or bomb_time is not None:
            errors.append(
                f"{location}: la fase {phase} requiere reloj de ronda=phase y reloj de bomba null"
            )
    elif phase in {"planted", "defusing"}:
        if round_clock is not None or bomb_time != phase_time:
            errors.append(
                f"{location}: la fase {phase} requiere reloj de ronda null y reloj de bomba=phase"
            )
    elif phase == "resolved" and (
        phase_time != 0 or round_clock is not None or bomb_time is not None
    ):
        errors.append(
            f"{location}: la fase resolved requiere phase=0 y relojes activos null"
        )

    side = record.get("side")
    has_c4 = record.get("has_c4") is True
    has_kit = record.get("has_defuse_kit") is True
    is_planting = record.get("is_planting") is True
    is_defusing = record.get("is_defusing") is True
    if is_planting and is_defusing:
        errors.append(f"{location}: un jugador no puede plantar y defusar a la vez")
    if not is_alive and (has_c4 or is_planting or is_defusing):
        errors.append(
            f"{location}: un jugador muerto no puede portar C4, plantar ni defusar"
        )
    if (has_c4 or is_planting) and side != "t":
        errors.append(f"{location}: C4 y plantado requieren side=t")
    if (has_kit or is_defusing) and side != "ct":
        errors.append(f"{location}: kit y defuse requieren side=ct")
    if is_planting and (not has_c4 or phase != "planting"):
        errors.append(
            f"{location}: is_planting requiere has_c4=true y objective_phase=planting"
        )
    if is_defusing and phase != "defusing":
        errors.append(f"{location}: is_defusing requiere objective_phase=defusing")
    if phase in {"planted", "defusing", "resolved"} and has_c4:
        errors.append(f"{location}: no puede haber portador C4 en fase {phase}")


def validate_player_state_semantics(records: list[dict], errors: list[str]) -> None:
    samples: dict[tuple[object, object], list[dict]] = {}
    states_by_tick: dict[tuple[object, object], list[dict]] = {}
    has_positive_velocity = False
    for index, record in enumerate(records):
        state_id = record.get("state_id")
        label = state_id if isinstance(state_id, str) and state_id else str(index)
        location = f"canonical/states/player_states: estado {label}"
        tick = record.get("tick")
        if isinstance(tick, bool) or not isinstance(tick, int) or tick < 0:
            errors.append(f"{location}: tick debe ser un entero no negativo")
        is_alive = record.get("is_alive")
        if not isinstance(is_alive, bool):
            errors.append(f"{location}: is_alive debe ser booleano")
            continue
        if finite_vector(record.get("position")) is None:
            errors.append(f"{location}: position debe contener x, y, z finitos")
        validate_velocity_contract(record, location, is_alive, errors)
        validate_active_weapon_contract(record, location, is_alive, errors)
        validate_player_objective_contract(record, location, is_alive, errors)
        tick_key = (record.get("round_number"), record.get("tick"))
        states_by_tick.setdefault(tick_key, []).append(record)
        if is_alive:
            horizontal = record.get("horizontal_velocity_world_units_per_second")
            if is_finite_number(horizontal) and float(horizontal) > 1e-6:
                has_positive_velocity = True
            key = (record.get("round_number"), record.get("player_id"))
            samples.setdefault(key, []).append(record)

    moving_intervals = 0
    for player_samples in samples.values():
        ordered = sorted(
            player_samples,
            key=lambda record: sortable_value(record.get("tick")),
        )
        previous: Mapping[str, object] | None = None
        for record in ordered:
            current_position = finite_vector(record.get("position"))
            previous_position = (
                finite_vector(previous.get("position"))
                if previous is not None
                else None
            )
            current_tick = record.get("tick")
            previous_tick = previous.get("tick") if previous is not None else None
            if (
                current_position is not None
                and previous_position is not None
                and isinstance(current_tick, int)
                and not isinstance(current_tick, bool)
                and isinstance(previous_tick, int)
                and not isinstance(previous_tick, bool)
                and current_tick > previous_tick
                and math.dist(current_position, previous_position) > 0.01
            ):
                moving_intervals += 1
            previous = record
    if moving_intervals >= 2 and not has_positive_velocity:
        errors.append(
            "canonical/states/player_states: las posiciones cambian repetidamente "
            "pero todas las velocidades disponibles de la partida son cero"
        )

    for (round_number, tick), tick_states in states_by_tick.items():
        location = f"canonical/states/player_states: ronda {round_number}, tick {tick}"
        for field in (
            "objective_phase",
            "phase_time_remaining_ms",
            "round_clock_remaining_ms",
            "bomb_time_remaining_ms",
        ):
            values = {state.get(field) for state in tick_states}
            if len(values) > 1:
                errors.append(f"{location}: {field} discrepa entre jugadores")
        carriers = [state for state in tick_states if state.get("has_c4") is True]
        if len(carriers) > 1:
            errors.append(f"{location}: hay mas de un portador de C4")


def engagement_event_key(event: Mapping[str, object]) -> tuple[int, int, int, int, str]:
    return (
        int(event.get("round_number", -1)),
        int(event.get("tick", -1)),
        int(event.get("sequence_in_tick", -1)),
        int(event.get("sequence_in_round", -1)),
        str(event.get("event_id", "")),
    )


def engagement_ticks(milliseconds: int, tick_rate: float) -> int:
    return math.ceil(milliseconds * tick_rate / 1_000.0)


def combat_event_closure(
    event_id: str, event_by_id: Mapping[str, Mapping[str, object]]
) -> set[str]:
    seen: set[str] = set()
    pending = [event_id]
    while pending:
        current_id = pending.pop()
        if current_id in seen:
            continue
        current = event_by_id.get(current_id)
        if not isinstance(current, Mapping):
            continue
        seen.add(current_id)
        sources = current.get("source_event_ids")
        if isinstance(sources, list):
            pending.extend(source for source in sources if isinstance(source, str))
    return seen


def validate_engagement_role(
    role: object,
    participant_ids: set[str],
    event_by_id: Mapping[str, Mapping[str, object]],
    first_exchange: Mapping[str, object],
    location: str,
    errors: list[str],
) -> None:
    if not isinstance(role, Mapping):
        errors.append(f"{location}: debe ser un objeto")
        return
    missing = [field for field in ENGAGEMENT_ROLE_FIELDS if field not in role]
    if missing:
        errors.append(f"{location}: faltan campos {', '.join(missing)}")
    player_id = role.get("player_id")
    status = role.get("status")
    if status not in ENGAGEMENT_ROLE_STATUSES:
        errors.append(f"{location}: status no es valido")
    if status == "unavailable":
        if player_id is not None or role.get("availability_tick") is not None:
            errors.append(f"{location}: unavailable requiere jugador y tick null")
        return
    if player_id not in participant_ids:
        errors.append(f"{location}: player_id no pertenece al engagement")
    availability_tick = role.get("availability_tick")
    if not isinstance(availability_tick, int) or isinstance(availability_tick, bool):
        errors.append(f"{location}: availability_tick debe ser entero")
    elif availability_tick > first_exchange.get("tick", -1):
        errors.append(f"{location}: availability_tick no puede ser posterior al primer dano")
    sources = role.get("source_event_ids")
    if not isinstance(sources, list) or any(source not in event_by_id for source in sources):
        errors.append(f"{location}: source_event_ids no es valido")


def validate_engagement_causal_context(
    context: object,
    causal_participant_ids: set[str],
    state_by_id: Mapping[str, Mapping[str, object]],
    start_tick: int,
    start_sequence: int,
    location: str,
    errors: list[str],
) -> int:
    if not isinstance(context, Mapping):
        errors.append(f"{location}: causal_context debe ser un objeto")
        return 0
    if context.get("t0_tick") != start_tick or context.get("t0_sequence_in_tick") != start_sequence:
        errors.append(f"{location}: causal_context.t0 no coincide con el inicio")
    states = context.get("participant_states")
    if not isinstance(states, list) or not all(isinstance(state, Mapping) for state in states):
        errors.append(f"{location}: participant_states debe ser una lista de objetos")
        return 0
    if {state.get("player_id") for state in states} != causal_participant_ids:
        errors.append(
            f"{location}: participant_states no reconcilia con participantes causales en T0"
        )
    warnings = 0
    for state in states:
        state_location = f"{location}.participant_states[{state.get('player_id')}]"
        status = state.get("status")
        velocity = state.get("horizontal_velocity_world_units_per_second")
        weapon = state.get("active_weapon")
        if status != "observed" or velocity is None or weapon is None:
            warnings += 1
        if status == "unavailable":
            if state.get("state_id") is not None or state.get("availability_tick") is not None:
                errors.append(f"{state_location}: unavailable requiere state_id y tick null")
            if state.get("movement_classification") is not None:
                errors.append(f"{state_location}: no puede clasificar movimiento sin estado")
            continue
        state_id = state.get("state_id")
        source = state_by_id.get(state_id) if isinstance(state_id, str) else None
        if (
            status != "observed"
            or not isinstance(source, Mapping)
            or source.get("player_id") != state.get("player_id")
            or source.get("tick") != state.get("availability_tick")
            or not isinstance(state.get("availability_tick"), int)
            or state.get("availability_tick") > start_tick
        ):
            errors.append(f"{state_location}: referencia un estado ausente o futuro")
        elif any(
            (
                state.get("side") != source.get("side"),
                state.get("position") != source.get("position"),
                state.get("horizontal_velocity_world_units_per_second")
                != source.get("horizontal_velocity_world_units_per_second"),
                state.get("velocity_measurement_window_ticks")
                != source.get("velocity_measurement_window_ticks"),
                state.get("active_weapon") != source.get("active_weapon"),
                state.get("active_weapon_status")
                != source.get("active_weapon_status"),
                state.get("health") != source.get("health"),
                state.get("armor") != source.get("armor"),
                state.get("is_alive") != source.get("is_alive"),
                state.get("objective_phase") != source.get("objective_phase"),
                state.get("round_clock_remaining_ms")
                != source.get("round_clock_remaining_ms"),
                state.get("bomb_time_remaining_ms")
                != source.get("bomb_time_remaining_ms"),
            )
        ):
            errors.append(f"{state_location}: proyeccion no coincide con player_state@3")
        classification = state.get("movement_classification")
        if velocity is None:
            if state.get("velocity_status") != "unavailable" or classification is not None:
                errors.append(f"{state_location}: velocidad no disponible fabrica clasificacion")
        else:
            expected = "peek" if float(velocity) > ENGAGEMENT_PEEK_VELOCITY_THRESHOLD_UPS else "hold"
            if state.get("velocity_status") != "observed" or classification != expected:
                errors.append(f"{state_location}: clasificacion de movimiento no reconcilia")
        if weapon is None and state.get("active_weapon_status") == "observed":
            errors.append(f"{state_location}: arma ausente marcada observed")
    if context.get("initial_distance_world_units") is None:
        warnings += 1
        if context.get("initial_distance_status") != "unavailable":
            errors.append(f"{location}: distancia inicial ausente no marcada unavailable")
    return warnings


def validate_trade_connections(
    candidate: Mapping[str, object],
    state_by_id: Mapping[str, Mapping[str, object]],
    errors: list[str],
    location: str,
) -> tuple[list[str], bool, bool]:
    candidate_id = candidate.get("trade_candidate_id")
    death_tick = candidate.get("death_tick")
    connections = candidate.get("connections")
    if not isinstance(connections, list) or not all(
        isinstance(item, Mapping) for item in connections
    ):
        errors.append(f"{location}: candidato {candidate_id} tiene connections invalido")
        return [], True, False

    required_fields = {
        "teammate_player_id",
        "player_id_usage",
        "player_state_id",
        "state_availability_tick",
        "state_status",
        "alive",
        "alive_status",
        "distance_world_units",
        "distance_status",
        "connection_time_ms",
        "connection_time_status",
        "line_of_sight",
        "line_of_sight_status",
        "facing_delta_deg",
        "facing_status",
        "map_geometry_status",
        "eligible",
        "eligibility_status",
        "ineligibility_reasons",
    }
    seen_players: set[str] = set()
    eligible_players: list[str] = []
    has_unavailable = False
    has_eligible = False
    for connection in connections:
        teammate_id = connection.get("teammate_player_id")
        if required_fields - connection.keys():
            errors.append(
                f"{location}: conexion de {candidate_id} carece de campos fisicos"
            )
        if (
            not isinstance(teammate_id, str)
            or teammate_id in seen_players
            or connection.get("player_id_usage") != "join_only"
        ):
            errors.append(
                f"{location}: conexion de {candidate_id} tiene identidad invalida"
            )
        else:
            seen_players.add(teammate_id)

        state_status = connection.get("state_status")
        state_id = connection.get("player_state_id")
        availability_tick = connection.get("state_availability_tick")
        if state_status == "observed":
            state = state_by_id.get(state_id) if isinstance(state_id, str) else None
            if (
                not isinstance(state, Mapping)
                or state.get("player_id") != teammate_id
                or not isinstance(availability_tick, int)
                or isinstance(availability_tick, bool)
                or availability_tick != state.get("tick")
                or not isinstance(death_tick, int)
                or availability_tick > death_tick
            ):
                errors.append(
                    f"{location}: conexion de {candidate_id} usa estado no causal"
                )
        elif state_status != "unavailable" or state_id is not None or availability_tick is not None:
            errors.append(
                f"{location}: conexion de {candidate_id} inventa estado ausente"
            )

        alive = connection.get("alive")
        alive_status = connection.get("alive_status")
        if (
            (alive_status == "observed") != isinstance(alive, bool)
            and not (alive_status == "unavailable" and alive is None)
        ):
            errors.append(
                f"{location}: conexion de {candidate_id} contradice alive/status"
            )

        numeric_contracts = (
            ("distance_world_units", "distance_status", 0.0, None),
            ("connection_time_ms", "connection_time_status", 0.0, None),
            ("facing_delta_deg", "facing_status", 0.0, 180.0),
        )
        for value_key, status_key, minimum, maximum in numeric_contracts:
            value = connection.get(value_key)
            status = connection.get(status_key)
            if status == "unavailable":
                if value is not None:
                    errors.append(
                        f"{location}: conexion de {candidate_id} inventa {value_key}=0/valor"
                    )
            elif status != "derived" or not is_finite_number(value) or float(value) < minimum or (
                maximum is not None and float(value) > maximum
            ):
                errors.append(
                    f"{location}: conexion de {candidate_id} tiene {value_key}/status invalido"
                )

        line_of_sight = connection.get("line_of_sight")
        line_status = connection.get("line_of_sight_status")
        if line_status == "unavailable":
            if line_of_sight is not None:
                errors.append(
                    f"{location}: conexion de {candidate_id} inventa line_of_sight"
                )
        elif line_status != "derived" or not isinstance(line_of_sight, bool):
            errors.append(
                f"{location}: conexion de {candidate_id} tiene LOS/status invalido"
            )

        map_status = connection.get("map_geometry_status")
        eligibility_status = connection.get("eligibility_status")
        eligible = connection.get("eligible")
        reasons = connection.get("ineligibility_reasons")
        if not isinstance(reasons, list) or not all(
            isinstance(reason, str) and reason for reason in reasons
        ) or len(reasons) != len(set(reasons)):
            errors.append(
                f"{location}: conexion de {candidate_id} tiene reasons invalidos"
            )
            reasons = []

        evidence_complete = all(
            (
                state_status == "observed",
                alive_status == "observed",
                connection.get("distance_status") == "derived",
                is_finite_number(connection.get("distance_world_units")),
                connection.get("connection_time_status") == "derived",
                is_finite_number(connection.get("connection_time_ms")),
                line_status == "derived",
                isinstance(line_of_sight, bool),
                connection.get("facing_status") == "derived",
                is_finite_number(connection.get("facing_delta_deg")),
                map_status == "observed",
            )
        )
        if eligibility_status == "unavailable":
            has_unavailable = True
            if eligible is not None or evidence_complete:
                errors.append(
                    f"{location}: conexion de {candidate_id} contradice abstencion fisica"
                )
        elif eligibility_status == "derived" and isinstance(eligible, bool) and evidence_complete:
            distance = float(connection["distance_world_units"])
            connection_time = float(connection["connection_time_ms"])
            facing = float(connection["facing_delta_deg"])
            expected_eligible = bool(
                alive
                and distance <= TRADE_MAX_DISTANCE_WORLD_UNITS
                and connection_time <= TRADE_WINDOW_MS
                and line_of_sight
                and facing <= TRADE_MAX_FACING_DELTA_DEG
            )
            if not math.isclose(
                connection_time,
                distance * 1_000.0 / TRADE_ASSUMED_MOVEMENT_SPEED_UPS,
                abs_tol=1e-6,
            ) or eligible != expected_eligible:
                errors.append(
                    f"{location}: conexion de {candidate_id} no reconcilia evidencia fisica"
                )
            if eligible:
                eligible_players.append(str(teammate_id))
                has_eligible = True
            elif not reasons:
                errors.append(
                    f"{location}: conexion de {candidate_id} omite motivo de ineligibilidad"
                )
        else:
            errors.append(
                f"{location}: conexion de {candidate_id} tiene eligibility/status invalido"
            )

    return sorted(eligible_players), has_unavailable, has_eligible


def validate_trade_semantics(
    trade_envelope: object,
    combat_events: list[dict],
    state_by_id: Mapping[str, Mapping[str, object]],
    player_stats: list[dict],
    tick_rate: float,
    errors: list[str],
) -> tuple[int, int, int]:
    location = "canonical/derived/trades.json"
    if not isinstance(trade_envelope, Mapping):
        errors.append(f"{location}: raiz invalida")
        return 0, 0, 0
    config = trade_envelope.get("config")
    expected_ticks = engagement_ticks(TRADE_WINDOW_MS, tick_rate)
    if not isinstance(config, Mapping) or any(
        (
            config.get("algorithm_version") != TRADE_ALGORITHM_VERSION,
            config.get("tick_rate_hz") != tick_rate,
            config.get("trade_window_ms") != TRADE_WINDOW_MS,
            config.get("trade_window_ticks") != expected_ticks,
            config.get("max_distance_world_units")
            != TRADE_MAX_DISTANCE_WORLD_UNITS,
            config.get("assumed_movement_speed_world_units_per_second")
            != TRADE_ASSUMED_MOVEMENT_SPEED_UPS,
            config.get("max_facing_delta_deg") != TRADE_MAX_FACING_DELTA_DEG,
            config.get("physical_evidence_requirement")
            != TRADE_PHYSICAL_EVIDENCE_REQUIREMENT,
        )
    ):
        errors.append(f"{location}: config no cumple trade_response@2")
    candidates = trade_envelope.get("trade_candidates")
    completions = trade_envelope.get("trade_completions")
    if not isinstance(candidates, list) or not all(isinstance(item, Mapping) for item in candidates):
        errors.append(f"{location}: trade_candidates debe ser una lista de objetos")
        candidates = []
    if not isinstance(completions, list) or not all(isinstance(item, Mapping) for item in completions):
        errors.append(f"{location}: trade_completions debe ser una lista de objetos")
        completions = []
    event_by_id = {
        event.get("event_id"): event
        for event in combat_events
        if isinstance(event.get("event_id"), str)
    }
    enemy_kills = {
        event_id: event
        for event_id, event in event_by_id.items()
        if event.get("event_type") == "kill"
        and event.get("relation") == "enemy"
        and isinstance(event.get("actor_player_id"), str)
        and isinstance(event.get("target_player_id"), str)
    }
    candidate_by_id: dict[str, Mapping[str, object]] = {}
    candidates_by_kill: dict[str, list[Mapping[str, object]]] = {}
    observation_warnings = 0
    for candidate in candidates:
        candidate_id = candidate.get("trade_candidate_id")
        kill_id = candidate.get("original_kill_event_id")
        if not isinstance(candidate_id, str) or candidate_id in candidate_by_id:
            errors.append(f"{location}: trade_candidate_id ausente o duplicado")
            continue
        candidate_by_id[candidate_id] = candidate
        candidates_by_kill.setdefault(str(kill_id), []).append(candidate)
        kill = enemy_kills.get(kill_id)
        if not isinstance(kill, Mapping) or any(
            (
                candidate.get("round_number") != kill.get("round_number"),
                candidate.get("death_tick") != kill.get("tick"),
                candidate.get("death_sequence_in_tick") != kill.get("sequence_in_tick"),
                candidate.get("original_victim_player_id") != kill.get("target_player_id"),
                candidate.get("original_killer_player_id") != kill.get("actor_player_id"),
                candidate.get("window_ms") != TRADE_WINDOW_MS,
                candidate.get("window_ticks") != expected_ticks,
                candidate.get("window_end_tick") != candidate.get("death_tick", -1) + expected_ticks,
            )
        ):
            errors.append(f"{location}: candidato {candidate_id} no reconcilia con su kill")
        evaluation = candidate.get("evaluation")
        if evaluation not in TRADE_EVALUATIONS:
            errors.append(f"{location}: candidato {candidate_id} tiene evaluation invalida")
        if (evaluation == "completed") != isinstance(candidate.get("trade_completion_id"), str):
            errors.append(f"{location}: candidato {candidate_id} contradice su completion")
        if evaluation == "not_evaluable":
            observation_warnings += 1
        for field in ("eligible_teammate_player_ids", "eligibility_state_ids", "attempt_event_ids", "source_event_ids"):
            value = candidate.get(field)
            if not isinstance(value, list) or len(value) != len(set(value)):
                errors.append(f"{location}: candidato {candidate_id} tiene {field} invalido")
        physically_eligible, has_unavailable, has_eligible = validate_trade_connections(
            candidate, state_by_id, errors, location
        )
        if candidate.get("eligible_teammate_player_ids") != physically_eligible:
            errors.append(
                f"{location}: candidato {candidate_id} no reconcilia conexiones elegibles"
            )
        expected_possible: bool | None
        expected_status: str
        if has_eligible:
            expected_possible, expected_status = True, "derived"
        elif has_unavailable:
            expected_possible, expected_status = None, "unavailable"
        else:
            expected_possible, expected_status = False, "derived"
        if any(
            (
                candidate.get("player_id_usage") != "join_only",
                candidate.get("trade_possible") is not expected_possible,
                candidate.get("trade_possible_status") != expected_status,
                candidate.get("eligibility_status") != expected_status,
                candidate.get("eligibility_source")
                != (
                    "player_state@3+physics_mesh_los"
                    if expected_status == "derived"
                    else "unavailable"
                ),
            )
        ):
            errors.append(
                f"{location}: candidato {candidate_id} contradice trade_possible tri-state"
            )
        if (expected_possible is False and evaluation != "not_tradeable") or (
            expected_status == "unavailable" and evaluation != "not_evaluable"
        ):
            errors.append(
                f"{location}: candidato {candidate_id} contradice tradeability/evaluation"
            )
        eligibility_state_ids = candidate.get("eligibility_state_ids")
        if isinstance(eligibility_state_ids, list) and any(
            state_id not in state_by_id for state_id in eligibility_state_ids
        ):
            errors.append(f"{location}: candidato {candidate_id} referencia estados desconocidos")
    for kill_id in enemy_kills:
        if len(candidates_by_kill.get(kill_id, [])) != 1:
            errors.append(f"{location}: kill {kill_id} no tiene exactamente un candidato")

    completion_by_id: dict[str, Mapping[str, object]] = {}
    used_candidates: set[str] = set()
    used_responses: set[str] = set()
    for completion in completions:
        completion_id = completion.get("trade_completion_id")
        candidate_id = completion.get("trade_candidate_id")
        response_id = completion.get("response_kill_event_id")
        candidate = candidate_by_id.get(candidate_id)
        response = enemy_kills.get(response_id)
        original = enemy_kills.get(completion.get("original_kill_event_id"))
        if not isinstance(completion_id, str) or completion_id in completion_by_id:
            errors.append(f"{location}: trade_completion_id ausente o duplicado")
            continue
        completion_by_id[completion_id] = completion
        if (
            not isinstance(candidate, Mapping)
            or not isinstance(response, Mapping)
            or not isinstance(original, Mapping)
            or completion.get("original_kill_event_id") != candidate.get("original_kill_event_id")
            or engagement_event_key(response) <= engagement_event_key(original)
            or response.get("tick", -1) - candidate.get("death_tick", -1) > expected_ticks
            or response.get("target_player_id") != candidate.get("original_killer_player_id")
            or response.get("actor_player_id") != completion.get("trader_player_id")
            or completion.get("trader_player_id") not in candidate.get("eligible_teammate_player_ids", [])
            or completion.get("elapsed_ticks") != response.get("tick", -1) - candidate.get("death_tick", -1)
            or not math.isclose(
                float(completion.get("elapsed_ms", -1)),
                float(completion.get("elapsed_ticks", -1)) * 1_000.0 / tick_rate,
                abs_tol=1e-6,
            )
        ):
            errors.append(f"{location}: completion {completion_id} no es una respuesta valida")
        if candidate_id in used_candidates or response_id in used_responses:
            errors.append(f"{location}: matching de completions no es uno-a-uno")
        used_candidates.add(str(candidate_id))
        used_responses.add(str(response_id))
    for candidate in candidates:
        completion_id = candidate.get("trade_completion_id")
        if isinstance(completion_id, str) and completion_by_id.get(completion_id, {}).get("trade_candidate_id") != candidate.get("trade_candidate_id"):
            errors.append(f"{location}: candidato referencia completion desconocido")
        counter_id = candidate.get("counter_trade_of_completion_id")
        if isinstance(counter_id, str) and completion_by_id.get(counter_id, {}).get("response_kill_event_id") != candidate.get("original_kill_event_id"):
            errors.append(f"{location}: counter_trade link invalido")

    expected_stats: dict[str, dict[str, int]] = {}
    def player_counts(player_id: str) -> dict[str, int]:
        return expected_stats.setdefault(
            player_id,
            {
                "trade_kills": 0,
                "traded_deaths": 0,
                "trade_attempts": 0,
                "failed_trade_attempts": 0,
                "untradeable_deaths": 0,
                "non_evaluable_trade_deaths": 0,
            },
        )
    completion_by_candidate = {
        completion.get("trade_candidate_id"): completion for completion in completions
    }
    for candidate in candidates:
        victim = str(candidate.get("original_victim_player_id", ""))
        if candidate.get("evaluation") == "not_tradeable":
            player_counts(victim)["untradeable_deaths"] += 1
        if candidate.get("evaluation") == "not_evaluable":
            player_counts(victim)["non_evaluable_trade_deaths"] += 1
        actors = {
            event_by_id.get(event_id, {}).get("actor_player_id")
            for event_id in candidate.get("attempt_event_ids", [])
        }
        completion = completion_by_candidate.get(candidate.get("trade_candidate_id"))
        if isinstance(completion, Mapping):
            actors.add(completion.get("trader_player_id"))
        for actor in actors - {None}:
            player_counts(str(actor))["trade_attempts"] += 1
            if candidate.get("evaluation") == "failed":
                player_counts(str(actor))["failed_trade_attempts"] += 1
    for completion in completions:
        player_counts(str(completion.get("trader_player_id", "")))["trade_kills"] += 1
        player_counts(str(completion.get("original_victim_player_id", "")))["traded_deaths"] += 1
    for player in player_stats:
        player_id = str(player.get("player_id", ""))
        metrics = player.get("metrics")
        if not isinstance(metrics, Mapping):
            errors.append(f"{location}: player_stats sin metrics")
            continue
        wanted = player_counts(player_id)
        if any(metrics.get(field) != count for field, count in wanted.items()):
            errors.append(f"{location}: stats de trade no reconcilian para {player_id}")
    return len(candidates), len(completions), observation_warnings


def validate_engagement_semantics(
    envelope: object,
    records: list[dict],
    trade_envelope: object,
    combat_events: list[dict],
    player_states: list[dict],
    player_stats: list[dict],
    tick_rate: float,
    errors: list[str],
) -> dict[str, int]:
    location = "canonical/derived/engagements.json"
    if not isinstance(envelope, Mapping):
        errors.append(f"{location}: raiz invalida")
        return {"engagements": 0, "trade_candidates": 0, "trade_completions": 0, "observation_warnings": 0}
    config = envelope.get("config")
    expected_config = {
        "algorithm_version": ENGAGEMENT_ALGORITHM_VERSION,
        "tick_rate_hz": tick_rate,
        "pair_continuation_window_ms": ENGAGEMENT_PAIR_WINDOW_MS,
        "pair_continuation_window_ticks": engagement_ticks(ENGAGEMENT_PAIR_WINDOW_MS, tick_rate),
        "multi_target_window_ms": ENGAGEMENT_MULTI_TARGET_WINDOW_MS,
        "multi_target_window_ticks": engagement_ticks(ENGAGEMENT_MULTI_TARGET_WINDOW_MS, tick_rate),
        "max_engagement_duration_ms": ENGAGEMENT_MAX_DURATION_MS,
        "max_engagement_duration_ticks": engagement_ticks(ENGAGEMENT_MAX_DURATION_MS, tick_rate),
        "aggressor_prelude_window_ms": ENGAGEMENT_PRELUDE_WINDOW_MS,
        "aggressor_prelude_window_ticks": engagement_ticks(ENGAGEMENT_PRELUDE_WINDOW_MS, tick_rate),
    }
    if not isinstance(config, Mapping) or any(config.get(field) != value for field, value in expected_config.items()):
        errors.append(f"{location}: config no cumple engagement_causal@2")
    event_by_id = {
        event.get("event_id"): event
        for event in combat_events
        if isinstance(event.get("event_id"), str)
    }
    state_by_id = {
        state.get("state_id"): state
        for state in player_states
        if isinstance(state.get("state_id"), str)
    }
    source_owner: dict[str, str] = {}
    hurt_owner: dict[str, str] = {}
    observation_warnings = 0
    previous_key: tuple[int, int, int, str] | None = None
    for index, record in enumerate(records):
        engagement_id = record.get("engagement_id")
        label = engagement_id if isinstance(engagement_id, str) else str(index)
        item_location = f"{location}: engagement {label}"
        key = (
            int(record.get("round_number", -1)),
            int(record.get("start_tick", -1)),
            int(record.get("start_sequence_in_tick", -1)),
            str(engagement_id),
        )
        if previous_key is not None and key < previous_key:
            errors.append(f"{item_location}: orden no determinista")
        previous_key = key
        start_tick = record.get("start_tick")
        end_tick = record.get("end_tick")
        if (
            not isinstance(start_tick, int)
            or isinstance(start_tick, bool)
            or not isinstance(end_tick, int)
            or isinstance(end_tick, bool)
            or end_tick < start_tick
            or end_tick - start_tick > engagement_ticks(ENGAGEMENT_MAX_DURATION_MS, tick_rate)
            or not is_finite_number(record.get("duration_ms"))
            or not math.isclose(float(record.get("duration_ms", -1)), (end_tick - start_tick) * 1_000.0 / tick_rate, abs_tol=1e-6)
        ):
            errors.append(f"{item_location}: limites o duracion invalidos")
        if record.get("algorithm_version") != ENGAGEMENT_ALGORITHM_VERSION:
            errors.append(f"{item_location}: algorithm_version invalida")
        exchanges = record.get("exchanges")
        participants = record.get("participants")
        if not isinstance(exchanges, list) or not exchanges or not all(isinstance(item, Mapping) for item in exchanges):
            errors.append(f"{item_location}: exchanges debe ser una lista no vacia")
            continue
        if not isinstance(participants, list) or not all(isinstance(item, Mapping) for item in participants):
            errors.append(f"{item_location}: participants debe ser una lista de objetos")
            continue
        participant_ids = [participant.get("player_id") for participant in participants]
        expected_participants = sorted(
            {
                player_id
                for exchange in exchanges
                for player_id in (exchange.get("actor_player_id"), exchange.get("target_player_id"))
                if isinstance(player_id, str)
            }
        )
        if participant_ids != expected_participants:
            errors.append(f"{item_location}: participantes no reconcilian con exchanges")
        participant_set = set(expected_participants)
        expected_sources: set[str] = set()
        previous_exchange_key: tuple[int, int, int, int, str] | None = None
        for exchange in exchanges:
            exchange_id = exchange.get("exchange_id")
            missing = [field for field in ENGAGEMENT_EXCHANGE_FIELDS if field not in exchange]
            if missing:
                errors.append(f"{item_location}: exchange {exchange_id} omite {', '.join(missing)}")
            hurt = event_by_id.get(exchange_id)
            if (
                not isinstance(hurt, Mapping)
                or hurt.get("event_type") != "player_hurt"
                or hurt.get("relation") != "enemy"
                or exchange.get("tick") != hurt.get("tick")
                or exchange.get("sequence_in_tick") != hurt.get("sequence_in_tick")
                or exchange.get("sequence_in_round") != hurt.get("sequence_in_round")
                or exchange.get("actor_player_id") != hurt.get("actor_player_id")
                or exchange.get("target_player_id") != hurt.get("target_player_id")
            ):
                errors.append(f"{item_location}: exchange {exchange_id} no proyecta player_hurt exacto")
                continue
            exchange_key = engagement_event_key(hurt)
            if previous_exchange_key is not None and exchange_key < previous_exchange_key:
                errors.append(f"{item_location}: exchanges fuera de orden atomico")
            previous_exchange_key = exchange_key
            if exchange_id in hurt_owner:
                errors.append(f"{item_location}: player_hurt {exchange_id} reutilizado")
            hurt_owner[str(exchange_id)] = str(engagement_id)
            closure = combat_event_closure(str(exchange_id), event_by_id)
            kill_id = exchange.get("kill_event_id")
            if (exchange.get("is_kill") is True) != isinstance(kill_id, str):
                errors.append(f"{item_location}: exchange {exchange_id} contradice is_kill")
            if isinstance(kill_id, str):
                kill = event_by_id.get(kill_id)
                if not isinstance(kill, Mapping) or kill.get("event_type") != "kill":
                    errors.append(f"{item_location}: kill_event_id no referencia kill")
                closure |= combat_event_closure(kill_id, event_by_id)
            if set(exchange.get("source_event_ids", [])) != closure:
                errors.append(f"{item_location}: exchange {exchange_id} tiene closure incompleto")
            expected_sources |= closure
        first_exchange = exchanges[0]
        for field in ("initiator", "first_aggressor", "first_damage_dealer"):
            validate_engagement_role(record.get(field), participant_set, event_by_id, first_exchange, f"{item_location}.{field}", errors)
        first_damage = record.get("first_damage_dealer")
        if not isinstance(first_damage, Mapping) or first_damage.get("player_id") != first_exchange.get("actor_player_id") or first_damage.get("status") != "observed":
            errors.append(f"{item_location}: first_damage_dealer no coincide con el primer exchange")
        initiator = record.get("initiator")
        if not isinstance(initiator, Mapping) or initiator.get("player_id") is None or initiator.get("source") == "winner":
            errors.append(f"{item_location}: initiator ausente o derivado del resultado")
        first_aggressor = record.get("first_aggressor")
        if isinstance(first_aggressor, Mapping) and first_aggressor.get("status") != "observed":
            observation_warnings += 1
        if isinstance(first_aggressor, Mapping) and first_aggressor.get("status") == "inferred":
            expected_sources |= set(first_aggressor.get("source_event_ids", []))
        actual_sources = record.get("source_event_ids")
        source_events: list[Mapping[str, object]] = []
        if not isinstance(actual_sources, list) or set(actual_sources) != expected_sources:
            errors.append(f"{item_location}: source_event_ids no reconcilia")
        else:
            for source_id in actual_sources:
                if source_id not in event_by_id:
                    errors.append(f"{item_location}: fuente atomica desconocida {source_id}")
                owner = source_owner.get(source_id)
                if owner is not None and owner != engagement_id:
                    errors.append(f"{item_location}: fuente {source_id} reutilizada por {owner}")
                source_owner[source_id] = str(engagement_id)
            source_events = sorted((event_by_id[source_id] for source_id in actual_sources if source_id in event_by_id), key=engagement_event_key)
            if source_events and (
                record.get("start_tick") != source_events[0].get("tick")
                or record.get("start_sequence_in_tick") != source_events[0].get("sequence_in_tick")
                or record.get("end_tick") != source_events[-1].get("tick")
                or record.get("end_sequence_in_tick") != source_events[-1].get("sequence_in_tick")
            ):
                errors.append(f"{item_location}: limites no coinciden con fuentes atomicas")
        outcome = record.get("outcome_context")
        kill_ids = [exchange.get("kill_event_id") for exchange in exchanges if isinstance(exchange.get("kill_event_id"), str)]
        if not isinstance(outcome, Mapping) or outcome.get("outcome") not in ENGAGEMENT_OUTCOMES:
            errors.append(f"{item_location}: outcome_context invalido")
        elif kill_ids:
            winners = {event_by_id[kill_id].get("actor_player_id") for kill_id in kill_ids if kill_id in event_by_id}
            losers = sorted(event_by_id[kill_id].get("target_player_id") for kill_id in kill_ids if kill_id in event_by_id)
            expected_winner = next(iter(winners)) if len(winners) == 1 else None
            if outcome.get("outcome") != "kill" or outcome.get("winner_player_id") != expected_winner or sorted(outcome.get("loser_player_ids", [])) != losers or sorted(outcome.get("terminal_kill_event_ids", [])) != sorted(kill_ids):
                errors.append(f"{item_location}: resultado no reconcilia con kills atomicos")
        elif outcome.get("outcome") != "disengaged" or outcome.get("winner_player_id") is not None or outcome.get("loser_player_ids") != []:
            errors.append(f"{item_location}: disengagement fabrica winner o losers")
        causal_participant_set: set[str] = set()
        if source_events:
            for player_id in (
                source_events[0].get("actor_player_id"),
                source_events[0].get("target_player_id"),
            ):
                if isinstance(player_id, str):
                    causal_participant_set.add(player_id)
        observation_warnings += validate_engagement_causal_context(
            record.get("causal_context"), causal_participant_set, state_by_id,
            int(record.get("start_tick", -1)), int(record.get("start_sequence_in_tick", -1)),
            item_location, errors,
        )
    enemy_hurts = {
        event_id
        for event_id, event in event_by_id.items()
        if event.get("event_type") == "player_hurt"
        and event.get("relation") == "enemy"
        and isinstance(event.get("actor_player_id"), str)
        and isinstance(event.get("target_player_id"), str)
    }
    if set(hurt_owner) != enemy_hurts:
        errors.append(f"{location}: exchanges no son una particion exacta de player_hurt enemigos")
    candidates, completions, trade_warnings = validate_trade_semantics(
        trade_envelope, combat_events, state_by_id, player_stats, tick_rate, errors
    )
    return {
        "engagements": len(records),
        "trade_candidates": candidates,
        "trade_completions": completions,
        "observation_warnings": observation_warnings + trade_warnings,
    }


def read_artifact(
    canonical_dir: Path,
    descriptor: Mapping[str, object],
    match_id: str,
    errors: list[str],
) -> tuple[
    str,
    ArtifactSpec,
    dict | None,
    list[dict],
    StreamedArtifact | None,
] | None:
    resolved = resolve_artifact_path(canonical_dir, descriptor.get("path"), errors)
    if resolved is None:
        return None
    path, file_path = resolved
    spec = get_artifact_spec(descriptor, path, errors)
    if spec is None:
        return None
    if not file_path.is_file():
        errors.append(f"canonical/{path}: fichero declarado inexistente")
        return None
    if descriptor.get("bytes") != file_path.stat().st_size:
        errors.append(f"canonical/{path}: bytes no coincide")
    checksum = descriptor.get("sha256")
    if not isinstance(checksum, str) or SHA256_PATTERN.fullmatch(checksum) is None:
        errors.append(f"canonical/{path}: sha256 no es válido")
    elif checksum != sha256_file(file_path):
        errors.append(f"canonical/{path}: sha256 no coincide")

    envelope = (
        load_json_object(file_path, errors) if spec.file_format == "json" else None
    )
    streamed: StreamedArtifact | None = None
    if spec.artifact_type in STREAMED_JSONL_ARTIFACT_TYPES:
        record_count = validate_streamed_jsonl_artifact(
            file_path, path, spec, descriptor, match_id, errors
        )
        records = []
        streamed = StreamedArtifact(file_path=file_path, record_count=record_count)
    elif spec.file_format == "jsonl":
        records = load_json_lines(file_path, errors)
    elif envelope is None:
        records = []
    elif spec.records_field is None:
        records = [envelope]
    else:
        raw_records = envelope.get(spec.records_field)
        if not isinstance(raw_records, list) or not all(
            isinstance(record, dict) for record in raw_records
        ):
            errors.append(
                f"canonical/{path}: {spec.records_field} debe ser una lista de objetos"
            )
            records = []
        else:
            records = raw_records

    if envelope is not None:
        if envelope.get("schema_id") != spec.schema_id:
            errors.append(f"canonical/{path}: schema_id del contenido no coincide")
        if envelope.get("match_id") != match_id:
            errors.append(f"canonical/{path}: match_id no coincide")
    for index, record in enumerate(
        records if spec.file_format == "jsonl" and streamed is None else []
    ):
        if record.get("schema_id") != spec.schema_id:
            errors.append(
                f"canonical/{path}: registro {index} tiene schema_id incorrecto"
            )
        if record.get("match_id") != match_id:
            errors.append(
                f"canonical/{path}: registro {index} tiene match_id incorrecto"
            )
    if streamed is None:
        if descriptor.get("record_count") != len(records):
            errors.append(f"canonical/{path}: record_count no coincide")
        validate_required_fields(spec, path, records, errors)
    if spec.artifact_type == "utility_events" and descriptor.get("sort_order") != list(
        UTILITY_SORT_ORDER
    ):
        errors.append(
            "canonical/events/utility_events.jsonl: sort_order no cumple utility_event@2"
        )
    if streamed is None:
        validate_sort_order(path, records, descriptor.get("sort_order"), errors)
    return path, spec, envelope, records, streamed


def validate_unique_ids(
    records: list[dict], field: str, location: str, errors: list[str]
) -> set[str]:
    identifiers = [record.get(field) for record in records]
    valid = {value for value in identifiers if isinstance(value, str) and value}
    if len(valid) != len(identifiers):
        errors.append(f"{location}: {field} debe existir y ser único")
    return valid


def collect_references(value: object, suffix: str) -> Iterable[tuple[str, object]]:
    if isinstance(value, Mapping):
        for key, child in value.items():
            if key.endswith(suffix):
                yield key, child
            yield from collect_references(child, suffix)
    elif isinstance(value, list):
        for child in value:
            yield from collect_references(child, suffix)


def validate_record_references(
    artifact_type: str,
    record: Mapping[str, object],
    round_numbers: Mapping[str, int],
    player_ids: set[str],
    event_ids: set[str],
    errors: list[str],
) -> None:
    """Validate the references shared by eager and streamed artifact records."""

    round_id = record.get("round_id")
    if round_id is not None and (
        not isinstance(round_id, str) or round_id not in round_numbers
    ):
        errors.append(f"canonical/{artifact_type}: referencia un round_id desconocido")
    if (
        isinstance(round_id, str)
        and round_id in round_numbers
        and record.get("round_number") != round_numbers[round_id]
    ):
        errors.append(f"canonical/{artifact_type}: round_id y round_number discrepan")
    for field, player_id in collect_references(record, "player_id"):
        if player_id is not None and (
            not isinstance(player_id, str) or player_id not in player_ids
        ):
            errors.append(
                f"canonical/{artifact_type}: {field} referencia un jugador desconocido"
            )
    for field, referenced_ids in collect_references(record, "player_ids"):
        if not isinstance(referenced_ids, list) or any(
            not isinstance(value, str) or value not in player_ids
            for value in referenced_ids
        ):
            errors.append(
                f"canonical/{artifact_type}: {field} contiene jugadores desconocidos"
            )
    source_event_ids = record.get("source_event_ids")
    if source_event_ids is not None and (
        not isinstance(source_event_ids, list)
        or any(
            not isinstance(event_id, str) or event_id not in event_ids
            for event_id in source_event_ids
        )
    ):
        errors.append(
            f"canonical/{artifact_type}: source_event_ids contiene eventos desconocidos"
        )


def iter_records_with_valid_references(
    artifact_type: str,
    rows: Iterable[dict],
    round_numbers: Mapping[str, int],
    player_ids: set[str],
    event_ids: set[str],
    errors: list[str],
) -> Iterable[dict]:
    """Validate one streamed row at a time and immediately release it downstream."""

    for record in rows:
        validate_record_references(
            artifact_type,
            record,
            round_numbers,
            player_ids,
            event_ids,
            errors,
        )
        yield record


def validate_utility_mapping(
    value: object, fields: tuple[str, ...], location: str, errors: list[str]
) -> Mapping[str, object] | None:
    if not isinstance(value, Mapping):
        errors.append(f"{location}: debe ser un objeto")
        return None
    missing = [field for field in fields if field not in value]
    if missing:
        errors.append(f"{location}: faltan campos {', '.join(missing)}")
    unexpected = sorted(set(value) - set(fields))
    if unexpected:
        errors.append(f"{location}: campos no permitidos {', '.join(unexpected)}")
    return value


def validate_utility_observation_header(
    value: object,
    fields: tuple[str, ...],
    location: str,
    errors: list[str],
    *,
    observed_sources: frozenset[str] | None = None,
    unavailable_sources: frozenset[str] = UTILITY_UNAVAILABLE_SOURCES,
    not_applicable_sources: frozenset[str] = frozenset(
        {"unavailable", "not_applicable"}
    ),
) -> tuple[Mapping[str, object], str] | None:
    observation = validate_utility_mapping(
        value, (*fields, "status", "source"), location, errors
    )
    if observation is None:
        return None
    status = observation.get("status")
    source = observation.get("source")
    if status not in UTILITY_OBSERVATION_STATUSES:
        errors.append(f"{location}: status no es valido")
        return None
    if source not in UTILITY_SOURCES:
        errors.append(f"{location}: source no es valido")
    if status == "observed" and (
        source in UTILITY_UNAVAILABLE_SOURCES | {"not_applicable"}
        or (observed_sources is not None and source not in observed_sources)
    ):
        errors.append(f"{location}: source no acredita esta observacion")
    if status == "unavailable" and source not in unavailable_sources:
        errors.append(f"{location}: source no explica la observacion no disponible")
    if status == "not_applicable" and source not in not_applicable_sources:
        errors.append(f"{location}: source no es coherente con not_applicable")
    return observation, str(status)


def validate_utility_vector_observation(
    value: object, location: str, errors: list[str]
) -> tuple[float, float, float] | None:
    result = validate_utility_observation_header(
        value,
        ("value",),
        location,
        errors,
        observed_sources=frozenset({"projectile_position"}),
        unavailable_sources=frozenset({"unavailable"}),
        not_applicable_sources=frozenset({"unavailable"}),
    )
    if result is None:
        return None
    observation, status = result
    vector = finite_vector(observation.get("value"))
    if status == "observed" and vector is None:
        errors.append(f"{location}: value observado debe ser un vector finito")
    if status != "observed" and observation.get("value") is not None:
        errors.append(f"{location}: value debe ser null cuando no esta observado")
    return vector


def validate_utility_view_observation(
    value: object, location: str, errors: list[str]
) -> None:
    result = validate_utility_observation_header(
        value,
        ("yaw_deg", "pitch_deg", "vector"),
        location,
        errors,
        observed_sources=frozenset({"player_view"}),
        unavailable_sources=frozenset({"unavailable"}),
        not_applicable_sources=frozenset({"unavailable"}),
    )
    if result is None:
        return
    observation, status = result
    yaw = observation.get("yaw_deg")
    pitch = observation.get("pitch_deg")
    vector = finite_vector(observation.get("vector"))
    if status != "observed":
        if any(item is not None for item in (yaw, pitch, observation.get("vector"))):
            errors.append(
                f"{location}: angulos y vector deben ser null sin observacion"
            )
        return
    if not is_finite_number(yaw) or not -180 <= float(yaw) <= 180:
        errors.append(f"{location}: yaw_deg observado no es valido")
    if not is_finite_number(pitch) or not -90 <= float(pitch) <= 90:
        errors.append(f"{location}: pitch_deg observado no es valido")
    if vector is None:
        errors.append(f"{location}: vector observado debe ser finito")
        return
    magnitude = math.sqrt(sum(component * component for component in vector))
    if not math.isclose(magnitude, 1.0, rel_tol=0.02, abs_tol=0.02):
        errors.append(f"{location}: vector de vista no es unitario")


def validate_utility_velocity_observation(
    value: object,
    launch_tick: object,
    observed_sources: frozenset[str],
    location: str,
    errors: list[str],
) -> None:
    fields = (
        "vector_world_units_per_second",
        "horizontal_world_units_per_second",
        "observed_tick",
        "measurement_window_ticks",
    )
    result = validate_utility_observation_header(
        value,
        fields,
        location,
        errors,
        observed_sources=observed_sources,
        unavailable_sources=UTILITY_VELOCITY_UNAVAILABLE_SOURCES,
        not_applicable_sources=frozenset({"not_applicable"}),
    )
    if result is None:
        return
    observation, status = result
    vector_value = observation.get("vector_world_units_per_second")
    horizontal = observation.get("horizontal_world_units_per_second")
    observed_tick = observation.get("observed_tick")
    window = observation.get("measurement_window_ticks")
    if status != "observed":
        if any(
            item is not None
            for item in (vector_value, horizontal, observed_tick, window)
        ):
            errors.append(f"{location}: valores deben ser null sin observacion")
        return
    vector = finite_vector(vector_value)
    if vector is None or not is_finite_number(horizontal) or float(horizontal) < 0:
        errors.append(f"{location}: velocidad observada no es valida")
    elif not math.isclose(
        float(horizontal), math.hypot(vector[0], vector[1]), abs_tol=1e-6
    ):
        errors.append(f"{location}: velocidad horizontal no coincide con el vector")
    if not isinstance(observed_tick, int) or isinstance(observed_tick, bool):
        errors.append(f"{location}: observed_tick debe ser entero")
    elif not isinstance(launch_tick, int) or observed_tick != launch_tick:
        errors.append(f"{location}: observed_tick debe coincidir con launch.tick")
    if not isinstance(window, int) or isinstance(window, bool) or window < 0:
        errors.append(f"{location}: measurement_window_ticks debe ser no negativo")
    elif observation.get("source") == "position_delta" and window == 0:
        errors.append(f"{location}: position_delta requiere ventana positiva")
    elif observation.get("source") in {"native", "projectile_velocity"} and window != 0:
        errors.append(f"{location}: source nativo requiere ventana cero")


def validate_utility_string_observation(
    value: object,
    location: str,
    errors: list[str],
    allowed_values: frozenset[str] | None = None,
    observed_sources: frozenset[str] | None = None,
) -> str | None:
    result = validate_utility_observation_header(
        value,
        ("value",),
        location,
        errors,
        observed_sources=observed_sources,
        unavailable_sources=frozenset({"unavailable"}),
        not_applicable_sources=frozenset({"unavailable"}),
    )
    if result is None:
        return None
    observation, status = result
    observed_value = observation.get("value")
    if status != "observed":
        if observed_value is not None:
            errors.append(f"{location}: value debe ser null cuando no esta observado")
        return None
    if not isinstance(observed_value, str) or not observed_value:
        errors.append(f"{location}: value observado debe ser texto no vacio")
        return None
    if allowed_values is not None and observed_value not in allowed_values:
        errors.append(f"{location}: value no es valido")
    return observed_value


def validate_utility_end_reason(
    value: object, location: str, errors: list[str]
) -> str | None:
    result = validate_utility_observation_header(value, ("value",), location, errors)
    if result is None:
        return None
    observation, status = result
    end_reason = observation.get("value")
    if status == "observed":
        if end_reason not in UTILITY_END_REASONS - {"unavailable"}:
            errors.append(f"{location}: value no es un end_reason valido")
            return None
        expected_source = {
            "expired": "expiration_callback",
            "smoke_extinguished": "spatial_smoke_overlap",
            "destroyed": "projectile_destroy",
            "round_end": "round_boundary",
        }[end_reason]
        if observation.get("source") != expected_source:
            errors.append(f"{location}: source no coincide con end_reason")
        return str(end_reason)
    if end_reason not in {None, "unavailable"}:
        errors.append(f"{location}: value no es valido sin observacion")
    return "unavailable" if end_reason == "unavailable" else None


def validate_utility_launch(
    value: object,
    round_record: Mapping[str, object] | None,
    location: str,
    errors: list[str],
) -> int | None:
    launch = validate_utility_mapping(
        value,
        (
            "tick",
            "tick_status",
            "tick_source",
            "position",
            "view",
            "thrower_velocity",
            "projectile_initial_velocity",
            "stance",
            "area",
        ),
        location,
        errors,
    )
    if launch is None:
        return None
    tick = launch.get("tick")
    tick_status = launch.get("tick_status")
    tick_source = launch.get("tick_source")
    if tick_status not in UTILITY_PLAYER_STATUSES:
        errors.append(f"{location}: tick_status no es valido")
    if tick_source not in UTILITY_SOURCES:
        errors.append(f"{location}: tick_source no es valido")
    if tick_status == "observed":
        if not isinstance(tick, int) or isinstance(tick, bool):
            errors.append(f"{location}: tick observado debe ser entero")
            tick = None
        if tick_source != "projectile_throw":
            errors.append(f"{location}: tick observado debe usar projectile_throw")
    else:
        if tick is not None:
            errors.append(f"{location}: tick debe ser null sin observacion")
            tick = None
        if tick_source != "unavailable":
            errors.append(f"{location}: tick no observado debe usar source unavailable")
    if tick is not None and round_record is not None:
        start_tick = round_record.get("start_tick")
        end_tick = round_record.get("end_tick")
        if isinstance(start_tick, int) and tick < start_tick:
            errors.append(f"{location}: tick anterior al inicio de ronda")
        if isinstance(end_tick, int) and tick > end_tick:
            errors.append(f"{location}: tick posterior al fin de ronda")
    validate_utility_vector_observation(
        launch.get("position"), f"{location}.position", errors
    )
    validate_utility_view_observation(launch.get("view"), f"{location}.view", errors)
    validate_utility_velocity_observation(
        launch.get("thrower_velocity"),
        tick,
        UTILITY_THROWER_VELOCITY_SOURCES,
        f"{location}.thrower_velocity",
        errors,
    )
    validate_utility_velocity_observation(
        launch.get("projectile_initial_velocity"),
        tick,
        UTILITY_PROJECTILE_VELOCITY_SOURCES,
        f"{location}.projectile_initial_velocity",
        errors,
    )
    validate_utility_string_observation(
        launch.get("stance"),
        f"{location}.stance",
        errors,
        UTILITY_STANCES,
        frozenset({"player_state"}),
    )
    validate_utility_string_observation(
        launch.get("area"),
        f"{location}.area",
        errors,
        observed_sources=UTILITY_AREA_SOURCES,
    )
    return tick


def validate_utility_trajectory(
    value: object,
    launch_tick: int | None,
    round_record: Mapping[str, object] | None,
    effect_terminal_tick: int | None,
    destroy_tick: int | None,
    destroy_position: tuple[float, float, float] | None,
    location: str,
    errors: list[str],
) -> None:
    fields = (
        "bounce_count",
        "bounce_status",
        "bounce_source",
        "samples",
        "bounces",
        "status",
        "source",
    )
    trajectory = validate_utility_mapping(value, fields, location, errors)
    if trajectory is None:
        return
    status = trajectory.get("status")
    source = trajectory.get("source")
    if status not in UTILITY_TRAJECTORY_STATUSES:
        errors.append(f"{location}: status no es valido")
    if source not in UTILITY_SOURCES:
        errors.append(f"{location}: source no es valido")
    samples = trajectory.get("samples")
    if not isinstance(samples, list):
        errors.append(f"{location}: samples debe ser una lista")
        samples = []
    sample_keys: list[tuple[int, float, float, float, str]] = []
    sample_sources: list[str] = []
    destroy_samples: list[tuple[int, int]] = []
    exact_destroy_observed = destroy_tick is not None and destroy_position is not None
    for index, item in enumerate(samples):
        sample_location = f"{location}.samples[{index}]"
        sample = validate_utility_mapping(
            item, ("tick", "position", "source"), sample_location, errors
        )
        if sample is None:
            continue
        tick = sample.get("tick")
        position = finite_vector(sample.get("position"))
        sample_source = sample.get("source")
        if not isinstance(tick, int) or isinstance(tick, bool):
            errors.append(f"{sample_location}: tick debe ser entero")
            continue
        if position is None:
            errors.append(f"{sample_location}: position debe ser un vector finito")
            continue
        if sample_source not in {"projectile_frames", "projectile_destroy"}:
            errors.append(f"{sample_location}: source no es valido")
        validate_utility_trajectory_tick(
            tick, launch_tick, round_record, None, sample_location, errors
        )
        if sample_source == "projectile_frames":
            flight_terminal_tick = (
                effect_terminal_tick
                if effect_terminal_tick is not None
                else destroy_tick
            )
            if flight_terminal_tick is not None and tick >= flight_terminal_tick:
                errors.append(
                    f"{sample_location}: sample de vuelo no es anterior al terminal"
                )
        elif sample_source == "projectile_destroy" and effect_terminal_tick is not None:
            errors.append(
                f"{sample_location}: destroy sample no aplica tras detonation/effect_start"
            )
        sample_keys.append((tick, *position, str(sample_source)))
        sample_sources.append(str(sample_source))
        if sample_source == "projectile_destroy":
            destroy_samples.append((index, tick))
    if sample_keys != sorted(sample_keys) or len(sample_keys) != len(set(sample_keys)):
        errors.append(f"{location}: samples no tiene orden total determinista")
    has_frames = "projectile_frames" in sample_sources
    if status in {"observed", "partial"} and not samples:
        errors.append(f"{location}: trayectoria disponible sin samples")
    if status == "observed":
        if source != "projectile_frames" or not has_frames:
            errors.append(f"{location}: observed requiere samples de frames")
        if effect_terminal_tick is None:
            if not exact_destroy_observed or len(destroy_samples) != 1:
                errors.append(
                    f"{location}: observed sin detonation/effect_start requiere destroy terminal"
                )
        elif destroy_samples:
            errors.append(
                f"{location}: detonation/effect_start no admite destroy sample"
            )
    elif status == "partial":
        if has_frames:
            if destroy_samples:
                errors.append(
                    f"{location}: frames con destroy terminal debe ser observed"
                )
            if source != "projectile_frames":
                errors.append(
                    f"{location}: partial con frames debe usar projectile_frames"
                )
            if effect_terminal_tick is not None or exact_destroy_observed:
                errors.append(
                    f"{location}: frames con terminal lifecycle debe ser observed"
                )
        elif not has_frames and (
            len(samples) != 1
            or len(destroy_samples) != 1
            or source != "projectile_destroy"
            or effect_terminal_tick is not None
            or not exact_destroy_observed
        ):
            errors.append(f"{location}: partial destroy-only no es coherente")
    elif status == "unavailable":
        if samples:
            errors.append(
                f"{location}: trayectoria no disponible no puede tener samples"
            )
        if source != "unavailable":
            errors.append(
                f"{location}: trayectoria no disponible debe usar unavailable"
            )
        if effect_terminal_tick is None and exact_destroy_observed:
            errors.append(f"{location}: unavailable omite destroy sample observado")
    if effect_terminal_tick is None and len(destroy_samples) > 1:
        errors.append(f"{location}: solo puede existir una muestra destroy")
    if effect_terminal_tick is None and destroy_samples:
        destroy_index, sample_destroy_tick = destroy_samples[0]
        if destroy_index != len(samples) - 1:
            errors.append(f"{location}: muestra destroy debe ser terminal")
        if destroy_tick is None or sample_destroy_tick != destroy_tick:
            errors.append(f"{location}: muestra destroy no reconcilia con lifecycle")
        sample_destroy_position = finite_vector(samples[destroy_index].get("position"))
        if destroy_position is None or sample_destroy_position != destroy_position:
            errors.append(f"{location}: posicion destroy no reconcilia con lifecycle")
    bounces = trajectory.get("bounces")
    if not isinstance(bounces, list):
        errors.append(f"{location}: bounces debe ser una lista")
        bounces = []
    bounce_keys: list[tuple[int, int, str, float, float, float, str]] = []
    bounce_numbers: list[int] = []
    for index, item in enumerate(bounces):
        bounce_location = f"{location}.bounces[{index}]"
        bounce = validate_utility_mapping(
            item,
            ("tick", "position", "position_status", "number", "source"),
            bounce_location,
            errors,
        )
        if bounce is None:
            continue
        tick = bounce.get("tick")
        number = bounce.get("number")
        position = bounce.get("position")
        position_status = bounce.get("position_status")
        bounce_source = bounce.get("source")
        if not isinstance(tick, int) or isinstance(tick, bool):
            errors.append(f"{bounce_location}: tick debe ser entero")
            continue
        if not isinstance(number, int) or isinstance(number, bool) or number <= 0:
            errors.append(f"{bounce_location}: number debe ser entero positivo")
            continue
        if position_status not in UTILITY_PLAYER_STATUSES:
            errors.append(f"{bounce_location}: position_status no es valido")
        vector = finite_vector(position)
        if position_status == "observed" and vector is None:
            errors.append(f"{bounce_location}: position observada no es finita")
        if position_status == "unavailable" and position is not None:
            errors.append(f"{bounce_location}: position debe ser null sin observacion")
        if bounce_source != "projectile_bounce":
            errors.append(f"{bounce_location}: source debe ser projectile_bounce")
        validate_utility_trajectory_tick(
            tick,
            launch_tick,
            round_record,
            effect_terminal_tick if effect_terminal_tick is not None else destroy_tick,
            bounce_location,
            errors,
        )
        comparable_vector = vector if vector is not None else (0.0, 0.0, 0.0)
        bounce_keys.append(
            (
                tick,
                number,
                str(position_status),
                *comparable_vector,
                str(bounce_source),
            )
        )
        bounce_numbers.append(number)
    if bounce_keys != sorted(bounce_keys) or len(bounce_keys) != len(set(bounce_keys)):
        errors.append(f"{location}: bounces no tiene orden total determinista")
    if any(right <= left for left, right in pairwise(bounce_numbers)):
        errors.append(f"{location}: bounce number debe ser estrictamente creciente")
    bounce_status = trajectory.get("bounce_status")
    bounce_source = trajectory.get("bounce_source")
    bounce_count = trajectory.get("bounce_count")
    if bounce_status not in UTILITY_PLAYER_STATUSES:
        errors.append(f"{location}: bounce_status no es valido")
    if bounce_source not in UTILITY_SOURCES:
        errors.append(f"{location}: bounce_source no es valido")
    if bounce_status == "observed":
        if (
            not isinstance(bounce_count, int)
            or isinstance(bounce_count, bool)
            or bounce_count <= 0
        ):
            errors.append(f"{location}: bounce_count observado debe ser positivo")
        if bounce_source != "projectile_bounce":
            errors.append(f"{location}: bounce observado debe usar projectile_bounce")
        if bounce_count != len(bounces):
            errors.append(f"{location}: bounce_count no coincide con bounces")
    else:
        if bounce_count is not None:
            errors.append(f"{location}: bounce_count debe ser null sin observacion")
        if bounce_source != "unavailable":
            errors.append(f"{location}: bounce no observado debe usar unavailable")
        if bounces:
            errors.append(f"{location}: bounce unavailable no admite callbacks")


def validate_utility_trajectory_tick(
    tick: int,
    launch_tick: int | None,
    round_record: Mapping[str, object] | None,
    terminal_tick: int | None,
    location: str,
    errors: list[str],
) -> None:
    if launch_tick is not None and tick < launch_tick:
        errors.append(f"{location}: tick anterior al lanzamiento")
    if terminal_tick is not None and tick >= terminal_tick:
        errors.append(f"{location}: tick no es anterior al terminal de trayectoria")
    if round_record is None:
        return
    start_tick = round_record.get("start_tick")
    end_tick = round_record.get("end_tick")
    if isinstance(start_tick, int) and tick < start_tick:
        errors.append(f"{location}: tick anterior al inicio de ronda")
    if isinstance(end_tick, int) and tick > end_tick:
        errors.append(f"{location}: tick posterior al fin de ronda")


def validate_utility_moment(
    value: object,
    moment_name: str,
    utility_type: object,
    launch_tick: int | None,
    round_record: Mapping[str, object] | None,
    location: str,
    errors: list[str],
    *,
    defer_round_end_check: bool = False,
) -> tuple[int | None, str | None, str | None]:
    observation = validate_utility_mapping(
        value,
        ("tick", "position", "status", "position_status", "source"),
        location,
        errors,
    )
    if observation is None:
        return None, None, None
    status = observation.get("status")
    position_status = observation.get("position_status")
    source = observation.get("source")
    if status not in UTILITY_OBSERVATION_STATUSES:
        errors.append(f"{location}: status no es valido")
    if position_status not in UTILITY_OBSERVATION_STATUSES:
        errors.append(f"{location}: position_status no es valido")
    if source not in UTILITY_SOURCES:
        errors.append(f"{location}: source no es valido")
    tick = observation.get("tick")
    position = observation.get("position")
    if status == "observed":
        if not isinstance(tick, int) or isinstance(tick, bool):
            errors.append(f"{location}: tick observado debe ser entero")
            tick = None
        if source in UTILITY_UNAVAILABLE_SOURCES | {"not_applicable"}:
            errors.append(f"{location}: source no acredita el tick observado")
        expected_sources = UTILITY_LIFECYCLE_MOMENT_SOURCES.get(moment_name, {}).get(
            utility_type, frozenset()
        )
        if source not in expected_sources:
            errors.append(f"{location}: source no corresponde al callback causal")
    elif tick is not None:
        errors.append(f"{location}: tick debe ser null sin observacion")
        tick = None
    if position_status == "observed":
        if finite_vector(position) is None:
            errors.append(f"{location}: position observada debe ser un vector finito")
    elif position is not None:
        errors.append(f"{location}: position debe ser null sin observacion")
    if status != "observed":
        if position_status == "observed":
            errors.append(f"{location}: position no puede observarse sin tick causal")
        if source != "unavailable":
            errors.append(
                f"{location}: moment no disponible debe usar source unavailable"
            )
    if tick is not None and launch_tick is not None and tick < launch_tick:
        errors.append(f"{location}: tick anterior al lanzamiento")
    if tick is not None and round_record is not None:
        start_tick = round_record.get("start_tick")
        end_tick = round_record.get("end_tick")
        if isinstance(start_tick, int) and tick < start_tick:
            errors.append(f"{location}: tick anterior al inicio de ronda")
        if (
            isinstance(end_tick, int)
            and tick > end_tick
            and not defer_round_end_check
        ):
            errors.append(f"{location}: tick posterior al fin de ronda")
    return (
        tick,
        str(status) if isinstance(status, str) else None,
        str(position_status) if isinstance(position_status, str) else None,
    )


def validate_utility_duration(
    value: object,
    location: str,
    errors: list[str],
    observed_sources: frozenset[str],
) -> tuple[float | None, str | None]:
    result = validate_utility_observation_header(
        value,
        ("milliseconds",),
        location,
        errors,
        observed_sources=observed_sources,
        unavailable_sources=frozenset({"unavailable"}),
        not_applicable_sources=frozenset({"unavailable"}),
    )
    if result is None:
        return None, None
    observation, status = result
    milliseconds = observation.get("milliseconds")
    if status == "observed":
        if not is_finite_number(milliseconds) or float(milliseconds) < 0:
            errors.append(
                f"{location}: milliseconds observado debe ser no negativo y finito"
            )
            return None, status
        return float(milliseconds), status
    if milliseconds is not None:
        errors.append(f"{location}: milliseconds debe ser null sin observacion")
    return None, status


def validate_utility_correlation(
    value: object,
    location: str,
    errors: list[str],
    observed_sources: frozenset[str] = frozenset(
        {"projectile_entity", "grenade_entity_id", "effect_entity_id"}
    ),
    inferred_sources: frozenset[str] = frozenset(
        {"thrower_type_position_tick", "type_position_tick"}
    ),
) -> tuple[str | None, str | None]:
    correlation = validate_utility_mapping(
        value, ("status", "source"), location, errors
    )
    if correlation is None:
        return None, None
    status = correlation.get("status")
    source = correlation.get("source")
    if status not in UTILITY_CORRELATION_STATUSES:
        errors.append(f"{location}: status no es valido")
    if source not in UTILITY_CORRELATION_SOURCES | inferred_sources:
        errors.append(f"{location}: source no es valido")
    expected_sources = {
        "observed": observed_sources,
        "inferred": inferred_sources,
        "unavailable": {"unavailable"},
    }
    if status in expected_sources and source not in expected_sources[status]:
        errors.append(f"{location}: status y source son incoherentes")
    return str(status) if isinstance(status, str) else None, str(source) if isinstance(
        source, str
    ) else None


def validate_utility_lifecycle(
    value: object,
    utility_type: object,
    launch_tick: int | None,
    round_record: Mapping[str, object] | None,
    tick_rate: float,
    location: str,
    errors: list[str],
) -> dict[str, object]:
    fields = (
        "status",
        "detonation",
        "effect_start",
        "expiration",
        "destroy",
        "extinguish",
        "duration",
        "area",
        "end_reason",
        "extinguished_by_throw_id",
        "extinguish_correlation",
    )
    lifecycle = validate_utility_mapping(value, fields, location, errors)
    if lifecycle is None:
        return {}
    status = lifecycle.get("status")
    if status not in UTILITY_LIFECYCLE_STATUSES:
        errors.append(f"{location}: status no es valido")
    moments: dict[str, tuple[int | None, str | None, str | None]] = {}
    terminal_moments = frozenset({"expiration", "destroy", "extinguish"})
    for name in ("detonation", "effect_start", "expiration", "destroy", "extinguish"):
        moments[name] = validate_utility_moment(
            lifecycle.get(name),
            name,
            utility_type,
            launch_tick,
            round_record,
            f"{location}.{name}",
            errors,
            defer_round_end_check=name in terminal_moments,
        )
    duration_ms, duration_status = validate_utility_duration(
        lifecycle.get("duration"),
        f"{location}.duration",
        errors,
        frozenset({"callback_ticks"}),
    )
    validate_utility_string_observation(
        lifecycle.get("area"),
        f"{location}.area",
        errors,
        observed_sources=UTILITY_AREA_SOURCES,
    )
    end_reason = validate_utility_end_reason(
        lifecycle.get("end_reason"), f"{location}.end_reason", errors
    )
    extinguished_by = validate_utility_string_observation(
        lifecycle.get("extinguished_by_throw_id"),
        f"{location}.extinguished_by_throw_id",
        errors,
        observed_sources=frozenset({"spatial_smoke_overlap"}),
    )
    extinguish_correlation_status, _ = validate_utility_correlation(
        lifecycle.get("extinguish_correlation"),
        f"{location}.extinguish_correlation",
        errors,
        inferred_sources=frozenset(
            {
                "thrower_type_position_tick",
                "type_position_tick",
                "spatial_smoke_overlap",
            }
        ),
    )
    detonation_tick = moments["detonation"][0]
    effect_start_tick = moments["effect_start"][0]
    expiration_tick = moments["expiration"][0]
    if (
        detonation_tick is not None
        and effect_start_tick is not None
        and effect_start_tick < detonation_tick
    ):
        errors.append(f"{location}: effect_start es anterior a detonation")
    lifecycle_start_tick = (
        effect_start_tick if effect_start_tick is not None else detonation_tick
    )
    lifecycle_start_name = (
        "effect_start" if effect_start_tick is not None else "detonation"
    )
    lifecycle_order_valid = True
    if lifecycle_start_tick is not None:
        for name in terminal_moments:
            terminal_tick = moments[name][0]
            if terminal_tick is not None and terminal_tick < lifecycle_start_tick:
                lifecycle_order_valid = False
                errors.append(
                    f"{location}: {name} es anterior a {lifecycle_start_name}"
                )
    if not lifecycle_order_valid and round_record is not None:
        end_tick = round_record.get("end_tick")
        if isinstance(end_tick, int):
            for name in terminal_moments:
                terminal_tick = moments[name][0]
                if terminal_tick is not None and terminal_tick > end_tick:
                    errors.append(
                        f"{location}.{name}: tick posterior al fin de ronda "
                        "con lifecycle incoherente"
                    )
    if status == "thrown" and (
        detonation_tick is not None or effect_start_tick is not None
    ):
        errors.append(f"{location}: status thrown no admite inicio de efecto")
    if status == "detonated" and detonation_tick is None:
        errors.append(f"{location}: status detonated requiere detonation observada")
    if status == "effect_active" and effect_start_tick is None:
        errors.append(f"{location}: status {status} requiere effect_start observado")
    if status == "effect_expired" and expiration_tick is None:
        errors.append(
            f"{location}: status effect_expired requiere expiration observada"
        )
    if status == "destroyed_without_detonation" and (
        moments["destroy"][0] is None or detonation_tick is not None
    ):
        errors.append(f"{location}: destroyed_without_detonation es incoherente")
    if status == "round_ended_unresolved" and end_reason != "round_end":
        errors.append(
            f"{location}: round_ended_unresolved requiere end_reason round_end"
        )
    instant_types = {"flashbang", "he"}
    persistent_types = {"smoke", "decoy", "molotov", "incendiary"}
    lifecycle_exceptions = {
        "destroyed_without_detonation",
        "round_ended_unresolved",
    }
    if (
        utility_type in instant_types
        and status not in {"detonated"} | lifecycle_exceptions
    ):
        errors.append(f"{location}: utilidad instantanea tiene lifecycle incoherente")
    if utility_type in instant_types and (
        effect_start_tick is not None
        or expiration_tick is not None
        or duration_status == "observed"
    ):
        errors.append(f"{location}: utilidad instantanea no tiene efecto persistente")
    if utility_type not in instant_types and status == "detonated":
        errors.append(
            f"{location}: utilidad persistente no puede terminar en detonated"
        )
    extinguish_status = moments["extinguish"][1]
    extinguished_by_value = lifecycle.get("extinguished_by_throw_id")
    extinguished_by_status = (
        extinguished_by_value.get("status")
        if isinstance(extinguished_by_value, Mapping)
        else None
    )
    if end_reason == "smoke_extinguished":
        extinguish_tick = moments["extinguish"][0]
        if (
            utility_type not in {"molotov", "incendiary", "unknown"}
            or status != "effect_expired"
        ):
            errors.append(
                f"{location}: smoke_extinguished solo aplica a fuego expirado"
            )
        if (
            expiration_tick is None
            or extinguish_tick is None
            or extinguished_by is None
            or extinguished_by_status != "observed"
            or extinguish_correlation_status not in {"observed", "inferred"}
        ):
            errors.append(
                f"{location}: extincion por smoke no tiene atribucion completa"
            )
        elif not 0 <= expiration_tick - extinguish_tick <= 16:
            errors.append(
                f"{location}: extincion y expiration no son temporalmente proximas"
            )
    elif (
        extinguish_status == "observed"
        or extinguished_by is not None
        or extinguished_by_status == "observed"
        or extinguish_correlation_status in {"observed", "inferred"}
    ):
        errors.append(
            f"{location}: campos de extincion presentes sin smoke_extinguished"
        )
    if end_reason == "expired" and (
        utility_type not in persistent_types | {"unknown"}
        or status != "effect_expired"
        or expiration_tick is None
    ):
        errors.append(f"{location}: end_reason expired no coincide con lifecycle")
    if end_reason == "destroyed" and (
        status != "destroyed_without_detonation" or moments["destroy"][0] is None
    ):
        errors.append(f"{location}: end_reason destroyed no coincide con lifecycle")
    if end_reason == "round_end" and status != "round_ended_unresolved":
        errors.append(f"{location}: end_reason round_end no coincide con lifecycle")
    if (
        duration_ms is not None
        and effect_start_tick is not None
        and expiration_tick is not None
        and tick_rate > 0
    ):
        expected_ms = (expiration_tick - effect_start_tick) * 1000 / tick_rate
        if not math.isclose(duration_ms, expected_ms, abs_tol=1e-6):
            errors.append(f"{location}: duration no coincide con ticks callback")
    if duration_status == "observed" and (
        effect_start_tick is None or expiration_tick is None
    ):
        errors.append(f"{location}: duration observada requiere ambos ticks callback")
    return {
        "status": status,
        "end_reason": end_reason,
        "extinguished_by": extinguished_by,
        **{f"{name}_tick": item[0] for name, item in moments.items()},
    }


def validate_utility_source_entity(
    record: Mapping[str, object], location: str, errors: list[str]
) -> tuple[int, int, int] | None:
    status = record.get("source_entity_status")
    source = record.get("source_entity_source")
    entity = record.get("source_entity")
    if status not in UTILITY_PLAYER_STATUSES:
        errors.append(f"{location}: source_entity_status no es valido")
    if source not in {"projectile_entity", "grenade_entity_id", "unavailable"}:
        errors.append(f"{location}: source_entity_source no es valido")
    if status != "observed":
        if entity is not None:
            errors.append(f"{location}: source_entity debe ser null sin observacion")
        if source != "unavailable":
            errors.append(
                f"{location}: source_entity no observado debe usar source unavailable"
            )
        return None
    if source not in {"projectile_entity", "grenade_entity_id"}:
        errors.append(f"{location}: source_entity observado tiene source incoherente")
    entity_record = validate_utility_mapping(
        entity,
        ("round_number", "entity_id", "generation"),
        f"{location}.source_entity",
        errors,
    )
    if entity_record is None:
        return None
    values = tuple(
        entity_record.get(field)
        for field in ("round_number", "entity_id", "generation")
    )
    if any(not isinstance(item, int) or isinstance(item, bool) for item in values):
        errors.append(f"{location}.source_entity: sus campos deben ser enteros")
        return None
    round_number, entity_id, generation = values
    if round_number != record.get("round_number") or entity_id <= 0 or generation <= 0:
        errors.append(f"{location}.source_entity: identidad scoped no es valida")
    return int(round_number), int(entity_id), int(generation)


def validate_utility_damage_event(
    value: object,
    launch_tick: int | None,
    round_record: Mapping[str, object] | None,
    location: str,
    errors: list[str],
) -> tuple[int, int, int, bool] | None:
    event = validate_utility_mapping(
        value,
        (
            "tick",
            "health_damage",
            "armor_damage",
            "is_kill",
            "source",
            "correlation",
        ),
        location,
        errors,
    )
    if event is None:
        return None
    tick = event.get("tick")
    health_damage = event.get("health_damage")
    armor_damage = event.get("armor_damage")
    is_kill = event.get("is_kill")
    if not isinstance(tick, int) or isinstance(tick, bool):
        errors.append(f"{location}: tick debe ser entero")
        return None
    if launch_tick is not None and tick < launch_tick:
        errors.append(f"{location}: tick anterior al lanzamiento")
    if (
        round_record is not None
        and isinstance(round_record.get("start_tick"), int)
        and tick < round_record["start_tick"]
    ):
        errors.append(f"{location}: tick anterior al inicio de ronda")
    if (
        round_record is not None
        and isinstance(round_record.get("end_tick"), int)
        and tick > round_record["end_tick"]
    ):
        errors.append(f"{location}: tick posterior al fin de ronda")
    if any(
        not isinstance(item, int) or isinstance(item, bool) or item < 0
        for item in (health_damage, armor_damage)
    ):
        errors.append(f"{location}: danos deben ser enteros no negativos")
        return None
    if not isinstance(is_kill, bool):
        errors.append(f"{location}: is_kill debe ser booleano")
        return None
    if event.get("source") != "player_hurt":
        errors.append(f"{location}: source debe ser player_hurt")
    validate_utility_correlation(
        event.get("correlation"),
        f"{location}.correlation",
        errors,
        observed_sources=frozenset(),
        inferred_sources=frozenset({"thrower_type_position_tick"}),
    )
    return tick, int(health_damage), int(armor_damage), is_kill


def expected_utility_relation(
    thrower_player_id: object,
    thrower_side: object,
    player_id: object,
    player_side: object,
) -> str:
    if not all(
        isinstance(item, str) and item for item in (thrower_player_id, player_id)
    ):
        return "unknown"
    if thrower_player_id == player_id:
        return "self"
    if thrower_side in {"ct", "t"} and player_side in {"ct", "t"}:
        return "teammate" if thrower_side == player_side else "enemy"
    return "unknown"


def validate_utility_affected_player(
    value: object,
    utility_type: object,
    thrower_player_id: object,
    thrower_side: object,
    expected_player_sides: Mapping[str, str],
    launch_tick: int | None,
    round_record: Mapping[str, object] | None,
    location: str,
    errors: list[str],
) -> dict[str, object] | None:
    fields = (
        "player_id",
        "side",
        "player_status",
        "player_source",
        "relation",
        "is_enemy",
        "is_self",
        "blind_duration",
        "blind_correlation",
        "damage",
        "armor_damage",
        "is_kill",
        "damage_events",
    )
    player = validate_utility_mapping(value, fields, location, errors)
    if player is None:
        return None
    player_status = player.get("player_status")
    player_source = player.get("player_source")
    player_id = player.get("player_id")
    side = player.get("side")
    if player_status not in UTILITY_PLAYER_STATUSES:
        errors.append(f"{location}: player_status no es valido")
    if player_source not in UTILITY_PLAYER_SOURCES:
        errors.append(f"{location}: player_source no es valido")
    if player_status == "observed":
        if player_source == "unavailable":
            errors.append(f"{location}: jugador observado no puede usar unavailable")
        if (
            not isinstance(player_id, str)
            or PLAYER_ID_PATTERN.fullmatch(player_id) is None
        ):
            errors.append(f"{location}: player_id observado no es valido")
        if side is not None and side not in {"ct", "t"}:
            errors.append(f"{location}: side observado no es valido")
        expected_side = expected_player_sides.get(player_id)
        if expected_side is not None and side is not None and side != expected_side:
            errors.append(f"{location}: side no coincide con side_assignments")
    else:
        if player_id is not None or side is not None:
            errors.append(
                f"{location}: jugador no disponible debe usar referencias null"
            )
    relation = player.get("relation")
    if relation not in UTILITY_RELATIONS:
        errors.append(f"{location}: relation no es valida")
    expected_relation = expected_utility_relation(
        thrower_player_id, thrower_side, player_id, side
    )
    if relation != expected_relation:
        errors.append(f"{location}: relation no coincide con thrower y victima")
    if player.get("is_enemy") is not (relation == "enemy"):
        errors.append(f"{location}: is_enemy no coincide con relation")
    if player.get("is_self") is not (relation == "self"):
        errors.append(f"{location}: is_self no coincide con relation")
    blind_ms, blind_status = validate_utility_duration(
        player.get("blind_duration"),
        f"{location}.blind_duration",
        errors,
        frozenset({"player_flashed"}),
    )
    blind_correlation_status, _ = validate_utility_correlation(
        player.get("blind_correlation"),
        f"{location}.blind_correlation",
        errors,
        observed_sources=frozenset({"projectile_entity"}),
    )
    has_flash_effect = blind_status in {"observed", "unavailable"}
    if utility_type == "flashbang" and not has_flash_effect:
        errors.append(f"{location}: una victima flash requiere blind observation")
    if utility_type != "flashbang" and blind_status != "not_applicable":
        errors.append(f"{location}: solo flashbang puede atribuir cegado")
    if not has_flash_effect and blind_correlation_status != "unavailable":
        errors.append(f"{location}: blind_correlation existe sin efecto flash")
    damage_events = player.get("damage_events")
    if not isinstance(damage_events, list):
        errors.append(f"{location}: damage_events debe ser una lista")
        damage_events = []
    validated_damage = [
        result
        for index, event in enumerate(damage_events)
        if (
            result := validate_utility_damage_event(
                event,
                launch_tick,
                round_record,
                f"{location}.damage_events[{index}]",
                errors,
            )
        )
        is not None
    ]
    damage_order = [
        (
            item[0],
            item[1],
            item[2],
            bool(item[3]),
        )
        for item in validated_damage
    ]
    if damage_order != sorted(damage_order):
        errors.append(f"{location}: damage_events no tiene orden determinista")
    if len(damage_order) != len(set(damage_order)):
        errors.append(f"{location}: damage_events contiene callbacks duplicados")
    health_damage = sum(item[1] for item in validated_damage)
    armor_damage = sum(item[2] for item in validated_damage)
    is_kill = any(item[3] for item in validated_damage)
    expected_player_source = (
        "player_hurt"
        if damage_events
        else "player_flashed"
        if has_flash_effect
        else "unavailable"
    )
    if player_source != expected_player_source:
        errors.append(
            f"{location}: player_source no coincide con los callbacks de efecto"
        )
    if damage_events:
        if utility_type not in {"he", "molotov", "incendiary", "unknown"}:
            errors.append(f"{location}: utility_type no puede causar dano")
        if (
            player.get("damage") != health_damage
            or player.get("armor_damage") != armor_damage
        ):
            errors.append(f"{location}: danos agregados no coinciden con damage_events")
        if player.get("is_kill") is not is_kill:
            errors.append(f"{location}: is_kill no coincide con damage_events")
    elif any(
        player.get(field) is not None for field in ("damage", "armor_damage", "is_kill")
    ):
        errors.append(f"{location}: danos agregados deben ser null sin damage_events")
    if not has_flash_effect and not damage_events:
        errors.append(f"{location}: affected_player no contiene un efecto observado")
    return {
        "player_id": player_id,
        "relation": relation,
        "has_flash_effect": has_flash_effect,
        "blind_duration_ms": blind_ms,
        "blind_duration_status": blind_status,
        "health_damage": health_damage,
        "armor_damage": armor_damage,
        "is_kill": is_kill,
        "has_positive_damage": health_damage > 0 or armor_damage > 0,
    }


def validate_utility_flash_summary(
    value: object, affected: list[dict[str, object]], location: str, errors: list[str]
) -> None:
    count_fields = (
        "players_total",
        "enemies_flashed",
        "teammates_flashed",
        "self_flashed",
        "unknown_flashed",
    )
    duration_fields = (
        "total_duration_ms",
        "enemy_duration_ms",
        "teammate_duration_ms",
        "self_duration_ms",
        "unknown_duration_ms",
    )
    summary = validate_utility_mapping(
        value, (*count_fields, *duration_fields), location, errors
    )
    if summary is None:
        return
    for field in count_fields:
        item = summary.get(field)
        if not isinstance(item, int) or isinstance(item, bool) or item < 0:
            errors.append(f"{location}: {field} debe ser entero no negativo")
    flash_effects = [item for item in affected if item.get("has_flash_effect") is True]
    by_relation = {
        relation: [item for item in flash_effects if item.get("relation") == relation]
        for relation in ("enemy", "teammate", "self", "unknown")
    }
    expected_counts = {
        "players_total": len(flash_effects),
        "enemies_flashed": len(by_relation["enemy"]),
        "teammates_flashed": len(by_relation["teammate"]),
        "self_flashed": len(by_relation["self"]),
        "unknown_flashed": len(by_relation["unknown"]),
    }
    for field, expected in expected_counts.items():
        if summary.get(field) != expected:
            errors.append(f"{location}: {field} no coincide con affected_players")

    def duration_sum(items: list[dict[str, object]]) -> float | None:
        if not items:
            return 0.0
        if any(item.get("blind_duration_status") != "observed" for item in items):
            return None
        return sum(float(item["blind_duration_ms"]) for item in items)

    expected_durations = {
        "total_duration_ms": duration_sum(flash_effects),
        "enemy_duration_ms": duration_sum(by_relation["enemy"]),
        "teammate_duration_ms": duration_sum(by_relation["teammate"]),
        "self_duration_ms": duration_sum(by_relation["self"]),
        "unknown_duration_ms": duration_sum(by_relation["unknown"]),
    }
    for field, expected in expected_durations.items():
        actual = summary.get(field)
        if expected is None:
            if actual is not None:
                errors.append(f"{location}: {field} debe ser null sin cobertura")
        elif (
            not is_finite_number(actual)
            or float(actual) < 0
            or not math.isclose(float(actual), expected, abs_tol=1e-6)
        ):
            errors.append(f"{location}: {field} no coincide con affected_players")


def validate_utility_damage_summary(
    value: object, affected: list[dict[str, object]], location: str, errors: list[str]
) -> None:
    integer_fields = (
        "total_damage",
        "enemy_damage",
        "teammate_damage",
        "self_damage",
        "unknown_damage",
        "total_armor_damage",
        "enemy_armor_damage",
        "teammate_armor_damage",
        "self_armor_damage",
        "unknown_armor_damage",
        "enemies_damaged",
        "teammates_damaged",
        "unknown_players_damaged",
        "enemy_kills",
        "teammate_kills",
        "self_kills",
        "unknown_kills",
    )
    summary = validate_utility_mapping(
        value, (*integer_fields, "self_damaged"), location, errors
    )
    if summary is None:
        return
    for field in integer_fields:
        item = summary.get(field)
        if not isinstance(item, int) or isinstance(item, bool) or item < 0:
            errors.append(f"{location}: {field} debe ser entero no negativo")
    if not isinstance(summary.get("self_damaged"), bool):
        errors.append(f"{location}: self_damaged debe ser booleano")
    health_by_relation = {
        relation: sum(
            int(item["health_damage"])
            for item in affected
            if item.get("relation") == relation
        )
        for relation in ("enemy", "teammate", "self", "unknown")
    }
    armor_by_relation = {
        relation: sum(
            int(item["armor_damage"])
            for item in affected
            if item.get("relation") == relation
        )
        for relation in ("enemy", "teammate", "self", "unknown")
    }
    expected = {
        "total_damage": sum(health_by_relation.values()),
        "enemy_damage": health_by_relation["enemy"],
        "teammate_damage": health_by_relation["teammate"],
        "self_damage": health_by_relation["self"],
        "unknown_damage": health_by_relation["unknown"],
        "total_armor_damage": sum(armor_by_relation.values()),
        "enemy_armor_damage": armor_by_relation["enemy"],
        "teammate_armor_damage": armor_by_relation["teammate"],
        "self_armor_damage": armor_by_relation["self"],
        "unknown_armor_damage": armor_by_relation["unknown"],
        "enemies_damaged": sum(
            item.get("relation") == "enemy" and item["has_positive_damage"] is True
            for item in affected
        ),
        "teammates_damaged": sum(
            item.get("relation") == "teammate" and item["has_positive_damage"] is True
            for item in affected
        ),
        "unknown_players_damaged": sum(
            item.get("relation") == "unknown" and item["has_positive_damage"] is True
            for item in affected
        ),
        "self_damaged": any(
            item.get("relation") == "self" and item["has_positive_damage"] is True
            for item in affected
        ),
        "enemy_kills": sum(
            item.get("relation") == "enemy" and item.get("is_kill") is True
            for item in affected
        ),
        "teammate_kills": sum(
            item.get("relation") == "teammate" and item.get("is_kill") is True
            for item in affected
        ),
        "self_kills": sum(
            item.get("relation") == "self" and item.get("is_kill") is True
            for item in affected
        ),
        "unknown_kills": sum(
            item.get("relation") == "unknown" and item.get("is_kill") is True
            for item in affected
        ),
    }
    for field, expected_value in expected.items():
        if summary.get(field) != expected_value:
            errors.append(f"{location}: {field} no coincide con affected_players")


def build_player_sides_by_round(
    participants: list[dict], rounds_by_number: Mapping[int, Mapping[str, object]]
) -> dict[int, dict[str, str]]:
    player_teams = {
        record.get("player_id"): record.get("team_id")
        for record in participants
        if isinstance(record.get("player_id"), str)
        and isinstance(record.get("team_id"), str)
    }
    result: dict[int, dict[str, str]] = {}
    for round_number, round_record in rounds_by_number.items():
        assignments = round_record.get("side_assignments")
        side_by_team = (
            {
                item.get("team_id"): item.get("side")
                for item in assignments
                if isinstance(item, Mapping)
            }
            if isinstance(assignments, list)
            else {}
        )
        result[round_number] = {
            player_id: side_by_team[team_id]
            for player_id, team_id in player_teams.items()
            if side_by_team.get(team_id) in {"ct", "t"}
        }
    return result


def validate_utility_semantics(
    match_id: str,
    match_envelope: Mapping[str, object],
    participants: list[dict],
    rounds: list[dict],
    records: list[dict],
    errors: list[str],
) -> None:
    rounds_by_number = {
        record.get("round_number"): record
        for record in rounds
        if isinstance(record.get("round_number"), int)
    }
    sides_by_round = build_player_sides_by_round(participants, rounds_by_number)
    tick_rate_value = match_envelope.get("tick_rate_hz")
    tick_rate = (
        float(tick_rate_value)
        if is_finite_number(tick_rate_value) and float(tick_rate_value) > 0
        else 0
    )
    source_entities: dict[tuple[int, int, int], str] = {}
    source_throws: dict[str, dict] = {}
    last_sequence_by_round: dict[int, int] = {}
    previous_sort_key: tuple[int, int, str] | None = None
    last_causal_tick_by_round: dict[int, int] = {}
    for index, record in enumerate(records):
        location = f"canonical/events/utility_events.jsonl: registro {index}"
        allowed_fields = {
            "schema_id",
            "match_id",
            *REQUIRED_RECORD_FIELDS["utility_events"],
        }
        unexpected_fields = sorted(set(record) - allowed_fields)
        if unexpected_fields:
            errors.append(
                f"{location}: campos no permitidos {', '.join(unexpected_fields)}"
            )
        round_number = record.get("round_number")
        sequence = record.get("sequence_in_round")
        source_throw_id = record.get("source_throw_id")
        launch = record.get("launch")
        if record.get("event_type") not in UTILITY_EVENT_TYPES:
            errors.append(f"{location}: event_type debe ser utility_throw")
        if record.get("utility_type") not in UTILITY_TYPES:
            errors.append(f"{location}: utility_type no es valido")
        utility_type_status = record.get("utility_type_status")
        if utility_type_status not in UTILITY_PLAYER_STATUSES:
            errors.append(f"{location}: utility_type_status no es valido")
        if record.get("utility_type_source") not in UTILITY_TYPE_SOURCES:
            errors.append(f"{location}: utility_type_source no es valido")
        if record.get("utility_type") == "unknown" and (
            utility_type_status != "unavailable"
            or record.get("utility_type_source") != "unavailable"
        ):
            errors.append(f"{location}: utility_type unknown debe declarar unavailable")
        if record.get("utility_type") in UTILITY_TYPES - {"unknown"} and record.get(
            "utility_type_source"
        ) not in {"weapon_instance", "callback_type"}:
            errors.append(f"{location}: utility_type observado no tiene provenance")
        if (
            record.get("utility_type") in UTILITY_TYPES - {"unknown"}
            and utility_type_status != "observed"
        ):
            errors.append(f"{location}: utility_type conocido debe estar observed")
        if not isinstance(sequence, int) or isinstance(sequence, bool) or sequence <= 0:
            errors.append(f"{location}: sequence_in_round debe ser positivo")
        elif isinstance(round_number, int):
            expected_sequence = last_sequence_by_round.get(round_number, 0) + 1
            if sequence != expected_sequence:
                errors.append(
                    f"{location}: sequence_in_round debe ser contiguo desde uno"
                )
            last_sequence_by_round[round_number] = sequence
        expected_throw_id = (
            f"r{round_number}-u{sequence:04d}"
            if isinstance(round_number, int) and isinstance(sequence, int)
            else ""
        )
        if source_throw_id != expected_throw_id:
            errors.append(
                f"{location}: source_throw_id no coincide con ronda y secuencia"
            )
        if isinstance(source_throw_id, str):
            if source_throw_id in source_throws:
                errors.append(f"{location}: source_throw_id duplicado")
            source_throws[source_throw_id] = record
        round_record = rounds_by_number.get(round_number)
        launch_tick = validate_utility_launch(
            launch, round_record, f"{location}.launch", errors
        )
        sort_key = (
            int(round_number) if isinstance(round_number, int) else -1,
            int(sequence) if isinstance(sequence, int) else -1,
            str(record.get("event_id", "")),
        )
        if previous_sort_key is not None and sort_key < previous_sort_key:
            errors.append(f"{location}: orden causal no determinista")
        previous_sort_key = sort_key
        source_entity = validate_utility_source_entity(record, location, errors)
        correlation_status, correlation_source = validate_utility_correlation(
            record.get("correlation"), f"{location}.correlation", errors
        )
        if (
            correlation_status == "observed"
            and correlation_source in {"projectile_entity", "grenade_entity_id"}
            and source_entity is None
        ):
            errors.append(f"{location}: correlacion observada requiere source_entity")
        if source_entity is not None:
            previous_throw = source_entities.get(source_entity)
            if previous_throw is not None and previous_throw != source_throw_id:
                errors.append(
                    f"{location}: una entidad estable corresponde a mas de un throw"
                )
            source_entities[source_entity] = str(source_throw_id)
        thrower_status = record.get("thrower_status")
        thrower_source = record.get("thrower_source")
        thrower_player_id = record.get("thrower_player_id")
        thrower_side = record.get("thrower_side")
        if thrower_status not in UTILITY_THROWER_STATUSES:
            errors.append(f"{location}: thrower_status no es valido")
        if thrower_source not in UTILITY_THROWER_SOURCES:
            errors.append(f"{location}: thrower_source no es valido")
        if thrower_status == "observed":
            if thrower_source == "unavailable":
                errors.append(
                    f"{location}: thrower observado no puede usar unavailable"
                )
            if (
                not isinstance(thrower_player_id, str)
                or PLAYER_ID_PATTERN.fullmatch(thrower_player_id) is None
            ):
                errors.append(f"{location}: thrower_player_id observado no es valido")
            if thrower_side is not None and thrower_side not in {"ct", "t"}:
                errors.append(f"{location}: thrower_side observado no es valido")
            expected_side = sides_by_round.get(round_number, {}).get(thrower_player_id)
            if (
                expected_side is not None
                and thrower_side is not None
                and expected_side != thrower_side
            ):
                errors.append(
                    f"{location}: thrower_side no coincide con side_assignments"
                )
        else:
            if thrower_player_id is not None or thrower_side is not None:
                errors.append(
                    f"{location}: thrower no disponible debe usar referencias null"
                )
            if thrower_source != "unavailable":
                errors.append(
                    f"{location}: thrower no disponible debe usar source unavailable"
                )
        lifecycle_result = validate_utility_lifecycle(
            record.get("lifecycle"),
            record.get("utility_type"),
            launch_tick,
            round_record,
            tick_rate,
            f"{location}.lifecycle",
            errors,
        )
        effect_terminal_ticks = [
            lifecycle_result.get(f"{name}_tick")
            for name in ("detonation", "effect_start")
            if isinstance(lifecycle_result.get(f"{name}_tick"), int)
        ]
        validate_utility_trajectory(
            record.get("trajectory"),
            launch_tick,
            round_record,
            min(effect_terminal_ticks) if effect_terminal_ticks else None,
            lifecycle_result.get("destroy_tick")
            if isinstance(lifecycle_result.get("destroy_tick"), int)
            else None,
            finite_vector(
                record.get("lifecycle", {}).get("destroy", {}).get("position")
            )
            if isinstance(record.get("lifecycle"), Mapping)
            and isinstance(record["lifecycle"].get("destroy"), Mapping)
            and record["lifecycle"]["destroy"].get("position_status") == "observed"
            else None,
            f"{location}.trajectory",
            errors,
        )
        lifecycle_ticks = [
            tick
            for key, tick in lifecycle_result.items()
            if key.endswith("_tick") and isinstance(tick, int)
        ]
        causal_tick = launch_tick
        if causal_tick is None and lifecycle_ticks:
            causal_tick = min(lifecycle_ticks)
        if isinstance(round_number, int) and isinstance(causal_tick, int):
            previous_causal_tick = last_causal_tick_by_round.get(round_number)
            if previous_causal_tick is not None and causal_tick < previous_causal_tick:
                errors.append(f"{location}: tick causal retrocede dentro de la ronda")
            last_causal_tick_by_round[round_number] = causal_tick
        affected_raw = record.get("affected_players")
        if not isinstance(affected_raw, list):
            errors.append(f"{location}: affected_players debe ser una lista")
            affected_raw = []
        affected = [
            result
            for affected_index, item in enumerate(affected_raw)
            if (
                result := validate_utility_affected_player(
                    item,
                    record.get("utility_type"),
                    thrower_player_id,
                    thrower_side,
                    sides_by_round.get(round_number, {}),
                    launch_tick,
                    round_record,
                    f"{location}.affected_players[{affected_index}]",
                    errors,
                )
            )
            is not None
        ]
        player_ids = [
            item.get("player_id")
            for item in affected
            if item.get("player_id") is not None
        ]
        if len(player_ids) != len(set(player_ids)):
            errors.append(f"{location}: affected_players contiene jugadores duplicados")
        numeric_player_ids = [
            int(str(player_id).split(":", 1)[1])
            for player_id in player_ids
            if PLAYER_ID_PATTERN.fullmatch(str(player_id))
        ]
        if numeric_player_ids != sorted(numeric_player_ids):
            errors.append(
                f"{location}: affected_players no esta ordenado por player_id"
            )
        validate_utility_flash_summary(
            record.get("flash_summary"),
            affected,
            f"{location}.flash_summary",
            errors,
        )
        validate_utility_damage_summary(
            record.get("damage_summary"), affected, f"{location}.damage_summary", errors
        )
    for source_throw_id, record in source_throws.items():
        lifecycle = record.get("lifecycle")
        if not isinstance(lifecycle, Mapping):
            continue
        reference = lifecycle.get("extinguished_by_throw_id")
        referenced_id = (
            reference.get("value") if isinstance(reference, Mapping) else None
        )
        if not isinstance(referenced_id, str) or not referenced_id:
            continue
        referenced = source_throws.get(referenced_id)
        if referenced is None:
            errors.append(
                f"canonical/events/utility_events.jsonl: {source_throw_id} referencia un throw extintor inexistente"
            )
        elif referenced.get("utility_type") != "smoke":
            errors.append(
                f"canonical/events/utility_events.jsonl: {source_throw_id} no referencia una smoke extintora"
            )
        elif referenced.get("round_number") != record.get("round_number"):
            errors.append(
                f"canonical/events/utility_events.jsonl: {source_throw_id} referencia una smoke de otra ronda"
            )


def validate_objective_event_fields(
    record: Mapping[str, object],
    index: int,
    rounds_by_number: Mapping[int, Mapping[str, object]],
    player_sides_by_round: Mapping[int, Mapping[str, str]],
    match_id: str,
    errors: list[str],
) -> None:
    location = f"canonical/events/objective_events.jsonl: registro {index}"
    event_type = record.get("event_type")
    if event_type not in OBJECTIVE_EVENT_TYPES:
        errors.append(f"{location}: event_type no es valido")
        return

    tick = record.get("tick")
    sequence = record.get("sequence_in_tick")
    round_number = record.get("round_number")
    if (
        isinstance(round_number, bool)
        or not isinstance(round_number, int)
        or round_number < 1
    ):
        errors.append(f"{location}: round_number debe ser un entero positivo")
    if isinstance(tick, bool) or not isinstance(tick, int) or tick < 0:
        errors.append(f"{location}: tick debe ser un entero no negativo")
    if isinstance(sequence, bool) or not isinstance(sequence, int) or sequence < 1:
        errors.append(f"{location}: sequence_in_tick debe ser un entero positivo")

    round_record = rounds_by_number.get(record.get("round_number"))
    if (
        round_record is not None
        and isinstance(tick, int)
        and not isinstance(tick, bool)
    ):
        start_tick = round_record.get("start_tick")
        end_tick = round_record.get("end_tick")
        if isinstance(start_tick, int) and tick < start_tick:
            errors.append(f"{location}: tick anterior al inicio de ronda")
        if isinstance(end_tick, int) and tick > end_tick:
            errors.append(f"{location}: tick posterior al fin de ronda")

    source = record.get("source")
    if source not in OBJECTIVE_EVENT_SOURCES:
        errors.append(f"{location}: source no es valido")
    expected_source = (
        "game_state_snapshot"
        if event_type == "bomb_carrier_snapshot"
        else "demoinfocs_event"
    )
    if source != expected_source:
        errors.append(f"{location}: {event_type} requiere source={expected_source}")

    position_status = record.get("position_status")
    position = record.get("position")
    if position_status not in OBJECTIVE_POSITION_STATUSES:
        errors.append(f"{location}: position_status no es valido")
    elif position_status == "observed" and finite_vector(position) is None:
        errors.append(f"{location}: position observed requiere vector finito")
    elif position_status == "unavailable" and position is not None:
        errors.append(f"{location}: position unavailable requiere position=null")

    expected_state = OBJECTIVE_STATE_AFTER_BY_EVENT[event_type]
    expected_states = {expected_state}
    if event_type == "bomb_plant_abort":
        expected_states.add("dropped")
    if record.get("state_after") not in OBJECTIVE_STATES:
        errors.append(f"{location}: state_after no es valido")
    elif record.get("state_after") not in expected_states:
        errors.append(
            f"{location}: {event_type} requiere state_after en {sorted(expected_states)}"
        )
    expected_phase = OBJECTIVE_PHASE_BY_STATE.get(
        record.get("state_after"), OBJECTIVE_PHASE_BY_STATE[expected_state]
    )
    if record.get("phase_after") not in OBJECTIVE_PHASES:
        errors.append(f"{location}: phase_after no es valido")
    elif record.get("phase_after") != expected_phase:
        errors.append(f"{location}: {event_type} requiere phase_after={expected_phase}")

    actor = record.get("actor_player_id")
    actor_side = record.get("actor_side")
    expected_side = OBJECTIVE_ACTOR_SIDE_BY_EVENT.get(event_type, "t")
    if (actor is None) != (actor_side is None):
        errors.append(f"{location}: actor_player_id y actor_side deben aparecer juntos")
    elif actor is not None:
        if not isinstance(actor, str) or not actor:
            errors.append(f"{location}: actor_player_id no es valido")
        if actor_side != expected_side:
            errors.append(
                f"{location}: {event_type} requiere actor_side={expected_side}"
            )
    if isinstance(actor, str):
        roster_side = player_sides_by_round.get(record.get("round_number"), {}).get(
            actor
        )
        if roster_side in {"ct", "t"} and actor_side != roster_side:
            errors.append(f"{location}: actor_side no coincide con side_assignments")

    site = record.get("site")
    if event_type in OBJECTIVE_SITE_EVENTS:
        if site not in {"A", "B"}:
            errors.append(f"{location}: {event_type} requiere site A o B")
    elif site is not None:
        errors.append(f"{location}: {event_type} requiere site=null")

    attempt_id = record.get("attempt_id")
    if event_type in OBJECTIVE_ATTEMPT_KIND_BY_EVENT:
        if not isinstance(attempt_id, str) or not attempt_id.strip():
            errors.append(f"{location}: {event_type} requiere attempt_id no vacio")
        else:
            kind = OBJECTIVE_ATTEMPT_KIND_BY_EVENT[event_type]
            round_number = record.get("round_number")
            expected_pattern = (
                re.compile(
                    rf"^{re.escape(match_id)}:objective-attempt:{round_number:03d}:{kind}:[0-9]{{3,}}$"
                )
                if isinstance(round_number, int)
                else None
            )
            if (
                expected_pattern is None
                or expected_pattern.fullmatch(attempt_id) is None
            ):
                errors.append(f"{location}: attempt_id no sigue el formato canonico")
    elif attempt_id is not None:
        errors.append(f"{location}: {event_type} requiere attempt_id=null")

    attempt_outcome = record.get("attempt_outcome")
    start_observed = record.get("attempt_start_observed")
    if event_type in OBJECTIVE_ATTEMPT_KIND_BY_EVENT:
        expected_outcome = OBJECTIVE_ATTEMPT_OUTCOME_BY_EVENT[event_type]
        if attempt_outcome != expected_outcome:
            errors.append(
                f"{location}: {event_type} requiere attempt_outcome={expected_outcome}"
            )
        if not isinstance(start_observed, bool):
            errors.append(
                f"{location}: {event_type} requiere attempt_start_observed booleano"
            )
        elif event_type in OBJECTIVE_ATTEMPT_START_EVENTS and not start_observed:
            errors.append(f"{location}: un start observado requiere valor true")
    else:
        if attempt_outcome is not None:
            errors.append(f"{location}: {event_type} requiere attempt_outcome=null")
        if start_observed is not None:
            errors.append(
                f"{location}: {event_type} requiere attempt_start_observed=null"
            )

    duration = record.get("action_duration_ms")
    if event_type in OBJECTIVE_ATTEMPT_END_EVENTS:
        if start_observed is False and duration is not None:
            errors.append(
                f"{location}: un terminal sin start observado requiere action_duration_ms=null"
            )
        elif start_observed is True and (
            isinstance(duration, bool) or not isinstance(duration, int) or duration < 0
        ):
            errors.append(
                f"{location}: {event_type} requiere action_duration_ms no negativo"
            )
    elif duration is not None:
        errors.append(f"{location}: {event_type} requiere action_duration_ms=null")

    has_kit = record.get("has_defuse_kit")
    if OBJECTIVE_ATTEMPT_KIND_BY_EVENT.get(event_type) == "defuse":
        if start_observed is False and has_kit is not None:
            errors.append(
                f"{location}: un defuse sin start observado requiere has_defuse_kit=null"
            )
        elif start_observed is not False and not isinstance(has_kit, bool):
            errors.append(f"{location}: el intento de defuse requiere has_defuse_kit")
    elif has_kit is not None:
        errors.append(f"{location}: has_defuse_kit solo aplica a intentos de defuse")

    entity_id = record.get("bomb_entity_id")
    if entity_id is not None and (
        isinstance(entity_id, bool) or not isinstance(entity_id, int) or entity_id < 0
    ):
        errors.append(f"{location}: bomb_entity_id debe ser null o entero no negativo")


def validate_objective_event_order(records: list[dict], errors: list[str]) -> None:
    keys: list[tuple[object, object, object, object]] = []
    sequences: dict[tuple[object, object], list[object]] = {}
    for record in records:
        key = (
            record.get("round_number"),
            record.get("tick"),
            record.get("sequence_in_tick"),
            record.get("event_id"),
        )
        keys.append(key)
        sequences.setdefault(key[:2], []).append(record.get("sequence_in_tick"))
    if keys != sorted(
        keys, key=lambda key: tuple(sortable_value(value) for value in key)
    ):
        errors.append(
            "canonical/events/objective_events.jsonl: eventos fuera de orden causal"
        )
    for (round_number, tick), values in sequences.items():
        if values != list(range(1, len(values) + 1)):
            errors.append(
                "canonical/events/objective_events.jsonl: "
                f"sequence_in_tick no es contiguo en ronda {round_number}, tick {tick}"
            )


def validate_objective_attempt(
    record: Mapping[str, object],
    open_attempts: dict[str, tuple[str, int, object, object] | None],
    used_attempts: set[str],
    terminal_attempts: set[str],
    defuse_attempt_kits: dict[str, bool],
    tick_rate: float,
    location: str,
    errors: list[str],
) -> None:
    event_type = record.get("event_type")
    kind = OBJECTIVE_ATTEMPT_KIND_BY_EVENT.get(event_type)
    if kind is None:
        return
    attempt_id = record.get("attempt_id")
    tick = record.get("tick")
    if not isinstance(attempt_id, str) or not isinstance(tick, int):
        return

    if event_type in OBJECTIVE_ATTEMPT_START_EVENTS:
        if attempt_id in used_attempts:
            errors.append(f"{location}: attempt_id reutilizado")
        if open_attempts[kind] is not None:
            errors.append(f"{location}: hay otro intento {kind} abierto")
        used_attempts.add(attempt_id)
        open_attempts[kind] = (
            attempt_id,
            tick,
            record.get("actor_player_id"),
            record.get("site"),
        )
        if kind == "defuse" and isinstance(record.get("has_defuse_kit"), bool):
            defuse_attempt_kits[attempt_id] = record["has_defuse_kit"]
        return

    start_observed = record.get("attempt_start_observed")
    if start_observed is False:
        if attempt_id in used_attempts:
            errors.append(
                f"{location}: attempt_start_observed=false contradice un start observado"
            )
        if open_attempts[kind] is not None:
            errors.append(
                f"{location}: terminal sin start contradice un intento {kind} abierto"
            )
        if attempt_id in terminal_attempts:
            errors.append(f"{location}: attempt_id terminal reutilizado")
        used_attempts.add(attempt_id)
        terminal_attempts.add(attempt_id)
        return

    opened = open_attempts[kind]
    if opened is None or opened[0] != attempt_id:
        errors.append(
            f"{location}: terminal/abort no referencia el intento {kind} abierto"
        )
        return
    if record.get("actor_player_id") != opened[2]:
        errors.append(f"{location}: actor_player_id cambia dentro del attempt_id")
    if record.get("site") != opened[3]:
        errors.append(f"{location}: site cambia dentro del attempt_id")
    if kind == "defuse" and record.get("has_defuse_kit") != defuse_attempt_kits.get(
        attempt_id
    ):
        errors.append(f"{location}: has_defuse_kit cambia dentro del attempt_id")
    duration = record.get("action_duration_ms")
    if (
        tick_rate > 0
        and isinstance(duration, int)
        and not isinstance(duration, bool)
        and duration >= 0
    ):
        raw_duration = (tick - opened[1]) * 1000 / tick_rate
        expected_duration = math.floor(raw_duration + 0.5)
        if duration != expected_duration:
            errors.append(
                f"{location}: action_duration_ms no coincide con los ticks del intento"
            )
    open_attempts[kind] = None
    terminal_attempts.add(attempt_id)


def validate_objective_state_machine(
    round_number: int,
    records: list[dict],
    tick_rate: float,
    errors: list[str],
) -> None:
    allowed_previous = {
        "bomb_carrier_snapshot": {None, "carried", "dropped"},
        "bomb_drop": {None, "carried", "planting"},
        "bomb_pickup": {None, "carried", "dropped"},
        "bomb_plant_start": {None, "carried"},
        "bomb_plant_abort": {"planting", "dropped"},
        "bomb_plant": {"planting"},
        "bomb_defuse_start": {"planted"},
        "bomb_defuse_abort": {"defusing"},
        "bomb_defuse": {"defusing"},
        "bomb_explode": {"planted", "defusing"},
    }
    state: str | None = None
    open_attempts: dict[str, tuple[str, int, object, object] | None] = {
        "plant": None,
        "defuse": None,
    }
    used_attempts: set[str] = set()
    terminal_attempts: set[str] = set()
    attempt_order: dict[str, list[str]] = {"plant": [], "defuse": []}
    seen_attempt_ids: set[str] = set()
    defuse_attempt_kits: dict[str, bool] = {}
    attempt_sites: dict[str, set[str]] = {}
    planted_site: str | None = None
    terminal_seen = False

    for index, record in enumerate(records):
        event_type = record.get("event_type")
        location = (
            "canonical/events/objective_events.jsonl: "
            f"ronda {round_number}, evento {index}"
        )
        if event_type not in OBJECTIVE_EVENT_TYPES:
            continue
        attempt_id = record.get("attempt_id")
        attempt_kind = OBJECTIVE_ATTEMPT_KIND_BY_EVENT.get(event_type)
        if (
            attempt_kind is not None
            and isinstance(attempt_id, str)
            and attempt_id not in seen_attempt_ids
        ):
            seen_attempt_ids.add(attempt_id)
            attempt_order[attempt_kind].append(attempt_id)
        if terminal_seen:
            errors.append(f"{location}: hay eventos despues del estado terminal")
        if state not in allowed_previous[event_type]:
            unobserved_previous = {
                "bomb_plant_abort": {None, "carried", "dropped"},
                "bomb_plant": {None, "carried"},
                "bomb_defuse_abort": {"planted"},
                "bomb_defuse": {"planted"},
            }
            is_unobserved_terminal = record.get(
                "attempt_start_observed"
            ) is False and state in unobserved_previous.get(event_type, set())
        else:
            is_unobserved_terminal = False
        if state not in allowed_previous[event_type] and not is_unobserved_terminal:
            errors.append(
                f"{location}: transicion {state!r} -> {event_type} no es valida"
            )
        validate_objective_attempt(
            record,
            open_attempts,
            used_attempts,
            terminal_attempts,
            defuse_attempt_kits,
            tick_rate,
            location,
            errors,
        )
        state = record.get("state_after")
        if state in {"defused", "exploded"}:
            terminal_seen = True
        site = record.get("site")
        if isinstance(site, str) and site:
            if isinstance(attempt_id, str) and attempt_id:
                attempt_sites.setdefault(attempt_id, set()).add(site)
            if event_type == "bomb_plant":
                planted_site = site
            elif (
                event_type
                in {
                    "bomb_defuse_start",
                    "bomb_defuse_abort",
                    "bomb_defuse",
                    "bomb_explode",
                }
                and planted_site is not None
                and site != planted_site
            ):
                errors.append(
                    f"{location}: bombsite {site!r} no coincide con el plant "
                    f"en {planted_site!r}"
                )

    for kind, attempt_ids in attempt_order.items():
        for ordinal, attempt_id in enumerate(attempt_ids, start=1):
            if attempt_id.rsplit(":", maxsplit=1)[-1] != f"{ordinal:03d}":
                errors.append(
                    "canonical/events/objective_events.jsonl: "
                    f"ronda {round_number}, ordinals {kind} no son contiguos"
                )
                break
    for attempt_id, sites in attempt_sites.items():
        if len(sites) > 1:
            errors.append(
                f"canonical/events/objective_events.jsonl: ronda {round_number}, "
                f"attempt_id {attempt_id!r} mezcla bombsites {sorted(sites)}"
            )


def validate_round_win_reason(record: Mapping[str, object], errors: list[str]) -> None:
    round_number = record.get("round_number")
    location = f"canonical/core/rounds.json: ronda {round_number}"
    win_reason = record.get("win_reason")
    raw_code = record.get("raw_win_reason_code")
    start_tick = record.get("start_tick")
    end_tick = record.get("end_tick")
    for field, value in (("start_tick", start_tick), ("end_tick", end_tick)):
        if value is not None and (
            isinstance(value, bool) or not isinstance(value, int) or value < 0
        ):
            errors.append(f"{location}: {field} debe ser null o entero no negativo")
    if (start_tick is None) != (end_tick is None):
        errors.append(f"{location}: start_tick y end_tick deben aparecer juntos")
    elif (
        isinstance(start_tick, int)
        and not isinstance(start_tick, bool)
        and isinstance(end_tick, int)
        and not isinstance(end_tick, bool)
        and end_tick < start_tick
    ):
        errors.append(f"{location}: end_tick no puede ser anterior a start_tick")
    if win_reason not in ROUND_WIN_REASONS:
        errors.append(f"{location}: win_reason no es valido")
    if isinstance(raw_code, bool) or not isinstance(raw_code, int) or raw_code < 0:
        errors.append(f"{location}: raw_win_reason_code debe ser entero no negativo")
        return
    expected_reason = ROUND_WIN_REASON_BY_RAW_CODE.get(raw_code, "unknown")
    if win_reason != expected_reason:
        errors.append(f"{location}: win_reason no coincide con raw_win_reason_code")


def validate_round_objective_summary(
    round_record: Mapping[str, object],
    events: list[dict],
    errors: list[str],
) -> None:
    round_number = round_record.get("round_number")
    location = f"canonical/core/rounds.json: ronda {round_number}, objective"
    objective = round_record.get("objective")
    if not isinstance(objective, Mapping):
        errors.append(f"{location}: debe ser un objeto")
        return
    required = (
        "was_bomb_planted",
        "plant_event_id",
        "site",
        "plant_tick",
        "planter_player_id",
        "outcome",
        "resolution_event_id",
        "resolution_tick",
        "resolver_player_id",
        "plant_attempts",
        "plant_aborts",
        "defuse_attempts",
        "defuse_aborts",
        "bomb_drops",
        "bomb_pickups",
    )
    missing = [field for field in required if field not in objective]
    if missing:
        errors.append(f"{location}: faltan campos {', '.join(missing)}")
        return

    by_type: dict[str, list[dict]] = {}
    for event in events:
        by_type.setdefault(str(event.get("event_type")), []).append(event)
    count_fields = {
        "plant_attempts": "bomb_plant_start",
        "plant_aborts": "bomb_plant_abort",
        "defuse_attempts": "bomb_defuse_start",
        "defuse_aborts": "bomb_defuse_abort",
        "bomb_drops": "bomb_drop",
        "bomb_pickups": "bomb_pickup",
    }
    for field, event_type in count_fields.items():
        value = objective.get(field)
        if isinstance(value, bool) or not isinstance(value, int) or value < 0:
            errors.append(f"{location}: {field} debe ser entero no negativo")
        elif value != len(by_type.get(event_type, [])):
            errors.append(f"{location}: {field} no coincide con el ledger")

    plants = by_type.get("bomb_plant", [])
    terminals = by_type.get("bomb_defuse", []) + by_type.get("bomb_explode", [])
    if len(plants) > 1:
        errors.append(f"{location}: hay mas de un bomb_plant")
    if len(terminals) > 1:
        errors.append(f"{location}: hay mas de un evento terminal")
    plant = plants[0] if len(plants) == 1 else None
    terminal = terminals[0] if len(terminals) == 1 else None
    if plant is not None and terminal is not None:
        plant_position = plant.get("position")
        terminal_position = terminal.get("position")
        if (
            terminal.get("event_type") == "bomb_explode"
            and isinstance(plant_position, Mapping)
            and isinstance(terminal_position, Mapping)
        ):
            coordinates = (
                plant_position.get("x"),
                plant_position.get("y"),
                terminal_position.get("x"),
                terminal_position.get("y"),
            )
            if all(
                not isinstance(value, bool)
                and isinstance(value, (int, float))
                and math.isfinite(value)
                for value in coordinates
            ):
                drift = math.hypot(
                    terminal_position["x"] - plant_position["x"],
                    terminal_position["y"] - plant_position["y"],
                )
                if drift > OBJECTIVE_EXPLOSION_POSITION_TOLERANCE_UNITS:
                    errors.append(
                        f"{location}: bomb_explode no conserva la posicion plantada"
                    )
    was_planted = plant is not None
    if objective.get("was_bomb_planted") is not was_planted:
        errors.append(f"{location}: was_bomb_planted no coincide con el ledger")

    plant_fields = {
        "plant_event_id": plant.get("event_id") if plant else None,
        "site": plant.get("site") if plant else None,
        "plant_tick": plant.get("tick") if plant else None,
        "planter_player_id": plant.get("actor_player_id") if plant else None,
    }
    for field, expected in plant_fields.items():
        if objective.get(field) != expected:
            errors.append(f"{location}: {field} no coincide con bomb_plant")

    resolution_fields = {
        "resolution_event_id": terminal.get("event_id") if terminal else None,
        "resolution_tick": terminal.get("tick") if terminal else None,
        "resolver_player_id": (
            terminal.get("actor_player_id")
            if terminal and terminal.get("event_type") == "bomb_defuse"
            else None
        ),
    }
    for field, expected in resolution_fields.items():
        if objective.get(field) != expected:
            errors.append(f"{location}: {field} no coincide con el evento terminal")

    win_reason = round_record.get("win_reason")
    expected_outcome = "not_planted"
    if terminal is not None:
        expected_outcome = (
            "defused" if terminal.get("event_type") == "bomb_defuse" else "exploded"
        )
    elif plant is not None:
        expected_outcome = "elimination_after_plant"
    elif win_reason == "target_saved":
        expected_outcome = "time_expired"
    outcome = objective.get("outcome")
    if outcome not in OBJECTIVE_OUTCOMES:
        errors.append(f"{location}: outcome no es valido")
    elif outcome != expected_outcome:
        errors.append(f"{location}: outcome no coincide con el ledger")

    terminal_reason = {
        "defused": "bomb_defused",
        "exploded": "target_bombed",
        "time_expired": "target_saved",
    }.get(expected_outcome)
    if terminal_reason is not None and win_reason != terminal_reason:
        errors.append(f"{location}: outcome y win_reason discrepan")
    required_outcome_by_reason = {
        "target_bombed": "exploded",
        "bomb_defused": "defused",
        "target_saved": "time_expired",
    }
    required_outcome = required_outcome_by_reason.get(win_reason)
    if required_outcome is not None and expected_outcome != required_outcome:
        errors.append(
            f"{location}: win_reason={win_reason} requiere outcome={required_outcome}"
        )
    if win_reason == "ct_win" and was_planted and terminal is None:
        errors.append(f"{location}: CT win tras plant requiere bomb_defuse")
    if win_reason == "terrorists_win" and expected_outcome == "defused":
        errors.append(f"{location}: terrorists_win no puede terminar en defuse")

    legacy_fields = {
        "bomb_planted": was_planted,
        "bomb_site": plant.get("site") if plant else None,
        "bomb_tick": plant.get("tick") if plant else None,
    }
    for field, expected in legacy_fields.items():
        actual = round_record.get(field)
        differs = (
            actual is not expected if isinstance(expected, bool) else actual != expected
        )
        if differs:
            errors.append(f"{location}: {field} legacy no coincide con objective")


def objective_state_at_tick(
    events: list[dict], tick: int
) -> tuple[str, str | None, bool]:
    state = "carried"
    carrier: str | None = None
    carrier_known = False
    for event in events:
        event_tick = event.get("tick")
        if not isinstance(event_tick, int) or event_tick > tick:
            break
        event_type = event.get("event_type")
        if event_type not in OBJECTIVE_STATE_AFTER_BY_EVENT:
            continue
        state = event.get("state_after")
        actor = event.get("actor_player_id")
        if state in {"carried", "planting"}:
            carrier = actor if isinstance(actor, str) else None
            carrier_known = carrier is not None
        else:
            carrier = None
            carrier_known = True
    return state, carrier, carrier_known


def validate_states_against_objective_ledger(
    events_by_round: Mapping[int, list[dict]],
    player_states: list[dict],
    errors: list[str],
) -> None:
    states_by_tick: dict[tuple[int, int], list[dict]] = {}
    for state in player_states:
        round_number = state.get("round_number")
        tick = state.get("tick")
        if isinstance(round_number, int) and isinstance(tick, int):
            states_by_tick.setdefault((round_number, tick), []).append(state)

    for (round_number, tick), tick_states in states_by_tick.items():
        state, carrier, carrier_known = objective_state_at_tick(
            events_by_round.get(round_number, []), tick
        )
        expected_phase = OBJECTIVE_PHASE_BY_STATE[state]
        location = f"canonical/states/player_states: ronda {round_number}, tick {tick}"
        if any(item.get("objective_phase") != expected_phase for item in tick_states):
            errors.append(f"{location}: objective_phase no coincide con el ledger")

        reported_carriers = {
            item.get("player_id") for item in tick_states if item.get("has_c4") is True
        }
        if carrier_known:
            expected_carriers = {carrier} if carrier is not None else set()
            if reported_carriers != expected_carriers:
                errors.append(
                    f"{location}: has_c4 no coincide con el portador del ledger"
                )

        active_actor = carrier if state == "planting" else None
        if state == "defusing":
            for event in reversed(events_by_round.get(round_number, [])):
                if event.get("tick") <= tick and event.get("state_after") == "defusing":
                    active_actor = event.get("actor_player_id")
                    break
        for item in tick_states:
            should_plant = state == "planting" and item.get("player_id") == active_actor
            should_defuse = (
                state == "defusing"
                and item.get("player_id") == active_actor
                and item.get("is_alive") is True
            )
            if item.get("is_planting") is not should_plant:
                errors.append(f"{location}: is_planting no coincide con el ledger")
                break
            if item.get("is_defusing") is not should_defuse:
                errors.append(f"{location}: is_defusing no coincide con el ledger")
                break


def replay_player_id(value: object) -> str | None:
    if isinstance(value, str) and value.isdecimal():
        return f"steam:{value}"
    if isinstance(value, int) and not isinstance(value, bool) and value >= 0:
        return f"steam:{value}"
    return None


def validate_replay_player_references(
    replay_rounds: list[dict], player_ids: set[str], errors: list[str]
) -> None:
    location = "canonical/presentation/replay"
    for envelope in replay_rounds:
        round_data = envelope.get("round") if isinstance(envelope, Mapping) else None
        if not isinstance(round_data, Mapping):
            continue
        for frame in round_data.get("frames", []):
            if not isinstance(frame, Mapping):
                continue
            seen: set[str] = set()
            players = frame.get("players")
            if not isinstance(players, list):
                errors.append(f"{location}: frame.players debe ser una lista")
                continue
            for player in players:
                steam_id = player.get("steam_id") if isinstance(player, Mapping) else None
                player_id = f"steam:{steam_id}" if isinstance(steam_id, str) else ""
                if PLAYER_ID_PATTERN.fullmatch(player_id) is None or player_id not in player_ids:
                    errors.append(
                        f"{location}: frame contiene un jugador no reconciliado"
                    )
                    continue
                if player_id in seen:
                    errors.append(f"{location}: frame contiene un jugador duplicado")
                seen.add(player_id)


def validate_replay_bomb_state(
    bomb: object,
    players: list[object],
    round_number: int,
    tick: int,
    errors: list[str],
) -> None:
    location = f"canonical/presentation/replay: ronda {round_number}, tick {tick}, bomb"
    if not isinstance(bomb, Mapping):
        errors.append(f"{location}: bomb debe ser un objeto")
        return

    required = (
        "state",
        "objective_phase",
        "is_planted_now",
        "was_planted_this_round",
        "x",
        "y",
        "carrier_id",
        "defuser_id",
        "position_status",
        "position_source",
    )
    missing = [field for field in required if field not in bomb]
    if missing:
        errors.append(f"{location}: faltan campos {', '.join(missing)}")
        return

    state = bomb.get("state")
    phase = bomb.get("objective_phase")
    if state not in OBJECTIVE_STATES | {"unknown"}:
        errors.append(f"{location}: state no es valido")
    if phase not in OBJECTIVE_PHASES:
        errors.append(f"{location}: objective_phase no es valido")
    if phase == "resolved" and state not in {"resolved", "defused", "exploded"}:
        errors.append(f"{location}: fase resolved conserva un estado activo")
    if state == "resolved" and phase != "resolved":
        errors.append(f"{location}: state resolved requiere objective_phase=resolved")

    is_planted = bomb.get("is_planted_now")
    was_planted = bomb.get("was_planted_this_round")
    if not isinstance(is_planted, bool) or not isinstance(was_planted, bool):
        errors.append(f"{location}: flags de plant deben ser booleanos")
    elif is_planted is not (phase in {"planted", "defusing"}):
        errors.append(f"{location}: is_planted_now no coincide con objective_phase")
    if (
        state in {"planted", "defusing", "defused", "exploded"}
        and was_planted is not True
    ):
        errors.append(
            f"{location}: estado postplant requiere was_planted_this_round=true"
        )

    position_status = bomb.get("position_status")
    position_source = bomb.get("position_source")
    if position_status not in OBJECTIVE_POSITION_STATUSES:
        errors.append(f"{location}: position_status no es valido")
    if position_source not in {
        "demoinfocs_event",
        "demoinfocs_native_snapshot",
    }:
        errors.append(f"{location}: position_source no es valido")
    if position_status == "observed" and (
        not is_finite_number(bomb.get("x")) or not is_finite_number(bomb.get("y"))
    ):
        errors.append(f"{location}: posicion observada no es finita")

    holders: set[str] = set()
    living_player_ids: set[str] = set()
    for player in players:
        if not isinstance(player, Mapping):
            continue
        player_id = replay_player_id(player.get("steam_id"))
        if player_id is None:
            continue
        if player.get("alive") is True:
            living_player_ids.add(player_id)
        if player.get("has_c4", False) is True:
            holders.add(player_id)
            if player.get("alive") is not True:
                errors.append(f"{location}: un jugador muerto conserva has_c4=true")
    if len(holders) > 1:
        errors.append(f"{location}: hay mas de un portador C4")

    carrier_raw = bomb.get("carrier_id")
    carrier = replay_player_id(carrier_raw) if carrier_raw is not None else None
    if carrier_raw is not None and carrier is None:
        errors.append(f"{location}: carrier_id no es valido")
    expects_carrier = phase in {"preplant", "planting"} and state in {
        "carried",
        "planting",
    }
    expected_holders = {carrier} if carrier is not None else set()
    if expects_carrier:
        if carrier is None or carrier not in living_player_ids:
            errors.append(f"{location}: estado carried/planting requiere carrier vivo")
        if holders != expected_holders:
            errors.append(f"{location}: has_c4 no coincide con carrier_id")
    elif carrier is not None or holders:
        errors.append(f"{location}: estado sin carrier conserva C4")

    defuser_raw = bomb.get("defuser_id")
    defuser = replay_player_id(defuser_raw) if defuser_raw is not None else None
    if defuser_raw is not None and defuser is None:
        errors.append(f"{location}: defuser_id no es valido")
    if defuser is not None and state not in {"defusing", "defused"}:
        errors.append(f"{location}: defuser_id aparece fuera de defuse")

    if was_planted is True:
        if bomb.get("site") not in {"A", "B"}:
            errors.append(f"{location}: postplant requiere site A o B")
        plant_tick = bomb.get("plant_tick")
        if (
            isinstance(plant_tick, bool)
            or not isinstance(plant_tick, int)
            or plant_tick < 0
        ):
            errors.append(f"{location}: postplant requiere plant_tick no negativo")


def validate_replay_objective_consistency(
    player_states: list[dict],
    replay_rounds: list[dict],
    errors: list[str],
    objective_events: list[dict] | None = None,
) -> None:
    if objective_events is not None:
        validate_replay_objective_markers(objective_events, replay_rounds, errors)
    states_by_tick = {
        (state.get("round_number"), state.get("tick"), state.get("player_id")): state
        for state in player_states
    }
    for envelope in replay_rounds:
        round_record = envelope.get("round")
        if not isinstance(round_record, Mapping):
            continue
        round_number = round_record.get("round")
        frames = round_record.get("frames")
        if not isinstance(round_number, int) or not isinstance(frames, list):
            continue
        for frame in frames:
            if not isinstance(frame, Mapping) or not isinstance(frame.get("tick"), int):
                continue
            tick = frame["tick"]
            players = frame.get("players")
            if not isinstance(players, list):
                continue
            validate_replay_bomb_state(
                frame.get("bomb"), players, round_number, tick, errors
            )
            for replay_player in players:
                if not isinstance(replay_player, Mapping):
                    continue
                player_id = replay_player_id(replay_player.get("steam_id"))
                state = states_by_tick.get((round_number, tick, player_id))
                if state is None:
                    continue
                location = (
                    "canonical/presentation/replay: "
                    f"ronda {round_number}, tick {tick}, jugador {player_id}"
                )
                replay_has_c4 = replay_player.get("has_c4", False)
                replay_has_kit = replay_player.get("has_defuse_kit", False)
                if not isinstance(replay_has_c4, bool):
                    errors.append(f"{location}: replay has_c4 debe ser booleano")
                elif state.get("has_c4") is not replay_has_c4:
                    errors.append(f"{location}: has_c4 discrepa de player_state")
                if not isinstance(replay_has_kit, bool):
                    errors.append(
                        f"{location}: replay has_defuse_kit debe ser booleano"
                    )
                elif state.get("has_defuse_kit") is not replay_has_kit:
                    errors.append(
                        f"{location}: has_defuse_kit discrepa de player_state"
                    )


def validate_replay_objective_markers(
    objective_events: list[dict], replay_rounds: list[dict], errors: list[str]
) -> None:
    if not replay_rounds:
        return
    location = "canonical/presentation/replay: eventos de objetivo"
    expected: dict[tuple[int, int, str], tuple[object, ...]] = {}
    actual: dict[tuple[int, int, str], tuple[object, ...]] = {}

    for event in objective_events:
        event_type = event.get("event_type")
        if event_type not in REPLAY_OBJECTIVE_EVENT_TYPES:
            continue
        key = (event.get("round_number"), event.get("tick"), event_type)
        if not isinstance(key[0], int) or not isinstance(key[1], int):
            continue
        position = event.get("position")
        x = position.get("x") if isinstance(position, Mapping) else None
        y = position.get("y") if isinstance(position, Mapping) else None
        if key in expected:
            errors.append(f"{location}: marker esperado duplicado {key}")
        expected[key] = (
            event.get("actor_player_id"),
            event.get("site"),
            x,
            y,
        )

    for envelope in replay_rounds:
        round_record = envelope.get("round")
        if not isinstance(round_record, Mapping):
            continue
        round_number = round_record.get("round")
        replay_events = round_record.get("events")
        if not isinstance(round_number, int) or not isinstance(replay_events, list):
            continue
        for event in replay_events:
            if not isinstance(event, Mapping):
                continue
            event_type = event.get("type")
            if event_type not in REPLAY_OBJECTIVE_EVENT_TYPES:
                continue
            tick = event.get("tick")
            if not isinstance(tick, int):
                errors.append(
                    f"{location}: marker sin tick valido en ronda {round_number}"
                )
                continue
            actor_raw = event.get("actor_id") or event.get("player_id")
            actor = (
                replay_player_id(actor_raw) if actor_raw not in (None, 0, "0") else None
            )
            key = (round_number, tick, event_type)
            if key in actual:
                errors.append(f"{location}: marker replay duplicado {key}")
            actual[key] = (
                actor,
                event.get("site") or None,
                event.get("x"),
                event.get("y"),
            )

    missing = sorted(set(expected) - set(actual))
    extra = sorted(set(actual) - set(expected))
    if missing:
        errors.append(f"{location}: faltan markers del ledger {missing}")
    if extra:
        errors.append(f"{location}: hay markers extra o post-resolucion {extra}")
    for key in sorted(set(expected) & set(actual)):
        expected_actor, expected_site, expected_x, expected_y = expected[key]
        actual_actor, actual_site, actual_x, actual_y = actual[key]
        if actual_actor != expected_actor:
            errors.append(f"{location}: actor no coincide con el ledger en {key}")
        if actual_site != expected_site:
            errors.append(f"{location}: site no coincide con el ledger en {key}")
        for axis, expected_value, actual_value in (
            ("x", expected_x, actual_x),
            ("y", expected_y, actual_y),
        ):
            if expected_value is None:
                continue
            if not is_finite_number(actual_value) or not math.isclose(
                float(actual_value), float(expected_value), abs_tol=1e-6
            ):
                errors.append(
                    f"{location}: posicion {axis} no coincide con el ledger en {key}"
                )


def utility_replay_expected_marker(event: Mapping[str, object]) -> dict | None:
    lifecycle = event.get("lifecycle")
    if not isinstance(lifecycle, Mapping):
        return None
    moment = next(
        (
            candidate
            for name in ("effect_start", "detonation")
            if isinstance((candidate := lifecycle.get(name)), Mapping)
            and candidate.get("status") == "observed"
        ),
        None,
    )
    if moment is None or not isinstance(moment.get("tick"), int):
        return None
    position = finite_vector(moment.get("position"))
    position_observed = moment.get("position_status") == "observed"
    coordinates = position if position_observed and position is not None else (0.0,) * 3

    thrower_player_id = event.get("thrower_player_id")
    actor_id = (
        thrower_player_id.removeprefix("steam:")
        if isinstance(thrower_player_id, str)
        and PLAYER_ID_PATTERN.fullmatch(thrower_player_id)
        else None
    )
    affected = event.get("affected_players")
    affected = affected if isinstance(affected, list) else []
    affected_player_ids = sorted(
        {
            player_id.removeprefix("steam:")
            for player in affected
            if isinstance(player, Mapping)
            and isinstance((blind := player.get("blind_duration")), Mapping)
            and blind.get("status") in {"observed", "unavailable"}
            and isinstance((player_id := player.get("player_id")), str)
            and PLAYER_ID_PATTERN.fullmatch(player_id)
        },
        key=int,
    )
    duration = lifecycle.get("duration")
    duration_status = "unavailable"
    duration_source = "unavailable"
    duration_ms = 0
    if (
        isinstance(duration, Mapping)
        and duration.get("status") == "observed"
        and is_finite_number(duration.get("milliseconds"))
    ):
        duration_ms = math.floor(float(duration["milliseconds"]) + 0.5)
        duration_status = "observed"
        duration_source = str(duration.get("source"))
    else:
        observed_blinds = [
            blind
            for player in affected
            if isinstance(player, Mapping)
            and isinstance((blind := player.get("blind_duration")), Mapping)
            and blind.get("status") == "observed"
            and is_finite_number(blind.get("milliseconds"))
        ]
        if observed_blinds:
            maximum = max(observed_blinds, key=lambda item: float(item["milliseconds"]))
            duration_ms = math.floor(float(maximum["milliseconds"]) + 0.5)
            duration_status = "observed"
            duration_source = str(maximum.get("source"))

    correlation = event.get("correlation")
    correlation = correlation if isinstance(correlation, Mapping) else {}
    damage_summary = event.get("damage_summary")
    damage_summary = damage_summary if isinstance(damage_summary, Mapping) else {}
    return {
        "id": f"utility:{event.get('source_throw_id')}",
        "tick": moment.get("tick"),
        "type": "utility_detonate",
        "grenade_type": event.get("utility_type"),
        "utility_type": event.get("utility_type"),
        "actor_id": actor_id,
        "player_id": actor_id,
        "x": coordinates[0],
        "y": coordinates[1],
        "z": coordinates[2],
        "position_status": moment.get("position_status"),
        "position_source": moment.get("source") if position_observed else "unavailable",
        "affected_player_ids": affected_player_ids,
        "damage": damage_summary.get("total_damage", 0),
        "duration_ms": duration_ms,
        "duration_status": duration_status,
        "duration_source": duration_source,
        "correlation_status": correlation.get("status"),
        "correlation_source": correlation.get("source"),
        "source_throw_id": event.get("source_throw_id"),
    }


def utility_replay_actual_value(marker: Mapping[str, object], field: str) -> object:
    defaults: dict[str, object] = {
        "actor_id": None,
        "player_id": None,
        "affected_player_ids": [],
        "damage": 0,
        "duration_ms": 0,
    }
    return marker.get(field, defaults.get(field))


def utility_replay_values_equal(actual: object, expected: object) -> bool:
    if isinstance(expected, bool):
        return isinstance(actual, bool) and actual is expected
    if isinstance(expected, int):
        return (
            isinstance(actual, int)
            and not isinstance(actual, bool)
            and actual == expected
        )
    if isinstance(expected, float):
        return is_finite_number(actual) and float(actual) == expected
    return type(actual) is type(expected) and actual == expected


def validate_replay_utility_markers(
    utility_events: list[dict], replay_rounds: list[dict], errors: list[str]
) -> int:
    location = "canonical/presentation/replay: markers utility_detonate"
    expected = {
        str(event.get("source_throw_id")): (
            event.get("round_number"),
            marker,
        )
        for event in utility_events
        if (marker := utility_replay_expected_marker(event)) is not None
    }
    seen: set[str] = set()
    mismatches = 0
    for envelope in replay_rounds:
        round_record = envelope.get("round")
        if not isinstance(round_record, Mapping):
            continue
        round_number = round_record.get("round")
        replay_events = round_record.get("events")
        if not isinstance(replay_events, list):
            continue
        for marker in replay_events:
            if (
                not isinstance(marker, Mapping)
                or marker.get("type") != "utility_detonate"
            ):
                continue
            source_throw_id = marker.get("source_throw_id")
            if not isinstance(source_throw_id, str) or source_throw_id not in expected:
                mismatches += 1
                errors.append(f"{location}: marker inesperado para {source_throw_id!r}")
                continue
            if source_throw_id in seen:
                mismatches += 1
                errors.append(f"{location}: marker duplicado para {source_throw_id}")
                continue
            seen.add(source_throw_id)
            expected_round, expected_marker = expected[source_throw_id]
            marker_mismatch = round_number != expected_round
            if marker_mismatch:
                errors.append(f"{location}: ronda no coincide para {source_throw_id}")
            required_provenance = (
                "duration_status",
                "duration_source",
                "position_status",
                "position_source",
                "correlation_status",
                "correlation_source",
                "source_throw_id",
            )
            for field in required_provenance:
                if not isinstance(marker.get(field), str) or not marker[field]:
                    marker_mismatch = True
                    errors.append(
                        f"{location}: {field} requerido para {source_throw_id}"
                    )
            for field, expected_value in expected_marker.items():
                actual_value = utility_replay_actual_value(marker, field)
                if not utility_replay_values_equal(actual_value, expected_value):
                    marker_mismatch = True
                    errors.append(
                        f"{location}: {field} no coincide para {source_throw_id}"
                    )
            if marker_mismatch:
                mismatches += 1
    for missing in sorted(set(expected) - seen):
        mismatches += 1
        errors.append(f"{location}: falta marker para {missing}")
    return mismatches


def validate_objective_semantics(
    match_id: str,
    match_envelope: Mapping[str, object],
    participants: list[dict],
    rounds: list[dict],
    objective_events: list[dict],
    player_states: list[dict],
    replay_rounds: list[dict],
    errors: list[str],
) -> None:
    rounds_by_number = {
        record.get("round_number"): record
        for record in rounds
        if isinstance(record.get("round_number"), int)
    }
    player_teams = {
        record.get("player_id"): record.get("team_id")
        for record in participants
        if isinstance(record.get("player_id"), str)
        and isinstance(record.get("team_id"), str)
    }
    player_sides_by_round: dict[int, dict[str, str]] = {}
    for round_number, round_record in rounds_by_number.items():
        assignments = round_record.get("side_assignments")
        side_by_team = (
            {
                assignment.get("team_id"): assignment.get("side")
                for assignment in assignments
                if isinstance(assignment, Mapping)
            }
            if isinstance(assignments, list)
            else {}
        )
        player_sides_by_round[round_number] = {
            player_id: side_by_team[team_id]
            for player_id, team_id in player_teams.items()
            if side_by_team.get(team_id) in {"ct", "t"}
        }
    for index, record in enumerate(objective_events):
        validate_objective_event_fields(
            record,
            index,
            rounds_by_number,
            player_sides_by_round,
            match_id,
            errors,
        )
    validate_objective_event_order(objective_events, errors)

    events_by_round: dict[int, list[dict]] = {}
    for record in objective_events:
        round_number = record.get("round_number")
        if isinstance(round_number, int):
            events_by_round.setdefault(round_number, []).append(record)
    tick_rate = match_envelope.get("tick_rate_hz")
    valid_tick_rate = (
        float(tick_rate) if is_finite_number(tick_rate) and float(tick_rate) > 0 else 0
    )
    for round_number, round_record in rounds_by_number.items():
        round_events = events_by_round.get(round_number, [])
        validate_round_win_reason(round_record, errors)
        validate_objective_state_machine(
            round_number, round_events, valid_tick_rate, errors
        )
        validate_round_objective_summary(round_record, round_events, errors)
    for state in player_states:
        round_number = state.get("round_number")
        player_id = state.get("player_id")
        location = f"canonical/states/player_states: estado {state.get('state_id')}"
        expected_team = player_teams.get(player_id)
        if expected_team is not None and state.get("team_id") != expected_team:
            errors.append(f"{location}: team_id no coincide con participants")
        expected_side = player_sides_by_round.get(round_number, {}).get(player_id)
        if expected_side is not None and state.get("side") != expected_side:
            errors.append(f"{location}: side no coincide con side_assignments")
        tick = state.get("tick")
        round_record = rounds_by_number.get(round_number)
        if isinstance(round_record, Mapping) and isinstance(tick, int):
            start_tick = round_record.get("start_tick")
            end_tick = round_record.get("end_tick")
            if isinstance(start_tick, int) and tick < start_tick:
                errors.append(f"{location}: tick anterior al inicio de ronda")
            if isinstance(end_tick, int) and tick > end_tick:
                errors.append(f"{location}: tick posterior al fin de ronda")
    validate_states_against_objective_ledger(events_by_round, player_states, errors)
    validate_replay_objective_consistency(
        player_states, replay_rounds, errors, objective_events
    )


def validate_rounds(
    match_id: str, records: list[dict], errors: list[str]
) -> dict[str, int]:
    round_ids = validate_unique_ids(
        records, "round_id", "canonical/core/rounds.json", errors
    )
    numbers = [record.get("round_number") for record in records]
    if numbers != list(range(1, len(records) + 1)):
        errors.append(
            "canonical/core/rounds.json: round_number debe ser contiguo desde 1"
        )
    expected_ids = {
        f"{match_id}:round:{number:03d}" for number in range(1, len(records) + 1)
    }
    if round_ids != expected_ids:
        errors.append(
            "canonical/core/rounds.json: round_id no sigue el formato canónico"
        )
    return {
        record["round_id"]: record["round_number"]
        for record in records
        if isinstance(record.get("round_id"), str)
        and isinstance(record.get("round_number"), int)
    }


def validate_round_outcomes(
    match_envelope: Mapping[str, object],
    rounds: list[dict],
    replay_index: Mapping[str, object],
    errors: list[str],
) -> None:
    if not isinstance(match_envelope.get("ct_score"), int) or not isinstance(
        match_envelope.get("t_score"), int
    ):
        errors.append("canonical/core/match.json: ct_score y t_score deben ser enteros")
    replay_rounds = replay_index.get("rounds")
    if isinstance(replay_rounds, list):
        replay_winners = {
            record.get("round_number"): record.get("winner_side")
            for record in replay_rounds
            if isinstance(record, Mapping)
        }
        for record in rounds:
            number = record.get("round_number")
            if (
                number in replay_winners
                and record.get("winner_side") != replay_winners[number]
            ):
                errors.append(
                    f"canonical/core/rounds.json: winner_side de ronda {number} no coincide con replay"
                )

    if not rounds:
        return
    final_round = rounds[-1]
    if any(
        record.get(field) is None
        for record in rounds
        for field in (
            "winner_side",
            "winner_team_id",
            "ct_score_after",
            "t_score_after",
        )
    ):
        errors.append("canonical/core/rounds.json: resultados de ronda incompletos")
    if final_round.get("ct_score_after") != match_envelope.get(
        "ct_score"
    ) or final_round.get("t_score_after") != match_envelope.get("t_score"):
        errors.append(
            "canonical/core/rounds.json: score final no coincide con core/match.json"
        )

    teams = match_envelope.get("teams")
    team_records = teams if isinstance(teams, list) else []
    team_wins = {
        team.get("team_id"): 0
        for team in team_records
        if isinstance(team, Mapping) and isinstance(team.get("team_id"), str)
    }
    previous_scores = dict(team_wins)
    for record in rounds:
        winner_team_id = record.get("winner_team_id")
        assignments = record.get("side_assignments")
        assignment_records = assignments if isinstance(assignments, list) else []
        assigned_sides = {
            item.get("team_id"): item.get("side")
            for item in assignment_records
            if isinstance(item, Mapping)
        }
        if set(assigned_sides) != set(team_wins) or assigned_sides.get(
            winner_team_id
        ) != record.get("winner_side"):
            errors.append(
                f"canonical/core/rounds.json: ganador y side_assignments discrepan en ronda {record.get('round_number')}"
            )
        if isinstance(winner_team_id, str):
            team_wins[winner_team_id] = team_wins.get(winner_team_id, 0) + 1
        raw_scores = record.get("team_scores_after")
        if not isinstance(raw_scores, list) or not all(
            isinstance(item, Mapping) for item in raw_scores
        ):
            errors.append(
                "canonical/core/rounds.json: team_scores_after debe ser una lista de objetos"
            )
            continue
        current_scores = {
            item.get("team_id"): item.get("score")
            for item in raw_scores
            if isinstance(item.get("team_id"), str)
            and isinstance(item.get("score"), int)
        }
        if set(current_scores) != set(team_wins) or len(current_scores) != len(
            raw_scores
        ):
            errors.append(
                "canonical/core/rounds.json: team_scores_after no contiene cada equipo exactamente una vez"
            )
            continue
        expected_scores = dict(previous_scores)
        if isinstance(winner_team_id, str) and winner_team_id in expected_scores:
            expected_scores[winner_team_id] += 1
        if current_scores != expected_scores:
            errors.append(
                f"canonical/core/rounds.json: transición de score inválida en ronda {record.get('round_number')}"
            )
        previous_scores = current_scores
    if isinstance(teams, list):
        declared_scores = {
            team.get("team_id"): team.get("score")
            for team in teams
            if isinstance(team, Mapping) and isinstance(team.get("team_id"), str)
        }
        if declared_scores != team_wins:
            errors.append(
                "canonical/core/match.json: scores por team_id no coinciden con ganadores de ronda"
            )
        if declared_scores != previous_scores:
            errors.append(
                "canonical/core/match.json: scores finales no coinciden con team_scores_after"
            )


def validate_event_ids(
    match_id: str,
    event_type: str,
    records: list[dict],
    errors: list[str],
) -> set[str]:
    location = f"canonical/events/{event_type}_events.jsonl"
    event_ids = validate_unique_ids(records, "event_id", location, errors)
    if event_type == "utility":
        for record in records:
            expected = f"{match_id}:utility:{record.get('source_throw_id', '')}"
            if record.get("event_id") != expected:
                errors.append(f"{location}: event_id no deriva 1:1 de source_throw_id")
        return event_ids
    pattern = re.compile(
        rf"^{re.escape(match_id)}:{event_type}:([0-9]{{3,}}):([0-9]+):([0-9]{{3,}})$"
    )
    for record in records:
        match = pattern.fullmatch(str(record.get("event_id", "")))
        if match is None:
            errors.append(f"{location}: event_id no sigue el formato canónico")
            continue
        if int(match.group(1)) != record.get("round_number") or int(
            match.group(2)
        ) != record.get("tick"):
            errors.append(f"{location}: event_id discrepa de ronda o tick")
    return event_ids


def utility_observation_metric_counts(
    utility_events: list[dict],
) -> dict[str, int]:
    metrics = {
        "utility_missing_type_observations": 0,
        "utility_missing_actor_observations": 0,
        "utility_missing_launch_tick_observations": 0,
        "utility_missing_launch_position_observations": 0,
        "utility_missing_launch_view_observations": 0,
        "utility_missing_thrower_velocity_observations": 0,
        "utility_missing_projectile_velocity_observations": 0,
        "utility_missing_trajectory_observations": 0,
        "utility_missing_lifecycle_observations": 0,
        "utility_missing_affected_player_observations": 0,
        "utility_missing_flash_duration_observations": 0,
        "utility_inferred_correlations": 0,
    }
    for event in utility_events:
        if (
            event.get("utility_type") == "unknown"
            or event.get("utility_type_status") != "observed"
            or event.get("utility_type_source") == "unavailable"
        ):
            metrics["utility_missing_type_observations"] += 1
        if event.get("thrower_status") != "observed" or event.get(
            "thrower_side"
        ) not in {"ct", "t"}:
            metrics["utility_missing_actor_observations"] += 1
        launch = event.get("launch")
        launch = launch if isinstance(launch, Mapping) else {}
        launch_fields = {
            "tick_status": "utility_missing_launch_tick_observations",
            "position": "utility_missing_launch_position_observations",
            "view": "utility_missing_launch_view_observations",
            "thrower_velocity": "utility_missing_thrower_velocity_observations",
            "projectile_initial_velocity": "utility_missing_projectile_velocity_observations",
        }
        for field, metric in launch_fields.items():
            status = (
                launch.get(field)
                if field == "tick_status"
                else launch.get(field, {}).get("status")
                if isinstance(launch.get(field), Mapping)
                else None
            )
            if status != "observed":
                metrics[metric] += 1
        trajectory = event.get("trajectory")
        if (
            not isinstance(trajectory, Mapping)
            or trajectory.get("status") != "observed"
        ):
            metrics["utility_missing_trajectory_observations"] += 1
        lifecycle = event.get("lifecycle")
        lifecycle = lifecycle if isinstance(lifecycle, Mapping) else {}
        utility_type = event.get("utility_type")
        if utility_type in {"flashbang", "he"}:
            expected_lifecycle_missing = not (
                isinstance(lifecycle.get("detonation"), Mapping)
                and lifecycle["detonation"].get("status") == "observed"
            )
        elif utility_type in {"smoke", "molotov", "incendiary", "decoy"}:
            expected_lifecycle_missing = any(
                not isinstance(lifecycle.get(field), Mapping)
                or lifecycle[field].get("status") != "observed"
                for field in ("effect_start", "expiration")
            )
        else:
            expected_lifecycle_missing = True
        position_coverage_missing = any(
            isinstance(lifecycle.get(field), Mapping)
            and lifecycle[field].get("status") == "observed"
            and lifecycle[field].get("position_status") != "observed"
            for field in (
                "detonation",
                "effect_start",
                "expiration",
                "destroy",
                "extinguish",
            )
        )
        lifecycle_area_missing = not (
            isinstance(lifecycle.get("area"), Mapping)
            and lifecycle["area"].get("status") == "observed"
        )
        correlation = event.get("correlation")
        correlation_status = (
            correlation.get("status") if isinstance(correlation, Mapping) else None
        )
        if (
            expected_lifecycle_missing
            or position_coverage_missing
            or lifecycle_area_missing
            or correlation_status == "unavailable"
        ):
            metrics["utility_missing_lifecycle_observations"] += 1
        if correlation_status == "inferred":
            metrics["utility_inferred_correlations"] += 1
        extinguish_correlation = lifecycle.get("extinguish_correlation")
        if (
            isinstance(extinguish_correlation, Mapping)
            and extinguish_correlation.get("status") == "inferred"
        ):
            metrics["utility_inferred_correlations"] += 1
        affected = event.get("affected_players")
        if not isinstance(affected, list):
            continue
        for player in affected:
            if not isinstance(player, Mapping):
                continue
            blind = player.get("blind_duration")
            blind_status = blind.get("status") if isinstance(blind, Mapping) else None
            has_flash = blind_status in {"observed", "unavailable"}
            if blind_status == "unavailable":
                metrics["utility_missing_flash_duration_observations"] += 1
            if player.get("player_status") == "observed":
                continue
            if has_flash:
                metrics["utility_missing_affected_player_observations"] += 1
            damage_events = player.get("damage_events")
            if isinstance(damage_events, list):
                metrics["utility_missing_affected_player_observations"] += len(
                    damage_events
                )
    return metrics


def utility_effect_correlation_counts(
    utility_events: list[dict],
) -> dict[str, int]:
    metrics = {
        "utility_observed_effect_correlations": 0,
        "utility_inferred_effect_correlations": 0,
        "utility_unavailable_effect_correlations": 0,
    }
    metric_by_status = {
        "observed": "utility_observed_effect_correlations",
        "inferred": "utility_inferred_effect_correlations",
        "unavailable": "utility_unavailable_effect_correlations",
    }
    for event in utility_events:
        affected = event.get("affected_players")
        if not isinstance(affected, list):
            continue
        for player in affected:
            if not isinstance(player, Mapping):
                continue
            blind = player.get("blind_duration")
            if isinstance(blind, Mapping) and blind.get("status") in {
                "observed",
                "unavailable",
            }:
                correlation = player.get("blind_correlation")
                status = (
                    correlation.get("status")
                    if isinstance(correlation, Mapping)
                    else None
                )
                if status in metric_by_status:
                    metrics[metric_by_status[status]] += 1
            damage_events = player.get("damage_events")
            if not isinstance(damage_events, list):
                continue
            for damage in damage_events:
                correlation = (
                    damage.get("correlation") if isinstance(damage, Mapping) else None
                )
                status = (
                    correlation.get("status")
                    if isinstance(correlation, Mapping)
                    else None
                )
                if status in metric_by_status:
                    metrics[metric_by_status[status]] += 1
    return metrics


def validate_utility_callback_diagnostics(
    value: object,
    report: Mapping[str, object],
    utility_events: list[dict],
    location: str,
    errors: list[str],
) -> None:
    diagnostics = validate_utility_mapping(
        value, UTILITY_CALLBACK_GROUPS, location, errors
    )
    if diagnostics is None:
        return
    groups: dict[str, dict[str, int]] = {}
    accounting_violations = 0
    for name in UTILITY_CALLBACK_GROUPS:
        group = validate_utility_mapping(
            diagnostics.get(name),
            UTILITY_CALLBACK_FIELDS,
            f"{location}.{name}",
            errors,
        )
        if group is None:
            continue
        parsed: dict[str, int] = {}
        for field in UTILITY_CALLBACK_FIELDS:
            item = group.get(field)
            if not isinstance(item, int) or isinstance(item, bool) or item < 0:
                errors.append(f"{location}.{name}: {field} debe ser entero no negativo")
                continue
            parsed[field] = item
        if len(parsed) != len(UTILITY_CALLBACK_FIELDS):
            continue
        accounted = sum(
            parsed[field] for field in UTILITY_CALLBACK_FIELDS if field != "observed"
        )
        if parsed["observed"] != accounted:
            accounting_violations += 1
            errors.append(f"{location}.{name}: callbacks no reconcilian")
        groups[name] = parsed
    if report.get("utility_callback_accounting_violations") != accounting_violations:
        errors.append(
            f"{location}: utility_callback_accounting_violations no reconcilia"
        )
    top_metric_by_group = {
        "throws": "utility_throw_callbacks",
        "bounces": "utility_bounce_callbacks",
        "lifecycle": "utility_lifecycle_callbacks",
        "player_flashed": "utility_player_flashed_callbacks",
        "damage": "utility_damage_callbacks",
    }
    for name, metric in top_metric_by_group.items():
        if name in groups and report.get(metric) != groups[name]["observed"]:
            errors.append(f"{location}: {metric} no coincide con diagnostics")
    aggregate_fields = {
        "utility_unmatched_callbacks": "unmatched",
        "utility_orphan_callbacks": "orphaned",
        "utility_inferred_callbacks": "inferred_correlated",
        "utility_deduplicated_callbacks": "deduplicated",
    }
    for metric, field in aggregate_fields.items():
        expected = sum(group[field] for group in groups.values())
        if (
            len(groups) == len(UTILITY_CALLBACK_GROUPS)
            and report.get(metric) != expected
        ):
            errors.append(f"{location}: {metric} no coincide con diagnostics")
    throws = groups.get("throws")
    if throws is not None:
        represented_launches = sum(
            isinstance(event.get("launch"), Mapping)
            and event["launch"].get("tick_status") == "observed"
            for event in utility_events
        )
        expected_launches = (
            throws["observed"] - throws["deduplicated"] - throws["unmatched"]
        )
        if represented_launches != expected_launches:
            errors.append(
                f"{location}.throws: launch ticks no reconcilian con callbacks"
            )


def validate_utility_quality_report(
    value: object,
    utility_events: list[dict],
    utility_replay_mismatches: int,
    errors: list[str],
) -> None:
    location = "canonical/diagnostics/quality_report.json"
    if not isinstance(value, Mapping):
        errors.append(f"{location}: report debe ser un objeto")
        return
    if value.get("schema_version") != QUALITY_SCHEMA_VERSION:
        errors.append(f"{location}: schema_version debe ser {QUALITY_SCHEMA_VERSION}")
    if value.get("parse_completed") is not True:
        errors.append(f"{location}: parse_completed debe ser true")
    if value.get("usable_for_training") is not True or value.get("status") not in {
        "pass",
        "warning",
    }:
        errors.append(f"{location}: el export no es utilizable para entrenamiento")
    checks = value.get("checks")
    if not isinstance(checks, list) or not all(
        isinstance(check, Mapping) for check in checks
    ):
        errors.append(f"{location}: checks debe ser una lista de objetos")
        return
    check_names = [
        check.get("name") for check in checks if isinstance(check.get("name"), str)
    ]
    if len(check_names) != len(checks):
        errors.append(f"{location}: cada check debe tener un name valido")
    if len(check_names) != len(set(check_names)):
        errors.append(f"{location}: checks contiene nombres duplicados")
    missing = sorted(REQUIRED_UTILITY_QUALITY_CHECKS - set(check_names))
    if missing:
        errors.append(
            f"{location}: faltan checks utility requeridos: {', '.join(missing)}"
        )
    checks_by_name = {
        check.get("name"): check
        for check in checks
        if isinstance(check.get("name"), str)
    }
    for name in REQUIRED_UTILITY_QUALITY_CHECKS - {"utility_observation_coverage"}:
        if checks_by_name.get(name, {}).get("status") != "pass":
            errors.append(f"{location}: el check {name} debe estar en pass")
    coverage_status = checks_by_name.get("utility_observation_coverage", {}).get(
        "status"
    )
    if coverage_status not in {"pass", "warning"}:
        errors.append(
            f"{location}: utility_observation_coverage debe ser pass o warning"
        )
    missing_metrics = sorted(REQUIRED_UTILITY_QUALITY_METRICS - value.keys())
    if missing_metrics:
        errors.append(
            f"{location}: faltan metricas utility requeridas: {', '.join(missing_metrics)}"
        )
    for metric in REQUIRED_UTILITY_QUALITY_METRICS:
        metric_value = value.get(metric)
        if (
            not isinstance(metric_value, int)
            or isinstance(metric_value, bool)
            or metric_value < 0
        ):
            errors.append(f"{location}: {metric} debe ser entero no negativo")
    for metric in HARD_UTILITY_QUALITY_METRICS:
        if value.get(metric) != 0:
            errors.append(f"{location}: {metric} debe ser cero")
    validate_utility_callback_diagnostics(
        value.get("utility_callback_diagnostics"),
        value,
        utility_events,
        f"{location}.utility_callback_diagnostics",
        errors,
    )
    flash_effects = 0
    damage_effects = 0
    for event in utility_events:
        affected = event.get("affected_players")
        if not isinstance(affected, list):
            continue
        for player in affected:
            if not isinstance(player, Mapping):
                continue
            blind = player.get("blind_duration")
            if isinstance(blind, Mapping) and blind.get("status") in {
                "observed",
                "unavailable",
            }:
                flash_effects += 1
            damage = player.get("damage_events")
            if isinstance(damage, list):
                damage_effects += len(damage)
    expected_counts = {
        "utility_throws": len(utility_events),
        "utility_canonical_events": len(utility_events),
        "utility_flash_effects": flash_effects,
        "utility_damage_effects": damage_effects,
        "utility_replay_projection_mismatches": utility_replay_mismatches,
        **utility_observation_metric_counts(utility_events),
        **utility_effect_correlation_counts(utility_events),
    }
    for metric, expected in expected_counts.items():
        if value.get(metric) != expected:
            errors.append(f"{location}: {metric} no coincide con el artefacto canonico")
    warning_parts = (
        "utility_missing_type_observations",
        "utility_missing_actor_observations",
        "utility_missing_launch_tick_observations",
        "utility_missing_launch_position_observations",
        "utility_missing_launch_view_observations",
        "utility_missing_thrower_velocity_observations",
        "utility_missing_projectile_velocity_observations",
        "utility_missing_trajectory_observations",
        "utility_missing_lifecycle_observations",
        "utility_missing_affected_player_observations",
        "utility_missing_flash_duration_observations",
        "utility_inferred_correlations",
        "utility_orphan_callbacks",
        "utility_inferred_callbacks",
        "utility_inferred_effect_correlations",
        "utility_unavailable_effect_correlations",
    )
    warning_count = sum(
        int(value.get(metric, 0))
        for metric in warning_parts
        if isinstance(value.get(metric, 0), int)
        and not isinstance(value.get(metric, 0), bool)
    )
    if value.get("utility_observation_warnings") != warning_count:
        errors.append(f"{location}: utility_observation_warnings no reconcilia")
    expected_coverage_status = "warning" if warning_count > 0 else "pass"
    if coverage_status != expected_coverage_status:
        errors.append(
            f"{location}: utility_observation_coverage no coincide con sus metricas"
        )
    if coverage_status == "warning" and value.get("status") != "warning":
        errors.append(f"{location}: status global debe reflejar warning de utility")
    if "grenade_events" in value and value.get("grenade_events") != len(utility_events):
        errors.append(
            f"{location}: grenade_events no coincide con utility_events.jsonl"
        )


def validate_combat_semantics(
    match_id: str,
    participants: list[dict],
    combat_events: list[dict],
    player_stats: list[dict],
    replay_rounds: list[dict],
    errors: list[str],
) -> dict[str, int]:
    location = "canonical/events/combat_events.jsonl"
    player_ids = {
        player.get("player_id")
        for player in participants
        if isinstance(player.get("player_id"), str)
    }
    by_id = {
        event.get("event_id"): event
        for event in combat_events
        if isinstance(event.get("event_id"), str)
    }
    index_by_id = {event.get("event_id"): index for index, event in enumerate(combat_events)}
    shots: dict[str, dict] = {}
    previous_round = previous_tick = previous_round_sequence = previous_tick_sequence = None
    contract_violations = 0

    def violation(message: str) -> None:
        nonlocal contract_violations
        contract_violations += 1
        errors.append(f"{location}: {message}")

    for index, event in enumerate(combat_events):
        event_type = event.get("event_type")
        if event.get("schema_id") != "stratai.combat_event@2":
            violation(f"registro {index} schema_id invalido")
        if event.get("match_id") != match_id or event_type not in COMBAT_EVENT_TYPES:
            violation(f"registro {index} identidad o event_type invalido")
        round_number, tick = event.get("round_number"), event.get("tick")
        sequence_round, sequence_tick = event.get("sequence_in_round"), event.get("sequence_in_tick")
        if not all(isinstance(value, int) and not isinstance(value, bool) for value in (round_number, tick, sequence_round, sequence_tick)):
            violation(f"registro {index} tiempo o secuencia no entero")
        else:
            expected_round_sequence = 1 if round_number != previous_round else previous_round_sequence + 1
            expected_tick_sequence = 1 if (round_number, tick) != (previous_round, previous_tick) else previous_tick_sequence + 1
            if sequence_round != expected_round_sequence or sequence_tick != expected_tick_sequence:
                violation(f"registro {index} secuencia no contigua")
            if previous_round is not None and (round_number, tick, sequence_tick) < (previous_round, previous_tick, previous_tick_sequence):
                violation(f"registro {index} fuera de orden causal")
            previous_round, previous_tick = round_number, tick
            previous_round_sequence, previous_tick_sequence = sequence_round, sequence_tick
        if event.get("tick_status") != "observed" or event.get("subtick") is not None or event.get("subtick_status") != "unavailable":
            violation(f"registro {index} precision temporal invalida")
        time_seconds = event.get("time_seconds")
        if event.get("time_seconds_status") == "derived":
            if not is_finite_number(time_seconds) or time_seconds < 0:
                violation(f"registro {index} time_seconds invalido")
        elif event.get("time_seconds_status") != "unavailable" or time_seconds is not None:
            violation(f"registro {index} disponibilidad temporal invalida")

        for prefix in ("actor", "target", "assister"):
            status = event.get(f"{prefix}_status")
            player_id = event.get(f"{prefix}_player_id")
            source = event.get(f"{prefix}_source")
            if status == "observed":
                if player_id not in player_ids or source in {None, "", "unavailable"}:
                    violation(f"registro {index} {prefix} observado invalido")
            elif status == "unavailable":
                if player_id is not None or event.get(f"{prefix}_side") is not None or source != "unavailable":
                    violation(f"registro {index} {prefix} unavailable invalido")
            else:
                violation(f"registro {index} {prefix}_status invalido")
        relation = event.get("relation")
        expected_relation = combat_relation(event)
        if relation not in COMBAT_RELATIONS or relation != expected_relation:
            violation(f"registro {index} relation no reconcilia")

        validate_combat_observation(event, "weapon", errors, location, index, scalar=True)
        validate_combat_observation(event, "actor_position", errors, location, index)
        validate_combat_observation(event, "target_position", errors, location, index)
        validate_combat_observation(event, "impact_position", errors, location, index, allow_derived=True)
        if event.get("weapon_status") == "observed" and not isinstance(event.get("weapon_is_utility"), bool):
            violation(f"registro {index} weapon_is_utility debe ser booleano factual")
        if event.get("weapon_status") == "unavailable" and event.get("weapon_is_utility") is not None:
            violation(f"registro {index} weapon_is_utility debe ser null")
        for field in (
            "view_yaw", "view_pitch", "bullet_distance_world_units", "kill_distance_world_units"
        ):
            value = event.get(field)
            if value is not None and not is_finite_number(value):
                violation(f"registro {index} {field} no finito")
        for field in ("damage_direction",):
            value = event.get(field)
            if value is not None and not is_finite_combat_vector(value):
                violation(f"registro {index} {field} no finito")

        source_ids = event.get("source_event_ids")
        if not isinstance(source_ids, list) or len(source_ids) != len(set(source_ids)):
            violation(f"registro {index} source_event_ids invalido")
        else:
            for source_id in source_ids:
                source = by_id.get(source_id)
                if source is None or index_by_id.get(source_id, index + 1) > index or (
                    source.get("round_number"), source.get("tick")
                ) > (round_number, tick):
                    violation(f"registro {index} referencia causal invalida")

        shot_id = event.get("shot_id")
        if event_type == "weapon_fire":
            if not isinstance(shot_id, str) or not shot_id:
                violation(f"registro {index} fire sin shot_id")
            elif shot_id in shots:
                violation(f"registro {index} shot_id duplicado")
            else:
                shots[shot_id] = event
            result, result_status = event.get("shot_result"), event.get("shot_result_status")
            available_tick = event.get("shot_result_availability_tick")
            if result_status == "derived":
                if result not in {"hit", "miss"} or not isinstance(available_tick, int) or available_tick < tick:
                    violation(f"registro {index} outcome de disparo invalido")
                if result == "miss" and event.get("shot_result_source") != "round_end_observed":
                    violation(f"registro {index} miss sin cierre de ronda")
            elif result_status != "unavailable" or result is not None or available_tick is not None:
                violation(f"registro {index} outcome unavailable invalido")
        else:
            if shot_id is not None and shot_id not in shots:
                violation(f"registro {index} referencia shot desconocido o futuro")
            if event.get("shot_result") is not None or event.get("shot_result_status") != "unavailable":
                violation(f"registro {index} outcome solo permitido en fire")
        if event.get("correlation_status") not in COMBAT_CORRELATIONS:
            violation(f"registro {index} correlation_status invalido")

        damage_fields = (
            "health_damage", "health_damage_taken", "armor_damage", "armor_damage_taken",
            "health_before", "health_after", "armor_before", "armor_after",
        )
        if event_type == "player_hurt":
            values = [event.get(field) for field in damage_fields]
            if any(not isinstance(value, int) or isinstance(value, bool) or value < 0 for value in values):
                violation(f"registro {index} dano factual incompleto")
            elif values[1] != values[4] - values[5] or values[3] != values[6] - values[7]:
                violation(f"registro {index} before/after no reconcilia")
            if event.get("damage_status") != "observed" or event.get("is_kill") is not False:
                violation(f"registro {index} semantica player_hurt invalida")
            hitgroup = event.get("hitgroup")
            hitgroup_known = hitgroup in COMBAT_HITGROUPS or (
                isinstance(hitgroup, str)
                and hitgroup.startswith("unknown_")
                and hitgroup.removeprefix("unknown_").isdigit()
            )
            if event.get("hitgroup_status") == "observed":
                if not hitgroup_known or event.get("is_headshot") != (hitgroup == "head"):
                    violation(f"registro {index} hitgroup/headshot no reconcilia")
            elif event.get("hitgroup_status") != "unavailable" or hitgroup is not None:
                violation(f"registro {index} hitgroup availability invalida")
        else:
            if any(event.get(field) is not None for field in damage_fields) or event.get("damage_status") != "unavailable":
                violation(f"registro {index} dano inventado fuera de player_hurt")
        if event_type == "kill":
            if event.get("is_kill") is not True or not isinstance(event.get("assisted_flash"), bool):
                violation(f"registro {index} semantica kill invalida")
        elif event.get("is_kill") is not False or event.get("assisted_flash") is not None:
            violation(f"registro {index} resultado kill fuera de kill")
        if event_type == "weapon_reload":
            if event.get("reload_phase") != "start" or event.get("reload_end_tick") is not None or event.get("reload_end_status") != "unavailable":
                violation(f"registro {index} lifecycle reload invalido")
        elif event.get("reload_phase") is not None:
            violation(f"registro {index} reload_phase fuera de reload")
        if event_type == "weapon_equip":
            previous_status = event.get("previous_weapon_status")
            if previous_status == "derived":
                if not isinstance(event.get("previous_weapon"), str) or not isinstance(event.get("is_weapon_switch"), bool):
                    violation(f"registro {index} previous_weapon derivado invalido")
            elif previous_status != "unavailable" or event.get("previous_weapon") is not None or event.get("is_weapon_switch") is not None:
                violation(f"registro {index} previous_weapon unavailable invalido")

    for event in combat_events:
        shot_id = event.get("shot_id")
        if shot_id is not None and shot_id not in shots:
            violation("shot_id no referencia un weapon_fire")
    hurt_shots = {
        event.get("shot_id")
        for event in combat_events
        if event.get("event_type") == "player_hurt"
        and isinstance(event.get("health_damage_taken"), int)
        and event["health_damage_taken"] > 0
    }
    for shot_id, fire in shots.items():
        if fire.get("shot_result") == "hit" and shot_id not in hurt_shots:
            violation("shot_result hit carece de player_hurt correlacionado")
        if fire.get("shot_result") == "miss" and shot_id in hurt_shots:
            violation("shot_result miss contradice player_hurt")

    expected_stats = aggregate_combat_stats(combat_events, player_stats)
    stats_mismatches, native_delta_mismatches = compare_combat_stats(
        expected_stats, player_stats, errors
    )
    replay_mismatches = validate_replay_combat_projection(
        combat_events, replay_rounds, errors
    )
    return {
        "contract_violations": contract_violations,
        "player_stats_mismatches": stats_mismatches,
        "native_delta_mismatches": native_delta_mismatches,
        "replay_projection_mismatches": replay_mismatches,
        "missing_impact_positions": sum(
            event.get("event_type") == "bullet_damage"
            and event.get("impact_position_status") != "observed"
            for event in combat_events
        ),
        "missing_reload_ends": sum(
            event.get("event_type") == "weapon_reload"
            and event.get("reload_end_status") != "observed"
            for event in combat_events
        ),
        "unavailable_shot_results": sum(
            event.get("event_type") == "weapon_fire"
            and event.get("shot_result_status") == "unavailable"
            for event in combat_events
        ),
    }


def combat_relation(event: Mapping[str, object]) -> str:
    target_status, actor_status = event.get("target_status"), event.get("actor_status")
    if target_status != "observed":
        return "unknown"
    if actor_status != "observed":
        return "world"
    actor_id, target_id = event.get("actor_player_id"), event.get("target_player_id")
    if actor_id == target_id:
        return "self"
    actor_side, target_side = event.get("actor_side"), event.get("target_side")
    if not actor_side or not target_side:
        return "unknown"
    return "friendly" if actor_side == target_side else "enemy"


def validate_combat_observation(
    event: Mapping[str, object],
    field: str,
    errors: list[str],
    location: str,
    index: int,
    *,
    scalar: bool = False,
    allow_derived: bool = False,
) -> None:
    value, status, source = event.get(field), event.get(f"{field}_status"), event.get(f"{field}_source")
    available = {"observed", "derived"} if allow_derived else {"observed"}
    if status in available:
        valid = isinstance(value, str) and bool(value) if scalar else is_finite_combat_vector(value)
        if not valid or source in {None, "", "unavailable"}:
            errors.append(f"{location}: registro {index} {field} observado invalido")
    elif status == "unavailable":
        if value is not None or source != "unavailable":
            errors.append(f"{location}: registro {index} {field} unavailable invalido")
    else:
        errors.append(f"{location}: registro {index} {field}_status invalido")


def is_finite_combat_vector(value: object) -> bool:
    return isinstance(value, Mapping) and all(
        is_finite_number(value.get(axis)) for axis in ("x", "y", "z")
    )


def empty_combat_summary() -> dict:
    return {
        "kills_observed": 0,
        "deaths_observed": 0,
        "assists_observed": 0,
        "flash_assists": 0,
        "headshots": 0,
        "combat_damage_observed": 0,
        "friendly_damage": 0,
        "self_damage": 0,
        "shots_fired": 0,
        "shots_hit": 0,
        "shots_missed": 0,
        "body_part_hits": {},
        "weapon_stats": {},
    }


def aggregate_combat_stats(combat_events: list[dict], player_stats: list[dict]) -> dict[str, dict]:
    summaries = {
        record.get("player_id"): empty_combat_summary()
        for record in player_stats
        if isinstance(record.get("player_id"), str)
    }
    for event in combat_events:
        for field in ("actor_player_id", "target_player_id", "assister_player_id"):
            player_id = event.get(field)
            if isinstance(player_id, str):
                summaries.setdefault(player_id, empty_combat_summary())
    damage_groups: dict[tuple, dict] = {}
    for event in combat_events:
        actor_id, target_id = event.get("actor_player_id"), event.get("target_player_id")
        event_type = event.get("event_type")
        if event_type == "weapon_fire" and isinstance(actor_id, str) and event.get("weapon_is_utility") is not True:
            summary = summaries[actor_id]
            summary["shots_fired"] += 1
            weapon = event.get("weapon") or ""
            weapon_stats = summary["weapon_stats"].setdefault(weapon, empty_weapon_summary())
            weapon_stats["shots_fired"] += 1
            if event.get("shot_result") == "hit":
                summary["shots_hit"] += 1
                weapon_stats["shots_hit"] += 1
            elif event.get("shot_result") == "miss":
                summary["shots_missed"] += 1
                weapon_stats["shots_missed"] += 1
        if event_type == "kill":
            if isinstance(target_id, str):
                summaries[target_id]["deaths_observed"] += 1
            if event.get("relation") != "enemy" or not isinstance(actor_id, str):
                continue
            summary = summaries[actor_id]
            summary["kills_observed"] += 1
            if event.get("is_headshot") is True:
                summary["headshots"] += 1
            if event.get("weapon_status") == "observed" and event.get("weapon_is_utility") is not True:
                weapon_stats = summary["weapon_stats"].setdefault(event.get("weapon") or "", empty_weapon_summary())
                weapon_stats["kills"] += 1
                if event.get("is_headshot") is True:
                    weapon_stats["headshots"] += 1
            assister_id = event.get("assister_player_id")
            if (
                isinstance(assister_id, str)
                and assister_id not in {actor_id, target_id}
                and event.get("assister_side")
                and event.get("assister_side") == event.get("actor_side")
            ):
                summaries[assister_id]["assists_observed"] += 1
                if event.get("assisted_flash") is True:
                    summaries[assister_id]["flash_assists"] += 1
        if event_type == "player_hurt" and isinstance(actor_id, str):
            key = (
                event.get("round_number"), event.get("tick"), actor_id, target_id,
                event.get("weapon"), event.get("relation"),
            )
            group = damage_groups.setdefault(
                key,
                {"max_before": None, "min_after": None, "fallback": 0,
                 "weapon_observed": event.get("weapon_status") == "observed",
                 "is_utility": event.get("weapon_is_utility") is True},
            )
            before, after, taken = event.get("health_before"), event.get("health_after"), event.get("health_damage_taken")
            if isinstance(before, int):
                group["max_before"] = before if group["max_before"] is None else max(group["max_before"], before)
            if isinstance(after, int):
                group["min_after"] = after if group["min_after"] is None else min(group["min_after"], after)
            if isinstance(taken, int):
                group["fallback"] += taken
            if event.get("relation") == "enemy" and isinstance(taken, int) and taken > 0 and isinstance(event.get("hitgroup"), str):
                hits = summaries[actor_id]["body_part_hits"]
                hits[event["hitgroup"]] = hits.get(event["hitgroup"], 0) + 1
    for key, group in damage_groups.items():
        _, _, actor_id, _, weapon, relation = key
        damage = group["fallback"]
        if group["max_before"] is not None and group["min_after"] is not None:
            damage = max(0, group["max_before"] - group["min_after"])
        summary = summaries[actor_id]
        metric = {"enemy": "combat_damage_observed", "friendly": "friendly_damage", "self": "self_damage"}.get(relation)
        if metric:
            summary[metric] += damage
        if relation == "enemy" and group["weapon_observed"] and not group["is_utility"]:
            summary["weapon_stats"].setdefault(weapon or "", empty_weapon_summary())["damage"] += damage
    return summaries


def empty_weapon_summary() -> dict[str, int]:
    return {"kills": 0, "headshots": 0, "damage": 0, "shots_fired": 0, "shots_hit": 0, "shots_missed": 0}


def compare_combat_stats(expected: dict[str, dict], player_stats: list[dict], errors: list[str]) -> tuple[int, int]:
    location = "canonical/derived/player_stats.json"
    actual = {record.get("player_id"): record.get("metrics") for record in player_stats}
    mismatches = native_mismatches = 0
    for player_id, want in expected.items():
        metrics = actual.get(player_id)
        if not isinstance(metrics, Mapping):
            mismatches += 1
            errors.append(f"{location}: falta projection de combate para {player_id}")
            continue
        scalar_fields = tuple(field for field in want if field not in {"body_part_hits", "weapon_stats"})
        different = any(metrics.get(field) != want[field] for field in scalar_fields)
        different |= metrics.get("body_part_hits") != want["body_part_hits"]
        weapon_stats = metrics.get("weapon_stats")
        if not isinstance(weapon_stats, Mapping) or set(weapon_stats) != set(want["weapon_stats"]):
            different = True
        else:
            different |= any(
                any(weapon_stats[weapon].get(field) != value for field, value in expected_weapon.items())
                for weapon, expected_weapon in want["weapon_stats"].items()
            )
        if different:
            mismatches += 1
            errors.append(f"{location}: stats de combate no reconcilian para {player_id}")
        native = metrics.get("native_scoreboard")
        if not isinstance(native, Mapping) or any(
            metrics.get(delta) != native.get(native_field, 0) - want[observed]
            for delta, native_field, observed in (
                ("kills_native_minus_observed", "kills", "kills_observed"),
                ("deaths_native_minus_observed", "deaths", "deaths_observed"),
                ("assists_native_minus_observed", "assists", "assists_observed"),
                ("combat_damage_unattributed_delta", "total_damage", "combat_damage_observed"),
            )
        ):
            native_mismatches += 1
            errors.append(f"{location}: deltas nativos no reconcilian para {player_id}")
    return mismatches, native_mismatches


def validate_replay_combat_projection(combat_events: list[dict], replay_rounds: list[dict], errors: list[str]) -> int:
    location = "canonical/presentation/replay"
    markers = {
        event["event_id"]: event
        for event in combat_events
        if event.get("event_type") in {"player_hurt", "kill"}
    }
    fires = {
        event["event_id"]: event
        for event in combat_events
        if event.get("event_type") == "weapon_fire"
    }
    marker_counts: dict[str, int] = {}
    fire_counts: dict[str, int] = {}
    mismatches = 0
    for envelope in replay_rounds:
        round_data = envelope.get("round") if isinstance(envelope, Mapping) else None
        if not isinstance(round_data, Mapping):
            continue
        round_number = round_data.get("round")
        for marker in round_data.get("events", []):
            if not isinstance(marker, Mapping) or marker.get("type") not in {"player_hurt", "kill"}:
                continue
            source_ids = marker.get("source_event_ids")
            source = markers.get(source_ids[0]) if isinstance(source_ids, list) and len(source_ids) == 1 else None
            if source is None or source.get("round_number") != round_number or source.get("event_type") != marker.get("type"):
                mismatches += 1
                errors.append(f"{location}: marker de combate con provenance invalida")
                continue
            expected_actor = str(source.get("actor_player_id") or "").removeprefix("steam:") or None
            expected_target = str(source.get("target_player_id") or "").removeprefix("steam:") or None
            expected_assister = str(source.get("assister_player_id") or "").removeprefix("steam:") or None
            if (
                marker.get("killer_id") != expected_actor
                or marker.get("victim_id") != expected_target
                or marker.get("assister_id") != expected_assister
                or (str(marker.get("killer_team") or "").lower() or None) != source.get("actor_side")
                or (str(marker.get("victim_team") or "").lower() or None) != source.get("target_side")
                or (str(marker.get("assister_team") or "").lower() or None) != source.get("assister_side")
                or marker.get("weapon") != source.get("weapon")
                or bool(marker.get("headshot", False)) != bool(source.get("is_headshot", False))
                or (
                    source.get("event_type") == "player_hurt"
                    and marker.get("damage", 0) != source.get("health_damage_taken")
                )
            ):
                mismatches += 1
                errors.append(f"{location}: marker de combate no reconcilia con source")
            marker_counts[source["event_id"]] = marker_counts.get(source["event_id"], 0) + 1
        combat_shots = round_data.get("combat_shots")
        if not isinstance(combat_shots, list):
            mismatches += 1
            errors.append(f"{location}: combat_shots debe ser una lista")
            combat_shots = []
        for shot in combat_shots:
            if not isinstance(shot, Mapping):
                mismatches += 1
                continue
            source = fires.get(shot.get("source_event_id"))
            actor_player_id = source.get("actor_player_id") if source is not None else None
            expected_shooter_id = (
                str(actor_player_id).removeprefix("steam:")
                if actor_player_id
                else "0"
                if source is not None
                and source.get("actor_status") == "unavailable"
                and source.get("actor_source") == "unavailable"
                else ""
            )
            if (
                source is None
                or source.get("round_number") != round_number
                or source.get("shot_id") != shot.get("shot_id")
                or source.get("tick") != shot.get("tick")
                or shot.get("shooter_id") != expected_shooter_id
                or shot.get("weapon") != source.get("weapon")
                or shot.get("position_status") != source.get("actor_position_status")
                or shot.get("position_source") != source.get("actor_position_source")
            ):
                mismatches += 1
                errors.append(f"{location}: combat_shot con provenance invalida")
                continue
            endpoint_status = shot.get("endpoint_status")
            endpoint_source = shot.get("endpoint_source")
            expected_result = source.get("shot_result") or "unavailable"
            if (
                (endpoint_status == "derived" and endpoint_source != "view_direction_projection")
                or (endpoint_status == "unavailable" and endpoint_source != "unavailable")
                or endpoint_status not in {"derived", "unavailable"}
                or shot.get("result") != expected_result
                or shot.get("result_status") != source.get("shot_result_status")
                or bool(shot.get("hit", False)) != (source.get("shot_result") == "hit")
            ):
                mismatches += 1
                errors.append(f"{location}: combat_shot no reconcilia con source")
            fire_counts[source["event_id"]] = fire_counts.get(source["event_id"], 0) + 1
        for frame in round_data.get("frames", []):
            if not isinstance(frame, Mapping):
                continue
            for shot in frame.get("shots", []):
                if not isinstance(shot, Mapping):
                    mismatches += 1
                    continue
                source = fires.get(shot.get("source_event_id"))
                actor_player_id = source.get("actor_player_id") if source is not None else None
                expected_shooter_id = (
                    str(actor_player_id).removeprefix("steam:")
                    if actor_player_id
                    else "0"
                    if source is not None
                    and source.get("actor_status") == "unavailable"
                    and source.get("actor_source") == "unavailable"
                    else ""
                )
                if (
                    source is None
                    or source.get("round_number") != round_number
                    or source.get("shot_id") != shot.get("shot_id")
                    or source.get("tick") != shot.get("tick")
                    or shot.get("shooter_id") != expected_shooter_id
                    or shot.get("weapon") != source.get("weapon")
                    or shot.get("position_status") != source.get("actor_position_status")
                    or shot.get("position_source") != source.get("actor_position_source")
                    or not isinstance(frame.get("tick"), int)
                    or frame["tick"] < shot.get("tick", frame["tick"] + 1)
                ):
                    mismatches += 1
                    errors.append(f"{location}: shot con provenance invalida")
                    continue
                endpoint_status = shot.get("endpoint_status")
                endpoint_source = shot.get("endpoint_source")
                if (
                    (endpoint_status == "derived" and endpoint_source != "view_direction_projection")
                    or (endpoint_status == "unavailable" and endpoint_source != "unavailable")
                    or endpoint_status not in {"derived", "unavailable"}
                ):
                    mismatches += 1
                    errors.append(f"{location}: endpoint derivado mal etiquetado")
                expected_result = source.get("shot_result") or "unavailable"
                if (
                    shot.get("result") != expected_result
                    or shot.get("result_status") != source.get("shot_result_status")
                    or bool(shot.get("hit", False)) != (source.get("shot_result") == "hit")
                ):
                    mismatches += 1
                    errors.append(f"{location}: outcome de shot no reconcilia")
    for event_id in markers:
        if marker_counts.get(event_id) != 1:
            mismatches += 1
            errors.append(f"{location}: marker {event_id} omitido o duplicado")
    for event_id in fires:
        if fire_counts.get(event_id) != 1:
            mismatches += 1
            errors.append(f"{location}: fire {event_id} omitido o duplicado en combat_shots")
    return mismatches


def validate_combat_quality_report(
    value: object,
    combat_events: list[dict],
    combat_metrics: Mapping[str, int],
    errors: list[str],
) -> None:
    location = "canonical/diagnostics/quality_report.json"
    if not isinstance(value, Mapping):
        return
    checks = value.get("checks")
    if not isinstance(checks, list):
        return
    checks_by_name = {
        check.get("name"): check
        for check in checks
        if isinstance(check, Mapping) and isinstance(check.get("name"), str)
    }
    missing_checks = sorted(REQUIRED_COMBAT_QUALITY_CHECKS - checks_by_name.keys())
    if missing_checks:
        errors.append(f"{location}: faltan checks combat requeridos: {', '.join(missing_checks)}")
    for name in REQUIRED_COMBAT_QUALITY_CHECKS - {"combat_observation_coverage"}:
        if checks_by_name.get(name, {}).get("status") != "pass":
            errors.append(f"{location}: el check {name} debe estar en pass")
    coverage_status = checks_by_name.get("combat_observation_coverage", {}).get("status")
    expected_coverage = "warning" if any(
        combat_metrics[name] > 0
        for name in ("missing_impact_positions", "missing_reload_ends", "unavailable_shot_results")
    ) or value.get("combat_discarded_callbacks", 0) > 0 else "pass"
    if coverage_status != expected_coverage:
        errors.append(f"{location}: combat_observation_coverage no reconcilia")
    missing_metrics = sorted(REQUIRED_COMBAT_QUALITY_METRICS - value.keys())
    if missing_metrics:
        errors.append(f"{location}: faltan metricas combat: {', '.join(missing_metrics)}")
    for metric in REQUIRED_COMBAT_QUALITY_METRICS:
        metric_value = value.get(metric)
        if not isinstance(metric_value, int) or isinstance(metric_value, bool) or metric_value < 0:
            errors.append(f"{location}: {metric} debe ser entero no negativo")
    for metric in HARD_COMBAT_QUALITY_METRICS:
        if value.get(metric) != 0:
            errors.append(f"{location}: {metric} debe ser cero")
    expected = {
        "combat_ledger_events": len(combat_events),
        "combat_contract_violations": combat_metrics["contract_violations"],
        "combat_player_stats_mismatches": combat_metrics["player_stats_mismatches"],
        "combat_replay_projection_mismatches": combat_metrics["replay_projection_mismatches"],
        "combat_native_delta_mismatches": combat_metrics["native_delta_mismatches"],
        "combat_missing_impact_positions": combat_metrics["missing_impact_positions"],
        "combat_missing_reload_ends": combat_metrics["missing_reload_ends"],
        "combat_unavailable_shot_results": combat_metrics["unavailable_shot_results"],
    }
    for metric, expected_value in expected.items():
        if value.get(metric) != expected_value:
            errors.append(f"{location}: {metric} no coincide con artefactos")
    diagnostics = value.get("combat_callback_diagnostics")
    discarded_total = 0
    if not isinstance(diagnostics, Mapping) or set(diagnostics) != set(COMBAT_CALLBACK_GROUPS):
        errors.append(f"{location}: combat_callback_diagnostics invalido")
    else:
        accounting = 0
        for group in COMBAT_CALLBACK_GROUPS:
            entry = diagnostics.get(group)
            if not isinstance(entry, Mapping) or any(
                not isinstance(entry.get(field), int) or isinstance(entry.get(field), bool) or entry.get(field) < 0
                for field in ("observed", "recorded", "discarded")
            ):
                errors.append(f"{location}: diagnostics {group} invalido")
                continue
            discarded_total += entry["discarded"]
            if entry["observed"] != entry["recorded"] + entry["discarded"]:
                accounting += 1
        reasons = value.get("combat_discarded_callback_reasons")
        allowed_reasons = {"warmup", "outside_official_round", "invalid_round_or_tick"}
        if not isinstance(reasons, Mapping) or any(
            reason not in allowed_reasons
            or not isinstance(count, int)
            or isinstance(count, bool)
            or count < 0
            for reason, count in reasons.items()
        ):
            errors.append(f"{location}: combat_discarded_callback_reasons invalido")
            accounting += 1
        elif sum(reasons.values()) != discarded_total:
            errors.append(f"{location}: combat_discarded_callback_reasons no reconcilia")
            accounting += 1
        if value.get("combat_discarded_callbacks") != discarded_total:
            errors.append(f"{location}: combat_discarded_callbacks no reconcilia")
        if value.get("combat_callback_accounting_violations") != accounting:
            errors.append(f"{location}: combat_callback_accounting_violations no reconcilia")


def validate_engagement_quality_report(
    value: object,
    engagement_metrics: Mapping[str, int],
    errors: list[str],
) -> None:
    location = "canonical/diagnostics/quality_report.json"
    if not isinstance(value, Mapping):
        return
    checks = value.get("checks")
    if not isinstance(checks, list) or not all(isinstance(check, Mapping) for check in checks):
        return
    checks_by_name = {
        check.get("name"): check
        for check in checks
        if isinstance(check.get("name"), str)
    }
    missing_checks = sorted(REQUIRED_ENGAGEMENT_QUALITY_CHECKS - checks_by_name.keys())
    if missing_checks:
        errors.append(
            f"{location}: faltan checks engagement requeridos: {', '.join(missing_checks)}"
        )
    for name in REQUIRED_ENGAGEMENT_QUALITY_CHECKS - {"engagement_observation_coverage"}:
        if checks_by_name.get(name, {}).get("status") != "pass":
            errors.append(f"{location}: el check {name} debe estar en pass")
    coverage = checks_by_name.get("engagement_observation_coverage", {})
    if coverage.get("status") not in {"pass", "warning"}:
        errors.append(
            f"{location}: engagement_observation_coverage debe ser pass o warning"
        )
    required_metrics = HARD_ENGAGEMENT_QUALITY_METRICS | {
        "engagements",
        "trade_candidates",
        "trade_completions",
        "engagement_observation_warnings",
    }
    missing_metrics = sorted(required_metrics - value.keys())
    if missing_metrics:
        errors.append(
            f"{location}: faltan metricas engagement requeridas: {', '.join(missing_metrics)}"
        )
    for metric in required_metrics:
        metric_value = value.get(metric)
        if (
            not isinstance(metric_value, int)
            or isinstance(metric_value, bool)
            or metric_value < 0
        ):
            errors.append(f"{location}: {metric} debe ser entero no negativo")
    for metric in HARD_ENGAGEMENT_QUALITY_METRICS:
        if value.get(metric) != 0:
            errors.append(f"{location}: {metric} debe ser cero")
    expected = {
        "engagements": engagement_metrics.get("engagements", 0),
        "trade_candidates": engagement_metrics.get("trade_candidates", 0),
        "trade_completions": engagement_metrics.get("trade_completions", 0),
        "engagement_observation_warnings": engagement_metrics.get(
            "observation_warnings", 0
        ),
    }
    for metric, count in expected.items():
        if value.get(metric) != count:
            errors.append(f"{location}: {metric} no coincide con artefactos")
    warning_count = expected["engagement_observation_warnings"]
    expected_status = "warning" if warning_count > 0 else "pass"
    if coverage.get("status") != expected_status:
        errors.append(
            f"{location}: engagement_observation_coverage no reconcilia con metricas"
        )
    if coverage.get("actual") != str(warning_count):
        errors.append(
            f"{location}: engagement_observation_coverage.actual no reconcilia"
        )
    if warning_count > 0 and value.get("status") != "warning":
        errors.append(f"{location}: status global debe reflejar warning de engagement")


def validate_block6_price(price: object, location: str, errors: list[str]) -> None:
    if not isinstance(price, Mapping):
        errors.append(f"{location}: price debe ser objeto")
        return
    amount = price.get("amount")
    status = price.get("status")
    if price.get("table_version") != "stratai.cs2_prices@1":
        errors.append(f"{location}: falta version de tabla de precios")
    if status == "unknown" and amount is not None:
        errors.append(f"{location}: precio unknown debe permanecer null")
    elif status == "known_zero" and amount != 0:
        errors.append(f"{location}: known_zero debe ser cero real")
    elif status == "known" and (not isinstance(amount, int) or amount <= 0):
        errors.append(f"{location}: precio conocido debe ser positivo")
    elif status not in {"unknown", "known_zero", "known"}:
        errors.append(f"{location}: price.status invalido")


def validate_block6_money_value(value: object, location: str, errors: list[str]) -> None:
    if not isinstance(value, Mapping):
        errors.append(f"{location}: valor monetario debe ser objeto")
        return
    status = value.get("status")
    amount = value.get("amount")
    if status not in {"observed", "calculated", "unavailable", "not_observed", "not_evaluable"}:
        errors.append(f"{location}: status monetario invalido")
    if status in {"unavailable", "not_observed", "not_evaluable"} and amount is not None:
        errors.append(f"{location}: dato no disponible debe ser null")
    if amount is not None and (not isinstance(amount, int) or not 0 <= amount <= 16000):
        errors.append(f"{location}: dinero fuera de 0..16000")


def expected_block6_clutches(
    participants: list[dict], combat_events: list[dict]
) -> dict[tuple[int, str], dict[str, object]]:
    player_team = {
        str(player.get("player_id")): str(player.get("team_id"))
        for player in participants
    }
    team_players: dict[str, set[str]] = {}
    for player_id, team_id in player_team.items():
        team_players.setdefault(team_id, set()).add(player_id)
    events_by_round: dict[int, list[dict]] = {}
    for event in combat_events:
        if event.get("is_kill") is True and event.get("target_player_id") in player_team:
            events_by_round.setdefault(int(event.get("round_number") or 0), []).append(event)
    expected: dict[tuple[int, str], dict[str, object]] = {}
    for round_number, events in events_by_round.items():
        alive = {team_id: set(players) for team_id, players in team_players.items()}
        attempted: set[str] = set()
        events.sort(
            key=lambda event: (
                int(event.get("tick") or 0),
                int(event.get("sequence_in_tick") or 0),
                str(event.get("event_id") or ""),
            )
        )
        for event in events:
            victim_id = str(event.get("target_player_id"))
            victim_team = player_team[victim_id]
            alive[victim_team].discard(victim_id)
            for team_id in sorted(alive):
                enemies = sum(
                    len(players)
                    for other_team, players in alive.items()
                    if other_team != team_id
                )
                if team_id in attempted or len(alive[team_id]) != 1 or not 1 <= enemies <= 5:
                    continue
                expected[(round_number, team_id)] = {
                    "player_id": next(iter(alive[team_id])),
                    "enemies_at_start": enemies,
                    "state": f"1v{enemies}",
                    "start_tick": event.get("tick"),
                    "trigger_event_id": event.get("event_id"),
                }
                attempted.add(team_id)
    return expected


def valid_block6_origin_timestamp(value: object) -> bool:
    if not isinstance(value, str) or not value:
        return False
    try:
        datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return False
    return bool(re.fullmatch(r"\d{4}-\d{2}-\d{2}(?:T.+(?:Z|[+-]\d{2}:\d{2}))?", value))


def validate_block6_versioned_config(
    value: object, location: str, errors: list[str]
) -> None:
    if not isinstance(value, Mapping):
        errors.append(f"{location}: configuracion versionada ausente")
        return
    if not value.get("version") or not value.get("source"):
        errors.append(f"{location}: version/source ausentes")
    if SHA256_PATTERN.fullmatch(str(value.get("checksum_sha256") or "")) is None:
        errors.append(f"{location}: checksum invalido")
    if not isinstance(value.get("effective_from"), str):
        errors.append(f"{location}: effective_from ausente")
    if not value.get("applicability_status"):
        errors.append(f"{location}: applicability_status ausente")


def validate_block6_semantics(
    match_id: str,
    envelopes: Mapping[str, object],
    records: Mapping[str, list[dict]],
    errors: list[str],
) -> None:
    participant_teams = {
        str(player.get("player_id")): str(player.get("team_id"))
        for player in records.get("participants", [])
    }
    stable_teams = set(participant_teams.values())
    if not stable_teams or stable_teams & {"CT", "T", "ct", "t", ""}:
        errors.append("canonical/core/participants.json: CT/T no puede ser team_id")
    sides_by_round: dict[tuple[int, str], str] = {}
    winners: dict[int, str | None] = {}
    for round_record in records.get("rounds", []):
        number = int(round_record.get("round_number") or 0)
        winners[number] = round_record.get("winner_team_id")
        for assignment in round_record.get("side_assignments", []):
            if isinstance(assignment, Mapping):
                sides_by_round[(number, str(assignment.get("team_id")))] = str(assignment.get("side"))

    transaction_ids: set[str] = set()
    for row in records.get("economy_players", []):
        location = f"canonical/derived/economy_players.json:{row.get('round_number')}:{row.get('player_id')}"
        player_id = str(row.get("player_id") or "")
        team_id = str(row.get("team_id") or "")
        side = str(row.get("side") or "")
        if int(row.get("round_number") or 0) < 1:
            errors.append(f"{location}: evento de warmup/freeze fuera de ronda competitiva")
        if participant_teams.get(player_id) != team_id or team_id not in stable_teams:
            errors.append(f"{location}: team_id no es estable")
        if sides_by_round.get((int(row.get("round_number") or 0), team_id)) != side:
            errors.append(f"{location}: side no coincide con side_assignments")
        money = row.get("money")
        if not isinstance(money, Mapping):
            errors.append(f"{location}: money debe ser objeto")
        else:
            for field in (
                "round_start_observed", "freeze_end_observed", "after_buy_observed",
                "after_buy_calculated", "round_end_observed", "next_round_observed",
                "next_round_calculated",
            ):
                validate_block6_money_value(money.get(field), f"{location}.money.{field}", errors)
            observed = money.get("after_buy_observed")
            calculated = money.get("after_buy_calculated")
            delta = money.get("native_calculated_delta")
            if (
                isinstance(observed, Mapping) and isinstance(calculated, Mapping)
                and isinstance(observed.get("amount"), int)
                and isinstance(calculated.get("amount"), int)
                and delta != observed["amount"] - calculated["amount"]
            ):
                errors.append(f"{location}: delta native/calculated incorrecto")
            if (
                delta is not None
                and not (
                    isinstance(observed, Mapping)
                    and isinstance(calculated, Mapping)
                    and isinstance(observed.get("amount"), int)
                    and isinstance(calculated.get("amount"), int)
                )
            ):
                errors.append(f"{location}: delta native/calculated sin ambos valores")
        for inventory_name in ("inventory_start", "inventory_freeze_end", "inventory_round_end"):
            inventory = row.get(inventory_name)
            if not isinstance(inventory, Mapping):
                errors.append(f"{location}.{inventory_name}: inventario debe ser objeto")
                continue
            if inventory.get("status") == "not_observed":
                if inventory.get("native_value") is not None or inventory.get("calculated_value") is not None or inventory.get("items"):
                    errors.append(f"{location}.{inventory_name}: inventario no observado debe conservar nulls")
            elif (
                inventory.get("status") != "observed_with_calculated_valuation"
                or not isinstance(inventory.get("native_value"), int)
                or not isinstance(inventory.get("calculated_value"), int)
            ):
                errors.append(f"{location}.{inventory_name}: native y calculated son obligatorios cuando se observa")
            for index, item in enumerate(inventory.get("items", [])):
                if isinstance(item, Mapping):
                    validate_block6_price(item.get("price"), f"{location}.{inventory_name}[{index}]", errors)
                    owner = item.get("original_owner_player_id")
                    if owner is not None and owner not in participant_teams:
                        errors.append(f"{location}.{inventory_name}[{index}]: propietario desconocido")
        refund_total = 0
        for index, transaction in enumerate(row.get("transactions", [])):
            if not isinstance(transaction, Mapping):
                errors.append(f"{location}.transactions[{index}]: debe ser objeto")
                continue
            transaction_id = str(transaction.get("transaction_id") or "")
            if not transaction_id or transaction_id in transaction_ids:
                errors.append(f"{location}.transactions[{index}]: ID ausente o duplicado")
            transaction_ids.add(transaction_id)
            if transaction.get("actor_player_id") != player_id or not transaction.get("source") or not transaction.get("source_event_ids"):
                errors.append(f"{location}.transactions[{index}]: provenance de jugador ausente")
            item = transaction.get("item")
            if not isinstance(item, Mapping):
                errors.append(f"{location}.transactions[{index}]: item ausente")
                continue
            validate_block6_price(item.get("price"), f"{location}.transactions[{index}]", errors)
            if transaction.get("type") == "purchase" and item.get("purchased_item") is None:
                errors.append(f"{location}.transactions[{index}]: compra sin purchased_item")
            if transaction.get("type") == "pickup" and item.get("original_owner_status") == "observed" and transaction.get("other_player_id") is None:
                errors.append(f"{location}.transactions[{index}]: pickup atribuido sin owner ID")
            if transaction.get("type") == "refund" and isinstance(transaction.get("amount"), int):
                refund_total += transaction["amount"]
        if isinstance(money, Mapping):
            start = money.get("round_start_observed")
            after_buy = money.get("after_buy_calculated")
            spent = row.get("spent_in_buy")
            if (
                isinstance(start, Mapping)
                and isinstance(after_buy, Mapping)
                and isinstance(spent, Mapping)
                and isinstance(start.get("amount"), int)
                and isinstance(after_buy.get("amount"), int)
                and isinstance(spent.get("amount"), int)
            ):
                expected_money = max(0, min(16000, start["amount"] - spent["amount"] + refund_total))
                if after_buy["amount"] != expected_money:
                    errors.append(f"{location}: transición after_buy_calculated imposible")

    reward_ids: set[str] = set()
    for row in records.get("economy_rounds", []):
        location = f"canonical/derived/economy_rounds.json:{row.get('round_number')}:{row.get('team_id')}"
        team_id = str(row.get("team_id") or "")
        if int(row.get("round_number") or 0) < 1:
            errors.append(f"{location}: evento de warmup/freeze fuera de ronda competitiva")
        if team_id not in stable_teams or sides_by_round.get((int(row.get("round_number") or 0), team_id)) != row.get("side"):
            errors.append(f"{location}: identidad o lado incoherente")
        loss_bonus = row.get("loss_bonus")
        if not isinstance(loss_bonus, Mapping) or loss_bonus.get("rules_version") != "stratai.cs2_economy_rules@1" or loss_bonus.get("amount") not in {1400, 1900, 2400, 2900, 3400}:
            errors.append(f"{location}: loss bonus sin reglas versionadas")
        for reward in row.get("rewards", []):
            if not isinstance(reward, Mapping):
                continue
            reward_id = str(reward.get("reward_id") or "")
            if not reward_id or reward_id in reward_ids:
                errors.append(f"{location}: premio duplicado")
            reward_ids.add(reward_id)
            if reward.get("observed_amount") is not None and reward.get("calculated_amount") is not None and reward.get("status") != "reconciled":
                errors.append(f"{location}: native y calculated se sobrescriben")

    clutch_by_player: dict[str, dict[str, int]] = {}
    clutch_keys: set[tuple[int, str]] = set()
    combat_event_ids = {
        str(event.get("event_id")) for event in records.get("combat_events", [])
    }
    expected_clutches = expected_block6_clutches(
        records.get("participants", []), records.get("combat_events", [])
    )
    for event in records.get("clutch_events", []):
        location = f"canonical/derived/clutch_events.json:{event.get('clutch_id')}"
        key = (int(event.get("round_number") or 0), str(event.get("team_id") or ""))
        if key in clutch_keys:
            errors.append(f"{location}: clutch duplicado por equipo/ronda")
        clutch_keys.add(key)
        if key[0] < 1:
            errors.append(f"{location}: clutch fuera de ronda competitiva")
        enemies = event.get("enemies_at_start")
        if event.get("attempt") is not True or not isinstance(enemies, int) or not 1 <= enemies <= 5 or event.get("state") != f"1v{enemies}" or not event.get("source_event_ids"):
            errors.append(f"{location}: clutch sin attempt o source events")
        source_event_ids = event.get("source_event_ids")
        if (
            not isinstance(source_event_ids, list)
            or event.get("trigger_event_id") not in source_event_ids
            or any(str(event_id) not in combat_event_ids for event_id in source_event_ids)
        ):
            errors.append(f"{location}: source events de clutch invalidos")
        expected_attempt = expected_clutches.get(key)
        if expected_attempt is None or any(
            event.get(field) != value for field, value in expected_attempt.items()
        ):
            errors.append(f"{location}: attempt no reconcilia con kills atomicas")
        winner = winners.get(key[0])
        expected = "not_evaluable" if winner is None else "won" if winner == key[1] else "lost"
        if event.get("result") != expected:
            errors.append(f"{location}: resultado no coincide con winner_team_id")
        summary = clutch_by_player.setdefault(str(event.get("player_id")), {"attempts": 0, "wins": 0, "losses": 0, "not_evaluable": 0})
        summary["attempts"] += 1
        summary[{"won": "wins", "lost": "losses"}.get(expected, "not_evaluable")] += 1
    for missing_key in sorted(set(expected_clutches) - clutch_keys):
        errors.append(
            "canonical/derived/clutch_events.json: falta attempt atomico "
            f"ronda={missing_key[0]} team_id={missing_key[1]}"
        )

    for player in records.get("player_stats", []):
        location = f"canonical/derived/player_stats.json:{player.get('player_id')}"
        player_id = str(player.get("player_id") or "")
        if participant_teams.get(player_id) != player.get("team_id"):
            errors.append(f"{location}: team_id no coincide con participants")
        native_status = player.get("native_scoreboard_status")
        native = player.get("native_scoreboard")
        metrics = player.get("metrics")
        derived = player.get("derived")
        if not isinstance(metrics, Mapping) or not isinstance(derived, Mapping):
            errors.append(f"{location}: metrics/derived ausentes")
            continue
        if native_status == "observed":
            if not isinstance(native, Mapping) or any(
                metrics.get(field) != native.get(field)
                for field in ("kills", "deaths", "assists", "total_damage", "utility_damage")
            ):
                errors.append(f"{location}: scoreboard nativo no reconcilia")
        elif native_status == "unavailable" and native is not None:
            errors.append(f"{location}: scoreboard unavailable debe ser null")
        if derived.get("utility_damage_observed") != metrics.get("utility_damage_observed"):
            errors.append(f"{location}: utilidad observada no reconcilia")
        rating = player.get("rating")
        if not isinstance(rating, Mapping) or rating.get("approximate") is not True or rating.get("algorithm_version") != "stratai.rating_hltv2_approx@1" or not rating.get("formula"):
            errors.append(f"{location}: rating aproximado sin provenance")
        clutch = player.get("clutch")
        expected = clutch_by_player.get(player_id, {"attempts": 0, "wins": 0, "losses": 0, "not_evaluable": 0})
        if not isinstance(clutch, Mapping) or any(clutch.get(field) != count for field, count in expected.items()):
            errors.append(f"{location}: resumen de clutch no reconcilia")

    metadata = envelopes.get("match_metadata")
    match = envelopes.get("match")
    manifest = envelopes.get("canonical_manifest")
    if not isinstance(metadata, Mapping):
        errors.append("canonical/core/match_metadata.json: raiz invalida")
    else:
        source = metadata.get("source")
        if metadata.get("match_id") != match_id or metadata.get("parser_version") != PARSER_SCHEMA_VERSION or metadata.get("export_format_version") != EXPORT_FORMAT_VERSION or metadata.get("quality_schema_version") != QUALITY_SCHEMA_VERSION:
            errors.append("canonical/core/match_metadata.json: versiones o match_id invalidos")
        if not isinstance(source, Mapping) or SHA256_PATTERN.fullmatch(str(source.get("checksum_sha256") or "")) is None:
            errors.append("canonical/core/match_metadata.json: checksum de demo ausente")
        required_hashes = {"economy_rules", "price_table", "clutch_algorithm", "rating_algorithm", "stats_algorithm"}
        configuration_hashes = metadata.get("configuration_hashes")
        if (
            not isinstance(configuration_hashes, Mapping)
            or not required_hashes <= set(configuration_hashes)
            or any(SHA256_PATTERN.fullmatch(str(value)) is None for value in configuration_hashes.values())
        ):
            errors.append("canonical/core/match_metadata.json: hashes de configuracion invalidos")
        transformations = metadata.get("transformation_versions")
        if not isinstance(transformations, Mapping) or not {"economy", "stats", "clutch", "metadata"} <= set(transformations):
            errors.append("canonical/core/match_metadata.json: versiones de transformacion ausentes")
        algorithms = metadata.get("algorithms")
        if not isinstance(algorithms, Mapping) or not {"clutch", "rating", "stats"} <= set(algorithms):
            errors.append("canonical/core/match_metadata.json: algoritmos versionados ausentes")
        else:
            for name in ("clutch", "rating", "stats"):
                validate_block6_versioned_config(
                    algorithms[name],
                    f"canonical/core/match_metadata.json.algorithms.{name}",
                    errors,
                )
        validate_block6_versioned_config(
            metadata.get("price_table"),
            "canonical/core/match_metadata.json.price_table",
            errors,
        )
        validate_block6_versioned_config(
            metadata.get("economy_rules"),
            "canonical/core/match_metadata.json.economy_rules",
            errors,
        )
        if isinstance(configuration_hashes, Mapping):
            expected_hashes = {
                "price_table": metadata.get("price_table", {}).get("checksum_sha256")
                if isinstance(metadata.get("price_table"), Mapping)
                else None,
                "economy_rules": metadata.get("economy_rules", {}).get("checksum_sha256")
                if isinstance(metadata.get("economy_rules"), Mapping)
                else None,
                "clutch_algorithm": algorithms.get("clutch", {}).get("checksum_sha256")
                if isinstance(algorithms, Mapping) and isinstance(algorithms.get("clutch"), Mapping)
                else None,
                "rating_algorithm": algorithms.get("rating", {}).get("checksum_sha256")
                if isinstance(algorithms, Mapping) and isinstance(algorithms.get("rating"), Mapping)
                else None,
                "stats_algorithm": algorithms.get("stats", {}).get("checksum_sha256")
                if isinstance(algorithms, Mapping) and isinstance(algorithms.get("stats"), Mapping)
                else None,
            }
            if any(configuration_hashes.get(name) != checksum for name, checksum in expected_hashes.items()):
                errors.append("canonical/core/match_metadata.json: hashes no reconcilian con configuraciones")
        played_at = metadata.get("played_at")
        if played_at is None:
            if metadata.get("played_at_status") != "unavailable" or metadata.get("played_at_source") is not None or metadata.get("origin_date") is not None:
                errors.append("canonical/core/match_metadata.json: played_at nulo sin unavailable")
        elif (
            metadata.get("played_at_status") != "observed"
            or metadata.get("played_at_source") != source.get("source")
            or str(source.get("source") or "").lower() in {"processing", "processed_at", "export"}
            or not valid_block6_origin_timestamp(played_at)
        ):
            errors.append("canonical/core/match_metadata.json: played_at sin fuente fiable")
        if isinstance(manifest, Mapping) and (
            manifest.get("demo_checksum_sha256") != source.get("checksum_sha256")
            or manifest.get("configuration_hashes") != configuration_hashes
            or manifest.get("transformation_versions") != transformations
        ):
            errors.append("canonical/core/match_metadata.json: lineage no coincide con manifest")
        if isinstance(match, Mapping) and match.get("played_at") != metadata.get("played_at"):
            errors.append("canonical/core/match_metadata.json: played_at no coincide con match")
        economy_players_document = envelopes.get("economy_players")
        economy_rounds_document = envelopes.get("economy_rounds")
        clutch_document = envelopes.get("clutch_events")
        if isinstance(economy_players_document, Mapping):
            validate_block6_versioned_config(
                economy_players_document.get("price_table"),
                "canonical/derived/economy_players.json.price_table",
                errors,
            )
            if economy_players_document.get("price_table") != metadata.get("price_table"):
                errors.append("canonical/derived/economy_players.json: price_table no reconcilia con metadata")
            if economy_players_document.get("economy_rules") != metadata.get("economy_rules"):
                errors.append("canonical/derived/economy_players.json: economy_rules no reconcilia con metadata")
        if isinstance(economy_rounds_document, Mapping):
            validate_block6_versioned_config(
                economy_rounds_document.get("economy_rules"),
                "canonical/derived/economy_rounds.json.economy_rules",
                errors,
            )
            if economy_rounds_document.get("economy_rules") != metadata.get("economy_rules"):
                errors.append("canonical/derived/economy_rounds.json: economy_rules no reconcilia con metadata")
        if isinstance(clutch_document, Mapping):
            validate_block6_versioned_config(
                clutch_document.get("algorithm"),
                "canonical/derived/clutch_events.json.algorithm",
                errors,
            )
            if isinstance(algorithms, Mapping) and clutch_document.get("algorithm") != algorithms.get("clutch"):
                errors.append("canonical/derived/clutch_events.json: algoritmo no reconcilia con metadata")


def validate_block6_quality_report(report: object, errors: list[str]) -> None:
    location = "canonical/diagnostics/quality_report.json"
    if not isinstance(report, Mapping):
        errors.append(f"{location}: report ausente")
        return
    checks = report.get("checks")
    if not isinstance(checks, list):
        errors.append(f"{location}: checks debe ser lista")
        return
    by_name = {check.get("name"): check for check in checks if isinstance(check, Mapping)}
    missing = REQUIRED_BLOCK6_QUALITY_CHECKS - set(by_name)
    if missing:
        errors.append(f"{location}: faltan gates Bloque 6 {sorted(missing)}")
    for name in REQUIRED_BLOCK6_QUALITY_CHECKS - {"economy_observation_coverage"}:
        if name in by_name and by_name[name].get("status") != "pass":
            errors.append(f"{location}: gate critico {name} no pasa")
    if "economy_observation_coverage" in by_name and by_name["economy_observation_coverage"].get("status") not in {"pass", "warning"}:
        errors.append(f"{location}: economy_observation_coverage invalido")
    for metric in HARD_BLOCK6_QUALITY_METRICS:
        if report.get(metric) != 0:
            errors.append(f"{location}: {metric} debe ser 0")


def validate_lineage(
    manifest: Mapping[str, object], descriptors: list[dict], errors: list[str]
) -> None:
    location = "canonical/manifest.json: lineage"
    lineage = manifest.get("lineage")
    if not isinstance(lineage, Mapping):
        errors.append(f"{location} debe ser un objeto")
        return
    expected_values = {
        "demo_checksum_sha256": manifest.get("demo_checksum_sha256"),
        "parser_version": PARSER_SCHEMA_VERSION,
        "export_format_version": EXPORT_FORMAT_VERSION,
        "validator_version": VALIDATOR_VERSION,
        "golden_corpus_version": GOLDEN_CORPUS_VERSION,
        "golden_corpus_manifest_id": "golden-demos-v2",
    }
    for field, expected in expected_values.items():
        if lineage.get(field) != expected:
            errors.append(f"{location}.{field} no coincide con {expected}")
    for field in (
        "demoinfocs_version",
        "map_name",
        "tick_rate_rules_version",
        "price_table_version",
        "price_table_sha256",
    ):
        if not isinstance(lineage.get(field), str) or not lineage[field].strip():
            errors.append(f"{location}.{field} debe ser una cadena no vacia")
    if not is_finite_number(lineage.get("tick_rate_hz")) or float(
        lineage.get("tick_rate_hz", 0)
    ) <= 0:
        errors.append(f"{location}.tick_rate_hz debe ser positivo")
    for field in (
        "algorithm_versions",
        "schema_versions",
        "configuration_hashes",
    ):
        value = lineage.get(field)
        if not isinstance(value, Mapping) or not value:
            errors.append(f"{location}.{field} debe ser un objeto no vacio")
    schema_versions = lineage.get("schema_versions")
    if isinstance(schema_versions, Mapping):
        expected_schemas = {
            str(descriptor.get("artifact_type")): descriptor.get("schema_id")
            for descriptor in descriptors
        }
        expected_schemas["canonical_manifest"] = MANIFEST_SCHEMA_ID
        for artifact_type, schema_id in expected_schemas.items():
            if schema_versions.get(artifact_type) != schema_id:
                errors.append(
                    f"{location}.schema_versions[{artifact_type}] no coincide con el catalogo"
                )
    for field in ("build_identifier", "processing_timestamp"):
        value = lineage.get(field)
        if not isinstance(value, Mapping):
            errors.append(f"{location}.{field} debe ser explicito")
            continue
        status = value.get("status")
        if status not in {"observed", "unavailable", "operational_only"}:
            errors.append(f"{location}.{field}.status invalido")
        if status in {"unavailable", "operational_only"} and value.get("value") is not None:
            errors.append(f"{location}.{field}.value debe ser null para {status}")
        if status == "unavailable" and not isinstance(value.get("source"), str):
            errors.append(f"{location}.{field}.source debe explicar la ausencia")
    input_hashes = lineage.get("input_hashes")
    if not isinstance(input_hashes, Mapping):
        errors.append(f"{location}.input_hashes debe ser un objeto")
    else:
        for field in ("physics_map", "nav_mesh", "callouts"):
            value = input_hashes.get(field)
            if not isinstance(value, Mapping) or value.get("status") not in {
                "observed",
                "unavailable",
            }:
                errors.append(f"{location}.input_hashes[{field}] debe ser explicito")
            elif value.get("status") == "unavailable" and value.get("value") is not None:
                errors.append(f"{location}.input_hashes[{field}].value debe ser null")
    for field in ("quality_flags", "warnings", "abstentions"):
        value = lineage.get(field)
        if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
            errors.append(f"{location}.{field} debe ser una lista de cadenas")


def validate_block7_quality_report(report: object, errors: list[str]) -> None:
    location = "canonical/diagnostics/quality_report.json"
    if not isinstance(report, Mapping):
        return
    domains = report.get("domains")
    if not isinstance(domains, list) or not all(
        isinstance(domain, Mapping) for domain in domains
    ):
        errors.append(f"{location}: domains debe ser una lista de objetos")
        return
    names: list[str] = []
    for index, domain in enumerate(domains):
        name = domain.get("name")
        if not isinstance(name, str):
            errors.append(
                f"{location}: domains[{index}]: name debe ser una cadena"
            )
            continue
        names.append(name)
    if len(names) != len(set(names)):
        errors.append(f"{location}: domains contiene nombres duplicados")
    missing = REQUIRED_BLOCK7_DOMAINS - set(names)
    unexpected = set(names) - REQUIRED_BLOCK7_DOMAINS
    if missing or unexpected:
        errors.append(
            f"{location}: dominios Bloque 7 incorrectos; faltan={sorted(missing)} "
            f"sobran={sorted(unexpected)}"
        )
    for index, domain in enumerate(domains):
        domain_location = f"{location}: domains[{index}]"
        absent = BLOCK7_DOMAIN_FIELDS - set(domain)
        if absent:
            errors.append(f"{domain_location}: faltan campos {sorted(absent)}")
        status = domain.get("status")
        severity = domain.get("severity")
        if status not in {"pass", "warning", "fail"}:
            errors.append(f"{domain_location}: status invalido")
        if severity not in {"hard", "warning"}:
            errors.append(f"{domain_location}: severity invalida")
        if status == "fail" or (severity == "hard" and status != "pass"):
            errors.append(f"{domain_location}: gate duro no pasa")
        coverage = domain.get("coverage")
        if not is_finite_number(coverage) or not 0 <= float(coverage) <= 1:
            errors.append(f"{domain_location}: coverage debe estar entre 0 y 1")
        for field in ("unavailable_count", "inferred_count"):
            value = domain.get(field)
            if isinstance(value, bool) or not isinstance(value, int) or value < 0:
                errors.append(f"{domain_location}: {field} debe ser entero no negativo")
        for field in (
            "warning_details",
            "hard_failure_details",
            "source_artifacts",
            "schema_versions",
        ):
            value = domain.get(field)
            if not isinstance(value, list) or not all(
                isinstance(item, str) for item in value
            ):
                errors.append(f"{domain_location}: {field} debe ser lista de cadenas")
    for metric in (
        "block7_artifact_integrity_violations",
        "block7_causal_availability_violations",
        "block7_future_leakage_violations",
        "block7_schema_compatibility_violations",
        "block7_determinism_violations",
        "block7_corpus_quality_violations",
    ):
        if report.get(metric) != 0:
            errors.append(f"{location}: {metric} debe ser 0")


def nested_keys(value: object) -> set[str]:
    keys: set[str] = set()
    if isinstance(value, Mapping):
        for key, child in value.items():
            keys.add(str(key))
            keys.update(nested_keys(child))
    elif isinstance(value, list):
        for child in value:
            keys.update(nested_keys(child))
    return keys


def validate_causal_partitions(records: Mapping[str, list[dict]], errors: list[str]) -> None:
    partition_names = (
        "decisions",
        "decision_features",
        "oracle_context",
        "decision_outcomes",
        "quality_masks",
    )
    by_partition: dict[str, dict[str, dict]] = {}
    for partition in partition_names:
        rows = records.get(partition, [])
        identifiers = [row.get("decision_id") for row in rows]
        if len(identifiers) != len(set(identifiers)):
            errors.append(f"canonical/causal/{partition}: decision_id duplicado")
        by_partition[partition] = {
            str(row.get("decision_id")): row for row in rows if row.get("decision_id")
        }
    expected_ids = set(by_partition["decisions"])
    for partition in partition_names[1:]:
        if set(by_partition[partition]) != expected_ids:
            errors.append(
                f"canonical/causal/{partition}: decision_id no coincide con decisions"
            )
    participant_ids = {
        str(row.get("player_id"))
        for row in records.get("participants", [])
        if row.get("player_id")
    }
    state_ids = {
        str(row.get("state_id"))
        for row in records.get("player_states", [])
        if row.get("state_id")
    }
    for decision_id, decision in by_partition["decisions"].items():
        location = f"canonical/causal/decisions:{decision_id}"
        actor_id = decision.get("actor_player_id")
        if actor_id not in participant_ids:
            errors.append(f"{location}: actor_player_id no referencia un participante")
        if decision.get("actor_id_usage") != "join_only":
            errors.append(f"{location}: actor_player_id debe declararse join_only")
        decision_type = decision.get("decision_type")
        action_taken = decision.get("action_taken")
        if decision_type not in MVP_DECISION_TYPES:
            errors.append(f"{location}: decision_type fuera del contrato MVP")
        elif action_taken not in MVP_DECISION_ACTIONS[decision_type]:
            errors.append(f"{location}: action_taken fuera del contrato de decision")
        t0_tick = decision.get("t0_tick")
        availability_tick = decision.get("availability_tick")
        if (
            isinstance(t0_tick, bool)
            or not isinstance(t0_tick, int)
            or isinstance(availability_tick, bool)
            or not isinstance(availability_tick, int)
            or availability_tick > t0_tick
        ):
            errors.append(f"{location}: availability_tick debe ser <= t0_tick")
        if decision.get("availability_status") not in {
            "observed", "inferred", "derived", "unavailable"
        } or decision.get("causal_role") != "decision":
            errors.append(f"{location}: disponibilidad o causal_role invalidos")
        expected_scope = {
            "peek_hold_or_reposition": "observable_proxy",
            "spacing_or_trade_connection": "observed_physical_proxy",
        }.get(decision_type)
        if decision.get("visibility_scope") != expected_scope:
            errors.append(f"{location}: visibility_scope no coincide con decision_type")
        state_ref = decision.get("observed_state_ref")
        state_status = decision.get("state_availability_status")
        if state_status == "observed":
            if not isinstance(state_ref, str) or state_ref not in state_ids:
                errors.append(f"{location}: observed_state_ref no referencia un estado")
        elif state_status == "unavailable":
            if state_ref is not None:
                errors.append(
                    f"{location}: estado unavailable no puede inventar una referencia"
                )
        else:
            errors.append(f"{location}: state_availability_status invalido")
        for field in ("source", "source_record_id", "algorithm_version"):
            if not isinstance(decision.get(field), str) or not decision[field]:
                errors.append(f"{location}: {field} debe ser no vacio")
        source_event_ids = decision.get("source_event_ids")
        if not isinstance(source_event_ids, list) or not all(
            isinstance(item, str) and item for item in source_event_ids
        ):
            errors.append(f"{location}: source_event_ids invalido")
        elif source_event_ids != sorted(set(source_event_ids)):
            errors.append(f"{location}: source_event_ids no es estable/unico")
    for decision_id, features in by_partition["decision_features"].items():
        location = f"canonical/causal/decision_features:{decision_id}"
        prohibited = nested_keys(features) & DECISION_FEATURE_PROHIBITED_KEYS
        if prohibited:
            errors.append(f"{location}: contiene campos futuros/prohibidos {sorted(prohibited)}")
        unknown = set(features) - DECISION_FEATURE_ALLOWED_KEYS
        missing = DECISION_FEATURE_ALLOWED_KEYS - set(features)
        if unknown:
            errors.append(f"{location}: contiene campos fuera de allowlist {sorted(unknown)}")
        if missing:
            errors.append(f"{location}: carece de campos causales {sorted(missing)}")
        decision = by_partition["decisions"].get(decision_id, {})
        if features.get("decision_type") != decision.get("decision_type"):
            errors.append(f"{location}: decision_type no coincide con decisions")
        t0_tick = features.get("t0_tick")
        availability_tick = features.get("availability_tick_max")
        if (
            isinstance(t0_tick, bool)
            or not isinstance(t0_tick, int)
            or isinstance(availability_tick, bool)
            or not isinstance(availability_tick, int)
            or availability_tick > t0_tick
        ):
            errors.append(f"{location}: availability_tick_max debe ser <= t0_tick")
        for count_field in (
            "participant_count",
            "observed_participant_states",
            "source_state_count",
        ):
            value = features.get(count_field)
            if isinstance(value, bool) or not isinstance(value, int) or value < 0:
                errors.append(f"{location}: {count_field} debe ser entero no negativo")
        participant_count = features.get("participant_count")
        observed_count = features.get("observed_participant_states")
        if isinstance(participant_count, int) and isinstance(observed_count, int) and observed_count > participant_count:
            errors.append(f"{location}: estados observados exceden participantes")
        alive_count = features.get("alive_participant_count")
        if alive_count is not None and (
            isinstance(alive_count, bool)
            or not isinstance(alive_count, int)
            or alive_count < 0
            or (isinstance(participant_count, int) and alive_count > participant_count)
        ):
            errors.append(f"{location}: alive_participant_count invalido")

        for value_field, status_field in (
            ("initial_distance_world_units", "initial_distance_status"),
            ("nearest_teammate_distance_world_units", "nearest_teammate_distance_status"),
            ("nearest_connection_time_ms", "nearest_connection_time_status"),
            ("minimum_facing_delta_deg", "facing_status"),
        ):
            value = features.get(value_field)
            status = features.get(status_field)
            if status == "unavailable":
                if value is not None:
                    errors.append(f"{location}: {value_field} unavailable debe ser null")
            elif status not in {"observed", "derived"} or not is_finite_number(value) or float(value) < 0:
                errors.append(f"{location}: {value_field}/{status_field} invalido")
        if (
            features.get("initial_distance_world_units") is not None
            or features.get("initial_distance_status") != "unavailable"
        ):
            errors.append(
                f"{location}: distancia actor-target exacta pertenece a oracle "
                "hasta existir evidencia visual causal"
            )
        facing = features.get("minimum_facing_delta_deg")
        if is_finite_number(facing) and float(facing) > 180:
            errors.append(f"{location}: minimum_facing_delta_deg fuera de rango")

        exposed = features.get("enemies_exposed_count")
        exposed_status = features.get("enemies_exposed_status")
        if exposed_status == "unavailable":
            if exposed is not None:
                errors.append(f"{location}: enemies_exposed_count unavailable debe ser null")
        elif exposed_status not in {"observed", "derived"} or isinstance(exposed, bool) or not isinstance(exposed, int) or exposed < 0:
            errors.append(f"{location}: enemies_exposed_count/status invalido")

        for value_field, status_field in (
            ("trade_possible", "trade_possible_status"),
            ("any_line_of_sight", "line_of_sight_status"),
        ):
            value = features.get(value_field)
            status = features.get(status_field)
            if status == "unavailable":
                if value is not None:
                    errors.append(f"{location}: {value_field} unavailable debe ser null")
            elif status != "derived" or not isinstance(value, bool):
                errors.append(f"{location}: {value_field}/{status_field} invalido")
        for clock_field in ("round_clock_remaining_ms", "bomb_time_remaining_ms"):
            value = features.get(clock_field)
            if value is not None and (
                isinstance(value, bool) or not isinstance(value, int) or value < 0
            ):
                errors.append(f"{location}: {clock_field} invalido")
        link = {field: features.get(field) for field in CAUSAL_LINK_FIELDS}
        for partition in (
            "decisions",
            "oracle_context",
            "decision_outcomes",
            "quality_masks",
        ):
            row = by_partition[partition].get(decision_id, {})
            for field in CAUSAL_LINK_FIELDS - {"schema_id"}:
                if row.get(field) != link[field]:
                    errors.append(
                        f"canonical/causal/{partition}:{decision_id}: {field} no coincide"
                    )
        outcome = by_partition["decision_outcomes"].get(decision_id, {})
        outcome_tick = outcome.get("outcome_tick")
        if not isinstance(outcome_tick, int) or not isinstance(t0_tick, int) or outcome_tick < t0_tick:
            errors.append(f"canonical/causal/outcomes:{decision_id}: outcome_tick anterior a t0")
        horizons = outcome.get("horizons")
        if not isinstance(horizons, list) or [
            item.get("horizon_seconds") if isinstance(item, Mapping) else None
            for item in horizons
        ] != [2, 5, 10]:
            errors.append(f"canonical/causal/outcomes:{decision_id}: faltan horizons 2/5/10")
        elif any(
            item.get("status") != "derived_outcome_only"
            or not isinstance(item.get("outcome"), str)
            or not item.get("outcome")
            or item.get("source") not in {"engagements@2", "trades@1"}
            for item in horizons
        ):
            errors.append(f"canonical/causal/outcomes:{decision_id}: horizons invalidos")
        oracle = by_partition["oracle_context"].get(decision_id, {})
        if oracle.get("available") is not False or oracle.get("status") != "unavailable":
            errors.append(
                f"canonical/causal/oracle_context:{decision_id}: oracle no confiable debe abstenerse"
            )
        mask = by_partition["quality_masks"].get(decision_id, {})
        mask_sets: list[set[str]] = []
        for field in ("available_fields", "unavailable_fields", "inferred_fields"):
            value = mask.get(field)
            if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
                errors.append(f"canonical/causal/quality_masks:{decision_id}: {field} invalido")
                mask_sets.append(set())
            else:
                if value != sorted(set(value)):
                    errors.append(
                        f"canonical/causal/quality_masks:{decision_id}: {field} no es estable/unico"
                    )
                mask_sets.append(set(value))
        if any(mask_sets[left] & mask_sets[right] for left, right in ((0, 1), (0, 2), (1, 2))):
            errors.append(
                f"canonical/causal/quality_masks:{decision_id}: mascaras solapadas"
            )


def tactical_target_ticks(start_tick: int, end_tick: int, period_ticks: float) -> list[int]:
    result: list[int] = []
    previous = start_tick - 1
    slot = 0
    while True:
        # Go math.Round is half-away-from-zero; all canonical ticks are non-negative.
        target = start_tick + math.floor(slot * period_ticks + 0.5)
        if target > end_tick:
            return result
        if target > previous:
            result.append(target)
            previous = target
        slot += 1


def validate_tactical_semantics(
    sampling: object,
    physical_rows: Iterable[dict],
    oracle_rows: Iterable[dict],
    gaps: list[dict],
    rounds: list[dict],
    participants: list[dict],
    match_id: str,
    tick_rate: float,
    errors: list[str],
    *,
    physical_row_count: int,
    oracle_row_count: int,
) -> None:
    location = "canonical/states/tactical"
    if not isinstance(sampling, Mapping):
        errors.append(f"{location}/sampling.json: raiz invalida")
        return
    config = sampling.get("sampling")
    period_ticks = tick_rate / 16.0
    if (
        sampling.get("identity_semantics") != "join_only"
        or sampling.get("join_keys") != {"match_id": match_id}
        or not isinstance(config, Mapping)
        or config.get("target_hz") != 16
        or config.get("tick_rate") != tick_rate
        or not is_finite_number(config.get("period_ticks"))
        or not math.isclose(float(config["period_ticks"]), period_ticks, abs_tol=1e-9)
        or config.get("strategy")
        != "round_anchored_right_closed_no_carry_forward"
    ):
        errors.append(f"{location}/sampling.json: contrato 16 Hz invalido")
    if any(
        (
            sampling.get("physical_row_count") != physical_row_count,
            sampling.get("oracle_row_count") != oracle_row_count,
            sampling.get("gap_count") != len(gaps),
        )
    ):
        errors.append(f"{location}/sampling.json: recuentos no coinciden")

    participant_team = {
        row.get("player_id"): row.get("team_id")
        for row in participants
        if isinstance(row.get("player_id"), str)
    }
    round_by_number = {
        row.get("round_number"): row
        for row in rounds
        if isinstance(row.get("round_number"), int)
    }
    expected_ticks: set[tuple[int, int]] = set()
    for round_number, round_row in round_by_number.items():
        start_tick = round_row.get("start_tick")
        end_tick = round_row.get("end_tick")
        if isinstance(start_tick, int) and isinstance(end_tick, int):
            expected_ticks.update(
                (round_number, tick)
                for tick in tactical_target_ticks(start_tick, end_tick, period_ticks)
            )

    physical_ticks: set[tuple[int, int]] = set()
    previous_physical_key: tuple[object, ...] | None = None
    physical_field_names = {
        "team", "position", "velocity_world_units_per_second",
        "horizontal_velocity_world_units_per_second", "yaw", "pitch", "health",
        "armor", "alive", "active_weapon", "grenades", "has_c4", "has_defuse_kit",
        "ammo_in_magazine", "ammo_reserve", "is_ducking", "is_walking", "is_scoped",
        "is_reloading", "is_blind", "flash_duration_seconds", "money", "is_defusing",
    }
    forbidden_identity_keys = {"display_name", "name", "steam_id", "player_id"}
    for row in physical_rows:
        row_location = f"{location}/observed.jsonl.gz:{row.get('round_number')}:{row.get('tick')}"
        join_keys = row.get("join_keys")
        provenance = row.get("provenance")
        state = row.get("state")
        if not isinstance(join_keys, Mapping) or not isinstance(provenance, Mapping) or not isinstance(state, Mapping):
            errors.append(f"{row_location}: join/state/provenance invalidos")
            continue
        observer = join_keys.get("observer_id")
        subject = join_keys.get("subject_id")
        round_number = row.get("round_number")
        key = (round_number, row.get("tick"), observer, subject)
        if key == previous_physical_key:
            errors.append(f"{row_location}: observacion duplicada")
        previous_physical_key = key
        if (
            row.get("identity_semantics") != "join_only"
            or join_keys.get("match_id") != match_id
            or join_keys.get("round_id")
            != (
                f"{match_id}:round:{round_number:03d}"
                if isinstance(round_number, int) and not isinstance(round_number, bool)
                else None
            )
            or observer not in participant_team
            or subject not in participant_team
        ):
            errors.append(f"{row_location}: join keys no son join_only/referenciales")
        tick = row.get("tick")
        availability = row.get("availability_tick")
        if (
            isinstance(tick, bool)
            or not isinstance(tick, int)
            or isinstance(availability, bool)
            or not isinstance(availability, int)
            or availability > tick
        ):
            errors.append(f"{row_location}: availability_tick debe ser <= tick")
        scope = row.get("visibility_scope")
        if row.get("status") != "observed" or row.get("causal_role") != "model_input_observation" or scope not in {"self", "team", "enemy_los"}:
            errors.append(f"{row_location}: rol/status/scope observado invalido")
        if scope == "self" and observer != subject:
            errors.append(f"{row_location}: scope self contradice IDs")
        if scope == "team" and (observer == subject or participant_team.get(observer) != participant_team.get(subject)):
            errors.append(f"{row_location}: scope team contradice roster")
        if scope == "enemy_los" and (
            participant_team.get(observer) == participant_team.get(subject)
            or provenance.get("geometry_status") != "loaded"
            or provenance.get("line_of_sight") is not True
        ):
            errors.append(f"{row_location}: enemigo sin geometria/LOS fiable")
        if scope in {"self", "team"} and (
            provenance.get("geometry_status") != "not_required"
            or provenance.get("line_of_sight") is not None
        ):
            errors.append(f"{row_location}: self/team no debe fabricar LOS")
        if nested_keys(state) & forbidden_identity_keys:
            errors.append(f"{row_location}: observed contiene identidad fuera de join_keys")
        availability = state.get("field_availability")
        if set(state) != physical_field_names | {"field_availability"} or not isinstance(availability, Mapping) or set(availability) != physical_field_names:
            errors.append(f"{row_location}: schema/availability fisico incompleto")
        else:
            bool_fields = {
                "alive", "has_c4", "has_defuse_kit", "is_ducking", "is_walking",
                "is_scoped", "is_reloading", "is_blind", "is_defusing",
            }
            int_fields = {"health", "armor", "ammo_in_magazine", "ammo_reserve", "money"}
            float_fields = {
                "horizontal_velocity_world_units_per_second", "yaw", "pitch",
                "flash_duration_seconds",
            }
            vector_fields = {"position", "velocity_world_units_per_second"}
            for field in physical_field_names:
                field_status = availability.get(field)
                value = state.get(field)
                if field_status == "unavailable":
                    if value is not None:
                        errors.append(f"{row_location}: {field} unavailable debe ser null")
                    continue
                if field_status not in {"observed", "derived"}:
                    errors.append(f"{row_location}: status de {field} invalido")
                    continue
                valid_value = True
                if field in bool_fields:
                    valid_value = isinstance(value, bool)
                elif field in int_fields:
                    valid_value = isinstance(value, int) and not isinstance(value, bool) and value >= 0
                elif field in float_fields:
                    valid_value = is_finite_number(value) and float(value) >= 0 if field == "flash_duration_seconds" else is_finite_number(value)
                elif field in vector_fields:
                    valid_value = isinstance(value, Mapping) and all(
                        is_finite_number(value.get(axis)) for axis in ("x", "y", "z")
                    )
                elif field == "grenades":
                    valid_value = isinstance(value, list) and all(isinstance(item, str) and item for item in value)
                elif field in {"team", "active_weapon"}:
                    valid_value = isinstance(value, str) and bool(value)
                if not valid_value:
                    errors.append(f"{row_location}: valor observado/derivado invalido para {field}")
            if availability.get("is_reloading") != "unavailable" or state.get("is_reloading") is not None:
                errors.append(f"{row_location}: recarga sin productor debe ser null/unavailable")
            allowed_by_scope = {
                "team": {
                    "team", "position", "velocity_world_units_per_second",
                    "horizontal_velocity_world_units_per_second", "health", "armor",
                    "alive", "has_c4",
                },
                "enemy_los": {
                    "team", "position", "yaw", "pitch", "alive", "active_weapon",
                    "is_ducking", "is_walking", "is_scoped", "is_defusing",
                },
            }
            if scope in allowed_by_scope and any(
                availability.get(field) != "unavailable"
                for field in physical_field_names - allowed_by_scope[scope]
            ):
                errors.append(f"{row_location}: scope {scope} contiene campos no observables")
        if isinstance(round_number, int) and not isinstance(round_number, bool) and isinstance(tick, int) and not isinstance(tick, bool):
            physical_ticks.add((round_number, tick))

    previous_oracle_key: tuple[object, ...] | None = None
    for row in oracle_rows:
        row_location = f"{location}/oracle.jsonl:{row.get('round_number')}:{row.get('tick')}"
        join_keys = row.get("join_keys")
        if not isinstance(join_keys, Mapping):
            errors.append(f"{row_location}: join_keys invalido")
            continue
        key = (row.get("round_number"), row.get("tick"), join_keys.get("subject_id"))
        if key == previous_oracle_key:
            errors.append(f"{row_location}: oracle duplicado")
        previous_oracle_key = key
        tick = row.get("tick")
        availability = row.get("availability_tick")
        round_number = row.get("round_number")
        if (
            row.get("identity_semantics") != "join_only"
            or join_keys.get("match_id") != match_id
            or join_keys.get("round_id")
            != (
                f"{match_id}:round:{round_number:03d}"
                if isinstance(round_number, int) and not isinstance(round_number, bool)
                else None
            )
            or join_keys.get("observer_id") is not None
            or join_keys.get("subject_id") not in participant_team
            or row.get("status") != "observed"
            or row.get("causal_role") != "label_only_oracle"
            or row.get("visibility_scope") != "oracle"
            or not isinstance(tick, int)
            or not isinstance(availability, int)
            or availability > tick
        ):
            errors.append(f"{row_location}: contrato oracle invalido")

    gap_ticks: set[tuple[int, int]] = set()
    for gap in gaps:
        row_location = f"{location}/gaps.jsonl:{gap.get('round_number')}:{gap.get('tick')}"
        key = (gap.get("round_number"), gap.get("tick"))
        if key in gap_ticks:
            errors.append(f"{row_location}: gap duplicado")
        if isinstance(key[0], int) and isinstance(key[1], int):
            gap_ticks.add((key[0], key[1]))
        if (
            gap.get("identity_semantics") != "join_only"
            or gap.get("availability_tick") is not None
            or gap.get("status") != "unavailable"
            or gap.get("causal_role") != "coverage_gap"
            or gap.get("visibility_scope") != "sampling_gap"
            or gap.get("reason") not in {
                "missing_replay_frame_for_target_window",
                "no_joinable_players",
                "no_eligible_observer",
            }
        ):
            errors.append(f"{row_location}: contrato de gap invalido")

    if physical_ticks & gap_ticks:
        errors.append(f"{location}: un target no puede ser observado y gap a la vez")
    actual_ticks = physical_ticks | gap_ticks
    if actual_ticks != expected_ticks:
        errors.append(f"{location}: stream 16 Hz tiene huecos silenciosos o ticks extra")


def validate_references(
    match_id: str,
    envelopes: Mapping[str, dict],
    records: Mapping[str, list[dict]],
    errors: list[str],
    streamed_artifacts: Mapping[str, StreamedArtifact] | None = None,
) -> None:
    streamed_artifacts = streamed_artifacts or {}
    players = records.get("participants", [])
    player_ids = validate_unique_ids(
        players, "player_id", "canonical/core/participants.json", errors
    )
    if any(PLAYER_ID_PATTERN.fullmatch(player_id) is None for player_id in player_ids):
        errors.append(
            "canonical/core/participants.json: player_id debe usar steam:<decimal positivo>"
        )
    validate_replay_player_references(records.get("replay_round", []), player_ids, errors)
    round_numbers = validate_rounds(match_id, records.get("rounds", []), errors)
    validate_block6_semantics(match_id, envelopes, records, errors)

    event_ids: set[str] = set()
    for event_type in ("combat", "utility", "objective"):
        event_ids |= validate_event_ids(
            match_id, event_type, records.get(f"{event_type}_events", []), errors
        )
    engagements = records.get("engagements", [])
    engagement_ids = validate_unique_ids(
        engagements, "engagement_id", "canonical/derived/engagements.json", errors
    )
    expected_engagement = re.compile(rf"^{re.escape(match_id)}:engagement:[0-9]{{6,}}$")
    if any(
        expected_engagement.fullmatch(identifier) is None
        for identifier in engagement_ids
    ):
        errors.append(
            "canonical/derived/engagements.json: engagement_id no sigue el formato canónico"
        )
    validate_unique_ids(
        records.get("player_states", []),
        "state_id",
        "canonical/states/player_states",
        errors,
    )
    state_id_pattern = re.compile(
        rf"^{re.escape(match_id)}:state:([0-9]{{3,}}):([0-9]{{9,}}):(steam:[1-9][0-9]*)$"
    )
    for record in records.get("player_states", []):
        match = state_id_pattern.fullmatch(str(record.get("state_id", "")))
        if match is None:
            errors.append(
                "canonical/states/player_states: state_id no sigue el formato canónico"
            )
            continue
        if (
            int(match.group(1)) != record.get("round_number")
            or int(match.group(2)) != record.get("tick")
            or match.group(3) != record.get("player_id")
        ):
            errors.append(
                "canonical/states/player_states: state_id discrepa de ronda, tick o jugador"
            )
    validate_player_state_semantics(records.get("player_states", []), errors)
    match_envelope = envelopes.get("match", {})
    tick_rate = match_envelope.get("tick_rate_hz")
    if not is_finite_number(tick_rate) or float(tick_rate) <= 0:
        errors.append("canonical/core/match.json: tick_rate_hz debe ser positivo")
        tick_rate = 64.0
    engagement_metrics = validate_engagement_semantics(
        envelopes.get("engagements"),
        engagements,
        envelopes.get("trades"),
        records.get("combat_events", []),
        records.get("player_states", []),
        records.get("player_stats", []),
        float(tick_rate),
        errors,
    )
    combat_metrics = validate_combat_semantics(
        match_id,
        records.get("participants", []),
        records.get("combat_events", []),
        records.get("player_stats", []),
        records.get("replay_round", []),
        errors,
    )
    validate_utility_semantics(
        match_id,
        envelopes.get("match", {}),
        records.get("participants", []),
        records.get("rounds", []),
        records.get("utility_events", []),
        errors,
    )
    validate_objective_semantics(
        match_id,
        envelopes.get("match", {}),
        records.get("participants", []),
        records.get("rounds", []),
        records.get("objective_events", []),
        records.get("player_states", []),
        records.get("replay_round", []),
        errors,
    )
    utility_replay_mismatches = validate_replay_utility_markers(
        records.get("utility_events", []),
        records.get("replay_round", []),
        errors,
    )

    non_canonical_reference_types = {
        "__paths__",
        "match",
        "participants",
        "rounds",
        "quality_report",
        "replay_index",
        "replay_round",
    }
    for artifact_type, artifact_records in records.items():
        if artifact_type in non_canonical_reference_types:
            continue
        for record in artifact_records:
            validate_record_references(
                artifact_type,
                record,
                round_numbers,
                player_ids,
                event_ids,
                errors,
            )

    if match_envelope.get("round_count") != len(round_numbers):
        errors.append(
            "canonical/core/match.json: round_count no coincide con rounds.json"
        )
    validate_round_outcomes(
        match_envelope,
        records.get("rounds", []),
        envelopes.get("replay_index", {}),
        errors,
    )
    quality_report = envelopes.get("quality_report", {}).get("report")
    validate_utility_quality_report(
        quality_report,
        records.get("utility_events", []),
        utility_replay_mismatches,
        errors,
    )
    validate_combat_quality_report(
        quality_report,
        records.get("combat_events", []),
        combat_metrics,
        errors,
    )
    validate_engagement_quality_report(quality_report, engagement_metrics, errors)
    validate_block6_quality_report(quality_report, errors)
    validate_block7_quality_report(quality_report, errors)
    validate_causal_partitions(records, errors)
    physical_artifact = streamed_artifacts.get("tactical_observations")
    oracle_artifact = streamed_artifacts.get("tactical_oracle")
    physical_rows: Iterable[dict]
    oracle_rows: Iterable[dict]
    if physical_artifact is None:
        physical_rows = records.get("tactical_observations", [])
        physical_row_count = len(records.get("tactical_observations", []))
    else:
        physical_rows = iter_records_with_valid_references(
            "tactical_observations",
            (
                record
                for _, record in iter_json_lines(physical_artifact.file_path, None)
            ),
            round_numbers,
            player_ids,
            event_ids,
            errors,
        )
        physical_row_count = physical_artifact.record_count
    if oracle_artifact is None:
        oracle_rows = records.get("tactical_oracle", [])
        oracle_row_count = len(records.get("tactical_oracle", []))
    else:
        oracle_rows = iter_records_with_valid_references(
            "tactical_oracle",
            (
                record
                for _, record in iter_json_lines(oracle_artifact.file_path, None)
            ),
            round_numbers,
            player_ids,
            event_ids,
            errors,
        )
        oracle_row_count = oracle_artifact.record_count
    validate_tactical_semantics(
        envelopes.get("tactical_sampling"),
        physical_rows,
        oracle_rows,
        records.get("tactical_gaps", []),
        records.get("rounds", []),
        records.get("participants", []),
        match_id,
        float(tick_rate),
        errors,
        physical_row_count=physical_row_count,
        oracle_row_count=oracle_row_count,
    )
    replay_metadata = envelopes.get("replay_index", {}).get("metadata")
    if replay_metadata is not None and (
        not isinstance(replay_metadata, Mapping)
        or replay_metadata.get("schema_version") != REPLAY_SCHEMA_VERSION
    ):
        errors.append(
            "canonical/presentation/replay/index.json: "
            f"metadata.schema_version debe ser {REPLAY_SCHEMA_VERSION}"
        )
    state_paths = {
        path
        for path, artifact_type in records.get("__paths__", [])
        if artifact_type == "player_states"
    }
    expected_paths = {
        f"states/player_states/round_{number:03d}.jsonl"
        for number in round_numbers.values()
    }
    if state_paths != expected_paths:
        errors.append(
            "canonical/states/player_states: debe existir exactamente un fichero por ronda"
        )


def validate_canonical_bundle(
    match_dir: Path, match_id: str, errors: list[str]
) -> None:
    canonical_dir = match_dir / "canonical"
    manifest_path = canonical_dir / "manifest.json"
    manifest = (
        load_json_object(manifest_path, errors) if manifest_path.is_file() else None
    )
    if manifest is None:
        if not manifest_path.is_file():
            errors.append("canonical/manifest.json: fichero requerido inexistente")
        return
    if manifest.get("schema_id") != MANIFEST_SCHEMA_ID:
        errors.append(
            f"canonical/manifest.json: schema_id debe ser {MANIFEST_SCHEMA_ID}"
        )
    if manifest.get("export_format_version") != EXPORT_FORMAT_VERSION:
        errors.append(
            f"canonical/manifest.json: export_format_version debe ser {EXPORT_FORMAT_VERSION}"
        )
    if manifest.get("match_id") != match_id:
        errors.append("canonical/manifest.json: match_id no coincide")
    if SHA256_PATTERN.fullmatch(str(manifest.get("demo_checksum_sha256") or "")) is None:
        errors.append("canonical/manifest.json: demo_checksum_sha256 invalido")
    if manifest.get("parser_version") != PARSER_SCHEMA_VERSION:
        errors.append(f"canonical/manifest.json: parser_version debe ser {PARSER_SCHEMA_VERSION}")
    configuration_hashes = manifest.get("configuration_hashes")
    if (
        not isinstance(configuration_hashes, Mapping)
        or not {
            "economy_rules",
            "price_table",
            "clutch_algorithm",
            "rating_algorithm",
            "stats_algorithm",
        }
        <= set(configuration_hashes)
        or any(
            SHA256_PATTERN.fullmatch(str(value)) is None
            for value in configuration_hashes.values()
        )
    ):
        errors.append("canonical/manifest.json: configuration_hashes invalidos")
    transformation_versions = manifest.get("transformation_versions")
    if (
        not isinstance(transformation_versions, Mapping)
        or not {"economy", "stats", "clutch", "metadata"}
        <= set(transformation_versions)
    ):
        errors.append("canonical/manifest.json: transformation_versions ausente")
    descriptors = manifest.get("artifacts")
    if not isinstance(descriptors, list) or not all(
        isinstance(item, dict) for item in descriptors
    ):
        errors.append(
            "canonical/manifest.json: artifacts debe ser una lista de objetos"
        )
        return
    validate_lineage(manifest, descriptors, errors)

    paths: list[str] = []
    counts: dict[str, int] = {}
    envelopes: dict[str, dict] = {}
    envelopes["canonical_manifest"] = manifest
    records: dict[str, list[dict]] = {"__paths__": []}
    streamed_artifacts: dict[str, StreamedArtifact] = {}
    for descriptor in descriptors:
        artifact = read_artifact(canonical_dir, descriptor, match_id, errors)
        if artifact is None:
            continue
        path, spec, envelope, artifact_records, streamed = artifact
        paths.append(path)
        counts[spec.artifact_type] = counts.get(spec.artifact_type, 0) + 1
        if envelope is not None:
            envelopes[spec.artifact_type] = envelope
        if streamed is not None:
            streamed_artifacts[spec.artifact_type] = streamed
        records.setdefault(spec.artifact_type, []).extend(artifact_records)
        records["__paths__"].append((path, spec.artifact_type))

    if paths != sorted(paths):
        errors.append("canonical/manifest.json: artifacts no está ordenado por path")
    if len(paths) != len(set(paths)):
        errors.append("canonical/manifest.json: hay paths duplicados")
    for spec in ARTIFACT_SPECS:
        count = counts.get(spec.artifact_type, 0)
        if spec.is_required and count == 0:
            errors.append(f"canonical/manifest.json: falta {spec.artifact_type}")
        if spec.is_singleton and count > 1:
            errors.append(
                f"canonical/manifest.json: {spec.artifact_type} debe ser único"
            )
    disk_paths = {
        path.relative_to(canonical_dir).as_posix()
        for path in canonical_dir.rglob("*")
        if path.is_file() and path != manifest_path
    }
    if disk_paths != set(paths):
        errors.append(
            "canonical/manifest.json: los artefactos declarados no coinciden con los ficheros existentes"
        )
    if all(
        counts.get(spec.artifact_type, 0) for spec in ARTIFACT_SPECS if spec.is_required
    ):
        validate_references(
            match_id, envelopes, records, errors, streamed_artifacts
        )


def infer_match_id(match_dir: Path) -> str:
    return (
        match_dir.name.removeprefix("match_")
        if match_dir.name.startswith("match_")
        else ""
    )


def validate_match_export(
    match_dir: Path,
    expected_match_id: str | None = None,
    expected_demo_checksum: str | None = None,
) -> list[str]:
    match_dir = Path(match_dir)
    match_id = expected_match_id or infer_match_id(match_dir)
    if not match_id:
        return [f"{match_dir}: no se pudo inferir match_id"]
    if not match_dir.is_dir():
        return [f"{match_dir}: directorio inexistente"]
    errors = ValidationErrors()
    validate_root_catalog(match_dir, match_id, errors, expected_demo_checksum)
    validate_canonical_bundle(match_dir, match_id, errors)
    return errors.as_report()


def find_match_dirs(paths: list[Path]) -> list[Path]:
    match_dirs: list[Path] = []
    for path in paths:
        if path.is_dir() and path.name.startswith("match_"):
            match_dirs.append(path)
        elif path.is_dir():
            match_dirs.extend(
                sorted(
                    candidate
                    for candidate in path.glob("match_*")
                    if candidate.is_dir()
                )
            )
        else:
            match_dirs.append(path)
    return match_dirs


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "paths", nargs="+", type=Path, help="Carpetas match_* o directorios de exports."
    )
    match_dirs = find_match_dirs(parser.parse_args().paths)
    if not match_dirs:
        print("No se encontraron carpetas match_*.")
        return 2
    invalid = 0
    for match_dir in match_dirs:
        errors = validate_match_export(match_dir)
        if not errors:
            print(f"OK {match_dir}")
            continue
        invalid += 1
        print(f"ERROR {match_dir}")
        for error in errors:
            print(f"  - {error}")
    return 1 if invalid else 0


if __name__ == "__main__":
    raise SystemExit(main())
