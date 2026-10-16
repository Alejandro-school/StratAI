from dataclasses import dataclass

from fastapi import HTTPException, Request
from . import sessions


@dataclass(frozen=True)
class SteamUser:
    steam_id: str
    username: str = ""
    avatar: str = ""


async def require_steam_user(request: Request) -> SteamUser:
    profile = await sessions.session_profile(request)
    if profile is None:
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    user = SteamUser(
        steam_id=profile["steam_id"],
        username=str(profile.get("username", "")),
        avatar=str(profile.get("avatar", "")),
    )
    request.state.steam_user = user
    return user
