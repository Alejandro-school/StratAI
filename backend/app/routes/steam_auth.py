import hashlib
import logging
import secrets
import urllib.parse
from datetime import datetime, timezone
from typing import Any

import httpx
import redis.asyncio as aioredis
from fastapi import APIRouter, HTTPException, Request
from starlette.responses import RedirectResponse

from ..config import (
    FRONTEND_URL,
    PIPELINE_NAMESPACE,
    PUBLIC_BACKEND_URL,
    REDIS_URL,
    STEAM_API_KEY,
)
from ..middleware.rate_limit import get_rate_limiter

router = APIRouter()
redis = aioredis.from_url(REDIS_URL, decode_responses=True)
rate_limiter = get_rate_limiter(REDIS_URL)

STEAM_OPENID_URL = "https://steamcommunity.com/openid/login"
STEAM_IDENTITY_PREFIX = "https://steamcommunity.com/openid/id/"
OPENID_NAMESPACE = "http://specs.openid.net/auth/2.0"
OPENID_REQUIRED_SIGNED_FIELDS = {
    "op_endpoint",
    "return_to",
    "response_nonce",
    "assoc_handle",
    "claimed_id",
    "identity",
}


def _callback_url(state: str) -> str:
    query = urllib.parse.urlencode({"state": state})
    return f"{PUBLIC_BACKEND_URL}/auth/steam/callback?{query}"


def _state_key(state: str) -> str:
    return f"{PIPELINE_NAMESPACE}:openid:state:{hashlib.sha256(state.encode()).hexdigest()}"


async def _verify_openid_response(request: Request, expected_return_to: str) -> str:
    params = dict(request.query_params)
    if params.get("openid.ns") != OPENID_NAMESPACE:
        raise HTTPException(status_code=400, detail="Versión OpenID inválida")
    if params.get("openid.mode") != "id_res":
        raise HTTPException(status_code=400, detail="Modo OpenID inválido")
    if params.get("openid.op_endpoint") != STEAM_OPENID_URL:
        raise HTTPException(status_code=400, detail="Proveedor OpenID inválido")
    if params.get("openid.return_to") != expected_return_to:
        raise HTTPException(status_code=400, detail="Destino OpenID inválido")

    claimed_id = params.get("openid.claimed_id", "")
    identity = params.get("openid.identity", "")
    if identity != claimed_id or not claimed_id.startswith(STEAM_IDENTITY_PREFIX):
        raise HTTPException(status_code=400, detail="Identidad OpenID inválida")

    signed_fields = set(params.get("openid.signed", "").split(","))
    if (
        not params.get("openid.sig")
        or not params.get("openid.assoc_handle")
        or not OPENID_REQUIRED_SIGNED_FIELDS.issubset(signed_fields)
        or any(not params.get(f"openid.{field}") for field in OPENID_REQUIRED_SIGNED_FIELDS)
    ):
        raise HTTPException(status_code=400, detail="Respuesta OpenID incompleta")

    response_nonce = params.get("openid.response_nonce", "")
    if not response_nonce:
        raise HTTPException(status_code=400, detail="Nonce OpenID ausente")
    try:
        issued_at = datetime.strptime(response_nonce[:20], "%Y-%m-%dT%H:%M:%SZ").replace(
            tzinfo=timezone.utc
        )
    except ValueError as exc:
        raise HTTPException(status_code=400, detail="Nonce OpenID inválido") from exc
    if abs((datetime.now(timezone.utc) - issued_at).total_seconds()) > 600:
        raise HTTPException(status_code=400, detail="Nonce OpenID caducado")
    nonce_digest = hashlib.sha256(response_nonce.encode()).hexdigest()
    nonce_key = f"{PIPELINE_NAMESPACE}:openid:nonce:{nonce_digest}"
    if not await redis.set(nonce_key, "1", ex=600, nx=True):
        raise HTTPException(status_code=409, detail="Respuesta OpenID reutilizada")

    verify_params = {**params, "openid.mode": "check_authentication"}
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            response = await client.post(STEAM_OPENID_URL, data=verify_params)
            response.raise_for_status()
    except httpx.HTTPError as exc:
        await redis.delete(nonce_key)
        logging.error("OpenID verification request failed: %s", type(exc).__name__)
        raise HTTPException(status_code=502, detail="Error verificando con Steam") from exc

    verification = {
        line.split(":", 1)[0]: line.split(":", 1)[1]
        for line in response.text.splitlines()
        if ":" in line
    }
    if verification.get("is_valid") != "true":
        raise HTTPException(status_code=403, detail="Verificación OpenID fallida")

    steam_id = claimed_id.removeprefix(STEAM_IDENTITY_PREFIX)
    if len(steam_id) != 17 or not steam_id.isdigit():
        raise HTTPException(status_code=400, detail="Steam ID inválido")
    return steam_id


