import json
import logging
import asyncio
from typing import Any
from urllib.parse import urlencode

import httpx
import redis.asyncio as aioredis
from fastapi import APIRouter, Depends, HTTPException

from ..auth.dependencies import SteamUser, require_steam_user
from ..config import NODE_SERVICE_URL, PIPELINE_NAMESPACE, REDIS_URL, STEAM_API_KEY
from ..security.service_auth import build_service_headers

router = APIRouter()
redis = aioredis.from_url(REDIS_URL, decode_responses=True)


def _json_bytes(payload: dict[str, Any]) -> bytes:
    return json.dumps(payload, separators=(",", ":"), sort_keys=True).encode()


async def _get_steam_data(steam_id: str) -> dict[str, Any]:
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            avatar_request = client.get(
                "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/",
                params={"key": STEAM_API_KEY, "steamids": steam_id},
            )
            rank_request = client.get(
                "https://api.steampowered.com/ICSGOPlayers_730/GetGamePersonalData/v1/",
                params={"key": STEAM_API_KEY, "steamid": steam_id},
            )
            avatar_response, rank_response = await asyncio.gather(
                avatar_request,
                rank_request,
                return_exceptions=True,
            )
    except httpx.HTTPError:
        return {"avatar": None, "rank": None}

    avatar = None
    rank = None
    if isinstance(avatar_response, httpx.Response) and avatar_response.is_success:
        try:
            avatar = avatar_response.json()["response"]["players"][0].get("avatarfull")
        except (KeyError, IndexError, ValueError):
            pass
    if isinstance(rank_response, httpx.Response) and rank_response.is_success:
        try:
            rank = rank_response.json()["result"].get("rank_type")
        except (KeyError, ValueError):
            pass
    return {"avatar": avatar, "rank": rank}


@router.get("/steam/check-friend-status")
async def check_friend_status(user: SteamUser = Depends(require_steam_user)) -> dict[str, Any]:
    cached, cached_ts = await redis.mget(
        f"{PIPELINE_NAMESPACE}:friend-status:{user.steam_id}",
        f"{PIPELINE_NAMESPACE}:friend-status-ts:{user.steam_id}",
    )
    path = f"/steam/check-friend?{urlencode({'steam_id': user.steam_id})}"
    try:
        async with httpx.AsyncClient(timeout=3.0) as client:
            response = await client.get(
                f"{NODE_SERVICE_URL}{path}",
                headers=build_service_headers("GET", path),
            )
            response.raise_for_status()
            data = response.json()
    except httpx.HTTPError as exc:
        logging.warning("Friend service unavailable: %s", type(exc).__name__)
        return {
            "is_friend": cached == "friend",
            "status": cached or "unknown",
            "service_down": True,
            "source": "cache",
            "cached_at": cached_ts,
        }

    status_value = data.get("status", "unknown")
    return {
        "is_friend": bool(data.get("is_friend")),
        "status": status_value,
        "bot_steam_id": data.get("bot_steam_id"),
        "service_down": bool(data.get("service_down")),
        "source": data.get("source", "live"),
        "cached_at": data.get("cached_at") or cached_ts,
    }


@router.post("/steam/send-friend-request")
async def send_friend_request(user: SteamUser = Depends(require_steam_user)) -> dict[str, Any]:
    path = "/steam/send-friend-request"
    body = _json_bytes({"steam_id": user.steam_id})
    headers = {
        **build_service_headers("POST", path, body),
        "Content-Type": "application/json",
    }
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.post(
                f"{NODE_SERVICE_URL}{path}",
                content=body,
                headers=headers,
            )
            response.raise_for_status()
            data = response.json()
    except httpx.HTTPStatusError as exc:
        try:
            error = exc.response.json()
        except ValueError:
            error = {}
        raise HTTPException(
            status_code=exc.response.status_code,
            detail=error.get("error", "Solicitud rechazada"),
        ) from exc
    except (httpx.RequestError, ValueError) as exc:
        raise HTTPException(status_code=503, detail="Bot no operativo") from exc

    return data


@router.get("/steam/get-user-data")
async def get_user_data(user: SteamUser = Depends(require_steam_user)) -> dict[str, Any]:
    steam_data = await _get_steam_data(user.steam_id)
    return {"steam_id": user.steam_id, **steam_data}
