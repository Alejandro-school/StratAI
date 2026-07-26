from datetime import datetime, timezone
from urllib.parse import parse_qs, urlencode, urlparse

import pytest
from fastapi import FastAPI
from fastapi import HTTPException
from fastapi.testclient import TestClient
from starlette.requests import Request
from starlette.middleware.sessions import SessionMiddleware

from backend.app.routes import steam_auth
from backend.app.middleware.security import CSRFMiddleware


class FakeRedis:
    def __init__(self) -> None:
        self.values: set[str] = set()
        self.users: set[str] = set()

    async def set(self, key: str, _value: str, **_options) -> bool:
        if key in self.values:
            return False
        self.values.add(key)
        return True

    async def delete(self, key: str) -> int:
        existed = key in self.values
        self.values.discard(key)
        return int(existed)

    async def sadd(self, _key: str, steam_id: str) -> None:
        self.users.add(steam_id)


class FakeResponse:
    text = "ns:http://specs.openid.net/auth/2.0\nis_valid:true\n"

    def raise_for_status(self) -> None:
        return None


class FakeHTTPClient:
    async def __aenter__(self):
        return self

    async def __aexit__(self, *_args):
        return None

    async def post(self, *_args, **_kwargs) -> FakeResponse:
        return FakeResponse()


def openid_params(state: str = "state-value") -> dict[str, str]:
    steam_id = "76561198000000000"
    identity = f"{steam_auth.STEAM_IDENTITY_PREFIX}{steam_id}"
    nonce_time = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    return {
        "state": state,
        "openid.ns": steam_auth.OPENID_NAMESPACE,
        "openid.mode": "id_res",
        "openid.op_endpoint": steam_auth.STEAM_OPENID_URL,
        "openid.return_to": steam_auth._callback_url(state),
        "openid.claimed_id": identity,
        "openid.identity": identity,
        "openid.response_nonce": f"{nonce_time}unique",
        "openid.assoc_handle": "handle",
        "openid.signed": ",".join(sorted(steam_auth.OPENID_REQUIRED_SIGNED_FIELDS)),
        "openid.sig": "signed-by-provider",
    }


def request_with_query(params: dict[str, str]) -> Request:
    return Request(
        {
            "type": "http",
            "method": "GET",
            "path": "/auth/steam/callback",
            "query_string": urlencode(params).encode(),
            "headers": [],
        }
    )


@pytest.mark.asyncio
async def test_openid_rejects_provider_return_to_and_missing_signed_fields(monkeypatch) -> None:
    monkeypatch.setattr(steam_auth, "redis", FakeRedis())
    monkeypatch.setattr(steam_auth.httpx, "AsyncClient", lambda **_kwargs: FakeHTTPClient())

    invalid_provider = openid_params()
    invalid_provider["openid.op_endpoint"] = "https://attacker.example/openid"
    with pytest.raises(HTTPException, match="Proveedor"):
        await steam_auth._verify_openid_response(
            request_with_query(invalid_provider),
            steam_auth._callback_url("state-value"),
        )

    invalid_return_to = openid_params()
    invalid_return_to["openid.return_to"] = "https://attacker.example/callback"
    with pytest.raises(HTTPException, match="Destino"):
        await steam_auth._verify_openid_response(
            request_with_query(invalid_return_to),
            steam_auth._callback_url("state-value"),
        )

    missing_signed = openid_params()
    missing_signed["openid.signed"] = "op_endpoint,return_to"
    with pytest.raises(HTTPException, match="incompleta"):
        await steam_auth._verify_openid_response(
            request_with_query(missing_signed),
            steam_auth._callback_url("state-value"),
        )


@pytest.mark.asyncio
async def test_openid_accepts_valid_response_once_and_rejects_replay(monkeypatch) -> None:
    fake_redis = FakeRedis()
    monkeypatch.setattr(steam_auth, "redis", fake_redis)
    monkeypatch.setattr(steam_auth.httpx, "AsyncClient", lambda **_kwargs: FakeHTTPClient())
    params = openid_params()
    request = request_with_query(params)

    steam_id = await steam_auth._verify_openid_response(
        request,
        steam_auth._callback_url("state-value"),
    )
    assert steam_id == "76561198000000000"

    with pytest.raises(HTTPException) as replay:
        await steam_auth._verify_openid_response(
            request_with_query(params),
            steam_auth._callback_url("state-value"),
        )
    assert replay.value.status_code == 409


def test_login_redirect_ignores_untrusted_host() -> None:
    callback = steam_auth._callback_url("fixed-state")
    parsed = urlparse(callback)
    assert parsed.netloc == urlparse(steam_auth.PUBLIC_BACKEND_URL).netloc
    assert parse_qs(parsed.query)["state"] == ["fixed-state"]


def test_callback_state_session_registration_and_csrf(monkeypatch) -> None:
    fake_redis = FakeRedis()
    monkeypatch.setattr(steam_auth, "redis", fake_redis)

    async def not_limited(*_args, **_kwargs):
        return False, 10

    async def valid_openid(_request, _return_to):
        return "76561198000000000"

    async def profile(_steam_id):
        return "Test User", "https://avatar.example/test.png"

    monkeypatch.setattr(steam_auth.rate_limiter, "is_rate_limited", not_limited)
    monkeypatch.setattr(steam_auth, "_verify_openid_response", valid_openid)
    monkeypatch.setattr(steam_auth, "_load_profile", profile)

    app = FastAPI()
    app.add_middleware(CSRFMiddleware)
    app.add_middleware(SessionMiddleware, secret_key="test-session-secret")
    app.include_router(steam_auth.router)
    client = TestClient(app)

    login = client.get(
        "/auth/steam/login",
        headers={"host": "attacker.example"},
        follow_redirects=False,
    )
    return_to = parse_qs(urlparse(login.headers["location"]).query)["openid.return_to"][0]
    state = parse_qs(urlparse(return_to).query)["state"][0]
    assert urlparse(return_to).netloc == urlparse(steam_auth.PUBLIC_BACKEND_URL).netloc

    wrong_state = client.get(
        "/auth/steam/callback?state=wrong",
        follow_redirects=False,
    )
    assert wrong_state.status_code == 400

    login = client.get("/auth/steam/login", follow_redirects=False)
    return_to = parse_qs(urlparse(login.headers["location"]).query)["openid.return_to"][0]
    state = parse_qs(urlparse(return_to).query)["state"][0]
    callback = client.get(
        f"/auth/steam/callback?state={state}",
        follow_redirects=False,
    )
    assert callback.status_code == 303
    assert "76561198000000000" in fake_redis.users
    assert client.get("/auth/steam/status").json()["authenticated"] is True

    blocked_logout = client.post(
        "/auth/steam/logout",
        headers={"origin": "https://attacker.example"},
    )
    assert blocked_logout.status_code == 403
