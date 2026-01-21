# backend/app/routes/performance.py
# -------------------------------
# Ruta para agregar estadísticas de rendimiento global.
# Lee múltiples players_summary.json y genera un perfil unificado.

import os
import json
import logging
from typing import Any, Dict, List
from fastapi import APIRouter, Request, HTTPException
import redis.asyncio as aioredis

router = APIRouter()

# Configuración básica
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

redis = aioredis.from_url("redis://localhost", decode_responses=True)

DATA_EXPORTS_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "data", "exports")
PLAYER_PROFILE_CACHE_FILE = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "data", "player_profile.json")

def load_json(path: str) -> Any:
    """Helper para cargar JSON de forma segura."""
    if not os.path.exists(path):
        return None
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception as e:
        logger.error(f"Error cargando JSON {path}: {e}")
        return None

def get_all_match_summaries(steam_id: str) -> List[Dict]:
    """Escanea exports y devuelve los summaries donde aparece el steam_id."""
    summaries = []
    if not os.path.exists(DATA_EXPORTS_DIR):
        return summaries

    for match_dir in os.listdir(DATA_EXPORTS_DIR):
        full_dir = os.path.join(DATA_EXPORTS_DIR, match_dir)
        summary_path = os.path.join(full_dir, "players_summary.json")
        metadata_path = os.path.join(full_dir, "metadata.json")
        
        player_data = load_json(summary_path)
        metadata = load_json(metadata_path)
        
        if player_data and "players" in player_data:
            # Buscar al jugador en este match
            target_player = next((p for p in player_data["players"] if str(p.get("steam_id")) == str(steam_id)), None)
            if target_player:
                # Enriquecer con metadata del match (mapa, fecha, resultado)
                target_player["_match_meta"] = {
                    "match_id": match_dir,
                    "map_name": metadata.get("map_name", "unknown") if metadata else "unknown",
                    "date": metadata.get("date", "") if metadata else "",
                    "score_t": metadata.get("score_t", 0) if metadata else 0,
                    "score_ct": metadata.get("score_ct", 0) if metadata else 0,
                    "winner": metadata.get("winner", "") if metadata else ""
                }
                summaries.append(target_player)
                
    return summaries

def calculate_weighted_average(values: List[float], weights: List[float]) -> float:
    """Calcula promedio ponderado."""
    total_weight = sum(weights)
    if total_weight == 0:
        return 0.0
    weighted_sum = sum(v * w for v, w in zip(values, weights))
    return weighted_sum / total_weight

