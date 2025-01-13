from fastapi import APIRouter, Request, HTTPException, Query
from starlette.responses import RedirectResponse
import urllib.parse
import requests
import os
import logging
from dotenv import load_dotenv


load_dotenv()
STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")


router = APIRouter()
logging.basicConfig(level=logging.INFO)

# URL base del OpenID de Steam para el flujo de autenticación.
STEAM_OPENID_URL = "https://steamcommunity.com/openid/login"

# URL de retorno (callback) en tu backend, donde Steam enviará la respuesta.
CALLBACK_URL = "http://localhost:8000/auth/steam/callback"

# URL del frontend al que redirigirás tras un login exitoso.
FRONTEND_SUCCESS_URL = "http://localhost:3000/steam-login-success"



@router.get("/auth/steam/login")
async def steam_login(request: Request):
    """
    Inicia el flujo de login con Steam mediante OpenID.
    
    1. Construye los parámetros necesarios para la petición de autenticación.
    2. Redirige al usuario a la URL de Steam (STEAM_OPENID_URL) junto a dichos parámetros
    
    Al finalizar, Steam redirige al CALLBACK_URL (definido arriba).
    """
    params = {
        "openid.ns":         "http://specs.openid.net/auth/2.0",
        "openid.mode":       "checkid_setup",
        "openid.return_to":  CALLBACK_URL,
        "openid.realm":      CALLBACK_URL,
        "openid.identity":   "http://specs.openid.net/auth/2.0/identifier_select",
        "openid.claimed_id": "http://specs.openid.net/auth/2.0/identifier_select",
    }
    query_string = urllib.parse.urlencode(params)
    redirect_url = f"{STEAM_OPENID_URL}?{query_string}"
    return RedirectResponse(url=redirect_url)


@router.get("/auth/steam/callback")
async def steam_callback(request: Request, openid_mode: str = Query(None, alias="openid.mode"), openid_claimed_id: str = Query(None, alias="openid.claimed_id")):
    if openid_mode != "id_res":
        logging.error(f"❌ OpenID mode inválido: {openid_mode}")
        raise HTTPException(status_code=400, detail=f"Modo OpenID inválido. Recibido: {openid_mode}")

    if not openid_claimed_id:
        logging.error("❌ 'openid.claimed_id' no recibido.")
        raise HTTPException(status_code=400, detail="No se recibió 'openid.claimed_id'.")

    steam_id = openid_claimed_id.split("/")[-1]
    logging.info(f"🔐 Steam ID extraído: {steam_id}")

    if not steam_id.isdigit():
        logging.error(f"❌ Steam ID inválido: {steam_id}")
        raise HTTPException(status_code=400, detail="Steam ID inválido.")

    request.session["steam_id"] = steam_id
    logging.info(f"✅ Steam ID guardado en la sesión: {steam_id}")

    return RedirectResponse(url=FRONTEND_SUCCESS_URL)


# Validación de la clave API
if not STEAM_API_KEY:
    logging.error("❌ La clave STEAM_API_KEY no está configurada.")
    raise RuntimeError("La clave STEAM_API_KEY no está configurada en el entorno.")


@router.get("/auth/steam/status")
async def steam_status(request: Request):
    """
    Devuelve información sobre el perfil de Steam (nickname, avatar, steam_id) 
    si el usuario está autenticado y su SteamID está en sesión.
    
    Para ello:
      1. Verifica si hay un steam_id en la sesión.
      2. Llama a la API de Steam (GetPlayerSummaries) usando STEAM_API_KEY.
      3. Retorna un JSON con la info del jugador si todo va bien.

    Errores:
      - 401 si no hay steam_id en sesión (no autenticado).
      - 404 si no se encuentra el perfil en la API de Steam.
      - 500 si ocurre un error de conexión con la API.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="No autenticado")

    steam_api_key = os.getenv("STEAM_API_KEY")
    if not steam_api_key:
        raise HTTPException(status_code=500, detail="Falta STEAM_API_KEY en configuración.")

    steam_api_url = (
        "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/"
        f"?key={steam_api_key}&steamids={steam_id}"
    )

    try:
        response = requests.get(steam_api_url)
        data = response.json()

        if (
            "response" not in data
            or "players" not in data["response"]
            or len(data["response"]["players"]) == 0
        ):
            raise HTTPException(status_code=404, detail="No se pudo obtener la información del perfil")

        player = data["response"]["players"][0]
        return {
            "authenticated": True,
            "username": player.get("personaname", "Usuario desconocido"),
            "avatar": player.get("avatar", ""),
            "steam_id": steam_id
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error al obtener el perfil: {str(e)}")
