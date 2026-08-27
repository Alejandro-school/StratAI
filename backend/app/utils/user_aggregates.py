# backend/app/utils/user_aggregates.py
# =====================================
# Utility module for managing user aggregate data files.
# These pre-calculated files enable O(1) lookups instead of O(n) folder scanning.
#
# Directory Structure:
#   data/users/{steam_id}/
#   ├── aggregate.json          # Global stats for Performance page
#   ├── match_history.json      # List of analyzed match IDs
#   └── maps/
#       ├── de_dust2.json       # All Dashboard data for this map
#       ├── de_mirage.json
#       └── ...

import hashlib
import json
import logging
import os
import tempfile
from datetime import datetime
from pathlib import Path
from typing import Any, Optional

from ..matches.canonical_repository import iter_canonical_matches
from ..matches.match_web_projection import (
    project_match_metadata,
    project_player_summary,
)

# Base path for user data (relative to this file's location)
BASE_DATA_PATH = Path(__file__).parent.parent.parent / "data"
USERS_PATH = BASE_DATA_PATH / "users"
EXPORTS_PATH = BASE_DATA_PATH / "exports"
USER_MAP_SCHEMA_VERSION = 3

# Competitive CS2 maps
COMPETITIVE_MAPS = [
    "de_dust2", "de_mirage", "de_inferno", "de_nuke",
    "de_overpass", "de_anubis", "de_ancient", "de_vertigo"
]


def _write_json_atomic(path: Path, data: Any) -> None:
    """Write JSON through a temporary file on the destination filesystem."""
    descriptor, temporary_name = tempfile.mkstemp(
        dir=path.parent,
        prefix=f".{path.name}.",
        suffix=".tmp",
    )
    temporary_path = Path(temporary_name)

    try:
        with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as file:
            descriptor = -1
            json.dump(data, file, indent=2, ensure_ascii=False)
            file.flush()
            os.fsync(file.fileno())
        os.replace(temporary_path, path)
    except Exception:
        if descriptor >= 0:
            os.close(descriptor)
        temporary_path.unlink(missing_ok=True)
        raise


def build_user_map_source_signature(steam_id: str, map_name: str) -> dict:
    """Describe the exports that contribute to one user's map aggregate."""
    match_ids = []
    latest_source_mtime_ns = 0
    steam_id_str = str(steam_id)

    for match_export in iter_canonical_matches(EXPORTS_PATH):
        metadata = project_match_metadata(match_export)
        players_summary = project_player_summary(match_export)
        if str(metadata.get("map_name", "")) != str(map_name):
            continue

        players = players_summary.get("players", [])
        if not any(
            isinstance(player, dict)
            and str(player.get("steam_id", "")) == steam_id_str
            for player in players
        ):
            continue

        summary_match_id = players_summary.get("match_id")
        match_id = metadata.get("match_id") or summary_match_id
        match_ids.append(str(match_id or match_export.match_id))
        latest_source_mtime_ns = max(
            latest_source_mtime_ns,
            match_export.manifest_mtime_ns(),
        )

    sorted_match_ids = sorted(match_ids)
    match_ids_sha256 = hashlib.sha256(
        "\n".join(sorted_match_ids).encode("utf-8")
    ).hexdigest()
    return {
        "match_count": len(sorted_match_ids),
        "latest_source_mtime_ns": latest_source_mtime_ns,
        "match_ids_sha256": match_ids_sha256,
    }


def attach_user_map_source_snapshot(
    steam_id: str,
    map_name: str,
    data: dict,
) -> dict:
    """Return map data carrying the current source freshness contract."""
    return {
        **data,
        "schema_version": USER_MAP_SCHEMA_VERSION,
        "source_snapshot": build_user_map_source_signature(steam_id, map_name),
    }


def is_user_map_data_fresh(
    steam_id: str,
    map_name: str,
    data: Optional[dict] = None,
) -> bool:
    """Check whether saved map data still represents the current exports."""
    stored_data = data if data is not None else load_user_map_data(
        steam_id,
        map_name,
    )
    if not isinstance(stored_data, dict):
        return False
    if stored_data.get("schema_version") != USER_MAP_SCHEMA_VERSION:
        return False

    return stored_data.get("source_snapshot") == (
        build_user_map_source_signature(steam_id, map_name)
    )


# =============================================================================
# DIRECTORY MANAGEMENT
# =============================================================================

