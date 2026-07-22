"""
Centralized security configuration for the FastAPI backend.
All environment variables and security settings in one place.
"""
import os
from pathlib import Path
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


SESSION_SECRET_KEY = _require_env("SESSION_SECRET_KEY")
STEAM_API_KEY = _require_env("STEAM_API_KEY")

# =============================================================================
# ENVIRONMENT DETECTION
# =============================================================================
ENV = os.getenv("APP_ENV", "development").lower()
IS_PRODUCTION = ENV == "production"

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
