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

router = APIRouter()

# Log básico
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# Redis cliente (routes-level)
redis = aioredis.from_url("redis://localhost", decode_responses=True)

def get_steam_data(steam_id: str) -> dict:
    """
    Obtiene avatar y rango (CS2 Premiere) vía API Steam.
    Lee la API key en tiempo de ejecución para evitar problemas de import.
    """
    STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")
    url_avatar = f"http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key={STEAM_API_KEY}&steamids={steam_id}"
    url_rank   = f"https://api.steampowered.com/ICSGOPlayers_730/GetGamePersonalData/v1/?key={STEAM_API_KEY}&steamid={steam_id}"

    avatar_url = None
    rank = None

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
async def save_steam_id(request: Request):
    """
    Guarda el Steam ID del usuario autenticado en Redis (set all_steam_ids).
    """
    logging.info(f"🔍 Cookies recibidas: {request.cookies}")
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    await redis.sadd("all_steam_ids", steam_id)
    logging.info(f"✅ Steam ID {steam_id} agregado al set 'all_steam_ids'.")
    return {"message": "Steam ID guardado correctamente en Redis."}


@router.get("/steam/all-sharecodes")
async def get_all_sharecodes(
    request: Request,
    auth_code: str = Query(..., alias="auth_code"),
    last_code: str | None = Query(None, alias="last_code"),
    known_code: str | None = Query(None, alias="known_code"),
):
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
    sharecodes = []
    max_retries = 5
    delay_seconds = 0.2
    max_total_codes = 50
    original_code = current_code
    resynced = False

    logging.info(f"[all-sharecodes] Usuario={steam_id} | code_inicial='{current_code}'")

    try:
        async with httpx.AsyncClient() as client:
            while True:
                success = False
                for attempt in range(max_retries):
                    # params en cada intento (respeta current_code actualizado)
                    params = {
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
async def save_sharecodes(request: Request):
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
    await redis.rpush(f"sharecodes:{steam_id}", *sharecodes)

    # Guarda auth_code y known_code
    await redis.set(f"{steam_id}:authCode", auth_code)
    await redis.set(f"{steam_id}:knownCode", known_code)

    logging.info(f"[save-sharecodes] Almacenados {len(sharecodes)} sharecodes en Redis para {steam_id}")
    return {"message": "✅ ShareCodes guardados. El bot se activará automáticamente."}


@router.get("/steam/check-friend-status")
async def check_friend_status(request: Request):
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
async def send_friend_request(request: Request):
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
async def check_sharecodes(request: Request):
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    existing_sharecodes = await redis.lrange(f"sharecodes:{steam_id}", 0, -1)
    return {"exists": bool(existing_sharecodes), "sharecodes": existing_sharecodes}


@router.get("/steam/get-processed-demos")
async def get_processed_demos(request: Request):
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    processed_demos = await redis.lrange(f"processed_demos:{steam_id}", 0, -1)
    if not processed_demos:
        raise HTTPException(status_code=404, detail="No hay partidas procesadas.")

    demos = [json.loads(demo) for demo in processed_demos]
    
    # DEBUG: Verificar si los event_logs están en los datos
    logging.info(f"DEBUG: Procesando {len(demos)} demos para steam_id {steam_id}")
    for i, demo in enumerate(demos):
        event_logs = demo.get("event_logs", [])
        logging.info(f"DEBUG: Demo {i} - event_logs type: {type(event_logs)}, length: {len(event_logs) if isinstance(event_logs, list) else 'not list'}")
        if len(event_logs) > 0:
            logging.info(f"DEBUG: Demo {i} - Primer evento: {event_logs[0]}")
    
    # Solo enriquecer datos del usuario autenticado (no de todos los jugadores)
    # Esto evita hacer 200+ llamadas a la API de Steam
    try:
        user_steam_data = get_steam_data(steam_id)
        for demo in demos:
            for player in demo.get("players", []):
                # Solo agregar avatar/rank al jugador autenticado
                if player.get("steamID") == steam_id:
                    player["avatar"] = user_steam_data.get("avatar")
                    player["rank"] = user_steam_data.get("rank")
    except Exception as e:
        # Si falla la API de Steam, no es crítico - continuar sin avatar/rank
        logging.warning(f"No se pudo obtener datos de Steam para {steam_id}: {e}")
    
    return {"steam_id": steam_id, "demos": demos}


@router.get("/steam/get-user-data")
async def get_user_data(request: Request):
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    steam_data = get_steam_data(steam_id)
    return {"steam_id": steam_id, "avatar": steam_data.get("avatar"), "rank": steam_data.get("rank")}