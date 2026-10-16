import json
import logging
import re
from datetime import datetime, timezone


_SECRET_FIELD = (
    r"(?:key|api[_-]?key|steamidkey|auth[_-]?code|access[_-]?token|"
    r"refresh[_-]?token|session[_-]?secret[_-]?key|internal[_-]?service[_-]?secret|"
    r"credential[_-]?encryption[_-]?key|openid\.sig|x-service-signature|password)"
)
_QUOTED_SECRET = re.compile(rf"(\b{_SECRET_FIELD}\b[\"']?\s*[:=]\s*)([\"'])(.*?)\2", re.I)
_BARE_SECRET = re.compile(rf"(\b{_SECRET_FIELD}\b[\"']?\s*[:=]\s*)[^\s&,;\"'}}]+", re.I)
_CREDENTIAL_HEADER = re.compile(r"(\b(?:authorization|cookie|set-cookie)\b[\"']?\s*[:=]\s*)[^\r\n]+", re.I)


def redact_sensitive_text(message: str) -> str:
    """Keep secrets out of all formatted logs, including third-party error URLs."""
    message = _CREDENTIAL_HEADER.sub(r"\1[REDACTED]", message)
    message = _QUOTED_SECRET.sub(r"\1\2[REDACTED]\2", message)
    return _BARE_SECRET.sub(r"\1[REDACTED]", message)


class JSONFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "level": record.levelname.lower(),
            "service": "fastapi",
            "logger": record.name,
            "message": redact_sensitive_text(record.getMessage()),
        }
        if record.exc_info:
            payload["exception"] = record.exc_info[0].__name__
        return json.dumps(payload, ensure_ascii=False)


def configure_logging() -> None:
    handler = logging.StreamHandler()
    handler.setFormatter(JSONFormatter())
    logging.basicConfig(level=logging.INFO, handlers=[handler], force=True)
    # Uvicorn configures its own non-propagating access handler before importing
    # the app. Route it through the same redactor, including OpenID query strings.
    for name in ("uvicorn", "uvicorn.error", "uvicorn.access"):
        logger = logging.getLogger(name)
        logger.handlers.clear()
        logger.propagate = True
    # HTTP clients include complete query strings in informational request logs.
    logging.getLogger("httpx").setLevel(logging.WARNING)
    logging.getLogger("httpcore").setLevel(logging.WARNING)
