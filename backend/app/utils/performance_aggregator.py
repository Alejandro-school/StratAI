from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

EXPORTS_PATH = Path(__file__).resolve().parents[2] / "data" / "exports"


def _safe_int(value: Any, default: int = 0) -> int:
    try:
        if value is None:
            return default
        return int(value)
    except (TypeError, ValueError):
        return default


def _safe_float(value: Any, default: float = 0.0) -> float:
    try:
        if value is None:
            return default
        return float(value)
    except (TypeError, ValueError):
        return default


def _read_json(file_path: Path) -> dict[str, Any] | None:
    if not file_path.exists():
        return None

    try:
        with file_path.open("r", encoding="utf-8") as handle:
            data = json.load(handle)
            return data if isinstance(data, dict) else None
    except Exception as exc:
        logger.warning("[performance-aggregator] Could not read %s: %s", file_path, exc)
        return None


def _extract_total_rounds(metadata: dict[str, Any] | None) -> int:
    if not metadata:
        return 30

    direct = _safe_int(metadata.get("total_rounds"), 0)
    if direct > 0:
        return direct

    score_ct = _safe_int(metadata.get("score_ct"), 0)
    score_t = _safe_int(metadata.get("score_t"), 0)
    if score_ct + score_t > 0:
        return score_ct + score_t

    final_score = metadata.get("final_score")
    if isinstance(final_score, dict):
        ct = _safe_int(final_score.get("ct"), 0)
        t = _safe_int(final_score.get("t"), 0)
        if ct + t > 0:
            return ct + t

    return 30


def _iter_player_matches(steam_id: str) -> list[dict[str, Any]]:
    matches: list[dict[str, Any]] = []
    steam_id_str = str(steam_id)

    if not EXPORTS_PATH.exists():
        return matches

    for match_dir in EXPORTS_PATH.iterdir():
        if not match_dir.is_dir() or not match_dir.name.startswith("match_"):
            continue

        players_summary = _read_json(match_dir / "players_summary.json")
        if not players_summary:
            continue

        players = players_summary.get("players", [])
        if not isinstance(players, list):
            continue

        player = next(
            (entry for entry in players if str(entry.get("steam_id")) == steam_id_str),
            None,
        )
        if not isinstance(player, dict):
            continue

        metadata = _read_json(match_dir / "metadata.json") or {}
        total_rounds = _extract_total_rounds(metadata)
        match_id = players_summary.get("match_id") or match_dir.name
        matches.append(
            {
                "match_id": str(match_id),
                "metadata": metadata,
                "player": player,
                "total_rounds": total_rounds,
            }
        )

    return matches


def _weighted_average(total: float, weight: float) -> float:
    if weight <= 0:
        return 0.0
    return total / weight


