#!/usr/bin/env python3
"""Independent semantic reconciliation for canonical combat exports."""

from __future__ import annotations

import argparse
import gzip
import json
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any

EVENT_TYPES = {
    "weapon_equip",
    "weapon_reload",
    "weapon_fire",
    "bullet_damage",
    "player_hurt",
    "kill",
}
HITGROUPS = {
    "generic",
    "head",
    "chest",
    "stomach",
    "left_arm",
    "right_arm",
    "left_leg",
    "right_leg",
    "neck",
    "gear",
}
DISCARD_REASONS = {"warmup", "outside_official_round", "invalid_round_or_tick"}


def read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line]


def replay_rounds(root: Path) -> list[dict[str, Any]]:
    rounds = []
    for path in sorted((root / "presentation" / "replay").glob("round_*.json.gz")):
        with gzip.open(path, "rt", encoding="utf-8") as file:
            envelope = json.load(file)
        rounds.append(envelope["round"])
    return rounds


def bare_player_id(player_id: str | None) -> str | None:
    if not player_id:
        return None
    return player_id.removeprefix("steam:")


def empty_summary() -> dict[str, Any]:
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
        "body_part_hits": Counter(),
        "weapon_stats": defaultdict(Counter),
    }


def aggregate(events: list[dict[str, Any]], participant_ids: set[str]) -> dict[str, dict[str, Any]]:
    summaries = {player_id: empty_summary() for player_id in participant_ids}
    damage_groups: dict[tuple[Any, ...], dict[str, Any]] = {}
    for event in events:
        actor = event.get("actor_player_id")
        target = event.get("target_player_id")
        assister = event.get("assister_player_id")
        event_type = event["event_type"]
        weapon = event.get("weapon") or ""
        non_utility = event.get("weapon_is_utility") is not True
        if event_type == "weapon_fire" and actor in summaries and non_utility:
            summary = summaries[actor]
            summary["shots_fired"] += 1
            summary["weapon_stats"][weapon]["shots_fired"] += 1
            if event.get("shot_result") == "hit":
                summary["shots_hit"] += 1
                summary["weapon_stats"][weapon]["shots_hit"] += 1
            elif event.get("shot_result") == "miss":
                summary["shots_missed"] += 1
                summary["weapon_stats"][weapon]["shots_missed"] += 1
        elif event_type == "kill":
            if target in summaries:
                summaries[target]["deaths_observed"] += 1
            if event.get("relation") == "enemy" and actor in summaries:
                summaries[actor]["kills_observed"] += 1
                if event.get("is_headshot") is True:
                    summaries[actor]["headshots"] += 1
                if event.get("weapon_status") == "observed" and non_utility:
                    summaries[actor]["weapon_stats"][weapon]["kills"] += 1
                    if event.get("is_headshot") is True:
                        summaries[actor]["weapon_stats"][weapon]["headshots"] += 1
                if (
                    assister in summaries
                    and assister not in {actor, target}
                    and event.get("assister_side") == event.get("actor_side")
                ):
                    summaries[assister]["assists_observed"] += 1
                    if event.get("assisted_flash") is True:
                        summaries[assister]["flash_assists"] += 1
        if event_type != "player_hurt" or actor not in summaries:
            continue
        key = (
            event["round_number"],
            event["tick"],
            actor,
            target,
            weapon,
            event["relation"],
        )
        group = damage_groups.setdefault(
            key,
            {"before": [], "after": [], "fallback": 0, "weapon_observed": False, "non_utility": non_utility},
        )
        group["before"].append(event["health_before"])
        group["after"].append(event["health_after"])
        group["fallback"] += event["health_damage_taken"]
        group["weapon_observed"] = event.get("weapon_status") == "observed"
        hitgroup = event.get("hitgroup")
        if event["relation"] == "enemy" and event["health_damage_taken"] > 0 and hitgroup:
            summaries[actor]["body_part_hits"][hitgroup] += 1
    for key, group in damage_groups.items():
        _, _, actor, _, weapon, relation = key
        damage = max(group["before"]) - min(group["after"])
        if damage < 0:
            damage = group["fallback"]
        field = {
            "enemy": "combat_damage_observed",
            "friendly": "friendly_damage",
            "self": "self_damage",
        }.get(relation)
        if field:
            summaries[actor][field] += damage
        if relation == "enemy" and group["weapon_observed"] and group["non_utility"]:
            summaries[actor]["weapon_stats"][weapon]["damage"] += damage
    return summaries


def relation(event: dict[str, Any]) -> str:
    actor = event.get("actor_player_id")
    target = event.get("target_player_id")
    if not actor:
        return "world" if target else "unknown"
    if not target:
        return "unknown"
    if actor == target:
        return "self"
    actor_side, target_side = event.get("actor_side"), event.get("target_side")
    if actor_side and target_side:
        return "friendly" if actor_side == target_side else "enemy"
    return "unknown"


