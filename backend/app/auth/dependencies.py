from dataclasses import dataclass

from fastapi import HTTPException, Request


@dataclass(frozen=True)
class SteamUser:
    steam_id: str
    username: str = ""
    avatar: str = ""


def require_steam_user(request: Request) -> SteamUser:
    steam_id = request.session.get("steam_id")
    if not isinstance(steam_id, str) or len(steam_id) != 17 or not steam_id.isdigit():
        raise HTTPException(status_code=401, detail="Usuario no autenticado.")
    return SteamUser(
        steam_id=steam_id,
        username=str(request.session.get("username", "")),
        avatar=str(request.session.get("avatar", "")),
    )
