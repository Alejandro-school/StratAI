from __future__ import annotations

from collections.abc import Mapping
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path
from types import MappingProxyType
from typing import Any

from .canonical_repository import CanonicalMatch, iter_canonical_matches
from .match_web_projection import project_match_metadata, project_player_summary


@dataclass(frozen=True, slots=True)
class PlayerMatch:
    export: CanonicalMatch
    match_id: str
    metadata: Mapping[str, Any]
    player: Mapping[str, Any]
    user_team: str
    winner: str
    result: str
    team_score: int
    opponent_score: int
    total_rounds: int


ManifestFingerprint = tuple[tuple[str, int, int], ...]


def list_player_matches(
    steam_id: str,
    exports_directory: Path | str,
) -> tuple[PlayerMatch, ...]:
    root = Path(exports_directory).resolve()
    fingerprint = _manifest_fingerprint(root)
    return _list_player_matches_cached(str(root), str(steam_id), fingerprint)


def clear_player_match_cache() -> None:
    _list_player_matches_cached.cache_clear()


def _manifest_fingerprint(root: Path) -> ManifestFingerprint:
    if not root.is_dir():
        return ()

    manifests: list[tuple[str, int, int]] = []
    try:
        directories = sorted(root.iterdir(), key=lambda path: path.name)
    except OSError:
        return ()

    for directory in directories:
        if not directory.is_dir() or not directory.name.startswith("match_"):
            continue
        try:
            stat = (directory / "manifest.json").stat()
        except OSError:
            continue
        manifests.append((directory.name, stat.st_mtime_ns, stat.st_size))
    return tuple(manifests)


@lru_cache(maxsize=128)
def _list_player_matches_cached(
    exports_directory: str,
    steam_id: str,
    _fingerprint: ManifestFingerprint,
) -> tuple[PlayerMatch, ...]:
    matches = [
        player_match
        for match in iter_canonical_matches(exports_directory)
        if (player_match := _project_player_match(match, steam_id)) is not None
    ]
    matches.sort(key=_player_match_sort_key, reverse=True)
    return tuple(matches)


def _project_player_match(
    match: CanonicalMatch,
    steam_id: str,
) -> PlayerMatch | None:
    players = project_player_summary(match).get("players", [])
    player = next(
        (
            candidate
            for candidate in players
            if str(candidate.get("steam_id", "")) == steam_id
        ),
        None,
    )
    if not isinstance(player, dict):
        return None

    metadata = project_match_metadata(match)
    ct_score, t_score = _extract_scores(metadata)
    user_team = str(player.get("team") or "").upper()
    winner = str(metadata.get("winner") or "").upper()
    team_score, opponent_score = _perspective_scores(
        user_team,
        ct_score,
        t_score,
    )
    return PlayerMatch(
        export=match,
        match_id=match.match_id,
        metadata=MappingProxyType(metadata),
        player=MappingProxyType(player),
        user_team=user_team,
        winner=winner,
        result=_result_for_player(user_team, winner, team_score, opponent_score),
        team_score=team_score,
        opponent_score=opponent_score,
        total_rounds=int(metadata.get("total_rounds") or ct_score + t_score),
    )


def _extract_scores(metadata: Mapping[str, Any]) -> tuple[int, int]:
    final_score = str(metadata.get("final_score") or "0-0")
    ct_raw, separator, t_raw = final_score.partition("-")
    if not separator:
        return 0, 0
    return _safe_int(ct_raw), _safe_int(t_raw)


def _perspective_scores(
    user_team: str,
    ct_score: int,
    t_score: int,
) -> tuple[int, int]:
    if user_team == "T":
        return t_score, ct_score
    return ct_score, t_score


def _result_for_player(
    user_team: str,
    winner: str,
    team_score: int,
    opponent_score: int,
) -> str:
    if user_team and winner:
        return "W" if user_team == winner else "L"
    if team_score == opponent_score:
        return "D"
    return "W" if team_score > opponent_score else "L"


def _safe_int(value: Any) -> int:
    try:
        return int(str(value).strip())
    except (TypeError, ValueError):
        return 0


def _player_match_sort_key(match: PlayerMatch) -> tuple[str, int, str]:
    return (
        str(match.metadata.get("date") or ""),
        match.export.manifest_mtime_ns(),
        match.match_id,
    )
