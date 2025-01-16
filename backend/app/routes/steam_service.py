import os
from fastapi import APIRouter, Request, HTTPException, Form, Query
import redis.asyncio as aioredis
import httpx
import logging
import asyncio

router = APIRouter()
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# Conexión a Redis (ajusta la URL a tu entorno)
redis = aioredis.from_url("redis://localhost", decode_responses=True)

# Carga la API KEY de Steam desde .env
STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")

@router.post("/steam/save-api-key")
async def save_api_key(request: Request, user_api_key: str = Form(...)):
    """
    Guarda la clave de API de Steam por usuario (steam_id) en Redis.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    await redis.set(f"user_api_key:{steam_id}", user_api_key)
    logging.info(f"🔐 Clave API guardada para Steam ID {steam_id}")

    return {"message": "Clave API guardada e iniciado el bot exitosamente."}

@router.get("/steam/all-sharecodes")
async def get_all_sharecodes(
    request: Request,
    auth_code: str = Query(...),
    last_code: str = Query(...)
):
    """
    Llama a la API oficial de Steam (GetNextMatchSharingCode) para obtener
    share codes de manera secuencial.
    Guarda auth_code y last_code en Redis.
    Devuelve hasta 5 share codes nuevos (ejemplo).
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    url = "https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/"
    sharecodes = []
    current_code = last_code

    # Ajusta límites de reintentos y backoff
    max_retries = 5
    delay_seconds = 0.2
    max_codes = 5

    try:
        async with httpx.AsyncClient() as client:
            while len(sharecodes) < max_codes:
                params = {
                    "key": STEAM_API_KEY,
                    "steamid": steam_id,
                    "steamidkey": auth_code,
                    "knowncode": current_code
                }
                for attempt in range(max_retries):
                    response = await client.get(url, params=params)
                    if response.status_code == 429:
                        # Too many requests, backoff
                        await asyncio.sleep(delay_seconds)
                        delay_seconds *= 2
                        continue
                    response.raise_for_status()
                    data = response.json()
                    break
                else:
                    raise HTTPException(status_code=429, detail="Demasiadas solicitudes. Intenta nuevamente más tarde.")
                
                next_code = data.get("result", {}).get("nextcode")
                if not next_code:
                    # Ya no hay más share codes
                    break

                sharecodes.append(next_code)
                current_code = next_code
                # Pequeño delay para no abusar
                await asyncio.sleep(1)

        # Guardamos authCode y lastCode en Redis
        await redis.set(f"{steam_id}:authCode", auth_code)
        await redis.set(f"{steam_id}:lastCode", current_code)

        logging.info(f"🔒 AuthCode y LastCode guardados en Redis para Steam ID {steam_id}")

        return {"sharecodes": sharecodes, "has_more": len(sharecodes) == max_codes}

    except httpx.HTTPError as e:
        raise HTTPException(status_code=500, detail=f"Error al obtener los share codes: {str(e)}")

@router.post("/steam/save-sharecodes")
async def save_sharecodes(request: Request):
    """
    Endpoint que recibe un JSON con 'sharecodes', 'auth_code' y 'last_code',
    los guarda en Redis y notifica al bot (Node.js) para que inicie la descarga.
    """
    data = await request.json()
    sharecodes = data.get("sharecodes", [])
    auth_code = data.get("auth_code")
    last_code = data.get("last_code")
    steam_id = request.session.get("steam_id")

    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    if not sharecodes:
        raise HTTPException(status_code=400, detail="No se proporcionaron share codes.")
    if not auth_code or not last_code:
        raise HTTPException(status_code=400, detail="No se proporcionaron códigos de autenticación.")

    # Borramos la lista de sharecodes previa y guardamos la nueva
    await redis.delete(f"sharecodes:{steam_id}")
    await redis.rpush(f"sharecodes:{steam_id}", *sharecodes)

    # Guardamos en Redis el auth_code y last_code
    await redis.set(f"{steam_id}:authCode", auth_code)
    await redis.set(f"{steam_id}:lastCode", last_code)

    # Notificamos a Node.js (opcional si quieres forzar descarga inmediata)
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                "http://localhost:3000/start-download",
                json={
                    "steam_id": steam_id,
                    "sharecodes": sharecodes
                }
            )
            if response.status_code != 200:
                logging.error("❌ Error al notificar al bot para iniciar la descarga.")
    except Exception as e:
        logging.error(f"❌ Error en la conexión con el bot: {str(e)}")

    return {"message": "✅ ShareCodes guardados y bot notificado para descargar demos."}
