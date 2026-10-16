"""Revocable sessions and expiry, using real routes and isolated Redis state."""

import base64
import importlib
import json
from urllib.parse import parse_qs, urlparse

import pytest
from fastapi import Depends, FastAPI, Request
from fastapi.testclient import TestClient
from redis.exceptions import ConnectionError as RedisConnectionError
from starlette.middleware.sessions import SessionMiddleware

from backend.app.auth.dependencies import SteamUser, require_steam_user
from backend.app.routes import steam_auth


STEAM_ID = "76561198000000001"
PROFILE_NAME = "Synthetic private profile"
PROFILE_AVATAR = "https://avatar.example/synthetic.png"


class Clock:
    def __init__(self):
        self.now = 1000.0

    def __call__(self):
        return self.now

    def advance(self, seconds):
        self.now += seconds


class FakeRedis:
    """Minimal Redis with NX and real expiry semantics under a virtual clock."""

    def __init__(self, clock):
        self.clock = clock
        self.values = {}
        self.deadlines = {}
        self.failures = set()
        self.users = set()

    def _before(self, operation, key):
        if operation in self.failures:
            raise RedisConnectionError("Synthetic session storage unavailable")
        if key in self.deadlines and self.deadlines[key] <= self.clock():
            self.values.pop(key, None)
            self.deadlines.pop(key, None)

    async def set(self, key, value, *, ex=None, nx=False):
        self._before("set", key)
        if nx and key in self.values:
            return None
        self.values[key] = value
        if ex is None:
            self.deadlines.pop(key, None)
        else:
            self.deadlines[key] = self.clock() + ex
        return True

    async def get(self, key):
        self._before("get", key)
        return self.values.get(key)

    async def expire(self, key, seconds):
        self._before("expire", key)
        if key not in self.values:
            return False
        self.deadlines[key] = self.clock() + seconds
        return True

    async def delete(self, key):
        self._before("delete", key)
        present = key in self.values
        self.values.pop(key, None)
        self.deadlines.pop(key, None)
        return int(present)

    async def sadd(self, _key, steam_id):
        self.users.add(steam_id)
        return 1


def cookie_payload(cookie):
    """Read the public payload of a signed Starlette cookie, without forging it."""
    encoded = cookie.split(".", 1)[0]
    return json.loads(base64.b64decode(encoded))


@pytest.fixture
def session_app(monkeypatch):
    # Imported here so unrelated tests can collect while this new module is built.
    sessions = importlib.import_module("backend.app.auth.sessions")
    clock = Clock()
    redis = FakeRedis(clock)
    store = sessions.SessionStore(
        redis, idle_seconds=10, absolute_seconds=30, clock=clock
    )
    monkeypatch.setattr(sessions, "session_store", store)
    monkeypatch.setattr(steam_auth, "redis", redis)

    async def not_limited(*_args, **_kwargs):
        return False, 10

    async def valid_openid(_request, _return_to):
        return STEAM_ID

    async def profile(_steam_id):
        return PROFILE_NAME, PROFILE_AVATAR

    monkeypatch.setattr(steam_auth.rate_limiter, "is_rate_limited", not_limited)
    monkeypatch.setattr(steam_auth, "_verify_openid_response", valid_openid)
    monkeypatch.setattr(steam_auth, "_load_profile", profile)

    app = FastAPI()
    app.add_middleware(SessionMiddleware, secret_key="synthetic-session-test-secret")
    app.include_router(steam_auth.router)

    @app.post("/_test/session")
    async def establish_session(request: Request):
        sid = await store.create(STEAM_ID, username=PROFILE_NAME, avatar=PROFILE_AVATAR)
        request.session.clear()
        request.session["sid"] = sid
        return {"ok": True}

    @app.post("/_test/legacy-session")
    async def establish_legacy_session(request: Request):
        request.session.clear()
        request.session.update({"steam_id": STEAM_ID, "username": PROFILE_NAME})
        return {"ok": True}

    @app.post("/_test/unknown-session")
    async def establish_unknown_session(request: Request):
        request.session.clear()
        request.session["sid"] = "x" * 43
        return {"ok": True}

    @app.get("/_test/protected")
    async def protected(user: SteamUser = Depends(require_steam_user)):
        return {"steam_id": user.steam_id, "username": user.username, "avatar": user.avatar}

    return app, store, redis, clock


def authenticated_client(app):
    client = TestClient(app)
    assert client.post("/_test/session").status_code == 200
    return client


def replay_client(app, cookie):
    client = TestClient(app)
    client.cookies.set("session", cookie)
    return client


def test_session_cookie_contains_only_an_opaque_server_session_id(session_app):
    app, _store, redis, _clock = session_app
    client = authenticated_client(app)
    payload = cookie_payload(client.cookies.get("session"))

    assert set(payload) == {"sid"}
    assert isinstance(payload["sid"], str)
    assert len(payload["sid"]) >= 32
    assert STEAM_ID not in payload["sid"]
    assert all(payload["sid"] not in key for key in redis.values)
    assert set(redis.deadlines) == set(redis.values), "Every stored session must expire"
    assert client.get("/_test/protected").json() == {
        "steam_id": STEAM_ID, "username": PROFILE_NAME, "avatar": PROFILE_AVATAR,
    }


