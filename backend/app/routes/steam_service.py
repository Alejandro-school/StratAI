from fastapi import APIRouter, Request, HTTPException, Form
import redis.asyncio as aioredis
import httpx
import logging

router = APIRouter()
logging.basicConfig(level=logging.INFO)

# 🔄 Conexión a Redis
redis = aioredis.from_url("redis://localhost", decode_responses=True)

STEAM_USER_STATS_URL = "https://api.steampowered.com/ISteamUserStats/GetUserStatsForGame/v2/"
APP_ID_CS2 = 730  # ID de CS2

# 🔑 Guardar la clave API en Redis
@router.post("/steam/save-api-key")
async def save_api_key(request: Request, user_api_key: str = Form(...)):
    steam_id = request.session.get("steam_id")
    
    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado")

    # Guardar la clave API asociada al Steam ID en Redis
    await redis.set(f"user_api_key:{steam_id}", user_api_key)
    logging.info(f"🔐 Clave API guardada para Steam ID {steam_id}")

    return {"message": "Clave API guardada exitosamente"}

# 📊 Obtener estadísticas desde la API de Steam
@router.get("/steam/match-history")
async def get_match_history(request: Request):
    steam_id = request.session.get("steam_id")

    if not steam_id:
        raise HTTPException(status_code=401, detail="Usuario no autenticado")

    # Recuperar la clave API desde Redis
    user_api_key = await redis.get(f"user_api_key:{steam_id}")

    if not user_api_key:
        raise HTTPException(status_code=401, detail="No se encontró una clave API en la sesión")

    # 🔎 Llamada a la API de Steam
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
