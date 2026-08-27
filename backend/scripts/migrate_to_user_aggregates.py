#!/usr/bin/env python3
# backend/scripts/migrate_to_user_aggregates.py
# ==============================================
# Migration script to generate user aggregate files from existing match data.
#
# This script scans all existing matches in data/exports/ and generates:
#   - data/users/{steam_id}/aggregate.json       (global stats)
#   - data/users/{steam_id}/match_history.json   (match list)
#   - data/users/{steam_id}/maps/{map}.json      (per-map Dashboard data)
#
# Run this script ONCE after implementing the aggregate system.
# After that, the Go service will update aggregates incrementally.
#
# Usage:
#   cd backend
#   python scripts/migrate_to_user_aggregates.py

import os
import sys
import json
import logging
from datetime import datetime
from collections import defaultdict
from pathlib import Path
from typing import Any

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from app.utils.user_aggregates import (
    ensure_user_directory,
    save_user_aggregate,
    save_user_map_data,
    add_match_to_history,
    create_empty_aggregate,
    create_empty_map_aggregate,
    EXPORTS_PATH,
    USERS_PATH
)
from app.matches.canonical_repository import CanonicalMatch
from app.matches.combat_web_projection import project_combat
from app.matches.match_web_projection import (
    project_match_metadata,
    project_player_summary,
    project_utility,
)
from app.utils.maps import normalize_callout, game_to_radar_percent

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s'
)
logger = logging.getLogger(__name__)

NUKE_Z_THRESHOLD = -500


