from fastapi import APIRouter, Request, HTTPException, Query
from starlette.responses import RedirectResponse
import urllib.parse
import requests
import os
import logging

router = APIRouter()
logging.basicConfig(level=logging.INFO)

STEAM_OPENID_URL = "https://steamcommunity.com/openid/login"
CALLBACK_URL = "http://localhost:8000/auth/steam/callback"
FRONTEND_SUCCESS_URL = "http://localhost:3000/steam-login-success"

@router.get("/auth/steam/login")
async def steam_login(request: Request):
    """
    Inicia el flujo de login de Steam redirigiendo al usuario a Steam OpenID.
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
    """
    Recibe la respuesta de Steam, valida los parámetros y guarda el steam_id en sesión.
    """
    if openid_mode != "id_res":
        raise HTTPException(status_code=400, detail=f"Modo OpenID inválido. Recibido: {openid_mode}")

    if not openid_claimed_id:
        raise HTTPException(status_code=400, detail="No se recibió 'openid.claimed_id'.")

    steam_id = openid_claimed_id.split("/")[-1]
    if not steam_id.isdigit():
        raise HTTPException(status_code=400, detail="Steam ID inválido.")

    request.session["steam_id"] = steam_id
    logging.info(f"Steam ID guardado en sesión: {steam_id}")
    return RedirectResponse(url=FRONTEND_SUCCESS_URL)

@router.get("/auth/steam/status")
async def steam_status(request: Request):
    """
    Retorna información del perfil de Steam almacenado en sesión.
    """
    steam_id = request.session.get("steam_id")
    if not steam_id:
        raise HTTPException(status_code=401, detail="No autenticado")

    steam_api_key = os.getenv("STEAM_API_KEY")
    steam_api_url = f"https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key={steam_api_key}&steamids={steam_id}"

    try:
        response = requests.get(steam_api_url)
        data = response.json()

        if "response" not in data or "players" not in data["response"] or len(data["response"]["players"]) == 0:
            raise HTTPException(status_code=404, detail="No se pudo obtener la información del perfil")

        player = data["response"]["players"][0]
        return {
            "authenticated": True,
            "username": player.get("personaname", "Usuario desconocido"),
            "avatar": player.get("avatarfull", ""),
            "steam_id": steam_id
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error al obtener el perfil: {str(e)}")
