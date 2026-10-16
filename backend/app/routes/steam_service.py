import asyncio
from typing import Any

import httpx
from fastapi import APIRouter, Depends

from ..auth.dependencies import SteamUser, require_steam_user
from ..config import STEAM_API_KEY

router = APIRouter()


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


@router.get("/steam/get-user-data")
async def get_user_data(user: SteamUser = Depends(require_steam_user)) -> dict[str, Any]:
    steam_data = await _get_steam_data(user.steam_id)
    return {"steam_id": user.steam_id, **steam_data}
