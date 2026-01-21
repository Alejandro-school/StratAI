from fastapi import APIRouter, Request, HTTPException, Query
from starlette.responses import RedirectResponse
import urllib.parse
import requests
import os
import logging
import re
from dotenv import load_dotenv
from typing import Any


load_dotenv()
STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")


router = APIRouter()
logging.basicConfig(level=logging.INFO)

# URL base del OpenID de Steam
STEAM_OPENID_URL = "https://steamcommunity.com/openid/login"

def get_base_url(request: Request) -> str:
    """Returns the base URL for BACKEND callbacks (e.g., port 8000 or tunnel)."""
    # X-Forwarded-Host can be a comma-separated list if multiple proxies are used
    forwarded_host = request.headers.get("x-forwarded-host")
    if forwarded_host:
        host = forwarded_host.split(",")[0].strip()
    else:
        host = request.headers.get("host", "localhost:8000")
    
    forwarded_proto = request.headers.get("x-forwarded-proto")
    if forwarded_proto:
        scheme = forwarded_proto.split(",")[0].strip()
    else:
        scheme = request.url.scheme
    
    # --- AGGRESSIVE SANITIZATION (Bulletproof V2) ---
    # Strip any protocol-like prefix RECURSIVELY (handles https://http:// etc.)
    clean_host = host
    while re.match(r'^https?[:/]+', clean_host, re.IGNORECASE):
        clean_host = re.sub(r'^https?[:/]+', '', clean_host, flags=re.IGNORECASE)
        
    # Strip everything except letters from the scheme
    clean_scheme = re.sub(r'[^a-zA-Z]', '', scheme).lower()
    
    base_url = f"{clean_scheme}://{clean_host}"
    
    # Print for terminal visibility (easier for user to see)
    print(f"\n[STEAM_AUTH] DEBUG URL CONSTRUCTION:")
    print(f"  - Raw Host Header: {request.headers.get('host')}")
    print(f"  - Raw X-Forwarded-Host: {forwarded_host}")
    print(f"  - Raw X-Forwarded-Proto: {forwarded_proto}")
    print(f"  - Cleaned Scheme: {clean_scheme}")
    print(f"  - Cleaned Host: {clean_host}")
    print(f"  - FINAL BASE URL: {base_url}\n")
    
    return base_url

def get_frontend_url(request: Request) -> str:
    """Returns the base URL for FRONTEND redirects."""
    forwarded_host = request.headers.get("x-forwarded-host")
    if forwarded_host:
        host = forwarded_host.split(",")[0].strip()
        clean_host = host
        while re.match(r'^https?[:/]+', clean_host, re.IGNORECASE):
            clean_host = re.sub(r'^https?[:/]+', '', clean_host, flags=re.IGNORECASE)
            
        forwarded_proto = request.headers.get("x-forwarded-proto", "https")
        scheme = forwarded_proto.split(",")[0].strip()
        clean_scheme = re.sub(r'[^a-zA-Z]', '', scheme).lower()
            
        return f"{clean_scheme}://{clean_host}"
    
    # Si no hay proxy, asumimos desarrollo local (puerto 3000)
    return "http://localhost:3000"

@router.get("/auth/steam/login")
async def steam_login(request: Request):
    """
    Inicia el flujo de login con Steam mediante OpenID.
    Dynamically constructs callback URL based on the request host.
    """
    logging.info("--- STEAM LOGIN START ---")
    logging.info(f"Headers: {dict(request.headers)}")
    
    base_url = get_base_url(request)
    callback_url = f"{base_url}/auth/steam/callback"
    realm_url = base_url
    
    logging.info(f"Callback URL: {callback_url}")
    logging.info(f"Realm URL: {realm_url}")
    
    params = {
        "openid.ns":         "http://specs.openid.net/auth/2.0",
        "openid.mode":       "checkid_setup",
        "openid.return_to":  callback_url,
        "openid.realm":      realm_url,
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

    # Redirigir al frontend usando la URL adecuada (3000 en local, túnel en remoto)
    frontend_base = get_frontend_url(request)
    frontend_success_url = f"{frontend_base}/steam-login-success"

    return RedirectResponse(frontend_success_url, status_code=303)


# ---------- STATUS ----------
@router.get("/auth/steam/status")
async def steam_status(request: Request) -> dict[str, Any]:
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

