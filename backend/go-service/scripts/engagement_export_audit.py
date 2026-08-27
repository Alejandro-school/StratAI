#!/usr/bin/env python3
"""Independent semantic audit for canonical Block 5 engagement artifacts."""

from __future__ import annotations

import argparse
import json
import math
import sys
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any

HARD_GATES = (
    "engagement_event_contract",
    "engagement_atomic_provenance",
    "engagement_participant_reconciliation",
    "engagement_role_consistency",
    "engagement_temporal_consistency",
    "engagement_causal_availability",
    "engagement_trade_reconciliation",
    "engagement_stats_reconciliation",
    "engagement_determinism",
)


def read_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    with path.open(encoding="utf-8") as handle:
        return [json.loads(line) for line in handle if line.strip()]


def canonical_root(export_dir: Path) -> Path:
    nested = export_dir / "canonical"
    return nested if nested.is_dir() else export_dir


def role_player(engagement: dict[str, Any], field: str) -> str | None:
    role = engagement.get(field)
    return role.get("player_id") if isinstance(role, dict) else None


def event_key(event: dict[str, Any]) -> tuple[int, int, int, str]:
    return (
        event["round_number"],
        event["tick"],
        event["sequence_in_tick"],
        event["event_id"],
    )


def closure(event_id: str, events: dict[str, dict[str, Any]]) -> set[str]:
    pending = [event_id]
    result: set[str] = set()
    while pending:
        current = pending.pop()
        if current in result:
            continue
        result.add(current)
        event = events.get(current)
        if event:
            pending.extend(event.get("source_event_ids") or [])
    return result


