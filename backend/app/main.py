import os
from pathlib import Path
from dotenv import load_dotenv

# Cargar variables de entorno
env_path = Path(__file__).resolve().parents[1] / ".env"
load_dotenv(dotenv_path=env_path)

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi_users import FastAPIUsers
from fastapi_users.authentication import CookieTransport, AuthenticationBackend
from fastapi_users.authentication.strategy.redis import RedisStrategy
from starlette.middleware.sessions import SessionMiddleware
import redis.asyncio as aioredis

# Rutas
from .routes import steam_auth, steam_service, auth_status, sharecodes, dashboard, performance

STEAM_API_KEY = os.getenv("STEAM_API_KEY", "")
print(f"STEAM_API_KEY cargada (longitud={len(STEAM_API_KEY)})")

app = FastAPI()

# CORS flexible para túneles dinámicos (trycloudflare.com)
app.add_middleware(
    CORSMiddleware,
    # Permitimos cualquier origen que termine en .trycloudflare.com, localhost o IPs locales
    allow_origin_regex=r"https?://.*\.trycloudflare\.com|https?://localhost(:[0-9]+)?|https?://127\.0\.0\.1(:[0-9]+)?",
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)
# Nota: Al unificar todo bajo el proxy (puerto 9000), el navegador lo ve como mismo origen,
# pero mantenemos esto por si se accede a servicios individuales.

# Conexión a Redis
redis = aioredis.from_url("redis://localhost", decode_responses=True)

# Configuración del backend de autenticación (FastAPI Users)
cookie_transport = CookieTransport(cookie_name="session", cookie_max_age=3600)

def get_redis_strategy() -> RedisStrategy:
    return RedisStrategy(redis, lifetime_seconds=3600)

auth_backend = AuthenticationBackend(
    name="redis",
    transport=cookie_transport,
    get_strategy=get_redis_strategy,
)

# Añadir middleware de sesión
app.add_middleware(
    SessionMiddleware,
    secret_key=os.getenv("SESSION_SECRET_KEY", "super_secret_key"),
    session_cookie="session",
    max_age=30 * 24 * 60 * 60,
    same_site="lax",
    https_only=False,
)

# Incluir Routers
app.include_router(steam_auth.router)      # <--- Asegúrate de que exista
app.include_router(steam_service.router)
app.include_router(auth_status.router)
app.include_router(sharecodes.router)
app.include_router(dashboard.router)
app.include_router(performance.router)


@app.on_event("startup")
async def startup():
    await redis.ping()
    print("Conectado a Redis")

@app.get("/ping")
def ping():
    return {"message": "pong"}

@app.on_event("startup")
async def startup_event():
    for route in app.routes:
        print(f"Endpoint registrado: {route.path}")

