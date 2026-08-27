from __future__ import annotations

from collections import defaultdict
from collections.abc import Mapping
from typing import Any

from .canonical_repository import CanonicalMatch, thaw_json


def project_match_metadata(match: CanonicalMatch) -> dict[str, Any]:
    data = match.match()
    return {
        "match_id": data.get("match_id", match.match_id),
        "map_name": data.get("map_name", ""),
        "final_score": f"{int(data.get('ct_score') or 0)}-{int(data.get('t_score') or 0)}",
        "winner": str(data.get("winner_side") or "").upper(),
        "date": data.get("played_at") or "",
        "duration_seconds": float(data.get("duration_ms") or 0) / 1000,
        "tick_rate": data.get("tick_rate_hz", 0),
        "total_rounds": data.get("round_count", 0),
    }


def project_player_summary(match: CanonicalMatch) -> dict[str, Any]:
    players = []
    for record in match.player_match_stats():
        metrics = record.get("metrics")
        if isinstance(metrics, Mapping):
            players.append(thaw_json(metrics))
    return {"match_id": match.match_id, "players": players}


def project_quality_report(match: CanonicalMatch) -> dict[str, Any]:
    return thaw_json(match.quality_report())


def project_economy(match: CanonicalMatch) -> list[dict[str, Any]]:
    records = match.economy_records()
    if any(isinstance(record.get("money"), Mapping) for record in records):
        return _project_block6_economy(match, records)
    if any(isinstance(record.get("details"), Mapping) for record in records):
        return _project_rich_economy(match, records)

    names = _participant_names(match)
    round_details = {
        int(item.get("round_number") or 0): item
        for item in match.rounds()
    }
    grouped: dict[int, list[dict[str, Any]]] = defaultdict(list)
    for record in records:
        player_id = str(record.get("player_id") or "")
        side = str(record.get("side") or "").upper()
        round_number = int(record.get("round_number") or 0)
        round_data = round_details.get(round_number, {})
        grouped[round_number].append(
            {
                "steam_id": player_id.removeprefix("steam:"),
                "name": names.get(player_id, ""),
                "team": side,
                "spawn_area": f"{side}Spawn" if side else "",
                "initial_money": record.get("initial_money", 0),
                "money_after_buy": record.get("money_after_buy", 0),
                "next_round_min_money": record.get("money_at_round_end", 0),
                "start_round_items": [],
                "equipment_value_start": record.get("equipment_value_start_calculated", 0),
                "equipment_value_start_calculated": record.get("equipment_value_start_calculated", 0),
                "spent_in_buy": record.get("spent_in_buy", 0),
                "purchases": [],
                "purchases_observed_value": 0,
                "purchases_vs_spent_delta": record.get("spent_in_buy", 0),
                "final_equipment": [],
                "final_equipment_value": record.get("equipment_value_freeze_end_calculated", 0),
                "final_equipment_value_calculated": record.get("equipment_value_freeze_end_calculated", 0),
                "final_money": record.get("money_at_round_end", 0),
                "money_at_round_end": record.get("money_at_round_end", 0),
                "equipment_value_end": record.get("equipment_value_round_end_calculated", 0),
                "equipment_value_end_calculated": record.get("equipment_value_round_end_calculated", 0),
                "end_equipment": [],
                "outcome": record.get("outcome", ""),
                "win_reason": _win_reason(round_data),
                "survived": bool(record.get("survived")),
            }
        )

    rounds = []
    for round_number in sorted(grouped):
        players = grouped[round_number]
        rounds.append(
            {
                "round": round_number,
                "teams": _team_economy(players),
                "players": players,
            }
        )
    return [{"match_id": match.match_id, "rounds": rounds}]


def project_utility(match: CanonicalMatch) -> dict[str, Any]:
    names = _participant_names(match)
    grouped: dict[int, list[dict[str, Any]]] = defaultdict(list)
    totals = {
        "grenades_thrown": 0,
        "smoke_thrown": 0,
        "flashbang_thrown": 0,
        "he_thrown": 0,
        "molotov_incendiary_thrown": 0,
        "decoy_thrown": 0,
    }
    for event in match.iter_utility_events():
        details = event.get("details")
        projected = thaw_json(details) if isinstance(details, Mapping) else None
        utility_type = str(event.get("utility_type") or "")
        type_name, total_key = _utility_names(utility_type)
        if projected is None:
            affected = event.get("affected_players") or []
            projected = {
                "type": type_name,
                "thrower": names.get(str(event.get("thrower_player_id") or ""), ""),
                "thrower_steam_id": str(event.get("thrower_player_id") or "").removeprefix("steam:"),
                "tick_throw": event.get("throw_tick", 0),
                "tick_explode": event.get("detonate_tick", 0),
                "thrower_area_name": event.get("thrower_area", ""),
                "thrower_side": str(event.get("thrower_side") or "").upper(),
                "land_area": event.get("land_area", ""),
                "start_position": event.get("start_position") or {},
                "end_position": event.get("end_position") or {},
                "did_bounce": bool(event.get("did_bounce")),
                "damage_dealt": sum(int(item.get("damage") or 0) for item in affected),
                "enemies_blinded": sum(1 for item in affected if item.get("is_enemy") and item.get("blind_ms")),
                "allies_blinded": sum(1 for item in affected if not item.get("is_enemy") and item.get("blind_ms")),
                "duration": float(event.get("duration_ms") or 0) / 1000,
                "extinguished": False,
                "kills": sum(1 for item in affected if item.get("is_kill")),
            }
        grouped[int(event.get("round_number") or 0)].append(projected)
        totals["grenades_thrown"] += 1
        totals[total_key] += 1

    rounds = []
    for round_number in sorted(grouped):
        events = grouped[round_number]
        counts = {key: 0 for key in totals if key != "grenades_thrown"}
        for event in events:
            _, total_key = _utility_names(str(event.get("type") or ""))
            if total_key in counts:
                counts[total_key] += 1
        rounds.append({"round": round_number, "grenades_thrown": len(events), **counts, "events": events})
    return {"totals": totals, "rounds": rounds}


