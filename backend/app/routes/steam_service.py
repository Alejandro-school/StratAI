# steam_service.py
# ----------------
# Rutas de FastAPI para gestionar sharecodes en CS:GO/CS2.

import os
import json
import logging
import httpx
import asyncio
import requests
import redis.asyncio as aioredis
from fastapi import APIRouter, Request, HTTPException, Query
from typing import Any

router = APIRouter()

# Log básico
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# Redis cliente (routes-level)
redis = aioredis.from_url("redis://localhost", decode_responses=True)  # type: ignore

def get_steam_data(steam_id: str) -> dict[str, Any]:
    """
    Obtiene avatar y rango (CS2 Premiere) vía API Steam.
    Lee la API key en tiempo de ejecución para evitar problemas de import.
    """
    STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")
    url_avatar = f"http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key={STEAM_API_KEY}&steamids={steam_id}"
    url_rank   = f"https://api.steampowered.com/ICSGOPlayers_730/GetGamePersonalData/v1/?key={STEAM_API_KEY}&steamid={steam_id}"

    avatar_url: str | None = None
    rank: int | None = None

    # Avatar
    try:
        response = requests.get(url_avatar, timeout=5)
        if response.status_code == 200 and response.text.strip():
            avatar_response = response.json()
            if "response" in avatar_response and "players" in avatar_response["response"]:
                players = avatar_response["response"]["players"]
                if players:
                    avatar_url = players[0].get("avatarfull", None)
    except requests.exceptions.Timeout:
        logging.debug(f"Timeout al obtener avatar para {steam_id}")
    except requests.exceptions.JSONDecodeError:
        logging.debug(f"Respuesta no-JSON al obtener avatar para {steam_id}")
    except Exception as e:
        logging.debug(f"Error al obtener avatar para {steam_id}: {e}")

    # Rango CS2 Premiere
    try:
        response = requests.get(url_rank, timeout=5)
        if response.status_code == 200 and response.text.strip():
            rank_response = response.json()
            if "result" in rank_response and "rank_type" in rank_response["result"]:
                rank = rank_response["result"]["rank_type"]
    except requests.exceptions.Timeout:
        logging.debug(f"Timeout al obtener rango para {steam_id}")
    except requests.exceptions.JSONDecodeError:
        logging.debug(f"Respuesta no-JSON al obtener rango para {steam_id}")
    except Exception as e:
        logging.debug(f"Error al obtener rango para {steam_id}: {e}")

    return {"avatar": avatar_url, "rank": rank}


@router.post("/steam/save-steam-id")
async def save_steam_id(request: Request) -> dict[str, str]:
    """
    Guarda el Steam ID del usuario autenticado en Redis (set all_steam_ids).
    """
    logging.info(f"🔍 Cookies recibidas: {request.cookies}")
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    await redis.sadd("all_steam_ids", steam_id)  # type: ignore
    logging.info(f"✅ Steam ID {steam_id} agregado al set 'all_steam_ids'.")
    return {"message": "Steam ID guardado correctamente en Redis."}


