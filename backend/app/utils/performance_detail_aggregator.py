from __future__ import annotations

from collections import Counter, defaultdict
from typing import Any

from ..matches.canonical_repository import CanonicalMatch
from ..matches.combat_web_projection import project_combat
from ..matches.match_web_projection import project_economy

GRENADE_NAMES = {
    "Decoy Grenade",
    "Flashbang",
    "HE Grenade",
    "High Explosive Grenade",
    "Incendiary Grenade",
    "Molotov",
    "Smoke Grenade",
}


def _number(value: Any, default: float = 0.0) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return default


def _steam_id(value: Any) -> str:
    try:
        return str(int(value))
    except (TypeError, ValueError):
        return str(value or "")


def _average(total: float, count: int, precision: int = 1) -> float:
    return round(total / count, precision) if count else 0.0


def _participant_metrics(accumulator: dict[str, Any], participant: dict[str, Any]) -> None:
    accumulator["engagements"] += 1
    accumulator["shots"] += int(_number(participant.get("shots_fired")))
    accumulator["hits"] += int(_number(participant.get("hits")))

    reaction = _number(participant.get("time_to_reaction"))
    if 0 < reaction < 5000:
        accumulator["reaction_total"] += reaction
        accumulator["reaction_samples"] += 1

    first_damage = _number(participant.get("time_to_first_damage"))
    if 0 < first_damage < 5000:
        accumulator["first_damage_total"] += first_damage
        accumulator["first_damage_samples"] += 1

    crosshair = _number(participant.get("initial_crosshair_error"))
    if crosshair > 0:
        accumulator["crosshair_total"] += crosshair
        accumulator["crosshair_samples"] += 1

    velocity = _number(participant.get("velocity"))
    accumulator["moving"] += int(velocity > 75)
    accumulator["ducking"] += int(bool(participant.get("is_ducking")))
    accumulator["blind"] += int(bool(participant.get("is_blind")))
    engagement_type = str(participant.get("engagement_type", "")).lower()
    accumulator["holds"] += int(engagement_type == "hold")
    accumulator["peeks"] += int(engagement_type == "peek")


def _new_encounter(name: str, steam_id: str) -> dict[str, Any]:
    return {
        "steam_id": steam_id,
        "name": name or "Jugador",
        "wins": 0,
        "losses": 0,
        "damage_duels": 0,
        "user_shots": 0,
        "user_hits": 0,
        "rival_shots": 0,
        "rival_hits": 0,
        "user_first_damage_total": 0.0,
        "user_first_damage_samples": 0,
        "rival_first_damage_total": 0.0,
        "rival_first_damage_samples": 0,
        "through_smoke": 0,
        "wallbangs": 0,
        "trades": 0,
        "openings": 0,
        "user_blind": 0,
        "rival_blind": 0,
        "weapons": Counter(),
        "rival_weapons": Counter(),
        "areas": Counter(),
    }


def _record_encounter(
    encounter: dict[str, Any],
    user: dict[str, Any],
    rival: dict[str, Any],
    context: dict[str, Any],
    user_won: bool,
    outcome: str,
) -> None:
    if outcome == "kill":
        encounter["wins" if user_won else "losses"] += 1
    else:
        encounter["damage_duels"] += 1

    encounter["user_shots"] += int(_number(user.get("shots_fired")))
    encounter["user_hits"] += int(_number(user.get("hits")))
    encounter["rival_shots"] += int(_number(rival.get("shots_fired")))
    encounter["rival_hits"] += int(_number(rival.get("hits")))

    for prefix, participant in (("user", user), ("rival", rival)):
        first_damage = _number(participant.get("time_to_first_damage"))
        if 0 < first_damage < 5000:
            encounter[f"{prefix}_first_damage_total"] += first_damage
            encounter[f"{prefix}_first_damage_samples"] += 1

    encounter["through_smoke"] += int(bool(context.get("through_smoke")))
    encounter["wallbangs"] += int(bool(context.get("is_wallbang")))
    encounter["trades"] += int(bool(context.get("is_trade")))
    encounter["openings"] += int(bool(context.get("is_opening_kill")))
    encounter["user_blind"] += int(bool(user.get("is_blind")))
    encounter["rival_blind"] += int(bool(rival.get("is_blind")))
    encounter["weapons"][str(user.get("weapon") or "Sin arma")] += 1
    encounter["rival_weapons"][str(rival.get("weapon") or "Sin arma")] += 1
    encounter["areas"][str(user.get("map_area") or "Sin zona")] += 1