class MigrationAggregator:
    """
    Aggregates data from multiple matches for a single user.
    """
    
    def __init__(self, steam_id: str):
        self.steam_id = str(steam_id)
        
        # Global stats (for aggregate.json)
        self.total_matches = 0
        self.total_kills = 0
        self.total_deaths = 0
        self.total_assists = 0
        self.total_adr_sum = 0.0
        self.total_hs_sum = 0.0
        self.wins = 0
        self.losses = 0
        
        # Aim stats accumulators
        self.body_part_hits = defaultdict(int)
        self.shots_fired = 0
        self.shots_hit = 0
        self.ttd_sum = 0.0
        self.ttd_count = 0
        self.crosshair_sum = 0.0
        self.crosshair_count = 0
        
        # Weapon stats
        self.weapon_stats = defaultdict(lambda: {
            "kills": 0, "headshots": 0, "damage": 0,
            "shots_fired": 0, "shots_hit": 0
        })
        
        # Map stats
        self.map_stats = defaultdict(lambda: {"wins": 0, "losses": 0, "matches": 0})
        
        # Recent matches (for aggregate.json)
        self.recent_matches = []
        
        # Per-map data (for maps/{map}.json)
        self.map_data = {}  # {map_name: MapAggregator}
    
    def get_map_aggregator(self, map_name: str) -> 'MapAggregator':
        """Get or create a MapAggregator for the given map."""
        if map_name not in self.map_data:
            self.map_data[map_name] = MapAggregator(self.steam_id, map_name)
        return self.map_data[map_name]
    
    def add_match(self, match_folder: Path, metadata: dict, player_data: dict):
        """Add a match to the aggregation."""
        map_name = metadata.get("map_name", "unknown")
        
        self.total_matches += 1
        self.total_kills += player_data.get("kills", 0)
        self.total_deaths += player_data.get("deaths", 0)
        self.total_assists += player_data.get("assists", 0)
        self.total_adr_sum += player_data.get("adr", 0)
        self.total_hs_sum += player_data.get("hs_percentage", 0)
        
        # Determine result
        final_score = metadata.get("final_score", "0-0")
        scores = final_score.split("-") if "-" in final_score else ["0", "0"]
        ct_score = int(scores[0].strip()) if scores[0].strip().isdigit() else 0
        t_score = int(scores[1].strip()) if len(scores) > 1 and scores[1].strip().isdigit() else 0
        
        winner = metadata.get("winner", "")
        user_team = player_data.get("team", "")
        is_victory = user_team == winner
        
        if user_team == "CT":
            team_score, opponent_score = ct_score, t_score
        else:
            team_score, opponent_score = t_score, ct_score
        
        if is_victory:
            self.wins += 1
            self.map_stats[map_name]["wins"] += 1
        else:
            self.losses += 1
            self.map_stats[map_name]["losses"] += 1
        
        self.map_stats[map_name]["matches"] += 1
        
        # Aim stats
        bp = player_data.get("body_part_hits", {})
        for part, count in bp.items():
            self.body_part_hits[part] += count
        
        self.shots_fired += player_data.get("shots_fired", 0)
        self.shots_hit += player_data.get("shots_hit", 0)
        
        ttd = player_data.get("time_to_damage_avg_ms", 0)
        if ttd > 0:
            self.ttd_sum += ttd
            self.ttd_count += 1
        
        cp = player_data.get("crosshair_placement_avg_error", 0)
        if cp > 0:
            self.crosshair_sum += cp
            self.crosshair_count += 1
        
        # Weapon stats
        for weapon, stats in player_data.get("weapon_stats", {}).items():
            ws = self.weapon_stats[weapon]
            ws["kills"] += stats.get("kills", 0)
            ws["headshots"] += stats.get("headshots", 0)
            ws["damage"] += stats.get("damage", 0)
            ws["shots_fired"] += stats.get("shots_fired", 0)
            ws["shots_hit"] += stats.get("shots_hit", 0)
        
        # Recent match info
        match_info = {
            "match_id": metadata.get("match_id", match_folder.name),
            "map": map_name,
            "date": metadata.get("date", ""),
            "result": "W" if is_victory else "L",
            "team_score": team_score,
            "opponent_score": opponent_score,
            "kills": player_data.get("kills", 0),
            "deaths": player_data.get("deaths", 0),
            "assists": player_data.get("assists", 0),
            "kd_ratio": player_data.get("kd_ratio", 0),
            "adr": player_data.get("adr", 0),
            "hs_percentage": player_data.get("hs_percentage", 0)
        }
        self.recent_matches.append(match_info)
    
    def build_aggregate(self) -> dict:
        """Build the final aggregate.json structure."""
        agg = create_empty_aggregate()
        
        agg["steam_id"] = self.steam_id
        agg["total_matches"] = self.total_matches
        
        # Stats
        agg["stats"]["total_kills"] = self.total_kills
        agg["stats"]["total_deaths"] = self.total_deaths
        agg["stats"]["avg_kd"] = round(self.total_kills / max(self.total_deaths, 1), 2)
        agg["stats"]["avg_adr"] = round(self.total_adr_sum / max(self.total_matches, 1), 1)
        agg["stats"]["avg_hs"] = round(self.total_hs_sum / max(self.total_matches, 1), 1)
        agg["stats"]["wins"] = self.wins
        agg["stats"]["losses"] = self.losses
        agg["stats"]["win_rate"] = round(self.wins / max(self.total_matches, 1) * 100, 1)
        
        # Aim stats
        agg["aim_stats"]["accuracy_overall"] = round(
            self.shots_hit / max(self.shots_fired, 1) * 100, 1
        )
        agg["aim_stats"]["time_to_damage_avg_ms"] = round(
            self.ttd_sum / max(self.ttd_count, 1), 1
        )
        agg["aim_stats"]["crosshair_placement_avg_error"] = round(
            self.crosshair_sum / max(self.crosshair_count, 1), 1
        )
        agg["aim_stats"]["body_part_hits"] = dict(self.body_part_hits)
        
        # Weapon stats (sorted by kills)
        weapon_list = []
        for weapon, stats in self.weapon_stats.items():
            if stats["kills"] > 0 or stats["shots_fired"] > 0:
                weapon_list.append({
                    "weapon": weapon,
                    "kills": stats["kills"],
                    "accuracy": round(
                        stats["shots_hit"] / max(stats["shots_fired"], 1) * 100, 1
                    ),
                    "hs_pct": round(
                        stats["headshots"] / max(stats["kills"], 1) * 100, 1
                    ),
                    "damage": stats["damage"]
                })
        weapon_list.sort(key=lambda x: x["kills"], reverse=True)
        agg["weapon_stats"] = weapon_list
        
        # Map stats (sorted by matches played)
        map_list = []
        for map_name, stats in self.map_stats.items():
            total = stats["wins"] + stats["losses"]
            map_list.append({
                "map": map_name,
                "wins": stats["wins"],
                "losses": stats["losses"],
                "win_rate": round(stats["wins"] / max(total, 1) * 100, 1)
            })
        map_list.sort(key=lambda x: x["wins"] + x["losses"], reverse=True)
        agg["map_stats"] = map_list
        
        # Recent matches (last 10, sorted by date)
        self.recent_matches.sort(key=lambda x: x.get("date", ""), reverse=True)
        agg["recent_matches"] = self.recent_matches[:10]
        
        return agg


