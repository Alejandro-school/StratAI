# backend/app/routes/performance.py
# -------------------------------
# Ruta para agregar estadísticas de rendimiento global.
# Lee múltiples players_summary.json y genera un perfil unificado.

import logging
from fastapi import APIRouter, Request

from ..utils.performance_aggregator import build_performance_overview

router = APIRouter()
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

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
