"""Freeze opportunity records using the v2 initial-state gate.

All execution checkpoints and callbacks are labels. Inventory membership and
native counter differences never become an itemized purchase or an eco label.
"""

from __future__ import annotations

import copy
from collections import Counter

from ai_coach.contracts import economy_initial_frontiers as initial
from ai_coach.contracts.runtime_contracts import RuntimeContractError

VERSION = "1.0.0-candidate.1"
DATA_SCHEMA = "stratai.economy_opportunities@1"
MANIFEST_SCHEMA = "stratai.economy_opportunities_manifest@1"
DATA_FILE = "economy_opportunities.json"
MANIFEST_FILE = "economy_opportunities_manifest.json"
NUMERIC_FIELDS = ("money", "spent_native", "armor", "equipment_value_native")
BOOLEAN_FIELDS = ("helmet", "kit", "buy_zone")
RULE_REASONS = [
    "price_rules_not_bound_to_game_build",
    "reward_rules_and_prior_bonus_unverified",
    "loadout_and_item_limits_unobserved",
    "refund_rules_unverified",
    "teammate_cooperation_unobserved",
]
ACTION_REASONS = [
    "itemized_purchases_and_quantities_unresolved",
    "team_buy_band_not_inferred",
    "next_round_scenarios_unavailable",
]


def initial_gate(
    data: dict, round_: dict, frontier: dict, side: str
) -> tuple[dict | None, str]:
    if frontier["round_link"]["status"] != "verified":
        return None, "competitive_round_unavailable"
    actor = next(
        (a for a in frontier["round_start_probe"]["actors"] if a["side"] == side), None
    )
    if actor is None:
        return None, "initial_roster_unavailable"
    try:
        state = initial.initial_team_state(
            data, round_["instance"], actor["actor_alias"], round_["start_tick"]
        )
    except RuntimeContractError:
        return None, "complete_initial_team_unavailable"
    return state, "complete_competitive_initial_team"


def rule_assessment(state: dict | None, data: dict, protocol: dict) -> dict:
    missing = (
        protocol["rules"]["server_convars"]
        if state is None
        else [
            name
            for name in protocol["rules"]["server_convars"]
            if state["context"][name]["status"] != "observed"
        ]
    )
    return {
        "status": "unverified_for_comparison",
        "server_configuration": "unavailable"
        if state is None
        else "partial"
        if missing
        else "observed",
        "missing_convars": list(missing),
        "initial_buy_zone": "unavailable"
        if state is None
        else "all_in_buy_zone"
        if all(a["facts"]["buy_zone"]["value"] is True for a in state["actors"])
        else "not_all_in_buy_zone",
        "legacy_references": copy.deepcopy(data["configs"]),
        "blockers": list(RULE_REASONS),
        "comparison_allowed": False,
    }


def execution_window(round_: dict) -> dict:
    end = round_["freeze_end_tick"]
    valid = end is not None and end > round_["start_tick"]
    return {
        "status": "observed" if valid else "unavailable",
        "start_exclusive_tick": round_["start_tick"],
        "end_exclusive_tick": end if valid else None,
        "duration_ticks": end - round_["start_tick"] if valid else None,
        "source": "RoundStart+RoundFreezetimeEnd",
        "partition": "execution_label_only",
        "post_freeze_policy": "excluded_not_buytime_end",
    }


def _checkpoint_actor(
    snapshot: dict | None, alias: str, side: str, end: int | None
) -> dict | None:
    if snapshot is None or end is None or snapshot["tick"] != end:
        return None
    actor = next((a for a in snapshot["actors"] if a["actor_alias"] == alias), None)
    if actor is None or not actor["connected"] or actor["side"] != side:
        return None
    return actor


def actor_changes(actor: dict, final_actor: dict | None, end: int | None) -> dict:
    before, after = actor["facts"], final_actor["facts"] if final_actor else {}
    changes = {}
    for key in (*NUMERIC_FIELDS, *BOOLEAN_FIELDS):
        observed = final_actor is not None and after[key]["status"] == "observed"
        changes[key] = {
            "before": before[key]["value"],
            "after": after[key]["value"] if observed else None,
            "delta": after[key]["value"] - before[key]["value"]
            if observed and key in NUMERIC_FIELDS
            else None,
            "status": "checkpoint_difference" if observed else "unavailable",
            "availability_tick": end if observed else None,
        }
    inventory = after.get("inventory")
    observed = inventory is not None and inventory["status"] == "observed"
    old = {w["entity_alias"]: w for w in before["inventory"]["value"]}
    new = {w["entity_alias"]: w for w in inventory["value"]} if observed else {}
    # Identity reuse with a different weapon cannot be silently called retained.
    conflicts = sorted(k for k in old.keys() & new.keys() if old[k] != new[k])
    valid = observed and not conflicts
    transition = {
        "status": "entity_membership_difference" if valid else "unavailable",
        "retained": [old[k] for k in sorted(old.keys() & new.keys())]
        if valid
        else None,
        "added": [new[k] for k in sorted(new.keys() - old.keys())] if valid else None,
        "removed": [old[k] for k in sorted(old.keys() - new.keys())] if valid else None,
        "identity_conflicts": conflicts,
        "availability_tick": end if valid else None,
    }
    return {
        "actor_alias": actor["actor_alias"],
        "changes": changes,
        "inventory_transition": transition,
    }