class MapAggregator:
    """
    Aggregates map-specific data for Dashboard visualizations.
    """
    
    def __init__(self, steam_id: str, map_name: str):
        self.steam_id = str(steam_id)
        self.map_name = map_name
        self.matches_analyzed = 0
        
        # Callout stats
        self.callout_stats = defaultdict(lambda: {
            "kills": 0, "deaths": 0,
            "ct_kills": 0, "ct_deaths": 0,
            "t_kills": 0, "t_deaths": 0,
            "positions_x": [], "positions_y": [], "positions_z": [],
            "weapon_kills": defaultdict(int),
            "weapon_deaths": defaultdict(int),
            "opening_kills": 0, "opening_attempts": 0,
            "trade_kills": 0, "trade_deaths": 0,
            "smoke_kills": 0, "smoke_deaths": 0,
            "wallbang_kills": 0,
            "distances": [],
            "time_to_damages": [],
            "flash_deaths": 0, "total_deaths_for_flash": 0
        })
        
        # Side stats
        self.side_stats = {
            "CT": {"kills": 0, "deaths": 0, "headshots": 0, "adr_sum": 0.0, "adr_count": 0},
            "T": {"kills": 0, "deaths": 0, "headshots": 0, "adr_sum": 0.0, "adr_count": 0}
        }
        
        # Grenades
        self.grenades = {
            "by_type": {"smoke": [], "flash": [], "he": [], "molotov": []},
            "totals": {
                "flash": {"thrown": 0, "blinded": 0, "team_blinded": 0, "assists": 0},
                "he": {"thrown": 0, "damage": 0},
                "molotov": {"thrown": 0, "damage": 0},
                "smoke": {"thrown": 0}
            }
        }
        
        # Movement
        self.grid_counts = defaultdict(lambda: {
            "total": 0,
            "ct": 0,
            "t": 0,
            "game_x_sum": 0.0,
            "game_y_sum": 0.0,
            "z_sum": 0.0,
            "z_count": 0,
        })
        self.flow_transitions = defaultdict(lambda: {
            "count": 0,
            "ct": 0,
            "t": 0,
            "from_x_sum": 0.0,
            "from_y_sum": 0.0,
            "to_x_sum": 0.0,
            "to_y_sum": 0.0,
            "from_z_sum": 0.0,
            "to_z_sum": 0.0,
            "z_count": 0,
        })
        self.area_time = defaultdict(lambda: {
            "total": 0,
            "ct": 0,
            "t": 0,
            "game_x_sum": 0.0,
            "game_y_sum": 0.0,
            "z_sum": 0.0,
            "z_count": 0,
        })
        self.time_to_site = {"A": {"ct": [], "t": []}, "B": {"ct": [], "t": []}}
        self.total_rounds = 0
        self.total_samples = 0
    
    def process_combat(self, combat_data: dict, player_summary: dict):
        """Process the canonical combat web projection for callout stats."""
        for round_item in combat_data.get("rounds", []):
            duels = round_item.get("duels", [])
            
            # Find first kill tick for opening detection
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
                
                # User is attacker
                if str(attacker.get("steam_id")) == self.steam_id:
                    raw_callout = attacker.get("map_area", "Unknown")
                    callout = normalize_callout(raw_callout, self.map_name)
                    user_team = attacker.get("team", "")
                    weapon = attacker.get("weapon", "Unknown")
                    
                    # Side stats
                    if outcome == "kill" and user_team in ["CT", "T"]:
                        self.side_stats[user_team]["kills"] += 1
                        self.side_stats[user_team]["headshots"] += attacker.get("headshots", 0)
                    
                    if callout:
                        cs = self.callout_stats[callout]
                        
                        if outcome == "kill":
                            cs["kills"] += 1
                            if user_team == "CT":
                                cs["ct_kills"] += 1
                            elif user_team == "T":
                                cs["t_kills"] += 1
                            
                            cs["weapon_kills"][weapon] += 1
                            
                            if is_opening:
                                cs["opening_kills"] += 1
                                cs["opening_attempts"] += 1
                            if context.get("is_trade", False):
                                cs["trade_kills"] += 1
                            if context.get("through_smoke", False):
                                cs["smoke_kills"] += 1
                            if context.get("is_wallbang", False) or context.get("penetrated_objects", 0) > 0:
                                cs["wallbang_kills"] += 1
                        
                        if context.get("distance"):
                            cs["distances"].append(context["distance"])
                        if attacker.get("time_to_first_damage"):
                            cs["time_to_damages"].append(attacker["time_to_first_damage"])
                
                # User is victim
                for vic in victims:
                    if str(vic.get("steam_id")) == self.steam_id:
                        raw_callout = vic.get("map_area", "Unknown")
                        callout = normalize_callout(raw_callout, self.map_name)
                        user_team = vic.get("team", "")
                        
                        # Side stats
                        if vic.get("health_after", 0) == 0 and user_team in ["CT", "T"]:
                            self.side_stats[user_team]["deaths"] += 1
                        
                        if callout:
                            cs = self.callout_stats[callout]
                            
                            if vic.get("health_after", 0) == 0:
                                cs["deaths"] += 1
                                cs["total_deaths_for_flash"] += 1
                                
                                if user_team == "CT":
                                    cs["ct_deaths"] += 1
                                elif user_team == "T":
                                    cs["t_deaths"] += 1
                                
                                killer_weapon = attacker.get("weapon", "Unknown")
                                cs["weapon_deaths"][killer_weapon] += 1
                                
                                if is_opening:
                                    cs["opening_attempts"] += 1
                                if context.get("is_trade", False):
                                    cs["trade_deaths"] += 1
                                if context.get("through_smoke", False):
                                    cs["smoke_deaths"] += 1
                                
                                if vic.get("is_blind", False):
                                    cs["flash_deaths"] += 1
        
        # ADR from player summary
        if "ct_adr" in player_summary:
            self.side_stats["CT"]["adr_sum"] += player_summary.get("ct_adr", 0)
            self.side_stats["CT"]["adr_count"] += 1
        if "t_adr" in player_summary:
            self.side_stats["T"]["adr_sum"] += player_summary.get("t_adr", 0)
            self.side_stats["T"]["adr_count"] += 1
    
    def process_grenades(self, grenades_data: dict, player_name: str):
        """Process the canonical utility projection for grenade stats."""
        for round_item in grenades_data.get("rounds", []):
            for event in round_item.get("events", []):
                if event.get("thrower") != player_name:
                    continue
                
                g_type = event.get("type", "").lower()
                
                # Normalize types
                type_map = {
                    "flashbang": "flash",
                    "smoke grenade": "smoke",
                    "he grenade": "he",
                    "incendiary grenade": "molotov"
                }
                g_type = type_map.get(g_type, g_type)
                
                if g_type not in ["smoke", "flash", "he", "molotov"]:
                    continue
                
                # Position data
                sp = event.get("start_position", {})
                ep = event.get("end_position", {})
                
                grenade_entry = {
                    "game_x": sp.get("x", 0),
                    "game_y": sp.get("y", 0),
                    "end_x": ep.get("x", 0),
                    "end_y": ep.get("y", 0),
                    "end_z": ep.get("z", 0),
                    "side": event.get("thrower_side", "unknown"),
                    "land_area": event.get("land_area", ""),
                    "damage_dealt": event.get("damage_dealt", 0),
                    "enemies_blinded": event.get("enemies_blinded", 0),
                    "allies_blinded": event.get("allies_blinded", 0)
                }
                
                self.grenades["by_type"][g_type].append(grenade_entry)
                
                # Totals
                if g_type == "flash":
                    self.grenades["totals"]["flash"]["thrown"] += 1
                    self.grenades["totals"]["flash"]["blinded"] += event.get("enemies_blinded", 0)
                    self.grenades["totals"]["flash"]["team_blinded"] += event.get("allies_blinded", 0)
                elif g_type == "he":
                    self.grenades["totals"]["he"]["thrown"] += 1
                    self.grenades["totals"]["he"]["damage"] += event.get("damage_dealt", 0)
                elif g_type == "molotov":
                    self.grenades["totals"]["molotov"]["thrown"] += 1
                    self.grenades["totals"]["molotov"]["damage"] += event.get("damage_dealt", 0)
                elif g_type == "smoke":
                    self.grenades["totals"]["smoke"]["thrown"] += 1
    
    def process_tracking(self, tracking_data: dict):
        """Process canonical state samples for movement and callout positions."""
        GRID_SIZE = 20
        
        for round_item in tracking_data.get("rounds", []):
            self.total_rounds += 1
            prev_area = None
            prev_position = None
            prev_z = None
            prev_level = None
            
            for tick_data in round_item.get("ticks", []):
                for player in tick_data.get("players", []):
                    player_id = player.get("player_steam_id")
                    
                    if str(player_id) != self.steam_id:
                        continue
                    
                    if not player.get("is_alive", False):
                        continue
                    
                    pos = player.get("pos", {})
                    area = player.get("area_name", "")
                    team = player.get("team", "").lower()
                    
                    if not pos:
                        continue
                    
                    game_x = pos.get("x", 0)
                    game_y = pos.get("y", 0)
                    game_z = pos.get("z")
                    team_key = team if team in {"ct", "t"} else None
                    radar_position = game_to_radar_percent(
                        game_x,
                        game_y,
                        self.map_name,
                    )
                    
                    # Keep vertically overlapping Nuke cells separate.
                    level_key = "all"
                    if self.map_name == "de_nuke":
                        if game_z is None:
                            level_key = "unknown"
                        else:
                            level_key = "upper" if game_z >= NUKE_Z_THRESHOLD else "lower"

                    grid_x = max(0, min(
                        int(radar_position["x"] / 100 * GRID_SIZE),
                        GRID_SIZE - 1,
                    ))
                    grid_y = max(0, min(
                        int(radar_position["y"] / 100 * GRID_SIZE),
                        GRID_SIZE - 1,
                    ))
                    
                    key = (grid_x, grid_y, level_key)
                    self.grid_counts[key]["total"] += 1
                    self.grid_counts[key]["game_x_sum"] += game_x
                    self.grid_counts[key]["game_y_sum"] += game_y
                    if game_z is not None:
                        self.grid_counts[key]["z_sum"] += game_z
                        self.grid_counts[key]["z_count"] += 1
                    
                    if team_key:
                        self.grid_counts[key][team_key] += 1
                    
                    self.total_samples += 1

                    normalized_area = normalize_callout(area, self.map_name) or area.strip()

                    if normalized_area:
                        area_key = (normalized_area, level_key)
                        area_stats = self.area_time[area_key]
                        area_stats["total"] += 1
                        area_stats["game_x_sum"] += game_x
                        area_stats["game_y_sum"] += game_y
                        if team_key:
                            area_stats[team_key] += 1
                        if game_z is not None:
                            area_stats["z_sum"] += game_z
                            area_stats["z_count"] += 1
                    
                    # Flow transitions
                    if (
                        normalized_area
                        and prev_area
                        and (
                            normalized_area != prev_area
                            or level_key != prev_level
                        )
                        and prev_position
                    ):
                        transition_key = (
                            prev_area,
                            prev_level,
                            normalized_area,
                            level_key,
                        )
                        transition = self.flow_transitions[transition_key]
                        transition["count"] += 1
                        transition["from_x_sum"] += prev_position["x"]
                        transition["from_y_sum"] += prev_position["y"]
                        transition["to_x_sum"] += radar_position["x"]
                        transition["to_y_sum"] += radar_position["y"]
                        if prev_z is not None and game_z is not None:
                            transition["from_z_sum"] += prev_z
                            transition["to_z_sum"] += game_z
                            transition["z_count"] += 1
                        if team_key:
                            transition[team_key] += 1
                    
                    # Callout positions
                    if normalized_area and normalized_area in self.callout_stats:
                        self.callout_stats[normalized_area]["positions_x"].append(game_x)
                        self.callout_stats[normalized_area]["positions_y"].append(game_y)
                        if game_z is not None:
                            self.callout_stats[normalized_area]["positions_z"].append(game_z)

                    if normalized_area:
                        prev_area = normalized_area
                        prev_position = radar_position
                        prev_z = game_z
                        prev_level = level_key
    
    def build_map_data(self) -> dict:
        """Build the final map aggregate structure."""
        data = create_empty_map_aggregate(self.map_name)
        
        data["steam_id"] = self.steam_id
        data["matches_analyzed"] = self.matches_analyzed
        
        # Build callout stats
        heatmap_data = []
        
        for callout, stats in self.callout_stats.items():
            kills = stats["kills"]
            deaths = stats["deaths"]
            total_duels = kills + deaths
            
            if total_duels == 0:
                continue
            
            kd = round(kills / max(deaths, 1), 2)
            win_rate = round(kills / total_duels * 100, 1)
            rating = "good" if win_rate >= 55 else "bad" if win_rate <= 45 else "neutral"
            
            # Position
            position = None
            avg_z = None
            if stats["positions_x"] and stats["positions_y"]:
                avg_x = sum(stats["positions_x"]) / len(stats["positions_x"])
                avg_y = sum(stats["positions_y"]) / len(stats["positions_y"])
                position = game_to_radar_percent(avg_x, avg_y, self.map_name)
                
                if stats["positions_z"]:
                    avg_z = sum(stats["positions_z"]) / len(stats["positions_z"])
            
            # Weapon stats
            weapon_stats = []
            all_weapons = set(stats["weapon_kills"].keys()) | set(stats["weapon_deaths"].keys())
            for weapon in all_weapons:
                w_kills = stats["weapon_kills"].get(weapon, 0)
                w_deaths = stats["weapon_deaths"].get(weapon, 0)
                if w_kills + w_deaths > 0:
                    weapon_stats.append({
                        "weapon": weapon,
                        "kills": w_kills,
                        "deaths": w_deaths,
                        "kd": round(w_kills / max(w_deaths, 1), 2)
                    })
            weapon_stats.sort(key=lambda w: w["kills"] + w["deaths"], reverse=True)
            weapon_stats = weapon_stats[:5]
            
            # Context stats
            context_stats = {
                "opening_kills": stats["opening_kills"],
                "opening_attempts": stats["opening_attempts"],
                "trade_kills": stats["trade_kills"],
                "trade_deaths": stats["trade_deaths"],
                "smoke_kills": stats["smoke_kills"],
                "smoke_deaths": stats["smoke_deaths"],
                "wallbang_kills": stats["wallbang_kills"]
            }
            
            # Metrics
            avg_distance = round(sum(stats["distances"]) / len(stats["distances"]), 0) if stats["distances"] else None
            avg_ttd = round(sum(stats["time_to_damages"]) / len(stats["time_to_damages"]), 0) if stats["time_to_damages"] else None
            flash_death_pct = round(stats["flash_deaths"] / stats["total_deaths_for_flash"] * 100, 1) if stats["total_deaths_for_flash"] > 0 else 0.0
            
            data["callout_stats"][callout] = {
                "kills": kills,
                "deaths": deaths,
                "kd": kd,
                "win_rate": win_rate,
                "rating": rating,
                "position": position,
                "avg_z": avg_z,
                "sample_size": total_duels,
                "ct_t_split": {
                    "ct_kills": stats["ct_kills"],
                    "ct_deaths": stats["ct_deaths"],
                    "t_kills": stats["t_kills"],
                    "t_deaths": stats["t_deaths"]
                },
                "weapon_stats": weapon_stats,
                "context_stats": context_stats,
                "avg_distance": avg_distance,
                "avg_time_to_damage": avg_ttd,
                "flash_death_pct": flash_death_pct
            }
            
            # Heatmap data
            if position:
                for _ in range(kills):
                    point = {"x": position["x"], "y": position["y"], "type": "kill", "callout": callout}
                    if avg_z is not None:
                        point["avg_z"] = avg_z
                    heatmap_data.append(point)
                
                for _ in range(deaths):
                    point = {"x": position["x"], "y": position["y"], "type": "death", "callout": callout}
                    if avg_z is not None:
                        point["avg_z"] = avg_z
                    heatmap_data.append(point)
        
        data["heatmap_data"] = heatmap_data
        
        # Side stats
        for side in ["CT", "T"]:
            s = self.side_stats[side]
            kills = s["kills"]
            deaths = s["deaths"]
            
            data["side_stats"][side] = {
                "kills": kills,
                "deaths": deaths,
                "kd": round(kills / max(deaths, 1), 2),
                "adr": round(s["adr_sum"] / max(s["adr_count"], 1), 1),
                "hs_pct": round(s["headshots"] / max(kills, 1) * 100, 1)
            }
        
        # Grenades
        data["grenades"]["by_type"] = self.grenades["by_type"]
        
        totals = self.grenades["totals"]
        data["grenades"]["summary"] = {
            "flash": {
                "thrown": totals["flash"]["thrown"],
                "avg_blinded": round(totals["flash"]["blinded"] / max(totals["flash"]["thrown"], 1), 1),
                "assists": totals["flash"]["assists"]
            },
            "he": {
                "thrown": totals["he"]["thrown"],
                "avg_damage": round(totals["he"]["damage"] / max(totals["he"]["thrown"], 1), 1)
            },
            "molotov": {
                "thrown": totals["molotov"]["thrown"],
                "avg_damage": round(totals["molotov"]["damage"] / max(totals["molotov"]["thrown"], 1), 1)
            },
            "smoke": {
                "thrown": totals["smoke"]["thrown"]
            }
        }
        
        # Movement heatmap
        heatmap_grid = []
        for (grid_x, grid_y, level), counts in self.grid_counts.items():
            if counts["total"] > 0:
                cell = {
                    "grid_x": grid_x,
                    "grid_y": grid_y,
                    "total": counts["total"],
                    "ct": counts["ct"],
                    "t": counts["t"],
                    "game_x": counts["game_x_sum"] / counts["total"],
                    "game_y": counts["game_y_sum"] / counts["total"],
                    "level": level,
                }
                if counts["z_count"] > 0:
                    cell["avg_z"] = counts["z_sum"] / counts["z_count"]
                heatmap_grid.append(cell)
        heatmap_grid.sort(key=lambda x: x["total"], reverse=True)
        data["movement"]["heatmap_grid"] = heatmap_grid
        
        # Flow lines (top 20)
        flow_lines = []
        for (
            from_area,
            from_level,
            to_area,
            to_level,
        ), counts in self.flow_transitions.items():
            if counts["count"] >= 3:  # Min threshold
                route = {
                    "from": from_area,
                    "to": to_area,
                    "from_level": from_level,
                    "to_level": to_level,
                    "count": counts["count"],
                    "ct": counts["ct"],
                    "t": counts["t"],
                    "from_x": counts["from_x_sum"] / counts["count"],
                    "from_y": counts["from_y_sum"] / counts["count"],
                    "to_x": counts["to_x_sum"] / counts["count"],
                    "to_y": counts["to_y_sum"] / counts["count"],
                }
                if counts["z_count"] > 0:
                    route["from_avg_z"] = counts["from_z_sum"] / counts["z_count"]
                    route["to_avg_z"] = counts["to_z_sum"] / counts["z_count"]
                flow_lines.append(route)
        flow_lines.sort(key=lambda x: x["count"], reverse=True)
        data["movement"]["flow_lines"] = flow_lines[:20]

        total_area_samples = sum(
            stats["total"] for stats in self.area_time.values()
        )
        top_positions = []
        for (area, level), stats in sorted(
            self.area_time.items(),
            key=lambda item: item[1]["total"],
            reverse=True,
        ):
            position = game_to_radar_percent(
                stats["game_x_sum"] / stats["total"],
                stats["game_y_sum"] / stats["total"],
                self.map_name,
            )
            top_position = {
                "area": area,
                "level": level,
                "time_percent": round(
                    stats["total"] / max(total_area_samples, 1) * 100,
                    1,
                ),
                "ct_percent": round(
                    stats["ct"] / max(stats["total"], 1) * 100,
                    1,
                ),
                "sample_count": stats["total"],
                "position": position,
            }
            if stats["z_count"] > 0:
                top_position["avg_z"] = stats["z_sum"] / stats["z_count"]
            top_positions.append(top_position)

        data["movement"]["metrics"]["top_positions"] = top_positions
        data["movement"]["metrics"]["total_rounds"] = self.total_rounds
        data["movement"]["metrics"]["total_samples"] = self.total_samples
        
        return data


