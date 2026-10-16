import json

import pytest
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient
from redis.exceptions import ConnectionError as RedisConnectionError
from starlette.middleware.sessions import SessionMiddleware

from backend.app.middleware.security import CSRFMiddleware
from backend.app.routes import sharecodes, steam_service
from backend.app.security.credentials import decrypt_credential
from backend.tests.session_support import install_session_store


STEAM_ID = "76561198000000000"
PAYLOAD = {
    "auth_code": "TEST-HISTORY-AUTH",
    "known_code": "CSGO-AAAAA-BBBBB-CCCCC-DDDDD-EEEEE",
}


class MemoryRedis:
    def __init__(self):
        self.hashes = {}
        self.locks = set()
        self.users = set()

    async def set(self, key, _value, *, nx, ex):
        assert nx and ex == 10
        if key in self.locks:
            return False
        self.locks.add(key)
        return True

    async def hset(self, key, mapping):
        self.hashes.setdefault(key, {}).update(mapping)

    async def sadd(self, _key, value):
        self.users.add(value)

    async def hmget(self, key, *fields):
        return [self.hashes.get(key, {}).get(field) for field in fields]

    async def hget(self, key, field):
        return self.hashes.get(key, {}).get(field)


@pytest.fixture
def setup_flow(monkeypatch):
    sessions = install_session_store(monkeypatch)
    memory = MemoryRedis()
    outbound = []

    async def node_post(path, payload):
        outbound.append((path, payload))
        return {"status": "queued", "discovery_job_id": "test-job"}

    monkeypatch.setattr(sharecodes, "redis", memory)
    monkeypatch.setattr(sharecodes, "_node_post", node_post)
    app = FastAPI()
    app.add_middleware(CSRFMiddleware)
    app.add_middleware(SessionMiddleware, secret_key="isolated-onboarding-test")
    app.include_router(sharecodes.router)
    app.include_router(steam_service.router)

    @app.get("/test-session")
    async def session(request: Request):
        request.session["sid"] = await sessions.create(STEAM_ID)
        return {}

    with TestClient(app) as client:
        yield client, memory, outbound


def test_full_onboarding_uses_only_authenticated_identity_and_encrypts_code(setup_flow):
    client, memory, outbound = setup_flow
    client.get("/test-session")
    response = client.post("/steam/onboarding", json=PAYLOAD)
    assert response.status_code == 202
    assert response.json()["discovery_job_id"] == "test-job"
    stored = memory.hashes[f"{sharecodes.PIPELINE_NAMESPACE}:user:{STEAM_ID}:credentials"]
    assert decrypt_credential(stored["auth_code"]) == PAYLOAD["auth_code"]
    assert stored["status"] == "pending_validation"
    assert outbound[0][1]["steam_id"] == STEAM_ID
    assert outbound[0][1]["credential_version"] == stored["version"]
    assert PAYLOAD["auth_code"] not in json.dumps(outbound)
    assert PAYLOAD["auth_code"] not in response.text


def test_saved_codes_remain_pending_until_steam_validates_them(setup_flow):
    client, memory, outbound = setup_flow
    client.get("/test-session")
    assert client.post("/steam/onboarding", json=PAYLOAD).status_code == 202
    stored = next(iter(memory.hashes.values()))
    assert stored["validated_at"] == ""
    assert stored["initial_code"] == PAYLOAD["known_code"]
    assert len(outbound) == 1


@pytest.mark.parametrize("path", ["/steam/onboarding", "/steam/discovery"])
def test_history_mutations_require_a_session(setup_flow, path):
    client, memory, outbound = setup_flow
    assert client.post(path, json=PAYLOAD).status_code == 401
    assert not memory.hashes and not outbound


def test_history_mutations_reject_cross_origin_requests(setup_flow):
    client, memory, outbound = setup_flow
    client.get("/test-session")
    response = client.post("/steam/onboarding", json=PAYLOAD, headers={"origin": "https://attacker.example"})
    assert response.status_code == 403
    assert not memory.hashes and not outbound


@pytest.mark.parametrize("payload", [
    {**PAYLOAD, "steam_id": "76561198000000099"},
    {**PAYLOAD, "auth_code": "      "},
    {**PAYLOAD, "known_code": "https://attacker.example/demo.dem"},
    {**PAYLOAD, "known_code": "N/A"},
])
def test_history_setup_rejects_invalid_or_extra_fields(setup_flow, payload):
    client, memory, outbound = setup_flow
    client.get("/test-session")
    assert client.post("/steam/onboarding", json=payload).status_code == 422
    assert not memory.hashes and not outbound


def test_repeated_setup_and_discovery_share_a_server_side_cooldown(setup_flow):
    client, memory, outbound = setup_flow
    client.get("/test-session")
    assert client.post("/steam/onboarding", json=PAYLOAD).status_code == 202
    original = dict(next(iter(memory.hashes.values())))
    for path in ["/steam/onboarding", "/steam/discovery"]:
        response = client.post(path, json=PAYLOAD)
        assert response.status_code == 429
        assert response.headers["retry-after"] == "10"
    assert next(iter(memory.hashes.values())) == original
    assert len(outbound) == 1


def test_discovery_requires_history_credentials(setup_flow):
    client, _memory, outbound = setup_flow
    client.get("/test-session")
    assert client.post("/steam/discovery").status_code == 409
    assert not outbound


def test_redis_failure_does_not_enqueue_or_expose_details(setup_flow, monkeypatch):
    client, memory, outbound = setup_flow

    async def unavailable(*_args, **_kwargs):
        raise RedisConnectionError("sensitive connection details")

    monkeypatch.setattr(memory, "set", unavailable)
    client.get("/test-session")
    response = client.post("/steam/onboarding", json=PAYLOAD)
    assert response.status_code == 503
    assert "sensitive" not in response.text
    assert not memory.hashes and not outbound


@pytest.mark.parametrize("method,path", [
    ("get", "/steam/check-friend-status"),
    ("post", "/steam/send-friend-request"),
])
def test_friendship_endpoints_are_no_longer_exposed(setup_flow, method, path):
    client, _memory, _outbound = setup_flow
    # Route matching occurs before authentication; obsolete routes must be absent.
    assert getattr(client, method)(path).status_code == 404
