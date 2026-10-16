"""Regression coverage for the HTTP request-body trust boundary (SEC-05)."""

import asyncio

import pytest
from starlette.requests import Request
from starlette.responses import JSONResponse

from backend.app.middleware.security import RequestSizeLimitMiddleware


BODY_LIMIT = 1024 * 1024
CHUNK_SIZE = 64 * 1024


async def _request(chunks, *, headers=(), consumer="body"):
    """Exercise the real middleware with bounded, in-memory ASGI transport."""
    pending = list(chunks) or [b""]
    state = {"reads": 0, "handler_bytes": 0, "materialized": False}
    messages = []
    response_finished = asyncio.Event()

    async def receive():
        if state["reads"] >= len(pending):
            # Match a connected HTTP client; never spin on empty messages.
            await response_finished.wait()
            return {"type": "http.disconnect"}
        index = state["reads"]
        state["reads"] += 1
        return {
            "type": "http.request",
            "body": pending[index],
            "more_body": index + 1 < len(pending),
        }

    async def send(message):
        messages.append(message)
        if message["type"] == "http.response.body" and not message.get("more_body"):
            response_finished.set()

    async def endpoint(scope, receive_body, send_response):
        request = Request(scope, receive_body)
        if consumer == "stream":
            async for chunk in request.stream():
                state["handler_bytes"] += len(chunk)
        elif consumer == "body":
            body = await request.body()
            state["handler_bytes"] = len(body)
            state["materialized"] = True
        elif consumer == "swallow_errors":
            # A route's error handling must not defeat the transport limit.
            try:
                await request.body()
            except Exception:
                pass
        await JSONResponse({"accepted": True})(scope, receive_body, send_response)

    scope = {
        "type": "http",
        "asgi": {"version": "3.0", "spec_version": "2.4"},
        "http_version": "1.1",
        "method": "POST",
        "scheme": "http",
        "path": "/",
        "raw_path": b"/",
        "query_string": b"",
        "root_path": "",
        "headers": list(headers),
        "client": ("127.0.0.1", 41000),
        "server": ("testserver", 80),
    }
    await asyncio.wait_for(
        RequestSizeLimitMiddleware(endpoint)(scope, receive, send), timeout=2
    )
    starts = [message for message in messages if message["type"] == "http.response.start"]
    assert len(starts) == 1, "The boundary must emit exactly one HTTP response"
    assert response_finished.is_set(), "The response must terminate"
    return starts[0]["status"], state


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "headers",
    [
        [],
        [(b"transfer-encoding", b"chunked")],
        [(b"content-length", b"1")],
    ],
    ids=["without-content-length", "chunked", "underreported-content-length"],
)
async def test_actual_body_bytes_above_one_mib_are_rejected(headers):
    status, state = await _request(
        [b"x" * CHUNK_SIZE] * (BODY_LIMIT // CHUNK_SIZE) + [b"x"],
        headers=headers,
    )

    assert status == 413
    assert not state["materialized"], "An oversized body must not reach materialization"


@pytest.mark.asyncio
@pytest.mark.parametrize("consumer", ["stream", "ignore", "swallow_errors"])
async def test_body_limit_does_not_depend_on_route_body_handling(consumer):
    status, state = await _request(
        [b"x" * CHUNK_SIZE] * (BODY_LIMIT // CHUNK_SIZE) + [b"x"],
        consumer=consumer,
    )

    assert status == 413
    assert state["handler_bytes"] <= BODY_LIMIT


@pytest.mark.asyncio
async def test_body_limit_stops_reading_at_the_first_oversized_chunk():
    within_limit = [b"x" * CHUNK_SIZE] * (BODY_LIMIT // CHUNK_SIZE)
    status, state = await _request(within_limit + [b"x", b"must not be read"])

    assert status == 413
    assert state["reads"] == len(within_limit) + 1
    assert not state["materialized"]


@pytest.mark.asyncio
@pytest.mark.parametrize("declared_length", [False, True])
@pytest.mark.parametrize("size", [0, BODY_LIMIT - 1, BODY_LIMIT])
async def test_bodies_up_to_the_limit_are_preserved(size, declared_length):
    headers = [(b"content-length", str(size).encode())] if declared_length else []
    body = b"x" * size
    chunks = [body[index : index + CHUNK_SIZE] for index in range(0, size, CHUNK_SIZE)]

    status, state = await _request(chunks, headers=headers)

    assert status == 200
    assert state["materialized"]
    assert state["handler_bytes"] == size


@pytest.mark.asyncio
async def test_announced_oversized_body_is_rejected_before_receiving_bytes():
    status, state = await _request(
        [b"not needed"], headers=[(b"content-length", str(BODY_LIMIT + 1).encode())]
    )

    assert status == 413
    assert state["reads"] == 0
    assert not state["materialized"]


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "headers",
    [
        [(b"content-length", b"not-a-number")],
        [(b"content-length", b"-1")],
        [(b"content-length", b"")],
        [(b"content-length", b"1"), (b"content-length", b"2")],
    ],
    ids=["nonnumeric", "negative", "empty", "conflicting-duplicates"],
)
async def test_invalid_content_length_is_rejected_before_receiving_bytes(headers):
    status, state = await _request([b"x"], headers=headers)

    assert status in {400, 413}
    assert state["reads"] == 0
    assert not state["materialized"]
