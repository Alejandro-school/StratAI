"""
Centralized security configuration for the FastAPI backend.
All environment variables and security settings in one place.
"""
import os
import hashlib
from pathlib import Path
from urllib.parse import urlparse
from dotenv import load_dotenv

# Load .env from backend root
env_path = Path(__file__).resolve().parents[1] / ".env"
load_dotenv(dotenv_path=env_path)


# =============================================================================
# CRITICAL SECRETS (fail-fast if missing)
# =============================================================================
def _require_env(key: str) -> str:
    """Require an environment variable to be set. Fail fast on startup."""
    value = os.getenv(key)
    if not value:
        raise RuntimeError(
            f"CRITICAL: Required environment variable '{key}' is not set. "
            f"Set it in {env_path} or as a system environment variable."
        )
    return value


ENV = os.getenv("APP_ENV", "development").lower()
IS_PRODUCTION = ENV == "production"

SESSION_SECRET_KEY = _require_env("SESSION_SECRET_KEY")
STEAM_API_KEY = _require_env("STEAM_API_KEY")
if IS_PRODUCTION and len(SESSION_SECRET_KEY) < 32:
    raise RuntimeError("CRITICAL: 'SESSION_SECRET_KEY' must contain at least 32 characters.")


def _production_secret(key: str, development_value: str) -> str:
    value = os.getenv(key)
    if value:
        if len(value) < 32:
            raise RuntimeError(f"CRITICAL: '{key}' must contain at least 32 characters.")
        return value
    if IS_PRODUCTION:
        raise RuntimeError(f"CRITICAL: Required production secret '{key}' is not set.")
    return development_value


INTERNAL_SERVICE_SECRET = _production_secret(
    "INTERNAL_SERVICE_SECRET",
    hashlib.sha256(f"{SESSION_SECRET_KEY}:internal-service".encode()).hexdigest(),
)
CREDENTIAL_ENCRYPTION_KEY = _production_secret(
    "CREDENTIAL_ENCRYPTION_KEY",
    hashlib.sha256(f"{SESSION_SECRET_KEY}:credential-encryption".encode()).hexdigest(),
)
if IS_PRODUCTION and len(
    {SESSION_SECRET_KEY, INTERNAL_SERVICE_SECRET, CREDENTIAL_ENCRYPTION_KEY}
) != 3:
    raise RuntimeError("CRITICAL: Session, service and credential secrets must be independent.")

# =============================================================================
# CORS CONFIGURATION
# =============================================================================
def _build_allowed_origins() -> list[str]:
    raw_origins = os.getenv("ALLOWED_ORIGINS", "")
    if raw_origins:
        origins = [origin.strip() for origin in raw_origins.split(",") if origin.strip()]
    else:
        origins = [
            "http://localhost:3000",
            "http://localhost:8000",
            "http://127.0.0.1:3000",
            "http://127.0.0.1:8000",
        ]

    tunnel_origin = os.getenv("TUNNEL_ORIGIN")
    if tunnel_origin:
        origins.append(tunnel_origin.strip())

    return origins


ALLOWED_ORIGINS = _build_allowed_origins()


def _base_url(key: str, development_default: str) -> str:
    value = os.getenv(key, development_default).rstrip("/")
    parsed = urlparse(value)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise RuntimeError(f"CRITICAL: '{key}' must be an absolute HTTP(S) URL.")
    if IS_PRODUCTION and parsed.scheme != "https":
        raise RuntimeError(f"CRITICAL: '{key}' must use HTTPS in production.")
    return value


PUBLIC_BACKEND_URL = _base_url("PUBLIC_BACKEND_URL", "http://localhost:8000")
FRONTEND_URL = _base_url("FRONTEND_URL", "http://localhost:3000")
if FRONTEND_URL not in ALLOWED_ORIGINS:
    ALLOWED_ORIGINS.append(FRONTEND_URL)
TRUSTED_HOSTS = [
    host.strip()
    for host in os.getenv(
        "TRUSTED_HOSTS",
        urlparse(PUBLIC_BACKEND_URL).hostname or "localhost",
    ).split(",")
    if host.strip()
]

# =============================================================================
# SESSION / COOKIE CONFIGURATION
# =============================================================================
SESSION_COOKIE_NAME = "session"
SESSION_MAX_AGE = int(os.getenv("SESSION_MAX_AGE", "604800"))  # 7 days default
SESSION_SAME_SITE = "lax"
SESSION_HTTPS_ONLY = IS_PRODUCTION  # True in production, False in dev
SESSION_COOKIE_HTTPONLY = True

# =============================================================================
# REDIS
# =============================================================================
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
NODE_SERVICE_URL = os.getenv("NODE_SERVICE_URL", "http://localhost:4000")
PIPELINE_NAMESPACE = os.getenv("PIPELINE_NAMESPACE", "stratai:v2")
PIPELINE_V2_ENABLED = os.getenv("PIPELINE_V2_ENABLED", "true").lower() == "true"

# =============================================================================
# RATE LIMITING
# =============================================================================
RATE_LIMIT_LOGIN = "10/minute"
RATE_LIMIT_API = "60/minute"
RATE_LIMIT_FEEDBACK = "5/hour"

# =============================================================================
# FILE PATHS
# =============================================================================
DATA_DIR = Path(__file__).resolve().parents[1] / "data"
EXPORTS_DIR = DATA_DIR / "exports"
DEMOS_DIR = DATA_DIR / "demos"

# =============================================================================
# ALLOWED HTTP METHODS (restrictive)
# =============================================================================
CORS_ALLOW_METHODS = ["GET", "POST", "OPTIONS"]
CORS_ALLOW_HEADERS = ["Content-Type", "Authorization"]