@router.get("/steam/all-sharecodes")
async def get_all_sharecodes(
    request: Request,
    auth_code: str = Query(..., alias="auth_code"),
    last_code: str | None = Query(None, alias="last_code"),
    known_code: str | None = Query(None, alias="known_code"),
) -> dict[str, Any]:
    """
    Obtiene sharecodes siguientes usando la API oficial de Steam.
    Reglas:
      - last_code/known_code es OBLIGATORIO si no hay uno previo en Redis.
      - Si ya hay uno guardado en Redis, puede omitirse en la query.
      - Un único fallback 'N/A' si Steam responde 412 para resincronizar.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Cargar API key en tiempo de petición (evita problemas de import/orden)
    STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")
    if not STEAM_API_KEY:
        raise HTTPException(status_code=400, detail="Falta Steam API Key")

    # Normaliza (sin cambiar mayúsculas/minúsculas)
    auth_code = auth_code.strip()

    # 1) Usa param de query si viene
    provided_code = (last_code or known_code)
    if provided_code:
        current_code = provided_code.strip()
    else:
        # 2) Si no viene, usa el último guardado en Redis
        saved_code = await redis.get(f"{steam_id}:knownCode")
        if saved_code:
            current_code = saved_code.strip()
        else:
            # 3) No hay en query ni en Redis -> obligatorio
            raise HTTPException(
                status_code=400,
                detail="Falta last_code/known_code y no hay valor previo en Redis. Indica tu último share code."
            )

    # Control
    sharecodes: list[str] = []
    max_retries = 5
    delay_seconds = 0.2
    max_total_codes = 50
    original_code = current_code
    resynced = False
    data: dict[str, Any] = {}

    logging.info(f"[all-sharecodes] Usuario={steam_id} | code_inicial='{current_code}'")

    try:
        async with httpx.AsyncClient() as client:
            while True:
                success = False
                for attempt in range(max_retries):
                    # params en cada intento (respeta current_code actualizado)
                    params: dict[str, str] = {
                        "key": STEAM_API_KEY,
                        "steamid": steam_id,
                        "steamidkey": auth_code,
                        "knowncode": current_code,
                    }

                    resp = await client.get(
                        "https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/",
                        params=params
                    )

                    logging.info(f"[all-sharecodes] HTTP {resp.status_code} con knowncode='{current_code}'")

                    # 412 -> code inválido/caducado: probar una sola vez con 'N/A'
                    if resp.status_code == 412:
                        logging.warning(
                            f"[all-sharecodes] 412 con knowncode='{current_code}'. "
                            f"Body: {resp.text[:300]}"
                        )
                        if not resynced and current_code != "N/A":
                            current_code = "N/A"
                            resynced = True
                            continue
                        raise HTTPException(
                            status_code=400,
                            detail=(
                                "Auth/ShareCode inválido: verifica que el AUTH CODE corresponde "
                                "a este SteamID y vuelve a generarlo en Steam Soporte. "
                                "Si el problema persiste, copia el último share code exacto desde Steam."
                            )
                        )

                    # 429 -> backoff incremental
                    if resp.status_code == 429:
                        wait = delay_seconds * (attempt + 1)
                        logging.warning(f"[all-sharecodes] 429 (rate limit). Reintentando en {wait:.2f}s…")
                        await asyncio.sleep(wait)
                        continue

                    resp.raise_for_status()
                    data = resp.json()
                    success = True
                    break

                if not success:
                    raise HTTPException(
                        status_code=429,
                        detail="Límite de reintentos alcanzado. Intente en unos minutos."
                    )

                next_code = data.get("result", {}).get("nextcode", "").strip()

                # Fin si vacío/nulo/N/A
                if not next_code or next_code.lower() in ("n/a", "null", "none"):
                    logging.info(f"[all-sharecodes] FIN | Usuario={steam_id} | motivo=respuesta vacía/nula")
                    break

                # Añadir y avanzar
                sharecodes.append(next_code)
                current_code = next_code

                # Límite de seguridad
                if len(sharecodes) >= max_total_codes:
                    logging.info(f"[all-sharecodes] LÍMITE | Usuario={steam_id} | códigos={max_total_codes}")
                    break

                await asyncio.sleep(delay_seconds)

        # Actualiza Redis con el último estado
        if sharecodes:
            await redis.set(f"{steam_id}:authCode", auth_code)
            await redis.set(f"{steam_id}:knownCode", sharecodes[-1])
            logging.info(f"[all-sharecodes] REDIS UPDATE | Usuario={steam_id} | nuevo_knownCode={sharecodes[-1]}")
        else:
            await redis.set(f"{steam_id}:knownCode", original_code)
            logging.info(f"[all-sharecodes] REDIS PRESERVE | Usuario={steam_id} | knownCode={original_code}")

        return {
            "sharecodes": sharecodes,
            "has_more": len(sharecodes) == max_total_codes
        }

    except httpx.HTTPError as e:
        logging.error(f"[all-sharecodes] ERROR HTTP | Usuario={steam_id} | {str(e)}")
        raise HTTPException(
            status_code=502,
            detail="Error de comunicación con los servidores de Steam. Intente nuevamente."
        )


@router.post("/steam/save-sharecodes")
async def save_sharecodes(request: Request) -> dict[str, str]:
    """
    Recibe sharecodes, auth_code y known_code, y los almacena en Redis.
    El bot Node.js se activa automáticamente al detectar el rpush.
    """
    data = await request.json()
    sharecodes = data.get("sharecodes", [])
    auth_code = data.get("auth_code")
    known_code = data.get("known_code")

    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    if not sharecodes:
        raise HTTPException(status_code=400, detail="No se proporcionaron sharecodes.")
    if not auth_code or not known_code:
        raise HTTPException(status_code=400, detail="Faltan auth_code o known_code.")

    # Actualiza lista de sharecodes del usuario
    await redis.delete(f"sharecodes:{steam_id}")
    await redis.rpush(f"sharecodes:{steam_id}", *sharecodes)  # type: ignore

    # Guarda auth_code y known_code
    await redis.set(f"{steam_id}:authCode", auth_code)
    await redis.set(f"{steam_id}:knownCode", known_code)

    logging.info(f"[save-sharecodes] Almacenados {len(sharecodes)} sharecodes en Redis para {steam_id}")
    return {"message": "✅ ShareCodes guardados. El bot se activará automáticamente."}


@router.get("/steam/check-friend-status")
async def check_friend_status(request: Request) -> dict[str, Any]:
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # 1) Cache rápida
    cached = await redis.get(f"friend_status:{steam_id}")   # friend|pending|not_friend|unknown
    cached_ts = await redis.get(f"friend_status_ts:{steam_id}")

    # 2) Intento live contra node-service
    try:
        async with httpx.AsyncClient(timeout=3.0) as client:
            response = await client.get(
                "http://localhost:4000/steam/check-friend",
                params={"steam_id": steam_id}
            )
            response.raise_for_status()
            data = response.json()
            # normalizamos
            is_friend = bool(data.get("is_friend", False))
            status = data.get("status", "unknown")
            service_down = bool(data.get("service_down", False))
            source = data.get("source", "live")

            # refresca cache si vino de live
            if source == "live":
                await redis.set(f"friend_status:{steam_id}", status, ex=86400)
                await redis.set(f"friend_status_ts:{steam_id}", 
                                data.get("cached_at") or __import__("datetime").datetime.utcnow().isoformat(), 
                                ex=86400)

            return {
                "is_friend": is_friend,
                "status": status,
                "service_down": service_down,
                "source": source,
                "cached_at": cached_ts
            }
    except httpx.RequestError as e:
        # node-service caído: devolvemos cache
        logging.warning(f"[check_friend_status] node-service caído: {e}")
        return {
            "is_friend": (cached == "friend"),
            "status": cached or "unknown",
            "service_down": True,
            "source": "cache",
            "cached_at": cached_ts
        }

@router.post("/steam/send-friend-request")
async def send_friend_request(request: Request) -> dict[str, Any]:
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.post("http://localhost:4000/steam/send-friend-request",
                                     json={"steam_id": steam_id})
            resp.raise_for_status()
            data = resp.json()
            # guardamos pending en cache por UX
            await redis.set(f"friend_status:{steam_id}", data.get("status", "pending"), ex=86400)
            await redis.set(f"friend_status_ts:{steam_id}", 
                            __import__("datetime").datetime.utcnow().isoformat(), ex=86400)
            return data
    except httpx.RequestError as e:
        raise HTTPException(status_code=503, detail=f"Bot no operativo: {e}")
    except httpx.HTTPStatusError as e:
        raise HTTPException(status_code=e.response.status_code, detail=e.response.text)




@router.get("/steam/check-sharecodes")
async def check_sharecodes(request: Request) -> dict[str, Any]:
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    existing_sharecodes: list[str] = await redis.lrange(f"sharecodes:{steam_id}", 0, -1)  # type: ignore
    return {"exists": bool(existing_sharecodes), "sharecodes": existing_sharecodes}  # type: ignore


@router.get("/steam/get-processed-demos")
async def get_processed_demos(request: Request) -> dict[str, Any]:
    """
    Endpoint para historial de partidas - LEE DESDE data/exports/.
    
    Ultra rápido: lee directamente match_info.json y players_summary.json
    sin cargar event_logs ni datos pesados.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Obtener lista de match_ids desde Redis
    processed_demos_raw: list[str] = await redis.lrange(f"processed_demos:{steam_id}", 0, -1)  # type: ignore
    if not processed_demos_raw:
        raise HTTPException(status_code=404, detail="No hay partidas procesadas.")

    # Leer datos DIRECTAMENTE desde data/exports/
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    demos = []
    
    for demo_raw in processed_demos_raw:
        demo = json.loads(demo_raw)
        match_id = demo.get("match_id")
        if not match_id:
            continue
        
        match_folder = os.path.join(exports_path, str(match_id))
        if not os.path.exists(match_folder):
            continue
        
        try:
            # 1. Cargar match_info.json
            match_info_path = os.path.join(match_folder, "match_info.json")
            if not os.path.exists(match_info_path):
                continue
                
            with open(match_info_path, 'r', encoding='utf-8') as f:
                match_info = json.load(f)
            
            # 2. Cargar players_summary.json
            players_path = os.path.join(match_folder, "players_summary.json")
            players_data = []
            if os.path.exists(players_path):
                with open(players_path, 'r', encoding='utf-8') as f:
                    players_json = json.load(f)
                    players_data = players_json.get("players", [])
            
            # 3. Construir objeto simplificado para el historial
            demo_simplified = {
                "match_id": match_id,
                "map_name": match_info.get("map_name", "unknown"),
                "match_date": match_info.get("date"),
                "match_duration": match_info.get("duration", "0"),
                "team_score": match_info.get("team_score", 0),
                "opponent_score": match_info.get("opponent_score", 0),
                "result": match_info.get("result", ""),
                "players": []
            }
            
            # 4. Reducir datos de players - solo stats básicas
            for player in players_data:
                simplified = {
                    "steamID": player.get("steamID"),
                    "name": player.get("name"),
                    "team": player.get("team"),
                    "kills": player.get("kills", 0),
                    "deaths": player.get("deaths", 0),
                    "assists": player.get("assists", 0),
                    "kd_ratio": player.get("kd_ratio", 0),
                    "adr": player.get("adr", 0),
                    "hs_percentage": player.get("hs_percentage", 0)
                }
                demo_simplified["players"].append(simplified)
            
            demos.append(demo_simplified)
            
        except Exception as e:
            logging.warning(f"Error leyendo exports para {match_id}: {e}")
            continue
    
    if not demos:
        raise HTTPException(status_code=404, detail="No se pudieron cargar partidas desde exports.")
    
    # Enriquecer datos del usuario autenticado con avatar
    try:
        user_steam_data = get_steam_data(steam_id)
        for demo in demos:
            for player in demo.get("players", []):
                if player.get("steamID") == steam_id:
                    player["avatar"] = user_steam_data.get("avatar")
                    player["rank"] = user_steam_data.get("rank")
    except Exception as e:
        logging.warning(f"No se pudo obtener datos de Steam para {steam_id}: {e}")
    
    logging.info(f"[get-processed-demos] Devolviendo {len(demos)} demos SIN event_logs para {steam_id}")
    return {"steam_id": steam_id, "demos": demos}


