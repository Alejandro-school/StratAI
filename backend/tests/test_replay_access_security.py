import gzip
import json

import pytest
from fastapi import FastAPI, HTTPException, Request
from fastapi.testclient import TestClient

from backend.app.auth.dependencies import SteamUser, require_steam_user
from backend.app.routes import dashboard


OWNER = "76561198000000001"
OTHER = "76561198000000002"


def write_replay_bundle(directory, owner=OWNER):
    canonical = directory / "canonical"
    canonical.mkdir(parents=True)
    documents = {
        "canonical_manifest": {},
        "player_stats": {"players": [{"metrics": {"steam_id": owner}}]},
        "replay_index": {
            "metadata": {"tick_rate": 64},
            "rounds": [{"round_number": 1, "path": "round.json.gz"}],
        },
    }
    artifacts = []
    for kind, content in documents.items():
        path = f"canonical/{kind}.json"
        (directory / path).write_text(json.dumps(content), encoding="utf-8")
        artifacts.append({"artifact_type": kind, "path": path})
    (directory / "manifest.json").write_text(json.dumps({"artifacts": artifacts}), encoding="utf-8")
    with gzip.open(canonical / "round.json.gz", "wt", encoding="utf-8") as stream:
        json.dump({"round": {"round": 1, "frames": [{"tick": 100}], "events": []}}, stream)


@pytest.fixture
def replay_client(tmp_path, monkeypatch):
    write_replay_bundle(tmp_path / "match_private")
    monkeypatch.setattr(dashboard, "EXPORTS_PATH", tmp_path)
    app = FastAPI()
    app.include_router(dashboard.router)

    def identity(request: Request):
        steam_id = request.headers.get("x-test-user")
        if not steam_id:
            raise HTTPException(401)
        return SteamUser(steam_id)

    app.dependency_overrides[require_steam_user] = identity
    return TestClient(app)


@pytest.mark.parametrize("suffix", ["", "/metadata", "/round/1"])
def test_replay_requires_authentication(replay_client, suffix):
    assert replay_client.get(f"/match/private/replay{suffix}").status_code == 401


@pytest.mark.parametrize("suffix", ["", "/metadata", "/round/1"])
def test_replay_rejects_unrelated_account(replay_client, suffix):
    response = replay_client.get(f"/match/private/replay{suffix}", headers={"x-test-user": OTHER})
    assert response.status_code == 403


@pytest.mark.parametrize("suffix", ["", "/metadata", "/round/1"])
def test_replay_allows_participant(replay_client, suffix):
    response = replay_client.get(f"/match/private/replay{suffix}", headers={"x-test-user": OWNER})
    assert response.status_code == 200
