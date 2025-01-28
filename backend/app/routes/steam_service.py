"""
steam_service.py
----------------
Rutas de FastAPI relacionadas con la gestión de sharecodes,
verificación de amistad, API Key, y obtención de partidas procesadas.
Obtiene tantos sharecodes como sea posible (sin límite estricto),
encadenando 'GetNextMatchSharingCode', guardando en Redis.
"""

import os
import json
import logging
import httpx
import asyncio
import redis.asyncio as aioredis
from fastapi import APIRouter, Request, HTTPException, Form, Query

router = APIRouter()
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# Conexión a Redis
redis = aioredis.from_url("redis://localhost", decode_responses=True)

# Carga la Steam API Key desde .env
STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")

@router.post("/steam/save-api-key")
async def save_api_key(request: Request, user_api_key: str = Form(...)):
    """
    Guarda la clave de API de Steam por usuario (steam_id) en Redis.
    """
    steam_id = request.cookies.get("session")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    await redis.set(f"user_api_key:{steam_id}", user_api_key)
    logging.info(f"🔐 Clave API guardada para Steam ID {steam_id}")

    return {"message": "Clave API guardada e iniciado el bot exitosamente."}

@router.get("/steam/check-friend-status")
async def check_friend_status(request: Request):
    """
    Verifica si el bot es amigo del usuario en Steam.
    """
    steam_id = request.cookies.get("session")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    friend_status = await redis.get(f"friend_status:{steam_id}")
    if friend_status is not None:
        return {"is_friend": friend_status == "true"}

    async with httpx.AsyncClient() as client:
        response = await client.get(f"http://localhost:4000/steam/check-friend?steam_id={steam_id}")
        if response.status_code != 200:
            raise HTTPException(status_code=500, detail="Error verificando la amistad con el bot.")
        friend_status = response.json().get("is_friend", False)
        await redis.set(f"friend_status:{steam_id}", "true" if friend_status else "false", ex=86400)

    return {"is_friend": friend_status}

@router.get("/steam/check-sharecodes")
async def check_sharecodes(request: Request):
    """
    Comprueba si ya existen sharecodes en Redis para el usuario.
    """
    steam_id = request.cookies.get("session")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    existing_sharecodes = await redis.lrange(f"sharecodes:{steam_id}", 0, -1)
    if existing_sharecodes:
        return {"exists": True, "sharecodes": existing_sharecodes}
    else:
        return {"exists": False}

@router.get("/steam/all-sharecodes")
async def get_all_sharecodes(
    request: Request,
    auth_code: str = Query(...),
    last_code: str = Query(...)
):
    """
    Llama de forma secuencial a la API oficial de Steam (GetNextMatchSharingCode)
    a partir de un 'knowncode' y 'auth_code', para obtener todos los share codes
    que estén disponibles, sin tope estricto (aunque hay un top de 300 por seguridad).
    """
    steam_id = request.cookies.get("session")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Cargamos la Steam API Key (por si la guardas por usuario o global)
    # Si no la usas por usuario, simplemente usa STEAM_API_KEY global
    # steam_api_key = await redis.get(f"user_api_key:{steam_id}") or STEAM_API_KEY
    steam_api_key = STEAM_API_KEY
    if not steam_api_key:
        raise HTTPException(status_code=400, detail="Falta Steam API Key")

    url = "https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/"
    sharecodes = []
    current_code = last_code

    max_retries = 5
    delay_seconds = 0.1
    max_total_codes = 50  # Evitar un bucle infinito, 300 es un tope razonable

    logging.info(f"[all-sharecodes] Iniciando obtención masiva para steam_id={steam_id}, from={last_code}")

    try:
        async with httpx.AsyncClient() as client:
            while True:
                params = {
                    "key": steam_api_key,
                    "steamid": steam_id,
                    "steamidkey": auth_code,
                    "knowncode": current_code
                }
                success = False
                for attempt in range(max_retries):
                    resp = await client.get(url, params=params)
                    if resp.status_code == 429:
                        logging.warning("Recibido 429, esperando %.1f s...", delay_seconds)
                        await asyncio.sleep(delay_seconds)
                        delay_seconds *= 1
                        continue
                    resp.raise_for_status()
                    data = resp.json()
                    success = True
                    break
                if not success:
                    # No logramos obtener una respuesta 2xx tras max_retries
                    raise HTTPException(
                        status_code=429,
                        detail="Demasiadas solicitudes a la API de Steam. Intenta luego."
                    )

                next_code = data.get("result", {}).get("nextcode")
                if not next_code:
                    # Significa que no hay más
                    break

                sharecodes.append(next_code)
                current_code = next_code

                # Delay para no saturar
                await asyncio.sleep(1)

                if len(sharecodes) >= max_total_codes:
                    logging.info("Se alcanzó el tope de %d sharecodes. Detenemos.", max_total_codes)
                    break

        # Guardamos en Redis la info
        await redis.set(f"{steam_id}:authCode", auth_code)
        await redis.set(f"{steam_id}:lastCode", current_code)

        logging.info(f"[all-sharecodes] Obtenidos {len(sharecodes)} sharecodes para {steam_id}")

        return {"sharecodes": sharecodes, "has_more": (len(sharecodes) == max_total_codes)}

    except httpx.HTTPError as e:
        raise HTTPException(status_code=500, detail=f"Error al obtener los sharecodes: {str(e)}")