def ensure_user_directory(steam_id: str) -> Path:
    """
    Ensure the user's data directory exists.
    
    Creates:
        data/users/{steam_id}/
        data/users/{steam_id}/maps/
    
    Returns:
        Path to the user's directory
    """
    user_dir = USERS_PATH / str(steam_id)
    maps_dir = user_dir / "maps"
    
    user_dir.mkdir(parents=True, exist_ok=True)
    maps_dir.mkdir(exist_ok=True)
    
    return user_dir


def get_user_aggregate_path(steam_id: str) -> Path:
    """Get path to user's global aggregate.json file."""
    return USERS_PATH / str(steam_id) / "aggregate.json"


def get_user_map_path(steam_id: str, map_name: str) -> Path:
    """Get path to user's map-specific aggregate file."""
    return USERS_PATH / str(steam_id) / "maps" / f"{map_name}.json"


def get_user_match_history_path(steam_id: str) -> Path:
    """Get path to user's match history file."""
    return USERS_PATH / str(steam_id) / "match_history.json"


# =============================================================================
# AGGREGATE FILE OPERATIONS
# =============================================================================

def load_user_aggregate(steam_id: str) -> Optional[dict]:
    """
    Load user's global aggregate data.
    
    Returns:
        Dict with aggregate data, or None if file doesn't exist
    """
    path = get_user_aggregate_path(steam_id)
    
    if not path.exists():
        return None
    
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception as e:
        logging.error(f"[user_aggregates] Error loading aggregate for {steam_id}: {e}")
        return None


def save_user_aggregate(steam_id: str, data: dict) -> bool:
    """
    Save user's global aggregate data.
    
    Args:
        steam_id: User's Steam ID
        data: Aggregate data to save
    
    Returns:
        True if successful, False otherwise
    """
    ensure_user_directory(steam_id)
    path = get_user_aggregate_path(steam_id)
    payload = {
        **data,
        "last_updated": datetime.now().isoformat(),
        "steam_id": str(steam_id),
    }

    try:
        _write_json_atomic(path, payload)
        return True
    except Exception as e:
        logging.error(f"[user_aggregates] Error saving aggregate for {steam_id}: {e}")
        return False


def load_user_map_data(steam_id: str, map_name: str) -> Optional[dict]:
    """
    Load user's map-specific aggregate data.
    
    Returns:
        Dict with map data, or None if file doesn't exist
    """
    path = get_user_map_path(steam_id, map_name)
    
    if not path.exists():
        return None
    
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception as e:
        logging.error(f"[user_aggregates] Error loading map data for {steam_id}/{map_name}: {e}")
        return None


def save_user_map_data(steam_id: str, map_name: str, data: dict) -> bool:
    """
    Save user's map-specific aggregate data.
    
    Args:
        steam_id: User's Steam ID
        map_name: Map name (e.g., 'de_dust2')
        data: Map aggregate data to save
    
    Returns:
        True if successful, False otherwise
    """
    ensure_user_directory(steam_id)
    path = get_user_map_path(steam_id, map_name)

    try:
        payload = attach_user_map_source_snapshot(
            steam_id,
            map_name,
            {
                **data,
                "last_updated": datetime.now().isoformat(),
                "steam_id": str(steam_id),
                "map_name": map_name,
            },
        )
        _write_json_atomic(path, payload)
        return True
    except Exception as e:
        logging.error(f"[user_aggregates] Error saving map data for {steam_id}/{map_name}: {e}")
        return False


# =============================================================================
# MATCH HISTORY MANAGEMENT
# =============================================================================

def load_match_history(steam_id: str) -> list:
    """Load user's match history list."""
    path = get_user_match_history_path(steam_id)
    
    if not path.exists():
        return []
    
    try:
        with open(path, 'r', encoding='utf-8') as f:
            data = json.load(f)
            return data.get("matches", [])
    except Exception as e:
        logging.error(f"[user_aggregates] Error loading match history for {steam_id}: {e}")
        return []


def add_match_to_history(steam_id: str, match_info: dict) -> bool:
    """
    Add a match to user's history.
    
    Args:
        steam_id: User's Steam ID
        match_info: Dict with match_id, map_name, date, result, etc.
    
    Returns:
        True if successful (or already exists), False on error
    """
    ensure_user_directory(steam_id)
    path = get_user_match_history_path(steam_id)
    
    # Load existing
    matches = load_match_history(steam_id)
    
    # Check if already exists
    match_id = match_info.get("match_id")
    if any(m.get("match_id") == match_id for m in matches):
        return True  # Already exists, skip
    
    # Add new match
    matches.append(match_info)
    
    # Sort by date (newest first)
    matches.sort(key=lambda x: x.get("date", ""), reverse=True)
    
    payload = {
        "steam_id": str(steam_id),
        "last_updated": datetime.now().isoformat(),
        "total_matches": len(matches),
        "matches": matches,
    }

    try:
        _write_json_atomic(path, payload)
        return True
    except Exception as e:
        logging.error(f"[user_aggregates] Error saving match history for {steam_id}: {e}")
        return False


