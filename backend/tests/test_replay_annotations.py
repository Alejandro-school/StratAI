from fastapi import FastAPI, Request
from fastapi.testclient import TestClient
from starlette.middleware.sessions import SessionMiddleware

from backend.app.routes import replay_annotations
from backend.app.utils import replay_annotations as annotation_storage


def create_client(tmp_path, monkeypatch) -> TestClient:
    def ensure_user_directory(steam_id: str):
        user_dir = tmp_path / steam_id
        user_dir.mkdir(parents=True, exist_ok=True)
        return user_dir

    monkeypatch.setattr(annotation_storage, "ensure_user_directory", ensure_user_directory)
    app = FastAPI()
    app.add_middleware(SessionMiddleware, secret_key="test-secret")
    app.include_router(replay_annotations.router)

    @app.post("/test/session/{steam_id}")
    async def set_session(steam_id: str, request: Request) -> dict:
        request.session["steam_id"] = steam_id
        return {"ok": True}

    return TestClient(app)


def annotation_payload() -> dict:
    return {
        "round": 3,
        "start_tick": 100,
        "end_tick": 200,
        "type": "arrow",
        "points": [{"x": -1200.5, "y": 800.25}, {"x": -900, "y": 600}],
        "text": "Rotación tardía",
        "color": "#63d7ff",
    }


def test_annotation_crud_and_user_isolation(tmp_path, monkeypatch) -> None:
    client = create_client(tmp_path, monkeypatch)

    client.post("/test/session/76561198000000001")
    created = client.post("/match/match-1/replay/annotations", json=annotation_payload())
    assert created.status_code == 201
    annotation_id = created.json()["id"]

    updated = client.patch(
        f"/match/match-1/replay/annotations/{annotation_id}",
        json={"text": "Rotación confirmada"},
    )
    assert updated.status_code == 200
    assert updated.json()["text"] == "Rotación confirmada"

    assert len(client.get("/match/match-1/replay/annotations").json()) == 1
    client.post("/test/session/76561198000000002")
    assert client.get("/match/match-1/replay/annotations").json() == []

    client.post("/test/session/76561198000000001")
    deleted = client.delete(f"/match/match-1/replay/annotations/{annotation_id}")
    assert deleted.status_code == 204
    assert client.get("/match/match-1/replay/annotations").json() == []


def test_annotations_require_session_and_validate_ranges(tmp_path, monkeypatch) -> None:
    client = create_client(tmp_path, monkeypatch)
    assert client.get("/match/match-1/replay/annotations").status_code == 401

    client.post("/test/session/76561198000000001")
    payload = annotation_payload()
    payload["end_tick"] = 50
    response = client.post("/match/match-1/replay/annotations", json=payload)
    assert response.status_code == 422

    payload = annotation_payload()
    payload["points"][0]["x"] = 50000
    assert client.post("/match/match-1/replay/annotations", json=payload).status_code == 422
