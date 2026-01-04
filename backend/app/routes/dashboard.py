
# backend/app/routes/dashboard.py
# -------------------------------
# Rutas para el dashboard del frontend (stats, mapas, granadas, etc.)

import os
import json
import logging
import redis.asyncio as aioredis
from typing import Any
from fastapi import APIRouter, Request, HTTPException

# Import utils
from ..utils.maps import normalize_callout, game_to_radar_percent, CALLOUT_FIXED_POSITIONS


router = APIRouter()

# Redis logic (same as other files for now)
redis = aioredis.from_url("redis://localhost", decode_responses=True)

# ============================================================================
# PROCESSED DEMOS
# ============================================================================
@router.get("/steam/get-processed-demos")
async def get_processed_demos(request: Request) -> dict[str, Any]:
    """
    Endpoint para historial de partidas - LEE DESDE data/exports/.
    
    Escanea la carpeta exports para encontrar TODAS las partidas
    donde el usuario ha jugado.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    steam_id_str = str(steam_id)
    logging.info(f"[get-processed-demos] Buscando partidas para {steam_id_str}")

    # Escanear directamente data/exports/
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    demos = []
    
    if not os.path.exists(exports_path):
        logging.warning(f"[get-processed-demos] exports path does not exist: {exports_path}")
        return {"matches": []}
    
    for folder_name in os.listdir(exports_path):
        if not folder_name.startswith("match_"):
            continue
        
        match_folder = os.path.join(exports_path, folder_name)
        if not os.path.isdir(match_folder):
            continue
        
        try:
            # 1. Verificar que el usuario está en esta partida
            players_path = os.path.join(match_folder, "players_summary.json")
            if not os.path.exists(players_path):
                continue
            
            with open(players_path, 'r', encoding='utf-8') as f:
                players_json = json.load(f)
                players_data = players_json.get("players", [])
            
            # Buscar al usuario
            user_player = None
            for p in players_data:
                if str(p.get("steam_id", "")) == steam_id_str:
                    user_player = p
                    break
            
            if not user_player:
                continue  # Usuario no está en esta partida
            
            # 2. Cargar metadata.json
            metadata_path = os.path.join(match_folder, "metadata.json")
            if not os.path.exists(metadata_path):
                continue
            
            with open(metadata_path, 'r', encoding='utf-8') as f:
                metadata = json.load(f)
            
            # 3. Determinar resultado para el usuario
            # El score en metadata es siempre CT-T (primer número CT, segundo T)
            final_score = metadata.get("final_score", "0-0")
            scores = final_score.split("-") if "-" in final_score else ["0", "0"]
            ct_score = int(scores[0].strip()) if scores[0].strip().isdigit() else 0
            t_score = int(scores[1].strip()) if len(scores) > 1 and scores[1].strip().isdigit() else 0
            
            # El campo winner indica qué equipo ganó (CT o T)
            winner = metadata.get("winner", "")
            user_team = user_player.get("team", "")
            
            # Victoria si el equipo del usuario es el ganador
            is_victory = user_team == winner
            result = "victory" if is_victory else "defeat"
            
            # Determinar team_score y opponent_score desde la perspectiva del usuario
            if user_team == "CT":
                team_score = ct_score
                opponent_score = t_score
            else:
                team_score = t_score
                opponent_score = ct_score
            
            # 4. Construir objeto para el historial
            demo_data = {
                "match_id": metadata.get("match_id", folder_name),
                "map_name": metadata.get("map_name", "unknown"),
                "match_date": metadata.get("date", ""),
                "match_duration": metadata.get("duration_seconds", 0),
                "result": result,
                "team_score": team_score,
                "opponent_score": opponent_score,
                "total_rounds": metadata.get("total_rounds", 0),
                "user_team": user_team,
                "players": [{
                    "steam_id": user_player.get("steam_id"),
                    "name": user_player.get("name", ""),
                    "kills": user_player.get("kills", 0),
                    "deaths": user_player.get("deaths", 0),
                    "assists": user_player.get("assists", 0),
                    "kd_ratio": user_player.get("kd_ratio", 0),
                    "adr": user_player.get("adr", 0),
                    "hs_percentage": user_player.get("hs_percentage", 0),
                    "hltv_rating": user_player.get("hltv_rating", 0)
                }]
            }
            demos.append(demo_data)
            
        except Exception as e:
            logging.error(f"[get_processed_demos] Error processing {folder_name}: {e}")
            continue
    
    # Ordenar por fecha (más reciente primero)
    demos.sort(key=lambda x: x.get("match_date", ""), reverse=True)
    
    logging.info(f"[get-processed-demos] Encontradas {len(demos)} partidas para {steam_id_str}")
    return {"matches": demos}


# ============================================================================
# MATCH DETAILS (Single match)
# ============================================================================
@router.get("/steam/get-match-details/{match_id}")
async def get_match_details(request: Request, match_id: str) -> dict[str, Any]:
    """
    Endpoint para obtener todos los detalles de una partida específica.
    Combina metadata.json y players_summary.json.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    steam_id_str = str(steam_id)
    logging.info(f"[get-match-details] Buscando detalles de {match_id} para {steam_id_str}")

    # Buscar la carpeta de la partida
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    
    # Try different folder name formats
    possible_folders = [
        os.path.join(exports_path, match_id),
        os.path.join(exports_path, f"match_{match_id}"),
        os.path.join(exports_path, match_id.replace("match_", "")),
    ]
    
    match_folder = None
    for folder in possible_folders:
        if os.path.exists(folder) and os.path.isdir(folder):
            match_folder = folder
            break
    
    if not match_folder:
        logging.warning(f"[get-match-details] Carpeta no encontrada: {match_id}")
        raise HTTPException(status_code=404, detail="Partida no encontrada.")

    try:
        # Leer metadata.json
        metadata_path = os.path.join(match_folder, "metadata.json")
        metadata = {}
        if os.path.exists(metadata_path):
            with open(metadata_path, 'r', encoding='utf-8') as f:
                metadata = json.load(f)
        
        # Leer players_summary.json
        players_path = os.path.join(match_folder, "players_summary.json")
        players_data = {"players": []}
        if os.path.exists(players_path):
            with open(players_path, 'r', encoding='utf-8') as f:
                players_data = json.load(f)
        
        # Separar jugadores por equipo
        players = players_data.get("players", [])
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
            "current_user_steam_id": steam_id_str
        }
        
    except Exception as e:
        logging.error(f"[get-match-details] Error: {e}")
        raise HTTPException(status_code=500, detail=f"Error al leer datos: {str(e)}")