@router.get("/steam/performance-stats")
async def get_performance_stats(request: Request, force_refresh: bool = False):
    """
    Agrega estadísticas de TODOS los players_summary disponibles.
    Devuelve un perfil completo de rendimiento (General, Aim, Mapas, Armas).
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        # Fallback para desarrollo si no hay cookie
        steam_id = "76561198088279615" # Default ID for dev
        # raise HTTPException(status_code=401, detail="Usuario no autenticado")

    # Intentar cargar cache si existe y no forzamos refresh
    # (Para MVP simplificado, recalculamos siempre para asegurar datos frescos, 
    # pero en prod usaríamos player_profile.json como cache)

    summaries = get_all_match_summaries(steam_id)
    if not summaries:
        return {"overall": {}, "aim": {}, "maps": {}, "weapons": {}}

    matches_count = len(summaries)
    
    # ----------------------------------------------------
    # 1. Agregación General (Lifetime Stats)
    # ----------------------------------------------------
    total_kills = sum(s.get("kills", 0) for s in summaries)
    total_deaths = sum(s.get("deaths", 0) for s in summaries)
    total_damage = sum(s.get("total_damage", 0) for s in summaries)
    total_headshots = sum(s.get("headshots", 0) for s in summaries)
    total_shots_fired = sum(s.get("shots_fired", 0) for s in summaries)
    total_shots_hit = sum(s.get("shots_hit", 0) for s in summaries)
    
    # Promedios ponderados por rondas jugadas (aprox por kills/deaths events para aim)
    rounds_played_list = [s.get("rounds_survived", 0) + s.get("deaths", 0) for s in summaries] # Aprox rounds
    # Mejor usar dato real de rondas si existiera, usaremos una constante 1 para promedio simple si no
    
    avg_kd = total_kills / total_deaths if total_deaths > 0 else total_kills
    avg_hs_percent = (total_headshots / total_kills * 100) if total_kills > 0 else 0
    avg_accuracy = (total_shots_hit / total_shots_fired * 100) if total_shots_fired > 0 else 0
    
    # ----------------------------------------------------
    # 2. Aim Stats Aggregation
    # ----------------------------------------------------
    # Weighted by 'hits' or 'kills' for strict aim metrics
    aim_weights = [s.get("hits", 1) for s in summaries]
    
    avg_time_to_damage = calculate_weighted_average(
        [s.get("time_to_damage_avg_ms", 0) for s in summaries], 
        aim_weights
    )
    avg_crosshair_error = calculate_weighted_average(
        [s.get("crosshair_placement_avg_error", 0) for s in summaries], 
        aim_weights
    )

    # ----------------------------------------------------
    # 3. Map Stats Aggregation
    # ----------------------------------------------------
    map_stats = {}
    for s in summaries:
        map_name = s["_match_meta"]["map_name"]
        if map_name not in map_stats:
            map_stats[map_name] = {"kills": 0, "deaths": 0, "matches": 0, "wins": 0}
        
        m = map_stats[map_name]
        m["kills"] += s.get("kills", 0)
        m["deaths"] += s.get("deaths", 0)
        m["matches"] += 1
        
        # Win calculation (simple logic assuming team side)
        # Esto es complejo sin saber en qué equipo terminó, 
        # asumiremos win si el score de su equipo > score enemigo.
        # players_summary tiene 'team' (CT/T) pero cambia en halftime. 
        # Por simplicidad MVP: Si winner == team (last known).
        # TODO: Mejorar lógica de Win Rate con datos de rounds exactos.
        m["wins"] += 0 # Placeholder for now

    # Finalize Map Stats (K/D per map)
    final_map_stats = []
    for m_name, data in map_stats.items():
        kd = data["kills"] / data["deaths"] if data["deaths"] > 0 else data["kills"]
        final_map_stats.append({
            "map_name": m_name,
            "matches": data["matches"],
            "kd": round(kd, 2),
            "win_rate": 0 # Pending better logic
        })

    # ----------------------------------------------------
    # 4. Weapon Stats Aggregation
    # ----------------------------------------------------
    weapon_aggr = {}
    for s in summaries:
        w_stats = s.get("weapon_stats", {})
        for w_name, w_data in w_stats.items():
            if w_name not in weapon_aggr:
                weapon_aggr[w_name] = {"kills": 0, "shots": 0, "hits": 0, "headshots": 0, "damage": 0}
            
            curr = weapon_aggr[w_name]
            curr["kills"] += w_data.get("kills", 0)
            curr["shots"] += w_data.get("shots_fired", 0)
            curr["hits"] += w_data.get("shots_hit", 0)
            curr["headshots"] += w_data.get("headshots", 0)
            curr["damage"] += w_data.get("damage", 0)

    # Format Weapon Stats
    final_weapon_stats = []
    for w_name, data in weapon_aggr.items():
        if data["shots"] > 10: # Filter insignificant usage
            acc = (data["hits"] / data["shots"] * 100) if data["shots"] > 0 else 0
            final_weapon_stats.append({
                "name": w_name,
                "kills": data["kills"],
                "accuracy": round(acc, 1),
                "hs_percent": round((data["headshots"] / data["kills"] * 100) if data["kills"] > 0 else 0, 1)
            })
    
    # Sort weapons by kills
    final_weapon_stats.sort(key=lambda x: x["kills"], reverse=True)

    profile = {
        "steam_id": steam_id,
        "matches_analyzed": matches_count,
        "overall": {
            "kd_ratio": round(avg_kd, 2),
            "hs_percent": round(avg_hs_percent, 1),
            "accuracy": round(avg_accuracy, 1),
            "total_kills": total_kills,
            "total_damage": total_damage
        },
        "aim": {
            "time_to_damage_ms": round(avg_time_to_damage, 0),
            "crosshair_placement_error": round(avg_crosshair_error, 2),
            "reaction_rating": "Good" if avg_time_to_damage < 500 else "Average" # Simple heuristic
        },
        "maps": final_map_stats,
        "weapons": final_weapon_stats[:5] # Top 5 weapons
    }
    
    return profile
