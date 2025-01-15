import os
from fastapi import APIRouter, Request, HTTPException, Form, Query
import redis.asyncio as aioredis
import httpx
import logging
import asyncio

router = APIRouter()
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# 🔄 Conexión a Redis
redis = aioredis.from_url("redis://localhost", decode_responses=True)

# 🔑 Cargar la API KEY desde el .env
STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")

@router.post("/steam/save-api-key")
async def save_api_key(request: Request, user_api_key: str = Form(...)):
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
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    url = "https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/"
    sharecodes = []
    current_code = last_code
    max_retries = 5
    delay_seconds = 0.2
    max_codes = 5  # Ajusta según cuántos ShareCodes quieras obtener

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
                        # Si hay demasiadas peticiones, esperamos y reintentamos
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
                    break
                sharecodes.append(next_code)
                current_code = next_code
                # Pequeña pausa para no saturar
                await asyncio.sleep(1)

        # 🔒 Guardar auth_code y last_code en Redis después de obtener los ShareCodes
        await redis.set(f"{steam_id}:authCode", auth_code)
        await redis.set(f"{steam_id}:lastCode", current_code)

        logging.info(f"🔒 AuthCode y LastCode guardados en Redis para Steam ID {steam_id}")

        return {"sharecodes": sharecodes, "has_more": len(sharecodes) == max_codes}

    except httpx.HTTPError as e:
        raise HTTPException(status_code=500, detail=f"Error al obtener los share codes: {str(e)}")

@router.post("/steam/save-sharecodes")
async def save_sharecodes(request: Request):
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

    # 🔒 Guardar ShareCodes en Redis
    await redis.delete(f"sharecodes:{steam_id}")
    await redis.rpush(f"sharecodes:{steam_id}", *sharecodes)

    # 🔑 Guardar auth_code y last_code en Redis
    await redis.set(f"{steam_id}:authCode", auth_code)
    await redis.set(f"{steam_id}:lastCode", last_code)

    logging.info(f"🔒 ShareCodes y códigos de autenticación guardados en Redis para Steam ID {steam_id}")

    # Cambiamos el mensaje final para que no mencione la descarga de demos
    return {"message": "✅ ShareCodes guardados y bot notificado para obtener estadísticas."}
