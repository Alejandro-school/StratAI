import hashlib
import json
import os
import time
from pathlib import Path

import pytest

from backend.app.utils import user_aggregates

STEAM_ID = "76561198000000001"
MAP_NAME = "de_mirage"


def configure_storage(tmp_path, monkeypatch) -> tuple[Path, Path]:
    exports_path = tmp_path / "exports"
    users_path = tmp_path / "users"
    exports_path.mkdir()
    monkeypatch.setattr(user_aggregates, "EXPORTS_PATH", exports_path)
    monkeypatch.setattr(user_aggregates, "USERS_PATH", users_path)
    return exports_path, users_path


def write_json(path: Path, data) -> None:
    path.write_text(json.dumps(data), encoding="utf-8")


def create_match(
    exports_path: Path,
    folder_name: str,
    match_id: str,
    *,
    map_name: str = MAP_NAME,
    steam_ids: tuple[str, ...] = (STEAM_ID,),
    summary_as_list: bool = False,
) -> Path:
    match_directory = exports_path / folder_name
    match_directory.mkdir()
    canonical = match_directory / "canonical"
    (canonical / "core").mkdir(parents=True)
    (canonical / "derived").mkdir()
    (canonical / "events").mkdir()
    (canonical / "states" / "player_states").mkdir(parents=True)
    write_json(
        canonical / "core" / "match.json",
        {"match_id": match_id, "map_name": map_name},
    )
    players_summary = {
        "match_id": match_id,
        "players": [
            {"player_id": f"steam:{steam_id}", "metrics": {"steam_id": steam_id}}
            for steam_id in steam_ids
        ],
    }
    write_json(canonical / "derived" / "player_match_stats.json", players_summary)
    write_json(canonical / "manifest.json", {})
    combat = canonical / "events" / "combat_events.jsonl"
    utility = canonical / "events" / "utility_events.jsonl"
    states = canonical / "states" / "player_states" / "round_001.jsonl"
    for path in (combat, utility, states):
        path.write_text("", encoding="utf-8")
    write_json(
        match_directory / "manifest.json",
        {
            "artifacts": [
                {"artifact_type": "canonical_manifest", "path": "canonical/manifest.json"},
                {"artifact_type": "match", "path": "canonical/core/match.json"},
                {
                    "artifact_type": "player_match_stats",
                    "path": "canonical/derived/player_match_stats.json",
                },
                {"artifact_type": "combat_events", "path": "canonical/events/combat_events.jsonl"},
                {"artifact_type": "utility_events", "path": "canonical/events/utility_events.jsonl"},
                {
                    "artifact_type": "player_states",
                    "path": "canonical/states/player_states/round_001.jsonl",
                },
            ]
        },
    )
    return match_directory


def set_future_mtime(path: Path, offset_ns: int) -> int:
    requested_mtime_ns = time.time_ns() + offset_ns
    os.utime(path, ns=(requested_mtime_ns, requested_mtime_ns))
    return path.stat().st_mtime_ns


def test_source_signature_is_deterministic_and_filters_map_and_player(
    tmp_path,
    monkeypatch,
) -> None:
    exports_path, _ = configure_storage(tmp_path, monkeypatch)
    second_match = create_match(
        exports_path,
        "match_folder_b",
        "match-b",
        summary_as_list=True,
    )
    create_match(exports_path, "match_folder_a", "match-a")
    wrong_map = create_match(
        exports_path,
        "match_wrong_map",
        "wrong-map",
        map_name="de_nuke",
    )
    create_match(
        exports_path,
        "match_wrong_player",
        "wrong-player",
        steam_ids=("76561198000000002",),
    )
    relevant_mtime = set_future_mtime(
        second_match / "manifest.json",
        10_000_000_000,
    )
    set_future_mtime(wrong_map / "manifest.json", 20_000_000_000)

    signature = user_aggregates.build_user_map_source_signature(
        STEAM_ID,
        MAP_NAME,
    )

    assert signature == user_aggregates.build_user_map_source_signature(
        STEAM_ID,
        MAP_NAME,
    )
    assert signature == {
        "match_count": 2,
        "latest_source_mtime_ns": relevant_mtime,
        "match_ids_sha256": hashlib.sha256(
            b"match-a\nmatch-b"
        ).hexdigest(),
    }


