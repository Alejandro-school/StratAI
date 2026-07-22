# backend/app/routes/performance.py
# -------------------------------
# Ruta para agregar estadísticas de rendimiento global.
# Lee múltiples players_summary.json y genera un perfil unificado.

import json
import logging
from pathlib import Path

from fastapi import APIRouter, Request, Query

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

    for match_dir in EXPORTS_PATH.iterdir():
        if not match_dir.is_dir() or not match_dir.name.startswith("match_"):
            continue

        summary_path = match_dir / "players_summary.json"
        if not summary_path.exists():
            continue

        try:
            with summary_path.open("r", encoding="utf-8") as f:
                data = json.load(f)
        except Exception:
            continue

        players = data.get("players", [])
        if not isinstance(players, list):
            continue

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
async def get_performance_overview(request: Request, force_refresh: bool = False):
    """Devuelve el perfil completo y agregado de rendimiento del jugador."""
    steam_id = request.session.get("steam_id")
    if not steam_id:
        steam_id = "76561198088279615"

    return build_performance_overview(str(steam_id))


@router.get("/steam/performance-stats")
async def get_performance_stats(request: Request, force_refresh: bool = False):
    """Compat endpoint: conserva la ruta histórica con un payload resumido."""
    full_payload = await get_performance_overview(request=request, force_refresh=force_refresh)

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
async def player_search(q: str = Query("", min_length=0, max_length=100)):
    """Search players by name across all match exports."""
    index = _build_player_index()

    if not q.strip():
        return {"players": index[:30]}

    query_lower = q.strip().lower()
    results = [p for p in index if query_lower in p["name"].lower()]
    return {"players": results[:30]}


@router.get("/steam/player-stats/{steam_id}")
async def get_player_stats(steam_id: str):
    """Return the performance overview for any player by steam_id."""
    import re
    if not re.match(r"^7656\d{13}$", steam_id):
        return {"error": "Invalid steam_id format"}

    data = build_performance_overview(steam_id)
    overview = data.get("overview", {})

    if not overview.get("total_matches"):
        return {"error": "No data found for this player", "steam_id": steam_id}

    return data
