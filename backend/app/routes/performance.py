# backend/app/routes/performance.py
# -------------------------------
# Ruta para agregar estadísticas de rendimiento global.
# Proyecta los resÃºmenes canÃ³nicos y genera un perfil unificado.

import logging
from pathlib import Path

from fastapi import APIRouter, Depends, Query

from ..auth.dependencies import SteamUser, require_steam_user
from ..matches.canonical_repository import iter_canonical_matches
from ..matches.match_web_projection import project_player_summary
from ..utils.performance_aggregator import build_performance_overview

router = APIRouter()
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

EXPORTS_PATH = Path(__file__).resolve().parents[2] / "data" / "exports"


def _build_player_index() -> list[dict]:
    """Scan all match exports and build a deduplicated player name→steam_id index."""
    seen: dict[str, dict] = {}

    if not EXPORTS_PATH.exists():
        return []

    for match_export in iter_canonical_matches(EXPORTS_PATH):
        data = project_player_summary(match_export)
        players = data.get("players", [])
        for player in players:
            sid = str(player.get("steam_id", ""))
            name = str(player.get("name", ""))
            if not sid or not name:
                continue
            if sid not in seen:
                seen[sid] = {"steam_id": sid, "name": name, "matches": 0}
            seen[sid]["matches"] += 1
            seen[sid]["name"] = name  # keep latest name

    return sorted(seen.values(), key=lambda p: p["matches"], reverse=True)

@router.get("/steam/performance-overview")
async def get_performance_overview(
    force_refresh: bool = False,
    map_name: str | None = Query(default=None, max_length=64),
    limit: int | None = Query(default=None, ge=1, le=200),
    user: SteamUser = Depends(require_steam_user),
):
    """Devuelve el perfil completo y agregado de rendimiento del jugador."""
    return build_performance_overview(user.steam_id, map_name=map_name, limit=limit)


@router.get("/steam/performance-stats")
async def get_performance_stats(
    force_refresh: bool = False,
    user: SteamUser = Depends(require_steam_user),
):
    """Compat endpoint: conserva la ruta histórica con un payload resumido."""
    full_payload = await get_performance_overview(
        force_refresh=force_refresh,
        map_name=None,
        limit=None,
        user=user,
    )

    overview = full_payload.get("overview", {})
    aim = full_payload.get("aim", {})
    maps = full_payload.get("maps", [])
    weapons = full_payload.get("weapons", [])

    return {
        "steam_id": full_payload.get("steam_id"),
        "matches_analyzed": overview.get("total_matches", 0),
        "overall": {
            "kd_ratio": overview.get("kd_ratio", 0.0),
            "hs_percent": overview.get("hs_pct", 0.0),
            "accuracy": aim.get("accuracy_overall", 0.0),
            "total_kills": overview.get("kills", 0),
            "total_damage": overview.get("total_damage", 0),
        },
        "aim": {
            "time_to_damage_ms": aim.get("time_to_damage_avg_ms", 0.0),
            "crosshair_placement_error": aim.get("crosshair_placement_avg_error", 0.0),
            "reaction_rating": "Good" if aim.get("time_to_damage_avg_ms", 0.0) < 500 else "Average",
        },
        "maps": maps,
        "weapons": weapons[:5],
    }


@router.get("/steam/player-search")
async def player_search(
    q: str = Query("", min_length=0, max_length=100),
    _user: SteamUser = Depends(require_steam_user),
):
    """Search players by name across all match exports."""
    index = _build_player_index()

    if not q.strip():
        return {"players": index[:30]}

    query_lower = q.strip().lower()
    results = [p for p in index if query_lower in p["name"].lower()]
    return {"players": results[:30]}


@router.get("/steam/player-stats/{steam_id}")
async def get_player_stats(
    steam_id: str,
    map_name: str | None = Query(default=None, max_length=64),
    limit: int | None = Query(default=None, ge=1, le=200),
    _user: SteamUser = Depends(require_steam_user),
):
    """Return the performance overview for any player by steam_id."""
    import re
    if not re.match(r"^7656\d{13}$", steam_id):
        return {"error": "Invalid steam_id format"}

    data = build_performance_overview(steam_id, map_name=map_name, limit=limit)
    overview = data.get("overview", {})

    if not overview.get("total_matches"):
        return {"error": "No data found for this player", "steam_id": steam_id}

    return data
