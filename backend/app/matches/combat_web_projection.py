from __future__ import annotations

import math
from bisect import bisect_left
from collections import defaultdict
from collections.abc import Mapping
from typing import Any

from .canonical_repository import CanonicalMatch, thaw_json


def project_combat(match: CanonicalMatch) -> dict[str, Any]:
    engagements = match.engagements()
    if any(
        isinstance(engagement.get("details"), Mapping) for engagement in engagements
    ):
        return _project_rich_combat(engagements)

    events = list(match.iter_combat_events())
    events_by_id = {str(event.get("event_id") or ""): event for event in events}
    engagement_events = [
        (engagement, _source_events(engagement, events_by_id))
        for engagement in engagements
    ]
    state_locations = _locate_engagement_states(match, engagement_events)
    participants = {
        str(player.get("player_id") or ""): player for player in match.participants()
    }
    grouped: dict[int, list[dict[str, Any]]] = defaultdict(list)
    opening_kills: set[int] = set()

    for index, (engagement, source_events) in enumerate(engagement_events, 1):
        if not source_events:
            continue
        round_number = int(engagement.get("round_number") or 0)
        duel = _project_engagement(
            engagement,
            source_events,
            participants,
            state_locations,
            index,
            round_number not in opening_kills,
        )
        if duel["outcome"] == "kill":
            opening_kills.add(round_number)
        grouped[round_number].append(duel)

    return {
        "rounds": [
            {"round": number, "duels": grouped[number]} for number in sorted(grouped)
        ]
    }


def _project_rich_combat(engagements: Any) -> dict[str, Any]:
    grouped: dict[int, list[dict[str, Any]]] = defaultdict(list)
    for engagement in engagements:
        details = engagement.get("details")
        if not isinstance(details, Mapping):
            continue
        grouped[int(engagement.get("round_number") or 0)].append(thaw_json(details))
    return {
        "rounds": [
            {"round": round_number, "duels": grouped[round_number]}
            for round_number in sorted(grouped)
        ]
    }


def reaction_averages(match: CanonicalMatch) -> dict[str, float]:
    samples: dict[str, list[float]] = defaultdict(list)
    for event in match.iter_combat_events():
        reaction = event.get("reaction_time_ms")
        player_id = str(event.get("attacker_player_id") or "").removeprefix("steam:")
        if not player_id or reaction is None or event.get("through_smoke"):
            continue
        try:
            value = float(reaction)
        except (TypeError, ValueError):
            continue
        if 50 <= value <= 2500:
            samples[player_id].append(value)
    return {
        player_id: round(sum(values) / len(values), 1)
        for player_id, values in samples.items()
        if values
    }


def _source_events(
    engagement: Mapping[str, Any],
    events_by_id: dict[str, dict[str, Any]],
) -> list[dict[str, Any]]:
    return [
        events_by_id[event_id]
        for event_id in engagement.get("source_event_ids", ())
        if event_id in events_by_id
    ]


def _project_engagement(
    engagement: Mapping[str, Any],
    events: list[dict[str, Any]],
    participants: dict[str, Mapping[str, Any]],
    states: dict[tuple[int, str, int], dict[str, Any]],
    index: int,
    can_be_opening: bool,
) -> dict[str, Any]:
    attacker_id = str(engagement.get("attacker_player_id") or "")
    victim_ids = [str(value) for value in engagement.get("victim_player_ids", ())]
    start_tick = int(engagement.get("start_tick") or events[0].get("tick") or 0)
    end_tick = int(engagement.get("end_tick") or events[-1].get("tick") or start_tick)
    round_number = int(engagement.get("round_number") or 0)
    attacker_events = [
        event for event in events if event.get("attacker_player_id") == attacker_id
    ]
    attacker = _project_participant(
        attacker_id,
        attacker_events,
        participants,
        states.get((round_number, attacker_id, start_tick), {}),
        is_attacker=True,
    )
    victims = [
        _project_participant(
            victim_id,
            [event for event in events if event.get("victim_player_id") == victim_id],
            participants,
            states.get((round_number, victim_id, start_tick), {}),
            is_attacker=False,
        )
        for victim_id in victim_ids
    ]
    outcome = str(engagement.get("outcome") or "damage")
    return {
        "duel_id": f"duel_{index}",
        "type": engagement.get("engagement_type", "duel"),
        "outcome": outcome,
        "victim_count": len(victims),
        "tick_start": start_tick,
        "tick_end": end_tick,
        "duration_ms": max(end_tick - start_tick, 0) * 1000 / 64,
        "attacker": attacker,
        "victims": victims,
        "exchanges": [_project_exchange(event, participants) for event in events],
        "context": {
            "distance": max(
                (float(event.get("distance_world_units") or 0) for event in events),
                default=0,
            ),
            "height_diff": _height_difference(events[0]),
            "is_trade": False,
            "through_smoke": any(bool(event.get("through_smoke")) for event in events),
            "is_wallbang": any(bool(event.get("is_wallbang")) for event in events),
            "penetrated_objects": 0,
            "bomb_planted": False,
            "alive_ct": 0,
            "alive_t": 0,
            "is_opening_kill": can_be_opening and outcome == "kill",
            "round_time_remaining": float(attacker.get("round_time_remaining") or 0),
        },
    }