def audit_match(match_dir: Path) -> tuple[dict[str, Any], list[str]]:
    root = match_dir / "canonical"
    events = read_jsonl(root / "events" / "combat_events.jsonl")
    participants = read_json(root / "core" / "participants.json")["players"]
    player_stats = read_json(root / "derived" / "player_stats.json")["players"]
    quality = read_json(root / "diagnostics" / "quality_report.json")["report"]
    rounds = replay_rounds(root)
    participant_ids = {player["player_id"] for player in participants}
    errors: list[str] = []
    event_ids = [event["event_id"] for event in events]
    by_id = {event["event_id"]: event for event in events}
    index_by_id = {event_id: index for index, event_id in enumerate(event_ids)}
    if len(event_ids) != len(set(event_ids)):
        errors.append("duplicate combat event IDs")
    previous_key: tuple[int, int, int, int] | None = None
    source_links = 0
    for index, event in enumerate(events):
        key = (event["round_number"], event["tick"], event["sequence_in_tick"], event["sequence_in_round"])
        if previous_key is not None and key < previous_key:
            errors.append(f"non-deterministic total order at {event['event_id']}")
        previous_key = key
        if event["event_type"] not in EVENT_TYPES:
            errors.append(f"unknown event type at {event['event_id']}")
        for field in ("actor_player_id", "target_player_id", "assister_player_id"):
            player_id = event.get(field)
            if player_id is not None and player_id not in participant_ids:
                errors.append(f"invalid participant {player_id} at {event['event_id']}")
        if event.get("relation") != relation(event):
            errors.append(f"relation mismatch at {event['event_id']}")
        if event["event_type"] == "player_hurt":
            if event["health_before"] - event["health_after"] != event["health_damage_taken"]:
                errors.append(f"health transition mismatch at {event['event_id']}")
            if event["armor_before"] - event["armor_after"] != event["armor_damage_taken"]:
                errors.append(f"armor transition mismatch at {event['event_id']}")
            hitgroup = event.get("hitgroup")
            if hitgroup not in HITGROUPS and not (
                isinstance(hitgroup, str) and hitgroup.startswith("unknown_") and hitgroup[8:].isdigit()
            ):
                errors.append(f"non-semantic hitgroup at {event['event_id']}: {hitgroup!r}")
            if event.get("is_headshot") != (hitgroup == "head"):
                errors.append(f"hurt headshot mismatch at {event['event_id']}")
        for source_id in event["source_event_ids"]:
            source_links += 1
            source = by_id.get(source_id)
            if source is None or index_by_id[source_id] > index:
                errors.append(f"invalid causal source at {event['event_id']}: {source_id}")
            elif (source["round_number"], source["tick"]) > (event["round_number"], event["tick"]):
                errors.append(f"future causal source at {event['event_id']}: {source_id}")
    expected = aggregate(events, participant_ids)
    stats_by_id = {row["player_id"]: row["metrics"] for row in player_stats}
    scalar_fields = (
        "kills_observed",
        "deaths_observed",
        "assists_observed",
        "flash_assists",
        "headshots",
        "combat_damage_observed",
        "friendly_damage",
        "self_damage",
        "shots_fired",
        "shots_hit",
        "shots_missed",
    )
    for player_id, summary in expected.items():
        actual = stats_by_id.get(player_id)
        if actual is None:
            errors.append(f"missing player stats for {player_id}")
            continue
        for field in scalar_fields:
            if actual.get(field) != summary[field]:
                errors.append(f"stats mismatch {player_id} {field}: {actual.get(field)} != {summary[field]}")
        if dict(actual.get("body_part_hits", {})) != dict(summary["body_part_hits"]):
            errors.append(f"body-part stats mismatch for {player_id}")
        expected_weapons = {
            weapon: {field: values.get(field, 0) for field in ("kills", "headshots", "damage", "shots_fired", "shots_hit", "shots_missed")}
            for weapon, values in summary["weapon_stats"].items()
        }
        actual_weapons = {
            weapon: {field: values.get(field, 0) for field in ("kills", "headshots", "damage", "shots_fired", "shots_hit", "shots_missed")}
            for weapon, values in actual.get("weapon_stats", {}).items()
        }
        if actual_weapons != expected_weapons:
            errors.append(f"weapon stats mismatch for {player_id}")
        for observed, native, delta in (
            ("kills_observed", "kills", "kills_native_minus_observed"),
            ("deaths_observed", "deaths", "deaths_native_minus_observed"),
            ("assists_observed", "assists", "assists_native_minus_observed"),
            ("combat_damage_observed", "total_damage", "combat_damage_unattributed_delta"),
        ):
            if actual["native_scoreboard"][native] - actual[observed] != actual[delta]:
                errors.append(f"native delta mismatch {player_id} {delta}")
    markers = {event_id: 0 for event_id, event in by_id.items() if event["event_type"] in {"player_hurt", "kill"}}
    fires = {event_id: event for event_id, event in by_id.items() if event["event_type"] == "weapon_fire"}
    fire_seen: set[str] = set()
    replay_shot_rows = 0
    for replay_round in rounds:
        for marker in replay_round.get("events", []):
            if marker.get("type") not in {"player_hurt", "kill"}:
                continue
            source_ids = marker.get("source_event_ids", [])
            if len(source_ids) != 1 or source_ids[0] not in markers:
                errors.append(f"invalid replay marker provenance in round {replay_round['round']}")
                continue
            source = by_id[source_ids[0]]
            markers[source_ids[0]] += 1
            if marker.get("killer_id") != bare_player_id(source.get("actor_player_id")):
                errors.append(f"replay marker actor mismatch for {source_ids[0]}")
            if marker.get("victim_id") != bare_player_id(source.get("target_player_id")):
                errors.append(f"replay marker target mismatch for {source_ids[0]}")
        for shot in replay_round.get("combat_shots", []):
            source = fires.get(shot.get("source_event_id"))
            if source is None or shot.get("shot_id") != source.get("shot_id"):
                errors.append(f"invalid combat_shot provenance in round {replay_round['round']}")
                continue
            expected_result = source.get("shot_result") or "unavailable"
            if shot.get("result") != expected_result or shot.get("result_status") != source.get("shot_result_status"):
                errors.append(f"combat_shot outcome mismatch for {source['event_id']}")
            if source["event_id"] in fire_seen:
                errors.append(f"duplicate combat_shot for {source['event_id']}")
            fire_seen.add(source["event_id"])
        for frame in replay_round.get("frames", []):
            for shot in frame.get("shots", []):
                replay_shot_rows += 1
                source = fires.get(shot.get("source_event_id"))
                if source is None or shot.get("shot_id") != source.get("shot_id"):
                    errors.append(f"invalid replay shot provenance in round {replay_round['round']}")
                    continue
                expected_result = source.get("shot_result") or "unavailable"
                if shot.get("result") != expected_result or shot.get("result_status") != source.get("shot_result_status"):
                    errors.append(f"replay shot outcome mismatch for {source['event_id']}")
    if any(count != 1 for count in markers.values()):
        errors.append("replay combat markers are omitted or duplicated")
    if set(fires) != fire_seen:
        errors.append("replay omits weapon_fire records")
    diagnostics = quality.get("combat_callback_diagnostics", {})
    discarded_total = 0
    for event_type in EVENT_TYPES:
        counts = diagnostics.get(event_type, {})
        discarded_total += counts.get("discarded", 0)
        if counts.get("observed") != counts.get("recorded", 0) + counts.get("discarded", 0):
            errors.append(f"callback accounting mismatch for {event_type}")
    discard_reasons = quality.get("combat_discarded_callback_reasons", {})
    if (
        not isinstance(discard_reasons, dict)
        or set(discard_reasons) - DISCARD_REASONS
        or any(not isinstance(count, int) or isinstance(count, bool) or count < 0 for count in discard_reasons.values())
        or sum(discard_reasons.values()) != discarded_total
        or quality.get("combat_discarded_callbacks") != discarded_total
    ):
        errors.append("discarded callback reasons do not reconcile")
    type_counts = Counter(event["event_type"] for event in events)
    relation_counts = Counter(event["relation"] for event in events)
    shot_results = Counter(
        (event.get("shot_result") or "unavailable")
        for event in events
        if event["event_type"] == "weapon_fire"
    )
    report = {
        "match_id": match_dir.name.removeprefix("match_"),
        "status": "pass" if not errors else "fail",
        "events": len(events),
        "event_counts": dict(sorted(type_counts.items())),
        "relation_counts": dict(sorted(relation_counts.items())),
        "shot_results": dict(sorted(shot_results.items())),
        "source_links": source_links,
        "replay_marker_sources": len(markers),
        "replay_shot_rows": replay_shot_rows,
        "replay_unique_fires": len(fire_seen),
        "participants": len(participant_ids),
        "stats_players": len(stats_by_id),
        "quality_status": quality.get("status"),
        "discarded_callback_reasons": dict(sorted(discard_reasons.items())),
        "errors": errors,
    }
    return report, errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("paths", nargs="+", type=Path)
    args = parser.parse_args()
    failed = False
    for path in args.paths:
        report, errors = audit_match(path.resolve())
        print(json.dumps(report, ensure_ascii=False, sort_keys=True))
        failed = failed or bool(errors)
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