def _finalize_encounters(encounters: dict[str, dict[str, Any]]) -> list[dict[str, Any]]:
    result = []
    for encounter in encounters.values():
        total_kills = encounter["wins"] + encounter["losses"]
        result.append(
            {
                "steam_id": encounter["steam_id"],
                "name": encounter["name"],
                "wins": encounter["wins"],
                "losses": encounter["losses"],
                "damage_duels": encounter["damage_duels"],
                "win_rate": round(encounter["wins"] / total_kills * 100, 1) if total_kills else 0.0,
                "user_shots": encounter["user_shots"],
                "rival_shots": encounter["rival_shots"],
                "user_accuracy": round(encounter["user_hits"] / encounter["user_shots"] * 100, 1)
                if encounter["user_shots"]
                else 0.0,
                "rival_accuracy": round(encounter["rival_hits"] / encounter["rival_shots"] * 100, 1)
                if encounter["rival_shots"]
                else 0.0,
                "user_first_damage_ms": _average(
                    encounter["user_first_damage_total"],
                    encounter["user_first_damage_samples"],
                ),
                "rival_first_damage_ms": _average(
                    encounter["rival_first_damage_total"],
                    encounter["rival_first_damage_samples"],
                ),
                "user_weapon": encounter["weapons"].most_common(1)[0][0],
                "rival_weapon": encounter["rival_weapons"].most_common(1)[0][0],
                "area": encounter["areas"].most_common(1)[0][0],
                "through_smoke": encounter["through_smoke"],
                "wallbangs": encounter["wallbangs"],
                "trades": encounter["trades"],
                "openings": encounter["openings"],
                "user_blind": encounter["user_blind"],
                "rival_blind": encounter["rival_blind"],
            }
        )

    return sorted(
        result,
        key=lambda item: item["wins"] + item["losses"] + item["damage_duels"],
        reverse=True,
    )[:8]


def _aggregate_combat(
    matches: list[dict[str, Any]],
    steam_id: str,
) -> tuple[dict[str, Any], dict[str, Any], int]:
    metrics = defaultdict(float)
    encounters: dict[str, dict[str, Any]] = {}
    loaded_matches = 0

    for match in matches:
        combat = project_combat(CanonicalMatch(match["match_dir"]))
        if not combat:
            continue
        loaded_matches += 1

        for round_data in combat.get("rounds", []):
            for duel in round_data.get("duels", []):
                if duel.get("type") != "duel":
                    continue

                attacker = duel.get("attacker") or {}
                victims = duel.get("victims") or []
                context = duel.get("context") or {}
                attacker_is_user = _steam_id(attacker.get("steam_id")) == steam_id
                user_victim = next(
                    (victim for victim in victims if _steam_id(victim.get("steam_id")) == steam_id),
                    None,
                )
                if not attacker_is_user and not user_victim:
                    continue

                metrics["duels"] += 1
                metrics["smoke"] += int(bool(context.get("through_smoke")))
                metrics["wallbang"] += int(bool(context.get("is_wallbang")))
                user_participant = attacker if attacker_is_user else user_victim
                _participant_metrics(metrics, user_participant)

                rivals = victims if attacker_is_user else [attacker]
                for rival in rivals:
                    rival_id = _steam_id(rival.get("steam_id"))
                    if not rival_id or rival_id == steam_id:
                        continue
                    encounter = encounters.setdefault(
                        rival_id,
                        _new_encounter(str(rival.get("name") or "Jugador"), rival_id),
                    )
                    _record_encounter(
                        encounter,
                        user_participant,
                        rival,
                        context,
                        attacker_is_user,
                        str(duel.get("outcome", "")),
                    )

    engagements = int(metrics["engagements"])
    mechanics = {
        "engagements": engagements,
        "reaction_time_avg_ms": _average(metrics["reaction_total"], int(metrics["reaction_samples"])),
        "time_to_first_damage_avg_ms": _average(
            metrics["first_damage_total"],
            int(metrics["first_damage_samples"]),
        ),
        "crosshair_error_avg": _average(metrics["crosshair_total"], int(metrics["crosshair_samples"]), 2),
        "accuracy": round(metrics["hits"] / metrics["shots"] * 100, 1) if metrics["shots"] else 0.0,
        "shots": int(metrics["shots"]),
        "hits": int(metrics["hits"]),
        "stationary_pct": round((engagements - metrics["moving"]) / engagements * 100, 1)
        if engagements
        else 0.0,
        "moving_pct": round(metrics["moving"] / engagements * 100, 1) if engagements else 0.0,
        "ducking_pct": round(metrics["ducking"] / engagements * 100, 1) if engagements else 0.0,
        "blind_pct": round(metrics["blind"] / engagements * 100, 1) if engagements else 0.0,
        "through_smoke_pct": round(metrics["smoke"] / metrics["duels"] * 100, 1)
        if metrics["duels"]
        else 0.0,
        "wallbang_pct": round(metrics["wallbang"] / metrics["duels"] * 100, 1)
        if metrics["duels"]
        else 0.0,
        "hold_pct": round(metrics["holds"] / engagements * 100, 1) if engagements else 0.0,
        "peek_pct": round(metrics["peeks"] / engagements * 100, 1) if engagements else 0.0,
    }

    all_encounters = list(encounters.values())
    finalized_encounters = _finalize_encounters(encounters)
    duels = {
        "total": sum(item["wins"] + item["losses"] + item["damage_duels"] for item in all_encounters),
        "kills_won": sum(item["wins"] for item in all_encounters),
        "kills_lost": sum(item["losses"] for item in all_encounters),
        "encounters": finalized_encounters,
    }
    decisive = duels["kills_won"] + duels["kills_lost"]
    duels["win_rate"] = round(duels["kills_won"] / decisive * 100, 1) if decisive else 0.0
    return duels, mechanics, loaded_matches


