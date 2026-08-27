from __future__ import annotations

import gzip
import json
from collections.abc import Iterator, Mapping, Sequence
from functools import lru_cache
from pathlib import Path
from types import MappingProxyType
from typing import Any

from ..utils.match_artifact_catalog import (
    resolve_match_artifact,
    resolve_match_artifacts,
)

JsonObject = Mapping[str, Any]


def _freeze(value: Any) -> Any:
    if isinstance(value, dict):
        return MappingProxyType({key: _freeze(item) for key, item in value.items()})
    if isinstance(value, list):
        return tuple(_freeze(item) for item in value)
    return value


def thaw_json(value: Any) -> Any:
    if isinstance(value, Mapping):
        return {key: thaw_json(item) for key, item in value.items()}
    if isinstance(value, tuple):
        return [thaw_json(item) for item in value]
    return value


@lru_cache(maxsize=256)
def _load_json_version(
    path: str,
    _mtime_ns: int,
    _size: int,
    is_gzip: bool,
) -> Any:
    opener = gzip.open if is_gzip else open
    with opener(path, "rt", encoding="utf-8") as file:
        return _freeze(json.load(file))


def _load_json(path: Path) -> Any:
    stat = path.stat()
    return _load_json_version(
        str(path),
        stat.st_mtime_ns,
        stat.st_size,
        path.suffix == ".gz",
    )


@lru_cache(maxsize=8)
def _load_replay_round_version(
    path: str,
    _mtime_ns: int,
    _size: int,
) -> Any:
    with gzip.open(path, "rt", encoding="utf-8") as file:
        return _freeze(json.load(file))