def test_logout_revokes_the_previously_copied_cookie(session_app):
    app, _store, _redis, _clock = session_app
    client = authenticated_client(app)
    copied_cookie = client.cookies.get("session")
    assert client.get("/_test/protected").status_code == 200

    assert client.post("/auth/steam/logout").status_code == 200

    attacker = replay_client(app, copied_cookie)
    assert attacker.get("/_test/protected").status_code == 401
    assert attacker.get("/auth/steam/status").json() == {"authenticated": False}
    assert client.get("/auth/steam/status").json() == {"authenticated": False}
    assert client.post("/auth/steam/logout").status_code == 200


@pytest.mark.parametrize("session_kind", ["legacy-session", "unknown-session"])
def test_signed_cookie_without_a_live_server_session_is_not_authentication(session_app, session_kind):
    app, _store, _redis, _clock = session_app
    client = TestClient(app)
    assert client.post(f"/_test/{session_kind}").status_code == 200

    assert client.get("/_test/protected").status_code == 401
    assert client.get("/auth/steam/status").json() == {"authenticated": False}


def test_idle_session_expires_even_when_the_signed_cookie_has_not(session_app):
    app, _store, _redis, clock = session_app
    client = authenticated_client(app)

    clock.advance(10)

    assert client.get("/_test/protected").status_code == 401
    assert client.get("/auth/steam/status").json() == {"authenticated": False}


def test_activity_refreshes_idle_ttl_but_never_extends_absolute_expiry(session_app):
    app, _store, redis, clock = session_app
    client = authenticated_client(app)
    absolute_deadline = clock() + 30

    for _ in range(3):
        clock.advance(9)
        assert client.get("/_test/protected").status_code == 200
        assert all(deadline <= absolute_deadline for deadline in redis.deadlines.values())
    clock.advance(3)

    assert client.get("/_test/protected").status_code == 401
    assert client.get("/auth/steam/status").json() == {"authenticated": False}


def test_absolute_expiry_is_enforced_even_if_a_redis_key_outlives_its_ttl(session_app):
    app, _store, redis, clock = session_app
    client = authenticated_client(app)
    # A restored cache or incorrect external TTL must not prolong an old session.
    redis.deadlines.clear()
    clock.advance(31)

    assert client.get("/_test/protected").status_code == 401


def test_new_logins_are_distinct_and_logout_does_not_revoke_another_session(session_app):
    app, _store, _redis, _clock = session_app
    first = authenticated_client(app)
    second = authenticated_client(app)
    assert cookie_payload(first.cookies.get("session"))["sid"] != cookie_payload(
        second.cookies.get("session")
    )["sid"]

    assert first.post("/auth/steam/logout").status_code == 200

    assert first.get("/_test/protected").status_code == 401
    assert second.get("/_test/protected").status_code == 200


def test_successful_openid_callback_rotates_and_revokes_the_previous_sid(session_app):
    app, _store, _redis, _clock = session_app
    client = authenticated_client(app)
    copied_cookie = client.cookies.get("session")
    previous_sid = cookie_payload(copied_cookie)["sid"]
    login = client.get("/auth/steam/login", follow_redirects=False)
    assert login.status_code in {302, 307}
    return_to = parse_qs(urlparse(login.headers["location"]).query)["openid.return_to"][0]
    state = parse_qs(urlparse(return_to).query)["state"][0]

    callback = client.get(f"/auth/steam/callback?state={state}", follow_redirects=False)

    assert callback.status_code == 303
    assert cookie_payload(client.cookies.get("session"))["sid"] != previous_sid
    assert client.get("/_test/protected").status_code == 200
    assert replay_client(app, copied_cookie).get("/_test/protected").status_code == 401


@pytest.mark.parametrize("operation", ["get", "expire"])
@pytest.mark.parametrize("path", ["/_test/protected", "/auth/steam/status"])
def test_session_storage_read_or_refresh_failures_fail_closed(session_app, operation, path):
    app, _store, redis, _clock = session_app
    client = authenticated_client(app)
    redis.failures.add(operation)

    response = client.get(path)

    assert response.status_code == 503
    assert STEAM_ID not in response.text
    assert PROFILE_NAME not in response.text


def test_failed_server_revocation_does_not_claim_successful_logout(session_app):
    app, _store, redis, _clock = session_app
    client = authenticated_client(app)
    copied_cookie = client.cookies.get("session")
    redis.failures.add("delete")

    response = client.post("/auth/steam/logout")

    assert response.status_code == 503
    redis.failures.clear()
    assert replay_client(app, copied_cookie).get("/_test/protected").status_code == 200


def test_session_creation_fails_closed_when_storage_is_unavailable(session_app):
    app, _store, redis, _clock = session_app
    redis.failures.add("set")
    client = TestClient(app)

    response = client.post("/_test/session")

    assert response.status_code == 503
    assert client.cookies.get("session") is None
    assert client.get("/_test/protected").status_code == 401


@pytest.mark.parametrize("corrupt_record", ["not-json", "[]", '{"steam_id":"bad"}'])
def test_corrupt_server_session_records_do_not_grant_access(session_app, corrupt_record):
    app, _store, redis, _clock = session_app
    client = authenticated_client(app)
    key = next(iter(redis.values))
    redis.values[key] = corrupt_record

    assert client.get("/_test/protected").status_code in {401, 503}
    status = client.get("/auth/steam/status")
    assert status.status_code == 503 or status.json() == {"authenticated": False}
