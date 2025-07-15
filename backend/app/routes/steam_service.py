# steam_service.py
# ----------------
# Rutas de FastAPI para gestionar sharecodes en CS:GO/CS2.
# Este módulo maneja aspectos como la validación de códigos,
# la comunicación con la API oficial de Steam para obtener
# match sharing codes, y el almacenamiento de estos códigos
# en Redis.

import os
import json
import logging
import httpx
import asyncio
import requests
import redis.asyncio as aioredis
from fastapi import APIRouter, Request, HTTPException, Form, Query

router = APIRouter()

# Configura el nivel de logs
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# Creamos una conexión a Redis en modo asíncrono
redis = aioredis.from_url("redis://localhost", decode_responses=True)

# Obtenemos la Steam API Key desde las variables de entorno
STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")

def get_steam_data(steam_id: str) -> dict:
    """
    Obtiene el avatar y el rango (CS2 Premiere) de un usuario mediante la API de Steam.
    """
    url_avatar = f"http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key={STEAM_API_KEY}&steamids={steam_id}"
    url_rank = f"https://api.steampowered.com/ICSGOPlayers_730/GetGamePersonalData/v1/?key={STEAM_API_KEY}&steamid={steam_id}"

    # Obtenemos el avatar
    try:
        avatar_response = requests.get(url_avatar).json()
        avatar_url = None
        if "response" in avatar_response and "players" in avatar_response["response"]:
            avatar_url = avatar_response["response"]["players"][0].get("avatarfull", None)
    except Exception as e:
        logging.error(f"Error al obtener el avatar para SteamID {steam_id}: {e}")
        avatar_url = None

    # Obtenemos el rango de CS2 Premiere
    try:
        rank_response = requests.get(url_rank).json()
        rank = None
        if "result" in rank_response and "rank_type" in rank_response["result"]:
            rank = rank_response["result"]["rank_type"]
    except Exception as e:
        logging.error(f"Error al obtener el rango para SteamID {steam_id}: {e}")
        rank = None

    return {"avatar": avatar_url, "rank": rank}


@router.post("/steam/save-steam-id")
async def save_steam_id(request: Request):
    """
    Guarda el Steam ID del usuario en Redis para su uso posterior.
    
    Returns:
        dict: Confirmación de que el Steam ID se ha guardado.
    """
    logging.info(f"🔍 Cookies recibidas: {request.cookies}")  # Verifica si la cookie session está en la petición

    # Obtenemos el Steam ID del usuario a partir de la cookie "session"
    steam_id = request.session.get("steam_id")            # ✅
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Guardamos el Steam ID en Redis
    await redis.sadd("all_steam_ids", steam_id)
    logging.info(f"✅ Steam ID {steam_id} agregado al set 'all_steam_ids'.")
    return {"message": "Steam ID guardado correctamente en Redis."}


@router.get("/steam/all-sharecodes")
async def get_all_sharecodes(
    request: Request,
    auth_code: str = Query(..., alias="auth_code"),
    known_code: str = Query(..., alias="known_code")
):
    """
    Obtiene todos los sharecodes siguientes a 'known_code' usando la API oficial de Steam.

    - Maneja errores 412 (sharecode inválido/caducado).
    - Si Steam devuelve 'n/a', se detiene la recolección.
    - Máximo 50 códigos por petición (para evitar timeouts).

    Returns:
        dict:
            - sharecodes (list[str]): Lista de sharecodes nuevos.
            - has_more (bool): Indica si probablemente existan más códigos.
    """
    steam_id = request.session.get("steam_id")            # ✅
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    if not STEAM_API_KEY:
        raise HTTPException(status_code=400, detail="Falta Steam API Key")

    current_code = known_code
    url = "https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/"
    sharecodes = []
    max_retries = 5
    delay_seconds = 0.2
    max_total_codes = 50
    original_code = known_code  # Preservar el último código conocido

    logging.info(f"[all-sharecodes] Iniciando para {steam_id}, código inicial: '{current_code}'")

    try:
        async with httpx.AsyncClient() as client:
            while True:
                # Preparamos los parámetros para la solicitud a Steam
                params = {
                    "key": STEAM_API_KEY,       # Nuestra clave de Steam
                    "steamid": steam_id,        # Steam ID del usuario
                    "steamidkey": auth_code,    # Auth code
                    "knowncode": current_code   # Código base para obtener el siguiente
                }

                success = False
                for attempt in range(max_retries):
                    resp = await client.get(url, params=params)
                    if resp.status_code == 412:
                        logging.error(f"CODE INVALIDO | Usuario: {steam_id} | Code: {current_code}")
                        raise HTTPException(
                            status_code=400,
                            detail="Código inválido o demo expirada (vida útil ~1 semana)."
                        )
                    if resp.status_code == 429:
                        logging.warning(f"RATELIMIT | Usuario: {steam_id} | Intento: {attempt+1}/{max_retries}")
                        await asyncio.sleep(delay_seconds * (attempt + 1))
                        continue
                    resp.raise_for_status()
                    data = resp.json()
                    success = True
                    break

                if not success:
                    raise HTTPException(
                        status_code=429,
                        detail="Límite de reintentos alcanzado. Intente en 5 minutos."
                    )

                next_code = data.get("result", {}).get("nextcode", "").strip()

                if not next_code or next_code.lower() in ("n/a", "null", "none"):
                    logging.info(f"FIN DE CÓDIGOS | Usuario: {steam_id} | Motivo: respuesta vacía/nula")
                    break

                sharecodes.append(next_code)
                current_code = next_code

                if len(sharecodes) >= max_total_codes:
                    logging.info(f"LÍMITE ALCANZADO | Usuario: {steam_id} | Códigos: {max_total_codes}")
                    break

                await asyncio.sleep(0.2)

        # Si obtuvimos nuevos sharecodes, actualizamos Redis
        if sharecodes:
            await redis.set(f"{steam_id}:authCode", auth_code)
            await redis.set(f"{steam_id}:knownCode", sharecodes[-1])
            logging.info(f"REDIS ACTUALIZADO | Usuario: {steam_id} | Nuevo código: {sharecodes[-1]}")
        else:
            await redis.set(f"{steam_id}:knownCode", original_code)
            logging.info(f"REDIS PRESERVADO | Usuario: {steam_id} | Código: {original_code}")

        return {
            "sharecodes": sharecodes,
            "has_more": len(sharecodes) == max_total_codes
        }

    except httpx.HTTPError as e:
        logging.error(f"ERROR DE CONEXIÓN | Usuario: {steam_id} | Error: {str(e)}")
        raise HTTPException(
            status_code=502,
            detail="Error de comunicación con los servidores de Steam. Intente nuevamente."
        )



