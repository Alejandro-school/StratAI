
# backend/app/routes/dashboard.py
# -------------------------------
# Rutas para el dashboard del frontend (stats, mapas, granadas, etc.)
#
# OPTIMIZATION (2026-01-08): 
# This file now uses pre-calculated user aggregate files for O(1) lookups
# instead of O(n) folder scanning. See utils/user_aggregates.py for details.

import asyncio
import json
import logging
import os
from collections.abc import Mapping
from typing import Any

import httpx
import redis.asyncio as aioredis
from fastapi import APIRouter, Depends, HTTPException, Request
from fastapi.responses import StreamingResponse

from ..auth.dependencies import SteamUser, require_steam_user
from ..config import PIPELINE_NAMESPACE, REDIS_URL, STEAM_API_KEY
from ..matches.canonical_repository import (
    CanonicalMatch,
    find_canonical_match,
    iter_canonical_matches,
    thaw_json,
)
from ..matches.combat_web_projection import project_combat, reaction_averages
from ..matches.match_web_projection import (
    project_economy,
    project_match_metadata,
    project_player_summary,
    project_utility,
)
from ..matches.player_match_catalog import (
    PlayerMatch,
    clear_player_match_cache,
    list_player_matches,
)
from ..middleware.rate_limit import get_rate_limiter
from ..middleware.security import sanitize_match_id
from ..tactical.movement_contract import normalize_movement_contract

# Import utils
from ..utils.maps import (
    CALLOUT_FIXED_POSITIONS,
    game_to_radar_percent,
    normalize_callout,
)
from ..utils.user_aggregates import (
    EXPORTS_PATH,
    is_user_map_data_fresh,
    load_user_map_data,
    user_has_map_data,
)

router = APIRouter()

# Redis logic (same as other files for now)
redis = aioredis.from_url(REDIS_URL, decode_responses=True)

rate_limiter = get_rate_limiter(REDIS_URL)

NUKE_Z_THRESHOLD = -500


def resolve_radar_map_name(base_map_name: str, avg_z: float | None) -> str:
    if base_map_name == "de_nuke" and avg_z is not None and avg_z < NUKE_Z_THRESHOLD:
        return "de_nuke_lower"
    return base_map_name


def resolve_level_name(map_name: str, avg_z: float | None) -> str | None:
    if map_name != "de_nuke" or avg_z is None:
        return None
    return "upper" if avg_z >= NUKE_Z_THRESHOLD else "lower"


def infer_nuke_level_from_callout(callout: str) -> str:
    lower_positions = CALLOUT_FIXED_POSITIONS.get("de_nuke_lower", {})
    return "lower" if callout in lower_positions else "upper"


def resolve_fixed_callout_position(map_name: str, callout: str, level: str | None):
    map_key = "de_nuke_lower" if map_name == "de_nuke" and level == "lower" else map_name
    return CALLOUT_FIXED_POSITIONS.get(map_key, {}).get(callout)


def infer_nuke_level_from_grenade_areas(areas: list[str]) -> str:
    if not areas:
        return "upper"

    lower_keywords = {
        "b site", "b-site", "bsite", "b",
        "toxic", "decon", "decontamination", "dark", "control",
        "ramp", "ramp room", "vents", "vent", "secret",
    }

    for area in areas:
        area_lower = (area or "").strip().lower()
        if not area_lower:
            continue
        if area_lower in lower_keywords or any(keyword in area_lower for keyword in lower_keywords):
            return "lower"

    return "upper"


def resolve_cluster_level(map_name: str, avg_z: float | None, areas: list[str]) -> str | None:
    level = resolve_level_name(map_name, avg_z)
    if map_name == "de_nuke" and level is None:
        level = infer_nuke_level_from_grenade_areas(areas)
    return level


def resolve_radar_map_name_by_level(map_name: str, avg_z: float | None, level: str | None) -> str:
    if map_name == "de_nuke" and level == "lower":
        return "de_nuke_lower"
    if map_name == "de_nuke" and level == "upper":
        return "de_nuke"
    return resolve_radar_map_name(map_name, avg_z)


def normalize_grenade_area(raw_area: str, map_name: str) -> str:
    normalized = normalize_callout(raw_area, map_name)
    if normalized:
        return normalized
    return raw_area or ""


def load_match_reaction_averages(match_folder: str) -> dict[str, float]:
    try:
        return reaction_averages(CanonicalMatch(match_folder))
    except Exception as exc:
        logging.warning("[get-match-details] No se pudo leer combate canÃ³nico: %s", exc)
        return {}


def _load_match_projection(match_folder: str, content: str, fallback: Any) -> Any:
    match = CanonicalMatch(match_folder)
    projections = {
        "match": project_match_metadata,
        "players": project_player_summary,
        "combat": project_combat,
        "economy": project_economy,
        "utility": project_utility,
    }
    projector = projections.get(content)
    if projector is None or not match.has_canonical_bundle():
        return fallback
    try:
        return projector(match)
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        logging.warning("[match-projection] No se pudo construir %s: %s", content, exc)
        return fallback


def _round_map(payload: dict[str, Any]) -> dict[int, dict[str, Any]]:
    result: dict[int, dict[str, Any]] = {}
    for item in payload.get("rounds", []):
        try:
            round_number = int(item.get("round"))
        except (TypeError, ValueError):
            continue
        result[round_number] = item
    return result


def _economy_rounds(payload: Any) -> list[dict[str, Any]]:
    if isinstance(payload, list):
        payload = payload[0] if payload else {}
    return payload.get("rounds", []) if isinstance(payload, dict) else []


def _steam_ids_match(left: Any, right: Any) -> bool:
    """Tolerate legacy exports that serialized 64-bit Steam IDs as JSON floats."""
    if str(left) == str(right):
        return True
    try:
        return abs(int(left) - int(right)) <= 128
    except (TypeError, ValueError):
        return False


def _summarize_match_rounds(
    match_folder: str,
    steam_id: str,
) -> tuple[list[dict[str, Any]], dict[str, Any]]:
    economy_data = _load_match_projection(match_folder, "economy", {})
    combat_data = _load_match_projection(match_folder, "combat", {"rounds": []})
    grenades_data = _load_match_projection(match_folder, "utility", {"rounds": []})

    combat_by_round = _round_map(combat_data)
    grenades_by_round = _round_map(grenades_data)
    summaries: list[dict[str, Any]] = []

    for economy_round in _economy_rounds(economy_data):
        try:
            round_number = int(economy_round.get("round"))
        except (TypeError, ValueError):
            continue

        players = economy_round.get("players") or []
        teams = economy_round.get("teams") or {}
        user_player = next(
            (
                player
                for player in players
                if _steam_ids_match(player.get("steam_id"), steam_id)
            ),
            None,
        )
        winning_player = next(
            (
                player
                for player in players
                if str(player.get("outcome", "")).lower() in {"win", "won", "victory"}
            ),
            None,
        )
        winner = (winning_player or {}).get("team", "")
        win_reason = (winning_player or {}).get("win_reason", "")

        combat_round = combat_by_round.get(round_number, {})
        combat_duels = combat_round.get("duels", [])
        kills = [
            duel
            for duel in combat_duels
            if str(duel.get("outcome", "")).lower() in {"kill", "multi_kill"}
        ]
        kills.sort(key=lambda duel: int(duel.get("tick_start") or 0))
        opening_duel = kills[0] if kills else {}
        attacker = opening_duel.get("attacker") or {}
        victims = opening_duel.get("victims") or []
        first_victim = victims[0] if victims else {}

        user_name = str((user_player or {}).get("name") or "")
        user_kills = 0
        user_deaths = 0
        user_damage = 0
        user_headshots = 0
        user_trade_kills = 0
        user_weapons: list[str] = []
        for duel in combat_duels:
            outcome = str(duel.get("outcome", "")).lower()
            duel_attacker = duel.get("attacker") or {}
            duel_victims = duel.get("victims") or []
            if _steam_ids_match(duel_attacker.get("steam_id"), steam_id):
                user_damage += int(duel_attacker.get("total_damage_dealt") or 0)
                if outcome in {"kill", "multi_kill"}:
                    user_kills += int(duel.get("victim_count") or len(duel_victims) or 1)
                    user_headshots += int(duel_attacker.get("headshots") or 0)
                    if (duel.get("context") or {}).get("is_trade"):
                        user_trade_kills += 1
                    weapon = duel_attacker.get("weapon")
                    if weapon and weapon not in user_weapons:
                        user_weapons.append(weapon)
            if outcome in {"kill", "multi_kill"} and any(
                _steam_ids_match(victim.get("steam_id"), steam_id)
                for victim in duel_victims
            ):
                user_deaths = 1

        grenade_events = grenades_by_round.get(round_number, {}).get("events", [])
        user_grenade_events = [
            event
            for event in grenade_events
            if user_name and str(event.get("thrower") or "") == user_name
        ]
        utility_damage = sum(int(event.get("damage_dealt") or 0) for event in grenade_events)
        user_utility_damage = sum(
            int(event.get("damage_dealt") or 0)
            for event in user_grenade_events
        )

        team_spend: dict[str, int] = {}
        team_equipment: dict[str, int] = {}
        survivors: dict[str, int] = {}
        for side in ("CT", "T"):
            side_players = [player for player in players if player.get("team") == side]
            team_spend[side] = sum(int(player.get("spent_in_buy") or 0) for player in side_players)
            team_equipment[side] = sum(
                int(player.get("final_equipment_value") or player.get("equipment_value_start") or 0)
                for player in side_players
            )
            survivors[side] = sum(1 for player in side_players if player.get("survived"))

        summaries.append(
            {
                "round": round_number,
                "winner": winner,
                "win_reason": win_reason,
                "user_outcome": (user_player or {}).get("outcome", ""),
                "teams": teams,
                "team_spend": team_spend,
                "team_equipment": team_equipment,
                "survivors": survivors,
                "opening": {
                    "tick": opening_duel.get("tick_start"),
                    "killer": attacker.get("name"),
                    "killer_steam_id": attacker.get("steam_id"),
                    "killer_team": attacker.get("team"),
                    "victim": first_victim.get("name"),
                    "victim_steam_id": first_victim.get("steam_id"),
                    "weapon": attacker.get("weapon"),
                    "is_trade": bool((opening_duel.get("context") or {}).get("is_trade")),
                } if opening_duel else None,
                "utility": {
                    "count": len(grenade_events),
                    "damage": utility_damage,
                },
                "user_combat": {
                    "kills": user_kills,
                    "deaths": user_deaths,
                    "damage": user_damage,
                    "headshots": user_headshots,
                    "trade_kills": user_trade_kills,
                    "weapons": user_weapons,
                },
                "user_utility": {
                    "count": len(user_grenade_events),
                    "damage": user_utility_damage,
                    "types": [event.get("type") for event in user_grenade_events],
                },
                "user_economy": {
                    "team": (user_player or {}).get("team"),
                    "initial_money": (user_player or {}).get("initial_money", 0),
                    "spent": (user_player or {}).get("spent_in_buy", 0),
                    "final_money": (user_player or {}).get("final_money", 0),
                    "equipment_value": (user_player or {}).get("final_equipment_value", 0),
                    "survived": bool((user_player or {}).get("survived")),
                    "purchases": (user_player or {}).get("purchases", []),
                },
            }
        )

    summaries.sort(key=lambda item: item["round"])
    total_user_spend = sum(int(item["user_economy"]["spent"] or 0) for item in summaries)
    full_buy_rounds = sum(
        1
        for item in summaries
        if int(item["user_economy"]["equipment_value"] or 0) >= 4000
    )
    low_buy_rounds = sum(
        1
        for item in summaries
        if int(item["user_economy"]["equipment_value"] or 0) < 2000
    )
    economy_summary = {
        "rounds_available": len(summaries),
        "total_user_spend": total_user_spend,
        "average_user_spend": round(total_user_spend / len(summaries)) if summaries else 0,
        "full_buy_rounds": full_buy_rounds,
        "low_buy_rounds": low_buy_rounds,
    }
    return summaries, economy_summary