@router.post("/steam/save-sharecodes")
async def save_sharecodes(request: Request):
    """
    Recibe un JSON con 'sharecodes', 'auth_code' y 'last_code'.
    Los guarda en Redis y notifica al bot Node.js para descargar .dem.
    """
    data = await request.json()
    sharecodes = data.get("sharecodes", [])
    auth_code = data.get("auth_code")
    last_code = data.get("last_code")
    steam_id = request.cookies.get("session")

    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    if not sharecodes:
        raise HTTPException(status_code=400, detail="No se proporcionaron share codes.")
    if not auth_code or not last_code:
        raise HTTPException(status_code=400, detail="No se proporcionaron códigos de autenticación.")

    # Borramos la lista anterior y guardamos la nueva
    await redis.delete(f"sharecodes:{steam_id}")
    await redis.rpush(f"sharecodes:{steam_id}", *sharecodes)
    await redis.set(f"{steam_id}:authCode", auth_code)
    await redis.set(f"{steam_id}:lastCode", last_code)

    logging.info(f"[save-sharecodes] Se almacenaron {len(sharecodes)} sharecodes en Redis para {steam_id}")

    # Notificar a Node.js
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                "http://localhost:4000/start-download",
                json={"steam_id": steam_id, "sharecodes": sharecodes}
            )
            if response.status_code != 200:
                logging.error("❌ Error al notificar al bot Node para descargar.")
    except Exception as e:
        logging.error(f"❌ Error de conexión con el bot Node.js: {str(e)}")

    return {"message": "✅ ShareCodes guardados y bot notificado para descargar demos."}

@router.get("/steam/get-processed-demos")
async def get_processed_demos(request: Request):
    """
    Devuelve las partidas procesadas (estadísticas) almacenadas en Redis.
    """
    steam_id = request.cookies.get("session")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    processed_demos = await redis.lrange(f"processed_demos:{steam_id}", 0, -1)
    if not processed_demos:
        raise HTTPException(status_code=404, detail="No hay partidas procesadas disponibles.")

    demos = [json.loads(demo) for demo in processed_demos]
    return {"steam_id": steam_id, "demos": demos}


@router.post("/steam/save-steam-id")
async def save_steam_id(request: Request):
    """
    Guarda automáticamente el Steam ID en el set de Redis "all_steam_ids"
    cuando el usuario inicia sesión, evitando duplicados.
    """
    steam_id = request.cookies.get("session")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Verificar si la ID ya existe en Redis
    is_member = await redis.sismember("all_steam_ids", steam_id)
    if is_member:
        logging.info(f"⚠️ Steam ID {steam_id} ya está registrado en all_steam_ids.")
        return {"message": "Steam ID ya estaba registrado."}

    # Agregar el steam_id al set de IDs activos
    await redis.sadd("all_steam_ids", steam_id)
    logging.info(f"✅ Steam ID {steam_id} guardado en all_steam_ids.")

    return {"message": "Steam ID registrado correctamente."}

