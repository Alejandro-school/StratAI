import asyncio
import json

from backend.app.auth.dependencies import SteamUser
from backend.app.matches.player_match_catalog import (
    clear_player_match_cache,
    list_player_matches,
)
from backend.app.routes import dashboard
from backend.app.utils import performance_aggregator

USER_STEAM_ID = "76561198116485358"


def _write_json(path, payload):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload), encoding="utf-8")


def _create_match(
    exports_path,
    match_id,
    *,
    played_at,
    user_team,
    winner,
    ct_score,
    t_score,
):
    match_path = exports_path / f"match_{match_id}"
    canonical_path = match_path / "canonical"
    _write_json(canonical_path / "manifest.json", {"match_id": match_id})
    _write_json(
        canonical_path / "core" / "match.json",
        {
            "match_id": match_id,
            "map_name": "de_nuke" if user_team == "CT" else "de_cache",
            "played_at": played_at,
            "duration_ms": 1_800_000,
            "tick_rate_hz": 64,
            "round_count": ct_score + t_score,
            "ct_score": ct_score,
            "t_score": t_score,
            "winner_side": winner,
        },
    )
    _write_json(
        canonical_path / "derived" / "player_match_stats.json",
        {
            "match_id": match_id,
            "players": [
                {
                    "player_id": f"steam:{USER_STEAM_ID}",
                    "metrics": {
                        "steam_id": USER_STEAM_ID,
                        "name": "Kerchak",
                        "team": user_team,
                        "kills": 20,
                        "deaths": 10,
                        "assists": 5,
                        "headshots": 10,
                        "kd_ratio": 2.0,
                        "adr": 100.0,
                        "hs_percentage": 50.0,
                        "hltv_rating": 1.4,
                        "kast": 80.0,
                        "impact_rating": 1.5,
                        "ct_rating": 1.4,
                        "t_rating": 1.3,
                        "ct_adr": 102.0,
                        "t_adr": 98.0,
                        "total_damage": 2_000,
                        "shots_fired": 100,
                        "shots_hit": 40,
                        "rounds_survived": 12,
                        "body_part_hits": {"head": 10, "chest": 20},
                        "weapon_stats": {},
                    },
                }
            ],
        },
    )
    _write_json(
        match_path / "manifest.json",
        {
            "match_id": match_id,
            "artifacts": [
                {
                    "artifact_type": "canonical_manifest",
                    "path": "canonical/manifest.json",
                },
                {
                    "artifact_type": "match",
                    "path": "canonical/core/match.json",
                },
                {
                    "artifact_type": "player_match_stats",
                    "path": "canonical/derived/player_match_stats.json",
                },
            ],
        },
    )


class _ForbiddenRedis:
    def __getattr__(self, name):
        raise AssertionError(
            f"Redis must not determine canonical match visibility: {name}"
        )


def _original_handler(handler):
    return getattr(handler, "__wrapped__", handler)


def test_history_dashboard_and_performance_share_canonical_matches(
    tmp_path,
    monkeypatch,
):
    exports_path = tmp_path / "exports"
    _create_match(
        exports_path,
        "sharecode-one",
        played_at="2026-07-05T14:59:23Z",
        user_team="CT",
        winner="ct",
        ct_score=13,
        t_score=7,
    )
    _create_match(
        exports_path,
        "sharecode-two",
        played_at="2026-07-24T21:23:03Z",
        user_team="T",
        winner="t",
        ct_score=9,
        t_score=13,
    )
    clear_player_match_cache()
    monkeypatch.setattr(dashboard, "EXPORTS_PATH", exports_path)
    monkeypatch.setattr(dashboard, "redis", _ForbiddenRedis())
    monkeypatch.setattr(performance_aggregator, "EXPORTS_PATH", exports_path)
    user = SteamUser(steam_id=USER_STEAM_ID)

    history = asyncio.run(
        _original_handler(dashboard.get_processed_demos)(None, user=user)
    )
    dashboard_stats = asyncio.run(
        _original_handler(dashboard.get_dashboard_stats)(None, user=user)
    )
    performance = performance_aggregator.build_performance_overview(USER_STEAM_ID)

    expected_ids = {"sharecode-one", "sharecode-two"}
    assert {match["match_id"] for match in history["matches"]} == expected_ids
    assert {
        match["match_id"] for match in dashboard_stats["recent_matches"]
    } == expected_ids
    assert {match["match_id"] for match in performance["match_history"]} == expected_ids
    assert dashboard_stats["stats"]["total_matches"] == 2
    assert performance["overview"]["total_matches"] == 2


def test_player_perspective_and_manifest_cache_invalidation(tmp_path):
    exports_path = tmp_path / "exports"
    _create_match(
        exports_path,
        "first",
        played_at="2026-07-05T14:59:23Z",
        user_team="CT",
        winner="ct",
        ct_score=13,
        t_score=7,
    )
    clear_player_match_cache()

    first_result = list_player_matches(USER_STEAM_ID, exports_path)
    assert len(first_result) == 1
    assert first_result[0].result == "W"
    assert (first_result[0].team_score, first_result[0].opponent_score) == (13, 7)

    _create_match(
        exports_path,
        "second",
        played_at="2026-07-24T21:23:03Z",
        user_team="T",
        winner="t",
        ct_score=9,
        t_score=13,
    )
    refreshed = list_player_matches(USER_STEAM_ID, exports_path)

    assert len(refreshed) == 2
    t_side_match = next(match for match in refreshed if match.match_id == "second")
    assert t_side_match.result == "W"
    assert (t_side_match.team_score, t_side_match.opponent_score) == (13, 9)