def build_performance_overview(steam_id: str) -> dict[str, Any]:
    player_matches = _iter_player_matches(steam_id)
    if not player_matches:
        return {
            "steam_id": str(steam_id),
            "overview": {},
            "sides": {},
            "aim": {},
            "combat": {},
            "utility": {},
            "weapons": [],
            "maps": [],
            "match_history": [],
            "economy": {},
        }

    total_matches = len(player_matches)

    wins = 0
    losses = 0

    total_kills = 0
    total_deaths = 0
    total_assists = 0
    total_headshots = 0
    total_damage = 0
    total_utility_damage = 0

    total_shots_fired = 0
    total_shots_hit = 0

    total_rounds_weight = 0.0
    adr_weighted_sum = 0.0
    kast_weighted_sum = 0.0
    rating_weighted_sum = 0.0
    impact_weighted_sum = 0.0
    ct_rating_weighted_sum = 0.0
    t_rating_weighted_sum = 0.0
    ct_adr_weighted_sum = 0.0
    t_adr_weighted_sum = 0.0

    total_aim_weight = 0.0
    ttd_weighted_sum = 0.0
    crosshair_avg_weighted_sum = 0.0
    crosshair_peek_weighted_sum = 0.0
    crosshair_hold_weighted_sum = 0.0

    body_part_hits = {
        "head": 0,
        "chest": 0,
        "stomach": 0,
        "left_arm": 0,
        "right_arm": 0,
        "left_leg": 0,
        "right_leg": 0,
        "generic": 0,
    }

    opening_duels_attempted = 0
    opening_duels_won = 0
    opening_duels_lost = 0

    trade_kills = 0
    traded_deaths = 0
    flash_assists = 0

    clutches = {
        "1v1": 0,
        "1v2": 0,
        "1v3": 0,
        "1v4": 0,
        "1v5": 0,
    }
    multikills = {"2k": 0, "3k": 0, "4k": 0, "ace": 0}

    utility = {
        "grenades_thrown_total": 0,
        "flashes_thrown": 0,
        "enemies_flashed_total": 0,
        "flash_duration_total": 0.0,
        "he_thrown": 0,
        "molotovs_thrown": 0,
        "smokes_thrown": 0,
        "grenade_damage_he": 0,
        "grenade_damage_molotov": 0,
        "grenade_damage_flash": 0,
        "grenade_damage_smoke": 0,
    }

    rounds_survived = 0
    rounds_total = 0

    weapon_totals: dict[str, dict[str, Any]] = {}
    map_totals: dict[str, dict[str, Any]] = {}
    history: list[dict[str, Any]] = []

    for match in player_matches:
        player = match["player"]
        metadata = match["metadata"]
        total_rounds = _safe_int(match["total_rounds"], 30)
        match_id = str(match["match_id"])
        map_name = str(metadata.get("map_name", "unknown"))
        date = str(metadata.get("date", ""))

        team = str(player.get("team", ""))
        winner = str(metadata.get("winner", ""))
        is_win = winner and team and winner == team
        if is_win:
            wins += 1
            result = "W"
        else:
            losses += 1
            result = "L"

        kills = _safe_int(player.get("kills"), 0)
        deaths = _safe_int(player.get("deaths"), 0)
        assists = _safe_int(player.get("assists"), 0)
        headshots = _safe_int(player.get("headshots"), 0)

        total_kills += kills
        total_deaths += deaths
        total_assists += assists
        total_headshots += headshots
        total_damage += _safe_int(player.get("total_damage"), 0)
        total_utility_damage += _safe_int(player.get("utility_damage"), 0)

        shots_fired = _safe_int(player.get("shots_fired"), 0)
        shots_hit = _safe_int(player.get("shots_hit"), 0)
        total_shots_fired += shots_fired
        total_shots_hit += shots_hit

        total_rounds_weight += float(total_rounds)
        adr_weighted_sum += _safe_float(player.get("adr"), 0.0) * total_rounds
        kast_weighted_sum += _safe_float(player.get("kast"), 0.0) * total_rounds
        rating_weighted_sum += _safe_float(player.get("hltv_rating"), 0.0) * total_rounds
        impact_weighted_sum += _safe_float(player.get("impact_rating"), 0.0) * total_rounds
        ct_rating_weighted_sum += _safe_float(player.get("ct_rating"), 0.0) * total_rounds
        t_rating_weighted_sum += _safe_float(player.get("t_rating"), 0.0) * total_rounds
        ct_adr_weighted_sum += _safe_float(player.get("ct_adr"), 0.0) * total_rounds
        t_adr_weighted_sum += _safe_float(player.get("t_adr"), 0.0) * total_rounds

        aim_weight = float(max(shots_hit, 0))
        total_aim_weight += aim_weight
        ttd_weighted_sum += _safe_float(player.get("time_to_damage_avg_ms"), 0.0) * aim_weight
        crosshair_avg_weighted_sum += _safe_float(player.get("crosshair_placement_avg_error"), 0.0) * aim_weight
        crosshair_peek_weighted_sum += _safe_float(player.get("crosshair_placement_peek"), 0.0) * aim_weight
        crosshair_hold_weighted_sum += _safe_float(player.get("crosshair_placement_hold"), 0.0) * aim_weight

        per_match_body_parts = player.get("body_part_hits", {})
        if isinstance(per_match_body_parts, dict):
            for key in body_part_hits.keys():
                body_part_hits[key] += _safe_int(per_match_body_parts.get(key), 0)

        opening_duels_attempted += _safe_int(player.get("opening_duels_attempted"), 0)
        opening_duels_won += _safe_int(player.get("opening_duels_won"), 0)
        opening_duels_lost += _safe_int(player.get("opening_duels_lost"), 0)

        trade_kills += _safe_int(player.get("trade_kills"), 0)
        traded_deaths += _safe_int(player.get("traded_deaths"), 0)
        flash_assists += _safe_int(player.get("flash_assists"), 0)

        clutches["1v1"] += _safe_int(player.get("clutches_1v1_won"), 0)
        clutches["1v2"] += _safe_int(player.get("clutches_1v2_won"), 0)
        clutches["1v3"] += _safe_int(player.get("clutches_1v3_won"), 0)
        clutches["1v4"] += _safe_int(player.get("clutches_1v4_won"), 0)
        clutches["1v5"] += _safe_int(player.get("clutches_1v5_won"), 0)

        multi = player.get("multikills", {})
        if isinstance(multi, dict):
            multikills["2k"] += _safe_int(multi.get("2k"), 0)
            multikills["3k"] += _safe_int(multi.get("3k"), 0)
            multikills["4k"] += _safe_int(multi.get("4k"), 0)
            multikills["ace"] += _safe_int(multi.get("ace"), 0)

        utility["grenades_thrown_total"] += _safe_int(player.get("grenades_thrown_total"), 0)
        utility["flashes_thrown"] += _safe_int(player.get("flashes_thrown"), 0)
        utility["enemies_flashed_total"] += _safe_int(player.get("enemies_flashed_total"), 0)
        utility["flash_duration_total"] += _safe_float(player.get("flash_duration_total"), 0.0)
        utility["he_thrown"] += _safe_int(player.get("he_thrown"), 0)
        utility["molotovs_thrown"] += _safe_int(player.get("molotovs_thrown"), 0)
        utility["smokes_thrown"] += _safe_int(player.get("smokes_thrown"), 0)

        grenade_damage = player.get("grenade_damage", {})
        if isinstance(grenade_damage, dict):
            utility["grenade_damage_he"] += _safe_int(grenade_damage.get("he"), 0)
            utility["grenade_damage_molotov"] += _safe_int(grenade_damage.get("molotov"), 0)
            utility["grenade_damage_flash"] += _safe_int(grenade_damage.get("flash"), 0)
            utility["grenade_damage_smoke"] += _safe_int(grenade_damage.get("smoke"), 0)

        survived = _safe_int(player.get("rounds_survived"), 0)
        rounds_survived += survived
        rounds_total += total_rounds

        weapon_stats = player.get("weapon_stats", {})
        if isinstance(weapon_stats, dict):
            for weapon, stats in weapon_stats.items():
                if weapon not in weapon_totals:
                    weapon_totals[weapon] = {
                        "weapon": weapon,
                        "kills": 0,
                        "headshots": 0,
                        "damage": 0,
                        "shots_fired": 0,
                        "shots_hit": 0,
                    }
                aggregate = weapon_totals[weapon]
                aggregate["kills"] += _safe_int(stats.get("kills"), 0)
                aggregate["headshots"] += _safe_int(stats.get("headshots"), 0)
                aggregate["damage"] += _safe_int(stats.get("damage"), 0)
                aggregate["shots_fired"] += _safe_int(stats.get("shots_fired"), 0)
                aggregate["shots_hit"] += _safe_int(stats.get("shots_hit"), 0)

        if map_name not in map_totals:
            map_totals[map_name] = {
                "map": map_name,
                "wins": 0,
                "losses": 0,
                "kills": 0,
                "deaths": 0,
                "adr_weighted": 0.0,
                "rating_weighted": 0.0,
                "rounds": 0,
                "matches": 0,
            }

        map_data = map_totals[map_name]
        map_data["wins"] += 1 if is_win else 0
        map_data["losses"] += 0 if is_win else 1
        map_data["kills"] += kills
        map_data["deaths"] += deaths
        map_data["adr_weighted"] += _safe_float(player.get("adr"), 0.0) * total_rounds
        map_data["rating_weighted"] += _safe_float(player.get("hltv_rating"), 0.0) * total_rounds
        map_data["rounds"] += total_rounds
        map_data["matches"] += 1

        history.append(
            {
                "match_id": match_id,
                "date": date,
                "map": map_name,
                "result": result,
                "team_score": _safe_int(metadata.get("score_t"), 0)
                if team == "T"
                else _safe_int(metadata.get("score_ct"), 0),
                "opponent_score": _safe_int(metadata.get("score_ct"), 0)
                if team == "T"
                else _safe_int(metadata.get("score_t"), 0),
                "kills": kills,
                "deaths": deaths,
                "assists": assists,
                "kd_ratio": _safe_float(player.get("kd_ratio"), 0.0),
                "adr": _safe_float(player.get("adr"), 0.0),
                "hs_percentage": _safe_float(player.get("hs_percentage"), 0.0),
                "hltv_rating": _safe_float(player.get("hltv_rating"), 0.0),
                "accuracy_overall": (_safe_float(shots_hit) / shots_fired * 100) if shots_fired > 0 else 0.0,
            }
        )

    overall_kd = (total_kills / total_deaths) if total_deaths > 0 else float(total_kills)
    overall_hs_pct = (total_headshots / total_kills * 100) if total_kills > 0 else 0.0
    overall_accuracy = (total_shots_hit / total_shots_fired * 100) if total_shots_fired > 0 else 0.0
    opening_success_rate = (
        opening_duels_won / opening_duels_attempted * 100 if opening_duels_attempted > 0 else 0.0
    )
    win_rate = (wins / total_matches * 100) if total_matches > 0 else 0.0
    survival_rate = (rounds_survived / rounds_total * 100) if rounds_total > 0 else 0.0

    weapons = []
    for _, data in weapon_totals.items():
        fired = data["shots_fired"]
        kills = data["kills"]
        hit = data["shots_hit"]
        hs = data["headshots"]
        weapons.append(
            {
                "weapon": data["weapon"],
                "kills": kills,
                "headshots": hs,
                "damage": data["damage"],
                "shots_fired": fired,
                "shots_hit": hit,
                "accuracy": round((hit / fired * 100), 1) if fired > 0 else 0.0,
                "hs_pct": round((hs / kills * 100), 1) if kills > 0 else 0.0,
            }
        )
    weapons.sort(key=lambda item: item["kills"], reverse=True)

    maps = []
    for _, data in map_totals.items():
        map_total_matches = data["wins"] + data["losses"]
        win_rate_map = (data["wins"] / map_total_matches * 100) if map_total_matches > 0 else 0.0
        kd_map = (data["kills"] / data["deaths"]) if data["deaths"] > 0 else float(data["kills"])
        avg_adr_map = _weighted_average(data["adr_weighted"], float(data["rounds"]))
        avg_rating_map = _weighted_average(data["rating_weighted"], float(data["rounds"]))
        maps.append(
            {
                "map": data["map"],
                "wins": data["wins"],
                "losses": data["losses"],
                "win_rate": round(win_rate_map, 1),
                "avg_kd": round(kd_map, 2),
                "avg_adr": round(avg_adr_map, 1),
                "avg_rating": round(avg_rating_map, 2),
                "matches": data["matches"],
            }
        )
    maps.sort(key=lambda item: item["matches"], reverse=True)

    history.sort(key=lambda item: item.get("date", ""), reverse=True)

    flashes_thrown = utility["flashes_thrown"]
    he_thrown = utility["he_thrown"]
    molotovs_thrown = utility["molotovs_thrown"]

    response = {
        "steam_id": str(steam_id),
        "overview": {
            "total_matches": total_matches,
            "wins": wins,
            "losses": losses,
            "win_rate": round(win_rate, 1),
            "kills": total_kills,
            "deaths": total_deaths,
            "assists": total_assists,
            "total_damage": total_damage,
            "kd_ratio": round(overall_kd, 2),
            "adr": round(_weighted_average(adr_weighted_sum, total_rounds_weight), 1),
            "hs_pct": round(overall_hs_pct, 1),
            "kast": round(_weighted_average(kast_weighted_sum, total_rounds_weight), 1),
            "hltv_rating": round(_weighted_average(rating_weighted_sum, total_rounds_weight), 2),
            "impact_rating": round(_weighted_average(impact_weighted_sum, total_rounds_weight), 2),
        },
        "sides": {
            "ct_rating": round(_weighted_average(ct_rating_weighted_sum, total_rounds_weight), 2),
            "t_rating": round(_weighted_average(t_rating_weighted_sum, total_rounds_weight), 2),
            "ct_adr": round(_weighted_average(ct_adr_weighted_sum, total_rounds_weight), 1),
            "t_adr": round(_weighted_average(t_adr_weighted_sum, total_rounds_weight), 1),
        },
        "aim": {
            "accuracy_overall": round(overall_accuracy, 1),
            "time_to_damage_avg_ms": round(_weighted_average(ttd_weighted_sum, total_aim_weight), 1),
            "crosshair_placement_avg_error": round(_weighted_average(crosshair_avg_weighted_sum, total_aim_weight), 2),
            "crosshair_placement_peek": round(_weighted_average(crosshair_peek_weighted_sum, total_aim_weight), 2),
            "crosshair_placement_hold": round(_weighted_average(crosshair_hold_weighted_sum, total_aim_weight), 2),
            "shots_fired": total_shots_fired,
            "shots_hit": total_shots_hit,
            "body_part_hits": body_part_hits,
        },
        "combat": {
            "opening_duels_attempted": opening_duels_attempted,
            "opening_duels_won": opening_duels_won,
            "opening_duels_lost": opening_duels_lost,
            "opening_success_rate": round(opening_success_rate, 1),
            "trade_kills": trade_kills,
            "traded_deaths": traded_deaths,
            "flash_assists": flash_assists,
            "clutches": clutches,
            "multikills": multikills,
        },
        "utility": {
            "grenades_thrown_total": utility["grenades_thrown_total"],
            "flashes_thrown": flashes_thrown,
            "enemies_flashed_total": utility["enemies_flashed_total"],
            "flash_duration_total": round(utility["flash_duration_total"], 2),
            "enemies_flashed_per_flash": round((utility["enemies_flashed_total"] / flashes_thrown), 2)
            if flashes_thrown > 0
            else 0.0,
            "blind_time_per_flash": round((utility["flash_duration_total"] / flashes_thrown), 2)
            if flashes_thrown > 0
            else 0.0,
            "he_thrown": he_thrown,
            "he_damage_per_nade": round((utility["grenade_damage_he"] / he_thrown), 2)
            if he_thrown > 0
            else 0.0,
            "molotovs_thrown": molotovs_thrown,
            "molotov_damage_per_nade": round((utility["grenade_damage_molotov"] / molotovs_thrown), 2)
            if molotovs_thrown > 0
            else 0.0,
            "smokes_thrown": utility["smokes_thrown"],
            "utility_damage": total_utility_damage,
            "grenade_damage": {
                "he": utility["grenade_damage_he"],
                "molotov": utility["grenade_damage_molotov"],
                "flash": utility["grenade_damage_flash"],
                "smoke": utility["grenade_damage_smoke"],
            },
        },
        "weapons": weapons,
        "maps": maps,
        "match_history": history[:20],
        "economy": {
            "rounds_survived": rounds_survived,
            "total_rounds": rounds_total,
            "survival_rate": round(survival_rate, 1),
        },
    }

    return response
