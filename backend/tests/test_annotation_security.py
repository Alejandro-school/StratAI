"""Replay annotation authorization and storage quotas with synthetic bundles."""

import json
import threading
from concurrent.futures import ThreadPoolExecutor
from types import SimpleNamespace

import pytest
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient
from starlette.middleware.sessions import SessionMiddleware

from backend.app.routes import replay_annotations
from backend.app.utils import replay_annotations as annotation_storage
from backend.tests.session_support import install_session_store
from backend.tests.test_replay_access_security import write_replay_bundle


OWNER = "76561198000000001"
PARTICIPANT = "76561198000000002"
UNRELATED = "76561198000000003"


def annotation_payload(text="Synthetic private annotation"):
    return {
        "round": 1,
        "start_tick": 100,
        "end_tick": 200,
        "type": "note",
        "points": [{"x": 1, "y": 2}],
        "text": text,
        "color": "#63d7ff",
    }


def annotation_url(match="owned-a", annotation_id=None):
    base = f"/match/{match}/replay/annotations"
    return f"{base}/{annotation_id}" if annotation_id else base


@pytest.fixture
def annotation_app(tmp_path, monkeypatch):
    exports = tmp_path / "exports"
    users = tmp_path / "users"
    for match in ("owned-a", "owned-b", "owned-c"):
        directory = exports / f"match_{match}"
        write_replay_bundle(directory, owner=OWNER)
        (directory / "canonical" / "player_stats.json").write_text(
            json.dumps({"players": [
                {"metrics": {"steam_id": OWNER}},
                {"metrics": {"steam_id": PARTICIPANT}},
            ]}),
            encoding="utf-8",
        )
    write_replay_bundle(exports / "match_foreign", owner=UNRELATED)
    monkeypatch.setattr(replay_annotations, "EXPORTS_PATH", exports, raising=False)
    store = install_session_store(monkeypatch)

    def ensure_user_directory(steam_id):
        directory = users / steam_id
        directory.mkdir(parents=True, exist_ok=True)
        return directory

    async def not_limited(*_args, **_kwargs):
        return False, 100

    monkeypatch.setattr(annotation_storage, "ensure_user_directory", ensure_user_directory)
    if hasattr(replay_annotations, "rate_limiter"):
        monkeypatch.setattr(replay_annotations.rate_limiter, "is_rate_limited", not_limited)

    app = FastAPI()
    app.add_middleware(SessionMiddleware, secret_key="synthetic-annotation-session-secret")
    app.include_router(replay_annotations.router)

    @app.post("/_test/session/{steam_id}")
    async def set_session(steam_id: str, request: Request):
        request.session.clear()
        request.session["sid"] = await store.create(steam_id)
        return {"ok": True}

    def client_for(steam_id=OWNER):
        client = TestClient(app)
        if steam_id is not None:
            assert client.post(f"/_test/session/{steam_id}").status_code == 200
        return client

    return SimpleNamespace(app=app, client_for=client_for, users=users)


def request_annotation(client, method, match):
    item_id = "synthetic-item" if method in {"patch", "delete"} else None
    options = {}
    if method == "post":
        options["json"] = annotation_payload()
    elif method == "patch":
        options["json"] = {"text": "Synthetic update"}
    return client.request(method, annotation_url(match, item_id), **options)


@pytest.mark.parametrize("method", ["get", "post", "patch", "delete"])
def test_annotation_routes_require_an_authenticated_session(annotation_app, method):
    client = annotation_app.client_for(None)

    assert request_annotation(client, method, "owned-a").status_code == 401
    assert not annotation_app.users.exists()


@pytest.mark.parametrize("method", ["get", "post", "patch", "delete"])
@pytest.mark.parametrize("match,status", [("missing", 404), ("foreign", 403)])
def test_denied_match_access_creates_no_annotation_storage(annotation_app, method, match, status):
    client = annotation_app.client_for()

    assert request_annotation(client, method, match).status_code == status
    assert not annotation_app.users.exists()


def test_authorized_participants_have_independent_annotation_crud(annotation_app):
    owner = annotation_app.client_for(OWNER)
    participant = annotation_app.client_for(PARTICIPANT)
    created = owner.post(annotation_url(), json=annotation_payload())
    assert created.status_code == 201
    annotation_id = created.json()["id"]

    assert participant.get(annotation_url()).json() == []
    assert participant.patch(annotation_url(annotation_id=annotation_id), json={"text": "Not mine"}).status_code == 404
    assert participant.delete(annotation_url(annotation_id=annotation_id)).status_code == 404
    updated = owner.patch(annotation_url(annotation_id=annotation_id), json={"text": "Owner correction"})
    assert updated.status_code == 200
    assert updated.json()["text"] == "Owner correction"
    assert owner.get(annotation_url()).json() == [updated.json()]
    assert owner.delete(annotation_url(annotation_id=annotation_id)).status_code == 204
    assert owner.get(annotation_url()).json() == []