def project_tracking(match: CanonicalMatch) -> dict[str, list[dict[str, Any]]]:
    """Project sharded canonical states into the aggregate builder's tick shape."""
    rounds: dict[int, dict[int, list[dict[str, Any]]]] = defaultdict(
        lambda: defaultdict(list),
    )
    for state in match.iter_player_states():
        round_number = int(state.get("round_number") or 0)
        tick = int(state.get("tick") or 0)
        player_id = str(state.get("player_id") or "").removeprefix("steam:")
        rounds[round_number][tick].append(
            {
                "tick": tick,
                "player_steam_id": player_id,
                "team": str(state.get("side") or "").upper(),
                "pos": dict(state.get("position") or {}),
                "area_name": state.get("area", ""),
                "view_yaw": state.get("view_yaw_deg", 0),
                "view_pitch": state.get("view_pitch_deg", 0),
                "vel_len": state.get("velocity_world_units_per_second", 0),
                "is_walking": bool(state.get("is_walking")),
                "is_ducking": bool(state.get("is_ducking")),
                "active_weapon": state.get("active_weapon", ""),
                "has_c4": bool(state.get("has_c4")),
                "health": state.get("health", 0),
                "armor": state.get("armor", 0),
                "nearby_teammates": state.get("nearby_teammates", 0),
                "is_alive": bool(state.get("is_alive")),
                "round_time_remaining": (
                    float(state.get("round_time_remaining_ms") or 0) / 1000
                ),
            },
        )

    return {
        "rounds": [
            {
                "round": round_number,
                "ticks": [
                    {"tick": tick, "players": ticks[tick]}
                    for tick in sorted(ticks)
                ],
            }
            for round_number, ticks in sorted(rounds.items())
        ],
    }


