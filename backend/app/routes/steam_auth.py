from fastapi import APIRouter, Request, HTTPException, Query, Response
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


# ---------- LOGIN CALLBACK ----------
@router.get("/auth/steam/callback")
async def steam_callback(
    request: Request,
    openid_mode: str = Query(alias="openid.mode"),
    openid_claimed_id: str = Query(alias="openid.claimed_id"),
):
    if openid_mode != "id_res":
        raise HTTPException(status_code=400, detail="Modo OpenID inválido")

    if not openid_claimed_id:
        raise HTTPException(status_code=400, detail="'openid.claimed_id' ausente")

    # Guarda **solo** el número, no la URL completa
    steam_id = openid_claimed_id.split("/")[-1]
    if not steam_id.isdigit():
        raise HTTPException(status_code=400, detail="Steam ID inválido")

    request.session["steam_id"] = steam_id
    logging.info(f"🔐 Steam ID guardado en sesión: {steam_id}")

    return RedirectResponse(FRONTEND_SUCCESS_URL, status_code=303)


# ---------- STATUS ----------
@router.get("/auth/steam/status")
async def steam_status(request: Request):
    """Devuelve si el usuario está autenticado y, opcionalmente, su perfil."""
    steam_id = request.session.get("steam_id")            # ✅ sesión, no cookie
    if not steam_id:
        raise HTTPException(status_code=401, detail="No autenticado")

    steam_api_key = os.getenv("STEAM_API_KEY")
    if not steam_api_key:                      # sin API-Key devolvemos lo básico
        return {"authenticated": True, "steam_id": steam_id}

    url = (
        "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/"
        f"?key={steam_api_key}&steamids={steam_id}"
    )
    try:
        data = requests.get(url, timeout=5).json()["response"]["players"][0]
        return {
            "authenticated": True,
            "steam_id": steam_id,
            "username": data.get("personaname"),
            "avatar":   data.get("avatarfull"),
        }
    except Exception as e:
        logging.warning(f"Steam API error: {e}")
        return {"authenticated": True, "steam_id": steam_id}


# ---------- LOGOUT ----------
@router.post("/auth/steam/logout")
async def steam_logout(request: Request):
    """Vacía la sesión y cierra la cookie firmada."""
    request.session.clear()
    return {"message": "Sesión cerrada correctamente"}