# =============================================================================
# DATA STRUCTURE TEMPLATES
# =============================================================================

def create_empty_aggregate() -> dict:
    """
    Create an empty aggregate structure for a new user.
    
    This structure mirrors what get_dashboard_stats returns.
    """
    return {
        "steam_id": "",
        "last_updated": "",
        "total_matches": 0,
        "stats": {
            "total_kills": 0,
            "total_deaths": 0,
            "avg_kd": 0.0,
            "avg_adr": 0.0,
            "avg_hs": 0.0,
            "win_rate": 0.0,
            "wins": 0,
            "losses": 0
        },
        "aim_stats": {
            "accuracy_overall": 0.0,
            "time_to_damage_avg_ms": 0.0,
            "crosshair_placement_avg_error": 0.0,
            "body_part_hits": {
                "head": 0, "chest": 0, "stomach": 0,
                "left_arm": 0, "right_arm": 0,
                "left_leg": 0, "right_leg": 0, "generic": 0
            }
        },
        "weapon_stats": [],  # [{weapon, kills, accuracy, hs_pct, damage}]
        "map_stats": [],     # [{map, wins, losses, win_rate}]
        "recent_matches": [] # Last 10 matches for quick display
    }


def create_empty_map_aggregate(map_name: str) -> dict:
    """
    Create an empty map-specific aggregate structure.
    
    This structure contains all data needed for Dashboard visualizations.
    """
    return {
        "schema_version": USER_MAP_SCHEMA_VERSION,
        "steam_id": "",
        "map_name": map_name,
        "last_updated": "",
        "matches_analyzed": 0,
        
        # Callout stats for Duels tab
        "callout_stats": {},  # {callout_name: {kills, deaths, kd, win_rate, position, ...}}
        
        # Heatmap data for kill/death visualization
        "heatmap_data": [],   # [{x, y, type, callout, avg_z}]
        
        # Side stats (CT/T performance)
        "side_stats": {
            "CT": {"kills": 0, "deaths": 0, "kd": 0.0, "adr": 0.0, "hs_pct": 0.0},
            "T": {"kills": 0, "deaths": 0, "kd": 0.0, "adr": 0.0, "hs_pct": 0.0}
        },
        
        # Grenade data for Grenades tab
        "grenades": {
            "by_type": {
                "smoke": [],
                "flash": [],
                "he": [],
                "molotov": []
            },
            "summary": {
                "flash": {"thrown": 0, "avg_blinded": 0, "assists": 0},
                "he": {"thrown": 0, "avg_damage": 0},
                "molotov": {"thrown": 0, "avg_damage": 0},
                "smoke": {"thrown": 0}
            },
            "insights": []
        },
        
        # Movement data for HeatMap tab
        "movement": {
            "heatmap_grid": [],  # [{grid_x, grid_y, total, ct, t, game_x, game_y}]
            "flow_lines": [],    # [{from, to, count, ct, t}]
            "metrics": {
                "avg_time_to_a": {"ct": None, "t": None},
                "avg_time_to_b": {"ct": None, "t": None},
                "total_rounds": 0,
                "total_samples": 0
            }
        }
    }


# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

def user_has_aggregates(steam_id: str) -> bool:
    """Check if user has pre-calculated aggregate files."""
    aggregate_path = get_user_aggregate_path(steam_id)
    return aggregate_path.exists()


def user_has_map_data(steam_id: str, map_name: str) -> bool:
    """Check if user has pre-calculated data for a specific map."""
    map_path = get_user_map_path(steam_id, map_name)
    return map_path.exists()


def get_all_user_steam_ids() -> list:
    """Get list of all Steam IDs that have aggregate data."""
    if not USERS_PATH.exists():
        return []
    
    return [
        d.name for d in USERS_PATH.iterdir()
        if d.is_dir() and d.name.isdigit()
    ]


def delete_user_aggregates(steam_id: str) -> bool:
    """
    Delete all aggregate data for a user.
    Useful for forcing regeneration.
    """
    import shutil
    user_dir = USERS_PATH / str(steam_id)
    
    if not user_dir.exists():
        return True
    
    try:
        shutil.rmtree(user_dir)
        return True
    except Exception as e:
        logging.error(f"[user_aggregates] Error deleting aggregates for {steam_id}: {e}")
        return False