def migrate():
    """Main migration function."""
    logger.info("=" * 60)
    logger.info("Starting migration to user aggregate files")
    logger.info("=" * 60)
    
    if not EXPORTS_PATH.exists():
        logger.error(f"Exports path does not exist: {EXPORTS_PATH}")
        return
    
    # Create users directory
    USERS_PATH.mkdir(parents=True, exist_ok=True)
    
    # Collect data by user
    user_aggregators = {}  # {steam_id: MigrationAggregator}
    
    match_folders = [
        f for f in EXPORTS_PATH.iterdir()
        if f.is_dir() and f.name.startswith("match_")
    ]
    
    logger.info(f"Found {len(match_folders)} match folders to process")
    
    for i, match_folder in enumerate(match_folders, 1):
        logger.info(f"Processing [{i}/{len(match_folders)}] {match_folder.name}")
        
        try:
            match = CanonicalMatch(match_folder)
            if not match.has_canonical_bundle():
                logger.warning("  - No canonical bundle, skipping")
                continue
            metadata = project_match_metadata(match)
            map_name = metadata.get("map_name", "unknown")
            players_data = project_player_summary(match)
            players = players_data.get("players", [])
            combat_data = project_combat(match)
            grenades_data = project_utility(match)
            tracking_data = project_tracking(match)
            
            # Process each player
            for player in players:
                steam_id = str(player.get("steam_id", ""))
                player_name = player.get("name", "")
                
                if not steam_id:
                    continue
                
                # Get or create aggregator
                if steam_id not in user_aggregators:
                    user_aggregators[steam_id] = MigrationAggregator(steam_id)
                
                aggregator = user_aggregators[steam_id]
                
                # Add match to global stats
                aggregator.add_match(match_folder, metadata, player)
                
                # Add match to history
                add_match_to_history(steam_id, {
                    "match_id": metadata.get("match_id", match_folder.name),
                    "map_name": map_name,
                    "date": metadata.get("date", "")
                })
                
                # Process map-specific data
                map_agg = aggregator.get_map_aggregator(map_name)
                map_agg.matches_analyzed += 1
                
                if combat_data:
                    map_agg.process_combat(combat_data, player)
                
                if grenades_data:
                    map_agg.process_grenades(grenades_data, player_name)
                
                if tracking_data:
                    map_agg.process_tracking(tracking_data)
        
        except Exception as e:
            logger.error(f"  - Error processing {match_folder.name}: {e}")
            import traceback
            traceback.print_exc()
            continue
    
    # Save all aggregates
    logger.info("=" * 60)
    logger.info(f"Saving aggregates for {len(user_aggregators)} users")
    logger.info("=" * 60)
    
    for steam_id, aggregator in user_aggregators.items():
        logger.info(f"Saving aggregates for {steam_id}")
        
        # Save global aggregate
        aggregate = aggregator.build_aggregate()
        save_user_aggregate(steam_id, aggregate)
        logger.info(f"  - aggregate.json: {aggregator.total_matches} matches")
        
        # Save map-specific data
        for map_name, map_agg in aggregator.map_data.items():
            map_data = map_agg.build_map_data()
            save_user_map_data(steam_id, map_name, map_data)
            logger.info(f"  - maps/{map_name}.json: {map_agg.matches_analyzed} matches")
    
    logger.info("=" * 60)
    logger.info("Migration complete!")
    logger.info("=" * 60)


if __name__ == "__main__":
    migrate()