def audit(export_dir: Path) -> dict[str, Any]:
    root = canonical_root(export_dir)
    engagements_doc = read_json(root / "derived" / "engagements.json")
    trades_doc = read_json(root / "derived" / "trades.json")
    stats_doc = read_json(root / "derived" / "player_stats.json")
    participants_doc = read_json(root / "core" / "participants.json")
    quality_doc = read_json(root / "diagnostics" / "quality_report.json")
    quality = quality_doc["report"]
    combat = read_jsonl(root / "events" / "combat_events.jsonl")
    states = []
    for path in sorted((root / "states" / "player_states").glob("round_*.jsonl")):
        states.extend(read_jsonl(path))

    errors: list[str] = []

    def require(condition: bool, message: str) -> None:
        if not condition:
            errors.append(message)

    require(
        engagements_doc.get("schema_id") == "stratai.engagements@2",
        "engagement schema is not stratai.engagements@2",
    )
    require(
        trades_doc.get("schema_id") == "stratai.trades@1",
        "trade schema is not stratai.trades@1",
    )
    require(
        stats_doc.get("schema_id") == "stratai.player_stats@1",
        "player stats schema is not stratai.player_stats@1",
    )

    events = {event["event_id"]: event for event in combat}
    roster = {player["player_id"] for player in participants_doc["players"]}
    state_by_id = {state["state_id"]: state for state in states}
    engagements = engagements_doc.get("engagements") or []
    candidates = trades_doc.get("trade_candidates") or []
    completions = trades_doc.get("trade_completions") or []

    enemy_hurts = {
        event["event_id"]: event
        for event in combat
        if event.get("event_type") == "player_hurt"
        and event.get("relation") == "enemy"
        and event.get("actor_player_id")
        and event.get("target_player_id")
    }
    enemy_kills = {
        event["event_id"]: event
        for event in combat
        if event.get("event_type") == "kill"
        and event.get("relation") == "enemy"
        and event.get("actor_player_id")
        and event.get("target_player_id")
    }
    hurt_owners: Counter[str] = Counter()
    source_owners: dict[str, str] = {}
    engagement_ids: set[str] = set()

    for engagement in engagements:
        engagement_id = engagement.get("engagement_id", "<missing>")
        require(engagement_id not in engagement_ids, f"duplicate engagement {engagement_id}")
        engagement_ids.add(engagement_id)
        start = (engagement.get("start_tick"), engagement.get("start_sequence_in_tick"))
        end = (engagement.get("end_tick"), engagement.get("end_sequence_in_tick"))
        require(start <= end, f"{engagement_id}: end precedes start")
        require(
            engagement.get("duration_ms", -1) >= 0,
            f"{engagement_id}: negative duration",
        )

        sources = engagement.get("source_event_ids") or []
        require(len(sources) == len(set(sources)), f"{engagement_id}: duplicate sources")
        for source_id in sources:
            require(source_id in events, f"{engagement_id}: missing source {source_id}")
            previous = source_owners.setdefault(source_id, engagement_id)
            require(
                previous == engagement_id,
                f"source {source_id} reused by {previous} and {engagement_id}",
            )

        participant_ids = [item.get("player_id") for item in engagement["participants"]]
        require(
            len(participant_ids) == len(set(participant_ids)),
            f"{engagement_id}: duplicate participant",
        )
        require(
            all(player_id in roster for player_id in participant_ids),
            f"{engagement_id}: participant outside roster",
        )

        exchange_players: set[str] = set()
        exchange_ids: set[str] = set()
        terminal_kills: list[str] = []
        exchange_events: list[dict[str, Any]] = []
        for exchange in engagement["exchanges"]:
            exchange_id = exchange.get("exchange_id", "<missing>")
            require(
                exchange_id not in exchange_ids,
                f"{engagement_id}: duplicate exchange {exchange_id}",
            )
            exchange_ids.add(exchange_id)
            exchange_players.update(
                (exchange.get("actor_player_id"), exchange.get("target_player_id"))
            )
            exchange_sources = set(exchange.get("source_event_ids") or [])
            hurts = [
                events[source_id]
                for source_id in exchange_sources
                if source_id in enemy_hurts
            ]
            require(
                len(hurts) == 1,
                f"{engagement_id}/{exchange_id}: expected one enemy hurt source",
            )
            if len(hurts) != 1:
                continue
            hurt = hurts[0]
            hurt_owners[hurt["event_id"]] += 1
            exchange_events.append(hurt)
            for field in (
                "tick",
                "sequence_in_tick",
                "sequence_in_round",
                "actor_player_id",
                "target_player_id",
                "weapon",
                "health_damage",
                "health_damage_taken",
                "armor_damage",
                "armor_damage_taken",
                "health_before",
                "health_after",
                "armor_before",
                "armor_after",
                "hitgroup",
                "is_headshot",
                "shot_id",
            ):
                require(
                    exchange.get(field) == hurt.get(field),
                    f"{engagement_id}/{exchange_id}: {field} differs from atomic hurt",
                )
            expected_sources = closure(hurt["event_id"], events)
            kill_id = exchange.get("kill_event_id")
            if exchange.get("is_kill"):
                require(kill_id in enemy_kills, f"{engagement_id}: invalid kill {kill_id}")
                if kill_id in enemy_kills:
                    expected_sources.update(closure(kill_id, events))
                    terminal_kills.append(kill_id)
            else:
                require(kill_id is None, f"{engagement_id}: non-kill has kill_event_id")
            require(
                exchange_sources == expected_sources,
                f"{engagement_id}/{exchange_id}: incomplete atomic closure",
            )

        require(
            set(participant_ids) == exchange_players,
            f"{engagement_id}: participants do not match exchange actors/targets",
        )
        require(
            sources and set(sources) == set().union(
                *(set(item.get("source_event_ids") or []) for item in engagement["exchanges"]),
                *(set(
                    (engagement.get(role) or {}).get("source_event_ids") or []
                ) for role in ("initiator", "first_aggressor")),
            ),
            f"{engagement_id}: engagement provenance is not exact",
        )

        if exchange_events:
            first_hurt = min(exchange_events, key=event_key)
            require(
                role_player(engagement, "first_damage_dealer")
                == first_hurt["actor_player_id"],
                f"{engagement_id}: first damage dealer is not earliest hurt actor",
            )
            aggressor = role_player(engagement, "first_aggressor")
            expected_initiator = aggressor or first_hurt["actor_player_id"]
            require(
                role_player(engagement, "initiator") == expected_initiator,
                f"{engagement_id}: initiator does not follow causal role precedence",
            )
        outcome = engagement["outcome_context"]
        require(
            set(outcome.get("terminal_kill_event_ids") or []) == set(terminal_kills),
            f"{engagement_id}: terminal kills do not reconcile",
        )
        causal = engagement["causal_context"]
        require(
            causal.get("t0_tick") == engagement.get("start_tick"),
            f"{engagement_id}: causal t0 differs from start",
        )
        for state_projection in causal.get("participant_states") or []:
            state_id = state_projection.get("state_id")
            if state_projection.get("status") == "unavailable":
                require(state_id is None, f"{engagement_id}: unavailable state has ID")
                require(
                    state_projection.get("movement_classification") is None,
                    f"{engagement_id}: unavailable velocity fabricated movement",
                )
                continue
            state = state_by_id.get(state_id)
            require(state is not None, f"{engagement_id}: missing causal state {state_id}")
            if state:
                require(
                    state["player_id"] == state_projection["player_id"],
                    f"{engagement_id}: causal state belongs to another player",
                )
                require(
                    state["tick"] <= causal["t0_tick"],
                    f"{engagement_id}: future state leaked into causal context",
                )
            if state_projection.get("velocity_status") == "unavailable":
                require(
                    state_projection.get("movement_classification") is None,
                    f"{engagement_id}: unavailable velocity fabricated movement",
                )

    require(
        set(hurt_owners) == set(enemy_hurts),
        "enemy hurt ledger is not fully partitioned into engagements",
    )
    for event_id, count in hurt_owners.items():
        require(count == 1, f"enemy hurt {event_id} belongs to {count} exchanges")

    candidate_by_id = {item["trade_candidate_id"]: item for item in candidates}
    candidate_kills = [item.get("original_kill_event_id") for item in candidates]
    require(
        set(candidate_kills) == set(enemy_kills) and len(candidate_kills) == len(enemy_kills),
        "trade candidates do not reconcile one-to-one with enemy kills",
    )
    completion_by_id = {item["trade_completion_id"]: item for item in completions}
    used_response_kills: set[str] = set()
    window_ms = trades_doc["config"]["trade_window_ms"]
    tick_rate = trades_doc["config"]["tick_rate_hz"]
    expected_window_ticks = math.ceil(window_ms * tick_rate / 1000)
    require(
        trades_doc["config"]["trade_window_ticks"] == expected_window_ticks,
        "trade window ticks were not derived from milliseconds and tick rate",
    )

    for candidate in candidates:
        candidate_id = candidate["trade_candidate_id"]
        kill = enemy_kills.get(candidate["original_kill_event_id"])
        require(kill is not None, f"{candidate_id}: missing original kill")
        if not kill:
            continue
        require(
            candidate["death_tick"] == kill["tick"]
            and candidate["original_victim_player_id"] == kill["target_player_id"]
            and candidate["original_killer_player_id"] == kill["actor_player_id"],
            f"{candidate_id}: original death projection differs from kill",
        )
        require(
            candidate["window_ticks"] == expected_window_ticks
            and candidate["window_end_tick"] == kill["tick"] + expected_window_ticks,
            f"{candidate_id}: inconsistent trade window",
        )
        completion_id = candidate.get("trade_completion_id")
        require(
            (candidate["evaluation"] == "completed") == (completion_id is not None),
            f"{candidate_id}: completion/evaluation contradiction",
        )
        if completion_id:
            require(
                completion_id in completion_by_id,
                f"{candidate_id}: completion link does not exist",
            )

    for completion in completions:
        completion_id = completion["trade_completion_id"]
        candidate = candidate_by_id.get(completion["trade_candidate_id"])
        response = enemy_kills.get(completion["response_kill_event_id"])
        require(candidate is not None, f"{completion_id}: missing candidate")
        require(response is not None, f"{completion_id}: missing response kill")
        if not candidate or not response:
            continue
        require(
            response["event_id"] not in used_response_kills,
            f"response kill {response['event_id']} reused by incompatible trades",
        )
        used_response_kills.add(response["event_id"])
        require(
            response["tick"] >= candidate["death_tick"]
            and response["tick"] <= candidate["window_end_tick"],
            f"{completion_id}: response outside trade window",
        )
        require(
            response["target_player_id"] == candidate["original_killer_player_id"],
            f"{completion_id}: response did not kill original killer",
        )
        require(
            response["actor_player_id"] == completion["trader_player_id"]
            and completion["trader_player_id"]
            in candidate["eligible_teammate_player_ids"],
            f"{completion_id}: trader was not an eligible teammate",
        )

    expected_stats: dict[str, Counter[str]] = defaultdict(Counter)
    for candidate in candidates:
        victim = candidate["original_victim_player_id"]
        evaluation = candidate["evaluation"]
        actors = {
            events[event_id].get("actor_player_id")
            for event_id in candidate.get("attempt_event_ids") or []
            if event_id in events
        }
        if evaluation == "completed":
            completion = completion_by_id[candidate["trade_completion_id"]]
            trader = completion["trader_player_id"]
            expected_stats[victim]["traded_deaths"] += 1
            expected_stats[trader]["trade_kills"] += 1
            actors.add(trader)
        elif evaluation == "not_tradeable":
            expected_stats[victim]["untradeable_deaths"] += 1
        elif evaluation == "not_evaluable":
            expected_stats[victim]["non_evaluable_trade_deaths"] += 1
        for actor in actors - {None}:
            expected_stats[actor]["trade_attempts"] += 1
            if evaluation == "failed":
                expected_stats[actor]["failed_trade_attempts"] += 1

    metric_names = (
        "trade_kills",
        "traded_deaths",
        "trade_attempts",
        "failed_trade_attempts",
        "untradeable_deaths",
        "non_evaluable_trade_deaths",
    )
    for player in stats_doc.get("players") or []:
        player_id = player["player_id"]
        actual = player["metrics"]
        for metric in metric_names:
            require(
                actual.get(metric) == expected_stats[player_id][metric],
                f"stats {player_id}/{metric} do not reconcile",
            )

    checks = {check["name"]: check for check in quality.get("checks") or []}
    for gate in HARD_GATES:
        require(gate in checks, f"quality report missing {gate}")
        if gate in checks:
            require(checks[gate].get("status") == "pass", f"quality gate {gate} failed")
    require(
        quality.get("usable_for_training") is True,
        "quality report is not usable for training",
    )

    role_metrics = Counter()
    for engagement in engagements:
        initiator = role_player(engagement, "initiator")
        aggressor = role_player(engagement, "first_aggressor")
        first_damage = role_player(engagement, "first_damage_dealer")
        winner = engagement["outcome_context"].get("winner_player_id")
        role_metrics["initiator_unavailable"] += initiator is None
        role_metrics["first_aggressor_unavailable"] += aggressor is None
        role_metrics["first_damage_unavailable"] += first_damage is None
        role_metrics["initiator_differs_from_first_damage"] += (
            initiator is not None and first_damage is not None and initiator != first_damage
        )
        role_metrics["initiator_differs_from_winner"] += (
            initiator is not None and winner is not None and initiator != winner
        )
        role_metrics["first_aggressor_differs_from_first_damage"] += (
            aggressor is not None
            and first_damage is not None
            and aggressor != first_damage
        )

    hard_metric_values = {
        key: value
        for key, value in quality.items()
        if key.startswith("engagement_")
        and key.endswith(("_violations", "_mismatches"))
    }
    result = {
        "status": "pass" if not errors else "fail",
        "match_id": engagements_doc.get("match_id"),
        "schemas": {
            "engagements": engagements_doc.get("schema_id"),
            "trades": trades_doc.get("schema_id"),
            "player_match_stats": stats_doc.get("schema_id"),
        },
        "metrics": {
            "atomic_combat_events": len(combat),
            "enemy_hurts": len(enemy_hurts),
            "enemy_kills": len(enemy_kills),
            "engagements": len(engagements),
            "exchanges": sum(len(item["exchanges"]) for item in engagements),
            "engagement_types": dict(
                sorted(Counter(item["engagement_type"] for item in engagements).items())
            ),
            "outcomes": dict(
                sorted(
                    Counter(
                        item["outcome_context"]["outcome"] for item in engagements
                    ).items()
                )
            ),
            "roles": dict(sorted(role_metrics.items())),
            "trade_candidates": len(candidates),
            "trade_completions": len(completions),
            "trade_evaluations": dict(
                sorted(Counter(item["evaluation"] for item in candidates).items())
            ),
        },
        "quality": {
            "hard_gates": {gate: checks.get(gate, {}).get("status") for gate in HARD_GATES},
            "hard_metrics": dict(sorted(hard_metric_values.items())),
            "warnings": quality.get("warnings") or [],
        },
        "errors": errors[:100],
        "error_count": len(errors),
    }
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("export_dir", type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    try:
        result = audit(args.export_dir.resolve())
    except (OSError, KeyError, TypeError, ValueError, json.JSONDecodeError) as error:
        print(f"AUDIT ERROR: {error}", file=sys.stderr)
        return 2
    payload = json.dumps(result, indent=2, sort_keys=True, ensure_ascii=False) + "\n"
    if args.output:
        args.output.write_text(payload, encoding="utf-8")
    print(payload, end="")
    return 0 if result["status"] == "pass" else 1


if __name__ == "__main__":
    raise SystemExit(main())
