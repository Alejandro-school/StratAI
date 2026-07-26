import logging
from pathlib import Path
from dotenv import load_dotenv

# Cargar variables de entorno
env_path = Path(__file__).resolve().parents[1] / ".env"
load_dotenv(dotenv_path=env_path)

from fastapi import FastAPI
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from starlette.middleware.sessions import SessionMiddleware
from starlette.middleware.trustedhost import TrustedHostMiddleware
import redis.asyncio as aioredis

# Security config & middleware
from .config import (
    SESSION_SECRET_KEY,
    ALLOWED_ORIGINS, REDIS_URL,
    SESSION_COOKIE_NAME, SESSION_MAX_AGE, SESSION_SAME_SITE,
    SESSION_HTTPS_ONLY, CORS_ALLOW_METHODS, CORS_ALLOW_HEADERS,
    IS_PRODUCTION, TRUSTED_HOSTS,
)
from .middleware.security import (
    CSRFMiddleware,
    RequestSizeLimitMiddleware,
    SecurityHeadersMiddleware,
    SessionRefreshMiddleware,
)
from .logging_config import configure_logging

configure_logging()

# Rutas
from .routes import steam_auth, steam_service, sharecodes, dashboard, performance, feedback, replay_annotations

app = FastAPI(
    docs_url=None if IS_PRODUCTION else "/docs",
    redoc_url=None if IS_PRODUCTION else "/redoc",
)

# ── Security Middleware (applied first = runs last, wraps everything) ──
app.add_middleware(SecurityHeadersMiddleware)
app.add_middleware(RequestSizeLimitMiddleware)
app.add_middleware(SessionRefreshMiddleware, max_age=SESSION_MAX_AGE)
app.add_middleware(CSRFMiddleware)
app.add_middleware(TrustedHostMiddleware, allowed_hosts=TRUSTED_HOSTS)

# ── CORS ── Whitelist-based (no wildcard regex)
app.add_middleware(
    CORSMiddleware,
    allow_origins=ALLOWED_ORIGINS,
    allow_credentials=True,
    allow_methods=CORS_ALLOW_METHODS,
    allow_headers=CORS_ALLOW_HEADERS,
)

# ── Conexión a Redis ──
redis = aioredis.from_url(REDIS_URL, decode_responses=True)

# Configuración del backend de autenticación (FastAPI Users)
# ── Session Middleware (hardened) ──
app.add_middleware(
    SessionMiddleware,
    secret_key=SESSION_SECRET_KEY,
    session_cookie=SESSION_COOKIE_NAME,
    max_age=SESSION_MAX_AGE,
    same_site=SESSION_SAME_SITE,
    https_only=SESSION_HTTPS_ONLY,
)

# Incluir Routers
app.include_router(steam_auth.router)      # <--- Asegúrate de que exista
app.include_router(steam_service.router)
app.include_router(sharecodes.router)
app.include_router(dashboard.router)
app.include_router(performance.router)
app.include_router(feedback.router)
app.include_router(replay_annotations.router)


@app.on_event("startup")
async def startup():
    await redis.ping()
    logging.info("Conectado a Redis")

@app.get("/ping")
def ping():
    return {"message": "pong"}


@app.get("/ready")
async def ready():
    try:
        await redis.ping()
    except Exception:
        return JSONResponse({"status": "not_ready"}, status_code=503)
    return {"status": "ready"}


@app.on_event("shutdown")
async def shutdown() -> None:
    await redis.aclose()

