import os
from fastapi import APIRouter, Request, HTTPException, Form, Query
import redis.asyncio as aioredis
import httpx
import logging

router = APIRouter()
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

# 🔄 Conexión a Redis
redis = aioredis.from_url("redis://localhost", decode_responses=True)

# 🔑 Cargar la API KEY desde el .env
STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")

# (Opcional) La “vieja” URL de stats, si quieres usarla
STEAM_USER_STATS_URL = "https://api.steampowered.com/ISteamUserStats/GetUserStatsForGame/v2/"
APP_ID_CS2 = 730  # ID de CS2

### -----------------------------------------------------
### EJEMPLO 1: Guardar una API KEY por usuario (opcional)
### -----------------------------------------------------
@router.post("/steam/save-api-key")
async def save_api_key(request: Request, user_api_key: str = Form(...)):
    steam_id = request.session.get("steam_id")
    
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Guardar la clave API asociada al Steam ID en Redis
    await redis.set(f"user_api_key:{steam_id}", user_api_key)
    logging.info(f"🔐 Clave API guardada para Steam ID {steam_id}")

    return {"message": "Clave API guardada exitosamente"}


@router.get("/steam/match-history")
async def get_match_history(request: Request):
    """
    Ejemplo de cómo usar la API KEY y el steam_id para obtener stats 
    (si usas la API oficial de Steam).
    """
    steam_id = request.session.get("steam_id")

    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Recuperar la clave API desde Redis
    user_api_key = await redis.get(f"user_api_key:{steam_id}")
    if not user_api_key:
        raise HTTPException(status_code=401, detail="No se encontró una clave API para este usuario en Redis.")

    # Llamada a la API de Steam
    params = {
        "key": user_api_key,
        "steamid": steam_id,
        "appid": APP_ID_CS2
    }

    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(STEAM_USER_STATS_URL, params=params)
            response.raise_for_status()
            data = response.json()

        if "playerstats" not in data or "stats" not in data["playerstats"]:
            raise HTTPException(status_code=404, detail="No se encontraron estadísticas para este usuario.")

        return {"stats": data["playerstats"]["stats"]}

    except httpx.HTTPError as e:
        logging.error(f"❌ Error al conectarse a la API de Steam: {str(e)}")
        raise HTTPException(status_code=500, detail="Error al conectarse a la API de Steam.")


### -----------------------------------------------------
### EJEMPLO 2: Guardar y usar un auth_code manualmente
### -----------------------------------------------------
@router.post("/steam/save-auth-code")
async def save_auth_code(request: Request):
    data = await request.json()
    auth_code = data.get("auth_code")
    
    steam_id = request.session.get("steam_id")

    if not steam_id:
        logging.error("❌ Usuario no autenticado. steam_id no encontrado en la sesión.")
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    
    if not auth_code:
        logging.error("❌ Código de autenticación no proporcionado.")
        raise HTTPException(status_code=400, detail="Código de autenticación no proporcionado.")
    
    # Guardamos el code en Redis
    await redis.set(f"auth_code:{steam_id}", auth_code)
    logging.info(f"🔐 Código de autenticación guardado para Steam ID {steam_id}")

    return {"message": "Código de autenticación guardado correctamente."}

    
@router.get("/steam/all-sharecodes")
async def get_all_sharecodes(request: Request, auth_code: str = Query(...), last_code: str = Query(...)):
    """
    Obtiene hasta 30 share codes de manera progresiva, con pausas para evitar límites de la API.
    """
    import asyncio
    steam_id = request.session.get("steam_id")

    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    url = "https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1/"
    sharecodes = []
    current_code = last_code
    max_retries = 5  # Número máximo de reintentos
    delay_seconds = 0.2  # Retardo inicial entre peticiones
    max_codes = 5  # Límite de códigos por sesión

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
                        await asyncio.sleep(delay_seconds)
                        delay_seconds *= 2  # Incrementa el tiempo de espera exponencialmente
                        continue  # Reintentar la solicitud
                    
                    response.raise_for_status()
                    data = response.json()
                    break
                else:
                    raise HTTPException(status_code=429, detail="Demasiadas solicitudes. Intenta nuevamente más tarde.")
                
                next_code = data.get("result", {}).get("nextcode")
                if not next_code:
                    break  # No hay más códigos disponibles
                
                sharecodes.append(next_code)
                current_code = next_code  # Actualizar el último código
                await asyncio.sleep(1)  # Retardo fijo para prevenir límites

        return {
            "sharecodes": sharecodes,
            "has_more": len(sharecodes) == max_codes  # Indica si hay más códigos por extraer
        }

    except httpx.HTTPError as e:
        raise HTTPException(status_code=500, detail=f"Error al obtener los share codes: {str(e)}")
    
@router.post("/steam/process-demos")
async def process_demos(request: Request):
    data = await request.json()
    sharecodes = data.get("sharecodes", [])
    steam_id = request.session.get("steam_id")  # ✅ Obtener el Steam ID

    if not sharecodes:
        raise HTTPException(status_code=400, detail="No se proporcionaron share codes.")

    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    processed_stats = []

    async with httpx.AsyncClient() as client:
        for code in sharecodes:
            try:
                # ✅ Enviar el Steam ID junto con el share code
                response = await client.post(
                    "http://localhost:8080/process-demo",
                    json={"share_code": code, "steam_id": steam_id}
                )
                response.raise_for_status()
                stats = response.json()
                processed_stats.append({"share_code": code, "stats": stats})
                print(f"✅ Share code procesado: {code}")
            except httpx.HTTPError as e:
                processed_stats.append({"share_code": code, "error": str(e)})
                print(f"❌ Error al procesar {code}: {e}")

    return {"processed_stats": processed_stats}


@router.post("/steam/save-sharecodes")
async def save_sharecodes(request: Request):
    """
    Guarda los share codes en Redis vinculados al Steam ID del usuario.
    """
    data = await request.json()
    sharecodes = data.get("sharecodes", [])
    steam_id = request.session.get("steam_id")

    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    if not sharecodes:
        raise HTTPException(status_code=400, detail="No se proporcionaron share codes.")

    # Guardar los share codes como una lista en Redis
    await redis.delete(f"sharecodes:{steam_id}")  # Limpiar previos
    await redis.rpush(f"sharecodes:{steam_id}", *sharecodes)

    return {"message": "Share codes guardados exitosamente."}

@router.get("/steam/get-saved-sharecodes")
async def get_saved_sharecodes(request: Request):
    """
    Devuelve los share codes guardados para el usuario autenticado.
    """
    steam_id = request.session.get("steam_id")

    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")

    # Recuperar los share codes de Redis
    sharecodes = await redis.lrange(f"sharecodes:{steam_id}", 0, -1)

    return {"sharecodes": sharecodes}