def _classify_buy(equipment_value: int, spent: int) -> str:
    if equipment_value >= 4000:
        return "full_buy"
    if spent >= 2000:
        return "force_buy"
    if equipment_value >= 2000:
        return "partial_buy"
    return "eco"


def _aggregate_economy(
    matches: list[dict[str, Any]],
    steam_id: str,
) -> tuple[dict[str, Any], int]:
    totals = defaultdict(float)
    buy_types = {
        key: {"rounds": 0, "wins": 0}
        for key in ("full_buy", "partial_buy", "eco", "force_buy")
    }
    loaded_matches = 0

    for match in matches:
        payload = project_economy(CanonicalMatch(match["match_dir"]))
        if not payload:
            continue
        economy_matches = payload if isinstance(payload, list) else [payload]
        loaded_matches += 1

        for economy_match in economy_matches:
            rounds = economy_match.get("rounds", []) if isinstance(economy_match, dict) else []
            player_rounds = []
            for round_data in rounds:
                player = next(
                    (
                        item
                        for item in round_data.get("players", [])
                        if _steam_id(item.get("steam_id")) == steam_id
                    ),
                    None,
                )
                if player:
                    player_rounds.append((round_data, player))

            for index, (round_data, player) in enumerate(player_rounds):
                spent = int(_number(player.get("spent_in_buy")))
                equipment = int(_number(player.get("final_equipment_value")))
                survived = bool(player.get("survived"))
                won = str(player.get("outcome", "")).lower() == "win"
                buy_type = _classify_buy(equipment, spent)
                buy_types[buy_type]["rounds"] += 1
                buy_types[buy_type]["wins"] += int(won)

                totals["rounds"] += 1
                totals["spent"] += spent
                totals["equipment"] += equipment
                totals["final_money"] += int(_number(player.get("final_money")))
                totals["survived"] += int(survived)
                totals["saved_value"] += int(_number(player.get("equipment_value_end"))) if survived else 0

                team = str(player.get("team", "")).upper()
                team_data = (round_data.get("teams") or {}).get(team, {})
                totals["team_gini"] += _number(team_data.get("gini_coefficient"))
                totals["team_gini_samples"] += int(bool(team_data))

                if survived and index + 1 < len(player_rounds):
                    totals["save_opportunities"] += 1
                    next_player = player_rounds[index + 1][1]
                    totals["save_converted"] += int(str(next_player.get("outcome", "")).lower() == "win")

    rounds = int(totals["rounds"])
    return (
        {
            "rounds": rounds,
            "total_spent": int(totals["spent"]),
            "avg_spent_per_round": _average(totals["spent"], rounds),
            "avg_equipment_value": _average(totals["equipment"], rounds),
            "avg_money_after_round": _average(totals["final_money"], rounds),
            "saved_equipment_value": int(totals["saved_value"]),
            "survived_rounds": int(totals["survived"]),
            "survival_rate": round(totals["survived"] / rounds * 100, 1) if rounds else 0.0,
            "save_conversion_rate": round(
                totals["save_converted"] / totals["save_opportunities"] * 100,
                1,
            )
            if totals["save_opportunities"]
            else 0.0,
            "team_money_gini": _average(
                totals["team_gini"],
                int(totals["team_gini_samples"]),
                3,
            ),
            "buy_types": [
                {
                    "type": key,
                    "rounds": value["rounds"],
                    "share": round(value["rounds"] / rounds * 100, 1) if rounds else 0.0,
                    "win_rate": round(value["wins"] / value["rounds"] * 100, 1)
                    if value["rounds"]
                    else 0.0,
                }
                for key, value in buy_types.items()
            ],
        },
        loaded_matches,
    )


def aggregate_performance_details(
    matches: list[dict[str, Any]],
    steam_id: str,
) -> dict[str, Any]:
    normalized_steam_id = str(steam_id)
    duels, mechanics, combat_sources = _aggregate_combat(matches, normalized_steam_id)
    economy, economy_sources = _aggregate_economy(matches, normalized_steam_id)
    return {
        "duels": duels,
        "mechanics": mechanics,
        "economy": economy,
        "sources": {
            "combat_matches": combat_sources,
            "economy_matches": economy_sources,
        },
    }