@router.get("/steam/get-user-data")
async def get_user_data(request: Request) -> dict[str, Any]:
    """Obtiene datos del usuario autenticado desde Steam API (avatar y rango)."""
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    steam_data = get_steam_data(steam_id)
    return {"steam_id": steam_id, "avatar": steam_data.get("avatar"), "rank": steam_data.get("rank")}


# ============================================================================
# NOTA: /steam/get-match-details/{match_id} ELIMINADO
# ============================================================================
# Este endpoint fue MIGRADO a Go service para mejor performance:
# - Go lee directamente desde data/exports/ sin overhead de Python
# - Avatares se obtienen DURANTE el procesamiento de la demo (una sola vez)
# - Frontend ahora usa: http://localhost:8080/match-details/{matchID}
#
# RAZÓN: Python agregaba avatares en tiempo real (10 llamadas a Steam API 
# cada vez que alguien veía un match). Go los obtiene una sola vez al 
# procesar la demo y los guarda en players_summary.json.
# ============================================================================


@router.get("/steam/get-dashboard-stats")
async def get_dashboard_stats(request: Request) -> dict[str, Any]:
    """
    Endpoint OPTIMIZADO para el dashboard.
    
    - NO devuelve event_logs (evita transferir miles de eventos innecesarios)
    - Lee desde data/exports/ cuando existe (datos pre-calculados)
    - Cache con TTL de 1 hora (invalidado automáticamente al procesar demos)
    - Solo incluye stats agregadas: KDA, ADR, HS%, mapas, armas
    
    Resultado: Carga 10-20x más rápida que get-processed-demos
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    # 1. Verificar caché (se invalida automáticamente al procesar demos)
    cache_key = f"dashboard_stats:{steam_id}"
    cached = await redis.get(cache_key)
    if cached:
        logging.info(f"[dashboard-stats] Cache HIT para {steam_id}")
        return json.loads(cached)
    
    logging.info(f"[dashboard-stats] Cache MISS para {steam_id} - calculando...")
    
    # 2. Obtener lista de match_ids procesadas desde Redis
    processed_demos_raw: list[str] = await redis.lrange(f"processed_demos:{steam_id}", 0, -1)  # type: ignore
    
    if not processed_demos_raw:
        empty_response = {
            "steam_id": steam_id,
            "stats": {
                "total_matches": 0,
                "total_kills": 0,
                "total_deaths": 0,
                "avg_kd": 0.0,
                "avg_adr": 0.0,
                "avg_hs": 0.0,
                "win_rate": 0.0,
                "wins": 0,
                "losses": 0
            },
            "recent_matches": [],
            "weapon_stats": [],
            "map_stats": []
        }
        await redis.set(cache_key, json.dumps(empty_response), ex=3600)
        return empty_response
    
    # 3. Cargar datos OPTIMIZADOS desde data/exports/ o Redis
    exports_path = os.path.join(os.path.dirname(__file__), "..", "..", "data", "exports")
    matches_data = []
    weapon_kills: dict[str, int] = {}
    map_results: dict[str, dict[str, int]] = {}
    
    for demo_raw in processed_demos_raw:
        demo = json.loads(demo_raw)
        match_id = demo.get("match_id")
        
        # Intentar leer desde data/exports/ primero (más eficiente)
        players_summary_path = os.path.join(exports_path, str(match_id), "players_summary.json")
        match_info_path = os.path.join(exports_path, str(match_id), "match_info.json")
        combat_analytics_path = os.path.join(exports_path, str(match_id), "combat_analytics.json")
        
        player_stats = None
        match_info = None
        
        # Leer players_summary.json para stats del jugador
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
        
        # Leer match_info.json para resultado y mapa
        if os.path.exists(match_info_path):
            try:
                with open(match_info_path, 'r', encoding='utf-8') as f:
                    match_info = json.load(f)
            except Exception as e:
                logging.warning(f"Error leyendo {match_info_path}: {e}")
        
        # Fallback: usar datos de Redis si no existen exports
        if not player_stats:
            # Buscar en demo.players
            for p in demo.get("players", []):
                if str(p.get("steamID", "")) == str(steam_id):
                    player_stats = {
                        "kills": p.get("kills", 0),
                        "deaths": p.get("deaths", 0),
                        "assists": p.get("assists", 0),
                        "kd_ratio": p.get("kd_ratio", 0.0),
                        "adr": p.get("adr", 0.0),
                        "hs_percentage": p.get("hs_percentage", 0.0),
                        "team": p.get("team", "")
                    }
                    break
        
        if not match_info:
            match_info = {
                "map_name": demo.get("map_name", "unknown"),
                "date": demo.get("date", ""),
                "team_score": demo.get("team_score", 0),
                "opponent_score": demo.get("opponent_score", 0)
            }
        
        if player_stats:
            # Determinar resultado
            team_score = match_info.get("team_score", 0)
            opponent_score = match_info.get("opponent_score", 0)
            result = "W" if team_score > opponent_score else "L"
            
            # Agregar a matches_data
            matches_data.append({
                "match_id": match_id,
                "map": match_info.get("map_name", "unknown"),
                "date": match_info.get("date", ""),
                "result": result,
                "kills": player_stats.get("kills", 0),
                "deaths": player_stats.get("deaths", 0),
                "assists": player_stats.get("assists", 0),
                "kd_ratio": player_stats.get("kd_ratio", 0.0),
                "adr": player_stats.get("adr", 0.0),
                "hs_percentage": player_stats.get("hs_percentage", 0.0),
                "team_score": team_score,
                "opponent_score": opponent_score
            })
            
            # Actualizar map_stats
            map_name = match_info.get("map_name", "unknown")
            if map_name not in map_results:
                map_results[map_name] = {"wins": 0, "losses": 0}
            if result == "W":
                map_results[map_name]["wins"] += 1
            else:
                map_results[map_name]["losses"] += 1
        
        # Leer weapon_stats desde players_summary.json (exportado por Go service)
        if player_stats and "weapon_stats" in player_stats:
            for weapon, stats in player_stats["weapon_stats"].items():
                kills = stats.get("kills", 0)
                if kills > 0:
                    weapon_kills[weapon] = weapon_kills.get(weapon, 0) + kills
    
    # 4. Calcular estadísticas agregadas
    total_matches = len(matches_data)
    total_kills = sum(m["kills"] for m in matches_data)
    total_deaths = sum(m["deaths"] for m in matches_data)
    avg_kd = total_kills / total_deaths if total_deaths > 0 else 0.0
    avg_adr = sum(m["adr"] for m in matches_data) / total_matches if total_matches > 0 else 0.0
    avg_hs = sum(m["hs_percentage"] for m in matches_data) / total_matches if total_matches > 0 else 0.0
    wins = sum(1 for m in matches_data if m["result"] == "W")
    losses = total_matches - wins
    win_rate = (wins / total_matches * 100) if total_matches > 0 else 0.0
    
    # 5. Top 4 armas
    top_weapons = sorted(weapon_kills.items(), key=lambda x: x[1], reverse=True)[:4]
    weapon_stats_list = [{"weapon": w, "kills": k} for w, k in top_weapons]
    
    # 6. Stats por mapa
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
    
    # 7. Últimas 10 partidas (ordenadas por fecha)
    recent_matches = sorted(matches_data, key=lambda x: x["date"], reverse=True)[:10]
    
    # 8. Construir respuesta
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
    
    # 9. Guardar en caché (TTL: 1 hora como safety net)
    # El caché se invalida automáticamente al procesar una nueva partida
    await redis.set(cache_key, json.dumps(response), ex=3600)
    logging.info(f"[dashboard-stats] Calculado y cacheado para {steam_id} - {total_matches} partidas")
    
    return response