def _project_participant(
    player_id: str,
    events: list[dict[str, Any]],
    participants: dict[str, Mapping[str, Any]],
    state: dict[str, Any],
    *,
    is_attacker: bool,
) -> dict[str, Any]:
    first = events[0] if events else {}
    identity = participants.get(player_id, {})
    position_key = "attacker_position" if is_attacker else "victim_position"
    side_key = "attacker_side" if is_attacker else "victim_side"
    damage = (
        sum(int(event.get("damage") or 0) for event in events) if is_attacker else 0
    )
    velocity = _horizontal_velocity(state)
    velocity_available = _horizontal_velocity_available(state)
    engagement_type = "unknown"
    if velocity_available:
        engagement_type = "peek" if velocity > 75 else "hold"
    was_killed = not is_attacker and any(bool(event.get("is_kill")) for event in events)
    return {
        "steam_id": player_id.removeprefix("steam:"),
        "name": identity.get("display_name", ""),
        "team": str(first.get(side_key) or state.get("side") or "").upper(),
        "map_area": state.get("area", ""),
        "position": first.get(position_key) or state.get("position") or {},
        "weapon": first.get("weapon", ""),
        "total_damage_dealt": damage,
        "hits": len(events) if is_attacker else 0,
        "headshots": sum(1 for event in events if event.get("is_headshot"))
        if is_attacker
        else 0,
        "health_after": 0 if was_killed else state.get("health", 0),
        "shots_fired": len(events) if is_attacker else 0,
        "time_to_reaction": _first_value(events, "reaction_time_ms"),
        "time_to_first_damage": _first_value(events, "time_to_damage_ms"),
        "initial_crosshair_error": 0,
        "velocity": velocity,
        "is_walking": bool(state.get("is_walking")),
        "is_ducking": bool(state.get("is_ducking")),
        "is_blind": False,
        "engagement_type": engagement_type,
        "round_time_remaining": float(
            state.get("phase_time_remaining_ms")
            if state.get("phase_time_remaining_ms") is not None
            else state.get("round_time_remaining_ms") or 0
        )
        / 1000,
    }


def _project_exchange(
    event: Mapping[str, Any],
    participants: dict[str, Mapping[str, Any]],
) -> dict[str, Any]:
    attacker = participants.get(str(event.get("attacker_player_id") or ""), {})
    return {
        "tick": event.get("tick", 0),
        "attacker": attacker.get("display_name", ""),
        "weapon": event.get("weapon", ""),
        "damage": event.get("damage", 0),
        "hitgroup": event.get("hitgroup", ""),
        "has_bullet_damage": True,
        "bullet_distance": event.get("distance_world_units", 0),
    }


def _first_value(events: list[dict[str, Any]], key: str) -> Any:
    return next((event[key] for event in events if event.get(key) is not None), None)


def _horizontal_velocity(state: Mapping[str, Any]) -> float:
    field = (
        "horizontal_velocity_world_units_per_second"
        if "horizontal_velocity_world_units_per_second" in state
        else "velocity_world_units_per_second"
    )
    try:
        velocity = float(state.get(field) or 0)
    except (TypeError, ValueError):
        return 0.0
    return velocity if math.isfinite(velocity) and velocity >= 0 else 0.0


def _horizontal_velocity_available(state: Mapping[str, Any]) -> bool:
    if "horizontal_velocity_world_units_per_second" in state:
        return state.get("velocity_source") in {
            "native",
            "position_delta",
        } and _is_valid_velocity(
            state.get("horizontal_velocity_world_units_per_second")
        )
    return _is_valid_velocity(state.get("velocity_world_units_per_second"))


def _is_valid_velocity(value: object) -> bool:
    try:
        velocity = float(value)
    except (TypeError, ValueError):
        return False
    return math.isfinite(velocity) and velocity >= 0


def _height_difference(event: Mapping[str, Any]) -> float:
    attacker = event.get("attacker_position") or {}
    victim = event.get("victim_position") or {}
    return float(attacker.get("z") or 0) - float(victim.get("z") or 0)


def _locate_engagement_states(
    match: CanonicalMatch,
    engagements: list[tuple[Mapping[str, Any], list[dict[str, Any]]]],
) -> dict[tuple[int, str, int], dict[str, Any]]:
    targets: dict[tuple[int, str], list[int]] = defaultdict(list)
    for engagement, events in engagements:
        if not events:
            continue
        round_number = int(engagement.get("round_number") or 0)
        tick = int(engagement.get("start_tick") or events[0].get("tick") or 0)
        player_ids = [
            engagement.get("attacker_player_id"),
            *engagement.get("victim_player_ids", ()),
        ]
        for player_id in player_ids:
            if player_id:
                targets[(round_number, str(player_id))].append(tick)
    targets = {key: sorted(set(values)) for key, values in targets.items()}

    closest: dict[tuple[int, str, int], tuple[int, dict[str, Any]]] = {}
    for state in match.iter_player_states():
        key = (int(state.get("round_number") or 0), str(state.get("player_id") or ""))
        target_ticks = targets.get(key)
        if not target_ticks:
            continue
        tick = int(state.get("tick") or 0)
        index = bisect_left(target_ticks, tick)
        for target_tick in target_ticks[max(index - 1, 0) : index + 1]:
            distance = abs(target_tick - tick)
            state_key = (*key, target_tick)
            previous = closest.get(state_key)
            if previous is None or distance < previous[0]:
                closest[state_key] = (distance, state)
    return {key: value[1] for key, value in closest.items()}