def _project_rich_economy(
    match: CanonicalMatch,
    records: Any,
) -> list[dict[str, Any]]:
    players_by_round: dict[int, list[dict[str, Any]]] = defaultdict(list)
    for record in records:
        details = record.get("details")
        if isinstance(details, Mapping):
            players_by_round[int(record.get("round_number") or 0)].append(thaw_json(details))

    contexts = {
        int(context.get("round_number") or 0): context
        for context in match.economy_rounds()
    }
    rounds = []
    for round_number in sorted(players_by_round):
        context = contexts.get(round_number, {})
        rounds.append(
            {
                "round": round_number,
                "teams": thaw_json(context.get("teams", {})),
                "players": players_by_round[round_number],
                **(
                    {"events": thaw_json(context["events"])}
                    if context.get("events") is not None
                    else {}
                ),
            }
        )
    return [{"match_id": match.match_id, "rounds": rounds}]


def _project_block6_economy(
    match: CanonicalMatch,
    records: Any,
) -> list[dict[str, Any]]:
    names = _participant_names(match)
    round_details = {
        int(item.get("round_number") or 0): item
        for item in match.rounds()
    }
    grouped: dict[int, list[dict[str, Any]]] = defaultdict(list)
    economy_rounds: dict[tuple[int, str], Mapping[str, Any]] = {
        (int(item.get("round_number") or 0), str(item.get("team_id") or "")): item
        for item in match.economy_rounds()
    }
    for record in records:
        money = record.get("money") if isinstance(record.get("money"), Mapping) else {}
        start = record.get("inventory_start") if isinstance(record.get("inventory_start"), Mapping) else {}
        freeze = record.get("inventory_freeze_end") if isinstance(record.get("inventory_freeze_end"), Mapping) else {}
        end = record.get("inventory_round_end") if isinstance(record.get("inventory_round_end"), Mapping) else {}
        transactions = record.get("transactions") if isinstance(record.get("transactions"), tuple) else ()
        purchases = [
            _project_block6_item(transaction.get("item"))
            for transaction in transactions
            if isinstance(transaction, Mapping) and transaction.get("type") == "purchase"
        ]
        purchase_prices = [item.get("price") for item in purchases]
        purchases_observed_value = (
            sum(int(price) for price in purchase_prices)
            if all(isinstance(price, int) for price in purchase_prices)
            else None
        )
        spent_in_buy = _amount(record.get("spent_in_buy"))
        player_id = str(record.get("player_id") or "")
        side = str(record.get("side") or "").upper()
        round_number = int(record.get("round_number") or 0)
        grouped[round_number].append(
            {
                "steam_id": player_id.removeprefix("steam:"),
                "name": names.get(player_id, ""),
                "team_id": record.get("team_id"),
                "team": side,
                "spawn_area": f"{side}Spawn" if side else "",
                "initial_money": _amount(money.get("round_start_observed")),
                "money_after_buy": _first_amount(
                    money.get("after_buy_observed"),
                    money.get("after_buy_calculated"),
                ),
                "next_round_min_money": _amount(money.get("next_round_observed")),
                "start_round_items": [
                    _project_block6_item(item) for item in start.get("items", ())
                ],
                "equipment_value_start": start.get("native_value", 0),
                "equipment_value_start_calculated": start.get("calculated_value", 0),
                "spent_in_buy": spent_in_buy,
                "purchases": purchases,
                "purchases_observed_value": purchases_observed_value,
                "purchases_unpriced_count": sum(
                    not isinstance(price, int) for price in purchase_prices
                ),
                "purchases_vs_spent_delta": (
                    spent_in_buy - purchases_observed_value
                    if spent_in_buy is not None
                    and purchases_observed_value is not None
                    else None
                ),
                "final_equipment": [
                    _project_block6_item(item) for item in freeze.get("items", ())
                ],
                "final_equipment_value": freeze.get("native_value", 0),
                "final_equipment_value_calculated": freeze.get("calculated_value", 0),
                "final_money": _amount(money.get("round_end_observed")),
                "money_at_round_end": _amount(money.get("round_end_observed")),
                "equipment_value_end": _first_scalar(
                    end.get("native_value"), end.get("calculated_value")
                ),
                "equipment_value_end_native": end.get("native_value", 0),
                "equipment_value_end_calculated": end.get("calculated_value", 0),
                "end_equipment": [
                    _project_block6_item(item) for item in end.get("items", ())
                ],
                "outcome": record.get("outcome", ""),
                "win_reason": _win_reason(round_details.get(round_number, {})),
                "survived": bool(record.get("survived")),
                "warnings": thaw_json(record.get("warnings", ())),
            }
        )
    rounds = [
        {
            "round": round_number,
            "teams": _team_economy(
                grouped[round_number],
                {
                    team_id: value
                    for (candidate_round, team_id), value in economy_rounds.items()
                    if candidate_round == round_number
                },
            ),
            "players": grouped[round_number],
        }
        for round_number in sorted(grouped)
    ]
    return [{"match_id": match.match_id, "rounds": rounds}]


