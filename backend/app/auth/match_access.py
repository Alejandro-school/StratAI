from pathlib import Path

from fastapi import HTTPException

from .dependencies import SteamUser
from ..matches.canonical_repository import CanonicalMatch, find_canonical_match
from ..matches.match_web_projection import project_player_summary


def require_match_access(
    match_id: str, user: SteamUser, exports_directory: Path | str,
) -> CanonicalMatch:
    """Authorize the resource before reading replay frames or private annotations."""
    match = find_canonical_match(exports_directory, match_id)
    if match is None:
        raise HTTPException(status_code=404, detail="Partida no encontrada.")
    players = project_player_summary(match).get("players", [])
    if not any(str(player.get("steam_id", "")) == user.steam_id for player in players):
        raise HTTPException(status_code=403, detail="No tienes acceso a esta partida.")
    return match