@router.post("/steam/save-sharecodes")
async def save_sharecodes(request: Request):
    """
    Recibe sharecodes, auth_code y known_code, y los almacena en Redis.
    Se asume que el bot de Node.js está suscrito a los eventos de rpush y
    se encargará de iniciar la descarga de demos cuando se actualice la lista.
    """
    data = await request.json()
    sharecodes = data.get("sharecodes", [])
    auth_code = data.get("auth_code")
    known_code = data.get("known_code")

    steam_id = request.session.get("steam_id")            # ✅
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    if not sharecodes:
        raise HTTPException(status_code=400, detail="No se proporcionaron sharecodes.")
    if not auth_code or not known_code:
        raise HTTPException(status_code=400, detail="Faltan auth_code o known_code.")

    # Actualizamos la lista de sharecodes del usuario
    await redis.delete(f"sharecodes:{steam_id}")
    await redis.rpush(f"sharecodes:{steam_id}", *sharecodes)

    # Guardamos auth_code y known_code en Redis para futuros procesos
    await redis.set(f"{steam_id}:authCode", auth_code)
    await redis.set(f"{steam_id}:knownCode", known_code)

    logging.info(f"[save-sharecodes] Almacenados {len(sharecodes)} sharecodes en Redis para {steam_id}")

    # Se elimina la notificación manual al bot de Node.js,
    # ya que el bot se activa automáticamente al detectar el rpush.
    return {"message": "✅ ShareCodes guardados. El bot se activará automáticamente."}

@router.get("/steam/check-friend-status")
async def check_friend_status(request: Request):
    """
    Verifica en Redis o llama al servicio de Node.js para ver si
    el usuario (steam_id) es amigo del bot de Steam.
    """
    steam_id = request.session.get("steam_id")            # ✅
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    friend_status = await redis.get(f"friend_status:{steam_id}")
    if friend_status is not None:
        return {"is_friend": friend_status == "true"}

    async with httpx.AsyncClient() as client:
        response = await client.get(f"http://localhost:4000/steam/check-friend?steam_id={steam_id}")
        if response.status_code != 200:
            raise HTTPException(status_code=500, detail="Error verificando amistad.")
        friend_status = response.json().get("is_friend", False)
        await redis.set(f"friend_status:{steam_id}", "true" if friend_status else "false", ex=86400)

    return {"is_friend": friend_status}


@router.get("/steam/check-sharecodes")
async def check_sharecodes(request: Request):
    """
    Devuelve la lista de sharecodes que el usuario (steam_id) tiene en Redis,
    junto con un indicador booleano 'exists' si no está vacío.
    """
    steam_id = request.session.get("steam_id")            # ✅
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    existing_sharecodes = await redis.lrange(f"sharecodes:{steam_id}", 0, -1)
    return {"exists": bool(existing_sharecodes), "sharecodes": existing_sharecodes}


@router.get("/steam/get-processed-demos")
async def get_processed_demos(request: Request):
    """
    Retorna la lista de demos procesadas para el usuario autenticado,
    añadiendo el avatar y el rango a cada jugador en cada demo.
    """
    steam_id = request.session.get("steam_id")            # ✅
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    processed_demos = await redis.lrange(f"processed_demos:{steam_id}", 0, -1)
    if not processed_demos:
        raise HTTPException(status_code=404, detail="No hay partidas procesadas.")

    demos = [json.loads(demo) for demo in processed_demos]

    for demo in demos:
        for player in demo.get("players", []):
            steam_data = get_steam_data(player["steamID"])
            player["avatar"] = steam_data.get("avatar")
            player["rank"] = steam_data.get("rank")
    return {"steam_id": steam_id, "demos": demos}


@router.get("/steam/get-user-data")
async def get_user_data(request: Request):
    """
    Retorna el avatar y el rango del usuario autenticado.
    """
    steam_id = request.session.get("steam_id")            # ✅
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    steam_data = get_steam_data(steam_id)
    return {"steam_id": steam_id, "avatar": steam_data.get("avatar"), "rank": steam_data.get("rank")}