def _project_block6_item(value: Any) -> dict[str, Any]:
    item = value if isinstance(value, Mapping) else {}
    price = item.get("price") if isinstance(item.get("price"), Mapping) else {}
    return {
        "weapon": item.get("observed_item") or item.get("purchased_item") or "",
        "price": price.get("amount"),
        "price_status": price.get("status", "unknown"),
        "price_table_version": price.get("table_version", ""),
        "entity_id": item.get("entity_id"),
        "original_owner_steam_id": str(item.get("original_owner_player_id") or "").removeprefix("steam:"),
        "original_owner_status": item.get("original_owner_status", "unavailable"),
    }


def _amount(value: Any) -> int | None:
    if not isinstance(value, Mapping) or value.get("amount") is None:
        return None
    return int(value["amount"])


def _first_amount(*values: Any) -> int | None:
    for value in values:
        if isinstance(value, Mapping) and value.get("amount") is not None:
            return int(value["amount"])
    return None


def _first_scalar(*values: Any) -> int | None:
    return next((int(value) for value in values if isinstance(value, int)), None)


def _participant_names(match: CanonicalMatch) -> dict[str, str]:
    return {
        str(player.get("player_id") or ""): str(player.get("display_name") or "")
        for player in match.participants()
    }


def _win_reason(round_data: Mapping[str, Any]) -> str:
    side = str(round_data.get("winner_side") or "").upper()
    return f"{side}Win" if side else ""


def _team_economy(
    players: list[dict[str, Any]],
    economy_rounds: Mapping[str, Mapping[str, Any]] | None = None,
) -> dict[str, Any]:
    result = {}
    round_details = economy_rounds or {}
    for side in ("CT", "T"):
        team_players = [player for player in players if player.get("team") == side]
        money = [
            int(player["initial_money"])
            for player in team_players
            if isinstance(player.get("initial_money"), int)
        ]
        complete = len(money) == len(team_players) and bool(team_players)
        total = sum(money) if complete else None
        team_id = next(
            (player.get("team_id") for player in team_players if player.get("team_id")),
            "",
        )
        loss_bonus = round_details.get(str(team_id), {}).get("loss_bonus", {})
        result[side] = {
            "team_id": team_id,
            "total_money": total,
            "loss_bonus": _amount(loss_bonus),
            "money_observed_players": len(money),
            "money_unavailable_players": len(team_players) - len(money),
            "average_money": round(total / len(money)) if total is not None else None,
            "money_spread": max(money) - min(money) if complete else None,
            "gini_coefficient": _gini(money) if complete else None,
            "rounds_won": 0,
        }
    return result


def _gini(values: list[int]) -> float:
    positive = sorted(max(value, 0) for value in values)
    total = sum(positive)
    if not positive or total == 0:
        return 0.0
    weighted = sum((index + 1) * value for index, value in enumerate(positive))
    return (2 * weighted) / (len(positive) * total) - (len(positive) + 1) / len(positive)


def _utility_names(value: str) -> tuple[str, str]:
    normalized = value.lower()
    mapping = {
        "smoke_grenade": ("Smoke", "smoke_thrown"),
        "smoke": ("Smoke", "smoke_thrown"),
        "flashbang": ("Flash", "flashbang_thrown"),
        "flash": ("Flash", "flashbang_thrown"),
        "he_grenade": ("HE", "he_thrown"),
        "he": ("HE", "he_thrown"),
        "molotov": ("Molotov", "molotov_incendiary_thrown"),
        "incendiary": ("Incendiary", "molotov_incendiary_thrown"),
        "decoy": ("Decoy", "decoy_thrown"),
    }
    return mapping.get(normalized, (value, "grenades_thrown"))