async def _load_profile(steam_id: str) -> tuple[str, str]:
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.get(
                "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/",
                params={"key": STEAM_API_KEY, "steamids": steam_id},
            )
            response.raise_for_status()
        player = response.json()["response"]["players"][0]
        return player.get("personaname", ""), player.get("avatarfull", "")
    except (httpx.HTTPError, KeyError, IndexError, ValueError) as exc:
        logging.warning("Could not load Steam profile: %s", type(exc).__name__)
        return "", ""


@router.get("/auth/steam/login")
@rate_limiter.limit(10, 60)
async def steam_login(request: Request) -> RedirectResponse:
    state = secrets.token_urlsafe(32)
    if not await redis.set(_state_key(state), "1", ex=600, nx=True):
        raise HTTPException(status_code=503, detail="No se pudo iniciar la autenticación")
    request.session["openid_state"] = state
    callback_url = _callback_url(state)
    params = {
        "openid.ns": OPENID_NAMESPACE,
        "openid.mode": "checkid_setup",
        "openid.return_to": callback_url,
        "openid.realm": PUBLIC_BACKEND_URL,
        "openid.identity": f"{OPENID_NAMESPACE}/identifier_select",
        "openid.claimed_id": f"{OPENID_NAMESPACE}/identifier_select",
    }
    return RedirectResponse(f"{STEAM_OPENID_URL}?{urllib.parse.urlencode(params)}")


@router.get("/auth/steam/callback")
@rate_limiter.limit(10, 60)
async def steam_callback(request: Request, state: str) -> RedirectResponse:
    expected_state = request.session.pop("openid_state", None)
    if not expected_state or not secrets.compare_digest(expected_state, state):
        raise HTTPException(status_code=400, detail="Estado OpenID inválido")
    if not await redis.delete(_state_key(state)):
        raise HTTPException(status_code=400, detail="Estado OpenID caducado o reutilizado")

    steam_id = await _verify_openid_response(request, _callback_url(state))
    username, avatar = await _load_profile(steam_id)

    request.session.clear()
    request.session.update(
        {
            "steam_id": steam_id,
            "username": username,
            "avatar": avatar,
        }
    )
    await redis.sadd(f"{PIPELINE_NAMESPACE}:users", steam_id)
    logging.info("Steam login verified for user ending in %s", steam_id[-4:])
    return RedirectResponse(f"{FRONTEND_URL}/steam-login-success", status_code=303)


@router.get("/auth/steam/status")
async def steam_status(request: Request) -> dict[str, Any]:
    steam_id = request.session.get("steam_id")
    if not steam_id:
        return {"authenticated": False}
    return {
        "authenticated": True,
        "steam_id": steam_id,
        "username": request.session.get("username", ""),
        "avatar": request.session.get("avatar", ""),
    }


@router.post("/auth/steam/logout")
async def steam_logout(request: Request) -> dict[str, str]:
    request.session.clear()
    return {"message": "Sesión cerrada correctamente"}