@pytest.mark.parametrize(
    "source_filename",
    ["combat_events", "player_states", "utility_events"],
)
def test_tactical_source_mtime_invalidates_snapshot(
    source_filename,
    tmp_path,
    monkeypatch,
) -> None:
    exports_path, _ = configure_storage(tmp_path, monkeypatch)
    match_directory = create_match(exports_path, "match_a", "match-a")
    data = user_aggregates.attach_user_map_source_snapshot(
        STEAM_ID,
        MAP_NAME,
        {"matches_analyzed": 1},
    )

    assert user_aggregates.is_user_map_data_fresh(
        STEAM_ID,
        MAP_NAME,
        data,
    )

    manifest = json.loads((match_directory / "manifest.json").read_text(encoding="utf-8"))
    artifact = next(item for item in manifest["artifacts"] if item["artifact_type"] == source_filename)
    set_future_mtime(match_directory / artifact["path"], 10_000_000_000)
    set_future_mtime(match_directory / "manifest.json", 11_000_000_000)

    assert not user_aggregates.is_user_map_data_fresh(
        STEAM_ID,
        MAP_NAME,
        data,
    )


def test_snapshot_attachment_is_non_mutating_and_rejects_legacy_data(
    tmp_path,
    monkeypatch,
) -> None:
    exports_path, _ = configure_storage(tmp_path, monkeypatch)
    create_match(exports_path, "match_a", "match-a")
    original = {"matches_analyzed": 1}

    attached = user_aggregates.attach_user_map_source_snapshot(
        STEAM_ID,
        MAP_NAME,
        original,
    )

    assert original == {"matches_analyzed": 1}
    assert attached["schema_version"] == 3
    assert attached["source_snapshot"]["match_count"] == 1
    assert not user_aggregates.is_user_map_data_fresh(
        STEAM_ID,
        MAP_NAME,
        {"matches_analyzed": 1},
    )
    assert not user_aggregates.is_user_map_data_fresh(
        STEAM_ID,
        MAP_NAME,
        {**attached, "schema_version": 1},
    )


def test_save_user_map_data_uses_atomic_replace_and_attaches_snapshot(
    tmp_path,
    monkeypatch,
) -> None:
    exports_path, users_path = configure_storage(tmp_path, monkeypatch)
    create_match(exports_path, "match_a", "match-a")
    replacements = []
    real_replace = os.replace

    def track_replace(source, destination) -> None:
        replacements.append((Path(source), Path(destination)))
        real_replace(source, destination)

    monkeypatch.setattr(user_aggregates.os, "replace", track_replace)
    original = {"matches_analyzed": 1}

    assert user_aggregates.save_user_map_data(
        STEAM_ID,
        MAP_NAME,
        original,
    )

    target = users_path / STEAM_ID / "maps" / f"{MAP_NAME}.json"
    saved = json.loads(target.read_text(encoding="utf-8"))
    assert original == {"matches_analyzed": 1}
    assert saved["schema_version"] == 3
    assert saved["source_snapshot"]["match_count"] == 1
    assert user_aggregates.is_user_map_data_fresh(STEAM_ID, MAP_NAME)
    assert len(replacements) == 1
    assert replacements[0][0].parent == target.parent
    assert replacements[0][1] == target
    assert list(target.parent.glob(f".{target.name}.*.tmp")) == []


def test_failed_atomic_replace_preserves_previous_map_data(
    tmp_path,
    monkeypatch,
) -> None:
    exports_path, users_path = configure_storage(tmp_path, monkeypatch)
    create_match(exports_path, "match_a", "match-a")
    assert user_aggregates.save_user_map_data(
        STEAM_ID,
        MAP_NAME,
        {"matches_analyzed": 1},
    )
    target = users_path / STEAM_ID / "maps" / f"{MAP_NAME}.json"
    previous_contents = target.read_text(encoding="utf-8")

    def fail_replace(_source, _destination) -> None:
        raise OSError("replace failed")

    monkeypatch.setattr(user_aggregates.os, "replace", fail_replace)

    assert not user_aggregates.save_user_map_data(
        STEAM_ID,
        MAP_NAME,
        {"matches_analyzed": 2},
    )
    assert target.read_text(encoding="utf-8") == previous_contents
    assert list(target.parent.glob(f".{target.name}.*.tmp")) == []
