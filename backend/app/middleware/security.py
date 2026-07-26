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
from starlette.types import ASGIApp

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


class RequestSizeLimitMiddleware(BaseHTTPMiddleware):
    """Limit request body size to prevent DoS via large payloads."""

    MAX_BODY_SIZE = 1024 * 1024

    async def dispatch(self, request: Request, call_next: Callable[[Request], Any]) -> Response:
        content_length = request.headers.get("content-length")
        try:
            parsed_content_length = int(content_length) if content_length else 0
        except ValueError:
            parsed_content_length = self.MAX_BODY_SIZE + 1
        if parsed_content_length > self.MAX_BODY_SIZE:
            return Response(
                content='{"detail": "Request body too large"}',
                status_code=413,
                media_type="application/json",
            )
        return await call_next(request)


class SessionRefreshMiddleware(BaseHTTPMiddleware):
    """Sliding window session: refreshes cookie expiry on each authenticated request."""

    def __init__(self, app: ASGIApp, max_age: int = 604800):
        super().__init__(app)
        self.max_age = max_age

    async def dispatch(self, request: Request, call_next: Callable[[Request], Any]) -> Response:
        response = await call_next(request)
        # If the session has a steam_id, the user is active — refresh the cookie
        if request.session.get("steam_id"):
            # Touching any session key forces SessionMiddleware to re-set the cookie
            request.session["_refreshed"] = True
        return response


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
    Sanitize match_id to prevent path traversal.
    Only allows alphanumeric characters, hyphens, and underscores.
    """
    import re
    sanitized = re.sub(r'[^a-zA-Z0-9_\-]', '', match_id)
    if not sanitized:
        raise ValueError("Invalid match_id")
    return sanitized