def _profile_from_player(player: dict[str, Any]) -> dict[str, Any]:
    rating = (
        player.get("premier_rating")
        or player.get("cs_rating")
        or player.get("rating_premier")
    )
    return {
        "avatar": player.get("avatar") or player.get("avatar_url"),
        "premier_rating": rating,
        "rank_type": player.get("rank_type") or player.get("rank"),
        "status": "available" if rating is not None else "unavailable",
        "source": "match",
    }


def _extract_premier_profile(result: Any) -> dict[str, Any]:
    if not isinstance(result, dict):
        return {"premier_rating": None, "rank_type": None, "status": "unavailable"}
    rating: Any = None
    for key in ("premier_rating", "cs_rating", "rating", "ranking"):
        candidate = result.get(key)
        if isinstance(candidate, dict):
            candidate = candidate.get("value") or candidate.get("rating")
        if candidate not in (None, ""):
            rating = candidate
            break
    try:
        rating = int(float(rating)) if rating is not None else None
    except (TypeError, ValueError):
        rating = None
    return {
        "premier_rating": rating,
        "rank_type": result.get("rank_type"),
        "status": "available" if rating is not None or result.get("rank_type") is not None else "unavailable",
    }


async def _load_player_profiles(players: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    profiles = {
        str(player.get("steam_id")): _profile_from_player(player)
        for player in players
        if player.get("steam_id")
    }
    steam_ids = list(profiles)
    if not steam_ids or not STEAM_API_KEY:
        return profiles

    cached_profiles: dict[str, dict[str, Any]] = {}
    missing: list[str] = []
    try:
        cached_values = await redis.mget(
            [f"{PIPELINE_NAMESPACE}:steam-profile:{steam_id}" for steam_id in steam_ids]
        )
        for steam_id, raw in zip(steam_ids, cached_values):
            if raw:
                cached_profiles[steam_id] = json.loads(raw)
            else:
                missing.append(steam_id)
    except Exception as exc:
        logging.warning("[get-match-details] Caché de perfiles no disponible: %s", exc)
        missing = steam_ids

    profiles.update(cached_profiles)
    if not missing:
        return profiles

    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            avatar_request = client.get(
                "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/",
                params={"key": STEAM_API_KEY, "steamids": ",".join(missing)},
            )
            rank_requests = [
                client.get(
                    "https://api.steampowered.com/ICSGOPlayers_730/GetGamePersonalData/v1/",
                    params={"key": STEAM_API_KEY, "steamid": steam_id},
                )
                for steam_id in missing
            ]
            responses = await asyncio.gather(
                avatar_request,
                *rank_requests,
                return_exceptions=True,
            )

        avatar_by_id: dict[str, str] = {}
        avatar_response = responses[0]
        if isinstance(avatar_response, httpx.Response) and avatar_response.is_success:
            avatar_by_id = {
                str(item.get("steamid")): item.get("avatarfull")
                for item in avatar_response.json().get("response", {}).get("players", [])
            }

        for steam_id, response in zip(missing, responses[1:]):
            profile = {**profiles.get(steam_id, {})}
            profile["avatar"] = avatar_by_id.get(steam_id) or profile.get("avatar")
            if isinstance(response, httpx.Response) and response.is_success:
                profile.update(_extract_premier_profile(response.json().get("result", {})))
                profile["source"] = "steam"
            profiles[steam_id] = profile
    except (httpx.HTTPError, ValueError, KeyError) as exc:
        logging.warning("[get-match-details] No se pudieron enriquecer perfiles Steam: %s", exc)

    try:
        pipeline = redis.pipeline()
        for steam_id in missing:
            pipeline.setex(
                f"{PIPELINE_NAMESPACE}:steam-profile:{steam_id}",
                21600,
                json.dumps(profiles.get(steam_id, {})),
            )
        await pipeline.execute()
    except Exception as exc:
        logging.warning("[get-match-details] No se pudo guardar la caché de perfiles: %s", exc)

    return profiles

@router.get("/steam/get-processed-demos")
@rate_limiter.limit(30, 60)  # 30 requests per minute per IP
async def get_processed_demos(
    request: Request,
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    matches = await asyncio.to_thread(
        list_player_matches,
        user.steam_id,
        EXPORTS_PATH,
    )
    return {"matches": [_project_match_list_item(match) for match in matches]}


def _project_match_list_item(match: PlayerMatch) -> dict[str, Any]:
    player = match.player
    return {
        "match_id": match.match_id,
        "map_name": match.metadata.get("map_name", "unknown"),
        "match_date": match.metadata.get("date", ""),
        "match_duration": match.metadata.get("duration_seconds", 0),
        "result": "victory" if match.result == "W" else "defeat",
        "team_score": match.team_score,
        "opponent_score": match.opponent_score,
        "total_rounds": match.total_rounds,
        "user_team": match.user_team,
        "players": [
            {
                "steam_id": player.get("steam_id"),
                "name": player.get("name", ""),
                "kills": player.get("kills", 0),
                "deaths": player.get("deaths", 0),
                "assists": player.get("assists", 0),
                "kd_ratio": player.get("kd_ratio", 0),
                "adr": player.get("adr", 0),
                "hs_percentage": player.get("hs_percentage", 0),
                "hltv_rating": player.get("hltv_rating", 0),
            }
        ],
    }


# ============================================================================
# MATCH DETAILS (Single match)
# ============================================================================
@router.get("/steam/get-match-details/{match_id}")
@rate_limiter.limit(60, 60)  # 60 requests per minute per IP
async def get_match_details(
    request: Request,
    match_id: str,
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    """
    Endpoint para obtener todos los detalles de una partida específica.
    Conserva el DTO pÃºblico a partir del bundle canÃ³nico.
    """
    steam_id_str = user.steam_id
    
    # Sanitize match_id to prevent path traversal
    try:
        match_id = sanitize_match_id(match_id)
    except ValueError:
        raise HTTPException(status_code=400, detail="match_id inválido.")
    
    logging.info(f"[get-match-details] Buscando detalles de {match_id} para {steam_id_str}")

    match_export = find_canonical_match(EXPORTS_PATH, match_id)
    if match_export is None:
        logging.warning(f"[get-match-details] Carpeta no encontrada: {match_id}")
        raise HTTPException(status_code=404, detail="Partida no encontrada.")
    match_folder = str(match_export.directory)

    try:
        metadata = project_match_metadata(match_export)
        players_data = project_player_summary(match_export)

        # Ownership validation: verify the authenticated user is a player in this match
        players = players_data.get("players", [])
        user_in_match = any(
            str(p.get("steam_id", "")) == steam_id_str for p in players
        )
        if not user_in_match:
            logging.warning(f"[get-match-details] User {steam_id_str} not in match {match_id}")
            raise HTTPException(status_code=403, detail="No tienes acceso a esta partida.")

        reaction_averages = load_match_reaction_averages(match_folder)
        
        # Separar jugadores por equipo
        players = players_data.get("players", [])
        for player in players:
            player_steam_id = str(player.get("steam_id", ""))
            if player_steam_id in reaction_averages:
                player["avg_time_to_reaction"] = reaction_averages[player_steam_id]

        team_ct = [p for p in players if p.get("team") == "CT"]
        team_t = [p for p in players if p.get("team") == "T"]
        
        # Ordenar por HLTV rating
        team_ct.sort(key=lambda x: x.get("hltv_rating", 0), reverse=True)
        team_t.sort(key=lambda x: x.get("hltv_rating", 0), reverse=True)
        
        # Encontrar al usuario actual
        current_user_stats = next(
            (p for p in players if str(p.get("steam_id", "")) == steam_id_str),
            None
        )
        
        # Calcular score del resultado (metadata score es siempre CT-T)
        final_score = metadata.get("final_score", "0-0")
        scores = final_score.split("-") if "-" in final_score else ["0", "0"]
        ct_score = int(scores[0].strip()) if scores[0].strip().isdigit() else 0
        t_score = int(scores[1].strip()) if len(scores) > 1 and scores[1].strip().isdigit() else 0
        
        # Determinar resultado para el usuario usando el campo winner
        winner = metadata.get("winner", "")
        user_team = current_user_stats.get("team") if current_user_stats else None
        is_victory = user_team == winner
        
        # Scores desde la perspectiva del usuario
        if user_team == "CT":
            user_team_score = ct_score
            opponent_team_score = t_score
        else:
            user_team_score = t_score
            opponent_team_score = ct_score

        round_summaries, economy_summary = _summarize_match_rounds(
            match_folder,
            steam_id_str,
        )
        player_profiles = await _load_player_profiles(players)
        
        return {
            "match_id": match_id,
            "metadata": {
                "map_name": metadata.get("map_name", "unknown"),
                "final_score": final_score,
                "team_score": user_team_score,
                "opponent_score": opponent_team_score,
                "ct_score": ct_score,
                "t_score": t_score,
                "winner": winner,
                "date": metadata.get("date", ""),
                "duration_seconds": metadata.get("duration_seconds", 0),
                "total_rounds": metadata.get("total_rounds", 0),
                "tick_rate": metadata.get("tick_rate", 64)
            },
            "result": "victory" if is_victory else "defeat",
            "team_ct": team_ct,
            "team_t": team_t,
            "current_user": current_user_stats,
            "current_user_steam_id": steam_id_str,
            "player_profiles": player_profiles,
            "rounds": round_summaries,
            "economy_summary": economy_summary,
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logging.error(f"[get-match-details] Error: {e}")
        raise HTTPException(status_code=500, detail="Error interno al leer datos de la partida.")


# ============================================================================
# DASHBOARD STATS (Aggregated)
# ============================================================================
@router.get("/steam/get-dashboard-stats")
@rate_limiter.limit(30, 60)  # 30 requests per minute per IP
async def get_dashboard_stats(
    request: Request,
    force_refresh: bool = False,
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    """Return the dashboard projection from the canonical player-match catalog."""
    steam_id = user.steam_id
    if force_refresh:
        clear_player_match_cache()
    player_matches = await asyncio.to_thread(
        list_player_matches,
        steam_id,
        EXPORTS_PATH,
    )

    matches_data = []
    weapon_stats_agg = {} # {weapon: {kills, shots, hits, damage, headshots}}
    map_results: dict[str, dict[str, int]] = {}
    
    # Aggregators for AIM
    agg_body_parts = {"head": 0, "chest": 0, "stomach": 0, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0, "generic": 0}
    agg_ttd_sum = 0
    agg_ttd_count = 0
    agg_crosshair_sum = 0
    agg_crosshair_count = 0
    agg_shots_fired = 0
    agg_shots_hit = 0

    if not player_matches:
        empty_response = {
            "steam_id": steam_id,
            "stats": {
                "total_matches": 0, "total_kills": 0, "total_deaths": 0,
                "avg_kd": 0.0, "avg_adr": 0.0, "avg_hs": 0.0, "win_rate": 0.0,
                "wins": 0, "losses": 0
            },
            "aim_stats": {},
            "recent_matches": [], "weapon_stats": [], "map_stats": []
        }
        return empty_response

    
    for player_match in player_matches:
        match_id = player_match.match_id
        player_stats = player_match.player
        metadata = player_match.metadata
        team_score = player_match.team_score
        opponent_score = player_match.opponent_score
        match_info = {
            "map_name": metadata.get("map_name", "unknown"),
            "date": metadata.get("date", ""),
            "team_score": team_score,
            "opponent_score": opponent_score,
            "winner": metadata.get("winner", ""),
            "duration_seconds": metadata.get("duration_seconds", 0),
            "total_rounds": metadata.get("total_rounds", 1),
        }
        
        if player_stats:
            result = player_match.result
            
            # --- START AGGREGATION FOR AIM & WEAPONS ---
            
            # 1. Body Parts (add to total)
            bp = player_stats.get("body_part_hits", {})
            for part, count in bp.items():
                if part in agg_body_parts:
                    agg_body_parts[part] += count
            
            # 2. Aim Metrics
            ttd = player_stats.get("time_to_damage_avg_ms", 0)
            if ttd > 0:
                agg_ttd_sum += ttd
                agg_ttd_count += 1
            
            cp = player_stats.get("crosshair_placement_avg_error", 0)
            if cp > 0:
                agg_crosshair_sum += cp
                agg_crosshair_count += 1
                
            agg_shots_fired += player_stats.get("shots_fired", 0)
            agg_shots_hit += player_stats.get("shots_hit", 0)

            # 3. Weapon Stats Aggregation
            if "weapon_stats" in player_stats:
                for weapon, stats in player_stats["weapon_stats"].items():
                    if weapon not in weapon_stats_agg:
                        weapon_stats_agg[weapon] = {"kills": 0, "shots": 0, "hits": 0, "damage": 0, "headshots": 0}
                    
                    ws = weapon_stats_agg[weapon]
                    ws["kills"] += stats.get("kills", 0)
                    ws["shots"] += stats.get("shots_fired", 0)
                    ws["hits"] += stats.get("shots_hit", 0)
                    ws["damage"] += stats.get("damage", 0)
                    ws["headshots"] += stats.get("headshots", 0)

            match_data = {
                "match_id": match_id,
                "map": match_info.get("map_name", "unknown"),
                "date": match_info.get("date", ""),
                "result": result,
                "team_score": team_score,
                "opponent_score": opponent_score,
                "kills": player_stats.get("kills", 0),
                "deaths": player_stats.get("deaths", 0),
                "assists": player_stats.get("assists", 0),
                "kd_ratio": player_stats.get("kd_ratio", 0.0),
                "adr": player_stats.get("adr", 0.0),
                "hs_percentage": player_stats.get("hs_percentage", 0.0),
                "headshots": player_stats.get("headshots", 0),
            }
            matches_data.append(match_data)
            
            map_name = match_info.get("map_name", "unknown")
            if map_name not in map_results:
                map_results[map_name] = {"wins": 0, "losses": 0}
            if result == "W":
                map_results[map_name]["wins"] += 1
            else:
                map_results[map_name]["losses"] += 1

    
    # Aggregation
    total_matches = len(matches_data)
    total_kills = sum(m["kills"] for m in matches_data)
    total_deaths = sum(m["deaths"] for m in matches_data)
    avg_kd = total_kills / total_deaths if total_deaths > 0 else 0.0
    avg_adr = sum(m["adr"] for m in matches_data) / total_matches if total_matches > 0 else 0.0
    avg_hs = sum(m["hs_percentage"] for m in matches_data) / total_matches if total_matches > 0 else 0.0
    wins = sum(1 for m in matches_data if m["result"] == "W")
    losses = total_matches - wins
    win_rate = (wins / total_matches * 100) if total_matches > 0 else 0.0
    
    # Final Aim Stats
    avg_ttd = agg_ttd_sum / agg_ttd_count if agg_ttd_count > 0 else 0
    avg_crosshair_error = agg_crosshair_sum / agg_crosshair_count if agg_crosshair_count > 0 else 0
    accuracy_overall = (agg_shots_hit / agg_shots_fired * 100) if agg_shots_fired > 0 else 0
    
    # Final Weapon Stats List (Full)
    weapon_stats_list = []
    for w, s in weapon_stats_agg.items():
        if s["kills"] > 0 or s["shots"] > 0:
            weapon_stats_list.append({
                "weapon": w,
                "kills": s["kills"],
                "accuracy": round((s["hits"] / s["shots"] * 100), 1) if s["shots"] > 0 else 0,
                "hs_pct": round((s["headshots"] / s["kills"] * 100), 1) if s["kills"] > 0 else 0,
                "damage": s["damage"]
            })
    # Sort by kills
    weapon_stats_list.sort(key=lambda x: x["kills"], reverse=True)
    
    # Map stats
    map_stats_list = []
    for map_name, results in map_results.items():
        total = results["wins"] + results["losses"]
        wr = (results["wins"] / total * 100) if total > 0 else 0.0
        map_stats_list.append({
            "map": map_name,
            "wins": results["wins"],
            "losses": results["losses"],
            "win_rate": round(wr, 1)
        })
    map_stats_list.sort(key=lambda x: x["wins"] + x["losses"], reverse=True)
    
    recent_matches = sorted(matches_data, key=lambda x: x["date"], reverse=True)[:10]
    
    response = {
        "steam_id": steam_id,
        "stats": {
            "total_matches": total_matches,
            "total_kills": total_kills,
            "total_deaths": total_deaths,
            "avg_kd": round(avg_kd, 2),
            "avg_adr": round(avg_adr, 1),
            "avg_hs": round(avg_hs, 1),
            "win_rate": round(win_rate, 1),
            "wins": wins,
            "losses": losses
        },
        "aim_stats": {
            "accuracy_overall": round(accuracy_overall, 1),
            "time_to_damage_avg_ms": round(avg_ttd, 1),
            "crosshair_placement_avg_error": round(avg_crosshair_error, 1),
            "body_part_hits": agg_body_parts
        },
        "recent_matches": recent_matches,
        "weapon_stats": weapon_stats_list,
        "map_stats": map_stats_list
    }
    
    logging.info("[dashboard-stats] Calculado para %s - %s partidas", steam_id, total_matches)
    
    return response


# ============================================================================
# MAP ZONE STATS
# ============================================================================

ZONE_MAPPING = {
    # Site A
    "BombsiteA": "site-a", "UnderA": "site-a", "ARamp": "site-a", "ASite": "site-a", "TopofA": "site-a", "A-Site": "site-a", "Palace": "site-a",
    # Site B
    "BombsiteB": "site-b", "BWindow": "site-b", "BSite": "site-b", "CloseDoors": "site-b", "BPlat": "site-b", "Side": "site-b", "Apartments": "site-b", "B-Site": "site-b",
    # Mid
    "MidDoors": "mid", "TopMid": "mid", "LowerMid": "mid", "Palm": "mid", "Xbox": "mid", "MidCT": "mid", "Mid": "mid", "Connector": "mid", "Middle": "mid", "Top-Mid": "mid",
    # Long A
    "LongA": "long-a", "LongDoors": "long-a", "Pit": "long-a", "BlueDoor": "long-a", "LongCorner": "long-a", "SideAlley": "long-a", "A-Entrance": "long-a",
    # B Tunnels
    "UpperTunnel": "b-tunnels", "LowerTunnel": "b-tunnels", "BTunnel": "b-tunnels", "TunnelStairs": "b-tunnels", "Tunnels": "b-tunnels", "B-Tunnel": "b-tunnels", "B-Tunnels": "b-tunnels",
    # Catwalk / Short
    "ShortStairs": "catwalk", "ShortA": "catwalk", "Catwalk": "catwalk", "Short": "catwalk", "ShortCorner": "catwalk",
    # T Spawn
    "TSpawn": "t-spawn", "TRamp": "t-spawn", "TPlat": "t-spawn", "Outside": "t-spawn", "OutsideTunnel": "t-spawn", "T-Spawn": "t-spawn", "Spawn": "t-spawn",
    # CT Spawn
    "CTSpawn": "ct-spawn", "BackSite": "ct-spawn", "CTMid": "ct-spawn", "CT-Spawn": "ct-spawn"
}

@router.get("/steam/get-map-zone-stats")
async def get_map_zone_stats(
    request: Request,
    map_name: str = "de_dust2",
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    """
    Endpoint for map zone statistics with side split (T/CT).
    """
    steam_id_str = user.steam_id
    logging.info(f"[map-zone-stats] Request for sid={steam_id_str}, map={map_name}")
    
    zone_ids = ["site-a", "site-b", "mid", "long-a", "b-tunnels", "catwalk", "t-spawn", "ct-spawn"]
    
    def empty_side_stats():
        return {"kills": 0, "deaths": 0, "duels_won": 0, "duels_lost": 0}
    
    # {zid: {"t": {...}, "ct": {...}, "combined": {...}}}
    zone_stats = {zid: {"t": empty_side_stats(), "ct": empty_side_stats(), "combined": empty_side_stats()} for zid in zone_ids}
    
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    matches_analyzed = 0
    
    if os.path.exists(exports_path):
        for match_export in iter_canonical_matches(exports_path):
            folder_name = match_export.directory.name
            meta = project_match_metadata(match_export)
            if meta.get("map_name") != map_name:
                continue

            players = project_player_summary(match_export).get("players", [])
            if not any(str(player.get("steam_id")) == steam_id_str for player in players):
                continue

            try:
                combat_data = project_combat(match_export)
                for round_item in combat_data.get("rounds", []):
                        for duel in round_item.get("duels", []):
                            attacker = duel.get("attacker", {})
                            victims = duel.get("victims", [])
                            outcome = duel.get("outcome", "")
                            
                            # User is attacker
                            if str(attacker.get("steam_id")) == steam_id_str:
                                zid = ZONE_MAPPING.get(attacker.get("map_area", ""))
                                team = attacker.get("team", "").lower()
                                if zid and zid in zone_stats and team in ["t", "ct"]:
                                    if outcome == "kill":
                                        zone_stats[zid][team]["kills"] += 1
                                        zone_stats[zid]["combined"]["kills"] += 1
                                        zone_stats[zid][team]["duels_won"] += 1
                                        zone_stats[zid]["combined"]["duels_won"] += 1
                                    elif outcome == "damage":
                                        zone_stats[zid][team]["duels_won"] += 1
                                        zone_stats[zid]["combined"]["duels_won"] += 1

                            # User is victim
                            for vic in victims:
                                if str(vic.get("steam_id")) == steam_id_str:
                                    zid = ZONE_MAPPING.get(vic.get("map_area", ""))
                                    team = vic.get("team", "").lower()
                                    if zid and zid in zone_stats and team in ["t", "ct"]:
                                        if vic.get("health_after", 0) == 0:
                                            zone_stats[zid][team]["deaths"] += 1
                                            zone_stats[zid]["combined"]["deaths"] += 1
                                        zone_stats[zid][team]["duels_lost"] += 1
                                        zone_stats[zid]["combined"]["duels_lost"] += 1
                matches_analyzed += 1
            except Exception as e:
                logging.error(f"[map-zone-stats] Error in match {folder_name}: {e}")
                continue
    
    # Calculate derived stats
    final_zones = {}
    for zid, sides in zone_stats.items():
        res = {}
        for side in ["t", "ct", "combined"]:
            s_data = sides[side]
            total = s_data["duels_won"] + s_data["duels_lost"]
            wr = round((s_data["duels_won"] / total * 100) if total > 0 else 50, 1)
            res[side] = {
                "kills": s_data["kills"],
                "deaths": s_data["deaths"],
                "win_rate": wr,
                "duels_won": s_data["duels_won"],
                "duels_total": total
            }
        
        rating = "good" if res["combined"]["win_rate"] >= 55 else "bad" if res["combined"]["win_rate"] <= 45 else "neutral"
        
        final_zones[zid] = {
            "kills": res["combined"]["kills"],
            "deaths": res["combined"]["deaths"],
            "winRate": res["combined"]["win_rate"],
            "rating": rating,
            "ct_stats": res["ct"],
            "t_stats": res["t"]
        }
    
    return {
        "zones": final_zones,
        "map_name": map_name,
        "matches_analyzed": matches_analyzed
    }


# ============================================================================
# CALLOUT STATS (Detailed)
# ============================================================================

@router.get("/steam/get-callout-stats")
async def get_callout_stats(
    request: Request,
    map_name: str = "de_dust2",
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    """
    Granular per-callout statistics with full data: K/D, win rates, weapons, context, etc.
    
    OPTIMIZATION (2026-01-08):
    - Now reads from pre-calculated data/users/{steam_id}/maps/{map_name}.json
    - O(1) file lookup instead of O(n) folder scanning
    """
    steam_id_str = user.steam_id
    logging.info(f"[callout-stats] Request for sid={steam_id_str}, map={map_name}")
    
    # ==========================================================================
    # FAST PATH: Try to load from pre-calculated map aggregate file (O(1))
    # ==========================================================================
    if (
        user_has_map_data(steam_id_str, map_name)
        and await asyncio.to_thread(
            is_user_map_data_fresh,
            steam_id_str,
            map_name,
        )
    ):
        logging.info(f"[callout-stats] Using pre-calculated map data for {steam_id_str}/{map_name}")
        map_data = await asyncio.to_thread(load_user_map_data, steam_id_str, map_name)
        
        if map_data:
            return {
                "callouts": map_data.get("callout_stats", {}),
                "heatmap_data": map_data.get("heatmap_data", []),
                "matches_analyzed": map_data.get("matches_analyzed", 0),
                "map_name": map_name,
                "side_stats": map_data.get("side_stats", {})
            }
    
    # ==========================================================================
    # FALLBACK: ProyecciÃ³n canÃ³nica bajo demanda si no existe agregado.
    # ==========================================================================
    logging.info(f"[callout-stats] Fallback to folder scanning for {steam_id_str}/{map_name}")
    

    # Init stats with full structure
    callout_stats = {}
    heatmap_points = []
    
    # Global Side Stats Accumulator
    side_stats_totals = {
        "CT": {"kills": 0, "deaths": 0, "headshots": 0, "adr_sum": 0, "adr_count": 0},
        "T": {"kills": 0, "deaths": 0, "headshots": 0, "adr_sum": 0, "adr_count": 0}
    }
    
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    
    def init_callout():
        return {
            "kills": 0, "deaths": 0,
            "ct_kills": 0, "ct_deaths": 0,
            "t_kills": 0, "t_deaths": 0,
            "positions_x": [], "positions_y": [], "positions_z": [],  # [MODIFIED] Added positions_z
            # Weapon tracking
            "weapon_kills": {},  # {weapon: kills}
            "weapon_deaths": {},  # {weapon: deaths}
            # Context stats
            "opening_kills": 0, "opening_attempts": 0,
            "trade_kills": 0, "trade_deaths": 0,
            "smoke_kills": 0, "smoke_deaths": 0,
            "wallbang_kills": 0,
            # Metrics
            "distances": [],
            "time_to_damages": [],
            "flash_deaths": 0, "total_deaths_for_flash": 0
        }
    
    if not os.path.exists(exports_path):
        return {"callouts": {}, "heatmap_data": [], "matches_analyzed": 0, "map_name": map_name}
    
    matches_analyzed = 0
    
    for match_export in iter_canonical_matches(exports_path):
        meta = project_match_metadata(match_export)
        if meta.get("map_name") != map_name:
            continue

        player_summary = None
        try:
            players = project_player_summary(match_export).get("players", [])
            for player in players:
                if str(player.get("steam_id")) == steam_id_str:
                    player_summary = player
                    break
            
            if not player_summary:
                continue
                
            # Collect ADR from summary (if available)
            # Note: We rely on players_summary for ADR as it handles round counting per side correctly
            if "ct_adr" in player_summary:
                side_stats_totals["CT"]["adr_sum"] += player_summary.get("ct_adr", 0)
                side_stats_totals["CT"]["adr_count"] += 1
            if "t_adr" in player_summary:
                side_stats_totals["T"]["adr_sum"] += player_summary.get("t_adr", 0)
                side_stats_totals["T"]["adr_count"] += 1
                
        except Exception: continue
        
        try:
            combat_data = project_combat(match_export)
            for round_item in combat_data.get("rounds", []):
                if isinstance(round_item, dict):
                    duels = round_item.get("duels", [])
                    
                    # Find first kill tick for opening duel detection
                    first_kill_tick = None
                    for d in duels:
                        if d.get("outcome") == "kill":
                            first_kill_tick = d.get("tick_start", 0)
                            break
                    
                    for duel in duels:
                        attacker = duel.get("attacker", {})
                        victims = duel.get("victims", [])
                        outcome = duel.get("outcome", "")
                        context = duel.get("context", {})
                        tick = duel.get("tick_start", 0)
                        
                        is_opening = (tick == first_kill_tick) if first_kill_tick else False
                        
                        # Attacker stats
                        if str(attacker.get("steam_id")) == steam_id_str:
                            raw_callout = attacker.get("map_area", "Unknown")
                            callout = normalize_callout(raw_callout, map_name)
                            user_team = attacker.get("team", "")
                            weapon = attacker.get("weapon", "Unknown")
                            
                            # Update Global Side Stats (Kills)
                            if outcome == "kill" and user_team in ["CT", "T"]:
                                side_stats_totals[user_team]["kills"] += 1
                                side_stats_totals[user_team]["headshots"] += attacker.get("headshots", 0)
                            
                            if callout:
                                if callout not in callout_stats:
                                    callout_stats[callout] = init_callout()
                                
                                cs = callout_stats[callout]
                                
                                if outcome == "kill":
                                    cs["kills"] += 1
                                    
                                    # CT/T
                                    if user_team == "CT":
                                        cs["ct_kills"] += 1
                                    elif user_team == "T":
                                        cs["t_kills"] += 1
                                    
                                    if weapon not in cs["weapon_kills"]:
                                        cs["weapon_kills"][weapon] = 0
                                    cs["weapon_kills"][weapon] += 1
                                    
                                    # Context stats
                                    if is_opening:
                                        cs["opening_kills"] += 1
                                        cs["opening_attempts"] += 1
                                    if context.get("is_trade", False):
                                        cs["trade_kills"] += 1
                                    if context.get("through_smoke", False):
                                        cs["smoke_kills"] += 1
                                    if context.get("is_wallbang", False) or context.get("penetrated_objects", 0) > 0:
                                        cs["wallbang_kills"] += 1
                                
                                # Distance and TTD
                                if context.get("distance"):
                                    cs["distances"].append(context["distance"])
                                if attacker.get("time_to_first_damage"):
                                    cs["time_to_damages"].append(attacker["time_to_first_damage"])
                        
                        # Victim stats
                        for vic in victims:
                            if str(vic.get("steam_id")) == steam_id_str:
                                raw_callout = vic.get("map_area", "Unknown")
                                callout = normalize_callout(raw_callout, map_name)
                                user_team = vic.get("team", "")
                                
                                # Update Global Side Stats (Deaths)
                                if vic.get("health_after", 0) == 0 and user_team in ["CT", "T"]:
                                    side_stats_totals[user_team]["deaths"] += 1
                                
                                if callout:
                                    if callout not in callout_stats:
                                        callout_stats[callout] = init_callout()
                                    
                                    cs = callout_stats[callout]
                                    
                                    if vic.get("health_after", 0) == 0:
                                        cs["deaths"] += 1
                                        cs["total_deaths_for_flash"] += 1
                                        
                                        # CT/T
                                        if user_team == "CT":
                                            cs["ct_deaths"] += 1
                                        elif user_team == "T":
                                            cs["t_deaths"] += 1
                                        
                                        # Track attacker weapon for death
                                        killer_weapon = attacker.get("weapon", "Unknown")
                                        if killer_weapon not in cs["weapon_deaths"]:
                                            cs["weapon_deaths"][killer_weapon] = 0
                                        cs["weapon_deaths"][killer_weapon] += 1
                                        
                                        # Context
                                        if is_opening:
                                            cs["opening_attempts"] += 1
                                        if context.get("is_trade", False):
                                            cs["trade_deaths"] += 1
                                        if context.get("through_smoke", False):
                                            cs["smoke_deaths"] += 1
                                        
                                        # Flash death
                                        if vic.get("is_blind", False):
                                            cs["flash_deaths"] += 1
                                        
            matches_analyzed += 1
        except Exception as e:
            logging.error(f"[callout-stats] Error {e}")
            continue
        
        try:
            for player in match_export.iter_player_states():
                if str(player.get("player_id", "")).removeprefix("steam:") != steam_id_str:
                    continue
                normalized_area = normalize_callout(player.get("area", ""), map_name)
                if not normalized_area or normalized_area not in callout_stats:
                    continue
                position = player.get("position") or {}
                if not position.get("x") or not position.get("y"):
                    continue
                callout_stats[normalized_area]["positions_x"].append(position["x"])
                callout_stats[normalized_area]["positions_y"].append(position["y"])
                if position.get("z") is not None:
                    callout_stats[normalized_area]["positions_z"].append(position["z"])
        except Exception as exc:
            logging.warning("[callout-stats] Error leyendo estados canÃ³nicos: %s", exc)

    # 5. Build final response with all fields
    final_callouts = {}
    for callout, stats in callout_stats.items():
        kills = stats["kills"]
        deaths = stats["deaths"]
        total_duels = kills + deaths
        
        kd = round(kills/deaths, 2) if deaths > 0 else float(kills)
        win_rate = round(kills / total_duels * 100, 1) if total_duels > 0 else 50.0
        rating = "good" if win_rate >= 55 else "bad" if win_rate <= 45 else "neutral"
        
        # Position from tracking
        position = None
        avg_z = None  # [NEW] Average Z coordinate for level detection
        if stats.get("positions_x") and stats.get("positions_y"):
            avg_x = sum(stats["positions_x"]) / len(stats["positions_x"])
            avg_y = sum(stats["positions_y"]) / len(stats["positions_y"])

            # [NEW] Calculate avg_z from tracking positions
            if stats.get("positions_z"):
                avg_z = sum(stats["positions_z"]) / len(stats["positions_z"])

            radar_map_name = resolve_radar_map_name(map_name, avg_z)
            position = game_to_radar_percent(avg_x, avg_y, radar_map_name)

        level = resolve_level_name(map_name, avg_z)
        if map_name == "de_nuke" and level is None:
            level = infer_nuke_level_from_callout(callout)

        fixed_position = resolve_fixed_callout_position(map_name, callout, level)
        if fixed_position:
            position = fixed_position
        
        # CT/T split
        ct_t_split = {
            "ct_kills": stats.get("ct_kills", 0),
            "ct_deaths": stats.get("ct_deaths", 0),
            "t_kills": stats.get("t_kills", 0),
            "t_deaths": stats.get("t_deaths", 0)
        }
        
        # Build weapon_stats array
        weapon_stats = []
        all_weapons = set(stats.get("weapon_kills", {}).keys()) | set(stats.get("weapon_deaths", {}).keys())
        for weapon in all_weapons:
            w_kills = stats.get("weapon_kills", {}).get(weapon, 0)
            w_deaths = stats.get("weapon_deaths", {}).get(weapon, 0)
            w_total = w_kills + w_deaths
            if w_total > 0:
                weapon_stats.append({
                    "weapon": weapon,
                    "kills": w_kills,
                    "deaths": w_deaths,
                    "kd": round(w_kills / w_deaths, 2) if w_deaths > 0 else float(w_kills)
                })
        weapon_stats.sort(key=lambda w: w["kills"] + w["deaths"], reverse=True)
        weapon_stats = weapon_stats[:5]  # Top 5
        
        # Context stats
        context_stats = {
            "opening_kills": stats.get("opening_kills", 0),
            "opening_attempts": stats.get("opening_attempts", 0),
            "trade_kills": stats.get("trade_kills", 0),
            "trade_deaths": stats.get("trade_deaths", 0),
            "smoke_kills": stats.get("smoke_kills", 0),
            "smoke_deaths": stats.get("smoke_deaths", 0),
            "wallbang_kills": stats.get("wallbang_kills", 0)
        }
        
        # Metrics
        avg_distance = round(sum(stats.get("distances", [])) / len(stats.get("distances", [])), 0) if stats.get("distances") else None
        avg_time_to_damage = round(sum(stats.get("time_to_damages", [])) / len(stats.get("time_to_damages", [])), 0) if stats.get("time_to_damages") else None
        flash_death_pct = round((stats["flash_deaths"] / stats["total_deaths_for_flash"]) * 100, 1) if stats["total_deaths_for_flash"] > 0 else 0.0
        
        final_callouts[callout] = {
            "kills": kills,
            "deaths": deaths,
            "kd": kd,
            "win_rate": win_rate,
            "rating": rating,
            "position": position,
            "avg_z": avg_z,  # [NEW] Z coordinate for level filtering
            "level": level,
            "sample_size": total_duels,
            "ct_t_split": ct_t_split,
            "weapon_stats": weapon_stats,
            "context_stats": context_stats,
            "avg_distance": avg_distance,
            "avg_time_to_damage": avg_time_to_damage,
            "flash_death_pct": flash_death_pct
        }
        
        # Heatmap points - Use avg_z from callout stats
        if position:
            for _ in range(kills):
                point = {"x": position["x"], "y": position["y"], "type": "kill", "callout": callout}
                if avg_z is not None:
                    point["avg_z"] = avg_z
                heatmap_points.append(point)
                
            for _ in range(deaths):
                point = {"x": position["x"], "y": position["y"], "type": "death", "callout": callout}
                if avg_z is not None:
                    point["avg_z"] = avg_z
                heatmap_points.append(point)

    # 6. Calculate Final Side Stats
    final_side_stats = {}
    for side in ["CT", "T"]:
        s = side_stats_totals[side]
        kills = s["kills"]
        deaths = s["deaths"]
        kd = round(kills / deaths, 2) if deaths > 0 else float(kills)
        
        # Average ADR
        adr_avg = round(s["adr_sum"] / s["adr_count"], 1) if s["adr_count"] > 0 else 0
        
        # HS Percentage
        hs_pct = round((s["headshots"] / kills * 100), 1) if kills > 0 else 0
        
        final_side_stats[side] = {
            "kills": kills,
            "deaths": deaths,
            "kd": kd,
            "adr": adr_avg,
            "hs_pct": hs_pct
        }

    return {
        "callouts": final_callouts,
        "heatmap_data": heatmap_points,
        "matches_analyzed": matches_analyzed,
        "map_name": map_name,
        "side_stats": final_side_stats
    }


# ============================================================================
# GRENADE STATS
# ============================================================================

def cluster_grenade_positions(positions: list, cluster_radius: float = 150.0) -> list:
    """
    Cluster grenade positions that are within cluster_radius units of each other.
    """
    if not positions:
        return []
    
    # Group by side first
    by_side = {}
    for pos in positions:
        side = pos.get("side", "unknown")
        if side not in by_side:
            by_side[side] = []
        by_side[side].append(pos)
    
    all_clusters = []
    
    for side, side_positions in by_side.items():
        used = set()
        for i, pos in enumerate(side_positions):
            if i in used:
                continue
            
            cluster_positions = [pos]
            cluster_indices = {i}
            
            for j, other_pos in enumerate(side_positions):
                if j in used or j == i:
                    continue
                dx = pos["game_x"] - other_pos["game_x"]
                dy = pos["game_y"] - other_pos["game_y"]
                dist = (dx**2 + dy**2) ** 0.5
                
                if dist <= cluster_radius:
                    cluster_positions.append(other_pos)
                    cluster_indices.add(j)
            
            used.update(cluster_indices)
            
            avg_x = sum(p["game_x"] for p in cluster_positions) / len(cluster_positions)
            avg_y = sum(p["game_y"] for p in cluster_positions) / len(cluster_positions)
            
            all_clusters.append({
                "game_x": avg_x,
                "game_y": avg_y,
                "count": len(cluster_positions),
                "side": side,
                "positions": cluster_positions
            })
    
    return all_clusters

@router.get("/steam/get-aggregate-grenades")
async def get_aggregate_grenades(
    request: Request,
    map_name: str = "de_dust2",
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    """
    Aggregate grenade statistics across all matches for a map.
    
    OPTIMIZATION (2026-01-08):
    - Now reads from pre-calculated data/users/{steam_id}/maps/{map_name}.json
    - O(1) file lookup instead of O(n) folder scanning
    """
    steam_id_str = user.steam_id
    logging.info(f"[aggregate-grenades] Request for sid={steam_id_str}, map={map_name}")
    
    # ==========================================================================
    # FAST PATH: Try to load from pre-calculated map aggregate file (O(1))
    # ==========================================================================
    if (
        user_has_map_data(steam_id_str, map_name)
        and await asyncio.to_thread(
            is_user_map_data_fresh,
            steam_id_str,
            map_name,
        )
    ):
        logging.info(f"[aggregate-grenades] Using pre-calculated map data for {steam_id_str}/{map_name}")
        map_data = await asyncio.to_thread(load_user_map_data, steam_id_str, map_name)
        
        if map_data and "grenades" in map_data:
            grenades = map_data["grenades"]
            raw_by_type = grenades.get("by_type", {})
            
            # Process each grenade type: cluster and compute stats (same as fallback)
            result_by_type = {}
            for g_type, positions in raw_by_type.items():
                if not positions:
                    result_by_type[g_type] = []
                    continue
                
                # Cluster positions using the same function as fallback
                clusters = cluster_grenade_positions(positions)
                
                # Process each cluster to add all required fields
                for c in clusters:
                    # Aggregate cluster stats from raw positions
                    raw_positions = c.get("positions", [])
                    c["total_damage"] = sum(p.get("damage_dealt", 0) for p in raw_positions)
                    c["avg_damage"] = round(c["total_damage"] / len(raw_positions), 1) if raw_positions else 0
                    c["total_blinded"] = sum(p.get("enemies_blinded", 0) for p in raw_positions)
                    c["avg_blinded"] = round(c["total_blinded"] / len(raw_positions), 2) if raw_positions else 0
                    
                    # Calculate avg_z for multi-level maps
                    z_values = [p.get("end_z", 0) for p in raw_positions if p.get("end_z", 0) != 0]
                    c["avg_z"] = round(sum(z_values) / len(z_values), 1) if z_values else None
                    c["areas"] = list(set(
                        normalize_grenade_area(p.get("land_area", ""), map_name)
                        for p in raw_positions
                        if p.get("land_area")
                    ))
                    c["level"] = resolve_cluster_level(map_name, c["avg_z"], c["areas"])

                    # Convert cluster center to radar coordinates
                    radar_map_name = resolve_radar_map_name_by_level(map_name, c["avg_z"], c["level"])
                    radar = game_to_radar_percent(c["game_x"], c["game_y"], radar_map_name)
                    c["x"] = radar["x"] if radar else 50
                    c["y"] = radar["y"] if radar else 50
                    
                    # Build trajectories from raw positions
                    trajectories = []
                    used_trajectories = set()
                    
                    for i, p in enumerate(raw_positions):
                        if i in used_trajectories:
                            continue
                        
                        end_x = p.get("end_x", 0)
                        end_y = p.get("end_y", 0)
                        
                        if end_x == 0 and end_y == 0:
                            continue
                        
                        # Group similar trajectories
                        traj_group = [p]
                        for j, other in enumerate(raw_positions):
                            if j == i or j in used_trajectories:
                                continue
                            other_end_x = other.get("end_x", 0)
                            other_end_y = other.get("end_y", 0)
                            dist = ((end_x - other_end_x) ** 2 + (end_y - other_end_y) ** 2) ** 0.5
                            if dist < 100:
                                traj_group.append(other)
                                used_trajectories.add(j)
                        
                        used_trajectories.add(i)
                        
                        # Average end position and convert to radar
                        avg_end_x = sum(t.get("end_x", 0) for t in traj_group) / len(traj_group)
                        avg_end_y = sum(t.get("end_y", 0) for t in traj_group) / len(traj_group)
                        
                        end_radar = game_to_radar_percent(avg_end_x, avg_end_y, radar_map_name)
                        
                        trajectories.append({
                            "x1": c["x"],
                            "y1": c["y"],
                            "x2": end_radar["x"] if end_radar else c["x"],
                            "y2": end_radar["y"] if end_radar else c["y"],
                            "count": len(traj_group),
                            "damage": sum(t.get("damage_dealt", 0) for t in traj_group),
                            "land_area": traj_group[0].get("land_area", "") if traj_group else ""
                        })
                    
                    # Sort by count and keep top 5
                    trajectories.sort(key=lambda t: t["count"], reverse=True)
                    c["trajectories"] = trajectories[:5]
                    
                    # Remove raw positions to reduce response size
                    if "positions" in c:
                        del c["positions"]
                
                result_by_type[g_type] = clusters
            
            return {
                "by_type": result_by_type,
                "summary": grenades.get("summary", {}),
                "insights": grenades.get("insights", []),
                "matches_analyzed": map_data.get("matches_analyzed", 0),
                "map_name": map_name
            }


    
    # ==========================================================================
    # FALLBACK: ProyecciÃ³n canÃ³nica bajo demanda si no existe agregado.
    # ==========================================================================
    logging.info(f"[aggregate-grenades] Fallback to folder scanning for {steam_id_str}/{map_name}")
    
    grenade_positions = {"smoke": [], "flash": [], "he": [], "molotov": []}
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    matches_analyzed = 0

    
    if os.path.exists(exports_path):
        for match_export in iter_canonical_matches(exports_path):
            meta = project_match_metadata(match_export)
            if meta.get("map_name") != map_name:
                continue

            players = project_player_summary(match_export).get("players", [])
            if not any(str(player.get("steam_id")) == steam_id_str for player in players):
                continue
            
            try:
                g_data = project_utility(match_export)
                for r_item in g_data.get("rounds", []):
                    for event in r_item.get("events", []):
                        if str(event.get("thrower_steam_id")) == steam_id_str:
                                    g_type = event.get("type", "").lower()
                                    # Normalize types
                                    if g_type == "flashbang":
                                        g_type = "flash"
                                    if g_type == "smoke grenade":
                                        g_type = "smoke"
                                    if g_type == "he grenade":
                                        g_type = "he"
                                    if g_type == "incendiary grenade":
                                        g_type = "molotov"
                                    
                                    if g_type in grenade_positions:
                                        sp = event.get("start_position", {})
                                        ep = event.get("end_position", {})
                                        if sp:
                                            grenade_positions[g_type].append({
                                                "game_x": sp.get("x", 0),
                                                "game_y": sp.get("y", 0),
                                                "end_x": ep.get("x", 0),
                                                "end_y": ep.get("y", 0),
                                                "end_z": ep.get("z", 0),  # Z coordinate for level filtering
                                                "side": event.get("thrower_side", "unknown"),
                                                "land_area": normalize_grenade_area(event.get("land_area", ""), map_name),
                                                # NEW: Extract full stats
                                                "damage_dealt": event.get("damage_dealt", 0),
                                                "enemies_blinded": event.get("enemies_blinded", 0),
                                                "allies_blinded": event.get("allies_blinded", 0),

                                                "duration": event.get("duration", 0),
                                                "extinguished": event.get("extinguished", False),
                                                "kills": event.get("kills", 0)
                                            })
            except Exception as exc:
                logging.warning("[aggregate-grenades] Error leyendo utilidad canÃ³nica: %s", exc)
            matches_analyzed += 1
            
    # Build summary stats from canonical utility events.
    summary = {
        "smoke": {"thrown": 0},
        "flash": {"thrown": 0, "total_blinded": 0, "avg_blinded": 0, "team_flashed": 0},
        "he": {"thrown": 0, "total_damage": 0, "avg_damage": 0, "kills": 0},
        "molotov": {"thrown": 0, "total_damage": 0, "avg_damage": 0, "avg_duration": 0, "extinguished": 0}
    }
    
    for g_type, positions in grenade_positions.items():
        count = len(positions)
        summary[g_type]["thrown"] = count
        
        if g_type == "smoke" and count > 0:
            pass
            
        elif g_type == "flash" and count > 0:
            total_enemies = sum(p.get("enemies_blinded", 0) for p in positions)
            total_allies = sum(p.get("allies_blinded", 0) for p in positions)
            summary["flash"]["total_blinded"] = total_enemies
            summary["flash"]["avg_blinded"] = round(total_enemies / count, 2) if count > 0 else 0
            summary["flash"]["team_flashed"] = total_allies
            
        elif g_type == "he" and count > 0:
            total_dmg = sum(p.get("damage_dealt", 0) for p in positions)
            total_kills = sum(p.get("kills", 0) for p in positions)
            summary["he"]["total_damage"] = total_dmg
            summary["he"]["avg_damage"] = round(total_dmg / count, 1) if count > 0 else 0
            summary["he"]["kills"] = total_kills
            
        elif g_type == "molotov" and count > 0:
            total_dmg = sum(p.get("damage_dealt", 0) for p in positions)
            total_duration = sum(p.get("duration", 0) for p in positions)
            extinguished = sum(1 for p in positions if p.get("extinguished", False))
            summary["molotov"]["total_damage"] = total_dmg
            summary["molotov"]["avg_damage"] = round(total_dmg / count, 1) if count > 0 else 0
            summary["molotov"]["avg_duration"] = round(total_duration / count, 2) if count > 0 else 0
            summary["molotov"]["extinguished"] = extinguished
    
    # Cluster and normalize
    result = {}
    for gtype, positions in grenade_positions.items():
        clusters = cluster_grenade_positions(positions)
        for c in clusters:
            # Aggregate cluster stats from raw positions
            raw_positions = c.get("positions", [])
            c["total_damage"] = sum(p.get("damage_dealt", 0) for p in raw_positions)
            c["avg_damage"] = round(c["total_damage"] / len(raw_positions), 1) if raw_positions else 0
            c["total_blinded"] = sum(p.get("enemies_blinded", 0) for p in raw_positions)
            c["avg_blinded"] = round(c["total_blinded"] / len(raw_positions), 2) if raw_positions else 0
            
            # Calculate avg_z for multi-level maps (Nuke, Vertigo, Train)
            z_values = [p.get("end_z", 0) for p in raw_positions if p.get("end_z", 0) != 0]
            c["avg_z"] = round(sum(z_values) / len(z_values), 1) if z_values else None
            c["areas"] = list(set(
                normalize_grenade_area(p.get("land_area", ""), map_name)
                for p in raw_positions
                if p.get("land_area")
            ))
            c["level"] = resolve_cluster_level(map_name, c["avg_z"], c["areas"])

            radar_map_name = resolve_radar_map_name_by_level(map_name, c["avg_z"], c["level"])
            radar = game_to_radar_percent(c["game_x"], c["game_y"], radar_map_name)
            c["x"] = radar["x"]
            c["y"] = radar["y"]
            
            # Build trajectories from raw positions
            trajectories = []
            used_trajectories = set()
            
            for i, p in enumerate(raw_positions):
                if i in used_trajectories:
                    continue
                    
                end_x = p.get("end_x", 0)
                end_y = p.get("end_y", 0)
                
                # Skip if no end position
                if end_x == 0 and end_y == 0:
                    continue
                
                # Group similar trajectories (end within 100 units)
                traj_group = [p]
                for j, other in enumerate(raw_positions):
                    if j == i or j in used_trajectories:
                        continue
                    other_end_x = other.get("end_x", 0)
                    other_end_y = other.get("end_y", 0)
                    dist = ((end_x - other_end_x) ** 2 + (end_y - other_end_y) ** 2) ** 0.5
                    if dist < 100:
                        traj_group.append(other)
                        used_trajectories.add(j)
                
                used_trajectories.add(i)
                
                # Average end position
                avg_end_x = sum(t.get("end_x", 0) for t in traj_group) / len(traj_group)
                avg_end_y = sum(t.get("end_y", 0) for t in traj_group) / len(traj_group)
                
                # Convert to radar coordinates
                end_radar = game_to_radar_percent(avg_end_x, avg_end_y, radar_map_name)
                
                trajectories.append({
                    "x1": radar["x"],
                    "y1": radar["y"],
                    "x2": end_radar["x"],
                    "y2": end_radar["y"],
                    "count": len(traj_group),
                    "land_area": traj_group[0].get("land_area", "")
                })
            
            c["trajectories"] = trajectories
            
            # Cleanup internal data
            del c["game_x"]
            del c["game_y"]
            if "positions" in c:
                del c["positions"]
                
        result[gtype] = clusters
    
    # Generate insights based on summary
    insights = []

    if summary["flash"]["avg_blinded"] >= 1.5:
        insights.append({
            "type": "strength",
            "text": f"✓ Excelentes flashes! Promedio de {summary['flash']['avg_blinded']} cegados por flash."
        })
    if summary["flash"]["team_flashed"] > summary["flash"]["total_blinded"] * 0.3:
        insights.append({
            "type": "warning",
            "text": f"⚠️ Alto ratio de team-flash ({summary['flash']['team_flashed']}). Coordina mejor con tu equipo."
        })
    if summary["he"]["avg_damage"] > 40:
        insights.append({
            "type": "strength",
            "text": f"✓ Buen uso de HE! Promedio de {summary['he']['avg_damage']} daño por granada."
        })
    if summary["molotov"]["extinguished"] > 0:
        insights.append({
            "type": "drill",
            "text": f"🎯 {summary['molotov']['extinguished']} molotovs apagados por smokes enemigos. Considera timing."
        })
        
    return {
        "by_type": result,
        "summary": summary,
        "insights": insights,
        "matches_analyzed": matches_analyzed,
        "map_name": map_name
    }


# ============================================================================
# MOVEMENT STATS (Hybrid Flow + Heatmap)
# ============================================================================

@router.get("/steam/get-movement-stats")
async def get_movement_stats(
    request: Request,
    map_name: str = "de_dust2",
    user: SteamUser = Depends(require_steam_user),
) -> dict[str, Any]:
    """
    Movement analysis for the Hybrid Flow + Heatmap visualization.
    
    OPTIMIZATION (2026-01-08):
    - Now reads from pre-calculated data/users/{steam_id}/maps/{map_name}.json
    - O(1) file lookup instead of O(n) folder scanning
    - Evita materializar estados completos de todas las partidas.
    
    Procesa estados canÃ³nicos por ronda para extraer:
    - Heatmap grid: Position density across 20x20 grid cells
    - Flow lines: Common routes between map areas
    - Metrics: Time-to-site, position frequency, etc.
    """
    steam_id_str = user.steam_id
    logging.info(f"[movement-stats] Request for sid={steam_id_str}, map={map_name}")
    
    # ==========================================================================
    # FAST PATH: Try to load from pre-calculated map aggregate file (O(1))
    # ==========================================================================
    if (
        user_has_map_data(steam_id_str, map_name)
        and await asyncio.to_thread(
            is_user_map_data_fresh,
            steam_id_str,
            map_name,
        )
    ):
        logging.info(f"[movement-stats] Using pre-calculated map data for {steam_id_str}/{map_name}")
        map_data = await asyncio.to_thread(load_user_map_data, steam_id_str, map_name)
        
        if map_data and "movement" in map_data:
            normalized_movement = normalize_movement_contract(
                map_data["movement"],
                map_name,
                map_data.get("callout_stats"),
            )
            
            return {
                **normalized_movement,
                "matches_analyzed": map_data.get("matches_analyzed", 0),
                "map_name": map_name
            }


    
    # ==========================================================================
    # FALLBACK: Estados canÃ³nicos por ronda si no existe agregado.
    # ==========================================================================
    logging.info(f"[movement-stats] Fallback to folder scanning for {steam_id_str}/{map_name}")
    
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    
    # Grid configuration (20x20 = 400 cells)
    GRID_SIZE = 20
    
    # Data structures
    grid_counts = {}  # {(grid_x, grid_y): {"total": 0, "ct": 0, "t": 0}}
    area_transitions = {}  # {(from_area, to_area): {"count": 0, "ct": 0, "t": 0}}
    area_time = {}  # {area: {"total_ticks": 0, "ct": 0, "t": 0}}
    time_to_site = {"A": {"ct": [], "t": []}, "B": {"ct": [], "t": []}}
    position_samples = []  # Raw positions for flow calculation
    
    matches_analyzed = 0
    total_rounds = 0
    
    if not os.path.exists(exports_path):
        return _empty_movement_response(map_name)

    
    for match_export in iter_canonical_matches(exports_path):
        meta = project_match_metadata(match_export)
        if meta.get("map_name") != map_name:
            continue

        players = project_player_summary(match_export).get("players", [])
        if not any(str(player.get("steam_id")) == steam_id_str for player in players):
            continue

        try:
            for round_num, states in match_export.iter_player_state_rounds(steam_id_str):
                total_rounds += 1
                
                # Track player's previous area for transitions
                prev_area = None
                prev_position = None
                prev_z = None
                prev_level = None
                round_team = None  # Team for this round (detected from spawn)
                round_start_tick = None
                reached_a = False
                reached_b = False
                first_a_tick = None
                first_b_tick = None
                first_tick_processed = False
                
                for player in states:
                    tick = player.get("tick", 0)
                    if not player.get("is_alive", False):
                        continue

                    pos = player.get("position") or {}
                    area = player.get("area", "")
                    if not pos or not area:
                        continue

                    game_x = pos.get("x", 0)
                    game_y = pos.get("y", 0)
                    team = str(player.get("side") or "").upper()
                    if not team and not first_tick_processed:
                        first_tick_processed = True
                        round_team = _infer_team_from_area(area, map_name)
                        round_start_tick = tick
                    if not team:
                        team = round_team

                    game_z = pos.get("z", 0)
                    if isinstance(game_z, (int, float)):
                        
                        # Grid heatmap
                        radar_pos = game_to_radar_percent(game_x, game_y, map_name)
                        grid_x = min(GRID_SIZE - 1, int(radar_pos["x"] / 100 * GRID_SIZE))
                        grid_y = min(GRID_SIZE - 1, int(radar_pos["y"] / 100 * GRID_SIZE))
                        level_key = (
                            "upper" if game_z >= NUKE_Z_THRESHOLD else "lower"
                        ) if map_name == "de_nuke" else "all"
                        grid_key = (grid_x, grid_y, level_key)
                        
                        if grid_key not in grid_counts:
                            grid_counts[grid_key] = {"total": 0, "ct": 0, "t": 0, "z_sum": 0.0}
                        grid_counts[grid_key]["total"] += 1
                        grid_counts[grid_key]["z_sum"] += game_z
                        if team:
                            grid_counts[grid_key][team.lower()] += 1
                        
                        # Area time tracking
                        normalized_area = normalize_callout(area, map_name) or area
                        area_key = (normalized_area, level_key)
                        if area_key not in area_time:
                            area_time[area_key] = {
                                "total_ticks": 0,
                                "ct": 0,
                                "t": 0,
                                "x_sum": 0.0,
                                "y_sum": 0.0,
                                "z_sum": 0.0,
                            }
                        area_time[area_key]["total_ticks"] += 1
                        area_time[area_key]["x_sum"] += radar_pos["x"]
                        area_time[area_key]["y_sum"] += radar_pos["y"]
                        area_time[area_key]["z_sum"] += game_z
                        if team:
                            area_time[area_key][team.lower()] += 1
                        
                        # Area transitions (flow lines)
                        if (
                            prev_area
                            and (
                                prev_area != normalized_area
                                or prev_level != level_key
                            )
                            and prev_position
                        ):
                            transition_key = (
                                prev_area,
                                prev_level,
                                normalized_area,
                                level_key,
                            )
                            if transition_key not in area_transitions:
                                area_transitions[transition_key] = {
                                    "count": 0,
                                    "ct": 0,
                                    "t": 0,
                                    "from_x_sum": 0.0,
                                    "from_y_sum": 0.0,
                                    "to_x_sum": 0.0,
                                    "to_y_sum": 0.0,
                                    "from_z_sum": 0.0,
                                    "to_z_sum": 0.0,
                                }
                            area_transitions[transition_key]["count"] += 1
                            area_transitions[transition_key]["from_x_sum"] += prev_position["x"]
                            area_transitions[transition_key]["from_y_sum"] += prev_position["y"]
                            area_transitions[transition_key]["to_x_sum"] += radar_pos["x"]
                            area_transitions[transition_key]["to_y_sum"] += radar_pos["y"]
                            area_transitions[transition_key]["from_z_sum"] += prev_z
                            area_transitions[transition_key]["to_z_sum"] += game_z
                            if team:
                                area_transitions[transition_key][team.lower()] += 1
                        
                        prev_area = normalized_area
                        prev_position = radar_pos
                        prev_z = game_z
                        prev_level = level_key
                        
                        # Time to site tracking
                        if not reached_a and _is_a_site(normalized_area):
                            reached_a = True
                            first_a_tick = tick
                        if not reached_b and _is_b_site(normalized_area):
                            reached_b = True
                            first_b_tick = tick
                        
                        # Store position sample for flow visualization
                        position_samples.append({
                            "x": radar_pos["x"],
                            "y": radar_pos["y"],
                            "area": normalized_area,
                            "team": team,
                            "tick": tick
                        })
                
                # Calculate time-to-site for this round
                if round_start_tick is not None:
                    if first_a_tick:
                        time_a = (first_a_tick - round_start_tick) / 64  # Convert ticks to seconds (64 tick)
                        side = round_team or "t"
                        time_to_site["A"][side.lower()].append(time_a)
                    if first_b_tick:
                        time_b = (first_b_tick - round_start_tick) / 64
                        side = round_team or "t"
                        time_to_site["B"][side.lower()].append(time_b)
            
            matches_analyzed += 1
            
        except Exception as exc:
            logging.warning("[movement-stats] Error leyendo estados de %s: %s", match_export.match_id, exc)
            continue
    
    # Build response
    
    # 1. Heatmap grid (20x20 cells with intensity)
    max_count = max((c["total"] for c in grid_counts.values()), default=1)
    heatmap_grid = []
    for (gx, gy, level), counts in grid_counts.items():
        intensity = round((counts["total"] / max_count) * 100, 1)
        if intensity > 1:  # Only include cells with meaningful data
            # Calculate average Z for multi-level maps (Nuke, Vertigo, Train)
            avg_z = counts.get("z_sum", 0) / counts["total"] if counts["total"] > 0 else 0
            heatmap_grid.append({
                "x": (gx + 0.5) * (100 / GRID_SIZE),  # Center of cell
                "y": (gy + 0.5) * (100 / GRID_SIZE),
                "intensity": intensity,
                "ct_count": counts["ct"],
                "t_count": counts["t"],
                "ct_ratio": round(counts["ct"] / counts["total"] * 100, 1) if counts["total"] > 0 else 50,
                "sample_count": counts["total"],
                "avg_z": round(avg_z, 1),  # For level filtering
                "level": level,
            })
    
    # Sort by intensity for rendering order
    heatmap_grid.sort(key=lambda x: x["intensity"])
    
    # 2. Flow lines (top 15 most common transitions)
    flow_lines = []
    sorted_transitions = sorted(area_transitions.items(), key=lambda x: x[1]["count"], reverse=True)
    
    for (
        from_area,
        from_level,
        to_area,
        to_level,
    ), data in sorted_transitions[:15]:
        if data["count"] < 2:  # Skip rare transitions
            continue
        
        flow_lines.append({
            "from_area": from_area,
            "to_area": to_area,
            "from_level": from_level,
            "to_level": to_level,
            "from_x": data["from_x_sum"] / data["count"],
            "from_y": data["from_y_sum"] / data["count"],
            "to_x": data["to_x_sum"] / data["count"],
            "to_y": data["to_y_sum"] / data["count"],
            "from_avg_z": data["from_z_sum"] / data["count"],
            "to_avg_z": data["to_z_sum"] / data["count"],
            "count": data["count"],
            "ct_count": data.get("ct", 0),
            "t_count": data.get("t", 0),
            "intensity": min(100, data["count"] * 5)  # Scale for visualization
        })
    
    # 3. Metrics
    # Top positions by time spent
    sorted_areas = sorted(area_time.items(), key=lambda x: x[1]["total_ticks"], reverse=True)
    top_positions = []
    total_ticks = sum(a["total_ticks"] for a in area_time.values())
    
    for (area, level), data in sorted_areas:
        pct = round((data["total_ticks"] / total_ticks) * 100, 1) if total_ticks > 0 else 0
        top_positions.append({
            "area": area,
            "level": level,
            "time_percent": pct,
            "ct_percent": round((data["ct"] / data["total_ticks"]) * 100, 1) if data["total_ticks"] > 0 else 50,
            "sample_count": data["total_ticks"],
            "position": {
                "x": data["x_sum"] / data["total_ticks"],
                "y": data["y_sum"] / data["total_ticks"],
            },
            "avg_z": data["z_sum"] / data["total_ticks"],
        })
    
    # Time to site averages
    avg_time_to_a = {
        "ct": round(sum(time_to_site["A"]["ct"]) / len(time_to_site["A"]["ct"]), 1) if time_to_site["A"]["ct"] else None,
        "t": round(sum(time_to_site["A"]["t"]) / len(time_to_site["A"]["t"]), 1) if time_to_site["A"]["t"] else None
    }
    avg_time_to_b = {
        "ct": round(sum(time_to_site["B"]["ct"]) / len(time_to_site["B"]["ct"]), 1) if time_to_site["B"]["ct"] else None,
        "t": round(sum(time_to_site["B"]["t"]) / len(time_to_site["B"]["t"]), 1) if time_to_site["B"]["t"] else None
    }
    
    metrics = {
        "avg_time_to_a": avg_time_to_a,
        "avg_time_to_b": avg_time_to_b,
        "top_positions": top_positions,
        "total_rounds": total_rounds,
        "total_samples": len(position_samples)
    }
    
    logging.info(f"[movement-stats] Processed {matches_analyzed} matches, {total_rounds} rounds, {len(heatmap_grid)} grid cells, {len(flow_lines)} flow lines")
    
    normalized_movement = normalize_movement_contract(
        {
            "heatmap_grid": heatmap_grid,
            "flow_lines": flow_lines,
            "metrics": metrics,
        },
        map_name,
    )

    return {
        **normalized_movement,
        "matches_analyzed": matches_analyzed,
        "map_name": map_name
    }


def _empty_movement_response(map_name: str) -> dict:
    """Return empty response structure for movement stats."""
    return {
        "heatmap_grid": [],
        "flow_lines": [],
        "metrics": {
            "avg_time_to_a": {"ct": None, "t": None},
            "avg_time_to_b": {"ct": None, "t": None},
            "top_positions": [],
            "total_rounds": 0,
            "total_samples": 0
        },
        "matches_analyzed": 0,
        "map_name": map_name
    }


def _infer_team_from_area(area: str, map_name: str) -> str | None:
    """Infer team from spawn area names."""
    area_lower = area.lower()
    if "ctspawn" in area_lower or "ct spawn" in area_lower or "ct_spawn" in area_lower:
        return "CT"
    if "tspawn" in area_lower or "t spawn" in area_lower or "t_spawn" in area_lower:
        return "T"
    return None


def _is_a_site(area: str) -> bool:
    """Check if area is A site or A-related."""
    area_lower = area.lower()
    return "a site" in area_lower or "asite" in area_lower or area_lower == "a"


def _is_b_site(area: str) -> bool:
    """Check if area is B site or B-related."""
    area_lower = area.lower()
    return "b site" in area_lower or "bsite" in area_lower or area_lower == "b"


def _load_canonical_replay_index(match_folder: str) -> dict | None:
    try:
        payload = CanonicalMatch(match_folder).replay_index()
    except (OSError, UnicodeError, json.JSONDecodeError):
        return None
    return thaw_json(payload) if payload else None


def _load_canonical_replay_round(match_folder: str, round_num: int) -> dict | None:
    try:
        round_data = CanonicalMatch(match_folder).replay_round(round_num)
    except (OSError, UnicodeError, json.JSONDecodeError):
        return None
    return thaw_json(round_data) if round_data else None


def _stream_match_replay(match: CanonicalMatch):
    index = match.replay_index()
    separators = (",", ":")
    yield '{"metadata":'
    yield json.dumps(
        thaw_json(index.get("metadata", {})),
        ensure_ascii=False,
        separators=separators,
    )
    yield ',"rounds":['
    first = True
    for entry in index.get("rounds", ()):
        if not isinstance(entry, Mapping):
            continue
        round_number = int(entry.get("round_number") or 0)
        round_data = match.replay_round(round_number)
        if not round_data:
            continue
        if not first:
            yield ","
        first = False
        yield json.dumps(
            thaw_json(round_data),
            ensure_ascii=False,
            separators=separators,
        )
    yield "]}"


@router.get("/match/{match_id}/replay")
async def get_match_replay(match_id: str) -> StreamingResponse:
    """
    Endpoint para obtener datos de replay 2D de una partida.
    
    Emite el replay segmentado sin materializar la partida completa en memoria.
    """
    logging.info(f"[get-match-replay] Fetching replay data for {match_id}")
    
    match_export = find_canonical_match(EXPORTS_PATH, match_id)
    if match_export is None or not match_export.replay_index():
        raise HTTPException(status_code=404, detail="Replay data not found for this match")
    return StreamingResponse(
        _stream_match_replay(match_export),
        media_type="application/json",
    )


@router.get("/match/{match_id}/replay/metadata")
async def get_match_replay_metadata(match_id: str) -> dict:
    """
    Endpoint para obtener solo los metadatos del replay (carga instantánea).
    
    Devuelve metadata + resumen de rondas sin los frames (para carga progresiva).
    """
    logging.info(f"[get-match-replay-metadata] Fetching metadata for {match_id}")
    
    match_export = find_canonical_match(EXPORTS_PATH, match_id)
    if match_export is None:
        raise HTTPException(status_code=404, detail="Replay data not found for this match")
    canonical_index = thaw_json(match_export.replay_index())
    if not canonical_index:
        raise HTTPException(status_code=404, detail="Replay data not found for this match")
    rounds_summary = [
        {
            "round": item.get("round_number"),
            "winner": (item.get("winner_side") or "").upper(),
            "start_tick": item.get("start_tick", 0),
            "end_tick": item.get("end_tick", 0),
            "frame_count": item.get("frame_count", 0),
            "event_count": item.get("event_count", 0),
            "events": item.get("events", []),
        }
        for item in canonical_index.get("rounds", [])
        if isinstance(item, dict)
    ]
    return {
        "metadata": canonical_index.get("metadata", {}),
        "rounds_summary": rounds_summary,
        "total_rounds": len(rounds_summary),
    }


@router.get("/match/{match_id}/replay/round/{round_num}")
async def get_match_replay_round(match_id: str, round_num: int) -> dict:
    """
    Endpoint para obtener los datos de una ronda específica (carga lazy).
    
    Solo devuelve los frames de la ronda solicitada para reducir tamaño de transferencia.
    """
    logging.info(f"[get-match-replay-round] Fetching round {round_num} for {match_id}")
    
    match_export = find_canonical_match(EXPORTS_PATH, match_id)
    if match_export is None:
        raise HTTPException(status_code=404, detail="Replay data not found for this match")
    try:
        round_data = thaw_json(match_export.replay_round(round_num))
        if not round_data:
            raise HTTPException(status_code=404, detail=f"Round {round_num} not found")
        return {
            "round": round_data.get("round", round_num),
            "winner": round_data.get("winner", ""),
            "start_tick": round_data.get("start_tick", 0),
            "end_tick": round_data.get("end_tick", 0),
            "frames": round_data.get("frames", []),
            "events": round_data.get("events", []),
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logging.error(f"[get-match-replay-round] Error: {e}")
        raise HTTPException(status_code=500, detail="Error reading round data")