def execution_labels(
    round_: dict, frontier: dict, state: dict | None, side: str, window: dict
) -> dict:
    aliases = {
        a["actor_alias"]
        for a in frontier["round_start_probe"]["actors"]
        if a["side"] == side
    }
    end, start = window["end_exclusive_tick"], round_["start_tick"]
    all_events = initial._events(round_)
    owned = [e for e in all_events if e["actor_alias"] in aliases]
    events = [e for e in owned if end is not None and start < e["tick"] < end]
    boundary = [
        e for e in owned if e["tick"] == start or (end is not None and e["tick"] == end)
    ]
    events.sort(key=lambda e: e["callback_sequence"])
    boundary.sort(key=lambda e: e["callback_sequence"])
    side_mismatch = any(e["side"] != side for e in [*events, *boundary])
    snapshot = round_["execution"]["freeze_snapshot"]
    checkpoint = None
    if snapshot is not None and end is not None and snapshot["tick"] == end:
        # Keep only the actors of this initial team, even if their observed side
        # later differs. A mismatch remains an explicit unavailable comparison.
        checkpoint = {
            "tick": snapshot["tick"],
            "callback_sequence": snapshot["callback_sequence"],
            "actors": [
                copy.deepcopy(a)
                for a in snapshot["actors"]
                if a["actor_alias"] in aliases
            ],
        }
    changes = []
    if state is not None:
        for actor in state["actors"]:
            final_actor = _checkpoint_actor(snapshot, actor["actor_alias"], side, end)
            changes.append(actor_changes(actor, final_actor, end))
    blockers = list(ACTION_REASONS)
    if state is None:
        blockers.append("initial_team_unavailable")
    if window["status"] != "observed":
        blockers.append("freeze_window_unavailable")
    if checkpoint is None:
        blockers.append("exact_freeze_checkpoint_unavailable")
    if side_mismatch:
        blockers.append("execution_actor_side_changed")
    if boundary:
        blockers.append("boundary_callbacks_have_no_subtick_order")
    return {
        "status": "partial"
        if state is not None
        and checkpoint is not None
        and window["status"] == "observed"
        else "unavailable",
        "action_id": None,
        "basket": None,
        "semantics": "checkpoint_differences_and_callbacks_not_itemized_transactions",
        "checkpoint": checkpoint,
        "actor_changes": changes,
        "callbacks": copy.deepcopy(events),
        "boundary_callbacks": copy.deepcopy(boundary),
        "pre_probe_callbacks": [
            copy.deepcopy(e)
            for e in frontier["pre_probe_callbacks"]
            if e["actor_alias"] in aliases
        ],
        "blockers": blockers,
    }


def build_opportunities(
    data: dict,
    protocol: dict,
    source_manifest_sha256: str,
    source_data_sha256: str,
    protocol_sha256: str,
    catalog_version: str,
    catalog_sha256: str,
) -> dict:
    result = {
        "schema_id": DATA_SCHEMA,
        "contract_version": VERSION,
        "source_alias": data["source_alias"],
        "source_sha256": data["source_sha256"],
        "map_id": data["map_id"],
        "tick_rate": data["tick_rate"],
        "source_candidate_manifest_sha256": source_manifest_sha256,
        "source_candidate_data_sha256": source_data_sha256,
        "protocol_sha256": protocol_sha256,
        "catalog_version": catalog_version,
        "catalog_sha256": catalog_sha256,
        "capability_id": protocol["capability_id"],
        "decision_id": protocol["decision_id"],
        "training_allowed": False,
        "product_supported": False,
        "comparison_allowed": False,
        "opportunities": [],
    }
    for round_, frontier in zip(data["rounds"], data["initial_frontiers"], strict=True):
        for side in ("CT", "T"):
            state, reason = initial_gate(data, round_, frontier, side)
            window = execution_window(round_)
            result["opportunities"].append(
                {
                    "opportunity_id": f"{data['source_alias']}:economy:{round_['instance']:05d}:{side}",
                    "instance": round_["instance"],
                    "round_number": round_["round_number"],
                    "side": side,
                    "t0_tick": round_["start_tick"],
                    "initial_gate": {
                        "status": "available" if state else "unavailable",
                        "reason": reason,
                    },
                    "initial_state": state,
                    "execution_window": window,
                    "rules": rule_assessment(state, data, protocol),
                    "observed_action": execution_labels(
                        round_, frontier, state, side, window
                    ),
                }
            )
    return result


def summary(data: dict) -> dict:
    counts = Counter(
        {
            k: 0
            for k in (
                "team_round_records",
                "initial_teams_available",
                "initial_teams_unavailable",
                "observed_windows",
                "partial_execution_records",
                "actor_checkpoint_records",
                "observed_numeric_differences",
                "inventory_transitions",
                "execution_callbacks",
                "boundary_callbacks",
                "rules_compatible_for_comparison",
                "classified_actions",
            )
        }
    )
    for row in data["opportunities"]:
        counts["team_round_records"] += 1
        counts[
            "initial_teams_available"
            if row["initial_gate"]["status"] == "available"
            else "initial_teams_unavailable"
        ] += 1
        counts["observed_windows"] += row["execution_window"]["status"] == "observed"
        execution = row["observed_action"]
        counts["partial_execution_records"] += execution["status"] == "partial"
        counts["actor_checkpoint_records"] += len(execution["actor_changes"])
        counts["execution_callbacks"] += len(execution["callbacks"])
        counts["boundary_callbacks"] += len(execution["boundary_callbacks"])
        for actor in execution["actor_changes"]:
            counts["observed_numeric_differences"] += sum(
                actor["changes"][k]["status"] == "checkpoint_difference"
                for k in NUMERIC_FIELDS
            )
            counts["inventory_transitions"] += (
                actor["inventory_transition"]["status"]
                == "entity_membership_difference"
            )
    return dict(counts)


def initial_features(row: dict) -> dict:
    """Feature-only read of a previously validated opportunity; never labels."""
    if row["initial_gate"]["status"] != "available" or row["initial_state"] is None:
        raise RuntimeContractError("economic initial team unavailable")
    return copy.deepcopy(row["initial_state"])
