import os
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi_users import FastAPIUsers
from fastapi_users.authentication import CookieTransport, RedisStrategy, AuthenticationBackend
from starlette.middleware.sessions import SessionMiddleware
import redis.asyncio as aioredis
from dotenv import load_dotenv
from app.routes import steam_auth, steam_service

# Cargar variables de entorno
load_dotenv()

app = FastAPI()

# CORS para frontend
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Conexión a Redis
redis = aioredis.from_url("redis://localhost", decode_responses=True)

# Configuración del backend de autenticación
cookie_transport = CookieTransport(cookie_name="session", cookie_max_age=3600)

def get_redis_strategy() -> RedisStrategy:
    return RedisStrategy(redis, lifetime_seconds=3600)

auth_backend = AuthenticationBackend(
    name="redis",
    transport=cookie_transport,
    get_strategy=get_redis_strategy,
)

# Añadir middleware de sesión (opcional)
app.add_middleware(
    SessionMiddleware,
    secret_key=os.getenv("SESSION_SECRET_KEY", "super_secret_key"),
    session_cookie="session",
    max_age=3600,
    same_site="lax",
    https_only=False,
)

# Incluir routers
app.include_router(steam_auth.router)
app.include_router(steam_service.router)

@app.on_event("startup")
async def startup():
    await redis.ping()
    print("✅ Conectado a Redis")

@app.get("/ping")
def ping():
    return {"message": "pong"}
