"""
Security middleware for the FastAPI backend.
- Security headers (OWASP recommended)
- Request validation
- Path traversal protection utility
"""
import os
from typing import Any, Callable
from urllib.parse import urlparse

from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response
from starlette.types import ASGIApp, Scope, Receive, Send

from ..config import ALLOWED_ORIGINS, IS_PRODUCTION


class SecurityHeadersMiddleware(BaseHTTPMiddleware):
    """Add security headers to every response (OWASP best practices)."""

    async def dispatch(self, request: Request, call_next: Callable[[Request], Any]) -> Response:
        response = await call_next(request)

        # Prevent MIME type sniffing
        response.headers["X-Content-Type-Options"] = "nosniff"

        # Prevent clickjacking
        response.headers["X-Frame-Options"] = "DENY"

        # XSS protection (legacy browsers)
        response.headers["X-XSS-Protection"] = "1; mode=block"

        # Prevent information leakage in referrer
        response.headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
        # Authenticated responses must not be reused across accounts by caches.
        response.headers["Cache-Control"] = "no-store"

        # Restrict browser features
        response.headers["Permissions-Policy"] = (
            "camera=(), microphone=(), geolocation=(), payment=()"
        )

        # Content Security Policy - adjust as needed for your frontend
        response.headers["Content-Security-Policy"] = (
            "default-src 'self'; "
            "script-src 'self'; "
            "style-src 'self'; "
            "img-src 'self' https://avatars.steamstatic.com https://steamcdn-a.akamaihd.net data:; "
            "connect-src 'self'; "
            "frame-ancestors 'none';"
        )

        # HSTS (only meaningful over HTTPS, but safe to include)
        response.headers["Strict-Transport-Security"] = (
            "max-age=31536000; includeSubDomains"
        )

        # Don't expose server technology
        if "server" in response.headers:
            del response.headers["server"]

        return response


class RequestSizeLimitMiddleware:
    """Bound actual body bytes before any handler can parse or ignore them."""

    MAX_BODY_SIZE = 1024 * 1024

    def __init__(self, app: ASGIApp):
        self.app = app

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        async def reject() -> None:
            response = Response('{"detail":"Request body too large or invalid length"}',
                                status_code=413, media_type="application/json")
            await response(scope, receive, send)

        lengths = [value for key, value in scope.get("headers", []) if key.lower() == b"content-length"]
        if lengths and (
            any(not value.isdigit() or len(value) > 20 for value in lengths)
            or len(set(lengths)) != 1
            or int(lengths[0]) > self.MAX_BODY_SIZE
        ):
            await reject()
            return
        # Buffer at most the configured limit; never trust Content-Length for enforcement.
        body = bytearray()
        while True:
            message = await receive()
            if message["type"] == "http.disconnect":
                return
            chunk = message.get("body", b"")
            if len(body) + len(chunk) > self.MAX_BODY_SIZE:
                await reject()
                return
            body.extend(chunk)
            if not message.get("more_body", False):
                break

        consumed = False

        async def bounded_receive():
            nonlocal consumed
            if not consumed:
                consumed = True
                return {"type": "http.request", "body": bytes(body), "more_body": False}
            return await receive()

        await self.app(scope, bounded_receive, send)


class CSRFMiddleware(BaseHTTPMiddleware):
    """Reject cross-origin cookie-authenticated state changes."""

    SAFE_METHODS = {"GET", "HEAD", "OPTIONS"}

    async def dispatch(self, request: Request, call_next: Callable[[Request], Any]) -> Response:
        if request.method in self.SAFE_METHODS:
            return await call_next(request)

        origin = request.headers.get("origin")
        if not origin:
            referer = request.headers.get("referer")
            if referer:
                parsed = urlparse(referer)
                origin = f"{parsed.scheme}://{parsed.netloc}"

        if not origin and not IS_PRODUCTION:
            return await call_next(request)

        if origin not in ALLOWED_ORIGINS:
            return Response(
                content='{"detail":"Origen no permitido"}',
                status_code=403,
                media_type="application/json",
            )
        return await call_next(request)


# =============================================================================
# PATH TRAVERSAL PROTECTION
# =============================================================================
def is_safe_path(base_path: str, target_path: str) -> bool:
    """
    Verify that target_path resolves to a location under base_path.
    Prevents path traversal attacks (e.g., ../../etc/passwd).
    """
    real_base = os.path.realpath(base_path)
    real_target = os.path.realpath(target_path)
    return real_target.startswith(real_base + os.sep) or real_target == real_base


def sanitize_match_id(match_id: str) -> str:
    """
    Reject malformed identifiers rather than aliasing them to another match.
    """
    import re
    if not isinstance(match_id, str) or not re.fullmatch(r"[A-Za-z0-9_-]{1,128}", match_id):
        raise ValueError("Invalid match_id")
    return match_id