def test_annotation_count_quota_rejects_without_overwriting_and_delete_releases_capacity(annotation_app, monkeypatch):
    monkeypatch.setattr(annotation_storage, "MAX_ANNOTATIONS_PER_MATCH", 2, raising=False)
    client = annotation_app.client_for()
    first = client.post(annotation_url(), json=annotation_payload("First"))
    second = client.post(annotation_url(), json=annotation_payload("Second"))
    assert first.status_code == second.status_code == 201
    original = client.get(annotation_url()).json()

    rejected = client.post(annotation_url(), json=annotation_payload("Over quota"))

    assert rejected.status_code == 413
    assert client.get(annotation_url()).json() == original
    assert client.delete(annotation_url(annotation_id=first.json()["id"])).status_code == 204
    assert client.post(annotation_url(), json=annotation_payload("Replacement")).status_code == 201


def test_match_file_quota_is_per_user_and_existing_files_remain_editable(annotation_app, monkeypatch):
    monkeypatch.setattr(annotation_storage, "MAX_MATCH_FILES_PER_USER", 2, raising=False)
    owner = annotation_app.client_for(OWNER)
    first = owner.post(annotation_url("owned-a"), json=annotation_payload("First match"))
    assert first.status_code == 201
    assert owner.post(annotation_url("owned-b"), json=annotation_payload("Second match")).status_code == 201

    assert owner.post(annotation_url("owned-c"), json=annotation_payload("Third match")).status_code == 413
    assert len(list((annotation_app.users / OWNER).rglob("*.json"))) == 2
    updated = owner.patch(annotation_url("owned-a", first.json()["id"]), json={"text": "Allowed edit"})
    assert updated.status_code == 200
    participant = annotation_app.client_for(PARTICIPANT)
    assert participant.post(annotation_url("owned-c"), json=annotation_payload("Independent quota")).status_code == 201


def test_user_byte_quota_rejects_a_large_first_record_without_writing_json(annotation_app, monkeypatch):
    monkeypatch.setattr(annotation_storage, "MAX_USER_BYTES", 300, raising=False)
    client = annotation_app.client_for()

    response = client.post(annotation_url(), json=annotation_payload("x" * 200))

    assert response.status_code == 413
    assert not list(annotation_app.users.rglob("*.json"))
    assert not list(annotation_app.users.rglob("*.tmp"))


def test_user_byte_quota_accounts_for_all_matches(annotation_app, monkeypatch):
    client = annotation_app.client_for()
    assert client.post(annotation_url("owned-a"), json=annotation_payload()).status_code == 201
    existing_file = next((annotation_app.users / OWNER).rglob("*.json"))
    before = existing_file.read_bytes()
    monkeypatch.setattr(annotation_storage, "MAX_USER_BYTES", len(before) + 1, raising=False)

    response = client.post(annotation_url("owned-b"), json=annotation_payload())

    assert response.status_code == 413
    assert existing_file.read_bytes() == before
    assert list((annotation_app.users / OWNER).rglob("*.json")) == [existing_file]


def test_patch_quota_counts_utf8_bytes_and_preserves_the_original_file(annotation_app, monkeypatch):
    client = annotation_app.client_for()
    created = client.post(annotation_url(), json=annotation_payload("x"))
    assert created.status_code == 201
    existing_file = next((annotation_app.users / OWNER).rglob("*.json"))
    before = existing_file.read_bytes()
    monkeypatch.setattr(annotation_storage, "MAX_USER_BYTES", len(before) + 100, raising=False)

    # This adds fewer than 100 characters, but more than 100 encoded UTF-8 bytes.
    rejected = client.patch(annotation_url(annotation_id=created.json()["id"]), json={"text": "é" * 75})

    assert rejected.status_code == 413
    assert existing_file.read_bytes() == before
    assert client.get(annotation_url()).json() == [created.json()]


def test_concurrent_mutations_preserve_each_annotation(annotation_app):
    worker_count = 8
    ready = threading.Barrier(worker_count)

    def worker(index):
        ready.wait(timeout=5)
        annotation_storage.mutate_annotations(
            OWNER, "owned-a", lambda records: [*records, {"id": str(index), "text": "Synthetic"}]
        )

    with ThreadPoolExecutor(max_workers=worker_count) as executor:
        list(executor.map(worker, range(worker_count)))

    records = annotation_storage.load_annotations(OWNER, "owned-a")
    assert len(records) == worker_count
    assert {record["id"] for record in records} == {str(index) for index in range(worker_count)}
