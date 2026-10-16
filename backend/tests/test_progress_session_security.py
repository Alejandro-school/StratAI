"""An already-open progress stream must stop disclosing data after session expiry."""
from contextlib import asynccontextmanager

import pytest
from starlette.requests import Request

from backend.app.auth import sessions
from backend.app.auth.dependencies import SteamUser
from backend.app.routes import sharecodes
from backend.tests.test_session_security import Clock, FakeRedis, STEAM_ID


@pytest.mark.asyncio
@pytest.mark.parametrize("invalidates", ["logout", "idle", "absolute", "redis_failure"])
async def test_progress_stream_stops_before_forwarding_after_session_invalidation(monkeypatch, invalidates):
    clock = Clock()
    redis = FakeRedis(clock)
    store = sessions.SessionStore(redis, idle_seconds=10, absolute_seconds=30, clock=clock)
    monkeypatch.setattr(sessions, "session_store", store)
    sid = await store.create(STEAM_ID)
    request = Request({"type": "http", "headers": [], "session": {"sid": sid}})
    closed = []

    class UpstreamResponse:
        def raise_for_status(self):
            pass

        async def aiter_bytes(self):
            yield b"event: pipeline\ndata: first\n\n"
            if invalidates == "logout":
                await store.revoke(sid)
            elif invalidates in ("idle", "absolute"):
                clock.advance(11 if invalidates == "idle" else 31)
            else:
                redis.failures.add("get")
            yield b"event: pipeline\ndata: SENTINEL_PRIVATE_AFTER_INVALIDATION\n\n"

    class UpstreamClient:
        def __init__(self, **_kwargs):
            pass

        async def __aenter__(self):
            return self

        async def __aexit__(self, *_args):
            closed.append("client")

        @asynccontextmanager
        async def stream(self, *_args, **_kwargs):
            try:
                yield UpstreamResponse()
            finally:
                closed.append("response")

    monkeypatch.setattr(sharecodes.httpx, "AsyncClient", UpstreamClient)
    response = await sharecodes.download_progress(request, SteamUser(steam_id=STEAM_ID))
    chunks = [chunk async for chunk in response.body_iterator]
    assert chunks == [b"event: pipeline\ndata: first\n\n"]
    assert closed == ["response", "client"]


@pytest.mark.asyncio
async def test_passive_session_check_does_not_extend_idle_expiry():
    clock = Clock()
    redis = FakeRedis(clock)
    store = sessions.SessionStore(redis, idle_seconds=10, absolute_seconds=30, clock=clock)
    sid = await store.create(STEAM_ID)
    clock.advance(9)
    assert await store.get(sid, refresh=False)
    clock.advance(2)
    assert await store.get(sid, refresh=False) is None