# ============================================================================
# DASHBOARD STATS (Aggregated)
# ============================================================================
@router.get("/steam/get-dashboard-stats")
async def get_dashboard_stats(request: Request, force_refresh: bool = False) -> dict[str, Any]:
    """
    Endpoint OPTIMIZADO para el dashboard.
    
    - NO devuelve event_logs (evita transferir miles de eventos innecesarios)
    - Lee desde data/exports/ cuando existe (datos pre-calculados)
    - Cache con TTL de 1 hora (invalidado automáticamente al procesar demos)
    - Solo incluye stats agregadas: KDA, ADR, HS%, mapas, armas
    - Param force_refresh=true para forzar recálculo sin usar cache
    
    Resultado: Carga 10-20x más rápida que get-processed-demos
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    # 1. Verificar caché (se invalida automáticamente al procesar demos)
    cache_key = f"dashboard_stats:{steam_id}"
    
    if not force_refresh:
        cached = await redis.get(cache_key)
        if cached:
            logging.info(f"[dashboard-stats] Cache HIT para {steam_id}")
            return json.loads(cached)
    else:
        logging.info(f"[dashboard-stats] Force refresh solicitado para {steam_id}")
        await redis.delete(cache_key)  # Limpiar cache viejo
    
    logging.info(f"[dashboard-stats] Cache MISS para {steam_id} - calculando...")
    
    # 2. Obtener lista de match_ids procesadas desde Redis
    processed_demos_raw: list[str] = await redis.lrange(f"processed_demos:{steam_id}", 0, -1)  # type: ignore
    
    # 3. Cargar datos OPTIMIZADOS desde data/exports/ o Redis
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    matches_data = []
    weapon_kills: dict[str, int] = {}
    map_results: dict[str, dict[str, int]] = {}
    
    # FALLBACK: Si Redis está vacío, escanear carpeta exports para encontrar partidas del usuario
    if not processed_demos_raw:
        logging.info(f"[dashboard-stats] Redis vacío para {steam_id} - escaneando exports/...")
        
        if os.path.exists(exports_path):
            for folder_name in os.listdir(exports_path):
                if not folder_name.startswith("match_"):
                    continue
                
                match_folder = os.path.join(exports_path, folder_name)
                players_summary_path = os.path.join(match_folder, "players_summary.json")
                
                if os.path.exists(players_summary_path):
                    try:
                        with open(players_summary_path, 'r', encoding='utf-8') as f:
                            players_data = json.load(f)
                        
                        # Buscar si el usuario está en esta partida
                        for p in players_data.get("players", []):
                            if str(p.get("steam_id", "")) == str(steam_id):
                                # Usuario encontrado en esta partida
                                match_id = folder_name.replace("match_", "")
                                processed_demos_raw.append(json.dumps({"match_id": match_id}))
                                break
                    except Exception as e:
                        logging.warning(f"Error leyendo {players_summary_path}: {e}")
        
        logging.info(f"[dashboard-stats] Encontradas {len(processed_demos_raw)} partidas en exports para {steam_id}")
    
    if not processed_demos_raw:
        empty_response = {
            "steam_id": steam_id,
            "stats": {
                "total_matches": 0, "total_kills": 0, "total_deaths": 0,
                "avg_kd": 0.0, "avg_adr": 0.0, "avg_hs": 0.0, "win_rate": 0.0,
                "wins": 0, "losses": 0
            },
            "recent_matches": [], "weapon_stats": [], "map_stats": []
        }
        await redis.set(cache_key, json.dumps(empty_response), ex=3600)
        return empty_response

    
    for demo_raw in processed_demos_raw:
        demo = json.loads(demo_raw)
        match_id = demo.get("match_id")
        
        # Intentar leer desde data/exports/ primero (más eficiente)
        match_folder = os.path.join(exports_path, f"match_{match_id}")
        players_summary_path = os.path.join(match_folder, "players_summary.json")
        match_info_path = os.path.join(match_folder, "match_info.json")
        combat_path = os.path.join(match_folder, "combat.json")
        metadata_path = os.path.join(match_folder, "metadata.json")
        
        player_stats = None
        match_info = None
        
        if not os.path.exists(match_folder):
            continue
        
        # Leer players_summary.json para stats del jugador (si existe)
        if os.path.exists(players_summary_path):
            try:
                with open(players_summary_path, 'r', encoding='utf-8') as f:
                    players_data = json.load(f)
                    # Buscar al jugador
                    for p in players_data.get("players", []):
                        if str(p.get("steam_id", "")) == str(steam_id):
                            player_stats = p
                            break
            except Exception as e:
                logging.warning(f"Error leyendo {players_summary_path}: {e}")
        
        # Leer metadata.json para resultado y mapa
        if os.path.exists(metadata_path):
            try:
                with open(metadata_path, 'r', encoding='utf-8') as f:
                    metadata = json.load(f)
                    final_score = metadata.get("final_score", "0-0")
                    scores = final_score.split("-") if "-" in final_score else ["0", "0"]
                    team_score = int(scores[0]) if scores[0].isdigit() else 0
                    opponent_score = int(scores[1]) if len(scores) > 1 and scores[1].isdigit() else 0
                    
                    match_info = {
                        "map_name": metadata.get("map_name", "unknown"),
                        "date": metadata.get("date", ""),
                        "team_score": team_score,
                        "opponent_score": opponent_score,
                        "winner": metadata.get("winner", ""),
                        "duration_seconds": metadata.get("duration_seconds", 0),
                        "total_rounds": metadata.get("total_rounds", 1)
                    }
            except Exception as e:
                logging.warning(f"Error leyendo {metadata_path}: {e}")
        
        # Fallback: Si no hay players_summary.json, calcular stats desde combat.json
        if not player_stats and os.path.exists(combat_path):
            try:
                with open(combat_path, 'r', encoding='utf-8') as f:
                    combat_data = json.load(f)
                
                kills = 0
                deaths = 0
                headshots = 0
                total_damage = 0
                
                for round_data in combat_data.get("rounds", []):
                    for duel in round_data.get("duels", []):
                        attacker = duel.get("attacker", {})
                        victims = duel.get("victims", [])
                        
                        # Usuario atacante
                        if str(attacker.get("steam_id", "")) == str(steam_id):
                            if duel.get("outcome") == "kill":
                                kills += duel.get("victim_count", 1)
                                headshots += attacker.get("headshots", 0)
                            total_damage += attacker.get("total_damage_dealt", 0)
                        
                        # Usuario víctima
                        for victim in victims:
                            if str(victim.get("steam_id", "")) == str(steam_id):
                                if duel.get("outcome") == "kill" and victim.get("health_after", 0) == 0:
                                    deaths += 1
                
                total_rounds = match_info.get("total_rounds", 1) if match_info else 1
                adr = total_damage / max(total_rounds, 1)
                hs_percentage = (headshots / kills * 100) if kills > 0 else 0
                kd_ratio = kills / max(deaths, 1)
                
                player_stats = {
                    "kills": kills, "deaths": deaths, "assists": 0,
                    "kd_ratio": round(kd_ratio, 2), "adr": round(adr, 1),
                    "hs_percentage": round(hs_percentage, 1), "headshots": headshots
                }
            except Exception as e:
                logging.warning(f"Error calculando stats desde combat.json para {match_id}: {e}")
        
        if not match_info:
            continue
        
        if player_stats:
            team_score = match_info.get("team_score", 0)
            opponent_score = match_info.get("opponent_score", 0)
            result = "W" if team_score > opponent_score else "L"
            
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

            if "weapon_stats" in player_stats:
                for weapon, stats in player_stats["weapon_stats"].items():
                    w_kills = stats.get("kills", 0)
                    if w_kills > 0:
                        weapon_kills[weapon] = weapon_kills.get(weapon, 0) + w_kills
    
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
    
    # Top weapons
    top_weapons = sorted(weapon_kills.items(), key=lambda x: x[1], reverse=True)[:4]
    weapon_stats_list = [{"weapon": w, "kills": k} for w, k in top_weapons]
    
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
        "recent_matches": recent_matches,
        "weapon_stats": weapon_stats_list,
        "map_stats": map_stats_list
    }
    
    await redis.set(cache_key, json.dumps(response), ex=3600)
    logging.info(f"[dashboard-stats] Calculado y cacheado para {steam_id} - {total_matches} partidas")
    
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
async def get_map_zone_stats(request: Request, map_name: str = "de_dust2") -> dict[str, Any]:
    """
    Endpoint for map zone statistics with side split (T/CT).
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    steam_id_str = str(steam_id)
    logging.info(f"[map-zone-stats] Request for sid={steam_id_str}, map={map_name}")
    
    zone_ids = ["site-a", "site-b", "mid", "long-a", "b-tunnels", "catwalk", "t-spawn", "ct-spawn"]
    
    def empty_side_stats():
        return {"kills": 0, "deaths": 0, "duels_won": 0, "duels_lost": 0}
    
    # {zid: {"t": {...}, "ct": {...}, "combined": {...}}}
    zone_stats = {zid: {"t": empty_side_stats(), "ct": empty_side_stats(), "combined": empty_side_stats()} for zid in zone_ids}
    
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    matches_analyzed = 0
    
    if os.path.exists(exports_path):
        for folder_name in os.listdir(exports_path):
            if not folder_name.startswith("match_"):
                continue
            
            match_folder = os.path.join(exports_path, folder_name)
            
            # 1. Metadata check
            metadata_path = os.path.join(match_folder, "metadata.json")
            if os.path.exists(metadata_path):
                try:
                    with open(metadata_path, 'r', encoding='utf-8') as f:
                        meta = json.load(f)
                        if meta.get("map_name") != map_name:
                            continue
                except Exception: continue
            
            # 2. Player presence check
            players_summary_path = os.path.join(match_folder, "players_summary.json")
            if not os.path.exists(players_summary_path): continue
            
            try:
                with open(players_summary_path, 'r', encoding='utf-8') as f:
                    p_data = json.load(f)
                    players = p_data.get("players", []) if isinstance(p_data, dict) else p_data
                    if not any(str(p.get("steam_id")) == steam_id_str for p in players):
                        continue
            except Exception: continue
            
            # 3. Combat Data
            combat_path = os.path.join(match_folder, "combat.json")
            if not os.path.exists(combat_path): continue
            
            try:
                with open(combat_path, 'r', encoding='utf-8') as f:
                    combat_data = json.load(f)
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
async def get_callout_stats(request: Request, map_name: str = "de_dust2") -> dict[str, Any]:
    """
    Granular per-callout statistics with full data: K/D, win rates, weapons, context, etc.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    steam_id_str = str(steam_id)
    logging.info(f"[callout-stats] Request for sid={steam_id_str}, map={map_name}")
    
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
    
    for folder_name in os.listdir(exports_path):
        if not folder_name.startswith("match_"):
            continue
        
        match_folder = os.path.join(exports_path, folder_name)
        
        # 1. Metadata filter
        metadata_path = os.path.join(match_folder, "metadata.json")
        if os.path.exists(metadata_path):
            try:
                with open(metadata_path, 'r', encoding='utf-8') as f:
                    meta = json.load(f)
                    if meta.get("map_name") != map_name:
                        continue
            except Exception: continue
        
        # 2. Player check & Summary Stats
        players_summary_path = os.path.join(match_folder, "players_summary.json")
        if not os.path.exists(players_summary_path): continue
        
        player_summary = None
        try:
            with open(players_summary_path, 'r', encoding='utf-8') as f:
                p_data = json.load(f)
                players = p_data.get("players", []) if isinstance(p_data, dict) else p_data
                for p in players:
                    if str(p.get("steam_id")) == steam_id_str:
                        player_summary = p
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
        
        # 3. Combat logic with full context
        combat_path = os.path.join(match_folder, "combat.json")
        if not os.path.exists(combat_path): continue
        
        try:
            with open(combat_path, 'r', encoding='utf-8') as f:
                combat_data = json.load(f)
                
                for round_item in combat_data.get("rounds", []):
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
        
        # 4. Get positions from tracking.json
        tracking_path = os.path.join(match_folder, "tracking.json")
        if os.path.exists(tracking_path):
            try:
                with open(tracking_path, 'r', encoding='utf-8') as f:
                    tracking_data = json.load(f)
                    for round_item in tracking_data.get("rounds", []):
                        for tick_data in round_item.get("ticks", []):
                            for player in tick_data.get("players", []):
                                if str(player.get("player_steam_id")) == steam_id_str:
                                    raw_area = player.get("area_name", "")
                                    normalized_area = normalize_callout(raw_area, map_name)
                                    if normalized_area and normalized_area in callout_stats:
                                        pos = player.get("pos", {})
                                        if pos.get("x") and pos.get("y"):
                                            callout_stats[normalized_area]["positions_x"].append(pos["x"])
                                            callout_stats[normalized_area]["positions_y"].append(pos["y"])
                                            # [NEW] Also capture Z coordinate
                                            if pos.get("z") is not None:
                                                callout_stats[normalized_area]["positions_z"].append(pos["z"])
                                    break
            except Exception as e:
                logging.warning(f"[callout-stats] Error parsing tracking: {e}")

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
            position = game_to_radar_percent(avg_x, avg_y, map_name)
            
            # [NEW] Calculate avg_z from tracking positions
            if stats.get("positions_z"):
                avg_z = sum(stats["positions_z"]) / len(stats["positions_z"])
        
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
async def get_aggregate_grenades(request: Request, map_name: str = "de_dust2") -> dict[str, Any]:
    """
    Aggregate grenade statistics across all matches for a map.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    steam_id_str = str(steam_id)
    logging.info(f"[aggregate-grenades] Request for sid={steam_id_str}, map={map_name}")
    
    grenade_positions = {"smoke": [], "flash": [], "he": [], "molotov": []}
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    matches_analyzed = 0
    
    if os.path.exists(exports_path):
        for folder_name in os.listdir(exports_path):
            if not folder_name.startswith("match_"):
                continue
            match_folder = os.path.join(exports_path, folder_name)
            
            # Metadata filter
            metadata_path = os.path.join(match_folder, "metadata.json")
            if os.path.exists(metadata_path):
                try:
                    with open(metadata_path, 'r', encoding='utf-8') as f:
                        meta = json.load(f)
                        if meta.get("map_name") != map_name: continue
                except Exception: continue
            
            # Identify User Name
            players_summary_path = os.path.join(match_folder, "players_summary.json")
            user_name = None
            if os.path.exists(players_summary_path):
                try:
                    with open(players_summary_path, 'r') as f:
                        p_data = json.load(f)
                        for p in p_data.get("players", []):
                            if str(p.get("steam_id")) == steam_id_str:
                                user_name = p.get("name")
                                break
                except: pass
            
            if not user_name: continue
            
            # Load grenades with FULL stats
            grenades_path = os.path.join(match_folder, "grenades.json")
            if os.path.exists(grenades_path):
                try:
                    with open(grenades_path, 'r', encoding='utf-8') as f:
                        g_data = json.load(f)
                        for r_item in g_data.get("rounds", []):
                            for event in r_item.get("events", []):
                                if event.get("thrower") == user_name:
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
                                                "land_area": event.get("land_area", ""),
                                                # NEW: Extract full stats
                                                "damage_dealt": event.get("damage_dealt", 0),
                                                "enemies_blinded": event.get("enemies_blinded", 0),
                                                "allies_blinded": event.get("allies_blinded", 0),

                                                "duration": event.get("duration", 0),
                                                "extinguished": event.get("extinguished", False),
                                                "kills": event.get("kills", 0)
                                            })
                except Exception as e:
                    logging.warning(f"[aggregate-grenades] Error parsing grenades.json: {e}")
            matches_analyzed += 1
            
    # Build summary stats from grenades.json data
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
            radar = game_to_radar_percent(c["game_x"], c["game_y"], map_name)
            c["x"] = radar["x"]
            c["y"] = radar["y"]
            
            # Aggregate cluster stats from raw positions
            raw_positions = c.get("positions", [])
            c["total_damage"] = sum(p.get("damage_dealt", 0) for p in raw_positions)
            c["avg_damage"] = round(c["total_damage"] / len(raw_positions), 1) if raw_positions else 0
            c["total_blinded"] = sum(p.get("enemies_blinded", 0) for p in raw_positions)
            c["avg_blinded"] = round(c["total_blinded"] / len(raw_positions), 2) if raw_positions else 0
            
            # Calculate avg_z for multi-level maps (Nuke, Vertigo, Train)
            z_values = [p.get("end_z", 0) for p in raw_positions if p.get("end_z", 0) != 0]
            c["avg_z"] = round(sum(z_values) / len(z_values), 1) if z_values else None

            c["areas"] = list(set(p.get("land_area", "") for p in raw_positions if p.get("land_area")))
            
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
                end_radar = game_to_radar_percent(avg_end_x, avg_end_y, map_name)
                
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
async def get_movement_stats(request: Request, map_name: str = "de_dust2") -> dict[str, Any]:
    """
    Movement analysis for the Hybrid Flow + Heatmap visualization.
    
    Processes tracking.json to extract:
    - Heatmap grid: Position density across 20x20 grid cells
    - Flow lines: Common routes between map areas
    - Metrics: Time-to-site, position frequency, etc.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    steam_id_str = str(steam_id)
    steam_id_int = int(steam_id)
    logging.info(f"[movement-stats] Request for sid={steam_id_str}, map={map_name}")
    
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
    
    for folder_name in os.listdir(exports_path):
        if not folder_name.startswith("match_"):
            continue
        
        match_folder = os.path.join(exports_path, folder_name)
        
        # 1. Metadata filter - must be correct map
        metadata_path = os.path.join(match_folder, "metadata.json")
        if os.path.exists(metadata_path):
            try:
                with open(metadata_path, 'r', encoding='utf-8') as f:
                    meta = json.load(f)
                    if meta.get("map_name") != map_name:
                        continue
            except Exception:
                continue
        
        # 2. Player check
        players_summary_path = os.path.join(match_folder, "players_summary.json")
        if not os.path.exists(players_summary_path):
            continue
        try:
            with open(players_summary_path, 'r', encoding='utf-8') as f:
                p_data = json.load(f)
                players = p_data.get("players", []) if isinstance(p_data, dict) else p_data
                player_found = any(str(p.get("steam_id")) == steam_id_str for p in players)
                if not player_found:
                    continue
        except Exception:
            continue
        
        # 3. Process tracking.json
        tracking_path = os.path.join(match_folder, "tracking.json")
        if not os.path.exists(tracking_path):
            continue
        
        try:
            with open(tracking_path, 'r', encoding='utf-8') as f:
                tracking_data = json.load(f)
            
            for round_item in tracking_data.get("rounds", []):
                round_num = round_item.get("round", 0)
                total_rounds += 1
                
                # Track player's previous area for transitions
                prev_area = None
                round_team = None  # Team for this round (detected from spawn)
                round_start_tick = None
                reached_a = False
                reached_b = False
                first_a_tick = None
                first_b_tick = None
                first_tick_processed = False
                
                for tick_data in round_item.get("ticks", []):
                    tick = tick_data.get("tick", 0)
                    
                    for player in tick_data.get("players", []):
                        player_id = player.get("player_steam_id")
                        
                        # Match by int or string (tracking uses int)
                        if player_id != steam_id_int and str(player_id) != steam_id_str:
                            continue
                        
                        if not player.get("is_alive", False):
                            continue
                        
                        pos = player.get("pos", {})
                        area = player.get("area_name", "")
                        
                        if not pos or not area:
                            continue
                        
                        game_x = pos.get("x", 0)
                        game_y = pos.get("y", 0)
                        
                        # Get team directly from tracking data (added in Go parser)
                        # Falls back to inferring from spawn if not present (old data)
                        team = player.get("team", "")
                        if not team and not first_tick_processed:
                            first_tick_processed = True
                            round_team = _infer_team_from_area(area, map_name)
                            round_start_tick = tick
                        if not team:
                            team = round_team
                        
                        # Get Z coordinate for multi-level map filtering
                        game_z = pos.get("z", 0)
                        
                        # Grid heatmap
                        radar_pos = game_to_radar_percent(game_x, game_y, map_name)
                        grid_x = min(GRID_SIZE - 1, int(radar_pos["x"] / 100 * GRID_SIZE))
                        grid_y = min(GRID_SIZE - 1, int(radar_pos["y"] / 100 * GRID_SIZE))
                        grid_key = (grid_x, grid_y)
                        
                        if grid_key not in grid_counts:
                            grid_counts[grid_key] = {"total": 0, "ct": 0, "t": 0, "z_sum": 0.0}
                        grid_counts[grid_key]["total"] += 1
                        grid_counts[grid_key]["z_sum"] += game_z
                        if team:
                            grid_counts[grid_key][team.lower()] += 1
                        
                        # Area time tracking
                        normalized_area = normalize_callout(area, map_name) or area
                        if normalized_area not in area_time:
                            area_time[normalized_area] = {"total_ticks": 0, "ct": 0, "t": 0}
                        area_time[normalized_area]["total_ticks"] += 1
                        if team:
                            area_time[normalized_area][team.lower()] += 1
                        
                        # Area transitions (flow lines)
                        if prev_area and prev_area != normalized_area:
                            transition_key = (prev_area, normalized_area)
                            if transition_key not in area_transitions:
                                area_transitions[transition_key] = {"count": 0, "ct": 0, "t": 0, "positions": []}
                            area_transitions[transition_key]["count"] += 1
                            area_transitions[transition_key]["positions"].append({
                                "x": radar_pos["x"],
                                "y": radar_pos["y"]
                            })
                            if team:
                                area_transitions[transition_key][team.lower()] += 1
                        
                        prev_area = normalized_area
                        
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
                        side = prev_team or "t"
                        time_to_site["A"][side.lower()].append(time_a)
                    if first_b_tick:
                        time_b = (first_b_tick - round_start_tick) / 64
                        side = prev_team or "t"
                        time_to_site["B"][side.lower()].append(time_b)
            
            matches_analyzed += 1
            
        except Exception as e:
            logging.warning(f"[movement-stats] Error parsing tracking for {folder_name}: {e}")
            continue
    
    # Build response
    
    # 1. Heatmap grid (20x20 cells with intensity)
    max_count = max((c["total"] for c in grid_counts.values()), default=1)
    heatmap_grid = []
    for (gx, gy), counts in grid_counts.items():
        intensity = round((counts["total"] / max_count) * 100, 1)
        if intensity > 1:  # Only include cells with meaningful data
            # Calculate average Z for multi-level maps (Nuke, Vertigo, Train)
            avg_z = counts.get("z_sum", 0) / counts["total"] if counts["total"] > 0 else 0
            heatmap_grid.append({
                "x": (gx + 0.5) * (100 / GRID_SIZE),  # Center of cell
                "y": (gy + 0.5) * (100 / GRID_SIZE),
                "intensity": intensity,
                "ct_ratio": round(counts["ct"] / counts["total"] * 100, 1) if counts["total"] > 0 else 50,
                "sample_count": counts["total"],
                "avg_z": round(avg_z, 1)  # For level filtering
            })
    
    # Sort by intensity for rendering order
    heatmap_grid.sort(key=lambda x: x["intensity"])
    
    # 2. Flow lines (top 15 most common transitions)
    flow_lines = []
    sorted_transitions = sorted(area_transitions.items(), key=lambda x: x[1]["count"], reverse=True)
    
    for (from_area, to_area), data in sorted_transitions[:15]:
        if data["count"] < 2:  # Skip rare transitions
            continue
        
        # Calculate average position along the route
        positions = data.get("positions", [])
        if positions:
            avg_x = sum(p["x"] for p in positions) / len(positions)
            avg_y = sum(p["y"] for p in positions) / len(positions)
        else:
            avg_x, avg_y = 50, 50
        
        # Get fixed positions for from/to areas
        from_pos = CALLOUT_FIXED_POSITIONS.get(map_name, {}).get(from_area)
        to_pos = CALLOUT_FIXED_POSITIONS.get(map_name, {}).get(to_area)
        
        flow_lines.append({
            "from_area": from_area,
            "to_area": to_area,
            "from_x": from_pos["x"] if from_pos else avg_x - 5,
            "from_y": from_pos["y"] if from_pos else avg_y - 5,
            "to_x": to_pos["x"] if to_pos else avg_x + 5,
            "to_y": to_pos["y"] if to_pos else avg_y + 5,
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
    
    for area, data in sorted_areas[:10]:
        pct = round((data["total_ticks"] / total_ticks) * 100, 1) if total_ticks > 0 else 0
        top_positions.append({
            "area": area,
            "time_percent": pct,
            "ct_percent": round((data["ct"] / data["total_ticks"]) * 100, 1) if data["total_ticks"] > 0 else 50,
            "sample_count": data["total_ticks"]
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
    
    return {
        "heatmap_grid": heatmap_grid,
        "flow_lines": flow_lines,
        "metrics": metrics,
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