class CanonicalMatch:
    __slots__ = ("directory",)

    def __init__(self, directory: Path | str):
        self.directory = Path(directory).resolve()

    @property
    def match_id(self) -> str:
        return str(self.match().get("match_id") or self.directory.name.removeprefix("match_"))

    def has_canonical_bundle(self) -> bool:
        return self.artifact_path("canonical_manifest") is not None

    def artifact_path(self, artifact_type: str) -> Path | None:
        return resolve_match_artifact(self.directory, artifact_type)

    def artifact_paths(self, artifact_type: str) -> tuple[Path, ...]:
        return resolve_match_artifacts(self.directory, artifact_type)

    def artifact_mtime_ns(self, artifact_type: str) -> int:
        values = []
        for path in self.artifact_paths(artifact_type):
            try:
                values.append(path.stat().st_mtime_ns)
            except OSError:
                continue
        return max(values, default=0)

    def match(self) -> JsonObject:
        return self._object("match")

    def participants(self) -> Sequence[JsonObject]:
        return self._sequence("participants", "players")

    def rounds(self) -> Sequence[JsonObject]:
        return self._sequence("rounds", "rounds")

    def player_match_stats(self) -> Sequence[JsonObject]:
        current = self._sequence("player_stats", "players")
        return current or self._sequence("player_match_stats", "players")

    def economy_records(self) -> Sequence[JsonObject]:
        current = self._sequence("economy_players", "players")
        return current or self._sequence("player_round_economy", "records")

    def economy_rounds(self) -> Sequence[JsonObject]:
        current = self._sequence("economy_rounds", "rounds")
        return current or self._sequence("player_round_economy", "rounds")

    def clutch_events(self) -> Sequence[JsonObject]:
        return self._sequence("clutch_events", "clutch_events")

    def match_metadata(self) -> JsonObject:
        return self._object("match_metadata")

    def engagements(self) -> Sequence[JsonObject]:
        return self._sequence("engagements", "engagements")

    def quality_report(self) -> JsonObject:
        return self._object("quality_report").get("report", MappingProxyType({}))

    def replay_index(self) -> JsonObject:
        return self._object("replay_index")

    def replay_round(self, round_number: int) -> JsonObject:
        index = self.replay_index()
        entry = next(
            (
                item
                for item in index.get("rounds", ())
                if isinstance(item, Mapping) and item.get("round_number") == round_number
            ),
            None,
        )
        if not isinstance(entry, Mapping):
            return MappingProxyType({})
        path = self._canonical_relative_path(entry.get("path"))
        if path is None:
            return MappingProxyType({})
        stat = path.stat()
        payload = _load_replay_round_version(
            str(path),
            stat.st_mtime_ns,
            stat.st_size,
        )
        if not isinstance(payload, Mapping):
            return MappingProxyType({})
        round_data = payload.get("round")
        return round_data if isinstance(round_data, Mapping) else MappingProxyType({})

    def iter_combat_events(self) -> Iterator[dict[str, Any]]:
        yield from self._iter_json_lines("combat_events")

    def iter_utility_events(self) -> Iterator[dict[str, Any]]:
        yield from self._iter_json_lines("utility_events")

    def iter_objective_events(self) -> Iterator[dict[str, Any]]:
        yield from self._iter_json_lines("objective_events")

    def iter_player_states(self) -> Iterator[dict[str, Any]]:
        for path in self.artifact_paths("player_states"):
            yield from self._read_json_lines(path)

    def iter_player_state_rounds(
        self,
        steam_id: str,
    ) -> Iterator[tuple[int, list[dict[str, Any]]]]:
        player_id = f"steam:{str(steam_id).removeprefix('steam:')}"
        for path in self.artifact_paths("player_states"):
            states = [
                state
                for state in self._read_json_lines(path)
                if state.get("player_id") == player_id
            ]
            if states:
                yield int(states[0].get("round_number") or 0), states

    def manifest_mtime_ns(self) -> int:
        try:
            return (self.directory / "manifest.json").stat().st_mtime_ns
        except OSError:
            return 0

    def _object(self, artifact_type: str) -> JsonObject:
        path = self.artifact_path(artifact_type)
        if path is None:
            return MappingProxyType({})
        try:
            payload = _load_json(path)
        except (OSError, UnicodeError, json.JSONDecodeError):
            return MappingProxyType({})
        return payload if isinstance(payload, Mapping) else MappingProxyType({})

    def _sequence(self, artifact_type: str, key: str) -> Sequence[JsonObject]:
        values = self._object(artifact_type).get(key, ())
        return values if isinstance(values, tuple) else ()

    def _iter_json_lines(self, artifact_type: str) -> Iterator[dict[str, Any]]:
        path = self.artifact_path(artifact_type)
        if path is not None:
            yield from self._read_json_lines(path)

    @staticmethod
    def _read_json_lines(path: Path) -> Iterator[dict[str, Any]]:
        try:
            with path.open("r", encoding="utf-8") as file:
                for line in file:
                    if not line.strip():
                        continue
                    value = json.loads(line)
                    if isinstance(value, dict):
                        yield value
        except (OSError, UnicodeError, json.JSONDecodeError):
            return

    def _canonical_relative_path(self, relative_path: object) -> Path | None:
        if not isinstance(relative_path, str) or not relative_path:
            return None
        canonical_directory = (self.directory / "canonical").resolve()
        candidate = (canonical_directory / relative_path).resolve()
        try:
            candidate.relative_to(canonical_directory)
        except ValueError:
            return None
        return candidate if candidate.is_file() else None


def iter_canonical_matches(exports_directory: Path | str) -> Iterator[CanonicalMatch]:
    root = Path(exports_directory)
    if not root.is_dir():
        return
    try:
        directories = sorted(root.iterdir(), key=lambda path: path.name)
    except OSError:
        return
    for directory in directories:
        if not directory.is_dir() or not directory.name.startswith("match_"):
            continue
        match = CanonicalMatch(directory)
        if match.has_canonical_bundle():
            yield match


def find_canonical_match(
    exports_directory: Path | str,
    match_id: str,
) -> CanonicalMatch | None:
    normalized = str(match_id).removeprefix("match_")
    candidate = Path(exports_directory).resolve() / f"match_{normalized}"
    match = CanonicalMatch(candidate)
    return match if candidate.is_dir() and match.has_canonical_bundle() else None
